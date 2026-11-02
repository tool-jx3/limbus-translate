package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

const deriveBufs = "Bufs.json"
const deriveSpeakers = "ScenarioModelCodes-AutoCreated.json"

// deriveCandidate is a row of "glossary derive --json".
type deriveCandidate struct {
	KR     string `json:"kr"`
	ZH     string `json:"zh"`
	Source string `json:"source"`
}

// deriveOut is "glossary derive --json".
type deriveOut struct {
	Candidates []deriveCandidate `json:"candidates"`
}

// deriveRepo is a repo root in the working directory with the rules of the
// repo and a glossary that lists 알려진, a raw snapshot whose KR pack has four
// keywords and two speakers, and a tw worktree with a translation of each
// name. 거절됨 is declined and 알려진 is in the glossary; 파열 and the speaker
// 료슈 are left.
func deriveRepo(t *testing.T) string {
	t.Helper()
	root := cleanRules(t)
	copyFile(t, filepath.Join(repoRoot, "lct.yaml"), filepath.Join(root, "lct.yaml"))
	writeTestFile(t, root, "glossary/terms.yaml", "- {kr: 알려진, zh: 已知, status: approved}\n- {kr: 알려진 사람, zh: 已知的人, status: candidate}\n")
	writeTestFile(t, root, "rules/declined.yaml", "- {id: 'term/거절됨', reason: 不需要, date: 2026-11-05}\n")
	writeTestFile(t, root, ".work/raw/meta/source.json", `{"tag": "raw-20261029", "token": "t", "created_at": "2026-10-29 02:55:23", "fetched_at": "2026-10-29T00:00:00Z", "files": {}}`)
	writeTestFile(t, root, ".work/raw/kr/KR_"+deriveBufs, `{"dataList":[
 {"id":"Burst","name":"파열"},
 {"id":"Known","name":"알려진"},
 {"id":"Declined","name":"거절됨"},
 {"id":"Korean","name":"한글 남음"}
]}`)
	writeTestFile(t, root, ".work/raw/kr/KR_"+deriveSpeakers, `{"dataList":[
 {"id":"료슈","name":"료슈"},
 {"id":"알려진 사람","name":"알려진 사람"}
]}`)
	writeTestFile(t, root, ".work/raw/en/EN_Other.json", "{}\n")
	writeTestFile(t, root, ".work/raw/jp/JP_Other.json", "{}\n")

	rec := func(file, pointer, kr, zh string) store.Record {
		return store.Record{Key: units.Key(file, pointer), KRSha: units.KRSha(kr), ZH: zh, Origin: store.OriginParatranz, Since: "raw-20261029"}
	}
	recs := map[string]store.Record{}
	for _, r := range []store.Record{
		rec(deriveBufs, "/dataList/0/name", "파열", "破裂"),
		rec(deriveBufs, "/dataList/1/name", "알려진", "已知"),
		rec(deriveBufs, "/dataList/2/name", "거절됨", "被拒"),
		rec(deriveBufs, "/dataList/3/name", "한글 남음", "한글 殘留"),
		rec(deriveSpeakers, "/dataList/0/name", "료슈", "良秀"),
		rec(deriveSpeakers, "/dataList/1/name", "알려진 사람", "已知的人"),
	} {
		recs[r.Key] = r
	}
	if err := store.Open(filepath.Join(root, ".work", "tw")).SaveRecords(recs); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	return root
}

