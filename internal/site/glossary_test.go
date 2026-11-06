package site

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/itemid"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/scope"
	"github.com/tool-jx3/limbus-translate/internal/suggest"
)

func TestGlossaryFiles(t *testing.T) {
	r := &rules.Rules{
		Routing:  rules.Routing{Families: []rules.FamilyRule{{ID: "story", Nodes: []string{"StoryData/"}}, {ID: "other"}}},
		Glossary: []rules.Term{{KR: "대상", ZH: "目標", Also: []string{"對象"}, Status: "approved"}},
	}
	id := itemid.GlossaryEntry("대상", "目標")
	long := strings.Repeat("字", 500)
	pad := strings.Repeat("甲", 50)
	krPad := strings.Repeat("가", 50)
	rep := scope.Empty()
	rep.Entries = []scope.Entry{{ID: id, KR: "대상", ZH: "目標"}}
	rep.Units[id] = []scope.EntryUnit{
		{Key: "Skills.json#/a", File: "Skills.json", KR: "대상을 <b>쓴다</b>", ZH: "使用<b>對象</b>", Origin: "claude", Note: "n1", Used: false},
		{Key: "StoryData/S1.json#/b", File: "StoryData/S1.json", KR: krPad + "대상" + krPad, ZH: pad + "目標" + pad, Origin: "paratranz", Used: true},
		{Key: "Skills.json#/c", File: "Skills.json", KR: "대상" + long, ZH: long, Origin: "human"},
		{Key: "Skills.json#/d", File: "Skills.json", KR: "대상", ZH: "好", Origin: "claude", Used: true},
	}
	acc := map[string]*Confirmed{
		itemid.UnitWarning("term-missing", "대상", "目標", "대상을 <b>쓴다</b>", "使用<b>對象</b>"): {Reason: "ok", Date: "2026-11-01"},
	}
	files := buildGlossaryFiles(r, rep, acc)
	g, ok := files[strings.ReplaceAll(strings.TrimPrefix(id, "glossary/"), "/", "-")+".json"]
	if !ok || len(files) != 1 {
		t.Fatalf("files = %v, want one named after the id", files)
	}
	if g.ID != id || len(g.Missing) != 2 || len(g.Used) != 2 {
		t.Fatalf("file = %+v", g)
	}
	m := g.Missing[0]
	if m.KR != "대상을 <b>쓴다</b>" || m.ZH != "使用<b>對象</b>" || m.Note != "n1" || m.Family != "other" || m.Top != "Skills" {
		t.Errorf("missing[0] = %+v, want the original text", m)
	}
	if m.ID == "" || m.Accepted == nil || m.Accepted.Reason != "ok" {
		t.Errorf("missing[0] id/accepted = %q %+v", m.ID, m.Accepted)
	}
	m2 := g.Missing[1]
	if want := strings.Repeat("字", 400) + "…"; m2.ZH != want || !strings.HasPrefix(m2.KR, "대상") || m2.Accepted != nil || m2.ID == "" {
		t.Errorf("missing[1] zh len %d, accepted %v", len([]rune(m2.ZH)), m2.Accepted)
	}
	if len([]rune(m2.KR)) != 401 {
		t.Errorf("missing[1] kr runes = %d, want 401", len([]rune(m2.KR)))
	}
	u := g.Used[0]
	if u.ID != "" || u.Accepted != nil || u.Family != "story" || u.Top != "StoryData/" || u.Sub != "StoryData/S" {
		t.Errorf("used[0] = %+v", u)
	}
	if want := "…" + strings.Repeat("가", 30) + "대상" + strings.Repeat("가", 30) + "…"; u.KR != want {
		t.Errorf("used kr = %q, want %q", u.KR, want)
	}
	if want := "…" + strings.Repeat("甲", 30) + "目標" + strings.Repeat("甲", 30) + "…"; u.ZH != want {
		t.Errorf("used zh = %q, want %q", u.ZH, want)
	}
	if g.Used[1].KR != "대상" || g.Used[1].ZH != "好" {
		t.Errorf("short used unit changed: %+v", g.Used[1])
	}

	// A translation with an also rendering is centred on it; tags are stripped.
	rep.Units[id] = []scope.EntryUnit{{Key: "k", File: "Skills.json", KR: "<b>대상</b>", ZH: "使用<b>對象</b>", Used: true}}
	got := buildGlossaryFiles(r, rep, nil)
	if x := got[strings.ReplaceAll(strings.TrimPrefix(id, "glossary/"), "/", "-")+".json"].Used[0]; x.KR != "대상" || x.ZH != "使用對象" {
		t.Errorf("used = %+v", x)
	}
}

func TestExportWritesGlossary(t *testing.T) {
	in := exportInputs(t)
	dir := t.TempDir()
	res, err := Export(in, Meta{}, dir)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "glossary", "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var idx struct {
		Thresholds scope.Thresholds `json:"thresholds"`
		Entries    []scope.Entry    `json:"entries"`
	}
	if err := json.Unmarshal(data, &idx); err != nil || idx.Thresholds.HardMinN == 0 || idx.Entries == nil {
		t.Errorf("glossary/index.json = %s (%v)", data, err)
	}
	if res.GlossaryBytes < int64(len(data)) || res.GlossaryBytes > res.Bytes {
		t.Errorf("GlossaryBytes = %d, Bytes = %d", res.GlossaryBytes, res.Bytes)
	}
	_ = suggest.Confirmed{}
}
