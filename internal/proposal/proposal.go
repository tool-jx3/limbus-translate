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
	"io/fs"
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
	// ExistingBodyFile is where Start saves the description the open rules PR
	// has now, relative to the repo root. Publish replaces the description of
	// the PR, so the new one is written from the saved one.
	ExistingBodyFile = "work/proposal/existing-body.md"
	// StartedFile is where Start writes what it decided, the JSON it returns,
	// relative to the repo root. It is never empty, unlike ExcludeFile when no
	// rules PR is open or was closed: the rules-review skill knows it runs in
	// CI, after Start, by this file.
	StartedFile = "work/proposal/started.json"

	dateLayout = "2006-01-02"

	// The marker of a suggestion in the description of a PR.
	markerStart = "<!-- suggestion:"
	markerEnd   = " -->"
)

// RuleFolders are the only folders a rules PR may change, relative to the
// repo root.
var RuleFolders = []string{"guides", "glossary", "rules"}

// credentialPatterns are the beginnings of the credentials a description or a
// rule file must not hold: the prefixes of GitHub's tokens, the user name a
// token goes with in a URL, and the same with its colon in base64 (how the
// extraheader of actions/checkout carries it), a basic Authorization header
// and the prefix of an API key. Both go to a public page (a description, a commit), and what the
// run can read includes the text of the game, which anyone can put words in.
// Fold says that the case does not matter.
var credentialPatterns = []struct {
	text string
	fold bool
}{
	{"ghp_", false}, {"gho_", false}, {"ghs_", false}, {"ghu_", false}, {"github_pat_", false},
	{"x-access-token", true}, {"eC1hY2Nlc3MtdG9rZW46", false}, {"AUTHORIZATION: basic", true}, {"sk-ant-", false},
}

// credentialIn returns the pattern that line holds, or "".
func credentialIn(line string) string {
	lower := strings.ToLower(line)
	for _, p := range credentialPatterns {
		if strings.Contains(line, p.text) || p.fold && strings.Contains(lower, strings.ToLower(p.text)) {
			return p.text
		}
	}
	return ""
}

// maxCredentialFindings is how many lines a refusal lists.
const maxCredentialFindings = 10

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
	// ExistingBody is the path, relative to the repo root, of the file that
	// holds the description of the open PR as it is now (edits by people
	// included); empty when no rules PR is open. Publish refuses a description
	// that lacks a suggestion this one has.
	ExistingBody string `json:"existing_body"`
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
	// State is OPEN, CLOSED or MERGED.
	State  string  `json:"state"`
	Labels []label `json:"labels"`
	// IsCrossRepository is true for a PR from a fork.
	IsCrossRepository bool `json:"isCrossRepository"`
	// MergedAt is null for a PR that was closed without a merge; older gh
	// versions print the zero time instead.
	MergedAt *string `json:"mergedAt"`
}

type label struct {
	Name string `json:"name"`
}

func (p pullRequest) hasLabel() bool {
	return slices.ContainsFunc(p.Labels, func(l label) bool { return l.Name == Label })
}

// isRulesPR says whether the PR belongs to the loop: it has the label, or its
// branch is a rules-proposal one, whatever a person did to the label. A PR from
// a fork never does, labelled or not: anyone can open one from a branch of any
// name, and its text and branch are not ours to read, follow or edit.
func (p pullRequest) isRulesPR() bool {
	return !p.IsCrossRepository && (p.hasLabel() || strings.HasPrefix(p.HeadRefName, BranchPrefix))
}

