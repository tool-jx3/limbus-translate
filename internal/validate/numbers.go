package validate

import "regexp"

// Numbers returns where the numbers of the text of s are, in order of
// appearance: each maximal run of ASCII digits that is no part of an [ID], a
// placeholder or a tag (IDs, Placeholders, Tags), as the byte offsets of its
// start and its end. Those tokens are not text: a translation keeps them as KR
// writes them, digits and all. Every run is a number of its own, so "1,000"
// is two.
func Numbers(s string) [][2]int {
	// token[i]: byte i of s is part of a token.
	var token []bool
	for _, re := range []*regexp.Regexp{idRe, placeholderRe, tagRe} {
		for _, loc := range re.FindAllStringIndex(s, -1) {
			if token == nil {
				token = make([]bool, len(s))
			}
			for i := loc[0]; i < loc[1]; i++ {
				token[i] = true
			}
		}
	}
	var out [][2]int
	for i := 0; i < len(s); {
		if !isDigit(s, i, token) {
			i++
			continue
		}
		start := i
		for i < len(s) && isDigit(s, i, token) {
			i++
		}
		out = append(out, [2]int{start, i})
	}
	return out
}

// isDigit reports whether byte i of s is an ASCII digit outside the tokens.
func isDigit(s string, i int, token []bool) bool {
	return s[i] >= '0' && s[i] <= '9' && (token == nil || !token[i])
}
