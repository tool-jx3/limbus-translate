package history

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/store"
)

// dataDir is the directory of the tw branch that holds the built files.
const dataDir = "LimbusCompany_Data"

// Side is one Release (or a tw worktree not yet committed): its State and
// where its files are on disk.
type Side struct {
	Tag, RawTag, RulesCommit string
	Time                     time.Time
	State                    State
	PackDir                  string // root of the built files (.../LimbusCompany_Data)
	// Applied are the rule files this build had applied (.meta/applied.json),
	// Pending its queue (.meta/pending.jsonl). Both are empty when the tree
	// has no such file.
	Applied store.Applied
	Pending []store.PendingItem
}

// ReadRev reads rev (a Release tag) of tw: it exports LimbusCompany_Data and
// .meta of that commit to tmp/tw, and the KR pack of the raw tag named by its
// .meta/build.json to tmp/raw (see raw.OpenKRAt). tmp must be a new, empty
// directory, which the caller removes.
func ReadRev(twRoot, rawRoot, rev, langDir, tmp string) (*Side, error) {
	at, err := gitx.RevTime(twRoot, rev)
	if err != nil {
		return nil, fmt.Errorf("history: read %s: %w", rev, err)
	}
	dest := filepath.Join(tmp, "tw")
	if err := gitx.Archive(twRoot, rev, []string{dataDir, ".meta"}, dest); err != nil {
		return nil, fmt.Errorf("history: read %s: %w", rev, err)
	}
	s, err := readSide(dest, rawRoot, langDir, tmp)
	if err != nil {
		return nil, fmt.Errorf("history: read %s: %w", rev, err)
	}
	s.Tag, s.Time = rev, at
	return s, nil
}

// ReadWorktree reads the files the tw worktree at twRoot holds now, the build
// that is not committed yet. The KR pack is exported to tmp/raw as in ReadRev.
// The Side has no Tag and no Time.
func ReadWorktree(twRoot, rawRoot, langDir, tmp string) (*Side, error) {
	s, err := readSide(twRoot, rawRoot, langDir, tmp)
	if err != nil {
		return nil, fmt.Errorf("history: read the worktree: %w", err)
	}
	return s, nil
}

// readSide reads the tw tree at root (a worktree or an exported commit).
func readSide(root, rawRoot, langDir, tmp string) (*Side, error) {
	info, err := build.ReadInfo(root)
	if err != nil {
		return nil, err
	}
	recs, err := store.Open(root).LoadRecords()
	if err != nil {
		return nil, err
	}
	applied, err := store.Open(root).LoadApplied()
	if err != nil {
		return nil, err
	}
	pending, err := store.Open(root).LoadPending()
	if err != nil {
		return nil, err
	}
	packDir := filepath.Join(root, dataDir)
	zh, err := pack.Open(filepath.Join(packDir, "Lang", langDir), "")
	if err != nil {
		return nil, err
	}
	kr, err := raw.OpenKRAt(rawRoot, info.RawTag, filepath.Join(tmp, "raw"))
	if err != nil {
		return nil, err
	}
	return &Side{
		RawTag:      info.RawTag,
		RulesCommit: info.RulesCommit,
		State:       State{KR: kr, ZH: zh, Records: recs},
		PackDir:     packDir,
		Applied:     applied,
		Pending:     pending,
	}, nil
}

// Make compares prev with cur and returns the Record of the Release called
// tag, made at at. Only the strings of files whose KR text or built file
// differs between the two are compared. Files below the pack directory that
// are not such pack files, nor Info/version.json (which changes every time),
// and differ or are on one side only are listed in OtherFiles, relative to
// LimbusCompany_Data.
//
// With rules, which gives the rules of the RulesCommit of each side, every
// entry gets its Causes (see inferCauses), and the Record the rules they point
// to, the number of entries with an unknown cause and CausesVersion. Rules
// that cannot be read stop nothing: the entries that need them have an unknown
// cause, whose note says why. Without rules (nil) no cause is inferred and
// CausesVersion is 0.
func Make(prev, cur *Side, tag string, at time.Time, langDir string, rules RulesSource) (*Record, error) {
	return makeRecord(prev, cur, tag, at, langDir, rules, Compare)
}

