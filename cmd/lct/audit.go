package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/audit"
	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/requeue"
	"github.com/tool-jx3/limbus-translate/internal/rulefile"
	"github.com/tool-jx3/limbus-translate/internal/site"
	"github.com/tool-jx3/limbus-translate/internal/translate"
)

func init() {
	cli.Register(cli.Command{
		Name: "audit",
		Usage: "rank the existing translations by quality signals and split them into review waves: [--wave-size 500] [--wave N] [--usd-per-unit X] [--to-file TOPIC]; " +
			"counts units, families and signals per wave and estimates the cost of translating a wave again " +
			"(--usd-per-unit, default cost_usd / (pending_before - pending_after) of .meta/translate.json, null when that cannot be told); " +
			"writes nothing, or with --to-file TOPIC (wave --wave, default 1) rules/requeue/<date>-audit-<topic>.yaml, the request lct requeue --to-file writes",
		Run: runAudit,
	})
}

// auditOutput is what lct audit prints.
type auditOutput struct {
	// Candidates is the number of units with a signal; WaveSize the size asked.
	Candidates int `json:"candidates"`
	WaveSize   int `json:"wave_size"`
	// USDPerUnit is the rate the estimates use; null when it is unknown.
	USDPerUnit *float64     `json:"usd_per_unit"`
	Waves      []audit.Wave `json:"waves"`
	// Written is the request file written by --to-file.
	Written *requeueWritten `json:"written,omitempty"`
}

func (o auditOutput) String() string {
	var b strings.Builder
	rate := "unknown"
	if o.USDPerUnit != nil {
		rate = fmt.Sprintf("%.4f USD per unit", *o.USDPerUnit)
	}
	fmt.Fprintf(&b, "%d units with a signal, waves of %d, %s\n", o.Candidates, o.WaveSize, rate)
	for _, w := range o.Waves {
		est := "cost unknown"
		if w.EstimatedUSD != nil {
			est = fmt.Sprintf("about %.2f USD", *w.EstimatedUSD)
		}
		fmt.Fprintf(&b, "wave %d: %d units, %s\n  families: %s\n  signals: %s\n", w.Number, w.Units, est, counts(w.Families, nil), counts(w.Signals, audit.Signals))
	}
	if o.Written != nil {
		b.WriteString(o.Written.String())
	}
	return b.String()
}

// counts lists m as name=n, in order when given, else by name.
func counts(m map[string]int, order []string) string {
	names := order
	if names == nil {
		for name := range m {
			names = append(names, name)
		}
		slices.Sort(names)
	}
	var parts []string
	for _, name := range names {
		if m[name] > 0 {
			parts = append(parts, fmt.Sprintf("%s=%d", name, m[name]))
		}
	}
	return strings.Join(parts, " ")
}

func runAudit(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("audit", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	size := flags.Int("wave-size", 500, "units per wave")
	wave := flags.Int("wave", 0, "list the keys of this wave (from 1); with --to-file the wave to write (default 1)")
	usd := flags.Float64("usd-per-unit", 0, "cost of one unit translated again, instead of the rate of .meta/translate.json")
	toFile := flags.String("to-file", "", "write the wave to rules/requeue/<date>-audit-<topic>.yaml (TOPIC is the topic) for CI to apply once")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("audit: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("audit: unexpected argument " + flags.Arg(0))
	}
	var given = map[string]bool{}
	flags.Visit(func(f *flag.Flag) { given[f.Name] = true })
	switch {
	case *size < 1:
		return cli.UserError("audit: --wave-size must be at least 1")
	case given["wave"] && *wave < 1:
		return cli.UserError("audit: --wave counts from 1")
	case given["usd-per-unit"] && *usd < 0:
		return cli.UserError("audit: --usd-per-unit cannot be negative")
	case given["to-file"] && strings.TrimSpace(*toFile) == "":
		return cli.UserError("audit: --to-file needs a topic, the name of the request file rules/requeue/<date>-audit-<topic>.yaml")
	}
	if given["to-file"] && !given["wave"] {
		*wave = 1
	}

	ss, err := openSiteSources(ctx, "audit")
	if err != nil {
		return err
	}
	in, err := loadReviewInputs(ctx, "audit", ss.Rules)
	if err != nil {
		return err
	}
	si := site.Inputs{KR: ss.Set.KR, EN: ss.Set.EN, JP: ss.Set.JP, ZH: ss.ZH, Records: ss.Records, Rules: ss.Rules, Review: &in}
	rev, err := site.BuildReview(si, site.OrderFiles(ss.Set.KR.Files(), ss.Rules))
	if err != nil {
		return err
	}
	pending := make(map[string]bool, len(in.Pending))
	for _, p := range in.Pending {
		pending[p.Key] = true
	}
	cands := audit.Candidates(audit.Input{
		Review: rev, Records: ss.Records, Pending: pending, Frozen: ctx.Config.Frozen,
		Family: ss.Rules.Family, Priority: ss.Rules.Priority,
	})

	var rate *float64
	if given["usd-per-unit"] {
		rate = usd
	} else {
		st, err := openTWStore(ctx, "audit")
		if err != nil {
			return err
		}
		var res translate.Result
		found, err := st.ReadMetaJSON(translate.MetaFile, &res)
		if err != nil {
			return err
		}
		if found {
			rate = audit.USDPerUnit(res.CostUSD, res.PendingBefore, res.PendingAfter)
		}
	}

	waves := audit.Waves(cands, *size, rate)
	out := auditOutput{Candidates: len(cands), WaveSize: *size, USDPerUnit: rate, Waves: waves}
	var picked *audit.Wave
	if *wave > 0 {
		if *wave > len(waves) {
			return cli.UserError(fmt.Sprintf("audit: --wave %d: there are %d waves", *wave, len(waves)))
		}
		w := waves[*wave-1]
		picked = &w
	}
	// Only the wave asked for lists its keys.
	for i := range out.Waves {
		if picked == nil || out.Waves[i].Number != picked.Number {
			out.Waves[i].Keys = nil
		}
	}
	if given["to-file"] {
		written, err := requeue.WriteFile(ctx.Root, ruleFileNow().Format("2006-01-02"), "audit-"+*toFile,
			requeue.Selector{Keys: picked.Keys}, picked.Reason())
		if errors.Is(err, requeue.ErrInvalid) {
			return cli.UserError("audit: " + err.Error())
		}
		if err != nil {
			return err
		}
		out.Written = &requeueWritten{Path: written, ID: rulefile.ID(path.Base(written))}
	}
	return ctx.Emit(out)
}
