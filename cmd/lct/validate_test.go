package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/cli"
)

func validateOut(t *testing.T, jsonOut bool, args ...string) (string, error) {
	t.Helper()
	// validate reads the glossary, so it needs the repo root.
	return validateIn(t, "../..", jsonOut, args...)
}

func validateIn(t *testing.T, root string, jsonOut bool, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := runValidate(&cli.Context{Root: root, JSON: jsonOut, Out: &out}, args)
	return out.String(), err
}

func TestValidateReportsNormalizedTextAndOK(t *testing.T) {
	out, err := validateOut(t, false, "--kr", "[Bleed] 위력", "--zh", "增加[Bleed]強度")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "增加 [Bleed] 強度") {
		t.Errorf("output should show the normalized text:\n%s", out)
	}
	if !strings.Contains(out, "ok") {
		t.Errorf("output should say the translation is ok:\n%s", out)
	}
}

func TestValidateHardIssueIsFindings(t *testing.T) {
	out, err := validateOut(t, false, "--kr", "[Bleed] 위력", "--zh", "流血強度")
	if !errors.Is(err, cli.ErrFindings) {
		t.Fatalf("err = %v, want cli.ErrFindings", err)
	}
	if !strings.Contains(out, "id-mismatch") || !strings.Contains(out, "hard") {
		t.Errorf("output should list the hard id-mismatch:\n%s", out)
	}
}

func TestValidateEnforcesGlossary(t *testing.T) {
	// 패키지 is 組合包 in the glossary, and 禮包 is forbidden for it.
	out, err := validateOut(t, false, "--kr", "패키지 구매", "--zh", "購買禮包")
	if !errors.Is(err, cli.ErrFindings) {
		t.Fatalf("err = %v, want cli.ErrFindings", err)
	}
	if !strings.Contains(out, "forbidden-term") || !strings.Contains(out, "禮包") {
		t.Errorf("output should list the forbidden term:\n%s", out)
	}

	// the adopted rendering passes
	if out, err := validateOut(t, false, "--kr", "패키지 구매", "--zh", "購買組合包"); err != nil {
		t.Errorf("the approved rendering must pass: %v\n%s", err, out)
	}
	// a forbidden rendering only counts when the KR contains the term
	if out, err := validateOut(t, false, "--kr", "구매", "--zh", "購買禮包"); err != nil {
		t.Errorf("a term absent from KR must not be enforced: %v\n%s", err, out)
	}
	// regular-expression entries work through the same path
	if _, err := validateOut(t, false, "--kr", "시즌 8", "--zh", "第8季"); !errors.Is(err, cli.ErrFindings) {
		t.Errorf("第8季 should be forbidden for 시즌: err = %v", err)
	}
}

