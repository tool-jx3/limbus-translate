// Package batch hands out the units waiting for translation in leased
// batches and takes the translations back (spec 9.1).
//
// Next cuts one batch from the pending queue: the units of one file, with
// everything a translator needs to see. Submit validates what came back,
// records the translations that pass and counts the failures of the rest.
// Both read and write the store in a single lock, so any number of processes
// can run them at once without handing out a unit twice.
package batch

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/tool-jx3/limbus-translate/internal/align"
	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/names"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// Limits used when a NextOptions or maxAttempts value is zero. They are the
// values lct.yaml ships with.
const (
	DefaultMaxUnits      = 40
	DefaultStoryMaxUnits = 120
	DefaultMaxAttempts   = 3
	DefaultLeaseTTL      = 120 * time.Minute
)

// contextElements is the most dataList elements (lines of a story, groups of
// texts of an RPG dialogue) on each side of the element of a unit of a story
// batch that are looked at for context; a small batch looks at fewer (see
// contextReach).
const contextElements = 3

// Env is what Next and Submit work with.
type Env struct {
	Store *store.Store
	// Src holds the official KR, EN and JP packs of the raw snapshot.
	Src   pack.Set
	Rules *rules.Rules
	// RawTag is the tag of that snapshot; a new record's Since.
	RawTag string
	// RulesCommit names the commit of the rules the translations are made
	// under (see gitx.Describe); every record Submit writes carries it as
	// Rules.
	RulesCommit string
	// ForeignNames are names the translation must not use (see
	// bootstrap.ForeignNames); Submit warns about them, except the words of
	// Rules.ForeignIgnore.
	ForeignNames []string
	// KeywordNames are the Chinese names of the keywords (see KeywordNames).
	// Submit puts a half-width space after each of them that follows a count
	// (validate.NormalizeKeywordSpace). Empty means no name is looked for.
	KeywordNames []string
	// Provisional are the provisional terms in force (see ActiveProvisional):
	// Next puts those a batch's units contain into its glossary, and Submit
	// holds the translations to them like to glossary terms.
	Provisional []rules.Term
	// Now is the clock: the time of the batch id and the base of lease expiry.
	// The zero value means the current time.
	Now time.Time
}

// Batch is what Next hands to a translator; it is written to
// work/batches/<id>.json.
type Batch struct {
	ID     string `json:"id"`
	Family string `json:"family"`
	File   string `json:"file"`
	// Guides are the guide files to read, in the order they layer.
	Guides []string `json:"guides"`
	// Glossary lists the approved terms the units contain.
	Glossary []rules.Term `json:"glossary"`
	// Keywords maps the [ID]s of the units, without brackets, to the name
	// this translation gives them.
	Keywords map[string]string `json:"keywords"`
	Units    []BatchUnit       `json:"units"`
	// Context are translated lines around the units of a story batch.
	Context []ContextLine `json:"context"`
}

// BatchUnit is one string to translate.
type BatchUnit struct {
	Key   string `json:"key"`
	Field string `json:"field"`
	// Speaker is a character id, or the raw model or speaker string of a
	// speaker that is no known character, or "".
	Speaker string `json:"speaker"`
	KR      string `json:"kr"`
	// Reason is why the unit is pending.
	Reason string `json:"reason"`
	// PrevKR and PrevZH are the previous version, Draft a text to fix with
	// as few changes as possible, Hint what a person who sent the unit back
	// for another translation wants fixed (the rest of PrevZH is to be kept).
	PrevKR string             `json:"prev_kr,omitempty"`
	PrevZH string             `json:"prev_zh,omitempty"`
	Draft  string             `json:"draft,omitempty"`
	Hint   string             `json:"hint,omitempty"`
	EN     *align.LangAligned `json:"en,omitempty"`
	JP     *align.LangAligned `json:"jp,omitempty"`
	// Refs are other translations to look at, such as "llc" and "ltm". They
	// are references, not answers.
	Refs map[string]string `json:"refs,omitempty"`
	Keep Constraints       `json:"keep"`
}

