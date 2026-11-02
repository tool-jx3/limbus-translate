package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/adjust"
	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/plan"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

func init() {
	cli.Register(cli.Command{
		Name: "adjust add",
		Usage: "check a person's translation of one unit and write it to rules/adjustments/<date>-<topic>.yaml: " +
			"--key K --zh Z --reason R --rule PATH|one-off --topic T [--date YYYY-MM-DD]",
		Run: runAdjustAdd,
	})
	cli.Register(cli.Command{
		Name: "adjust tidy",
		Usage: "bring rules/adjustments up to date with the plan of the current snapshot: " +
			"give a moved adjustment the key of its unit now, and delete an expired one whose unit was translated again",
		Run: runAdjustTidy,
	})
}

// ruleFileNow is the clock of the commands that name a file under rules/ after
// the day ("adjust add", "requeue --to-file"); the tests replace it.
var ruleFileNow = time.Now

// adjustAdded is what "adjust add" prints when the adjustment is written.
type adjustAdded struct {
	// Path is the adjustment file, relative to the repo root.
	Path string `json:"path"`
	Key  string `json:"key"`
	// ZH is the text as written: after validate.Normalize.
	ZH string `json:"zh"`
	// Warnings are the issues that did not stop the adjustment.
	Warnings []validate.Issue `json:"warnings,omitempty"`
}

func (r adjustAdded) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "wrote %s\nkey: %s\nzh: %s\n", r.Path, r.Key, r.ZH)
	for _, is := range r.Warnings {
		fmt.Fprintf(&b, "%-4s  %-10s  %s: %s\n", is.Severity, is.Class, is.Code, is.Message)
	}
	return b.String()
}

func runAdjustAdd(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("adjust add", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	key := flags.String("key", "", "the unit, <file>|<JSON pointer>")
	zh := flags.String("zh", "", "the translation the unit should have")
	reason := flags.String("reason", "", "why this translation is better")
	rule := flags.String("rule", "", "the file in the repo that holds the rule behind the adjustment, such as guides/characters/honglu.md, or one-off")
	topic := flags.String("topic", "", "names the file: rules/adjustments/<date>-<topic>.yaml")
	date := flags.String("date", "", "YYYY-MM-DD, the day of the file name (default: today)")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("adjust add: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("adjust add: unexpected argument " + flags.Arg(0))
	}
	for _, f := range []struct{ name, value string }{{"key", *key}, {"zh", *zh}, {"reason", *reason}, {"rule", *rule}, {"topic", *topic}} {
		if strings.TrimSpace(f.value) == "" {
			return cli.UserError(fmt.Sprintf("adjust add: --%s is required; usage: lct adjust add --key K --zh Z --reason R --rule PATH|one-off --topic T [--date YYYY-MM-DD]", f.name))
		}
	}
	if *date == "" {
		*date = ruleFileNow().Format("2006-01-02")
	}
	if err := adjust.ValidateName(*date, *topic); err != nil {
		return cli.UserError("adjust add: " + err.Error())
	}

	// Everything that can be checked without the raw snapshot is checked
	// first, so that a typo does not wait for the checks of the text.
	a := adjust.Adjustment{Key: *key, ZH: *zh, Reason: *reason, Rule: *rule}
	if a.Rule != adjust.OneOff {
		a.Rule = path.Clean(filepath.ToSlash(a.Rule))
	}
	for _, p := range a.Problems(ctx.Root) {
		if p.Field != "raw" && p.Field != "kr_sha" { // known after KR is read
			return cli.UserError("adjust add: " + p.Message)
		}
	}

	src, set, kr, err := adjustKRText(ctx, "adjust add", a.Key)
	if err != nil {
		return err
	}
	r, err := rules.Load(ctx.Root)
	if err != nil {
		return err
	}
	st, err := openTWStore(ctx, "adjust add")
	if err != nil {
		return err
	}
	keywords, err := keywordNames(st, set)
	if err != nil {
		return err
	}

	file, _, _ := units.SplitKey(a.Key) // adjustKRText checked the key
	normalized, issues := prepareAdjustmentText(r, keywords, file, kr, a.ZH)
	if validate.HasHard(issues) {
		if err := ctx.Emit(validateReport{Normalized: normalized, OK: false, Issues: issues}); err != nil {
			return err
		}
		return cli.ErrFindings
	}

	a.ZH = normalized
	a.Raw = src.Tag
	a.KRSha = units.KRSha(kr)
	written, err := adjust.Add(ctx.Root, a, *date, *topic)
	if errors.Is(err, adjust.ErrInvalid) {
		return cli.UserError("adjust add: " + err.Error())
	}
	if err != nil {
		return err
	}
	return ctx.Emit(adjustAdded{Path: written, Key: a.Key, ZH: a.ZH, Warnings: issues})
}

// prepareAdjustmentText makes the text of a new adjustment ready for the
// pack and checks it against the KR text and the glossary terms that text
// contains in file, the file of the unit. This is the only place that does
// it: whatever has to happen to a translation before it is checked, as it
// does for a batch submission (Normalize, then the space after the keyword
// names), goes here. The text is a new one, so content issues are hard.
func prepareAdjustmentText(r *rules.Rules, keywordNames []string, file, kr, zh string) (string, []validate.Issue) {
	zh = validate.NormalizeKeywordSpace(validate.Normalize(kr, zh), keywordNames)
	return zh, validate.Check(kr, zh, validate.Context{Terms: r.ValidateTerms(file, kr), Existing: false, File: file})
}

// adjustKRText returns the raw snapshot the repo is at, its packs and the KR
// text of the unit key in it. A key that names no text of KR is a user error.
func adjustKRText(ctx *cli.Context, command, key string) (*raw.Source, pack.Set, string, error) {
	cfg := ctx.Config
	if cfg.Worktrees.Raw == "" || cfg.Worktrees.TW == "" {
		return nil, pack.Set{}, "", cli.UserError(command + ": worktrees.raw and worktrees.tw must be set in lct.yaml")
	}
	rawWorktree := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.Raw))
	src, err := raw.LastSource(rawWorktree)
	if err != nil {
		return nil, pack.Set{}, "", err
	}
	if src == nil {
		return nil, pack.Set{}, "", cli.UserError(fmt.Sprintf("%s: %s has no snapshot; run lct snapshot first", command, cfg.Worktrees.Raw))
	}
	set, err := pack.OpenRaw(rawWorktree)
	if err != nil {
		return nil, pack.Set{}, "", err
	}

	file, pointer, err := units.SplitKey(key)
	if err != nil {
		return nil, pack.Set{}, "", cli.UserError(command + ": " + err.Error())
	}
	if !set.KR.Has(file) {
		return nil, pack.Set{}, "", cli.UserError(fmt.Sprintf("%s: %s is not a file of the KR text in %s", command, file, src.Tag))
	}
	doc, err := set.KR.Doc(file)
	if err != nil {
		return nil, pack.Set{}, "", err
	}
	for _, u := range units.Extract(file, doc) {
		if u.Pointer == pointer {
			return src, set, u.KR, nil
		}
	}
	return nil, pack.Set{}, "", cli.UserError(fmt.Sprintf("%s: %s has no text at %s in the KR text of %s (is the key from an older version?)", command, file, pointer, src.Tag))
}

