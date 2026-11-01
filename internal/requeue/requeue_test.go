package requeue

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/batch"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

const (
	rawTag    = "raw-20261029"
	hint      = "新規則：稱呼改用「您」"
	storyFile = "StoryData/S001A.json"
	bgmFile   = "BgmLyrics/X.json"
	otherFile = "Other.json"
)

var frozen = []string{"BgmLyrics/**"}

func writeFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// openRaw writes the KR files (file name to JSON text) of a raw snapshot,
// with empty EN and JP packs, and opens it.
func openRaw(t *testing.T, kr map[string]string) pack.Set {
	t.Helper()
	raw := t.TempDir()
	for _, lang := range []string{"kr", "en", "jp"} {
		if err := os.MkdirAll(filepath.Join(raw, lang), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for file, text := range kr {
		dir, name := path.Split(file)
		writeFile(t, filepath.Join(raw, "kr", filepath.FromSlash(dir), "KR_"+name), text)
	}
	set, err := pack.OpenRaw(raw)
	if err != nil {
		t.Fatal(err)
	}
	return set
}

// dataList is a KR file of n lines, "대사 0" ... "대사 n-1".
func dataList(n int) string {
	s := `{"dataList":[`
	for i := 0; i < n; i++ {
		if i > 0 {
			s += ","
		}
		s += fmt.Sprintf(`{"id":%d,"content":"대사 %d"}`, i, i)
	}
	return s + `]}`
}

func key(file string, i int) string { return fmt.Sprintf("%s|/dataList/%d/content", file, i) }

func rec(key, zh, origin string) store.Record {
	return store.Record{Key: key, KRSha: units.KRSha("대사"), ZH: zh, Origin: origin, Since: rawTag}
}

func newStore(t *testing.T) *store.Store {
	t.Helper()
	return store.Open(filepath.Join(t.TempDir(), "tw"))
}

func saveRecords(t *testing.T, st *store.Store, recs ...store.Record) {
	t.Helper()
	m := map[string]store.Record{}
	for _, r := range recs {
		m[r.Key] = r
	}
	if err := st.SaveRecords(m); err != nil {
		t.Fatal(err)
	}
}

func savePending(t *testing.T, st *store.Store, items ...store.PendingItem) {
	t.Helper()
	if err := st.SavePending(items); err != nil {
		t.Fatal(err)
	}
}

func loadRecords(t *testing.T, st *store.Store) map[string]store.Record {
	t.Helper()
	recs, err := st.LoadRecords()
	if err != nil {
		t.Fatal(err)
	}
	return recs
}

func loadPending(t *testing.T, st *store.Store) []store.PendingItem {
	t.Helper()
	items, err := st.LoadPending()
	if err != nil {
		t.Fatal(err)
	}
	return items
}

func pendingByKey(t *testing.T, st *store.Store) map[string]store.PendingItem {
	t.Helper()
	m := map[string]store.PendingItem{}
	for _, it := range loadPending(t, st) {
		m[it.Key] = it
	}
	return m
}

func mustApply(t *testing.T, st *store.Store, kr *pack.Pack, sel Selector) *Result {
	t.Helper()
	res, err := Apply(st, kr, frozen, sel, hint)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	return res
}

func checkResult(t *testing.T, got *Result, want Result) {
	t.Helper()
	if want.MissingKeys == nil {
		want.MissingKeys = []string{}
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("result = %#v, want %#v", *got, want)
	}
}

// The rules of the tests that cut batches: one family, no characters.
func testRules(t *testing.T) *rules.Rules {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "guides/routing.yaml"), "families:\n  - id: other\n    match: []\n    guide: guides/families.md\npriority:\n  - other\n")
	writeFile(t, filepath.Join(root, "glossary/terms.yaml"), "- kr: 위력\n  zh: 威力\n  status: approved\n")
	writeFile(t, filepath.Join(root, "guides/characters/.keep"), "")
	r, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestRequeueByFile(t *testing.T) {
	set := openRaw(t, map[string]string{storyFile: dataList(4), bgmFile: dataList(1), otherFile: dataList(1)})
	st := newStore(t)
	saveRecords(t, st,
		rec(key(storyFile, 0), "譯文 0", store.OriginClaude),
		rec(key(storyFile, 1), "譯文 1", store.OriginClaude),
		rec(key(storyFile, 2), "ID_01", store.OriginCopy),
		rec(key(bgmFile, 0), "歌詞", store.OriginClaude),
		rec(key(otherFile, 0), "其他", store.OriginClaude),
	)
	before := loadRecords(t, st)

	res := mustApply(t, st, set.KR, Selector{Files: []string{storyFile}})
	checkResult(t, res, Result{Requeued: 2, SkippedCopy: 1})

	got := loadPending(t, st)
	want := []store.PendingItem{
		{Key: key(storyFile, 0), Reason: "requeue", PrevZH: "譯文 0", Hint: hint},
		{Key: key(storyFile, 1), Reason: "requeue", PrevZH: "譯文 1", Hint: hint},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("pending = %+v, want %+v", got, want)
	}
	if after := loadRecords(t, st); !reflect.DeepEqual(after, before) {
		t.Errorf("records changed:\n got  %+v\n want %+v", after, before)
	}
}

func TestRequeueKeysAndFilesAreAUnion(t *testing.T) {
	set := openRaw(t, map[string]string{storyFile: dataList(2), "StoryData/S002A.json": dataList(1), otherFile: dataList(2)})
	st := newStore(t)
	saveRecords(t, st,
		rec(key(storyFile, 0), "a", store.OriginClaude),
		rec(key(storyFile, 1), "b", store.OriginClaude),
		rec(key("StoryData/S002A.json", 0), "c", store.OriginClaude),
		rec(key(otherFile, 0), "d", store.OriginClaude),
		rec(key(otherFile, 1), "e", store.OriginClaude),
	)
	// A glob for the stories, and one key of another file.
	res := mustApply(t, st, set.KR, Selector{Files: []string{"StoryData/S*.json"}, Keys: []string{key(otherFile, 1), key(otherFile, 1)}})
	checkResult(t, res, Result{Requeued: 4})
	got := pendingByKey(t, st)
	for _, k := range []string{key(storyFile, 0), key(storyFile, 1), key("StoryData/S002A.json", 0), key(otherFile, 1)} {
		if _, ok := got[k]; !ok {
			t.Errorf("%s is not pending", k)
		}
	}
	if _, ok := got[key(otherFile, 0)]; ok || len(got) != 4 {
		t.Errorf("pending = %v, want the stories and Other.json line 1 only", got)
	}
}

func TestRequeueOriginFilter(t *testing.T) {
	set := openRaw(t, map[string]string{otherFile: dataList(5), storyFile: dataList(1)})
	seed := func(t *testing.T) *store.Store {
		st := newStore(t)
		saveRecords(t, st,
			rec(key(otherFile, 0), "p", store.OriginParatranz),
			rec(key(otherFile, 1), "c", store.OriginClaude),
			rec(key(otherFile, 2), "h", store.OriginHuman),
			rec(key(otherFile, 3), "f", store.OriginFallback),
			rec(key(otherFile, 4), "ID", store.OriginCopy),
			rec(key(storyFile, 0), "s", store.OriginClaude),
		)
		return st
	}
	pendingKeys := func(t *testing.T, st *store.Store) []string {
		var ks []string
		for _, it := range loadPending(t, st) {
			ks = append(ks, it.Key)
		}
		return ks
	}

	t.Run("default origins leave out copy", func(t *testing.T) {
		st := seed(t)
		res := mustApply(t, st, set.KR, Selector{Files: []string{otherFile}})
		checkResult(t, res, Result{Requeued: 4, SkippedCopy: 1})
	})
	t.Run("one origin in a file", func(t *testing.T) {
		st := seed(t)
		res := mustApply(t, st, set.KR, Selector{Files: []string{otherFile}, Origins: []string{"claude"}})
		checkResult(t, res, Result{Requeued: 1})
		if got := pendingKeys(t, st); !slices.Equal(got, []string{key(otherFile, 1)}) {
			t.Errorf("pending = %v", got)
		}
	})
	t.Run("origins alone select across files", func(t *testing.T) {
		st := seed(t)
		res := mustApply(t, st, set.KR, Selector{Origins: []string{"claude", "human"}})
		checkResult(t, res, Result{Requeued: 3})
		if got := pendingKeys(t, st); !slices.Equal(got, []string{key(otherFile, 1), key(otherFile, 2), key(storyFile, 0)}) {
			t.Errorf("pending = %v", got)
		}
	})
	t.Run("origin filters keys too", func(t *testing.T) {
		st := seed(t)
		res := mustApply(t, st, set.KR, Selector{Keys: []string{key(otherFile, 0), key(otherFile, 1)}, Origins: []string{"claude"}})
		checkResult(t, res, Result{Requeued: 1})
	})
	t.Run("origin also leaves out a key without a record", func(t *testing.T) {
		st := seed(t)
		noRecord := key(otherFile, 9)
		res := mustApply(t, st, set.KR, Selector{Keys: []string{noRecord, key(otherFile, 1)}, Origins: []string{"claude"}})
		checkResult(t, res, Result{Requeued: 1})
		// Without Origins the same key is reported.
		res = mustApply(t, st, set.KR, Selector{Keys: []string{noRecord}})
		checkResult(t, res, Result{Missing: 1, MissingKeys: []string{noRecord}})
	})
	t.Run("a key of a copy record", func(t *testing.T) {
		st := seed(t)
		res := mustApply(t, st, set.KR, Selector{Keys: []string{key(otherFile, 4)}})
		checkResult(t, res, Result{SkippedCopy: 1})
		if got := loadPending(t, st); len(got) != 0 {
			t.Errorf("pending = %+v, want none", got)
		}
	})
	t.Run("copy asked for explicitly is still skipped", func(t *testing.T) {
		st := seed(t)
		res := mustApply(t, st, set.KR, Selector{Origins: []string{"copy"}})
		checkResult(t, res, Result{SkippedCopy: 1})
	})
}

func TestRequeueSkipsFrozenAndMissing(t *testing.T) {
	set := openRaw(t, map[string]string{storyFile: dataList(2), bgmFile: dataList(1)})
	gone := key(storyFile, 9)       // the pointer left KR
	goneFile := key("Gone.json", 0) // the whole file left KR
	noRecord := key(storyFile, 1)   // in KR, but never recorded and not pending
	st := newStore(t)
	saveRecords(t, st,
		rec(key(storyFile, 0), "a", store.OriginClaude),
		rec(key(bgmFile, 0), "歌詞", store.OriginClaude),
		rec(gone, "x", store.OriginClaude),
		rec(goneFile, "y", store.OriginParatranz),
	)

	res := mustApply(t, st, set.KR, Selector{Files: []string{"BgmLyrics/**"}})
	checkResult(t, res, Result{SkippedFrozen: 1})
	if got := loadPending(t, st); len(got) != 0 {
		t.Fatalf("a frozen file was requeued: %+v", got)
	}

	res = mustApply(t, st, set.KR, Selector{Keys: []string{gone, goneFile, noRecord, key(bgmFile, 0)}})
	wantMissing := []string{gone, goneFile, noRecord}
	slices.Sort(wantMissing)
	checkResult(t, res, Result{Missing: 3, MissingKeys: wantMissing, SkippedFrozen: 1})
	if got := loadPending(t, st); len(got) != 0 {
		t.Errorf("a key that left KR was requeued: %+v", got)
	}
}

func TestRequeueNeedsReview(t *testing.T) {
	set := openRaw(t, map[string]string{otherFile: dataList(3)})
	twRoot := filepath.Join(t.TempDir(), "tw")
	st := store.Open(twRoot)
	k0, k1, k2 := key(otherFile, 0), key(otherFile, 1), key(otherFile, 2)
	fallback := rec(k0, "대사 0", store.OriginFallback)
	fallback.Flags = []string{store.FlagNeedsReview}
	saveRecords(t, st, fallback)
	savePending(t, st,
		store.PendingItem{Key: k0, Reason: "new", Attempts: 3, Flags: []string{store.FlagNeedsReview, "other"},
			Lease: &store.Lease{Batch: "20261101T000000-aaaaaaaa", Until: time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)}},
		store.PendingItem{Key: k1, Reason: "new", Attempts: 1},
		store.PendingItem{Key: k2, Reason: "kr-changed", PrevKR: "old", PrevZH: "舊", Attempts: 3, Flags: []string{store.FlagNeedsReview}},
	)

	res := mustApply(t, st, set.KR, Selector{NeedsReview: true, Keys: []string{k0}})
	checkResult(t, res, Result{Reset: 1})

	got := pendingByKey(t, st)
	want := store.PendingItem{Key: k0, Reason: "new", Hint: hint, Flags: []string{"other"}}
	if !reflect.DeepEqual(got[k0], want) {
		t.Errorf("reset item = %+v, want %+v", got[k0], want)
	}
	if got[k1].Attempts != 1 || got[k1].Hint != "" {
		t.Errorf("an item outside the selection changed: %+v", got[k1])
	}
	if !slices.Contains(got[k2].Flags, store.FlagNeedsReview) || got[k2].Attempts != 3 {
		t.Errorf("an item outside the selection was reset: %+v", got[k2])
	}
	if recs := loadRecords(t, st); !reflect.DeepEqual(recs[k0], fallback) {
		t.Errorf("the fallback record changed: %+v", recs[k0])
	}

	// All the rest of them, without a scope.
	res = mustApply(t, st, set.KR, Selector{NeedsReview: true})
	checkResult(t, res, Result{Reset: 1})
	got = pendingByKey(t, st)
	if len(got[k2].Flags) != 0 || got[k2].Attempts != 0 || got[k2].Hint != hint || got[k2].PrevZH != "舊" {
		t.Errorf("item 2 = %+v, want it reset with its previous text kept", got[k2])
	}

	// Next now hands the units out, with the hint.
	env := batch.Env{Store: store.Open(twRoot), Src: set, Rules: testRules(t), RawTag: rawTag, Now: time.Date(2026, 11, 1, 3, 4, 5, 0, time.UTC)}
	b, err := batch.Next(env, batch.NextOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if b == nil || len(b.Units) != 3 {
		t.Fatalf("batch = %+v, want 3 units", b)
	}
	for _, u := range b.Units {
		wantHint := hint
		if u.Key == k1 {
			wantHint = ""
		}
		if u.Hint != wantHint {
			t.Errorf("unit %s hint = %q, want %q", u.Key, u.Hint, wantHint)
		}
	}
}

func TestRequeueLeased(t *testing.T) {
	set := openRaw(t, map[string]string{otherFile: dataList(4), storyFile: dataList(1)})
	st := newStore(t)
	live := &store.Lease{Batch: "20261101T000000-aaaaaaaa", Until: time.Now().Add(time.Hour)}
	dead := &store.Lease{Batch: "20261031T000000-bbbbbbbb", Until: time.Now().Add(-time.Hour)}
	savePending(t, st,
		store.PendingItem{Key: key(otherFile, 0), Reason: "new", Attempts: 2, Lease: live},
		store.PendingItem{Key: key(otherFile, 1), Reason: "new", Attempts: 1, Lease: dead},
		store.PendingItem{Key: key(otherFile, 2), Reason: "new"},
		store.PendingItem{Key: key(otherFile, 3), Reason: "new", Lease: live, Hint: "old"},
		store.PendingItem{Key: key(storyFile, 0), Reason: "new", Lease: live},
	)

	res := mustApply(t, st, set.KR, Selector{Leased: true, Files: []string{otherFile}})
	checkResult(t, res, Result{Released: 3})
	got := pendingByKey(t, st)
	if got[key(otherFile, 0)].Lease != nil || got[key(otherFile, 0)].Attempts != 2 {
		t.Errorf("a live lease: %+v, want it cleared with the attempts kept", got[key(otherFile, 0)])
	}
	if got[key(otherFile, 1)].Lease != nil || got[key(otherFile, 1)].Attempts != 1 {
		t.Errorf("an expired lease: %+v, want it cleared with the attempts kept", got[key(otherFile, 1)])
	}
	if got[key(otherFile, 3)].Hint != "old" {
		t.Errorf("a released item's hint = %q, want it left alone", got[key(otherFile, 3)].Hint)
	}
	if got[key(storyFile, 0)].Lease == nil {
		t.Error("the lease of a file outside the selection was cleared")
	}

	// Without a scope, every lease goes.
	res = mustApply(t, st, set.KR, Selector{Leased: true})
	checkResult(t, res, Result{Released: 1})
	for _, it := range loadPending(t, st) {
		if it.Lease != nil {
			t.Errorf("%s is still leased", it.Key)
		}
	}
	res = mustApply(t, st, set.KR, Selector{Leased: true})
	checkResult(t, res, Result{})
}

func TestRequeueAlreadyPending(t *testing.T) {
	set := openRaw(t, map[string]string{otherFile: dataList(2)})
	st := newStore(t)
	k := key(otherFile, 0)
	saveRecords(t, st, rec(k, "舊譯", store.OriginClaude), rec(key(otherFile, 1), "舊譯 1", store.OriginClaude))
	lease := &store.Lease{Batch: "20261101T000000-aaaaaaaa", Until: time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)}
	savePending(t, st, store.PendingItem{Key: k, Reason: "kr-changed", PrevKR: "舊原文", PrevZH: "前譯", Draft: "草稿", Attempts: 1, Lease: lease})

	res := mustApply(t, st, set.KR, Selector{Keys: []string{k}})
	checkResult(t, res, Result{AlreadyPending: 1})
	want := store.PendingItem{Key: k, Reason: "kr-changed", PrevKR: "舊原文", PrevZH: "前譯", Draft: "草稿", Hint: hint, Attempts: 1, Lease: lease}
	if got := loadPending(t, st); !reflect.DeepEqual(got, []store.PendingItem{want}) {
		t.Errorf("pending = %+v, want only the hint added: %+v", got, want)
	}

	// A second reason replaces the first; the other line is requeued.
	res, err := Apply(st, set.KR, frozen, Selector{Files: []string{otherFile}}, "另一個理由")
	if err != nil {
		t.Fatal(err)
	}
	checkResult(t, res, Result{AlreadyPending: 1, Requeued: 1})
	got := pendingByKey(t, st)
	if len(got) != 2 || got[k].Hint != "另一個理由" || got[key(otherFile, 1)].Reason != "requeue" {
		t.Errorf("pending = %+v", got)
	}
}

