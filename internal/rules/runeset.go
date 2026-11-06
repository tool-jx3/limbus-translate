package rules

import "unicode/utf8"

// RuneSet is a Bloom filter of the runes and of the pairs of neighbouring
// runes of a text. With thousands of provisional terms, scanning each of them
// in every unit's text is the cost of TermsIn, and a term whose first rune, or
// first two, are not in the text cannot occur in it.
type RuneSet struct {
	one  [16]uint64 // 1024 bits
	pair [64]uint64 // 4096 bits
}

func pairHash(a, b rune) uint32 { return (uint32(a)*31 + uint32(b)) * 2654435761 >> 20 }

// NewRuneSet makes the set of text.
func NewRuneSet(text string) *RuneSet {
	var set RuneSet
	prev := utf8.RuneError
	for i, r := range text {
		h := uint32(r) % 1024
		set.one[h/64] |= 1 << (h % 64)
		if i > 0 {
			h = pairHash(prev, r)
			set.pair[h/64] |= 1 << (h % 64)
		}
		prev = r
	}
	return &set
}

// MayStartTerm reports whether a term with this KR text may occur in the text
// of the set: false only when its first rune, or its first two, are not in
// the text. A KR whose first runes are not valid UTF-8 (which a byte search
// reads differently from range) always may.
func (s *RuneSet) MayStartTerm(kr string) bool {
	a, size := utf8.DecodeRuneInString(kr)
	if a == utf8.RuneError {
		return true
	}
	b, _ := utf8.DecodeRuneInString(kr[size:])
	if b == utf8.RuneError {
		// One rune, or one followed by something to leave to the search.
		h := uint32(a) % 1024
		return s.one[h/64]&(1<<(h%64)) != 0
	}
	h := pairHash(a, b)
	return s.pair[h/64]&(1<<(h%64)) != 0
}
