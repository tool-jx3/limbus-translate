package build

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"

	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

const (
	testTag   = "raw-20261029"
	testToken = "l20261029_test"
)

// testBuiltAt has sub-second parts on purpose: only whole seconds may reach
// the output.
var testBuiltAt = time.Date(2026, 10, 31, 12, 30, 45, 999, time.UTC)

// testEnv is a raw checkout, rules and options for one build.
type testEnv struct {
	t   *testing.T
	src pack.Set
	fr  *fields.Rules
	opt Options
}

// writeLang writes files, keyed by pack file name, under <root>/<lang> with
// the language prefix on the file name.
func writeLang(t *testing.T, root, lang, prefix string, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, lang), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		dir, base := path.Split(name)
		full := filepath.Join(root, lang, filepath.FromSlash(dir), prefix+base)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// newEnv makes a raw checkout with the given KR and EN files (JP is empty),
// the repo's field rules and options pointing at fresh directories. The font
// is Go Regular: small, with Latin letters and no Han.
func newEnv(t *testing.T, kr, en map[string]string) *testEnv {
	t.Helper()
	dir := t.TempDir()
	raw := filepath.Join(dir, "raw")
	writeLang(t, raw, "kr", "KR_", kr)
	writeLang(t, raw, "en", "EN_", en)
	writeLang(t, raw, "jp", "JP_", nil)
	src, err := pack.OpenRaw(raw)
	if err != nil {
		t.Fatal(err)
	}
	fr, err := fields.Load("../..")
	if err != nil {
		t.Fatal(err)
	}

	fontDir := filepath.Join(dir, "fonts")
	if err := os.MkdirAll(fontDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(fontDir, fontFile), goregular.TTF)
	mustWrite(t, filepath.Join(fontDir, "OFL.txt"), []byte("OFL text\n"))
	mustWrite(t, filepath.Join(dir, "LICENSE-CONTENT"), []byte("content license\n"))
	sum := sha256.Sum256(goregular.TTF)

	tw := filepath.Join(dir, "tw")
	if err := os.MkdirAll(tw, 0o755); err != nil {
		t.Fatal(err)
	}
	return &testEnv{t: t, src: src, fr: fr, opt: Options{
		TWRoot:      tw,
		OutDir:      filepath.Join(dir, "out"),
		LangDir:     "zh-TW",
		FontPath:    filepath.Join(fontDir, fontFile),
		FontSHA256:  hex.EncodeToString(sum[:]),
		LicensePath: filepath.Join(dir, "LICENSE-CONTENT"),
		RawTag:      testTag,
		RawToken:    testToken,
		RulesCommit: "abc1234",
		BuiltAt:     testBuiltAt,
		Frozen:      []string{"BgmLyrics/**"},
		Upstream:    map[string]string{"paratranz": "owner/pz@11", "ltm": "owner/ltm@22"},
	}}
}

