package history

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/adjust"
	"github.com/tool-jx3/limbus-translate/internal/requeue"
	"github.com/tool-jx3/limbus-translate/internal/rewrite"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

const (
	commitPrev = "1111111111111111111111111111111111111111"
	commitCur  = "2222222222222222222222222222222222222222"
)

// ptrAt is the pointer of the desc of item i of F.json, keyAt its key.
func ptrAt(i int) string { return fmt.Sprintf("/dataList/%d/desc", i) }
func keyAt(i int) string { return units.Key("F.json", ptrAt(i)) }

// item is an item of F.json with id and desc.
func item(id int, desc string) string { return fmt.Sprintf(`{"id":%d,"desc":%q}`, id, desc) }

// recAt is the record of item i of F.json, made from the KR text kr.
func recAt(i int, kr, zh, origin, batch string) store.Record {
	return store.Record{Key: keyAt(i), KRSha: units.KRSha(kr), ZH: zh, Origin: origin, Batch: batch}
}

// sideSpec is one Release of F.json for the tests of causes.
type sideSpec struct {
	kr, zh  string
	recs    []store.Record
	applied store.Applied
	pending []store.PendingItem
	rules   string
}

func (s sideSpec) side(t *testing.T) *Side {
	t.Helper()
	return &Side{
		RulesCommit: s.rules, State: state(t, s.kr, s.zh, s.recs...), PackDir: t.TempDir(),
		Applied: s.applied, Pending: s.pending,
	}
}

// causesOf compares prev with cur and infers the causes of each entry.
func causesOf(t *testing.T, prev, cur *Side, prevRules, curRules *RuleFiles) []Entry {
	t.Helper()
	entries := compare(t, prev.State, cur.State)
	for i := range entries {
		entries[i].Causes = inferCauses(prev, cur, prevRules, curRules, entries[i])
	}
	return entries
}

func wantCauses(t *testing.T, e Entry, want ...Cause) {
	t.Helper()
	if !reflect.DeepEqual(e.Causes, want) {
		t.Errorf("%s %s: causes = %+v, want %+v", e.Kind, e.Pointer, e.Causes, want)
	}
}

// rulesFrom is a RulesSource of the rules of the commits in m; any other
// commit is an error.
func rulesFrom(m map[string]*RuleFiles) RulesSource {
	return func(commit string) (*RuleFiles, error) {
		if r, ok := m[commit]; ok {
			return r, nil
		}
		return nil, fmt.Errorf("history: read the rules of %s: no such commit", commit)
	}
}

func rw(id, pattern, replace string) rewrite.File {
	return rewrite.File{ID: id, Reason: "改寫 " + id, Files: []string{"F.json"}, Pattern: pattern, Replace: replace}
}

func rq(id string, sel requeue.Selector) requeue.File {
	return requeue.File{ID: id, Reason: "重翻 " + id, Selector: sel}
}

func makeWith(t *testing.T, prev, cur *Side, rules RulesSource) *Record {
	t.Helper()
	rec, err := Make(prev, cur, "zh-TW-20261108.1", time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC), testLang, rules)
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

func TestCausesRewriteChain(t *testing.T) {
	kr := list(item(1, "a"))
	prev := sideSpec{
		kr: kr, zh: list(item(1, "甲 乙")),
		recs:    []store.Record{recAt(0, "a", "甲 乙", store.OriginParatranz, "")},
		applied: store.Applied{Rewrite: []string{"r0"}},
	}.side(t)
	// Applied in another order than that of the file names, which is the
	// order the plan applies them in.
	cur := sideSpec{
		kr: kr, zh: list(item(1, "甲~乙")),
		recs:    []store.Record{recAt(0, "a", "甲~乙", store.OriginParatranz, "")},
		applied: store.Applied{Rewrite: []string{"r0", "r3", "r2", "r1", "r4"}},
	}.side(t)
	r2 := rw("r2", "-", "~")
	r2.KRPattern = "^a$"
	r4 := rw("r4", "甲", "丁")
	r4.Origins = []string{store.OriginClaude} // not the origin of the record
	rules := &RuleFiles{Commit: commitCur, Rewrites: []rewrite.File{
		rw("r0", "甲", "丙"), // applied before prev: not replayed
		rw("r1", " ", "-"),
		r2,
		rw("r3", "zzz", "y"), // changes nothing
		r4,
	}}
	got := causesOf(t, prev, cur, rules, rules)
	if len(got) != 1 {
		t.Fatalf("entries = %+v", got)
	}
	wantCauses(t, got[0], Cause{Kind: CauseKindRewrite, ID: "r1"}, Cause{Kind: CauseKindRewrite, ID: "r2"})
}

func TestCausesRewriteOverwritten(t *testing.T) {
	kr := list(item(1, "a"))
	prev := sideSpec{kr: kr, zh: list(item(1, "甲 乙")), recs: []store.Record{recAt(0, "a", "甲 乙", store.OriginParatranz, "")}}.side(t)
	// r1 was applied, but the unit was translated again afterwards.
	cur := sideSpec{
		kr: kr, zh: list(item(1, "全新譯文")),
		recs:    []store.Record{recAt(0, "a", "全新譯文", store.OriginClaude, "b1")},
		applied: store.Applied{Rewrite: []string{"r1"}},
	}.side(t)
	rules := &RuleFiles{Commit: commitCur, Rewrites: []rewrite.File{rw("r1", " ", "-")}}
	got := causesOf(t, prev, cur, rules, rules)
	if len(got) != 1 {
		t.Fatalf("entries = %+v", got)
	}
	wantCauses(t, got[0], Cause{Kind: CauseKindRetranslated, Batch: "b1"})
}

func adjustment(file, key, kr, zh, reason string) adjust.Adjustment {
	return adjust.Adjustment{Key: key, Raw: "raw-1", KRSha: units.KRSha(kr), ZH: zh, Reason: reason, Rule: adjust.OneOff, File: file}
}

