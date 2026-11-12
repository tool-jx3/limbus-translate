package site

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/existing"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/suggest"
)

func TestExportNames(t *testing.T) {
	in, _ := exportReviewInputs(t)
	in.Names = &existing.Table{RawTag: "t1", Fields: []string{"name", "content"}, Names: []existing.Name{
		{KR: "격돌", Variants: []existing.Variant{{EN: "Clash", Renderings: []existing.Rendering{{ZH: "格鬥", Units: 1}}, Examples: []string{"a"}}}},
		{KR: "노을빛 검", Variants: []existing.Variant{
			{JP: "夕", Renderings: []existing.Rendering{{ZH: "暮光劍", Units: 2}, {ZH: "晚霞劍", Units: 1}}, Examples: []string{"b"}},
			{EN: "Sword", Renderings: []existing.Rendering{{ZH: "暮光之劍", Units: 1}}, Examples: []string{"c"}},
		}},
		{KR: "아지랑이", Variants: []existing.Variant{{Renderings: []existing.Rendering{{ZH: "遊絲", Units: 3}}, Examples: []string{"d"}}}},
	}}
	in.Rules.Glossary = append(in.Rules.Glossary, rules.Term{KR: "격돌", ZH: "衝突", Status: rules.StatusCandidate})
	in.Review.Declined = []suggest.Declined{{ID: existing.SuggestionID("아지랑이"), Reason: "x", Date: "2026-11-12"}}
	in.Review.Build.Report.NameDiffs = []build.NameDiff{{
		KR: "노을빛 검", Existing: []existing.Rendering{{ZH: "暮光劍", Units: 2}},
		Units: []build.NameDiffUnit{{Key: k(rvA, 0), ZH: "一", Kind: build.NameDiffField}},
	}}
	out := filepath.Join(t.TempDir(), "data")
	if _, err := Export(in, Meta{RawTag: "t1"}, out); err != nil {
		t.Fatal(err)
	}
	var got NamesData
	readJSON(t, out, "names.json", &got)
	if got.RawTag != "t1" || !reflect.DeepEqual(got.Fields, []string{"name", "content"}) || len(got.Names) != 3 {
		t.Fatalf("names.json = %+v", got)
	}
	want := []NameRow{
		{KR: "격돌", Units: 1, Variants: in.Names.Names[0].Variants, Glossary: true},
		{KR: "노을빛 검", Units: 4, Variants: in.Names.Names[1].Variants, Review: true},
		{KR: "아지랑이", Units: 3, Variants: in.Names.Names[2].Variants, Declined: true},
	}
	if !reflect.DeepEqual(got.Names, want) {
		t.Errorf("names = %+v\nwant %+v", got.Names, want)
	}
}

func TestExportNamesWithoutTable(t *testing.T) {
	in := exportInputs(t)
	out := filepath.Join(t.TempDir(), "data")
	if _, err := Export(in, Meta{}, out); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(out, "names.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(b)); got != `{"raw_tag":"","fields":[],"names":[]}` {
		t.Errorf("names.json = %s", got)
	}
}
