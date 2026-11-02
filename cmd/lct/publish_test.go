package main

import (
	"encoding/json"
	"errors"
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
	if want := []string{"assets", "changed", "notes_path", "release_needed", "tag", "tw_commit"}; !slices.Equal(keys, want) {
		t.Errorf("output keys = %v, want %v", keys, want)
	}

	out := filepath.Join(root, "out")
	if !res.Changed || !res.ReleaseNeeded || res.Tag != "zh-TW-20261029.1" || res.TWCommit != runGitIn(t, tw, "rev-parse", "HEAD") ||
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
	ctx, _ := testCtx(t, root, "lang_dir: zh-TW\nworktrees: { raw: .work/raw, tw: .work/tw }\n", true)
	if err := runPublish(ctx, nil); !isUserError(err) || !strings.Contains(err.Error(), "run lct build") {
		t.Errorf("publish without a build: err = %v (%T), want a user error that says to build", err, err)
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

// A build made before the rules were committed is "+dirty": publish refuses it
// as a user error that says so, and tw gets no commit.
func TestPublishRefusesADirtyBuild(t *testing.T) {
	root := buildRepo(t)
	t.Setenv(sourceDateEpoch, "")
	if _, err := runBuildJSON(t, root); err != nil { // the rules were copied in after the last commit
		t.Fatal(err)
	}
	tw := filepath.Join(root, ".work", "tw")
	before := runGitIn(t, tw, "status", "--porcelain")

	ctx, _ := testCtx(t, root, "", true)
	if err := runPublish(ctx, nil); !isUserError(err) || !strings.Contains(err.Error(), "+dirty") {
		t.Errorf("publish of a dirty build: err = %v (%T), want a user error naming +dirty", err, err)
	}
	if after := runGitIn(t, tw, "status", "--porcelain"); after != before {
		t.Errorf("a refusal changed tw: %q -> %q", before, after)
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

// lang_dir names the zips, as it does for lct build.
func TestPublishUsesTheConfiguredLanguageDirectory(t *testing.T) {
	root := publishRepo(t)
	saved, err := os.ReadFile(filepath.Join(root, "lct.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	original := string(saved)

	// Not set: a user error that says so, before tw is touched.
	ctx, _ := testCtx(t, root, "worktrees: { raw: .work/raw, tw: .work/tw }\n", true)
	if err := runPublish(ctx, nil); !isUserError(err) || !strings.Contains(err.Error(), "lang_dir") {
		t.Errorf("publish without lang_dir: err = %v (%T), want a user error naming lang_dir", err, err)
	}

	// Another directory: its zips are the ones looked for.
	ctx, _ = testCtx(t, root, "lang_dir: zh-XX\nworktrees: { raw: .work/raw, tw: .work/tw }\n", true)
	if err := runPublish(ctx, nil); !isUserError(err) || !strings.Contains(err.Error(), "zh-XX_text_raw-20261029.zip") {
		t.Errorf("publish with lang_dir zh-XX: err = %v (%T), want a user error naming zh-XX_text_raw-20261029.zip", err, err)
	}
	// Built with that directory, the zips are named after it, in build.json
	// too, and they are published.
	ctx, _ = testCtx(t, root, strings.Replace(original, "lang_dir: zh-TW", "lang_dir: zh-XX", 1), true)
	if err := runBuild(ctx, nil); err != nil {
		t.Fatal(err)
	}
	res, _, err := runPublishJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{filepath.Join(root, "out", "zh-XX_text_raw-20261029.zip"), filepath.Join(root, "out", "zh-XX_complete_raw-20261029.zip")}; !slices.Equal(res.Assets, want) {
		t.Errorf("Assets = %v, want %v", res.Assets, want)
	}
}

// runCheckJSON runs lct publish --check --json and returns its output keys and
// its value.
func runCheckJSON(t *testing.T, root string) (publish.Check, map[string]json.RawMessage) {
	t.Helper()
	ctx, out := testCtx(t, root, "", true)
	if err := runPublish(ctx, []string{"--check"}); err != nil {
		t.Fatal(err)
	}
	var check publish.Check
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &check); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if err := json.Unmarshal(out.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	return check, fields
}

// --check reads the commits and the tags of tw and nothing else: it commits
// nothing, writes no notes, and needs neither a build nor the zips.
func TestPublishCheckCommitsNothing(t *testing.T) {
	root := publishRepo(t)
	tw := filepath.Join(root, ".work", "tw")
	state := func() string {
		return runGitIn(t, tw, "status", "--porcelain", "-uall") + "\n" + runGitIn(t, tw, "log", "--format=%H", "--all")
	}
	wantKeys := []string{"release_needed", "same_as", "tag", "tw_commit"}
	keysOf := func(fields map[string]json.RawMessage) []string {
		var keys []string
		for k := range fields {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		return keys
	}

	// The build is not committed yet: tw has no commit, and so no Release is
	// due for it. The files of the build stay where they are.
	before := state()
	check, fields := runCheckJSON(t, root)
	if keys := keysOf(fields); !slices.Equal(keys, wantKeys) {
		t.Errorf("output keys = %v, want %v", keys, wantKeys)
	}
	if check != (publish.Check{}) {
		t.Errorf("check of a tw with no commit = %+v, want nothing", check)
	}
	if after := state(); after != before {
		t.Errorf("--check changed tw:\n%s\n->\n%s", before, after)
	}
	if _, err := os.Stat(filepath.Join(root, "out", "release-notes.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("--check wrote the release notes: %v", err)
	}

	// Committed by publish, not tagged: a Release is due.
	res, _, err := runPublishJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	before = state()
	check, fields = runCheckJSON(t, root)
	if keys := keysOf(fields); !slices.Equal(keys, wantKeys) {
		t.Errorf("output keys = %v, want %v", keys, wantKeys)
	}
	if want := (publish.Check{TWCommit: res.TWCommit, ReleaseNeeded: true}); check != want {
		t.Errorf("check = %+v, want %+v", check, want)
	}

	// Released: the tag is on the commit, and the pack is the one of that tag.
	// Neither the zips nor build.json are needed to say so.
	runGitIn(t, tw, "tag", res.Tag)
	for _, name := range res.Assets {
		if err := os.Remove(name); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Remove(filepath.Join(tw, ".meta", "build.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(res.NotesPath); err != nil {
		t.Fatal(err)
	}
	before = state()
	check, fields = runCheckJSON(t, root)
	if keys := keysOf(fields); !slices.Equal(keys, wantKeys) {
		t.Errorf("output keys = %v, want %v", keys, wantKeys)
	}
	if want := (publish.Check{TWCommit: res.TWCommit, SameAs: res.Tag, Tag: res.Tag}); check != want {
		t.Errorf("check of a released commit = %+v, want %+v", check, want)
	}
	if after := state(); after != before {
		t.Errorf("--check changed tw:\n%s\n->\n%s", before, after)
	}
	if _, err := os.Stat(res.NotesPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("--check wrote the release notes: %v", err)
	}
	var sameAs string
	if err := json.Unmarshal(fields["same_as"], &sameAs); err != nil || sameAs != res.Tag {
		t.Errorf("same_as = %s, %v, want %q", fields["same_as"], err, res.Tag)
	}
}

// The text form says what the JSON says.
func TestPublishCheckText(t *testing.T) {
	root := publishRepo(t)
	res, _, err := runPublishJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	ctx, out := testCtx(t, root, "", false)
	if err := runPublish(ctx, []string{"--check"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{res.TWCommit, "needed"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
}

// --check needs lang_dir, as publish does: the file it leaves out of the
// comparison is in the language directory.
func TestPublishCheckNeedsTheLanguageDirectory(t *testing.T) {
	root := publishRepo(t)
	ctx, _ := testCtx(t, root, "worktrees: { raw: .work/raw, tw: .work/tw }\n", true)
	if err := runPublish(ctx, []string{"--check"}); !isUserError(err) || !strings.Contains(err.Error(), "lang_dir") {
		t.Errorf("--check without lang_dir: err = %v (%T), want a user error naming lang_dir", err, err)
	}
	ctx, _ = testCtx(t, root, "", true)
	if err := runPublish(ctx, []string{"--check", "extra"}); !isUserError(err) {
		t.Errorf("--check extra: err = %v, want a user error", err)
	}
}

// With a real build: a build from another rules commit that translates the
// same way differs from the Release in Info/version.json alone, so it is
// committed to tw and not released. This is also where the layout publish
// assumes of the pack is held to the one lct build writes.
func TestPublishDoesNotReleaseAPackThatOnlyHasANewVersion(t *testing.T) {
	root := publishRepo(t)
	tw := filepath.Join(root, ".work", "tw")
	first, _, err := runPublishJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	runGitIn(t, tw, "tag", first.Tag)

	runGitIn(t, root, "commit", "--allow-empty", "-q", "-m", "rules: nothing that changes a translation")
	if _, err := runBuildJSON(t, root); err != nil {
		t.Fatal(err)
	}
	res, _, err := runPublishJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Changed || res.ReleaseNeeded || res.SameAs != first.Tag || res.TWCommit == first.TWCommit {
		t.Fatalf("publish = %+v, want a new commit of tw, no Release, the pack of %s", res, first.Tag)
	}
	// The one file of the pack that differs is the version file.
	changed := runGitIn(t, tw, "diff", "--name-only", first.TWCommit, res.TWCommit, "--", "LimbusCompany_Data")
	if want := "LimbusCompany_Data/Lang/zh-TW/Info/version.json"; changed != want {
		t.Errorf("files of the pack that differ = %q, want %q", changed, want)
	}
	if check, _ := runCheckJSON(t, root); check.ReleaseNeeded || check.SameAs != first.Tag || check.TWCommit != res.TWCommit {
		t.Errorf("check = %+v, want no Release, the pack of %s at %s", check, first.Tag, res.TWCommit)
	}
}
