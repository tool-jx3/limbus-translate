package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/history"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/site"
	"github.com/tool-jx3/limbus-translate/internal/store"
)

// siteRepo is the repo of the build tests with the pack built: raw and tw as
// they are when the viewer is made.
func siteRepo(t *testing.T) string {
	t.Helper()
	root := buildRepo(t)
	t.Setenv(sourceDateEpoch, "")
	if _, err := runBuildJSON(t, root); err != nil {
		t.Fatal(err)
	}
	return root
}

func runSiteJSON(t *testing.T, root string, args ...string) (map[string]json.RawMessage, error) {
	t.Helper()
	ctx, out := testCtx(t, root, "", true)
	err := runSite(ctx, args)
	var got map[string]json.RawMessage
	if err == nil {
		if err := json.Unmarshal(out.Bytes(), &got); err != nil {
			t.Fatalf("output is not JSON: %v\n%s", err, out)
		}
	}
	return got, err
}

// field decodes the field key of a JSON object into v.
func field(t *testing.T, obj map[string]json.RawMessage, key string, v any) {
	t.Helper()
	raw, ok := obj[key]
	if !ok {
		t.Fatalf("no field %q in %v", key, obj)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatalf("field %q: %v", key, err)
	}
}

// readSiteMeta reads work/site/meta.json of root.
func readSiteMeta(t *testing.T, root string) site.Meta {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, "work", "site", "meta.json"))
	if err != nil {
		t.Fatal(err)
	}
	var meta site.Meta
	if err := json.Unmarshal(b, &meta); err != nil {
		t.Fatal(err)
	}
	return meta
}

