package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/plan"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
)

const (
	suggestRawTag = "raw-20261105"

	suggestMovedFrom = "StoryData/E002X.json|/dataList/14/content"
	suggestMovedTo   = "StoryData/E002X.json|/dataList/16/content"
	suggestExpired   = "StoryData/E002X.json|/dataList/20/content"
	suggestNoteKey   = "StoryData/S001A.json|/dataList/3/content"

	suggestInconsistentID = "inconsistent/ally/name/fd93b875" // sha256("도약 모듈")[:8]
	suggestNoteID         = "note/" + suggestNoteKey + "/3713708f"
)

// suggestOut is the part of "suggest --json" the tests look at.
type suggestOut struct {
	Suggestions []struct {
		ID       string `json:"id"`
		Kind     string `json:"kind"`
		Title    string `json:"title"`
		Evidence []struct {
			Key   string `json:"key"`
			KR    string `json:"kr"`
			ZH    string `json:"zh"`
			Note  string `json:"note"`
			Count int    `json:"count"`
		} `json:"evidence"`
	} `json:"suggestions"`
	Excluded int `json:"excluded"`
	Total    int `json:"total"`
}

func (o suggestOut) ids() []string {
	out := []string{}
	for _, s := range o.Suggestions {
		out = append(out, s.ID)
	}
	return out
}

