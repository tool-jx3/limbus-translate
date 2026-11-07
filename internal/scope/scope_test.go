package scope

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/tool-jx3/limbus-translate/internal/itemid"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/suggest"
)

// The files of the fixtures, one of each group: their family by testRules and
// their directory group.
const (
	storyFile = "StoryData/S1.json"           // story, group:StoryData/
	voiceFile = "PersonalityVoiceDlg/V1.json" // voice, group:PersonalityVoiceDlg/
	skillFile = "Skills_A.json"               // skill, group:Skills
	otherFile = "Other.json"                  // other, group:Other
)

// testRules has four families: story, voice, skill and the catch-all other.
func testRules(terms ...rules.Term) *rules.Rules {
	return &rules.Rules{Glossary: terms, Routing: rules.Routing{
		Families: []rules.FamilyRule{
			{ID: "story", Nodes: []string{"StoryData/"}},
			{ID: "voice", Nodes: []string{"PersonalityVoiceDlg/"}},
			{ID: "skill", Nodes: []string{"Skills*"}},
			{ID: "other"},
		},
		Priority: []string{"story", "voice", "skill", "other"},
	}}
}

// approved is an approved entry kr -> zh that matches as a word.
func approved(kr, zh string) rules.Term {
	return rules.Term{KR: kr, ZH: zh, Status: rules.StatusApproved}
}

// fixture builds units: each call of add puts n units of one text in file,
// translated by Claude, with keys that never repeat.
type fixture struct{ units []Unit }

func (f *fixture) add(file string, n int, kr, zh string) *fixture {
	return f.addAs(file, n, kr, zh, store.OriginClaude, "")
}

func (f *fixture) addAs(file string, n int, kr, zh, origin, note string) *fixture {
	for range n {
		key := fmt.Sprintf("%s|/dataList/%d/content", file, len(f.units))
		f.units = append(f.units, Unit{Key: key, File: file, KR: kr, ZH: zh, Origin: origin, Note: note})
	}
	return f
}

// proposals returns the proposals of rep about the entry kr, by kind.
func proposals(rep Report, kr string) map[string]Proposal {
	out := map[string]Proposal{}
	for _, p := range rep.Proposals {
		if p.KR == kr {
			out[p.Kind] = p
		}
	}
	return out
}

func kinds(m map[string]Proposal) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	slices.Sort(ks)
	return ks
}

func entryOf(t *testing.T, rep Report, kr string) Entry {
	t.Helper()
	for _, e := range rep.Entries {
		if e.KR == kr {
			return e
		}
	}
	t.Fatalf("no entry %s in %+v", kr, rep.Entries)
	return Entry{}
}

// onlyProposal fails unless rep has exactly the proposal of kind about kr, and
// returns it.
func onlyProposal(t *testing.T, rep Report, kr, kind string) Proposal {
	t.Helper()
	ps := proposals(rep, kr)
	p, ok := ps[kind]
	if !ok || len(ps) != 1 {
		t.Fatalf("proposals of %s = %v, want only %s:\n%+v", kr, kinds(ps), kind, rep.Proposals)
	}
	if want := itemid.Scope(kr, p.ZH, kind); p.ID != want {
		t.Errorf("id = %s, want %s", p.ID, want)
	}
	return p
}

// 보스 is used everywhere but in the stories: exclude them.
func notFilesFixture() (*rules.Rules, []Unit) {
	f := &fixture{}
	f.add(skillFile, 4, "보스 등장", "BOSS 登場")
	f.add(storyFile, 3, "그 보스다", "那個老大")
	return testRules(approved("보스", "BOSS")), f.units
}

func TestNotFiles(t *testing.T) {
	r, us := notFilesFixture()
	rep := Analyze(Inputs{Rules: r, Units: us})
	p := onlyProposal(t, rep, "보스", KindNotFiles)
	// The family and the directory say the same; the family is named.
	if !slices.Equal(p.Patterns, []string{"family:story"}) || p.WarnRemoved != 3 || p.UsedLost != 0 || p.Flagged || p.Match != "" {
		t.Errorf("proposal = %+v, want not_files family:story removing 3 warnings, no check lost", p)
	}
	if len(p.Examples) != 3 {
		t.Fatalf("examples = %+v, want the 3 units of the stories", p.Examples)
	}
	for _, ex := range p.Examples {
		if ex.Group != "family:story" || ex.KR != "그 보스다" || ex.ZH != "那個老大" || !strings.HasPrefix(ex.Key, storyFile+"|") {
			t.Errorf("example = %+v, want a unit of the stories", ex)
		}
	}
}

// 공격 시작 is used in the voice lines only: limit it to them.
func TestFiles(t *testing.T) {
	f := &fixture{}
	f.add(voiceFile, 3, "공격 시작!", "進行攻擊！")
	f.add(storyFile, 3, "공격 시작", "開始攻擊")
	f.add(skillFile, 3, "공격 시작", "開始攻擊")
	f.add(otherFile, 3, "공격 시작", "開始攻擊")
	rep := Analyze(Inputs{Rules: testRules(approved("공격 시작", "進行攻擊")), Units: f.units})
	p := onlyProposal(t, rep, "공격 시작", KindFiles)
	if !slices.Equal(p.Patterns, []string{"family:voice"}) || p.WarnRemoved != 9 || p.UsedLost != 0 || p.Flagged {
		t.Errorf("proposal = %+v, want files family:voice removing 9 warnings", p)
	}
	if len(p.Examples) != 9 {
		t.Errorf("examples = %+v, want 3 of each group left out", p.Examples)
	}
}

// 사무소 inside 정사무소 uses the rendering: match it inside words too.
func TestSubstring(t *testing.T) {
	f := &fixture{}
	f.add(skillFile, 3, "정사무소", "正事務所")
	f.add(skillFile, 1, "사무소", "事務所")
	rep := Analyze(Inputs{Rules: testRules(approved("사무소", "事務所")), Units: f.units})
	p := onlyProposal(t, rep, "사무소", KindSubstring)
	if p.Match != rules.MatchSubstring || len(p.Patterns) != 0 || p.WarnRemoved != 0 || p.UsedLost != 0 || p.Flagged || len(p.Examples) != 3 {
		t.Errorf("proposal = %+v, want match substring, nothing removed, 3 examples", p)
	}
	e := entryOf(t, rep, "사무소")
	if e.Compound != (Count{N: 3, Used: 3}) || !reflect.DeepEqual(e.Groups, []Group{{Name: "family:skill", Kind: GroupFamily, N: 1, Used: 1}, {Name: "group:Skills", Kind: GroupNode, N: 1, Used: 1}}) {
		t.Errorf("entry = %+v, want 3 of 3 compound units and the one word in the groups", e)
	}
}

