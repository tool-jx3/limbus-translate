package batch

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// Codes of the issues Submit adds to those of validate.Check.
const (
	// CodeForeignName is a warning: the translation uses a name that only
	// the LTM translation uses.
	CodeForeignName = "foreign-name"
	// CodeLeaseLost: the unit is no longer pending, or it is not leased to
	// the batch (its lease was released or belongs to another batch now).
	CodeLeaseLost = "lease-lost"
	// CodeNotInBatch: the key is not one of the batch's units.
	CodeNotInBatch = "not-in-batch"
	// CodeDuplicateKey: the key was already answered in this submission.
	CodeDuplicateKey = "duplicate-key"
	// CodeKRMissing: the unit has left KR since the batch was cut.
	CodeKRMissing = "kr-missing"
	// CodeKRChanged: the KR text of the unit is not the one the batch was
	// cut with, so the translation answers an old text.
	CodeKRChanged = "kr-changed"
	// CodeNeedsReview: the unit ran out of attempts and waits for a person.
	CodeNeedsReview = "needs-review"
)

// ErrBadKey is returned by Submit when the batch or its out file holds a key
// that is not a unit key. Such a file is not the translator's answer to the
// batch; Submit has written nothing.
var ErrBadKey = errors.New("batch: invalid key")

// SubmitResult says what Submit did with the units it was given. The lists
// are never nil.
type SubmitResult struct {
	// Accepted are the keys whose translation was recorded.
	Accepted []string `json:"accepted"`
	// Reused are the keys of pending units that were not submitted and got
	// the translation of an accepted unit with the same text (twins).
	Reused []string `json:"reused"`
	// Rejected are the units that were not recorded, with the reasons.
	Rejected []Rejection `json:"rejected"`
	// Warnings are issues of accepted units.
	Warnings []Rejection `json:"warnings"`
	// NeedsReview are the rejected units that ran out of attempts and now
	// wait for a person: they stand in with their fallback text, or keep the
	// record they had when that is a translation of the current KR text.
	NeedsReview []string `json:"needs_review"`
	// Released are the units of the batch that were not submitted and were
	// still leased to it: their lease ended and they are free again.
	Released []string `json:"released"`
	// Notes is how many translator notes were added to .meta/notes.jsonl.
	// A note already stored with the same key, batch and text is not added
	// or counted again.
	Notes int `json:"notes"`
	// Terms is how many provisional terms were newly kept in
	// .meta/terms.jsonl.
	Terms int `json:"terms"`
	// TermsSkipped are the terms of accepted units that were not kept.
	TermsSkipped []SkippedTerm `json:"terms_skipped"`
}

// Rejection is a unit with the issues found in it. It is written as
// {key, errors} (spec 9.1); the warnings of accepted units have the same form.
type Rejection struct {
	Key    string           `json:"key"`
	Issues []validate.Issue `json:"errors"`
}

// protocolIssue is a hard issue about how a unit was submitted, not about its
// text.
func protocolIssue(code, format string, args ...any) validate.Issue {
	return validate.Issue{
		Code:     code,
		Severity: validate.SeverityHard,
		Class:    validate.ClassStructural,
		Message:  fmt.Sprintf(format, args...),
	}
}

// foreignNameIssues warns about each of names that zh contains, except the
// names on the ignore list.
func foreignNameIssues(zh string, names, ignore []string) []validate.Issue {
	var issues []validate.Issue
	for _, name := range names {
		if name != "" && strings.Contains(zh, name) && !slices.Contains(ignore, name) {
			issues = append(issues, validate.Issue{
				Code:     CodeForeignName,
				Severity: validate.SeverityWarn,
				Class:    validate.ClassContent,
				Message:  fmt.Sprintf("the translation uses %q, a name only the LTM translation uses; use the name this translation gives the speaker", name),
			})
		}
	}
	return issues
}

