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
}

// Thresholds are the limits the proposals are made by.
type Thresholds struct {
	// MinN is the fewest units a group, or the compound units, need to say
	// anything.
	MinN int `json:"min_n"`
	// Used is the least share of units that use the rendering for a group
	// to use it; NotUsed the most for a group not to use it. A group in
	// between is mixed.
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
type Proposal struct {
	ID          string             `json:"id"`
	Kind        string             `json:"kind"`
	KR          string             `json:"kr"`
	ZH          string             `json:"zh"`
	Patterns    []string           `json:"patterns"`
	Match       string             `json:"match"`
	WarnRemoved int                `json:"warn_removed"`
	UsedLost    int                `json:"used_lost"`
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
// Whether an entry hits a unit is what Rules.GlossaryHits says, scopes and the
// longer entry that covers a shorter one included. A unit's groups are its
// family (Rules.Family, as family:<id>) and its directory group (Dirs, or
// RestGroup). A unit is a compound unit of an entry when its hit depends on
// the entry's match: of a word entry, a unit that GlossaryHits would give the
// entry as a substring entry (its KR holds the entry inside a longer word); of
// a substring entry, a unit it hits that GlossaryHits would not give it as a
// word entry. The groups count the hits that are no compound units, so that
// the two questions, where and how the entry matches, stay apart.
//
// The proposals list only what differs from the entry as it is, and each
// says what it does to the entry as it is: WarnRemoved, the hits that lack
// the rendering and would be hits no more (the term-missing warnings of the
// review page that go), and UsedLost, the hits that use it and would be hits
// no more (units the check no longer covers). A proposal that adds hits
// (substring) removes and loses nothing; the entry's Compound says what it
// adds. Flagged marks a proposal that loses LostUnits units or more, or a
// LostShare of the hits that use the rendering. A proposal whose id is in
// in.Declined carries its confirmation.
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
	term   rules.Term
	groups map[string]*Group
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
	r     *rules.Rules
	units []Unit
	accs  []*acc
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
	an := &analysis{r: in.Rules, units: in.Units, sameKR: map[string]bool{}, places: map[string]place{}, byRune: map[rune][]int{}}
	n := map[string]int{}
	for _, t := range in.Rules.Glossary {
		if t.Status != rules.StatusApproved || t.KR == "" {
			continue
		}
		first, _ := utf8.DecodeRuneInString(t.KR)
		an.byRune[first] = append(an.byRune[first], len(an.accs))
		an.accs = append(an.accs, &acc{term: t, groups: map[string]*Group{}})
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
//
// GlossaryHits is asked about the entries whose KR the text holds only: of the
// others none occurs, so none would be a hit, and none would take the place
// of another (an entry with files stands for the entries of its KR text, and
// a covering entry occurs where it covers). The answer is the same as of the
// whole glossary, for a fraction of the work.
func (an *analysis) unit(i int32, u Unit) {
	idx := an.contained(validate.MatchText(u.KR))
	if len(idx) == 0 {
		return
	}
	an.cands = an.cands[:0]
	for _, j := range idx {
		an.cands = append(an.cands, an.accs[j].term)
	}
	sub := *an.r
	sub.Glossary = an.cands
	an.isHit = marks(an.isHit, an.cands, sub.GlossaryHits(u.File, u.KR))
	isHit := an.isHit
	pl := an.placeOf(u.File)
	for c, j := range idx {
		a := an.accs[j]
		t := a.term
		switch {
		case isHit[c]:
			h := hit{unit: i, used: t.Used(u.ZH)}
			h.compound = t.Match == rules.MatchSubstring && !an.hitsAs(&sub, c, withMatch(t, rules.MatchWord), u)
			a.add(h, pl, u.Note)
		case t.Match != rules.MatchSubstring && an.hitsAs(&sub, c, withMatch(t, rules.MatchSubstring), u):
			h := hit{unit: i, used: t.Used(u.ZH), compound: true}
			a.compound.N++
			if h.used {
				a.compound.Used++
			}
			a.extra = append(a.extra, h)
		}
	}
}

// hitsAs reports whether GlossaryHits of u would give entry c of an.cands
// (sub's glossary) if it were alt.
func (an *analysis) hitsAs(sub *rules.Rules, c int, alt rules.Term, u Unit) bool {
	an.alt = append(an.alt[:0], an.cands...)
	an.alt[c] = alt
	sub.Glossary = an.alt
	hits := sub.GlossaryHits(u.File, u.KR)
	sub.Glossary = an.cands
	an.altHit = marks(an.altHit, an.alt, hits)
	return an.altHit[c]
}

// marks sets into[i] to whether terms[i] is among hits, which GlossaryHits
// returned for terms: a subsequence of them, in their order.
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
	classFew     = iota // fewer than minN units
	classUsed           // usedShare or more use the rendering
	classNotUsed        // notUsedShare or fewer
	classMixed
)

func classOf(n, used int) int {
	switch share := float64(used) / float64(n); {
	case n < minN:
		return classFew
	case share >= usedShare:
		return classUsed
	case share <= notUsedShare:
		return classNotUsed
	}
	return classMixed
}

// option is a change of files or not_files that one kind of group proposes.
type option struct {
	kind     string
	patterns []string
	// left are the groups whose units the change leaves out.
	left          []string
	removed, lost int
	flagged       bool
}

// proposals returns the proposals about a.
func (an *analysis) proposals(a *acc) []Proposal {
	var out []Proposal
	t := a.term
	newProposal := func(kind string) Proposal {
		return Proposal{ID: itemid.Scope(t.KR, t.ZH, kind), Kind: kind, KR: t.KR, ZH: t.ZH, Patterns: []string{}, Examples: []Example{}}
	}

	// Where: by the families, and by the directories.
	var opts []option
	var mixedGroups []string
	for _, kind := range []string{GroupFamily, GroupDir} {
		o, mixed := an.scopeOption(a, kind)
		if o != nil {
			an.effect(a, o)
			opts = append(opts, *o)
		}
		mixedGroups = append(mixedGroups, mixed...)
	}
	// The change that loses fewer checks, then the shorter, then the one that
	// loses fewer units and removes more warnings; the families first.
	slices.SortStableFunc(opts, func(x, y option) int {
		return cmp.Or(compareBool(x.flagged, y.flagged), cmp.Compare(len(x.patterns), len(y.patterns)),
			cmp.Compare(x.lost, y.lost), cmp.Compare(y.removed, x.removed))
	})
	switch {
	case len(opts) > 0:
		o := opts[0]
		p := newProposal(o.kind)
		p.Patterns, p.WarnRemoved, p.UsedLost, p.Flagged = o.patterns, o.removed, o.lost, o.flagged
		p.Examples = an.groupExamples(a, o.left)
		out = append(out, p)
	case len(mixedGroups) > 0:
		p := newProposal(KindMixed)
		p.Examples = an.groupExamples(a, mixedGroups)
		out = append(out, p)
	}

	// How: as a word or inside words too.
	c := a.compound
	switch {
	case t.Match != rules.MatchSubstring && c.N >= minN && float64(c.Used) >= usedShare*float64(c.N):
		p := newProposal(KindSubstring)
		p.Match = rules.MatchSubstring
		p.Examples = an.unitExamples(a, a.extra)
		out = append(out, p)
	case t.Match == rules.MatchSubstring && c.N >= minN && float64(c.Used) <= notUsedShare*float64(c.N):
		p := newProposal(KindWord)
		p.Match = rules.MatchWord
		p.WarnRemoved, p.UsedLost = c.N-c.Used, c.Used
		p.Flagged = a.flag(c.Used)
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
// limited to the second, when they are fewer. RestGroup names no files: a
// change that needs it is not made. files is not proposed for an entry that
// has files, or whose KR text has another entry, which the change would leave
// out of those files.
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
		if pats := slices.DeleteFunc(slices.Clone(notUsed), func(p string) bool { return slices.Contains(a.term.NotFiles, p) }); len(pats) > 0 {
			nf = &option{kind: KindNotFiles, patterns: pats, left: notUsed}
		}
	}
	if !slices.Contains(used, RestGroup) && len(a.term.Files) == 0 && !an.sameKR[a.term.KR] {
		fl = &option{kind: KindFiles, patterns: used, left: notUsed}
	}
	if fl != nil && (nf == nil || len(fl.patterns) < len(nf.patterns)) {
		return fl, nil
	}
	return nf, nil
}

// effect sets what the change o does to the hits of a.
func (an *analysis) effect(a *acc, o *option) {
	t := a.term
	switch o.kind {
	case KindNotFiles:
		t.NotFiles = append(slices.Clone(t.NotFiles), o.patterns...)
	case KindFiles:
		t.Files = o.patterns
	}
	in := map[string]bool{}
	for _, h := range a.hits {
		file := an.units[h.unit].File
		still, ok := in[file]
		if !ok {
			still = an.r.Applies(t, file)
			in[file] = still
		}
		switch {
		case still:
		case h.used:
			o.lost++
		default:
			o.removed++
		}
	}
	o.flagged = a.flag(o.lost)
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
	entry := p.KR + " → " + p.ZH
	effect := fmt.Sprintf("（移除 %d 個警告，%d 個用了譯名的單位不再檢查", p.WarnRemoved, p.UsedLost)
	if p.Flagged {
		effect += "；失去檢查偏多"
	}
	effect += "）"
	pats := strings.Join(p.Patterns, "、")
	switch p.Kind {
	case KindNotFiles:
		return "詞彙範圍：" + entry + " 的 not_files 加上 " + pats + effect
	case KindFiles:
		return "詞彙範圍：" + entry + " 的 files 限定為 " + pats + effect
	case KindSubstring:
		return "詞彙範圍：" + entry + " 改為 match: substring（詞中詞也比對，不移除警告）"
	case KindWord:
		return "詞彙範圍：" + entry + " 改為 match: word" + effect
	case KindMixed:
		return "詞彙範圍：" + entry + " 有群組只部分用了譯名，需要人看"
	case KindExplained:
		return "詞彙範圍：" + entry + " 常有 note 說明不用譯名，考慮 also、縮小範圍或改譯名"
	}
	return "詞彙範圍：" + entry
}
