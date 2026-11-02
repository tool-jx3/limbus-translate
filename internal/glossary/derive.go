// Package glossary derives glossary candidates from the language pack itself
// (spec 15): the keyword names and speaker names that this translation
// already uses and the glossary does not list.
package glossary

import (
	"cmp"
	"errors"
	"slices"
	"strings"
	"unicode"

	"github.com/tool-jx3/limbus-translate/internal/batch"
	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/names"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/suggest"
)

// The sources of a Candidate.
const (
	// SourceKeyword is a name from the BattleKeywords* and Bufs* files.
	SourceKeyword = "keyword"
	// SourceSpeaker is a name from ScenarioModelCodes-AutoCreated.
	SourceSpeaker = "speaker"
)

// Sources lists the sources, in the order Derive reads them.
var Sources = []string{SourceKeyword, SourceSpeaker}

// Candidate is a name the translation uses that the glossary does not list: a
// proposal for an entry, for a person to decide on.
type Candidate struct {
	// KR is the name in the KR pack.
	KR string `json:"kr"`
	// ZH is the name this translation gives it.
	ZH string `json:"zh"`
	// Source is SourceKeyword or SourceSpeaker.
	Source string `json:"source"`
}

// Inputs is what Derive looks at.
type Inputs struct {
	// KR is the KR pack of the raw snapshot.
	KR *pack.Pack
	// Records are the translations (store.Store.LoadRecords).
	Records map[string]store.Record
	// Terms are the glossary entries (rules.Rules.Glossary).
	Terms []rules.Term
	// Declined are the suggestions that people turned down
	// (suggest.LoadDeclined).
	Declined []suggest.Declined
}

// Derive lists the keyword names and the speaker names of the translation,
// sorted by KR (then source and ZH). A keyword is matched to its translation
// by id and a speaker by model key, in the KR documents and the same
// documents with the translations written over them (batch.Overlay).
//
// It leaves out a KR name that any glossary entry has as its kr, whatever the
// entry's status or files, a KR name whose term suggestion ("term/<kr>") is
// declined, and a name the translation has not rendered yet, one whose ZH is
// still Korean. The names are compared without the blanks around them, which
// the pack has on a few; a candidate has the KR as the pack has it. A name of several ids or files that is rendered the same way
// is listed once; one that is rendered in several ways is listed once for
// each.
func Derive(in Inputs) ([]Candidate, error) {
	if in.KR == nil {
		return nil, errors.New("glossary: no KR pack to derive from")
	}
	skip := map[string]bool{}
	for _, t := range in.Terms {
		skip[strings.TrimSpace(t.KR)] = true
	}
	for _, d := range in.Declined {
		if kr, ok := strings.CutPrefix(d.ID, suggest.KindTerm+"/"); ok {
			skip[strings.TrimSpace(kr)] = true
		}
	}

	found := []Candidate{}
	add := func(source string, krNames, zhNames map[string]string) {
		for id, kr := range krNames {
			zh, ok := zhNames[id]
			if ok && !skip[strings.TrimSpace(kr)] && !hasHangul(zh) {
				found = append(found, Candidate{KR: kr, ZH: zh, Source: source})
			}
		}
	}

	krDocs, zhDocs := map[string]*jsonx.Node{}, map[string]*jsonx.Node{}
	for _, file := range in.KR.Files() {
		if !names.IsKeywordFile(file) {
			continue
		}
		if err := in.read(file, krDocs, zhDocs); err != nil {
			return nil, err
		}
	}
	add(SourceKeyword, names.KeywordTable(krDocs), names.KeywordTable(zhDocs))

	if in.KR.Has(names.SpeakerFile) {
		krDocs, zhDocs = map[string]*jsonx.Node{}, map[string]*jsonx.Node{}
		if err := in.read(names.SpeakerFile, krDocs, zhDocs); err != nil {
			return nil, err
		}
		add(SourceSpeaker, names.SpeakerTable(krDocs[names.SpeakerFile]), names.SpeakerTable(zhDocs[names.SpeakerFile]))
	}

	slices.SortFunc(found, func(a, b Candidate) int {
		return cmp.Or(strings.Compare(a.KR, b.KR), strings.Compare(a.Source, b.Source), strings.Compare(a.ZH, b.ZH))
	})
	return slices.Compact(found), nil
}

// read puts the KR document of file into kr and the document with the
// translations written over it into zh.
func (in Inputs) read(file string, kr, zh map[string]*jsonx.Node) error {
	doc, err := in.KR.Doc(file)
	if err != nil {
		return err
	}
	over, err := batch.Overlay(in.KR, in.Records, file)
	if err != nil {
		return err
	}
	kr[file], zh[file] = doc, over
	return nil
}

func hasHangul(s string) bool {
	return strings.ContainsFunc(s, func(r rune) bool { return unicode.Is(unicode.Hangul, r) })
}
