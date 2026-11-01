package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/fetch"
	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/raw"
)

// rawBranch is the orphan branch that keeps the original text.
const rawBranch = "raw"

func init() {
	cli.Register(cli.Command{
		Name:  "snapshot",
		Usage: "commit and tag the fetched packs on the raw branch --from DIR --token T --created-at S",
		Run:   runSnapshot,
	})
}

// snapshotResult is what "snapshot" prints.
type snapshotResult struct {
	Tag     string `json:"tag"`
	Commit  string `json:"commit"`
	Changed bool   `json:"changed"`
}

func (r snapshotResult) String() string {
	if r.Changed {
		return fmt.Sprintf("snapshot %s at %s", r.Tag, r.Commit)
	}
	return fmt.Sprintf("unchanged: %s is already the latest snapshot (%s)", r.Tag, r.Commit)
}

func runSnapshot(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("snapshot", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	from := flags.String("from", "", "directory holding kr/, en/ and jp/ (the output of lct fetch)")
	token := flags.String("token", "", "CDN token of the game update")
	createdAt := flags.String("created-at", "", "creation time of the token, such as \"2026-10-29 02:55:23\"")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("snapshot: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("snapshot: unexpected argument " + flags.Arg(0))
	}
	for _, f := range []struct{ name, value string }{{"--from", *from}, {"--token", *token}, {"--created-at", *createdAt}} {
		if f.value == "" {
			return cli.UserError("snapshot: " + f.name + " is required")
		}
	}
	if err := fetch.CheckToken(*token); err != nil {
		return cli.UserError("snapshot: " + err.Error())
	}
	if ctx.Config.Worktrees.Raw == "" {
		return cli.UserError("snapshot: worktrees.raw is not set in lct.yaml")
	}
	packs := make(map[string]string, len(packLangs))
	for _, lang := range packLangs {
		dir := filepath.Join(*from, lang)
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			return cli.UserError(fmt.Sprintf("snapshot: --from %s has no %s directory", *from, lang))
		}
		packs[lang] = dir
	}

	worktree := filepath.Join(ctx.Root, filepath.FromSlash(ctx.Config.Worktrees.Raw))
	if err := gitx.EnsureWorktree(ctx.Root, rawBranch, worktree); err != nil {
		return err
	}
	tag, changed, err := raw.Snapshot(worktree, packs, raw.Source{Token: *token, CreatedAt: *createdAt})
	if errors.Is(err, raw.ErrInvalidInput) {
		return cli.UserError("snapshot: " + err.Error())
	}
	if err != nil {
		return err
	}
	commit, err := gitx.Head(worktree)
	if err != nil {
		return err
	}
	return ctx.Emit(snapshotResult{Tag: tag, Commit: commit, Changed: changed})
}
