package existing

import (
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/rules"
)

func differsIndex(r *rules.Rules, declined map[string]bool) *Index {
	t := &Table{RawTag: "raw1", Fields: []string{"name"}, Names: []Name{{
		KR: "노을빛 검",
		Variants: []Variant{{
			JP: "茜さす剣", EN: "Glow Blade",
			Renderings: []Rendering{{"霞光劍刃", 3}, {"霞光之劍", 1}},
		}},
	}}}
	return NewIndex(t, r, declined)
}

func TestDiffersNeedsMatchingJPOrEN(t *testing.T) {
	ix := differsIndex(&rules.Rules{}, nil)
	is := ix.Differs("name", "노을빛 검", "晚霞之劍", "", "茜さす剣", "")
	if is == nil || is.Code != CodeNameDiffers || is.Term != "노을빛 검" || is.TermZH != "霞光劍刃" {
		t.Fatalf("issue = %+v, want name-differs about 霞光劍刃", is)
	}
	if is := ix.Differs("name", "노을빛 검", "晚霞之劍", "", "夕焼け", "Sunset"); is != nil {
		t.Errorf("other JP and EN: issue = %+v, want nil", is)
	}
	if is := ix.Differs("name", "노을빛 검", "晚霞之劍", "", "", "glow blade"); is == nil {
		t.Error("EN matches ignoring case: want an issue")
	}
}

func TestDiffersAcceptsAnyExistingRendering(t *testing.T) {
	ix := differsIndex(&rules.Rules{}, nil)
	for _, zh := range []string{"霞光之劍", "<b>霞光劍刃</b>", " 霞光劍刃 "} {
		if is := ix.Differs("name", "노을빛 검", zh, "", "茜さす剣", ""); is != nil {
			t.Errorf("zh %q: issue = %+v, want nil", zh, is)
		}
	}
}

func TestDiffersNoteNamesKR(t *testing.T) {
	ix := differsIndex(&rules.Rules{}, nil)
	if is := ix.Differs("name", "노을빛 검", "晚霞之劍", "노을빛 검：此處是劇情描寫", "茜さす剣", ""); is != nil {
		t.Errorf("issue = %+v, want nil", is)
	}
}

func TestDiffersOnlyNameFields(t *testing.T) {
	ix := differsIndex(&rules.Rules{}, nil)
	if is := ix.Differs("desc", "노을빛 검", "晚霞之劍", "", "茜さす剣", ""); is != nil {
		t.Errorf("desc field: issue = %+v, want nil", is)
	}
	if is := ix.Differs("name", "다른 이름", "晚霞之劍", "", "茜さす剣", ""); is != nil {
		t.Errorf("unknown KR: issue = %+v, want nil", is)
	}
}

func TestDiffersDeclinedOrGlossary(t *testing.T) {
	ix := differsIndex(&rules.Rules{}, map[string]bool{"name/노을빛 검": true})
	if is := ix.Differs("name", "노을빛 검", "晚霞之劍", "", "茜さす剣", ""); is != nil {
		t.Errorf("declined: issue = %+v, want nil", is)
	}
	r := &rules.Rules{Glossary: []rules.Term{{KR: "노을빛 검", ZH: "霞光", Status: "candidate"}}}
	if is := differsIndex(r, nil).Differs("name", "노을빛 검", "晚霞之劍", "", "茜さす剣", ""); is != nil {
		t.Errorf("glossary: issue = %+v, want nil", is)
	}
}

func TestDiffersWithoutJPEN(t *testing.T) {
	ix := differsIndex(&rules.Rules{}, nil)
	if is := ix.Differs("name", "노을빛 검", "晚霞之劍", "", "", ""); is != nil {
		t.Errorf("issue = %+v, want nil", is)
	}
	var nilIx *Index
	if is := nilIx.Differs("name", "노을빛 검", "晚霞之劍", "", "茜さす剣", ""); is != nil {
		t.Errorf("nil index: issue = %+v, want nil", is)
	}
}

func twoVariantIndex() *Index {
	t := &Table{RawTag: "raw1", Fields: []string{"name"}, Names: []Name{{
		KR: "노을빛 검",
		Variants: []Variant{
			{JP: "茜さす剣", EN: "Glow Blade", Renderings: []Rendering{{"霞光劍刃", 3}}},
			{JP: "夕焼けの剣", EN: "Glow Blade", Renderings: []Rendering{{"夕照之劍", 2}}},
		},
	}}}
	return NewIndex(t, &rules.Rules{}, nil)
}

func TestDiffersJudgesAgainstMatchedVariant(t *testing.T) {
	ix := twoVariantIndex()
	if is := ix.Differs("name", "노을빛 검", "霞光劍刃", "", "夕焼けの剣", ""); is == nil || is.TermZH != "夕照之劍" {
		t.Errorf("rendering of the first variant against the second: issue = %+v, want one about 夕照之劍", is)
	}
	if is := ix.Differs("name", "노을빛 검", "夕照之劍", "", "夕焼けの剣", ""); is != nil {
		t.Errorf("rendering of the second variant: issue = %+v, want nil", is)
	}
	// Both variants match by EN: the first in order wins.
	if is := ix.Differs("name", "노을빛 검", "夕照之劍", "", "", "Glow Blade"); is == nil || is.TermZH != "霞光劍刃" {
		t.Errorf("both match: issue = %+v, want one about the first variant", is)
	}
	if is := ix.Differs("name", "노을빛 검", "霞光劍刃", "", "", "Glow Blade"); is != nil {
		t.Errorf("both match, first variant's rendering: issue = %+v, want nil", is)
	}
}

func TestDiffersMessage(t *testing.T) {
	is := twoVariantIndex().Differs("name", "노을빛 검", "<b>晚霞之劍</b>", "", "茜さす剣", "")
	if is == nil {
		t.Fatal("want an issue")
	}
	want := `the existing translations name "노을빛 검" "霞光劍刃" (3 units); this one says "晚霞之劍": use the existing name when it is the same thing, or write the KR word and the reason in the note`
	if is.Message != want {
		t.Errorf("Message = %q, want %q", is.Message, want)
	}
}
