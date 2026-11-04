package rules

import (
	"fmt"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// GlossaryFile is the glossary, relative to the repo root.
const GlossaryFile = "glossary/terms.yaml"

// Term is one entry of glossary/terms.yaml.
type Term struct {
	KR string `yaml:"kr" json:"kr"`
	JP string `yaml:"jp" json:"jp,omitempty"`
	EN string `yaml:"en" json:"en,omitempty"`
	ZH string `yaml:"zh" json:"zh"`
	// Status is "approved" or "candidate".
	Status string `yaml:"status" json:"status"`
	Notes  string `yaml:"notes" json:"notes,omitempty"`
	// Forbidden lists renderings the translation must not contain; an entry
	// starting with "re:" is a regular expression.
	Forbidden []string `yaml:"forbidden" json:"forbidden,omitempty"`
	Category  []string `yaml:"category" json:"category,omitempty"`
	// Files are the files the term applies to (see config.MatchAny); empty
	// means every file.
	Files []string `yaml:"files" json:"files,omitempty"`
}

// ToValidate returns the part of the term that validate.Check enforces.
func (t Term) ToValidate() validate.Term {
	return validate.Term{KR: t.KR, Forbidden: t.Forbidden}
}

// termFields are the keys an entry may have; "evidence" is statistics for
// people and is not loaded.
var termFields = map[string]bool{
	"kr": true, "jp": true, "en": true, "zh": true, "status": true,
	"forbidden": true, "category": true, "evidence": true, "notes": true, "files": true,
}

// parseGlossary decodes the glossary. unknown[i] lists the keys of entry i
// that are not term fields; Load ignores them, Lint reports them.
func parseGlossary(data []byte) (terms []Term, unknown [][]string, err error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, nil, err
	}
	if doc.Kind == 0 { // empty file
		return nil, nil, nil
	}
	list := doc.Content[0]
	if list.Kind != yaml.SequenceNode {
		return nil, nil, fmt.Errorf("line %d: the glossary must be a list of entries", list.Line)
	}
	for _, entry := range list.Content {
		if entry.Kind != yaml.MappingNode {
			return nil, nil, fmt.Errorf("line %d: a glossary entry must be a mapping", entry.Line)
		}
		unknown = append(unknown, unknownKeys(entry, termFields))
	}
	if err := list.Decode(&terms); err != nil {
		return nil, nil, err
	}
	return terms, unknown, nil
}

// The two Status values of a term. An approved term binds; a candidate term
// is only a proposal and binds nothing until a person approves it.
const (
	StatusApproved  = "approved"
	StatusCandidate = "candidate"
)

// GlossaryHits returns the approved terms whose KR text occurs in kr, a text
// of file, in glossary order. Candidate terms stay in Rules.Glossary but are
// never returned, so they are neither injected into a batch nor enforced by
// validation.
//
// A term with Files applies to the files that match one of them (see
// config.MatchAny), a term without to every file. Of several entries for one
// KR text, those that match file stand for it alone: the entry without Files
// is the one for the files no other entry takes.
func (r *Rules) GlossaryHits(file, kr string) []Term {
	var hits []Term
	scoped := map[string]bool{} // the KR texts with an entry for file
	for _, t := range r.Glossary {
		if t.Status != StatusApproved || t.KR == "" || !strings.Contains(kr, t.KR) {
			continue
		}
		if len(t.Files) > 0 {
			if !config.MatchAny(t.Files, file) {
				continue
			}
			scoped[t.KR] = true
		}
		hits = append(hits, t)
	}
	return slices.DeleteFunc(hits, func(t Term) bool { return len(t.Files) == 0 && scoped[t.KR] })
}

// KanaRenderings returns the renderings of the glossary hits of kr in file
// that hold kana, such as し協會: their kana are meant, so the kana check
// leaves them alone (validate.Context.KanaOK).
func (r *Rules) KanaRenderings(file, kr string) []string {
	var out []string
	for _, t := range r.GlossaryHits(file, kr) {
		if strings.ContainsFunc(t.ZH, func(c rune) bool { return c >= 0x3040 && c <= 0x30FF && c != 0x30FB }) && !slices.Contains(out, t.ZH) {
			out = append(out, t.ZH)
		}
	}
	return out
}

// ValidateTerms returns the part of the glossary hits of kr in file that
// validation enforces.
func (r *Rules) ValidateTerms(file, kr string) []validate.Term {
	var terms []validate.Term
	for _, t := range r.GlossaryHits(file, kr) {
		terms = append(terms, t.ToValidate())
	}
	return terms
}
