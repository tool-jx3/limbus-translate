package store

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const notesName = "notes.jsonl"

// Note is what a translator reported about a unit: one line of
// .meta/notes.jsonl. KRSha is the sha256 of the KR text the batch was cut
// with, Model the model that wrote the translation.
type Note struct {
	Key   string `json:"key"`
	Batch string `json:"batch"`
	KRSha string `json:"kr_sha"`
	Model string `json:"model"`
	Note  string `json:"note"`
}

func (s *Store) notesPath() string { return filepath.Join(s.metaDir(), notesName) }

// LoadNotes reads every note in the order they were added. A worktree
// without notes yields none.
func (s *Store) LoadNotes() ([]Note, error) {
	notes := []Note{}
	err := readLines(s.notesPath(), func(line int, text []byte) error {
		var n Note
		if err := decodeStrict(text, &n); err != nil {
			return err
		}
		notes = append(notes, n)
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("store: load notes: %w", err)
	}
	return notes, nil
}

// AppendNotes adds notes to the end of .meta/notes.jsonl, which only ever
// grows, and returns how many it added. A note with the same key, batch and
// text as one already stored (or earlier in notes) is not added again, so a
// batch can be submitted more than once. Every key must be a valid unit key;
// otherwise nothing is written.
//
// The file is rewritten atomically. Like the other writers it does not take
// the lock: the caller runs it inside WithLock.
func (s *Store) AppendNotes(notes []Note) (int, error) {
	type id struct{ key, batch, note string }
	existing, err := s.LoadNotes()
	if err != nil {
		return 0, err
	}
	seen := make(map[id]bool, len(existing))
	for _, n := range existing {
		seen[id{n.Key, n.Batch, n.Note}] = true
	}
	var add bytes.Buffer
	added := 0
	for _, n := range notes {
		if _, err := fileOfKey(n.Key); err != nil {
			return 0, fmt.Errorf("store: append notes: %w", err)
		}
		if k := (id{n.Key, n.Batch, n.Note}); !seen[k] {
			seen[k] = true
			if err := appendLine(&add, n); err != nil {
				return 0, fmt.Errorf("store: append notes: %w", err)
			}
			added++
		}
	}
	if added == 0 {
		return 0, nil
	}
	old, err := os.ReadFile(s.notesPath())
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return 0, fmt.Errorf("store: append notes: %w", err)
	}
	if len(old) > 0 && old[len(old)-1] != '\n' {
		old = append(old, '\n')
	}
	if err := writeFileAtomic(s.notesPath(), append(old, add.Bytes()...)); err != nil {
		return 0, fmt.Errorf("store: append notes: %w", err)
	}
	return added, nil
}