// 다리 is substring, but 다리미 is no bridge: match it as a word.
func TestWord(t *testing.T) {
	f := &fixture{}
	f.add(skillFile, 3, "다리미", "熨斗")
	f.add(skillFile, 3, "다리 위", "橋上")
	term := approved("다리", "橋")
	term.Match = rules.MatchSubstring
	rep := Analyze(Inputs{Rules: testRules(term), Units: f.units})
	p := onlyProposal(t, rep, "다리", KindWord)
	if p.Match != rules.MatchWord || p.WarnRemoved != 3 || p.UsedLost != 0 || p.Flagged || len(p.Examples) != 3 || p.Examples[0].KR != "다리미" {
		t.Errorf("proposal = %+v, want match word removing the 3 warnings of 다리미", p)
	}
	e := entryOf(t, rep, "다리")
	if e.Match != rules.MatchSubstring || e.Compound != (Count{N: 3, Used: 0}) || e.Groups[0] != (Group{Name: "family:skill", Kind: GroupFamily, N: 3, Used: 3}) {
		t.Errorf("entry = %+v, want the 3 compound units apart from the groups", e)
	}
}

// 복도 is half 走廊 in the stories: a person looks.
func TestMixed(t *testing.T) {
	f := &fixture{}
	f.add(storyFile, 3, "복도", "走廊")
	f.add(storyFile, 3, "복도", "通道")
	f.add(skillFile, 3, "복도", "走廊")
	rep := Analyze(Inputs{Rules: testRules(approved("복도", "走廊")), Units: f.units})
	p := onlyProposal(t, rep, "복도", KindMixed)
	if len(p.Patterns) != 0 || p.Match != "" || p.WarnRemoved != 0 || p.WarnAdded != 0 || p.UsedLost != 0 || p.UsedGained != 0 || p.Flagged {
		t.Errorf("proposal = %+v, want no change and no effect", p)
	}
	if !slices.Equal(p.MixedGroups, []string{"family:story", "group:StoryData/"}) || !strings.Contains(p.Title(), "只部分用了譯名") {
		t.Errorf("mixed groups %v, title %q: want the stories, both ways", p.MixedGroups, p.Title())
	}
	if len(p.Examples) != 3 || p.Examples[0].ZH != "通道" || p.Examples[0].Group != "family:story" {
		t.Errorf("examples = %+v, want the 3 units of the stories that do not use it, once", p.Examples)
	}
}

// A group that uses the rendering in part, but lacks it in fewer than MinN
// units, says nothing: it is no mixed group, and it does not keep a change
// that other groups ask for from being proposed.
func TestSmallMixedSaysNothing(t *testing.T) {
	f := &fixture{}
	f.add(storyFile, 2, "복도", "走廊").add(storyFile, 2, "복도", "通道")
	f.add(skillFile, 3, "복도", "走廊")
	if rep := Analyze(Inputs{Rules: testRules(approved("복도", "走廊")), Units: f.units}); len(rep.Proposals) != 0 {
		t.Errorf("2 of 4 without it: proposals %+v, want none", rep.Proposals)
	}
	r, us := notFilesFixture()
	g := &fixture{units: us}
	g.add(voiceFile, 2, "보스", "BOSS").add(voiceFile, 2, "보스", "老闆")
	p := onlyProposal(t, Analyze(Inputs{Rules: r, Units: g.units}), "보스", KindNotFiles)
	if !slices.Equal(p.Patterns, []string{"family:story"}) || len(p.MixedGroups) != 0 {
		t.Errorf("proposal = %+v, want not_files family:story and no mixed group", p)
	}
}

// No group uses the rendering: a person looks, and the title says so.
func TestNoGroupUsesIt(t *testing.T) {
	f := &fixture{}
	f.add(storyFile, 3, "복도", "通道").add(skillFile, 3, "복도", "通道")
	p := onlyProposal(t, Analyze(Inputs{Rules: testRules(approved("복도", "走廊")), Units: f.units}), "복도", KindMixed)
	if len(p.MixedGroups) != 0 || !strings.Contains(p.Title(), "各群組都很少用這個譯名") || strings.Contains(p.Title(), "只部分") {
		t.Errorf("proposal = %+v, title %q: want no mixed group and the title of a rendering used nowhere", p, p.Title())
	}
	if len(p.Examples) != 6 {
		t.Errorf("examples = %+v, want 3 of each group, no unit twice", p.Examples)
	}
}

// 대상 is used in each group, but the notes of three units say why not there.
func TestExplained(t *testing.T) {
	f := &fixture{}
	for _, file := range []string{storyFile, skillFile, otherFile} {
		f.add(file, 4, "대상", "目標")
		f.addAs(file, 1, "대상에게", "給對象", store.OriginClaude, "這裡的 대상 指人，譯為對象")
	}
	f.add(voiceFile, 4, "대상", "目標")
	f.addAs(voiceFile, 1, "대상", "對象", store.OriginClaude, "指人")
	rep := Analyze(Inputs{Rules: testRules(approved("대상", "目標")), Units: f.units})
	p := onlyProposal(t, rep, "대상", KindExplained)
	if len(p.Patterns) != 0 || p.WarnRemoved != 0 || p.UsedLost != 0 || p.Flagged || len(p.Examples) != 3 {
		t.Errorf("proposal = %+v, want no change, 3 examples", p)
	}
	for _, ex := range p.Examples {
		if ex.ZH != "給對象" {
			t.Errorf("example = %+v, want a unit whose note explains", ex)
		}
	}
	if e := entryOf(t, rep, "대상"); e.Explained != 3 || e.Unexplained != 1 {
		t.Errorf("explained %d, unexplained %d, want 3 and 1", e.Explained, e.Unexplained)
	}
}

