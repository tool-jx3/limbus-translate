package plan

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/adjust"
	"github.com/tool-jx3/limbus-translate/internal/align"
	"github.com/tool-jx3/limbus-translate/internal/batch"
	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/requeue"
	"github.com/tool-jx3/limbus-translate/internal/rewrite"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// File is the file of the tw worktree's .meta that holds the Summary of the
// last plan. Its raw tag is the snapshot the translation state belongs to.
const File = "plan.json"

// ReasonAdjustmentExpired is the Reason of the pending item of a unit whose
// human adjustment no longer applies; its Refs hold the adjustment under
// RefAdjustment (store.RefAdjustment, where batch can see it), for the
// translator to look at.
const (
	ReasonAdjustmentExpired = "adjustment-expired"
	RefAdjustment           = store.RefAdjustment
)

// The prefixes of the Batch of the records Run writes: the adjustment file
// (its name in rules/adjustments) and the key of the unit whose translation
// was reused.
const (
	batchAdjustment = "adjustment:"
	batchReuse      = "reuse:"
)

// Inputs is what Run works with.
type Inputs struct {
	// Root is the repo root: the rules, and the adjustment, requeue and
	// rewrite files under rules/.
	Root string
	// RawWorktree is the raw worktree, at the current snapshot; its tags hold
	// the earlier ones.
	RawWorktree string
	// TWRoot is the tw worktree, whose store Run changes.
	TWRoot string
	// RulesCommit names the commit of the rules (see gitx.Describe); every
	// record Run writes carries it as Rules.
	RulesCommit string
	// BuildRawTag is the raw tag of the last build (.meta/build.json), the
	// snapshot the state belongs to when no plan has run yet; "" when nothing
	// was built either.
	BuildRawTag string
	// Src holds the official KR, EN and JP packs of the current snapshot.
	Src pack.Set
	// Frozen are the patterns of the frozen files (lct.yaml).
	Frozen []string
	Fields *fields.Rules
	Rules  *rules.Rules
	// TmpDir is where the KR pack of the previous snapshot is exported for
	// the time of the run; "" is the system's temporary directory.
	TmpDir string
}

// Summary is what Run did; .meta/plan.json holds it. The lists are never nil.
type Summary struct {
	// RawTag is the current snapshot, PrevRawTag the one the state belonged
	// to before it. They are equal when the plan of RawTag found no older
	// state; a plan of a snapshot that was planned already keeps PrevRawTag,
	// Changes, FrozenChanged, NewSpeakers and NewFilesDefaultFamily of that
	// plan (see Run).
	RawTag      string `json:"raw_tag"`
	PrevRawTag  string `json:"prev_raw_tag"`
	RulesCommit string `json:"rules_commit"`
	// Changes are what became of the strings between the two snapshots; all
	// zero when there is no older one.
	Changes Changes `json:"changes"`
	// Copy counts the copy records written: units the field rules do not
	// translate that had no copy of their KR text yet.
	Copy int `json:"copy"`
	// Reused counts the units that took the translation of the same text.
	Reused      int               `json:"reused"`
	Adjustments AdjustmentSummary `json:"adjustments"`
	// RequeueApplied and RewriteApplied are the request and rewrite files
	// applied this time (each file is applied once), in the order they were.
	RequeueApplied []Applied `json:"requeue_applied"`
	RewriteApplied []Applied `json:"rewrite_applied"`
	// FrozenChanged are the keys of the units of frozen files whose KR text
	// changed; they kept their translation (see Moved.FrozenChanged).
	FrozenChanged []string `json:"frozen_changed"`
	// NewSpeakers are the model and speaker strings, as KR writes them, of the
	// stories and RPG dialogues (batch.IsStory) of the current snapshot that
	// the previous one did not have, sorted.
	NewSpeakers []string `json:"new_speakers"`
	// FieldChanges is what the field rules learned from the current packs
	// add to or remove from rules/fields.yaml (see lct fields learn).
	FieldChanges []fields.Change `json:"field_changes"`
	// NewFilesDefaultFamily are the files added in this snapshot that fall in
	// the catch-all family of guides/routing.yaml: no family was made for them.
	NewFilesDefaultFamily []string `json:"new_files_default_family"`
	// Pending counts the pending units a batch can take now: in KR, not
	// leased and not waiting for a person.
	Pending int `json:"pending"`
}

// AdjustmentSummary says what became of the adjustments of rules/adjustments.
type AdjustmentSummary struct {
	// Applied counts the adjustments in effect at their own key.
	Applied int `json:"applied"`
	// Moved are the adjustments in effect at another key, by key, sorted.
	Moved []MovedKey `json:"moved"`
	// Expired are the keys of the adjustments not in effect, sorted: the KR
	// text they were made for is gone (the unit is translated again, with the
	// adjustment as a reference), the unit is copied or is in a frozen file
	// that has a translation, or another adjustment of the unit won.
	Expired []string `json:"expired"`
	// ExpiredUnits maps the key of an expired adjustment to the key of the
	// unit it is about in this snapshot, where that is not its own key: a
	// line inserted before the unit moved it. It is the unit plan queued as
	// adjustment-expired. An adjustment whose unit kept its key, or whose
	// unit the plan cannot tell (see expiredUnit), has no entry. Every run
	// works it out, but a run of a snapshot that was planned already, which
	// makes no transition, cannot follow a unit through one: it keeps the
	// entries of that plan (see Run).
	ExpiredUnits map[string]string `json:"expired_units,omitempty"`
}

