package rules_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/rules"
)

// loadGlossary loads the repo rules with text as the glossary.
func loadGlossary(t *testing.T, text string) []rules.Term {
	t.Helper()
	root := copyRules(t)
	if err := os.WriteFile(filepath.Join(root, "glossary", "terms.yaml"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return r.Glossary
}

// jp and en take a string or a list of the forms of one word; an empty value
// (absent, null, "" or []) is no form.
func TestTermFormsFromYAML(t *testing.T) {
	terms := loadGlossary(t, `- kr: 가
  jp: 甲
  en: [Resistance, Resist]
  zh: 甲
  status: approved
- kr: 나
  jp:
  en: ""
  zh: 乙
  status: approved
- kr: 다
  jp: []
  zh: 丙
  status: approved
`)
	want := []struct{ jp, en rules.Forms }{
		{rules.Forms{"甲"}, rules.Forms{"Resistance", "Resist"}},
		{nil, nil},
		{nil, nil},
	}
	if len(terms) != len(want) {
		t.Fatalf("terms = %+v", terms)
	}
	for i, w := range want {
		if !reflect.DeepEqual(terms[i].JP, w.jp) || !reflect.DeepEqual(terms[i].EN, w.en) {
			t.Errorf("%s: jp/en = %#v/%#v, want %#v/%#v", terms[i].KR, terms[i].JP, terms[i].EN, w.jp, w.en)
		}
	}
}

// A mapping is neither a form nor a list of forms.
func TestTermFormsRejectAMapping(t *testing.T) {
	root := copyRules(t)
	edit(t, root, "glossary/terms.yaml", func(s string) string {
		return s + "\n- kr: 백화점\n  en: {a: b}\n  zh: 百貨公司\n  status: approved\n"
	})
	if _, err := rules.Load(root); err == nil || !strings.Contains(err.Error(), "en") {
		t.Errorf("Load = %v, want an error about en", err)
	}
}

// One form is written as the string it always was, so a batch of single-form
// entries is byte for byte what it was; more forms are a list; none is left
// out. The JSON reads back either way.
func TestTermFormsJSON(t *testing.T) {
	for _, tc := range []struct {
		term rules.Term
		want string
	}{
		{rules.Term{KR: "가", JP: rules.Forms{"甲<b>&"}, EN: rules.Forms{"A"}, ZH: "甲"}, `{"kr":"가","jp":"甲<b>&","en":"A","zh":"甲","status":""}`},
		{rules.Term{KR: "가", EN: rules.Forms{"Resistance", "Resist"}, ZH: "甲"}, `{"kr":"가","en":["Resistance","Resist"],"zh":"甲","status":""}`},
		{rules.Term{KR: "가", JP: rules.Forms{}, ZH: "甲"}, `{"kr":"가","zh":"甲","status":""}`},
	} {
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(tc.term); err != nil {
			t.Fatal(err)
		}
		if got := strings.TrimSuffix(buf.String(), "\n"); got != tc.want {
			t.Errorf("JSON = %s\nwant   %s", got, tc.want)
		}
		var back rules.Term
		if err := json.Unmarshal(buf.Bytes(), &back); err != nil {
			t.Fatal(err)
		}
		if len(back.JP) != len(tc.term.JP) || len(back.EN) != len(tc.term.EN) ||
			len(back.JP) > 0 && !reflect.DeepEqual(back.JP, tc.term.JP) || len(back.EN) > 0 && !reflect.DeepEqual(back.EN, tc.term.EN) {
			t.Errorf("read back jp/en = %#v/%#v, want %#v/%#v", back.JP, back.EN, tc.term.JP, tc.term.EN)
		}
	}
	// json.Marshal escapes as it always did.
	data, err := json.Marshal(rules.Term{KR: "가", JP: rules.Forms{"<"}, ZH: "甲"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"jp":"\u003c"`) {
		t.Errorf("json.Marshal = %s, want the < escaped as json.Marshal escapes a string", data)
	}
}

// The forms of a list are each a form of the word: an empty one or one given
// twice is a mistake lint reports.
func TestLintGlossaryForms(t *testing.T) {
	for _, tc := range []struct {
		name, extra string
		want        string
	}{
		{"string", "  en: Resistance\n", ""},
		{"list", "  en: [Resistance, Resist]\n  jp: [耐性]\n", ""},
		{"empty in a list", "  en: [Resistance, \"\"]\n", "en holds an empty form"},
		{"twice", "  jp: [耐性, 耐性]\n", `jp holds "耐性" twice`},
		{"blank", "  en: [\" \"]\n", "en holds an empty form"},
		// EN is compared ignoring case and the number of its last word, as
		// evidence.Holds does: these are one form for every check.
		{"EN case", "  en: [Resist, resist]\n", `en holds "Resist" and "resist"`},
		{"EN plural", "  en: [Resistance, Resistances]\n", `en holds "Resistance" and "Resistances"`},
		{"EN y to ies", "  en: [Ability, abilities]\n", `en holds "Ability" and "abilities"`},
		{"EN other words", "  en: [Resistance, Resist, Physical Resistance]\n", ""},
		{"JP is exact", "  jp: [耐性, 耐性値]\n", ""},
	} {
		root := copyRules(t)
		edit(t, root, "glossary/terms.yaml", func(s string) string {
			return s + "\n- kr: 백화점\n  zh: 百貨公司\n  status: approved\n" + tc.extra
		})
		msg := lintMessages(rules.Lint(root))
		if tc.want == "" {
			if strings.Contains(msg, "백화점") {
				t.Errorf("%s: lint reports\n%s", tc.name, msg)
			}
			continue
		}
		if !strings.Contains(msg, tc.want) {
			t.Errorf("%s: lint output does not mention %q:\n%s", tc.name, tc.want, msg)
		}
	}
}
