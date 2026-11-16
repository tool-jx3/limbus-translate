package gametext

import (
	"cmp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/units"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// crossMin is the least number of characters of a KR text, the spaces around
// it aside, that the cross-group check compares: short texts ("네.", "방어")
// are rightly translated by their context.
const crossMin = 8

// lineClass is the class of the lines people say: the story text and the
// voice lines, whose fields have different names (content, dlg).
const lineClass = "line"

// voiceFolders hold, with the speech bubbles, the voice lines of lineClass.
var voiceFolders = []string{"PersonalityVoiceDlg/", "EGOVoiceDig/", "BattleAnnouncerDlg/"}

// class names what a unit may be compared with across groups: the units of
// lineClass with each other, any other unit with the units whose field has
// the same name.
func class(file, field string) string {
	switch field {
	case "content":
		if strings.HasPrefix(file, "StoryData/") {
			return lineClass
		}
	case "dlg":
		if family(file) == "BattleSpeechBubbleDlg" {
			return lineClass
		}
		for _, dir := range voiceFolders {
			if strings.HasPrefix(file, dir) {
				return lineClass
			}
		}
	}
	return field
}

// crossGroup is the units of one group (family and field) with one KR text.
type crossGroup struct {
	text validate.CrossText
	keys []string
}

// cross builds GameInput.Cross: for every unit with a usable record whose KR
// text, of crossMin characters or more, another group of its class has too,
// those other groups, by family and then field.
func cross(kr *pack.Pack, text Text, familyOf func(file string) string) (map[string][]validate.CrossText, error) {
	byText := map[string][]*crossGroup{} // by class and KR text
	var shared []string                  // the keys of byText with two groups or more
	for _, file := range kr.Files() {
		doc, err := kr.Doc(file)
		if err != nil {
			return nil, err
		}
		fam := ""
		if familyOf != nil {
			fam = familyOf(file)
		}
		for _, u := range units.Extract(file, doc) {
			if utf8.RuneCountInString(strings.TrimSpace(u.KR)) < crossMin {
				continue
			}
			zh, ok := text(file, u.Pointer, u.KR)
			if !ok {
				continue
			}
			k := class(file, u.Field) + "\x00" + u.KR
			groups := byText[k]
			i := slices.IndexFunc(groups, func(g *crossGroup) bool { return g.text.Family == fam && g.text.Field == u.Field })
			key := units.Key(file, u.Pointer)
			if i < 0 {
				groups = append(groups, &crossGroup{text: validate.CrossText{Family: fam, Field: u.Field, Key: key}})
				byText[k], i = groups, len(groups)-1
				if len(groups) == 2 {
					shared = append(shared, k)
				}
			}
			g := groups[i]
			g.keys = append(g.keys, key)
			if !slices.Contains(g.text.ZH, zh) {
				g.text.ZH = append(g.text.ZH, zh)
			}
		}
	}
	out := map[string][]validate.CrossText{}
	for _, k := range shared {
		groups := byText[k]
		slices.SortFunc(groups, func(a, b *crossGroup) int {
			return cmp.Or(cmp.Compare(a.text.Family, b.text.Family), cmp.Compare(a.text.Field, b.text.Field))
		})
		for i, g := range groups {
			others := make([]validate.CrossText, 0, len(groups)-1)
			for j, o := range groups {
				if j != i {
					others = append(others, o.text)
				}
			}
			for _, key := range g.keys {
				out[key] = others
			}
		}
	}
	return out, nil
}
