package translate

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// fakeScript stands in for the claude CLI. What it does comes from the
// environment of the test:
//
//	FAKE_DIR     directory of its state: the count of pending units (count)
//	             and what each run records (args, model, cwd, pid, child)
//	FAKE_DEC     how much a run takes off the count
//	FAKE_OUT     what a run prints to stdout
//	FAKE_STDERR  what a run prints to stderr
//	FAKE_EXIT    its exit code
//	FAKE_HANG    when set, the run starts a background sleep and waits for it
const fakeScript = `#!/bin/sh
d="$FAKE_DIR"
echo "$*" >> "$d/args"
echo "$LCT_MODEL" >> "$d/model"
pwd >> "$d/cwd"
if [ -n "$FAKE_HANG" ]; then
  echo $$ > "$d/pid"
  sleep 60 &
  echo $! > "$d/child"
  wait
fi
n=$(cat "$d/count")
echo $((n - ${FAKE_DEC:-0})) > "$d/count"
if [ -n "$FAKE_STDERR" ]; then echo "$FAKE_STDERR" >&2; fi
printf '%s\n' "$FAKE_OUT"
exit ${FAKE_EXIT:-0}
`

const okOut = `{"type":"result","subtype":"success","is_error":false,"num_turns":4,"total_cost_usd":0.1}`

// fake is a fake claude and the options to run it with.
type fake struct {
	t      *testing.T
	dir    string // the state directory, also the script's directory
	script string
	root   string // the working directory of the runs
	log    strings.Builder
}

// newFake writes the script and sets the environment it reads. count is the
// number of units pending at the start and dec what each run takes off it.
func newFake(t *testing.T, count, dec int) *fake {
	t.Helper()
	dir := t.TempDir()
	f := &fake{t: t, dir: dir, script: filepath.Join(dir, "claude"), root: t.TempDir()}
	if err := os.WriteFile(f.script, []byte(fakeScript), 0o755); err != nil {
		t.Fatal(err)
	}
	f.setCount(count)
	t.Setenv("FAKE_DIR", dir)
	t.Setenv("FAKE_DEC", strconv.Itoa(dec))
	t.Setenv("FAKE_OUT", okOut)
	t.Setenv("FAKE_STDERR", "")
	t.Setenv("FAKE_EXIT", "0")
	t.Setenv("FAKE_HANG", "")
	return f
}

func (f *fake) setCount(n int) {
	f.t.Helper()
	if err := os.WriteFile(filepath.Join(f.dir, "count"), []byte(strconv.Itoa(n)+"\n"), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fake) available() (int, error) {
	b, err := os.ReadFile(filepath.Join(f.dir, "count"))
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(b)))
}

