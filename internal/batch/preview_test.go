package batch

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// recOf makes a record of the translation zh of the KR text kr.
func recOf(k, kr, zh, origin string) store.Record {
	return store.Record{Key: k, KRSha: units.KRSha(kr), ZH: zh, Origin: origin, Since: testTag}
}

func saveRecs(t *testing.T, env Env, recs ...store.Record) {
	t.Helper()
	m := map[string]store.Record{}
	for _, r := range recs {
		m[r.Key] = r
	}
	if err := env.Store.SaveRecords(m); err != nil {
		t.Fatal(err)
	}
}

func newPreviewer(t *testing.T, env Env, frozen ...string) *Previewer {
	t.Helper()
	p, err := NewPreviewer(env, frozen)
	if err != nil {
		t.Fatalf("NewPreviewer: %v", err)
	}
	return p
}

func mustUnit(t *testing.T, p *Previewer, k string) *UnitPreview {
	t.Helper()
	pv, err := p.Unit(k)
	if err != nil {
		t.Fatalf("Unit(%s): %v", k, err)
	}
	return pv
}

// A preview is the batch Next cuts for the unit alone: same units, context,
// glossary, guides and keywords, only the id differs.
func TestPreviewMatchesNext(t *testing.T) {
	const file = "StoryData/S001A.json"
	env := newEnv(t, map[string]string{file: dataList(t, 6, "파우스트", line)}, nil, nil)
	saveRecs(t, env,
		recOf(key(file, 1), line(1), "第一行", store.OriginClaude),
		recOf(key(file, 2), line(2), "第二行", store.OriginClaude),
		recOf(key(file, 4), line(4), "第四行", store.OriginParatranz),
	)
	seed(t, env, store.PendingItem{Key: key(file, 3), Reason: "new"})

	// The preview first: Next writes a lease.
	pv := mustUnit(t, newPreviewer(t, env), key(file, 3))
	b := mustNext(t, env, NextOptions{MaxUnits: 1})
	if b == nil {
		t.Fatal("Next handed out nothing")
	}
	if len(b.Context) == 0 || !slices.Contains(b.Guides, "guides/characters/faust.md") {
		t.Fatalf("batch = %+v, want context and the guide of the speaker: the test checks nothing otherwise", b)
	}
	if pv.Batch.ID != PreviewID {
		t.Errorf("preview id = %q, want %q", pv.Batch.ID, PreviewID)
	}
	pv.Batch.ID = b.ID
	if !reflect.DeepEqual(pv.Batch, b) {
		t.Errorf("preview batch = %+v\nNext batch    = %+v", pv.Batch, b)
	}
}

func TestPreviewContextSkipsPendingNeighbours(t *testing.T) {
	const file = "StoryData/S001A.json"
	env := newEnv(t, map[string]string{file: dataList(t, 8, "파우스트", line)}, nil, nil)
	var recs []store.Record
	for _, i := range []int{1, 3, 5, 6} {
		recs = append(recs, recOf(key(file, i), line(i), "譯文", store.OriginClaude))
	}
	saveRecs(t, env, recs...)
	// Line 5 is translated but waits for a new translation.
	seed(t, env, store.PendingItem{Key: key(file, 4), Reason: "new"}, store.PendingItem{Key: key(file, 5), Reason: "requeue", PrevZH: "譯文"})

	pv := mustUnit(t, newPreviewer(t, env), key(file, 4))
	var got []string
	for _, c := range pv.Batch.Context {
		got = append(got, c.Key)
	}
	// Line 3 is next to the unit; 5 waits; 1 and 6 are out of reach(1).
	if want := []string{key(file, 3)}; !slices.Equal(got, want) {
		t.Errorf("context = %v, want %v", got, want)
	}
}