// Excluding the stories loses the check of 3 of the 13 units that use 속도:
// flagged.
func TestFlagged(t *testing.T) {
	f := &fixture{}
	f.add(storyFile, 3, "속도", "速度值")
	f.add(storyFile, 7, "속도", "速度")
	f.add(skillFile, 10, "속도", "速度值")
	rep := Analyze(Inputs{Rules: testRules(approved("속도", "速度值")), Units: f.units})
	p := onlyProposal(t, rep, "속도", KindNotFiles)
	if p.WarnRemoved != 7 || p.UsedLost != 3 || !p.Flagged {
		t.Errorf("proposal = %+v, want 7 removed, 3 lost, flagged", p)
	}
}

// A declined proposal is still listed, with its confirmation.
func TestDeclined(t *testing.T) {
	r, us := notFilesFixture()
	id := itemid.Scope("보스", "BOSS", KindNotFiles)
	c := &suggest.Confirmed{Reason: "BOSS 在劇情也可以", Date: "2026-11-06"}
	rep := Analyze(Inputs{Rules: r, Units: us, Declined: map[string]*suggest.Confirmed{id: c, "scope/00000000/00000000/word": c}})
	p := onlyProposal(t, rep, "보스", KindNotFiles)
	if p.Declined == nil || *p.Declined != *c {
		t.Errorf("declined = %+v, want %+v", p.Declined, c)
	}
	if rep := Analyze(Inputs{Rules: r, Units: us}); rep.Proposals[0].Declined != nil {
		t.Errorf("not declined: %+v", rep.Proposals[0].Declined)
	}
}

// Only the translations of their own count: Claude's, a person's and
// Paratranz's. Copies and fallbacks would make the stories use BOSS.
func TestOnlyTranslationsCount(t *testing.T) {
	r, us := notFilesFixture()
	us[0].Origin, us[1].Origin = store.OriginHuman, store.OriginParatranz
	f := &fixture{units: us}
	f.addAs(storyFile, 5, "보스", "BOSS", store.OriginCopy, "")
	f.addAs(storyFile, 5, "보스", "BOSS", store.OriginFallback, "")
	f.addAs(storyFile, 5, "보스", "BOSS", "", "")
	rep := Analyze(Inputs{Rules: r, Units: f.units})
	onlyProposal(t, rep, "보스", KindNotFiles)
	want := []Group{
		{Name: "family:story", Kind: GroupFamily, N: 3, Used: 0, Unknown: 3},
		{Name: "family:skill", Kind: GroupFamily, N: 4, Used: 4},
		{Name: "group:Skills", Kind: GroupNode, N: 4, Used: 4},
		{Name: "group:StoryData/", Kind: GroupNode, N: 3, Used: 0, Unknown: 3},
	}
	if e := entryOf(t, rep, "보스"); !reflect.DeepEqual(e.Groups, want) {
		t.Errorf("groups = %+v, want %+v", e.Groups, want)
	}
}

// What the entry is set to already is not proposed: a group not_files takes
// out has no hits, so its pattern does not come back; an entry that matches as
// a substring is not proposed substring.
func TestOnlyDifferences(t *testing.T) {
	r, us := notFilesFixture()
	r.Glossary[0].NotFiles = []string{"family:story"}
	if rep := Analyze(Inputs{Rules: r, Units: us}); len(rep.Proposals) != 0 {
		t.Errorf("not_files family:story already: proposals %+v, want none", rep.Proposals)
	}
	f := &fixture{}
	f.add(skillFile, 3, "정사무소", "正事務所")
	term := approved("사무소", "事務所")
	term.Match = rules.MatchSubstring
	if rep := Analyze(Inputs{Rules: testRules(term), Units: f.units}); len(rep.Proposals) != 0 {
		t.Errorf("substring already: proposals %+v, want none", rep.Proposals)
	}
}

// The longer entry stands for its text: 보스 in 조직의 보스 is neither a hit
// of 보스 nor a word in a word, so it moves none of the numbers of 보스.
func TestCoveredIsNoHitNorCompound(t *testing.T) {
	f := &fixture{}
	f.add(storyFile, 3, "조직의 보스", "組織的老大")
	f.add(skillFile, 3, "보스", "BOSS")
	rep := Analyze(Inputs{Rules: testRules(approved("보스", "BOSS"), approved("조직의 보스", "組織的老大")), Units: f.units})
	e := entryOf(t, rep, "보스")
	if e.Compound != (Count{}) || len(e.Groups) != 2 || e.Groups[0] != (Group{Name: "family:skill", Kind: GroupFamily, N: 3, Used: 3}) {
		t.Errorf("보스 = %+v, want only the 3 units of the skills", e)
	}
	if len(rep.Proposals) != 0 {
		t.Errorf("proposals = %+v, want none", rep.Proposals)
	}
}

// combined is every fixture in one: each entry with its own text.
func combined() (*rules.Rules, []Unit) {
	f := &fixture{}
	f.add(skillFile, 4, "보스 등장", "BOSS 登場").add(storyFile, 3, "그 보스다", "那個老大")
	f.add(voiceFile, 3, "공격 시작!", "進行攻擊！").add(storyFile, 3, "공격 시작", "開始攻擊").add(skillFile, 3, "공격 시작", "開始攻擊").add(otherFile, 3, "공격 시작", "開始攻擊")
	f.add(skillFile, 3, "정사무소", "正事務所").add(skillFile, 1, "사무소", "事務所")
	f.add(skillFile, 3, "다리미", "熨斗").add(skillFile, 3, "다리 위", "橋上")
	f.add(storyFile, 3, "복도", "走廊").add(storyFile, 3, "복도", "通道").add(skillFile, 3, "복도", "走廊")
	for _, file := range []string{storyFile, skillFile, otherFile} {
		f.add(file, 4, "대상", "目標").addAs(file, 1, "대상에게", "給對象", store.OriginClaude, "這裡的 대상 指人")
	}
	f.add(storyFile, 3, "속도", "速度值").add(storyFile, 7, "속도", "速度").add(skillFile, 10, "속도", "速度值")
	f.add(storyFile, 3, "조직의 보스", "組織的老大")
	bridge := approved("다리", "橋")
	bridge.Match = rules.MatchSubstring
	return testRules(approved("보스", "BOSS"), approved("공격 시작", "進行攻擊"), approved("사무소", "事務所"), bridge,
		approved("복도", "走廊"), approved("대상", "目標"), approved("속도", "速度值"), approved("조직의 보스", "組織的老大"),
		rules.Term{KR: "후보", ZH: "候選", Status: rules.StatusCandidate}), f.units
}

