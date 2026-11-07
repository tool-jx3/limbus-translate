package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/evidence"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/rules"
)

func init() {
	cli.Register(cli.Command{
		Name: "glossary evidence",
		Usage: "print, for glossary entries, what the official packs say: the names (the JP and EN aligned to units whose whole KR text is the entry's KR, " +
			"grouped by pair, with unit keys and counts), the descriptions of keyword names, and for each candidate JP or EN rendering " +
			"(the entry's jp and en, the most frequent official names and those given) how many aligned units hold the term (n), how many of them hold the candidate (hits), " +
			"the rate and ok (n >= 10 and rate >= 0.6); reads only the raw worktree and the rules, writes nothing: " +
			"(a name is matched against the whole KR text of a unit, trimmed at both ends; n counts only units with non-empty aligned text, tags and spaces aside) " +
			"[--kr K]... (default every approved entry) [--candidates FILE.yaml] (a mapping of KR to {jp: [...], en: [...]})",
		Run: runGlossaryEvidence,
	})
}

// evidenceReport is what "glossary evidence" prints; --json prints the items.
type evidenceReport []evidence.Item

func (r evidenceReport) String() string {
	var b strings.Builder
	for i, it := range r {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "%s -> %s", it.KR, it.ZH)
		if it.JP != "" {
			fmt.Fprintf(&b, "  jp=%s", it.JP)
		}
		if it.EN != "" {
			fmt.Fprintf(&b, "  en=%s", it.EN)
		}
		b.WriteString("\n")
		if len(it.Names) == 0 {
			b.WriteString("  names: none\n")
		}
		for _, n := range it.Names {
			fmt.Fprintf(&b, "  name JP=%q EN=%q x%d: %s\n", n.JP, n.EN, n.Count, strings.Join(n.Keys, " "))
		}
		for _, d := range it.Descs {
			fmt.Fprintf(&b, "  desc %s\n    KR %s\n    JP %s\n    EN %s\n", d.Key, d.KR, d.JP, d.EN)
		}
		for _, c := range it.Checks {
			fmt.Fprintf(&b, "  check %s %q N=%d Hits=%d Rate=%.2f OK=%v\n", c.Lang, c.Candidate, c.N, c.Hits, c.Rate, c.OK)
		}
	}
	return b.String()
}

type krList []string

func (l *krList) String() string     { return strings.Join(*l, ",") }
func (l *krList) Set(v string) error { *l = append(*l, v); return nil }

// runGlossaryEvidence prints the evidence for glossary entries. The evidence
// is the result and not a finding, so the exit code is 0.
func runGlossaryEvidence(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("glossary evidence", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var krs krList
	flags.Var(&krs, "kr", "an entry to give the evidence of (repeatable; default every approved entry)")
	candFile := flags.String("candidates", "", "a YAML file of KR to {jp: [...], en: [...]}: renderings to check as well")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("glossary evidence: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("glossary evidence: unexpected argument " + flags.Arg(0))
	}

	var candidates map[string]evidence.Candidates
	if *candFile != "" {
		data, err := os.ReadFile(*candFile)
		if err != nil {
			return cli.UserError("glossary evidence: --candidates: " + err.Error())
		}
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)
		if err := dec.Decode(&candidates); err != nil && err != io.EOF {
			return cli.UserError(fmt.Sprintf("glossary evidence: --candidates %s: %v", *candFile, err))
		}
	}

	cfg := ctx.Config
	if cfg.Worktrees.Raw == "" {
		return cli.UserError("glossary evidence: worktrees.raw must be set in lct.yaml")
	}
	rawWorktree := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.Raw))
	src, err := raw.LastSource(rawWorktree)
	if err != nil {
		return err
	}
	if src == nil {
		return cli.UserError(fmt.Sprintf("glossary evidence: %s has no snapshot; run lct snapshot first", cfg.Worktrees.Raw))
	}
	set, err := pack.OpenRaw(rawWorktree)
	if err != nil {
		return err
	}
	r, err := rules.Load(ctx.Root)
	if err != nil {
		return err
	}
	items, err := evidence.Collect(r, set, krs, candidates)
	if err != nil {
		if evidence.IsUserError(err) {
			return cli.UserError("glossary: " + err.Error())
		}
		return err
	}
	if ctx.JSON {
		return ctx.Emit(items)
	}
	return ctx.Emit(evidenceReport(items))
}
