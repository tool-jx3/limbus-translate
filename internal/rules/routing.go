package rules

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// RoutingFile is the file-family routing, relative to the repo root.
const RoutingFile = "guides/routing.yaml"

// Routing says which family a file belongs to and in what order families are
// translated.
type Routing struct {
	// Families are in routing order: the order decides which family takes a
	// node that two families list with a wildcard. The last one lists no
	// nodes and takes every file the others leave.
	Families []FamilyRule `yaml:"families" json:"families"`
	// Priority lists the family ids in the order they are translated, first
	// first.
	Priority []string `yaml:"priority" json:"priority"`
	// Guides are extra guide files layered on a family's guide by node or by
	// tag (see GuideRule), in the order they are listed.
	Guides []GuideRule `yaml:"guides,omitempty" json:"guides,omitempty"`
}

// GuideRule adds Guide to the files it fits.
type GuideRule struct {
	// When is a node name or a path.Match pattern over node names (matched
	// against the top and the sub node of FileNode, the same syntax as nodes),
	// or a tag: chapter:<n>, character:<id> or event:<id> (see FileTag).
	When string `yaml:"when" json:"when"`
	// Guide is the guide file, relative to the repo root.
	Guide string `yaml:"guide" json:"guide"`
}

// FamilyRule is one file family.
type FamilyRule struct {
	ID string `yaml:"id" json:"id"`
	// Nodes are the nodes of the group tree the family takes (see FileNode):
	// a full node name such as "Skills_Enemy", or a path.Match pattern over
	// node names such as "Skills_Enemy*". An empty list is the fallback
	// family (unsorted), which takes the files no pattern of another family
	// reaches; it comes last.
	Nodes []string `yaml:"nodes" json:"nodes"`
	// Guide is the family's guide file, relative to the repo root.
	Guide string `yaml:"guide" json:"guide"`
	// Include lists shared guide files (guides/shared/<topic>.md) the family
	// takes in, in order, right after its guide.
	Include []string `yaml:"include,omitempty" json:"include,omitempty"`
	// Glossary is how the glossary binds the family: "" or "hard" (a missing
	// rendering is rejected at submission unless the note names the term) or
	// "warn" (a missing rendering stays a warning).
	Glossary string `yaml:"glossary,omitempty" json:"glossary,omitempty"`
}

// Glossary modes of a family.
const (
	GlossaryHard = "hard"
	GlossaryWarn = "warn"
)

// parseRouting decodes routing.yaml. Whether the content is sound is for
// patternErrors and Lint to say. The old match key is read as absent; Lint
// reports it (see matchKeys).
func parseRouting(data []byte) (Routing, error) {
	var rt Routing
	if err := yaml.Unmarshal(data, &rt); err != nil {
		return Routing{}, err
	}
	return rt, nil
}

// matchKeys lists the labels of the families of data that still have the
// retired match key, in file order.
func matchKeys(data []byte) []string {
	var doc yaml.Node
	if yaml.Unmarshal(data, &doc) != nil || len(doc.Content) == 0 {
		return nil
	}
	var out []string
	forKey(doc.Content[0], "families", func(list *yaml.Node) {
		for i, fam := range list.Content {
			has := false
			id := ""
			forKey(fam, "match", func(*yaml.Node) { has = true })
			forKey(fam, "id", func(v *yaml.Node) { id = v.Value })
			if has {
				if id == "" {
					id = fmt.Sprintf("#%d", i+1)
				}
				out = append(out, id)
			}
		}
	})
	return out
}

// forKey calls fn with the value of key in the mapping node m, if it has one.
func forKey(m *yaml.Node, key string, fn func(*yaml.Node)) {
	if m == nil || m.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			fn(m.Content[i+1])
			return
		}
	}
}

// patternErrors lists the node patterns of rt that path.Match cannot use.
func patternErrors(rt Routing) []error {
	var errs []error
	for _, f := range rt.Families {
		for _, pattern := range f.Nodes {
			if pattern == "" {
				errs = append(errs, fmt.Errorf("family %q: a nodes pattern is empty", f.ID))
			} else if _, err := path.Match(pattern, ""); err != nil {
				errs = append(errs, fmt.Errorf("family %q: bad pattern %q: %w", f.ID, pattern, err))
			}
		}
	}
	return errs
}

