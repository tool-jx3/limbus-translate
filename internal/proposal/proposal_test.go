package proposal

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// fakeGH stands in for the gh command: it appends the arguments of each call
// to gh.log next to it (one a line, a "---" line after the call) and answers
// from the environment: FAKE_GH_OPEN and FAKE_GH_CLOSED are the JSON of
// "pr list" for the state asked for (FAKE_GH_ALL for "all", which is the PRs
// of one head branch), FAKE_GH_URL is what "pr create" prints,
// and a call named in FAKE_GH_FAIL (such as "pr create") fails.
const fakeGH = `#!/bin/sh
dir=$(dirname "$0")
{ for a in "$@"; do printf '%s\n' "$a"; done; echo '---'; } >> "$dir/gh.log"
if [ "$1 $2" = "$FAKE_GH_FAIL" ]; then
	echo "gh: HTTP 500 from the fake" >&2
	exit 1
fi
case "$1 $2" in
"pr list")
	state=""
	prev=""
	for a in "$@"; do
		if [ "$prev" = "--state" ]; then state=$a; fi
		prev=$a
	done
	case "$state" in
	open) printf '%s' "${FAKE_GH_OPEN:-[]}" ;;
	closed) printf '%s' "${FAKE_GH_CLOSED:-[]}" ;;
	all) printf '%s' "${FAKE_GH_ALL:-[]}" ;;
	esac
	;;
"pr create") printf '%s\n' "${FAKE_GH_URL:-https://github.com/o/r/pull/7}" ;;
esac
`

const defaultPRURL = "https://github.com/o/r/pull/7"

// ghPR is a pull request as the fake gh prints it.
type ghPR struct {
	Number      int       `json:"number"`
	URL         string    `json:"url,omitempty"`
	HeadRefName string    `json:"headRefName,omitempty"`
	Body        string    `json:"body"`
	State       string    `json:"state,omitempty"`
	Labels      []ghLabel `json:"labels"`
	MergedAt    *string   `json:"mergedAt,omitempty"`
}

type ghLabel struct {
	Name string `json:"name"`
}

// rulesLabel is the label of a rules PR.
var rulesLabel = []ghLabel{{Name: "rules-proposal"}}

func merged() *string { s := "2026-11-03T04:05:06Z"; return &s }

// env is a work clone on main with a bare origin and a fake gh.
type env struct {
	t      *testing.T
	repo   string
	origin string
	gh     string
}

func newEnv(t *testing.T) *env {
	t.Helper()
	// Neither the machine's git settings nor its tokens count.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL", "EMAIL"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
	t.Setenv("GH_TOKEN", "test-token")
	t.Setenv("GITHUB_TOKEN", "")
	for _, k := range []string{"FAKE_GH_OPEN", "FAKE_GH_CLOSED", "FAKE_GH_ALL", "FAKE_GH_URL", "FAKE_GH_FAIL"} {
		t.Setenv(k, "")
	}

	dir := t.TempDir()
	e := &env{t: t, repo: filepath.Join(dir, "repo"), origin: filepath.Join(dir, "origin.git"), gh: filepath.Join(dir, "bin", "gh")}
	if err := os.MkdirAll(filepath.Dir(e.gh), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(e.gh, []byte(fakeGH), 0o755); err != nil {
		t.Fatal(err)
	}
	runIn(t, dir, "git", "init", "-q", "--bare", "-b", "main", e.origin)
	runIn(t, dir, "git", "init", "-q", "-b", "main", e.repo)
	e.git("config", "user.name", "Test")
	e.git("config", "user.email", "test@example.com")
	e.git("config", "commit.gpgsign", "false")
	e.write(".gitignore", "/work/\n/bin/\n")
	e.write("guides/README.md", "guides\n")
	e.write("glossary/terms.yaml", "- {kr: 도약 모듈, zh: 跳躍成長模組, status: approved}\n")
	e.write("rules/declined.yaml", "[]\n")
	e.write("internal/x.go", "package x\n")
	e.write(".github/workflows/ci.yml", "name: ci\n")
	e.write("docs/說明 文件.md", "doc\n")
	e.git("add", "-A")
	e.git("commit", "-q", "-m", "init")
	e.git("remote", "add", "origin", e.origin)
	e.git("push", "-q", "-u", "origin", "main")
	return e
}

func runIn(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	// Only the end is trimmed: a line of git status starts with a blank.
	return strings.TrimRight(string(out), "\n")
}

func (e *env) git(args ...string) string { return runIn(e.t, e.repo, "git", args...) }

func (e *env) originGit(args ...string) string { return runIn(e.t, e.origin, "git", args...) }

func (e *env) write(rel, content string) {
	e.t.Helper()
	path := filepath.Join(e.repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		e.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		e.t.Fatal(err)
	}
}

func (e *env) read(rel string) string {
	e.t.Helper()
	data, err := os.ReadFile(filepath.Join(e.repo, filepath.FromSlash(rel)))
	if err != nil {
		e.t.Fatal(err)
	}
	return string(data)
}

func (e *env) opts(date string) Options {
	return Options{Repo: e.repo, Date: date, Git: "git", GH: e.gh}
}

func (e *env) setPRs(open, closed []ghPR) {
	e.t.Helper()
	e.setPRLists(map[string][]ghPR{"FAKE_GH_OPEN": open, "FAKE_GH_CLOSED": closed})
}

// setAllPRs sets the answer to the question which PRs, in any state, have a
// head branch.
func (e *env) setAllPRs(prs ...ghPR) {
	e.t.Helper()
	e.setPRLists(map[string][]ghPR{"FAKE_GH_ALL": prs})
}

func (e *env) setPRLists(lists map[string][]ghPR) {
	e.t.Helper()
	for k, list := range lists {
		if list == nil {
			continue
		}
		data, err := json.Marshal(list)
		if err != nil {
			e.t.Fatal(err)
		}
		e.t.Setenv(k, string(data))
	}
}

// calls returns the gh calls so far, each as its arguments.
func (e *env) calls() [][]string {
	e.t.Helper()
	data, err := os.ReadFile(filepath.Join(filepath.Dir(e.gh), "gh.log"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		e.t.Fatal(err)
	}
	var calls [][]string
	var cur []string
	for _, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		if line == "---" {
			calls = append(calls, cur)
			cur = nil
		} else {
			cur = append(cur, line)
		}
	}
	return calls
}

// callsOf returns the gh calls that start with the words.
func (e *env) callsOf(words ...string) [][]string {
	var out [][]string
	for _, c := range e.calls() {
		if len(c) >= len(words) && slices.Equal(c[:len(words)], words) {
			out = append(out, c)
		}
	}
	return out
}

// pushBranch makes branch exist on origin with one commit that changes file,
// and leaves no trace of it in the work clone but the checkout of main.
func (e *env) pushBranch(branch, file, content string) string {
	e.t.Helper()
	e.git("checkout", "-q", "-B", branch, "origin/main")
	e.write(file, content)
	e.git("commit", "-q", "-am", "remote work")
	sha := e.git("rev-parse", "HEAD")
	e.git("push", "-q", "origin", branch)
	e.git("checkout", "-q", "main")
	e.git("branch", "-q", "-D", branch)
	e.git("branch", "-q", "-dr", "origin/"+branch)
	return sha
}

func (e *env) branch() string { return e.git("branch", "--show-current") }

func (e *env) excludeIDs() []string {
	e.t.Helper()
	data := e.read("work/proposal/exclude.txt")
	if data == "" {
		return []string{}
	}
	return strings.Split(strings.TrimSuffix(data, "\n"), "\n")
}

func body(ids ...string) string {
	var b strings.Builder
	b.WriteString("## 規則建議\n\n")
	for _, id := range ids {
		b.WriteString("### " + id + "\n<!-- suggestion:" + id + " -->\n理由與證據\n\n")
	}
	return b.String()
}

func isRefusal(err error) bool {
	var r *Refusal
	return errors.As(err, &r)
}

func mustRefuse(t *testing.T, err error, parts ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("no error, want a refusal with %q", parts)
	}
	if !isRefusal(err) {
		t.Fatalf("error = %v, want a *Refusal", err)
	}
	for _, p := range parts {
		if !strings.Contains(err.Error(), p) {
			t.Errorf("error = %q, want it to hold %q", err, p)
		}
	}
}

