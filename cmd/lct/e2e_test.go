package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/batch"
	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/store"
)

// e2eFixture is the data set of the phase 1 dry run:
//
//	kr/ en/ jp/   the official packs as the CDN zips hold them, language
//	              prefix included (KR_Bufs.json, StoryData/KR_S001A.json); the
//	              test adds the LocalizeTemp_<lang>/ folder, a UTF-8 BOM and
//	              CRLF line ends when it zips them
//	upstream/     the three pinned upstream releases as "lct bootstrap
//	              --upstream" expects them: paratranz/ and ltm/ with their
//	              files at the root, llc/LimbusCompany_Data/Lang/LLC_zh-CN/
//
// The files are small on purpose. S001A.json is a story (place, model,
// teller, content; a Dante line wrapped in <…>, a <color> tag, a {0}
// placeholder), Skills_personality-01.json is a nested skill file with a
// [Laceration] keyword, and Bufs.json is where the keyword gets its name.
// What each Paratranz string is meant to be is written next to it in
// e2eWant.
const e2eFixture = "../../testdata/e2e"

// e2eFont is the real font, which lct.yaml pins by sha256.
const e2eFont = "assets/fonts/SarasaGothicTC-Bold.ttf"

// e2eWant is every string of the three output files. Copies stay KR, Paratranz
// strings are adopted (with the keyword and spacing repairs of bootstrap), and
// the four strings the fake translator handles carry what it makes of them.
var e2eWant = map[string]map[string]string{
	"StoryData/S001A.json": {
		"/dataList/0/place":   "南部12區外圍",
		"/dataList/0/content": "太陽正在下山。",
		"/dataList/1/model":   "이스마엘",
		"/dataList/1/teller":  "以實瑪利",
		"/dataList/1/content": "船長，今天要去哪裡？",
		"/dataList/2/model":   "돈키호테",
		"/dataList/2/content": "那當然是<color=#f8c200>冒險</color>囉！",
		"/dataList/3/model":   "단테",
		// Paratranz dropped the <…> of Dante's line: the repair keeps its
		// text as the draft and the fake translator wraps it again.
		"/dataList/3/content": "<但願今天也能平安無事。>",
		// Paratranz only has the EN text here: translated from scratch.
		"/dataList/4/content": "譯{0}",
		"/dataList/5/model":   "이스마엘",
		"/dataList/5/content": "好，走吧。",
	},
	"Skills_personality-01.json": {
		"/dataList/0/levelList/0/name": "斬擊",
		// Paratranz wrote 流血 where KR has [Laceration]: the ID is put back
		// and spaced.
		"/dataList/0/levelList/0/desc": "賦予{0}層 [Laceration] 強度",
		"/dataList/0/levelList/1/name": "突刺",
		// Paratranz dropped the <color> tags: draft plus the missing tags.
		"/dataList/0/levelList/1/desc": "攻擊時傷害增加<color=#f8c200>譯</color>譯",
		"/dataList/1/levelList/0/name": "防禦",
		"/dataList/1/levelList/0/desc": "譯{0} [Laceration]",
	},
	"Bufs.json": {
		"/dataList/0/id":   "Laceration",
		"/dataList/0/name": "流血",
	},
}

// lct runs the real entry point of the CLI, cli.Main, in the working directory
// and returns the exit code with what it printed.
func lct(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	dir := t.TempDir()
	outFile, err := os.Create(filepath.Join(dir, "stdout"))
	if err != nil {
		t.Fatal(err)
	}
	defer outFile.Close()
	errFile, err := os.Create(filepath.Join(dir, "stderr"))
	if err != nil {
		t.Fatal(err)
	}
	defer errFile.Close()

	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outFile, errFile
	func() {
		defer func() { os.Stdout, os.Stderr = oldOut, oldErr }()
		code = cli.Main(args)
	}()

	for _, f := range []struct {
		name string
		dst  *string
	}{{"stdout", &stdout}, {"stderr", &stderr}} {
		data, err := os.ReadFile(filepath.Join(dir, f.name))
		if err != nil {
			t.Fatal(err)
		}
		*f.dst = string(data)
	}
	return code, stdout, stderr
}

