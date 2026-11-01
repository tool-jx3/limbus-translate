package rules

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

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
}

// ToValidate returns the part of the term that validate.Check enforces.
func (t Term) ToValidate() validate.Term {
	return validate.Term{KR: t.KR, Forbidden: t.Forbidden}
}

// termFields are the keys an entry may have; "evidence" is statistics for
// people and is not loaded.
var termFields = map[string]bool{
	"kr": true, "jp": true, "en": true, "zh": true, "status": true,
	"forbidden": true, "category": true, "evidence": true, "notes": true,
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

// StatusApproved is the Status of a term that binds. A "candidate" term is
// only a proposal and binds nothing until a person approves it.
const StatusApproved = "approved"

// GlossaryHits returns the approved terms whose KR text occurs in kr, in
// glossary order. Candidate terms stay in Rules.Glossary but are never
// returned, so they are neither injected into a batch nor enforced by
// validation.
func (r *Rules) GlossaryHits(kr string) []Term {
	var hits []Term
	for _, t := range r.Glossary {
		if t.Status == StatusApproved && t.KR != "" && strings.Contains(kr, t.KR) {
			hits = append(hits, t)
		}
	}
	return hits
}
