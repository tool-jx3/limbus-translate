// Package evidence gathers, for glossary entries, what the official packs say
// about them: the names the game gives the KR text, the descriptions of those
// names, and how often a candidate JP or EN rendering shows up in the aligned
// text of the units that hold the term. It is evidence for a person (or the
// skill) filling jp, en and meaning of an entry; it decides nothing.
package evidence

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tool-jx3/limbus-translate/internal/align"
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
// the entry's KR (after trimming spaces): the aligned JP and EN, and the keys
// of those units.
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

// Check says how often a candidate rendering is in the aligned text of the
// units that hold the term. N counts the units whose KR text (as
// validate.MatchText leaves it) holds the term by Term.OccursIn and that have
// aligned, non-empty text in the language; Hits those whose text holds the
// candidate (EN: ignoring case, at word boundaries; JP: as a substring).
type Check struct {
	Lang      string  `json:"lang"`
	Candidate string  `json:"candidate"`
	N         int     `json:"n"`
	Hits      int     `json:"hits"`
	Rate      float64 `json:"rate"`
	OK        bool    `json:"ok"`
}

// Item is the evidence for one glossary entry. JP and EN are the entry's own.
type Item struct {
	KR     string  `json:"kr"`
	ZH     string  `json:"zh"`
	JP     string  `json:"jp,omitempty"`
	EN     string  `json:"en,omitempty"`
	Names  []Name  `json:"names"`
	Descs  []Desc  `json:"descs"`
	Checks []Check `json:"checks"`
}

// Candidates are renderings to check beside the ones Collect finds itself.
type Candidates struct {
	JP []string `yaml:"jp" json:"jp,omitempty"`
	EN []string `yaml:"en" json:"en,omitempty"`
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

// nameAcc collects the names of one KR text.
type nameAcc struct {
	by    map[nameKey]*Name
	order []nameKey
	descs []Desc
}

// Collect gathers the evidence for the entries of r whose KR is in krs, or for
// every approved entry when krs is empty, in glossary order. It reads each KR
// file of set once, aligns it once, and answers every entry from that pass.
// candidates (by KR) are checked beside the entry's own jp and en and the most
// frequent official names. A KR in krs or in candidates that is not in the
// glossary is an error for which IsUserError is true.
func Collect(r *rules.Rules, set pack.Set, krs []string, candidates map[string]Candidates) ([]Item, error) {
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

	var terms []rules.Term
	for _, t := range r.Glossary {
		if t.KR == "" {
			continue
		}
		if len(krs) == 0 && t.Status != rules.StatusApproved || len(krs) > 0 && !slices.Contains(krs, t.KR) {
			continue
		}
		terms = append(terms, t)
	}

	accs := map[string]*nameAcc{} // by trimmed KR
	occs := make([][]occurrence, len(terms))
	for _, t := range terms {
		if accs[strings.TrimSpace(t.KR)] == nil {
			accs[strings.TrimSpace(t.KR)] = &nameAcc{by: map[nameKey]*Name{}}
		}
	}

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
			acc := accs[trimmed]
			var hit []int
			text := validate.MatchText(u.KR)
			for i, t := range terms {
				if strings.Contains(text, t.KR) && t.OccursIn(text) {
					hit = append(hit, i)
				}
			}
			if acc == nil && len(hit) == 0 {
				continue
			}
			en, jp, err := fa.Get(u.Pointer)
			if err != nil {
				return nil, err
			}
			o := alignedText(en, jp)
			for _, i := range hit {
				occs[i] = append(occs[i], o)
			}
			if acc == nil {
				continue
			}
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
			if u.Field == "name" && u.ItemIndex >= 0 && names.IsKeywordFile(file) {
				for _, d := range us {
					if d.ItemIndex != u.ItemIndex || d.Field != "desc" {
						continue
					}
					den, djp, err := fa.Get(d.Pointer)
					if err != nil {
						return nil, err
					}
					dt := alignedText(den, djp)
					acc.descs = append(acc.descs, Desc{Key: units.Key(file, d.Pointer), KR: d.KR, JP: dt.jp, EN: dt.en})
				}
			}
		}
	}

	items := make([]Item, 0, len(terms))
	for i, t := range terms {
		acc := accs[strings.TrimSpace(t.KR)]
		it := Item{KR: t.KR, ZH: t.ZH, JP: t.JP, EN: t.EN, Names: []Name{}, Descs: []Desc{}, Checks: []Check{}}
		for _, k := range acc.order {
			it.Names = append(it.Names, *acc.by[k])
		}
		slices.SortStableFunc(it.Names, func(a, b Name) int {
			return cmp.Or(cmp.Compare(b.Count, a.Count), cmp.Compare(a.JP, b.JP), cmp.Compare(a.EN, b.EN))
		})
		it.Descs = append(it.Descs, acc.descs...)

		given := candidates[t.KR]
		jps := distinct(t.JP, mostCommon(it.Names, func(n Name) string { return n.JP }), given.JP)
		ens := distinct(t.EN, mostCommon(it.Names, func(n Name) string { return n.EN }), given.EN)
		for _, c := range jps {
			it.Checks = append(it.Checks, check(LangJP, c, occs[i]))
		}
		for _, c := range ens {
			it.Checks = append(it.Checks, check(LangEN, c, occs[i]))
		}
		items = append(items, it)
	}
	return items, nil
}

