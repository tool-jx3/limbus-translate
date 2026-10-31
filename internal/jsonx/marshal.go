package jsonx

import (
	"bytes"
	"encoding/json"
)

const indentUnit = "  "

// Marshal renders n as UTF-8 JSON without BOM: LF newlines, 2-space
// indentation, no escaping of <, > or &, member order and number literals as
// held in the tree, and a single trailing newline. Empty containers are
// written as {} and []. The same tree always yields the same bytes.
func Marshal(n *Node) []byte {
	w := &writer{}
	w.enc = json.NewEncoder(&w.buf)
	w.enc.SetEscapeHTML(false)
	w.value(n, 0)
	w.buf.WriteByte('\n')
	return w.buf.Bytes()
}

type writer struct {
	buf bytes.Buffer
	enc *json.Encoder
}

func (w *writer) value(n *Node, depth int) {
	if n == nil {
		w.buf.WriteString("null")
		return
	}
	switch n.Kind {
	case Object:
		if len(n.Members) == 0 {
			w.buf.WriteString("{}")
			return
		}
		w.buf.WriteString("{\n")
		for i, m := range n.Members {
			w.indent(depth + 1)
			w.str(m.Key)
			w.buf.WriteString(": ")
			w.value(m.Value, depth+1)
			if i < len(n.Members)-1 {
				w.buf.WriteByte(',')
			}
			w.buf.WriteByte('\n')
		}
		w.indent(depth)
		w.buf.WriteByte('}')
	case Array:
		if len(n.Items) == 0 {
			w.buf.WriteString("[]")
			return
		}
		w.buf.WriteString("[\n")
		for i, item := range n.Items {
			w.indent(depth + 1)
			w.value(item, depth+1)
			if i < len(n.Items)-1 {
				w.buf.WriteByte(',')
			}
			w.buf.WriteByte('\n')
		}
		w.indent(depth)
		w.buf.WriteByte(']')
	case String:
		w.str(n.Str)
	default: // Number, Bool, Null keep their source text
		w.buf.WriteString(n.Lit)
	}
}

func (w *writer) indent(depth int) {
	for i := 0; i < depth; i++ {
		w.buf.WriteString(indentUnit)
	}
}

// str writes s as a JSON string. Encoder.Encode appends a newline that is
// trimmed again; encoding a string cannot fail.
func (w *writer) str(s string) {
	_ = w.enc.Encode(s)
	w.buf.Truncate(w.buf.Len() - 1)
}
