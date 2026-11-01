package publish

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/gitx"
)

const (
	testRawTag   = "raw-20261029"
	testToken    = "l20261029_n2rX1Jku6_9sBiu9Np24"
	testRules    = "0123456789abcdef0123456789abcdef01234567+dirty"
	testReleaseA = "zh-TW-20261029.1"
	testLang     = "zh-TW"
)

// testBuildJSON is .meta/build.json as internal/build writes it.
const testBuildJSON = `{
  "raw_tag": "` + testRawTag + `",
  "raw_token": "` + testToken + `",
  "rules_commit": "` + testRules + `",
  "built_at": "2026-10-29T03:00:00Z",
  "upstream": {
    "paratranz": "owner/pz@11"
  },
  "report": {
    "origins": {
      "claude": 7,
      "copy": 3,
      "paratranz": 30
    },
    "pending": 1234,
    "needs_review": 56,
    "structural_fallback": 789,
    "frozen_fallback": [
      "BgmLyrics/L.json|/dataList/0/content",
      "BgmLyrics/L.json|/dataList/1/content"
    ],
    "unaligned": 4,
    "field_rule_override": 0,
    "warnings": {
      "W1": 2
    },
    "missing_glyphs": [
      26381,
      96,
      7
    ]
  }
}
`

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fixture is a repo with a tw worktree that holds a build, and the out
// directory with the two zips of that build.
type fixture struct {
	root, tw, out string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init", "-q", "-b", "main")
	runGit(t, root, "config", "user.name", "Test")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "commit.gpgsign", "false")
	runGit(t, root, "config", "tag.gpgsign", "false")
	write(t, root, "lct.yaml", "worktrees: { raw: .work/raw, tw: .work/tw }\n")
	write(t, root, ".gitignore", "/.work/\n/out/\n")
	runGit(t, root, "add", "-A")
	runGit(t, root, "commit", "-q", "-m", "init")

	f := fixture{root: root, tw: filepath.Join(root, ".work", "tw"), out: filepath.Join(root, "out")}
	if err := gitx.EnsureWorktree(root, "tw", f.tw); err != nil {
		t.Fatal(err)
	}
	write(t, f.tw, ".meta/build.json", testBuildJSON)
	write(t, f.tw, "LimbusCompany_Data/Lang/config.json", "{\"lang\": \"zh-TW\"}\n")
	write(t, f.tw, "LimbusCompany_Data/Lang/zh-TW/Sample.json", "{}\n")
	write(t, f.out, "zh-TW_text_"+testRawTag+".zip", "text")
	write(t, f.out, "zh-TW_complete_"+testRawTag+".zip", "complete")
	return f
}

func TestReleaseTag(t *testing.T) {
	for _, tc := range []struct {
		name     string
		rawTag   string
		existing []string
		want     string
	}{
		{"first of the day", "raw-20261029", nil, "zh-TW-20261029.1"},
		{"empty list", "raw-20261029", []string{}, "zh-TW-20261029.1"},
		{"after the highest of the day", "raw-20261029", []string{"zh-TW-20261029.1", "zh-TW-20261029.3", "zh-TW-20261105.1"}, "zh-TW-20261029.4"},
		{"the order of the list does not matter", "raw-20261029", []string{"zh-TW-20261029.10", "zh-TW-20261029.2"}, "zh-TW-20261029.11"},
		{"a second snapshot of the day has the same date", "raw-20261029.2", nil, "zh-TW-20261029.1"},
		{"a second snapshot continues the numbers", "raw-20261029.2", []string{"zh-TW-20261029.1"}, "zh-TW-20261029.2"},
		{"other days do not count", "raw-20261105", []string{"zh-TW-20261029.5"}, "zh-TW-20261105.1"},
		{"names that are not release tags are ignored", "raw-20261029",
			[]string{"zh-TW-20261029", "zh-TW-20261029.x", "zh-TW-20261029.1.2", "zh-TW-202610290.9", "raw-20261029.7", "zh-TW-20261029.2"}, "zh-TW-20261029.3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ReleaseTag(tc.rawTag, tc.existing)
			if err != nil || got != tc.want {
				t.Errorf("ReleaseTag(%q, %v) = %q, %v; want %q", tc.rawTag, tc.existing, got, err, tc.want)
			}
		})
	}

	for _, bad := range []string{"", "bogus", "raw-", "raw-2026102", "raw-20261029.", "raw-20261029.x", "raw-20261329", "zh-TW-20261029.1", " raw-20261029"} {
		if got, err := ReleaseTag(bad, nil); err == nil {
			t.Errorf("ReleaseTag(%q) = %q, want an error", bad, got)
		}
	}
}

