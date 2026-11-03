package gitx

import (
	"archive/tar"
	"bytes"
	"maps"
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

func TestTracked(t *testing.T) {
	repo := newRepo(t)
	write(t, repo, "sub/untracked.txt", "u\n")
	write(t, repo, "staged/s.txt", "s\n")
	runGit(t, repo, "add", "staged")
	write(t, repo, ".gitignore", "/ignored/\n")
	write(t, repo, "ignored/i.txt", "i\n")
	write(t, repo, "s*/glob.txt", "g\n") // a directory called "s*"
	runGit(t, repo, "add", ".gitignore")

	for _, tt := range []struct {
		path string
		want []string
	}{
		{"sub", []string{"sub/b.txt"}}, // not the untracked file beside it
		{"sub/", []string{"sub/b.txt"}},
		{"a.txt", []string{"a.txt"}}, // a file
		{"sub/b.txt", []string{"sub/b.txt"}},
		{"staged", []string{"staged/s.txt"}}, // in the index, not committed
		{"sub/untracked.txt", []string{}},
		{"ignored", []string{}}, // only ignored files
		{"missing/dir", []string{}},
		{"s*", []string{}}, // a path, not a pattern: it matches neither sub nor staged
		{".", []string{".gitignore", "a.txt", "staged/s.txt", "sub/b.txt"}},
	} {
		got, err := Tracked(repo, tt.path)
		if err != nil {
			t.Errorf("Tracked(%q): %v", tt.path, err)
			continue
		}
		if got == nil || !slices.Equal(got, tt.want) {
			t.Errorf("Tracked(%q) = %#v, want %#v", tt.path, got, tt.want)
		}
	}
	// A name with characters that are not ASCII comes back as written.
	write(t, repo, "資料/檔案.txt", "x\n")
	runGit(t, repo, "add", "資料")
	if got, err := Tracked(repo, "資料"); err != nil || !slices.Equal(got, []string{"資料/檔案.txt"}) {
		t.Errorf("Tracked(資料) = %v, %v", got, err)
	}
	if _, err := Tracked(t.TempDir(), "x"); err == nil {
		t.Error("Tracked outside a repo: want an error")
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

// Two versions of a file whose bytes a text conversion would change.
const (
	bomCRLF1 = "\xef\xbb\xbf{\r\n  \"v\": 1\r\n}\r\n"
	bomCRLF2 = "\xef\xbb\xbf{\r\n  \"v\": 2\r\n}\r\n"
)

func TestArchiveExtractsTheTreeOfARevision(t *testing.T) {
	repo := newRepo(t)
	// Like the raw branch: no line-ending conversion, even where core.autocrlf
	// asks for it.
	runGit(t, repo, "config", "core.autocrlf", "true")
	write(t, repo, ".gitattributes", "* -text\n")
	write(t, repo, "kr/KR_A.json", bomCRLF1)
	write(t, repo, "kr/StoryData/KR_S.json", "{}\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "first")
	if err := Tag(repo, "v1"); err != nil {
		t.Fatal(err)
	}
	write(t, repo, "kr/KR_A.json", bomCRLF2)
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "second")

	dest := filepath.Join(t.TempDir(), "dest")
	if err := Archive(repo, "v1", []string{"kr"}, dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "kr", "KR_A.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != bomCRLF1 {
		t.Errorf("kr/KR_A.json = %q, want the bytes of v1 %q", got, bomCRLF1)
	}
	if got := read(t, dest, "kr/StoryData/KR_S.json"); got != "{}\n" {
		t.Errorf("kr/StoryData/KR_S.json = %q", got)
	}
	// Only the paths asked for come out: no other file, no archive header file.
	if names := listFiles(t, dest); !slices.Equal(names, []string{"kr"}) {
		t.Errorf("dest holds %v, want just kr", names)
	}
}

func TestArchiveErrors(t *testing.T) {
	repo := newRepo(t)
	if err := Tag(repo, "v1"); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		rev   string
		paths []string
		want  string // must appear in the error
	}{
		{"unknown revision", "nope", []string{"sub"}, "nope"},
		{"path not in the revision", "v1", []string{"missing"}, "missing"},
		{"revision that looks like an option", "--output=/tmp/x", []string{"sub"}, "--output=/tmp/x"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dest := filepath.Join(t.TempDir(), "dest")
			err := Archive(repo, tc.rev, tc.paths, dest)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Archive(%q, %v) error = %v, want one naming %q", tc.rev, tc.paths, err, tc.want)
			}
		})
	}
}

