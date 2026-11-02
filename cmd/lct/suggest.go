package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/adjust"
	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/plan"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/suggest"
)

func init() {
	cli.Register(cli.Command{
		Name: "suggest",
		Usage: "list what the rules could learn from the last build and plan of the current snapshot, with stable ids; " +
			"skips rules/declined.yaml and the ids in each exclude file (one id a line; blank lines and # lines are ignored): " +
			"[--exclude-file F]...",
		Run: runSuggest,
	})
}

// suggestReport is what "suggest" prints.
type suggestReport struct {
	Suggestions []suggest.Suggestion `json:"suggestions"`
	// Excluded counts the suggestions that were left out because their id is
	// declined or in an exclude file.
	Excluded int `json:"excluded"`
}

func (r suggestReport) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d suggestion(s), %d excluded\n", len(r.Suggestions), r.Excluded)
	for _, s := range r.Suggestions {
		fmt.Fprintf(&b, "\n%s\n  %s\n", s.ID, s.Title)
		for _, e := range s.Evidence {
			var parts []string
			if e.Key != "" {
				parts = append(parts, e.Key)
			}
			if e.KR != "" {
				parts = append(parts, "KR: "+e.KR)
			}
			if e.ZH != "" {
				parts = append(parts, "ZH: "+e.ZH)
			}
			if e.Count > 0 {
				parts = append(parts, fmt.Sprintf("x%d", e.Count))
			}
			fmt.Fprintf(&b, "    %s\n", strings.Join(parts, " | "))
		}
	}
	return b.String()
}

// runSuggest finds the suggestions for the rules (spec 11.2). Having
// suggestions is the result and not a finding, so the exit code is 0.
func runSuggest(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("suggest", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var excludeFiles stringList
	flags.Var(&excludeFiles, "exclude-file", "a file of suggestion ids to leave out, one a line (repeatable)")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("suggest: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("suggest: unexpected argument " + flags.Arg(0))
	}
	skip := map[string]bool{}
	for _, f := range excludeFiles {
		ids, err := readExcludeFile(f)
		if err != nil {
			return cli.UserError(fmt.Sprintf("suggest: --exclude-file %s", err))
		}
		for _, id := range ids {
			skip[id] = true
		}
	}

	cfg := ctx.Config
	if cfg.Worktrees.Raw == "" || cfg.Worktrees.TW == "" {
		return cli.UserError("suggest: worktrees.raw and worktrees.tw must be set in lct.yaml")
	}
	src, err := raw.LastSource(filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.Raw)))
	if err != nil {
		return err
	}
	if src == nil {
		return cli.UserError(fmt.Sprintf("suggest: %s has no snapshot; run lct snapshot first", cfg.Worktrees.Raw))
	}

	r, err := rules.Load(ctx.Root)
	if err != nil {
		return err
	}
	adjustments, err := adjust.Load(ctx.Root)
	if err != nil {
		return err
	}
	declined, err := suggest.LoadDeclined(ctx.Root)
	if err != nil {
		return err
	}
	for _, d := range declined {
		skip[d.ID] = true
	}

	// The build report and the plan are those of the tw worktree; suggest
	// uses them only if they belong to the current snapshot. A worktree that
	// is not there is an error, but one that has not been built or planned
	// yet only gives less to suggest.
	st, err := openTWStore(ctx, "suggest")
	if err != nil {
		return err
	}
	twRoot := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.TW))
	in := suggest.Inputs{RawTag: src.Tag, Rules: r, Adjustments: adjustments}
	switch info, err := build.ReadInfo(twRoot); {
	case err == nil:
		in.Build = &info
	case !errors.Is(err, build.ErrNotBuilt):
		return err
	}
	var sum plan.Summary
	if found, err := st.ReadMetaJSON(plan.File, &sum); err != nil {
		return err
	} else if found {
		in.Plan = &sum
	}

	all, err := suggest.Run(in)
	if err != nil {
		return err
	}
	kept, excluded := suggest.Exclude(all, skip)
	return ctx.Emit(suggestReport{Suggestions: kept, Excluded: excluded})
}

// utf8BOM is what some editors put at the start of a text file.
const utf8BOM = "\xef\xbb\xbf"

// readExcludeFile reads the ids of an exclude file: one a line, the blanks
// around it dropped; blank lines and lines that start with # are ignored. The
// error says which file and why, and is for a person.
func readExcludeFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		var pe *fs.PathError
		if errors.As(err, &pe) {
			err = pe.Err // the path is said once
		}
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	var ids []string
	for _, line := range strings.Split(strings.TrimPrefix(string(data), utf8BOM), "\n") {
		if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") {
			ids = append(ids, line)
		}
	}
	return ids, nil
}
