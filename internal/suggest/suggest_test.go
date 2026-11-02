package suggest

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/adjust"
	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/plan"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
)

const (
	rawTag = "raw-20261105"

	movedFrom   = "StoryData/E002X.json|/dataList/14/content"
	movedTo     = "StoryData/E002X.json|/dataList/16/content"
	expiredKey  = "StoryData/E002X.json|/dataList/20/content"
	noteKey     = "StoryData/S001A.json|/dataList/3/content"
	krInconsist = "도약 모듈"
	krNote      = "KR 這句有雙關"
)

// testRules is the rules the fixture suggests against: one character with one
// model, one family for the S stories and the catch-all, and a glossary with
// one candidate term.
func testRules() *rules.Rules {
	return &rules.Rules{
		Glossary: []rules.Term{
			{KR: krInconsist, ZH: "跳躍成長模組", Status: rules.StatusApproved},
			{KR: "신규 용어", ZH: "新術語", Status: rules.StatusCandidate},
		},
		Characters: []rules.Character{
			{ID: "yisang", Models: []string{"이상"}, ModelsOtherSelf: []string{"이상다침"}, ModelsNotThis: []string{"이상&파우스트"}},
		},
		Routing: rules.Routing{
			Families: []rules.FamilyRule{
				{ID: "story-main", Match: []string{"StoryData/S*"}},
				{ID: "other", Match: []string{}},
			},
			Priority: []string{"story-main", "other"},
		},
	}
}

// fixture holds one of each of the eight kinds.
func fixture() Inputs {
	return Inputs{
		RawTag: rawTag,
		Rules:  testRules(),
		Adjustments: []adjust.Adjustment{
			{Key: movedFrom, ZH: "移動的調整", File: "2026-11-01-a.yaml"},
			{Key: expiredKey, ZH: "過期的調整", File: "2026-11-01-a.yaml"},
		},
		Build: &build.Info{
			RawTag: rawTag,
			Report: build.Report{
				Inconsistent: []build.Inconsistency{{
					Family: "ally", Field: "name", KR: krInconsist,
					ZH:     []string{"跳躍成長模組", "跳躍模組"},
					Keys:   []string{"Skills_personality-01.json|/dataList/0/name", "Skills_personality-02.json|/dataList/3/name"},
					Counts: []int{9, 3},
					Units:  12,
				}},
				Notes: []store.Note{{Key: noteKey, Batch: "b1", KRSha: "x", Model: "m", Note: krNote}},
			},
		},
		Plan: &plan.Summary{
			RawTag:                rawTag,
			NewSpeakers:           []string{"새 화자"},
			FieldChanges:          []fields.Change{{Op: "add", Group: "BattleSpeechBubbleDlg", Field: "desc"}},
			NewFilesDefaultFamily: []string{"Foo-twth.json"},
			Adjustments: plan.AdjustmentSummary{
				Moved:   []plan.MovedKey{{From: movedFrom, To: movedTo}},
				Expired: []string{expiredKey},
			},
		},
	}
}

func ids(list []Suggestion) []string {
	out := make([]string, len(list))
	for i, s := range list {
		out[i] = s.ID
	}
	return out
}

func run(t *testing.T, in Inputs) []Suggestion {
	t.Helper()
	got, err := Run(in)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return got
}

// The ids are the contract with the rules PRs (<!-- suggestion:<id> -->), so
// they are pinned to the text. They come in the order of Kinds.
func TestSuggestKindsAndIDs(t *testing.T) {
	got := run(t, fixture())
	want := []string{
		"note/" + noteKey + "/3713708f", // sha256("KR 這句有雙關")[:8]
		"speaker/새 화자",
		"family/Foo-twth.json",
		"field/add/BattleSpeechBubbleDlg/desc",
		"adjustment-moved/" + movedFrom,
		"adjustment-expired/" + expiredKey,
		"term/신규 용어",
		"inconsistent/ally/name/fd93b875", // sha256("도약 모듈")[:8]
	}
	if !slices.Equal(ids(got), want) {
		t.Fatalf("ids =\n  %s\nwant\n  %s", strings.Join(ids(got), "\n  "), strings.Join(want, "\n  "))
	}

	kinds := make([]string, len(got))
	for i, s := range got {
		kinds[i], _, _ = strings.Cut(s.ID, "/")
		if s.Kind != kinds[i] {
			t.Errorf("%s: Kind = %q, want the part of the id before the first /", s.ID, s.Kind)
		}
		if s.Title == "" {
			t.Errorf("%s: no title", s.ID)
		}
		if s.Evidence == nil {
			t.Errorf("%s: Evidence is nil; it is written as [] and not null", s.ID)
		}
		if len(s.Evidence) > MaxEvidence {
			t.Errorf("%s: %d evidence rows, at most %d", s.ID, len(s.Evidence), MaxEvidence)
		}
	}
	if !slices.Equal(kinds, Kinds) {
		t.Errorf("kinds = %v, want one of each of %v, in that order", kinds, Kinds)
	}
}

