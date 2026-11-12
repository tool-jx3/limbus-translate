package main

import (
	"encoding/json"
	"github.com/tool-jx3/limbus-translate/internal/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// planOut is the part of "plan --json" the tests look at.
type planOut struct {
	RawTag     string `json:"raw_tag"`
	PrevRawTag string `json:"prev_raw_tag"`
	Changes    struct {
		Unchanged, Changed, New, Deleted int
	} `json:"changes"`
	Pending   int `json:"pending"`
	NameTable struct {
		RawTag string `json:"raw_tag"`
		Built  bool   `json:"built"`
		Names  int    `json:"names"`
	} `json:"name_table"`
}

// runPlanCLI runs "lct --json plan" in the working directory and returns what
// it printed with the content of .meta/plan.json, which must be the same.
func runPlanCLI(t *testing.T, root string) planOut {
	t.Helper()
	code, stdout, stderr := lct(t, "--json", "plan")
	if code != 0 || stderr != "" {
		t.Fatalf("lct plan: exit code %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	written := readFile(t, filepath.Join(root, ".work", "tw", ".meta", "plan.json"))
	if stdout != string(written) {
		t.Errorf("lct plan printed\n%s\nbut .meta/plan.json holds\n%s", stdout, written)
	}
	var out planOut
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("lct plan: output is not JSON: %v\n%s", err, stdout)
	}
	return out
}

func TestPlanCommandWritesPlanJSON(t *testing.T) {
	fixture, err := filepath.Abs(e2eFixture)
	if err != nil {
		t.Fatal(err)
	}
	root := e2eRepo(t)
	t.Chdir(root)
	mustLct(t, nil, "snapshot", "--from", fixture, "--token", testToken, "--created-at", testCreatedAt)
	mustLct(t, nil, "bootstrap", "--upstream", filepath.Join(fixture, "upstream"))

	first := runPlanCLI(t, root)
	if first.RawTag != "raw-20261029" || first.PrevRawTag != "raw-20261029" || first.Changes.Unchanged != 0 {
		t.Errorf("first plan = %+v, want raw-20261029 from itself, no changes", first)
	}

	// The next game update edits one line of the story.
	next := filepath.Join(t.TempDir(), "next")
	copyDir(t, fixture, next)
	story := filepath.Join(next, "kr", "StoryData", "KR_S001A.json")
	data := readFile(t, story)
	edited := strings.Replace(string(data), "해가 저물고 있었다.", "해가 천천히 저물고 있었다.", 1)
	if edited == string(data) {
		t.Fatalf("the fixture %s does not have the line the test edits", story)
	}
	if err := os.WriteFile(story, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	mustLct(t, nil, "snapshot", "--from", next, "--token", "l20261105_testtoken", "--created-at", "2026-11-05 02:55:23")

	second := runPlanCLI(t, root)
	if second.RawTag != "raw-20261105" || second.PrevRawTag != "raw-20261029" || second.Changes.Changed != 1 {
		t.Errorf("second plan = %+v, want raw-20261105 from raw-20261029 with one changed line", second)
	}
	if second.Pending < 1 {
		t.Errorf("second plan has %d pending units, want the changed line among them", second.Pending)
	}

	// As text it is a summary for people.
	code, stdout, stderr := lct(t, "plan")
	if code != 0 || !strings.Contains(stdout, "raw-20261105") {
		t.Errorf("lct plan as text: exit code %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
}

func TestPlanCommandUserErrors(t *testing.T) {
	root := e2eRepo(t)
	t.Chdir(root)
	for _, args := range [][]string{{"plan", "extra"}, {"plan", "--nope"}, {"plan"}} {
		code, stdout, stderr := lct(t, args...)
		if code != 1 {
			t.Errorf("lct %s: exit code %d, want 1 (no snapshot, or bad arguments)\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), code, stdout, stderr)
		}
	}
}

func TestPlanCommandPassesNameFields(t *testing.T) {
	fixture, err := filepath.Abs(e2eFixture)
	if err != nil {
		t.Fatal(err)
	}
	root := e2eRepo(t)
	t.Chdir(root)
	if cfg, err := config.Load(root); err != nil || len(cfg.Names.Fields) == 0 {
		t.Fatalf("lct.yaml names.fields = %v (%v), want some", cfg.Names.Fields, err)
	}
	mustLct(t, nil, "snapshot", "--from", fixture, "--token", testToken, "--created-at", testCreatedAt)
	mustLct(t, nil, "bootstrap", "--upstream", filepath.Join(fixture, "upstream"))

	out := runPlanCLI(t, root)
	if !out.NameTable.Built || out.NameTable.Names <= 0 {
		t.Errorf("name_table = %+v, want built with names", out.NameTable)
	}
}
