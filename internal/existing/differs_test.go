package existing

import (
	"strings"
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
	// Both variants match by EN, and neither by JP: a rendering of either is
	// accepted, and another is judged against the first in order.
	if is := ix.Differs("name", "노을빛 검", "夕照之劍", "", "", "Glow Blade"); is != nil {
		t.Errorf("both match, second variant's rendering: issue = %+v, want nil", is)
	}
	if is := ix.Differs("name", "노을빛 검", "霞光劍刃", "", "", "Glow Blade"); is != nil {
		t.Errorf("both match, first variant's rendering: issue = %+v, want nil", is)
	}
	if is := ix.Differs("name", "노을빛 검", "晚霞之劍", "", "", "Glow Blade"); is == nil || is.TermZH != "霞光劍刃" {
		t.Errorf("both match: issue = %+v, want one about the first variant", is)
	}
}

// regretIndex has one KR with two variants of one JP: the EN tells them
// apart, and the first has more units. A third variant matches neither.
func regretIndex() *Index {
	t := &Table{RawTag: "raw1", Fields: []string{"name"}, Names: []Name{{
		KR: "후회",
		Variants: []Variant{
			{JP: "後悔", EN: "Rue", Renderings: []Rendering{{"悔恨交加", 5}}},
			{JP: "後悔", EN: "Regret", Renderings: []Rendering{{"悔恨", 2}}},
			{JP: "未練", EN: "Lingering", Renderings: []Rendering{{"眷戀", 1}}},
		},
	}}}
	return NewIndex(t, &rules.Rules{}, nil)
}

// A unit whose JP and EN both match a later variant is judged against that
// variant, not against an earlier one that only its JP matches.
func TestDiffersPrefersTheVariantMatchingBoth(t *testing.T) {
	ix := regretIndex()
	for _, en := range []string{"Regret", "regret"} {
		is := ix.Differs("name", "후회", "遺憾", "", "後悔", en)
		if is == nil || is.Term != "후회" || is.TermZH != "悔恨" || !strings.Contains(is.Message, `"悔恨" (2 units)`) {
			t.Errorf("EN %q: issue = %+v, want one about 悔恨", en, is)
		}
	}
	if is := ix.Differs("name", "후회", "遺憾", "", "後悔", "Rue"); is == nil || is.TermZH != "悔恨交加" {
		t.Errorf("EN Rue: issue = %+v, want one about 悔恨交加", is)
	}
}

// A field that only one side has does not count against a variant: with the
// JP of one variant and the EN of another, the one with no JP whose EN
// matches is a better match than the one whose EN differs.
func TestDiffersPrefersAVariantNothingContradicts(t *testing.T) {
	tbl := &Table{RawTag: "raw1", Fields: []string{"name"}, Names: []Name{{
		KR: "견제",
		Variants: []Variant{
			{JP: "牽制", EN: "Obstruction", Renderings: []Rendering{{"牽制", 4}}},
			{EN: "Careful Obstruction", Renderings: []Rendering{{"牽制戰術", 1}}},
		},
	}}}
	ix := NewIndex(tbl, &rules.Rules{}, nil)
	if is := ix.Differs("name", "견제", "謹慎牽制", "", "牽制", "Careful Obstruction"); is == nil || is.TermZH != "牽制戰術" {
		t.Errorf("issue = %+v, want one about 牽制戰術", is)
	}
}

// A translation that is a rendering of any variant the JP or the EN points at
// is accepted, though another variant matches better.
func TestDiffersAcceptsARenderingOfAnyMatchingVariant(t *testing.T) {
	ix := regretIndex()
	if is := ix.Differs("name", "후회", "悔恨交加", "", "後悔", "Regret"); is != nil {
		t.Errorf("a rendering of the variant the JP points at: issue = %+v, want nil", is)
	}
	if is := ix.Differs("name", "후회", "悔恨交加", "", "", "Regret"); is == nil || is.TermZH != "悔恨" {
		t.Errorf("only the EN, of the other variant: issue = %+v, want one about 悔恨", is)
	}
	if is := ix.Differs("name", "후회", "眷戀", "", "後悔", "Regret"); is == nil || is.TermZH != "悔恨" {
		t.Errorf("a rendering of a variant neither points at: issue = %+v, want one about 悔恨", is)
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
