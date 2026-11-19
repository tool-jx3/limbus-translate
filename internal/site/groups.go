package site

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/rules"
)

// IndexGroup is a node of the group tree of the sidebar. Kind is "top", "sub"
// (a sub node, or a section of story groups), KindStory (a group of the story
// order), or the kind of a third-level tag ("chapter", "character", "event");
// Node is the node name, the section, the group id or the tag value, Label
// the node name, the section, FileGroup.Label or Tag.Label. Files are
// the positions in Index.Files of the files listed directly in the group.
type IndexGroup struct {
	Kind     string       `json:"kind"`
	Node     string       `json:"node"`
	Label    string       `json:"label"`
	Files    []int        `json:"files"`
	Children []IndexGroup `json:"children"`
}

// FileGroup is a group of the story order (rules.StoryOrderGroup) as the
// directory shows it.
type FileGroup struct {
	// Section is the level the group hangs under; empty hangs it under the top.
	Section string `json:"section,omitempty"`
	ID      string `json:"id"`
	// Label is the name of the group: the text its title points to, else
	// its label, else its id.
	Label string `json:"label"`
}

// KindStory is the kind of an IndexGroup that is a group of the story order.
const KindStory = "story"

// storyGroups gives each file of sums that the story order lists its group.
// The name of a group is read once, from the built pack (the KR pack where
// the built one has no text).
func storyGroups(in Inputs, sums []FileSummary) {
	named := map[*rules.StoryOrderGroup]*FileGroup{}
	for i := range sums {
		g, ok := in.Rules.StoryGroup(sums[i].Path)
		if !ok {
			continue
		}
		fg := named[g]
		if fg == nil {
			fg = &FileGroup{Section: g.Section, ID: g.ID, Label: storyGroupLabel(in, g)}
			named[g] = fg
		}
		sums[i].Group = fg
	}
}

// storyGroupLabel returns the name of g: the fields of its title that have a
// text, joined by a space, else its label, else its id; after 第 N 章 when g
// is a chapter.
func storyGroupLabel(in Inputs, g *rules.StoryOrderGroup) string {
	label := ""
	if t := g.Title; t != nil {
		zh, kr := titleFields(in.ZH, t), titleFields(in.KR, t)
		var parts []string
		for _, f := range t.Fields {
			text := zh[f]
			if text == "" {
				text = kr[f]
			}
			if text != "" {
				parts = append(parts, text)
			}
		}
		if len(parts) > 0 {
			label = strings.Join(parts, " ")
		}
	}
	if label == "" {
		label = g.Label
	}
	if label == "" {
		label = g.ID
	}
	if g.Chapter != nil {
		label = "第 " + strconv.Itoa(*g.Chapter) + " 章 " + label
	}
	return label
}

// titleFields returns the string fields of the entry of dataList of t.File
// whose id is t.ID (a string, or a number written the same), in p; nil when p
// has no such file or entry.
func titleFields(p *pack.Pack, t *rules.StoryTitle) map[string]string {
	if p == nil || !p.Has(t.File) {
		return nil
	}
	doc, err := p.Doc(t.File)
	if err != nil {
		return nil
	}
	list, ok := doc.Get("/dataList")
	if !ok || list.Kind != jsonx.Array {
		return nil
	}
	for _, item := range list.Items {
		id, ok := item.Get("/id")
		if !ok || !(id.Kind == jsonx.String && id.Str == t.ID || id.Kind == jsonx.Number && id.Lit == t.ID) {
			continue
		}
		out := map[string]string{}
		for _, m := range item.Members {
			if m.Value.Kind == jsonx.String {
				if _, seen := out[m.Key]; !seen {
					out[m.Key] = m.Value.Str
				}
			}
		}
		return out
	}
	return nil
}

