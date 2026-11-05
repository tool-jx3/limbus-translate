package translate

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
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
//	FAKE_ORPHAN  when set, the run leaves a background sleep behind it that
//	             holds neither stdout nor stderr (child)
//	FAKE_HOLD    when set, the run leaves a background sleep behind it that
//	             holds stdout and stderr, so that they never reach end of file
//	             (child)
//	FAKE_TURNS   how many assistant events the run streams before FAKE_OUT
//	FAKE_STOP    a file the run waits for after those events, up to
//	             FAKE_WAIT seconds; it records seen or absent (stop)
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
if [ -n "$FAKE_ORPHAN" ]; then
  sleep 60 >/dev/null 2>&1 &
  echo $! > "$d/child"
fi
if [ -n "$FAKE_HOLD" ]; then
  sleep 60 &
  echo $! > "$d/child"
fi
i=0
while [ "$i" -lt "${FAKE_TURNS:-0}" ]; do
  printf '%s\n' '{"type":"assistant","message":{"content":[]}}'
  i=$((i + 1))
done
if [ -n "$FAKE_STOP" ]; then
  w=0
  while [ ! -e "$FAKE_STOP" ] && [ "$w" -lt "$((${FAKE_WAIT:-1} * 10))" ]; do sleep 0.1; w=$((w + 1)); done
  if [ -e "$FAKE_STOP" ]; then echo seen >> "$d/stop"; else echo absent >> "$d/stop"; fi
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
	t.Setenv("FAKE_ORPHAN", "")
	t.Setenv("FAKE_HOLD", "")
	t.Setenv("FAKE_TURNS", "0")
	t.Setenv("FAKE_STOP", "")
	t.Setenv("FAKE_WAIT", "1")
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

	// Two lines a round: the round, the cost so far and the units available;
	// then how many turns the run took and what it cost.
	lines := strings.Split(strings.TrimSpace(f.log.String()), "\n")
	if len(lines) != 6 {
		t.Fatalf("log has %d lines, want two for each of the 3 rounds:\n%s", len(lines), f.log.String())
	}
	if lines[1] != "round 1: ended after 4 turns, cost $0.10" {
		t.Errorf("log line 2 = %q, want the 4 turns and the $0.10 of round 1", lines[1])
	}
	for i, want := range []string{"round 1", "$0.00", "available 3"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("log line 1 lacks %q (checked %d): %s", want, i, lines[0])
		}
	}
	if !strings.Contains(lines[4], "round 3") || !strings.Contains(lines[4], "$0.20") || !strings.Contains(lines[4], "available 1") {
		t.Errorf("log line 5 = %q, want round 3, $0.20 so far and 1 available", lines[4])
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

// AfterRun is called after every run that ends normally, with its round; its
// error is logged and the loop goes on: what it failed to keep is kept after
// the next run, or by the caller once Run is over.
func TestRunCallsAfterRunAfterEveryRun(t *testing.T) {
	f := newFake(t, 3, 1)
	opt := f.opts()
	var rounds []int
	opt.AfterRun = func(round int) error {
		rounds = append(rounds, round)
		if round == 2 {
			return errors.New("push refused")
		}
		return nil
	}
	res := mustRun(t, opt)
	if res.Stopped != StopDone || res.Runs != 3 {
		t.Errorf("result = %+v, want done after 3 runs: a failed AfterRun does not stop the loop", res)
	}
	if !slices.Equal(rounds, []int{1, 2, 3}) {
		t.Errorf("AfterRun was called for rounds %v, want 1, 2 and 3", rounds)
	}
	if !strings.Contains(f.log.String(), "round 2") || !strings.Contains(f.log.String(), "push refused") {
		t.Errorf("the log does not say that AfterRun failed in round 2:\n%s", f.log.String())
	}
}

// A run that fails ends Run at once, without AfterRun: the caller keeps what
// there is when Run is over.
func TestRunDoesNotCallAfterRunForAFailedRun(t *testing.T) {
	f := newFake(t, 5, 1)
	t.Setenv("FAKE_OUT", `{"is_error": true}`)
	opt := f.opts()
	called := false
	opt.AfterRun = func(int) error { called = true; return nil }
	if res := mustRun(t, opt); res.Stopped != StopError {
		t.Fatalf("result = %+v, want an error", res)
	}
	if called {
		t.Error("AfterRun was called after a run that failed")
	}
}

// A run streams its events, one JSON object a line (stream-json); its result
// is the event of type result.
func TestRunReadsTheResultOfAStream(t *testing.T) {
	f := newFake(t, 1, 1)
	t.Setenv("FAKE_OUT", `{"type":"system","subtype":"init"}
{"type":"assistant","message":{"content":[]}}
{"type":"result","subtype":"success","is_error":false,"num_turns":2,"total_cost_usd":0.3}`)
	res := mustRun(t, f.opts())
	if res.Stopped != StopDone || !near(res.CostUSD, 0.3) {
		t.Errorf("result = %+v, want done at the 0.3 of the result event", res)
	}
	if !strings.Contains(f.log.String(), "round 1: ended after 2 turns, cost $0.30") {
		t.Errorf("the log does not say how many turns round 1 took:\n%s", f.log.String())
	}
}

// A stream without a result event says nothing of what the run cost: an error.
func TestRunStopsOnAStreamWithoutAResult(t *testing.T) {
	f := newFake(t, 1, 1)
	t.Setenv("FAKE_OUT", `{"type":"system","subtype":"init"}
{"type":"assistant","message":{"content":[]}}`)
	res := mustRun(t, f.opts())
	if res.Stopped != StopError || !strings.Contains(res.Error, "no result") {
		t.Errorf("result = %+v, want an error that the run printed no result", res)
	}
}

// Once a run has taken SoftTurns turns, StopFile is made: lct batch next hands
// out no new batch, so the run ends once its batch is submitted, before the
// hard limit of MaxTurns cuts a batch in half. The file is gone after Run.
func TestRunSignalsTheStopAfterSoftTurns(t *testing.T) {
	for _, tc := range []struct {
		name  string
		turns string
		want  string
	}{
		{"soft turns reached", "3", "seen"},
		{"soft turns not reached", "1", "absent"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFake(t, 1, 1)
			stop := filepath.Join(f.root, "stop")
			t.Setenv("FAKE_TURNS", tc.turns)
			t.Setenv("FAKE_STOP", stop)
			t.Setenv("FAKE_WAIT", "3")
			if tc.want == "absent" {
				t.Setenv("FAKE_WAIT", "1")
			}
			opt := f.opts()
			opt.SoftTurns, opt.StopFile = 2, stop
			if res := mustRun(t, opt); res.Stopped != StopDone {
				t.Fatalf("result = %+v, want done", res)
			}
			if got := f.recorded("stop"); len(got) != 1 || got[0] != tc.want {
				t.Errorf("the run saw the stop file: %q, want %s", got, tc.want)
			}
			if _, err := os.Stat(stop); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("the stop file is still there after Run: %v", err)
			}
			if tc.want == "seen" && !strings.Contains(f.log.String(), "round 1: 2 turns, no new batch") {
				t.Errorf("the log does not say that round 1 stopped taking batches:\n%s", f.log.String())
			}
		})
	}
}

