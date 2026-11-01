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
	// It cannot match <0>, which is a placeholder.
	tagRe = regexp.MustCompile(`</?[A-Za-z][A-Za-z0-9-]*(=[^>]*)?( [^>]*)?>`)
	// langStyleRe matches a per-language style tag such as <style="dkr"> or
	// <style="DEN_CP9_1">. KR writes dkr, EN den and JP djp; the suffix tells
	// the lyric lines apart. The groups are the opening quote, the language
	// prefix (d plus the language code), the suffix and the closing quote;
	// callers accept a match only when the quotes agree.
	langStyleRe = regexp.MustCompile(`<style=("?)((?i:d(?:kr|en|jp)))((?:_[A-Za-z0-9_]+)?)("?)>`)
)

// IDs returns the [ID] tokens of s in order of appearance, duplicates kept.
func IDs(s string) []string { return idRe.FindAllString(s, -1) }

// Placeholders returns the {…} and <0> placeholders of s in order of
// appearance, duplicates kept.
func Placeholders(s string) []string { return placeholderRe.FindAllString(s, -1) }

// Tags returns the TextMeshPro tags of s, opening and closing, in order of
// appearance, duplicates kept.
func Tags(s string) []string { return tagRe.FindAllString(s, -1) }

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

// comparableTag returns the form of tag token tok, which Tags returned, in
// which tags compare equal when they mean the same: a closing tag is its
// name, and a per-language style tag has its language code replaced by * and
// its suffix lower-cased. The result is only for comparing.
func comparableTag(tok string) string {
	if name, closing := tagName(tok); closing {
		return "</" + name + ">"
	}
	if quote, _, suffix, ok := langStyle(tok); ok {
		return "<style=" + quote + "d*" + strings.ToLower(suffix) + quote + ">"
	}
	return tok
}
