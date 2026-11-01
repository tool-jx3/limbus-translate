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
