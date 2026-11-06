package batch

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/requeue"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

const (
	// ReasonNew is the Reason of a pending item for a unit that has no
	// translation yet (see plan.Transition); the preview gives it to a unit
	// that is neither pending nor translated.
	ReasonNew = "new"
	// PreviewID is the id of the batch of a preview. It is no real batch id,
	// so a preview is never taken for a batch that was handed out.
	PreviewID = "preview"
)

var (
	// ErrNotInKR is returned for a unit whose file or pointer is not in the KR
	// pack: there is nothing to translate.
	ErrNotInKR = errors.New("the unit is not in KR")
	// ErrCopy is returned for a unit whose record is a copy: an identifier or
	// a text that is the same in every language is never sent for translation.
	ErrCopy = errors.New("the unit is a copy")
)

// PreviewState is where a unit stands in the translation queue, whether or
// not it would really be handed out: a preview shows the batch of a unit
// that waits for a person, is leased or is frozen all the same.
type PreviewState struct {
	// Pending is true when the unit is in the pending queue.
	Pending bool `json:"pending"`
	// NeedsReview is true when the pending item waits for a person, so that
	// Next does not hand it out.
	NeedsReview bool `json:"needs_review"`
	// LeasedTo is the batch the pending item is leased to, expired or not.
	LeasedTo string `json:"leased_to,omitempty"`
	// Frozen is true when the file of the unit is frozen: it is not sent for
	// translation.
	Frozen bool `json:"frozen"`
	// Attempts and LastError are those of the pending item.
	Attempts  int    `json:"attempts,omitempty"`
	LastError string `json:"last_error,omitempty"`
	// RulesCommit names the rules the preview was made under.
	RulesCommit string `json:"rules_commit"`
}

// Check is how the current translation of a unit fares under the submit
// check (see Judge).
type Check struct {
	// Verdict is "reject" when an issue is hard, else "warn" when there are
	// issues, else "accept".
	Verdict string `json:"verdict"`
	// ZH is the translation after normalization, the text that was checked.
	ZH string `json:"zh"`
	// Origin is the origin of the record.
	Origin string `json:"origin"`
	// StaleKR is true when the record was made from another KR text than the
	// one there is now.
	StaleKR bool             `json:"stale_kr"`
	Issues  []validate.Issue `json:"issues"`
}

// UnitPreview is what happens to one unit if it is sent for translation by
// itself.
type UnitPreview struct {
	// Batch is the batch Next would cut for the unit alone.
	Batch *Batch       `json:"batch"`
	State PreviewState `json:"state"`
	// Pending is the item the batch was made from: the pending item of the
	// unit, or the one a requeue (a translated unit) or the plan (a new unit)
	// would make.
	Pending store.PendingItem `json:"pending"`
	// Record is the stored translation, nil if there is none.
	Record *store.Record `json:"record"`
	// Notes are the translator's notes made for the KR text there is now.
	Notes []store.Note `json:"notes"`
	// Check is the submit check of the record, nil if there is none.
	Check *Check `json:"check"`
}

// Previewer works out what a unit would get if it were sent for translation
// alone, without changing anything: it reads the store once and never
// writes to it. It shares its code with Next (assemble) and Submit (Judge), so
// the preview cannot differ from the real thing.
//
// A Previewer holds no KR file between calls: each Unit and File opens the
// files it needs and lets them go, so that a pass over every file of the pack
// does not keep them all in memory. It is not safe for concurrent use.
type Previewer struct {
	env     Env
	frozen  []string
	pending []store.PendingItem // the real queue, with room for one more item
	byKey   map[string]int      // key -> index in pending
	byFile  map[string][]int    // file -> indexes in pending
	records map[string]store.Record
	notes   map[string][]store.Note // key -> its notes, in store order

	keywords    map[string]string // the keyword table, made on first use
	keywordsErr error
	keywordsOK  bool
}

// NewPreviewer reads the pending queue, the records and the notes of env.Store
// once. frozen are the patterns of the frozen files (config.MatchAny).
func NewPreviewer(env Env, frozen []string) (*Previewer, error) {
	if err := env.check(); err != nil {
		return nil, err
	}
	recs, err := env.Store.LoadRecords()
	if err != nil {
		return nil, err
	}
	return NewPreviewerWith(env, frozen, recs)
}

// NewPreviewerWith is NewPreviewer for a caller that has read the records
// already (store.Store.LoadRecords); the Previewer reads them no more and does
// not change the map.
func NewPreviewerWith(env Env, frozen []string, recs map[string]store.Record) (*Previewer, error) {
	if err := env.check(); err != nil {
		return nil, err
	}
	if env.Scripts == nil {
		env.Scripts = rules.NewScriptCache(env.Rules)
	}
	pending, err := env.Store.LoadPending()
	if err != nil {
		return nil, err
	}
	notes, err := env.Store.LoadNotes()
	if err != nil {
		return nil, err
	}
	p := &Previewer{
		env:     env,
		frozen:  frozen,
		pending: append(make([]store.PendingItem, 0, len(pending)+1), pending...),
		byKey:   make(map[string]int, len(pending)),
		byFile:  map[string][]int{},
		records: recs,
		notes:   make(map[string][]store.Note),
	}
	for _, n := range notes {
		p.notes[n.Key] = append(p.notes[n.Key], n)
	}
	for i, it := range pending {
		p.byKey[it.Key] = i
		if file, _, err := units.SplitKey(it.Key); err == nil { // LoadPending checked the keys
			p.byFile[file] = append(p.byFile[file], i)
		}
	}
	return p, nil
}

