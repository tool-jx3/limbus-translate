package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/adjust"
	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/rules"
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
}

// adjustNow is the clock of "adjust add"; the tests replace it.
var adjustNow = time.Now

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
		*date = adjustNow().Format("2006-01-02")
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

	src, kr, err := adjustKRText(ctx, "adjust add", a.Key)
	if err != nil {
		return err
	}
	r, err := rules.Load(ctx.Root)
	if err != nil {
		return err
	}

	normalized, issues := prepareAdjustmentText(r, kr, a.ZH)
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
// contains. This is the only place that does it: whatever has to happen to a
// translation before it is checked, as it does for a batch submission, goes
// here. The text is a new one, so content issues are hard.
func prepareAdjustmentText(r *rules.Rules, kr, zh string) (string, []validate.Issue) {
	zh = validate.Normalize(kr, zh)
	var terms []validate.Term
	for _, t := range r.GlossaryHits(kr) {
		terms = append(terms, t.ToValidate())
	}
	return zh, validate.Check(kr, zh, validate.Context{Terms: terms, Existing: false})
}

// adjustKRText returns the raw snapshot the repo is at and the KR text of the
// unit key in it. A key that names no text of KR is a user error.
func adjustKRText(ctx *cli.Context, command, key string) (*raw.Source, string, error) {
	cfg := ctx.Config
	if cfg.Worktrees.Raw == "" {
		return nil, "", cli.UserError(command + ": worktrees.raw must be set in lct.yaml")
	}
	rawWorktree := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.Raw))
	src, err := raw.LastSource(rawWorktree)
	if err != nil {
		return nil, "", err
	}
	if src == nil {
		return nil, "", cli.UserError(fmt.Sprintf("%s: %s has no snapshot; run lct snapshot first", command, cfg.Worktrees.Raw))
	}
	set, err := pack.OpenRaw(rawWorktree)
	if err != nil {
		return nil, "", err
	}

	file, pointer, err := units.SplitKey(key)
	if err != nil {
		return nil, "", cli.UserError(command + ": " + err.Error())
	}
	if !set.KR.Has(file) {
		return nil, "", cli.UserError(fmt.Sprintf("%s: %s is not a file of the KR text in %s", command, file, src.Tag))
	}
	doc, err := set.KR.Doc(file)
	if err != nil {
		return nil, "", err
	}
	for _, u := range units.Extract(file, doc) {
		if u.Pointer == pointer {
			return src, u.KR, nil
		}
	}
	return nil, "", cli.UserError(fmt.Sprintf("%s: %s has no text at %s in the KR text of %s (is the key from an older version?)", command, file, pointer, src.Tag))
}
