package validate

import (
	"slices"
	"testing"
)

func TestFamilyAndField(t *testing.T) {
	for _, c := range []struct{ file, want string }{
		{"Skills_personality-01.json", "Skills"}, {"Passive.json", "Passive"}, {"Passives-a1c9p1.json", "Passives"},
		{"AbEventsResultLog.json", "AbEventsResultLog"}, {"AbEvents-a1c5p1.json", "AbEvents"},
		{"RPGSystem/Skills-x.json", ""}, {"StoryData/S001A.json", ""},
	} {
		if got := family(c.file); got != c.want {
			t.Errorf("family(%q) = %q, want %q", c.file, got, c.want)
		}
	}
	for _, c := range []struct{ pointer, want string }{
		{"/dataList/3/levelList/1/coinlist/0/coindescs/2/desc", "levelList/coinlist/coindescs/desc"},
		{"/dataList/0/simpleDesc/1/simpleDesc", "simpleDesc/simpleDesc"}, {"/dataList/7/name", "name"}, {"", ""},
	} {
		if got := fieldPath(c.pointer); got != c.want {
			t.Errorf("fieldPath(%q) = %q, want %q", c.pointer, got, c.want)
		}
	}
}

// gameCodes are the codes of the game checks.
var gameCodes = []string{CodeBracketText, CodePageCount, CodeStoryComment, CodeStrayBrace, CodeRegexChar,
	CodeLiteralNewline, CodeReservedValue, CodeNewlineSpace, CodeQmarkCount, CodeBulletCount, CodeRubyForm,
	CodeKeywordLink, CodeLevelDiff}

// codesOf returns the codes of the game checks Check reports, in order.
func codesOf(kr, zh, file, pointer string) []string {
	var out []string
	for _, is := range Check(kr, zh, Context{File: file, Pointer: pointer}) {
		if slices.Contains(gameCodes, is.Code) {
			out = append(out, is.Code)
		}
	}
	return out
}

func TestGameChecksPerUnit(t *testing.T) {
	const skill = "/dataList/0/levelList/0/desc"
	for _, c := range []struct {
		name, file, pointer, kr, zh string
		want                        []string
	}{
		{"bracket text", "Passives.json", "/dataList/0/desc", "본국검술 사용", "使用本國劍術[肉]", []string{CodeBracketText}},
		{"an [ID] is no bracket text", "Passives.json", "/dataList/0/desc", "[Laceration] 1", "1層 [Laceration]", nil},
		{"KR has the bracket too", "Skills.json", skill, "사영 [루주]", "映射 [赤紅]", nil},
		{"brackets outside the processed fields", "Bufs.json", "/dataList/0/desc", "설명", "說明[肉]", nil},
		{"fewer pages", "AbEvents.json", "/dataList/0/successDesc/0", "가\n\n나\n\n다", "甲\n\n乙", []string{CodePageCount}},
		{"more pages", "ActionEvents.json", "/dataList/0/options/1/result/0", "가", "甲\n\n乙", []string{CodePageCount}},
		{"KR Error counts no page", "AbEvents.json", "/dataList/0/eventDesc", "Error", "甲", []string{CodePageCount}},
		{"both one page", "AbEvents.json", "/dataList/0/eventDesc", "가", "甲", nil},
		{"same pages", "AbEvents.json", "/dataList/0/eventDesc", "가\n\n나", "甲\n\n乙", nil},
		{"only result/0 is paged", "ActionEvents.json", "/dataList/0/options/1/result/1", "가", "甲\n\n乙", nil},
		{"not AbEventsResultLog", "AbEventsResultLog.json", "/dataList/0/content", "가", "甲\n\n乙", nil},
		{"story comment", "StoryData/S001A.json", "/dataList/0/content", "말", "話 // 註", []string{CodeStoryComment}},
		{"KR comment kept", "StoryData/S001A.json", "/dataList/0/content", "SE // 쾅", "音效 // 砰", nil},
		{"stray brace", "X.json", "/dataList/0/content", "{0} 획득", "獲得{0}}", []string{CodeStrayBrace}},
		{"regex char", "ChoiceEventEffect.json", "/dataList/0/content", "보상: {reward}", "獎勵({reward})", []string{CodeRegexChar}},
		{"full-width brackets are fine", "ChoiceEventEffect.json", "/dataList/0/content", "보상: {reward}", "獎勵（{reward}）", nil},
		{"literal newline", "X.json", "/dataList/0/content", "가", `甲\n乙`, []string{CodeLiteralNewline}},
		{"KR has it too", "X.json", "/dataList/0/content", `가\n나`, `甲\n乙`, nil},
		{"reserved value", "X.json", "/dataList/0/content", "미상", " unknown ", []string{CodeReservedValue}},
		{"Error", "X.json", "/dataList/0/content", "오류", "Error", []string{CodeReservedValue}},
		{"space before newline", "Personalities.json", "/dataList/0/title", "R사\n제4무리", "R公司 \n第四集團軍", []string{CodeNewlineSpace}},
		{"full-width space after newline", "Egos.json", "/dataList/0/name", "가\n나", "甲\n　乙", []string{CodeNewlineSpace}},
		{"no space", "Personalities.json", "/dataList/0/title", "가\n나", "甲\n乙", nil},
		{"other fields may", "Personalities.json", "/dataList/0/desc", "가\n나", "甲 \n乙", nil},
		{"sin type question marks", "Enemies.json", "/dataList/0/name", "탐식 죄종??", "暴食罪孽？？", []string{CodeQmarkCount}},
		{"kept", "Enemies.json", "/dataList/0/name", "탐식 죄종?", "暴食罪孽?", nil},
		{"other names use the full-width mark", "Enemies.json", "/dataList/0/name", "히스클리프?", "希斯克利夫？", nil},
		{"bullets", "StoryTheaterDanteNoteDetail.json", "/dataList/0/desc", "가\n- 나\n- 다", "甲\n- 乙\n丙", []string{CodeBulletCount}},
		{"short ruby", "StoryData/S001A.json", "/dataList/0/content", "말", "<r=讀>話</r>", []string{CodeRubyForm}},
		{"ruby without a reading", "StoryData/S001A.json", "/dataList/0/content", "말", "<ruby>話</ruby>", []string{CodeRubyForm}},
		{"upper case", "StoryData/S001A.json", "/dataList/0/content", "말", "<Ruby=讀>話</Ruby>", []string{CodeRubyForm}},
		{"a proper ruby", "StoryData/S001A.json", "/dataList/0/content", "말", "<ruby=讀>話</ruby>", nil},
	} {
		if got := codesOf(c.kr, c.zh, c.file, c.pointer); !slices.Equal(got, c.want) {
			t.Errorf("%s: codes = %v, want %v", c.name, got, c.want)
		}
	}
}

