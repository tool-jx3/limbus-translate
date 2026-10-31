package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/pack"
)

// generatedRules is the file "lct fields learn --write" writes, relative to
// the repo root.
const generatedRules = "rules/fields.yaml"

func init() {
	cli.Register(cli.Command{
		Name:  "fields learn",
		Usage: "relearn the field rules from the raw KR/EN/JP packs [--write]",
		Run:   runFieldsLearn,
	})
}

// learnReport is what "fields learn" prints.
type learnReport struct {
	Learned *fields.Learned `json:"learned"`
	// Existing says whether rules/fields.yaml existed before this run;
	// Changes is what Learned adds to or removes from it.
	Existing bool            `json:"existing"`
	Changes  []fields.Change `json:"changes"`
	Written  bool            `json:"written"`
}

// noUnitsCounted explains an empty result: the packs have no file in common
// with aligned text.
const noUnitsCounted = "the raw packs counted no units (no file in all three languages has aligned text)"

func (r learnReport) String() string {
	var b strings.Builder
	b.Write(r.Learned.YAML())
	b.WriteString("\n")
	if r.Learned.Counted == 0 {
		fmt.Fprintf(&b, "warning: %s, so --write is refused\n", noUnitsCounted)
	}
	switch {
	case r.Written:
		fmt.Fprintf(&b, "wrote %s", generatedRules)
		if r.Existing && len(r.Changes) == 0 {
			b.WriteString(" (no changes)")
		}
		b.WriteString("\n")
	case !r.Existing:
		fmt.Fprintf(&b, "%s does not exist; run with --write to create it\n", generatedRules)
	case len(r.Changes) == 0:
		fmt.Fprintf(&b, "%s is up to date\n", generatedRules)
	default:
		fmt.Fprintf(&b, "%s would change (run with --write to apply):\n", generatedRules)
	}
	for _, c := range r.Changes {
		fmt.Fprintf(&b, "  %s\n", c)
	}
	return b.String()
}

func runFieldsLearn(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("fields learn", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	write := flags.Bool("write", false, "write "+generatedRules)
	if err := flags.Parse(args); err != nil {
		return cli.UserError("fields learn: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("fields learn: unexpected argument " + flags.Arg(0))
	}

	rawRoot := filepath.Join(ctx.Root, filepath.FromSlash(ctx.Config.Worktrees.Raw))
	if _, err := os.Stat(rawRoot); errors.Is(err, os.ErrNotExist) {
		return cli.UserError(fmt.Sprintf("fields learn: no raw worktree at %s", rawRoot))
	}
	set, err := pack.OpenRaw(rawRoot)
	if err != nil {
		return err
	}
	// A pack without files teaches nothing, and writing the empty result
	// would wipe the rules.
	for _, p := range []struct {
		lang string
		pack *pack.Pack
	}{{"KR", set.KR}, {"EN", set.EN}, {"JP", set.JP}} {
		if len(p.pack.Files()) == 0 {
			return cli.UserError(fmt.Sprintf("fields learn: the raw worktree at %s has no %s files", rawRoot, p.lang))
		}
	}

	manual, err := fields.LoadManual(ctx.Root)
	if err != nil {
		return err
	}
	learned, err := fields.Learn(set.KR, set.EN, set.JP)
	if err != nil {
		return err
	}
	learned = learned.Without(manual)

	target := filepath.Join(ctx.Root, filepath.FromSlash(generatedRules))
	report := learnReport{Learned: learned, Changes: []fields.Change{}}
	if _, err := os.Stat(target); err == nil {
		report.Existing = true
		old, err := fields.LoadLearned(ctx.Root)
		if err != nil {
			return err
		}
		report.Changes = learned.Diff(old)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if *write {
		if learned.Counted == 0 {
			return cli.UserError("fields learn: " + noUnitsCounted + "; refusing to overwrite " + generatedRules)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, learned.YAML(), 0o644); err != nil {
			return err
		}
		report.Written = true
	}
	return ctx.Emit(report)
}
