package site

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/rawdiff"
)

// DataChange is what a snapshot of the raw branch changed in the KR pack as
// data, from the snapshot before it (Diff.From to Diff.To), and the time of
// the commit of the snapshot (RFC 3339, UTC, to the second). Others are the
// same for the packs of the other languages, each with its Lang and how it
// differs from KR (VsKR).
type DataChange struct {
	Diff   *rawdiff.Diff
	Others []*rawdiff.Diff
	At     string
}

// DataSnapshot is one row of rawdiff/index.json: a snapshot and how much it
// differs from the one before.
type DataSnapshot struct {
	Tag     string         `json:"tag"`
	PrevTag string         `json:"prev_tag"`
	At      string         `json:"at"`
	Counts  rawdiff.Counts `json:"counts"` // of the KR pack
	// VsKR is, by the language of each other pack that has a file, the number
	// of changes that only it or only KR has. Never nil.
	VsKR map[string]int `json:"vs_kr"`
}

// DataIndex is rawdiff/index.json, the newest snapshot first. The first
// snapshot of the raw branch has nothing before it and no row.
type DataIndex struct {
	Snapshots []DataSnapshot `json:"snapshots"` // never nil
}

// dataLangRe matches the language of another pack, as it is in a file name.
var dataLangRe = regexp.MustCompile(`^[a-z]{2,8}$`)

// ExportDataChanges writes <outDir>/rawdiff/index.json and, for each of
// changes, <outDir>/rawdiff/<tag>.json: its Diff, under the tag of the newer
// snapshot, and <outDir>/rawdiff/<tag>.<lang>.json for each of its Others.
// changes are the oldest first; the index lists them the newest first. Like
// ExportUpdates it empties nothing.
func ExportDataChanges(changes []DataChange, outDir string) (DataIndex, error) {
	if outDir == "" {
		return DataIndex{}, errors.New("site: no output directory")
	}
	w := &writer{dir: outDir}
	idx := DataIndex{Snapshots: []DataSnapshot{}}
	for i := len(changes) - 1; i >= 0; i-- {
		c := changes[i]
		tag := c.Diff.To
		if tag == "" || !filepath.IsLocal(tag) || strings.ContainsAny(tag, `/\`) {
			return DataIndex{}, fmt.Errorf("site: %q is not a snapshot tag", tag)
		}
		if err := w.writeJSON("rawdiff/"+tag+".json", c.Diff); err != nil {
			return DataIndex{}, err
		}
		row := DataSnapshot{Tag: tag, PrevTag: c.Diff.From, At: c.At, Counts: c.Diff.Counts, VsKR: map[string]int{}}
		for _, other := range c.Others {
			if !dataLangRe.MatchString(other.Lang) || other.Lang == "kr" || other.VsKR == nil {
				return DataIndex{}, fmt.Errorf("site: the data change of %s has another pack that is not one (language %q)", tag, other.Lang)
			}
			if err := w.writeJSON("rawdiff/"+tag+"."+other.Lang+".json", other); err != nil {
				return DataIndex{}, err
			}
			row.VsKR[other.Lang] = len(other.VsKR.OnlyHere) + len(other.VsKR.OnlyOther)
		}
		idx.Snapshots = append(idx.Snapshots, row)
	}
	if err := w.writeJSON("rawdiff/index.json", idx); err != nil {
		return DataIndex{}, err
	}
	return idx, nil
}