// adjustTidied is what "adjust tidy" prints.
type adjustTidied struct {
	// Changed are the adjustment files that were edited or deleted, as
	// rules/adjustments/<name>.
	Changed []string `json:"changed"`
	// Moved counts the adjustments that got the key of their unit now,
	// Removed those that were deleted.
	Moved   int `json:"moved"`
	Removed int `json:"removed"`
	// Note says why nothing was done, when it is for lack of a plan of the
	// current snapshot. The text shows it; with --json it goes to stderr.
	Note string `json:"-"`
}

func (r adjustTidied) String() string {
	var b strings.Builder
	if r.Note != "" {
		fmt.Fprintln(&b, r.Note)
	}
	if len(r.Changed) == 0 {
		b.WriteString("nothing to tidy\n")
		return b.String()
	}
	fmt.Fprintf(&b, "tidied: %d moved, %d removed\n", r.Moved, r.Removed)
	for _, f := range r.Changed {
		fmt.Fprintf(&b, "  %s\n", f)
	}
	return b.String()
}

// runAdjustTidy keeps the adjustment files current after the plan (spec 11.1,
// 14.2): what .meta/plan.json says about the adjustments of the current
// snapshot is applied to the files. It does what the plan found and no more,
// and checks each adjustment against the current KR again, so a plan that is
// older than a change to the files never deletes an adjustment that applies.
func runAdjustTidy(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("adjust tidy", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	if err := flags.Parse(args); err != nil {
		return cli.UserError("adjust tidy: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("adjust tidy: unexpected argument " + flags.Arg(0))
	}
	cfg := ctx.Config
	if cfg.Worktrees.Raw == "" || cfg.Worktrees.TW == "" {
		return cli.UserError("adjust tidy: worktrees.raw and worktrees.tw must be set in lct.yaml")
	}
	rawWorktree := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.Raw))
	src, err := raw.LastSource(rawWorktree)
	if err != nil {
		return err
	}
	if src == nil {
		return cli.UserError(fmt.Sprintf("adjust tidy: %s has no snapshot; run lct snapshot first", cfg.Worktrees.Raw))
	}

	// Only the plan of the current snapshot knows what became of the
	// adjustments against its KR.
	st, err := openTWStore(ctx, "adjust tidy")
	if err != nil {
		return err
	}
	var sum plan.Summary
	found, err := st.ReadMetaJSON(plan.File, &sum)
	if err != nil {
		return err
	}
	report := adjustTidied{Changed: []string{}}
	switch {
	case !found:
		report.Note = "there is no plan yet (.meta/plan.json); run lct plan first"
	case sum.RawTag != src.Tag:
		report.Note = fmt.Sprintf("the plan is of %s, not of the current snapshot %s; run lct plan first", sum.RawTag, src.Tag)
	}
	if report.Note != "" {
		// The JSON has no place for it: whoever reads the JSON, a person or
		// the log of a run, still learns why nothing was tidied.
		if ctx.JSON {
			fmt.Fprintf(os.Stderr, "lct: adjust tidy: %s\n", report.Note)
		}
		return ctx.Emit(report)
	}

	set, err := pack.OpenRaw(rawWorktree)
	if err != nil {
		return err
	}
	before, err := adjust.Load(ctx.Root)
	if err != nil {
		return err
	}
	byKey := map[string]adjust.Adjustment{}
	for _, a := range before {
		byKey[a.Key] = a
	}
	var moves []adjust.Move
	for _, m := range sum.Adjustments.Moved {
		if a, ok := byKey[m.From]; ok {
			if res := adjust.Resolve(a, set.KR); res.Status == adjust.Moved && res.Key == m.To {
				moves = append(moves, adjust.Move{From: m.From, To: m.To})
			}
		}
	}
	var expired []string
	for _, key := range sum.Adjustments.Expired {
		if a, ok := byKey[key]; ok && adjust.Resolve(a, set.KR).Status == adjust.Expired {
			expired = append(expired, key)
		}
	}

	// Whether the unit of an expired adjustment was translated again is found
	// out before anything is written.
	settledKeys := map[string]bool{}
	if len(expired) > 0 {
		settled, err := settledUnits(st, set.KR)
		if err != nil {
			return err
		}
		for _, key := range expired {
			unit := key
			if u, ok := sum.Adjustments.ExpiredUnits[key]; ok {
				unit = u
			}
			if settledKeys[key], err = settled(unit); err != nil {
				return err
			}
		}
	}

	done, err := adjust.TidyReport(ctx.Root, moves, expired, func(key string) bool { return settledKeys[key] })
	if errors.Is(err, adjust.ErrInvalid) {
		return cli.UserError("adjust tidy: " + err.Error())
	}
	if err != nil {
		return err
	}
	if len(done.Changed) > 0 {
		report = adjustTidied{Changed: done.Changed, Moved: done.Moved, Removed: done.Removed}
	}
	return ctx.Emit(report)
}

