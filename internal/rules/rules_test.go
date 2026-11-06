package rules_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// repoRoot is the repo itself: these tests read the real guides and glossary.
const repoRoot = "../.."

// anyFile is a file no scoped glossary term of the repo applies to.
const anyFile = "Other.json"

func loadRepo(t *testing.T) *rules.Rules {
	t.Helper()
	r, err := rules.Load(repoRoot)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return r
}

// The rules are edited all through phase 1, so nothing here counts the guides
// or the terms: it checks that they load and that a few known entries are in.
func TestLoadRepoRules(t *testing.T) {
	r := loadRepo(t)
	if len(r.Characters) < 1 {
		t.Error("no character guide loaded")
	}
	if len(r.Glossary) < 1 {
		t.Error("no glossary term loaded")
	}
	if i := slices.IndexFunc(r.Glossary, func(g rules.Term) bool { return g.KR == "도약 모듈" }); i < 0 || r.Glossary[i].ZH != "跳躍成長模組" {
		t.Errorf("the glossary should hold 도약 모듈 → 跳躍成長模組, got %+v", r.Glossary)
	}
	ishmael, ok := r.CharacterByID("ishmael")
	if !ok {
		t.Fatal("no ishmael")
	}
	if ishmael.Gender != "female" {
		t.Errorf("ishmael gender = %q, want female", ishmael.Gender)
	}
	if ishmael.Name != "以實瑪利" || ishmael.NameKR != "이스마엘" {
		t.Errorf("ishmael names = %q / %q", ishmael.Name, ishmael.NameKR)
	}
	if ishmael.Path != "guides/characters/ishmael.md" {
		t.Errorf("ishmael path = %q", ishmael.Path)
	}
	if ishmael.Number == nil || *ishmael.Number != 8 || ishmael.Code == nil || *ishmael.Code != "08" {
		t.Errorf("ishmael number/code = %v / %v", ishmael.Number, ishmael.Code)
	}
	// number: null and code: null are kept as nil, and empty lists are fine.
	dante, ok := r.CharacterByID("dante")
	if !ok {
		t.Fatal("no dante")
	}
	if dante.Number != nil || dante.Code != nil || len(dante.FileNames) != 0 {
		t.Errorf("dante number/code/file_names = %v / %v / %v", dante.Number, dante.Code, dante.FileNames)
	}
	// Lint requires every family to be in the priority list exactly once.
	if len(r.Routing.Families) < 1 || len(r.Routing.Priority) != len(r.Routing.Families) {
		t.Errorf("routing has %d families and %d priorities, want at least one family and as many priorities", len(r.Routing.Families), len(r.Routing.Priority))
	}
}

func TestCharacterByModel(t *testing.T) {
	r := loadRepo(t)
	tests := []struct {
		model     string
		id        string // "" = not ok
		otherSelf bool
	}{
		{"이스마엘", "ishmael", false},
		{"이스마엘다침", "ishmael", false},
		{"과거료슈3", "ryoshu", true},
		{"단테2", "dante", false},
		{"프롤로그단테", "dante", true},
		{"뫼르소엄마", "", false},   // Meursault's mother, not Meursault
		{"파우스트와삼조", "", false}, // several people speak at once
		{"료슈딸", "", false},     // Ryoshu's daughter
		{"이스마엘 ", "", false},   // exact match only
		{"", "", false},        // no model
		{"알수없는모델", "", false},  // not in any file
		{"어린히스클리프", "heathcliff", true},
	}
	for _, tc := range tests {
		c, otherSelf, ok := r.CharacterByModel(tc.model)
		if tc.id == "" {
			if ok || c != nil || otherSelf {
				t.Errorf("CharacterByModel(%q) = %v, %v, %v; want no match", tc.model, c, otherSelf, ok)
			}
			continue
		}
		if !ok || c == nil || c.ID != tc.id || otherSelf != tc.otherSelf {
			t.Errorf("CharacterByModel(%q) = %v, %v, %v; want %s, otherSelf=%v", tc.model, c, otherSelf, ok, tc.id, tc.otherSelf)
		}
	}
}

func TestCharacterByFile(t *testing.T) {
	r := loadRepo(t)
	tests := []struct {
		file string
		id   string // "" = no character
	}{
		// file_names, with the spellings the game files really use
		{"PersonalityVoiceDlg/Voice_Meursalut_xxx_10511.json", "meursault"},
		{"AbDlg_Merusault.json", "meursault"},
		{"AbDlg_Meursault.json", "meursault"},
		{"AbDlg_meursault.json", "meursault"}, // case-insensitive
		{"AbDlg_HONGLU.json", "honglu"},
		{"AbDlg_Donquixote.json", "donquixote"},
		{"PersonalityVoiceDlg/Voice_Outist_1_11101.json", "outis"},
		{"EGOVoiceDig/Voice_EGO_Faust_20202_1.json", "faust"},
		{"EGOVoiceDig/Voice_EGO_yisang_1.json", "yisang"},
		// no character is in the file name, so the numeric code decides
		{"StoryData/P11005.json", "sinclair"},
		{"StoryData/P11101.json", "outis"},
		{"StoryData/P11201.json", "gregor"},
		{"StoryData/P10102.json", "yisang"},
		{"StoryData/P10805.json", "ishmael"},
		{"PersonalityVoiceDlg/Voice_Nobody_11203.json", "gregor"},
		{"SomeFile_10405.json", "ryoshu"},
		// the name beats a conflicting number
		{"PersonalityVoiceDlg/Voice_Faust_10511.json", "faust"},
		// nothing to go on
		{"StoryData/S1037B.json", ""},
		{"StoryData/P19901.json", ""},  // no character has code 99
		{"StoryData/P110050.json", ""}, // one digit too many
		{"StoryData/P1100.json", ""},   // one digit too few
		{"AbDlg_Nobody.json", ""},
		{"AbDlg_MeursaultTwo.json", ""}, // the name must end at _ or the end of the name
		{"PersonalityVoiceDlg/Voice_Nobody_xxx.json", ""},
		{"Skills_personality-08.json", ""},
		{"SomeFile_110511.json", ""}, // six digits are not a personality id
		{"MainUIText.json", ""},
	}
	for _, tc := range tests {
		c, ok := r.CharacterByFile(tc.file)
		if tc.id == "" {
			if ok || c != nil {
				t.Errorf("CharacterByFile(%q) = %v, %v; want none", tc.file, c, ok)
			}
			continue
		}
		if !ok || c == nil || c.ID != tc.id {
			t.Errorf("CharacterByFile(%q) = %v, %v; want %s", tc.file, c, ok, tc.id)
		}
	}
}

func TestGlossaryWarnOnly(t *testing.T) {
	r := loadRepo(t)
	for file, want := range map[string]bool{
		"StoryData/S1037B.json":                      true,
		"StoryData/E020X.json":                       true,
		"PersonalityVoiceDlg/Voice_Faust_10211.json": true,
		"EGOVoiceDig/Voice_EGO_Faust_20202_1.json":   true,
		"BattleAnnouncerDlg/Announcer_1.json":        true,
		"StoryTheaterDanteNote_1.json":               true,
		"Skills_Ego_Personality-08.json":             false,
		"Egos.json":                                  false,
		"Other.json":                                 false,
	} {
		if got := r.GlossaryWarnOnly(file); got != want {
			t.Errorf("GlossaryWarnOnly(%q) = %v, want %v", file, got, want)
		}
	}
}

func TestFamily(t *testing.T) {
	r := loadRepo(t)
	tests := map[string]string{
		"StoryData/S1037B.json":                        "story-main",
		"StoryData/S001A.json":                         "story-main",
		"StoryData/1D101A.json":                        "story-main",
		"StoryData/5D302B.json":                        "story-main",
		"BgmLyrics/Foo.json":                           "story-main",
		"AbEvents-a1c5p1.json":                         "story-main",
		"DungeonNode_1.json":                           "story-main",
		"StoryData/PC01A.json":                         "story-interlude",
		"StoryData/E020X.json":                         "story-interlude",
		"StoryData/ES120A.json":                        "story-interlude",
		"StoryData/P11005.json":                        "identity",
		"PersonalityVoiceDlg/Voice_Faust_10211.json":   "identity",
		"BattleSpeechBubbleDlg-twth.json":              "identity",
		"BattleSpeechBubbleDlg.json":                   "identity",
		"EGOVoiceDig/Voice_EGO_Faust_20202_1.json":     "ego-voice",
		"BattleAnnouncerDlg/Announcer_1.json":          "announcer",
		"Skills_personality-08.json":                   "ally",
		"Skills_Ego_Personality-08.json":               "ally",
		"Passives-a1c9p1.json":                         "ally",
		"Egos.json":                                    "ally",
		"Personality.json":                             "ally",
		"Skills_Assist-1.json":                         "ally",
		"Passives_Assist-1.json":                       "ally",
		"Passives_Enemy-twth.json":                     "enemy",
		"Enemies.json":                                 "enemy",
		"Skills_Enemy-1.json":                          "enemy",
		"Skills_Abnormality-1.json":                    "enemy",
		"Passives_Abnormality-1.json":                  "enemy",
		"AbnormalityGuides.json":                       "abnormality",
		"AbDlg_Faust.json":                             "abnormality",
		"AbEvents.json":                                "abnormality",
		"AbEventsResultLog.json":                       "abnormality",
		"StoryTheaterDanteNote.json":                   "dante-note",
		"StoryTheaterDanteNoteDetail_1.json":           "dante-note",
		"AbEvents_Mirror3.json":                        "mirror",
		"ActionEvents_Mirror1.json":                    "mirror",
		"MirrorDungeon-1.json":                         "mirror",
		"EGOgift_MirrorDungeon-StoryTheme_2.json":      "mirror",
		"EGOgift_Other.json":                           "mirror",
		"DungeonStartBuffs.json":                       "mirror",
		"AbEvents-walpu6.json":                         "walpu-refraction",
		"AbEvents-Refraction1.json":                    "walpu-refraction",
		"RailwayDungeonUI_5.json":                      "walpu-refraction",
		"MainUIText.json":                              "other",
		"RPGSystem/rpg-loc-dialog.json":                "other",
		"StoryData/Z1.json":                            "other",
		"":                                             "other",
		"SomethingCompletelyNew/Whatever.json":         "other",
		"StoryData/S1037B.json.bak/extra/deep/path.js": "other",
	}
	for file, want := range tests {
		if got := r.Family(file); got != want {
			t.Errorf("Family(%q) = %q, want %q", file, got, want)
		}
	}
}

