// Package existing holds the table of existing names: the current
// translations of the name fields, grouped by KR name, so that a batch can
// show a name's earlier rendering and a check can tell when it was dropped.
package existing

import (
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/tool-jx3/limbus-translate/internal/names"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// NamesFile is the name of the table under .meta.
const NamesFile = "names.json"

// maxExamples is how many unit keys a variant keeps as examples.
const maxExamples = 3

// Table is the table of existing names, built from the translations of the
// name fields at one raw tag.
type Table struct {
	RawTag string   `json:"raw_tag"`
	Fields []string `json:"fields"`
	Names  []Name   `json:"names"` // sorted by KR
}

// Name is one KR name and the ways it has been rendered.
type Name struct {
	KR       string    `json:"kr"`       // trimmed
	Variants []Variant `json:"variants"` // most units first, then JP, then EN
}

// Variant is a KR name as the JP and EN packs spell it, with the renderings
// the translation gives it.
type Variant struct {
	JP         string      `json:"jp,omitempty"`
	EN         string      `json:"en,omitempty"`
	Renderings []Rendering `json:"renderings"` // most units first, then ZH; [0] is the main one
	Examples   []string    `json:"examples"`   // at most 3 keys
}

// Rendering is one translation of a name and the number of units that have it.
type Rendering struct {
	ZH    string `json:"zh"`
	Units int    `json:"units"`
}

// Source is a unit as Collect reads it. JP and EN are "" when none is aligned.
type Source struct{ Key, Field, KR, ZH, Origin, JP, EN string }

// Collect builds the table of the units whose Field is one of fields. Units
// copied, fallen back or without an origin are skipped, and so are those
// whose KR is shorter than two runes or whose translation is empty once its
// tags are gone, equals the KR or still holds Hangul. The result does not
// depend on the order of units.
func Collect(rawTag string, fields []string, units []Source) *Table {
	sorted := slices.Clone(units)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Key < sorted[j].Key })

	type rendering struct {
		units int
		keys  []string
	}
	type variant struct {
		jp, en string
		byZH   map[string]*rendering
		units  int
	}
	type entry struct {
		variants map[[2]string]*variant
	}
	byKR := map[string]*entry{}

	for _, u := range sorted {
		if !slices.Contains(fields, u.Field) {
			continue
		}
		switch u.Origin {
		case store.OriginCopy, store.OriginFallback, "":
			continue
		}
		kr := strings.TrimSpace(u.KR)
		if utf8.RuneCountInString(kr) < 2 {
			continue
		}
		zh := strings.TrimSpace(validate.StripTags(u.ZH))
		if zh == "" || zh == kr || names.HasHangul(zh) {
			continue
		}
		e := byKR[kr]
		if e == nil {
			e = &entry{variants: map[[2]string]*variant{}}
			byKR[kr] = e
		}
		jp, en := strings.TrimSpace(u.JP), strings.TrimSpace(u.EN)
		vk := [2]string{jp, strings.ToLower(en)}
		v := e.variants[vk]
		if v == nil {
			v = &variant{jp: jp, en: en, byZH: map[string]*rendering{}}
			e.variants[vk] = v
		}
		r := v.byZH[zh]
		if r == nil {
			r = &rendering{}
			v.byZH[zh] = r
		}
		r.units++
		r.keys = append(r.keys, u.Key)
		v.units++
	}

	t := &Table{RawTag: rawTag, Fields: slices.Clone(fields), Names: []Name{}}
	for kr, e := range byKR {
		type built struct {
			v     Variant
			units int
		}
		var vs []built
		for _, v := range e.variants {
			zhs := make([]string, 0, len(v.byZH))
			for zh := range v.byZH {
				zhs = append(zhs, zh)
			}
			sort.Slice(zhs, func(i, j int) bool {
				a, b := v.byZH[zhs[i]], v.byZH[zhs[j]]
				if a.units != b.units {
					return a.units > b.units
				}
				return zhs[i] < zhs[j]
			})
			out := Variant{JP: v.jp, EN: v.en, Examples: []string{}}
			for _, zh := range zhs {
				out.Renderings = append(out.Renderings, Rendering{ZH: zh, Units: v.byZH[zh].units})
				out.Examples = append(out.Examples, v.byZH[zh].keys...)
			}
			if len(out.Examples) > maxExamples {
				out.Examples = out.Examples[:maxExamples]
			}
			vs = append(vs, built{out, v.units})
		}
		sort.Slice(vs, func(i, j int) bool {
			a, b := vs[i], vs[j]
			if a.units != b.units {
				return a.units > b.units
			}
			if a.v.JP != b.v.JP {
				return a.v.JP < b.v.JP
			}
			return a.v.EN < b.v.EN
		})
		n := Name{KR: kr}
		for _, b := range vs {
			n.Variants = append(n.Variants, b.v)
		}
		t.Names = append(t.Names, n)
	}
	sort.Slice(t.Names, func(i, j int) bool { return t.Names[i].KR < t.Names[j].KR })
	return t
}

// LoadTable reads the table from .meta/names.json. It returns nil, nil when
// there is none.
func LoadTable(st *store.Store) (*Table, error) {
	var t Table
	found, err := st.ReadMetaJSON(NamesFile, &t)
	if err != nil || !found {
		return nil, err
	}
	return &t, nil
}

// SaveTable writes the table to .meta/names.json. Like the other state
// writers it does not take the lock.
func SaveTable(st *store.Store, t *Table) error {
	return st.WriteMetaJSON(NamesFile, t)
}

// SuggestionID is the id under which a declined suggestion about the name kr
// is recorded.
func SuggestionID(kr string) string { return "name/" + kr }

// Has reports whether zh, as a term is looked for in it, holds any rendering
// of any variant of n.
func (n *Name) Has(zh string) bool {
	text := validate.MatchText(zh)
	for _, v := range n.Variants {
		for _, r := range v.Renderings {
			if strings.Contains(text, r.ZH) {
				return true
			}
		}
	}
	return false
}