// Constraints are what a translation must keep from KR; validation compares
// them as multisets.
type Constraints struct {
	IDs          []string `json:"ids"`
	Placeholders []string `json:"placeholders"`
	Tags         []string `json:"tags"`
	// DanteWrapper is true when KR is a line of Dante wrapped in <…>.
	DanteWrapper bool `json:"dante_wrapper"`
}

// ContextLine is a line of a story that is already translated.
type ContextLine struct {
	Key     string `json:"key"`
	Speaker string `json:"speaker"`
	KR      string `json:"kr"`
	ZH      string `json:"zh"`
}

// OutUnit is one translation a translator returns; work/batches/<id>.out.json
// is a list of them.
type OutUnit struct {
	Key  string `json:"key"`
	ZH   string `json:"zh"`
	Note string `json:"note,omitempty"`
	// Terms are new terms the unit uses: names the glossary lacks and that
	// should be rendered the same way everywhere.
	Terms []NewTerm `json:"terms,omitempty"`
}

// NewTerm is a term a translator registers: kr is the word as the KR text of
// the unit has it, zh its rendering in the translation.
type NewTerm struct {
	KR string `json:"kr"`
	ZH string `json:"zh"`
}

// SkippedTerm is a registered term that was not kept, with the reason.
type SkippedTerm struct {
	Key    string `json:"key"`
	KR     string `json:"kr"`
	ZH     string `json:"zh"`
	Reason string `json:"reason"`
}

// check reports a missing part of the Env.
func (e Env) check() error {
	switch {
	case e.Store == nil:
		return errors.New("batch: no store")
	case e.Rules == nil:
		return errors.New("batch: no rules")
	case e.Src.KR == nil:
		return errors.New("batch: no KR pack")
	}
	return nil
}

func (e Env) now() time.Time {
	if e.Now.IsZero() {
		return time.Now()
	}
	return e.Now
}

// The RPG dialogue files are read as stories: dataList[].texts[] holds the
// lines of a conversation. The files of the options (rpg-loc-dialogue-choice-*)
// match the first pattern too but hold single phrases.
const (
	rpgDialogueGlob = "RPGSystem/rpg-loc-dialogue-*"
	rpgChoiceGlob   = "RPGSystem/rpg-loc-dialogue-choice-*"
)

// IsStory reports whether file is a story script: a StoryData/ file or an RPG
// dialogue file. Its units are batched by the whole file, with context, and
// its lines are said by speakers (see SpeakerKey).
func IsStory(file string) bool {
	if strings.HasPrefix(file, align.StoryPrefix) {
		return true
	}
	// The patterns are fixed and valid, so Match cannot fail.
	if dialogue, _ := path.Match(rpgDialogueGlob, file); dialogue {
		choice, _ := path.Match(rpgChoiceGlob, file)
		return !choice
	}
	return false
}

// hasFlag reports whether flags holds flag.
func hasFlag(flags []string, flag string) bool { return slices.Contains(flags, flag) }

// withFlag returns flags with flag added once.
func withFlag(flags []string, flag string) []string {
	if hasFlag(flags, flag) {
		return flags
	}
	return append(slices.Clone(flags), flag)
}

// withoutFlag returns flags without flag, nil if none is left.
func withoutFlag(flags []string, flag string) []string {
	var out []string
	for _, f := range flags {
		if f != flag {
			out = append(out, f)
		}
	}
	return out
}

// The languages whose strings are aligned to the units (fileData.aligned).
const (
	langEN = iota
	langJP
)

// fileData is one KR file with its units in document order and, when asked
// for, the EN and JP strings aligned to them.
type fileData struct {
	env   Env
	file  string
	doc   *jsonx.Node
	units []units.Unit
	index map[string]int // pointer -> position in units

	langs  [2]map[string]align.LangAligned // by pointer
	loaded [2]bool
}

