package existing

import (
	"fmt"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// CodeNameDiffers is a warning: the translation of a name differs from every
// existing rendering of that name in the variants the JP or EN text points
// at.
const CodeNameDiffers = "name-differs"

// Differs returns the name-differs warning for a unit of field with KR text
// kr, translation zh, note, and the JP and EN text aligned to it ("" when
// none). It returns nil unless kr is a known name of a name field, the
// suggestion was not declined, the JP or EN text points at a variant of the
// name (see Variant.Matches), zh is none of the renderings of any variant
// they point at and the note does not write the KR name. The warning is
// about the variant that matches best (see bestVariant): its main rendering.
func (ix *Index) Differs(field, kr, zh, note, jp, en string) *validate.Issue {
	if !ix.IsName(field, kr) || ix.Declined(kr) {
		return nil
	}
	name := ix.Lookup(kr)
	if strings.Contains(note, name.KR) {
		return nil
	}
	jp, en = strings.TrimSpace(jp), strings.TrimSpace(en)
	best := bestVariant(name, jp, en)
	if best == nil {
		return nil
	}
	plainZH := Plain(zh)
	for _, v := range name.Variants {
		if !v.Matches(jp, en) {
			continue
		}
		for _, r := range v.Renderings {
			if plainZH == r.ZH {
				return nil
			}
		}
	}
	main := best.Renderings[0]
	return &validate.Issue{
		Code:     CodeNameDiffers,
		Severity: validate.SeverityWarn,
		Class:    validate.ClassStyle,
		Message: fmt.Sprintf("the existing translations name %q %q (%d units); this one says %q: use the existing name when it is the same thing, or write the KR word and the reason in the note",
			name.KR, main.ZH, main.Units, plainZH),
		Term:   name.KR,
		TermZH: main.ZH,
	}
}

// Matches reports whether the JP text jp or the EN text en, both trimmed,
// points at v: v has a rendering, and a JP text that is jp or an EN text that
// is en but for case.
func (v *Variant) Matches(jp, en string) bool {
	return len(v.Renderings) > 0 && (v.matchesJP(jp) || v.matchesEN(en))
}

func (v *Variant) matchesJP(jp string) bool { return v.JP != "" && jp == v.JP }

func (v *Variant) matchesEN(en string) bool { return v.EN != "" && strings.EqualFold(en, v.EN) }

// MatchRank says how well the JP text jp and the EN text en, both trimmed,
// point at v: 0 when they do not (see Matches); 3 when both match; 2 when one
// matches and the other does not contradict, as when the unit or the variant
// has no text in that language; 1 when one matches and the other differs.
func (v *Variant) MatchRank(jp, en string) int {
	if !v.Matches(jp, en) {
		return 0
	}
	mJP, mEN := v.matchesJP(jp), v.matchesEN(en)
	switch {
	case mJP && mEN:
		return 3
	case (mJP || v.JP == "" || jp == "") && (mEN || v.EN == "" || en == ""):
		return 2
	}
	return 1
}

// bestVariant returns the variant of n that the JP text jp and the EN text en
// point at best (the highest MatchRank), or nil when they point at none.
// Among equals the first in order wins.
func bestVariant(n *Name, jp, en string) *Variant {
	var best *Variant
	bestRank := 0
	for i := range n.Variants {
		v := &n.Variants[i]
		if rank := v.MatchRank(jp, en); rank > bestRank {
			best, bestRank = v, rank
		}
	}
	return best
}
