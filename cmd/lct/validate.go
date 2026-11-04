package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

func init() {
	cli.Register(cli.Command{
		Name:  "validate",
		Usage: "normalize and check one translation against the glossary: --kr S --zh S [--file F]",
		Run:   runValidate,
	})
}

// validateReport is what "validate" prints.
type validateReport struct {
	// Normalized is the translation after validate.Normalize; the issues are
	// those of the normalized text.
	Normalized string           `json:"normalized"`
	OK         bool             `json:"ok"` // no hard issue
	Issues     []validate.Issue `json:"issues"`
}

func (r validateReport) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "normalized: %s\n", r.Normalized)
	if len(r.Issues) == 0 {
		b.WriteString("ok: no issues\n")
	}
	for _, is := range r.Issues {
		fmt.Fprintf(&b, "%-4s  %-10s  %s: %s\n", is.Severity, is.Class, is.Code, is.Message)
	}
	if len(r.Issues) > 0 {
		if r.OK {
			b.WriteString("ok: warnings only\n")
		} else {
			b.WriteString("failed: hard issues\n")
		}
	}
	return b.String()
}

func runValidate(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	kr := flags.String("kr", "", "the Korean text")
	zh := flags.String("zh", "", "the translation")
	file := flags.String("file", "", "the pack file the text is from, such as StoryData/S1001B.json: glossary terms that apply to some files only are enforced when it matches, and half-width brackets are reported in the labels of RPGSystem/")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("validate: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("validate: unexpected argument " + flags.Arg(0))
	}
	given := map[string]bool{}
	flags.Visit(func(f *flag.Flag) { given[f.Name] = true })
	if !given["kr"] || !given["zh"] {
		return cli.UserError("validate: usage: lct validate --kr TEXT --zh TEXT [--file FILE]")
	}

	// The glossary terms the KR text contains are enforced on the translation.
	r, err := rules.Load(ctx.Root)
	if err != nil {
		return err
	}
	normalized := validate.Normalize(*kr, *zh)
	issues := validate.Check(*kr, normalized, validate.Context{Terms: r.ValidateTerms(*file, *kr), File: *file, ScriptOK: r.ScriptRenderings(*file)})
	if issues == nil {
		issues = []validate.Issue{}
	}
	hard := validate.HasHard(issues)
	if err := ctx.Emit(validateReport{Normalized: normalized, OK: !hard, Issues: issues}); err != nil {
		return err
	}
	if hard {
		return cli.ErrFindings
	}
	return nil
}
