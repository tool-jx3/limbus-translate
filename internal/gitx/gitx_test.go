package gitx

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
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

// upstreamWithTags is a repo whose raw branch has one commit tagged
// raw-20261029 and raw-20261029.2, and whose main branch carries the unrelated
// tag v1.
func upstreamWithTags(t *testing.T) string {
	t.Helper()
	upstream := newRepo(t)
	if err := Tag(upstream, "v1"); err != nil {
		t.Fatal(err)
	}
	rawDir := filepath.Join(t.TempDir(), "raw")
	if err := EnsureWorktree(upstream, "raw", rawDir); err != nil {
		t.Fatal(err)
	}
	write(t, rawDir, "kr/KR_A.json", "{}\n")
	if _, _, err := CommitAll(rawDir, "raw: tok"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"raw-20261029", "raw-20261029.2"} {
		if err := Tag(rawDir, name); err != nil {
			t.Fatal(err)
		}
	}
	return upstream
}

func TestEnsureWorktreeFetchesTagsOfThePrefix(t *testing.T) {
	upstream := upstreamWithTags(t)
	// --no-tags: the clone has origin/raw but none of upstream's tags.
	clone := filepath.Join(t.TempDir(), "clone")
	runGit(t, filepath.Dir(clone), "clone", "-q", "--no-tags", upstream, clone)
	configure(t, clone)
	if runGit(t, clone, "rev-parse", "--verify", "refs/remotes/origin/raw") == "" {
		t.Fatal("the clone should already have origin/raw")
	}
	if got, err := Tags(clone, ""); err != nil || len(got) != 0 {
		t.Fatalf("the --no-tags clone has tags %v (%v)", got, err)
	}

	dir := filepath.Join(clone, ".work", "raw")
	if err := EnsureWorktree(clone, "raw", dir, "raw-"); err != nil {
		t.Fatal(err)
	}
	got, err := Tags(clone, "")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"raw-20261029", "raw-20261029.2"}; !slices.Equal(got, want) {
		t.Errorf("tags after EnsureWorktree = %v, want %v (the prefix only, not v1)", got, want)
	}
}

func TestEnsureWorktreeFetchesNoTagsWithoutAPrefix(t *testing.T) {
	upstream := upstreamWithTags(t)
	clone := filepath.Join(t.TempDir(), "clone")
	runGit(t, filepath.Dir(clone), "clone", "-q", "--no-tags", upstream, clone)
	configure(t, clone)
	if err := EnsureWorktree(clone, "raw", filepath.Join(clone, ".work", "raw")); err != nil {
		t.Fatal(err)
	}
	if got, err := Tags(clone, ""); err != nil || len(got) != 0 {
		t.Errorf("tags = %v (%v), want none", got, err)
	}
}

func TestEnsureWorktreeKeepsLocalTagsOnConflict(t *testing.T) {
	// A local tag of the same name on another commit is not moved to
	// upstream's; the mismatch is reported.
	upstream := upstreamWithTags(t)
	clone := filepath.Join(t.TempDir(), "clone")
	runGit(t, filepath.Dir(clone), "clone", "-q", "--no-tags", upstream, clone)
	configure(t, clone)
	runGit(t, clone, "tag", "raw-20261029", "main")
	err := EnsureWorktree(clone, "raw", filepath.Join(clone, ".work", "raw"), "raw-")
	if err == nil {
		t.Fatal("a tag that differs from upstream's must be an error")
	}
	if got := runGit(t, clone, "rev-parse", "raw-20261029^{commit}"); got != runGit(t, clone, "rev-parse", "main") {
		t.Errorf("the local tag was moved to %s", got)
	}
}

