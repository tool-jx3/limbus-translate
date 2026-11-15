package validate

import (
	"slices"
	"strings"
	"testing"
)

func testGame() *Game {
	return NewGame(GameInput{
		Bufs: []GameBuf{
			{ID: "Laceration", KR: "출혈", ZH: "流血"},
			{ID: "Vibration", KR: "진동", ZH: "震顫"},
			{ID: "VibrationExplosion", KR: "진동 폭발", ZH: "震顫引爆"},
			{ID: "AttackLevelAdder", KR: "공격 레벨+", ZH: "攻擊等級+"},
			{ID: "CheerUpXichun", KR: "원[援]", ZH: "援"},
			{ID: "ConfessionThirdPhase", KR: "〜", ZH: "～"},
			{ID: "ComboSkill", KR: "연", ZH: "連", ExcludeAutoLink: true},
			{ID: "ChoiSwordsmanship", KR: "본국검술[살]", ZH: "本國劍術[肉]"},
			{ID: "HardA", KR: "고난", ZH: "苦難"}, {ID: "HardB", KR: "고난", ZH: "苦難"},
			{ID: "Wrath", KR: "분노", ZH: "憤怒"},
			{ID: "Empty", KR: "빈", ZH: ""}, {ID: "Unknown", KR: "모름", ZH: "Unknown"},
		},
		Resonance: "共鳴", PerfectResonance: "完全共鳴",
	})
}

// names returns the Name of each link.
func names(links []Link) []string {
	var out []string
	for _, l := range links {
		out = append(out, l.Name)
	}
	return out
}

// hasCode reports whether an issue of the code is among issues.
func hasCode(issues []Issue, code string) bool {
	return slices.ContainsFunc(issues, func(is Issue) bool { return is.Code == code })
}

func TestLinkedAfterChar(t *testing.T) {
	g := testGame()
	for _, c := range []struct {
		text string
		want []string
	}{
		{"施加1級流血 強度", []string{"流血"}}, {"施加流血", []string{"流血"}}, {"流血3", []string{"流血"}},
		{"流血　強度", []string{"流血"}}, {"流血３", []string{"流血"}}, {"流血\n", []string{"流血"}},
		{"「流血」", []string{"流血"}}, {"流血・震顫 ", []string{"流血", "震顫"}}, {"流血,", []string{"流血"}},
		{"流血，", nil}, {"流血。", nil}, {"流血強度", nil}, {"流血)", nil}, {"流血</color>", nil},
		{"​流血 ", nil}, {"流血​ ", nil},
		{"攻擊等級+1", []string{"攻擊等級+"}}, {"使用「加斥我援」對", []string{"援"}}, {"增加1～2回", []string{"～"}},
		{"連 ", nil}, // excludeAutoLink
		{"震顫引爆 ", []string{"震顫引爆"}}, // the longest name takes the place
		{"震顫引爆，", nil},              // and keeps it even when it is not linked
		{"憤怒 共鳴時", nil}, {"憤怒 完全 共鳴時", nil}, {"憤怒 攻擊", []string{"憤怒"}},
		// the game only looks for the word when the text goes on after it
		{"憤怒 共鳴", []string{"憤怒"}},
	} {
		if got := names(g.Linked(c.text, false)); !slices.Equal(got, c.want) {
			t.Errorf("Linked(%q) = %v, want %v", c.text, got, c.want)
		}
	}
}

func TestLinkedTagPass(t *testing.T) {
	g := testGame()
	// [TabExplain] is no buff id: with the tag pass it becomes a tag, without it stays "[": neither links.
	for _, tagPass := range []bool{true, false} {
		if got := g.Linked("流血[TabExplain] 強度", tagPass); got != nil {
			t.Errorf("tagPass %v: %v, want none", tagPass, got)
		}
	}
	// A name with brackets is cut by the tag pass, and matched as written without it.
	if got := g.Linked("本國劍術[肉] ", true); got != nil {
		t.Errorf("with the tag pass: %v", got)
	}
	if got := names(g.Linked("本國劍術[肉] ", false)); !slices.Equal(got, []string{"本國劍術[肉]"}) {
		t.Errorf("without the tag pass: %v", got)
	}
	// A buff id in brackets is left alone by the tag pass: the name before it is followed by "[".
	if got := g.Linked("流血[Vibration]", true); got != nil {
		t.Errorf("before an [ID]: %v", got)
	}
}

