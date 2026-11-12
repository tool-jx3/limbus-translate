package existing

import (
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// Index looks up the names of a Table. Every method returns the zero value
// on a nil receiver.
type Index struct {
	rawTag   string
	fields   []string
	byKR     map[string]*Name
	order    []*Name // longest KR first (by runes), then by KR
	declined map[string]bool
}

// NewIndex indexes t, leaving out each name that the glossary has an entry
// for, of any status. A nil t gives an empty index. declined holds the ids
// (see SuggestionID) of the suggestions that were declined.
func NewIndex(t *Table, r *rules.Rules, declined map[string]bool) *Index {
	ix := &Index{byKR: map[string]*Name{}, declined: declined}
	if t == nil {
		return ix
	}
	ix.rawTag, ix.fields = t.RawTag, t.Fields
	inGlossary := map[string]bool{}
	if r != nil {
		for _, g := range r.Glossary {
			inGlossary[strings.TrimSpace(g.KR)] = true
		}
	}
	for i := range t.Names {
		n := &t.Names[i]
		kr := strings.TrimSpace(n.KR)
		if kr == "" || inGlossary[kr] {
			continue
		}
		ix.byKR[kr] = n
		ix.order = append(ix.order, n)
	}
	sort.Slice(ix.order, func(i, j int) bool {
		a, b := ix.order[i], ix.order[j]
		if x, y := utf8.RuneCountInString(a.KR), utf8.RuneCountInString(b.KR); x != y {
			return x > y
		}
		return a.KR < b.KR
	})
	return ix
}

// RawTag is the raw tag the table was built at.
func (ix *Index) RawTag() string {
	if ix == nil {
		return ""
	}
	return ix.rawTag
}

// Fields are the fields the table was built from.
func (ix *Index) Fields() []string {
	if ix == nil {
		return nil
	}
	return ix.fields
}

// Lookup returns the name whose KR is kr, trimmed, or nil.
func (ix *Index) Lookup(kr string) *Name {
	if ix == nil {
		return nil
	}
	return ix.byKR[strings.TrimSpace(kr)]
}

// IsName reports whether field is a name field and kr is a known name.
func (ix *Index) IsName(field, kr string) bool {
	return ix != nil && slices.Contains(ix.fields, field) && ix.Lookup(kr) != nil
}

// Declined reports whether the suggestion about the name kr was declined.
func (ix *Index) Declined(kr string) bool {
	return ix != nil && ix.declined[SuggestionID(strings.TrimSpace(kr))]
}

// Mentions returns the names that kr mentions, longest first. A name counts
// when it stands as a word in kr (as a glossary term does), and either has
// three runes or more, holds a space, stands in quotes, or is in always. A
// name whose every hit lies inside a hit of a longer one is left out. The
// names point into the index; callers must not modify them.
func (ix *Index) Mentions(kr string, always map[string]bool) []*Name {
	if ix == nil || len(ix.order) == 0 {
		return nil
	}
	plain := validate.MatchText(kr)
	set := rules.NewRuneSet(plain)
	var hits []*Name
	var terms []rules.Term
	for _, n := range ix.order {
		if !set.MayStartTerm(n.KR) {
			continue
		}
		term := rules.Term{KR: n.KR, Match: rules.MatchWord}
		if !term.OccursIn(plain) {
			continue
		}
		if utf8.RuneCountInString(n.KR) < 3 && !strings.ContainsAny(n.KR, " \t　") &&
			!quoted(plain, n.KR) && !always[n.KR] {
			continue
		}
		hits = append(hits, n)
		terms = append(terms, term)
	}
	if len(hits) == 0 {
		return nil
	}
	kept := map[string]bool{}
	for _, t := range rules.DropCovered(plain, terms) {
		kept[t.KR] = true
	}
	out := hits[:0]
	for _, n := range hits {
		if kept[n.KR] {
			out = append(out, n)
		}
	}
	return out
}

// quoted reports whether kr stands in plain between a pair of quotes.
func quoted(plain, kr string) bool {
	for _, q := range [][2]string{{"'", "'"}, {"‘", "’"}, {"「", "」"}, {`"`, `"`}} {
		if strings.Contains(plain, q[0]+kr+q[1]) {
			return true
		}
	}
	return false
}
