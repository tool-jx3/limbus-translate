package existing

import (
	"slices"
	"sort"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/store"
)

// SameTextFile is the name of the same-text index under .meta.
const SameTextFile = "same-text.json"

// MaxSameText is the most renderings For returns.
const MaxSameText = 2

// SameText holds, for each KR sentence that appears in more than one group
// (family|field|speaker), the translation it has in each group.
type SameText struct {
	RawTag string                     `json:"raw_tag"`
	Texts  map[string][]SameRendering `json:"texts"` // KR text -> one rendering per group, by group
}

// SameRendering is the translation of a KR sentence in one group.
type SameRendering struct {
	Group   string `json:"group"`
	ZH      string `json:"zh"`
	File    string `json:"file"`
	Speaker string `json:"speaker,omitempty"`
}

// TextUnit is one unit as CollectSameText reads it. ZH is "" when the unit
// has no usable translation.
type TextUnit struct{ Key, Group, File, Speaker, KR, ZH string }

// CollectSameText builds the index from units. It does not look at the
// length of the text; the caller passes only the units worth indexing. A
// sentence is kept when it appears in at least two groups and at least one
// of its units has a translation.
func CollectSameText(rawTag string, units []TextUnit) *SameText {
	us := slices.Clone(units)
	sort.SliceStable(us, func(i, j int) bool { return us[i].Key < us[j].Key })
	byKR := map[string][]TextUnit{}
	for _, u := range us {
		byKR[u.KR] = append(byKR[u.KR], u)
	}
	s := &SameText{RawTag: rawTag, Texts: map[string][]SameRendering{}}
	for kr, group := range byKR {
		byGroup := map[string][]TextUnit{}
		for _, u := range group {
			byGroup[u.Group] = append(byGroup[u.Group], u)
		}
		if len(byGroup) < 2 {
			continue
		}
		var rs []SameRendering
		for g, gus := range byGroup {
			if r, ok := mainRendering(g, gus); ok {
				rs = append(rs, r)
			}
		}
		if len(rs) == 0 {
			continue
		}
		sort.Slice(rs, func(i, j int) bool { return rs[i].Group < rs[j].Group })
		s.Texts[kr] = rs
	}
	return s
}

// mainRendering picks the translation most units of one group use. us is
// in key order, so on a tie the translation whose first unit comes first
// wins, and File and Speaker come from that first unit.
func mainRendering(group string, us []TextUnit) (SameRendering, bool) {
	counts := map[string]int{}
	var best *TextUnit
	for i := range us {
		u := &us[i]
		if u.ZH == "" {
			continue
		}
		counts[u.ZH]++
	}
	for i := range us {
		u := &us[i]
		if u.ZH == "" {
			continue
		}
		if best == nil || counts[u.ZH] > counts[best.ZH] {
			best = u
		}
	}
	if best == nil {
		return SameRendering{}, false
	}
	return SameRendering{Group: group, ZH: best.ZH, File: best.File, Speaker: best.Speaker}, true
}

// For returns the renderings of kr in groups other than group, without
// repeating a ZH, at most MaxSameText of them. A nil receiver returns nil.
func (s *SameText) For(kr, group string) []SameRendering {
	if s == nil {
		return nil
	}
	var out []SameRendering
	seen := map[string]bool{}
	for _, r := range s.Texts[kr] {
		if r.Group == group || seen[r.ZH] {
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
