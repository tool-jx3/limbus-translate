package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/align"
	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/gold"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// goldNow is the clock of lct gold set (a test replaces it).
var goldNow = time.Now

func init() {
	cli.Register(cli.Command{
		Name: "gold sample",
		Usage: "draw the gold sample into rules/gold.yaml: from each family the units that are no copy and have a record, " +
			"the first N by sha256(seed|key) (all of them when there are fewer); refused when the file holds a judgement, unless --force: " +
			"[--per-family 20] [--seed 20261107] [--force]",
		Run: runGoldSample,
	})
	cli.Register(cli.Command{
		Name: "gold set",
		Usage: "record the judgement of a unit of the gold sample, with the translation now judged and the day (UTC): " +
			"<key> <correct|wrong|unsure> [--note TEXT]",
		Run: runGoldSet,
	})
	cli.Register(cli.Command{
		Name: "gold report",
		Usage: "count the judgements of the gold sample by family and origin: judged, correct, wrong, unsure, " +
			"accuracy (correct/(correct+wrong), null with neither) and stale (the translation is no more the one judged)",
		Run: runGoldReport,
	})
}

// goldSampleOut is what "gold sample" prints.
type goldSampleOut struct {
	Seed      int            `json:"seed"`
	PerFamily int            `json:"per_family"`
	Units     int            `json:"units"`
	Families  map[string]int `json:"families"`
}

func (o goldSampleOut) String() string {
	fams := make([]string, 0, len(o.Families))
	for f := range o.Families {
		fams = append(fams, f)
	}
	sort.Strings(fams)
	var b strings.Builder
	fmt.Fprintf(&b, "wrote %d units to %s (seed %d, up to %d per family)", o.Units, gold.File, o.Seed, o.PerFamily)
	for _, f := range fams {
		fmt.Fprintf(&b, "\n  %s: %d", f, o.Families[f])
	}
	return b.String()
}

func runGoldSample(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("gold sample", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	per := flags.Int("per-family", gold.DefaultPerFamily, "units per family")
	seed := flags.Int("seed", gold.DefaultSeed, "seed of the draw")
	force := flags.Bool("force", false, "overwrite a file that holds judgements")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("gold sample: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("gold sample: unexpected argument " + flags.Arg(0))
	}
	if *per < 1 {
		return cli.UserError("gold sample: --per-family must be at least 1")
	}
	// A file that cannot be read has nothing to lose, and --force replaces it.
	old, err := gold.Load(ctx.Root)
	if err != nil && !*force {
		return cli.UserError(fmt.Sprintf("gold sample: %v; fix it, or use --force to draw a new sample over it", err))
	}
	if n := old.Judged(); n > 0 && !*force {
		return cli.UserError(fmt.Sprintf("gold sample: %s holds %d judgements, which a new draw would lose; use --force to overwrite them", gold.File, n))
	}
	cfg := ctx.Config
	if cfg.Worktrees.Raw == "" || cfg.Worktrees.TW == "" {
		return cli.UserError("gold sample: worktrees.raw and worktrees.tw must be set in lct.yaml")
	}
	st, err := openTWStore(ctx, "gold sample")
	if err != nil {
		return err
	}
	records, err := st.LoadRecords()
	if err != nil {
		return err
	}
	r, err := rules.Load(ctx.Root)
	if err != nil {
		return err
	}
	fieldRules, err := fields.Load(ctx.Root)
	if err != nil {
		return err
	}
	set, err := pack.OpenRaw(filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.Raw)))
	if err != nil {
		return err
	}
	var cands []gold.Candidate
	for _, file := range set.KR.Files() {
		doc, err := set.KR.Doc(file)
		if err != nil {
			return err
		}
		family := r.Family(file)
		aligner := align.NewFileAligner(file, doc, set.EN, set.JP)
		for _, u := range units.Extract(file, doc) {
			key := units.Key(file, u.Pointer)
			if rec, ok := records[key]; ok && rec.Origin != store.OriginCopy {
				// The field rules decide what is translated at all.
				en, jp, err := aligner.Get(u.Pointer)
				if err != nil {
					return err
				}
				if !fieldRules.Classify(u, en, jp).Translate {
					continue
				}
				cands = append(cands, gold.Candidate{Key: key, Family: family, Origin: rec.Origin})
			}
		}
	}
	s := gold.Sample{Seed: *seed, PerFamily: *per, Units: gold.Draw(cands, *seed, *per)}
	if err := gold.Save(ctx.Root, s); err != nil {
		return err
	}
	out := goldSampleOut{Seed: s.Seed, PerFamily: s.PerFamily, Units: len(s.Units), Families: map[string]int{}}
	for _, u := range s.Units {
		out.Families[u.Family]++
	}
	return ctx.Emit(out)
}

