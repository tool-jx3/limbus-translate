package rules_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/rules"
)

func writeStoryOrder(t *testing.T, root, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rules.StoryOrderFile))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The table of the repo: in the game the story before a battle (B) comes
// before the one after it (A), and a dungeon after the chapter's own stories.
func TestStoryOrderOfTheRepo(t *testing.T) {
	at := loadRepo(t).StoryOrder.Index()
	for _, pair := range [][2]string{
		{"StoryData/S001B.json", "StoryData/S001A.json"},
		{"StoryData/S111B.json", "StoryData/1D101A.json"},
		{"StoryData/1D306A.json", "StoryData/S201B.json"},
		{"StoryData/S632B.json", "StoryData/S632I.json"},
		{"StoryData/S632I.json", "StoryData/S632A.json"},
	} {
		a, okA := at[pair[0]]
		b, okB := at[pair[1]]
		if !okA || !okB || a >= b {
			t.Errorf("%s at %d (%v), %s at %d (%v): want the first before the second", pair[0], a, okA, pair[1], b, okB)
		}
	}
}

func TestStoryOrderIndex(t *testing.T) {
	o := rules.StoryOrder{Groups: []rules.StoryOrderGroup{
		{ID: "a", Files: []string{"StoryData/S1B.json", "StoryData/S1A.json"}},
		{ID: "b", Files: []string{"StoryData/E1.json"}},
	}}
	at := o.Index()
	if len(at) != 3 || at["StoryData/S1B.json"] != 0 || at["StoryData/S1A.json"] != 1 || at["StoryData/E1.json"] != 2 {
		t.Errorf("Index = %v", at)
	}
	if at := (rules.StoryOrder{}).Index(); len(at) != 0 {
		t.Errorf("Index of no table = %v", at)
	}
}

func TestStoryOrderMissingFile(t *testing.T) {
	root := copyRules(t)
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(rules.StoryOrderFile))); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	r, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.StoryOrder.Groups) != 0 {
		t.Errorf("groups without the file: %v", r.StoryOrder.Groups)
	}
	if errs := rules.Lint(root); len(errs) != 0 {
		t.Errorf("lint without the file: %v", errs)
	}
	if w := rules.LintStoryOrderWarnings(root, []string{"StoryData/S1B.json"}); len(w) != 0 {
		t.Errorf("warnings without the file: %v", w)
	}
}

func TestStoryOrderLoad(t *testing.T) {
	root := copyRules(t)
	writeStoryOrder(t, root, "groups:\n  - id: main-100\n    files:\n      - StoryData/S1B.json  # 推定\n      - StoryData/S1A.json\n")
	r, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if at := r.StoryOrder.Index(); len(at) != 2 || at["StoryData/S1A.json"] != 1 {
		t.Errorf("Index = %v", at)
	}
	writeStoryOrder(t, root, "groups: [1, 2\n")
	if _, err := rules.Load(root); err == nil {
		t.Error("Load of a file that does not parse: no error")
	}
}

func TestStoryOrderLint(t *testing.T) {
	const good = "groups:\n  - id: a\n    files:\n      - StoryData/S1B.json\n"
	cases := map[string]string{
		"not a mapping":      "- a\n",
		"unknown top key":    good + "extra: 1\n",
		"unknown group key":  good + "    other: 1\n",
		"missing id":         good + "  - files:\n      - StoryData/S2B.json\n",
		"duplicate id":       good + "  - id: a\n    files:\n      - StoryData/S2B.json\n",
		"no files":           good + "  - id: b\n",
		"empty file":         good + "      - ''\n",
		"same file twice":    good + "      - StoryData/S1B.json\n",
		"same file two sets": good + "  - id: b\n    files:\n      - StoryData/S1B.json\n",
		"not a pack path":    good + "      - /StoryData/S2B.json\n",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			root := copyRules(t)
			writeStoryOrder(t, root, content)
			errs := rules.Lint(root)
			if len(errs) != 1 {
				t.Fatalf("errors = %v, want 1", errs)
			}
			if le, ok := errs[0].(rules.LintError); !ok || le.File != rules.StoryOrderFile {
				t.Errorf("error = %#v, want one of %s", errs[0], rules.StoryOrderFile)
			}
		})
	}
	root := copyRules(t)
	writeStoryOrder(t, root, good)
	if errs := rules.Lint(root); len(errs) != 0 {
		t.Errorf("good file: %v", errs)
	}
}

// A listed file the KR pack does not have (a misspelled path, or a file the
// game has dropped) gets a warning; without the pack there is nothing to say.
func TestStoryOrderWarnings(t *testing.T) {
	root := copyRules(t)
	writeStoryOrder(t, root, "groups:\n  - id: a\n    files:\n      - StoryData/S1B.json\n      - StoryData/S9B.json\n")
	w := rules.LintStoryOrderWarnings(root, []string{"StoryData/S1B.json", "StoryData/S2B.json"})
	if len(w) != 1 || w[0].File != rules.StoryOrderFile {
		t.Fatalf("warnings = %v, want 1 of %s", w, rules.StoryOrderFile)
	}
	if w := rules.LintStoryOrderWarnings(root, nil); len(w) != 0 {
		t.Errorf("warnings without the pack = %v", w)
	}
}

