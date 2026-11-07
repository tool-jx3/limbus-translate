package validate

import (
	"fmt"
	"path"
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
// halfwidth-bracket, placeholder-bracket, extra-keyword-id, ascii-ellipsis,
// curly-quotes and length-ratio.
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
	CodeHalfwidthBracket    = "halfwidth-bracket"
	CodePlaceholderBracket  = "placeholder-bracket"
	CodeExtraKeywordID      = "extra-keyword-id"
	CodeASCIIEllipsis       = "ascii-ellipsis"
	CodeCurlyQuotes         = "curly-quotes"
	CodeLengthRatio         = "length-ratio"
	CodeTermMissing         = "term-missing"
)

// Issue is one problem Check found in a translation.
type Issue struct {
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Class    Class    `json:"class"`
	Message  string   `json:"message"`
	// Term is the KR of the glossary entry behind a term-missing or
	// forbidden-term issue.
	Term string `json:"term,omitempty"`
	// TermZH is the rendering of the entry behind a term-missing issue: two
	// entries of one KR text (one of them limited by files) differ in it.
	TermZH string `json:"term_zh,omitempty"`
}

// Term is the part of a glossary entry that Check needs.
type Term struct {
	// KR is the Korean term. The term only applies to a unit whose KR text
	// contains it.
	KR string
	// ZH is the approved rendering. term-missing warns when a unit whose KR
	// contains KR lacks it; "" or the same as KR means nothing to check.
	ZH string
	// Forbidden lists renderings the translation must not contain. An entry
	// starting with "re:" is a regular expression (Go syntax), any other
	// entry is a substring.
	Forbidden []string
	// Also lists other correct renderings of KR: a translation that has one
	// of them is as good as one that has ZH (see Used).
	Also []string
	// NoMissing says a translation without ZH is no issue: the entry is
	// general vocabulary (check: off in the glossary), a rendering to offer
	// and not to hold to. Its Forbidden renderings are checked all the same.
	NoMissing bool
}

// Used reports whether zh, a translation, uses a rendering of the term: it
// holds ZH or one of Also, after MatchText on both sides, so tags are no
// text. A term without a ZH to check, or whose ZH is its KR, is used by every
// translation: there is nothing to look for. It is the one place that decides
// it; term-missing and the review page go through it.
func (t Term) Used(zh string) bool {
	if t.ZH == "" || t.ZH == t.KR {
		return true
	}
	plain := MatchText(zh)
	if strings.Contains(plain, MatchText(t.ZH)) {
		return true
	}
	for _, r := range t.Also {
		if r != "" && strings.Contains(plain, MatchText(r)) {
			return true
		}
	}
	return false
}

