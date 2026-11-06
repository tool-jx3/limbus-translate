package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/batch"
	"github.com/tool-jx3/limbus-translate/internal/bootstrap"
	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
)

func init() {
	cli.Register(cli.Command{
		Name:  "batch preview",
		Usage: "show the batch a unit would get if it were sent alone, and how its translation fares under the submit check now: <key> [--json]",
		Run:   runBatchPreview,
	})
}

// openPreviewEnv gathers what a preview works with, for command: the same
// inputs as the batch commands, read-only, and the patterns of the frozen
// files. lct batch preview and lct site both load it here, so they preview
// the same thing. records are the records of the store if the caller has read
// them already, else nil and they are read here; the records used are
// returned, so that the Previewer need not read them again.
func openPreviewEnv(ctx *cli.Context, command string, records map[string]store.Record) (batch.Env, []string, map[string]store.Record, error) {
	st, err := openTWStore(ctx, command)
	if err != nil {
		return batch.Env{}, nil, nil, err
	}
	cfg := ctx.Config
	if cfg.Worktrees.Raw == "" {
		return batch.Env{}, nil, nil, cli.UserError(command + ": worktrees.raw must be set in lct.yaml")
	}
	rawWorktree := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.Raw))
	src, err := raw.LastSource(rawWorktree)
	if err != nil {
		return batch.Env{}, nil, nil, err
	}
	if src == nil {
		return batch.Env{}, nil, nil, cli.UserError(fmt.Sprintf("%s: %s has no snapshot; run lct snapshot first", command, cfg.Worktrees.Raw))
	}
	set, err := pack.OpenRaw(rawWorktree)
	if err != nil {
		return batch.Env{}, nil, nil, err
	}
	r, err := rules.Load(ctx.Root)
	if err != nil {
		return batch.Env{}, nil, nil, err
	}
	commit, err := rulesCommit(ctx, command)
	if err != nil {
		return batch.Env{}, nil, nil, err
	}
	var foreign []string
	if _, err := st.ReadMetaJSON(bootstrap.ForeignNamesFile, &foreign); err != nil {
		return batch.Env{}, nil, nil, err
	}
	provisional, err := activeProvisional(ctx.Root, st, r)
	if err != nil {
		return batch.Env{}, nil, nil, err
	}
	if records == nil {
		if records, err = st.LoadRecords(); err != nil {
			return batch.Env{}, nil, nil, err
		}
	}
	names, err := batch.KeywordNames(set.KR, records)
	if err != nil {
		return batch.Env{}, nil, nil, err
	}
	return batch.Env{
		Store:        st,
		Src:          set,
		Rules:        r,
		RawTag:       src.Tag,
		RulesCommit:  commit,
		ForeignNames: foreign,
		KeywordNames: names,
		Provisional:  provisional,
		Now:          batchNow(),
	}, cfg.Frozen, records, nil
}

// previewReport is what "batch preview" prints; as JSON it is the preview.
type previewReport struct {
	*batch.UnitPreview
}

func (r previewReport) String() string {
	p := r.UnitPreview
	var sb strings.Builder
	line := func(format string, args ...any) { fmt.Fprintf(&sb, format+"\n", args...) }
	b := p.Batch
	line("preview of %s", p.Pending.Key)
	line("family: %s", b.Family)
	if b.GlossaryCheck != "" {
		line("glossary_check: %s", b.GlossaryCheck)
	}
	line("guides:")
	for _, g := range b.Guides {
		line("  %s", g)
	}
	line("glossary: %d", len(b.Glossary))
	for _, t := range b.Glossary {
		extra := ""
		if t.Check == rules.CheckHard {
			extra = "  [check: hard]"
		}
		if t.Status != "approved" {
			extra += "  [provisional]"
		}
		line("  %s → %s%s", t.KR, t.ZH, extra)
	}
	line("keywords: %d", len(b.Keywords))
	for _, id := range slices.Sorted(maps.Keys(b.Keywords)) {
		line("  %s → %s", id, b.Keywords[id])
	}
	line("context lines: %d", len(b.Context))

	line("pending: %s", p.Pending.Reason)
	if p.Pending.Hint != "" {
		line("  hint: %s", p.Pending.Hint)
	}
	if p.Pending.Draft != "" {
		line("  draft: %s", p.Pending.Draft)
	}
	if !p.State.Pending {
		line("  (not in the pending list: this is what a requeue or the plan would make)")
	}
	s := p.State
	if s.NeedsReview {
		line("state: needs_review (waits for a person, is not handed out now)")
	}
	if s.LeasedTo != "" {
		line("state: leased to %s", s.LeasedTo)
	}
	if s.Frozen {
		line("state: frozen (the file is not sent for translation)")
	}

	if rec := p.Record; rec == nil {
		line("record: none")
	} else {
		same := "differs from the current rules"
		if rec.Rules == s.RulesCommit {
			same = "same as the current rules"
		}
		line("record: origin %s, model %s, batch %s, rules %s (%s)", rec.Origin, dashed(rec.Model), dashed(rec.Batch), dashed(rec.Rules), same)
	}
	for _, n := range p.Notes {
		line("note: %s", n.Note)
	}
	if c := p.Check; c == nil {
		line("check: nothing to check")
	} else {
		line("check: %s", c.Verdict)
		if c.StaleKR {
			line("  the translation was made from an older KR")
		}
		for _, is := range c.Issues {
			line("  %s [%s] %s", is.Code, is.Severity, is.Message)
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

func runBatchPreview(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("batch preview", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	positional, err := parseInterspersed(flags, args)
	if err != nil {
		return cli.UserError("batch preview: " + err.Error())
	}
	switch {
	case len(positional) == 0:
		return cli.UserError("batch preview: missing unit key")
	case len(positional) > 1:
		return cli.UserError("batch preview: unexpected argument " + positional[1])
	}
	env, frozen, records, err := openPreviewEnv(ctx, "batch preview", nil)
	if err != nil {
		return err
	}
	pv, err := batch.NewPreviewerWith(env, frozen, records)
	if err != nil {
		return err
	}
	u, err := pv.Unit(positional[0])
	switch {
	case errors.Is(err, batch.ErrCopy):
		return cli.UserError(fmt.Sprintf("batch preview: %s: the record is a copy (an identifier, or the same in every language), which is not sent for translation", positional[0]))
	case errors.Is(err, batch.ErrNotInKR):
		return cli.UserError(fmt.Sprintf("batch preview: %s: %v", positional[0], err))
	case err != nil:
		return err
	}
	return ctx.Emit(previewReport{UnitPreview: u})
}

// dashed is s, or "-" when s is empty.
func dashed(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