func mustWrite(t *testing.T, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(name, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// run builds with the environment's options.
func (e *testEnv) run() *Report {
	e.t.Helper()
	rep, err := Run(e.opt, e.src, e.fr)
	if err != nil {
		e.t.Fatalf("Run: %v", err)
	}
	return rep
}

// out reads a file of the built language directory.
func (e *testEnv) out(name string) string {
	e.t.Helper()
	b, err := os.ReadFile(filepath.Join(e.opt.TWRoot, dataDir, "Lang", e.opt.LangDir, filepath.FromSlash(name)))
	if err != nil {
		e.t.Fatal(err)
	}
	return string(b)
}

func (e *testEnv) save(recs ...store.Record) {
	e.t.Helper()
	m := map[string]store.Record{}
	for _, r := range recs {
		m[r.Key] = r
	}
	if err := store.Open(e.opt.TWRoot).SaveRecords(m); err != nil {
		e.t.Fatal(err)
	}
}

func (e *testEnv) pending() []store.PendingItem {
	e.t.Helper()
	items, err := store.Open(e.opt.TWRoot).LoadPending()
	if err != nil {
		e.t.Fatal(err)
	}
	return items
}

// rec is a record of file|pointer translated from kr.
func rec(file, pointer, kr, zh string) store.Record {
	return store.Record{
		Key: units.Key(file, pointer), KRSha: units.KRSha(kr), ZH: zh,
		Origin: store.OriginClaude, Since: testTag,
	}
}

const (
	fillKR = "\xef\xbb\xbf{\r\n\"dataList\": [\r\n" +
		`{"id": 1, "title": "가", "content": "<color=#ff0000>나</color> [Bleed] 다"},` + "\r\n" +
		`{"id": 2, "title": "라"}` + "\r\n]}\r\n"
	fillEN = `{"dataList":[{"id":1,"title":"A","content":"<color=#ff0000>B</color> [Bleed] C"},{"id":2,"title":"Third EN"}]}`
)

func TestBuildFillsAndOrders(t *testing.T) {
	e := newEnv(t, map[string]string{"A.json": fillKR}, map[string]string{"A.json": fillEN})
	e.save(
		rec("A.json", "/dataList/0/title", "가", "甲"),
		rec("A.json", "/dataList/0/content", "<color=#ff0000>나</color> [Bleed] 다", "<color=#ff0000>乙</color> [Bleed] 丙"),
	)
	rep := e.run()

	want := `{
  "dataList": [
    {
      "id": 1,
      "title": "甲",
      "content": "<color=#ff0000>乙</color> [Bleed] 丙"
    },
    {
      "id": 2,
      "title": "Third EN"
    }
  ]
}
`
	got := e.out("A.json")
	if got != want {
		t.Errorf("A.json =\n%s\nwant\n%s", got, want)
	}
	if strings.HasPrefix(got, "\xef\xbb\xbf") || strings.Contains(got, "\r") {
		t.Errorf("output has a BOM or CR: %q", got)
	}
	if !strings.Contains(got, "<color") || strings.Contains(got, `\u003c`) {
		t.Errorf("output escapes <: %q", got)
	}
	if rep.Pending != 1 || rep.Unaligned != 0 || rep.StructuralFallback != 0 || rep.NeedsReview != 0 {
		t.Errorf("Pending=%d Unaligned=%d StructuralFallback=%d NeedsReview=%d, want 1 0 0 0",
			rep.Pending, rep.Unaligned, rep.StructuralFallback, rep.NeedsReview)
	}
	if want := map[string]int{"claude": 2}; !reflect.DeepEqual(rep.Origins, want) {
		t.Errorf("Origins = %v, want %v", rep.Origins, want)
	}
	if rep.Warnings == nil || len(rep.Warnings) != 0 {
		t.Errorf("Warnings = %#v, want an empty map", rep.Warnings)
	}
}

func TestBuildCountsPendingAndSkipsWhatIsNotTranslated(t *testing.T) {
	kr := `{"dataList":[{"id":1,"model":"Meursault","title":"가","desc":"-","name":"나"}]}`
	en := `{"dataList":[{"id":1,"model":"Meursault","title":"A","desc":"-"}]}`
	e := newEnv(t, map[string]string{"A.json": kr}, map[string]string{"A.json": en})
	rep := e.run()

	// model is an identifier field and "-" is empty: both keep KR and count
	// nowhere. title has EN. name has no EN: KR stands in and it is unaligned.
	want := `{
  "dataList": [
    {
      "id": 1,
      "model": "Meursault",
      "title": "A",
      "desc": "-",
      "name": "나"
    }
  ]
}
`
	if got := e.out("A.json"); got != want {
		t.Errorf("A.json =\n%s\nwant\n%s", got, want)
	}
	if rep.Pending != 2 || rep.Unaligned != 1 {
		t.Errorf("Pending=%d Unaligned=%d, want 2 1", rep.Pending, rep.Unaligned)
	}
	if len(rep.Origins) != 0 {
		t.Errorf("Origins = %v, want none: no record was used", rep.Origins)
	}
	if got := e.pending(); len(got) != 0 {
		t.Errorf("pending = %v: build must not queue units that have no record", got)
	}
}

// Field rules come before records (spec 8.2): a unit they do not translate
// keeps KR, whatever a record says.
func TestBuildFieldRulesBeatRecords(t *testing.T) {
	kr := `{"dataList":[{"id":1,"model":"뫼르소","content":"가"},{"id":2,"model":"이상","content":"-"}]}`
	e := newEnv(t, map[string]string{"A.json": kr}, nil)
	model := rec("A.json", "/dataList/0/model", "뫼르소", "莫梭") // model is an identifier field
	dash := rec("A.json", "/dataList/1/content", "-", "－")   // "-" is an empty value
	dash.Origin = store.OriginHuman
	copied := rec("A.json", "/dataList/1/model", "이상", "이상")
	copied.Origin = store.OriginCopy
	e.save(model, dash, copied)
	rep := e.run()

	want := `{
  "dataList": [
    {
      "id": 1,
      "model": "뫼르소",
      "content": "가"
    },
    {
      "id": 2,
      "model": "이상",
      "content": "-"
    }
  ]
}
`
	if got := e.out("A.json"); got != want {
		t.Errorf("A.json =\n%s\nwant\n%s", got, want)
	}
	// The copy agrees with the field rules and counts as used; the other two
	// were overridden.
	if want := map[string]int{"copy": 1}; !reflect.DeepEqual(rep.Origins, want) {
		t.Errorf("Origins = %v, want %v", rep.Origins, want)
	}
	if rep.Pending != 1 {
		t.Errorf("Pending = %d, want 1: only the content of entry 1", rep.Pending)
	}
	if rep.FieldRuleOverride != 2 {
		t.Errorf("FieldRuleOverride = %d, want 2", rep.FieldRuleOverride)
	}
	if md := string(Markdown(e.opt, rep)); !strings.Contains(md, "| Field rule override | 2 |") {
		t.Errorf("report lacks the field rule override row:\n%s", md)
	}
}

func TestBuildUsesRecordOnlyForTheCurrentKR(t *testing.T) {
	krFile := `{"dataList":[{"id":1,"content":"가"}]}`
	enFile := `{"dataList":[{"id":1,"content":"EN text"}]}`
	e := newEnv(t,
		map[string]string{"A.json": krFile, "BgmLyrics/L.json": krFile},
		map[string]string{"A.json": enFile, "BgmLyrics/L.json": enFile})
	stale := rec("A.json", "/dataList/0/content", "옛날", "舊譯")
	frozen := rec("BgmLyrics/L.json", "/dataList/0/content", "옛날", "凍結的譯文")
	e.save(stale, frozen)
	rep := e.run()

	if got := e.out("A.json"); !strings.Contains(got, `"EN text"`) {
		t.Errorf("A.json uses a record made from other KR text:\n%s", got)
	}
	if got := e.out("BgmLyrics/L.json"); !strings.Contains(got, `"凍結的譯文"`) {
		t.Errorf("a frozen file must keep its record although KR changed:\n%s", got)
	}
	if rep.Pending != 1 || rep.Origins["claude"] != 1 {
		t.Errorf("Pending=%d Origins=%v, want 1 pending and one used record", rep.Pending, rep.Origins)
	}
}

func TestBuildStructuralFallback(t *testing.T) {
	kr := `{"dataList":[{"id":1,"content":"[Bleed] 중첩"},{"id":2,"content":"나"}]}`
	en := `{"dataList":[{"id":1,"content":"[Bleed] stack"},{"id":2,"content":"B"}]}`
	e := newEnv(t, map[string]string{"A.json": kr}, map[string]string{"A.json": en})
	broken := rec("A.json", "/dataList/0/content", "[Bleed] 중첩", "中疊")
	fine := rec("A.json", "/dataList/1/content", "나", "乙")
	e.save(broken, fine)

	rep := e.run()
	if got := e.out("A.json"); !strings.Contains(got, `"[Bleed] stack"`) || strings.Contains(got, "中疊") || !strings.Contains(got, `"乙"`) {
		t.Errorf("A.json =\n%s\nwant EN for the unit that lost [Bleed] and the translation of the other", got)
	}
	if rep.StructuralFallback != 1 || rep.Pending != 0 {
		t.Errorf("StructuralFallback=%d Pending=%d, want 1 0", rep.StructuralFallback, rep.Pending)
	}
	if want := map[string]int{"claude": 2}; !reflect.DeepEqual(rep.Origins, want) {
		t.Errorf("Origins = %v, want %v (the record was used, then rejected)", rep.Origins, want)
	}
	// The broken translation rides along as the draft to fix.
	want := []store.PendingItem{{Key: broken.Key, Reason: "structural-fail", Draft: "中疊"}}
	if got := e.pending(); !reflect.DeepEqual(got, want) {
		t.Errorf("pending = %+v, want %+v", got, want)
	}
	// The record is not touched.
	if recs, err := store.Open(e.opt.TWRoot).LoadRecords(); err != nil || !reflect.DeepEqual(recs[broken.Key], broken) {
		t.Errorf("record changed: %+v, %v", recs[broken.Key], err)
	}

	// Building again changes nothing: the item is queued once.
	e.run()
	if got := e.pending(); !reflect.DeepEqual(got, want) {
		t.Errorf("pending after a second build = %+v, want %+v", got, want)
	}
}

func TestBuildStructuralFallbackKeepsQueuedItem(t *testing.T) {
	kr := `{"dataList":[{"id":1,"content":"[Bleed] 중첩"}]}`
	e := newEnv(t, map[string]string{"A.json": kr}, nil)
	broken := rec("A.json", "/dataList/0/content", "[Bleed] 중첩", "中疊")
	e.save(broken)
	queued := store.PendingItem{Key: broken.Key, Reason: "requeue", Attempts: 1}
	if err := store.Open(e.opt.TWRoot).SavePending([]store.PendingItem{queued}); err != nil {
		t.Fatal(err)
	}

	rep := e.run()
	// No EN: KR stands in.
	if got := e.out("A.json"); !strings.Contains(got, `"[Bleed] 중첩"`) {
		t.Errorf("A.json =\n%s\nwant KR when there is no EN", got)
	}
	if rep.StructuralFallback != 1 {
		t.Errorf("StructuralFallback = %d, want 1", rep.StructuralFallback)
	}
	if got := e.pending(); !reflect.DeepEqual(got, []store.PendingItem{queued}) {
		t.Errorf("pending = %+v, want the queued item unchanged", got)
	}
}

func TestBuildReportsFlagsAndWarnings(t *testing.T) {
	kr := `{"dataList":[{"id":1,"content":"가"},{"id":2,"content":"나"},{"id":3,"content":"다"}]}`
	e := newEnv(t, map[string]string{"A.json": kr}, nil)
	warned := rec("A.json", "/dataList/0/content", "가", "甲가") // Hangul left: a warning for an existing unit
	review := rec("A.json", "/dataList/1/content", "나", "B")
	review.Origin, review.Flags = store.OriginFallback, []string{store.FlagNeedsReview}
	human := rec("A.json", "/dataList/2/content", "다", "丙")
	human.Origin = store.OriginHuman
	e.save(warned, review, human)

	rep := e.run()
	if got := e.out("A.json"); !strings.Contains(got, `"甲가"`) {
		t.Errorf("a warning must not stop the translation from being used:\n%s", got)
	}
	if rep.NeedsReview != 1 || rep.StructuralFallback != 0 {
		t.Errorf("NeedsReview=%d StructuralFallback=%d, want 1 0", rep.NeedsReview, rep.StructuralFallback)
	}
	if want := map[string]int{"claude": 1, "fallback": 1, "human": 1}; !reflect.DeepEqual(rep.Origins, want) {
		t.Errorf("Origins = %v, want %v", rep.Origins, want)
	}
	if rep.Warnings["hangul"] != 1 {
		t.Errorf("Warnings = %v, want one hangul", rep.Warnings)
	}
}

// A copy record holds the KR text itself: it is not checked, so a name kept in
// Hangul adds no warning.
func TestBuildDoesNotCheckCopyRecords(t *testing.T) {
	kr := `{"dataList":[{"id":1,"content":"이스마엘"},{"id":2,"content":"가"}]}`
	e := newEnv(t, map[string]string{"A.json": kr}, nil)
	name := rec("A.json", "/dataList/0/content", "이스마엘", "이스마엘")
	name.Origin = store.OriginCopy
	warned := rec("A.json", "/dataList/1/content", "가", "甲가")
	e.save(name, warned)

	rep := e.run()
	if got := e.out("A.json"); !strings.Contains(got, `"이스마엘"`) {
		t.Errorf("A.json =\n%s\nwant the copy", got)
	}
	if want := map[string]int{"copy": 1, "claude": 1}; !reflect.DeepEqual(rep.Origins, want) {
		t.Errorf("Origins = %v, want %v", rep.Origins, want)
	}
	if want := map[string]int{"hangul": 1}; !reflect.DeepEqual(rep.Warnings, want) {
		t.Errorf("Warnings = %v, want %v: only the claude record", rep.Warnings, want)
	}
}

// A record that falls back structurally is still a used record: its warnings
// are counted with its origin and its review flag, so that all three cover the
// same set of records.
func TestBuildCountsWarningsOfRecordsThatFallBack(t *testing.T) {
	kr := `{"dataList":[{"id":1,"content":"[Bleed] 중첩"},{"id":2,"content":"나"}]}`
	en := `{"dataList":[{"id":1,"content":"[Bleed] stack"},{"id":2,"content":"B"}]}`
	e := newEnv(t, map[string]string{"A.json": kr}, map[string]string{"A.json": en})
	// Loses [Bleed] (hard) and keeps Hangul (a warning for an existing unit).
	broken := rec("A.json", "/dataList/0/content", "[Bleed] 중첩", "中疊가")
	broken.Flags = []string{store.FlagNeedsReview}
	warned := rec("A.json", "/dataList/1/content", "나", "乙나")
	e.save(broken, warned)

	rep := e.run()
	if got := e.out("A.json"); !strings.Contains(got, `"[Bleed] stack"`) || strings.Contains(got, "中疊") {
		t.Errorf("A.json =\n%s\nwant EN for the unit that lost [Bleed]", got)
	}
	if rep.StructuralFallback != 1 || rep.NeedsReview != 1 {
		t.Errorf("StructuralFallback=%d NeedsReview=%d, want 1 1", rep.StructuralFallback, rep.NeedsReview)
	}
	if want := map[string]int{"claude": 2}; !reflect.DeepEqual(rep.Origins, want) {
		t.Errorf("Origins = %v, want %v", rep.Origins, want)
	}
	if rep.Warnings["hangul"] != 2 {
		t.Errorf("Warnings = %v, want hangul 2: the unit that fell back counts too", rep.Warnings)
	}
	// Only warnings are counted, never the hard issue that caused the fallback.
	if rep.Warnings["id-mismatch"] != 0 {
		t.Errorf("Warnings = %v counts a hard issue", rep.Warnings)
	}
}

// A frozen file is not translated again, so its records stand even when KR
// changed. A record that breaks the structure of KR would break the game:
// EN (or KR) stands in, the key is listed as a frozen fallback, and nothing is
// queued or counted as a structural fallback.
func TestBuildFrozenRecords(t *testing.T) {
	kr := `{"dataList":[{"id":1,"title":"[Bleed] 가","content":"[Bleed] 중첩"},{"id":2,"content":"나"}]}`
	en := `{"dataList":[{"id":1,"title":"[Bleed] A","content":"[Bleed] stack"},{"id":2,"content":"B"}]}`
	e := newEnv(t,
		map[string]string{"BgmLyrics/L.json": kr, "A.json": kr},
		map[string]string{"BgmLyrics/L.json": en, "A.json": en})
	// Lost [Bleed] (hard), kept Hangul (warning), flagged, and made from other KR text.
	lyric := rec("BgmLyrics/L.json", "/dataList/0/content", "옛날", "中疊가")
	lyric.Origin, lyric.Flags = store.OriginParatranz, []string{store.FlagNeedsReview}
	title := rec("BgmLyrics/L.json", "/dataList/0/title", "[Bleed] 가", "甲") // lost [Bleed]
	// Made from other KR text but keeps the structure: used as it is.
	kept := rec("BgmLyrics/L.json", "/dataList/1/content", "옛날", "乙나")
	// The same broken record in a file that is not frozen falls back and is queued.
	plain := rec("A.json", "/dataList/0/content", "[Bleed] 중첩", "中疊가")
	e.save(lyric, title, kept, plain)

	rep := e.run()
	got := e.out("BgmLyrics/L.json")
	for _, want := range []string{`"[Bleed] stack"`, `"[Bleed] A"`, `"乙나"`} {
		if !strings.Contains(got, want) {
			t.Errorf("frozen file lacks %s:\n%s", want, got)
		}
	}
	if strings.Contains(got, "中疊") || strings.Contains(got, "甲") {
		t.Errorf("frozen file ships a record that breaks the structure:\n%s", got)
	}
	if got := e.out("A.json"); !strings.Contains(got, `"[Bleed] stack"`) {
		t.Errorf("a file that is not frozen must still fall back:\n%s", got)
	}
	if rep.StructuralFallback != 1 {
		t.Errorf("StructuralFallback = %d, want 1: only the file that is not frozen", rep.StructuralFallback)
	}
	// Sorted keys, not document order (title comes first in the file).
	if want := []string{lyric.Key, title.Key}; !slices.Equal(rep.FrozenFallback, want) {
		t.Errorf("FrozenFallback = %v, want %v", rep.FrozenFallback, want)
	}
	if want := map[string]int{"paratranz": 1, "claude": 3}; !reflect.DeepEqual(rep.Origins, want) {
		t.Errorf("Origins = %v, want %v", rep.Origins, want)
	}
	if rep.NeedsReview != 0 {
		t.Errorf("NeedsReview = %d, want 0: the flagged frozen record is listed in FrozenFallback, once", rep.NeedsReview)
	}
	if rep.Warnings["hangul"] != 3 {
		t.Errorf("Warnings = %v, want hangul 3: the frozen records' warnings count", rep.Warnings)
	}
	want := []store.PendingItem{{Key: plain.Key, Reason: "structural-fail", Draft: "中疊가"}}
	if got := e.pending(); !reflect.DeepEqual(got, want) {
		t.Errorf("pending = %+v, want only %+v: a frozen record is never queued", got, want)
	}
	md := string(Markdown(e.opt, rep))
	for _, want := range []string{"| Frozen fallback | 2 |", "- `" + lyric.Key + "`\n- `" + title.Key + "`\n"} {
		if !strings.Contains(md, want) {
			t.Errorf("report lacks %q:\n%s", want, md)
		}
	}

	// Without any, the list is empty, not null.
	e2 := newEnv(t, map[string]string{"A.json": `{"dataList":[]}`}, nil)
	data, err := json.Marshal(e2.run())
	if err != nil || !strings.Contains(string(data), `"frozen_fallback":[]`) {
		t.Errorf("report JSON = %s, %v, want an empty frozen_fallback list", data, err)
	}
}

// A frozen record that breaks the structure of KR is a problem for a person,
// listed in FrozenFallback. Its needs_review flag does not count it a second
// time: NeedsReview is for the records that were used as they are.
func TestBuildDoesNotCountFrozenFallbacksAsNeedsReview(t *testing.T) {
	kr := `{"dataList":[{"id":1,"content":"[Bleed] 중첩"},{"id":2,"content":"나"}]}`
	en := `{"dataList":[{"id":1,"content":"[Bleed] stack"},{"id":2,"content":"B"}]}`
	e := newEnv(t, map[string]string{"BgmLyrics/L.json": kr}, map[string]string{"BgmLyrics/L.json": en})
	broken := rec("BgmLyrics/L.json", "/dataList/0/content", "[Bleed] 중첩", "中疊") // lost [Bleed]
	broken.Flags = []string{store.FlagNeedsReview}
	flagged := rec("BgmLyrics/L.json", "/dataList/1/content", "나", "乙") // fine, but a person should look
	flagged.Flags = []string{store.FlagNeedsReview}
	e.save(broken, flagged)

	rep := e.run()
	if want := []string{broken.Key}; !slices.Equal(rep.FrozenFallback, want) {
		t.Errorf("FrozenFallback = %v, want %v", rep.FrozenFallback, want)
	}
	if rep.NeedsReview != 1 {
		t.Errorf("NeedsReview = %d, want 1: only the flagged record that was used, not the frozen fallback", rep.NeedsReview)
	}
	// Both records are still used ones.
	if want := map[string]int{"claude": 2}; !reflect.DeepEqual(rep.Origins, want) {
		t.Errorf("Origins = %v, want %v", rep.Origins, want)
	}

	// With only the frozen fallback, nothing needs review.
	e2 := newEnv(t, map[string]string{"BgmLyrics/L.json": kr}, map[string]string{"BgmLyrics/L.json": en})
	e2.save(broken)
	if rep := e2.run(); rep.NeedsReview != 0 || len(rep.FrozenFallback) != 1 {
		t.Errorf("NeedsReview = %d, FrozenFallback = %v, want 0 and the one key", rep.NeedsReview, rep.FrozenFallback)
	}
}

func TestBuildEmptyFile(t *testing.T) {
	e := newEnv(t, map[string]string{"E.json": `{"dataList":[]}`}, nil)
	rep := e.run()
	if got, want := e.out("E.json"), "{\n  \"dataList\": []\n}\n"; got != want {
		t.Errorf("E.json = %q, want %q", got, want)
	}
	if rep.Pending != 0 {
		t.Errorf("Pending = %d, want 0", rep.Pending)
	}
}

func TestBuildFileSetFollowsKR(t *testing.T) {
	e := newEnv(t,
		map[string]string{"A.json": `{"dataList":[]}`, "StoryData/S1.json": `{"dataList":[]}`},
		map[string]string{"A.json": `{"dataList":[]}`, "OnlyEN.json": `{"dataList":[]}`})
	e.run()
	got := listFiles(t, filepath.Join(e.opt.TWRoot, dataDir, "Lang", "zh-TW"))
	want := []string{"A.json", "Info/LICENSE-CONTENT", "Info/version.json", "StoryData/S1.json"}
	if !slices.Equal(got, want) {
		t.Errorf("files = %v, want %v (a file only EN has is not built)", got, want)
	}
}

func TestBuildRemovesStaleOutput(t *testing.T) {
	e := newEnv(t, map[string]string{"A.json": `{"dataList":[]}`}, nil)
	lang := filepath.Join(e.opt.TWRoot, dataDir, "Lang", "zh-TW")
	mustMkdirWrite(t, filepath.Join(lang, "Gone.json"), "{}")
	mustMkdirWrite(t, filepath.Join(lang, "StoryData", "Old", "S.json"), "{}")
	mustMkdirWrite(t, filepath.Join(lang, "Font", "Context", fontFile), "old font")
	mustMkdirWrite(t, filepath.Join(e.opt.TWRoot, dataDir, "Lang", "LLC_zh-CN", "X.json"), "{}")
	mustMkdirWrite(t, filepath.Join(e.opt.TWRoot, ".meta", "notes.json"), "{}")
	mustMkdirWrite(t, filepath.Join(e.opt.TWRoot, "README.md"), "keep")
	e.run()

	got := listFiles(t, filepath.Join(e.opt.TWRoot, dataDir))
	want := []string{"Lang/config.json", "Lang/zh-TW/A.json", "Lang/zh-TW/Info/LICENSE-CONTENT", "Lang/zh-TW/Info/version.json"}
	if !slices.Equal(got, want) {
		t.Errorf("files = %v, want %v", got, want)
	}
	for _, dir := range []string{
		filepath.Join(lang, "StoryData"), filepath.Join(lang, "Font"), filepath.Join(e.opt.TWRoot, dataDir, "Lang", "LLC_zh-CN"),
	} {
		if _, err := os.Stat(dir); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("empty directory %s was left behind (%v)", dir, err)
		}
	}
	for _, keep := range []string{filepath.Join(".meta", "notes.json"), "README.md"} {
		if _, err := os.Stat(filepath.Join(e.opt.TWRoot, keep)); err != nil {
			t.Errorf("%s outside %s was touched: %v", keep, dataDir, err)
		}
	}
}

func mustMkdirWrite(t *testing.T, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, name, []byte(content))
}