func TestPreviewItem(t *testing.T) {
	const file = "Other.json"
	env := newEnv(t, map[string]string{file: dataList(t, 6, "", line)}, nil, nil)
	saveRecs(t, env,
		recOf(key(file, 1), line(1), "舊譯", store.OriginClaude),
		recOf(key(file, 3), line(3), "3", store.OriginCopy),
	)
	asked := store.PendingItem{
		Key: key(file, 0), Reason: "kr-changed", PrevKR: "옛 대사", PrevZH: "舊句", Draft: "草稿", Hint: "用您",
		Refs: map[string]string{"參考": "參考文"},
	}
	seed(t, env, asked)
	p := newPreviewer(t, env)

	t.Run("pending: the real item", func(t *testing.T) {
		pv := mustUnit(t, p, key(file, 0))
		u := pv.Batch.Units[0]
		if !reflect.DeepEqual(pv.Pending, asked) || len(pv.Batch.Units) != 1 {
			t.Fatalf("pending = %+v, units %d", pv.Pending, len(pv.Batch.Units))
		}
		if u.Reason != "kr-changed" || u.PrevKR != "옛 대사" || u.PrevZH != "舊句" || u.Draft != "草稿" || u.Hint != "用您" || u.Refs["參考"] != "參考文" {
			t.Errorf("unit = %+v, want the fields of the pending item", u)
		}
		if !pv.State.Pending {
			t.Error("state is not pending")
		}
	})
	t.Run("translated: a requeue", func(t *testing.T) {
		pv := mustUnit(t, p, key(file, 1))
		u := pv.Batch.Units[0]
		if u.Reason != "requeue" || u.PrevZH != "舊譯" || u.Hint != "" {
			t.Errorf("unit = %+v, want a requeue of 舊譯 without a hint", u)
		}
		if pv.State.Pending || pv.Pending.Reason != "requeue" {
			t.Errorf("state %+v, pending %+v", pv.State, pv.Pending)
		}
	})
	t.Run("neither: new", func(t *testing.T) {
		pv := mustUnit(t, p, key(file, 2))
		if u := pv.Batch.Units[0]; u.Reason != ReasonNew || u.PrevZH != "" || u.Draft != "" {
			t.Errorf("unit = %+v, want a new unit", u)
		}
	})
	t.Run("copy", func(t *testing.T) {
		if _, err := p.Unit(key(file, 3)); !errors.Is(err, ErrCopy) {
			t.Errorf("err = %v, want ErrCopy", err)
		}
	})
	t.Run("not in KR", func(t *testing.T) {
		for _, k := range []string{key(file, 99), key("Gone.json", 0)} {
			if _, err := p.Unit(k); !errors.Is(err, ErrNotInKR) {
				t.Errorf("Unit(%s) err = %v, want ErrNotInKR", k, err)
			}
		}
		if _, err := p.File("Gone.json"); !errors.Is(err, ErrNotInKR) {
			t.Errorf("File err = %v, want ErrNotInKR", err)
		}
	})
	t.Run("File: document order, nil for a copy", func(t *testing.T) {
		all, err := p.File(file)
		if err != nil {
			t.Fatal(err)
		}
		if len(all) != 6 || all[3] != nil {
			t.Fatalf("File = %d entries, entry 3 %v", len(all), all[3])
		}
		for i, pv := range all {
			if i != 3 && (pv == nil || pv.Batch.Units[0].Key != key(file, i)) {
				t.Errorf("entry %d = %+v", i, pv)
			}
		}
	})
}