func TestSiteCommandJSON(t *testing.T) {
	root := siteRepo(t)
	// What an earlier run left in the directory does not stay.
	writeTestFile(t, root, "work/site/stale.json", "{}")

	got, err := runSiteJSON(t, root, "--out", "work/site")
	if err != nil {
		t.Fatal(err)
	}
	var files, units, searchUnits, unplaced int
	var counts site.ReviewCounts
	var mismatches []string
	field(t, got, "review", &counts)
	field(t, got, "review_mismatches", &mismatches)
	field(t, got, "notes_unplaced", &unplaced)
	if mismatches == nil || unplaced != 0 {
		t.Errorf("review_mismatches = %v, notes_unplaced = %d, want a list and 0", mismatches, unplaced)
	}
	if meta := readSiteMeta(t, root); meta.Review != counts {
		t.Errorf("meta.review = %+v, the result says %+v", meta.Review, counts)
	}
	var size int64
	var outDir string
	field(t, got, "files", &files)
	field(t, got, "units", &units)
	field(t, got, "search_units", &searchUnits)
	field(t, got, "bytes", &size)
	field(t, got, "out", &outDir)
	if outDir != filepath.Join(root, "work", "site") || files < 1 || units < files || searchUnits < 1 || searchUnits > units || size < 1 {
		t.Errorf("output = %v, want the directory and counts that make sense", got)
	}

	set, err := pack.OpenRaw(filepath.Join(root, ".work", "raw"))
	if err != nil {
		t.Fatal(err)
	}
	if files != len(set.KR.Files()) {
		t.Errorf("files = %d, the KR pack has %d", files, len(set.KR.Files()))
	}

	// What is on disk is what the output says.
	out := filepath.Join(root, "work", "site")
	if _, err := os.Stat(filepath.Join(out, "stale.json")); !os.IsNotExist(err) {
		t.Errorf("stale.json is still there (%v)", err)
	}
	var onDisk int64
	var names []string
	err = filepath.WalkDir(out, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		onDisk += info.Size()
		rel, _ := filepath.Rel(out, p)
		names = append(names, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if onDisk != size {
		t.Errorf("bytes = %d, the files hold %d", size, onDisk)
	}
	for _, want := range []string{"meta.json", "index.json", "search/keys.json", "search/kr.json", "search/jp.json", "search/en.json", "search/zh.json",
		"review/suggestions.json", "review/warnings.json", "review/notes.json", "review/todo.json"} {
		if !slices.Contains(names, want) {
			t.Errorf("no %s in %v", want, names)
		}
	}
	for _, f := range set.KR.Files() {
		if !slices.Contains(names, "files/"+f) {
			t.Errorf("no files/%s", f)
		}
	}

	// The meta names the inputs of the build.
	meta := readSiteMeta(t, root)
	info, err := build.ReadInfo(filepath.Join(root, ".work", "tw"))
	if err != nil {
		t.Fatal(err)
	}
	if meta.Format != 2 || meta.RawTag != "raw-20261029" || meta.RulesCommit != info.RulesCommit || meta.RulesCommit == "" ||
		meta.Files != files || meta.Units != units || !slices.Equal(meta.Langs, []string{"kr", "jp", "en", "zh"}) {
		t.Errorf("meta = %+v, want raw-20261029, rules commit %q, %d files, %d units", meta, info.RulesCommit, files, units)
	}
	if meta.Release != "" {
		t.Errorf("release = %q, want none: tw has no Release tag", meta.Release)
	}
}

// The release is the latest Release tag on the commit tw stands at; the commit
// is the one of tw.
func TestSiteCommandNamesTheTWCommitAndRelease(t *testing.T) {
	root := siteRepo(t)
	tw := filepath.Join(root, ".work", "tw")
	head, _, err := gitx.CommitAll(tw, "build: raw-20261029")
	if err != nil || head == "" {
		t.Fatalf("commit of tw = %q, %v", head, err)
	}
	// .10 is after .9, which a sort by name does not say; zh-TW-latest is no
	// Release, and a Release of another commit is not this one's.
	for _, tag := range []string{"zh-TW-20261029.9", "zh-TW-20261029.10", "zh-TW-latest", "raw-20261029"} {
		if err := gitx.Tag(tw, tag); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := runSiteJSON(t, root, "--out", "work/site"); err != nil {
		t.Fatal(err)
	}
	if meta := readSiteMeta(t, root); meta.TWCommit != head || meta.Release != "zh-TW-20261029.10" {
		t.Errorf("tw_commit %q release %q, want %q zh-TW-20261029.10", meta.TWCommit, meta.Release, head)
	}

	// A commit without a Release tag has no Release.
	runGitIn(t, tw, "commit", "-q", "--allow-empty", "-m", "elsewhere")
	if _, err := runSiteJSON(t, root, "--out", "work/site"); err != nil {
		t.Fatal(err)
	}
	if meta := readSiteMeta(t, root); meta.TWCommit != runGitIn(t, tw, "rev-parse", "HEAD") || meta.Release != "" {
		t.Errorf("tw_commit %q release %q, want the new HEAD and no release", meta.TWCommit, meta.Release)
	}
}

func TestLatestReleaseTag(t *testing.T) {
	for _, tt := range []struct {
		tags []string
		want string
	}{
		{nil, ""},
		{[]string{"zh-TW-20261029.1"}, "zh-TW-20261029.1"},
		{[]string{"zh-TW-20261029.10", "zh-TW-20261029.9"}, "zh-TW-20261029.10"},
		{[]string{"zh-TW-20261105.1", "zh-TW-20261029.12"}, "zh-TW-20261105.1"},
		{[]string{"zh-TW-latest", "zh-TW-20261029.x", "zh-TW-20261029", "zh-TW-20261029.2"}, "zh-TW-20261029.2"},
		{[]string{"zh-TW-latest"}, ""},
	} {
		if got := latestReleaseTag(tt.tags); got != tt.want {
			t.Errorf("latestReleaseTag(%v) = %q, want %q", tt.tags, got, tt.want)
		}
	}
}

func TestSiteCommandRefusesOutDir(t *testing.T) {
	root := siteRepo(t)
	outside := t.TempDir()
	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	// Files that must survive every refusal, one in each directory an --out
	// below points at.
	keep := []string{
		filepath.Join(root, "keep.txt"),
		filepath.Join(filepath.Dir(root), "keep.txt"),
		filepath.Join(outside, "x", "keep.txt"),
		filepath.Join(root, ".git", "x", "keep.txt"),
		filepath.Join(root, ".work", "tw", "site", "keep.txt"),
		filepath.Join(root, "work", "a", ".git", "keep.txt"),
	}
	for _, k := range keep {
		if err := os.MkdirAll(filepath.Dir(k), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(k, []byte("keep"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { os.Remove(filepath.Join(filepath.Dir(root), "keep.txt")) })

	for _, out := range []string{
		"lct.yaml", // a file
		".", "./", "..", "../x", "work/../..", "work/..",
		"/tmp/x", filepath.Join(outside, "x"), root,
		".git", ".git/x", "work/a/.git", "work/a/.git/b",
		"link", "link/x", // a link out of the repo
		".work", ".work/raw", ".work/tw", ".work/tw/site", // the worktrees the data is made from
	} {
		_, err := runSiteJSON(t, root, "--out", out)
		if !isUserError(err) {
			t.Errorf("--out %q: err = %v, want a user error", out, err)
		}
		for _, k := range keep {
			if _, err := os.Stat(k); err != nil {
				t.Errorf("--out %q removed %s", out, k)
			}
		}
		if _, err := os.Stat(filepath.Join(root, "lct.yaml")); err != nil {
			t.Errorf("--out %q removed lct.yaml", out)
		}
	}
}

func TestSiteCommandRefusesWhatItCannotRead(t *testing.T) {
	tests := []struct {
		name string
		edit func(t *testing.T, root string)
		args []string
		want string
	}{
		{"no --out", func(t *testing.T, root string) {}, nil, "--out"},
		{"an empty --out", func(t *testing.T, root string) {}, []string{"--out", ""}, "--out"},
		{"a stray argument", func(t *testing.T, root string) {}, []string{"--out", "work/site", "extra"}, "unexpected argument"},
		{"an unknown flag", func(t *testing.T, root string) {}, []string{"--nope"}, "site:"},
		{"tw not built", func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, ".work", "tw", ".meta", "build.json")); err != nil {
				t.Fatal(err)
			}
		}, []string{"--out", "work/site"}, "lct build"},
		{"tw built from another snapshot", func(t *testing.T, root string) {
			p := filepath.Join(root, ".work", "tw", ".meta", "build.json")
			b, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			b = []byte(strings.Replace(string(b), `"raw_tag": "raw-20261029"`, `"raw_tag": "raw-20261022"`, 1))
			if err := os.WriteFile(p, b, 0o644); err != nil {
				t.Fatal(err)
			}
		}, []string{"--out", "work/site"}, "raw-20261022"},
		{"no snapshot", func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, ".work", "raw", "meta", "source.json")); err != nil {
				t.Fatal(err)
			}
		}, []string{"--out", "work/site"}, "lct snapshot"},
		{"no tw worktree", func(t *testing.T, root string) {
			if err := os.RemoveAll(filepath.Join(root, ".work", "tw", ".meta")); err != nil {
				t.Fatal(err)
			}
		}, []string{"--out", "work/site"}, "worktree"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := siteRepo(t)
			writeTestFile(t, root, "work/site/old.json", "{}")
			tt.edit(t, root)
			_, err := runSiteJSON(t, root, tt.args...)
			if !isUserError(err) || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want a user error mentioning %q", err, tt.want)
			}
			if _, err := os.Stat(filepath.Join(root, "work", "site", "old.json")); err != nil {
				t.Errorf("the output directory was emptied although the command was refused (%v)", err)
			}
		})
	}
}

