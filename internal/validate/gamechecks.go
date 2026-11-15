package validate

import (
	"regexp"
	"regexp/syntax"
	"strings"
)

// The game checks look at what the game does with a string, which depends on
// the file and the field it is in. They are always warnings, and a message
// that tells what the game does says so "by the game's code": it is worked
// out from the pack, not observed.

var (
	// bracketRe matches any [text] on one line.
	bracketRe = regexp.MustCompile(`\[[^\]\n]*\]`)
	// sinTypeRe matches the name of a sin-type enemy: 죄종 and one or more ?.
	sinTypeRe = regexp.MustCompile(`죄종\?+$`)
	// rubyTagRe matches the start of a ruby tag in any case.
	rubyTagRe = regexp.MustCompile(`(?i)</?ruby`)
	// rubyWrongRe matches the ruby forms the game's dialogue box does not
	// convert: <r=…>, </r> and <ruby> without a reading.
	rubyWrongRe = regexp.MustCompile(`<r=|</r>|<ruby>`)
)

// family returns the family of a pack file: the name of a file in the root
// without .json, up to the first - or _ (Skills_personality-01.json is
// Skills). A file in a folder has no family.
func family(file string) string {
	if strings.Contains(file, "/") {
		return ""
	}
	name := strings.TrimSuffix(file, ".json")
	if i := strings.IndexAny(name, "-_"); i >= 0 {
		name = name[:i]
	}
	return name
}

