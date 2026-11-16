package validate

import (
	"regexp"
	"slices"
	"sort"
	"strings"
	"unicode"
)

// GameBuf is one entry of the Bufs* files.
type GameBuf struct {
	ID              string
	KR              string // the KR name
	ZH              string // the name the pack ships: the record's text, or KR when the unit has no usable record
	ExcludeAutoLink bool
}

// LevelText is the text of the same field one uptie level lower.
type LevelText struct {
	Level  int // the lower level
	KR, ZH string
}

// GameInput is what NewGame builds a Game from.
type GameInput struct {
	Bufs []GameBuf // in file-name order, then document order
	// Resonance and PerfectResonance are the texts the pack ships for the ids
	// resonance and perfect_resonance of BattleUIText.json; "" when there is none.
	Resonance, PerfectResonance string
	// PrevLevel is, by unit key (units.Key), the text one level lower of a
	// Skills desc or coin desc; a unit without a lower level, or whose lower
	// level has no usable record, has no entry.
	PrevLevel map[string]LevelText
	// SharedLines is, by the translation the pack ships, the KR texts of the
	// battle lines (the dlg of the files of BattleLineFile) that have it, each
	// once.
	SharedLines map[string][]string
	// Cross is, by unit key (units.Key), the other groups (family and field)
	// that have the KR text of the unit, each with its translations; a unit
	// whose text no other group has has no entry.
	Cross map[string][]CrossText
}

// CrossText is the translations a KR text has in one group: the units of a
// family and a field.
type CrossText struct {
	Family, Field string
	ZH            []string // each translation once, in the order met
	Key           string   // the first unit of the group with the text
}

// Game is the data of the whole KR pack that the keyword-link, level-diff,
// shared-line and cross-group checks need.
type Game struct {
	ids       map[string][]string // by shipped name, the ids that have it, in input order
	byFirst   map[rune][]string   // the names that are matched, by their first rune
	bufIDs    map[string]bool     // every buff id
	krName    map[string]string   // by id, its KR name
	resonance []rune
	perfect   []rune
	prev      map[string]LevelText   // PrevLevel, see prevLevel
	shared    map[string][]string    // SharedLines, see otherLines
	cross     map[string][]CrossText // Cross, see crossGroup
}

// NewGame builds a Game. An id listed more than once keeps its first entry.
// A name is matched unless it is empty, is Unknown in any case, or an entry
// with it is excluded from auto-linking.
func NewGame(in GameInput) *Game {
	g := &Game{
		ids:       map[string][]string{},
		byFirst:   map[rune][]string{},
		bufIDs:    map[string]bool{},
		krName:    map[string]string{},
		resonance: []rune(in.Resonance),
		perfect:   []rune(in.PerfectResonance),
		prev:      in.PrevLevel,
		shared:    in.SharedLines,
		cross:     in.Cross,
	}
	var order []string
	excluded := map[string]bool{}
	for _, b := range in.Bufs {
		if g.bufIDs[b.ID] {
			continue
		}
		g.bufIDs[b.ID] = true
		g.krName[b.ID] = b.KR
		if b.ZH == "" || strings.EqualFold(b.ZH, "unknown") {
			continue
		}
		if b.ExcludeAutoLink {
			excluded[b.ZH] = true
		}
		if g.ids[b.ZH] == nil {
			order = append(order, b.ZH)
		}
		g.ids[b.ZH] = append(g.ids[b.ZH], b.ID)
	}
	for _, name := range order {
		if !excluded[name] {
			first := []rune(name)[0]
			g.byFirst[first] = append(g.byFirst[first], name)
		}
	}
	return g
}

// Link is a name of the text that the game turns into a keyword link.
type Link struct {
	Start int // byte offset of the name in the text
	Name  string
	IDs   []string // every id whose name it is, in the order of GameInput.Bufs
}

// skillTagRe matches a [text] on one line, up to the first ].
var skillTagRe = regexp.MustCompile(`\[(.*?)\]`)

// linkAfter are the characters besides spaces and digits after which a name
// is linked.
const linkAfter = "・「」『』.,/&;"

// zwsp is U+200B ZERO WIDTH SPACE: a name next to it is not matched.
const zwsp = '\u200B'

