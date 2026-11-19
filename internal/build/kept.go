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

// alignLines pairs the lines of old and now that are equal, in order: a
// longest common subsequence, as indexes into the two.
func alignLines(old, now []string) [][2]int {
	// lcs[i][j] is the length of the longest common subsequence of old[i:]
	// and now[j:].
	lcs := make([][]int, len(old)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(now)+1)
	}
	for i := len(old) - 1; i >= 0; i-- {
		for j := len(now) - 1; j >= 0; j-- {
			if old[i] == now[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	var pairs [][2]int
	for i, j := 0, 0; i < len(old) && j < len(now); {
		switch {
		case old[i] == now[j]:
			pairs = append(pairs, [2]int{i, j})
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			i++
		default:
			j++
		}
	}
	return pairs
}

// splitPaired splits the two KR texts and their translations into lines; ok
// is whether each translation has the lines of its KR text one for one.
func splitPaired(prevKR, prevZH, kr, zh string) (oldKR, oldZH, newKR, newZH []string, ok bool) {
	oldKR, oldZH = strings.Split(prevKR, "\n"), strings.Split(prevZH, "\n")
	newKR, newZH = strings.Split(kr, "\n"), strings.Split(zh, "\n")
	return oldKR, oldZH, newKR, newZH, len(oldKR) == len(oldZH) && len(newKR) == len(newZH)
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
	oldKR, oldZH, newKR, newZH, ok := splitPaired(prevKR, prevZH, kr, zh)
	if !ok {
		return nil
	}
	plain := func(s string) string { return strings.TrimRight(rubyx.Strip(s), " \t\r") }
	var out []KeptLine
	for _, p := range alignLines(oldKR, newKR) {
		i, j := p[0], p[1]
		if strings.TrimSpace(oldKR[i]) != "" && plain(oldZH[i]) != plain(newZH[j]) {
			out = append(out, KeptLine{KR: newKR[j], Old: oldZH[i], New: newZH[j]})
		}
	}
	return out
}

// LineHunk is a run of lines of a unit's KR text that is not the same as
// before, with the lines of the translations at the same places: what went
// (OldKR, OldZH) and what came (NewKR, NewZH), lines joined by "\n". A side
// is empty when lines were only added or only taken away.
type LineHunk struct {
	OldKR string `json:"old_kr"`
	NewKR string `json:"new_kr"`
	OldZH string `json:"old_zh"`
	NewZH string `json:"new_zh"`
}

// Retranslated is a unit translated again for this raw tag because its KR
// changed: the hunks of its KR text that changed with their translations,
// and Kept, the number of its lines that were rewritten although their KR is
// the same (the lines of its KeptRewrite).
type Retranslated struct {
	Key   string     `json:"key"`
	Hunks []LineHunk `json:"hunks"`
	Kept  int        `json:"kept,omitempty"`
}

// lineHunks returns the hunks between prevKR, translated prevZH, and kr,
// translated zh: the runs of lines between the lines the two KR texts share
// (see alignLines). When a translation does not have the lines of its KR text
// one for one, the whole texts are one hunk. It returns nil when there is no
// previous text.
func lineHunks(prevKR, prevZH, kr, zh string) []LineHunk {
	if prevKR == "" || prevZH == "" {
		return nil
	}
	oldKR, oldZH, newKR, newZH, ok := splitPaired(prevKR, prevZH, kr, zh)
	if !ok {
		return []LineHunk{{OldKR: prevKR, NewKR: kr, OldZH: prevZH, NewZH: zh}}
	}
	var out []LineHunk
	hunk := func(i0, i1, j0, j1 int) {
		if i0 == i1 && j0 == j1 {
			return
		}
		out = append(out, LineHunk{
			OldKR: strings.Join(oldKR[i0:i1], "\n"), NewKR: strings.Join(newKR[j0:j1], "\n"),
			OldZH: strings.Join(oldZH[i0:i1], "\n"), NewZH: strings.Join(newZH[j0:j1], "\n"),
		})
	}
	i, j := 0, 0
	for _, p := range alignLines(oldKR, newKR) {
		hunk(i, p[0], j, p[1])
		i, j = p[0]+1, p[1]+1
	}
	hunk(i, len(oldKR), j, len(newKR))
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
