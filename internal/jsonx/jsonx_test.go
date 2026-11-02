package jsonx

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func mustParse(t *testing.T, s string) *Node {
	t.Helper()
	n, err := Parse([]byte(s))
	if err != nil {
		t.Fatalf("Parse(%q): %v", s, err)
	}
	return n
}

// uEsc returns the six-character JSON escape for the given 4 hex digits,
// built at run time so the test source holds no literal escape sequences.
func uEsc(hex string) string { return string(rune(92)) + "u" + hex }

func TestRoundTripEscapes(t *testing.T) {
	const want = "a\nb\"c\\d 😀 <color=#ff0000>X</color> &"
	in := &Node{Kind: Object, Members: []Member{
		{Key: "v", Value: &Node{Kind: String, Str: want}},
	}}
	out := Marshal(in)
	if !bytes.Contains(out, []byte("<color=#ff0000>")) || !bytes.Contains(out, []byte("&")) {
		t.Fatalf("marshal escaped HTML characters: %s", out)
	}
	if bytes.Contains(out, []byte(uEsc("003c"))) || bytes.Contains(out, []byte(uEsc("0026"))) || bytes.Contains(out, []byte(uEsc("003e"))) {
		t.Fatalf("marshal contains HTML escapes: %s", out)
	}
	back := mustParse(t, string(out))
	if got := back.Members[0].Value.Str; got != want {
		t.Fatalf("round trip Str = %q, want %q", got, want)
	}
	// Parsing escaped source text also yields the same value.
	src := `{"v":"a\nb\"c\\d ` + uEsc("d83d") + uEsc("de00") + ` ` + uEsc("003c") + `color=#ff0000` + uEsc("003e") + `X` + uEsc("003c") + `/color` + uEsc("003e") + ` ` + uEsc("0026") + `"}`
	if got := mustParse(t, src).Members[0].Value.Str; got != want {
		t.Fatalf("parse escaped Str = %q, want %q", got, want)
	}
}

