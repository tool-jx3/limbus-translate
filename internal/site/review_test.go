package site

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/itemid"
	"github.com/tool-jx3/limbus-translate/internal/plan"
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
	// The JP/EN groups of the variants are TestReviewJPENGroups' business.
	got := make([]Variant, len(s.Variants))
	for i, v := range s.Variants {
		got[i] = Variant{ZH: v.ZH, Count: v.Count, Keys: v.Keys}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Variants = %+v, want %+v", got, want)
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

// withBatch sets the batch of the record at key.
func withBatch(in Inputs, key, batch string) {
	r := in.Records[key]
	r.Batch = batch
	in.Records[key] = r
}

func noteOf(key, batch, kr, text string) store.Note {
	return store.Note{Key: key, Batch: batch, KRSha: units.KRSha(kr), Note: text}
}

func TestReviewNotes(t *testing.T) {
	t.Run("note on its unit", func(t *testing.T) {
		in, files := reviewInputs(t)
		withBatch(in, k(rvA, 0), "b1")
		in.Review.Notes = []store.Note{noteOf(k(rvA, 0), "b1", rvKR, "意譯")}
		r, err := BuildReview(in, files)
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Notes) != 1 || r.NotesUnplaced != 0 {
			t.Fatalf("Notes = %+v, unplaced %d", r.Notes, r.NotesUnplaced)
		}
		n := r.Notes[0]
		want := NoteItem{ID: itemid.Note(k(rvA, 0), "意譯"), Key: k(rvA, 0), Family: "other", Batch: "b1", Note: "意譯", KR: rvKR, ZH: "一", O: "claude"}
		if !reflect.DeepEqual(n, want) {
			t.Errorf("note = %+v, want %+v", n, want)
		}
		if r.NoteText[k(rvA, 0)] != "意譯" {
			t.Errorf("NoteText = %v", r.NoteText)
		}
		if !slices.Contains(r.Refs[k(rvA, 0)], want.ID) {
			t.Errorf("Refs = %v lacks the note", r.Refs)
		}
	})
	t.Run("note follows a moved record", func(t *testing.T) {
		in, files := reviewInputs(t)
		withBatch(in, k(rvA, 4), "b0")
		withBatch(in, k(rvA, 2), "b2")
		in.Review.Notes = []store.Note{noteOf(k(rvA, 4), "b2", rvKana, "雙關")}
		r, err := BuildReview(in, files)
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Notes) != 1 || r.Notes[0].Key != k(rvA, 2) {
			t.Fatalf("Notes = %+v", r.Notes)
		}
		if want := itemid.Note(k(rvA, 4), "雙關"); r.Notes[0].ID != want {
			t.Errorf("ID = %s, want %s (the key of notes.jsonl)", r.Notes[0].ID, want)
		}
		if r.NoteText[k(rvA, 2)] != "雙關" {
			t.Errorf("NoteText = %v", r.NoteText)
		}
	})
	t.Run("note with no place", func(t *testing.T) {
		in, files := reviewInputs(t)
		withBatch(in, k(rvA, 0), "b3")
		withBatch(in, k(rvA, 1), "b3")
		in.Review.Notes = []store.Note{
			noteOf(k(rvA, 4), "b3", rvKR, "兩個候選"),
			noteOf(k(rvA, 4), "b7", rvKR, "沒有候選"),
		}
		r, err := BuildReview(in, files)
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Notes) != 0 || r.NotesUnplaced != 2 {
			t.Errorf("Notes = %+v, unplaced %d", r.Notes, r.NotesUnplaced)
		}
	})
	t.Run("note of an older translation", func(t *testing.T) {
		in, files := reviewInputs(t)
		withBatch(in, k(rvA, 0), "b2")
		in.Review.Notes = []store.Note{noteOf(k(rvA, 0), "b1", rvKR, "舊的")}
		r, err := BuildReview(in, files)
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Notes) != 0 || r.NotesUnplaced != 1 || len(r.NoteText) != 0 {
			t.Errorf("Notes = %+v, unplaced %d, text %v", r.Notes, r.NotesUnplaced, r.NoteText)
		}
	})
	t.Run("note text kept when declined", func(t *testing.T) {
		in, files := reviewInputs(t)
		withBatch(in, k(rvA, 0), "b1")
		n1, n2 := noteOf(k(rvA, 0), "b1", rvKR, "甲"), noteOf(k(rvA, 0), "b1", rvKR, "乙")
		in.Review.Notes = []store.Note{n1, n2}
		in.Review.Declined = []suggest.Declined{{ID: itemid.Note(n1.Key, n1.Note), Reason: "不用改", Date: "2026-11-03"}}
		r, err := BuildReview(in, files)
		if err != nil {
			t.Fatal(err)
		}
		if r.NoteText[k(rvA, 0)] != "甲\n乙" {
			t.Errorf("NoteText = %q", r.NoteText[k(rvA, 0)])
		}
		if r.Notes[0].Declined == nil || r.Notes[1].Declined != nil {
			t.Errorf("Declined = %+v / %+v", r.Notes[0].Declined, r.Notes[1].Declined)
		}
		if refs := r.Refs[k(rvA, 0)]; slices.Contains(refs, r.Notes[0].ID) || !slices.Contains(refs, r.Notes[1].ID) {
			t.Errorf("Refs = %v", refs)
		}
	})
	t.Run("notes follow file, unit and batch order", func(t *testing.T) {
		in, files := reviewInputs(t)
		withBatch(in, k(rvA, 0), "b1")
		withBatch(in, k(rvB, 0), "b1")
		withBatch(in, k(rvA, 1), "b1")
		in.Review.Notes = []store.Note{
			noteOf(k(rvB, 0), "b1", rvKR, "B0"),
			noteOf(k(rvA, 1), "b1", rvKR, "A1"),
			noteOf(k(rvA, 0), "b1", rvKR, "A0"),
		}
		r, err := BuildReview(in, files)
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, n := range r.Notes {
			got = append(got, n.Note)
		}
		if !reflect.DeepEqual(got, []string{"A0", "A1", "B0"}) {
			t.Errorf("order = %v", got)
		}
	})
}

