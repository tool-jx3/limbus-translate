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
