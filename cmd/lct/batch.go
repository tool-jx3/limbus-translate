package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/batch"
	"github.com/tool-jx3/limbus-translate/internal/bootstrap"
	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/existing"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/suggest"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// batchDir is where batches and their translations are exchanged, relative
// to the repo root (it is not under version control).
const batchDir = "work/batches"

// batchStopFile is made by lct translate once a run has used its soft turns:
// while it is there, lct batch next hands out no batch, so the skill ends the
// run once the batch it has is submitted (see translate.Options.SoftTurns).
const batchStopFile = batchDir + "/.stop"

func init() {
	cli.Register(cli.Command{
		Name:  "batch next",
		Usage: "lease a batch of pending units and write work/batches/<id>.json [--family F] [--max-units N]",
		Run:   runBatchNext,
	})
	cli.Register(cli.Command{
		Name:  "batch submit",
		Usage: "check work/batches/<id>.out.json and record what passes: <id> [--model M]",
		Run:   runBatchSubmit,
	})
}

// batchIDRe is the shape of a batch id (batch.Next may append "-2", "-3"...
// to keep it unique); it keeps the id from naming a path.
var batchIDRe = regexp.MustCompile(`^\d{8}T\d{6}-[0-9a-f]{8}(-[1-9][0-9]*)?$`)

// batchNow is the clock of the batch commands; the tests replace it.
var batchNow = time.Now

// openStore opens the store of the tw worktree. It does not look whether the
// worktree is there: a missing one reads as a store with nothing in it.
func openStore(ctx *cli.Context) *store.Store {
	return store.Open(filepath.Join(ctx.Root, filepath.FromSlash(ctx.Config.Worktrees.TW)))
}

// openTWStore opens the store of the tw worktree for command, which reads or
// changes the state kept there, and refuses a worktree that is not there: with
// no .meta/ in it every command would find no translations and no pending units
// and answer as if there were none. The error is a user error that says how to
// make the worktree (lct does not make it; CI and a person do, from the tw
// branch). The commands for looking things up and for working on the rules
// (search, requeue, adjust, review, impact, suggest, glossary derive, batch
// preview) open it here; the other batch commands and pending keep openStore.
func openTWStore(ctx *cli.Context, command string) (*store.Store, error) {
	tw := ctx.Config.Worktrees.TW
	if tw == "" {
		return nil, cli.UserError(command + ": worktrees.tw must be set in lct.yaml")
	}
	st := openStore(ctx)
	has, err := st.HasMeta()
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, cli.UserError(fmt.Sprintf("%s: %s has no .meta/, so there is no worktree of the tw branch there; "+
			"make it (git fetch origin tw, then git worktree add %s tw)", command, tw, tw))
	}
	return st, nil
}

// parseInterspersed parses args, in which flags and positional arguments may
// come in any order ("batch submit ID --model M"), and returns the positional
// arguments.
func parseInterspersed(flags *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := flags.Parse(args); err != nil {
			return nil, err
		}
		if flags.NArg() == 0 {
			return positional, nil
		}
		positional = append(positional, flags.Arg(0))
		args = flags.Args()[1:]
	}
}

// openBatchEnv gathers what the batch package works with: the raw packs and
// their tag, the rules, the store and the foreign names bootstrap found.
func openBatchEnv(ctx *cli.Context, command string) (batch.Env, *rules.Rules, error) {
	cfg := ctx.Config
	if cfg.Worktrees.Raw == "" || cfg.Worktrees.TW == "" {
		return batch.Env{}, nil, cli.UserError(command + ": worktrees.raw and worktrees.tw must be set in lct.yaml")
	}
	env, err := baseEnv(ctx, command, openStore(ctx), nil)
	if err != nil {
		return batch.Env{}, nil, err
	}
	return env, env.Rules, nil
}