// A group may say where the directory puts it (section), what it is called
// (label, or title: a text of the pack) and which chapter its files are of.
func TestStoryOrderGroupFields(t *testing.T) {
	root := copyRules(t)
	writeStoryOrder(t, root, "groups:\n"+
		"  - id: main-110\n    section: 主線\n    chapter: 10\n"+
		"    title: { file: StageChapterText.json, id: chapter_n_110, fields: [chapter, chaptertitle] }\n"+
		"    files:\n      - StoryData/S1003B.json\n      - StoryData/S9991B.json\n"+
		"  - id: other-0\n    section: 插曲與活動\n    label: 其他\n    files:\n      - StoryData/E000X.json\n")
	if errs := rules.Lint(root); len(errs) != 0 {
		t.Fatalf("lint: %v", errs)
	}
	r, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	g, ok := r.StoryGroup("StoryData/S9991B.json")
	if !ok || g.ID != "main-110" || g.Section != "主線" || g.Chapter == nil || *g.Chapter != 10 || g.Title == nil ||
		g.Title.File != "StageChapterText.json" || g.Title.ID != "chapter_n_110" || len(g.Title.Fields) != 2 || g.Title.Fields[1] != "chaptertitle" {
		t.Errorf("StoryGroup = %+v, %v", g, ok)
	}
	if g, ok := r.StoryGroup("StoryData/E000X.json"); !ok || g.Label != "其他" || g.Chapter != nil || g.Title != nil {
		t.Errorf("StoryGroup of E000X = %+v, %v", g, ok)
	}
	if _, ok := r.StoryGroup("StoryData/RPG1.json"); ok {
		t.Error("StoryGroup of a file that is not listed: ok")
	}

	// The chapter of the group decides the tag, whatever the name says: by
	// its name S9991B would be of chapter 99.
	if tag, ok := r.FileTag("StoryData/S9991B.json"); !ok || tag.Kind != rules.TagChapter || tag.Value != "10" {
		t.Errorf("FileTag of S9991B = %+v, %v, want chapter 10", tag, ok)
	}
	if tag, ok := r.FileTag("StoryData/E000X.json"); ok {
		t.Errorf("FileTag of a file of a group without chapter = %+v", tag)
	}
}

// StoryGroup works on rules that were not loaded from files, too.
func TestStoryGroupOfRulesBuiltByHand(t *testing.T) {
	r := &rules.Rules{StoryOrder: rules.StoryOrder{Groups: []rules.StoryOrderGroup{
		{ID: "a", Files: []string{"StoryData/S1B.json"}},
		{ID: "b", Files: []string{"StoryData/S2B.json"}},
	}}}
	if g, ok := r.StoryGroup("StoryData/S2B.json"); !ok || g.ID != "b" {
		t.Errorf("StoryGroup = %+v, %v", g, ok)
	}
}

// A story whose suffix carries a number (the second scene inside a battle) is
// of the chapter of its number like the others.
func TestFileTagOfANumberedStorySuffix(t *testing.T) {
	r := &rules.Rules{}
	for file, want := range map[string]string{"StoryData/S641I2.json": "6", "StoryData/S1034A2.json": "10", "StoryData/S101B.json": "1"} {
		if tag, ok := r.FileTag(file); !ok || tag.Kind != rules.TagChapter || tag.Value != want {
			t.Errorf("FileTag(%s) = %+v, %v, want chapter %s", file, tag, ok, want)
		}
	}
}

func TestStoryOrderGroupFieldsLint(t *testing.T) {
	const head = "groups:\n  - id: a\n    files:\n      - StoryData/S1B.json\n"
	cases := map[string]string{
		"negative chapter":     head + "    chapter: -1\n",
		"chapter not a number": head + "    chapter: one\n",
		"unknown title key":    head + "    title: { file: A.json, id: x, fields: [title], other: 1 }\n",
		"title without file":   head + "    title: { id: x, fields: [title] }\n",
		"title without id":     head + "    title: { file: A.json, fields: [title] }\n",
		"title without fields": head + "    title: { file: A.json, id: x }\n",
		"title file not clean": head + "    title: { file: ../A.json, id: x, fields: [title] }\n",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			root := copyRules(t)
			writeStoryOrder(t, root, content)
			errs := rules.Lint(root)
			if len(errs) != 1 {
				t.Fatalf("errors = %v, want 1", errs)
			}
			if le, ok := errs[0].(rules.LintError); !ok || le.File != rules.StoryOrderFile {
				t.Errorf("error = %#v, want one of %s", errs[0], rules.StoryOrderFile)
			}
		})
	}
}

// The groups of the repo: every main chapter has its number, and every group
// a section and a name.
func TestStoryOrderGroupsOfTheRepo(t *testing.T) {
	r := loadRepo(t)
	for _, g := range r.StoryOrder.Groups {
		if g.Section == "" || (g.Title == nil && g.Label == "") {
			t.Errorf("group %s: section %q, title %v, label %q", g.ID, g.Section, g.Title, g.Label)
		}
	}
	for file, want := range map[string]string{"StoryData/S9991B.json": "10", "StoryData/8D209I.json": "8", "StoryData/S001B.json": "0"} {
		if tag, ok := r.FileTag(file); !ok || tag.Value != want {
			t.Errorf("FileTag(%s) = %+v, %v, want chapter %s", file, tag, ok, want)
		}
	}
}
