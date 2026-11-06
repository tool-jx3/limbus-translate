package batch

import "testing"

// EncodeFile is how every batch file is written: indented by two, with a final
// newline, <, > and & as they are; Go escapes U+2028 and U+2029 all the same.
func TestEncodeFile(t *testing.T) {
	got, err := EncodeFile(map[string]any{"b": "<a&b>", "a": []int{1}, "s": "x y"})
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"a\": [\n    1\n  ],\n  \"b\": \"<a&b>\",\n  \"s\": \"x\\u2028y\"\n}\n"
	if string(got) != want {
		t.Errorf("EncodeFile = %q, want %q", got, want)
	}
	if _, err := EncodeFile(make(chan int)); err == nil {
		t.Error("EncodeFile of a channel: no error")
	}
}
