package rules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// Forms are the JP or the EN of a glossary entry: the forms the word takes
// there (Resistance and Resist), any of which counts as the entry's. In the
// glossary it is a string or a list of strings; an absent, null or empty
// value is no form. In JSON one form is the string it always was, so a batch
// of single-form entries is unchanged byte for byte; more forms are a list;
// none is left out (the fields say omitempty).
//
// The forms are kept as written: lint reports an empty one or one given twice
// (see Lint), and every reader skips an empty form (see Each).
type Forms []string

// Each returns the forms that are not empty, in order.
func (f Forms) Each() []string {
	var out []string
	for _, s := range f {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// UnmarshalYAML reads a string, a list of strings or null.
func (f *Forms) UnmarshalYAML(n *yaml.Node) error {
	switch n.Kind {
	case yaml.ScalarNode:
		if n.Tag == "!!null" || n.Value == "" {
			*f = nil
			return nil
		}
		var s string
		if err := n.Decode(&s); err != nil {
			return err
		}
		*f = Forms{s}
		return nil
	case yaml.SequenceNode:
		var list []string
		if err := n.Decode(&list); err != nil {
			return err
		}
		if len(list) == 0 {
			list = nil
		}
		*f = list
		return nil
	}
	return fmt.Errorf("line %d: jp and en take a string or a list of strings", n.Line)
}

// MarshalJSON writes one form as a string and more as a list. It escapes
// nothing beyond what JSON needs, so that an encoder with SetEscapeHTML(false)
// writes < and & as it would in a string field; json.Marshal escapes them
// afterwards as it does any value.
func (f Forms) MarshalJSON() ([]byte, error) {
	var v any = []string(f)
	switch len(f) {
	case 0:
		v = []string{}
	case 1:
		v = f[0]
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

// UnmarshalJSON reads a string, a list of strings or null.
func (f *Forms) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	switch {
	case bytes.Equal(data, []byte("null")):
		*f = nil
		return nil
	case len(data) > 0 && data[0] == '"':
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*f = nil
		if s != "" {
			*f = Forms{s}
		}
		return nil
	}
	var list []string
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	if len(list) == 0 {
		list = nil
	}
	*f = list
	return nil
}

// HoldsEN reports whether the EN text holds the EN form: as a word, ignoring
// case, with its last word as a plural or a singular too (numberForms). It is
// how an EN form is compared everywhere: evidence.Holds (the checks of lct
// glossary evidence and the senses of lct glossary scope) reads EN through
// it, and lint tells two EN forms of an entry apart by it. text is taken as
// it is; evidence.Holds strips its tags first.
func HoldsEN(text, form string) bool {
	for _, v := range numberForms(form) {
		if containsWord(text, v) {
			return true
		}
	}
	return false
}

// sameForm reports whether the forms a and b of lang ("jp" or "en") are one
// form for every check: each holds the other (HoldsEN) in EN, so case and the
// number of the last word do not tell them apart; JP compares as a substring,
// so only the same text does.
func sameForm(lang, a, b string) bool {
	if lang == "en" {
		return HoldsEN(a, b) && HoldsEN(b, a)
	}
	return a == b
}

// numberForms is the candidate and the forms its last word takes as a plural
// or a singular: +s, +es, y to ies, and without a final s, es or ies (to y).
// The words before it stay as they are.
func numberForms(candidate string) []string {
	// The head ends after the last non-letter, whatever its width in bytes.
	cut := 0
	if i := strings.LastIndexFunc(candidate, func(r rune) bool { return !unicode.IsLetter(r) }); i >= 0 {
		_, w := utf8.DecodeRuneInString(candidate[i:])
		cut = i + w
	}
	head, last := candidate[:cut], strings.ToLower(candidate[cut:])
	out := []string{candidate}
	if last == "" {
		return out
	}
	add := func(stem string) { out = append(out, head+stem) }
	// Case follows the text compared with: containsWord ignores it.
	add(last + "s")
	add(last + "es")
	if strings.HasSuffix(last, "y") && len(last) > 1 {
		add(last[:len(last)-1] + "ies")
	}
	if strings.HasSuffix(last, "ies") && len(last) > 4 {
		add(last[:len(last)-3] + "y")
	}
	if strings.HasSuffix(last, "es") && len(last) > 4 {
		add(last[:len(last)-2])
	}
	if strings.HasSuffix(last, "s") && len(last) > 3 {
		add(last[:len(last)-1])
	}
	return out
}

// containsWord reports whether text holds word, ignoring case, with no letter
// or digit right before or after it where it starts or ends with one.
func containsWord(text, word string) bool {
	text, word = strings.ToLower(text), strings.ToLower(word)
	if word == "" {
		return false
	}
	first, _ := utf8.DecodeRuneInString(word)
	last, _ := utf8.DecodeLastRuneInString(word)
	for from := 0; from < len(text); {
		i := strings.Index(text[from:], word)
		if i < 0 {
			return false
		}
		at := from + i
		from = at + 1
		if r, _ := utf8.DecodeLastRuneInString(text[:at]); at > 0 && isWordRune(r) && isWordRune(first) {
			continue
		}
		if r, _ := utf8.DecodeRuneInString(text[at+len(word):]); at+len(word) < len(text) && isWordRune(r) && isWordRune(last) {
			continue
		}
		return true
	}
	return false
}

func isWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' }
