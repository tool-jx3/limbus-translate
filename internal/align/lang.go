// Package align finds, for each KR string, the string at the same place in
// another file whose entries may be in a different order, numbered
// differently or have lines added.
package align

import (
	"slices"
	"strconv"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/jsonx"
)

// Method says how a string of another file was matched to a KR string.
type Method string

const (
	MethodPointer  Method = "pointer"  // same JSON pointer
	MethodID       Method = "id"       // dataList entries with the same id
	MethodSequence Method = "sequence" // dataList entries matched in order by shape
)

// LangAligned is the string another file has for one KR string.
type LangAligned struct {
	Value  string
	Method Method
}

// storyPrefix starts the file names of story scripts, whose ids are line
// numbers and are never used to match entries.
const storyPrefix = "StoryData/"

// AlignLang matches the strings of kr to the strings of other, another
// language's version of file. The result is keyed by KR pointer; a KR string
// with no trustworthy match is left out, because a wrong reference is worse
// than none.
//
// Unless both documents have a dataList array whose entries differ, every
// string pairs with the one at the same pointer. Otherwise strings outside
// dataList still do, and the entries are paired first:
//
//   - by id, outside StoryData/, when every entry on both sides has an id,
//     no id repeats within a side, and at least 90% of KR's ids are found;
//   - else by sequence: a longest common subsequence of entry fingerprints
//     (model and the sorted keys other than id). Among equally long ones,
//     the one pairing the most entries with equal ids wins, so a line added
//     without an id does not shift its neighbours.
//
// Paired entries whose models differ are dropped. Within a pair, object
// members match by key and array elements by id when every element on both
// sides has a distinct id, by position otherwise.
func AlignLang(file string, kr, other *jsonx.Node) map[string]LangAligned {
	out := make(map[string]LangAligned)
	if kr == nil || other == nil {
		return out
	}
	krList, ok1 := dataList(kr)
	otherList, ok2 := dataList(other)
	var krEntries, otherEntries []entry
	if ok1 && ok2 {
		krEntries, otherEntries = describe(krList.Items), describe(otherList.Items)
	}
	byEntry := ok1 && ok2 && !slices.Equal(krEntries, otherEntries)

	for _, f := range kr.Strings() {
		if byEntry && strings.HasPrefix(f.Pointer, "/dataList/") {
			continue
		}
		if n, ok := other.Get(f.Pointer); ok && n.Kind == jsonx.String {
			out[f.Pointer] = LangAligned{Value: n.Str, Method: MethodPointer}
		}
	}
	if !byEntry {
		return out
	}

	pairs, method := pairEntries(file, krList.Items, otherList.Items, krEntries, otherEntries)
	for _, p := range pairs {
		if krEntries[p[0]].fp.model != otherEntries[p[1]].fp.model {
			continue
		}
		pairNodes(krList.Items[p[0]], otherList.Items[p[1]], "/dataList/"+strconv.Itoa(p[0]), method, out)
	}
	return out
}

// dataList returns the document's top-level "dataList" array.
func dataList(doc *jsonx.Node) (*jsonx.Node, bool) {
	list, ok := doc.Get("/dataList")
	return list, ok && list.Kind == jsonx.Array
}

// entry is what decides how one dataList element is matched.
type entry struct {
	fp    fingerprint
	id    string
	hasID bool
}

// fingerprint is the shape of an entry, compared when ids cannot be trusted.
type fingerprint struct {
	kind  jsonx.Kind
	model string // the "model" string, or ""
	keys  string // member keys other than "id", sorted, joined by "\x00"
}

func describe(items []*jsonx.Node) []entry {
	out := make([]entry, len(items))
	for i, n := range items {
		e := entry{fp: fingerprint{kind: n.Kind}}
		e.id, e.hasID = idOf(n)
		if n.Kind == jsonx.Object {
			if model, ok := n.Get("/model"); ok && model.Kind == jsonx.String {
				e.fp.model = model.Str
			}
			keys := make([]string, 0, len(n.Members))
			for _, m := range n.Members {
				if m.Key != "id" {
					keys = append(keys, m.Key)
				}
			}
			slices.Sort(keys)
			e.fp.keys = strings.Join(keys, "\x00")
		}
		out[i] = e
	}
	return out
}

