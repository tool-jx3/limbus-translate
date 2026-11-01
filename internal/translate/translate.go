// Package translate is the translation loop of CI (spec 9.2). It runs the
// Claude Code CLI headless with the translate-pending skill, round after
// round, until nothing is left to hand out, the money is spent, the time is
// nearly up, a round gets nowhere or a round fails. Whatever was translated
// by then is built and released; the rest stays in the pending queue for the
// next time.
//
// The skill does the translating (through lct batch next and batch submit);
// this package only starts it, bounds it and keeps the books. It reads the
// queue through Options.Available and never touches the store.
package translate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

// MetaFile is the file of the store (.meta/) that holds the Result of the
// last run of lct translate.
const MetaFile = "translate.json"

// The reasons Run stops (Result.Stopped).
const (
	// StopDone: no pending unit is left to hand out.
	StopDone = "done"
	// StopCost: the runs cost as much as the cap, or more.
	StopCost = "cost"
	// StopDeadline: another run might not end before the deadline.
	StopDeadline = "deadline"
	// StopNoProgress: two runs in a row left as many units as before.
	StopNoProgress = "no-progress"
	// StopError: a run failed, or the queue could not be counted.
	StopError = "error"
)

const (
	// defaultClaude is the command run when Options.Claude is empty.
	defaultClaude = "claude"
	// skill is the prompt of a run: the skill that translates the queue.
	skill = "/translate-pending"
	// stallLimit is how many runs in a row may leave the count of available
	// units where it was before Run gives up.
	stallLimit = 2
	// defaultWaitDelay is how long Wait waits for the output of a process that
	// is killed or has exited, in case a descendant that left the process
	// group still holds the pipes.
	defaultWaitDelay = 5 * time.Second
	// stderrKept is how much of the end of a run's stderr an error carries.
	stderrKept = 1024
	// subtypeMaxTurns is the subtype of the result of a run that stopped at its
	// --max-turns limit.
	subtypeMaxTurns = "error_max_turns"
	// snippetLen is the most characters of a run's output an error quotes.
	snippetLen = 300
)

// Options say what to run and when to stop. Every field but Claude and Log
// must be set.
type Options struct {
	// Root is the working directory of every run: the repo root, where the
	// skill and bin/lct are.
	Root string
	// Claude is the command to run; the empty string means "claude", looked
	// up in PATH.
	Claude string
	// Model is the model of the runs. It is given with --model and in the
	// environment variable LCT_MODEL, which lct batch submit records.
	Model string
	// MaxUSD caps the spend: no new run starts once the runs so far cost this
	// much, so the last run may go over it by what one run costs.
	MaxUSD float64
	// Deadline is the time by which every run must be over. No run starts
	// that could still be going at Deadline, i.e. when now + RunTimeout is
	// later.
	Deadline time.Time
	// MaxTurns is the --max-turns of a run.
	MaxTurns int
	// RunTimeout is how long one run may take. A run that takes longer is
	// killed, with every process it started, and is an error.
	RunTimeout time.Duration
	// Available counts the units a run can still take: the pending ones that
	// are in KR, not leased and not waiting for a person (batch.Available).
	// It is called before every run, so it must read the queue anew.
	Available func() (int, error)
	// Log gets one line a round. Nil means no log.
	Log io.Writer

	// waitDelay is how long Wait waits for the output of a run that has
	// exited; zero means defaultWaitDelay. Only the tests set it.
	waitDelay time.Duration
}

// Result is what a Run did.
type Result struct {
	Model string `json:"model"`
	// Runs counts the claude processes that were started.
	Runs int `json:"runs"`
	// CostUSD is the sum of total_cost_usd of the runs. A run that failed but
	// said what it cost is in it.
	CostUSD float64 `json:"cost_usd"`
	// Stopped is why Run ended, one of the Stop constants; Error says what
	// went wrong when it is StopError.
	Stopped string `json:"stopped"`
	Error   string `json:"error,omitempty"`
	// PendingBefore and PendingAfter are the available units before the first
	// run and as of the end.
	PendingBefore int `json:"pending_before"`
	PendingAfter  int `json:"pending_after"`
	// RawTag and RulesCommit name the inputs the translation was made under.
	// Run leaves them empty; the caller, who knows them, fills them in before
	// the Result is stored, so that a build can tell whether it is of the same
	// inputs.
	RawTag      string `json:"raw_tag"`
	RulesCommit string `json:"rules_commit"`
}

