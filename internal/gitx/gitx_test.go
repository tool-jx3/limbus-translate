package gitx

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// runGit runs a git command in dir and returns its trimmed output.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// configure gives a repo an identity and turns signing off, whatever the
// global git config says.
func configure(t *testing.T, repo string) {
	t.Helper()
	runGit(t, repo, "config", "user.name", "Test")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "commit.gpgsign", "false")
	runGit(t, repo, "config", "tag.gpgsign", "false")
}

func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newRepo makes a repo whose main branch holds a.txt and sub/b.txt.
func newRepo(t *testing.T) string {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "init", "-q", "-b", "main")
	configure(t, repo)
	write(t, repo, "a.txt", "a\n")
	write(t, repo, "sub/b.txt", "b\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "init")
	return repo
}

func listFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		if e.Name() != ".git" {
			names = append(names, e.Name())
		}
	}
	return names
}

func TestEnsureWorktreeOrphan(t *testing.T) {
	repo := newRepo(t)
	dir := filepath.Join(repo, ".work", "raw")
	if err := EnsureWorktree(repo, "raw", dir); err != nil {
		t.Fatal(err)
	}
	if names := listFiles(t, dir); len(names) != 0 {
		t.Errorf("the orphan worktree holds %v, want no files from main", names)
	}
	if got := runGit(t, dir, "symbolic-ref", "HEAD"); got != "refs/heads/raw" {
		t.Errorf("worktree HEAD = %s, want refs/heads/raw", got)
	}
	if got := runGit(t, dir, "status", "--porcelain"); got != "" {
		t.Errorf("worktree is not clean:\n%s", got)
	}
	// The orphan branch shares no history with main.
	write(t, dir, "x.txt", "x\n")
	sha, changed, err := CommitAll(dir, "first")
	if err != nil || !changed {
		t.Fatalf("CommitAll = %q, %v, %v", sha, changed, err)
	}
	if got := runGit(t, dir, "rev-list", "--count", "HEAD"); got != "1" {
		t.Errorf("raw has %s commits, want a root commit", got)
	}
	if out, err := exec.Command("git", "-C", repo, "merge-base", "main", "raw").CombinedOutput(); err == nil {
		t.Errorf("main and raw share history: %s", out)
	}
}

func TestEnsureWorktreeIdempotent(t *testing.T) {
	repo := newRepo(t)
	dir := filepath.Join(repo, ".work", "raw")
	if err := EnsureWorktree(repo, "raw", dir); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "x.txt", "x\n")
	// A second call (an unborn branch, then a committed one) changes nothing.
	if err := EnsureWorktree(repo, "raw", dir); err != nil {
		t.Fatalf("second call on the unborn branch: %v", err)
	}
	if _, _, err := CommitAll(dir, "first"); err != nil {
		t.Fatal(err)
	}
	if err := EnsureWorktree(repo, "raw", dir); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if got := listFiles(t, dir); !slices.Equal(got, []string{"x.txt"}) {
		t.Errorf("worktree holds %v, want just x.txt", got)
	}
	if err := EnsureWorktree(repo, "other", dir); err == nil {
		t.Error("a worktree on another branch must be an error")
	}
}

func TestEnsureWorktreeLocalBranch(t *testing.T) {
	repo := newRepo(t)
	runGit(t, repo, "branch", "raw")
	dir := filepath.Join(t.TempDir(), "wt")
	if err := EnsureWorktree(repo, "raw", dir); err != nil {
		t.Fatal(err)
	}
	// The branch is main's copy, not an orphan.
	if got := listFiles(t, dir); !slices.Equal(got, []string{"a.txt", "sub"}) {
		t.Errorf("worktree holds %v, want the files of the existing branch", got)
	}
}

func TestEnsureWorktreeFromOrigin(t *testing.T) {
	// upstream has a raw branch with one file; a clone of it has no local raw.
	upstream := newRepo(t)
	rawDir := filepath.Join(t.TempDir(), "raw")
	if err := EnsureWorktree(upstream, "raw", rawDir); err != nil {
		t.Fatal(err)
	}
	write(t, rawDir, "kr/KR_A.json", "{}\n")
	if _, _, err := CommitAll(rawDir, "raw: tok"); err != nil {
		t.Fatal(err)
	}

	clone := filepath.Join(t.TempDir(), "clone")
	runGit(t, filepath.Dir(clone), "clone", "-q", upstream, clone)
	configure(t, clone)
	dir := filepath.Join(clone, ".work", "raw")
	if err := EnsureWorktree(clone, "raw", dir); err != nil {
		t.Fatal(err)
	}
	if got := listFiles(t, dir); !slices.Equal(got, []string{"kr"}) {
		t.Errorf("worktree holds %v, want upstream's raw files", got)
	}
	if got := runGit(t, dir, "rev-parse", "--abbrev-ref", "raw@{upstream}"); got != "origin/raw" {
		t.Errorf("raw tracks %q, want origin/raw", got)
	}
}

func TestEnsureWorktreeOriginWithoutBranch(t *testing.T) {
	upstream := newRepo(t)
	clone := filepath.Join(t.TempDir(), "clone")
	runGit(t, filepath.Dir(clone), "clone", "-q", upstream, clone)
	configure(t, clone)
	dir := filepath.Join(clone, ".work", "raw")
	if err := EnsureWorktree(clone, "raw", dir); err != nil {
		t.Fatal(err)
	}
	if names := listFiles(t, dir); len(names) != 0 {
		t.Errorf("the orphan worktree holds %v", names)
	}
}