func TestCausesAdjustmentByKRSha(t *testing.T) {
	kr := list(item(1, "a"), item(2, "b"))
	prev := sideSpec{
		kr: kr, zh: list(item(1, "甲"), item(2, "乙")),
		recs: []store.Record{recAt(0, "a", "甲", store.OriginClaude, "b0"), recAt(1, "b", "乙", store.OriginClaude, "b0")},
	}.side(t)
	cur := sideSpec{
		kr: kr, zh: list(item(1, "人工甲"), item(2, "人工乙")),
		recs: []store.Record{
			recAt(0, "a", "人工甲", store.OriginHuman, "adjustment:2026-11.yaml"),
			recAt(1, "b", "人工乙", store.OriginHuman, "adjustment:2026-10.yaml"),
		},
	}.side(t)
	curRules := &RuleFiles{Commit: commitCur, Adjustments: []adjust.Adjustment{
		// At the key of the unit, but of another KR text: not the one.
		adjustment("2026-11.yaml", keyAt(0), "x", "人工甲", "鍵相同"),
		// The key moved by lct adjust tidy; the KR text and zh still tell.
		adjustment("2026-11.yaml", keyAt(7), "a", "人工甲", "語氣"),
		adjustment("2026-11.yaml", keyAt(8), "a", "別的譯文", "別的"),
	}}
	// 2026-10.yaml is only in the rules of prev.
	prevRules := &RuleFiles{Commit: commitPrev, Adjustments: []adjust.Adjustment{adjustment("2026-10.yaml", keyAt(1), "b", "人工乙", "舊檔")}}
	got := causesOf(t, prev, cur, prevRules, curRules)
	if len(got) != 2 {
		t.Fatalf("entries = %+v", got)
	}
	wantCauses(t, got[0], Cause{Kind: CauseKindAdjustment, ID: "2026-11.yaml", Reason: "語氣"})
	wantCauses(t, got[1], Cause{Kind: CauseKindAdjustment, ID: "2026-10.yaml", Reason: "舊檔"})
}

func TestCausesAdjustmentExpired(t *testing.T) {
	prev := sideSpec{
		kr: list(item(1, "a"), item(2, "b")), zh: list(item(1, "人工"), item(2, "人工乙")),
		recs: []store.Record{
			recAt(0, "a", "人工", store.OriginHuman, "adjustment:x.yaml"),
			recAt(1, "b", "人工乙", store.OriginHuman, "adjustment:x.yaml"),
		},
	}.side(t)
	// The KR texts changed. The first unit was translated again, the second
	// stands in for now.
	cur := sideSpec{
		kr: list(item(1, "A"), item(2, "B")), zh: list(item(1, "新譯"), item(2, "B")),
		recs: []store.Record{recAt(0, "A", "新譯", store.OriginClaude, "b1")},
	}.side(t)
	// lct adjust tidy deleted the adjustments that expired.
	prevRules := &RuleFiles{Commit: commitPrev, Adjustments: []adjust.Adjustment{
		adjustment("x.yaml", keyAt(0), "a", "人工", "舊理由"),
		adjustment("x.yaml", keyAt(1), "b", "人工乙", "舊理由二"),
	}}
	curRules := &RuleFiles{Commit: commitCur}
	got := causesOf(t, prev, cur, prevRules, curRules)
	if len(got) != 2 {
		t.Fatalf("entries = %+v", got)
	}
	wantCauses(t, got[0],
		Cause{Kind: CauseKindKRChanged, Batch: "b1"},
		Cause{Kind: CauseKindAdjustmentExpired, ID: "x.yaml", Batch: "b1", Reason: "舊理由"})
	wantCauses(t, got[1],
		Cause{Kind: CauseKindKRChanged},
		Cause{Kind: CauseKindAdjustmentExpired, ID: "x.yaml", Reason: "舊理由二"})
}

func TestCausesRequeueAcrossReleases(t *testing.T) {
	kr := list(item(1, "a"), item(2, "b"))
	// The request q1 was applied before prev; the units waited in its queue.
	applied := store.Applied{Requeue: []string{"q0", "q1", "q2"}}
	prev := sideSpec{
		kr: kr, zh: list(item(1, "甲"), item(2, "乙")),
		recs:    []store.Record{recAt(0, "a", "甲", store.OriginClaude, "b0"), recAt(1, "b", "乙", store.OriginClaude, "b0")},
		applied: applied,
		pending: []store.PendingItem{{Key: keyAt(0), Reason: requeue.Reason}, {Key: keyAt(1), Reason: requeue.Reason}},
	}.side(t)
	cur := sideSpec{
		kr: kr, zh: list(item(1, "甲改"), item(2, "乙改")),
		recs:    []store.Record{recAt(0, "a", "甲改", store.OriginClaude, "b1"), recAt(1, "b", "乙改", store.OriginClaude, "b1")},
		applied: applied,
	}.side(t)
	rules := &RuleFiles{Commit: commitCur, Requeues: []requeue.File{
		rq("q0", requeue.Selector{Files: []string{"F.json"}}),
		rq("q1", requeue.Selector{Keys: []string{keyAt(0)}}),
		rq("q2", requeue.Selector{Files: []string{"F.json"}, Origins: []string{store.OriginParatranz}}),
		rq("q9", requeue.Selector{Files: []string{"F.json"}}), // not applied
	}}
	got := causesOf(t, prev, cur, rules, rules)
	if len(got) != 2 {
		t.Fatalf("entries = %+v", got)
	}
	// The last request in file name order that selects the unit.
	wantCauses(t, got[0], Cause{Kind: CauseKindRequeue, ID: "q1", Batch: "b1"})
	wantCauses(t, got[1], Cause{Kind: CauseKindRequeue, ID: "q0", Batch: "b1"})
}

