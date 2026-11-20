package build

import (
	"reflect"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// keptLines pairs the lines of the old and the new KR text that are the same
// and returns those whose translation is not: the lines the translator
// rewrote although their KR did not change.
func TestKeptLines(t *testing.T) {
	prevKR := "턴 시작시 발동 (편성 기준으로)\n첫 턴이면 부여\n불안정 E.G.O 밀랍날개 상태"
	prevZH := "回合開始時觸發(以編隊為準)\n第一個回合獲得\n不穩定E.G.O::蠟翼狀態"
	kr := "턴 시작시 발동 (대기 인원 포함)\n첫 턴이면 부여\n새 줄\n불안정 E.G.O 밀랍날개 상태"
	zh := "回合開始時觸發(包含待命人員)\n第一個回合獲得\n新的一行\n不穩定E.G.O::蜜蠟之翼狀態"
	want := []KeptLine{{KR: "불안정 E.G.O 밀랍날개 상태", Old: "不穩定E.G.O::蠟翼狀態", New: "不穩定E.G.O::蜜蠟之翼狀態"}}
	if got := keptLines(prevKR, prevZH, kr, zh); !reflect.DeepEqual(got, want) {
		t.Errorf("keptLines = %+v, want %+v", got, want)
	}

	for name, c := range map[string][4]string{
		"nothing rewritten":         {prevKR, prevZH, kr, "回合開始時觸發(包含待命人員)\n第一個回合獲得\n新的一行\n不穩定E.G.O::蠟翼狀態"},
		"one line, its KR changed":  {"공격 5", "攻擊5", "공격 7", "打擊7"},
		"the same text":             {"가\n나", "甲\n乙", "가\n나", "甲\n乙"},
		"old lines do not pair up":  {"가\n나", "甲乙", "가\n다", "子\n丑"},
		"new lines do not pair up":  {"가\n나", "甲\n乙", "가\n다", "子丑"},
		"no previous text":          {"", "", "가\n나", "甲\n乙"},
		"an empty line stays empty": {"가\n\n나", "甲\n\n乙", "가\n\n다", "甲\n\n丙"},
		"only the line ends differ": {"가\n나", "甲 \n乙", "가\n다", "甲\n丙"},
	} {
		if got := keptLines(c[0], c[1], c[2], c[3]); len(got) != 0 {
			t.Errorf("%s: keptLines = %+v, want none", name, got)
		}
	}

	// A line that stands twice pairs in order.
	got := keptLines("가\n나\n가", "甲\n乙\n甲", "가\n다\n가", "甲\n丙\n子")
	if want := []KeptLine{{KR: "가", Old: "甲", New: "子"}}; !reflect.DeepEqual(got, want) {
		t.Errorf("repeated line: keptLines = %+v, want %+v", got, want)
	}
}

const keptKR = `{"dataList":[{"id":1,"desc":"가 (대기 포함)\n노을빛 검 발동\n다"},{"id":2,"desc":"가\n나"},{"id":3,"desc":"가\n나"}]}`

// keptRecords are records of keptKR: one written for this raw tag in which a
// line whose KR is the same was rewritten, one written for it that keeps its
// lines, and an older one with a rewritten line.
func keptRecords() []store.Record {
	a := written(rec("Skills.json", "/dataList/0/desc", "가 (대기 포함)\n노을빛 검 발동\n다", "甲(包含待命)\n發動霞光劍刃\n新丙"))
	a.PrevKR, a.PrevZH = "가 (편성 기준)\n노을빛 검 발동\n다", "甲(以編隊為準)\n發動霞光劍刃\n丙"
	b := written(rec("Skills.json", "/dataList/1/desc", "가\n나", "甲\n新乙"))
	b.PrevKR, b.PrevZH = "가\n옛", "甲\n乙"
	c := rec("Skills.json", "/dataList/2/desc", "가\n나", "子\n新乙")
	c.PrevKR, c.PrevZH = "가\n옛", "甲\n乙"
	return []store.Record{a, b, c}
}

// The build lists the units translated again for this raw tag in which a line
// whose KR did not change was rewritten, with what the translator noted.
func TestBuildListsKeptRewrites(t *testing.T) {
	recs := keptRecords()
	e := nameEnv(t, keptKR, nil, recs...)
	if _, err := store.Open(e.opt.TWRoot).AppendNotes([]store.Note{
		{Key: recs[0].Key, Batch: "b1", KRSha: units.KRSha("가 (대기 포함)\n노을빛 검 발동\n다"), Model: "m", Note: "다：舊譯「丙」改為「新丙」"},
	}); err != nil {
		t.Fatal(err)
	}
	rep := e.run()
	want := []KeptRewrite{{
		Key:   recs[0].Key,
		Lines: []KeptLine{{KR: "다", Old: "丙", New: "新丙"}},
		Note:  "다：舊譯「丙」改為「新丙」",
	}}
	if !reflect.DeepEqual(rep.KeptRewrites, want) {
		t.Errorf("KeptRewrites = %+v, want %+v", rep.KeptRewrites, want)
	}
}

// A name the KR mentions that the translation renders in no existing way is
// not listed when the translation the unit had before, of a KR text that
// mentioned the name too, did the same: the wording is the old one, not this
// update's.
func TestBuildSkipsAMentionRenderedAsBefore(t *testing.T) {
	kr := `{"dataList":[{"id":1,"desc":"노을빛 검 발동\n새 줄"},{"id":2,"desc":"노을빛 검 발동\n새 줄"}]}`
	a := written(rec("Skills.json", "/dataList/0/desc", "노을빛 검 발동\n새 줄", "發動晚霞之劍\n新行"))
	a.PrevKR, a.PrevZH = "노을빛 검 발동", "發動晚霞之劍"
	b := written(rec("Skills.json", "/dataList/1/desc", "노을빛 검 발동\n새 줄", "發動晚霞之劍\n新行"))
	b.PrevKR, b.PrevZH = "노을빛 검 발동", "發動霞光劍刃"
	rep := nameEnv(t, kr, nil, a, b).run()
	want := []NameDiff{{
		KR: "노을빛 검", Existing: wantNameExisting,
		Units: []NameDiffUnit{{Key: b.Key, ZH: "發動晚霞之劍\n新行", Kind: NameDiffMention}},
	}}
	if !reflect.DeepEqual(rep.NameDiffs, want) {
		t.Errorf("NameDiffs = %+v, want %+v", rep.NameDiffs, want)
	}
}

// The review issue has a section of the units in which lines were rewritten
// although their KR did not change.
func TestKeptRewriteItems(t *testing.T) {
	rep := Report{KeptRewrites: []KeptRewrite{
		{Key: "A.json|/dataList/0/desc", Lines: []KeptLine{{KR: "다", Old: "丙", New: "新丙"}, {KR: "라", Old: "丁", New: "新丁"}}, Note: "다：改寫\n第二則"},
		{Key: "B.json|/dataList/1/desc", Lines: []KeptLine{{KR: "가", Old: "甲", New: "子"}}},
	}}
	items := keptRewriteItems(rep)
	want := []string{
		"`A.json|/dataList/0/desc`：2 行，`丙` → `新丙`…等；備註：다：改寫 第二則",
		"`B.json|/dataList/1/desc`：1 行，`甲` → `子`",
	}
	if !reflect.DeepEqual(items, want) {
		t.Errorf("items = %q, want %q", items, want)
	}
}

// The report counts the warnings of the translations written for this raw
// tag apart from all of them, and its table has a column for each.
func TestBuildCountsWarningsOfThisUpdate(t *testing.T) {
	kr := `{"dataList":[{"id":1,"content":"가"},{"id":2,"content":"나"},{"id":3,"content":"다"}]}`
	e := newEnv(t, map[string]string{"A.json": kr}, nil)
	e.save(
		rec("A.json", "/dataList/0/content", "가", "甲가"),          // an older translation with Hangul left
		written(rec("A.json", "/dataList/1/content", "나", "乙나")), // one of this update with Hangul left
		written(rec("A.json", "/dataList/2/content", "다", "丙")),
	)
	rep := e.run()
	if rep.Warnings["hangul"] != 2 || rep.NewWarnings["hangul"] != 1 || len(rep.NewWarnings) != 1 {
		t.Errorf("Warnings = %v, NewWarnings = %v; want 2 hangul, 1 of them new", rep.Warnings, rep.NewWarnings)
	}
	got := section(t, e.readReport(), "### 驗證警告")
	for _, want := range []string{"| Code | 全部 | 這次新譯 |", "| hangul | 2 | 1 |"} {
		if !strings.Contains(got, want) {
			t.Errorf("report lacks %q:\n%s", want, got)
		}
	}
}

// changedSpan shows where two lines differ: what they share at both ends is
// cut down to a few runes, marked with an ellipsis.
func TestChangedSpan(t *testing.T) {
	for _, c := range [][4]string{
		{"甲乙", "甲丙", "甲乙", "甲丙"},
		{"一二三四五六七八九十懷念的往日擁抱：拚點勝利時觸發效果並結束", "一二三四五六七八九十思念的擁抱：拚點勝利時觸發效果並結束",
			"…五六七八九十懷念的往日擁抱：拚點勝…", "…五六七八九十思念的擁抱：拚點勝…"},
		{"攻擊結束後，對所有目標施加", "攻擊結束後，於下回合對所有目標施加", "攻擊結束後，對所有目標施…", "攻擊結束後，於下回合對所有目標施…"},
		{"相同", "相同", "相同", "相同"},
		{"", "新", "", "新"},
	} {
		if old, now := changedSpan(c[0], c[1], 6); old != c[2] || now != c[3] {
			t.Errorf("changedSpan(%q, %q) = %q, %q; want %q, %q", c[0], c[1], old, now, c[2], c[3])
		}
	}
}

// lineHunks returns what changed between the previous and the present text of
// a unit: each run of lines of the KR text that is not in both, with the lines
// of the translations at the same places.
func TestLineHunks(t *testing.T) {
	prevKR := "가 (편성 기준)\n나\n다"
	prevZH := "甲(以編隊為準)\n乙\n丙"
	kr := "가 (대기 포함)\n나\n새 줄\n다"
	zh := "甲(包含待命)\n乙\n新行\n新丙"
	want := []LineHunk{
		{OldKR: "가 (편성 기준)", NewKR: "가 (대기 포함)", OldZH: "甲(以編隊為準)", NewZH: "甲(包含待命)"},
		{OldKR: "", NewKR: "새 줄", OldZH: "", NewZH: "新行"},
	}
	if got := lineHunks(prevKR, prevZH, kr, zh); !reflect.DeepEqual(got, want) {
		t.Errorf("lineHunks = %+v, want %+v", got, want)
	}
	// Lines that went away are a hunk too; neighbours make one hunk.
	got := lineHunks("가\n나\n다\n라", "甲\n乙\n丙\n丁", "가\n새\n라", "甲\n新\n丁")
	if want := []LineHunk{{OldKR: "나\n다", NewKR: "새", OldZH: "乙\n丙", NewZH: "新"}}; !reflect.DeepEqual(got, want) {
		t.Errorf("lineHunks = %+v, want %+v", got, want)
	}
	// A translation that does not have the lines of its KR one for one: the
	// whole text is the hunk.
	got = lineHunks("가\n나", "甲乙", "가\n다", "甲\n丙")
	if want := []LineHunk{{OldKR: "가\n나", NewKR: "가\n다", OldZH: "甲乙", NewZH: "甲\n丙"}}; !reflect.DeepEqual(got, want) {
		t.Errorf("lineHunks = %+v, want %+v", got, want)
	}
	if got := lineHunks("", "", "가", "甲"); got != nil {
		t.Errorf("lineHunks without a previous text = %+v", got)
	}
}

// The build lists every unit translated again for this raw tag with what
// changed in its KR text and in its translation, and how many of its lines
// were rewritten although their KR is the same; the report has the table.
func TestBuildListsRetranslated(t *testing.T) {
	recs := keptRecords()
	e := nameEnv(t, keptKR, nil, recs...)
	rep := e.run()
	want := []Retranslated{
		{Key: recs[0].Key, Kept: 1, Hunks: []LineHunk{{OldKR: "가 (편성 기준)", NewKR: "가 (대기 포함)", OldZH: "甲(以編隊為準)", NewZH: "甲(包含待命)"}}},
		{Key: recs[1].Key, Hunks: []LineHunk{{OldKR: "옛", NewKR: "나", OldZH: "乙", NewZH: "新乙"}}},
	}
	if !reflect.DeepEqual(rep.Retranslated, want) {
		t.Errorf("Retranslated = %+v, want %+v", rep.Retranslated, want)
	}
	got := section(t, e.readReport(), "### 這次重翻的單位與舊譯的差異")
	for _, want := range []string{
		"| 單位 | KR 的變動 | 譯文的變動 | 原文沒變卻被改寫 |",
		"| `Skills.json\\|/dataList/0/desc` | `가 (편성 기준)` → `가 (대기 포함)` | `甲(以編隊為準)` → `甲(包含待命)` | 1 行 |",
		"| `Skills.json\\|/dataList/1/desc` | `옛` → `나` | `乙` → `新乙` | — |",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("report lacks %q:\n%s", want, got)
		}
	}
}

// hunkCell is a cell of the table of retranslated units: the two texts cut
// down to where they differ, a side that has no line said so, and texts that
// are the same said so.
func TestHunkCell(t *testing.T) {
	for _, c := range [][3]string{
		{"甲乙", "甲丙", "`甲乙` → `甲丙`"},
		{"", "新的一行", "（無） → `新的一行`"},
		{"舊的一行", "", "`舊的一行` → （無）"},
		{"吾無法專心", "吾無法專心", "（沒有改）"},
		{"a|b", "a|c", "`a\\|b` → `a\\|c`"},
	} {
		if got := hunkCell(c[0], c[1]); got != c[2] {
			t.Errorf("hunkCell(%q, %q) = %q, want %q", c[0], c[1], got, c[2])
		}
	}
}
