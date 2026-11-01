package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/batch"
	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// defaultSearchLimit is the number of units "search" returns at most.
const defaultSearchLimit = 200

func init() {
	cli.Register(cli.Command{
		Name: "search",
		Usage: "find units in the current KR and their translations; all conditions must hold: " +
			"[--key K]... [--text RE] [--speaker S] [--file GLOB]... [--origin O]... [--limit N] (default 200)",
		Run: runSearch,
	})
}

// searchHit is one unit found: the unit of the current KR text with the
// translation its record holds. ZH and Origin are empty for a unit that has
// no record.
type searchHit struct {
	Key   string `json:"key"`
	Field string `json:"field"`
	// Speaker is batch.SpeakerOf: the character id when the rules know the
	// speaker, else the raw model or speaker string, possibly empty.
	Speaker string `json:"speaker"`
	KR      string `json:"kr"`
	ZH      string `json:"zh"`
	Origin  string `json:"origin"`
}

// searchText is what "search" prints as text: a block of three lines for each
// unit, and a note when the limit cut the list.
type searchText struct {
	hits      []searchHit
	limit     int
	truncated bool
}

func (r searchText) String() string {
	if len(r.hits) == 0 {
		return "no units found"
	}
	var b strings.Builder
	for _, h := range r.hits {
		fmt.Fprintf(&b, "%s\tspeaker=%s\torigin=%s\n  kr: %s\n  zh: %s\n", h.Key, h.Speaker, h.Origin, oneLine(h.KR), oneLine(h.ZH))
	}
	fmt.Fprintf(&b, "%d units", len(r.hits))
	if r.truncated {
		fmt.Fprintf(&b, " (cut at --limit %d: narrow the search or raise --limit)", r.limit)
	}
	return b.String()
}

// oneLine writes the line breaks of a text as \n, so that a unit takes the same
// lines wherever its text breaks.
func oneLine(s string) string {
	return strings.NewReplacer("\r", `\r`, "\n", `\n`).Replace(s)
}

// searchQuery is the conditions of a search. A unit is found when it meets
// every condition given.
type searchQuery struct {
	keys    map[string]bool // unit keys; nil when not given
	files   []string        // patterns (config.MatchAny); nil when not given
	origins []string        // origins of the record; nil when not given: any but copy
	text    *regexp.Regexp  // against the KR text or the translation; nil when not given
	speaker *string         // against batch.SpeakerOf; nil when not given
}