// --needs-review and --leased work on the pending items only. --key, --file
// and --origin then just narrow which pending items they reset or release
// (--origin by the origin of the item's record); no record is requeued.
func TestRequeueSelectorsCombine(t *testing.T) {
	set := openRaw(t, map[string]string{otherFile: dataList(3), storyFile: dataList(3)})
	review := []string{store.FlagNeedsReview}
	lease := &store.Lease{Batch: "20261101T000000-aaaaaaaa", Until: time.Now().Add(time.Hour)}
	seed := func(t *testing.T) *store.Store {
		st := newStore(t)
		fb := rec(key(otherFile, 0), "f", store.OriginFallback)
		fb.Flags = review
		saveRecords(t, st, fb,
			rec(key(otherFile, 1), "c", store.OriginClaude),
			rec(key(otherFile, 2), "c2", store.OriginClaude),
			rec(key(storyFile, 0), "s0", store.OriginClaude),
			rec(key(storyFile, 1), "s1", store.OriginHuman),
		)
		savePending(t, st,
			store.PendingItem{Key: key(otherFile, 0), Reason: "new", Attempts: 3, Flags: review},
			store.PendingItem{Key: key(otherFile, 2), Reason: "new", Lease: lease},
			store.PendingItem{Key: key(storyFile, 1), Reason: "new", Lease: lease},
			store.PendingItem{Key: key(storyFile, 2), Reason: "new", Attempts: 3, Flags: review},
		)
		return st
	}

	// noNewItems fails when a record was sent back: the pending list must
	// still be the four items of seed.
	noNewItems := func(t *testing.T, st *store.Store) map[string]store.PendingItem {
		t.Helper()
		got := pendingByKey(t, st)
		if len(got) != 4 {
			t.Errorf("pending has %d items, want the 4 it started with: %+v", len(got), got)
		}
		for k, it := range got {
			if it.Reason == "requeue" {
				t.Errorf("%s was requeued: %+v", k, it)
			}
		}
		return got
	}

	t.Run("a file narrows needs-review and no record is requeued", func(t *testing.T) {
		st := seed(t)
		res := mustApply(t, st, set.KR, Selector{NeedsReview: true, Files: []string{otherFile}})
		// otherFile|0 is reset; the claude record of |1 stays where it is, and
		// |2 is pending already but gets no hint; the story file is out of scope.
		checkResult(t, res, Result{Reset: 1})
		got := noNewItems(t, st)
		if len(got[key(otherFile, 0)].Flags) != 0 || got[key(otherFile, 0)].Hint != hint || got[key(storyFile, 2)].Attempts != 3 {
			t.Errorf("pending = %+v", got)
		}
		if got[key(otherFile, 2)].Hint != "" {
			t.Errorf("a pending unit that is not flagged got a hint: %+v", got[key(otherFile, 2)])
		}
	})
	t.Run("an origin narrows leased", func(t *testing.T) {
		st := seed(t)
		res := mustApply(t, st, set.KR, Selector{Leased: true, Origins: []string{"human"}})
		// storyFile|1 is the only leased item whose record is human; the other
		// records of that origin are not requeued.
		checkResult(t, res, Result{Released: 1})
		got := noNewItems(t, st)
		if got[key(storyFile, 1)].Lease != nil || got[key(otherFile, 2)].Lease == nil {
			t.Errorf("pending = %+v", got)
		}
	})
	t.Run("an origin leaves out a pending unit without a record", func(t *testing.T) {
		st := seed(t)
		savePending(t, st, store.PendingItem{Key: key(storyFile, 2), Reason: "new", Lease: lease}) // no record
		res := mustApply(t, st, set.KR, Selector{Leased: true, Origins: []string{"claude", "human"}})
		checkResult(t, res, Result{})
		if got := pendingByKey(t, st); got[key(storyFile, 2)].Lease == nil {
			t.Errorf("the lease of a unit without a record was cleared: %+v", got)
		}
	})
	t.Run("a key", func(t *testing.T) {
		st := seed(t)
		res := mustApply(t, st, set.KR, Selector{Leased: true, Keys: []string{key(otherFile, 2)}})
		checkResult(t, res, Result{Released: 1})
		if got := noNewItems(t, st); got[key(storyFile, 1)].Lease == nil {
			t.Errorf("the lease of another key was cleared: %+v", got)
		}
	})
	t.Run("a key that is no pending unit is not requeued, nor reported missing", func(t *testing.T) {
		st := seed(t)
		gone := key(storyFile, 9) // not in KR, no record, not pending
		res := mustApply(t, st, set.KR, Selector{NeedsReview: true, Keys: []string{key(otherFile, 1), gone}})
		checkResult(t, res, Result{})
		noNewItems(t, st)
		res = mustApply(t, st, set.KR, Selector{Leased: true, Files: []string{"Gone.json", "BgmLyrics/**"}})
		checkResult(t, res, Result{})
		noNewItems(t, st)
	})
	t.Run("both flags at once", func(t *testing.T) {
		st := seed(t)
		res := mustApply(t, st, set.KR, Selector{NeedsReview: true, Leased: true})
		checkResult(t, res, Result{Reset: 2, Released: 2})
		noNewItems(t, st)
	})
}

