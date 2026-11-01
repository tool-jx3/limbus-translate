package batch

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// NextOptions limit one batch. A zero limit means the Default value.
type NextOptions struct {
	// Family restricts the batch to one file family; "" means any.
	Family string
	// MaxUnits is the most units of a batch outside StoryData/, StoryMaxUnits
	// the most of a story batch.
	MaxUnits, StoryMaxUnits int
	// LeaseTTL is how long the units stay reserved for the batch.
	LeaseTTL time.Duration
	// IDTaken, when set, tells whether a batch id is already in use outside
	// the pending queue (such as by a batch file). An error, when it cannot
	// tell, makes Next fail. It is called inside the store lock, so it must
	// not use the store.
	IDTaken func(id string) (bool, error)
}

func (o NextOptions) withDefaults() (NextOptions, error) {
	if o.MaxUnits < 0 || o.StoryMaxUnits < 0 || o.LeaseTTL < 0 {
		return o, errors.New("batch: limits must not be negative")
	}
	if o.MaxUnits == 0 {
		o.MaxUnits = DefaultMaxUnits
	}
	if o.StoryMaxUnits == 0 {
		o.StoryMaxUnits = DefaultStoryMaxUnits
	}
	if o.LeaseTTL == 0 {
		o.LeaseTTL = DefaultLeaseTTL
	}
	return o, nil
}

// Next cuts one batch from the pending queue and leases its units. Units are
// handed out by the priority of their file family, then file name, then
// document order. Only units that are not leased and not flagged needs_review
// are eligible; units whose file or pointer has left KR are left alone.
//
// The id of a batch is the UTC time and the start of the hash of its first
// key (see batchID). If a lease already names that id, or IDTaken says it is
// in use, "-2", "-3" and so on is appended until the id is unique; after
// maxIDSuffixes suffixes Next gives up with an error.
//
// A batch holds units of one file. Outside StoryData/ that is the first
// MaxUnits eligible units of the file. A story batch is a run of consecutive
// eligible units, at most StoryMaxUnits: translated lines in between do not
// end the run, a pending line that is not eligible does. It comes with
// Context, translated lines around and between its units.
//
// It returns nil, nil when there is nothing to hand out. The whole cut is one
// store lock, so concurrent calls never share a unit.
func Next(env Env, opt NextOptions) (*Batch, error) {
	if err := env.check(); err != nil {
		return nil, err
	}
	opt, err := opt.withDefaults()
	if err != nil {
		return nil, err
	}
	now := env.now()

	var batch *Batch
	err = env.Store.WithLock(func() error {
		pending, err := env.Store.LoadPending()
		if err != nil {
			return err
		}
		b, leased, err := cut(env, opt, now, pending)
		if err != nil || b == nil {
			return err
		}
		lease := store.Lease{Batch: b.ID, Until: now.Add(opt.LeaseTTL)}
		for _, i := range leased {
			l := lease
			pending[i].Lease = &l
		}
		if err := env.Store.SavePending(pending); err != nil {
			return err
		}
		batch = b
		return nil
	})
	if err != nil {
		return nil, err
	}
	return batch, nil
}

// slot is a pending item that is still in KR: its position among the units of
// its file and its index in the pending list.
type slot struct{ pos, idx int }

// eligible reports whether a pending item may be handed out.
func eligible(it store.PendingItem, now time.Time) bool {
	return store.LeaseFree(it, now) && !hasFlag(it.Flags, store.FlagNeedsReview)
}