func (p pullRequest) isOpen() bool { return strings.EqualFold(p.State, "OPEN") }

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
// origin, so the new suggestions are added to the PR. A branch that is reused
// (this one, or one a run left on origin without a PR) also gets origin/<Base>
// merged into it, so that the code, the skill and the rules in the work tree
// are those of the base branch as it is now, not as it was when the PR was
// opened; a merge that conflicts is a refusal and leaves the branch as origin
// has it.
//
// Without an open rules PR the branch is rules-proposal/<Date>, made from
// origin/<Base>, except that origin may have that name already:
//   - no PR was ever opened from it (an earlier run pushed it and failed before
//     the PR): it is checked out as origin has it, and the run goes on;
//   - a PR was (it was closed or merged): the first rules-proposal/<Date>-N,
//     N = 2, 3, ..., that origin does not have, made from origin/<Base>.
//
// A PR that is open for rules-proposal/<Date>, which the listing of the open
// PRs missed, is found by the second question and takes the first path.
//
// It also writes ExcludeFile:
// the ids that lct suggest must leave out, because the open PR has them, people
// closed a PR with them, or they were merged and still stand (Started.Handled);
// ExistingBodyFile, the description of the open PR (the file is removed when
// there is none); and last StartedFile, the result as JSON. The PRs are
// listed without gh's --label filter, which goes through the search index and
// can miss a PR that is new or lost its label; a rules PR is one with the
// label or with a rules-proposal/* branch. Nothing is changed when it fails
// before it checks the branch out; the three files are written last, once the
// branch is checked out.
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

	open, err := s.listPRs("open", "--state", "open", "--json", "number,url,headRefName,body,labels,isCrossRepository", "--limit", "100")
	if err != nil {
		return nil, err
	}
	closed, err := s.listPRs("closed", "--state", "closed", "--json", "number,body,mergedAt,labels,headRefName,isCrossRepository", "--limit", "200")
	if err != nil {
		return nil, err
	}
	notRules := func(p pullRequest) bool { return !p.isRulesPR() }
	open, closed = slices.DeleteFunc(open, notRules), slices.DeleteFunc(closed, notRules)

	// There should be one open PR at most. If there are more, the oldest
	// gets the new suggestions, and the ids of all of them are left out.
	slices.SortFunc(open, func(a, b pullRequest) int { return a.Number - b.Number })
	for _, p := range open {
		if !strings.HasPrefix(p.HeadRefName, BranchPrefix) {
			return nil, refuse("the open PR #%d has the label %s but its branch is %q, not %s*; take the label off or close it",
				p.Number, Label, p.HeadRefName, BranchPrefix)
		}
	}
	var declined, handled []string
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

	res := &Started{}
	var chosen *pullRequest // the open PR that gets the new suggestions
	if len(open) > 0 {
		chosen = &open[0]
		res.Branch = chosen.HeadRefName
		err = s.checkoutFromOrigin(res.Branch)
	} else {
		res.Branch, chosen, err = s.checkoutNewBranch()
		if chosen != nil {
			open = append(open, *chosen)
		}
	}
	if err != nil {
		return nil, err
	}
	var existing []string
	for _, p := range open {
		existing = append(existing, suggestionIDs(p.Body)...)
	}
	if chosen != nil {
		res.PR, res.URL = chosen.Number, chosen.URL
	}
	res.Existing, res.Declined, res.Handled = uniqueSorted(existing), uniqueSorted(declined), uniqueSorted(handled)

	if err := writeExclude(s.Repo, slices.Concat(res.Existing, res.Declined, res.Handled)); err != nil {
		return nil, err
	}
	if res.ExistingBody, err = writeExistingBody(s.Repo, chosen); err != nil {
		return nil, err
	}
	if err := writeStarted(s.Repo, res); err != nil {
		return nil, err
	}
	return res, nil
}

// checkoutFromOrigin fetches the branch, checks it out at what origin has and
// merges origin/<Base> into it. The branch is named in full: a tag of the same
// name would be fetched instead.
func (s *session) checkoutFromOrigin(branch string) error {
	if _, err := s.git.run("fetch", "origin", "refs/heads/"+branch); err != nil {
		return err
	}
	if err := s.refuseLocalCommits(branch, "FETCH_HEAD", "origin"); err != nil {
		return err
	}
	if _, err := s.git.run("checkout", "-q", "-B", branch, "FETCH_HEAD"); err != nil {
		return err
	}
	return s.mergeBase(branch)
}

