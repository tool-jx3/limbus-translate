package evidence

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/rules"
)

// packFiles maps a file name (without language prefix) to its content, per
// language.
type packFiles map[string]string

func openSet(t *testing.T, kr, en, jp packFiles) pack.Set {
	t.Helper()
	root := t.TempDir()
	open := func(dir, prefix string, files packFiles) *pack.Pack {
		for name, content := range files {
			full := filepath.Join(root, dir, filepath.Dir(name), prefix+filepath.Base(name))
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
		p, err := pack.Open(filepath.Join(root, dir), prefix)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	return pack.Set{KR: open("kr", "KR_", kr), EN: open("en", "EN_", en), JP: open("jp", "JP_", jp)}
}

func glossary(terms ...rules.Term) *rules.Rules { return &rules.Rules{Glossary: terms} }

func term(kr, zh string) rules.Term {
	return rules.Term{KR: kr, ZH: zh, Status: rules.StatusApproved}
}

// story builds a file of n dataList lines, line i having the KR text text(i).
func lines(n int, f func(i int) string) string {
	var b strings.Builder
	b.WriteString(`{"dataList":[`)
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, `{"id":%d,"content":%q}`, i+1, f(i))
	}
	b.WriteString(`]}`)
	return b.String()
}

func TestCollectNamesAggregateByPair(t *testing.T) {
	set := openSet(t,
		packFiles{
			"Bufs.json":   `{"dataList":[{"id":"Burst","name":"파열 ","desc":"파열 설명"},{"id":"Other","name":"다른"}]}`,
			"Skills.json": `{"dataList":[{"id":1,"name":"파열"},{"id":2,"name":"파열"}]}`,
			"Misc.json":   `{"dataList":[{"id":1,"text":"파열이 일어났다"}]}`,
		},
		packFiles{
			"Bufs.json":   `{"dataList":[{"id":"Burst","name":"Rupture","desc":"Rupture desc"},{"id":"Other","name":"x"}]}`,
			"Skills.json": `{"dataList":[{"id":1,"name":"Rupture"},{"id":2,"name":"Burst"}]}`,
			"Misc.json":   `{"dataList":[{"id":1,"text":"A rupture happened"}]}`,
		},
		packFiles{
			"Bufs.json":   `{"dataList":[{"id":"Burst","name":"破裂","desc":"破裂説明"},{"id":"Other","name":"x"}]}`,
			"Skills.json": `{"dataList":[{"id":1,"name":"破裂"},{"id":2,"name":"炸裂"}]}`,
			"Misc.json":   `{"dataList":[{"id":1,"text":"破裂した"}]}`,
		})
	items, err := Collect(glossary(term("파열", "破裂")), set, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	it := items[0]
	wantNames := []Name{
		{JP: "破裂", EN: "Rupture", Keys: []string{"Bufs.json|/dataList/0/name", "Skills.json|/dataList/0/name"}, Count: 2},
		{JP: "炸裂", EN: "Burst", Keys: []string{"Skills.json|/dataList/1/name"}, Count: 1},
	}
	if !reflect.DeepEqual(it.Names, wantNames) {
		t.Errorf("Names = %+v, want %+v", it.Names, wantNames)
	}
	// Only the keyword file's name has a desc; the Skills.json names have none.
	wantDescs := []Desc{{Key: "Bufs.json|/dataList/0/desc", KR: "파열 설명", JP: "破裂説明", EN: "Rupture desc"}}
	if !reflect.DeepEqual(it.Descs, wantDescs) {
		t.Errorf("Descs = %+v, want %+v", it.Descs, wantDescs)
	}
	if it.KR != "파열" || it.ZH != "破裂" {
		t.Errorf("Item = %q/%q", it.KR, it.ZH)
	}
}

func TestCollectChecksThreshold(t *testing.T) {
	// Each case: n units hold the term, the first hits of them have EN text
	// hitText, the rest "Miss".
	tests := []struct {
		name     string
		n, hits  int
		hitText  string
		cand     string
		wantHits int
		wantOK   bool
	}{
		{"nine of nine", 9, 9, "Hit", "Hit", 9, false},
		{"six of ten", 10, 6, "Hit", "Hit", 6, true},
		{"five of ten", 10, 5, "Hit", "Hit", 5, false},
		{"case-insensitive", 10, 10, "A HIT here", "hit", 10, true},
		{"word boundary", 10, 10, "Hitting it", "Hit", 0, false},
		{"word boundary, later occurrence", 10, 10, "Hitting. Hit!", "Hit", 10, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			kr := lines(tc.n, func(int) string { return "타격 있음" })
			en := lines(tc.n, func(i int) string {
				if i < tc.hits {
					return tc.hitText
				}
				return "Miss"
			})
			set := openSet(t, packFiles{"Lines.json": kr}, packFiles{"Lines.json": en}, packFiles{})
			items, err := Collect(glossary(term("타격", "打擊")), set, nil, map[string]Candidates{"타격": {EN: []string{tc.cand}}})
			if err != nil {
				t.Fatal(err)
			}
			var got *Check
			for i, c := range items[0].Checks {
				if c.Lang == "EN" && c.Candidate == tc.cand {
					got = &items[0].Checks[i]
				}
			}
			if got == nil {
				t.Fatalf("no EN check for %q in %+v", tc.cand, items[0].Checks)
			}
			if got.N != tc.n || got.Hits != tc.wantHits || got.OK != tc.wantOK {
				t.Errorf("check = %+v, want N %d Hits %d OK %v", *got, tc.n, tc.wantHits, tc.wantOK)
			}
		})
	}
}