// The suggestions come by kind, in the order of Kinds: those a person answers
// at once and that are few first, the groups of different translations, of
// which a build finds hundreds, last; the biggest group first. The id decides
// between two of the same kind (and two groups of the same size). A page of
// lct suggest is taken from this order, so a run of the rules-review skill
// gets the most useful suggestions first.
func TestSuggestOrder(t *testing.T) {
	in := fixture()
	group := func(kr string, units int) build.Inconsistency {
		return build.Inconsistency{Family: "ally", Field: "name", KR: kr, ZH: []string{"甲", "乙"}, Keys: []string{"a|/x", "b|/y"}, Units: units}
	}
	in.Build.InconsistentAll = []build.Inconsistency{
		group("작은", 2), group(krInconsist, 12), group("큰", 40), group("같은 크기", 12),
	}
	in.Build.Report.Notes = append(in.Build.Report.Notes, store.Note{Key: "A.json|/a", Note: "另一則"})
	in.Plan.NewSpeakers = []string{"새 화자", "가 화자"}
	got := ids(run(t, in))
	want := []string{
		"note/A.json|/a/" + sha8("另一則"),
		"note/" + noteKey + "/3713708f",
		"speaker/가 화자",
		"speaker/새 화자",
		"family/Foo-twth.json",
		"field/add/BattleSpeechBubbleDlg/desc",
		"adjustment-moved/" + movedFrom,
		"adjustment-expired/" + expiredKey,
		"term/신규 용어",
		"inconsistent/ally/name/" + sha8("큰"), // 40 units
		min("inconsistent/ally/name/"+sha8(krInconsist), "inconsistent/ally/name/"+sha8("같은 크기")), // 12 units each: by id
		max("inconsistent/ally/name/"+sha8(krInconsist), "inconsistent/ally/name/"+sha8("같은 크기")),
		"inconsistent/ally/name/" + sha8("작은"), // 2 units
	}
	if !slices.Equal(got, want) {
		t.Errorf("order =\n  %s\nwant\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
}

func TestSuggestEvidence(t *testing.T) {
	byID := map[string]Suggestion{}
	for _, s := range run(t, fixture()) {
		byID[s.ID] = s
	}

	// The KR with each translation and the first unit that has it.
	// The KR with each translation, the first unit that has it and how many
	// units do.
	inc := byID["inconsistent/ally/name/fd93b875"]
	wantInc := []Evidence{
		{Key: "Skills_personality-01.json|/dataList/0/name", KR: krInconsist, ZH: "跳躍成長模組", Count: 9},
		{Key: "Skills_personality-02.json|/dataList/3/name", KR: krInconsist, ZH: "跳躍模組", Count: 3},
	}
	if !reflect.DeepEqual(inc.Evidence, wantInc) {
		t.Errorf("inconsistent evidence = %+v, want %+v", inc.Evidence, wantInc)
	}
	// A build.json of an older lct has no counts.
	old := fixture()
	old.Build.Report.Inconsistent[0].Counts = nil
	for _, s := range run(t, old) {
		if s.Kind != KindInconsistent {
			continue
		}
		for _, e := range s.Evidence {
			if e.Count != 0 {
				t.Errorf("evidence of a group without counts = %+v, want no count", s.Evidence)
			}
		}
	}
	for _, want := range []string{"ally", "name", krInconsist, "12", "2"} {
		if !strings.Contains(inc.Title, want) {
			t.Errorf("inconsistent title %q lacks %q", inc.Title, want)
		}
	}

	if got := byID["speaker/새 화자"].Evidence; !reflect.DeepEqual(got, []Evidence{{KR: "새 화자"}}) {
		t.Errorf("speaker evidence = %+v, want the model", got)
	}
	if got := byID["term/신규 용어"].Evidence; !reflect.DeepEqual(got, []Evidence{{KR: "신규 용어", ZH: "新術語"}}) {
		t.Errorf("term evidence = %+v, want the term", got)
	}
	n := byID["note/"+noteKey+"/3713708f"]
	if !reflect.DeepEqual(n.Evidence, []Evidence{{Key: noteKey, Note: krNote}}) || !strings.Contains(n.Title, krNote) {
		t.Errorf("note = %+v, want the key and the note as evidence and the note in the title", n)
	}

	// The adjustment as written, and where its text is now.
	moved := byID["adjustment-moved/"+movedFrom]
	if want := []Evidence{{Key: movedFrom, ZH: "移動的調整"}, {Key: movedTo}}; !reflect.DeepEqual(moved.Evidence, want) {
		t.Errorf("moved evidence = %+v, want %+v", moved.Evidence, want)
	}
	for _, want := range []string{movedFrom, movedTo, "rules/adjustments/2026-11-01-a.yaml"} {
		if !strings.Contains(moved.Title, want) {
			t.Errorf("moved title %q lacks %q", moved.Title, want)
		}
	}
	expired := byID["adjustment-expired/"+expiredKey]
	if want := []Evidence{{Key: expiredKey, ZH: "過期的調整"}}; !reflect.DeepEqual(expired.Evidence, want) {
		t.Errorf("expired evidence = %+v, want %+v", expired.Evidence, want)
	}
}

// A field rule without a group is an identifier field: "-" stands for the
// group in the id.
func TestSuggestFieldIDWithoutGroup(t *testing.T) {
	in := fixture()
	in.Plan.FieldChanges = []fields.Change{
		{Op: "remove", Field: "key"},
		{Op: "add", Group: "Skills", Field: "tag"},
		{Op: "remove", Field: "key"}, // the same thing twice is one suggestion
	}
	var got []string
	for _, s := range run(t, in) {
		if s.Kind == KindField {
			got = append(got, s.ID)
		}
	}
	if want := []string{"field/add/Skills/tag", "field/remove/-/key"}; !slices.Equal(got, want) {
		t.Errorf("field ids = %v, want %v", got, want)
	}
}

func TestSuggestIDsAreStable(t *testing.T) {
	first := run(t, fixture())
	if again := run(t, fixture()); !reflect.DeepEqual(first, again) {
		t.Errorf("a second run gave\n%+v\nwant\n%+v", again, first)
	}

	// The same facts in another order are the same suggestions in the same
	// order: ZH and Keys are listed in the order the units were met.
	in := fixture()
	inc := &in.Build.Report.Inconsistent[0]
	slices.Reverse(inc.ZH)
	slices.Reverse(inc.Keys)
	slices.Reverse(inc.Counts)
	in.Build.Report.Notes = append(in.Build.Report.Notes, store.Note{Key: "a.json|/x", Note: "n"})
	in.Plan.NewSpeakers = []string{"새 화자", "다른 화자"}
	in.Plan.NewFilesDefaultFamily = []string{"Foo-twth.json", "Bar-twth.json"}
	reordered := run(t, in)

	in2 := fixture()
	in2.Build.Report.Notes = []store.Note{{Key: "a.json|/x", Note: "n"}, in2.Build.Report.Notes[0]}
	in2.Plan.NewSpeakers = []string{"다른 화자", "새 화자"}
	in2.Plan.NewFilesDefaultFamily = []string{"Bar-twth.json", "Foo-twth.json"}
	if other := run(t, in2); !reflect.DeepEqual(other, reordered) {
		t.Errorf("the same facts in another order gave\n%+v\nwant\n%+v", other, reordered)
	}

	// The id of a group does not depend on its translations.
	if want, got := "inconsistent/ally/name/fd93b875", ids(reordered)[slices.IndexFunc(reordered, func(s Suggestion) bool { return s.Kind == KindInconsistent })]; got != want {
		t.Errorf("inconsistent id = %q, want %q", got, want)
	}
	// The order is that of the kinds, then of the ids, whatever order the
	// facts came in.
	for i := 1; i < len(reordered); i++ {
		a, b := reordered[i-1], reordered[i]
		if ka, kb := slices.Index(Kinds, a.Kind), slices.Index(Kinds, b.Kind); ka > kb || ka == kb && a.ID > b.ID {
			t.Errorf("%s comes before %s", a.ID, b.ID)
		}
	}
}

// What the rules answer already is not suggested again; this is checked on
// every run, since plan.json keeps its lists until the next snapshot.
func TestSuggestSkipsDeclinedAndResolved(t *testing.T) {
	in := fixture()
	in.Plan.NewSpeakers = append(in.Plan.NewSpeakers,
		"이상",      // a model of a character guide
		"이상다침",    // the same person at another time of life
		"이상&파우스트", // a key a guide says is not that character
	)
	in.Plan.NewFilesDefaultFamily = append(in.Plan.NewFilesDefaultFamily,
		"StoryData/S1037B.json", // routing.yaml has a family for it now
	)
	in.Plan.Adjustments.Moved = append(in.Plan.Adjustments.Moved,
		plan.MovedKey{From: "StoryData/E002X.json|/dataList/1/content", To: "StoryData/E002X.json|/dataList/2/content"}, // the file says the new key already
	)
	in.Plan.Adjustments.Expired = append(in.Plan.Adjustments.Expired,
		"StoryData/E002X.json|/dataList/30/content", // the file does not have the adjustment any more
	)
	in.Adjustments = append(in.Adjustments, adjust.Adjustment{Key: "StoryData/E002X.json|/dataList/2/content"})

	want := ids(run(t, fixture()))
	if got := ids(run(t, in)); !slices.Equal(got, want) {
		t.Errorf("ids = %v, want the same as without the resolved ones: %v", got, want)
	}

	// Of those that are left, Exclude drops the ids it is given, and counts
	// only the ones that would have been suggested.
	got := run(t, in)
	kept, n := Exclude(got, map[string]bool{
		"speaker/새 화자":                    true, // from an exclude file
		"term/신규 용어":                      true, // from rules/declined.yaml
		"speaker/이상":                      true, // resolved above: nothing to drop
		"term/something that is not here": true,
	})
	if n != 2 || len(kept) != len(got)-2 {
		t.Fatalf("Exclude = %d kept, %d excluded; want %d kept, 2 excluded", len(kept), n, len(got)-2)
	}
	for _, s := range kept {
		if s.ID == "speaker/새 화자" || s.ID == "term/신규 용어" {
			t.Errorf("%s was not dropped", s.ID)
		}
	}
	if kept, n := Exclude(got, nil); n != 0 || !reflect.DeepEqual(kept, got) {
		t.Errorf("Exclude with nothing = %d excluded, %d kept; want everything kept", n, len(kept))
	}
	if kept, n := Exclude(nil, map[string]bool{"x": true}); kept == nil || len(kept) != 0 || n != 0 {
		t.Errorf("Exclude of nothing = %v, %d; want an empty list that is not nil", kept, n)
	}
}

// A default family of nothing (a routing without a catch-all, which lint
// rejects) does not make every new file a suggestion.
func TestSuggestWithoutACatchAllFamily(t *testing.T) {
	in := fixture()
	in.Rules.Routing.Families = in.Rules.Routing.Families[:1]
	for _, s := range run(t, in) {
		if s.Kind == KindFamily {
			t.Errorf("suggested %s without a catch-all family", s.ID)
		}
	}
}

// The plan and the build report say something about the snapshot they were
// made for. Of another one they say nothing about this snapshot.
func TestSuggestIgnoresAnotherRawTag(t *testing.T) {
	fromPlan := []string{KindSpeaker, KindField, KindFamily, KindAdjustmentMoved, KindAdjustmentExpired}
	fromBuild := []string{KindInconsistent, KindNote}
	kindsOf := func(list []Suggestion) []string {
		var kinds []string
		for _, s := range list {
			kinds = append(kinds, s.Kind)
		}
		return kinds
	}

	t.Run("plan", func(t *testing.T) {
		in := fixture()
		in.Plan.RawTag = "raw-20261029"
		got := kindsOf(run(t, in))
		for _, k := range fromPlan {
			if slices.Contains(got, k) {
				t.Errorf("a plan of another snapshot gave a %s suggestion", k)
			}
		}
		for _, k := range append(fromBuild, KindTerm) {
			if !slices.Contains(got, k) {
				t.Errorf("no %s suggestion; the plan's tag has nothing to do with it", k)
			}
		}
	})
	t.Run("build", func(t *testing.T) {
		in := fixture()
		in.Build.RawTag = "raw-20261029"
		got := kindsOf(run(t, in))
		for _, k := range fromBuild {
			if slices.Contains(got, k) {
				t.Errorf("a build of another snapshot gave a %s suggestion", k)
			}
		}
		for _, k := range append(fromPlan, KindTerm) {
			if !slices.Contains(got, k) {
				t.Errorf("no %s suggestion; the build's tag has nothing to do with it", k)
			}
		}
	})
	t.Run("neither", func(t *testing.T) {
		in := fixture()
		in.Plan, in.Build = nil, nil
		if got := ids(run(t, in)); !slices.Equal(got, []string{"term/신규 용어"}) {
			t.Errorf("ids = %v, want only the term from the glossary", got)
		}
	})
	t.Run("no current tag", func(t *testing.T) {
		in := fixture()
		in.RawTag, in.Plan.RawTag, in.Build.RawTag = "", "", ""
		if got := ids(run(t, in)); !slices.Equal(got, []string{"term/신규 용어"}) {
			t.Errorf("ids = %v, want nothing from a plan and a build that belong to no snapshot", got)
		}
	})
}

func TestSuggestWithoutRules(t *testing.T) {
	if _, err := Run(Inputs{RawTag: rawTag}); err == nil {
		t.Error("Run without rules = nil, want an error")
	}
}

// A candidate term is the only term that is suggested; one KR text with
// several entries (they apply to different files) is one suggestion, and the
// evidence is bounded.
func TestSuggestTermsAndBoundedEvidence(t *testing.T) {
	in := fixture()
	in.Rules.Glossary = nil
	for i := range 12 {
		in.Rules.Glossary = append(in.Rules.Glossary, rules.Term{
			KR: "다의어", ZH: fmt.Sprintf("譯法%02d", i), Status: rules.StatusCandidate, Files: []string{fmt.Sprintf("F%02d*", i)},
		})
	}
	in.Rules.Glossary = append(in.Rules.Glossary, rules.Term{KR: "승인됨", ZH: "已採用", Status: rules.StatusApproved})

	var terms []Suggestion
	for _, s := range run(t, in) {
		if s.Kind == KindTerm {
			terms = append(terms, s)
		}
	}
	if len(terms) != 1 || terms[0].ID != "term/다의어" {
		t.Fatalf("terms = %+v, want the one candidate KR text", terms)
	}
	if got := len(terms[0].Evidence); got != MaxEvidence {
		t.Errorf("%d evidence rows, want the bound of %d", got, MaxEvidence)
	}
	if first := terms[0].Evidence[0]; first.KR != "다의어" || first.ZH != "譯法00" {
		t.Errorf("first evidence row = %+v, want the entries in glossary order", first)
	}

	// Many translations of one text: the rows are cut at the bound too.
	in = fixture()
	inc := &in.Build.Report.Inconsistent[0]
	inc.ZH, inc.Keys = nil, nil
	for i := range 14 {
		inc.ZH = append(inc.ZH, fmt.Sprintf("譯%02d", i))
		inc.Keys = append(inc.Keys, fmt.Sprintf("F.json|/dataList/%d/name", i))
	}
	for _, s := range run(t, in) {
		if s.Kind == KindInconsistent && len(s.Evidence) != MaxEvidence {
			t.Errorf("inconsistent: %d evidence rows, want the bound of %d", len(s.Evidence), MaxEvidence)
		}
	}
}

// A note that is told twice (two batches) is one suggestion with one row of
// evidence, which holds the whole note as written; the title says it on one
// line, cut after 120 characters.
func TestSuggestNotes(t *testing.T) {
	in := fixture()
	long := "第一行\n\n   第二行 " + strings.Repeat("長", 200)
	in.Build.Report.Notes = []store.Note{
		{Key: noteKey, Note: long}, {Key: noteKey, Note: long, Batch: "later"},
		{Key: noteKey, Note: "another note on the same unit"},
	}
	var notes []Suggestion
	for _, s := range run(t, in) {
		if s.Kind == KindNote {
			notes = append(notes, s)
		}
	}
	if len(notes) != 2 {
		t.Fatalf("notes = %+v, want two (the repeated note is one)", notes)
	}
	for _, n := range notes {
		if len(n.Evidence) != 1 {
			t.Errorf("%s: evidence = %+v, want one row", n.ID, n.Evidence)
		}
		if strings.ContainsAny(n.Title, "\r\n") {
			t.Errorf("%s: the title is not one line: %q", n.ID, n.Title)
		}
	}
	byID := map[string]Suggestion{}
	for _, n := range notes {
		byID[n.ID] = n
	}
	longNote := byID["note/"+noteKey+"/"+sha8(long)]
	folded := []rune("第一行 第二行 " + strings.Repeat("長", 200))
	if want := "譯者備註（" + noteKey + "）：" + string(folded[:120]) + "…"; longNote.Title != want {
		t.Errorf("title = %q, want %q: the note on one line, cut after 120 characters", longNote.Title, want)
	}
	if want := []Evidence{{Key: noteKey, Note: long}}; !reflect.DeepEqual(longNote.Evidence, want) {
		t.Errorf("evidence = %+v, want the whole note as written", longNote.Evidence)
	}
	if short := byID["note/"+noteKey+"/"+sha8("another note on the same unit")]; !strings.HasSuffix(short.Title, "：another note on the same unit") {
		t.Errorf("title = %q, want the short note whole", short.Title)
	}
	for _, id := range []string{"note/" + noteKey + "/" + sha8(long), "note/" + noteKey + "/" + sha8("another note on the same unit")} {
		if !slices.Contains(ids(notes), id) {
			t.Errorf("ids = %v, want %s (the id is of the whole note)", ids(notes), id)
		}
	}
}

func TestSha8(t *testing.T) {
	for text, want := range map[string]string{
		"도약 모듈":    "fd93b875",
		"KR 這句有雙關": "3713708f",
		"":         "e3b0c442", // sha256 of nothing
	} {
		if got := sha8(text); got != want {
			t.Errorf("sha8(%q) = %q, want %q", text, got, want)
		}
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadDeclined(t *testing.T) {
	root := t.TempDir()
	if got, err := LoadDeclined(root); err != nil || len(got) != 0 {
		t.Errorf("without the file: LoadDeclined = %v, %v; want nothing", got, err)
	}

	writeFile(t, root, DeclinedFile, "[]\n")
	if got, err := LoadDeclined(root); err != nil || len(got) != 0 {
		t.Errorf("empty list: LoadDeclined = %v, %v; want nothing", got, err)
	}

	// An unquoted date is still the text of the date.
	writeFile(t, root, DeclinedFile, "# why not\n- id: speaker/아무개\n  reason: 路人，不需要指引\n  date: 2026-11-05\n- {id: 'term/전용 파편', reason: 之後再說, date: '2026-11-06'}\n")
	got, err := LoadDeclined(root)
	want := []Declined{
		{ID: "speaker/아무개", Reason: "路人，不需要指引", Date: "2026-11-05"},
		{ID: "term/전용 파편", Reason: "之後再說", Date: "2026-11-06"},
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("LoadDeclined = %+v, %v; want %+v", got, err, want)
	}

	// What lint would report about an entry does not stop the ids from being
	// used, but a file that cannot be read does.
	writeFile(t, root, DeclinedFile, "- id: speaker/x\n  date: nope\n")
	if got, err := LoadDeclined(root); err != nil || len(got) != 1 || got[0].ID != "speaker/x" {
		t.Errorf("an entry with a bad date: LoadDeclined = %+v, %v; want its id", got, err)
	}
	for name, text := range map[string]string{
		"not a list":    "id: speaker/x\n",
		"not a mapping": "- speaker/x\n",
		"unknown field": "- id: speaker/x\n  resaon: typo\n",
		"id not a text": "- id: [a]\n",
		"broken yaml":   "- id: [a\n",
	} {
		writeFile(t, root, DeclinedFile, text)
		if got, err := LoadDeclined(root); err == nil || !strings.Contains(err.Error(), DeclinedFile) {
			t.Errorf("%s: LoadDeclined = %v, %v; want an error that names the file", name, got, err)
		}
	}
}

func TestLintDeclined(t *testing.T) {
	root := t.TempDir()
	if got := LintDeclined(root); len(got) != 0 {
		t.Errorf("without the file: %v, want no issues", got)
	}
	writeFile(t, root, DeclinedFile, "[]\n")
	if got := LintDeclined(root); len(got) != 0 {
		t.Errorf("empty list: %v, want no issues", got)
	}
	var good strings.Builder
	for _, k := range Kinds {
		fmt.Fprintf(&good, "- {id: '%s/x', reason: r, date: 2026-11-05}\n", k)
	}
	writeFile(t, root, DeclinedFile, good.String())
	if got := LintDeclined(root); len(got) != 0 {
		t.Errorf("an entry of every kind: %v, want no issues", got)
	}

	// One problem each.
	writeFile(t, root, DeclinedFile, `
- {id: 'bogus/x', reason: r, date: 2026-11-05}
- {id: 'speaker/a', reason: '', date: 2026-11-05}
- {id: 'speaker/b', reason: r, date: 2026-13-45}
- {id: 'speaker/c', reason: r, date: 2026-11-05}
- {id: 'speaker/c', reason: other, date: 2026-11-06}
`)
	got := LintDeclined(root)
	wantIn := []string{"kind", "reason", "date", "twice"}
	if len(got) != len(wantIn) {
		t.Fatalf("issues = %v, want exactly %d", got, len(wantIn))
	}
	for i, want := range wantIn {
		if got[i].File != DeclinedFile || !strings.Contains(got[i].Message, want) {
			t.Errorf("issue %d = %+v, want one in %s about %q", i, got[i], DeclinedFile, want)
		}
	}
	if !strings.Contains(got[0].Message, "bogus") || !strings.Contains(got[3].Message, "speaker/c") {
		t.Errorf("the issues should name the id: %v", got)
	}

	// Missing and null fields, a date that is not padded or not a date, an id
	// with nothing after the kind.
	for name, tc := range map[string]struct{ text, want string }{
		"no id":         {"- {reason: r, date: 2026-11-05}\n", "id"},
		"null reason":   {"- {id: 'term/x', reason: , date: 2026-11-05}\n", "reason"},
		"blank reason":  {"- {id: 'term/x', reason: '  ', date: 2026-11-05}\n", "reason"},
		"no date":       {"- {id: 'term/x', reason: r}\n", "date"},
		"short date":    {"- {id: 'term/x', reason: r, date: '2026-1-5'}\n", "date"},
		"not a day":     {"- {id: 'term/x', reason: r, date: 2026-02-30}\n", "date"},
		"only the kind": {"- {id: 'term', reason: r, date: 2026-11-05}\n", "after the kind"},
		"empty rest":    {"- {id: 'term/', reason: r, date: 2026-11-05}\n", "after the kind"},
		"unknown field": {"- {id: 'term/x', reason: r, date: 2026-11-05, why: y}\n", "why"},
		"not a mapping": {"- term/x\n", "mapping"},
		"not a list":    {"id: term/x\n", "list"},
		"broken yaml":   {"- id: [x\n", "yaml"},
	} {
		writeFile(t, root, DeclinedFile, tc.text)
		got := LintDeclined(root)
		if len(got) != 1 || got[0].File != DeclinedFile || !strings.Contains(got[0].Message, tc.want) {
			t.Errorf("%s: issues = %v, want one in %s about %q", name, got, DeclinedFile, tc.want)
		}
	}
}

// The groups of different translations come from the list of the build record
// that has them all (Info.InconsistentAll), not from the twenty of the report:
// those that are declined or in a rules PR would otherwise hold their places
// for ever. A build.json of an older lct has no such list, and its report is
// read instead.
func TestSuggestReadsEveryInconsistentGroup(t *testing.T) {
	group := func(i int) build.Inconsistency {
		return build.Inconsistency{
			Family: "ally", Field: "name", KR: fmt.Sprintf("텍스트 %d", i),
			ZH: []string{"甲", "乙"}, Keys: []string{"a.json|/x", "b.json|/y"}, Units: 30 - i,
		}
	}
	var all []build.Inconsistency
	for i := range 25 {
		all = append(all, group(i))
	}
	count := func(list []Suggestion) int {
		n := 0
		for _, s := range list {
			if s.Kind == KindInconsistent {
				n++
			}
		}
		return n
	}

	in := fixture()
	in.Build.Report.Inconsistent = all[:20]
	in.Build.InconsistentAll = all
	if got := count(run(t, in)); got != 25 {
		t.Errorf("%d inconsistent suggestions, want the 25 of the build record", got)
	}
	in.Build.InconsistentAll = nil // an older build.json
	if got := count(run(t, in)); got != 20 {
		t.Errorf("%d inconsistent suggestions from an older build.json, want the 20 of its report", got)
	}
	in.Build.InconsistentAll = []build.Inconsistency{} // a build that found none
	if got := count(run(t, in)); got != 0 {
		t.Errorf("%d inconsistent suggestions from a build that found none, want 0", got)
	}
}
