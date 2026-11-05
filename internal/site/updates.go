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
	Kind       string         `json:"kind"` // "game" (another raw tag than the one before), "rebuild" or "first"; a Release without a record is a "rebuild" with no counts
	Counts     history.Counts `json:"counts"`
	CopyCounts history.Counts `json:"copy_counts"`
	OtherFiles int            `json:"other_files"`
	Files      int            `json:"files"`    // the number of files with a change
	Recorded   bool           `json:"recorded"` // false for the first Release and one whose record is not made yet: no page
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

// Release is a Release tag with the time of its commit (RFC 3339, UTC, to the
// second): the list of Releases and their dates are those of the tags.
type Release struct {
	Tag string
	At  string
}

// ExportUpdates writes <outDir>/updates/index.json and, for each Release that
// has a record among records, <outDir>/updates/<tag>.json, the Record with its
// entries stably re-ordered by the order of OrderFiles (the order inside a file
// is kept). releases are the Release tags, the oldest first, and decide what is
// listed: a record whose tag is not among them is left out. The index has one
// row for each Release, the newest first: the oldest is Kind "first" with no
// counts, one with a record has its kind, counts and prev_tag, and a later one
// without a record has no counts and Recorded false (it gets its record with
// the next build of the pack). Unlike Export it empties nothing.
func ExportUpdates(releases []Release, records []history.Record, r *rules.Rules, outDir string) (UpdateIndex, error) {
	if outDir == "" {
		return UpdateIndex{}, errors.New("site: no output directory")
	}
	w := &writer{dir: outDir}
	idx := UpdateIndex{Releases: []UpdateRelease{}}
	byTag := make(map[string]history.Record, len(records))
	for _, rec := range records {
		byTag[rec.Tag] = rec
	}
	for i := len(releases) - 1; i >= 0; i-- {
		rel := releases[i]
		if rel.Tag == "" || !filepath.IsLocal(rel.Tag) || strings.ContainsAny(rel.Tag, `/\`) {
			return UpdateIndex{}, fmt.Errorf("site: %q is not a Release tag", rel.Tag)
		}
		rec, has := byTag[rel.Tag]
		row := UpdateRelease{Tag: rel.Tag, ReleasedAt: rel.At}
		switch {
		case i == 0:
			row.Kind = UpdateFirst
			// The record of the Release after it knows the snapshot it was made from.
			if next, ok := byTag[nextTag(releases, i)]; ok {
				row.RawTag = next.PrevRawTag
			}
		case !has:
			row.Kind = UpdateRebuild
		default:
			rec.Entries = orderEntries(rec.Entries, r)
			if rec.OtherFiles == nil {
				rec.OtherFiles = []string{}
			}
			files := map[string]bool{}
			for _, e := range rec.Entries {
				files[e.File] = true
			}
			row.Kind = UpdateRebuild
			if rec.RawTag != rec.PrevRawTag {
				row.Kind = UpdateGame
			}
			row.Recorded = true
			row.PrevTag, row.RawTag, row.PrevRawTag = rec.PrevTag, rec.RawTag, rec.PrevRawTag
			row.Counts, row.CopyCounts, row.OtherFiles, row.Files = rec.Counts, rec.CopyCounts, len(rec.OtherFiles), len(files)
			if err := w.writeJSON("updates/"+rel.Tag+".json", rec); err != nil {
				return UpdateIndex{}, err
			}
		}
		idx.Releases = append(idx.Releases, row)
	}
	if err := w.writeJSON("updates/index.json", idx); err != nil {
		return UpdateIndex{}, err
	}
	return idx, nil
}

// nextTag returns the tag after releases[i], "" if there is none.
func nextTag(releases []Release, i int) string {
	if i+1 < len(releases) {
		return releases[i+1].Tag
	}
	return ""
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
