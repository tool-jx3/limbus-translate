package audit

import (
	"slices"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/scope"
	"github.com/tool-jx3/limbus-translate/internal/site"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/suggest"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

func k(file string, n int) string { return file + "|/dataList/" + string(rune('0'+n)) + "/content" }

func rec(origin string) store.Record { return store.Record{ZH: "譯", Origin: origin} }

// family: "A" files are story, "B" files are skill; skill goes first.
func in(r site.Review, recs map[string]store.Record) Input {
	return Input{
		Review:  r,
		Records: recs,
		Frozen:  []string{"Frozen/**"},
		Family: func(f string) string {
			if len(f) > 0 && f[0] == 'B' {
				return "skill"
			}
			return "story"
		},
		Priority: func(f string) int {
			if f == "skill" {
				return 0
			}
			return 1
		},
	}
}

func TestCandidatesExcludeCopyPendingFrozen(t *testing.T) {
	keys := []string{k("A.json", 0), k("A.json", 1), k("A.json", 2), k("Frozen/F.json", 0), k("A.json", 3)}
	r := site.Review{Warnings: []site.WarningItem{{Code: "ellipsis", Keys: keys}}}
	recs := map[string]store.Record{
		keys[0]: rec(store.OriginClaude),
		keys[1]: rec(store.OriginCopy),
		keys[2]: rec(store.OriginClaude),
		keys[3]: rec(store.OriginClaude),
		// keys[4] has no record
	}
	i := in(r, recs)
	i.Pending = map[string]bool{keys[2]: true}
	got := Candidates(i)
	if len(got) != 1 || got[0].Key != keys[0] || got[0].Score != 1 || !slices.Equal(got[0].Signals, []string{SignalFormat}) {
		t.Fatalf("candidates = %+v, want only %s with format", got, keys[0])
	}
}

func TestSignalsAndWeights(t *testing.T) {
	a, b, c, d, e, f := k("A.json", 0), k("A.json", 1), k("A.json", 2), k("A.json", 3), k("A.json", 4), k("A.json", 5)
	r := site.Review{
		Suggestions: []site.SuggestionItem{
			{Kind: suggest.KindInconsistent, Variants: []site.Variant{{Keys: []string{a}}, {Keys: []string{b}}}},
			{Kind: suggest.KindInconsistent, Declined: &site.Confirmed{}, Variants: []site.Variant{{Keys: []string{f}}}},
			{Kind: suggest.KindNote, Evidence: nil},
		},
		Warnings: []site.WarningItem{
			{Code: validate.CodeLengthRatio, Keys: []string{a, c}},
			{Code: "ellipsis", Keys: []string{c}},
			{Code: "ellipsis", Keys: []string{f}, Accepted: &site.Confirmed{}},
			{Code: validate.CodeTermMissing, Term: &site.TermRef{KR: "침식", ZH: "侵蝕"}, Units: []site.TermUnit{
				{Key: d}, {Key: e}, {Key: b}, {Key: f, Accepted: &site.Confirmed{}},
			}},
		},
		Scope: scope.Report{Units: map[string][]scope.EntryUnit{
			"x": nil,
		}},
	}
	// Entry units come by the entry id of the term; the audit looks it up by KR and ZH.
	r.Scope.Units = map[string][]scope.EntryUnit{
		entryID("침식", "侵蝕"): {
			{Key: d, Sense: scope.SenseSame},
			{Key: e, Sense: scope.SenseOther},
			{Key: b, Sense: scope.SenseOther, Explained: true},
		},
	}
	recs := map[string]store.Record{}
	for _, key := range []string{a, b, c, d, e, f} {
		recs[key] = rec(store.OriginClaude)
	}
	got := map[string]Candidate{}
	for _, c := range Candidates(in(r, recs)) {
		got[c.Key] = c
	}
	want := map[string]struct {
		score   int
		signals []string
	}{
		a: {3 + 2, []string{SignalInconsistent, SignalLengthRatio}},
		b: {3, []string{SignalInconsistent}}, // explained: no term-missing
		c: {2 + 1, []string{SignalLengthRatio, SignalFormat}},
		d: {3, []string{SignalTermMissingSame}},
		e: {1, []string{SignalTermMissing}},
	}
	if len(got) != len(want) {
		t.Fatalf("candidates = %+v, want keys %v", got, want)
	}
	for key, w := range want {
		g := got[key]
		if g.Score != w.score || !slices.Equal(g.Signals, w.signals) {
			t.Errorf("%s = %d %v, want %d %v", key, g.Score, g.Signals, w.score, w.signals)
		}
	}
}

func TestOrder(t *testing.T) {
	// Same score: skill family (priority 0) before story; then by key.
	s1, s2, b1 := k("A.json", 2), k("A.json", 1), k("B.json", 9)
	hi := k("A.json", 7)
	r := site.Review{Warnings: []site.WarningItem{
		{Code: "ellipsis", Keys: []string{s1, s2, b1}},
		{Code: validate.CodeLengthRatio, Keys: []string{hi}},
	}}
	recs := map[string]store.Record{}
	for _, key := range []string{s1, s2, b1, hi} {
		recs[key] = rec(store.OriginClaude)
	}
	var got []string
	for _, c := range Candidates(in(r, recs)) {
		got = append(got, c.Key)
	}
	want := []string{hi, b1, s2, s1}
	if !slices.Equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestWaves(t *testing.T) {
	var cands []Candidate
	for i := 0; i < 5; i++ {
		fam := "story"
		if i == 0 {
			fam = "skill"
		}
		cands = append(cands, Candidate{Key: k("A.json", i), Family: fam, Score: 1, Signals: []string{SignalFormat}})
	}
	cands[1].Signals = []string{SignalLengthRatio, SignalFormat}
	usd := 0.02
	ws := Waves(cands, 2, &usd)
	if len(ws) != 3 || ws[0].Units != 2 || ws[1].Units != 2 || ws[2].Units != 1 {
		t.Fatalf("waves = %+v, want 2/2/1 units", ws)
	}
	w := ws[0]
	if w.Number != 1 || w.Families["skill"] != 1 || w.Families["story"] != 1 || w.Signals[SignalFormat] != 2 || w.Signals[SignalLengthRatio] != 1 {
		t.Errorf("wave 1 = %+v", w)
	}
	if w.EstimatedUSD == nil || *w.EstimatedUSD != 0.04 {
		t.Errorf("estimate = %v, want 0.04", w.EstimatedUSD)
	}
	if !slices.Equal(w.Keys, []string{cands[0].Key, cands[1].Key}) {
		t.Errorf("keys = %v", w.Keys)
	}
	if got := Waves(cands, 2, nil); got[0].EstimatedUSD != nil {
		t.Errorf("without a rate the estimate is %v, want nil", got[0].EstimatedUSD)
	}
	if got := Waves(nil, 2, &usd); len(got) != 0 {
		t.Errorf("no candidates give waves %v", got)
	}
}

func TestUSDPerUnit(t *testing.T) {
	for name, tc := range map[string]struct {
		cost          float64
		before, after int
		want          *float64
	}{
		"ratio":        {3, 100, 40, ptr(0.05)},
		"no progress":  {3, 100, 100, nil},
		"went up":      {3, 100, 120, nil},
		"nothing done": {0, 0, 0, nil},
	} {
		got := USDPerUnit(tc.cost, tc.before, tc.after)
		if (got == nil) != (tc.want == nil) || got != nil && *got != *tc.want {
			t.Errorf("%s: %v, want %v", name, got, tc.want)
		}
	}
}

func ptr(f float64) *float64 { return &f }

func TestReason(t *testing.T) {
	w := Wave{Signals: map[string]int{SignalFormat: 2, SignalInconsistent: 1, SignalLengthRatio: 4}}
	want := "複查：inconsistent、length-ratio、format；保留原措辭，只修正問題"
	if got := w.Reason(); got != want {
		t.Errorf("reason = %q, want %q", got, want)
	}
}