// String is the report of the CLI.
func (r *Result) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "translate: stopped (%s) after %d runs of %s costing $%.2f; available %d -> %d",
		r.Stopped, r.Runs, r.Model, r.CostUSD, r.PendingBefore, r.PendingAfter)
	if r.Error != "" {
		b.WriteString("\nerror: " + r.Error)
	}
	return b.String()
}

// check reports an option that is missing or out of range.
func (o Options) check() error {
	var bad string
	switch {
	case o.Root == "":
		bad = "Root is empty"
	case o.Model == "":
		bad = "Model is empty"
	case !(o.MaxUSD > 0):
		bad = "MaxUSD must be positive"
	case o.Deadline.IsZero():
		bad = "Deadline is not set"
	case o.MaxTurns <= 0:
		bad = "MaxTurns must be positive"
	case o.RunTimeout <= 0:
		bad = "RunTimeout must be positive"
	case o.Available == nil:
		bad = "Available is not set"
	}
	if bad != "" {
		return errors.New("translate: " + bad)
	}
	return nil
}

// Run translates until it must stop. Before each round it checks, in this
// order: whether nothing is available (StopDone), the cap is reached
// (StopCost), a run might not end before the deadline (StopDeadline) and the
// last two runs got nowhere (StopNoProgress); when none holds it starts one
// run of the skill. A run that fails ends Run with StopError.
//
// A run fails when it cannot be started, exits non-zero, prints something
// that is not a JSON object or reports is_error, when its total_cost_usd is
// missing or is not a cost (negative), and when it takes longer than
// RunTimeout (it is killed). A run that exits cleanly and prints its result
// is not failed by a descendant that keeps the pipes open, and whatever a
// run left running is killed when it is over. The one exception is a run that stopped at
// its --max-turns limit (subtype "error_max_turns"): the skill hands out batch
// after batch, so running out of turns is how a run normally ends. Whatever
// is_error and the exit status say, such a round counts as a normal one: its
// cost is added and the loop goes on, left to the other reasons to stop. It
// must still report its cost, and a run that timed out is not let through.
//
// The error is for bad options only. Every problem of the runs themselves,
// and of Available, ends in a Result that says StopError, so that what the
// runs already cost is never lost.
func Run(opt Options) (*Result, error) {
	if err := opt.check(); err != nil {
		return nil, err
	}
	if opt.Claude == "" {
		opt.Claude = defaultClaude
	}
	log := opt.Log
	if log == nil {
		log = io.Discard
	}

	res := &Result{Model: opt.Model}
	finish := func(reason string) (*Result, error) {
		res.Stopped = reason
		return res, nil
	}
	fail := func(format string, args ...any) (*Result, error) {
		res.Error = fmt.Sprintf(format, args...)
		return finish(StopError)
	}

	var last, stalls int
	for round := 1; ; round++ {
		avail, err := opt.Available()
		if err != nil {
			return fail("counting the available units: %v", err)
		}
		res.PendingAfter = avail
		if round == 1 {
			res.PendingBefore = avail
		} else if avail < last {
			stalls = 0
		} else {
			stalls++
		}
		last = avail

		switch {
		case avail == 0:
			return finish(StopDone)
		case res.CostUSD >= opt.MaxUSD:
			return finish(StopCost)
		case time.Now().Add(opt.RunTimeout).After(opt.Deadline):
			return finish(StopDeadline)
		case stalls >= stallLimit:
			return finish(StopNoProgress)
		}

		fmt.Fprintf(log, "round %d: cost so far $%.2f, available %d\n", round, res.CostUSD, avail)
		cost, started, err := runOnce(opt)
		if started {
			res.Runs++
		}
		res.CostUSD += cost
		if err != nil {
			// The run may have translated some units before it failed.
			if n, aerr := opt.Available(); aerr == nil {
				res.PendingAfter = n
			}
			return fail("round %d: %v", round, err)
		}
	}
}

// errNoCost is the error of a run whose result does not say what it cost.
var errNoCost = errors.New("claude printed no total_cost_usd: the cost of the run is unknown")

// output is the JSON object a run prints with --output-format json; only the
// fields Run uses are named.
type output struct {
	TotalCostUSD *float64 `json:"total_cost_usd"`
	IsError      bool     `json:"is_error"`
	Subtype      string   `json:"subtype"`
	NumTurns     int      `json:"num_turns"`
	Result       string   `json:"result"`
}

