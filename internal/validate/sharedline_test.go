package validate

import (
	"strings"
	"testing"
)

func TestSharedLine(t *testing.T) {
	const (
		zh     = "沒有理由離開吧，女兒。因為妳遺忘了一切。"
		bubble = "BattleSpeechBubbleDlg-a1c9p3.json"
		ptr    = "/dataList/258/dlg"
	)
	two := map[string][]string{zh: {"떠날 이유는 없을 거야, 딸.", "미리 익숙해지는 게 낫지 않겠니."}}
	// check returns the messages of the shared-line issues.
	check := func(shared map[string][]string, kr, zh, file, pointer string) []string {
		var g *Game
		if shared != nil {
			g = NewGame(GameInput{SharedLines: shared})
		}
		var out []string
		for _, is := range Check(kr, zh, Context{File: file, Pointer: pointer, Game: g}) {
			if is.Code == CodeSharedLine {
				if is.Class != ClassStyle {
					t.Errorf("class %q, want a warning", is.Class)
				}
				out = append(out, is.Message)
			}
		}
		return out
	}
	const kr = "미리 익숙해지는 게 낫지 않겠니."
	for _, c := range []struct {
		name          string
		shared        map[string][]string
		kr, zh        string
		file, pointer string
		issues        int
	}{
		{"another KR line has the same translation", two, kr, zh, bubble, ptr, 1},
		{"a voice line", two, kr, zh, "PersonalityVoiceDlg/Voice_Faust_10201.json", "/dataList/3/dlg", 1},
		{"only this KR has the translation", map[string][]string{zh: {kr}}, kr, zh, bubble, ptr, 0},
		{"the other KR differs only in the spaces around it", map[string][]string{zh: {kr, " " + kr + " "}}, kr, zh, bubble, ptr, 0},
		{"a short line", map[string][]string{"到此為止。": {"거기까지.", "이제 끝내지."}}, "거기까지.", "到此為止。", bubble, ptr, 0},
		{"tags do not count as length", map[string][]string{"<i>到此為止。</i>": {"거기까지.", "이제 끝내지."}}, "거기까지.", "<i>到此為止。</i>", bubble, ptr, 0},
		{"the note of a bubble", two, kr, zh, bubble, "/dataList/258/desc", 0},
		{"a story line", two, kr, zh, "StoryData/S950I2.json", "/dataList/2/content", 0},
		{"no pointer", two, kr, zh, bubble, "", 0},
		{"no pack data", nil, kr, zh, bubble, ptr, 0},
	} {
		if got := check(c.shared, c.kr, c.zh, c.file, c.pointer); len(got) != c.issues {
			t.Errorf("%s: %v, want %d issue", c.name, got, c.issues)
		}
	}
	msg := check(two, kr, zh, bubble, ptr)
	if len(msg) != 1 || !strings.Contains(msg[0], "떠날 이유는 없을 거야, 딸.") || strings.Contains(msg[0], kr) {
		t.Errorf("message %q: want the other KR line and not this one", msg)
	}
}

func TestBattleLineFile(t *testing.T) {
	for file, want := range map[string]bool{
		"BattleSpeechBubbleDlg.json":                 true,
		"BattleSpeechBubbleDlg_mowe.json":            true,
		"PersonalityVoiceDlg/Voice_Faust_10201.json": true,
		"EGOVoiceDig/Voice_EGO_Faust.json":           false,
		"StoryData/S950I2.json":                      false,
		"BattleUIText.json":                          false,
	} {
		if got := BattleLineFile(file); got != want {
			t.Errorf("BattleLineFile(%q) = %v, want %v", file, got, want)
		}
	}
}
