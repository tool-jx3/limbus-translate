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
// A batch holds units of one file: the first MaxUnits eligible units of the
// file in document order, or for a story file (StoryData/ and the RPG
// dialogue, see IsStory) the first StoryMaxUnits. The units need not be next
// to each other, and a pending line that is not eligible is skipped. The next
// batch of the file starts at the first eligible unit left. A story batch comes
// with Context, translated lines near its units.
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

// Claimable reports whether a pending item may be handed out at now: it is not
// leased (or its lease has run out) and not flagged needs_review, which marks
// an item that waits for a person. It is the one definition of an item a batch
// can take; Next, Available and CountClaimable (so lct plan and lct translate)
// all rest on it. That the unit is still in KR is checked apart, by whoever
// has KR at hand.
func Claimable(it store.PendingItem, now time.Time) bool {
	return store.LeaseFree(it, now) && !hasFlag(it.Flags, store.FlagNeedsReview)
}

// CountClaimable counts the items of pending that a batch can take at now:
// the claimable ones whose unit inKR says is still in KR. inKR is asked only
// about claimable items.
func CountClaimable(pending []store.PendingItem, now time.Time, inKR func(key string) bool) int {
	n := 0
	for _, it := range pending {
		if Claimable(it, now) && inKR(it.Key) {
			n++
		}
	}
	return n
}

// Available counts the units Next could hand out at now: it reads the pending
// queue of env.Store and counts the claimable items whose file is in the KR
// pack of env and whose pointer names a unit of that file. It takes no lock; the
// count is what the queue held when it was read.
func Available(env Env, now time.Time) (int, error) {
	if err := env.check(); err != nil {
		return 0, err
	}
	pending, err := env.Store.LoadPending()
	if err != nil {
		return 0, err
	}
	ld := newLoader(env)
	var loadErr error // the first KR file that could not be read
	n := CountClaimable(pending, now, func(key string) bool {
		if loadErr != nil {
			return false
		}
		file, pointer, err := units.SplitKey(key) // LoadPending checked the keys
		if err != nil {
			return false
		}
		fd, err := ld.file(file)
		if err != nil {
			loadErr = err
			return false
		}
		if fd == nil {
			return false
		}
		_, ok := fd.index[pointer]
		return ok
	})
	if loadErr != nil {
		return 0, loadErr
	}
	return n, nil
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
		if slices.ContainsFunc(idxs, func(i int) bool { return Claimable(pending[i], now) }) {
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
	busy, err := busyTwins(env, ld, pending, now)
	if err != nil {
		return nil, nil, err
	}
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

		limit := opt.MaxUnits
		if IsStory(c.file) {
			limit = opt.StoryMaxUnits
		}
		twins := newTwinFilter(env, fd, busy)
		chosen := firstEligible(slots, func(s slot) bool {
			return Claimable(pending[s.idx], now) && twins.pick(s.pos, pending[s.idx])
		}, limit)
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
	needRecords := IsStory(fd.file)
	for _, s := range chosen {
		item, u := pending[s.idx], fd.units[s.pos]
		who, known := speaker(env.Rules, fd.doc, u)
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
			Hint:    item.Hint,
			EN:      en,
			JP:      jp,
			Refs:    cleanRefs(item.Refs),
			Keep:    constraints(u.KR),
		})
		krs = append(krs, u.KR)
		needRecords = needRecords || len(b.Units[len(b.Units)-1].Keep.IDs) > 0
	}
	b.Guides = env.Rules.GuidesFor(fd.file, speakers)
	b.Glossary = glossaryFor(env.Rules, fd.file, krs)

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
	if IsStory(fd.file) {
		pendingByKey := make(map[string]store.PendingItem, len(filePending))
		for _, i := range filePending {
			pendingByKey[pending[i].Key] = pending[i]
		}
		b.Context = storyContext(env, fd, recs, pendingByKey, chosen)
	}
	return b, nil
}

// contextReach is how many dataList elements on each side of the element of
// a unit are looked at for the context of a story batch of n units: as many
// as the batch has units, so that a batch of one unit does not come with more
// context than text to translate, and at most contextElements.
func contextReach(n int) int { return min(contextElements, max(1, n)) }

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

// storyContext returns the translated strings near the units of a story batch:
// those of every dataList element (a line of a story, or a group of texts of an
// RPG dialogue) up to contextReach(len(chosen)) before or after the element of
// any unit of the batch, each once, in document order. A string is shown if it
// has a record that is not a copy and was made from the KR text it has now,
// and it does not wait for translation. So the units of the batch are never
// shown, and neither is a string that needs review. Strings outside the
// dataList belong to no element and are not looked at.
func storyContext(env Env, fd *fileData, recs map[string]store.Record, pendingByKey map[string]store.PendingItem, chosen []slot) []ContextLine {
	reach := contextReach(len(chosen))
	near := map[int]bool{} // the elements in reach
	for _, s := range chosen {
		if at := fd.units[s.pos].ItemIndex; at >= 0 {
			for i := at - reach; i <= at+reach; i++ {
				near[i] = true
			}
		}
	}

	lines := []ContextLine{}
	for _, u := range fd.units {
		if u.ItemIndex < 0 || !near[u.ItemIndex] {
			continue
		}
		key := units.Key(u.File, u.Pointer)
		if _, waits := pendingByKey[key]; waits {
			continue
		}
		rec, ok := recs[key]
		if !ok || rec.Origin == store.OriginCopy || rec.ZH == "" || rec.KRSha != units.KRSha(u.KR) {
			continue
		}
		id, _ := speaker(env.Rules, fd.doc, u)
		lines = append(lines, ContextLine{Key: key, Speaker: id, KR: u.KR, ZH: rec.ZH})
	}
	return lines
}
