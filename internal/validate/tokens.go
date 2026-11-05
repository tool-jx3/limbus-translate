// Package validate normalizes and checks translated strings. Every string,
// whether it comes from the upstream import, from Claude or from a person,
// passes through Normalize and then Check before it can ship.
package validate

import (
	"regexp"
	"strings"
)

var (
	// idRe matches a keyword reference such as [Laceration] or [ATTRIBUTES].
	idRe = regexp.MustCompile(`\[[A-Za-z][A-Za-z0-9_]*\]`)
	// placeholderRe matches a runtime substitution such as {0}, {Slot} or <0>.
	placeholderRe = regexp.MustCompile(`\{[^{}]+\}|<\d+>`)
	// tagRe matches a TextMeshPro tag such as <color=#f8c200> or </color>.
	// Only the names in richTextTags count, in any case: KR also writes <…>
	// around plain text (the report title <DAY ■■, ■■ ■■ …>), which is not a
	// tag. A name followed by a space needs an attribute (name=value) after
	// it, as in <sprite name="Bleed">; the start of a line of Dante's
	// dialogue such as <B 사무소 …> is text. It cannot match <0>, which is a
	// placeholder.
	tagRe = regexp.MustCompile(`(?i)</?(?:` + strings.Join(richTextTags, "|") + `)(=[^>]*)?( +(?:[a-z][a-z0-9_-]*=[^>]*)?)?>`)
	// tagShapeRe matches anything written like a tag, known name or not.
	// IsDanteLine uses it so that a line starting with such a token is never
	// taken for a line of Dante's dialogue.
	tagShapeRe = regexp.MustCompile(`</?[A-Za-z][A-Za-z0-9-]*(=[^>]*)?( [^>]*)?>`)
	// langStyleRe matches a per-language style tag such as <style="dkr"> or
	// <style="DEN_CP9_1">. KR writes dkr, EN den and JP djp; the suffix tells
	// the lyric lines apart. The groups are the opening quote, the language
	// prefix (d plus the language code), the suffix and the closing quote;
	// callers accept a match only when the quotes agree.
	langStyleRe = regexp.MustCompile(`<(?i:style)=("?)((?i:d(?:kr|en|jp)))((?:_[A-Za-z0-9_]+)?)("?)>`)
)

// richTextTags lists the names of the tags Tags recognises: the TextMeshPro
// rich-text tags, and ruby, which the game adds for the reading above a word.
var richTextTags = []string{
	"align", "allcaps", "alpha", "b", "br", "color", "cspace", "font", "font-weight", "gradient", "i", "indent",
	"line-height", "line-indent", "link", "lowercase", "margin", "mark", "mspace", "nobr", "noparse", "page",
	"pos", "rotate", "s", "size", "smallcaps", "space", "sprite", "strikethrough", "style", "sub", "sup", "u",
	"uppercase", "voffset", "width", "ruby",
}

// IDs returns the [ID] tokens of s in order of appearance, duplicates kept.
func IDs(s string) []string { return idRe.FindAllString(s, -1) }

// Placeholders returns the {…} and <0> placeholders of s in order of
// appearance, duplicates kept.
func Placeholders(s string) []string { return placeholderRe.FindAllString(s, -1) }

// Tags returns the TextMeshPro tags of s, opening and closing, in order of
// appearance, duplicates kept.
func Tags(s string) []string { return tagRe.FindAllString(s, -1) }

// StripTags is s without its tags: they are markup, not text, so a term is
// looked for in the text that is left.
func StripTags(s string) string { return tagRe.ReplaceAllString(s, "") }

// selfClosing lists the tags that have no closing tag.
var selfClosing = map[string]bool{"sprite": true, "br": true}

// tagName returns the name of tag token tok, which Tags returned, and whether
// it is a closing tag.
func tagName(tok string) (name string, closing bool) {
	rest := strings.TrimPrefix(tok, "<")
	rest, closing = strings.CutPrefix(rest, "/")
	end := strings.IndexFunc(rest, func(r rune) bool {
		return !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-')
	})
	if end < 0 {
		end = len(rest)
	}
	return rest[:end], closing
}

// langStyleKey starts the comparison form of every per-language style tag.
const langStyleKey = "\x00lang-style="

// langStyle splits an opening tag tok, which Tags returned, into the quote
// around its value, the language prefix (d plus the language code) and the
// suffix, if it is a per-language style tag. Anything else, such as
// <style="highlight">, <style="dzh"> or a value with one quote, is not.
func langStyle(tok string) (quote, prefix, suffix string, ok bool) {
	m := langStyleRe.FindStringSubmatch(tok)
	if m == nil || m[0] != tok || m[1] != m[4] {
		return "", "", "", false
	}
	return m[1], m[2], m[3], true
}

// rubyOpenKey is the comparison form of every opening <ruby=…> tag.
const rubyOpenKey = "\x00ruby"

// comparableTag returns the form of tag token tok, which Tags returned, in
// which tags compare equal when they mean the same: a closing tag is its
// name, an opening <ruby=…> tag, whatever the case of ruby, is its name only
// (its value is the reading shown above the text, which KR writes in Korean
// and a translation must replace), and a per-language style tag has its
// language code left out and its suffix lower-cased. The name of any other tag
// is compared as written, like the rest of the tag: base.md asks to keep what
// is inside <> unchanged. The result is only for comparing, and the form of a
// ruby or per-language style tag starts with a NUL, which no tag token does,
// so no tag in a text can be equal to it.
func comparableTag(tok string) string {
	name, closing := tagName(tok)
	switch {
	case closing:
		return "</" + name + ">"
	case strings.EqualFold(name, "ruby"):
		return rubyOpenKey
	}
	if quote, _, suffix, ok := langStyle(tok); ok {
		return langStyleKey + quote + strings.ToLower(suffix) + quote
	}
	return tok
}
