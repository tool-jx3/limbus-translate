package batch

import (
	"time"

	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// Twins are pending units of one TextGroup: lines that say the same thing in
// the same way, so one translation serves them all. Next hands out only one
// of them at a time, and Submit gives the others the translation it accepted.
//
// A group that is FileBound only holds twins of one file; twinKey adds the
// file to such a group. A unit that carries what a person wrote for it (see
// CarriesInput) is a case of its own: it is never held back for a twin and
// never filled in by one, since what the person wrote was meant for that unit
// and the twin's translator has not seen it. Its twins may still wait for it
// and take its translation.

// twinKey is what two pending units have in common when they are twins.
type twinKey struct {
	TextGroup
	// File is the file of the units when the text is FileBound, else "".
	File string
}

// twinKeyOf is the twinKey of a unit of file whose group is g.
func twinKeyOf(g TextGroup, file string) twinKey {
	if !g.FileBound() {
		file = ""
	}
	return twinKey{TextGroup: g, File: file}
}

// CarriesInput reports whether the pending item carries what a person wrote
// for that unit: the hint of a request for another translation, or the human
// text of an adjustment that expired (store.RefAdjustment), which the unit is
// translated again with as a reference. Such a unit is never filled with
// another unit's translation, by a twin or by the plan's same-text reuse:
// what the person wrote must reach a translator.
func CarriesInput(it store.PendingItem) bool {
	return it.Hint != "" || it.Refs[store.RefAdjustment] != ""
}

// busyTwins returns the twinKeys of the pending units that are leased to a
// batch at now, which are being translated. It opens the files of those units
// only. Items whose file or pointer has left KR are left out.
func busyTwins(env Env, ld *loader, pending []store.PendingItem, now time.Time) (map[twinKey]bool, error) {
	var busy map[twinKey]bool
	for _, it := range pending {
		if store.LeaseFree(it, now) {
			continue
		}
		file, pointer, err := units.SplitKey(it.Key)
		if err != nil {
			continue // LoadPending checked the keys
		}
		fd, err := ld.file(file)
		if err != nil {
			return nil, err
		}
		if fd == nil {
			continue
		}
		pos, ok := fd.index[pointer]
		if !ok {
			continue
		}
		if busy == nil {
			busy = map[twinKey]bool{}
		}
		busy[twinKeyOf(TextGroupOf(env.Rules, fd.doc, fd.units[pos]), file)] = true
	}
	return busy, nil
}

// twinFilter picks the units of one file for a batch so that no text is
// translated twice: a unit is left out while a twin of it is being translated
// in another batch (busy), or has been picked for this one.
type twinFilter struct {
	env   Env
	fd    *fileData
	busy  map[twinKey]bool
	taken map[twinKey]bool
}

func newTwinFilter(env Env, fd *fileData, busy map[twinKey]bool) *twinFilter {
	return &twinFilter{env: env, fd: fd, busy: busy, taken: map[twinKey]bool{}}
}

// pick reports whether the unit at position pos of the file, which is
// eligible, goes into the batch, and if it does notes its text as taken.
func (f *twinFilter) pick(pos int, item store.PendingItem) bool {
	k := twinKeyOf(TextGroupOf(f.env.Rules, f.fd.doc, f.fd.units[pos]), f.fd.file)
	if !CarriesInput(item) && (f.busy[k] || f.taken[k]) {
		return false
	}
	f.taken[k] = true
	return true
}

// acceptedUnit is a unit a submission accepted: the translation its twins can
// take over.
type acceptedUnit struct {
	file  string
	group TextGroup
	zh    string
}

// reuseForTwins gives the translations that the submission accepted to their
// twins: the pending units of the same TextGroup (and the same file, for a
// short text) that are free to take it. A twin is left alone, and stays in the
// queue, when
//   - it is a unit of the batch, which its out file answers;
//   - it is leased to another batch that has not run out, flagged needs_review,
//     or carries what a person wrote for it (see CarriesInput);
//   - the translation fails CheckNew in the twin's file: the glossary may name
//     terms for some files only.
//
// A twin that takes the translation gets the record of an accepted unit and
// leaves the queue; its key goes in Reused. The files of the units that can be
// twins are opened, which are the pending units of the families of the
// accepted texts, and for the short texts only those of their own file.
func (s *submission) reuseForTwins() error {
	if len(s.accepted) == 0 {
		return nil
	}
	sources := map[twinKey]acceptedUnit{} // the first accepted text wins
	families := map[string]bool{}         // of the texts shared between files
	files := map[string]bool{}            // of the short texts
	for _, a := range s.accepted {
		k := twinKeyOf(a.group, a.file)
		if _, dup := sources[k]; dup {
			continue
		}
		sources[k] = a
		if a.group.FileBound() {
			files[a.file] = true
		} else {
			families[a.group.Family] = true
		}
	}

	now := s.env.now()
	for i, it := range s.pending {
		if s.removed[i] || hasFlag(it.Flags, store.FlagNeedsReview) || CarriesInput(it) || !store.LeaseFree(it, now) {
			continue
		}
		if _, inBatch := s.batchKR[it.Key]; inBatch {
			continue
		}
		file, pointer, err := units.SplitKey(it.Key)
		if err != nil {
			continue // LoadPending checked the keys
		}
		if !families[s.env.Rules.Family(file)] && !files[file] {
			continue
		}
		fd, err := s.ld.file(file)
		if err != nil {
			return err
		}
		if fd == nil {
			continue
		}
		pos, ok := fd.index[pointer]
		if !ok {
			continue
		}
		u := fd.units[pos]
		src, ok := sources[twinKeyOf(TextGroupOf(s.env.Rules, fd.doc, u), file)]
		if !ok || validate.HasHard(CheckNewTerms(s.env.Rules, s.env.Provisional, file, u.KR, src.zh)) {
			continue
		}
		s.records[it.Key] = s.claudeRecord(it.Key, u.KR, src.zh)
		s.removed[i] = true
		s.res.Reused = append(s.res.Reused, it.Key)
	}
	return nil
}