// suggestRepo is a repo root in the working directory with the rules of the
// repo, a glossary of one candidate term, the raw snapshot suggestRawTag, and
// a tw worktree whose build report and plan have, between them, one of
// everything that can be suggested and three things the rules answer already:
// the model of a character, a file the routing has a family for, and an
// adjustment whose file has the new key. It returns the root and the model.
func suggestRepo(t *testing.T) (root, knownModel string) {
	t.Helper()
	root = cleanRules(t)
	copyFile(t, filepath.Join(repoRoot, "lct.yaml"), filepath.Join(root, "lct.yaml"))
	writeTestFile(t, root, "glossary/terms.yaml", "- {kr: 도약 모듈, zh: 跳躍成長模組, status: approved}\n- {kr: 신규 용어, zh: 新術語, status: candidate}\n")
	writeTestFile(t, root, ".work/raw/meta/source.json", `{"tag": "`+suggestRawTag+`", "token": "t", "created_at": "2026-11-05 02:55:23", "fetched_at": "2026-11-05T00:00:00Z", "files": {}}`)
	writeTestFile(t, root, "rules/adjustments/2026-11-01-a.yaml", `- key: `+suggestMovedTo+`
  raw: raw-20261029
  kr_sha: `+strings.Repeat("a", 64)+`
  zh: 移動的調整
  reason: r
  rule: one-off
- key: `+suggestExpired+`
  raw: raw-20261029
  kr_sha: `+strings.Repeat("b", 64)+`
  zh: 過期的調整
  reason: r
  rule: one-off
`)

	r, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	knownModel = r.Characters[0].Models[0]

	st := store.Open(filepath.Join(root, ".work", "tw"))
	if err := st.WriteMetaJSON("build.json", build.Info{
		RawTag: suggestRawTag,
		Zips:   map[string]string{},
		Report: build.Report{
			Inconsistent: []build.Inconsistency{{
				Family: "ally", Field: "name", KR: "도약 모듈",
				ZH: []string{"跳躍成長模組", "跳躍模組"}, Keys: []string{"a.json|/dataList/0/name", "b.json|/dataList/1/name"}, Units: 4,
			}},
			Notes: []store.Note{{Key: suggestNoteKey, Batch: "b", KRSha: "x", Model: "m", Note: "KR 這句有雙關"}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.WriteMetaJSON("plan.json", plan.Summary{
		RawTag:                suggestRawTag,
		NewSpeakers:           []string{knownModel, "새 화자"},
		FieldChanges:          []fields.Change{{Op: "add", Group: "BattleSpeechBubbleDlg", Field: "desc"}},
		NewFilesDefaultFamily: []string{"StoryData/S1037B.json", "Zzz-twth.json"},
		Adjustments: plan.AdjustmentSummary{
			Moved:   []plan.MovedKey{{From: suggestMovedFrom, To: suggestMovedTo}},
			Expired: []string{suggestExpired},
		},
	}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	return root, knownModel
}

func suggestJSON(t *testing.T, args ...string) suggestOut {
	t.Helper()
	var out suggestOut
	mustLct(t, &out, append([]string{"suggest"}, args...)...)
	return out
}

func TestSuggestCommand(t *testing.T) {
	root, _ := suggestRepo(t)

	// Everything that is left is suggested, by kind (the groups of different
	// translations last), and having suggestions is not a finding: the exit
	// code is 0.
	out := suggestJSON(t)
	want := []string{
		suggestNoteID,
		"speaker/새 화자",
		"family/Zzz-twth.json",
		"field/add/BattleSpeechBubbleDlg/desc",
		"adjustment-expired/" + suggestExpired,
		"term/신규 용어",
		suggestInconsistentID,
	}
	if !slices.Equal(out.ids(), want) || out.Excluded != 0 {
		t.Fatalf("suggestions = %v, excluded %d; want %v and none excluded", out.ids(), out.Excluded, want)
	}
	for _, s := range out.Suggestions {
		if s.Kind == "" || s.Title == "" || s.Evidence == nil {
			t.Errorf("suggestion %+v lacks a kind, a title or evidence", s)
		}
	}

	// The declined list and an exclude file are skipped. Two of the ids are
	// suggestions of this run; the third was answered by the rules before.
	writeTestFile(t, root, "rules/declined.yaml", "- {id: '"+suggestInconsistentID+"', reason: 兩種都可以, date: 2026-11-06}\n- {id: 'speaker/不在這裡', reason: r, date: 2026-11-06}\n")
	writeTestFile(t, root, "work/proposal/exclude.txt", "# 已在開著的規則 PR 中\n\n  speaker/새 화자  \r\nterm/不在這裡\n")
	out = suggestJSON(t, "--exclude-file", "work/proposal/exclude.txt")
	want = slices.DeleteFunc(slices.Clone(want), func(id string) bool { return id == suggestInconsistentID || id == "speaker/새 화자" })
	if !slices.Equal(out.ids(), want) || out.Excluded != 2 {
		t.Errorf("suggestions = %v, excluded %d; want %v and 2 excluded", out.ids(), out.Excluded, want)
	}
}

func TestSuggestAsText(t *testing.T) {
	suggestRepo(t)
	code, stdout, stderr := lct(t, "suggest")
	if code != 0 || stderr != "" {
		t.Fatalf("lct suggest: exit code %d, stderr %q", code, stderr)
	}
	for _, want := range []string{suggestInconsistentID, "speaker/새 화자", "term/신규 용어", "7 suggestion"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the text lacks %q:\n%s", want, stdout)
		}
	}
	if json.Valid([]byte(stdout)) {
		t.Errorf("without --json the output should be text for a person:\n%s", stdout)
	}

	// Nothing to suggest: an empty list, not null, and still exit 0.
	writeTestFile(t, ".", "glossary/terms.yaml", "[]\n")
	st := store.Open(filepath.Join(".work", "tw"))
	if err := st.WriteMetaJSON("build.json", build.Info{RawTag: suggestRawTag, Zips: map[string]string{}}); err != nil {
		t.Fatal(err)
	}
	if err := st.WriteMetaJSON("plan.json", plan.Summary{RawTag: suggestRawTag}); err != nil {
		t.Fatal(err)
	}
	code, stdout, _ = lct(t, "--json", "suggest")
	if code != 0 || !strings.Contains(stdout, `"suggestions": []`) || !strings.Contains(stdout, `"excluded": 0`) {
		t.Errorf("nothing to suggest: exit code %d\n%s", code, stdout)
	}
	if code, stdout, _ := lct(t, "suggest"); code != 0 || !strings.Contains(stdout, "0 suggestion") {
		t.Errorf("nothing to suggest, as text: exit code %d\n%s", code, stdout)
	}
}

func TestSuggestExcludeFiles(t *testing.T) {
	root, _ := suggestRepo(t)
	writeTestFile(t, root, "one.txt", "speaker/새 화자\n")
	writeTestFile(t, root, "two.txt", "# comment\n\n"+suggestNoteID+"\r\n   \n#term/신규 용어\n")

	// The flag is given more than once; a line that starts with # is a
	// comment, so that id is still suggested.
	out := suggestJSON(t, "--exclude-file", "one.txt", "--exclude-file", "two.txt")
	if out.Excluded != 2 || slices.Contains(out.ids(), "speaker/새 화자") || slices.Contains(out.ids(), suggestNoteID) || !slices.Contains(out.ids(), "term/신규 용어") {
		t.Errorf("suggestions = %v, excluded %d; want the speaker and the note dropped and the term left", out.ids(), out.Excluded)
	}

	// An id that is in the declined list and in the file counts once.
	writeTestFile(t, root, "rules/declined.yaml", "- {id: speaker/새 화자, reason: r, date: 2026-11-06}\n")
	if out := suggestJSON(t, "--exclude-file", "one.txt"); out.Excluded != 1 {
		t.Errorf("excluded = %d, want 1 for an id that is both declined and in a file", out.Excluded)
	}
}

func TestSuggestUserErrors(t *testing.T) {
	root, _ := suggestRepo(t)
	writeTestFile(t, root, "empty.txt", "")
	if err := os.Mkdir(filepath.Join(root, "adir"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, tc := range map[string]struct {
		args   []string
		stderr string // what the message says
	}{
		"an argument":              {[]string{"suggest", "extra"}, "extra"},
		"an unknown flag":          {[]string{"suggest", "--nope"}, "nope"},
		"a flag without its value": {[]string{"suggest", "--exclude-file"}, "exclude-file"},
		"a missing exclude file":   {[]string{"suggest", "--exclude-file", "empty.txt", "--exclude-file", "missing.txt"}, "missing.txt"},
		"a directory":              {[]string{"suggest", "--exclude-file", "adir"}, "adir"},
	} {
		code, stdout, stderr := lct(t, tc.args...)
		if code != 1 || stdout != "" || !strings.Contains(stderr, tc.stderr) {
			t.Errorf("%s: exit code %d, stdout %q, stderr %q; want exit code 1 and a message with %q", name, code, stdout, stderr, tc.stderr)
		}
	}

	// No snapshot to compare the report and the plan with.
	if err := os.Remove(filepath.Join(root, ".work/raw/meta/source.json")); err != nil {
		t.Fatal(err)
	}
	if code, _, stderr := lct(t, "suggest"); code != 1 || !strings.Contains(stderr, "snapshot") {
		t.Errorf("without a snapshot: exit code %d, stderr %q; want 1 and a message about the snapshot", code, stderr)
	}
}

// A declined list that cannot be read is not skipped in silence: a suggestion
// that was declined would come back. That is a system error.
func TestSuggestBrokenDeclinedList(t *testing.T) {
	root, _ := suggestRepo(t)
	writeTestFile(t, root, "rules/declined.yaml", "- id: [x\n")
	code, _, stderr := lct(t, "suggest")
	if code != 2 || !strings.Contains(stderr, "rules/declined.yaml") {
		t.Errorf("exit code %d, stderr %q; want 2 and the name of the file", code, stderr)
	}
}

// The report and the plan of another snapshot say nothing about this one.
func TestSuggestOnlyTheCurrentSnapshot(t *testing.T) {
	root, _ := suggestRepo(t)
	st := store.Open(filepath.Join(root, ".work", "tw"))
	var info build.Info
	var sum plan.Summary
	for name, v := range map[string]any{"build.json": &info, "plan.json": &sum} {
		if found, err := st.ReadMetaJSON(name, v); err != nil || !found {
			t.Fatalf("reading %s: %v, %v", name, found, err)
		}
	}
	info.RawTag, sum.RawTag = "raw-20261029", "raw-20261029"
	if err := st.WriteMetaJSON("build.json", info); err != nil {
		t.Fatal(err)
	}
	if err := st.WriteMetaJSON("plan.json", sum); err != nil {
		t.Fatal(err)
	}
	if out := suggestJSON(t); !slices.Equal(out.ids(), []string{"term/신규 용어"}) {
		t.Errorf("suggestions = %v, want only the term", out.ids())
	}

	// No build.json at all is not an error either.
	if err := os.Remove(filepath.Join(root, ".work/tw/.meta/build.json")); err != nil {
		t.Fatal(err)
	}
	if out := suggestJSON(t); !slices.Equal(out.ids(), []string{"term/신규 용어"}) {
		t.Errorf("without a build: suggestions = %v, want only the term", out.ids())
	}
}

// "rules lint" checks rules/declined.yaml with the other rule files.
func TestRulesLintDeclined(t *testing.T) {
	root := cleanRules(t)
	copyFile(t, filepath.Join(repoRoot, "lct.yaml"), filepath.Join(root, "lct.yaml"))
	t.Chdir(root)
	writeTestFile(t, root, "rules/declined.yaml", "- {id: 'speaker/아무개', reason: 路人, date: 2026-11-05}\n")
	if code, stdout, stderr := lct(t, "rules", "lint"); code != 0 {
		t.Fatalf("lint of a good declined list: exit code %d\n%s%s", code, stdout, stderr)
	}

	writeTestFile(t, root, "rules/declined.yaml", "- {id: 'bogus/x', reason: '', date: 2026-13-45}\n")
	code, stdout, _ := lct(t, "--json", "rules", "lint")
	var report struct {
		OK     bool
		Errors []struct{ File, Message string }
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, stdout)
	}
	if code != 1 || report.OK || len(report.Errors) != 3 {
		t.Fatalf("exit code %d, report %+v; want 1 and three findings (kind, reason, date)", code, report)
	}
	for _, e := range report.Errors {
		if e.File != "rules/declined.yaml" {
			t.Errorf("finding %+v is not about rules/declined.yaml", e)
		}
	}
}

// --kind keeps the suggestions of the kinds named, --offset and --limit take a
// page of the sorted list after the declined and excluded ids are left out, and
// total is how many there are before the page is taken. The rules-review skill
// asks for as many as it handles in one run.
func TestSuggestKindsAndPages(t *testing.T) {
	root, _ := suggestRepo(t)
	all := suggestJSON(t).ids()
	if out := suggestJSON(t); out.Total != len(all) {
		t.Fatalf("total = %d, want %d without paging", out.Total, len(all))
	}

	// A page is taken from the order of the kinds: a note comes first.
	out := suggestJSON(t, "--limit", "3")
	if !slices.Equal(out.ids(), all[:3]) || out.Total != len(all) || out.ids()[0] != suggestNoteID {
		t.Errorf("--limit 3: %v, total %d; want %v and %d", out.ids(), out.Total, all[:3], len(all))
	}
	out = suggestJSON(t, "--offset", "2", "--limit", "2")
	if !slices.Equal(out.ids(), all[2:4]) || out.Total != len(all) {
		t.Errorf("--offset 2 --limit 2: %v, total %d; want %v", out.ids(), out.Total, all[2:4])
	}
	if out = suggestJSON(t, "--offset", "100"); len(out.ids()) != 0 || out.Suggestions == nil || out.Total != len(all) {
		t.Errorf("a page past the end: %v, total %d; want an empty list", out.ids(), out.Total)
	}

	out = suggestJSON(t, "--kind", "speaker", "--kind", "term")
	if want := []string{"speaker/새 화자", "term/신규 용어"}; !slices.Equal(out.ids(), want) || out.Total != 2 {
		t.Errorf("--kind speaker --kind term: %v, total %d; want %v", out.ids(), out.Total, want)
	}

	// The page is taken after what is left out; excluded counts the kinds
	// asked for.
	writeTestFile(t, root, "skip.txt", "speaker/새 화자\n"+suggestInconsistentID+"\n")
	out = suggestJSON(t, "--exclude-file", "skip.txt", "--kind", "speaker", "--kind", "term", "--limit", "1")
	if !slices.Equal(out.ids(), []string{"term/신규 용어"}) || out.Total != 1 || out.Excluded != 1 {
		t.Errorf("excluded, then a page: %v, total %d, excluded %d; want the term, 1, 1", out.ids(), out.Total, out.Excluded)
	}

	// The note is quoted in the title, and whole in its evidence.
	out = suggestJSON(t, "--kind", "note")
	if len(out.Suggestions) != 1 || len(out.Suggestions[0].Evidence) != 1 || out.Suggestions[0].Evidence[0].Note != "KR 這句有雙關" {
		t.Errorf("--kind note: %+v, want the note with its text as evidence", out.Suggestions)
	}

	code, stdout, stderr := lct(t, "suggest", "--limit", "2")
	if code != 0 || stderr != "" || !strings.Contains(stdout, "2 of 7 suggestion") {
		t.Errorf("a page as text: exit code %d\n%s%s", code, stdout, stderr)
	}

	for name, tc := range map[string]struct {
		args []string
		want string
	}{
		"an unknown kind":      {[]string{"--kind", "speakers"}, "speakers"},
		"a limit of zero":      {[]string{"--limit", "0"}, "--limit"},
		"a negative offset":    {[]string{"--offset", "-1"}, "--offset"},
		"a limit not a number": {[]string{"--limit", "x"}, "limit"},
	} {
		code, stdout, stderr := lct(t, append([]string{"suggest"}, tc.args...)...)
		if code != 1 || stdout != "" || !strings.Contains(stderr, tc.want) {
			t.Errorf("%s: exit code %d, stdout %q, stderr %q; want 1 and a message with %q", name, code, stdout, stderr, tc.want)
		}
	}
}

// The provisional terms in force (batch.ActiveProvisional: not in the
// glossary, not declined) are listed as term-new/<kr>, with the conflicting
// renderings of the KR as evidence; --skip-kind leaves a kind out.
func TestSuggestListsProvisionalTerms(t *testing.T) {
	root, _ := suggestRepo(t)
	st := store.Open(filepath.Join(root, ".work", "tw"))
	if _, err := st.AppendTerms([]store.ProvisionalTerm{
		{KR: "우레", ZH: "雷鳴", Key: "a.json|/dataList/0/name", Batch: "b1", RawTag: suggestRawTag, Time: "2026-11-05T00:00:00Z"},
		{KR: "우레", ZH: "驚雷", Key: "b.json|/dataList/0/name", Batch: "b2", RawTag: suggestRawTag, Time: "2026-11-05T00:00:00Z", Conflict: true},
		{KR: "신규 용어", ZH: "新術語二", Key: "c.json|/dataList/0/name", Batch: "b3", RawTag: suggestRawTag, Time: "2026-11-05T00:00:00Z"},
		{KR: "번개", ZH: "閃電", Key: "d.json|/dataList/0/name", Batch: "b4", RawTag: suggestRawTag, Time: "2026-11-05T00:00:00Z"},
	}); err != nil {
		t.Fatal(err)
	}
	// 신규 용어 is in the glossary (as a candidate), so it is not listed.
	out := suggestJSON(t, "--kind", "term-new")
	if want := []string{"term-new/번개", "term-new/우레"}; !slices.Equal(out.ids(), want) {
		t.Fatalf("term-new = %v, want %v", out.ids(), want)
	}
	if n := len(out.Suggestions[1].Evidence); n != 2 {
		t.Errorf("우레 has %d evidence rows, want the unit and the conflict", n)
	}
	all := suggestJSON(t)
	if i, j := slices.Index(all.ids(), "term-new/우레"), slices.Index(all.ids(), "term/신규 용어"); i < 0 || j < 0 || i > j {
		t.Errorf("ids = %v, want term-new before term", all.ids())
	}

	// A declined id is not listed again.
	writeTestFile(t, root, "rules/declined.yaml", "- {id: 'term-new/우레', reason: r, date: 2026-11-06}\n")
	if got := suggestJSON(t, "--kind", "term-new"); !slices.Equal(got.ids(), []string{"term-new/번개"}) {
		t.Errorf("after declining: %v", got.ids())
	}

	// --skip-kind is what CI uses to leave term-new to a person.
	skipped := suggestJSON(t, "--skip-kind", "term-new")
	if slices.ContainsFunc(skipped.ids(), func(id string) bool { return strings.HasPrefix(id, "term-new/") }) || len(skipped.Suggestions) == 0 {
		t.Errorf("--skip-kind term-new: %v", skipped.ids())
	}
	if code, _, stderr := lct(t, "suggest", "--skip-kind", "bogus"); code != 1 || !strings.Contains(stderr, "bogus") {
		t.Errorf("--skip-kind bogus: exit code %d, stderr %q; want 1 and a message with bogus", code, stderr)
	}
}
