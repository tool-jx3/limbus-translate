package validate

import (
	"regexp"
	"slices"
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
	// keywordCountRe matches the count a keyword name follows: 2級, 3層, 1回.
	keywordCountRe = regexp.MustCompile(`[0-9]+[級層回]`)
	// keywordOrPlaceholderEndRe matches a text that ends with an [ID] or with
	// a bracketed placeholder such as [{0}].
	keywordOrPlaceholderEndRe = regexp.MustCompile(`(?:\[[A-Za-z][A-Za-z0-9_]*\]|\[\{[^{}]+\}\])$`)
)

// Normalize repairs the mechanical formatting problems of translation zh:
//
//   - &lt; and &gt; become < and >
//   - a half-width space is put on both sides of an [ID], unless the neighbor
//     is a string end, a space or full-width punctuation; before an [ID] the
//     neighbor may also be > or (, after it < ) or [ (half-width brackets
//     are punctuation too); a "]" before an [ID] asks for a space only if it
//     ends an [ID] or a placeholder [{n}], not if it ends another pair of
//     brackets ("名][Burn]" stays)
//   - adjacent IDs are separated: "[Burn][Bleed]" becomes "[Burn] [Bleed]",
//     while a "][" between other brackets is left alone
//   - all-caps variables such as [ATTRIBUTES] get no added space
//   - the space before [TabExplain] is removed
//   - the language of a per-language style tag becomes jp, whatever KR and EN
//     write: <style="dkr_CP6_2"> becomes <style="djp_CP6_2">, <style="DKR_CP9_1">
//     becomes <style="DJP_CP9_1"> (the case of the letters, the quotes and the
//     suffix are kept; other styles such as "highlight" are left alone)
//
// Normalize is idempotent. kr is the Korean original; no rule depends on it
// today, but callers pass it so a rule that does can be added without
// changing them.
func Normalize(kr, zh string) string {
	s := unescaper.Replace(zh)
	s = tabExplainRe.ReplaceAllString(s, "[TabExplain]")
	s = langStyleRe.ReplaceAllStringFunc(s, toJapaneseStyle)
	s = spaceAroundIDs(s)
	return separateAdjacentIDs(s)
}

// toJapaneseStyle rewrites a per-language style tag, which langStyleRe
// matched, to the language the Chinese translation uses. Each letter of the
// new language code takes the case of the letter it replaces.
func toJapaneseStyle(tok string) string {
	quote, prefix, suffix, ok := langStyle(tok)
	if !ok {
		return tok
	}
	code := []byte("jp")
	for i := range code {
		if prefix[1+i] >= 'A' && prefix[1+i] <= 'Z' {
			code[i] -= 'a' - 'A'
		}
	}
	return tok[:len("<style")] + "=" + quote + prefix[:1] + string(code) + suffix + quote + ">" // the name keeps its case
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
	if r == ']' && !keywordOrPlaceholderEndRe.MatchString(text) {
		return false // the end of another pair of brackets, such as the "][" of a mirrored name
	}
	return text != "" && !unicode.IsSpace(r) && !strings.ContainsRune(fullwidthPunct, r) && !strings.ContainsRune(">(", r)
}

// needsSpaceAfter reports whether an [ID] that precedes text needs a space.
func needsSpaceAfter(text string) bool {
	r, _ := utf8.DecodeRuneInString(text)
	return text != "" && !unicode.IsSpace(r) && !strings.ContainsRune(fullwidthPunct, r) && !strings.ContainsRune("<)[", r)
}

// separateAdjacentIDs puts a space between two [ID]s that touch, except before
// a [TabExplain], which must stay attached to what precedes it. Other
// brackets are left alone: a mirrored event name such as 射影戰鬥][鬥戰影射
// keeps its "][" in every language.
func separateAdjacentIDs(s string) string {
	locs := idRe.FindAllStringIndex(s, -1)
	var b strings.Builder
	last := 0
	for i := 1; i < len(locs); i++ {
		start, end := locs[i][0], locs[i][1]
		if locs[i-1][1] == start && s[start:end] != "[TabExplain]" {
			b.WriteString(s[last:start])
			b.WriteByte(' ')
			last = start
		}
	}
	b.WriteString(s[last:])
	return b.String()
}

// NormalizeKeywordSpace puts a half-width space after each keyword name that
// follows a count (2級, 3層, 1回) and is not followed by one: "2級破裂強度"
// becomes "2級破裂 強度", "2層強壯，" becomes "2層強壯 ，" and a name at the
// end of the text gets the space too. The game matches "keyword name + space"
// to show the effect text of a status, so KR text that names a status in
// Korean, without an [ID], needs it in the translation.
//
// names are the Chinese keyword names. The longest name that fits is used
// ("破裂易損" before "破裂"), whatever the order of names. A name not
// preceded by a count is left alone, and so is a name followed by
// [TabExplain], which must touch the text before it (see Normalize).
// NormalizeKeywordSpace is idempotent and does not change names.
func NormalizeKeywordSpace(zh string, names []string) string {
	locs := keywordCountRe.FindAllStringIndex(zh, -1)
	if locs == nil || len(names) == 0 {
		return zh
	}
	names = slices.DeleteFunc(slices.Clone(names), func(n string) bool { return n == "" })
	// A name that is the start of another one is shorter in bytes, too.
	slices.SortFunc(names, func(a, b string) int { return len(b) - len(a) })

	var b strings.Builder
	last := 0
	for _, loc := range locs {
		if loc[0] < last {
			continue // inside the name before it
		}
		rest := zh[loc[1]:]
		i := slices.IndexFunc(names, func(n string) bool { return strings.HasPrefix(rest, n) })
		if i < 0 {
			continue
		}
		after := loc[1] + len(names[i])
		if strings.HasPrefix(zh[after:], " ") || strings.HasPrefix(zh[after:], "[TabExplain]") {
			continue
		}
		b.WriteString(zh[last:after])
		b.WriteByte(' ')
		last = after
	}
	b.WriteString(zh[last:])
	return b.String()
}
