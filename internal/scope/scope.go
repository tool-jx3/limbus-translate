// Package scope measures where the translations use each approved entry of the
// glossary, and proposes where the entry should apply (spec of the glossary in
// context, section 6): for each file family and each directory group, how many
// translated units the entry hits and how many of them use its rendering; how
// many units hold it inside a longer word; how many of the units that lack the
// rendering have a note that names the term. lct site writes the result as
// review/scope.json, and lct glossary scope prints it.
package scope

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/tool-jx3/limbus-translate/internal/batch"
	"github.com/tool-jx3/limbus-translate/internal/itemid"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/suggest"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// The kinds of proposal, in the order of Kinds (itemid.Scope).
const (
	KindNotFiles  = itemid.ScopeNotFiles
	KindFiles     = itemid.ScopeFiles
	KindSubstring = itemid.ScopeSubstring
	KindWord      = itemid.ScopeWord
	KindMixed     = itemid.ScopeMixed
	KindExplained = itemid.ScopeExplained
)

// Kinds lists the kinds of proposal in the order Analyze lists them.
var Kinds = itemid.ScopeKinds

// The kinds of group.
const (
	GroupFamily = "family"
	GroupDir    = "dir"
)

// Dirs are the directory groups, as the patterns that name them; a file in
// none is in RestGroup.
var Dirs = []string{"StoryData/**", "PersonalityVoiceDlg/**", "EGOVoiceDig/**", "BattleAnnouncerDlg/**", "RPGSystem/**"}

// RestGroup is the name of the directory group of the files in none of Dirs.
// No pattern names it, so no proposal does.
const RestGroup = "rest"

// Unit is one translated unit: Origin is its record's, ZH its record's text,
// Note the notes on it joined by "\n".
type Unit struct {
	Key, File, KR, ZH, Origin, Note string
}

// Inputs is what Analyze looks at. Declined are the confirmations of
// rules/declined.yaml by id.
type Inputs struct {
	Rules    *rules.Rules
	Units    []Unit
	Declined map[string]*suggest.Confirmed
	// Provisional are the provisional terms in force
	// (batch.ActiveProvisional), which cover glossary entries as on the
	// review page (batch.TermsIn).
	Provisional []rules.Term
}

// Thresholds are the limits the proposals are made by.
type Thresholds struct {
	// MinN is the fewest units a group, or the compound units, need to say
	// anything, and the fewest units without the rendering that make a group
	// mixed.
	MinN int `json:"min_n"`
	// Used is the least share of units that use the rendering for a group
	// to use it; NotUsed the most for a group not to use it. A group in
	// between is mixed when MinN of its units or more lack the rendering;
	// with fewer it says nothing.
	Used    float64 `json:"used"`
	NotUsed float64 `json:"not_used"`
	// Explained is the fewest explained units that make an explained proposal.
	Explained int `json:"explained"`
	// LostUnits and LostShare flag a proposal that would leave so many units
	// that use the rendering unchecked, or so large a share of them.
	LostUnits int     `json:"lost_units"`
	LostShare float64 `json:"lost_share"`
}

// Report is the result of Analyze.
type Report struct {
	Thresholds Thresholds `json:"thresholds"`
	Entries    []Entry    `json:"entries"`
	Proposals  []Proposal `json:"proposals"`
}

// Entry is one approved glossary entry and its counts.
type Entry struct {
	ID       string   `json:"id"`
	KR       string   `json:"kr"`
	ZH       string   `json:"zh"`
	Also     []string `json:"also"`
	Files    []string `json:"files"`
	NotFiles []string `json:"not_files"`
	Match    string   `json:"match"`
	Notes    string   `json:"notes"`
	Groups   []Group  `json:"groups"`
	Compound Count    `json:"compound"`
	// Explained and Unexplained are the units the entry hits that do not use
	// its rendering, with a note that names the term and without.
	Explained   int `json:"explained"`
	Unexplained int `json:"unexplained"`
}

// Group is the units an entry hits in one family or one directory group.
type Group struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	N    int    `json:"n"`
	Used int    `json:"used"`
}

// Count is a number of units and how many of them use the rendering.
type Count struct {
	N    int `json:"n"`
	Used int `json:"used"`
}

