package validate

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// Severity says whether an issue blocks a translation.
type Severity string

const (
	// SeverityHard issues reject the translation.
	SeverityHard Severity = "hard"
	// SeverityWarn issues are reported but do not reject it.
	SeverityWarn Severity = "warn"
)

// Class groups issue codes by what they protect.
type Class string

const (
	// ClassStructural issues break how the game displays the text. They are
	// always hard.
	ClassStructural Class = "structural"
	// ClassContent issues break the wording rules (leftover Hangul,
	// Simplified characters, forbidden terms). They are hard, but only a
	// warning for an existing unit (Context.Existing), because the rules may
	// have changed after the unit shipped.
	ClassContent Class = "content"
	// ClassStyle issues are always warnings.
	ClassStyle Class = "style"
)

// Issue codes. Structural codes are empty, id-mismatch, placeholder-mismatch,
// tag-mismatch, tag-nesting and dante-wrapper; content codes are hangul,
// simplified and forbidden-term; style codes are kana, halfwidth-punct,
// ascii-ellipsis, curly-quotes and length-ratio.
const (
	CodeEmpty               = "empty"
	CodeIDMismatch          = "id-mismatch"
	CodePlaceholderMismatch = "placeholder-mismatch"
	CodeTagMismatch         = "tag-mismatch"
	CodeTagNesting          = "tag-nesting"
	CodeDanteWrapper        = "dante-wrapper"
	CodeHangul              = "hangul"
	CodeSimplified          = "simplified"
	CodeForbiddenTerm       = "forbidden-term"
	CodeKana                = "kana"
	CodeHalfwidthPunct      = "halfwidth-punct"
	CodeASCIIEllipsis       = "ascii-ellipsis"
	CodeCurlyQuotes         = "curly-quotes"
	CodeLengthRatio         = "length-ratio"
)

// Issue is one problem Check found in a translation.
type Issue struct {
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Class    Class    `json:"class"`
	Message  string   `json:"message"`
}

// Term is the part of a glossary entry that Check needs.
type Term struct {
	// KR is the Korean term. The term only applies to a unit whose KR text
	// contains it.
	KR string
	// Forbidden lists renderings the translation must not contain. An entry
	// starting with "re:" is a regular expression (Go syntax), any other
	// entry is a substring.
	Forbidden []string
}

// Context is what Check needs besides the two strings.
type Context struct {
	// Terms are the glossary terms to enforce.
	Terms []Term
	// Existing is true for a unit that already shipped. Content issues of
	// such a unit are warnings, since the rules may have changed since.
	Existing bool
}

// issue builds an Issue with the severity its class has in this context.
func (c Context) issue(code string, class Class, format string, args ...any) Issue {
	sev := SeverityWarn
	switch class {
	case ClassStructural:
		sev = SeverityHard
	case ClassContent:
		sev = SeverityHard
		if c.Existing {
			sev = SeverityWarn
		}
	}
	return Issue{Code: code, Severity: sev, Class: class, Message: fmt.Sprintf(format, args...)}
}

// HasHard reports whether any of issues is hard.
func HasHard(issues []Issue) bool {
	return slices.ContainsFunc(issues, func(is Issue) bool { return is.Severity == SeverityHard })
}

var (
	hangulRe   = regexp.MustCompile(`[\x{AC00}-\x{D7A3}\x{1100}-\x{11FF}\x{3130}-\x{318F}]`)
	asciiEllRe = regexp.MustCompile(`\.{3,}`)
)

// IsDanteLine reports whether s, ignoring blanks at both ends, is a whole line
// of Dante's dialogue: wrapped in <…> without starting with a tag or a <0>
// placeholder. Any other character can follow the <, a letter or a digit
// included (<N사에서 …>, <0.06%면 …>). A token written like a tag counts as one
// here even when its name is not a tag name (<DAY ■■ …>, a report title).
func IsDanteLine(s string) bool {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "<") || !strings.HasSuffix(s, ">") {
		return false
	}
	for _, re := range []*regexp.Regexp{tagShapeRe, placeholderRe} {
		if loc := re.FindStringIndex(s); loc != nil && loc[0] == 0 {
			return false // <color=#fff>文字</color>, </b>, <0>
		}
	}
	return true
}