// alignedText is the text aligned to a unit, tags removed as for a term.
func alignedText(en, jp *align.LangAligned) occurrence {
	var o occurrence
	if en != nil {
		o.en = strings.TrimSpace(en.Value)
	}
	if jp != nil {
		o.jp = strings.TrimSpace(jp.Value)
	}
	o.hasEN, o.hasJP = o.en != "", o.jp != ""
	return o
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

// distinct lists the non-empty values of the arguments (a string or a list),
// without repeats, in order.
func distinct(parts ...any) []string {
	var out []string
	add := func(s string) {
		if s = strings.TrimSpace(s); s != "" && !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	for _, p := range parts {
		switch v := p.(type) {
		case string:
			add(v)
		case []string:
			for _, s := range v {
				add(s)
			}
		}
	}
	return out
}

func check(lang, candidate string, occ []occurrence) Check {
	c := Check{Lang: lang, Candidate: candidate}
	for _, o := range occ {
		text, has := o.jp, o.hasJP
		if lang == LangEN {
			text, has = o.en, o.hasEN
		}
		if !has {
			continue
		}
		c.N++
		text = validate.MatchText(text)
		if lang == LangEN && containsWord(text, candidate) || lang == LangJP && strings.Contains(text, candidate) {
			c.Hits++
		}
	}
	if c.N > 0 {
		c.Rate = float64(c.Hits) / float64(c.N)
	}
	c.OK = c.N >= MinN && c.Rate >= MinRate
	return c
}

// containsWord reports whether text holds word, ignoring case, with no letter
// or digit right before or after it where it starts or ends with one.
func containsWord(text, word string) bool {
	text, word = strings.ToLower(text), strings.ToLower(word)
	if word == "" {
		return false
	}
	first, _ := utf8.DecodeRuneInString(word)
	last, _ := utf8.DecodeLastRuneInString(word)
	for from := 0; from < len(text); {
		i := strings.Index(text[from:], word)
		if i < 0 {
			return false
		}
		at := from + i
		from = at + 1
		if r, _ := utf8.DecodeLastRuneInString(text[:at]); at > 0 && isWordRune(r) && isWordRune(first) {
			continue
		}
		if r, _ := utf8.DecodeRuneInString(text[at+len(word):]); at+len(word) < len(text) && isWordRune(r) && isWordRune(last) {
			continue
		}
		return true
	}
	return false
}

func isWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' }
