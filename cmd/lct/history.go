package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/history"
	"github.com/tool-jx3/limbus-translate/internal/publish"
)

func init() {
	cli.Register(cli.Command{
		Name: "history make",
		Usage: "make the update record of a Release (the changes from the Release before it, each with its causes) from the tags and the rules commits, " +
			"and print it, for checking: <tag> [--out FILE]; text prints a summary, --json prints the whole record; --out writes the whole record (JSON) to FILE " +
			"and then stdout carries only the summary (as JSON with --json, so --json --out does not print the record); writes nothing to tw",
		Run: runHistoryMake,
	})
}

// ruleCount is the number of entries that name one rule as a cause.
type ruleCount struct {
	Key   string `json:"key"`
	Units int    `json:"units"`
}

// historyMakeSummary is what "history make" prints in text mode, and in JSON
// mode when --out sends the record to a file.
type historyMakeSummary struct {
	Tag     string         `json:"tag"`
	PrevTag string         `json:"prev_tag"`
	Entries int            `json:"entries"`
	Kinds   map[string]int `json:"kinds"` // entries per cause kind; an entry with several causes counts in each
	Rules   []ruleCount    `json:"rules"` // per rule, the most entries first
	Unknown int            `json:"unknown"`
	Out     string         `json:"out,omitempty"`
}

// summarizeRecord counts the causes of rec. out is the file the record went to,
// "" if none.
func summarizeRecord(rec *history.Record, out string) historyMakeSummary {
	s := historyMakeSummary{Tag: rec.Tag, PrevTag: rec.PrevTag, Entries: len(rec.Entries), Kinds: map[string]int{},
		Rules: []ruleCount{}, Unknown: rec.Unknown, Out: out}
	perRule := map[string]int{}
	for _, e := range rec.Entries {
		kinds, rules := map[string]bool{}, map[string]bool{}
		for _, c := range e.Causes {
			kinds[c.Kind] = true
			if history.IsRuleCause(c.Kind) {
				rules[history.RuleKey(c.Kind, c.ID)] = true
			}
		}
		for k := range kinds {
			s.Kinds[k]++
		}
		for k := range rules {
			perRule[k]++
		}
	}
	for k, n := range perRule {
		s.Rules = append(s.Rules, ruleCount{k, n})
	}
	sort.Slice(s.Rules, func(i, j int) bool {
		if s.Rules[i].Units != s.Rules[j].Units {
			return s.Rules[i].Units > s.Rules[j].Units
		}
		return s.Rules[i].Key < s.Rules[j].Key
	})
	return s
}

func (s historyMakeSummary) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s (after %s): %d entries", s.Tag, s.PrevTag, s.Entries)
	kinds := make([]string, 0, len(s.Kinds))
	for k := range s.Kinds {
		kinds = append(kinds, k)
	}
	sort.Slice(kinds, func(i, j int) bool {
		if s.Kinds[kinds[i]] != s.Kinds[kinds[j]] {
			return s.Kinds[kinds[i]] > s.Kinds[kinds[j]]
		}
		return kinds[i] < kinds[j]
	})
	for _, k := range kinds {
		fmt.Fprintf(&b, "\n  %s: %d", k, s.Kinds[k])
	}
	for _, r := range s.Rules {
		fmt.Fprintf(&b, "\n  rule %s: %d", r.Key, r.Units)
	}
	fmt.Fprintf(&b, "\n  unknown: %d", s.Unknown)
	if s.Out != "" {
		fmt.Fprintf(&b, "\nwrote %s", s.Out)
	}
	return b.String()
}

// runHistoryMake is read-only: it reads the Releases from the tags and the
// rules from the commits of the repo, and writes the record to stdout or to
// --out, never into the tw worktree or a ref.
func runHistoryMake(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("history make", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	outFile := flags.String("out", "", "write the record (JSON) to this file instead of printing it")
	pos, err := parseInterspersed(flags, args)
	if err != nil {
		return cli.UserError("history make: " + err.Error())
	}
	if len(pos) != 1 {
		return cli.UserError("history make: expected one <tag>, a Release such as zh-TW-20261029.1")
	}
	cfg := ctx.Config
	if cfg.Worktrees.Raw == "" || cfg.Worktrees.TW == "" || cfg.LangDir == "" {
		return cli.UserError("history make: worktrees.raw, worktrees.tw and lang_dir must be set in lct.yaml")
	}
	twRoot := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.TW))
	rawRoot := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.Raw))
	rec, err := publish.MakeRecord(ctx.Root, twRoot, rawRoot, cfg.LangDir, pos[0])
	if errors.Is(err, publish.ErrUnknownRelease) || errors.Is(err, publish.ErrNoPreviousRelease) {
		return cli.UserError("history make: " + err.Error())
	}
	if err != nil {
		return err
	}
	if *outFile != "" {
		data, err := json.MarshalIndent(rec, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(*outFile, append(data, '\n'), 0o644); err != nil {
			return err
		}
		return ctx.Emit(summarizeRecord(rec, *outFile))
	}
	if ctx.JSON {
		return ctx.Emit(rec)
	}
	return ctx.Emit(summarizeRecord(rec, ""))
}