// Check validates translation zh of the Korean text kr. It does not change zh;
// callers pass it through Normalize first. The issues come structural first,
// then content, then style.
func Check(kr, zh string, ctx Context) []Issue {
	if strings.TrimSpace(zh) == "" {
		if strings.TrimSpace(kr) == "" {
			return nil // nothing to translate
		}
		// Everything else would only repeat that the translation is empty.
		return []Issue{ctx.issue(CodeEmpty, ClassStructural, "the translation is empty but the KR text is not")}
	}
	var issues []Issue
	issues = append(issues, checkStructure(kr, zh, ctx)...)
	issues = append(issues, checkContent(kr, zh, ctx)...)
	issues = append(issues, checkStyle(kr, zh, ctx)...)
	return issues
}

func checkStructure(kr, zh string, ctx Context) []Issue {
	var issues []Issue
	if missing, extra := multisetDiff(IDs(kr), IDs(zh)); len(missing)+len(extra) > 0 {
		issues = append(issues, ctx.issue(CodeIDMismatch, ClassStructural,
			"[ID]s differ from KR%s; keep every [ID] exactly as in KR", describeDiff(missing, extra)))
	}
	if missing, extra := multisetDiff(Placeholders(kr), Placeholders(zh)); len(missing)+len(extra) > 0 {
		issues = append(issues, ctx.issue(CodePlaceholderMismatch, ClassStructural,
			"placeholders differ from KR%s", describeDiff(missing, extra)))
	}
	if missing, extra := tagDiff(kr, zh); len(missing)+len(extra) > 0 {
		issues = append(issues, ctx.issue(CodeTagMismatch, ClassStructural,
			"tags differ from KR%s; opening tags must keep their attributes", describeDiff(missing, extra)))
	}
	// KR strings that are fragments (a lone </color>, an unclosed <color>)
	// cannot be nested correctly by any translation, so the nesting of zh is
	// only checked when KR's own tags are balanced. The comparison above still
	// holds a fragment's translation to the same tags.
	if nestingProblem(kr) == "" {
		if problem := nestingProblem(zh); problem != "" {
			issues = append(issues, ctx.issue(CodeTagNesting, ClassStructural, "tags are not nested correctly: %s", problem))
		}
	}
	if IsDanteLine(kr) && !IsDanteLine(zh) {
		issues = append(issues, ctx.issue(CodeDanteWrapper, ClassStructural,
			"KR is a line of Dante's dialogue wrapped in <…>; the translation must be wrapped in <…> too"))
	}
	return issues
}

// tagDiff compares the tags of kr and zh: opening tags by their full text,
// closing tags by name. Closing tags are written as </name> in the result.
// The language of a per-language style tag and the value of a <ruby=…> tag do
// not count (see comparableTag), but the result names the tags as written: the
// KR tag for a missing one, the zh tag for an extra one.
//
// zh may close a tag that KR opens and never closes (the lyric <style="dkr">,
// which the official EN text closes at the end of the line): per tag name, as
// many extra closing tags are accepted as KR has openings without a closing
// tag, which is none if KR's tags of that name are balanced. Tags without a
// closing tag (<br>, <sprite>) never take part.
func tagDiff(kr, zh string) (missing, extra []string) {
	split := func(s string) (keys []string, written map[string]string) {
		written = map[string]string{}
		for _, tok := range Tags(s) {
			key := comparableTag(tok)
			keys = append(keys, key)
			if _, seen := written[key]; seen {
				continue
			}
			written[key] = key // a closing tag is shown as </name>
			if _, closing := tagName(tok); !closing {
				written[key] = tok
			}
		}
		return keys, written
	}
	krKeys, krWritten := split(kr)
	zhKeys, zhWritten := split(zh)
	missing, extra = multisetDiff(krKeys, zhKeys)
	unclosed := unclosedTags(kr)
	extra = slices.DeleteFunc(extra, func(key string) bool {
		name, isClosing := strings.CutPrefix(key, "</")
		name = strings.TrimSuffix(name, ">")
		if !isClosing || unclosed[name] <= 0 {
			return false
		}
		unclosed[name]--
		return true
	})
	for i, key := range missing {
		missing[i] = krWritten[key]
	}
	for i, key := range extra {
		extra[i] = zhWritten[key]
	}
	slices.Sort(missing)
	slices.Sort(extra)
	return missing, extra
}