// baseEnv is what openBatchEnv and openPreviewEnv share: the env of st, with
// the raw packs and their tag, the rules, the foreign names bootstrap found
// and the provisional terms in force. The caller has checked that
// worktrees.raw is set. set is the raw packs if the caller has opened them
// already (a pack caches the files it parses, so they are opened once); nil
// and they are opened here.
func baseEnv(ctx *cli.Context, command string, st *store.Store, set *pack.Set) (batch.Env, error) {
	cfg := ctx.Config
	rawWorktree := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.Raw))
	src, err := raw.LastSource(rawWorktree)
	if err != nil {
		return batch.Env{}, err
	}
	if src == nil {
		return batch.Env{}, cli.UserError(fmt.Sprintf("%s: %s has no snapshot; run lct snapshot first", command, cfg.Worktrees.Raw))
	}
	if set == nil {
		opened, err := pack.OpenRaw(rawWorktree)
		if err != nil {
			return batch.Env{}, err
		}
		set = &opened
	}
	r, err := rules.Load(ctx.Root)
	if err != nil {
		return batch.Env{}, err
	}
	var foreign []string
	if _, err := st.ReadMetaJSON(bootstrap.ForeignNamesFile, &foreign); err != nil {
		return batch.Env{}, err
	}
	all, err := st.LoadTerms()
	if err != nil {
		return batch.Env{}, err
	}
	declined, err := declinedIDs(ctx.Root)
	if err != nil {
		return batch.Env{}, err
	}
	table, err := existing.LoadTable(st)
	if err != nil {
		return batch.Env{}, err
	}
	sameText, err := existing.LoadSameText(st)
	if err != nil {
		return batch.Env{}, err
	}
	// A provisional term that differs from an existing name brings the name's
	// entry to the batches that contain it (spec 1b).
	boost := map[string]bool{}
	for _, p := range all {
		if p.Old != nil && p.Old.Existing != "" {
			boost[p.KR] = true
		}
	}
	return batch.Env{
		Store:        st,
		Src:          *set,
		Rules:        r,
		RawTag:       src.Tag,
		ForeignNames: foreign,
		Provisional:  batch.ActiveProvisional(all, r, declined),
		Names:        existing.NewIndex(table, r, declined),
		NameBoost:    boost,
		SameText:     sameText,
		Now:          batchNow(),
	}, nil
}

// declinedIDs reads the declined list of root as a set of ids.
func declinedIDs(root string) (map[string]bool, error) {
	list, err := suggest.LoadDeclined(root)
	if err != nil {
		return nil, err
	}
	declined := make(map[string]bool, len(list))
	for _, d := range list {
		declined[d.ID] = true
	}
	return declined, nil
}

// activeProvisional reads the provisional terms of the store and the declined
// list of root and returns the terms in force (see batch.ActiveProvisional).
func activeProvisional(root string, st *store.Store, r *rules.Rules) ([]rules.Term, error) {
	_, terms, err := activeProvisionalRecords(root, st, r)
	return terms, err
}

// activeProvisionalRecords is activeProvisional with the records of the
// suggestions "term-new/<kr>" too: every record, conflicts included and in
// the order of the file, of a KR that is in force or that only a person's
// decline keeps out. lct suggest and the review page list those KR and treat
// the declined ones as they treat any declined suggestion (lct suggest leaves
// them out, the review page flags them). Which KR count is decided by
// batch.ActiveProvisional alone, so the batches and the suggestions never
// disagree.
func activeProvisionalRecords(root string, st *store.Store, r *rules.Rules) ([]store.ProvisionalTerm, []rules.Term, error) {
	all, err := st.LoadTerms()
	if err != nil {
		return nil, nil, err
	}
	declined, err := declinedIDs(root)
	if err != nil {
		return nil, nil, err
	}
	terms := batch.ActiveProvisional(all, r, declined)
	listed := map[string]bool{}
	for _, t := range batch.ActiveProvisional(all, r, nil) {
		listed[t.KR] = true
	}
	records := slices.DeleteFunc(slices.Clone(all), func(p store.ProvisionalTerm) bool { return !listed[p.KR] })
	return records, terms, nil
}

