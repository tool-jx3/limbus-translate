package jsonx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// Parse reads one JSON value from data, ignoring a leading UTF-8 BOM. Object
// member order and duplicate keys are preserved, and number literals are kept
// verbatim in Node.Lit.
func Parse(data []byte) (*Node, error) {
	dec := json.NewDecoder(bytes.NewReader(bytes.TrimPrefix(data, utf8BOM)))
	dec.UseNumber()
	n, err := parseValue(dec)
	if err != nil {
		return nil, fmt.Errorf("jsonx: parse error near offset %d: %w", dec.InputOffset(), err)
	}
	switch _, err := dec.Token(); {
	case err == io.EOF:
		return n, nil
	case err != nil:
		return nil, fmt.Errorf("jsonx: parse error near offset %d: %w", dec.InputOffset(), err)
	default:
		return nil, fmt.Errorf("jsonx: unexpected data after top-level value near offset %d", dec.InputOffset())
	}
}

func parseValue(dec *json.Decoder) (*Node, error) {
	tok, err := dec.Token()
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			return parseObject(dec)
		case '[':
			return parseArray(dec)
		}
		return nil, fmt.Errorf("unexpected %q", rune(t))
	case string:
		return &Node{Kind: String, Str: t}, nil
	case json.Number:
		return &Node{Kind: Number, Lit: t.String()}, nil
	case bool:
		return &Node{Kind: Bool, Lit: strconv.FormatBool(t)}, nil
	case nil:
		return &Node{Kind: Null, Lit: "null"}, nil
	}
	return nil, fmt.Errorf("unexpected token %v", tok)
}

func parseObject(dec *json.Decoder) (*Node, error) {
	n := &Node{Kind: Object}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, fmt.Errorf("object key must be a string, got %v", keyTok)
		}
		v, err := parseValue(dec)
		if err != nil {
			return nil, err
		}
		n.Members = append(n.Members, Member{Key: key, Value: v})
	}
	if err := expectClose(dec, '}'); err != nil {
		return nil, err
	}
	return n, nil
}

func parseArray(dec *json.Decoder) (*Node, error) {
	n := &Node{Kind: Array}
	for dec.More() {
		v, err := parseValue(dec)
		if err != nil {
			return nil, err
		}
		n.Items = append(n.Items, v)
	}
	if err := expectClose(dec, ']'); err != nil {
		return nil, err
	}
	return n, nil
}

func expectClose(dec *json.Decoder, want json.Delim) error {
	tok, err := dec.Token()
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return err
	}
	if d, ok := tok.(json.Delim); !ok || d != want {
		return errors.New("malformed container end")
	}
	return nil
}
