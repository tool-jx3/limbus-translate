package validate

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// fullwidthPunct is the punctuation that needs no space beside an [ID].
const fullwidthPunct = "，。、；：！？「」『』（）《》〈〉【】…—～"

var (
	// tabExplainRe matches a [TabExplain] and the blanks before it. The
	// marker only stops keyword triggers and must touch the text before it.
	tabExplainRe = regexp.MustCompile(`[ \t\x{3000}]+\[TabExplain\]`)
	// variableRe matches the name of an all-caps variable, such as ATTRIBUTES,
	// which the game replaces at run time.
	variableRe = regexp.MustCompile(`^[A-Z0-9_]{2,}$`)
	unescaper  = strings.NewReplacer("&lt;", "<", "&gt;", ">")
)

// Normalize repairs the mechanical formatting problems of translation zh:
//
//   - &lt; and &gt; become < and >
//   - a half-width space is put on both sides of an [ID], unless the neighbor
//     is a string end, a space or full-width punctuation; before an [ID] the
//     neighbor may also be > or (, after it < ) or [ (half-width brackets
//     are punctuation too)
//   - adjacent IDs are separated: "][" becomes "] ["
//   - all-caps variables such as [ATTRIBUTES] get no added space
//   - the space before [TabExplain] is removed
//
// Normalize is idempotent. kr is the Korean original; no rule depends on it
// today, but callers pass it so a rule that does can be added without
// changing them.
func Normalize(kr, zh string) string {
	s := unescaper.Replace(zh)
	s = tabExplainRe.ReplaceAllString(s, "[TabExplain]")
	s = spaceAroundIDs(s)
	return separateBrackets(s)
}

// spaceAroundIDs adds the spaces base.md asks for around each [ID].
func spaceAroundIDs(s string) string {
	locs := idRe.FindAllStringIndex(s, -1)
	if locs == nil {
		return s
	}
	var b strings.Builder
	last := 0
	for _, loc := range locs {
		start, end := loc[0], loc[1]
		name := s[start+1 : end-1]
		spaced := name != "TabExplain" && !variableRe.MatchString(name)

		b.WriteString(s[last:start])
		if spaced && needsSpaceBefore(s[:start]) {
			b.WriteByte(' ')
		}
		b.WriteString(s[start:end])
		if spaced && needsSpaceAfter(s[end:]) {
			b.WriteByte(' ')
		}
		last = end
	}
	b.WriteString(s[last:])
	return b.String()
}

// needsSpaceBefore reports whether an [ID] that follows text needs a space.
func needsSpaceBefore(text string) bool {
	r, _ := utf8.DecodeLastRuneInString(text)
	return text != "" && !unicode.IsSpace(r) && !strings.ContainsRune(fullwidthPunct, r) && !strings.ContainsRune(">(", r)
}

// needsSpaceAfter reports whether an [ID] that precedes text needs a space.
func needsSpaceAfter(text string) bool {
	r, _ := utf8.DecodeRuneInString(text)
	return text != "" && !unicode.IsSpace(r) && !strings.ContainsRune(fullwidthPunct, r) && !strings.ContainsRune("<)[", r)
}

// separateBrackets turns "][" into "] [", except before a [TabExplain], which
// must stay attached to what precedes it.
func separateBrackets(s string) string {
	var b strings.Builder
	for {
		i := strings.Index(s, "][")
		if i < 0 {
			break
		}
		b.WriteString(s[:i+1])
		s = s[i+1:]
		if !strings.HasPrefix(s, "[TabExplain]") {
			b.WriteByte(' ')
		}
	}
	b.WriteString(s)
	return b.String()
}
