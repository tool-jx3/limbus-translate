package rules

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// StoryOrderFile is the order of the story files in the game, relative to the
// repo root. It is generated from the game's static data (see the header of
// the file) and optional: without it files are ordered by name.
const StoryOrderFile = "rules/story-order.yaml"

// StoryOrder is the content of StoryOrderFile: the groups in the order of the
// game, and in each the files in the order they play.
type StoryOrder struct {
	Groups []StoryOrderGroup `yaml:"groups"`
}

// StoryOrderGroup is a chapter, an event or the stories of a character.
type StoryOrderGroup struct {
	// ID is the group's own name, such as main-101.
	ID string `yaml:"id"`
	// Section is the level of the viewer's directory the group hangs under,
	// such as 主線; the groups of one section are listed together. Empty
	// hangs the group under the directory itself.
	Section string `yaml:"section"`
	// Chapter is the number of the main chapter the files are of, for the
	// chapter tag (Rules.FileTag); nil for a group that is not a chapter.
	Chapter *int `yaml:"chapter"`
	// Title points to the text of the pack that names the group, Label is
	// the name when there is no such text. Both may be missing.
	Title *StoryTitle `yaml:"title"`
	Label string      `yaml:"label"`
	// Files are paths in the pack, such as StoryData/S101B.json.
	Files []string `yaml:"files"`
}

// StoryTitle is a text of the pack: the fields Fields of the entry of
// dataList whose id is ID, in the file File.
type StoryTitle struct {
	File   string   `yaml:"file"`
	ID     string   `yaml:"id"`
	Fields []string `yaml:"fields"`
}

var (
	storyOrderTopFields   = map[string]bool{"groups": true}
	storyOrderGroupFields = map[string]bool{"id": true, "section": true, "chapter": true, "title": true, "label": true, "files": true}
	storyTitleFields      = map[string]bool{"file": true, "id": true, "fields": true}
)

// StoryGroup returns the group of the story order that lists file.
func (r *Rules) StoryGroup(file string) (*StoryOrderGroup, bool) {
	if r.storyAt != nil {
		i, ok := r.storyAt[file]
		if !ok {
			return nil, false
		}
		return &r.StoryOrder.Groups[i], true
	}
	// Rules that Load did not build: there is no index.
	for i := range r.StoryOrder.Groups {
		if slices.Contains(r.StoryOrder.Groups[i].Files, file) {
			return &r.StoryOrder.Groups[i], true
		}
	}
	return nil, false
}

// groupIndex returns the group of each listed file; a file listed twice (Lint
// reports it) keeps its first group.
func (o StoryOrder) groupIndex() map[string]int {
	at := map[string]int{}
	for i, g := range o.Groups {
		for _, f := range g.Files {
			if _, ok := at[f]; !ok {
				at[f] = i
			}
		}
	}
	return at
}

// Index returns the position of each listed file, counted through the groups
// from 0. A file listed twice (Lint reports it) keeps its first position.
func (o StoryOrder) Index() map[string]int {
	at := map[string]int{}
	n := 0
	for _, g := range o.Groups {
		for _, f := range g.Files {
			if _, ok := at[f]; !ok {
				at[f] = n
			}
			n++
		}
	}
	return at
}

// parseStoryOrder decodes the file. unknownTop and unknownGroup hold the
// unknown keys of the file and of each group, those of a group's title
// written title.<key>.
func parseStoryOrder(data []byte) (o StoryOrder, unknownTop []string, unknownGroup [][]string, err error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return o, nil, nil, err
	}
	if doc.Kind == 0 { // an empty file
		return o, nil, nil, nil
	}
	top := doc.Content[0]
	if top.Kind != yaml.MappingNode {
		return o, nil, nil, errors.New("must be a mapping with groups")
	}
	if err := top.Decode(&o); err != nil {
		return o, nil, nil, err
	}
	unknownTop = unknownKeys(top, storyOrderTopFields)
	for i := 0; i+1 < len(top.Content); i += 2 {
		if top.Content[i].Value != "groups" {
			continue
		}
		for _, n := range top.Content[i+1].Content {
			unknown := unknownKeys(n, storyOrderGroupFields)
			for j := 0; j+1 < len(n.Content); j += 2 {
				if n.Content[j].Value == "title" && n.Content[j+1].Kind == yaml.MappingNode {
					for _, key := range unknownKeys(n.Content[j+1], storyTitleFields) {
						unknown = append(unknown, "title."+key)
					}
				}
			}
			unknownGroup = append(unknownGroup, unknown)
		}
	}
	return o, unknownTop, unknownGroup, nil
}