// listFiles lists the regular files below dir as sorted slash paths.
func listFiles(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		out = append(out, filepath.ToSlash(rel))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(out)
	return out
}

// zipEntries reads a zip: the entry names in file order, their contents and
// their headers.
func zipEntries(t *testing.T, name string) (names []string, data map[string]string, files map[string]*zip.File) {
	t.Helper()
	zr, err := zip.OpenReader(name)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { zr.Close() })
	data, files = map[string]string{}, map[string]*zip.File{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, f.Name)
		data[f.Name], files[f.Name] = string(b), f
	}
	return names, data, files
}

func TestBuildConfigAndZip(t *testing.T) {
	e := newEnv(t, map[string]string{"A.json": `{"dataList":[]}`, "StoryData/S1.json": `{"dataList":[]}`}, nil)
	e.run()

	if got, want := e.out("../config.json"), "{\"lang\": \"zh-TW\"}\n"; got != want {
		t.Errorf("Lang/config.json = %q, want %q", got, want)
	}
	if got := e.out("Info/LICENSE-CONTENT"); got != "content license\n" {
		t.Errorf("Info/LICENSE-CONTENT = %q", got)
	}
	wantVersion := `{
  "raw_tag": "raw-20261029",
  "raw_token": "l20261029_test",
  "rules_commit": "abc1234",
  "built_at": "2026-10-31T12:30:45Z",
  "upstream": {
    "ltm": "owner/ltm@22",
    "paratranz": "owner/pz@11"
  }
}
`
	if got := e.out("Info/version.json"); got != wantVersion {
		t.Errorf("Info/version.json =\n%s\nwant\n%s", got, wantVersion)
	}
	// The font goes into the complete zip only, not into the worktree.
	if _, err := os.Stat(filepath.Join(e.opt.TWRoot, dataDir, "Lang", "zh-TW", "Font")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the tw worktree has a Font directory (%v)", err)
	}

	textZip, completeZip := ZipPaths(e.opt)
	if filepath.Base(textZip) != "zh-TW_text_raw-20261029.zip" || filepath.Base(completeZip) != "zh-TW_complete_raw-20261029.zip" {
		t.Errorf("zip names = %s, %s", textZip, completeZip)
	}

	lang := dataDir + "/Lang/zh-TW/"
	textNames, text, _ := zipEntries(t, textZip)
	wantText := []string{
		dataDir + "/Lang/config.json", lang + "A.json", lang + "Info/LICENSE-CONTENT", lang + "Info/version.json", lang + "StoryData/S1.json",
	}
	if !slices.Equal(textNames, wantText) {
		t.Errorf("text zip entries = %v, want %v", textNames, wantText)
	}
	if text[lang+"A.json"] != e.out("A.json") {
		t.Errorf("text zip A.json differs from the worktree file")
	}

	completeNames, complete, files := zipEntries(t, completeZip)
	wantComplete := []string{
		dataDir + "/Lang/config.json", lang + "A.json",
		lang + "Font/Context/" + fontFile, lang + "Font/OFL.txt", lang + "Font/Title/" + fontFile,
		lang + "Info/LICENSE-CONTENT", lang + "Info/version.json", lang + "StoryData/S1.json",
	}
	if !slices.Equal(completeNames, wantComplete) {
		t.Errorf("complete zip entries = %v, want %v", completeNames, wantComplete)
	}
	for _, f := range []string{lang + "Font/Context/" + fontFile, lang + "Font/Title/" + fontFile} {
		if complete[f] != string(goregular.TTF) {
			t.Errorf("%s is not the font", f)
		}
	}
	if complete[lang+"Font/OFL.txt"] != "OFL text\n" {
		t.Errorf("Font/OFL.txt = %q", complete[lang+"Font/OFL.txt"])
	}
	for name, data := range text {
		if complete[name] != data {
			t.Errorf("%s differs between the zips", name)
		}
	}

	// Reproducible zips: one time stamp, deflated, no directory entries.
	wantTime := testBuiltAt.Truncate(time.Second)
	for name, f := range files {
		if !f.Modified.Equal(wantTime) {
			t.Errorf("%s Modified = %v, want %v", name, f.Modified, wantTime)
		}
		if f.Method != zip.Deflate {
			t.Errorf("%s method = %d, want Deflate", name, f.Method)
		}
		if strings.HasSuffix(name, "/") {
			t.Errorf("directory entry %s", name)
		}
	}
}

