package site

import (
	"slices"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/rules"
)

// NaturalLess reports whether a comes before b when runs of ASCII digits are
// compared as numbers and everything else byte by byte: "S2" is before "S10",
// "P10102" before "P10205". Runs that are the same number but are written
// differently ("S01", "S1") are told apart by the bytes of the strings, so that
// it is a total order; it is false for equal strings.
func NaturalLess(a, b string) bool {
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if isDigit(a[i]) && isDigit(b[j]) {
			ie, je := digitsEnd(a, i), digitsEnd(b, j)
			// Without their leading zeros a longer run is the greater number.
			na, nb := strings.TrimLeft(a[i:ie], "0"), strings.TrimLeft(b[j:je], "0")
			if len(na) != len(nb) {
				return len(na) < len(nb)
			}
			if na != nb {
				return na < nb
			}
			i, j = ie, je
			continue
		}
		if a[i] != b[j] {
			return a[i] < b[j]
		}
		i++
		j++
	}
	if aDone, bDone := i == len(a), j == len(b); aDone != bDone {
		return aDone // the shorter one, as a prefix
	}
	return a < b
}

func isDigit(c byte) bool { return '0' <= c && c <= '9' }

// digitsEnd returns the index after the run of digits of s that starts at i.
func digitsEnd(s string, i int) int {
	for i < len(s) && isDigit(s[i]) {
		i++
	}
	return i
}

// OrderFiles returns paths in the order of the viewer's previous and next file:
// by family in the order of the routing's priority (a family it does not list
// comes after the listed ones, by id), and by NaturalLess within a family. It
// returns a new slice and never nil; paths is not changed.
func OrderFiles(paths []string, r *rules.Rules) []string {
	type entry struct {
		path, family string
		rank         int
	}
	entries := make([]entry, len(paths))
	for i, p := range paths {
		f := r.Family(p)
		entries[i] = entry{path: p, family: f, rank: r.Priority(f)}
	}
	slices.SortFunc(entries, func(x, y entry) int {
		switch {
		case x.rank != y.rank:
			return x.rank - y.rank
		case x.family != y.family:
			return strings.Compare(x.family, y.family)
		case NaturalLess(x.path, y.path):
			return -1
		case NaturalLess(y.path, x.path):
			return 1
		}
		return 0
	})
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.path
	}
	return out
}