// 신부 is 牧師 in the files of chapter 10 and 神父 in the others (the glossary's
// files); --file says which file the text is from.
func TestValidateEnforcesTermsByFile(t *testing.T) {
	out, err := validateOut(t, false, "--kr", "신부가 왔다", "--zh", "神父來了", "--file", "StoryData/S1034B.json")
	if !errors.Is(err, cli.ErrFindings) || !strings.Contains(out, "forbidden-term") {
		t.Errorf("神父 in a file of chapter 10: err = %v, want a forbidden-term\n%s", err, out)
	}
	if out, err := validateOut(t, false, "--kr", "신부가 왔다", "--zh", "牧師來了", "--file", "StoryData/S1034B.json"); err != nil {
		t.Errorf("牧師 in a file of chapter 10: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"--file", "StoryData/S725B.json"},
		{}, // no file: only the terms for every file
	} {
		args = append([]string{"--kr", "신부가 왔다", "--zh", "神父來了"}, args...)
		if out, err := validateOut(t, false, args...); err != nil {
			t.Errorf("神父 with %q: %v\n%s", args[4:], err, out)
		}
	}
}

func TestValidateIgnoresCandidateTerms(t *testing.T) {
	// A glossary of its own, so that approving a candidate in the real one
	// does not break this: 전용 파편 is only a candidate, and 人格碎片 is on its
	// forbidden list, so it must not be enforced yet.
	root := t.TempDir()
	copyDir(t, filepath.Join(repoRoot, "guides"), filepath.Join(root, "guides"))
	writeTestFile(t, root, "glossary/terms.yaml", `- kr: 자아 파편
  zh: 自我碎片
  status: approved
  forbidden: [人格碎片]
- kr: 전용 파편
  zh: 專屬碎片
  status: candidate
  forbidden: [人格碎片]
`)
	if out, err := validateIn(t, root, false, "--kr", "전용 파편", "--zh", "專用人格碎片"); err != nil {
		t.Errorf("a candidate term must not bind: %v\n%s", err, out)
	}
	// the approved 자아 파편 forbids the same rendering, and does bind
	if _, err := validateIn(t, root, false, "--kr", "자아 파편", "--zh", "人格碎片"); !errors.Is(err, cli.ErrFindings) {
		t.Errorf("an approved term must bind: err = %v", err)
	}
}

func TestValidateNeedsRules(t *testing.T) {
	var out bytes.Buffer
	err := runValidate(&cli.Context{Root: t.TempDir(), Out: &out}, []string{"--kr", "가", "--zh", "好"})
	if err == nil || errors.Is(err, cli.ErrFindings) {
		t.Errorf("err = %v, want a system error about the missing rules", err)
	}
}

func TestValidateWarningsOnlyIsSuccess(t *testing.T) {
	out, err := validateOut(t, false, "--kr", "가", "--zh", "效果,增加")
	if err != nil {
		t.Fatalf("a warning must not fail the command: %v", err)
	}
	if !strings.Contains(out, "halfwidth-punct") || !strings.Contains(out, "warn") {
		t.Errorf("output should list the warning:\n%s", out)
	}
}

func TestValidateJSON(t *testing.T) {
	out, err := validateOut(t, true, "--kr", "가", "--zh", "&lt;b&gt;加&lt;/b&gt;")
	if !errors.Is(err, cli.ErrFindings) {
		t.Fatalf("err = %v, want cli.ErrFindings (the tags are not in KR)", err)
	}
	var got struct {
		Normalized string `json:"normalized"`
		OK         bool   `json:"ok"`
		Issues     []struct {
			Code     string `json:"code"`
			Severity string `json:"severity"`
			Class    string `json:"class"`
			Message  string `json:"message"`
		} `json:"issues"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if got.Normalized != "<b>加</b>" || got.OK {
		t.Errorf("normalized = %q, ok = %v", got.Normalized, got.OK)
	}
	if len(got.Issues) != 1 || got.Issues[0].Code != "tag-mismatch" || got.Issues[0].Severity != "hard" || got.Issues[0].Class != "structural" {
		t.Errorf("issues = %+v", got.Issues)
	}
	if strings.Contains(out, `\`+`u003c`) {
		t.Errorf("JSON must not escape < and >:\n%s", out)
	}
}

func TestValidateJSONHasEmptyIssueList(t *testing.T) {
	out, err := validateOut(t, true, "--kr", "가", "--zh", "好")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"issues": []`) {
		t.Errorf("a clean result should have an empty issues array, not null:\n%s", out)
	}
}

func TestValidateUsageErrors(t *testing.T) {
	tests := [][]string{
		{},
		{"--kr", "가"},
		{"--zh", "好"},
		{"--kr", "가", "--zh", "好", "extra"},
		{"--bogus"},
	}
	for _, args := range tests {
		_, err := validateOut(t, false, args...)
		if err == nil || errors.Is(err, cli.ErrFindings) {
			t.Errorf("args %q: err = %v, want a user error", args, err)
		}
	}
}

func TestValidateEmptyKRIsAllowed(t *testing.T) {
	if _, err := validateOut(t, false, "--kr", "", "--zh", ""); err != nil {
		t.Errorf("empty KR and zh: %v", err)
	}
}

// The file of the text decides whether half-width brackets are reported.
func TestValidateFileDecidesTheBracketWarning(t *testing.T) {
	const kr, zh = "퀘스트", "(內部)任務"
	out, err := validateOut(t, false, "--kr", kr, "--zh", zh, "--file", "RPGSystem/rpg-loc-quest-floor-3.json")
	if err != nil || !strings.Contains(out, "halfwidth-bracket") {
		t.Errorf("a quest label: output %q, error %v, want the halfwidth-bracket warning", out, err)
	}
	for _, args := range [][]string{
		{"--kr", kr, "--zh", zh, "--file", "RPGSystem/rpg-loc-dialogue-floor-3.json"},
		{"--kr", kr, "--zh", zh},
	} {
		if out, err := validateOut(t, false, args...); err != nil || strings.Contains(out, "halfwidth-bracket") {
			t.Errorf("%v: output %q, error %v, want no halfwidth-bracket warning", args, out, err)
		}
	}
}
