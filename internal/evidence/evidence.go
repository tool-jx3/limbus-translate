// Package evidence gathers, for glossary entries, what the official packs say
// about them: the names the game gives the KR text, the descriptions of those
// names, and how often a candidate JP or EN rendering shows up in the aligned
// text of the units the entry applies to. It is evidence for a person (or the
// skill) filling jp, en and meaning of an entry; it decides nothing.
package evidence

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/align"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/names"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/units"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// A candidate is OK when it is in at least MinRate of at least MinN aligned
// units that hold the term.
const (
	MinN    = 10
	MinRate = 0.6
)

// The values of Check.Lang.
const (
	LangJP = "JP"
	LangEN = "EN"
)

// Name is what the packs call a KR text, over the units whose whole KR text is
// the entry's KR (after trimming spaces) and of a field the field rules
// translate: the aligned JP and EN, and the keys of those units.
type Name struct {
	JP    string   `json:"jp"`
	EN    string   `json:"en"`
	Keys  []string `json:"keys"`
	Count int      `json:"count"`
}

// Desc is the description (field desc) in the same dataList element as a name
// unit of a keyword file (names.IsKeywordFile).
type Desc struct {
	Key string `json:"key"`
	KR  string `json:"kr"`
	JP  string `json:"jp,omitempty"`
	EN  string `json:"en,omitempty"`
}

// Check says how often a candidate rendering (Candidate), or any form of a
// set of them (Values: a unit holds the set when it holds one of its forms),
// is in the aligned text of the units the entry applies to (see Collect). N
// counts those units whose KR text (as validate.MatchText leaves it) holds the
// term and that have aligned, non-empty text in the language, and are of a
// field the field rules translate; Hits those whose text holds the candidate
// (EN: ignoring case, at word boundaries; JP: as a substring; see Holds).
type Check struct {
	Lang      string   `json:"lang"`
	Candidate string   `json:"candidate,omitempty"`
	Values    []string `json:"values,omitempty"`
	N         int      `json:"n"`
	Hits      int      `json:"hits"`
	Rate      float64  `json:"rate"`
	OK        bool     `json:"ok"`
}

// Item is the evidence for one glossary entry. JP and EN are the entry's own.
type Item struct {
	KR     string      `json:"kr"`
	ZH     string      `json:"zh"`
	JP     rules.Forms `json:"jp,omitempty"`
	EN     rules.Forms `json:"en,omitempty"`
	Names  []Name      `json:"names"`
	Descs  []Desc      `json:"descs"`
	Checks []Check     `json:"checks"`
}

// Candidates are renderings to check beside the ones Collect finds itself.
type Candidates struct {
	JP rules.Forms `yaml:"jp" json:"jp,omitempty"`
	EN rules.Forms `yaml:"en" json:"en,omitempty"`
}

type userError string

func (e userError) Error() string { return string(e) }

// IsUserError reports whether err is the caller's mistake: a KR that is not in
// the glossary.
func IsUserError(err error) bool {
	var u userError
	return errors.As(err, &u)
}

// occurrence is a unit that holds a term, with the text aligned to it.
type occurrence struct {
	jp, en       string
	hasJP, hasEN bool
}

type nameKey struct{ jp, en string }

// nameAcc collects the names of one entry.
type nameAcc struct {
	by    map[nameKey]*Name
	order []nameKey
	descs []Desc
}

