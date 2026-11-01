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
	hit := hitFor(r.GlossaryHits("이 패키지는 한정 판매입니다."), "패키지")
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
		for _, h := range r.GlossaryHits(text) {
			if !strings.Contains(text, h.KR) || h.Status != "approved" {
				t.Errorf("GlossaryHits(%q) returned %+v: not an approved term of the text", text, h)
			}
		}
	}
	if hits := r.GlossaryHits(""); len(hits) != 0 {
		t.Errorf("empty KR, but hits = %+v", hits)
	}

	// several hits come back in glossary order, not text order
	hits := r.GlossaryHits("시즌 보상: 도약 모듈")
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
	if hits := r.GlossaryHits("시즌 8 동기화 · 환상해석 전용 파편 (공용)"); len(hits) != 1 || hits[0].KR != "시즌" {
		t.Errorf("hits = %+v, want only the approved 시즌", hits)
	}
	if hits := r.GlossaryHits("전용 파편"); len(hits) != 0 {
		t.Errorf("a candidate term must not be returned, got %+v", hits)
	}
	// an approved term in the same text is still returned
	hits := r.GlossaryHits("자아 파편과 전용 파편")
	if len(hits) != 1 || hits[0].KR != "자아 파편" || hits[0].Status != "approved" {
		t.Errorf("hits = %+v, want only the approved 자아 파편", hits)
	}
	// a Korean text with none of the terms has no hit
	if hits := r.GlossaryHits("아무 용어도 없는 문장"); len(hits) != 0 {
		t.Errorf("no term in the text, but hits = %+v", hits)
	}

	// and in the real glossary, whatever candidates it holds at the moment
	live := loadRepo(t)
	for _, term := range live.Glossary {
		if term.Status == "candidate" && hitFor(live.GlossaryHits(term.KR), term.KR) != nil {
			t.Errorf("the candidate %s is among the hits of its own text", term.KR)
		}
	}
}

func TestToValidate(t *testing.T) {
	r := loadRepo(t)
	hit := hitFor(r.GlossaryHits("시즌"), "시즌")
	if hit == nil {
		t.Fatal("시즌 should be an approved term")
	}
	if !slices.Contains(hit.Forbidden, `re:第\d+季`) {
		t.Fatalf("시즌 forbids %v, want the pattern re:第\\d+季 among them", hit.Forbidden)
	}
	want := validate.Term{KR: "시즌", Forbidden: hit.Forbidden}
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
	const danteNote = "  - id: dante-note\n    match: [\"StoryTheaterDanteNote*\"]\n    guide: guides/families.md\n"
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
