// Package suggest finds what the rules could learn (spec 11.2): the groups of
// units that one text is translated differently in, the new speakers and
// files, the notes of the translator, the glossary candidates and the human
// adjustments whose key moved or expired. Every suggestion has an id that is
// the same each time the same thing is found, so that a suggestion a person
// turned down (rules/declined.yaml), or one that a rules PR holds already, is
// not made again.
//
// What a suggestion is worth is for the rules-review skill and the person who
// reviews the rules PR to say; this package only finds and names them.
package suggest

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/adjust"
	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/existing"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/itemid"
	"github.com/tool-jx3/limbus-translate/internal/plan"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
)

// The kinds of suggestion. The id of a suggestion is its kind, a "/" and what
// names the thing (sha8 is the first 8 lowercase hex digits of the sha256 of
// the text):
//
//	note/<key>/<sha8(note text)>               a note of the translator
//	speaker/<model>                            a new speaker the guides lack
//	family/<file>                              a new file of the default family
//	field/<add|remove>/<group or ->/<field>    a change of rules/fields.yaml
//	adjustment-moved/<old key>                 an adjustment whose key moved
//	adjustment-expired/<key>                   an adjustment that expired
//	name/<kr>                                  a name this update translated otherwise than the existing translations
//	term-new/<kr>                              a provisional term a translator registered
//	term/<kr>                                  a glossary term that is a candidate
//	inconsistent/<family>/<field>/<sha8(KR)>   one group of Info.InconsistentAll
//
// A proposal of where a glossary entry applies (internal/scope, the review
// page's scope tab) may be declined as a suggestion is, under its own id
// itemid.Scope, scope/<sha8(KR)>/<sha8(ZH)>/<kind>; KindScope names it.
// lct suggest does not make such proposals, so it is not one of Kinds.
const (
	KindInconsistent      = "inconsistent"
	KindSpeaker           = "speaker"
	KindField             = "field"
	KindFamily            = "family"
	KindNote              = "note"
	KindName              = "name"
	KindTerm              = "term"
	KindNewTerm           = "term-new"
	KindAdjustmentMoved   = "adjustment-moved"
	KindAdjustmentExpired = "adjustment-expired"
	KindScope             = "scope"
)

// Kinds lists the kinds in the order Run lists their suggestions, the order
// the doc above shows them: the few that a person answers at once first (a
// note is made only once, see the rules-review skill), the groups of
// different translations, of which a build finds hundreds, last.
var Kinds = []string{
	KindNote, KindSpeaker, KindFamily, KindField,
	KindAdjustmentMoved, KindAdjustmentExpired, KindName, KindNewTerm, KindTerm, KindInconsistent,
}

// MaxEvidence is the most rows of evidence a suggestion has.
const MaxEvidence = 10

// noGroup stands for the group of an identifier field in the id of a field
// suggestion.
const noGroup = "-"

// maxTitleText is how many characters of a KR text the title of a suggestion
// quotes.
const maxTitleText = 40

// maxNoteTitle is how many characters of a translator's note the title of its
// suggestion quotes; the evidence has the whole note.
const maxNoteTitle = 120

// The notes of the rows of a term-new suggestion that are not the unit that
// registered the term: another rendering of it, and an old unit that holds
// its KR.
const (
	noteConflict   = "衝突"
	noteOldExample = "舊譯例句"
)

// keptExamples is how many old units of a provisional term (TermDiff.Examples)
// its evidence keeps room for: the conflicting renderings leave room for them,
// and a name suggestion it is merged into keeps them ahead of its own rows.
const keptExamples = 2

// Suggestion is one thing the rules could learn.
type Suggestion struct {
	// ID is the same each time the same thing is found (see the kinds).
	ID string `json:"id"`
	// Kind is the part of the ID before the first "/".
	Kind string `json:"kind"`
	// Title says in one line what the suggestion is about.
	Title string `json:"title"`
	// Evidence is at most MaxEvidence rows that show it; never nil.
	Evidence []Evidence `json:"evidence"`
}

// Confirmed says that a person confirmed an item, and why: the reason and
// the date of its entry in DeclinedFile or AcceptedFile.
type Confirmed struct {
	Reason string `json:"reason"`
	Date   string `json:"date"`
}