func TestReviewTodoHasFiveKinds(t *testing.T) {
	in, files := reviewInputs(t)
	in.Review.Plan = &plan.Summary{NewSpeakers: []string{"새 화자"}}
	in.Review.Pending = []store.PendingItem{{Key: k(rvA, 0), Flags: []string{store.FlagNeedsReview}, LastError: "boom"}}
	r, err := BuildReview(in, files)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Todo) != 1 || r.Todo[0].Kind != build.KindNeedsReview {
		t.Fatalf("Todo = %+v", r.Todo)
	}
	sec := r.Todo[0]
	if len(sec.Items) != 1 || sec.Items[0].Key != k(rvA, 0) || sec.Title == "" || sec.Hint == "" || sec.Items[0].Text == "" {
		t.Errorf("section = %+v", sec)
	}
	in.Review.Build = nil
	if r, err = BuildReview(in, files); err != nil || len(r.Todo) != 0 {
		t.Errorf("without a build: Todo = %+v, err %v", r.Todo, err)
	}
}

func TestReviewPendingCounted(t *testing.T) {
	in, files := reviewInputs(t)
	in.Review.Pending = []store.PendingItem{{Key: k(rvA, 2)}}
	withBatch(in, k(rvA, 2), "b2")
	in.Review.Notes = []store.Note{noteOf(k(rvA, 2), "b2", rvKana, "n")}
	r, err := BuildReview(in, files)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Queued[k(rvA, 2)] || len(r.Queued) != 1 {
		t.Errorf("Queued = %v", r.Queued)
	}
	for _, w := range r.Warnings {
		if w.ID == rvKanaID && w.Pending != 1 {
			t.Errorf("warning Pending = %d, want 1 of 2 units", w.Pending)
		}
	}
	if len(r.Notes) != 1 || r.Notes[0].Pending != 1 {
		t.Errorf("Notes = %+v", r.Notes)
	}
	if s := findSuggestion(t, r, rvIncID); s.Pending != 0 {
		t.Errorf("suggestion Pending = %d", s.Pending)
	}
	in.Review.Pending = []store.PendingItem{{Key: k(rvA, 0)}, {Key: k(rvB, 0)}}
	if r, err = BuildReview(in, files); err != nil {
		t.Fatal(err)
	}
	if s := findSuggestion(t, r, rvIncID); s.Pending != 2 {
		t.Errorf("suggestion Pending = %d, want 2", s.Pending)
	}
}