func TestCausesRequeueSameRelease(t *testing.T) {
	kr := list(item(1, "a"), item(2, "b"))
	prev := sideSpec{
		kr: kr, zh: list(item(1, "甲"), item(2, "乙")),
		recs: []store.Record{recAt(0, "a", "甲", store.OriginClaude, "b0"), recAt(1, "b", "乙", store.OriginClaude, "b0")},
	}.side(t)
	// q1 was applied between the tags, and the unit it sent back was
	// translated before cur.
	cur := sideSpec{
		kr: kr, zh: list(item(1, "甲改"), item(2, "乙改")),
		recs:    []store.Record{recAt(0, "a", "甲改", store.OriginClaude, "b1"), recAt(1, "b", "乙改", store.OriginClaude, "b1")},
		applied: store.Applied{Requeue: []string{"q1"}},
	}.side(t)
	rules := &RuleFiles{Commit: commitCur, Requeues: []requeue.File{rq("q1", requeue.Selector{Keys: []string{keyAt(0)}})}}
	got := causesOf(t, prev, cur, rules, rules)
	if len(got) != 2 {
		t.Fatalf("entries = %+v", got)
	}
	wantCauses(t, got[0], Cause{Kind: CauseKindRequeue, ID: "q1", Batch: "b1"})
	wantCauses(t, got[1], Cause{Kind: CauseKindRetranslated, Batch: "b1"})
}

func TestCausesRetranslatedByPendingReason(t *testing.T) {
	kr := list(item(1, "a"))
	prev := sideSpec{
		kr: kr, zh: list(item(1, "甲")),
		recs:    []store.Record{recAt(0, "a", "甲", store.OriginClaude, "b0")},
		pending: []store.PendingItem{{Key: keyAt(0), Reason: "structural-fail"}},
	}.side(t)
	cur := sideSpec{kr: kr, zh: list(item(1, "甲改")), recs: []store.Record{recAt(0, "a", "甲改", store.OriginClaude, "b1")}}.side(t)
	rules := &RuleFiles{Commit: commitCur}
	got := causesOf(t, prev, cur, rules, rules)
	if len(got) != 1 {
		t.Fatalf("entries = %+v", got)
	}
	wantCauses(t, got[0], Cause{Kind: CauseKindRetranslated, Batch: "b1", Reason: "structural-fail"})
}

func TestCausesReuseLateKRNewDeleted(t *testing.T) {
	prev := sideSpec{
		kr: list(item(1, "a"), item(2, "b"), item(3, "c"), item(4, "d"), item(6, "f")),
		zh: list(item(1, "甲"), item(2, "乙"), item(3, "c"), item(4, "丁"), item(6, "f")),
		recs: []store.Record{
			recAt(0, "a", "甲", store.OriginParatranz, ""),
			recAt(1, "b", "乙", store.OriginClaude, "b0"),
			recAt(2, "c", "c", store.OriginFallback, ""),
			recAt(3, "d", "丁", store.OriginClaude, "b0"),
			// id 6 has no record: a stand-in.
		},
		// A unit that was a stand-in is mostly queued as new.
		pending: []store.PendingItem{{Key: keyAt(2), Reason: "new"}, {Key: keyAt(4), Reason: "new"}},
	}.side(t)
	cur := sideSpec{
		kr: list(item(1, "a"), item(2, "B"), item(3, "c"), item(5, "e"), item(6, "f")),
		zh: list(item(1, "甲二"), item(2, "乙乙"), item(3, "丙"), item(5, "戊"), item(6, "己")),
		recs: []store.Record{
			recAt(0, "a", "甲二", store.OriginClaude, "reuse:F.json|/dataList/9/desc"),
			recAt(1, "B", "乙乙", store.OriginClaude, "b1"),
			recAt(2, "c", "丙", store.OriginClaude, "b1"),
			recAt(3, "e", "戊", store.OriginClaude, "b1"),
			recAt(4, "f", "己", store.OriginClaude, "b1"),
		},
	}.side(t)
	rules := &RuleFiles{Commit: commitCur}
	got := causesOf(t, prev, cur, rules, rules)
	byKind := map[string][]Entry{}
	for _, e := range got {
		byKind[e.Kind] = append(byKind[e.Kind], e)
	}
	if len(got) != 6 || len(byKind[KindZHChanged]) != 3 {
		t.Fatalf("entries = %+v", got)
	}
	zh := byKind[KindZHChanged]
	wantCauses(t, zh[0], Cause{Kind: CauseKindReuse, Batch: "reuse:F.json|/dataList/9/desc"})
	wantCauses(t, zh[1], Cause{Kind: CauseKindLate, Batch: "b1"}) // was a fallback
	wantCauses(t, zh[2], Cause{Kind: CauseKindLate, Batch: "b1"}) // had no record
	wantCauses(t, byKind[KindKRChanged][0], Cause{Kind: CauseKindKRChanged, Batch: "b1"})
	wantCauses(t, byKind[KindNew][0], Cause{Kind: CauseKindNew})
	wantCauses(t, byKind[KindDeleted][0], Cause{Kind: CauseKindDeleted})
}

func TestCausesOther(t *testing.T) {
	kr := list(item(1, "a"), item(2, "b"))
	prev := sideSpec{
		kr: kr, zh: list(item(1, "甲"), item(2, "乙")),
		recs: []store.Record{recAt(0, "a", "甲", store.OriginParatranz, ""), recAt(1, "b", "乙", store.OriginClaude, "reuse:F.json|/dataList/9/desc")},
	}.side(t)
	// Nothing explains the changes: no rule, no translation, no reuse made now.
	cur := sideSpec{
		kr: kr, zh: list(item(1, "甲甲"), item(2, "乙乙")),
		recs:    []store.Record{recAt(0, "a", "甲甲", store.OriginParatranz, ""), recAt(1, "b", "乙乙", store.OriginClaude, "reuse:F.json|/dataList/9/desc")},
		applied: store.Applied{Rewrite: []string{"r1"}},
	}.side(t)
	// r1 rewrote the reused text, so the reuse is no cause of it.
	rules := &RuleFiles{Commit: commitCur, Rewrites: []rewrite.File{rw("r1", "乙", "乙乙")}}
	got := causesOf(t, prev, cur, rules, rules)
	if len(got) != 2 {
		t.Fatalf("entries = %+v", got)
	}
	wantCauses(t, got[0], Cause{Kind: CauseKindOther})
	wantCauses(t, got[1], Cause{Kind: CauseKindRewrite, ID: "r1"})
}

