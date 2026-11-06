// Package plan decides, for a new KR snapshot, what happens to every
// translation unit: which translations carry over, which units are queued.
package plan

import (
	"fmt"
	"maps"
	"slices"
	"sort"

	"github.com/tool-jx3/limbus-translate/internal/align"
	"github.com/tool-jx3/limbus-translate/internal/batch"
	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// Reasons of the pending items Transition adds (store.PendingItem.Reason).
const (
	ReasonNew       = batch.ReasonNew
	ReasonKRChanged = "kr-changed"
)

// Moved is the translation state after a move from one KR version to the
// next: every key is a unit of the new version.
type Moved struct {
	Records map[string]store.Record
	Pending []store.PendingItem // sorted by key
	// Deleted lists the units of the old version that have no counterpart in
	// the new one, files in name order, each file in document order.
	Deleted []DeletedUnit
	Changes Changes
	// FrozenChanged lists, sorted, the new keys of frozen units whose KR text
	// changed and whose translation was kept as it was.
	FrozenChanged []string
}

// DeletedUnit is a string of the old version that the new one no longer
// has. Its KR text and translation may serve a new unit with the same text.
type DeletedUnit struct {
	File, Field, KR string
	Pointer         string        // where the string was in the old version
	Record          *store.Record // nil when the unit had no record
	// Pending is the pending item the unit had, as it was (nil when it had
	// none): the plan hands a request on it to the unit that takes its place.
	Pending *store.PendingItem
}

// Changes counts what became of the units. Unchanged includes the units
// that only moved to another pointer; Moved counts those. So Unchanged,
// Changed and New add up to the strings of the new version, Deleted counts
// strings of the old one. The file lists are sorted.
type Changes struct {
	Unchanged int `json:"unchanged"`
	Moved     int `json:"moved"`
	Changed   int `json:"changed"`
	New       int `json:"new"`
	Deleted   int `json:"deleted"`

	FilesAdded   []string `json:"files_added,omitempty"`   // only in the new version
	FilesDeleted []string `json:"files_deleted,omitempty"` // only in the old version
	// FilesChanged are the files of both versions with a changed, new or
	// deleted unit.
	FilesChanged []string `json:"files_changed,omitempty"`
	// FilesShifted are the files of both versions with an unchanged unit at
	// another pointer: lines were inserted or removed in the middle.
	FilesShifted []string `json:"files_shifted,omitempty"`
}

// Transition carries the translation state from oldKR, the KR pack the state
// belongs to, to newKR, the KR pack of the new snapshot, so that a unit
// whose KR text did not change keeps its translation even when its pointer
// did. Every file is matched by align.AlignVersions, and each string of the
// new version becomes:
//
//   - unchanged: its record and pending item move to the new key; the pending
//     item loses its lease, which was taken on the old version;
//   - changed: its record is dropped and the unit is queued as "kr-changed"
//     with the old KR and the old translation, keeping the Hint of a pending
//     item it had. With no record but a pending item that carries a previous
//     translation (the KR changed twice before the unit was retranslated),
//     the new item keeps that item's PrevKR and PrevZH instead. A changed
//     unit of a frozen file (frozen holds the patterns of lct.yaml) keeps
//     its record instead, with the old kr_sha, and is listed in
//     FrozenChanged; with no record to keep, it is queued like any other
//     changed unit;
//   - new: queued as "new".
//
// A string only the old version has is deleted: its record and pending item
// leave the state, and it is listed in Deleted with them. A file only the old
// version has is all deleted, and a file only the new version has is all new.
// State whose key is not a string of the old version has nowhere to go and is
// dropped.
//
// A moved record keeps its Since, the raw tag its translation was written
// for; a pending item has no version.
//
// Transition changes neither recs nor pending nor anything they hold; the
// result shares no memory with them.
func Transition(oldKR, newKR *pack.Pack, recs map[string]store.Record, pending []store.PendingItem, frozen []string) (*Moved, error) {
	m := &mover{
		recs:    recs,
		pending: make(map[string]store.PendingItem, len(pending)),
		frozen:  frozen,
		out:     &Moved{Records: make(map[string]store.Record, len(recs))},
	}
	for _, p := range pending {
		m.pending[p.Key] = p
	}

	files := slices.Concat(oldKR.Files(), newKR.Files())
	slices.Sort(files)
	for _, file := range slices.Compact(files) {
		oldDoc, err := docOf(oldKR, file)
		if err != nil {
			return nil, err
		}
		newDoc, err := docOf(newKR, file)
		if err != nil {
			return nil, err
		}
		m.file(file, oldDoc, newDoc)
	}

	sort.Slice(m.out.Pending, func(i, j int) bool { return m.out.Pending[i].Key < m.out.Pending[j].Key })
	slices.Sort(m.out.FrozenChanged)
	return m.out, nil
}

// docOf returns the document of file in p, or nil when p has no such file.
func docOf(p *pack.Pack, file string) (*jsonx.Node, error) {
	if !p.Has(file) {
		return nil, nil
	}
	doc, err := p.Doc(file)
	if err != nil {
		return nil, fmt.Errorf("plan: transition: %w", err)
	}
	return doc, nil
}

// mover holds the state being moved and the result being built.
type mover struct {
	recs    map[string]store.Record
	pending map[string]store.PendingItem // by key
	frozen  []string
	out     *Moved
}

// file moves the state of one file. A nil oldDoc is a file only the new
// version has, a nil newDoc one only the old version has.
func (m *mover) file(file string, oldDoc, newDoc *jsonx.Node) {
	c := &m.out.Changes
	frozen := config.MatchAny(m.frozen, file)
	oldUnits := make(map[string]units.Unit)
	for _, u := range units.Extract(file, oldDoc) {
		oldUnits[u.Pointer] = u
	}

	var changed, shifted bool
	for _, vm := range align.AlignVersions(file, oldDoc, newDoc) {
		newKey := units.Key(file, vm.NewPointer)
		var (
			old     units.Unit // what a string of the old version was
			rec     store.Record
			item    store.PendingItem
			hasRec  bool
			hasItem bool
		)
		if vm.Status != align.StatusNew {
			oldKey := units.Key(file, vm.OldPointer)
			old = oldUnits[vm.OldPointer]
			rec, hasRec = m.recs[oldKey]
			item, hasItem = m.pending[oldKey]
		}

		switch vm.Status {
		case align.StatusUnchanged:
			c.Unchanged++
			if vm.OldPointer != vm.NewPointer {
				c.Moved++
				shifted = true
			}
			if hasRec {
				m.out.Records[newKey] = moveRecord(rec, newKey)
			}
			if hasItem {
				item = clonePending(item)
				item.Key, item.Lease = newKey, nil
				m.out.Pending = append(m.out.Pending, item)
			}

		case align.StatusChanged:
			c.Changed++
			changed = true
			if frozen && hasRec {
				m.out.Records[newKey] = moveRecord(rec, newKey)
				m.out.FrozenChanged = append(m.out.FrozenChanged, newKey)
				continue
			}
			queued := store.PendingItem{Key: newKey, Reason: ReasonKRChanged, PrevKR: old.KR}
			switch {
			case hasRec:
				queued.PrevZH = rec.ZH
			case hasItem && item.PrevZH != "":
				// The KR changed again before the unit was retranslated:
				// keep the last translated pair, so PrevZH stays paired
				// with the KR it translated.
				queued.PrevKR, queued.PrevZH = item.PrevKR, item.PrevZH
			}
			if hasItem {
				queued.Hint = item.Hint
			}
			m.out.Pending = append(m.out.Pending, queued)

		case align.StatusNew:
			c.New++
			changed = true
			m.out.Pending = append(m.out.Pending, store.PendingItem{Key: newKey, Reason: ReasonNew})

		case align.StatusDeleted:
			c.Deleted++
			changed = true
			d := DeletedUnit{File: file, Field: old.Field, KR: old.KR, Pointer: vm.OldPointer}
			if hasRec {
				r := cloneRecord(rec)
				d.Record = &r
			}
			if hasItem {
				it := clonePending(item)
				d.Pending = &it
			}
			m.out.Deleted = append(m.out.Deleted, d)
		}
	}

	switch {
	case newDoc == nil:
		c.FilesDeleted = append(c.FilesDeleted, file)
	case oldDoc == nil:
		c.FilesAdded = append(c.FilesAdded, file)
	default:
		if changed {
			c.FilesChanged = append(c.FilesChanged, file)
		}
		if shifted {
			c.FilesShifted = append(c.FilesShifted, file)
		}
	}
}

// moveRecord returns a copy of r for key.
func moveRecord(r store.Record, key string) store.Record {
	r = cloneRecord(r)
	r.Key = key
	return r
}

func cloneRecord(r store.Record) store.Record {
	r.Flags = slices.Clone(r.Flags)
	return r
}

func clonePending(p store.PendingItem) store.PendingItem {
	p.Refs = maps.Clone(p.Refs)
	p.Flags = slices.Clone(p.Flags)
	if p.Lease != nil {
		l := *p.Lease
		p.Lease = &l
	}
	return p
}
