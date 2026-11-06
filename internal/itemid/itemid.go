// Package itemid builds the stable ids of the items people review: a group of
// inconsistent translations, a translator's note, a check warning. It imports
// nothing from this repo, so that both internal/build and internal/suggest can
// use it.
package itemid

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"slices"
)

// Sha8 is the first 8 lowercase hex digits of the sha256 of text.
func Sha8(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:4])
}

// Inconsistent is the id of one group of inconsistent translations.
func Inconsistent(family, field, kr string) string {
	return "inconsistent/" + family + "/" + field + "/" + Sha8(kr)
}

// Warning is the id of a check warning of the given code about this KR text and
// its translation. A change of the translation changes the id.
func Warning(code, kr, zh string) string {
	return "warning/" + code + "/" + Sha8(kr) + "/" + Sha8(zh)
}

// UnitWarning is the id under which rules/accepted.yaml records a check warning
// of code about one unit's KR text and its translation; termKR and termZH are
// the KR and the rendering of the glossary entry the warning is about, if any
// (validate.Issue.Term and TermZH). The build and the review page both use it,
// so that they agree on what is accepted.
//
// A term-missing warning is about one glossary entry, and a unit that lacks
// two entries is a unit of two term groups: its first part hashes the entry, KR
// and rendering, in with the KR text, so that the unit has an id in each group
// and accepting it in one leaves the other open. The rendering counts, since
// two entries of one KR text (one limited by files, such as 신부 → 牧師 in
// chapter 10 and 神父 elsewhere) are two groups. Any other code ignores the
// entry and is Warning.
func UnitWarning(code, termKR, termZH, kr, zh string) string {
	if code == termMissing {
		return Warning(code, kr+"\x00"+termKR+"\x00"+termZH, zh)
	}
	return Warning(code, kr, zh)
}

// termMissing is the code of the warning of a missing glossary rendering
// (validate.CodeTermMissing, which this package cannot import).
const termMissing = "term-missing"

// TermGroupPrefix is what the id of a term group starts with (TermGroup).
const TermGroupPrefix = "termgroup/"

// TermGroup is the id of the group of units that lack the rendering of one
// glossary entry (term-missing warnings). It is no warning id, so a unit is
// accepted by its own UnitWarning id.
func TermGroup(kr, zh string) string {
	return TermGroupPrefix + Sha8(kr) + "/" + Sha8(zh)
}

// Note is the id of a translator's note of the unit with this key.
func Note(key, note string) string {
	return "note/" + key + "/" + Sha8(note)
}

// ScopePrefix is what the id of a scope proposal starts with (Scope).
const ScopePrefix = "scope/"

// The kinds of a scope proposal: where an entry of the glossary could apply,
// as the translations use it (internal/scope). The first four are a change of
// the entry; the last two leave the change to a person.
const (
	ScopeNotFiles  = "not-files" // add patterns to not_files
	ScopeFiles     = "files"     // limit files to patterns
	ScopeSubstring = "substring" // match: substring
	ScopeWord      = "word"      // match: word, in place of substring
	ScopeMixed     = "mixed"     // a group uses the rendering in part: a person looks
	ScopeExplained = "explained" // notes often explain why the rendering is not used
)

// ScopeKinds lists the kinds of a scope proposal, in the order the review page
// shows them.
var ScopeKinds = []string{ScopeNotFiles, ScopeFiles, ScopeSubstring, ScopeWord, ScopeMixed, ScopeExplained}

// Scope is the id of the proposal of kind about the glossary entry kr -> zh:
// scope/<sha8 KR>/<sha8 ZH>/<kind>. The id does not change with what the
// proposal says, so a declined proposal stays declined when the numbers
// behind it move.
func Scope(kr, zh, kind string) string {
	return ScopePrefix + Sha8(kr) + "/" + Sha8(zh) + "/" + kind
}

// scopeRe is the id of a scope proposal; the kind is checked apart.
var scopeRe = regexp.MustCompile(`^scope/[0-9a-f]{8}/[0-9a-f]{8}/([a-z-]+)$`)

// IsScope reports whether id is the id of a scope proposal: Scope of some KR,
// rendering and one of ScopeKinds.
func IsScope(id string) bool {
	m := scopeRe.FindStringSubmatch(id)
	return m != nil && slices.Contains(ScopeKinds, m[1])
}

// GlossaryEntryPrefix is what the id of a glossary entry starts with
// (GlossaryEntry).
const GlossaryEntryPrefix = "glossary/"

// GlossaryEntry is the id of the glossary entry kr -> zh:
// glossary/<sha8 KR>/<sha8 ZH>. A person copies it from the review page to
// give an instruction about the entry ("add also 對象"); it names no item to
// confirm, so lct accept and rules/declined.yaml take none.
func GlossaryEntry(kr, zh string) string {
	return GlossaryEntryPrefix + Sha8(kr) + "/" + Sha8(zh)
}
