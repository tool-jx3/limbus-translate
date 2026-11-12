package main

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/itemid"
	"github.com/tool-jx3/limbus-translate/internal/plan"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/scope"
	"github.com/tool-jx3/limbus-translate/internal/site"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/suggest"
	"github.com/tool-jx3/limbus-translate/internal/units"
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
	Empty    []string `json:"empty"`
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
		Title    string   `json:"title"`
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
		Units   []struct {
			ID string `json:"id"`
		} `json:"units"`
		OpenUnits *openUnits `json:"open_units"`
	} `json:"items"`
	Unknown           []string `json:"unknown"`
	UnknownTermGroups []string `json:"unknown_term_groups"`
	RawTag            string   `json:"raw_tag"`
	TWCommit          string   `json:"tw_commit"`
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

// --file is read and its lines that are no item are printed back, so it names
// only a file of the repo outside .git, links resolved: the skill that runs
// lct accept --dry-run unattended in CI may read nothing its tools keep closed.
func TestAcceptFileStaysInTheRepo(t *testing.T) {
	s := acceptRepo(t)
	const secret = "ghs_notarealtoken0123456789abcdef"
	outside := filepath.Join(t.TempDir(), "environ")
	if err := os.WriteFile(outside, []byte(secret+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, s.root, ".git/lct-secret", secret+"\n")
	if err := os.Symlink(outside, filepath.Join(s.root, "work", "link.txt")); err != nil {
		t.Fatal(err)
	}
	up, err := filepath.Rel(s.root, outside)
	if err != nil {
		t.Fatal(err)
	}
	was := readOptional(t, s.root, "rules/accepted.yaml") + readOptional(t, s.root, "rules/declined.yaml")
	for name, file := range map[string]string{
		"an absolute path outside": outside,
		"a relative path outside":  up,
		"a link that leads out":    "work/link.txt",
		"the folder of git":        ".git/lct-secret",
	} {
		for _, mode := range [][]string{{"--dry-run"}, {"--reason", "r"}} {
			args := append(append([]string{"accept"}, mode...), "--file", file, s.warnings[0])
			code, stdout, stderr := lct(t, args...)
			if code != 1 || strings.Contains(stdout+stderr, secret) || !strings.Contains(stderr, "--file") {
				t.Errorf("%s %v: exit code %d, want 1 refusing --file without its lines\n%s%s", name, mode, code, stdout, stderr)
			}
		}
	}
	if got := readOptional(t, s.root, "rules/accepted.yaml") + readOptional(t, s.root, "rules/declined.yaml"); got != was {
		t.Error("a refused --file wrote a list")
	}
}

// --dry-run turned off again is refused: the grant of lct accept --dry-run to
// a skill reads, and must not write.
func TestAcceptDryRunCannotBeTurnedOff(t *testing.T) {
	s := acceptRepo(t)
	was := readOptional(t, s.root, "rules/accepted.yaml") + readOptional(t, s.root, "rules/declined.yaml")
	for _, args := range [][]string{
		{"accept", "--dry-run=false", "--reason", "r", s.warnings[0]},
		{"accept", "--dry-run", "--reason", "r", "--dry-run=0", s.warnings[0]},
	} {
		if code, stdout, stderr := lct(t, args...); code != 1 {
			t.Errorf("%v: exit code %d, want 1\n%s%s", args, code, stdout, stderr)
		}
	}
	if got := readOptional(t, s.root, "rules/accepted.yaml") + readOptional(t, s.root, "rules/declined.yaml"); got != was {
		t.Error("lct accept with --dry-run turned off wrote a list")
	}
}

// lct accept declines a provisional term like any rule suggestion: the id
// goes to rules/declined.yaml, the dry run knows it, and the batches stop
// carrying the term.
func TestAcceptDeclinesANewTerm(t *testing.T) {
	s := acceptRepo(t)
	st := store.Open(filepath.Join(s.root, ".work", "tw"))
	records, err := st.LoadRecords()
	if err != nil {
		t.Fatal(err)
	}
	key := slices.Sorted(maps.Keys(records))[0]
	if _, err := st.AppendTerms([]store.ProvisionalTerm{
		{KR: "우레", ZH: "雷鳴", Key: key, Batch: "b1", RawTag: "raw-20261029", Time: "2026-11-05T00:00:00Z"},
		{KR: "우레", ZH: "驚雷", Key: key, Batch: "b2", RawTag: "raw-20261029", Time: "2026-11-05T00:00:00Z", Conflict: true},
	}); err != nil {
		t.Fatal(err)
	}
	const id = "term-new/우레"

	var look lookupOut
	mustLct(t, &look, "accept", "--dry-run", id)
	if len(look.Items) != 1 || look.Items[0].Kind != "term-new" || look.Items[0].List != "declined" || len(look.Unknown) != 0 {
		t.Fatalf("dry run = %+v, want the term-new item that goes to declined", look)
	}
	if it := look.Items[0]; it.KR != "우레" || it.ZH != "雷鳴" {
		t.Errorf("dry run KR %q TW %q, want the term 우레 → 雷鳴", it.KR, it.ZH)
	}
	mustLct(t, nil, "site", "--out", "work/site")
	before := reviewCounts(t, s.root)
	r, err := rules.Load(s.root)
	if err != nil {
		t.Fatal(err)
	}
	if _, got, err := activeProvisionalRecords(s.root, st, r); err != nil || len(got) != 1 || got[0].KR != "우레" {
		t.Fatalf("before: active = %+v, %v", got, err)
	}

	var out acceptOut
	mustLct(t, &out, "accept", "--reason", "譯法還沒定", "--date", "2026-11-05", id)
	if !slices.Equal(out.Declined, []string{id}) {
		t.Fatalf("accept = %+v", out)
	}
	if declined := readOptional(t, s.root, "rules/declined.yaml"); !strings.Contains(declined, `- id: "`+id+`"`) {
		t.Errorf("declined.yaml lacks %s:\n%s", id, declined)
	}
	if _, got, err := activeProvisionalRecords(s.root, st, r); err != nil || len(got) != 0 {
		t.Errorf("after: active = %+v, %v; want none", got, err)
	}
	var sug suggestOut
	mustLct(t, &sug, "suggest", "--kind", "term-new")
	if len(sug.Suggestions) != 0 || sug.Excluded != 1 {
		t.Errorf("suggest lists %v, %d excluded; want none, the declined one excluded", sug.ids(), sug.Excluded)
	}

	// The review page lists it as declined, as it does any declined
	// suggestion, and counts one less waiting.
	mustLct(t, nil, "site", "--out", "work/site")
	type pageItem struct {
		ID   string `json:"id"`
		Term *struct {
			KR string `json:"kr"`
			ZH string `json:"zh"`
		} `json:"term"`
		Declined *struct {
			Reason string `json:"reason"`
			Date   string `json:"date"`
		} `json:"declined"`
	}
	var page struct {
		Items []pageItem `json:"items"`
	}
	if err := json.Unmarshal(readFile(t, filepath.Join(s.root, "work", "site", "review", "suggestions.json")), &page); err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(page.Items, func(it pageItem) bool { return it.ID == id })
	if i < 0 {
		t.Fatalf("suggestions.json lacks %s", id)
	}
	if it := page.Items[i]; it.Declined == nil || it.Declined.Reason != "譯法還沒定" || it.Declined.Date != "2026-11-05" || it.Term == nil || it.Term.ZH != "雷鳴" {
		t.Errorf("%s = %+v, want it declined with its term", id, it)
	}
	if after := reviewCounts(t, s.root); after.Suggestions != before.Suggestions-1 {
		t.Errorf("meta.review = %+v, before %+v: want one suggestion less", after, before)
	}
}

// termGroupUnit is a unit of a term group as the review page has it.
type termGroupUnit struct {
	ID  string `json:"id"`
	Key string `json:"key"`
	KR  string `json:"kr"`
	ZH  string `json:"zh"`
}

// termGroupRepo is acceptRepo with a glossary entry the fixture never renders
// (열상 → 絕無此譯): it returns the id of its term group on the review page and
// the units of the group.
func termGroupRepo(t *testing.T) (s acceptState, group string, units []termGroupUnit) {
	t.Helper()
	s = acceptRepo(t)
	glossary := filepath.Join(s.root, "glossary", "terms.yaml")
	data, err := os.ReadFile(glossary)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(glossary, append(data, "\n\n- kr: 열상\n  zh: 絕無此譯\n  status: approved\n"...), 0o644); err != nil {
		t.Fatal(err)
	}
	mustLct(t, nil, "site", "--out", "work/site")
	var wr struct {
		Items []struct {
			ID    string          `json:"id"`
			Code  string          `json:"code"`
			KR    string          `json:"kr"`
			Units []termGroupUnit `json:"units"`
		} `json:"items"`
	}
	if err := json.Unmarshal(readFile(t, filepath.Join(s.root, "work", "site", "review", "warnings.json")), &wr); err != nil {
		t.Fatal(err)
	}
	for _, w := range wr.Items {
		if w.Code == "term-missing" && strings.HasPrefix(w.ID, "termgroup/") && w.KR == "열상" {
			group, units = w.ID, w.Units
		}
	}
	if group == "" || len(units) == 0 {
		t.Fatalf("no term group in %+v", wr.Items)
	}
	return s, group, units
}

// lct accept expands a termgroup/… id into the warning ids of the units of the
// group that are not accepted yet, and writes those.
func TestAcceptExpandsATermGroup(t *testing.T) {
	s, group, units := termGroupRepo(t)
	var unitIDs []string
	for _, u := range units {
		unitIDs = append(unitIDs, u.ID)
	}

	var look lookupOut
	before := readOptional(t, s.root, "rules/accepted.yaml")
	mustLct(t, &look, "accept", "--dry-run", group)
	if len(look.Items) != 1 || len(look.Unknown) != 0 || readOptional(t, s.root, "rules/accepted.yaml") != before ||
		len(look.Items[0].Units) != len(unitIDs) || look.Items[0].Units[0].ID != unitIDs[0] {
		t.Fatalf("dry run = %+v, ids %v, before %q after %q", look, unitIDs, before, readOptional(t, s.root, "rules/accepted.yaml"))
	}
	var out acceptOut
	mustLct(t, &out, "accept", "--reason", "簡稱即可", "--date", "2026-11-05", group)
	if !slices.Equal(out.Accepted, unitIDs) {
		t.Fatalf("accepted = %v, want %v", out.Accepted, unitIDs)
	}
	accepted := readOptional(t, s.root, "rules/accepted.yaml")
	for _, id := range unitIDs {
		if !strings.HasPrefix(id, "warning/term-missing/") || !strings.Contains(accepted, `- id: "`+id+`"`) {
			t.Errorf("accepted.yaml lacks %s:\n%s", id, accepted)
		}
	}
	if strings.Contains(accepted, "termgroup/") {
		t.Errorf("accepted.yaml holds the group id:\n%s", accepted)
	}

	// --tidy keeps the entries of the units of a group: they are warnings of the build.
	var tidy acceptOut
	mustLct(t, &tidy, "accept", "--tidy")
	if len(tidy.Removed) != 0 || readOptional(t, s.root, "rules/accepted.yaml") != accepted {
		t.Errorf("tidy removed %v:\n%s", tidy.Removed, readOptional(t, s.root, "rules/accepted.yaml"))
	}

	// Everything is accepted now: the group is a no-op.
	var again acceptOut
	mustLct(t, &again, "accept", "--reason", "again", group)
	if len(again.Accepted) != 0 || !slices.Equal(again.Empty, []string{group}) {
		t.Errorf("second accept = %+v", again)
	}
	if after := readOptional(t, s.root, "rules/accepted.yaml"); after != accepted {
		t.Errorf("a group without unaccepted units wrote:\n%s", after)
	}
}

// The dry run knows the id of a unit of a term group, which the review page
// lets a person copy to keep that unit alone, and says once it is accepted.
func TestAcceptDryRunKnowsTheUnitsOfATermGroup(t *testing.T) {
	s, _, units := termGroupRepo(t)
	u := units[0]
	var look lookupOut
	mustLct(t, &look, "accept", "--dry-run", u.ID)
	if len(look.Unknown) != 0 || len(look.Items) != 1 {
		t.Fatalf("dry run of %s = %+v, want the unit", u.ID, look)
	}
	it := look.Items[0]
	if it.ID != u.ID || it.Kind != "warning" || it.List != "accepted" || it.Confirmed != nil ||
		it.KR != u.KR || it.ZH != u.ZH || !slices.Equal(it.Keys, []string{u.Key}) || it.Message != "這句譯文沒有用詞彙表的譯名「絕無此譯」（열상）。" {
		t.Errorf("item = %+v, want the open warning of unit %+v", it, u)
	}

	mustLct(t, nil, "accept", "--reason", "這裡不譯流血", "--date", "2026-11-05", u.ID)
	if accepted := readOptional(t, s.root, "rules/accepted.yaml"); !strings.Contains(accepted, `- id: "`+u.ID+`"`) {
		t.Fatalf("accepted.yaml lacks %s:\n%s", u.ID, accepted)
	}
	look = lookupOut{}
	mustLct(t, &look, "accept", "--dry-run", u.ID)
	if len(look.Unknown) != 0 || len(look.Items) != 1 || look.Items[0].Confirmed == nil || look.Items[0].Confirmed.Reason != "這裡不譯流血" {
		t.Errorf("dry run after accept = %+v, want the unit confirmed", look)
	}
}

// Units of one KR text and one translation share an id: its item lists every
// one of them, as lct accept of the id accepts them all. The item of a unit
// says what that one translation lacks, not what the group's do.
func TestAddUnitLookupsListsEveryUnitOfAnID(t *testing.T) {
	const unitMessage = "這句譯文沒有用詞彙表的譯名「流血」（열상）。"
	w := site.WarningItem{ID: "termgroup/aaaaaaaa/bbbbbbbb", Code: "term-missing", Message: "這些譯文沒有用詞彙表的譯名「流血」（열상）。", Term: &site.TermRef{KR: "열상", ZH: "流血"}, Units: []site.TermUnit{
		{Key: "A.json|/dataList/0/name", ID: "warning/term-missing/11111111/22222222", KR: "열상", ZH: "裂傷"},
		{Key: "B.json|/dataList/0/name", ID: "warning/term-missing/11111111/22222222", KR: "열상", ZH: "裂傷"},
		{Key: "C.json|/dataList/0/name", ID: "warning/term-missing/33333333/22222222", KR: "열상이다", ZH: "裂傷", Note: "刻意"},
	}}
	items := map[string]lookupItem{}
	addUnitLookups(items, w, map[string]bool{"B.json|/dataList/0/name": true})
	shared := items["warning/term-missing/11111111/22222222"]
	if !slices.Equal(shared.Keys, []string{"A.json|/dataList/0/name", "B.json|/dataList/0/name"}) || shared.Pending != 1 ||
		shared.Kind != "warning" || shared.List != listAccepted || shared.KR != "열상" || shared.ZH != "裂傷" || shared.Message != unitMessage {
		t.Errorf("shared id = %+v, want both units, one waiting", shared)
	}
	if own := items["warning/term-missing/33333333/22222222"]; !slices.Equal(own.Keys, []string{"C.json|/dataList/0/name"}) || own.Pending != 0 || own.Note != "刻意" || own.Message != unitMessage {
		t.Errorf("own id = %+v", own)
	}
}

// A term group id may carry the number of its units not accepted yet that the
// review page showed: termgroup/<a>/<b>@<N>. lct accept writes the units when
// the last build has that many open, and refuses the id, writing nothing,
// when it has another number; the dry run shows the comparison. Without @N
// the id is as before (TestAcceptExpandsATermGroup).
func TestAcceptTermGroupCount(t *testing.T) {
	s, group, units := termGroupRepo(t)
	n := len(units)
	at := func(k int) string { return fmt.Sprintf("%s@%d", group, k) }
	lists := func() string {
		return readOptional(t, s.root, "rules/accepted.yaml") + readOptional(t, s.root, "rules/declined.yaml")
	}
	before := lists()

	// Another number: refused with both numbers, and nothing is written, not
	// the other id either.
	code, stdout, stderr := lct(t, "accept", "--reason", "簡稱即可", "--date", "2026-11-05", s.suggestion, at(n+1))
	if code != 1 || !strings.Contains(stderr, at(n+1)) || !strings.Contains(stderr, fmt.Sprintf("the review page showed %d", n+1)) ||
		!strings.Contains(stderr, fmt.Sprintf("the last build has %d", n)) {
		t.Errorf("accept %s: exit code %d, want 1 naming %d and %d\n%s%s", at(n+1), code, n+1, n, stdout, stderr)
	}
	if lists() != before {
		t.Errorf("a refused count wrote:\n%s", lists())
	}

	// The dry run shows the comparison, and exits 1 as accept would refuse.
	code, stdout, stderr = lct(t, "--json", "accept", "--dry-run", at(n+1))
	var look lookupOut
	if err := json.Unmarshal([]byte(stdout), &look); err != nil {
		t.Fatalf("%v\n%s%s", err, stdout, stderr)
	}
	if code != 1 || len(look.Items) != 1 || look.Items[0].ID != at(n+1) || look.Items[0].OpenUnits == nil ||
		*look.Items[0].OpenUnits != (openUnits{Page: n + 1, Now: n, Match: false}) || len(look.Items[0].Units) != n {
		t.Errorf("dry run of %s: exit code %d, %+v", at(n+1), code, look)
	}
	code, stdout, _ = lct(t, "accept", "--dry-run", at(n+1))
	if code != 1 || !strings.Contains(stdout, fmt.Sprintf("the review page showed %d, the last build has %d", n+1, n)) {
		t.Errorf("text dry run (exit %d):\n%s", code, stdout)
	}
	look = lookupOut{}
	mustLct(t, &look, "accept", "--dry-run", at(n))
	if len(look.Items) != 1 || look.Items[0].OpenUnits == nil || *look.Items[0].OpenUnits != (openUnits{Page: n, Now: n, Match: true}) {
		t.Errorf("dry run of %s = %+v, want a match", at(n), look)
	}
	if lists() != before {
		t.Errorf("the dry run wrote:\n%s", lists())
	}

	// The same number: the units are written, as for the group id.
	var want []string
	for _, u := range units {
		if !slices.Contains(want, u.ID) {
			want = append(want, u.ID)
		}
	}
	var out acceptOut
	mustLct(t, &out, "accept", "--reason", "簡稱即可", "--date", "2026-11-05", at(n))
	if !slices.Equal(out.Accepted, want) {
		t.Errorf("accepted = %v, want %v", out.Accepted, want)
	}
	accepted := readOptional(t, s.root, "rules/accepted.yaml")
	if strings.Contains(accepted, "termgroup/") || strings.Contains(accepted, "@") {
		t.Errorf("accepted.yaml holds the group id:\n%s", accepted)
	}

	// None is open now: the page's number is stale, and 0 is a no-op.
	if code, stdout, stderr := lct(t, "accept", "--reason", "again", at(n)); code != 1 || !strings.Contains(stderr, "the last build has 0") {
		t.Errorf("accept %s again: exit code %d, want 1\n%s%s", at(n), code, stdout, stderr)
	}
	var again acceptOut
	mustLct(t, &again, "accept", "--reason", "again", at(0))
	if len(again.Accepted) != 0 || !slices.Equal(again.Empty, []string{at(0)}) {
		t.Errorf("accept %s = %+v, want a no-op", at(0), again)
	}
	if readOptional(t, s.root, "rules/accepted.yaml") != accepted {
		t.Errorf("a stale or empty group wrote:\n%s", readOptional(t, s.root, "rules/accepted.yaml"))
	}

	// A count that is no number is refused, in both modes.
	for _, bad := range []string{group + "@", group + "@x", group + "@-1", group + "@1@2", group + "@ 1"} {
		for _, mode := range [][]string{{"--dry-run"}, {"--reason", "r"}} {
			args := append(append([]string{"accept"}, mode...), bad)
			if code, stdout, stderr := lct(t, args...); code != 1 || !strings.Contains(stderr, "@") {
				t.Errorf("%v: exit code %d, want 1\n%s%s", args, code, stdout, stderr)
			}
		}
	}
	if readOptional(t, s.root, "rules/accepted.yaml") != accepted {
		t.Error("a bad count wrote a list")
	}
}

// Only a term group id takes @N; an @ in any other id is part of it.
func TestParseGivenID(t *testing.T) {
	const group = "termgroup/aaaaaaaa/bbbbbbbb"
	for _, c := range []struct {
		id   string
		want givenID
	}{
		{group, givenID{ID: group, Item: group}},
		{group + "@3", givenID{ID: group + "@3", Item: group, Count: 3, HasCount: true}},
		{group + "@0", givenID{ID: group + "@0", Item: group, Count: 0, HasCount: true}},
		{"note/A.json|/a@b/12345678", givenID{ID: "note/A.json|/a@b/12345678", Item: "note/A.json|/a@b/12345678"}},
		{"warning/kana/11111111/22222222@3", givenID{ID: "warning/kana/11111111/22222222@3", Item: "warning/kana/11111111/22222222@3"}},
	} {
		if got, err := parseGivenID(c.id); err != nil || got != c.want {
			t.Errorf("parseGivenID(%q) = %+v, %v; want %+v", c.id, got, err, c.want)
		}
	}
	for _, id := range []string{group + "@", group + "@x", group + "@-1", group + "@+1", group + "@1@2", "termgroup/@3", group + "@99999999999999999999"} {
		if got, err := parseGivenID(id); err == nil {
			t.Errorf("parseGivenID(%q) = %+v, want an error", id, got)
		}
	}
}

// The usage of lct accept says what each list of its report holds, empty
// included: a term group id none of whose units was left to accept.
func TestAcceptUsageNamesTheReport(t *testing.T) {
	code, stdout, stderr := lct(t, "help")
	if code != 0 {
		t.Fatalf("lct help: exit code %d\n%s%s", code, stdout, stderr)
	}
	var line string
	for _, l := range strings.Split(stdout, "\n") {
		if f := strings.Fields(l); len(f) > 0 && f[0] == "accept" {
			line = l
		}
	}
	for _, want := range []string{
		"--json prints declined and accepted (the ids written to each list), skipped (those confirmed before)",
		"empty (the term group ids none of whose units was left to accept: nothing is written for them)",
	} {
		if !strings.Contains(line, want) {
			t.Errorf("the usage of accept lacks %q:\n%s", want, line)
		}
	}
}

// A term group id that is in no item of the last build (the glossary or the
// translations changed since the review page was made) is not a plain unknown
// id: lct accept and its dry run say that the group is not in the current
// review data, which comes from .work/raw, .work/tw and glossary/, and that
// the worktrees want the page's data or the page a refresh. Exit
// code 1, and nothing is written.
func TestAcceptTermGroupNotInTheBuild(t *testing.T) {
	s := acceptRepo(t)
	const group = "termgroup/00000000/00000000"
	const other = "warning/kana/00000000/00000000"
	lists := func() string {
		return readOptional(t, s.root, "rules/accepted.yaml") + readOptional(t, s.root, "rules/declined.yaml")
	}
	before := lists()
	for _, id := range []string{group + "@2", group} {
		code, stdout, stderr := lct(t, "accept", "--reason", "r", "--date", "2026-11-05", s.warnings[0], id)
		if code != 1 || !strings.Contains(stderr, id) || !strings.Contains(stderr, "not in the current review data") ||
			!termGroupGoneSaid(stderr) || strings.Contains(stderr, "no item of the last build") {
			t.Errorf("accept %s: exit code %d, want 1 saying the group is not in the review data\n%s%s", id, code, stdout, stderr)
		}
	}
	if lists() != before {
		t.Errorf("a term group not in the build wrote:\n%s", lists())
	}

	// The dry run lists it among the unknown ids and apart, and says so.
	code, stdout, stderr := lct(t, "--json", "accept", "--dry-run", group+"@2", other)
	var look lookupOut
	if err := json.Unmarshal([]byte(stdout), &look); err != nil {
		t.Fatalf("%v\n%s%s", err, stdout, stderr)
	}
	if code != 1 || !slices.Equal(look.Unknown, []string{group + "@2", other}) || !slices.Equal(look.UnknownTermGroups, []string{group + "@2"}) {
		t.Errorf("dry run: exit code %d, %+v", code, look)
	}
	code, stdout, _ = lct(t, "accept", "--dry-run", group+"@2", other)
	gone, plain, found := strings.Cut(stdout, "no item of the last build:")
	if code != 1 || !found || !strings.Contains(stdout, "not in the current review data") || !termGroupGoneSaid(stdout) ||
		!strings.Contains(gone+plain, group+"@2") || strings.Contains(plain, group) || !strings.Contains(plain, other) {
		t.Errorf("text dry run (exit %d):\n%s", code, stdout)
	}
	if lists() != before {
		t.Errorf("the dry run wrote:\n%s", lists())
	}
}

// termGroupGoneSaid reports whether out gives the cause and the way out of a
// term group that is not in the review data built here: the data comes from
// .work/raw, .work/tw and glossary/, the raw_tag and tw_commit can be compared
// with the page's, and the way out is the page's data or a fresh page. It does
// not send the person to lct build, which writes none of that data.
func termGroupGoneSaid(out string) bool {
	return strings.Contains(out, ".work/raw, .work/tw and glossary/") && strings.Contains(out, "raw_tag and tw_commit") &&
		strings.Contains(out, "update the worktrees to the page's raw tag and tw commit, or refresh the review page and paste again") &&
		!strings.Contains(out, "lct build")
}

// scopeKR, scopeZH and scopeID are the entry and the proposal of scopeRepo: 다
// as a substring stands inside a word in three units of the story, none of
// which says 絕無此譯, so the page proposes to match it as a word.
const (
	scopeKR = "다"
	scopeZH = "絕無此譯"
)

var scopeID = itemid.Scope(scopeKR, scopeZH, scope.KindWord)

// scopeProposal is a proposal of review/scope.json, as the tests read it.
type scopeProposal struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	WarnRemoved int    `json:"warn_removed"`
	Declined    *struct {
		Reason string `json:"reason"`
		Date   string `json:"date"`
	} `json:"declined"`
}