// MovedKey is an adjustment whose key is out of date: the text it was made
// for is at To now.
type MovedKey struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Applied is one requeue or rewrite file that was applied: Units counts the
// units it sent back to the translator (requeued, already pending and given
// its reason, or waiting for a person and reset) or the records it rewrote.
type Applied struct {
	ID    string `json:"id"`
	Units int    `json:"units"`
}

// String is the report of the CLI.
func (s *Summary) String() string {
	var b strings.Builder
	if s.PrevRawTag == s.RawTag {
		fmt.Fprintf(&b, "plan of %s (no new snapshot), rules %s\n", s.RawTag, s.RulesCommit)
	} else {
		c := s.Changes
		fmt.Fprintf(&b, "plan of %s after %s, rules %s\n", s.RawTag, s.PrevRawTag, s.RulesCommit)
		fmt.Fprintf(&b, "strings: %d unchanged (%d moved), %d changed, %d new, %d deleted\n", c.Unchanged, c.Moved, c.Changed, c.New, c.Deleted)
		fmt.Fprintf(&b, "files: %d added, %d deleted, %d changed, %d shifted\n", len(c.FilesAdded), len(c.FilesDeleted), len(c.FilesChanged), len(c.FilesShifted))
	}
	a := s.Adjustments
	fmt.Fprintf(&b, "copied %d, reused %d\n", s.Copy, s.Reused)
	fmt.Fprintf(&b, "adjustments: %d applied, %d moved, %d expired\n", a.Applied, len(a.Moved), len(a.Expired))
	for _, m := range a.Moved {
		fmt.Fprintf(&b, "  moved: %s -> %s\n", m.From, m.To)
	}
	for _, k := range a.Expired {
		fmt.Fprintf(&b, "  expired: %s\n", k)
	}
	for _, f := range s.RequeueApplied {
		fmt.Fprintf(&b, "requeue %s: %d units\n", f.ID, f.Units)
	}
	for _, f := range s.RewriteApplied {
		fmt.Fprintf(&b, "rewrite %s: %d units\n", f.ID, f.Units)
	}
	for _, l := range []struct {
		what  string
		items []string
	}{
		{"frozen KR changed", s.FrozenChanged},
		{"new speaker", s.NewSpeakers},
		{"new file of the default family", s.NewFilesDefaultFamily},
	} {
		for _, it := range l.items {
			fmt.Fprintf(&b, "%s: %s\n", l.what, it)
		}
	}
	for _, c := range s.FieldChanges {
		fmt.Fprintf(&b, "field rule: %s\n", c)
	}
	fmt.Fprintf(&b, "pending: %d", s.Pending)
	return b.String()
}

func (in Inputs) check() error {
	switch {
	case in.Root == "" || in.RawWorktree == "" || in.TWRoot == "":
		return errors.New("plan: the repo root, the raw worktree and the tw worktree are all needed")
	case in.Src.KR == nil || in.Src.EN == nil || in.Src.JP == nil:
		return errors.New("plan: the KR, EN and JP packs are all needed")
	case in.Fields == nil || in.Rules == nil:
		return errors.New("plan: no field rules or no rules")
	}
	return nil
}

