// Package proposal is the branch and pull request side of the rules PR (spec
// 11.2). Start chooses the branch the suggestions go on and lists the ids that
// must not be suggested again; Publish commits what changed in the rule
// folders and opens the PR or updates the one that is open.
//
// These two functions are the only code that pushes a branch or touches a pull
// request, so they refuse anything but a rules-proposal/* branch and anything
// but the files of guides/, glossary/ and rules/. They run the git and gh
// command lines in the repo, with explicit arguments and no shell, and never
// force a push.
package proposal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/suggest"
)

const (
	// Label marks the pull requests of the rules improvement loop.
	Label = "rules-proposal"
	// BranchPrefix starts the name of every branch a rules PR is made on; the
	// rest is the date of the run that opened it.
	BranchPrefix = Label + "/"
	// DefaultBase is the branch the PR is opened against.
	DefaultBase = "main"
	// LabelColor and LabelDescription are given to the label every time it is
	// created with --force, so that gh does not pick a new colour each time.
	LabelColor       = "0E8A16"
	LabelDescription = "規則建議，由 CI 提出"
	// ExcludeFile is where Start writes the ids that lct suggest must leave out,
	// relative to the repo root. The work folder is not under version control.
	ExcludeFile = "work/proposal/exclude.txt"

	dateLayout = "2006-01-02"

	// The marker of a suggestion in the description of a PR.
	markerStart = "<!-- suggestion:"
	markerEnd   = " -->"
)

// RuleFolders are the only folders a rules PR may change, relative to the
// repo root.
var RuleFolders = []string{"guides", "glossary", "rules"}

// Options says which repo to work in and when.
type Options struct {
	// Repo is the root of the repo checkout; the commands run there. Empty
	// means the current directory.
	Repo string
	// Date is YYYY-MM-DD: the day of the run. Start names the branch after it
	// and requires it; Publish uses it for the commit and the title of a new PR,
	// and takes the day from the branch when it is empty.
	Date string
	// Base is the branch the PR is opened against. Empty means main.
	Base string
	// Git and GH are the executables to run. Empty means "git" and "gh".
	Git, GH string
}

// Started is what Start decided.
type Started struct {
	// Branch is checked out, and is where the changes go.
	Branch string `json:"branch"`
	// PR and URL are those of the open rules PR the branch belongs to; they
	// are 0 and empty when no rules PR is open.
	PR  int    `json:"pr"`
	URL string `json:"url"`
	// Existing are the ids the open PR has already.
	Existing []string `json:"existing"`
	// Declined are the ids of the PRs that were closed without a merge: people
	// did not want them.
	Declined []string `json:"declined"`
	// Handled are the ids of the merged PRs that still have to be left out: the
	// inconsistent translations. A merged rule does not change what was
	// translated before, so the same inconsistency would be suggested again.
	// The other kinds are answered by the rules they changed.
	Handled []string `json:"handled"`
}

func (s *Started) String() string {
	var b strings.Builder
	if s.PR != 0 {
		fmt.Fprintf(&b, "%s: the open rules PR #%d, %s\n", s.Branch, s.PR, s.URL)
	} else {
		fmt.Fprintf(&b, "%s: no rules PR is open\n", s.Branch)
	}
	fmt.Fprintf(&b, "%d suggestion(s) in the PR already, %d declined, %d handled; all are left out of %s\n",
		len(s.Existing), len(s.Declined), len(s.Handled), ExcludeFile)
	return b.String()
}

// Published is what Publish did.
type Published struct {
	Branch string `json:"branch"`
	// URL is that of the PR, new or updated.
	URL string `json:"url"`
	// Commit is the HEAD that was pushed.
	Commit string `json:"commit"`
	// Created is true when the PR is new, false when the open PR was updated.
	Created bool `json:"created"`
}

func (p *Published) String() string {
	verb := "updated"
	if p.Created {
		verb = "opened"
	}
	return fmt.Sprintf("%s: %s the rules PR %s at %s\n", p.Branch, verb, p.URL, p.Commit)
}

