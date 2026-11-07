// Package audit ranks the translations that exist by quality signals, splits
// them into review waves and estimates what a wave costs to translate again.
// It only reads what the review of the last build already knows
// (site.BuildReview: the suggestions, the check warnings and the sense of each
// unit that lacks a glossary rendering); it changes no state.
package audit

import (
	"cmp"
	"slices"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/itemid"
	"github.com/tool-jx3/limbus-translate/internal/scope"
	"github.com/tool-jx3/limbus-translate/internal/site"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/suggest"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// The signals, in the order a unit lists them (heaviest first).
const (
	// SignalInconsistent: the unit is in a group of inconsistent translations,
	// outside the group's unique largest variant and not explained by JP/EN
	// (signalled).
	SignalInconsistent = "inconsistent"
	// SignalTermMissingSame: the unit lacks a glossary rendering although
	// every available EN/JP alignment holds the entry's, so they agree
	// (scope.SenseSame), and no note explains it.
	SignalTermMissingSame = "term-missing-same"
	// SignalLengthRatio: the length-ratio warning.
	SignalLengthRatio = "length-ratio"
	// SignalTermMissing: the unit lacks a rendering, in another sense or
	// without alignment and its note does not explain it, or in the same
	// sense with a note that does.
	SignalTermMissing = "term-missing"
	// SignalFormat: any other check warning.
	SignalFormat = "format"
)

// Signals lists the signals by weight, then in the order they are named.
var Signals = []string{SignalInconsistent, SignalTermMissingSame, SignalLengthRatio, SignalTermMissing, SignalFormat}

// Weights are the points a signal adds to the score of a unit.
var Weights = map[string]int{
	SignalInconsistent:    3,
	SignalTermMissingSame: 3,
	SignalLengthRatio:     2,
	SignalTermMissing:     1,
	SignalFormat:          1,
}

// Input is what Candidates looks at.
type Input struct {
	Review  site.Review
	Records map[string]store.Record
	// Pending holds the keys of the pending list; Frozen the frozen patterns
	// of lct.yaml.
	Pending map[string]bool
	Frozen  []string
	// Family returns the family of a file; Priority the place of a family in
	// the translation order, smaller first (rules.Family, rules.Priority).
	Family   func(file string) string
	Priority func(family string) int
}

// Candidate is one unit worth a second look.
type Candidate struct {
	Key     string   `json:"key"`
	Family  string   `json:"family"`
	Score   int      `json:"score"`
	Signals []string `json:"signals"`
}

// Candidates returns the units that have at least one signal, best first: the
// higher score, then the family translated first, then the key. A copy or
// human record, a unit without a record, a pending unit and a unit of a frozen file is never
// a candidate. A suggestion that was declined and a warning that was accepted
// (or a unit of a term group that was) are confirmed by a person: no signal.
func Candidates(in Input) []Candidate {
	has := map[string]map[string]bool{}
	add := func(key, signal string) {
		if has[key] == nil {
			has[key] = map[string]bool{}
		}
		has[key][signal] = true
	}

	for _, s := range in.Review.Suggestions {
		if s.Kind != suggest.KindInconsistent || s.Declined != nil {
			continue
		}
		for _, v := range signalled(s) {
			for _, key := range v.Keys {
				add(key, SignalInconsistent)
			}
		}
	}
	for _, w := range in.Review.Warnings {
		switch {
		case w.Term != nil:
			senses := map[string]scope.EntryUnit{}
			for _, u := range in.Review.Scope.Units[itemid.GlossaryEntry(w.Term.KR, w.Term.ZH)] {
				senses[u.Key] = u
			}
			for _, u := range w.Units {
				if u.Accepted != nil {
					continue
				}
				// A note that names the term explains the miss: the unit is
				// then term-missing at most, even when the sense is the same.
				switch eu := senses[u.Key]; {
				case eu.Sense == scope.SenseSame && !eu.Explained:
					add(u.Key, SignalTermMissingSame)
				case eu.Sense == scope.SenseSame || !eu.Explained:
					add(u.Key, SignalTermMissing)
				}
			}
		case w.Accepted != nil:
		default:
			signal := SignalFormat
			if w.Code == validate.CodeLengthRatio {
				signal = SignalLengthRatio
			}
			for _, key := range w.Keys {
				add(key, signal)
			}
		}
	}

	var out []Candidate
	for key, set := range has {
		rec, ok := in.Records[key]
		file, _, _ := strings.Cut(key, "|")
		if !ok || rec.Origin == store.OriginCopy || rec.Origin == store.OriginHuman || in.Pending[key] || config.MatchAny(in.Frozen, file) {
			continue
		}
		c := Candidate{Key: key, Family: in.Family(file)}
		for _, s := range Signals {
			if set[s] {
				c.Signals = append(c.Signals, s)
				c.Score += Weights[s]
			}
		}
		out = append(out, c)
	}
	slices.SortFunc(out, func(a, b Candidate) int {
		return cmp.Or(
			cmp.Compare(b.Score, a.Score),
			cmp.Compare(in.Priority(a.Family), in.Priority(b.Family)),
			cmp.Compare(a.Key, b.Key),
		)
	})
	return out
}

// signalled returns the variants of a group of inconsistent translations
// whose units get the signal. The unique largest variant is the form the
// group agrees on, so it does not (on a tie for largest every tied variant
// does). A group that JP/EN explain (site.JPENExplained) gives no signal, and
// in a partial group the variants that follow their JP/EN do not: those whose
// every JP/EN pair has no other translation. Without per-variant JP/EN data a
// partial group is an ordinary one.
func signalled(s site.SuggestionItem) []site.Variant {
	if s.JPENClass == site.JPENExplained {
		return nil
	}
	size := func(v site.Variant) int { return max(v.Count, len(v.Keys)) }
	largest, ties := 0, 0
	for _, v := range s.Variants {
		switch n := size(v); {
		case n > largest:
			largest, ties = n, 1
		case n == largest:
			ties++
		}
	}
	var owners map[string]int // JP/EN pair -> the number of variants that have it
	if s.JPENClass == site.JPENPartial {
		owners = map[string]int{}
		for _, v := range s.Variants {
			for _, p := range pairs(v) {
				owners[p]++
			}
		}
	}
	var out []site.Variant
	for _, v := range s.Variants {
		if ties == 1 && size(v) == largest {
			continue
		}
		if owners != nil && followsJPEN(v, owners) {
			continue
		}
		out = append(out, v)
	}
	return out
}

// pairs lists the JP/EN pairs of v as map keys; a missing language differs
// from an empty string.
func pairs(v site.Variant) []string {
	part := func(p *string) string {
		if p == nil {
			return "-"
		}
		return "+" + *p
	}
	var out []string
	for _, g := range v.JPEN {
		out = append(out, part(g.JP)+"\x00"+part(g.EN))
	}
	return out
}

// followsJPEN reports whether v has JP/EN data and no pair of it is shared
// with another variant.
func followsJPEN(v site.Variant, owners map[string]int) bool {
	ps := pairs(v)
	for _, p := range ps {
		if owners[p] > 1 {
			return false
		}
	}
	return len(ps) > 0
}

// Wave is one slice of the ranked candidates.
type Wave struct {
	// Number counts from 1.
	Number int `json:"wave"`
	Units  int `json:"units"`
	// Families and Signals count the units of the wave by family and by
	// signal (a unit counts once for each signal it has).
	Families map[string]int `json:"families"`
	Signals  map[string]int `json:"signals"`
	// EstimatedUSD is Units times the rate; nil without a rate.
	EstimatedUSD *float64 `json:"estimated_usd"`
	// Keys are the units of the wave, in rank order.
	Keys []string `json:"keys,omitempty"`
}

// Waves splits cands (already ranked) into waves of at most size units;
// usdPerUnit may be nil.
func Waves(cands []Candidate, size int, usdPerUnit *float64) []Wave {
	waves := []Wave{}
	for i := 0; i < len(cands); i += size {
		part := cands[i:min(i+size, len(cands))]
		w := Wave{Number: len(waves) + 1, Units: len(part), Families: map[string]int{}, Signals: map[string]int{}, Keys: []string{}}
		for _, c := range part {
			w.Families[c.Family]++
			for _, s := range c.Signals {
				w.Signals[s]++
			}
			w.Keys = append(w.Keys, c.Key)
		}
		if usdPerUnit != nil {
			est := float64(w.Units) * *usdPerUnit
			w.EstimatedUSD = &est
		}
		waves = append(waves, w)
	}
	return waves
}

// USDPerUnit is cost / (before - after), the cost of one unit translated by
// the last translation run; nil when it cannot be told (nothing was done).
func USDPerUnit(cost float64, before, after int) *float64 {
	if before-after <= 0 {
		return nil
	}
	v := cost / float64(before-after)
	return &v
}

// SignalSet is the units of a wave that have the same signals.
type SignalSet struct {
	Signals []string
	Keys    []string
}

// Sets splits cands, the units of one wave, by their signals; the sets come in
// the order of their first unit.
func Sets(cands []Candidate) []SignalSet {
	var out []SignalSet
	at := map[string]int{}
	for _, c := range cands {
		id := strings.Join(c.Signals, " ")
		i, ok := at[id]
		if !ok {
			i = len(out)
			at[id] = i
			out = append(out, SignalSet{Signals: c.Signals})
		}
		out[i].Keys = append(out[i].Keys, c.Key)
	}
	return out
}

// reasons says each signal in a few words for the translator of a requeue.
var reasons = map[string]string{
	SignalInconsistent:    "同一 KR 在其他單位有不同譯法",
	SignalTermMissingSame: "缺詞彙表譯名，EN/JP 顯示同義",
	SignalTermMissing:     "缺詞彙表譯名",
	SignalLengthRatio:     "譯文長度與 KR 比例異常",
	SignalFormat:          "格式警告",
}

// Reason is the hint a requeue request for the units of the set carries: it
// names only the signals of the set.
func (s SignalSet) Reason() string {
	var names []string
	for _, sig := range s.Signals {
		names = append(names, reasons[sig])
	}
	return "複查：" + strings.Join(names, "、") + "；保留原措辭，只修正問題"
}

// Topic is the suffix of the file name of the set's request: its signals.
func (s SignalSet) Topic() string { return strings.Join(s.Signals, "-") }
