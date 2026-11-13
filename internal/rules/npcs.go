package rules

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// NPCsFile lists the speakers who need no character guide of their own, such
// as one-off passers-by, relative to the repo root. It is optional.
const NPCsFile = "guides/npcs.yaml"

// NPC is a speaker the guides know who needs no guide: a model on the list is
// no new speaker to a person (KnowsModel), and no character either
// (CharacterByModel).
type NPC struct {
	// Model is the model key of the story lines, or the speaker of an RPG
	// dialogue, as KR writes it.
	Model string `yaml:"model" json:"model"`
	// Gender is male, female or unspecified, as in a character guide; empty
	// when the text does not say.
	Gender string `yaml:"gender" json:"gender,omitempty"`
	// Note says who the speaker is, for a person.
	Note string `yaml:"note" json:"note,omitempty"`
}

var npcFields = map[string]bool{"model": true, "gender": true, "note": true}

// Genders are the values the gender of an NPC may take.
var Genders = []string{"male", "female", "unspecified"}

// parseNPCs decodes the list of NPCs: a list of mappings, or an empty file.
// unknown holds the unknown keys of each entry, in order.
func parseNPCs(data []byte) (npcs []NPC, unknown [][]string, err error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, nil, err
	}
	if doc.Kind == 0 { // an empty file
		return nil, nil, nil
	}
	list := doc.Content[0]
	if list.Kind != yaml.SequenceNode {
		return nil, nil, errors.New("must be a list of entries")
	}
	for _, n := range list.Content {
		if n.Kind != yaml.MappingNode {
			return nil, nil, fmt.Errorf("line %d: an entry must be a mapping", n.Line)
		}
		var e NPC
		if err := n.Decode(&e); err != nil {
			return nil, nil, fmt.Errorf("line %d: %w", n.Line, err)
		}
		npcs = append(npcs, e)
		unknown = append(unknown, unknownKeys(n, npcFields))
	}
	return npcs, unknown, nil
}

// npcs checks the list of NPCs, when there is one: each entry has a model,
// listed once, that no character guide lists (a character's model is no NPC),
// a gender of Genders or none, and no other field.
func (l *linter) npcs() {
	if !l.fileExists(NPCsFile) {
		return
	}
	data, ok := l.read(NPCsFile)
	if !ok {
		return
	}
	npcs, unknown, err := parseNPCs(data)
	if err != nil {
		l.report(NPCsFile, "must be a list of entries of model, gender and note: %v", err)
		return
	}
	owner := map[string]string{} // model -> the character guide that lists it
	if files, err := characterFiles(l.root); err == nil {
		for _, rel := range files {
			data, err := os.ReadFile(filepath.Join(l.root, filepath.FromSlash(rel)))
			if err != nil {
				continue // characters() reports it
			}
			c, _, err := parseCharacter(data)
			if err != nil {
				continue
			}
			for _, m := range slices.Concat(c.Models, c.ModelsOtherSelf, c.ModelsNotThis) {
				if _, dup := owner[m]; !dup {
					owner[m] = rel
				}
			}
		}
	}
	seen := map[string]bool{}
	for i, e := range npcs {
		label := fmt.Sprintf("entry %d", i+1)
		if e.Model != "" {
			label = fmt.Sprintf("model %q", e.Model)
		}
		for _, key := range unknown[i] {
			l.report(NPCsFile, "%s: unknown field %q", label, key)
		}
		if strings.TrimSpace(e.Model) == "" {
			l.report(NPCsFile, "%s: model is missing", label)
			continue
		}
		if seen[e.Model] {
			l.report(NPCsFile, "%s is listed twice", label)
		}
		seen[e.Model] = true
		if e.Gender != "" && !slices.Contains(Genders, e.Gender) {
			l.report(NPCsFile, "%s: gender %q must be one of %s", label, e.Gender, strings.Join(Genders, ", "))
		}
		if rel, ok := owner[e.Model]; ok {
			l.report(NPCsFile, "%s is listed by %s: a model of a character guide is no NPC", label, rel)
		}
	}
}
