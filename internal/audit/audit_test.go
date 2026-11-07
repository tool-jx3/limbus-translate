package audit

import (
	"slices"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/itemid"
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
	a, b, c, d, e, f, g := k("A.json", 0), k("A.json", 1), k("A.json", 2), k("A.json", 3), k("A.json", 4), k("A.json", 5), k("A.json", 6)
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
				{Key: d}, {Key: e}, {Key: b}, {Key: g}, {Key: f, Accepted: &site.Confirmed{}},
			}},
		},
		Scope: scope.Report{Units: map[string][]scope.EntryUnit{
			"x": nil,
		}},
	}
	// Entry units come by the entry id of the term; the audit looks it up by KR and ZH.
	r.Scope.Units = map[string][]scope.EntryUnit{
		itemid.GlossaryEntry("침식", "侵蝕"): {
			{Key: d, Sense: scope.SenseSame},
			{Key: e, Sense: scope.SenseOther},
			{Key: b, Sense: scope.SenseOther, Explained: true},
			{Key: g, Sense: scope.SenseSame, Explained: true},
		},
	}
	recs := map[string]store.Record{}
	for _, key := range []string{a, b, c, d, e, f, g} {
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
		g: {1, []string{SignalTermMissing}}, // same sense but explained: not term-missing-same
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

func TestSetsAndReasons(t *testing.T) {
	cands := []Candidate{
		{Key: "a", Signals: []string{SignalInconsistent, SignalFormat}},
		{Key: "b", Signals: []string{SignalFormat}},
		{Key: "c", Signals: []string{SignalInconsistent, SignalFormat}},
	}
	sets := Sets(cands)
	if len(sets) != 2 || !slices.Equal(sets[0].Keys, []string{"a", "c"}) || !slices.Equal(sets[1].Keys, []string{"b"}) {
		t.Fatalf("sets = %+v, want {a c} then {b}", sets)
	}
	if got, want := sets[0].Reason(), "複查：同一 KR 在其他單位有不同譯法、格式警告；保留原措辭，只修正問題"; got != want {
		t.Errorf("reason = %q, want %q", got, want)
	}
	if got, want := sets[1].Reason(), "複查：格式警告；保留原措辭，只修正問題"; got != want {
		t.Errorf("reason = %q, want %q", got, want)
	}
	if got := sets[0].Topic(); got != "inconsistent-format" {
		t.Errorf("topic = %q", got)
	}
	for _, sig := range Signals {
		if reasons[sig] == "" {
			t.Errorf("signal %s has no reason", sig)
		}
	}
}

// variant builds a variant of n units of file f, starting at unit from, whose
// units all have the JP/EN pair (jp, en).
func variant(zh, f string, from, n int, jp, en string) site.Variant {
	v := site.Variant{ZH: zh, Count: n}
	g := site.JPENGroup{JP: &jp, EN: &en, Count: n}
	for i := from; i < from+n; i++ {
		v.Keys = append(v.Keys, k(f, i))
		g.Keys = append(g.Keys, k(f, i))
	}
	v.JPEN = []site.JPENGroup{g}
	return v
}

func inconsistent(class string, vs ...site.Variant) site.Review {
	return site.Review{Suggestions: []site.SuggestionItem{{Kind: suggest.KindInconsistent, JPENClass: class, Variants: vs}}}
}

func candidateKeys(r site.Review) []string {
	recs := map[string]store.Record{}
	for _, s := range r.Suggestions {
		for _, v := range s.Variants {
			for _, key := range v.Keys {
				recs[key] = rec(store.OriginClaude)
			}
		}
	}
	var out []string
	for _, c := range Candidates(in(r, recs)) {
		out = append(out, c.Key)
	}
	slices.Sort(out)
	return out
}

func TestInconsistentOnlyOutsideTheLargestVariant(t *testing.T) {
	// 3 to 1: only the single unit is signalled.
	r := inconsistent(site.JPENSame, variant("甲", "A.json", 0, 3, "j", "e"), variant("乙", "A.json", 3, 1, "j", "e"))
	if got := candidateKeys(r); !slices.Equal(got, []string{k("A.json", 3)}) {
		t.Errorf("3-to-1: candidates %v, want only the minority unit", got)
	}
	// 2 to 2: a tie signals every member.
	r = inconsistent(site.JPENSame, variant("甲", "A.json", 0, 2, "j", "e"), variant("乙", "A.json", 2, 2, "j", "e"))
	if got := candidateKeys(r); len(got) != 4 {
		t.Errorf("2-to-2: candidates %v, want all 4", got)
	}
	// A tie for largest with a smaller third: every member signals.
	r = inconsistent(site.JPENSame, variant("甲", "A.json", 0, 2, "j", "e"), variant("乙", "A.json", 2, 2, "j", "e"), variant("丙", "A.json", 4, 1, "j", "e"))
	if got := candidateKeys(r); len(got) != 5 {
		t.Errorf("2-2-1: candidates %v, want all 5", got)
	}
}

func TestInconsistentSkipsExplainedGroups(t *testing.T) {
	r := inconsistent(site.JPENExplained, variant("甲", "A.json", 0, 3, "j1", "e1"), variant("乙", "A.json", 3, 3, "j2", "e2"))
	if got := candidateKeys(r); len(got) != 0 {
		t.Errorf("explained group: candidates %v, want none", got)
	}
}

func TestInconsistentPartialSkipsVariantsThatFollowTheirJPEN(t *testing.T) {
	// Pair p has two renderings (甲, 乙); 丙 is the only rendering of pair q.
	// 甲 is the unique largest, 乙 is disputed, 丙 follows its JP/EN.
	r := inconsistent(site.JPENPartial,
		variant("甲", "A.json", 0, 4, "p", "p"),
		variant("乙", "A.json", 4, 2, "p", "p"),
		variant("丙", "A.json", 6, 3, "q", "q"),
	)
	if got := candidateKeys(r); !slices.Equal(got, []string{k("A.json", 4), k("A.json", 5)}) {
		t.Errorf("partial: candidates %v, want only the units of 乙", got)
	}
}

func TestInconsistentPartialWithoutJPENDataIsAnOrdinaryGroup(t *testing.T) {
	a, b := variant("甲", "A.json", 0, 3, "p", "p"), variant("乙", "A.json", 3, 1, "p", "p")
	a.JPEN, b.JPEN = nil, nil
	r := inconsistent(site.JPENPartial, a, b)
	if got := candidateKeys(r); !slices.Equal(got, []string{k("A.json", 3)}) {
		t.Errorf("partial without data: candidates %v, want the minority unit", got)
	}
}

func TestCandidatesExcludeHuman(t *testing.T) {
	a, b := k("A.json", 0), k("A.json", 1)
	r := site.Review{Warnings: []site.WarningItem{{Code: "ellipsis", Keys: []string{a, b}}}}
	got := Candidates(in(r, map[string]store.Record{a: rec(store.OriginHuman), b: rec(store.OriginClaude)}))
	if len(got) != 1 || got[0].Key != b {
		t.Fatalf("candidates = %+v, want only the non-human unit", got)
	}
}
