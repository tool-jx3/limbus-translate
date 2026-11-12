package existing

import (
	"fmt"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// CodeNameDiffers is a warning: the translation of a name differs from every
// existing rendering of that name in the variant the JP or EN text points at.
const CodeNameDiffers = "name-differs"

// Differs returns the name-differs warning for a unit of field with KR text
// kr, translation zh, note, and the JP and EN text aligned to it ("" when
// none). It returns nil unless kr is a known name of a name field, the
// suggestion was not declined, the JP or EN text identifies one variant of
// the name, zh is none of that variant's renderings and the note does not
// write the KR name.
func (ix *Index) Differs(field, kr, zh, note, jp, en string) *validate.Issue {
	if !ix.IsName(field, kr) || ix.Declined(kr) {
		return nil
	}
	name := ix.Lookup(kr)
	if strings.Contains(note, name.KR) {
		return nil
	}
	jp, en = strings.TrimSpace(jp), strings.TrimSpace(en)
	for _, v := range name.Variants {
		if len(v.Renderings) == 0 {
			continue
		}
		if !(v.JP != "" && jp == v.JP) && !(v.EN != "" && strings.EqualFold(en, v.EN)) {
			continue
		}
		plainZH := Plain(zh)
		for _, r := range v.Renderings {
			if plainZH == r.ZH {
				return nil
			}
		}
		main := v.Renderings[0]
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
	return nil
}