// Refusal is an error that the person who runs the command fixes: a branch,
// a file, a missing token or identity, a bad argument. The other errors are
// failures of git or gh.
type Refusal struct{ Msg string }

func (e *Refusal) Error() string { return e.Msg }

func refuse(format string, args ...any) error { return &Refusal{Msg: fmt.Sprintf(format, args...)} }

// CommandError is a git or gh command that failed.
type CommandError struct {
	Name   string   // "git" or "gh"
	Args   []string // what it was given
	Stderr string   // what it printed on stderr, trimmed
	Err    error    // usually an *exec.ExitError
}

func (e *CommandError) Error() string {
	detail := e.Stderr
	if detail == "" {
		detail = e.Err.Error()
	}
	return fmt.Sprintf("%s %s: %s", e.Name, strings.Join(e.Args, " "), detail)
}

func (e *CommandError) Unwrap() error { return e.Err }

// runner runs one executable in the repo.
type runner struct {
	name string // as the errors call it
	bin  string
	dir  string
	env  []string
}

func (r runner) run(args ...string) (string, error) {
	cmd := exec.Command(r.bin, args...)
	cmd.Dir = r.dir
	cmd.Env = append(os.Environ(), r.env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return "", &CommandError{Name: r.name, Args: args, Stderr: strings.TrimSpace(stderr.String()), Err: err}
	}
	return stdout.String(), nil
}

// exitCode returns the exit status of a command that ran and failed, or -1
// for any other error, such as an executable that was not found.
func exitCode(err error) int {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return -1
}

// session is what Start and Publish work with once their options are checked.
type session struct {
	Options
	git, gh runner
}

func newSession(o Options) *session {
	if o.Repo == "" {
		o.Repo = "."
	}
	if o.Base == "" {
		o.Base = DefaultBase
	}
	if o.Git == "" {
		o.Git = "git"
	}
	if o.GH == "" {
		o.GH = "gh"
	}
	return &session{
		Options: o,
		// A missing credential must fail, not wait for someone to type it.
		git: runner{name: "git", bin: o.Git, dir: o.Repo, env: []string{"GIT_TERMINAL_PROMPT=0"}},
		gh:  runner{name: "gh", bin: o.GH, dir: o.Repo, env: []string{"GH_PROMPT_DISABLED=1"}},
	}
}

// checkBase refuses a base that git could take for an option.
func (s *session) checkBase() error {
	if strings.HasPrefix(s.Base, "-") {
		return refuse("the base branch %q looks like an option", s.Base)
	}
	return nil
}

func checkDate(date string) error {
	if _, err := time.Parse(dateLayout, date); err != nil {
		return refuse("the date %q is not YYYY-MM-DD", date)
	}
	return nil
}

// needToken refuses to start when gh has no token to work with. A workflow
// passes ${{ github.token }} as GH_TOKEN.
func needToken() error {
	if os.Getenv("GH_TOKEN") != "" || os.Getenv("GITHUB_TOKEN") != "" {
		return nil
	}
	return refuse("gh needs a token: set GH_TOKEN or GITHUB_TOKEN in the environment (in a workflow: env: GH_TOKEN: ${{ github.token }})")
}

// pullRequest is the part of a gh pr list entry that is used.
type pullRequest struct {
	Number      int    `json:"number"`
	URL         string `json:"url"`
	HeadRefName string `json:"headRefName"`
	Body        string `json:"body"`
	// MergedAt is null for a PR that was closed without a merge; older gh
	// versions print the zero time instead.
	MergedAt *string `json:"mergedAt"`
}

func (p pullRequest) merged() bool {
	return p.MergedAt != nil && *p.MergedAt != "" && !strings.HasPrefix(*p.MergedAt, "0001-")
}

// listPRs runs "gh pr list" with args and reads the JSON it prints.
func (s *session) listPRs(what string, args ...string) ([]pullRequest, error) {
	out, err := s.gh.run(append([]string{"pr", "list"}, args...)...)
	if err != nil {
		return nil, err
	}
	var list []pullRequest
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		return nil, fmt.Errorf("gh pr list (%s): the output is not JSON: %w", what, err)
	}
	return list, nil
}