// recorded returns the lines a run of the fake wrote to the file name.
func (f *fake) recorded(name string) []string {
	f.t.Helper()
	b, err := os.ReadFile(filepath.Join(f.dir, name))
	if err != nil {
		f.t.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}

// opts are options under which nothing stops the loop but the fake.
func (f *fake) opts() Options {
	return Options{
		Root:       f.root,
		Claude:     f.script,
		Model:      "m",
		MaxUSD:     100,
		Deadline:   time.Now().Add(time.Hour),
		MaxTurns:   7,
		RunTimeout: 10 * time.Second,
		Available:  f.available,
		Log:        &f.log,
	}
}

func mustRun(t *testing.T, opt Options) *Result {
	t.Helper()
	res, err := Run(opt)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return res
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestRunStopsWhenNothingIsPending(t *testing.T) {
	f := newFake(t, 3, 1)
	res := mustRun(t, f.opts())

	if res.Stopped != StopDone || res.Runs != 3 || res.Error != "" {
		t.Errorf("result = %+v, want done after 3 runs", res)
	}
	if !near(res.CostUSD, 0.3) {
		t.Errorf("CostUSD = %v, want the sum of the three runs, 0.3", res.CostUSD)
	}
	if res.PendingBefore != 3 || res.PendingAfter != 0 {
		t.Errorf("pending %d -> %d, want 3 -> 0", res.PendingBefore, res.PendingAfter)
	}
	if res.Model != "m" || res.RawTag != "" || res.RulesCommit != "" {
		t.Errorf("Model, RawTag, RulesCommit = %q, %q, %q, want m and the other two left to the caller", res.Model, res.RawTag, res.RulesCommit)
	}

	// One line a round: the round, the cost so far and the units available.
	lines := strings.Split(strings.TrimSpace(f.log.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("log has %d lines, want one for each of the 3 rounds:\n%s", len(lines), f.log.String())
	}
	for i, want := range []string{"round 1", "$0.00", "available 3"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("log line 1 lacks %q (checked %d): %s", want, i, lines[0])
		}
	}
	if !strings.Contains(lines[2], "round 3") || !strings.Contains(lines[2], "$0.20") || !strings.Contains(lines[2], "available 1") {
		t.Errorf("log line 3 = %q, want round 3, $0.20 so far and 1 available", lines[2])
	}
}

func TestRunDoesNotRunWhenNothingIsPending(t *testing.T) {
	f := newFake(t, 0, 0)
	res := mustRun(t, f.opts())
	if res.Stopped != StopDone || res.Runs != 0 || res.CostUSD != 0 || res.PendingBefore != 0 || res.PendingAfter != 0 {
		t.Errorf("result = %+v, want done at once", res)
	}
	if _, err := os.Stat(filepath.Join(f.dir, "args")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("claude ran although nothing was pending (%v)", err)
	}
}

func TestRunStopsAtCostCap(t *testing.T) {
	f := newFake(t, 10, 1)
	t.Setenv("FAKE_OUT", `{"is_error":false,"total_cost_usd":0.4}`)
	opt := f.opts()
	opt.MaxUSD = 1.0
	res := mustRun(t, opt)

	if res.Stopped != StopCost || res.Runs != 3 {
		t.Errorf("result = %+v, want cost after 3 runs", res)
	}
	if !near(res.CostUSD, 1.2) {
		t.Errorf("CostUSD = %v, want 1.2", res.CostUSD)
	}
	if res.PendingBefore != 10 || res.PendingAfter != 7 {
		t.Errorf("pending %d -> %d, want 10 -> 7", res.PendingBefore, res.PendingAfter)
	}
}

// Nothing left to translate is the better reason to stop, whatever the last
// run cost.
func TestRunChecksDoneBeforeCost(t *testing.T) {
	f := newFake(t, 1, 1)
	t.Setenv("FAKE_OUT", `{"is_error":false,"total_cost_usd":5}`)
	opt := f.opts()
	opt.MaxUSD = 1
	if res := mustRun(t, opt); res.Stopped != StopDone || res.Runs != 1 {
		t.Errorf("result = %+v, want done after 1 run", res)
	}
}

func TestRunStopsNearTheDeadline(t *testing.T) {
	f := newFake(t, 5, 1)
	opt := f.opts()
	opt.Deadline = time.Now().Add(time.Minute)
	opt.RunTimeout = 5 * time.Minute // a run may not fit
	res := mustRun(t, opt)

	if res.Stopped != StopDeadline || res.Runs != 0 || res.PendingBefore != 5 || res.PendingAfter != 5 {
		t.Errorf("result = %+v, want deadline before the first run", res)
	}
}

func TestRunStopsWithoutProgress(t *testing.T) {
	f := newFake(t, 5, 0)
	res := mustRun(t, f.opts())

	if res.Stopped != StopNoProgress || res.Runs != 2 {
		t.Errorf("result = %+v, want no-progress after 2 runs", res)
	}
	if res.PendingBefore != 5 || res.PendingAfter != 5 {
		t.Errorf("pending %d -> %d, want 5 -> 5", res.PendingBefore, res.PendingAfter)
	}
}

// One run that gets nowhere is not the end: the units it leased are out of
// the count until the next run takes them up.
func TestRunTakesAStallOfOneRun(t *testing.T) {
	f := newFake(t, 4, 0)
	calls := 0
	opt := f.opts()
	opt.Available = func() (int, error) {
		calls++
		n, err := f.available()
		if calls == 3 { // after the second run
			n--
			f.setCount(n)
		}
		return n, err
	}
	res := mustRun(t, opt)
	// 4, 4 (stall 1), 3 (progress), 3 (stall 1), 3 (stall 2).
	if res.Stopped != StopNoProgress || res.Runs != 4 {
		t.Errorf("result = %+v, want no-progress after 4 runs", res)
	}
}

func TestRunStopsOnAnError(t *testing.T) {
	f := newFake(t, 5, 1)
	t.Setenv("FAKE_OUT", `{"is_error": true}`)
	res := mustRun(t, f.opts())

	if res.Stopped != StopError || res.Runs != 1 || res.Error == "" {
		t.Errorf("result = %+v, want error after 1 run with a message", res)
	}
}

func TestRunStopsOnAnErrorAndKeepsItsCost(t *testing.T) {
	f := newFake(t, 5, 1)
	t.Setenv("FAKE_OUT", `{"type":"result","subtype":"error_during_execution","is_error":true,"num_turns":9,"total_cost_usd":0.7,"result":"it broke"}`)
	res := mustRun(t, f.opts())

	if res.Stopped != StopError || !near(res.CostUSD, 0.7) {
		t.Errorf("result = %+v, want error and the 0.7 the run cost", res)
	}
	for _, want := range []string{"error_during_execution", "9 turns", "it broke"} {
		if !strings.Contains(res.Error, want) {
			t.Errorf("Error = %q, lacks %q", res.Error, want)
		}
	}
	// The run took one unit before it failed; the result says where it stands.
	if res.PendingBefore != 5 || res.PendingAfter != 4 {
		t.Errorf("pending %d -> %d, want 5 -> 4", res.PendingBefore, res.PendingAfter)
	}
}

// A skill that works batch after batch is expected to run out of turns: such a
// round ends normally, whatever is_error says, and the loop goes on.
func TestRunContinuesAfterARunOutOfTurns(t *testing.T) {
	const outOfTurns = `{"type":"result","subtype":"error_max_turns","is_error":true,"total_cost_usd":0.2,"num_turns":7}`
	for name, exit := range map[string]string{"exit 0": "0", "exit 1": "1"} {
		f := newFake(t, 3, 1)
		t.Setenv("FAKE_OUT", outOfTurns)
		t.Setenv("FAKE_EXIT", exit)
		res := mustRun(t, f.opts())

		if res.Stopped != StopDone || res.Runs != 3 || res.Error != "" {
			t.Errorf("%s: result = %+v, want done after 3 runs, no error", name, res)
		}
		if !near(res.CostUSD, 0.6) {
			t.Errorf("%s: CostUSD = %v, want the three runs' 0.6", name, res.CostUSD)
		}
	}
}

// Running out of turns changes nothing about the other reasons to stop.
func TestRunStopsAfterRunsOutOfTurnsAsUsual(t *testing.T) {
	for _, c := range []struct {
		name         string
		count, dec   int
		out          string
		maxUSD       float64
		stopped      string
		runs         int
		wantCostNear float64
	}{
		{"cost cap", 10, 1, `{"subtype":"error_max_turns","is_error":true,"total_cost_usd":0.6}`, 1.0, StopCost, 2, 1.2},
		{"no progress", 10, 0, `{"subtype":"error_max_turns","is_error":true,"total_cost_usd":0.1}`, 100, StopNoProgress, 2, 0.2},
	} {
		f := newFake(t, c.count, c.dec)
		t.Setenv("FAKE_OUT", c.out)
		opt := f.opts()
		opt.MaxUSD = c.maxUSD
		res := mustRun(t, opt)
		if res.Stopped != c.stopped || res.Runs != c.runs || !near(res.CostUSD, c.wantCostNear) {
			t.Errorf("%s: result = %+v, want %s after %d runs costing %v", c.name, res, c.stopped, c.runs, c.wantCostNear)
		}
	}
}

// Only the turn limit is let through: the cost must still be reported, and a
// run that hung is still killed and still an error.
func TestRunOutOfTurnsStillNeedsItsCost(t *testing.T) {
	f := newFake(t, 5, 1)
	t.Setenv("FAKE_OUT", `{"subtype":"error_max_turns","is_error":true,"num_turns":7}`)
	res := mustRun(t, f.opts())
	if res.Stopped != StopError || res.Runs != 1 || !strings.Contains(res.Error, "total_cost_usd") {
		t.Errorf("result = %+v, want error after 1 run, naming total_cost_usd", res)
	}
}

func TestRunStopsOnANonZeroExit(t *testing.T) {
	f := newFake(t, 5, 1)
	t.Setenv("FAKE_OUT", "")
	t.Setenv("FAKE_EXIT", "3")
	t.Setenv("FAKE_STDERR", "boom")
	res := mustRun(t, f.opts())

	if res.Stopped != StopError || res.Runs != 1 {
		t.Fatalf("result = %+v, want error after 1 run", res)
	}
	for _, want := range []string{"exit status 3", "boom"} {
		if !strings.Contains(res.Error, want) {
			t.Errorf("Error = %q, lacks %q", res.Error, want)
		}
	}
}

// A run that fails but still says what it cost is paid for.
func TestRunStopsOnANonZeroExitAndKeepsItsCost(t *testing.T) {
	f := newFake(t, 5, 1)
	t.Setenv("FAKE_OUT", `{"is_error":false,"total_cost_usd":0.25}`)
	t.Setenv("FAKE_EXIT", "1")
	res := mustRun(t, f.opts())
	if res.Stopped != StopError || !near(res.CostUSD, 0.25) {
		t.Errorf("result = %+v, want error and 0.25", res)
	}
}

func TestRunStopsOnOutputThatIsNotJSON(t *testing.T) {
	for _, out := range []string{"not json", "", "[1,2]"} {
		f := newFake(t, 5, 1)
		t.Setenv("FAKE_OUT", out)
		res := mustRun(t, f.opts())
		if res.Stopped != StopError || res.Runs != 1 || res.Error == "" {
			t.Errorf("output %q: result = %+v, want error after 1 run", out, res)
		}
	}
}

// Without the cost of a run the cap means nothing, so a run that does not
// report it is not trusted.
func TestRunStopsWhenTheCostIsNotReported(t *testing.T) {
	f := newFake(t, 5, 1)
	t.Setenv("FAKE_OUT", `{"type":"result","subtype":"success","is_error":false}`)
	res := mustRun(t, f.opts())
	if res.Stopped != StopError || res.Runs != 1 || !strings.Contains(res.Error, "total_cost_usd") {
		t.Errorf("result = %+v, want error after 1 run, naming total_cost_usd", res)
	}
}

func TestRunStopsWhenClaudeCannotBeStarted(t *testing.T) {
	f := newFake(t, 5, 1)
	opt := f.opts()
	opt.Claude = filepath.Join(f.dir, "no-such-claude")
	res := mustRun(t, opt)
	if res.Stopped != StopError || res.Runs != 0 || res.Error == "" {
		t.Errorf("result = %+v, want error with no run counted", res)
	}
}

func TestRunStopsWhenAvailableFails(t *testing.T) {
	f := newFake(t, 5, 1)
	opt := f.opts()
	calls := 0
	opt.Available = func() (int, error) {
		calls++
		if calls == 2 {
			return 0, errors.New("store is gone")
		}
		return f.available()
	}
	res := mustRun(t, opt)

	// What the first run cost is not lost with the error.
	if res.Stopped != StopError || res.Runs != 1 || !near(res.CostUSD, 0.1) || !strings.Contains(res.Error, "store is gone") {
		t.Errorf("result = %+v, want error after 1 run, 0.1 spent and the message", res)
	}
}

func TestRunKillsAHungRun(t *testing.T) {
	f := newFake(t, 5, 1)
	t.Setenv("FAKE_HANG", "1")
	opt := f.opts()
	opt.RunTimeout = 200 * time.Millisecond

	start := time.Now()
	res := mustRun(t, opt)
	if d := time.Since(start); d > time.Second {
		t.Errorf("Run took %v, want it back within a second", d)
	}
	if res.Stopped != StopError || res.Runs != 1 || !strings.Contains(res.Error, "timed out") {
		t.Errorf("result = %+v, want error after 1 run, timed out", res)
	}

	// The script and the sleep it started are both gone.
	for _, name := range []string{"pid", "child"} {
		pid, err := strconv.Atoi(f.recorded(name)[0])
		if err != nil {
			t.Fatal(err)
		}
		waitGone(t, name, pid)
	}
}

// waitGone fails the test unless the process is gone (or a zombie, which is
// dead but not yet collected by its new parent) within a second. It looks in
// /proc; where there is none, every process counts as gone.
func waitGone(t *testing.T, what string, pid int) {
	t.Helper()
	for deadline := time.Now().Add(time.Second); ; time.Sleep(10 * time.Millisecond) {
		stat, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		if err == nil {
			// pid (comm) state ...; comm may hold spaces, so cut after the last ")".
			s := string(stat)
			if rest := strings.TrimSpace(s[strings.LastIndexByte(s, ')')+1:]); strings.HasPrefix(rest, "Z") {
				return
			}
		} else if errors.Is(err, syscall.ESRCH) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("the %s process %d is still running", what, pid)
		}
	}
}

func TestRunPassesModelAndArguments(t *testing.T) {
	f := newFake(t, 1, 1)
	t.Setenv("LCT_MODEL", "from-the-environment") // the run's own model wins
	mustRun(t, f.opts())

	if got := f.recorded("args"); len(got) != 1 || got[0] != "-p /translate-pending --output-format json --model m --max-turns 7" {
		t.Errorf("arguments = %q, want -p /translate-pending --output-format json --model m --max-turns 7", got)
	}
	if got := f.recorded("model"); len(got) != 1 || got[0] != "m" {
		t.Errorf("LCT_MODEL = %q, want m", got)
	}
	want, err := filepath.EvalSymlinks(f.root)
	if err != nil {
		t.Fatal(err)
	}
	if got := f.recorded("cwd"); len(got) != 1 || got[0] != want {
		t.Errorf("working directory = %q, want the root %s", got, want)
	}
}

func TestRunFindsClaudeOnThePath(t *testing.T) {
	f := newFake(t, 1, 1)
	t.Setenv("PATH", f.dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	opt := f.opts()
	opt.Claude = ""
	if res := mustRun(t, opt); res.Stopped != StopDone || res.Runs != 1 {
		t.Errorf("result = %+v, want done after 1 run of the claude on the PATH", res)
	}
}

func TestRunRejectsBadOptions(t *testing.T) {
	f := newFake(t, 1, 1)
	for name, change := range map[string]func(*Options){
		"no root":       func(o *Options) { o.Root = "" },
		"no model":      func(o *Options) { o.Model = "" },
		"no cap":        func(o *Options) { o.MaxUSD = 0 },
		"no deadline":   func(o *Options) { o.Deadline = time.Time{} },
		"no max turns":  func(o *Options) { o.MaxTurns = 0 },
		"no run limit":  func(o *Options) { o.RunTimeout = 0 },
		"no available":  func(o *Options) { o.Available = nil },
		"negative cap":  func(o *Options) { o.MaxUSD = -1 },
		"negative time": func(o *Options) { o.RunTimeout = -time.Second },
	} {
		opt := f.opts()
		change(&opt)
		if res, err := Run(opt); err == nil {
			t.Errorf("%s: Run = %+v, nil; want an error", name, res)
		}
	}
	if _, err := os.Stat(filepath.Join(f.dir, "args")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("claude ran under bad options (%v)", err)
	}
}

func TestResultJSON(t *testing.T) {
	want := Result{Model: "m", Runs: 2, CostUSD: 1.5, Stopped: StopCost, PendingBefore: 9, PendingAfter: 4, RawTag: "raw-1", RulesCommit: "abc"}
	b, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	const text = `{"model":"m","runs":2,"cost_usd":1.5,"stopped":"cost","pending_before":9,"pending_after":4,"raw_tag":"raw-1","rules_commit":"abc"}`
	if string(b) != text {
		t.Errorf("JSON = %s, want %s", b, text)
	}
	want.Stopped, want.Error = StopError, "boom"
	b, _ = json.Marshal(want)
	if !strings.Contains(string(b), `"error":"boom"`) {
		t.Errorf("JSON = %s, want the error in it", b)
	}
}
