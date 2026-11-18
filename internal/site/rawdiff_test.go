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
	// The packs of the other languages of the newest snapshot: JP differs from KR in three changes, EN in none.
	jp := dataDiff("raw-20261105", "raw-20261112", rawdiff.Counts{FilesAdded: 29})
	jp.Lang = "jp"
	jp.VsKR = &rawdiff.Versus{
		OnlyHere:  []rawdiff.Item{{Kind: rawdiff.ItemFile, Change: rawdiff.Added, Name: "A.json"}, {Kind: rawdiff.ItemFile, Change: rawdiff.Added, Name: "B.json"}},
		OnlyOther: []rawdiff.Item{{Kind: rawdiff.ItemKey, Change: rawdiff.Added, Name: "Bufs dataList[].excludeAutoLink"}},
	}
	en := dataDiff("raw-20261105", "raw-20261112", rawdiff.Counts{FilesAdded: 27})
	en.Lang = "en"
	en.VsKR = &rawdiff.Versus{OnlyHere: []rawdiff.Item{}, OnlyOther: []rawdiff.Item{}}
	idx, err := ExportDataChanges([]DataChange{
		{Diff: dataDiff("raw-20261029", "raw-20261105", rawdiff.Counts{FilesAdded: 3}), At: "2026-11-05T03:00:00Z"},
		{Diff: dataDiff("raw-20261105", "raw-20261112", rawdiff.Counts{FilesAdded: 27, NewGroups: 2}), At: "2026-11-12T03:00:00Z", Others: []*rawdiff.Diff{jp, en}},
	}, dir)
	if err != nil {
		t.Fatal(err)
	}
	want := DataIndex{Snapshots: []DataSnapshot{
		{Tag: "raw-20261112", PrevTag: "raw-20261105", At: "2026-11-12T03:00:00Z", Counts: rawdiff.Counts{FilesAdded: 27, NewGroups: 2}, VsKR: map[string]int{"jp": 3, "en": 0}},
		{Tag: "raw-20261105", PrevTag: "raw-20261029", At: "2026-11-05T03:00:00Z", Counts: rawdiff.Counts{FilesAdded: 3}, VsKR: map[string]int{}},
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
	// The pack of another language has a file of its own, with what differs from KR.
	b, err = os.ReadFile(filepath.Join(dir, "rawdiff", "raw-20261112.jp.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got rawdiff.Diff
	if err := json.Unmarshal(b, &got); err != nil || got.Lang != "jp" || got.Counts.FilesAdded != 29 || got.VsKR == nil || len(got.VsKR.OnlyHere) != 2 {
		t.Errorf("raw-20261112.jp.json = %s, %v", b, err)
	}
	for name, want := range map[string]bool{"raw-20261112.en.json": true, "raw-20261105.jp.json": false} {
		if _, err := os.Stat(filepath.Join(dir, "rawdiff", name)); (err == nil) != want {
			t.Errorf("rawdiff/%s: %v, want written = %v", name, err, want)
		}
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
	for _, lang := range []string{"", "kr", "../x", "JP", "j/p"} {
		other := dataDiff("raw-1", "raw-2", rawdiff.Counts{})
		other.Lang = lang
		other.VsKR = &rawdiff.Versus{}
		if _, err := ExportDataChanges([]DataChange{{Diff: dataDiff("raw-1", "raw-2", rawdiff.Counts{}), Others: []*rawdiff.Diff{other}}}, t.TempDir()); err == nil {
			t.Errorf("another pack of the language %q: no error", lang)
		}
	}
	if _, err := ExportDataChanges(nil, ""); err == nil {
		t.Error("no output directory: no error")
	}
}
