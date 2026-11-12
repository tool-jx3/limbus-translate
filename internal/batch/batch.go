// Package batch hands out the units waiting for translation in leased
// batches and takes the translations back (spec 9.1).
//
// Next cuts one batch from the pending queue: the units of one story file, or
// of one or more other files of a family, with everything a translator needs
// to see. Submit validates what came back, records the translations that pass
// and counts the failures of the rest. Both read and write the store in a
// single lock, so any number of processes can run them at once without
// handing out a unit twice.
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
	"github.com/tool-jx3/limbus-translate/internal/existing"
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
	// Names indexes the existing names (see existing.Table): Next puts those
	// the units of a batch use into its names. Nil means none.
	Names *existing.Index
	// NameBoost holds the KR of the provisional terms that differ from an
	// existing name: a batch brings their entry of the table even where the
	// name is short and unquoted (see existing.Index.Mentions).
	NameBoost map[string]bool
	// SameText holds the renderings of the same KR sentence in other groups:
	// Next puts them into refs.same_text. Nil means none.
	SameText *existing.SameText
	// Scripts, if set, remembers rules.ScriptRenderings by file, which Judge
	// would otherwise work out for every unit; the preview, which judges many
	// units of one file, sets it. Nil means no cache. See rules.ScriptCache.
	Scripts *rules.ScriptCache
	// Now is the clock: the time of the batch id and the base of lease expiry.
	// The zero value means the current time.
	Now time.Time
}

// Batch is what Next hands to a translator; it is written to
// work/batches/<id>.json.
type Batch struct {
	ID     string `json:"id"`
	Family string `json:"family"`
	// GlossaryCheck is "warn" when the family's glossary renderings are only
	// a default (guides/routing.yaml, glossary: warn), else empty.
	GlossaryCheck string `json:"glossary_check,omitempty"`
	// File is the first file of Files.
	File string `json:"file"`
	// Files are the files of the units, in the order they were handed out;
	// never empty. A story batch has one.
	Files []string `json:"files"`
	// Guides are the guide files to read, in the order they layer: those of
	// the files and of the speakers of their units.
	Guides []string `json:"guides"`
	// Glossary lists the approved terms the units contain, each in the file
	// of its unit.
	Glossary []rules.Term `json:"glossary"`
	// Keywords maps the [ID]s of the units, without brackets, to the name
	// this translation gives them.
	Keywords map[string]string `json:"keywords"`
	// Names are existing renderings of the names the units use, at most
	// MaxNames; never nil.
	Names []NameRef   `json:"names"`
	Units []BatchUnit `json:"units"`
	// Context are translated lines around the units of a story batch.
	Context []ContextLine `json:"context"`
}

// Limits of Batch.Names, and the name of the reference with the renderings of
// the same sentence in other groups.
const (
	MaxNames        = 40
	maxNameVariants = 3
	RefSameText     = "same_text"
)

// NameRef is an existing name a batch's units use, in one of the ways the
// JP and EN packs spell it.
type NameRef struct {
	KR string `json:"kr"`
	// ZH is the main rendering of the variant and Count the units that have
	// it; Others are the other renderings.
	ZH     string               `json:"zh"`
	Count  int                  `json:"count"`
	Others []existing.Rendering `json:"others,omitempty"`
	JP     string               `json:"jp,omitempty"`
	EN     string               `json:"en,omitempty"`
	// Units are the keys of the units of the batch the name is for, in the
	// order of the batch.
	Units []string `json:"units"`
}

// nameUse is a name a unit of a batch uses: exact when the whole KR of the
// unit is the name (in a name field), else mentioned in its text.
type nameUse struct {
	name  *existing.Name
	exact bool
	key   string
}