func TestPriority(t *testing.T) {
	r := loadRepo(t)
	if !(r.Priority("story-main") < r.Priority("ally") && r.Priority("ally") < r.Priority("enemy") && r.Priority("enemy") < r.Priority("other")) {
		t.Errorf("priorities out of order: main %d, ally %d, enemy %d, other %d",
			r.Priority("story-main"), r.Priority("ally"), r.Priority("enemy"), r.Priority("other"))
	}
	// the order of guides/families.md "照列出的順序"
	want := []string{"story-main", "story-interlude", "ego-voice", "announcer", "identity", "ally", "enemy", "abnormality", "dante-note", "mirror", "walpu-refraction", "other"}
	for i, id := range want {
		if got := r.Priority(id); got != i {
			t.Errorf("Priority(%q) = %d, want %d", id, got, i)
		}
	}
	// an unknown family goes after everything
	if got := r.Priority("no-such-family"); got <= r.Priority("other") {
		t.Errorf("Priority(unknown) = %d, want more than other (%d)", got, r.Priority("other"))
	}
}

func TestGuidesFor(t *testing.T) {
	r := loadRepo(t)
	tests := []struct {
		name     string
		file     string
		speakers []string
		want     []string
	}{
		{
			"story with speakers", "StoryData/S1037B.json", []string{"faust", "dante"},
			[]string{"guides/base.md", "guides/families.md", "guides/characters/faust.md", "guides/characters/dante.md"},
		},
		{
			"no speakers", "MainUIText.json", nil,
			[]string{"guides/base.md", "guides/families.md"},
		},
		{
			"character of the file comes before the speakers and is not repeated",
			"PersonalityVoiceDlg/Voice_Meursalut_xxx_10511.json", []string{"dante", "meursault"},
			[]string{"guides/base.md", "guides/families.md", "guides/characters/meursault.md", "guides/characters/dante.md"},
		},
		{
			"repeated speakers", "StoryData/S1037B.json", []string{"faust", "dante", "faust"},
			[]string{"guides/base.md", "guides/families.md", "guides/characters/faust.md", "guides/characters/dante.md"},
		},
		{
			"unknown speaker is skipped", "StoryData/S1037B.json", []string{"nobody", "dante"},
			[]string{"guides/base.md", "guides/families.md", "guides/characters/dante.md"},
		},
	}
	for _, tc := range tests {
		if got := r.GuidesFor(tc.file, tc.speakers); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: GuidesFor(%q, %v) = %v, want %v", tc.name, tc.file, tc.speakers, got, tc.want)
		}
	}
}

// hitFor returns the hit for the term kr, nil if there is none.
func hitFor(hits []rules.Term, kr string) *rules.Term {
	if i := slices.IndexFunc(hits, func(h rules.Term) bool { return h.KR == kr }); i >= 0 {
		return &hits[i]
	}
	return nil
}

// The text of these tests is picked from terms that are in the glossary today,
// but they only look for those terms among the hits: a term added later may
// well occur in the same sentence.
func TestGlossaryHits(t *testing.T) {
	r := loadRepo(t)
	hit := hitFor(r.GlossaryHits(anyFile, "이 패키지는 한정 판매입니다."), "패키지")
	if hit == nil {
		t.Fatal("the text contains 패키지, but it is not among the hits")
	}
	if hit.ZH != "組合包" || !slices.Contains(hit.Forbidden, "禮包") || hit.Status != "approved" {
		t.Errorf("hit fields = %+v, want 組合包, forbidden 禮包, approved", *hit)
	}

	// Whatever terms the glossary holds, a hit is an approved term that occurs
	// in the text. (That a text without a term has no hit is checked on a
	// glossary of its own, in TestGlossaryHitsSkipCandidates.)
	for _, text := range []string{"이 패키지는 한정 판매입니다.", "아무 용어도 없는 문장", "시즌 보상: 도약 모듈"} {
		for _, h := range r.GlossaryHits(anyFile, text) {
			if !strings.Contains(text, h.KR) || h.Status != "approved" {
				t.Errorf("GlossaryHits(%q) returned %+v: not an approved term of the text", text, h)
			}
		}
	}
	if hits := r.GlossaryHits(anyFile, ""); len(hits) != 0 {
		t.Errorf("empty KR, but hits = %+v", hits)
	}

	// several hits come back in glossary order, not text order
	hits := r.GlossaryHits(anyFile, "시즌 보상: 도약 모듈")
	var got []string
	for _, h := range hits {
		if h.KR == "도약 모듈" || h.KR == "시즌" {
			got = append(got, h.KR)
		}
	}
	if want := []string{"도약 모듈", "시즌"}; !slices.Equal(got, want) {
		t.Errorf("hit order = %v, want %v", got, want)
	}
}

// Candidate terms do not bind until they are approved.
func TestGlossaryHitsSkipCandidates(t *testing.T) {
	root := copyRules(t)
	edit(t, root, "glossary/terms.yaml", func(string) string {
		return `- kr: 시즌
  zh: 賽季
  status: approved
- kr: 자아 파편
  zh: 自我碎片
  status: approved
  forbidden: [人格碎片]
- kr: 전용 파편
  zh: 專屬碎片
  status: candidate
  forbidden: [人格碎片]
`
	})
	r, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	var candidate *rules.Term
	for i := range r.Glossary {
		if r.Glossary[i].KR == "전용 파편" {
			candidate = &r.Glossary[i]
		}
	}
	if candidate == nil || candidate.Status != "candidate" {
		t.Fatalf("the glossary should hold 전용 파편 as a candidate, got %+v", candidate)
	}
	// Loading keeps the candidate; only the hits leave it out.
	if hits := r.GlossaryHits(anyFile, "시즌 8 동기화 · 환상해석 전용 파편 (공용)"); len(hits) != 1 || hits[0].KR != "시즌" {
		t.Errorf("hits = %+v, want only the approved 시즌", hits)
	}
	if hits := r.GlossaryHits(anyFile, "전용 파편"); len(hits) != 0 {
		t.Errorf("a candidate term must not be returned, got %+v", hits)
	}
	// an approved term in the same text is still returned
	hits := r.GlossaryHits(anyFile, "자아 파편과 전용 파편")
	if len(hits) != 1 || hits[0].KR != "자아 파편" || hits[0].Status != "approved" {
		t.Errorf("hits = %+v, want only the approved 자아 파편", hits)
	}
	// a Korean text with none of the terms has no hit
	if hits := r.GlossaryHits(anyFile, "아무 용어도 없는 문장"); len(hits) != 0 {
		t.Errorf("no term in the text, but hits = %+v", hits)
	}

	// and in the real glossary, whatever candidates it holds at the moment
	live := loadRepo(t)
	for _, term := range live.Glossary {
		if term.Status == "candidate" && hitFor(live.GlossaryHits(anyFile, term.KR), term.KR) != nil {
			t.Errorf("the candidate %s is among the hits of its own text", term.KR)
		}
	}
}

