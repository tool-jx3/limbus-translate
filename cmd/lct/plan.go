package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/plan"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/rules"
)

func init() {
	cli.Register(cli.Command{
		Name: "plan",
		Usage: "carry the tw state to the current raw snapshot, decide the source of every unit, " +
			"apply the adjustments and the requeue and rewrite files, and write .meta/plan.json",
		Run: runPlan,
	})
}

// runPlan is the plan step of CI (spec 8.2): it runs after every snapshot and
// every change to the rules on main, before the translation.
func runPlan(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("plan", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	if err := flags.Parse(args); err != nil {
		return cli.UserError("plan: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("plan: unexpected argument " + flags.Arg(0))
	}
	cfg := ctx.Config
	if cfg.Worktrees.Raw == "" || cfg.Worktrees.TW == "" {
		return cli.UserError("plan: worktrees.raw and worktrees.tw must be set in lct.yaml")
	}

	rawWorktree := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.Raw))
	src, err := raw.LastSource(rawWorktree)
	if err != nil {
		return err
	}
	if src == nil {
		return cli.UserError(fmt.Sprintf("plan: %s has no snapshot; run lct snapshot first", cfg.Worktrees.Raw))
	}
	set, err := pack.OpenRaw(rawWorktree)
	if err != nil {
		return err
	}
	r, err := rules.Load(ctx.Root)
	if err != nil {
		return err
	}
	fr, err := fields.Load(ctx.Root)
	if err != nil {
		return err
	}
	commit, err := rulesCommit(ctx, "plan")
	if err != nil {
		return err
	}

	twRoot := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.TW))
	if err := gitx.EnsureWorktree(ctx.Root, twBranch, twRoot); err != nil {
		return err
	}
	// Before the first plan the state belongs to the snapshot of the last
	// build, if there was one.
	var buildRawTag string
	info, err := build.ReadInfo(twRoot)
	switch {
	case err == nil:
		buildRawTag = info.RawTag
	case !errors.Is(err, build.ErrNotBuilt):
		return err
	}

	sum, err := plan.Run(plan.Inputs{
		Root:        ctx.Root,
		RawWorktree: rawWorktree,
		TWRoot:      twRoot,
		RulesCommit: commit,
		BuildRawTag: buildRawTag,
		Src:         set,
		Frozen:      cfg.Frozen,
		Fields:      fr,
		Rules:       r,
		TmpDir:      os.TempDir(),
	})
	if err != nil {
		return err
	}
	return ctx.Emit(sum)
}