func TestCombinedHasEveryKind(t *testing.T) {
	r, us := combined()
	rep := Analyze(Inputs{Rules: r, Units: us})
	var got []string
	for _, p := range rep.Proposals {
		got = append(got, p.Kind+" "+p.KR)
	}
	// By kind, then in the order of the glossary.
	want := []string{
		"not-files 보스", "not-files 속도", "files 공격 시작", "substring 사무소", "word 다리", "mixed 복도", "explained 대상",
	}
	if !slices.Equal(got, want) {
		t.Errorf("proposals =\n%v\nwant\n%v", got, want)
	}
	// Every approved entry, in glossary order, and no candidate.
	var krs []string
	for _, e := range rep.Entries {
		krs = append(krs, e.KR)
	}
	if want := []string{"보스", "공격 시작", "사무소", "다리", "복도", "대상", "속도", "조직의 보스"}; !slices.Equal(krs, want) {
		t.Errorf("entries = %v, want %v", krs, want)
	}
	if rep.Thresholds != (Thresholds{MinN: 3, Used: 0.8, NotUsed: 0.3, Explained: 3, LostUnits: 20, LostShare: 0.1, HardRate: 0.95, HardMinN: 20}) {
		t.Errorf("thresholds = %+v", rep.Thresholds)
	}
}

// Review focus: a proposed pattern names a family of the routing or one of the
// fixed directories, and matches a file of the input. No glob of a family's
// match list, and no pattern that matches nothing.
func TestProposalPatternsMatchFiles(t *testing.T) {
	r, us := combined()
	// A family whose match list is a glob (as story-main's is), and one that
	// takes a directory and a file outside it.
	r.Routing.Families = append([]rules.FamilyRule{{ID: "lyrics", Nodes: []string{"BgmLyrics/"}}, {ID: "rpg", Nodes: []string{"RPGSystem/", "Rpg*"}}}, r.Routing.Families...)
	f := &fixture{units: us}
	// 보스 is used in two families and not in two: not_files, the glob family too.
	f.add("BgmLyrics/L1.json", 3, "보스", "老大").add(voiceFile, 3, "보스", "BOSS")
	// rpg is half 速度值, so the families cannot say it; the directories can.
	f.add("RPGSystem/R1.json", 3, "속도", "速度").add("RpgTop.json", 3, "속도", "速度值")
	f.add("EGOVoiceDig/E1.json", 3, "공격 시작", "開始攻擊").add("BattleAnnouncerDlg/A1.json", 3, "공격 시작", "開始攻擊")
	rep := Analyze(Inputs{Rules: r, Units: f.units})

	files, tops := map[string]bool{}, map[string]bool{}
	for _, u := range f.units {
		files[u.File] = true
		tops[rules.FileNode(u.File).Top] = true
	}
	families := map[string]bool{}
	for _, fam := range r.Routing.Families {
		families[fam.ID] = true
	}
	seen := 0
	for _, p := range rep.Proposals {
		if (p.Kind == KindNotFiles || p.Kind == KindFiles) != (len(p.Patterns) > 0) {
			t.Errorf("%s: patterns %v", p.ID, p.Patterns)
		}
		for _, pat := range p.Patterns {
			seen++
			id, isFamily := strings.CutPrefix(pat, rules.FamilyPrefix)
			top, isGroup := strings.CutPrefix(pat, rules.GroupPrefix)
			if isFamily && !families[id] || isGroup && !tops[top] || !isFamily && !isGroup {
				t.Errorf("%s: pattern %q is neither family:<id> of the routing nor group:<top node> of an input file", p.ID, pat)
			}
			matched := false
			for file := range files {
				matched = matched || r.Applies(rules.Term{Files: []string{pat}}, file)
			}
			if !matched {
				t.Errorf("%s: pattern %q matches no input file", p.ID, pat)
			}
		}
	}
	if seen < 4 {
		t.Fatalf("only %d patterns proposed: %+v", seen, rep.Proposals)
	}
	want := map[string][]string{
		"보스":    {"family:lyrics", "family:story"},
		"속도":    {"group:RPGSystem/", "group:StoryData/"},
		"공격 시작": {"family:voice"},
	}
	for kr, pats := range want {
		ps := proposals(rep, kr)
		if p := ps[KindNotFiles]; kr != "공격 시작" && !slices.Equal(p.Patterns, pats) {
			t.Errorf("%s: not-files %v, want %v", kr, p.Patterns, pats)
		}
		if p := ps[KindFiles]; kr == "공격 시작" && !slices.Equal(p.Patterns, pats) {
			t.Errorf("%s: files %v, want %v", kr, p.Patterns, pats)
		}
	}
}

// The hits are those of GlossaryHits: each entry's units in its groups (and,
// for a substring entry, its compound units) are the units GlossaryHits
// gives it.
func TestHitsAreGlossaryHits(t *testing.T) {
	r, us := combined()
	rep := Analyze(Inputs{Rules: r, Units: us})
	for _, e := range rep.Entries {
		want := 0
		for _, u := range us {
			if slices.ContainsFunc(r.GlossaryHits(u.File, u.KR), func(t rules.Term) bool { return t.KR == e.KR && t.ZH == e.ZH }) {
				want++
			}
		}
		got := 0
		for _, g := range e.Groups {
			if g.Kind == GroupFamily {
				got += g.N
			}
		}
		if e.Match == rules.MatchSubstring {
			got += e.Compound.N
		}
		if got != want {
			t.Errorf("%s: %d units in the groups, GlossaryHits gives it %d", e.KR, got, want)
		}
	}
}