// Proposal is one change of an entry, or one entry a person should look at.
//
// MixedGroups are the groups of the entry that use the rendering in part, by
// name (Group.Name), on a proposal of where the entry applies (not-files,
// files, mixed); empty on the others, and on a mixed proposal of an entry
// that no group uses. A change of files or not_files is proposed beside them
// when the other kind of group splits cleanly: the person sees what it leaves
// as it is.
//
// The effect is the review page before and after the change, over every
// entry (shorter entries that a changed entry no longer covers, the entry
// without files that takes a KR text an entry with files gives up):
// WarnRemoved and WarnAdded are the term-missing warnings that go and come,
// UsedLost and UsedGained the hits that use their rendering that go (units the
// check no longer covers) and come. A mixed or explained proposal changes
// nothing and has none.
type Proposal struct {
	ID          string             `json:"id"`
	Kind        string             `json:"kind"`
	KR          string             `json:"kr"`
	ZH          string             `json:"zh"`
	Patterns    []string           `json:"patterns"`
	Match       string             `json:"match"`
	MixedGroups []string           `json:"mixed_groups"`
	WarnRemoved int                `json:"warn_removed"`
	WarnAdded   int                `json:"warn_added"`
	UsedLost    int                `json:"used_lost"`
	UsedGained  int                `json:"used_gained"`
	Flagged     bool               `json:"flagged"`
	Examples    []Example          `json:"examples"`
	Declined    *suggest.Confirmed `json:"declined,omitempty"`
}

// Example is one unit behind a proposal, as short pieces of its texts.
type Example struct {
	Key   string `json:"key"`
	Group string `json:"group"`
	KR    string `json:"kr"`
	ZH    string `json:"zh"`
}

// The thresholds (Thresholds, which the report carries for the page to show).
const (
	minN         = 3
	usedShare    = 0.8
	notUsedShare = 0.3
	minExplained = 3
	lostUnits    = 20
	lostShare    = 0.1
)

// maxExamples is the most examples of a group a proposal lists, and of the
// units behind a proposal without groups; maxSnippet the most characters of
// each text of an example.
const (
	maxExamples = 3
	maxSnippet  = 60
)

func thresholds() Thresholds {
	return Thresholds{MinN: minN, Used: usedShare, NotUsed: notUsedShare, Explained: minExplained, LostUnits: lostUnits, LostShare: lostShare}
}

// Empty is the report of no entries.
func Empty() Report {
	return Report{Thresholds: thresholds(), Entries: []Entry{}, Proposals: []Proposal{}}
}

// counted reports whether a unit of origin o is a translation of its own,
// which says whether the rendering is used: not a copy, a fallback or a unit
// without a record.
func counted(o string) bool {
	return o == store.OriginClaude || o == store.OriginHuman || o == store.OriginParatranz
}

// Analyze measures every approved entry of in.Rules over in.Units and makes the
// proposals; Entries are in glossary order, Proposals by kind (Kinds) and
// then in glossary order. Only the units of a counted origin (Claude's, a
// person's, Paratranz's) count.
//
// Whether an entry hits a unit is what the review page says: the glossary
// terms of batch.TermsIn with in.Provisional, that is Rules.GlossaryHits
// (scopes, and the longer entry that covers a shorter one), the provisional
// terms in force covering entries too. A unit's groups are its
// family (Rules.Family, as family:<id>) and its directory group (Dirs, or
// RestGroup). A unit is a compound unit of an entry when its hit depends on
// the entry's match: of a word entry, a unit the entry would hit as a
// substring entry (its KR holds the entry inside a longer word); of a
// substring entry, a unit it hits that it would not hit as a word entry. The groups count the hits that are no compound units, so that
// the two questions, where and how the entry matches, stay apart.
//
// The proposals list only what differs from the entry as it is, and each
// says what it does to the review page as it is (see Proposal). Flagged marks
// a proposal whose UsedLost is LostUnits or more, or a LostShare of the hits
// of the entry that use its rendering. A proposal whose id is in in.Declined
// carries its confirmation.
//
// Of the two kinds of group, each may propose a change of files or not_files
// (see scopeOption); of two, the one with fewer patterns is proposed, then
// the one that loses fewer checks, then the one that removes more warnings,
// then the families'. Without one, a mixed proposal is made when a group uses
// the rendering in part, or no group of a kind uses it.
func Analyze(in Inputs) Report {
	if in.Rules == nil {
		return Empty()
	}
	an := newAnalysis(in)
	for i, u := range in.Units {
		if counted(u.Origin) {
			an.unit(int32(i), u)
		}
	}
	rep := Empty()
	byKind := map[string][]Proposal{}
	for _, a := range an.accs {
		rep.Entries = append(rep.Entries, an.entry(a))
		for _, p := range an.proposals(a) {
			p.Declined = in.Declined[p.ID]
			byKind[p.Kind] = append(byKind[p.Kind], p)
		}
	}
	for _, k := range Kinds {
		rep.Proposals = append(rep.Proposals, byKind[k]...)
	}
	return rep
}