func TestReviewPendingAlwaysMarshalled(t *testing.T) {
	for _, v := range []any{SuggestionItem{}, WarningItem{}, NoteItem{}} {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), `"pending":0`) {
			t.Errorf("%T marshals to %s, want \"pending\":0", v, b)
		}
	}
}

func TestReviewNoteSuggestionNotListedTwice(t *testing.T) {
	placed := noteOf(k(rvA, 0), "b1", rvKR, "意譯")
	unplaced := noteOf(k(rvA, 0), "b9", rvKR, "找不到位置")
	in, files := reviewInputs(t)
	withBatch(in, k(rvA, 0), "b1")
	in.Review.Notes = []store.Note{placed}
	in.Review.Build.Report.Notes = []store.Note{placed, unplaced}
	r, err := BuildReview(in, files)
	if err != nil {
		t.Fatal(err)
	}
	placedID, unplacedID := itemid.Note(placed.Key, placed.Note), itemid.Note(unplaced.Key, unplaced.Note)
	if len(r.Notes) != 1 || r.Notes[0].ID != placedID {
		t.Fatalf("Notes = %+v", r.Notes)
	}
	var ids []string
	for _, s := range r.Suggestions {
		ids = append(ids, s.ID)
	}
	if slices.Contains(ids, placedID) {
		t.Errorf("the note %s is on the notes tab and in the suggestions too: %v", placedID, ids)
	}
	if !slices.Contains(ids, unplacedID) {
		t.Errorf("the unplaced note %s is gone from the suggestions: %v", unplacedID, ids)
	}
}

func jpenOf(t *testing.T, r Review, zh string) []JPENGroup {
	t.Helper()
	for _, v := range findSuggestion(t, r, rvIncID).Variants {
		if v.ZH == zh {
			return v.JPEN
		}
	}
	t.Fatalf("no variant %s", zh)
	return nil
}

// Each variant lists its units by their JP and EN; equal counts keep the
// order of the first unit.
func TestReviewJPENGroups(t *testing.T) {
	in, files := reviewInputs(t)
	r, err := BuildReview(in, files)
	if err != nil {
		t.Fatal(err)
	}
	one := jpenOf(t, r, "一")
	if len(one) != 2 || !same(one[0].JP, "いち") || !same(one[0].EN, "one") || one[0].Count != 1 ||
		!slices.Equal(one[0].Keys, []string{k(rvA, 0)}) || one[1].JP != nil || !same(one[1].EN, "one") ||
		!slices.Equal(one[1].Keys, []string{k(rvB, 0)}) {
		t.Errorf("一 JPEN = %+v, want (いち, one) [A0] then (—, one) [B0]", one)
	}
	if two := jpenOf(t, r, "壹"); len(two) != 1 || !same(two[0].JP, "いち") || two[0].Count != 1 {
		t.Errorf("壹 JPEN = %+v, want one group (いち, one)", two)
	}
}

