// Package pack opens language packs on disk: the official KR/EN/JP packs and
// the upstream translation packs. A pack is a directory tree of JSON files
// whose names carry a language prefix such as "KR_".
package pack

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/tool-jx3/limbus-translate/internal/jsonx"
)

// Pack is one language pack. File names exposed by Pack are slash-separated
// paths relative to the pack root with the language prefix removed from the
// file name only, for example "StoryData/S001A.json" for
// <root>/StoryData/KR_S001A.json. Directory names are never changed.
type Pack struct {
	root   string
	prefix string
	files  []string
	set    map[string]struct{}

	mu   sync.Mutex
	docs map[string]*jsonx.Node
}

// Set is the official packs of the three languages.
type Set struct {
	KR, EN, JP *Pack
}

// Open lists the pack under root. Only ".json" files whose own name starts
// with prefix belong to the pack; everything else is ignored. An empty prefix
// accepts every ".json" file (upstream translation packs have none). The
// files are not read until Doc is called.
func Open(root, prefix string) (*Pack, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("pack: open %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("pack: open %s: not a directory", root)
	}
	p := &Pack{
		root:   root,
		prefix: prefix,
		set:    make(map[string]struct{}),
		docs:   make(map[string]*jsonx.Node),
	}
	err = filepath.WalkDir(root, func(full string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(d.Name()) != ".json" {
			return nil
		}
		name, ok := strings.CutPrefix(d.Name(), prefix)
		if !ok {
			return nil
		}
		rel, err := filepath.Rel(root, filepath.Dir(full))
		if err != nil {
			return err
		}
		file := path.Join(filepath.ToSlash(rel), name) // path.Join cleans the "." of the root dir
		p.files = append(p.files, file)
		p.set[file] = struct{}{}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("pack: scan %s: %w", root, err)
	}
	sort.Strings(p.files)
	return p, nil
}

// OpenRaw opens the three official packs held in a raw checkout:
// <rawRoot>/kr (files KR_*), <rawRoot>/en (EN_*) and <rawRoot>/jp (JP_*).
// All three directories must exist.
func OpenRaw(rawRoot string) (Set, error) {
	var s Set
	for _, l := range []struct {
		dir, prefix string
		dst         **Pack
	}{
		{"kr", "KR_", &s.KR},
		{"en", "EN_", &s.EN},
		{"jp", "JP_", &s.JP},
	} {
		p, err := Open(filepath.Join(rawRoot, l.dir), l.prefix)
		if err != nil {
			return Set{}, fmt.Errorf("pack: raw %s pack: %w", l.dir, err)
		}
		*l.dst = p
	}
	return s, nil
}

// Files lists the pack's files, sorted, as slash-separated relative paths
// without the language prefix. The caller may modify the returned slice.
func (p *Pack) Files() []string {
	return slices.Clone(p.files)
}

// Has reports whether file is part of the pack.
func (p *Pack) Has(file string) bool {
	_, ok := p.set[file]
	return ok
}

// Path returns the on-disk path of file: the prefix goes back onto the file
// name. It does not check that the file exists.
func (p *Pack) Path(file string) string {
	dir, name := path.Split(file)
	return filepath.Join(p.root, filepath.FromSlash(dir), p.prefix+name)
}

// Doc parses file and caches the tree, so later calls return the same node.
// Callers that modify the tree must Clone it first.
func (p *Pack) Doc(file string) (*jsonx.Node, error) {
	if !p.Has(file) {
		return nil, fmt.Errorf("pack: %s is not in the pack at %s", file, p.root)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if doc, ok := p.docs[file]; ok {
		return doc, nil
	}
	data, err := os.ReadFile(p.Path(file))
	if err != nil {
		return nil, fmt.Errorf("pack: read %s: %w", file, err)
	}
	doc, err := jsonx.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("pack: parse %s: %w", file, err)
	}
	p.docs[file] = doc
	return doc, nil
}
