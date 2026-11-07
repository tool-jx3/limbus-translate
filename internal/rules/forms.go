package rules

import (
	"bytes"
	"encoding/json"
	"fmt"

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