// Start chooses the branch the suggestions of this run go on and checks it
// out. With an open rules PR it is that PR's branch, brought up to date from
// origin, so the new suggestions are added to the PR. Without one it is
// rules-proposal/<Date>, made from origin/<Base>, except that origin may have
// that name already:
//   - no PR was ever opened from it (an earlier run pushed it and failed before
//     the PR): it is checked out as origin has it, and the run goes on;
//   - a PR was (it was closed or merged): the first rules-proposal/<Date>-N,
//     N = 2, 3, ..., that origin does not have, made from origin/<Base>.
//
// It also writes ExcludeFile:
// the ids that lct suggest must leave out, because the open PR has them, people
// closed a PR with them, or they were merged and still stand (Started.Handled).
// Nothing is changed when it fails before it checks the branch out.
func Start(o Options) (*Started, error) {
	s := newSession(o)
	if err := checkDate(s.Date); err != nil {
		return nil, err
	}
	if err := s.checkBase(); err != nil {
		return nil, err
	}
	if err := needToken(); err != nil {
		return nil, err
	}

	open, err := s.listPRs("open", "--label", Label, "--state", "open", "--json", "number,url,headRefName,body")
	if err != nil {
		return nil, err
	}
	closed, err := s.listPRs("closed", "--label", Label, "--state", "closed", "--json", "number,body,mergedAt", "--limit", "200")
	if err != nil {
		return nil, err
	}

	// There should be one open PR at most. If there are more, the oldest
	// gets the new suggestions, and the ids of all of them are left out.
	slices.SortFunc(open, func(a, b pullRequest) int { return a.Number - b.Number })
	res := &Started{}
	var existing, declined, handled []string
	for _, p := range open {
		if !strings.HasPrefix(p.HeadRefName, BranchPrefix) {
			return nil, refuse("the open PR #%d has the label %s but its branch is %q, not %s*; take the label off or close it",
				p.Number, Label, p.HeadRefName, BranchPrefix)
		}
		existing = append(existing, suggestionIDs(p.Body)...)
	}
	for _, p := range closed {
		for _, id := range suggestionIDs(p.Body) {
			switch {
			case !p.merged():
				declined = append(declined, id)
			case strings.HasPrefix(id, suggest.KindInconsistent+"/"):
				handled = append(handled, id)
			}
		}
	}
	res.Existing, res.Declined, res.Handled = uniqueSorted(existing), uniqueSorted(declined), uniqueSorted(handled)

	if len(open) > 0 {
		res.Branch, res.PR, res.URL = open[0].HeadRefName, open[0].Number, open[0].URL
		err = s.checkoutFromOrigin(res.Branch)
	} else {
		res.Branch, err = s.checkoutNewBranch()
	}
	if err != nil {
		return nil, err
	}

	if err := writeExclude(s.Repo, slices.Concat(res.Existing, res.Declined, res.Handled)); err != nil {
		return nil, err
	}
	return res, nil
}

// checkoutFromOrigin fetches the branch and checks it out at what origin has.
func (s *session) checkoutFromOrigin(branch string) error {
	if _, err := s.git.run("fetch", "origin", branch); err != nil {
		return err
	}
	if err := s.refuseLocalCommits(branch, "FETCH_HEAD", "origin"); err != nil {
		return err
	}
	_, err := s.git.run("checkout", "-q", "-B", branch, "FETCH_HEAD")
	return err
}

// maxBranchSuffix is where the search for a free rules-proposal/<Date>-N gives
// up. A day with that many closed rules PRs is not a day to go on.
const maxBranchSuffix = 50

