package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/glossary"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/suggest"
)

func init() {
	cli.Register(cli.Command{
		Name: "glossary derive",
		Usage: "list the keyword and speaker names the translation uses that the glossary does not list " +
			"and that are not declined, as candidates for the glossary; writes nothing: [--source keyword|speaker]",
		Run: runGlossaryDerive,
	})
}

// deriveReport is what "glossary derive" prints.
type deriveReport struct {
	Candidates []glossary.Candidate `json:"candidates"`
}

func (r deriveReport) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d candidate(s)\n", len(r.Candidates))
	for _, c := range r.Candidates {
		fmt.Fprintf(&b, "  %s -> %s (%s)\n", c.KR, c.ZH, c.Source)
	}
	return b.String()
}

// runGlossaryDerive lists the glossary candidates (spec 15). Having candidates
// is the result and not a finding, so the exit code is 0.
func runGlossaryDerive(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("glossary derive", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	source := flags.String("source", "", "only the names of this source: "+strings.Join(glossary.Sources, " or "))
	if err := flags.Parse(args); err != nil {
		return cli.UserError("glossary derive: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("glossary derive: unexpected argument " + flags.Arg(0))
	}
	sourceGiven := false
	flags.Visit(func(f *flag.Flag) { sourceGiven = sourceGiven || f.Name == "source" })
	if sourceGiven && !slices.Contains(glossary.Sources, *source) {
		return cli.UserError(fmt.Sprintf("glossary derive: --source %q is not one of %s", *source, strings.Join(glossary.Sources, ", ")))
	}

	cfg := ctx.Config
	if cfg.Worktrees.Raw == "" || cfg.Worktrees.TW == "" {
		return cli.UserError("glossary derive: worktrees.raw and worktrees.tw must be set in lct.yaml")
	}
	rawWorktree := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.Raw))
	src, err := raw.LastSource(rawWorktree)
	if err != nil {
		return err
	}
	if src == nil {
		return cli.UserError(fmt.Sprintf("glossary derive: %s has no snapshot; run lct snapshot first", cfg.Worktrees.Raw))
	}
	set, err := pack.OpenRaw(rawWorktree)
	if err != nil {
		return err
	}
	records, err := openStore(ctx).LoadRecords()
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return cli.UserError(fmt.Sprintf("glossary derive: %s has no translations; make the worktree of the tw branch (git worktree add %s tw)", cfg.Worktrees.TW, cfg.Worktrees.TW))
	}
	r, err := rules.Load(ctx.Root)
	if err != nil {
		return err
	}
	declined, err := suggest.LoadDeclined(ctx.Root)
	if err != nil {
		return err
	}

	found, err := glossary.Derive(glossary.Inputs{KR: set.KR, Records: records, Terms: r.Glossary, Declined: declined})
	if err != nil {
		return err
	}
	if sourceGiven {
		found = slices.DeleteFunc(found, func(c glossary.Candidate) bool { return c.Source != *source })
	}
	return ctx.Emit(deriveReport{Candidates: found})
}