// hit is one unit an entry hits, or (in acc.extra) one compound unit of a word
// entry, which it does not hit.
type hit struct {
	unit      int32 // in Inputs.Units
	used      bool  // the translation uses the rendering
	compound  bool
	explained bool // not used, and the note names the term
}

// acc is what Analyze gathers of one entry.
type acc struct {
	term rules.Term
	// idx is the place of the entry in analysis.accs; contains are the units
	// whose KR text holds its KR, which a change of the entry may change.
	idx      int
	contains []int32
	groups   map[string]*Group
	// hits are the units the entry hits; extra the compound units of a word
	// entry.
	hits, extra            []hit
	compound               Count
	explained, unexplained int
	usedHits               int
}

// place is the groups of a file.
type place struct{ family, dir string }

type analysis struct {
	r           *rules.Rules
	provisional []rules.Term
	units       []Unit
	accs        []*acc
	// sameKR holds the KR texts of more than one entry: limiting one of them
	// to files would make it stand for the others there.
	sameKR map[string]bool
	places map[string]place
	// byRune lists the entries by the first character of their KR, in
	// glossary order; seen and found are the scratch of contained.
	byRune map[rune][]int
	seen   []bool
	found  []int
	// cands and isHit are the scratch of unit, alt and altHit of hitsAs.
	cands, alt    []rules.Term
	isHit, altHit []bool
}

func newAnalysis(in Inputs) *analysis {
	an := &analysis{r: in.Rules, provisional: in.Provisional, units: in.Units, sameKR: map[string]bool{}, places: map[string]place{}, byRune: map[rune][]int{}}
	n := map[string]int{}
	for _, t := range in.Rules.Glossary {
		if t.Status != rules.StatusApproved || t.KR == "" {
			continue
		}
		first, _ := utf8.DecodeRuneInString(t.KR)
		an.byRune[first] = append(an.byRune[first], len(an.accs))
		an.accs = append(an.accs, &acc{term: t, idx: len(an.accs), groups: map[string]*Group{}})
		if n[t.KR]++; n[t.KR] > 1 {
			an.sameKR[t.KR] = true
		}
	}
	an.seen = make([]bool, len(an.accs))
	return an
}

// contained returns the entries whose KR occurs in plain, in glossary order.
// An entry hits only where its KR occurs, so they are all the entries that may.
func (an *analysis) contained(plain string) []int {
	an.found = an.found[:0]
	for at, c := range plain {
		for _, i := range an.byRune[c] {
			if !an.seen[i] && strings.HasPrefix(plain[at:], an.accs[i].term.KR) {
				an.seen[i] = true
				an.found = append(an.found, i)
			}
		}
	}
	for _, i := range an.found {
		an.seen[i] = false
	}
	slices.Sort(an.found)
	return an.found
}

// unit counts the unit u, number i of the inputs.
func (an *analysis) unit(i int32, u Unit) {
	idx := an.setCands(u)
	if len(idx) == 0 {
		return
	}
	an.isHit = an.hitsIn(u, -1, rules.Term{}, an.isHit)
	isHit := an.isHit
	pl := an.placeOf(u.File)
	for c, j := range idx {
		a := an.accs[j]
		a.contains = append(a.contains, i)
		t := a.term
		switch {
		case isHit[c]:
			h := hit{unit: i, used: t.Used(u.ZH)}
			h.compound = t.Match == rules.MatchSubstring && !an.hitsAs(u, c, withMatch(t, rules.MatchWord))
			a.add(h, pl, u.Note)
		case t.Match != rules.MatchSubstring && an.hitsAs(u, c, withMatch(t, rules.MatchSubstring)):
			h := hit{unit: i, used: t.Used(u.ZH), compound: true}
			a.compound.N++
			if h.used {
				a.compound.Used++
			}
			a.extra = append(a.extra, h)
		}
	}
}