func testInfo(t *testing.T) BuildInfo {
	t.Helper()
	f := newFixture(t)
	info, err := readBuildInfo(f.tw)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func TestNotes(t *testing.T) {
	info := testInfo(t)
	notes := string(Notes(info, testReleaseA))

	for _, want := range []string{
		testReleaseA, testRawTag, testToken, testRules, // the rules commit is shown as built, "+dirty" included
		"CC BY-NC-SA 4.0", "Project Moon", "Sarasa Gothic", "SIL OFL", "Paratranz",
		"使用 Claude Code 翻譯",
		"LimbusCompany_Data/", "zh-TW",
	} {
		if !strings.Contains(notes, want) {
			t.Errorf("notes lack %q:\n%s", want, notes)
		}
	}
	if strings.Contains(notes, "Anthropic") {
		t.Errorf("notes mention Anthropic:\n%s", notes)
	}
	if !strings.Contains(notes, "https://creativecommons.org/licenses/by-nc-sa/4.0/") {
		t.Errorf("notes lack the link to the licence:\n%s", notes)
	}
	if !strings.HasSuffix(notes, "\n") || strings.HasSuffix(notes, "\n\n") {
		t.Errorf("notes must end in exactly one newline: %q", notes[max(0, len(notes)-20):])
	}

	// The title is the tag; neither it nor any heading names the product.
	lines := strings.Split(notes, "\n")
	if lines[0] != "# "+testReleaseA {
		t.Errorf("first line = %q, want the tag as the title", lines[0])
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "#") && (strings.Contains(line, "Claude Code") || strings.Contains(line, "Anthropic")) {
			t.Errorf("heading %q names the product", line)
		}
	}

	// The seven parts come in the order of the brief.
	order := []string{testReleaseA, "使用 Claude Code 翻譯", "LimbusCompany_Data/", testToken, testRules, "pending", "CC BY-NC-SA 4.0"}
	last := -1
	for _, part := range order {
		i := strings.Index(notes, part)
		if i <= last {
			t.Errorf("%q is at %d, not after %d; notes:\n%s", part, i, last, notes)
		}
		last = max(last, i)
	}
}

// tableRow returns the first table row of notes that names item.
func tableRow(t *testing.T, notes, item string) string {
	t.Helper()
	for _, line := range strings.Split(notes, "\n") {
		if strings.HasPrefix(line, "|") && strings.Contains(line, item) {
			return line
		}
	}
	t.Errorf("no table row names %q:\n%s", item, notes)
	return ""
}

// The upstream sources are credited as "name: repo@tag" under the licence, in
// a fixed order whatever the map order; a build with none lists none.
func TestNotesCreditsTheUpstreamSources(t *testing.T) {
	info := testInfo(t)
	info.Upstream = map[string]string{
		"llc":       "owner/llc@33",
		"zzz":       "owner/zzz@9",
		"paratranz": "owner/pz@11",
		"ltm":       "owner/ltm@22",
	}
	notes := string(Notes(info, testReleaseA))
	licence := strings.Index(notes, "## 授權與來源")
	last := licence
	if licence < 0 {
		t.Fatalf("no licence part:\n%s", notes)
	}
	// An unknown name is not dropped; it follows the known ones.
	for _, line := range []string{"paratranz: owner/pz@11", "ltm: owner/ltm@22", "llc: owner/llc@33", "zzz: owner/zzz@9"} {
		i := strings.Index(notes, "  - "+line+"\n")
		if i <= last {
			t.Errorf("%q is at %d, want it after %d, in the licence part; notes:\n%s", line, i, last, notes)
		}
		last = max(last, i)
	}

	// The same bytes however the map was filled.
	for range 20 {
		if again := string(Notes(info, testReleaseA)); again != notes {
			t.Fatalf("Notes is not deterministic:\n%s\n%s", notes, again)
		}
	}

	for _, empty := range []map[string]string{nil, {}} {
		info.Upstream = empty
		without := string(Notes(info, testReleaseA))
		if strings.Contains(without, "上游") || strings.Contains(without, "@") {
			t.Errorf("notes of a build with no upstream (%#v) list sources:\n%s", empty, without)
		}
		if !strings.Contains(without, "CC BY-NC-SA 4.0") || !strings.HasSuffix(without, "\n") {
			t.Errorf("notes of a build with no upstream are incomplete:\n%s", without)
		}
	}
}