func TestGlossaryDeriveCommand(t *testing.T) {
	root := deriveRepo(t)
	glossaryBefore, err := os.ReadFile(filepath.Join(root, "glossary", "terms.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	keyword := deriveCandidate{KR: "파열", ZH: "破裂", Source: "keyword"}
	speaker := deriveCandidate{KR: "료슈", ZH: "良秀", Source: "speaker"}
	tests := []struct {
		args []string
		want []deriveCandidate
	}{
		{nil, []deriveCandidate{speaker, keyword}},
		{[]string{"--source", "keyword"}, []deriveCandidate{keyword}},
		{[]string{"--source=speaker"}, []deriveCandidate{speaker}},
	}
	for _, tc := range tests {
		var out deriveOut
		mustLct(t, &out, append([]string{"glossary", "derive"}, tc.args...)...)
		if !reflect.DeepEqual(out.Candidates, tc.want) {
			t.Errorf("glossary derive %v = %+v, want %+v", tc.args, out.Candidates, tc.want)
		}
	}

	// The text output names each candidate; nothing is written.
	code, stdout, stderr := lct(t, "glossary", "derive")
	if code != 0 || stderr != "" {
		t.Fatalf("glossary derive: exit code %d, stderr %q", code, stderr)
	}
	for _, want := range []string{"2 candidate", "파열", "破裂", "keyword", "료슈", "良秀", "speaker"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the text output does not mention %q:\n%s", want, stdout)
		}
	}
	if after, err := os.ReadFile(filepath.Join(root, "glossary", "terms.yaml")); err != nil || string(after) != string(glossaryBefore) {
		t.Errorf("glossary derive changed the glossary: %v", err)
	}
}

func TestGlossaryDeriveCommandNothingLeft(t *testing.T) {
	root := deriveRepo(t)
	writeTestFile(t, root, "glossary/terms.yaml", "- {kr: 파열, zh: 破裂, status: approved}\n- {kr: 료슈, zh: 良秀, status: candidate}\n- {kr: 알려진, zh: 已知, status: approved}\n- {kr: 알려진 사람, zh: 已知的人, status: approved}\n")
	var out deriveOut
	mustLct(t, &out, "glossary", "derive")
	if out.Candidates == nil || len(out.Candidates) != 0 {
		t.Errorf("candidates = %#v, want an empty list", out.Candidates)
	}
	if code, stdout, _ := lct(t, "glossary", "derive"); code != 0 || !strings.Contains(stdout, "0 candidate") {
		t.Errorf("glossary derive with nothing to list: exit code %d, output %q", code, stdout)
	}
}

func TestGlossaryDeriveCommandUserErrors(t *testing.T) {
	root := deriveRepo(t)
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"unknown source", []string{"--source", "buff"}, "--source"},
		{"empty source", []string{"--source="}, "--source"},
		{"unexpected argument", []string{"extra"}, "unexpected argument"},
		{"unknown flag", []string{"--nope"}, "derive"},
	}
	for _, tc := range tests {
		code, stdout, stderr := lct(t, append([]string{"glossary", "derive"}, tc.args...)...)
		if code != 1 || !strings.Contains(stderr, tc.want) {
			t.Errorf("%s: exit code %d (want 1), stderr %q (want it to mention %q), stdout %q", tc.name, code, stderr, tc.want, stdout)
		}
	}

	// A broken declined list is an error and not an empty list: it would let a
	// declined term come back.
	writeTestFile(t, root, "rules/declined.yaml", "not: a list\n")
	if code, _, stderr := lct(t, "glossary", "derive"); code == 0 || !strings.Contains(stderr, "declined.yaml") {
		t.Errorf("with a broken declined list: exit code %d, stderr %q", code, stderr)
	}

	// Without the tw worktree there are no translations to derive from.
	if err := os.RemoveAll(filepath.Join(root, ".work", "tw")); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, "rules/declined.yaml", "")
	if code, _, stderr := lct(t, "glossary", "derive"); code != 1 || !strings.Contains(stderr, "tw") {
		t.Errorf("without the tw worktree: exit code %d (want 1), stderr %q (want it to mention tw)", code, stderr)
	}
}

func TestGlossaryDeriveCommandNeedsASnapshot(t *testing.T) {
	root := deriveRepo(t)
	if err := os.RemoveAll(filepath.Join(root, ".work", "raw")); err != nil {
		t.Fatal(err)
	}
	if code, _, stderr := lct(t, "glossary", "derive"); code != 1 || !strings.Contains(stderr, "snapshot") {
		t.Errorf("without a snapshot: exit code %d (want 1), stderr %q", code, stderr)
	}
}
