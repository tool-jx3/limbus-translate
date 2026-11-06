package site

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/align"
	"github.com/tool-jx3/limbus-translate/internal/batch"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// updatePreview rewrites site/fixtures/preview/ from the test pack instead of
// comparing with it: go test ./internal/site -run TestPreviewFixturesUpToDate -update-preview
var updatePreview = flag.Bool("update-preview", false, "rewrite site/fixtures/preview/ instead of comparing with it")

const (
	pvRulesCommit = "0123456789abcdef"
	pvPrompt      = "# 翻譯提示詞\n\n照指引翻譯，<標籤>與 & 原樣保留。\n"
)

const pvRouting = `families:
  - id: story-main
    nodes: ["StoryData/S"]
    guide: guides/families.md
  - id: loose
    nodes: ["Frozen"]
    guide: guides/families.md
    glossary: warn
  - id: other
    nodes: []
    guide: guides/families.md
priority:
  - story-main
  - loose
  - other
`

const pvGlossary = `- kr: 위력
  zh: 威力
  status: approved
  forbidden: [力量]
- kr: 공격
  zh: 攻擊
  status: approved
- kr: 후보
  zh: 候選
  status: candidate
`

const (
	pvFaust = "---\nid: faust\nname: 浮士德\nname_kr: 파우스트\nmodels: [파우스트]\nfile_names: [Faust]\n---\n\n# 浮士德\n"
	pvSinc  = "---\nid: sinclair\nname: 辛克萊\nname_kr: 싱클레어\nmodels: [싱클레어]\nfile_names: [Sinclair]\n---\n\n# 辛克萊\n"
)

// pvGuides are the texts of the guide files the test rules name.
var pvGuides = map[string]string{
	"guides/base.md":                "# 基本\n\n通用規則。\n",
	"guides/families.md":            "# 家族\n\n各家族的規則。\n",
	"guides/characters/faust.md":    pvFaust,
	"guides/characters/sinclair.md": pvSinc,
}

// The test pack. S001A.json is a story: line 2 is pending with everything a
// pending item can carry, line 3 names a keyword and hits no glossary entry,
// line 5 holds U+2028 (as a JSON escape) and line 6 is a copy.
var pvKR = map[string]string{
	"StoryData/S001A.json": `{"dataList":[
 {"id":0,"model":"파우스트","content":"대사 0"},
 {"id":1,"model":"파우스트","content":"대사 1"},
 {"id":2,"model":"파우스트","content":"<color=#f8c200>공격</color> 위력 대사 2"},
 {"id":3,"model":"싱클레어","content":"[Bleed] 5 부여"},
 {"id":4,"model":"파우스트","content":"위력 대사 4"},
 {"id":5,"model":"파우스트","content":"첫 줄 둘째 줄"},
 {"id":6,"model":"파우스트","content":"1234"}
]}`,
	"Other.json": `{"dataList":[
 {"id":0,"content":"<b>굵게</b> & 위력"},
 {"id":1,"content":"새 문장"},
 {"id":2,"content":"검토 대기"},
 {"id":3,"content":"임대 중"}
]}`,
	"Frozen.json": `{"dataList":[{"id":0,"content":"얼린 문장"}]}`,
	"Bufs.json":   `{"dataList":[{"id":"Bleed","name":"출혈"},{"id":"Burn","name":"화상"}]}`,
}

var pvEN = map[string]string{
	"StoryData/S001A.json": `{"dataList":[
 {"id":0,"model":"파우스트","content":"Line 0"},
 {"id":1,"model":"파우스트","content":"Line 1"},
 {"id":2,"model":"파우스트","content":"<color=#f8c200>Attack</color> Potency line 2"},
 {"id":3,"model":"싱클레어","content":"Inflict 5 [Bleed]"},
 {"id":4,"model":"파우스트","content":"Potency line 4"},
 {"id":5,"model":"파우스트","content":"First line\nSecond line"},
 {"id":6,"model":"파우스트","content":"1234"}
]}`,
	"Other.json": `{"dataList":[
 {"id":0,"content":"<b>Bold</b> & Potency"},
 {"id":1,"content":"New sentence"},
 {"id":2,"content":"Awaiting review"},
 {"id":3,"content":"Leased"}
]}`,
}