// setCands sets an.cands to the entries whose KR the text of u holds and
// returns their places in an.accs, in glossary order.
//
// Only they are asked about: of the others none occurs, so none would be a
// hit, and none would take the place of another (an entry with files stands
// for the entries of its KR text, and a covering entry occurs where it
// covers). The answer is the same as of the whole glossary, for a fraction of
// the work.
func (an *analysis) setCands(u Unit) []int {
	idx := an.contained(validate.MatchText(u.KR))
	an.cands = an.cands[:0]
	for _, j := range idx {
		an.cands = append(an.cands, an.accs[j].term)
	}
	return idx
}

// hitsIn sets into[k] to whether entry k of an.cands hits u, entry c taken as
// alt when c is 0 or more, and returns into. The hits are those of the review
// page: the glossary terms of batch.TermsIn over the entries and the
// provisional terms in force.
func (an *analysis) hitsIn(u Unit, c int, alt rules.Term, into []bool) []bool {
	terms := an.cands
	if c >= 0 {
		an.alt = append(an.alt[:0], an.cands...)
		an.alt[c] = alt
		terms = an.alt
	}
	sub := *an.r
	sub.Glossary = terms
	hits := slices.DeleteFunc(batch.TermsIn(&sub, an.provisional, u.File, u.KR), func(t rules.Term) bool { return t.Provisional })
	return marks(into, terms, hits)
}

// hitsAs reports whether entry c of an.cands would hit u if it were alt.
func (an *analysis) hitsAs(u Unit, c int, alt rules.Term) bool {
	an.altHit = an.hitsIn(u, c, alt, an.altHit)
	return an.altHit[c]
}

// effect is what changing an entry does to the review page.
type effect struct {
	removed, added, lost, gained int
	flagged                      bool
}

// effectOf returns what the entry of a does to the review page as alt: over
// the units that hold its KR, the hits of every entry before and after.
func (an *analysis) effectOf(a *acc, alt rules.Term) effect {
	var e effect
	before := []bool{}
	for _, i := range a.contains {
		u := an.units[i]
		idx := an.setCands(u)
		c := slices.Index(idx, a.idx)
		before = an.hitsIn(u, -1, rules.Term{}, before)
		after := an.hitsIn(u, c, alt, an.altHit)
		an.altHit = after
		for k, j := range idx {
			if before[k] == after[k] {
				continue
			}
			used := an.accs[j].term.Used(u.ZH) // alt has the renderings of the entry
			switch {
			case before[k] && used:
				e.lost++
			case before[k]:
				e.removed++
			case used:
				e.gained++
			default:
				e.added++
			}
		}
	}
	e.flagged = a.flag(e.lost)
	return e
}

// marks sets into[i] to whether terms[i] is among hits, which GlossaryHits
// (or batch.TermsIn, without its provisional terms) returned for terms: a
// subsequence of them, in their order.
func marks(into []bool, terms, hits []rules.Term) []bool {
	into = slices.Grow(into[:0], len(terms))[:len(terms)]
	j := 0
	for i, t := range terms {
		into[i] = j < len(hits) && sameEntry(t, hits[j])
		if into[i] {
			j++
		}
	}
	return into
}

// sameEntry reports whether a and b are alike in all that decides a hit.
func sameEntry(a, b rules.Term) bool {
	return a.KR == b.KR && a.ZH == b.ZH && a.Match == b.Match && slices.Equal(a.Files, b.Files) && slices.Equal(a.NotFiles, b.NotFiles)
}

func withMatch(t rules.Term, match string) rules.Term {
	t.Match = match
	return t
}

// add counts the hit h in the groups pl, and whether the note explains it.
func (a *acc) add(h hit, pl place, note string) {
	switch {
	case h.used:
		a.usedHits++
	case a.term.OccursIn(note):
		h.explained = true
		a.explained++
	default:
		a.unexplained++
	}
	if h.compound {
		a.compound.N++
		if h.used {
			a.compound.Used++
		}
	} else {
		for _, g := range []struct{ name, kind string }{{pl.family, GroupFamily}, {pl.dir, GroupDir}} {
			if g.name == "" {
				continue
			}
			gr := a.groups[g.name]
			if gr == nil {
				gr = &Group{Name: g.name, Kind: g.kind}
				a.groups[g.name] = gr
			}
			gr.N++
			if h.used {
				gr.Used++
			}
		}
	}
	a.hits = append(a.hits, h)
}