// scopeRepo is acceptRepo with the entry 다 → 絕無此譯 (substring) in the
// glossary, two more lines of the story translated (the build left them to
// EN) and the site written: it returns the proposals of review/scope.json.
func scopeRepo(t *testing.T) (acceptState, []scopeProposal) {
	t.Helper()
	s := acceptRepo(t)
	st := store.Open(filepath.Join(s.root, ".work", "tw"))
	records, err := st.LoadRecords()
	if err != nil {
		t.Fatal(err)
	}
	for pointer, text := range map[string][2]string{
		"/dataList/3/content": {"<오늘도 아무 일 없이 지나가길 바랄 뿐이다.>", "<只希望今天也平安無事。>"},
		"/dataList/4/content": {"{0}번 승객은 이미 배에서 내린 뒤였다.", "{0}號乘客已經下船了。"},
	} {
		key := units.Key("StoryData/S001A.json", pointer)
		rec := records[key]
		rec.Key, rec.KRSha, rec.ZH, rec.Origin = key, units.KRSha(text[0]), text[1], store.OriginParatranz
		records[key] = rec
	}
	if err := st.SaveRecords(records); err != nil {
		t.Fatal(err)
	}
	appendTestFile(t, s.root, "glossary/terms.yaml", "\n- kr: "+scopeKR+"\n  zh: "+scopeZH+"\n  status: approved\n  match: substring\n")
	mustLct(t, nil, "site", "--out", "work/site")
	return s, readScopeProposals(t, s.root)
}