// Evidence is one row of what a suggestion rests on. A field that does not
// apply is left empty.
type Evidence struct {
	// Key is the unit the row is about.
	Key string `json:"key,omitempty"`
	KR  string `json:"kr,omitempty"`
	ZH  string `json:"zh,omitempty"`
	// Note is the whole text of a translator's note, as it was written.
	Note string `json:"note,omitempty"`
	// Count says how many times; 0 when the row does not count.
	Count int `json:"count,omitempty"`
}

// Inputs is what Run looks at.
type Inputs struct {
	// RawTag is the current raw tag. The build report and the plan say
	// something about the snapshot they were made for; those of another one
	// are not looked at. Without a RawTag neither is.
	RawTag string
	// Rules is what the rules answer already.
	Rules *rules.Rules
	// Adjustments are the adjustments in rules/adjustments (adjust.Load).
	Adjustments []adjust.Adjustment
	// Build is .meta/build.json of the tw worktree, nil if there is none.
	Build *build.Info
	// Plan is .meta/plan.json of the tw worktree, nil if there is none.
	Plan *plan.Summary
	// Provisional are the records of the provisional terms (.meta/terms.jsonl),
	// conflicts included, of the KR that batch.ProvisionalKRs lists: those in
	// force, and those not in force because a person declined them or because
	// they differ from the old translations (Old). The caller has left out the
	// KR the glossary has and the too short ones. Run lists every KR it gets;
	// a declined one is left out (Exclude) or flagged as declined as the
	// suggestions of any kind are.
	Provisional []store.ProvisionalTerm
}

// Run lists the suggestions by kind, in the order of Kinds; the groups of
// different translations by their number of units, the biggest first; and by
// id among those that are equal so far, so that the same input gives the same
// order. A page of lct suggest is taken from this order. It leaves out what
// the rules
// answer already: a speaker a guide knows (as the character, as the same
// person at another time of life, or as not that character), a new file that
// the routing gives a family of its own now, an adjustment whose file has the
// new key already or does not have the adjustment any more. The plan keeps
// such lists until the next snapshot, so this is checked on every run.
//
// It does not know what a person turned down or what a rules PR holds; see
// Exclude.
func Run(in Inputs) ([]Suggestion, error) {
	if in.Rules == nil {
		return nil, errors.New("suggest: no rules to suggest against")
	}
	c := collector{byID: map[string]*Suggestion{}, units: map[string]int{}}
	if in.RawTag != "" && in.Build != nil && in.Build.RawTag == in.RawTag {
		c.fromBuild(*in.Build)
	}
	if in.RawTag != "" && in.Plan != nil && in.Plan.RawTag == in.RawTag {
		c.fromPlan(in.Plan, in.Rules, in.Adjustments)
	}
	c.fromGlossary(in.Rules.Glossary)
	var termDiffs []build.TermDiff
	if in.RawTag != "" && in.Build != nil && in.Build.RawTag == in.RawTag {
		termDiffs = in.Build.Report.TermDiffs
	}
	c.fromProvisional(in.Provisional, termDiffs)
	c.addTermExamples(termDiffs)
	c.mergeTermNewIntoName()

	out := make([]Suggestion, 0, len(c.byID))
	for _, s := range c.byID {
		if s.Evidence == nil {
			s.Evidence = []Evidence{}
		}
		out = append(out, *s)
	}
	slices.SortFunc(out, func(a, b Suggestion) int {
		return cmp.Or(
			cmp.Compare(slices.Index(Kinds, a.Kind), slices.Index(Kinds, b.Kind)),
			cmp.Compare(c.units[b.ID], c.units[a.ID]), // the biggest group first
			cmp.Compare(a.ID, b.ID),
		)
	})
	return out, nil
}

// Exclude drops the suggestions whose id is in ids. It returns the others,
// never nil, and how many it dropped: an id of ids that is not among the
// suggestions counts for nothing.
func Exclude(list []Suggestion, ids map[string]bool) (kept []Suggestion, excluded int) {
	kept = make([]Suggestion, 0, len(list))
	for _, s := range list {
		if ids[s.ID] {
			excluded++
			continue
		}
		kept = append(kept, s)
	}
	return kept, excluded
}

