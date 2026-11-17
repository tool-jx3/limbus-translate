package rules_test

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/rules"
)

func TestRubyCustomizeAt(t *testing.T) {
	r := loadRepo(t)
	yes := [][2]string{
		{"ScenarioModelCodes-AutoCreated.json", "/dataList/630/name"},
		{"StoryData/P10916.json", "/dataList/3/teller"},
		{"PersonalityVoiceDlg/Voice_Heathcliff_KumoThumb_10716.json", "/dataList/6/dlg"},
		{"BattleAnnouncerDlg/Announcer_RienSora_54.json", "/dataList/1/dlg"},
		{"BattleSpeechBubbleDlg-a1c9p3.json", "/dataList/275/dlg"},
	}
	no := [][2]string{
		{"ScenarioModelCodes-AutoCreated.json", "/dataList/630/nickName"},
		{"StoryData/S945B.json", "/dataList/47/title"},
		{"StoryData/S945B.json", "/dataList/47/place"},
		{"StoryData/S945B.json", "/dataList/47/content"},
		{"PersonalityVoiceDlg/Voice_Heathcliff_KumoThumb_10716.json", "/dataList/6/desc"},
		{"RPGSystem/rpg-loc-dialogue-common-a1c10p2.json", "/dataList/9/texts/6/text"},
		{"Skills_personality-x1p1c1.json", "/dataList/3/levelList/0/name"},
		{"ScenarioModelCodes-AutoCreated.json", ""},
	}
	for _, c := range yes {
		if !r.RubyCustomizeAt(c[0], c[1]) {
			t.Errorf("%v not allowed", c)
		}
	}
	for _, c := range no {
		if r.RubyCustomizeAt(c[0], c[1]) {
			t.Errorf("%v allowed", c)
		}
	}
	st, err := r.RubyStyle()
	if err != nil || st.Scale != 0.5 || st.VOffset != "1em" {
		t.Errorf("RubyStyle = %v, %v", st, err)
	}
}

func writeRuby(t *testing.T, root, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rules.RubyCustomizeFile))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRubyCustomizeMissingFile(t *testing.T) {
	root := copyRules(t)
	r, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if r.RubyCustomizeAt("StoryData/P1.json", "/dataList/0/teller") {
		t.Error("allowed without the file")
	}
	if errs := rules.Lint(root); len(errs) != 0 {
		t.Errorf("lint without the file: %v", errs)
	}
	if w := rules.LintRubyCustomizeWarnings(root, []string{"Skills.json"}, nil); len(w) != 0 {
		t.Errorf("warnings without the file: %v", w)
	}
}

func TestRubyCustomizeLint(t *testing.T) {
	const good = "size: 50%\nvoffset: 1em\nfields:\n  - files: ['StoryData/**']\n    field: teller\n"
	cases := map[string]string{
		"unknown top key":   good + "extra: 1\n",
		"unknown entry key": good + "  - files: ['A*']\n    field: x\n    other: 1\n",
		"missing files":     good + "  - field: x\n",
		"missing field":     good + "  - files: ['A*']\n",
		"bad size":          "size: 50\nvoffset: 1em\nfields: []\n",
		"bad pattern":       good + "  - files: ['[']\n    field: x\n",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			root := copyRules(t)
			writeRuby(t, root, content)
			errs := rules.Lint(root)
			if len(errs) != 1 {
				t.Fatalf("errors = %v, want 1", errs)
			}
			if le, ok := errs[0].(rules.LintError); !ok || le.File != rules.RubyCustomizeFile {
				t.Errorf("error = %#v, want one of %s", errs[0], rules.RubyCustomizeFile)
			}
		})
	}
	root := copyRules(t)
	writeRuby(t, root, good)
	if errs := rules.Lint(root); len(errs) != 0 {
		t.Errorf("good file: %v", errs)
	}
}

func TestRubyCustomizeWarnings(t *testing.T) {
	root := copyRules(t)
	data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(rules.RubyCustomizeFile)))
	if err != nil {
		t.Fatal(err)
	}
	writeRuby(t, root, string(data))
	if w := rules.LintRubyCustomizeWarnings(root, []string{"Skills.json"}, nil); len(w) != 5 {
		t.Errorf("warnings = %v, want 5", w)
	}
	shipped := map[string]string{
		"StoryData/P1.json": "teller", "ScenarioModelCodes-A.json": "name", "PersonalityVoiceDlg/a.json": "dlg",
		"BattleAnnouncerDlg/a.json": "dlg", "BattleSpeechBubbleDlg-x.json": "dlg",
	}
	krFiles := slices.Sorted(maps.Keys(shipped))
	has := func(file, field string) bool { return shipped[file] == field }
	if w := rules.LintRubyCustomizeWarnings(root, krFiles, has); len(w) != 0 {
		t.Errorf("warnings = %v, want none", w)
	}
	if w := rules.LintRubyCustomizeWarnings(root, krFiles, nil); len(w) != 0 {
		t.Errorf("warnings without hasField = %v, want none", w)
	}
}

// An entry whose files match KR files none of which has a unit of its field
// (a misspelled field) allows nothing: it gets a warning that names it.
func TestRubyCustomizeWarnsOfAFieldNoUnitHas(t *testing.T) {
	root := copyRules(t)
	writeRuby(t, root, "size: 50%\nvoffset: 1em\nfields:\n"+
		"  - files: ['StoryData/**']\n    field: teller\n"+
		"  - files: ['StoryData/**']\n    field: tellr\n")
	krFiles := []string{"StoryData/P1.json", "StoryData/P2.json"}
	has := func(file, field string) bool { return file == "StoryData/P2.json" && field == "teller" }
	w := rules.LintRubyCustomizeWarnings(root, krFiles, has)
	if len(w) != 1 || w[0].File != rules.RubyCustomizeFile || !strings.Contains(w[0].Message, "fields entry 2") || !strings.Contains(w[0].Message, `"tellr"`) {
		t.Errorf("warnings = %v, want one of fields entry 2 that names tellr", w)
	}
	if w := rules.LintRubyCustomizeWarnings(root, krFiles, nil); len(w) != 0 {
		t.Errorf("warnings without hasField = %v, want none: the files match", w)
	}
}
