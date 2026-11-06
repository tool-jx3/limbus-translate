package rules

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// CharactersDir holds one guide per character, relative to the repo root.
const CharactersDir = "guides/characters"

// Character is the front matter of a guides/characters/<id>.md file.
type Character struct {
	ID     string `yaml:"id" json:"id"`
	Name   string `yaml:"name" json:"name"`
	NameKR string `yaml:"name_kr" json:"name_kr"`
	Gender string `yaml:"gender" json:"gender,omitempty"`
	// Number is the in-game sinner number, nil for characters without one.
	Number *int `yaml:"number" json:"number"`
	// Code is the two-digit code used in file names (P1<code>XX and the
	// personality id 1<code>XX), nil for characters without one.
	Code *string `yaml:"code" json:"code"`
	// Models are the speaker keys of the character themselves.
	Models []string `yaml:"models" json:"models"`
	// ModelsOtherSelf are keys of the same person at another time of life.
	ModelsOtherSelf []string `yaml:"models_other_self" json:"models_other_self"`
	// ModelsNotThis are keys that contain the name but are someone else, or
	// several people at once.
	ModelsNotThis []string `yaml:"models_not_this" json:"models_not_this"`
	// FileNames are the spellings of the name in voice and AbDlg file names.
	FileNames []string `yaml:"file_names" json:"file_names"`
	// Path is the guide file, relative to the repo root with forward slashes.
	Path string `yaml:"-" json:"path"`
}

// characterFields are the front matter keys a character file may have.
var characterFields = map[string]bool{
	"id": true, "name": true, "name_kr": true, "gender": true, "number": true,
	"code": true, "models": true, "models_other_self": true,
	"models_not_this": true, "file_names": true,
}

const utf8BOM = "\xef\xbb\xbf"

// splitFrontMatter returns the YAML between the leading "---" line and the
// next "---" line.
func splitFrontMatter(data []byte) (string, error) {
	s := strings.ReplaceAll(strings.TrimPrefix(string(data), utf8BOM), "\r\n", "\n")
	rest, ok := strings.CutPrefix(s, "---\n")
	if !ok {
		return "", errors.New("no front matter: the file must start with a --- line")
	}
	lines := strings.Split(rest, "\n")
	for i, l := range lines {
		if strings.TrimRight(l, " \t") == "---" {
			return strings.Join(lines[:i], "\n"), nil
		}
	}
	return "", errors.New("front matter is not closed by a --- line")
}

// parseCharacter decodes the front matter of a character guide. unknown lists
// the keys that are not character fields; Load ignores them, Lint reports
// them. The caller sets Path.
func parseCharacter(data []byte) (c Character, unknown []string, err error) {
	front, err := splitFrontMatter(data)
	if err != nil {
		return c, nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(front), &doc); err != nil {
		return c, nil, fmt.Errorf("front matter: %w", err)
	}
	if doc.Kind == 0 { // empty front matter
		return c, nil, nil
	}
	m := doc.Content[0]
	if m.Kind != yaml.MappingNode {
		return c, nil, errors.New("front matter must be a mapping")
	}
	if err := m.Decode(&c); err != nil {
		return c, nil, fmt.Errorf("front matter: %w", err)
	}
	return c, unknownKeys(m, characterFields), nil
}

// unknownKeys lists the keys of mapping m that are not in known, sorted.
func unknownKeys(m *yaml.Node, known map[string]bool) []string {
	var out []string
	for i := 0; i+1 < len(m.Content); i += 2 {
		if k := m.Content[i].Value; !known[k] {
			out = append(out, k)
		}
	}
	slices.Sort(out)
	return out
}

// CharacterByID returns the character with the given id.
func (r *Rules) CharacterByID(id string) (*Character, bool) {
	for i := range r.Characters {
		if r.Characters[i].ID == id {
			return &r.Characters[i], true
		}
	}
	return nil, false
}

// CharacterByModel finds the character a story line's model key belongs to.
// The key must match exactly. otherSelf is true when the key is the same
// person at another time of life, whose manner of speaking may differ. A key
// in models_not_this, or in no list, is not a match.
func (r *Rules) CharacterByModel(model string) (c *Character, otherSelf bool, ok bool) {
	if model == "" {
		return nil, false, false
	}
	for i := range r.Characters {
		if slices.Contains(r.Characters[i].Models, model) {
			return &r.Characters[i], false, true
		}
	}
	for i := range r.Characters {
		if slices.Contains(r.Characters[i].ModelsOtherSelf, model) {
			return &r.Characters[i], true, true
		}
	}
	return nil, false, false
}

var (
	// StoryData/P1<code>XX: the personality stories.
	storyPersonaRe = regexp.MustCompile(`^StoryData/P1(\d{2})\d{2}$`)
	// Skills_personality-<code> and Skills_Ego_Personality-<code>.
	skillPersonaRe = regexp.MustCompile(`^Skills_(?:personality|Ego_Personality)-(\d\d)$`)
	// A personality id 1<code>XX closing the file name.
	trailingIDRe = regexp.MustCompile(`_1(\d{2})\d{2}$`)
)

// nameFilePrefixes are the file names that carry a character's name right
// after the prefix: AbDlg_<name>, PersonalityVoiceDlg/Voice_<name>_...,
// EGOVoiceDig/Voice_EGO_<name>_...
var nameFilePrefixes = []string{
	"AbDlg_",
	"PersonalityVoiceDlg/Voice_",
	"EGOVoiceDig/Voice_EGO_",
}

// CharacterByFile finds the character a file belongs to. file is the relative
// path without language prefix. The name in the file name (matched against
// file_names, ignoring case) decides first; otherwise the two-digit code of
// StoryData/P1<code>XX, Skills_personality-<code> or of a trailing personality id 1<code>XX does.
func (r *Rules) CharacterByFile(file string) (*Character, bool) {
	stem := strings.TrimSuffix(file, ".json")
	for _, prefix := range nameFilePrefixes {
		rest, ok := strings.CutPrefix(stem, prefix)
		if !ok {
			continue
		}
		name, _, _ := strings.Cut(rest, "_")
		for i := range r.Characters {
			for _, fn := range r.Characters[i].FileNames {
				if strings.EqualFold(name, fn) {
					return &r.Characters[i], true
				}
			}
		}
	}
	for _, re := range []*regexp.Regexp{storyPersonaRe, skillPersonaRe, trailingIDRe} {
		if m := re.FindStringSubmatch(stem); m != nil {
			for i := range r.Characters {
				if code := r.Characters[i].Code; code != nil && *code == m[1] {
					return &r.Characters[i], true
				}
			}
		}
	}
	return nil, false
}
