package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/adjust"
	"github.com/tool-jx3/limbus-translate/internal/plan"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

const (
	adjustKR0  = "[Laceration] 위력 5 부여" // batchKey0
	adjustRule = "guides/characters/ryoshu.md"
)

// adjustRepo is batchRepo with the repo as the working directory, for the
// tests that run the real entry point.
func adjustRepo(t *testing.T) string {
	t.Helper()
	root, _ := batchRepo(t)
	oldNow := ruleFileNow
	ruleFileNow = func() time.Time { return cliNow }
	t.Cleanup(func() { ruleFileNow = oldNow })
	t.Chdir(root)
	return root
}

// adjustArgs is a valid "adjust add" call for batchKey0; the tests change
// single flags with adjustWith and adjustWithout.
func adjustArgs() []string {
	return []string{"adjust", "add",
		"--key", batchKey0,
		"--zh", "賦予 5 層 [Laceration] 強度",
		"--reason", "稱呼改用您",
		"--rule", adjustRule,
		"--topic", "honglu",
		"--date", "2026-11-02",
	}
}

// adjustWith returns args with the value of flag replaced, or the flag added.
func adjustWith(args []string, flag, value string) []string {
	out := slices.Clone(args)
	if i := slices.Index(out, flag); i >= 0 {
		out[i+1] = value
		return out
	}
	return append(out, flag, value)
}

func adjustWithout(args []string, flag string) []string {
	out := slices.Clone(args)
	i := slices.Index(out, flag)
	return slices.Delete(out, i, i+2)
}

type adjustOut struct {
	Path     string `json:"path"`
	Key      string `json:"key"`
	ZH       string `json:"zh"`
	Warnings []struct{ Code, Severity string }
}

func TestAdjustAddRejectsHardIssues(t *testing.T) {
	root := adjustRepo(t)

	// The [Laceration] of KR is missing.
	code, stdout, stderr := lct(t, adjustWith(adjustArgs(), "--zh", "賦予 5 層強度")...)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	for _, want := range []string{"id-mismatch", "hard", "Laceration"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout lacks %q:\n%s", want, stdout)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "rules", "adjustments")); err == nil {
		t.Error("a rejected adjustment created rules/adjustments")
	}

	// As JSON the issues are structured, the same way "lct validate" has them.
	code, stdout, _ = lct(t, append([]string{"--json"}, adjustWith(adjustArgs(), "--zh", "賦予 5 層強度")...)...)
	var rejected struct {
		OK     bool `json:"ok"`
		Issues []struct{ Code, Severity string }
	}
	if err := json.Unmarshal([]byte(stdout), &rejected); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, stdout)
	}
	if code != 1 || rejected.OK || len(rejected.Issues) == 0 || rejected.Issues[0].Code != "id-mismatch" || rejected.Issues[0].Severity != "hard" {
		t.Errorf("exit code = %d, report = %+v, want 1 and a hard id-mismatch", code, rejected)
	}
}