// keywordNames reads the Chinese names of the keywords from the records in
// the store, over the KR keyword files of src (see batch.KeywordNames). They
// are what the commands that take a new translation (batch submit, adjust add)
// put the space after a keyword name with.
func keywordNames(st *store.Store, src pack.Set) ([]string, error) {
	records, err := st.LoadRecords()
	if err != nil {
		return nil, err
	}
	return batch.KeywordNames(src.KR, records)
}

// batchPath is the file of a batch (suffix ".json") or of its translations
// (".out.json").
func batchPath(ctx *cli.Context, id, suffix string) string {
	return filepath.Join(ctx.Root, filepath.FromSlash(batchDir), id+suffix)
}

// nextReport is what "batch next" prints.
type nextReport struct {
	Batch *string `json:"batch"`
	Path  string  `json:"path,omitempty"`
	Units int     `json:"units,omitempty"`
	// Stopped says why no batch was handed out though units may be free: the
	// run of lct translate has used its turns.
	Stopped string `json:"stopped,omitempty"`
}

func (r nextReport) String() string {
	if r.Batch == nil && r.Stopped != "" {
		return "nothing to hand out: " + r.Stopped
	}
	if r.Batch == nil {
		return "nothing to hand out: no pending unit is free"
	}
	return fmt.Sprintf("batch %s: %d units\n%s", *r.Batch, r.Units, r.Path)
}

func runBatchNext(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("batch next", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	family := flags.String("family", "", "only hand out units of this file family")
	maxUnits := flags.Int("max-units", 0, "most units of a batch outside StoryData/ (default: batch.max_units of lct.yaml)")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("batch next: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("batch next: unexpected argument " + flags.Arg(0))
	}
	maxUnitsSet := false
	flags.Visit(func(f *flag.Flag) { maxUnitsSet = maxUnitsSet || f.Name == "max-units" })
	if maxUnitsSet && *maxUnits < 1 {
		return cli.UserError("batch next: --max-units must be at least 1")
	}

	if stopped, err := pathExists(filepath.Join(ctx.Root, filepath.FromSlash(batchStopFile))); err != nil {
		return err
	} else if stopped {
		return ctx.Emit(nextReport{Stopped: "this run of lct translate has used its turns; end the run, and the next run takes the next batch"})
	}
	env, r, err := openBatchEnv(ctx, "batch next")
	if err != nil {
		return err
	}
	if *family != "" && !slices.ContainsFunc(r.Routing.Families, func(f rules.FamilyRule) bool { return f.ID == *family }) {
		return cli.UserError(fmt.Sprintf("batch next: unknown family %q (guides/routing.yaml)", *family))
	}
	cfg := ctx.Config.Batch
	opt := batch.NextOptions{
		Family:        *family,
		MaxUnits:      cfg.MaxUnits,
		StoryMaxUnits: cfg.StoryMaxUnits,
		LeaseTTL:      time.Duration(cfg.LeaseMinutes) * time.Minute,
		// A batch file of an earlier batch, or its translations, must not be
		// overwritten or picked up by a new batch with the same id.
		IDTaken: func(id string) (bool, error) {
			for _, suffix := range []string{".json", ".out.json"} {
				if taken, err := pathExists(batchPath(ctx, id, suffix)); taken || err != nil {
					return taken, err
				}
			}
			return false, nil
		},
	}
	if maxUnitsSet {
		opt.MaxUnits = *maxUnits
	}
	b, err := batch.Next(env, opt)
	if err != nil {
		return err
	}
	if b == nil {
		return ctx.Emit(nextReport{})
	}
	path := batchPath(ctx, b.ID, ".json")
	if err := writeJSONFile(path, b); err != nil {
		return err
	}
	logBatchEvent(ctx, batchEvent{Event: "next", Batch: b.ID, File: b.File, Files: len(b.Files), Units: len(b.Units)})
	return ctx.Emit(nextReport{Batch: &b.ID, Path: path, Units: len(b.Units)})
}

// pathExists reports whether something is at path. An error other than "does
// not exist" is returned: it says nothing about whether the path is in use.
func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	}
	return false, err
}

