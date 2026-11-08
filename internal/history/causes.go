package history

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