func TestNotesSummarizesTheReport(t *testing.T) {
	notes := string(Notes(testInfo(t), testReleaseA))
	for item, count := range map[string]string{
		"claude": "7", "copy": "3", "paratranz": "30",
		"pending": "1234", "needs_review": "56", "structural_fallback": "789",
		"frozen_fallback": "2", "missing glyphs": "3",
	} {
		if r := tableRow(t, notes, item); !strings.HasSuffix(r, "| "+count+" |") {
			t.Errorf("row for %q = %q, want the count %s", item, r, count)
		}
	}
	// The origins come in alphabetical order, whatever the map order.
	if c, p := strings.Index(notes, "| claude "), strings.Index(notes, "| paratranz "); c < 0 || p < 0 || c > p {
		t.Errorf("origins are not sorted: claude at %d, paratranz at %d", c, p)
	}
}

// A report with nothing in it (no origins, no lists) still gives complete
// notes, and the same bytes every time.
func TestNotesOfAnEmptyReport(t *testing.T) {
	info := BuildInfo{RawTag: testRawTag, RawToken: testToken, RulesCommit: "abc"}
	notes := string(Notes(info, testReleaseA))
	for _, want := range []string{"# " + testReleaseA, testToken, "abc", "CC BY-NC-SA 4.0"} {
		if !strings.Contains(notes, want) {
			t.Errorf("notes lack %q:\n%s", want, notes)
		}
	}
	for _, item := range []string{"pending", "needs_review", "structural_fallback", "frozen_fallback", "missing glyphs"} {
		if r := tableRow(t, notes, item); !strings.HasSuffix(r, "| 0 |") {
			t.Errorf("row for %q = %q, want the count 0", item, r)
		}
	}
	if string(Notes(info, testReleaseA)) != notes {
		t.Error("Notes is not deterministic")
	}
}

