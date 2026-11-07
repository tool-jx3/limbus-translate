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
			"counts units, families and signals per wave (the unit keys are printed with --wave, for that wave, and with --to-file, which writes wave 1 unless --wave says otherwise) " +
			"and estimates the cost of translating a wave again " +
			"(--usd-per-unit, default cost_usd / (pending_before - pending_after) of .meta/translate.json, null when that cannot be told; " +
			"rate_source and rate_units say what the rate rests on); " +
			"writes nothing, or with --to-file TOPIC one rules/requeue/<date>-audit-<topic>-<signals>.yaml for each set of signals in the wave, the requests lct requeue --to-file writes",
		Run: runAudit,
	})
}

// auditOutput is what lct audit prints.
type auditOutput struct {
	// Candidates is the number of units with a signal; WaveSize the size asked.
	Candidates int `json:"candidates"`
	WaveSize   int `json:"wave_size"`
	// USDPerUnit is the rate the estimates use; null when it is unknown.
	USDPerUnit *float64 `json:"usd_per_unit"`
	// RateSource says where the rate comes from: translate.MetaFile or
	// "--usd-per-unit"; RateUnits is the number of units it was told from
	// (pending_before - pending_after), absent for --usd-per-unit.
	RateSource string       `json:"rate_source,omitempty"`
	RateUnits  *int         `json:"rate_units,omitempty"`
	Waves      []audit.Wave `json:"waves"`
	// Written are the request files written by --to-file, one for each set of
	// signals in the wave.
	Written []requeueWritten `json:"written,omitempty"`
}

func (o auditOutput) String() string {
	var b strings.Builder
	rate := "unknown"
	if o.USDPerUnit != nil {
		rate = fmt.Sprintf("%.4f USD per unit (%s", *o.USDPerUnit, o.RateSource)
		if o.RateUnits != nil {
			rate += fmt.Sprintf(", from %d units", *o.RateUnits)
		}
		rate += ")"
	}
	fmt.Fprintf(&b, "%d units with a signal, waves of %d, %s\n", o.Candidates, o.WaveSize, rate)
	for _, w := range o.Waves {
		est := "cost unknown"
		if w.EstimatedUSD != nil {
			est = fmt.Sprintf("about %.2f USD", *w.EstimatedUSD)
		}
		fmt.Fprintf(&b, "wave %d: %d units, %s\n  families: %s\n  signals: %s\n", w.Number, w.Units, est, counts(w.Families, nil), counts(w.Signals, audit.Signals))
	}
	for _, w := range o.Written {
		b.WriteString(w.String())
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
	var rateSource string
	var rateUnits *int
	if given["usd-per-unit"] {
		rate, rateSource = usd, "--usd-per-unit"
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
			if rate != nil {
				n := res.PendingBefore - res.PendingAfter
				rateSource, rateUnits = ".meta/"+translate.MetaFile, &n
			}
		}
	}

	waves := audit.Waves(cands, *size, rate)
	out := auditOutput{Candidates: len(cands), WaveSize: *size, USDPerUnit: rate, RateSource: rateSource, RateUnits: rateUnits, Waves: waves}
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
		// One request for each set of signals, so that the hint of a request
		// names only what its units have.
		from := (picked.Number - 1) * *size
		for _, set := range audit.Sets(cands[from : from+picked.Units]) {
			written, err := requeue.WriteFile(ctx.Root, ruleFileNow().Format("2006-01-02"), "audit-"+*toFile+"-"+set.Topic(),
				requeue.Selector{Keys: set.Keys}, set.Reason())
			if errors.Is(err, requeue.ErrInvalid) {
				return cli.UserError("audit: " + err.Error())
			}
			if err != nil {
				return err
			}
			out.Written = append(out.Written, requeueWritten{Path: written, ID: rulefile.ID(path.Base(written))})
		}
	}
	return ctx.Emit(out)
}