// An entry lists its settings and the counts, never a null list.
func TestEntryFields(t *testing.T) {
	r, us := notFilesFixture()
	r.Glossary[0].Notes = "指組織頭頭時用老大"
	r.Glossary[0].Also = []string{"老大"}
	r.Glossary[0].JP, r.Glossary[0].EN = "ボス", "Boss"
	r.Glossary[0].Meaning, r.Glossary[0].Sources = "組織的首領", []string{"game:Other.json#/dataList/0/content"}
	rep := Analyze(Inputs{Rules: r, Units: us})
	e := entryOf(t, rep, "보스")
	if e.ID != itemid.GlossaryEntry("보스", "BOSS") || e.ZH != "BOSS" || !slices.Equal(e.Also, []string{"老大"}) || e.Files == nil || e.NotFiles == nil ||
		e.Match != rules.MatchWord || e.Notes != "指組織頭頭時用老大" {
		t.Errorf("entry = %+v", e)
	}
	if e.JP != "ボス" || e.EN != "Boss" || e.Meaning != "組織的首領" || !slices.Equal(e.Sources, []string{"game:Other.json#/dataList/0/content"}) {
		t.Errorf("jp, en, meaning, sources = %q %q %q %q", e.JP, e.EN, e.Meaning, e.Sources)
	}
	// With also 老大 the stories use it: nothing to propose.
	if len(rep.Proposals) != 0 {
		t.Errorf("proposals = %+v, want none once also holds 老大", rep.Proposals)
	}
	empty := Analyze(Inputs{Rules: testRules(approved("없음", "無"))})
	if e := entryOf(t, empty, "없음"); e.Groups == nil || e.Also == nil || len(empty.Proposals) != 0 || empty.Proposals == nil {
		t.Errorf("entry without hits = %+v, proposals %v", e, empty.Proposals)
	}
	if rep := Analyze(Inputs{}); rep.Entries == nil || rep.Proposals == nil || rep.Thresholds.MinN != 3 {
		t.Errorf("no rules: %+v", rep)
	}
}

// An example is a short piece of each text, around the entry in the KR.
func TestExamplesAreShort(t *testing.T) {
	long := strings.Repeat("가나다라 ", 15) + "보스가 나타났다 " + strings.Repeat("마바사아 ", 15)
	zh := strings.Repeat("甲乙丙丁", 15) + "老大出現了" + strings.Repeat("戊己庚辛", 15)
	f := &fixture{}
	f.add(skillFile, 4, "보스", "BOSS")
	f.add(storyFile, 5, long, zh)
	rep := Analyze(Inputs{Rules: testRules(approved("보스", "BOSS")), Units: f.units})
	p := onlyProposal(t, rep, "보스", KindNotFiles)
	if len(p.Examples) != maxExamples {
		t.Fatalf("examples = %d, want %d", len(p.Examples), maxExamples)
	}
	ex := p.Examples[0]
	if n := utf8.RuneCountInString(ex.KR); n > maxSnippet || !strings.Contains(ex.KR, "보스가 나타났다") || !strings.HasPrefix(ex.KR, "…") || !strings.HasSuffix(ex.KR, "…") {
		t.Errorf("KR = %q (%d runes), want at most %d around 보스", ex.KR, n, maxSnippet)
	}
	if n := utf8.RuneCountInString(ex.ZH); n > maxSnippet || !strings.Contains(ex.ZH, "老大") {
		t.Errorf("ZH = %q (%d runes), want at most %d about where the KR has it", ex.ZH, n, maxSnippet)
	}
}

// A proposal says in one line what it changes and what that does.
func TestProposalTitle(t *testing.T) {
	for _, tc := range []struct {
		p    Proposal
		want []string
	}{
		{Proposal{Kind: KindNotFiles, KR: "보스", ZH: "BOSS", Patterns: []string{"family:story", "group:StoryData/"}, WarnRemoved: 3, UsedLost: 1, UsedGained: 2},
			[]string{"보스 → BOSS", "not_files", "family:story、group:StoryData/", "移除 3 個警告", "新增 0 個", "1 個用了譯名的單位不再檢查", "2 個開始檢查"}},
		{Proposal{Kind: KindFiles, KR: "섭취하라", ZH: "吸食殆盡", Patterns: []string{"group:StoryData/"}, MixedGroups: []string{"family:identity"}, WarnRemoved: 9, UsedLost: 25, Flagged: true},
			[]string{"files", "group:StoryData/", "失去檢查偏多", "family:identity 只部分用了譯名"}},
		{Proposal{Kind: KindSubstring, KR: "대상", ZH: "目標", Match: rules.MatchSubstring, WarnAdded: 1, UsedGained: 4}, []string{"match: substring", "新增 1 個", "4 個開始檢查"}},
		{Proposal{Kind: KindWord, KR: "다리", ZH: "橋", Match: rules.MatchWord, WarnRemoved: 3}, []string{"match: word", "移除 3 個警告"}},
		{Proposal{Kind: KindMixed, KR: "복도", ZH: "走廊", MixedGroups: []string{"family:story"}}, []string{"family:story", "只部分用了譯名", "需要人看"}},
		{Proposal{Kind: KindMixed, KR: "복도", ZH: "走廊"}, []string{"各群組都很少用這個譯名", "需要人看"}},
		{Proposal{Kind: KindExplained, KR: "대상", ZH: "目標"}, []string{"note", "also"}},
	} {
		got := tc.p.Title()
		for _, w := range tc.want {
			if !strings.Contains(got, w) {
				t.Errorf("Title of %s = %q, want %q in it", tc.p.Kind, got, w)
			}
		}
	}
}

// Review focus of the fix round: the provisional terms in force cover glossary
// entries as on the review page (batch.TermsIn). 보스 inside the provisional
// 조직의 보스 is no hit of 보스, nor a compound unit.
func TestProvisionalTermsCover(t *testing.T) {
	f := &fixture{}
	f.add(skillFile, 4, "보스 등장", "BOSS 登場")
	f.add(storyFile, 3, "조직의 보스다", "組織的老大")
	f.add(otherFile, 3, "최종보스와", "與最終BOSS")
	r := testRules(approved("보스", "BOSS"))
	provisional := []rules.Term{
		{KR: "조직의 보스", ZH: "組織的老大", Status: rules.StatusApproved, Match: rules.MatchWord, Provisional: true},
		{KR: "최종보스", ZH: "最終BOSS", Status: rules.StatusApproved, Match: rules.MatchWord, Provisional: true},
	}
	// Without them, the stories lack BOSS and 최종보스 holds it in a word.
	without := Analyze(Inputs{Rules: r, Units: f.units})
	if ps := proposals(without, "보스"); len(ps) != 2 || ps[KindNotFiles].WarnRemoved != 3 || ps[KindSubstring].ID == "" {
		t.Fatalf("without provisional terms: %v, want not-files removing 3 and substring", kinds(ps))
	}
	rep := Analyze(Inputs{Rules: r, Units: f.units, Provisional: provisional})
	e := entryOf(t, rep, "보스")
	want := []Group{{Name: "family:skill", Kind: GroupFamily, N: 4, Used: 4}, {Name: "group:Skills", Kind: GroupNode, N: 4, Used: 4}}
	if !reflect.DeepEqual(e.Groups, want) || e.Compound != (Count{}) || e.Unexplained != 0 || len(rep.Proposals) != 0 {
		t.Errorf("보스 = %+v, proposals %+v: want the 4 units of the skills alone, and no proposal", e, rep.Proposals)
	}
}

