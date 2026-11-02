package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// proposalFakeGH records its arguments in gh.log next to it, one a line and a
// "---" line after each call, and answers pr list from FAKE_GH_OPEN, pr create
// with a URL; a call named in FAKE_GH_FAIL fails.
const proposalFakeGH = `#!/bin/sh
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
	*) printf '%s' "${FAKE_GH_ALL:-[]}" ;;
	esac
	;;
"pr create") echo "https://github.com/o/r/pull/7" ;;
esac
`

// proposalRepo is a repo root on main, in the working directory, with a bare
// origin that has main, and a fake gh first in the PATH. It returns the root
// and the directory of the fake gh.
func proposalRepo(t *testing.T) (root, ghDir string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL", "EMAIL", "FAKE_GH_FAIL", "FAKE_GH_OPEN", "FAKE_GH_CLOSED", "FAKE_GH_ALL", "GITHUB_TOKEN"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
	t.Setenv("GH_TOKEN", "test-token")

	root = snapshotRepo(t)
	writeTestFile(t, root, ".gitignore", "/.work/\n/work/\n")
	writeTestFile(t, root, "guides/README.md", "guides\n")
	writeTestFile(t, root, "glossary/terms.yaml", "[]\n")
	writeTestFile(t, root, "rules/declined.yaml", "[]\n")
	writeTestFile(t, root, "internal/x.go", "package x\n")
	runGitIn(t, root, "add", "-A")
	runGitIn(t, root, "commit", "-q", "-m", "rules")
	origin := filepath.Join(t.TempDir(), "origin.git")
	runGitIn(t, root, "init", "-q", "--bare", "-b", "main", origin)
	runGitIn(t, root, "remote", "add", "origin", origin)
	runGitIn(t, root, "push", "-q", "-u", "origin", "main")

	ghDir = t.TempDir()
	writeTestFile(t, ghDir, "gh", proposalFakeGH)
	if err := os.Chmod(filepath.Join(ghDir, "gh"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", ghDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Chdir(root)
	return root, ghDir
}

// proposalGHCalls returns the gh calls so far, each as its arguments.
func proposalGHCalls(t *testing.T, ghDir string) [][]string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(ghDir, "gh.log"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
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

func jsonKeys(t *testing.T, raw string) []string {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, raw)
	}
	var keys []string
	for k := range fields {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func TestProposalStartCommand(t *testing.T) {
	root, _ := proposalRepo(t)
	t.Setenv("FAKE_GH_CLOSED", `[{"number":4,"labels":[{"name":"rules-proposal"}],"body":"<!-- suggestion:term/신규 용어 -->","mergedAt":null},{"number":3,"labels":[{"name":"rules-proposal"}],"body":"<!-- suggestion:inconsistent/ally/name/fd93b875 -->","mergedAt":"2026-11-03T00:00:00Z"}]`)

	code, stdout, stderr := lct(t, "--json", "proposal", "start", "--date", "2026-11-05")
	if code != 0 || stderr != "" {
		t.Fatalf("exit code %d, stderr:\n%s", code, stderr)
	}
	// The workflow reads these keys with jq.
	if got, want := jsonKeys(t, stdout), []string{"branch", "declined", "existing", "existing_body", "handled", "pr", "url"}; !slices.Equal(got, want) {
		t.Errorf("output keys = %v, want %v\n%s", got, want, stdout)
	}
	var started struct {
		Branch   string   `json:"branch"`
		PR       int      `json:"pr"`
		Existing []string `json:"existing"`
		Declined []string `json:"declined"`
		Handled  []string `json:"handled"`
	}
	if err := json.Unmarshal([]byte(stdout), &started); err != nil {
		t.Fatal(err)
	}
	if started.Branch != "rules-proposal/2026-11-05" || started.PR != 0 ||
		!slices.Equal(started.Declined, []string{"term/신규 용어"}) || !slices.Equal(started.Handled, []string{"inconsistent/ally/name/fd93b875"}) ||
		started.Existing == nil || len(started.Existing) != 0 {
		t.Errorf("output = %+v\n%s", started, stdout)
	}
	if b := runGitIn(t, root, "branch", "--show-current"); b != started.Branch {
		t.Errorf("checked out %q", b)
	}
	data, err := os.ReadFile(filepath.Join(root, "work", "proposal", "exclude.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "inconsistent/ally/name/fd93b875\nterm/신규 용어\n"; string(data) != want {
		t.Errorf("exclude.txt = %q, want %q", data, want)
	}

	// Without --json it is a sentence.
	code, stdout, _ = lct(t, "proposal", "start", "--date", "2026-11-05")
	if code != 0 || !strings.Contains(stdout, "rules-proposal/2026-11-05") || strings.HasPrefix(stdout, "{") {
		t.Errorf("text output (exit %d):\n%s", code, stdout)
	}
}

func TestProposalPublishCommand(t *testing.T) {
	root, ghDir := proposalRepo(t)
	if code, _, stderr := lct(t, "--json", "proposal", "start", "--date", "2026-11-05"); code != 0 {
		t.Fatalf("start: exit code %d\n%s", code, stderr)
	}
	writeTestFile(t, root, "glossary/terms.yaml", "- {kr: 신규 용어, zh: 新術語, status: approved}\n")
	writeTestFile(t, root, "work/proposal/body.md", "規則建議\n<!-- suggestion:term/신규 용어 -->\n")

	// The body file is relative to where lct is run, which need not be the root.
	t.Chdir(filepath.Join(root, "work", "proposal"))
	var out struct {
		Branch  string `json:"branch"`
		URL     string `json:"url"`
		Commit  string `json:"commit"`
		Created bool   `json:"created"`
	}
	code, stdout, stderr := lct(t, "--json", "proposal", "publish", "--body-file", "body.md")
	if code != 0 || stderr != "" {
		t.Fatalf("exit code %d, stderr:\n%s", code, stderr)
	}
	if got, want := jsonKeys(t, stdout), []string{"branch", "commit", "created", "url"}; !slices.Equal(got, want) {
		t.Errorf("output keys = %v, want %v\n%s", got, want, stdout)
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatal(err)
	}
	if head := runGitIn(t, root, "rev-parse", "HEAD"); out.Branch != "rules-proposal/2026-11-05" || out.URL != "https://github.com/o/r/pull/7" || !out.Created || out.Commit != head {
		t.Errorf("output = %+v, HEAD %s", out, head)
	}
	if got := runGitIn(t, root, "ls-remote", "origin", "refs/heads/"+out.Branch); !strings.HasPrefix(got, out.Commit) {
		t.Errorf("origin: %q, want the branch at %s", got, out.Commit)
	}
	if files := runGitIn(t, root, "show", "--name-only", "--format=%s", "HEAD"); files != "rules: proposal 2026-11-05\n\nglossary/terms.yaml" {
		t.Errorf("commit:\n%s", files)
	}
	var create []string
	for _, c := range proposalGHCalls(t, ghDir) {
		if len(c) > 1 && c[0] == "pr" && c[1] == "create" {
			create = c
		}
	}
	if want := filepath.Join(root, "work", "proposal", "body.md"); !slices.Contains(create, want) || !slices.Contains(create, "規則建議：2026-11-05") {
		t.Errorf("pr create got %q, want the body file as %s and the title", create, want)
	}

	// Text output names the PR.
	writeTestFile(t, root, "glossary/terms.yaml", "- {kr: 신규 용어, zh: 新術語二, status: approved}\n")
	code, stdout, _ = lct(t, "proposal", "publish", "--body-file", "body.md")
	if code != 0 || !strings.Contains(stdout, out.Branch) || strings.HasPrefix(stdout, "{") {
		t.Errorf("text output (exit %d):\n%s", code, stdout)
	}
}

func TestProposalExitCodes(t *testing.T) {
	root, _ := proposalRepo(t)
	writeTestFile(t, root, "work/proposal/body.md", "<!-- suggestion:term/a -->\n")

	tests := []struct {
		name     string
		args     []string
		token    string
		fail     string
		wantCode int
		wantErr  string
	}{
		{"start without a date", []string{"proposal", "start"}, "t", "", 1, "--date is required"},
		{"start with a bad date", []string{"proposal", "start", "--date", "tomorrow"}, "t", "", 1, "tomorrow"},
		{"start with an argument", []string{"proposal", "start", "--date", "2026-11-05", "x"}, "t", "", 1, "unexpected argument x"},
		{"start with an unknown flag", []string{"proposal", "start", "--date", "2026-11-05", "--force"}, "t", "", 1, "force"},
		{"start without a token", []string{"proposal", "start", "--date", "2026-11-05"}, "", "", 1, "GH_TOKEN"},
		{"start when gh fails", []string{"proposal", "start", "--date", "2026-11-05"}, "t", "pr list", 2, "HTTP 500"},
		{"publish without a body file", []string{"proposal", "publish"}, "t", "", 1, "--body-file is required"},
		{"publish with a missing body file", []string{"proposal", "publish", "--body-file", "work/none.md"}, "t", "", 1, "none.md"},
		{"publish on main", []string{"proposal", "publish", "--body-file", "work/proposal/body.md"}, "t", "", 1, "rules-proposal/"},
		{"publish with a bad date", []string{"proposal", "publish", "--body-file", "work/proposal/body.md", "--date", "5 Nov"}, "t", "", 1, "5 Nov"},
		{"publish with an argument", []string{"proposal", "publish", "--body-file", "work/proposal/body.md", "x"}, "t", "", 1, "unexpected argument x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GH_TOKEN", tt.token)
			t.Setenv("FAKE_GH_FAIL", tt.fail)
			code, stdout, stderr := lct(t, tt.args...)
			if code != tt.wantCode || !strings.Contains(stderr, tt.wantErr) || stdout != "" {
				t.Errorf("exit code %d, want %d; stderr %q should hold %q; stdout %q", code, tt.wantCode, stderr, tt.wantErr, stdout)
			}
		})
	}
	if b := runGitIn(t, root, "branch", "--show-current"); b != "main" {
		t.Errorf("a refused command left the repo on %q", b)
	}

	// Files outside the rules are a refusal that lists them.
	runGitIn(t, root, "checkout", "-q", "-b", "rules-proposal/2026-11-05")
	writeTestFile(t, root, "internal/x.go", "package x // changed\n")
	code, _, stderr := lct(t, "proposal", "publish", "--body-file", "work/proposal/body.md")
	if code != 1 || !strings.Contains(stderr, "internal/x.go") {
		t.Errorf("exit code %d, stderr %q; want 1 and the file listed", code, stderr)
	}
}

func TestProposalPublishRefusesToDropSuggestionsOfTheOpenPR(t *testing.T) {
	root, ghDir := proposalRepo(t)
	if code, _, stderr := lct(t, "--json", "proposal", "start", "--date", "2026-11-05"); code != 0 {
		t.Fatalf("start: exit code %d\n%s", code, stderr)
	}
	writeTestFile(t, root, "glossary/terms.yaml", "- {kr: 신규 용어, zh: 新術語, status: approved}\n")
	writeTestFile(t, root, "work/proposal/body.md", "<!-- suggestion:term/a -->\n")
	t.Setenv("FAKE_GH_OPEN", `[{"number":5,"url":"https://github.com/o/r/pull/5","headRefName":"rules-proposal/2026-11-05","body":"<!-- suggestion:term/a -->\n<!-- suggestion:speaker/새 화자 -->\n"}]`)

	head := runGitIn(t, root, "rev-parse", "HEAD")
	code, stdout, stderr := lct(t, "proposal", "publish", "--body-file", "work/proposal/body.md")
	if code != 1 || stdout != "" || !strings.Contains(stderr, "speaker/새 화자") || strings.Contains(stderr, "term/a") {
		t.Errorf("exit code %d, stdout %q, stderr %q; want 1 and only the missing id named", code, stdout, stderr)
	}
	if runGitIn(t, root, "rev-parse", "HEAD") != head {
		t.Error("a commit was made")
	}
	for _, c := range proposalGHCalls(t, ghDir) {
		if len(c) > 1 && c[0] == "pr" && (c[1] == "edit" || c[1] == "create") {
			t.Errorf("gh %q was called", c)
		}
	}
}
