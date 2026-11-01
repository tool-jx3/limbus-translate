package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/cli"
)

func validateOut(t *testing.T, jsonOut bool, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := runValidate(&cli.Context{JSON: jsonOut, Out: &out}, args)
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
