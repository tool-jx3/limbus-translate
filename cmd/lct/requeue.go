package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/requeue"
)

func init() {
	cli.Register(cli.Command{
		Name:  "requeue",
		Usage: "send units back to the pending list with a reason: [--key K]... [--file GLOB]... [--origin O]... [--needs-review] [--leased] --reason TEXT",
		Run:   runRequeue,
	})
}

// runRequeue is the phase 1 direct mode of requeue: it writes the local
// pending list. It is a command for the person (or the controller) who runs
// the translation, not for the translators, so no skill is allowed to run it.
func runRequeue(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("requeue", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var keys, files, origins stringList
	flags.Var(&keys, "key", "a unit key, <file>|<JSON pointer> (repeatable)")
	flags.Var(&files, "file", "a pack file name or a glob such as StoryData/S1* or BgmLyrics/** (repeatable)")
	flags.Var(&origins, "origin", "only records of this origin: paratranz, claude, human, fallback or copy (repeatable; default: all but copy)")
	needsReview := flags.Bool("needs-review", false, "reset the pending units that wait for a person")
	leased := flags.Bool("leased", false, "release the leases of pending units, expired or not")
	reason := flags.String("reason", "", "why the units go back; the translator sees it as the hint")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("requeue: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("requeue: unexpected argument " + flags.Arg(0))
	}
	sel := requeue.Selector{Keys: keys, Files: files, Origins: origins, NeedsReview: *needsReview, Leased: *leased}
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