// fieldPath returns the field of a unit from its RFC 6901 pointer: without
// the leading dataList and the numeric steps, joined by / (/dataList/3/
// levelList/1/desc is levelList/desc).
func fieldPath(pointer string) string {
	var steps []string
	for i, step := range strings.Split(pointer, "/") {
		if step == "" || (i == 1 && step == "dataList") || isDigits(step) {
			continue
		}
		steps = append(steps, step)
	}
	return strings.Join(steps, "/")
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

// inField reports whether the unit is a field fieldPaths lists for family fam.
func inField(fam, field string, scope map[string][]string) bool {
	for _, f := range scope[fam] {
		if f == field {
			return true
		}
	}
	return false
}

var (
	bracketScope = map[string][]string{
		"Skills":   {"levelList/desc", "levelList/coinlist/coindescs/desc", "levelList/coinlist/coindescs/summary"},
		"Passive":  {"desc", "summary"},
		"Passives": {"desc", "summary"},
	}
	pageScope = map[string][]string{
		"AbEvents":     {"eventDesc", "prevDesc", "successDesc", "failureDesc"},
		"ActionEvents": {"desc"},
	}
	newlineSpaceScope = map[string][]string{
		"Personalities":        {"title"},
		"PersonalityCompanion": {"title"},
		"Egos":                 {"name"},
	}
	bulletScope = map[string][]string{
		"StoryTheaterDanteNote":       {"desc"},
		"StoryTheaterDanteNoteDetail": {"desc"},
	}
)

// checkGame runs the game checks on one unit. Those that look at the field
// need ctx.Pointer and are skipped without it.
func checkGame(kr, zh string, ctx Context) []Issue {
	var issues []Issue
	add := func(code, format string, args ...any) {
		issues = append(issues, ctx.issue(code, ClassStyle, format, args...))
	}
	fam, field := family(ctx.File), ""
	known := ctx.Pointer != ""
	if known {
		field = fieldPath(ctx.Pointer)
	}

	if known && inField(fam, field, bracketScope) {
		// Compared by count: the text inside differs between languages.
		if k, z := nonIDBrackets(kr), nonIDBrackets(zh); len(z) > len(k) {
			add(CodeBracketText, "%d bracketed texts where KR has %d: %s; by the game's code a [text] that is no [ID] is looked up as a label and shows Unknown", len(z), len(k), strings.Join(z, " "))
		}
	}
	if known && (inField(fam, field, pageScope) ||
		(fam == "ActionEvents" && field == "options/result" && strings.HasSuffix(ctx.Pointer, "/result/0"))) {
		if k, z := pageCount(kr), pageCount(zh); k != z {
			msg := "%d pages where KR has %d"
			if z < k {
				msg += `; by the game's code a custom language is padded with "..." pages`
			}
			add(CodePageCount, msg, z, k)
		}
	}
	if known && strings.HasPrefix(ctx.File, "StoryData/") && field == "content" {
		if strings.Count(zh, "//") > strings.Count(kr, "//") {
			add(CodeStoryComment, "the translation has more // than KR; by the game's code only the text before the first // is shown")
		}
	}
	if strayBraces(zh) > strayBraces(kr) {
		add(CodeStrayBrace, "the translation has a { or } that is no placeholder and KR does not")
	}
	if known && fam == "ChoiceEventEffect" && field == "content" {
		if bad := badRegexSegment(zh); bad != "" && badRegexSegment(kr) == "" {
			add(CodeRegexChar, "the text %q between braces is no valid regular expression; by the game's code it is compiled as one, so use full-width brackets", bad)
		}
	}
	if strings.Count(zh, `\n`) > strings.Count(kr, `\n`) {
		add(CodeLiteralNewline, `the translation has a literal \n (backslash and n) that KR does not have; use a real line break`)
	}
	if isReserved(zh) && !isReserved(kr) {
		add(CodeReservedValue, "the translation is %q; by the game's code this value means there is no data", strings.TrimSpace(zh))
	}
	if known && inField(fam, field, newlineSpaceScope) && hasSpaceBesideNewline(zh) {
		add(CodeNewlineSpace, "there is a space before or after a line break; by the game's code a line break here becomes a half-width space on a single line")
	}
	if known && fam == "Enemies" && field == "name" && sinTypeRe.MatchString(kr) {
		if k, z := strings.Count(kr, "?"), strings.Count(zh, "?"); k != z {
			add(CodeQmarkCount, "%d half-width ? where KR has %d; by the game's code the count gives the sin type", z, k)
		}
	}
	if known && inField(fam, field, bulletScope) {
		if k, z := strings.Count(kr, "\n-"), strings.Count(zh, "\n-"); k != z {
			add(CodeBulletCount, "%d lines start with a half-width - where KR has %d", z, k)
		}
	}
	if hasWrongRuby(zh) {
		add(CodeRubyForm, "ruby is not written <ruby=…>…</ruby> in lower case; by the game's code only that form is converted")
	}
	if known && ctx.Game != nil && inField(fam, field, keywordScope) {
		if links := ctx.Game.unnamedLinks(kr, zh, fam); len(links) > 0 {
			add(CodeKeywordLink, "by the game's code these words are linked to states KR does not name: %s", strings.Join(links, ", "))
		}
	}
	return issues
}

// nonIDBrackets returns the [text] of s that is not a whole [ID].
func nonIDBrackets(s string) []string {
	var out []string
	for _, m := range bracketRe.FindAllString(s, -1) {
		if loc := idRe.FindStringIndex(m); loc != nil && loc[0] == 0 && loc[1] == len(m) {
			continue
		}
		out = append(out, m)
	}
	return out
}

// pageCount is the number of pages of s: its parts split at a blank line; ""
// and Error have none.
func pageCount(s string) int {
	if s == "" || s == "Error" {
		return 0
	}
	return len(strings.Split(s, "\n\n"))
}

// strayBraces counts the { and } of s that are no part of a placeholder.
func strayBraces(s string) int {
	s = placeholderRe.ReplaceAllString(s, "")
	return strings.Count(s, "{") + strings.Count(s, "}")
}

// badRegexSegment returns the first text between braces of s that does not
// compile as the regular expression {text}, or "" when all do.
func badRegexSegment(s string) string {
	for _, p := range strings.FieldsFunc(s, func(r rune) bool { return r == '{' || r == '}' }) {
		if _, err := syntax.Parse("{"+p+"}", syntax.Perl); err != nil {
			return p
		}
	}
	return ""
}

// isReserved reports whether s is Unknown or Error, the values the game takes
// for no data.
func isReserved(s string) bool {
	t := strings.TrimSpace(s)
	return strings.EqualFold(t, "unknown") || t == "Error"
}

// hasSpaceBesideNewline reports whether a line break of s has a half-width or
// full-width space right before or after it.
func hasSpaceBesideNewline(s string) bool {
	rs := []rune(s)
	isSpace := func(r rune) bool { return r == ' ' || r == '　' }
	for i, r := range rs {
		if r != '\n' {
			continue
		}
		if (i > 0 && isSpace(rs[i-1])) || (i+1 < len(rs) && isSpace(rs[i+1])) {
			return true
		}
	}
	return false
}

// hasWrongRuby reports whether s holds a ruby form the game does not convert.
func hasWrongRuby(s string) bool {
	if rubyWrongRe.MatchString(s) {
		return true
	}
	for _, m := range rubyTagRe.FindAllString(s, -1) {
		if m != "<ruby" && m != "</ruby" {
			return true
		}
	}
	return false
}
