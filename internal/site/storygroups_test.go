package site

import (
	"reflect"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/rules"
)

// The files of a story group hang under the group, the groups under their
// section, in the order of the index; the sub of the file name and its tag
// are not levels there. The files without a group stay as they were.
func TestIndexGroupsByStoryGroup(t *testing.T) {
	main1 := &FileGroup{Section: "主線", ID: "main-101", Label: "章節I 身無所屬"}
	main2 := &FileGroup{Section: "主線", ID: "main-102", Label: "章節II"}
	event := &FileGroup{Section: "插曲與活動", ID: "other-9103", Label: "間章I 地獄雞"}
	loose := &FileGroup{ID: "loose", Label: "沒有分區"}
	chapter := &rules.Tag{Kind: rules.TagChapter, Value: "1", Label: "第 1 章"}
	sums := []FileSummary{
		{Path: "StoryData/S101B.json", Top: "StoryData/", Sub: "StoryData/S", Tag: chapter, Group: main1},
		{Path: "StoryData/S101A.json", Top: "StoryData/", Sub: "StoryData/S", Tag: chapter, Group: main1},
		{Path: "StoryData/1D101A.json", Top: "StoryData/", Sub: "StoryData/D", Tag: chapter, Group: main1},
		{Path: "StoryData/S201B.json", Top: "StoryData/", Sub: "StoryData/S", Group: main2},
		{Path: "StoryData/E301B.json", Top: "StoryData/", Sub: "StoryData/E", Group: event},
		{Path: "StoryData/ES1B.json", Top: "StoryData/", Sub: "StoryData/ES", Group: event},
		{Path: "StoryData/X1.json", Top: "StoryData/", Sub: "StoryData/X", Group: loose},
		{Path: "StoryData/RPG1.json", Top: "StoryData/", Sub: "StoryData/RPG"},
		{Path: "Skills.json", Top: "Skills"},
	}
	got := buildGroups(sums)
	want := []IndexGroup{
		{Kind: "top", Node: "StoryData/", Label: "StoryData/", Files: []int{7}, Children: []IndexGroup{
			{Kind: "sub", Node: "主線", Label: "主線", Files: []int{}, Children: []IndexGroup{
				{Kind: "story", Node: "main-101", Label: "章節I 身無所屬", Files: []int{0, 1, 2}, Children: []IndexGroup{}},
				{Kind: "story", Node: "main-102", Label: "章節II", Files: []int{3}, Children: []IndexGroup{}},
			}},
			{Kind: "sub", Node: "插曲與活動", Label: "插曲與活動", Files: []int{}, Children: []IndexGroup{
				{Kind: "story", Node: "other-9103", Label: "間章I 地獄雞", Files: []int{4, 5}, Children: []IndexGroup{}},
			}},
			{Kind: "story", Node: "loose", Label: "沒有分區", Files: []int{6}, Children: []IndexGroup{}},
		}},
		{Kind: "top", Node: "Skills", Label: "Skills", Files: []int{8}, Children: []IndexGroup{}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("groups =\n%+v\nwant\n%+v", got, want)
	}
}

// Export gives each file of a story group the group, named by the text of the
// built pack its title points to (the fields joined by a space; the KR text
// where the built pack has none), else by its label, else by its id; the name
// of a chapter starts with its number.
func TestExportStoryGroups(t *testing.T) {
	in := exportInputs(t)
	const titles = "StageChapterText.json"
	kr := `{"dataList":[{"id":"chapter_d_101","chapter":"1장","chaptertitle":"속하지 못하는"},{"id":7,"name":"히스클리프"},{"id":"empty","chapter":"","chaptertitle":"밤"}]}`
	zh := `{"dataList":[{"id":"chapter_d_101","chapter":"章節I","chaptertitle":"身無所屬"},{"id":7,"name":"希斯克里夫"},{"id":"empty","chapter":"","chaptertitle":""}]}`
	in.KR = writePack(t, "KR_", map[string]string{
		"StoryData/S10.json": `{"dataList":[{"id":1,"teller":"가","content":"셋"}]}`,
		"StoryData/S2.json":  `{"dataList":[{"id":1,"teller":"가","content":"하나"}]}`,
		"StoryData/S3.json":  `{"dataList":[{"id":1,"teller":"가","content":"둘"}]}`,
		"StoryData/S4.json":  `{"dataList":[{"id":1,"teller":"가","content":"넷"}]}`,
		"StoryData/S5.json":  `{"dataList":[{"id":1,"teller":"가","content":"다섯"}]}`,
		"StoryData/S6.json":  `{"dataList":[{"id":1,"teller":"가","content":"여섯"}]}`,
		"StoryData/S7.json":  `{"dataList":[{"id":1,"teller":"가","content":"일곱"}]}`,
		titles:               kr,
	})
	in.EN, in.JP = writePack(t, "EN_", map[string]string{titles: kr}), writePack(t, "JP_", map[string]string{titles: kr})
	in.ZH = writePack(t, "", map[string]string{
		"StoryData/S10.json": `{"dataList":[{"id":1,"teller":"甲","content":"三"}]}`,
		"StoryData/S2.json":  `{"dataList":[{"id":1,"teller":"甲","content":"一"}]}`,
		"StoryData/S3.json":  `{"dataList":[{"id":1,"teller":"甲","content":"二"}]}`,
		"StoryData/S4.json":  `{"dataList":[{"id":1,"teller":"甲","content":"四"}]}`,
		"StoryData/S5.json":  `{"dataList":[{"id":1,"teller":"甲","content":"五"}]}`,
		"StoryData/S6.json":  `{"dataList":[{"id":1,"teller":"甲","content":"六"}]}`,
		"StoryData/S7.json":  `{"dataList":[{"id":1,"teller":"甲","content":"七"}]}`,
		titles:               zh,
	})
	in.Records = nil
	one := 1
	in.Rules.StoryOrder = rules.StoryOrder{Groups: []rules.StoryOrderGroup{
		{ID: "main-101", Section: "主線", Chapter: &one, Files: []string{"StoryData/S10.json"},
			Title: &rules.StoryTitle{File: titles, ID: "chapter_d_101", Fields: []string{"chaptertitle"}}},
		{ID: "event", Files: []string{"StoryData/S6.json"}, Title: &rules.StoryTitle{File: titles, ID: "chapter_d_101", Fields: []string{"chapter", "chaptertitle"}}},
		{ID: "personality-07", Files: []string{"StoryData/S2.json"}, Title: &rules.StoryTitle{File: titles, ID: "7", Fields: []string{"name"}}},
		{ID: "kr", Files: []string{"StoryData/S3.json"}, Title: &rules.StoryTitle{File: titles, ID: "empty", Fields: []string{"chapter", "chaptertitle"}}},
		{ID: "labelled", Label: "其他", Files: []string{"StoryData/S4.json"}, Title: &rules.StoryTitle{File: titles, ID: "nothing", Fields: []string{"name"}}},
		{ID: "bare", Files: []string{"StoryData/S5.json"}, Title: &rules.StoryTitle{File: "Missing.json", ID: "x", Fields: []string{"name"}}},
	}}
	out := t.TempDir()
	if _, err := Export(in, Meta{}, out); err != nil {
		t.Fatal(err)
	}
	var idx Index
	readJSON(t, out, "index.json", &idx)
	got := map[string]*FileGroup{}
	for _, f := range idx.Files {
		got[f.Path] = f.Group
	}
	want := map[string]*FileGroup{
		"StoryData/S10.json":    {Section: "主線", ID: "main-101", Label: "第 1 章 身無所屬"},
		"StoryData/S6.json":     {ID: "event", Label: "章節I 身無所屬"},
		"StoryData/S2.json":     {ID: "personality-07", Label: "希斯克里夫"},
		"StoryData/S3.json":     {ID: "kr", Label: "밤"},
		"StoryData/S4.json":     {ID: "labelled", Label: "其他"},
		"StoryData/S5.json":     {ID: "bare", Label: "bare"},
		"StoryData/S7.json":     nil,
		"StageChapterText.json": nil,
	}
	if !reflect.DeepEqual(got, want) {
		for p, g := range got {
			t.Logf("%s: %+v", p, g)
		}
		t.Errorf("groups of the files differ from %d expected", len(want))
	}
	if g := idx.Groups[0]; g.Node != "StoryData/" || len(g.Children) == 0 || g.Children[0].Kind != "sub" || g.Children[0].Node != "主線" ||
		len(g.Children[0].Children) != 1 || g.Children[0].Children[0].Kind != "story" || g.Children[0].Children[0].Label != "第 1 章 身無所屬" {
		t.Errorf("tree = %+v", idx.Groups)
	}
}