func readScopeProposals(t *testing.T, root string) []scopeProposal {
	t.Helper()
	var rep struct {
		Proposals []scopeProposal `json:"proposals"`
	}
	if err := json.Unmarshal(readFile(t, filepath.Join(root, "work", "site", "review", "scope.json")), &rep); err != nil {
		t.Fatal(err)
	}
	return rep.Proposals
}

// A scope proposal is declined as a rule suggestion is: lct accept writes its
// id to rules/declined.yaml, the dry run shows it, and the review page lists
// it as declined and counts one less waiting.
func TestAcceptDeclinesAScopeProposal(t *testing.T) {
	s, props := scopeRepo(t)
	i := slices.IndexFunc(props, func(p scopeProposal) bool { return p.ID == scopeID })
	if i < 0 || props[i].WarnRemoved != 3 || props[i].Declined != nil {
		t.Fatalf("scope.json proposals = %+v, want %s removing 3 warnings", props, scopeID)
	}
	before := readSiteMeta(t, s.root).Review

	var look lookupOut
	mustLct(t, &look, "accept", "--dry-run", scopeID)
	if len(look.Items) != 1 || len(look.Unknown) != 0 {
		t.Fatalf("dry run = %+v, want the proposal", look)
	}
	if it := look.Items[0]; it.Kind != "scope" || it.List != "declined" || it.Confirmed != nil || it.KR != scopeKR || it.ZH != scopeZH ||
		!strings.Contains(it.Title, "match: word") {
		t.Errorf("item = %+v, want the open proposal of %s → %s that goes to declined", it, scopeKR, scopeZH)
	}

	var out acceptOut
	mustLct(t, &out, "accept", "--reason", "維持 substring", "--date", "2026-11-06", scopeID)
	if !slices.Equal(out.Declined, []string{scopeID}) || len(out.Accepted) != 0 {
		t.Fatalf("accept = %+v", out)
	}
	if declined := readOptional(t, s.root, "rules/declined.yaml"); !strings.Contains(declined, `- id: "`+scopeID+`"`) {
		t.Errorf("declined.yaml lacks %s:\n%s", scopeID, declined)
	}
	if issues := suggest.LintDeclined(s.root); len(issues) != 0 {
		t.Errorf("lint of declined.yaml: %v", issues)
	}

	mustLct(t, nil, "site", "--out", "work/site")
	props = readScopeProposals(t, s.root)
	i = slices.IndexFunc(props, func(p scopeProposal) bool { return p.ID == scopeID })
	if i < 0 || props[i].Declined == nil || props[i].Declined.Reason != "維持 substring" || props[i].Declined.Date != "2026-11-06" {
		t.Errorf("scope.json proposals = %+v, want %s declined", props, scopeID)
	}
	if after := readSiteMeta(t, s.root).Review; after.Scope != before.Scope-1 || before.Scope == 0 {
		t.Errorf("meta.review = %+v, before %+v: want one scope proposal less", after, before)
	}
	look = lookupOut{}
	mustLct(t, &look, "accept", "--dry-run", scopeID)
	if len(look.Items) != 1 || look.Items[0].Confirmed == nil || look.Items[0].Confirmed.Reason != "維持 substring" {
		t.Errorf("dry run after accept = %+v, want the proposal confirmed", look)
	}
	var again acceptOut
	mustLct(t, &again, "accept", "--reason", "again", scopeID)
	if !slices.Equal(again.Skipped, []string{scopeID}) {
		t.Errorf("second accept = %+v, want it skipped", again)
	}
}

// A glossary entry id is for an instruction to the skill: lct accept refuses
// it, dry run or not, says what it is for, and writes nothing.
func TestAcceptRefusesAGlossaryEntryID(t *testing.T) {
	s := acceptRepo(t)
	writeTestFile(t, s.root, "rules/declined.yaml", "# keep\n")
	entry := itemid.GlossaryEntry(scopeKR, scopeZH)
	for _, args := range [][]string{
		{"accept", "--reason", "r", entry},
		{"accept", "--reason", "r", s.suggestion, entry},
		{"accept", "--dry-run", entry},
	} {
		code, stdout, stderr := lct(t, args...)
		if code != 1 || !strings.Contains(stderr, entry) || !strings.Contains(stderr, "glossary entry") || !strings.Contains(stderr, "skill") {
			t.Errorf("%v: exit code %d, want 1 saying the id is a glossary entry for the skill\n%s%s", args, code, stdout, stderr)
		}
	}
	if got := readOptional(t, s.root, "rules/declined.yaml") + readOptional(t, s.root, "rules/accepted.yaml"); !strings.HasPrefix(got, "# keep\n") || strings.Contains(got, s.suggestion) {
		t.Errorf("a list changed:\n%s", got)
	}
}
