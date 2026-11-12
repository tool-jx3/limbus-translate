// Package build turns the translation records of the tw store into the zh-TW
// language pack. For every KR file it clones the KR tree and fills in each
// unit, writes the files into the tw worktree, packs a text-only zip and a
// complete zip (with the font) for the Release, and writes a report.
//
// The output is reproducible: the same inputs give the same bytes, in the
// files and in the zips. Nothing in it depends on the clock, the file system
// order or the machine; the build time is an input (Options.BuiltAt).
package build

import (
	"cmp"
	"errors"
	"fmt"
	"github.com/tool-jx3/limbus-translate/internal/itemid"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/align"
	"github.com/tool-jx3/limbus-translate/internal/batch"
	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/existing"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// ReasonStructuralFail is the reason of a pending item Run adds for a record
// whose translation breaks the structure of its KR text.
const ReasonStructuralFail = "structural-fail"

// ShownStandIn is the key of Report.Shown that counts the units showing a
// stand-in text; the other keys are the origins of the translations shown.
const ShownStandIn = "stand-in"

// minBuiltYear is the first year a zip file can store.
const minBuiltYear = 1980

// ErrFontChecksum is returned by Run when the font file does not have the
// sha256 the options pin. Nothing has been written then.
var ErrFontChecksum = errors.New("font checksum mismatch")

// Options is what a build is made from.
type Options struct {
	// TWRoot is the tw worktree: it holds the store (.meta/) and receives
	// LimbusCompany_Data/.
	TWRoot string
	// OutDir receives the two zips. It is created when missing.
	OutDir string
	// LangDir is the name of the language directory, "zh-TW".
	LangDir string
	// FontPath is the font file and FontSHA256 the lowercase or uppercase hex
	// sha256 it must have. OFL.txt must lie next to the font.
	FontPath, FontSHA256 string
	// LicensePath is the file copied to Info/LICENSE-CONTENT.
	LicensePath string
	// RawTag and RawToken name the KR/EN/JP snapshot built from, RulesCommit
	// the commit of the rules. Upstream names the pinned upstream releases as
	// "<repo>@<tag>" by source; it may be empty. All four go into
	// Info/version.json.
	RawTag, RawToken, RulesCommit string
	Upstream                      map[string]string
	// BuiltAt is the build time: Info/version.json says it and every zip entry
	// carries it. It is an input so that equal inputs give equal bytes. Zips
	// cannot store a time before 1980.
	BuiltAt time.Time
	// Frozen lists the files (see config.MatchAny) whose records are used
	// although KR changed since they were made.
	Frozen []string
	// Rules gives the family of a file (guides/routing.yaml), which groups the
	// units whose different translations of one text the report lists. With
	// nil every file is of one family.
	Rules *rules.Rules
	// RepoURL is the URL of the repo on GitHub, such as
	// https://github.com/tool-jx3/limbus-translate; the report links its
	// compare pages. With "" it has no links.
	RepoURL string
	// PrevTWCommit is the HEAD of tw before this build, the start of the
	// compare page of the translations; "" when tw had no commit. Only
	// JobSummary uses it: nothing Run commits to tw depends on it, because it
	// changes with every publish.
	PrevTWCommit string
	// Declined and Accepted are the sets of ids (internal/itemid) of the
	// suggestions a person confirmed: a declined group of inconsistent
	// renderings is left out of the report, an accepted warning is not
	// counted, and a declined term-new/<kr> is no provisional term in force
	// (batch.ActiveProvisional). nil is empty.
	Declined, Accepted map[string]bool
	// Names is the table of existing names (existing.NewIndex). With it and
	// with Rules, a record of origin claude that names an existing name
	// otherwise warns name-differs, and the report lists the translations of
	// this raw tag that differ from the existing names (Report.NameDiffs) and
	// the provisional terms of this raw tag that differ from the old
	// translations (Report.TermDiffs). With nil nothing is compared with them.
	Names *existing.Index
}

const (
	// maxWarningSamples is how many keys of each warning the report keeps.
	maxWarningSamples = 10
	// maxInconsistent is how many groups of units that translate one text
	// differently the report keeps, the biggest first.
	maxInconsistent = 20
	// maxInconsistentAll is how many of them the build record keeps
	// (Info.InconsistentAll), in the same order, besides those with units of
	// this update, which it keeps all.
	maxInconsistentAll = 500
	// maxRenderings is how many different translations of one text a group
	// of Inconsistent lists.
	maxRenderings = 10
	// maxMisaligned is how many files the report lists as misaligned, the
	// worst first.
	maxMisaligned = 20
	// maxNameRenderings is how many existing renderings a NameDiff lists.
	maxNameRenderings = 5
	// maxTermKeys is how many units of this update a TermDiff lists.
	maxTermKeys = 20
)

// Report counts what a build did.
type Report struct {
	// Origins counts the records used, by origin. A copy record of a unit the
	// field rules do not translate counts as used: KR is what it holds.
	Origins map[string]int `json:"origins"`
	// Pending counts the translatable units that have no usable record; their
	// EN text, or KR when there is none, stands in.
	Pending int `json:"pending"`
	// NeedsReview counts the used records that are flagged needs_review,
	// except those listed in FrozenFallback: a problem is counted once.
	NeedsReview int `json:"needs_review"`
	// StructuralFallback counts the used records that broke the structure of
	// KR (tags, [ID]s, placeholders) and were replaced by EN, or KR, and
	// queued for translation again. Records of frozen files are not counted
	// here but in FrozenFallback.
	StructuralFallback int `json:"structural_fallback"`
	// FrozenFallback lists, sorted, the keys of the used records of frozen
	// files that broke the structure of KR. They were replaced by EN, or KR,
	// like structural fallbacks, but not queued: a frozen file is not
	// translated again, so a person must fix them.
	FrozenFallback []string `json:"frozen_fallback"`
	// Unaligned counts the pending units that have no EN text either.
	Unaligned int `json:"unaligned"`
	// FieldRuleOverride counts the records, other than copies, that the
	// field rules overrode: the unit is not translated, so KR is shown.
	FieldRuleOverride int `json:"field_rule_override"`
	// Warnings counts the warnings of the used records, by issue code. It
	// covers the same records as Origins and NeedsReview, those that fall back
	// structurally included, except copies, which are not checked.
	Warnings map[string]int `json:"warnings"`
	// MissingGlyphs lists, sorted, the characters of the output that the font
	// cannot show. It does not stop the build.
	MissingGlyphs []rune `json:"missing_glyphs"`
	// StandIn counts the units that show a stand-in text, EN or else KR: the
	// pending units, the used records of origin fallback, and the used
	// records that were replaced because they break the structure of KR
	// (structural and frozen fallbacks). Each unit counts once.
	StandIn int `json:"stand_in"`
	// WarningSamples holds, for each code of Warnings, the first 10 of the
	// keys of the units that warn, in sorted order.
	WarningSamples map[string][]string `json:"warning_samples"`
	// Inconsistent lists the groups of units of one family and field with
	// the same KR text and different translations, the biggest 20 by their
	// number of units. Only used translations count: not copies, not stand-ins.
	Inconsistent []Inconsistency `json:"inconsistent"`
	// Notes are the translator's notes added since the previous build (see
	// Info.NotesSeen).
	Notes []store.Note `json:"notes"`
	// NewTranslated counts the records of origin claude written for this raw
	// tag (their Since is the raw tag).
	NewTranslated int `json:"new_translated"`
	// Shown counts the units by what they finally show, each unit once: the
	// origin of the translation (paratranz, claude, human), copy for the text
	// of KR kept as it is (a unit that is not translated), and ShownStandIn
	// for the EN, or else KR, text that stands in for a translation. Unlike
	// Origins, which counts the records used, a record that broke the
	// structure of KR and was replaced counts as a stand-in only. The values
	// add up to the number of units.
	Shown map[string]int `json:"shown"`
	// Misaligned lists the files with units of KR that EN or JP has no
	// counterpart for, the worst 20 by the sum of both counts, then by name.
	// A language that does not have the file at all is not counted for it.
	Misaligned []FileMisaligned `json:"misaligned"`
	// MisalignedFiles counts the misaligned files, those past the 20 of
	// Misaligned included. A build.json of an older lct has 0.
	MisalignedFiles int `json:"misaligned_files"`
	// NameDiffs lists, by KR, the existing names that translations written
	// for this raw tag render otherwise: in a name field (a name-differs
	// warning that is not accepted) or where a text mentions the name. A
	// declined name is not listed. TermDiffs lists, by KR, the provisional
	// terms registered at this raw tag that differ from the existing names or
	// the old translations and are not declined. Both weigh the records that
	// break the structure of KR too: such a record is queued again with its
	// text as the draft, so how it renders a name or a term still matters.
	// Both are never nil, though a build.json of an older lct has neither
	// (see Options.Names).
	NameDiffs []NameDiff `json:"name_diffs"`
	TermDiffs []TermDiff `json:"term_diffs"`
}

// NameDiff is an existing name that translations of this raw tag render
// otherwise. Existing holds the renderings of all its variants, the units of
// one rendering summed, the most units first and then by ZH, at most five;
// Units are the units, by key.
type NameDiff struct {
	KR       string               `json:"kr"`
	Existing []existing.Rendering `json:"existing"`
	Units    []NameDiffUnit       `json:"units"`
}

// NameDiffUnit is a unit of a NameDiff: its translation, the notes of the
// translator on it (joined by newlines) and how it renders the name
// otherwise, NameDiffField or NameDiffMention.
type NameDiffUnit struct {
	Key  string `json:"key"`
	ZH   string `json:"zh"`
	Note string `json:"note,omitempty"`
	Kind string `json:"kind"`
}

// The kinds of a NameDiffUnit: a name field that warns name-differs, and a
// text that mentions the name and has none of its renderings.
const (
	NameDiffField   = existing.CodeNameDiffers
	NameDiffMention = "mention"
)

// TermDiff is a provisional term, KR -> ZH, registered at this raw tag that
// differs from the existing name or the old translations, so it is not in
// force. Existing is the main rendering of KR in the table of existing names,
// or else the one the term was stored with (store.OldUse); OldUnits counts
// the old units (existing.IsOld) whose KR holds the term and OldWith those of
// them whose translation holds ZH, as this build finds them; Examples are up
// to three of those units (existing.OldTally.Use). Keys are the units of this
// update whose KR holds the term and whose translation holds ZH, by key, at
// most 20.
type TermDiff struct {
	KR       string                `json:"kr"`
	ZH       string                `json:"zh"`
	Existing string                `json:"existing,omitempty"`
	OldUnits int                   `json:"old_units"`
	OldWith  int                   `json:"old_with"`
	Examples []existing.OldExample `json:"examples"`
	Keys     []string              `json:"keys"`
}

// FileMisaligned says how many units of the KR file have no EN, and no JP,
// string aligned to them (see package align).
type FileMisaligned struct {
	File string `json:"file"`
	EN   int    `json:"en"`
	JP   int    `json:"jp"`
}

// Inconsistency is a group of units with the same KR text, in one family and
// field, that do not all have the same translation. ZH holds the different
// translations in the order the units are met (at most 10), Keys[i] is the
// first unit that has ZH[i] and Counts[i] how many units have it. Units counts
// the units of the group, those of a translation past the tenth included. New
// counts the units of the group translated for this raw tag
// (store.Record.TranslatedIn) and NewKeys are their keys, sorted; both are
// left out when there are none. A build.json of an older lct has no Counts
// and no New.
type Inconsistency struct {
	Family  string   `json:"family"`
	Field   string   `json:"field"`
	KR      string   `json:"kr"`
	ZH      []string `json:"zh"`
	Keys    []string `json:"keys"`
	Counts  []int    `json:"counts,omitempty"`
	Units   int      `json:"units"`
	New     int      `json:"new,omitempty"`
	NewKeys []string `json:"new_keys,omitempty"`
}

// builder is the state of one Run.
type builder struct {
	opt     Options
	src     pack.Set
	fr      *fields.Rules
	records map[string]store.Record
	queued  map[string]bool // keys of the pending queue, old and new
	added   []store.PendingItem
	rep     *Report
	glyphs  runeSet
	scripts *rules.ScriptCache
	// warnKeys are the keys of the units that warn, by code, in the order
	// they were met; renders groups the translations by the text they render.
	warnKeys   map[string][]string
	renders    map[renderKey]*rendering
	families   map[string]string // the family of each file, once asked
	misaligned []FileMisaligned  // every file that has units without EN or JP
	// inconsistentAll are the groups of Report.Inconsistent and those after
	// them, up to maxInconsistentAll, and every later one with units of this
	// update; see Info.InconsistentAll.
	inconsistentAll []Inconsistency
	// provisional are the provisional terms in force, which cover glossary
	// terms in the checks as they do at submission (batch.ValidateTermsIn).
	provisional []rules.Term
	// notes are the notes of the translator on each unit, by key, in the
	// order they were added (see noteOn).
	notes map[string][]store.Note
	// nameDiffs are the entries of Report.NameDiffs, by KR; termDiffs the
	// provisional terms of Report.TermDiffs, by KR, and tally counts the old
	// units that hold them. All three stay empty without Options.Names.
	nameDiffs map[string]*NameDiff
	termDiffs []*termDiff
	tally     *existing.OldTally
}

// termDiff is a provisional term of Report.TermDiffs while the units are
// met: its record, the term to look for (its KR as a word) and the units of
// this update that use its rendering.
type termDiff struct {
	p    store.ProvisionalTerm
	term rules.Term
	keys []string
}

// Run builds the language pack for src (the raw KR, EN and JP packs) from the
// records of the store at opt.TWRoot. It writes the pack into the tw worktree,
// adds a pending item for every record that fails the structure check, writes
// the zips to opt.OutDir and the build record and report to the store. The
// whole run holds the store lock.
//
// Every input is checked before anything is written; a font that does not match
// opt.FontSHA256 is ErrFontChecksum.
//
// The store may hold .meta/translate.json, the record of the last translation
// run (see package translate). When that run was made for opt.RawTag and
// opt.RulesCommit, its model and cost go into Info/version.json and the build
// record, and how it ended into the build record and the report. It may hold
// .meta/plan.json too (see package plan); when that is the plan of
// opt.RawTag, the report tells what it did. The report lists the translator's
// notes that came after the last build (Info.NotesSeen).
func Run(opt Options, src pack.Set, fr *fields.Rules) (*Report, error) {
	if err := checkOptions(opt); err != nil {
		return nil, err
	}
	if src.KR == nil || fr == nil {
		return nil, errors.New("build: the KR pack and the field rules are required")
	}
	if len(src.KR.Files()) == 0 {
		return nil, errors.New("build: the KR pack has no files; building would delete the whole language pack")
	}
	in, err := loadInputs(opt)
	if err != nil {
		return nil, err
	}

	st := store.Open(opt.TWRoot)
	var rep *Report
	err = st.WithLock(func() error {
		var err error
		rep, err = buildLocked(st, opt, in, src, fr)
		return err
	})
	if err != nil {
		return nil, err
	}
	return rep, nil
}

func checkOptions(opt Options) error {
	for _, o := range []struct{ name, value string }{
		{"TWRoot", opt.TWRoot}, {"OutDir", opt.OutDir}, {"FontPath", opt.FontPath}, {"LicensePath", opt.LicensePath},
	} {
		if o.value == "" {
			return fmt.Errorf("build: option %s is empty", o.name)
		}
	}
	// Both end up as one component of a path.
	for _, o := range []struct{ name, value string }{{"LangDir", opt.LangDir}, {"RawTag", opt.RawTag}} {
		if o.value == "" || o.value == "." || strings.ContainsAny(o.value, `/\`) || !filepath.IsLocal(o.value) {
			return fmt.Errorf("build: option %s = %q is not a plain name", o.name, o.value)
		}
	}
	if opt.BuiltAt.IsZero() || opt.BuiltAt.Year() < minBuiltYear {
		return fmt.Errorf("build: option BuiltAt = %v is not set or is before %d, which zip files cannot store", opt.BuiltAt, minBuiltYear)
	}
	return nil
}

// buildLocked is Run's work, done under the store lock.
func buildLocked(st *store.Store, opt Options, in *inputs, src pack.Set, fr *fields.Rules) (*Report, error) {
	records, err := st.LoadRecords()
	if err != nil {
		return nil, err
	}
	pending, err := st.LoadPending()
	if err != nil {
		return nil, err
	}
	// Read before anything is written, like the other inputs.
	tr, err := readTranslation(st, opt)
	if err != nil {
		return nil, err
	}
	sum, err := readPlan(st, opt.RawTag)
	if err != nil {
		return nil, err
	}
	notes, seen, byKey, err := newNotes(st)
	if err != nil {
		return nil, err
	}
	b := &builder{
		opt: opt, src: src, fr: fr,
		scripts: rules.NewScriptCache(opt.Rules),
		records: records,
		queued:  make(map[string]bool, len(pending)),
		rep: &Report{
			Origins: map[string]int{}, Warnings: map[string]int{}, FrozenFallback: []string{}, MissingGlyphs: []rune{},
			WarningSamples: map[string][]string{}, Inconsistent: []Inconsistency{}, Notes: notes,
			Shown: map[string]int{}, Misaligned: []FileMisaligned{}, NameDiffs: []NameDiff{}, TermDiffs: []TermDiff{},
		},
		warnKeys:  map[string][]string{},
		renders:   map[renderKey]*rendering{},
		families:  map[string]string{},
		notes:     byKey,
		nameDiffs: map[string]*NameDiff{},
	}
	for _, it := range pending {
		b.queued[it.Key] = true
	}
	if opt.Rules != nil {
		all, err := st.LoadTerms()
		if err != nil {
			return nil, err
		}
		b.provisional = batch.ActiveProvisional(all, opt.Rules, opt.Declined)
		if b.comparing() {
			b.termDiffs = differingTerms(all, opt.Rules, opt.RawTag, opt.Declined)
			terms := make([]rules.Term, len(b.termDiffs))
			for i, d := range b.termDiffs {
				terms[i] = d.term
			}
			b.tally = existing.NewOldTally(terms)
		}
	}

	tree := newTree(opt.TWRoot)
	for _, file := range src.KR.Files() {
		data, err := b.buildFile(file)
		if err != nil {
			return nil, err
		}
		if err := tree.write(langPath(opt, file), data); err != nil {
			return nil, err
		}
	}
	if len(b.added) > 0 {
		pending = append(pending, b.added...)
		if err := st.SavePending(pending); err != nil {
			return nil, err
		}
	}
	slices.Sort(b.rep.FrozenFallback)
	b.finish()
	if b.rep.MissingGlyphs, err = in.missingGlyphs(&b.glyphs); err != nil {
		return nil, err
	}

	version := newVersionInfo(opt, tr)
	info, err := encodeJSON(version)
	if err != nil {
		return nil, err
	}
	for _, f := range []struct {
		rel  string
		data []byte
	}{
		{configPath, configJSON(opt.LangDir)},
		{langPath(opt, "Info/LICENSE-CONTENT"), in.license},
		{langPath(opt, "Info/version.json"), info},
	} {
		if err := tree.write(f.rel, f.data); err != nil {
			return nil, err
		}
	}
	if err := tree.prune(); err != nil {
		return nil, err
	}

	zips, err := writeZips(opt, tree, in)
	if err != nil {
		return nil, err
	}
	if err := writeMeta(st, opt, newInfo(version, tr, zips, b.rep, b.inconsistentAll, seen), sum, pending); err != nil {
		return nil, err
	}
	return b.rep, nil
}

// buildFile returns the output of one KR file: the KR tree with every unit
// filled in.
func (b *builder) buildFile(file string) ([]byte, error) {
	kr, err := b.src.KR.Doc(file)
	if err != nil {
		return nil, err
	}
	out := kr.Clone() // the parsed tree is shared; never modify it
	al := align.NewFileAligner(file, kr, b.src.EN, b.src.JP)
	frozen := config.MatchAny(b.opt.Frozen, file)
	for _, u := range units.Extract(file, kr) {
		text, err := b.fill(u, frozen, al)
		if err != nil {
			return nil, fmt.Errorf("build: %s: %w", file, err)
		}
		b.glyphs.addString(text)
		if text == u.KR {
			continue
		}
		if err := out.SetString(u.Pointer, text); err != nil {
			return nil, fmt.Errorf("build: %s: %w", file, err)
		}
	}
	en, jp, err := al.Unmatched()
	if err != nil {
		return nil, fmt.Errorf("build: %s: %w", file, err)
	}
	if en+jp > 0 {
		b.misaligned = append(b.misaligned, FileMisaligned{File: file, EN: en, JP: jp})
	}
	return jsonx.Marshal(out), nil
}

// fill decides the text of one unit and counts what it shows (Report.Shown).
//
// The field rules come first (spec 8.2): a unit they do not translate keeps
// KR, whatever its record says. Otherwise a record is used when it was made
// from the KR text there is now, or when the file is frozen. A used record
// that breaks the structure of KR is replaced by EN (or KR) and queued for
// translation again, with the broken text as the draft; a frozen file's record
// is replaced the same way but never queued, since a frozen file is not
// translated again (it is listed in FrozenFallback). A unit with no usable
// record shows EN (or KR) and counts as pending.
func (b *builder) fill(u units.Unit, frozen bool, al *align.FileAligner) (string, error) {
	text, shown, err := b.decide(u, frozen, al)
	if err != nil {
		return "", err
	}
	b.rep.Shown[shown]++
	return text, nil
}

// decide is fill without the count of Shown: it returns the text of the unit
// and what it shows, a key of Report.Shown.
func (b *builder) decide(u units.Unit, frozen bool, al *align.FileAligner) (text, shown string, err error) {
	key := units.Key(u.File, u.Pointer)
	rec, ok := b.records[key]
	usable := ok && (frozen || rec.KRSha == units.KRSha(u.KR))

	en, jp, err := al.Get(u.Pointer)
	if err != nil {
		return "", "", err
	}
	if !b.fr.Classify(u, en, jp).Translate {
		switch {
		case usable && rec.Origin == store.OriginCopy:
			b.countUsed(rec, true)
		case usable:
			b.rep.FieldRuleOverride++
		}
		// KR as it is, with or without a record: nothing is translated.
		return u.KR, store.OriginCopy, nil
	}
	if usable {
		text, shown := b.useRecord(u, key, rec, frozen, en, jp)
		return text, shown, nil
	}
	b.rep.Pending++
	b.rep.StandIn++
	if en == nil || en.Value == "" {
		b.rep.Unaligned++
	}
	return standIn(u, en), ShownStandIn, nil
}

// countUsed counts rec among the records used. review says whether its
// needs_review flag counts too.
func (b *builder) countUsed(rec store.Record, review bool) {
	b.rep.Origins[rec.Origin]++
	if review && slices.Contains(rec.Flags, store.FlagNeedsReview) {
		b.rep.NeedsReview++
	}
}

// useRecord returns the text a used record gives the unit, and what that
// shows: the origin of the record, or ShownStandIn when the text is a
// stand-in (a record of origin fallback, or a record replaced for breaking
// the structure of KR). en and jp are the texts aligned to the unit, nil when
// none.
func (b *builder) useRecord(u units.Unit, key string, rec store.Record, frozen bool, en, jp *align.LangAligned) (text, shown string) {
	// A copy is the KR text itself: there is nothing to check, and the Hangul
	// it keeps would only flood the warnings.
	if rec.Origin == store.OriginCopy {
		b.countUsed(rec, true)
		return rec.ZH, store.OriginCopy
	}
	// Existing: a record that shipped is held to the structure of KR only; the
	// wording rules may have changed since.
	issues := validate.Check(u.KR, rec.ZH, validate.Context{Existing: true, File: u.File, ScriptOK: b.scriptOK(u.File), Terms: b.terms(u.File, u.KR)})
	if named := b.nameIssue(u, key, rec, en, jp); named != nil {
		issues = append(issues, *named)
	}
	// Every used record counts its warnings, whether or not its text is
	// written, so that Origins, NeedsReview and Warnings cover the same records.
	for _, is := range issues {
		if is.Severity == validate.SeverityWarn {
			if b.opt.Accepted[itemid.UnitWarning(is.Code, is.Term, is.TermZH, u.KR, rec.ZH)] {
				continue
			}
			b.rep.Warnings[is.Code]++
			// One unit can warn twice of one code; its key is one sample.
			if keys := b.warnKeys[is.Code]; len(keys) == 0 || keys[len(keys)-1] != key {
				b.warnKeys[is.Code] = append(keys, key)
			}
			if is.Code == existing.CodeNameDiffers && rec.TranslatedIn(b.opt.RawTag) {
				b.addNameDiff(b.opt.Names.Lookup(u.KR), NameDiffUnit{Key: key, ZH: rec.ZH, Note: b.noteOn(u, key), Kind: NameDiffField})
			}
		}
	}
	b.compareWithExisting(u, key, rec)
	if !validate.HasHard(issues) {
		b.countUsed(rec, true)
		if rec.Origin == store.OriginFallback {
			b.rep.StandIn++ // the stand-in text is the record's text
			return rec.ZH, ShownStandIn
		}
		b.addRendering(u, key, rec.ZH, rec.TranslatedIn(b.opt.RawTag))
		return rec.ZH, rec.Origin
	}
	b.rep.StandIn++

	if frozen {
		// Not translated again, so not queued: a person fixes it. It is
		// listed in FrozenFallback and nowhere else, so a needs_review flag
		// does not count it a second time.
		b.countUsed(rec, false)
		b.rep.FrozenFallback = append(b.rep.FrozenFallback, key)
	} else {
		b.countUsed(rec, true)
		b.rep.StructuralFallback++
		if !b.queued[key] {
			b.queued[key] = true
			b.added = append(b.added, store.PendingItem{Key: key, Reason: ReasonStructuralFail, Draft: rec.ZH})
		}
	}
	return standIn(u, en), ShownStandIn
}

// standIn is the text shown for a unit that has no translation: the EN text
// aligned to it, or KR.
func standIn(u units.Unit, en *align.LangAligned) string {
	if en != nil && en.Value != "" {
		return en.Value
	}
	return u.KR
}

// comparing reports whether the build compares the translations with the
// existing names and the old translations (see Options.Names).
func (b *builder) comparing() bool {
	return b.opt.Names != nil && b.opt.Rules != nil
}

// nameIssue is the name-differs warning of rec, the record of unit u (key),
// or nil: only a record of origin claude of a name field is checked, against
// the variant of the name that en or jp, the texts aligned to the unit, point
// at (see existing.Index.Differs).
func (b *builder) nameIssue(u units.Unit, key string, rec store.Record, en, jp *align.LangAligned) *validate.Issue {
	if !b.comparing() || rec.Origin != store.OriginClaude || !b.opt.Names.IsName(u.Field, u.KR) {
		return nil
	}
	return b.opt.Names.Differs(u.Field, u.KR, rec.ZH, b.noteOn(u, key), alignedText(jp), alignedText(en))
}

// noteOn is what the translator noted on unit u (key) when translating the KR
// text it has now: those notes of key, joined by newlines in the order they
// were added. Notes made from an earlier KR text are left out.
func (b *builder) noteOn(u units.Unit, key string) string {
	return NoteText(b.notes[key], u.KR)
}

// NoteText is what the translator noted on a unit when translating the KR
// text kr: those of notes (the notes of one unit) made from kr, joined by
// newlines in the order they were added. The review page uses it too, so that
// it lists the name-differs warnings the build counts.
func NoteText(notes []store.Note, kr string) string {
	sha := units.KRSha(kr)
	var texts []string
	for _, n := range notes {
		if n.KRSha == sha {
			texts = append(texts, n.Note)
		}
	}
	return strings.Join(texts, "\n")
}

// alignedText is the text of a, or "" when nothing is aligned.
func alignedText(a *align.LangAligned) string {
	if a == nil {
		return ""
	}
	return a.Value
}

// compareWithExisting weighs rec, the record of unit u (key) and not a copy,
// against the existing translations, whether or not it keeps the structure of
// KR. An old record counts in the tally of the provisional terms of
// Report.TermDiffs. A record of this raw tag adds u to the NameDiff of each
// name its KR mentions, other than the name a name field is, that is not
// declined and whose renderings its text all lacks, and to the TermDiff of
// each term its KR holds and whose rendering its text uses.
func (b *builder) compareWithExisting(u units.Unit, key string, rec store.Record) {
	if !b.comparing() {
		return
	}
	if existing.IsOld(rec, b.opt.RawTag) {
		if len(b.termDiffs) > 0 {
			b.tally.Add(key, u.KR, rec.ZH)
		}
		return
	}
	if !rec.TranslatedIn(b.opt.RawTag) {
		return
	}
	ix := b.opt.Names
	var exact *existing.Name // the name a name field is, warned of by nameIssue
	if ix.IsName(u.Field, u.KR) {
		exact = ix.Lookup(u.KR)
	}
	for _, n := range ix.Mentions(u.KR, nil) {
		// By Lookup, so that a table listing the name twice is one name.
		if (exact != nil && ix.Lookup(n.KR) == exact) || ix.Declined(n.KR) || n.Has(rec.ZH) {
			continue
		}
		b.addNameDiff(n, NameDiffUnit{Key: key, ZH: rec.ZH, Note: b.noteOn(u, key), Kind: NameDiffMention})
	}
	if len(b.termDiffs) > 0 {
		plain, text := validate.MatchText(u.KR), validate.MatchText(rec.ZH)
		for _, d := range b.termDiffs {
			if d.term.OccursIn(plain) && strings.Contains(text, d.term.ZH) {
				d.keys = append(d.keys, key)
			}
		}
	}
}

// addNameDiff lists unit under the existing name n. A unit is listed once
// under a name.
func (b *builder) addNameDiff(n *existing.Name, unit NameDiffUnit) {
	d, ok := b.nameDiffs[n.KR]
	if !ok {
		d = &NameDiff{KR: n.KR, Existing: renderingsOf(n)}
		b.nameDiffs[n.KR] = d
	}
	if last := len(d.Units) - 1; last >= 0 && d.Units[last].Key == unit.Key {
		return
	}
	d.Units = append(d.Units, unit)
}

// renderingsOf returns the renderings of every variant of n, the units of one
// rendering summed, the most units first and then by ZH, at most
// maxNameRenderings; never nil.
func renderingsOf(n *existing.Name) []existing.Rendering {
	byZH := map[string]int{}
	for _, v := range n.Variants {
		for _, r := range v.Renderings {
			byZH[r.ZH] += r.Units
		}
	}
	out := make([]existing.Rendering, 0, len(byZH))
	for zh, count := range byZH {
		out = append(out, existing.Rendering{ZH: zh, Units: count})
	}
	slices.SortFunc(out, func(x, y existing.Rendering) int {
		return cmp.Or(cmp.Compare(y.Units, x.Units), cmp.Compare(x.ZH, y.ZH))
	})
	return out[:min(len(out), maxNameRenderings)]
}

// differingTerms returns the provisional terms registered at rawTag that
// differ from the existing names or the old translations, sorted by KR. As in
// batch.ActiveProvisional, of the records of one KR the first that is not a
// conflict decides: it must have Old and be one of batch.ProvisionalKRs, and
// its suggestion "term-new/<kr>" must not be declined.
func differingTerms(all []store.ProvisionalTerm, r *rules.Rules, rawTag string, declined map[string]bool) []*termDiff {
	listed := batch.ProvisionalKRs(all, r)
	seen := map[string]bool{}
	var out []*termDiff
	for _, p := range all {
		if p.Conflict || seen[p.KR] {
			continue
		}
		seen[p.KR] = true
		if p.Old == nil || p.RawTag != rawTag || !listed[p.KR] || declined["term-new/"+p.KR] {
			continue
		}
		out = append(out, &termDiff{p: p, term: rules.Term{KR: p.KR, ZH: p.ZH, Match: rules.MatchWord}})
	}
	slices.SortFunc(out, func(x, y *termDiff) int { return cmp.Compare(x.p.KR, y.p.KR) })
	return out
}

// renderKey is what makes two units the same text to translate: the family
// and the field they are in, and the KR text.
type renderKey struct{ family, field, kr string }

// rendering is what the units of one renderKey were translated to: the
// translations in the order they were met, at most maxRenderings, each with
// its first unit and the number of units that show it. newKeys are the units
// translated for this raw tag, in the order they were met.
type rendering struct {
	units    int
	variants []variant
	newKeys  []string
}

type variant struct {
	zh, key string
	units   int
}

// addRendering notes that unit u (key) shows zh, the translation of a person,
// of the translator or of Paratranz; isNew says it was translated for this raw
// tag.
func (b *builder) addRendering(u units.Unit, key, zh string, isNew bool) {
	k := renderKey{b.familyOf(u.File), u.Field, u.KR}
	g, ok := b.renders[k]
	if !ok {
		g = &rendering{}
		b.renders[k] = g
	}
	g.units++
	if isNew {
		g.newKeys = append(g.newKeys, key)
	}
	for i := range g.variants {
		if g.variants[i].zh == zh {
			g.variants[i].units++
			return
		}
	}
	if len(g.variants) < maxRenderings {
		g.variants = append(g.variants, variant{zh, key, 1})
	}
}

// scriptOK is what the kana and Hangul checks leave alone in a unit of file:
// the approved renderings for file with kana or Hangul; none without rules.
func (b *builder) scriptOK(file string) []string {
	return b.scripts.For(file)
}

// terms are the glossary terms u.KR contains in file, weighed with the
// provisional terms in force (batch.ValidateTermsIn); none without rules.
func (b *builder) terms(file, kr string) []validate.Term {
	if b.opt.Rules == nil {
		return nil
	}
	return batch.ValidateTermsIn(b.opt.Rules, b.provisional, file, kr)
}

// familyOf is the family of file; every file is of one family without rules.
func (b *builder) familyOf(file string) string {
	if b.opt.Rules == nil {
		return ""
	}
	family, ok := b.families[file]
	if !ok {
		family = b.opt.Rules.Family(file)
		b.families[file] = family
	}
	return family
}

// finish completes the report with what it can only know once every unit has
// been met: the samples of each warning, the texts with different
// translations, the worst misaligned files, the records written for this raw
// tag, and the translations that differ from the existing ones.
func (b *builder) finish() {
	for code, keys := range b.warnKeys {
		slices.Sort(keys)
		b.rep.WarningSamples[code] = keys[:min(len(keys), maxWarningSamples)]
	}

	var groups []Inconsistency
	for k, g := range b.renders {
		if len(g.variants) < 2 {
			continue
		}
		if b.opt.Declined[itemid.Inconsistent(k.family, k.field, k.kr)] {
			continue
		}
		inc := Inconsistency{Family: k.family, Field: k.field, KR: k.kr, Units: g.units, New: len(g.newKeys)}
		for _, v := range g.variants {
			inc.ZH, inc.Keys, inc.Counts = append(inc.ZH, v.zh), append(inc.Keys, v.key), append(inc.Counts, v.units)
		}
		if inc.New > 0 {
			inc.NewKeys = slices.Sorted(slices.Values(g.newKeys))
		}
		groups = append(groups, inc)
	}
	slices.SortFunc(groups, func(x, y Inconsistency) int {
		return cmp.Or(
			cmp.Compare(y.Units, x.Units), // the biggest first
			cmp.Compare(x.Family, y.Family),
			cmp.Compare(x.Field, y.Field),
			cmp.Compare(x.KR, y.KR),
		)
	})
	b.rep.Inconsistent = append(b.rep.Inconsistent, groups[:min(len(groups), maxInconsistent)]...)
	// The groups with units of this update are listed in the report and the
	// review issue however small they are, so none of them is cut.
	kept := min(len(groups), maxInconsistentAll)
	b.inconsistentAll = append([]Inconsistency{}, groups[:kept]...)
	for _, g := range groups[kept:] {
		if g.New > 0 {
			b.inconsistentAll = append(b.inconsistentAll, g)
		}
	}

	slices.SortFunc(b.misaligned, func(x, y FileMisaligned) int {
		return cmp.Or(
			cmp.Compare(y.EN+y.JP, x.EN+x.JP), // the worst first
			cmp.Compare(x.File, y.File),
		)
	})
	b.rep.Misaligned = append(b.rep.Misaligned, b.misaligned[:min(len(b.misaligned), maxMisaligned)]...)
	b.rep.MisalignedFiles = len(b.misaligned)

	for _, rec := range b.records {
		if rec.Origin == store.OriginClaude && rec.Since == b.opt.RawTag {
			b.rep.NewTranslated++
		}
	}

	for _, d := range b.nameDiffs {
		slices.SortFunc(d.Units, func(x, y NameDiffUnit) int { return cmp.Compare(x.Key, y.Key) })
		b.rep.NameDiffs = append(b.rep.NameDiffs, *d)
	}
	slices.SortFunc(b.rep.NameDiffs, func(x, y NameDiff) int { return cmp.Compare(x.KR, y.KR) })
	for _, d := range b.termDiffs {
		use, examples, _ := b.tally.Use(b.opt.Names, d.p.KR, d.p.ZH)
		if use.Existing == "" {
			use.Existing = d.p.Old.Existing
		}
		if examples == nil {
			examples = []existing.OldExample{}
		}
		keys := slices.Sorted(slices.Values(d.keys))
		b.rep.TermDiffs = append(b.rep.TermDiffs, TermDiff{
			KR: d.p.KR, ZH: d.p.ZH, Existing: use.Existing, OldUnits: use.Units, OldWith: use.With,
			Examples: examples, Keys: append([]string{}, keys[:min(len(keys), maxTermKeys)]...),
		})
	}
}

// newNotes returns the notes added since the previous build, the number of
// notes there are in all, and every note of each unit, by key, in the order
// they were added. The previous build counted the notes it reported in
// .meta/build.json; a count above the number of notes (the file was replaced)
// starts over from the first note.
func newNotes(st *store.Store) (notes []store.Note, seen int, byKey map[string][]store.Note, err error) {
	var prev Info
	if _, err := st.ReadMetaJSON(infoName, &prev); err != nil {
		return nil, 0, nil, err
	}
	all, err := st.LoadNotes()
	if err != nil {
		return nil, 0, nil, err
	}
	from := prev.NotesSeen
	if from < 0 || from > len(all) {
		from = 0
	}
	byKey = map[string][]store.Note{}
	for _, n := range all {
		byKey[n.Key] = append(byKey[n.Key], n)
	}
	return append([]store.Note{}, all[from:]...), len(all), byKey, nil
}