// unclosedTags returns, per tag name as written, how many more times s opens
// the tag than it closes it. The count is zero or negative for a name that s
// closes as often as it opens or more often, and self-closing tags are left
// out.
func unclosedTags(s string) map[string]int {
	open := map[string]int{}
	for _, tok := range Tags(s) {
		name, closing := tagName(tok)
		switch {
		case selfClosing[strings.ToLower(name)]:
		case closing:
			open[name]--
		default:
			open[name]++
		}
	}
	return open
}

// nestingProblem returns what is wrong with the nesting of the tags of s, or
// "" if it is fine. Self-closing tags do not take part.
func nestingProblem(s string) string {
	var open []string // lower-case names
	for _, tok := range Tags(s) {
		name, closing := tagName(tok)
		name = strings.ToLower(name)
		if selfClosing[name] {
			continue
		}
		switch {
		case !closing:
			open = append(open, name)
		case len(open) == 0:
			return fmt.Sprintf("%s has no opening tag", tok)
		case open[len(open)-1] != name:
			return fmt.Sprintf("%s closes <%s>, which is still open inside it", tok, open[len(open)-1])
		default:
			open = open[:len(open)-1]
		}
	}
	if len(open) > 0 {
		return fmt.Sprintf("<%s> is never closed", open[len(open)-1])
	}
	return ""
}

func checkContent(kr, zh string, ctx Context) []Issue {
	var issues []Issue
	if hangul := distinct(hangulRe.FindAllString(zh, -1)); len(hangul) > 0 {
		issues = append(issues, ctx.issue(CodeHangul, ClassContent, "the translation contains Hangul: %s", quoteList(hangul)))
	}
	var simplified []string
	table := simplifiedChars()
	for _, r := range zh {
		if table[r] {
			simplified = append(simplified, string(r))
		}
	}
	if simplified = distinct(simplified); len(simplified) > 0 {
		issues = append(issues, ctx.issue(CodeSimplified, ClassContent,
			"the translation contains Simplified Chinese characters: %s", quoteList(simplified)))
	}
	return append(issues, checkTerms(kr, zh, ctx)...)
}

// checkTerms reports every forbidden rendering of the terms kr contains.
func checkTerms(kr, zh string, ctx Context) []Issue {
	var issues []Issue
	for _, term := range ctx.Terms {
		if term.KR == "" || !strings.Contains(kr, term.KR) {
			continue
		}
		for _, f := range term.Forbidden {
			if f == "" || f == "re:" {
				continue
			}
			pattern, isRegexp := strings.CutPrefix(f, "re:")
			if !isRegexp {
				if strings.Contains(zh, f) {
					issues = append(issues, ctx.issue(CodeForbiddenTerm, ClassContent,
						"%q is a forbidden rendering of %s", f, term.KR))
				}
				continue
			}
			re, err := compilePattern(pattern)
			if err != nil {
				// A broken rule must not pass silently.
				issues = append(issues, ctx.issue(CodeForbiddenTerm, ClassContent,
					"invalid forbidden pattern %q for %s: %v", f, term.KR, err))
			} else if m := re.FindString(zh); m != "" {
				issues = append(issues, ctx.issue(CodeForbiddenTerm, ClassContent,
					"%q is a forbidden rendering of %s (matches %s)", m, term.KR, f))
			}
		}
	}
	return issues
}

type compiledPattern struct {
	re  *regexp.Regexp
	err error
}

// patterns caches compiled forbidden patterns; Check runs for every unit.
var patterns sync.Map // pattern string -> compiledPattern

func compilePattern(pattern string) (*regexp.Regexp, error) {
	if c, ok := patterns.Load(pattern); ok {
		return c.(compiledPattern).re, c.(compiledPattern).err
	}
	re, err := regexp.Compile(pattern)
	patterns.Store(pattern, compiledPattern{re, err})
	return re, err
}