func TestKeyOrderAndNumbers(t *testing.T) {
	out := string(Marshal(mustParse(t, `{"z":1,"a":1020901,"f":0.10,"e":1e5}`)))
	want := "{\n  \"z\": 1,\n  \"a\": 1020901,\n  \"f\": 0.10,\n  \"e\": 1e5\n}\n"
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestBOM(t *testing.T) {
	n, err := Parse([]byte("\xEF\xBB\xBF{\"a\":\"b\"}"))
	if err != nil {
		t.Fatalf("Parse with BOM: %v", err)
	}
	if n.Members[0].Key != "a" || n.Members[0].Value.Str != "b" {
		t.Fatalf("unexpected tree: %+v", n)
	}
	if out := Marshal(n); bytes.HasPrefix(out, []byte("\xEF\xBB\xBF")) || bytes.Contains(out, []byte("\xEF\xBB\xBF")) {
		t.Fatalf("marshal output contains BOM: %q", out)
	}
}

func TestPointerEscaping(t *testing.T) {
	const key = "a/b~c "
	n := mustParse(t, `{"a/b~c ":"v","other":"w"}`)
	fields := n.Strings()
	if len(fields) != 2 || fields[0].Pointer != "/a~1b~0c " || fields[0].Value != "v" {
		t.Fatalf("Strings() = %+v", fields)
	}
	if got := "/" + EscapeToken(key); got != "/a~1b~0c " {
		t.Fatalf("EscapeToken = %q", got)
	}
	got, ok := n.Get("/a~1b~0c ")
	if !ok || got != n.Members[0].Value {
		t.Fatalf("Get did not return the member node: %v %v", got, ok)
	}
	if err := n.SetString("/a~1b~0c ", "new"); err != nil {
		t.Fatalf("SetString: %v", err)
	}
	if n.Members[0].Value.Str != "new" || n.Members[1].Value.Str != "w" {
		t.Fatalf("SetString wrote wrong node: %+v", n)
	}
}

func TestStringsOrder(t *testing.T) {
	n := mustParse(t, `{"dataList":[{"id":1,"name":"N","desc":"D"}]}`)
	want := []StringField{
		{Pointer: "/dataList/0/name", Value: "N"},
		{Pointer: "/dataList/0/desc", Value: "D"},
	}
	if got := n.Strings(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Strings() = %+v, want %+v", got, want)
	}
}

func TestStringsSkipsNonStringLeaves(t *testing.T) {
	n := mustParse(t, `["s",1,true,false,null,"t",[ "u" ]]`)
	want := []StringField{
		{Pointer: "/0", Value: "s"},
		{Pointer: "/5", Value: "t"},
		{Pointer: "/6/0", Value: "u"},
	}
	if got := n.Strings(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Strings() = %+v, want %+v", got, want)
	}
	root := mustParse(t, `"only"`)
	if got := root.Strings(); !reflect.DeepEqual(got, []StringField{{Pointer: "", Value: "only"}}) {
		t.Fatalf("root string Strings() = %+v", got)
	}
}

func TestEmptyStructures(t *testing.T) {
	for _, src := range []string{`{}`, `[]`, `{"dataList":[]}`, `[{}]`} {
		n := mustParse(t, src)
		if got := n.Strings(); len(got) != 0 {
			t.Errorf("%s: Strings() = %+v, want empty", src, got)
		}
		out := Marshal(n)
		again := mustParse(t, string(out))
		if !reflect.DeepEqual(n, again) {
			t.Errorf("%s: structure changed after marshal %q", src, out)
		}
	}
	if got := string(Marshal(mustParse(t, `{}`))); got != "{}\n" {
		t.Errorf("empty object marshals to %q", got)
	}
	if got := string(Marshal(mustParse(t, `[]`))); got != "[]\n" {
		t.Errorf("empty array marshals to %q", got)
	}
	got := string(Marshal(mustParse(t, `{"dataList":[],"o":{}}`)))
	if want := "{\n  \"dataList\": [],\n  \"o\": {}\n}\n"; got != want {
		t.Errorf("nested empties: got %q want %q", got, want)
	}
}

func TestDeterministic(t *testing.T) {
	n := mustParse(t, `{"dataList":[{"id":1,"name":"N <b>&","desc":["x","y"]},{"id":2.50}],"k":null,"t":true}`)
	a, b := Marshal(n), Marshal(n)
	if !bytes.Equal(a, b) {
		t.Fatalf("Marshal not deterministic:\n%s\n%s", a, b)
	}
	if !bytes.HasSuffix(a, []byte("\n")) || bytes.HasSuffix(a, []byte("\n\n")) {
		t.Fatalf("output must end with exactly one newline: %q", a)
	}
	if bytes.Contains(a, []byte("\r")) {
		t.Fatalf("output contains CR: %q", a)
	}
}

func TestMarshalLayout(t *testing.T) {
	n := mustParse(t, `{"a":[1,true,null,{"b":"x"}],"c":{},"d":[],"e":"`+uEsc("00e9")+`\t"}`)
	want := "{\n" +
		"  \"a\": [\n" +
		"    1,\n" +
		"    true,\n" +
		"    null,\n" +
		"    {\n" +
		"      \"b\": \"x\"\n" +
		"    }\n" +
		"  ],\n" +
		"  \"c\": {},\n" +
		"  \"d\": [],\n" +
		"  \"e\": \"é\\t\"\n" +
		"}\n"
	if got := string(Marshal(n)); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestDuplicateAndTrailingSpaceKeys(t *testing.T) {
	n := mustParse(t, `{"debugNodeId ":"x","k":"first","k":"second"}`)
	if len(n.Members) != 3 || n.Members[1].Value.Str != "first" || n.Members[2].Value.Str != "second" {
		t.Fatalf("duplicates not preserved: %+v", n.Members)
	}
	got, ok := n.Get("/k")
	if !ok || got.Str != "first" {
		t.Fatalf("Get(/k) = %v %v, want first member", got, ok)
	}
	if err := n.SetString("/k", "changed"); err != nil {
		t.Fatal(err)
	}
	if n.Members[1].Value.Str != "changed" || n.Members[2].Value.Str != "second" {
		t.Fatalf("SetString must target first duplicate: %+v", n.Members)
	}
	if got, ok := n.Get("/debugNodeId "); !ok || got.Str != "x" {
		t.Fatalf("trailing-space key lookup failed: %v %v", got, ok)
	}
	if _, ok := n.Get("/debugNodeId"); ok {
		t.Fatal("key lookup must be exact (no trimming)")
	}
	want := "{\n  \"debugNodeId \": \"x\",\n  \"k\": \"changed\",\n  \"k\": \"second\"\n}\n"
	if got := string(Marshal(n)); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	fields := n.Strings()
	if len(fields) != 3 || fields[1].Pointer != "/k" || fields[2].Pointer != "/k" {
		t.Fatalf("Strings() must list duplicates in order: %+v", fields)
	}
}

func TestGetArrayAndRoot(t *testing.T) {
	n := mustParse(t, `{"l":[{"n":"a"},{"n":"b"}]}`)
	if got, ok := n.Get(""); !ok || got != n {
		t.Fatal("empty pointer must return the root")
	}
	if got, ok := n.Get("/l/1/n"); !ok || got.Str != "b" {
		t.Fatalf("Get(/l/1/n) = %v %v", got, ok)
	}
	for _, p := range []string{"/l/2", "/l/-1", "/l/01", "/l/x", "/l/1/n/z", "/missing", "l", "/l/1/n~2"} {
		if got, ok := n.Get(p); ok {
			t.Errorf("Get(%q) = %v, want not found", p, got)
		}
	}
}

func TestSetStringErrors(t *testing.T) {
	n := mustParse(t, `{"s":"v","num":1,"o":{}}`)
	for _, p := range []string{"/missing", "/num", "/o", "", "bad"} {
		if err := n.SetString(p, "x"); err == nil {
			t.Errorf("SetString(%q) succeeded, want error", p)
		}
	}
	if n.Members[0].Value.Str != "v" || n.Members[1].Value.Lit != "1" {
		t.Fatalf("failed SetString changed the tree: %+v", n)
	}
}

func TestClone(t *testing.T) {
	n := mustParse(t, `{"a":["x",{"b":"y"}],"n":1.0}`)
	c := n.Clone()
	if !reflect.DeepEqual(n, c) {
		t.Fatal("clone differs from original")
	}
	if err := c.SetString("/a/1/b", "changed"); err != nil {
		t.Fatal(err)
	}
	c.Members[0].Value.Items[0].Str = "changed"
	c.Members[1].Value.Lit = "2"
	if got, _ := n.Get("/a/1/b"); got.Str != "y" {
		t.Fatalf("original mutated: %q", got.Str)
	}
	if n.Members[0].Value.Items[0].Str != "x" || n.Members[1].Value.Lit != "1.0" {
		t.Fatal("original mutated through clone")
	}
	var nilNode *Node
	if nilNode.Clone() != nil {
		t.Fatal("Clone of nil must be nil")
	}
}

func TestParseErrors(t *testing.T) {
	for _, src := range []string{``, "\xEF\xBB\xBF", `{`, `{"a":}`, `{"a":1,}`, `[1,2`, `{"a":1} x`, `{} {}`, `nope`, `{"a" 1}`} {
		if n, err := Parse([]byte(src)); err == nil {
			t.Errorf("Parse(%q) = %+v, want error", src, n)
		}
	}
}

func TestScalarRoots(t *testing.T) {
	for _, src := range []string{`1`, `0.10`, `true`, `false`, `null`, `"s"`} {
		n := mustParse(t, src)
		if got := string(Marshal(n)); got != src+"\n" {
			t.Errorf("Marshal(%s) = %q", src, got)
		}
	}
}

func TestSplitPointerAndEscapeToken(t *testing.T) {
	cases := []struct {
		p    string
		want []string
	}{
		{"", nil},
		{"/", []string{""}},
		{"/a/b", []string{"a", "b"}},
		{"/a~1b/c~0d", []string{"a/b", "c~d"}},
		{"/~01", []string{"~1"}},
		{"//x", []string{"", "x"}},
	}
	for _, c := range cases {
		got, err := SplitPointer(c.p)
		if err != nil {
			t.Errorf("SplitPointer(%q): %v", c.p, err)
			continue
		}
		if len(got) != len(c.want) || (len(got) > 0 && !reflect.DeepEqual(got, c.want)) {
			t.Errorf("SplitPointer(%q) = %#v, want %#v", c.p, got, c.want)
		}
	}
	for _, p := range []string{"a", "a/b", "/a~", "/a~2"} {
		if _, err := SplitPointer(p); err == nil {
			t.Errorf("SplitPointer(%q) succeeded, want error", p)
		}
	}
	for _, tok := range []string{"", "plain", "a/b", "a~b", "~1", "/~", "a/b~c "} {
		back, err := SplitPointer("/" + EscapeToken(tok))
		if err != nil || len(back) != 1 || back[0] != tok {
			t.Errorf("token %q did not round trip: %v %v", tok, back, err)
		}
	}
	if strings.Contains(EscapeToken("a/b"), "/") {
		t.Error("EscapeToken left a slash")
	}
}

// A node built by hand has no source text: an empty Null or Bool must still be
// written as valid JSON.
func TestEncodeManualNullAndBool(t *testing.T) {
	root := &Node{Kind: Object, Members: []Member{
		{Key: "n", Value: &Node{Kind: Null}},
		{Key: "b", Value: &Node{Kind: Bool}},
		{Key: "t", Value: &Node{Kind: Bool, Lit: "true"}},
	}}
	out := Marshal(root)
	back, err := Parse(out)
	if err != nil {
		t.Fatalf("Marshal wrote invalid JSON: %v\n%s", err, out)
	}
	if got, want := string(Marshal(back)), "{\n  \"n\": null,\n  \"b\": false,\n  \"t\": true\n}\n"; got != want {
		t.Errorf("Marshal =\n%s\nwant\n%s", got, want)
	}
	for kind, n := range map[string]*Node{"null": {Kind: Null}, "bool": {Kind: Bool}} {
		if _, err := Parse(Marshal(n)); err != nil {
			t.Errorf("a bare %s node is not valid JSON: %v", kind, err)
		}
	}
}
