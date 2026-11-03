// Package gitx wraps the few git operations lct needs to keep the raw branch:
// worktrees, commits and tags. It runs the git command line, so git must be
// installed, and every error carries what git printed on stderr.
package gitx

import (
	"bytes"
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
)

// Error is a failed git command.
type Error struct {
	Args   []string // the arguments after "git -C <dir>"
	Stderr string   // what git printed on stderr, trimmed
	Err    error    // usually an *exec.ExitError
}

func (e *Error) Error() string {
	detail := e.Stderr
	if detail == "" {
		detail = e.Err.Error()
	}
	return fmt.Sprintf("git %s: %s", strings.Join(e.Args, " "), detail)
}

func (e *Error) Unwrap() error { return e.Err }

// git runs "git -C dir args..." and returns its stdout.
func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	// A missing credential must fail, not wait for someone to type it.
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if err := cmd.Run(); err != nil {
		return "", &Error{Args: args, Stderr: strings.TrimSpace(stderr.String()), Err: err}
	}
	return stdout.String(), nil
}

// exitCode returns the exit status of a failed git command, or -1 if err is
// not one.
func exitCode(err error) int {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return -1
}

// refExists reports whether ref (such as "refs/heads/raw") exists in repo.
func refExists(repo, ref string) (bool, error) {
	_, err := git(repo, "show-ref", "--verify", "--quiet", ref)
	switch {
	case err == nil:
		return true, nil
	case exitCode(err) == 1:
		return false, nil
	}
	return false, err
}

type worktree struct {
	path   string
	branch string // full ref name, empty for a detached HEAD
}

func worktrees(repo string) ([]worktree, error) {
	out, err := git(repo, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	var list []worktree
	for _, line := range strings.Split(out, "\n") {
		switch key, value, _ := strings.Cut(line, " "); key {
		case "worktree":
			list = append(list, worktree{path: value})
		case "branch":
			if n := len(list); n > 0 {
				list[n-1].branch = value
			}
		}
	}
	return list, nil
}

// realPath resolves symbolic links when it can, so that the paths git prints
// and the paths callers pass compare equal.
func realPath(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return filepath.Clean(p)
}

// EnsureWorktree makes dir a worktree of repo on branch, creating the branch if
// needed. Where the branch comes from, in order:
//   - dir is already a worktree on branch: nothing is done;
//   - repo has a local branch of that name: it is checked out;
//   - origin has it: it is fetched and a local branch tracking it is checked out;
//   - otherwise an orphan branch with no files is created.
//
// When the branch comes from origin, the tags of origin that start with any of
// tagPrefixes are fetched too (a clone made with --no-tags, or by a CI
// checkout, has none), so that callers who number tags by what exists see
// them. Prefixes are literal: they may not hold glob characters. A local tag
// that differs from origin's is an error, and is not moved.
//
// A dir that is non-empty and not a worktree, or a worktree on another branch,
// is an error. An unreachable origin is an error too, because making a new
// orphan branch then would fork the history origin already has.
func EnsureWorktree(repo, branch, dir string, tagPrefixes ...string) error {
	for _, prefix := range tagPrefixes {
		if strings.ContainsAny(prefix, "*?[\\") {
			return fmt.Errorf("gitx: tag prefix %q must not hold glob characters", prefix)
		}
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
		// A deleted worktree directory stays registered; forget it so dir
		// can be used again.
		if _, err := git(repo, "worktree", "prune"); err != nil {
			return err
		}
	}

	wts, err := worktrees(repo)
	if err != nil {
		return err
	}
	for _, wt := range wts {
		if realPath(wt.path) != realPath(dir) {
			continue
		}
		if wt.branch == "refs/heads/"+branch {
			return nil
		}
		on := "a detached HEAD"
		if wt.branch != "" {
			on = strings.TrimPrefix(wt.branch, "refs/heads/")
		}
		return fmt.Errorf("gitx: %s is a worktree on %s, not on %s", dir, on, branch)
	}
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return fmt.Errorf("gitx: %s exists and is not a worktree of %s", dir, repo)
	}

	local, err := refExists(repo, "refs/heads/"+branch)
	if err != nil {
		return err
	}
	if local {
		_, err := git(repo, "worktree", "add", "--quiet", dir, branch)
		return err
	}
	onOrigin, err := fetchFromOrigin(repo, branch, tagPrefixes)
	if err != nil {
		return err
	}
	if onOrigin {
		_, err := git(repo, "worktree", "add", "--quiet", "--track", "-b", branch, dir, "origin/"+branch)
		return err
	}
	return addOrphanWorktree(repo, branch, dir)
}

// fetchFromOrigin brings origin's branch into refs/remotes/origin/<branch>,
// along with origin's tags that start with any of tagPrefixes, and reports
// whether origin has the branch. A repo without an origin remote has none. If
// origin cannot be asked but the branch was fetched earlier, that copy is used
// and no tags are fetched.
func fetchFromOrigin(repo, branch string, tagPrefixes []string) (bool, error) {
	remotes, err := git(repo, "remote")
	if err != nil {
		return false, err
	}
	if !slices.Contains(strings.Fields(remotes), "origin") {
		return false, nil
	}
	remoteRef := "refs/remotes/origin/" + branch
	head := "refs/heads/" + branch
	out, err := git(repo, "ls-remote", "--heads", "origin", head)
	if err != nil {
		if fetched, refErr := refExists(repo, remoteRef); refErr == nil && fetched {
			return true, nil
		}
		return false, err
	}
	found := false
	for _, line := range strings.Split(out, "\n") {
		if fields := strings.Fields(line); len(fields) == 2 && fields[1] == head {
			found = true
		}
	}
	if !found {
		return false, nil
	}
	// Tag refspecs have no "+": a tag that already exists locally with other
	// contents makes the fetch fail instead of being moved.
	refspecs := []string{"+" + head + ":" + remoteRef}
	for _, prefix := range tagPrefixes {
		refspecs = append(refspecs, "refs/tags/"+prefix+"*:refs/tags/"+prefix+"*")
	}
	if _, err := git(repo, append([]string{"fetch", "--quiet", "origin"}, refspecs...)...); err != nil {
		return false, err
	}
	return true, nil
}