func TestCollectCheckJPSubstringAndNoAlignment(t *testing.T) {
	kr := lines(10, func(int) string { return "타격 있음" })
	jp := lines(10, func(i int) string { return "強い打撃だ" })
	// the EN pack has no such file: N is 0 for EN.
	set := openSet(t, packFiles{"Lines.json": kr}, packFiles{}, packFiles{"Lines.json": jp})
	items, err := Collect(glossary(term("타격", "打擊")), set, nil, map[string]Candidates{"타격": {JP: []string{"打撃"}, EN: []string{"Hit"}}})
	if err != nil {
		t.Fatal(err)
	}
	want := []Check{
		{Lang: "JP", Candidate: "打撃", N: 10, Hits: 10, Rate: 1, OK: true},
		{Lang: "EN", Candidate: "Hit", N: 0, Hits: 0, Rate: 0, OK: false},
	}
	if !reflect.DeepEqual(items[0].Checks, want) {
		t.Errorf("Checks = %+v, want %+v", items[0].Checks, want)
	}
}

// The candidates are the entry's own jp and en, the most frequent official
// names, and the ones given, without repeats.
func TestCollectCandidateUnion(t *testing.T) {
	set := openSet(t,
		packFiles{"Skills.json": `{"dataList":[{"id":1,"name":"타격"},{"id":2,"name":"타격"},{"id":3,"name":"타격"}]}`},
		packFiles{"Skills.json": `{"dataList":[{"id":1,"name":"Hit"},{"id":2,"name":"Hit"},{"id":3,"name":"Strike"}]}`},
		packFiles{"Skills.json": `{"dataList":[{"id":1,"name":"打撃"},{"id":2,"name":"打撃"},{"id":3,"name":"打撃"}]}`})
	e := term("타격", "打擊")
	e.JP, e.EN = "打撃", "Blow"
	items, err := Collect(glossary(e), set, []string{"타격"}, map[string]Candidates{"타격": {EN: []string{"Blow", "Smash"}}})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range items[0].Checks {
		got = append(got, c.Lang+":"+c.Candidate)
	}
	want := []string{"JP:打撃", "EN:Blow", "EN:Hit", "EN:Smash"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("candidates = %v, want %v", got, want)
	}
	if items[0].JP != "打撃" || items[0].EN != "Blow" {
		t.Errorf("Item JP/EN = %q/%q", items[0].JP, items[0].EN)
	}
}

func TestCollectSelection(t *testing.T) {
	set := openSet(t, packFiles{"A.json": `{"dataList":[{"id":1,"name":"가"}]}`}, packFiles{}, packFiles{})
	cand := term("후보", "候選")
	cand.Status = "candidate"
	r := glossary(term("가", "甲"), term("나", "乙"), cand)

	items, err := Collect(r, set, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].KR != "가" || items[1].KR != "나" {
		t.Errorf("default selection = %+v, want the approved entries 가, 나", items)
	}
	items, err = Collect(r, set, []string{"나"}, nil)
	if err != nil || len(items) != 1 || items[0].KR != "나" {
		t.Errorf("--kr 나 = %+v, %v", items, err)
	}
	if _, err := Collect(r, set, []string{"없음"}, nil); err == nil || !IsUserError(err) {
		t.Errorf("unknown kr: err = %v, want a user error", err)
	}
	if _, err := Collect(r, set, nil, map[string]Candidates{"없음": {EN: []string{"x"}}}); err == nil || !IsUserError(err) {
		t.Errorf("candidates for unknown kr: err = %v, want a user error", err)
	}
	// An entry with no hits has empty (not nil) lists.
	if items[0].Names == nil || items[0].Descs == nil || items[0].Checks == nil {
		t.Errorf("empty lists must not be nil: %+v", items[0])
	}
}

// A nested name pairs with the desc beside it, not with the other levels' or
// the element's own.
func TestCollectDescIsTheSiblingOfTheName(t *testing.T) {
	kr := `{"dataList":[{"id":"K","name":"K","desc":"top","levelList":[{"name":"레벨","desc":"d0"},{"name":"레벨 ","desc":"d1"}]}]}`
	set := openSet(t, packFiles{"Bufs.json": kr}, packFiles{}, packFiles{})
	items, err := Collect(glossary(term("레벨", "等級")), set, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, d := range items[0].Descs {
		keys = append(keys, d.Key)
	}
	want := []string{"Bufs.json|/dataList/0/levelList/0/desc", "Bufs.json|/dataList/0/levelList/1/desc"}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("desc keys = %v, want %v", keys, want)
	}
	// each desc is the sibling of its own name
	if items[0].Descs[0].KR != "d0" || items[0].Descs[1].KR != "d1" {
		t.Errorf("descs = %+v", items[0].Descs)
	}
}