// rulesWithGlossary loads the rules of the repo with the glossary replaced by
// glossary.
func rulesWithGlossary(t *testing.T, glossary string) *rules.Rules {
	t.Helper()
	root := copyRules(t)
	edit(t, root, "glossary/terms.yaml", func(string) string { return glossary })
	r, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func zhOf(hits []rules.Term) []string {
	var out []string
	for _, h := range hits {
		out = append(out, h.ZH)
	}
	return out
}

// An entry with match: word hits only where its KR stands as a word: not right
// after a Hangul syllable, and followed by no Hangul or by a particle or an
// ending (이/가/을/를/의/에서, 하다/되다…), so 백화 does not hit 백화점, nor
// 버스 림버스. An entry with match: substring hits inside a longer word.
func TestGlossaryHitsWholeWord(t *testing.T) {
	r := rulesWithGlossary(t, `- kr: 백화
  zh: 白化
  status: approved
  match: word

- kr: 버스
  zh: 巴士
  status: approved
  match: word

- kr: 사무소
  zh: 事務所
  status: approved
  match: substring
`)
	for _, tc := range []struct {
		text string
		want []string
	}{
		{"백화 현상이 일어났다", []string{"白化"}},
		{"백화가 시작되었다", []string{"白化"}},
		{"백화를 막아라", []string{"白化"}},
		{"백화된 몸", []string{"白化"}},
		{"‘백화’라고 부른다", []string{"白化"}},
		{"백화점에 가자", nil},
		{"백화점의 직원이 본 백화", []string{"白化"}},
		{"림버스 컴퍼니", nil},
		{"림버스 컴퍼니의 버스에 탄다", []string{"巴士"}},
		{"정사무소 대표", []string{"事務所"}},
	} {
		if got := zhOf(r.GlossaryHits(anyFile, tc.text)); !slices.Equal(got, tc.want) {
			t.Errorf("GlossaryHits(%q) = %v, want %v", tc.text, got, tc.want)
		}
	}
}

// match takes the values word and substring; anything else is a mistake lint
// reports.
func TestLintGlossaryMatch(t *testing.T) {
	root := copyRules(t)
	edit(t, root, "glossary/terms.yaml", func(s string) string {
		return s + "\n- kr: 백화점\n  zh: 百貨公司\n  status: approved\n  match: words\n"
	})
	msg := lintMessages(rules.Lint(root))
	for _, want := range []string{"glossary/terms.yaml", "백화점", "match", `"words"`} {
		if !strings.Contains(msg, want) {
			t.Errorf("lint output does not mention %q:\n%s", want, msg)
		}
	}
}

// A term with files applies to the files that match; a term without files to
// all of them. Where both are given for one KR text, the term that matches
// the file stands for it alone, so each file has one rendering.
func TestGlossaryHitsByFile(t *testing.T) {
	r := rulesWithGlossary(t, `- kr: 위력
  zh: 威力
  status: approved
  files: ["StoryData/S10*", "Enemies-a1c10*", "RPGSystem/**"]
- kr: 신부
  zh: 牧師
  forbidden: [神父]
  status: approved
  files: ["StoryData/S10*", "*a1c10*"]
- kr: 신부
  zh: 神父
  status: approved
- kr: 시즌
  zh: 賽季
  status: approved
- kr: 후보
  zh: 候選
  status: approved
  files: ["StoryData/S10*"]
- kr: 후보
  zh: 候補
  status: candidate
  files: ["Other.json"]
`)
	const text = "위력 신부 시즌 후보"
	for _, tc := range []struct {
		file string
		want []string
	}{
		{"StoryData/S1001A.json", []string{"威力", "牧師", "賽季", "候選"}},
		{"StoryData/S2001A.json", []string{"神父", "賽季"}}, // only the terms without files
		{"Enemies-a1c10p1.json", []string{"威力", "牧師", "賽季"}},
		{"Skills-a1c10p2.json", []string{"牧師", "賽季"}},
		{"RPGSystem/rpg-loc-dialogue-floor-1.json", []string{"威力", "神父", "賽季"}},
		{"RPGSystem/deeper/x.json", []string{"威力", "神父", "賽季"}}, // "/**" takes the whole directory
		{"Other.json", []string{"神父", "賽季"}},                    // a candidate scoped to the file binds nothing
		{"", []string{"神父", "賽季"}},                              // no file: only the terms for every file
	} {
		if got := zhOf(r.GlossaryHits(tc.file, text)); !slices.Equal(got, tc.want) {
			t.Errorf("GlossaryHits(%q) = %v, want %v", tc.file, got, tc.want)
		}
	}

	// The terms validation enforces are the same terms.
	forbidden := func(file string) []string {
		var out []string
		for _, term := range r.ValidateTerms(file, "신부") {
			out = append(out, term.Forbidden...)
		}
		return out
	}
	if got := forbidden("StoryData/S1001A.json"); !slices.Equal(got, []string{"神父"}) {
		t.Errorf("forbidden in a file of the term = %v, want 神父", got)
	}
	if got := forbidden("StoryData/S2001A.json"); len(got) != 0 {
		t.Errorf("forbidden in another file = %v, want none", got)
	}
}

// Both renderings of 신부 are in the glossary: 牧師 for the files of chapter
// 10, 神父 for the rest.
func TestGlossaryHoldsBothRenderingsOfSinbu(t *testing.T) {
	r := loadRepo(t)
	for _, tc := range []struct{ file, zh string }{
		{"StoryData/S1034B.json", "牧師"},
		{"StoryData/S1034A2.json", "牧師"},
		{"StoryData/S1000B.json", "牧師"},
		{"StoryData/S1037B.json", "牧師"},
		{"Enemies-a1c10p1.json", "牧師"},
		{"Bufs-a1c10p3.json", "牧師"},
		{"Skills_Abnormality-a1c10p3.json", "牧師"},
		{"RPGSystem/rpg-loc-dialogue-floor-1-c2.json", "牧師"},
		{"StoryData/S725B.json", "神父"},
		{"StoryData/S101B.json", "神父"}, // chapter 1, not chapter 10
		{"StoryData/S109B.json", "神父"},
		{"Personality_Get_Condition.json", "神父"},
		{"PersonalityVoiceDlg/Voice_DonQuixote_Bloodfiend_10310.json", "神父"},
		{"Enemies_Refraction5.json", "神父"},
	} {
		var got []string
		for _, h := range r.GlossaryHits(tc.file, "신부님") {
			if h.KR == "신부" {
				got = append(got, h.ZH)
			}
		}
		if !slices.Equal(got, []string{tc.zh}) {
			t.Errorf("신부 in %s = %v, want only %s", tc.file, got, tc.zh)
		}
	}
	// 牧師 forbids 神父, 神父 forbids nothing.
	var forbidden = map[string][]string{}
	for _, term := range r.Glossary {
		if term.KR == "신부" {
			forbidden[term.ZH] = term.Forbidden
		}
	}
	if !slices.Equal(forbidden["牧師"], []string{"神父"}) || len(forbidden["神父"]) != 0 {
		t.Errorf("forbidden = %v, want 牧師 to forbid 神父 and 神父 to forbid nothing", forbidden)
	}
}

func TestToValidateCarriesTheRendering(t *testing.T) {
	term := rules.Term{KR: "우레", ZH: "雷鳴"}
	if got := term.ToValidate().ZH; got != "雷鳴" {
		t.Errorf("ToValidate().ZH = %q, want 雷鳴", got)
	}
}

func TestToValidate(t *testing.T) {
	r := loadRepo(t)
	hit := hitFor(r.GlossaryHits(anyFile, "시즌"), "시즌")
	if hit == nil {
		t.Fatal("시즌 should be an approved term")
	}
	if !slices.Contains(hit.Forbidden, `re:第\d+季`) {
		t.Fatalf("시즌 forbids %v, want the pattern re:第\\d+季 among them", hit.Forbidden)
	}
	want := validate.Term{KR: "시즌", ZH: hit.ZH, Forbidden: hit.Forbidden}
	if got := hit.ToValidate(); !reflect.DeepEqual(got, want) {
		t.Errorf("ToValidate = %+v, want %+v", got, want)
	}
	// and the converted term really drives validate.Check
	ctx := validate.Context{Terms: []validate.Term{hit.ToValidate()}}
	if !validate.HasHard(validate.Check("시즌 8", "第8季", ctx)) {
		t.Error("第8季 should be a hard forbidden-term issue")
	}
	if validate.HasHard(validate.Check("시즌 8", "第8賽季", ctx)) {
		t.Error("第8賽季 should pass")
	}
}

// copyRules copies the files Lint reads into a fresh root.
func copyRules(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	for _, dir := range []string{"guides", "glossary"} {
		src := filepath.Join(repoRoot, dir)
		err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(repoRoot, p)
			if d.IsDir() {
				return os.MkdirAll(filepath.Join(dst, rel), 0o755)
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(dst, rel), data, 0o644)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return dst
}

// edit rewrites one file under root, requiring that the edit changes it.
func edit(t *testing.T, root, rel string, fn func(string) string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	out := fn(string(data))
	if out == string(data) {
		t.Fatalf("edit of %s changed nothing", rel)
	}
	if err := os.WriteFile(p, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
}

func replaceOnce(old, new string) func(string) string {
	return func(s string) string {
		if strings.Count(s, old) != 1 {
			panic("replaceOnce: " + old + " does not occur exactly once")
		}
		return strings.Replace(s, old, new, 1)
	}
}

func lintMessages(errs []error) string {
	var parts []string
	for _, err := range errs {
		parts = append(parts, err.Error())
	}
	return strings.Join(parts, "\n")
}

func TestLintRepoRulesAreClean(t *testing.T) {
	if errs := rules.Lint(repoRoot); len(errs) != 0 {
		t.Errorf("the repo's own rules fail lint:\n%s", lintMessages(errs))
	}
}

func TestLint(t *testing.T) {
	const ryoshu = "guides/characters/ryoshu.md"
	const danteNote = "  - id: dante-note\n    match: [\"StoryTheaterDanteNote*\"]\n    glossary: warn\n    guide: guides/families.md\n"
	const other = "  - id: other\n    match: []\n    guide: guides/families.md\n"
	tests := []struct {
		name string
		file string // file to edit, relative to the root
		edit func(string) string
		want []string // every one must appear in the lint output
	}{
		{"missing name_kr", ryoshu, replaceOnce("name_kr: 료슈\n", ""), []string{ryoshu, "name_kr"}},
		{"missing id", ryoshu, replaceOnce("id: ryoshu\n", ""), []string{ryoshu, "id"}},
		{"missing name", ryoshu, replaceOnce("name: 良秀\n", ""), []string{ryoshu, "name"}},
		{"missing models", ryoshu, func(s string) string {
			lines := strings.Split(s, "\n")
			for i, l := range lines {
				if strings.HasPrefix(l, "models: ") {
					lines = append(lines[:i], lines[i+1:]...)
					break
				}
			}
			return strings.Join(lines, "\n")
		}, []string{ryoshu, "models"}},
		{"empty models", ryoshu, func(s string) string {
			lines := strings.Split(s, "\n")
			for i, l := range lines {
				if strings.HasPrefix(l, "models: ") {
					lines[i] = "models: []"
				}
			}
			return strings.Join(lines, "\n")
		}, []string{ryoshu, "models"}},
		{"unknown field", ryoshu, replaceOnce("gender: female\n", "gender: female\nnickname: 良\n"), []string{ryoshu, "nickname"}},
		{"misspelled field", ryoshu, replaceOnce("file_names:", "file_name:"), []string{ryoshu, "file_name"}},
		{"id differs from the file name", ryoshu, replaceOnce("id: ryoshu\n", "id: ryoushu\n"), []string{ryoshu, "ryoushu"}},
		{"duplicate id", ryoshu, replaceOnce("id: ryoshu\n", "id: yisang\n"), []string{"yisang"}},
		{"model of two characters", ryoshu, replaceOnce("models: [료슈,", "models: [이상, 료슈,"), []string{ryoshu, "이상"}},
		{"no front matter", ryoshu, func(string) string { return "# 良秀\n" }, []string{ryoshu, "front matter"}},
		{"front matter not closed", ryoshu, func(s string) string { return strings.Replace(s, "\n---\n", "\n", 1) }, []string{ryoshu, "front matter"}},
		{"broken front matter", ryoshu, replaceOnce("gender: female\n", "gender: [female\n"), []string{ryoshu}},

		{"term without kr", "glossary/terms.yaml", replaceOnce("- kr: 패키지\n", "- kr: \"\"\n"), []string{"glossary/terms.yaml", "kr"}},
		{"term without zh", "glossary/terms.yaml", replaceOnce("  zh: 組合包\n", ""), []string{"glossary/terms.yaml", "패키지", "zh"}},
		{"bad status", "glossary/terms.yaml", replaceOnce("  zh: 組合包\n  forbidden: [禮包]\n  status: approved\n", "  zh: 組合包\n  forbidden: [禮包]\n  status: draft\n"), []string{"glossary/terms.yaml", "패키지", "status", "draft"}},
		{"missing status", "glossary/terms.yaml", replaceOnce("  forbidden: [禮包]\n  status: approved\n", "  forbidden: [禮包]\n"), []string{"glossary/terms.yaml", "패키지", "status"}},
		{"regexp that does not compile", "glossary/terms.yaml", replaceOnce(`"re:第\\d+季"`, `"re:第(\\d+季"`), []string{"glossary/terms.yaml", "시즌", "第(\\d+季"}},
		{"misspelled term field", "glossary/terms.yaml", replaceOnce("  forbidden: [禮包]\n", "  forbiden: [禮包]\n"), []string{"glossary/terms.yaml", "패키지", "forbiden"}},
		{"glossary is not a list", "glossary/terms.yaml", func(string) string { return "kr: x\n" }, []string{"glossary/terms.yaml"}},

		{"family guide does not exist", "guides/routing.yaml", replaceOnce(danteNote, strings.Replace(danteNote, "guides/families.md", "guides/families/nothing.md", 1)), []string{"guides/routing.yaml", "dante-note", "guides/families/nothing.md"}},
		{"family without guide", "guides/routing.yaml", replaceOnce(danteNote, strings.Replace(danteNote, "    guide: guides/families.md\n", "", 1)), []string{"guides/routing.yaml", "dante-note", "guide"}},
		{"family without id", "guides/routing.yaml", replaceOnce(danteNote, strings.Replace(danteNote, "  - id: dante-note\n    match", "  - match", 1)), []string{"guides/routing.yaml", "id"}},
		{"unknown glossary mode", "guides/routing.yaml", replaceOnce(danteNote, strings.Replace(danteNote, "glossary: warn", "glossary: soft", 1)), []string{"guides/routing.yaml", "dante-note", "glossary", "soft"}},
		{"family id used twice", "guides/routing.yaml", replaceOnce(danteNote, strings.Replace(danteNote, "dante-note", "mirror", 1)), []string{"guides/routing.yaml", "mirror", "twice"}},
		{"family without match is not the last", "guides/routing.yaml", replaceOnce(danteNote, strings.Replace(danteNote, `["StoryTheaterDanteNote*"]`, "[]", 1)), []string{"guides/routing.yaml", "dante-note", "last"}},
		{"priority id not in families", "guides/routing.yaml", replaceOnce("  - story-main\n", "  - story-main\n  - no-such-family\n"), []string{"guides/routing.yaml", "no-such-family"}},
		{"family missing from priority", "guides/routing.yaml", replaceOnce("  - story-main\n", ""), []string{"guides/routing.yaml", "story-main", "priority"}},
		{"duplicate priority", "guides/routing.yaml", replaceOnce("  - story-main\n", "  - story-main\n  - story-main\n"), []string{"guides/routing.yaml", "story-main"}},
		{"bad glob", "guides/routing.yaml", replaceOnce(`"*walpu*"`, `"*walpu[*"`), []string{"guides/routing.yaml", "walpu["}},
		{"catch-all moved to the front", "guides/routing.yaml", func(s string) string {
			s = strings.Replace(s, "\n"+other, "", 1)
			return strings.Replace(s, "families:\n", "families:\n"+other+"\n", 1)
		}, []string{"guides/routing.yaml", `"other"`, "last"}},
		{"no catch-all", "guides/routing.yaml", replaceOnce(other, strings.Replace(other, "match: []", `match: ["*.json"]`, 1)), []string{"guides/routing.yaml", "last family"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := copyRules(t)
			edit(t, root, tc.file, tc.edit)
			errs := rules.Lint(root)
			if len(errs) == 0 {
				t.Fatal("Lint found nothing")
			}
			msg := lintMessages(errs)
			for _, want := range tc.want {
				if !strings.Contains(msg, want) {
					t.Errorf("lint output does not mention %q:\n%s", want, msg)
				}
			}
		})
	}

	t.Run("missing base guide", func(t *testing.T) {
		root := copyRules(t)
		if err := os.Remove(filepath.Join(root, "guides/base.md")); err != nil {
			t.Fatal(err)
		}
		if msg := lintMessages(rules.Lint(root)); !strings.Contains(msg, "guides/base.md") {
			t.Errorf("lint output does not mention guides/base.md:\n%s", msg)
		}
	})
	t.Run("missing files", func(t *testing.T) {
		msg := lintMessages(rules.Lint(t.TempDir()))
		for _, want := range []string{"glossary/terms.yaml", "guides/routing.yaml", "guides/characters"} {
			if !strings.Contains(msg, want) {
				t.Errorf("lint output does not mention %q:\n%s", want, msg)
			}
		}
	})
}

// The files of a term are patterns of file names, and a KR text may have
// several entries as long as at most one is for every file.
func TestLintGlossaryFiles(t *testing.T) {
	const term = "- kr: 패키지\n"
	const second = "\n- kr: 패키지\n  zh: 包裹\n  status: approved\n"
	lint := func(t *testing.T, fn func(string) string) string {
		t.Helper()
		root := copyRules(t)
		edit(t, root, "glossary/terms.yaml", fn)
		return lintMessages(rules.Lint(root))
	}

	t.Run("bad pattern", func(t *testing.T) {
		msg := lint(t, replaceOnce(term, term+"  files: [\"[\"]\n"))
		for _, want := range []string{"glossary/terms.yaml", "패키지", "files", `"["`} {
			if !strings.Contains(msg, want) {
				t.Errorf("lint output does not mention %q:\n%s", want, msg)
			}
		}
	})
	t.Run("empty pattern", func(t *testing.T) {
		if msg := lint(t, replaceOnce(term, term+"  files: [\"\"]\n")); !strings.Contains(msg, "files") {
			t.Errorf("lint output does not mention files:\n%s", msg)
		}
	})
	t.Run("two entries without files", func(t *testing.T) {
		msg := lint(t, func(s string) string { return s + second })
		for _, want := range []string{"glossary/terms.yaml", "패키지", "files"} {
			if !strings.Contains(msg, want) {
				t.Errorf("lint output does not mention %q:\n%s", want, msg)
			}
		}
	})
	t.Run("one of two entries has files", func(t *testing.T) {
		if msg := lint(t, func(s string) string { return s + second + "  files: [\"StoryData/S10*\"]\n" }); msg != "" {
			t.Errorf("lint found problems in a term with a second entry for some files:\n%s", msg)
		}
	})
	t.Run("two entries with files", func(t *testing.T) {
		more := second + "  files: [\"StoryData/S10*\"]\n" + second + "  files: [\"Enemies*\"]\n"
		if msg := lint(t, func(s string) string { return s + more }); msg != "" {
			t.Errorf("lint found problems in a term with entries for some files only:\n%s", msg)
		}
	})
	t.Run("directory pattern", func(t *testing.T) {
		if msg := lint(t, replaceOnce(term, term+"  files: [\"RPGSystem/**\"]\n")); msg != "" {
			t.Errorf("lint rejects a directory pattern:\n%s", msg)
		}
	})
}

// Words that the foreign-name warning leaves alone are a YAML list.
func TestForeignIgnore(t *testing.T) {
	r := loadRepo(t)
	for _, word := range []string{"媽媽", "獨白"} {
		if !slices.Contains(r.ForeignIgnore, word) {
			t.Errorf("ForeignIgnore = %q, want it to hold %s", r.ForeignIgnore, word)
		}
	}

	root := copyRules(t)
	r, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.ForeignIgnore) != 0 {
		t.Errorf("without a rules/foreign-names.ignore.yaml ForeignIgnore = %q, want none", r.ForeignIgnore)
	}
	if errs := rules.Lint(root); len(errs) != 0 {
		t.Errorf("lint of rules without the file: %v", errs)
	}

	write := func(content string) {
		t.Helper()
		p := filepath.Join(root, "rules", "foreign-names.ignore.yaml")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("# 常用詞\n- 媽媽\n- 老師\n")
	if r, err := rules.Load(root); err != nil || !slices.Equal(r.ForeignIgnore, []string{"媽媽", "老師"}) {
		t.Errorf("ForeignIgnore = %q, %v, want [媽媽 老師]", r.ForeignIgnore, err)
	}
	if errs := rules.Lint(root); len(errs) != 0 {
		t.Errorf("lint of a good file: %v", errs)
	}

	for _, tc := range []struct {
		name, content, want string
		loadFails           bool
	}{
		{"not a list", "媽媽: 1\n", "foreign-names.ignore.yaml", true},
		{"not text", "- [媽媽]\n", "foreign-names.ignore.yaml", true},
		{"empty word", "- 媽媽\n- \"\"\n", "empty", false},
		{"word twice", "- 媽媽\n- 媽媽\n", "twice", false},
	} {
		write(tc.content)
		if _, err := rules.Load(root); (err != nil) != tc.loadFails {
			t.Errorf("%s: Load error = %v, want an error: %v", tc.name, err, tc.loadFails)
		}
		if msg := lintMessages(rules.Lint(root)); !strings.Contains(msg, tc.want) {
			t.Errorf("%s: lint output does not mention %q:\n%s", tc.name, tc.want, msg)
		}
	}
}

// adjustmentYAML is one entry of an adjustment file.
func adjustmentYAML(key, sha, rule string) string {
	return "- key: " + key + "\n  raw: raw-20261029\n  kr_sha: " + sha + "\n  zh: 新譯文\n  reason: 測試\n  rule: " + rule + "\n"
}

func TestLintAdjustments(t *testing.T) {
	sha := strings.Repeat("ab", 32)
	const keyA, keyB = "Story.json|/dataList/1/content", "Story.json|/dataList/2/content"
	const fileA, fileB = "rules/adjustments/2026-11-01-a.yaml", "rules/adjustments/2026-11-02-b.yaml"

	write := func(root, rel, content string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	tests := []struct {
		name  string
		files map[string]string
		want  []string // the messages of the errors, each of which must contain these parts
		in    []string // the file each error is about
	}{
		{"valid files", map[string]string{
			fileA:                               adjustmentYAML(keyA, sha, "guides/base.md") + adjustmentYAML(keyB, sha, "one-off"),
			"rules/adjustments/.gitkeep":        "",
			"rules/adjustments/2026-12-01.yaml": "",
		}, nil, nil},
		{"wrong sha", map[string]string{fileA: adjustmentYAML(keyA, "abc", "one-off")}, []string{"kr_sha"}, []string{fileA}},
		{"upper case sha", map[string]string{fileA: adjustmentYAML(keyA, strings.ToUpper(sha), "one-off")}, []string{"kr_sha"}, []string{fileA}},
		{"rule file does not exist", map[string]string{fileA: adjustmentYAML(keyA, sha, "guides/nothing.md")}, []string{"guides/nothing.md"}, []string{fileA}},
		{"rule above the repo", map[string]string{fileA: adjustmentYAML(keyA, sha, "../outside.md")}, []string{"../outside.md"}, []string{fileA}},
		{"rule is a directory", map[string]string{fileA: adjustmentYAML(keyA, sha, "guides")}, []string{"guides"}, []string{fileA}},
		{"same key in two files", map[string]string{
			fileA: adjustmentYAML(keyA, sha, "one-off"),
			fileB: adjustmentYAML(keyA, sha, "one-off"),
		}, []string{keyA, "2026-11-01-a.yaml"}, []string{fileB}},
		{"same key twice in a file", map[string]string{
			fileA: adjustmentYAML(keyA, sha, "one-off") + adjustmentYAML(keyA, sha, "one-off"),
		}, []string{keyA, "entry 1"}, []string{fileA}},
		{"missing reason", map[string]string{fileA: strings.Replace(adjustmentYAML(keyA, sha, "one-off"), "  reason: 測試\n", "", 1)}, []string{"reason", "missing"}, []string{fileA}},
		{"missing key", map[string]string{fileA: strings.Replace(adjustmentYAML(keyA, sha, "one-off"), "- key: "+keyA+"\n  ", "- ", 1)}, []string{"key", "missing"}, []string{fileA}},
		{"empty zh", map[string]string{fileA: strings.Replace(adjustmentYAML(keyA, sha, "one-off"), "zh: 新譯文", `zh: ""`, 1)}, []string{"zh"}, []string{fileA}},
		{"key is not a unit key", map[string]string{fileA: adjustmentYAML("Story.json", sha, "one-off")}, []string{"key", "Story.json"}, []string{fileA}},
		{"misspelled field", map[string]string{fileA: strings.Replace(adjustmentYAML(keyA, sha, "one-off"), "kr_sha:", "kr-sha:", 1)}, []string{"kr-sha"}, []string{fileA, fileA}}, // and kr_sha is missing
		{"entry is not a mapping", map[string]string{fileA: "- just text\n"}, []string{"entry 1", "mapping"}, []string{fileA}},
		{"file is not a list", map[string]string{fileA: "key: x\n"}, []string{"list"}, []string{fileA}},
		{"broken yaml", map[string]string{fileA: "- key: [x\n"}, nil, []string{fileA}},
		{"a file Load would not read", map[string]string{"rules/adjustments/2026-11-01-a.yml": adjustmentYAML(keyA, sha, "one-off")}, []string{".yaml"}, []string{"rules/adjustments/2026-11-01-a.yml"}},
		{"a directory", map[string]string{"rules/adjustments/sub/2026-11-01-a.yaml": adjustmentYAML(keyA, sha, "one-off")}, []string{"sub"}, []string{"rules/adjustments/sub"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := copyRules(t)
			if errs := rules.Lint(root); len(errs) != 0 {
				t.Fatalf("the rules fail lint before the adjustments are added:\n%s", lintMessages(errs))
			}
			for rel, content := range tc.files {
				write(root, rel, content)
			}
			errs := rules.Lint(root)
			if len(errs) != len(tc.in) {
				t.Fatalf("Lint found %d problem(s), want %d (in %v):\n%s", len(errs), len(tc.in), tc.in, lintMessages(errs))
			}
			for i, err := range errs {
				var le rules.LintError
				if !errors.As(err, &le) || le.File != tc.in[i] {
					t.Errorf("error %d = %#v, want one about %s", i, err, tc.in[i])
				}
			}
			msg := lintMessages(errs)
			for _, want := range tc.want {
				if !strings.Contains(msg, want) {
					t.Errorf("lint output does not mention %q:\n%s", want, msg)
				}
			}
		})
	}

	// Several problems in one run: each is reported, in file order.
	t.Run("every problem is reported", func(t *testing.T) {
		root := copyRules(t)
		write(root, fileA, adjustmentYAML(keyA, "abc", "one-off"))
		write(root, fileB, adjustmentYAML(keyB, sha, "guides/nothing.md")+adjustmentYAML(keyA, sha, "one-off"))
		errs := rules.Lint(root)
		msg := lintMessages(errs)
		if len(errs) != 3 || !strings.Contains(errs[0].Error(), "kr_sha") || !strings.Contains(errs[1].Error(), "guides/nothing.md") || !strings.Contains(errs[2].Error(), keyA) {
			t.Errorf("Lint found %d problem(s), want 3 (sha, rule, duplicate key) in order:\n%s", len(errs), msg)
		}
	})
}

func TestLintReportsEveryProblem(t *testing.T) {
	root := copyRules(t)
	edit(t, root, "guides/characters/ryoshu.md", replaceOnce("name_kr: 료슈\n", ""))
	edit(t, root, "guides/characters/faust.md", replaceOnce("id: faust\n", ""))
	edit(t, root, "glossary/terms.yaml", replaceOnce("  zh: 組合包\n", ""))
	msg := lintMessages(rules.Lint(root))
	for _, want := range []string{"ryoshu.md", "faust.md", "패키지"} {
		if !strings.Contains(msg, want) {
			t.Errorf("lint output does not mention %q:\n%s", want, msg)
		}
	}
}

func TestLintErrorsCarryTheFile(t *testing.T) {
	root := copyRules(t)
	edit(t, root, "guides/characters/ryoshu.md", replaceOnce("name_kr: 료슈\n", ""))
	errs := rules.Lint(root)
	if len(errs) != 1 {
		t.Fatalf("errs = %v, want exactly one", errs)
	}
	var le rules.LintError
	if !errors.As(errs[0], &le) || le.File != "guides/characters/ryoshu.md" || !strings.Contains(le.Message, "name_kr") {
		t.Errorf("error = %#v", errs[0])
	}
}

func TestLoadRejectsUnparseableFiles(t *testing.T) {
	tests := []struct {
		name string
		file string
		edit func(string) string
		want string
	}{
		{"glossary", "glossary/terms.yaml", func(s string) string { return "- kr: [unclosed\n" + s }, "glossary/terms.yaml"},
		{"character", "guides/characters/faust.md", replaceOnce("gender: female\n", "gender: [female\n"), "faust.md"},
		{"character without front matter", "guides/characters/faust.md", func(string) string { return "# 浮士德\n" }, "faust.md"},
		{"routing", "guides/routing.yaml", func(s string) string { return "families: [unclosed\n" + s }, "guides/routing.yaml"},
		{"routing glob", "guides/routing.yaml", replaceOnce(`"*walpu*"`, `"*walpu[*"`), "guides/routing.yaml"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := copyRules(t)
			edit(t, root, tc.file, tc.edit)
			_, err := rules.Load(root)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Load error = %v, want one mentioning %s", err, tc.want)
			}
		})
	}
	if _, err := rules.Load(t.TempDir()); err == nil {
		t.Error("Load of an empty directory should fail")
	}
}

// A character file with unknown fields still loads: only Lint rejects them.
func TestLoadToleratesUnknownFields(t *testing.T) {
	root := copyRules(t)
	edit(t, root, "guides/characters/faust.md", replaceOnce("gender: female\n", "gender: female\nnickname: 浮\n"))
	r, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if c, ok := r.CharacterByID("faust"); !ok || c.Name != "浮士德" {
		t.Errorf("faust = %+v, %v", c, ok)
	}
}

// The rule "at most one entry of a KR text for every file" counts the approved
// entries: a candidate binds nothing, so it may stand beside the approved one.
func TestLintGlossaryCountsApprovedOnly(t *testing.T) {
	entry := func(status string) string {
		return "\n- kr: 패키지\n  zh: 包裹\n  status: " + status + "\n"
	}
	lint := func(t *testing.T, add string) string {
		t.Helper()
		root := copyRules(t)
		edit(t, root, "glossary/terms.yaml", func(s string) string { return s + add })
		return lintMessages(rules.Lint(root))
	}

	if msg := lint(t, entry("candidate")); msg != "" {
		t.Errorf("lint found problems in an approved entry with a candidate entry beside it, neither with files:\n%s", msg)
	}
	if msg := lint(t, entry("candidate")+entry("candidate")); msg != "" {
		t.Errorf("lint found problems in two candidate entries beside an approved one:\n%s", msg)
	}
	msg := lint(t, entry("approved"))
	for _, want := range []string{"glossary/terms.yaml", "패키지", "files"} {
		if !strings.Contains(msg, want) {
			t.Errorf("lint output for two approved entries without files does not mention %q:\n%s", want, msg)
		}
	}
	// A candidate does not count towards the limit of the approved ones.
	if msg := lint(t, entry("candidate")+entry("approved")); !strings.Contains(msg, "패키지") {
		t.Errorf("lint missed the second approved entry without files:\n%s", msg)
	}
}

// ScriptRenderings are the approved renderings for a file that hold kana or
// Hangul, whatever the KR says: a line may name an association by a short form
// (하나 for 하나 협회) that its translation writes in full, and the kana and
// Hangul checks leave them alone.
func TestScriptRenderings(t *testing.T) {
	r := loadRepo(t)
	got := r.ScriptRenderings(anyFile)
	for _, want := range []string{"し協會", "하나協會"} {
		if !slices.Contains(got, want) {
			t.Errorf("ScriptRenderings = %q, want %s among them", got, want)
		}
	}
	if slices.Contains(got, "組合包") {
		t.Errorf("組合包 has no kana or Hangul, but ScriptRenderings = %q", got)
	}
}

// provisional marks a term the project registered; it is never a glossary key,
// so a glossary that writes it is reported like any unknown key.
func TestLintRejectsProvisionalInTheGlossary(t *testing.T) {
	root := copyRules(t)
	edit(t, root, "glossary/terms.yaml", func(s string) string {
		return s + "\n- kr: 우레\n  zh: 雷鳴\n  status: approved\n  provisional: true\n"
	})
	msg := lintMessages(rules.Lint(root))
	for _, want := range []string{"glossary/terms.yaml", "provisional"} {
		if !strings.Contains(msg, want) {
			t.Errorf("lint output does not mention %q:\n%s", want, msg)
		}
	}
}

func TestTermOccursIn(t *testing.T) {
	for _, match := range []string{"", rules.MatchWord} {
		word := rules.Term{KR: "속도", Match: match}
		for text, want := range map[string]bool{
			"빠른 속도로": true, "속도가 붙는다": true, "속도": true,
			"약속도": false, "고속도로": false, "도시락": false,
		} {
			if got := word.OccursIn(text); got != want {
				t.Errorf("Term{Match: %q}.OccursIn(%q) = %v, want %v", match, text, got, want)
			}
		}
	}
	if !(rules.Term{KR: "도시"}).OccursIn("도시가") || (rules.Term{KR: "도시"}).OccursIn("도시락") {
		t.Error("OccursIn without match should be a whole-word test")
	}
	sub := rules.Term{KR: "사무소", Match: rules.MatchSubstring}
	if !sub.OccursIn("정사무소") || !sub.OccursIn("사무소") || sub.OccursIn("꽃") {
		t.Error("OccursIn with match: substring should hit inside a longer word")
	}
}

// Endings that follow a vowel-final noun and keep it a word (복도니까요,
// 도시잖아요, 관리자답게…) were measured on the real translations: each hit
// below is a unit that uses the rendering. 예 (날개예요 half the time) and 네
// (no unit) start nouns too (예정, 네트워크) and are not endings of a word.
func TestTermOccursInEndings(t *testing.T) {
	for _, tc := range []struct {
		kr, text string
		want     bool
	}{
		{"복도", "복도니까요.", true},
		{"관리자", "관리자니까.", true},
		{"가주", "가주시다.", true},
		{"언더보스", "언더보스셨다는", true},
		{"도시", "도시잖아요.", true},
		{"주홍십자", "주홍십자겠지?", true},
		{"가주", "가주더라도", true},
		{"날개", "날개든,", true},
		{"지부", "지부거나", true},
		{"낭술회", "낭술회냐며", true},
		{"복도", "복도됐다", true},
		{"관리자", "관리자답게,", true},
		{"유로지비", "유로지비끼리", true},
		{"날개", "날개씩이나", true},
		{"거울", "거울던전", false},
		{"도시", "도시거리", false},
		{"도시", "도시더미", false},
		{"도시", "도시더블", false},
		{"복도", "복도예정", false},
		{"복도", "복도네트워크", false},
		{"엄지", "엄지군", false},
		{"엄지", "엄지분", false},
	} {
		if got := (rules.Term{KR: tc.kr}).OccursIn(tc.text); got != tc.want {
			t.Errorf("Term{KR: %q}.OccursIn(%q) = %v, want %v", tc.kr, tc.text, got, tc.want)
		}
	}
}

// A match value other than word and substring is a lint error (see the "words"
// case of the glossary lint test); where one gets through anyway, the entry
// hits as one without match does. It neither panics nor skips.
func TestTermOccursInWithAnInvalidMatch(t *testing.T) {
	bad := rules.Term{KR: "백화", Match: "words"}
	if bad.OccursIn("백화점") || !bad.OccursIn("백화가") {
		t.Error("OccursIn with an unknown match should be a whole-word test")
	}
}

func TestApplies(t *testing.T) {
	r := rulesWithGlossary(t, "- kr: 도시\n  zh: 都市\n  status: approved\n")
	for _, tc := range []struct {
		name string
		term rules.Term
		file string
		want bool
	}{
		{"no scope", rules.Term{}, "StoryData/S1.json", true},
		{"files hit", rules.Term{Files: []string{"StoryData/**"}}, "StoryData/S1.json", true},
		{"files miss", rules.Term{Files: []string{"StoryData/**"}}, "Bufs.json", false},
		{"not_files hit", rules.Term{NotFiles: []string{"StoryData/**"}}, "StoryData/S1.json", false},
		{"not_files miss", rules.Term{NotFiles: []string{"StoryData/**"}}, "Bufs.json", true},
		{"both: in files, not excluded", rules.Term{Files: []string{"StoryData/**"}, NotFiles: []string{"StoryData/S2*"}}, "StoryData/S1.json", true},
		{"both: in files, excluded", rules.Term{Files: []string{"StoryData/**"}, NotFiles: []string{"StoryData/S2*"}}, "StoryData/S2.json", false},
		{"both: outside files", rules.Term{Files: []string{"StoryData/**"}, NotFiles: []string{"StoryData/S2*"}}, "Bufs.json", false},
		// family:<id> is the files the routing puts in the family, first match wins.
		{"family in files, in family", rules.Term{Files: []string{"family:story-main"}}, "StoryData/S1037B.json", true},
		{"family in files, other family", rules.Term{Files: []string{"family:story-main"}}, "StoryData/E1.json", false},
		{"family in files, shadowed by an earlier family", rules.Term{Files: []string{"family:ally"}}, "Skills_Assist-walpu6.json", false},
		{"family in files, the family it routes to", rules.Term{Files: []string{"family:walpu-refraction"}}, "Skills_Assist-walpu6.json", true},
		{"family in not_files, in family", rules.Term{NotFiles: []string{"family:story-main"}}, "StoryData/S1037B.json", false},
		{"family in not_files, other family", rules.Term{NotFiles: []string{"family:story-main"}}, "StoryData/E1.json", true},
		{"family mixed with a pattern in files, by family", rules.Term{Files: []string{"family:story-main", "Bufs*"}}, "StoryData/S1.json", true},
		{"family mixed with a pattern in files, by pattern", rules.Term{Files: []string{"family:story-main", "Bufs*"}}, "Bufs.json", true},
		{"family mixed with a pattern in files, neither", rules.Term{Files: []string{"family:story-main", "Bufs*"}}, "StoryData/E1.json", false},
		{"family in files, pattern in not_files", rules.Term{Files: []string{"family:story-main"}, NotFiles: []string{"StoryData/S2*"}}, "StoryData/S2.json", false},
		{"unknown family matches nothing", rules.Term{Files: []string{"family:nope"}}, "StoryData/S1.json", false},
		{"unknown family in not_files excludes nothing", rules.Term{NotFiles: []string{"family:nope"}}, "StoryData/S1.json", true},
	} {
		if got := r.Applies(tc.term, tc.file); got != tc.want {
			t.Errorf("%s: Applies(%q) = %v, want %v", tc.name, tc.file, got, tc.want)
		}
	}
}

// An entry whose files is a family replaces the general entry of its KR in the
// files of that family, and only there; not_files with a family agrees.
func TestGlossaryHitsFamilyScopes(t *testing.T) {
	r := rulesWithGlossary(t, `- kr: 신부
  zh: 神父
  status: approved
  not_files: ["family:story-main"]

- kr: 신부
  zh: 牧師
  status: approved
  files: ["family:story-main"]

- kr: 관통
  zh: 貫穿
  status: approved
  files: ["family:ally"]
  forbidden: ["穿透"]

- kr: 관통
  zh: 穿刺
  status: approved
`)
	for file, want := range map[string]string{"StoryData/S1037B.json": "牧師", "StoryData/E1.json": "神父", "Bufs.json": "神父"} {
		if got := zhOf(r.GlossaryHits(file, "신부가 왔다")); !slices.Equal(got, []string{want}) {
			t.Errorf("GlossaryHits(%q) = %v, want [%s]", file, got, want)
		}
	}
	for file, want := range map[string]string{"Skills_personality-1.json": "貫穿", "Skills_Assist-walpu6.json": "穿刺", "Bufs.json": "穿刺"} {
		if got := zhOf(r.GlossaryHits(file, "관통이다")); !slices.Equal(got, []string{want}) {
			t.Errorf("GlossaryHits(%q) = %v, want [%s]", file, got, want)
		}
	}
}

func TestScriptRenderingsFamilyScope(t *testing.T) {
	r := rulesWithGlossary(t, "- kr: 하나\n  zh: 하나協會\n  status: approved\n  not_files: [\"family:story-main\"]\n")
	if got := r.ScriptRenderings("StoryData/S1.json"); len(got) != 0 {
		t.Errorf("ScriptRenderings in the excluded family = %v, want none", got)
	}
	if got := r.ScriptRenderings("Bufs.json"); !slices.Equal(got, []string{"하나協會"}) {
		t.Errorf("ScriptRenderings elsewhere = %v, want the rendering", got)
	}
}

// Two complementary entries of one KR: 神父 for every file but the chapter,
// 牧師 for the chapter alone (the entry with files stands for its KR there, and
// the other one's not_files agrees). Each file has exactly one.
func TestGlossaryHitsNotFilesComplement(t *testing.T) {
	r := rulesWithGlossary(t, `- kr: 신부
  zh: 神父
  status: approved
  not_files: ["StoryData/ch10*"]

- kr: 신부
  zh: 牧師
  status: approved
  files: ["StoryData/ch10*"]
`)
	for file, want := range map[string]string{"StoryData/ch10_a.json": "牧師", "StoryData/ch09.json": "神父", "Bufs.json": "神父"} {
		if got := zhOf(r.GlossaryHits(file, "신부가 왔다")); !slices.Equal(got, []string{want}) {
			t.Errorf("GlossaryHits(%q) = %v, want [%s]", file, got, want)
		}
		if got := r.ValidateTerms(file, "신부가 왔다"); len(got) != 1 || got[0].ZH != want {
			t.Errorf("ValidateTerms(%q) = %+v, want [%s]", file, got, want)
		}
	}
}

// not_files with files: only the files that match files and not not_files.
func TestGlossaryHitsFilesAndNotFiles(t *testing.T) {
	r := rulesWithGlossary(t, `- kr: 위력
  zh: 威力
  status: approved
  files: ["StoryData/**"]
  not_files: ["StoryData/S2*"]
`)
	for file, want := range map[string]int{"StoryData/S1.json": 1, "StoryData/S2.json": 0, "Bufs.json": 0} {
		if got := len(r.GlossaryHits(file, "위력 5")); got != want {
			t.Errorf("GlossaryHits(%q) = %d hits, want %d", file, got, want)
		}
	}
}

// not_files on a term with kana or Hangul in its rendering hides it from
// ScriptRenderings in the excluded files, as it does from GlossaryHits.
func TestScriptRenderingsNotFiles(t *testing.T) {
	r := rulesWithGlossary(t, `- kr: 하나 협회
  zh: 하나協會
  status: approved
  not_files: ["StoryData/**"]
`)
	if got := r.ScriptRenderings("StoryData/S1.json"); len(got) != 0 {
		t.Errorf("ScriptRenderings in an excluded file = %v, want none", got)
	}
	if got := r.ScriptRenderings("Bufs.json"); !slices.Equal(got, []string{"하나協會"}) {
		t.Errorf("ScriptRenderings = %v, want the rendering", got)
	}
}

func TestLintGlossaryNotFiles(t *testing.T) {
	for name, tc := range map[string]struct{ entry, want string }{
		"bad pattern":   {"  not_files: [\"[\"]\n", "not_files pattern"},
		"empty pattern": {"  not_files: [\"\"]\n", "not_files holds an empty pattern"},
		"same pattern":  {"  files: [\"StoryData/**\"]\n  not_files: [\"StoryData/**\"]\n", "applies to no file"},
	} {
		root := copyRules(t)
		edit(t, root, "glossary/terms.yaml", func(s string) string {
			return s + "\n- kr: 백화점\n  zh: 百貨公司\n  status: approved\n" + tc.entry
		})
		if msg := lintMessages(rules.Lint(root)); !strings.Contains(msg, tc.want) || !strings.Contains(msg, "백화점") {
			t.Errorf("%s: lint output does not mention %q:\n%s", name, tc.want, msg)
		}
	}
	// Valid entries, and the old match: word, are clean.
	root := copyRules(t)
	edit(t, root, "glossary/terms.yaml", func(s string) string {
		return s + "\n- kr: 백화점\n  zh: 百貨公司\n  status: approved\n  match: word\n  files: [\"StoryData/**\"]\n  not_files: [\"StoryData/S2*\"]\n" +
			"\n- kr: 사무소2\n  zh: 事務所\n  status: approved\n  match: substring\n"
	})
	if msg := lintMessages(rules.Lint(root)); strings.Contains(msg, "백화점") || strings.Contains(msg, "사무소2") {
		t.Errorf("lint reports a valid entry:\n%s", msg)
	}
}

// An unknown family id in files or not_files is a lint error; known ids and
// mixing with patterns are clean.
func TestLintGlossaryFamilyScope(t *testing.T) {
	for name, tc := range map[string]struct{ entry, want string }{
		"unknown in files":     {"  files: [\"family:nope\"]\n", `family "nope"`},
		"unknown in not_files": {"  not_files: [\"Bufs*\", \"family:story-mian\"]\n", `family "story-mian"`},
		"empty id":             {"  files: [\"family:\"]\n", "family"},
		"same family":          {"  files: [\"family:ally\"]\n  not_files: [\"family:ally\"]\n", "applies to no file"},
	} {
		root := copyRules(t)
		edit(t, root, "glossary/terms.yaml", func(s string) string {
			return s + "\n- kr: 백화점\n  zh: 百貨公司\n  status: approved\n" + tc.entry
		})
		if msg := lintMessages(rules.Lint(root)); !strings.Contains(msg, tc.want) || !strings.Contains(msg, "백화점") {
			t.Errorf("%s: lint output does not mention %q:\n%s", name, tc.want, msg)
		}
	}
	root := copyRules(t)
	edit(t, root, "glossary/terms.yaml", func(s string) string {
		return s + "\n- kr: 백화점\n  zh: 百貨公司\n  status: approved\n  files: [\"family:story-main\", \"Bufs*\"]\n  not_files: [\"family:walpu-refraction\"]\n"
	})
	if msg := lintMessages(rules.Lint(root)); strings.Contains(msg, "백화점") {
		t.Errorf("lint reports a valid family scope:\n%s", msg)
	}
}

// A family: token warns when no file of the KR pack belongs to the family.
func TestLintScopeWarningsFamily(t *testing.T) {
	root := copyRules(t)
	edit(t, root, "glossary/terms.yaml", func(string) string {
		return "- kr: 속도\n  zh: 速度\n  status: approved\n  files: [\"family:ally\", \"family:mirror\", \"family:nope\"]\n  not_files: [\"family:story-main\"]\n"
	})
	files := []string{"Skills_Ego-1.json", "StoryData/S1.json"}
	var msgs []string
	for _, w := range rules.LintScopeWarnings(root, files) {
		msgs = append(msgs, w.Message)
	}
	if len(msgs) != 1 || !strings.Contains(msgs[0], `"family:mirror"`) || !strings.Contains(msgs[0], "files") || !strings.Contains(msgs[0], `"속도"`) {
		t.Errorf("warnings = %q, want one for family:mirror in files and none for the unknown family:nope", msgs)
	}
}

// Tags are not text: a KR term cut by a tag, or wrapped in one, is found.
func TestGlossaryHitsIgnoreTagsInKR(t *testing.T) {
	root := copyRules(t)
	edit(t, root, "glossary/terms.yaml", func(string) string {
		return "- kr: 우레\n  zh: 雷鳴\n  status: approved\n- kr: 백화\n  zh: 白花\n  status: approved\n  match: word\n"
	})
	r, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, kr := range []string{"우<b>레</b>가", "<b>우레</b>가", "<color=#ffffff>백</color>화가 핀다"} {
		if len(r.GlossaryHits(anyFile, kr)) != 1 || len(r.ValidateTerms(anyFile, kr)) != 1 {
			t.Errorf("GlossaryHits(%q) = %+v, want the one entry", kr, r.GlossaryHits(anyFile, kr))
		}
	}
	if hits := r.GlossaryHits(anyFile, "<b>백화</b>점"); len(hits) != 0 {
		t.Errorf("GlossaryHits = %+v, want none: 백화점 is a longer word", hits)
	}
}

// A self-closing tag stands between words: it does not fuse them into a term.
func TestGlossaryHitsSelfClosingTagSeparatesWords(t *testing.T) {
	root := copyRules(t)
	edit(t, root, "glossary/terms.yaml", func(string) string {
		return "- kr: 당 피해량\n  zh: 每次傷害量\n  status: approved\n- kr: 와일드헌트\n  zh: 狂獵\n  status: approved\n- kr: 턴 당 2회\n  zh: 每回合 2 次\n  status: approved\n"
	})
	r, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, kr := range []string{`3%당 <sprite name="AttackDmgUp">피해량 증가`, `당 <sprite name="X">피해량`, "당 <br>피해량"} {
		if hits := r.GlossaryHits(anyFile, kr); len(hits) != 0 {
			t.Errorf("GlossaryHits(%q) = %+v, want none: the tag stands between two words", kr, hits)
		}
		if terms := r.ValidateTerms(anyFile, kr); len(terms) != 0 {
			t.Errorf("ValidateTerms(%q) = %+v, want none", kr, terms)
		}
	}
	for _, kr := range []string{"<b>와일드</b>헌트", "(턴 당 <style=x>2</style>회)"} {
		if len(r.GlossaryHits(anyFile, kr)) != 1 {
			t.Errorf("GlossaryHits(%q) = %+v, want the one entry", kr, r.GlossaryHits(anyFile, kr))
		}
	}
}

// An entry with files stands for its KR text only in the files where it
// applies: files match and not_files does not. Where not_files takes it out,
// the entry without files is the one for the file.
func TestGlossaryHitsScopedEntryExcludedByNotFiles(t *testing.T) {
	r := rulesWithGlossary(t, `- kr: 신부
  zh: 牧師
  status: approved
  files: ["StoryData/**"]
  not_files: ["StoryData/S10*"]

- kr: 신부
  zh: 神父
  status: approved
`)
	for file, want := range map[string]string{"StoryData/S1001.json": "神父", "StoryData/S0901.json": "牧師", "Skills.json": "神父"} {
		if got := zhOf(r.GlossaryHits(file, "신부가 왔다")); !slices.Equal(got, []string{want}) {
			t.Errorf("GlossaryHits(%q) = %v, want [%s]", file, got, want)
		}
	}
}

// A files or not_files pattern that matches no file of the KR pack is a
// slip (a misspelled name, a pack file that was renamed): LintScopeWarnings
// names each one. It is a warning, not an error.
func TestLintScopeWarnings(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "glossary"), 0o755); err != nil {
		t.Fatal(err)
	}
	glossary := `- kr: 속도
  zh: 速度
  status: approved
  files: ["Skills_Ego*", "Skillz*", "Mirror/**"]
  not_files: ["Passives.json", "Passive_Typo*", "StoryData/**"]

- kr: 도시
  zh: 都市
  status: approved
`
	if err := os.WriteFile(filepath.Join(root, "glossary/terms.yaml"), []byte(glossary), 0o644); err != nil {
		t.Fatal(err)
	}
	files := []string{"Skills_Ego-1.json", "Passives.json", "StoryData/S1.json"}
	got := rules.LintScopeWarnings(root, files)
	var msgs []string
	for _, w := range got {
		if w.File != "glossary/terms.yaml" {
			t.Errorf("warning %+v should name glossary/terms.yaml", w)
		}
		msgs = append(msgs, w.Message)
	}
	want := []string{`"Skillz*"`, `"Mirror/**"`, `"Passive_Typo*"`}
	if len(msgs) != len(want) {
		t.Fatalf("warnings = %q, want one for each of %q", msgs, want)
	}
	for i, w := range want {
		if !strings.Contains(msgs[i], w) || !strings.Contains(msgs[i], `"속도"`) {
			t.Errorf("warning %d = %q, want it to name the term and %s", i, msgs[i], w)
		}
	}
	if !strings.Contains(msgs[0], "files") || !strings.Contains(msgs[2], "not_files") {
		t.Errorf("warnings should say which list the pattern is in: %q", msgs)
	}
	if got := rules.LintScopeWarnings(root, nil); len(got) != 0 {
		t.Errorf("without a file list there is nothing to compare with: %v", got)
	}
}

