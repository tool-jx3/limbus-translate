// Package rules loads the translation rules that live in the repo: the guides
// (guides/), the glossary (glossary/) and the file-family routing
// (guides/routing.yaml). Translation batches use it to pick the guides for a
// file and speakers and the glossary terms that apply to a text.
package rules

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// BaseGuide applies to every batch.
const BaseGuide = "guides/base.md"

// ForeignIgnoreFile is the list of words the foreign-name warning leaves
// alone, relative to the repo root: a YAML list of strings. It is optional.
const ForeignIgnoreFile = "rules/foreign-names.ignore.yaml"

// Rules is everything Load reads.
type Rules struct {
	Glossary   []Term
	Characters []Character
	Routing    Routing
	// ForeignIgnore are the words that are never warned about as foreign
	// names (names only the LTM translation uses): common words that LTM
	// happens to give a speaker, such as 媽媽.
	ForeignIgnore []string
}

// Load reads the rules under root, the repo root. It fails when a file is
// missing (the list of ignored foreign names may be) or cannot be parsed;
// Lint checks the content as well.
func Load(root string) (*Rules, error) {
	r := &Rules{}

	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(GlossaryFile)))
	if err != nil {
		return nil, err
	}
	if r.Glossary, _, err = parseGlossary(data); err != nil {
		return nil, fmt.Errorf("%s: %w", GlossaryFile, err)
	}

	files, err := characterFiles(root)
	if err != nil {
		return nil, err
	}
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return nil, err
		}
		c, _, err := parseCharacter(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rel, err)
		}
		c.Path = rel
		r.Characters = append(r.Characters, c)
	}

	data, err = os.ReadFile(filepath.Join(root, filepath.FromSlash(RoutingFile)))
	if err != nil {
		return nil, err
	}
	if r.Routing, err = parseRouting(data); err != nil {
		return nil, fmt.Errorf("%s: %w", RoutingFile, err)
	}
	// A pattern path.Match rejects would never match, silently.
	if errs := patternErrors(r.Routing); len(errs) > 0 {
		return nil, fmt.Errorf("%s: %w", RoutingFile, errs[0])
	}

	data, err = os.ReadFile(filepath.Join(root, filepath.FromSlash(ForeignIgnoreFile)))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err == nil {
		if r.ForeignIgnore, err = parseForeignIgnore(data); err != nil {
			return nil, fmt.Errorf("%s: %w", ForeignIgnoreFile, err)
		}
	}
	return r, nil
}

// parseForeignIgnore decodes the ignore list: a list of strings, or an empty
// file.
func parseForeignIgnore(data []byte) ([]string, error) {
	var words []string
	if err := yaml.Unmarshal(data, &words); err != nil {
		return nil, err
	}
	return words, nil
}

// characterFiles lists the character guides, as paths relative to root with
// forward slashes, in name order.
func characterFiles(root string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(CharactersDir)))
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			files = append(files, CharactersDir+"/"+e.Name())
		}
	}
	return files, nil
}

// GuidesFor lists the guide files to apply to a batch, in the order they are
// layered: the base guide, the guide of the file's family, the guides: entries
// of the routing that fit the file by node and then those that fit by tag (each
// in list order), the character the file belongs to, then the characters of
// the speakers (ids, in order of appearance). Nothing is listed twice, and a
// speaker id that is not a character is left out.
func (r *Rules) GuidesFor(file string, speakers []string) []string {
	guides := []string{BaseGuide}
	add := func(p string) {
		if p != "" && !slices.Contains(guides, p) {
			guides = append(guides, p)
		}
	}
	add(r.familyGuide(r.Family(file)))
	for _, g := range r.guideLayers(file) {
		add(g)
	}
	if c, ok := r.CharacterByFile(file); ok {
		add(c.Path)
	}
	for _, id := range speakers {
		if c, ok := r.CharacterByID(id); ok {
			add(c.Path)
		}
	}
	return guides
}