var pvJP = map[string]string{
	"StoryData/S001A.json": `{"dataList":[
 {"id":0,"model":"파우스트","content":"行0"},
 {"id":1,"model":"파우스트","content":"行1"},
 {"id":2,"model":"파우스트","content":"<color=#f8c200>攻撃</color> 威力 行2"},
 {"id":3,"model":"싱클레어","content":"[Bleed] を5付与"},
 {"id":4,"model":"파우스트","content":"威力 行4"},
 {"id":5,"model":"파우스트","content":"一行目\n二行目"},
 {"id":6,"model":"파우스트","content":"1234"}
]}`,
}

// pvZH is the built pack: the translation of a unit that has one (line 1 of the
// story differs from its record, as a pack built before the record changed
// would), the EN or KR text where there is none.
var pvZH = map[string]string{
	"StoryData/S001A.json": `{"dataList":[
 {"id":0,"model":"파우스트","content":"第零行"},
 {"id":1,"model":"파우스트","content":"第一行（舊版）"},
 {"id":2,"model":"파우스트","content":"<color=#f8c200>Attack</color> Potency line 2"},
 {"id":3,"model":"싱클레어","content":"Inflict 5 [Bleed]"},
 {"id":4,"model":"파우스트","content":"威力 第四行"},
 {"id":5,"model":"파우스트","content":"First line\nSecond line"},
 {"id":6,"model":"파우스트","content":"1234"}
]}`,
	"Other.json": `{"dataList":[
 {"id":0,"content":"<b>粗體</b> & 威力"},
 {"id":1,"content":"New sentence"},
 {"id":2,"content":"Awaiting review"},
 {"id":3,"content":"Leased"}
]}`,
	"Frozen.json": `{"dataList":[{"id":0,"content":"얼린 문장"}]}`,
	"Bufs.json":   `{"dataList":[{"id":"Bleed","name":"流血"},{"id":"Burn","name":"화상"}]}`,
}