func runSearch(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("search", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var keys, files, origins stringList
	flags.Var(&keys, "key", "a unit key, <file>|<JSON pointer> (repeatable: any of them)")
	text := flags.String("text", "", "a Go regular expression that the KR text or the translation matches, such as '(?i)manager' or '^您'")
	speaker := flags.String("speaker", "", "who says the unit: a character id such as honglu when the rules know the speaker, else the model or speaker string of KR")
	flags.Var(&files, "file", "a pack file name or a glob such as StoryData/S1* or BgmLyrics/** (repeatable: any of them)")
	flags.Var(&origins, "origin", "only units whose record has this origin: paratranz, claude, human, copy or fallback (repeatable: any of them; default: every unit but the copies, which are identifiers and texts that are the same in every language; a unit without a record has no origin)")
	limit := flags.Int("limit", defaultSearchLimit, "the most units to return")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("search: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("search: unexpected argument " + flags.Arg(0))
	}
	given := map[string]bool{}
	flags.Visit(func(f *flag.Flag) { given[f.Name] = true })
	if *limit < 1 {
		return cli.UserError(fmt.Sprintf("search: --limit %d: want at least 1", *limit))
	}

	q := searchQuery{files: files, origins: origins}
	for _, k := range keys {
		if _, _, err := units.SplitKey(k); err != nil {
			return cli.UserError("search: --key: " + err.Error())
		}
		if q.keys == nil {
			q.keys = map[string]bool{}
		}
		q.keys[k] = true
	}
	for _, p := range files {
		if err := config.ValidPattern(p); err != nil {
			return cli.UserError(fmt.Sprintf("search: --file %q: %v", p, err))
		}
	}
	for _, o := range origins {
		if !slices.Contains(store.Origins, o) {
			return cli.UserError(fmt.Sprintf("search: unknown origin %q (want one of %s)", o, strings.Join(store.Origins, ", ")))
		}
	}
	if given["text"] {
		re, err := regexp.Compile(*text)
		if err != nil {
			return cli.UserError(fmt.Sprintf("search: --text: %v", err))
		}
		q.text = re
	}
	if given["speaker"] {
		q.speaker = speaker
	}

	cfg := ctx.Config
	if cfg.Worktrees.Raw == "" || cfg.Worktrees.TW == "" {
		return cli.UserError("search: worktrees.raw and worktrees.tw must be set in lct.yaml")
	}
	rawWorktree := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.Raw))
	src, err := raw.LastSource(rawWorktree)
	if err != nil {
		return err
	}
	if src == nil {
		return cli.UserError(fmt.Sprintf("search: %s has no snapshot; run lct snapshot first", cfg.Worktrees.Raw))
	}
	set, err := pack.OpenRaw(rawWorktree)
	if err != nil {
		return err
	}
	records, err := openStore(ctx).LoadRecords()
	if err != nil {
		return err
	}
	r, err := rules.Load(ctx.Root)
	if err != nil {
		return err
	}

	// One more than the limit is looked for, to know whether the list was cut.
	hits, err := searchUnits(set.KR, r, records, q, *limit+1)
	if err != nil {
		return err
	}
	truncated := len(hits) > *limit
	if truncated {
		hits = hits[:*limit]
	}
	if ctx.JSON {
		return ctx.Emit(hits)
	}
	return ctx.Emit(searchText{hits: hits, limit: *limit, truncated: truncated})
}

// searchUnits returns the units of kr that meet q, at most n of them, in the
// order of the files (sorted by name) and of the documents. records are the
// translations by unit key. The result is never nil.
func searchUnits(kr *pack.Pack, r *rules.Rules, records map[string]store.Record, q searchQuery, n int) ([]searchHit, error) {
	hits := []searchHit{}
	var keyFiles map[string]bool // the files that hold the keys asked for
	if q.keys != nil {
		keyFiles = map[string]bool{}
		for k := range q.keys {
			file, _, _ := units.SplitKey(k) // runSearch checked the key
			keyFiles[file] = true
		}
	}
	for _, file := range kr.Files() {
		if (q.files != nil && !config.MatchAny(q.files, file)) || (keyFiles != nil && !keyFiles[file]) {
			continue
		}
		doc, err := kr.Doc(file)
		if err != nil {
			return nil, fmt.Errorf("search: %w", err)
		}
		for _, u := range units.Extract(file, doc) {
			key := units.Key(file, u.Pointer)
			if q.keys != nil && !q.keys[key] {
				continue
			}
			rec, hasRecord := records[key]
			if q.origins == nil && hasRecord && rec.Origin == store.OriginCopy {
				continue // the model and id fields of every line: nothing to find in them
			}
			if q.origins != nil && (!hasRecord || !slices.Contains(q.origins, rec.Origin)) {
				continue
			}
			if q.text != nil && !q.text.MatchString(u.KR) && !(hasRecord && q.text.MatchString(rec.ZH)) {
				continue
			}
			// The speaker costs more than the other conditions: it is looked up
			// for the units that passed them.
			who := batch.SpeakerOf(r, doc, u)
			if q.speaker != nil && who != *q.speaker {
				continue
			}
			hits = append(hits, searchHit{Key: key, Field: u.Field, Speaker: who, KR: u.KR, ZH: rec.ZH, Origin: rec.Origin})
			if len(hits) >= n {
				return hits, nil
			}
		}
	}
	return hits, nil
}
