package rules

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/rubyx"
	"github.com/tool-jx3/limbus-translate/internal/units"
	"gopkg.in/yaml.v3"
)

// RubyCustomizeFile lists where a translator may write <rubycustomize> and
// the style of the conversion, relative to the repo root. It is optional:
// without it no position is allowed.
const RubyCustomizeFile = "rules/ruby-customize.yaml"

// RubyCustomize is the content of RubyCustomizeFile.
type RubyCustomize struct {
	// Size is the font size of the reading, such as 50%.
	Size string `yaml:"size"`
	// VOffset is how far the reading is raised, such as 1em.
	VOffset string `yaml:"voffset"`
	// Fields are the allowed positions.
	Fields []RubyField `yaml:"fields"`
}

// RubyField allows a unit field in the files that match one of Files.
type RubyField struct {
	// Files are file patterns (config.MatchAny) over the paths in the pack.
	Files []string `yaml:"files"`
	// Field is the field name of the unit (units.FieldName).
	Field string `yaml:"field"`
}

var (
	rubyTopFields   = map[string]bool{"size": true, "voffset": true, "fields": true}
	rubyEntryFields = map[string]bool{"files": true, "field": true}
)

// parseRubyCustomize decodes the file. unknownTop and unknownEntry hold the
// unknown keys of the file and of each entry of fields.
func parseRubyCustomize(data []byte) (rc RubyCustomize, unknownTop []string, unknownEntry [][]string, err error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return rc, nil, nil, err
	}
	if doc.Kind == 0 { // an empty file
		return rc, nil, nil, nil
	}
	top := doc.Content[0]
	if top.Kind != yaml.MappingNode {
		return rc, nil, nil, errors.New("must be a mapping of size, voffset and fields")
	}
	if err := top.Decode(&rc); err != nil {
		return rc, nil, nil, err
	}
	unknownTop = unknownKeys(top, rubyTopFields)
	for i := 0; i+1 < len(top.Content); i += 2 {
		if top.Content[i].Value != "fields" {
			continue
		}
		for _, n := range top.Content[i+1].Content {
			unknownEntry = append(unknownEntry, unknownKeys(n, rubyEntryFields))
		}
	}
	return rc, unknownTop, unknownEntry, nil
}

// RubyCustomizeAt reports whether <rubycustomize> may be written in the unit
// at pointer (a JSON pointer) of file. An empty pointer is no unit.
func (r *Rules) RubyCustomizeAt(file, pointer string) bool {
	if pointer == "" {
		return false
	}
	field := units.FieldName(pointer)
	for _, f := range r.RubyCustomize.Fields {
		if f.Field == field && config.MatchAny(f.Files, file) {
			return true
		}
	}
	return false
}

// RubyStyle returns the style of the conversion.
func (r *Rules) RubyStyle() (rubyx.Style, error) {
	return rubyx.ParseStyle(r.RubyCustomize.Size, r.RubyCustomize.VOffset)
}

// rubyCustomize checks the whitelist, when there is one: no unknown key,
// a size and voffset that parse, and entries of valid file patterns and a
// field.
func (l *linter) rubyCustomize() {
	if !l.fileExists(RubyCustomizeFile) {
		return
	}
	data, ok := l.read(RubyCustomizeFile)
	if !ok {
		return
	}
	rc, unknownTop, unknownEntry, err := parseRubyCustomize(data)
	if err != nil {
		l.report(RubyCustomizeFile, "must be a mapping of size, voffset and fields: %v", err)
		return
	}
	for _, key := range unknownTop {
		l.report(RubyCustomizeFile, "unknown field %q", key)
	}
	if _, err := rubyx.ParseStyle(rc.Size, rc.VOffset); err != nil {
		l.report(RubyCustomizeFile, "%v", err)
	}
	for i, f := range rc.Fields {
		label := fmt.Sprintf("fields entry %d", i+1)
		if i < len(unknownEntry) {
			for _, key := range unknownEntry[i] {
				l.report(RubyCustomizeFile, "%s: unknown field %q", label, key)
			}
		}
		if len(f.Files) == 0 {
			l.report(RubyCustomizeFile, "%s: files is missing", label)
		}
		for _, p := range f.Files {
			if err := config.ValidPattern(p); err != nil {
				l.report(RubyCustomizeFile, "%s: files %q: %v", label, p, err)
			}
		}
		if strings.TrimSpace(f.Field) == "" {
			l.report(RubyCustomizeFile, "%s: field is missing", label)
		}
	}
}

// LintRubyCustomizeWarnings returns a warning for each entry of the whitelist
// whose files match no file of the KR pack krFiles (a misspelled pattern, or
// a file the game has dropped). krFiles empty means the pack is not at hand,
// and there are no warnings. A file that does not parse is Lint's to report.
func LintRubyCustomizeWarnings(root string, krFiles []string) []LintError {
	if len(krFiles) == 0 {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(RubyCustomizeFile)))
	if err != nil {
		return nil
	}
	rc, _, _, err := parseRubyCustomize(data)
	if err != nil {
		return nil
	}
	var out []LintError
	for i, f := range rc.Fields {
		if len(f.Files) == 0 || hasInvalidPattern(f.Files) {
			continue // Lint reports it
		}
		hit := false
		for _, k := range krFiles {
			if config.MatchAny(f.Files, k) {
				hit = true
				break
			}
		}
		if !hit {
			out = append(out, LintError{File: RubyCustomizeFile, Message: fmt.Sprintf("fields entry %d (%s): files %v match no file of the KR pack", i+1, f.Field, f.Files)})
		}
	}
	return out
}

func hasInvalidPattern(patterns []string) bool {
	for _, p := range patterns {
		if config.ValidPattern(p) != nil {
			return true
		}
	}
	return false
}
