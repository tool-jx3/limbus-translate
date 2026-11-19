package build

import (
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/rubyx"
)

// KeptLine is a line of a unit whose KR text is the same as before and whose
// translation is not: Old is the line of the translation the unit had, New
// that of the one it has now.
type KeptLine struct {
	KR  string `json:"kr"`
	Old string `json:"old"`
	New string `json:"new"`
}

// KeptRewrite is a unit translated again for this raw tag because its KR
// changed, in which the translator rewrote lines whose KR did not change.
// Note is what the translator noted on the unit, as NameDiffUnit.Note.
type KeptRewrite struct {
	Key   string     `json:"key"`
	Lines []KeptLine `json:"lines"`
	Note  string     `json:"note,omitempty"`
}

// keptLines returns the lines of zh, the translation of kr, that differ from
// the line of prevZH, the translation of prevKR, although the KR line is the
// same in both. The lines of the two KR texts are paired in order (a longest
// common subsequence of equal lines), blank lines left out; a translation is
// compared without its <rubycustomize> tags and the space at its end. It
// returns nil when there is no previous text, or when a translation does not
// have the lines of its KR text one for one: then nothing can be told.
func keptLines(prevKR, prevZH, kr, zh string) []KeptLine {
	if prevKR == "" || prevZH == "" {
		return nil
	}
	oldKR, oldZH := strings.Split(prevKR, "\n"), strings.Split(prevZH, "\n")
	newKR, newZH := strings.Split(kr, "\n"), strings.Split(zh, "\n")
	if len(oldKR) != len(oldZH) || len(newKR) != len(newZH) {
		return nil
	}
	// lcs[i][j] is the length of the longest common subsequence of
	// oldKR[i:] and newKR[j:].
	lcs := make([][]int, len(oldKR)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(newKR)+1)
	}
	for i := len(oldKR) - 1; i >= 0; i-- {
		for j := len(newKR) - 1; j >= 0; j-- {
			if oldKR[i] == newKR[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	plain := func(s string) string { return strings.TrimRight(rubyx.Strip(s), " \t\r") }
	var out []KeptLine
	for i, j := 0, 0; i < len(oldKR) && j < len(newKR); {
		switch {
		case oldKR[i] == newKR[j]:
			if strings.TrimSpace(oldKR[i]) != "" && plain(oldZH[i]) != plain(newZH[j]) {
				out = append(out, KeptLine{KR: newKR[j], Old: oldZH[i], New: newZH[j]})
			}
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			i++
		default:
			j++
		}
	}
	return out
}

// keptContext is how many runes of what two lines share are shown on each
// side of where they differ.
const keptContext = 8

// changedSpan cuts old and now, two lines, down to where they differ: of the
// runes they share at the start and at the end, the context nearest the
// difference stay, and an ellipsis stands for the rest. Lines that are the
// same, or share less than context at an end, are returned as they are there.
func changedSpan(old, now string, context int) (string, string) {
	a, b := []rune(old), []rune(now)
	if old == now {
		return old, now
	}
	pre := 0
	for pre < len(a) && pre < len(b) && a[pre] == b[pre] {
		pre++
	}
	suf := 0
	for suf < len(a)-pre && suf < len(b)-pre && a[len(a)-1-suf] == b[len(b)-1-suf] {
		suf++
	}
	cut := func(r []rune) string {
		start, end := 0, len(r)
		head, tail := "", ""
		if pre > context {
			start, head = pre-context, "…"
		}
		if suf > context {
			end, tail = len(r)-suf+context, "…"
		}
		return head + string(r[start:end]) + tail
	}
	return cut(a), cut(b)
}
