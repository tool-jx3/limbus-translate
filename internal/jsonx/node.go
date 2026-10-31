// Package jsonx is an order-preserving JSON tree. It keeps object member
// order, duplicate keys and number literals exactly as parsed, so game files
// can be edited by JSON pointer and written back without reformatting noise.
package jsonx

import "strconv"

// Kind is the JSON type of a Node.
type Kind int

const (
	Object Kind = iota
	Array
	String
	Number
	Bool
	Null
)

// Node is one JSON value.
//
// Object uses Members, Array uses Items, String uses Str. Number, Bool and
// Null keep their source text in Lit (for example "0.10", "1e5", "true",
// "null") so they are written back unchanged.
type Node struct {
	Kind    Kind
	Members []Member
	Items   []*Node
	Str     string
	Lit     string
}

// Member is one key/value pair of an object. Keys are not unique: duplicates
// are kept in document order.
type Member struct {
	Key   string
	Value *Node
}

// StringField is a string-valued leaf together with its RFC 6901 pointer.
type StringField struct {
	Pointer string
	Value   string
}

// Strings lists every string-valued leaf in document order. Numbers, bools
// and nulls are not listed.
func (n *Node) Strings() []StringField {
	var out []StringField
	n.collectStrings("", &out)
	return out
}

func (n *Node) collectStrings(pointer string, out *[]StringField) {
	if n == nil {
		return
	}
	switch n.Kind {
	case String:
		*out = append(*out, StringField{Pointer: pointer, Value: n.Str})
	case Object:
		for _, m := range n.Members {
			m.Value.collectStrings(pointer+"/"+EscapeToken(m.Key), out)
		}
	case Array:
		for i, item := range n.Items {
			item.collectStrings(pointer+"/"+strconv.Itoa(i), out)
		}
	}
}

// Clone returns a deep copy of n. Cloning nil returns nil.
func (n *Node) Clone() *Node {
	if n == nil {
		return nil
	}
	c := *n
	if n.Members != nil {
		c.Members = make([]Member, len(n.Members))
		for i, m := range n.Members {
			c.Members[i] = Member{Key: m.Key, Value: m.Value.Clone()}
		}
	}
	if n.Items != nil {
		c.Items = make([]*Node, len(n.Items))
		for i, item := range n.Items {
			c.Items[i] = item.Clone()
		}
	}
	return &c
}
