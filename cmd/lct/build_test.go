package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"golang.org/x/image/font/gofont/goregular"

	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/plan"
	"github.com/tool-jx3/limbus-translate/internal/store"
)

// buildYAML is the lct.yaml of the bootstrap tests with the language directory.
var buildYAML = "lang_dir: zh-TW\n" + bootstrapYAML

// buildRepo makes the repo of the bootstrap tests with a font (Go Regular, a
// small stand-in for the pinned one), its licence and the content licence,
// and imports the fixture, so that the tw store has records.
func buildRepo(t *testing.T) string {
	t.Helper()
	root, upstream := bootstrapRepo(t, false)
	sum := sha256.Sum256(goregular.TTF)
	writeTestFile(t, root, "lct.yaml", buildYAML+
		"font: { path: assets/fonts/SarasaGothicTC-Bold.ttf, version: \"1.0.43\", sha256: \""+hex.EncodeToString(sum[:])+"\" }\n")
	if err := os.MkdirAll(filepath.Join(root, "assets", "fonts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "assets", "fonts", "SarasaGothicTC-Bold.ttf"), goregular.TTF, 0o644); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, "assets/fonts/OFL.txt", "OFL text\n")
	writeTestFile(t, root, "LICENSE-CONTENT", "content license\n")
	// What lct site reads for the preview: the prompt and the guides.
	for _, f := range []string{"prompts/translate.md", "guides/base.md", "guides/families.md"} {
		copyFile(t, filepath.Join(repoRoot, f), filepath.Join(root, f))
	}
	copyDir(t, filepath.Join(repoRoot, "guides/families"), filepath.Join(root, "guides/families"))
	copyDir(t, filepath.Join(repoRoot, "guides/shared"), filepath.Join(root, "guides/shared"))
	if _, err := runBootstrapJSON(t, root, "--upstream", upstream); err != nil {
		t.Fatal(err)
	}
	return root
}

type buildOut struct {
	RawTag      string        `json:"raw_tag"`
	TextZip     string        `json:"text_zip"`
	CompleteZip string        `json:"complete_zip"`
	Report      *build.Report `json:"report"`
	ReportPath  string        `json:"report_path"`

	ReviewIssueParts int `json:"review_issue_parts"`
}

func runBuildJSON(t *testing.T, root string, args ...string) (buildOut, error) {
	t.Helper()
	ctx, out := testCtx(t, root, "", true)
	err := runBuild(ctx, args)
	var got buildOut
	if err == nil {
		if err := json.Unmarshal(out.Bytes(), &got); err != nil {
			t.Fatalf("output is not JSON: %v\n%s", err, out)
		}
	}
	return got, err
}

func zipNames(t *testing.T, name string) []string {
	t.Helper()
	zr, err := zip.OpenReader(name)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	return names
}

func TestBuildCommandBuildsThePack(t *testing.T) {
	root := buildRepo(t)
	t.Setenv(sourceDateEpoch, "")
	got, err := runBuildJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(root, "out")
	if got.RawTag != "raw-20261029" ||
		got.TextZip != filepath.Join(out, "zh-TW_text_raw-20261029.zip") ||
		got.CompleteZip != filepath.Join(out, "zh-TW_complete_raw-20261029.zip") ||
		got.ReportPath != filepath.Join(root, ".work", "tw", ".meta", "reports", "raw-20261029.md") {
		t.Errorf("output = %+v", got)
	}
	// The fixture import leaves two units to translate; they show EN until then.
	if got.Report == nil || got.Report.Pending != 2 || got.Report.StructuralFallback != 0 {
		t.Errorf("report = %+v, want 2 pending", got.Report)
	}

	tw := filepath.Join(root, ".work", "tw")
	lang := filepath.Join(tw, "LimbusCompany_Data", "Lang")
	if b, err := os.ReadFile(filepath.Join(lang, "config.json")); err != nil || string(b) != "{\"lang\": \"zh-TW\"}\n" {
		t.Errorf("config.json = %q, %v", b, err)
	}
	sample, err := os.ReadFile(filepath.Join(lang, "zh-TW", "Sample.json"))
	if err != nil || !strings.Contains(string(sample), "增加5級 [Laceration] 強度") || !strings.Contains(string(sample), "<color=#f8c200>") {
		t.Errorf("Sample.json = %q, %v", sample, err)
	}
	for _, name := range []string{got.ReportPath, filepath.Join(tw, ".meta", "build.json"), got.TextZip, got.CompleteZip} {
		if _, err := os.Stat(name); err != nil {
			t.Error(err)
		}
	}
	if names := zipNames(t, got.CompleteZip); len(names) <= len(zipNames(t, got.TextZip)) {
		t.Errorf("the complete zip (%d files) must hold more than the text zip (%d)", len(names), len(zipNames(t, got.TextZip)))
	}

	// The inputs are recorded; the time is the one of the commit.
	var meta struct {
		RawTag      string            `json:"raw_tag"`
		RawToken    string            `json:"raw_token"`
		RulesCommit string            `json:"rules_commit"`
		BuiltAt     string            `json:"built_at"`
		Upstream    map[string]string `json:"upstream"`
	}
	b, err := os.ReadFile(filepath.Join(lang, "zh-TW", "Info", "version.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &meta); err != nil {
		t.Fatal(err)
	}
	sec, err := strconv.ParseInt(runGitIn(t, root, "log", "-1", "--format=%ct"), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	wantTime := time.Unix(sec, 0).UTC().Format(time.RFC3339)
	// The rules were copied in after the repo's only commit: they are dirty.
	if meta.RawTag != "raw-20261029" || meta.RawToken != testToken || meta.RulesCommit != runGitIn(t, root, "rev-parse", "HEAD")+"+dirty" || meta.BuiltAt != wantTime {
		t.Errorf("version.json = %+v, want the commit time %s and HEAD+dirty", meta, wantTime)
	}
	if meta.Upstream["paratranz"] != "owner/pz@11" || meta.Upstream["ltm"] != "owner/ltm@22" || meta.Upstream["llc"] != "owner/llc@33" {
		t.Errorf("upstream = %v", meta.Upstream)
	}

	// Building again gives the same bytes.
	before, err := os.ReadFile(got.CompleteZip)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runBuildJSON(t, root); err != nil {
		t.Fatal(err)
	}
	if after, err := os.ReadFile(got.CompleteZip); err != nil || string(after) != string(before) {
		t.Errorf("a second build changed the complete zip (%v)", err)
	}

	// Once the rules are committed, the build names the bare commit, in
	// version.json and in build.json.
	runGitIn(t, root, "add", "-A")
	runGitIn(t, root, "commit", "-q", "-m", "rules")
	if _, err := runBuildJSON(t, root); err != nil {
		t.Fatal(err)
	}
	head := runGitIn(t, root, "rev-parse", "HEAD")
	for _, f := range []string{filepath.Join(lang, "zh-TW", "Info", "version.json"), filepath.Join(tw, ".meta", "build.json")} {
		if b, err := os.ReadFile(f); err != nil || !strings.Contains(string(b), `"rules_commit": "`+head+`"`) {
			t.Errorf("%s = %s, %v, want rules_commit %s", f, b, err, head)
		}
	}
}

func TestBuildCommandJSONHasTheReportObject(t *testing.T) {
	root := buildRepo(t)
	t.Setenv(sourceDateEpoch, "")
	ctx, out := testCtx(t, root, "", true)
	if err := runBuild(ctx, nil); err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	var keys []string
	for k := range fields {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	if want := []string{"complete_zip", "raw_tag", "report", "report_path", "review_issue_parts", "text_zip"}; !slices.Equal(keys, want) {
		t.Errorf("output keys = %v, want %v", keys, want)
	}

	// "report" is the Report that .meta/build.json holds.
	var fromOutput build.Report
	if err := json.Unmarshal(fields["report"], &fromOutput); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, ".work", "tw", ".meta", "build.json"))
	if err != nil {
		t.Fatal(err)
	}
	var meta struct {
		Report build.Report `json:"report"`
	}
	if err := json.Unmarshal(b, &meta); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fromOutput, meta.Report) || fromOutput.Pending != 2 {
		t.Errorf("report = %+v, build.json has %+v", fromOutput, meta.Report)
	}
	var path string
	if err := json.Unmarshal(fields["report_path"], &path); err != nil || !strings.HasSuffix(path, "raw-20261029.md") {
		t.Errorf("report_path = %q, %v", path, err)
	}
}

func TestBuildCommandUsesSourceDateEpoch(t *testing.T) {
	root := buildRepo(t)
	t.Setenv(sourceDateEpoch, "1793500000")
	if _, err := runBuildJSON(t, root); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, ".work", "tw", "LimbusCompany_Data", "Lang", "zh-TW", "Info", "version.json"))
	if err != nil || !strings.Contains(string(b), `"built_at": "`+time.Unix(1793500000, 0).UTC().Format(time.RFC3339)+`"`) {
		t.Errorf("version.json = %q, %v", b, err)
	}

	for _, bad := range []string{"yesterday", "-5", "1.5", "0"} {
		t.Setenv(sourceDateEpoch, bad)
		if _, err := runBuildJSON(t, root); !isUserError(err) || !strings.Contains(err.Error(), sourceDateEpoch) {
			t.Errorf("%s=%q: err = %v, want a user error naming the variable", sourceDateEpoch, bad, err)
		}
	}
}