// placeOf returns the groups of file. A group is named by the item of files or
// not_files that takes it (Rules.Applies takes the file by it), so that a
// proposal is made of group names.
func (an *analysis) placeOf(file string) place {
	if pl, ok := an.places[file]; ok {
		return pl
	}
	pl := place{dir: RestGroup}
	if f := an.r.Family(file); f != "" {
		pl.family = rules.FamilyPrefix + f
	}
	for _, d := range Dirs {
		if an.r.Applies(rules.Term{Files: []string{d}}, file) {
			pl.dir = d
			break
		}
	}
	an.places[file] = pl
	return pl
}

// orderedGroups returns the groups of a: the families in the order of the
// routing, then the directories in the order of Dirs, then RestGroup.
func (an *analysis) orderedGroups(a *acc) []Group {
	out := []Group{}
	var names []string
	for _, f := range an.r.Routing.Families {
		names = append(names, rules.FamilyPrefix+f.ID)
	}
	names = append(append(names, Dirs...), RestGroup)
	for _, n := range names {
		if g := a.groups[n]; g != nil {
			out = append(out, *g)
		}
	}
	return out
}

func (an *analysis) entry(a *acc) Entry {
	t := a.term
	return Entry{
		ID: itemid.GlossaryEntry(t.KR, t.ZH), KR: t.KR, ZH: t.ZH,
		Also: nonNil(t.Also), Files: nonNil(t.Files), NotFiles: nonNil(t.NotFiles),
		Match: matchOf(t), Notes: t.Notes, Groups: an.orderedGroups(a),
		Compound: a.compound, Explained: a.explained, Unexplained: a.unexplained,
	}
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return slices.Clone(s)
}

// matchOf is how t matches: rules.MatchSubstring or rules.MatchWord.
func matchOf(t rules.Term) string {
	if t.Match == rules.MatchSubstring {
		return rules.MatchSubstring
	}
	return rules.MatchWord
}

// The classes of a group.
const (
	classFew     = iota // says nothing: fewer than minN units, or fewer than minN without the rendering in a group in between
	classUsed           // usedShare or more use the rendering
	classNotUsed        // notUsedShare or fewer
	classMixed          // in between, and minN units or more lack the rendering
)

func classOf(n, used int) int {
	switch share := float64(used) / float64(n); {
	case n < minN:
		return classFew
	case share >= usedShare:
		return classUsed
	case share <= notUsedShare:
		return classNotUsed
	case n-used < minN:
		return classFew
	}
	return classMixed
}

// mixedGroups returns the names of the groups of a that use the rendering in
// part, in group order.
func (an *analysis) mixedGroups(a *acc) []string {
	out := []string{}
	for _, g := range an.orderedGroups(a) {
		if classOf(g.N, g.Used) == classMixed {
			out = append(out, g.Name)
		}
	}
	return out
}

// option is a change of files or not_files that one kind of group proposes.
type option struct {
	kind     string
	patterns []string
	// left are the groups whose units the change leaves out.
	left []string
	effect
}