// addOrphanWorktree adds a worktree at dir on a new branch with no commits and
// no files.
func addOrphanWorktree(repo, branch, dir string) error {
	if _, err := git(repo, "worktree", "add", "--quiet", "--detach", dir); err != nil {
		return err
	}
	err := emptyOrphan(branch, dir)
	if err != nil {
		git(repo, "worktree", "remove", "--force", dir)
	}
	return err
}

func emptyOrphan(branch, dir string) error {
	if _, err := git(dir, "checkout", "--quiet", "--orphan", branch); err != nil {
		return err
	}
	// --ignore-unmatch: a repo whose HEAD tree is empty has nothing to remove.
	_, err := git(dir, "rm", "-rf", "--quiet", "--ignore-unmatch", ".")
	return err
}

// CommitAll stages everything in the worktree at dir (git add -A) and commits
// it with msg, unsigned whatever commit.gpgSign says (like Tag, it must run
// unattended). It returns the new HEAD. If nothing changed no commit is made,
// changed is false and sha is the current HEAD, or "" on a branch with no
// commits.
func CommitAll(dir, msg string) (sha string, changed bool, err error) {
	if _, err := git(dir, "add", "-A"); err != nil {
		return "", false, err
	}
	clean, err := IsClean(dir)
	if err != nil {
		return "", false, err
	}
	if clean {
		sha, err := Head(dir)
		return sha, false, err
	}
	if _, err := git(dir, "commit", "--quiet", "--no-gpg-sign", "-m", msg); err != nil {
		return "", false, err
	}
	sha, err = Head(dir)
	return sha, true, err
}

// IsClean reports whether the worktree at dir has no staged, unstaged or
// untracked changes.
func IsClean(dir string) (bool, error) {
	status, err := git(dir, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(status) == "", nil
}

// Tracked lists the files of the repo at dir that git tracks (that are in its
// index, committed or staged) and that are path or lie under it, as paths
// relative to dir, sorted; never nil. path is relative to dir, a file or a
// directory, and is taken literally: "*" is a file name, not a pattern. Files
// that are untracked or ignored are not listed.
func Tracked(dir, path string) ([]string, error) {
	out, err := git(dir, "--literal-pathspecs", "ls-files", "-z", "--", path)
	if err != nil {
		return nil, err
	}
	names := []string{}
	for _, name := range strings.Split(out, "\x00") {
		if name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}

// Head returns the commit HEAD points to, or "" if the branch has no commits.
func Head(dir string) (string, error) {
	out, err := git(dir, "rev-parse", "--verify", "--quiet", "HEAD")
	switch {
	case err == nil:
		return strings.TrimSpace(out), nil
	case exitCode(err) == 1:
		return "", nil
	}
	return "", err
}

// DirtySuffix ends the string Describe returns when paths have changes that are
// not committed.
const DirtySuffix = "+dirty"

// Describe returns the commit HEAD points to, followed by "+dirty" when any of
// paths (pathspecs relative to dir) has changes that are not committed:
// staged, unstaged or untracked. It returns "" if the branch has no commits.
func Describe(dir string, paths ...string) (string, error) {
	head, err := Head(dir)
	if err != nil || head == "" {
		return "", err
	}
	status, err := git(dir, append([]string{"status", "--porcelain", "--"}, paths...)...)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(status) != "" {
		head += DirtySuffix
	}
	return head, nil
}

// CommitTime returns the committer time of HEAD in UTC, to whole seconds. It
// is an error if the branch has no commits.
func CommitTime(dir string) (time.Time, error) {
	out, err := git(dir, "log", "-1", "--format=%ct")
	if err != nil {
		return time.Time{}, err
	}
	sec, err := strconv.ParseInt(strings.TrimSpace(out), 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("gitx: commit time of %s: %w", dir, err)
	}
	return time.Unix(sec, 0).UTC(), nil
}

// Tag creates a lightweight tag called name on HEAD. It is an error if the tag
// exists. The tag.gpgSign setting is ignored: a signed tag needs a message and
// an editor, and these tags only mark snapshots.
func Tag(dir, name string) error {
	_, err := git(dir, "tag", "--no-sign", name)
	return err
}

// Tags lists the tags whose names start with prefix, sorted by name.
func Tags(dir, prefix string) ([]string, error) {
	out, err := git(dir, "tag", "--list")
	if err != nil {
		return nil, err
	}
	return withPrefix(out, prefix), nil
}

// TagsAt lists the tags on HEAD whose names start with prefix, sorted by name.
func TagsAt(dir, prefix string) ([]string, error) {
	out, err := git(dir, "tag", "--list", "--points-at", "HEAD")
	if err != nil {
		return nil, err
	}
	return withPrefix(out, prefix), nil
}

// withPrefix splits git's one-name-per-line output and keeps the names that
// start with prefix.
func withPrefix(out, prefix string) []string {
	names := []string{}
	for _, name := range strings.Split(out, "\n") {
		if name != "" && strings.HasPrefix(name, prefix) {
			names = append(names, name)
		}
	}
	return names
}