// checkoutNewBranch checks out the branch for a run without an open rules PR
// and returns its name; see Start for which one it is. A name that origin has
// is dealt with here, and not left to the push: that would be rejected after
// the suggestions were reviewed.
func (s *session) checkoutNewBranch() (string, error) {
	branch := BranchPrefix + s.Date
	onOrigin, err := s.originHas(branch)
	if err != nil {
		return "", err
	}
	if !onOrigin {
		return branch, s.checkoutFromBase(branch)
	}
	used, err := s.headOfAnyPR(branch)
	if err != nil {
		return "", err
	}
	if !used {
		return branch, s.checkoutFromOrigin(branch)
	}
	for n := 2; n <= maxBranchSuffix; n++ {
		next := fmt.Sprintf("%s-%d", branch, n)
		if onOrigin, err := s.originHas(next); err != nil {
			return "", err
		} else if !onOrigin {
			return next, s.checkoutFromBase(next)
		}
	}
	return "", refuse("origin has %s and the branches %s-2 to %s-%d, and a PR was opened from %s; delete some of them", branch, branch, branch, maxBranchSuffix, branch)
}

// checkoutFromBase makes the branch from origin/<Base>.
func (s *session) checkoutFromBase(branch string) error {
	base := "origin/" + s.Base
	if err := s.refuseLocalCommits(branch, base, base); err != nil {
		return err
	}
	_, err := s.git.run("checkout", "-q", "-B", branch, base)
	return err
}

// originHas asks origin whether it has the branch.
func (s *session) originHas(branch string) (bool, error) {
	out, err := s.git.run("ls-remote", "--heads", "origin", "refs/heads/"+branch)
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasSuffix(line, "\trefs/heads/"+branch) {
			return true, nil
		}
	}
	return false, nil
}

// headOfAnyPR says whether a PR, in any state and with any label, was opened
// from the branch.
func (s *session) headOfAnyPR(branch string) (bool, error) {
	prs, err := s.listPRs("any state", "--state", "all", "--head", branch, "--json", "number,headRefName")
	if err != nil {
		return false, err
	}
	return slices.ContainsFunc(prs, func(p pullRequest) bool { return p.HeadRefName == branch }), nil
}

// refuseLocalCommits refuses to reset the local branch, if it exists, to
// target when that would drop commits that target does not have.
func (s *session) refuseLocalCommits(branch, target, targetName string) error {
	if _, err := s.git.run("rev-parse", "--verify", "--quiet", "refs/heads/"+branch); err != nil {
		if exitCode(err) == 1 {
			return nil // there is no such local branch
		}
		return err
	}
	out, err := s.git.run("rev-list", "-n", "1", target+"..refs/heads/"+branch)
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != "" {
		return refuse("the local branch %s has commits that are not on %s; push them or delete the branch first", branch, targetName)
	}
	return nil
}