// Collect gathers the evidence for the entries of r whose KR is in krs, or for
// every approved entry when krs is empty, in glossary order. It reads each KR
// file of set once, aligns it once, and answers every entry from that pass.
// fr, the field rules, leave the units of fields that are not translated out
// of the checks. candidates (by KR) are checked beside the entry's own jp and
// en and the most frequent official names (see checksOf). A KR in krs or in
// candidates that is not in the glossary is an error for which IsUserError
// is true.
//
// The evidence of an entry is the units it applies to, as a batch sees them:
// those whose KR text it is a glossary hit of in their file
// (rules.Rules.GlossaryHits, which is batch.TermsIn without provisional
// terms: evidence reads no store). So files and not_files count, an entry
// with files stands for its KR text in its files and the entry without files
// in the others, and a longer entry that covers the term takes the unit (see
// rules.DropCovered). Its names and descs are of those units too. A candidate
// entry binds nothing and takes no unit from another; asked for by krs, it
// counts the units it would apply to if it were approved.
func Collect(r *rules.Rules, set pack.Set, fr *fields.Rules, krs []string, candidates map[string]Candidates) ([]Item, error) {
	known := map[string]bool{}
	for _, t := range r.Glossary {
		known[t.KR] = true
	}
	for _, kr := range krs {
		if !known[kr] {
			return nil, userError(fmt.Sprintf("evidence: %q is not a glossary entry", kr))
		}
	}
	for kr := range candidates {
		if !known[kr] {
			return nil, userError(fmt.Sprintf("evidence: candidates for %q, which is not a glossary entry", kr))
		}
	}

	sel := newSelection(r, krs)
	terms := sel.terms
	accs := make([]*nameAcc, len(terms))
	for i := range accs {
		accs[i] = &nameAcc{by: map[nameKey]*Name{}}
	}
	occs := make([][]occurrence, len(terms))

	for _, file := range set.KR.Files() {
		doc, err := set.KR.Doc(file)
		if err != nil {
			return nil, err
		}
		us := units.Extract(file, doc)
		fa := align.NewFileAligner(file, doc, set.EN, set.JP)
		for _, u := range us {
			trimmed := strings.TrimSpace(u.KR)
			if trimmed == "" {
				continue
			}
			hit := sel.hits(file, u.KR)
			if len(hit) == 0 {
				continue
			}
			en, jp, err := fa.Get(u.Pointer)
			if err != nil {
				return nil, err
			}
			// Only what is translated says anything about a rendering: a unit
			// of a field the field rules leave alone (an id or model, whose
			// aligned text is the KR id itself) counts neither in the checks
			// nor among the names, as lct glossary scope leaves it out. The
			// records are not read here, so a copy-origin unit stays counted;
			// the field rules alone decide.
			if !fr.Classify(u, en, jp).Translate {
				continue
			}
			o := alignedText(u.KR, en, jp)
			for _, i := range hit {
				occs[i] = append(occs[i], o)
				if trimmed != strings.TrimSpace(terms[i].KR) {
					continue
				}
				if err := accs[i].name(file, u, us, o, fa); err != nil {
					return nil, err
				}
			}
		}
	}

	items := make([]Item, 0, len(terms))
	for i, t := range terms {
		acc := accs[i]
		it := Item{KR: t.KR, ZH: t.ZH, JP: t.JP, EN: t.EN, Names: []Name{}, Descs: []Desc{}, Checks: []Check{}}
		for _, k := range acc.order {
			it.Names = append(it.Names, *acc.by[k])
		}
		slices.SortStableFunc(it.Names, func(a, b Name) int {
			return cmp.Or(cmp.Compare(b.Count, a.Count), cmp.Compare(a.JP, b.JP), cmp.Compare(a.EN, b.EN))
		})
		it.Descs = append(it.Descs, acc.descs...)
		given := candidates[t.KR]
		it.Checks = append(it.Checks, checksOf(LangJP, t.JP, mostCommon(it.Names, func(n Name) string { return n.JP }), given.JP, occs[i])...)
		it.Checks = append(it.Checks, checksOf(LangEN, t.EN, mostCommon(it.Names, func(n Name) string { return n.EN }), given.EN, occs[i])...)
		items = append(items, it)
	}
	return items, nil
}