// Run decides, for every unit of the current snapshot, where its translation
// comes from (spec 8.2), changes the state of the tw store to match, and
// writes the Summary to .meta/plan.json.
//
// The state belongs to the snapshot of the last plan (the raw tag of
// .meta/plan.json), or else to that of the last build (BuildRawTag), or else
// to the current one. When that is not the current snapshot, the state is
// first carried over to it (Transition). When it is, and the last plan was of
// it too, the summary keeps what that plan said the snapshot brought
// (PrevRawTag, Changes, FrozenChanged, NewSpeakers, NewFilesDefaultFamily, and
// the ExpiredUnits of the adjustments that still are expired); the rest of it
// is what this run did. Then every unit of the current KR gets
// the first source that holds:
//
//  1. the field rules do not translate it: a copy record (its KR text), and
//     it leaves the queue;
//  2. it is in a frozen file and has a record that is not a copy (a copy is
//     no translation): the record stays as it is;
//  3. a human adjustment applies to it (adjust.Resolve, at its key or moved
//     to it): a human record with the adjustment's text, and it leaves the
//     queue. Of two adjustments of one unit, the one at its own key wins over
//     one that moved, and of equals the later one in file order. An
//     adjustment of a unit that steps 1 or 2 decide is not applied;
//  4. it has a record of its KR text (not a copy): the record stays;
//  5. it is queued as new or changed, and a unit of the same text
//     (batch.TextGroup) has a translation of it, the current record of a unit
//     that is not queued itself, or the record of a unit the new snapshot
//     deleted: that translation is copied, if it passes the checks of a new
//     translation (batch.CheckNew); a text that is FileBound only takes one
//     from its own file. A deleted unit that had a request pending serves no
//     reuse: a unit of its text takes the request over instead (see
//     planner.reuse), so an entry numbered again is translated again as asked;
//  6. otherwise it is queued: a unit that is not queued yet is queued as
//     new, or, when it has a record of another KR text, as kr-changed with
//     that record's text as the previous translation (as a transition would
//     have queued it), and a copy record it has goes (a copy is no
//     translation). A unit whose adjustment expired (its KR text changed; see
//     expiredUnit for how the unit is found) is queued as adjustment-expired,
//     with the adjustment as a reference, unless it has a record of its KR
//     text already that is not human: the unit is queued once, and no more
//     once it is translated again.
//
// Steps 1 to 4 and 6 come first, and the adjustments that expired are queued.
// The rewrite files and the request files of rules/rewrite and rules/requeue
// that .meta/applied.json does not list are applied, each in file name order,
// and their ids are added to applied.json: the rewrites, then step 5, then the
// requests. So the translations step 5 copies are rewritten already; a record
// a request sends back serves no reuse (what the requests will queue is
// worked out before step 5 and counts as queued); and a requested unit that
// step 5 fills is sent back with the copied text as its previous
// translation, its request kept. A requeued unit shows the rewritten text as
// its previous translation too, and so does a unit queued already whose
// previous translation was the text of the record rewritten. A request that
// names a unit waiting for a person (needs_review) resets it, so that it is
// handed out again. Neither kind of file touches a unit that steps 1 to 3
// decided: the next plan would set it back. A rewrite changes no record of a
// frozen file and no text it would leave with a hard issue of a new
// translation; a request leaves out frozen files and copies too (see
// requeue.Apply).
//
// The whole run is one store lock, and the store is written at its end: the
// records, the queue, applied.json, then plan.json. The ids of the applied
// files go in after the changes they made, so a file is never listed as
// applied without them. Each file is replaced atomically, but not the four
// together: a run that dies while writing them leaves a store to reset to its
// last commit (CI commits the tw worktree only after a plan that went
// through).
func Run(in Inputs) (*Summary, error) {
	if err := in.check(); err != nil {
		return nil, err
	}
	src, err := raw.LastSource(in.RawWorktree)
	if err != nil {
		return nil, fmt.Errorf("plan: %w", err)
	}
	if src == nil {
		return nil, fmt.Errorf("plan: %s has no snapshot", in.RawWorktree)
	}

	// What does not depend on the state is read before the lock is taken.
	adjustments, err := adjust.Load(in.Root)
	if err != nil {
		return nil, fmt.Errorf("plan: %w", err)
	}
	requeues, err := requeue.LoadFiles(in.Root)
	if err != nil {
		return nil, fmt.Errorf("plan: %w", err)
	}
	rewrites, err := rewrite.LoadFiles(in.Root)
	if err != nil {
		return nil, fmt.Errorf("plan: %w", err)
	}
	fieldChanges, err := learnFieldChanges(in)
	if err != nil {
		return nil, err
	}

	st := store.Open(in.TWRoot)
	var sum *Summary
	err = st.WithLock(func() error {
		p := &planner{
			in: in,
			sum: &Summary{
				RawTag: src.Tag, RulesCommit: in.RulesCommit,
				Adjustments:    AdjustmentSummary{Moved: []MovedKey{}, Expired: []string{}},
				RequeueApplied: []Applied{}, RewriteApplied: []Applied{},
				FrozenChanged: []string{}, NewSpeakers: []string{},
				FieldChanges: fieldChanges, NewFilesDefaultFamily: []string{},
			},
			cur:     map[string]unitRef{},
			settled: map[string]bool{},
		}
		if err := p.load(st); err != nil {
			return err
		}
		if p.from != p.sum.RawTag {
			tmp, err := os.MkdirTemp(in.TmpDir, "lct-plan-")
			if err != nil {
				return fmt.Errorf("plan: %w", err)
			}
			defer os.RemoveAll(tmp)
			if err := p.transition(tmp); err != nil {
				return err
			}
		}
		adjusted := resolveAdjustments(adjustments, in.Src.KR)
		if err := p.settle(adjusted.claims); err != nil {
			return err
		}
		if err := p.expire(adjusted); err != nil {
			return err
		}
		applied, err := st.LoadApplied()
		if err != nil {
			return err
		}
		p.applyRewrites(&applied, rewrites)
		var todo []requeue.File // the request files to apply
		for _, f := range requeues {
			if !slices.Contains(applied.Requeue, f.ID) {
				todo = append(todo, f)
			}
		}
		// What the requests will queue counts as queued for step 5: a record
		// they send back serves no reuse.
		preview, _, err := p.requeue(todo)
		if err != nil {
			return err
		}
		if err := p.reuse(keySet(preview)); err != nil {
			return err
		}
		pending, done, err := p.requeue(todo)
		if err != nil {
			return err
		}
		for _, a := range done {
			applied.Requeue = append(applied.Requeue, a.ID)
		}
		p.sum.RequeueApplied = append(p.sum.RequeueApplied, done...)
		p.report(pending)

		// In this order; see the doc of Run.
		if err := st.SaveRecords(p.records); err != nil {
			return err
		}
		if err := st.SavePending(pending); err != nil {
			return err
		}
		if err := st.SaveApplied(applied); err != nil {
			return err
		}
		if err := st.WriteMetaJSON(File, p.sum); err != nil {
			return err
		}
		sum = p.sum
		return nil
	})
	if err != nil {
		return nil, err
	}
	return sum, nil
}