func TestEnsureWorktreeOriginUnreachable(t *testing.T) {
	repo := newRepo(t)
	runGit(t, repo, "remote", "add", "origin", filepath.Join(t.TempDir(), "does-not-exist"))
	dir := filepath.Join(repo, ".work", "raw")
	err := EnsureWorktree(repo, "raw", dir)
	if err == nil {
		t.Fatal("an unreachable origin must be an error, not a silent new orphan branch")
	}
	if _, statErr := os.Stat(dir); !os.IsNotExist(statErr) {
		t.Errorf("no worktree may be created, stat err = %v", statErr)
	}
}

func TestEnsureWorktreeRefusesForeignDir(t *testing.T) {
	repo := newRepo(t)
	dir := filepath.Join(t.TempDir(), "occupied")
	write(t, dir, "keep.txt", "mine\n")
	if err := EnsureWorktree(repo, "raw", dir); err == nil {
		t.Error("a non-empty directory that is not a worktree must be an error")
	}
	if got := listFiles(t, dir); !slices.Equal(got, []string{"keep.txt"}) {
		t.Errorf("the directory was touched: %v", got)
	}
}

func TestEnsureWorktreeAfterDirectoryDeleted(t *testing.T) {
	repo := newRepo(t)
	dir := filepath.Join(repo, ".work", "raw")
	if err := EnsureWorktree(repo, "raw", dir); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "x.txt", "x\n")
	if _, _, err := CommitAll(dir, "first"); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := EnsureWorktree(repo, "raw", dir); err != nil {
		t.Fatalf("recreate after the directory was deleted: %v", err)
	}
	if got := listFiles(t, dir); !slices.Equal(got, []string{"x.txt"}) {
		t.Errorf("worktree holds %v, want x.txt", got)
	}
}

func TestCommitAll(t *testing.T) {
	repo := newRepo(t)
	head := runGit(t, repo, "rev-parse", "HEAD")

	sha, changed, err := CommitAll(repo, "nothing")
	if err != nil || changed || sha != head {
		t.Errorf("clean tree: CommitAll = %q, %v, %v; want %q, false, nil", sha, changed, err, head)
	}

	write(t, repo, "new.txt", "n\n")
	if err := os.Remove(filepath.Join(repo, "a.txt")); err != nil {
		t.Fatal(err)
	}
	sha, changed, err = CommitAll(repo, "add and remove")
	if err != nil || !changed {
		t.Fatalf("CommitAll = %q, %v, %v", sha, changed, err)
	}
	if want := runGit(t, repo, "rev-parse", "HEAD"); sha != want || sha == head {
		t.Errorf("sha = %q, HEAD = %q, previous = %q", sha, want, head)
	}
	if got := runGit(t, repo, "log", "-1", "--format=%s"); got != "add and remove" {
		t.Errorf("commit message = %q", got)
	}
	if got := runGit(t, repo, "ls-files"); got != "new.txt\nsub/b.txt" {
		t.Errorf("tracked files = %q", got)
	}
}

func TestCommitAllUnbornBranch(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "empty")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "init", "-q", "-b", "main")
	configure(t, repo)
	sha, changed, err := CommitAll(repo, "nothing")
	if err != nil || changed || sha != "" {
		t.Errorf("empty unborn branch: CommitAll = %q, %v, %v; want \"\", false, nil", sha, changed, err)
	}
	write(t, repo, "f.txt", "f\n")
	sha, changed, err = CommitAll(repo, "first")
	if err != nil || !changed || len(sha) != 40 {
		t.Errorf("first commit: CommitAll = %q, %v, %v", sha, changed, err)
	}
}

func TestCommitAllErrorCarriesGitStderr(t *testing.T) {
	_, _, err := CommitAll(filepath.Join(t.TempDir(), "missing"), "x")
	if err == nil {
		t.Fatal("a missing directory must be an error")
	}
	if !strings.Contains(err.Error(), "git") {
		t.Errorf("error does not mention git: %v", err)
	}
}

func TestTagAndTags(t *testing.T) {
	repo := newRepo(t)
	if got, err := Tags(repo, "raw-"); err != nil || len(got) != 0 {
		t.Errorf("Tags on a repo without tags = %v, %v", got, err)
	}
	for _, name := range []string{"raw-20261029", "raw-20261029.2", "v1", "raw-20261105"} {
		if err := Tag(repo, name); err != nil {
			t.Fatal(err)
		}
	}
	got, err := Tags(repo, "raw-")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"raw-20261029", "raw-20261029.2", "raw-20261105"}; !slices.Equal(got, want) {
		t.Errorf("Tags = %v, want %v", got, want)
	}
	if err := Tag(repo, "raw-20261029"); err == nil {
		t.Error("tagging with an existing name must be an error")
	}
	if got := runGit(t, repo, "cat-file", "-t", "raw-20261105"); got != "commit" {
		t.Errorf("tag is a %s object, want a lightweight tag", got)
	}
}

func TestTagIgnoresSigningConfig(t *testing.T) {
	repo := newRepo(t)
	runGit(t, repo, "config", "tag.gpgsign", "true")
	runGit(t, repo, "config", "gpg.program", "false")
	if err := Tag(repo, "raw-20261029"); err != nil {
		t.Errorf("tag.gpgsign must not make tagging fail or need an editor: %v", err)
	}
}

func TestHead(t *testing.T) {
	repo := newRepo(t)
	got, err := Head(repo)
	if err != nil || got != runGit(t, repo, "rev-parse", "HEAD") {
		t.Errorf("Head = %q, %v", got, err)
	}
}