// isAncestor says whether the commit a is an ancestor of b (or b itself).
func (s *session) isAncestor(a, b string) (bool, error) {
	_, err := s.git.run("merge-base", "--is-ancestor", a, b)
	switch {
	case err == nil:
		return true, nil
	case exitCode(err) == 1:
		return false, nil
	}
	return false, err
}

// mergeBase merges origin/<Base> into the branch that is checked out, unless
// the branch has it already. A merge that is only a fast-forward needs no
// identity; one that makes a commit does. When the two conflict the merge is
// aborted, which leaves the branch at the commit it was checked out at, and
// the conflicting files are named in the refusal.
func (s *session) mergeBase(branch string) error {
	base := "origin/" + s.Base
	if has, err := s.isAncestor(base, "HEAD"); err != nil || has {
		return err
	}
	if forward, err := s.isAncestor("HEAD", base); err != nil {
		return err
	} else if !forward {
		// Both sides have commits of their own: a merge commit is made.
		if err := s.checkIdentity(); err != nil {
			return err
		}
	}
	_, mergeErr := s.git.run("-c", "user.useConfigOnly=true", "merge", "--no-edit", "--no-gpg-sign", base)
	if mergeErr == nil {
		return nil
	}
	if exitCode(mergeErr) != 1 {
		return mergeErr
	}
	out, err := s.git.run("-c", "core.quotePath=false", "diff", "--name-only", "-z", "--diff-filter=U")
	if err != nil {
		return err
	}
	var conflicts []string
	for _, f := range strings.Split(out, "\x00") {
		if f != "" {
			conflicts = append(conflicts, f)
		}
	}
	// A merge that stopped for another reason (a file in the way) has nothing
	// to abort.
	if _, err := s.git.run("rev-parse", "-q", "--verify", "MERGE_HEAD"); err == nil {
		if _, err := s.git.run("merge", "--abort"); err != nil {
			return err
		}
	}
	if len(conflicts) == 0 {
		return mergeErr
	}
	return refuse("the branch %s conflicts with %s in:\n  %s\n"+
		"merge %s into the branch, resolve the conflicts and push it; the branch has been left as origin has it",
		branch, base, strings.Join(conflicts, "\n  "), base)
}

// maxBranchSuffix is where the search for a free rules-proposal/<Date>-N gives
// up. A day with that many closed rules PRs is not a day to go on.
const maxBranchSuffix = 50

