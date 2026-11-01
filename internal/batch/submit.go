package batch

import (
	"errors"
	"fmt"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// Codes of the issues Submit adds to those of validate.Check.
const (
	// CodeForeignName is a warning: the translation uses a name that only
	// the LTM translation uses.
	CodeForeignName = "foreign-name"
	// CodeLeaseLost: the unit is no longer pending, or its lease now belongs
	// to another batch.
	CodeLeaseLost = "lease-lost"
	// CodeNotInBatch: the key is not one of the batch's units.
	CodeNotInBatch = "not-in-batch"
	// CodeDuplicateKey: the key was already answered in this submission.
	CodeDuplicateKey = "duplicate-key"
	// CodeKRMissing: the unit has left KR since the batch was cut.
	CodeKRMissing = "kr-missing"
)

// SubmitResult says what Submit did with the units it was given. The lists
// are never nil.
type SubmitResult struct {
	// Accepted are the keys whose translation was recorded.
	Accepted []string `json:"accepted"`
	// Rejected are the units that were not recorded, with the reasons.
	Rejected []Rejection `json:"rejected"`
	// Warnings are issues of accepted units.
	Warnings []Rejection `json:"warnings"`
	// NeedsReview are the rejected units that ran out of attempts and now
	// stand in with their fallback text.
	NeedsReview []string `json:"needs_review"`
}

// Rejection is a unit with the issues found in it.
type Rejection struct {
	Key    string           `json:"key"`
	Issues []validate.Issue `json:"issues"`
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

// foreignNameIssues warns about each of names that zh contains.
func foreignNameIssues(zh string, names []string) []validate.Issue {
	var issues []validate.Issue
	for _, name := range names {
		if name != "" && strings.Contains(zh, name) {
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
// is processed while it is still pending and its lease is free or b's.
// Its text is normalized and checked as a new translation. Without a hard
// issue it becomes a record (origin claude) and leaves the queue. With one,
// its attempts grow and its lease ends; at maxAttempts the unit gets a
// fallback record (the EN text, else KR) flagged needs_review and stays in
// the queue flagged. Units of b that are not in out only lose b's lease.
//
// Everything happens in one store lock. maxAttempts 0 means DefaultMaxAttempts.
func Submit(env Env, b *Batch, out []OutUnit, model string, maxAttempts int) (*SubmitResult, error) {
	if err := env.check(); err != nil {
		return nil, err
	}
	if b == nil {
		return nil, errors.New("batch: no batch to submit")
	}
	if maxAttempts < 0 {
		return nil, errors.New("batch: maxAttempts must not be negative")
	}
	if maxAttempts == 0 {
		maxAttempts = DefaultMaxAttempts
	}

	res := &SubmitResult{
		Accepted: []string{}, Rejected: []Rejection{}, Warnings: []Rejection{}, NeedsReview: []string{},
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
			inBatch: make(map[string]bool, len(b.Units)),
			seen:    map[string]bool{},
			removed: map[int]bool{},
			ld:      newLoader(env),
		}
		for i, it := range pending {
			s.index[it.Key] = i
		}
		for _, u := range b.Units {
			s.inBatch[u.Key] = true
		}
		for _, o := range out {
			if err := s.unit(o); err != nil {
				return err
			}
		}
		s.releaseOmitted()

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

// submission is the state of one Submit call.
type submission struct {
	env         Env
	batch       *Batch
	model       string
	maxAttempts int

	records map[string]store.Record
	pending []store.PendingItem
	res     *SubmitResult

	index   map[string]int  // key -> index in pending
	inBatch map[string]bool // keys of the batch's units
	seen    map[string]bool // keys of the batch's units already answered
	removed map[int]bool    // pending indexes to drop
	ld      *loader
}

func (s *submission) reject(key string, issues ...validate.Issue) {
	s.res.Rejected = append(s.res.Rejected, Rejection{Key: key, Issues: issues})
}

// owns reports whether the lease of pending item i is free or the batch's.
func (s *submission) owns(i int) bool {
	l := s.pending[i].Lease
	return l == nil || l.Batch == s.batch.ID
}

// unit handles one translation.
func (s *submission) unit(o OutUnit) error {
	if !s.inBatch[o.Key] {
		s.reject(o.Key, protocolIssue(CodeNotInBatch, "%s is not a unit of batch %s", o.Key, s.batch.ID))
		return nil
	}
	if s.seen[o.Key] {
		s.reject(o.Key, protocolIssue(CodeDuplicateKey, "%s was submitted more than once; only the first is used", o.Key))
		return nil
	}
	s.seen[o.Key] = true

	i, pending := s.index[o.Key]
	if !pending || !s.owns(i) {
		s.reject(o.Key, protocolIssue(CodeLeaseLost, "%s is no longer leased to batch %s; it was translated, requeued or handed to another batch", o.Key, s.batch.ID))
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
		s.reject(o.Key, protocolIssue(CodeKRMissing, "%s is no longer in the KR text", o.Key))
		return nil
	}
	kr := fd.units[pos].KR

	zh := validate.Normalize(kr, o.ZH)
	issues := validate.Check(kr, zh, validate.Context{Terms: validateTerms(s.env.Rules, kr)})
	existing, hasRecord := s.records[o.Key]
	since := s.env.RawTag
	if hasRecord && existing.Since != "" {
		since = existing.Since
	}

	item := &s.pending[i]
	if validate.HasHard(issues) {
		s.reject(o.Key, issues...)
		item.Attempts++
		item.Lease = nil
		if item.Attempts < s.maxAttempts {
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
			Flags:  withFlag(existing.Flags, store.FlagNeedsReview),
		}
		item.Flags = withFlag(item.Flags, store.FlagNeedsReview)
		s.res.NeedsReview = append(s.res.NeedsReview, o.Key)
		return nil
	}

	s.records[o.Key] = store.Record{
		Key:    o.Key,
		KRSha:  units.KRSha(kr),
		ZH:     zh,
		Origin: store.OriginClaude,
		Since:  since,
		Model:  s.model,
		Batch:  s.batch.ID,
		Flags:  withoutFlag(existing.Flags, store.FlagNeedsReview),
	}
	s.removed[i] = true
	s.res.Accepted = append(s.res.Accepted, o.Key)
	if warnings := append(issues, foreignNameIssues(zh, s.env.ForeignNames)...); len(warnings) > 0 {
		s.res.Warnings = append(s.res.Warnings, Rejection{Key: o.Key, Issues: warnings})
	}
	return nil
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

// releaseOmitted ends the lease of the batch's units that were not answered.
// A lease that has passed to another batch is not ours to end.
func (s *submission) releaseOmitted() {
	for _, u := range s.batch.Units {
		if s.seen[u.Key] {
			continue
		}
		if i, ok := s.index[u.Key]; ok && s.pending[i].Lease != nil && s.pending[i].Lease.Batch == s.batch.ID {
			s.pending[i].Lease = nil
		}
	}
}
