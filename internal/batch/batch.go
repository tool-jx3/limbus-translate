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
	"unicode"

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

// contextLines is how many units on each side of a unit of a story batch are
// looked at for context, counted in the file's document order.
const contextLines = 3

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
	// bootstrap.ForeignNames).
	ForeignNames []string
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

// isStory reports whether file is a story script: a StoryData/ file or an RPG
// dialogue file. Its units are batched by the whole file, with context.
func isStory(file string) bool {
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

// speaker names the speaker of a unit: the character id when the character is
// known, otherwise the raw key (possibly empty). Voice and AbDlg files belong
// to the character in the file name; the lines of other files to the character
// of their model or speaker (see speakerKey).
func speaker(r *rules.Rules, doc *jsonx.Node, u units.Unit) (id string, known bool) {
	if slices.ContainsFunc(voiceFilePrefixes, func(p string) bool { return strings.HasPrefix(u.File, p) }) {
		if c, ok := r.CharacterByFile(u.File); ok {
			return c.ID, true
		}
		return u.Model, false
	}
	key := speakerKey(doc, u)
	if c, _, ok := r.CharacterByModel(key); ok {
		return c.ID, true
	}
	return key, false
}

// speakerKey is the string that names who says a unit in KR: the model of the
// dataList element it is in or, when that is empty, the "speaker" string of the
// object the unit itself is in (the RPG dialogue files keep it there:
// dataList[].texts[].speaker). doc is the file the unit is from. It is "" when
// there is neither.
func speakerKey(doc *jsonx.Node, u units.Unit) string {
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

// glossaryFor is the union of the approved terms the texts contain, once per
// KR term, in order of first appearance.
func glossaryFor(r *rules.Rules, krs []string) []rules.Term {
	terms := []rules.Term{}
	seen := map[string]bool{}
	for _, kr := range krs {
		for _, t := range r.GlossaryHits(kr) {
			if !seen[t.KR] {
				seen[t.KR] = true
				terms = append(terms, t)
			}
		}
	}
	return terms
}

// validateTerms is the part of the glossary hits of kr that validation
// enforces.
func validateTerms(r *rules.Rules, kr string) []validate.Term {
	var terms []validate.Term
	for _, t := range r.GlossaryHits(kr) {
		terms = append(terms, t.ToValidate())
	}
	return terms
}

// keywordNames builds the names this translation gives the [ID]s that occur
// in krs. The table is read from the KR keyword files with every translated
// name written over its KR text; a name that is still Korean is no name of
// ours and is left out. Only a record made from the KR text there is now
// counts as a translation.
func keywordNames(env Env, records map[string]store.Record, krs []string) (map[string]string, error) {
	want := map[string]bool{}
	for _, kr := range krs {
		for _, id := range validate.IDs(kr) {
			want[strings.TrimSuffix(strings.TrimPrefix(id, "["), "]")] = true
		}
	}
	table := map[string]string{}
	if len(want) == 0 {
		return table, nil
	}

	docs := map[string]*jsonx.Node{}
	for _, file := range env.Src.KR.Files() {
		if !names.IsKeywordFile(file) {
			continue
		}
		doc, err := env.Src.KR.Doc(file)
		if err != nil {
			return nil, fmt.Errorf("batch: %w", err)
		}
		doc = doc.Clone() // the pack's tree is shared and read-only
		for _, f := range doc.Strings() {
			rec, ok := records[units.Key(file, f.Pointer)]
			if ok && rec.KRSha == units.KRSha(f.Value) && !hasFlag(rec.Flags, store.FlagNeedsReview) {
				if err := doc.SetString(f.Pointer, rec.ZH); err != nil {
					return nil, fmt.Errorf("batch: %w", err)
				}
			}
		}
		docs[file] = doc
	}
	for id, name := range names.KeywordTable(docs) {
		if want[id] && !strings.ContainsFunc(name, func(r rune) bool { return unicode.Is(unicode.Hangul, r) }) {
			table[id] = name
		}
	}
	return table, nil
}
