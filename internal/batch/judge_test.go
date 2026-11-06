package batch

import (
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/validate"
)

func TestJudge(t *testing.T) {
	const routing = `families:
  - id: voice
    nodes: ["Warn*"]
    glossary: warn
    guide: guides/families.md
  - id: other
    nodes: []
    guide: guides/families.md
priority:
  - voice
  - other
`
	const glossary = "- kr: 시즌\n  zh: 賽季\n  status: approved\n  check: hard\n"
	const kr = "두 시즌 전부터"
	env := Env{
		Rules:        testRulesWithRouting(t, routing, glossary),
		ForeignNames: []string{"賽局"},
		KeywordNames: []string{"破裂"},
	}
	termMissing := func(t *testing.T, v Verdict, sev validate.Severity) {
		t.Helper()
		for _, is := range v.Issues {
			if is.Code == validate.CodeTermMissing {
				if is.Severity != sev {
					t.Errorf("term-missing severity = %q, want %q", is.Severity, sev)
				}
				return
			}
		}
		t.Errorf("issues = %+v, want a term-missing", v.Issues)
	}

	t.Run("hard entry missing, note silent: rejected without foreign names", func(t *testing.T) {
		v := Judge(env, nil, "Other.json", kr, "從兩個賽局前起", "")
		if !v.Rejected {
			t.Fatal("not rejected")
		}
		termMissing(t, v, validate.SeverityHard)
		if hasCode(v.Issues, CodeForeignName) {
			t.Errorf("issues = %+v, a rejected verdict must not carry foreign-name", v.Issues)
		}
	})
	t.Run("note names the term: accepted with a warning", func(t *testing.T) {
		v := Judge(env, nil, "Other.json", kr, "從兩個賽局前起", "시즌：指賽局")
		if v.Rejected {
			t.Fatalf("rejected: %+v", v.Issues)
		}
		termMissing(t, v, validate.SeverityWarn)
	})
	t.Run("warn family: accepted", func(t *testing.T) {
		v := Judge(env, nil, "Warn.json", kr, "從兩個賽局前起", "")
		if v.Rejected {
			t.Fatalf("rejected: %+v", v.Issues)
		}
		termMissing(t, v, validate.SeverityWarn)
	})
	t.Run("accepted text with a foreign name", func(t *testing.T) {
		v := Judge(env, nil, "Other.json", kr, "從兩個賽局前起賽季", "")
		if v.Rejected {
			t.Fatalf("rejected: %+v", v.Issues)
		}
		if !hasCode(v.Issues, CodeForeignName) {
			t.Errorf("issues = %+v, want foreign-name", v.Issues)
		}
	})
	t.Run("ZH is normalized", func(t *testing.T) {
		v := Judge(env, nil, "Other.json", "2단계 파열 강도", "2級破裂強度", "")
		if v.ZH != "2級破裂 強度" {
			t.Errorf("ZH = %q, want the space after the keyword name", v.ZH)
		}
	})
}
