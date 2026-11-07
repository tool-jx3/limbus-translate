package evidence

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/fields"
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
	items, err := Collect(glossary(term("파열", "破裂")), set, testFields(t), nil, nil)
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
			items, err := Collect(glossary(term("타격", "打擊")), set, testFields(t), nil, map[string]Candidates{"타격": {EN: []string{tc.cand}}})
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
	items, err := Collect(glossary(term("타격", "打擊")), set, testFields(t), nil, map[string]Candidates{"타격": {JP: []string{"打撃"}, EN: []string{"Hit"}}})
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
	e.JP, e.EN = rules.Forms{"打撃"}, rules.Forms{"Blow"}
	items, err := Collect(glossary(e), set, testFields(t), []string{"타격"}, map[string]Candidates{"타격": {EN: []string{"Blow", "Smash"}}})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range items[0].Checks {
		if len(c.Values) > 0 {
			got = append(got, c.Lang+":"+strings.Join(c.Values, "+"))
			continue
		}
		got = append(got, c.Lang+":"+c.Candidate)
	}
	// Then the candidates given, as a set (the entry's Blow among them).
	want := []string{"JP:打撃", "EN:Blow", "EN:Hit", "EN:Smash", "EN:Blow+Smash"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("candidates = %v, want %v", got, want)
	}
	if !reflect.DeepEqual(items[0].JP, rules.Forms{"打撃"}) || !reflect.DeepEqual(items[0].EN, rules.Forms{"Blow"}) {
		t.Errorf("Item JP/EN = %q/%q", items[0].JP, items[0].EN)
	}
}

