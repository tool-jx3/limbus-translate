package site

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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
