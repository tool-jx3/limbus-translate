package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/gold"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/requeue"
	"github.com/tool-jx3/limbus-translate/internal/rewrite"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/suggest"
)

func init() {
	cli.Register(cli.Command{
		Name:  "rules lint",
		Usage: "check the guides, glossary, routing, adjustment, requeue, rewrite, declined and accepted files",
		Run:   runRulesLint,
	})
	cli.Register(cli.Command{
		Name:  "guide resolve",
		Usage: "list the guide files for a file and its speakers: --file F [--speaker ID]...",
		Run:   runGuideResolve,
	})
}

// lintReport is what "rules lint" prints.
type lintReport struct {
	OK     bool              `json:"ok"`
	Errors []rules.LintError `json:"errors"`
	// Warnings are findings that do not fail the lint: they are given only
	// when the raw worktree is present.
	Warnings []rules.LintError `json:"warnings"`
}

func (r lintReport) String() string {
	var b strings.Builder
	for _, w := range r.Warnings {
		fmt.Fprintln(&b, "warning: "+w.Error())
	}
	if r.OK {
		b.WriteString("ok: the rule files are valid\n")
		return b.String()
	}
	for _, e := range r.Errors {
		fmt.Fprintln(&b, e.Error())
	}
	fmt.Fprintf(&b, "%d problem(s) in the rule files\n", len(r.Errors))
	return b.String()
}

func runRulesLint(ctx *cli.Context, args []string) error {
	if len(args) > 0 {
		return cli.UserError("rules lint: unexpected argument " + args[0])
	}
	report := lintReport{Errors: []rules.LintError{}, Warnings: []rules.LintError{}}
	for _, err := range rules.Lint(ctx.Root) {
		var le rules.LintError
		if !errors.As(err, &le) {
			le = rules.LintError{Message: err.Error()}
		}
		report.Errors = append(report.Errors, le)
	}
	// The requeue, rewrite, declined and accepted files are checked by their own
	// packages. rules cannot import requeue, whose tests build on rules (an
	// import cycle), nor suggest, which builds on rules.
	for _, is := range slices.Concat(requeue.Lint(ctx.Root), rewrite.Lint(ctx.Root), suggest.LintDeclined(ctx.Root), suggest.LintAccepted(ctx.Root)) {
		report.Errors = append(report.Errors, rules.LintError{File: is.File, Message: is.Message})
	}
	// lct site loads rules/gold.yaml and stops on a file that does not parse.
	if _, err := gold.Load(ctx.Root); err != nil {
		report.Errors = append(report.Errors, rules.LintError{File: gold.File, Message: strings.TrimPrefix(err.Error(), gold.File+": ")})
	}
	report.OK = len(report.Errors) == 0
	krFiles := krPackFiles(ctx.Root)
	report.Warnings = append(report.Warnings, rules.LintScopeWarnings(ctx.Root, krFiles)...)
	report.Warnings = append(report.Warnings, rules.LintRoutingWarnings(ctx.Root, krFiles)...)
	if err := ctx.Emit(report); err != nil {
		return err
	}
	if !report.OK {
		return cli.ErrFindings
	}
	return nil
}

// krPackFiles lists the files of the KR pack in the raw worktree, or nil when
// there is none (no lct.yaml, no worktree, no snapshot): lint then has nothing
// to compare the glossary's file patterns with and says nothing of them.
func krPackFiles(root string) []string {
	cfg, err := config.Load(root)
	if err != nil || cfg.Worktrees.Raw == "" {
		return nil
	}
	worktree := filepath.Join(root, filepath.FromSlash(cfg.Worktrees.Raw))
	if src, err := raw.LastSource(worktree); err != nil || src == nil {
		return nil
	}
	set, err := pack.OpenRaw(worktree)
	if err != nil {
		return nil
	}
	return set.KR.Files()
}

// resolveReport is what "guide resolve" prints.
type resolveReport struct {
	File   string   `json:"file"`
	Family string   `json:"family"`
	Guides []string `json:"guides"`
}

func (r resolveReport) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "file: %s\nfamily: %s\nguides:\n", r.File, r.Family)
	for _, g := range r.Guides {
		fmt.Fprintf(&b, "  %s\n", g)
	}
	return b.String()
}

// stringList is a flag that can be given several times.
type stringList []string

func (l *stringList) String() string     { return strings.Join(*l, ",") }
func (l *stringList) Set(v string) error { *l = append(*l, v); return nil }

func runGuideResolve(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("guide resolve", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	file := flags.String("file", "", "the file, such as StoryData/S1037B.json")
	var speakers stringList
	flags.Var(&speakers, "speaker", "a character id that speaks in the batch (repeatable)")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("guide resolve: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("guide resolve: unexpected argument " + flags.Arg(0))
	}
	if *file == "" {
		return cli.UserError("guide resolve: usage: lct guide resolve --file F [--speaker ID]...")
	}

	r, err := rules.Load(ctx.Root)
	if err != nil {
		return err
	}
	for _, id := range speakers {
		if _, ok := r.CharacterByID(id); !ok {
			return cli.UserError(fmt.Sprintf("guide resolve: unknown speaker %q", id))
		}
	}
	return ctx.Emit(resolveReport{
		File:   *file,
		Family: r.Family(*file),
		Guides: r.GuidesFor(*file, speakers),
	})
}
