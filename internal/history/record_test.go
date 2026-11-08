package history

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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

func TestRecordRoundTripsCauses(t *testing.T) {
	root := t.TempDir()
	in := Record{
		Tag:           "zh-TW-20261108.1",
		CausesVersion: CausesVersion,
		Unknown:       1,
		Rules: map[string]RuleInfo{
			"rewrite:r1": {Kind: "rewrite", ID: "r1", Path: "rules/rewrite.yaml", Title: "T", Commit: "abc"},
		},
		Entries: []Entry{{
			File: "a.json", Field: "f", Pointer: "/0", Kind: KindZHChanged, Cause: CauseAdjustment,
			Causes: []Cause{
				{Kind: CauseKindRewrite, ID: "r1"},
				{Kind: CauseKindAdjustment, ID: "a1", Batch: "b1", Reason: "why"},
				{Kind: CauseKindUnknown, Note: "n"},
			},
		}},
	}
	if err := Write(root, &in); err != nil {
		t.Fatal(err)
	}
	got, err := List(root)
	if err != nil || len(got) != 1 {
		t.Fatalf("List = %v, %v", got, err)
	}
	if got[0].CausesVersion != 1 || got[0].Unknown != 1 {
		t.Errorf("version/unknown = %d/%d", got[0].CausesVersion, got[0].Unknown)
	}
	if !reflect.DeepEqual(got[0].Rules, in.Rules) {
		t.Errorf("Rules = %v, want %v", got[0].Rules, in.Rules)
	}
	if !reflect.DeepEqual(got[0].Entries[0].Causes, in.Entries[0].Causes) {
		t.Errorf("Causes = %v, want %v", got[0].Entries[0].Causes, in.Entries[0].Causes)
	}
}

// A record without causes is written as one was before causes existed, so
// that the site exports the records made then byte for byte as before.
func TestRecordWithoutCausesHasNoCauseFields(t *testing.T) {
	root := t.TempDir()
	rec := Record{Tag: "zh-TW-20261101.1", Entries: []Entry{{File: "a.json", Field: "f", Pointer: "/0", Kind: KindZHChanged, Cause: CauseLate}}}
	if err := Write(root, &rec); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(filepath.Join(root, ".meta", "history", "zh-TW-20261101.1.json"))
	if err != nil {
		t.Fatal(err)
	}
	marshalled, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range [][]byte{written, marshalled} {
		for _, field := range []string{`"causes_version"`, `"unknown"`, `"rules"`, `"causes"`} {
			if bytes.Contains(data, []byte(field)) {
				t.Errorf("a record without causes has %s:\n%s", field, data)
			}
		}
	}
}

func TestOldRecordReadsWithoutCauses(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".meta", "history")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	old := `{"tag":"zh-TW-20261101.1","entries":[{"file":"a.json","field":"f","ptr":"/0","kind":"zh-changed","cause":"late"}]}`
	if err := os.WriteFile(filepath.Join(dir, "zh-TW-20261101.1.json"), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := List(root)
	if err != nil || len(got) != 1 {
		t.Fatalf("List = %v, %v", got, err)
	}
	r := got[0]
	if r.CausesVersion != 0 || r.Rules != nil || r.Entries[0].Causes != nil {
		t.Errorf("new fields not empty: %+v", r)
	}
	if r.Entries[0].Cause != CauseLate {
		t.Errorf("Cause = %q", r.Entries[0].Cause)
	}
}

func TestIsRuleCause(t *testing.T) {
	for _, k := range []string{CauseKindRewrite, CauseKindAdjustment, CauseKindAdjustmentExpired, CauseKindRequeue} {
		if !IsRuleCause(k) {
			t.Errorf("IsRuleCause(%q) = false", k)
		}
	}
	for _, k := range []string{CauseKindReuse, CauseKindKRChanged, CauseKindNew, CauseKindDeleted,
		CauseKindLate, CauseKindRetranslated, CauseKindOther, CauseKindUnknown, ""} {
		if IsRuleCause(k) {
			t.Errorf("IsRuleCause(%q) = true", k)
		}
	}
}
