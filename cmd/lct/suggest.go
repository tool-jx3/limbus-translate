package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
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
		Usage: "list what the rules could learn from the last build and plan of the current snapshot, with stable ids, " +
			"by kind in the order of --kind below (the biggest inconsistent group first), then by id; " +
			"skips rules/declined.yaml and the ids in each exclude file (one id a line; blank lines and # lines are ignored): " +
			"[--exclude-file F]... [--kind K]... (only these kinds: " + strings.Join(suggest.Kinds, ", ") + ") [--skip-kind K]... (all but these kinds) " +
			"[--limit N] [--offset N] (a page of what is left; total is how many that is)",
		Run: runSuggest,
	})
}

// suggestReport is what "suggest" prints.
type suggestReport struct {
	// Suggestions is the page that --limit and --offset asked for, or all.
	Suggestions []suggest.Suggestion `json:"suggestions"`
	// Excluded counts the suggestions of the kinds asked for that were left
	// out because their id is declined or in an exclude file.
	Excluded int `json:"excluded"`
	// Total is how many suggestions are left after that, before the page is
	// taken.
	Total int `json:"total"`

	// offset and paged are for the text: where the page starts, and whether
	// a page was asked for.
	offset int
	paged  bool
}

func (r suggestReport) String() string {
	var b strings.Builder
	if r.paged {
		fmt.Fprintf(&b, "%d of %d suggestion(s) from offset %d, %d excluded\n", len(r.Suggestions), r.Total, r.offset, r.Excluded)
	} else {
		fmt.Fprintf(&b, "%d suggestion(s), %d excluded\n", len(r.Suggestions), r.Excluded)
	}
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
			if e.Note != "" {
				parts = append(parts, "note: "+strings.Join(strings.Fields(e.Note), " "))
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
	var excludeFiles, kinds, skipKinds stringList
	flags.Var(&excludeFiles, "exclude-file", "a file of suggestion ids to leave out, one a line (repeatable)")
	flags.Var(&kinds, "kind", "only the suggestions of this kind (repeatable): "+strings.Join(suggest.Kinds, ", "))
	flags.Var(&skipKinds, "skip-kind", "leave out the suggestions of this kind (repeatable): "+strings.Join(suggest.Kinds, ", "))
	limit := flags.Int("limit", 0, "the most suggestions to list (at least 1; default all)")
	offset := flags.Int("offset", 0, "the suggestions to skip first (default 0)")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("suggest: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("suggest: unexpected argument " + flags.Arg(0))
	}
	given := map[string]bool{}
	flags.Visit(func(f *flag.Flag) { given[f.Name] = true })
	for _, k := range kinds {
		if !slices.Contains(suggest.Kinds, k) {
			return cli.UserError(fmt.Sprintf("suggest: --kind %q is not one of %s", k, strings.Join(suggest.Kinds, ", ")))
		}
	}
	for _, k := range skipKinds {
		if !slices.Contains(suggest.Kinds, k) {
			return cli.UserError(fmt.Sprintf("suggest: --skip-kind %q is not one of %s", k, strings.Join(suggest.Kinds, ", ")))
		}
	}
	if given["limit"] && *limit < 1 {
		return cli.UserError(fmt.Sprintf("suggest: --limit %d: want at least 1", *limit))
	}
	if *offset < 0 {
		return cli.UserError(fmt.Sprintf("suggest: --offset %d: want 0 or more", *offset))
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
	// The KR listed are decided by batch.ProvisionalKRs: the terms in force
	// (batch.ActiveProvisional) and those not in force because they differ
	// from the old translations (Old) or were declined. suggest lists them for
	// a person to decide on. A declined one comes too and is excluded below,
	// as any declined suggestion is.
	if in.Provisional, _, err = activeProvisionalRecords(ctx.Root, st, r); err != nil {
		return err
	}
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
	if len(kinds) > 0 {
		all = slices.DeleteFunc(all, func(s suggest.Suggestion) bool { return !slices.Contains(kinds, s.Kind) })
	}
	if len(skipKinds) > 0 {
		all = slices.DeleteFunc(all, func(s suggest.Suggestion) bool { return slices.Contains(skipKinds, s.Kind) })
	}
	kept, excluded := suggest.Exclude(all, skip)
	// The page is taken from the list in the order of suggest.Run (by kind,
	// the biggest group first, then by id), after the kinds were chosen and
	// the ids left out; a page past the end is an empty list.
	total := len(kept)
	page := kept[min(*offset, total):]
	if given["limit"] {
		page = page[:min(*limit, len(page))]
	}
	return ctx.Emit(suggestReport{Suggestions: page, Excluded: excluded, Total: total, offset: *offset, paged: given["limit"] || given["offset"]})
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
