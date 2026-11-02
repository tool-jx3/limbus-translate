package names_test

import (
	"reflect"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/names"
)

func doc(t *testing.T, src string) *jsonx.Node {
	t.Helper()
	n, err := jsonx.Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse %s: %v", src, err)
	}
	return n
}

func TestKeywordTable(t *testing.T) {
	docs := map[string]*jsonx.Node{
		// two Bufs fixtures that disagree about Burn
		"Bufs-a1c9p1.json": doc(t, `{"dataList":[
			{"id":"Burn","name":"灼燒"},
			{"id":"Bleed","name":"流血"}]}`),
		"Bufs.json": doc(t, `{"dataList":[
			{"id":"Burn","name":"燒傷"},
			{"id":"Burn","name":"重複的燒傷"},
			{"id":"Sinking","name":""},
			{"id":"Poise","name":"呼吸法"}]}`),
		"BattleKeywords.json": doc(t, `{"dataList":[
			{"id":"Sinking","name":"沉淪"},
			{"id":"Poise","name":"呼吸"},
			{"id":101,"name":"數字編號"},
			{"id":"NoName"},
			{"name":"沒有編號"},
			{"id":"NotText","name":7},
			{"id":null,"name":"空編號"}]}`),
		// files of other families do not feed the table
		"Skills_personality-01.json": doc(t, `{"dataList":[{"id":"Burn","name":"不該採用"},{"id":"Skill","name":"技能"}]}`),
		"StoryData/S001A.json":       doc(t, `{"dataList":[{"id":"Story","name":"不該採用"}]}`),
		// a file with no dataList, and a nil document, are tolerated
		"Bufs-empty.json": doc(t, `{"other":[]}`),
		"Bufs-nil.json":   nil,
	}
	want := map[string]string{
		"Burn":    "燒傷", // Bufs.json comes before Bufs-a1c9p1.json
		"Bleed":   "流血", // only the later file has it, so it fills the gap
		"Sinking": "沉淪", // the empty name in Bufs.json is a gap
		"Poise":   "呼吸", // BattleKeywords* is read before Bufs*
		"101":     "數字編號",
	}
	// the map order of docs must not matter
	for i := 0; i < 20; i++ {
		if got := names.KeywordTable(docs); !reflect.DeepEqual(got, want) {
			t.Fatalf("run %d: KeywordTable = %v, want %v", i, got, want)
		}
	}
}

func TestKeywordTableEmpty(t *testing.T) {
	for _, docs := range []map[string]*jsonx.Node{nil, {}} {
		got := names.KeywordTable(docs)
		if got == nil || len(got) != 0 {
			t.Errorf("KeywordTable(%v) = %#v, want an empty, non-nil map", docs, got)
		}
	}
}

func TestKeywordTableKeepsNumberLiteral(t *testing.T) {
	docs := map[string]*jsonx.Node{
		"Bufs.json": doc(t, `{"dataList":[{"id":10,"name":"甲"},{"id":1.50,"name":"乙"}]}`),
	}
	got := names.KeywordTable(docs)
	if got["10"] != "甲" || got["1.50"] != "乙" {
		t.Errorf("KeywordTable = %v", got)
	}
}

func TestSpeakerTable(t *testing.T) {
	d := doc(t, `{"dataList":[
		{"id":"이상","name":"李箱","nickName":"1號囚犯"},
		{"id":"단테","name":"但丁"},
		{"id":"이상","name":"重複"},
		{"id":"무명","name":""},
		{"id":"숫자","name":7},
		{"name":"沒有編號"},
		{"id":12,"name":"十二"}]}`)
	want := map[string]string{"이상": "李箱", "단테": "但丁", "12": "十二"}
	if got := names.SpeakerTable(d); !reflect.DeepEqual(got, want) {
		t.Errorf("SpeakerTable = %v, want %v", got, want)
	}
	for _, empty := range []*jsonx.Node{nil, doc(t, `{}`), doc(t, `{"dataList":{}}`), doc(t, `[]`)} {
		if got := names.SpeakerTable(empty); got == nil || len(got) != 0 {
			t.Errorf("SpeakerTable(%v) = %#v, want an empty, non-nil map", empty, got)
		}
	}
}

func TestIsKeywordFile(t *testing.T) {
	for file, want := range map[string]bool{
		"Bufs.json":                      true,
		"Bufs_Enemy.json":                true,
		"BattleKeywords.json":            true,
		"Sub/BattleKeywords-extra.json":  true,
		"Skills.json":                    false,
		"StoryData/Bufs-story.json":      true, // the file name decides, not the directory
		"AbnormalityBufs.json":           false,
		"ScenarioModelCodes-AutoCreated": false,
	} {
		if got := names.IsKeywordFile(file); got != want {
			t.Errorf("IsKeywordFile(%q) = %v, want %v", file, got, want)
		}
	}
}

func TestHasHangul(t *testing.T) {
	for s, want := range map[string]bool{
		"파열": true, "破裂": false, "": false, "Burst": false, "[Burst] 한": true, "ㄱ": true, "破裂ㅋ": true,
	} {
		if got := names.HasHangul(s); got != want {
			t.Errorf("HasHangul(%q) = %v, want %v", s, got, want)
		}
	}
}