// learnFieldChanges learns the field rules from the packs (as lct fields
// learn does) and returns what they would change in rules/fields.yaml. Packs
// from which nothing is learned hold no evidence and change nothing.
func learnFieldChanges(in Inputs) ([]fields.Change, error) {
	learned, err := fields.Learn(in.Src.KR, in.Src.EN, in.Src.JP)
	if err != nil {
		return nil, fmt.Errorf("plan: %w", err)
	}
	if learned.Counted == 0 {
		return []fields.Change{}, nil
	}
	manual, err := fields.LoadManual(in.Root)
	if err != nil {
		return nil, fmt.Errorf("plan: %w", err)
	}
	old, err := fields.LoadLearned(in.Root)
	if err != nil {
		return nil, fmt.Errorf("plan: %w", err)
	}
	changes := learned.Without(manual).Diff(old)
	if changes == nil {
		changes = []fields.Change{}
	}
	return changes, nil
}

// planner is the state of one Run.
type planner struct {
	in  Inputs
	sum *Summary

	records map[string]store.Record
	pending map[string]store.PendingItem // by key; the request files make the final queue from it

	// After a transition: the KR pack of the previous snapshot, the units it
	// had that the current one has not, and the records as they were before
	// the transition, by their keys in the previous snapshot.
	oldKR       *pack.Pack
	deleted     []DeletedUnit
	prevRecords map[string]store.Record

	// from is the snapshot the state belongs to when Run starts.
	from string
	// planned holds, when the current snapshot was planned already, the
	// ExpiredUnits that plan found (see expire).
	planned map[string]string

	files   []*krFile          // the current KR files, in name order
	cur     map[string]unitRef // the current units, by key
	settled map[string]bool    // keys whose source steps 1 to 3 decided
}

// krFile is one file of the current KR pack.
type krFile struct {
	name   string
	doc    *jsonx.Node
	units  []units.Unit
	frozen bool
}

// unitRef is a unit of the current KR pack with the sha of its KR text.
type unitRef struct {
	file *krFile
	i    int
	sha  string
}

func (r unitRef) unit() units.Unit { return r.file.units[r.i] }

// load reads the state and the snapshot it belongs to.
func (p *planner) load(st *store.Store) error {
	var last Summary
	found, err := st.ReadMetaJSON(File, &last)
	if err != nil {
		return err
	}
	switch {
	case found && last.RawTag != "":
		p.from = last.RawTag
	case p.in.BuildRawTag != "":
		p.from = p.in.BuildRawTag
	default:
		p.from = p.sum.RawTag
	}
	p.sum.PrevRawTag = p.from
	if found && last.RawTag == p.sum.RawTag && last.PrevRawTag != "" {
		// This snapshot was planned already: what it brought stays in the
		// summary, for the report and the review issue of its raw tag.
		s := p.sum
		s.PrevRawTag, s.Changes = last.PrevRawTag, last.Changes
		s.FrozenChanged, s.NewSpeakers, s.NewFilesDefaultFamily = nonNil(last.FrozenChanged), nonNil(last.NewSpeakers), nonNil(last.NewFilesDefaultFamily)
		p.planned = last.Adjustments.ExpiredUnits
	}
	if p.records, err = st.LoadRecords(); err != nil {
		return err
	}
	pending, err := st.LoadPending()
	if err != nil {
		return err
	}
	p.setPending(pending)
	return nil
}

func (p *planner) setPending(items []store.PendingItem) {
	p.pending = make(map[string]store.PendingItem, len(items))
	for _, it := range items {
		p.pending[it.Key] = it
	}
}

// pendingList is the queue sorted by key.
func (p *planner) pendingList() []store.PendingItem {
	items := slices.Collect(maps.Values(p.pending))
	sort.Slice(items, func(i, j int) bool { return items[i].Key < items[j].Key })
	return items
}

// transition carries the state from the previous snapshot to the current one,
// exporting the previous KR pack into dir, and notes what the new snapshot
// brought: the changes, the new speakers, the new files of the catch-all
// family.
func (p *planner) transition(dir string) error {
	old, err := raw.OpenKRAt(p.in.RawWorktree, p.sum.PrevRawTag, dir)
	if err != nil {
		return fmt.Errorf("plan: %w", err)
	}
	moved, err := Transition(old, p.in.Src.KR, p.records, p.pendingList(), p.in.Frozen)
	if err != nil {
		return err
	}
	p.oldKR, p.deleted, p.prevRecords, p.records = old, moved.Deleted, p.records, moved.Records
	p.setPending(moved.Pending)
	p.sum.Changes, p.sum.FrozenChanged = moved.Changes, moved.FrozenChanged
	if p.sum.FrozenChanged == nil {
		p.sum.FrozenChanged = []string{}
	}
	before, err := storySpeakers(old)
	if err != nil {
		return err
	}
	after, err := storySpeakers(p.in.Src.KR)
	if err != nil {
		return err
	}
	for s := range after {
		if !before[s] {
			p.sum.NewSpeakers = append(p.sum.NewSpeakers, s)
		}
	}
	slices.Sort(p.sum.NewSpeakers)
	if def := p.in.Rules.DefaultFamily(); def != "" {
		for _, f := range p.sum.Changes.FilesAdded {
			if p.in.Rules.Family(f) == def {
				p.sum.NewFilesDefaultFamily = append(p.sum.NewFilesDefaultFamily, f)
			}
		}
	}
	return nil
}

