package validate

import (
	"strings"
	"testing"
)

func TestLevelDiff(t *testing.T) {
	const key = "Skills.json|/dataList/0/levelList/1/desc"
	// check returns the messages of the level-diff issues, with a Game whose
	// PrevLevel has prev under key.
	check := func(prev LevelText, kr, zh string) []string {
		g := NewGame(GameInput{PrevLevel: map[string]LevelText{key: prev}})
		var out []string
		for _, is := range Check(kr, zh, Context{File: "Skills.json", Pointer: "/dataList/0/levelList/1/desc", Game: g}) {
			if is.Code == CodeLevelDiff {
				out = append(out, is.Message)
			}
		}
		return out
	}
	same := LevelText{Level: 1, KR: "가\n나 3", ZH: "甲\n乙3 "}
	for _, c := range []struct {
		name   string
		prev   LevelText
		kr, zh string
		issues int
	}{
		{"the unchanged line differs", same, "가\n나 4", "甲 \n乙4 ", 1},
		{"only the changed line differs", same, "가\n나 4", "甲\n乙 4", 0},
		{"identical", same, "가\n나 3", "甲\n乙3 ", 0},
		{"the lower level lacks the line in zh", LevelText{Level: 1, KR: "가\n나", ZH: "甲"}, "가\n나", "甲\n乙", 1},
		{"KR has more lines now", same, "가\n나 3\n다", "甲\n乙3 \n丙", 0},
		{"a trailing space counts", same, "가\n나 3", "甲\n乙3", 1},
	} {
		if got := check(c.prev, c.kr, c.zh); len(got) != c.issues {
			t.Errorf("%s: %v, want %d issue", c.name, got, c.issues)
		}
	}
	msg := check(same, "가\n나 4", "甲 \n乙4 ")
	if len(msg) != 1 || !strings.Contains(msg[0], "line 1") || !strings.Contains(msg[0], "level 1") {
		t.Errorf("%v", msg)
	}
	if !strings.Contains(msg[0], "by the game's code") {
		t.Errorf("%v", msg)
	}
}

func TestLevelDiffMore(t *testing.T) {
	const key = "Skills.json|/dataList/0/levelList/1/desc"
	long := strings.Repeat("字", 50)
	prev := LevelText{Level: 2, KR: "가\n나\n다", ZH: long + "\n乙\n丙"}
	g := NewGame(GameInput{PrevLevel: map[string]LevelText{key: prev}})
	var msgs []string
	for _, is := range Check("가\n나\n다", "x\ny\n丙", Context{File: "Skills.json", Pointer: "/dataList/0/levelList/1/desc", Game: g}) {
		if is.Code == CodeLevelDiff {
			msgs = append(msgs, is.Message)
		}
	}
	if len(msgs) != 1 || !strings.Contains(msg0(msgs), strings.Repeat("字", 40)+`"`) ||
		strings.Contains(msg0(msgs), strings.Repeat("字", 41)) || !strings.Contains(msg0(msgs), ", and 1 more") {
		t.Errorf("%v", msgs)
	}
}

func msg0(m []string) string {
	if len(m) == 0 {
		return ""
	}
	return m[0]
}

// A unit with no entry of PrevLevel, or checked without a Game, has no issue.
func TestLevelDiffWithoutLowerLevel(t *testing.T) {
	ctx := Context{File: "Skills.json", Pointer: "/dataList/0/levelList/1/desc"}
	ctx.Game = NewGame(GameInput{PrevLevel: map[string]LevelText{"Skills.json|/dataList/9/levelList/1/desc": {Level: 1, KR: "가", ZH: "甲"}}})
	for _, c := range []Context{ctx, {File: ctx.File, Pointer: ctx.Pointer}, {Game: ctx.Game}} {
		for _, is := range Check("가", "乙", c) {
			if is.Code == CodeLevelDiff {
				t.Errorf("%+v: %v", c, is)
			}
		}
	}
	var g *Game
	if _, ok := g.prevLevel("a", "b"); ok {
		t.Error("nil Game has a lower level")
	}
}
