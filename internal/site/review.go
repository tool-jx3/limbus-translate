package site

import (
	"cmp"
	"errors"
	"slices"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/adjust"
	"github.com/tool-jx3/limbus-translate/internal/align"
	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/itemid"
	"github.com/tool-jx3/limbus-translate/internal/plan"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/suggest"
	"github.com/tool-jx3/limbus-translate/internal/units"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// ReviewInputs is what the items that need a person are computed from, besides
// the packs and records of Inputs.
type ReviewInputs struct {
	RawTag string
	// Build is .meta/build.json of the tw worktree; nil without one.
	Build *build.Info
	// Plan is .meta/plan.json; nil when there is none of RawTag.
	Plan        *plan.Summary
	Adjustments []adjust.Adjustment
	// Fields are the field rules the build used.
	Fields *fields.Rules
	// Frozen are the frozen patterns of lct.yaml.
	Frozen  []string
	Notes   []store.Note
	Pending []store.PendingItem
	// Declined are the entries of rules/declined.yaml, Accepted those of
	// rules/accepted.yaml.
	Declined []suggest.Declined
	Accepted []suggest.Declined
	// Provisional are the records of the provisional terms in force, which
	// the review page lists as term-new suggestions (see suggest.Inputs).
	Provisional []store.ProvisionalTerm
}

// Confirmed says that a person confirmed an item, and why.
type Confirmed struct {
	Reason string `json:"reason"`
	Date   string `json:"date"`
}

// Variant is one translation of a group of inconsistent translations, with
// every unit that shows it.
type Variant struct {
	ZH    string   `json:"zh"`
	Count int      `json:"count"`
	Keys  []string `json:"keys"`
	// JPEN are the units of the variant by their JP and EN, the most units
	// first; equal counts keep the order of the first key.
	JPEN []JPENGroup `json:"jpen"`
}

// JPENGroup is the units of a variant that have one JP and one EN string; JP
// or EN is nil where the units have no string in that language.
type JPENGroup struct {
	JP    *string  `json:"jp,omitempty"`
	EN    *string  `json:"en,omitempty"`
	Count int      `json:"count"`
	Keys  []string `json:"keys"`
}

// The classes of a group of inconsistent translations by its JP and EN
// (SuggestionItem.JPENClass).
const (
	// JPENSame: every unit has the same JP and EN.
	JPENSame = "same"
	// JPENExplained: more than one JP/EN pair, each with one translation.
	JPENExplained = "explained"
	// JPENPartial: some JP/EN pair has two translations or more.
	JPENPartial = "partial"
)

// SuggestionItem is one rule suggestion. Family, Field, KR, JP, EN and
// Variants are those of a group of inconsistent translations only.
type SuggestionItem struct {
	ID       string    `json:"id"`
	Kind     string    `json:"kind"`
	Title    string    `json:"title"`
	Family   string    `json:"family,omitempty"`
	Field    string    `json:"field,omitempty"`
	KR       string    `json:"kr,omitempty"`
	JP       *string   `json:"jp,omitempty"`
	EN       *string   `json:"en,omitempty"`
	Variants []Variant `json:"variants,omitempty"`
	// JPENClass says whether JP and EN tell the variants apart: JPENSame,
	// JPENExplained or JPENPartial; inconsistent groups only.
	JPENClass string             `json:"jpen_class,omitempty"`
	Evidence  []suggest.Evidence `json:"evidence"`
	Pending   int                `json:"pending"`
	Declined  *Confirmed         `json:"declined,omitempty"`
}

// WarningItem is one check warning: a code about one KR text and its
// translation, with every unit that has the combination.
type WarningItem struct {
	ID       string     `json:"id"`
	Code     string     `json:"code"`
	KR       string     `json:"kr"`
	ZH       string     `json:"zh"`
	Message  string     `json:"message"`
	Keys     []string   `json:"keys"`
	Families []string   `json:"families"`
	Origins  []string   `json:"origins"`
	Pending  int        `json:"pending"`
	Accepted *Confirmed `json:"accepted,omitempty"`
	// Term and Units are set on a term-missing group only: one glossary
	// entry, and every unit that lacks its rendering. KR and ZH are those of
	// the entry, Keys the units not accepted yet.
	Term  *TermRef   `json:"term,omitempty"`
	Units []TermUnit `json:"units,omitempty"`
}

// TermRef is the glossary entry a term-missing group is about.
type TermRef struct {
	KR string `json:"kr"`
	ZH string `json:"zh"`
}

// TermUnit is one unit of a term-missing group. ID is the warning id of its
// KR and TW; Accepted is set when it is in rules/accepted.yaml.
type TermUnit struct {
	Key      string     `json:"key"`
	ID       string     `json:"id"`
	KR       string     `json:"kr"`
	ZH       string     `json:"zh"`
	Note     string     `json:"note,omitempty"`
	Accepted *Confirmed `json:"accepted,omitempty"`
}

// NoteItem is one translator's note on the unit it explains. ID is made of
// the key as written in notes.jsonl; Key is the unit the note is placed on,
// which differs after the record moved to another pointer.
type NoteItem struct {
	ID       string     `json:"id"`
	Key      string     `json:"key"`
	Family   string     `json:"family"`
	Batch    string     `json:"batch"`
	Note     string     `json:"note"`
	KR       string     `json:"kr"`
	ZH       string     `json:"zh"`
	O        string     `json:"o"`
	Pending  int        `json:"pending"`
	Declined *Confirmed `json:"declined,omitempty"`
}

// TodoItem is one line of a to-do section; Key is the unit it is about, if any.
type TodoItem struct {
	Text string `json:"text"`
	Key  string `json:"key,omitempty"`
}

// TodoSection is one kind of case that only adjusting or retranslating solves.
type TodoSection struct {
	Kind  string     `json:"kind"`
	Title string     `json:"title"`
	Hint  string     `json:"hint"`
	Items []TodoItem `json:"items"`
}

// todoKinds are the kinds of build.ReviewItems the to-do page lists; the
// others have a rule suggestion.
var todoKinds = []string{
	build.KindNeedsReview, build.KindFrozenChanged, build.KindFrozenFallback,
	build.KindLoopError, build.KindNotTranslated,
}

// placedUnit is a unit of a file that has a record, kept to place notes.
type placedUnit struct {
	key  string
	unit Unit
	rec  store.Record
}

// Review is every item that needs a person.
type Review struct {
	Suggestions []SuggestionItem
	Warnings    []WarningItem
	Notes       []NoteItem
	Todo        []TodoSection
	// Mismatches are the ids of the groups of inconsistent translations whose
	// units do not add up to the evidence of the suggestion.
	Mismatches    []string
	NotesUnplaced int
	// Refs lists, for each unit key, the ids of the unconfirmed items it is in,
	// sorted.
	Refs map[string][]string
	// NoteText is the notes of each unit (confirmed ones too) joined by "\n".
	NoteText map[string]string
	// Queued holds the keys of the pending list.
	Queued map[string]bool
}

// shownOrigin says whether a unit of this origin counts as a rendering the
// build groups (build.addRendering).
func shownOrigin(o string) bool {
	return o == store.OriginParatranz || o == store.OriginClaude || o == store.OriginHuman
}

type groupKey struct{ family, field, kr string }

// group is the units of one groupKey, grouped by their TW text.
type group struct {
	family, field, kr string
	first             map[string][2]*string // zh -> JP, EN of its first unit
	order             []string              // zh in order of first unit
	keys              map[string][]string
	jpen              map[string][2]*string // unit key -> its JP, EN
}

// newTermHits is the units whose KR a provisional term hits, as evidence.
type newTermHits struct {
	term  rules.Term
	units []suggest.Evidence
}

// usedNote is the note of a unit a provisional term hits: whether its
// translation uses the term.
func usedNote(used bool) string {
	if used {
		return "有用"
	}
	return "未用"
}

type warnAcc struct {
	units    []TermUnit
	item     WarningItem
	families map[string]bool
	origins  map[string]bool
}

// BuildReview computes the rule suggestions and the check warnings over files,
// the file names of the KR pack in the order of the index. in.Review must be set.
func BuildReview(in Inputs, files []string) (Review, error) {
	rv := in.Review
	if rv == nil {
		return Review{}, errors.New("site: BuildReview needs Inputs.Review")
	}
	if rv.Fields == nil {
		return Review{}, errors.New("site: BuildReview needs Review.Fields")
	}
	suggestions, err := suggest.Run(suggest.Inputs{
		RawTag: rv.RawTag, Rules: in.Rules, Adjustments: rv.Adjustments, Build: rv.Build, Plan: rv.Plan,
		Provisional: rv.Provisional,
	})
	if err != nil {
		return Review{}, err
	}

	queued := make(map[string]bool, len(rv.Pending))
	for _, p := range rv.Pending {
		queued[p.Key] = true
	}
	notesOf := map[string][]store.Note{}
	for _, n := range rv.Notes {
		if file, _, err := units.SplitKey(n.Key); err == nil {
			notesOf[file] = append(notesOf[file], n)
		}
	}
	var noteItems []NoteItem
	noteText := map[string]string{}
	noteRefs := map[string][]string{}
	placedCount := 0
	declined := confirmedIndex(rv.Declined)
	accepted := confirmedIndex(rv.Accepted)

	// The provisional terms with a suggestion, and the units their KR hits.
	newTerms := map[string]*newTermHits{}
	for _, s := range suggestions {
		if s.Kind != suggest.KindNewTerm {
			continue
		}
		kr := strings.TrimPrefix(s.ID, s.Kind+"/")
		for _, p := range rv.Provisional {
			if p.KR == kr && !p.Conflict {
				newTerms[s.ID] = &newTermHits{term: rules.Term{KR: kr, ZH: p.ZH, Match: rules.MatchWord}}
				break
			}
		}
	}

	groups := map[string]*group{}
	var warns []*warnAcc
	warnByID := map[string]*warnAcc{}

	for _, path := range files {
		frozen := config.MatchAny(rv.Frozen, path)
		family := in.Rules.Family(path)
		var withRec []placedUnit
		visit := func(u Unit, ku units.Unit, en, jp *align.LangAligned) {
			key := units.Key(path, ku.Pointer)
			// Only the shipped text of a record counts as a rendering, as in
			// the build's addRendering.
			if shownOrigin(u.O) {
				id := itemid.Inconsistent(family, ku.Field, ku.KR)
				g := groups[id]
				if g == nil {
					g = &group{family: family, field: ku.Field, kr: ku.KR, first: map[string][2]*string{}, keys: map[string][]string{}, jpen: map[string][2]*string{}}
					groups[id] = g
				}
				if _, ok := g.keys[u.ZH]; !ok {
					g.order = append(g.order, u.ZH)
					g.first[u.ZH] = [2]*string{u.JP, u.EN}
				}
				g.keys[u.ZH] = append(g.keys[u.ZH], key)
				g.jpen[key] = [2]*string{u.JP, u.EN}
			}
			for _, h := range newTerms {
				if h.term.OccursIn(ku.KR) {
					h.units = append(h.units, suggest.Evidence{Key: key, KR: ku.KR, ZH: u.ZH, Note: usedNote(strings.Contains(u.ZH, h.term.ZH))})
				}
			}
			// The warnings are chosen as the build chooses what to check.
			rec, ok := in.Records[key]
			if ok {
				withRec = append(withRec, placedUnit{key: key, unit: u, rec: rec})
			}
			if !ok || !(frozen || rec.KRSha == units.KRSha(ku.KR)) {
				return
			}
			if !rv.Fields.Classify(ku, en, jp).Translate || rec.Origin == store.OriginCopy {
				return
			}
			seen := map[string]bool{}
			terms := in.Rules.ValidateTerms(path, ku.KR)
			for _, is := range validate.Check(ku.KR, rec.ZH, validate.Context{Existing: true, File: path, ScriptOK: in.Rules.ScriptRenderings(path), Terms: terms}) {
				if is.Severity != validate.SeverityWarn {
					continue
				}
				if is.Code == validate.CodeTermMissing {
					// One item per glossary entry; a unit lacks an entry once.
					termZH := ""
					for _, t := range terms {
						if t.KR == is.Term {
							termZH = t.ZH
						}
					}
					gid := itemid.TermGroup(is.Term, termZH)
					if seen[gid] {
						continue
					}
					seen[gid] = true
					w := warnByID[gid]
					if w == nil {
						w = &warnAcc{
							item:     WarningItem{ID: gid, Code: is.Code, KR: is.Term, ZH: termZH, Message: is.Message, Keys: []string{}, Term: &TermRef{KR: is.Term, ZH: termZH}},
							families: map[string]bool{}, origins: map[string]bool{},
						}
						warnByID[gid] = w
						warns = append(warns, w)
					}
					w.units = append(w.units, TermUnit{Key: key, ID: itemid.Warning(is.Code, ku.KR, rec.ZH), KR: ku.KR, ZH: rec.ZH})
					w.families[family] = true
					w.origins[rec.Origin] = true
					continue
				}
				if seen[is.Code] {
					continue
				}
				seen[is.Code] = true
				id := itemid.Warning(is.Code, ku.KR, rec.ZH)
				w := warnByID[id]
				if w == nil {
					w = &warnAcc{
						item:     WarningItem{ID: id, Code: is.Code, KR: ku.KR, ZH: rec.ZH, Message: is.Message},
						families: map[string]bool{}, origins: map[string]bool{},
					}
					warnByID[id] = w
					warns = append(warns, w)
				}
				w.item.Keys = append(w.item.Keys, key)
				w.families[family] = true
				w.origins[rec.Origin] = true
			}
		}
		if _, _, err := buildFile(in, path, visit); err != nil {
			return Review{}, err
		}
		placed := placeNotes(notesOf[path], withRec)
		for _, p := range placed {
			placedCount++
			id := itemid.Note(p.note.Key, p.note.Note)
			item := NoteItem{
				ID: id, Key: p.at.key, Family: family, Batch: p.note.Batch, Note: p.note.Note,
				KR: p.at.unit.KR, ZH: p.at.unit.ZH, O: p.at.unit.O,
				Declined: declined[id],
			}
			if queued[p.at.key] {
				item.Pending = 1
			}
			if prev, ok := noteText[p.at.key]; ok {
				noteText[p.at.key] = prev + "\n" + p.note.Note
			} else {
				noteText[p.at.key] = p.note.Note
			}
			if item.Declined == nil {
				noteRefs[p.at.key] = append(noteRefs[p.at.key], id)
			}
			noteItems = append(noteItems, item)
		}
	}

	out := Review{
		Suggestions:   make([]SuggestionItem, 0, len(suggestions)),
		Warnings:      make([]WarningItem, 0, len(warns)),
		Notes:         append([]NoteItem{}, noteItems...),
		Todo:          []TodoSection{},
		Mismatches:    []string{},
		Refs:          map[string][]string{},
		NoteText:      noteText,
		Queued:        queued,
		NotesUnplaced: len(rv.Notes) - placedCount,
	}
	for key, ids := range noteRefs {
		out.Refs[key] = append(out.Refs[key], ids...)
	}
	pendingOf := func(keys []string) int {
		n := 0
		for _, k := range keys {
			if queued[k] {
				n++
			}
		}
		return n
	}
	ref := func(id string, keys []string) {
		for _, k := range keys {
			out.Refs[k] = append(out.Refs[k], id)
		}
	}

	onNotesTab := make(map[string]bool, len(noteItems))
	for _, n := range noteItems {
		onNotesTab[n.ID] = true
	}
	for _, s := range suggestions {
		// A note placed on the notes tab is not listed a second time as a suggestion.
		if s.Kind == suggest.KindNote && onNotesTab[s.ID] {
			continue
		}
		item := SuggestionItem{ID: s.ID, Kind: s.Kind, Title: s.Title, Evidence: s.Evidence, Declined: declined[s.ID]}
		if h := newTerms[s.ID]; h != nil {
			item.Evidence = append(slices.Clone(s.Evidence), h.units[:min(len(h.units), suggest.MaxEvidence)]...)
		}
		if s.Kind == suggest.KindInconsistent {
			g := groups[s.ID]
			ok := false
			if g != nil {
				item.Family, item.Field, item.KR = g.family, g.field, g.kr
				item.Variants = variantsOf(g)
				item.JPENClass = jpenClass(item.Variants)
				for _, v := range item.Variants {
					item.Pending += pendingOf(v.Keys)
				}
				ok = evidenceAdds(s.Evidence, item.Variants)
				// The first unit of the first variant gives JP and EN.
				item.JP, item.EN = g.first[item.Variants[0].ZH][0], g.first[item.Variants[0].ZH][1]
			}
			if !ok {
				out.Mismatches = append(out.Mismatches, s.ID)
			}
			if item.Declined == nil {
				for _, v := range item.Variants {
					ref(s.ID, v.Keys)
				}
			}
		}
		out.Suggestions = append(out.Suggestions, item)
	}

	slices.SortStableFunc(warns, func(a, b *warnAcc) int { return cmp.Compare(a.item.Code, b.item.Code) })
	for _, w := range warns {
		w.item.Families = sortedKeys(w.families)
		w.item.Origins = sortedKeys(w.origins)
		if w.item.Term != nil {
			for _, u := range w.units {
				u.Accepted = accepted[u.ID]
				u.Note = noteText[u.Key]
				w.item.Units = append(w.item.Units, u)
				if u.Accepted == nil {
					w.item.Keys = append(w.item.Keys, u.Key)
				}
			}
		} else {
			w.item.Accepted = accepted[w.item.ID]
		}
		if w.item.Term != nil && len(w.item.Units) > 0 && len(w.item.Keys) == 0 {
			// Every unit is accepted: the group reads as confirmed, with the
			// confirmation of its last unit. A group with an unaccepted unit
			// stays open and its Keys are those units.
			w.item.Accepted = w.item.Units[len(w.item.Units)-1].Accepted
		}
		w.item.Pending = pendingOf(w.item.Keys)
		if w.item.Accepted == nil {
			ref(w.item.ID, w.item.Keys)
		}
		out.Warnings = append(out.Warnings, w.item)
	}
	if rv.Build != nil {
		for _, sec := range build.ReviewItems(*rv.Build, rv.Plan, rv.Pending) {
			if !slices.Contains(todoKinds, sec.Kind) {
				continue
			}
			ts := TodoSection{Kind: sec.Kind, Title: sec.Title, Hint: sec.Hint, Items: make([]TodoItem, 0, len(sec.Items))}
			for i, text := range sec.Items {
				it := TodoItem{Text: text}
				if i < len(sec.Keys) {
					it.Key = sec.Keys[i]
				}
				ts.Items = append(ts.Items, it)
			}
			out.Todo = append(out.Todo, ts)
		}
	}
	for k, ids := range out.Refs {
		slices.Sort(ids)
		out.Refs[k] = slices.Compact(ids)
	}
	return out, nil
}

// confirmedIndex indexes the entries of a confirmation list by id.
func confirmedIndex(list []suggest.Declined) map[string]*Confirmed {
	m := make(map[string]*Confirmed, len(list))
	for _, d := range list {
		m[d.ID] = &Confirmed{Reason: d.Reason, Date: d.Date}
	}
	return m
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// variantsOf lists the translations of g, the one with the most units first;
// equal counts keep the order of the first key (files, then units).
func variantsOf(g *group) []Variant {
	out := make([]Variant, 0, len(g.order))
	for _, zh := range g.order {
		out = append(out, Variant{ZH: zh, Count: len(g.keys[zh]), Keys: g.keys[zh], JPEN: jpenGroups(g, g.keys[zh])})
	}
	// g.order is already the order of the first unit.
	slices.SortStableFunc(out, func(a, b Variant) int { return cmp.Compare(b.Count, a.Count) })
	return out
}

// jpenGroups groups keys, the units of one variant of g, by their JP and EN,
// the most units first; equal counts keep the order of the first key.
func jpenGroups(g *group, keys []string) []JPENGroup {
	var out []JPENGroup
	at := map[string]int{}
	for _, key := range keys {
		je := g.jpen[key]
		p := pairKey(je)
		i, ok := at[p]
		if !ok {
			i = len(out)
			at[p] = i
			out = append(out, JPENGroup{JP: je[0], EN: je[1]})
		}
		out[i].Count++
		out[i].Keys = append(out[i].Keys, key)
	}
	slices.SortStableFunc(out, func(a, b JPENGroup) int { return cmp.Compare(b.Count, a.Count) })
	return out
}

// pairKey is a JP/EN pair as a map key; a missing language is a value of its
// own, apart from an empty string.
func pairKey(je [2]*string) string {
	part := func(p *string) string {
		if p == nil {
			return "-"
		}
		return "+" + *p
	}
	return part(je[0]) + "\x00" + part(je[1])
}

// jpenClass says whether JP and EN tell variants apart (spec of the JP/EN
// view): same when every unit has one JP/EN pair, explained when each pair
// has one translation, partial otherwise.
func jpenClass(variants []Variant) string {
	trans := map[string]map[string]bool{} // JP/EN pair -> its translations
	for _, v := range variants {
		for _, j := range v.JPEN {
			p := pairKey([2]*string{j.JP, j.EN})
			if trans[p] == nil {
				trans[p] = map[string]bool{}
			}
			trans[p][v.ZH] = true
		}
	}
	if len(trans) <= 1 {
		return JPENSame
	}
	for _, zh := range trans {
		if len(zh) > 1 {
			return JPENPartial
		}
	}
	return JPENExplained
}

// evidenceAdds reports whether every counted row of the evidence has a variant
// with the same translation and the same number of units. The build keeps only
// some variants, so a variant the evidence lacks is not a mismatch.
func evidenceAdds(ev []suggest.Evidence, vs []Variant) bool {
	for _, e := range ev {
		if e.Count == 0 {
			continue
		}
		i := slices.IndexFunc(vs, func(v Variant) bool { return v.ZH == e.ZH })
		if i < 0 || vs[i].Count != e.Count {
			return false
		}
	}
	return true
}

type placedNote struct {
	note store.Note
	at   placedUnit
}

// placeNotes puts the notes of one file on its units (spec 5.3): the unit of
// the note's key when its record has the note's batch and kr_sha, otherwise
// the one record of the file that has both; a note with no place is left out.
// The result is in unit order, then in batch order.
func placeNotes(notes []store.Note, us []placedUnit) []placedNote {
	if len(notes) == 0 {
		return nil
	}
	type slot struct {
		placedNote
		idx int
	}
	var out []slot
	for _, n := range notes {
		at := -1
		for i, u := range us {
			if u.key == n.Key && u.rec.Batch == n.Batch && u.rec.KRSha == n.KRSha {
				at = i
				break
			}
		}
		if at < 0 {
			for i, u := range us {
				if u.rec.Batch == n.Batch && u.rec.KRSha == n.KRSha {
					if at >= 0 {
						at = -1
						break
					}
					at = i
				}
			}
		}
		if at >= 0 {
			out = append(out, slot{placedNote{n, us[at]}, at})
		}
	}
	slices.SortStableFunc(out, func(a, b slot) int {
		return cmp.Or(cmp.Compare(a.idx, b.idx), cmp.Compare(a.note.Batch, b.note.Batch))
	})
	res := make([]placedNote, len(out))
	for i, s := range out {
		res[i] = s.placedNote
	}
	return res
}