func TestGameNames(t *testing.T) {
	g := testGame()
	if got := g.Linked("Unknown 與 的 ", false); got != nil { // "" and Unknown are no names
		t.Errorf("%v, want none", got)
	}
	l := g.Linked("苦難 ", false)
	if len(l) != 1 || !slices.Equal(l[0].IDs, []string{"HardA", "HardB"}) || l[0].Start != 0 {
		t.Errorf("%+v", l)
	}
	if got := g.Linked("好流血 ", false); len(got) != 1 || got[0].Start != len("好") {
		t.Errorf("%+v, want Start at the byte offset", got)
	}
}

func TestKeywordLinkCheck(t *testing.T) {
	g := testGame()
	// check returns the Message of each keyword-link issue of Check with Game g.
	check := func(file, pointer, kr, zh string) []string {
		var out []string
		for _, is := range Check(kr, zh, Context{File: file, Pointer: pointer, Game: g}) {
			if is.Code == CodeKeywordLink {
				out = append(out, is.Message)
			}
		}
		return out
	}
	const desc = "/dataList/0/levelList/0/desc"
	for _, c := range []struct {
		name, file, pointer, kr, zh string
		issues                      int
	}{
		{"KR never names the state", "Skills.json", desc, "1~2 증가", "增加1～2回", 1},
		{"KR names it by id", "Skills.json", desc, "[Laceration] 부여", "施加流血 ", 0},
		{"KR names it by its KR name", "Skills.json", desc, "출혈을 부여", "施加流血 ", 0},
		{"one of the ids of the name", "Skills.json", desc, "[HardB] 부여", "施加苦難 ", 0},
		{"not a processed field", "Skills.json", "/dataList/0/levelList/0/name", "이름", "流血 ", 0},
		{"not a processed file", "Bufs.json", "/dataList/0/desc", "설명", "流血 ", 0},
		{"gift desc", "EGOgift_MirrorDungeon.json", "/dataList/0/desc", "설명", "流血 ", 1},
		{"gift simpleDesc", "EGOgift.json", "/dataList/0/simpleDesc/1/simpleDesc", "설명", "流血 ", 1},
		{"panic", "PanicInfo.json", "/dataList/0/panicDescription", "설명", "流血 ", 1},
		{"passive summary", "Passives.json", "/dataList/0/summary", "설명", "流血 ", 1},
		{"coin summary", "Skills.json", "/dataList/0/levelList/0/coinlist/0/coindescs/0/summary", "설명", "流血 ", 1},
	} {
		if got := check(c.file, c.pointer, c.kr, c.zh); len(got) != c.issues {
			t.Errorf("%s: %v, want %d issue", c.name, got, c.issues)
		}
	}
	// One issue per unit names every link; the message is no statement of fact.
	msg := check("Skills.json", desc, "설명", "流血 與增加1～2回")
	if len(msg) != 1 || !strings.Contains(msg[0], "流血") || !strings.Contains(msg[0], "ConfessionThirdPhase") || !strings.Contains(msg[0], "by the game's code") {
		t.Errorf("%v", msg)
	}
	// No Game, or no Pointer: nothing.
	if is := Check("설명", "流血 ", Context{File: "Skills.json", Pointer: desc}); hasCode(is, CodeKeywordLink) {
		t.Error("issue without a Game")
	}
	if is := Check("설명", "流血 ", Context{File: "Skills.json", Game: g}); hasCode(is, CodeKeywordLink) {
		t.Error("issue without a Pointer")
	}
}
