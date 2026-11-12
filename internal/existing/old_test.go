package existing

import (
	"fmt"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
)

func provisional(kr, zh string) rules.Term {
	return rules.Term{KR: kr, ZH: zh, Match: rules.MatchWord, Status: rules.StatusApproved, Provisional: true}
}

func TestOldTallyFlagsDifferent(t *testing.T) {
	tally := NewOldTally([]rules.Term{provisional("도발치", "嘲諷值")})
	for _, i := range []int{3, 1, 5, 0, 4, 2} {
		tally.Add(fmt.Sprintf("F.json|/%d", i), "도발치가 쌓였다", "挑釁值累積了")
	}
	use, examples, differs := tally.Use(nil, "도발치", "嘲諷值")
	if !differs || use.Units != 6 || use.With != 0 {
		t.Fatalf("use = %+v, differs = %v, want 6 units, 0 with, differs", use, differs)
	}
	if len(examples) != 3 || examples[0].Key != "F.json|/0" || examples[1].Key != "F.json|/1" || examples[2].Key != "F.json|/2" {
		t.Errorf("examples = %+v, want the first three by key", examples)
	}
	if examples[0].KR != "도발치가 쌓였다" || examples[0].ZH != "挑釁值累積了" {
		t.Errorf("example = %+v", examples[0])
	}
}

func TestOldTallyAgrees(t *testing.T) {
	tally := NewOldTally([]rules.Term{provisional("도발치", "嘲諷值")})
	for i := 0; i < 6; i++ {
		zh := "挑釁值"
		if i < 4 {
			zh = "嘲諷值"
		}
		tally.Add(fmt.Sprintf("F.json|/%d", i), "도발치", zh)
	}
	use, examples, differs := tally.Use(nil, "도발치", "嘲諷值")
	if differs || use.Units != 6 || use.With != 4 {
		t.Errorf("use = %+v, differs = %v, want no difference", use, differs)
	}
	if len(examples) == 0 || examples[0].ZH != "挑釁值" {
		t.Errorf("examples = %+v, want the ones without the term first", examples)
	}
}

func TestOldTallyTooFew(t *testing.T) {
	tally := NewOldTally([]rules.Term{provisional("도발치", "嘲諷值")})
	for i := 0; i < 4; i++ {
		tally.Add(fmt.Sprintf("F.json|/%d", i), "도발치", "挑釁值")
	}
	if use, _, differs := tally.Use(nil, "도발치", "嘲諷值"); differs || use.Units != 4 {
		t.Errorf("use = %+v, differs = %v, want 4 units and no difference", use, differs)
	}
}

func TestOldTallyUsesNameTable(t *testing.T) {
	ix := NewIndex(&Table{RawTag: "raw1", Fields: []string{"name"}, Names: []Name{
		{KR: "격돌", Variants: []Variant{{Renderings: []Rendering{{"交鋒", 3}}}}},
	}}, nil, nil)
	tally := NewOldTally([]rules.Term{provisional("격돌", "衝突"), provisional("격돌", "交鋒")})
	tally.Add("F.json|/0", "격돌", "交鋒")
	use, _, differs := tally.Use(ix, "격돌", "衝突")
	if !differs || use.Existing != "交鋒" || use.Units != 1 || use.With != 0 {
		t.Errorf("use = %+v, differs = %v, want differs with Existing 交鋒", use, differs)
	}
	if use, _, differs := tally.Use(ix, "격돌", "交鋒"); differs || use.Existing != "交鋒" {
		t.Errorf("use = %+v, differs = %v, want no difference", use, differs)
	}
}

// A name whose suggestion was declined does not decide: a person kept the
// new rendering, so the term is weighed against the old translations as one
// the table lacks.
func TestOldTallyIgnoresADeclinedName(t *testing.T) {
	ix := NewIndex(&Table{RawTag: "raw1", Fields: []string{"name"}, Names: []Name{
		{KR: "격돌", Variants: []Variant{{Renderings: []Rendering{{"交鋒", 3}}}}},
	}}, nil, map[string]bool{SuggestionID("격돌"): true})
	few := NewOldTally([]rules.Term{provisional("격돌", "衝突")})
	few.Add("F.json|/0", "격돌", "交鋒")
	if use, _, differs := few.Use(ix, "격돌", "衝突"); differs || use.Existing != "" || use.Units != 1 {
		t.Errorf("one old unit: use = %+v, differs = %v, want no difference and no Existing", use, differs)
	}
	many := NewOldTally([]rules.Term{provisional("격돌", "衝突")})
	for i := range MinOldUnits {
		many.Add(fmt.Sprintf("F.json|/%d", i), "격돌", "交鋒")
	}
	if use, _, differs := many.Use(ix, "격돌", "衝突"); !differs || use.Existing != "" || use.Units != MinOldUnits || use.With != 0 {
		t.Errorf("%d old units without it: use = %+v, differs = %v, want a difference by the tally", MinOldUnits, use, differs)
	}
}

func TestOldTallyWordMatch(t *testing.T) {
	tally := NewOldTally([]rules.Term{provisional("불주먹 사무소", "火拳事務所")})
	tally.Add("F.json|/0", "큰불주먹 사무소", "大火拳事務所")
	tally.Add("F.json|/1", "불주먹 사무소에서", "在火拳事務所")
	use, _, _ := tally.Use(nil, "불주먹 사무소", "火拳事務所")
	if use.Units != 1 || use.With != 1 {
		t.Errorf("use = %+v, want only the second unit counted", use)
	}
}

func TestIsOld(t *testing.T) {
	const tag = "raw2"
	cases := []struct {
		name string
		rec  store.Record
		want bool
	}{
		{"paratranz", store.Record{Origin: store.OriginParatranz}, true},
		{"human", store.Record{Origin: store.OriginHuman}, true},
		{"claude of another tag", store.Record{Origin: store.OriginClaude, Written: "raw1"}, true},
		{"claude of this tag", store.Record{Origin: store.OriginClaude, Written: tag}, false},
		{"copy", store.Record{Origin: store.OriginCopy}, false},
		{"fallback", store.Record{Origin: store.OriginFallback}, false},
	}
	for _, c := range cases {
		if got := IsOld(c.rec, tag); got != c.want {
			t.Errorf("%s: IsOld = %v, want %v", c.name, got, c.want)
		}
	}
}
