package publish

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/history"
)

// now is the clock of the history records; tests replace it.
var now = time.Now

// sortedReleases returns the Release tags among names, the oldest first, in the
// order latestRelease picks the last of.
func sortedReleases(names []string) []string {
	type rel struct {
		name, day string
		n         int
	}
	var rels []rel
	for _, name := range names {
		m := releaseTagRe.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[2])
		if err != nil {
			continue
		}
		rels = append(rels, rel{name, m[1], n})
	}
	slices.SortFunc(rels, func(a, b rel) int {
		if a.day != b.day {
			return strings.Compare(a.day, b.day)
		}
		return a.n - b.n
	})
	out := make([]string, len(rels))
	for i, r := range rels {
		out[i] = r.name
	}
	return out
}

// readSide is a Release read from git, with the directory it was exported to.
type readSide struct {
	side *history.Side
	dir  string
	err  error
}

func (r *readSide) remove() {
	if r != nil && r.dir != "" {
		os.RemoveAll(r.dir)
	}
}

func readRelease(twRoot, rawRoot, tag, langDir string) *readSide {
	dir, err := os.MkdirTemp("", "lct-history-")
	if err != nil {
		return &readSide{err: err}
	}
	side, err := history.ReadRev(twRoot, rawRoot, tag, langDir, dir)
	return &readSide{side: side, dir: dir, err: err}
}

// writeHistory writes, in the tw worktree, the history record of the Release
// tag that Run prepares (compared with the latest Release; nothing when there is
// none, or the pack did not change) and the records the earlier Releases lack
// (each compared with the Release before it; the first one has none). It
// returns the tags it wrote and one warning for each record it could not make:
// a failure never stops the others, nor Run.
func writeHistory(root, twRoot, rawRoot, langDir, tag string) (written, warnings []string) {
	warn := func(t string, err error) {
		warnings = append(warnings, fmt.Sprintf("%s: %v", t, err))
	}
	names, err := gitx.Tags(root, TagPrefix)
	if err != nil {
		warn(tag, err)
		return
	}
	rels := sortedReleases(names)

	// Each Release is read once and dropped when no later record needs it.
	var prev *readSide
	defer func() { prev.remove() }()
	for i := 1; i < len(rels); i++ {
		has, err := history.Has(twRoot, rels[i])
		if err != nil {
			warn(rels[i], err)
		}
		if has || err != nil {
			prev.remove()
			prev = nil
			continue
		}
		if prev == nil {
			prev = readRelease(twRoot, rawRoot, rels[i-1], langDir)
		}
		cur := readRelease(twRoot, rawRoot, rels[i], langDir)
		switch {
		case prev.err != nil:
			warn(rels[i], fmt.Errorf("the Release before it: %w", prev.err))
		case cur.err != nil:
			warn(rels[i], cur.err)
		default:
			rec, err := history.Make(prev.side, cur.side, rels[i], cur.side.Time, langDir)
			if err == nil {
				err = history.Write(twRoot, rec)
			}
			if err != nil {
				warn(rels[i], err)
			} else {
				written = append(written, rels[i])
			}
		}
		prev.remove()
		prev = cur
	}

	// The Release this run prepares, from the worktree not yet committed.
	if len(rels) == 0 {
		return
	}
	if prev == nil {
		prev = readRelease(twRoot, rawRoot, rels[len(rels)-1], langDir)
	}
	if prev.err != nil {
		warn(tag, fmt.Errorf("the latest Release: %w", prev.err))
		return
	}
	dir, err := os.MkdirTemp("", "lct-history-")
	if err != nil {
		warn(tag, err)
		return
	}
	defer os.RemoveAll(dir)
	cur, err := history.ReadWorktree(twRoot, rawRoot, langDir, dir)
	if err != nil {
		warn(tag, err)
		return
	}
	rec, err := history.Make(prev.side, cur, tag, now().UTC().Truncate(time.Second), langDir)
	if err != nil {
		warn(tag, err)
		return
	}
	file := filepath.Join(twRoot, ".meta", "history", tag+".json")
	if rec.Empty() {
		// An earlier run, whose pack differed, may have left a record of this tag.
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			warn(tag, err)
		}
		return
	}
	if err := history.Write(twRoot, rec); err != nil {
		warn(tag, err)
		return
	}
	written = append(written, tag)
	return
}
