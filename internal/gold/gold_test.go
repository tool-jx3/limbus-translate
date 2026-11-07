package gold

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func cands(family string, n int) []Candidate {
	var out []Candidate
	for i := 0; i < n; i++ {
		out = append(out, Candidate{Key: fmt.Sprintf("%s.json|/dataList/%d/content", family, i), Family: family, Origin: "claude"})
	}
	return out
}

func TestDrawIsDeterministicAndCapped(t *testing.T) {
	all := append(cands("skill", 30), cands("story", 5)...)
	a := Draw(all, DefaultSeed, 20)
	if got := len(a); got != 25 {
		t.Fatalf("drew %d units, want 20 of skill and all 5 of story", got)
	}
	count := map[string]int{}
	for _, u := range a {
		count[u.Family]++
		if u.Verdict != "" {
			t.Errorf("%s has a verdict", u.Key)
		}
	}
	if count["skill"] != 20 || count["story"] != 5 {
		t.Errorf("per family = %v", count)
	}
	// The input order does not matter; the seed does.
	rev := append([]Candidate(nil), all...)
	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}
	if b := Draw(rev, DefaultSeed, 20); !reflect.DeepEqual(a, b) {
		t.Error("the draw depends on the order of the candidates")
	}
	if c := Draw(all, 1, 20); reflect.DeepEqual(a, c) {
		t.Error("another seed drew the same units")
	}
}

func TestSaveLoadRoundTripAndSet(t *testing.T) {
	root := t.TempDir()
	if s, err := Load(root); err != nil || len(s.Units) != 0 {
		t.Fatalf("Load without file = %+v, %v", s, err)
	}
	s := Sample{Seed: 7, PerFamily: 2, Units: []Unit{{Key: "a|/x", Family: "f", Origin: "claude"}}}
	if err := Save(root, s); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(filepath.Join(root, File))
	got, err := Load(root)
	if err != nil || !reflect.DeepEqual(got, s) {
		t.Fatalf("Load = %+v, %v; want %+v", got, err, s)
	}
	if err := got.Set("a|/x", "wrong", "漏譯: 冒號", "譯文", time.Date(2026, 11, 7, 23, 0, 0, 0, time.FixedZone("x", 8*3600))); err != nil {
		t.Fatal(err)
	}
	u := got.Units[0]
	if u.Verdict != "wrong" || u.Note != "漏譯: 冒號" || u.JudgedZH != "譯文" || u.JudgedAt != "2026-11-07" {
		t.Errorf("unit = %+v", u)
	}
	if err := Save(root, got); err != nil {
		t.Fatal(err)
	}
	again, err := Load(root)
	if err != nil || !reflect.DeepEqual(again, got) {
		t.Errorf("round trip = %+v, %v", again, err)
	}
	// Saving the unchanged sample gives the same bytes.
	if err := Save(root, s); err != nil {
		t.Fatal(err)
	}
	if second, _ := os.ReadFile(filepath.Join(root, File)); string(second) != string(first) {
		t.Error("Save is not stable")
	}
	if err := got.Set("nope|/x", "correct", "", "", time.Now()); err == nil || !strings.Contains(err.Error(), "nope|/x") {
		t.Errorf("unknown key error = %v", err)
	}
	if err := got.Set("a|/x", "fine", "", "", time.Now()); err == nil {
		t.Error("an unknown verdict was accepted")
	}
}

func TestLoadRefusesBadFiles(t *testing.T) {
	for name, body := range map[string]string{
		"verdict":   "seed: 1\nper_family: 1\nunits:\n  - {key: a|/x, family: f, origin: o, verdict: great}\n",
		"duplicate": "seed: 1\nper_family: 1\nunits:\n  - {key: a|/x, family: f, origin: o}\n  - {key: a|/x, family: f, origin: o}\n",
		"unknown":   "seed: 1\nbogus: 1\n",
	} {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, "rules"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, File), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(root); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestMakeReport(t *testing.T) {
	s := Sample{Units: []Unit{
		{Key: "1", Family: "skill", Origin: "claude", Verdict: "correct", JudgedZH: "a"},
		{Key: "2", Family: "skill", Origin: "claude", Verdict: "wrong", JudgedZH: "b"},
		{Key: "3", Family: "skill", Origin: "claude", Verdict: "unsure", JudgedZH: "c"},
		{Key: "4", Family: "skill", Origin: "paratranz", Verdict: "correct", JudgedZH: "d"},
		{Key: "5", Family: "story", Origin: "claude"},
		{Key: "6", Family: "story", Origin: "claude", Verdict: "unsure", JudgedZH: "f"},
	}}
	cur := map[string]string{"1": "a", "2": "changed", "3": "c", "4": "d", "5": "e", "6": "f"}
	r := MakeReport(s, cur)
	find := func(rows []Row, f, o string) Row {
		for _, x := range rows {
			if x.Family == f && x.Origin == o {
				return x
			}
		}
		t.Fatalf("no row %s/%s", f, o)
		return Row{}
	}
	sc := find(r.Rows, "skill", "claude")
	if sc.Sampled != 3 || sc.Judged != 3 || sc.Correct != 1 || sc.Wrong != 1 || sc.Unsure != 1 || sc.Stale != 1 {
		t.Errorf("skill/claude = %+v", sc)
	}
	if sc.Accuracy == nil || *sc.Accuracy != 0.5 {
		t.Errorf("accuracy = %v, want 0.5", sc.Accuracy)
	}
	// Only unsure judged: no accuracy. An unjudged unit is not stale.
	st := find(r.Rows, "story", "claude")
	if st.Sampled != 2 || st.Judged != 1 || st.Accuracy != nil || st.Stale != 0 {
		t.Errorf("story/claude = %+v", st)
	}
	if f := find(r.Families, "skill", ""); f.Correct != 2 || f.Wrong != 1 {
		t.Errorf("family skill = %+v", f)
	}
	if r.Total.Sampled != 6 || r.Total.Judged != 5 || r.Total.Stale != 1 {
		t.Errorf("total = %+v", r.Total)
	}
	if a := r.Total.Accuracy; a == nil || *a < 0.66 || *a > 0.67 {
		t.Errorf("total accuracy = %v", a)
	}
	if !strings.Contains(r.String(), "skill/claude") {
		t.Errorf("text report = %q", r.String())
	}
}