// buildGroups builds the display tree of sums (in the order of Index.Files):
//   - the files of a story group (FileSummary.Group) hang under the group, of
//     kind KindStory, and the groups under a sub of their section (under the
//     top when they have none), sections and groups in the order of their
//     first file; these come before the other children of the top. The rest
//     of this list is about the other files;
//   - tops and subs go by their number of files, most first, then by name;
//   - a sub with one file is not shown, its file is listed in the top; a top
//     with at most one sub of two files or more left shows none, all its
//     files are listed in the top;
//   - under a sub (or the top, where no sub is shown) the files with a tag go
//     into a group of the tag, one group even for one file; when no file of the
//     level has a tag, none is shown. Files without a tag stay in the level.
//     Chapters go by number, then characters and events by label.
func buildGroups(sums []FileSummary) []IndexGroup {
	type top struct {
		name  string
		count int   // every file of the top, those of story groups too
		all   []int // the files outside the story groups
		subs  map[string][]int
		order []string
		bare  []int
		story []IndexGroup // the sections and the groups without one
	}
	tops := map[string]*top{}
	var order []string
	for n, s := range sums {
		t := tops[s.Top]
		if t == nil {
			t = &top{name: s.Top, subs: map[string][]int{}}
			tops[s.Top] = t
			order = append(order, s.Top)
		}
		t.count++
		if s.Group != nil {
			t.story = addStoryFile(t.story, s.Group, n)
			continue
		}
		t.all = append(t.all, n)
		if s.Sub == "" {
			t.bare = append(t.bare, n)
			continue
		}
		if _, ok := t.subs[s.Sub]; !ok {
			t.order = append(t.order, s.Sub)
		}
		t.subs[s.Sub] = append(t.subs[s.Sub], n)
	}
	slices.SortStableFunc(order, func(a, b string) int {
		return cmp.Or(cmp.Compare(tops[b].count, tops[a].count), cmp.Compare(a, b))
	})
	out := make([]IndexGroup, 0, len(order))
	for _, name := range order {
		t := tops[name]
		g := IndexGroup{Kind: "top", Node: name, Label: name, Files: []int{}, Children: []IndexGroup{}}
		rest := slices.Clone(t.bare)
		var shown []string
		for _, sub := range t.order {
			if len(t.subs[sub]) == 1 {
				rest = append(rest, t.subs[sub][0])
			} else {
				shown = append(shown, sub)
			}
		}
		if len(shown) <= 1 {
			// At most one sub is left after the single-file ones fold: none
			// is shown, and the third level hangs under the top.
			g.Files, g.Children = level(sums, t.all)
			g.Children = append(append([]IndexGroup{}, t.story...), g.Children...)
			out = append(out, g)
			continue
		}
		slices.Sort(rest)
		g.Files, g.Children = level(sums, rest)
		g.Children = append(append([]IndexGroup{}, t.story...), g.Children...)
		slices.SortStableFunc(shown, func(a, b string) int {
			return cmp.Or(cmp.Compare(len(t.subs[b]), len(t.subs[a])), cmp.Compare(a, b))
		})
		for _, sub := range shown {
			sg := IndexGroup{Kind: "sub", Node: sub, Label: sub}
			sg.Files, sg.Children = level(sums, t.subs[sub])
			g.Children = append(g.Children, sg)
		}
		out = append(out, g)
	}
	return out
}

// addStoryFile adds file n to its group fg in tree, the sections and the
// groups without a section of a top, and returns tree.
func addStoryFile(tree []IndexGroup, fg *FileGroup, n int) []IndexGroup {
	find := func(list []IndexGroup, kind, node, label string) ([]IndexGroup, *IndexGroup) {
		for i := range list {
			if list[i].Kind == kind && list[i].Node == node {
				return list, &list[i]
			}
		}
		list = append(list, IndexGroup{Kind: kind, Node: node, Label: label, Files: []int{}, Children: []IndexGroup{}})
		return list, &list[len(list)-1]
	}
	if fg.Section == "" {
		var g *IndexGroup
		tree, g = find(tree, KindStory, fg.ID, fg.Label)
		g.Files = append(g.Files, n)
		return tree
	}
	var section, g *IndexGroup
	tree, section = find(tree, "sub", fg.Section, fg.Section)
	section.Children, g = find(section.Children, KindStory, fg.ID, fg.Label)
	g.Files = append(g.Files, n)
	return tree
}

// level splits the files of one level into the groups of their tags and the
// files that have none.
func level(sums []FileSummary, files []int) ([]int, []IndexGroup) {
	direct := []int{}
	children := []IndexGroup{}
	at := map[[2]string]int{}
	for _, n := range files {
		tag := sums[n].Tag
		if tag == nil {
			direct = append(direct, n)
			continue
		}
		k := [2]string{tag.Kind, tag.Value}
		i, ok := at[k]
		if !ok {
			i = len(children)
			at[k] = i
			children = append(children, IndexGroup{Kind: tag.Kind, Node: tag.Value, Label: tag.Label, Files: []int{}, Children: []IndexGroup{}})
		}
		children[i].Files = append(children[i].Files, n)
	}
	rank := func(kind string) int {
		switch kind {
		case rules.TagChapter:
			return 0
		case rules.TagCharacter:
			return 1
		}
		return 2
	}
	slices.SortStableFunc(children, func(a, b IndexGroup) int {
		if c := cmp.Compare(rank(a.Kind), rank(b.Kind)); c != 0 {
			return c
		}
		if a.Kind == rules.TagChapter {
			x, _ := strconv.Atoi(a.Node)
			y, _ := strconv.Atoi(b.Node)
			return cmp.Compare(x, y)
		}
		return cmp.Compare(a.Label, b.Label)
	})
	return direct, children
}
