package validate

import (
	"strings"
	"testing"
)

func TestCrossGroup(t *testing.T) {
	const (
		file = "Skills.json"
		ptr  = "/dataList/4/levelList/0/name"
		key  = file + "|" + ptr
		kr   = "전입 신고 처리하기"
	)
	enemy := CrossText{Family: "enemy", Field: "name", ZH: []string{"處理小鎮新移居申請"}, Key: "Skills_Enemy.json|/dataList/9/levelList/0/name"}
	both := CrossText{Family: "keywords", Field: "name", ZH: []string{"別的譯法", "處理小鎮移居申請"}, Key: "Bufs.json|/dataList/1/name"}
	// check returns the messages of the cross-group issues.
	check := func(cross map[string][]CrossText, zh, pointer string) []string {
		var g *Game
		if cross != nil {
			g = NewGame(GameInput{Cross: cross})
		}
		var out []string
		for _, is := range Check(kr, zh, Context{File: file, Pointer: pointer, Game: g}) {
			if is.Code == CodeCrossGroup {
				if is.Class != ClassStyle {
					t.Errorf("class %q, want a warning", is.Class)
				}
				out = append(out, is.Message)
			}
		}
		return out
	}
	for _, c := range []struct {
		name    string
		cross   map[string][]CrossText
		zh, ptr string
		issues  int
	}{
		{"another group translates the text otherwise", map[string][]CrossText{key: {enemy}}, "處理小鎮移居申請", ptr, 1},
		{"the other group has this translation", map[string][]CrossText{key: {enemy}}, "處理小鎮新移居申請", ptr, 0},
		{"the other group has this translation among others", map[string][]CrossText{key: {both}}, "處理小鎮移居申請", ptr, 0},
		{"two groups, one differs", map[string][]CrossText{key: {both, enemy}}, "處理小鎮移居申請", ptr, 1},
		{"another unit", map[string][]CrossText{"Skills.json|/dataList/5/levelList/0/name": {enemy}}, "處理小鎮移居申請", ptr, 0},
		{"no pointer", map[string][]CrossText{key: {enemy}}, "處理小鎮移居申請", "", 0},
		{"no pack data", nil, "處理小鎮移居申請", ptr, 0},
	} {
		if got := check(c.cross, c.zh, c.ptr); len(got) != c.issues {
			t.Errorf("%s: %v, want %d issue", c.name, got, c.issues)
		}
	}
	msg := check(map[string][]CrossText{key: {both, enemy}}, "處理小鎮移居申請", ptr)
	if len(msg) != 1 || !strings.Contains(msg[0], "enemy") || !strings.Contains(msg[0], "處理小鎮新移居申請") || !strings.Contains(msg[0], enemy.Key) || strings.Contains(msg[0], "keywords") {
		t.Errorf("message %q: want the family, the unit and the translation of the group that differs", msg)
	}
	two := check(map[string][]CrossText{key: {enemy, {Family: "mirror", Field: "name", ZH: []string{"第三種"}, Key: "EGOgift.json|/dataList/0/name"}}}, "處理小鎮移居申請", ptr)
	if len(two) != 1 || !strings.Contains(two[0], "1 more") {
		t.Errorf("message %q: want one issue that counts the second group", two)
	}
}