// proposals returns the proposals about a.
func (an *analysis) proposals(a *acc) []Proposal {
	var out []Proposal
	t := a.term
	newProposal := func(kind string) Proposal {
		return Proposal{ID: itemid.Scope(t.KR, t.ZH, kind), Kind: kind, KR: t.KR, ZH: t.ZH, Patterns: []string{}, MixedGroups: []string{}, Examples: []Example{}}
	}

	withEffect := func(p Proposal, e effect) Proposal {
		p.WarnRemoved, p.WarnAdded, p.UsedLost, p.UsedGained, p.Flagged = e.removed, e.added, e.lost, e.gained, e.flagged
		return p
	}

	// Where: by the families, and by the directories.
	var opts []option
	var mixedOrUnused []string
	for _, kind := range []string{GroupFamily, GroupDir} {
		o, groups := an.scopeOption(a, kind)
		if o != nil {
			alt := t
			switch o.kind {
			case KindNotFiles:
				alt.NotFiles = append(slices.Clone(t.NotFiles), o.patterns...)
			case KindFiles:
				alt.Files = o.patterns
			}
			o.effect = an.effectOf(a, alt)
			opts = append(opts, *o)
		}
		mixedOrUnused = append(mixedOrUnused, groups...)
	}
	// The shorter change, then the one that loses fewer checks, then the one
	// that removes more warnings; the families first.
	slices.SortStableFunc(opts, func(x, y option) int {
		return cmp.Or(cmp.Compare(len(x.patterns), len(y.patterns)), cmp.Compare(x.lost, y.lost), cmp.Compare(y.removed, x.removed))
	})
	switch {
	case len(opts) > 0:
		o := opts[0]
		p := withEffect(newProposal(o.kind), o.effect)
		p.Patterns, p.MixedGroups = o.patterns, an.mixedGroups(a)
		p.Examples = an.groupExamples(a, o.left)
		out = append(out, p)
	case len(mixedOrUnused) > 0:
		p := newProposal(KindMixed)
		p.MixedGroups = an.mixedGroups(a)
		p.Examples = an.groupExamples(a, mixedOrUnused)
		out = append(out, p)
	}

	// How: as a word or inside words too.
	c := a.compound
	switch {
	case t.Match != rules.MatchSubstring && c.N >= minN && float64(c.Used) >= usedShare*float64(c.N):
		p := withEffect(newProposal(KindSubstring), an.effectOf(a, withMatch(t, rules.MatchSubstring)))
		p.Match = rules.MatchSubstring
		p.Examples = an.unitExamples(a, a.extra)
		out = append(out, p)
	case t.Match == rules.MatchSubstring && c.N >= minN && float64(c.Used) <= notUsedShare*float64(c.N):
		p := withEffect(newProposal(KindWord), an.effectOf(a, withMatch(t, rules.MatchWord)))
		p.Match = rules.MatchWord
		p.Examples = an.unitExamples(a, slices.DeleteFunc(slices.Clone(a.hits), func(h hit) bool { return !h.compound }))
		out = append(out, p)
	}

	if a.explained >= minExplained {
		p := newProposal(KindExplained)
		p.Examples = an.unitExamples(a, slices.DeleteFunc(slices.Clone(a.hits), func(h hit) bool { return !h.explained }))
		out = append(out, p)
	}
	return out
}

// compareBool orders false before true.
func compareBool(x, y bool) int {
	switch {
	case x == y:
		return 0
	case !x:
		return -1
	}
	return 1
}

// scopeOption returns the change of files or not_files that the groups of
// kind propose, if any, and else the groups that make the entry mixed, if
// any: groups that use the rendering in part, or the groups that do not use
// it when none of kind does.
//
// Of groups (MinN units or more) that do not use the rendering beside groups
// that use it, and none in part: not_files gets the first, or files is
// limited to the second, when they are fewer. A group not_files takes out
// already has no hits, so not_files never gets a pattern it has. RestGroup
// names no files: a change that needs it is not made. files is not proposed
// for an entry that has files, or whose KR text has another entry, which the
// change would leave out of those files.
func (an *analysis) scopeOption(a *acc, kind string) (*option, []string) {
	var notUsed, used, mixed []string
	for _, g := range an.orderedGroups(a) {
		if g.Kind != kind {
			continue
		}
		switch classOf(g.N, g.Used) {
		case classUsed:
			used = append(used, g.Name)
		case classNotUsed:
			notUsed = append(notUsed, g.Name)
		case classMixed:
			mixed = append(mixed, g.Name)
		}
	}
	switch {
	case len(mixed) > 0:
		return nil, append(mixed, notUsed...)
	case len(notUsed) == 0:
		return nil, nil
	case len(used) == 0:
		return nil, notUsed
	}
	var nf, fl *option
	if !slices.Contains(notUsed, RestGroup) {
		nf = &option{kind: KindNotFiles, patterns: notUsed, left: notUsed}
	}
	if !slices.Contains(used, RestGroup) && len(a.term.Files) == 0 && !an.sameKR[a.term.KR] {
		fl = &option{kind: KindFiles, patterns: used, left: notUsed}
	}
	if fl != nil && (nf == nil || len(fl.patterns) < len(nf.patterns)) {
		return fl, nil
	}
	return nf, nil
}