func TestBuildCommandRefusesWhatItCannotBuild(t *testing.T) {
	t.Setenv(sourceDateEpoch, "")
	tests := []struct {
		name string
		edit func(t *testing.T, root string)
		args []string
		want string
	}{
		{"wrong font checksum", func(t *testing.T, root string) {
			writeTestFile(t, root, "lct.yaml", buildYAML+"font: { path: assets/fonts/SarasaGothicTC-Bold.ttf, sha256: \""+strings.Repeat("0", 64)+"\" }\n")
		}, nil, "checksum"},
		{"no font in lct.yaml", func(t *testing.T, root string) {
			writeTestFile(t, root, "lct.yaml", buildYAML)
		}, nil, "font.sha256"},
		{"font file missing", func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, "assets", "fonts", "SarasaGothicTC-Bold.ttf")); err != nil {
				t.Fatal(err)
			}
		}, nil, "SarasaGothicTC-Bold.ttf"},
		{"font licence missing", func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, "assets", "fonts", "OFL.txt")); err != nil {
				t.Fatal(err)
			}
		}, nil, "OFL.txt"},
		{"repo without commits", func(t *testing.T, root string) {
			// With SOURCE_DATE_EPOCH unset, the build time would come from
			// HEAD: the missing commit must be reported before it is asked for.
			runGitIn(t, root, "checkout", "-q", "--orphan", "unborn")
		}, nil, "no commit"},
		{"content licence missing", func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, "LICENSE-CONTENT")); err != nil {
				t.Fatal(err)
			}
		}, nil, "LICENSE-CONTENT"},
		{"no snapshot", func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, ".work", "raw", "meta", "source.json")); err != nil {
				t.Fatal(err)
			}
		}, nil, "lct snapshot"},
		{"stray argument", func(t *testing.T, root string) {}, []string{"extra"}, "unexpected argument"},
		{"unknown flag", func(t *testing.T, root string) {}, []string{"--nope"}, "build:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := buildRepo(t)
			tt.edit(t, root)
			_, err := runBuildJSON(t, root, tt.args...)
			if !isUserError(err) || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want a user error mentioning %q", err, tt.want)
			}
			if _, err := os.Stat(filepath.Join(root, "out")); err == nil {
				t.Error("out/ was created although the build was refused")
			}
			if _, err := os.Stat(filepath.Join(root, ".work", "tw", "LimbusCompany_Data")); err == nil {
				t.Error("the pack was written although the build was refused")
			}
		})
	}
}