// storySpeakers is the set of model and speaker strings (batch.SpeakerKey) of
// the stories and RPG dialogues of kr.
func storySpeakers(kr *pack.Pack) (map[string]bool, error) {
	set := map[string]bool{}
	for _, file := range kr.Files() {
		if !batch.IsStory(file) {
			continue
		}
		doc, err := kr.Doc(file)
		if err != nil {
			return nil, fmt.Errorf("plan: %w", err)
		}
		for _, u := range units.Extract(file, doc) {
			if s := batch.SpeakerKey(doc, u); s != "" {
				set[s] = true
			}
		}
	}
	return set, nil
}

// claim is an adjustment that applies to a unit of the current KR.
type claim struct {
	a      adjust.Adjustment
	status adjust.Status // Applied or Moved
	// blocked: the unit's source is decided before the adjustments (steps 1
	// and 2), so the adjustment is not in effect.
	blocked bool
}

// resolved is where the adjustments apply.
type resolved struct {
	claims  map[string]*claim   // the claim that won each unit, by key
	expired []adjust.Adjustment // the KR text is gone
	beaten  []beaten            // another adjustment won the unit
}

// beaten is an adjustment that lost its unit to another one.
type beaten struct {
	a    adjust.Adjustment
	unit string // the key of the unit
}

// resolveAdjustments finds the unit of every adjustment of list, which is in
// load order. Of two that apply to one unit, the one at its own key beats one
// that moved there, and of equals the later one wins.
func resolveAdjustments(list []adjust.Adjustment, kr *pack.Pack) resolved {
	r := resolved{claims: map[string]*claim{}}
	for _, a := range list {
		res := adjust.Resolve(a, kr)
		if res.Status == adjust.Expired {
			r.expired = append(r.expired, a)
			continue
		}
		c := &claim{a: a, status: res.Status}
		w, taken := r.claims[res.Key]
		switch {
		case !taken:
		case w.status == adjust.Applied && c.status == adjust.Moved:
			r.beaten = append(r.beaten, beaten{a, res.Key})
			continue
		default:
			r.beaten = append(r.beaten, beaten{w.a, res.Key})
		}
		r.claims[res.Key] = c
	}
	return r
}

// humanRecord is the record the adjustment of c writes at key.
func (p *planner) humanRecord(key string, c *claim) store.Record {
	return store.Record{
		Key: key, KRSha: c.a.KRSha, ZH: c.a.ZH, Origin: store.OriginHuman,
		Since: p.sum.RawTag, Rules: p.in.RulesCommit, Batch: batchAdjustment + c.a.File,
	}
}

// sameRecord reports whether a and b say the same, whenever and under whatever
// rules they were written (Since and Rules).
func sameRecord(a, b store.Record) bool {
	a.Since, a.Rules, b.Since, b.Rules = "", "", "", ""
	return a.Key == b.Key && a.KRSha == b.KRSha && a.ZH == b.ZH && a.Origin == b.Origin &&
		a.Model == b.Model && a.Batch == b.Batch && slices.Equal(a.Flags, b.Flags)
}

// settle goes through every unit of the current KR and settles steps 1 to 4
// and 6 of Run; claims are the adjustments by the key they apply to.
func (p *planner) settle(claims map[string]*claim) error {
	kr := p.in.Src.KR
	for _, name := range kr.Files() {
		doc, err := kr.Doc(name)
		if err != nil {
			return fmt.Errorf("plan: %w", err)
		}
		f := &krFile{name: name, doc: doc, units: units.Extract(name, doc), frozen: config.MatchAny(p.in.Frozen, name)}
		p.files = append(p.files, f)
		al := align.NewFileAligner(name, doc, p.in.Src.EN, p.in.Src.JP)
		for i, u := range f.units {
			key := units.Key(name, u.Pointer)
			ref := unitRef{file: f, i: i, sha: units.KRSha(u.KR)}
			p.cur[key] = ref
			en, jp, err := al.Get(u.Pointer)
			if err != nil {
				return fmt.Errorf("plan: %w", err)
			}
			p.unit(key, ref, p.in.Fields.Classify(u, en, jp).Translate, claims[key])
		}
	}
	return nil
}

// unit settles one unit: translate is what the field rules say, c the
// adjustment that applies to it (nil if none).
func (p *planner) unit(key string, ref unitRef, translate bool, c *claim) {
	u := ref.unit()
	rec, hasRec := p.records[key]
	switch {
	case !translate: // 1
		p.settled[key] = true
		copyRec := store.Record{Key: key, KRSha: ref.sha, ZH: u.KR, Origin: store.OriginCopy, Since: p.sum.RawTag, Rules: p.in.RulesCommit}
		if !hasRec || !sameRecord(rec, copyRec) {
			p.records[key] = copyRec
			p.sum.Copy++
		}
		delete(p.pending, key)
		if c != nil {
			c.blocked = true
		}
		return
	case ref.file.frozen && hasRec && rec.Origin != store.OriginCopy: // 2
		p.settled[key] = true
		if c != nil && !sameRecord(rec, p.humanRecord(key, c)) {
			c.blocked = true // the record is not the adjustment's own
		}
		return
	case c != nil: // 3
		p.settled[key] = true
		if want := p.humanRecord(key, c); !hasRec || !sameRecord(rec, want) {
			p.records[key] = want
		}
		delete(p.pending, key)
		return
	case hasRec && rec.KRSha == ref.sha && rec.Origin != store.OriginCopy: // 4
		return
	}
	// 6, unless step 5 finds the same text later.
	if hasRec && rec.Origin == store.OriginCopy {
		delete(p.records, key)
		hasRec = false
	}
	if _, queued := p.pending[key]; !queued {
		item := store.PendingItem{Key: key, Reason: ReasonNew}
		if hasRec {
			// A record of another KR text, which no transition queued: as
			// one would have, its text is the previous translation.
			item = store.PendingItem{Key: key, Reason: ReasonKRChanged, PrevZH: rec.ZH}
		}
		p.pending[key] = item
	}
}