// name adds the unit u of file, whose whole KR text is the entry's, to the
// names of the entry, o being its aligned text; the desc beside it in a
// keyword file goes to the descs.
func (acc *nameAcc) name(file string, u units.Unit, us []units.Unit, o occurrence, fa *align.FileAligner) error {
	// A name with no aligned text says nothing about JP or EN.
	if o.jp != "" || o.en != "" {
		k := nameKey{o.jp, o.en}
		n := acc.by[k]
		if n == nil {
			n = &Name{JP: o.jp, EN: o.en}
			acc.by[k] = n
			acc.order = append(acc.order, k)
		}
		n.Keys = append(n.Keys, units.Key(file, u.Pointer))
		n.Count++
	}
	if u.Field != "name" || u.ItemIndex < 0 || !names.IsKeywordFile(file) {
		return nil
	}
	// The desc beside the name: the same parent, last segment desc.
	parent, _ := strings.CutSuffix(u.Pointer, "name")
	for _, d := range us {
		if d.Pointer != parent+"desc" {
			continue
		}
		key := units.Key(file, d.Pointer)
		if slices.ContainsFunc(acc.descs, func(x Desc) bool { return x.Key == key }) {
			continue
		}
		den, djp, err := fa.Get(d.Pointer)
		if err != nil {
			return err
		}
		dt := alignedText(d.KR, den, djp)
		acc.descs = append(acc.descs, Desc{Key: key, KR: d.KR, JP: dt.jp, EN: dt.en})
	}
	return nil
}

// selection is the entries Collect gives the evidence of, and how to tell
// which of them a unit is evidence of.
type selection struct {
	r     *rules.Rules
	terms []rules.Term
	// byEntry are the places in terms of the approved entries, by entryKey.
	byEntry map[string][]int
	// promoted are the selected candidate entries: their place in terms, and
	// the rules with the entry approved.
	promoted []promotedEntry
}

type promotedEntry struct {
	at int
	r  *rules.Rules
}

// newSelection selects the entries of r whose KR is in krs, or every approved
// entry when krs is empty, in glossary order.
func newSelection(r *rules.Rules, krs []string) *selection {
	s := &selection{r: r, byEntry: map[string][]int{}}
	for gi, t := range r.Glossary {
		if t.KR == "" {
			continue
		}
		if len(krs) == 0 && t.Status != rules.StatusApproved || len(krs) > 0 && !slices.Contains(krs, t.KR) {
			continue
		}
		at := len(s.terms)
		s.terms = append(s.terms, t)
		if t.Status == rules.StatusApproved {
			s.byEntry[entryKey(t)] = append(s.byEntry[entryKey(t)], at)
			continue
		}
		sub := *r
		sub.Glossary = slices.Clone(r.Glossary)
		sub.Glossary[gi].Status = rules.StatusApproved
		s.promoted = append(s.promoted, promotedEntry{at: at, r: &sub})
	}
	return s
}

