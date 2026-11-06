package site

import (
	"path"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/itemid"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/scope"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

const (
	// missingMax caps the text of a unit that lacks the rendering.
	missingMax = 400
	// snippetSide is the characters kept on each side of the term in a unit
	// that uses the rendering.
	snippetSide = 30
)

// GlossaryUnit is one unit an entry hits, in data/glossary/<a>-<b>.json.
// Missing units keep the KR and TW text (up to missingMax characters); used
// units keep the piece around the term. ID and Accepted are set on missing
// units only.
type GlossaryUnit struct {
	Key      string     `json:"key"`
	Family   string     `json:"family"`
	Top      string     `json:"top"`
	Sub      string     `json:"sub,omitempty"`
	Tag      *rules.Tag `json:"tag,omitempty"`
	Origin   string     `json:"origin"`
	KR       string     `json:"kr"`
	ZH       string     `json:"zh"`
	Note     string     `json:"note,omitempty"`
	ID       string     `json:"id,omitempty"`
	Accepted *Confirmed `json:"accepted,omitempty"`
}

// glossaryFile is data/glossary/<a>-<b>.json.
type glossaryFile struct {
	ID      string         `json:"id"`
	Missing []GlossaryUnit `json:"missing"`
	Used    []GlossaryUnit `json:"used"`
}

// glossaryFileName is the file name of an entry id glossary/<a>/<b>.
func glossaryFileName(id string) string {
	return strings.ReplaceAll(strings.TrimPrefix(id, itemid.GlossaryEntryPrefix), "/", "-") + ".json"
}

// buildGlossaryFiles makes the file of each entry of rep, by file name.
// accepted is rules/accepted.yaml by id (nil: none).
func buildGlossaryFiles(r *rules.Rules, rep scope.Report, accepted map[string]*Confirmed) map[string]glossaryFile {
	out := make(map[string]glossaryFile, len(rep.Entries))
	for _, e := range rep.Entries {
		term := rules.Term{KR: e.KR, ZH: e.ZH, Also: e.Also}
		for _, t := range r.Glossary {
			if t.KR == e.KR && t.ZH == e.ZH {
				term = t
				break
			}
		}
		f := glossaryFile{ID: e.ID, Missing: []GlossaryUnit{}, Used: []GlossaryUnit{}}
		for _, u := range rep.Units[e.ID] {
			node := rules.FileNode(u.File)
			g := GlossaryUnit{
				Key: u.Key, Family: r.Family(u.File), Top: node.Top, Sub: node.Sub,
				Origin: u.Origin, Note: u.Note,
			}
			if tag, ok := r.FileTag(u.File); ok {
				g.Tag = &tag
			}
			if u.Used {
				g.KR, g.ZH = usedSnippets(term, u.KR, u.ZH)
				f.Used = append(f.Used, g)
				continue
			}
			g.KR, g.ZH = clip(u.KR), clip(u.ZH)
			g.ID = itemid.UnitWarning(validate.CodeTermMissing, e.KR, e.ZH, u.KR, u.ZH)
			g.Accepted = accepted[g.ID]
			f.Missing = append(f.Missing, g)
		}
		out[glossaryFileName(e.ID)] = f
	}
	return out
}

// clip keeps s whole, or its first missingMax characters of validate.MatchText
// and … when it is longer.
func clip(s string) string {
	if len([]rune(s)) <= missingMax {
		return s
	}
	m := []rune(validate.MatchText(s))
	return string(m[:min(missingMax, len(m))]) + "…"
}

// usedSnippets cuts the KR around the first hit of the term and the TW around
// the first rendering (ZH or Also), both as validate.MatchText leaves them.
func usedSnippets(t rules.Term, kr, zh string) (string, string) {
	k, z := validate.MatchText(kr), validate.MatchText(zh)
	at, size := 0, 0
	if spans := t.Spans(k); len(spans) > 0 {
		at, size = spans[0][0], spans[0][1]-spans[0][0]
	}
	// A used unit always has a span; without one the piece starts at 0.
	zat, zsize := -1, 0
	for _, r := range append([]string{t.ZH}, t.Also...) {
		if r == "" {
			continue
		}
		if i := strings.Index(z, r); i >= 0 && (zat < 0 || i < zat) {
			zat, zsize = i, len(r)
		}
	}
	if zat < 0 {
		zat = 0
	}
	return around(k, at, size), around(z, zat, zsize)
}

// around returns the snippetSide characters before and after s[at:at+size]
// and the piece itself, with … where it is cut.
func around(s string, at, size int) string {
	pre, mid, post := []rune(s[:at]), []rune(s[at:at+size]), []rune(s[at+size:])
	out := ""
	if len(pre) > snippetSide {
		out, pre = "…", pre[len(pre)-snippetSide:]
	}
	out += string(pre) + string(mid)
	if len(post) > snippetSide {
		return out + string(post[:snippetSide]) + "…"
	}
	return out + string(post)
}

// writeGlossary writes data/glossary/index.json and the file of each entry.
func writeGlossary(w *writer, r *rules.Rules, rep scope.Report, accepted map[string]*Confirmed) error {
	entries := rep.Entries
	if entries == nil {
		entries = []scope.Entry{}
	}
	if err := w.writeJSON("glossary/index.json", struct {
		Thresholds scope.Thresholds `json:"thresholds"`
		Entries    []scope.Entry    `json:"entries"`
	}{rep.Thresholds, entries}); err != nil {
		return err
	}
	for name, f := range buildGlossaryFiles(r, rep, accepted) {
		if err := w.writeJSON(path.Join("glossary", name), f); err != nil {
			return err
		}
	}
	return nil
}
