package history

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// writePack writes files, keyed by pack file name, under a fresh directory
// with prefix on every file name, and opens it as a pack.
func writePack(t *testing.T, prefix string, files map[string]string) *pack.Pack {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		dir, base := path.Split(name)
		full := filepath.Join(root, filepath.FromSlash(dir), prefix+base)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p, err := pack.Open(root, prefix)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// rec is the record of file's string at pointer.
func rec(file, pointer, zh, origin string) store.Record {
	return store.Record{Key: units.Key(file, pointer), ZH: zh, Origin: origin}
}

func recs(rs ...store.Record) map[string]store.Record {
	m := map[string]store.Record{}
	for _, r := range rs {
		m[r.Key] = r
	}
	return m
}

// state builds a Release of one file: kr and zh are the JSON of its KR and built
// versions, either "" for a Release without the file.
func state(t *testing.T, kr, zh string, rs ...store.Record) State {
	t.Helper()
	krFiles, zhFiles := map[string]string{}, map[string]string{}
	if kr != "" {
		krFiles["F.json"] = kr
	}
	if zh != "" {
		zhFiles["F.json"] = zh
	}
	return State{KR: writePack(t, "KR_", krFiles), ZH: writePack(t, "", zhFiles), Records: recs(rs...)}
}

func list(ids ...string) string {
	s := `{"dataList":[`
	for i, d := range ids {
		if i > 0 {
			s += ","
		}
		s += d
	}
	return s + `]}`
}

func str(e *string) string {
	if e == nil {
		return "<nil>"
	}
	return *e
}

func sp(s string) *string { return &s }

func same(a, b *string) bool {
	return (a == nil) == (b == nil) && (a == nil || *a == *b)
}

func compare(t *testing.T, old, new State) []Entry {
	t.Helper()
	got, err := Compare(old, new, []string{"F.json"})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// want checks the strings of e against the given old KR, new KR, old ZH, new ZH
// (nil where the side has none).
func want(t *testing.T, e Entry, kind, ptr string, oldKR, newKR, oldZH, newZH *string) {
	t.Helper()
	if e.Kind != kind || e.Pointer != ptr || !same(e.OldKR, oldKR) || !same(e.NewKR, newKR) ||
		!same(e.OldZH, oldZH) || !same(e.NewZH, newZH) {
		t.Errorf("entry = %s %s oldKR=%s newKR=%s oldZH=%s newZH=%s", e.Kind, e.Pointer,
			str(e.OldKR), str(e.NewKR), str(e.OldZH), str(e.NewZH))
	}
}

func TestCompareTellsTheFourKinds(t *testing.T) {
	old := state(t,
		`{"dataList":[{"id":1,"desc":"a"},{"id":2,"desc":"b"},{"id":3,"desc":"c"}]}`,
		`{"dataList":[{"id":1,"desc":"甲"},{"id":2,"desc":"乙"},{"id":3,"desc":"丙"}]}`)
	new := state(t,
		`{"dataList":[{"id":1,"desc":"a"},{"id":2,"desc":"B"},{"id":4,"desc":"d"}]}`,
		`{"dataList":[{"id":1,"desc":"甲甲"},{"id":2,"desc":"乙乙"},{"id":4,"desc":"丁"}]}`)
	got := compare(t, old, new)
	if len(got) != 4 {
		t.Fatalf("got %d entries: %+v", len(got), got)
	}
	want(t, got[0], KindZHChanged, "/dataList/0/desc", nil, sp("a"), sp("甲"), sp("甲甲"))
	want(t, got[1], KindKRChanged, "/dataList/1/desc", sp("b"), sp("B"), sp("乙"), sp("乙乙"))
	want(t, got[2], KindNew, "/dataList/2/desc", nil, sp("d"), nil, sp("丁"))
	want(t, got[3], KindDeleted, "/dataList/2/desc", sp("c"), nil, sp("丙"), nil)
	if got[0].File != "F.json" || got[0].Field != "desc" {
		t.Errorf("file/field = %q %q", got[0].File, got[0].Field)
	}
}

func TestCompareLeavesOutWhatDidNotChange(t *testing.T) {
	kr := `{"dataList":[{"id":1,"desc":"a"},{"id":2,"desc":"b"}]}`
	zh := `{"dataList":[{"id":1,"desc":"甲"},{"id":2,"desc":"乙"}]}`
	if got := compare(t, state(t, kr, zh), state(t, kr, zh)); len(got) != 0 {
		t.Errorf("got %+v", got)
	}
}

func TestCompareAnEntryInsertedBeforeIsNoChange(t *testing.T) {
	old := state(t, list(`{"desc":"a"}`, `{"desc":"b"}`), list(`{"desc":"甲"}`, `{"desc":"乙"}`))
	new := state(t, list(`{"desc":"x"}`, `{"desc":"a"}`, `{"desc":"b"}`),
		list(`{"desc":"叉"}`, `{"desc":"甲"}`, `{"desc":"乙"}`))
	got := compare(t, old, new)
	if len(got) != 1 {
		t.Fatalf("got %+v", got)
	}
	want(t, got[0], KindNew, "/dataList/0/desc", nil, sp("x"), nil, sp("叉"))
}

func TestCompareAMovedUnitWhoseTranslationChanged(t *testing.T) {
	old := state(t, list(`{"desc":"a"}`, `{"desc":"b"}`), list(`{"desc":"甲"}`, `{"desc":"乙"}`))
	new := state(t, list(`{"desc":"x"}`, `{"desc":"a"}`, `{"desc":"b"}`),
		list(`{"desc":"叉"}`, `{"desc":"甲"}`, `{"desc":"乙乙"}`))
	got := compare(t, old, new)
	if len(got) != 2 {
		t.Fatalf("got %+v", got)
	}
	want(t, got[1], KindZHChanged, "/dataList/2/desc", nil, sp("b"), sp("乙"), sp("乙乙"))
	if got[1].OldPointer != "/dataList/1/desc" {
		t.Errorf("OldPointer = %q", got[1].OldPointer)
	}
	if got[0].OldPointer != "" {
		t.Errorf("new entry has OldPointer %q", got[0].OldPointer)
	}
}

func TestCompareMatchesByID(t *testing.T) {
	old := state(t,
		`{"dataList":[{"id":200341,"name":"song A"},{"id":200342,"name":"song B"},{"id":200343,"name":"song C"}]}`,
		`{"dataList":[{"id":200341,"name":"歌A"},{"id":200342,"name":"歌B"},{"id":200343,"name":"歌C"}]}`)
	new := state(t,
		`{"dataList":[{"id":200341,"name":"song A"},{"id":200342,"name":"song X"},{"id":200343,"name":"song Y"},{"id":200344,"name":"song Z"}]}`,
		`{"dataList":[{"id":200341,"name":"歌A"},{"id":200342,"name":"歌X"},{"id":200343,"name":"歌Y"},{"id":200344,"name":"歌Z"}]}`)
	got := compare(t, old, new)
	if len(got) != 3 {
		t.Fatalf("got %+v", got)
	}
	want(t, got[0], KindKRChanged, "/dataList/1/name", sp("song B"), sp("song X"), sp("歌B"), sp("歌X"))
	want(t, got[1], KindKRChanged, "/dataList/2/name", sp("song C"), sp("song Y"), sp("歌C"), sp("歌Y"))
	want(t, got[2], KindNew, "/dataList/3/name", nil, sp("song Z"), nil, sp("歌Z"))
}

func TestCompareAFileOnlyOnOneSide(t *testing.T) {
	kr := list(`{"id":1,"desc":"a"}`, `{"id":2,"desc":"b"}`)
	zh := list(`{"id":1,"desc":"甲"}`, `{"id":2,"desc":"乙"}`)
	none := state(t, "", "")

	got := compare(t, none, state(t, kr, zh))
	if len(got) != 2 {
		t.Fatalf("new file: got %+v", got)
	}
	want(t, got[0], KindNew, "/dataList/0/desc", nil, sp("a"), nil, sp("甲"))
	want(t, got[1], KindNew, "/dataList/1/desc", nil, sp("b"), nil, sp("乙"))

	got = compare(t, state(t, kr, zh), none)
	if len(got) != 2 {
		t.Fatalf("deleted file: got %+v", got)
	}
	want(t, got[0], KindDeleted, "/dataList/0/desc", sp("a"), nil, sp("甲"), nil)
	want(t, got[1], KindDeleted, "/dataList/1/desc", sp("b"), nil, sp("乙"), nil)

	// The built pack lacks the file: its translations are empty strings.
	got = compare(t, none, state(t, kr, ""))
	if len(got) != 2 {
		t.Fatalf("unbuilt file: got %+v", got)
	}
	want(t, got[0], KindNew, "/dataList/0/desc", nil, sp("a"), nil, sp(""))
}

func TestCompareOriginsCopyAndCause(t *testing.T) {
	const f = "F.json"
	oneKR := list(`{"id":1,"desc":"a"}`)
	oneZH := func(zh string) string { return list(`{"id":1,"desc":"` + zh + `"}`) }
	p := "/dataList/0/desc"

	// New is human: adjustment. The record's origin counts: its ZH is the built string.
	e := compare(t,
		state(t, oneKR, oneZH("甲"), rec(f, p, "甲", store.OriginClaude)),
		state(t, oneKR, oneZH("乙"), rec(f, p, "乙", store.OriginHuman)))[0]
	if e.Kind != KindZHChanged || e.Cause != CauseAdjustment || e.OldOrigin != store.OriginClaude ||
		e.NewOrigin != store.OriginHuman || e.Copy {
		t.Errorf("adjustment: %+v", e)
	}

	// The old string is not the record's, or has no record: a stand-in. Late.
	e = compare(t,
		state(t, oneKR, oneZH("a"), rec(f, p, "別的", store.OriginClaude)),
		state(t, oneKR, oneZH("乙"), rec(f, p, "乙", store.OriginClaude)))[0]
	if e.OldOrigin != OriginStandIn || e.Cause != CauseLate {
		t.Errorf("late: %+v", e)
	}

	// The old origin is fallback (EN or KR stood in): late too.
	e = compare(t,
		state(t, oneKR, oneZH("a"), rec(f, p, "a", store.OriginFallback)),
		state(t, oneKR, oneZH("乙"), rec(f, p, "乙", store.OriginClaude)))[0]
	if e.OldOrigin != store.OriginFallback || e.Cause != CauseLate {
		t.Errorf("late after fallback: %+v", e)
	}

	// Old paratranz, new claude: retranslated.
	e = compare(t,
		state(t, oneKR, oneZH("甲"), rec(f, p, "甲", store.OriginParatranz)),
		state(t, oneKR, oneZH("乙"), rec(f, p, "乙", store.OriginClaude)))[0]
	if e.Cause != CauseRetranslated {
		t.Errorf("retranslated: %+v", e)
	}

	// Copy follows the new origin, the old one for a deleted string.
	e = compare(t,
		state(t, oneKR, oneZH("甲"), rec(f, p, "甲", store.OriginClaude)),
		state(t, oneKR, oneZH("a"), rec(f, p, "a", store.OriginCopy)))[0]
	if !e.Copy {
		t.Errorf("copy: %+v", e)
	}
	e = compare(t, state(t, oneKR, oneZH("a"), rec(f, p, "a", store.OriginCopy)), state(t, "", ""))[0]
	if e.Kind != KindDeleted || !e.Copy || e.OldOrigin != store.OriginCopy {
		t.Errorf("deleted copy: %+v", e)
	}
}
