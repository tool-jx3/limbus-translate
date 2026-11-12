package existing

import (
	"sort"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// MinOldUnits is how many old units must hold a KR word before a provisional
// term that most of them do not use is taken to differ from the old
// translations.
const MinOldUnits = 5

// maxOldExamples is how many old units Use returns as examples.
const maxOldExamples = 3

// OldExample is an old unit: its key, its KR text and its translation.
type OldExample struct {
	Key string `json:"key"`
	KR  string `json:"kr"`
	ZH  string `json:"zh"`
}

// IsOld reports whether rec is a translation made before the update of
// rawTag: its origin is paratranz, claude or human, and it was not written
// by Submit at rawTag.
func IsOld(rec store.Record, rawTag string) bool {
	switch rec.Origin {
	case store.OriginParatranz, store.OriginClaude, store.OriginHuman:
		return !rec.TranslatedIn(rawTag)
	}
	return false
}

// oldHit is an old unit that holds the KR of a term.
type oldHit struct {
	OldExample
	with bool // the translation holds the term's ZH
}

// oldTerm is the tally of one term.
type oldTerm struct {
	term  rules.Term
	units int
	with  int
	hits  []oldHit
}

// OldTally counts, for some provisional terms, the old units that hold the
// KR of the term and those whose translation holds its ZH.
type OldTally struct {
	terms []*oldTerm
}

// NewOldTally returns a tally of terms, each by its KR and ZH. The KR is
// looked for as a word.
func NewOldTally(terms []rules.Term) *OldTally {
	t := &OldTally{}
	for _, term := range terms {
		term.Match = rules.MatchWord
		t.terms = append(t.terms, &oldTerm{term: term})
	}
	return t
}

// Add counts an old unit: its KR text and its translation.
func (t *OldTally) Add(key, kr, zh string) {
	plain := validate.MatchText(kr)
	text := validate.MatchText(zh)
	set := rules.NewRuneSet(plain)
	for _, o := range t.terms {
		if !set.MayStartTerm(o.term.KR) || !o.term.OccursIn(plain) {
			continue
		}
		with := o.term.ZH != "" && strings.Contains(text, o.term.ZH)
		o.units++
		if with {
			o.with++
		}
		o.hits = append(o.hits, oldHit{OldExample{Key: key, KR: kr, ZH: zh}, with})
	}
}

// Use says how the term kr -> zh stands with the old units. use holds the
// counts of the tally and, when the table of existing names has kr, its main
// rendering as Existing. differs is whether the term differs from the old
// translations: with a name in the table, when zh is none of its renderings;
// without, when at least MinOldUnits old units hold kr and fewer than half of
// them hold zh. examples are up to three old units that hold kr, those whose
// translation does not hold zh first, each group in key order.
func (t *OldTally) Use(ix *Index, kr, zh string) (use store.OldUse, examples []OldExample, differs bool) {
	var o *oldTerm
	for _, c := range t.terms {
		if c.term.KR == kr && c.term.ZH == zh {
			o = c
			break
		}
	}
	if o != nil {
		use.Units, use.With = o.units, o.with
	}
	if name := ix.Lookup(kr); name != nil {
		if len(name.Variants) > 0 && len(name.Variants[0].Renderings) > 0 {
			use.Existing = name.Variants[0].Renderings[0].ZH
		}
		differs = !name.Has(zh)
	} else {
		differs = use.Units >= MinOldUnits && use.With*2 < use.Units
	}
	if o != nil {
		hits := append([]oldHit(nil), o.hits...)
		sort.SliceStable(hits, func(i, j int) bool {
			if hits[i].with != hits[j].with {
				return !hits[i].with
			}
			return hits[i].Key < hits[j].Key
		})
		for _, h := range hits {
			if len(examples) == maxOldExamples {
				break
			}
			examples = append(examples, h.OldExample)
		}
	}
	return use, examples, differs
}