// expire reports what became of the adjustments, and queues the units of
// those that expired (step 6).
func (p *planner) expire(r resolved) error {
	adj := &p.sum.Adjustments
	// unitOf notes the unit an expired adjustment is about, unless it is the
	// unit of its own key.
	unitOf := func(a adjust.Adjustment, unit string) {
		if unit == "" || unit == a.Key {
			return
		}
		if adj.ExpiredUnits == nil {
			adj.ExpiredUnits = map[string]string{}
		}
		adj.ExpiredUnits[a.Key] = unit
	}
	for key, c := range r.claims {
		switch {
		case c.blocked:
			adj.Expired = append(adj.Expired, c.a.Key)
			unitOf(c.a, key)
		case c.status == adjust.Applied:
			adj.Applied++
		default:
			adj.Moved = append(adj.Moved, MovedKey{From: c.a.Key, To: key})
		}
	}
	for _, b := range r.beaten {
		adj.Expired = append(adj.Expired, b.a.Key) // the unit has the winner's record
		unitOf(b.a, b.unit)
	}
	var finder *unitFinder
	if len(r.expired) > 0 {
		finder = p.newUnitFinder()
	}
	for _, a := range r.expired {
		adj.Expired = append(adj.Expired, a.Key)
		key, err := finder.expiredUnit(a)
		if err != nil {
			return err
		}
		if key != "" {
			unitOf(a, key)
		} else {
			// Without a transition in this run the unit cannot be followed
			// from the previous snapshot; the run that made the transition
			// found it, and queued it.
			unitOf(a, p.planned[a.Key])
		}
		// A unit that steps 1 to 3 settled (another adjustment may have
		// moved to it) has its source.
		ref, ok := p.cur[key]
		if !ok || p.settled[key] {
			continue
		}
		if rec, has := p.records[key]; has && rec.KRSha == ref.sha && rec.Origin != store.OriginHuman {
			continue // translated again already
		}
		item, queued := p.pending[key]
		if !queued {
			item = store.PendingItem{Key: key}
		}
		item.Reason = ReasonAdjustmentExpired
		item.Refs = maps.Clone(item.Refs)
		if item.Refs == nil {
			item.Refs = map[string]string{}
		}
		item.Refs[RefAdjustment] = a.ZH
		p.pending[key] = item
	}
	slices.Sort(adj.Expired)
	sort.Slice(adj.Moved, func(i, j int) bool { return adj.Moved[i].From < adj.Moved[j].From })
	return nil
}

// unitFinder finds the units of the adjustments that expired (expiredUnit).
type unitFinder struct {
	p *planner
	// human holds the keys of the human records the state had before the
	// transition, by what wrote them.
	human map[humanRef][]string
	// versions are the alignments of the files of the previous snapshot with
	// the current one, made as they are needed.
	versions map[string]*fileVersions
}

// humanRef is what tells the human record of an adjustment apart: the
// adjustment file (Record.Batch), the KR text and the translation.
type humanRef struct{ batch, krSha, zh string }

// fileVersions is one file of the previous snapshot and what the current one
// made of it.
type fileVersions struct {
	old   map[string]units.Unit         // its units, by pointer
	moves map[string]align.VersionMatch // by old pointer
}

func (p *planner) newUnitFinder() *unitFinder {
	f := &unitFinder{p: p, human: map[humanRef][]string{}, versions: map[string]*fileVersions{}}
	for key, rec := range p.prevRecords {
		if rec.Origin == store.OriginHuman && strings.HasPrefix(rec.Batch, batchAdjustment) {
			ref := humanRef{rec.Batch, rec.KRSha, rec.ZH}
			f.human[ref] = append(f.human[ref], key)
		}
	}
	return f
}

// expiredUnit returns the key, in the current KR, of the unit the expired
// adjustment a was made for, or "" when that unit is gone or the plan cannot
// tell which it is. Only an adjustment made for the current snapshot names its
// unit by its key: a key names a unit of one snapshot, a line inserted before
// the unit moves it to another key, and its changed KR keeps adjust.Resolve
// from finding it by its text. So, after a transition in this run, the unit
// is taken where the adjustment was in the snapshot the state belonged to and
// followed to the current one as the transition moved it (align.AlignVersions,
// whose changed strings are the only ones an expired adjustment can be of):
//
//   - the unit the adjustment was in effect at, which has its human record;
//   - else, for an adjustment made for that snapshot, the unit of its key, if
//     that unit had the adjustment's KR text there.
//
// An adjustment made for an older snapshot and in effect at none of its units
// then has no unit the plan can tell, and without a transition no adjustment
// made for another snapshot has one. On a rerun of the plan that is what keeps
// the state as it is: the run that made the transition queued the unit, and
// the state carries that item.
func (f *unitFinder) expiredUnit(a adjust.Adjustment) (string, error) {
	p := f.p
	if a.Raw == p.sum.RawTag {
		return a.Key, nil
	}
	if p.oldKR == nil {
		return "", nil
	}
	from, mustHaveText := "", false
	switch keys := f.human[humanRef{batchAdjustment + a.File, a.KRSha, a.ZH}]; {
	case slices.Contains(keys, a.Key):
		from = a.Key
	case len(keys) == 1:
		from = keys[0]
	case a.Raw == p.sum.PrevRawTag:
		from, mustHaveText = a.Key, true
	default:
		return "", nil
	}
	file, pointer, err := units.SplitKey(from)
	if err != nil {
		return "", nil // a key that names no unit (adjust.Resolve said expired)
	}
	v, err := f.fileVersions(file)
	if err != nil {
		return "", err
	}
	if old, ok := v.old[pointer]; !ok || mustHaveText && units.KRSha(old.KR) != a.KRSha {
		return "", nil
	}
	if m, ok := v.moves[pointer]; ok && m.Status == align.StatusChanged {
		return units.Key(file, m.NewPointer), nil
	}
	return "", nil
}

