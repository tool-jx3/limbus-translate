package site

import (
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/existing"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/suggest"
)

// NamesData is names.json: the table of existing names for the page that
// shows it.
type NamesData struct {
	RawTag string    `json:"raw_tag"` // the raw tag the table was made for; "" when there is none
	Fields []string  `json:"fields"`
	Names  []NameRow `json:"names"` // the order of the table (by KR)
}

// NameRow is one name of the table with what else the site knows about it.
type NameRow struct {
	KR       string             `json:"kr"`
	Units    int                `json:"units"` // every unit of every variant
	Variants []existing.Variant `json:"variants"`
	Glossary bool               `json:"glossary,omitempty"` // the glossary has an entry of KR (any status): the entry wins
	Declined bool               `json:"declined,omitempty"` // name/<kr> is in rules/declined.yaml
	Review   bool               `json:"review,omitempty"`   // name/<kr> is an unconfirmed suggestion of the review data
}

// buildNames makes names.json of the table t (nil for none). The whole table
// is exported; the flags say which names the glossary, the declined list and
// the review data already speak of.
func buildNames(t *existing.Table, glossary []rules.Term, declined []suggest.Declined, suggestions []SuggestionItem) NamesData {
	data := NamesData{Fields: []string{}, Names: []NameRow{}}
	if t == nil {
		return data
	}
	data.RawTag = t.RawTag
	if t.Fields != nil {
		data.Fields = t.Fields
	}
	inGlossary := make(map[string]bool, len(glossary))
	for _, g := range glossary {
		inGlossary[strings.TrimSpace(g.KR)] = true
	}
	isDeclined := make(map[string]bool, len(declined))
	for _, d := range declined {
		isDeclined[d.ID] = true
	}
	inReview := map[string]bool{}
	for _, s := range suggestions {
		if s.Declined == nil {
			inReview[s.ID] = true
		}
	}
	for _, n := range t.Names {
		if !existing.HasLetter(n.KR) { // an older names.json may still hold one
			continue
		}
		id := existing.SuggestionID(n.KR)
		row := NameRow{KR: n.KR, Variants: n.Variants, Glossary: inGlossary[strings.TrimSpace(n.KR)], Declined: isDeclined[id], Review: inReview[id]}
		if row.Variants == nil {
			row.Variants = []existing.Variant{}
		}
		for _, v := range n.Variants {
			for _, r := range v.Renderings {
				row.Units += r.Units
			}
		}
		data.Names = append(data.Names, row)
	}
	return data
}