// nameRefs makes the names of a batch from the uses, in batch order: one
// NameRef for each of the first maxNameVariants variants of a name. When more
// than MaxNames come out, those that stand for a whole unit are kept first,
// then the mentioned ones, each class longest KR first (by runes), then by KR.
// A name whose variants do not all fit ends the list. A name that is exact
// for one unit and mentioned in another counts as exact.
func nameRefs(uses []nameUse) []NameRef {
	type entry struct {
		name  *existing.Name
		exact bool
		keys  []string
	}
	var order []*entry
	byKR := map[string]*entry{}
	for _, u := range uses {
		e := byKR[u.name.KR]
		if e == nil {
			e = &entry{name: u.name}
			byKR[u.name.KR] = e
			order = append(order, e)
		}
		e.exact = e.exact || u.exact
		if !slices.Contains(e.keys, u.key) {
			e.keys = append(e.keys, u.key)
		}
	}
	slices.SortStableFunc(order, func(a, b *entry) int {
		if a.exact != b.exact {
			if a.exact {
				return -1
			}
			return 1
		}
		if x, y := utf8.RuneCountInString(a.name.KR), utf8.RuneCountInString(b.name.KR); x != y {
			return y - x
		}
		return strings.Compare(a.name.KR, b.name.KR)
	})
	refs := []NameRef{}
	for _, e := range order {
		var mine []NameRef
		for _, v := range e.name.Variants[:min(len(e.name.Variants), maxNameVariants)] {
			if len(v.Renderings) == 0 {
				continue
			}
			ref := NameRef{KR: e.name.KR, ZH: v.Renderings[0].ZH, Count: v.Renderings[0].Units, JP: v.JP, EN: v.EN, Units: e.keys}
			if len(v.Renderings) > 1 {
				ref.Others = v.Renderings[1:]
			}
			mine = append(mine, ref)
		}
		// A name is kept with all its variants or not at all, so that the
		// translator never sees a name without the variants that tell its
		// meanings apart.
		if len(refs)+len(mine) > MaxNames {
			break
		}
		refs = append(refs, mine...)
	}
	return refs
}

