// Package rules loads the translation rules that live in the repo: the guides
// (guides/), the glossary (glossary/) and the file-family routing
// (guides/routing.yaml). Translation batches use it to pick the guides for a
// file and speakers and the glossary terms that apply to a text.
package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// BaseGuide applies to every batch.
const BaseGuide = "guides/base.md"

// Rules is everything Load reads.
type Rules struct {
	Glossary   []Term
	Characters []Character
	Routing    Routing
}

// Load reads the rules under root, the repo root. It fails when a file is
// missing or cannot be parsed; Lint checks the content as well.
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
	if errs := globErrors(r.Routing); len(errs) > 0 {
		return nil, fmt.Errorf("%s: %w", RoutingFile, errs[0])
	}
	return r, nil
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
// layered: the base guide, the guide of the file's family, the character the
// file belongs to, then the characters of the speakers (ids, in order of
// appearance). Nothing is listed twice, and a speaker id that is not a
// character is left out.
func (r *Rules) GuidesFor(file string, speakers []string) []string {
	guides := []string{BaseGuide}
	add := func(p string) {
		if p != "" && !slices.Contains(guides, p) {
			guides = append(guides, p)
		}
	}
	add(r.familyGuide(r.Family(file)))
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