func TestPreviewState(t *testing.T) {
	const file, frozenFile = "Other.json", "Frozen.json"
	env := newEnv(t, map[string]string{file: dataList(t, 4, "", line), frozenFile: dataList(t, 1, "", line)}, nil, nil)
	env.RulesCommit = "abc1234"
	stuck := store.PendingItem{Key: key(file, 0), Reason: "new", Attempts: 3, LastError: "hangul: 譯文含諺文", Flags: []string{store.FlagNeedsReview}}
	// A lease that ran out is reported all the same.
	leased := store.PendingItem{Key: key(file, 1), Reason: "new", Lease: &store.Lease{Batch: "20260101T000000-aaaaaaaa", Until: testNow.Add(-time.Hour)}}
	seed(t, env, stuck, leased, store.PendingItem{Key: key(frozenFile, 0), Reason: "new"})
	p := newPreviewer(t, env, "Frozen*")

	pv := mustUnit(t, p, stuck.Key)
	want := PreviewState{Pending: true, NeedsReview: true, Attempts: 3, LastError: "hangul: 譯文含諺文", RulesCommit: "abc1234"}
	if pv.State != want || len(pv.Batch.Units) != 1 {
		t.Errorf("state = %+v, want %+v (and a batch)", pv.State, want)
	}
	pv = mustUnit(t, p, leased.Key)
	if pv.State.LeasedTo != "20260101T000000-aaaaaaaa" || pv.State.NeedsReview || len(pv.Batch.Units) != 1 {
		t.Errorf("state = %+v, want the lease and a batch", pv.State)
	}
	pv = mustUnit(t, p, key(frozenFile, 0))
	if !pv.State.Frozen || len(pv.Batch.Units) != 1 {
		t.Errorf("state = %+v, want frozen and a batch", pv.State)
	}
	pv = mustUnit(t, p, key(file, 2))
	if pv.State.Pending || pv.State.Frozen || pv.State.LeasedTo != "" {
		t.Errorf("state = %+v, want a plain unit", pv.State)
	}
}

func TestPreviewCheck(t *testing.T) {
	const file = "Other.json"
	const hard = "두 시즌 전" // 시즌 is check: hard
	const prov = "번개가 친다" // 번개 is a provisional term
	kr := []string{hard, hard, hard, "안녕", "안녕 둘", "안녕", "안녕", "안녕", prov}
	env := newEnv(t, map[string]string{file: dataList(t, len(kr), "", func(i int) string { return kr[i] })}, nil, nil)
	env.Rules = testRulesWith(t, "- kr: 시즌\n  zh: 賽季\n  status: approved\n  check: hard\n")
	env.ForeignNames = []string{"賽局"}
	env.Provisional = []rules.Term{{KR: "번개", ZH: "閃電", Status: rules.StatusApproved, Match: rules.MatchWord, Provisional: true}}

	rec := func(i int, zh, origin string) store.Record { return recOf(key(file, i), kr[i], zh, origin) }
	stale := rec(6, "你好", store.OriginClaude)
	stale.KRSha = units.KRSha("옛 대사")
	saveRecs(t, env,
		rec(0, "兩個賽局前", store.OriginClaude), // hard term missing
		rec(1, "兩個賽局前", store.OriginClaude), // ... but the note names it
		rec(2, "兩個賽局前", store.OriginClaude), // ... the note is for another KR
		rec(3, "你好", store.OriginClaude),
		rec(4, "你好賽局", store.OriginHuman), // only a foreign name
		rec(5, "hello", store.OriginFallback),
		stale,
		rec(8, "雷打下來", store.OriginClaude), // provisional term missing: never hard
	)
	if _, err := env.Store.AppendNotes([]store.Note{
		{Key: key(file, 1), KRSha: units.KRSha(hard), Note: "시즌：指賽局"},
		{Key: key(file, 2), KRSha: units.KRSha("옛 대사"), Note: "시즌：指賽局"},
		{Key: key(file, 1), KRSha: units.KRSha(hard), Note: "第二則"},
	}); err != nil {
		t.Fatal(err)
	}
	p := newPreviewer(t, env)

	verdict := func(i int) *Check {
		t.Helper()
		pv := mustUnit(t, p, key(file, i))
		if pv.Check == nil {
			t.Fatalf("unit %d: no check", i)
		}
		return pv.Check
	}
	if c := verdict(0); c.Verdict != "reject" || !hasCode(c.Issues, validate.CodeTermMissing) || c.StaleKR {
		t.Errorf("hard term missing: %+v", c)
	}
	pv := mustUnit(t, p, key(file, 1))
	if c := pv.Check; c.Verdict != "warn" {
		t.Errorf("note names the term: %+v, want warn", c)
	}
	if len(pv.Notes) != 2 || pv.Notes[0].Note != "시즌：指賽局" || pv.Notes[1].Note != "第二則" {
		t.Errorf("notes = %+v, want the two notes of the current KR in store order", pv.Notes)
	}
	pv = mustUnit(t, p, key(file, 2))
	if pv.Check.Verdict != "reject" || len(pv.Notes) != 0 || pv.Notes == nil {
		t.Errorf("note of another KR: %+v, notes %#v; want reject and empty notes", pv.Check, pv.Notes)
	}
	if c := verdict(3); c.Verdict != "accept" || len(c.Issues) != 0 || c.Issues == nil || c.ZH != "你好" || c.Origin != store.OriginClaude {
		t.Errorf("clean: %+v", c)
	}
	if c := verdict(4); c.Verdict != "warn" || !hasCode(c.Issues, CodeForeignName) || c.Origin != store.OriginHuman {
		t.Errorf("foreign name: %+v", c)
	}
	if c := verdict(5); c.Origin != store.OriginFallback {
		t.Errorf("fallback: %+v", c)
	}
	if c := verdict(6); !c.StaleKR {
		t.Errorf("record of an old KR: %+v, want stale_kr", c)
	}
	pv = mustUnit(t, p, key(file, 8))
	if pv.Check.Verdict == "reject" || !hasCode(pv.Check.Issues, validate.CodeTermMissing) {
		t.Errorf("provisional term missing: %+v, want a warning", pv.Check)
	}
	if !slices.ContainsFunc(pv.Batch.Glossary, func(t rules.Term) bool { return t.KR == "번개" && t.Provisional }) {
		t.Errorf("glossary = %+v, want the provisional term the check is held to", pv.Batch.Glossary)
	}
	if pv := mustUnit(t, p, key(file, 7)); pv.Check != nil || pv.Record != nil {
		t.Errorf("no record: check %+v, record %+v", pv.Check, pv.Record)
	}
}

