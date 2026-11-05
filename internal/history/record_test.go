package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteThenListNewestFirst(t *testing.T) {
	root := t.TempDir()
	for _, r := range []Record{
		{Tag: "zh-TW-20261101.1", ReleasedAt: "2026-11-01T00:00:00Z"},
		{Tag: "zh-TW-20261103.1", ReleasedAt: "2026-11-03T00:00:00Z"},
		{Tag: "zh-TW-20261102.2", ReleasedAt: "2026-11-02T00:00:00Z"},
		{Tag: "zh-TW-20261102.1", ReleasedAt: "2026-11-02T00:00:00Z"},
	} {
		r := r
		if err := Write(root, &r); err != nil {
			t.Fatal(err)
		}
	}
	got, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	var tags []string
	for _, r := range got {
		tags = append(tags, r.Tag)
	}
	want := "zh-TW-20261103.1 zh-TW-20261102.2 zh-TW-20261102.1 zh-TW-20261101.1"
	if strings.Join(tags, " ") != want {
		t.Errorf("List = %v, want %s", tags, want)
	}
	if has, err := Has(root, "zh-TW-20261102.1"); err != nil || !has {
		t.Errorf("Has(written) = %v, %v", has, err)
	}
	if has, err := Has(root, "zh-TW-20261231.1"); err != nil || has {
		t.Errorf("Has(unwritten) = %v, %v", has, err)
	}
}

func TestListOfNoHistoryIsEmpty(t *testing.T) {
	got, err := List(t.TempDir())
	if err != nil || len(got) != 0 {
		t.Errorf("List = %v, %v", got, err)
	}
}

func TestRecordJSONHasNoNilLists(t *testing.T) {
	root := t.TempDir()
	if err := Write(root, &Record{Tag: "zh-TW-20261101.1"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".meta", "history", "zh-TW-20261101.1.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"other_files": []`, `"entries": []`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("missing %s in\n%s", want, data)
		}
	}
	got, err := List(root)
	if err != nil || len(got) != 1 || got[0].OtherFiles == nil || got[0].Entries == nil {
		t.Errorf("List = %+v, %v; want non-nil lists", got, err)
	}
}

func TestRecordEmpty(t *testing.T) {
	if !(&Record{}).Empty() {
		t.Error("blank Record is not Empty")
	}
	if (&Record{OtherFiles: []string{"x"}}).Empty() || (&Record{Entries: []Entry{{}}}).Empty() {
		t.Error("Record with content is Empty")
	}
}
