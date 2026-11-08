package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/history"
)

// historyRepo is a publishRepo with two Releases: the build, and a commit of
// tw that adds a file to the pack. It returns the repo and the two tags.
func historyRepo(t *testing.T) (root, first, second string) {
	t.Helper()
	root = publishRepo(t)
	tw := filepath.Join(root, ".work", "tw")
	// The Releases name the snapshot they were built from.
	raw := filepath.Join(root, ".work", "raw")
	runGitIn(t, raw, "init", "-q", "-b", "raw")
	for _, kv := range [][2]string{{"user.name", "Test"}, {"user.email", "test@example.com"}, {"commit.gpgsign", "false"}, {"tag.gpgsign", "false"}} {
		runGitIn(t, raw, "config", kv[0], kv[1])
	}
	runGitIn(t, raw, "add", "-A")
	runGitIn(t, raw, "commit", "-q", "-m", "snapshot")
	runGitIn(t, raw, "tag", "raw-20261029")
	res, _, err := runPublishJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	first = res.Tag
	runGitIn(t, tw, "tag", first)
	second = "zh-TW-20261030.1"
	writeTestFile(t, tw, "LimbusCompany_Data/Lang/zh-TW/Extra.json", "{}\n")
	// A rewrite rule of main turns 流血 into 出血; the second Release is built
	// from the commit that holds it and records it as applied.
	writeTestFile(t, root, "rules/rewrite/laceration.yaml", "reason: 統一用詞\nfiles: ['**']\npattern: '流血'\nreplace: '出血'\n")
	runGitIn(t, root, "add", "rules")
	runGitIn(t, root, "commit", "-q", "-m", "rules: laceration")
	build, err := os.ReadFile(filepath.Join(tw, ".meta", "build.json"))
	if err != nil {
		t.Fatal(err)
	}
	var info struct {
		RulesCommit string `json:"rules_commit"`
	}
	if err := json.Unmarshal(build, &info); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, tw, ".meta/build.json", strings.Replace(string(build), info.RulesCommit, runGitIn(t, root, "rev-parse", "HEAD"), 1))
	writeTestFile(t, tw, ".meta/applied.json", "{\"requeue\": [], \"rewrite\": [\"laceration\"]}\n")
	// One translation changes, in the pack and in its record.
	for _, f := range []string{"LimbusCompany_Data/Lang/zh-TW/Bufs.json", ".meta/units/Bufs.json.jsonl"} {
		data, err := os.ReadFile(filepath.Join(tw, f))
		if err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, tw, f, strings.Replace(string(data), "流血", "出血", 1))
	}
	runGitIn(t, tw, "add", "-A")
	runGitIn(t, tw, "commit", "-q", "-m", "an added file")
	runGitIn(t, tw, "tag", second)
	return root, first, second
}

