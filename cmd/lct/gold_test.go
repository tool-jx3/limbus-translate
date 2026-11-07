package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/gold"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
)

// goldRepo is batchRepo with a KR pack of two files (Sample.json with six
// units, StoryData/S1.json with two), records for all but one of them, one of
// the records a copy, and the repo as the working directory.
func goldRepo(t *testing.T) (root string, keys map[string][]string) {
	t.Helper()
	root, st := batchRepo(t)
	rawWT := filepath.Join(root, ".work", "raw")
	var sample, story []string
	recs := map[string]store.Record{}
	keys = map[string][]string{}
	add := func(file string, i int, origin string, record bool) {
		key := fmt.Sprintf("%s|/dataList/%d/content", file, i)
		if record {
			recs[key] = store.Record{Key: key, KRSha: "00", ZH: fmt.Sprintf("譯%d", i), Origin: origin, Since: "raw-20261029"}
		}
		fam := mustFamily(t, root, file)
		if record && origin != store.OriginCopy {
			keys[fam] = append(keys[fam], key)
		}
	}
	for i := 0; i < 6; i++ {
		sample = append(sample, fmt.Sprintf(`{"id":%d,"content":"가%d"}`, i, i))
	}
	for i := 0; i < 2; i++ {
		story = append(story, fmt.Sprintf(`{"id":%d,"content":"나%d"}`, i, i))
	}
	writeTestFile(t, rawWT, "kr/KR_Sample.json", `{"dataList":[`+strings.Join(sample, ",")+`]}`)
	writeTestFile(t, rawWT, "kr/StoryData/KR_S1.json", `{"dataList":[`+strings.Join(story, ",")+`]}`)
	add("Sample.json", 0, store.OriginClaude, true)
	add("Sample.json", 1, store.OriginClaude, true)
	add("Sample.json", 2, store.OriginHuman, true)
	add("Sample.json", 3, store.OriginCopy, true)    // a copy: never drawn
	add("Sample.json", 4, store.OriginClaude, false) // no record: never drawn
	add("StoryData/S1.json", 0, store.OriginParatranz, true)
	add("StoryData/S1.json", 1, store.OriginClaude, true)
	if err := st.SaveRecords(recs); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	return root, keys
}

func mustFamily(t *testing.T, root, file string) string {
	t.Helper()
	r, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return r.Family(file)
}

type goldSampleResult struct {
	Seed      int            `json:"seed"`
	PerFamily int            `json:"per_family"`
	Units     int            `json:"units"`
	Families  map[string]int `json:"families"`
}

func goldSample(t *testing.T, args ...string) (int, goldSampleResult, string) {
	t.Helper()
	code, stdout, stderr := lct(t, append([]string{"--json", "gold", "sample"}, args...)...)
	var out goldSampleResult
	if code == 0 {
		if err := json.Unmarshal([]byte(stdout), &out); err != nil {
			t.Fatalf("not JSON: %v\n%s", err, stdout)
		}
	}
	return code, out, stderr
}

func TestGoldSampleDrawsPerFamily(t *testing.T) {
	root, keys := goldRepo(t)
	sampleFam, storyFam := mustFamily(t, root, "Sample.json"), mustFamily(t, root, "StoryData/S1.json")
	if sampleFam == storyFam {
		t.Fatalf("the fixture needs two families, both are %q", sampleFam)
	}
	code, out, stderr := goldSample(t, "--per-family", "2")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if out.Seed != gold.DefaultSeed || out.PerFamily != 2 || out.Units != 4 || out.Families[sampleFam] != 2 || out.Families[storyFam] != 2 {
		t.Errorf("output = %+v", out)
	}
	s, err := gold.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if s.Seed != gold.DefaultSeed || s.PerFamily != 2 || len(s.Units) != 4 {
		t.Fatalf("sample = %+v", s)
	}
	for _, u := range s.Units {
		if strings.HasPrefix(u.Key, "Sample.json|/dataList/3") || strings.HasPrefix(u.Key, "Sample.json|/dataList/4") {
			t.Errorf("%s: a copy or a unit without a record was drawn", u.Key)
		}
	}
	_ = keys

	// Fewer than N: all of them. Same input, same file.
	before, _ := os.ReadFile(filepath.Join(root, gold.File))
	if code, out, stderr = goldSample(t); code != 0 || out.Units != 5 {
		t.Fatalf("default per family: exit %d, %+v, %s", code, out, stderr)
	}
	if code, _, _ := goldSample(t, "--per-family", "2"); code != 0 {
		t.Fatal("redrawing an unjudged sample was refused")
	}
	if after, _ := os.ReadFile(filepath.Join(root, gold.File)); string(after) != string(before) {
		t.Error("the same draw gave other bytes")
	}
}

