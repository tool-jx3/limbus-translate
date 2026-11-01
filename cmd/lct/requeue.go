package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/requeue"
	"github.com/tool-jx3/limbus-translate/internal/rulefile"
)

func init() {
	cli.Register(cli.Command{
		Name: "requeue",
		Usage: "send units back to the pending list: [--key K]... [--file GLOB]... [--origin O]... --reason TEXT (always required); " +
			"or reset (--needs-review) / release (--leased) pending units, narrowed by --key/--file/--origin, requeueing no record; " +
			"a hint set on a leased unit misses the batch in flight: release it with --leased --key K; " +
			"or, with --to-file TOPIC [--date YYYY-MM-DD], write the request to rules/requeue/<date>-<topic>.yaml for CI to apply once, and touch no state",
		Run: runRequeue,
	})
}

// runRequeue is the phase 1 direct mode of requeue: it writes the local
// pending list. It is a command for the person (or the controller) who runs
// the translation, not for the translators, so no skill is allowed to run it.
//
// --key, --file and --origin select records to requeue. Given with
// --needs-review or --leased they only narrow which pending units those reset
// or release (one command, one kind of change). Given with --to-file they are
// written to a request file on main instead, and the plan of CI requeues the
// records once; nothing under tw is touched then.
func runRequeue(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("requeue", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var keys, files, origins stringList
	flags.Var(&keys, "key", "a unit key, <file>|<JSON pointer> (repeatable)")
	flags.Var(&files, "file", "a pack file name or a glob such as StoryData/S1* or BgmLyrics/** (repeatable)")
	flags.Var(&origins, "origin", "only records of this origin: paratranz, claude, human, fallback or copy (repeatable; default: all but copy); a key without a record is left out")
	needsReview := flags.Bool("needs-review", false, "reset the pending units that wait for a person")
	leased := flags.Bool("leased", false, "release the leases of pending units, expired or not")
	reason := flags.String("reason", "", "why the units go back; the translator sees it as the hint (not for a unit whose batch is in flight: release it with --leased)")
	toFile := flags.String("to-file", "", "write the selection to rules/requeue/<date>-<topic>.yaml (TOPIC is the topic) for CI to apply once, instead of requeueing here; not with --needs-review or --leased")
	date := flags.String("date", "", "with --to-file: YYYY-MM-DD, the day of the file name (default: today)")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("requeue: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("requeue: unexpected argument " + flags.Arg(0))
	}
	sel := requeue.Selector{Keys: keys, Files: files, Origins: origins, NeedsReview: *needsReview, Leased: *leased}
	if *toFile == "" && *date != "" {
		return cli.UserError("requeue: --date goes with --to-file")
	}
	if *toFile != "" && (sel.NeedsReview || sel.Leased) {
		return cli.UserError("requeue: --to-file writes a request for records; it cannot be combined with --needs-review or --leased, which act on the pending list here")
	}
	if *toFile != "" && len(keys)+len(files) == 0 {
		return cli.UserError("requeue: --to-file needs at least one --key or --file (--origin alone would requeue every file)")
	}
	if len(keys)+len(files)+len(origins) == 0 && !sel.NeedsReview && !sel.Leased {
		return cli.UserError("requeue: give at least one of --key, --file, --origin, --needs-review, --leased")
	}
	hint := strings.TrimSpace(*reason)
	if hint == "" {
		return cli.UserError("requeue: --reason is required")
	}
	if err := sel.Validate(); err != nil {
		return cli.UserError("requeue: " + err.Error())
	}
	if *toFile != "" {
		return runRequeueToFile(ctx, sel, hint, *toFile, *date)
	}

	cfg := ctx.Config
	if cfg.Worktrees.Raw == "" || cfg.Worktrees.TW == "" {
		return cli.UserError("requeue: worktrees.raw and worktrees.tw must be set in lct.yaml")
	}
	rawWorktree := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.Raw))
	src, err := raw.LastSource(rawWorktree)
	if err != nil {
		return err
	}
	if src == nil {
		return cli.UserError(fmt.Sprintf("requeue: %s has no snapshot; run lct snapshot first", cfg.Worktrees.Raw))
	}
	set, err := pack.OpenRaw(rawWorktree)
	if err != nil {
		return err
	}

	res, err := requeue.Apply(openStore(ctx), set.KR, cfg.Frozen, sel, hint)
	if err != nil {
		return err
	}
	return ctx.Emit(res)
}

// requeueWritten is what "requeue --to-file" prints.
type requeueWritten struct {
	// Path is the request file, relative to the repo root.
	Path string `json:"path"`
	// ID is what the plan records when it has applied the file.
	ID string `json:"id"`
}

func (r requeueWritten) String() string {
	return fmt.Sprintf("wrote %s\ncommit it to main: CI requeues the units once, when it next builds\n", r.Path)
}

// runRequeueToFile writes the request file of sel. It needs neither the raw
// snapshot nor the tw worktree: the units are looked up when CI applies the
// file, against the KR of that day.
func runRequeueToFile(ctx *cli.Context, sel requeue.Selector, reason, topic, date string) error {
	if date == "" {
		date = ruleFileNow().Format("2006-01-02")
	}
	written, err := requeue.WriteFile(ctx.Root, date, topic, sel, reason)
	if errors.Is(err, requeue.ErrInvalid) {
		return cli.UserError("requeue: " + err.Error())
	}
	if err != nil {
		return err
	}
	return ctx.Emit(requeueWritten{Path: written, ID: rulefile.ID(path.Base(written))})
}
