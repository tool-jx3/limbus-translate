package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/batch"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

func runPreview(t *testing.T, root string, jsonOut bool, args ...string) (string, error) {
	t.Helper()
	ctx, out := testCtx(t, root, "", jsonOut)
	err := runBatchPreview(ctx, args)
	return out.String(), err
}

func TestBatchPreviewJSON(t *testing.T) {
	root, _ := batchRepo(t)
	out, err := runPreview(t, root, true, batchKey0)
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &top); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	var keys []string
	for k := range top {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if got := strings.Join(keys, ","); got != "batch,check,notes,pending,record,state" {
		t.Errorf("keys = %s", got)
	}
	var b struct{ ID string }
	if err := json.Unmarshal(top["batch"], &b); err != nil || b.ID != "preview" {
		t.Errorf("batch id = %q, %v", b.ID, err)
	}
}

func TestBatchPreviewText(t *testing.T) {
	root, st := batchRepo(t)
	const kr = "특정 추출 시작"
	writeTestFile(t, filepath.Join(root, ".work", "raw"), "kr/KR_Term.json", `{"dataList":[{"id":1,"content":"`+kr+`"}]}`)
	const key = "Term.json|/dataList/0/content"
	if err := st.SaveRecords(map[string]store.Record{
		key: {Key: key, KRSha: units.KRSha(kr), ZH: "開始特定提取", Origin: store.OriginParatranz, Since: "raw-20261029"},
	}); err != nil {
		t.Fatal(err)
	}
	out, err := runPreview(t, root, false, key)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"family", "guides/", "특정 추출 → 特定提取", "check", "paratranz"} {
		if !strings.Contains(out, want) {
			t.Errorf("text lacks %q:\n%s", want, out)
		}
	}
}

func TestBatchPreviewCopyIsUserError(t *testing.T) {
	root, st := batchRepo(t)
	if err := st.SaveRecords(map[string]store.Record{
		batchKey1: {Key: batchKey1, KRSha: units.KRSha("뫼르소가 말했다"), ZH: "뫼르소가 말했다", Origin: store.OriginCopy, Since: "raw-20261029"},
	}); err != nil {
		t.Fatal(err)
	}
	_, err := runPreview(t, root, true, batchKey1)
	if !isUserError(err) || !strings.Contains(err.Error(), "copy") {
		t.Errorf("err = %v, want a user error that mentions copy", err)
	}
}

func TestBatchPreviewUnknownKey(t *testing.T) {
	root, _ := batchRepo(t)
	for _, args := range [][]string{{"Nope.json|/dataList/0/content"}, {"Sample.json|/dataList/9/content"}, {"bad"}, {}, {batchKey0, batchKey1}} {
		if _, err := runPreview(t, root, true, args...); !isUserError(err) {
			t.Errorf("args %v: err = %v, want a user error", args, err)
		}
	}
}