func TestRequeueRejectsBadSelectors(t *testing.T) {
	set := openRaw(t, map[string]string{otherFile: dataList(1)})
	st := newStore(t)
	saveRecords(t, st, rec(key(otherFile, 0), "a", store.OriginClaude))
	for name, sel := range map[string]Selector{
		"nothing selected":  {},
		"unknown origin":    {Origins: []string{"claude", "robot"}},
		"malformed key":     {Keys: []string{"no-separator"}},
		"malformed pattern": {Files: []string{"Story[Data"}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Apply(st, set.KR, frozen, sel, hint); err == nil {
				t.Error("Apply = nil error")
			}
			if err := sel.Validate(); err == nil {
				t.Error("Validate = nil error")
			}
		})
	}
	if _, err := Apply(st, nil, frozen, Selector{Leased: true}, hint); err == nil {
		t.Error("Apply without a KR pack = nil error")
	}
	if got := loadPending(t, st); len(got) != 0 {
		t.Errorf("a rejected call changed the pending list: %+v", got)
	}
}

// Requeue and Submit both read and write the records and the pending list;
// run side by side they must not lose each other's updates.
func TestRequeueOneLock(t *testing.T) {
	const total, old = 144, 48 // the first old lines are translated; the rest are new
	set := openRaw(t, map[string]string{otherFile: dataList(total)})
	twRoot := filepath.Join(t.TempDir(), "tw")
	st := store.Open(twRoot)
	var recs []store.Record
	var pending []store.PendingItem
	for i := 0; i < total; i++ {
		if i < old {
			recs = append(recs, rec(key(otherFile, i), fmt.Sprintf("舊譯 %d", i), store.OriginParatranz))
		} else {
			pending = append(pending, store.PendingItem{Key: key(otherFile, i), Reason: "new"})
		}
	}
	saveRecords(t, st, recs...)
	savePending(t, st, pending...)
	r := testRules(t)

	var (
		wg        sync.WaitGroup
		requeuing = make(chan struct{}) // closed when every requeue has returned
		errs      = make(chan error, 64)
		start     = make(chan struct{})
		entryRe   = regexp.MustCompile(`/dataList/(\d+)/`)
	)
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			env := batch.Env{Store: store.Open(twRoot), Src: set, Rules: r, RawTag: rawTag} // its own store, as another process would have
			<-start
			for {
				done := false
				select {
				case <-requeuing:
					done = true
				default:
				}
				b, err := batch.Next(env, batch.NextOptions{MaxUnits: 2})
				if err != nil {
					errs <- fmt.Errorf("Next: %w", err)
					return
				}
				if b == nil {
					if done {
						return
					}
					time.Sleep(time.Millisecond)
					continue
				}
				var out []batch.OutUnit
				for _, u := range b.Units {
					n, _ := strconv.Atoi(entryRe.FindStringSubmatch(u.Key)[1])
					// A requeued line comes with its reason and its old text,
					// a new one with neither.
					requeued := n < old
					if (u.Hint == hint) != requeued || (u.Hint != "" && !requeued) || (u.PrevZH != "") != requeued {
						errs <- fmt.Errorf("unit %s: hint %q, prev_zh %q", u.Key, u.Hint, u.PrevZH)
					}
					out = append(out, batch.OutUnit{Key: u.Key, ZH: fmt.Sprintf("譯文 %d", n)})
				}
				if _, err := batch.Submit(env, b, out, "test-model", 0); err != nil {
					errs <- fmt.Errorf("Submit: %w", err)
					return
				}
			}
		}()
	}

	var mu sync.Mutex
	var requeued, already, missing int
	var rq sync.WaitGroup
	for g := 0; g < 2; g++ { // two requeuers, each with its own store
		rq.Add(1)
		go func() {
			defer rq.Done()
			s := store.Open(twRoot)
			<-start
			for i := g; i < old; i += 2 {
				res, err := Apply(s, set.KR, frozen, Selector{Keys: []string{key(otherFile, i)}}, hint)
				if err != nil {
					errs <- fmt.Errorf("Apply: %w", err)
					return
				}
				mu.Lock()
				requeued += res.Requeued
				already += res.AlreadyPending
				missing += res.Missing
				mu.Unlock()
			}
		}()
	}
	close(start)
	rq.Wait()
	close(requeuing)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}

	if requeued != old || already != 0 || missing != 0 {
		t.Errorf("requeued %d, already pending %d, missing %d, want %d requeued and nothing else", requeued, already, missing, old)
	}
	if left := loadPending(t, st); len(left) != 0 {
		t.Errorf("%d units are still pending, such as %+v: a requeue was lost, or a translated unit came back", len(left), left[0])
	}
	got := loadRecords(t, st)
	if len(got) != total {
		t.Errorf("%d records, want %d", len(got), total)
	}
	for i := 0; i < total; i++ {
		k := key(otherFile, i)
		if r := got[k]; r.Origin != store.OriginClaude || r.ZH != fmt.Sprintf("譯文 %d", i) {
			t.Errorf("record %s = %+v, want the new translation", k, r)
			break
		}
	}
}

func TestRequeueResultJSONAndText(t *testing.T) {
	set := openRaw(t, map[string]string{otherFile: dataList(1)})
	res := mustApply(t, newStore(t), set.KR, Selector{Leased: true})
	data, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"requeued":0,"reset":0,"released":0,"already_pending":0,"skipped_frozen":0,"skipped_copy":0,"missing":0,"missing_keys":[]}`
	if string(data) != want {
		t.Errorf("result JSON = %s, want %s", data, want)
	}

	text := (&Result{Requeued: 2, Reset: 3, Released: 4, AlreadyPending: 5, SkippedFrozen: 6, SkippedCopy: 7, Missing: 1, MissingKeys: []string{"A.json|/x"}}).String()
	for _, part := range []string{"requeued 2", "reset 3", "released 4", "already pending 5", "6 frozen", "7 copy", "missing (gone from KR, or no record): 1", "A.json|/x"} {
		if !strings.Contains(text, part) {
			t.Errorf("text %q lacks %q", text, part)
		}
	}
}
