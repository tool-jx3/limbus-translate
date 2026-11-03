package itemid

import (
	"strings"
	"testing"
)

func TestSha8(t *testing.T) {
	for text, want := range map[string]string{
		"도약 모듈":    "fd93b875",
		"KR 這句有雙關": "3713708f",
		"":         "e3b0c442", // sha256 of nothing
	} {
		if got := Sha8(text); got != want {
			t.Errorf("Sha8(%q) = %q, want %q", text, got, want)
		}
	}
}

func TestWarning(t *testing.T) {
	got := Warning("kana", "시 히스클리프", "し協會 希斯克利夫")
	if !strings.HasPrefix(got, "warning/kana/") || len(strings.Split(got, "/")) != 4 {
		t.Fatalf("Warning = %q, want warning/kana/<sha8>/<sha8>", got)
	}
	if want := "warning/kana/" + Sha8("시 히스클리프") + "/" + Sha8("し協會 希斯克利夫"); got != want {
		t.Errorf("Warning = %q, want %q", got, want)
	}
}

func TestInconsistent(t *testing.T) {
	if got, want := Inconsistent("ally", "name", "큰"), "inconsistent/ally/name/"+Sha8("큰"); got != want {
		t.Errorf("Inconsistent = %q, want %q", got, want)
	}
}

func TestNote(t *testing.T) {
	key := "A.json|/dataList/0/content"
	if got, want := Note(key, "x"), "note/"+key+"/"+Sha8("x"); got != want {
		t.Errorf("Note = %q, want %q", got, want)
	}
}