// loader opens KR files once per call.
type loader struct {
	env   Env
	files map[string]*fileData
}

func newLoader(env Env) *loader {
	return &loader{env: env, files: map[string]*fileData{}}
}

// file returns the data of a KR file, or nil if the pack has no such file.
func (l *loader) file(name string) (*fileData, error) {
	if fd, ok := l.files[name]; ok {
		return fd, nil
	}
	var fd *fileData
	if l.env.Src.KR.Has(name) {
		doc, err := l.env.Src.KR.Doc(name)
		if err != nil {
			return nil, fmt.Errorf("batch: %w", err)
		}
		fd = &fileData{env: l.env, file: name, doc: doc, units: units.Extract(name, doc)}
		fd.index = make(map[string]int, len(fd.units))
		for i, u := range fd.units {
			fd.index[u.Pointer] = i
		}
	}
	l.files[name] = fd
	return fd, nil
}

// aligned returns the string of langEN or langJP aligned to pointer, nil if
// there is none. The alignment is made once per file.
func (fd *fileData) aligned(lang int, pointer string) (*align.LangAligned, error) {
	if !fd.loaded[lang] {
		src := fd.env.Src.EN
		if lang == langJP {
			src = fd.env.Src.JP
		}
		if src != nil && src.Has(fd.file) {
			doc, err := src.Doc(fd.file)
			if err != nil {
				return nil, fmt.Errorf("batch: %w", err)
			}
			fd.langs[lang] = align.AlignLang(fd.file, fd.doc, doc)
		}
		fd.loaded[lang] = true
	}
	if a, ok := fd.langs[lang][pointer]; ok {
		return &a, nil
	}
	return nil, nil
}

// voiceFilePrefixes are the files whose lines all belong to the character
// named in the file name.
var voiceFilePrefixes = []string{"PersonalityVoiceDlg/", "EGOVoiceDig/", "BattleAnnouncerDlg/", "AbDlg_"}

// SpeakerOf names the speaker of u, a unit of doc: the character id when the
// character is known, otherwise the raw model or speaker string (possibly
// empty). Voice and AbDlg files belong to the character in the file name; the
// lines of other files to the character of their model or speaker (see
// SpeakerKey).
func SpeakerOf(r *rules.Rules, doc *jsonx.Node, u units.Unit) string {
	id, _ := speaker(r, doc, u)
	return id
}

// speaker is SpeakerOf, and whether the speaker is a known character.
func speaker(r *rules.Rules, doc *jsonx.Node, u units.Unit) (id string, known bool) {
	if slices.ContainsFunc(voiceFilePrefixes, func(p string) bool { return strings.HasPrefix(u.File, p) }) {
		if c, ok := r.CharacterByFile(u.File); ok {
			return c.ID, true
		}
		return u.Model, false
	}
	key := SpeakerKey(doc, u)
	if c, _, ok := r.CharacterByModel(key); ok {
		return c.ID, true
	}
	return key, false
}

// SpeakerKey is the string that names who says a unit in KR: the model of the
// dataList element it is in or, when that is empty, the "speaker" string of the
// object the unit itself is in (the RPG dialogue files keep it there:
// dataList[].texts[].speaker). doc is the file the unit is from. It is "" when
// there is neither.
func SpeakerKey(doc *jsonx.Node, u units.Unit) string {
	if u.Model != "" {
		return u.Model
	}
	i := strings.LastIndex(u.Pointer, "/")
	if i < 0 {
		return ""
	}
	if n, ok := doc.Get(u.Pointer[:i] + "/speaker"); ok && n.Kind == jsonx.String {
		return n.Str
	}
	return ""
}

// constraints reads what a translation of kr must keep.
func constraints(kr string) Constraints {
	return Constraints{
		IDs:          nonNil(validate.IDs(kr)),
		Placeholders: nonNil(validate.Placeholders(kr)),
		Tags:         nonNil(validate.Tags(kr)),
		DanteWrapper: validate.IsDanteLine(kr),
	}
}

