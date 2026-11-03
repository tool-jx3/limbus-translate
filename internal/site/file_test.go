package site

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// writePack writes files, keyed by pack file name, under a fresh directory
// with prefix on every file name, and opens it as a pack.
func writePack(t *testing.T, prefix string, files map[string]string) *pack.Pack {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		dir, base := path.Split(name)
		full := filepath.Join(root, filepath.FromSlash(dir), prefix+base)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p, err := pack.Open(root, prefix)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// testRules has a story family and a catch-all one.
func testRules() *rules.Rules {
	return &rules.Rules{Routing: rules.Routing{
		Families: []rules.FamilyRule{{ID: "story", Match: []string{"StoryData/*"}}, {ID: "other"}},
		Priority: []string{"story", "other"},
	}}
}

// recordsOf indexes records by their key.
func recordsOf(rs ...store.Record) map[string]store.Record {
	m := make(map[string]store.Record, len(rs))
	for _, r := range rs {
		m[r.Key] = r
	}
	return m
}

func record(file, pointer, zh, origin string, flags ...string) store.Record {
	return store.Record{Key: units.Key(file, pointer), ZH: zh, Origin: origin, Flags: flags}
}

// unitAt returns the unit of f at pointer.
func unitAt(t *testing.T, f File, pointer string) Unit {
	t.Helper()
	for _, u := range f.Units {
		if u.P == pointer {
			return u
		}
	}
	t.Fatalf("%s has no unit at %s", f.Path, pointer)
	return Unit{}
}

// same reports whether got points to a string equal to want.
func same(got *string, want string) bool { return got != nil && *got == want }

func show(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%q", *p)
}

func TestBuildFileAlignsByIDNotPosition(t *testing.T) {
	const file = "Sample.json"
	in := Inputs{
		KR: writePack(t, "KR_", map[string]string{file: `{"dataList":[
			{"id":1,"content":"하나"},{"id":2,"content":"둘"},{"id":3,"content":"셋"}]}`}),
		EN: writePack(t, "EN_", map[string]string{file: `{"dataList":[
			{"id":2,"content":"two"},{"id":1,"content":"one"},{"id":3,"content":"three"}]}`}),
		JP: writePack(t, "JP_", map[string]string{file: `{"dataList":[
			{"id":2,"content":"に"},{"id":1,"content":"いち"},{"id":3,"content":"さん"}]}`}),
		ZH: writePack(t, "", map[string]string{file: `{"dataList":[
			{"id":1,"content":"一"},{"id":2,"content":"二"},{"id":3,"content":"三"}]}`}),
		Rules: testRules(),
	}
	f, sum, err := BuildFile(in, file)
	if err != nil {
		t.Fatal(err)
	}
	if f.Path != file || f.Family != "other" || sum.Path != file || sum.Family != "other" {
		t.Errorf("path/family = %q %q, summary %q %q, want %q other", f.Path, f.Family, sum.Path, sum.Family, file)
	}
	for ptr, want := range map[string][2]string{
		"/dataList/0/content": {"いち", "one"}, // not the first JP/EN entry
		"/dataList/1/content": {"に", "two"},
		"/dataList/2/content": {"さん", "three"},
	} {
		u := unitAt(t, f, ptr)
		if !same(u.JP, want[0]) || !same(u.EN, want[1]) {
			t.Errorf("%s: JP %s EN %s, want %q %q", ptr, show(u.JP), show(u.EN), want[0], want[1])
		}
	}
	if u := unitAt(t, f, "/dataList/0/content"); u.KR != "하나" || u.ZH != "一" {
		t.Errorf("KR %q ZH %q, want 하나 一", u.KR, u.ZH)
	}
	if sum.Missing != (Missing{}) {
		t.Errorf("Missing = %+v, want none", sum.Missing)
	}
}

func TestBuildFileMissingLanguage(t *testing.T) {
	const file = "Sample.json"
	kr := `{"dataList":[{"id":1,"content":"하나"},{"id":2,"content":"둘"},{"id":3,"content":"셋"}]}`
	zh := `{"dataList":[{"id":1,"content":"一"},{"id":2,"content":"二"},{"id":3,"content":"三"}]}`
	en := `{"dataList":[{"id":1,"content":"one"},{"id":2,"content":"two"}]}`
	jp := `{"dataList":[{"id":1,"content":"いち"},{"id":2,"content":"に"},{"id":3,"content":"さん"}]}`

	t.Run("a unit the language lacks", func(t *testing.T) {
		in := Inputs{
			KR:    writePack(t, "KR_", map[string]string{file: kr}),
			EN:    writePack(t, "EN_", map[string]string{file: en}),
			JP:    writePack(t, "JP_", map[string]string{file: jp}),
			ZH:    writePack(t, "", map[string]string{file: zh}),
			Rules: testRules(),
		}
		f, sum, err := BuildFile(in, file)
		if err != nil {
			t.Fatal(err)
		}
		if u := unitAt(t, f, "/dataList/2/content"); u.EN != nil {
			t.Errorf("EN of id 3 = %s, want nil", show(u.EN))
		}
		if u := unitAt(t, f, "/dataList/1/content"); !same(u.EN, "two") {
			t.Errorf("EN of id 2 = %s, want two", show(u.EN))
		}
		if sum.Missing != (Missing{EN: 1, JP: 0}) {
			t.Errorf("Missing = %+v, want {EN:1 JP:0}", sum.Missing)
		}
	})

	t.Run("a pack without the file", func(t *testing.T) {
		in := Inputs{
			KR:    writePack(t, "KR_", map[string]string{file: kr}),
			EN:    writePack(t, "EN_", map[string]string{"Other.json": en}),
			JP:    writePack(t, "JP_", map[string]string{file: jp}),
			ZH:    writePack(t, "", map[string]string{file: zh}),
			Rules: testRules(),
		}
		f, sum, err := BuildFile(in, file)
		if err != nil {
			t.Fatal(err)
		}
		if len(f.Units) != 3 {
			t.Fatalf("%d units, want 3", len(f.Units))
		}
		for _, u := range f.Units {
			if u.EN != nil {
				t.Errorf("%s: EN = %s, want nil", u.P, show(u.EN))
			}
			if u.JP == nil {
				t.Errorf("%s: JP = nil, want a string", u.P)
			}
		}
		// the file is not there for EN: every unit lacks a string in it (spec 5.2)
		if sum.Missing != (Missing{EN: 3, JP: 0}) {
			t.Errorf("Missing = %+v, want {EN:3 JP:0}", sum.Missing)
		}
	})

	t.Run("no pack has the file", func(t *testing.T) {
		in := Inputs{
			KR:    writePack(t, "KR_", map[string]string{file: kr}),
			EN:    writePack(t, "EN_", nil),
			JP:    writePack(t, "JP_", map[string]string{"Other.json": jp}),
			ZH:    writePack(t, "", map[string]string{file: zh}),
			Rules: testRules(),
		}
		_, sum, err := BuildFile(in, file)
		if err != nil {
			t.Fatal(err)
		}
		if sum.Missing != (Missing{EN: 3, JP: 3}) {
			t.Errorf("Missing = %+v, want {EN:3 JP:3}", sum.Missing)
		}
	})

	t.Run("a file without strings has nothing missing", func(t *testing.T) {
		in := Inputs{
			KR:    writePack(t, "KR_", map[string]string{file: `{"dataList":[{"id":1}]}`}),
			EN:    writePack(t, "EN_", nil),
			JP:    writePack(t, "JP_", nil),
			ZH:    writePack(t, "", map[string]string{file: `{"dataList":[{"id":1}]}`}),
			Rules: testRules(),
		}
		_, sum, err := BuildFile(in, file)
		if err != nil {
			t.Fatal(err)
		}
		if sum.Missing != (Missing{}) {
			t.Errorf("Missing = %+v, want none", sum.Missing)
		}
	})
}

func TestBuildFileOrigin(t *testing.T) {
	const file = "Sample.json"
	krDoc := `{"dataList":[
		{"id":1,"content":"가"},{"id":2,"content":"나"},{"id":3,"content":"다"},
		{"id":4,"content":"라"},{"id":5,"content":"마"},{"id":6,"content":"바"}]}`
	zhDoc := `{"dataList":[
		{"id":1,"content":"甲"},{"id":2,"content":"乙"},{"id":3,"content":"丙"},
		{"id":4,"content":"丁"},{"id":5,"content":"戊"},{"id":6,"content":"己"}]}`
	in := Inputs{
		KR: writePack(t, "KR_", map[string]string{file: krDoc}),
		EN: writePack(t, "EN_", nil),
		JP: writePack(t, "JP_", nil),
		ZH: writePack(t, "", map[string]string{file: zhDoc}),
		Records: recordsOf(
			record(file, "/dataList/0/content", "甲", store.OriginParatranz),
			record(file, "/dataList/1/content", "不同", store.OriginClaude), // the pack shows something else
			// /dataList/2: no record
			record(file, "/dataList/3/content", "丁", store.OriginClaude, store.FlagNeedsReview),
			record(file, "/dataList/4/content", "戊", store.OriginParatranz),
			// A flagged record whose string the pack does not show: a stand-in
			// that still carries the flag.
			record(file, "/dataList/5/content", "不符", store.OriginClaude, store.FlagNeedsReview),
		),
		Rules: testRules(),
	}
	f, sum, err := BuildFile(in, file)
	if err != nil {
		t.Fatal(err)
	}
	for ptr, want := range map[string]string{
		"/dataList/0/content": store.OriginParatranz,
		"/dataList/1/content": OriginStandIn,
		"/dataList/2/content": OriginStandIn,
		"/dataList/3/content": store.OriginClaude,
		"/dataList/4/content": store.OriginParatranz,
		"/dataList/5/content": OriginStandIn,
	} {
		if got := unitAt(t, f, ptr).O; got != want {
			t.Errorf("%s: O = %q, want %q", ptr, got, want)
		}
	}
	if fl := unitAt(t, f, "/dataList/3/content").Fl; !slices.Equal(fl, []string{store.FlagNeedsReview}) {
		t.Errorf("Fl = %v, want [needs_review]", fl)
	}
	if fl := unitAt(t, f, "/dataList/5/content").Fl; !slices.Equal(fl, []string{store.FlagNeedsReview}) {
		t.Errorf("Fl of a stand-in with a flagged record = %v, want [needs_review]", fl)
	}
	if fl := unitAt(t, f, "/dataList/0/content").Fl; fl != nil {
		t.Errorf("Fl of a record without flags = %v, want nil", fl)
	}
	wantOrigins := map[string]int{store.OriginParatranz: 2, store.OriginClaude: 1, OriginStandIn: 3}
	if len(sum.Origins) != len(wantOrigins) {
		t.Errorf("Origins = %v, want %v", sum.Origins, wantOrigins)
	}
	for o, n := range wantOrigins {
		if sum.Origins[o] != n {
			t.Errorf("Origins[%s] = %d, want %d", o, sum.Origins[o], n)
		}
	}
	if sum.Units != 6 {
		t.Errorf("Units = %d, want 6", sum.Units)
	}
}

func TestBuildFileItemIndex(t *testing.T) {
	const file = "Sample.json"
	doc := `{"title":"%s","dataList":[
		{"id":1,"name":"%s","levelList":[{"name":"%s"}]},
		{"id":2,"name":"%s"}],"footer":"%s"}`
	in := Inputs{
		KR:    writePack(t, "KR_", map[string]string{file: fmt.Sprintf(doc, "제목", "가", "나", "다", "끝")}),
		EN:    writePack(t, "EN_", nil),
		JP:    writePack(t, "JP_", nil),
		ZH:    writePack(t, "", map[string]string{file: fmt.Sprintf(doc, "標題", "甲", "乙", "丙", "尾")}),
		Rules: testRules(),
	}
	f, _, err := BuildFile(in, file)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{
		"/title":                       -1,
		"/dataList/0/name":             0,
		"/dataList/0/levelList/0/name": 0, // nested lists belong to their element
		"/dataList/1/name":             1,
		"/footer":                      -1,
	}
	if len(f.Units) != len(want) {
		t.Fatalf("%d units, want %d", len(f.Units), len(want))
	}
	for ptr, i := range want {
		u := unitAt(t, f, ptr)
		if u.I != i {
			t.Errorf("%s: I = %d, want %d", ptr, u.I, i)
		}
		if u.F != units.FieldName(ptr) {
			t.Errorf("%s: F = %q, want %q", ptr, u.F, units.FieldName(ptr))
		}
	}
	// Units come in the order of the KR file.
	var got []string
	for _, u := range f.Units {
		got = append(got, u.P)
	}
	wantOrder := []string{"/title", "/dataList/0/name", "/dataList/0/levelList/0/name", "/dataList/1/name", "/footer"}
	if !slices.Equal(got, wantOrder) {
		t.Errorf("order = %v, want %v", got, wantOrder)
	}
}

// storyDoc is a story file whose items have the given places and tellers.
func storyDoc(places, tellers []string) string {
	var items []string
	for i := range max(len(places), len(tellers)) {
		item := fmt.Sprintf(`{"id":%d`, i+1)
		if i < len(places) {
			item += fmt.Sprintf(`,"place":%q`, places[i])
		}
		if i < len(tellers) {
			item += fmt.Sprintf(`,"teller":%q`, tellers[i])
		}
		items = append(items, item+`,"content":"."}`)
	}
	return `{"dataList":[` + strings.Join(items, ",") + `]}`
}

func TestBuildFileStoryLabelAndSpeakers(t *testing.T) {
	const story = "StoryData/S9.json"

	build := func(t *testing.T, file, krDoc, zhDoc string) (File, FileSummary) {
		t.Helper()
		in := Inputs{
			KR:    writePack(t, "KR_", map[string]string{file: krDoc}),
			EN:    writePack(t, "EN_", nil),
			JP:    writePack(t, "JP_", nil),
			ZH:    writePack(t, "", map[string]string{file: zhDoc}),
			Rules: testRules(),
		}
		f, sum, err := BuildFile(in, file)
		if err != nil {
			t.Fatal(err)
		}
		return f, sum
	}

	t.Run("label is the first non-empty place and speakers are unique", func(t *testing.T) {
		kr := storyDoc([]string{"", "장소", "다른 곳", "", ""}, []string{"", "가", "나", "가", "다"})
		zh := storyDoc([]string{"", "某地", "別處", "", ""}, []string{"", "甲", "乙", "甲", "丙"})
		f, sum := build(t, story, kr, zh)
		if f.Family != "story" || sum.Family != "story" {
			t.Errorf("family = %q / %q, want story", f.Family, sum.Family)
		}
		if sum.Label != "某地" {
			t.Errorf("Label = %q, want 某地", sum.Label)
		}
		if want := []string{"甲", "乙", "丙"}; !slices.Equal(sum.Speakers, want) {
			t.Errorf("Speakers = %v, want %v", sum.Speakers, want)
		}
	})

	t.Run("the label falls back to KR", func(t *testing.T) {
		kr := storyDoc([]string{"장소"}, nil)
		zh := storyDoc([]string{""}, nil)
		if _, sum := build(t, story, kr, zh); sum.Label != "장소" {
			t.Errorf("Label = %q, want 장소", sum.Label)
		}
	})

	t.Run("no place, no speaker", func(t *testing.T) {
		_, sum := build(t, story, storyDoc(nil, nil), storyDoc(nil, nil))
		if sum.Label != "" {
			t.Errorf("Label = %q, want empty", sum.Label)
		}
		if sum.Speakers == nil || len(sum.Speakers) != 0 {
			t.Errorf("Speakers = %#v, want an empty, non-nil slice", sum.Speakers)
		}
	})

	t.Run("at most 20 speakers", func(t *testing.T) {
		var kr, zh []string
		for i := range 25 {
			kr = append(kr, fmt.Sprintf("화자%d", i))
			zh = append(zh, fmt.Sprintf("說話者%d", i))
		}
		_, sum := build(t, story, storyDoc(nil, kr), storyDoc(nil, zh))
		if len(sum.Speakers) != 20 {
			t.Fatalf("%d speakers, want 20", len(sum.Speakers))
		}
		if sum.Speakers[0] != "說話者0" || sum.Speakers[19] != "說話者19" {
			t.Errorf("Speakers = %v, want the first 20 in order", sum.Speakers)
		}
	})

	t.Run("other files have no label", func(t *testing.T) {
		kr := storyDoc([]string{"장소"}, []string{"가"})
		zh := storyDoc([]string{"某地"}, []string{"甲"})
		_, sum := build(t, "Other.json", kr, zh)
		if sum.Label != "" {
			t.Errorf("Label = %q, want empty outside StoryData/", sum.Label)
		}
	})
}

func TestBuildFileJSONShape(t *testing.T) {
	const file = "Sample.json"
	in := Inputs{
		KR: writePack(t, "KR_", map[string]string{file: `{"dataList":[{"id":1,"content":"가"},{"id":2,"content":"나"}]}`}),
		EN: writePack(t, "EN_", map[string]string{file: `{"dataList":[{"id":1,"content":"a"},{"id":2,"content":"b"}]}`}),
		JP: writePack(t, "JP_", map[string]string{file: `{"dataList":[{"id":2,"content":"に"}]}`}), // lacks id 1
		ZH: writePack(t, "", map[string]string{file: `{"dataList":[{"id":1,"content":"甲"},{"id":2,"content":"乙"}]}`}),
		Records: recordsOf(
			record(file, "/dataList/0/content", "甲", store.OriginParatranz),
			record(file, "/dataList/1/content", "乙", store.OriginClaude, store.FlagNeedsReview),
		),
		Rules: testRules(),
	}
	f, _, err := BuildFile(in, file)
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"path":"Sample.json","family":"other","units":[` +
		`{"p":"/dataList/0/content","f":"content","i":0,"kr":"가","en":"a","zh":"甲","o":"paratranz"},` +
		`{"p":"/dataList/1/content","f":"content","i":1,"kr":"나","jp":"に","en":"b","zh":"乙","o":"claude","fl":["needs_review"]}]}`
	if string(got) != want {
		t.Errorf("JSON =\n%s\nwant\n%s", got, want)
	}
}

func TestFileSummaryJSONShape(t *testing.T) {
	sum := FileSummary{
		Path: "StoryData/S1.json", Family: "story", Label: "某地", Speakers: []string{"甲"},
		Units: 2, Origins: map[string]int{"paratranz": 1, OriginStandIn: 1}, Missing: Missing{EN: 1, JP: 0},
	}
	got, err := json.Marshal(sum)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"path":"StoryData/S1.json","family":"story","label":"某地","speakers":["甲"],` +
		`"units":2,"origins":{"paratranz":1,"stand-in":1},"missing":{"en":1,"jp":0}}`
	if string(got) != want {
		t.Errorf("JSON =\n%s\nwant\n%s", got, want)
	}
}

func TestBuildFileUnknownFile(t *testing.T) {
	in := Inputs{
		KR:    writePack(t, "KR_", map[string]string{"A.json": `{"dataList":[]}`}),
		EN:    writePack(t, "EN_", nil),
		JP:    writePack(t, "JP_", nil),
		ZH:    writePack(t, "", map[string]string{"A.json": `{"dataList":[]}`}),
		Rules: testRules(),
	}
	if _, _, err := BuildFile(in, "B.json"); err == nil {
		t.Error("BuildFile of a file the KR pack lacks: want an error")
	}
	in.KR = writePack(t, "KR_", map[string]string{"A.json": `{"dataList":[]}`, "B.json": `{"dataList":[]}`})
	if _, _, err := BuildFile(in, "B.json"); err == nil {
		t.Error("BuildFile of a file the built pack lacks: want an error")
	}
}
