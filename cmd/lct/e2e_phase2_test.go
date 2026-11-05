package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/adjust"
	"github.com/tool-jx3/limbus-translate/internal/batch"
	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/history"
	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/names"
	"github.com/tool-jx3/limbus-translate/internal/plan"
	"github.com/tool-jx3/limbus-translate/internal/publish"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/translate"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// phase2Fixture holds the two snapshots of the phase 2 replay, v1/ and v2/,
// each with kr/, en/ and jp/ as the CDN zips hold them (see e2eFixture). From
// v1 to v2:
//
//   - StoryData/S1.json gets a narrator line before its line 2, and line 2
//     (now 3) is reworded; every line after it moves down one;
//   - Bufs.json: the entry gets a desc;
//   - BgmLyrics/L.json, a frozen file: its second line is reworded;
//   - StoryData/S2.json is new, and its first line is a line of S1, with the
//     same speaker;
//   - StoryData/S3.json is gone;
//   - Passives.json: the second entry is renumbered (1002 becomes 1003).
const phase2Fixture = "../../testdata/phase2"

// The two game updates of the replay. The first is the update of phase 1.
const (
	phase2Token2     = "l20261105_Qm7Tz2Lr8_4pXvKd91Hc"
	phase2CreatedAt2 = "2026-11-05 03:01:47"
	phase2Raw1       = "raw-20261029"
	phase2Raw2       = "raw-20261105"
)

// fakeClaudeEnv makes the test binary the claude CLI of lct translate (see
// TestMain); its value is a directory the fake logs to.
const fakeClaudeEnv = "LCT_E2E_FAKE_CLAUDE"

// fakeClaudeResult is what the fake prints when it is done, as the claude
// CLI does with --output-format json.
const fakeClaudeResult = `{"total_cost_usd":0.01,"is_error":false,"subtype":"success","num_turns":3}`

// phase2MaxTurns is the --max-turns the pipeline gives lct translate, which
// passes it on to claude.
const phase2MaxTurns = "300"

// TestMain doubles as the claude CLI of TestPhase2Replay: when fakeClaudeEnv
// is set, the binary is run by lct translate in place of claude and does what
// the translate-pending skill does, instead of running the tests. It is also
// the gh CLI of TestRulesLoopReplay when fakeGHEnv is set (see fakeGH).
func TestMain(m *testing.M) {
	if state := os.Getenv(fakeGHEnv); state != "" {
		os.Exit(fakeGH(state, os.Args[1:]))
	}
	if dir := os.Getenv(fakeClaudeEnv); dir != "" {
		if err := fakeClaude(dir, os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, "fake claude:", err)
			os.Exit(1)
		}
		fmt.Println(fakeClaudeResult)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// fakeClaude is one run of the skill: it takes batch after batch with lct
// batch next, translates every unit with fakeTranslate and hands the batch in
// with lct batch submit, until no batch is handed out. It runs lct the way
// the tests do, through cli.Main, in the working directory lct translate gave
// it. It logs to dir: one line per run in runs (its model), and every batch it
// was handed in batches.jsonl.
func fakeClaude(dir string, args []string) error {
	model := os.Getenv("LCT_MODEL")
	want := []string{"-p", "/translate-pending", "--output-format", "stream-json", "--verbose", "--model", model, "--max-turns", phase2MaxTurns, "--disallowedTools", "Read(./.git/**)"}
	if model == "" || !slices.Equal(args, want) {
		return fmt.Errorf("run with %q and LCT_MODEL=%q, want %q", args, model, want)
	}
	if err := appendLine(filepath.Join(dir, "runs"), "model="+model); err != nil {
		return err
	}
	for round := 1; ; round++ {
		if round > 20 {
			return fmt.Errorf("batch next still hands out batches after %d rounds", round-1)
		}
		var next nextOut
		if err := runMain(&next, "batch", "next"); err != nil {
			return err
		}
		if next.Batch == nil {
			return nil
		}
		data, err := os.ReadFile(next.Path)
		if err != nil {
			return err
		}
		var b batch.Batch
		if err := json.Unmarshal(data, &b); err != nil {
			return fmt.Errorf("%s: %w", next.Path, err)
		}
		line, err := json.Marshal(b)
		if err != nil {
			return err
		}
		if err := appendLine(filepath.Join(dir, "batches.jsonl"), string(line)); err != nil {
			return err
		}
		out := make([]batch.OutUnit, 0, len(b.Units))
		for _, u := range b.Units {
			out = append(out, batch.OutUnit{Key: u.Key, ZH: fakeTranslate(u)})
		}
		data, err = json.Marshal(out)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(filepath.Dir(next.Path), b.ID+".out.json"), data, 0o644); err != nil {
			return err
		}
		var sub submitOut
		if err := runMain(&sub, "batch", "submit", b.ID); err != nil {
			return err
		}
		if len(sub.Rejected) > 0 || len(sub.NeedsReview) > 0 || len(sub.Released) > 0 || len(sub.Accepted) != len(b.Units) {
			return fmt.Errorf("batch %s: submit = %+v, want all %d units accepted", b.ID, sub, len(b.Units))
		}
	}
}