func TestSiteCommandTextOutput(t *testing.T) {
	root := siteRepo(t)
	ctx, out := testCtx(t, root, "", false)
	if err := runSite(ctx, []string{"--out", "work/site"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"files", "units", filepath.Join(root, "work", "site")} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output does not mention %q:\n%s", want, out)
		}
	}
}

// An absolute path inside the repo is as good as a relative one.
func TestSiteCommandAcceptsAnAbsolutePathInsideTheRepo(t *testing.T) {
	root := siteRepo(t)
	if _, err := runSiteJSON(t, root, "--out", filepath.Join(root, "work", "abs")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "work", "abs", "meta.json")); err != nil {
		t.Error(err)
	}
}

// Export empties the directory, so a directory that holds a file git tracks is
// refused: the frontend source, the rules. One that holds only untracked or
// ignored files (the data of an earlier run) is the directory to write to.
func TestSiteCommandRefusesADirectoryWithTrackedFiles(t *testing.T) {
	root := siteRepo(t)
	writeTestFile(t, root, ".gitignore", "/.work/\n/frontend/public/data/\n")
	writeTestFile(t, root, "frontend/src/app.ts", "tracked")
	writeTestFile(t, root, "docs/notes.md", "tracked")
	writeTestFile(t, root, "staged/x.txt", "staged, not committed")
	runGitIn(t, root, "add", ".gitignore", "frontend", "docs")
	runGitIn(t, root, "commit", "-q", "-m", "frontend")
	runGitIn(t, root, "add", "staged")
	// Untracked and ignored files of the directories that are allowed.
	writeTestFile(t, root, "frontend/public/data/old.json", "ignored")
	writeTestFile(t, root, "scratch/old.json", "untracked")

	for _, out := range []string{"frontend", "frontend/src", "./frontend/", "staged", "docs", filepath.Join(root, "frontend")} {
		_, err := runSiteJSON(t, root, "--out", out)
		if !isUserError(err) || !strings.Contains(err.Error(), "tracked") {
			t.Errorf("--out %q: err = %v, want a user error that says the directory holds tracked files", out, err)
		}
	}
	for _, kept := range []string{"frontend/src/app.ts", "staged/x.txt", "docs/notes.md"} {
		if b, err := os.ReadFile(filepath.Join(root, kept)); err != nil || len(b) == 0 {
			t.Errorf("%s was removed (%v)", kept, err)
		}
	}

	for _, out := range []string{"frontend/public/data", "scratch", "new/dir", "frontend/new"} {
		if _, err := runSiteJSON(t, root, "--out", out); err != nil {
			t.Errorf("--out %q: %v, want it allowed: no tracked file in it", out, err)
			continue
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(out), "meta.json")); err != nil {
			t.Errorf("--out %q: %v", out, err)
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(out), "old.json")); !os.IsNotExist(err) {
			t.Errorf("--out %q: old.json is still there (%v)", out, err)
		}
	}
	if b, err := os.ReadFile(filepath.Join(root, "frontend", "src", "app.ts")); err != nil || string(b) != "tracked" {
		t.Errorf("frontend/src/app.ts = %q, %v", b, err)
	}
}

