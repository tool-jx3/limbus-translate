package validate

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// sharedLineMin is the least number of characters, tags aside, of a
// translation that shared-line looks at: short lines ("到此為止。") are
// rightly the translation of several KR lines.
const sharedLineMin = 10

// BattleLineFile reports whether file holds battle lines: the speech bubbles
// and the voice lines of the identities, which the game shows in the same
// bubble over the unit.
func BattleLineFile(file string) bool {
	return family(file) == "BattleSpeechBubbleDlg" || strings.HasPrefix(file, "PersonalityVoiceDlg/")
}

// otherLines returns the KR texts of the battle lines that ship zh and are
// not kr; texts that differ only in the spaces around them are the same.
func (g *Game) otherLines(kr, zh string) []string {
	if g == nil {
		return nil
	}
	var out []string
	own := strings.TrimSpace(kr)
	for _, other := range g.shared[zh] {
		if strings.TrimSpace(other) != own {
			out = append(out, other)
		}
	}
	return out
}

// sharedLine returns the message of the shared-line issue, or "" when there
// is none: the translation of a battle line that other KR lines have too.
func sharedLine(kr, zh string, g *Game) string {
	if utf8.RuneCountInString(strings.TrimSpace(tagRe.ReplaceAllString(zh, ""))) < sharedLineMin {
		return ""
	}
	others := g.otherLines(kr, zh)
	if len(others) == 0 {
		return ""
	}
	out := fmt.Sprintf("another KR line of the bubbles and voice lines has the same translation: %q; check that it was not pasted from that line", truncateRunes(others[0], 60))
	if len(others) > 1 {
		out += fmt.Sprintf(", and %d more", len(others)-1)
	}
	return out
}
