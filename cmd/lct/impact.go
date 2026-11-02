package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/impact"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/rules"
)

func init() {
	cli.Register(cli.Command{
		Name: "impact",
		Usage: "count the translations that the rules of the working tree reject and the rules of a revision do not " +
			"(guides/, glossary/ and rules/ of REV are the old rules): --base REV [--markdown FILE]",
		Run: runImpact,
	})
}

// baseRulesPaths are the directories whose files are the rules rules.Load reads.
var baseRulesPaths = []string{"guides", "glossary", "rules"}

// runImpact is the assessment of a rule change (spec 11.2 and 12): the old
// rules are those of the revision, the new rules those of the working tree,
// and the translations are those of the tw worktree. A change that breaks
// translations is no finding: it is the result, and the exit code is 0.
func runImpact(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("impact", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	base := flags.String("base", "", "the revision whose guides/, glossary/ and rules/ are the old rules, such as origin/main (required)")
	markdown := flags.String("markdown", "", "also write the result as Markdown, for a pull request or a summary page, to this file")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("impact: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("impact: unexpected argument " + flags.Arg(0))
	}
	switch {
	case *base == "":
		return cli.UserError("impact: --base is required: the revision whose rules the working tree is compared with")
	case strings.HasPrefix(*base, "-"):
		return cli.UserError(fmt.Sprintf("impact: --base %s looks like an option, not a revision", *base))
	}
	cfg := ctx.Config
	if cfg.Worktrees.Raw == "" || cfg.Worktrees.TW == "" {
		return cli.UserError("impact: worktrees.raw and worktrees.tw must be set in lct.yaml")
	}

	rawWorktree := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.Raw))
	src, err := raw.LastSource(rawWorktree)
	if err != nil {
		return err
	}
	if src == nil {
		return cli.UserError(fmt.Sprintf("impact: %s has no snapshot; make the worktree of the raw branch (git worktree add %s raw)", cfg.Worktrees.Raw, cfg.Worktrees.Raw))
	}
	set, err := pack.OpenRaw(rawWorktree)
	if err != nil {
		return err
	}
	st, err := openTWStore(ctx, "impact")
	if err != nil {
		return err
	}
	records, err := st.LoadRecords()
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return cli.UserError(fmt.Sprintf("impact: %s has no translations (.meta/units/ is empty)", cfg.Worktrees.TW))
	}

	newRules, err := rules.Load(ctx.Root)
	if err != nil {
		return err
	}
	oldRules, err := loadRulesAt(ctx.Root, *base)
	if err != nil {
		return err
	}

	res, err := impact.Run(impact.Inputs{Old: oldRules, New: newRules, Records: records, KR: set.KR, Frozen: cfg.Frozen})
	if err != nil {
		return err
	}
	if *markdown != "" {
		if err := os.MkdirAll(filepath.Dir(*markdown), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(*markdown, []byte(res.Markdown(*base)), 0o644); err != nil {
			return err
		}
	}
	return ctx.Emit(res)
}

// loadRulesAt loads the rules as they are in the revision rev of the repo at
// root: guides/, glossary/ and rules/ are exported to a folder of their own,
// which is removed again. A revision git does not know, or one that lacks one
// of the folders, is a user error, and so are rules that do not load.
func loadRulesAt(root, rev string) (*rules.Rules, error) {
	dir, err := os.MkdirTemp("", "lct-impact-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	if err := gitx.Archive(root, rev, baseRulesPaths, dir); err != nil {
		var exited *exec.ExitError
		if errors.As(err, &exited) { // git ran and said no
			return nil, cli.UserError(fmt.Sprintf("impact: the rules of --base %s: %v", rev, err))
		}
		return nil, err
	}
	r, err := rules.Load(dir)
	if err != nil {
		return nil, cli.UserError(fmt.Sprintf("impact: the rules of --base %s do not load: %v", rev, err))
	}
	return r, nil
}