// treeFiles reads every file under the data directory and the build records
// of a worktree, plus the zips, as name -> content.
func treeFiles(t *testing.T, opt Options) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, root := range []string{filepath.Join(opt.TWRoot, dataDir), filepath.Join(opt.TWRoot, ".meta", "reports"), opt.OutDir} {
		for _, rel := range listFiles(t, root) {
			b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
			if err != nil {
				t.Fatal(err)
			}
			out[filepath.Base(root)+"/"+rel] = string(b)
		}
	}
	b, err := os.ReadFile(filepath.Join(opt.TWRoot, ".meta", "build.json"))
	if err != nil {
		t.Fatal(err)
	}
	out["build.json"] = string(b)
	return out
}

func TestBuildDeterministic(t *testing.T) {
	kr := map[string]string{"A.json": fillKR, "StoryData/S1.json": `{"dataList":[{"id":1,"content":"가"}]}`}
	en := map[string]string{"A.json": fillEN}
	recs := []store.Record{
		rec("A.json", "/dataList/0/title", "가", "甲"),
		rec("StoryData/S1.json", "/dataList/0/content", "가", "乙"),
	}
	e := newEnv(t, kr, en)
	e.save(recs...)
	e.run()
	first := treeFiles(t, e.opt)

	// The same inputs in other directories, built a second later.
	other := *e
	other.opt.TWRoot = filepath.Join(t.TempDir(), "tw")
	other.opt.OutDir = filepath.Join(t.TempDir(), "out")
	other.save(recs...)
	other.run()
	if second := treeFiles(t, other.opt); !reflect.DeepEqual(first, second) {
		t.Errorf("two builds of the same inputs differ: %v", diffKeys(first, second))
	}

	// Building again over the output changes nothing either.
	time.Sleep(1100 * time.Millisecond)
	e.run()
	if third := treeFiles(t, e.opt); !reflect.DeepEqual(first, third) {
		t.Errorf("a rebuild differs from the first build: %v", diffKeys(first, third))
	}
}