// Context is what Check needs besides the two strings.
type Context struct {
	// Terms are the glossary terms to enforce.
	Terms []Term
	// Existing is true for a unit that already shipped. Content issues of
	// such a unit are warnings, since the rules may have changed since.
	Existing bool
	// File is the pack file the unit is from, such as RPGSystem/rpg-loc-quest-1.json;
	// "" when it is not known.
	File string
	// ScriptOK are approved glossary renderings whose kana or Hangul are
	// meant, such as し協會 and 하나協會 (association names may be foreign
	// words): the kana and Hangul checks leave the text of these renderings
	// alone (rules.Rules.ScriptRenderings).
	ScriptOK []string
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
	// openingBracketRe and closingBracketRe match the halves of the bracketed
	// placeholder [{0}]: the opening bracket with the placeholder after it,
	// and the placeholder with the closing bracket after it.
	openingBracketRe = regexp.MustCompile(`\[\{[0-9]+\}`)
	closingBracketRe = regexp.MustCompile(`\{[0-9]+\}\]`)
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
	if missing, extra := tagDiff(kr, zh, isStory(ctx.File)); len(missing)+len(extra) > 0 {
		issues = append(issues, ctx.issue(CodeTagMismatch, ClassStructural,
			"tags differ from KR%s; opening tags must keep their attributes", describeDiff(missing, extra)))
	}
	// <noparse> pairs take no part in the comparison above (base.md), so they
	// are checked on their own, whatever KR is.
	// KR strings that are fragments (a lone </color>, an unclosed <color>)
	// cannot be nested correctly by any translation, so the nesting of zh is
	// only checked when KR's own tags are balanced. The comparison above still
	// holds a fragment's translation to the same tags.
	if problem := noparseProblem(zh); problem != "" {
		issues = append(issues, ctx.issue(CodeTagNesting, ClassStructural, "<noparse> must open and close in turn and hold plain text: %s", problem))
	} else if nestingProblem(kr) == "" {
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
// which the official EN text closes at the end of the line), but only at the
// end of the text (base.md): per tag name, as many extra closing tags are
// accepted as KR has openings without a closing tag, which is none if KR's
// tags of that name are balanced, and only those in the run of closing tags
// that ends zh (see trailingClosings). Tags without a closing tag (<br>,
// <sprite>) never take part.
//
// The other way round, zh may open a tag that KR closes and never opens
// (base.md): per tag name, as many extra openings as KR has closings without
// an opening, less the openings zh leaves unclosed.
//
// With addRuby, the text of a story file (StoryData/ or RPG dialogue), zh may also hold <ruby=…>…</ruby>
// pairs KR does not have (base.md: the ruby of a story line is kept even
// where KR writes no ruby, such as 良秀 read Yoshihide or Ryōshū), as long as
// the ruby tags of zh, taken alone, open and close in turn. A ruby of KR still
// counts: zh with fewer than KR has is still missing them.
func tagDiff(kr, zh string, addRuby bool) (missing, extra []string) {
	split := func(s string) (keys []string, written map[string]string) {
		written = map[string]string{}
		for _, tok := range Tags(s) {
			if isNoparse(tok) {
				continue // zh places its own (see noparseProblem)
			}
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
	zhNames := map[string]string{} // comparison key of an opening tag of zh -> its lower-case name
	for _, tok := range Tags(zh) {
		if name, closing := tagName(tok); !closing {
			zhNames[comparableTag(tok)] = strings.ToLower(name)
		}
	}
	missing, extra = multisetDiff(krKeys, zhKeys)
	unclosed := unclosedTags(kr)
	atEnd := map[string]int{}
	for _, name := range trailingClosings(zh) {
		atEnd[name]++
	}
	extra = slices.DeleteFunc(extra, func(key string) bool {
		name, isClosing := strings.CutPrefix(key, "</")
		name = strings.ToLower(strings.TrimSuffix(name, ">"))
		if !isClosing || unclosed[name] <= 0 || atEnd[name] <= 0 {
			return false
		}
		unclosed[name]--
		atEnd[name]--
		return true
	})
	// KR may close a tag it never opens (the lyric `몸을 둥글게</size> 말고`):
	// zh may open it, as JP does, as often as KR leaves it unopened, but only
	// openings that zh closes again (see openLeft) count.
	for name, allowed := range unopenedTags(kr) {
		isOpening := func(key string) bool { return zhNames[key] == name }
		if n := min(allowed, countKeys(extra, isOpening)-openLeft(zh)[name]); n > 0 {
			extra = dropKeys(extra, isOpening, n)
		}
	}
	if addRuby && rubyPaired(zh) {
		added := min(countKeys(extra, isRubyOpen), countKeys(extra, isRubyClose))
		extra = dropKeys(extra, isRubyOpen, added)
		extra = dropKeys(extra, isRubyClose, added)
	}
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

// isStory reports whether file, a pack file name, holds story text, whose
// lines may carry a ruby KR does not have (see tagDiff): a file of StoryData/
// or the dialogue of the RPG (guides/base.md).
func isStory(file string) bool { return strings.HasPrefix(file, "StoryData/") || isRPGDialogue(file) }

// isRPGDialogue reports whether file is a dialogue file of the RPG: a file
// under RPGSystem/ whose name holds "dialogue". The other files of RPGSystem/
// are labels.
func isRPGDialogue(file string) bool {
	return strings.HasPrefix(file, "RPGSystem/") && strings.Contains(path.Base(file), "dialogue")
}

// isRubyOpen and isRubyClose report whether a comparison key of tagDiff is
// that of an opening or a closing ruby tag, in any case.
func isRubyOpen(key string) bool  { return key == rubyOpenKey }
func isRubyClose(key string) bool { return strings.EqualFold(key, "</ruby>") }

// countKeys returns how many of keys match.
func countKeys(keys []string, match func(string) bool) int {
	n := 0
	for _, k := range keys {
		if match(k) {
			n++
		}
	}
	return n
}

// dropKeys returns keys without the first n that match.
func dropKeys(keys []string, match func(string) bool, n int) []string {
	return slices.DeleteFunc(keys, func(k string) bool {
		if n > 0 && match(k) {
			n--
			return true
		}
		return false
	})
}

// rubyPaired reports whether the ruby tags of s, taken alone, open and close
// in turn: each opening is closed before the next one, and the last is
// closed. A text with no ruby is paired.
func rubyPaired(s string) bool {
	open := false
	for _, tok := range Tags(s) {
		name, closing := tagName(tok)
		if !strings.EqualFold(name, "ruby") {
			continue
		}
		if closing != open {
			return false
		}
		open = !closing
	}
	return !open
}

// trailingClosings returns the lower-case names of the closing tags that end
// s: the closing tags at the end of s with nothing but blanks between them
// and after the last. A closing tag followed by text, or by a tag that does
// not close, is not among them, nor is any tag before it.
func trailingClosings(s string) []string {
	locs := tagRe.FindAllStringIndex(s, -1)
	var names []string
	end := len(s)
	for i := len(locs) - 1; i >= 0; i-- {
		start, stop := locs[i][0], locs[i][1]
		name, closing := tagName(s[start:stop])
		if !closing || strings.TrimSpace(s[stop:end]) != "" {
			break
		}
		names = append(names, strings.ToLower(name))
		end = start
	}
	return names
}

// unclosedTags returns, per lower-case tag name, how many more times s opens
// the tag than it closes it. The count is zero or negative for a name that s
// closes as often as it opens or more often, and self-closing tags are left
// out.
func unclosedTags(s string) map[string]int {
	open := map[string]int{}
	for _, tok := range Tags(s) {
		name, closing := tagName(tok)
		name = strings.ToLower(name)
		switch {
		case selfClosing[name]:
		case closing:
			open[name]--
		default:
			open[name]++
		}
	}
	return open
}

// unopenedTags returns, per lower-case tag name, how many more times s closes
// the tag than it opens it, for the names it closes more often.
func unopenedTags(s string) map[string]int {
	unopened := map[string]int{}
	for name, n := range unclosedTags(s) {
		if n < 0 {
			unopened[name] = -n
		}
	}
	return unopened
}

// openLeft returns, per lower-case tag name, how many openings of s are
// never closed: a closing tag closes the last opening of its name before it,
// and one with no such opening closes nothing. Self-closing tags are left
// out.
func openLeft(s string) map[string]int {
	depth := map[string]int{}
	for _, tok := range Tags(s) {
		name, closing := tagName(tok)
		name = strings.ToLower(name)
		switch {
		case selfClosing[name]:
		case !closing:
			depth[name]++
		case depth[name] > 0:
			depth[name]--
		}
	}
	return depth
}

// isNoparse reports whether tok is a <noparse> or </noparse> tag.
func isNoparse(tok string) bool {
	name, _ := tagName(tok)
	return strings.EqualFold(name, "noparse")
}

var noparseRe = regexp.MustCompile(`(?i)<(/?)noparse>`)

// noparseProblem returns what is wrong with the <noparse> pairs of s, or "" if
// they are fine. <noparse> keeps the game from reading a keyword name in the
// text; the names of KR and the translation differ, so a translation places
// its own pairs (base.md), which open and close in turn, one at a time, and
// hold plain text: the game shows a tag or an [ID] inside a pair as written.
func noparseProblem(s string) string {
	open := false
	last := 0
	for _, m := range noparseRe.FindAllStringSubmatchIndex(s, -1) {
		closing := m[3] > m[2]
		if open {
			inside := s[last:m[0]]
			if tags := Tags(inside); len(tags) > 0 {
				return fmt.Sprintf("<noparse>…</noparse> holds %s", tags[0])
			}
			if ids := IDs(inside); len(ids) > 0 {
				return fmt.Sprintf("<noparse>…</noparse> holds %s", ids[0])
			}
		}
		switch {
		case closing && !open:
			return "</noparse> has no <noparse> before it"
		case !closing && open:
			return "<noparse> inside <noparse>"
		}
		open = !closing
		last = m[1]
	}
	if open {
		return "<noparse> is never closed"
	}
	return ""
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
	// The Hangul of an approved rendering such as 하나協會 are meant.
	if hangul := distinct(hangulRe.FindAllString(withoutRenderings(zh, ctx.ScriptOK), -1)); len(hangul) > 0 {
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

// checkTerms reports every forbidden rendering of the terms kr contains, and
// a missing rendering of each of them that is not NoMissing.
func checkTerms(kr, zh string, ctx Context) []Issue {
	var issues []Issue
	// Tags are not text: a tag in the middle of the KR term, or around it,
	// does not change whether the entry hits the unit.
	kr = MatchText(kr)
	for _, term := range ctx.Terms {
		if term.KR == "" || !strings.Contains(kr, term.KR) {
			continue
		}
		first := len(issues)
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
		for i := first; i < len(issues); i++ {
			issues[i].Term = term.KR
		}
		if !term.NoMissing && !term.Used(zh) {
			// A warning whatever Existing says: a reasonable exception is
			// explained in the batch note, not fixed in the text.
			is := ctx.issue(CodeTermMissing, ClassContent,
				"%s should be rendered %q, which the translation lacks; if this is a reasonable exception, name the term %q and the reason in the note",
				term.KR, term.ZH, term.KR)
			is.Severity = SeverityWarn
			is.Term, is.TermZH = term.KR, term.ZH
			issues = append(issues, is)
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

// withoutRenderings is text with every occurrence of the renderings taken out.
func withoutRenderings(text string, renderings []string) string {
	for _, ok := range renderings {
		if ok != "" {
			text = strings.ReplaceAll(text, ok, "")
		}
	}
	return text
}

func checkStyle(kr, zh string, ctx Context) []Issue {
	var issues []Issue
	// ・ (U+30FB) is a middle dot that Chinese text uses, so it is not kana.
	var kana []string
	// The reading of a ruby may be kana (base.md: 良秀 read ヨシヒデ); tags
	// are not text; nor are the kana of an approved rendering such as し協會.
	text := withoutRenderings(tagRe.ReplaceAllString(zh, ""), ctx.ScriptOK)
	for _, r := range text {
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
	if marks := halfwidthBrackets(ctx.File, zh); len(marks) > 0 {
		issues = append(issues, ctx.issue(CodeHalfwidthBracket, ClassStyle,
			"half-width %s in a label of RPGSystem/; use （） instead", quoteList(marks)))
	}
	if frags := unpairedPlaceholderBrackets(kr, zh); len(frags) > 0 {
		issues = append(issues, ctx.issue(CodePlaceholderBracket, ClassStyle,
			"%s is missing its other bracket; keep both brackets of the placeholder as KR does", quoteList(frags)))
	}
	if ids := extraKeywordIDs(kr, zh); len(ids) > 0 {
		issues = append(issues, ctx.issue(CodeExtraKeywordID, ClassStyle,
			"%s is not in KR; where KR names a state in Hangul, write its Chinese name, not an [ID] (guides/base.md)", quoteList(ids)))
	}
	if asciiEllRe.MatchString(zh) {
		issues = append(issues, ctx.issue(CodeASCIIEllipsis, ClassStyle, `"..." should be "……"`))
	}
	if strings.ContainsAny(zh, "“”") {
		issues = append(issues, ctx.issue(CodeCurlyQuotes, ClassStyle, "curly quotes “” should be 「」 or 『』"))
	}
	// Tags are not text: a ruby a story line adds would otherwise count.
	textLen := func(s string) int { return utf8.RuneCountInString(strings.TrimSpace(tagRe.ReplaceAllString(s, ""))) }
	krLen, zhLen := textLen(kr), textLen(zh)
	if krLen >= 10 && (zhLen > 3*krLen || zhLen*5 < krLen) {
		issues = append(issues, ctx.issue(CodeLengthRatio, ClassStyle,
			"the translation has %d characters and KR has %d; the ratio is outside 0.2 to 3", zhLen, krLen))
	}
	return issues
}

// extraKeywordIDs returns the distinct [ID]s of zh that kr does not contain,
// sorted. How often an ID occurs is id-mismatch's to compare; this is about an
// ID that KR has nowhere, such as [Burn] for a state KR writes as 화상. A
// bracketed placeholder like [{0}] is not an ID.
func extraKeywordIDs(kr, zh string) []string {
	have := IDs(kr)
	var extra []string
	for _, id := range IDs(zh) {
		if !slices.Contains(have, id) && !slices.Contains(extra, id) {
			extra = append(extra, id)
		}
	}
	slices.Sort(extra)
	return extra
}

// halfwidthPunct returns the distinct half-width , . : in s that touch Chinese
// text. Tags and placeholders are ignored, and so are the dots of "...", which
// ascii-ellipsis reports, and the colons of "::", as in E.G.O::.
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
		if r == '.' && (prev == '.' || next == '.') || r == ':' && (prev == ':' || next == ':') {
			continue
		}
		if isChinese(prev) || isChinese(next) {
			found = append(found, string(r))
		}
	}
	return distinct(found)
}

// halfwidthBrackets returns the distinct half-width ( ) in zh, a text of file,
// if file is a label of the RPG: a file under RPGSystem/ that is not
// dialogue. Labels use the full-width （） (see guides/families/rpg.md). Tags are
// not text.
func halfwidthBrackets(file, zh string) []string {
	if !strings.HasPrefix(file, "RPGSystem/") || isRPGDialogue(file) {
		return nil
	}
	var found []string
	for _, r := range tagRe.ReplaceAllString(zh, "") {
		if r == '(' || r == ')' {
			found = append(found, string(r))
		}
	}
	return distinct(found)
}

// unpairedPlaceholderBrackets returns the pieces of the bracketed placeholder
// [{n}] that zh has broken: a "[{n}" with no "]" right after it, or a "{n}]"
// with no "[" right before it. Something is only reported when KR pairs its
// brackets and zh breaks more of them than KR does: KR itself writes
// "[{0}장 {1}]", whose pieces are unpaired in the same way.
func unpairedPlaceholderBrackets(kr, zh string) []string {
	if !bracketsBalanced(kr) {
		return nil
	}
	zhPieces := unpairedPieces(zh)
	if len(zhPieces) <= len(unpairedPieces(kr)) {
		return nil
	}
	return distinct(zhPieces)
}

// unpairedPieces returns the pieces of [{n}] in s that lack their other
// bracket, in order of appearance.
func unpairedPieces(s string) []string {
	var pieces []string
	for _, loc := range openingBracketRe.FindAllStringIndex(s, -1) {
		if !strings.HasPrefix(s[loc[1]:], "]") {
			pieces = append(pieces, s[loc[0]:loc[1]])
		}
	}
	for _, loc := range closingBracketRe.FindAllStringIndex(s, -1) {
		if !strings.HasSuffix(s[:loc[0]], "[") {
			pieces = append(pieces, s[loc[0]:loc[1]])
		}
	}
	return pieces
}

// bracketsBalanced reports whether every [ of s is closed by a later ] and
// no ] comes before its [.
func bracketsBalanced(s string) bool {
	depth := 0
	for _, r := range s {
		switch r {
		case '[':
			depth++
		case ']':
			if depth--; depth < 0 {
				return false
			}
		}
	}
	return depth == 0
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
