package history

import (
	"fmt"
	"slices"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/adjust"
	"github.com/tool-jx3/limbus-translate/internal/plan"
	"github.com/tool-jx3/limbus-translate/internal/requeue"
	"github.com/tool-jx3/limbus-translate/internal/rewrite"
	"github.com/tool-jx3/limbus-translate/internal/rulefile"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// CausesVersion is the format version of the causes in a Record. A Record
// without it (0) was written before causes existed.
const CausesVersion = 1

// The kinds of a Cause. The first four are rules.
const (
	CauseKindRewrite           = "rewrite"
	CauseKindAdjustment        = "adjustment"
	CauseKindAdjustmentExpired = "adjustment-expired"
	CauseKindRequeue           = "requeue"
	CauseKindReuse             = "reuse"
	CauseKindKRChanged         = "kr-changed"
	CauseKindNew               = "new"
	CauseKindDeleted           = "deleted"
	CauseKindLate              = "late"         // was a stand-in, translated for the first time
	CauseKindRetranslated      = "retranslated" // translated again, no rule found
	CauseKindOther             = "other"
	CauseKindUnknown           = "unknown"
)

// Cause is one reason an Entry changed.
type Cause struct {
	Kind   string `json:"kind"`
	ID     string `json:"id,omitempty"`
	Batch  string `json:"batch,omitempty"`  // the translation batch
	Reason string `json:"reason,omitempty"` // of the adjustment, or from pending.jsonl
	Note   string `json:"note,omitempty"`   // what an unknown cause is about
}

// RuleInfo describes a rule a Cause points to. Record.Rules keys it as
// "<kind>:<id>".
type RuleInfo struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Path   string `json:"path"`
	Title  string `json:"title"`
	Commit string `json:"commit"`
}

// IsRuleCause reports whether kind is a rule: rewrite, adjustment,
// adjustment-expired or requeue.
func IsRuleCause(kind string) bool {
	switch kind {
	case CauseKindRewrite, CauseKindAdjustment, CauseKindAdjustmentExpired, CauseKindRequeue:
		return true
	}
	return false
}

// RuleKey is the key of a rule in Record.Rules: "<kind>:<id>".
func RuleKey(kind, id string) string { return kind + ":" + id }

// inferCauses returns the causes of e, an entry of the Record that compares
// prev with cur. prevRules and curRules are the rules of their RulesCommit; a
// nil one could not be read. Make infers the causes of all its entries at once
// (inference), which is the same.
func inferCauses(prev, cur *Side, prevRules, curRules *RuleFiles, e Entry) []Cause {
	return newInference(prev, cur,
		sideRules{commit: prev.RulesCommit, rules: prevRules},
		sideRules{commit: cur.RulesCommit, rules: curRules}).causes(e)
}

// sideRules are the rules of one Release: those of its rules commit, or why
// they could not be read.
type sideRules struct {
	commit string
	rules  *RuleFiles // nil when err is set
	err    error
}

// inference infers the causes of the entries of one Record, as the spec
// (原因的推斷) lays down. What every entry needs is worked out once.
type inference struct {
	prev, cur           *Side
	prevRules, curRules sideRules
	pending             map[string]store.PendingItem // the queue of prev, by key

	rewrites    []replay        // the rewrite files applied since prev, in file name order
	rewriteGone []string        // the ids of those that neither rules have
	requeues    []foundRequeue  // the request files cur has applied, in file name order
	requeueGone []string        // the ids of those that neither rules have
	newRequeue  map[string]bool // whether cur.Applied.Requeue[i] was applied since prev, by id

	rules map[string]RuleInfo // the rules the causes point to, by RuleKey
}

// foundRequeue is a request file and the rules commit it was read from.
type foundRequeue struct {
	f      requeue.File
	commit string
}

