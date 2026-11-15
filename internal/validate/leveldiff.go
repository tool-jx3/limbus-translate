package validate

import (
	"fmt"
	"strings"
)

// prevLevel returns the text one uptie level lower of the unit at pointer in
// file. PrevLevel is keyed by file + "|" + pointer, the format of units.Key;
// validate does not import units, so the key is built here.
func (g *Game) prevLevel(file, pointer string) (LevelText, bool) {
	if g == nil {
		return LevelText{}, false
	}
	p, ok := g.prev[file+"|"+pointer]
	return p, ok
}

// levelDiff returns the message of the level-diff issue, or "" when there is
// none: a line that is the same in KR at the lower level and now, but differs
// in the translation. A line that only one level of the translation has counts
// as different; one that neither has does not.
func levelDiff(kr, zh string, prev LevelText) string {
	prevKR, prevZH := strings.Split(prev.KR, "\n"), strings.Split(prev.ZH, "\n")
	curKR, curZH := strings.Split(kr, "\n"), strings.Split(zh, "\n")
	line := func(ls []string, i int) (string, bool) {
		if i < len(ls) {
			return ls[i], true
		}
		return "", false
	}
	first, n := -1, 0
	for i := range curKR {
		if i >= len(prevKR) || prevKR[i] != curKR[i] {
			continue
		}
		a, aok := line(prevZH, i)
		b, bok := line(curZH, i)
		if aok == bok && a == b {
			continue
		}
		if first < 0 {
			first = i
		}
		n++
	}
	if first < 0 {
		return ""
	}
	a, _ := line(prevZH, first)
	b, _ := line(curZH, first)
	msg := `line %d is the same as level %d in KR but differs in the translation (%q / %q); by the game's code the uptie comparison marks it as changed`
	out := fmt.Sprintf(msg, first+1, prev.Level, truncateRunes(a, 40), truncateRunes(b, 40))
	if n > 1 {
		out += fmt.Sprintf(", and %d more", n-1)
	}
	return out
}

// truncateRunes cuts s to at most n runes.
func truncateRunes(s string, n int) string {
	if rs := []rune(s); len(rs) > n {
		return string(rs[:n])
	}
	return s
}
