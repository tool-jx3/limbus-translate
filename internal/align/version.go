package align

import (
	"strconv"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// Status says what became of one string between two versions of a KR file.
type Status string

const (
	StatusUnchanged Status = "unchanged" // same field and KR text, maybe moved
	StatusChanged   Status = "changed"   // matched to an old string, text differs
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
//     dataList strings in document order. Between two matched strings, the
//     unmatched old and new ones pair in order up to the shorter run; the
//     rest of the longer run are new or deleted.
//
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
			if olds[o].Field == u.Field && olds[o].KR == u.KR {
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

// matchSequence pairs the strings olds[oldSeq] with news[newSeq] by sequence,
// as described on AlignVersions, and records the pairs in match.
func matchSequence(olds, news []units.Unit, oldSeq, newSeq []int, match []int) {
	a := make([]text, len(oldSeq))
	for k, i := range oldSeq {
		a[k] = text{olds[i].Field, olds[i].KR}
	}
	b := make([]text, len(newSeq))
	for k, i := range newSeq {
		b[k] = text{news[i].Field, news[i].KR}
	}

	// The final pair past both ends closes the last unmatched run.
	pairs := append(LCS(a, b), [2]int{len(a), len(b)})
	i, j := 0, 0
	for _, p := range pairs {
		for k := range min(p[0]-i, p[1]-j) {
			match[newSeq[j+k]] = oldSeq[i+k]
		}
		if p[0] < len(a) {
			match[newSeq[p[1]]] = oldSeq[p[0]]
		}
		i, j = p[0]+1, p[1]+1
	}
}
