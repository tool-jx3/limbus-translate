package bootstrap

import (
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// RestoreIDs puts back the [ID]s that the upstream translation replaced with
// their official Chinese names. Paratranz writes the name and a half-width
// space where KR has "[Laceration]", for example "增加5級流血 強度" for
// "[Laceration] 위력 5 부여". keywords maps each ID (without brackets) to that
// name.
//
// Only an [ID] that KR has more often than zh is restored: for each missing
// one, the first "<name> " in zh becomes "[ID]". The spaces around the new
// [ID] are left to validate.Normalize. An ID with no name in keywords, or
// whose name is not followed by a space anywhere in zh, is left as it is.
func RestoreIDs(kr, zh string, keywords map[string]string) string {
	have := map[string]int{}
	for _, id := range validate.IDs(zh) {
		have[id]++
	}
	want := map[string]int{}
	var order []string // KR's distinct IDs, in order of appearance
	for _, id := range validate.IDs(kr) {
		if want[id] == 0 {
			order = append(order, id)
		}
		want[id]++
	}

	for _, id := range order {
		name := keywords[strings.Trim(id, "[]")]
		if name == "" {
			continue
		}
		for missing := want[id] - have[id]; missing > 0; missing-- {
			before, after, found := strings.Cut(zh, name+" ")
			if !found {
				break
			}
			zh = before + id + after
		}
	}
	return zh
}