// The class says whether JP and EN tell the translations apart: same (every
// unit has one JP/EN pair), explained (each pair has one translation),
// partial (some pair has two). A missing JP is a value of its own.
func TestReviewJPENClass(t *testing.T) {
	cases := []struct {
		name string
		jpB  string // the JP of file B ("" for none)
		enA1 string // the EN of /dataList/1 of file A, the unit of 壹
		want string
	}{
		{"a missing JP is a pair of its own", "", "one", "partial"},
		{"every unit has the same JP and EN", "いち", "one", "same"},
		{"each JP/EN pair has one translation", "いち", "One!", "explained"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in, files := reviewInputs(t)
			in.EN = writePack(t, "EN_", map[string]string{
				rvA: `{"dataList":[{"id":1,"content":"one"},{"id":2,"content":"` + c.enA1 + `"},{"id":3,"content":"Heathcliff"},{"id":4,"content":"ka"},{"id":5,"content":"five"}]}`,
				rvB: `{"dataList":[{"id":1,"content":"one"},{"id":2,"content":"Heathcliff"}]}`,
			})
			jp := map[string]string{rvA: `{"dataList":[{"id":1,"content":"いち"},{"id":2,"content":"いち"},{"id":3,"content":"ヒース"},{"id":4,"content":"か"},{"id":5,"content":"ご"}]}`}
			if c.jpB != "" {
				jp[rvB] = `{"dataList":[{"id":1,"content":"` + c.jpB + `"},{"id":2,"content":"ヒース"}]}`
			}
			in.JP = writePack(t, "JP_", jp)
			r, err := BuildReview(in, files)
			if err != nil {
				t.Fatal(err)
			}
			if got := findSuggestion(t, r, rvIncID).JPENClass; got != c.want {
				t.Errorf("JPENClass = %q, want %q", got, c.want)
			}
		})
	}
}

