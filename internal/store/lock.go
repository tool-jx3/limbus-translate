package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// WithLock runs fn while holding an exclusive lock on <twRoot>/.meta/.lock.
// The lock serializes goroutines and processes alike, so a read-modify-write
// of the pending queue (handing out leases) cannot interleave with another.
// It returns fn's error, or the error from taking the lock when that fails
// (fn is not called then).
//
// LoadRecords, LoadPending and the Save methods do not take the lock
// themselves. A caller that changes state must run the whole read-modify-write
// inside one WithLock call; two separate calls leave a gap in which another
// process can change the state. The lock is not reentrant: calling WithLock
// from inside fn deadlocks.
//
// WithLock also makes sure .meta/.gitignore exists and ignores the lock file
// and leftover temporary files (see ensureGitignore), so commits of the tw
// worktree never carry them.
func (s *Store) WithLock(fn func() error) error {
	if err := os.MkdirAll(s.metaDir(), 0o755); err != nil {
		return fmt.Errorf("store: create lock directory: %w", err)
	}
	if err := s.ensureGitignore(); err != nil {
		return err
	}
	// A fresh descriptor per call: flock locks belong to the open file
	// description, so two descriptors in one process exclude each other.
	f, err := os.OpenFile(filepath.Join(s.metaDir(), lockName), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("store: open lock file: %w", err)
	}
	defer f.Close() // closing the descriptor drops the lock

	for {
		err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
		if !errors.Is(err, syscall.EINTR) {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("store: lock %s: %w", f.Name(), err)
	}

	return fn()
}

// gitignoreLines are the lines .meta/.gitignore must hold: the lock file, and
// the temporary files writeFileAtomic leaves when lct is killed mid-write.
var gitignoreLines = []string{lockName, ".*.tmp*"}

// ensureGitignore makes sure .meta/.gitignore holds every line of
// gitignoreLines. A missing file is created with them; an existing one keeps
// its lines and gets the missing ones appended.
func (s *Store) ensureGitignore() error {
	path := filepath.Join(s.metaDir(), gitignoreName)
	old, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("store: read %s: %w", path, err)
	}
	have := map[string]bool{}
	for _, line := range strings.Split(string(old), "\n") {
		have[strings.TrimSpace(line)] = true
	}
	data := old
	for _, line := range gitignoreLines {
		if have[line] {
			continue
		}
		if len(data) > 0 && data[len(data)-1] != '\n' {
			data = append(data, '\n')
		}
		data = append(data, line+"\n"...)
	}
	if err := writeFileAtomic(path, data); err != nil {
		return fmt.Errorf("store: write %s: %w", path, err)
	}
	return nil
}