// The effect of a proposal is the review page before and after it, over every
// entry: a substring entry warns where the word was no hit (시대상 is no 대상).
func TestSubstringEffectAddsWarnings(t *testing.T) {
	f := &fixture{}
	f.add(skillFile, 4, "공격대상", "攻擊目標")
	f.add(storyFile, 1, "시대상이", "年代感")
	p := onlyProposal(t, Analyze(Inputs{Rules: testRules(approved("대상", "目標")), Units: f.units}), "대상", KindSubstring)
	if p.WarnRemoved != 0 || p.WarnAdded != 1 || p.UsedLost != 0 || p.UsedGained != 4 || p.Flagged {
		t.Errorf("proposal = %+v, want 1 warning added and 4 units checked", p)
	}
	if p.Examples[0].KR != "시대상이" {
		t.Errorf("examples = %+v, want the warning it adds first", p.Examples)
	}
}

// Matching 공격 시작 as a word no longer covers 시작 in 선공격 시작: 시작 is
// checked there, and it is used.
func TestWordEffectUncoversAShorterEntry(t *testing.T) {
	f := &fixture{}
	f.add(skillFile, 3, "선공격 시작", "先攻開始")
	long := approved("공격 시작", "進行攻擊")
	long.Match = rules.MatchSubstring
	p := onlyProposal(t, Analyze(Inputs{Rules: testRules(long, approved("시작", "開始")), Units: f.units}), "공격 시작", KindWord)
	if p.WarnRemoved != 3 || p.WarnAdded != 0 || p.UsedLost != 0 || p.UsedGained != 3 {
		t.Errorf("proposal = %+v, want 3 warnings removed and the 3 units checked for 시작", p)
	}
}

// Taking the stories out of an entry with files hands 신부 there to the entry
// without files, whose rendering the stories use.
func TestNotFilesEffectHandsTheKRToTheGeneralEntry(t *testing.T) {
	f := &fixture{}
	f.add(storyFile, 3, "신부님", "神父大人")
	f.add(voiceFile, 3, "신부", "牧師")
	scoped := approved("신부", "牧師")
	scoped.Files = []string{"family:story", "family:voice"}
	rep := Analyze(Inputs{Rules: testRules(scoped, approved("신부", "神父")), Units: f.units})
	var p Proposal
	for _, q := range rep.Proposals {
		if q.ZH == "牧師" {
			p = q
		}
	}
	if p.Kind != KindNotFiles || !slices.Equal(p.Patterns, []string{"family:story"}) || p.WarnRemoved != 3 || p.UsedGained != 3 || p.UsedLost != 0 || p.WarnAdded != 0 {
		t.Errorf("proposal = %+v, want not_files family:story removing 3 warnings, and 神父 checked in those 3 units", p)
	}
}

// A clean split by one kind of group is proposed although the other kind has
// a mixed group, which the proposal names: rpg is half 速度值, the directories
// tell its two halves apart.
func TestConcreteNamesTheMixedGroups(t *testing.T) {
	r := testRules(approved("속도", "速度值"))
	r.Routing.Families = append([]rules.FamilyRule{{ID: "rpg", Nodes: []string{"RPGSystem/", "Rpg*"}}}, r.Routing.Families...)
	f := &fixture{}
	f.add(skillFile, 10, "속도", "速度值")
	f.add("RPGSystem/R1.json", 3, "속도", "速度").add("RpgTop.json", 3, "속도", "速度值")
	p := onlyProposal(t, Analyze(Inputs{Rules: r, Units: f.units}), "속도", KindNotFiles)
	if !slices.Equal(p.Patterns, []string{"group:RPGSystem/"}) || !slices.Equal(p.MixedGroups, []string{"family:rpg"}) || p.WarnRemoved != 3 {
		t.Errorf("proposal = %+v, want not_files group:RPGSystem/ naming family:rpg as mixed", p)
	}
	if !strings.Contains(p.Title(), "family:rpg") {
		t.Errorf("title %q, want the mixed group in it", p.Title())
	}
}

// Of two changes, the unflagged one first, then the one with fewer patterns,
// then the one that loses fewer checks. The family needs one pattern; with the
// 2 units of Narr.json that use 速度值 it loses them and is flagged, and the
// directories, two patterns losing none, are proposed. Without them both lose
// nothing, and the one family is proposed.
func TestFewerPatternsFirst(t *testing.T) {
	r := testRules(approved("속도", "速度值"))
	r.Routing.Families = append([]rules.FamilyRule{{ID: "narrative", Nodes: []string{"StoryData/", "PersonalityVoiceDlg/", "Narr*"}}}, r.Routing.Families...)
	f := &fixture{}
	f.add(skillFile, 10, "속도", "速度值")
	f.add(storyFile, 3, "속도", "速度").add(voiceFile, 3, "속도", "速度")
	p := onlyProposal(t, Analyze(Inputs{Rules: r, Units: f.units}), "속도", KindNotFiles)
	if !slices.Equal(p.Patterns, []string{"family:narrative"}) || p.UsedLost != 0 || p.Flagged {
		t.Errorf("both unflagged: proposal = %+v, want the one family over the two directories", p)
	}
	f.add("Narr.json", 2, "속도", "速度值")
	p = onlyProposal(t, Analyze(Inputs{Rules: r, Units: f.units}), "속도", KindNotFiles)
	if !slices.Equal(p.Patterns, []string{"group:PersonalityVoiceDlg/", "group:StoryData/"}) || p.UsedLost != 0 || p.Flagged {
		t.Errorf("the family flagged: proposal = %+v, want the two directories, which lose nothing", p)
	}
}

