package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/rawdiff"
	"github.com/tool-jx3/limbus-translate/internal/rules"
)

func init() {
	cli.Register(cli.Command{
		Name: "rawdiff",
		Usage: "compare the KR pack of two snapshots of the raw branch as data (the files, the kinds of file, the keys of the JSON, the entries and the tokens " +
			"of the text that only one has; not what the text says): [--from TAG] [--to TAG]; --to defaults to the newest snapshot, --from to the one before --to; " +
			"reads the tags of the raw worktree and writes nothing",
		Run: runRawdiff,
	})
}

// rawdiffOutput is what "rawdiff" prints: the Diff, as text a list of what
// differs.
type rawdiffOutput struct{ *rawdiff.Diff }

func (o rawdiffOutput) String() string {
	d := o.Diff
	var b strings.Builder
	fmt.Fprintf(&b, "%s -> %s\n", d.From, d.To)
	file := func(sign string, f rawdiff.File) {
		family := f.Family
		if family == "" {
			family = "no family"
		}
		fmt.Fprintf(&b, "  %s %s (%s, %d entries)\n", sign, f.File, family, f.Entries)
	}
	fmt.Fprintf(&b, "files: +%d -%d\n", d.Counts.FilesAdded, d.Counts.FilesRemoved)
	for _, f := range d.FilesAdded {
		file("+", f)
	}
	for _, f := range d.FilesRemoved {
		file("-", f)
	}
	fmt.Fprintf(&b, "groups: +%d -%d\n", d.Counts.NewGroups, d.Counts.GoneGroups)
	group := func(sign string, g rawdiff.Group) {
		fmt.Fprintf(&b, "  %s %s (%d files)\n", sign, g.Group, g.Files)
		for _, k := range g.Keys {
			fmt.Fprintf(&b, "      %s (%s, %d)\n", k.Path, strings.Join(k.Kinds, "/"), k.Count)
		}
	}
	for _, g := range d.NewGroups {
		group("+", g)
	}
	for _, g := range d.GoneGroups {
		group("-", g)
	}
	fmt.Fprintf(&b, "keys: +%d -%d\n", d.Counts.KeysAdded, d.Counts.KeysRemoved)
	key := func(sign, group string, k rawdiff.Key) {
		fmt.Fprintf(&b, "  %s %s: %s (%s, %d; %s %s)\n", sign, group, k.Path, strings.Join(k.Kinds, "/"), k.Count, k.File, k.Pointer)
	}
	for _, g := range d.Keys {
		for _, k := range g.Added {
			key("+", g.Group, k)
		}
		for _, k := range g.Removed {
			key("-", g.Group, k)
		}
	}
	fmt.Fprintf(&b, "entries: +%d -%d in %d files\n", d.Counts.EntriesAdded, d.Counts.EntriesRemoved, len(d.Entries))
	for _, e := range d.Entries {
		fmt.Fprintf(&b, "  %s: %d -> %d (+%d -%d)\n", e.File, e.Before, e.After, len(e.Added), len(e.Removed))
	}
	fmt.Fprintf(&b, "tokens: +%d -%d\n", d.Counts.TokensAdded, d.Counts.TokensRemoved)
	for _, t := range d.Tokens {
		sign := "+"
		if t.Change == rawdiff.Removed {
			sign = "-"
		}
		fmt.Fprintf(&b, "  %s %s %s (%d; %s %s", sign, t.Kind, t.Name, t.Count, t.File, t.Pointer)
		if t.Kind == rawdiff.TokenBracket {
			if len(t.DefinedIn) == 0 {
				b.WriteString("; defined in no group")
			} else {
				b.WriteString("; defined in " + strings.Join(t.DefinedIn, ", "))
			}
		}
		b.WriteString(")\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func runRawdiff(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("rawdiff", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	from := flags.String("from", "", "the older snapshot tag (default: the one before --to)")
	to := flags.String("to", "", "the newer snapshot tag (default: the newest)")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("rawdiff: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("rawdiff: unexpected argument " + flags.Arg(0))
	}
	if ctx.Config.Worktrees.Raw == "" {
		return cli.UserError("rawdiff: worktrees.raw must be set in lct.yaml")
	}
	rawRoot := filepath.Join(ctx.Root, filepath.FromSlash(ctx.Config.Worktrees.Raw))
	names, err := gitx.Tags(rawRoot, raw.TagPrefix)
	if err != nil {
		return err
	}
	tags := raw.SortedTags(names)
	for _, tag := range []string{*from, *to} {
		if tag != "" && !slices.Contains(tags, tag) {
			return cli.UserError(fmt.Sprintf("rawdiff: %s is not a snapshot tag of the raw branch (there are %d, from %s)", tag, len(tags), strings.Join(tags, ", ")))
		}
	}
	if *to == "" {
		if len(tags) == 0 {
			return cli.UserError("rawdiff: the raw branch has no snapshot tag; it takes two snapshots to compare")
		}
		*to = tags[len(tags)-1]
	}
	if *from == "" {
		i := slices.Index(tags, *to)
		if i == 0 {
			return cli.UserError(fmt.Sprintf("rawdiff: %s is the first snapshot; it takes two snapshots to compare", *to))
		}
		*from = tags[i-1]
	}
	r, err := rules.Load(ctx.Root)
	if err != nil {
		return err
	}
	old, err := rawSummary(rawRoot, *from)
	if err != nil {
		return err
	}
	cur, err := rawSummary(rawRoot, *to)
	if err != nil {
		return err
	}
	d := rawdiff.Compare(old, cur, r.Family)
	d.From, d.To = *from, *to
	return ctx.Emit(rawdiffOutput{d})
}

// rawSummary summarizes the KR pack of the snapshot tag of the raw worktree at
// rawRoot, which it does not touch.
func rawSummary(rawRoot, tag string) (*rawdiff.Summary, error) {
	tmp, err := os.MkdirTemp("", "lct-rawdiff-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	kr, err := raw.OpenKRAt(rawRoot, tag, tmp)
	if err != nil {
		return nil, err
	}
	return rawdiff.Summarize(kr)
}
