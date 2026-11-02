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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/adjust"
	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/plan"
	"github.com/tool-jx3/limbus-translate/internal/rules"
)

// The kinds of suggestion. The id of a suggestion is its kind, a "/" and what
// names the thing (sha8 is the first 8 lowercase hex digits of the sha256 of
// the text):
//
//	inconsistent/<family>/<field>/<sha8(KR)>   one group of Info.InconsistentAll
//	speaker/<model>                            a new speaker the guides lack
//	field/<add|remove>/<group or ->/<field>    a change of rules/fields.yaml
//	family/<file>                              a new file of the default family
//	note/<key>/<sha8(note text)>               a note of the translator
//	term/<kr>                                  a glossary term that is a candidate
//	adjustment-moved/<old key>                 an adjustment whose key moved
//	adjustment-expired/<key>                   an adjustment that expired
const (
	KindInconsistent      = "inconsistent"
	KindSpeaker           = "speaker"
	KindField             = "field"
	KindFamily            = "family"
	KindNote              = "note"
	KindTerm              = "term"
	KindAdjustmentMoved   = "adjustment-moved"
	KindAdjustmentExpired = "adjustment-expired"
)

// Kinds lists the kinds, in the order the doc above shows them.
var Kinds = []string{
	KindInconsistent, KindSpeaker, KindField, KindFamily,
	KindNote, KindTerm, KindAdjustmentMoved, KindAdjustmentExpired,
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
}

// Run lists the suggestions, sorted by id. It leaves out what the rules
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
	c := collector{byID: map[string]*Suggestion{}}
	if in.RawTag != "" && in.Build != nil && in.Build.RawTag == in.RawTag {
		c.fromBuild(*in.Build)
	}
	if in.RawTag != "" && in.Plan != nil && in.Plan.RawTag == in.RawTag {
		c.fromPlan(in.Plan, in.Rules, in.Adjustments)
	}
	c.fromGlossary(in.Rules.Glossary)

	out := make([]Suggestion, 0, len(c.byID))
	for _, s := range c.byID {
		if s.Evidence == nil {
			s.Evidence = []Evidence{}
		}
		out = append(out, *s)
	}
	slices.SortFunc(out, func(a, b Suggestion) int { return cmp.Compare(a.ID, b.ID) })
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

// sha8 is the first 8 lowercase hex digits of the sha256 of text.
func sha8(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:4])
}

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
// twenty of the report, or, in a build.json of an older lct that lacks the
// list, those of the report.
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
		}
		// The translations come in the order the units were met; the
		// evidence does not depend on it.
		slices.SortFunc(rows, func(a, b Evidence) int {
			return cmp.Or(cmp.Compare(a.ZH, b.ZH), cmp.Compare(a.Key, b.Key))
		})
		c.add(KindInconsistent, g.Family+"/"+g.Field+"/"+sha8(g.KR),
			fmt.Sprintf("不一致的譯法：%s（%s／%s，%d 個單位）", clip(g.KR, maxTitleText), g.Family, g.Field, g.Units),
			rows...)
	}
	for _, n := range rep.Notes {
		if strings.TrimSpace(n.Note) == "" {
			continue
		}
		c.add(KindNote, n.Key+"/"+sha8(n.Note),
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