func diffKeys(a, b map[string]string) []string {
	var diff []string
	for k, v := range a {
		if w, ok := b[k]; !ok || w != v {
			diff = append(diff, k)
		}
	}
	for k := range b {
		if _, ok := a[k]; !ok {
			diff = append(diff, k)
		}
	}
	slices.Sort(diff)
	return diff
}

func TestBuildNeedsAFontThatMatches(t *testing.T) {
	e := newEnv(t, map[string]string{"A.json": `{"dataList":[]}`}, nil)
	e.opt.FontSHA256 = strings.Repeat("0", 64)
	_, err := Run(e.opt, e.src, e.fr)
	if !errors.Is(err, ErrFontChecksum) {
		t.Fatalf("Run with a wrong checksum = %v, want ErrFontChecksum", err)
	}
	if _, err := os.Stat(filepath.Join(e.opt.TWRoot, dataDir)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the pack was written before the font was checked (%v)", err)
	}
	if _, err := os.Stat(e.opt.OutDir); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("out was created before the font was checked (%v)", err)
	}

	// The checksum is compared as hex in either case.
	e.opt.FontSHA256 = strings.ToUpper(hex.EncodeToString(sha256Of(goregular.TTF)))
	if _, err := Run(e.opt, e.src, e.fr); err != nil {
		t.Errorf("Run with an upper-case checksum: %v", err)
	}
}

