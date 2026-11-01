package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/batch"
	"github.com/tool-jx3/limbus-translate/internal/bootstrap"
	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

const batchYAML = `worktrees: { raw: .work/raw, tw: .work/tw }
batch: { max_units: 40, story_max_units: 120, max_attempts: 3, lease_minutes: 120 }
`

const (
	batchKR = `{"dataList":[
 {"id":1,"content":"[Laceration] 위력 5 부여"},
 {"id":2,"content":"뫼르소가 말했다"}
]}`
	batchEN = `{"dataList":[
 {"id":1,"content":"Inflict 5 [Laceration] Potency"},
 {"id":2,"content":"Meursault said"}
]}`
	batchJP = `{"dataList":[
 {"id":1,"content":"[Laceration]を5付与"},
 {"id":2,"content":"ムルソーは言った"}
]}`
)

// cliNow is the clock of the batch commands in these tests.
var cliNow = time.Date(2026, 11, 1, 3, 4, 5, 0, time.UTC)

var (
	batchKey0 = "Sample.json|/dataList/0/content"
	batchKey1 = "Sample.json|/dataList/1/content"
)

// batchRepo makes an lct repo root ready for the batch commands: a git repo
// with the real rules committed, a raw worktree with one file and its
// snapshot record, and two pending units. The foreign name 默爾索 is
// registered, and LCT_MODEL is test-model.
func batchRepo(t *testing.T) (root string, st *store.Store) {
	t.Helper()
	// A fixed clock: batch ids and lease times do not depend on when the
	// test runs.
	oldNow := batchNow
	batchNow = func() time.Time { return cliNow }
	t.Cleanup(func() { batchNow = oldNow })
	t.Setenv("LCT_MODEL", "test-model")

	root = snapshotRepo(t)
	writeTestFile(t, root, "lct.yaml", batchYAML)
	writeTestFile(t, root, ".gitignore", "/.work/\n/work/\n")
	for _, f := range []string{"glossary/terms.yaml", "guides/routing.yaml"} {
		copyFile(t, filepath.Join(repoRoot, f), filepath.Join(root, f))
	}
	copyDir(t, filepath.Join(repoRoot, "guides/characters"), filepath.Join(root, "guides/characters"))
	runGitIn(t, root, "add", "-A")
	runGitIn(t, root, "commit", "-q", "-m", "rules")

	rawWT := filepath.Join(root, ".work", "raw")
	writeTestFile(t, rawWT, "kr/KR_Sample.json", batchKR)
	writeTestFile(t, rawWT, "en/EN_Sample.json", batchEN)
	writeTestFile(t, rawWT, "jp/JP_Sample.json", batchJP)
	writeTestFile(t, rawWT, "meta/source.json", `{"tag":"raw-20261029","token":"t","created_at":"2026-10-29 02:55:23","fetched_at":"2026-10-29T03:00:00Z","files":{}}`+"\n")

	st = store.Open(filepath.Join(root, ".work", "tw"))
	if err := st.SavePending([]store.PendingItem{
		{Key: batchKey0, Reason: "new", Refs: map[string]string{"llc": "赋予 5 级裂伤"}},
		{Key: batchKey1, Reason: "new"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.WriteMetaJSON(bootstrap.ForeignNamesFile, []string{"默爾索"}); err != nil {
		t.Fatal(err)
	}
	return root, st
}

type nextOut struct {
	Batch *string `json:"batch"`
	Path  string  `json:"path"`
	Units int     `json:"units"`
}

func runNextJSON(t *testing.T, root string, args ...string) (nextOut, string, error) {
	t.Helper()
	ctx, out := testCtx(t, root, "", true)
	err := runBatchNext(ctx, args)
	var got nextOut
	if err == nil {
		if err := json.Unmarshal(out.Bytes(), &got); err != nil {
			t.Fatalf("output is not JSON: %v\n%s", err, out)
		}
	}
	return got, out.String(), err
}

type submitOut struct {
	Batch    string `json:"batch"`
	Accepted []string
	Reused   []string
	Rejected []struct {
		Key    string
		Errors []struct{ Code, Message string }
	}
	Warnings []struct {
		Key    string
		Errors []struct{ Code string }
	}
	NeedsReview []string `json:"needs_review"`
	Released    []string
	Notes       int
}

func runSubmitJSON(t *testing.T, root string, args ...string) (submitOut, string, error) {
	t.Helper()
	ctx, out := testCtx(t, root, "", true)
	err := runBatchSubmit(ctx, args)
	var got submitOut
	if out.Len() > 0 {
		if jerr := json.Unmarshal(out.Bytes(), &got); jerr != nil {
			t.Fatalf("output is not JSON: %v\n%s", jerr, out)
		}
	}
	return got, out.String(), err
}

func writeOut(t *testing.T, root, id string, units ...batch.OutUnit) {
	t.Helper()
	data, err := json.Marshal(units)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, "work/batches/"+id+".out.json", string(data))
}

func TestBatchNextWritesTheBatchFile(t *testing.T) {
	root, st := batchRepo(t)
	got, raw, err := runNextJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Batch == nil || got.Units != 2 {
		t.Fatalf("output = %s, want a batch of 2 units", raw)
	}
	id := *got.Batch
	wantPath := filepath.Join(root, "work", "batches", id+".json")
	if got.Path != wantPath {
		t.Errorf("path = %q, want %q", got.Path, wantPath)
	}

	data, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(data), "}\n") || !strings.Contains(string(data), "\n  \"id\": \""+id+"\"") {
		t.Errorf("the batch file should be indented JSON with a final newline:\n%.200s", data)
	}
	if strings.Contains(string(data), `\u003c`) {
		t.Error("the batch file escapes < and >")
	}
	var b batch.Batch
	if err := json.Unmarshal(data, &b); err != nil {
		t.Fatal(err)
	}
	if b.ID != id || b.File != "Sample.json" || len(b.Units) != 2 {
		t.Fatalf("batch = %+v", b)
	}
	u := b.Units[0]
	if u.EN == nil || u.JP == nil || u.Refs["llc"] == "" || !slices.Equal(u.Keep.IDs, []string{"[Laceration]"}) {
		t.Errorf("unit 0 = %+v, want EN, JP, the llc ref and the [Laceration] ID", u)
	}

	pending, err := st.LoadPending()
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range pending {
		if it.Lease == nil || it.Lease.Batch != id {
			t.Errorf("item %s has lease %+v, want one for %s", it.Key, it.Lease, id)
		}
	}

	// Nothing is left to hand out.
	_, raw, err = runNextJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	var null map[string]any
	if err := json.Unmarshal([]byte(raw), &null); err != nil {
		t.Fatal(err)
	}
	if v, ok := null["batch"]; !ok || v != nil || len(null) != 1 {
		t.Errorf("output = %s, want {\"batch\": null}", raw)
	}
	ctx, out := testCtx(t, root, "", false)
	if err := runBatchNext(ctx, nil); err != nil || !strings.Contains(out.String(), "nothing") {
		t.Errorf("text output = %q, %v, want a note that there is nothing to hand out", out, err)
	}
}

func TestBatchNextOptions(t *testing.T) {
	root, _ := batchRepo(t)
	got, raw, err := runNextJSON(t, root, "--max-units", "1", "--family", "other")
	if err != nil || got.Units != 1 {
		t.Fatalf("output = %s, %v, want 1 unit", raw, err)
	}
	if got, raw, err := runNextJSON(t, root, "--family", "story-main"); err != nil || got.Batch != nil {
		t.Errorf("output = %s, %v, want no batch in another family", raw, err)
	}
	got, raw, err = runNextJSON(t, root, "--family", "other")
	if err != nil || got.Units != 1 {
		t.Errorf("output = %s, %v, want the other unit", raw, err)
	}
}

func TestBatchNextUserErrors(t *testing.T) {
	root, _ := batchRepo(t)
	for _, args := range [][]string{
		{"--max-units", "0"},
		{"--max-units", "-3"},
		{"--family", "nonsense"},
		{"extra"},
		{"--nope"},
	} {
		if _, _, err := runNextJSON(t, root, args...); !isUserError(err) {
			t.Errorf("batch next %v: err = %v, want a user error", args, err)
		}
	}

	if err := os.Remove(filepath.Join(root, ".work", "raw", "meta", "source.json")); err != nil {
		t.Fatal(err)
	}
	_, _, err := runNextJSON(t, root)
	if !isUserError(err) || !strings.Contains(err.Error(), "snapshot") {
		t.Errorf("without a snapshot: err = %v, want a user error that mentions snapshot", err)
	}
}

// A line that another file says the same way is translated once: the second
// file's line waits while the first is out, and takes its translation.
func TestBatchSubmitReusesTheTranslationOfTwins(t *testing.T) {
	root, st := batchRepo(t)
	const twinKey = "Twin.json|/dataList/0/content"
	writeTestFile(t, filepath.Join(root, ".work", "raw"), "kr/KR_Twin.json", `{"dataList":[{"id":1,"content":"[Laceration] 위력 5 부여"}]}`)
	pending, err := st.LoadPending()
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SavePending(append(pending, store.PendingItem{Key: twinKey, Reason: "new"})); err != nil {
		t.Fatal(err)
	}

	next, _, err := runNextJSON(t, root)
	if err != nil || next.Batch == nil || next.Units != 2 {
		t.Fatalf("batch next = %+v, %v, want the two units of Sample.json", next, err)
	}
	if again, raw, err := runNextJSON(t, root); err != nil || again.Batch != nil {
		t.Fatalf("second batch next = %s, %v, want no batch: only the twin is left", raw, err)
	}

	id := *next.Batch
	writeOut(t, root, id,
		batch.OutUnit{Key: batchKey0, ZH: "賦予5級[Laceration]威力"},
		batch.OutUnit{Key: batchKey1, ZH: "說了話"},
	)
	got, raw, err := runSubmitJSON(t, root, id)
	if err != nil {
		t.Fatalf("submit: %v\n%s", err, raw)
	}
	if !slices.Equal(got.Reused, []string{twinKey}) || len(got.Accepted) != 2 {
		t.Errorf("output = %s, want %s reused", raw, twinKey)
	}
	recs, err := st.LoadRecords()
	if err != nil {
		t.Fatal(err)
	}
	if rec := recs[twinKey]; rec.ZH != "賦予5級 [Laceration] 威力" || rec.Origin != store.OriginClaude || rec.Batch != id || rec.Model != "test-model" {
		t.Errorf("record of the twin = %+v, want the translation of %s from batch %s", rec, batchKey0, id)
	}
	if left, _ := st.LoadPending(); len(left) != 0 {
		t.Errorf("pending = %+v, want empty", left)
	}

	// The text report counts them.
	text := submitReport{Batch: id, SubmitResult: batch.SubmitResult{Reused: []string{twinKey}}}.String()
	if !strings.Contains(text, "1 reused") || !strings.Contains(text, twinKey) {
		t.Errorf("text report = %q, want the reused count and key", text)
	}
}

func TestBatchSubmit(t *testing.T) {
	root, st := batchRepo(t)
	next, _, err := runNextJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	id := *next.Batch
	writeOut(t, root, id,
		batch.OutUnit{Key: batchKey0, ZH: "賦予5級[Laceration]威力", Note: "裂傷改用流血？"},
		batch.OutUnit{Key: batchKey1, ZH: "默爾索說了話"}, // a foreign name: accepted with a warning
	)

	t.Setenv("LCT_MODEL", "env-model")
	got, raw, err := runSubmitJSON(t, root, id)
	if err != nil {
		t.Fatalf("submit: %v\n%s", err, raw)
	}
	if got.Batch != id || len(got.Accepted) != 2 || len(got.Rejected) != 0 {
		t.Fatalf("output = %s", raw)
	}
	if len(got.Warnings) != 1 || got.Warnings[0].Key != batchKey1 || got.Warnings[0].Errors[0].Code != "foreign-name" {
		t.Errorf("warnings = %+v, want foreign-name on %s", got.Warnings, batchKey1)
	}
	if got.Notes != 1 {
		t.Errorf("notes = %d, want 1", got.Notes)
	}
	if notes, err := st.LoadNotes(); err != nil || len(notes) != 1 || notes[0].Key != batchKey0 || notes[0].Model != "env-model" || notes[0].Batch != id {
		t.Errorf("stored notes = %+v, %v, want the note of %s by env-model", notes, err, batchKey0)
	}
	for _, field := range []string{`"accepted"`, `"reused"`, `"rejected"`, `"warnings"`, `"needs_review"`, `"released"`, `"notes"`} {
		if !strings.Contains(raw, field) {
			t.Errorf("output lacks %s:\n%s", field, raw)
		}
	}

	recs, err := st.LoadRecords()
	if err != nil {
		t.Fatal(err)
	}
	rec := recs[batchKey0]
	if rec.Origin != store.OriginClaude || rec.Model != "env-model" || rec.Batch != id || rec.Since != "raw-20261029" {
		t.Errorf("record = %+v, want claude, LCT_MODEL, the batch and the raw tag", rec)
	}
	if head := runGitIn(t, root, "rev-parse", "HEAD"); rec.Rules != head {
		t.Errorf("record rules = %q, want HEAD %s: the rules are committed", rec.Rules, head)
	}
	if rec.ZH != "賦予5級 [Laceration] 威力" {
		t.Errorf("record ZH = %q, want it normalized", rec.ZH)
	}
	if pending, _ := st.LoadPending(); len(pending) != 0 {
		t.Errorf("pending = %v, want empty", pending)
	}
}

func TestBatchSubmitModelFlagBeatsEnvironment(t *testing.T) {
	root, st := batchRepo(t)
	next, _, err := runNextJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	id := *next.Batch
	writeOut(t, root, id, batch.OutUnit{Key: batchKey1, ZH: "有人說了話"})

	t.Setenv("LCT_MODEL", "env-model")
	// The flag may come after the id, and units left out are released.
	got, _, err := runSubmitJSON(t, root, id, "--model", "flag-model")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{batchKey0}; !slices.Equal(got.Released, want) {
		t.Errorf("released = %v, want %v", got.Released, want)
	}
	recs, _ := st.LoadRecords()
	if recs[batchKey1].Model != "flag-model" {
		t.Errorf("model = %q, want flag-model", recs[batchKey1].Model)
	}
	pending, _ := st.LoadPending()
	if len(pending) != 1 || pending[0].Key != batchKey0 || pending[0].Lease != nil {
		t.Errorf("pending = %+v, want the omitted unit, released", pending)
	}
}

func TestBatchSubmitRejectionIsAFinding(t *testing.T) {
	root, st := batchRepo(t)
	next, _, err := runNextJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	id := *next.Batch
	writeOut(t, root, id,
		batch.OutUnit{Key: batchKey0, ZH: "賦予5級威力"}, // [Laceration] is missing
		batch.OutUnit{Key: batchKey1, ZH: "有人說了話"},
	)
	got, raw, err := runSubmitJSON(t, root, id)
	if !errors.Is(err, cli.ErrFindings) {
		t.Fatalf("err = %v, want cli.ErrFindings (exit 1)", err)
	}
	if len(got.Accepted) != 1 || len(got.Rejected) != 1 || got.Rejected[0].Key != batchKey0 {
		t.Fatalf("output = %s", raw)
	}
	if is := got.Rejected[0].Errors; len(is) == 0 || is[0].Code != "id-mismatch" || !strings.Contains(is[0].Message, "[Laceration]") {
		t.Errorf("rejected errors = %+v, want id-mismatch naming [Laceration]", is)
	}
	// Spec 9.1: {key, errors}.
	if !strings.Contains(raw, `"errors"`) || strings.Contains(raw, `"issues"`) {
		t.Errorf("a rejection should be {key, errors}:\n%s", raw)
	}
	// The rejected unit keeps its lease: its owner sends the fix.
	item, _ := pendingItem(t, st, batchKey0)
	if item.Attempts != 1 || item.Lease == nil || item.Lease.Batch != id {
		t.Errorf("pending item = %+v, want attempts 1 and the lease of %s", item, id)
	}

	// Again, as text. Unit 1 was accepted and is not in this file; it is not
	// "not submitted".
	writeOut(t, root, id, batch.OutUnit{Key: batchKey0, ZH: "還是沒有"})
	ctx, out := testCtx(t, root, "", false)
	err = runBatchSubmit(ctx, []string{id})
	if !errors.Is(err, cli.ErrFindings) {
		t.Fatalf("text run: err = %v, want cli.ErrFindings", err)
	}
	for _, want := range []string{"1 rejected", batchKey0, "id-mismatch", "0 notes"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("text output lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out.String(), "not submitted (") {
		t.Errorf("the accepted unit is listed as not submitted:\n%s", out)
	}
	if item, _ := pendingItem(t, st, batchKey0); item.Attempts != 2 {
		t.Errorf("attempts = %d, want 2", item.Attempts)
	}

	// The whole file again, with the fix: nothing is rejected.
	writeOut(t, root, id,
		batch.OutUnit{Key: batchKey0, ZH: "賦予5級[Laceration]威力"},
		batch.OutUnit{Key: batchKey1, ZH: "有人說了話"},
	)
	got, raw, err = runSubmitJSON(t, root, id)
	if err != nil {
		t.Fatalf("resend: %v\n%s", err, raw)
	}
	if len(got.Accepted) != 2 || len(got.Rejected) != 0 || len(got.Released) != 0 {
		t.Errorf("resend output = %s, want both accepted and nothing released", raw)
	}
	if pending, _ := st.LoadPending(); len(pending) != 0 {
		t.Errorf("pending = %+v, want empty", pending)
	}
}

func TestBatchNextKeepsBatchIDsUnique(t *testing.T) {
	root, _ := batchRepo(t)
	first, _, err := runNextJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	// Submit nothing: both units are free again.
	writeOut(t, root, *first.Batch)
	if got, raw, err := runSubmitJSON(t, root, *first.Batch); err != nil || len(got.Released) != 2 {
		t.Fatalf("empty submit: %v\n%s", err, raw)
	}

	// Same second, same first unit: the id would be the same, and the file of
	// the first batch is still there.
	second, _, err := runNextJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	if want := *first.Batch + "-2"; *second.Batch != want {
		t.Fatalf("second id = %s, want %s", *second.Batch, want)
	}
	for _, id := range []string{*first.Batch, *second.Batch} {
		if _, err := os.Stat(filepath.Join(root, "work", "batches", id+".json")); err != nil {
			t.Errorf("batch file of %s: %v", id, err)
		}
	}
	// The first file was not overwritten.
	var b batch.Batch
	data, err := os.ReadFile(filepath.Join(root, "work", "batches", *first.Batch+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &b); err != nil || b.ID != *first.Batch {
		t.Errorf("first batch file holds %q, %v", b.ID, err)
	}

	// The suffixed id can be submitted.
	writeOut(t, root, *second.Batch,
		batch.OutUnit{Key: batchKey0, ZH: "賦予5級[Laceration]威力"},
		batch.OutUnit{Key: batchKey1, ZH: "有人說了話"},
	)
	if got, raw, err := runSubmitJSON(t, root, *second.Batch); err != nil || len(got.Accepted) != 2 {
		t.Errorf("submit %s: %v\n%s", *second.Batch, err, raw)
	}
}

// A work/batches that cannot be looked into is an error, not an id in use: the
// search for a free id under the lock must end.
func TestBatchNextFailsWhenBatchFilesCannotBeChecked(t *testing.T) {
	root, st := batchRepo(t)
	writeTestFile(t, root, "work/batches", "not a directory")
	_, _, err := runNextJSON(t, root)
	if err == nil || isUserError(err) {
		t.Fatalf("err = %v, want a system error", err)
	}
	pending, err := st.LoadPending()
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range pending {
		if it.Lease != nil {
			t.Errorf("item %s was leased by a batch that failed: %+v", it.Key, it.Lease)
		}
	}
}

// A record made while the rules have changes that are not committed says so.
func TestBatchSubmitMarksDirtyRules(t *testing.T) {
	root, st := batchRepo(t)
	next, _, err := runNextJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, "guides/base.md", "# 未提交的指引\n")
	writeOut(t, root, *next.Batch, batch.OutUnit{Key: batchKey1, ZH: "有人說了話"})
	if _, raw, err := runSubmitJSON(t, root, *next.Batch); err != nil {
		t.Fatalf("submit: %v\n%s", err, raw)
	}
	recs, err := st.LoadRecords()
	if err != nil {
		t.Fatal(err)
	}
	if want := runGitIn(t, root, "rev-parse", "HEAD") + "+dirty"; recs[batchKey1].Rules != want {
		t.Errorf("record rules = %q, want %q", recs[batchKey1].Rules, want)
	}
}

func pendingItem(t *testing.T, st *store.Store, key string) (store.PendingItem, bool) {
	t.Helper()
	items, err := st.LoadPending()
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if it.Key == key {
			return it, true
		}
	}
	return store.PendingItem{}, false
}

func TestBatchSubmitUserErrors(t *testing.T) {
	root, _ := batchRepo(t)
	next, _, err := runNextJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	id := *next.Batch

	tests := []struct {
		name string
		args []string
		prep func()
		want string
	}{
		{"no id", nil, nil, "batch id"},
		{"two ids", []string{id, id}, nil, "unexpected"},
		{"id that leaves the directory", []string{"../x"}, nil, "invalid batch id"},
		{"unknown batch", []string{"20250101T000000-00000000"}, nil, "does not exist"},
		{"no out file", []string{id}, nil, ".out.json"},
		{"out is not a list", []string{id}, func() {
			writeTestFile(t, root, "work/batches/"+id+".out.json", `{"units": []}`)
		}, "not valid"},
		{"out has an unknown field", []string{id}, func() {
			writeTestFile(t, root, "work/batches/"+id+".out.json", `[{"key":"k","translation":"x"}]`)
		}, "not valid"},
		{"out is not JSON", []string{id}, func() {
			writeTestFile(t, root, "work/batches/"+id+".out.json", `[{"key":`)
		}, "not valid"},
		{"a key that is no unit key", []string{id}, func() {
			writeOut(t, root, id, batch.OutUnit{Key: "no-separator", ZH: "x", Note: "a note on it"})
		}, "invalid key"},
		{"batch file of another id", []string{id}, func() {
			writeTestFile(t, root, "work/batches/"+id+".out.json", `[]`)
			data, _ := os.ReadFile(filepath.Join(root, "work", "batches", id+".json"))
			other := "20250101T000000-00000000"
			writeTestFile(t, root, "work/batches/"+id+".json", strings.Replace(string(data), id, other, 1))
		}, "another batch"},
		// Last: it unsets LCT_MODEL for the rest of the test.
		{"no model", []string{id}, func() { t.Setenv("LCT_MODEL", "") }, "LCT_MODEL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prep != nil {
				tt.prep()
			}
			_, _, err := runSubmitJSON(t, root, tt.args...)
			if !isUserError(err) || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want a user error containing %q", err, tt.want)
			}
		})
	}
}