func assertIDs(t *testing.T, what string, got []string, want ...string) {
	t.Helper()
	if want == nil {
		want = []string{}
	}
	if got == nil || !slices.Equal(got, want) {
		t.Errorf("%s = %#v, want %#v", what, got, want)
	}
}

func TestProposalStartNewBranch(t *testing.T) {
	e := newEnv(t)
	// A description saved by an earlier run is not the description of a PR
	// that is not there.
	e.write("work/proposal/existing-body.md", "stale\n")
	got, err := Start(e.opts("2026-11-05"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(e.repo, "work", "proposal", "existing-body.md")); err == nil {
		t.Error("the stale existing-body.md is still there")
	}
	if got.ExistingBody != "" {
		t.Errorf("ExistingBody = %q, want none", got.ExistingBody)
	}
	if got.Branch != "rules-proposal/2026-11-05" || got.PR != 0 || got.URL != "" {
		t.Errorf("Started = %+v", got)
	}
	assertIDs(t, "Existing", got.Existing)
	assertIDs(t, "Declined", got.Declined)
	assertIDs(t, "Handled", got.Handled)
	if b := e.branch(); b != "rules-proposal/2026-11-05" {
		t.Errorf("checked out %q", b)
	}
	if head, base := e.git("rev-parse", "HEAD"), e.git("rev-parse", "origin/main"); head != base {
		t.Errorf("HEAD = %s, want origin/main = %s", head, base)
	}
	assertIDs(t, "exclude.txt", e.excludeIDs())

	// Both lists are asked for with exactly these arguments: no --label,
	// which makes gh use the search index, and so can miss a PR that is
	// new or lost its label. The rules PRs are picked out here.
	open := e.callsOf("pr", "list", "--state", "open")
	if len(open) != 1 || !slices.Equal(open[0], []string{"pr", "list", "--state", "open", "--json", "number,url,headRefName,body,labels", "--limit", "100"}) {
		t.Errorf("open list calls = %q", open)
	}
	closed := e.callsOf("pr", "list", "--state", "closed")
	if len(closed) != 1 || !slices.Equal(closed[0], []string{"pr", "list", "--state", "closed", "--json", "number,body,mergedAt,labels,headRefName", "--limit", "200"}) {
		t.Errorf("closed list calls = %q", closed)
	}
	for _, c := range e.calls() {
		if slices.Contains(c, "--label") {
			t.Errorf("gh was called with --label: %q", c)
		}
	}
}

func TestProposalStartReusesTheOpenPR(t *testing.T) {
	e := newEnv(t)
	const head = "rules-proposal/2026-10-29"
	sha := e.pushBranch(head, "glossary/terms.yaml", "- {kr: 신규 용어, zh: 新術語, status: approved}\n")
	prBody := strings.ReplaceAll(body("speaker/새 화자", "field/add/BattleSpeechBubbleDlg/desc"), "\n", "\r\n") + "edited in the web page\r\n"
	e.setPRs([]ghPR{{Number: 5, URL: "https://github.com/o/r/pull/5", HeadRefName: head, Labels: rulesLabel, Body: prBody}}, nil)

	got, err := Start(e.opts("2026-11-05"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Branch != head || got.PR != 5 || got.URL != "https://github.com/o/r/pull/5" {
		t.Errorf("Started = %+v", got)
	}
	assertIDs(t, "Existing", got.Existing, "field/add/BattleSpeechBubbleDlg/desc", "speaker/새 화자")
	assertIDs(t, "Declined", got.Declined)
	assertIDs(t, "exclude.txt", e.excludeIDs(), "field/add/BattleSpeechBubbleDlg/desc", "speaker/새 화자")
	// The description the PR has now is saved as it is: publish replaces it,
	// so the new one has to be written from this one.
	if got.ExistingBody != "work/proposal/existing-body.md" || e.read(got.ExistingBody) != prBody {
		t.Errorf("ExistingBody = %q holding %q, want the PR's description %q", got.ExistingBody, e.read("work/proposal/existing-body.md"), prBody)
	}
	if b := e.branch(); b != head {
		t.Errorf("checked out %q, want %q", b, head)
	}
	if h := e.git("rev-parse", "HEAD"); h != sha {
		t.Errorf("HEAD = %s, want the PR's %s", h, sha)
	}
}

func TestProposalStartCollectsClosedPRs(t *testing.T) {
	e := newEnv(t)
	e.setPRs(nil, []ghPR{
		{Number: 11, Labels: rulesLabel, Body: body("speaker/새 화자", "inconsistent/ally/name/fd93b875"), MergedAt: merged()},
		{Number: 12, Labels: rulesLabel, Body: body("term/신규 용어", "family/Zzz-twth.json", "inconsistent/ally/name/00000001")},
		{Number: 13, Labels: rulesLabel, Body: body("note/StoryData/S001A.json|/dataList/3/content/3713708f"), MergedAt: merged()},
		// The label was taken off, but the branch says what it is.
		{Number: 14, HeadRefName: "rules-proposal/2026-10-01", Body: body("family/Yyy.json")},
		// Not a rules PR: another label, another branch.
		{Number: 15, HeadRefName: "feature/x", Labels: []ghLabel{{Name: "bug"}}, Body: body("family/Other.json")},
	})
	got, err := Start(e.opts("2026-11-05"))
	if err != nil {
		t.Fatal(err)
	}
	// Only the PR that was closed without a merge is declined; a merged one
	// resolves its suggestions by itself, but the inconsistency of what was
	// translated before stays and would be suggested again.
	assertIDs(t, "Declined", got.Declined, "family/Yyy.json", "family/Zzz-twth.json", "inconsistent/ally/name/00000001", "term/신규 용어")
	assertIDs(t, "Handled", got.Handled, "inconsistent/ally/name/fd93b875")
	assertIDs(t, "Existing", got.Existing)
	assertIDs(t, "exclude.txt", e.excludeIDs(),
		"family/Yyy.json", "family/Zzz-twth.json", "inconsistent/ally/name/00000001", "inconsistent/ally/name/fd93b875", "term/신규 용어")
}

func TestProposalStartReadsMergedAtTheWayGHPrintsIt(t *testing.T) {
	e := newEnv(t)
	// gh prints null for a PR that was not merged, and older versions the
	// zero time.
	t.Setenv("FAKE_GH_CLOSED", `[
		{"number":1,"labels":[{"name":"rules-proposal"}],"body":"<!-- suggestion:term/a -->","mergedAt":null},
		{"number":2,"labels":[{"name":"rules-proposal"}],"body":"<!-- suggestion:term/b -->","mergedAt":"0001-01-01T00:00:00Z"},
		{"number":3,"labels":[{"name":"rules-proposal"}],"body":"<!-- suggestion:term/c -->","mergedAt":""},
		{"number":4,"labels":[{"name":"rules-proposal"}],"body":"<!-- suggestion:term/d -->"},
		{"number":5,"labels":[{"name":"rules-proposal"}],"body":"<!-- suggestion:inconsistent/e -->","mergedAt":"2026-11-03T04:05:06Z"}]`)
	got, err := Start(e.opts("2026-11-05"))
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, "Declined", got.Declined, "term/a", "term/b", "term/c", "term/d")
	assertIDs(t, "Handled", got.Handled, "inconsistent/e")
}

func TestProposalStartKeepsIDsAsFound(t *testing.T) {
	e := newEnv(t)
	moved := "adjustment-moved/StoryData/S1.json|/dataList/2/content"
	name := "speaker/아무개  (두 칸)"
	open := "<!-- suggestion:" + name + " -->\r\ntext <!-- suggestion:" + moved + " --> trailing -->\r\n" +
		"<!-- suggestion:" + name + " -->\n<!-- suggestion:broken\n<!-- suggestion: -->\n"
	e.setPRs([]ghPR{{Number: 5, URL: "u", HeadRefName: "rules-proposal/2026-10-29", Body: open}}, nil)
	e.pushBranch("rules-proposal/2026-10-29", "guides/README.md", "x\n")

	got, err := Start(e.opts("2026-11-05"))
	if err != nil {
		t.Fatal(err)
	}
	// The id is everything up to the last " -->" of its marker: the spaces
	// stay, the | stays, and a repeated id is one id.
	assertIDs(t, "Existing", got.Existing, moved+" --> trailing", name)
	assertIDs(t, "exclude.txt", e.excludeIDs(), moved+" --> trailing", name)
}

func TestProposalStartRefusesAnOpenPRFromAnotherBranch(t *testing.T) {
	e := newEnv(t)
	e.setPRs([]ghPR{{Number: 5, URL: "u", HeadRefName: "feature/x", Labels: rulesLabel, Body: body("term/a")}}, nil)
	_, err := Start(e.opts("2026-11-05"))
	mustRefuse(t, err, "#5", "feature/x")
	if b := e.branch(); b != "main" {
		t.Errorf("checked out %q, the refusal must leave main", b)
	}
	if _, statErr := os.Stat(filepath.Join(e.repo, "work", "proposal", "exclude.txt")); statErr == nil {
		t.Error("exclude.txt was written")
	}
}

func TestProposalStartResumesABranchNoPRWasOpenedFrom(t *testing.T) {
	e := newEnv(t)
	const branch = "rules-proposal/2026-11-05"
	// An earlier run pushed the branch, and its step that opens the PR failed.
	sha := e.pushBranch(branch, "glossary/terms.yaml", "- {kr: 신규 용어, zh: 新術語, status: approved}\n")

	got, err := Start(e.opts("2026-11-05"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Branch != branch || got.PR != 0 || got.URL != "" {
		t.Errorf("Started = %+v", got)
	}
	if b, h := e.branch(), e.git("rev-parse", "HEAD"); b != branch || h != sha {
		t.Errorf("checked out %q at %s, want %q at the pushed %s", b, h, branch, sha)
	}
	// The question was asked of every PR, whatever its state and label.
	asked := e.callsOf("pr", "list", "--state", "all")
	if len(asked) != 1 || !slices.Equal(asked[0], []string{"pr", "list", "--state", "all", "--head", branch, "--json", "number,url,headRefName,body,state"}) {
		t.Errorf("calls = %q", asked)
	}
	assertIDs(t, "exclude.txt", e.excludeIDs())

	// Publish goes on from there: the branch gets its PR.
	e.write("work/proposal/body.md", body("term/신규 용어"))
	e.write("guides/README.md", "more\n")
	pub, err := Publish(e.opts("2026-11-05"), e.bodyFile())
	if err != nil {
		t.Fatal(err)
	}
	if !pub.Created || pub.Branch != branch || e.originGit("rev-parse", "refs/heads/"+branch) != pub.Commit {
		t.Errorf("Published = %+v", pub)
	}
}

func TestProposalStartTakesAFreeBranchWhenAClosedPRUsedTheName(t *testing.T) {
	e := newEnv(t)
	const branch = "rules-proposal/2026-11-05"
	e.pushBranch(branch, "guides/README.md", "closed PR work\n")
	// A closed PR was opened from the branch; one that was merged would be the same.
	e.setAllPRs(ghPR{Number: 9, HeadRefName: branch, State: "CLOSED", Body: body("term/a")})

	got, err := Start(e.opts("2026-11-05"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Branch != branch+"-2" || got.PR != 0 {
		t.Errorf("Started = %+v, want %s-2", got, branch)
	}
	if b, h, base := e.branch(), e.git("rev-parse", "HEAD"), e.git("rev-parse", "origin/main"); b != branch+"-2" || h != base {
		t.Errorf("checked out %q at %s, want %s-2 at origin/main %s", b, h, branch, base)
	}

	// -2 taken on origin as well: the first one origin does not have.
	e.pushBranch(branch+"-2", "guides/README.md", "second\n")
	e.pushBranch(branch+"-4", "guides/README.md", "fourth\n")
	got, err = Start(e.opts("2026-11-05"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Branch != branch+"-3" || e.branch() != branch+"-3" {
		t.Errorf("Started = %+v, checked out %q, want %s-3", got, e.branch(), branch)
	}

	// A PR from another branch is no reason: only the head counts.
	e.git("checkout", "-q", "main")
	e.setAllPRs(ghPR{Number: 9, HeadRefName: "rules-proposal/2026-11-04", State: "OPEN"})
	got, err = Start(e.opts("2026-11-05"))
	if err != nil || got.Branch != branch {
		t.Errorf("Started = %+v, %v; want to resume %s", got, err, branch)
	}
}

func TestProposalStartGivesUpWhenEveryBranchIsTaken(t *testing.T) {
	e := newEnv(t)
	const branch = "rules-proposal/2026-11-05"
	e.setAllPRs(ghPR{Number: 9, HeadRefName: branch, State: "MERGED"})
	e.git("checkout", "-q", "-B", branch, "origin/main")
	e.write("guides/README.md", "x\n")
	e.git("commit", "-q", "-am", "x")
	refspecs := []string{"HEAD:refs/heads/" + branch}
	for n := 2; n <= maxBranchSuffix; n++ {
		refspecs = append(refspecs, "HEAD:refs/heads/"+branch+"-"+strconv.Itoa(n))
	}
	e.git(append([]string{"push", "-q", "origin"}, refspecs...)...)
	e.git("checkout", "-q", "main")
	e.git("branch", "-q", "-D", branch)

	_, err := Start(e.opts("2026-11-05"))
	mustRefuse(t, err, branch)
	if b := e.branch(); b != "main" {
		t.Errorf("checked out %q", b)
	}
}

func TestProposalStartRefusesALocalBranchThatDiverged(t *testing.T) {
	e := newEnv(t)
	const head = "rules-proposal/2026-10-29"
	e.pushBranch(head, "guides/README.md", "remote\n")
	e.git("checkout", "-q", "-b", head, "origin/main")
	e.write("guides/README.md", "local only\n")
	e.git("commit", "-q", "-am", "local only")
	local := e.git("rev-parse", "HEAD")
	e.git("checkout", "-q", "main")
	e.setPRs([]ghPR{{Number: 5, URL: "u", HeadRefName: head, Body: body("term/a")}}, nil)

	_, err := Start(e.opts("2026-11-05"))
	mustRefuse(t, err, head, "not on origin")
	if got := e.git("rev-parse", head); got != local {
		t.Errorf("local branch moved to %s, want %s kept", got, local)
	}
}

func TestProposalStartRefusesToDropLocalCommitsOfANewBranch(t *testing.T) {
	e := newEnv(t)
	const branch = "rules-proposal/2026-11-05"
	e.git("checkout", "-q", "-b", branch, "origin/main")
	e.write("guides/README.md", "written by hand, never pushed\n")
	e.git("commit", "-q", "-am", "by hand")
	local := e.git("rev-parse", "HEAD")
	e.git("checkout", "-q", "main")

	_, err := Start(e.opts("2026-11-05"))
	mustRefuse(t, err, branch, "origin/main")
	if got := e.git("rev-parse", branch); got != local {
		t.Errorf("local branch moved to %s, want %s kept", got, local)
	}

	// A local branch with nothing of its own is made again from the base.
	e.git("branch", "-q", "-f", branch, "origin/main")
	if _, err := Start(e.opts("2026-11-05")); err != nil {
		t.Error(err)
	}
}

func TestProposalStartNeedsAToken(t *testing.T) {
	e := newEnv(t)
	t.Setenv("GH_TOKEN", "")
	_, err := Start(e.opts("2026-11-05"))
	mustRefuse(t, err, "GH_TOKEN", "GITHUB_TOKEN")
	if n := len(e.calls()); n != 0 {
		t.Errorf("gh was called %d times without a token", n)
	}
	if b := e.branch(); b != "main" {
		t.Errorf("checked out %q", b)
	}

	t.Setenv("GITHUB_TOKEN", "from-the-workflow")
	if _, err := Start(e.opts("2026-11-05")); err != nil {
		t.Errorf("GITHUB_TOKEN alone: %v", err)
	}
}

func TestProposalStartNeedsADate(t *testing.T) {
	e := newEnv(t)
	for _, date := range []string{"", "2026-11-5", "05/11/2026", "2026-13-01", "tomorrow"} {
		o := e.opts(date)
		if _, err := Start(o); !isRefusal(err) {
			t.Errorf("date %q: error = %v, want a refusal", date, err)
		}
	}
}

func TestProposalStartSystemErrors(t *testing.T) {
	e := newEnv(t)
	t.Setenv("FAKE_GH_FAIL", "pr list")
	_, err := Start(e.opts("2026-11-05"))
	if err == nil || isRefusal(err) || !strings.Contains(err.Error(), "HTTP 500 from the fake") || !strings.Contains(err.Error(), "gh pr list") {
		t.Errorf("error = %v, want the failure of gh pr list", err)
	}
	if b := e.branch(); b != "main" {
		t.Errorf("checked out %q after a failure to list", b)
	}

	t.Setenv("FAKE_GH_FAIL", "")
	t.Setenv("FAKE_GH_OPEN", "not json")
	if _, err := Start(e.opts("2026-11-05")); err == nil || isRefusal(err) || !strings.Contains(err.Error(), "open") {
		t.Errorf("error = %v, want a system error about the open pull requests", err)
	}
}

func TestProposalSuggestionIDs(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{"none", "no markers here\n", nil},
		{"one", "<!-- suggestion:term/a -->", []string{"term/a"}},
		{"cjk and spaces", "<!-- suggestion:speaker/새 화자 -->\n<!-- suggestion:term/跳躍 成長 -->", []string{"speaker/새 화자", "term/跳躍 成長"}},
		{"a unit key", "<!-- suggestion:adjustment-expired/StoryData/S1.json|/dataList/2/content -->", []string{"adjustment-expired/StoryData/S1.json|/dataList/2/content"}},
		{"crlf", "a\r\n<!-- suggestion:term/a -->\r\nb\r\n", []string{"term/a"}},
		{"text around", "- x <!-- suggestion:term/a --> y\n", []string{"term/a"}},
		{"the last --> of the marker", "<!-- suggestion:note/k --> z --> w\n", []string{"note/k --> z"}},
		{"no end", "<!-- suggestion:term/a\nnext line -->\n", nil},
		{"empty", "<!-- suggestion: -->", nil},
		{"two on a line", "<!-- suggestion:term/a --><!-- suggestion:term/b -->", []string{"term/a", "term/b"}},
		{"in order, repeated", "<!-- suggestion:b -->\n<!-- suggestion:a -->\n<!-- suggestion:b -->", []string{"b", "a", "b"}},
		{"not a marker", "<!--suggestion:term/a -->\n<!-- suggestions:term/b -->\n", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := suggestionIDs(tt.body); !slices.Equal(got, tt.want) {
				t.Errorf("suggestionIDs(%q) = %q, want %q", tt.body, got, tt.want)
			}
		})
	}
}

// onBranch starts the PR branch the way Start does, and writes the body.
func (e *env) onBranch(branch string, ids ...string) {
	e.t.Helper()
	e.git("checkout", "-q", "-B", branch, "origin/main")
	e.write("work/proposal/body.md", body(append([]string{"term/신규 용어"}, ids...)...))
}

func (e *env) bodyFile() string { return filepath.Join(e.repo, "work", "proposal", "body.md") }

func (e *env) statusLines() string { return e.git("status", "--porcelain") }

func TestProposalPublishCreatesPR(t *testing.T) {
	e := newEnv(t)
	const branch = "rules-proposal/2026-11-05"
	e.onBranch(branch)
	e.write("glossary/terms.yaml", e.read("glossary/terms.yaml")+"- {kr: 신규 용어, zh: 新術語, status: approved}\n")
	mainBefore := e.originGit("rev-parse", "refs/heads/main")

	got, err := Publish(e.opts("2026-11-05"), e.bodyFile())
	if err != nil {
		t.Fatal(err)
	}
	head := e.git("rev-parse", "HEAD")
	if got.Branch != branch || got.URL != defaultPRURL || got.Commit != head || !got.Created {
		t.Errorf("Published = %+v, want branch %s, url %s, commit %s, created", got, branch, defaultPRURL, head)
	}

	// The commit holds the rule file and nothing else, and says nothing but
	// its subject.
	if files := e.git("show", "--name-only", "--format=", "HEAD"); files != "glossary/terms.yaml" {
		t.Errorf("commit files = %q", files)
	}
	if msg := e.git("log", "-1", "--format=%B"); msg != "rules: proposal 2026-11-05" {
		t.Errorf("commit message = %q", msg)
	}
	if parent := e.git("rev-parse", "HEAD~1"); parent != e.git("rev-parse", "origin/main") {
		t.Errorf("the commit is on %s, not on origin/main", parent)
	}
	if pushed := e.originGit("rev-parse", "refs/heads/"+branch); pushed != head {
		t.Errorf("origin has %s at %s, want %s", branch, pushed, head)
	}
	if main := e.originGit("rev-parse", "refs/heads/main"); main != mainBefore {
		t.Errorf("origin main moved to %s", main)
	}

	create := e.callsOf("pr", "create")
	want := []string{"pr", "create", "--base", "main", "--head", branch, "--title", "規則建議：2026-11-05", "--label", "rules-proposal", "--body-file", e.bodyFile()}
	if len(create) != 1 || !slices.Equal(create[0], want) {
		t.Errorf("pr create calls = %q, want %q", create, want)
	}
	// The label exists before the PR asks for it.
	var order []string
	for _, c := range e.calls() {
		order = append(order, strings.Join(c[:2], " "))
	}
	if want := []string{"pr list", "label create", "pr create"}; !slices.Equal(order, want) {
		t.Errorf("gh calls = %q, want %q", order, want)
	}
	if label := e.callsOf("label", "create"); len(label) != 1 || !slices.Equal(label[0], []string{"label", "create", "rules-proposal", "--force", "--color", "0E8A16", "--description", "規則建議，由 CI 提出"}) {
		t.Errorf("label calls = %q", label)
	}
	if e.callsOf("pr", "edit") != nil {
		t.Error("pr edit was called for a new PR")
	}
	if st := e.statusLines(); st != "" {
		t.Errorf("the clone is not clean afterwards:\n%s", st)
	}
}

func TestProposalPublishUpdatesTheOpenPR(t *testing.T) {
	e := newEnv(t)
	const branch = "rules-proposal/2026-10-29"
	e.onBranch(branch)
	e.write("guides/README.md", "guides, changed\n")
	e.setPRs([]ghPR{
		{Number: 3, URL: "https://github.com/o/r/pull/3", HeadRefName: "rules-proposal/2026-10-01"},
		{Number: 5, URL: "https://github.com/o/r/pull/5", HeadRefName: branch},
	}, nil)

	got, err := Publish(e.opts("2026-11-05"), e.bodyFile())
	if err != nil {
		t.Fatal(err)
	}
	if got.Branch != branch || got.URL != "https://github.com/o/r/pull/5" || got.Created || got.Commit != e.git("rev-parse", "HEAD") {
		t.Errorf("Published = %+v", got)
	}
	if edit := e.callsOf("pr", "edit"); len(edit) != 1 || !slices.Equal(edit[0], []string{"pr", "edit", "5", "--body-file", e.bodyFile()}) {
		t.Errorf("pr edit calls = %q", edit)
	}
	if e.callsOf("pr", "create") != nil || e.callsOf("label", "create") != nil {
		t.Errorf("a second PR was opened: %q", e.calls())
	}
	// The commit carries the date of this run, not that of the branch.
	if msg := e.git("log", "-1", "--format=%s"); msg != "rules: proposal 2026-11-05" {
		t.Errorf("commit message = %q", msg)
	}
	if pushed := e.originGit("rev-parse", "refs/heads/"+branch); pushed != got.Commit {
		t.Errorf("origin has %s, want %s", pushed, got.Commit)
	}
	list := e.callsOf("pr", "list")
	if len(list) != 1 || !slices.Equal(list[0], []string{"pr", "list", "--state", "open", "--head", branch, "--json", "number,url,headRefName,body"}) {
		t.Errorf("pr list calls = %q", list)
	}
}

func TestProposalPublishTakesTheDateFromTheBranch(t *testing.T) {
	e := newEnv(t)
	e.onBranch("rules-proposal/2026-11-05")
	e.write("guides/README.md", "changed\n")
	if _, err := Publish(e.opts(""), e.bodyFile()); err != nil {
		t.Fatal(err)
	}
	if msg := e.git("log", "-1", "--format=%s"); msg != "rules: proposal 2026-11-05" {
		t.Errorf("commit message = %q", msg)
	}
	if create := e.callsOf("pr", "create"); len(create) != 1 || !slices.Contains(create[0], "規則建議：2026-11-05") {
		t.Errorf("pr create calls = %q", create)
	}

	// The suffix of a branch that Start took because the name was used is no
	// part of the date.
	e.onBranch("rules-proposal/2026-11-06-3")
	e.write("guides/README.md", "changed with a suffix\n")
	if _, err := Publish(e.opts(""), e.bodyFile()); err != nil {
		t.Fatal(err)
	}
	if msg := e.git("log", "-1", "--format=%s"); msg != "rules: proposal 2026-11-06" {
		t.Errorf("commit message = %q", msg)
	}

	// A branch without a date in its name needs the date.
	e.onBranch("rules-proposal/manual")
	e.write("guides/README.md", "changed again\n")
	_, err := Publish(e.opts(""), e.bodyFile())
	mustRefuse(t, err, "--date")
	_, err = Publish(e.opts("11/05"), e.bodyFile())
	mustRefuse(t, err, "11/05")
}

func TestProposalPublishRefusesOtherBranches(t *testing.T) {
	for _, branch := range []string{"main", "tw", "raw", "rules-proposal", "dev/x", "detached"} {
		t.Run(branch, func(t *testing.T) {
			e := newEnv(t)
			switch branch {
			case "detached":
				e.git("checkout", "-q", "--detach")
			case "main":
			default:
				e.git("checkout", "-q", "-b", branch)
			}
			e.write("work/proposal/body.md", body("term/a"))
			e.write("glossary/terms.yaml", "- {kr: x, zh: y, status: approved}\n")
			headBefore := e.git("rev-parse", "HEAD")
			branchesBefore := e.originGit("for-each-ref", "--format=%(refname) %(objectname)")

			_, err := Publish(e.opts("2026-11-05"), e.bodyFile())
			mustRefuse(t, err, "rules-proposal/")
			if head := e.git("rev-parse", "HEAD"); head != headBefore {
				t.Errorf("a commit was made: HEAD %s -> %s", headBefore, head)
			}
			if now := e.originGit("for-each-ref", "--format=%(refname) %(objectname)"); now != branchesBefore {
				t.Errorf("origin changed:\n%s\n->\n%s", branchesBefore, now)
			}
			if n := len(e.calls()); n != 0 {
				t.Errorf("gh was called %d times", n)
			}
			if st := e.statusLines(); !strings.Contains(st, " M glossary/terms.yaml") {
				t.Errorf("the change was staged or lost:\n%s", st)
			}
		})
	}
}

func TestProposalPublishRefusesFilesOutsideRules(t *testing.T) {
	e := newEnv(t)
	e.onBranch("rules-proposal/2026-11-05")
	e.write("glossary/terms.yaml", "- {kr: x, zh: y, status: approved}\n")
	e.write("internal/x.go", "package x // changed\n")
	e.write("docs/說明 文件.md", "changed\n")
	e.write(".github/workflows/ci.yml", "name: changed\n")
	e.git("add", ".github/workflows/ci.yml") // staged, the others are not
	e.git("mv", "guides/README.md", "README-moved.md")
	before := e.statusLines()
	headBefore := e.git("rev-parse", "HEAD")

	_, err := Publish(e.opts("2026-11-05"), e.bodyFile())
	mustRefuse(t, err, "internal/x.go", ".github/workflows/ci.yml", "docs/說明 文件.md", "README-moved.md")
	if strings.Contains(err.Error(), "glossary/terms.yaml") {
		t.Errorf("the refusal lists a rule file: %v", err)
	}
	if head := e.git("rev-parse", "HEAD"); head != headBefore {
		t.Errorf("a commit was made")
	}
	if now := e.statusLines(); now != before {
		t.Errorf("the index or the tree changed:\n%s\n->\n%s", before, now)
	}
	if branches := e.originGit("for-each-ref", "refs/heads"); strings.Contains(branches, "rules-proposal") {
		t.Errorf("the branch was pushed:\n%s", branches)
	}
	if n := len(e.calls()); n != 0 {
		t.Errorf("gh was called %d times", n)
	}
}

func TestProposalPublishLeavesUntrackedFilesOutsideRules(t *testing.T) {
	e := newEnv(t)
	e.onBranch("rules-proposal/2026-11-05")
	e.write("rules/requeue/2026-11-05-a.yaml", "- key: x\n")
	e.write("rules/newdir/deeper/b.yaml", "- key: y\n")
	e.write("guides/characters/新 角色.md", "角色\n")
	e.write("rules/requeue/x[1]*.yaml", "- key: z\n")
	e.write("scratch.txt", "mine\n")
	e.write("notes/todo.md", "mine\n")
	if err := os.Remove(filepath.Join(e.repo, "rules", "declined.yaml")); err != nil {
		t.Fatal(err)
	}

	if _, err := Publish(e.opts("2026-11-05"), e.bodyFile()); err != nil {
		t.Fatal(err)
	}
	files := strings.Split(e.git("-c", "core.quotePath=false", "show", "--name-only", "--format=", "HEAD"), "\n")
	slices.Sort(files)
	want := []string{"guides/characters/新 角色.md", "rules/declined.yaml", "rules/newdir/deeper/b.yaml", "rules/requeue/2026-11-05-a.yaml", "rules/requeue/x[1]*.yaml"}
	if !slices.Equal(files, want) {
		t.Errorf("commit files = %q, want %q", files, want)
	}
	if st := e.statusLines(); !strings.Contains(st, "?? scratch.txt") || !strings.Contains(st, "?? notes/") || strings.Count(st, "\n") != 1 {
		t.Errorf("the files outside the rules should stay untracked and nothing else is left:\n%s", st)
	}
}

func TestProposalPublishNeedsSuggestionMarkers(t *testing.T) {
	for name, content := range map[string]string{
		"none":      "規則建議，沒有標記\n",
		"empty id":  "<!-- suggestion: -->\n",
		"no end":    "<!-- suggestion:term/a\n",
		"empty":     "",
		"not exact": "<!--suggestion:term/a -->\n",
	} {
		t.Run(name, func(t *testing.T) {
			e := newEnv(t)
			e.onBranch("rules-proposal/2026-11-05")
			e.write("work/proposal/body.md", content)
			e.write("guides/README.md", "changed\n")
			headBefore := e.git("rev-parse", "HEAD")

			_, err := Publish(e.opts("2026-11-05"), e.bodyFile())
			mustRefuse(t, err, "suggestion")
			if e.git("rev-parse", "HEAD") != headBefore || len(e.calls()) != 0 {
				t.Error("something was committed or asked of gh")
			}
		})
	}

	e := newEnv(t)
	e.onBranch("rules-proposal/2026-11-05")
	_, err := Publish(e.opts("2026-11-05"), filepath.Join(e.repo, "work", "nope.md"))
	mustRefuse(t, err, "nope.md")
	_, err = Publish(e.opts("2026-11-05"), "")
	mustRefuse(t, err, "body")
}

func TestProposalPublishNeedsAToken(t *testing.T) {
	e := newEnv(t)
	e.onBranch("rules-proposal/2026-11-05")
	e.write("guides/README.md", "changed\n")
	t.Setenv("GH_TOKEN", "")
	headBefore := e.git("rev-parse", "HEAD")
	_, err := Publish(e.opts("2026-11-05"), e.bodyFile())
	mustRefuse(t, err, "GH_TOKEN", "GITHUB_TOKEN")
	if e.git("rev-parse", "HEAD") != headBefore || len(e.calls()) != 0 {
		t.Error("something was committed or asked of gh")
	}
	if !strings.Contains(e.statusLines(), " M guides/README.md") {
		t.Errorf("the change was staged:\n%s", e.statusLines())
	}
}

func TestProposalPublishNeedsAGitIdentity(t *testing.T) {
	e := newEnv(t)
	e.onBranch("rules-proposal/2026-11-05")
	e.write("guides/README.md", "changed\n")
	e.git("config", "--unset", "user.name")
	e.git("config", "--unset", "user.email")
	headBefore := e.git("rev-parse", "HEAD")

	_, err := Publish(e.opts("2026-11-05"), e.bodyFile())
	mustRefuse(t, err, "user.name", "user.email")
	if e.git("rev-parse", "HEAD") != headBefore || len(e.calls()) != 0 {
		t.Error("something was committed or asked of gh")
	}
	if !strings.Contains(e.statusLines(), " M guides/README.md") {
		t.Errorf("the change was staged:\n%s", e.statusLines())
	}

	// An identity from the environment is one too.
	t.Setenv("GIT_AUTHOR_NAME", "CI")
	t.Setenv("GIT_AUTHOR_EMAIL", "ci@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "CI")
	t.Setenv("GIT_COMMITTER_EMAIL", "ci@example.com")
	if _, err := Publish(e.opts("2026-11-05"), e.bodyFile()); err != nil {
		t.Fatal(err)
	}
	if who := e.git("log", "-1", "--format=%an <%ae>"); who != "CI <ci@example.com>" {
		t.Errorf("author = %q", who)
	}
}

func TestProposalPublishNeedsSomethingToPropose(t *testing.T) {
	e := newEnv(t)
	e.onBranch("rules-proposal/2026-11-05")
	_, err := Publish(e.opts("2026-11-05"), e.bodyFile())
	mustRefuse(t, err, "origin/main")
	if strings.Contains(e.originGit("for-each-ref", "refs/heads"), "rules-proposal") || len(e.calls()) != 0 {
		t.Error("the branch was pushed or gh was called")
	}
}

func TestProposalPublishPushesEarlierCommitsAfterAFailedPush(t *testing.T) {
	e := newEnv(t)
	const branch = "rules-proposal/2026-11-05"
	e.onBranch(branch)
	e.write("guides/README.md", "committed by an earlier run\n")
	e.git("commit", "-q", "-am", "rules: proposal 2026-11-05")
	commit := e.git("rev-parse", "HEAD")

	got, err := Publish(e.opts("2026-11-05"), e.bodyFile())
	if err != nil {
		t.Fatal(err)
	}
	if got.Commit != commit || !got.Created || e.originGit("rev-parse", "refs/heads/"+branch) != commit {
		t.Errorf("Published = %+v, origin has %s", got, e.originGit("rev-parse", "refs/heads/"+branch))
	}
	if n := e.git("rev-list", "--count", "origin/main..HEAD"); n != "1" {
		t.Errorf("%s commits on the branch, want the earlier one only", n)
	}
}

func TestProposalPublishNeverForcesThePush(t *testing.T) {
	e := newEnv(t)
	const branch = "rules-proposal/2026-11-05"
	theirs := e.pushBranch(branch, "guides/README.md", "someone else\n")
	e.onBranch(branch)
	e.write("glossary/terms.yaml", "- {kr: x, zh: y, status: approved}\n")

	_, err := Publish(e.opts("2026-11-05"), e.bodyFile())
	if err == nil || isRefusal(err) || !strings.Contains(err.Error(), "git push") {
		t.Fatalf("error = %v, want the failure of git push", err)
	}
	if got := e.originGit("rev-parse", "refs/heads/"+branch); got != theirs {
		t.Errorf("origin's branch is %s, want it untouched at %s", got, theirs)
	}
	if len(e.callsOf("pr", "create")) != 0 || len(e.callsOf("pr", "edit")) != 0 || len(e.callsOf("label", "create")) != 0 {
		t.Errorf("a PR was opened or edited after a failed push: %q", e.calls())
	}
}

func TestProposalPublishSystemErrors(t *testing.T) {
	for _, failing := range []string{"pr list", "label create", "pr create"} {
		t.Run(failing, func(t *testing.T) {
			e := newEnv(t)
			e.onBranch("rules-proposal/2026-11-05")
			e.write("guides/README.md", "changed\n")
			t.Setenv("FAKE_GH_FAIL", failing)
			_, err := Publish(e.opts("2026-11-05"), e.bodyFile())
			if err == nil || isRefusal(err) || !strings.Contains(err.Error(), "gh "+failing) || !strings.Contains(err.Error(), "HTTP 500 from the fake") {
				t.Errorf("error = %v, want the failure of gh %s", err, failing)
			}
		})
	}

	e := newEnv(t)
	e.onBranch("rules-proposal/2026-11-05")
	e.write("guides/README.md", "changed\n")
	t.Setenv("FAKE_GH_URL", "no url here")
	if _, err := Publish(e.opts("2026-11-05"), e.bodyFile()); err == nil || isRefusal(err) {
		t.Errorf("error = %v, want a system error when gh prints no URL", err)
	}
}

func TestProposalStartNeedsGit(t *testing.T) {
	e := newEnv(t)
	o := e.opts("2026-11-05")
	o.Git = filepath.Join(t.TempDir(), "no-such-git")
	_, err := Start(o)
	if err == nil || isRefusal(err) {
		t.Errorf("error = %v, want a system error", err)
	}
}

func TestProposalBranchDate(t *testing.T) {
	tests := map[string]string{
		"rules-proposal/2026-11-05":    "2026-11-05",
		"rules-proposal/2026-11-05-2":  "2026-11-05",
		"rules-proposal/2026-11-05-12": "2026-11-05",
		"rules-proposal/2026-11-05-":   "",
		"rules-proposal/2026-11-05-x":  "",
		"rules-proposal/2026-11-05-+2": "",
		"rules-proposal/2026-11-05--2": "",
		"rules-proposal/2026-11-05a":   "",
		"rules-proposal/2026-13-05":    "",
		"rules-proposal/manual":        "",
		"rules-proposal/":              "",
	}
	for branch, want := range tests {
		if got := branchDate(branch); got != want {
			t.Errorf("branchDate(%q) = %q, want %q", branch, got, want)
		}
	}
}

func TestProposalStartReusesAnOpenPRWhoseLabelWasRemoved(t *testing.T) {
	e := newEnv(t)
	const head = "rules-proposal/2026-10-29"
	sha := e.pushBranch(head, "guides/README.md", "work of the PR\n")
	// The label is gone, the branch is the rules branch. Two other PRs are
	// not rules PRs and their markers (a quoted one, say) mean nothing.
	e.setPRs([]ghPR{
		{Number: 4, URL: "u4", HeadRefName: "feature/y", Labels: []ghLabel{{Name: "bug"}}, Body: body("term/not-ours")},
		{Number: 5, URL: "u5", HeadRefName: head, Body: body("speaker/새 화자")},
		{Number: 6, URL: "u6", HeadRefName: "feature/z", Body: body("term/not-ours-either")},
	}, nil)

	got, err := Start(e.opts("2026-11-05"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Branch != head || got.PR != 5 || got.URL != "u5" {
		t.Errorf("Started = %+v", got)
	}
	assertIDs(t, "Existing", got.Existing, "speaker/새 화자")
	assertIDs(t, "exclude.txt", e.excludeIDs(), "speaker/새 화자")
	if e.branch() != head || e.git("rev-parse", "HEAD") != sha {
		t.Errorf("checked out %q", e.branch())
	}
}

func TestProposalStartReusesAnOpenPRThatOnlyTheHeadQueryFinds(t *testing.T) {
	e := newEnv(t)
	const branch = "rules-proposal/2026-11-05"
	sha := e.pushBranch(branch, "guides/README.md", "work of the PR\n")
	// The listing of the open PRs missed it (the PR is minutes old, or has no
	// label), but asking for the head finds it, open, next to an older closed one.
	prBody := body("speaker/새 화자", "term/신규 용어")
	e.setAllPRs(
		ghPR{Number: 3, URL: "u3", HeadRefName: branch, State: "CLOSED", Body: body("term/closed")},
		ghPR{Number: 7, URL: "u7", HeadRefName: branch, State: "OPEN", Body: prBody},
	)

	got, err := Start(e.opts("2026-11-05"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Branch != branch || got.PR != 7 || got.URL != "u7" {
		t.Errorf("Started = %+v, want the open PR #7 on %s and not a second PR on %s-2", got, branch, branch)
	}
	assertIDs(t, "Existing", got.Existing, "speaker/새 화자", "term/신규 용어")
	assertIDs(t, "exclude.txt", e.excludeIDs(), "speaker/새 화자", "term/신규 용어")
	if e.branch() != branch || e.git("rev-parse", "HEAD") != sha {
		t.Errorf("checked out %q at %s", e.branch(), e.git("rev-parse", "HEAD"))
	}
	if got.ExistingBody != "work/proposal/existing-body.md" || e.read(got.ExistingBody) != prBody {
		t.Errorf("ExistingBody = %q", got.ExistingBody)
	}

	// Publish then updates that PR.
	e.t.Setenv("FAKE_GH_OPEN", `[{"number":7,"url":"u7","headRefName":"`+branch+`","body":`+jsonString(prBody)+`}]`)
	e.write("work/proposal/body.md", prBody)
	e.write("guides/README.md", "more\n")
	pub, err := Publish(e.opts("2026-11-05"), e.bodyFile())
	if err != nil {
		t.Fatal(err)
	}
	if pub.Created || pub.URL != "u7" {
		t.Errorf("Published = %+v", pub)
	}
}

func jsonString(s string) string {
	data, _ := json.Marshal(s)
	return string(data)
}

func TestProposalPublishKeepsTheSuggestionsTheOpenPRHas(t *testing.T) {
	e := newEnv(t)
	const branch = "rules-proposal/2026-10-29"
	e.onBranch(branch)
	e.write("guides/README.md", "changed\n")
	current := body("speaker/새 화자", "adjustment-moved/StoryData/S1.json|/dataList/2/content", "term/신규 용어")
	e.setPRs([]ghPR{{Number: 5, URL: "u5", HeadRefName: branch, Body: current}}, nil)
	headBefore := e.git("rev-parse", "HEAD")
	originBefore := e.originGit("for-each-ref", "--format=%(refname) %(objectname)")

	// gh pr edit replaces the description: one that lacks a suggestion of the
	// PR would drop it, and the next run would suggest it again.
	e.write("work/proposal/body.md", body("term/신규 용어", "term/new-one"))
	_, err := Publish(e.opts("2026-11-05"), e.bodyFile())
	mustRefuse(t, err, "#5", "speaker/새 화자", "adjustment-moved/StoryData/S1.json|/dataList/2/content")
	if strings.Contains(err.Error(), "term/new-one") || strings.Contains(err.Error(), "term/신규 용어") {
		t.Errorf("the refusal names ids that are kept: %v", err)
	}
	if e.git("rev-parse", "HEAD") != headBefore || e.originGit("for-each-ref", "--format=%(refname) %(objectname)") != originBefore {
		t.Error("something was committed or pushed")
	}
	if len(e.callsOf("pr", "edit")) != 0 || !strings.Contains(e.statusLines(), " M guides/README.md") {
		t.Errorf("the PR was edited or the change staged:\n%s", e.statusLines())
	}

	// Every id of the PR, in any order and with CRLF, and more: updated.
	e.write("work/proposal/body.md", strings.ReplaceAll(body("term/new-one", "term/신규 용어", "adjustment-moved/StoryData/S1.json|/dataList/2/content", "speaker/새 화자"), "\n", "\r\n"))
	got, err := Publish(e.opts("2026-11-05"), e.bodyFile())
	if err != nil {
		t.Fatal(err)
	}
	if got.Created || len(e.callsOf("pr", "edit")) != 1 {
		t.Errorf("Published = %+v, calls %q", got, e.calls())
	}
}
