package build

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/tool-jx3/limbus-translate/internal/itemid"
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
	"unicode/utf8"

	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"

	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/existing"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/plan"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/translate"
	"github.com/tool-jx3/limbus-translate/internal/units"
	"github.com/tool-jx3/limbus-translate/internal/validate"
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
	if md := e.readReport(); !strings.Contains(md, "| Field rule override | 2 |") {
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
	md := e.readReport()
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

// writeTranslation records a translation run in the tw worktree, as lct
// translate does.
func writeTranslation(t *testing.T, e *testEnv, res translate.Result) {
	t.Helper()
	if err := store.Open(e.opt.TWRoot).WriteMetaJSON(translate.MetaFile, res); err != nil {
		t.Fatal(err)
	}
}

// jsonKeys decodes a JSON file (Info/version.json, build.json) into a map, to
// tell a field that is absent from one that is empty.
func jsonKeys(t *testing.T, name string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return m
}

// The model and the cost of the translation go into build.json and
// version.json, but only when the translation is of the inputs of this
// build: the same raw tag and the same rules.
func TestBuildRecordsModelAndCost(t *testing.T) {
	e := newEnv(t, map[string]string{"A.json": `{"dataList":[{"id":1,"content":"가"}]}`}, nil)
	e.save(rec("A.json", "/dataList/0/content", "가", "甲"))
	buildJSON := filepath.Join(e.opt.TWRoot, ".meta", "build.json")
	versionJSON := func() string {
		return filepath.Join(e.opt.TWRoot, dataDir, "Lang", e.opt.LangDir, "Info", "version.json")
	}
	same := translate.Result{
		Model: "model-x", Runs: 3, CostUSD: 1.25, Stopped: translate.StopDone,
		PendingBefore: 9, RawTag: testTag, RulesCommit: "abc1234",
	}

	writeTranslation(t, e, same)
	e.run()
	for _, name := range []string{buildJSON, versionJSON()} {
		m := jsonKeys(t, name)
		if m["model"] != "model-x" || m["cost_usd"] != 1.25 {
			t.Errorf("%s: model, cost_usd = %v, %v, want model-x, 1.25", filepath.Base(name), m["model"], m["cost_usd"])
		}
	}
	info, err := ReadInfo(e.opt.TWRoot)
	if err != nil {
		t.Fatal(err)
	}
	if info.Model != "model-x" || info.CostUSD != 1.25 {
		t.Errorf("ReadInfo: Model, CostUSD = %q, %v, want model-x, 1.25", info.Model, info.CostUSD)
	}

	for name, other := range map[string]translate.Result{
		"another raw tag":        {Model: "model-x", CostUSD: 1.25, RawTag: "raw-20260101", RulesCommit: "abc1234"},
		"another rules":          {Model: "model-x", CostUSD: 1.25, RawTag: testTag, RulesCommit: "def5678"},
		"dirty rules":            {Model: "model-x", CostUSD: 1.25, RawTag: testTag, RulesCommit: "abc1234+dirty"},
		"neither tag nor commit": {Model: "model-x", CostUSD: 1.25},
	} {
		writeTranslation(t, e, other)
		e.run()
		for _, f := range []string{buildJSON, versionJSON()} {
			m := jsonKeys(t, f)
			if _, ok := m["model"]; ok {
				t.Errorf("%s, %s: has model %v", name, filepath.Base(f), m["model"])
			}
			if _, ok := m["cost_usd"]; ok {
				t.Errorf("%s, %s: has cost_usd %v", name, filepath.Base(f), m["cost_usd"])
			}
		}
	}

	// No translation at all is the common case (the first build, a dry run).
	if err := os.Remove(filepath.Join(e.opt.TWRoot, ".meta", translate.MetaFile)); err != nil {
		t.Fatal(err)
	}
	e.run()
	if _, ok := jsonKeys(t, buildJSON)["model"]; ok {
		t.Error("build.json has a model although nothing was translated")
	}

	// A translation record that cannot be read is an error, not a build
	// without a model.
	mustWrite(t, filepath.Join(e.opt.TWRoot, ".meta", translate.MetaFile), []byte("{"))
	if _, err := Run(e.opt, e.src, e.fr); err == nil {
		t.Error("Run with a broken translate.json succeeded")
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
	info := Info{RawTag: testTag, RawToken: testToken, RulesCommit: "abc1234", BuiltAt: "2026-10-31T12:30:45Z", Report: *rep}
	got := string(Markdown(opt, info, nil, nil))
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
	if again := string(Markdown(opt, info, nil, nil)); again != got {
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

// writePlan leaves a plan summary in the tw worktree, as lct plan does.
func writePlan(t *testing.T, e *testEnv, sum plan.Summary) {
	t.Helper()
	if err := store.Open(e.opt.TWRoot).WriteMetaJSON(plan.File, sum); err != nil {
		t.Fatal(err)
	}
}

// readReport reads the markdown report of the last build.
func (e *testEnv) readReport() string {
	e.t.Helper()
	b, err := os.ReadFile(ReportPath(e.opt))
	if err != nil {
		e.t.Fatal(err)
	}
	return string(b)
}

// section returns the part of md between the heading and the next heading of
// the same level, heading included.
func section(t *testing.T, md, heading string) string {
	t.Helper()
	i := strings.Index(md, "\n"+heading+"\n")
	if i < 0 {
		t.Fatalf("report lacks the heading %q:\n%s", heading, md)
	}
	rest := md[i+1:]
	if j := strings.Index(rest[len(heading):], "\n## "); j >= 0 {
		return rest[:len(heading)+j]
	}
	return rest
}

// subsection returns the part of md from the level-three heading to the next
// heading of level three or two, heading included.
func subsection(t *testing.T, md, heading string) string {
	t.Helper()
	i := strings.Index(md, "\n"+heading+"\n")
	if i < 0 {
		t.Fatalf("report lacks the heading %q:\n%s", heading, md)
	}
	rest := md[i+1+len(heading):]
	if j := strings.Index(rest, "\n##"); j >= 0 {
		rest = rest[:j]
	}
	return heading + rest
}

const twoUnitsKR = `{"dataList":[{"id":1,"content":"가"},{"id":2,"content":"나"}]}`

// jobSummary renders the report for the summary page of a run from what the
// last build left in the store.
func (e *testEnv) jobSummary() string {
	e.t.Helper()
	info, err := ReadInfo(e.opt.TWRoot)
	if err != nil {
		e.t.Fatal(err)
	}
	sum, err := ReadPlan(e.opt.TWRoot, info.RawTag)
	if err != nil {
		e.t.Fatal(err)
	}
	return string(JobSummary(e.opt, info, sum, e.pending()))
}

// The report has the two parts of spec 14, in this order: what needs a person,
// then the information. The archived report links the compare page of the KR
// text; the compare page of tw is for the summary of the run only (see
// TestArchivedReportDoesNotDependOnTW).
func TestReportHasBothSections(t *testing.T) {
	e := newEnv(t, map[string]string{"A.json": twoUnitsKR}, nil)
	e.opt.RepoURL = "https://github.com/tool-jx3/limbus-translate"
	e.opt.PrevTWCommit = "0123abc"
	stuck := rec("A.json", "/dataList/1/content", "나", "B")
	stuck.Origin, stuck.Flags = store.OriginFallback, []string{store.FlagNeedsReview}
	e.save(rec("A.json", "/dataList/0/content", "가", "甲"), stuck)
	if err := store.Open(e.opt.TWRoot).SavePending([]store.PendingItem{{
		Key: "A.json|/dataList/1/content", Reason: "new", Attempts: 3, Flags: []string{store.FlagNeedsReview},
		LastError: "id-mismatch: [Bleed] is missing",
	}}); err != nil {
		t.Fatal(err)
	}
	writePlan(t, e, plan.Summary{
		RawTag: testTag, PrevRawTag: "raw-20261022", RulesCommit: "abc1234",
		Changes: plan.Changes{Unchanged: 90, Moved: 3, Changed: 4, New: 5, Deleted: 6, FilesAdded: []string{"New.json"}, FilesShifted: []string{"A.json"}},
		Adjustments: plan.AdjustmentSummary{
			Moved:   []plan.MovedKey{{From: "A.json|/dataList/5/content", To: "A.json|/dataList/6/content"}},
			Expired: []string{"A.json|/dataList/9/content"},
		},
	})
	e.run()

	md := e.readReport()
	needs, info := strings.Index(md, "\n## 需要你處理\n"), strings.Index(md, "\n## 資訊\n")
	if needs < 0 || info < 0 || needs > info {
		t.Fatalf("sections: 需要你處理 at %d, 資訊 at %d, want both, in this order:\n%s", needs, info, md)
	}
	todo, rest := section(t, md, "## 需要你處理"), section(t, md, "## 資訊")
	for _, want := range []string{
		"A.json|/dataList/1/content", "id-mismatch: [Bleed] is missing", // the unit that ran out of attempts and why
		"A.json|/dataList/9/content", // the adjustment that expired
	} {
		if !strings.Contains(todo, want) {
			t.Errorf("the part that needs a person lacks %q:\n%s", want, todo)
		}
	}
	for _, want := range []string{
		"https://github.com/tool-jx3/limbus-translate/compare/raw-20261022...raw-20261029",
		"新增 5、修改 4、刪除 6", "New.json", // what the new snapshot changed
		"沿用原本的譯文：3 個單位",                                            // the units that only moved
		"A.json|/dataList/5/content", "A.json|/dataList/6/content", // the adjustment whose key moved
		testToken,
	} {
		if !strings.Contains(rest, want) {
			t.Errorf("the information lacks %q:\n%s", want, rest)
		}
	}
	// What is information is not in the part for the person.
	for _, not := range []string{"compare/", "A.json|/dataList/5/content"} {
		if strings.Contains(todo, not) {
			t.Errorf("the part that needs a person holds %q:\n%s", not, todo)
		}
	}

	// The compare page of tw is not in the archived report: where tw stands
	// before a build changes with every publish.
	for _, not := range []string{"...tw", "0123abc"} {
		if strings.Contains(md, not) {
			t.Errorf("the archived report holds %q:\n%s", not, md)
		}
	}

	// The summary of the run is that report and the compare page of tw.
	js := e.jobSummary()
	for _, want := range []string{
		"https://github.com/tool-jx3/limbus-translate/compare/raw-20261022...raw-20261029",
		"https://github.com/tool-jx3/limbus-translate/compare/0123abc...tw",
		"## 需要你處理", "## 資訊", "A.json|/dataList/9/content",
	} {
		if !strings.Contains(js, want) {
			t.Errorf("the summary lacks %q:\n%s", want, js)
		}
	}

	// Without a repo URL there are no links; without a previous tw commit only
	// the one of the KR text.
	e.opt.RepoURL = ""
	e.run()
	if md := e.readReport(); strings.Contains(md, "/compare/") {
		t.Errorf("a report without a repo URL has compare links:\n%s", md)
	}
	if js := e.jobSummary(); strings.Contains(js, "/compare/") {
		t.Errorf("a summary without a repo URL has compare links:\n%s", js)
	}
	e.opt.RepoURL, e.opt.PrevTWCommit = "https://github.com/tool-jx3/limbus-translate/", ""
	e.run()
	if md := e.readReport(); !strings.Contains(md, "limbus-translate/compare/raw-20261022...raw-20261029") {
		t.Errorf("a report lacks the link of the KR text:\n%s", md)
	}
	if js := e.jobSummary(); !strings.Contains(js, "limbus-translate/compare/raw-20261022...raw-20261029") || strings.Contains(js, "...tw") {
		t.Errorf("a summary without a previous tw commit has only the link of the KR text:\n%s", js)
	}
}

// What is committed to tw does not depend on where tw stands before the build:
// that moves with every publish, and a report that followed it would change the
// branch on every build, so a Release would follow even when the pack did not
// change. Only the summary of the run links the compare page of tw.
func TestArchivedReportDoesNotDependOnTW(t *testing.T) {
	e := newEnv(t, map[string]string{"A.json": twoUnitsKR}, nil)
	e.save(rec("A.json", "/dataList/0/content", "가", "甲"))
	e.opt.RepoURL = "https://github.com/tool-jx3/limbus-translate"
	writePlan(t, e, plan.Summary{RawTag: testTag, PrevRawTag: "raw-20261022", Changes: plan.Changes{New: 1}})

	// The pack, the report, build.json and the zips.
	committed := func() map[string]string { return treeFiles(t, e.opt) }

	e.opt.PrevTWCommit = ""
	e.run()
	first, firstSummary := committed(), e.jobSummary()
	e.opt.PrevTWCommit = "1111111"
	e.run()
	second, secondSummary := committed(), e.jobSummary()
	e.opt.PrevTWCommit = "2222222"
	e.run()
	third, thirdSummary := committed(), e.jobSummary()

	if d := diffKeys(first, second); len(d) > 0 {
		t.Errorf("a build after another tw commit changed %v", d)
	}
	if d := diffKeys(second, third); len(d) > 0 {
		t.Errorf("a build after yet another tw commit changed %v", d)
	}
	for name, content := range third {
		if strings.Contains(content, "1111111") || strings.Contains(content, "2222222") || strings.Contains(content, "...tw") {
			t.Errorf("%s holds the state of tw before the build", name)
		}
	}
	// The summary does follow it.
	if strings.Contains(firstSummary, "...tw") ||
		!strings.Contains(secondSummary, "/compare/1111111...tw") || strings.Contains(secondSummary, "2222222") ||
		!strings.Contains(thirdSummary, "/compare/2222222...tw") || strings.Contains(thirdSummary, "1111111") {
		t.Errorf("the summaries do not link the compare page of tw from where it stood:\n%s\n%s\n%s", firstSummary, secondSummary, thirdSummary)
	}
}

// With nothing for a person to do, the part says so.
func TestReportWithNothingToDo(t *testing.T) {
	e := newEnv(t, map[string]string{"A.json": twoUnitsKR}, nil)
	e.save(rec("A.json", "/dataList/0/content", "가", "甲"), rec("A.json", "/dataList/1/content", "나", "乙"))
	e.run()
	md := e.readReport()
	todo := section(t, md, "## 需要你處理")
	if !strings.Contains(todo, "沒有") || strings.Contains(todo, "- ") {
		t.Errorf("the part that needs a person should say there is nothing:\n%s", todo)
	}
	if strings.Index(md, "## 需要你處理") > strings.Index(md, "## 資訊") {
		t.Errorf("sections out of order:\n%s", md)
	}
}

// A plan of another snapshot says nothing about this build.
func TestReportIgnoresAPlanOfAnotherSnapshot(t *testing.T) {
	e := newEnv(t, map[string]string{"A.json": twoUnitsKR}, nil)
	e.save(rec("A.json", "/dataList/0/content", "가", "甲"))
	e.opt.RepoURL = "https://github.com/tool-jx3/limbus-translate"
	writePlan(t, e, plan.Summary{
		RawTag: "raw-20260101", PrevRawTag: "raw-20251225",
		Adjustments: plan.AdjustmentSummary{Expired: []string{"A.json|/dataList/9/content"}},
		NewSpeakers: []string{"someone"},
	})
	e.run()
	md := e.readReport()
	for _, not := range []string{"A.json|/dataList/9/content", "someone", "raw-20251225"} {
		if strings.Contains(md, not) {
			t.Errorf("the report holds %q of another snapshot's plan:\n%s", not, md)
		}
	}
}

// The inputs of the build and of the translation are information.
// A frozen file whose KR changed keeps its translation, so it is not among the
// units that need translating again; the count never goes below zero.
func TestReportCountsOnlyTheUnitsThatAreRetranslated(t *testing.T) {
	e := newEnv(t, map[string]string{"A.json": twoUnitsKR}, nil)
	e.save(rec("A.json", "/dataList/0/content", "가", "甲"))
	for _, tc := range []struct {
		name    string
		changed int
		frozen  []string
		want    string
	}{
		{"none frozen", 4, nil, "因 KR 變更需要重翻：4 個單位"},
		{"some frozen", 5, []string{"BgmLyrics/L1.json|/dataList/0/content", "BgmLyrics/L1.json|/dataList/1/content"}, "因 KR 變更需要重翻：3 個單位"},
		{"all frozen", 2, []string{"BgmLyrics/L1.json|/dataList/0/content", "BgmLyrics/L1.json|/dataList/1/content"}, "因 KR 變更需要重翻：0 個單位"},
		{"more frozen than changed", 1, []string{"a|/x", "b|/y"}, "因 KR 變更需要重翻：0 個單位"},
	} {
		writePlan(t, e, plan.Summary{
			RawTag: testTag, PrevRawTag: "raw-20261022", RulesCommit: "abc1234",
			Changes: plan.Changes{Changed: tc.changed}, FrozenChanged: tc.frozen,
		})
		e.run()
		if rest := section(t, e.readReport(), "## 資訊"); !strings.Contains(rest, tc.want) {
			t.Errorf("%s: the information lacks %q:\n%s", tc.name, tc.want, rest)
		}
	}
}

func TestReportShowsInputsAndTranslation(t *testing.T) {
	e := newEnv(t, map[string]string{"A.json": twoUnitsKR}, nil)
	e.save(rec("A.json", "/dataList/0/content", "가", "甲"))
	writeTranslation(t, e, translate.Result{
		Model: "model-x", Runs: 3, CostUSD: 1.25, Stopped: translate.StopCost,
		RawTag: testTag, RulesCommit: "abc1234",
	})
	e.run()
	rest := section(t, e.readReport(), "## 資訊")
	for _, want := range []string{testTag, testToken, "abc1234", "model-x", "$1.25", "cost", "owner/pz@11"} {
		if !strings.Contains(rest, want) {
			t.Errorf("the information lacks %q:\n%s", want, rest)
		}
	}
	info, err := ReadInfo(e.opt.TWRoot)
	if err != nil {
		t.Fatal(err)
	}
	if info.Stopped != translate.StopCost || info.StopError != "" {
		t.Errorf("Info.Stopped, StopError = %q, %q, want %q, empty", info.Stopped, info.StopError, translate.StopCost)
	}

	// A translation that failed: its error is in build.json and in the report.
	writeTranslation(t, e, translate.Result{
		Model: "model-x", CostUSD: 0.5, Stopped: translate.StopError, Error: "exit status 1: boom",
		RawTag: testTag, RulesCommit: "abc1234",
	})
	e.run()
	if info, err := ReadInfo(e.opt.TWRoot); err != nil || info.Stopped != translate.StopError || info.StopError != "exit status 1: boom" {
		t.Errorf("Info = %+v, %v, want the error of the translation", info, err)
	}
	if rest := section(t, e.readReport(), "## 資訊"); !strings.Contains(rest, "exit status 1: boom") {
		t.Errorf("the information lacks the error of the translation:\n%s", rest)
	}

	// A translation of other inputs says nothing about this build.
	writeTranslation(t, e, translate.Result{
		Model: "model-x", Stopped: translate.StopError, Error: "old failure", RawTag: "raw-20260101", RulesCommit: "abc1234",
	})
	e.run()
	if info, _ := ReadInfo(e.opt.TWRoot); info.Stopped != "" || info.StopError != "" {
		t.Errorf("Info = %+v, want no stop reason of a translation of another snapshot", info)
	}
	if strings.Contains(e.readReport(), "old failure") {
		t.Error("the report holds the failure of another snapshot's translation")
	}
}

func appendNotes(t *testing.T, e *testEnv, texts ...string) {
	t.Helper()
	var notes []store.Note
	for _, text := range texts {
		notes = append(notes, store.Note{Key: "A.json|/dataList/0/content", Batch: "b1", KRSha: units.KRSha("가"), Model: "m", Note: text})
	}
	if _, err := store.Open(e.opt.TWRoot).AppendNotes(notes); err != nil {
		t.Fatal(err)
	}
}

// The notes of the translator are in the report once: a build reports those
// that came after the last build.
func TestReportListsNewNotesOnce(t *testing.T) {
	e := newEnv(t, map[string]string{"A.json": twoUnitsKR}, nil)
	e.save(rec("A.json", "/dataList/0/content", "가", "甲"))
	appendNotes(t, e, "note one", "note two")

	rep := e.run()
	if len(rep.Notes) != 2 || rep.Notes[0].Note != "note one" || rep.Notes[1].Note != "note two" {
		t.Fatalf("Notes = %+v, want the two notes", rep.Notes)
	}
	if info, err := ReadInfo(e.opt.TWRoot); err != nil || info.NotesSeen != 2 {
		t.Fatalf("NotesSeen = %d, %v, want 2", info.NotesSeen, err)
	}
	md := e.readReport()
	for _, want := range []string{"note one", "note two", "A.json|/dataList/0/content"} {
		if !strings.Contains(md, want) {
			t.Errorf("the first report lacks %q:\n%s", want, md)
		}
	}

	appendNotes(t, e, "note three")
	rep = e.run()
	if len(rep.Notes) != 1 || rep.Notes[0].Note != "note three" {
		t.Fatalf("Notes = %+v, want only the new note", rep.Notes)
	}
	info, err := ReadInfo(e.opt.TWRoot)
	if err != nil || info.NotesSeen != 3 {
		t.Fatalf("NotesSeen = %d, %v, want 3", info.NotesSeen, err)
	}
	if !reflect.DeepEqual(info.Report.Notes, rep.Notes) {
		t.Errorf("build.json notes = %+v, want %+v", info.Report.Notes, rep.Notes)
	}
	md = e.readReport()
	if !strings.Contains(md, "note three") || strings.Contains(md, "note one") || strings.Contains(md, "note two") {
		t.Errorf("the second report should list only the new note:\n%s", md)
	}

	// Nothing new: no notes, the count stays.
	rep = e.run()
	if len(rep.Notes) != 0 {
		t.Errorf("Notes = %+v, want none", rep.Notes)
	}
	if info, _ := ReadInfo(e.opt.TWRoot); info.NotesSeen != 3 {
		t.Errorf("NotesSeen = %d, want 3", info.NotesSeen)
	}

	// A build record that counts more notes than there are (the file was
	// replaced) starts again from the first note.
	if err := os.Remove(filepath.Join(e.opt.TWRoot, ".meta", "notes.jsonl")); err != nil {
		t.Fatal(err)
	}
	appendNotes(t, e, "fresh")
	if rep = e.run(); len(rep.Notes) != 1 || rep.Notes[0].Note != "fresh" {
		t.Errorf("Notes = %+v, want the one note of the new file", rep.Notes)
	}
}

// Twelve units warn of kana: the report keeps the first ten keys of them,
// sorted, and the count of them all.
func TestReportWarningSamples(t *testing.T) {
	var entries []string
	var recs []store.Record
	for i := range 12 {
		kr := "가" + string(rune('가'+i))
		entries = append(entries, `{"id":`+strconv.Itoa(i)+`,"content":"`+kr+`"}`)
		recs = append(recs, rec("A.json", "/dataList/"+strconv.Itoa(i)+"/content", kr, "甲か"))
	}
	e := newEnv(t, map[string]string{"A.json": `{"dataList":[` + strings.Join(entries, ",") + `]}`}, nil)
	e.save(recs...)
	rep := e.run()

	if rep.Warnings["kana"] != 12 {
		t.Fatalf("Warnings = %v, want 12 kana", rep.Warnings)
	}
	var all []string
	for _, r := range recs {
		all = append(all, r.Key)
	}
	slices.Sort(all)
	got := rep.WarningSamples["kana"]
	if !slices.Equal(got, all[:10]) {
		t.Errorf("WarningSamples[kana] = %v, want the first 10 of the sorted keys %v", got, all[:10])
	}
	if len(rep.WarningSamples) != 1 {
		t.Errorf("WarningSamples = %v, want only kana", rep.WarningSamples)
	}
	warnings := section(t, e.readReport(), "## 資訊")
	if !strings.Contains(warnings, "kana") || !strings.Contains(warnings, "12") || !strings.Contains(warnings, got[0]) {
		t.Errorf("the report does not show the count and the samples of the kana warnings:\n%s", warnings)
	}

	// Fewer than ten: all of them. A unit with two issues of one code is one
	// sample.
	e2 := newEnv(t, map[string]string{"A.json": `{"dataList":[{"id":1,"content":"가"}]}`}, nil)
	e2.save(rec("A.json", "/dataList/0/content", "가", "甲か乙か"))
	rep = e2.run()
	if got := rep.WarningSamples["kana"]; !slices.Equal(got, []string{"A.json|/dataList/0/content"}) {
		t.Errorf("WarningSamples[kana] = %v, want the one key once", got)
	}
}

func routingRules() *rules.Rules {
	return &rules.Rules{Routing: rules.Routing{
		Families: []rules.FamilyRule{{ID: "alpha", Nodes: []string{"A*"}}, {ID: "other"}},
		Priority: []string{"alpha", "other"},
	}}
}

// The same KR text in one family and field with different translations is one
// group; the same text in another family, or with one translation, is not.
func TestReportInconsistentRenderings(t *testing.T) {
	e := newEnv(t, map[string]string{
		"A.json": `{"dataList":[{"id":1,"content":"가"},{"id":2,"content":"가"},{"id":3,"content":"가"},{"id":4,"content":"나"},{"id":5,"content":"나"}]}`,
		"B.json": `{"dataList":[{"id":1,"content":"가"}]}`,
	}, nil)
	e.opt.Rules = routingRules()
	other := rec("A.json", "/dataList/4/content", "나", "丙")
	other.Origin = store.OriginParatranz
	e.save(
		rec("A.json", "/dataList/0/content", "가", "甲"),
		rec("A.json", "/dataList/1/content", "가", "乙"),
		rec("A.json", "/dataList/2/content", "가", "甲"),
		rec("A.json", "/dataList/3/content", "나", "丙"), // one translation twice: consistent
		other,
		rec("B.json", "/dataList/0/content", "가", "丁"), // another family: a group of its own
	)
	rep := e.run()

	want := []Inconsistency{{
		Family: "alpha", Field: "content", KR: "가",
		ZH:     []string{"甲", "乙"},
		Keys:   []string{"A.json|/dataList/0/content", "A.json|/dataList/1/content"},
		Counts: []int{2, 1}, // 甲 twice, 乙 once
		Units:  3,
	}}
	if !reflect.DeepEqual(rep.Inconsistent, want) {
		t.Errorf("Inconsistent = %+v, want %+v", rep.Inconsistent, want)
	}
	md := section(t, e.readReport(), "## 資訊")
	for _, w := range []string{"甲", "乙", "A.json|/dataList/1/content"} {
		if !strings.Contains(md, w) {
			t.Errorf("the report lacks %q in the list of different renderings:\n%s", w, md)
		}
	}

	// Without the rules every file is one family: B.json joins the group.
	e.opt.Rules = nil
	rep = e.run()
	if len(rep.Inconsistent) != 1 || rep.Inconsistent[0].Units != 4 || len(rep.Inconsistent[0].ZH) != 3 || rep.Inconsistent[0].Family != "" ||
		!slices.Equal(rep.Inconsistent[0].Counts, []int{2, 1, 1}) {
		t.Errorf("Inconsistent without rules = %+v, want one group of 4 units and 3 renderings, counted 2, 1 and 1", rep.Inconsistent)
	}
}

// A group lists at most ten translations; the units of the others count in
// Units and in no count of a translation.
func TestReportInconsistentCountsTheListedRenderings(t *testing.T) {
	var entries []string
	var recs []store.Record
	for i := range 12 {
		entries = append(entries, `{"id":`+strconv.Itoa(i)+`,"content":"가"}`)
		recs = append(recs, rec("A.json", "/dataList/"+strconv.Itoa(i)+"/content", "가", "譯"+strconv.Itoa(i)))
	}
	entries = append(entries, `{"id":12,"content":"가"}`)
	recs = append(recs, rec("A.json", "/dataList/12/content", "가", "譯0"))
	e := newEnv(t, map[string]string{"A.json": `{"dataList":[` + strings.Join(entries, ",") + `]}`}, nil)
	e.save(recs...)
	rep := e.run()
	if len(rep.Inconsistent) != 1 {
		t.Fatalf("Inconsistent = %+v, want one group", rep.Inconsistent)
	}
	g := rep.Inconsistent[0]
	want := []int{2, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	if g.Units != 13 || len(g.ZH) != 10 || !slices.Equal(g.Counts, want) {
		t.Errorf("group = %+v, want 13 units, 10 translations counted %v", g, want)
	}
}

// Copies, stand-ins and units that are pending are not translations of the
// text: they make no inconsistency. The biggest groups come first and there
// are at most twenty.
func TestReportInconsistentIsBounded(t *testing.T) {
	var entries []string
	var recs []store.Record
	add := func(kr, zh string, origin string) {
		i := len(entries)
		entries = append(entries, `{"id":`+strconv.Itoa(i)+`,"content":"`+kr+`"}`)
		r := rec("A.json", "/dataList/"+strconv.Itoa(i)+"/content", kr, zh)
		r.Origin = origin
		recs = append(recs, r)
	}
	for i := range 25 { // 25 groups of two, in KR order
		kr := "가" + string(rune('가'+i))
		add(kr, "甲"+strconv.Itoa(i), store.OriginClaude)
		add(kr, "乙"+strconv.Itoa(i), store.OriginClaude)
	}
	big := "다다"
	add(big, "x1", store.OriginClaude)
	add(big, "x2", store.OriginHuman)
	add(big, "x3", store.OriginParatranz)
	add("라라", "라라", store.OriginCopy) // a copy and a stand-in
	add("라라", "stand-in", store.OriginFallback)
	add("라라", "real", store.OriginClaude)

	e := newEnv(t, map[string]string{"A.json": `{"dataList":[` + strings.Join(entries, ",") + `]}`}, nil)
	e.save(recs...)
	rep := e.run()
	if len(rep.Inconsistent) != 20 {
		t.Fatalf("%d groups, want 20", len(rep.Inconsistent))
	}
	if g := rep.Inconsistent[0]; g.KR != big || g.Units != 3 || len(g.ZH) != 3 || len(g.Keys) != 3 {
		t.Errorf("first group = %+v, want the biggest", g)
	}
	for i, g := range rep.Inconsistent[1:] {
		if want := "가" + string(rune('가'+i)); g.KR != want || g.Units != 2 {
			t.Errorf("group %d = %+v, want the text %q with 2 units", i+1, g, want)
		}
	}
	for _, g := range rep.Inconsistent {
		if g.KR == "라라" {
			t.Errorf("a copy and a stand-in made a group: %+v", g)
		}
	}

	// The build record keeps every group, in the same order, for lct suggest:
	// a group that a person declined or a rules PR holds must not keep one
	// of the twenty places of the report for ever.
	info, err := ReadInfo(e.opt.TWRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.InconsistentAll) != 26 || !reflect.DeepEqual(info.InconsistentAll[:20], rep.Inconsistent) {
		t.Fatalf("InconsistentAll has %d groups, want all 26 with the 20 of the report first", len(info.InconsistentAll))
	}
	for i, g := range info.InconsistentAll[20:] {
		if want := "가" + string(rune('가'+19+i)); g.KR != want || g.Units != 2 {
			t.Errorf("group %d = %+v, want the text %q with 2 units", 21+i, g, want)
		}
	}
	data, err := os.ReadFile(filepath.Join(e.opt.TWRoot, ".meta", "build.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"inconsistent_all": [`) {
		t.Errorf("build.json has no inconsistent_all list")
	}
}

// The build record keeps at most 500 groups, the biggest first, and a build
// without any writes an empty list: lct suggest takes a missing list for a
// build.json of an older lct.
func TestInfoInconsistentAllIsBounded(t *testing.T) {
	var entries []string
	var recs []store.Record
	for i := range 503 {
		for j, zh := range []string{"甲", "乙"} {
			n := len(entries)
			kr := "가" + strconv.Itoa(i)
			entries = append(entries, `{"id":`+strconv.Itoa(n)+`,"content":"`+kr+`"}`)
			recs = append(recs, rec("A.json", "/dataList/"+strconv.Itoa(n)+"/content", kr, zh+strconv.Itoa(j)))
		}
	}
	e := newEnv(t, map[string]string{"A.json": `{"dataList":[` + strings.Join(entries, ",") + `]}`}, nil)
	e.save(recs...)
	rep := e.run()
	info, err := ReadInfo(e.opt.TWRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Inconsistent) != 20 || len(info.InconsistentAll) != 500 || !reflect.DeepEqual(info.InconsistentAll[:20], rep.Inconsistent) {
		t.Errorf("%d groups in the report and %d in the record, want 20 and 500, the same first", len(rep.Inconsistent), len(info.InconsistentAll))
	}

	e2 := newEnv(t, map[string]string{"A.json": `{"dataList":[{"id":1,"content":"가"}]}`}, nil)
	e2.save(rec("A.json", "/dataList/0/content", "가", "甲"))
	e2.run()
	info, err = ReadInfo(e2.opt.TWRoot)
	if err != nil {
		t.Fatal(err)
	}
	if info.InconsistentAll == nil || len(info.InconsistentAll) != 0 {
		t.Errorf("InconsistentAll = %#v, want an empty list", info.InconsistentAll)
	}
}

// The counts the notes need: how many units show a stand-in and how many
// units the translator did for this snapshot.
func TestBuildCountsStandInsAndNewTranslations(t *testing.T) {
	kr := `{"dataList":[{"id":1,"content":"가"},{"id":2,"content":"나"},{"id":3,"content":"다"},{"id":4,"content":"라"},{"id":5,"content":"마"},{"id":6,"content":"[Bleed] 바"}]}`
	en := `{"dataList":[{"id":1,"content":"A"},{"id":2,"content":"B"},{"id":3,"content":"C"},{"id":4,"content":"D"},{"id":5,"content":"E"},{"id":6,"content":"[Bleed] F"}]}`
	e := newEnv(t, map[string]string{"A.json": kr}, map[string]string{"A.json": en})
	fresh := rec("A.json", "/dataList/0/content", "가", "甲") // this snapshot's translation
	old := rec("A.json", "/dataList/1/content", "나", "乙")   // an earlier one
	old.Since = "raw-20260101"
	fallback := rec("A.json", "/dataList/2/content", "다", "C") // a stand-in
	fallback.Origin = store.OriginFallback
	fallback.Flags = []string{store.FlagNeedsReview}
	human := rec("A.json", "/dataList/3/content", "라", "丁") // not the translator's
	human.Origin = store.OriginHuman
	broken := rec("A.json", "/dataList/5/content", "[Bleed] 바", "流血") // breaks the structure: EN stands in
	e.save(fresh, old, fallback, human, broken)
	rep := e.run()

	// Pending: unit 5 (no record). Stand-ins: that, the fallback record, the broken one.
	if rep.Pending != 1 || rep.StructuralFallback != 1 {
		t.Fatalf("Pending=%d StructuralFallback=%d, want 1 1", rep.Pending, rep.StructuralFallback)
	}
	if rep.StandIn != 3 {
		t.Errorf("StandIn = %d, want 3: one pending, one fallback record, one structural fallback", rep.StandIn)
	}
	if rep.NewTranslated != 2 { // fresh and the broken one are claude records of this snapshot
		t.Errorf("NewTranslated = %d, want 2", rep.NewTranslated)
	}
}

// out/summary.md is the report; one that is too long is cut at a line.
func TestSummaryMarkdownIsCut(t *testing.T) {
	small := []byte("# Build report\n\nshort\n")
	if got := SummaryMarkdown(small, ".meta/reports/raw-20261105.md"); string(got) != string(small) {
		t.Errorf("a short report changed: %q", got)
	}

	line := "- 一行很長的報表內容 😀 " + strings.Repeat("字", 40) + "\n"
	big := []byte(strings.Repeat(line, 2*SummaryLimit/len(line)))
	got := SummaryMarkdown(big, ".meta/reports/raw-20261105.md")
	if len(got) > SummaryLimit {
		t.Errorf("the summary is %d bytes, over %d", len(got), SummaryLimit)
	}
	if !utf8.Valid(got) {
		t.Error("the summary is cut inside a character")
	}
	if !strings.Contains(string(got), ".meta/reports/raw-20261105.md") || !strings.Contains(string(got), "截斷") {
		t.Errorf("the cut summary does not say it is cut and where the whole report is:\n...%s", got[len(got)-300:])
	}
	// Whole lines: what comes before the notice is a number of full lines.
	body := string(got[:strings.LastIndex(string(got), "\n\n")])
	if !strings.HasSuffix(body, strings.TrimSuffix(line, "\n")) {
		t.Errorf("the summary is cut inside a line: ...%q", body[len(body)-40:])
	}
	// One line that never ends is not cut inside a character either.
	oneLine := []byte(strings.Repeat("字", SummaryLimit/3+10))
	for range 3 { // however the limit falls on the three bytes of a character
		oneLine = append([]byte("a"), oneLine...)
		if got := SummaryMarkdown(oneLine, "x.md"); !utf8.Valid(got) || len(got) > SummaryLimit {
			t.Errorf("a cut line: valid UTF-8 %v, %d bytes", utf8.Valid(got), len(got))
		}
	}
	// Exactly at the limit is not cut.
	exact := bytes.Repeat([]byte("a"), SummaryLimit)
	if got := SummaryMarkdown(exact, "x.md"); len(got) != SummaryLimit {
		t.Errorf("a report of exactly %d bytes was cut to %d", SummaryLimit, len(got))
	}
}

// misalignedEntries is a dataList of n entries with an id and a title each,
// without the entries whose ids are in drop.
func misalignedEntries(n int, drop ...int) string {
	var entries []string
	for id := 1; id <= n; id++ {
		if slices.Contains(drop, id) {
			continue
		}
		entries = append(entries, `{"id":`+strconv.Itoa(id)+`,"title":"T`+strconv.Itoa(id)+`"}`)
	}
	return `{"dataList":[` + strings.Join(entries, ",") + `]}`
}

// withJP gives the environment a JP pack made of files (newEnv makes none).
func (e *testEnv) withJP(files map[string]string) {
	e.t.Helper()
	raw := filepath.Join(filepath.Dir(e.opt.TWRoot), "raw")
	writeLang(e.t, raw, "jp", "JP_", files)
	src, err := pack.OpenRaw(raw)
	if err != nil {
		e.t.Fatal(err)
	}
	e.src = src
}

// The files whose EN or JP lack units of KR are listed with how many, the
// worst first. A file the other language does not have at all is not
// misaligned but missing; a file that lines up is not listed.
func TestReportListsMisalignedFiles(t *testing.T) {
	kr := map[string]string{
		"A.json": misalignedEntries(12), "B.json": misalignedEntries(12), "C.json": misalignedEntries(12),
		"OnlyKR.json": misalignedEntries(12),
	}
	en := map[string]string{
		"A.json": misalignedEntries(12, 5),    // one line less
		"B.json": misalignedEntries(12, 3, 4), // two lines less
		"C.json": misalignedEntries(12),
	}
	e := newEnv(t, kr, en)
	e.withJP(map[string]string{
		"A.json": misalignedEntries(12),
		"B.json": misalignedEntries(12, 9), // and JP lacks one more
		"C.json": misalignedEntries(12),
	})
	rep := e.run()

	want := []FileMisaligned{{File: "B.json", EN: 2, JP: 1}, {File: "A.json", EN: 1, JP: 0}}
	if !reflect.DeepEqual(rep.Misaligned, want) || rep.MisalignedFiles != 2 {
		t.Errorf("Misaligned = %+v (%d in all), want %+v and 2", rep.Misaligned, rep.MisalignedFiles, want)
	}

	// The build record keeps it under the names of the plan.
	info, err := ReadInfo(e.opt.TWRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(info.Report.Misaligned, want) {
		t.Errorf("the build record has Misaligned = %+v, want %+v", info.Report.Misaligned, want)
	}
	raw, err := json.Marshal(want[0])
	if err != nil || string(raw) != `{"file":"B.json","en":2,"jp":1}` {
		t.Errorf("a misaligned file encodes as %s, %v", raw, err)
	}

	// The report names them in the information.
	misaligned := subsection(t, e.readReport(), "### 跨語言未對齊的檔")
	for _, line := range []string{"- `B.json`：EN 缺 2 個、JP 缺 1 個", "- `A.json`：EN 缺 1 個、JP 缺 0 個"} {
		if !strings.Contains(misaligned, line) {
			t.Errorf("the report lacks %q:\n%s", line, misaligned)
		}
	}
	if strings.Contains(misaligned, "C.json") || strings.Contains(misaligned, "OnlyKR.json") {
		t.Errorf("the report lists a file that lines up, or one that is missing in the other languages:\n%s", misaligned)
	}
}

// Nothing misaligned is an empty list that encodes as [], and the report
// says so.
func TestReportWithoutMisalignedFiles(t *testing.T) {
	e := newEnv(t, map[string]string{"A.json": misalignedEntries(3)}, map[string]string{"A.json": misalignedEntries(3)})
	rep := e.run()
	if rep.Misaligned == nil || len(rep.Misaligned) != 0 {
		t.Errorf("Misaligned = %#v, want an empty list that is not nil", rep.Misaligned)
	}
	if b, err := json.Marshal(rep); err != nil || !strings.Contains(string(b), `"misaligned":[]`) {
		t.Errorf("the report encodes without misaligned:[]: %s, %v", b, err)
	}
	if got := subsection(t, e.readReport(), "### 跨語言未對齊的檔"); !strings.Contains(got, "None.") {
		t.Errorf("the report does not say none is misaligned:\n%s", got)
	}
}

// Only the 20 worst files are kept; files with the same count go by name.
func TestReportMisalignedIsBounded(t *testing.T) {
	kr, en := map[string]string{}, map[string]string{}
	for i := range 25 {
		name := "F" + strconv.Itoa(10+i) + ".json"
		kr[name] = misalignedEntries(12)
		en[name] = misalignedEntries(12, 1) // one line less in each
	}
	kr["Worst.json"], en["Worst.json"] = misalignedEntries(12), misalignedEntries(12, 1, 2)
	e := newEnv(t, kr, en)
	rep := e.run()

	if len(rep.Misaligned) != 20 {
		t.Fatalf("%d files, want 20", len(rep.Misaligned))
	}
	if g := rep.Misaligned[0]; g.File != "Worst.json" || g.EN != 2 {
		t.Errorf("first = %+v, want Worst.json with 2", g)
	}
	for i, g := range rep.Misaligned[1:] {
		if want := "F" + strconv.Itoa(10+i) + ".json"; g.File != want || g.EN != 1 {
			t.Errorf("file %d = %+v, want %s with 1", i+1, g, want)
		}
	}
	// The report says how many there are in all: the list is not all of them.
	if rep.MisalignedFiles != 26 {
		t.Errorf("MisalignedFiles = %d, want 26", rep.MisalignedFiles)
	}
	if got := subsection(t, e.readReport(), "### 跨語言未對齊的檔"); !strings.Contains(got, "共 26 個") {
		t.Errorf("the report does not say that 26 files are misaligned:\n%s", got)
	}
}

// Every unit counts once in what it finally shows: a translation by its
// origin, a stand-in text (EN, or else KR) as a stand-in, and a unit that is
// not translated as a copy of KR. A record that broke the structure is shown
// as the stand-in only, although Origins counts the record as used.
func TestBuildShownCountsEachUnitOnce(t *testing.T) {
	kr := `{"dataList":[` +
		`{"id":0,"content":"가"},{"id":1,"content":"나"},{"id":2,"content":"다"},` +
		`{"id":3,"content":"[Bleed] 라"},{"id":4,"content":"마"},{"id":5,"content":"바"},` +
		`{"id":6,"model":"뫼르소"},{"id":7,"model":"이상"},{"id":8,"model":"로쟈"}]}`
	en := `{"dataList":[` +
		`{"id":0,"content":"A"},{"id":1,"content":"B"},{"id":2,"content":"C"},` +
		`{"id":3,"content":"[Bleed] D"},{"id":4,"content":"E"},{"id":5,"content":"F"},` +
		`{"id":6,"model":"Meursault"},{"id":7,"model":"Ishmael"},{"id":8,"model":"Rodion"}]}`
	lyrics := `{"dataList":[{"id":0,"content":"[Bleed] 사"}]}`
	e := newEnv(t,
		map[string]string{"A.json": kr, "BgmLyrics/L.json": lyrics},
		map[string]string{"A.json": en, "BgmLyrics/L.json": `{"dataList":[{"id":0,"content":"[Bleed] G"}]}`})

	pz := rec("A.json", "/dataList/0/content", "가", "甲")
	pz.Origin = store.OriginParatranz
	claude := rec("A.json", "/dataList/1/content", "나", "乙")
	human := rec("A.json", "/dataList/2/content", "다", "丙")
	human.Origin = store.OriginHuman
	broken := rec("A.json", "/dataList/3/content", "[Bleed] 라", "流血") // claude, but EN is shown
	fallback := rec("A.json", "/dataList/4/content", "마", "E")
	fallback.Origin = store.OriginFallback
	// Unit 5 has no record: pending. Unit 6 is not translated and has a copy;
	// 7 has a record the field rules override; 8 has none: all show KR.
	copied := rec("A.json", "/dataList/6/model", "뫼르소", "뫼르소")
	copied.Origin = store.OriginCopy
	overridden := rec("A.json", "/dataList/7/model", "이상", "伊")
	overridden.Origin = store.OriginHuman
	frozenBroken := rec("BgmLyrics/L.json", "/dataList/0/content", "[Bleed] 사", "流血") // a frozen file
	e.save(pz, claude, human, broken, fallback, copied, overridden, frozenBroken)
	rep := e.run()

	want := map[string]int{"paratranz": 1, "claude": 1, "human": 1, "copy": 3, "stand-in": 4}
	if !reflect.DeepEqual(rep.Shown, want) {
		t.Errorf("Shown = %v, want %v", rep.Shown, want)
	}
	total := 0
	for _, n := range rep.Shown {
		total += n
	}
	if total != 10 { // 6 contents and 3 models of A.json, and the lyric
		t.Errorf("Shown counts %d units in all, want each of the 10 once", total)
	}
	if rep.Shown[ShownStandIn] != rep.StandIn {
		t.Errorf("Shown[stand-in] = %d, StandIn = %d: they count the same units", rep.Shown[ShownStandIn], rep.StandIn)
	}
	// Origins counts the records used, the broken ones included: three claude
	// records, but only one of them is shown.
	if rep.Origins["claude"] != 3 {
		t.Errorf("Origins = %v, want 3 claude records used, the two broken ones too", rep.Origins)
	}

	info, err := ReadInfo(e.opt.TWRoot)
	if err != nil || !reflect.DeepEqual(info.Report.Shown, want) {
		t.Errorf("the build record has Shown = %v, %v, want %v", info.Report.Shown, err, want)
	}
}

// A translation that was not run is in the report and in build.json with its
// reason, not as the error of a run.
func TestReportShowsATranslationThatDidNotRun(t *testing.T) {
	e := newEnv(t, map[string]string{"A.json": twoUnitsKR}, nil)
	e.save(rec("A.json", "/dataList/0/content", "가", "甲"))
	writeTranslation(t, e, translate.Result{
		Stopped: translate.StopNotRun, Error: "缺少 ANTHROPIC_API_KEY 或 LCT_MODEL",
		PendingBefore: 3, PendingAfter: 3, RawTag: testTag, RulesCommit: "abc1234",
	})
	e.run()
	info, err := ReadInfo(e.opt.TWRoot)
	if err != nil || info.Stopped != translate.StopNotRun || info.StopError != "缺少 ANTHROPIC_API_KEY 或 LCT_MODEL" || info.Model != "" {
		t.Errorf("Info = %+v, %v, want not-run with its reason and no model", info, err)
	}
	rest := section(t, e.readReport(), "## 資訊")
	for _, want := range []string{"not-run（沒有執行）", "原因：`缺少 ANTHROPIC_API_KEY 或 LCT_MODEL`"} {
		if !strings.Contains(rest, want) {
			t.Errorf("the information lacks %q:\n%s", want, rest)
		}
	}
	if strings.Contains(rest, "錯誤") {
		t.Errorf("a translation that did not run is reported as an error:\n%s", rest)
	}
}

// A warning a person accepted (by the id of its KR and translation) is not
// counted and not sampled; the same unit with another translation still is.
func TestBuildAcceptedWarningNotCounted(t *testing.T) {
	kr := map[string]string{"A.json": `{"dataList":[{"id":1,"content":"가"},{"id":2,"content":"나"}]}`}
	recs := []store.Record{
		rec("A.json", "/dataList/0/content", "가", "甲か"),
		rec("A.json", "/dataList/1/content", "나", "乙か"),
	}
	e := newEnv(t, kr, nil)
	e.save(recs...)
	if rep := e.run(); rep.Warnings["kana"] != 2 {
		t.Fatalf("Warnings = %v, want 2 kana without an accepted list", rep.Warnings)
	}

	e = newEnv(t, kr, nil)
	e.save(recs...)
	e.opt.Accepted = map[string]bool{itemid.Warning("kana", "가", "甲か"): true}
	rep := e.run()
	if rep.Warnings["kana"] != 1 {
		t.Errorf("Warnings = %v, want 1 kana", rep.Warnings)
	}
	if got := rep.WarningSamples["kana"]; !slices.Equal(got, []string{recs[1].Key}) {
		t.Errorf("WarningSamples[kana] = %v, want only %q", got, recs[1].Key)
	}

	// Another translation of the same KR has another id: counted as usual.
	e = newEnv(t, kr, nil)
	e.save(recs...)
	e.opt.Accepted = map[string]bool{itemid.Warning("kana", "가", "別か"): true}
	if rep := e.run(); rep.Warnings["kana"] != 2 {
		t.Errorf("Warnings = %v, want 2 kana: the accepted id is of another translation", rep.Warnings)
	}
}

// A declined group of inconsistent renderings leaves the report and the build
// record, and the next group takes its place.
func TestBuildDeclinedGroupGivesUpItsPlace(t *testing.T) {
	e := newEnv(t, map[string]string{
		"A.json": `{"dataList":[{"id":1,"content":"가"},{"id":2,"content":"가"},{"id":3,"content":"가"},{"id":4,"content":"나"},{"id":5,"content":"나"}]}`,
	}, nil)
	e.opt.Rules = routingRules()
	e.opt.Declined = map[string]bool{itemid.Inconsistent("alpha", "content", "가"): true}
	e.save(
		rec("A.json", "/dataList/0/content", "가", "甲"),
		rec("A.json", "/dataList/1/content", "가", "乙"),
		rec("A.json", "/dataList/2/content", "가", "甲"),
		rec("A.json", "/dataList/3/content", "나", "丙"),
		rec("A.json", "/dataList/4/content", "나", "丁"),
	)
	rep := e.run()
	if len(rep.Inconsistent) != 1 || rep.Inconsistent[0].KR != "나" {
		t.Errorf("Inconsistent = %+v, want only the group of 나", rep.Inconsistent)
	}
	info, err := ReadInfo(e.opt.TWRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.InconsistentAll) != 1 || info.InconsistentAll[0].KR != "나" {
		t.Errorf("InconsistentAll = %+v, want only the group of 나", info.InconsistentAll)
	}
}

// A used record that lacks the glossary rendering warns term-missing, unless
// its warning was accepted.
func TestBuildCountsTermMissingWarnings(t *testing.T) {
	const glossary = "- kr: 탐식\n  zh: 暴食\n  status: approved\n"
	root := t.TempDir()
	for _, d := range []string{"guides/characters", "glossary"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(t, filepath.Join(root, "guides/routing.yaml"), []byte("families:\n  - id: other\n    nodes: []\npriority:\n  - other\n"))
	mustWrite(t, filepath.Join(root, "glossary/terms.yaml"), []byte(glossary))
	rl, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	kr := map[string]string{"A.json": `{"dataList":[{"id":1,"content":"탐식의 재림"}]}`}
	r := rec("A.json", "/dataList/0/content", "탐식의 재림", "貪食的再臨")

	e := newEnv(t, kr, nil)
	e.opt.Rules = rl
	e.save(r)
	if rep := e.run(); rep.Warnings["term-missing"] != 1 {
		t.Fatalf("Warnings = %v, want one term-missing", rep.Warnings)
	}

	e = newEnv(t, kr, nil)
	e.opt.Rules = rl
	e.save(r)
	e.opt.Accepted = map[string]bool{itemid.UnitWarning(validate.CodeTermMissing, "탐식", "暴食", "탐식의 재림", "貪食的再臨"): true}
	if rep := e.run(); rep.Warnings["term-missing"] != 0 {
		t.Errorf("Warnings = %v, want none: the warning was accepted", rep.Warnings)
	}
}

// An entry of check: off warns of no missing rendering in the build; its
// forbidden renderings warn as ever.
func TestBuildCheckOffWarnsOnlyForbidden(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"guides/characters", "glossary"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(t, filepath.Join(root, "guides/routing.yaml"), []byte("families:\n  - id: other\n    nodes: []\npriority:\n  - other\n"))
	mustWrite(t, filepath.Join(root, "glossary/terms.yaml"), []byte("- kr: 탐식\n  zh: 暴食\n  status: approved\n  check: off\n  forbidden: [饕餮]\n"))
	rl, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	kr := map[string]string{"A.json": `{"dataList":[{"id":1,"content":"탐식의 재림"},{"id":2,"content":"탐식"}]}`}
	e := newEnv(t, kr, nil)
	e.opt.Rules = rl
	e.save(rec("A.json", "/dataList/0/content", "탐식의 재림", "貪食的再臨"), rec("A.json", "/dataList/1/content", "탐식", "饕餮"))
	rep := e.run()
	if rep.Warnings["term-missing"] != 0 || rep.Warnings["forbidden-term"] != 1 {
		t.Errorf("Warnings = %v, want no term-missing and one forbidden-term", rep.Warnings)
	}
}

// The term-missing warnings weigh the provisional terms in force with the
// glossary, as Submit does (batch.TermsIn): a glossary entry each of whose
// hits lies inside a provisional term's does not warn, and the provisional
// terms themselves never warn on an existing translation. A declined one, or
// none in the store, covers nothing.
func TestBuildWeighsProvisionalTermsAsCoverers(t *testing.T) {
	const glossary = "- kr: 와일드헌트\n  zh: 狂獵\n  status: approved\n"
	root := t.TempDir()
	for _, d := range []string{"guides/characters", "glossary"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(t, filepath.Join(root, "guides/routing.yaml"), []byte("families:\n  - id: other\n    nodes: []\npriority:\n  - other\n"))
	mustWrite(t, filepath.Join(root, "glossary/terms.yaml"), []byte(glossary))
	rl, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	kr := map[string]string{"A.json": `{"dataList":[{"id":1,"content":"와일드헌트 히스클리프가 왔다"},{"id":2,"content":"와일드헌트 히스클리프 둘"},{"id":3,"content":"와일드헌트가 왔다"}]}`}
	recs := []store.Record{
		rec("A.json", "/dataList/0/content", "와일드헌트 히스클리프가 왔다", "荒獵希斯克利夫來了"),
		rec("A.json", "/dataList/1/content", "와일드헌트 히스클리프 둘", "希斯克利夫二"),
		rec("A.json", "/dataList/2/content", "와일드헌트가 왔다", "來了"),
	}
	run := func(provisional bool, declined map[string]bool) *Report {
		e := newEnv(t, kr, nil)
		e.opt.Rules = rl
		e.opt.Declined = declined
		e.save(recs...)
		if provisional {
			if _, err := store.Open(e.opt.TWRoot).AppendTerms([]store.ProvisionalTerm{{KR: "와일드헌트 히스클리프", ZH: "荒獵希斯克利夫", Key: recs[0].Key, Batch: "b1"}}); err != nil {
				t.Fatal(err)
			}
		}
		return e.run()
	}
	rep := run(true, nil)
	if rep.Warnings["term-missing"] != 1 || !slices.Equal(rep.WarningSamples["term-missing"], []string{recs[2].Key}) {
		t.Errorf("provisional in force: Warnings = %v, samples %v; want only %s", rep.Warnings, rep.WarningSamples["term-missing"], recs[2].Key)
	}
	for name, rep := range map[string]*Report{
		"declined":       run(true, map[string]bool{"term-new/와일드헌트 히스클리프": true}),
		"no provisional": run(false, nil),
	} {
		if rep.Warnings["term-missing"] != 3 {
			t.Errorf("%s: Warnings = %v, want three term-missing", name, rep.Warnings)
		}
	}
}

// A unit that lacks two glossary entries warns term-missing for each, and
// the one accepted (for its entry, as the review page's term group has it)
// leaves the other counted.
func TestBuildAcceptsTermMissingByTerm(t *testing.T) {
	const glossary = "- kr: 탐식\n  zh: 暴食\n  status: approved\n\n- kr: 재림\n  zh: 降臨\n  status: approved\n"
	root := t.TempDir()
	for _, d := range []string{"guides/characters", "glossary"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(t, filepath.Join(root, "guides/routing.yaml"), []byte("families:\n  - id: other\n    nodes: []\npriority:\n  - other\n"))
	mustWrite(t, filepath.Join(root, "glossary/terms.yaml"), []byte(glossary))
	rl, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	kr := map[string]string{"A.json": `{"dataList":[{"id":1,"content":"탐식의 재림"}]}`}
	r := rec("A.json", "/dataList/0/content", "탐식의 재림", "貪食的再臨")
	run := func(terms ...string) *Report {
		e := newEnv(t, kr, nil)
		e.opt.Rules = rl
		e.save(r)
		e.opt.Accepted = map[string]bool{}
		for _, term := range terms {
			zh := map[string]string{"탐식": "暴食", "재림": "降臨"}[term]
			e.opt.Accepted[itemid.UnitWarning(validate.CodeTermMissing, term, zh, "탐식의 재림", "貪食的再臨")] = true
		}
		return e.run()
	}
	if rep := run(); rep.Warnings["term-missing"] != 2 {
		t.Fatalf("Warnings = %v, want two term-missing", rep.Warnings)
	}
	rep := run("탐식")
	if rep.Warnings["term-missing"] != 1 || !slices.Equal(rep.WarningSamples["term-missing"], []string{r.Key}) {
		t.Errorf("탐식 accepted: Warnings = %v, samples %v; want the one of 재림", rep.Warnings, rep.WarningSamples["term-missing"])
	}
	if rep := run("탐식", "재림"); rep.Warnings["term-missing"] != 0 {
		t.Errorf("both accepted: Warnings = %v, want none", rep.Warnings)
	}
}

// Two entries of one KR text with two renderings: one for A.json, one for
// the other files (as 신부 is 牧師 in chapter 10 and 神父 elsewhere), and two
// that both apply to C.json. A warning accepted for one entry leaves the
// warning of the other counted.
func TestBuildAcceptsTermMissingByRendering(t *testing.T) {
	const glossary = "- kr: 신부\n  zh: 牧師\n  files: [\"A.json\", \"C.json\"]\n  status: approved\n\n- kr: 신부\n  zh: 神父\n  status: approved\n\n" +
		"- kr: 신부\n  zh: 神職者\n  files: [\"C.json\"]\n  status: approved\n"
	root := t.TempDir()
	for _, d := range []string{"guides/characters", "glossary"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(t, filepath.Join(root, "guides/routing.yaml"), []byte("families:\n  - id: other\n    nodes: []\npriority:\n  - other\n"))
	mustWrite(t, filepath.Join(root, "glossary/terms.yaml"), []byte(glossary))
	rl, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	const kr, zh = "신부님이 왔다", "神甫來了"
	doc := `{"dataList":[{"id":1,"content":"` + kr + `"}]}`
	files := map[string]string{"A.json": doc, "B.json": doc, "C.json": doc}
	recs := []store.Record{
		rec("A.json", "/dataList/0/content", kr, zh),
		rec("B.json", "/dataList/0/content", kr, zh),
		rec("C.json", "/dataList/0/content", kr, zh),
	}
	run := func(renderings ...string) *Report {
		e := newEnv(t, files, nil)
		e.opt.Rules = rl
		e.save(recs...)
		e.opt.Accepted = map[string]bool{}
		for _, r := range renderings {
			e.opt.Accepted[itemid.UnitWarning(validate.CodeTermMissing, "신부", r, kr, zh)] = true
		}
		return e.run()
	}
	// A.json lacks 牧師, B.json 神父, C.json 牧師 and 神職者.
	if rep := run(); rep.Warnings["term-missing"] != 4 {
		t.Fatalf("Warnings = %v, want four term-missing", rep.Warnings)
	}
	rep := run("牧師")
	if rep.Warnings["term-missing"] != 2 || !slices.Equal(rep.WarningSamples["term-missing"], []string{recs[1].Key, recs[2].Key}) {
		t.Errorf("牧師 accepted: Warnings = %v, samples %v; want 神父 of B.json and 神職者 of C.json", rep.Warnings, rep.WarningSamples["term-missing"])
	}
	if rep := run("牧師", "神父", "神職者"); rep.Warnings["term-missing"] != 0 {
		t.Errorf("all accepted: Warnings = %v, want none", rep.Warnings)
	}
}

// existingNames is an index of one existing name, 노을빛 검: 霞光劍刃 in 12
// units and 霞光之劍 in one where JP says 茜さす剣, 霞光劍刃 in two and 夕霞劍
// in one where EN says Dusk Blade.
func existingNames(declined map[string]bool) *existing.Index {
	return existing.NewIndex(&existing.Table{RawTag: "raw-20261001", Fields: []string{"name"}, Names: []existing.Name{{
		KR: "노을빛 검",
		Variants: []existing.Variant{
			{JP: "茜さす剣", Renderings: []existing.Rendering{{ZH: "霞光劍刃", Units: 12}, {ZH: "霞光之劍", Units: 1}}},
			{EN: "Dusk Blade", Renderings: []existing.Rendering{{ZH: "霞光劍刃", Units: 2}, {ZH: "夕霞劍", Units: 1}}},
		},
	}}}, routingRules(), declined)
}

// wantNameExisting are the renderings of existingNames, summed over the
// variants.
var wantNameExisting = []existing.Rendering{{ZH: "霞光劍刃", Units: 14}, {ZH: "夕霞劍", Units: 1}, {ZH: "霞光之劍", Units: 1}}

// written is r as Submit writes it at the raw tag of the build.
func written(r store.Record) store.Record {
	r.Written = testTag
	return r
}

// nameEnv is an environment of Skills.json (kr), whose JP says 茜さす剣 where
// kr says 노을빛 검, with the routing rules, existingNames and recs. declined
// is the declined list.
func nameEnv(t *testing.T, kr string, declined map[string]bool, recs ...store.Record) *testEnv {
	t.Helper()
	e := newEnv(t, map[string]string{"Skills.json": kr}, nil)
	e.withJP(map[string]string{"Skills.json": strings.ReplaceAll(kr, "노을빛 검", "茜さす剣")})
	e.opt.Rules = routingRules()
	e.opt.Declined = declined
	e.opt.Names = existingNames(declined)
	e.save(recs...)
	return e
}

// A record of origin claude whose name field renders an existing name
// otherwise than the variant its JP points at warns name-differs, whenever it
// was written. One written for this raw tag is listed too, with the notes of
// the translator and every rendering the name has.
func TestBuildListsNameDiffers(t *testing.T) {
	const doc = `{"dataList":[{"id":1,"name":"노을빛 검"},{"id":2,"name":"노을빛 검"},{"id":3,"name":"노을빛 검"},{"id":4,"name":"노을빛 검"}]}`
	paratranz := rec("Skills.json", "/dataList/3/name", "노을빛 검", "晚霞之劍")
	paratranz.Origin = store.OriginParatranz
	recs := []store.Record{
		written(rec("Skills.json", "/dataList/0/name", "노을빛 검", "晚霞之劍")),
		rec("Skills.json", "/dataList/1/name", "노을빛 검", "晚霞之劍"), // an older translation: it warns but is not listed
		written(rec("Skills.json", "/dataList/2/name", "노을빛 검", "霞光劍刃")),
		paratranz, // not the translator's: not checked
	}
	e := nameEnv(t, doc, nil, recs...)
	if _, err := store.Open(e.opt.TWRoot).AppendNotes([]store.Note{
		{Key: recs[0].Key, Batch: "b1", KRSha: units.KRSha("노을빛 검"), Model: "m", Note: "依技能演出"},
		{Key: recs[0].Key, Batch: "b2", KRSha: units.KRSha("노을빛 검"), Model: "m", Note: "第二則"},
	}); err != nil {
		t.Fatal(err)
	}
	rep := e.run()
	if rep.Warnings[existing.CodeNameDiffers] != 2 || !slices.Equal(rep.WarningSamples[existing.CodeNameDiffers], []string{recs[0].Key, recs[1].Key}) {
		t.Errorf("Warnings = %v, samples %v; want two name-differs, of %s and %s", rep.Warnings, rep.WarningSamples, recs[0].Key, recs[1].Key)
	}
	want := []NameDiff{{
		KR: "노을빛 검", Existing: wantNameExisting,
		Units: []NameDiffUnit{{Key: recs[0].Key, ZH: "晚霞之劍", Note: "依技能演出\n第二則", Kind: NameDiffField}},
	}}
	if !reflect.DeepEqual(rep.NameDiffs, want) {
		t.Errorf("NameDiffs = %+v, want %+v", rep.NameDiffs, want)
	}
}

// NameDiff.Existing sums the renderings of every variant and keeps the five
// with the most units.
func TestBuildNameDiffExistingIsBounded(t *testing.T) {
	const doc = `{"dataList":[{"id":1,"name":"노을빛 검"}]}`
	e := nameEnv(t, doc, nil, written(rec("Skills.json", "/dataList/0/name", "노을빛 검", "晚霞之劍")))
	e.opt.Names = existing.NewIndex(&existing.Table{RawTag: "raw-20261001", Fields: []string{"name"}, Names: []existing.Name{{
		KR: "노을빛 검",
		Variants: []existing.Variant{
			{JP: "茜さす剣", Renderings: []existing.Rendering{{ZH: "甲", Units: 4}, {ZH: "乙", Units: 2}, {ZH: "丙", Units: 1}}},
			{EN: "Dusk Blade", Renderings: []existing.Rendering{{ZH: "丁", Units: 3}, {ZH: "乙", Units: 2}, {ZH: "戊", Units: 1}, {ZH: "己", Units: 1}}},
		},
	}}}, e.opt.Rules, nil)
	rep := e.run()
	// 乙 sorts before 甲, 丙 before 己 and 己 before 戊.
	want := []existing.Rendering{{ZH: "乙", Units: 4}, {ZH: "甲", Units: 4}, {ZH: "丁", Units: 3}, {ZH: "丙", Units: 1}, {ZH: "己", Units: 1}}
	if len(rep.NameDiffs) != 1 || !reflect.DeepEqual(rep.NameDiffs[0].Existing, want) {
		t.Errorf("NameDiffs = %+v, want one with the renderings %+v", rep.NameDiffs, want)
	}
}

// An accepted name-differs warning is neither counted nor listed.
func TestBuildNameDifferAccepted(t *testing.T) {
	const doc = `{"dataList":[{"id":1,"name":"노을빛 검"}]}`
	r := written(rec("Skills.json", "/dataList/0/name", "노을빛 검", "晚霞之劍"))
	if rep := nameEnv(t, doc, nil, r).run(); rep.Warnings[existing.CodeNameDiffers] != 1 || len(rep.NameDiffs) != 1 {
		t.Fatalf("Warnings = %v, NameDiffs = %+v; want the warning listed without an accepted list", rep.Warnings, rep.NameDiffs)
	}
	e := nameEnv(t, doc, nil, r)
	e.opt.Accepted = map[string]bool{itemid.UnitWarning(existing.CodeNameDiffers, "노을빛 검", "霞光劍刃", "노을빛 검", "晚霞之劍"): true}
	rep := e.run()
	if rep.Warnings[existing.CodeNameDiffers] != 0 || len(rep.NameDiffs) != 0 {
		t.Errorf("Warnings = %v, NameDiffs = %+v; want neither: the warning was accepted", rep.Warnings, rep.NameDiffs)
	}
}

// mentionsKR has three units of desc that mention 노을빛 검 and one of name
// that is the name.
const mentionsKR = `{"dataList":[{"id":1,"desc":"노을빛 검 발동"},{"id":2,"desc":"노을빛 검 발동!"},{"id":3,"desc":"노을빛 검 발동."},{"id":4,"name":"노을빛 검"}]}`

// mentionsRecords are records of mentionsKR: a translation of this raw tag
// that renders the name in no existing way, one that does, an older one that
// does not, and a name of this raw tag that differs.
func mentionsRecords() []store.Record {
	return []store.Record{
		written(rec("Skills.json", "/dataList/0/desc", "노을빛 검 발동", "發動晚霞之劍")),
		written(rec("Skills.json", "/dataList/1/desc", "노을빛 검 발동!", "發動霞光之劍！")),
		rec("Skills.json", "/dataList/2/desc", "노을빛 검 발동.", "發動晚霞之劍。"),
		written(rec("Skills.json", "/dataList/3/name", "노을빛 검", "晚霞之劍")),
	}
}

// A translation of this raw tag whose KR mentions an existing name and whose
// text has none of its renderings is listed, without a warning; one that has
// a rendering, an older one, and a name field that is the name itself are
// not listed as mentions.
func TestBuildListsMentions(t *testing.T) {
	recs := mentionsRecords()
	rep := nameEnv(t, mentionsKR, nil, recs...).run()
	want := []NameDiff{{
		KR: "노을빛 검", Existing: wantNameExisting,
		Units: []NameDiffUnit{
			{Key: recs[0].Key, ZH: "發動晚霞之劍", Kind: NameDiffMention},
			{Key: recs[3].Key, ZH: "晚霞之劍", Kind: NameDiffField},
		},
	}}
	if !reflect.DeepEqual(rep.NameDiffs, want) {
		t.Errorf("NameDiffs = %+v, want %+v", rep.NameDiffs, want)
	}
	if rep.Warnings[existing.CodeNameDiffers] != 1 {
		t.Errorf("Warnings = %v, want one name-differs: a mention does not warn", rep.Warnings)
	}
}

// A declined name neither warns nor is listed.
func TestBuildSkipsDeclinedNames(t *testing.T) {
	rep := nameEnv(t, mentionsKR, map[string]bool{existing.SuggestionID("노을빛 검"): true}, mentionsRecords()...).run()
	if rep.Warnings[existing.CodeNameDiffers] != 0 || len(rep.NameDiffs) != 0 {
		t.Errorf("Warnings = %v, NameDiffs = %+v; want neither: the name was declined", rep.Warnings, rep.NameDiffs)
	}
}

// A provisional term of this raw tag that differs from the old translations
// is listed with the old units counted again, three of them as examples, and
// the units of this update that use its rendering. Terms of another raw tag,
// declined, in force, or decided by an earlier record are not listed.
func TestBuildListsTermDiffs(t *testing.T) {
	var entries, krs []string
	var recs []store.Record
	add := func(kr, zh string, isNew bool) {
		i := strconv.Itoa(len(entries))
		entries = append(entries, `{"id":`+i+`,"desc":"`+kr+`"}`)
		krs = append(krs, kr)
		r := rec("Bufs.json", "/dataList/"+i+"/desc", kr, zh)
		if isNew {
			r = written(r)
		}
		recs = append(recs, r)
	}
	for i := range 6 {
		n := strconv.Itoa(i + 1)
		add("도발치 "+n+" 증가", "挑釁值增加"+n, false)
	}
	add("도발치 7 증가", "嘲諷值增加7", true)
	add("도발치를 얻는다", "獲得嘲諷值", true)
	add("도발치 감소", "挑釁值減少", true)   // this update, but not the provisional rendering
	add("큰도발치 감소", "大嘲諷值減少", true) // 도발치 is no word of its own here

	run := func(names *existing.Index) *Report {
		e := newEnv(t, map[string]string{"Bufs.json": `{"dataList":[` + strings.Join(entries, ",") + `]}`}, nil)
		e.opt.Rules = routingRules()
		e.opt.Declined = map[string]bool{"term-new/회피치": true}
		e.opt.Names = names
		e.save(recs...)
		old := &store.OldUse{Existing: "挑釁值", Units: 95}
		if _, err := store.Open(e.opt.TWRoot).AppendTerms([]store.ProvisionalTerm{
			{KR: "방어치", ZH: "格擋值", Key: recs[0].Key, Batch: "b1", RawTag: testTag, Conflict: true, Old: old}, // a conflict decides nothing
			{KR: "도발치", ZH: "嘲諷值", Key: recs[6].Key, Batch: "b1", RawTag: testTag, Old: old},
			{KR: "도발치", ZH: "挑撥值", Key: recs[7].Key, Batch: "b2", RawTag: testTag, Conflict: true},
			{KR: "방어치", ZH: "防禦值", Key: recs[0].Key, Batch: "b1", RawTag: testTag},                  // in force
			{KR: "방어치", ZH: "守備值", Key: recs[0].Key, Batch: "b1", RawTag: testTag, Old: old},        // the first record decided
			{KR: "회피치", ZH: "迴避值", Key: recs[0].Key, Batch: "b1", RawTag: testTag, Old: old},        // declined
			{KR: "속도치", ZH: "速度值", Key: recs[0].Key, Batch: "b0", RawTag: "raw-20261001", Old: old}, // of an earlier update
		}); err != nil {
			t.Fatal(err)
		}
		return e.run()
	}

	var examples []existing.OldExample // the first three old units by key
	for i, r := range recs[:3] {
		examples = append(examples, existing.OldExample{Key: r.Key, KR: krs[i], ZH: r.ZH})
	}
	want := []TermDiff{{
		KR: "도발치", ZH: "嘲諷值", Existing: "挑釁值", OldUnits: 6, OldWith: 0,
		Examples: examples, Keys: []string{recs[6].Key, recs[7].Key},
	}}
	// Without the name in the table, Existing is the one the term was stored with.
	if rep := run(existing.NewIndex(nil, routingRules(), nil)); !reflect.DeepEqual(rep.TermDiffs, want) {
		t.Errorf("TermDiffs = %+v, want %+v", rep.TermDiffs, want)
	}

	// With it, the table says.
	names := existing.NewIndex(&existing.Table{RawTag: "raw-20261001", Fields: []string{"name"}, Names: []existing.Name{
		{KR: "도발치", Variants: []existing.Variant{{Renderings: []existing.Rendering{{ZH: "挑戰值", Units: 4}}}}},
	}}, routingRules(), nil)
	want[0].Existing = "挑戰值"
	if rep := run(names); !reflect.DeepEqual(rep.TermDiffs, want) {
		t.Errorf("with the name in the table: TermDiffs = %+v, want %+v", rep.TermDiffs, want)
	}
}

// A group of inconsistent renderings counts its units translated for this
// raw tag; a group of older translations only has none.
func TestBuildMarksNewInconsistency(t *testing.T) {
	e := newEnv(t, map[string]string{
		"A.json": `{"dataList":[{"id":1,"content":"가"},{"id":2,"content":"가"},{"id":3,"content":"가"},{"id":4,"content":"가"},{"id":5,"content":"나"},{"id":6,"content":"나"}]}`,
	}, nil)
	e.opt.Rules = routingRules()
	recs := []store.Record{
		rec("A.json", "/dataList/0/content", "가", "甲"),
		rec("A.json", "/dataList/1/content", "가", "甲"),
		rec("A.json", "/dataList/2/content", "가", "甲"),
		written(rec("A.json", "/dataList/3/content", "가", "乙")),
		rec("A.json", "/dataList/4/content", "나", "丙"),
		rec("A.json", "/dataList/5/content", "나", "丁"),
	}
	e.save(recs...)
	rep := e.run()
	want := []Inconsistency{
		{
			Family: "alpha", Field: "content", KR: "가", ZH: []string{"甲", "乙"}, Keys: []string{recs[0].Key, recs[3].Key},
			Counts: []int{3, 1}, Units: 4, New: 1, NewKeys: []string{recs[3].Key},
		},
		{
			Family: "alpha", Field: "content", KR: "나", ZH: []string{"丙", "丁"}, Keys: []string{recs[4].Key, recs[5].Key},
			Counts: []int{1, 1}, Units: 2,
		},
	}
	if !reflect.DeepEqual(rep.Inconsistent, want) {
		t.Errorf("Inconsistent = %+v, want %+v", rep.Inconsistent, want)
	}
	info, err := ReadInfo(e.opt.TWRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(info.InconsistentAll, want) {
		t.Errorf("InconsistentAll = %+v, want %+v", info.InconsistentAll, want)
	}
	// A group without new units says nothing of them.
	all := jsonKeys(t, filepath.Join(e.opt.TWRoot, ".meta", "build.json"))["inconsistent_all"].([]any)
	if g := all[1].(map[string]any); g["new"] != nil || g["new_keys"] != nil {
		t.Errorf("the group of older translations = %v, want no new nor new_keys", g)
	}
}

// A build.json of an older lct, without name_diffs, term_diffs and new, is
// read.
func TestInfoWithoutNameDiffsDecodes(t *testing.T) {
	dir := t.TempDir()
	mustMkdirWrite(t, filepath.Join(dir, ".meta", "build.json"), `{
  "raw_tag": "raw-20261001", "raw_token": "t", "rules_commit": "abc", "built_at": "2026-10-01T00:00:00Z", "notes_seen": 0, "zips": {},
  "report": {"origins": {}, "pending": 0, "warnings": {}, "inconsistent": [{"family": "a", "field": "content", "kr": "가", "zh": ["甲", "乙"], "keys": ["k1", "k2"], "units": 2}]},
  "inconsistent_all": [{"family": "a", "field": "content", "kr": "가", "zh": ["甲", "乙"], "keys": ["k1", "k2"], "units": 2}]
}`)
	info, err := ReadInfo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Report.NameDiffs) != 0 || len(info.Report.TermDiffs) != 0 || info.Report.Inconsistent[0].New != 0 || info.InconsistentAll[0].NewKeys != nil {
		t.Errorf("Info = %+v, want no differences and no new units", info)
	}
}

// Without the existing names nothing is compared with them: there is no
// name-differs warning, no difference is listed, and the rest of the report
// is what it is with them.
func TestBuildWithoutNames(t *testing.T) {
	recs := mentionsRecords()
	with := nameEnv(t, mentionsKR, nil, recs...).run()
	e := nameEnv(t, mentionsKR, nil, recs...)
	e.opt.Names = nil
	rep := e.run()
	if rep.NameDiffs == nil || len(rep.NameDiffs) != 0 || rep.TermDiffs == nil || len(rep.TermDiffs) != 0 {
		t.Errorf("NameDiffs = %#v, TermDiffs = %#v; want both empty and not nil", rep.NameDiffs, rep.TermDiffs)
	}
	if len(with.NameDiffs) == 0 || with.Warnings[existing.CodeNameDiffers] == 0 {
		t.Fatalf("with the names: NameDiffs = %+v, Warnings = %v; want both", with.NameDiffs, with.Warnings)
	}
	with.NameDiffs, with.TermDiffs = []NameDiff{}, []TermDiff{}
	delete(with.Warnings, existing.CodeNameDiffers)
	delete(with.WarningSamples, existing.CodeNameDiffers)
	if !reflect.DeepEqual(rep, with) {
		t.Errorf("report without the names = %+v, want %+v", rep, with)
	}
	b, err := os.ReadFile(filepath.Join(e.opt.TWRoot, ".meta", "build.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{`"name_diffs": []`, `"term_diffs": []`} {
		if !strings.Contains(string(b), w) {
			t.Errorf("build.json lacks %s", w)
		}
	}
}
