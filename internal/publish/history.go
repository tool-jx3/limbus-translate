package publish

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
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

// SortedReleases returns the Release tags among names, the oldest first: by
// day, then by N as a number. Names that are not Release tags are left out. It
// is the one place that parses and orders Release tags; latestRelease takes the
// last of it.
func SortedReleases(names []string) []string {
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
		// A number too large for an int cannot be told from another: skipped,
		// as ReleaseTag does.
		n, err := strconv.Atoi(m[2])
		if err != nil {
			continue
		}
		rels = append(rels, rel{name, m[1], n})
	}
	// The days have eight digits each, so they compare as text.
	slices.SortStableFunc(rels, func(a, b rel) int {
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

// historyPlan is what Run writes to the tw worktree, inside the lock and just
// before the commit: the records to store, and the tag whose record is stale.
type historyPlan struct {
	records []*history.Record
	remove  string // a tag whose record is to go, "" if none
}

// apply writes the plan to the worktree at twRoot and returns the tags written
// and one warning for each file it could not write or remove.
func (p historyPlan) apply(twRoot string) (written, warnings []string) {
	for _, rec := range p.records {
		if err := history.Write(twRoot, rec); err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", rec.Tag, err))
			continue
		}
		written = append(written, rec.Tag)
	}
	if p.remove != "" {
		if err := history.Remove(twRoot, p.remove); err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", p.remove, err))
		}
	}
	return written, warnings
}

// planHistory compares Releases and returns what to write, without writing:
// the records the earlier Releases lack (each compared with the Release before
// it; the first one has none), and the record of the Release tag that Run
// prepares, compared with the latest Release. That one is made exactly when the
// pack of the worktree differs from that of the latest Release (the rule of
// SamePack), even if no string differs; when it does not, a stale record of tag
// is to be removed. There is none when there is no Release. A record equal to
// the one the worktree holds, but for its time, is not written again, so that
// running again changes nothing. It returns one warning for each record it
// could not make: a failure never stops the others, nor Run.
func planHistory(root, twRoot, rawRoot, langDir, tag string) (plan historyPlan, warnings []string) {
	warn := func(t string, err error) {
		warnings = append(warnings, fmt.Sprintf("%s: %v", t, err))
	}
	names, err := gitx.Tags(root, TagPrefix)
	if err != nil {
		warn(tag, err)
		return
	}
	rels := SortedReleases(names)

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
			if rec, err := history.Make(prev.side, cur.side, rels[i], cur.side.Time, langDir, nil); err != nil {
				warn(rels[i], err)
			} else {
				plan.records = append(plan.records, rec)
			}
		}
		prev.remove()
		prev = cur
	}

	// The Release this run prepares, from the worktree not yet committed. The
	// Release it is compared with is the one the decision of Run is made on.
	last := latestRelease(names)
	if last == "" {
		return
	}
	// Until its record is made (or found unchanged), a record of tag from an
	// earlier attempt is stale and must not be committed: the backfill of a
	// later run makes it from git once the Release exists.
	plan.remove = tag
	if prev == nil || prev.side == nil || prev.side.Tag != last {
		prev.remove()
		prev = readRelease(twRoot, rawRoot, last, langDir)
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
	same, err := samePackDirs(prev.side.PackDir, cur.PackDir, langDir)
	if err != nil {
		warn(tag, err)
		return
	}
	if same {
		plan.remove = tag
		return
	}
	rec, err := history.Make(prev.side, cur, tag, now().UTC().Truncate(time.Second), langDir, nil)
	if err != nil {
		warn(tag, err)
		return
	}
	plan.remove = ""
	if unchanged(twRoot, rec) {
		return
	}
	plan.records = append(plan.records, rec)
	return
}

// unchanged reports whether the worktree holds a record of rec.Tag that is rec
// but for the time of the Release.
func unchanged(twRoot string, rec *history.Record) bool {
	data, err := os.ReadFile(filepath.Join(twRoot, ".meta", "history", rec.Tag+".json"))
	if err != nil {
		return false
	}
	var old history.Record
	if json.Unmarshal(data, &old) != nil {
		return false
	}
	old.ReleasedAt = rec.ReleasedAt
	a, errA := json.Marshal(old)
	b, errB := json.Marshal(rec)
	return errA == nil && errB == nil && bytes.Equal(a, b)
}

// samePackDirs reports whether the pack directories a and b hold the same files
// with the same bytes, but for Info/version.json of langDir: the rule of
// SamePack, for a directory not committed yet.
func samePackDirs(a, b, langDir string) (bool, error) {
	skip := path.Join("Lang", langDir, "Info", "version.json")
	files := func(dir string) (map[string]bool, error) {
		out := map[string]bool{}
		err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, err := filepath.Rel(dir, p)
			if err != nil {
				return err
			}
			if rel = filepath.ToSlash(rel); rel != skip {
				out[rel] = true
			}
			return nil
		})
		return out, err
	}
	left, err := files(a)
	if err != nil {
		return false, err
	}
	right, err := files(b)
	if err != nil {
		return false, err
	}
	if len(left) != len(right) {
		return false, nil
	}
	for rel := range left {
		if !right[rel] {
			return false, nil
		}
		x, err := os.ReadFile(filepath.Join(a, filepath.FromSlash(rel)))
		if err != nil {
			return false, err
		}
		y, err := os.ReadFile(filepath.Join(b, filepath.FromSlash(rel)))
		if err != nil {
			return false, err
		}
		if !bytes.Equal(x, y) {
			return false, nil
		}
	}
	return true, nil
}