// cut chooses the next batch and returns it with the pending indexes of its
// units. It changes nothing.
func cut(env Env, opt NextOptions, now time.Time, pending []store.PendingItem) (*Batch, []int, error) {
	byFile := map[string][]int{}
	for i, it := range pending {
		if file, _, err := units.SplitKey(it.Key); err == nil { // LoadPending checked the keys
			byFile[file] = append(byFile[file], i)
		}
	}
	type candidate struct {
		file   string
		family string
		prio   int
	}
	var files []candidate
	for file, idxs := range byFile {
		family := env.Rules.Family(file)
		if !env.Src.KR.Has(file) || opt.Family != "" && family != opt.Family {
			continue
		}
		if slices.ContainsFunc(idxs, func(i int) bool { return eligible(pending[i], now) }) {
			files = append(files, candidate{file, family, env.Rules.Priority(family)})
		}
	}
	slices.SortFunc(files, func(a, b candidate) int {
		if a.prio != b.prio {
			return a.prio - b.prio
		}
		return strings.Compare(a.file, b.file)
	})

	ld := newLoader(env)
	var recs map[string]store.Record
	records := func() (map[string]store.Record, error) {
		if recs == nil {
			r, err := env.Store.LoadRecords()
			if err != nil {
				return nil, err
			}
			recs = r
		}
		return recs, nil
	}

	for _, c := range files {
		fd, err := ld.file(c.file)
		if err != nil {
			return nil, nil, err
		}
		var slots []slot // in document order
		for _, i := range byFile[c.file] {
			_, pointer, _ := units.SplitKey(pending[i].Key)
			if pos, ok := fd.index[pointer]; ok {
				slots = append(slots, slot{pos, i})
			}
		}
		slices.SortFunc(slots, func(a, b slot) int { return a.pos - b.pos })

		ok := func(s slot) bool { return eligible(pending[s.idx], now) }
		var chosen []slot
		if isStory(c.file) {
			chosen = storyRun(slots, ok, opt.StoryMaxUnits)
		} else {
			chosen = firstEligible(slots, ok, opt.MaxUnits)
		}
		if len(chosen) == 0 {
			continue
		}
		id, err := uniqueID(batchID(now, pending[chosen[0].idx].Key), pending, opt.IDTaken)
		if err != nil {
			return nil, nil, err
		}
		b, err := assemble(env, fd, id, c.family, pending, byFile[c.file], chosen, records)
		if err != nil {
			return nil, nil, err
		}
		leased := make([]int, len(chosen))
		for i, s := range chosen {
			leased[i] = s.idx
		}
		return b, leased, nil
	}
	return nil, nil, nil
}

// firstEligible returns the first max slots that ok accepts.
func firstEligible(slots []slot, ok func(slot) bool, max int) []slot {
	var out []slot
	for _, s := range slots {
		if len(out) == max {
			break
		}
		if ok(s) {
			out = append(out, s)
		}
	}
	return out
}

// storyRun returns the first run of slots that ok accepts, at most max. Slots
// before the run that ok refuses are skipped; the first one after it ends it.
func storyRun(slots []slot, ok func(slot) bool, max int) []slot {
	var run []slot
	for _, s := range slots {
		switch {
		case !ok(s) && len(run) > 0:
			return run
		case !ok(s):
			// not started yet
		case len(run) == max:
			return run
		default:
			run = append(run, s)
		}
	}
	return run
}

// batchID is the UTC time and the start of the hash of the first key, so
// batches cut in one second still differ.
func batchID(now time.Time, firstKey string) string {
	sum := sha256.Sum256([]byte(firstKey))
	return now.UTC().Format("20060102T150405") + "-" + hex.EncodeToString(sum[:])[:8]
}

// maxIDSuffixes is how many suffixes uniqueID tries after the plain id.
const maxIDSuffixes = 1000

