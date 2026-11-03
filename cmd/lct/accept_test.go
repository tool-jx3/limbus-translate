package main

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/plan"
	"github.com/tool-jx3/limbus-translate/internal/store"
)

// acceptState is a repo with a build, a rule suggestion, warnings and a note
// to accept: root is the working directory.
type acceptState struct {
	root       string
	suggestion string   // a speaker/… id
	note       string   // a note/… id
	warnings   []string // warning/… ids, at least two
}

// acceptRepo builds the e2e repo with two broken records (an ASCII ellipsis and
// kana, which the build warns about), a new speaker in the plan
// and a note on a stored unit.
func acceptRepo(t *testing.T) acceptState {
	t.Helper()
	fixture, err := filepath.Abs(e2eFixture)
	if err != nil {
		t.Fatal(err)
	}
	root := e2eRepo(t)
	template := e2eCDN(t, fixture, testToken)
	t.Chdir(root)
	t.Setenv("LCT_MODEL", "e2e")
	t.Setenv(sourceDateEpoch, "1793000000")

	mustLct(t, nil, "fetch", "--token", testToken, "--created-at", testCreatedAt, "--cdn-template", template, "--out", "work/fetch")
	mustLct(t, nil, "snapshot", "--from", "work/fetch", "--token", testToken, "--created-at", testCreatedAt)
	mustLct(t, nil, "bootstrap", "--upstream", filepath.Join(fixture, "upstream"))
	st := store.Open(filepath.Join(root, ".work", "tw"))
	records, err := st.LoadRecords()
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	edits := []string{"...", "か"}
	for _, key := range slices.Sorted(maps.Keys(records)) {
		rec := records[key]
		if rec.Origin != store.OriginParatranz || rec.ZH == "" {
			continue
		}
		keys = append(keys, key)
		if len(edits) > 0 {
			rec.ZH += edits[0]
			records[key] = rec
			edits = edits[1:]
		}
	}
	if len(edits) != 0 || len(keys) < 3 {
		t.Fatal("the fixture has too few paratranz records")
	}
	if err := st.SaveRecords(records); err != nil {
		t.Fatal(err)
	}
	mustLct(t, nil, "build")

	// A new speaker in the plan and a note, as the planner and the
	// translators leave them.
	if err := st.WriteMetaJSON("plan.json", plan.Summary{RawTag: "raw-20261029", NewSpeakers: []string{"새 화자"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AppendNotes([]store.Note{{Key: keys[2], Batch: records[keys[2]].Batch, KRSha: records[keys[2]].KRSha, Model: "m", Note: "KR 這句有雙關"}}); err != nil {
		t.Fatal(err)
	}

	s := acceptState{root: root}
	mustLct(t, nil, "site", "--out", "work/site")
	var sg, wr, nt struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	for name, into := range map[string]any{"suggestions": &sg, "warnings": &wr, "notes": &nt} {
		if err := json.Unmarshal(readFile(t, filepath.Join(root, "work", "site", "review", name+".json")), into); err != nil {
			t.Fatal(err)
		}
	}
	for _, it := range sg.Items {
		if strings.HasPrefix(it.ID, "speaker/") {
			s.suggestion = it.ID
		}
	}
	for _, it := range nt.Items {
		s.note = it.ID
	}
	for _, it := range wr.Items {
		s.warnings = append(s.warnings, it.ID)
	}
	if s.suggestion == "" || s.note == "" || len(s.warnings) < 2 {
		t.Fatalf("the repo lacks items to accept: %+v", s)
	}
	return s
}

// acceptOut is "accept --json".
type acceptOut struct {
	Declined []string `json:"declined"`
	Accepted []string `json:"accepted"`
	Skipped  []string `json:"skipped"`
	RawTag   string   `json:"raw_tag"`
	TWCommit string   `json:"tw_commit"`
	Removed  []string `json:"removed"`
}

func readOptional(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func reviewCounts(t *testing.T, root string) (c struct{ Suggestions, Warnings, Notes int }) {
	t.Helper()
	meta := readSiteMeta(t, root)
	c.Suggestions, c.Warnings, c.Notes = meta.Review.Suggestions, meta.Review.Warnings, meta.Review.Notes
	return c
}

func TestAcceptWritesBothLists(t *testing.T) {
	s := acceptRepo(t)
	before := reviewCounts(t, s.root)
	var out acceptOut
	mustLct(t, &out, "accept", "--reason", "兩種都可以", "--date", "2026-11-03", s.suggestion, s.note, s.warnings[0])
	if !slices.Equal(out.Declined, []string{s.suggestion, s.note}) || !slices.Equal(out.Accepted, []string{s.warnings[0]}) ||
		len(out.Skipped) != 0 || out.RawTag != "raw-20261029" {
		t.Fatalf("accept = %+v", out)
	}
	declined := readOptional(t, s.root, "rules/declined.yaml")
	accepted := readOptional(t, s.root, "rules/accepted.yaml")
	for _, id := range []string{s.suggestion, s.note} {
		if !strings.Contains(declined, `- id: "`+id+`"`) {
			t.Errorf("declined.yaml lacks %s:\n%s", id, declined)
		}
	}
	if !strings.Contains(accepted, `- id: "`+s.warnings[0]+`"`) || strings.Contains(accepted, s.suggestion) {
		t.Errorf("accepted.yaml:\n%s", accepted)
	}
	if strings.Count(declined, "reason: 兩種都可以\n  date: 2026-11-03") != 2 || !strings.Contains(accepted, "reason: 兩種都可以\n  date: 2026-11-03") {
		t.Errorf("reason or date are not written as given:\n%s\n%s", declined, accepted)
	}

	// lct suggest skips the declined, and the site counts one less of each.
	var sug suggestOut
	mustLct(t, &sug, "suggest")
	if slices.Contains(sug.ids(), s.suggestion) || slices.Contains(sug.ids(), s.note) {
		t.Errorf("suggest still lists %v", sug.ids())
	}
	mustLct(t, nil, "site", "--out", "work/site")
	after := reviewCounts(t, s.root)
	if after.Suggestions != before.Suggestions-1 || after.Warnings != before.Warnings-1 || after.Notes != before.Notes-1 {
		t.Errorf("meta.review = %+v, before %+v: want one less of each", after, before)
	}
}

func TestAcceptRejectsUnknownIDs(t *testing.T) {
	s := acceptRepo(t)
	writeTestFile(t, s.root, "rules/accepted.yaml", "# keep\n")
	writeTestFile(t, s.root, "rules/declined.yaml", "# keep\n")
	const unknown = "warning/kana/00000000/00000000"
	code, stdout, stderr := lct(t, "accept", "--reason", "r", s.suggestion, unknown)
	if code != 1 || !strings.Contains(stdout+stderr, unknown) {
		t.Fatalf("exit code %d, want 1 naming %s\n%s%s", code, unknown, stdout, stderr)
	}
	if got := readOptional(t, s.root, "rules/accepted.yaml") + readOptional(t, s.root, "rules/declined.yaml"); got != "# keep\n# keep\n" {
		t.Errorf("a file changed:\n%s", got)
	}
}

func TestAcceptSkipsKnownAndDuplicates(t *testing.T) {
	s := acceptRepo(t)
	mustLct(t, nil, "accept", "--reason", "第一次", "--date", "2026-11-01", s.warnings[0])
	var out acceptOut
	mustLct(t, &out, "accept", "--reason", "第二次", "--date", "2026-11-03", s.warnings[0], s.warnings[1], s.warnings[1], s.suggestion)
	if !slices.Equal(out.Accepted, []string{s.warnings[1]}) || !slices.Equal(out.Declined, []string{s.suggestion}) ||
		!slices.Equal(out.Skipped, []string{s.warnings[0]}) {
		t.Fatalf("accept = %+v", out)
	}
	accepted := readOptional(t, s.root, "rules/accepted.yaml")
	if strings.Count(accepted, s.warnings[0]) != 1 || strings.Count(accepted, s.warnings[1]) != 1 {
		t.Errorf("accepted.yaml:\n%s", accepted)
	}
}

func TestAcceptFile(t *testing.T) {
	s := acceptRepo(t)
	writeTestFile(t, s.root, "work/ids.txt", "# 這次確認的\n\n"+s.suggestion+"\n   \n#"+s.note+"\n  "+s.warnings[1]+"  \r\n")
	var out acceptOut
	mustLct(t, &out, "accept", "--reason", "r", "--date", "2026-11-03", "--file", "work/ids.txt", s.warnings[0], s.warnings[1])
	if !slices.Equal(out.Accepted, []string{s.warnings[0], s.warnings[1]}) || !slices.Equal(out.Declined, []string{s.suggestion}) || len(out.Skipped) != 0 {
		t.Errorf("accept = %+v: the commented note must not be read", out)
	}
}

func TestAcceptTidy(t *testing.T) {
	s := acceptRepo(t)
	mustLct(t, nil, "accept", "--reason", "r", "--date", "2026-11-03", s.warnings[0])
	const stale = "warning/kana/00000000/00000000"
	path := filepath.Join(s.root, "rules", "accepted.yaml")
	data := readFile(t, path)
	stale1 := "- id: \"" + stale + "\"\n  reason: 過時\n  date: 2026-11-01\n"
	if err := os.WriteFile(path, append(data, stale1...), 0o644); err != nil {
		t.Fatal(err)
	}
	var out acceptOut
	mustLct(t, &out, "accept", "--tidy")
	if !slices.Equal(out.Removed, []string{stale}) {
		t.Errorf("removed = %v", out.Removed)
	}
	if got := readOptional(t, s.root, "rules/accepted.yaml"); got != string(data) {
		t.Errorf("accepted.yaml after --tidy:\n%s\nwant\n%s", got, data)
	}
}

func TestAcceptUsage(t *testing.T) {
	s := acceptRepo(t)
	was := readOptional(t, s.root, "rules/accepted.yaml") + readOptional(t, s.root, "rules/declined.yaml")
	for name, args := range map[string][]string{
		"no id":         {"accept", "--reason", "r"},
		"empty reason":  {"accept", "--reason", "", s.warnings[0]},
		"no reason":     {"accept", s.warnings[0]},
		"bad date":      {"accept", "--reason", "r", "--date", "2026-13-40", s.warnings[0]},
		"date format":   {"accept", "--reason", "r", "--date", "11/03", s.warnings[0]},
		"tidy with ids": {"accept", "--tidy", s.warnings[0]},
	} {
		if code, stdout, stderr := lct(t, args...); code != 1 {
			t.Errorf("%s: exit code %d, want 1\n%s%s", name, code, stdout, stderr)
		}
	}
	if got := readOptional(t, s.root, "rules/accepted.yaml") + readOptional(t, s.root, "rules/declined.yaml"); got != was {
		t.Error("a failed accept wrote a list")
	}
}