// loadGoldSample reads the sample; none yet is a user error.
func loadGoldSample(ctx *cli.Context, cmd string) (gold.Sample, error) {
	s, err := gold.Load(ctx.Root)
	if err != nil {
		return s, err
	}
	if len(s.Units) == 0 {
		return s, cli.UserError(fmt.Sprintf("%s: %s has no sample; run lct gold sample first", cmd, gold.File))
	}
	return s, nil
}

// goldSetOut is what "gold set" prints.
type goldSetOut struct {
	Key      string `json:"key"`
	Verdict  string `json:"verdict"`
	Note     string `json:"note"`
	JudgedZH string `json:"judged_zh"`
	JudgedAt string `json:"judged_at"`
}

func (o goldSetOut) String() string { return fmt.Sprintf("%s: %s", o.Key, o.Verdict) }

func runGoldSet(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("gold set", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	note := flags.String("note", "", "why, for a wrong or unsure unit")
	pos, err := parseInterspersed(flags, args)
	if err != nil {
		return cli.UserError("gold set: " + err.Error())
	}
	if len(pos) != 2 {
		return cli.UserError("gold set: expected <key> <correct|wrong|unsure>")
	}
	key, verdict := pos[0], pos[1]
	if !gold.ValidVerdict(verdict) {
		return cli.UserError(fmt.Sprintf("gold set: verdict %q is not correct, wrong or unsure", verdict))
	}
	s, err := loadGoldSample(ctx, "gold set")
	if err != nil {
		return err
	}
	st, err := openTWStore(ctx, "gold set")
	if err != nil {
		return err
	}
	records, err := st.LoadRecords()
	if err != nil {
		return err
	}
	if !s.Has(key) {
		return cli.UserError(fmt.Sprintf("gold set: %s is not in the gold sample", key))
	}
	rec, ok := records[key]
	if !ok {
		return cli.UserError(fmt.Sprintf("gold set: %s has no record, so there is no translation to judge", key))
	}
	if err := s.Set(key, verdict, strings.TrimSpace(*note), rec.ZH, goldNow()); err != nil {
		return cli.UserError("gold set: " + err.Error())
	}
	if err := gold.Save(ctx.Root, s); err != nil {
		return err
	}
	for _, u := range s.Units {
		if u.Key == key {
			return ctx.Emit(goldSetOut{Key: key, Verdict: u.Verdict, Note: u.Note, JudgedZH: u.JudgedZH, JudgedAt: u.JudgedAt})
		}
	}
	return nil
}

func runGoldReport(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("gold report", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	if err := flags.Parse(args); err != nil {
		return cli.UserError("gold report: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("gold report: unexpected argument " + flags.Arg(0))
	}
	s, err := loadGoldSample(ctx, "gold report")
	if err != nil {
		return err
	}
	st, err := openTWStore(ctx, "gold report")
	if err != nil {
		return err
	}
	records, err := st.LoadRecords()
	if err != nil {
		return err
	}
	current := make(map[string]string, len(s.Units))
	for _, u := range s.Units {
		current[u.Key] = records[u.Key].ZH
	}
	set, err := pack.OpenRaw(filepath.Join(ctx.Root, filepath.FromSlash(ctx.Config.Worktrees.Raw)))
	if err != nil {
		return err
	}
	rep := gold.MakeReport(s, current)
	for _, u := range s.Units {
		file, ptr, err := units.SplitKey(u.Key)
		if err != nil || !set.KR.Has(file) {
			rep.Missing++
			continue
		}
		doc, err := set.KR.Doc(file)
		if err != nil {
			return err
		}
		if _, ok := doc.Get(ptr); !ok {
			rep.Missing++
		}
	}
	return ctx.Emit(rep)
}