// mustLct runs lct with --json and fails the test unless it exits 0. It
// decodes what the command printed into into (when not nil).
func mustLct(t *testing.T, into any, args ...string) {
	t.Helper()
	cmd := "lct " + strings.Join(args, " ")
	code, stdout, stderr := lct(t, append([]string{"--json"}, args...)...)
	if code != 0 {
		t.Fatalf("%s: exit code %d, want 0\nstdout:\n%s\nstderr:\n%s", cmd, code, stdout, stderr)
	}
	if stderr != "" {
		t.Errorf("%s printed on stderr:\n%s", cmd, stderr)
	}
	if into == nil {
		return
	}
	if err := json.Unmarshal([]byte(stdout), into); err != nil {
		t.Fatalf("%s: output is not the expected JSON: %v\n%s", cmd, err, stdout)
	}
}

// e2eRepo makes the repo the dry run works in: a git repo on main holding what
// the real repo holds besides code, with the real font linked in rather than
// copied (it is 23 MB).
func e2eRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, f := range []string{"lct.yaml", "LICENSE-CONTENT", ".gitignore", "assets/fonts/OFL.txt"} {
		copyFile(t, filepath.Join(repoRoot, f), filepath.Join(root, f))
	}
	for _, d := range []string{"guides", "glossary", "rules"} {
		copyDir(t, filepath.Join(repoRoot, d), filepath.Join(root, d))
	}
	// The request, rewrite and adjustment files of the repo are made for its
	// tw and its KR; the dry run has a tw and a KR of its own, and its files.
	for _, d := range []string{"rules/requeue", "rules/rewrite", "rules/adjustments"} {
		files, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(d), "*.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			if err := os.Remove(f); err != nil {
				t.Fatal(err)
			}
		}
	}
	// The dry run checks the frozen files on BgmLyrics/L.json, whatever the
	// repo freezes now.
	cfgPath := filepath.Join(root, "lct.yaml")
	cfg, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	frozenLine := regexp.MustCompile(`(?m)^frozen: .*$`)
	if !frozenLine.Match(cfg) {
		t.Fatal("lct.yaml has no frozen line")
	}
	if err := os.WriteFile(cfgPath, frozenLine.ReplaceAll(cfg, []byte(`frozen: ["BgmLyrics/**"]`)), 0o644); err != nil {
		t.Fatal(err)
	}
	font, err := filepath.Abs(filepath.Join(repoRoot, e2eFont))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(font, filepath.Join(root, e2eFont)); err != nil {
		t.Fatal(err)
	}

	runGitIn(t, root, "init", "-q", "-b", "main")
	runGitIn(t, root, "config", "user.name", "Test")
	runGitIn(t, root, "config", "user.email", "test@example.com")
	runGitIn(t, root, "config", "commit.gpgsign", "false")
	runGitIn(t, root, "config", "tag.gpgsign", "false")
	runGitIn(t, root, "add", "-A")
	runGitIn(t, root, "commit", "-q", "-m", "rules")
	return root
}

// e2eCDN writes the three pack zips of fixture the way the CDN serves them
// for token and returns the file:// template that points at them. The packs
// get what the real ones have: a UTF-8 BOM and CRLF line ends.
func e2eCDN(t *testing.T, fixture, token string) (template string) {
	t.Helper()
	cdn := t.TempDir()
	dir := filepath.Join(cdn, token)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, lang := range packLangs {
		files := map[string]string{}
		base := filepath.Join(fixture, lang)
		err := filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, err := filepath.Rel(base, p)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			files[filepath.ToSlash(rel)] = "\xef\xbb\xbf" + strings.ReplaceAll(string(data), "\n", "\r\n")
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		writePackZip(t, dir, "localize_"+lang+".zip", lang, files)
	}
	return "file://" + cdn + "/{token}/localize_{lang}.zip"
}

