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