func TestBuildCommandTextOutput(t *testing.T) {
	root := buildRepo(t)
	t.Setenv(sourceDateEpoch, "")
	ctx, out := testCtx(t, root, "", false)
	if err := runBuild(ctx, nil); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"raw-20261029", "2 pending", "zh-TW_text_raw-20261029.zip", "zh-TW_complete_raw-20261029.zip", "raw-20261029.md"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output does not mention %q:\n%s", want, out)
		}
	}
}

// markNeedsReview adds n pending items that wait for a person to the tw store
// of root, as batch submit leaves them after the last attempt.
func markNeedsReview(t *testing.T, root string, n int) []store.PendingItem {
	t.Helper()
	st := store.Open(filepath.Join(root, ".work", "tw"))
	pending, err := st.LoadPending()
	if err != nil {
		t.Fatal(err)
	}
	var added []store.PendingItem
	for i := range n {
		added = append(added, store.PendingItem{
			Key:       fmt.Sprintf("Stuck%02d.json|/dataList/%d/content", i%40, i),
			Reason:    "new",
			Attempts:  3,
			Flags:     []string{store.FlagNeedsReview},
			LastError: fmt.Sprintf("id-mismatch: 缺少 [Bleed] 😀 %d", i),
		})
	}
	if err := st.SavePending(append(pending, added...)); err != nil {
		t.Fatal(err)
	}
	return added
}