// hits returns the places in s.terms of the entries the KR text kr of file is
// evidence of, in order: the glossary hits of kr among the approved ones, and
// each candidate that would be a hit if it were approved.
func (s *selection) hits(file, kr string) []int {
	plain := validate.MatchText(kr)
	// Only a unit that holds a selected KR text can be a hit of one; most
	// hold none, and the glossary hits need not be looked up.
	if !slices.ContainsFunc(s.terms, func(t rules.Term) bool { return strings.Contains(plain, t.KR) }) {
		return nil
	}
	var out []int
	if len(s.byEntry) > 0 {
		for _, h := range s.r.GlossaryHits(file, kr) {
			out = append(out, s.byEntry[entryKey(h)]...)
		}
	}
	for _, p := range s.promoted {
		t := s.terms[p.at]
		if !strings.Contains(plain, t.KR) {
			continue
		}
		if slices.ContainsFunc(p.r.GlossaryHits(file, kr), func(h rules.Term) bool { return entryKey(h) == entryKey(t) }) {
			out = append(out, p.at)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// entryKey is what tells a glossary entry from the others of its KR text: a
// hit GlossaryHits returns is a copy of the entry it is.
func entryKey(t rules.Term) string {
	return strings.Join([]string{t.KR, t.ZH, t.Match, strings.Join(t.Files, "\x01"), strings.Join(t.NotFiles, "\x01")}, "\x00")
}

// checksOf are the checks of lang for an entry whose forms there are own,
// over occ: one for each candidate (the entry's own forms, the most common
// official name, the candidates given), then one for each set of forms (see
// formSets), whose hits are the units that hold any of its forms.
func checksOf(lang string, own rules.Forms, name string, given rules.Forms, occ []occurrence) []Check {
	var out []Check
	for _, c := range distinct(own, []string{name}, given) {
		out = append(out, check(lang, []string{c}, occ))
	}
	for _, set := range formSets(distinct(own), distinct(given)) {
		out = append(out, check(lang, set, occ))
	}
	return out
}

// formSets are the sets of forms checked as one: the entry's own forms, the
// candidates given, and both together (what the entry would have with the
// candidates added), in that order. A set of one form is the check of that
// form, and is left out; so is a set listed already (the same forms in any
// order).
func formSets(own, given []string) [][]string {
	var out [][]string
	add := func(set []string) {
		if len(set) < 2 {
			return
		}
		for _, o := range out {
			if len(o) == len(set) && !slices.ContainsFunc(set, func(v string) bool { return !slices.Contains(o, v) }) {
				return
			}
		}
		out = append(out, set)
	}
	add(own)
	add(given)
	if len(own) > 0 && len(given) > 0 {
		add(distinct(own, given))
	}
	return out
}

// alignedText is the text aligned to a unit whose KR text is kr. A text that
// is only tags and spaces (validate.MatchText) is no text: it is left out, as
// a check would not find anything in it either. So is a text that is the KR
// text itself (isCopy): the language was not translated, and the unit says
// nothing about its rendering. Checks, names and descs all read it here.
func alignedText(kr string, en, jp *align.LangAligned) occurrence {
	var o occurrence
	if en != nil && usable(kr, en.Value) {
		o.en, o.hasEN = strings.TrimSpace(en.Value), true
	}
	if jp != nil && usable(kr, jp.Value) {
		o.jp, o.hasJP = strings.TrimSpace(jp.Value), true
	}
	return o
}

// usable reports whether the aligned text says anything about a rendering of
// kr: it has text beyond tags and spaces, and is not a copy of kr.
func usable(kr, text string) bool {
	t := strings.TrimSpace(validate.MatchText(text))
	return t != "" && t != strings.TrimSpace(validate.MatchText(kr))
}

// mostCommon returns the value of field that most names have, counting units;
// the least such value breaks a tie. "" if no name has one.
func mostCommon(ns []Name, field func(Name) string) string {
	count := map[string]int{}
	for _, n := range ns {
		if v := field(n); v != "" {
			count[v] += n.Count
		}
	}
	best, bestN := "", 0
	for v, c := range count {
		if c > bestN || c == bestN && v < best {
			best, bestN = v, c
		}
	}
	return best
}

// distinct lists the non-empty values of the lists, trimmed, without
// repeats, in order. It is typed: a rules.Forms is a list of strings and is
// taken as one, and a single value has to be passed as a list.
func distinct(lists ...[]string) []string {
	var out []string
	for _, list := range lists {
		for _, s := range list {
			if s = strings.TrimSpace(s); s != "" && !slices.Contains(out, s) {
				out = append(out, s)
			}
		}
	}
	return out
}

// Holds reports whether the aligned text of lang holds the candidate rendering,
// as a check counts a hit: tags removed, EN by word, ignoring case and the
// number of the last word (rules.HoldsEN, which lint compares the EN forms of
// an entry by), JP as a substring. The scope analysis classifies units by it
// too.
func Holds(lang, text, candidate string) bool {
	text = validate.MatchText(text)
	if lang == LangEN {
		return rules.HoldsEN(text, candidate)
	}
	return strings.Contains(text, candidate)
}

// check counts, over occ, the units with aligned text in lang (N) and those
// whose text holds one of forms (Hits). One form is a Check of Candidate,
// more a Check of Values.
func check(lang string, forms []string, occ []occurrence) Check {
	c := Check{Lang: lang}
	if len(forms) == 1 {
		c.Candidate = forms[0]
	} else {
		c.Values = forms
	}
	for _, o := range occ {
		text, has := o.jp, o.hasJP
		if lang == LangEN {
			text, has = o.en, o.hasEN
		}
		if !has {
			continue
		}
		c.N++
		if slices.ContainsFunc(forms, func(f string) bool { return Holds(lang, text, f) }) {
			c.Hits++
		}
	}
	if c.N > 0 {
		c.Rate = float64(c.Hits) / float64(c.N)
	}
	c.OK = c.N >= MinN && c.Rate >= MinRate
	return c
}