// also reaches validate through ToValidate, and Used counts zh and also.
func TestTermUsedAndToValidateCarryAlso(t *testing.T) {
	term := rules.Term{KR: "보스", ZH: "BOSS", Also: []string{"老大", "首領"}}
	if got := term.ToValidate().Also; !slices.Equal(got, term.Also) {
		t.Errorf("ToValidate().Also = %v, want %v", got, term.Also)
	}
	for zh, want := range map[string]bool{"BOSS": true, "<b>老</b>大": true, "首領": true, "頭目": false} {
		if got := term.Used(zh); got != want {
			t.Errorf("Used(%q) = %v, want %v", zh, got, want)
		}
	}
}

// also is a term field, and lint rejects an empty item, the zh again, and a
// rendering that forbidden also lists.
func TestLintGlossaryAlso(t *testing.T) {
	root := copyRules(t)
	edit(t, root, "glossary/terms.yaml", func(s string) string {
		return s + "\n- kr: 백화점\n  zh: 百貨公司\n  also: [\"\", 百貨公司, 百貨, 商場, \"<b>x</b>\"]\n  forbidden: [百貨, re:商.]\n  status: approved\n"
	})
	msg := lintMessages(rules.Lint(root))
	for _, want := range []string{"백화점", "also", "empty", "百貨公司", "百貨"} {
		if !strings.Contains(msg, want) {
			t.Errorf("lint output does not mention %q:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "unknown") && strings.Contains(msg, `"also"`) {
		t.Errorf("also is reported as an unknown key:\n%s", msg)
	}
}

// An also made only of tags is no text: Used would find it in every
// translation, so lint rejects it.
func TestLintGlossaryAlsoOfTagsOnly(t *testing.T) {
	root := copyRules(t)
	edit(t, root, "glossary/terms.yaml", func(s string) string {
		return s + "\n- kr: 백화점\n  zh: 百貨公司\n  also: [\"<b></b>\"]\n  status: approved\n"
	})
	msg := lintMessages(rules.Lint(root))
	for _, want := range []string{"백화점", "also", "only tags"} {
		if !strings.Contains(msg, want) {
			t.Errorf("lint output does not mention %q:\n%s", want, msg)
		}
	}
}

// Spans are the byte ranges of the hits of a term in a text as MatchText
// leaves it, by its match: only where its KR stands as a word, or at every
// occurrence for match: substring, overlapping ones too.
func TestTermSpans(t *testing.T) {
	text := "정사무소와 사무소, 사무소"
	for _, tc := range []struct {
		term rules.Term
		text string
		want [][2]int
	}{
		{rules.Term{KR: "사무소", Match: rules.MatchSubstring}, text, [][2]int{{3, 12}, {16, 25}, {27, 36}}},
		{rules.Term{KR: "사무소"}, text, [][2]int{{16, 25}, {27, 36}}},
		{rules.Term{KR: "사무소", Match: rules.MatchWord}, text, [][2]int{{16, 25}, {27, 36}}},
		{rules.Term{KR: "가가", Match: rules.MatchSubstring}, "가가가", [][2]int{{0, 6}, {3, 9}}},
		{rules.Term{KR: "꽃"}, text, nil},
		{rules.Term{KR: "", Match: rules.MatchSubstring}, text, nil},
	} {
		if got := tc.term.Spans(tc.text); !slices.Equal(got, tc.want) {
			t.Errorf("%+v.Spans(%q) = %v, want %v", tc.term, tc.text, got, tc.want)
		}
		if got := len(tc.term.Spans(tc.text)) > 0; got != tc.term.OccursIn(tc.text) {
			t.Errorf("%+v: Spans and OccursIn disagree on %q", tc.term, tc.text)
		}
	}
	// An empty KR occurs nowhere, whatever the match.
	for _, match := range []string{"", rules.MatchWord, rules.MatchSubstring} {
		if (rules.Term{Match: match}).OccursIn(text) {
			t.Errorf("Term{Match: %q} with no KR occurs in %q", match, text)
		}
	}
}

// Review Focus 1. A longer entry takes the place of a shorter one only where
// it covers it: 보스 inside 조직의 보스 does not count, a 보스 elsewhere in the
// same text does, and then both entries hit the unit.
func TestLongerEntryCoversOnlyItsOwnSpan(t *testing.T) {
	r := rulesWithGlossary(t, `- kr: 보스
  zh: BOSS
  status: approved
- kr: 조직의 보스
  zh: 組織的老大
  status: approved
`)
	for _, tc := range []struct {
		text string
		want []string
	}{
		{"조직의 보스가 보스 전투에 나섰다", []string{"BOSS", "組織的老大"}},
		{"보스 전투 전에 조직의 보스가 왔다", []string{"BOSS", "組織的老大"}},
		{"조직의 보스가 왔다", []string{"組織的老大"}},
		{"조직의 보스와 조직의 보스", []string{"組織的老大"}},
		{"<b>조직의</b> 보스가 왔다", []string{"組織的老大"}},
		{"보스가 왔다", []string{"BOSS"}},
	} {
		if got := zhOf(r.GlossaryHits(anyFile, tc.text)); !slices.Equal(got, tc.want) {
			t.Errorf("GlossaryHits(%q) = %v, want %v", tc.text, got, tc.want)
		}
	}

	boss := rules.Term{KR: "보스", ZH: "BOSS"}
	org := rules.Term{KR: "조직의 보스", ZH: "組織的老大"}
	if got := zhOf(rules.DropCovered("조직의 보스가 보스 전투에", []rules.Term{boss, org})); !slices.Equal(got, []string{"BOSS", "組織的老大"}) {
		t.Errorf("DropCovered with an uncovered 보스 = %v, want both", got)
	}
	if got := zhOf(rules.DropCovered("조직의 보스가 왔다", []rules.Term{org, boss})); !slices.Equal(got, []string{"組織的老大"}) {
		t.Errorf("DropCovered with 보스 covered = %v, want only the longer entry", got)
	}
	// A term that does not occur is not weighed: DropCovered keeps it.
	absent := rules.Term{KR: "꽃", ZH: "花"}
	if got := zhOf(rules.DropCovered("보스가 왔다", []rules.Term{boss, absent})); !slices.Equal(got, []string{"BOSS", "花"}) {
		t.Errorf("DropCovered with a term that does not occur = %v, want both", got)
	}
}

// Review Focus 2. Entries that only overlap cover each other nowhere: both
// hit, of the same length or not (a longer entry that overlaps a shorter one
// without holding all of it does not cover it), whichever side the longer
// one is on.
func TestPartialOverlapKeepsBoth(t *testing.T) {
	for _, tc := range []struct{ text, a, b string }{
		{"가나다라", "가나다", "나다라"},
		{"가나다라마", "가나다", "나다라마"},
		{"가나다라마", "가나다라", "다라마"},
	} {
		r := rulesWithGlossary(t, "- kr: "+tc.a+"\n  zh: "+tc.a+"\n  status: approved\n  match: substring\n"+
			"- kr: "+tc.b+"\n  zh: "+tc.b+"\n  status: approved\n  match: substring\n")
		if got := zhOf(r.GlossaryHits(anyFile, tc.text)); !slices.Equal(got, []string{tc.a, tc.b}) {
			t.Errorf("GlossaryHits(%q) with %s and %s = %v, want both", tc.text, tc.a, tc.b, got)
		}
		a := rules.Term{KR: tc.a, ZH: tc.a, Match: rules.MatchSubstring}
		b := rules.Term{KR: tc.b, ZH: tc.b, Match: rules.MatchSubstring}
		if got := zhOf(rules.DropCovered(tc.text, []rules.Term{b, a})); !slices.Equal(got, []string{tc.b, tc.a}) {
			t.Errorf("DropCovered(%q, %s, %s) = %v, want both", tc.text, tc.b, tc.a, got)
		}
	}
}

// Two entries with the same span do not cover each other (coverage is
// strict): both stay.
func TestSameSpanEntriesBothStay(t *testing.T) {
	r := rulesWithGlossary(t, `- kr: 보스
  zh: BOSS
  status: approved
  files: ["A*.json"]
- kr: 보스
  zh: 首領
  status: approved
  files: ["*B.json"]
- kr: 조직의 보스
  zh: 組織的老大
  status: approved
`)
	if got := zhOf(r.GlossaryHits("AB.json", "보스가 왔다")); !slices.Equal(got, []string{"BOSS", "首領"}) {
		t.Errorf("GlossaryHits = %v, want both entries of 보스", got)
	}
	if got := zhOf(r.GlossaryHits("AB.json", "조직의 보스가 왔다")); !slices.Equal(got, []string{"組織的老大"}) {
		t.Errorf("GlossaryHits = %v, want only the longer entry: it covers both", got)
	}
	word := rules.Term{KR: "보스", ZH: "BOSS"}
	sub := rules.Term{KR: "보스", ZH: "首領", Match: rules.MatchSubstring}
	if got := zhOf(rules.DropCovered("보스가 왔다", []rules.Term{word, sub})); !slices.Equal(got, []string{"BOSS", "首領"}) {
		t.Errorf("DropCovered with the same span = %v, want both", got)
	}
}

// The hits of a substring entry take part as any: one inside a longer word is
// covered by a longer entry, and covers a shorter one. A longer entry covers
// only where it hits by its own match.
func TestSubstringEntrySpansTakePart(t *testing.T) {
	r := rulesWithGlossary(t, `- kr: 사무소
  zh: 事務所
  status: approved
  match: substring
- kr: 정사무소
  zh: 正事務所
  status: approved
- kr: 보스
  zh: BOSS
  status: approved
- kr: 중간 보스
  zh: 中頭目
  status: approved
  match: substring
`)
	for _, tc := range []struct {
		text string
		want []string
	}{
		{"정사무소 대표", []string{"正事務所"}},
		{"정사무소와 사무소", []string{"事務所", "正事務所"}},
		// 정사무소 is no word here: it hits nowhere, so it covers nothing.
		{"새정사무소", []string{"事務所"}},
		// The hit of 중간 보스 inside 최중간 covers the word 보스.
		{"최중간 보스가 왔다", []string{"中頭目"}},
	} {
		if got := zhOf(r.GlossaryHits(anyFile, tc.text)); !slices.Equal(got, tc.want) {
			t.Errorf("GlossaryHits(%q) = %v, want %v", tc.text, got, tc.want)
		}
	}
}
