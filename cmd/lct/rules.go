package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/requeue"
	"github.com/tool-jx3/limbus-translate/internal/rewrite"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/suggest"
)

func init() {
	cli.Register(cli.Command{
		Name:  "rules lint",
		Usage: "check the guides, glossary, routing, adjustment, requeue, rewrite and declined files",
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
}

func (r lintReport) String() string {
	if r.OK {
		return "ok: the rule files are valid\n"
	}
	var b strings.Builder
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
	report := lintReport{Errors: []rules.LintError{}}
	for _, err := range rules.Lint(ctx.Root) {
		var le rules.LintError
		if !errors.As(err, &le) {
			le = rules.LintError{Message: err.Error()}
		}
		report.Errors = append(report.Errors, le)
	}
	// The requeue, rewrite and declined files are checked by their own
	// packages. rules cannot import requeue, whose tests build on rules (an
	// import cycle), nor suggest, which builds on rules.
	for _, is := range slices.Concat(requeue.Lint(ctx.Root), rewrite.Lint(ctx.Root), suggest.LintDeclined(ctx.Root)) {
		report.Errors = append(report.Errors, rules.LintError{File: is.File, Message: is.Message})
	}
	report.OK = len(report.Errors) == 0
	if err := ctx.Emit(report); err != nil {
		return err
	}
	if !report.OK {
		return cli.ErrFindings
	}
	return nil
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