// Unit previews the unit of key. It returns an error wrapping ErrNotInKR if
// the key is malformed or its file or pointer is not in KR, and one wrapping ErrCopy if the record
// of the unit is a copy.
func (p *Previewer) Unit(key string) (*UnitPreview, error) {
	file, pointer, err := units.SplitKey(key)
	if err != nil {
		// A key that cannot be split names no unit of KR.
		return nil, fmt.Errorf("batch: preview: %w: %w", ErrNotInKR, err)
	}
	fd, err := newLoader(p.env).file(file)
	if err != nil {
		return nil, err
	}
	if fd == nil {
		return nil, fmt.Errorf("batch: preview %s: %w", key, ErrNotInKR)
	}
	pos, ok := fd.index[pointer]
	if !ok {
		return nil, fmt.Errorf("batch: preview %s: %w", key, ErrNotInKR)
	}
	if p.records[key].Origin == store.OriginCopy {
		return nil, fmt.Errorf("batch: preview %s: %w", key, ErrCopy)
	}
	return p.unit(fd, pos)
}

// File previews every unit of a KR file, in document order. The entry of a
// copy is nil. The file is opened for this call alone (a Previewer keeps no
// file data), and ErrNotInKR is returned if the pack has no such file.
func (p *Previewer) File(path string) ([]*UnitPreview, error) {
	fd, err := newLoader(p.env).file(path)
	if err != nil {
		return nil, err
	}
	if fd == nil {
		return nil, fmt.Errorf("batch: preview %s: %w", path, ErrNotInKR)
	}
	out := make([]*UnitPreview, len(fd.units))
	for pos, u := range fd.units {
		if p.records[units.Key(path, u.Pointer)].Origin == store.OriginCopy {
			continue
		}
		if out[pos], err = p.unit(fd, pos); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// unit previews the unit at pos of fd, which is not a copy.
func (p *Previewer) unit(fd *fileData, pos int) (*UnitPreview, error) {
	u := fd.units[pos]
	key := units.Key(fd.file, u.Pointer)
	rec, hasRec := p.records[key]
	real, isPending := p.byKey[key]

	// The item the unit would be handed out with: the one it has, or the one
	// that would be made for it.
	var item store.PendingItem
	switch {
	case isPending:
		item = p.pending[real]
	case hasRec:
		item = requeue.Item(key, rec, "")
	default:
		item = store.PendingItem{Key: key, Reason: ReasonNew}
	}

	// The queue the batch is cut from: the real one, and the item of the
	// unit at its end when it has none. The room for it was made by
	// NewPreviewer, so no unit copies the queue.
	idx, pending, byFile := real, p.pending, p.byFile
	if !isPending {
		idx = len(p.pending)
		pending = append(p.pending, item)
		byFile = map[string][]int{fd.file: append(slices.Clone(p.byFile[fd.file]), idx)}
	}
	b, err := assemble(p.env, PreviewID, p.env.Rules.Family(fd.file), pending, byFile,
		[]part{{fd, []slot{{pos, idx}}}}, p.loadedRecords, p.keywordTable)
	if err != nil {
		return nil, err
	}

	state := PreviewState{
		Pending:     isPending,
		NeedsReview: hasFlag(item.Flags, store.FlagNeedsReview),
		Frozen:      config.MatchAny(p.frozen, fd.file),
		Attempts:    item.Attempts,
		LastError:   item.LastError,
		RulesCommit: p.env.RulesCommit,
	}
	if item.Lease != nil {
		state.LeasedTo = item.Lease.Batch
	}
	pv := &UnitPreview{Batch: b, State: state, Pending: item, Notes: []store.Note{}}
	krSha := units.KRSha(u.KR)
	var noteTexts []string
	for _, n := range p.notes[key] {
		if n.KRSha == krSha {
			pv.Notes = append(pv.Notes, n)
			noteTexts = append(noteTexts, n.Note)
		}
	}
	if !hasRec {
		return pv, nil
	}
	pv.Record = &rec
	v := Judge(p.env, provisionalOf(b.Glossary, p.env.Provisional), fd.file, u.KR, rec.ZH, strings.Join(noteTexts, "\n"))
	c := &Check{Verdict: "accept", ZH: v.ZH, Origin: rec.Origin, StaleKR: rec.KRSha != krSha, Issues: v.Issues}
	if c.Issues == nil {
		c.Issues = []validate.Issue{}
	}
	switch {
	case v.Rejected:
		c.Verdict = "reject"
	case len(v.Issues) > 0:
		c.Verdict = "warn"
	}
	pv.Check = c
	return pv, nil
}

// loadedRecords is the records for assemble: those the Previewer has read.
func (p *Previewer) loadedRecords() (map[string]store.Record, error) { return p.records, nil }

// keywordTable is the keyword table of the records of the Previewer, made on
// the first call and kept: it reads every keyword file, too much to do for
// each unit of a pass over the pack.
func (p *Previewer) keywordTable(recs map[string]store.Record) (map[string]string, error) {
	if !p.keywordsOK {
		p.keywords, p.keywordsErr = keywordTable(p.env.Src.KR, recs)
		p.keywordsOK = true
	}
	return p.keywords, p.keywordsErr
}