func newInference(prev, cur *Side, prevRules, curRules sideRules) *inference {
	for _, s := range []*sideRules{&prevRules, &curRules} {
		if s.rules == nil && s.err == nil {
			s.err = fmt.Errorf("history: the rules of %s were not read", s.commit)
		}
	}
	in := &inference{
		prev: prev, cur: cur, prevRules: prevRules, curRules: curRules,
		pending: map[string]store.PendingItem{}, newRequeue: map[string]bool{}, rules: map[string]RuleInfo{},
	}
	for _, it := range prev.Pending {
		in.pending[it.Key] = it
	}
	for _, id := range cur.Applied.Rewrite {
		if slices.Contains(prev.Applied.Rewrite, id) {
			continue
		}
		f, commit, ok := find(curRules, prevRules, func(r *RuleFiles) []rewrite.File { return r.Rewrites },
			func(f rewrite.File) bool { return f.ID == id })
		if !ok {
			in.rewriteGone = append(in.rewriteGone, id)
			continue
		}
		in.rewrites = append(in.rewrites, newReplay(f, commit))
	}
	for _, id := range cur.Applied.Requeue {
		in.newRequeue[id] = !slices.Contains(prev.Applied.Requeue, id)
		f, commit, ok := find(curRules, prevRules, func(r *RuleFiles) []requeue.File { return r.Requeues },
			func(f requeue.File) bool { return f.ID == id })
		if !ok {
			in.requeueGone = append(in.requeueGone, id)
			continue
		}
		in.requeues = append(in.requeues, foundRequeue{f, commit})
	}
	// The order of rewrite.LoadFiles and requeue.LoadFiles, which the plan
	// applies them in: that of the file names.
	slices.SortFunc(in.rewrites, func(a, b replay) int { return fileOrder(a.f.ID, b.f.ID) })
	slices.SortFunc(in.requeues, func(a, b foundRequeue) int { return fileOrder(a.f.ID, b.f.ID) })
	slices.SortFunc(in.rewriteGone, fileOrder)
	slices.SortFunc(in.requeueGone, fileOrder)
	return in
}

// fileOrder compares the ids of two rule files as their file names compare.
func fileOrder(a, b string) int { return strings.Compare(a+rulefile.Ext, b+rulefile.Ext) }

// find returns the last of the rules of first that match, with the commit of
// those rules, or else the last of second; rules that could not be read have
// none. The last, because of two adjustments for one unit the later one wins.
func find[T any](first, second sideRules, list func(*RuleFiles) []T, match func(T) bool) (T, string, bool) {
	for _, s := range []sideRules{first, second} {
		if s.rules == nil {
			continue
		}
		var found *T
		for _, x := range list(s.rules) {
			if match(x) {
				found = &x
			}
		}
		if found != nil {
			return *found, s.commit, true
		}
	}
	var zero T
	return zero, "", false
}

// gone is the note of a cause that rests on a rule which neither rules have:
// what the rule is, and, for each rules commit, why it is not there.
func (in *inference) gone(what string) string {
	var why []string
	for _, s := range []sideRules{in.curRules, in.prevRules} {
		w := "the rules of " + s.commit + " lack it"
		if s.err != nil {
			w = s.err.Error()
		}
		if !slices.Contains(why, w) {
			why = append(why, w)
		}
	}
	return what + " not found: " + strings.Join(why, "; ")
}

// use adds a rule a cause points to to the index of the Record. A rule found
// in both rules commits is described as cur has it.
func (in *inference) use(kind, id, path, title, commit string) {
	key := RuleKey(kind, id)
	if old, ok := in.rules[key]; ok && (old.Commit == in.curRules.commit || commit != in.curRules.commit) {
		return
	}
	in.rules[key] = RuleInfo{Kind: kind, ID: id, Path: path, Title: strings.TrimSpace(title), Commit: commit}
}

// unitState is what the two Releases know of the unit of an entry.
type unitState struct {
	prevKey, curKey string
	prev, cur       store.Record
	hasPrev, hasCur bool
	// translated: a translation batch made the record of cur since prev.
	translated bool
}

func (in *inference) unit(e Entry) unitState {
	u := unitState{curKey: units.Key(e.File, e.Pointer)}
	u.prevKey = u.curKey
	if e.OldPointer != "" {
		u.prevKey = units.Key(e.File, e.OldPointer)
	}
	if e.Kind != KindNew {
		u.prev, u.hasPrev = in.prev.State.Records[u.prevKey]
	}
	if e.Kind != KindDeleted {
		u.cur, u.hasCur = in.cur.State.Records[u.curKey]
	}
	u.translated = translated(u.prev, u.hasPrev, u.cur, u.hasCur)
	return u
}