// snapshotTree is what a read-only command must leave as it was: the status,
// the refs and the bytes of every file of the worktree (but .git).
func snapshotTree(t *testing.T, dir string) string {
	t.Helper()
	var b strings.Builder
	b.WriteString(runGitIn(t, dir, "status", "--porcelain", "--ignored") + "\n")
	b.WriteString(runGitIn(t, dir, "for-each-ref") + "\n")
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		b.WriteString(p + "\x00" + string(data) + "\x00")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func TestHistoryMakeIsReadOnly(t *testing.T) {
	root, first, second := historyRepo(t)
	tw := filepath.Join(root, ".work", "tw")
	before := snapshotTree(t, tw)
	mainRefs := runGitIn(t, root, "for-each-ref")

	ctx, out := testCtx(t, root, "", true)
	if err := runHistoryMake(ctx, []string{second}); err != nil {
		t.Fatal(err)
	}
	if after := snapshotTree(t, tw); after != before {
		t.Errorf("history make changed the tw worktree")
	}
	if runGitIn(t, root, "for-each-ref") != mainRefs {
		t.Errorf("history make changed the refs of the repo")
	}
	var rec history.Record
	if err := json.Unmarshal(out.Bytes(), &rec); err != nil {
		t.Fatalf("output is not a record: %v\n%s", err, out)
	}
	if rec.Tag != second || rec.PrevTag != first || rec.CausesVersion != history.CausesVersion {
		t.Errorf("record = %+v, want %s after %s with causes_version %d", rec, second, first, history.CausesVersion)
	}
	if !strings.Contains(out.String(), "other_files") {
		t.Errorf("output lacks the record fields:\n%s", out)
	}
}

func TestHistoryMakeJSONCarriesCauses(t *testing.T) {
	root, _, second := historyRepo(t)
	ctx, out := testCtx(t, root, "", true)
	if err := runHistoryMake(ctx, []string{second}); err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	if string(fields["causes_version"]) != "1" {
		t.Errorf("causes_version = %s, want 1", fields["causes_version"])
	}
	var rec history.Record
	if err := json.Unmarshal(out.Bytes(), &rec); err != nil {
		t.Fatal(err)
	}
	if len(rec.Entries) != 1 {
		t.Fatalf("entries = %+v, want the changed translation", rec.Entries)
	}
	want := []history.Cause{{Kind: history.CauseKindRewrite, ID: "laceration"}}
	if got := rec.Entries[0].Causes; !reflect.DeepEqual(got, want) {
		t.Errorf("causes = %+v, want %+v", got, want)
	}
	if info, ok := rec.Rules["rewrite:laceration"]; !ok || info.Path != "rules/rewrite/laceration.yaml" || rec.Unknown != 0 {
		t.Errorf("rules = %+v, unknown = %d, want the rule laceration known", rec.Rules, rec.Unknown)
	}
	if !strings.Contains(out.String(), `"causes"`) {
		t.Errorf("output lacks causes:\n%s", out)
	}
}

func TestHistoryMakeText(t *testing.T) {
	root, first, second := historyRepo(t)
	ctx, out := testCtx(t, root, "", false)
	if err := runHistoryMake(ctx, []string{second}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{second, first, "1 entries"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("text lacks %q:\n%s", want, out)
		}
	}
}

func TestHistoryMakeOutWritesAFile(t *testing.T) {
	root, _, second := historyRepo(t)
	file := filepath.Join(root, "record.json")
	ctx, _ := testCtx(t, root, "", false)
	if err := runHistoryMake(ctx, []string{second, "--out", file}); err != nil {
		t.Fatal(err)
	}
	var rec history.Record
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &rec); err != nil || rec.Tag != second {
		t.Errorf("file = %s, %v, want the record of %s", data, err, second)
	}
}

func TestHistoryMakeUserErrors(t *testing.T) {
	root, first, _ := historyRepo(t)
	for name, args := range map[string][]string{
		"no tag":        nil,
		"unknown tag":   {"zh-TW-20200101.1"},
		"first Release": {first},
		"two tags":      {first, first},
	} {
		ctx, _ := testCtx(t, root, "", false)
		if err := runHistoryMake(ctx, args); !isUserError(err) {
			t.Errorf("%s: err = %v (%T), want a user error", name, err, err)
		}
	}
}

func TestSummarizeCauses(t *testing.T) {
	rec := &history.Record{Tag: "zh-TW-2", PrevTag: "zh-TW-1", Unknown: 1, Entries: []history.Entry{
		{Causes: []history.Cause{{Kind: history.CauseKindRewrite, ID: "a"}}},
		{Causes: []history.Cause{{Kind: history.CauseKindRewrite, ID: "b"}, {Kind: history.CauseKindNew}}},
		{Causes: []history.Cause{{Kind: history.CauseKindRewrite, ID: "b"}}},
		{Causes: []history.Cause{{Kind: history.CauseKindUnknown}}},
	}}
	s := summarizeRecord(rec, "")
	if s.Entries != 4 || s.Kinds["rewrite"] != 3 || s.Kinds["new"] != 1 || s.Unknown != 1 {
		t.Errorf("summary = %+v", s)
	}
	if len(s.Rules) != 2 || s.Rules[0].Key != "rewrite:b" || s.Rules[0].Units != 2 || s.Rules[1].Key != "rewrite:a" {
		t.Errorf("rules = %+v, want rewrite:b (2) before rewrite:a (1)", s.Rules)
	}
	text := s.String()
	for _, want := range []string{"zh-TW-2", "zh-TW-1", "rewrite:b", "unknown"} {
		if !strings.Contains(text, want) {
			t.Errorf("text lacks %q:\n%s", want, text)
		}
	}
}