// Without history in tw the output has an empty list of updates; with it, the
// releases of the history.
func TestSiteCommandExportsUpdates(t *testing.T) {
	root := siteRepo(t)
	if _, err := runSiteJSON(t, root, "--out", "work/site"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "work", "site", "updates", "index.json"))
	if err != nil || strings.TrimSpace(string(b)) != `{"releases":[]}` {
		t.Fatalf("index.json = %q, %v; want no releases", b, err)
	}
	tw := filepath.Join(root, ".work", "tw")
	runGitIn(t, tw, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "release")
	runGitIn(t, tw, "tag", "zh-TW-20261028.1")
	runGitIn(t, tw, "tag", "zh-TW-20261029.1")
	// The record of a tag that never became a Release is left out.
	for _, rec := range []*history.Record{
		{Tag: "zh-TW-20261029.1", PrevTag: "zh-TW-20261028.1", ReleasedAt: "2026-10-29T00:00:00Z", RawTag: "raw-20261029", PrevRawTag: "raw-20261022"},
		{Tag: "zh-TW-20261030.1", PrevTag: "zh-TW-20261029.1", ReleasedAt: "2026-10-30T00:00:00Z", RawTag: "raw-20261030", PrevRawTag: "raw-20261029"},
	} {
		if err := history.Write(tw, rec); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := runSiteJSON(t, root, "--out", "work/site"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"index.json", "zh-TW-20261029.1.json"} {
		if _, err := os.Stat(filepath.Join(root, "work", "site", "updates", name)); err != nil {
			t.Errorf("updates/%s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "work", "site", "updates", "zh-TW-20261030.1.json")); !os.IsNotExist(err) {
		t.Errorf("updates/zh-TW-20261030.1.json (no Release tag): %v", err)
	}
	b, err = os.ReadFile(filepath.Join(root, "work", "site", "updates", "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var idx site.UpdateIndex
	if err := json.Unmarshal(b, &idx); err != nil || len(idx.Releases) != 2 ||
		idx.Releases[0].Tag != "zh-TW-20261029.1" || !idx.Releases[0].Recorded || idx.Releases[1].Kind != site.UpdateFirst {
		t.Errorf("index.json = %s, %v; want the two Release tags, the newest with its record", b, err)
	}
}

// loadReviewInputs computes from the rules its caller has loaded, and does
// not read them again: rules that differ from the repo's are the ones it uses
// (here a glossary entry of the KR of a provisional term, which keeps the
// term out).
func TestLoadReviewInputsUsesTheRulesItIsGiven(t *testing.T) {
	root := siteRepo(t)
	ctx, _ := testCtx(t, root, "", true)
	st := store.Open(filepath.Join(root, filepath.FromSlash(ctx.Config.Worktrees.TW)))
	if _, err := st.AppendTerms([]store.ProvisionalTerm{{KR: "우레", ZH: "雷鳴", Key: "A.json|/dataList/0/content", Batch: "b1", RawTag: "t", Time: "2026-11-05T00:00:00Z"}}); err != nil {
		t.Fatal(err)
	}
	r, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	in, err := loadReviewInputs(ctx, "site", r)
	if err != nil {
		t.Fatal(err)
	}
	if len(in.Provisional) != 1 || in.Provisional[0].KR != "우레" {
		t.Fatalf("with the repo's rules: provisional = %+v, want 우레", in.Provisional)
	}
	given := *r
	given.Glossary = append(slices.Clone(r.Glossary), rules.Term{KR: "우레", ZH: "雷", Status: rules.StatusApproved})
	if in, err = loadReviewInputs(ctx, "site", &given); err != nil {
		t.Fatal(err)
	}
	if len(in.Provisional) != 0 {
		t.Errorf("with a glossary entry of 우레: provisional = %+v, want none", in.Provisional)
	}
}

// lct site writes the preview data: the shared prompt and guides, keyed by the
// repo-relative paths the batches name, and a file per file of the pack.
func TestSiteCommandWritesThePreview(t *testing.T) {
	root := siteRepo(t)
	got, err := runSiteJSON(t, root, "--out", "work/site")
	if err != nil {
		t.Fatal(err)
	}
	var previewBytes, total int64
	field(t, got, "preview_bytes", &previewBytes)
	field(t, got, "bytes", &total)
	if previewBytes <= 0 || previewBytes >= total {
		t.Errorf("preview_bytes = %d of %d bytes, want some and part of the whole", previewBytes, total)
	}
	var shared site.PreviewShared
	readTestJSON(t, filepath.Join(root, "work", "site", "preview.json"), &shared)
	if shared.RulesCommit == "" || !strings.Contains(shared.Prompt, "#") {
		t.Errorf("preview.json = commit %q, prompt %.30q, want both", shared.RulesCommit, shared.Prompt)
	}
	paths := map[string]bool{}
	for _, g := range shared.Guides {
		paths[g.Path] = g.Text != ""
	}
	if !paths["guides/base.md"] {
		t.Errorf("guides = %v, want guides/base.md with its text", paths)
	}
	var idx site.Index
	readTestJSON(t, filepath.Join(root, "work", "site", "index.json"), &idx)
	for _, f := range idx.Files {
		if _, err := os.Stat(filepath.Join(root, "work", "site", "preview", filepath.FromSlash(f.Path))); err != nil {
			t.Errorf("no preview data for %s: %v", f.Path, err)
		}
	}
}

func readTestJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}
