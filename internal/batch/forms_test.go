package batch

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/rules"
)

// formsBatch cuts a batch of one unit whose KR holds 공격, an entry with one
// form of each language, and 위력, an entry with two EN forms.
func formsBatch(t *testing.T) (*Batch, []byte) {
	t.Helper()
	const file = "Other.json"
	kr := map[string]string{file: `{"dataList":[{"id":0,"content":"공격 위력"}]}`}
	en := map[string]string{file: `{"dataList":[{"id":0,"content":"Attack Power"}]}`}
	jp := map[string]string{file: `{"dataList":[{"id":0,"content":"攻撃 威力"}]}`}
	env := newEnv(t, kr, en, jp)
	env.Rules = testRulesWith(t, "- kr: 공격\n  jp: 攻撃\n  en: Attack\n  zh: 攻擊\n  status: approved\n"+
		"- kr: 위력\n  jp: [威力]\n  en: [Power, Potency]\n  zh: 威力\n  status: approved\n")
	seed(t, env, items(file, 0, 1)...)
	b := mustNext(t, env, NextOptions{})
	data, err := EncodeFile(b)
	if err != nil {
		t.Fatal(err)
	}
	return b, data
}

// An entry with one form of jp and en is written as it always was, a string
// each; one with more forms writes the list. A list of one form is one form.
func TestBatchGlossaryForms(t *testing.T) {
	b, data := formsBatch(t)
	text := string(data)
	single := "    {\n      \"kr\": \"공격\",\n      \"jp\": \"攻撃\",\n      \"en\": \"Attack\",\n      \"zh\": \"攻擊\",\n      \"status\": \"approved\",\n      \"units\": [\n        \"Other.json|/dataList/0/content\"\n      ]\n    }"
	if !strings.Contains(text, single) {
		t.Errorf("the single-form entry is not written as before:\n%s", text)
	}
	multi := "      \"kr\": \"위력\",\n      \"jp\": \"威力\",\n      \"en\": [\n        \"Power\",\n        \"Potency\"\n      ],\n      \"zh\": \"威力\","
	if !strings.Contains(text, multi) {
		t.Errorf("the entry of two EN forms does not write them as a list:\n%s", text)
	}
	// The batch reads back from its file.
	var back Batch
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(&back, b) {
		t.Errorf("round trip changed the batch:\n got  %+v\n want %+v", back, *b)
	}
	if got := back.Glossary[1].EN; !reflect.DeepEqual(got, rules.Forms{"Power", "Potency"}) {
		t.Errorf("read back en = %#v", got)
	}
}

// The schema of a glossary entry takes jp and en as a string or a list of
// strings.
func TestBatchSchemaTakesForms(t *testing.T) {
	_, data := formsBatch(t)
	var top struct {
		Glossary []map[string]any `json:"glossary"`
	}
	if err := json.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	schema := loadSchema(t, "batch.schema.json")
	props, _ := schemaProps(t, schema)
	glossProps, _ := schemaProps(t, props["glossary"]["items"].(map[string]any))
	for _, g := range top.Glossary {
		for _, lang := range []string{"jp", "en"} {
			if v, ok := g[lang]; ok && !schemaAllows(glossProps[lang], v) {
				t.Errorf("%s: the schema of %s = %v rejects %#v", g["kr"], lang, glossProps[lang], v)
			}
		}
	}
	if schemaAllows(glossProps["en"], 3.0) || schemaAllows(glossProps["en"], []any{1.0}) {
		t.Errorf("the schema of en = %v takes a number", glossProps["en"])
	}
}

// schemaAllows reports whether v is of a type prop takes: its type, or one
// of oneOf, with the items of an array.
func schemaAllows(prop map[string]any, v any) bool {
	if alts, ok := prop["oneOf"].([]any); ok {
		for _, a := range alts {
			if schemaAllows(a.(map[string]any), v) {
				return true
			}
		}
		return false
	}
	switch prop["type"] {
	case "string":
		_, ok := v.(string)
		return ok
	case "array":
		list, ok := v.([]any)
		if !ok {
			return false
		}
		items, _ := prop["items"].(map[string]any)
		for _, x := range list {
			if items != nil && !schemaAllows(items, x) {
				return false
			}
		}
		return true
	}
	return false
}