func TestAdjustAddEnforcesTheGlossaryOfTheUnit(t *testing.T) {
	root := adjustRepo(t)
	writeTestFile(t, filepath.Join(root, ".work", "raw"), "kr/KR_Pack.json", `{"dataList":[{"id":1,"content":"패키지 구매"}]}`)
	args := adjustWith(adjustWith(adjustArgs(), "--key", "Pack.json|/dataList/0/content"), "--zh", "購買禮包")

	// 禮包 is forbidden for 패키지: a content issue, hard for a new text.
	code, stdout, stderr := lct(t, args...)
	if code != 1 || !strings.Contains(stdout, "forbidden-term") {
		t.Fatalf("exit code = %d, want 1 with a forbidden-term\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	// A term the KR text of the unit does not contain is not enforced.
	if code, stdout, stderr := lct(t, adjustWith(args, "--key", batchKey0)...); code != 1 || !strings.Contains(stdout, "id-mismatch") || strings.Contains(stdout, "forbidden-term") {
		t.Errorf("exit code = %d, want 1 for the [Laceration] only\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	// The adopted rendering passes.
	if code, stdout, stderr := lct(t, adjustWith(args, "--zh", "購買組合包")...); code != 0 {
		t.Errorf("exit code = %d, want 0\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
}

// The glossary terms of the unit's file are enforced: 신부 is 牧師 in the files
// of chapter 10 and 神父 in the others.
func TestAdjustAddEnforcesTheTermsOfTheFileOfTheUnit(t *testing.T) {
	root := adjustRepo(t)
	rawWT := filepath.Join(root, ".work", "raw")
	const kr = `{"dataList":[{"id":1,"content":"신부가 왔다"}]}`
	writeTestFile(t, rawWT, "kr/StoryData/KR_S1034B.json", kr)
	writeTestFile(t, rawWT, "kr/StoryData/KR_S725B.json", kr)
	args := adjustWith(adjustArgs(), "--zh", "神父來了")

	code, stdout, stderr := lct(t, adjustWith(args, "--key", "StoryData/S1034B.json|/dataList/0/content")...)
	if code != 1 || !strings.Contains(stdout, "forbidden-term") {
		t.Errorf("神父 in S1034B: exit code = %d, want 1 with a forbidden-term\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if code, stdout, stderr := lct(t, adjustWith(args, "--key", "StoryData/S725B.json|/dataList/0/content")...); code != 0 {
		t.Errorf("神父 in S725B: exit code = %d, want 0\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
}

func TestAdjustAddWritesTheRawTagAndShaOfTheCurrentKR(t *testing.T) {
	root := adjustRepo(t)
	rawWT := filepath.Join(root, ".work", "raw")

	// The text is normalized before it is checked and written: the spaces
	// around the [ID] are put in.
	var out adjustOut
	mustLct(t, &out, adjustWith(adjustArgs(), "--zh", "賦予 5 層[Laceration]強度")...)
	const wantPath = "rules/adjustments/2026-11-02-honglu.yaml"
	if out.Path != wantPath || out.Key != batchKey0 || out.ZH != "賦予 5 層 [Laceration] 強度" || len(out.Warnings) != 0 {
		t.Errorf("output = %+v, want path %s, key %s and the normalized zh", out, wantPath, batchKey0)
	}
	got, err := adjust.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	want := adjust.Adjustment{
		Key:    batchKey0,
		Raw:    "raw-20261029",
		KRSha:  units.KRSha(adjustKR0),
		ZH:     "賦予 5 層 [Laceration] 強度",
		Reason: "稱呼改用您",
		Rule:   adjustRule,
		File:   "2026-11-02-honglu.yaml",
	}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("Load = %+v, want %+v", got, want)
	}

	// A later snapshot: the tag and the KR text are the new ones.
	writeTestFile(t, rawWT, "kr/KR_Sample.json", `{"dataList":[
 {"id":1,"content":"[Laceration] 위력 5 부여"},
 {"id":2,"content":"뫼르소가 말했다!"}
]}`)
	writeTestFile(t, rawWT, "meta/source.json", `{"tag":"raw-20261105","token":"t","created_at":"2026-11-05 02:55:23","fetched_at":"2026-11-05T03:00:00Z","files":{}}`+"\n")
	mustLct(t, &out, adjustWith(adjustWith(adjustArgs(), "--key", batchKey1), "--zh", "莫梭說了！")...)
	got, err = adjust.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1].Key != batchKey1 || got[1].Raw != "raw-20261105" || got[1].KRSha != units.KRSha("뫼르소가 말했다!") {
		t.Errorf("second adjustment = %+v, want the new tag and the sha of the new KR text", got)
	}
	if got[0] != want {
		t.Errorf("the first adjustment changed: %+v", got[0])
	}
}

func TestAdjustAddOutput(t *testing.T) {
	adjustRepo(t)

	// Text output: where it went and what was written.
	code, stdout, stderr := lct(t, adjustArgs()...)
	if code != 0 || stderr != "" {
		t.Fatalf("exit code = %d, stderr = %q, want 0", code, stderr)
	}
	for _, want := range []string{"rules/adjustments/2026-11-02-honglu.yaml", batchKey0, "賦予 5 層 [Laceration] 強度"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output lacks %q:\n%s", want, stdout)
		}
	}

	// A warning does not stop the adjustment; it is reported next to the result.
	var out adjustOut
	mustLct(t, &out, adjustWith(adjustWith(adjustArgs(), "--key", batchKey1), "--zh", "莫梭說,好")...)
	if len(out.Warnings) != 1 || out.Warnings[0].Code != "halfwidth-punct" || out.Warnings[0].Severity != "warn" {
		t.Errorf("warnings = %v, want the halfwidth-punct warning", out.Warnings)
	}
}

func TestAdjustAddJSONIsPathKeyZH(t *testing.T) {
	adjustRepo(t)
	var raw map[string]json.RawMessage
	mustLct(t, &raw, adjustArgs()...)
	var names []string
	for name := range raw {
		names = append(names, name)
	}
	slices.Sort(names)
	if want := []string{"key", "path", "zh"}; !slices.Equal(names, want) {
		t.Errorf("JSON fields = %v, want %v (no warnings, so no warnings field)", names, want)
	}
}

func TestAdjustAddDateDefaultsToToday(t *testing.T) {
	adjustRepo(t)
	var out adjustOut
	mustLct(t, &out, adjustWithout(adjustArgs(), "--date")...)
	if want := "rules/adjustments/" + cliNow.Format("2006-01-02") + "-honglu.yaml"; out.Path != want {
		t.Errorf("path = %q, want %q", out.Path, want)
	}
}

func TestAdjustAddCleansTheRulePath(t *testing.T) {
	root := adjustRepo(t)
	mustLct(t, nil, adjustWith(adjustArgs(), "--rule", "./guides/characters/../characters/ryoshu.md")...)
	got, err := adjust.Load(root)
	if err != nil || len(got) != 1 || got[0].Rule != adjustRule {
		t.Errorf("Load = %+v, %v, want the rule %s", got, err, adjustRule)
	}
}

func TestAdjustAddRefusesAKeyAnyFileHolds(t *testing.T) {
	root := adjustRepo(t)
	mustLct(t, nil, adjustArgs()...)

	for _, topic := range []string{"honglu", "other-topic"} {
		code, stdout, stderr := lct(t, adjustWith(adjustArgs(), "--topic", topic)...)
		if code != 1 || stdout != "" || !strings.Contains(stderr, "2026-11-02-honglu.yaml") {
			t.Errorf("topic %s: exit code = %d, stdout = %q, stderr = %q, want 1 and the name of the file that has the key", topic, code, stdout, stderr)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "rules", "adjustments", "2026-11-02-other-topic.yaml")); err == nil {
		t.Error("a refused adjustment was written")
	}
}

func TestAdjustAddUserErrors(t *testing.T) {
	root := adjustRepo(t)

	for name, tc := range map[string]struct {
		args   []string
		stderr string
	}{
		"no key":               {adjustWithout(adjustArgs(), "--key"), "--key"},
		"no zh":                {adjustWithout(adjustArgs(), "--zh"), "--zh"},
		"empty zh":             {adjustWith(adjustArgs(), "--zh", " "), "--zh"},
		"no reason":            {adjustWithout(adjustArgs(), "--reason"), "--reason"},
		"empty reason":         {adjustWith(adjustArgs(), "--reason", " "), "--reason"},
		"no rule":              {adjustWithout(adjustArgs(), "--rule"), "--rule"},
		"no topic":             {adjustWithout(adjustArgs(), "--topic"), "--topic"},
		"unknown flag":         {append(adjustArgs(), "--bogus"), "bogus"},
		"unexpected argument":  {append(adjustArgs(), "extra"), "extra"},
		"malformed key":        {adjustWith(adjustArgs(), "--key", "nope"), "nope"},
		"file not in KR":       {adjustWith(adjustArgs(), "--key", "Gone.json|/dataList/0/content"), "Gone.json"},
		"pointer not in KR":    {adjustWith(adjustArgs(), "--key", "Sample.json|/dataList/9/content"), "/dataList/9/content"},
		"pointer is a number":  {adjustWith(adjustArgs(), "--key", "Sample.json|/dataList/0/id"), "/dataList/0/id"},
		"rule file is missing": {adjustWith(adjustArgs(), "--rule", "guides/nothing.md"), "guides/nothing.md"},
		"rule above the repo":  {adjustWith(adjustArgs(), "--rule", "../x.md"), "../x.md"},
		"bad date":             {adjustWith(adjustArgs(), "--date", "2026-13-01"), "date"},
		"bad topic":            {adjustWith(adjustArgs(), "--topic", "a/b"), "topic"},
	} {
		t.Run(name, func(t *testing.T) {
			code, stdout, stderr := lct(t, tc.args...)
			if code != 1 {
				t.Errorf("exit code = %d, want 1\nstdout: %s\nstderr: %s", code, stdout, stderr)
			}
			if stdout != "" || !strings.Contains(stderr, tc.stderr) {
				t.Errorf("stdout = %q, stderr = %q, want nothing on stdout and a message with %q", stdout, stderr, tc.stderr)
			}
			if _, err := os.Stat(filepath.Join(root, "rules", "adjustments")); err == nil {
				t.Error("a refused call created rules/adjustments")
			}
		})
	}

	// Without a snapshot there is no KR to take the tag and the sha from.
	t.Run("no snapshot", func(t *testing.T) {
		t.Chdir(snapshotRepo(t))
		code, _, stderr := lct(t, adjustWith(adjustArgs(), "--rule", "one-off")...)
		if code != 1 || !strings.Contains(stderr, "snapshot") {
			t.Errorf("exit code = %d, stderr = %q, want 1 and a hint to run lct snapshot", code, stderr)
		}
	})
}

func TestAdjustAddSystemErrors(t *testing.T) {
	root := adjustRepo(t)
	// An adjustment file that cannot be read cannot be checked for the key.
	writeTestFile(t, root, "rules/adjustments/2026-11-01-bad.yaml", "key: not a list\n")
	code, _, stderr := lct(t, adjustArgs()...)
	if code != 2 || !strings.Contains(stderr, "2026-11-01-bad.yaml") {
		t.Errorf("exit code = %d, stderr = %q, want 2 naming the broken file", code, stderr)
	}
}

func TestAdjustAddAddsTheSpaceAfterKeywordNames(t *testing.T) {
	root, _ := keywordRepo(t)
	oldNow := ruleFileNow
	ruleFileNow = func() time.Time { return cliNow }
	t.Cleanup(func() { ruleFileNow = oldNow })
	t.Chdir(root)

	var out adjustOut
	mustLct(t, &out, adjustWith(adjustWith(adjustArgs(), "--key", statusKey), "--zh", "使目標增加2級破裂強度")...)
	const want = "使目標增加2級破裂 強度"
	if out.ZH != want {
		t.Errorf("output zh = %q, want %q", out.ZH, want)
	}
	got, err := adjust.Load(root)
	if err != nil || len(got) != 1 || got[0].ZH != want {
		t.Errorf("Load = %+v, %v, want the adjustment with %q", got, err, want)
	}
}

// tidyOut is what "adjust tidy" prints.
type tidyOut struct {
	Changed []string `json:"changed"`
	Moved   int      `json:"moved"`
	Removed int      `json:"removed"`
}

const tidyAdjustments = "rules/adjustments/2026-11-01-wording.yaml"

// tidyEntry is an adjustment entry for key, made for the first snapshot.
func tidyEntry(key, kr, zh string) string {
	return "- key: " + key + "\n  raw: raw-20261029\n  kr_sha: " + units.KRSha(kr) + "\n  zh: " + zh + "\n  reason: 測試\n  rule: one-off\n"
}

// tidyRepo is adjustRepo with the state of a plan of the current snapshot:
// the plan moved one adjustment, and four expired. Of these, two are about
// units that were translated again since (one of them found by the key of the
// adjustment, one by ExpiredUnits), and two are not: one unit has only the
// human record of another adjustment, one is not a unit of KR.
func tidyRepo(t *testing.T) (root string, files string) {
	t.Helper()
	root = adjustRepo(t)
	const gone = "Sample.json|/dataList/%d/content"
	key := func(n int) string { return strings.Replace(gone, "%d", string(rune('0'+n)), 1) }

	files = tidyEntry(key(7), "뫼르소가 말했다", "移動的一筆") +
		tidyEntry(key(8), "舊的原文八", "八已重翻") +
		tidyEntry(key(9), "舊的原文九", "九還沒有重翻") +
		tidyEntry(batchKey0, "오래된 원문", "同一個位置重翻") +
		tidyEntry(key(5), "舊的原文五", "五只有人工紀錄")
	writeTestFile(t, root, tidyAdjustments, files)

	st := store.Open(filepath.Join(root, ".work", "tw"))
	sha0, sha1 := units.KRSha(adjustKR0), units.KRSha("뫼르소가 말했다")
	if err := st.SaveRecords(map[string]store.Record{
		// translated again: claude's record of the current text
		batchKey0: {Key: batchKey0, KRSha: sha0, ZH: "賦予 5 層 [Laceration] 強度", Origin: store.OriginClaude, Since: "raw-20261029"},
		// the human record of an adjustment is no retranslation
		batchKey1: {Key: batchKey1, KRSha: sha1, ZH: "莫梭說了話", Origin: store.OriginHuman, Since: "raw-20261029"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.SavePending([]store.PendingItem{{Key: key(9), Reason: plan.ReasonAdjustmentExpired}}); err != nil {
		t.Fatal(err)
	}
	writePlan(t, st, plan.Summary{
		RawTag: "raw-20261029", PrevRawTag: "raw-20261022",
		Adjustments: plan.AdjustmentSummary{
			Moved:   []plan.MovedKey{{From: key(7), To: batchKey1}},
			Expired: []string{batchKey0, key(5), key(8), key(9)},
			// The unit of the key 8 moved to batchKey0; the unit of 5 is
			// batchKey1; the others are at their own keys.
			ExpiredUnits: map[string]string{key(8): batchKey0, key(5): batchKey1},
		},
	})
	return root, files
}

func writePlan(t *testing.T, st *store.Store, sum plan.Summary) {
	t.Helper()
	sum.Changes.FilesAdded, sum.Changes.FilesDeleted, sum.Changes.FilesChanged, sum.Changes.FilesShifted = []string{}, []string{}, []string{}, []string{}
	if sum.Adjustments.Moved == nil {
		sum.Adjustments.Moved = []plan.MovedKey{}
	}
	if sum.Adjustments.Expired == nil {
		sum.Adjustments.Expired = []string{}
	}
	sum.RequeueApplied, sum.RewriteApplied, sum.FrozenChanged, sum.NewSpeakers = []plan.Applied{}, []plan.Applied{}, []string{}, []string{}
	sum.NewFilesDefaultFamily = []string{}
	if err := st.WriteMetaJSON(plan.File, sum); err != nil {
		t.Fatal(err)
	}
}

func TestAdjustTidyMovesKeysAndRemovesRetranslatedAdjustments(t *testing.T) {
	root, files := tidyRepo(t)

	var out tidyOut
	mustLct(t, &out, "adjust", "tidy")

	if want := (tidyOut{Changed: []string{tidyAdjustments}, Moved: 1, Removed: 2}); !slices.Equal(out.Changed, want.Changed) || out.Moved != want.Moved || out.Removed != want.Removed {
		t.Errorf("output = %+v, want %+v", out, want)
	}
	// Only the key of the first entry changed, and the two entries of the
	// unit that has claude's translation of the current text are gone.
	want := strings.Replace(files, "key: Sample.json|/dataList/7/content", "key: "+batchKey1, 1)
	want = strings.Replace(want, tidyEntry("Sample.json|/dataList/8/content", "舊的原文八", "八已重翻"), "", 1)
	want = strings.Replace(want, tidyEntry(batchKey0, "오래된 원문", "同一個位置重翻"), "", 1)
	if got, err := os.ReadFile(filepath.Join(root, tidyAdjustments)); err != nil || string(got) != want {
		t.Errorf("file =\n%s\nwant\n%s\n(%v)", got, want, err)
	}

	// The second run has nothing left to do.
	mustLct(t, &out, "adjust", "tidy")
	if len(out.Changed) != 0 || out.Moved != 0 || out.Removed != 0 {
		t.Errorf("second run = %+v, want nothing", out)
	}
}

// A unit that is translated again but queued again (a requeue) is not settled.
func TestAdjustTidyKeepsAdjustmentsOfUnitsThatAreQueued(t *testing.T) {
	root, files := tidyRepo(t)
	st := store.Open(filepath.Join(root, ".work", "tw"))
	if err := st.SavePending([]store.PendingItem{{Key: batchKey0, Reason: "requeue"}}); err != nil {
		t.Fatal(err)
	}
	var out tidyOut
	mustLct(t, &out, "adjust", "tidy")
	if out.Removed != 0 {
		t.Errorf("removed = %d, want the adjustment kept while its unit is in the queue", out.Removed)
	}
	want := strings.Replace(files, "key: Sample.json|/dataList/7/content", "key: "+batchKey1, 1)
	if got, _ := os.ReadFile(filepath.Join(root, tidyAdjustments)); string(got) != want {
		t.Errorf("file =\n%s\nwant only the moved key changed", got)
	}
}

// A record of an older KR text is not a retranslation either.
func TestAdjustTidyKeepsAdjustmentsOfUnitsWithAStaleRecord(t *testing.T) {
	root, _ := tidyRepo(t)
	st := store.Open(filepath.Join(root, ".work", "tw"))
	if err := st.SaveRecords(map[string]store.Record{
		batchKey0: {Key: batchKey0, KRSha: units.KRSha("older text"), ZH: "舊譯文", Origin: store.OriginClaude, Since: "raw-20261022"},
	}); err != nil {
		t.Fatal(err)
	}
	var out tidyOut
	mustLct(t, &out, "adjust", "tidy")
	if out.Removed != 0 || out.Moved != 1 {
		t.Errorf("output = %+v, want one move and nothing removed", out)
	}
}

// The plan can be older than the files: an adjustment it called expired may
// have been written again since. Tidy asks the KR again and keeps one that
// applies, and does not move one to a place the KR does not show.
func TestAdjustTidyChecksTheAdjustmentsAgainstTheKR(t *testing.T) {
	root, _ := tidyRepo(t)
	// Settled units, but the adjustments apply to them (the first at its key,
	// the second would be moved to batchKey1, not to the key the plan says).
	files := tidyEntry(batchKey0, adjustKR0, "仍然有效") + tidyEntry("Sample.json|/dataList/7/content", "뫼르소가 말했다", "移到別處")
	writeTestFile(t, root, tidyAdjustments, files)
	st := store.Open(filepath.Join(root, ".work", "tw"))
	writePlan(t, st, plan.Summary{
		RawTag: "raw-20261029", PrevRawTag: "raw-20261022",
		Adjustments: plan.AdjustmentSummary{
			Moved:   []plan.MovedKey{{From: "Sample.json|/dataList/7/content", To: "Sample.json|/dataList/3/content"}},
			Expired: []string{batchKey0},
		},
	})

	var out tidyOut
	mustLct(t, &out, "adjust", "tidy")

	if len(out.Changed) != 0 || out.Moved != 0 || out.Removed != 0 {
		t.Errorf("output = %+v, want nothing done", out)
	}
	if got, _ := os.ReadFile(filepath.Join(root, tidyAdjustments)); string(got) != files {
		t.Errorf("the adjustment file changed:\n%s", got)
	}
}

func TestAdjustTidyNeedsAPlanOfTheCurrentSnapshot(t *testing.T) {
	root, files := tidyRepo(t)
	st := store.Open(filepath.Join(root, ".work", "tw"))

	for name, setup := range map[string]func(){
		"a plan of an older snapshot": func() {
			writePlan(t, st, plan.Summary{
				RawTag: "raw-20261022", PrevRawTag: "raw-20261022",
				Adjustments: plan.AdjustmentSummary{Moved: []plan.MovedKey{{From: "Sample.json|/dataList/7/content", To: batchKey1}}},
			})
		},
		"no plan": func() { os.Remove(filepath.Join(root, ".work", "tw", ".meta", plan.File)) },
	} {
		setup()
		// With --json the note why nothing was done goes to stderr, where a
		// person or the log of a run sees it.
		code, stdout, stderr := lct(t, "--json", "adjust", "tidy")
		var out tidyOut
		if err := json.Unmarshal([]byte(stdout), &out); code != 0 || err != nil {
			t.Fatalf("%s: exit code %d, output %q (%v)", name, code, stdout, err)
		}
		if len(out.Changed) != 0 || out.Moved != 0 || out.Removed != 0 {
			t.Errorf("%s: output = %+v, want nothing done", name, out)
		}
		if !strings.Contains(stderr, "run lct plan first") || strings.Contains(stdout, "run lct plan first") {
			t.Errorf("%s: stderr = %q, stdout = %q; want the note on stderr only", name, stderr, stdout)
		}
		if code, stdout, stderr := lct(t, "adjust", "tidy"); code != 0 || !strings.Contains(stdout, "run lct plan first") || stderr != "" {
			t.Errorf("%s as text: exit code %d, stdout %q, stderr %q; want the note in the text", name, code, stdout, stderr)
		}
		if got, _ := os.ReadFile(filepath.Join(root, tidyAdjustments)); string(got) != files {
			t.Errorf("%s: the adjustment file changed", name)
		}
	}
}

func TestAdjustTidyOutput(t *testing.T) {
	tidyRepo(t)

	code, stdout, stderr := lct(t, "adjust", "tidy")
	if code != 0 || stderr != "" {
		t.Fatalf("exit code = %d, stderr = %q, want 0", code, stderr)
	}
	for _, want := range []string{tidyAdjustments, "1 moved", "2 removed"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output lacks %q:\n%s", want, stdout)
		}
	}

	code, stdout, _ = lct(t, "adjust", "tidy")
	if code != 0 || !strings.Contains(stdout, "nothing") {
		t.Errorf("second run: exit code = %d, output %q, want 0 and a word that nothing was done", code, stdout)
	}
	var raw map[string]json.RawMessage
	mustLct(t, &raw, "adjust", "tidy")
	if got := string(raw["changed"]); got != "[]" || string(raw["moved"]) != "0" || string(raw["removed"]) != "0" {
		t.Errorf("JSON of nothing to do = %v, want changed [] and zero counts", raw)
	}
}

func TestAdjustTidyErrors(t *testing.T) {
	root, _ := tidyRepo(t)
	if code, _, stderr := lct(t, "adjust", "tidy", "extra"); code != 1 || !strings.Contains(stderr, "extra") {
		t.Errorf("an argument: exit code = %d, stderr = %q, want 1", code, stderr)
	}
	if code, _, _ := lct(t, "adjust", "tidy", "--bogus"); code != 1 {
		t.Errorf("an unknown flag: exit code = %d, want 1", code)
	}

	// A file that cannot be edited safely is the person's to edit: exit 1,
	// and nothing changes.
	flow := "[{key: Sample.json|/dataList/7/content, raw: raw-20261029, kr_sha: " + units.KRSha("뫼르소가 말했다") + ", zh: 甲, reason: r, rule: one-off}]\n"
	writeTestFile(t, root, tidyAdjustments, flow)
	code, _, stderr := lct(t, "adjust", "tidy")
	if code != 1 || !strings.Contains(stderr, "2026-11-01-wording.yaml") {
		t.Errorf("a flow list: exit code = %d, stderr = %q, want 1 naming the file", code, stderr)
	}
	if got, _ := os.ReadFile(filepath.Join(root, tidyAdjustments)); string(got) != flow {
		t.Errorf("the file changed:\n%s", got)
	}

	// An adjustment file that does not load is a system error, as for plan.
	writeTestFile(t, root, tidyAdjustments, "key: not a list\n")
	if code, _, stderr := lct(t, "adjust", "tidy"); code != 2 || !strings.Contains(stderr, "2026-11-01-wording.yaml") {
		t.Errorf("a broken file: exit code = %d, stderr = %q, want 2 naming the file", code, stderr)
	}
}

// The commands that read or change the state in the tw worktree refuse to run
// without it, and say how to make it: with no worktree they would find no
// translations and no pending units, and answer as if there were none.
func TestCommandsNeedTheTWWorktree(t *testing.T) {
	root := adjustRepo(t)
	tw := filepath.Join(root, ".work", "tw")
	for name, args := range map[string][]string{
		"search":                 {"search", "--key", batchKey0},
		"requeue --key":          {"requeue", "--key", batchKey0, "--reason", "x"},
		"requeue --file":         {"requeue", "--file", "Sample.json", "--reason", "x"},
		"requeue --needs-review": {"requeue", "--needs-review", "--reason", "x"},
		"requeue --leased":       {"requeue", "--leased", "--reason", "x"},
		"adjust add":             adjustArgs(),
		"adjust tidy":            {"adjust", "tidy"},
		"review":                 {"review"},
		"impact":                 {"impact", "--base", "HEAD"},
		"suggest":                {"suggest"},
		"glossary derive":        {"glossary", "derive"},
	} {
		for variant, prepare := range map[string]func(){
			"no worktree":            func() { os.RemoveAll(tw) },
			"no .meta":               func() { os.RemoveAll(tw); os.MkdirAll(tw, 0o755) },
			"a file as .meta":        func() { os.RemoveAll(tw); os.MkdirAll(tw, 0o755); os.WriteFile(filepath.Join(tw, ".meta"), nil, 0o644) },
			"a file as the worktree": func() { os.RemoveAll(tw); os.WriteFile(tw, nil, 0o644) },
		} {
			t.Run(name+", "+variant, func(t *testing.T) {
				prepare()
				// A command that ignores the missing worktree must not
				// make the next case fail for its file.
				for _, dir := range []string{"rules/adjustments", "rules/requeue"} {
					os.RemoveAll(filepath.Join(root, dir))
				}
				code, stdout, stderr := lct(t, args...)
				if code != 1 || stdout != "" {
					t.Errorf("exit code %d, stdout %q, want 1 and nothing\nstderr:\n%s", code, stdout, stderr)
				}
				if !strings.Contains(stderr, ".work/tw") || !strings.Contains(stderr, "git worktree add .work/tw tw") {
					t.Errorf("stderr %q should name .work/tw and say how to make it", stderr)
				}
				// Nothing was made on the way: not the worktree, not a state in it,
				// not a file under rules/.
				if _, err := os.Stat(filepath.Join(tw, ".meta", "units")); err == nil {
					t.Errorf("%s made state in the tw worktree", name)
				}
				for _, dir := range []string{"rules/adjustments", "rules/requeue"} {
					if _, err := os.Stat(filepath.Join(root, dir)); err == nil {
						t.Errorf("%s wrote %s", name, dir)
					}
				}
			})
		}
	}

	// A request file for CI to apply touches no state: it needs no worktree.
	os.RemoveAll(tw)
	code, stdout, stderr := lct(t, "requeue", "--to-file", "honglu", "--key", batchKey0, "--reason", "x", "--date", "2026-11-02")
	if code != 0 {
		t.Errorf("requeue --to-file without the tw worktree: exit code %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
}
