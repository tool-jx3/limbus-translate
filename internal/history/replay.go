package history

import (
	"slices"

	"github.com/tool-jx3/limbus-translate/internal/rewrite"
	"github.com/tool-jx3/limbus-translate/internal/rulefile"
)

// replay is a rewrite file applied since prev, to replay on the old record of
// a unit: what rewrite.Apply does to one record (rewrite.Compiled), and the
// rules commit it was read from.
type replay struct {
	c      *rewrite.Compiled
	commit string
}

// rewriteCauses is step 5: the rewrite files applied since prev that,
// replayed in order on the old record (its text and origin, with the KR text
// of cur), change its text, provided the result is the new text exactly;
// otherwise something else wrote over them, and they are no cause. The checks
// of rewrite.Apply's ok, made by the plan against the rules of the time, are
// not repeated: the new text shows whether they let the rewrite through.
//
// When some of those files cannot be read, the note says so for an entry that
// looks like a rewrite: the new record kept the batch and origin of the old
// one but not its text, and that origin is one a rewrite file changes when it
// names none (rewrite.DefaultOrigins; what the unread files name is not
// known). A person's adjustment edited in its file looks so too, but its
// origin is human.
func (in *inference) rewriteCauses(e Entry, u unitState) ([]Cause, string) {
	if len(in.rewrites) == 0 && in.rewriteGone == "" || !u.hasPrev || e.NewKR == nil || e.NewZH == nil {
		return nil, ""
	}
	zh := u.prev.ZH
	var applied []replay
	for _, r := range in.rewrites {
		if out, ok := r.c.Rewrite(e.File, u.prev.Origin, zh); ok && r.c.KROK(*e.NewKR) {
			zh = out
			applied = append(applied, r)
		}
	}
	if len(applied) > 0 && zh == *e.NewZH {
		causes := make([]Cause, len(applied))
		for i, r := range applied {
			f := r.c.File
			causes[i] = Cause{Kind: CauseKindRewrite, ID: f.ID}
			in.use(CauseKindRewrite, f.ID, rewrite.Dir+"/"+f.ID+rulefile.Ext, f.Reason, r.commit)
		}
		return causes, ""
	}
	looksRewritten := e.Kind == KindZHChanged && u.hasCur && u.cur.Batch == u.prev.Batch &&
		u.cur.Origin == u.prev.Origin && u.cur.ZH != u.prev.ZH &&
		slices.Contains(rewrite.DefaultOrigins, u.prev.Origin)
	if looksRewritten {
		return nil, in.rewriteGone
	}
	return nil, ""
}