// sha8 is itemid.Sha8: the first 8 lowercase hex digits of the sha256 of text.
func sha8(text string) string { return itemid.Sha8(text) }

// oneLine folds the blanks of text, line ends included, into single spaces.
func oneLine(text string) string { return strings.Join(strings.Fields(text), " ") }

// clip is text on one line, cut after max characters.
func clip(text string, max int) string {
	r := []rune(oneLine(text))
	if len(r) <= max {
		return string(r)
	}
	return string(r[:max]) + "…"
}

// collector gathers the suggestions by id. A thing that is found twice is one
// suggestion.
type collector struct {
	byID map[string]*Suggestion
	// units is the number of units of each group of different translations,
	// by id; Run lists the biggest first.
	units map[string]int
}

// add makes the suggestion of kind and rest (the id is "<kind>/<rest>"). When
// the id exists already the rows of evidence that it lacks are added, up to
// MaxEvidence; the first title stays.
func (c *collector) add(kind, rest, title string, evidence ...Evidence) {
	id := kind + "/" + rest
	s, ok := c.byID[id]
	if !ok {
		s = &Suggestion{ID: id, Kind: kind, Title: title}
		c.byID[id] = s
	}
	for _, e := range evidence {
		if len(s.Evidence) < MaxEvidence && !slices.Contains(s.Evidence, e) {
			s.Evidence = append(s.Evidence, e)
		}
	}
}

// fromBuild makes the suggestions of the build record: the groups of units
// that one text is translated differently in, and the notes of the translator.
// The groups are those of info.InconsistentAll, which has more than the
// twenty of the report (the biggest build.max_inconsistent of lct.yaml and
// every group with units of this update), or, in a build.json of an older lct that lacks the list, those of
// the report.
func (c *collector) fromBuild(info build.Info) {
	groups := info.InconsistentAll
	if groups == nil {
		groups = info.Report.Inconsistent
	}
	rep := info.Report
	for _, g := range groups {
		rows := make([]Evidence, len(g.ZH))
		for i, zh := range g.ZH {
			rows[i] = Evidence{KR: g.KR, ZH: zh}
			if i < len(g.Keys) {
				rows[i].Key = g.Keys[i]
			}
			if i < len(g.Counts) { // a build.json of an older lct has none
				rows[i].Count = g.Counts[i]
			}
		}
		// The translations come in the order the units were met; the
		// evidence does not depend on it.
		slices.SortFunc(rows, func(a, b Evidence) int {
			return cmp.Or(cmp.Compare(a.ZH, b.ZH), cmp.Compare(a.Key, b.Key))
		})
		id := itemid.Inconsistent(g.Family, g.Field, g.KR)
		rest := strings.TrimPrefix(id, KindInconsistent+"/")
		c.add(KindInconsistent, rest,
			fmt.Sprintf("不一致的譯法：%s（%s／%s，%d 個單位）", clip(g.KR, maxTitleText), g.Family, g.Field, g.Units),
			rows...)
		c.units[id] = max(c.units[id], g.Units)
	}
	for _, d := range rep.NameDiffs {
		if d.KR == "" || len(d.Units) == 0 {
			continue
		}
		main := ""
		if len(d.Existing) > 0 {
			main = d.Existing[0].ZH
		}
		rows := make([]Evidence, 0, len(d.Existing)+len(d.Units))
		for _, r := range d.Existing {
			rows = append(rows, Evidence{KR: d.KR, ZH: r.ZH, Count: r.Units})
		}
		for _, u := range d.Units {
			rows = append(rows, Evidence{Key: u.Key, KR: d.KR, ZH: u.ZH, Note: u.Note})
		}
		rest := strings.TrimPrefix(existing.SuggestionID(d.KR), KindName+"/")
		c.add(KindName, rest,
			fmt.Sprintf("與既有譯名不同：%s（既有 %s，這次 %d 個單位不同）", clip(d.KR, maxTitleText), main, len(d.Units)),
			rows...)
		c.units[existing.SuggestionID(d.KR)] = len(d.Units)
	}
	for _, n := range rep.Notes {
		if strings.TrimSpace(n.Note) == "" {
			continue
		}
		c.add(KindNote, strings.TrimPrefix(itemid.Note(n.Key, n.Note), KindNote+"/"),
			fmt.Sprintf("譯者備註（%s）：%s", n.Key, clip(n.Note, maxNoteTitle)),
			Evidence{Key: n.Key, Note: n.Note})
	}
}