func TestPending(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "lct.yaml", batchYAML)
	st := store.Open(filepath.Join(root, ".work", "tw"))
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)
	if err := st.SavePending([]store.PendingItem{
		{Key: "A.json|/dataList/0/content", Reason: "new"},
		{Key: "A.json|/dataList/1/content", Reason: "new", Lease: &store.Lease{Batch: "b", Until: future}},
		{Key: "A.json|/dataList/2/content", Reason: "kr-changed", Lease: &store.Lease{Batch: "b", Until: past}},
		{Key: "B.json|/dataList/0/content", Reason: "kr-changed", Attempts: 3, Flags: []string{store.FlagNeedsReview}},
		{Key: "B.json|/dataList/1/content", Reason: "paratranz-repair"},
	}); err != nil {
		t.Fatal(err)
	}

	ctx, out := testCtx(t, root, "", true)
	if err := runPending(ctx, []string{"--count"}); err != nil {
		t.Fatal(err)
	}
	var count struct {
		Total       int            `json:"total"`
		ByReason    map[string]int `json:"by_reason"`
		Leased      int            `json:"leased"`
		NeedsReview int            `json:"needs_review"`
		Available   int            `json:"available"`
	}
	if err := json.Unmarshal(out.Bytes(), &count); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if count.Total != 5 || count.Leased != 1 || count.NeedsReview != 1 || count.Available != 3 ||
		count.ByReason["new"] != 2 || count.ByReason["kr-changed"] != 2 || count.ByReason["paratranz-repair"] != 1 {
		t.Errorf("count = %+v, want 5 pending, 1 leased (the other lease ran out), 1 needs_review, 3 available", count)
	}

	ctx, out = testCtx(t, root, "", false)
	if err := runPending(ctx, []string{"--count"}); err != nil {
		t.Fatal(err)
	}
	// One number: the pending units that are not waiting for a person (the
	// leased one counts).
	if out.String() != "4\n" {
		t.Errorf("text count = %q, want \"4\\n\"", out)
	}

	ctx, out = testCtx(t, root, "", true)
	if err := runPending(ctx, nil); err != nil {
		t.Fatal(err)
	}
	var list []store.PendingItem
	if err := json.Unmarshal(out.Bytes(), &list); err != nil || len(list) != 5 {
		t.Errorf("list = %d items, %v\n%s", len(list), err, out)
	}
	ctx, out = testCtx(t, root, "", false)
	if err := runPending(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(out.String(), "\n"); got != 5 || !strings.Contains(out.String(), "B.json|/dataList/0/content") {
		t.Errorf("text list should have a line per item:\n%s", out)
	}

	if err := runPending(ctx, []string{"extra"}); !isUserError(err) {
		t.Errorf("stray argument: err = %v, want a user error", err)
	}
}

