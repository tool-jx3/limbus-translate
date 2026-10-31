package align

import (
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// Status says what became of one string between two versions of a KR file.
type Status string

const (
	StatusUnchanged Status = "unchanged" // same field and KR text, maybe moved
	StatusChanged   Status = "changed"   // an old string edited: text differs
	StatusNew       Status = "new"       // only in the new version
	StatusDeleted   Status = "deleted"   // only in the old version
)

// VersionMatch is one string of the new version and the old string it
// replaces. OldPointer is empty for StatusNew and NewPointer for
// StatusDeleted.
type VersionMatch struct {
	OldPointer, NewPointer string
	Status                 Status
}

// AlignVersions matches the strings of new, this version of a KR file, to
// those of old, the previous version, so that strings moved by an entry or
// field inserted before them keep their translation.
//
// Strings outside the top-level dataList array match by pointer. Strings
// inside it match:
//
//   - by entry id and pointer within the entry, outside StoryData/, when
//     every entry on both sides is an object with an id and no id repeats
//     within a side;
//   - else by a longest common subsequence of (field name, KR text) over all
//     dataList strings in document order. Between two matched strings, an
//     unmatched old string pairs as edited with an unmatched new one of the
//     same field whose text is alike: a Dice coefficient on rune bigrams of
//     at least 0.3. Of the pairings that keep both orders, the one with the
//     most similarity in total wins. The rest are new or deleted.
//
// KR text is compared after every "\r\n" becomes "\n", as in units.KRSha.
// A pair is unchanged when field name and KR text are the same, and changed
// otherwise. The result lists every string of new in document order, then
// the deleted strings of old in document order. A nil document has no
// strings.
func AlignVersions(file string, old, new *jsonx.Node) []VersionMatch {
	olds, news := units.Extract(file, old), units.Extract(file, new)
	byID := entriesHaveIDs(file, old, new)

	// match[i] is the index in olds of the string paired with news[i], or -1.
	match := make([]int, len(news))
	index := make(map[versionKey][]int) // old strings by key, in order
	var oldSeq, newSeq []int            // dataList strings to match by sequence
	for i, u := range olds {
		if u.ItemIndex >= 0 && !byID {
			oldSeq = append(oldSeq, i)
			continue
		}
		k := keyOf(u)
		index[k] = append(index[k], i)
	}
	for i, u := range news {
		match[i] = -1
		if u.ItemIndex >= 0 && !byID {
			newSeq = append(newSeq, i)
			continue
		}
		// A key repeats only with duplicate JSON members; those pair in order.
		k := keyOf(u)
		if q := index[k]; len(q) > 0 {
			match[i], index[k] = q[0], q[1:]
		}
	}
	matchSequence(olds, news, oldSeq, newSeq, match)

	out := make([]VersionMatch, 0, len(news))
	used := make([]bool, len(olds))
	for i, u := range news {
		m := VersionMatch{NewPointer: u.Pointer, Status: StatusNew}
		if o := match[i]; o >= 0 {
			used[o] = true
			m.OldPointer, m.Status = olds[o].Pointer, StatusChanged
			if textOf(olds[o]) == textOf(u) {
				m.Status = StatusUnchanged
			}
		}
		out = append(out, m)
	}
	for o, u := range olds {
		if !used[o] {
			out = append(out, VersionMatch{OldPointer: u.Pointer, Status: StatusDeleted})
		}
	}
	return out
}

// entriesHaveIDs reports whether the dataList strings of file's two versions
// match by entry id, as described on AlignVersions.
func entriesHaveIDs(file string, old, new *jsonx.Node) bool {
	if strings.HasPrefix(file, storyPrefix) {
		return false
	}
	for _, doc := range []*jsonx.Node{old, new} {
		list, ok := dataList(doc)
		if !ok {
			return false
		}
		if _, ok := indexIDs(list.Items); !ok {
			return false
		}
	}
	return true
}

// versionKey identifies a string that matches by key: by entry id and the
// pointer within the entry for a dataList string, by pointer otherwise.
type versionKey struct {
	inEntry bool
	id      string
	pointer string
}

func keyOf(u units.Unit) versionKey {
	if u.ItemIndex < 0 {
		return versionKey{pointer: u.Pointer}
	}
	rel := strings.TrimPrefix(u.Pointer, "/dataList/"+strconv.Itoa(u.ItemIndex))
	return versionKey{inEntry: true, id: u.ItemID, pointer: rel}
}

// text is what sequence matching compares.
type text struct{ field, kr string }

func textOf(u units.Unit) text {
	return text{u.Field, strings.ReplaceAll(u.KR, "\r\n", "\n")}
}

// matchSequence pairs the strings olds[oldSeq] with news[newSeq] by sequence,
// as described on AlignVersions, and records the pairs in match.
func matchSequence(olds, news []units.Unit, oldSeq, newSeq []int, match []int) {
	a := make([]text, len(oldSeq))
	for k, i := range oldSeq {
		a[k] = textOf(olds[i])
	}
	b := make([]text, len(newSeq))
	for k, i := range newSeq {
		b[k] = textOf(news[i])
	}

	// The final pair past both ends closes the last unmatched run.
	pairs := append(LCS(a, b), [2]int{len(a), len(b)})
	i, j := 0, 0
	for _, p := range pairs {
		for _, e := range pairEdits(a[i:p[0]], b[j:p[1]]) {
			match[newSeq[j+e[1]]] = oldSeq[i+e[0]]
		}
		if p[0] < len(a) {
			match[newSeq[p[1]]] = oldSeq[p[0]]
		}
		i, j = p[0]+1, p[1]+1
	}
}

// minDice is the least similarity at which a new string counts as an edit
// of an old one rather than a different string.
const minDice = 0.3

// pairEdits pairs a run of old strings with the run of new strings that
// replaced them, as described on AlignVersions.
func pairEdits(a, b []text) [][2]int {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	ga := make([][]uint64, len(a))
	for k, t := range a {
		ga[k] = grams(t.kr)
	}
	gb := make([][]uint64, len(b))
	for k, t := range b {
		gb[k] = grams(t.kr)
	}
	return heaviestMatch(len(a), len(b), func(x, y int) int32 {
		if a[x].field != b[y].field {
			return 0
		}
		d := dice(ga[x], gb[y])
		if d < minDice {
			return 0
		}
		return int32(math.Round(1000 * d))
	})
}

// grams returns the distinct rune bigrams of s, sorted, or its distinct
// runes when s is shorter than two runes. A single rune has the top bit set
// so that it never equals a bigram.
func grams(s string) []uint64 {
	rs := []rune(s)
	if len(rs) < 2 {
		out := make([]uint64, len(rs))
		for k, r := range rs {
			out[k] = 1<<63 | uint64(r)
		}
		return out
	}
	out := make([]uint64, len(rs)-1)
	for k := range out {
		out[k] = uint64(rs[k])<<32 | uint64(rs[k+1])
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// dice returns the Dice coefficient 2|A∩B| / (|A|+|B|) of two sorted gram
// sets: 1 for equal sets, 0 for disjoint ones.
func dice(a, b []uint64) float64 {
	if len(a)+len(b) == 0 {
		return 1
	}
	common := 0
	for i, j := 0, 0; i < len(a) && j < len(b); {
		switch {
		case a[i] < b[j]:
			i++
		case a[i] > b[j]:
			j++
		default:
			common++
			i++
			j++
		}
	}
	return 2 * float64(common) / float64(len(a)+len(b))
}