// isWildcard reports whether a nodes entry is a pattern and not a full name.
func isWildcard(pattern string) bool {
	return strings.ContainsAny(pattern, `*?[\`)
}

// fits reports whether the family lists node, by full name (exact) or by a
// wildcard pattern.
func (f FamilyRule) fits(node string) (exact, wildcard bool) {
	for _, pattern := range f.Nodes {
		if !isWildcard(pattern) {
			exact = exact || pattern == node
		} else if ok, _ := path.Match(pattern, node); ok {
			wildcard = true
		}
	}
	return exact, wildcard
}

// routeNode returns the index of the family that takes node, or -1: the first
// family that lists the full name, else the first one a wildcard pattern fits.
func (rt Routing) routeNode(node string) int {
	if node == "" {
		return -1
	}
	wild := -1
	for i, f := range rt.Families {
		exact, wildcard := f.fits(node)
		if exact {
			return i
		}
		if wildcard && wild < 0 {
			wild = i
		}
	}
	return wild
}

// route returns the index of the family of file, the relative path without
// language prefix (such as "StoryData/S1037B.json"): the sub node first, then
// the top node, then the fallback family. -1 when the routing has no fallback.
func (rt Routing) route(file string) int {
	n := FileNode(file)
	if i := rt.routeNode(n.Sub); i >= 0 {
		return i
	}
	if i := rt.routeNode(n.Top); i >= 0 {
		return i
	}
	return rt.fallback()
}

// fallback is the index of the family without nodes, or -1.
func (rt Routing) fallback() int {
	for i, f := range rt.Families {
		if len(f.Nodes) == 0 {
			return i
		}
	}
	return -1
}

// Family returns the id of the family of file, the relative path without
// language prefix (such as "StoryData/S1037B.json"). It is "" only when the
// routing has no fallback family, which Lint rejects.
func (r *Rules) Family(file string) string {
	if i := r.Routing.route(file); i >= 0 {
		return r.Routing.Families[i].ID
	}
	return ""
}

// GlossaryWarnOnly reports whether the family of file has glossary: warn, so
// that a glossary rendering missing from a translation is only a warning.
func (r *Rules) GlossaryWarnOnly(file string) bool {
	if i := r.Routing.route(file); i >= 0 {
		return r.Routing.Families[i].Glossary == GlossaryWarn
	}
	return false
}

// DefaultFamily is the id of the fallback family of the routing, the one
// without nodes that takes every file the other families leave, or "" if
// there is none (Lint rejects such a routing).
func (r *Rules) DefaultFamily() string {
	if i := r.Routing.fallback(); i >= 0 {
		return r.Routing.Families[i].ID
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

// familyInclude returns the include files of a family, or nil.
func (r *Rules) familyInclude(id string) []string {
	for _, f := range r.Routing.Families {
		if f.ID == id {
			return f.Include
		}
	}
	return nil
}

// guideLayerTags are the prefixes of a GuideRule.When that name a tag.
var guideLayerTags = []string{ChapterPrefix, CharacterPrefix, EventPrefix}

// isTagWhen reports whether when names a tag, not a node.
func isTagWhen(when string) bool {
	return slices.ContainsFunc(guideLayerTags, func(p string) bool { return strings.HasPrefix(when, p) })
}

// whenItem turns a GuideRule.When into the scope item matchScope reads: a tag
// as it is, a node as a group: token.
func whenItem(when string) string {
	if isTagWhen(when) {
		return when
	}
	return GroupPrefix + when
}

// whenError describes what is wrong with a GuideRule.When, or returns "".
// characters are the character ids a character: tag may name; nil leaves that
// tag unjudged.
func whenError(when string, characters []string) string {
	if prefix, _, ok := cutScopeToken(when); ok && !isTagWhen(when) {
		return fmt.Sprintf("when %q: %s is not a node or a tag (chapter:, character:, event:)", when, prefix)
	}
	if when == "" {
		return "when is empty"
	}
	if msg := scopeItemError(whenItem(when), nil, characters); msg != "" {
		return fmt.Sprintf("when %q: %s", when, msg)
	}
	return ""
}

// guideLayers lists the guide files of the entries of Guides that fit file,
// the node entries first and then the tag entries, each in list order.
func (r *Rules) guideLayers(file string) []string {
	var out []string
	for _, tags := range []bool{false, true} {
		for _, g := range r.Routing.Guides {
			if isTagWhen(g.When) == tags && g.Guide != "" && r.matchScope([]string{whenItem(g.When)}, file) {
				out = append(out, g.Guide)
			}
		}
	}
	return out
}