// Linked returns the names of text the game links, by its code: text is a
// field as written; tagPass says the field goes through the skill-tag pass first.
func (g *Game) Linked(text string, tagPass bool) []Link {
	rs, offs := g.runes(text, tagPass)

	type cand struct{ start, n int } // in runes
	var cands []cand
	names := map[cand]string{}
	for i, r := range rs {
		for _, name := range g.byFirst[r] {
			nr := []rune(name)
			if i+len(nr) <= len(rs) && string(rs[i:i+len(nr)]) == name {
				c := cand{i, len(nr)}
				cands = append(cands, c)
				names[c] = name
			}
		}
	}
	// Longest first; the same length by position.
	sort.SliceStable(cands, func(a, b int) bool { return cands[a].n > cands[b].n })

	used := make([]bool, len(rs))
	var links []Link
	for _, c := range cands {
		end := c.start + c.n
		if slices.Contains(used[c.start:end], true) {
			continue
		}
		if (c.start > 0 && rs[c.start-1] == zwsp) || (end < len(rs) && rs[end] == zwsp) {
			continue
		}
		for i := c.start; i < end; i++ {
			used[i] = true
		}
		// A name taken keeps its place even when it is not linked.
		if !g.linksBefore(rs[end:]) {
			continue
		}
		name := names[c]
		links = append(links, Link{Start: offs[c.start], Name: name, IDs: g.ids[name]})
	}
	sort.Slice(links, func(a, b int) bool { return links[a].Start < links[b].Start })
	return links
}

// runes returns the runes of text the names are matched in, with the byte
// offset in text of each. With tagPass a [text] whose text is no buff id
// becomes a single <, as the skill-tag pass turns it into a color tag.
func (g *Game) runes(text string, tagPass bool) ([]rune, []int) {
	var rs []rune
	var offs []int
	add := func(s string, base int) {
		for i, r := range s {
			rs = append(rs, r)
			offs = append(offs, base+i)
		}
	}
	at := 0
	if tagPass {
		for _, m := range skillTagRe.FindAllStringSubmatchIndex(text, -1) {
			if g.bufIDs[text[m[2]:m[3]]] {
				continue
			}
			add(text[at:m[0]], at)
			rs = append(rs, '<')
			offs = append(offs, m[0])
			at = m[1]
		}
	}
	add(text[at:], at)
	return rs, offs
}

// linksBefore reports whether a name followed by rest is linked.
func (g *Game) linksBefore(rest []rune) bool {
	if n := len(g.resonance); n > 0 && len(rest) > n+1 &&
		strings.Contains(string(rest[:n+1]), string(g.resonance)) {
		return false
	}
	if n := len(g.perfect); n > 0 && len(rest) > n+2 &&
		strings.Contains(strings.ReplaceAll(string(rest[:n+2]), " ", ""), string(g.perfect)) {
		return false
	}
	if len(rest) == 0 {
		return true
	}
	r := rest[0]
	return unicode.IsSpace(r) || unicode.IsDigit(r) || strings.ContainsRune(linkAfter, r)
}

// keywordScope lists the fields the game links names in, by family;
// keywordTagPass are the families whose fields go through the skill-tag pass
// first.
var (
	keywordScope = map[string][]string{
		"Skills":    {"levelList/desc", "levelList/coinlist/coindescs/desc", "levelList/coinlist/coindescs/summary"},
		"Passive":   {"desc", "summary"},
		"Passives":  {"desc", "summary"},
		"EGOgift":   {"desc", "simpleDesc/simpleDesc"},
		"PanicInfo": {"lowMoraleDescription", "panicDescription"},
	}
	keywordTagPass = map[string]bool{"Skills": true, "Passive": true, "Passives": true}
)

// unnamedLinks returns, as 「name」 (id), the links of zh to states kr names
// neither by an [id] nor by the KR name of any id of the link.
func (g *Game) unnamedLinks(kr, zh, fam string) []string {
	var out []string
	for _, l := range g.Linked(zh, keywordTagPass[fam]) {
		named := false
		for _, id := range l.IDs {
			if strings.Contains(kr, "["+id+"]") || (g.krName[id] != "" && strings.Contains(kr, g.krName[id])) {
				named = true
				break
			}
		}
		if s := "「" + l.Name + "」 (" + l.IDs[0] + ")"; !named && !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}