// settledUnits returns the test of a unit key for lct adjust tidy: the unit
// has a record of the KR text it has now that is not a human one (the record
// of an adjustment is human, and it is the adjustment that is being asked
// about), and it is not waiting in the queue. A unit that KR does not have is
// not settled.
func settledUnits(st *store.Store, kr *pack.Pack) (func(unit string) (bool, error), error) {
	records, err := st.LoadRecords()
	if err != nil {
		return nil, err
	}
	items, err := st.LoadPending()
	if err != nil {
		return nil, err
	}
	queued := map[string]bool{}
	for _, it := range items {
		queued[it.Key] = true
	}
	shas := map[string]map[string]string{} // file -> pointer -> units.KRSha, made as needed
	return func(unit string) (bool, error) {
		rec, ok := records[unit]
		if queued[unit] || !ok || rec.Origin == store.OriginHuman {
			return false, nil
		}
		file, pointer, err := units.SplitKey(unit)
		if err != nil || !kr.Has(file) {
			return false, nil
		}
		if _, ok := shas[file]; !ok {
			doc, err := kr.Doc(file)
			if err != nil {
				return false, err
			}
			shas[file] = map[string]string{}
			for _, u := range units.Extract(file, doc) {
				shas[file][u.Pointer] = units.KRSha(u.KR)
			}
		}
		sha, ok := shas[file][pointer]
		return ok && rec.KRSha == sha, nil
	}, nil
}