func TestCausesRulesUnavailable(t *testing.T) {
	kr := list(item(1, "a"), item(2, "b"), item(3, "c"), item(4, "d"), item(5, "e"))
	prevSpec := sideSpec{
		kr: kr, zh: list(item(1, "甲 甲"), item(2, "乙"), item(3, "丙"), item(4, "丁"), item(5, "戊")),
		recs: []store.Record{
			recAt(0, "a", "甲 甲", store.OriginParatranz, ""),
			recAt(1, "b", "乙", store.OriginClaude, "b0"),
			recAt(2, "c", "丙", store.OriginClaude, "b0"),
			recAt(3, "d", "丁", store.OriginClaude, "b0"),
			recAt(4, "e", "戊", store.OriginClaude, "b0"),
		},
		applied: store.Applied{Requeue: []string{"q1"}},
		pending: []store.PendingItem{{Key: keyAt(1), Reason: requeue.Reason}},
		rules:   commitPrev,
	}
	curSpec := sideSpec{
		kr: list(item(1, "a"), item(2, "b"), item(3, "c"), item(4, "D"), item(5, "e")),
		zh: list(item(1, "甲-甲"), item(2, "乙改"), item(3, "人工"), item(4, "丁改"), item(5, "戊改")),
		recs: []store.Record{
			recAt(0, "a", "甲-甲", store.OriginParatranz, ""),             // rewritten by r1
			recAt(1, "b", "乙改", store.OriginClaude, "b1"),               // requeued by q1
			recAt(2, "c", "人工", store.OriginHuman, "adjustment:a.yaml"), // adjusted
			recAt(3, "D", "丁改", store.OriginClaude, "b1"),               // KR changed: no rule needed
			recAt(4, "e", "戊改", store.OriginClaude, "b1"),               // retranslated: no rule needed
		},
		applied: store.Applied{Requeue: []string{"q1"}, Rewrite: []string{"r1"}},
		rules:   commitCur,
	}
	wantOthers := func(t *testing.T, rec *Record) {
		t.Helper()
		if len(rec.Entries) != 5 || rec.CausesVersion != CausesVersion {
			t.Fatalf("record = %+v", rec)
		}
		wantCauses(t, rec.Entries[3], Cause{Kind: CauseKindKRChanged, Batch: "b1"})
		wantCauses(t, rec.Entries[4], Cause{Kind: CauseKindRetranslated, Batch: "b1"})
		if rec.Unknown != 3 {
			t.Errorf("Unknown = %d, want 3", rec.Unknown)
		}
		if len(rec.Rules) != 0 {
			t.Errorf("Rules = %+v, want none", rec.Rules)
		}
	}
	unknown := func(t *testing.T, e Entry, notes ...string) {
		t.Helper()
		if len(e.Causes) != 1 || e.Causes[0].Kind != CauseKindUnknown {
			t.Fatalf("%s: causes = %+v, want one unknown", e.Pointer, e.Causes)
		}
		for _, n := range notes {
			if !strings.Contains(e.Causes[0].Note, n) {
				t.Errorf("%s: note %q does not say %q", e.Pointer, e.Causes[0].Note, n)
			}
		}
	}

	t.Run("no rules can be read", func(t *testing.T) {
		rec := makeWith(t, prevSpec.side(t), curSpec.side(t), rulesFrom(nil))
		wantOthers(t, rec)
		errText := "history: read the rules of " + commitCur + ": no such commit"
		unknown(t, rec.Entries[0], "r1", errText)
		unknown(t, rec.Entries[1], requeue.Reason, errText)
		unknown(t, rec.Entries[2], "a.yaml", errText)
	})

	t.Run("prev names no rules commit, cur's lack the files", func(t *testing.T) {
		p := prevSpec
		p.rules = ""
		rec := makeWith(t, p.side(t), curSpec.side(t), rulesFrom(map[string]*RuleFiles{commitCur: {Commit: commitCur}}))
		wantOthers(t, rec)
		for i := 0; i < 3; i++ {
			unknown(t, rec.Entries[i], "rules commit")
		}
		unknown(t, rec.Entries[0], "r1")
	})
}

