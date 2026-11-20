package align

import (
	"strconv"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// goneKey is the key of an old string of an element that the new version of
// its array does not have (see shiftedItems): it matches no new string. No
// string has it: a pointer is empty or starts with "/".
var goneKey = versionKey{pointer: gonePointer}

const gonePointer = "\x00"

// entryKeys returns the keys by which olds, the strings of old, match those
// of new when the dataList entries match by id (entriesHaveIDs): keyOf, but
// for the strings of an entry that new has too, the pointer within the entry
// is the one new writes for the string (newPointers), and the key is goneKey
// when new has no place for it.
func entryKeys(olds []units.Unit, old, new *jsonx.Node) []versionKey {
	keys := make([]versionKey, len(olds))
	for i, u := range olds {
		keys[i] = keyOf(u)
	}
	oldList, _ := dataList(old)
	newList, _ := dataList(new)
	newAt, _ := indexIDs(newList.Items)

	var pointers []string
	for from := 0; from < len(olds); {
		e := olds[from].ItemIndex
		to := from + 1
		if e < 0 {
			from = to
			continue
		}
		// The strings of an entry follow one another.
		for to < len(olds) && olds[to].ItemIndex == e {
			to++
		}
		if at, ok := newAt[olds[from].ItemID]; ok {
			pointers = newPointers(oldList.Items[e], newList.Items[at], "", pointers[:0])
			if len(pointers) == to-from {
				for k, p := range pointers {
					if p == gonePointer {
						keys[from+k] = goneKey
					} else {
						keys[from+k].pointer = p
					}
				}
			}
		}
		from = to
	}
	return keys
}

// newPointers appends to out, for every string under o in document order, the
// pointer at which the new version has the place of that string, or
// gonePointer when it has none. o is a node of the old version and n the node
// of the new one in its place (nil when there is none), which pointer names.
//
// The place of a member of an object is the member of the same key, and that
// of an element of an array is the element at the same index, unless
// shiftedItems finds that elements of the array moved: then it is where
// shiftedItems says, and an element it pairs with none has no place.
func newPointers(o, n *jsonx.Node, pointer string, out []string) []string {
	if o == nil {
		return out
	}
	switch o.Kind {
	case jsonx.String:
		out = append(out, pointer)
	case jsonx.Object:
		for i, m := range o.Members {
			out = newPointers(m.Value, twinMember(o, n, i), pointer+"/"+jsonx.EscapeToken(m.Key), out)
		}
	case jsonx.Array:
		var items []*jsonx.Node // of n, when it is an array too
		var to []int
		if n != nil && n.Kind == jsonx.Array {
			items = n.Items
			to = shiftedItems(o.Items, items)
		}
		for i, item := range o.Items {
			j := i
			if to != nil {
				j = to[i]
			}
			if j < 0 {
				for range item.Strings() {
					out = append(out, gonePointer)
				}
				continue
			}
			var child *jsonx.Node
			if j < len(items) {
				child = items[j]
			}
			out = newPointers(item, child, pointer+"/"+strconv.Itoa(j), out)
		}
	}
	return out
}

// twinMember returns the member of n in the place of o.Members[i]: the one
// of the same key, and of members with one key (jsonx keeps duplicates) the
// one with as many of them before it. It is nil when n is no object or has
// no such member.
func twinMember(o, n *jsonx.Node, i int) *jsonx.Node {
	if n == nil || n.Kind != jsonx.Object {
		return nil
	}
	key, skip := o.Members[i].Key, 0
	for _, m := range o.Members[:i] {
		if m.Key == key {
			skip++
		}
	}
	for _, m := range n.Members {
		if m.Key != key {
			continue
		}
		if skip == 0 {
			return m.Value
		}
		skip--
	}
	return nil
}

// maxShiftCells bounds the table shiftedItems fills for one array (the product
// of the lengths of what is left of both versions after their common start).
// A longer array pairs by position.
const maxShiftCells = 1 << 20

// shiftedItems finds the elements of a, an array within a dataList entry of
// the old version, that are at another index in b, the same array of the new
// version, because elements were inserted or removed before them. It returns
// nil when it finds none: the elements then pair by position, as they always
// did. Otherwise the result holds, for each element of a, the index of the
// element of b it pairs with, or -1 when it pairs with none.
//
// Elements are the same when they hold the same strings (sameContent). The
// elements that stayed or moved are a longest common subsequence of a and b,
// so they keep their order, and of several the one that leaves the most
// elements at their index: an element counts as moved only when no reading of
// the two arrays leaves as many elements unchanged without it. Unless that
// subsequence has an element at another index, nothing moved.
//
// Between two elements that stayed or moved, the old elements pair with the
// new ones in order when there are as many of each: they were edited in
// place. Otherwise an old element pairs with the new one it is like, as
// pairEdits pairs the strings of a story (the strings of an element joined by
// "\n" being its text); the rest of the old ones pair with none, and the rest
// of the new ones are new.
func shiftedItems(a, b []*jsonx.Node) []int {
	start := 0
	for start < len(a) && start < len(b) && sameContent(a[start], b[start]) {
		start++
	}
	a, b = a[start:], b[start:]
	if len(a) == 0 || len(b) == 0 || len(a)*len(b) > maxShiftCells {
		return nil // elements were only added or removed at the end, if at all
	}

	// Every pair counts for more than all the pairs at their index together.
	one := int32(min(len(a), len(b))) + 1
	same := heaviestMatch(len(a), len(b), func(i, j int) int32 {
		switch {
		case !sameContent(a[i], b[j]):
			return 0
		case i == j:
			return one + 1
		}
		return one
	})
	shifted := false
	for _, p := range same {
		shifted = shifted || p[0] != p[1]
	}
	if !shifted {
		return nil
	}

	to := make([]int, start+len(a))
	for i := range to {
		to[i] = i
		if i >= start {
			to[i] = -1
		}
	}
	// The final pair past both ends closes the last run.
	i, j := 0, 0
	for _, p := range append(same, [2]int{len(a), len(b)}) {
		if p[0]-i == p[1]-j {
			for k := range p[0] - i {
				to[start+i+k] = start + j + k
			}
		} else {
			for _, e := range pairEdits(contentTexts(a[i:p[0]]), contentTexts(b[j:p[1]])) {
				to[start+i+e[0]] = start + j + e[1]
			}
		}
		if p[0] < len(a) {
			to[start+p[0]] = start + p[1]
		}
		i, j = p[0]+1, p[1]+1
	}
	return to
}

// sameContent reports whether x and y hold the same strings: they are of one
// kind, two strings have the same KR text (as textOf compares it), two arrays
// the same content element by element, and two objects the same content
// member by member, with the same keys in the same order. Numbers, bools and
// nulls hold no string: any two of them are the same, and such a member of an
// object is passed over.
func sameContent(x, y *jsonx.Node) bool {
	if isScalar(x) && isScalar(y) {
		return true
	}
	if x.Kind != y.Kind {
		return false
	}
	switch x.Kind {
	case jsonx.String:
		return x.Str == y.Str || strings.ReplaceAll(x.Str, "\r\n", "\n") == strings.ReplaceAll(y.Str, "\r\n", "\n")
	case jsonx.Array:
		if len(x.Items) != len(y.Items) {
			return false
		}
		for i := range x.Items {
			if !sameContent(x.Items[i], y.Items[i]) {
				return false
			}
		}
		return true
	case jsonx.Object:
		i, j := 0, 0
		for {
			for i < len(x.Members) && isScalar(x.Members[i].Value) {
				i++
			}
			for j < len(y.Members) && isScalar(y.Members[j].Value) {
				j++
			}
			if i == len(x.Members) || j == len(y.Members) {
				return i == len(x.Members) && j == len(y.Members)
			}
			if x.Members[i].Key != y.Members[j].Key || !sameContent(x.Members[i].Value, y.Members[j].Value) {
				return false
			}
			i++
			j++
		}
	}
	return false
}

// isScalar reports whether n is a number, a bool or a null.
func isScalar(n *jsonx.Node) bool {
	return n.Kind == jsonx.Number || n.Kind == jsonx.Bool || n.Kind == jsonx.Null
}

// contentTexts is the text of each of items for pairEdits: its strings in
// document order, joined by "\n".
func contentTexts(items []*jsonx.Node) []text {
	out := make([]text, len(items))
	for i, n := range items {
		var b strings.Builder
		for k, s := range n.Strings() {
			if k > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(strings.ReplaceAll(s.Value, "\r\n", "\n"))
		}
		out[i] = text{kr: b.String()}
	}
	return out
}
