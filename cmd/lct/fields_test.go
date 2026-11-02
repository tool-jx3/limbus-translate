package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/fields"
)

func writeTestFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// learnRepo builds a repo root with a small raw worktree in which "key" and
// "codeName" are the same in all three languages three times; codeName is
// also a manual identifier, so only key is learned.
func learnRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTestFile(t, root, "lct.yaml", "worktrees: { raw: .work/raw, tw: .work/tw }\n")
	manual, err := os.ReadFile("../../rules/fields.manual.yaml")
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, "rules/fields.manual.yaml", string(manual))
	for _, l := range []struct{ dir, prefix, text string }{
		{"kr", "KR_", "한국어"}, {"en", "EN_", "English"}, {"jp", "JP_", "日本語"},
	} {
		writeTestFile(t, root, ".work/raw/"+l.dir+"/"+l.prefix+"Sample-1.json", `{"dataList": [
			{"id": 1, "key": "k1", "codeName": "c1", "text": "`+l.text+`1"},
			{"id": 2, "key": "k2", "codeName": "c2", "text": "`+l.text+`2"},
			{"id": 3, "key": "k3", "codeName": "c3", "text": "`+l.text+`3"}
		]}`)
	}
	return root
}

// runLearn runs "lct fields learn" with args in root and returns its output.
func runLearn(t *testing.T, root string, jsonOut bool, args ...string) (string, error) {
	t.Helper()
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err = runFieldsLearn(&cli.Context{Root: root, Config: cfg, JSON: jsonOut, Out: &out}, args)
	return out.String(), err
}