// fromPlan makes the suggestions of the plan of the snapshot: what the new
// snapshot brought that the rules do not answer.
func (c *collector) fromPlan(sum *plan.Summary, r *rules.Rules, adjustments []adjust.Adjustment) {
	for _, model := range sum.NewSpeakers {
		if knowsSpeaker(r, model) {
			continue
		}
		c.add(KindSpeaker, model,
			fmt.Sprintf("新說話者「%s」：%s/ 還沒有這個 model", clip(model, maxTitleText), rules.CharactersDir),
			Evidence{KR: model})
	}
	for _, ch := range sum.FieldChanges {
		group := cmp.Or(ch.Group, noGroup)
		c.add(KindField, ch.Op+"/"+group+"/"+ch.Field, fieldTitle(ch))
	}
	if def := r.DefaultFamily(); def != "" {
		for _, file := range sum.NewFilesDefaultFamily {
			if r.Family(file) != def {
				continue // the routing has a family for the file now
			}
			c.add(KindFamily, file, fmt.Sprintf("新檔案 %s 落在預設族群「%s」，還沒有專屬的族群", file, def))
		}
	}

	byKey := map[string]adjust.Adjustment{}
	for _, a := range adjustments {
		if _, dup := byKey[a.Key]; !dup {
			byKey[a.Key] = a
		}
	}
	for _, m := range sum.Adjustments.Moved {
		a, ok := byKey[m.From]
		if !ok {
			continue // the file has the new key, or the adjustment is gone
		}
		c.add(KindAdjustmentMoved, m.From,
			fmt.Sprintf("調整的 key 已移動：%s → %s（%s）", m.From, m.To, adjustmentFile(a)),
			Evidence{Key: m.From, ZH: a.ZH}, Evidence{Key: m.To})
	}
	for _, key := range sum.Adjustments.Expired {
		a, ok := byKey[key]
		if !ok {
			continue // the adjustment is gone
		}
		c.add(KindAdjustmentExpired, key,
			fmt.Sprintf("人工調整已過期：%s（%s）", key, adjustmentFile(a)),
			Evidence{Key: key, ZH: a.ZH})
	}
}

// fromGlossary makes the suggestion of every term that is a candidate. A KR
// text with several entries (they apply to different files) is one suggestion
// and the entries are its evidence.
func (c *collector) fromGlossary(terms []rules.Term) {
	for _, t := range terms {
		if t.Status != rules.StatusCandidate || t.KR == "" {
			continue
		}
		c.add(KindTerm, t.KR, fmt.Sprintf("詞彙候選：%s → %s", clip(t.KR, maxTitleText), clip(t.ZH, maxTitleText)),
			Evidence{KR: t.KR, ZH: t.ZH})
	}
}

// fromProvisional makes the suggestion of every KR of the provisional terms.
// The record of a KR that is no conflict is the term the batches use: the
// title says its rendering, and the evidence is the unit that registered it
// (its batch in Note) and then each conflicting rendering, leaving room for
// up to keptExamples of the old units its TermDiff of diffs has, which
// addTermExamples adds. A KR that has only conflicts is not suggested; there
// is no term to decide on.
func (c *collector) fromProvisional(terms []store.ProvisionalTerm, diffs []build.TermDiff) {
	room := map[string]int{} // the rows left for the old units of a KR
	for _, d := range diffs {
		room[d.KR] = max(room[d.KR], min(len(d.Examples), keptExamples))
	}
	seen := map[string]bool{}
	for _, t := range terms {
		if t.Conflict || t.KR == "" {
			continue
		}
		first := !seen[t.KR]
		seen[t.KR] = true
		title := fmt.Sprintf("暫定詞：%s → %s", clip(t.KR, maxTitleText), clip(t.ZH, maxTitleText))
		rows := []Evidence{{Key: t.Key, KR: t.KR, ZH: t.ZH, Note: t.Batch}}
		if first && t.Old != nil {
			title = fmt.Sprintf("暫定詞（與舊譯不同，未生效）：%s → %s", clip(t.KR, maxTitleText), clip(t.ZH, maxTitleText))
			rows = append(rows, Evidence{KR: t.KR, ZH: t.Old.Existing, Count: t.Old.Units,
				Note: fmt.Sprintf("與舊譯不同：舊譯 %d 個單位，含暫定譯法 %d 個", t.Old.Units, t.Old.With)})
		}
		c.add(KindNewTerm, t.KR, title, rows...)
	}
	for _, t := range terms {
		if s, ok := c.byID[KindNewTerm+"/"+t.KR]; ok && t.Conflict && len(s.Evidence) < MaxEvidence-room[t.KR] {
			c.add(KindNewTerm, t.KR, "", Evidence{Key: t.Key, KR: t.KR, ZH: t.ZH, Note: noteConflict})
		}
	}
}

