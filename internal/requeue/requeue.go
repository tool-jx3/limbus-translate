// Package requeue sends translated units back to the pending queue, with a
// reason the translator sees as the unit's hint, and resets or releases
// pending units that are stuck (spec 11.1, phase 1 direct mode).
//
// A requeue never changes a record: the old translation stays in the build
// until the new one is submitted.
package requeue

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// Reason is the Reason of the pending items Apply adds.
const Reason = "requeue"

// defaultOrigins are the origins a record needs to be requeued when the
// selector names none. Copies are left out: they are identifiers and texts
// that are the same in every language, there is nothing to translate again.
var defaultOrigins = []string{store.OriginParatranz, store.OriginClaude, store.OriginFallback, store.OriginHuman}

// knownOrigins are the values Record.Origin can take.
var knownOrigins = append(slices.Clone(defaultOrigins), store.OriginCopy)

// Selector says which units to act on.
//
// Keys, Files and Origins select records. Keys are unit keys; Files are
// patterns for the file part of a key, in the form of lct.yaml's frozen list
// (see config.MatchAny). A record is selected when its key is in Keys or its
// file matches Files; with neither, every record is. Origins then keep the
// records of those origins; without Origins that is every origin but copy
// (copies in the selection are counted as skipped, not requeued). A Selector
// whose Keys, Files and Origins are all empty selects no record.
//
// NeedsReview and Leased act on the pending queue only: units flagged
// needs_review are reset, units with a lease are released. When Keys, Files
// or Origins are given they narrow these too (Origins by the origin of the
// unit's record: a pending unit without one is out); otherwise they act on the
// whole queue. A pending unit they handled is not counted again for its
// record.
type Selector struct {
	Keys        []string
	Files       []string
	Origins     []string
	NeedsReview bool
	Leased      bool
}

// Result counts what Apply did.
type Result struct {
	// Requeued units got a pending item (reason "requeue", the record's
	// translation as PrevZH, the hint).
	Requeued int `json:"requeued"`
	// Reset units were flagged needs_review: the flag, the attempts and the
	// lease are gone and they carry the hint.
	Reset int `json:"reset"`
	// Released units lost their lease; nothing else changed.
	Released int `json:"released"`
	// AlreadyPending units were in the queue already; only their hint is new.
	AlreadyPending int `json:"already_pending"`
	// SkippedFrozen units are in a frozen file.
	SkippedFrozen int `json:"skipped_frozen"`
	// SkippedCopy units are copies.
	SkippedCopy int `json:"skipped_copy"`
	// Missing counts the keys that cannot be requeued: the file or pointer
	// has left KR, or (for a key named in Keys) there is no record to send
	// back. MissingKeys lists them in order; it is never nil.
	Missing     int      `json:"missing"`
	MissingKeys []string `json:"missing_keys"`
}

// String is the report of the CLI.
func (r Result) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "requeued %d, reset %d, released %d\n", r.Requeued, r.Reset, r.Released)
	fmt.Fprintf(&b, "already pending %d, skipped %d frozen and %d copy, missing %d", r.AlreadyPending, r.SkippedFrozen, r.SkippedCopy, r.Missing)
	for _, k := range r.MissingKeys {
		fmt.Fprintf(&b, "\n  missing: %s", k)
	}
	return b.String()
}

// recordMode reports whether the selector picks records.
func (s Selector) recordMode() bool {
	return len(s.Keys)+len(s.Files)+len(s.Origins) > 0
}

// Validate reports a selector Apply cannot use: one that selects nothing, an
// unknown origin, a malformed key or a malformed file pattern. A pattern that
// merely matches nothing is not an error.
func (s Selector) Validate() error {
	if !s.recordMode() && !s.NeedsReview && !s.Leased {
		return errors.New("nothing is selected")
	}
	for _, o := range s.Origins {
		if !slices.Contains(knownOrigins, o) {
			return fmt.Errorf("unknown origin %q (want one of %s)", o, strings.Join(knownOrigins, ", "))
		}
	}
	for _, k := range s.Keys {
		if _, _, err := units.SplitKey(k); err != nil {
			return err
		}
	}
	for _, p := range s.Files {
		if strings.HasSuffix(p, "/**") {
			continue // config.MatchAny reads these as a directory
		}
		if _, err := path.Match(p, ""); err != nil {
			return fmt.Errorf("file pattern %q: %w", p, err)
		}
	}
	return nil
}

