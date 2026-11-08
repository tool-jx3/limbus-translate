package site

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/history"
)

func updRecord(tag, prev, at, prevAt, rawTag, prevRaw string, entries ...history.Entry) history.Record {
	return history.Record{
		Tag: tag, PrevTag: prev, ReleasedAt: at, PrevReleasedAt: prevAt, RawTag: rawTag, PrevRawTag: prevRaw,
		Counts: history.Counts{New: len(entries)}, OtherFiles: []string{"a", "b"}, Entries: entries,
	}
}

func readUpdateIndex(t *testing.T, dir string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "updates", "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestExportUpdatesListsEveryReleaseNewestFirst(t *testing.T) {
	dir := t.TempDir()
	e := history.Entry{File: "Skills.json", Pointer: "/a", Kind: history.KindNew}
	recs := []history.Record{
		updRecord("zh-TW-20261103.3", "zh-TW-20261103.2", "2026-11-03T12:00:00Z", "2026-11-03T08:00:00Z", "raw-2", "raw-2", e),
		updRecord("zh-TW-20261103.2", "zh-TW-20261103.1", "2026-11-03T08:00:00Z", "2026-11-02T08:00:00Z", "raw-2", "raw-1", e, e),
	}
	rels := []Release{
		{"zh-TW-20261103.1", "2026-11-02T08:00:00Z"},
		{"zh-TW-20261103.2", "2026-11-03T08:00:00Z"},
		{"zh-TW-20261103.3", "2026-11-03T12:00:00Z"},
	}
	idx, err := ExportUpdates(rels, recs, testRules(), dir)
	if err != nil {
		t.Fatal(err)
	}
	var kinds, tags []string
	for _, r := range idx.Releases {
		kinds, tags = append(kinds, r.Kind), append(tags, r.Tag)
	}
	if want := []string{"rebuild", "game", "first"}; !reflect.DeepEqual(kinds, want) {
		t.Errorf("kinds = %v, want %v", kinds, want)
	}
	if want := []string{"zh-TW-20261103.3", "zh-TW-20261103.2", "zh-TW-20261103.1"}; !reflect.DeepEqual(tags, want) {
		t.Errorf("tags = %v, want %v", tags, want)
	}
	if r := idx.Releases[1]; !r.Recorded || r.Files != 1 || r.OtherFiles != 2 || r.Counts.New != 2 || r.PrevRawTag != "raw-1" {
		t.Errorf("game row = %+v", r)
	}
	if f := idx.Releases[2]; f.ReleasedAt != "2026-11-02T08:00:00Z" || f.RawTag != "raw-1" || f.Counts != (history.Counts{}) || f.PrevTag != "" {
		t.Errorf("first row = %+v", f)
	}
	var onDisk UpdateIndex
	if err := json.Unmarshal(readUpdateIndex(t, dir), &onDisk); err != nil || !reflect.DeepEqual(onDisk, idx) {
		t.Errorf("index.json = %+v, %v; want %+v", onDisk, err, idx)
	}
	b, err := os.ReadFile(filepath.Join(dir, "updates", "zh-TW-20261103.3.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rec history.Record
	if err := json.Unmarshal(b, &rec); err != nil || rec.Tag != "zh-TW-20261103.3" || len(rec.Entries) != 1 {
		t.Errorf("record file = %+v, %v", rec, err)
	}
}

func TestExportUpdatesOrdersTheEntriesByFile(t *testing.T) {
	dir := t.TempDir()
	mk := func(file, ptr string) history.Entry {
		return history.Entry{File: file, Pointer: ptr, Kind: history.KindNew}
	}
	rec := updRecord("zh-TW-20261103.1", "", "2026-11-03T08:00:00Z", "", "raw-1", "raw-1",
		mk("Skills.json", "/2"), mk("StoryData/S10.json", "/1"), mk("Skills.json", "/1"),
		mk("StoryData/S2.json", "/9"), mk("StoryData/S10.json", "/0"))
	if _, err := ExportUpdates([]Release{{"zh-TW-20261102.1", "2026-11-02T08:00:00Z"}, {"zh-TW-20261103.1", "2026-11-03T08:00:00Z"}}, []history.Record{rec}, testRules(), dir); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "updates", "zh-TW-20261103.1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got history.Record
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, e := range got.Entries {
		order = append(order, e.File+e.Pointer)
	}
	want := []string{"StoryData/S2.json/9", "StoryData/S10.json/1", "StoryData/S10.json/0", "Skills.json/2", "Skills.json/1"}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("order = %v, want %v", order, want)
	}
}

func TestExportUpdatesWithNoRecords(t *testing.T) {
	dir := t.TempDir()
	idx, err := ExportUpdates(nil, nil, testRules(), dir)
	if err != nil || idx.Releases == nil || len(idx.Releases) != 0 {
		t.Fatalf("idx = %+v, %v", idx, err)
	}
	if got := string(readUpdateIndex(t, dir)); got != "{\"releases\":[]}\n" {
		t.Errorf("index.json = %q", got)
	}
}

// A record whose tag never became a Release (a prepared tag that was dropped)
// is neither listed nor written.
func TestExportUpdatesSkipsARecordWithoutAReleaseTag(t *testing.T) {
	dir := t.TempDir()
	rels := []Release{{"zh-TW-20261101.1", "2026-11-01T08:00:00Z"}, {"zh-TW-20261102.1", "2026-11-02T08:00:00Z"}}
	recs := []history.Record{
		updRecord("zh-TW-20261103.1", "zh-TW-20261102.1", "2026-11-03T08:00:00Z", "2026-11-02T08:00:00Z", "raw-3", "raw-2"),
		updRecord("zh-TW-20261102.1", "zh-TW-20261101.1", "2026-11-02T08:00:00Z", "2026-11-01T08:00:00Z", "raw-2", "raw-1"),
	}
	idx, err := ExportUpdates(rels, recs, testRules(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Releases) != 2 || idx.Releases[0].Tag != "zh-TW-20261102.1" || idx.Releases[1].Kind != UpdateFirst {
		t.Errorf("releases = %+v, want the two Release tags", idx.Releases)
	}
	if _, err := os.Stat(filepath.Join(dir, "updates", "zh-TW-20261103.1.json")); !os.IsNotExist(err) {
		t.Errorf("the record of an abandoned tag was written: %v", err)
	}
}

// A middle Release without a record is listed, without counts and unlinked; the
// oldest stays the first; the earlier ones do not vanish.
func TestExportUpdatesListsAReleaseWithoutARecord(t *testing.T) {
	dir := t.TempDir()
	rels := []Release{
		{"zh-TW-20261101.1", "2026-11-01T08:00:00Z"},
		{"zh-TW-20261102.1", "2026-11-02T08:00:00Z"},
		{"zh-TW-20261103.1", "2026-11-03T08:00:00Z"},
	}
	e := history.Entry{File: "Skills.json", Pointer: "/a", Kind: history.KindNew}
	recs := []history.Record{updRecord("zh-TW-20261103.1", "zh-TW-20261102.1", "2026-11-03T09:00:00Z", "2026-11-02T08:00:00Z", "raw-3", "raw-2", e)}
	idx, err := ExportUpdates(rels, recs, testRules(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Releases) != 3 {
		t.Fatalf("releases = %+v, want 3 rows", idx.Releases)
	}
	top, mid, first := idx.Releases[0], idx.Releases[1], idx.Releases[2]
	if top.Tag != "zh-TW-20261103.1" || !top.Recorded || top.Kind != UpdateGame || top.Counts.New != 1 || top.PrevTag != "zh-TW-20261102.1" {
		t.Errorf("newest row = %+v", top)
	}
	if mid.Tag != "zh-TW-20261102.1" || mid.Recorded || mid.Counts != (history.Counts{}) || mid.PrevTag != "" || mid.ReleasedAt != "2026-11-02T08:00:00Z" {
		t.Errorf("middle row = %+v, want no record: no counts, not recorded, the date of the tag", mid)
	}
	if first.Tag != "zh-TW-20261101.1" || first.Kind != UpdateFirst || first.Recorded {
		t.Errorf("first row = %+v", first)
	}
	if _, err := os.Stat(filepath.Join(dir, "updates", "zh-TW-20261102.1.json")); !os.IsNotExist(err) {
		t.Errorf("a file for the Release without a record: %v", err)
	}
}

func TestExportUpdatesKeepsCausesAndRules(t *testing.T) {
	dir := t.TempDir()
	e := history.Entry{File: "Skills.json", Pointer: "/a", Kind: history.KindZHChanged,
		Causes: []history.Cause{{Kind: history.CauseKindRewrite, ID: "r1"}, {Kind: history.CauseKindUnknown, Note: "x"}}}
	rec := updRecord("zh-TW-20261103.1", "", "2026-11-03T08:00:00Z", "", "raw-1", "raw-1", e)
	rec.CausesVersion, rec.Unknown = history.CausesVersion, 1
	rec.Rules = map[string]history.RuleInfo{"rewrite:r1": {Kind: "rewrite", ID: "r1", Path: "rules/rewrites/r1.yaml", Title: "T", Commit: "abc"}}
	if _, err := ExportUpdates([]Release{{"zh-TW-20261102.1", "2026-11-02T08:00:00Z"}, {"zh-TW-20261103.1", "2026-11-03T08:00:00Z"}}, []history.Record{rec}, testRules(), dir); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "updates", "zh-TW-20261103.1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got history.Record
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Entries[0].Causes, e.Causes) || !reflect.DeepEqual(got.Rules, rec.Rules) || got.CausesVersion != 1 || got.Unknown != 1 {
		t.Errorf("got causes %+v rules %+v v%d unknown %d", got.Entries[0].Causes, got.Rules, got.CausesVersion, got.Unknown)
	}
}
