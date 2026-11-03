package site

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/itemid"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/suggest"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

const (
	rvA     = "A.json"
	rvB     = "B.json"
	rvKR    = "하나"
	rvKana  = "시 히스클리프"
	rvKanaZ = "し協會 希斯克利夫"
)

func rvKey(file, ptr string) string { return units.Key(file, ptr) }

// withSha is a record made from kr, as a build keeps it.
func withSha(r store.Record, kr string) store.Record {
	r.KRSha = units.KRSha(kr)
	return r
}

// reviewInputs has three units of one KR text with two translations (A, B, A;
// k1 k2 in A.json and k3 in B.json), a kana warning in two units, a copy with
// kana, and a unit whose record is of another KR text.
//
//	A.json: 0 하나 -> 一   1 하나 -> 壹   2 시 히스클리프 -> し協會 希斯克利夫
//	        3 copy "か" (origin copy)    4 stale: KR 다섯, record of another KR, kana
//	B.json: 0 하나 -> 一   1 시 히스클리프 -> し協會 希斯克利夫
func reviewInputs(t *testing.T) (Inputs, []string) {
	t.Helper()
	fr, err := fields.Load("../..")
	if err != nil {
		t.Fatal(err)
	}
	const kanaCopy = "か"
	in := Inputs{
		KR: writePack(t, "KR_", map[string]string{
			rvA: `{"dataList":[{"id":1,"content":"하나"},{"id":2,"content":"하나"},{"id":3,"content":"시 히스클리프"},{"id":4,"content":"か"},{"id":5,"content":"다섯"}]}`,
			rvB: `{"dataList":[{"id":1,"content":"하나"},{"id":2,"content":"시 히스클리프"}]}`,
		}),
		EN: writePack(t, "EN_", map[string]string{
			rvA: `{"dataList":[{"id":1,"content":"one"},{"id":2,"content":"one"},{"id":3,"content":"Heathcliff"},{"id":4,"content":"ka"},{"id":5,"content":"five"}]}`,
			rvB: `{"dataList":[{"id":1,"content":"one"},{"id":2,"content":"Heathcliff"}]}`,
		}),
		JP: writePack(t, "JP_", map[string]string{
			rvA: `{"dataList":[{"id":1,"content":"いち"},{"id":2,"content":"いち"},{"id":3,"content":"ヒース"},{"id":4,"content":"か"},{"id":5,"content":"ご"}]}`,
		}),
		ZH: writePack(t, "", map[string]string{
			rvA: `{"dataList":[{"id":1,"content":"一"},{"id":2,"content":"壹"},{"id":3,"content":"し協會 希斯克利夫"},{"id":4,"content":"か"},{"id":5,"content":"五か"}]}`,
			rvB: `{"dataList":[{"id":1,"content":"一"},{"id":2,"content":"し協會 希斯克利夫"}]}`,
		}),
		Records: recordsOf(
			withSha(record(rvA, "/dataList/0/content", "一", store.OriginClaude), rvKR),
			withSha(record(rvA, "/dataList/1/content", "壹", store.OriginParatranz), rvKR),
			withSha(record(rvA, "/dataList/2/content", rvKanaZ, store.OriginClaude), rvKana),
			withSha(record(rvA, "/dataList/3/content", kanaCopy, store.OriginCopy), kanaCopy),
			withSha(record(rvA, "/dataList/4/content", "五か", store.OriginClaude), "예전 다섯"),
			withSha(record(rvB, "/dataList/0/content", "一", store.OriginClaude), rvKR),
			withSha(record(rvB, "/dataList/1/content", rvKanaZ, store.OriginHuman), rvKana),
		),
		Rules: testRules(),
	}
	in.Review = &ReviewInputs{
		RawTag: "t1",
		Fields: fr,
		Build: &build.Info{RawTag: "t1", InconsistentAll: []build.Inconsistency{{
			Family: "other", Field: "content", KR: rvKR,
			ZH: []string{"一", "壹"}, Keys: []string{rvKey(rvA, "/dataList/0/content"), rvKey(rvA, "/dataList/1/content")},
			Counts: []int{2, 1}, Units: 3,
		}}},
	}
	return in, []string{rvA, rvB}
}

func k(file string, i int) string { return rvKey(file, "/dataList/"+string(rune('0'+i))+"/content") }

