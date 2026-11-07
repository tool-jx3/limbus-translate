package rules

import (
	"slices"
	"testing"
	"unicode/utf8"
)

// A last separator of more than one byte (U+3000, a typographic apostrophe)
// is kept whole in the head of the plural forms.
func TestNumberFormsMultibyteSeparator(t *testing.T) {
	for _, c := range []struct{ cand, want string }{
		{"Big\u3000Burn", "Big\u3000burns"},
		{"Dante’Burn", "Dante’burns"},
	} {
		got := numberForms(c.cand)
		if !slices.Contains(got, c.want) {
			t.Errorf("numberForms(%q) = %q, want %q among them", c.cand, got, c.want)
		}
		for _, f := range got {
			if !utf8.ValidString(f) {
				t.Errorf("numberForms(%q) holds the invalid UTF-8 %q", c.cand, f)
			}
		}
	}
}
