// Package itemid builds the stable ids of the items people review: a group of
// inconsistent translations, a translator's note, a check warning. It imports
// nothing from this repo, so that both internal/build and internal/suggest can
// use it.
package itemid

import (
	"crypto/sha256"
	"encoding/hex"
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
// of code about one unit's KR text and its translation; term is the KR of the
// glossary entry the warning is about, if any (validate.Issue.Term). The build
// and the review page both use it, so that they agree on what is accepted.
//
// A term-missing warning is about one glossary entry, and a unit that lacks
// two entries is a unit of two term groups: its first part hashes the term in
// with the KR text, so that the unit has an id in each group and accepting it
// in one leaves the other open. Any other code ignores term and is Warning.
func UnitWarning(code, term, kr, zh string) string {
	if code == termMissing {
		return Warning(code, kr+"\x00"+term, zh)
	}
	return Warning(code, kr, zh)
}

// termMissing is the code of the warning of a missing glossary rendering
// (validate.CodeTermMissing, which this package cannot import).
const termMissing = "term-missing"

// TermGroup is the id of the group of units that lack the rendering of one
// glossary entry (term-missing warnings). It is no warning id, so a unit is
// accepted by its own UnitWarning id.
func TermGroup(kr, zh string) string {
	return "termgroup/" + Sha8(kr) + "/" + Sha8(zh)
}

// Note is the id of a translator's note of the unit with this key.
func Note(key, note string) string {
	return "note/" + key + "/" + Sha8(note)
}