// storyOrder checks the table, when there is one: no unknown key, groups with
// an id of their own and files, a chapter that is a number of one, a title
// that names a text, and each file a clean pack path listed once.
func (l *linter) storyOrder() {
	if !l.fileExists(StoryOrderFile) {
		return
	}
	data, ok := l.read(StoryOrderFile)
	if !ok {
		return
	}
	o, unknownTop, unknownGroup, err := parseStoryOrder(data)
	if err != nil {
		l.report(StoryOrderFile, "must be a mapping with groups: %v", err)
		return
	}
	for _, key := range unknownTop {
		l.report(StoryOrderFile, "unknown field %q", key)
	}
	ids := map[string]bool{}
	owner := map[string]string{}
	for i, g := range o.Groups {
		label := fmt.Sprintf("groups entry %d", i+1)
		if id := strings.TrimSpace(g.ID); id == "" {
			l.report(StoryOrderFile, "%s: id is missing", label)
		} else {
			label = "group " + id
			if ids[id] {
				l.report(StoryOrderFile, "%s: the id is used twice", label)
			}
			ids[id] = true
		}
		if i < len(unknownGroup) {
			for _, key := range unknownGroup[i] {
				l.report(StoryOrderFile, "%s: unknown field %q", label, key)
			}
		}
		if g.Chapter != nil && *g.Chapter < 0 {
			l.report(StoryOrderFile, "%s: chapter %d is not a chapter number", label, *g.Chapter)
		}
		if t := g.Title; t != nil {
			switch {
			case !packPath(t.File):
				l.report(StoryOrderFile, "%s: title: file %q is not a path in the pack", label, t.File)
			case strings.TrimSpace(t.ID) == "":
				l.report(StoryOrderFile, "%s: title: id is missing", label)
			case len(t.Fields) == 0 || slices.Contains(t.Fields, ""):
				l.report(StoryOrderFile, "%s: title: fields is missing", label)
			}
		}
		if len(g.Files) == 0 {
			l.report(StoryOrderFile, "%s: files is missing", label)
		}
		for _, f := range g.Files {
			switch first, listed := owner[f]; {
			case !packPath(f):
				l.report(StoryOrderFile, "%s: %q is not a path in the pack", label, f)
			case listed:
				l.report(StoryOrderFile, "%s: %s is already listed in %s", label, f, first)
			default:
				owner[f] = label
			}
		}
	}
}

// packPath reports whether f is a clean, relative, slash-separated path.
func packPath(f string) bool {
	return f != "" && path.Clean(f) == f && !path.IsAbs(f) && f != ".." && !strings.HasPrefix(f, "../") && !strings.Contains(f, `\`)
}

// LintStoryOrderWarnings returns a warning for each listed file that krFiles,
// the files of the KR pack, does not have: a misspelled path, or a file the
// game has dropped. krFiles empty means the pack is not at hand, and there
// are no warnings. A file that does not parse is Lint's to report.
func LintStoryOrderWarnings(root string, krFiles []string) []LintError {
	if len(krFiles) == 0 {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(StoryOrderFile)))
	if err != nil {
		return nil
	}
	o, _, _, err := parseStoryOrder(data)
	if err != nil {
		return nil
	}
	have := make(map[string]bool, len(krFiles))
	for _, k := range krFiles {
		have[k] = true
	}
	var out []LintError
	for _, g := range o.Groups {
		for _, f := range g.Files {
			if !have[f] {
				out = append(out, LintError{File: StoryOrderFile, Message: fmt.Sprintf("group %s: %s is not a file of the KR pack", g.ID, f)})
			}
		}
	}
	return out
}
