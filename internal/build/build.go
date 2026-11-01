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
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/align"
	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// ReasonStructuralFail is the reason of a pending item Run adds for a record
// whose translation breaks the structure of its KR text.
const ReasonStructuralFail = "structural-fail"

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
}

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
}

// Run builds the language pack for src (the raw KR, EN and JP packs) from the
// records of the store at opt.TWRoot. It writes the pack into the tw worktree,
// adds a pending item for every record that fails the structure check, writes
// the zips to opt.OutDir and the build record and report to the store. The
// whole run holds the store lock.
//
// Every input is checked before anything is written; a font that does not match
// opt.FontSHA256 is ErrFontChecksum.
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
	b := &builder{
		opt: opt, src: src, fr: fr,
		records: records,
		queued:  make(map[string]bool, len(pending)),
		rep:     &Report{Origins: map[string]int{}, Warnings: map[string]int{}, FrozenFallback: []string{}, MissingGlyphs: []rune{}},
	}
	for _, it := range pending {
		b.queued[it.Key] = true
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
		if err := st.SavePending(append(pending, b.added...)); err != nil {
			return nil, err
		}
	}
	slices.Sort(b.rep.FrozenFallback)
	if b.rep.MissingGlyphs, err = in.missingGlyphs(&b.glyphs); err != nil {
		return nil, err
	}

	info, err := encodeJSON(newVersionInfo(opt))
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
	if err := writeMeta(st, opt, zips, b.rep); err != nil {
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
	return jsonx.Marshal(out), nil
}

// fill decides the text of one unit.
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
	key := units.Key(u.File, u.Pointer)
	rec, ok := b.records[key]
	usable := ok && (frozen || rec.KRSha == units.KRSha(u.KR))

	en, jp, err := al.Get(u.Pointer)
	if err != nil {
		return "", err
	}
	if !b.fr.Classify(u, en, jp).Translate {
		switch {
		case usable && rec.Origin == store.OriginCopy:
			b.countUsed(rec, true)
		case usable:
			b.rep.FieldRuleOverride++
		}
		return u.KR, nil
	}
	if usable {
		return b.useRecord(u, key, rec, frozen, al)
	}
	b.rep.Pending++
	if en == nil || en.Value == "" {
		b.rep.Unaligned++
	}
	return standIn(u, en), nil
}

// countUsed counts rec among the records used. review says whether its
// needs_review flag counts too.
func (b *builder) countUsed(rec store.Record, review bool) {
	b.rep.Origins[rec.Origin]++
	if review && slices.Contains(rec.Flags, store.FlagNeedsReview) {
		b.rep.NeedsReview++
	}
}

func (b *builder) useRecord(u units.Unit, key string, rec store.Record, frozen bool, al *align.FileAligner) (string, error) {
	// A copy is the KR text itself: there is nothing to check, and the Hangul
	// it keeps would only flood the warnings.
	if rec.Origin == store.OriginCopy {
		b.countUsed(rec, true)
		return rec.ZH, nil
	}
	// Existing: a record that shipped is held to the structure of KR only; the
	// wording rules may have changed since.
	issues := validate.Check(u.KR, rec.ZH, validate.Context{Existing: true, File: u.File})
	// Every used record counts its warnings, whether or not its text is
	// written, so that Origins, NeedsReview and Warnings cover the same records.
	for _, is := range issues {
		if is.Severity == validate.SeverityWarn {
			b.rep.Warnings[is.Code]++
		}
	}
	if !validate.HasHard(issues) {
		b.countUsed(rec, true)
		return rec.ZH, nil
	}

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
	en, _, err := al.Get(u.Pointer)
	if err != nil {
		return "", err
	}
	return standIn(u, en), nil
}

// standIn is the text shown for a unit that has no translation: the EN text
// aligned to it, or KR.
func standIn(u units.Unit, en *align.LangAligned) string {
	if en != nil && en.Value != "" {
		return en.Value
	}
	return u.KR
}
