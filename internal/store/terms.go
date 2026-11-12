package store

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const termsName = "terms.jsonl"

// ProvisionalTerm is a term a translator registered for a unit and that is
// kept until a person decides on it: one line of .meta/terms.jsonl. Time is
// RFC 3339 in UTC. Conflict marks a term whose KR already had a provisional
// term with another rendering.
type ProvisionalTerm struct {
	KR       string `json:"kr"`
	ZH       string `json:"zh"`
	Key      string `json:"key"`
	Batch    string `json:"batch"`
	RawTag   string `json:"raw_tag"`
	Time     string `json:"time"`
	Conflict bool   `json:"conflict,omitempty"`
	// Old, when set, records that at registration the term differed from the
	// existing name or the old translation: it is noted but has no effect.
	Old *OldUse `json:"old,omitempty"`
}

// OldUse describes how a term compares with an existing name: Existing is the
// existing rendering, Units the number of units that use it and With how many
// of those use it together with the term's KR.
type OldUse struct {
	Existing string `json:"existing,omitempty"`
	Units    int    `json:"units"`
	With     int    `json:"with"`
}

func (s *Store) termsPath() string { return filepath.Join(s.metaDir(), termsName) }

// LoadTerms reads every provisional term in the order they were added. A
// worktree without terms yields none.
func (s *Store) LoadTerms() ([]ProvisionalTerm, error) {
	terms := []ProvisionalTerm{}
	err := readLines(s.termsPath(), func(line int, text []byte) error {
		var t ProvisionalTerm
		if err := decodeStrict(text, &t); err != nil {
			return err
		}
		terms = append(terms, t)
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("store: load terms: %w", err)
	}
	return terms, nil
}

// AppendTerms adds terms to the end of .meta/terms.jsonl, which only ever
// grows, and returns how many it added. A term with the same KR and ZH as one
// already stored (or earlier in terms) is not added again. Every key must be a
// valid unit key; otherwise nothing is written.
//
// The file is rewritten atomically. Like the other writers it does not take
// the lock: the caller runs it inside WithLock.
func (s *Store) AppendTerms(terms []ProvisionalTerm) (int, error) {
	type id struct{ kr, zh string }
	existing, err := s.LoadTerms()
	if err != nil {
		return 0, err
	}
	seen := make(map[id]bool, len(existing))
	for _, t := range existing {
		seen[id{t.KR, t.ZH}] = true
	}
	var add bytes.Buffer
	added := 0
	for _, t := range terms {
		if _, err := fileOfKey(t.Key); err != nil {
			return 0, fmt.Errorf("store: append terms: %w", err)
		}
		if k := (id{t.KR, t.ZH}); !seen[k] {
			seen[k] = true
			if err := appendLine(&add, t); err != nil {
				return 0, fmt.Errorf("store: append terms: %w", err)
			}
			added++
		}
	}
	if added == 0 {
		return 0, nil
	}
	old, err := os.ReadFile(s.termsPath())
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return 0, fmt.Errorf("store: append terms: %w", err)
	}
	if len(old) > 0 && old[len(old)-1] != '\n' {
		old = append(old, '\n')
	}
	if err := writeFileAtomic(s.termsPath(), append(old, add.Bytes()...)); err != nil {
		return 0, fmt.Errorf("store: append terms: %w", err)
	}
	return added, nil
}
