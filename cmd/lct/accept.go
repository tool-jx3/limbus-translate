package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/site"
	"github.com/tool-jx3/limbus-translate/internal/suggest"
)

func init() {
	cli.Register(cli.Command{
		Name: "accept",
		Usage: "record the review items a person confirmed, so the review page, lct suggest and the build report skip them: " +
			"--reason R [--date YYYY-MM-DD] [--file F] [ID...] (a rule suggestion or note/… id goes to rules/declined.yaml, a warning/… id to rules/accepted.yaml, a termgroup/… id to rules/accepted.yaml unit by unit (the warning/term-missing/… ids of the units of the group not accepted yet; none left is a no-op it says so); " +
			"--file holds ids one a line, blank lines and lines that start with # ignored, and must be a file of the repo outside .git; the date defaults to today; an id that is not an item of the last build is refused and nothing is written; commits nothing); " +
			"or --dry-run [--file F] ID...: write nothing, print each item as the review page has it (kind, the list it goes to, whether and why it is confirmed, KR, TW, note or message, units, variants) and the ids that are no item of the last build (exit code 1 if any); " +
			"or --tidy: delete the entries of rules/accepted.yaml that are no warning of the last build any more",
		Run: runAccept,
	})
}

// acceptReport is what "accept" prints. Declined and Accepted are the ids
// written to rules/declined.yaml and rules/accepted.yaml, Skipped those that
// were confirmed before this call. RawTag and TWCommit name the build the ids
// were checked against. With --tidy only Removed is set.
type acceptReport struct {
	Declined []string `json:"declined"`
	Accepted []string `json:"accepted"`
	Skipped  []string `json:"skipped"`
	// Empty are the termgroup/… ids whose units are all accepted: nothing to write.
	Empty    []string `json:"empty"`
	RawTag   string   `json:"raw_tag"`
	TWCommit string   `json:"tw_commit"`
}

func (r acceptReport) String() string {
	s := fmt.Sprintf("%d to %s, %d to %s, %d already confirmed (build %s, tw %s)",
		len(r.Declined), suggest.DeclinedFile, len(r.Accepted), suggest.AcceptedFile, len(r.Skipped), r.RawTag, r.TWCommit)
	if len(r.Empty) > 0 {
		s += "\nno unit left to accept in:\n  " + strings.Join(r.Empty, "\n  ")
	}
	return s
}

// tidyReport is what "accept --tidy" prints: the entries removed, and the
// build they were checked against.
type tidyReport struct {
	Removed  []string `json:"removed"`
	RawTag   string   `json:"raw_tag"`
	TWCommit string   `json:"tw_commit"`
}

func (r tidyReport) String() string {
	build := fmt.Sprintf("(build %s, tw %s)", r.RawTag, r.TWCommit)
	if len(r.Removed) == 0 {
		return "nothing to remove from " + suggest.AcceptedFile + " " + build
	}
	return fmt.Sprintf("removed %d from %s %s:\n  %s", len(r.Removed), suggest.AcceptedFile, build, strings.Join(r.Removed, "\n  "))
}

// The values of lookupItem.List.
const (
	listDeclined = "declined"
	listAccepted = "accepted"
)

// lookupItem is one id of "accept --dry-run": the item as the review page has
// it. List is the file the id goes to, "declined" or "accepted"; Confirmed is
// set when it is in that file already. KR and ZH are the texts the item is
// about, Note a translator's note, Message the check's description of a
// warning; Variants are those of a group of inconsistent translations only.
type lookupItem struct {
	ID        string          `json:"id"`
	Kind      string          `json:"kind"`
	List      string          `json:"list"`
	Confirmed *site.Confirmed `json:"confirmed"`
	Title     string          `json:"title,omitempty"`
	KR        string          `json:"kr,omitempty"`
	ZH        string          `json:"zh,omitempty"`
	Note      string          `json:"note,omitempty"`
	Message   string          `json:"message,omitempty"`
	Keys      []string        `json:"keys"`
	Variants  []site.Variant  `json:"variants,omitempty"`
	// JPENClass says whether JP and EN tell the variants apart (site.JPENSame,
	// site.JPENExplained, site.JPENPartial); inconsistent groups only.
	JPENClass string `json:"jpen_class,omitempty"`
	Pending   int    `json:"pending"`
	// Units are the units of a term-missing group (termgroup/…); accepting the
	// group accepts those without Accepted, one id each.
	Units []site.TermUnit `json:"units,omitempty"`
}

// lookupReport is what "accept --dry-run" prints: the items in the order of the
// ids, the ids that are no item of the last build, and that build.
type lookupReport struct {
	Items    []lookupItem `json:"items"`
	Unknown  []string     `json:"unknown"`
	RawTag   string       `json:"raw_tag"`
	TWCommit string       `json:"tw_commit"`
}

