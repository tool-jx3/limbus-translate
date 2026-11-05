package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/publish"
)

func init() {
	cli.Register(cli.Command{
		Name:  "publish",
		Usage: "commit the build in the tw worktree and prepare the Release (tag, notes, zips; due only when the pack differs from the last Release); --check only says whether one is due for tw as it is; pushes and releases nothing",
		Run:   runPublish,
	})
}

// runPublish is a command for the workflow and the person who releases, not
// for the translators, so no skill is allowed to run it.
func runPublish(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("publish", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	check := flags.Bool("check", false, "only say, from the commits, tags and trees of tw, whether a Release is due for its HEAD; commit and write nothing, and need no build")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("publish: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("publish: unexpected argument " + flags.Arg(0))
	}
	if ctx.Config.Worktrees.TW == "" || ctx.Config.LangDir == "" {
		return cli.UserError("publish: worktrees.tw and lang_dir must be set in lct.yaml")
	}

	// The commit is made with git -C <tw>: it must be the worktree of the tw
	// branch, or it would commit into whatever repo holds that directory. The
	// Release tags are fetched too if the worktree is made from origin, as
	// they number the Release. --check reads the same tw, but only its git
	// state.
	twRoot := filepath.Join(ctx.Root, filepath.FromSlash(ctx.Config.Worktrees.TW))
	if err := gitx.EnsureWorktree(ctx.Root, twBranch, twRoot, publish.TagPrefix); err != nil {
		return err
	}
	if *check {
		res, err := publish.CheckTW(twRoot, ctx.Config.LangDir)
		if err != nil {
			return err
		}
		return ctx.Emit(res)
	}
	rawRoot := filepath.Join(ctx.Root, filepath.FromSlash(ctx.Config.Worktrees.Raw))
	res, err := publish.Run(ctx.Root, twRoot, rawRoot, filepath.Join(ctx.Root, buildOutDir), ctx.Config.LangDir)
	// Both are fixed by building again: a user error, not a system one.
	if errors.Is(err, publish.ErrNotBuilt) || errors.Is(err, publish.ErrDirtyRules) {
		return cli.UserError(err.Error())
	}
	if err != nil {
		return err
	}
	for _, w := range res.HistoryWarnings {
		fmt.Fprintf(os.Stderr, "lct: warning: history: %s\n", w)
	}
	return ctx.Emit(res)
}