// flag reports whether losing the check of lost units that use the rendering
// is too much: lostUnits or more, or lostShare of the hits that use it.
func (a *acc) flag(lost int) bool {
	return lost >= lostUnits || lost > 0 && float64(lost) >= lostShare*float64(a.usedHits)
}

// groupExamples returns at most maxExamples units of each of groups that the
// entry hits and that do not use the rendering, no unit twice. Compound units
// are in no group.
func (an *analysis) groupExamples(a *acc, groups []string) []Example {
	out := []Example{}
	listed := map[int32]bool{}
	for _, g := range groups {
		n := 0
		for _, h := range a.hits {
			if n == maxExamples {
				break
			}
			if h.used || h.compound || listed[h.unit] {
				continue
			}
			if pl := an.placeOf(an.units[h.unit].File); pl.family != g && pl.dir != g {
				continue
			}
			listed[h.unit] = true
			n++
			out = append(out, an.example(a.term, h.unit, g))
		}
	}
	return out
}

// unitExamples returns at most maxExamples of hs, those that do not use the
// rendering first; the group of each is its family.
func (an *analysis) unitExamples(a *acc, hs []hit) []Example {
	slices.SortStableFunc(hs, func(x, y hit) int { return compareBool(x.used, y.used) })
	out := []Example{}
	for _, h := range hs[:min(len(hs), maxExamples)] {
		out = append(out, an.example(a.term, h.unit, an.placeOf(an.units[h.unit].File).family))
	}
	return out
}

// example is unit i as an example of t in group: the piece of its KR around
// the first place t occurs, and the piece of its TW at about the same place.
func (an *analysis) example(t rules.Term, i int32, group string) Example {
	u := an.units[i]
	kr, zh := []rune(validate.MatchText(u.KR)), []rune(validate.MatchText(u.ZH))
	plain := string(kr)
	at, size := 0, 0
	if spans := t.Spans(plain); len(spans) > 0 {
		at, size = spans[0][0], spans[0][1]-spans[0][0]
	} else if j := strings.Index(plain, t.KR); j >= 0 {
		at, size = j, len(t.KR)
	}
	start, n := utf8.RuneCountInString(plain[:at]), utf8.RuneCountInString(plain[at:at+size])
	mid := 0
	if len(kr) > 0 {
		mid = (start + n/2) * len(zh) / len(kr)
	}
	return Example{Key: u.Key, Group: group, KR: snippet(kr, start, n), ZH: snippet(zh, mid, 0)}
}

// snippet returns at most maxSnippet characters of text around the n
// characters from start, with … where it is cut.
func snippet(text []rune, start, n int) string {
	if len(text) <= maxSnippet {
		return string(text)
	}
	room := maxSnippet - 2 // the two …
	from := max(0, min(start-(room-n)/2, len(text)-room))
	to := from + room
	out := string(text[from:to])
	if from > 0 {
		out = "…" + out
	}
	if to < len(text) {
		out += "…"
	}
	return out
}

// Title says in one line what the proposal changes and what that does.
func (p Proposal) Title() string {
	entry := "詞彙範圍：" + p.KR + " → " + p.ZH
	effect := fmt.Sprintf("（移除 %d 個警告、新增 %d 個；%d 個用了譯名的單位不再檢查、%d 個開始檢查", p.WarnRemoved, p.WarnAdded, p.UsedLost, p.UsedGained)
	if p.Flagged {
		effect += "；失去檢查偏多"
	}
	effect += "）"
	mixed := ""
	if len(p.MixedGroups) > 0 {
		mixed = "；" + strings.Join(p.MixedGroups, "、") + " 只部分用了譯名"
	}
	pats := strings.Join(p.Patterns, "、")
	switch p.Kind {
	case KindNotFiles:
		return entry + " 的 not_files 加上 " + pats + effect + mixed
	case KindFiles:
		return entry + " 的 files 限定為 " + pats + effect + mixed
	case KindSubstring:
		return entry + " 改為 match: substring" + effect
	case KindWord:
		return entry + " 改為 match: word" + effect
	case KindMixed:
		if len(p.MixedGroups) == 0 {
			return entry + " 各群組都很少用這個譯名，需要人看"
		}
		return entry + " 有群組只部分用了譯名（" + strings.Join(p.MixedGroups, "、") + "），需要人看"
	case KindExplained:
		return entry + " 常有 note 說明不用譯名，考慮 also、縮小範圍或改譯名"
	}
	return entry
}
