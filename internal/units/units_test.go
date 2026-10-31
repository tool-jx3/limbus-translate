package units

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/pack"
)

func mustParse(t *testing.T, src string) *jsonx.Node {
	t.Helper()
	n, err := jsonx.Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestKey(t *testing.T) {
	key := Key("StoryData/S1037B.json", "/dataList/12/content")
	if want := "StoryData/S1037B.json|/dataList/12/content"; key != want {
		t.Fatalf("Key = %q, want %q", key, want)
	}
	file, pointer, err := SplitKey(key)
	if err != nil {
		t.Fatal(err)
	}
	if file != "StoryData/S1037B.json" || pointer != "/dataList/12/content" {
		t.Fatalf("SplitKey = %q, %q", file, pointer)
	}
}

func TestSplitKey(t *testing.T) {
	cases := []struct{ file, pointer string }{
		{"Sample.json", "/dataList/0/name"},
		{"Sample.json", "/dataList/0/a|b"}, // a "|" inside the pointer
		{"Sample.json", "/a|b|c"},
		{"Sample.json", "/dataList/0/a~1b"},
		{"Sample.json", ""}, // whole document is a string
	}
	for _, c := range cases {
		file, pointer, err := SplitKey(Key(c.file, c.pointer))
		if err != nil {
			t.Errorf("SplitKey(Key(%q, %q)): %v", c.file, c.pointer, err)
			continue
		}
		if file != c.file || pointer != c.pointer {
			t.Errorf("SplitKey(Key(%q, %q)) = %q, %q", c.file, c.pointer, file, pointer)
		}
	}
}

func TestSplitKeyErrors(t *testing.T) {
	for _, key := range []string{
		"",
		"Sample.json",              // no separator
		"|/dataList/0/name",        // empty file
		"Sample.json|dataList/0/x", // pointer without leading "/"
		"Sample.json|/a~2b",        // invalid escape
	} {
		if file, pointer, err := SplitKey(key); err == nil {
			t.Errorf("SplitKey(%q) = %q, %q, want an error", key, file, pointer)
		}
	}
}

func TestKRSha(t *testing.T) {
	if KRSha("a\r\nb") != KRSha("a\nb") {
		t.Error(`KRSha("a\r\nb") != KRSha("a\nb")`)
	}
	sum := sha256.Sum256([]byte("a\nb"))
	if got, want := KRSha("a\r\nb"), hex.EncodeToString(sum[:]); got != want {
		t.Errorf("KRSha = %q, want %q", got, want)
	}
	if got, want := KRSha(""), "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"; got != want {
		t.Errorf(`KRSha("") = %q, want %q`, got, want)
	}
	// Only CRLF pairs are normalised; a lone CR is content.
	if KRSha("a\rb") == KRSha("a\nb") {
		t.Error("lone CR was normalised")
	}
	// Normalisation is a single pass: "\r\r\n" keeps its first CR.
	sum = sha256.Sum256([]byte("a\r\nb"))
	if got, want := KRSha("a\r\r\nb"), hex.EncodeToString(sum[:]); got != want {
		t.Errorf("KRSha with doubled CR = %q, want %q", got, want)
	}
}

func TestFieldName(t *testing.T) {
	cases := map[string]string{
		"/dataList/3/levelList/0/name": "name",
		"/dataList/12/content":         "content",
		"/dataList/3/levelList/0":      "levelList",
		"/dataList/3/tags/0/1":         "tags",
		"/dataList/3":                  "dataList",
		"/dataList/3/a~1b":             "a/b",
		"/dataList/3/a~0b/0":           "a~b",
		"/dataList/3/ID2":              "ID2",
		"/dataList/3/":                 "", // empty key
		"/0":                           "",
		"":                             "",
		"no-leading-slash":             "",
	}
	for pointer, want := range cases {
		if got := FieldName(pointer); got != want {
			t.Errorf("FieldName(%q) = %q, want %q", pointer, got, want)
		}
	}
}

func TestExtractContext(t *testing.T) {
	p, err := pack.Open("../../testdata/pack/kr", "KR_")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := p.Doc("StoryData/S001A.json")
	if err != nil {
		t.Fatal(err)
	}
	got := Extract("StoryData/S001A.json", doc)
	want := []Unit{
		{File: "StoryData/S001A.json", Pointer: "/dataList/0/place", Field: "place", KR: "남부 12구 외곽", ItemID: "10", Model: "", ItemIndex: 0},
		{File: "StoryData/S001A.json", Pointer: "/dataList/0/content", Field: "content", KR: "해가 저물고 있었다.", ItemID: "10", Model: "", ItemIndex: 0},
		{File: "StoryData/S001A.json", Pointer: "/dataList/1/model", Field: "model", KR: "이스마엘", ItemID: "11", Model: "이스마엘", ItemIndex: 1},
		{File: "StoryData/S001A.json", Pointer: "/dataList/1/content", Field: "content", KR: "선장, 오늘은 어디로 가는 거야?", ItemID: "11", Model: "이스마엘", ItemIndex: 1},
		{File: "StoryData/S001A.json", Pointer: "/dataList/2/model", Field: "model", KR: "돈키호테", ItemID: "12", Model: "돈키호테", ItemIndex: 2},
		{File: "StoryData/S001A.json", Pointer: "/dataList/2/content", Field: "content", KR: "그야 물론 모험이지!", ItemID: "12", Model: "돈키호테", ItemIndex: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Extract:\n got %+v\nwant %+v", got, want)
	}
}

func TestExtractNumericID(t *testing.T) {
	p, err := pack.Open("../../testdata/pack/kr", "KR_")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := p.Doc("Sample.json")
	if err != nil {
		t.Fatal(err)
	}
	got := Extract("Sample.json", doc)
	if len(got) != 4 {
		t.Fatalf("got %d units, want 4: %+v", len(got), got)
	}
	last := got[3]
	if last.Pointer != "/dataList/1/desc" || last.Field != "desc" || last.ItemID != "1020902" || last.ItemIndex != 1 || last.Model != "" {
		t.Errorf("last unit = %+v", last)
	}
}

func TestExtractNestedAndEdgeCases(t *testing.T) {
	doc := mustParse(t, `{
  "title": "top",
  "dataList": [
    {"id": "A-1", "model": "M", "levelList": [{"id": 5, "name": "deep", "n": 7, "ok": true, "none": null}], "empty": ""},
    {"name": "no id"},
    {"id": 3.50, "model": 9, "text": "numeric model"},
    "bare",
    ["x"]
  ]
}`)
	got := Extract("f.json", doc)
	type row struct {
		pointer, field, kr, id, model string
		index                         int
	}
	want := []row{
		{"/title", "title", "top", "", "", -1},
		{"/dataList/0/id", "id", "A-1", "A-1", "M", 0},
		{"/dataList/0/model", "model", "M", "A-1", "M", 0},
		{"/dataList/0/levelList/0/name", "name", "deep", "A-1", "M", 0},
		{"/dataList/0/empty", "empty", "", "A-1", "M", 0},
		{"/dataList/1/name", "name", "no id", "", "", 1},
		{"/dataList/2/text", "text", "numeric model", "3.50", "", 2},
		{"/dataList/3", "dataList", "bare", "", "", 3},
		{"/dataList/4/0", "dataList", "x", "", "", 4},
	}
	var rows []row
	for _, u := range got {
		if u.File != "f.json" {
			t.Errorf("File = %q", u.File)
		}
		rows = append(rows, row{u.Pointer, u.Field, u.KR, u.ItemID, u.Model, u.ItemIndex})
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("Extract:\n got %+v\nwant %+v", rows, want)
	}
}

func TestExtractWithoutDataList(t *testing.T) {
	for name, src := range map[string]string{
		"object without dataList": `{"a": "x", "b": {"c": "y"}}`,
		"array root":              `[{"id": 1, "name": "x"}]`,
		"dataList is an object":   `{"dataList": {"0": {"id": 1, "name": "x"}}}`,
		"dataList is a string":    `{"dataList": "x"}`,
	} {
		for _, u := range Extract("f.json", mustParse(t, src)) {
			if u.ItemIndex != -1 || u.ItemID != "" || u.Model != "" {
				t.Errorf("%s: unit %+v has item context", name, u)
			}
		}
	}
	if got := Extract("f.json", mustParse(t, `{"a": "x", "b": {"c": "y"}}`)); len(got) != 2 {
		t.Errorf("got %d units, want 2", len(got))
	}
	if got := Extract("f.json", mustParse(t, `{"n": 1}`)); len(got) != 0 {
		t.Errorf("got %d units, want 0", len(got))
	}
	if got := Extract("f.json", nil); len(got) != 0 {
		t.Errorf("Extract(nil) returned %d units", len(got))
	}
}