// The text of "batch preview", branch by branch, from previews made by hand.
func TestPreviewReportText(t *testing.T) {
	const key = "Sample.json|/dataList/0/content"
	preview := func(mod func(p *batch.UnitPreview)) *batch.UnitPreview {
		p := &batch.UnitPreview{
			Batch:   &batch.Batch{Family: "story", Keywords: map[string]string{}},
			State:   batch.PreviewState{Pending: true, RulesCommit: "abc1234"},
			Pending: store.PendingItem{Key: key, Reason: "new"},
		}
		mod(p)
		return p
	}
	rec := func(rules string) *store.Record {
		return &store.Record{Key: key, ZH: "譯文", Origin: store.OriginClaude, Model: "m1", Batch: "b1", Rules: rules}
	}
	for _, c := range []struct {
		name string
		mod  func(p *batch.UnitPreview)
		want []string
		not  []string
	}{
		{"new unit, nothing else", func(p *batch.UnitPreview) {}, []string{
			"preview of " + key, "family: story", "pending: new", "record: none", "check: nothing to check",
		}, []string{"hint:", "draft:", "state:", "not in the pending list", "glossary_check"}},
		{"reason, hint and draft", func(p *batch.UnitPreview) {
			p.Pending.Reason, p.Pending.Hint, p.Pending.Draft = "kr-changed", "再正式一點", "草稿"
		}, []string{"pending: kr-changed", "  hint: 再正式一點", "  draft: 草稿"}, nil},
		{"glossary_check", func(p *batch.UnitPreview) { p.Batch.GlossaryCheck = "warn" }, []string{"glossary_check: warn"}, nil},
		{"not pending", func(p *batch.UnitPreview) { p.State.Pending = false }, []string{"  (not in the pending list: this is what a requeue or the plan would make)"}, nil},
		{"needs_review", func(p *batch.UnitPreview) { p.State.NeedsReview = true }, []string{"state: needs_review (waits for a person, is not handed out now)"}, nil},
		{"leased", func(p *batch.UnitPreview) { p.State.LeasedTo = "20261101-0002" }, []string{"state: leased to 20261101-0002"}, nil},
		{"frozen", func(p *batch.UnitPreview) { p.State.Frozen = true }, []string{"state: frozen (the file is not sent for translation)"}, nil},
		{"record under the same rules", func(p *batch.UnitPreview) { p.Record = rec("abc1234") },
			[]string{"record: origin claude, model m1, batch b1, rules abc1234 (same as the current rules)"}, nil},
		{"record under other rules", func(p *batch.UnitPreview) { p.Record = rec("old9999") },
			[]string{"record: origin claude, model m1, batch b1, rules old9999 (differs from the current rules)"}, nil},
		{"record with no model, batch or rules", func(p *batch.UnitPreview) {
			r := rec("")
			r.Model, r.Batch = "", ""
			p.Record = r
		}, []string{"record: origin claude, model -, batch -, rules - (differs from the current rules)"}, nil},
		{"notes", func(p *batch.UnitPreview) { p.Notes = []store.Note{{Note: "第一則"}, {Note: "第二則"}} }, []string{"note: 第一則", "note: 第二則"}, nil},
		{"check with issues, stale KR", func(p *batch.UnitPreview) {
			p.Record = rec("abc1234")
			p.Check = &batch.Check{Verdict: "reject", StaleKR: true, Issues: []validate.Issue{
				{Code: "hangul", Severity: validate.SeverityHard, Class: "structure", Message: "still Hangul"},
				{Code: "term-missing", Severity: validate.SeverityWarn, Class: "glossary", Message: "威力 is missing"},
			}}
		}, []string{
			"check: reject", "  the translation was made from an older KR",
			"  hangul [hard] [structure] still Hangul", "  term-missing [warn] [glossary] 威力 is missing",
		}, []string{"nothing to check"}},
		{"issue lines name their class", func(p *batch.UnitPreview) {
			p.Record = rec("abc1234")
			p.Check = &batch.Check{Verdict: "reject", Issues: []validate.Issue{
				{Code: "hangul", Severity: validate.SeverityHard, Class: "structure", Message: "still Hangul"},
				{Code: "term-missing", Severity: validate.SeverityWarn, Class: "glossary", Message: "威力 is missing"},
			}}
		}, []string{"  hangul [hard] [structure] still Hangul", "  term-missing [warn] [glossary] 威力 is missing"}, nil},
		{"glossary marks provisional and hard terms", func(p *batch.UnitPreview) {
			p.Batch.Glossary = []rules.Term{
				{KR: "일반", ZH: "一般", Status: "approved"},
				{KR: "임시", ZH: "暫定", Status: "approved", Provisional: true},
				{KR: "강제", ZH: "強制", Status: "approved", Check: rules.CheckHard},
				{KR: "보통", ZH: "普通", Status: "approved", Check: rules.CheckOff},
			}
		}, []string{"  일반 → 一般", "  임시 → 暫定  [provisional]", "  강제 → 強制  [check: hard]", "  보통 → 普通  [check: off]"}, nil},
		{"record since and flags", func(p *batch.UnitPreview) {
			r := rec("abc1234")
			r.Since, r.Flags = "raw-20261101", []string{"fallback-en", "x"}
			p.Record = r
		}, []string{"  since: raw-20261101", "  flags: fallback-en, x"}, nil},
		{"record with no since or flags", func(p *batch.UnitPreview) { p.Record = rec("abc1234") }, nil, []string{"since:", "flags:"}},
		{"pending attempts and last error", func(p *batch.UnitPreview) {
			p.Pending.Attempts, p.Pending.LastError = 2, "hangul: still Hangul"
		}, []string{"  attempts: 2", "  last_error: hangul: still Hangul"}, nil},
		{"pending with no attempts or error", func(p *batch.UnitPreview) {}, nil, []string{"attempts:", "last_error:"}},
		{"accepted check", func(p *batch.UnitPreview) {
			p.Record = rec("abc1234")
			p.Check = &batch.Check{Verdict: "accept"}
		}, []string{"check: accept"}, []string{"older KR"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := previewReport{UnitPreview: preview(c.mod)}.String()
			lines := strings.Split(got, "\n") // a want is a whole line, the first one too
			for _, w := range c.want {
				if !slices.Contains(lines, w) {
					t.Errorf("text lacks the line %q:\n%s", w, got)
				}
			}
			for _, n := range c.not {
				if strings.Contains(got, n) {
					t.Errorf("text has %q:\n%s", n, got)
				}
			}
		})
	}
}