// fakeTranslate stands in for the model. It makes the shortest text that
// passes validation: 譯, followed by every placeholder, every [ID] and every
// tag of the unit's keep list (tags in KR order, so that they nest), wrapped in
// <…> for a line of Dante. A unit with a draft starts from the draft and only
// gets what the draft lacks.
func fakeTranslate(u batch.BatchUnit) string {
	zh := "譯"
	if u.Draft != "" {
		zh = u.Draft
	}
	if u.Keep.DanteWrapper { // the wrapper goes back on at the end
		zh = strings.TrimSuffix(strings.TrimPrefix(zh, "<"), ">")
	}

	// add appends the items zh has fewer of than the keep list names.
	add := func(items []string, after string) {
		want := map[string]int{}
		for _, item := range items {
			want[item]++
			if strings.Count(zh, item) < want[item] {
				zh += item + after
			}
		}
	}
	add(u.Keep.Placeholders, "")
	add(u.Keep.IDs, "")
	add(u.Keep.Tags, "譯")

	if u.Keep.DanteWrapper {
		zh = "<" + zh + ">"
	}
	return zh
}

// pendingNow reads "lct pending --count".
func pendingNow(t *testing.T) pendingCount {
	t.Helper()
	var c pendingCount
	mustLct(t, &c, "pending", "--count")
	return c
}