// translated reports whether cur, the record of a unit in the new Release
// (hasCur: it has one), was made by a translation batch since prev, the
// record of the old one (hasPrev: it had one). Such a record carries the id
// of its batch, and prev had another. A copy, fallback or Paratranz record has
// no batch; an adjustment or a reuse has a batch of its own kind.
func translated(prev store.Record, hasPrev bool, cur store.Record, hasCur bool) bool {
	if !hasCur || cur.Batch == "" || isAdjustment(cur) || strings.HasPrefix(cur.Batch, plan.BatchReuse) {
		return false
	}
	return !hasPrev || prev.Batch != cur.Batch
}

// isAdjustment reports whether rec is the record of a human adjustment.
func isAdjustment(rec store.Record) bool {
	return rec.Origin == store.OriginHuman && strings.HasPrefix(rec.Batch, plan.BatchAdjustment)
}

// written reports whether the record of the unit in cur was written since
// prev: it is not the record prev had, but for a change of the text that the
// rewrites since prev account for (rewritten).
func (u unitState) written(rewritten bool) bool {
	return !u.hasPrev || u.prev.Batch != u.cur.Batch || u.prev.KRSha != u.cur.KRSha ||
		u.prev.ZH != u.cur.ZH && !rewritten
}

// causes infers the causes of e in the order of the spec, steps 1 to 9: all
// that hold, "other" when none does. What cannot be told for want of a rule is
// one unknown cause, its notes joined.
func (in *inference) causes(e Entry) []Cause {
	u := in.unit(e)
	var out []Cause
	add := func(c ...Cause) { out = append(out, c...) }
	unknown := func(note string) {
		if note == "" {
			return
		}
		for i := range out {
			if out[i].Kind == CauseKindUnknown {
				out[i].Note += "; " + note
				return
			}
		}
		add(Cause{Kind: CauseKindUnknown, Note: note})
	}

	// 1, 2: the kind of the entry. A deleted unit has no new record to have
	// any other cause.
	switch e.Kind {
	case KindDeleted:
		return []Cause{{Kind: CauseKindDeleted}}
	case KindNew:
		add(Cause{Kind: CauseKindNew})
	case KindKRChanged:
		c := Cause{Kind: CauseKindKRChanged}
		if u.translated {
			c.Batch = u.cur.Batch
		}
		add(c)
	}
	// The rewrites are worked out first: a rewrite of the record of an
	// adjustment or a reuse is not a new adjustment or reuse.
	rewrites, rewriteNote := in.rewriteCauses(e, u)
	rewritten := len(rewrites) > 0

	// 3: an adjustment made the new record.
	if u.hasCur && isAdjustment(u.cur) && u.written(rewritten) {
		c, note := in.adjustmentCause(u)
		if note != "" {
			unknown(note)
		} else {
			add(c)
		}
	}
	// 4: the old record was an adjustment, the new one is none.
	if u.hasPrev && isAdjustment(u.prev) && !(u.hasCur && isAdjustment(u.cur)) {
		c, note := in.expiredCause(u)
		if note != "" {
			unknown(note)
		} else {
			add(c)
		}
	}
	// 5
	add(rewrites...)
	unknown(rewriteNote)
	// 6
	requeues, requeueNote, undecided := in.requeueCauses(u)
	add(requeues...)
	unknown(requeueNote)
	// 7
	if u.hasCur && strings.HasPrefix(u.cur.Batch, plan.BatchReuse) && u.written(rewritten) {
		add(Cause{Kind: CauseKindReuse, Batch: u.cur.Batch})
	}
	// 8: translated, for none of the reasons above.
	if len(out) == 0 && u.translated {
		switch {
		case undecided != "":
			unknown(undecided)
		case e.OldOrigin == OriginStandIn || e.OldOrigin == store.OriginFallback:
			add(Cause{Kind: CauseKindLate, Batch: u.cur.Batch})
		default:
			add(Cause{Kind: CauseKindRetranslated, Batch: u.cur.Batch, Reason: in.pending[u.prevKey].Reason})
		}
	}
	// 9
	if len(out) == 0 {
		add(Cause{Kind: CauseKindOther})
	}
	return out
}

