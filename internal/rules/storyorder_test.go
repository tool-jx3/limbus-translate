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
