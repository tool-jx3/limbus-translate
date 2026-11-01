package store

import "fmt"

// appliedName is the file in .meta that records which requeue and rewrite
// files have been applied.
const appliedName = "applied.json"

// Applied is the ids of the files in rules/requeue and rules/rewrite that
// were applied to this worktree. Each such file changes the records once:
// the plan skips the ids that are listed here. The ids keep the order they
// were applied in.
type Applied struct {
	Requeue []string `json:"requeue"`
	Rewrite []string `json:"rewrite"`
}

// LoadApplied reads .meta/applied.json. A worktree that has none has applied
// nothing yet. An unreadable or malformed file is an error: guessing "nothing"
// would apply every file again.
func (s *Store) LoadApplied() (Applied, error) {
	var a Applied
	if _, err := s.ReadMetaJSON(appliedName, &a); err != nil {
		return Applied{}, err
	}
	// An empty list reads back as nil, however it was written.
	if len(a.Requeue) == 0 {
		a.Requeue = nil
	}
	if len(a.Rewrite) == 0 {
		a.Rewrite = nil
	}
	return a, nil
}

// SaveApplied replaces .meta/applied.json. Like the other writers it does not
// take the lock; run it inside WithLock together with the changes it records.
func (s *Store) SaveApplied(a Applied) error {
	// Empty lists are written as [], not null.
	if a.Requeue == nil {
		a.Requeue = []string{}
	}
	if a.Rewrite == nil {
		a.Rewrite = []string{}
	}
	if err := s.WriteMetaJSON(appliedName, a); err != nil {
		return fmt.Errorf("store: save applied: %w", err)
	}
	return nil
}
