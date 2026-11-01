// Package bootstrap seeds the translation store from fixed versions of three
// upstream translations (spec 8.3): the Paratranz zh-TW translation, which is
// adopted where it is good, and LTM zh-TW and LLC zh-CN, which only serve as
// references for the units Claude translates later.
//
// Run is pure: it reads packs and returns what the store should hold. Writing
// it is up to the caller.
package bootstrap

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tool-jx3/limbus-translate/internal/align"
	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/names"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// Keys of Result.Stats.
const (
	StatCopy            = "copy"                // not translated; the KR text is kept
	StatParatranz       = "paratranz"           // adopted from Paratranz
	StatParatranzRepair = "paratranz-repair"    // Paratranz text failed validation; queued with it as the draft
	StatNew             = "new"                 // queued for translation
	StatUnalignedPZ     = "unaligned-paratranz" // no Paratranz string for the unit
	StatFrozen          = "frozen"              // adopted in a frozen file (also counted in paratranz)
	StatRefsLLC         = "refs-llc"            // queued items that got an LLC reference
	StatRefsLTM         = "refs-ltm"            // queued items that got an LTM reference
)

// The reasons of the pending items Run creates. They double as the Stats key
// that counts the items of that reason.
const (
	reasonNew             = StatNew
	reasonParatranzRepair = StatParatranzRepair
)

// ForeignNamesFile is the name, below the store's .meta directory, of the
// JSON list of ForeignNames that "lct bootstrap" writes (Store.WriteMetaJSON)
// and later steps read (Store.ReadMetaJSON).
const ForeignNamesFile = "foreign-names.json"

// minForeignNameRunes is the fewest characters a foreign name has.
const minForeignNameRunes = 2

// scenarioModelCodes lists the speakers. Its names are compared between our
// translation and LTM's (ForeignNames).
const scenarioModelCodes = "ScenarioModelCodes-AutoCreated.json"

// Inputs are the packs a bootstrap reads.
type Inputs struct {
	// Raw holds the official KR, EN and JP packs.
	Raw pack.Set
	// Paratranz is the primary translation. LTM and LLC are references and
	// may be nil. All three are opened with an empty prefix.
	Paratranz, LTM, LLC *pack.Pack
	// RawTag is the raw snapshot tag the packs of Raw come from. It becomes
	// the Since of every record.
	RawTag string
	// RulesCommit names the commit of the rules the import is made under
	// (see gitx.Describe). It becomes the Rules of every record.
	RulesCommit string
	// Frozen lists the file patterns of lct.yaml's frozen list (see
	// config.MatchAny).
	Frozen []string
}

// Result is what a bootstrap decided.
type Result struct {
	// Records holds the units that are settled: copied or adopted from
	// Paratranz. Units that are queued have no record.
	Records map[string]store.Record
	// Pending holds the units left for Claude, in KR file order and then
	// unit order.
	Pending []store.PendingItem
	// Stats counts what happened under the Stat* keys. Every key is present.
	Stats map[string]int
	// ForeignNames are the names LTM gives to speakers that we name
	// differently (see ForeignNames). It is never nil.
	ForeignNames []string
}

// Run decides, for every unit of every KR file, whether it is copied,
// adopted from Paratranz or queued (spec 8.3):
//
//  1. A unit that is not translated is copied.
//  2. A unit with no Paratranz string aligned to it is queued ("new").
//  3. A Paratranz string that is not a translation (ValidParatranz) is queued
//     ("new").
//  4. Otherwise the string gets its keyword IDs restored and is normalized
//     and validated. Without a hard issue it is adopted; with one it is
//     queued ("paratranz-repair") with the normalized text as the draft.
//
// Files matching Inputs.Frozen skip steps 3 and 4: a non-empty Paratranz
// string without Hangul is adopted as it stands, flagged needs_review if the
// validation finds a hard issue.
//
// Queued units carry the LLC and LTM strings aligned to them as Refs.
func Run(in Inputs, fr *fields.Rules, r *rules.Rules) (*Result, error) {
	switch {
	case in.RawTag == "":
		return nil, errors.New("bootstrap: no raw tag")
	case in.Raw.KR == nil || in.Raw.EN == nil || in.Raw.JP == nil:
		return nil, errors.New("bootstrap: the KR, EN and JP packs are all required")
	case in.Paratranz == nil:
		return nil, errors.New("bootstrap: no Paratranz pack")
	case fr == nil || r == nil:
		return nil, errors.New("bootstrap: field rules and rules are required")
	}

	keywords, err := keywordTable(in.Paratranz)
	if err != nil {
		return nil, err
	}
	b := &builder{
		in:       in,
		fr:       fr,
		rules:    r,
		keywords: keywords,
		res: &Result{
			Records: map[string]store.Record{},
			Stats: map[string]int{
				StatCopy: 0, StatParatranz: 0, StatParatranzRepair: 0,
				StatNew: 0, StatUnalignedPZ: 0, StatFrozen: 0,
				StatRefsLLC: 0, StatRefsLTM: 0,
			},
		},
	}
	for _, file := range in.Raw.KR.Files() {
		if err := b.file(file); err != nil {
			return nil, err
		}
	}
	if b.res.ForeignNames, err = foreignNamesOf(in.Paratranz, in.LTM); err != nil {
		return nil, err
	}
	return b.res, nil
}

