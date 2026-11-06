package site

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/tool-jx3/limbus-translate/internal/scope"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// FormatVersion is Meta.Format: the version of the layout and the JSON keys of
// the data. The viewer asks for a refresh when it meets another version.
const FormatVersion = 2

// searchLangs are the languages of the search corpus, in the order of
// Meta.Langs: search/<lang>.json holds one string per entry of search/keys.json.
var searchLangs = []string{"kr", "jp", "en", "zh"}

// Meta is data/meta.json: where the data comes from and how much there is.
type Meta struct {
	// Format is FormatVersion; Export sets it.
	Format int `json:"format"`
	// RawTag is the snapshot of the KR, JP and EN texts.
	RawTag string `json:"raw_tag"`
	// RulesCommit is the rules commit the pack was built under.
	RulesCommit string `json:"rules_commit"`
	// TWCommit is the commit of the tw branch the TW texts are from.
	TWCommit string `json:"tw_commit"`
	// Release is the Release tag of TWCommit; "" when it has none.
	Release string `json:"release"`
	// Langs is the languages of the data; Export sets it.
	Langs []string `json:"langs"`
	// Files and Units are the number of files and of units; Export sets them.
	Files int `json:"files"`
	Units int `json:"units"`
	// Review counts the review items nobody confirmed; Export sets it.
	Review ReviewCounts `json:"review"`
}

// ReviewCounts is the number of unconfirmed items of each page of the review.
// Todo counts the lines of the to-do sections, Scope the scope proposals
// nobody declined.
type ReviewCounts struct {
	Suggestions int `json:"suggestions"`
	Warnings    int `json:"warnings"`
	Notes       int `json:"notes"`
	Todo        int `json:"todo"`
	Scope       int `json:"scope"`
}

// IndexFamily is a family of files in data/index.json: Files are the positions
// of its files in Index.Files.
type IndexFamily struct {
	ID    string `json:"id"`
	Files []int  `json:"files"`
}

// Index is data/index.json. Files are in the order of OrderFiles, the order of
// the viewer's previous and next file, and the position of a file is its number
// in search/keys.json. Families are in the order of the files, the priority of
// the routing; a family without files is left out.
type Index struct {
	Families []IndexFamily `json:"families"`
	Files    []FileSummary `json:"files"`
}

// Result is what Export wrote: Out is the directory, Files the number of files
// of the pack, Units the units in them, SearchUnits those in the search corpus
// and Bytes the size of everything written.
type Result struct {
	Out         string `json:"out"`
	Files       int    `json:"files"`
	Units       int    `json:"units"`
	SearchUnits int    `json:"search_units"`
	Bytes       int64  `json:"bytes"`
	// Review is Meta.Review. ReviewMismatches are the groups of inconsistent
	// translations whose units do not add up to the evidence of the suggestion,
	// and NotesUnplaced the notes that sit on no unit.
	Review           ReviewCounts `json:"review"`
	ReviewMismatches []string     `json:"review_mismatches"`
	NotesUnplaced    int          `json:"notes_unplaced"`
}

