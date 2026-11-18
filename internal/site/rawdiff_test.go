package site

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/rawdiff"
)

func dataDiff(from, to string, c rawdiff.Counts) *rawdiff.Diff {
	return &rawdiff.Diff{From: from, To: to, Counts: c,
		NewGroups: []rawdiff.Group{}, GoneGroups: []rawdiff.Group{}, FilesAdded: []rawdiff.File{}, FilesRemoved: []rawdiff.File{},
		Keys: []rawdiff.GroupKeys{}, Entries: []rawdiff.FileEntries{}, Tokens: []rawdiff.Token{}}
}

func TestExportDataChangesListsEverySnapshotNewestFirst(t *testing.T) {
	dir := t.TempDir()
	idx, err := ExportDataChanges([]DataChange{
		{Diff: dataDiff("raw-20261029", "raw-20261105", rawdiff.Counts{FilesAdded: 3}), At: "2026-11-05T03:00:00Z"},
		{Diff: dataDiff("raw-20261105", "raw-20261112", rawdiff.Counts{FilesAdded: 27, NewGroups: 2}), At: "2026-11-12T03:00:00Z"},
	}, dir)
	if err != nil {
		t.Fatal(err)
	}
	want := DataIndex{Snapshots: []DataSnapshot{
		{Tag: "raw-20261112", PrevTag: "raw-20261105", At: "2026-11-12T03:00:00Z", Counts: rawdiff.Counts{FilesAdded: 27, NewGroups: 2}},
		{Tag: "raw-20261105", PrevTag: "raw-20261029", At: "2026-11-05T03:00:00Z", Counts: rawdiff.Counts{FilesAdded: 3}},
	}}
	if !reflect.DeepEqual(idx, want) {
		t.Errorf("index = %+v, want %+v", idx, want)
	}
	b, err := os.ReadFile(filepath.Join(dir, "rawdiff", "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var onDisk DataIndex
	if err := json.Unmarshal(b, &onDisk); err != nil || !reflect.DeepEqual(onDisk, idx) {
		t.Errorf("index.json = %s, %v; want %+v", b, err, idx)
	}
	b, err = os.ReadFile(filepath.Join(dir, "rawdiff", "raw-20261112.json"))
	if err != nil {
		t.Fatal(err)
	}
	var d rawdiff.Diff
	if err := json.Unmarshal(b, &d); err != nil || d.From != "raw-20261105" || d.To != "raw-20261112" || d.Counts.NewGroups != 2 {
		t.Errorf("raw-20261112.json = %s, %v", b, err)
	}
	if !strings.Contains(string(b), `"tokens":[]`) {
		t.Errorf("raw-20261112.json has a list that is null: %s", b)
	}
}

func TestExportDataChangesWithNoSnapshotToCompare(t *testing.T) {
	dir := t.TempDir()
	idx, err := ExportDataChanges(nil, dir)
	if err != nil || idx.Snapshots == nil || len(idx.Snapshots) != 0 {
		t.Fatalf("index = %+v, %v; want an empty list", idx, err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "rawdiff", "index.json"))
	if err != nil || strings.TrimSpace(string(b)) != `{"snapshots":[]}` {
		t.Errorf("index.json = %q, %v", b, err)
	}
}

func TestExportDataChangesRefusesWhatIsNoTag(t *testing.T) {
	for _, tag := range []string{"", "../x", "a/b", `a\b`} {
		if _, err := ExportDataChanges([]DataChange{{Diff: dataDiff("raw-1", tag, rawdiff.Counts{})}}, t.TempDir()); err == nil {
			t.Errorf("tag %q: no error", tag)
		}
	}
	if _, err := ExportDataChanges(nil, ""); err == nil {
		t.Error("no output directory: no error")
	}
}