func TestGoldSampleRefusesToOverwriteJudgements(t *testing.T) {
	root, _ := goldRepo(t)
	if code, _, stderr := goldSample(t); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	s, _ := gold.Load(root)
	key := s.Units[0].Key
	if code, _, stderr := lct(t, "gold", "set", key, "correct"); code != 0 {
		t.Fatalf("set: exit %d: %s", code, stderr)
	}
	before, _ := os.ReadFile(filepath.Join(root, gold.File))
	code, _, stderr := goldSample(t, "--seed", "5")
	if code != 1 || !strings.Contains(stderr, "--force") {
		t.Errorf("exit %d, stderr %q; want 1 and a hint to use --force", code, stderr)
	}
	if after, _ := os.ReadFile(filepath.Join(root, gold.File)); string(after) != string(before) {
		t.Error("a refused call changed the file")
	}
	if code, _, stderr := goldSample(t, "--seed", "5", "--force"); code != 0 {
		t.Fatalf("--force: exit %d: %s", code, stderr)
	}
	s, _ = gold.Load(root)
	if s.Seed != 5 || s.Judged() != 0 {
		t.Errorf("after --force: seed %d, judged %d", s.Seed, s.Judged())
	}
}

func TestGoldSetAndReport(t *testing.T) {
	root, _ := goldRepo(t)
	oldNow := goldNow
	goldNow = func() time.Time { return time.Date(2026, 11, 7, 23, 30, 0, 0, time.UTC) }
	t.Cleanup(func() { goldNow = oldNow })
	if code, _, stderr := goldSample(t); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	s, _ := gold.Load(root)
	var claude, other []gold.Unit
	for _, u := range s.Units {
		if u.Origin == store.OriginClaude {
			claude = append(claude, u)
		} else {
			other = append(other, u)
		}
	}
	if len(claude) < 2 || len(other) < 1 {
		t.Fatalf("fixture drew %d claude, %d other", len(claude), len(other))
	}

	// Errors: key not in the sample, a bad verdict, too few arguments.
	for name, args := range map[string][]string{
		"unknown key": {"Sample.json|/dataList/3/content", "correct"},
		"bad verdict": {claude[0].Key, "good"},
		"no verdict":  {claude[0].Key},
		"extra":       {claude[0].Key, "correct", "x"},
	} {
		if code, _, stderr := lct(t, append([]string{"gold", "set"}, args...)...); code != 1 || stderr == "" {
			t.Errorf("%s: exit %d, stderr %q; want 1", name, code, stderr)
		}
	}

	if code, _, stderr := lct(t, "gold", "set", claude[0].Key, "correct", "--note", "ok"); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if code, _, stderr := lct(t, "gold", "set", "--note", "漏譯", claude[1].Key, "wrong"); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if code, _, stderr := lct(t, "gold", "set", other[0].Key, "unsure"); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	s, _ = gold.Load(root)
	for _, u := range s.Units {
		if u.Key != claude[1].Key {
			continue
		}
		if u.Verdict != "wrong" || u.Note != "漏譯" || u.JudgedAt != "2026-11-07" || u.JudgedZH == "" {
			t.Errorf("unit = %+v", u)
		}
	}

	// Change the translation of the first judged unit: it is stale.
	st := store.Open(filepath.Join(root, ".work", "tw"))
	recs, err := st.LoadRecords()
	if err != nil {
		t.Fatal(err)
	}
	r := recs[claude[0].Key]
	r.ZH = "改過"
	recs[claude[0].Key] = r
	if err := st.SaveRecords(recs); err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := lct(t, "--json", "gold", "report")
	if code != 0 {
		t.Fatalf("report: exit %d: %s", code, stderr)
	}
	var rep gold.Report
	if err := json.Unmarshal([]byte(stdout), &rep); err != nil {
		t.Fatalf("%v\n%s", err, stdout)
	}
	if rep.Total.Judged != 3 || rep.Total.Correct != 1 || rep.Total.Wrong != 1 || rep.Total.Unsure != 1 || rep.Total.Stale != 1 {
		t.Errorf("total = %+v", rep.Total)
	}
	if rep.Total.Accuracy == nil || *rep.Total.Accuracy != 0.5 {
		t.Errorf("accuracy = %v", rep.Total.Accuracy)
	}
	if code, stdout, _ := lct(t, "gold", "report"); code != 0 || !strings.Contains(stdout, "accuracy") {
		t.Errorf("text report: exit %d, %q", code, stdout)
	}
}

func TestGoldWithoutSample(t *testing.T) {
	goldRepo(t)
	for _, args := range [][]string{{"gold", "set", "A|/x", "correct"}, {"gold", "report"}} {
		if code, _, stderr := lct(t, args...); code != 1 || !strings.Contains(stderr, "gold sample") {
			t.Errorf("%v: exit %d, stderr %q; want 1 and a hint to run lct gold sample", args, code, stderr)
		}
	}
}