// Submit takes the translations out for batch b.
//
// Only units of b are processed, each once; the others are rejected. A unit
// is processed while it is still pending, it is not flagged needs_review, it
// is leased to b (the lease may have run out, but no other batch may hold
// it), and its KR text is the one b was cut with. Its text is
// normalized (validate.Normalize, then the space after the keyword names of
// env.KeywordNames) and checked as a new translation in the file of its key,
// whichever of the files of b that is. Without a hard issue it
// becomes a record (origin claude) and leaves the queue. With one, its attempts grow
// and it keeps its lease, so the owner of b can send the fix; at maxAttempts
// the lease ends and the unit gets a fallback record (the EN text, else KR)
// flagged needs_review and stays in the queue flagged. A record that is a
// translation of the KR text there is now (see keepsRecord) is not replaced by
// the fallback: its text stays as it is, and the record and the queue item are
// flagged needs_review.
//
// A unit that is accepted passes its translation on to its twins, the pending
// units of the same text (see reuseForTwins); they are listed in Reused.
//
// A unit that b already got accepted is accepted again without being written
// again, so the whole out file can be sent once more. Units of b that are not
// in out lose b's lease. A unit that left KR, or whose KR changed, is
// rejected without counting an attempt and loses b's lease.
//
// The note of every unit of b in out, accepted or not, is kept in
// .meta/notes.jsonl with the batch, the sha of the KR text b was cut with and
// the model (see store.AppendNotes); notes on keys outside b are dropped.
//
// Every key of the batch and of out must be a unit key (ErrBadKey), checked
// before anything is written.
//
// Everything happens in one store lock. maxAttempts 0 means DefaultMaxAttempts.
func Submit(env Env, b *Batch, out []OutUnit, model string, maxAttempts int) (*SubmitResult, error) {
	if err := env.check(); err != nil {
		return nil, err
	}
	if b == nil || b.ID == "" {
		return nil, errors.New("batch: no batch to submit")
	}
	if maxAttempts < 0 {
		return nil, errors.New("batch: maxAttempts must not be negative")
	}
	if maxAttempts == 0 {
		maxAttempts = DefaultMaxAttempts
	}

	// Every key is checked before anything is written, notes included: a file
	// with a key that is no key is refused whole.
	if err := checkKeys(b, out); err != nil {
		return nil, err
	}

	res := &SubmitResult{
		Accepted: []string{}, Reused: []string{}, Rejected: []Rejection{}, Warnings: []Rejection{}, NeedsReview: []string{}, Released: []string{}, TermsSkipped: []SkippedTerm{},
	}
	err := env.Store.WithLock(func() error {
		records, err := env.Store.LoadRecords()
		if err != nil {
			return err
		}
		pending, err := env.Store.LoadPending()
		if err != nil {
			return err
		}
		s := &submission{
			env: env, batch: b, model: model, maxAttempts: maxAttempts,
			records: records, pending: pending, res: res,
			index:   make(map[string]int, len(pending)),
			batchKR: make(map[string]string, len(b.Units)),
			seen:    map[string]bool{},
			removed: map[int]bool{},
			ld:      newLoader(env),
		}
		for i, it := range pending {
			s.index[it.Key] = i
		}
		for _, u := range b.Units {
			s.batchKR[u.Key] = u.KR
		}
		for _, o := range out {
			if err := s.unit(o); err != nil {
				return err
			}
		}
		if err := s.reuseForTwins(); err != nil {
			return err
		}
		s.releaseOmitted()

		// Notes and terms first: storing one twice is harmless, and a failure
		// here leaves the rest unchanged.
		if res.Notes, err = env.Store.AppendNotes(s.notes); err != nil {
			return err
		}
		if res.Terms, err = s.keepTerms(); err != nil {
			return err
		}
		kept := make([]store.PendingItem, 0, len(pending))
		for i, it := range pending {
			if !s.removed[i] {
				kept = append(kept, it)
			}
		}
		// Records first: a crash in between leaves accepted units pending,
		// which costs a second translation, instead of losing them.
		if err := env.Store.SaveRecords(records); err != nil {
			return err
		}
		return env.Store.SavePending(kept)
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// checkKeys returns an ErrBadKey error for the first key of the batch or of
// its out file that is not a unit key.
func checkKeys(b *Batch, out []OutUnit) error {
	for _, u := range b.Units {
		if _, _, err := units.SplitKey(u.Key); err != nil {
			return fmt.Errorf("%w in batch %s: %w", ErrBadKey, b.ID, err)
		}
	}
	for _, o := range out {
		if _, _, err := units.SplitKey(o.Key); err != nil {
			return fmt.Errorf("%w in the out file of batch %s: %w", ErrBadKey, b.ID, err)
		}
	}
	return nil
}

// submission is the state of one Submit call.
type submission struct {
	env         Env
	batch       *Batch
	model       string
	maxAttempts int

	records map[string]store.Record
	pending []store.PendingItem
	res     *SubmitResult

	index   map[string]int    // key -> index in pending
	batchKR map[string]string // keys of the batch's units -> the KR text they were cut with
	seen    map[string]bool   // keys of the batch's units already answered
	removed map[int]bool      // pending indexes to drop
	ld      *loader
	notes   []store.Note // the notes of the batch's units, in out order

	accepted []acceptedUnit // the units accepted, in out order
	terms    []termCandidate
}

// termCandidate is a term of an accepted unit that passed the checks.
type termCandidate struct {
	key string
	NewTerm
}

// collectTerms checks the terms a unit registers against its KR text and its
// normalized translation. Those that fit wait in s.terms; the others are
// listed in the result.
func (s *submission) collectTerms(key, kr, zh string, terms []NewTerm) {
	for _, t := range terms {
		reason := ""
		switch {
		case strings.TrimSpace(t.KR) == "":
			reason = "kr is empty"
		case utf8.RuneCountInString(t.KR) < 2:
			reason = "kr is shorter than two characters"
		case !strings.Contains(kr, t.KR):
			reason = "kr is not in the KR text of the unit"
		case t.ZH == "" || !strings.Contains(zh, t.ZH):
			reason = "zh is not in the translation"
		}
		if reason != "" {
			s.res.TermsSkipped = append(s.res.TermsSkipped, SkippedTerm{Key: key, KR: t.KR, ZH: t.ZH, Reason: reason})
			continue
		}
		s.terms = append(s.terms, termCandidate{key: key, NewTerm: t})
	}
}

// keepTerms stores the terms collected as provisional terms. The first
// rendering of a KR word stands; another rendering of it is stored flagged as
// a conflict; one already stored is not stored again.
func (s *submission) keepTerms() (int, error) {
	if len(s.terms) == 0 {
		return 0, nil
	}
	stored, err := s.env.Store.LoadTerms()
	if err != nil {
		return 0, err
	}
	renderings := map[string][]string{}
	for _, t := range stored {
		renderings[t.KR] = append(renderings[t.KR], t.ZH)
	}
	now := s.env.now().UTC().Format(time.RFC3339)
	var add []store.ProvisionalTerm
	for _, c := range s.terms {
		have := renderings[c.KR]
		if slices.Contains(have, c.ZH) {
			continue
		}
		renderings[c.KR] = append(have, c.ZH)
		add = append(add, store.ProvisionalTerm{
			KR: c.KR, ZH: c.ZH, Key: c.key, Batch: s.batch.ID, RawTag: s.env.RawTag, Time: now, Conflict: len(have) > 0,
		})
	}
	return s.env.Store.AppendTerms(add)
}

func (s *submission) reject(key string, issues ...validate.Issue) {
	s.res.Rejected = append(s.res.Rejected, Rejection{Key: key, Issues: issues})
}

// owns reports whether pending item i is leased to the batch, whether or not
// the lease has run out. A unit without a lease was never handed to the batch,
// or has been released; the batch file cannot make it the batch's.
func (s *submission) owns(i int) bool {
	l := s.pending[i].Lease
	return l != nil && l.Batch == s.batch.ID
}

// release ends the lease of pending item i if it is the batch's.
func (s *submission) release(i int) bool {
	if l := s.pending[i].Lease; l != nil && l.Batch == s.batch.ID {
		s.pending[i].Lease = nil
		return true
	}
	return false
}

// unit handles one translation.
func (s *submission) unit(o OutUnit) error {
	batchKR, inBatch := s.batchKR[o.Key]
	if !inBatch {
		s.reject(o.Key, protocolIssue(CodeNotInBatch, "%s is not a unit of batch %s", o.Key, s.batch.ID))
		return nil
	}
	if o.Note != "" {
		s.notes = append(s.notes, store.Note{
			Key: o.Key, Batch: s.batch.ID, KRSha: units.KRSha(batchKR), Model: s.model, Note: o.Note,
		})
	}
	if s.seen[o.Key] {
		s.reject(o.Key, protocolIssue(CodeDuplicateKey, "%s was submitted more than once; only the first is used", o.Key))
		return nil
	}
	s.seen[o.Key] = true

	i, pending := s.index[o.Key]
	if !pending {
		// Sending the whole out file again must not fail on what was
		// accepted the first time.
		if rec, ok := s.records[o.Key]; ok && rec.Origin == store.OriginClaude && rec.Batch == s.batch.ID {
			s.res.Accepted = append(s.res.Accepted, o.Key)
			return nil
		}
		s.reject(o.Key, protocolIssue(CodeLeaseLost, "%s is no longer pending; it was translated or requeued since batch %s was cut", o.Key, s.batch.ID))
		return nil
	}
	item := &s.pending[i]
	// Before the lease: a unit that waits for a person has none.
	if hasFlag(item.Flags, store.FlagNeedsReview) {
		s.reject(o.Key, protocolIssue(CodeNeedsReview, "%s ran out of attempts and waits for a person; it takes no more translations", o.Key))
		return nil
	}
	if !s.owns(i) {
		if item.Lease == nil {
			s.reject(o.Key, protocolIssue(CodeLeaseLost, "%s is not leased to batch %s: its lease was released, or the batch never held it", o.Key, s.batch.ID))
		} else {
			s.reject(o.Key, protocolIssue(CodeLeaseLost, "%s is leased to another batch now, not to %s", o.Key, s.batch.ID))
		}
		return nil
	}

	file, pointer, err := units.SplitKey(o.Key)
	if err != nil {
		return err // the pending queue only holds valid keys
	}
	fd, err := s.ld.file(file)
	if err != nil {
		return err
	}
	pos, inKR := 0, false
	if fd != nil {
		pos, inKR = fd.index[pointer]
	}
	if !inKR {
		s.release(i)
		s.reject(o.Key, protocolIssue(CodeKRMissing, "%s is no longer in the KR text", o.Key))
		return nil
	}
	kr := fd.units[pos].KR
	if kr != batchKR {
		s.release(i)
		s.reject(o.Key, protocolIssue(CodeKRChanged, "the KR text of %s changed since batch %s was cut; the unit will be handed out again with the new text", o.Key, s.batch.ID))
		return nil
	}

	zh := validate.NormalizeKeywordSpace(validate.Normalize(kr, o.ZH), s.env.KeywordNames)
	prov := s.provisional()
	issues := CheckNewTerms(s.env.Rules, prov, file, kr, zh)
	// A missing glossary rendering is only a warning in CheckNew; a submission
	// without it is rejected unless the note names the term, the way the
	// entry itself matches a KR text. A family with glossary: warn keeps it a
	// warning whatever the note says.
	entries := append(s.env.Rules.GlossaryHits(file, kr), prov...)
	warnOnly := s.env.Rules.GlossaryWarnOnly(file)
	for j := range issues {
		if issues[j].Code == validate.CodeTermMissing && !warnOnly && !noteNames(entries, issues[j], o.Note) {
			issues[j].Severity = validate.SeverityHard
		}
	}
	existing, hasRecord := s.records[o.Key]
	since := s.since(o.Key)

	if validate.HasHard(issues) {
		s.reject(o.Key, issues...)
		item.Attempts++
		item.LastError = hardIssues(issues)
		if item.Attempts < s.maxAttempts {
			return nil // the lease stays, for the fix
		}
		item.Lease = nil
		item.Flags = withFlag(item.Flags, store.FlagNeedsReview)
		s.res.NeedsReview = append(s.res.NeedsReview, o.Key)
		if hasRecord && keepsRecord(existing, kr) {
			// The record stays, but the unit is stuck: flag it, so that the
			// build reports it for a person to look at.
			existing.Flags = withFlag(existing.Flags, store.FlagNeedsReview)
			s.records[o.Key] = existing
			return nil
		}
		text, err := fallbackText(fd, pointer, kr)
		if err != nil {
			return err
		}
		s.records[o.Key] = store.Record{
			Key:    o.Key,
			KRSha:  units.KRSha(kr),
			ZH:     text,
			Origin: store.OriginFallback,
			Since:  since,
			Rules:  s.env.RulesCommit,
			Flags:  withFlag(existing.Flags, store.FlagNeedsReview),
		}
		return nil
	}

	s.records[o.Key] = s.claudeRecord(o.Key, kr, zh)
	s.removed[i] = true
	s.res.Accepted = append(s.res.Accepted, o.Key)
	s.collectTerms(o.Key, kr, zh, o.Terms)
	s.accepted = append(s.accepted, acceptedUnit{file: file, group: TextGroupOf(s.env.Rules, fd.doc, fd.units[pos]), zh: zh})
	if warnings := append(issues, foreignNameIssues(zh, s.env.ForeignNames, s.env.Rules.ForeignIgnore)...); len(warnings) > 0 {
		s.res.Warnings = append(s.res.Warnings, Rejection{Key: o.Key, Issues: warnings})
	}
	return nil
}

// noteNames reports whether note writes the KR term of the entry behind the
// term-missing issue, as the entry matches a KR text (an entry that matches as a word
// needs the term as a word).
func noteNames(entries []rules.Term, is validate.Issue, note string) bool {
	for _, t := range entries {
		if t.KR == is.Term && t.ZH == is.TermZH && t.OccursIn(note) {
			return true
		}
	}
	// No entry behind the issue is a broken invariant: the issue stays hard.
	return false
}

// provisional returns the provisional terms Submit holds the translations to:
// those the batch carried in its glossary (provisional: true) that are still
// in force. A term registered after the batch was cut is not asked of it, and
// one declined or taken into the glossary since is no longer asked at all.
// The batch glossary only decides which KRs are checked; the rendering is the
// one in env.Provisional now, the one the next batch would carry.
//
// A term declined between cut and submit no longer covers a glossary term
// either (TermsIn), so a glossary term it covered in a unit becomes required
// there at submission. This is the same edge as an edit of the glossary
// between cut and submit: the rules in force at submission decide.
func (s *submission) provisional() []rules.Term {
	var out []rules.Term
	for _, g := range s.batch.Glossary {
		if !g.Provisional {
			continue
		}
		for _, t := range s.env.Provisional {
			if t.KR == g.KR {
				out = append(out, t)
				break
			}
		}
	}
	return out
}

// hardIssues is the text PendingItem.LastError holds for issues: the code and
// the message of each hard issue, on one line.
func hardIssues(issues []validate.Issue) string {
	var parts []string
	for _, is := range issues {
		if is.Severity == validate.SeverityHard {
			parts = append(parts, is.Code+": "+strings.Join(strings.Fields(is.Message), " "))
		}
	}
	return strings.Join(parts, "; ")
}

// since is the raw tag a record written for key now starts at: that of the
// record it replaces, else that of the snapshot.
func (s *submission) since(key string) string {
	if rec, ok := s.records[key]; ok && rec.Since != "" {
		return rec.Since
	}
	return s.env.RawTag
}

// claudeRecord is the record of a translation accepted for key, a unit whose
// KR text is kr: the record of a new translation under the rules of now,
// made by the submitter's model in this batch.
func (s *submission) claudeRecord(key, kr, zh string) store.Record {
	return store.Record{
		Key:    key,
		KRSha:  units.KRSha(kr),
		ZH:     zh,
		Origin: store.OriginClaude,
		Since:  s.since(key),
		Rules:  s.env.RulesCommit,
		Model:  s.model,
		Batch:  s.batch.ID,
		Flags:  withoutFlag(s.records[key].Flags, store.FlagNeedsReview),
	}
}

// keepsRecord reports whether a unit that ran out of attempts keeps the record
// it had: the record was made from the KR text there is now and is a
// translation of it, not itself a stand-in. A requeued unit that cannot be
// translated again then keeps the translation a person wanted improved, which
// is better than the EN text. Any other record (made from another KR text, or
// a fallback) is replaced by a fresh stand-in.
func keepsRecord(rec store.Record, kr string) bool {
	return rec.KRSha == units.KRSha(kr) && rec.Origin != store.OriginFallback
}

// fallbackText is the text that stands in for a translation that failed: the
// EN text aligned to the unit, or KR when there is none.
func fallbackText(fd *fileData, pointer, kr string) (string, error) {
	en, err := fd.aligned(langEN, pointer)
	if err != nil {
		return "", err
	}
	if en != nil && strings.TrimSpace(en.Value) != "" {
		return en.Value, nil
	}
	return kr, nil
}

// releaseOmitted ends the lease of the batch's units that were not answered
// and reports them in Released. A lease that has passed to another batch is
// not ours to end.
func (s *submission) releaseOmitted() {
	for _, u := range s.batch.Units {
		if s.seen[u.Key] {
			continue
		}
		if i, ok := s.index[u.Key]; ok && s.release(i) {
			s.res.Released = append(s.res.Released, u.Key)
		}
	}
}