// writeJSONFile writes v as batch.EncodeFile encodes it. The file appears whole
// or not at all.
func writeJSONFile(path string, v any) error {
	data, err := batch.EncodeFile(v)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp*")
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(f.Name(), 0o644)
	}
	if err == nil {
		err = os.Rename(f.Name(), path)
	}
	if err != nil {
		os.Remove(f.Name())
	}
	return err
}

// readStrictJSON decodes the JSON file at path into v, refusing unknown
// fields and trailing data. A missing or malformed file is a user error: the
// file is what the translator wrote.
func readStrictJSON(ctx *cli.Context, path string, v any) error {
	rel, err := filepath.Rel(ctx.Root, path)
	if err != nil {
		rel = path
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cli.UserError(fmt.Sprintf("batch submit: %s does not exist", filepath.ToSlash(rel)))
	}
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return cli.UserError(fmt.Sprintf("batch submit: %s is not valid: %v", filepath.ToSlash(rel), err))
	}
	if _, err := dec.Token(); err != io.EOF {
		return cli.UserError(fmt.Sprintf("batch submit: %s is not valid: unexpected data after the JSON value", filepath.ToSlash(rel)))
	}
	return nil
}

// codeRulesDirty is the code of the warning that the rules have changes that
// are not committed. It is about the whole submission, so the entry of
// "warnings" that carries it has no key.
const codeRulesDirty = "rules-dirty"

// rulesDirtyWarning is the entry of "warnings" that says the records of the
// submission name rulesCommit, which does not hold the rules they were made
// under.
func rulesDirtyWarning(rulesCommit string) batch.Rejection {
	return batch.Rejection{Issues: []validate.Issue{{
		Code:     codeRulesDirty,
		Severity: validate.SeverityWarn,
		Class:    validate.ClassStyle,
		Message: fmt.Sprintf("the rules have changes that are not committed, so the records of this batch name the rules commit %s, "+
			"which does not hold them; commit %s/ before the next batch", rulesCommit, strings.Join(rulesPaths, "/, ")),
	}}}
}

// submitReport is what "batch submit" prints: the result of Submit.
type submitReport struct {
	Batch string `json:"batch"`
	batch.SubmitResult
	// file and units are of the batch, for the batch log.
	file  string
	units int
}

func (r submitReport) String() string {
	var b strings.Builder
	// The warning about the rules is no unit's: it has no key.
	unitWarnings := slices.DeleteFunc(slices.Clone(r.Warnings), func(w batch.Rejection) bool { return w.Key == "" })
	fmt.Fprintf(&b, "batch %s: %d accepted, %d reused, %d rejected, %d with warnings, %d need review, %d not submitted, %d notes kept\n",
		r.Batch, len(r.Accepted), len(r.Reused), len(r.Rejected), len(unitWarnings), len(r.NeedsReview), len(r.Released), r.Notes)
	if r.Terms > 0 || len(r.TermsSkipped) > 0 {
		fmt.Fprintf(&b, "%d provisional terms kept\n", r.Terms)
	}
	for _, t := range r.TermsSkipped {
		fmt.Fprintf(&b, "  term skipped, %s: %s -> %s (%s)\n", t.Key, t.KR, t.ZH, t.Reason)
	}
	for _, w := range r.Warnings {
		if w.Key == "" {
			for _, is := range w.Issues {
				fmt.Fprintf(&b, "warning, %s: %s\n", is.Code, is.Message)
			}
		}
	}
	list := func(title string, items []batch.Rejection) {
		if len(items) == 0 {
			return
		}
		fmt.Fprintf(&b, "%s:\n", title)
		for _, it := range items {
			fmt.Fprintf(&b, "  %s\n", it.Key)
			for _, is := range it.Issues {
				fmt.Fprintf(&b, "    %s: %s\n", is.Code, is.Message)
			}
		}
	}
	list("rejected", r.Rejected)
	list("warnings", unitWarnings)
	keys := func(title string, items []string) {
		if len(items) == 0 {
			return
		}
		fmt.Fprintf(&b, "%s:\n", title)
		for _, k := range items {
			fmt.Fprintf(&b, "  %s\n", k)
		}
	}
	keys("reused (the same text as an accepted unit, translated once)", r.Reused)
	keys("needs review (the fallback text, or the earlier translation, stands in)", r.NeedsReview)
	keys("not submitted (free again)", r.Released)
	return b.String()
}

