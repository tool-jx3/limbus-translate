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
	"github.com/tool-jx3/limbus-translate/internal/scope"
	"github.com/tool-jx3/limbus-translate/internal/suggest"
)

func init() {
	cli.Register(cli.Command{
		Name: "glossary derive",
		Usage: "list the keyword and speaker names the translation uses that the glossary does not list " +
			"and that are not declined, as candidates for the glossary; writes nothing: [--source keyword|speaker] " +
			"[--limit N] [--offset N] (a page of the sorted list; total is how many there are)",
		Run: runGlossaryDerive,
	})
	cli.Register(cli.Command{
		Name: "glossary scope",
		Usage: "print where the translations use each approved glossary entry and the proposals of where it should apply, " +
			"the review/scope.json of lct site, computed from the raw and tw worktrees and the rules as lct site computes it; writes nothing: " +
			"--json prints thresholds (with hard_rate and hard_min_n, the least rate and number of units for hard_candidate), entries (id glossary/…, kr, zh, jp and en (a string, or a list of forms), also, files, not_files, match, check (hard, off, or absent for warn; an off entry is measured for information only: it is never a hard_candidate and has no proposal), notes, n, used and rate over the units the entry hits, hard_candidate, groups with name, kind (family: or group: <top node of the file tree>), n and used, compound, explained, unexplained, examples: at most 3 units of each group that does not use the rendering) " +
			"and proposals (id scope/…, kind, kr, zh, patterns, match, mixed_groups, warn_removed, warn_added, used_lost, used_gained, flagged, examples, declined when rules/declined.yaml has the id)",
		Run: runGlossaryScope,
	})
}

// scopeOutput is what "glossary scope" prints: the scope report.
type scopeOutput struct{ scope.Report }

func (o scopeOutput) String() string {
	var b strings.Builder
	open, by := 0, map[string]int{}
	for _, p := range o.Proposals {
		by[p.Kind]++
		if p.Declined == nil {
			open++
		}
	}
	var counts []string
	for _, k := range scope.Kinds {
		counts = append(counts, fmt.Sprintf("%s %d", k, by[k]))
	}
	fmt.Fprintf(&b, "%d entries, %d proposals (%d not declined): %s\n", len(o.Entries), len(o.Proposals), open, strings.Join(counts, ", "))
	// An entry of check: off misses nothing: its units are left out.
	var same, other, unknown, off int
	for _, e := range o.Entries {
		if e.Check == rules.CheckOff {
			off++
			continue
		}
		same, other, unknown = same+e.Same, other+e.Other, unknown+e.Unknown
	}
	fmt.Fprintf(&b, "units without the rendering: same sense %d, other sense %d, unknown %d", same, other, unknown)
	switch off {
	case 0:
	case 1:
		b.WriteString(" (1 entry of check: off left out)")
	default:
		fmt.Fprintf(&b, " (%d entries of check: off left out)", off)
	}
	b.WriteString("\n")
	for _, p := range o.Proposals {
		fmt.Fprintf(&b, "  %s  %s", p.ID, p.Title())
		if p.Declined != nil {
			fmt.Fprintf(&b, " [declined %s: %s]", p.Declined.Date, p.Declined.Reason)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// runGlossaryScope prints the scope report of the last build. Proposals are the
// result and not a finding, so the exit code is 0.
func runGlossaryScope(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("glossary scope", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	if err := flags.Parse(args); err != nil {
		return cli.UserError("glossary scope: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("glossary scope: unexpected argument " + flags.Arg(0))
	}
	rev, _, _, err := lastReview(ctx, "glossary scope")
	if err != nil {
		return err
	}
	return ctx.Emit(scopeOutput{rev.Scope})
}

// deriveReport is what "glossary derive" prints. Total is the number of
// candidates there are (of the source asked for), Candidates the page of them
// that --limit and --offset asked for, or all of them without either.
type deriveReport struct {
	Total      int                  `json:"total"`
	Candidates []glossary.Candidate `json:"candidates"`

	// offset and paged are for the text: where the page starts, and whether
	// a page was asked for.
	offset int
	paged  bool
}

func (r deriveReport) String() string {
	var b strings.Builder
	if r.paged {
		fmt.Fprintf(&b, "%d of %d candidate(s), from offset %d\n", len(r.Candidates), r.Total, r.offset)
	} else {
		fmt.Fprintf(&b, "%d candidate(s)\n", len(r.Candidates))
	}
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
	limit := flags.Int("limit", 0, "the most candidates to list (at least 1; default all)")
	offset := flags.Int("offset", 0, "the candidates to skip first (default 0)")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("glossary derive: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("glossary derive: unexpected argument " + flags.Arg(0))
	}
	given := map[string]bool{}
	flags.Visit(func(f *flag.Flag) { given[f.Name] = true })
	sourceGiven := given["source"]
	if sourceGiven && !slices.Contains(glossary.Sources, *source) {
		return cli.UserError(fmt.Sprintf("glossary derive: --source %q is not one of %s", *source, strings.Join(glossary.Sources, ", ")))
	}
	if given["limit"] && *limit < 1 {
		return cli.UserError(fmt.Sprintf("glossary derive: --limit %d: want at least 1", *limit))
	}
	if *offset < 0 {
		return cli.UserError(fmt.Sprintf("glossary derive: --offset %d: want 0 or more", *offset))
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
	st, err := openTWStore(ctx, "glossary derive")
	if err != nil {
		return err
	}
	records, err := st.LoadRecords()
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return cli.UserError(fmt.Sprintf("glossary derive: %s has no translations (.meta/units/ is empty)", cfg.Worktrees.TW))
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
	// The page is taken from the sorted list, after the source was chosen;
	// a page past the end is an empty list.
	total := len(found)
	page := found[min(*offset, total):]
	if given["limit"] {
		page = page[:min(*limit, len(page))]
	}
	return ctx.Emit(deriveReport{Total: total, Candidates: page, offset: *offset, paged: given["limit"] || given["offset"]})
}