func TestCollectSelection(t *testing.T) {
	set := openSet(t, packFiles{"A.json": `{"dataList":[{"id":1,"name":"가"}]}`}, packFiles{}, packFiles{})
	cand := term("후보", "候選")
	cand.Status = "candidate"
	r := glossary(term("가", "甲"), term("나", "乙"), cand)

	items, err := Collect(r, set, testFields(t), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].KR != "가" || items[1].KR != "나" {
		t.Errorf("default selection = %+v, want the approved entries 가, 나", items)
	}
	items, err = Collect(r, set, testFields(t), []string{"나"}, nil)
	if err != nil || len(items) != 1 || items[0].KR != "나" {
		t.Errorf("--kr 나 = %+v, %v", items, err)
	}
	if _, err := Collect(r, set, testFields(t), []string{"없음"}, nil); err == nil || !IsUserError(err) {
		t.Errorf("unknown kr: err = %v, want a user error", err)
	}
	if _, err := Collect(r, set, testFields(t), nil, map[string]Candidates{"없음": {EN: []string{"x"}}}); err == nil || !IsUserError(err) {
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
	items, err := Collect(glossary(term("레벨", "等級")), set, testFields(t), nil, nil)
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
	items, err := Collect(glossary(term("파열", "破裂")), set, testFields(t), nil, nil)
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
	items, err := Collect(glossary(term("파열", "破裂")), set, testFields(t), nil, nil)
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
	items, err := Collect(glossary(term("파열", "破裂")), set, testFields(t), nil, nil)
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
	items, err := Collect(glossary(term("타격", "打擊")), set, testFields(t), nil, map[string]Candidates{"타격": {EN: []string{"Hit"}}})
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
		items, err := Collect(glossary(tc.term), set, testFields(t), nil, cands)
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

// testFields loads the real field rules (rules/fields.manual.yaml, which says
// id and model are identifier fields) from a temp root.
func testFields(t *testing.T) *fields.Rules {
	t.Helper()
	manual, err := os.ReadFile(filepath.Join("..", "..", "rules", "fields.manual.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "rules", "fields.manual.yaml"), manual, 0o644); err != nil {
		t.Fatal(err)
	}
	fr, err := fields.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return fr
}

// Units of a field the field rules do not translate (an identifier, whose
// aligned text is the KR id itself) are not counted in N, as in the scope
// analysis.
func TestCollectSkipsUntranslatedFields(t *testing.T) {
	const n = 12
	var kr, en, jp strings.Builder
	for _, s := range []*strings.Builder{&kr, &en, &jp} {
		s.WriteString(`{"dataList":[`)
	}
	for i := 0; i < n; i++ {
		sep := ","
		if i == 0 {
			sep = ""
		}
		fmt.Fprintf(&kr, `%s{"id":%d,"codeName":"티페리트","content":"티페리트가 왔다"}`, sep, i+1)
		// The codeName unit has aligned text (a model field cannot differ: the
		// aligner pairs entries by it) that differs from its KR and does
		// not hold the candidate, so only the field rule can leave it out.
		fmt.Fprintf(&en, `%s{"id":%d,"codeName":"Codename-X","content":"Tiphereth came"}`, sep, i+1)
		fmt.Fprintf(&jp, `%s{"id":%d,"codeName":"モデルX","content":"ティファレトが来た"}`, sep, i+1)
	}
	for _, s := range []*strings.Builder{&kr, &en, &jp} {
		s.WriteString(`]}`)
	}
	set := openSet(t, packFiles{"Story.json": kr.String()}, packFiles{"Story.json": en.String()}, packFiles{"Story.json": jp.String()})
	items, err := Collect(glossary(term("티페리트", "迭菲瑞特")), set, testFields(t), nil, map[string]Candidates{"티페리트": {EN: []string{"Tiphereth"}}})
	if err != nil {
		t.Fatal(err)
	}
	var got *Check
	for i := range items[0].Checks {
		if items[0].Checks[i].Lang == LangEN && items[0].Checks[i].Candidate == "Tiphereth" {
			got = &items[0].Checks[i]
		}
	}
	if got == nil {
		t.Fatalf("no EN check: %+v", items[0].Checks)
	}
	if got.N != n || got.Hits != n || !got.OK {
		t.Errorf("check = %+v, want N %d Hits %d OK (the codeName units are not counted)", *got, n, n)
	}
}

// The names and descs skip those fields too: a codeName unit would otherwise give
// the term the "name" of its own Hangul id, the most frequent one, and with it
// a candidate that is no rendering.
func TestCollectNamesSkipUntranslatedFields(t *testing.T) {
	set := openSet(t,
		packFiles{"Story.json": `{"dataList":[{"id":1,"codeName":"잔느"},{"id":2,"codeName":"잔느"},{"id":3,"codeName":"잔느"},{"id":4,"name":"잔느"}]}`},
		packFiles{"Story.json": `{"dataList":[{"id":1,"codeName":"Codename-X"},{"id":2,"codeName":"Codename-X"},{"id":3,"codeName":"Codename-X"},{"id":4,"name":"Jeanne"}]}`},
		packFiles{"Story.json": `{"dataList":[{"id":1,"codeName":"モデルX"},{"id":2,"codeName":"モデルX"},{"id":3,"codeName":"モデルX"},{"id":4,"name":"ジャンヌ"}]}`})
	items, err := Collect(glossary(term("잔느", "賈恩")), set, testFields(t), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	it := items[0]
	if len(it.Names) != 1 || it.Names[0].EN != "Jeanne" || it.Names[0].JP != "ジャンヌ" || it.Names[0].Count != 1 {
		t.Errorf("Names = %+v, want only the name unit (Jeanne, ジャンヌ)", it.Names)
	}
	for _, c := range it.Checks {
		if c.Candidate == "잔느" {
			t.Errorf("candidate %+v is the Hangul id of the codeName units", c)
		}
	}
}

// A language whose aligned text is the KR text itself (tags and spaces aside)
// is a copy: the unit is not counted in that language's N. A text that differs
// is counted.
func TestCollectSkipsLanguageEqualToKR(t *testing.T) {
	const n = 10
	var kr, en, jp strings.Builder
	for _, s := range []*strings.Builder{&kr, &en, &jp} {
		s.WriteString(`{"dataList":[`)
	}
	for i := 0; i < 2*n; i++ {
		sep := ","
		if i == 0 {
			sep = ""
		}
		fmt.Fprintf(&kr, `%s{"id":%d,"content":"잔느가 왔다"}`, sep, i+1)
		if i < n {
			// EN is the Hangul text (with a tag and a space around it): a copy.
			fmt.Fprintf(&en, `%s{"id":%d,"content":" <b>잔느가 왔다</b>"}`, sep, i+1)
		} else {
			fmt.Fprintf(&en, `%s{"id":%d,"content":"Jeanne came"}`, sep, i+1)
		}
		fmt.Fprintf(&jp, `%s{"id":%d,"content":"ジャンヌが来た"}`, sep, i+1)
	}
	for _, s := range []*strings.Builder{&kr, &en, &jp} {
		s.WriteString(`]}`)
	}
	set := openSet(t, packFiles{"Story.json": kr.String()}, packFiles{"Story.json": en.String()}, packFiles{"Story.json": jp.String()})
	items, err := Collect(glossary(term("잔느", "賈恩")), set, testFields(t), nil,
		map[string]Candidates{"잔느": {EN: []string{"Jeanne"}, JP: []string{"ジャンヌ"}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range items[0].Checks {
		switch {
		case c.Lang == LangEN && c.Candidate == "Jeanne":
			if c.N != n || c.Hits != n || !c.OK {
				t.Errorf("EN check = %+v, want N %d (the %d copies are not counted)", c, n, n)
			}
		case c.Lang == LangJP && c.Candidate == "ジャンヌ":
			if c.N != 2*n || c.Hits != 2*n {
				t.Errorf("JP check = %+v, want N %d (JP differs in every unit)", c, 2*n)
			}
		}
	}
}

// checkOf returns the check of lang for the candidate, or for the set of
// values when values is given.
func checkOf(t *testing.T, it Item, lang, candidate string, values ...string) Check {
	t.Helper()
	for _, c := range it.Checks {
		if c.Lang != lang {
			continue
		}
		if len(values) == 0 && c.Candidate == candidate && len(c.Values) == 0 || len(values) > 0 && reflect.DeepEqual(c.Values, values) {
			return c
		}
	}
	t.Fatalf("%s: no %s check for %q %v in %+v", it.KR, lang, candidate, values, it.Checks)
	return Check{}
}

// An entry with several forms of its EN is checked as a set: a unit holds it
// when its EN holds any form. Each form keeps a check of its own, so that
// each form's share shows; an entry with one form has no set.
func TestCollectFormsAreCheckedAsASet(t *testing.T) {
	kr := lines(10, func(int) string { return "내성 있음" })
	en := lines(10, func(i int) string {
		switch {
		case i < 5:
			return "Resistance up"
		case i < 9:
			return "Resist it"
		}
		return "Other"
	})
	jp := lines(10, func(int) string { return "耐性あり" })
	set := openSet(t, packFiles{"Lines.json": kr}, packFiles{"Lines.json": en}, packFiles{"Lines.json": jp})
	e := term("내성", "抗性")
	e.JP, e.EN = rules.Forms{"耐性"}, rules.Forms{"Resistance", "Resist"}
	items, err := Collect(glossary(e), set, testFields(t), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	it := items[0]
	if !reflect.DeepEqual(it.EN, rules.Forms{"Resistance", "Resist"}) || !reflect.DeepEqual(it.JP, rules.Forms{"耐性"}) {
		t.Errorf("Item JP/EN = %v/%v", it.JP, it.EN)
	}
	if c := checkOf(t, it, LangEN, "", "Resistance", "Resist"); c.N != 10 || c.Hits != 9 || !c.OK || c.Candidate != "" {
		t.Errorf("set check = %+v, want N 10 Hits 9 OK", c)
	}
	if c := checkOf(t, it, LangEN, "Resistance"); c.N != 10 || c.Hits != 5 || c.OK {
		t.Errorf("Resistance check = %+v, want N 10 Hits 5", c)
	}
	if c := checkOf(t, it, LangEN, "Resist"); c.N != 10 || c.Hits != 4 || c.OK {
		t.Errorf("Resist check = %+v, want N 10 Hits 4", c)
	}
	for _, c := range it.Checks {
		if c.Lang == LangJP && len(c.Values) > 0 {
			t.Errorf("JP has one form, yet a set check %+v", c)
		}
	}
}

// The candidates of a language are checked one by one, as a set, and as a
// set with the entry's own forms (what the entry would have with them added);
// a set of one form is its own check, and a set is listed once.
func TestCollectCandidateSets(t *testing.T) {
	kr := lines(10, func(int) string { return "내성 있음" })
	en := lines(10, func(i int) string { return []string{"A x", "B x", "C x", "D x", "E x"}[i%5] })
	set := openSet(t, packFiles{"Lines.json": kr}, packFiles{"Lines.json": en}, packFiles{})
	sets := func(own rules.Forms, given ...string) [][]string {
		e := term("내성", "抗性")
		e.EN = own
		items, err := Collect(glossary(e), set, testFields(t), nil, map[string]Candidates{"내성": {EN: given}})
		if err != nil {
			t.Fatal(err)
		}
		var out [][]string
		for _, c := range items[0].Checks {
			if len(c.Values) > 0 {
				out = append(out, c.Values)
			}
		}
		return out
	}
	for _, tc := range []struct {
		name  string
		own   rules.Forms
		given []string
		want  [][]string
	}{
		{"no own forms", nil, []string{"A", "B"}, [][]string{{"A", "B"}}},
		{"one own form, one candidate", rules.Forms{"A"}, []string{"B"}, [][]string{{"A", "B"}}},
		{"one own form, two candidates", rules.Forms{"A"}, []string{"B", "C"}, [][]string{{"B", "C"}, {"A", "B", "C"}}},
		{"two own forms, a candidate", rules.Forms{"A", "B"}, []string{"C"}, [][]string{{"A", "B"}, {"A", "B", "C"}}},
		{"candidates among the own forms", rules.Forms{"A", "B"}, []string{"B", "A"}, [][]string{{"A", "B"}}},
		{"one candidate alone", nil, []string{"A"}, nil},
	} {
		if got := sets(tc.own, tc.given...); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: sets = %v, want %v", tc.name, got, tc.want)
		}
	}
	// The set of A and B holds 4 of the 10 units.
	e := term("내성", "抗性")
	items, err := Collect(glossary(e), set, testFields(t), nil, map[string]Candidates{"내성": {EN: []string{"A", "B"}}})
	if err != nil {
		t.Fatal(err)
	}
	if c := checkOf(t, items[0], LangEN, "", "A", "B"); c.N != 10 || c.Hits != 4 {
		t.Errorf("set check = %+v, want N 10 Hits 4", c)
	}
}

// sinbu are the two entries of 신부: 牧師 for the chapter 10 files, 神父 for
// the others, as in the glossary.
func sinbu() (chaplain, father rules.Term) {
	chaplain = term("신부", "牧師")
	chaplain.EN, chaplain.Files = rules.Forms{"The Chaplain"}, []string{"Ch10*"}
	father = term("신부", "神父")
	father.EN = rules.Forms{"Father"}
	return chaplain, father
}

// The evidence of an entry is the units it applies to, as a batch sees them:
// an entry with files counts the units of those files only, and the entry of
// the same KR without files the units of the other files. So are its names.
func TestCollectCountsOnlyWhereTheEntryApplies(t *testing.T) {
	krLines := lines(10, func(int) string { return "신부가 왔다" })
	set := openSet(t,
		packFiles{"Ch10.json": krLines, "Other.json": krLines, "Names.json": `{"dataList":[{"id":1,"name":"신부"}]}`, "Ch10Names.json": `{"dataList":[{"id":1,"name":"신부"}]}`},
		packFiles{"Ch10.json": lines(10, func(int) string { return "The Chaplain came" }), "Other.json": lines(10, func(int) string { return "Father came" }),
			"Names.json": `{"dataList":[{"id":1,"name":"Father"}]}`, "Ch10Names.json": `{"dataList":[{"id":1,"name":"The Chaplain"}]}`},
		packFiles{})
	chaplain, father := sinbu()
	items, err := Collect(glossary(chaplain, father), set, testFields(t), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %+v", items)
	}
	if c := checkOf(t, items[0], LangEN, "The Chaplain"); c.N != 11 || c.Hits != 11 || !c.OK {
		t.Errorf("牧師 check = %+v, want the 11 units of the Ch10 files", c)
	}
	if c := checkOf(t, items[1], LangEN, "Father"); c.N != 11 || c.Hits != 11 || !c.OK {
		t.Errorf("神父 check = %+v, want the 11 units of the other files", c)
	}
	for i, want := range []string{"The Chaplain", "Father"} {
		if ns := items[i].Names; len(ns) != 1 || ns[0].EN != want {
			t.Errorf("%s names = %+v, want only %s", items[i].ZH, ns, want)
		}
	}
	// The name of the other entry's files is no candidate of this one.
	for _, c := range items[1].Checks {
		if c.Candidate == "The Chaplain" {
			t.Errorf("神父 checks the name of the Ch10 files: %+v", c)
		}
	}

	// not_files takes files out the same way.
	only := term("신부", "神父")
	only.EN, only.NotFiles = rules.Forms{"Father"}, []string{"Ch10*"}
	items, err = Collect(glossary(only), set, testFields(t), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if c := checkOf(t, items[0], LangEN, "Father"); c.N != 11 || c.Hits != 11 {
		t.Errorf("not_files check = %+v, want the 11 units outside the Ch10 files", c)
	}
}

// Where a longer entry covers the term (전투 승리 in EX CLEAR 전투 승리), the
// unit is the longer entry's, and not evidence of the shorter one.
func TestCollectLongerEntryTakesItsUnits(t *testing.T) {
	kr := lines(20, func(i int) string {
		if i < 10 {
			return "EX CLEAR 전투 승리 달성"
		}
		return "전투 승리"
	})
	en := lines(20, func(i int) string {
		if i < 10 {
			return "EX CLEAR achieved"
		}
		return "Victory"
	})
	set := openSet(t, packFiles{"Lines.json": kr}, packFiles{"Lines.json": en}, packFiles{})
	short, long := term("전투 승리", "戰鬥勝利"), term("EX CLEAR 전투 승리", "EX CLEAR 戰鬥勝利")
	short.EN, long.EN = rules.Forms{"Victory"}, rules.Forms{"EX CLEAR"}
	items, err := Collect(glossary(short, long), set, testFields(t), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if c := checkOf(t, items[0], LangEN, "Victory"); c.N != 10 || c.Hits != 10 {
		t.Errorf("전투 승리 check = %+v, want only the 10 units the longer entry does not cover", c)
	}
	if c := checkOf(t, items[1], LangEN, "EX CLEAR"); c.N != 10 || c.Hits != 10 {
		t.Errorf("EX CLEAR 전투 승리 check = %+v, want its 10 units", c)
	}
	// Asked for alone, the shorter entry is still weighed against the longer.
	items, err = Collect(glossary(short, long), set, testFields(t), []string{"전투 승리"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if c := checkOf(t, items[0], LangEN, "Victory"); c.N != 10 {
		t.Errorf("--kr 전투 승리 check = %+v, want N 10", c)
	}
}

// A candidate entry binds nothing, so it takes no unit from an approved one;
// asked for by --kr, it counts the units it would apply to once approved.
func TestCollectCandidateEntryCountsWhereItWouldApply(t *testing.T) {
	krLines := lines(10, func(int) string { return "신부가 왔다" })
	set := openSet(t,
		packFiles{"Ch10.json": krLines, "Other.json": krLines},
		packFiles{"Ch10.json": lines(10, func(int) string { return "The Chaplain came" }), "Other.json": lines(10, func(int) string { return "Father came" })},
		packFiles{})
	chaplain, father := sinbu()
	chaplain.Status = rules.StatusCandidate
	father.EN = rules.Forms{"came"}
	items, err := Collect(glossary(chaplain, father), set, testFields(t), []string{"신부"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if c := checkOf(t, items[0], LangEN, "The Chaplain"); c.N != 10 || c.Hits != 10 {
		t.Errorf("candidate check = %+v, want the 10 units of Ch10.json", c)
	}
	if c := checkOf(t, items[1], LangEN, "came"); c.N != 20 {
		t.Errorf("approved check = %+v, want all 20 units: the candidate takes none", c)
	}
}

// distinct takes lists of forms, a rules.Forms as well as a []string: each
// non-blank value once, trimmed, in order.
func TestDistinctTakesForms(t *testing.T) {
	got := distinct(rules.Forms{"Resistance", "", " Resist "}, []string{"Resist", "Ineff."}, nil)
	if want := []string{"Resistance", "Resist", "Ineff."}; !reflect.DeepEqual(got, want) {
		t.Errorf("distinct = %q, want %q", got, want)
	}
}
