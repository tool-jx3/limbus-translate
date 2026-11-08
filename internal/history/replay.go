package history

import (
	"regexp"
	"slices"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/rewrite"
	"github.com/tool-jx3/limbus-translate/internal/rulefile"
	"github.com/tool-jx3/limbus-translate/internal/store"
)

// rewriteCauses is step 5: the rewrite files applied since prev that, replayed
// in order on the old record (its text and origin, with the KR text of cur),
// change its text, provided the result is the new text exactly; otherwise
// something else wrote over them, and they are no cause. When some of those files cannot be read and the entry looks like a
// rewrite (the new record kept the batch and origin of the old one but not its
// text), the note says so.
func (in *inference) rewriteCauses(e Entry, u unitState) ([]Cause, string) {
	if len(in.rewrites)+len(in.rewriteGone) == 0 || !u.hasPrev || e.NewKR == nil || e.NewZH == nil {
		return nil, ""
	}
	zh := u.prev.ZH
	var applied []replay
	for _, r := range in.rewrites {
		if out, ok := r.rewrite(e.File, u.prev.Origin, zh, *e.NewKR); ok {
			zh = out
			applied = append(applied, r)
		}
	}
	if len(applied) > 0 && zh == *e.NewZH {
		causes := make([]Cause, len(applied))
		for i, r := range applied {
			causes[i] = Cause{Kind: CauseKindRewrite, ID: r.f.ID}
			in.use(CauseKindRewrite, r.f.ID, rewrite.Dir+"/"+r.f.ID+rulefile.Ext, r.f.Reason, r.commit)
		}
		return causes, ""
	}
	looksRewritten := e.Kind == KindZHChanged && u.hasCur && u.cur.Batch == u.prev.Batch &&
		u.cur.Origin == u.prev.Origin && u.cur.ZH != u.prev.ZH
	if len(in.rewriteGone) > 0 && looksRewritten {
		return nil, in.gone("rewrite " + strings.Join(in.rewriteGone, ", "))
	}
	return nil, ""
}

// replay is a rewrite file to replay on one record at a time.
type replay struct {
	f        rewrite.File
	commit   string // of the rules it was read from
	re, krRe *regexp.Regexp
	origins  []string
}

// rewriteOrigins are the origins a rewrite file that names none changes, as
// rewrite.Apply has them.
var rewriteOrigins = []string{store.OriginParatranz, store.OriginClaude}

// newReplay compiles f, which must be valid, as a File from RulesAt is.
func newReplay(f rewrite.File, commit string) replay {
	r := replay{f: f, commit: commit, re: regexp.MustCompile(f.Pattern), origins: f.Origins}
	if f.KRPattern != "" {
		r.krRe = regexp.MustCompile(f.KRPattern)
	}
	if len(r.origins) == 0 {
		r.origins = rewriteOrigins
	}
	return r
}

// rewrite returns what the rewrite file makes of zh, the text of a record of
// origin, of a unit of file whose KR text is kr, and whether it changes it:
// the conditions of rewrite.Apply, without its ok. The checks ok made when the
// plan applied the file, against the rules of the time, are not repeated; the
// new text shows whether they let the rewrite through.
func (r replay) rewrite(file, origin, zh, kr string) (string, bool) {
	if !slices.Contains(r.origins, origin) || !config.MatchAny(r.f.Files, file) || !r.re.MatchString(zh) {
		return zh, false
	}
	out := r.re.ReplaceAllString(zh, r.f.Replace)
	if out == zh || r.krRe != nil && !r.krRe.MatchString(kr) {
		return zh, false
	}
	return out, true
}