func TestMakeRulesIndex(t *testing.T) {
	f := newRepoFixture(t)
	put(t, f.root, "rules/rewrite/r1.yaml", "reason: 改寫一\nfiles: [F.json]\npattern: ' '\nreplace: '-'\n")
	put(t, f.root, "rules/rewrite/r2.yaml", "reason: 改寫二\nfiles: [F.json]\npattern: 'zzz'\nreplace: 'y'\n")
	put(t, f.root, "rules/requeue/q1.yaml", "reason: 重翻一\nkeys: ['"+keyAt(1)+"']\n")
	put(t, f.root, "rules/requeue/q2.yaml", "reason: 重翻二\nfiles: [Other.json]\n")
	adj := func(key, kr, zh, reason string) string {
		return "- key: " + key + "\n  raw: raw-1\n  kr_sha: " + units.KRSha(kr) + "\n  zh: " + zh + "\n  reason: " + reason + "\n  rule: one-off\n"
	}
	put(t, f.root, "rules/adjustments/2026-10.yaml", adj(keyAt(3), "d", "人工舊", "舊調整"))
	cPrev := commitRules(t, f.root, "prev rules")
	put(t, f.root, "rules/adjustments/2026-10.yaml", "[]\n") // tidied away
	put(t, f.root, "rules/adjustments/2026-11.yaml", adj(keyAt(2), "c", "人工", "調整一")+adj(keyAt(9), "z", "別的", "調整二"))
	cCur := commitRules(t, f.root, "cur rules")

	prev := sideSpec{
		kr: list(item(1, "a"), item(2, "b"), item(3, "c"), item(4, "d")),
		zh: list(item(1, "甲 乙"), item(2, "乙"), item(3, "丙"), item(4, "人工舊")),
		recs: []store.Record{
			recAt(0, "a", "甲 乙", store.OriginParatranz, ""),
			recAt(1, "b", "乙", store.OriginClaude, "b0"),
			recAt(2, "c", "丙", store.OriginClaude, "b0"),
			recAt(3, "d", "人工舊", store.OriginHuman, "adjustment:2026-10.yaml"),
		},
		rules: cPrev,
	}.side(t)
	cur := sideSpec{
		kr: list(item(1, "a"), item(2, "b"), item(3, "c"), item(4, "D")),
		zh: list(item(1, "甲-乙"), item(2, "乙改"), item(3, "人工"), item(4, "丁新")),
		recs: []store.Record{
			recAt(0, "a", "甲-乙", store.OriginParatranz, ""),
			recAt(1, "b", "乙改", store.OriginClaude, "b1"),
			recAt(2, "c", "人工", store.OriginHuman, "adjustment:2026-11.yaml"),
			recAt(3, "D", "丁新", store.OriginClaude, "b1"),
		},
		applied: store.Applied{Rewrite: []string{"r1", "r2"}, Requeue: []string{"q1", "q2"}},
		rules:   cCur,
	}.side(t)

	rec := makeWith(t, prev, cur, CachedRules(f.root, t.TempDir()))
	if len(rec.Entries) != 4 {
		t.Fatalf("entries = %+v", rec.Entries)
	}
	wantCauses(t, rec.Entries[0], Cause{Kind: CauseKindRewrite, ID: "r1"})
	wantCauses(t, rec.Entries[1], Cause{Kind: CauseKindRequeue, ID: "q1", Batch: "b1"})
	wantCauses(t, rec.Entries[2], Cause{Kind: CauseKindAdjustment, ID: "2026-11.yaml", Reason: "調整一"})
	wantCauses(t, rec.Entries[3],
		Cause{Kind: CauseKindKRChanged, Batch: "b1"},
		Cause{Kind: CauseKindAdjustmentExpired, ID: "2026-10.yaml", Batch: "b1", Reason: "舊調整"})
	want := map[string]RuleInfo{
		"rewrite:r1":                      {Kind: CauseKindRewrite, ID: "r1", Path: "rules/rewrite/r1.yaml", Title: "改寫一", Commit: cCur},
		"requeue:q1":                      {Kind: CauseKindRequeue, ID: "q1", Path: "rules/requeue/q1.yaml", Title: "重翻一", Commit: cCur},
		"adjustment:2026-11.yaml":         {Kind: CauseKindAdjustment, ID: "2026-11.yaml", Path: "rules/adjustments/2026-11.yaml", Title: "2026-11.yaml", Commit: cCur},
		"adjustment-expired:2026-10.yaml": {Kind: CauseKindAdjustmentExpired, ID: "2026-10.yaml", Path: "rules/adjustments/2026-10.yaml", Title: "2026-10.yaml", Commit: cPrev},
	}
	if !reflect.DeepEqual(rec.Rules, want) {
		t.Errorf("Rules = %+v\nwant    %+v", rec.Rules, want)
	}
	for k, r := range rec.Rules {
		if k != RuleKey(r.Kind, r.ID) {
			t.Errorf("rule %q is keyed %q, want %q", r.ID, k, RuleKey(r.Kind, r.ID))
		}
	}
	if rec.Unknown != 0 || rec.CausesVersion != CausesVersion {
		t.Errorf("Unknown = %d, CausesVersion = %d", rec.Unknown, rec.CausesVersion)
	}
}