// uniqueID returns id, or id with "-2", "-3"... appended, such that no lease
// of pending (live or not: a batch that is still working under an old id must
// not be mistaken for the new one) and no taken(id) holds it. It fails when
// taken fails or the last of maxIDSuffixes suffixes is taken too.
func uniqueID(id string, pending []store.PendingItem, taken func(string) (bool, error)) (string, error) {
	held := map[string]bool{}
	for _, it := range pending {
		if it.Lease != nil {
			held[it.Lease.Batch] = true
		}
	}
	for n := 1; n <= 1+maxIDSuffixes; n++ { // id, then id-2 ... id-1001
		candidate := id
		if n > 1 {
			candidate += "-" + strconv.Itoa(n)
		}
		inUse := held[candidate]
		if !inUse && taken != nil {
			var err error
			if inUse, err = taken(candidate); err != nil {
				return "", fmt.Errorf("batch: check batch id %s: %w", candidate, err)
			}
		}
		if !inUse {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("batch: no free batch id: %s and its %d suffixes are all in use", id, maxIDSuffixes)
}

// assemble builds the batch of the chosen slots. filePending holds the
// pending indexes of the whole file; records loads the stored translations
// (only story batches and batches with [ID]s need them).
func assemble(
	env Env, fd *fileData, id, family string,
	pending []store.PendingItem, filePending []int, chosen []slot,
	records func() (map[string]store.Record, error),
) (*Batch, error) {
	b := &Batch{
		ID:      id,
		Family:  family,
		File:    fd.file,
		Units:   make([]BatchUnit, 0, len(chosen)),
		Context: []ContextLine{},
	}
	var krs, speakers []string
	needRecords := isStory(fd.file)
	for _, s := range chosen {
		item, u := pending[s.idx], fd.units[s.pos]
		who, known := speaker(env.Rules, u)
		if known && !slices.Contains(speakers, who) {
			speakers = append(speakers, who)
		}
		en, err := fd.aligned(langEN, u.Pointer)
		if err != nil {
			return nil, err
		}
		jp, err := fd.aligned(langJP, u.Pointer)
		if err != nil {
			return nil, err
		}
		b.Units = append(b.Units, BatchUnit{
			Key:     item.Key,
			Field:   u.Field,
			Speaker: who,
			KR:      u.KR,
			Reason:  item.Reason,
			PrevKR:  item.PrevKR,
			PrevZH:  item.PrevZH,
			Draft:   item.Draft,
			EN:      en,
			JP:      jp,
			Refs:    cleanRefs(item.Refs),
			Keep:    constraints(u.KR),
		})
		krs = append(krs, u.KR)
		needRecords = needRecords || len(b.Units[len(b.Units)-1].Keep.IDs) > 0
	}
	b.Guides = env.Rules.GuidesFor(fd.file, speakers)
	b.Glossary = glossaryFor(env.Rules, krs)

	b.Keywords = map[string]string{}
	if !needRecords {
		return b, nil
	}
	recs, err := records()
	if err != nil {
		return nil, err
	}
	if b.Keywords, err = keywordNames(env, recs, krs); err != nil {
		return nil, err
	}
	if isStory(fd.file) {
		pendingByKey := make(map[string]store.PendingItem, len(filePending))
		for _, i := range filePending {
			pendingByKey[pending[i].Key] = pending[i]
		}
		b.Context = storyContext(env, fd, recs, pendingByKey, chosen)
	}
	return b, nil
}

// cleanRefs drops the empty references; nil when none is left.
func cleanRefs(refs map[string]string) map[string]string {
	var out map[string]string
	for name, text := range refs {
		if text == "" {
			continue
		}
		if out == nil {
			out = map[string]string{}
		}
		out[name] = text
	}
	return out
}

// What a story line is to the context of a batch.
const (
	lineSkip    = iota // no translation to show
	lineShow           // a translated line
	lineBlocked        // waits for translation: the translated text ends here
)

// storyContext returns the translated lines around and between the units of a
// story batch, in document order: up to contextLines before the first unit
// and after the last, and every one between them. Walking outwards stops at
// the first line that waits for translation, since nothing translated lies
// right next to the batch beyond it. A line that needs review does not wait
// for the translation that is being made now, so it is skipped like the lines
// that are not translations (copied identifiers, the fallback text of a
// pending unit, a record made from another KR text).
func storyContext(env Env, fd *fileData, recs map[string]store.Record, pendingByKey map[string]store.PendingItem, chosen []slot) []ContextLine {
	classify := func(pos int) (ContextLine, int) {
		u := fd.units[pos]
		key := units.Key(u.File, u.Pointer)
		if it, waits := pendingByKey[key]; waits {
			if hasFlag(it.Flags, store.FlagNeedsReview) {
				return ContextLine{}, lineSkip
			}
			return ContextLine{}, lineBlocked
		}
		rec, ok := recs[key]
		if !ok || rec.Origin == store.OriginCopy || rec.ZH == "" || rec.KRSha != units.KRSha(u.KR) {
			return ContextLine{}, lineSkip
		}
		id, _ := speaker(env.Rules, u)
		return ContextLine{Key: key, Speaker: id, KR: u.KR, ZH: rec.ZH}, lineShow
	}

	inBatch := make(map[int]bool, len(chosen))
	for _, s := range chosen {
		inBatch[s.pos] = true
	}
	first, last := chosen[0].pos, chosen[len(chosen)-1].pos

	var before []ContextLine
walkBack:
	for p := first - 1; p >= 0 && len(before) < contextLines; p-- {
		switch l, kind := classify(p); kind {
		case lineBlocked:
			break walkBack
		case lineShow:
			before = append(before, l)
		}
	}
	slices.Reverse(before)

	lines := before
	for p := first + 1; p < last; p++ {
		if inBatch[p] {
			continue
		}
		if l, kind := classify(p); kind == lineShow {
			lines = append(lines, l)
		}
	}

	shown := 0
walkOn:
	for p := last + 1; p < len(fd.units) && shown < contextLines; p++ {
		switch l, kind := classify(p); kind {
		case lineBlocked:
			break walkOn
		case lineShow:
			lines = append(lines, l)
			shown++
		}
	}
	if lines == nil {
		return []ContextLine{}
	}
	return lines
}
