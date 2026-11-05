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

// TermGroup is the id of the group of units that lack the rendering of one
// glossary entry (term-missing warnings). It is no warning id, so a unit is
// accepted by its own Warning id.
func TermGroup(kr, zh string) string {
	return "termgroup/" + Sha8(kr) + "/" + Sha8(zh)
}

// Note is the id of a translator's note of the unit with this key.
func Note(key, note string) string {
	return "note/" + key + "/" + Sha8(note)
}
