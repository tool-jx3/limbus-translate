// Package history compares the strings of two Releases: which are new, whose
// KR text changed, which are gone, and which only had their translation
// changed. It keeps no state of its own.
package history

import (
	"fmt"

	"github.com/tool-jx3/limbus-translate/internal/align"
	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// The kinds of change an Entry can be.
const (
	KindNew       = "new"        // only in the new Release
	KindKRChanged = "kr-changed" // the KR text of the string changed
	KindDeleted   = "deleted"    // only in the old Release
	KindZHChanged = "zh-changed" // same KR text, another translation
)

// The causes of a KindZHChanged entry.
const (
	CauseAdjustment   = "adjustment"   // the new origin is human
	CauseLate         = "late"         // the old origin was a stand-in or a fallback
	CauseRetranslated = "retranslated" // anything else
)

// OriginStandIn means what site.OriginStandIn does: the string has no record,
// or is not the record's. It is repeated here so that history need not import
// site.
const OriginStandIn = "stand-in"

// State is one Release: the KR snapshot, the built language pack (opened with
// an empty prefix) and the translation records.
type State struct {
	KR, ZH  *pack.Pack
	Records map[string]store.Record
}

// Entry is one change. A *string field is set when its side has the string,
// an empty string included.
type Entry struct {
	File       string  `json:"file"`
	Field      string  `json:"field"`
	Pointer    string  `json:"ptr"`               // the new one; the old one for a deleted string
	OldPointer string  `json:"old_ptr,omitempty"` // where the string was, if it moved
	Kind       string  `json:"kind"`
	OldKR      *string `json:"old_kr,omitempty"` // kr-changed and deleted only
	NewKR      *string `json:"new_kr,omitempty"` // new, kr-changed and zh-changed
	OldZH      *string `json:"old_zh,omitempty"` // kr-changed, deleted and zh-changed
	NewZH      *string `json:"new_zh,omitempty"` // new, kr-changed and zh-changed
	OldOrigin  string  `json:"old_origin,omitempty"`
	NewOrigin  string  `json:"new_origin,omitempty"`
	Copy       bool    `json:"copy,omitempty"`
	Cause      string  `json:"cause,omitempty"` // zh-changed only
}

// side is the strings of one file in one Release, by pointer.
type side struct {
	doc   *jsonx.Node
	units map[string]units.Unit
	zh    map[string]string
	state State
	file  string
}

func load(s State, file string) (side, error) {
	out := side{units: map[string]units.Unit{}, zh: map[string]string{}, state: s, file: file}
	if s.KR.Has(file) {
		doc, err := s.KR.Doc(file)
		if err != nil {
			return side{}, err
		}
		out.doc = doc
		for _, u := range units.Extract(file, doc) {
			out.units[u.Pointer] = u
		}
	}
	if s.ZH.Has(file) {
		doc, err := s.ZH.Doc(file)
		if err != nil {
			return side{}, err
		}
		for _, f := range doc.Strings() {
			out.zh[f.Pointer] = f.Value
		}
	}
	return out, nil
}

// origin follows the rule of the site: the record's origin where the built
// string is the record's, a stand-in otherwise.
func (s side) origin(pointer string) string {
	if rec, ok := s.state.Records[units.Key(s.file, pointer)]; ok && rec.ZH == s.zh[pointer] {
		return rec.Origin
	}
	return OriginStandIn
}

// Compare compares each of files (KR file names, such as "Passives.json") in
// the old and new Release and returns the changes, in the order of files. In a
// file they are in the order of align.AlignVersions: the strings of the new
// Release in document order, then the deleted ones. A file missing from a KR
// pack has no strings there; one missing from a built pack has empty ones.
func Compare(old, new State, files []string) ([]Entry, error) {
	var out []Entry
	for _, file := range files {
		o, err := load(old, file)
		if err != nil {
			return nil, fmt.Errorf("history: %s (old): %w", file, err)
		}
		n, err := load(new, file)
		if err != nil {
			return nil, fmt.Errorf("history: %s (new): %w", file, err)
		}
		for _, m := range align.AlignVersions(file, o.doc, n.doc) {
			if e, ok := entryOf(file, m, o, n); ok {
				out = append(out, e)
			}
		}
	}
	return out, nil
}

func entryOf(file string, m align.VersionMatch, o, n side) (Entry, bool) {
	e := Entry{File: file}
	str := func(s string) *string { return &s }
	switch m.Status {
	case align.StatusDeleted:
		u := o.units[m.OldPointer]
		e.Field, e.Pointer, e.Kind = u.Field, m.OldPointer, KindDeleted
		e.OldKR, e.OldZH = str(u.KR), str(o.zh[m.OldPointer])
		e.OldOrigin = o.origin(m.OldPointer)
		e.Copy = e.OldOrigin == store.OriginCopy
		return e, true
	case align.StatusNew:
		u := n.units[m.NewPointer]
		e.Field, e.Pointer, e.Kind = u.Field, m.NewPointer, KindNew
		e.NewKR, e.NewZH = str(u.KR), str(n.zh[m.NewPointer])
		e.NewOrigin = n.origin(m.NewPointer)
		e.Copy = e.NewOrigin == store.OriginCopy
		return e, true
	}

	ou, nu := o.units[m.OldPointer], n.units[m.NewPointer]
	e.Field, e.Pointer = nu.Field, m.NewPointer
	if m.OldPointer != m.NewPointer {
		e.OldPointer = m.OldPointer
	}
	e.OldZH, e.NewZH = str(o.zh[m.OldPointer]), str(n.zh[m.NewPointer])
	e.NewKR = str(nu.KR)
	e.OldOrigin, e.NewOrigin = o.origin(m.OldPointer), n.origin(m.NewPointer)
	e.Copy = e.NewOrigin == store.OriginCopy
	if m.Status == align.StatusChanged {
		e.Kind = KindKRChanged
		e.OldKR = str(ou.KR)
		return e, true
	}
	if *e.OldZH == *e.NewZH {
		return Entry{}, false
	}
	e.Kind = KindZHChanged
	switch {
	case e.NewOrigin == store.OriginHuman:
		e.Cause = CauseAdjustment
	case e.OldOrigin == OriginStandIn || e.OldOrigin == store.OriginFallback:
		e.Cause = CauseLate
	default:
		e.Cause = CauseRetranslated
	}
	return e, true
}
