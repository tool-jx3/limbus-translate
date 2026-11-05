package site

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/history"
	"github.com/tool-jx3/limbus-translate/internal/rules"
)

// UpdateRelease is one row of updates/index.json: a Release and how much it
// changed from the one before.
type UpdateRelease struct {
	Tag        string         `json:"tag"`
	ReleasedAt string         `json:"released_at"`
	PrevTag    string         `json:"prev_tag,omitempty"`
	RawTag     string         `json:"raw_tag"`
	PrevRawTag string         `json:"prev_raw_tag,omitempty"`
	Kind       string         `json:"kind"` // "game" (another raw tag than the one before), "rebuild" or "first"
	Counts     history.Counts `json:"counts"`
	CopyCounts history.Counts `json:"copy_counts"`
	OtherFiles int            `json:"other_files"`
	Files      int            `json:"files"` // the number of files with a change
}

// UpdateIndex is updates/index.json, the newest Release first.
type UpdateIndex struct {
	Releases []UpdateRelease `json:"releases"` // never nil
}

// The values of UpdateRelease.Kind.
const (
	UpdateGame    = "game"
	UpdateRebuild = "rebuild"
	UpdateFirst   = "first"
)

// ExportUpdates writes <outDir>/updates/index.json and, for each of records
// (newest first, as history.List gives them), <outDir>/updates/<tag>.json, the
// Record with its entries stably re-ordered by the order of OrderFiles (the
// order inside a file is kept). Unlike Export it empties nothing. The earliest
// record whose PrevTag has no record of its own gets one more row of Kind
// "first", with no counts, made from its Prev fields.
func ExportUpdates(records []history.Record, r *rules.Rules, outDir string) (UpdateIndex, error) {
	if outDir == "" {
		return UpdateIndex{}, errors.New("site: no output directory")
	}
	w := &writer{dir: outDir}
	idx := UpdateIndex{Releases: []UpdateRelease{}}
	have := make(map[string]bool, len(records))
	for _, rec := range records {
		have[rec.Tag] = true
	}
	for _, rec := range records {
		if rec.Tag == "" || !filepath.IsLocal(rec.Tag) || strings.ContainsAny(rec.Tag, `/\`) {
			return UpdateIndex{}, fmt.Errorf("site: %q is not a Release tag", rec.Tag)
		}
		rec.Entries = orderEntries(rec.Entries, r)
		if rec.OtherFiles == nil {
			rec.OtherFiles = []string{}
		}
		files := map[string]bool{}
		for _, e := range rec.Entries {
			files[e.File] = true
		}
		kind := UpdateRebuild
		if rec.RawTag != rec.PrevRawTag {
			kind = UpdateGame
		}
		idx.Releases = append(idx.Releases, UpdateRelease{
			Tag: rec.Tag, ReleasedAt: rec.ReleasedAt, PrevTag: rec.PrevTag, RawTag: rec.RawTag, PrevRawTag: rec.PrevRawTag,
			Kind: kind, Counts: rec.Counts, CopyCounts: rec.CopyCounts, OtherFiles: len(rec.OtherFiles), Files: len(files),
		})
		if err := w.writeJSON("updates/"+rec.Tag+".json", rec); err != nil {
			return UpdateIndex{}, err
		}
	}
	// The earliest record is the last one.
	if n := len(records); n > 0 {
		if e := records[n-1]; e.PrevTag != "" && !have[e.PrevTag] {
			idx.Releases = append(idx.Releases, UpdateRelease{
				Tag: e.PrevTag, ReleasedAt: e.PrevReleasedAt, RawTag: e.PrevRawTag, Kind: UpdateFirst,
			})
		}
	}
	if err := w.writeJSON("updates/index.json", idx); err != nil {
		return UpdateIndex{}, err
	}
	return idx, nil
}

// orderEntries returns entries sorted by the position of their file in
// OrderFiles; the sort is stable, so the entries of a file keep their order.
func orderEntries(entries []history.Entry, r *rules.Rules) []history.Entry {
	seen := map[string]bool{}
	var paths []string
	for _, e := range entries {
		if !seen[e.File] {
			seen[e.File] = true
			paths = append(paths, e.File)
		}
	}
	rank := make(map[string]int, len(paths))
	for i, p := range OrderFiles(paths, r) {
		rank[p] = i
	}
	out := append([]history.Entry{}, entries...)
	sort.SliceStable(out, func(i, j int) bool { return rank[out[i].File] < rank[out[j].File] })
	return out
}