func TestPhase1DryRun(t *testing.T) {
	fixture, err := filepath.Abs(e2eFixture)
	if err != nil {
		t.Fatal(err)
	}
	root := e2eRepo(t)
	template := e2eCDN(t, fixture, testToken)
	t.Chdir(root)
	t.Setenv("LCT_MODEL", "e2e") // not a real model name
	t.Setenv(sourceDateEpoch, "1793000000")

	// 1. fetch
	var fetched fetchResult
	mustLct(t, &fetched, "fetch", "--token", testToken, "--created-at", testCreatedAt,
		"--cdn-template", template, "--out", "work/fetch")
	if fetched.Token != testToken || fetched.CreatedAt != testCreatedAt {
		t.Errorf("fetch = %+v, want %s (%s)", fetched, testToken, testCreatedAt)
	}
	for _, f := range []string{"kr/KR_Bufs.json", "kr/StoryData/KR_S001A.json", "en/EN_Bufs.json", "jp/JP_Bufs.json"} {
		if _, err := os.Stat(filepath.Join(root, "work", "fetch", filepath.FromSlash(f))); err != nil {
			t.Errorf("fetch did not extract %s: %v", f, err)
		}
	}

	// 2. snapshot
	var snap snapshotOut
	mustLct(t, &snap, "snapshot", "--from", "work/fetch", "--token", testToken, "--created-at", testCreatedAt)
	if snap.Tag != "raw-20261029" || !snap.Changed {
		t.Errorf("snapshot = %+v, want a new raw-20261029", snap)
	}

	// 3. bootstrap
	var boot bootstrapOut
	mustLct(t, &boot, "bootstrap", "--upstream", filepath.Join(fixture, "upstream"))
	if boot.RawTag != "raw-20261029" {
		t.Errorf("bootstrap raw tag = %q, want raw-20261029", boot.RawTag)
	}
	for _, stat := range []string{"copy", "paratranz", "new", "paratranz-repair"} {
		if boot.Stats[stat] < 1 {
			t.Errorf("bootstrap stats[%s] = %d, want at least 1 (stats %v)", stat, boot.Stats[stat], boot.Stats)
		}
	}
	// 20 strings: 5 identifiers, 11 adopted, 2 to repair, 2 to translate;
	// LLC and LTM both align to the 4 queued units.
	wantStats := map[string]int{
		"copy": 5, "paratranz": 11, "paratranz-repair": 2, "new": 2, "unaligned-paratranz": 0, "frozen": 0,
		"refs-llc": 4, "refs-ltm": 4,
	}
	if len(boot.Stats) != len(wantStats) {
		t.Errorf("bootstrap stats = %v, want %v", boot.Stats, wantStats)
	}
	for stat, n := range wantStats {
		if boot.Stats[stat] != n {
			t.Errorf("bootstrap stats[%s] = %d, want %d (stats %v)", stat, boot.Stats[stat], n, boot.Stats)
		}
	}
	if boot.Records != 16 || boot.Pending != 4 {
		t.Errorf("bootstrap = %d records and %d pending, want 16 and 4", boot.Records, boot.Pending)
	}
	if count := pendingNow(t); count.Total != 4 || count.Available != 4 || count.ByReason["new"] != 2 || count.ByReason["paratranz-repair"] != 2 {
		t.Errorf("pending after bootstrap = %+v, want 4 available (2 new, 2 paratranz-repair)", count)
	}

	// 4. translate everything that is pending
	var batches []batch.Batch
	for round := 1; ; round++ {
		if round > 5 {
			t.Fatalf("batch next still hands out batches after %d rounds: %v", round-1, batchIDs(batches))
		}
		var next nextOut
		mustLct(t, &next, "batch", "next")
		if next.Batch == nil {
			break
		}
		data, err := os.ReadFile(next.Path)
		if err != nil {
			t.Fatal(err)
		}
		var b batch.Batch
		if err := json.Unmarshal(data, &b); err != nil {
			t.Fatalf("%s: %v", next.Path, err)
		}
		if b.ID != *next.Batch || len(b.Units) != next.Units {
			t.Fatalf("batch file = %s with %d units, batch next said %s with %d", b.ID, len(b.Units), *next.Batch, next.Units)
		}
		batches = append(batches, b)

		var out []batch.OutUnit
		for _, u := range b.Units {
			out = append(out, batch.OutUnit{Key: u.Key, ZH: fakeTranslate(u)})
		}
		writeOut(t, root, b.ID, out...)

		var sub submitOut
		mustLct(t, &sub, "batch", "submit", b.ID)
		if len(sub.Rejected) > 0 || len(sub.NeedsReview) > 0 || len(sub.Released) > 0 || len(sub.Accepted) != len(b.Units) {
			t.Fatalf("batch %s: submit = %+v, want all %d units accepted", b.ID, sub, len(b.Units))
		}
	}
	checkE2EBatches(t, batches)

	if count := pendingNow(t); count.Total != 0 {
		t.Errorf("pending after the batches = %+v, want none", count)
	}
	checkE2ERecords(t, root)

	// 5. build
	var built buildOut
	mustLct(t, &built, "build")
	completeZip := filepath.Join(root, "out", "zh-TW_complete_raw-20261029.zip")
	textZip := filepath.Join(root, "out", "zh-TW_text_raw-20261029.zip")
	if built.RawTag != "raw-20261029" || built.CompleteZip != completeZip || built.TextZip != textZip {
		t.Errorf("build = %+v, want zips %s and %s", built, completeZip, textZip)
	}
	for _, z := range []string{completeZip, textZip} {
		if _, err := os.Stat(z); err != nil {
			t.Errorf("build left no zip: %v", err)
		}
	}
	if r := built.Report; r == nil || r.Pending != 0 || r.StructuralFallback != 0 || len(r.FrozenFallback) != 0 || r.FieldRuleOverride != 0 ||
		r.NeedsReview != 0 || len(r.MissingGlyphs) != 0 || r.Warnings["hangul"] != 0 ||
		r.Origins["claude"] != 4 || r.Origins["paratranz"] != 11 || r.Origins["copy"] != 5 {
		t.Errorf("build report = %+v, want nothing pending, no fallback or override, no hangul warning (the copies keep KR), 4 claude, 11 paratranz and 5 copy records", built.Report)
	}
	checkE2EOutput(t, filepath.Join(root, ".work", "tw", "LimbusCompany_Data", "Lang"))

	names := zipNames(t, completeZip)
	for _, want := range []string{
		"LimbusCompany_Data/Lang/config.json",
		"LimbusCompany_Data/Lang/zh-TW/Info/version.json",
		"LimbusCompany_Data/Lang/zh-TW/StoryData/S001A.json",
		"LimbusCompany_Data/Lang/zh-TW/Font/Context/SarasaGothicTC-Bold.ttf",
	} {
		if !slices.Contains(names, want) {
			t.Errorf("complete zip lacks %s (has %v)", want, names)
		}
	}
	if textNames := zipNames(t, textZip); slices.Contains(textNames, "LimbusCompany_Data/Lang/zh-TW/Font/Context/SarasaGothicTC-Bold.ttf") {
		t.Errorf("text zip holds the font: %v", textNames)
	}

	// Nothing is pending after the build either, in either output format.
	code, stdout, stderr := lct(t, "pending", "--count")
	if code != 0 || strings.TrimSpace(stdout) != "0" {
		t.Errorf("lct pending --count = exit %d, %q (stderr %q), want 0 and a lone 0", code, stdout, stderr)
	}
	if count := pendingNow(t); count.Total != 0 {
		t.Errorf("pending after the build = %+v, want none", count)
	}
	var again nextOut
	mustLct(t, &again, "batch", "next")
	if again.Batch != nil {
		t.Errorf("batch next after the build handed out %s", *again.Batch)
	}

	// A second build of the same state gives the same bytes.
	textBefore, completeBefore := readFile(t, textZip), readFile(t, completeZip)
	mustLct(t, nil, "build")
	if !bytes.Equal(textBefore, readFile(t, textZip)) || !bytes.Equal(completeBefore, readFile(t, completeZip)) {
		t.Error("a second build changed the zips")
	}
}

