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

func TestUnitWarning(t *testing.T) {
	a := UnitWarning("term-missing", "자판기", "음료 자판기", "販賣機")
	b := UnitWarning("term-missing", "음료", "음료 자판기", "販賣機")
	if a == b {
		t.Errorf("a unit that lacks two entries has one id for both: %s", a)
	}
	if want := "warning/term-missing/" + Sha8("음료 자판기\x00자판기") + "/" + Sha8("販賣機"); a != want {
		t.Errorf("UnitWarning = %q, want %q", a, want)
	}
	// Any other code is Warning, term or not: the ids people accepted stay.
	if got, want := UnitWarning("forbidden-term", "자판기", "자판기", "販賣機"), Warning("forbidden-term", "자판기", "販賣機"); got != want {
		t.Errorf("UnitWarning(forbidden-term) = %q, want %q", got, want)
	}
	if got, want := UnitWarning("kana", "", "가", "か"), Warning("kana", "가", "か"); got != want {
		t.Errorf("UnitWarning(kana) = %q, want %q", got, want)
	}
}