func TestFieldsLearnReportsWithoutWriting(t *testing.T) {
	root := learnRepo(t)
	out, err := runLearn(t, root, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "  - key\n") {
		t.Errorf("output should list the learned field key:\n%s", out)
	}
	if strings.Contains(out, "codeName") {
		t.Errorf("a manual identifier must not be repeated in the learned rules:\n%s", out)
	}
	if !strings.Contains(out, "does not exist") {
		t.Errorf("output should say there is no rules/fields.yaml yet:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(root, "rules/fields.yaml")); !os.IsNotExist(err) {
		t.Errorf("rules/fields.yaml must not be written without --write, stat err = %v", err)
	}
}

func TestFieldsLearnWriteThenDiff(t *testing.T) {
	root := learnRepo(t)
	target := filepath.Join(root, "rules/fields.yaml")

	out, err := runLearn(t, root, false, "--write")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "wrote rules/fields.yaml") {
		t.Errorf("output should report the write:\n%s", out)
	}
	written, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	want := (&fields.Learned{IdentifierFields: []string{"key"}}).YAML()
	if !bytes.Equal(written, want) {
		t.Errorf("rules/fields.yaml =\n%s\nwant\n%s", written, want)
	}
	if _, err := fields.Load(root); err != nil {
		t.Fatalf("the written file must load: %v", err)
	}

	out, err = runLearn(t, root, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "up to date") {
		t.Errorf("a second run should find nothing to change:\n%s", out)
	}

	// Pretend the file was learned from older data: it lacks key and has a
	// field that is gone.
	writeTestFile(t, root, "rules/fields.yaml", string((&fields.Learned{IdentifierFields: []string{"usage"}}).YAML()))
	out, err = runLearn(t, root, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"+ identifier_fields: key", "- identifier_fields: usage", "would change"} {
		if !strings.Contains(out, line) {
			t.Errorf("output should contain %q:\n%s", line, out)
		}
	}
	if got, _ := os.ReadFile(target); strings.Contains(string(got), "key") {
		t.Errorf("a run without --write must leave the file alone:\n%s", got)
	}

	if _, err := runLearn(t, root, false, "--write"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(target); !bytes.Equal(got, want) {
		t.Errorf("--write should bring the file up to date, got:\n%s", got)
	}
}

func TestFieldsLearnJSON(t *testing.T) {
	root := learnRepo(t)
	writeTestFile(t, root, "rules/fields.yaml", string((&fields.Learned{IdentifierFields: []string{"usage"}}).YAML()))
	out, err := runLearn(t, root, true)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Learned struct {
			IdentifierFields []string            `json:"identifier_fields"`
			GroupFields      map[string][]string `json:"group_fields"`
		} `json:"learned"`
		Existing bool `json:"existing"`
		Changes  []struct {
			Op, Group, Field string
		} `json:"changes"`
		Written bool `json:"written"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if len(got.Learned.IdentifierFields) != 1 || got.Learned.IdentifierFields[0] != "key" {
		t.Errorf("learned = %+v", got.Learned)
	}
	if !got.Existing || got.Written || len(got.Changes) != 2 {
		t.Errorf("existing=%v written=%v changes=%+v", got.Existing, got.Written, got.Changes)
	}
}

// A pack without files would make Learn return no rules at all, and --write
// would then wipe rules/fields.yaml.
func TestFieldsLearnRefusesEmptyENOrJPPack(t *testing.T) {
	for _, c := range []struct{ lang, file, want string }{
		{"en", ".work/raw/en/EN_Sample-1.json", "no EN files"},
		{"jp", ".work/raw/jp/JP_Sample-1.json", "no JP files"},
	} {
		t.Run(c.lang, func(t *testing.T) {
			root := learnRepo(t)
			keep := string((&fields.Learned{IdentifierFields: []string{"usage"}}).YAML())
			writeTestFile(t, root, "rules/fields.yaml", keep)
			if err := os.Remove(filepath.Join(root, c.file)); err != nil {
				t.Fatal(err)
			}
			for _, args := range [][]string{nil, {"--write"}} {
				_, err := runLearn(t, root, false, args...)
				if err == nil || !strings.Contains(err.Error(), c.want) {
					t.Fatalf("args %v: error = %v, want one mentioning %q", args, err, c.want)
				}
				if reflect.TypeOf(err) != reflect.TypeOf(cli.UserError("")) {
					t.Errorf("error %v (%T) should be a user error (exit 1)", err, err)
				}
			}
			if got, _ := os.ReadFile(filepath.Join(root, "rules/fields.yaml")); string(got) != keep {
				t.Errorf("rules/fields.yaml was modified:\n%s", got)
			}
		})
	}
}

// All three packs have files, but none in common (or none with aligned
// text), so nothing was counted. Reporting is fine; writing would wipe the
// rules.
func TestFieldsLearnRefusesWriteWhenNothingCounted(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "lct.yaml", "worktrees: { raw: .work/raw }\n")
	manual, err := os.ReadFile("../../rules/fields.manual.yaml")
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, "rules/fields.manual.yaml", string(manual))
	writeTestFile(t, root, ".work/raw/kr/KR_A.json", `{"dataList": [{"id": 1, "key": "x"}]}`)
	writeTestFile(t, root, ".work/raw/en/EN_B.json", `{"dataList": [{"id": 1, "key": "x"}]}`)
	writeTestFile(t, root, ".work/raw/jp/JP_C.json", `{"dataList": [{"id": 1, "key": "x"}]}`)
	keep := string((&fields.Learned{IdentifierFields: []string{"usage"}}).YAML())
	writeTestFile(t, root, "rules/fields.yaml", keep)

	_, err = runLearn(t, root, false, "--write")
	if err == nil || !strings.Contains(err.Error(), "counted no units") {
		t.Fatalf("error = %v, want one saying no units were counted", err)
	}
	if reflect.TypeOf(err) != reflect.TypeOf(cli.UserError("")) {
		t.Errorf("error %v (%T) should be a user error (exit 1)", err, err)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "rules/fields.yaml")); string(got) != keep {
		t.Errorf("rules/fields.yaml was modified:\n%s", got)
	}

	// Without --write it still reports, and says why --write would refuse.
	out, err := runLearn(t, root, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "counted no units") {
		t.Errorf("the report should warn that nothing was counted:\n%s", out)
	}
}

func TestFieldsLearnUserErrors(t *testing.T) {
	root := learnRepo(t)
	empty := t.TempDir()
	writeTestFile(t, empty, "lct.yaml", "worktrees: { raw: .work/raw }\n")
	writeTestFile(t, empty, ".work/raw/kr/readme.txt", "no KR files")
	writeTestFile(t, empty, ".work/raw/en/readme.txt", "")
	writeTestFile(t, empty, ".work/raw/jp/readme.txt", "")
	noRaw := t.TempDir()
	writeTestFile(t, noRaw, "lct.yaml", "worktrees: { raw: .work/raw }\n")

	cases := []struct {
		name string
		root string
		args []string
		want string
	}{
		{"unknown flag", root, []string{"--nope"}, "nope"},
		{"positional argument", root, []string{"extra"}, "extra"},
		{"no raw worktree", noRaw, nil, "no raw worktree"},
		{"raw worktree without KR files", empty, nil, "no KR files"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runLearn(t, c.root, false, c.args...)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error = %v, want one mentioning %q", err, c.want)
			}
			// All of these are mistakes of the user, not system failures.
			if reflect.TypeOf(err) != reflect.TypeOf(cli.UserError("")) {
				t.Errorf("error %v (%T) should be a user error (exit 1)", err, err)
			}
		})
	}
}

// Units were counted, but no field is the same in all three languages, so no
// layer passes its threshold. The empty result must not replace the rules.
func TestLearnWriteKeepsRulesWhenNothingPasses(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "lct.yaml", "worktrees: { raw: .work/raw }\n")
	manual, err := os.ReadFile("../../rules/fields.manual.yaml")
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, "rules/fields.manual.yaml", string(manual))
	for _, l := range []struct{ dir, prefix, text string }{
		{"kr", "KR_", "한국어"}, {"en", "EN_", "English"}, {"jp", "JP_", "日本語"},
	} {
		writeTestFile(t, root, ".work/raw/"+l.dir+"/"+l.prefix+"Sample-1.json", `{"dataList": [
			{"id": 1, "text": "`+l.text+`1"}, {"id": 2, "text": "`+l.text+`2"}, {"id": 3, "text": "`+l.text+`3"}
		]}`)
	}
	keep := string((&fields.Learned{IdentifierFields: []string{"usage"}}).YAML())
	writeTestFile(t, root, "rules/fields.yaml", keep)

	_, err = runLearn(t, root, false, "--write")
	if err == nil || !strings.Contains(err.Error(), "no field") {
		t.Fatalf("error = %v, want one saying no field passed", err)
	}
	if reflect.TypeOf(err) != reflect.TypeOf(cli.UserError("")) {
		t.Errorf("error %v (%T) should be a user error (exit 1)", err, err)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "rules/fields.yaml")); string(got) != keep {
		t.Errorf("rules/fields.yaml was modified:\n%s", got)
	}

	// Without --write it reports, and says why --write would refuse.
	out, err := runLearn(t, root, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "no field") {
		t.Errorf("the report should warn that nothing passed:\n%s", out)
	}

	// The JSON says so too, for a reader that does not see the warning.
	out, err = runLearn(t, root, true)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil || got["nothing_passed"] != true {
		t.Errorf("--json = %s (%v), want nothing_passed true", out, err)
	}
	out, err = runLearn(t, learnRepo(t), true)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil || got["nothing_passed"] != false {
		t.Errorf("--json of a pack that teaches = %s (%v), want nothing_passed false", out, err)
	}
}

// Everything learned is already in the manual rules: that is a real result
// (the generated file is then empty), not a lack of evidence.
func TestLearnWriteEmptiesRulesTheManualFileCovers(t *testing.T) {
	root := learnRepo(t)
	manual, err := os.ReadFile(filepath.Join(root, "rules/fields.manual.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	covered := strings.Replace(string(manual), "[id, model, codeName]", "[id, model, codeName, key]", 1)
	if covered == string(manual) {
		t.Fatal("the manual rules no longer have the identifier list this test extends")
	}
	writeTestFile(t, root, "rules/fields.manual.yaml", covered)
	writeTestFile(t, root, "rules/fields.yaml", string((&fields.Learned{IdentifierFields: []string{"key"}}).YAML()))

	if _, err := runLearn(t, root, false, "--write"); err != nil {
		t.Fatal(err)
	}
	want := (&fields.Learned{}).YAML()
	if got, _ := os.ReadFile(filepath.Join(root, "rules/fields.yaml")); !bytes.Equal(got, want) {
		t.Errorf("rules/fields.yaml =\n%s\nwant\n%s", got, want)
	}
}

// A typo in a rule file is the author's mistake, not a failure of the system:
// exit 1 with the message, whichever of the two files holds it.
func TestFieldsTypoIsAUserError(t *testing.T) {
	for _, c := range []struct{ name, file, content, want string }{
		{"unknown key in the manual file", "rules/fields.manual.yaml", "identifer_fields: [id]\n", "identifer_fields"},
		{"unknown key in the generated file", "rules/fields.yaml", "identifer_fields: [id]\n", "identifer_fields"},
		{"empty name in the manual file", "rules/fields.manual.yaml", "identifier_fields: [\"\"]\n", "empty name"},
		{"bad pattern in the manual file", "rules/fields.manual.yaml", "name_only: { file_glob: \"*\", field: desc, pattern: \"(\" }\n", "name_only"},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := learnRepo(t)
			writeTestFile(t, root, c.file, c.content)
			t.Chdir(root)
			code, _, stderr := lct(t, "fields", "learn")
			if code != 1 {
				t.Errorf("exit code = %d, want 1\nstderr:\n%s", code, stderr)
			}
			if !strings.Contains(stderr, c.want) {
				t.Errorf("stderr should say %q:\n%s", c.want, stderr)
			}
		})
	}

	// A rule file that cannot be read at all is still a system error.
	root := learnRepo(t)
	if err := os.Remove(filepath.Join(root, "rules/fields.manual.yaml")); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	if code, _, stderr := lct(t, "fields", "learn"); code != 2 {
		t.Errorf("exit code with a missing manual file = %d, want 2\nstderr:\n%s", code, stderr)
	}
}