func makeRecord(prev, cur *Side, tag string, at time.Time, langDir string, rules RulesSource,
	compare func(old, new State, files []string) ([]Entry, error)) (*Record, error) {
	rec := &Record{
		Tag:             tag,
		PrevTag:         prev.Tag,
		ReleasedAt:      formatTime(at),
		PrevReleasedAt:  formatTime(prev.Time),
		RawTag:          cur.RawTag,
		PrevRawTag:      prev.RawTag,
		RulesCommit:     cur.RulesCommit,
		PrevRulesCommit: prev.RulesCommit,
		OtherFiles:      []string{},
		Entries:         []Entry{},
	}

	krFiles := map[string]bool{}
	for _, f := range prev.State.KR.Files() {
		krFiles[f] = true
	}
	for _, f := range cur.State.KR.Files() {
		krFiles[f] = true
	}
	var files []string
	for f := range krFiles {
		differ, err := packFileDiffers(prev.State.KR, cur.State.KR, f)
		if err == nil && !differ {
			differ, err = packFileDiffers(prev.State.ZH, cur.State.ZH, f)
		}
		if err != nil {
			return nil, fmt.Errorf("history: %w", err)
		}
		if differ {
			files = append(files, f)
		}
	}
	sort.Strings(files)

	entries, err := compare(prev.State, cur.State, files)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		c := &rec.Counts
		if e.Copy {
			c = &rec.CopyCounts
		}
		switch e.Kind {
		case KindNew:
			c.New++
		case KindKRChanged:
			c.KRChanged++
		case KindDeleted:
			c.Deleted++
		case KindZHChanged:
			c.ZHChanged++
		}
	}
	if rules != nil {
		in := newInference(prev, cur, readRules(rules, prev, prev.Tag), readRules(rules, cur, tag))
		for i := range entries {
			entries[i].Causes = in.causes(entries[i])
			if slices.ContainsFunc(entries[i].Causes, func(c Cause) bool { return c.Kind == CauseKindUnknown }) {
				rec.Unknown++
			}
		}
		rec.CausesVersion = CausesVersion
		if len(in.rules) > 0 {
			rec.Rules = in.rules
		}
	}
	if entries != nil {
		rec.Entries = entries
	}

	other, err := otherFiles(prev.PackDir, cur.PackDir, langDir, krFiles)
	if err != nil {
		return nil, fmt.Errorf("history: %w", err)
	}
	if other != nil {
		rec.OtherFiles = other
	}
	return rec, nil
}

// readRules reads the rules of s, the build of the Release called name, from
// rules. A build that names no rules commit has none to read.
func readRules(rules RulesSource, s *Side, name string) sideRules {
	out := sideRules{commit: s.RulesCommit}
	if s.RulesCommit == "" {
		if name == "" {
			name = "the build"
		}
		out.err = fmt.Errorf("history: %s names no rules commit", name)
		return out
	}
	if out.rules, out.err = rules(s.RulesCommit); out.err != nil {
		out.rules = nil
	}
	return out
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Truncate(time.Second).Format(time.RFC3339)
}

// readPackFile returns the bytes of file in p, and nil, false if p lacks it.
func readPackFile(p *pack.Pack, file string) ([]byte, bool, error) {
	if !p.Has(file) {
		return nil, false, nil
	}
	data, err := os.ReadFile(p.Path(file))
	return data, err == nil, err
}

// packFileDiffers reports whether file is on one side only, or its bytes
// differ.
func packFileDiffers(a, b *pack.Pack, file string) (bool, error) {
	x, okX, err := readPackFile(a, file)
	if err != nil {
		return false, err
	}
	y, okY, err := readPackFile(b, file)
	if err != nil {
		return false, err
	}
	return okX != okY || !bytes.Equal(x, y), nil
}

// otherFiles lists the files below the two pack directories, other than the
// built copies of KR files and Info/version.json, that differ or are on one
// side only.
func otherFiles(prevDir, curDir, langDir string, krFiles map[string]bool) ([]string, error) {
	skip := map[string]bool{path.Join("Lang", langDir, "Info", "version.json"): true}
	for f := range krFiles {
		skip[path.Join("Lang", langDir, f)] = true
	}
	seen := map[string]bool{}
	for _, dir := range []string{prevDir, curDir} {
		err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(dir, p)
			if err != nil {
				return err
			}
			if rel = filepath.ToSlash(rel); !skip[rel] {
				seen[rel] = true
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	var out []string
	for rel := range seen {
		x, errX := os.ReadFile(filepath.Join(prevDir, filepath.FromSlash(rel)))
		y, errY := os.ReadFile(filepath.Join(curDir, filepath.FromSlash(rel)))
		for _, e := range []error{errX, errY} {
			if e != nil && !errors.Is(e, fs.ErrNotExist) {
				return nil, e
			}
		}
		if (errX == nil) != (errY == nil) || !bytes.Equal(x, y) {
			out = append(out, rel)
		}
	}
	slices.Sort(out)
	return out, nil
}
