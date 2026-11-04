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
	"github.com/tool-jx3/limbus-translate/internal/site"
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

// lookupOut is "accept --dry-run --json".
type lookupOut struct {
	Items []struct {
		ID        string `json:"id"`
		Kind      string `json:"kind"`
		List      string `json:"list"`
		Confirmed *struct {
			Reason string `json:"reason"`
			Date   string `json:"date"`
		} `json:"confirmed"`
		KR       string   `json:"kr"`
		ZH       string   `json:"zh"`
		Note     string   `json:"note"`
		Message  string   `json:"message"`
		Keys     []string `json:"keys"`
		Variants []struct {
			ZH    string   `json:"zh"`
			Count int      `json:"count"`
			Keys  []string `json:"keys"`
		} `json:"variants"`
		Pending int `json:"pending"`
	} `json:"items"`
	Unknown  []string `json:"unknown"`
	RawTag   string   `json:"raw_tag"`
	TWCommit string   `json:"tw_commit"`
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

func TestAcceptDryRunLooksItemsUp(t *testing.T) {
	s := acceptRepo(t)
	mustLct(t, nil, "accept", "--reason", "兩種都可以", "--date", "2026-11-03", s.warnings[1])
	listsBefore := readOptional(t, s.root, "rules/accepted.yaml") + readOptional(t, s.root, "rules/declined.yaml")

	var out lookupOut
	mustLct(t, &out, "accept", "--dry-run", s.suggestion, s.note, s.warnings[0], s.warnings[1])
	if len(out.Items) != 4 || len(out.Unknown) != 0 || out.RawTag != "raw-20261029" {
		t.Fatalf("lookup = %+v", out)
	}
	by := map[string]int{}
	for i, it := range out.Items {
		by[it.ID] = i
	}
	sg, nt, w0, w1 := out.Items[by[s.suggestion]], out.Items[by[s.note]], out.Items[by[s.warnings[0]]], out.Items[by[s.warnings[1]]]
	if sg.Kind != "speaker" || sg.List != "declined" || sg.Confirmed != nil || sg.Keys == nil {
		t.Errorf("suggestion = %+v", sg)
	}
	if nt.Kind != "note" || nt.List != "declined" || nt.Note != "KR 這句有雙關" || len(nt.Keys) != 1 || nt.KR == "" || nt.ZH == "" {
		t.Errorf("note = %+v", nt)
	}
	if w0.Kind != "warning" || w0.List != "accepted" || w0.Confirmed != nil || w0.Message == "" || w0.KR == "" || w0.ZH == "" || len(w0.Keys) == 0 {
		t.Errorf("warning = %+v", w0)
	}
	if w1.Confirmed == nil || w1.Confirmed.Reason != "兩種都可以" || w1.Confirmed.Date != "2026-11-03" {
		t.Errorf("confirmed warning = %+v", w1)
	}
	if got := readOptional(t, s.root, "rules/accepted.yaml") + readOptional(t, s.root, "rules/declined.yaml"); got != listsBefore {
		t.Errorf("--dry-run wrote a list:\n%s", got)
	}

	// An unknown id is listed and makes the exit code 1; the known one is still printed.
	const unknown = "warning/kana/00000000/00000000"
	code, stdout, stderr := lct(t, "--json", "accept", "--dry-run", s.warnings[0], unknown)
	if code != 1 {
		t.Fatalf("exit code %d, want 1\n%s%s", code, stdout, stderr)
	}
	var partial lookupOut
	if err := json.Unmarshal([]byte(stdout), &partial); err != nil {
		t.Fatalf("%v\n%s", err, stdout)
	}
	if len(partial.Items) != 1 || partial.Items[0].ID != s.warnings[0] || !slices.Equal(partial.Unknown, []string{unknown}) {
		t.Errorf("lookup = %+v", partial)
	}

	// The text output names the same ids.
	code, stdout, _ = lct(t, "accept", "--dry-run", s.warnings[0], s.note)
	if code != 0 || !strings.Contains(stdout, s.warnings[0]) || !strings.Contains(stdout, s.note) {
		t.Errorf("text output (exit %d):\n%s", code, stdout)
	}
}

func TestAcceptDryRunFile(t *testing.T) {
	s := acceptRepo(t)
	writeTestFile(t, s.root, "work/ids.txt", s.warnings[0]+"\n")
	var out lookupOut
	mustLct(t, &out, "accept", "--dry-run", "--file", "work/ids.txt", s.note)
	if len(out.Items) != 2 {
		t.Errorf("lookup = %+v", out)
	}
	if code, _, _ := lct(t, "accept", "--dry-run"); code != 1 {
		t.Errorf("--dry-run with no id: exit code %d, want 1", code)
	}
	if code, _, _ := lct(t, "accept", "--dry-run", "--tidy"); code != 1 {
		t.Errorf("--dry-run with --tidy: exit code %d, want 1", code)
	}
}

func TestAcceptSkipsAnIDTheBuildNoLongerHas(t *testing.T) {
	s := acceptRepo(t)
	// The build drops a declined group of inconsistent translations, so the id is in the list only.
	const dropped = "inconsistent/other/name/00000000"
	writeTestFile(t, s.root, "rules/declined.yaml", "- id: \""+dropped+"\"\n  reason: 都可以\n  date: 2026-11-01\n")
	var out acceptOut
	mustLct(t, &out, "accept", "--reason", "r", "--date", "2026-11-03", dropped, s.warnings[0])
	if !slices.Equal(out.Skipped, []string{dropped}) || !slices.Equal(out.Accepted, []string{s.warnings[0]}) {
		t.Errorf("accept = %+v", out)
	}
	var look lookupOut
	mustLct(t, &look, "accept", "--dry-run", dropped)
	if len(look.Unknown) != 0 || len(look.Items) != 1 || look.Items[0].List != "declined" || look.Items[0].Confirmed == nil || look.Items[0].Confirmed.Reason != "都可以" {
		t.Errorf("lookup = %+v", look)
	}
}

func TestAcceptTidyNamesTheBuild(t *testing.T) {
	acceptRepo(t)
	code, stdout, stderr := lct(t, "--json", "accept", "--tidy")
	if code != 0 {
		t.Fatalf("exit code %d\n%s%s", code, stdout, stderr)
	}
	// The test repo has no commit in the tw worktree, so tw_commit is empty but present.
	if !strings.Contains(stdout, `"raw_tag": "raw-20261029"`) || !strings.Contains(stdout, `"tw_commit"`) {
		t.Errorf("tidy output lacks the build it was checked against:\n%s", stdout)
	}
}

// The dry run shows each variant's JP/EN groups and the class of the group,
// in the JSON and in the text, so the review skills see them.
func TestAcceptDryRunShowsJPEN(t *testing.T) {
	jp, en := "末弟", "Little Brother"
	en2 := "Little Sister"
	rep := lookupReport{Items: []lookupItem{{
		ID: "inconsistent/identity/title/af7bae96", Kind: "inconsistent", List: "declined", KR: "중지 작은 아우", Keys: []string{"A.json|/a", "A.json|/b"},
		JPENClass: site.JPENExplained,
		Variants: []site.Variant{
			{ZH: "中指 幼弟", Count: 1, Keys: []string{"A.json|/a"}, JPEN: []site.JPENGroup{{JP: &jp, EN: &en, Count: 1, Keys: []string{"A.json|/a"}}}},
			{ZH: "中指 幼妹", Count: 1, Keys: []string{"A.json|/b"}, JPEN: []site.JPENGroup{{EN: &en2, Count: 1, Keys: []string{"A.json|/b"}}}},
		},
	}}, Unknown: []string{}}
	data, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"jpen_class":"explained"`, `"jpen":[{"jp":"末弟","en":"Little Brother","count":1`, `"jpen":[{"en":"Little Sister"`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("JSON lacks %s:\n%s", want, data)
		}
	}
	text := rep.String()
	for _, want := range []string{"JP/EN: explained", "JP 末弟／EN Little Brother（1）", "JP —／EN Little Sister（1）"} {
		if !strings.Contains(text, want) {
			t.Errorf("text lacks %q:\n%s", want, text)
		}
	}
}