// decodeOutput reads the object a run printed. Anything else is an error.
func decodeOutput(b []byte) (output, error) {
	var out output
	b = bytes.TrimSpace(b)
	if len(b) == 0 || b[0] != '{' {
		return out, errors.New("not a JSON object")
	}
	if err := json.NewDecoder(bytes.NewReader(b)).Decode(&out); err != nil {
		return output{}, err
	}
	return out, nil
}

// runOnce runs claude once in the process group of its own, so that a run
// that hangs is killed with everything it started. It returns the cost the
// run reported (0 when it did not), whether the process was started, and the
// reason the run failed, if it did.
func runOnce(opt Options) (cost float64, started bool, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), opt.RunTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, opt.Claude,
		"-p", skill,
		"--output-format", "json",
		"--model", opt.Model,
		"--max-turns", strconv.Itoa(opt.MaxTurns))
	cmd.Dir = opt.Root
	cmd.Env = append(os.Environ(), "LCT_MODEL="+opt.Model)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		// The process id is the id of its group. The group may be gone
		// already, which is what the kill is for.
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			return err
		}
		return nil
	}
	cmd.WaitDelay = defaultWaitDelay
	if opt.waitDelay > 0 {
		cmd.WaitDelay = opt.waitDelay
	}
	var stdout bytes.Buffer
	stderr := &tail{max: stderrKept}
	cmd.Stdout, cmd.Stderr = &stdout, stderr

	if err := cmd.Start(); err != nil {
		return 0, false, err
	}
	waitErr := cmd.Wait()
	timedOut := errors.Is(ctx.Err(), context.DeadlineExceeded)
	// Whatever the run left running dies with it, so that no descendant lives
	// on into the next round. The group may be gone already: best effort.
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)

	out, decodeErr := decodeOutput(stdout.Bytes())
	badCost := false
	if decodeErr == nil && out.TotalCostUSD != nil {
		if validCost(*out.TotalCostUSD) {
			cost = *out.TotalCostUSD
		} else {
			badCost = true
		}
	}
	// A descendant that held the pipes makes Wait give up on the output after
	// WaitDelay and say so, although claude exited cleanly and printed its
	// result: such a run is as good as one without the descendant.
	if errors.Is(waitErr, exec.ErrWaitDelay) && cmd.ProcessState != nil && cmd.ProcessState.Success() && decodeErr == nil {
		waitErr = nil
	}
	switch {
	case timedOut:
		return cost, true, fmt.Errorf("claude timed out after %v and was killed", opt.RunTimeout)
	case badCost:
		return 0, true, fmt.Errorf("claude printed total_cost_usd %v, which is not a cost: the cost of the run is unknown", *out.TotalCostUSD)
	case decodeErr == nil && out.Subtype == subtypeMaxTurns:
		// The normal end of a long run; see Run.
		if out.TotalCostUSD == nil {
			return 0, true, errNoCost
		}
	case waitErr != nil:
		msg := waitErr.Error()
		if s := stderr.String(); s != "" {
			msg += ": " + s
		}
		return cost, true, fmt.Errorf("claude: %s", msg)
	case decodeErr != nil:
		return cost, true, fmt.Errorf("claude printed %s: %s", decodeErr, snippet(stdout.String()))
	case out.IsError:
		return cost, true, fmt.Errorf("claude reported an error (subtype %q, %d turns): %s", out.Subtype, out.NumTurns, snippet(out.Result))
	case out.TotalCostUSD == nil:
		return 0, true, errNoCost
	}
	return cost, true, nil
}

// validCost reports whether c can be the cost of a run: not negative, not a
// number that is not a number, not infinite. A cost that is none would lower
// the total the cap is held against.
func validCost(c float64) bool { return c >= 0 && !math.IsInf(c, 0) }

// snippet shortens s to one line for an error message.
func snippet(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) > snippetLen {
		s = string([]rune(s)[:snippetLen]) + "..."
	}
	return s
}

// tail is a writer that keeps the last max bytes written to it.
type tail struct {
	max int
	buf []byte
}

func (t *tail) Write(p []byte) (int, error) {
	t.buf = append(t.buf, p...)
	if len(t.buf) > t.max {
		t.buf = t.buf[len(t.buf)-t.max:]
	}
	return len(p), nil
}

// String is the kept bytes as one line.
func (t *tail) String() string {
	return strings.Join(strings.Fields(strings.ToValidUTF8(string(t.buf), "")), " ")
}
