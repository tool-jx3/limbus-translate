// Package gametext builds the whole-pack data (validate.Game) that the
// keyword-link, level-diff and shared-line checks need, from the KR pack files and the
// translation records.
package gametext

import (
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// uiFile is the file that holds the resonance strings.
const uiFile = "BattleUIText.json"

// Text returns the translation the pack ships for the unit of file and
// pointer whose KR text is kr; ok is false when the unit has no usable record.
type Text func(file, pointer, kr string) (zh string, ok bool)

// UsedText is the Text of a build: the text of the record of the unit when
// the record was made from kr (or the file is frozen) and is not a copy.
// The field rules are not asked: the units Load reads are all translated.
func UsedText(records map[string]store.Record, frozen []string) Text {
	return func(file, pointer, kr string) (string, bool) {
		rec, ok := records[units.Key(file, pointer)]
		if !ok || rec.Origin == store.OriginCopy {
			return "", false
		}
		if rec.KRSha != units.KRSha(kr) && !config.MatchAny(frozen, file) {
			return "", false
		}
		return rec.ZH, true
	}
}

// Load builds the game information of the pack kr with the translations text gives.
func Load(kr *pack.Pack, text Text) (*validate.Game, error) {
	in, err := Input(kr, text)
	if err != nil {
		return nil, err
	}
	return validate.NewGame(in), nil
}

// Input is Load before NewGame, for tests.
func Input(kr *pack.Pack, text Text) (validate.GameInput, error) {
	in := validate.GameInput{PrevLevel: map[string]validate.LevelText{}, SharedLines: map[string][]string{}}
	var bufFiles, skillFiles, lineFiles []string
	for _, file := range kr.Files() {
		if validate.BattleLineFile(file) {
			lineFiles = append(lineFiles, file)
		}
		switch family(file) {
		case "Bufs":
			bufFiles = append(bufFiles, file)
		case "Skills":
			skillFiles = append(skillFiles, file)
		}
	}
	slices.Sort(lineFiles)
	// Name order ignoring ".json", as names.KeywordTable reads them.
	slices.SortFunc(bufFiles, func(a, b string) int {
		return strings.Compare(strings.TrimSuffix(a, ".json"), strings.TrimSuffix(b, ".json"))
	})
	for _, file := range bufFiles {
		doc, err := kr.Doc(file)
		if err != nil {
			return in, err
		}
		in.Bufs = append(in.Bufs, bufs(file, doc, text)...)
	}
	for _, file := range skillFiles {
		doc, err := kr.Doc(file)
		if err != nil {
			return in, err
		}
		prevLevels(file, doc, text, in.PrevLevel)
	}
	for _, file := range lineFiles {
		doc, err := kr.Doc(file)
		if err != nil {
			return in, err
		}
		sharedLines(file, doc, text, in.SharedLines)
	}
	if kr.Has(uiFile) {
		doc, err := kr.Doc(uiFile)
		if err != nil {
			return in, err
		}
		in.Resonance, in.PerfectResonance = resonance(doc, text)
	}
	return in, nil
}

// family is validate.family (unexported there): the name of a file in the
// root without .json, up to the first - or _. A file in a folder has none.
func family(file string) string {
	if strings.Contains(file, "/") {
		return ""
	}
	name := strings.TrimSuffix(file, ".json")
	if i := strings.IndexAny(name, "-_"); i >= 0 {
		name = name[:i]
	}
	return name
}

func member(obj *jsonx.Node, key string) *jsonx.Node {
	if obj == nil || obj.Kind != jsonx.Object {
		return nil
	}
	for _, m := range obj.Members {
		if m.Key == key {
			return m.Value
		}
	}
	return nil
}

func dataList(doc *jsonx.Node) []*jsonx.Node {
	list, ok := doc.Get("/dataList")
	if !ok || list.Kind != jsonx.Array {
		return nil
	}
	return list.Items
}

// idOf is the key of an id value: a string, or the literal of a number.
func idOf(n *jsonx.Node) (string, bool) {
	if n == nil {
		return "", false
	}
	switch n.Kind {
	case jsonx.String:
		return n.Str, n.Str != ""
	case jsonx.Number:
		return n.Lit, true
	}
	return "", false
}

func bufs(file string, doc *jsonx.Node, text Text) []validate.GameBuf {
	var out []validate.GameBuf
	for i, entry := range dataList(doc) {
		id, ok := idOf(member(entry, "id"))
		name := member(entry, "name")
		if !ok || name == nil || name.Kind != jsonx.String {
			continue
		}
		b := validate.GameBuf{ID: id, KR: name.Str, ZH: name.Str}
		if zh, ok := text(file, "/dataList/"+strconv.Itoa(i)+"/name", name.Str); ok {
			b.ZH = zh
		}
		if ex := member(entry, "excludeAutoLink"); ex != nil && ex.Kind == jsonx.Bool && ex.Lit == "true" {
			b.ExcludeAutoLink = true
		}
		out = append(out, b)
	}
	return out
}

// sharedLines adds to shared, under the translation of each dlg of file with
// a usable record, its KR text; a KR text is listed once.
func sharedLines(file string, doc *jsonx.Node, text Text, shared map[string][]string) {
	for i, entry := range dataList(doc) {
		dlg := member(entry, "dlg")
		if dlg == nil || dlg.Kind != jsonx.String {
			continue
		}
		zh, ok := text(file, "/dataList/"+strconv.Itoa(i)+"/dlg", dlg.Str)
		if !ok || slices.Contains(shared[zh], dlg.Str) {
			continue
		}
		shared[zh] = append(shared[zh], dlg.Str)
	}
}

func resonance(doc *jsonx.Node, text Text) (res, perfect string) {
	for i, entry := range dataList(doc) {
		id := member(entry, "id")
		content := member(entry, "content")
		if id == nil || id.Kind != jsonx.String || content == nil || content.Kind != jsonx.String {
			continue
		}
		var dst *string
		switch id.Str {
		case "resonance":
			dst = &res
		case "perfect_resonance":
			dst = &perfect
		default:
			continue
		}
		*dst = content.Str
		if zh, ok := text(uiFile, "/dataList/"+strconv.Itoa(i)+"/content", content.Str); ok {
			*dst = zh
		}
	}
	return res, perfect
}

// level is one levelList entry with an integer level.
type level struct {
	n     int
	index int
	node  *jsonx.Node
}

// prevLevels fills prev with, for each desc and coin desc of a skill level,
// the text of the next lower level. A skill whose levels are not all distinct
// integers yields nothing.
func prevLevels(file string, doc *jsonx.Node, text Text, prev map[string]validate.LevelText) {
	for i, skill := range dataList(doc) {
		list := member(skill, "levelList")
		if list == nil || list.Kind != jsonx.Array {
			continue
		}
		var levels []level
		valid := true
		for j, item := range list.Items {
			n, err := strconv.Atoi(litOf(member(item, "level")))
			if err != nil {
				valid = false
				break
			}
			levels = append(levels, level{n: n, index: j, node: item})
		}
		if !valid {
			continue
		}
		sort.SliceStable(levels, func(a, b int) bool { return levels[a].n < levels[b].n })
		for k := 1; k < len(levels); k++ {
			if levels[k].n == levels[k-1].n {
				valid = false
				break
			}
		}
		if !valid {
			continue
		}
		base := "/dataList/" + strconv.Itoa(i) + "/levelList/"
		for k := 1; k < len(levels); k++ {
			lo, hi := levels[k-1], levels[k]
			loBase := base + strconv.Itoa(lo.index)
			hiBase := base + strconv.Itoa(hi.index)
			pair(file, loBase+"/desc", hiBase+"/desc", lo, member(lo.node, "desc"), member(hi.node, "desc"), text, prev)
			loCoins, hiCoins := items(member(lo.node, "coinlist")), items(member(hi.node, "coinlist"))
			for c := 0; c < len(loCoins) && c < len(hiCoins); c++ {
				loDescs, hiDescs := items(member(loCoins[c], "coindescs")), items(member(hiCoins[c], "coindescs"))
				for d := 0; d < len(loDescs) && d < len(hiDescs); d++ {
					suffix := "/coinlist/" + strconv.Itoa(c) + "/coindescs/" + strconv.Itoa(d) + "/desc"
					pair(file, loBase+suffix, hiBase+suffix, lo, member(loDescs[d], "desc"), member(hiDescs[d], "desc"), text, prev)
				}
			}
		}
	}
}

func litOf(n *jsonx.Node) string {
	if n == nil || n.Kind != jsonx.Number {
		return ""
	}
	return n.Lit
}

func items(n *jsonx.Node) []*jsonx.Node {
	if n == nil || n.Kind != jsonx.Array {
		return nil
	}
	return n.Items
}

// pair records the lower level's text under the key of the higher level's
// desc when both are strings with a usable record.
func pair(file, loPtr, hiPtr string, lo level, loDesc, hiDesc *jsonx.Node, text Text, prev map[string]validate.LevelText) {
	if loDesc == nil || loDesc.Kind != jsonx.String || hiDesc == nil || hiDesc.Kind != jsonx.String {
		return
	}
	loZH, ok := text(file, loPtr, loDesc.Str)
	if !ok {
		return
	}
	if _, ok := text(file, hiPtr, hiDesc.Str); !ok {
		return
	}
	prev[units.Key(file, hiPtr)] = validate.LevelText{Level: lo.n, KR: loDesc.Str, ZH: loZH}
}
