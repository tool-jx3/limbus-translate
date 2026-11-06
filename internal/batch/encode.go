package batch

import (
	"bytes"
	"encoding/json"
)

// EncodeFile is v as lct writes a batch file: JSON indented by two spaces,
// with a final newline and without escaping <, > and &. lct batch next writes
// it and the viewer's fixtures are made with it, so that they cannot differ.
func EncodeFile(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