// Inside one partition, files is not chosen over not_files for having fewer
// patterns before the effects are known. 속도 is used in the voice lines and
// in 2 lines of Narr.json, which no group of three units covers: files
// [family:voice] removes the same 6 warnings as not_files [skill, story], but
// loses those 2 checks and is flagged, so the unflagged not_files is proposed.
func TestUnflaggedNotFilesBeatsFlaggedFiles(t *testing.T) {
	r := testRules(approved("속도", "速度值"))
	r.Routing.Families = append([]rules.FamilyRule{{ID: "narr", Nodes: []string{"Narr*"}}}, r.Routing.Families...)
	f := &fixture{}
	f.add(voiceFile, 3, "속도", "速度值")
	f.add(storyFile, 3, "속도", "速度").add(skillFile, 3, "속도", "速度")
	f.add("Narr.json", 2, "속도", "速度值")
	p := onlyProposal(t, Analyze(Inputs{Rules: r, Units: f.units}), "속도", KindNotFiles)
	if len(p.Patterns) != 2 || !slices.Contains(p.Patterns, "family:story") || !slices.Contains(p.Patterns, "family:skill") ||
		p.WarnRemoved != 6 || p.UsedLost != 0 || p.Flagged {
		t.Errorf("proposal = %+v, want the unflagged not_files [family:skill family:story] removing 6", p)
	}
}

// An entry carries up to maxExamples units of each group that does not use its
// rendering, and none of a group that does.
func TestEntryExamplesOfNonUsingGroups(t *testing.T) {
	f := &fixture{}
	f.add(skillFile, 4, "보스 등장", "BOSS 登場")
	f.add(storyFile, 5, "그 보스다", "那個老大")
	rep := Analyze(Inputs{Rules: testRules(approved("보스", "BOSS")), Units: f.units})
	e := entryOf(t, rep, "보스")
	perGroup := map[string]int{}
	for _, ex := range e.Examples {
		perGroup[ex.Group]++
		if ex.Key == "" || ex.KR != "그 보스다" || ex.ZH != "那個老大" || !strings.HasPrefix(ex.Key, storyFile+"|") {
			t.Errorf("example = %+v, want a unit of the stories with its KR and ZH", ex)
		}
	}
	// The family and the directory of the stories each get their 3 of the 5.
	want := map[string]int{"family:story": maxExamples, "group:StoryData/": maxExamples}
	if !reflect.DeepEqual(perGroup, want) {
		t.Errorf("examples per group = %v, want %v (none for the groups that use it)", perGroup, want)
	}
	if empty := entryOf(t, Analyze(Inputs{Rules: testRules(approved("없음", "無"))}), "없음"); empty.Examples == nil || len(empty.Examples) != 0 {
		t.Errorf("entry without hits has examples %v, want an empty list", empty.Examples)
	}
}

// A file is grouped by its top node: the files of StoryData/ in
// group:StoryData/, Skills_Enemy-x.json in group:Skills.
func TestGroupsAreTopNodes(t *testing.T) {
	f := &fixture{}
	f.add("StoryData/S1.json", 1, "보스", "BOSS").add("StoryData/S2.json", 1, "보스", "BOSS")
	f.add("Skills_Enemy-x.json", 1, "보스", "BOSS")
	rep := Analyze(Inputs{Rules: testRules(approved("보스", "BOSS")), Units: f.units})
	var got []string
	for _, g := range entryOf(t, rep, "보스").Groups {
		if g.Kind == GroupNode {
			got = append(got, fmt.Sprintf("%s=%d", g.Name, g.N))
		}
	}
	if want := []string{"group:Skills=1", "group:StoryData/=2"}; !slices.Equal(got, want) {
		t.Errorf("node groups = %v, want %v", got, want)
	}
}

// rateEntry is the entry of 보스 with n hits, used of them using BOSS.
func rateEntry(t *testing.T, term rules.Term, n, used int) Entry {
	t.Helper()
	f := &fixture{}
	f.add(skillFile, used, "보스 등장", "BOSS 登場").add(skillFile, n-used, "보스 등장", "老大登場")
	return entryOf(t, Analyze(Inputs{Rules: testRules(term), Units: f.units}), "보스")
}

func TestEntryRateAndHardCandidate(t *testing.T) {
	hard := approved("보스", "BOSS")
	hard.Check = rules.CheckHard
	for _, c := range []struct {
		name    string
		term    rules.Term
		n, used int
		rate    float64
		cand    bool
	}{
		{"20 of 20", approved("보스", "BOSS"), 20, 20, 1, true},
		{"19 of 20 is 0.95, not above it", approved("보스", "BOSS"), 20, 19, 0.95, false},
		{"19 of 19 is too few", approved("보스", "BOSS"), 19, 19, 1, false},
		{"already hard", hard, 20, 20, 1, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			e := rateEntry(t, c.term, c.n, c.used)
			if e.N != c.n || e.Used != c.used || e.Rate != c.rate || e.HardCandidate != c.cand {
				t.Errorf("n %d used %d rate %v candidate %v, want %d %d %v %v", e.N, e.Used, e.Rate, e.HardCandidate, c.n, c.used, c.rate, c.cand)
			}
		})
	}
	if e := rateEntry(t, hard, 3, 3); e.Check != rules.CheckHard {
		t.Errorf("check = %q, want hard", e.Check)
	}
	if th := thresholds(); th.HardRate != 0.95 || th.HardMinN != 20 {
		t.Errorf("thresholds = %+v", th)
	}
	if e := entryOf(t, Analyze(Inputs{Rules: testRules(approved("없음", "無"))}), "없음"); e.Rate != 0 || e.HardCandidate {
		t.Errorf("entry without hits = %+v", e)
	}
}

