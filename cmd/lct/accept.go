package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/site"
	"github.com/tool-jx3/limbus-translate/internal/suggest"
)

func init() {
	cli.Register(cli.Command{
		Name: "accept",
		Usage: "record the review items a person confirmed, so the review page, lct suggest and the build report skip them: " +
			"--reason R [--date YYYY-MM-DD] [--file F] [ID...] (a rule suggestion or note/… id goes to rules/declined.yaml, a warning/… id to rules/accepted.yaml; " +
			"--file holds ids one a line, blank lines and lines that start with # ignored; the date defaults to today; an id that is not an item of the last build is refused and nothing is written; commits nothing); " +
			"or --tidy: delete the entries of rules/accepted.yaml that are no warning of the last build any more",
		Run: runAccept,
	})
}

// acceptReport is what "accept" prints. Declined and Accepted are the ids
// written to rules/declined.yaml and rules/accepted.yaml, Skipped those that
// were confirmed before this call. RawTag and TWCommit name the build the ids
// were checked against. With --tidy only Removed is set.
type acceptReport struct {
	Declined []string `json:"declined"`
	Accepted []string `json:"accepted"`
	Skipped  []string `json:"skipped"`
	RawTag   string   `json:"raw_tag"`
	TWCommit string   `json:"tw_commit"`
}

func (r acceptReport) String() string {
	return fmt.Sprintf("%d to %s, %d to %s, %d already confirmed (build %s, tw %s)",
		len(r.Declined), suggest.DeclinedFile, len(r.Accepted), suggest.AcceptedFile, len(r.Skipped), r.RawTag, r.TWCommit)
}

// tidyReport is what "accept --tidy" prints.
type tidyReport struct {
	Removed []string `json:"removed"`
}

func (r tidyReport) String() string {
	if len(r.Removed) == 0 {
		return "nothing to remove from " + suggest.AcceptedFile
	}
	return fmt.Sprintf("removed %d from %s:\n  %s", len(r.Removed), suggest.AcceptedFile, strings.Join(r.Removed, "\n  "))
}

func runAccept(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("accept", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	reason := flags.String("reason", "", "why the items are confirmed (required)")
	date := flags.String("date", "", "the date to write, YYYY-MM-DD (default: today)")
	idFile := flags.String("file", "", "a file of ids, one a line")
	tidy := flags.Bool("tidy", false, "delete the entries of "+suggest.AcceptedFile+" that are no warning of the last build")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("accept: " + err.Error())
	}

	if *tidy {
		if flags.NArg() > 0 || *idFile != "" || *reason != "" || *date != "" {
			return cli.UserError("accept: --tidy takes no ids, --file, --reason or --date")
		}
		return acceptTidy(ctx)
	}

	var ids []string
	seen := map[string]bool{}
	add := func(id string) {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, id := range flags.Args() {
		if id = strings.TrimSpace(id); id != "" {
			add(id)
		}
	}
	if *idFile != "" {
		fromFile, err := readExcludeFile(*idFile)
		if err != nil {
			return cli.UserError(fmt.Sprintf("accept: --file %s", err))
		}
		for _, id := range fromFile {
			add(id)
		}
	}
	if len(ids) == 0 {
		return cli.UserError("accept: give at least one ID (as arguments or with --file), or --tidy")
	}
	if strings.TrimSpace(*reason) == "" {
		return cli.UserError("accept: --reason is required and must not be empty")
	}
	day := *date
	if day == "" {
		day = time.Now().Format("2006-01-02")
	} else if t, err := time.Parse("2006-01-02", day); err != nil || t.Format("2006-01-02") != day {
		return cli.UserError(fmt.Sprintf("accept: --date %q: want YYYY-MM-DD", day))
	}

	rev, in, head, err := acceptReview(ctx)
	if err != nil {
		return err
	}
	toDeclined := map[string]bool{}
	for _, s := range rev.Suggestions {
		toDeclined[s.ID] = true
	}
	for _, n := range rev.Notes {
		toDeclined[n.ID] = true
	}
	toAccepted := map[string]bool{}
	for _, w := range rev.Warnings {
		toAccepted[w.ID] = true
	}
	known := map[string]bool{}
	for _, e := range in.Declined {
		known[e.ID] = true
	}
	for _, e := range in.Accepted {
		known[e.ID] = true
	}

	var unknown []string
	rep := acceptReport{Declined: []string{}, Accepted: []string{}, Skipped: []string{}, RawTag: in.RawTag, TWCommit: head}
	var declined, accepted []suggest.Declined
	for _, id := range ids {
		switch {
		case !toDeclined[id] && !toAccepted[id]:
			unknown = append(unknown, id)
		case known[id]:
			rep.Skipped = append(rep.Skipped, id)
		case toDeclined[id]:
			rep.Declined = append(rep.Declined, id)
			declined = append(declined, suggest.Declined{ID: id, Reason: *reason, Date: day})
		default:
			rep.Accepted = append(rep.Accepted, id)
			accepted = append(accepted, suggest.Declined{ID: id, Reason: *reason, Date: day})
		}
	}
	if len(unknown) > 0 {
		return cli.UserError(fmt.Sprintf("accept: %d of the ids are no item of the last build (%s), so nothing was written:\n  %s",
			len(unknown), in.RawTag, strings.Join(unknown, "\n  ")))
	}
	if err := suggest.AppendEntries(ctx.Root, suggest.DeclinedFile, declined); err != nil {
		return err
	}
	if err := suggest.AppendEntries(ctx.Root, suggest.AcceptedFile, accepted); err != nil {
		return err
	}
	return ctx.Emit(rep)
}

// acceptTidy deletes the entries of rules/accepted.yaml that are no warning of
// the last build.
func acceptTidy(ctx *cli.Context) error {
	rev, in, _, err := acceptReview(ctx)
	if err != nil {
		return err
	}
	current := map[string]bool{}
	for _, w := range rev.Warnings {
		current[w.ID] = true
	}
	stale := map[string]bool{}
	for _, e := range in.Accepted {
		if !current[e.ID] {
			stale[e.ID] = true
		}
	}
	removed, err := suggest.RemoveEntries(ctx.Root, suggest.AcceptedFile, stale)
	if err != nil {
		return err
	}
	if removed == nil {
		removed = []string{}
	}
	return ctx.Emit(tidyReport{Removed: removed})
}

// acceptReview computes the review items of the last build, the way lct site
// does, and returns them with their inputs and the commit of the tw worktree.
func acceptReview(ctx *cli.Context) (site.Review, site.ReviewInputs, string, error) {
	ss, err := openSiteSources(ctx, "accept")
	if err != nil {
		return site.Review{}, site.ReviewInputs{}, "", err
	}
	in, err := loadReviewInputs(ctx, "accept")
	if err != nil {
		return site.Review{}, in, "", err
	}
	head, err := gitx.Head(filepath.Join(ctx.Root, filepath.FromSlash(ctx.Config.Worktrees.TW)))
	if err != nil {
		return site.Review{}, in, "", err
	}
	si := site.Inputs{KR: ss.Set.KR, EN: ss.Set.EN, JP: ss.Set.JP, ZH: ss.ZH, Records: ss.Records, Rules: ss.Rules, Review: &in}
	rev, err := site.BuildReview(si, site.OrderFiles(ss.Set.KR.Files(), ss.Rules))
	return rev, in, head, err
}
