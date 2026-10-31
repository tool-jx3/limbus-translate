// Package units turns the string fields of a language file into translation
// units, the smallest thing that is translated, tracked and validated.
package units

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/jsonx"
)

// keySep separates the file name from the JSON pointer in a unit key.
const keySep = "|"

// Unit is one string field of one file.
//
// ItemIndex, ItemID and Model describe the element of the top-level dataList
// array that the string lives under (nested lists such as levelList belong to
// the same element). For a string outside dataList, ItemIndex is -1 and
// ItemID and Model are empty.
type Unit struct {
	File      string // pack file name, e.g. "StoryData/S001A.json"
	Pointer   string // RFC 6901 pointer of the string within the file
	Field     string // last non-numeric segment of Pointer
	KR        string // the KR source text
	ItemID    string // the element's "id" as text; numbers keep their literal
	Model     string // the element's "model" string, if any
	ItemIndex int    // index of the element in dataList, or -1
}

// Key returns the unit key "<file>|<pointer>". file is the pack file name
// (see Pack.Files) and pointer follows RFC 6901.
func Key(file, pointer string) string {
	return file + keySep + pointer
}

// SplitKey is the inverse of Key. It cuts at the first "|", so a pointer may
// itself contain "|". It fails when the key has no separator, an empty file
// name, or a pointer that is not valid RFC 6901.
func SplitKey(key string) (file, pointer string, err error) {
	file, pointer, ok := strings.Cut(key, keySep)
	if !ok {
		return "", "", fmt.Errorf("units: invalid key %q: missing %q", key, keySep)
	}
	if file == "" {
		return "", "", fmt.Errorf("units: invalid key %q: empty file name", key)
	}
	if _, err := jsonx.SplitPointer(pointer); err != nil {
		return "", "", fmt.Errorf("units: invalid key %q: %w", key, err)
	}
	return file, pointer, nil
}

// KRSha returns the lowercase hex sha256 of a KR string after every "\r\n" is
// replaced by "\n". It identifies the KR text a translation was made from.
func KRSha(text string) string {
	sum := sha256.Sum256([]byte(strings.ReplaceAll(text, "\r\n", "\n")))
	return hex.EncodeToString(sum[:])
}

// FieldName returns the last non-numeric segment of a pointer, unescaped, so
// "/dataList/3/levelList/0/name" gives "name" and "/dataList/3/tags/0" gives
// "tags". It returns "" for an invalid pointer or one with no such segment.
func FieldName(pointer string) string {
	tokens, err := jsonx.SplitPointer(pointer)
	if err != nil {
		return ""
	}
	for i := len(tokens) - 1; i >= 0; i-- {
		if !isIndex(tokens[i]) {
			return tokens[i]
		}
	}
	return ""
}

// isIndex reports whether a pointer token is made of digits only.
func isIndex(tok string) bool {
	if tok == "" {
		return false
	}
	for i := 0; i < len(tok); i++ {
		if tok[i] < '0' || tok[i] > '9' {
			return false
		}
	}
	return true
}

// Extract returns one Unit per string field of doc, in document order. Empty
// strings are included; numbers, bools and nulls are not.
func Extract(file string, doc *jsonx.Node) []Unit {
	fields := doc.Strings()
	out := make([]Unit, 0, len(fields))

	var items []itemInfo
	if list, ok := doc.Get("/dataList"); ok && list.Kind == jsonx.Array {
		items = make([]itemInfo, len(list.Items))
		for i, item := range list.Items {
			items[i] = describeItem(item)
		}
	}

	for _, f := range fields {
		u := Unit{
			File:      file,
			Pointer:   f.Pointer,
			Field:     FieldName(f.Pointer),
			KR:        f.Value,
			ItemIndex: -1,
		}
		if i, ok := itemIndex(f.Pointer, len(items)); ok {
			u.ItemIndex, u.ItemID, u.Model = i, items[i].id, items[i].model
		}
		out = append(out, u)
	}
	return out
}

// itemInfo is what Extract reads from one dataList element.
type itemInfo struct{ id, model string }

func describeItem(item *jsonx.Node) itemInfo {
	var info itemInfo
	if id, ok := item.Get("/id"); ok {
		switch id.Kind {
		case jsonx.String:
			info.id = id.Str
		case jsonx.Number:
			info.id = id.Lit
		}
	}
	if model, ok := item.Get("/model"); ok && model.Kind == jsonx.String {
		info.model = model.Str
	}
	return info
}

// itemIndex returns the dataList index a pointer of the form
// "/dataList/<n>[/...]" falls under, if it is below count.
func itemIndex(pointer string, count int) (int, bool) {
	rest, ok := strings.CutPrefix(pointer, "/dataList/")
	if !ok {
		return 0, false
	}
	tok, _, _ := strings.Cut(rest, "/")
	i, err := strconv.Atoi(tok)
	if err != nil || i < 0 || i >= count {
		return 0, false
	}
	return i, true
}
