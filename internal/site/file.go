// Package site builds the data of the static viewer: for each file of the
// language pack, its units with the KR text, the JP and EN texts aligned to it,
// and the TW text a player sees, with where that text came from.
package site

import (
	"fmt"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/align"
	"github.com/tool-jx3/limbus-translate/internal/gold"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// OriginStandIn is the origin of a unit with no record, or whose string in the
// built pack is not the record's: the build showed EN or KR in its place.
const OriginStandIn = "stand-in"

// maxSpeakers caps FileSummary.Speakers.
const maxSpeakers = 20

// Inputs are the packs and the state a file's units are built from. KR, ZH and
// Rules must be set; EN and JP may be nil, and then no unit has a string in that
// language.
type Inputs struct {
	KR, EN, JP *pack.Pack
	// ZH is the built pack, <tw>/LimbusCompany_Data/Lang/<lang_dir>, opened
	// with an empty prefix. Its file names are those of KR.
	ZH *pack.Pack
	// Records are the stored translations, by unit key.
	Records map[string]store.Record
	Rules   *rules.Rules
	// Review is what BuildReview needs besides the packs; BuildFile and the
	// export of the files do not look at it.
	Review *ReviewInputs
	// Preview is what Export writes the preview data from (preview.json and
	// preview/<path>); nil writes none.
	Preview *PreviewInputs
	// Gold is the gold sample (rules/gold.yaml); nil when there is none.
	Gold *gold.Sample
}

// Unit is one string of a file in the four languages. JP and EN are nil, and
// left out of the JSON, where the language has no string aligned to the unit.
type Unit struct {
	P  string   `json:"p"` // JSON pointer
	F  string   `json:"f"` // field name
	I  int      `json:"i"` // index of the dataList element, or -1
	KR string   `json:"kr"`
	JP *string  `json:"jp,omitempty"`
	EN *string  `json:"en,omitempty"`
	ZH string   `json:"zh"`           // the string of the built pack
	O  string   `json:"o"`            // a store origin, or OriginStandIn
	Fl []string `json:"fl,omitempty"` // flags of the record
	// RV are the ids of the unconfirmed review items the unit is in, sorted.
	RV []string `json:"rv,omitempty"`
	// NT is the translator's notes on the unit, joined by a line feed.
	NT string `json:"nt,omitempty"`
	// PQ is 1 when the unit is in the pending list.
	PQ int `json:"pq,omitempty"`
}

// File is the data of one file of the pack: its units in the order of KR.
type File struct {
	Path   string `json:"path"`
	Family string `json:"family"`
	Units  []Unit `json:"units"`
}

// Missing counts the KR strings of a file that a language has no string for.
// Where the language lacks the whole file, that is all of them. (The build's
// count of misaligned units, align.FileAligner.Unmatched, leaves such a file
// out: a file that is not there is not a misalignment.)
type Missing struct {
	EN int `json:"en"`
	JP int `json:"jp"`
}

// FileSummary is what the viewer's index says of a file.
type FileSummary struct {
	Path   string `json:"path"`
	Family string `json:"family"`
	// Top and Sub are the node of the file in the group tree (rules.FileNode),
	// Tag its third-level mark (rules.Rules.FileTag), if any.
	Top string     `json:"top"`
	Sub string     `json:"sub,omitempty"`
	Tag *rules.Tag `json:"tag,omitempty"`
	// Label is the place a story starts in; empty outside StoryData/.
	Label string `json:"label"`
	// Speakers are the tellers of the file, each once, first seen first.
	Speakers []string `json:"speakers"`
	Units    int      `json:"units"`
	// Origins counts the units by Unit.O.
	Origins map[string]int `json:"origins"`
	Missing Missing        `json:"missing"`
}

// BuildFile returns the units of path, a file name of the KR pack, and the
// summary of the file. The units are those of units.Extract; the JP and EN
// strings are aligned to them by align, and the TW string is read from the
// built pack at the unit's pointer. A unit has its record's origin when the
// built pack has the record's string, and is a stand-in otherwise (no record,
// or EN or KR built in its place); its flags are those of its record, if any.
func BuildFile(in Inputs, path string) (File, FileSummary, error) {
	f, sum, _, err := buildFile(in, path, nil)
	return f, sum, err
}

// buildFile is BuildFile; visit, if set, is called for each unit with the
// built unit, the KR unit and the aligned EN and JP strings (nil where there is
// none). The aligner it returns has aligned the file already when a unit asked.
func buildFile(in Inputs, path string, visit func(u Unit, ku units.Unit, en, jp *align.LangAligned)) (File, FileSummary, *align.FileAligner, error) {
	krDoc, err := in.KR.Doc(path)
	if err != nil {
		return File{}, FileSummary{}, nil, err
	}
	if !in.ZH.Has(path) {
		return File{}, FileSummary{}, nil, fmt.Errorf("site: %s is not in the built pack", path)
	}
	zhDoc, err := in.ZH.Doc(path)
	if err != nil {
		return File{}, FileSummary{}, nil, err
	}
	zhAt := make(map[string]string)
	for _, f := range zhDoc.Strings() {
		zhAt[f.Pointer] = f.Value
	}

	// One aligner per file: it aligns both languages when first asked.
	aligner := align.NewFileAligner(path, krDoc, in.EN, in.JP)
	krUnits := units.Extract(path, krDoc)
	file := File{Path: path, Family: in.Rules.Family(path), Units: make([]Unit, 0, len(krUnits))}
	sum := FileSummary{Path: path, Family: file.Family, Origins: make(map[string]int)}
	node := rules.FileNode(path)
	sum.Top, sum.Sub = node.Top, node.Sub
	if tag, ok := in.Rules.FileTag(path); ok {
		sum.Tag = &tag
	}
	for _, ku := range krUnits {
		en, jp, err := aligner.Get(ku.Pointer)
		if err != nil {
			return File{}, FileSummary{}, nil, fmt.Errorf("site: align %s: %w", path, err)
		}
		u := Unit{P: ku.Pointer, F: ku.Field, I: ku.ItemIndex, KR: ku.KR, ZH: zhAt[ku.Pointer], O: OriginStandIn}
		if en != nil {
			u.EN = &en.Value
		} else {
			sum.Missing.EN++
		}
		if jp != nil {
			u.JP = &jp.Value
		} else {
			sum.Missing.JP++
		}
		if rec, ok := in.Records[units.Key(path, ku.Pointer)]; ok {
			if rec.ZH == u.ZH {
				u.O = rec.Origin
			}
			if len(rec.Flags) > 0 {
				u.Fl = append([]string(nil), rec.Flags...)
			}
		}
		sum.Origins[u.O]++
		file.Units = append(file.Units, u)
		if visit != nil {
			visit(u, ku, en, jp)
		}
	}
	sum.Units = len(file.Units)
	sum.Label = storyLabel(path, file.Units)
	sum.Speakers = speakers(file.Units)
	return file, sum, aligner, nil
}

// storyLabel is the TW text of the first non-empty "place" of a story file, or
// its KR text where the built pack has none; "" for any other file.
func storyLabel(path string, us []Unit) string {
	if !strings.HasPrefix(path, align.StoryPrefix) {
		return ""
	}
	for _, u := range us {
		if u.F != "place" {
			continue
		}
		if u.ZH != "" {
			return u.ZH
		}
		if u.KR != "" {
			return u.KR
		}
	}
	return ""
}

// speakers lists the TW texts of the "teller" fields, each once in order of
// first appearance, at most maxSpeakers of them.
func speakers(us []Unit) []string {
	out := []string{}
	seen := make(map[string]bool)
	for _, u := range us {
		if u.F != "teller" || u.ZH == "" || seen[u.ZH] {
			continue
		}
		seen[u.ZH] = true
		out = append(out, u.ZH)
		if len(out) == maxSpeakers {
			break
		}
	}
	return out
}
