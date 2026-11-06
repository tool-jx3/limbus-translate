package site

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/rules"
)

// TestSiteFixtureIndexMatchesRules keeps the file entries of the front end's
// fixture index (site/fixtures/data/index.json) to what Go derives: the tests
// of the site run on that data, so it must be data lct site could write.
func TestSiteFixtureIndexMatchesRules(t *testing.T) {
	root := filepath.Join("..", "..")
	r, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "site", "fixtures", "data", "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var idx struct {
		Files  []FileSummary `json:"files"`
		Groups []IndexGroup  `json:"groups"`
	}
	if err := json.Unmarshal(data, &idx); err != nil {
		t.Fatal(err)
	}
	if len(idx.Files) == 0 {
		t.Fatal("the fixture index has no files")
	}
	for _, f := range idx.Files {
		n := rules.FileNode(f.Path)
		if f.Top != n.Top || f.Sub != n.Sub {
			t.Errorf("%s: top/sub = %q/%q, rules.FileNode gives %q/%q", f.Path, f.Top, f.Sub, n.Top, n.Sub)
		}
		var want *rules.Tag
		if tag, ok := r.FileTag(f.Path); ok {
			want = &tag
		}
		if !reflect.DeepEqual(f.Tag, want) {
			t.Errorf("%s: tag = %+v, rules.FileTag gives %+v", f.Path, f.Tag, want)
		}
	}
	if got := buildGroups(idx.Files); !reflect.DeepEqual(got, idx.Groups) {
		t.Errorf("groups = %+v, buildGroups gives %+v", idx.Groups, got)
	}
}

// TestSiteFixtureGlossaryUnitsMatchIndex keeps the units of the glossary
// fixtures to the tags their file has in the fixture index: a unit carries
// the family, top, sub and tag of its file, as lct site writes them.
func TestSiteFixtureGlossaryUnitsMatchIndex(t *testing.T) {
	dir := filepath.Join("..", "..", "site", "fixtures", "data")
	data, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var idx struct {
		Files []FileSummary `json:"files"`
	}
	if err := json.Unmarshal(data, &idx); err != nil {
		t.Fatal(err)
	}
	byPath := map[string]FileSummary{}
	for _, f := range idx.Files {
		byPath[f.Path] = f
	}
	paths, err := filepath.Glob(filepath.Join(dir, "glossary", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, p := range paths {
		if filepath.Base(p) == "index.json" {
			continue
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var g struct {
			Missing []GlossaryUnit `json:"missing"`
			Used    []GlossaryUnit `json:"used"`
		}
		if err := json.Unmarshal(raw, &g); err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		for _, u := range append(g.Missing, g.Used...) {
			path, _, _ := strings.Cut(u.Key, "|")
			f, ok := byPath[path]
			if !ok {
				t.Errorf("%s: %s: file %q is not in the fixture index", filepath.Base(p), u.Key, path)
				continue
			}
			checked++
			if u.Family != f.Family || u.Top != f.Top || u.Sub != f.Sub || !reflect.DeepEqual(u.Tag, f.Tag) {
				t.Errorf("%s: %s: family/top/sub/tag = %q/%q/%q/%+v, the index has %q/%q/%q/%+v",
					filepath.Base(p), u.Key, u.Family, u.Top, u.Sub, u.Tag, f.Family, f.Top, f.Sub, f.Tag)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no glossary fixture unit was checked")
	}
}