func (r lookupReport) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d items, %d unknown (build %s, tw %s)\n", len(r.Items), len(r.Unknown), r.RawTag, r.TWCommit)
	for _, it := range r.Items {
		fmt.Fprintf(&b, "\n%s\n  kind: %s, goes to: %s", it.ID, it.Kind, it.List)
		if it.Confirmed != nil {
			fmt.Fprintf(&b, ", confirmed %s: %s", it.Confirmed.Date, it.Confirmed.Reason)
		}
		b.WriteString("\n")
		for _, f := range []struct{ label, text string }{{"title", it.Title}, {"KR", it.KR}, {"TW", it.ZH}, {"note", it.Note}, {"message", it.Message}} {
			if f.text != "" {
				fmt.Fprintf(&b, "  %s: %s\n", f.label, f.text)
			}
		}
		if it.JPENClass != "" {
			fmt.Fprintf(&b, "  JP/EN: %s\n", it.JPENClass)
		}
		for _, v := range it.Variants {
			fmt.Fprintf(&b, "  variant (%d): %s\n", v.Count, v.ZH)
			for _, j := range v.JPEN {
				fmt.Fprintf(&b, "    JP %s／EN %s（%d）\n", orDash(j.JP), orDash(j.EN), j.Count)
			}
		}
		for _, u := range it.Units {
			state := "open"
			if u.Accepted != nil {
				state = "accepted"
			}
			fmt.Fprintf(&b, "  unit %s (%s)\n    id: %s\n    TW: %s\n", u.Key, state, u.ID, u.ZH)
		}
		if len(it.Keys) > 0 {
			fmt.Fprintf(&b, "  units (%d, %d waiting for retranslation):\n    %s\n", len(it.Keys), it.Pending, strings.Join(it.Keys, "\n    "))
		}
	}
	if len(r.Unknown) > 0 {
		fmt.Fprintf(&b, "\nno item of the last build:\n  %s\n", strings.Join(r.Unknown, "\n  "))
	}
	return b.String()
}

// orDash is the text of p, or 「—」 when the unit has no string in that language.
func orDash(p *string) string {
	if p == nil {
		return "—"
	}
	return *p
}

