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
		Usage: "compare an official pack of two snapshots of the raw branch as data (the files, the kinds of file, the keys of the JSON, the entries and the tokens " +
			"of the text that only one has, and for KR the lists of the common files of the snapshots; not what the text says): [--from TAG] [--to TAG] [--lang kr|jp|en]; --to defaults to the newest snapshot, " +
			"--from to the one before --to, --lang to kr; for jp and en it also lists what changed in that pack and not in KR, and the reverse; " +
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
	fmt.Fprintf(&b, "%s -> %s (%s)\n", d.From, d.To, strings.ToUpper(d.Lang))
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
	if l := d.Lists; l != nil {
		fmt.Fprintf(&b, "lists: +%d -%d, %d files newly unlisted\n", d.Counts.ListsAdded, d.Counts.ListsRemoved, d.Counts.UnlistedNew)
		for _, c := range l.Changes {
			if len(c.Added) > 0 {
				fmt.Fprintf(&b, "  + %s %s: %s\n", c.File, c.List, strings.Join(c.Added, ", "))
			}
			if len(c.Removed) > 0 {
				fmt.Fprintf(&b, "  - %s %s: %s\n", c.File, c.List, strings.Join(c.Removed, ", "))
			}
		}
		for _, u := range l.Unlisted {
			isNew := ""
			if u.New {
				isNew = ", new"
			}
			fmt.Fprintf(&b, "  unlisted: %s (%d entries%s)\n", u.File, u.Entries, isNew)
		}
	}
	if d.VsKR != nil {
		lang := strings.ToUpper(d.Lang)
		fmt.Fprintf(&b, "differs from KR: %d only in %s, %d only in KR\n", len(d.VsKR.OnlyHere), lang, len(d.VsKR.OnlyOther))
		item := func(pack string, it rawdiff.Item) {
			sign := "+"
			if it.Change == rawdiff.Removed {
				sign = "-"
			}
			fmt.Fprintf(&b, "  %s %s %s %s", pack, sign, it.Kind, it.Name)
			if it.Kind == rawdiff.ItemEntries {
				fmt.Fprintf(&b, " (%d)", it.Count)
			}
			b.WriteString("\n")
		}
		for _, it := range d.VsKR.OnlyHere {
			item(lang, it)
		}
		for _, it := range d.VsKR.OnlyOther {
			item("KR", it)
		}
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func runRawdiff(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("rawdiff", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	from := flags.String("from", "", "the older snapshot tag (default: the one before --to)")
	to := flags.String("to", "", "the newer snapshot tag (default: the newest)")
	lang := flags.String("lang", "kr", "the pack to compare: kr, jp or en")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("rawdiff: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("rawdiff: unexpected argument " + flags.Arg(0))
	}
	if !slices.Contains(rawLangs, *lang) {
		return cli.UserError(fmt.Sprintf("rawdiff: --lang %s: the packs of a snapshot are %s", *lang, strings.Join(rawLangs, ", ")))
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
	d, err := rawDiff(rawRoot, *from, *to, *lang, r.Family)
	if err != nil {
		return err
	}
	if *lang != rawLangs[0] {
		kr, err := rawDiff(rawRoot, *from, *to, rawLangs[0], r.Family)
		if err != nil {
			return err
		}
		vs := rawdiff.Against(d, kr)
		d.VsKR = &vs
	}
	return ctx.Emit(rawdiffOutput{d})
}

// rawLangs are the packs of a snapshot: KR, which the others are compared
// with, first.
var rawLangs = []string{"kr", "jp", "en"}

// rawDiff compares the pack of lang of the snapshots from and to of the raw
// worktree at rawRoot.
func rawDiff(rawRoot, from, to, lang string, familyOf func(string) string) (*rawdiff.Diff, error) {
	old, err := rawSummary(rawRoot, from, lang)
	if err != nil {
		return nil, err
	}
	cur, err := rawSummary(rawRoot, to, lang)
	if err != nil {
		return nil, err
	}
	d := rawdiff.Compare(old, cur, familyOf)
	d.Lang, d.From, d.To = lang, from, to
	return d, nil
}

// rawSummary summarizes the pack of lang of the snapshot tag of the raw
// worktree at rawRoot, which it does not touch; for the KR pack, with the
// lists of the common files of the snapshot, if it has any.
func rawSummary(rawRoot, tag, lang string) (*rawdiff.Summary, error) {
	tmp, err := os.MkdirTemp("", "lct-rawdiff-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	p, err := raw.OpenAt(rawRoot, tag, lang, tmp)
	if err != nil {
		return nil, err
	}
	s, err := rawdiff.Summarize(p)
	if err != nil || lang != rawLangs[0] {
		return s, err
	}
	// The lists of the common files go with the KR pack; a snapshot made
	// before they were fetched has none.
	has, err := gitx.HasPath(rawRoot, "refs/tags/"+tag, raw.CommonDir)
	if err != nil || !has {
		return s, err
	}
	if err := gitx.Archive(rawRoot, "refs/tags/"+tag, []string{raw.CommonDir}, tmp); err != nil {
		return nil, err
	}
	lists, err := rawdiff.ReadLists(filepath.Join(tmp, raw.CommonDir))
	if err != nil {
		return nil, err
	}
	s.SetLists(lists)
	return s, nil
}