func findSuggestion(t *testing.T, r Review, id string) SuggestionItem {
	t.Helper()
	for _, s := range r.Suggestions {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("no suggestion %s in %v", id, r.Suggestions)
	return SuggestionItem{}
}

var rvIncID = itemid.Inconsistent("other", "content", rvKR)
var rvKanaID = itemid.Warning("kana", rvKana, rvKanaZ)

func TestReviewRequiresInputs(t *testing.T) {
	in, files := reviewInputs(t)
	in.Review = nil
	if _, err := BuildReview(in, files); err == nil {
		t.Fatal("BuildReview with no Review returned no error")
	}
}

func TestReviewSuggestionsMatchSuggest(t *testing.T) {
	in, files := reviewInputs(t)
	rv := in.Review
	want, err := suggest.Run(suggest.Inputs{RawTag: rv.RawTag, Rules: in.Rules, Adjustments: rv.Adjustments, Build: rv.Build, Plan: rv.Plan})
	if err != nil {
		t.Fatal(err)
	}
	r, err := BuildReview(in, files)
	if err != nil {
		t.Fatal(err)
	}
	var got, ids []string
	for _, s := range r.Suggestions {
		got = append(got, s.ID)
	}
	for _, s := range want {
		ids = append(ids, s.ID)
	}
	if len(ids) == 0 || !slices.Equal(got, ids) {
		t.Errorf("ids = %v, want %v", got, ids)
	}
}

func TestReviewInconsistentListsEveryUnit(t *testing.T) {
	in, files := reviewInputs(t)
	r, err := BuildReview(in, files)
	if err != nil {
		t.Fatal(err)
	}
	s := findSuggestion(t, r, rvIncID)
	want := []Variant{
		{ZH: "一", Count: 2, Keys: []string{k(rvA, 0), k(rvB, 0)}},
		{ZH: "壹", Count: 1, Keys: []string{k(rvA, 1)}},
	}
	if !reflect.DeepEqual(s.Variants, want) {
		t.Errorf("Variants = %+v, want %+v", s.Variants, want)
	}
	if s.Family != "other" || s.Field != "content" || s.KR != rvKR {
		t.Errorf("family/field/kr = %q %q %q", s.Family, s.Field, s.KR)
	}
	if !same(s.JP, "いち") || !same(s.EN, "one") {
		t.Errorf("JP %s EN %s, want the first unit of the first variant", show(s.JP), show(s.EN))
	}
	if len(r.Mismatches) != 0 {
		t.Errorf("Mismatches = %v, want none", r.Mismatches)
	}
	if s.Declined != nil {
		t.Errorf("Declined = %+v, want nil", s.Declined)
	}
	for _, key := range []string{k(rvA, 0), k(rvA, 1), k(rvB, 0)} {
		if !slices.Contains(r.Refs[key], rvIncID) {
			t.Errorf("Refs[%s] = %v lacks the inconsistent id", key, r.Refs[key])
		}
	}
}

// A variant past the build's ten is listed though the evidence lacks it; the
// evidence variants still agree.
func TestReviewVariantsTheEvidenceLacks(t *testing.T) {
	in, files := reviewInputs(t)
	in.Review.Build.InconsistentAll[0].ZH = []string{"一"}
	in.Review.Build.InconsistentAll[0].Keys = in.Review.Build.InconsistentAll[0].Keys[:1]
	in.Review.Build.InconsistentAll[0].Counts = []int{2}
	r, err := BuildReview(in, files)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(findSuggestion(t, r, rvIncID).Variants); got != 2 {
		t.Errorf("%d variants, want both", got)
	}
	if len(r.Mismatches) != 0 {
		t.Errorf("Mismatches = %v, want none", r.Mismatches)
	}
}

func TestReviewRoutingChangedIsNotFatal(t *testing.T) {
	in, files := reviewInputs(t)
	// B.json moves to the story family: the group of the build is now two units.
	in.Rules = &rules.Rules{Routing: rules.Routing{
		Families: []rules.FamilyRule{{ID: "story", Match: []string{"B.json"}}, {ID: "other"}},
		Priority: []string{"story", "other"},
	}}
	r, err := BuildReview(in, files)
	if err != nil {
		t.Fatalf("a changed routing is an error: %v", err)
	}
	if !slices.Equal(r.Mismatches, []string{rvIncID}) {
		t.Errorf("Mismatches = %v, want [%s]", r.Mismatches, rvIncID)
	}
	s := findSuggestion(t, r, rvIncID)
	if len(s.Variants) != 2 || s.Variants[0].Count != 1 {
		t.Errorf("Variants = %+v, want what was found", s.Variants)
	}
}

func TestReviewWarningsLikeTheBuild(t *testing.T) {
	in, files := reviewInputs(t)
	r, err := BuildReview(in, files)
	if err != nil {
		t.Fatal(err)
	}
	var kana []WarningItem
	for _, w := range r.Warnings {
		if w.Code == "kana" {
			kana = append(kana, w)
		}
	}
	if len(kana) != 1 {
		t.Fatalf("kana warnings = %+v, want one (the copy and the stale record are not checked)", r.Warnings)
	}
	w := kana[0]
	if w.ID != rvKanaID || w.KR != rvKana || w.ZH != rvKanaZ || w.Message == "" {
		t.Errorf("warning = %+v", w)
	}
	if want := []string{k(rvA, 2), k(rvB, 1)}; !slices.Equal(w.Keys, want) {
		t.Errorf("Keys = %v, want %v", w.Keys, want)
	}
	if !slices.Equal(w.Families, []string{"other"}) || !slices.Equal(w.Origins, []string{"claude", "human"}) {
		t.Errorf("Families %v Origins %v", w.Families, w.Origins)
	}
	if w.Accepted != nil {
		t.Errorf("Accepted = %+v, want nil", w.Accepted)
	}
	for _, key := range w.Keys {
		if !slices.Contains(r.Refs[key], rvKanaID) {
			t.Errorf("Refs[%s] = %v lacks the warning", key, r.Refs[key])
		}
	}
	for _, key := range []string{k(rvA, 3), k(rvA, 4)} {
		for _, id := range r.Refs[key] {
			if strings.HasPrefix(id, "warning/") {
				t.Errorf("Refs[%s] = %v: a unit the build does not check", key, r.Refs[key])
			}
		}
	}
}

// A frozen file's record is used whatever its KR sha is.
func TestReviewFrozenFileChecksStaleRecord(t *testing.T) {
	in, files := reviewInputs(t)
	in.Review.Frozen = []string{rvA}
	r, err := BuildReview(in, files)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range r.Warnings {
		if w.ZH == "五か" && w.Code == "kana" {
			found = true
		}
	}
	if !found {
		t.Errorf("no kana warning of the record in the frozen file: %+v", r.Warnings)
	}
}

func TestReviewAcceptedWarningFlaggedNotInRefs(t *testing.T) {
	in, files := reviewInputs(t)
	in.Review.Accepted = []suggest.Declined{{ID: rvKanaID, Reason: "兩個都是名字", Date: "2026-11-02"}}
	r, err := BuildReview(in, files)
	if err != nil {
		t.Fatal(err)
	}
	var w *WarningItem
	for i := range r.Warnings {
		if r.Warnings[i].ID == rvKanaID {
			w = &r.Warnings[i]
		}
	}
	if w == nil || w.Accepted == nil || *w.Accepted != (Confirmed{Reason: "兩個都是名字", Date: "2026-11-02"}) {
		t.Fatalf("warning = %+v, want it flagged accepted", w)
	}
	for key, ids := range r.Refs {
		if slices.Contains(ids, rvKanaID) {
			t.Errorf("Refs[%s] has the accepted warning", key)
		}
	}

	// A changed translation is another warning, which is not accepted.
	in.Records[k(rvB, 1)] = withSha(record(rvB, "/dataList/1/content", "し協會 希斯", store.OriginHuman), rvKana)
	r, err = BuildReview(in, files)
	if err != nil {
		t.Fatal(err)
	}
	newID := itemid.Warning("kana", rvKana, "し協會 希斯")
	var seen bool
	for _, w := range r.Warnings {
		if w.ID == newID {
			seen = true
			if w.Accepted != nil {
				t.Errorf("the new warning %s is flagged accepted", w.ID)
			}
		}
	}
	if !seen {
		t.Errorf("no warning %s in %+v", newID, r.Warnings)
	}
}

func TestReviewDeclinedSuggestionFlaggedNotInRefs(t *testing.T) {
	in, files := reviewInputs(t)
	in.Review.Declined = []suggest.Declined{{ID: rvIncID, Reason: "都可以", Date: "2026-11-02"}}
	r, err := BuildReview(in, files)
	if err != nil {
		t.Fatal(err)
	}
	s := findSuggestion(t, r, rvIncID)
	if s.Declined == nil || *s.Declined != (Confirmed{Reason: "都可以", Date: "2026-11-02"}) {
		t.Errorf("Declined = %+v", s.Declined)
	}
	if len(s.Variants) != 2 {
		t.Errorf("a declined group still lists its variants, got %+v", s.Variants)
	}
	for key, ids := range r.Refs {
		if slices.Contains(ids, rvIncID) {
			t.Errorf("Refs[%s] has the declined group", key)
		}
	}
}

func TestReviewShapeIsStable(t *testing.T) {
	in, files := reviewInputs(t)
	r, err := BuildReview(in, files)
	if err != nil {
		t.Fatal(err)
	}
	if r.Notes == nil || r.Todo == nil || r.Mismatches == nil || r.Refs == nil || r.NoteText == nil || r.Queued == nil {
		t.Errorf("a field is nil: %+v", r)
	}
	for key, ids := range r.Refs {
		if !slices.IsSorted(ids) {
			t.Errorf("Refs[%s] = %v is not sorted", key, ids)
		}
	}
}