// The message names the key once, whether the key is well formed or not.
func TestBatchPreviewNotInKRMessage(t *testing.T) {
	root, _ := batchRepo(t)
	for _, c := range []struct{ key, want string }{
		{"Nope.json|/dataList/0/content", "batch preview: Nope.json|/dataList/0/content is not a unit of the KR pack"},
		{"Sample.json|/dataList/9/content", "batch preview: Sample.json|/dataList/9/content is not a unit of the KR pack"},
		{"bad", `batch preview: bad is not a unit of the KR pack (missing "|")`},
	} {
		_, err := runPreview(t, root, true, c.key)
		if !isUserError(err) || err.Error() != c.want {
			t.Errorf("key %q: err = %v, want the user error %q", c.key, err, c.want)
		}
	}
}

func readMeta(t *testing.T, root string) map[string]string {
	t.Helper()
	got := map[string]string{}
	dir := filepath.Join(root, ".work", "tw", ".meta")
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(p)
		got[p] = string(data)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestBatchPreviewReadOnly(t *testing.T) {
	root, _ := batchRepo(t)
	before := readMeta(t, root)
	for _, key := range []string{batchKey0, batchKey1} {
		if _, err := runPreview(t, root, true, key); err != nil {
			t.Fatal(err)
		}
		if _, err := runPreview(t, root, false, key); err != nil {
			t.Fatal(err)
		}
	}
	after := readMeta(t, root)
	if len(before) != len(after) {
		t.Fatalf("files %d -> %d", len(before), len(after))
	}
	for p, b := range before {
		if after[p] != b {
			t.Errorf("%s changed", p)
		}
	}
}

func TestBatchPreviewKeywordSpace(t *testing.T) {
	root, st := keywordRepo(t)
	recs, err := st.LoadRecords()
	if err != nil {
		t.Fatal(err)
	}
	recs[statusKey] = store.Record{Key: statusKey, KRSha: units.KRSha(statusKR), ZH: "使目標增加2級破裂強度", Origin: store.OriginClaude, Since: "raw-20261029"}
	if err := st.SaveRecords(recs); err != nil {
		t.Fatal(err)
	}
	out, err := runPreview(t, root, true, statusKey)
	if err != nil {
		t.Fatal(err)
	}
	var got struct{ Check struct{ ZH string } }
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if got.Check.ZH != "使目標增加2級破裂 強度" {
		t.Errorf("check.zh = %q, want the space after 破裂", got.Check.ZH)
	}
}