func reviewIssueDir(root string) string { return filepath.Join(root, "out", "review-issue") }

// The full report is the summary of the run. It goes to out/summary.md.
func TestBuildCommandWritesTheSummary(t *testing.T) {
	root := buildRepo(t)
	t.Setenv(sourceDateEpoch, "")
	got, err := runBuildJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	summary, err := os.ReadFile(filepath.Join(root, "out", "summary.md"))
	if err != nil {
		t.Fatal(err)
	}
	report, err := os.ReadFile(got.ReportPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(summary) != string(report) {
		t.Errorf("out/summary.md is not the report:\n%s", summary)
	}
	if !strings.Contains(string(summary), "## 需要你處理") || !strings.Contains(string(summary), "## 資訊") {
		t.Errorf("the summary lacks the two parts:\n%s", summary)
	}
}

// With nothing that needs a person there is no issue: the files of an earlier
// build are gone, all of them.
func TestReviewIssueOnlyWithItems(t *testing.T) {
	root := buildRepo(t)
	t.Setenv(sourceDateEpoch, "")
	dir := reviewIssueDir(root)
	for _, name := range []string{"title.txt", "01.md", "02.md", "03.md"} {
		writeTestFile(t, dir, name, "stale\n")
	}
	got, err := runBuildJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	if got.ReviewIssueParts != 0 {
		t.Errorf("review_issue_parts = %d, want 0", got.ReviewIssueParts)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("out/review-issue is still there (%v)", err)
	}

	// A build with no stale directory works the same.
	if _, err := runBuildJSON(t, root); err != nil {
		t.Fatal(err)
	}
}

// Units that need a person make the issue: the title, and a file for each
// part, as many as the parts the output says.
func TestBuildCommandWritesTheReviewIssue(t *testing.T) {
	root := buildRepo(t)
	t.Setenv(sourceDateEpoch, "")
	stuck := markNeedsReview(t, root, 3000)
	dir := reviewIssueDir(root)
	writeTestFile(t, dir, "09.md", "stale\n") // of an earlier build with more parts

	got, err := runBuildJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	if got.ReviewIssueParts < 2 {
		t.Fatalf("review_issue_parts = %d, want several parts for 3000 units", got.ReviewIssueParts)
	}
	title, err := os.ReadFile(filepath.Join(dir, "title.txt"))
	if err != nil || strings.TrimSpace(string(title)) != "審核：2026-10-29 更新" {
		t.Errorf("title.txt = %q, %v", title, err)
	}

	// One file per part, in order, and nothing else.
	var names []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		names = append(names, e.Name())
	}
	var want []string // ReadDir sorts: the numbers come before the title
	for i := 1; i <= got.ReviewIssueParts; i++ {
		want = append(want, fmt.Sprintf("%02d.md", i))
	}
	want = append(want, "title.txt")
	if !slices.Equal(names, want) {
		t.Fatalf("files = %v, want %v", names, want)
	}

	items := 0
	for i := 1; i <= got.ReviewIssueParts; i++ {
		b, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("%02d.md", i)))
		if err != nil {
			t.Fatal(err)
		}
		text := string(b)
		if n := len(utf16.Encode([]rune(text))); n > 60000 {
			t.Errorf("part %d has %d UTF-16 code units", i, n)
		}
		if head := fmt.Sprintf("（第 %d／%d 段）", i, got.ReviewIssueParts); !strings.HasPrefix(text, head) {
			t.Errorf("part %d starts %q, want %q", i, text[:min(30, len(text))], head)
		}
		items += strings.Count(text, "- [ ] ")
	}
	if items != len(stuck) {
		t.Errorf("%d items in the parts, want %d", items, len(stuck))
	}

	// The report lists them too, for the summary of the run.
	summary, _ := os.ReadFile(filepath.Join(root, "out", "summary.md"))
	if !strings.Contains(string(summary), stuck[0].Key) || !strings.Contains(string(summary), "id-mismatch") {
		t.Errorf("the summary does not list the units that need a person")
	}

	// Fewer units fit in one part: no header, one file, and the stale ones go.
	st := store.Open(filepath.Join(root, ".work", "tw"))
	pending, err := st.LoadPending()
	if err != nil {
		t.Fatal(err)
	}
	pending = slices.DeleteFunc(pending, func(it store.PendingItem) bool { return slices.Contains(it.Flags, store.FlagNeedsReview) })
	if err := st.SavePending(append(pending, stuck[:3]...)); err != nil {
		t.Fatal(err)
	}
	got, err = runBuildJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	if got.ReviewIssueParts != 1 {
		t.Fatalf("review_issue_parts = %d, want 1", got.ReviewIssueParts)
	}
	entries, _ = os.ReadDir(dir)
	if len(entries) != 2 {
		t.Errorf("%d files, want title.txt and 01.md only", len(entries))
	}
}