// pvWorld is the test pack with its store, the Previewer over it and the
// Inputs Export takes.
type pvWorld struct {
	In  Inputs
	Env batch.Env
}

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, text := range files {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func pvRec(file, pointer, kr, zh, origin string) store.Record {
	return store.Record{Key: units.Key(file, pointer), KRSha: units.KRSha(kr), ZH: zh, Origin: origin, Since: "raw-20261029"}
}

// newPvWorld builds the test pack, its store and a Previewer; frozen are the
// frozen file patterns.
func newPvWorld(t *testing.T, frozen ...string) pvWorld {
	t.Helper()
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"guides/routing.yaml":           pvRouting,
		"glossary/terms.yaml":           pvGlossary,
		"guides/characters/faust.md":    pvFaust,
		"guides/characters/sinclair.md": pvSinc,
	})
	r, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	rawRoot := t.TempDir()
	for lang, set := range map[string]struct {
		prefix string
		files  map[string]string
	}{"kr": {"KR_", pvKR}, "en": {"EN_", pvEN}, "jp": {"JP_", pvJP}} {
		files := map[string]string{}
		for name, text := range set.files {
			dir, base := filepath.Split(name)
			files[filepath.ToSlash(filepath.Join(lang, dir, set.prefix+base))] = text
		}
		if err := os.MkdirAll(filepath.Join(rawRoot, lang), 0o755); err != nil {
			t.Fatal(err)
		}
		writeTree(t, rawRoot, files)
	}
	src, err := pack.OpenRaw(rawRoot)
	if err != nil {
		t.Fatal(err)
	}
	st := store.Open(filepath.Join(t.TempDir(), "tw"))
	story, other := "StoryData/S001A.json", "Other.json"
	kr := func(file, pointer string) string {
		doc, err := src.KR.Doc(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range doc.Strings() {
			if f.Pointer == pointer {
				return f.Value
			}
		}
		t.Fatalf("%s has no %s", file, pointer)
		return ""
	}
	at := func(file string, i int) string { return "/dataList/" + strconv.Itoa(i) + "/content" }
	rec := func(file string, i int, zh, origin string) store.Record {
		p := at(file, i)
		return pvRec(file, p, kr(file, p), zh, origin)
	}
	full := rec(story, 0, "第零行", store.OriginClaude)
	full.Model, full.Batch, full.Rules, full.Flags = "test-model", "20261030-0001", "abc1234", []string{"checked"}
	stale := rec(other, 0, "<b>粗體</b> & 威力", store.OriginParatranz)
	stale.KRSha = units.KRSha("옛 원문")
	records := map[string]store.Record{}
	for _, r := range []store.Record{
		full,
		rec(story, 1, "第一行", store.OriginClaude),
		rec(story, 4, "威力 第四行", store.OriginClaude),
		rec(story, 6, "1234", store.OriginCopy),
		pvRec("Bufs.json", "/dataList/0/name", "출혈", "流血", store.OriginParatranz),
		stale,
	} {
		records[r.Key] = r
	}
	if err := st.SaveRecords(records); err != nil {
		t.Fatal(err)
	}
	if err := st.SavePending([]store.PendingItem{
		{
			Key: units.Key(story, at(story, 2)), Reason: "kr-changed", PrevKR: "옛 대사 2", PrevZH: "舊的第二行",
			Draft: "<color=#f8c200>攻擊</color> 威力 第二行", Hint: "語氣再正式一點", Refs: map[string]string{"llc": "參考譯文", "ltm": ""},
		},
		{Key: units.Key(story, at(story, 3)), Reason: "new"},
		{Key: units.Key(story, at(story, 5)), Reason: "new"},
		{
			Key: units.Key(other, at(other, 2)), Reason: "structural-fail", Attempts: 3,
			LastError: "tags: <b> missing", Flags: []string{store.FlagNeedsReview},
		},
		{
			Key: units.Key(other, at(other, 3)), Reason: "new",
			Lease: &store.Lease{Batch: "20261101-0002", Until: time.Date(2026, 11, 1, 5, 0, 0, 0, time.UTC)},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AppendNotes([]store.Note{
		{Key: units.Key(other, at(other, 0)), Batch: "b1", KRSha: units.KRSha(kr(other, at(other, 0))), Model: "test-model", Note: "<b> 保留標籤"},
		{Key: units.Key(other, at(other, 0)), Batch: "b2", KRSha: units.KRSha(kr(other, at(other, 0))), Model: "test-model", Note: "第二則 & 備註"},
		{Key: units.Key(other, at(other, 0)), Batch: "b0", KRSha: units.KRSha("옛 원문"), Model: "test-model", Note: "舊原文的備註"},
	}); err != nil {
		t.Fatal(err)
	}
	loaded, err := st.LoadRecords()
	if err != nil {
		t.Fatal(err)
	}
	env := batch.Env{
		Store: st, Src: src, Rules: r, RawTag: "raw-20261029", RulesCommit: pvRulesCommit,
		Now: time.Date(2026, 11, 1, 3, 4, 5, 0, time.UTC),
	}
	pv, err := batch.NewPreviewer(env, frozen)
	if err != nil {
		t.Fatal(err)
	}
	zhRoot := t.TempDir()
	writeTree(t, zhRoot, pvZH)
	zh, err := pack.Open(zhRoot, "")
	if err != nil {
		t.Fatal(err)
	}
	return pvWorld{
		Env: env,
		In: Inputs{
			KR: src.KR, EN: src.EN, JP: src.JP, ZH: zh, Records: loaded, Rules: r,
			Preview: &PreviewInputs{Previewer: pv, RulesCommit: pvRulesCommit, Prompt: pvPrompt, Guides: pvGuides},
		},
	}
}

// encodeBatch writes v as lct batch next writes a batch file (writeJSONFile).
func encodeBatch(t *testing.T, v any) string {
	t.Helper()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// rebuildBatch is how the viewer builds the batch of unit i of a file from the
// files lct site wrote: file is files/<path>, pf preview/<path>, shared
// preview.json.
func rebuildBatch(t *testing.T, file File, pf PreviewFile, shared PreviewShared, i int) *batch.Batch {
	t.Helper()
	u, pu := file.Units[i], pf.Units[i]
	key := units.Key(file.Path, u.P)
	b := &batch.Batch{
		ID: batch.PreviewID, Family: pf.Family, GlossaryCheck: pf.GlossaryCheck,
		File: file.Path, Files: []string{file.Path},
		Guides: []string{}, Glossary: []rules.Term{}, Keywords: map[string]string{}, Context: []batch.ContextLine{},
	}
	for _, g := range pu.G {
		b.Guides = append(b.Guides, shared.Guides[g].Path)
	}
	for _, n := range pu.T {
		term := pf.Terms[n]
		term.Units = []string{key}
		b.Glossary = append(b.Glossary, term)
	}
	for _, id := range pu.K {
		b.Keywords[id] = pf.Keywords[id]
	}
	bu := batch.BatchUnit{
		Key: key, File: file.Path, Field: u.F, Speaker: pu.SP, KR: u.KR,
		Reason: pu.It.Reason, PrevKR: pu.It.PrevKR, PrevZH: pu.It.PrevZH, Draft: pu.It.Draft, Hint: pu.It.Hint,
		Refs: pu.It.Refs, Keep: pu.Keep,
	}
	if pu.EM != "" {
		bu.EN = &align.LangAligned{Value: *u.EN, Method: align.Method(pu.EM)}
	}
	if pu.JM != "" {
		bu.JP = &align.LangAligned{Value: *u.JP, Method: align.Method(pu.JM)}
	}
	b.Units = []batch.BatchUnit{bu}
	for _, c := range pu.C {
		at := int(c[0].(float64))
		cu, cp := file.Units[at], pf.Units[at]
		zh := cu.ZH
		if len(c) == 2 {
			zh = c[1].(string)
		}
		b.Context = append(b.Context, batch.ContextLine{Key: units.Key(file.Path, cu.P), Speaker: cp.SP, KR: cu.KR, ZH: zh})
	}
	return b
}

func TestExportPreviewRoundTrip(t *testing.T) {
	w := newPvWorld(t, "Frozen.json")
	out := t.TempDir()
	res, err := Export(w.In, Meta{}, out)
	if err != nil {
		t.Fatal(err)
	}
	if res.PreviewBytes == 0 || res.PreviewBytes >= res.Bytes {
		t.Errorf("PreviewBytes = %d of %d, want some, and part of Bytes", res.PreviewBytes, res.Bytes)
	}
	var shared PreviewShared
	readJSON(t, out, "preview.json", &shared)
	if shared.RulesCommit != pvRulesCommit || shared.Prompt != pvPrompt {
		t.Errorf("preview.json = %q %q, want the rules commit and the prompt", shared.RulesCommit, shared.Prompt)
	}
	checked := 0
	for _, path := range w.In.KR.Files() {
		var file File
		var pf PreviewFile
		readJSON(t, out, "files/"+path, &file)
		readJSON(t, out, "preview/"+path, &pf)
		if len(pf.Units) != len(file.Units) {
			t.Fatalf("%s: %d preview units, %d units", path, len(pf.Units), len(file.Units))
		}
		for i, pu := range pf.Units {
			if pu == nil {
				continue
			}
			key := units.Key(path, file.Units[i].P)
			want, err := w.In.Preview.Previewer.Unit(key)
			if err != nil {
				t.Fatal(err)
			}
			got := rebuildBatch(t, file, pf, shared, i)
			if g, w := encodeBatch(t, got), encodeBatch(t, want.Batch); g != w {
				t.Errorf("%s: rebuilt batch differs\n got: %s\nwant: %s", key, g, w)
			}
			u := file.Units[i]
			// The values the batch shares with files/<path>.
			if bu := want.Batch.Units[0]; (bu.EN == nil) != (u.EN == nil) || (bu.EN != nil && bu.EN.Value != *u.EN) ||
				(bu.JP == nil) != (u.JP == nil) || (bu.JP != nil && bu.JP.Value != *u.JP) {
				t.Errorf("%s: aligned EN/JP of the batch %+v %+v, file data %s %s", key, bu.EN, bu.JP, show(u.EN), show(u.JP))
			}
			for _, c := range want.Batch.Context {
				_, pointer, _ := units.SplitKey(c.Key)
				cu := unitAt(t, file, pointer)
				if c.KR != cu.KR || c.Speaker != pf.Units[indexOf(file, pointer)].SP {
					t.Errorf("%s: context line %s = %+v, want the KR and speaker of its unit", key, c.Key, c)
				}
			}
			if want.Batch.Family != pf.Family || want.Batch.GlossaryCheck != pf.GlossaryCheck {
				t.Errorf("%s: family %q glossary_check %q, file %q %q", key, want.Batch.Family, want.Batch.GlossaryCheck, pf.Family, pf.GlossaryCheck)
			}
			checked++
		}
	}
	if checked < 10 {
		t.Errorf("checked %d units, want the whole test pack", checked)
	}
}

func indexOf(f File, pointer string) int {
	for i, u := range f.Units {
		if u.P == pointer {
			return i
		}
	}
	return -1
}

// A copy has no batch: its entry is null.
func TestExportPreviewCopyNull(t *testing.T) {
	w := newPvWorld(t)
	out := t.TempDir()
	if _, err := Export(w.In, Meta{}, out); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "preview", "StoryData", "S001A.json"))
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		Units []json.RawMessage `json:"units"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if len(raw.Units) != 14 {
		t.Fatalf("%d units, want 14", len(raw.Units))
	}
	// Each of the 7 lines has a model and a content unit; the copy is the content of line 6.
	for i, u := range raw.Units {
		if isNull := string(u) == "null"; isNull != (i == 13) {
			t.Errorf("unit %d = %.40s, null only for the copy at 13", i, u)
		}
	}
}

func TestExportPreviewSharedGuides(t *testing.T) {
	w := newPvWorld(t)
	out := t.TempDir()
	if _, err := Export(w.In, Meta{}, out); err != nil {
		t.Fatal(err)
	}
	var shared PreviewShared
	readJSON(t, out, "preview.json", &shared)
	// Files go in the order of the index; the story is first, its first unit
	// has the base guide, the family's and the speaker's.
	var paths []string
	for _, g := range shared.Guides {
		paths = append(paths, g.Path)
		if g.Text != pvGuides[g.Path] {
			t.Errorf("guide %s = %q, want the text given", g.Path, g.Text)
		}
	}
	want := []string{"guides/base.md", "guides/families.md", "guides/characters/faust.md", "guides/characters/sinclair.md"}
	if !reflect.DeepEqual(paths, want) {
		t.Errorf("guides = %v, want %v", paths, want)
	}
	again := t.TempDir()
	if _, err := Export(newPvWorld(t).In, Meta{}, again); err != nil {
		t.Fatal(err)
	}
	a, _ := os.ReadFile(filepath.Join(out, "preview.json"))
	b, _ := os.ReadFile(filepath.Join(again, "preview.json"))
	if !bytes.Equal(a, b) {
		t.Error("the numbering of the guides is not stable")
	}

	// A guide that is named and not given is an error; one never named is not listed.
	short := newPvWorld(t)
	short.In.Preview.Guides = map[string]string{"guides/base.md": "base", "guides/families.md": "families"}
	if _, err := Export(short.In, Meta{}, t.TempDir()); err == nil || !strings.Contains(err.Error(), "guides/characters/faust.md") {
		t.Errorf("Export without a guide that is named: err = %v, want one that names guides/characters/faust.md", err)
	}
}

func TestExportWithoutPreview(t *testing.T) {
	w := newPvWorld(t)
	w.In.Preview = nil
	out := t.TempDir()
	res, err := Export(w.In, Meta{}, out)
	if err != nil {
		t.Fatal(err)
	}
	if res.PreviewBytes != 0 {
		t.Errorf("PreviewBytes = %d, want 0", res.PreviewBytes)
	}
	for _, name := range []string{"preview.json", "preview"} {
		if _, err := os.Stat(filepath.Join(out, name)); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s: %v, want it not written", name, err)
		}
	}
}

// pvCases are the units whose batches the fixtures hold, each with what it
// stands for.
var pvCases = []struct {
	path    string
	pointer string
	why     string
}{
	{"StoryData/S001A.json", "/dataList/2/content", "story: context, speaker guide, <tag>, pending with draft, hint and refs, two glossary entries, a context line with its own zh"},
	{"StoryData/S001A.json", "/dataList/3/content", "story: [ID] with a keyword, no glossary entry, another speaker"},
	{"StoryData/S001A.json", "/dataList/5/content", "story: U+2028 in KR"},
	{"StoryData/S001A.json", "/dataList/4/content", "story: translated, shares a glossary entry with unit 2, record and check"},
	{"Other.json", "/dataList/0/content", "not a story: < > & in KR, stale record, notes"},
	{"Other.json", "/dataList/1/content", "not a story: no record, no pending item (new)"},
	{"Other.json", "/dataList/2/content", "needs_review pending item with attempts and an error"},
	{"Other.json", "/dataList/3/content", "pending item leased to a batch"},
	{"Frozen.json", "/dataList/0/content", "frozen file, glossary_check warn"},
}

func TestPreviewFixturesUpToDate(t *testing.T) {
	w := newPvWorld(t, "Frozen.json")
	out := t.TempDir()
	if _, err := Export(w.In, Meta{}, out); err != nil {
		t.Fatal(err)
	}
	got := map[string][]byte{}
	for _, sub := range []string{"files", "preview"} {
		err := filepath.WalkDir(filepath.Join(out, sub), func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, _ := filepath.Rel(out, p)
			got[filepath.ToSlash(rel)], err = os.ReadFile(p)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(filepath.Join(out, "preview.json"))
	if err != nil {
		t.Fatal(err)
	}
	got["preview.json"] = data

	type caseJSON struct {
		Path  string `json:"path"`
		Index int    `json:"index"`
		Batch string `json:"batch"`
	}
	cases := make([]caseJSON, len(pvCases))
	for n, c := range pvCases {
		var file File
		readJSON(t, out, "files/"+c.path, &file)
		pv, err := w.In.Preview.Previewer.Unit(units.Key(c.path, c.pointer))
		if err != nil {
			t.Fatal(err)
		}
		name := "batches/" + strconv.Itoa(n) + ".json"
		got[name] = []byte(encodeBatch(t, pv.Batch))
		cases[n] = caseJSON{c.path, indexOf(file, c.pointer), name}
	}
	got["cases.json"] = []byte(encodeBatch(t, cases))

	dir := filepath.Join("..", "..", "site", "fixtures", "preview")
	if *updatePreview {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
		for name, data := range got {
			full := filepath.Join(dir, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(full, data, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	have := map[string]bool{}
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		name := filepath.ToSlash(rel)
		have[name] = true
		disk, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if want, ok := got[name]; !ok {
			t.Errorf("site/fixtures/preview/%s is not written any more", name)
		} else if !bytes.Equal(disk, want) {
			t.Errorf("site/fixtures/preview/%s is out of date", name)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("%v (run go test ./internal/site -run TestPreviewFixturesUpToDate -update-preview)", err)
	}
	var missing []string
	for name := range got {
		if !have[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("site/fixtures/preview lacks %v", missing)
	}
	if t.Failed() {
		t.Log("run go test ./internal/site -run TestPreviewFixturesUpToDate -update-preview")
	}
}