// Progress is called while a run streams, after each line of its output, and
// not only once it is over: the fake waits for a file after its first events,
// and Progress makes it.
func TestRunReportsProgressWhileARunStreams(t *testing.T) {
	f := newFake(t, 2, 1)
	seen := filepath.Join(f.root, "seen")
	t.Setenv("FAKE_TURNS", "2")
	t.Setenv("FAKE_STOP", seen)
	t.Setenv("FAKE_WAIT", "3")
	opt := f.opts()
	var mu sync.Mutex
	var rounds []int
	opt.Progress = func(round int) {
		mu.Lock()
		defer mu.Unlock()
		rounds = append(rounds, round)
		if err := os.WriteFile(seen, nil, 0o644); err != nil {
			t.Error(err)
		}
	}
	// Each run waits for a file of its own.
	opt.Settle = func(int) error { return os.Remove(seen) }
	if res := mustRun(t, opt); res.Stopped != StopDone || res.Runs != 2 {
		t.Fatalf("result = %+v, want done after 2 runs", res)
	}
	if got := f.recorded("stop"); !slices.Equal(got, []string{"seen", "seen"}) {
		t.Errorf("the runs saw the file of Progress: %q, want seen twice: Progress was not called while they streamed", got)
	}
	// Two assistant events and the result, a run.
	mu.Lock()
	defer mu.Unlock()
	if want := []int{1, 1, 1, 2, 2, 2}; !slices.Equal(rounds, want) {
		t.Errorf("Progress was called with the rounds %v, want %v: once a line of output", rounds, want)
	}
}