func TestMakeWithoutRulesInfersNothing(t *testing.T) {
	f := twoReleases(t)
	rec, err := Make(f.rev(t, relA), f.rev(t, relB), relB, timeB, testLang, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rec.CausesVersion != 0 || rec.Unknown != 0 || rec.Rules != nil || len(rec.Entries) == 0 {
		t.Errorf("record = %+v", rec)
	}
	for _, e := range rec.Entries {
		if e.Causes != nil {
			t.Errorf("%s has causes %+v", e.Pointer, e.Causes)
		}
	}
}

func TestCausesRequeueQueuedAgain(t *testing.T) {
	kr := list(item(1, "a"), item(2, "b"), item(3, "c"))
	two := func(z0, z1, z2 string) string { return list(item(1, z0), item(2, z1), item(3, z2)) }
	prev := sideSpec{
		kr: kr, zh: two("甲", "乙", "丙"),
		recs: []store.Record{
			recAt(0, "a", "甲", store.OriginClaude, "b0"),
			recAt(1, "b", "乙", store.OriginClaude, "b0"),
			recAt(2, "c", "丙", store.OriginClaude, "b0"),
		},
		applied: store.Applied{Requeue: []string{"q0"}},
		pending: []store.PendingItem{{Key: keyAt(1), Reason: requeue.Reason}},
	}.side(t)
	// Between the tags the units were translated, and then q1 sent the first
	// two back again: their new translations are still to come.
	cur := sideSpec{
		kr: kr, zh: two("甲改", "乙改", "丙改"),
		recs: []store.Record{
			recAt(0, "a", "甲改", store.OriginClaude, "b1"),
			recAt(1, "b", "乙改", store.OriginClaude, "b1"),
			recAt(2, "c", "丙改", store.OriginClaude, "b1"),
		},
		applied: store.Applied{Requeue: []string{"q0", "q1"}},
		pending: []store.PendingItem{{Key: keyAt(0), Reason: requeue.Reason}, {Key: keyAt(1), Reason: requeue.Reason}},
	}.side(t)
	all := requeue.Selector{Files: []string{"F.json"}}
	rules := &RuleFiles{Commit: commitCur, Requeues: []requeue.File{rq("q0", all), rq("q1", all)}}
	got := causesOf(t, prev, cur, rules, rules)
	if len(got) != 3 {
		t.Fatalf("entries = %+v", got)
	}
	// (b) passes over a unit queued again.
	wantCauses(t, got[0], Cause{Kind: CauseKindRetranslated, Batch: "b1"})
	// (a) passes over the requests applied since prev: q1 is still to come.
	wantCauses(t, got[1], Cause{Kind: CauseKindRequeue, ID: "q0", Batch: "b1"})
	// Not queued again: q1, applied since prev, sent it back.
	wantCauses(t, got[2], Cause{Kind: CauseKindRequeue, ID: "q1", Batch: "b1"})
}

func TestCausesRequeueBothWaysOnce(t *testing.T) {
	kr := list(item(1, "a"))
	all := requeue.Selector{Files: []string{"F.json"}}
	prev := sideSpec{
		kr: kr, zh: list(item(1, "甲")), recs: []store.Record{recAt(0, "a", "甲", store.OriginClaude, "b0")},
		applied: store.Applied{Requeue: []string{"q0"}},
		pending: []store.PendingItem{{Key: keyAt(0), Reason: requeue.Reason}},
	}.side(t)
	cur := sideSpec{
		kr: kr, zh: list(item(1, "甲改")), recs: []store.Record{recAt(0, "a", "甲改", store.OriginClaude, "b1")},
		applied: store.Applied{Requeue: []string{"q0", "q1"}},
	}.side(t)
	rules := &RuleFiles{Commit: commitCur, Requeues: []requeue.File{rq("q0", all), rq("q1", all)}}
	got := causesOf(t, prev, cur, rules, rules)
	if len(got) != 1 {
		t.Fatalf("entries = %+v", got)
	}
	// q1 is the last that selects it (a) and was applied since prev (b).
	wantCauses(t, got[0], Cause{Kind: CauseKindRequeue, ID: "q1", Batch: "b1"})
}

// manualHint is the hint of a requeue the pipeline workflow was asked for
// (lct requeue --reason): no request file has it as its reason.
const manualHint = "手動執行 workflow 要求重翻"

func TestCausesRequeueSelectsNone(t *testing.T) {
	kr := list(item(1, "a"))
	for name, tc := range map[string]struct {
		hint    string
		applied []string
		want    Cause  // when note is ""
		note    string // what the note of the one unknown cause says
	}{
		// Without a hint the request is the last that selects the unit:
		// there is none.
		"no hint": {applied: []string{"q1"}, note: "no request file applied selects it"},
		// A manual requeue: no request file sent it back.
		"a manual requeue": {hint: manualHint, applied: []string{"q1"}, want: Cause{Kind: CauseKindRetranslated, Batch: "b1", Reason: requeue.Reason}},
		// The request file that is gone may be the one that sent it back.
		"a manual requeue, a request file gone": {hint: manualHint, applied: []string{"q-gone", "q1"}, note: "request q-gone not found"},
	} {
		t.Run(name, func(t *testing.T) {
			applied := store.Applied{Requeue: tc.applied}
			prev := sideSpec{
				kr: kr, zh: list(item(1, "甲")), recs: []store.Record{recAt(0, "a", "甲", store.OriginClaude, "b0")},
				applied: applied, pending: []store.PendingItem{{Key: keyAt(0), Reason: requeue.Reason, Hint: tc.hint}},
			}.side(t)
			cur := sideSpec{
				kr: kr, zh: list(item(1, "甲改")), recs: []store.Record{recAt(0, "a", "甲改", store.OriginClaude, "b1")},
				applied: applied,
			}.side(t)
			rules := &RuleFiles{Commit: commitCur, Requeues: []requeue.File{rq("q1", requeue.Selector{Keys: []string{keyAt(5)}})}}
			got := causesOf(t, prev, cur, rules, rules)
			if len(got) != 1 {
				t.Fatalf("entries = %+v", got)
			}
			if tc.note == "" {
				wantCauses(t, got[0], tc.want)
				return
			}
			if c := got[0].Causes; len(c) != 1 || c[0].Kind != CauseKindUnknown || !strings.Contains(c[0].Note, tc.note) {
				t.Errorf("causes = %+v, want one unknown that says %q", c, tc.note)
			}
		})
	}
}

// A manual requeue of the pipeline workflow (no request file) in one Release,
// translated in the next: an old request file that selects the unit too is
// not its cause.
func TestCausesRequeueManualAcrossReleases(t *testing.T) {
	kr := list(item(1, "a"), item(2, "b"))
	all := requeue.Selector{Files: []string{"F.json"}}
	applied := store.Applied{Requeue: []string{"q-old"}} // done long before prev
	prev := sideSpec{
		kr: kr, zh: list(item(1, "甲"), item(2, "乙")),
		recs:    []store.Record{recAt(0, "a", "甲", store.OriginClaude, "b0"), recAt(1, "b", "乙", store.OriginClaude, "b0")},
		applied: applied,
		pending: []store.PendingItem{
			{Key: keyAt(0), Reason: requeue.Reason, Hint: manualHint},
			{Key: keyAt(1), Reason: requeue.Reason, Hint: " 重翻 q-old "}, // still waiting for q-old
		},
		rules: commitPrev,
	}.side(t)
	cur := sideSpec{
		kr: kr, zh: list(item(1, "甲改"), item(2, "乙改")),
		recs:    []store.Record{recAt(0, "a", "甲改", store.OriginClaude, "b1"), recAt(1, "b", "乙改", store.OriginClaude, "b1")},
		applied: applied, rules: commitCur,
	}.side(t)
	rules := &RuleFiles{Commit: commitCur, Requeues: []requeue.File{rq("q-old", all)}}
	got := causesOf(t, prev, cur, rules, rules)
	if len(got) != 2 {
		t.Fatalf("entries = %+v", got)
	}
	wantCauses(t, got[0], Cause{Kind: CauseKindRetranslated, Batch: "b1", Reason: requeue.Reason})
	wantCauses(t, got[1], Cause{Kind: CauseKindRequeue, ID: "q-old", Batch: "b1"})
}

// The hint names the request that sent the unit back last, which need not be
// the last that selects it; its reason may have changed since, so either
// rules commit's reason will do.
func TestCausesRequeueByHint(t *testing.T) {
	kr := list(item(1, "a"), item(2, "b"), item(3, "c"))
	all := requeue.Selector{Files: []string{"F.json"}}
	applied := store.Applied{Requeue: []string{"q0", "q1"}}
	prev := sideSpec{
		kr: kr, zh: list(item(1, "甲"), item(2, "乙"), item(3, "丙")),
		recs: []store.Record{
			recAt(0, "a", "甲", store.OriginClaude, "b0"),
			recAt(1, "b", "乙", store.OriginClaude, "b0"),
			recAt(2, "c", "丙", store.OriginClaude, "b0"),
		},
		applied: applied,
		pending: []store.PendingItem{
			{Key: keyAt(0), Reason: requeue.Reason, Hint: "重翻 q0"},
			{Key: keyAt(1), Reason: requeue.Reason, Hint: "q1 舊理由"},
			{Key: keyAt(2), Reason: "structural-fail", Hint: "重翻 q0"}, // reset by q0
		},
		rules: commitPrev,
	}.side(t)
	cur := sideSpec{
		kr: kr, zh: list(item(1, "甲改"), item(2, "乙改"), item(3, "丙改")),
		recs: []store.Record{
			recAt(0, "a", "甲改", store.OriginClaude, "b1"),
			recAt(1, "b", "乙改", store.OriginClaude, "b1"),
			recAt(2, "c", "丙改", store.OriginClaude, "b1"),
		},
		applied: applied, rules: commitCur,
	}.side(t)
	q1old := rq("q1", all)
	q1old.Reason = "q1 舊理由"
	prevRules := &RuleFiles{Commit: commitPrev, Requeues: []requeue.File{rq("q0", all), q1old}}
	curRules := &RuleFiles{Commit: commitCur, Requeues: []requeue.File{rq("q0", all), rq("q1", all)}}
	got := causesOf(t, prev, cur, prevRules, curRules)
	if len(got) != 3 {
		t.Fatalf("entries = %+v", got)
	}
	wantCauses(t, got[0], Cause{Kind: CauseKindRequeue, ID: "q0", Batch: "b1"})
	wantCauses(t, got[1], Cause{Kind: CauseKindRequeue, ID: "q1", Batch: "b1"})
	wantCauses(t, got[2], Cause{Kind: CauseKindRequeue, ID: "q0", Batch: "b1"})
}

// A unit queued for another reason that a request reset from needs_review
// carries the request's hint; one a manual requeue reset does not.
func TestCausesRequeueHintOnResetUnit(t *testing.T) {
	kr := list(item(1, "a"), item(2, "b"))
	all := requeue.Selector{Files: []string{"F.json"}}
	applied := store.Applied{Requeue: []string{"q0"}}
	prev := sideSpec{
		kr: kr, zh: list(item(1, "甲"), item(2, "乙")),
		recs:    []store.Record{recAt(0, "a", "甲", store.OriginClaude, "b0"), recAt(1, "b", "乙", store.OriginClaude, "b0")},
		applied: applied,
		pending: []store.PendingItem{
			{Key: keyAt(0), Reason: "structural-fail", Hint: "重翻 q0"},
			{Key: keyAt(1), Reason: "structural-fail", Hint: manualHint},
		},
	}.side(t)
	cur := sideSpec{
		kr: kr, zh: list(item(1, "甲改"), item(2, "乙改")),
		recs:    []store.Record{recAt(0, "a", "甲改", store.OriginClaude, "b1"), recAt(1, "b", "乙改", store.OriginClaude, "b1")},
		applied: applied,
	}.side(t)
	rules := &RuleFiles{Commit: commitCur, Requeues: []requeue.File{rq("q0", all)}}
	got := causesOf(t, prev, cur, rules, rules)
	if len(got) != 2 {
		t.Fatalf("entries = %+v", got)
	}
	wantCauses(t, got[0], Cause{Kind: CauseKindRequeue, ID: "q0", Batch: "b1"})
	wantCauses(t, got[1], Cause{Kind: CauseKindRetranslated, Batch: "b1", Reason: "structural-fail"})
}

func TestCausesRequeueUnreadableIsUndecided(t *testing.T) {
	prev := sideSpec{
		kr: list(item(1, "a"), item(2, "b")), zh: list(item(1, "甲"), item(2, "b")),
		recs: []store.Record{recAt(0, "a", "甲", store.OriginClaude, "b0")}, // id 2 stands in
	}.side(t)
	cur := sideSpec{
		kr: list(item(1, "a"), item(2, "b")), zh: list(item(1, "甲改"), item(2, "乙")),
		recs:    []store.Record{recAt(0, "a", "甲改", store.OriginClaude, "b1"), recAt(1, "b", "乙", store.OriginClaude, "b1")},
		applied: store.Applied{Requeue: []string{"q-gone"}},
	}.side(t)
	rules := &RuleFiles{Commit: commitCur}
	got := causesOf(t, prev, cur, rules, rules)
	if len(got) != 2 {
		t.Fatalf("entries = %+v", got)
	}
	// q-gone may have sent the unit back: whether it did cannot be told.
	if c := got[0].Causes; len(c) != 1 || c[0].Kind != CauseKindUnknown || !strings.Contains(c[0].Note, "request q-gone not found") {
		t.Errorf("causes = %+v, want one unknown about q-gone", c)
	}
	// A unit without a record could not be selected: late, as before.
	wantCauses(t, got[1], Cause{Kind: CauseKindLate, Batch: "b1"})
}

func TestCausesRewriteUnreadableSparesAdjustments(t *testing.T) {
	kr := list(item(1, "a"), item(2, "b"), item(3, "c"))
	prev := sideSpec{
		kr: kr, zh: list(item(1, "人工一"), item(2, "甲"), item(3, "丙")),
		recs: []store.Record{
			recAt(0, "a", "人工一", store.OriginHuman, "adjustment:F.yaml"),
			recAt(1, "b", "甲", store.OriginClaude, "reuse:F.json|/dataList/9/desc"),
			recAt(2, "c", "丙", store.OriginParatranz, ""),
		},
	}.side(t)
	cur := sideSpec{
		kr: kr, zh: list(item(1, "人工二"), item(2, "甲改"), item(3, "丙改")),
		recs: []store.Record{
			recAt(0, "a", "人工二", store.OriginHuman, "adjustment:F.yaml"), // the adjustment was edited
			recAt(1, "b", "甲改", store.OriginClaude, "reuse:F.json|/dataList/9/desc"),
			recAt(2, "c", "丙改", store.OriginParatranz, ""),
		},
		applied: store.Applied{Rewrite: []string{"r-gone"}},
	}.side(t)
	rules := &RuleFiles{Commit: commitCur, Adjustments: []adjust.Adjustment{adjustment("F.yaml", keyAt(0), "a", "人工二", "改正")}}
	got := causesOf(t, prev, cur, rules, rules)
	if len(got) != 3 {
		t.Fatalf("entries = %+v", got)
	}
	// A human record is no rewrite of a file that names no origins.
	wantCauses(t, got[0], Cause{Kind: CauseKindAdjustment, ID: "F.yaml", Reason: "改正"})
	// The reused text may have been rewritten by r-gone: the reuse made
	// before prev is no cause of it, and the rewrite cannot be told.
	for _, e := range got[1:] {
		if len(e.Causes) != 1 || e.Causes[0].Kind != CauseKindUnknown || !strings.Contains(e.Causes[0].Note, "rewrite r-gone not found") {
			t.Errorf("%s: causes = %+v, want one unknown about r-gone", e.Pointer, e.Causes)
		}
	}
}

func TestCausesAdjustmentToAnotherFile(t *testing.T) {
	kr := list(item(1, "a"))
	prev := sideSpec{kr: kr, zh: list(item(1, "人工一")), recs: []store.Record{recAt(0, "a", "人工一", store.OriginHuman, "adjustment:A.yaml")}}.side(t)
	cur := sideSpec{kr: kr, zh: list(item(1, "人工二")), recs: []store.Record{recAt(0, "a", "人工二", store.OriginHuman, "adjustment:B.yaml")}}.side(t)
	prevRules := &RuleFiles{Commit: commitPrev, Adjustments: []adjust.Adjustment{adjustment("A.yaml", keyAt(0), "a", "人工一", "一")}}
	curRules := &RuleFiles{Commit: commitCur, Adjustments: []adjust.Adjustment{adjustment("B.yaml", keyAt(0), "a", "人工二", "二")}}
	got := causesOf(t, prev, cur, prevRules, curRules)
	if len(got) != 1 {
		t.Fatalf("entries = %+v", got)
	}
	// B took over from A: that is B's adjustment, not A expiring.
	wantCauses(t, got[0], Cause{Kind: CauseKindAdjustment, ID: "B.yaml", Reason: "二"})
}

func TestMakeRulesPreferCurCommit(t *testing.T) {
	kr := list(item(1, "a"), item(2, "b"))
	prev := sideSpec{
		kr: kr, zh: list(item(1, "甲"), item(2, "乙")),
		recs:  []store.Record{recAt(0, "a", "甲", store.OriginClaude, "b0"), recAt(1, "b", "乙", store.OriginClaude, "b0")},
		rules: commitPrev,
	}.side(t)
	cur := sideSpec{
		kr: kr, zh: list(item(1, "人工甲"), item(2, "人工乙")),
		recs: []store.Record{
			recAt(0, "a", "人工甲", store.OriginHuman, "adjustment:F.yaml"),
			recAt(1, "b", "人工乙", store.OriginHuman, "adjustment:F.yaml"),
		},
		rules: commitCur,
	}.side(t)
	// The first unit's entry is only in prev's F.yaml, the second's in cur's.
	rec := makeWith(t, prev, cur, rulesFrom(map[string]*RuleFiles{
		commitPrev: {Commit: commitPrev, Adjustments: []adjust.Adjustment{adjustment("F.yaml", keyAt(0), "a", "人工甲", "舊")}},
		commitCur:  {Commit: commitCur, Adjustments: []adjust.Adjustment{adjustment("F.yaml", keyAt(1), "b", "人工乙", "新")}},
	}))
	wantCauses(t, rec.Entries[0], Cause{Kind: CauseKindAdjustment, ID: "F.yaml", Reason: "舊"})
	wantCauses(t, rec.Entries[1], Cause{Kind: CauseKindAdjustment, ID: "F.yaml", Reason: "新"})
	if got := rec.Rules[RuleKey(CauseKindAdjustment, "F.yaml")].Commit; got != commitCur || len(rec.Rules) != 1 {
		t.Errorf("Rules = %+v, want F.yaml as of %s", rec.Rules, commitCur)
	}
}

func TestNewSideRules(t *testing.T) {
	read := &RuleFiles{Commit: commitCur}
	for name, tc := range map[string]struct {
		got        sideRules
		commit     string
		rules, err bool
	}{
		"the rules name their commit": {newSideRules("", read, nil), commitCur, true, false},
		"rules without a commit":      {newSideRules(commitPrev, &RuleFiles{}, nil), commitPrev, true, false},
		"refused":                     {newSideRules(commitPrev, nil, fmt.Errorf("boom")), commitPrev, false, true},
		"not read":                    {newSideRules(commitPrev, nil, nil), commitPrev, false, true},
	} {
		if tc.got.commit != tc.commit || (tc.got.rules != nil) != tc.rules || (tc.got.err != nil) != tc.err {
			t.Errorf("%s: %+v", name, tc.got)
		}
	}
}