// BatchUnit is one string to translate.
type BatchUnit struct {
	Key string `json:"key"`
	// File is the file of the unit, that of its key.
	File  string `json:"file"`
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
	built int // how many fileData the loader has made
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
		l.built++
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

// unitText is the KR text of a unit and the file it is in.
type unitText struct{ file, kr, key string }

// glossaryFor is the union of the terms each text is held to (TermsIn), each
// in its own file (so a term for some files only comes with a text of those
// files): the approved terms once per KR term, rendering and files, in order
// of first appearance; then the provisional terms, once per KR. They are the
// terms CheckNew enforces on the units: a text that is shown with a term is
// held to it, and the other way round. A term that a longer one covers
// wherever it occurs in a text does not come with that text.
func glossaryFor(r *rules.Rules, provisional []rules.Term, texts []unitText) []rules.Term {
	terms := []rules.Term{}
	type termKey struct{ kr, zh, files, notFiles string }
	index := map[termKey]int{} // into terms
	// No glossary KR is deduped against the provisional terms: ActiveProvisional
	// already leaves out a KR the glossary has an entry for. Only the texts
	// of the batch can bring the same provisional term twice.
	var prov []rules.Term
	provIndex := map[string]int{} // KR -> into prov
	for _, x := range texts {
		for _, t := range TermsIn(r, provisional, x.file, x.kr) {
			if t.Provisional {
				i, ok := provIndex[t.KR]
				if !ok {
					i = len(prov)
					provIndex[t.KR] = i
					prov = append(prov, t)
				}
				prov[i].Units = appendUnit(prov[i].Units, x.key)
				continue
			}
			k := termKey{t.KR, t.ZH, strings.Join(t.Files, "\x00"), strings.Join(t.NotFiles, "\x00")}
			i, ok := index[k]
			if !ok {
				i = len(terms)
				index[k] = i
				terms = append(terms, t)
			}
			terms[i].Units = appendUnit(terms[i].Units, x.key)
		}
	}
	return append(terms, prov...)
}

// appendUnit adds key to units unless it is there already (a unit has
// several texts, and a key is listed once).
func appendUnit(units []string, key string) []string {
	if key == "" || slices.Contains(units, key) {
		return units
	}
	return append(units, key)
}

// TermsIn returns the terms the KR text kr of file is held to: its glossary
// hits (rules.Rules.GlossaryHits), then the provisional terms that occur in
// it, without a term each of whose hits lies inside a hit of a longer one
// (rules.DropCovered), whichever of the two is provisional. provisional are
// the provisional terms in force (ActiveProvisional); those returned are
// marked Provisional, whether they came so or not.
func TermsIn(r *rules.Rules, provisional []rules.Term, file, kr string) []rules.Term {
	return termsIn(r.GlossaryHits(file, kr), provisional, kr)
}

// termsIn is TermsIn on the glossary hits of kr, which the caller has looked
// up (rules.Rules.GlossaryHits). It does not write into hits.
func termsIn(hits []rules.Term, provisional []rules.Term, kr string) []rules.Term {
	terms := slices.Clip(hits)
	if len(provisional) == 0 {
		return terms
	}
	plain := validate.MatchText(kr) // as GlossaryHits looks for its terms
	n := len(terms)
	runes := rules.NewRuneSet(plain)
	for _, t := range provisional {
		if !runes.MayStartTerm(t.KR) {
			continue
		}
		if t.OccursIn(plain) {
			t.Provisional = true
			terms = append(terms, t)
		}
	}
	if len(terms) == n {
		return terms
	}
	return rules.DropCovered(plain, terms)
}

// ValidateTermsIn returns the glossary terms of TermsIn as validation
// enforces them on an existing translation, which the build and the review
// page check: the provisional terms in force cover glossary terms there as
// at submission, but are not enforced themselves, so a translation made
// before a term was registered warns for none.
func ValidateTermsIn(r *rules.Rules, provisional []rules.Term, file, kr string) []validate.Term {
	var out []validate.Term
	for _, t := range TermsIn(r, provisional, file, kr) {
		if !t.Provisional {
			out = append(out, t.ToValidate())
		}
	}
	return out
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
// to as well, weighed with the glossary terms (TermsIn).
func CheckNewTerms(r *rules.Rules, provisional []rules.Term, file, kr, zh string) []validate.Issue {
	return checkNewTerms(r.GlossaryHits(file, kr), provisional, file, kr, zh, r.ScriptRenderings(file))
}

// checkNewTerms is CheckNewTerms on the glossary hits of kr and the script
// renderings of file, which the caller has worked out.
func checkNewTerms(hits []rules.Term, provisional []rules.Term, file, kr, zh string, scriptOK []string) []validate.Issue {
	var terms []validate.Term
	for _, t := range termsIn(hits, provisional, kr) {
		terms = append(terms, t.ToValidate())
	}
	return validate.Check(kr, zh, validate.Context{Terms: terms, File: file, ScriptOK: scriptOK})
}

// ActiveProvisional returns the provisional terms in force, in the order of
// their first record. Of the records of one KR the first one that is not a
// conflict counts; the later records of that KR are ignored. A KR does not
// take effect when it has fewer than two characters, when the glossary has an
// entry with the same KR (in any status), or when its suggestion
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

// SameTextGroup names the group of g without its text, as the same-text
// index keys its renderings: family, field and speaker joined by "|".
func SameTextGroup(g TextGroup) string {
	return g.Family + "|" + g.Field + "|" + g.Speaker
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

// keywordIDs returns the keyword ids ("[Id]" without the brackets) that occur
// in krs.
func keywordIDs(krs []string) map[string]bool {
	want := map[string]bool{}
	for _, kr := range krs {
		for _, id := range validate.IDs(kr) {
			want[strings.TrimSuffix(strings.TrimPrefix(id, "["), "]")] = true
		}
	}
	return want
}

// keywordNames returns the names of table (see keywordTable) for the [ID]s
// that occur in krs. It only filters: the table is built by the caller, a
// batch cut by Next from the records it has just loaded, a Previewer once for
// all the units it is asked about.
func keywordNames(table map[string]string, krs []string) map[string]string {
	out := map[string]string{}
	for id := range keywordIDs(krs) {
		if name, ok := table[id]; ok {
			out[id] = name
		}
	}
	return out
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