// tarOf builds a tar stream from entries; Linkname applies to links.
func tarOf(t *testing.T, entries ...tar.Header) *bytes.Reader {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, h := range entries {
		h := h
		if h.Typeflag == tar.TypeReg {
			h.Size = int64(len("x"))
		}
		if h.Mode == 0 && h.Typeflag != tar.TypeXGlobalHeader {
			h.Mode = 0o644
		}
		if err := tw.WriteHeader(&h); err != nil {
			t.Fatal(err)
		}
		if h.Typeflag == tar.TypeReg {
			if _, err := tw.Write([]byte("x")); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes())
}

func TestExtractTarRejectsUnsafeEntries(t *testing.T) {
	outside := t.TempDir() // somewhere a careless extractor could write
	tests := []struct {
		name  string
		entry tar.Header
	}{
		{"parent directory", tar.Header{Name: "../x", Typeflag: tar.TypeReg}},
		{"parent directory inside a path", tar.Header{Name: "kr/../../x", Typeflag: tar.TypeReg}},
		{"parent directory that stays inside", tar.Header{Name: "kr/../x", Typeflag: tar.TypeReg}},
		{"parent directory as a directory", tar.Header{Name: "../d/", Typeflag: tar.TypeDir}},
		{"absolute path", tar.Header{Name: filepath.Join(outside, "abs.txt"), Typeflag: tar.TypeReg}},
		{"symbolic link", tar.Header{Name: "link", Typeflag: tar.TypeSymlink, Linkname: outside}},
		{"hard link", tar.Header{Name: "link", Typeflag: tar.TypeLink, Linkname: "kr/ok.txt"}},
		{"device", tar.Header{Name: "dev", Typeflag: tar.TypeChar}},
		{"empty name", tar.Header{Name: "", Typeflag: tar.TypeReg}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parent := t.TempDir()
			dest := filepath.Join(parent, "dest")
			// A good entry first: an unsafe one later must still be an error.
			good := tar.Header{Name: "kr/ok.txt", Typeflag: tar.TypeReg}
			if err := extractTar(tarOf(t, good, tc.entry), dest); err == nil {
				t.Fatal("extractTar succeeded, want an error")
			}
			if entries, err := os.ReadDir(parent); err != nil || len(entries) != 1 || entries[0].Name() != "dest" {
				t.Errorf("parent holds %v (%v), want only dest", entries, err)
			}
			if entries, err := os.ReadDir(outside); err != nil || len(entries) != 0 {
				t.Errorf("outside holds %v (%v), want nothing", entries, err)
			}
			if _, err := os.Lstat(filepath.Join(dest, "link")); err == nil {
				t.Error("a link was created in dest")
			}
		})
	}
}

func TestExtractTarWritesFilesAndDirectories(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "dest")
	r := tarOf(t,
		tar.Header{Name: "kr/", Typeflag: tar.TypeDir, Mode: 0o755},
		tar.Header{Name: "kr/a.txt", Typeflag: tar.TypeReg},
		tar.Header{Name: "deep/er/b.txt", Typeflag: tar.TypeReg},                                      // no entry for its directories
		tar.Header{Typeflag: tar.TypeXGlobalHeader, PAXRecords: map[string]string{"comment": "0123"}}, // as git archive starts
	)
	if err := extractTar(r, dest); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"kr/a.txt", "deep/er/b.txt"} {
		if got := read(t, dest, rel); got != "x" {
			t.Errorf("%s = %q, want x", rel, got)
		}
	}
	if err := extractTar(tarOf(t, tar.Header{Name: "kr/a.txt", Typeflag: tar.TypeReg}), dest); err == nil {
		t.Error("extracting over a file that exists must be an error")
	}
}

func read(t *testing.T, dir, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestTreeBlobs(t *testing.T) {
	repo := newRepo(t)
	write(t, repo, "a/x.json", "{\"x\": 1}\n")
	write(t, repo, "a/b/y.json", "{\"y\": 2}\n")
	write(t, repo, "c.txt", "c\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "files")

	got, err := TreeBlobs(repo, "HEAD", "a")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"a/x.json":   runGit(t, repo, "rev-parse", "HEAD:a/x.json"),
		"a/b/y.json": runGit(t, repo, "rev-parse", "HEAD:a/b/y.json"),
	}
	if !maps.Equal(got, want) {
		t.Errorf("TreeBlobs(HEAD, a) = %v, want %v", got, want)
	}

	// The paths are those of the root of the repo, also when dir is a folder
	// of it; a path that holds nothing gives an empty map, not an error.
	if got, err := TreeBlobs(filepath.Join(repo, "a", "b"), "HEAD", "a"); err != nil || !maps.Equal(got, want) {
		t.Errorf("TreeBlobs from a subdirectory = %v, %v, want %v", got, err, want)
	}
	if got, err := TreeBlobs(repo, "HEAD", "nothing-here"); err != nil || got == nil || len(got) != 0 {
		t.Errorf("TreeBlobs of a path that is not there = %v, %v, want an empty map", got, err)
	}
}

// The tree is read from the revision, not from the files on disk, and a name
// that git would quote (such as a Chinese one) comes back as it is.
func TestTreeBlobsReadsTheRevisionAndUnquotedNames(t *testing.T) {
	repo := newRepo(t)
	write(t, repo, "d/譯文.json", "{}\n")
	write(t, repo, "d/with space.json", "{}\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "names")
	first := runGit(t, repo, "rev-parse", "HEAD")
	write(t, repo, "d/譯文.json", "{\"changed\": true}\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "changed")

	got, err := TreeBlobs(repo, first, "d")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"d/譯文.json":         runGit(t, repo, "rev-parse", first+":d/譯文.json"),
		"d/with space.json": runGit(t, repo, "rev-parse", first+":d/with space.json"),
	}
	if !maps.Equal(got, want) {
		t.Errorf("TreeBlobs(%s, d) = %v, want %v", first, got, want)
	}
	if now, err := TreeBlobs(repo, "HEAD", "d"); err != nil || now["d/譯文.json"] == got["d/譯文.json"] {
		t.Errorf("the changed file has the same blob in HEAD: %v, %v", now, err)
	}
}

func TestTreeBlobsErrors(t *testing.T) {
	repo := newRepo(t)
	for _, rev := range []string{"no-such-rev", "zh-TW-20261029.1", "-HEAD", "--all"} {
		if got, err := TreeBlobs(repo, rev, "sub"); err == nil {
			t.Errorf("TreeBlobs(%q) = %v, want an error", rev, got)
		}
	}
	// A branch with no commits has no revision either.
	empty := filepath.Join(t.TempDir(), "empty")
	if err := os.MkdirAll(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, empty, "init", "-q", "-b", "main")
	if got, err := TreeBlobs(empty, "HEAD", "."); err == nil {
		t.Errorf("TreeBlobs on an unborn branch = %v, want an error", got)
	}
}
