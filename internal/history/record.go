package history

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/store"
)

// Counts is how many entries of each kind a Record has.
type Counts struct {
	New       int `json:"new"`
	KRChanged int `json:"kr_changed"`
	Deleted   int `json:"deleted"`
	ZHChanged int `json:"zh_changed"`
}

// Record is .meta/history/<Tag>.json: what changed from the Release before
// this one (PrevTag) to this one (Tag).
type Record struct {
	Tag             string   `json:"tag"`
	PrevTag         string   `json:"prev_tag"`
	ReleasedAt      string   `json:"released_at"` // RFC 3339, UTC, to the second
	PrevReleasedAt  string   `json:"prev_released_at"`
	RawTag          string   `json:"raw_tag"`
	PrevRawTag      string   `json:"prev_raw_tag"`
	RulesCommit     string   `json:"rules_commit"`
	PrevRulesCommit string   `json:"prev_rules_commit"`
	Counts          Counts   `json:"counts"`      // without the strings that need no translation
	CopyCounts      Counts   `json:"copy_counts"` // only the strings that need no translation
	OtherFiles      []string `json:"other_files"` // never nil
	Entries         []Entry  `json:"entries"`     // never nil
}

// Empty reports whether the Record has no entry and no other file.
func (r *Record) Empty() bool {
	return len(r.Entries) == 0 && len(r.OtherFiles) == 0
}

const historyDir = "history"

func recordName(tag string) string { return historyDir + "/" + tag + ".json" }

// Write stores rec as .meta/history/<Tag>.json of the tw worktree at twRoot,
// replacing the one that is there. Nil lists are written as empty ones.
func Write(twRoot string, rec *Record) error {
	out := *rec
	if out.OtherFiles == nil {
		out.OtherFiles = []string{}
	}
	if out.Entries == nil {
		out.Entries = []Entry{}
	}
	return store.Open(twRoot).WriteMetaJSON(recordName(out.Tag), &out)
}

// Remove deletes the Record of tag from the worktree at twRoot; one that is not
// there is no error.
func Remove(twRoot, tag string) error {
	return store.Open(twRoot).RemoveMetaJSON(recordName(tag))
}

// Has reports whether the worktree at twRoot has the Record of tag.
func Has(twRoot, tag string) (bool, error) {
	var rec Record
	return store.Open(twRoot).ReadMetaJSON(recordName(tag), &rec)
}

// List reads every Record of the worktree at twRoot, the newest ReleasedAt
// first and, at the same time, the greater Tag first. A worktree without
// history yields an empty list.
func List(twRoot string) ([]Record, error) {
	entries, err := os.ReadDir(filepath.Join(twRoot, ".meta", historyDir))
	if errors.Is(err, fs.ErrNotExist) {
		return []Record{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("history: list: %w", err)
	}
	out := []Record{}
	for _, e := range entries {
		tag, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok || e.IsDir() {
			continue
		}
		var rec Record
		if _, err := store.Open(twRoot).ReadMetaJSON(recordName(tag), &rec); err != nil {
			return nil, fmt.Errorf("history: list: %w", err)
		}
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ReleasedAt != out[j].ReleasedAt {
			return out[i].ReleasedAt > out[j].ReleasedAt
		}
		return out[i].Tag > out[j].Tag
	})
	return out, nil
}