// The compare pages of the report come from the repo the workflow runs in and
// from the tw commit the build starts at.
func TestBuildCommandLinksTheComparePages(t *testing.T) {
	root := buildRepo(t)
	t.Setenv(sourceDateEpoch, "")
	t.Setenv("GITHUB_SERVER_URL", "https://github.com")
	t.Setenv("GITHUB_REPOSITORY", "tool-jx3/limbus-translate")
	tw := filepath.Join(root, ".work", "tw")
	head, _, err := gitx.CommitAll(tw, "before")
	if err != nil {
		t.Fatal(err)
	}
	if head == "" {
		t.Fatal("tw has no commit")
	}
	// A plan of this snapshot that follows an older one.
	if err := store.Open(tw).WriteMetaJSON(plan.File, map[string]any{
		"raw_tag": "raw-20261029", "prev_raw_tag": "raw-20261022", "rules_commit": "x",
		"changes": map[string]any{"unchanged": 0, "moved": 0, "changed": 0, "new": 0, "deleted": 0},
		"copy":    0, "reused": 0,
		"adjustments":     map[string]any{"applied": 0, "moved": []any{}, "expired": []any{}},
		"requeue_applied": []any{}, "rewrite_applied": []any{}, "frozen_changed": []any{}, "new_speakers": []any{},
		"field_changes": []any{}, "new_files_default_family": []any{}, "pending": 0,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := runBuildJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	report, err := os.ReadFile(got.ReportPath)
	if err != nil {
		t.Fatal(err)
	}
	summary, err := os.ReadFile(filepath.Join(root, "out", "summary.md"))
	if err != nil {
		t.Fatal(err)
	}
	rawLink := "https://github.com/tool-jx3/limbus-translate/compare/raw-20261022...raw-20261029"
	twLink := "https://github.com/tool-jx3/limbus-translate/compare/" + head + "...tw"
	// The archived report, which is committed to tw, links the KR text only;
	// the summary of the run has the compare page of tw too.
	if !strings.Contains(string(report), rawLink) || strings.Contains(string(report), "...tw") || strings.Contains(string(report), head) {
		t.Errorf("the archived report should link the KR text and nothing of tw's state:\n%s", report)
	}
	for _, want := range []string{rawLink, twLink} {
		if !strings.Contains(string(summary), want) {
			t.Errorf("the summary lacks %q:\n%s", want, summary)
		}
	}

	// Outside GitHub Actions there is no repo to link.
	t.Setenv("GITHUB_SERVER_URL", "")
	t.Setenv("GITHUB_REPOSITORY", "")
	got, err = runBuildJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{got.ReportPath, filepath.Join(root, "out", "summary.md")} {
		if b, _ := os.ReadFile(name); strings.Contains(string(b), "/compare/") {
			t.Errorf("%s has compare links without a repo:\n%s", name, b)
		}
	}
}

// A build of what was published is no change: tw settles. In CI the report
// links the repo, and tw stands somewhere else after every publish; nothing
// committed may follow that, or a rebuild would commit and release a pack that
// did not change.
func TestBuildAfterPublishLeavesTWAlone(t *testing.T) {
	t.Setenv("GITHUB_SERVER_URL", "https://github.com")
	t.Setenv("GITHUB_REPOSITORY", "tool-jx3/limbus-translate")
	root := publishRepo(t)
	tw := filepath.Join(root, ".work", "tw")

	res, _, err := runPublishJSON(t, root)
	if err != nil || !res.Changed {
		t.Fatalf("first publish = %+v, %v, want a commit", res, err)
	}
	for round, advance := range []func(){
		func() {}, // tw stands at the commit of the publish
		func() { runGitIn(t, tw, "commit", "-q", "--allow-empty", "-m", "elsewhere") }, // and now somewhere else
		func() { runGitIn(t, tw, "commit", "-q", "--allow-empty", "-m", "and again") },
	} {
		advance()
		head := runGitIn(t, tw, "rev-parse", "HEAD")
		got, err := runBuildJSON(t, root)
		if err != nil {
			t.Fatal(err)
		}
		if status := runGitIn(t, tw, "status", "--porcelain"); status != "" {
			t.Errorf("round %d: a build of the published inputs changed tw:\n%s", round, status)
		}
		res, _, err := runPublishJSON(t, root)
		if err != nil || res.Changed || res.TWCommit != head {
			t.Errorf("round %d: publish = %+v, %v, want no new commit (tw at %s)", round, res, err, head)
		}
		// The summary of the run does say where tw stood.
		summary, err := os.ReadFile(filepath.Join(root, "out", "summary.md"))
		if err != nil || !strings.Contains(string(summary), "/compare/"+head+"...tw") {
			t.Errorf("round %d: the summary lacks the compare page from %s (%v)", round, head, err)
		}
		if report, err := os.ReadFile(got.ReportPath); err != nil || strings.Contains(string(report), "...tw") || strings.Contains(string(report), head) {
			t.Errorf("round %d: the archived report holds the state of tw (%v)", round, err)
		}
	}
}