// Apply carries out sel on the store of one tw worktree. kr is the KR pack
// of the raw snapshot, which tells whether a key is still there; frozen are
// the patterns of the files that must not change (lct.yaml). hint is the
// reason the translator is shown for every unit that is requeued, reset, or
// already pending.
//
// The whole read-modify-write is one store lock, so Apply can run next to
// Next and Submit. It changes nothing when it returns an error.
func Apply(st *store.Store, kr *pack.Pack, frozen []string, sel Selector, hint string) (*Result, error) {
	if st == nil {
		return nil, errors.New("requeue: no store")
	}
	if kr == nil {
		return nil, errors.New("requeue: no KR pack")
	}
	if err := sel.Validate(); err != nil {
		return nil, fmt.Errorf("requeue: %w", err)
	}

	res := &Result{MissingKeys: []string{}}
	err := st.WithLock(func() error {
		pending, err := st.LoadPending()
		if err != nil {
			return err
		}
		var records map[string]store.Record
		if sel.recordMode() {
			if records, err = st.LoadRecords(); err != nil {
				return err
			}
		}
		r := newRun(kr, frozen, sel, hint, records, pending, res)
		r.stuckItems()
		if err := r.requeueRecords(); err != nil {
			return err
		}
		if !r.changed {
			return nil
		}
		return st.SavePending(r.pending)
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// run is the state of one Apply call.
type run struct {
	kr      *pack.Pack
	frozen  []string
	sel     Selector
	hint    string
	records map[string]store.Record // nil unless sel picks records
	pending []store.PendingItem
	res     *Result

	index   map[string]int  // key -> index in pending
	touched map[string]bool // keys stuckItems handled
	origins map[string]bool // origins that count
	changed bool            // pending differs from what was loaded

	krPointers map[string]map[string]bool // file -> pointers in KR; nil for a file KR lacks
}

func newRun(kr *pack.Pack, frozen []string, sel Selector, hint string, records map[string]store.Record, pending []store.PendingItem, res *Result) *run {
	r := &run{
		kr: kr, frozen: frozen, sel: sel, hint: hint, records: records, pending: pending, res: res,
		index:      make(map[string]int, len(pending)),
		touched:    map[string]bool{},
		origins:    map[string]bool{},
		krPointers: map[string]map[string]bool{},
	}
	for i, it := range pending {
		r.index[it.Key] = i
	}
	// Without Origins every origin counts, copy too: a copy is then reported
	// as skipped instead of passing unnoticed.
	origins := sel.Origins
	if len(origins) == 0 {
		origins = knownOrigins
	}
	for _, o := range origins {
		r.origins[o] = true
	}
	return r
}

// fileOf is the file part of a unit key. Keys are valid: the store and
// Selector.Validate check them.
func fileOf(key string) string {
	file, _, _ := strings.Cut(key, "|")
	return file
}

// inScope reports whether key is among the Keys or in a file of Files; with
// neither given, everything is.
func (r *run) inScope(key string) bool {
	if len(r.sel.Keys)+len(r.sel.Files) == 0 {
		return true
	}
	return slices.Contains(r.sel.Keys, key) || config.MatchAny(r.sel.Files, fileOf(key))
}

// originOK reports whether the record of key passes Origins, when they are
// given. A key without a record passes only when no Origins are given.
func (r *run) originOK(key string) bool {
	if len(r.sel.Origins) == 0 {
		return true
	}
	rec, ok := r.records[key]
	return ok && r.origins[rec.Origin]
}

// stuckItems resets the units flagged needs_review and releases the leased
// ones, as the selector asks.
func (r *run) stuckItems() {
	if !r.sel.NeedsReview && !r.sel.Leased {
		return
	}
	for i := range r.pending {
		it := &r.pending[i]
		review := r.sel.NeedsReview && slices.Contains(it.Flags, store.FlagNeedsReview)
		leased := r.sel.Leased && it.Lease != nil
		if !(review || leased) || !r.inScope(it.Key) || !r.originOK(it.Key) {
			continue
		}
		if review {
			it.Flags = slices.DeleteFunc(slices.Clone(it.Flags), func(f string) bool { return f == store.FlagNeedsReview })
			if len(it.Flags) == 0 {
				it.Flags = nil
			}
			it.Attempts = 0
			it.Lease = nil
			it.Hint = r.hint
			r.res.Reset++
		} else {
			it.Lease = nil
			r.res.Released++
		}
		r.touched[it.Key] = true
		r.changed = true
	}
}

// candidates are the keys the record selection looks at, sorted and unique:
// the Keys, whether or not they have a record, and the records in scope.
func (r *run) candidates() []string {
	seen := map[string]bool{}
	var keys []string
	add := func(k string) {
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	for _, k := range r.sel.Keys {
		add(k)
	}
	if len(r.sel.Files) > 0 || len(r.sel.Keys) == 0 {
		for k := range r.records {
			if len(r.sel.Files) == 0 || config.MatchAny(r.sel.Files, fileOf(k)) {
				add(k)
			}
		}
	}
	slices.Sort(keys)
	return keys
}

// requeueRecords sends the selected records back to the queue.
func (r *run) requeueRecords() error {
	if !r.sel.recordMode() {
		return nil
	}
	for _, key := range r.candidates() {
		rec, hasRec := r.records[key]
		if r.touched[key] || hasRec && !r.origins[rec.Origin] {
			continue
		}
		file := fileOf(key)
		switch i, queued := r.index[key]; {
		case hasRec && rec.Origin == store.OriginCopy:
			r.res.SkippedCopy++
		case config.MatchAny(r.frozen, file):
			r.res.SkippedFrozen++
		case queued:
			r.pending[i].Hint = r.hint
			r.res.AlreadyPending++
			r.changed = true
		default:
			missing := !hasRec // nothing to send back
			if hasRec {
				in, err := r.inKR(key)
				if err != nil {
					return err
				}
				missing = !in
			}
			if missing {
				r.res.Missing++
				r.res.MissingKeys = append(r.res.MissingKeys, key)
				continue
			}
			r.pending = append(r.pending, store.PendingItem{Key: key, Reason: Reason, PrevZH: rec.ZH, Hint: r.hint})
			r.res.Requeued++
			r.changed = true
		}
	}
	return nil
}

// inKR reports whether KR still has the unit of key. A KR file is read once,
// and only when a key of it gets this far.
func (r *run) inKR(key string) (bool, error) {
	file, pointer, err := units.SplitKey(key)
	if err != nil {
		return false, err
	}
	ptrs, loaded := r.krPointers[file]
	if !loaded {
		if r.kr.Has(file) {
			doc, err := r.kr.Doc(file)
			if err != nil {
				return false, fmt.Errorf("requeue: %w", err)
			}
			list := units.Extract(file, doc)
			ptrs = make(map[string]bool, len(list))
			for _, u := range list {
				ptrs[u.Pointer] = true
			}
		}
		r.krPointers[file] = ptrs
	}
	return ptrs[pointer], nil
}
