package publish

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/history"
	"github.com/tool-jx3/limbus-translate/internal/plan"
)

const (
	testRawTag   = "raw-20261029"
	testToken    = "l20261029_n2rX1Jku6_9sBiu9Np24"
	testRules    = "0123456789abcdef0123456789abcdef01234567"
	testReleaseA = "zh-TW-20261029.1"
	testLang     = "zh-TW"
)

// zipSum is the sha256 build.json records for a zip with the content text.
func zipSum(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// The zips of the fixture: their file names, and what they hold.
const (
	testTextZip     = "zh-TW_text_" + testRawTag + ".zip"
	testCompleteZip = "zh-TW_complete_" + testRawTag + ".zip"
)

// testBuildJSON is .meta/build.json as internal/build writes it.
var testBuildJSON = `{
  "raw_tag": "` + testRawTag + `",
  "raw_token": "` + testToken + `",
  "rules_commit": "` + testRules + `",
  "built_at": "2026-10-29T03:00:00Z",
  "upstream": {
    "paratranz": "owner/pz@11"
  },
  "zips": {
    "` + testCompleteZip + `": "` + zipSum("complete") + `",
    "` + testTextZip + `": "` + zipSum("text") + `"
  },
  "report": {
    "origins": {
      "claude": 7,
      "copy": 3,
      "paratranz": 30
    },
    "shown": {
      "claude": 7,
      "copy": 3,
      "paratranz": 30,
      "stand-in": 1300
    },
    "pending": 1234,
    "stand_in": 1300,
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
	root, tw, raw, out string
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

	f := fixture{root: root, tw: filepath.Join(root, ".work", "tw"), raw: filepath.Join(root, ".work", "raw"), out: filepath.Join(root, "out")}
	if err := gitx.EnsureWorktree(root, "tw", f.tw); err != nil {
		t.Fatal(err)
	}
	// The raw snapshots the history compares: the KR pack is the same in all.
	if err := gitx.EnsureWorktree(root, "raw", f.raw); err != nil {
		t.Fatal(err)
	}
	write(t, f.raw, "kr/KR_Other.json", "{}\n")
	runGit(t, f.raw, "add", "-A")
	runGit(t, f.raw, "commit", "-q", "-m", testRawTag)
	runGit(t, f.raw, "tag", testRawTag)
	write(t, f.tw, ".meta/build.json", testBuildJSON)
	write(t, f.tw, "LimbusCompany_Data/Lang/config.json", "{\"lang\": \"zh-TW\"}\n")
	write(t, f.tw, "LimbusCompany_Data/Lang/zh-TW/Sample.json", "{}\n")
	write(t, f.out, testTextZip, "text")
	write(t, f.out, testCompleteZip, "complete")
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

func testInfo(t *testing.T) build.Info {
	t.Helper()
	f := newFixture(t)
	info, err := build.ReadInfo(f.tw)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func TestNotes(t *testing.T) {
	info := testInfo(t)
	notes := string(Notes(info, nil, testReleaseA, testLang))

	for _, want := range []string{
		testReleaseA, testRawTag, testToken, testRules,
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

	// The parts come in the order of the brief.
	order := []string{testReleaseA, "使用 Claude Code 翻譯", "LimbusCompany_Data/", testToken, testRules, "仍顯示英文", "CC BY-NC-SA 4.0"}
	last := -1
	for _, part := range order {
		i := strings.Index(notes, part)
		if i <= last {
			t.Errorf("%q is at %d, not after %d; notes:\n%s", part, i, last, notes)
		}
		last = max(last, i)
	}
}

// The notes tell the sources that went into the pack from those that were only
// read: paratranz is imported (a line "匯入：paratranz: repo@tag"), the others are
// reference translations that were not imported. They come in a fixed order
// whatever the map order. A build that used no imported record lists none.
func TestNotesSeparatesImportedAndReferenceSources(t *testing.T) {
	info := testInfo(t)
	info.Upstream = map[string]string{
		"llc":       "owner/llc@33",
		"zzz":       "owner/zzz@9",
		"paratranz": "owner/pz@11",
		"ltm":       "owner/ltm@22",
	}
	if info.Report.Origins["paratranz"] == 0 {
		t.Fatal("the fixture has no paratranz records")
	}
	notes := string(Notes(info, nil, testReleaseA, testLang))

	licence := strings.Index(notes, "## 授權與來源")
	imported := strings.Index(notes, "匯入：paratranz: owner/pz@11\n")
	reference := strings.Index(notes, "第一階段的參考譯文（未匯入）")
	if licence < 0 || imported <= licence || reference <= imported {
		t.Fatalf("licence at %d, imported at %d, reference at %d; want them in this order:\n%s", licence, imported, reference, notes)
	}
	last := reference
	// An unknown name is not dropped; it follows the known ones.
	for _, line := range []string{"ltm: owner/ltm@22", "llc: owner/llc@33", "zzz: owner/zzz@9"} {
		i := strings.Index(notes, "  - "+line+"\n")
		if i <= last {
			t.Errorf("%q is at %d, want it after %d, under the reference translations; notes:\n%s", line, i, last, notes)
		}
		last = max(last, i)
	}
	// paratranz is imported, so it is not also listed as a reference.
	if strings.Contains(notes[reference:], "paratranz") {
		t.Errorf("paratranz is listed among the reference translations:\n%s", notes)
	}

	// The same bytes however the map was filled.
	for range 20 {
		if again := string(Notes(info, nil, testReleaseA, testLang)); again != notes {
			t.Fatalf("Notes is not deterministic:\n%s\n%s", notes, again)
		}
	}

	// Nothing was imported: no upstream is listed, whatever the build knows
	// about, and Paratranz is not credited.
	info.Report.Origins, info.Report.Shown = map[string]int{"claude": 7}, map[string]int{"claude": 7}
	for _, upstream := range []map[string]string{info.Upstream, nil, {}} {
		info.Upstream = upstream
		without := string(Notes(info, nil, testReleaseA, testLang))
		for _, banned := range []string{"匯入", "參考譯文", "@", "上游", "paratranz", "ltm", "llc", "Paratranz"} {
			if strings.Contains(without, banned) {
				t.Errorf("notes of a build with no imported records (upstream %#v) mention %q:\n%s", upstream, banned, without)
			}
		}
		if !strings.Contains(without, "CC BY-NC-SA 4.0") || !strings.HasSuffix(without, "\n") {
			t.Errorf("notes of a build with no imported records are incomplete:\n%s", without)
		}
	}
}

// The notes are short: what changed, the share of each source, how much is
// still in English, and the licence. The whole report is not repeated.
func TestNotesAreConcise(t *testing.T) {
	info := testInfo(t)
	info.Report.Origins = map[string]int{"paratranz": 60, "claude": 30, "fallback": 2, "copy": 8}
	info.Report.Shown = map[string]int{"paratranz": 60, "claude": 28, "stand-in": 4, "copy": 8}
	info.Report.Pending, info.Report.StandIn = 2, 4
	sum := &plan.Summary{
		RawTag: testRawTag, PrevRawTag: "raw-20261022",
		Changes: plan.Changes{New: 5, Changed: 4, Deleted: 6, Unchanged: 900},
	}
	notes := string(Notes(info, sum, testReleaseA, testLang))

	for _, want := range []string{
		"paratranz 60.0%", "claude 28.0%", "stand-in 4.0%", "copy 8.0%", // each source, to one decimal
		"仍顯示英文：4 個單位",
		"新增 5", "修改 4", "刪除 6", "raw-20261022", // what changed in the KR text, and since when
		testRawTag, testToken,
	} {
		if !strings.Contains(notes, want) {
			t.Errorf("notes lack %q:\n%s", want, notes)
		}
	}
	// No table of the report, and none of its counts.
	for _, line := range strings.Split(notes, "\n") {
		if strings.HasPrefix(line, "|") {
			t.Errorf("notes hold a table row: %q", line)
		}
	}
	for _, not := range []string{"needs_review", "structural_fallback", "frozen_fallback", "missing glyphs", "origins", "fallback"} {
		if strings.Contains(notes, not) {
			t.Errorf("notes still hold %q of the report:\n%s", not, notes)
		}
	}

	// Units that show English are in the whole the shares are of, and have
	// a share of their own.
	info.Report.Shown = map[string]int{"paratranz": 45, "claude": 45, "stand-in": 10}
	info.Report.Pending, info.Report.StandIn = 10, 10
	notes = string(Notes(info, sum, testReleaseA, testLang))
	for _, want := range []string{"paratranz 45.0%", "claude 45.0%", "stand-in 10.0%", "仍顯示英文：10 個單位"} {
		if !strings.Contains(notes, want) {
			t.Errorf("notes lack %q:\n%s", want, notes)
		}
	}

	// A third of a unit shows as one decimal, rounded.
	info.Report.Shown = map[string]int{"paratranz": 1, "claude": 2}
	info.Report.Pending, info.Report.StandIn = 0, 0
	if notes := string(Notes(info, sum, testReleaseA, testLang)); !strings.Contains(notes, "paratranz 33.3%") || !strings.Contains(notes, "claude 66.7%") {
		t.Errorf("shares are not rounded to one decimal:\n%s", notes)
	}

	// Without a plan, or with a plan that found no older snapshot, the notes
	// say nothing of changes.
	for name, s := range map[string]*plan.Summary{
		"no plan":        nil,
		"first snapshot": {RawTag: testRawTag, PrevRawTag: testRawTag, Changes: plan.Changes{New: 5}},
	} {
		if notes := string(Notes(info, s, testReleaseA, testLang)); strings.Contains(notes, "新增") || strings.Contains(notes, "修改") {
			t.Errorf("%s: notes tell of changes:\n%s", name, notes)
		}
	}
}

// A record that was used and then replaced by EN because it breaks the
// structure of KR is counted by Origins and by StandIn. The shares count each
// unit once, in what it shows: the unit is a stand-in only, and the shares add
// up to the whole.
func TestNotesSharesCountEachUnitOnce(t *testing.T) {
	info := testInfo(t)
	info.Report.Origins = map[string]int{"claude": 10, "paratranz": 90} // one of the claude records broke
	info.Report.Pending, info.Report.StandIn = 0, 1
	info.Report.Shown = map[string]int{"claude": 9, "paratranz": 90, "stand-in": 1}
	notes := string(Notes(info, nil, testReleaseA, testLang))

	for _, want := range []string{"- claude 9.0%", "- paratranz 90.0%", "- stand-in 1.0%", "全部 100 個單位"} {
		if !strings.Contains(notes, want) {
			t.Errorf("notes lack %q:\n%s", want, notes)
		}
	}
	if strings.Contains(notes, "claude 10.0%") {
		t.Errorf("the replaced record is also counted as claude:\n%s", notes)
	}
	if got := sharesTotal(t, notes); got < 99.95 || got > 100.05 {
		t.Errorf("the shares add up to %.1f%%, want 100.0%%:\n%s", got, notes)
	}

	// The same when units wait for a translation or show KR, and the pending
	// ones have no record in Origins at all.
	info.Report.Origins = map[string]int{"claude": 4, "copy": 3}
	info.Report.Pending, info.Report.StandIn = 2, 3
	info.Report.Shown = map[string]int{"claude": 3, "copy": 4, "stand-in": 3}
	notes = string(Notes(info, nil, testReleaseA, testLang))
	if got := sharesTotal(t, notes); got < 99.95 || got > 100.05 {
		t.Errorf("the shares add up to %.1f%%, want 100.0%%:\n%s", got, notes)
	}
	for _, want := range []string{"- claude 30.0%", "- copy 40.0%", "- stand-in 30.0%", "全部 10 個單位"} {
		if !strings.Contains(notes, want) {
			t.Errorf("notes lack %q:\n%s", want, notes)
		}
	}
}

// sharesTotal adds up the percentages of the lines of the notes that give the
// share of a source ("- name 12.3%").
func sharesTotal(t *testing.T, notes string) float64 {
	t.Helper()
	re := regexp.MustCompile(`(?m)^- [a-z-]+ (\d+\.\d)%`)
	matches := re.FindAllStringSubmatch(notes, -1)
	if len(matches) == 0 {
		t.Fatalf("notes give no share:\n%s", notes)
	}
	total := 0.0
	for _, m := range matches {
		v, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			t.Fatal(err)
		}
		total += v
	}
	return total
}

// Every unit that shows English counts once: the stand-ins the build counted,
// not pending plus the records that fell back. A build record from before the
// stand-ins were counted still gives at least the pending units.
func TestNotesCountEnglishOnce(t *testing.T) {
	info := testInfo(t)
	info.Report.Pending, info.Report.StandIn = 5, 12
	if notes := string(Notes(info, nil, testReleaseA, testLang)); !strings.Contains(notes, "仍顯示英文：12 個單位") ||
		!strings.Contains(notes, "尚未翻譯 5") {
		t.Errorf("notes do not count the stand-ins once:\n%s", notes)
	}
	info.Report.StandIn = 0
	if notes := string(Notes(info, nil, testReleaseA, testLang)); !strings.Contains(notes, "仍顯示英文：5 個單位") {
		t.Errorf("notes of an old build record lack the pending units:\n%s", notes)
	}
}

// A report with nothing in it (no origins, no lists) still gives complete
// notes, and the same bytes every time.
func TestNotesOfAnEmptyReport(t *testing.T) {
	info := build.Info{RawTag: testRawTag, RawToken: testToken, RulesCommit: "abc"}
	notes := string(Notes(info, nil, testReleaseA, testLang))
	for _, want := range []string{"# " + testReleaseA, testToken, "abc", "CC BY-NC-SA 4.0", "仍顯示英文：0 個單位"} {
		if !strings.Contains(notes, want) {
			t.Errorf("notes lack %q:\n%s", want, notes)
		}
	}
	if strings.Contains(notes, "NaN") || strings.Contains(notes, "%") {
		t.Errorf("notes of an empty report hold a share:\n%s", notes)
	}
	if string(Notes(info, nil, testReleaseA, testLang)) != notes {
		t.Error("Notes is not deterministic")
	}
}

// section returns the part of the notes under the heading, up to the next one.
func section(t *testing.T, notes, heading string) string {
	t.Helper()
	_, rest, ok := strings.Cut(notes, heading+"\n")
	if !ok {
		t.Fatalf("notes have no %q:\n%s", heading, notes)
	}
	body, _, _ := strings.Cut(rest, "\n## ")
	return body
}

// The install text names the language directory lct.yaml gives (lang_dir), the
// one the build wrote config.json for, not a fixed one.
func TestNotesUseTheLangDir(t *testing.T) {
	const lang = "xx-YY"
	info := testInfo(t)

	install := section(t, string(Notes(info, nil, testReleaseA, lang)), "## 安裝方式")
	if !strings.Contains(install, "選擇 "+lang+"。") {
		t.Errorf("the install text does not name %s:\n%s", lang, install)
	}
	if strings.Contains(install, testLang) {
		t.Errorf("the install text still names %s:\n%s", testLang, install)
	}
	// The default language directory is named as before.
	if install := section(t, string(Notes(info, nil, testReleaseA, testLang)), "## 安裝方式"); !strings.Contains(install, "選擇 "+testLang+"。") {
		t.Errorf("the install text does not name %s:\n%s", testLang, install)
	}

	// Run writes the notes with the language directory it was given. The zips
	// are named after it.
	f := newFixture(t)
	write(t, f.tw, ".meta/build.json", strings.ReplaceAll(testBuildJSON, testLang+"_", lang+"_"))
	write(t, f.out, lang+"_text_"+testRawTag+".zip", "text")
	write(t, f.out, lang+"_complete_"+testRawTag+".zip", "complete")
	res, err := Run(f.root, f.tw, f.raw, f.out, lang)
	if err != nil {
		t.Fatal(err)
	}
	notes, err := os.ReadFile(res.NotesPath)
	if err != nil {
		t.Fatal(err)
	}
	if install := section(t, string(notes), "## 安裝方式"); !strings.Contains(install, "選擇 "+lang+"。") || strings.Contains(install, testLang) {
		t.Errorf("the notes Run wrote do not name only %s in the install text:\n%s", lang, install)
	}
}

// publish reads the plan of the snapshot that was built for the notes; the
// plan of another snapshot is not read as its own.
func TestPublishNotesCarryThePlan(t *testing.T) {
	f := newFixture(t)
	writePlan := func(sum plan.Summary) {
		b, err := json.MarshalIndent(sum, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		write(t, f.tw, ".meta/plan.json", string(b)+"\n")
	}
	writePlan(plan.Summary{
		RawTag: testRawTag, PrevRawTag: "raw-20261022",
		Changes: plan.Changes{New: 15, Changed: 14, Deleted: 16},
	})
	res, err := Run(f.root, f.tw, f.raw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	notes, err := os.ReadFile(res.NotesPath)
	if err != nil || !strings.Contains(string(notes), "新增 15") || !strings.Contains(string(notes), "刪除 16") {
		t.Errorf("notes = %q, %v, want the changes of the plan", notes, err)
	}

	writePlan(plan.Summary{RawTag: "raw-20260101", PrevRawTag: "raw-20251225", Changes: plan.Changes{New: 77}})
	res, err = Run(f.root, f.tw, f.raw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	if notes, err := os.ReadFile(res.NotesPath); err != nil || strings.Contains(string(notes), "77") || strings.Contains(string(notes), "新增") {
		t.Errorf("notes = %q, %v, want nothing of the plan of another snapshot", notes, err)
	}
}

func TestPublishCommitsAndTags(t *testing.T) {
	f := newFixture(t)

	res, err := Run(f.root, f.tw, f.raw, f.out, testLang)
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
	again, err := Run(f.root, f.tw, f.raw, f.out, testLang)
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
	next, err := Run(f.root, f.tw, f.raw, f.out, testLang)
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
	changed, err := Run(f.root, f.tw, f.raw, f.out, testLang)
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

			res, err := Run(f.root, f.tw, f.raw, f.out, testLang)
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
			res, err := Run(f.root, f.tw, f.raw, f.out, testLang)
			if err == nil || errors.Is(err, ErrNotBuilt) {
				t.Errorf("Run = %+v, %v; want a system error that is not ErrNotBuilt", res, err)
			}
		})
	}
}

func TestPublishReadsTheRealFormat(t *testing.T) {
	f := newFixture(t)
	info, err := build.ReadInfo(f.tw)
	if err != nil {
		t.Fatal(err)
	}
	want := build.Info{
		RawTag: testRawTag, RawToken: testToken, RulesCommit: testRules,
		BuiltAt:  "2026-10-29T03:00:00Z",
		Upstream: map[string]string{"paratranz": "owner/pz@11"},
		Zips:     map[string]string{testTextZip: zipSum("text"), testCompleteZip: zipSum("complete")},
		Report: build.Report{
			Origins:            map[string]int{"claude": 7, "copy": 3, "paratranz": 30},
			Shown:              map[string]int{"claude": 7, "copy": 3, "paratranz": 30, "stand-in": 1300},
			Pending:            1234,
			StandIn:            1300,
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
	if res, err := Run(f.root, f.tw, f.raw, f.out, testLang); !errors.Is(err, ErrNotBuilt) || !strings.Contains(err.Error(), "zh-TW_text_"+testRawTag+".zip") {
		t.Fatalf("Run(%q) = %+v, %v; want ErrNotBuilt naming the zip", testLang, res, err)
	}
	if head, err := gitx.Head(f.tw); err != nil || head != "" {
		t.Fatalf("tw has a commit after a refusal: %q, %v", head, err)
	}

	// A build with lang_dir zh-XX names the zips, in build.json too, after it.
	write(t, f.tw, ".meta/build.json", strings.NewReplacer(
		testTextZip, "zh-XX_text_"+testRawTag+".zip",
		testCompleteZip, "zh-XX_complete_"+testRawTag+".zip",
	).Replace(testBuildJSON))
	res, err := Run(f.root, f.tw, f.raw, f.out, "zh-XX")
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
		res, err := Run(f.root, f.tw, f.raw, f.out, bad)
		if err == nil || errors.Is(err, ErrNotBuilt) {
			t.Errorf("Run(lang %q) = %+v, %v; want an error that is not ErrNotBuilt", bad, res, err)
		}
		if head, err := gitx.Head(f.tw); err != nil || head != "" {
			t.Errorf("Run(lang %q) committed to tw: %q, %v", bad, head, err)
		}
	}
}

// A build made from rules with changes that were not committed cannot be
// reproduced from the repo, so it is not published; tw gets no commit.
func TestPublishRefusesDirtyRules(t *testing.T) {
	f := newFixture(t)
	write(t, f.tw, ".meta/build.json", strings.Replace(testBuildJSON, testRules, "abc+dirty", 1))
	before := runGit(t, f.tw, "status", "--porcelain", "-uall")

	res, err := Run(f.root, f.tw, f.raw, f.out, testLang)
	if !errors.Is(err, ErrDirtyRules) {
		t.Fatalf("Run = %+v, %v; want ErrDirtyRules", res, err)
	}
	if errors.Is(err, ErrNotBuilt) {
		t.Errorf("a dirty build is not an unbuilt one: %v", err)
	}
	if head, err := gitx.Head(f.tw); err != nil || head != "" {
		t.Errorf("tw has a commit after a refusal: %q, %v", head, err)
	}
	if after := runGit(t, f.tw, "status", "--porcelain", "-uall"); after != before {
		t.Errorf("status changed from\n%s\nto\n%s", before, after)
	}
	if _, err := os.Stat(filepath.Join(f.out, "release-notes.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("release-notes.md after a refusal: %v", err)
	}
}

// The zips in out/ must be the ones the build made, byte for byte: after a
// second build, or by hand, they may be another build's.
func TestPublishRefusesZipsOfAnotherBuild(t *testing.T) {
	for _, tc := range []struct {
		name   string
		damage func(t *testing.T, f fixture)
		zip    string // the zip the error must name
	}{
		{"text zip", func(t *testing.T, f fixture) { write(t, f.out, testTextZip, "another build") }, testTextZip},
		{"complete zip", func(t *testing.T, f fixture) { write(t, f.out, testCompleteZip, "another build") }, testCompleteZip},
		{"no checksum recorded", func(t *testing.T, f fixture) {
			write(t, f.tw, ".meta/build.json", strings.Replace(testBuildJSON, `"`+testTextZip+`": "`+zipSum("text")+`"`, `"other.zip": "0"`, 1))
		}, testTextZip},
		{"no checksums at all", func(t *testing.T, f fixture) {
			var m map[string]json.RawMessage
			if err := json.Unmarshal([]byte(testBuildJSON), &m); err != nil {
				t.Fatal(err)
			}
			delete(m, "zips")
			b, err := json.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			write(t, f.tw, ".meta/build.json", string(b))
		}, testTextZip},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.damage(t, f)
			before := runGit(t, f.tw, "status", "--porcelain", "-uall")

			res, err := Run(f.root, f.tw, f.raw, f.out, testLang)
			if !errors.Is(err, ErrNotBuilt) || !strings.Contains(err.Error(), tc.zip) {
				t.Fatalf("Run = %+v, %v; want ErrNotBuilt naming %s", res, err, tc.zip)
			}
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

// A Release is needed when this run made a commit, or when the commit has no
// Release tag yet (a Release that failed halfway). A tag that is not a Release
// tag does not count.
func TestPublishReleaseNeeded(t *testing.T) {
	f := newFixture(t)

	first, err := Run(f.root, f.tw, f.raw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Changed || !first.ReleaseNeeded {
		t.Fatalf("first run = %+v, want Changed and ReleaseNeeded", first)
	}

	// Unchanged, and no Release tag on the commit: the Release is still due.
	if err := gitx.Tag(f.tw, "other-20261029"); err != nil {
		t.Fatal(err)
	}
	pending, err := Run(f.root, f.tw, f.raw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	if pending.Changed || !pending.ReleaseNeeded || pending.TWCommit != first.TWCommit {
		t.Errorf("second run = %+v, want no change but a Release needed at %s", pending, first.TWCommit)
	}

	// The Release was made: its tag is on the commit of tw.
	if err := gitx.Tag(f.tw, "zh-TW-20261029.1"); err != nil {
		t.Fatal(err)
	}
	done, err := Run(f.root, f.tw, f.raw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	if done.Changed || done.ReleaseNeeded || done.TWCommit != first.TWCommit {
		t.Errorf("third run = %+v, want neither a change nor a Release needed", done)
	}

	// A new commit needs a Release, whatever the tags of the older ones.
	write(t, f.tw, "LimbusCompany_Data/Lang/zh-TW/Sample.json", "{\"changed\": true}\n")
	next, err := Run(f.root, f.tw, f.raw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	if !next.Changed || !next.ReleaseNeeded {
		t.Errorf("fourth run = %+v, want Changed and ReleaseNeeded", next)
	}

	// The workflow reads it as release_needed.
	data, err := json.Marshal(done)
	if err != nil || !strings.Contains(string(data), `"release_needed":false`) {
		t.Errorf("result JSON = %s, %v, want release_needed", data, err)
	}
}

// versionJSON is Info/version.json as a build with the rules commit rules
// would write it: the only file of the pack that differs between two builds
// of the same translations.
func versionJSON(rules string) string {
	return `{"raw_tag": "` + testRawTag + `", "rules_commit": "` + rules + `"}` + "\n"
}

const (
	testRulesB = "89abcdef0123456789abcdef0123456789abcdef"
	versionRel = "LimbusCompany_Data/Lang/zh-TW/Info/version.json"
	sampleRel  = "LimbusCompany_Data/Lang/zh-TW/Sample.json"
)

// rebuild is what lct build does to the worktree for the rules commit rules:
// Info/version.json and the rules commit of .meta/build.json change, and the
// translations are those of records (Sample.json).
func (f fixture) rebuild(t *testing.T, rules, records string) {
	t.Helper()
	write(t, f.tw, versionRel, versionJSON(rules))
	write(t, f.tw, ".meta/build.json", strings.Replace(testBuildJSON, testRules, rules, 1))
	write(t, f.tw, sampleRel, records)
}

// releaseWith publishes the worktree and, as the workflow does when a Release
// is needed, tags the commit of tw with the tag of the result.
func (f fixture) releaseWith(t *testing.T) *Result {
	t.Helper()
	res, err := Run(f.root, f.tw, f.raw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	if !res.ReleaseNeeded {
		t.Fatalf("Run = %+v, want a Release needed", res)
	}
	if err := gitx.Tag(f.tw, res.Tag); err != nil {
		t.Fatal(err)
	}
	return res
}

func TestLastReleaseOrdering(t *testing.T) {
	repo := newFixture(t).root
	if got, err := LastRelease(repo); err != nil || got != "" {
		t.Errorf("LastRelease without Release tags = %q, %v, want \"\"", got, err)
	}

	// The tags need a commit to point to; they all point to the same one.
	for _, name := range []string{"zh-TW-20261029.1", "zh-TW-20261029.2", "zh-TW-20261029.10"} {
		if err := gitx.Tag(repo, name); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := LastRelease(repo); err != nil || got != "zh-TW-20261029.10" {
		t.Errorf("LastRelease = %q, %v, want zh-TW-20261029.10 (.10 comes after .9 and .2)", got, err)
	}

	// A later day wins whatever N the earlier day reached; names that are not
	// Release tags do not count, however late they sort.
	for _, name := range []string{"zh-TW-20261105.1", "raw-20261112", "zh-TW-20261112", "zh-TW-20261112.x", "zh-TW-latest"} {
		if err := gitx.Tag(repo, name); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := LastRelease(repo); err != nil || got != "zh-TW-20261105.1" {
		t.Errorf("LastRelease = %q, %v, want zh-TW-20261105.1", got, err)
	}
}

func TestLastReleaseIgnoresOtherTags(t *testing.T) {
	repo := newFixture(t).root
	for _, name := range []string{"raw-20261112", "v1", "zh-TW-20261112"} {
		if err := gitx.Tag(repo, name); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := LastRelease(repo); err != nil || got != "" {
		t.Errorf("LastRelease = %q, %v, want \"\" when no tag is a Release tag", got, err)
	}
}

func TestSamePack(t *testing.T) {
	f := newFixture(t)
	f.rebuild(t, testRules, "{\"a\": 1}\n")
	write(t, f.tw, "LimbusCompany_Data/Lang/zh-TW/Info/LICENSE-CONTENT", "licence\n")
	commit := func(msg string) string {
		t.Helper()
		runGit(t, f.tw, "add", "-A")
		runGit(t, f.tw, "commit", "-q", "-m", msg)
		return runGit(t, f.tw, "rev-parse", "HEAD")
	}
	base := commit("base")

	// Only version.json and what is outside LimbusCompany_Data/ differ.
	f.rebuild(t, testRulesB, "{\"a\": 1}\n")
	write(t, f.tw, ".meta/plan.json", "{}\n")
	versionOnly := commit("version only")

	// A translation differs.
	write(t, f.tw, sampleRel, "{\"a\": 2}\n")
	edited := commit("edited")

	// A file is added; a file is removed.
	write(t, f.tw, "LimbusCompany_Data/Lang/zh-TW/New.json", "{}\n")
	added := commit("added")
	runGit(t, f.tw, "rm", "-q", sampleRel)
	removed := commit("removed")

	// version.json of another language directory is a file of the pack.
	f.rebuild(t, testRulesB, "{\"a\": 1}\n")
	write(t, f.tw, "LimbusCompany_Data/Lang/zh-XX/Info/version.json", "one\n")
	other1 := commit("other 1")
	write(t, f.tw, "LimbusCompany_Data/Lang/zh-XX/Info/version.json", "two\n")
	other2 := commit("other 2")

	for _, tc := range []struct {
		name     string
		a, b     string
		langDir  string
		want     bool
		wantFail bool
	}{
		{"a commit with itself", base, base, testLang, true, false},
		{"only version.json differs", base, versionOnly, testLang, true, false},
		{"only version.json differs, the other way", versionOnly, base, testLang, true, false},
		{"version.json counts when it is not the one of the language directory", base, versionOnly, "zh-XX", false, false},
		{"a translation differs", versionOnly, edited, testLang, false, false},
		{"a file was added", edited, added, testLang, false, false},
		{"a file was removed", added, removed, testLang, false, false},
		{"the version.json of another language directory differs", other1, other2, testLang, false, false},
		{"HEAD is a revision too", base, "HEAD", testLang, false, false},
		{"no such revision", base, "no-such-rev", testLang, false, true},
		{"no such revision, first", "no-such-rev", base, testLang, false, true},
		{"language directory with a slash", base, base, "a/b", false, true},
		{"empty language directory", base, base, "", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SamePack(f.tw, tc.a, tc.b, tc.langDir)
			if (err != nil) != tc.wantFail || got != tc.want {
				t.Errorf("SamePack(%s, %s, %q) = %v, %v; want %v (error: %v)", tc.a, tc.b, tc.langDir, got, err, tc.want, tc.wantFail)
			}
		})
	}
}

// The first Release: no tag of a Release exists, so there is nothing to
// compare with.
func TestPublishFirstRelease(t *testing.T) {
	f := newFixture(t)
	f.rebuild(t, testRules, "{}\n")
	res, err := Run(f.root, f.tw, f.raw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Changed || !res.ReleaseNeeded || res.SameAs != "" || res.Tag != testReleaseA {
		t.Errorf("first run = %+v, want a Release needed, same as nothing, tagged %s", res, testReleaseA)
	}
	// Until the Release is made, the next run says so again.
	again, err := Run(f.root, f.tw, f.raw, f.out, testLang)
	if err != nil || !again.ReleaseNeeded || again.SameAs != "" {
		t.Errorf("second run = %+v, %v, want the Release still needed", again, err)
	}
}

// A build whose only difference from the last Release is Info/version.json
// (the rules commit changed, the translations did not) is committed to tw but
// not released.
func TestPublishSkipsAnIdenticalPack(t *testing.T) {
	f := newFixture(t)
	f.rebuild(t, testRules, "{\"a\": 1}\n")
	first := f.releaseWith(t)

	f.rebuild(t, testRulesB, "{\"a\": 1}\n")
	res, err := Run(f.root, f.tw, f.raw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Changed || res.ReleaseNeeded || res.SameAs != first.Tag {
		t.Errorf("run = %+v, want a commit, no Release, same as %s", res, first.Tag)
	}
	if res.TWCommit == first.TWCommit || res.TWCommit != runGit(t, f.tw, "rev-parse", "HEAD") {
		t.Errorf("TWCommit = %s, want a new commit on top of %s", res.TWCommit, first.TWCommit)
	}
	// The notes and the tag are those of the build all the same.
	if res.Tag != "zh-TW-20261029.2" {
		t.Errorf("Tag = %q, want zh-TW-20261029.2", res.Tag)
	}
	if notes, err := os.ReadFile(res.NotesPath); err != nil || !strings.HasPrefix(string(notes), "# zh-TW-20261029.2\n") || !strings.Contains(string(notes), testRulesB) {
		t.Errorf("notes = %q, %v, want the notes of the build", notes, err)
	}

	// Running again changes nothing, and still no Release.
	again, err := Run(f.root, f.tw, f.raw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	if again.Changed || again.ReleaseNeeded || again.SameAs != first.Tag || again.TWCommit != res.TWCommit {
		t.Errorf("run again = %+v, want no change, no Release, same as %s", again, first.Tag)
	}

	// The JSON the workflow reads says which Release it equals.
	data, err := json.Marshal(res)
	if err != nil || !strings.Contains(string(data), `"same_as":"`+first.Tag+`"`) || !strings.Contains(string(data), `"release_needed":false`) {
		t.Errorf("result JSON = %s, %v", data, err)
	}
}

func TestPublishReleasesWhenThePackChanged(t *testing.T) {
	f := newFixture(t)
	f.rebuild(t, testRules, "{\"a\": 1}\n")
	f.releaseWith(t)

	f.rebuild(t, testRulesB, "{\"a\": 2}\n")
	res, err := Run(f.root, f.tw, f.raw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Changed || !res.ReleaseNeeded || res.SameAs != "" {
		t.Errorf("run = %+v, want a commit and a Release, same as nothing", res)
	}
	data, err := json.Marshal(res)
	if err != nil || strings.Contains(string(data), "same_as") {
		t.Errorf("result JSON = %s, %v, want no same_as key when the pack is new", data, err)
	}
}

// The pack is compared with the last Release, not with the commit before: two
// builds that differ from each other by version.json only, after a build that
// changed the pack, are all of that new pack.
func TestPublishComparesWithTheLastRelease(t *testing.T) {
	f := newFixture(t)
	f.rebuild(t, testRules, "{\"a\": 1}\n")
	first := f.releaseWith(t)

	// Version only: no Release.
	f.rebuild(t, testRulesB, "{\"a\": 1}\n")
	if res, err := Run(f.root, f.tw, f.raw, f.out, testLang); err != nil || res.ReleaseNeeded || res.SameAs != first.Tag {
		t.Fatalf("version-only build = %+v, %v, want no Release, same as %s", res, err, first.Tag)
	}
	// The pack changes: a Release, though the commit before it had none.
	f.rebuild(t, testRulesB, "{\"a\": 2}\n")
	second := f.releaseWith(t)
	if second.SameAs != "" {
		t.Errorf("changed pack: SameAs = %q", second.SameAs)
	}
	// Back to the first Release's translations: not the same as the last one.
	f.rebuild(t, testRules, "{\"a\": 1}\n")
	if res, err := Run(f.root, f.tw, f.raw, f.out, testLang); err != nil || !res.ReleaseNeeded || res.SameAs != "" {
		t.Errorf("reverted build = %+v, %v, want a Release (the last one is %s)", res, err, second.Tag)
	}
}

// A pack that changed, committed by a run whose Release did not follow, keeps
// needing its Release on the next run, which commits nothing.
func TestPublishReleasesAfterAFailedRelease(t *testing.T) {
	f := newFixture(t)
	f.rebuild(t, testRules, "{\"a\": 1}\n")
	f.releaseWith(t)

	f.rebuild(t, testRulesB, "{\"a\": 2}\n")
	failed, err := Run(f.root, f.tw, f.raw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	if !failed.Changed || !failed.ReleaseNeeded {
		t.Fatalf("run = %+v, want a commit and a Release", failed)
	}
	// The Release failed: its tag was not made.
	again, err := Run(f.root, f.tw, f.raw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	if again.Changed || !again.ReleaseNeeded || again.SameAs != "" || again.TWCommit != failed.TWCommit || again.Tag != failed.Tag {
		t.Errorf("run again = %+v, want no change, a Release for %s at %s", again, failed.Tag, failed.TWCommit)
	}
}

func TestCheckTW(t *testing.T) {
	f := newFixture(t)
	check := func() *Check {
		t.Helper()
		c, err := CheckTW(f.tw, testLang)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}

	// A tw with no commit has no pack to release.
	if c := check(); *c != (Check{}) {
		t.Errorf("check of an empty tw = %+v, want nothing", c)
	}

	// A commit and no Release at all.
	f.rebuild(t, testRules, "{\"a\": 1}\n")
	runGit(t, f.tw, "add", "-A")
	runGit(t, f.tw, "commit", "-q", "-m", "build 1")
	head1 := runGit(t, f.tw, "rev-parse", "HEAD")
	if c := check(); *c != (Check{TWCommit: head1, ReleaseNeeded: true}) {
		t.Errorf("check before the first Release = %+v, want a Release needed at %s", c, head1)
	}
	if err := gitx.Tag(f.tw, testReleaseA); err != nil {
		t.Fatal(err)
	}
	if c := check(); *c != (Check{TWCommit: head1, SameAs: testReleaseA, Tag: testReleaseA}) {
		t.Errorf("check of a released commit = %+v, want it tagged and the same as %s", c, testReleaseA)
	}

	// A version-only commit: no Release, the same pack as the last one.
	f.rebuild(t, testRulesB, "{\"a\": 1}\n")
	runGit(t, f.tw, "add", "-A")
	runGit(t, f.tw, "commit", "-q", "-m", "build 2")
	head2 := runGit(t, f.tw, "rev-parse", "HEAD")
	if c := check(); *c != (Check{TWCommit: head2, SameAs: testReleaseA}) {
		t.Errorf("check of a version-only commit = %+v, want no Release, same as %s", c, testReleaseA)
	}

	// A commit with another pack.
	f.rebuild(t, testRulesB, "{\"a\": 2}\n")
	runGit(t, f.tw, "add", "-A")
	runGit(t, f.tw, "commit", "-q", "-m", "build 3")
	head3 := runGit(t, f.tw, "rev-parse", "HEAD")
	if c := check(); *c != (Check{TWCommit: head3, ReleaseNeeded: true}) {
		t.Errorf("check of a new pack = %+v, want a Release needed at %s", c, head3)
	}

	// A tag that starts like a Release tag and is not one does not count, and
	// the text says what the check found.
	if err := gitx.Tag(f.tw, "zh-TW-latest"); err != nil {
		t.Fatal(err)
	}
	if c := check(); *c != (Check{TWCommit: head3, ReleaseNeeded: true}) || c.String() != "tw at "+head3+": release needed" {
		t.Errorf("check of a new pack with a tag that is no Release tag = %+v (%q), want a Release needed at %s", c, c.String(), head3)
	}

	// Files that are not committed do not count: the check reads commits.
	write(t, f.tw, sampleRel, "{\"a\": 3}\n")
	if c := check(); *c != (Check{TWCommit: head3, ReleaseNeeded: true}) {
		t.Errorf("check with an uncommitted change = %+v", c)
	}
	if got := runGit(t, f.tw, "rev-parse", "HEAD"); got != head3 {
		t.Errorf("HEAD moved to %s", got)
	}

	if _, err := CheckTW(f.tw, "a/b"); err == nil {
		t.Error("CheckTW with a language directory that is not a name must be an error")
	}
}

// The check says what Run would say of the same commit, without a build.
// The text of the check says each case in words.
func TestCheckString(t *testing.T) {
	for _, tc := range []struct {
		c    Check
		want string
	}{
		{Check{}, "tw has no commit: no Release needed"},
		{Check{TWCommit: "abc", ReleaseNeeded: true}, "tw at abc: release needed"},
		{Check{TWCommit: "abc", Tag: testReleaseA, SameAs: testReleaseA}, "tw at abc: released as " + testReleaseA},
		{Check{TWCommit: "abc", SameAs: testReleaseA}, "tw at abc: release not needed, the pack is the one of " + testReleaseA},
	} {
		if got := tc.c.String(); got != tc.want {
			t.Errorf("%+v: String() = %q, want %q", tc.c, got, tc.want)
		}
	}
}

func TestCheckTWAgreesWithRun(t *testing.T) {
	f := newFixture(t)
	f.rebuild(t, testRules, "{\"a\": 1}\n")
	f.releaseWith(t)
	for _, records := range []string{"{\"a\": 1}\n", "{\"a\": 2}\n"} {
		f.rebuild(t, testRulesB, records)
		res, err := Run(f.root, f.tw, f.raw, f.out, testLang)
		if err != nil {
			t.Fatal(err)
		}
		c, err := CheckTW(f.tw, testLang)
		if err != nil {
			t.Fatal(err)
		}
		if c.TWCommit != res.TWCommit || c.ReleaseNeeded != res.ReleaseNeeded || c.SameAs != res.SameAs {
			t.Errorf("records %q: check = %+v, run = %+v", records, c, res)
		}
	}
}

// oldRelease commits the tw worktree with the pack sample, as the build of the
// raw snapshot rawTag (made here, with the same KR pack), and tags it tag, as
// a Release made earlier.
func (f fixture) oldRelease(t *testing.T, tag, rawTag, sample string) {
	t.Helper()
	if rawTag != testRawTag {
		runGit(t, f.raw, "tag", rawTag)
	}
	write(t, f.tw, ".meta/build.json", strings.Replace(testBuildJSON, testRawTag, rawTag, 1))
	write(t, f.tw, sampleRel, sample)
	runGit(t, f.tw, "add", "-A")
	runGit(t, f.tw, "commit", "-q", "--allow-empty", "-m", "build: "+rawTag)
	runGit(t, f.tw, "tag", tag)
}

// readHistory reads .meta/history/<tag>.json of the HEAD of tw; ok is false if
// there is none.
func (f fixture) readHistory(t *testing.T, tag string) (rec history.Record, ok bool) {
	t.Helper()
	out, err := exec.Command("git", "-C", f.tw, "show", "HEAD:.meta/history/"+tag+".json").Output()
	if err != nil {
		return rec, false
	}
	if err := json.Unmarshal(out, &rec); err != nil {
		t.Fatal(err)
	}
	return rec, true
}

func TestRunWritesTheHistoryOfTheReleaseItPrepares(t *testing.T) {
	f := newFixture(t)
	first := f.releaseWith(t)
	if len(first.History) != 0 || len(first.HistoryWarnings) != 0 {
		t.Errorf("the first Run = %+v, want no history: there is no Release to compare with", first)
	}
	if _, ok := f.readHistory(t, first.Tag); ok {
		t.Errorf("the first Release has a history record")
	}
	f.rebuild(t, testRulesB, "{\"a\": 1}\n")
	write(t, f.tw, "LimbusCompany_Data/Lang/zh-TW/Info/LICENSE-CONTENT", "licence\n")
	res, err := Run(f.root, f.tw, f.raw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	if !res.ReleaseNeeded || !slices.Equal(res.History, []string{res.Tag}) || len(res.HistoryWarnings) != 0 {
		t.Fatalf("Run = %+v, want a Release with the history of %s", res, res.Tag)
	}
	rec, ok := f.readHistory(t, res.Tag)
	if !ok || rec.PrevTag != first.Tag || rec.Tag != res.Tag || rec.RawTag != testRawTag {
		t.Errorf("record in the build commit = %+v, %v, want %s compared with %s", rec, ok, res.Tag, first.Tag)
	}
	if want := []string{"Lang/zh-TW/Info/LICENSE-CONTENT", "Lang/zh-TW/Sample.json"}; !slices.Equal(rec.OtherFiles, want) {
		t.Errorf("other files = %v, want %v", rec.OtherFiles, want)
	}
	if status := runGit(t, f.tw, "status", "--porcelain"); status != "" {
		t.Errorf("the worktree is not clean after Run: %q", status)
	}
}

func TestRunWritesNoHistoryWhenThePackIsTheSame(t *testing.T) {
	f := newFixture(t)
	f.releaseWith(t)
	f.rebuild(t, testRulesB, "{}\n") // only the rules commit differs
	res, err := Run(f.root, f.tw, f.raw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	if res.ReleaseNeeded || len(res.History) != 0 || len(res.HistoryWarnings) != 0 {
		t.Errorf("Run = %+v, want no Release and no history", res)
	}
	if _, ok := f.readHistory(t, res.Tag); ok {
		t.Errorf("a record of %s was written for a pack that did not change", res.Tag)
	}
}

func TestRunFillsInTheMissingHistories(t *testing.T) {
	f := newFixture(t)
	f.oldRelease(t, "zh-TW-20261101.1", "raw-20261101", "{\"v\": 1}\n")
	f.oldRelease(t, "zh-TW-20261102.1", "raw-20261102", "{\"v\": 2}\n")
	f.oldRelease(t, "zh-TW-20261103.1", "raw-20261103", "{\"v\": 3}\n")
	// The build of the worktree is that of the last Release.
	write(t, f.tw, ".meta/build.json", testBuildJSON)
	res, err := Run(f.root, f.tw, f.raw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"zh-TW-20261102.1", "zh-TW-20261103.1"}; !slices.Equal(res.History, want) || len(res.HistoryWarnings) != 0 {
		t.Fatalf("Run = %+v, want the histories %v", res, want)
	}
	for tag, prev := range map[string]string{"zh-TW-20261102.1": "zh-TW-20261101.1", "zh-TW-20261103.1": "zh-TW-20261102.1"} {
		if rec, ok := f.readHistory(t, tag); !ok || rec.PrevTag != prev {
			t.Errorf("record of %s = %+v, %v, want it compared with %s", tag, rec, ok, prev)
		}
	}
	if _, ok := f.readHistory(t, "zh-TW-20261101.1"); ok {
		t.Errorf("the oldest Release has a record: it has nothing before it")
	}
	head := runGit(t, f.tw, "rev-parse", "HEAD")
	again, err := Run(f.root, f.tw, f.raw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.History) != 0 || len(again.HistoryWarnings) != 0 || again.Changed || runGit(t, f.tw, "rev-parse", "HEAD") != head {
		t.Errorf("second Run = %+v, want nothing new in tw (at %s)", again, head)
	}
}

func TestRunRewritesTheHistoryOfATagNotYetReleased(t *testing.T) {
	f := newFixture(t)
	first := f.releaseWith(t)
	f.rebuild(t, testRulesB, "{\"a\": 1}\n")
	one, err := Run(f.root, f.tw, f.raw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	f.rebuild(t, testRulesB, "{\"a\": 2}\n")
	write(t, f.tw, "LimbusCompany_Data/Lang/zh-TW/Extra.json", "{}\n")
	two, err := Run(f.root, f.tw, f.raw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	if one.Tag != two.Tag || !slices.Equal(two.History, []string{two.Tag}) {
		t.Fatalf("runs = %+v, %+v, want the history of the same tag twice", one, two)
	}
	rec, ok := f.readHistory(t, two.Tag)
	if want := []string{"Lang/zh-TW/Extra.json", "Lang/zh-TW/Sample.json"}; !ok || rec.PrevTag != first.Tag || !slices.Equal(rec.OtherFiles, want) {
		t.Errorf("record = %+v, %v, want the other files %v", rec, ok, want)
	}
	// Back to the pack of the Release: the record of the tag goes.
	f.rebuild(t, testRulesB, "{}\n")
	if err := os.Remove(filepath.Join(f.tw, "LimbusCompany_Data/Lang/zh-TW/Extra.json")); err != nil {
		t.Fatal(err)
	}
	three, err := Run(f.root, f.tw, f.raw, f.out, testLang)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := f.readHistory(t, three.Tag); ok || len(three.History) != 0 {
		t.Errorf("Run = %+v: the record of an unchanged pack is still there (%v)", three, ok)
	}
}

func TestRunWarnsWhenAHistoryCannotBeMade(t *testing.T) {
	f := newFixture(t)
	f.oldRelease(t, "zh-TW-20261101.1", "raw-20261101", "{\"v\": 1}\n")
	f.oldRelease(t, "zh-TW-20261102.1", "raw-20261102", "{\"v\": 2}\n")
	f.oldRelease(t, "zh-TW-20261103.1", "raw-20261103", "{\"v\": 3}\n")
	write(t, f.tw, ".meta/build.json", testBuildJSON)
	runGit(t, f.root, "tag", "-d", "raw-20261102")
	res, err := Run(f.root, f.tw, f.raw, f.out, testLang)
	if err != nil {
		t.Fatalf("Run: %v, want the failed history to be a warning", err)
	}
	if res.TWCommit == "" || !res.Changed || res.ReleaseNeeded || res.SameAs != "zh-TW-20261103.1" {
		t.Errorf("Run = %+v, want tw committed, no Release, the pack of zh-TW-20261103.1", res)
	}
	if !slices.ContainsFunc(res.HistoryWarnings, func(w string) bool { return strings.Contains(w, "zh-TW-20261102.1") }) {
		t.Errorf("warnings = %q, want one that names zh-TW-20261102.1", res.HistoryWarnings)
	}
	if _, ok := f.readHistory(t, "zh-TW-20261102.1"); ok {
		t.Errorf("zh-TW-20261102.1 has a record though its snapshot is gone")
	}
}
