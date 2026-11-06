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
	// NotFiles are the files the term does not apply to (patterns as Files).
	NotFiles []string `yaml:"not_files" json:"not_files,omitempty"`
	// Match is MatchWord, or empty for the same, for a term that hits only
	// where its KR stands as a word (see standsAsWord), and MatchSubstring for
	// one that hits wherever its KR occurs, inside a longer word too.
	Match string `yaml:"match" json:"match,omitempty"`
	// Provisional marks a term registered by a translator and not yet decided
	// on by a person (see batch.ActiveProvisional). It is never read from the
	// glossary: "provisional" is not a glossary key.
	Provisional bool `yaml:"-" json:"provisional,omitempty"`
}

// MatchWord is the default value of Term.Match: the term hits only where its
// KR stands as a word, so 백화 does not hit 백화점 nor 버스 림버스.
const MatchWord = "word"

// MatchSubstring is the other value of Term.Match: the term hits wherever its
// KR occurs, inside a longer word too (사무소 in 정사무소).
const MatchSubstring = "substring"

// FamilyPrefix starts an item of Files or NotFiles that names a routing family
// instead of a file pattern: "family:story-main" stands for the files whose
// Family is story-main.
const FamilyPrefix = "family:"

// Applies reports whether t is in force in file: file matches one of Files
// (or t has none) and matches none of NotFiles. An item of those lists is a
// file pattern (see config.MatchAny) or FamilyPrefix and a family id, which
// matches the files of that family by the routing (Rules.Family: the first
// family that matches a file is its family). It is the one place that decides
// it; GlossaryHits and ScriptRenderings go through it. Whether an entry
// without Files gives way to one with is a matter of the KR text (scopedTo),
// not of this entry alone.
func (r *Rules) Applies(t Term, file string) bool {
	return (len(t.Files) == 0 || r.matchScope(t.Files, file)) && !r.matchScope(t.NotFiles, file)
}

// matchScope reports whether file matches one item of items, patterns and
// family ids alike.
func (r *Rules) matchScope(items []string, file string) bool {
	family := ""
	haveFamily := false
	for _, item := range items {
		id, isFamily := strings.CutPrefix(item, FamilyPrefix)
		if !isFamily {
			if config.MatchAny([]string{item}, file) {
				return true
			}
			continue
		}
		if !haveFamily {
			family, haveFamily = r.Family(file), true
		}
		if family == id {
			return true
		}
	}
	return false
}

// scopedTo reports whether t has Files and applies to file: t then stands for
// its KR text in file, in place of an entry of that KR without Files. Where
// NotFiles takes t out of file, the entry without Files is the one for it.
func (r *Rules) scopedTo(t Term, file string) bool {
	return len(t.Files) > 0 && r.Applies(t, file)
}

// OccursIn reports whether the KR text of t occurs in kr: as a word, unless t
// asks for MatchSubstring.
func (t Term) OccursIn(kr string) bool {
	if t.Match == MatchSubstring {
		return strings.Contains(kr, t.KR)
	}
	return standsAsWord(kr, t.KR)
}

// wordFollowers are what may follow a word in KR and keep it that word: the
// particles, the copula and the endings of 하다 and 되다. A Hangul syllable
// after the term that starts none of them makes it part of a longer word.
//
// The endings from 니 on were added for vowel-final nouns (복도니까요,
// 도시잖아요, 관리자답게) after measuring, on the translated units, how many
// of the hits they add use the rendering: 80% or more kept an ending. Left
// out: 예 (half; it starts 예정), 네 (no unit), 던 (only 던전), 군 and 분
// (they start compounds such as 엄지군). 더라 and 거나 are the whole endings,
// since 더 and 거 start 더미 and 거리; 됐 starts no noun.
var wordFollowers = []string{
	"이", "가", "을", "를", "은", "는", "의", "에", "께", "로", "으로", "와", "과", "도", "만", "까지", "부터",
	"처럼", "보다", "한테", "랑", "나", "고", "며", "다", "입니다", "야", "요", "들", "란", "라", "마저", "조차",
	"밖에", "뿐", "대로", "만큼", "마다", "면", "엔", "님", "씨", "지", "죠", "여", "아", "였", "었", "인", "일", "임",
	"하", "했", "한", "할", "해", "함", "합", "히", "되", "된", "될", "돼", "됨", "됩",
	"니", "시", "셨", "잖", "겠", "더라", "든", "거나", "냐", "됐", "답", "끼리", "씩",
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
	"forbidden": true, "category": true, "evidence": true, "notes": true, "files": true, "not_files": true, "match": true,
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
// config.MatchAny), a term without to every file; a term does not apply to the
// files that match one of its NotFiles (see Rules.Applies). Of several entries for one
// KR text, those that apply to file (see Rules.Applies) stand for it alone: the entry without Files
// is the one for the files no other entry takes.
func (r *Rules) GlossaryHits(file, kr string) []Term {
	kr = validate.MatchText(kr) // a paired tag inside a term does not split it; a self-closing one does not fuse words
	var hits []Term
	scoped := map[string]bool{} // the KR texts with an entry for file
	for _, t := range r.Glossary {
		if t.Status != StatusApproved || t.KR == "" || !t.OccursIn(kr) {
			continue
		}
		if r.scopedTo(t, file) {
			scoped[t.KR] = true
		}
		if !r.Applies(t, file) {
			continue
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
		if t.Status == StatusApproved && r.scopedTo(t, file) {
			scoped[t.KR] = true
		}
	}
	var out []string
	for _, t := range r.Glossary {
		if t.Status != StatusApproved || !strings.ContainsFunc(t.ZH, foreignScript) || slices.Contains(out, t.ZH) {
			continue
		}
		if !r.Applies(t, file) || len(t.Files) == 0 && scoped[t.KR] {
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
