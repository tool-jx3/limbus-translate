package store

import (
	"errors"
	"fmt"
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
// Load and Save do not take the lock themselves; callers that change state
// wrap the whole read-modify-write in WithLock. The lock is not reentrant:
// calling WithLock from inside fn deadlocks.
func (s *Store) WithLock(fn func() error) error {
	if err := os.MkdirAll(s.metaDir(), 0o755); err != nil {
		return fmt.Errorf("store: create lock directory: %w", err)
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
