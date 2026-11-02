package glossary_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/glossary"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/suggest"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// krPack opens a KR pack of the given files (name to content).
func krPack(t *testing.T, files map[string]string) *pack.Pack {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		p := filepath.Join(root, "KR_"+name)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p, err := pack.Open(root, "KR_")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// translate is the record that makes zh the translation of the KR text kr at
// pointer of file.
func translate(file, pointer, kr, zh string, flags ...string) store.Record {
	k := units.Key(file, pointer)
	return store.Record{Key: k, KRSha: units.KRSha(kr), ZH: zh, Origin: store.OriginParatranz, Since: "raw-20261029", Flags: flags}
}

func recordsOf(recs ...store.Record) map[string]store.Record {
	m := map[string]store.Record{}
	for _, r := range recs {
		m[r.Key] = r
	}
	return m
}

const (
	bufsFile     = "Bufs.json"
	speakersFile = "ScenarioModelCodes-AutoCreated.json"
)

func TestGlossaryDeriveKeywordsAndSpeakers(t *testing.T) {
	kr := krPack(t, map[string]string{
		bufsFile:     `{"dataList":[{"id":"Burst","name":"파열"}]}`,
		speakersFile: `{"dataList":[{"id":"료슈","name":"료슈","nickName":"4번 수감자"}]}`,
	})
	records := recordsOf(
		translate(bufsFile, "/dataList/0/name", "파열", "破裂"),
		translate(speakersFile, "/dataList/0/name", "료슈", "良秀"),
		translate(speakersFile, "/dataList/0/nickName", "4번 수감자", "4號罪人"),
	)

	got, err := glossary.Derive(glossary.Inputs{KR: kr, Records: records})
	if err != nil {
		t.Fatal(err)
	}
	want := []glossary.Candidate{
		{KR: "료슈", ZH: "良秀", Source: "speaker"},
		{KR: "파열", ZH: "破裂", Source: "keyword"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Derive = %+v, want %+v: the names the translation uses, sorted by KR, each with its source", got, want)
	}
}

func TestGlossaryDeriveSkipsKnownAndDeclined(t *testing.T) {
	kr := krPack(t, map[string]string{
		bufsFile: `{"dataList":[
 {"id":"Known","name":"알려진"},
 {"id":"Scoped","name":"한정"},
 {"id":"Declined","name":"거절됨"},
 {"id":"Korean","name":"한글 남음"},
 {"id":"Untranslated","name":"번역 안 됨"},
 {"id":"Stale","name":"바뀐 원문"},
 {"id":"Review","name":"검토 중"},
 {"id":"Kept","name":"남는다"}
]}`,
		speakersFile: `{"dataList":[
 {"id":"알려진 사람","name":"알려진 사람"},
 {"id":"거절된 사람","name":"거절된 사람"},
 {"id":"새 사람","name":"새 사람"}
]}`,
	})
	records := recordsOf(
		translate(bufsFile, "/dataList/0/name", "알려진", "已知"),
		translate(bufsFile, "/dataList/1/name", "한정", "限定"),
		translate(bufsFile, "/dataList/2/name", "거절됨", "被拒"),
		translate(bufsFile, "/dataList/3/name", "한글 남음", "한글 殘留"),
		// Made from the KR text of an earlier version.
		translate(bufsFile, "/dataList/5/name", "옛 원문", "舊文"),
		translate(bufsFile, "/dataList/6/name", "검토 중", "審核中", store.FlagNeedsReview),
		translate(bufsFile, "/dataList/7/name", "남는다", "留下"),
		translate(speakersFile, "/dataList/0/name", "알려진 사람", "已知的人"),
		translate(speakersFile, "/dataList/1/name", "거절된 사람", "被拒的人"),
		translate(speakersFile, "/dataList/2/name", "새 사람", "新人"),
	)
	in := glossary.Inputs{
		KR:      kr,
		Records: records,
		Terms: []rules.Term{
			// Any status and any files list the KR text.
			{KR: "알려진", ZH: "已知", Status: rules.StatusCandidate},
			{KR: "한정", ZH: "限定", Status: rules.StatusApproved, Files: []string{"StoryData/**"}},
			{KR: "알려진 사람", ZH: "已知的人", Status: rules.StatusApproved},
		},
		Declined: []suggest.Declined{
			{ID: "term/거절됨", Reason: "不需要", Date: "2026-11-05"},
			{ID: "term/거절된 사람", Reason: "不需要", Date: "2026-11-05"},
			// Not a term: does not decline the keyword of that name.
			{ID: "speaker/남는다", Reason: "r", Date: "2026-11-05"},
		},
	}

	got, err := glossary.Derive(in)
	if err != nil {
		t.Fatal(err)
	}
	want := []glossary.Candidate{
		{KR: "남는다", ZH: "留下", Source: "keyword"},
		{KR: "새 사람", ZH: "新人", Source: "speaker"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Derive = %+v, want %+v: not the KR texts of the glossary or of declined terms, not names that are still Korean", got, want)
	}
}

// A name that several ids share is one candidate, and two renderings of one
// KR name are two.
func TestGlossaryDeriveSameNameOnce(t *testing.T) {
	kr := krPack(t, map[string]string{
		bufsFile:                `{"dataList":[{"id":"A","name":"파열"},{"id":"B","name":"파열"},{"id":"C","name":"출혈"}]}`,
		"BattleKeywords.json":   `{"dataList":[{"id":"A","name":"파열"}]}`,
		"BattleKeywords-x.json": `{"dataList":[{"id":"D","name":"출혈"}]}`,
	})
	records := recordsOf(
		translate(bufsFile, "/dataList/0/name", "파열", "破裂"),
		translate(bufsFile, "/dataList/1/name", "파열", "破裂"),
		translate(bufsFile, "/dataList/2/name", "출혈", "流血"),
		translate("BattleKeywords.json", "/dataList/0/name", "파열", "破裂"),
		translate("BattleKeywords-x.json", "/dataList/0/name", "출혈", "出血"),
	)
	got, err := glossary.Derive(glossary.Inputs{KR: kr, Records: records})
	if err != nil {
		t.Fatal(err)
	}
	want := []glossary.Candidate{
		{KR: "출혈", ZH: "出血", Source: "keyword"},
		{KR: "출혈", ZH: "流血", Source: "keyword"},
		{KR: "파열", ZH: "破裂", Source: "keyword"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Derive = %+v, want %+v", got, want)
	}
}

// A pack without a speaker file, or without keyword files, has nothing to
// derive from them; and nothing derived is an empty list, not nil.
func TestGlossaryDeriveWithoutFiles(t *testing.T) {
	kr := krPack(t, map[string]string{"Other.json": `{"dataList":[{"id":"x","name":"이름"}]}`})
	got, err := glossary.Derive(glossary.Inputs{KR: kr})
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("Derive = %#v, want an empty list", got)
	}
	if _, err := glossary.Derive(glossary.Inputs{}); err == nil {
		t.Error("Derive without a KR pack did not fail")
	}
}