func sha256Of(b []byte) []byte {
	sum := sha256.Sum256(b)
	return sum[:]
}

func TestRunRejectsBadOptions(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Options)
	}{
		{"no raw tag", func(o *Options) { o.RawTag = "" }},
		{"raw tag names a path", func(o *Options) { o.RawTag = "../raw" }},
		{"no lang dir", func(o *Options) { o.LangDir = "" }},
		{"lang dir names a path", func(o *Options) { o.LangDir = "a/b" }},
		{"no build time", func(o *Options) { o.BuiltAt = time.Time{} }},
		{"build time zip cannot store", func(o *Options) { o.BuiltAt = time.Unix(0, 0) }},
		{"no tw root", func(o *Options) { o.TWRoot = "" }},
		{"no out dir", func(o *Options) { o.OutDir = "" }},
		{"missing font", func(o *Options) { o.FontPath = filepath.Join(o.OutDir, "nope.ttf") }},
		{"missing license", func(o *Options) { o.LicensePath = filepath.Join(o.OutDir, "nope") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t, map[string]string{"A.json": `{"dataList":[]}`}, nil)
			tt.edit(&e.opt)
			if _, err := Run(e.opt, e.src, e.fr); err == nil {
				t.Fatal("Run succeeded")
			}
			if _, err := os.Stat(filepath.Join(e.opt.TWRoot, dataDir)); err == nil {
				t.Error("the pack was written although the options are bad")
			}
		})
	}
}