func TestPublishCommitsAndTags(t *testing.T) {
	f := newFixture(t)

	res, err := Run(f.root, f.tw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	head := runGit(t, f.tw, "rev-parse", "HEAD")
	if !res.Changed || res.TWCommit != head || res.Tag != testReleaseA {
		t.Errorf("first run = %+v, want a change at %s tagged %s", res, head, testReleaseA)
	}
	if got := runGit(t, f.tw, "log", "-1", "--format=%s"); got != "build: "+testRawTag+" (rules "+testRules+")" {
		t.Errorf("commit message = %q", got)
	}
	// The build is in the commit and the worktree is clean.
	if got := runGit(t, f.tw, "ls-files"); !strings.Contains(got, ".meta/build.json") || !strings.Contains(got, "LimbusCompany_Data/Lang/config.json") {
		t.Errorf("tracked files:\n%s", got)
	}
	if got := runGit(t, f.tw, "status", "--porcelain"); got != "" {
		t.Errorf("the worktree is not clean after publish:\n%s", got)
	}
	wantAssets := []string{
		filepath.Join(f.out, "zh-TW_text_"+testRawTag+".zip"),
		filepath.Join(f.out, "zh-TW_complete_"+testRawTag+".zip"),
	}
	if !slices.Equal(res.Assets, wantAssets) {
		t.Errorf("Assets = %v, want %v", res.Assets, wantAssets)
	}
	if res.NotesPath != filepath.Join(f.out, "release-notes.md") {
		t.Errorf("NotesPath = %q", res.NotesPath)
	}
	notes, err := os.ReadFile(res.NotesPath)
	if err != nil || !strings.HasPrefix(string(notes), "# "+testReleaseA+"\n") || !strings.Contains(string(notes), testToken) {
		t.Errorf("release notes = %q, %v", notes, err)
	}
	// The notes are not part of the tw branch.
	if strings.Contains(runGit(t, f.tw, "ls-files"), "release-notes") {
		t.Error("release-notes.md was committed to tw")
	}

	// Again: nothing changed, so no commit, and as no Release tag exists yet,
	// the same tag.
	commits := runGit(t, f.tw, "rev-list", "--count", "HEAD")
	again, err := Run(f.root, f.tw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	if again.Changed || again.TWCommit != head || again.Tag != testReleaseA {
		t.Errorf("second run = %+v, want no change at %s tagged %s", again, head, testReleaseA)
	}
	if got := runGit(t, f.tw, "rev-list", "--count", "HEAD"); got != commits {
		t.Errorf("commits = %s after the second run, want %s", got, commits)
	}

	// Once the Release exists (its tag is in the repo), the next one is .2,
	// still without a new tw commit.
	if err := gitx.Tag(f.root, testReleaseA); err != nil {
		t.Fatal(err)
	}
	next, err := Run(f.root, f.tw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	if next.Changed || next.TWCommit != head || next.Tag != "zh-TW-20261029.2" {
		t.Errorf("third run = %+v, want no change at %s tagged zh-TW-20261029.2", next, head)
	}
	if notes, err := os.ReadFile(next.NotesPath); err != nil || !strings.HasPrefix(string(notes), "# zh-TW-20261029.2\n") {
		t.Errorf("the notes were not rewritten for the new tag: %q, %v", notes, err)
	}

	// A change in tw is a new commit on top of the first.
	write(t, f.tw, "LimbusCompany_Data/Lang/zh-TW/Sample.json", "{\"changed\": true}\n")
	changed, err := Run(f.root, f.tw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	if !changed.Changed || changed.TWCommit == head || changed.TWCommit != runGit(t, f.tw, "rev-parse", "HEAD") {
		t.Errorf("fourth run = %+v, want a new commit on top of %s", changed, head)
	}
	if parent := runGit(t, f.tw, "rev-parse", "HEAD^"); parent != head {
		t.Errorf("parent = %s, want %s", parent, head)
	}
}

func TestPublishNeedsBuild(t *testing.T) {
	tests := []struct {
		name   string
		damage func(t *testing.T, f fixture)
	}{
		{"no build.json", func(t *testing.T, f fixture) {
			if err := os.Remove(filepath.Join(f.tw, ".meta", "build.json")); err != nil {
				t.Fatal(err)
			}
		}},
		{"no text zip", func(t *testing.T, f fixture) {
			if err := os.Remove(filepath.Join(f.out, "zh-TW_text_"+testRawTag+".zip")); err != nil {
				t.Fatal(err)
			}
		}},
		{"no complete zip", func(t *testing.T, f fixture) {
			if err := os.Remove(filepath.Join(f.out, "zh-TW_complete_"+testRawTag+".zip")); err != nil {
				t.Fatal(err)
			}
		}},
		{"zips of another build", func(t *testing.T, f fixture) {
			for _, kind := range []string{"text", "complete"} {
				old := filepath.Join(f.out, "zh-TW_"+kind+"_"+testRawTag+".zip")
				if err := os.Rename(old, filepath.Join(f.out, "zh-TW_"+kind+"_raw-20260101.zip")); err != nil {
					t.Fatal(err)
				}
			}
		}},
		{"a directory instead of a zip", func(t *testing.T, f fixture) {
			p := filepath.Join(f.out, "zh-TW_text_"+testRawTag+".zip")
			if err := os.Remove(p); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(p, 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{"a raw tag that is not one", func(t *testing.T, f fixture) {
			write(t, f.tw, ".meta/build.json", strings.Replace(testBuildJSON, `"`+testRawTag+`"`, `"bogus"`, 1))
		}},
		{"no raw tag", func(t *testing.T, f fixture) {
			write(t, f.tw, ".meta/build.json", strings.Replace(testBuildJSON, `"`+testRawTag+`"`, `""`, 1))
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.damage(t, f)
			before := runGit(t, f.tw, "status", "--porcelain", "-uall")

			res, err := Run(f.root, f.tw, f.out, testLang)
			if !errors.Is(err, ErrNotBuilt) {
				t.Fatalf("Run = %+v, %v; want ErrNotBuilt", res, err)
			}
			// A refusal leaves tw alone: nothing is committed, and no
			// notes are written.
			if head, err := gitx.Head(f.tw); err != nil || head != "" {
				t.Errorf("tw has a commit after a refusal: %q, %v", head, err)
			}
			if after := runGit(t, f.tw, "status", "--porcelain", "-uall"); after != before {
				t.Errorf("status changed from\n%s\nto\n%s", before, after)
			}
			if _, err := os.Stat(filepath.Join(f.out, "release-notes.md")); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("release-notes.md after a refusal: %v", err)
			}
		})
	}
}

func TestPublishRejectsAnUnreadableBuildJSON(t *testing.T) {
	for name, content := range map[string]string{
		"not JSON":      "{",
		"unknown field": strings.Replace(testBuildJSON, `"built_at"`, `"surprise": 1, "built_at"`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			write(t, f.tw, ".meta/build.json", content)
			res, err := Run(f.root, f.tw, f.out, testLang)
			if err == nil || errors.Is(err, ErrNotBuilt) {
				t.Errorf("Run = %+v, %v; want a system error that is not ErrNotBuilt", res, err)
			}
		})
	}
}

func TestReadBuildInfoOfTheRealFormat(t *testing.T) {
	f := newFixture(t)
	info, err := readBuildInfo(f.tw)
	if err != nil {
		t.Fatal(err)
	}
	want := BuildInfo{
		RawTag: testRawTag, RawToken: testToken, RulesCommit: testRules,
		BuiltAt:  "2026-10-29T03:00:00Z",
		Upstream: map[string]string{"paratranz": "owner/pz@11"},
		Report: build.Report{
			Origins:            map[string]int{"claude": 7, "copy": 3, "paratranz": 30},
			Pending:            1234,
			NeedsReview:        56,
			StructuralFallback: 789,
			FrozenFallback:     []string{"BgmLyrics/L.json|/dataList/0/content", "BgmLyrics/L.json|/dataList/1/content"},
			Unaligned:          4,
			Warnings:           map[string]int{"W1": 2},
			MissingGlyphs:      []rune{26381, 96, 7},
		},
	}
	if !reflect.DeepEqual(info, want) {
		t.Errorf("info = %+v\nwant %+v", info, want)
	}
}

// The zips are named after the language directory the build was given, the
// same way lct build names them.
func TestPublishUsesTheLanguageDirectory(t *testing.T) {
	f := newFixture(t)
	for _, kind := range []string{"text", "complete"} {
		if err := os.Rename(
			filepath.Join(f.out, testLang+"_"+kind+"_"+testRawTag+".zip"),
			filepath.Join(f.out, "zh-XX_"+kind+"_"+testRawTag+".zip")); err != nil {
			t.Fatal(err)
		}
	}

	// The zips of the other language directory are not the build's.
	if res, err := Run(f.root, f.tw, f.out, testLang); !errors.Is(err, ErrNotBuilt) || !strings.Contains(err.Error(), "zh-TW_text_"+testRawTag+".zip") {
		t.Fatalf("Run(%q) = %+v, %v; want ErrNotBuilt naming the zip", testLang, res, err)
	}
	if head, err := gitx.Head(f.tw); err != nil || head != "" {
		t.Fatalf("tw has a commit after a refusal: %q, %v", head, err)
	}

	res, err := Run(f.root, f.tw, f.out, "zh-XX")
	if err != nil {
		t.Fatal(err)
	}
	text, complete := build.ZipPaths(build.Options{OutDir: f.out, LangDir: "zh-XX", RawTag: testRawTag})
	if want := []string{text, complete}; !slices.Equal(res.Assets, want) {
		t.Errorf("Assets = %v, want %v (build.ZipPaths)", res.Assets, want)
	}
	// Only the zips follow the language directory; the Release tag does not.
	if res.Tag != testReleaseA {
		t.Errorf("Tag = %q, want %q", res.Tag, testReleaseA)
	}
}

func TestPublishRejectsALanguageDirectoryThatIsNotAName(t *testing.T) {
	for _, bad := range []string{"", ".", "..", "a/b", `a\b`, "/zh-TW", "../zh-TW"} {
		f := newFixture(t)
		res, err := Run(f.root, f.tw, f.out, bad)
		if err == nil || errors.Is(err, ErrNotBuilt) {
			t.Errorf("Run(lang %q) = %+v, %v; want an error that is not ErrNotBuilt", bad, res, err)
		}
		if head, err := gitx.Head(f.tw); err != nil || head != "" {
			t.Errorf("Run(lang %q) committed to tw: %q, %v", bad, head, err)
		}
	}
}