func TestPendingEmptyStore(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "lct.yaml", batchYAML)
	ctx, out := testCtx(t, root, "", true)
	if err := runPending(ctx, []string{"--count"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"total": 0`) || !strings.Contains(out.String(), `"by_reason": {}`) || !strings.Contains(out.String(), `"available": 0`) {
		t.Errorf("count of a store without state:\n%s", out)
	}
	ctx, out = testCtx(t, root, "", true)
	if err := runPending(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "[]" {
		t.Errorf("list of a store without state = %q, want []", out)
	}
}

const (
	statusKey = "Status.json|/dataList/0/content"
	statusKR  = "대상에게 파열 2레벨 부여"
)

// keywordRepo is batchRepo with a unit that names a status in Korean, without
// an [ID], and the official name of that status, translated, in the store: the
// only pending unit is the one with the status.
func keywordRepo(t *testing.T) (root string, st *store.Store) {
	t.Helper()
	root, st = batchRepo(t)
	rawWT := filepath.Join(root, ".work", "raw")
	writeTestFile(t, rawWT, "kr/KR_Status.json", `{"dataList":[{"id":1,"content":"`+statusKR+`"}]}`)
	writeTestFile(t, rawWT, "kr/KR_Bufs.json", `{"dataList":[{"id":"Burst","name":"파열"}]}`)
	const nameKey = "Bufs.json|/dataList/0/name"
	if err := st.SaveRecords(map[string]store.Record{
		nameKey: {Key: nameKey, KRSha: units.KRSha("파열"), ZH: "破裂", Origin: store.OriginParatranz, Since: "raw-20261029"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.SavePending([]store.PendingItem{{Key: statusKey, Reason: "new"}}); err != nil {
		t.Fatal(err)
	}
	return root, st
}

// The name of a status that follows a count gets the half-width space the game
// needs to show the effect text.
func TestBatchSubmitAddsTheSpaceAfterKeywordNames(t *testing.T) {
	root, st := keywordRepo(t)
	next, _, err := runNextJSON(t, root)
	if err != nil || next.Batch == nil {
		t.Fatalf("batch next = %+v, %v", next, err)
	}
	writeOut(t, root, *next.Batch, batch.OutUnit{Key: statusKey, ZH: "使目標增加2級破裂強度"})
	got, raw, err := runSubmitJSON(t, root, *next.Batch)
	if err != nil || !slices.Equal(got.Accepted, []string{statusKey}) {
		t.Fatalf("submit: %v\n%s", err, raw)
	}
	recs, err := st.LoadRecords()
	if err != nil {
		t.Fatal(err)
	}
	if rec := recs[statusKey]; rec.ZH != "使目標增加2級破裂 強度" {
		t.Errorf("record ZH = %q, want the space after 破裂", rec.ZH)
	}
}