// Without a pointer the checks that look at the field are skipped; the others still run.
func TestGameChecksWithoutPointer(t *testing.T) {
	if got := codesOf("가\n\n나", "甲", "AbEvents.json", ""); got != nil {
		t.Errorf("codes = %v, want none", got)
	}
	if got := codesOf("가", `甲\n`, "", ""); !slices.Equal(got, []string{CodeLiteralNewline}) {
		t.Errorf("codes = %v, want literal-newline", got)
	}
}

// The new issues are warnings and come after the ones Check had before.
func TestGameChecksAreWarningsAndLast(t *testing.T) {
	issues := Check("가...", "甲..."+`\n`, Context{})
	if len(issues) < 2 {
		t.Fatalf("issues = %v, want at least two", codes(issues))
	}
	last := issues[len(issues)-1]
	if last.Code != CodeLiteralNewline || last.Severity != SeverityWarn || last.Class != ClassStyle {
		t.Errorf("last issue = %+v, want a literal-newline style warning", last)
	}
	ell := slices.IndexFunc(issues, func(is Issue) bool { return is.Code == CodeASCIIEllipsis })
	if ell < 0 || ell >= len(issues)-1 {
		t.Errorf("codes = %v, want ascii-ellipsis before literal-newline", codes(issues))
	}
}

// KeywordField names the fields the game links status names in: only there
// does "name + space" mean anything.
func TestKeywordField(t *testing.T) {
	for _, c := range []struct {
		file, pointer string
		want          bool
	}{
		{"Skills_personality-01.json", "/dataList/3/levelList/1/desc", true},
		{"Skills.json", "/dataList/3/levelList/1/coinlist/0/coindescs/2/desc", true},
		{"Skills_Enemy.json", "/dataList/3/levelList/1/coinlist/0/coindescs/2/summary", true},
		{"Passives.json", "/dataList/0/desc", true},
		{"Passive-a1c971.json", "/dataList/0/summary", true},
		{"EGOgift_MirrorDungeon.json", "/dataList/0/desc", true},
		{"EGOgift_MirrorDungeon.json", "/dataList/0/simpleDesc/1/simpleDesc", true},
		{"PanicInfo.json", "/dataList/0/panicDescription", true},
		// the names, and the tables the game shows as they are
		{"Skills.json", "/dataList/3/levelList/1/name", false},
		{"Passives.json", "/dataList/0/name", false},
		{"Bufs.json", "/dataList/0/desc", false},
		{"Bufs.json", "/dataList/0/summary", false},
		{"BattleKeywords.json", "/dataList/0/desc", false},
		{"PanicInfo.json", "/dataList/0/flavor", false},
		{"StoryData/S001A.json", "/dataList/0/content", false},
		{"Other.json", "/dataList/0/content", false},
		// an unknown field
		{"Passives.json", "", false},
	} {
		if got := KeywordField(c.file, c.pointer); got != c.want {
			t.Errorf("KeywordField(%q, %q) = %v, want %v", c.file, c.pointer, got, c.want)
		}
	}
}
