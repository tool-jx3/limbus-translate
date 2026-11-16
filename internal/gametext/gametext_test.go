package gametext

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

func openPack(t *testing.T, files map[string]string) *pack.Pack {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p, err := pack.Open(dir, "KR_")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func rec(kr, zh, origin string) store.Record {
	return store.Record{ZH: zh, KRSha: units.KRSha(kr), Origin: origin}
}

func TestInputBufs(t *testing.T) {
	p := openPack(t, map[string]string{
		"KR_Bufs.json":           `{"dataList":[{"id":"A","name":"가"},{"id":"B","name":"나","excludeAutoLink":true},{"id":"C","name":"다"},{"id":3,"name":"라"}]}`,
		"KR_Bufs-x.json":         `{"dataList":[{"id":"D","name":"마"}]}`,
		"KR_BattleKeywords.json": `{"dataList":[{"id":"Z","name":"자"}]}`,
	})
	records := map[string]store.Record{
		units.Key("Bufs.json", "/dataList/0/name"):   rec("가", "甲", store.OriginClaude),
		units.Key("Bufs.json", "/dataList/2/name"):   rec("다", "다", store.OriginCopy),
		units.Key("Bufs-x.json", "/dataList/0/name"): rec("other", "乙", store.OriginClaude),
	}
	in, err := Input(p, UsedText(records, nil))
	if err != nil {
		t.Fatal(err)
	}
	want := []validate.GameBuf{
		{ID: "A", KR: "가", ZH: "甲"},
		{ID: "B", KR: "나", ZH: "나", ExcludeAutoLink: true},
		{ID: "C", KR: "다", ZH: "다"},
		{ID: "3", KR: "라", ZH: "라"},
		{ID: "D", KR: "마", ZH: "마"},
	}
	if !reflect.DeepEqual(in.Bufs, want) {
		t.Fatalf("Bufs = %+v\nwant %+v", in.Bufs, want)
	}
}

func TestInputResonance(t *testing.T) {
	p := openPack(t, map[string]string{
		"KR_BattleUIText.json": `{"dataList":[{"id":"resonance","content":"공명"},{"id":"perfect_resonance","content":"완전공명"}]}`,
	})
	records := map[string]store.Record{
		units.Key("BattleUIText.json", "/dataList/0/content"): rec("공명", "共鳴", store.OriginClaude),
	}
	in, err := Input(p, UsedText(records, nil))
	if err != nil {
		t.Fatal(err)
	}
	if in.Resonance != "共鳴" || in.PerfectResonance != "완전공명" {
		t.Fatalf("got %q, %q", in.Resonance, in.PerfectResonance)
	}
	none := openPack(t, map[string]string{"KR_Bufs.json": `{"dataList":[]}`})
	in, err = Input(none, UsedText(nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	if in.Resonance != "" || in.PerfectResonance != "" {
		t.Fatalf("got %q, %q", in.Resonance, in.PerfectResonance)
	}
}

const skillsDoc = `{"dataList":[{"levelList":[
 {"level":1,"desc":"가","coinlist":[{"coindescs":[{"desc":"c1"}]}]},
 {"level":3,"desc":"가2","coinlist":[{"coindescs":[{"desc":"c3"}]}]},
 {"level":4,"desc":"가3","coinlist":[{"coindescs":[{"desc":"c4"}]},{"coindescs":[{"desc":"c4b"}]}]}
]}]}`

func skillRecords() map[string]store.Record {
	r := map[string]store.Record{}
	add := func(ptr, kr string) {
		r[units.Key("Skills.json", ptr)] = rec(kr, "zh-"+kr, store.OriginClaude)
	}
	add("/dataList/0/levelList/0/desc", "가")
	add("/dataList/0/levelList/1/desc", "가2")
	add("/dataList/0/levelList/2/desc", "가3")
	add("/dataList/0/levelList/0/coinlist/0/coindescs/0/desc", "c1")
	add("/dataList/0/levelList/1/coinlist/0/coindescs/0/desc", "c3")
	add("/dataList/0/levelList/2/coinlist/0/coindescs/0/desc", "c4")
	add("/dataList/0/levelList/2/coinlist/1/coindescs/0/desc", "c4b")
	return r
}

func TestLevelPairs(t *testing.T) {
	p := openPack(t, map[string]string{
		"KR_Skills.json":             skillsDoc,
		"RPGSystem/KR_Skills-x.json": skillsDoc,
		"KR_SkillTag.json":           skillsDoc,
	})
	k := func(ptr string) string { return units.Key("Skills.json", ptr) }
	in, err := Input(p, UsedText(skillRecords(), nil))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]validate.LevelText{
		k("/dataList/0/levelList/1/desc"):                        {Level: 1, KR: "가", ZH: "zh-가"},
		k("/dataList/0/levelList/2/desc"):                        {Level: 3, KR: "가2", ZH: "zh-가2"},
		k("/dataList/0/levelList/1/coinlist/0/coindescs/0/desc"): {Level: 1, KR: "c1", ZH: "zh-c1"},
		k("/dataList/0/levelList/2/coinlist/0/coindescs/0/desc"): {Level: 3, KR: "c3", ZH: "zh-c3"},
	}
	if !reflect.DeepEqual(in.PrevLevel, want) {
		t.Fatalf("PrevLevel = %+v\nwant %+v", in.PrevLevel, want)
	}

	// The level 1 desc without a usable record: level 3 desc has no entry.
	r := skillRecords()
	delete(r, k("/dataList/0/levelList/0/desc"))
	in, err = Input(p, UsedText(r, nil))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := in.PrevLevel[k("/dataList/0/levelList/1/desc")]; ok {
		t.Fatal("entry without a record of the lower level")
	}
	if _, ok := in.PrevLevel[k("/dataList/0/levelList/2/desc")]; !ok {
		t.Fatal("level 4 entry lost")
	}
}

func TestLevelPairsSkipped(t *testing.T) {
	for name, levels := range map[string]string{
		"repeat":    `[{"level":1,"desc":"가"},{"level":1,"desc":"나"}]`,
		"not-int":   `[{"level":"1","desc":"가"},{"level":2,"desc":"나"}]`,
		"fractions": `[{"level":1.5,"desc":"가"},{"level":2,"desc":"나"}]`,
	} {
		t.Run(name, func(t *testing.T) {
			p := openPack(t, map[string]string{"KR_Skills.json": `{"dataList":[{"levelList":` + levels + `}]}`})
			records := map[string]store.Record{}
			for i, kr := range []string{"가", "나"} {
				records[units.Key("Skills.json", "/dataList/0/levelList/"+string(rune('0'+i))+"/desc")] = rec(kr, "zh", store.OriginClaude)
			}
			in, err := Input(p, UsedText(records, nil))
			if err != nil {
				t.Fatal(err)
			}
			if len(in.PrevLevel) != 0 {
				t.Fatalf("PrevLevel = %+v", in.PrevLevel)
			}
		})
	}
}

func TestUsedText(t *testing.T) {
	records := map[string]store.Record{
		units.Key("A.json", "/p"): rec("kr", "zh", store.OriginClaude),
		units.Key("A.json", "/o"): rec("old", "zh2", store.OriginClaude),
		units.Key("A.json", "/c"): rec("kr", "kr", store.OriginCopy),
		units.Key("F.json", "/o"): rec("old", "zh3", store.OriginHuman),
		units.Key("F.json", "/c"): rec("kr", "kr", store.OriginCopy),
	}
	text := UsedText(records, []string{"F.json"})
	for _, c := range []struct {
		file, ptr, kr, zh string
		ok                bool
	}{
		{"A.json", "/p", "kr", "zh", true},
		{"A.json", "/o", "kr", "", false},
		{"F.json", "/o", "kr", "zh3", true},
		{"A.json", "/c", "kr", "", false},
		{"F.json", "/c", "kr", "", false},
		{"A.json", "/none", "kr", "", false},
	} {
		zh, ok := text(c.file, c.ptr, c.kr)
		if zh != c.zh || ok != c.ok {
			t.Errorf("%s%s: got %q,%v want %q,%v", c.file, c.ptr, zh, ok, c.zh, c.ok)
		}
	}
}

func TestInputSharedLines(t *testing.T) {
	p := openPack(t, map[string]string{
		"KR_BattleSpeechBubbleDlg.json":                 `{"dataList":[{"id":"a","desc":"메모","dlg":"가"},{"id":"b","dlg":"나"},{"id":"c","dlg":"다"},{"id":"d"}]}`,
		"PersonalityVoiceDlg/KR_Voice_Faust_10201.json": `{"dataList":[{"id":"e","desc":"제목","dlg":"나"},{"id":"f","dlg":"라"}]}`,
		"EGOVoiceDig/KR_Voice_EGO_Faust.json":           `{"dataList":[{"id":"g","dlg":"마"}]}`,
		"StoryData/KR_S1.json":                          `{"dataList":[{"id":0,"content":"가"}]}`,
		"KR_BattleSpeechBubbleDlg-x.json":               `{"dataList":[{"id":"h","dlg":"바"}]}`,
	})
	records := map[string]store.Record{
		units.Key("BattleSpeechBubbleDlg.json", "/dataList/0/dlg"):                 rec("가", "甲", store.OriginClaude),
		units.Key("BattleSpeechBubbleDlg.json", "/dataList/0/desc"):                rec("메모", "甲", store.OriginClaude),
		units.Key("BattleSpeechBubbleDlg.json", "/dataList/1/dlg"):                 rec("나", "甲", store.OriginClaude),
		units.Key("BattleSpeechBubbleDlg.json", "/dataList/2/dlg"):                 rec("old", "甲", store.OriginClaude),
		units.Key("PersonalityVoiceDlg/Voice_Faust_10201.json", "/dataList/0/dlg"): rec("나", "甲", store.OriginClaude),
		units.Key("PersonalityVoiceDlg/Voice_Faust_10201.json", "/dataList/1/dlg"): rec("라", "乙", store.OriginClaude),
		units.Key("EGOVoiceDig/Voice_EGO_Faust.json", "/dataList/0/dlg"):           rec("마", "甲", store.OriginClaude),
		units.Key("StoryData/S1.json", "/dataList/0/content"):                      rec("가", "甲", store.OriginClaude),
		units.Key("BattleSpeechBubbleDlg-x.json", "/dataList/0/dlg"):               rec("바", "바", store.OriginCopy),
	}
	in, err := Input(p, UsedText(records, nil))
	if err != nil {
		t.Fatal(err)
	}
	// Only the dlg of the bubbles and the voice lines with a usable record,
	// each KR text once, in order.
	want := map[string][]string{"甲": {"가", "나"}, "乙": {"라"}}
	if !reflect.DeepEqual(in.SharedLines, want) {
		t.Errorf("SharedLines = %v, want %v", in.SharedLines, want)
	}
}
