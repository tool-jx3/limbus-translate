package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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
// (see ensureGitignore), so commits of the tw worktree never carry it.
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

// ensureGitignore creates .meta/.gitignore with the line ".lock" when the
// file is missing. An existing file is left as it is.
func (s *Store) ensureGitignore() error {
	path := filepath.Join(s.metaDir(), gitignoreName)
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("store: check %s: %w", path, err)
	}
	if err := writeFileAtomic(path, []byte(lockName+"\n")); err != nil {
		return fmt.Errorf("store: create %s: %w", path, err)
	}
	return nil
}