// A turn is one message of the model: the stream may send it in more than one
// assistant event, with the same message id, and it counts once.
func TestRunCountsATurnOnceForEachMessage(t *testing.T) {
	f := newFake(t, 1, 1)
	stop := filepath.Join(f.root, "stop")
	t.Setenv("FAKE_OUT", `{"type":"assistant","message":{"id":"m1","content":[{"type":"text"}]}}
{"type":"assistant","message":{"id":"m1","content":[{"type":"tool_use"}]}}
{"type":"assistant","message":{"id":"m2","content":[]}}
{"type":"result","subtype":"success","is_error":false,"num_turns":2,"total_cost_usd":0.1}`)
	opt := f.opts()
	opt.SoftTurns, opt.StopFile = 3, stop
	mustRun(t, opt)
	if strings.Contains(f.log.String(), "no new batch") {
		t.Errorf("two messages in three events were taken for three turns:\n%s", f.log.String())
	}
}

// Settle is called after every run that started, one that failed too, before
// AfterRun: it deals with the batches the run left.
func TestRunSettlesEveryRun(t *testing.T) {
	f := newFake(t, 2, 1)
	opt := f.opts()
	var settled []int
	opt.Settle = func(round int) error { settled = append(settled, round); return nil }
	mustRun(t, opt)
	if !slices.Equal(settled, []int{1, 2}) {
		t.Errorf("Settle was called for rounds %v, want 1 and 2", settled)
	}

	g := newFake(t, 5, 1)
	t.Setenv("FAKE_OUT", `{"is_error": true}`)
	opt = g.opts()
	settled = nil
	opt.Settle = func(round int) error { settled = append(settled, round); return errors.New("cannot settle") }
	if res := mustRun(t, opt); res.Stopped != StopError {
		t.Fatalf("result = %+v, want an error", res)
	}
	if !slices.Equal(settled, []int{1}) {
		t.Errorf("Settle was called for rounds %v after a failed run, want 1", settled)
	}
	if !strings.Contains(g.log.String(), "cannot settle") {
		t.Errorf("the log does not say that Settle failed:\n%s", g.log.String())
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
	opt.RunTimeout = 500 * time.Millisecond

	start := time.Now()
	res := mustRun(t, opt)
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("Run took %v, want it back within 2 seconds", d)
	}
	if res.Stopped != StopError || res.Runs != 1 || !strings.Contains(res.Error, "timed out") {
		t.Errorf("result = %+v, want error after 1 run, timed out", res)
	}

	// The script and the sleep it started are both gone.
	f.wantProcessesGone("pid", "child")
}