func (f *unitFinder) fileVersions(file string) (*fileVersions, error) {
	if v, ok := f.versions[file]; ok {
		return v, nil
	}
	oldDoc, err := docOf(f.p.oldKR, file)
	if err != nil {
		return nil, err
	}
	newDoc, err := docOf(f.p.in.Src.KR, file)
	if err != nil {
		return nil, err
	}
	v := &fileVersions{old: map[string]units.Unit{}, moves: map[string]align.VersionMatch{}}
	for _, u := range units.Extract(file, oldDoc) {
		v.old[u.Pointer] = u
	}
	for _, m := range align.AlignVersions(file, oldDoc, newDoc) {
		if m.OldPointer != "" {
			v.moves[m.OldPointer] = m
		}
	}
	f.versions[file] = v
	return v, nil
}

// source is a translation that a unit of the same text may take.
type source struct {
	key  string // the unit's key; for a deleted unit, in the previous snapshot
	file string
	rec  store.Record
}

// reusable reports whether rec, the record of a unit whose KR text has sha,
// may serve another unit: it is a translation of that text that nobody has
// to look at.
func reusable(rec store.Record, sha string) bool {
	switch rec.Origin {
	case store.OriginParatranz, store.OriginClaude, store.OriginHuman:
	default:
		return false
	}
	return rec.KRSha == sha && !slices.Contains(rec.Flags, store.FlagNeedsReview)
}

// deletedRequest is the pending item of a deleted unit that holds what a
// person asked for (carriesRequest), which a unit of its text takes over (see
// reuse).
type deletedRequest struct {
	file   string // the deleted unit's
	item   store.PendingItem
	handed bool // a unit took it over already
}

// carriesRequest reports whether it, the pending item of a unit the new
// snapshot deleted, holds what a person asked for: a request to translate
// the unit again, or a hint (a request that found the unit queued already).
func carriesRequest(it store.PendingItem) bool {
	return it.Reason == requeue.Reason || it.Hint != ""
}

// reuseTarget reports whether a unit queued for reason is one step 5 fills.
func reuseTarget(reason string) bool { return reason == ReasonNew || reason == ReasonKRChanged }

// reuse is step 5 of Run. A unit in queued, or pending, serves no other.
//
// A deleted unit with a request pending (carriesRequest) serves no other
// either: the first target of its text (by key; from the deleted unit's file
// first, from another file only when the text is not FileBound) takes its
// pending item over, with its reason, hint and previous texts and without its
// lease. That is how a request on an entry numbered again reaches the entry,
// instead of the entry taking the translation sent back. A target that takes
// over a request for a new or changed unit may then still take a translation,
// as a target with such an item of its own may.
func (p *planner) reuse(queued map[string]bool) error {
	var targets []string
	want := map[string]bool{} // the KR texts of the targets
	for key, it := range p.pending {
		if !reuseTarget(it.Reason) {
			continue
		}
		if ref, ok := p.cur[key]; ok {
			targets = append(targets, key)
			want[ref.unit().KR] = true
		}
	}
	if len(targets) == 0 {
		return nil
	}
	slices.Sort(targets)

	// The candidates of every group: the current units first, in file and
	// document order, then the deleted ones in their order.
	sources := map[batch.TextGroup][]source{}
	for _, f := range p.files {
		for _, u := range f.units {
			if !want[u.KR] {
				continue
			}
			key := units.Key(f.name, u.Pointer)
			rec, has := p.records[key]
			if _, pending := p.pending[key]; !has || pending || queued[key] || !reusable(rec, p.cur[key].sha) {
				continue
			}
			g := batch.TextGroupOf(p.in.Rules, f.doc, u)
			sources[g] = append(sources[g], source{key: key, file: f.name, rec: rec})
		}
	}
	requests := map[batch.TextGroup][]*deletedRequest{}
	oldUnits := map[string]map[string]units.Unit{} // file -> pointer -> unit
	for _, d := range p.deleted {
		asked := d.Pending != nil && carriesRequest(*d.Pending)
		if !want[d.KR] || !asked && (d.Record == nil || !reusable(*d.Record, units.KRSha(d.KR))) {
			continue
		}
		doc, err := p.oldKR.Doc(d.File)
		if err != nil {
			return fmt.Errorf("plan: %w", err)
		}
		byPointer, ok := oldUnits[d.File]
		if !ok {
			byPointer = map[string]units.Unit{}
			for _, u := range units.Extract(d.File, doc) {
				byPointer[u.Pointer] = u
			}
			oldUnits[d.File] = byPointer
		}
		g := batch.TextGroupOf(p.in.Rules, doc, byPointer[d.Pointer])
		if asked {
			requests[g] = append(requests[g], &deletedRequest{file: d.File, item: *d.Pending})
			continue
		}
		sources[g] = append(sources[g], source{key: units.Key(d.File, d.Pointer), file: d.File, rec: *d.Record})
	}

	for _, key := range targets {
		ref := p.cur[key]
		u := ref.unit()
		g := batch.TextGroupOf(p.in.Rules, ref.file.doc, u)
		if r, ok := handOn(requests[g], ref.file.name, g.FileBound()); ok {
			it := r.item
			p.pending[key] = store.PendingItem{Key: key, Reason: it.Reason, Hint: it.Hint, PrevKR: it.PrevKR, PrevZH: it.PrevZH}
			if !reuseTarget(it.Reason) {
				continue
			}
		}
		s, ok := p.pick(sources[g], ref.file.name, g.FileBound(), u.KR)
		if !ok {
			continue
		}
		p.records[key] = store.Record{
			Key: key, KRSha: ref.sha, ZH: s.rec.ZH, Origin: s.rec.Origin, Model: s.rec.Model,
			Since: p.sum.RawTag, Rules: p.in.RulesCommit, Batch: batchReuse + s.key,
		}
		delete(p.pending, key)
		p.sum.Reused++
	}
	return nil
}