func TestRunNeedsKRFiles(t *testing.T) {
	e := newEnv(t, nil, nil)
	if _, err := Run(e.opt, e.src, e.fr); err == nil {
		t.Error("Run with an empty KR pack succeeded: it would delete the whole pack")
	}
}

func TestMissingGlyphs(t *testing.T) {
	kr := `{"dataList":[{"id":1,"content":"가\ue000\n나"},{"id":2,"content":"다"}]}`
	e := newEnv(t, map[string]string{"A.json": kr}, nil)
	e.save(
		rec("A.json", "/dataList/0/content", "가\ue000\n나", "A\ue000\nB\ue000"),
		rec("A.json", "/dataList/1/content", "다", "\ue001\tC"),
	)
	rep := e.run()
	// Go Regular has Latin letters, but nothing in the private use area; the
	// newline and the tab are not glyphs.
	if want := []rune{0xE000, 0xE001}; !slices.Equal(rep.MissingGlyphs, want) {
		t.Errorf("MissingGlyphs = %U, want %U", rep.MissingGlyphs, want)
	}
}

func TestMissingGlyphsNone(t *testing.T) {
	e := newEnv(t, map[string]string{"A.json": `{"dataList":[{"id":1,"content":"가"}]}`}, nil)
	e.save(rec("A.json", "/dataList/0/content", "가", "Hello"))
	rep := e.run()
	if rep.MissingGlyphs == nil || len(rep.MissingGlyphs) != 0 {
		t.Errorf("MissingGlyphs = %#v, want an empty slice", rep.MissingGlyphs)
	}
}