func runAccept(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("accept", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	reason := flags.String("reason", "", "why the items are confirmed (required)")
	date := flags.String("date", "", "the date to write, YYYY-MM-DD (default: today)")
	idFile := flags.String("file", "", "a file of ids, one a line")
	dryRun := flags.Bool("dry-run", false, "write nothing; print the items of the ids")
	tidy := flags.Bool("tidy", false, "delete the entries of "+suggest.AcceptedFile+" that are no warning of the last build")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("accept: " + err.Error())
	}
	// A skill is granted lct accept --dry-run to look items up: turned off
	// again, it would write.
	turnedOff := false
	flags.Visit(func(f *flag.Flag) { turnedOff = turnedOff || (f.Name == "dry-run" && !*dryRun) })
	if turnedOff {
		return cli.UserError("accept: --dry-run takes no false value; leave it out to write")
	}

	if *tidy {
		if flags.NArg() > 0 || *idFile != "" || *reason != "" || *date != "" || *dryRun {
			return cli.UserError("accept: --tidy takes no ids, --file, --reason, --date or --dry-run")
		}
		return acceptTidy(ctx)
	}

	var ids []string
	seen := map[string]bool{}
	add := func(id string) {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, id := range flags.Args() {
		if id = strings.TrimSpace(id); id != "" {
			add(id)
		}
	}
	if *idFile != "" {
		path, err := repoFile(ctx.Root, *idFile)
		if err != nil {
			return cli.UserError(fmt.Sprintf("accept: --file %s", err))
		}
		fromFile, err := readExcludeFile(path)
		if err != nil {
			return cli.UserError(fmt.Sprintf("accept: --file %s", err))
		}
		for _, id := range fromFile {
			add(id)
		}
	}
	if len(ids) == 0 {
		return cli.UserError("accept: give at least one ID (as arguments or with --file), or --tidy")
	}
	if *dryRun {
		return acceptLookup(ctx, ids)
	}
	if strings.TrimSpace(*reason) == "" {
		return cli.UserError("accept: --reason is required and must not be empty")
	}
	day := *date
	if day == "" {
		day = time.Now().Format("2006-01-02")
	} else if t, err := time.Parse("2006-01-02", day); err != nil || t.Format("2006-01-02") != day {
		return cli.UserError(fmt.Sprintf("accept: --date %q: want YYYY-MM-DD", day))
	}

	rev, in, head, err := acceptReview(ctx)
	if err != nil {
		return err
	}
	toDeclined := map[string]bool{}
	for _, s := range rev.Suggestions {
		toDeclined[s.ID] = true
	}
	for _, n := range rev.Notes {
		toDeclined[n.ID] = true
	}
	toAccepted := map[string]bool{}
	// A term group goes to accepted.yaml unit by unit: its id stands for the
	// ids of the units not accepted yet.
	groupUnits := map[string][]string{}
	for _, w := range rev.Warnings {
		toAccepted[w.ID] = true
		if w.Term == nil {
			continue
		}
		groupUnits[w.ID] = []string{}
		for _, u := range w.Units {
			toAccepted[u.ID] = true
			if u.Accepted == nil && !slices.Contains(groupUnits[w.ID], u.ID) {
				groupUnits[w.ID] = append(groupUnits[w.ID], u.ID)
			}
		}
	}
	written := map[string]bool{}
	known := map[string]bool{}
	for _, e := range in.Declined {
		known[e.ID] = true
	}
	for _, e := range in.Accepted {
		known[e.ID] = true
	}

	var unknown []string
	rep := acceptReport{Declined: []string{}, Accepted: []string{}, Skipped: []string{}, Empty: []string{}, RawTag: in.RawTag, TWCommit: head}
	var declined, accepted []suggest.Declined
	for _, id := range ids {
		switch {
		case known[id]:
			// Already in a list: the build may no longer have the item (it drops
			// a declined group of inconsistent translations), so this goes first.
			rep.Skipped = append(rep.Skipped, id)
		case !toDeclined[id] && !toAccepted[id]:
			unknown = append(unknown, id)
		case toDeclined[id]:
			rep.Declined = append(rep.Declined, id)
			declined = append(declined, suggest.Declined{ID: id, Reason: *reason, Date: day})
		default:
			targets := []string{id}
			if us, isGroup := groupUnits[id]; isGroup {
				if targets = us; len(us) == 0 {
					rep.Empty = append(rep.Empty, id)
				}
			}
			for _, t := range targets {
				if known[t] || written[t] {
					continue
				}
				written[t] = true
				rep.Accepted = append(rep.Accepted, t)
				accepted = append(accepted, suggest.Declined{ID: t, Reason: *reason, Date: day})
			}
		}
	}
	if len(unknown) > 0 {
		return cli.UserError(fmt.Sprintf("accept: %d of the ids are no item of the last build (%s), so nothing was written:\n  %s",
			len(unknown), in.RawTag, strings.Join(unknown, "\n  ")))
	}
	if err := suggest.AppendEntries(ctx.Root, suggest.DeclinedFile, declined); err != nil {
		return err
	}
	if err := suggest.AppendEntries(ctx.Root, suggest.AcceptedFile, accepted); err != nil {
		return err
	}
	return ctx.Emit(rep)
}

// acceptTidy deletes the entries of rules/accepted.yaml that are no warning of
// the last build.
func acceptTidy(ctx *cli.Context) error {
	rev, in, head, err := acceptReview(ctx)
	if err != nil {
		return err
	}
	current := map[string]bool{}
	for _, w := range rev.Warnings {
		current[w.ID] = true
		// A term group is accepted by the ids of its units.
		for _, u := range w.Units {
			current[u.ID] = true
		}
	}
	stale := map[string]bool{}
	for _, e := range in.Accepted {
		if !current[e.ID] {
			stale[e.ID] = true
		}
	}
	removed, err := suggest.RemoveEntries(ctx.Root, suggest.AcceptedFile, stale)
	if err != nil {
		return err
	}
	if removed == nil {
		removed = []string{}
	}
	return ctx.Emit(tidyReport{Removed: removed, RawTag: in.RawTag, TWCommit: head})
}

// repoFile resolves path, links and all, and refuses a file outside the repo
// at root or inside a folder named .git: lct accept prints back the lines of
// --file that are no item, so --file may read only what the tools of a skill
// may (the repo, never .git/), not a file of the system or of git.
func repoFile(root, path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("%s: %v", path, err)
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		var pe *fs.PathError
		if errors.As(err, &pe) {
			err = pe.Err // the path is said once
		}
		return "", fmt.Errorf("%s: %v", path, err)
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(realRoot, real)
	if err != nil || !within(realRoot, real) {
		return "", fmt.Errorf("%s: not a file of the repo; name one inside it", path)
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == ".git" {
			return "", fmt.Errorf("%s: inside .git, which lct accept does not read", path)
		}
	}
	return real, nil
}

// acceptLookup prints the items of ids as the review page has them and writes
// nothing. An id that is in a confirmation list but no item of the last build
// (the build drops a declined group of inconsistent translations) is printed
// with what the list says; the other ids that are no item are unknown, which
// is exit code 1.
func acceptLookup(ctx *cli.Context, ids []string) error {
	rev, in, head, err := acceptReview(ctx)
	if err != nil {
		return err
	}
	items := map[string]lookupItem{}
	for _, sg := range rev.Suggestions {
		it := lookupItem{ID: sg.ID, Kind: sg.Kind, List: listDeclined, Confirmed: sg.Declined, Title: sg.Title, KR: sg.KR, Variants: sg.Variants, JPENClass: sg.JPENClass, Pending: sg.Pending, Keys: []string{}}
		for _, v := range sg.Variants {
			it.Keys = append(it.Keys, v.Keys...)
		}
		if sg.Term != nil {
			it.KR, it.ZH = sg.Term.KR, sg.Term.ZH
		}
		if sg.Kind == suggest.KindNote && len(sg.Evidence) > 0 {
			it.Note = sg.Evidence[0].Note
			if sg.Evidence[0].Key != "" {
				it.Keys = []string{sg.Evidence[0].Key}
			}
		}
		items[sg.ID] = it
	}
	for _, n := range rev.Notes {
		items[n.ID] = lookupItem{ID: n.ID, Kind: suggest.KindNote, List: listDeclined, Confirmed: n.Declined, KR: n.KR, ZH: n.ZH, Note: n.Note, Keys: []string{n.Key}, Pending: n.Pending}
	}
	for _, w := range rev.Warnings {
		items[w.ID] = lookupItem{ID: w.ID, Kind: "warning", List: listAccepted, Confirmed: w.Accepted, KR: w.KR, ZH: w.ZH, Message: w.Message, Keys: w.Keys, Pending: w.Pending, Units: w.Units}
		addUnitLookups(items, w, rev.Queued)
	}
	listed := map[string]lookupItem{}
	for _, f := range []struct {
		list    string
		entries []suggest.Declined
	}{{listAccepted, in.Accepted}, {listDeclined, in.Declined}} {
		for _, e := range f.entries {
			kind, _, _ := strings.Cut(e.ID, "/")
			listed[e.ID] = lookupItem{ID: e.ID, Kind: kind, List: f.list, Confirmed: &site.Confirmed{Reason: e.Reason, Date: e.Date}, Keys: []string{}}
		}
	}
	rep := lookupReport{Items: []lookupItem{}, Unknown: []string{}, RawTag: in.RawTag, TWCommit: head}
	for _, id := range ids {
		if it, ok := items[id]; ok {
			rep.Items = append(rep.Items, it)
		} else if it, ok := listed[id]; ok {
			rep.Items = append(rep.Items, it)
		} else {
			rep.Unknown = append(rep.Unknown, id)
		}
	}
	if err := ctx.Emit(rep); err != nil {
		return err
	}
	if len(rep.Unknown) > 0 {
		return cli.ErrFindings
	}
	return nil
}

// addUnitLookups adds to items the units of the term group w: a unit is
// accepted by its own id, which the review page lets a person copy to keep that
// unit alone. Units of one KR text and one translation share the id, and its
// item lists them all. queued holds the keys waiting for retranslation.
func addUnitLookups(items map[string]lookupItem, w site.WarningItem, queued map[string]bool) {
	for _, u := range w.Units {
		it, ok := items[u.ID]
		if !ok {
			it = lookupItem{ID: u.ID, Kind: "warning", List: listAccepted, Confirmed: u.Accepted, KR: u.KR, ZH: u.ZH, Note: u.Note, Message: w.Message, Keys: []string{}}
		}
		it.Keys = append(it.Keys, u.Key)
		if queued[u.Key] {
			it.Pending++
		}
		items[u.ID] = it
	}
}

// acceptReview computes the review items of the last build, the way lct site
// does, and returns them with their inputs and the commit of the tw worktree.
func acceptReview(ctx *cli.Context) (site.Review, site.ReviewInputs, string, error) {
	ss, err := openSiteSources(ctx, "accept")
	if err != nil {
		return site.Review{}, site.ReviewInputs{}, "", err
	}
	in, err := loadReviewInputs(ctx, "accept", ss.Rules)
	if err != nil {
		return site.Review{}, in, "", err
	}
	head, err := gitx.Head(filepath.Join(ctx.Root, filepath.FromSlash(ctx.Config.Worktrees.TW)))
	if err != nil {
		return site.Review{}, in, "", err
	}
	si := site.Inputs{KR: ss.Set.KR, EN: ss.Set.EN, JP: ss.Set.JP, ZH: ss.ZH, Records: ss.Records, Rules: ss.Rules, Review: &in}
	rev, err := site.BuildReview(si, site.OrderFiles(ss.Set.KR.Files(), ss.Rules))
	return rev, in, head, err
}