// adjustmentCause is the adjustment that made the new record of the unit. It
// is told by the KR text and the translation, not by the key, which lct
// adjust tidy moves; it is looked for in the rules of cur, then of prev. The
// note says why it cannot be told.
func (in *inference) adjustmentCause(u unitState) (Cause, string) {
	file := strings.TrimPrefix(u.cur.Batch, plan.BatchAdjustment)
	a, commit, ok := find(in.curRules, in.prevRules, adjustmentsOf, func(a adjust.Adjustment) bool {
		return a.File == file && a.KRSha == u.cur.KRSha && a.ZH == u.cur.ZH
	})
	if !ok {
		return Cause{}, in.gone("the adjustment of " + file + " that made the record")
	}
	in.use(CauseKindAdjustment, file, adjust.Dir+"/"+file, file, commit)
	return Cause{Kind: CauseKindAdjustment, ID: file, Reason: strings.TrimSpace(a.Reason)}, ""
}

// expiredCause is the adjustment that made the old record of the unit and
// made the new one no more. It is looked for in the rules of prev first: the
// new ones may have deleted it.
func (in *inference) expiredCause(u unitState) (Cause, string) {
	file := strings.TrimPrefix(u.prev.Batch, plan.BatchAdjustment)
	a, commit, ok := find(in.prevRules, in.curRules, adjustmentsOf, func(a adjust.Adjustment) bool {
		return a.File == file && a.KRSha == u.prev.KRSha && a.ZH == u.prev.ZH
	})
	if !ok {
		return Cause{}, in.gone("the adjustment of " + file + " that made the old record")
	}
	in.use(CauseKindAdjustmentExpired, file, adjust.Dir+"/"+file, file, commit)
	c := Cause{Kind: CauseKindAdjustmentExpired, ID: file, Reason: strings.TrimSpace(a.Reason)}
	if u.translated {
		c.Batch = u.cur.Batch
	}
	return c, ""
}

func adjustmentsOf(r *RuleFiles) []adjust.Adjustment { return r.Adjustments }

// requeueCauses is step 6, for a unit translated since prev: the request files
// that sent it back. A unit queued in prev as requeue was sent back by the
// last request file cur has applied that selects it, as the request file
// selects (by the old record); one sent back and translated since prev, by
// each of the request files applied since prev that selects it. Each cause
// carries the translation batch.
//
// note says why a unit queued as requeue has none. undecided names the
// request files applied since prev that might have selected the unit but
// cannot be read; it matters only when the unit has no other cause.
func (in *inference) requeueCauses(u unitState) (causes []Cause, note, undecided string) {
	if !u.translated {
		return nil, "", ""
	}
	selects := func(f requeue.File) bool {
		return u.hasPrev && (f.Selector.Selects(u.prevKey, u.prev) || f.Selector.Selects(u.curKey, u.prev))
	}
	var picked []foundRequeue
	if it, ok := in.pending[u.prevKey]; ok && it.Reason == requeue.Reason {
		var last *foundRequeue
		for i := range in.requeues {
			if selects(in.requeues[i].f) {
				last = &in.requeues[i]
			}
		}
		if last != nil {
			picked = append(picked, *last)
		} else {
			note = "queued as " + requeue.Reason + ", but no request file applied selects it"
			if len(in.requeueGone) > 0 {
				note += "; " + in.gone("request "+strings.Join(in.requeueGone, ", "))
			}
		}
	}
	for _, r := range in.requeues {
		if in.newRequeue[r.f.ID] && selects(r.f) &&
			!slices.ContainsFunc(picked, func(p foundRequeue) bool { return p.f.ID == r.f.ID }) {
			picked = append(picked, r)
		}
	}
	for _, r := range picked {
		causes = append(causes, Cause{Kind: CauseKindRequeue, ID: r.f.ID, Batch: u.cur.Batch})
		in.use(CauseKindRequeue, r.f.ID, requeue.Dir+"/"+r.f.ID+rulefile.Ext, r.f.Reason, r.commit)
	}
	var gone []string
	for _, id := range in.requeueGone {
		if in.newRequeue[id] {
			gone = append(gone, id)
		}
	}
	if u.hasPrev && len(gone) > 0 {
		undecided = in.gone("request " + strings.Join(gone, ", "))
	}
	return causes, note, undecided
}
