package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/cli"
)

// repoRoot is the repo itself: the commands read its real rule files.
const repoRoot = "../.."

func runIn(t *testing.T, run func(*cli.Context, []string) error, root string, jsonOut bool, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := run(&cli.Context{Root: root, JSON: jsonOut, Out: &out}, args)
	return out.String(), err
}

func TestRulesLintCleanRepo(t *testing.T) {
	out, err := runIn(t, runRulesLint, repoRoot, false)
	if err != nil {
		t.Fatalf("lint of the repo's rules: %v\n%s", err, out)
	}
	if !strings.Contains(out, "ok") {
		t.Errorf("output should say the rules are ok:\n%s", out)
	}
}

// cleanRules is a root with a copy of the repo's rule files.
func cleanRules(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, f := range []string{"glossary/terms.yaml", "guides/base.md", "guides/families.md", "guides/routing.yaml"} {
		copyFile(t, filepath.Join(repoRoot, f), filepath.Join(root, f))
	}
	entries, err := os.ReadDir(filepath.Join(repoRoot, "guides/characters"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		src := filepath.Join(repoRoot, "guides/characters", e.Name())
		dst := filepath.Join(root, "guides/characters", e.Name())
		copyFile(t, src, dst)
	}
	return root
}

// brokenRules is a root whose ryoshu.md lacks name_kr.
func brokenRules(t *testing.T) string {
	t.Helper()
	root := cleanRules(t)
	p := filepath.Join(root, "guides/characters/ryoshu.md")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(strings.Replace(string(data), "name_kr: 료슈\n", "", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRulesLintReportsFindings(t *testing.T) {
	out, err := runIn(t, runRulesLint, brokenRules(t), false)
	if !errors.Is(err, cli.ErrFindings) {
		t.Fatalf("err = %v, want cli.ErrFindings", err)
	}
	if !strings.Contains(out, "guides/characters/ryoshu.md") || !strings.Contains(out, "name_kr") {
		t.Errorf("output should name the file and the field:\n%s", out)
	}
}

func TestRulesLintJSON(t *testing.T) {
	out, err := runIn(t, runRulesLint, brokenRules(t), true)
	if !errors.Is(err, cli.ErrFindings) {
		t.Fatalf("err = %v, want cli.ErrFindings", err)
	}
	var got struct {
		OK     bool `json:"ok"`
		Errors []struct {
			File    string `json:"file"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if got.OK || len(got.Errors) != 1 || got.Errors[0].File != "guides/characters/ryoshu.md" || !strings.Contains(got.Errors[0].Message, "name_kr") {
		t.Errorf("report = %+v", got)
	}

	out, err = runIn(t, runRulesLint, repoRoot, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"ok": true`) || !strings.Contains(out, `"errors": []`) {
		t.Errorf("a clean result should be ok with an empty errors array:\n%s", out)
	}
}

func TestRulesLintRejectsArguments(t *testing.T) {
	_, err := runIn(t, runRulesLint, repoRoot, false, "extra")
	if err == nil || errors.Is(err, cli.ErrFindings) {
		t.Errorf("err = %v, want a user error", err)
	}
}

func TestGuideResolve(t *testing.T) {
	out, err := runIn(t, runGuideResolve, repoRoot, true, "--file", "StoryData/S1037B.json", "--speaker", "faust", "--speaker", "dante")
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		File   string   `json:"file"`
		Family string   `json:"family"`
		Guides []string `json:"guides"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	want := []string{"guides/base.md", "guides/families.md", "guides/characters/faust.md", "guides/characters/dante.md"}
	if got.File != "StoryData/S1037B.json" || got.Family != "story-main" || strings.Join(got.Guides, ",") != strings.Join(want, ",") {
		t.Errorf("resolved = %+v", got)
	}

	text, err := runIn(t, runGuideResolve, repoRoot, false, "--file", "AbDlg_Merusault.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"abnormality", "guides/base.md", "guides/characters/meursault.md"} {
		if !strings.Contains(text, want) {
			t.Errorf("text output lacks %q:\n%s", want, text)
		}
	}
}

func TestGuideResolveUsageErrors(t *testing.T) {
	tests := [][]string{
		{},
		{"--file", ""},
		{"--file", "StoryData/S1037B.json", "--speaker", "nobody"},
		{"--file", "StoryData/S1037B.json", "extra"},
		{"--bogus"},
	}
	for _, args := range tests {
		_, err := runIn(t, runGuideResolve, repoRoot, false, args...)
		if err == nil || errors.Is(err, cli.ErrFindings) {
			t.Errorf("args %q: err = %v, want a user error", args, err)
		}
	}
}

func TestCommandsAreRegistered(t *testing.T) {
	names := strings.Join(cli.Commands(), " ")
	for _, want := range []string{"rules lint", "guide resolve", "adjust add", "adjust tidy"} {
		if !strings.Contains(names, want) {
			t.Errorf("command %q is not registered (have: %s)", want, names)
		}
	}
}

// The requeue and rewrite files are checked by "rules lint" too, next to the
// guides, the glossary and the adjustments.
func TestRulesLintChecksRequeueAndRewriteFiles(t *testing.T) {
	root := cleanRules(t)
	writeTestFile(t, root, "rules/requeue/.gitkeep", "")
	writeTestFile(t, root, "rules/rewrite/.gitkeep", "")
	writeTestFile(t, root, "rules/requeue/2026-11-05-good.yaml", "reason: r\nfiles: [Bufs*.json]\n")
	writeTestFile(t, root, "rules/rewrite/2026-11-05-good.yaml", "reason: r\nfiles: [StoryData/**]\npattern: '(a)(b)'\nreplace: '$1 $2'\n")
	if out, err := runIn(t, runRulesLint, root, false); err != nil {
		t.Fatalf("lint of good files: %v\n%s", err, out)
	}

	writeTestFile(t, root, "rules/requeue/2026-11-06-bad.yaml", "reason: ''\nfoo: 1\n")
	writeTestFile(t, root, "rules/rewrite/2026-11-06-bad.yaml", "reason: r\nfiles: [StoryData/**]\npattern: '('\nreplace: ''\n")
	out, err := runIn(t, runRulesLint, root, true)
	if !errors.Is(err, cli.ErrFindings) {
		t.Fatalf("err = %v, want cli.ErrFindings\n%s", err, out)
	}
	var report struct {
		OK     bool
		Errors []struct{ File, Message string }
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	byFile := map[string]string{}
	for _, e := range report.Errors {
		byFile[e.File] += e.Message + "\n"
	}
	if report.OK || len(byFile) != 2 {
		t.Fatalf("report = %+v, want findings in exactly the two bad files", report)
	}
	for file, wants := range map[string][]string{
		"rules/requeue/2026-11-06-bad.yaml": {"reason", "foo", "key or file"},
		"rules/rewrite/2026-11-06-bad.yaml": {"pattern"},
	} {
		for _, want := range wants {
			if !strings.Contains(byFile[file], want) {
				t.Errorf("findings in %s = %q, want one mentioning %q", file, byFile[file], want)
			}
		}
	}
}

// A person can leave a README.md (and the empty .gitkeep that keeps the
// folder in git) in each folder of rule files; any other file that is not
// .yaml is still reported, since it would be silently ignored.
func TestLintAllowsAReadme(t *testing.T) {
	root := cleanRules(t)
	dirs := []string{"rules/adjustments", "rules/requeue", "rules/rewrite"}
	for _, dir := range dirs {
		writeTestFile(t, root, dir+"/.gitkeep", "")
		writeTestFile(t, root, dir+"/README.md", "# 這個資料夾放什麼\n")
	}
	if out, err := runIn(t, runRulesLint, root, false); err != nil {
		t.Fatalf("lint with a README.md in each folder: %v\n%s", err, out)
	}

	for _, dir := range dirs {
		writeTestFile(t, root, dir+"/x.yml", "reason: r\n")
		writeTestFile(t, root, dir+"/notes.md", "# 不是說明檔\n")
	}
	out, err := runIn(t, runRulesLint, root, true)
	if !errors.Is(err, cli.ErrFindings) {
		t.Fatalf("err = %v, want cli.ErrFindings\n%s", err, out)
	}
	var report struct {
		Errors []struct{ File, Message string }
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	got := map[string]bool{}
	for _, e := range report.Errors {
		got[e.File] = true
	}
	want := map[string]bool{}
	for _, dir := range dirs {
		want[dir+"/x.yml"], want[dir+"/notes.md"] = true, true
	}
	if len(got) != len(want) {
		t.Errorf("findings in %v, want exactly %v (no finding for a README.md)", got, want)
	}
	for file := range want {
		if !got[file] {
			t.Errorf("no finding for %s", file)
		}
	}
}

func TestRulesLintChecksAcceptedFile(t *testing.T) {
	root := cleanRules(t)
	writeTestFile(t, root, "rules/accepted.yaml", "- id: \"warning/kana/0123abcd/89ef4567\"\n  reason: ok\n  date: 2026-11-03\n")
	if out, err := runIn(t, runRulesLint, root, false); err != nil {
		t.Fatalf("lint of a good accepted.yaml: %v\n%s", err, out)
	}
	writeTestFile(t, root, "rules/accepted.yaml", "- id: warning/kana/xyz\n  reason: ''\n  date: x\n")
	out, err := runIn(t, runRulesLint, root, false)
	if !errors.Is(err, cli.ErrFindings) || !strings.Contains(out, "rules/accepted.yaml") {
		t.Fatalf("err = %v, want cli.ErrFindings naming rules/accepted.yaml\n%s", err, out)
	}
}

// With the raw worktree present, rules lint warns about a files or not_files
// pattern that matches no KR pack file; the warning does not fail the lint.
// Without the worktree, lint says nothing about it.
func TestRulesLintWarnsOfUnmatchedPatterns(t *testing.T) {
	root := cleanRules(t)
	path := filepath.Join(root, "glossary/terms.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	extra := "\n- kr: 속도테스트\n  zh: 測試\n  status: approved\n  not_files: [\"Nothing_Here*\"]\n"
	if err := os.WriteFile(path, append(data, extra...), 0o644); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, "lct.yaml", "lang_dir: zh-TW\nworktrees: { raw: .work/raw, tw: .work/tw }\n")

	out, err := runIn(t, runRulesLint, root, false)
	if err != nil || strings.Contains(out, "Nothing_Here") {
		t.Fatalf("without the raw worktree: err = %v, output:\n%s", err, out)
	}

	writeTestFile(t, root, ".work/raw/meta/source.json", `{"tag":"raw-20261029","token":"t","created_at":"2026-10-29 00:00:00","fetched_at":"2026-11-01T00:00:00Z","files":{}}`)
	writeTestFile(t, root, ".work/raw/kr/KR_Passives.json", "{}")
	writeTestFile(t, root, ".work/raw/en/EN_Passives.json", "{}")
	writeTestFile(t, root, ".work/raw/jp/JP_Passives.json", "{}")
	out, err = runIn(t, runRulesLint, root, false)
	if err != nil {
		t.Fatalf("a warning must not fail the lint: %v\n%s", err, out)
	}
	if !strings.Contains(out, "warning") || !strings.Contains(out, "Nothing_Here*") || !strings.Contains(out, "ok") {
		t.Errorf("output should be ok and warn of the pattern:\n%s", out)
	}

	out, err = runIn(t, runRulesLint, root, true)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		OK       bool `json:"ok"`
		Warnings []struct {
			File    string `json:"file"`
			Message string `json:"message"`
		} `json:"warnings"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	found := false
	for _, w := range got.Warnings {
		found = found || w.File == "glossary/terms.yaml" && strings.Contains(w.Message, `"속도테스트"`) && strings.Contains(w.Message, "Nothing_Here*")
	}
	if !got.OK || !found {
		t.Errorf("report = %+v", got)
	}
}