func TestPreviewReadOnly(t *testing.T) {
	const file = "StoryData/S001A.json"
	twRoot := filepath.Join(t.TempDir(), "tw")
	env := newEnvIn(t, twRoot, map[string]string{file: dataList(t, 5, "파우스트", line)}, nil, nil)
	saveRecs(t, env, recOf(key(file, 0), line(0), "譯", store.OriginClaude))
	seed(t, env, store.PendingItem{Key: key(file, 1), Reason: "new"}, store.PendingItem{Key: key(file, 2), Reason: "new", Attempts: 1})
	if _, err := env.Store.AppendNotes([]store.Note{{Key: key(file, 0), KRSha: units.KRSha(line(0)), Note: "註"}}); err != nil {
		t.Fatal(err)
	}
	snapshot := func() map[string]string {
		out := map[string]string{}
		err := filepath.WalkDir(twRoot, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			b, err := os.ReadFile(p)
			out[p] = string(b)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	before := snapshot()
	if len(before) == 0 {
		t.Fatal("the store is empty: the test checks nothing")
	}
	p := newPreviewer(t, env)
	mustUnit(t, p, key(file, 1))
	mustUnit(t, p, key(file, 3))
	if _, err := p.File(file); err != nil {
		t.Fatal(err)
	}
	if after := snapshot(); !reflect.DeepEqual(before, after) {
		t.Error("the store changed")
	}
}

func TestPreviewFileFreshLoader(t *testing.T) {
	env := newEnv(t, map[string]string{"A.json": dataList(t, 2, "", line), "B.json": dataList(t, 2, "", line)}, nil, nil)
	p := newPreviewer(t, env)
	for _, f := range []string{"A.json", "B.json"} {
		if all, err := p.File(f); err != nil || len(all) != 2 {
			t.Fatalf("File(%s) = %d entries, %v", f, len(all), err)
		}
	}
	// The data of a file lives in a loader; the Previewer must not hold one.
	rt := reflect.TypeOf(*p)
	for i := 0; i < rt.NumField(); i++ {
		if s := rt.Field(i).Type.String(); strings.Contains(s, "loader") || strings.Contains(s, "fileData") {
			t.Errorf("Previewer.%s has type %s: it keeps file data between calls", rt.Field(i).Name, s)
		}
	}
}

// A text that names a keyword comes with the name this translation gives it,
// the same as in the batch Next cuts, and a second preview on the same
// Previewer (which keeps the keyword table) gives the same names.
func TestPreviewKeywords(t *testing.T) {
	const file = "Other.json"
	const kr = "[Bleed] 위력 5 부여"
	env := newEnv(t, map[string]string{
		file:        dataList(t, 2, "", func(int) string { return kr }),
		"Bufs.json": `{"dataList":[{"id":"Bleed","name":"출혈"}]}`,
	}, nil, nil)
	saveRecs(t, env, recOf("Bufs.json|/dataList/0/name", "출혈", "流血", store.OriginParatranz))
	seed(t, env, store.PendingItem{Key: key(file, 0), Reason: "new"})

	p := newPreviewer(t, env)
	first := mustUnit(t, p, key(file, 0))
	again := mustUnit(t, p, key(file, 0))
	other := mustUnit(t, p, key(file, 1)) // not pending: the table is shared
	b := mustNext(t, env, NextOptions{MaxUnits: 1})

	if len(b.Keywords) == 0 || b.Keywords["Bleed"] != "流血" {
		t.Fatalf("Next keywords = %v, want Bleed: the test checks nothing otherwise", b.Keywords)
	}
	if !reflect.DeepEqual(first.Batch.Keywords, b.Keywords) {
		t.Errorf("preview keywords = %v, Next keywords = %v", first.Batch.Keywords, b.Keywords)
	}
	if !reflect.DeepEqual(again.Batch.Keywords, first.Batch.Keywords) || !reflect.DeepEqual(other.Batch.Keywords, first.Batch.Keywords) {
		t.Errorf("keywords change between previews: %v, %v, %v", first.Batch.Keywords, again.Batch.Keywords, other.Batch.Keywords)
	}
}

// A key that cannot be split names no unit: the user's mistake, like a key
// that is not in KR.
func TestPreviewMalformedKey(t *testing.T) {
	env := newEnv(t, map[string]string{"Other.json": dataList(t, 1, "", line)}, nil, nil)
	_, err := newPreviewer(t, env).Unit("no-separator")
	if !errors.Is(err, ErrNotInKR) {
		t.Fatalf("err = %v, want ErrNotInKR", err)
	}
	_, _, want := units.SplitKey("no-separator")
	if want == nil || !strings.Contains(err.Error(), want.Error()) {
		t.Errorf("err = %q, want the message of SplitKey (%v)", err, want)
	}
}

// NewPreviewerWith takes the records the caller has read: the previews are
// those of a Previewer that reads them itself when the records are the
// store's, and they follow the records passed when those differ from the
// store's (a Previewer that ignored them and read the store would not).
func TestNewPreviewerWithRecords(t *testing.T) {
	const file = "StoryData/S001A.json"
	env := newEnv(t, map[string]string{file: dataList(t, 4, "파우스트", line)}, nil, nil)
	saveRecs(t, env,
		recOf(key(file, 1), line(1), "第一行", store.OriginClaude),
		recOf(key(file, 2), line(2), "第二行", store.OriginParatranz),
	)
	recs, err := env.Store.LoadRecords()
	if err != nil {
		t.Fatal(err)
	}
	own := newPreviewer(t, env)
	with, err := NewPreviewerWith(env, nil, recs)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		k := key(file, i)
		if got, want := mustUnit(t, with, k), mustUnit(t, own, k); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: with records = %+v\nown            = %+v", k, got, want)
		}
	}

	// Records that are not the store's: the unit 1 has another translation.
	other := maps.Clone(recs)
	r1 := other[key(file, 1)]
	r1.ZH = "換掉的譯文"
	other[key(file, 1)] = r1
	delete(other, key(file, 2))
	diff, err := NewPreviewerWith(env, nil, other)
	if err != nil {
		t.Fatal(err)
	}
	u1 := mustUnit(t, diff, key(file, 1))
	if u1.Record == nil || u1.Record.ZH != "換掉的譯文" || u1.Check == nil || u1.Check.ZH != "換掉的譯文" {
		t.Errorf("unit 1: record = %+v, check = %+v, want the translation of the records passed", u1.Record, u1.Check)
	}
	if u2 := mustUnit(t, diff, key(file, 2)); u2.Record != nil {
		t.Errorf("unit 2: record = %+v, want none (it is not in the records passed)", u2.Record)
	}
}