// keywordTable reads the keyword names from the Paratranz files that hold
// them (the same families names.KeywordTable reads).
func keywordTable(pz *pack.Pack) (map[string]string, error) {
	docs := map[string]*jsonx.Node{}
	for _, file := range pz.Files() {
		if !names.IsKeywordFile(file) {
			continue
		}
		doc, err := pz.Doc(file)
		if err != nil {
			return nil, fmt.Errorf("bootstrap: %w", err)
		}
		docs[file] = doc
	}
	return names.KeywordTable(docs), nil
}

// foreignNamesOf compares the speaker tables of the Paratranz and LTM packs.
// A pack without the file has no names.
func foreignNamesOf(pz, ltm *pack.Pack) ([]string, error) {
	load := func(p *pack.Pack) (*jsonx.Node, error) {
		if p == nil || !p.Has(scenarioModelCodes) {
			return nil, nil
		}
		doc, err := p.Doc(scenarioModelCodes)
		if err != nil {
			return nil, fmt.Errorf("bootstrap: %w", err)
		}
		return doc, nil
	}
	ours, err := load(pz)
	if err != nil {
		return nil, err
	}
	theirs, err := load(ltm)
	if err != nil {
		return nil, err
	}
	return ForeignNames(ours, theirs), nil
}

// ForeignNames returns the names LTM gives to speakers that we name
// differently: for every id in the dataList of both ScenarioModelCodes-
// AutoCreated documents whose name differs, LTM's name (LTM calls 뫼르소
// "默爾索", we call it "莫梭"). They are names the translation must not use,
// so a name that ours also uses for any id is left out, and so is a name of
// fewer than two characters: it would match ordinary words ("空" in "空洞").
// The result is sorted and has no duplicates; it is empty, not nil, when there
// is none.
func ForeignNames(ours, ltm *jsonx.Node) []string {
	mine, theirs := names.SpeakerTable(ours), names.SpeakerTable(ltm)
	used := make(map[string]bool, len(mine))
	for _, name := range mine {
		used[name] = true
	}
	found := []string{}
	for id, name := range theirs {
		if mineName, ok := mine[id]; ok && mineName != name && !used[name] && utf8.RuneCountInString(name) >= minForeignNameRunes {
			found = append(found, name)
		}
	}
	slices.Sort(found)
	return slices.Compact(found)
}

// ValidParatranz reports whether zh, the Paratranz string for the KR text kr,
// is a translation.
//
// A KR text without Hangul (ellipses, "…!", a Latin name or label) has nothing
// to translate, so Paratranz holds the right text however it looks: zh is
// valid if it is not blank and has no Hangul. It need not have Han characters
// and may equal kr. But ParatranzUploader fills untranslated entries with the
// EN text, so zh is not valid if it is that fill: zh equals en, en differs from
// kr, and zh has an ASCII letter or two dots in a row (an EN text like "......"
// where KR has "……", or "The LCB & The LCD" where KR has "LCB&LCD").
//
// For a KR text with Hangul, ParatranzUploader fills untranslated entries with
// the EN text and the pack holds some KR text and symbol-only values, so zh is
// valid only if it differs from kr and from en (the EN text aligned to the
// unit, "" if there is none), has no Hangul and has at least one Han
// character.
func ValidParatranz(kr, en, zh string) bool {
	if !hasHangul(kr) {
		if strings.TrimSpace(zh) == "" || hasHangul(zh) {
			return false
		}
		isENFill := zh == en && en != kr && (strings.Contains(zh, "..") || strings.IndexFunc(zh, isASCIILetter) >= 0)
		return !isENFill
	}
	if zh == kr || zh == en {
		return false
	}
	return !hasHangul(zh) && strings.IndexFunc(zh, isHan) >= 0
}