func runBatchSubmit(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("batch submit", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	model := flags.String("model", "", "model that translated the batch (default: $LCT_MODEL)")
	positional, err := parseInterspersed(flags, args)
	if err != nil {
		return cli.UserError("batch submit: " + err.Error())
	}
	switch {
	case len(positional) == 0:
		return cli.UserError("batch submit: missing batch id")
	case len(positional) > 1:
		return cli.UserError("batch submit: unexpected argument " + positional[1])
	case !batchIDRe.MatchString(positional[0]):
		return cli.UserError(fmt.Sprintf("batch submit: invalid batch id %q", positional[0]))
	}
	id := positional[0]
	if *model == "" {
		*model = os.Getenv("LCT_MODEL")
	}
	if *model == "" {
		return cli.UserError("batch submit: no model to record: set LCT_MODEL (or give --model)")
	}

	rep, err := submitBatch(ctx, id, *model)
	if err != nil {
		return err
	}
	logBatchEvent(ctx, submitEvent(id, rep.file, rep.units, rep))
	res := &rep.SubmitResult
	if err := ctx.Emit(submitReport{Batch: id, SubmitResult: *res}); err != nil {
		return err
	}
	if len(res.Rejected) > 0 {
		return cli.ErrFindings
	}
	return nil
}

// submitBatch submits work/batches/<id>.out.json as the translations of the
// batch id by model, as lct batch submit does, and returns what it printed.
func submitBatch(ctx *cli.Context, id, model string) (*submitReport, error) {
	var b batch.Batch
	if err := readStrictJSON(ctx, batchPath(ctx, id, ".json"), &b); err != nil {
		return nil, err
	}
	if b.ID != id {
		return nil, cli.UserError(fmt.Sprintf("batch submit: work/batches/%s.json holds another batch (%s)", id, b.ID))
	}
	var out []batch.OutUnit
	if err := readStrictJSON(ctx, batchPath(ctx, id, ".out.json"), &out); err != nil {
		return nil, err
	}

	env, _, err := openBatchEnv(ctx, "batch submit")
	if err != nil {
		return nil, err
	}
	if env.RulesCommit, err = rulesCommit(ctx, "batch submit"); err != nil {
		return nil, err
	}
	// The records are made all the same: a person who runs a batch by hand
	// with rules not yet committed is told, once, and decides.
	var rulesWarning *batch.Rejection
	if rulesDirty(env.RulesCommit) {
		w := rulesDirtyWarning(env.RulesCommit)
		rulesWarning = &w
		fmt.Fprintf(os.Stderr, "lct: warning: batch submit: %s\n", w.Issues[0].Message)
	}
	if env.KeywordNames, err = keywordNames(env.Store, env.Src); err != nil {
		return nil, err
	}
	res, err := batch.Submit(env, &b, out, model, ctx.Config.Batch.MaxAttempts)
	if errors.Is(err, batch.ErrBadKey) {
		// The translator's file is wrong, not the system; nothing was written.
		return nil, cli.UserError(fmt.Sprintf("batch submit: work/batches/%s.out.json: %v", id, err))
	}
	if err != nil {
		return nil, err
	}
	if rulesWarning != nil {
		res.Warnings = append([]batch.Rejection{*rulesWarning}, res.Warnings...)
	}

	return &submitReport{Batch: id, SubmitResult: *res, file: b.File, units: len(b.Units)}, nil
}