func checkStyle(kr, zh string, ctx Context) []Issue {
	var issues []Issue
	// ・ (U+30FB) is a middle dot that Chinese text uses, so it is not kana.
	var kana []string
	for _, r := range zh {
		if r >= 0x3040 && r <= 0x30FF && r != 0x30FB {
			kana = append(kana, string(r))
		}
	}
	if kana = distinct(kana); len(kana) > 0 {
		issues = append(issues, ctx.issue(CodeKana, ClassStyle, "the translation contains kana: %s", quoteList(kana)))
	}
	if marks := halfwidthPunct(zh); len(marks) > 0 {
		issues = append(issues, ctx.issue(CodeHalfwidthPunct, ClassStyle,
			"half-width %s next to Chinese text; use ，。： instead", quoteList(marks)))
	}
	if asciiEllRe.MatchString(zh) {
		issues = append(issues, ctx.issue(CodeASCIIEllipsis, ClassStyle, `"..." should be "……"`))
	}
	if strings.ContainsAny(zh, "“”") {
		issues = append(issues, ctx.issue(CodeCurlyQuotes, ClassStyle, "curly quotes “” should be 「」 or 『』"))
	}
	krLen, zhLen := utf8.RuneCountInString(strings.TrimSpace(kr)), utf8.RuneCountInString(strings.TrimSpace(zh))
	if krLen >= 10 && (zhLen > 3*krLen || zhLen*5 < krLen) {
		issues = append(issues, ctx.issue(CodeLengthRatio, ClassStyle,
			"the translation has %d characters and KR has %d; the ratio is outside 0.2 to 3", zhLen, krLen))
	}
	return issues
}

// halfwidthPunct returns the distinct half-width , . : in s that touch Chinese
// text. Tags and placeholders are ignored, and so are the dots of "...", which
// ascii-ellipsis reports.
func halfwidthPunct(s string) []string {
	s = placeholderRe.ReplaceAllString(tagRe.ReplaceAllString(s, ""), "")
	runes := []rune(s)
	var found []string
	for i, r := range runes {
		if r != ',' && r != '.' && r != ':' {
			continue
		}
		var prev, next rune
		if i > 0 {
			prev = runes[i-1]
		}
		if i+1 < len(runes) {
			next = runes[i+1]
		}
		if r == '.' && (prev == '.' || next == '.') {
			continue
		}
		if isChinese(prev) || isChinese(next) {
			found = append(found, string(r))
		}
	}
	return distinct(found)
}

// isChinese reports whether r is a Han character or CJK punctuation.
func isChinese(r rune) bool {
	return unicode.Is(unicode.Han, r) || r >= 0x3000 && r <= 0x303F || r >= 0xFF00 && r <= 0xFFEF
}

// multisetDiff returns the elements of want that got lacks and the elements of
// got that want lacks, counting duplicates, each sorted.
func multisetDiff(want, got []string) (missing, extra []string) {
	counts := map[string]int{}
	for _, s := range want {
		counts[s]++
	}
	for _, s := range got {
		counts[s]--
	}
	for s, n := range counts {
		for ; n > 0; n-- {
			missing = append(missing, s)
		}
		for ; n < 0; n++ {
			extra = append(extra, s)
		}
	}
	slices.Sort(missing)
	slices.Sort(extra)
	return missing, extra
}

// describeDiff words the result of multisetDiff, with a leading ": ".
func describeDiff(missing, extra []string) string {
	var parts []string
	if len(missing) > 0 {
		parts = append(parts, "missing "+strings.Join(missing, " "))
	}
	if len(extra) > 0 {
		parts = append(parts, "unexpected "+strings.Join(extra, " "))
	}
	return ": " + strings.Join(parts, "; ")
}

// distinct returns the elements of s without duplicates, in order of first
// appearance.
func distinct(s []string) []string {
	var out []string
	for _, e := range s {
		if !slices.Contains(out, e) {
			out = append(out, e)
		}
	}
	return out
}

func quoteList(s []string) string {
	return `"` + strings.Join(s, `" "`) + `"`
}