func TestCollectDescTopLevelNameGetsOnlyTopDesc(t *testing.T) {
	kr := `{"dataList":[{"id":"K","name":"파열","desc":"top","levelList":[{"name":"x","desc":"d0"}]}]}`
	set := openSet(t, packFiles{"Bufs.json": kr}, packFiles{}, packFiles{})
	items, err := Collect(glossary(term("파열", "破裂")), set, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items[0].Descs) != 1 || items[0].Descs[0].Key != "Bufs.json|/dataList/0/desc" {
		t.Errorf("descs = %+v", items[0].Descs)
	}
}

// Two names equal to the KR in one element give each desc once.
func TestCollectDescsAreNotRepeated(t *testing.T) {
	kr := `{"dataList":[{"id":"K","name":"파열","desc":"설명","levelList":[{"name":"파열"}]}]}`
	set := openSet(t, packFiles{"Bufs.json": kr}, packFiles{}, packFiles{})
	items, err := Collect(glossary(term("파열", "破裂")), set, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items[0].Descs) != 1 {
		t.Errorf("descs = %+v, want one", items[0].Descs)
	}
}

// A name outside dataList has no element, so no desc.
func TestCollectNoDescOutsideDataList(t *testing.T) {
	kr := `{"name":"파열","desc":"설명","dataList":[]}`
	set := openSet(t, packFiles{"Bufs.json": kr}, packFiles{}, packFiles{})
	items, err := Collect(glossary(term("파열", "破裂")), set, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items[0].Names) != 0 || len(items[0].Descs) != 0 {
		t.Errorf("item = %+v, want no desc", items[0])
	}
}

// Aligned text that is only tags is no text: it is not in N.
func TestCollectTagOnlyAlignedTextIsNotCounted(t *testing.T) {
	kr := lines(10, func(int) string { return "타격 있음" })
	en := lines(10, func(i int) string {
		if i < 5 {
			return "Hit"
		}
		return "<sprite name=x/>"
	})
	set := openSet(t, packFiles{"Lines.json": kr}, packFiles{"Lines.json": en}, packFiles{})
	items, err := Collect(glossary(term("타격", "打擊")), set, nil, map[string]Candidates{"타격": {EN: []string{"Hit"}}})
	if err != nil {
		t.Fatal(err)
	}
	c := items[0].Checks[0]
	if c.N != 5 || c.Hits != 5 || c.OK {
		t.Errorf("check = %+v, want N 5 Hits 5 not ok", c)
	}
}

// The term occurs as a word by default, anywhere with match: substring.
func TestCollectOccurrenceFollowsMatch(t *testing.T) {
	// 타격 inside 강타격감: not a word, since the Hangul before it
	kr := lines(10, func(int) string { return "강타격감" })
	en := lines(10, func(int) string { return "Hit" })
	set := openSet(t, packFiles{"Lines.json": kr}, packFiles{"Lines.json": en}, packFiles{})
	word := term("타격", "打擊")
	sub := term("타격", "打擊")
	sub.Match = rules.MatchSubstring
	cands := map[string]Candidates{"타격": {EN: []string{"Hit"}}}
	for _, tc := range []struct {
		term rules.Term
		n    int
	}{{word, 0}, {sub, 10}} {
		items, err := Collect(glossary(tc.term), set, nil, cands)
		if err != nil {
			t.Fatal(err)
		}
		if got := items[0].Checks[0].N; got != tc.n {
			t.Errorf("match %q: N = %d, want %d", tc.term.Match, got, tc.n)
		}
	}
}

// The last word of an EN candidate may be a plural or singular of it; the
// other words, the boundaries and the case rules stay.
func TestHoldsENPlurals(t *testing.T) {
	tests := []struct {
		text, cand string
		want       bool
	}{
		{"Two Wings of fire", "Wing", true},
		{"a wing", "Wings", true},
		{"Skill Slots left", "Skill Slot", true},
		{"Abilities rise", "Ability", true},
		{"one ability", "Abilities", true},
		{"Two Buses", "Bus", true},
		{"the bus", "Buses", true},
		{"Boxes", "Box", true},
		{"Wingman", "Wing", false},
		{"Slotted", "Slot", false},
		{"Slots of fire", "Slot", true},
		{"Skill Slots", "Skills Slot", false}, // only the last word varies
		{"Skills Slot", "Skill Slot", false},
		{"Wing", "Wing", true},
		{"WINGS", "wing", true},
		{"Bu", "Bus", false}, // a stem too short to drop to
	}
	for _, tc := range tests {
		if got := Holds(LangEN, tc.text, tc.cand); got != tc.want {
			t.Errorf("Holds(EN, %q, %q) = %v, want %v", tc.text, tc.cand, got, tc.want)
		}
	}
	// JP is not touched.
	if Holds(LangJP, "翼たち", "翼s") || !Holds(LangJP, "翼たち", "翼") {
		t.Error("JP matching changed")
	}
}
