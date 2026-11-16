package validate

import (
	"fmt"
	"slices"
)

// crossGroups returns the other groups (family and field) that have the KR
// text of the unit at pointer in file. Cross is keyed by file + "|" +
// pointer, the format of units.Key, as PrevLevel is.
func (g *Game) crossGroups(file, pointer string) []CrossText {
	if g == nil {
		return nil
	}
	return g.cross[file+"|"+pointer]
}

// crossGroup returns the message of the cross-group issue, or "" when there
// is none: a group in others that has the KR text and not the translation zh.
// A group that has zh among its translations differs only within itself,
// which is for the suggestion of the texts translated differently to show.
func crossGroup(zh string, others []CrossText) string {
	var first *CrossText
	n := 0
	for i := range others {
		if len(others[i].ZH) == 0 || slices.Contains(others[i].ZH, zh) {
			continue
		}
		if first == nil {
			first = &others[i]
		}
		n++
	}
	if first == nil {
		return ""
	}
	out := fmt.Sprintf("the same KR text is translated otherwise in %s (%s of %s): %q", first.Family, first.Field, first.Key, truncateRunes(first.ZH[0], 60))
	if n > 1 {
		out += fmt.Sprintf(", and in %d more groups", n-1)
	}
	return out
}
