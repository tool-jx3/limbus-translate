package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// snapshotRepo makes an lct repo root: a git repo whose main branch holds
// lct.yaml, with the identity and settings the snapshot commit needs.
func snapshotRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTestFile(t, root, "lct.yaml", "worktrees: { raw: .work/raw, tw: .work/tw }\n")
	writeTestFile(t, root, ".gitignore", "/.work/\n")
	runGitIn(t, root, "init", "-q", "-b", "main")
	runGitIn(t, root, "config", "user.name", "Test")
	runGitIn(t, root, "config", "user.email", "test@example.com")
	runGitIn(t, root, "config", "commit.gpgsign", "false")
	runGitIn(t, root, "config", "tag.gpgsign", "false")
	runGitIn(t, root, "add", "-A")
	runGitIn(t, root, "commit", "-q", "-m", "init")
	return root
}

// extractedDir is what "lct fetch --out" leaves: kr/, en/ and jp/.
func extractedDir(t *testing.T, krText string) string {
	t.Helper()
	dir := t.TempDir()
	writeTestFile(t, dir, "kr/KR_A.json", krText)
	writeTestFile(t, dir, "kr/StoryData/KR_S.json", "{}\n")
	writeTestFile(t, dir, "en/EN_A.json", "{}\n")
	writeTestFile(t, dir, "jp/JP_A.json", "{}\n")
	return dir
}

type snapshotOut struct {
	Tag     string `json:"tag"`
	Commit  string `json:"commit"`
	Changed bool   `json:"changed"`
}

func runSnapshotJSON(t *testing.T, root string, args ...string) (snapshotOut, error) {
	t.Helper()
	ctx, out := testCtx(t, root, "", true)
	err := runSnapshot(ctx, args)
	var got snapshotOut
	if err == nil {
		if err := json.Unmarshal(out.Bytes(), &got); err != nil {
			t.Fatalf("output is not JSON: %v\n%s", err, out)
		}
	}
	return got, err
}

func TestSnapshotCreatesRawBranchAndIsIdempotent(t *testing.T) {
	root := snapshotRepo(t)
	from := extractedDir(t, "\xef\xbb\xbf{\r\n}\r\n")
	args := []string{"--from", from, "--token", testToken, "--created-at", testCreatedAt}

	first, err := runSnapshotJSON(t, root, args...)
	if err != nil {
		t.Fatal(err)
	}
	wt := filepath.Join(root, ".work", "raw")
	if !first.Changed || first.Tag != "raw-20261029" {
		t.Errorf("first run = %+v, want changed raw-20261029", first)
	}
	if head := runGitIn(t, wt, "rev-parse", "HEAD"); first.Commit != head {
		t.Errorf("commit = %q, want HEAD %q", first.Commit, head)
	}
	if got := runGitIn(t, root, "rev-parse", "raw^{commit}"); got != first.Commit {
		t.Errorf("branch raw is at %s, want %s", got, first.Commit)
	}
	if b, err := os.ReadFile(filepath.Join(wt, "kr", "KR_A.json")); err != nil || string(b) != "\xef\xbb\xbf{\r\n}\r\n" {
		t.Errorf("kr/KR_A.json = %q, %v", b, err)
	}
	if b, err := os.ReadFile(filepath.Join(wt, "meta", "source.json")); err != nil || !strings.Contains(string(b), `"token": "`+testToken+`"`) {
		t.Errorf("meta/source.json = %q, %v", b, err)
	}

	second, err := runSnapshotJSON(t, root, args...)
	if err != nil {
		t.Fatal(err)
	}
	if second.Changed || second.Tag != first.Tag || second.Commit != first.Commit {
		t.Errorf("second run = %+v, want unchanged %+v", second, first)
	}

	// A new token on the same day adds a commit and a suffixed tag.
	third, err := runSnapshotJSON(t, root, "--from", extractedDir(t, "{}\n"), "--token", "l20261029_second", "--created-at", "2026-10-29 20:00:00")
	if err != nil {
		t.Fatal(err)
	}
	if !third.Changed || third.Tag != "raw-20261029.2" || third.Commit == first.Commit {
		t.Errorf("third run = %+v", third)
	}
}

func TestSnapshotTextOutput(t *testing.T) {
	root := snapshotRepo(t)
	ctx, out := testCtx(t, root, "", false)
	err := runSnapshot(ctx, []string{"--from", extractedDir(t, "{}\n"), "--token", testToken, "--created-at", testCreatedAt})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "raw-20261029") {
		t.Errorf("output does not name the tag:\n%s", out)
	}
}

func TestSnapshotUsageErrors(t *testing.T) {
	root := snapshotRepo(t)
	from := extractedDir(t, "{}\n")
	noEN := extractedDir(t, "{}\n")
	if err := os.RemoveAll(filepath.Join(noEN, "en")); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no --from", []string{"--token", testToken, "--created-at", testCreatedAt}, "--from"},
		{"no --token", []string{"--from", from, "--created-at", testCreatedAt}, "--token"},
		{"no --created-at", []string{"--from", from, "--token", testToken}, "--created-at"},
		{"bad token", []string{"--from", from, "--token", "a b", "--created-at", testCreatedAt}, "token"},
		{"bad created-at", []string{"--from", from, "--token", testToken, "--created-at", "yesterday"}, "created_at"},
		{"missing en directory", []string{"--from", noEN, "--token", testToken, "--created-at", testCreatedAt}, "en"},
		{"missing --from directory", []string{"--from", filepath.Join(t.TempDir(), "nope"), "--token", testToken, "--created-at", testCreatedAt}, "kr"},
		{"extra argument", []string{"--from", from, "--token", testToken, "--created-at", testCreatedAt, "stray"}, "stray"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runSnapshotJSON(t, root, tc.args...)
			if !isUserError(err) {
				t.Fatalf("err = %v (%T), want a user error", err, err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %q, want it to mention %q", err, tc.want)
			}
		})
	}
}