// checkoutNewBranch checks out the branch for a run without an open rules PR
// listed and returns its name; see Start for which one it is. When a PR is open
// for the plain name (the listing missed it), that PR is returned too. A name
// that origin has is dealt with here, and not left to the push: that would be
// rejected after the suggestions were reviewed.
func (s *session) checkoutNewBranch() (string, *pullRequest, error) {
	branch := BranchPrefix + s.Date
	onOrigin, err := s.originHas(branch)
	if err != nil {
		return "", nil, err
	}
	if !onOrigin {
		return branch, nil, s.checkoutFromBase(branch)
	}
	used, err := s.pullRequestsOfHead(branch)
	if err != nil {
		return "", nil, err
	}
	if len(used) == 0 {
		return branch, nil, s.checkoutFromOrigin(branch)
	}
	// An open PR, the oldest if there are more, is the PR of this run.
	slices.SortFunc(used, func(a, b pullRequest) int { return a.Number - b.Number })
	if i := slices.IndexFunc(used, pullRequest.isOpen); i >= 0 {
		return branch, &used[i], s.checkoutFromOrigin(branch)
	}
	for n := 2; n <= maxBranchSuffix; n++ {
		next := fmt.Sprintf("%s-%d", branch, n)
		if onOrigin, err := s.originHas(next); err != nil {
			return "", nil, err
		} else if !onOrigin {
			return next, nil, s.checkoutFromBase(next)
		}
	}
	return "", nil, refuse("origin has %s and the branches %s-2 to %s-%d, and a PR was opened from %s; delete some of them", branch, branch, branch, maxBranchSuffix, branch)
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

// pullRequestsOfHead returns the PRs, in any state and with any label, that
// were opened from the branch of this repo; those from forks do not count.
func (s *session) pullRequestsOfHead(branch string) ([]pullRequest, error) {
	prs, err := s.listPRs("any state", "--state", "all", "--head", branch, "--json", "number,url,headRefName,body,state,isCrossRepository")
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(prs, func(p pullRequest) bool { return p.IsCrossRepository || p.HeadRefName != branch }), nil
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

// writeStarted writes res to StartedFile as the JSON lct proposal start
// prints: indented, with no HTML escaping, and a line end.
func writeStarted(repo string, res *Started) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(res); err != nil {
		return err
	}
	path := filepath.Join(repo, filepath.FromSlash(StartedFile))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// writeExistingBody saves the description of the PR to ExistingBodyFile as it
// is, and returns the path; without a PR it removes a file an earlier run left
// and returns "".
func writeExistingBody(repo string, pr *pullRequest) (string, error) {
	path := filepath.Join(repo, filepath.FromSlash(ExistingBodyFile))
	if pr == nil {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		return "", nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	return ExistingBodyFile, os.WriteFile(path, []byte(pr.Body), 0o644)
}

// suggestionIDs returns the ids of the markers <!-- suggestion:<id> --> of a
// PR description, in order and as written. A marker is a line of its own, as
// the rules-review skill writes it and the last step of rules-proposal.yml
// reads it: the line, without the carriage return of a description edited on
// the web page, starts with "<!-- suggestion:" and ends with " -->", and the
// id is everything between the two: it may hold spaces, CJK text, "|" and
// " -->". A marker within a line of text is not one (the opening of the
// description names the marker, and evidence may quote one), nor is a marker
// whose id is blank.
func suggestionIDs(body string) []string {
	var ids []string
	for _, line := range strings.Split(body, "\n") {
		rest, ok := strings.CutPrefix(strings.TrimSuffix(line, "\r"), markerStart)
		if !ok {
			continue
		}
		if id, ok := strings.CutSuffix(rest, markerEnd); ok && strings.TrimSpace(id) != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// brokenMarkers names the lines of body ("line <n>") that start, after
// blanks, with "<!-- suggestion:" and that suggestionIDs does not read as a
// marker: one with blanks before it, text after it, no end or a blank id.
func brokenMarkers(body string) []string {
	var broken []string
	for i, line := range strings.Split(body, "\n") {
		line = strings.TrimSuffix(line, "\r")
		if !strings.HasPrefix(strings.TrimLeft(line, " \t"), markerStart) {
			continue
		}
		if len(suggestionIDs(line)) == 0 {
			broken = append(broken, fmt.Sprintf("line %d", i+1))
		}
	}
	return broken
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
// What a run proposes is the rule files it changed, or the suggestions its
// description adds to the open PR (all of them when there is none): the
// rules-review skill lists the items it is not sure of under 「需要人決定」
// for a person to decide, and changes no file for them. A new PR needs a
// commit that origin/<Base> does not have, so when nothing changed and the
// branch has no such commit, a commit of no change is made.
//
// It refuses, before it changes anything: a branch that is not
// rules-proposal/*, a tracked file changed (staged or not) outside the rule
// folders, a commit of the branch that origin/<Base> does not have and that
// changes a file outside them (the merges are not looked at: those of Start
// bring in what origin/<Base> has), no token for gh, no git identity, a body without markers, nothing
// to propose (no rule file changed, and the body adds no suggestion to the
// open PR), and, when the branch has an open PR, a body that lacks a
// suggestion marker the PR has now (the edit replaces the description, and a
// suggestion that dropped out would be suggested again). Untracked files
// outside the rule folders are left alone and not committed. The commit is
// not signed, like the merge of Start: the identity of a workflow has no key.
// The push names the branch in full on both sides, so that a tag of the same
// name is not pushed instead, and is never forced. A line of the body that
// starts like a marker and is not one is refused with its number: its
// suggestion would not be read, and so not left out.
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
	if broken := brokenMarkers(string(data)); len(broken) > 0 {
		return nil, refuse("the body file %s has lines that start like a marker but are not one, so their suggestions would be lost: %s\n"+
			"a marker is %s<id>%s alone on its line, with nothing before or after it",
			bodyFile, strings.Join(broken, ", "), markerStart, markerEnd)
	}
	bodyIDs := suggestionIDs(string(data))
	if len(bodyIDs) == 0 {
		return nil, refuse("the body file %s has no %s<id>%s marker on a line of its own; every suggestion in the PR needs one", bodyFile, markerStart, markerEnd)
	}
	if found := credentialsIn(bodyFile, string(data), nil); len(found) > 0 {
		return nil, refuse("the body file holds what looks like a credential, and the description is posted to a public page:\n  %s", strings.Join(found, "\n  "))
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
	if err := s.refuseCommitsOutside(); err != nil {
		return nil, err
	}
	if found, err := s.credentialsAdded(changed); err != nil {
		return nil, err
	} else if len(found) > 0 {
		return nil, refuse("the changes of the rule files hold what looks like a credential, and they are committed to a public repo:\n  %s", strings.Join(found, "\n  "))
	}
	out, err := s.git.run("rev-list", "--count", "origin/"+s.Base+"..HEAD")
	if err != nil {
		return nil, err
	}
	ahead, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return nil, fmt.Errorf("git rev-list --count: %q is not a number", strings.TrimSpace(out))
	}
	// A commit is made for the changes, or, when there are none and the
	// branch has no commit of its own, for a new PR to be opened on.
	if len(changed) > 0 || ahead == 0 {
		if err := s.checkIdentity(); err != nil {
			return nil, err
		}
	}

	// A PR of this branch that is open is updated, whatever its label.
	current, err := s.openPROf(branch)
	if err != nil {
		return nil, err
	}
	var have []string
	if current != nil {
		have = suggestionIDs(current.Body)
		if missing := missingIDs(have, bodyIDs); len(missing) > 0 {
			return nil, refuse("the body file %s lacks the suggestions the open PR #%d has, which its description would lose:\n  %s\n"+
				"write the description from the one the PR has now (%s, saved by lct proposal start) and keep their sections",
				bodyFile, current.Number, strings.Join(missing, "\n  "), ExistingBodyFile)
		}
	}
	if len(changed) == 0 && len(missingIDs(bodyIDs, have)) == 0 {
		// Only an open PR has every suggestion of the body.
		return nil, refuse("nothing to propose: no file of %s changed, and the body file adds no suggestion to the open PR #%d",
			ruleFolderList(), current.Number)
	}

	switch {
	case len(changed) > 0:
		// The names are paths, not patterns: a "[" or a "*" in one is a character.
		if _, err := s.git.run(append([]string{"--literal-pathspecs", "add", "-A", "--"}, changed...)...); err != nil {
			return nil, err
		}
		if _, err := s.git.run("-c", "user.useConfigOnly=true", "commit", "-q", "--no-gpg-sign", "-m", "rules: proposal "+s.Date); err != nil {
			return nil, err
		}
	case ahead == 0 && current == nil:
		if _, err := s.git.run("-c", "user.useConfigOnly=true", "commit", "-q", "--no-gpg-sign", "--allow-empty", "-m", "rules: proposal "+s.Date); err != nil {
			return nil, err
		}
	}
	if _, err := s.git.run("push", "origin", "refs/heads/"+branch+":refs/heads/"+branch); err != nil {
		return nil, err
	}
	head, err := s.git.run("rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	res := &Published{Branch: branch, Commit: strings.TrimSpace(head)}

	if current != nil {
		if _, err := s.gh.run("pr", "edit", strconv.Itoa(current.Number), "--body-file", bodyFile); err != nil {
			return nil, err
		}
		res.URL = current.URL
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

// openPROf returns the open PR whose branch is the given one, whatever its
// label, or nil. A PR from a fork is not one.
func (s *session) openPROf(branch string) (*pullRequest, error) {
	open, err := s.listPRs("branch", "--state", "open", "--head", branch, "--json", "number,url,headRefName,body,isCrossRepository")
	if err != nil {
		return nil, err
	}
	for _, p := range open {
		if p.HeadRefName == branch && !p.IsCrossRepository {
			return &p, nil
		}
	}
	return nil, nil
}

// missingIDs returns the ids of have that want does not have, sorted.
func missingIDs(have, want []string) []string {
	var missing []string
	for _, id := range have {
		if !slices.Contains(want, id) {
			missing = append(missing, id)
		}
	}
	return uniqueSorted(missing)
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
// PR is made on. The full name is asked for: the short one of a branch that a
// tag shares its name with is "heads/<branch>".
func (s *session) currentBranch() (string, error) {
	out, err := s.git.run("symbolic-ref", "--quiet", "HEAD")
	if exitCode(err) == 1 {
		return "", refuse("HEAD is not on a branch; run lct proposal start first, which checks out a %s* branch", BranchPrefix)
	}
	if err != nil {
		return "", err
	}
	ref := strings.TrimSpace(out)
	branch, local := strings.CutPrefix(ref, "refs/heads/")
	if !local {
		branch = ref
	}
	if !local || !strings.HasPrefix(branch, BranchPrefix) || branch == BranchPrefix {
		return "", refuse("the branch is %s; a rules PR is made only on a %s* branch (run lct proposal start first)", branch, BranchPrefix)
	}
	return branch, nil
}

// ruleFolderList names the rule folders for a message.
func ruleFolderList() string { return strings.Join(RuleFolders, "/, ") + "/" }

// credentialsIn lists the lines of text that hold a credential pattern, as
// "<where> line <n> holds "<pattern>"" (the text itself is never repeated: the
// refusal ends up in a log). Lines that old has, as many times as old has
// them, are not looked at: only what is added counts.
func credentialsIn(where, text string, old []string) []string {
	have := map[string]int{}
	for _, line := range old {
		have[line]++
	}
	var found []string
	for i, line := range strings.Split(text, "\n") {
		if have[line] > 0 {
			have[line]--
			continue
		}
		if p := credentialIn(line); p != "" && len(found) < maxCredentialFindings {
			found = append(found, fmt.Sprintf("%s line %d holds %q", where, i+1, p))
		}
	}
	return found
}

// credentialsAdded looks for credential patterns in what the commit of the
// rule changes would add: the lines of each changed file that the file does
// not have at HEAD (every line of a new file), and everything in a new folder.
// A rule file that holds such a string already does not stop every later PR.
// It reads the work tree and changes nothing.
func (s *session) credentialsAdded(changed []string) ([]string, error) {
	var found []string
	for _, p := range changed {
		root := filepath.Join(s.Repo, filepath.FromSlash(p))
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return nil // deleted, or the old name of a rename
				}
				return err
			}
			if !d.Type().IsRegular() {
				return nil // a folder, which WalkDir goes into, or a link, which commits no text
			}
			rel, err := filepath.Rel(s.Repo, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var old []string
			if committed, err := s.git.run("show", "HEAD:"+rel); err == nil {
				old = strings.Split(committed, "\n")
			}
			found = append(found, credentialsIn(rel, string(data), old)...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	if len(found) > maxCredentialFindings {
		found = found[:maxCredentialFindings]
	}
	return found, nil
}

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

// refuseCommitsOutside refuses the commits of the branch that origin/<Base>
// does not have, other than merges, when they change a path outside the rule
// folders: the PR would take them to main. Both names of a rename count.
func (s *session) refuseCommitsOutside() error {
	out, err := s.git.run("-c", "core.quotePath=false", "log", "--no-merges", "--no-renames", "--format=", "--name-only", "-z", "origin/"+s.Base+"..HEAD")
	if err != nil {
		return err
	}
	var outside []string
	for _, p := range strings.Split(out, "\x00") {
		if p = strings.TrimLeft(p, "\n"); p != "" && !inRuleFolder(p) {
			outside = append(outside, p)
		}
	}
	if len(outside) == 0 {
		return nil
	}
	return refuse("the branch has commits that origin/%s does not have and that change files outside %s, which a rules PR must not hold:\n  %s\n"+
		"take those commits off the branch first",
		s.Base, ruleFolderList(), strings.Join(uniqueSorted(outside), "\n  "))
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