func readFile(t *testing.T, p string) []byte {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func batchIDs(batches []batch.Batch) []string {
	ids := make([]string, len(batches))
	for i, b := range batches {
		ids[i] = b.ID
	}
	return ids
}

// checkE2EBatches checks what bootstrap handed on to the batches: the story
// comes first, with its translated lines around it, and both batches carry
// the drafts, references, keep lists and keyword names their units call for.
func checkE2EBatches(t *testing.T, batches []batch.Batch) {
	t.Helper()
	if len(batches) != 2 || batches[0].File != "StoryData/S001A.json" || batches[1].File != "Skills_personality-01.json" {
		t.Fatalf("batches = %v, want the story and then the skills", batchIDs(batches))
	}
	story, skills := batches[0], batches[1]

	if story.Family != "story-main" || skills.Family != "ally" {
		t.Errorf("families = %q and %q, want story-main and ally", story.Family, skills.Family)
	}
	if !slices.Contains(story.Guides, "guides/base.md") || !slices.Contains(story.Guides, "guides/characters/dante.md") {
		t.Errorf("story guides = %v, want the base guide and Dante's", story.Guides)
	}
	if len(story.Units) != 2 {
		t.Fatalf("story batch = %+v, want 2 units", story.Units)
	}
	dante, passenger := story.Units[0], story.Units[1]
	if dante.Key != "StoryData/S001A.json|/dataList/3/content" || dante.Reason != "paratranz-repair" ||
		dante.Draft != "但願今天也能平安無事。" || !dante.Keep.DanteWrapper || dante.Speaker != "dante" {
		t.Errorf("Dante's unit = %+v, want the repair with its draft, the wrapper and speaker dante", dante)
	}
	if passenger.Key != "StoryData/S001A.json|/dataList/4/content" || passenger.Reason != "new" || passenger.Draft != "" ||
		!slices.Equal(passenger.Keep.Placeholders, []string{"{0}"}) {
		t.Errorf("passenger unit = %+v, want a new unit with the {0} placeholder", passenger)
	}
	for _, u := range story.Units {
		if u.EN == nil || u.JP == nil || u.Refs["ltm"] == "" || u.Refs["llc"] == "" {
			t.Errorf("unit %s = EN %v, JP %v, refs %v, want EN, JP, ltm and llc", u.Key, u.EN, u.JP, u.Refs)
		}
	}
	// The adopted strings of the lines within two lines of the batch's two
	// units (a batch of two units looks at two lines on each side), but the
	// models, which are copies.
	var context []string
	for _, c := range story.Context {
		context = append(context, c.Key[strings.Index(c.Key, "|")+1:]+"="+c.ZH)
	}
	wantContext := []string{
		"/dataList/1/teller=以實瑪利",
		"/dataList/1/content=船長，今天要去哪裡？",
		"/dataList/2/content=那當然是<color=#f8c200>冒險</color>囉！",
		"/dataList/5/content=好，走吧。",
	}
	if !slices.Equal(context, wantContext) {
		t.Errorf("story context = %q, want %q", context, wantContext)
	}

	if len(skills.Units) != 2 {
		t.Fatalf("skills batch = %+v, want 2 units", skills.Units)
	}
	tags, keyword := skills.Units[0], skills.Units[1]
	if tags.Key != "Skills_personality-01.json|/dataList/0/levelList/1/desc" || tags.Reason != "paratranz-repair" || tags.Draft != "攻擊時傷害增加" ||
		!slices.Equal(tags.Keep.Tags, []string{"<color=#f8c200>", "</color>"}) {
		t.Errorf("tag unit = %+v, want the repair with its draft and the color tags", tags)
	}
	if keyword.Key != "Skills_personality-01.json|/dataList/1/levelList/0/desc" || keyword.Reason != "new" ||
		!slices.Equal(keyword.Keep.IDs, []string{"[Laceration]"}) || !slices.Equal(keyword.Keep.Placeholders, []string{"{0}"}) {
		t.Errorf("keyword unit = %+v, want a new unit with [Laceration] and {0}", keyword)
	}
	// The name of the keyword is the Paratranz name that was adopted.
	if skills.Keywords["Laceration"] != "流血" {
		t.Errorf("skills keywords = %v, want Laceration to be 流血", skills.Keywords)
	}
}

// checkE2ERecords checks the store after the translation: the four units went
// in as claude records with the model and the batch that made them, and every
// record names the commit of the rules, which are committed.
func checkE2ERecords(t *testing.T, root string) {
	t.Helper()
	records, err := store.Open(filepath.Join(root, ".work", "tw")).LoadRecords()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 20 {
		t.Errorf("the store holds %d records, want 20", len(records))
	}
	head := runGitIn(t, root, "rev-parse", "HEAD")
	claude := 0
	for key, rec := range records {
		if rec.Rules != head {
			t.Errorf("record %s: rules = %q, want HEAD %s", key, rec.Rules, head)
		}
		if rec.Origin != store.OriginClaude {
			continue
		}
		claude++
		if rec.Model != "e2e" || rec.Batch == "" || rec.Since != "raw-20261029" || len(rec.Flags) > 0 {
			t.Errorf("record %s = %+v, want model e2e, a batch, since raw-20261029 and no flags", key, rec)
		}
	}
	if claude != 4 {
		t.Errorf("%d records came from claude, want 4", claude)
	}
}

// checkE2EOutput checks the language pack in the tw worktree: every KR file is
// there as clean JSON (no BOM, LF only, a final newline) and holds exactly the
// strings of e2eWant.
func checkE2EOutput(t *testing.T, langRoot string) {
	t.Helper()
	cfg := readFile(t, filepath.Join(langRoot, "config.json"))
	if string(cfg) != "{\"lang\": \"zh-TW\"}\n" {
		t.Errorf("config.json = %q", cfg)
	}
	for file, want := range e2eWant {
		data := readFile(t, filepath.Join(langRoot, "zh-TW", filepath.FromSlash(file)))
		if bytes.HasPrefix(data, []byte("\xef\xbb\xbf")) || bytes.Contains(data, []byte("\r")) || !bytes.HasSuffix(data, []byte("}\n")) {
			t.Errorf("%s is not UTF-8 without BOM, with LF and a final newline:\n%q", file, data)
		}
		doc, err := jsonx.Parse(data)
		if err != nil {
			t.Errorf("%s does not parse: %v", file, err)
			continue
		}
		got := map[string]string{}
		for _, f := range doc.Strings() {
			got[f.Pointer] = f.Value
		}
		for pointer, text := range want {
			if got[pointer] != text {
				t.Errorf("%s %s = %q, want %q", file, pointer, got[pointer], text)
			}
		}
		if len(got) != len(want) {
			t.Errorf("%s has %d strings, want %d: %v", file, len(got), len(want), got)
		}
	}
}

// The warnings of the review page are those of the build: after lct build,
// lct site finds as many as the report says.
func TestSiteWarningsMatchBuild(t *testing.T) {
	fixture, err := filepath.Abs(e2eFixture)
	if err != nil {
		t.Fatal(err)
	}
	root := e2eRepo(t)
	template := e2eCDN(t, fixture, testToken)
	t.Chdir(root)
	t.Setenv("LCT_MODEL", "e2e")
	t.Setenv(sourceDateEpoch, "1793000000")

	mustLct(t, nil, "fetch", "--token", testToken, "--created-at", testCreatedAt, "--cdn-template", template, "--out", "work/fetch")
	mustLct(t, nil, "snapshot", "--from", "work/fetch", "--token", testToken, "--created-at", testCreatedAt)
	mustLct(t, nil, "bootstrap", "--upstream", filepath.Join(fixture, "upstream"))
	// The shared fixture is warning-free: this test gives some of its
	// records an ASCII ellipsis and kana, which the build warns about.
	st := store.Open(filepath.Join(root, ".work", "tw"))
	records, err := st.LoadRecords()
	if err != nil {
		t.Fatal(err)
	}
	edits := []string{"...", "か"}
	for _, key := range slices.Sorted(maps.Keys(records)) {
		rec := records[key]
		if len(edits) == 0 || rec.Origin != store.OriginParatranz || rec.ZH == "" {
			continue
		}
		rec.ZH += edits[0]
		records[key] = rec
		edits = edits[1:]
	}
	if len(edits) != 0 {
		t.Fatal("the fixture has too few paratranz records to break")
	}
	if err := st.SaveRecords(records); err != nil {
		t.Fatal(err)
	}
	var built buildOut
	mustLct(t, &built, "build")
	if built.Report == nil {
		t.Fatal("build has no report")
	}
	if len(built.Report.Warnings) < 2 {
		t.Fatalf("build warnings = %v, want at least two codes, or the test pins nothing", built.Report.Warnings)
	}
	mustLct(t, nil, "site", "--out", "work/site")

	var rep struct {
		Report struct {
			Warnings map[string]int `json:"warnings"`
		} `json:"report"`
	}
	if err := json.Unmarshal(readFile(t, filepath.Join(root, ".work", "tw", ".meta", "build.json")), &rep); err != nil {
		t.Fatal(err)
	}
	var wr struct {
		Items []struct {
			Code string   `json:"code"`
			Keys []string `json:"keys"`
		} `json:"items"`
	}
	if err := json.Unmarshal(readFile(t, filepath.Join(root, "work", "site", "review", "warnings.json")), &wr); err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for _, it := range wr.Items {
		got[it.Code] += len(it.Keys)
	}
	for code, n := range rep.Report.Warnings {
		if n == 0 {
			delete(rep.Report.Warnings, code)
		}
	}
	if !maps.Equal(got, rep.Report.Warnings) {
		t.Errorf("review/warnings.json counts %v, build.json report.warnings %v", got, rep.Report.Warnings)
	}
}