// Units lists the real hits of each entry, as the review page decides them:
// a hit covered by a longer entry is not one, nor is the compound unit of a
// word entry; a substring entry's compound unit is.
func TestReportUnits(t *testing.T) {
	f := &fixture{}
	f.add(skillFile, 2, "보스 등장", "BOSS 登場")
	f.addAs(skillFile, 1, "보스 등장", "老大登場", store.OriginHuman, "보스 用老大")
	f.add(skillFile, 1, "최종 보스", "最終頭目")  // covered by the longer entry
	f.add(skillFile, 1, "최종보스", "最終BOSS") // compound: no word hit
	r := testRules(approved("보스", "BOSS"), approved("최종 보스", "最終頭目"))
	rep := Analyze(Inputs{Rules: r, Units: f.units})
	wordRep := rep
	boss, long := entryOf(t, rep, "보스"), entryOf(t, rep, "최종 보스")
	us := rep.Units[boss.ID]
	if len(us) != 3 || boss.N != 3 || boss.Used != 2 {
		t.Fatalf("units of 보스 = %+v, entry n %d used %d, want the 3 units of 보스 등장", us, boss.N, boss.Used)
	}
	for _, u := range us {
		hit := r.Glossary[0]
		if u.KR != "보스 등장" || u.File != skillFile || u.Key == "" || u.Used != hit.Used(u.ZH) {
			t.Errorf("unit = %+v", u)
		}
	}
	if us[2].Origin != store.OriginHuman || us[2].Note != "보스 用老大" || us[2].Used {
		t.Errorf("unit = %+v, want the human unit with its note, not used", us[2])
	}
	if got := rep.Units[long.ID]; len(got) != 1 || got[0].KR != "최종 보스" {
		t.Errorf("units of 최종 보스 = %+v", got)
	}
	// A substring entry lists its compound unit too: it is a hit.
	sub := approved("보스", "BOSS")
	sub.Match = rules.MatchSubstring
	rep = Analyze(Inputs{Rules: testRules(sub), Units: f.units})
	e := entryOf(t, rep, "보스")
	got := rep.Units[e.ID]
	usedN := 0
	for _, u := range got {
		if u.Used {
			usedN++
		}
	}
	if len(got) != 5 || e.N != len(got) || e.Used != usedN || usedN != 3 {
		t.Errorf("substring entry: %d units, n %d, used %d (counted %d), want 5, n and used the units', 3 used", len(got), e.N, e.Used, usedN)
	}
	// A word entry counts its compound unit nowhere.
	wordUsed := 0
	for _, u := range wordRep.Units[boss.ID] {
		if u.Used {
			wordUsed++
		}
	}
	if boss.N != len(wordRep.Units[boss.ID]) || boss.Used != wordUsed {
		t.Errorf("word entry n %d used %d, units %d used %d", boss.N, boss.Used, len(wordRep.Units[boss.ID]), wordUsed)
	}
}

// senseUnits builds units of one file for the sense counts.
func senseUnit(i int, zh, en, jp string) Unit {
	return Unit{Key: fmt.Sprintf("%s|/dataList/%d/content", storyFile, i), File: storyFile, KR: "침식", ZH: zh, EN: en, JP: jp, Origin: store.OriginClaude}
}

func TestSenseCounts(t *testing.T) {
	term := approved("침식", "侵蝕")
	term.EN, term.JP = "Erosion", "浸食"
	us := []Unit{
		senseUnit(0, "侵蝕", "Erosion", "浸食"),         // used: not classified
		senseUnit(1, "腐蝕", "the Erosion grows", ""), // same (EN, word, any case)
		senseUnit(2, "腐蝕", "Erosions grow", ""),     // other: EN aligned, not as a word
		senseUnit(3, "腐蝕", "a corrosion", "浸食された"),  // other: EN decides, JP is not asked
		senseUnit(4, "腐蝕", "", "浸食された"),             // same via JP
		senseUnit(5, "腐蝕", "", "腐食"),                // other via JP
		senseUnit(6, "腐蝕", "", ""),                  // unknown: no alignment
		senseUnit(7, "腐蝕", "  ", ""),                // unknown: blank alignment
	}
	e := entryOf(t, Analyze(Inputs{Rules: testRules(term), Units: us}), "침식")
	if e.Same != 2 || e.Other != 3 || e.Unknown != 2 || e.Used != 1 || e.N != 8 {
		t.Fatalf("entry same/other/unknown = %d/%d/%d (used %d, n %d), want 2/3/2 (1, 8)", e.Same, e.Other, e.Unknown, e.Used, e.N)
	}
	var g Group
	for _, x := range e.Groups {
		if x.Name == "family:story" {
			g = x
		}
	}
	if g.Same != 2 || g.Other != 3 || g.Unknown != 2 {
		t.Fatalf("group same/other/unknown = %d/%d/%d, want 2/3/2", g.Same, g.Other, g.Unknown)
	}
}

func TestSenseUnknownWithoutAlignment(t *testing.T) {
	enOnly := approved("침식", "侵蝕")
	enOnly.EN = "Erosion"
	none := approved("침식", "侵蝕")
	us := []Unit{senseUnit(0, "腐蝕", "", "浸食された"), senseUnit(1, "腐蝕", "Erosion", "")}
	// The entry has an en and the unit no EN alignment, and no jp to look at: unknown.
	e := entryOf(t, Analyze(Inputs{Rules: testRules(enOnly), Units: us}), "침식")
	if e.Same != 1 || e.Other != 0 || e.Unknown != 1 {
		t.Fatalf("en-only entry = %d/%d/%d, want 1/0/1", e.Same, e.Other, e.Unknown)
	}
	// An entry with neither en nor jp knows nothing of any unit.
	e = entryOf(t, Analyze(Inputs{Rules: testRules(none), Units: us}), "침식")
	if e.Same != 0 || e.Other != 0 || e.Unknown != 2 {
		t.Fatalf("empty entry = %d/%d/%d, want 0/0/2", e.Same, e.Other, e.Unknown)
	}
}

func TestEntryUnitsCarryTheSense(t *testing.T) {
	term := approved("침식", "侵蝕")
	term.EN = "Erosion"
	us := []Unit{
		senseUnit(0, "侵蝕", "Erosion", ""),
		senseUnit(1, "腐蝕", "the Erosion grows", ""),
		senseUnit(2, "腐蝕", "a corrosion", ""),
		senseUnit(3, "腐蝕", "", ""),
	}
	rep := Analyze(Inputs{Rules: testRules(term), Units: us})
	e := entryOf(t, rep, "침식")
	got := map[string]string{}
	for _, u := range rep.Units[e.ID] {
		got[u.Key] = u.Sense
	}
	want := map[string]string{us[0].Key: "", us[1].Key: SenseSame, us[2].Key: SenseOther, us[3].Key: SenseUnknown}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("senses = %v, want %v", got, want)
	}
}
