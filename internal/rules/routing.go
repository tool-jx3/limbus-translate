package rules

import (
	"fmt"
	"path"
	"slices"

	"gopkg.in/yaml.v3"
)

// RoutingFile is the file-family routing, relative to the repo root.
const RoutingFile = "guides/routing.yaml"

// Routing says which family a file belongs to and in what order families are
// translated.
type Routing struct {
	// Families are tried in order and the first one that matches a file wins,
	// so more specific families come first.
	Families []FamilyRule `yaml:"families" json:"families"`
	// Priority lists the family ids in the order they are translated, first
	// first.
	Priority []string `yaml:"priority" json:"priority"`
}

// FamilyRule is one file family.
type FamilyRule struct {
	ID string `yaml:"id" json:"id"`
	// Match are path.Match globs for the relative file path. An empty list
	// matches every file (the catch-all, which comes last).
	Match []string `yaml:"match" json:"match"`
	// Guide is the family's guide file, relative to the repo root.
	Guide string `yaml:"guide" json:"guide"`
}

// parseRouting decodes routing.yaml. Whether the content is sound is for
// globErrors and Lint to say.
func parseRouting(data []byte) (Routing, error) {
	var rt Routing
	if err := yaml.Unmarshal(data, &rt); err != nil {
		return Routing{}, err
	}
	return rt, nil
}

// globErrors lists the match patterns of rt that path.Match cannot use.
func globErrors(rt Routing) []error {
	var errs []error
	for _, f := range rt.Families {
		for _, pattern := range f.Match {
			if _, err := path.Match(pattern, ""); err != nil {
				errs = append(errs, fmt.Errorf("family %q: bad pattern %q: %w", f.ID, pattern, err))
			}
		}
	}
	return errs
}

// matches reports whether the family takes file.
func (f FamilyRule) matches(file string) bool {
	if len(f.Match) == 0 {
		return true
	}
	for _, pattern := range f.Match {
		if ok, _ := path.Match(pattern, file); ok {
			return true
		}
	}
	return false
}

// Family returns the id of the family of file, the relative path without
// language prefix (such as "StoryData/S1037B.json"). It is "" only when the
// routing has no catch-all family, which Lint rejects.
func (r *Rules) Family(file string) string {
	for _, f := range r.Routing.Families {
		if f.matches(file) {
			return f.ID
		}
	}
	return ""
}

// DefaultFamily is the id of the catch-all family of the routing, the one
// without patterns that takes every file the other families leave, or "" if
// there is none (Lint rejects such a routing).
func (r *Rules) DefaultFamily() string {
	for _, f := range r.Routing.Families {
		if len(f.Match) == 0 {
			return f.ID
		}
	}
	return ""
}

// Priority is the position of a family in the translation order; a smaller
// number is translated earlier. An unknown family comes after all others.
func (r *Rules) Priority(family string) int {
	if i := slices.Index(r.Routing.Priority, family); i >= 0 {
		return i
	}
	return len(r.Routing.Priority)
}

// familyGuide returns the guide file of a family, or "".
func (r *Rules) familyGuide(id string) string {
	for _, f := range r.Routing.Families {
		if f.ID == id {
			return f.Guide
		}
	}
	return ""
}
