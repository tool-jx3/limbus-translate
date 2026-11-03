package site

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/tool-jx3/limbus-translate/internal/store"
)

// FormatVersion is Meta.Format: the version of the layout and the JSON keys of
// the data. The viewer asks for a refresh when it meets another version.
const FormatVersion = 1

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
}

// Export writes the data of the viewer to outDir, after emptying it: meta.json,
// index.json, files/<path> for each file of the KR pack, search/keys.json and
// search/{kr,jp,en,zh}.json. The corpus holds the units of every file in the
// order of the index, but not the copies (identifiers and texts that are the
// same in every language). meta supplies what Export cannot know: RawTag,
// RulesCommit, TWCommit and Release. The output depends on in alone, so the
// same inputs give the same bytes.
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
	w := &writer{dir: outDir}
	if err := w.clear(); err != nil {
		return Result{}, err
	}

	idx := Index{Families: []IndexFamily{}, Files: make([]FileSummary, 0, len(paths))}
	var keys [][2]any
	corpus := make([][]string, len(searchLangs)) // by searchLangs
	units := 0
	for n, p := range paths {
		file, sum, err := BuildFile(in, p)
		if err != nil {
			return Result{}, err
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
		units += len(file.Units)
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
	meta.Format = FormatVersion
	meta.Langs = append([]string(nil), searchLangs...)
	meta.Files, meta.Units = len(paths), units
	if err := w.writeJSON("meta.json", meta); err != nil {
		return Result{}, err
	}
	return Result{Out: outDir, Files: len(paths), Units: units, SearchUnits: len(keys), Bytes: w.bytes}, nil
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
