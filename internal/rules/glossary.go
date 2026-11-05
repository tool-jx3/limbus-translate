package rules

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

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
	// Match is MatchWord for a term that hits only where its KR stands as a
	// word (see standsAsWord); empty for one that hits wherever its KR occurs,
	// inside a longer word too.
	Match string `yaml:"match" json:"match,omitempty"`
	// Provisional marks a term registered by a translator and not yet decided
	// on by a person (see batch.ActiveProvisional). It is never read from the
	// glossary: "provisional" is not a glossary key.
	Provisional bool `yaml:"-" json:"provisional,omitempty"`
}

// MatchWord is the one value of Term.Match: the term hits only where its KR
// stands as a word, so 백화 does not hit 백화점 nor 버스 림버스.
const MatchWord = "word"

// OccursIn reports whether the KR text of t occurs in kr, as a word where t
// asks for it.
func (t Term) OccursIn(kr string) bool {
	if t.Match == MatchWord {
		return standsAsWord(kr, t.KR)
	}
	return strings.Contains(kr, t.KR)
}

// wordFollowers are what may follow a word in KR and keep it that word: the
// particles, the copula and the endings of 하다 and 되다. A Hangul syllable
// after the term that starts none of them makes it part of a longer word.
var wordFollowers = []string{
	"이", "가", "을", "를", "은", "는", "의", "에", "께", "로", "으로", "와", "과", "도", "만", "까지", "부터",
	"처럼", "보다", "한테", "랑", "나", "고", "며", "다", "입니다", "야", "요", "들", "란", "라", "마저", "조차",
	"밖에", "뿐", "대로", "만큼", "마다", "면", "엔", "님", "씨", "지", "죠", "여", "아", "였", "었", "인", "일", "임",
	"하", "했", "한", "할", "해", "함", "합", "히", "되", "된", "될", "돼", "됨", "됩",
}

func isHangulSyllable(r rune) bool { return r >= 0xAC00 && r <= 0xD7A3 }

// standsAsWord reports whether term occurs in text as a word of its own: not
// right after a Hangul syllable, and followed by no Hangul syllable, or by
// one that starts a particle or an ending (wordFollowers).
func standsAsWord(text, term string) bool {
	if term == "" {
		return false
	}
	first, _ := utf8.DecodeRuneInString(term)
	last, _ := utf8.DecodeLastRuneInString(term)
	for from := 0; from < len(text); {
		i := strings.Index(text[from:], term)
		if i < 0 {
			return false
		}
		at := from + i
		_, size := utf8.DecodeRuneInString(text[at:])
		from = at + size
		if before, _ := utf8.DecodeLastRuneInString(text[:at]); at > 0 && isHangulSyllable(before) && isHangulSyllable(first) {
			continue
		}
		rest := text[at+len(term):]
		if next, _ := utf8.DecodeRuneInString(rest); rest == "" || !isHangulSyllable(next) || !isHangulSyllable(last) {
			return true
		}
		for _, f := range wordFollowers {
			if strings.HasPrefix(rest, f) {
				return true
			}
		}
	}
	return false
}

// ToValidate returns the part of the term that validate.Check enforces.
func (t Term) ToValidate() validate.Term {
	return validate.Term{KR: t.KR, ZH: t.ZH, Forbidden: t.Forbidden}
}

// termFields are the keys an entry may have; "evidence" is statistics for
// people and is not loaded.
var termFields = map[string]bool{
	"kr": true, "jp": true, "en": true, "zh": true, "status": true,
	"forbidden": true, "category": true, "evidence": true, "notes": true, "files": true, "match": true,
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
	kr = validate.MatchText(kr) // a paired tag inside a term does not split it; a self-closing one does not fuse words
	var hits []Term
	scoped := map[string]bool{} // the KR texts with an entry for file
	for _, t := range r.Glossary {
		if t.Status != StatusApproved || t.KR == "" || !t.OccursIn(kr) {
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

// ScriptRenderings returns the approved renderings for file that hold kana or
// Hangul, such as し協會 and 하나協會: their kana and Hangul are meant, so the
// kana and Hangul checks leave them alone (validate.Context.ScriptOK). They
// count whatever the KR says, since a line may name an association by a short
// form (하나 for 하나 협회) that its translation writes in full. An entry with
// files stands for its KR text alone in the files it matches, as in
// GlossaryHits.
func (r *Rules) ScriptRenderings(file string) []string {
	scoped := map[string]bool{} // the KR texts with an entry for file
	for _, t := range r.Glossary {
		if t.Status == StatusApproved && len(t.Files) > 0 && config.MatchAny(t.Files, file) {
			scoped[t.KR] = true
		}
	}
	var out []string
	for _, t := range r.Glossary {
		if t.Status != StatusApproved || !strings.ContainsFunc(t.ZH, foreignScript) || slices.Contains(out, t.ZH) {
			continue
		}
		if len(t.Files) > 0 && !config.MatchAny(t.Files, file) || len(t.Files) == 0 && scoped[t.KR] {
			continue
		}
		out = append(out, t.ZH)
	}
	return out
}

// foreignScript reports whether c is kana (but not the middle dot ・, which
// Chinese uses) or Hangul, the scripts validate flags in a translation.
func foreignScript(c rune) bool {
	kana := c >= 0x3040 && c <= 0x30FF && c != 0x30FB
	hangul := (c >= 0xAC00 && c <= 0xD7A3) || (c >= 0x1100 && c <= 0x11FF) || (c >= 0x3130 && c <= 0x318F)
	return kana || hangul
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
