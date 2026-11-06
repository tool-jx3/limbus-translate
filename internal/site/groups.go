package site

import (
	"cmp"
	"slices"
	"strconv"

	"github.com/tool-jx3/limbus-translate/internal/rules"
)

// IndexGroup is a node of the group tree of the sidebar. Kind is "top", "sub",
// or the kind of a third-level tag ("chapter", "character", "event"); Node is
// the node name or the tag value, Label the node name or Tag.Label. Files are
// the positions in Index.Files of the files listed directly in the group.
type IndexGroup struct {
	Kind     string       `json:"kind"`
	Node     string       `json:"node"`
	Label    string       `json:"label"`
	Files    []int        `json:"files"`
	Children []IndexGroup `json:"children"`
}

// buildGroups builds the display tree of sums (in the order of Index.Files):
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
		all   []int
		subs  map[string][]int
		order []string
		bare  []int
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
		return cmp.Or(cmp.Compare(len(tops[b].all), len(tops[a].all)), cmp.Compare(a, b))
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
			out = append(out, g)
			continue
		}
		slices.Sort(rest)
		g.Files, g.Children = level(sums, rest)
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