// runMain runs lct with --json through cli.Main, as lct does in a test, and
// decodes what it printed into into. Anything but exit code 0 is an error.
func runMain(into any, args ...string) error {
	tmp, err := os.MkdirTemp("", "lct-fake-claude-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	outFile, err := os.Create(filepath.Join(tmp, "stdout"))
	if err != nil {
		return err
	}
	defer outFile.Close()
	errFile, err := os.Create(filepath.Join(tmp, "stderr"))
	if err != nil {
		return err
	}
	defer errFile.Close()

	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outFile, errFile
	code := cli.Main(append([]string{"--json"}, args...))
	os.Stdout, os.Stderr = oldOut, oldErr

	stdout, err := os.ReadFile(outFile.Name())
	if err != nil {
		return err
	}
	stderr, err := os.ReadFile(errFile.Name())
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("lct %s: exit code %d\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), code, stdout, stderr)
	}
	if err := json.Unmarshal(stdout, into); err != nil {
		return fmt.Errorf("lct %s: %w\n%s", strings.Join(args, " "), err, stdout)
	}
	return nil
}

func appendLine(name, line string) error {
	f, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(f, line); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// phase2 is the replay as it goes: the repo, the fake claude and its log.
type phase2 struct {
	t       *testing.T
	root    string
	fixture string
	claude  string // the test binary
	fakeDir string
}

func (p *phase2) store() *store.Store { return store.Open(filepath.Join(p.root, ".work", "tw")) }

// update fetches the packs of version (v1 or v2) for token from a fake CDN
// and snapshots them on raw, as the weekly update does, and checks the tag.
func (p *phase2) update(version, token, createdAt, wantTag string) {
	p.t.Helper()
	template := e2eCDN(p.t, filepath.Join(p.fixture, version), token)
	out := filepath.Join("work", "fetch-"+version)
	mustLct(p.t, nil, "fetch", "--token", token, "--created-at", createdAt, "--cdn-template", template, "--out", out)
	var snap snapshotOut
	mustLct(p.t, &snap, "snapshot", "--from", out, "--token", token, "--created-at", createdAt)
	if snap.Tag != wantTag || !snap.Changed {
		p.t.Fatalf("snapshot of %s = %+v, want a new %s", version, snap, wantTag)
	}
}

func (p *phase2) plan() plan.Summary {
	p.t.Helper()
	var sum plan.Summary
	mustLct(p.t, &sum, "plan")
	return sum
}

// translate runs lct translate with the flags of the pipeline and the fake
// claude, and checks that it translated everything in one run.
func (p *phase2) translate(wantBefore int) translate.Result {
	p.t.Helper()
	code, stdout, stderr := lct(p.t, "--json", "translate", "--claude", p.claude,
		"--max-usd", "30", "--deadline", "5h", "--max-turns", phase2MaxTurns, "--run-timeout", "90m")
	if code != 0 {
		p.t.Fatalf("lct translate: exit code %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	var res translate.Result
	if err := json.Unmarshal([]byte(stdout), &res); err != nil {
		p.t.Fatalf("lct translate printed %q: %v", stdout, err)
	}
	if res.Model != "e2e" || res.Runs != 1 || res.CostUSD != 0.01 || res.Stopped != translate.StopDone ||
		res.PendingBefore != wantBefore || res.PendingAfter != 0 || res.Error != "" {
		p.t.Errorf("translate = %+v, want one run of e2e costing $0.01 that leaves none of the %d units", res, wantBefore)
	}
	if c := pendingNow(p.t); c.Total != 0 {
		p.t.Errorf("pending after translate = %+v, want none", c)
	}
	// Every batch the run took and handed in is in the log, as the batch
	// commands logged it, and the round adds them up.
	handedOut := regexp.MustCompile(`(?m)^round 1: batch \S+: \d+ units? of \S+( and \d+ more files?)?$`).FindAllString(stderr, -1)
	submitted := regexp.MustCompile(`(?m)^round 1: batch \S+ submitted: \d+ accepted, `).FindAllString(stderr, -1)
	if len(handedOut) == 0 || len(submitted) != len(handedOut) {
		p.t.Errorf("lct translate logged %d batches handed out and %d submitted, want as many, and some:\n%s", len(handedOut), len(submitted), stderr)
	}
	sum := regexp.MustCompile(`(?m)^round 1: (\d+) batch(es)?, \d+ units?; (\d+) submitted: `).FindStringSubmatch(stderr)
	if sum == nil || sum[1] != strconv.Itoa(len(handedOut)) || sum[3] != strconv.Itoa(len(submitted)) {
		p.t.Errorf("the round 1 summary is %q, want %d batches handed out and submitted:\n%s", sum, len(handedOut), stderr)
	}
	if !strings.Contains(stderr, "translate: 1 round, ") {
		p.t.Errorf("lct translate does not sum the batches of its round up:\n%s", stderr)
	}
	return res
}

func (p *phase2) build() buildOut {
	p.t.Helper()
	var b buildOut
	mustLct(p.t, &b, "build")
	return b
}

// publish prepares the Release and, when one is needed, makes it as the
// workflow does with gh release create: the tag goes on the tw commit.
func (p *phase2) publish() publish.Result {
	p.t.Helper()
	var res publish.Result
	mustLct(p.t, &res, "publish")
	if res.ReleaseNeeded {
		runGitIn(p.t, p.root, "tag", res.Tag, res.TWCommit)
	}
	return res
}

// batches reads the batches the fake claude was handed, in order.
func (p *phase2) batches() []batch.Batch {
	p.t.Helper()
	f, err := os.Open(filepath.Join(p.fakeDir, "batches.jsonl"))
	if err != nil {
		p.t.Fatal(err)
	}
	defer f.Close()
	var out []batch.Batch
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 1<<24)
	for sc.Scan() {
		var b batch.Batch
		if err := json.Unmarshal(sc.Bytes(), &b); err != nil {
			p.t.Fatal(err)
		}
		out = append(out, b)
	}
	if err := sc.Err(); err != nil {
		p.t.Fatal(err)
	}
	return out
}

// handedOut is every unit of batches, by key.
func handedOut(batches []batch.Batch) map[string]batch.BatchUnit {
	out := map[string]batch.BatchUnit{}
	for _, b := range batches {
		for _, u := range b.Units {
			out[u.Key] = u
		}
	}
	return out
}

func (p *phase2) records() map[string]store.Record {
	p.t.Helper()
	recs, err := p.store().LoadRecords()
	if err != nil {
		p.t.Fatal(err)
	}
	return recs
}

func (p *phase2) pending() map[string]store.PendingItem {
	p.t.Helper()
	items, err := p.store().LoadPending()
	if err != nil {
		p.t.Fatal(err)
	}
	out := map[string]store.PendingItem{}
	for _, it := range items {
		out[it.Key] = it
	}
	return out
}

// state is every file of the translation state in the tw worktree but the
// plan summary: the records, the queue, applied.json and the rest of .meta.
func (p *phase2) state() map[string]string {
	p.t.Helper()
	meta := filepath.Join(p.root, ".work", "tw", ".meta")
	files := map[string]string{}
	err := filepath.WalkDir(meta, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(meta, path)
		if err != nil {
			return err
		}
		if rel == plan.File || rel == ".lock" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		p.t.Fatal(err)
	}
	return files
}

// packStrings reads the strings of file in the language pack of the tw
// worktree, by pointer.
func (p *phase2) packStrings(file string) map[string]string {
	p.t.Helper()
	data := readFile(p.t, filepath.Join(p.root, ".work", "tw", "LimbusCompany_Data", "Lang", "zh-TW", filepath.FromSlash(file)))
	doc, err := jsonx.Parse(data)
	if err != nil {
		p.t.Fatalf("%s: %v", file, err)
	}
	got := map[string]string{}
	for _, f := range doc.Strings() {
		got[f.Pointer] = f.Value
	}
	return got
}

// The files and units the replay looks at.
const (
	s1     = "StoryData/S1.json"
	s2     = "StoryData/S2.json"
	s3     = "StoryData/S3.json"
	bufs   = "Bufs.json"
	pas    = "Passives.json"
	lyrics = "BgmLyrics/L.json"
)

func p2Key(file, pointer string) string { return units.Key(file, pointer) }

func p2Content(file string, i int) string { return p2Key(file, fmt.Sprintf("/dataList/%d/content", i)) }

// The texts of the human adjustments made between the two snapshots: one of
// a line v2 keeps (it moves from line 3 to 4), one of the line v2 rewords.
const (
	adjKeptZH    = "早知道就帶傘來了。"
	adjChangedZH = "雨天的冒險也不壞！"
	requeueHint  = "狀態名稱改用新譯名"
	toneHint     = "語氣再口語一點"
)

func TestPhase2Replay(t *testing.T) {
	fixture, err := filepath.Abs(phase2Fixture)
	if err != nil {
		t.Fatal(err)
	}
	claude, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	// The replay holds whatever the glossary holds: a person approving an
	// ordinary term the fixture's texts have (피해, in KR_Passives.json and
	// the like) does not change what it checks.
	p := &phase2{t: t, root: e2eRepo(t, "- kr: 피해\n  zh: 傷害\n  status: approved\n"), fixture: fixture, claude: claude, fakeDir: t.TempDir()}
	t.Chdir(p.root)
	t.Setenv("LCT_MODEL", "e2e") // not a real model name
	// As in CI: the build time is the time of the commit of the rules.
	t.Setenv(sourceDateEpoch, "")
	t.Setenv(fakeClaudeEnv, p.fakeDir)

	// 1. The first update: snapshot, plan, translate, build, publish.
	p.update("v1", testToken, testCreatedAt, phase2Raw1)
	// The repo has the rules/fields.yaml learned from the real packs, which
	// the small ones do not bear out. A person keeps it in step with the packs,
	// as lct fields learn does, so that no plan reports a change of the field
	// rules.
	var learned learnReport
	mustLct(t, &learned, "fields", "learn", "--write")
	if !learned.Written {
		t.Fatalf("fields learn = %+v, want rules/fields.yaml written", learned)
	}
	runGitIn(t, p.root, "commit", "-q", "-am", "rules: learn the field rules of the packs")
	plan1 := p.plan()
	checkPhase2FirstPlan(t, plan1, runGitIn(t, p.root, "rev-parse", "HEAD"))
	p.translate(14)
	checkPhase2BatchFiles(t, p.root, p.batches())
	recs1 := p.records()
	if len(recs1) != 21 {
		t.Errorf("after the first translation the store holds %d records, want 21", len(recs1))
	}
	for key, rec := range recs1 {
		if rec.Origin != store.OriginCopy && (rec.Origin != store.OriginClaude || rec.ZH != "譯" || rec.Model != "e2e" || rec.Since != phase2Raw1) {
			t.Errorf("record %s = %+v, want a copy or the fake's 譯 by e2e for %s", key, rec, phase2Raw1)
		}
	}
	built1 := p.build()
	if r := built1.Report; r == nil || r.Pending != 0 || r.Origins["claude"] != 14 || r.Origins["copy"] != 7 {
		t.Errorf("first build report = %+v, want nothing pending, 14 claude and 7 copy records", built1.Report)
	}
	// Nothing needs a person: no review issue.
	if _, err := os.Stat(filepath.Join(p.root, "out", "review-issue")); built1.ReviewIssueParts != 0 || err == nil {
		t.Errorf("first build: %d review issue parts (out/review-issue: %v), want none", built1.ReviewIssueParts, err)
	}
	pub1 := p.publish()
	if !pub1.Changed || !pub1.ReleaseNeeded || pub1.Tag != "zh-TW-20261029.1" {
		t.Errorf("first publish = %+v, want a new tw commit to release as zh-TW-20261029.1", pub1)
	}

	// 2. Before the next update a person asks for two lines of S1 to be
	// translated again: the reworded line to come (line 2) and the last line,
	// which the insertion will move. rebuild.yml runs: the plan queues them,
	// and the translation run takes them in a batch and is killed before it
	// hands the batch in, so they stay leased; build and publish follow.
	var written requeueWritten
	mustLct(t, &written, "requeue", "--key", p2Content(s1, 1), "--key", p2Content(s1, 4), "--reason", toneHint, "--to-file", "s1-tone", "--date", "2026-11-02")
	runGitIn(t, p.root, "add", "-A")
	runGitIn(t, p.root, "commit", "-q", "-m", "rules: requeue two lines of S1")
	rebuild := p.plan()
	if rebuild.RawTag != phase2Raw1 || rebuild.PrevRawTag != phase2Raw1 || !reflect.DeepEqual(rebuild.RequeueApplied, []plan.Applied{{ID: "2026-11-02-s1-tone", Units: 2}}) || rebuild.Pending != 2 {
		t.Errorf("rebuild plan = %+v, want the request of %s applied to 2 units", rebuild, phase2Raw1)
	}
	var killed nextOut
	mustLct(t, &killed, "batch", "next")
	if killed.Batch == nil || killed.Units != 2 {
		t.Fatalf("batch next = %+v, want the two requeued lines", killed)
	}
	for _, key := range []string{p2Content(s1, 1), p2Content(s1, 4)} {
		want := store.PendingItem{Key: key, Reason: "requeue", PrevZH: "譯", Hint: toneHint, Lease: &store.Lease{Batch: *killed.Batch}}
		got := p.pending()[key]
		if got.Lease != nil {
			want.Lease.Until = got.Lease.Until
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s before the update = %+v, want %+v", key, got, want)
		}
	}
	p.build()
	// Only the queue and the rules commit changed: the pack is the one of the
	// first Release, so tw is committed and nothing is released.
	if pub := p.publish(); !pub.Changed || pub.ReleaseNeeded || pub.SameAs != "zh-TW-20261029.1" {
		t.Errorf("rebuild publish = %+v, want the new queue committed to tw and no Release, as the pack is the one of zh-TW-20261029.1", pub)
	}

	// 3. A person works on main: two adjustments of S1, a request to
	// translate Bufs.json again and a rewrite of the short lines of S1.
	var added adjustAdded
	mustLct(t, &added, "adjust", "add", "--key", p2Content(s1, 2), "--zh", adjKeptZH,
		"--reason", "以實瑪利的口吻", "--rule", "one-off", "--topic", "umbrella", "--date", "2026-11-03")
	mustLct(t, &added, "adjust", "add", "--key", p2Content(s1, 1), "--zh", adjChangedZH,
		"--reason", "唐吉訶德的口吻", "--rule", "one-off", "--topic", "rain", "--date", "2026-11-03")
	mustLct(t, &written, "requeue", "--file", bufs, "--reason", requeueHint, "--to-file", "bufs", "--date", "2026-11-03")
	if written.ID != "2026-11-03-bufs" {
		t.Errorf("requeue --to-file = %+v, want the id 2026-11-03-bufs", written)
	}
	writeTestFile(t, p.root, "rules/rewrite/2026-11-03-s1-lines.yaml",
		"reason: S1 的短譯文補成「譯文」\nfiles: [\"StoryData/S1.json\"]\npattern: '^譯$'\nreplace: '譯文'\n")
	mustLct(t, nil, "rules", "lint")
	runGitIn(t, p.root, "add", "-A")
	runGitIn(t, p.root, "commit", "-q", "-m", "rules: adjust S1, requeue Bufs, rewrite S1")
	rules2 := runGitIn(t, p.root, "rev-parse", "HEAD")

	// 4. The second update.
	p.update("v2", phase2Token2, phase2CreatedAt2, phase2Raw2)
	plan2 := p.plan()
	checkPhase2SecondPlan(t, plan2, rules2)
	checkPhase2SecondState(t, p, recs1, rules2)
	checkPhase2StaleBatch(t, p, *killed.Batch)

	// 5. The same plan again (a rebuild, a rerun) changes nothing.
	before := p.state()
	again := p.plan()
	if after := p.state(); !reflect.DeepEqual(after, before) {
		t.Errorf("a second plan of %s changed the state:\nbefore %v\nafter  %v", phase2Raw2, before, after)
	}
	if again.PrevRawTag != phase2Raw1 || !reflect.DeepEqual(again.Changes, plan2.Changes) ||
		!slices.Equal(again.FrozenChanged, plan2.FrozenChanged) || !slices.Equal(again.NewSpeakers, plan2.NewSpeakers) ||
		!slices.Equal(again.NewFilesDefaultFamily, plan2.NewFilesDefaultFamily) || !reflect.DeepEqual(again.Adjustments, plan2.Adjustments) {
		t.Errorf("second plan = %+v, want what the snapshot brought and the adjustments as in %+v", again, plan2)
	}
	if len(again.RequeueApplied)+len(again.RewriteApplied) != 0 || again.Reused != 0 || again.Copy != 0 || again.Pending != plan2.Pending {
		t.Errorf("second plan = %+v, want no file applied again, nothing reused or copied, %d pending", again, plan2.Pending)
	}
	var onDisk plan.Summary
	if _, err := p.store().ReadMetaJSON(plan.File, &onDisk); err != nil || !reflect.DeepEqual(onDisk, again) {
		t.Errorf(".meta/plan.json = %+v (%v), want what the second plan printed", onDisk, err)
	}

	// 6. Translate and build.
	firstRun := len(p.batches())
	p.translate(6)
	checkPhase2BatchFiles(t, p.root, p.batches()[firstRun:])
	checkPhase2Batches(t, p.batches()[firstRun:])
	built2 := p.build()
	checkPhase2SecondBuild(t, p, built2)

	// 7. Publish: a Release; then a build of nothing new releases nothing.
	pub2 := p.publish()
	if !pub2.Changed || !pub2.ReleaseNeeded || pub2.Tag != "zh-TW-20261105.1" {
		t.Errorf("second publish = %+v, want a new tw commit to release as zh-TW-20261105.1", pub2)
	}
	checkPhase2History(t, p, pub2, plan2.Changes)
	p.build()
	pub3 := p.publish()
	if pub3.Changed || pub3.ReleaseNeeded || pub3.TWCommit != pub2.TWCommit {
		t.Errorf("publish of a build that changed nothing = %+v, want no commit and no Release (tw stays at %s)", pub3, pub2.TWCommit)
	}
	if runs := strings.Fields(string(readFile(t, filepath.Join(p.fakeDir, "runs")))); !slices.Equal(runs, []string{"model=e2e", "model=e2e"}) {
		t.Errorf("the fake claude ran %q, want twice with LCT_MODEL e2e", runs)
	}

	// 8. The person tidies the adjustment files with what the second plan
	// found: the adjustment that moved gets its new key, and the one that
	// expired goes, as its unit is translated again. The files lint clean, and
	// a plan after it changes nothing in the state but the adjustments it
	// reports: the moved one is at its own key now.
	var tidied tidyOut
	mustLct(t, &tidied, "adjust", "tidy")
	if want := []string{"rules/adjustments/2026-11-03-rain.yaml", "rules/adjustments/2026-11-03-umbrella.yaml"}; !slices.Equal(tidied.Changed, want) || tidied.Moved != 1 || tidied.Removed != 1 {
		t.Errorf("adjust tidy = %+v, want %v changed, 1 moved and 1 removed", tidied, want)
	}
	if left, err := adjust.Load(p.root); err != nil || len(left) != 1 || left[0].Key != p2Content(s1, 3) || left[0].ZH != adjKeptZH {
		t.Errorf("the adjustments left = %+v, %v, want the moved one at %s", left, err, p2Content(s1, 3))
	}
	mustLct(t, nil, "rules", "lint")
	beforeTidyPlan := p.state()
	if replan := p.plan(); replan.Adjustments.Applied != 1 || len(replan.Adjustments.Moved) != 0 || len(replan.Adjustments.Expired) != 0 {
		t.Errorf("plan after adjust tidy: Adjustments = %+v, want the one adjustment applied at its own key", replan.Adjustments)
	}
	if after := p.state(); !reflect.DeepEqual(after, beforeTidyPlan) {
		t.Errorf("a plan after adjust tidy changed the state:\nbefore %v\nafter  %v", beforeTidyPlan, after)
	}
}

// checkPhase2History checks the history record of the second Release in the
// HEAD of tw: compared with the first Release, and with the numbers of new,
// changed and deleted strings that the plan of the second snapshot found (the
// plan counts every string, the record the copies apart).
func checkPhase2History(t *testing.T, p *phase2, pub publish.Result, want plan.Changes) {
	t.Helper()
	if !slices.Equal(pub.History, []string{pub.Tag}) || len(pub.HistoryWarnings) != 0 {
		t.Errorf("second publish: history %v, warnings %q, want only the record of %s", pub.History, pub.HistoryWarnings, pub.Tag)
	}
	tw := filepath.Join(p.root, ".work", "tw")
	var rec history.Record
	data := runGitIn(t, tw, "show", pub.TWCommit+":.meta/history/"+pub.Tag+".json")
	if err := json.Unmarshal([]byte(data), &rec); err != nil {
		t.Fatalf(".meta/history/%s.json in tw: %v\n%s", pub.Tag, err, data)
	}
	if rec.Tag != pub.Tag || rec.PrevTag != "zh-TW-20261029.1" || rec.RawTag != phase2Raw2 || rec.PrevRawTag != phase2Raw1 {
		t.Errorf("history record = %+v, want %s after zh-TW-20261029.1, from %s after %s", rec, pub.Tag, phase2Raw2, phase2Raw1)
	}
	got := plan.Changes{
		New:     rec.Counts.New + rec.CopyCounts.New,
		Changed: rec.Counts.KRChanged + rec.CopyCounts.KRChanged,
		Deleted: rec.Counts.Deleted + rec.CopyCounts.Deleted,
	}
	if got.New != want.New || got.Changed != want.Changed || got.Deleted != want.Deleted {
		t.Errorf("history counts %+v and copies %+v add up to %+v, want the new %d, changed %d, deleted %d of the plan", rec.Counts, rec.CopyCounts, got, want.New, want.Changed, want.Deleted)
	}
	if len(rec.Entries) == 0 {
		t.Errorf("history record has no entry")
	}
}

// checkPhase2FirstPlan checks the plan of the first snapshot: there is no
// earlier one, so every string is copied (identifiers) or queued as new.
func checkPhase2FirstPlan(t *testing.T, sum plan.Summary, rules string) {
	t.Helper()
	want := plan.Summary{
		RawTag: phase2Raw1, PrevRawTag: phase2Raw1, RulesCommit: rules,
		// 4 models of S1, the id of Bufs, the 2 ids of the lyrics.
		Copy:           7,
		Adjustments:    plan.AdjustmentSummary{Moved: []plan.MovedKey{}, Expired: []string{}},
		RequeueApplied: []plan.Applied{}, RewriteApplied: []plan.Applied{},
		FrozenChanged: []string{}, NewSpeakers: []string{}, FieldChanges: []fields.Change{}, NewFilesDefaultFamily: []string{},
		// S1: the place and 5 lines; S3: 1 line; Bufs: the name; Passives: 2
		// names and 2 descs; the lyrics: 2 lines.
		Pending: 14,
	}
	if !reflect.DeepEqual(sum, want) {
		t.Errorf("first plan = %+v\nwant %+v", sum, want)
	}
}

// checkPhase2SecondPlan checks what the plan of the second snapshot says.
func checkPhase2SecondPlan(t *testing.T, sum plan.Summary, rules string) {
	t.Helper()
	want := plan.Summary{
		RawTag: phase2Raw2, PrevRawTag: phase2Raw1, RulesCommit: rules,
		Changes: plan.Changes{
			// S1: 9 of its 11 strings, 7 of them one line down; Bufs: id and
			// name; Passives: the entry that kept its id; the lyrics: both ids
			// and the first line.
			Unchanged: 16, Moved: 7,
			// The reworded line of S1 and of the lyrics.
			Changed: 2,
			// The narrator line of S1, the 4 strings of S2, the desc of Bufs,
			// the renumbered entry of Passives.
			New: 8,
			// S3, and the entry of Passives under its old id.
			Deleted:      3,
			FilesAdded:   []string{s2},
			FilesDeleted: []string{s3},
			FilesChanged: []string{lyrics, bufs, pas, s1},
			FilesShifted: []string{s1},
		},
		// The models of S2.
		Copy: 2,
		// The line of S2 that S1 has, and the renumbered entry of Passives.
		Reused: 3,
		Adjustments: plan.AdjustmentSummary{
			Moved:   []plan.MovedKey{{From: p2Content(s1, 2), To: p2Content(s1, 3)}},
			Expired: []string{p2Content(s1, 1)},
			// The reworded line moved one line down, to the key of the
			// adjustment that moved on.
			ExpiredUnits: map[string]string{p2Content(s1, 1): p2Content(s1, 2)},
		},
		RequeueApplied: []plan.Applied{{ID: "2026-11-03-bufs", Units: 1}},
		// The 4 claude lines of S1 that kept their KR, but the adjusted one.
		RewriteApplied:        []plan.Applied{{ID: "2026-11-03-s1-lines", Units: 4}},
		FrozenChanged:         []string{p2Content(lyrics, 1)},
		NewSpeakers:           []string{},
		FieldChanges:          []fields.Change{},
		NewFilesDefaultFamily: []string{},
		// The narrator line, the reworded line and the requeued last line of
		// S1, the second line of S2, the name (requeued) and the desc of Bufs.
		Pending: 6,
	}
	if !reflect.DeepEqual(sum, want) {
		t.Errorf("second plan = %+v\nwant %+v", sum, want)
	}
}

// checkPhase2SecondState checks the records and the queue after the plan of
// the second snapshot against recs1, the records of the first one.
func checkPhase2SecondState(t *testing.T, p *phase2, recs1 map[string]store.Record, rules string) {
	t.Helper()
	recs, pending := p.records(), p.pending()

	// Every line of S1 whose KR did not change kept its record, at its new
	// key; the rewrite changed the text of those that are not adjusted.
	moved := func(from, to string, zh string) store.Record {
		rec, ok := recs1[from]
		if !ok {
			t.Fatalf("the first translation left no record of %s", from)
		}
		rec.Key = to
		if zh != "" {
			rec.ZH = zh
		}
		return rec
	}
	model := func(i int) string { return p2Key(s1, fmt.Sprintf("/dataList/%d/model", i)) }
	wantS1 := map[string]store.Record{
		p2Key(s1, "/dataList/0/place"): moved(p2Key(s1, "/dataList/0/place"), p2Key(s1, "/dataList/0/place"), "譯文"),
		p2Content(s1, 0):               moved(p2Content(s1, 0), p2Content(s1, 0), "譯文"),
		model(2):                       moved(model(1), model(2), ""),
		model(3):                       moved(model(2), model(3), ""),
		p2Content(s1, 3): {
			Key: p2Content(s1, 3), KRSha: units.KRSha("우산을 챙겨 올 걸 그랬어."), ZH: adjKeptZH, Origin: store.OriginHuman,
			Since: phase2Raw2, Rules: rules, Batch: "adjustment:2026-11-03-umbrella.yaml",
		},
		model(4):         moved(model(3), model(4), ""),
		p2Content(s1, 4): moved(p2Content(s1, 3), p2Content(s1, 4), "譯文"),
		model(5):         moved(model(4), model(5), ""),
		p2Content(s1, 5): moved(p2Content(s1, 4), p2Content(s1, 5), "譯文"),
	}
	gotS1 := map[string]store.Record{}
	for key, rec := range recs {
		if strings.HasPrefix(key, s1+"|") {
			gotS1[key] = rec
		}
	}
	if !reflect.DeepEqual(gotS1, wantS1) {
		t.Errorf("records of S1 = %v\nwant %v", gotS1, wantS1)
	}
	for key := range wantS1 {
		if it, ok := pending[key]; ok && key != p2Content(s1, 5) { // requeued before the update
			t.Errorf("%s kept its KR but is queued: %+v", key, it)
		}
	}

	// S2 took the translation of the same line of S1, rewritten.
	reused := recs[p2Content(s2, 0)]
	if reused.ZH != "譯文" || reused.Origin != store.OriginClaude || reused.Model != "e2e" || reused.Since != phase2Raw2 ||
		reused.Rules != rules || reused.Batch != "reuse:"+p2Content(s1, 4) {
		t.Errorf("S2 line 1 = %+v, want the rewritten translation of S1 line 5 reused", reused)
	}
	// The renumbered entry of Passives took the translations of its old id.
	for _, field := range []string{"name", "desc"} {
		key := p2Key(pas, "/dataList/1/"+field)
		if rec := recs[key]; rec.ZH != "譯" || rec.Origin != store.OriginClaude || rec.Batch != "reuse:"+key {
			t.Errorf("%s = %+v, want the translation of the entry under its old id", key, rec)
		}
	}
	// The frozen line kept its translation and the sha of its old KR.
	if got, want := recs[p2Content(lyrics, 1)], recs1[p2Content(lyrics, 1)]; !reflect.DeepEqual(got, want) {
		t.Errorf("frozen line = %+v, want it kept as %+v", got, want)
	}
	for key := range recs {
		if strings.HasPrefix(key, s3+"|") {
			t.Errorf("record %s of the deleted S3 is still there", key)
		}
	}

	// The queue: the reworded line of S1, which was pending already, with
	// the KR and the translation of the last version paired, the hint of its
	// request and the adjustment that expired; the inserted line as new; the
	// requeued last line of S1 at its new key; the requeued name of Bufs with
	// the hint, and the new strings without a source. The transition cleared
	// the leases of the killed batch.
	want := map[string]store.PendingItem{
		bufs + "|/dataList/0/desc": {Key: bufs + "|/dataList/0/desc", Reason: "new"},
		bufs + "|/dataList/0/name": {Key: bufs + "|/dataList/0/name", Reason: "requeue", PrevZH: "譯", Hint: requeueHint},
		p2Content(s1, 1):           {Key: p2Content(s1, 1), Reason: "new"},
		p2Content(s1, 2): {
			Key: p2Content(s1, 2), Reason: plan.ReasonAdjustmentExpired, PrevKR: "비 오는 날의 모험도 나쁘지 않소!", PrevZH: "譯",
			Hint: toneHint, Refs: map[string]string{plan.RefAdjustment: adjChangedZH},
		},
		// It was queued with the text of its record, which the rewrite
		// changed: the item shows the rewritten text, so the translation made
		// from it keeps the rewrite.
		p2Content(s1, 5): {Key: p2Content(s1, 5), Reason: "requeue", PrevZH: "譯文", Hint: toneHint},
		p2Content(s2, 1): {Key: p2Content(s2, 1), Reason: "new"},
	}
	if !reflect.DeepEqual(pending, want) {
		t.Errorf("pending = %v\nwant %v", pending, want)
	}
	if it, ok := pending[p2Content(s1, 4)]; ok {
		t.Errorf("the old key of the requeued last line is still queued: %+v", it)
	}
	for key, it := range pending {
		if it.Lease != nil {
			t.Errorf("%s is still leased after the transition: %+v", key, it.Lease)
		}
	}

	applied, err := p.store().LoadApplied()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(applied.Requeue, []string{"2026-11-02-s1-tone", "2026-11-03-bufs"}) || !slices.Equal(applied.Rewrite, []string{"2026-11-03-s1-lines"}) {
		t.Errorf("applied.json = %+v, want both request files and the rewrite file", applied)
	}
}

// checkPhase2StaleBatch hands in the batch the killed run took before the
// second snapshot, as a run that comes back to life would: its keys are of
// the first snapshot, and the transition took its leases, so every unit is
// turned away and nothing changes (the first key now names the inserted line,
// the second the line that took the place of the requeued one).
func checkPhase2StaleBatch(t *testing.T, p *phase2, id string) {
	t.Helper()
	var b batch.Batch
	if err := json.Unmarshal(readFile(t, filepath.Join(p.root, "work", "batches", id+".json")), &b); err != nil {
		t.Fatal(err)
	}
	var out []batch.OutUnit
	for _, u := range b.Units {
		out = append(out, batch.OutUnit{Key: u.Key, ZH: fakeTranslate(u)})
	}
	writeOut(t, p.root, id, out...)
	before := p.state()
	code, stdout, stderr := lct(t, "--json", "batch", "submit", id)
	var sub submitOut
	if err := json.Unmarshal([]byte(stdout), &sub); err != nil {
		t.Fatalf("batch submit %s: exit %d, %q (stderr %q): %v", id, code, stdout, stderr, err)
	}
	var rejected []string
	for _, r := range sub.Rejected {
		if len(r.Errors) == 1 && r.Errors[0].Code == batch.CodeLeaseLost {
			rejected = append(rejected, r.Key)
		}
	}
	slices.Sort(rejected)
	if code != 1 || len(sub.Accepted)+len(sub.Released) != 0 || !slices.Equal(rejected, []string{p2Content(s1, 1), p2Content(s1, 4)}) {
		t.Errorf("stale batch %s: exit %d, submit = %+v, want both units turned away as %s", id, code, sub, batch.CodeLeaseLost)
	}
	if after := p.state(); !reflect.DeepEqual(after, before) {
		t.Errorf("handing in the stale batch changed the state:\nbefore %v\nafter  %v", before, after)
	}
}

// checkPhase2BatchFiles checks the files of the batches of one translation
// run, in the order they were handed out: the files that name the keywords
// (Bufs.json) come first, in batches of keyword files only; a batch lists the
// files of its units, which name their own, all of the family of the batch; a
// story batch holds one file. The files of the replay are small, so a run that
// hands out two files of one family outside the stories, both keyword files
// or both not, has a batch with both, and one that does not has none with
// more than one file. The replay has no such two files (Bufs.json is of
// other, Passives.json of ally, and BgmLyrics/L.json is the one file of
// story-main outside StoryData/), so every batch of it holds one file.
func checkPhase2BatchFiles(t *testing.T, root string, batches []batch.Batch) {
	t.Helper()
	r, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(batches) == 0 {
		t.Fatal("the run handed out no batch")
	}
	keywordsDone, merged := false, false
	// The files outside the stories, by family and by whether they name
	// keywords: the files that may share a batch.
	plain := map[string]map[string]bool{}
	for _, b := range batches {
		keyword := names.IsKeywordFile(b.File)
		if keyword && keywordsDone {
			t.Errorf("batch %s of %s comes after a batch of no keyword file", b.ID, b.File)
		}
		if keyword && slices.ContainsFunc(b.Files, func(f string) bool { return !names.IsKeywordFile(f) }) {
			t.Errorf("batch %s of the keyword file %s holds files that are none: %q", b.ID, b.File, b.Files)
		}
		keywordsDone = keywordsDone || !keyword
		merged = merged || len(b.Files) > 1
		var files []string
		for _, u := range b.Units {
			if file, _, err := units.SplitKey(u.Key); err != nil || u.File != file {
				t.Errorf("batch %s: unit %s names the file %q", b.ID, u.Key, u.File)
			}
			if !slices.Contains(files, u.File) {
				files = append(files, u.File)
			}
		}
		if len(b.Files) == 0 || !slices.Equal(b.Files, files) || b.File != b.Files[0] {
			t.Errorf("batch %s: file %q, files %q, want the files of its units %q", b.ID, b.File, b.Files, files)
		}
		for _, f := range b.Files {
			if r.Family(f) != b.Family {
				t.Errorf("batch %s of %s holds %s, of %s", b.ID, b.Family, f, r.Family(f))
			}
			if batch.IsStory(f) {
				if len(b.Files) != 1 {
					t.Errorf("story batch %s holds %q, want one file", b.ID, b.Files)
				}
				continue
			}
			group := fmt.Sprintf("%s, keyword file %v", b.Family, names.IsKeywordFile(f))
			if plain[group] == nil {
				plain[group] = map[string]bool{}
			}
			plain[group][f] = true
		}
	}
	mergeable := false
	for _, files := range plain {
		mergeable = mergeable || len(files) > 1
	}
	if merged != mergeable {
		t.Errorf("a batch of several files: %v; files of one family outside the stories: %v; want both or neither (%v)", merged, mergeable, plain)
	}
}

// checkPhase2Batches checks what the translator was handed after the second
// plan: each unit with what it needs to be translated again.
func checkPhase2Batches(t *testing.T, batches []batch.Batch) {
	t.Helper()
	handed := handedOut(batches)
	keys := slices.Sorted(maps.Keys(handed))
	if want := []string{bufs + "|/dataList/0/desc", bufs + "|/dataList/0/name", p2Content(s1, 1), p2Content(s1, 2), p2Content(s1, 5), p2Content(s2, 1)}; !slices.Equal(keys, want) {
		t.Fatalf("units handed out = %q, want %q", keys, want)
	}
	if u := handed[p2Content(s1, 2)]; u.Reason != plan.ReasonAdjustmentExpired || u.PrevKR != "비 오는 날의 모험도 나쁘지 않소!" || u.PrevZH != "譯" ||
		u.Hint != toneHint || u.Refs[plan.RefAdjustment] != adjChangedZH || u.Speaker == "" {
		t.Errorf("the reworded line = %+v, want its previous version, the hint of its request, the expired adjustment as a reference and its speaker", u)
	}
	if u := handed[p2Content(s1, 5)]; u.Reason != "requeue" || u.Hint != toneHint || u.PrevZH != "譯文" || u.KR != "감기에 걸려도 난 모른다." {
		t.Errorf("the requeued last line = %+v, want it at its new key with the hint and its previous translation, rewritten", u)
	}
	if u := handed[p2Content(s1, 1)]; u.Reason != "new" || u.Refs[plan.RefAdjustment] != "" {
		t.Errorf("the inserted line = %+v, want a new line without the adjustment of another", u)
	}
	if u := handed[bufs+"|/dataList/0/name"]; u.Reason != "requeue" || u.Hint != requeueHint || u.PrevZH != "譯" {
		t.Errorf("the requeued name = %+v, want the hint of the request and its previous translation", u)
	}
}

// checkPhase2SecondBuild checks the build of the second snapshot: the review
// issue, the build record and the strings of the pack.
func checkPhase2SecondBuild(t *testing.T, p *phase2, built buildOut) {
	t.Helper()
	if r := built.Report; r == nil || r.Pending != 0 || len(r.FrozenFallback) != 0 || r.StructuralFallback != 0 ||
		r.Origins["claude"] != 16 || r.Origins["human"] != 1 || r.Origins["copy"] != 9 {
		t.Errorf("second build report = %+v, want nothing pending, 16 claude, 1 human and 9 copy records", built.Report)
	}

	info, err := build.ReadInfo(filepath.Join(p.root, ".work", "tw"))
	if err != nil {
		t.Fatal(err)
	}
	if info.RawTag != phase2Raw2 || info.Model != "e2e" || info.CostUSD != 0.01 || info.Stopped != translate.StopDone {
		t.Errorf("build.json = %+v, want %s translated by e2e for $0.01", info, phase2Raw2)
	}

	if built.ReviewIssueParts != 1 {
		t.Errorf("review issue parts = %d, want 1", built.ReviewIssueParts)
	}
	title := string(readFile(t, filepath.Join(p.root, "out", "review-issue", "title.txt")))
	if title != "審核：2026-11-05 更新\n" {
		t.Errorf("review issue title = %q", title)
	}
	// The adjustment that expired and the frozen line whose KR changed, and
	// nothing else.
	body := string(readFile(t, filepath.Join(p.root, "out", "review-issue", "01.md")))
	for _, want := range []string{
		"本次更新有 2 項需要你處理",
		// The reworded line moved one line down: the item names both keys.
		"### 人工調整已過期", "- [ ] 調整 `" + p2Content(s1, 1) + "`（這一版的單位：`" + p2Content(s1, 2) + "`）",
		"### 凍結檔的 KR 有變", "- [ ] `" + p2Content(lyrics, 1) + "`",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("review issue lacks %q:\n%s", want, body)
		}
	}
	if n := strings.Count(body, "### "); n != 2 {
		t.Errorf("review issue has %d sections, want 2:\n%s", n, body)
	}

	wantS1 := map[string]string{
		"/dataList/0/place": "譯文", "/dataList/0/content": "譯文",
		"/dataList/1/content": "譯",
		"/dataList/2/model":   "돈키호테", "/dataList/2/content": "譯",
		"/dataList/3/model": "이스마엘", "/dataList/3/content": adjKeptZH,
		"/dataList/4/model": "파우스트", "/dataList/4/content": "譯文",
		"/dataList/5/model": "이스마엘", "/dataList/5/content": "譯", // translated again
	}
	if got := p.packStrings(s1); !reflect.DeepEqual(got, wantS1) {
		t.Errorf("S1 in the pack = %v\nwant %v", got, wantS1)
	}
	if got := p.packStrings(s2); got["/dataList/0/content"] != "譯文" || got["/dataList/1/content"] != "譯" {
		t.Errorf("S2 in the pack = %v, want the reused line and a new one", got)
	}
	if got := p.packStrings(lyrics); got["/dataList/1/content"] != "譯" {
		t.Errorf("lyrics in the pack = %v, want the old translation of the reworded line", got)
	}
	if _, err := os.Stat(filepath.Join(p.root, ".work", "tw", "LimbusCompany_Data", "Lang", "zh-TW", "StoryData", "S3.json")); err == nil {
		t.Error("the pack still holds the deleted S3.json")
	}
}

// A pipeline with units to translate and no key or no model records that, and
// that is all it takes for a person to be told: the build reads the record,
// the review issue and lct review list the translation that did not run, and
// the publish commit takes the record to tw.
func TestNotRunReachesTheReviewIssue(t *testing.T) {
	fixture, err := filepath.Abs(phase2Fixture)
	if err != nil {
		t.Fatal(err)
	}
	p := &phase2{t: t, root: e2eRepo(t), fixture: fixture, fakeDir: t.TempDir()}
	t.Chdir(p.root)
	t.Setenv("LCT_MODEL", "") // the case: no model, and no key
	t.Setenv(sourceDateEpoch, "")

	p.update("v1", testToken, testCreatedAt, phase2Raw1)
	var learned learnReport
	mustLct(t, &learned, "fields", "learn", "--write")
	runGitIn(t, p.root, "commit", "-q", "-am", "rules: learn the field rules of the packs")
	p.plan()

	const why = "缺少 ANTHROPIC_API_KEY 或 LCT_MODEL，待譯的單位沒有翻譯"
	var res translate.Result
	mustLct(t, &res, "translate", "--not-run", why)
	if res.Stopped != translate.StopNotRun || res.Error != why || res.PendingBefore != 14 || res.Runs != 0 {
		t.Fatalf("translate --not-run = %+v, want the record of 14 units that were not translated", res)
	}
	if c := pendingNow(t); c.Total != 14 {
		t.Errorf("pending after --not-run = %+v, want the 14 units still waiting", c)
	}

	built := p.build()
	if built.ReviewIssueParts != 1 {
		t.Fatalf("build made %d review issue parts, want 1", built.ReviewIssueParts)
	}
	issue := string(readFile(t, filepath.Join(p.root, "out", "review-issue", "01.md")))
	for _, want := range []string{"### 翻譯沒有執行", "`" + why + "`（待譯 14 個單位）"} {
		if !strings.Contains(issue, want) {
			t.Errorf("the review issue lacks %q:\n%s", want, issue)
		}
	}
	var review reviewReport
	mustLct(t, &review, "review")
	if len(review.Sections) != 1 || review.Sections[0].Kind != build.KindNotTranslated || len(review.Sections[0].Items) != 1 {
		t.Errorf("lct review = %+v, want the one section %s", review.Sections, build.KindNotTranslated)
	}
	info, err := build.ReadInfo(filepath.Join(p.root, ".work", "tw"))
	if err != nil || info.Stopped != translate.StopNotRun || info.StopError != why || info.Model != "" {
		t.Errorf("build.json = %+v, %v, want not-run with its reason and no model", info, err)
	}

	pub := p.publish()
	if got := runGitIn(t, p.root, "-C", filepath.Join(".work", "tw"), "show", "HEAD:.meta/"+translate.MetaFile); !strings.Contains(got, `"stopped": "not-run"`) {
		t.Errorf("the tw commit %s holds .meta/translate.json = %q, want the record of the translation that did not run", pub.TWCommit, got)
	}

	// Once the units are translated the record no longer says anything.
	t.Setenv("LCT_MODEL", "e2e")
	t.Setenv(fakeClaudeEnv, p.fakeDir)
	claude, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	p.claude = claude
	p.translate(14)
	p.build()
	if _, err := os.Stat(filepath.Join(p.root, "out", "review-issue")); err == nil {
		t.Error("the review issue is still made after the units were translated")
	}
}