// Export writes the data of the viewer to outDir: meta.json, index.json,
// files/<path> for each file of the KR pack, search/keys.json and
// search/{kr,jp,en,zh}.json, and review/{suggestions,warnings,notes,todo}.json
// and review/scope.json (Review.Scope), empty lists without in.Review. The
// corpus holds the units of every file in the order of the index, but not the
// copies (identifiers and texts that are the same in every language). meta supplies what Export cannot know: RawTag,
// RulesCommit, TWCommit and Release. The output depends on in alone, so the
// same inputs give the same bytes.
//
// outDir is emptied first: everything in it is removed, and Export does not
// check where it is, so the caller must. "." would be emptied. `lct site`
// refuses a directory outside the repo, the repo root, one that holds .git or
// tracked files, and the raw and tw worktrees.
func Export(in Inputs, meta Meta, outDir string) (Result, error) {
	if outDir == "" {
		return Result{}, errors.New("site: no output directory")
	}
	paths := OrderFiles(in.KR.Files(), in.Rules)
	// Everything that can be told without reading a file comes before the
	// directory is emptied.
	for _, p := range paths {
		if !filepath.IsLocal(filepath.FromSlash(p)) {
			return Result{}, fmt.Errorf("site: %q is not a path inside the pack", p)
		}
		if !in.ZH.Has(p) {
			return Result{}, fmt.Errorf("site: %s is not in the built pack; build the pack again", p)
		}
	}
	var rev Review
	if in.Review != nil {
		var err error
		if rev, err = BuildReview(in, paths); err != nil {
			return Result{}, err
		}
	}
	w := &writer{dir: outDir}
	if err := w.clear(); err != nil {
		return Result{}, err
	}

	idx := Index{Families: []IndexFamily{}, Files: make([]FileSummary, 0, len(paths))}
	var keys [][2]any
	corpus := make([][]string, len(searchLangs)) // by searchLangs
	total := 0
	for n, p := range paths {
		file, sum, err := BuildFile(in, p)
		if err != nil {
			return Result{}, err
		}
		for i := range file.Units {
			u := &file.Units[i]
			key := units.Key(p, u.P)
			u.RV = rev.Refs[key]
			u.NT = rev.NoteText[key]
			if rev.Queued[key] {
				u.PQ = 1
			}
		}
		if err := w.writeJSON("files/"+p, file); err != nil {
			return Result{}, err
		}
		idx.Files = append(idx.Files, sum)
		if last := len(idx.Families) - 1; last >= 0 && idx.Families[last].ID == sum.Family {
			idx.Families[last].Files = append(idx.Families[last].Files, n)
		} else {
			idx.Families = append(idx.Families, IndexFamily{ID: sum.Family, Files: []int{n}})
		}
		total += len(file.Units)
		for _, u := range file.Units {
			if u.O == store.OriginCopy {
				continue
			}
			keys = append(keys, [2]any{n, u.P})
			corpus[0] = append(corpus[0], u.KR)
			corpus[1] = append(corpus[1], deref(u.JP))
			corpus[2] = append(corpus[2], deref(u.EN))
			corpus[3] = append(corpus[3], u.ZH)
		}
	}

	if keys == nil {
		keys = [][2]any{}
	}
	if err := w.writeJSON("search/keys.json", keys); err != nil {
		return Result{}, err
	}
	for i, lang := range searchLangs {
		texts := corpus[i]
		if texts == nil {
			texts = []string{}
		}
		if err := w.writeJSON("search/"+lang+".json", texts); err != nil {
			return Result{}, err
		}
	}
	if err := w.writeJSON("index.json", idx); err != nil {
		return Result{}, err
	}
	counts := countReview(rev)
	if rev.Suggestions == nil {
		rev.Suggestions, rev.Warnings, rev.Notes, rev.Todo = []SuggestionItem{}, []WarningItem{}, []NoteItem{}, []TodoSection{}
		rev.Scope = scope.Empty()
	}
	for _, f := range []struct {
		name string
		v    any
	}{
		{"suggestions", struct {
			Items []SuggestionItem `json:"items"`
		}{rev.Suggestions}},
		{"warnings", struct {
			Items []WarningItem `json:"items"`
		}{rev.Warnings}},
		{"notes", struct {
			Items []NoteItem `json:"items"`
		}{rev.Notes}},
		{"todo", struct {
			Sections []TodoSection `json:"sections"`
		}{rev.Todo}},
		{"scope", rev.Scope},
	} {
		if err := w.writeJSON("review/"+f.name+".json", f.v); err != nil {
			return Result{}, err
		}
	}
	meta.Review = counts
	meta.Format = FormatVersion
	meta.Langs = append([]string(nil), searchLangs...)
	meta.Files, meta.Units = len(paths), total
	if err := w.writeJSON("meta.json", meta); err != nil {
		return Result{}, err
	}
	mismatches := rev.Mismatches
	if mismatches == nil {
		mismatches = []string{}
	}
	return Result{
		Out: outDir, Files: len(paths), Units: total, SearchUnits: len(keys), Bytes: w.bytes,
		Review: counts, ReviewMismatches: mismatches, NotesUnplaced: rev.NotesUnplaced,
	}, nil
}

// countReview counts the items of r that nobody confirmed.
func countReview(r Review) ReviewCounts {
	var c ReviewCounts
	for _, s := range r.Suggestions {
		if s.Declined == nil {
			c.Suggestions++
		}
	}
	for _, w := range r.Warnings {
		if w.Accepted == nil {
			c.Warnings++
		}
	}
	for _, n := range r.Notes {
		if n.Declined == nil {
			c.Notes++
		}
	}
	for _, sec := range r.Todo {
		c.Todo += len(sec.Items)
	}
	for _, p := range r.Scope.Proposals {
		if p.Declined == nil {
			c.Scope++
		}
	}
	return c
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// writer writes the files of an export under dir and counts their bytes.
type writer struct {
	dir   string
	bytes int64
}

// clear empties the directory, which is made when it is not there.
func (w *writer) clear() error {
	entries, err := os.ReadDir(w.dir)
	if errors.Is(err, fs.ErrNotExist) {
		return os.MkdirAll(w.dir, 0o755)
	}
	if err != nil {
		return fmt.Errorf("site: %w", err)
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(w.dir, e.Name())); err != nil {
			return fmt.Errorf("site: %w", err)
		}
	}
	return nil
}

// writeJSON writes v as JSON, compact and without escaping HTML characters, to
// rel, a slash-separated path under the directory. The data goes to a temporary
// file first, so that a file is either whole or not there.
func (w *writer) writeJSON(rel string, v any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("site: encode %s: %w", rel, err)
	}
	full := filepath.Join(w.dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return fmt.Errorf("site: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(full), ".tmp-*")
	if err != nil {
		return fmt.Errorf("site: %w", err)
	}
	_, werr := tmp.Write(buf.Bytes())
	cerr := tmp.Close()
	if err := errors.Join(werr, cerr, os.Chmod(tmp.Name(), 0o644)); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("site: write %s: %w", rel, err)
	}
	if err := os.Rename(tmp.Name(), full); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("site: write %s: %w", rel, err)
	}
	w.bytes += int64(buf.Len())
	return nil
}
