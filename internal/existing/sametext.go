package existing

import (
	"slices"
	"sort"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// SameTextFile is the name of the same-text index under .meta.
const SameTextFile = "same-text.json"

// MaxSameText is the most renderings For returns.
const MaxSameText = 2

// MaxSameKeys is the most units of one group the index keeps for a sentence.
const MaxSameKeys = 20

// SameText holds, for each KR sentence that appears in more than one group
// (family|field|speaker), the units of each group that have it. It holds no
// translation: For reads the translations of the units when it is asked, so
// that a translation made after the index, earlier in the same update, is a
// reference too.
type SameText struct {
	RawTag string                 `json:"raw_tag"`
	Texts  map[string][]SameGroup `json:"texts"` // KR text -> its groups, by Group
}

// SameGroup is the units of one group that have a KR sentence.
type SameGroup struct {
	Group   string   `json:"group"`
	Speaker string   `json:"speaker,omitempty"`
	Keys    []string `json:"keys"` // the group's units with this KR, by key, at most MaxSameKeys
}

// SameRendering is the translation of a KR sentence in one group.
type SameRendering struct {
	Group   string `json:"group"`
	ZH      string `json:"zh"`
	File    string `json:"file"`
	Speaker string `json:"speaker,omitempty"`
}

// TextUnit is one unit as CollectSameText reads it.
type TextUnit struct{ Key, Group, Speaker, KR string }

// CollectSameText builds the index from in. It does not look at the length
// of the text; the caller passes only the units worth indexing. A sentence is
// kept when it appears in at least two groups, whether or not any of its
// units has a translation. Each group keeps its first MaxSameKeys keys, in
// key order, and the speaker of its first unit.
func CollectSameText(rawTag string, in []TextUnit) *SameText {
	us := slices.Clone(in)
	sort.SliceStable(us, func(i, j int) bool { return us[i].Key < us[j].Key })
	byKR := map[string]map[string]*SameGroup{}
	for _, u := range us {
		groups := byKR[u.KR]
		if groups == nil {
			groups = map[string]*SameGroup{}
			byKR[u.KR] = groups
		}
		g := groups[u.Group]
		if g == nil {
			g = &SameGroup{Group: u.Group, Speaker: u.Speaker}
			groups[u.Group] = g
		}
		if len(g.Keys) < MaxSameKeys {
			g.Keys = append(g.Keys, u.Key)
		}
	}
	s := &SameText{RawTag: rawTag, Texts: map[string][]SameGroup{}}
	for kr, groups := range byKR {
		if len(groups) < 2 {
			continue
		}
		gs := make([]SameGroup, 0, len(groups))
		for _, g := range groups {
			gs = append(gs, *g)
		}
		sort.Slice(gs, func(i, j int) bool { return gs[i].Group < gs[j].Group })
		s.Texts[kr] = gs
	}
	return s
}

// For returns the renderings of kr in groups other than group, without
// repeating a ZH, at most MaxSameText of them, in group order. zhOf gives the
// translation the unit of a key has now, with ok false when it has none (an
// empty translation counts as none); a group none of whose units has one is
// left out. A nil receiver returns nil.
func (s *SameText) For(kr, group string, zhOf func(key string) (zh string, ok bool)) []SameRendering {
	if s == nil {
		return nil
	}
	var out []SameRendering
	seen := map[string]bool{}
	for _, g := range s.Texts[kr] {
		if g.Group == group {
			continue
		}
		r, ok := g.rendering(zhOf)
		if !ok || seen[r.ZH] {
			continue
		}
		seen[r.ZH] = true
		out = append(out, r)
		if len(out) == MaxSameText {
			break
		}
	}
	return out
}

// rendering picks the translation most units of g have. The keys are in key
// order, so on a tie the translation whose first unit comes first wins, and
// File is the file of that unit.
func (g SameGroup) rendering(zhOf func(string) (string, bool)) (SameRendering, bool) {
	counts := map[string]int{}
	first := map[string]string{} // ZH -> the first key that has it
	var order []string           // the ZH in the order of their first keys
	for _, k := range g.Keys {
		zh, ok := zhOf(k)
		if !ok || zh == "" {
			continue
		}
		if counts[zh] == 0 {
			first[zh] = k
			order = append(order, zh)
		}
		counts[zh]++
	}
	if len(order) == 0 {
		return SameRendering{}, false
	}
	best := order[0]
	for _, zh := range order[1:] {
		if counts[zh] > counts[best] {
			best = zh
		}
	}
	file, _, err := units.SplitKey(first[best])
	if err != nil {
		file = first[best] // a malformed key names itself
	}
	return SameRendering{Group: g.Group, ZH: best, File: file, Speaker: g.Speaker}, true
}

// Usable reports whether rec, the record of a unit whose KR text is kr, may
// serve as the translation of that unit: it was made by a translation
// (paratranz, claude or human, not a copy or a fallback), from kr itself, and
// nobody has to look at it (no needs_review flag).
func Usable(rec store.Record, kr string) bool {
	switch rec.Origin {
	case store.OriginParatranz, store.OriginClaude, store.OriginHuman:
	default:
		return false
	}
	return rec.KRSha == units.KRSha(kr) && !slices.Contains(rec.Flags, store.FlagNeedsReview)
}

// SameTextRef renders rs as one line each: "<file>（<speaker>）：<zh>", or
// "<file>：<zh>" without a speaker.
func SameTextRef(rs []SameRendering) string {
	lines := make([]string, 0, len(rs))
	for _, r := range rs {
		who := ""
		if r.Speaker != "" {
			who = "（" + r.Speaker + "）"
		}
		lines = append(lines, r.File+who+"："+r.ZH)
	}
	return strings.Join(lines, "\n")
}

// LoadSameText reads the index from .meta/same-text.json. It returns nil,
// nil when there is none.
func LoadSameText(st *store.Store) (*SameText, error) {
	var s SameText
	found, err := st.ReadMetaJSON(SameTextFile, &s)
	if err != nil || !found {
		return nil, err
	}
	return &s, nil
}

// SaveSameText writes the index to .meta/same-text.json. Like the other
// state writers it does not take the lock.
func SaveSameText(st *store.Store, s *SameText) error {
	return st.WriteMetaJSON(SameTextFile, s)
}