// A JP/EN group leaves out the language the units have no string in.
func TestReviewJPENMissingLanguage(t *testing.T) {
	in, files := reviewInputs(t)
	r, err := BuildReview(in, files)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(jpenOf(t, r, "一")[1])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"jp"`) || !strings.Contains(string(data), `"en":"one"`) {
		t.Errorf("JSON = %s, want no jp key and en one", data)
	}
}

// A used record without the glossary rendering is a term-missing warning on
// the review page, like the build counts it.
func TestReviewWarnsOnTermMissing(t *testing.T) {
	in, files := reviewInputs(t)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "guides/characters"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"guides/routing.yaml": "families:\n  - id: other\n    match: []\npriority:\n  - other\n",
		"glossary/terms.yaml": "- kr: 하나\n  zh: 壹\n  status: approved\n",
	} {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	rl, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	in.Rules = rl
	r, err := BuildReview(in, files)
	if err != nil {
		t.Fatal(err)
	}
	var got *WarningItem
	for i, w := range r.Warnings {
		if w.Code == "term-missing" {
			got = &r.Warnings[i]
		}
	}
	if got == nil || got.KR != rvKR || got.ZH != "壹" || len(got.Units) != 2 || got.Units[0].ZH != "一" {
		t.Fatalf("warnings = %+v, want the term-missing group of 하나 -> 壹", r.Warnings)
	}
}

// acceptedIDPattern is suggest's pattern of the id of a check warning.
var acceptedIDPattern = regexp.MustCompile(`^warning/[a-z0-9-]+/[0-9a-f]{8}/[0-9a-f]{8}$`)

// glossaryRules loads the rules of a repo with one catch-all family and the
// given glossary.
func glossaryRules(t *testing.T, glossary string) *rules.Rules {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "guides/characters"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"guides/routing.yaml": "families:\n  - id: other\n    match: []\npriority:\n  - other\n",
		"glossary/terms.yaml": glossary,
	} {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	rl, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return rl
}

// vendingInputs has three units that hold 자판기 and a fourth that renders it.
func vendingInputs(t *testing.T) (Inputs, []string) {
	t.Helper()
	fr, err := fields.Load("../..")
	if err != nil {
		t.Fatal(err)
	}
	krs := []string{"자판기 하나", "자판기 둘", "자판기 셋", "자판기 넷"}
	zhs := []string{"販賣機一", "販賣機二", "販賣機三", "自動販賣機四"}
	var kr, en, zh []string
	var recs []store.Record
	for i := range krs {
		ptr := "/dataList/" + string(rune('0'+i)) + "/content"
		kr = append(kr, `{"id":`+string(rune('1'+i))+`,"content":"`+krs[i]+`"}`)
		en = append(en, `{"id":`+string(rune('1'+i))+`,"content":"vending"}`)
		zh = append(zh, `{"id":`+string(rune('1'+i))+`,"content":"`+zhs[i]+`"}`)
		recs = append(recs, withSha(record(rvA, ptr, zhs[i], store.OriginClaude), krs[i]))
	}
	doc := func(items []string) string { return `{"dataList":[` + strings.Join(items, ",") + `]}` }
	in := Inputs{
		KR:      writePack(t, "KR_", map[string]string{rvA: doc(kr)}),
		EN:      writePack(t, "EN_", map[string]string{rvA: doc(en)}),
		JP:      writePack(t, "JP_", map[string]string{rvA: doc(en)}),
		ZH:      writePack(t, "", map[string]string{rvA: doc(zh)}),
		Records: recordsOf(recs...),
		Rules:   glossaryRules(t, "- kr: 자판기\n  zh: 自動販賣機\n  status: approved\n"),
	}
	in.Review = &ReviewInputs{RawTag: "t1", Fields: fr}
	return in, []string{rvA}
}

func termGroupOf(t *testing.T, r Review) WarningItem {
	t.Helper()
	var got []WarningItem
	for _, w := range r.Warnings {
		if w.Code == "term-missing" {
			got = append(got, w)
		}
	}
	if len(got) != 1 {
		t.Fatalf("term-missing items = %+v, want one group", got)
	}
	return got[0]
}

func TestReviewGroupsTermMissingByTerm(t *testing.T) {
	in, files := vendingInputs(t)
	r, err := BuildReview(in, files)
	if err != nil {
		t.Fatal(err)
	}
	w := termGroupOf(t, r)
	if w.ID != itemid.TermGroup("자판기", "自動販賣機") || acceptedIDPattern.MatchString(w.ID) {
		t.Errorf("ID = %s", w.ID)
	}
	if w.Term == nil || *w.Term != (TermRef{KR: "자판기", ZH: "自動販賣機"}) || w.KR != "자판기" || w.ZH != "自動販賣機" {
		t.Errorf("term = %+v, KR %q ZH %q", w.Term, w.KR, w.ZH)
	}
	if len(w.Units) != 3 || len(w.Keys) != 3 {
		t.Fatalf("units = %+v, keys = %v, want 3 each", w.Units, w.Keys)
	}
	for i, u := range w.Units {
		if u.Key != k(rvA, i) || u.ID != itemid.Warning("term-missing", u.KR, u.ZH) || !acceptedIDPattern.MatchString(u.ID) || u.Accepted != nil {
			t.Errorf("unit %d = %+v", i, u)
		}
	}
	for _, key := range w.Keys {
		if !slices.Contains(r.Refs[key], w.ID) {
			t.Errorf("Refs[%s] = %v, want the group", key, r.Refs[key])
		}
	}
}

func TestReviewTermGroupKeepsAcceptedUnits(t *testing.T) {
	in, files := vendingInputs(t)
	one := itemid.Warning("term-missing", "자판기 둘", "販賣機二")
	in.Review.Accepted = []suggest.Declined{{ID: one, Reason: "自動販賣機太長", Date: "2026-11-05"}}
	r, err := BuildReview(in, files)
	if err != nil {
		t.Fatal(err)
	}
	w := termGroupOf(t, r)
	if len(w.Units) != 3 || len(w.Keys) != 2 || slices.Contains(w.Keys, k(rvA, 1)) {
		t.Fatalf("keys = %v, units = %+v", w.Keys, w.Units)
	}
	if u := w.Units[1]; u.ID != one || u.Accepted == nil || u.Accepted.Reason != "自動販賣機太長" {
		t.Errorf("unit 1 = %+v, want it accepted", u)
	}
	if slices.Contains(r.Refs[k(rvA, 1)], w.ID) {
		t.Errorf("Refs of the accepted unit has the group: %v", r.Refs[k(rvA, 1)])
	}
}

func TestReviewTermGroupUnitsCarryNotes(t *testing.T) {
	in, files := vendingInputs(t)
	in.Review.Notes = []store.Note{{Key: k(rvA, 0), KRSha: units.KRSha("자판기 하나"), Note: "刻意簡稱"}}
	r, err := BuildReview(in, files)
	if err != nil {
		t.Fatal(err)
	}
	w := termGroupOf(t, r)
	if w.Units[0].Note != "刻意簡稱" || w.Units[1].Note != "" {
		t.Errorf("notes = %q, %q", w.Units[0].Note, w.Units[1].Note)
	}
}

func TestReviewNewTermEvidenceListsTheUnitsItHits(t *testing.T) {
	in, files := vendingInputs(t)
	in.Rules = glossaryRules(t, "[]\n")
	in.Review.Provisional = []store.ProvisionalTerm{{KR: "자판기", ZH: "販賣機", Key: k(rvA, 0), Batch: "b1", RawTag: "t1"}}
	// The fourth unit does not use the provisional rendering.
	in.ZH = writePack(t, "", map[string]string{rvA: `{"dataList":[{"id":1,"content":"販賣機一"},{"id":2,"content":"販賣機二"},{"id":3,"content":"販賣機三"},{"id":4,"content":"自動賣貨機四"}]}`})
	r, err := BuildReview(in, files)
	if err != nil {
		t.Fatal(err)
	}
	s := findSuggestion(t, r, "term-new/자판기")
	var hits []suggest.Evidence
	for _, e := range s.Evidence {
		if e.Note == "有用" || e.Note == "未用" {
			hits = append(hits, e)
		}
	}
	if len(hits) != 4 {
		t.Fatalf("evidence = %+v, want the four units it hits", s.Evidence)
	}
	for i, h := range hits {
		want := "有用"
		if i == 3 {
			want = "未用"
		}
		if h.Key != k(rvA, i) || h.Note != want || h.KR == "" || h.ZH == "" {
			t.Errorf("hit %d = %+v, want note %s", i, h, want)
		}
	}
}

// A group reads as confirmed once every unit is accepted, and counts as an
// open warning while one is not.
func TestReviewCountsATermGroupByItsUnits(t *testing.T) {
	in, files := vendingInputs(t)
	ids := []string{
		itemid.Warning("term-missing", "자판기 하나", "販賣機一"),
		itemid.Warning("term-missing", "자판기 둘", "販賣機二"),
		itemid.Warning("term-missing", "자판기 셋", "販賣機三"),
	}
	count := func(accepted ...string) (ReviewCounts, WarningItem) {
		in.Review.Accepted = nil
		for _, id := range accepted {
			in.Review.Accepted = append(in.Review.Accepted, suggest.Declined{ID: id, Reason: "簡稱", Date: "2026-11-05"})
		}
		r, err := BuildReview(in, files)
		if err != nil {
			t.Fatal(err)
		}
		return countReview(r), termGroupOf(t, r)
	}
	if c, w := count(); c.Warnings != 1 || w.Accepted != nil {
		t.Errorf("none accepted: counts %+v, accepted %v", c, w.Accepted)
	}
	if c, w := count(ids[:2]...); c.Warnings != 1 || w.Accepted != nil || len(w.Keys) != 1 {
		t.Errorf("partly accepted: counts %+v, accepted %v, keys %v", c, w.Accepted, w.Keys)
	}
	c, w := count(ids...)
	if c.Warnings != 0 || w.Accepted == nil || len(w.Keys) != 0 || w.Accepted.Reason != "簡稱" {
		t.Errorf("all accepted: counts %+v, accepted %v, keys %v", c, w.Accepted, w.Keys)
	}
}
