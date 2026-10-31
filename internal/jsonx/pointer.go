package jsonx

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var pointerEscaper = strings.NewReplacer("~", "~0", "/", "~1")

// EscapeToken escapes one reference token for use in an RFC 6901 pointer
// ("~" becomes "~0", "/" becomes "~1").
func EscapeToken(s string) string {
	return pointerEscaper.Replace(s)
}

// SplitPointer splits an RFC 6901 pointer into unescaped reference tokens.
// The empty pointer (the whole document) yields no tokens.
func SplitPointer(p string) ([]string, error) {
	if p == "" {
		return nil, nil
	}
	if p[0] != '/' {
		return nil, fmt.Errorf("jsonx: invalid pointer %q: must be empty or start with '/'", p)
	}
	parts := strings.Split(p[1:], "/")
	for i, part := range parts {
		tok, err := unescapeToken(part)
		if err != nil {
			return nil, fmt.Errorf("jsonx: invalid pointer %q: %w", p, err)
		}
		parts[i] = tok
	}
	return parts, nil
}

func unescapeToken(s string) (string, error) {
	if !strings.Contains(s, "~") {
		return s, nil
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '~' {
			b.WriteByte(s[i])
			continue
		}
		i++
		if i == len(s) {
			return "", errors.New("'~' must be followed by '0' or '1'")
		}
		switch s[i] {
		case '0':
			b.WriteByte('~')
		case '1':
			b.WriteByte('/')
		default:
			return "", fmt.Errorf("'~%c' is not a valid escape", s[i])
		}
	}
	return b.String(), nil
}

// Get returns the node a pointer refers to. An object segment selects the
// first member with that exact key. It reports false for an invalid pointer
// or a path that does not exist.
func (n *Node) Get(pointer string) (*Node, bool) {
	tokens, err := SplitPointer(pointer)
	if err != nil {
		return nil, false
	}
	cur := n
	for _, tok := range tokens {
		if cur == nil {
			return nil, false
		}
		switch cur.Kind {
		case Object:
			var next *Node
			for _, m := range cur.Members {
				if m.Key == tok {
					next = m.Value
					break
				}
			}
			if next == nil {
				return nil, false
			}
			cur = next
		case Array:
			i, ok := arrayIndex(tok)
			if !ok || i >= len(cur.Items) {
				return nil, false
			}
			cur = cur.Items[i]
		default:
			return nil, false
		}
	}
	if cur == nil {
		return nil, false
	}
	return cur, true
}

// SetString replaces the string value at pointer. It fails when the pointer
// does not resolve or the target is not a string.
func (n *Node) SetString(pointer, value string) error {
	if _, err := SplitPointer(pointer); err != nil {
		return err
	}
	target, ok := n.Get(pointer)
	if !ok {
		return fmt.Errorf("jsonx: pointer %q not found", pointer)
	}
	if target.Kind != String {
		return fmt.Errorf("jsonx: pointer %q is not a string", pointer)
	}
	target.Str = value
	return nil
}

// arrayIndex parses an RFC 6901 array index: digits only, no leading zeros.
func arrayIndex(tok string) (int, bool) {
	if tok == "" || (len(tok) > 1 && tok[0] == '0') {
		return 0, false
	}
	for i := 0; i < len(tok); i++ {
		if tok[i] < '0' || tok[i] > '9' {
			return 0, false
		}
	}
	i, err := strconv.Atoi(tok)
	if err != nil {
		return 0, false
	}
	return i, true
}