func TestBuildWritesBuildRecordAndReport(t *testing.T) {
	kr := `{"dataList":[{"id":1,"content":"가\ue000"},{"id":2,"content":"나"}]}`
	e := newEnv(t, map[string]string{"A.json": kr}, nil)
	e.save(rec("A.json", "/dataList/0/content", "가\ue000", "甲\ue000"))
	rep := e.run()

	var got struct {
		RawTag      string            `json:"raw_tag"`
		RawToken    string            `json:"raw_token"`
		RulesCommit string            `json:"rules_commit"`
		BuiltAt     string            `json:"built_at"`
		Upstream    map[string]string `json:"upstream"`
		Report      *Report           `json:"report"`
	}
	b, err := os.ReadFile(filepath.Join(e.opt.TWRoot, ".meta", "build.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.RawTag != testTag || got.RawToken != testToken || got.RulesCommit != "abc1234" || got.BuiltAt != "2026-10-31T12:30:45Z" {
		t.Errorf("build.json = %+v", got)
	}
	if !reflect.DeepEqual(got.Report, rep) {
		t.Errorf("build.json report = %+v, want %+v", got.Report, rep)
	}
	if !reflect.DeepEqual(got.Upstream, e.opt.Upstream) {
		t.Errorf("build.json upstream = %v", got.Upstream)
	}

	md, err := os.ReadFile(filepath.Join(e.opt.TWRoot, ".meta", "reports", testTag+".md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(md)
	for _, want := range []string{testTag, "claude", "U+E000", "U+7532"} {
		if !strings.Contains(text, want) {
			t.Errorf("report lacks %q:\n%s", want, text)
		}
	}
	if ReportPath(e.opt) != filepath.Join(e.opt.TWRoot, ".meta", "reports", testTag+".md") {
		t.Errorf("ReportPath = %s", ReportPath(e.opt))
	}
}

// build.json records the sha256 of the zips, by file name, so that publish can
// tell the zips of this build from those of another.
func TestBuildRecordsZipChecksums(t *testing.T) {
	e := newEnv(t, map[string]string{"A.json": `{"dataList":[{"id":1,"content":"가"}]}`}, nil)
	e.save(rec("A.json", "/dataList/0/content", "가", "甲"))
	e.run()

	text, complete := ZipPaths(e.opt)
	want := map[string]string{}
	for _, name := range []string{text, complete} {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		want[filepath.Base(name)] = hex.EncodeToString(sha256Of(data))
	}
	if len(want) != 2 {
		t.Fatalf("the two zips have the same name: %v", want)
	}

	info, err := ReadInfo(e.opt.TWRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(info.Zips, want) {
		t.Errorf("Zips = %v, want %v", info.Zips, want)
	}

	// The key of the file.
	b, err := os.ReadFile(filepath.Join(e.opt.TWRoot, ".meta", "build.json"))
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		Zips map[string]string `json:"zips"`
	}
	if err := json.Unmarshal(b, &raw); err != nil || !reflect.DeepEqual(raw.Zips, want) {
		t.Errorf("build.json zips = %v, %v, want %v", raw.Zips, err, want)
	}

	// The build is reproducible, so are the checksums.
	e.run()
	if again, err := ReadInfo(e.opt.TWRoot); err != nil || !reflect.DeepEqual(again.Zips, want) {
		t.Errorf("Zips after a second build = %v, %v, want %v", again.Zips, err, want)
	}
}

// ReadInfo reads what Run wrote; a worktree that was never built is
// ErrNotBuilt, a file that cannot be read is another error.
func TestReadInfo(t *testing.T) {
	e := newEnv(t, map[string]string{"A.json": `{"dataList":[{"id":1,"content":"가"}]}`}, nil)
	if _, err := ReadInfo(e.opt.TWRoot); !errors.Is(err, ErrNotBuilt) {
		t.Errorf("ReadInfo of a worktree that was not built: err = %v, want ErrNotBuilt", err)
	}

	rep := e.run()
	info, err := ReadInfo(e.opt.TWRoot)
	if err != nil {
		t.Fatal(err)
	}
	if info.RawTag != testTag || info.RawToken != testToken || info.RulesCommit != "abc1234" || info.BuiltAt != "2026-10-31T12:30:45Z" ||
		!reflect.DeepEqual(info.Upstream, e.opt.Upstream) || !reflect.DeepEqual(&info.Report, rep) {
		t.Errorf("ReadInfo = %+v, want the inputs and the report of the build (%+v)", info, rep)
	}

	mustWrite(t, filepath.Join(e.opt.TWRoot, ".meta", "build.json"), []byte("{"))
	if _, err := ReadInfo(e.opt.TWRoot); err == nil || errors.Is(err, ErrNotBuilt) {
		t.Errorf("ReadInfo of a broken file: err = %v, want an error that is not ErrNotBuilt", err)
	}
}

func TestReportMarkdown(t *testing.T) {
	rep := &Report{
		Origins:            map[string]int{"paratranz": 12, "claude": 3},
		Pending:            4,
		NeedsReview:        2,
		StructuralFallback: 1,
		Unaligned:          5,
		FieldRuleOverride:  6,
		Warnings:           map[string]int{"length-ratio": 7, "hangul": 1},
		MissingGlyphs:      []rune{'`', 0xE000, '字'},
	}
	opt := Options{RawTag: testTag, RawToken: testToken, RulesCommit: "abc1234", BuiltAt: testBuiltAt}
	got := string(Markdown(opt, rep))
	for _, want := range []string{
		"# Build report raw-20261029\n",
		"| claude | 3 |\n| paratranz | 12 |\n", // sorted by name
		"| hangul | 1 |\n| length-ratio | 7 |\n",
		"- U+0060\n", "- U+E000\n", "- U+5B57 `字`\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("report lacks %q:\n%s", want, got)
		}
	}
	for label, n := range map[string]int{"Pending": 4, "Needs review": 2, "Structural fallback": 1, "Unaligned": 5, "Field rule override": 6} {
		if !strings.Contains(got, label) || !strings.Contains(got, "| "+label+" | "+strconv.Itoa(n)+" |") {
			t.Errorf("report lacks the row %q = %d:\n%s", label, n, got)
		}
	}
	if strings.Contains(got, "```") {
		t.Errorf("a backtick must not be put in a code span:\n%s", got)
	}
	if again := string(Markdown(opt, rep)); again != got {
		t.Error("Markdown is not deterministic")
	}
	if !strings.HasSuffix(got, "\n") || strings.HasSuffix(got, "\n\n") {
		t.Errorf("report must end in exactly one newline: %q", got)
	}
}

// The pinned font is the file lct.yaml says it is, the licence is next to it,
// and the font has what the game text needs. It reads the real 24 MB file but
// parses only the tables it uses.
func TestPinnedFont(t *testing.T) {
	cfg, err := config.Load("../..")
	if err != nil {
		t.Fatal(err)
	}
	fontPath := filepath.Join("..", "..", filepath.FromSlash(cfg.Font.Path))
	data, err := os.ReadFile(fontPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(sha256Of(data)); got != cfg.Font.SHA256 || len(cfg.Font.SHA256) != 64 {
		t.Errorf("sha256 of %s = %s, lct.yaml pins %q", cfg.Font.Path, got, cfg.Font.SHA256)
	}
	if cfg.Font.Version == "" {
		t.Error("lct.yaml pins no font version")
	}
	if ofl, err := os.ReadFile(filepath.Join(filepath.Dir(fontPath), "OFL.txt")); err != nil || !strings.Contains(string(ofl), "SIL Open Font License") {
		t.Errorf("OFL.txt next to the font: %v", err)
	}

	f, err := sfnt.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	var b sfnt.Buffer
	for _, r := range "臺灣繁體中文的 [Bleed]，「」…" {
		if idx, err := f.GlyphIndex(&b, r); err != nil || idx == 0 {
			t.Errorf("the pinned font has no glyph for %q (%v)", r, err)
		}
	}
	if idx, _ := f.GlyphIndex(&b, 0xE000); idx != 0 {
		t.Error("the pinned font has a glyph for U+E000: MissingGlyphs would be meaningless")
	}
}