// addTermExamples adds the examples of the old translation of each TermDiff to
// the term-new suggestion of its KR, when there is one.
func (c *collector) addTermExamples(diffs []build.TermDiff) {
	for _, d := range diffs {
		if _, ok := c.byID[KindNewTerm+"/"+d.KR]; !ok {
			continue
		}
		rows := make([]Evidence, len(d.Examples))
		for i, ex := range d.Examples {
			rows[i] = Evidence{Key: ex.Key, KR: ex.KR, ZH: ex.ZH, Note: noteOldExample}
		}
		c.add(KindNewTerm, d.KR, "", rows...)
	}
}

// mergeTermNewIntoName folds the term-new suggestion of a KR that has a name
// suggestion too into the name suggestion: the name is what a person decides
// on, and the provisional term is part of its evidence. A name may have rows
// enough of its own to fill the evidence, so the rows that say what the term
// is come first (see termRowsFirst), then those of the name, then the rest of
// the term's, up to MaxEvidence.
func (c *collector) mergeTermNewIntoName() {
	for id, s := range c.byID {
		if s.Kind != KindNewTerm {
			continue
		}
		kr := strings.TrimPrefix(id, KindNewTerm+"/")
		name, ok := c.byID[existing.SuggestionID(kr)]
		if !ok {
			continue
		}
		first, rest := termRowsFirst(s.Evidence)
		own := name.Evidence
		name.Evidence = nil
		c.add(KindName, kr, "", first...)
		c.add(KindName, kr, "", own...)
		c.add(KindName, kr, "", rest...)
		delete(c.byID, id)
		delete(c.units, id)
	}
}

// termRowsFirst splits the rows of a term-new suggestion into those that a
// name suggestion it is merged into puts before its own: the units that
// registered the term, how it differs from the old translations and the
// first keptExamples old units; and the rest (the conflicting renderings
// and the other old units). Both keep their order.
func termRowsFirst(rows []Evidence) (first, rest []Evidence) {
	examples := 0
	for _, e := range rows {
		switch {
		case e.Note == noteConflict:
			rest = append(rest, e)
		case e.Note == noteOldExample && examples == keptExamples:
			rest = append(rest, e)
		default:
			if e.Note == noteOldExample {
				examples++
			}
			first = append(first, e)
		}
	}
	return first, rest
}

// knowsSpeaker reports whether the guides say who the speaker model is: the
// character, the same person at another time of life, or not the character a
// name in it suggests.
func knowsSpeaker(r *rules.Rules, model string) bool {
	if _, _, ok := r.CharacterByModel(model); ok {
		return true
	}
	for _, c := range r.Characters {
		if slices.Contains(c.ModelsNotThis, model) {
			return true
		}
	}
	return false
}

// fieldTitle says what a change of the field rules does.
func fieldTitle(ch fields.Change) string {
	verb := map[string]string{"add": "新增", "remove": "移除"}[ch.Op]
	if verb == "" {
		verb = ch.Op
	}
	if ch.Group == "" {
		return fmt.Sprintf("欄位規則：%s identifier_fields「%s」", verb, ch.Field)
	}
	return fmt.Sprintf("欄位規則：%s group_fields「%s.%s」", verb, ch.Group, ch.Field)
}

// adjustmentFile is the file of the adjustment a, relative to the repo root.
func adjustmentFile(a adjust.Adjustment) string {
	if a.File == "" {
		return adjust.Dir
	}
	return adjust.Dir + "/" + a.File
}