// nonNil turns a nil list into an empty one, so it is written as [] and not
// null.
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// glossaryFor is the union of the approved terms the texts contain in file,
// once per KR term and rendering, in order of first appearance, then the
// provisional terms they contain, once per KR. They are the
// terms CheckNew enforces on the units of file: a text that is shown with a
// term is held to it, and the other way round.
func glossaryFor(r *rules.Rules, provisional []rules.Term, file string, krs []string) []rules.Term {
	terms := []rules.Term{}
	type termKey struct{ kr, zh string }
	seen := map[termKey]bool{}
	for _, kr := range krs {
		for _, t := range r.GlossaryHits(file, kr) {
			if k := (termKey{t.KR, t.ZH}); !seen[k] {
				seen[k] = true
				terms = append(terms, t)
			}
		}
	}
	seenKR := map[string]bool{}
	for _, t := range terms {
		seenKR[t.KR] = true
	}
	for _, kr := range krs {
		for _, t := range provisional {
			if t.OccursIn(kr) && !seenKR[t.KR] {
				seenKR[t.KR] = true
				terms = append(terms, t)
			}
		}
	}
	return terms
}

// CheckNew checks zh as a new translation of the KR text kr of file, which is
// what Submit holds a translation to: the structure of kr, and the wording
// rules with the glossary terms kr contains in file (see
// rules.Rules.GlossaryHits), content issues being hard. zh is checked as it
// is; Submit normalizes a translation first.
func CheckNew(r *rules.Rules, file, kr, zh string) []validate.Issue {
	return CheckNewTerms(r, nil, file, kr, zh)
}

// CheckNewTerms is CheckNew with the provisional terms that occur in kr held
// to as well.
func CheckNewTerms(r *rules.Rules, provisional []rules.Term, file, kr, zh string) []validate.Issue {
	terms := r.ValidateTerms(file, kr)
	for _, t := range provisional {
		if t.OccursIn(kr) {
			terms = append(terms, t.ToValidate())
		}
	}
	return validate.Check(kr, zh, validate.Context{Terms: terms, File: file, ScriptOK: r.ScriptRenderings(file)})
}

// ActiveProvisional returns the provisional terms in force, in the order of
// their first record. Of the records of one KR the first one that is no
// conflict counts. Left out are a KR of fewer than two characters, a KR the
// glossary has an entry for in any status, and a KR whose suggestion
// "term-new/<kr>" is in declined. The terms hit only where their KR stands as
// a word.
func ActiveProvisional(all []store.ProvisionalTerm, r *rules.Rules, declined map[string]bool) []rules.Term {
	inGlossary := map[string]bool{}
	for _, t := range r.Glossary {
		inGlossary[t.KR] = true
	}
	var out []rules.Term
	seen := map[string]bool{}
	for _, p := range all {
		if p.Conflict || seen[p.KR] || utf8.RuneCountInString(p.KR) < 2 || inGlossary[p.KR] || declined["term-new/"+p.KR] {
			continue
		}
		seen[p.KR] = true
		out = append(out, rules.Term{KR: p.KR, ZH: p.ZH, Status: rules.StatusApproved, Match: rules.MatchWord, Provisional: true})
	}
	return out
}

// ShortTextRunes is the length, in runes of the KR text, under which a text
// is too short to be taken for the same text in another file: "확인" or a
// lone name means different things in different places. See
// TextGroup.FileBound.
const ShortTextRunes = 10

// TextGroup is what makes two units "the same text" (spec 8.1): the family
// of their files (rules.Rules.Family), the field name, the speaker (SpeakerOf;
// "" for a unit nobody says) and the exact KR text. Units of one group can
// share one translation. The plan copies the translation of a unit to a new
// unit of its group instead of queueing it (same-text reuse), and the
// translation of a pending unit serves the other pending units of its group
// (twins). Either way, a text that is FileBound is shared only within its file.
//
// TextGroup is comparable, so it can key a map.
type TextGroup struct {
	Family  string
	Field   string
	Speaker string
	KR      string
}