// wantProcessesGone fails the test unless the processes whose ids the fake
// recorded in the files names are gone. A file that is missing means the
// script was killed before it got that far, and the test says so instead of
// reading a file that is not there.
func (f *fake) wantProcessesGone(names ...string) {
	f.t.Helper()
	for _, name := range names {
		b, err := os.ReadFile(filepath.Join(f.dir, name))
		if err != nil {
			f.t.Fatalf("script never started: it recorded no %s before the run was killed (%v); the run timeout is too short for this machine", name, err)
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
		if err != nil {
			f.t.Fatalf("script never started: the %s file holds %q, not a process id", name, b)
		}
		waitGone(f.t, name, pid)
	}
}

// A descendant that outlives a run that ended well is killed with its group:
// it must not live on into the next round.
func TestRunKillsWhatAnEndedRunLeftBehind(t *testing.T) {
	f := newFake(t, 1, 1)
	t.Setenv("FAKE_ORPHAN", "1")
	res := mustRun(t, f.opts())
	if res.Stopped != StopDone || res.Runs != 1 || res.Error != "" {
		t.Fatalf("result = %+v, want done after 1 run", res)
	}
	f.wantProcessesGone("child")
}

// A descendant that holds the pipes makes Wait return ErrWaitDelay even though
// claude exited cleanly and printed its result: the round is a good one, and
// the descendant is killed.
func TestRunTakesACleanExitWhoseDescendantHoldsThePipes(t *testing.T) {
	f := newFake(t, 2, 1)
	t.Setenv("FAKE_HOLD", "1")
	opt := f.opts()
	opt.waitDelay = 100 * time.Millisecond

	start := time.Now()
	res := mustRun(t, opt)
	if res.Stopped != StopDone || res.Runs != 2 || res.Error != "" || !near(res.CostUSD, 0.2) {
		t.Fatalf("result = %+v, want done after 2 runs costing 0.2, no error", res)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("Run took %v, want it back without waiting for the descendant", d)
	}
	f.wantProcessesGone("child")
}

// ErrWaitDelay is forgiven only for a run that exited cleanly and printed a
// result: with another exit status or output that is no result it stays an
// error.
func TestRunStillFailsWhenTheDescendantHoldsThePipesAndTheRunFailed(t *testing.T) {
	for name, c := range map[string]struct{ out, exit string }{
		"a non-zero exit":   {okOut, "3"},
		"no JSON":           {"done", "0"},
		"an error result":   {`{"is_error":true,"total_cost_usd":0.1}`, "0"},
		"a result w/o cost": {`{"is_error":false}`, "0"},
	} {
		f := newFake(t, 2, 1)
		t.Setenv("FAKE_HOLD", "1")
		t.Setenv("FAKE_OUT", c.out)
		t.Setenv("FAKE_EXIT", c.exit)
		opt := f.opts()
		opt.waitDelay = 100 * time.Millisecond
		res := mustRun(t, opt)
		if res.Stopped != StopError || res.Runs != 1 || res.Error == "" {
			t.Errorf("%s: result = %+v, want error after 1 run", name, res)
		}
		f.wantProcessesGone("child")
	}
}

// A cost that is negative, or not a number, is no cost: it would lower the
// total the cap is held against.
func TestRunStopsOnACostThatIsNotACost(t *testing.T) {
	for name, out := range map[string]string{
		"negative":               `{"type":"result","subtype":"success","is_error":false,"total_cost_usd":-0.5}`,
		"negative, out of turns": `{"subtype":"error_max_turns","is_error":true,"total_cost_usd":-0.5}`,
		"too large for a float":  `{"subtype":"success","is_error":false,"total_cost_usd":1e999}`,
	} {
		f := newFake(t, 5, 1)
		t.Setenv("FAKE_OUT", out)
		res := mustRun(t, f.opts())
		if res.Stopped != StopError || res.Runs != 1 || !strings.Contains(res.Error, "total_cost_usd") {
			t.Errorf("%s: result = %+v, want error after 1 run, naming total_cost_usd", name, res)
		}
		if res.CostUSD != 0 {
			t.Errorf("%s: CostUSD = %v, want 0: a cost that is not one is not added", name, res.CostUSD)
		}
	}
}

func TestValidCost(t *testing.T) {
	for _, c := range []float64{0, 0.01, 12.5} {
		if !validCost(c) {
			t.Errorf("validCost(%v) = false, want true", c)
		}
	}
	for _, c := range []float64{-0.01, -1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if validCost(c) {
			t.Errorf("validCost(%v) = true, want false", c)
		}
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

	// The folder of git is closed to the run: it holds the token of the checkout.
	if got := f.recorded("args"); len(got) != 1 || got[0] != "-p /translate-pending --output-format stream-json --verbose --model m --max-turns 7 --disallowedTools Read(./.git/**)" {
		t.Errorf("arguments = %q, want -p /translate-pending --output-format stream-json --verbose --model m --max-turns 7 --disallowedTools Read(./.git/**)", got)
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