// writeExclude writes the ids to ExcludeFile, one a line, as they are. An id
// is never changed: lct suggest compares them as they are.
func writeExclude(repo string, ids []string) error {
	path := filepath.Join(repo, filepath.FromSlash(ExcludeFile))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	for _, id := range uniqueSorted(ids) {
		b.WriteString(id + "\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// suggestionIDs returns the ids of the markers <!-- suggestion:<id> --> of a
// PR description, in order and as written. An id is everything between
// "suggestion:" and the last " -->" of its marker: it may hold spaces, CJK
// text and "|". A marker ends on its line or before the next marker of that
// line; one that has no " -->" or an id of blanks is not a marker.
func suggestionIDs(body string) []string {
	var ids []string
	for _, line := range strings.Split(body, "\n") {
		for {
			i := strings.Index(line, markerStart)
			if i < 0 {
				break
			}
			marker := line[i+len(markerStart):]
			next := strings.Index(marker, markerStart)
			if next >= 0 {
				marker, line = marker[:next], marker[next:]
			} else {
				line = ""
			}
			if end := strings.LastIndex(marker, markerEnd); end >= 0 && strings.TrimSpace(marker[:end]) != "" {
				ids = append(ids, marker[:end])
			}
		}
	}
	return ids
}

func uniqueSorted(ids []string) []string {
	out := slices.Clone(ids)
	if out == nil {
		out = []string{}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// Publish commits the changes of the rule folders to the current branch,
// pushes it and opens the rules PR, or updates the description of the one that
// is open for the branch. bodyFile is the description, and it must hold at
// least one suggestion marker.
//
// It refuses, before it changes anything: a branch that is not
// rules-proposal/*, a tracked file changed (staged or not) outside the rule
// folders, no token for gh, no git identity, a body without markers, and a
// branch with nothing in it that origin/<Base> does not have. Untracked files
// outside the rule folders are left alone and not committed. The push is never
// forced.
func Publish(o Options, bodyFile string) (*Published, error) {
	s := newSession(o)
	if err := s.checkBase(); err != nil {
		return nil, err
	}
	if s.Date != "" {
		if err := checkDate(s.Date); err != nil {
			return nil, err
		}
	}
	if bodyFile == "" {
		return nil, refuse("no body file: the description of the PR is needed, and it holds the <!-- suggestion:<id> --> markers")
	}
	bodyFile, err := filepath.Abs(bodyFile)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(bodyFile)
	if err != nil {
		return nil, refuse("cannot read the body file %s: %v", bodyFile, unwrapPathError(err))
	}
	if len(suggestionIDs(string(data))) == 0 {
		return nil, refuse("the body file %s has no %s<id>%s marker; every suggestion in the PR needs one", bodyFile, markerStart, markerEnd)
	}
	if err := needToken(); err != nil {
		return nil, err
	}

	branch, err := s.currentBranch()
	if err != nil {
		return nil, err
	}
	if s.Date == "" {
		if s.Date = branchDate(branch); s.Date == "" {
			return nil, refuse("the branch %s has no date in its name; pass --date YYYY-MM-DD", branch)
		}
	}

	changed, err := s.ruleChanges()
	if err != nil {
		return nil, err
	}
	if len(changed) > 0 {
		if err := s.checkIdentity(); err != nil {
			return nil, err
		}
		// The names are paths, not patterns: a "[" or a "*" in one is a character.
		if _, err := s.git.run(append([]string{"--literal-pathspecs", "add", "-A", "--"}, changed...)...); err != nil {
			return nil, err
		}
		if _, err := s.git.run("-c", "user.useConfigOnly=true", "commit", "-q", "-m", "rules: proposal "+s.Date); err != nil {
			return nil, err
		}
	}

	out, err := s.git.run("rev-list", "--count", "origin/"+s.Base+"..HEAD")
	if err != nil {
		return nil, err
	}
	if n, err := strconv.Atoi(strings.TrimSpace(out)); err != nil {
		return nil, fmt.Errorf("git rev-list --count: %q is not a number", strings.TrimSpace(out))
	} else if n == 0 {
		return nil, refuse("nothing to propose: %s has no change in %s compared with origin/%s",
			branch, ruleFolderList(), s.Base)
	}
	if _, err := s.git.run("push", "origin", branch); err != nil {
		return nil, err
	}
	head, err := s.git.run("rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	res := &Published{Branch: branch, Commit: strings.TrimSpace(head)}

	// A PR of this branch that is open is updated, whatever its label.
	open, err := s.listPRs("branch", "--state", "open", "--head", branch, "--json", "number,url,headRefName")
	if err != nil {
		return nil, err
	}
	for _, p := range open {
		if p.HeadRefName != branch {
			continue
		}
		if _, err := s.gh.run("pr", "edit", strconv.Itoa(p.Number), "--body-file", bodyFile); err != nil {
			return nil, err
		}
		res.URL = p.URL
		return res, nil
	}

	if _, err := s.gh.run("label", "create", Label, "--force", "--color", LabelColor, "--description", LabelDescription); err != nil {
		return nil, err
	}
	out, err = s.gh.run("pr", "create", "--base", s.Base, "--head", branch,
		"--title", "規則建議："+s.Date, "--label", Label, "--body-file", bodyFile)
	if err != nil {
		return nil, err
	}
	lines := strings.Fields(out)
	if len(lines) == 0 || !strings.HasPrefix(lines[len(lines)-1], "http") {
		return nil, fmt.Errorf("gh pr create printed no URL: %q", strings.TrimSpace(out))
	}
	res.URL, res.Created = lines[len(lines)-1], true
	return res, nil
}

// branchDate returns the date in the name of a rules-proposal/<Date> or
// rules-proposal/<Date>-N branch, or "" if there is none.
func branchDate(branch string) string {
	tail := strings.TrimPrefix(branch, BranchPrefix)
	if len(tail) < len(dateLayout) || checkDate(tail[:len(dateLayout)]) != nil {
		return ""
	}
	if suffix := tail[len(dateLayout):]; suffix != "" {
		n, ok := strings.CutPrefix(suffix, "-")
		if !ok || n == "" || strings.Trim(n, "0123456789") != "" {
			return ""
		}
	}
	return tail[:len(dateLayout)]
}

func unwrapPathError(err error) error {
	var pe *os.PathError
	if errors.As(err, &pe) {
		return pe.Err
	}
	return err
}

// currentBranch returns the branch that is checked out, if it is one a rules
// PR is made on.
func (s *session) currentBranch() (string, error) {
	out, err := s.git.run("symbolic-ref", "--quiet", "--short", "HEAD")
	if exitCode(err) == 1 {
		return "", refuse("HEAD is not on a branch; run lct proposal start first, which checks out a %s* branch", BranchPrefix)
	}
	if err != nil {
		return "", err
	}
	branch := strings.TrimSpace(out)
	if !strings.HasPrefix(branch, BranchPrefix) || branch == BranchPrefix {
		return "", refuse("the branch is %s; a rules PR is made only on a %s* branch (run lct proposal start first)", branch, BranchPrefix)
	}
	return branch, nil
}

// ruleFolderList names the rule folders for a message.
func ruleFolderList() string { return strings.Join(RuleFolders, "/, ") + "/" }

func inRuleFolder(path string) bool {
	return slices.ContainsFunc(RuleFolders, func(dir string) bool { return strings.HasPrefix(path, dir+"/") })
}

// ruleChanges returns the paths to commit: everything git status reports
// below the rule folders, new files and deletions included. Any other path
// that is staged or changed, except an untracked file, is a refusal that lists
// them. Untracked files outside the rule folders are not touched.
func (s *session) ruleChanges() ([]string, error) {
	out, err := s.git.run("status", "--porcelain=v1", "-z")
	if err != nil {
		return nil, err
	}
	var inside, outside []string
	entries := strings.Split(out, "\x00")
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		if len(entry) < 4 {
			continue // the end
		}
		xy, paths := entry[:2], []string{entry[3:]}
		if strings.ContainsAny(xy, "RC") && i+1 < len(entries) {
			i++
			paths = append(paths, entries[i]) // the path it was renamed or copied from
		}
		for _, p := range paths {
			switch {
			case inRuleFolder(p):
				inside = append(inside, p)
			case xy != "??":
				outside = append(outside, p)
			}
		}
	}
	if len(outside) > 0 {
		outside = uniqueSorted(outside)
		return nil, refuse("changes outside %s, which a rules PR must not hold (commit, stash or discard them first):\n  %s",
			ruleFolderList(), strings.Join(outside, "\n  "))
	}
	return uniqueSorted(inside), nil
}

// checkIdentity refuses to commit without a git identity from the config or
// the environment: git must not guess one from the host name.
func (s *session) checkIdentity() error {
	for _, ident := range []string{"GIT_AUTHOR_IDENT", "GIT_COMMITTER_IDENT"} {
		if _, err := s.git.run("-c", "user.useConfigOnly=true", "var", ident); err != nil {
			if exitCode(err) < 0 {
				return err
			}
			return refuse("git has no identity to commit with: set user.name and user.email (git config user.name ...), or GIT_AUTHOR_NAME, GIT_AUTHOR_EMAIL, GIT_COMMITTER_NAME and GIT_COMMITTER_EMAIL")
		}
	}
	return nil
}