func isASCIILetter(r rune) bool { return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' }

func isHan(r rune) bool { return unicode.Is(unicode.Han, r) }

func hasHangul(s string) bool {
	return strings.IndexFunc(s, func(r rune) bool { return unicode.Is(unicode.Hangul, r) }) >= 0
}

// builder holds what Run needs while it walks the KR files.
type builder struct {
	in       Inputs
	fr       *fields.Rules
	rules    *rules.Rules
	keywords map[string]string
	res      *Result
}

// alignedFile is the strings of the other packs aligned to one KR file, keyed
// by KR pointer. A pack without the file contributes an empty map.
type alignedFile struct {
	en, jp, pz, ltm, llc map[string]align.LangAligned
}

// file settles every unit of one KR file.
func (b *builder) file(file string) error {
	krDoc, err := b.in.Raw.KR.Doc(file)
	if err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}
	var a alignedFile
	for _, l := range []struct {
		src *pack.Pack
		dst *map[string]align.LangAligned
	}{
		{b.in.Raw.EN, &a.en}, {b.in.Raw.JP, &a.jp},
		{b.in.Paratranz, &a.pz}, {b.in.LTM, &a.ltm}, {b.in.LLC, &a.llc},
	} {
		if *l.dst, err = alignWith(l.src, file, krDoc); err != nil {
			return err
		}
	}

	frozen := config.MatchAny(b.in.Frozen, file)
	for _, u := range units.Extract(file, krDoc) {
		b.unit(u, a, frozen)
	}
	return nil
}

// alignWith aligns file of src to krDoc. A missing pack or file has nothing
// aligned.
func alignWith(src *pack.Pack, file string, krDoc *jsonx.Node) (map[string]align.LangAligned, error) {
	if src == nil || !src.Has(file) {
		return nil, nil
	}
	doc, err := src.Doc(file)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: %w", err)
	}
	return align.AlignLang(file, krDoc, doc), nil
}

// unit settles one unit: the steps of Run.
func (b *builder) unit(u units.Unit, a alignedFile, frozen bool) {
	en, hasEN := a.en[u.Pointer]
	jp, hasJP := a.jp[u.Pointer]
	var enRef, jpRef *align.LangAligned
	if hasEN {
		enRef = &en
	}
	if hasJP {
		jpRef = &jp
	}
	if !b.fr.Classify(u, enRef, jpRef).Translate {
		b.settle(u, u.KR, store.OriginCopy, nil)
		b.res.Stats[StatCopy]++
		return
	}

	pz, ok := a.pz[u.Pointer]
	if !ok {
		b.res.Stats[StatUnalignedPZ]++
		b.queue(u, a, reasonNew, "")
		return
	}

	if frozen && strings.TrimSpace(pz.Value) != "" && !hasHangul(pz.Value) {
		var flags []string
		if validate.HasHard(b.check(u.File, u.KR, pz.Value)) {
			flags = []string{store.FlagNeedsReview}
		}
		b.settle(u, pz.Value, store.OriginParatranz, flags)
		b.res.Stats[StatParatranz]++
		b.res.Stats[StatFrozen]++
		return
	}

	if !ValidParatranz(u.KR, en.Value, pz.Value) {
		b.queue(u, a, reasonNew, "")
		return
	}
	zh := validate.Normalize(u.KR, RestoreIDs(u.KR, pz.Value, b.keywords))
	if validate.HasHard(b.check(u.File, u.KR, zh)) {
		b.queue(u, a, reasonParatranzRepair, zh)
		return
	}
	b.settle(u, zh, store.OriginParatranz, nil)
	b.res.Stats[StatParatranz]++
}

// check validates zh as a new translation of kr, a text of file, under the
// approved glossary terms kr contains in that file.
func (b *builder) check(file, kr, zh string) []validate.Issue {
	return validate.Check(kr, zh, validate.Context{Terms: b.rules.ValidateTerms(file, kr)})
}

// settle records zh as the translation of u.
func (b *builder) settle(u units.Unit, zh, origin string, flags []string) {
	key := units.Key(u.File, u.Pointer)
	b.res.Records[key] = store.Record{
		Key:    key,
		KRSha:  units.KRSha(u.KR),
		ZH:     zh,
		Origin: origin,
		Since:  b.in.RawTag,
		Rules:  b.in.RulesCommit,
		Flags:  flags,
	}
}

// queue adds u to the pending items with the LLC and LTM strings as Refs,
// counting the items that get each.
func (b *builder) queue(u units.Unit, a alignedFile, reason, draft string) {
	var refs map[string]string
	for _, ref := range []struct {
		name, stat string
		aligned    map[string]align.LangAligned
	}{{"llc", StatRefsLLC, a.llc}, {"ltm", StatRefsLTM, a.ltm}} {
		if v, ok := ref.aligned[u.Pointer]; ok && v.Value != "" {
			if refs == nil {
				refs = map[string]string{}
			}
			refs[ref.name] = v.Value
			b.res.Stats[ref.stat]++
		}
	}
	b.res.Pending = append(b.res.Pending, store.PendingItem{
		Key:    units.Key(u.File, u.Pointer),
		Reason: reason,
		Draft:  draft,
		Refs:   refs,
	})
	b.res.Stats[reason]++
}