func TestEnsureWorktreeRejectsGlobInTagPrefix(t *testing.T) {
	upstream := upstreamWithTags(t)
	clone := filepath.Join(t.TempDir(), "clone")
	runGit(t, filepath.Dir(clone), "clone", "-q", "--no-tags", upstream, clone)
	if err := EnsureWorktree(clone, "raw", filepath.Join(clone, ".work", "raw"), "raw-*"); err == nil {
		t.Error("a glob character in the tag prefix must be an error")
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

func TestCommitAllIgnoresSigningConfig(t *testing.T) {
	repo := newRepo(t)
	// Signing must fail if it is attempted, whatever the global gpg.format.
	runGit(t, repo, "config", "commit.gpgsign", "true")
	runGit(t, repo, "config", "gpg.format", "openpgp")
	runGit(t, repo, "config", "gpg.program", "false")
	write(t, repo, "new.txt", "n\n")
	if _, changed, err := CommitAll(repo, "unsigned"); err != nil || !changed {
		t.Errorf("commit.gpgsign must not make CommitAll fail: changed = %v, err = %v", changed, err)
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
	runGit(t, repo, "config", "gpg.format", "openpgp")
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

func TestCommitTime(t *testing.T) {
	repo := newRepo(t)
	cmd := exec.Command("git", "-C", repo, "commit", "-q", "--allow-empty", "-m", "dated")
	cmd.Env = append(os.Environ(), "GIT_COMMITTER_DATE=2026-10-31T12:30:45+08:00", "GIT_AUTHOR_DATE=2000-01-01T00:00:00Z")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	got, err := CommitTime(repo)
	if err != nil {
		t.Fatal(err)
	}
	// The committer date counts, not the author date, and the result is UTC.
	if want := time.Date(2026, 10, 31, 4, 30, 45, 0, time.UTC); !got.Equal(want) || got.Location() != time.UTC {
		t.Errorf("CommitTime = %v, want %v", got, want)
	}
}

func TestCommitTimeUnbornBranch(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "init", "-q", "-b", "main")
	if _, err := CommitTime(repo); err == nil {
		t.Error("CommitTime of a branch without commits succeeded")
	}
}

func TestIsClean(t *testing.T) {
	repo := newRepo(t)
	if clean, err := IsClean(repo); err != nil || !clean {
		t.Errorf("fresh repo: IsClean = %v, %v", clean, err)
	}
	write(t, repo, "untracked.txt", "u\n")
	if clean, err := IsClean(repo); err != nil || clean {
		t.Errorf("untracked file: IsClean = %v, %v; want false", clean, err)
	}
	runGit(t, repo, "add", "-A")
	if clean, err := IsClean(repo); err != nil || clean {
		t.Errorf("staged file: IsClean = %v, %v; want false", clean, err)
	}
	runGit(t, repo, "commit", "-q", "-m", "more")
	write(t, repo, "a.txt", "changed\n")
	if clean, err := IsClean(repo); err != nil || clean {
		t.Errorf("modified file: IsClean = %v, %v; want false", clean, err)
	}
}

func TestTagsAt(t *testing.T) {
	repo := newRepo(t)
	if err := Tag(repo, "raw-20261022"); err != nil { // an older commit
		t.Fatal(err)
	}
	write(t, repo, "new.txt", "n\n")
	if _, _, err := CommitAll(repo, "second"); err != nil {
		t.Fatal(err)
	}
	if got, err := TagsAt(repo, "raw-"); err != nil || len(got) != 0 {
		t.Errorf("HEAD without tags: TagsAt = %v, %v", got, err)
	}
	for _, name := range []string{"raw-20261029", "v2"} {
		if err := Tag(repo, name); err != nil {
			t.Fatal(err)
		}
	}
	got, err := TagsAt(repo, "raw-")
	if err != nil || !slices.Equal(got, []string{"raw-20261029"}) {
		t.Errorf("TagsAt = %v, %v; want only raw-20261029", got, err)
	}
}

func TestDescribe(t *testing.T) {
	repo := newRepo(t)
	write(t, repo, "guides/base.md", "base\n")
	write(t, repo, "prompts/p.md", "p\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "rules")
	head := runGit(t, repo, "rev-parse", "HEAD")
	paths := []string{"guides", "glossary", "rules", "prompts"}

	describe := func() string {
		t.Helper()
		got, err := Describe(repo, paths...)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	if got := describe(); got != head {
		t.Errorf("clean: Describe = %q, want %q", got, head)
	}

	// Changes elsewhere do not count.
	write(t, repo, "a.txt", "changed\n")
	write(t, repo, "notes.txt", "untracked\n")
	if got := describe(); got != head {
		t.Errorf("changes outside the paths: Describe = %q, want %q", got, head)
	}

	for _, tc := range []struct {
		name string
		edit func()
		undo func()
	}{
		{"unstaged change", func() { write(t, repo, "guides/base.md", "edited\n") }, func() { runGit(t, repo, "checkout", "--", "guides/base.md") }},
		{"staged change", func() { write(t, repo, "prompts/p.md", "edited\n"); runGit(t, repo, "add", "prompts/p.md") }, func() {
			runGit(t, repo, "reset", "-q", "--", "prompts/p.md")
			runGit(t, repo, "checkout", "--", "prompts/p.md")
		}},
		{"untracked file", func() { write(t, repo, "rules/new.yaml", "x\n") }, func() { os.Remove(filepath.Join(repo, "rules", "new.yaml")) }},
		{"deleted file", func() { os.Remove(filepath.Join(repo, "guides", "base.md")) }, func() { runGit(t, repo, "checkout", "--", "guides/base.md") }},
	} {
		tc.edit()
		if got := describe(); got != head+"+dirty" {
			t.Errorf("%s: Describe = %q, want %q", tc.name, got, head+"+dirty")
		}
		tc.undo()
		if got := describe(); got != head {
			t.Errorf("after undoing the %s: Describe = %q, want %q", tc.name, got, head)
		}
	}
}

func TestDescribeUnbornBranch(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "init", "-q", "-b", "main")
	write(t, repo, "guides/base.md", "base\n")
	if got, err := Describe(repo, "guides"); err != nil || got != "" {
		t.Errorf("Describe = %q, %v, want \"\" on a branch with no commits", got, err)
	}
	if _, err := Describe(filepath.Join(t.TempDir(), "nowhere"), "guides"); err == nil {
		t.Error("Describe outside a repo = nil error")
	}
}
