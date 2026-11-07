package batch

import (
	"reflect"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/rules"
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
	t.Run("rejected: ZH still has the space after the keyword name", func(t *testing.T) {
		v := Judge(env, nil, "Other.json", "2단계 파열 강도 시즌", "2級破裂強度", "")
		if !v.Rejected {
			t.Fatalf("not rejected: %+v", v.Issues)
		}
		if v.ZH != "2級破裂 強度" {
			t.Errorf("ZH = %q, want the normalized text with the space after the keyword name", v.ZH)
		}
	})
}

// Env.Scripts only saves work: a Judge with the cache gives the verdict of
// one without, including for an entry whose rendering has kana (so that
// ScriptRenderings is not empty) and with a provisional term in play.
func TestJudgeScriptCacheSameVerdict(t *testing.T) {
	const routing = `families:
  - id: other
    nodes: []
    guide: guides/families.md
priority:
  - other
`
	const glossary = "- kr: 하나 협회\n  zh: し協會\n  status: approved\n  check: hard\n" +
		"- kr: 시즌\n  zh: 賽季\n  status: approved\n  check: hard\n"
	const kr = "하나 협회의 두 시즌 전부터 밝은 별"
	r := testRulesWithRouting(t, routing, glossary)
	if len(r.ScriptRenderings("Other.json")) == 0 {
		t.Fatal("no script renderings: the test checks nothing")
	}
	prov := []rules.Term{{KR: "밝은 별", ZH: "明星", Status: rules.StatusApproved, Match: rules.MatchWord, Provisional: true}}
	plain := Env{Rules: r, ForeignNames: []string{"賽局"}}
	cached := plain
	cached.Scripts = rules.NewScriptCache(r)
	for _, zh := range []string{
		"し協會的兩個賽季前起的明星",
		"ハナ協會的兩個賽局前起的亮星",
		"하나協會的兩個賽季前起",
	} {
		for pass := 0; pass < 2; pass++ { // the second pass reads the cache
			want := Judge(plain, prov, "Other.json", kr, zh, "")
			got := Judge(cached, prov, "Other.json", kr, zh, "")
			if !reflect.DeepEqual(got, want) {
				t.Errorf("zh %q pass %d:\ncached = %+v\nplain  = %+v", zh, pass, got, want)
			}
		}
	}
}

// An entry of check: off is general vocabulary: a translation without its
// rendering passes the submit check with no term-missing at all, while its
// forbidden renderings are rejected as ever. The batch still carries it.
func TestJudgeCheckOff(t *testing.T) {
	const glossary = "- kr: 시즌\n  zh: 賽季\n  status: approved\n  check: off\n  forbidden: [季度]\n"
	const kr = "두 시즌 전부터"
	env := Env{Rules: testRulesWith(t, glossary)}
	v := Judge(env, nil, "Other.json", kr, "從兩個賽局前起", "")
	if v.Rejected || hasCode(v.Issues, validate.CodeTermMissing) {
		t.Errorf("missing rendering: %+v, want accepted without term-missing", v)
	}
	if issues := CheckNew(env.Rules, "Other.json", kr, "從兩個賽局前起"); hasCode(issues, validate.CodeTermMissing) {
		t.Errorf("CheckNew = %+v, want no term-missing", issues)
	}
	v = Judge(env, nil, "Other.json", kr, "從兩個季度前起", "")
	if !v.Rejected || !hasCode(v.Issues, validate.CodeForbiddenTerm) {
		t.Errorf("forbidden rendering: %+v, want rejected for forbidden-term", v)
	}
	// The entry is in the glossary of the batch, check: off and all.
	terms := TermsIn(env.Rules, nil, "Other.json", kr)
	if len(terms) != 1 || terms[0].Check != rules.CheckOff {
		t.Errorf("TermsIn = %+v, want the off entry", terms)
	}
}