// pick chooses the translation a unit of file with the KR text kr takes from
// cands: the first that passes the checks of a new translation, from the same
// file first, and from no other file when fileBound.
func (p *planner) pick(cands []source, file string, fileBound bool, kr string) (source, bool) {
	for _, sameFile := range []bool{true, false} {
		if !sameFile && fileBound {
			break
		}
		for _, c := range cands {
			if (c.file == file) != sameFile {
				continue
			}
			if !validate.HasHard(batch.CheckNew(p.in.Rules, file, kr, c.rec.ZH)) {
				return c, true
			}
		}
	}
	return source{}, false
}

// handOn returns the first request of reqs that no unit took over yet, and
// marks it taken: one of file first, and of no other file when fileBound.
func handOn(reqs []*deletedRequest, file string, fileBound bool) (*deletedRequest, bool) {
	for _, sameFile := range []bool{true, false} {
		if !sameFile && fileBound {
			break
		}
		for _, r := range reqs {
			if !r.handed && (r.file == file) == sameFile {
				r.handed = true
				return r, true
			}
		}
	}
	return nil, false
}

// held reports whether steps 1 to 3 decided the source of the unit of key (a
// copy, a frozen unit with a translation, an adjusted unit). No rewrite or
// request file touches such a unit: the next plan would only set it back.
func (p *planner) held(key string) bool { return p.settled[key] }

// applyRewrites applies the rewrite files that applied does not list, and
// adds their ids to it.
func (p *planner) applyRewrites(applied *store.Applied, files []rewrite.File) {
	ok := func(key, kr, zh string) bool {
		file, _, _ := strings.Cut(key, "|")
		return !p.held(key) && !config.MatchAny(p.in.Frozen, file) && !validate.HasHard(batch.CheckNew(p.in.Rules, file, kr, zh))
	}
	for _, f := range files {
		if slices.Contains(applied.Rewrite, f.ID) {
			continue
		}
		// A pending item that shows the text of its unit's record as the
		// previous translation shows the rewritten text: the translation made
		// from it then keeps the rewrite.
		follow := map[string]bool{}
		for key, it := range p.pending {
			if rec, has := p.records[key]; has && it.PrevZH != "" && it.PrevZH == rec.ZH {
				follow[key] = true
			}
		}
		changed := rewrite.Apply(f, p.records, p.in.Src.KR, ok)
		for _, key := range changed {
			if follow[key] {
				it := p.pending[key]
				it.PrevZH = p.records[key].ZH
				p.pending[key] = it
			}
		}
		applied.Rewrite = append(applied.Rewrite, f.ID)
		p.sum.RewriteApplied = append(p.sum.RewriteApplied, Applied{ID: f.ID, Units: len(changed)})
	}
}

// requeue applies the request files, in order, to the records and the queue
// there are now, and returns the queue they leave with what each did. It
// changes neither the records nor the queue of p.
func (p *planner) requeue(files []requeue.File) ([]store.PendingItem, []Applied, error) {
	pending := p.pendingList()
	var done []Applied
	for _, f := range files {
		out, res, err := requeue.ApplyTo(p.records, pending, p.in.Src.KR, p.in.Frozen, f.Selector, strings.TrimSpace(f.Reason), p.held)
		if err != nil {
			return nil, nil, fmt.Errorf("plan: %s/%s.yaml: %w", requeue.Dir, f.ID, err)
		}
		pending = out
		done = append(done, Applied{ID: f.ID, Units: res.Requeued + res.AlreadyPending + res.Reset})
	}
	return pending, done, nil
}

// keySet is the set of the keys of items.
func keySet(items []store.PendingItem) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, it := range items {
		set[it.Key] = true
	}
	return set
}

// report counts the units a batch can take now.
func (p *planner) report(pending []store.PendingItem) {
	p.sum.Pending = batch.CountClaimable(pending, time.Now(), func(key string) bool {
		_, inKR := p.cur[key]
		return inKR
	})
}

// nonNil turns a nil list into an empty one, which is written as [].
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