// TextGroupOf returns the group of u, a unit of doc.
func TextGroupOf(r *rules.Rules, doc *jsonx.Node, u units.Unit) TextGroup {
	return TextGroup{Family: r.Family(u.File), Field: u.Field, Speaker: SpeakerOf(r, doc, u), KR: u.KR}
}

// FileBound reports whether the text of g is shorter than ShortTextRunes
// runes, so that only units of one file share a translation.
func (g TextGroup) FileBound() bool {
	return utf8.RuneCountInString(g.KR) < ShortTextRunes
}

// Overlay returns the document of file in kr with the translation written
// over every string that has one: the string of the record of its unit, when
// the record was made from the KR text there is now (its kr_sha) and is not
// flagged needs_review. Strings without such a record keep their KR text.
//
// The document is a copy; the pack's own tree, which is shared and read-only,
// is not changed. It is how the names of this translation are read from KR
// documents: the keyword names (keywordTable) and the speaker names.
func Overlay(kr *pack.Pack, records map[string]store.Record, file string) (*jsonx.Node, error) {
	doc, err := kr.Doc(file)
	if err != nil {
		return nil, fmt.Errorf("batch: %w", err)
	}
	doc = doc.Clone()
	for _, f := range doc.Strings() {
		rec, ok := records[units.Key(file, f.Pointer)]
		if ok && rec.KRSha == units.KRSha(f.Value) && !hasFlag(rec.Flags, store.FlagNeedsReview) {
			if err := doc.SetString(f.Pointer, rec.ZH); err != nil {
				return nil, fmt.Errorf("batch: %w", err)
			}
		}
	}
	return doc, nil
}

// keywordTable is the name this translation gives each keyword id: the table
// read from the KR keyword files with every translated name written over its
// KR text (see Overlay). A name that is still Korean is no name of ours and
// is left out.
func keywordTable(kr *pack.Pack, records map[string]store.Record) (map[string]string, error) {
	docs := map[string]*jsonx.Node{}
	for _, file := range kr.Files() {
		if !names.IsKeywordFile(file) {
			continue
		}
		doc, err := Overlay(kr, records, file)
		if err != nil {
			return nil, err
		}
		docs[file] = doc
	}
	table := map[string]string{}
	for id, name := range names.KeywordTable(docs) {
		if !names.HasHangul(name) {
			table[id] = name
		}
	}
	return table, nil
}

// keywordNames builds the names this translation gives the [ID]s that occur
// in krs (see keywordTable).
func keywordNames(env Env, records map[string]store.Record, krs []string) (map[string]string, error) {
	want := map[string]bool{}
	for _, kr := range krs {
		for _, id := range validate.IDs(kr) {
			want[strings.TrimSuffix(strings.TrimPrefix(id, "["), "]")] = true
		}
	}
	if len(want) == 0 {
		return map[string]string{}, nil
	}
	all, err := keywordTable(env.Src.KR, records)
	if err != nil {
		return nil, err
	}
	table := map[string]string{}
	for id, name := range all {
		if want[id] {
			table[id] = name
		}
	}
	return table, nil
}

// KeywordNames lists, sorted and once each, the Chinese names this
// translation gives the keywords, read from the records of the Bufs* and
// BattleKeywords* files of kr (see keywordTable). It is what Env.KeywordNames
// holds.
func KeywordNames(kr *pack.Pack, records map[string]store.Record) ([]string, error) {
	table, err := keywordTable(kr, records)
	if err != nil {
		return nil, err
	}
	list := make([]string, 0, len(table))
	for _, name := range table {
		list = append(list, name)
	}
	slices.Sort(list)
	return slices.Compact(list), nil
}