// idOf returns the text of an object's "id": a string's value or a number's
// literal.
func idOf(n *jsonx.Node) (string, bool) {
	if n.Kind != jsonx.Object {
		return "", false
	}
	id, ok := n.Get("/id")
	switch {
	case !ok:
		return "", false
	case id.Kind == jsonx.String:
		return id.Str, true
	case id.Kind == jsonx.Number:
		return id.Lit, true
	}
	return "", false
}

// pairEntries pairs the dataList elements of KR and the other file, as
// described on AlignLang.
func pairEntries(file string, kr, other []*jsonx.Node, krEntries, otherEntries []entry) ([][2]int, Method) {
	if !strings.HasPrefix(file, storyPrefix) {
		if pairs, ok := pairByID(kr, other); ok && len(pairs)*10 >= len(kr)*9 {
			return pairs, MethodID
		}
	}
	// One more fingerprint match must outweigh any number of id matches.
	base := int32(min(len(krEntries), len(otherEntries)) + 1)
	pairs := heaviestMatch(len(krEntries), len(otherEntries), func(i, j int) int32 {
		a, b := krEntries[i], otherEntries[j]
		switch {
		case a.fp != b.fp:
			return 0
		case a.hasID && b.hasID && a.id == b.id:
			return base + 1
		}
		return base
	})
	return pairs, MethodSequence
}

// pairByID pairs the elements of a and b that have the same id, in a's
// order. It pairs nothing and reports false unless every element on both
// sides is an object with an id and no id repeats within a side.
func pairByID(a, b []*jsonx.Node) ([][2]int, bool) {
	if _, ok := indexIDs(a); !ok {
		return nil, false
	}
	index, ok := indexIDs(b)
	if !ok {
		return nil, false
	}
	var pairs [][2]int
	for i, n := range a {
		id, _ := idOf(n)
		if j, found := index[id]; found {
			pairs = append(pairs, [2]int{i, j})
		}
	}
	return pairs, true
}

// indexIDs maps each element's id to its position. It reports false when an
// element has no id or an id repeats.
func indexIDs(items []*jsonx.Node) (map[string]int, bool) {
	index := make(map[string]int)
	for i, n := range items {
		id, ok := idOf(n)
		if !ok {
			return nil, false
		}
		if _, dup := index[id]; dup {
			return nil, false
		}
		index[id] = i
	}
	return index, true
}

// pairNodes records, for every string under kr (whose pointer is pointer),
// the string at the corresponding place under other.
func pairNodes(kr, other *jsonx.Node, pointer string, method Method, out map[string]LangAligned) {
	if kr.Kind != other.Kind {
		return
	}
	switch kr.Kind {
	case jsonx.String:
		out[pointer] = LangAligned{Value: other.Str, Method: method}
	case jsonx.Object:
		for _, m := range kr.Members {
			if v := member(other, m.Key); v != nil {
				pairNodes(m.Value, v, pointer+"/"+jsonx.EscapeToken(m.Key), method, out)
			}
		}
	case jsonx.Array:
		if pairs, ok := pairByID(kr.Items, other.Items); ok {
			for _, p := range pairs {
				pairNodes(kr.Items[p[0]], other.Items[p[1]], pointer+"/"+strconv.Itoa(p[0]), method, out)
			}
			return
		}
		for i := range min(len(kr.Items), len(other.Items)) {
			pairNodes(kr.Items[i], other.Items[i], pointer+"/"+strconv.Itoa(i), method, out)
		}
	}
}

// member returns the value of an object's first member with key, or nil.
func member(n *jsonx.Node, key string) *jsonx.Node {
	for _, m := range n.Members {
		if m.Key == key {
			return m.Value
		}
	}
	return nil
}
