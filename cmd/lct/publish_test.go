package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/publish"
)

// publishRepo is a repo whose tw worktree holds a real build (buildRepo, then
// lct build), with the rules committed so that the build is not "+dirty".
func publishRepo(t *testing.T) string {
	t.Helper()
	root := buildRepo(t)
	t.Setenv(sourceDateEpoch, "")
	runGitIn(t, root, "add", "-A")
	runGitIn(t, root, "commit", "-q", "-m", "rules")
	if _, err := runBuildJSON(t, root); err != nil {
		t.Fatal(err)
	}
	return root
}

func runPublishJSON(t *testing.T, root string, args ...string) (publish.Result, map[string]json.RawMessage, error) {
	t.Helper()
	ctx, out := testCtx(t, root, "", true)
	err := runPublish(ctx, args)
	var res publish.Result
	var fields map[string]json.RawMessage
	if err == nil {
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("output is not JSON: %v\n%s", err, out)
		}
		if err := json.Unmarshal(out.Bytes(), &fields); err != nil {
			t.Fatal(err)
		}
	}
	return res, fields, err
}

func TestPublishCommandCommitsTheBuild(t *testing.T) {
	root := publishRepo(t)
	tw := filepath.Join(root, ".work", "tw")

	res, fields, err := runPublishJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	// rebuild.yml reads these keys with jq.
	var keys []string
	for k := range fields {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	if want := []string{"assets", "changed", "notes_path", "tag", "tw_commit"}; !slices.Equal(keys, want) {
		t.Errorf("output keys = %v, want %v", keys, want)
	}

	out := filepath.Join(root, "out")
	if !res.Changed || res.Tag != "zh-TW-20261029.1" || res.TWCommit != runGitIn(t, tw, "rev-parse", "HEAD") ||
		res.NotesPath != filepath.Join(out, "release-notes.md") ||
		!slices.Equal(res.Assets, []string{
			filepath.Join(out, "zh-TW_text_raw-20261029.zip"),
			filepath.Join(out, "zh-TW_complete_raw-20261029.zip"),
		}) {
		t.Errorf("output = %+v", res)
	}
	for _, name := range res.Assets {
		if _, err := os.Stat(name); err != nil {
			t.Error(err)
		}
	}
	if got, want := runGitIn(t, tw, "log", "-1", "--format=%s"), "build: raw-20261029 (rules "+runGitIn(t, root, "rev-parse", "HEAD")+")"; got != want {
		t.Errorf("commit message = %q, want %q", got, want)
	}
	if got := runGitIn(t, tw, "status", "--porcelain"); got != "" {
		t.Errorf("tw is not clean after publish:\n%s", got)
	}

	// The notes carry the inputs of the build read from the real build.json.
	notes, err := os.ReadFile(res.NotesPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# zh-TW-20261029.1", "raw-20261029", testToken, runGitIn(t, root, "rev-parse", "HEAD"), "CC BY-NC-SA 4.0", "Project Moon"} {
		if !strings.Contains(string(notes), want) {
			t.Errorf("notes lack %q:\n%s", want, notes)
		}
	}
	if strings.Contains(string(notes), "+dirty") {
		t.Errorf("the rules were committed, but the notes say dirty:\n%s", notes)
	}

	// Building and publishing again changes nothing: the build is
	// reproducible, so rebuild.yml will find tw as it was pushed.
	if _, err := runBuildJSON(t, root); err != nil {
		t.Fatal(err)
	}
	again, _, err := runPublishJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	if again.Changed || again.TWCommit != res.TWCommit || again.Tag != res.Tag {
		t.Errorf("publish after a rebuild = %+v, want no change at %s tagged %s", again, res.TWCommit, res.Tag)
	}
}

func TestPublishCommandText(t *testing.T) {
	root := publishRepo(t)
	ctx, out := testCtx(t, root, "", false)
	if err := runPublish(ctx, nil); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"zh-TW-20261029.1", "release-notes.md", "zh-TW_text_raw-20261029.zip", "zh-TW_complete_raw-20261029.zip"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
}

func TestPublishNeedsBuild(t *testing.T) {
	// A repo that was never built: tw has nothing to publish.
	root := snapshotRepo(t)
	ctx, _ := testCtx(t, root, "", true)
	if err := runPublish(ctx, nil); !isUserError(err) {
		t.Errorf("publish without a build: err = %v (%T), want a user error", err, err)
	}

	// A built repo that lost a zip.
	root = publishRepo(t)
	for _, kind := range []string{"text", "complete"} {
		zip := filepath.Join(root, "out", "zh-TW_"+kind+"_raw-20261029.zip")
		saved, err := os.ReadFile(zip)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(zip); err != nil {
			t.Fatal(err)
		}
		before := runGitIn(t, filepath.Join(root, ".work", "tw"), "status", "--porcelain")
		ctx, _ := testCtx(t, root, "", true)
		if err := runPublish(ctx, nil); !isUserError(err) || !strings.Contains(err.Error(), "zh-TW_"+kind+"_raw-20261029.zip") {
			t.Errorf("publish without the %s zip: err = %v (%T), want a user error naming it", kind, err, err)
		}
		if after := runGitIn(t, filepath.Join(root, ".work", "tw"), "status", "--porcelain"); after != before {
			t.Errorf("a refusal changed tw: %q -> %q", before, after)
		}
		if err := os.WriteFile(zip, saved, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPublishRejectsArguments(t *testing.T) {
	root := publishRepo(t)
	ctx, _ := testCtx(t, root, "", true)
	for _, args := range [][]string{{"extra"}, {"--no-such-flag"}} {
		if err := runPublish(ctx, args); !isUserError(err) {
			t.Errorf("publish %v: err = %v, want a user error", args, err)
		}
	}
}
