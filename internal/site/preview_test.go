package site

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
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
	"github.com/tool-jx3/limbus-translate/internal/validate"
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
    include: [guides/shared/tone.md]
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
  jp: 攻撃
  en: Attack
  zh: 攻擊
  status: approved
  check: hard
  meaning: 戰鬥中的攻擊行動
  sources: ["game:Bufs.json#/dataList/0/name", "https://limbuscompany.wiki.gg/wiki/Attack"]
- kr: 후보
  zh: 候選
  status: candidate
- kr: 첫째
  jp: [一つ目, 一番目]
  en: [First, Firstly]
  zh: 第一
  status: approved
- kr: 셋째
  en: Third
  zh: 三號
  status: approved
  check: off
  forbidden: [第叁]
`

const (
	pvFaust = "---\nid: faust\nname: 浮士德\nname_kr: 파우스트\nmodels: [파우스트]\nfile_names: [Faust]\n---\n\n# 浮士德\n"
	pvSinc  = "---\nid: sinclair\nname: 辛克萊\nname_kr: 싱클레어\nmodels: [싱클레어]\nfile_names: [Sinclair]\n---\n\n# 辛克萊\n"
)

// pvGuides are the texts of the guide files the test rules name.
var pvGuides = map[string]string{
	"guides/base.md":                "# 基本\n\n通用規則。\n",
	"guides/families.md":            "# 家族\n\n各家族的規則。\n",
	"guides/shared/tone.md":         "# 共用語氣\n\n跨族群的規則。\n",
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
	// Rules.json: its EN is paired by id (the entries are in another order) and
	// its JP by sequence (the ids are other ones); its records show which
	// fields of a record are kept where they differ from the file's.
	"Rules.json": `{"dataList":[{"id":"A","content":"첫째"},{"id":"B","content":"둘째"},{"id":"C","content":"셋째"}]}`,
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
	"Rules.json": `{"dataList":[{"id":"C","content":"Third"},{"id":"A","content":"First"},{"id":"B","content":"Second"}]}`,
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
	"Rules.json": `{"dataList":[{"id":"x","content":"一つ目"},{"id":"y","content":"二つ目"},{"id":"z","content":"三つ目"}]}`,
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
	"Rules.json":  `{"dataList":[{"id":"A","content":"第一"},{"id":"B","content":"第二"},{"id":"C","content":"第三"}]}`,
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
	story, other, rulesFile := "StoryData/S001A.json", "Other.json", "Rules.json"
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
	ruleRec := func(r store.Record, rules, since string) store.Record {
		r.Rules, r.Since = rules, since
		return r
	}
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
		// still Hangul: the check does not accept it, and its text is that of the file
		pvRec("Bufs.json", "/dataList/1/name", "화상", "화상", store.OriginClaude),
		stale,
		ruleRec(rec(rulesFile, 0, "第一", store.OriginClaude), "abc1234", "raw-20261029"),
		ruleRec(rec(rulesFile, 1, "第二", store.OriginClaude), "abc1234", "raw-20261029"),
		// no rules commit, and another since, where the other two have them
		ruleRec(rec(rulesFile, 2, "第三", store.OriginClaude), "", "raw-20261001"),
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

// encodeBatch writes v as lct batch next writes a batch file.
func encodeBatch(t *testing.T, v any) string {
	t.Helper()
	data, err := batch.EncodeFile(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
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
		Guides: []string{}, Glossary: []rules.Term{}, Keywords: map[string]string{}, Names: []batch.NameRef{}, Context: []batch.ContextLine{},
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
	ex := expandPreviewUnit(t, file, pf, i)
	bu := batch.BatchUnit{
		Key: key, File: file.Path, Field: u.F, Speaker: pu.SP, KR: u.KR,
		Reason: pu.It.Reason, PrevKR: pu.It.PrevKR, PrevZH: ex.PrevZH, Draft: pu.It.Draft, Hint: pu.It.Hint,
		Refs: pu.It.Refs, Keep: ex.Keep,
	}
	if ex.EM != "" {
		bu.EN = &align.LangAligned{Value: *u.EN, Method: align.Method(ex.EM)}
	}
	if ex.JM != "" {
		bu.JP = &align.LangAligned{Value: *u.JP, Method: align.Method(ex.JM)}
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

// pvExpanded is a unit of preview/<path> with every default put back, the
// shape the viewer expands the compact format to (site/src/preview/decode.ts).
type pvExpanded struct {
	Keep   batch.Constraints
	PrevZH string
	EM, JM string // "" for none
	State  PreviewState
	Rec    *PreviewRecord // every field set
	Chk    *PreviewCheck  // every field set
}

func expandMethod(t *testing.T, c byte) string {
	t.Helper()
	switch c {
	case '-':
		return ""
	case 'p':
		return string(align.MethodPointer)
	case 'i':
		return string(align.MethodID)
	case 's':
		return string(align.MethodSequence)
	}
	t.Fatalf("method character %q", c)
	return ""
}

func str(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}

// expandPreviewUnit puts back what the compact format of unit i leaves out.
func expandPreviewUnit(t *testing.T, file File, pf PreviewFile, i int) pvExpanded {
	t.Helper()
	pu := pf.Units[i]
	fu := file.Units[i]
	ex := pvExpanded{PrevZH: pu.It.PrevZH}
	if pu.PZ == 1 {
		ex.PrevZH = fu.ZH
	}
	ex.Keep = batch.Constraints{IDs: []string{}, Placeholders: []string{}, Tags: []string{}}
	if pu.Keep != nil {
		ex.Keep = *pu.Keep
	}
	if pu.St != nil {
		ex.State = *pu.St
	}
	if pu.M != "" {
		if len(pu.M) != 2 {
			t.Fatalf("m = %q, want two characters", pu.M)
		}
		ex.EM, ex.JM = expandMethod(t, pu.M[0]), expandMethod(t, pu.M[1])
	}
	if pu.Rec != nil {
		if pf.RD == nil {
			t.Fatalf("a record but no rd")
		}
		r := *pu.Rec
		rec := PreviewRecord{
			Model: r.Model, Batch: r.Batch, Flags: r.Flags, Stale: r.Stale,
			Origin: ptr(str(r.Origin, pf.RD.Origin)), Rules: ptr(str(r.Rules, pf.RD.Rules)), Since: ptr(str(r.Since, pf.RD.Since)),
		}
		ex.Rec = &rec
		chk := PreviewCheck{Verdict: "accept", ZH: ptr(fu.ZH), Issues: []validate.Issue{}}
		if pu.Chk != nil {
			chk.Verdict = pu.Chk.Verdict
			chk.ZH = ptr(str(pu.Chk.ZH, fu.ZH))
			if pu.Chk.Issues != nil {
				chk.Issues = pu.Chk.Issues
			}
		}
		ex.Chk = &chk
	} else if pu.Chk != nil {
		t.Fatalf("a check but no record")
	}
	return ex
}

func ptr[T any](v T) *T { return &v }

// checkExpanded compares an expanded unit with what Previewer.Unit says.
func checkExpanded(t *testing.T, key string, ex pvExpanded, want *batch.UnitPreview) {
	t.Helper()
	ws := want.State
	wantState := PreviewState{Pending: ws.Pending, NeedsReview: ws.NeedsReview, LeasedTo: ws.LeasedTo, Frozen: ws.Frozen, Attempts: ws.Attempts, LastError: ws.LastError}
	if ex.State != wantState {
		t.Errorf("%s: state %+v, want %+v", key, ex.State, wantState)
	}
	if ex.PrevZH != want.Pending.PrevZH {
		t.Errorf("%s: prev_zh %q, want %q", key, ex.PrevZH, want.Pending.PrevZH)
	}
	if (ex.Rec == nil) != (want.Record == nil) || (ex.Chk == nil) != (want.Check == nil) {
		t.Fatalf("%s: record %v check %v, want record %v check %v", key, ex.Rec, ex.Chk, want.Record, want.Check)
	}
	if r := want.Record; r != nil {
		got := *ex.Rec
		if *got.Origin != r.Origin || *got.Rules != r.Rules || *got.Since != r.Since || got.Model != r.Model || got.Batch != r.Batch ||
			!slices.Equal(got.Flags, r.Flags) {
			t.Errorf("%s: record %+v, want %+v", key, got, *r)
		}
	}
	if c := want.Check; c != nil {
		wantIssues := c.Issues
		if wantIssues == nil {
			wantIssues = []validate.Issue{}
		}
		if ex.Chk.Verdict != c.Verdict || *ex.Chk.ZH != c.ZH || !reflect.DeepEqual(ex.Chk.Issues, wantIssues) {
			t.Errorf("%s: check %+v, want %+v", key, *ex.Chk, *c)
		}
		if ex.Rec.Stale != c.StaleKR {
			t.Errorf("%s: stale %v, want %v", key, ex.Rec.Stale, c.StaleKR)
		}
	}
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
			checkExpanded(t, key, expandPreviewUnit(t, file, pf, i), want)
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

// The compact format leaves out what a reader can put back: each of these has to
// happen somewhere in the test pack, or the test of the decoder proves nothing.
func TestExportPreviewOmissions(t *testing.T) {
	w := newPvWorld(t, "Frozen.json")
	out := t.TempDir()
	if _, err := Export(w.In, Meta{}, out); err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	for _, path := range w.In.KR.Files() {
		var file File
		var pf PreviewFile
		readJSON(t, out, "files/"+path, &file)
		readJSON(t, out, "preview/"+path, &pf)
		for _, pu := range pf.Units {
			if pu == nil {
				continue
			}
			note := func(name string, ok bool) {
				if ok {
					seen[name]++
				}
			}
			note("keep omitted", pu.Keep == nil)
			note("keep kept", pu.Keep != nil)
			note("st omitted", pu.St == nil)
			note("st kept", pu.St != nil)
			note("pz", pu.PZ == 1)
			note("prev_zh kept", pu.It.PrevZH != "")
			note("chk omitted with a record", pu.Chk == nil && pu.Rec != nil)
			note("chk kept", pu.Chk != nil)
			note("chk zh omitted", pu.Chk != nil && pu.Chk.ZH == nil)
			note("chk zh kept", pu.Chk != nil && pu.Chk.ZH != nil)
			note("no record", pu.Rec == nil)
			note("rec origin omitted", pu.Rec != nil && pu.Rec.Origin == nil)
			note("rec origin kept", pu.Rec != nil && pu.Rec.Origin != nil)
			note("rec rules omitted", pu.Rec != nil && pu.Rec.Rules == nil)
			note("rec rules kept", pu.Rec != nil && pu.Rec.Rules != nil)
			note("rec since omitted", pu.Rec != nil && pu.Rec.Since == nil)
			note("rec since kept", pu.Rec != nil && pu.Rec.Since != nil)
			note("rec rules kept empty", pu.Rec != nil && pu.Rec.Rules != nil && *pu.Rec.Rules == "" && pf.RD != nil && pf.RD.Rules != "")
			note("m by id", strings.HasPrefix(pu.M, "i") || strings.HasSuffix(pu.M, "i"))
			note("m by sequence", strings.HasPrefix(pu.M, "s") || strings.HasSuffix(pu.M, "s"))
			note("m omitted", pu.M == "")
			note("m kept", pu.M != "")
		}
		if hasRec := slices.ContainsFunc(pf.Units, func(u *PreviewUnit) bool { return u != nil && u.Rec != nil }); hasRec != (pf.RD != nil) {
			t.Errorf("%s: rd %v but records %v", path, pf.RD, hasRec)
		}
	}
	// A record with no rules commit, in a file where the others have one, says
	// so in the file: leaving it out would give it the commit of the others.
	text, err := os.ReadFile(filepath.Join(out, "preview", "Rules.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(text), `"rules":""`) {
		t.Errorf("preview/Rules.json has no %q: %s", `"rules":""`, text)
	}
	for _, name := range []string{
		"keep omitted", "keep kept", "st omitted", "st kept", "pz", "prev_zh kept", "chk omitted with a record", "chk kept",
		"chk zh omitted", "chk zh kept", "no record", "rec origin omitted", "rec origin kept", "rec rules omitted", "rec rules kept",
		"rec rules kept empty", "rec since omitted", "rec since kept", "m omitted", "m kept", "m by id", "m by sequence",
	} {
		if seen[name] == 0 {
			t.Errorf("no unit of the test pack has %q", name)
		}
	}
}

func TestPreviewMostCommon(t *testing.T) {
	for _, c := range []struct {
		in   []string
		want string
	}{
		{nil, ""},
		{[]string{"b", "a", "b"}, "b"},
		{[]string{"b", "a"}, "a"},
		{[]string{"", "x", "x", ""}, ""},
	} {
		if got := mostCommon(c.in); got != c.want {
			t.Errorf("mostCommon(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPreviewMethodChars(t *testing.T) {
	for _, c := range []struct {
		en, jp *align.LangAligned
		want   string
	}{
		{nil, nil, ""},
		{&align.LangAligned{Method: align.MethodPointer}, nil, "p-"},
		{nil, &align.LangAligned{Method: align.MethodID}, "-i"},
		{&align.LangAligned{Method: align.MethodSequence}, &align.LangAligned{Method: align.MethodPointer}, "sp"},
	} {
		got, err := methodChars(c.en, c.jp)
		if err != nil || got != c.want {
			t.Errorf("methodChars = %q, %v, want %q", got, err, c.want)
		}
	}
	if _, err := methodChars(&align.LangAligned{Method: "other"}, nil); err == nil {
		t.Error("an unknown method is no error")
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
	want := []string{"guides/base.md", "guides/families.md", "guides/shared/tone.md", "guides/characters/faust.md", "guides/characters/sinclair.md"}
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
	short.In.Preview.Guides = map[string]string{"guides/base.md": "base", "guides/families.md": "families", "guides/shared/tone.md": "tone"}
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
	{"Rules.json", "/dataList/0/content", "EN aligned by id, JP by sequence; a record like the file's other records; a glossary entry with several jp and en forms"},
	{"Rules.json", "/dataList/2/content", "a record with no rules commit and another since than the file's others; a check: off entry the record lacks, with no term-missing"},
}

// pvGoldUnit is a unit of preview/<path> as the viewer works on it, with every
// default put back (PreviewUnit of site/src/types.ts). The fields the compact
// format keeps as they are (sp, g, t, k, c, nt) are those of the compact unit;
// the others are made from the UnitPreview of the Previewer, never from the
// compact unit, so a decoder that puts a default back wrong is caught.
type pvGoldUnit struct {
	SP   string            `json:"sp"`
	G    []string          `json:"g"`
	T    []rules.Term      `json:"t"`
	K    []string          `json:"k"`
	It   PreviewItem       `json:"it"`
	EM   string            `json:"em,omitempty"`
	JM   string            `json:"jm,omitempty"`
	Keep batch.Constraints `json:"keep"`
	C    [][]any           `json:"c"`
	St   pvGoldState       `json:"st"`
	Rec  *pvGoldRec        `json:"rec,omitempty"`
	Chk  *pvGoldCheck      `json:"chk,omitempty"`
	NT   string            `json:"nt,omitempty"`
}

type pvGoldState struct {
	Pending     bool   `json:"pending"`
	NeedsReview bool   `json:"needs_review"`
	LeasedTo    string `json:"leased_to,omitempty"`
	Frozen      bool   `json:"frozen"`
	Attempts    int    `json:"attempts,omitempty"`
	LastError   string `json:"last_error,omitempty"`
}

type pvGoldRec struct {
	Origin string   `json:"origin"`
	Model  string   `json:"model,omitempty"`
	Batch  string   `json:"batch,omitempty"`
	Since  string   `json:"since,omitempty"`
	Rules  string   `json:"rules,omitempty"`
	Flags  []string `json:"flags,omitempty"`
	Stale  bool     `json:"stale"`
}

type pvGoldCheck struct {
	Verdict string           `json:"verdict"`
	ZH      string           `json:"zh"`
	Issues  []validate.Issue `json:"issues"`
}

type pvGoldFile struct {
	Family        string            `json:"family"`
	GlossaryCheck string            `json:"glossary_check"`
	Keywords      map[string]string `json:"keywords"`
	Units         []*pvGoldUnit     `json:"units"`
}

// goldenPreviewFile is preview/<path> in the shape the viewer expands the
// compact file to, made straight from the Previewer and nothing of the
// encoder: the guides are paths and the glossary entries their content (the
// compact format numbers them), the keywords are those the batches name, and
// the context lines are the numbers of the units in file. The viewer's tests
// compare its decoder with it, after they put the content in place of the
// numbers.
func goldenPreviewFile(t *testing.T, pv *batch.Previewer, file File) pvGoldFile {
	t.Helper()
	out := pvGoldFile{Keywords: map[string]string{}, Units: make([]*pvGoldUnit, len(file.Units))}
	at := make(map[string]int, len(file.Units))
	for i, fu := range file.Units {
		at[fu.P] = i
	}
	for i := range file.Units {
		p, err := pv.Unit(units.Key(file.Path, file.Units[i].P))
		if errors.Is(err, batch.ErrCopy) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		b := p.Batch
		out.Family, out.GlossaryCheck = b.Family, b.GlossaryCheck
		maps.Copy(out.Keywords, b.Keywords)
		bu := p.Batch.Units[0]
		g := &pvGoldUnit{SP: bu.Speaker, G: nonNil(slices.Clone(b.Guides)), T: []rules.Term{}, K: nonNil(slices.Sorted(maps.Keys(b.Keywords))), C: [][]any{}}
		for _, term := range b.Glossary {
			term.Units = nil
			g.T = append(g.T, term)
		}
		for _, line := range b.Context {
			_, pointer, err := units.SplitKey(line.Key)
			if err != nil {
				t.Fatal(err)
			}
			n, ok := at[pointer]
			if !ok {
				t.Fatalf("context line %s is no unit of %s", line.Key, file.Path)
			}
			if line.ZH == file.Units[n].ZH {
				g.C = append(g.C, []any{n})
			} else {
				g.C = append(g.C, []any{n, line.ZH})
			}
		}
		for _, n := range p.Notes {
			if g.NT != "" {
				g.NT += "\n"
			}
			g.NT += n.Note
		}
		g.It = PreviewItem{Reason: bu.Reason, PrevKR: bu.PrevKR, PrevZH: bu.PrevZH, Draft: bu.Draft, Hint: bu.Hint, Refs: bu.Refs}
		if bu.EN != nil {
			g.EM = string(bu.EN.Method)
		}
		if bu.JP != nil {
			g.JM = string(bu.JP.Method)
		}
		g.Keep = bu.Keep
		g.Keep.IDs, g.Keep.Placeholders, g.Keep.Tags = nonNil(g.Keep.IDs), nonNil(g.Keep.Placeholders), nonNil(g.Keep.Tags)
		s := p.State
		g.St = pvGoldState{Pending: s.Pending, NeedsReview: s.NeedsReview, LeasedTo: s.LeasedTo, Frozen: s.Frozen, Attempts: s.Attempts, LastError: s.LastError}
		if r := p.Record; r != nil {
			g.Rec = &pvGoldRec{Origin: r.Origin, Model: r.Model, Batch: r.Batch, Since: r.Since, Rules: r.Rules, Flags: r.Flags, Stale: r.KRSha != units.KRSha(file.Units[i].KR)}
		}
		if c := p.Check; c != nil {
			g.Chk = &pvGoldCheck{Verdict: c.Verdict, ZH: c.ZH, Issues: nonNil(c.Issues)}
		}
		out.Units[i] = g
	}
	return out
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
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
	for _, path := range w.In.KR.Files() {
		var file File
		readJSON(t, out, "files/"+path, &file)
		got["expanded/"+path] = []byte(encodeBatch(t, goldenPreviewFile(t, w.In.Preview.Previewer, file)))
	}

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

// The glossary_check of a file is the one its batches carry (they are made by
// the Previewer), not a second look at the rules.
func TestBuildPreviewFileGlossaryCheck(t *testing.T) {
	w := newPvWorld(t)
	out := t.TempDir()
	if _, err := Export(w.In, Meta{}, out); err != nil {
		t.Fatal(err)
	}
	var file File
	readJSON(t, out, "files/Other.json", &file)
	build := func(set func(i int, b *batch.Batch)) (PreviewFile, error) {
		pvs, err := w.In.Preview.Previewer.File("Other.json")
		if err != nil {
			t.Fatal(err)
		}
		for i, pv := range pvs {
			if pv != nil {
				set(i, pv.Batch)
			}
		}
		g := &guideTable{texts: pvGuides, at: map[string]int{}}
		return buildPreviewFile(file, pvs, g)
	}

	pf, err := build(func(int, *batch.Batch) {})
	if err != nil || pf.GlossaryCheck != "" {
		t.Fatalf("glossary_check = %q, %v, want \"\" (the batches say nothing)", pf.GlossaryCheck, err)
	}
	pf, err = build(func(_ int, b *batch.Batch) { b.GlossaryCheck = rules.GlossaryWarn })
	if err != nil || pf.GlossaryCheck != rules.GlossaryWarn {
		t.Errorf("glossary_check = %q, %v, want %q from the batches", pf.GlossaryCheck, err, rules.GlossaryWarn)
	}
	_, err = build(func(i int, b *batch.Batch) {
		if i == 0 {
			b.GlossaryCheck = rules.GlossaryWarn
		}
	})
	if err == nil || !strings.Contains(err.Error(), "glossary_check") {
		t.Errorf("batches that disagree: err = %v, want one about glossary_check", err)
	}
}

// The fixtures hold a glossary entry with several forms of jp and en, as
// lists in the batch, and an entry of check: off, which the batch carries and
// whose missing rendering the check does not report.
func TestPreviewFixtureFormsAndCheckOff(t *testing.T) {
	w := newPvWorld(t, "Frozen.json")
	pv := w.In.Preview.Previewer
	first, err := pv.Unit(units.Key("Rules.json", "/dataList/0/content"))
	if err != nil {
		t.Fatal(err)
	}
	text := encodeBatch(t, first.Batch)
	if !strings.Contains(text, "\"jp\": [\n        \"一つ目\",\n        \"一番目\"\n      ],") || !strings.Contains(text, "\"en\": [\n        \"First\",\n        \"Firstly\"\n      ],") {
		t.Errorf("the batch of 첫째 lacks its forms as lists:\n%s", text)
	}
	third, err := pv.Unit(units.Key("Rules.json", "/dataList/2/content"))
	if err != nil {
		t.Fatal(err)
	}
	if len(third.Batch.Glossary) != 1 || third.Batch.Glossary[0].Check != rules.CheckOff || third.Batch.Glossary[0].EN[0] != "Third" {
		t.Fatalf("glossary of 셋째 = %+v, want the off entry", third.Batch.Glossary)
	}
	if c := third.Check; c == nil || c.Verdict != "accept" || slices.ContainsFunc(c.Issues, func(is validate.Issue) bool { return is.Code == validate.CodeTermMissing }) {
		t.Errorf("check = %+v, want accepted without term-missing (the record 第三 lacks 三號)", c)
	}
}
