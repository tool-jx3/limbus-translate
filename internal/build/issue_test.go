package build

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/tool-jx3/limbus-translate/internal/existing"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/itemid"
	"github.com/tool-jx3/limbus-translate/internal/plan"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/translate"
)

// countUTF16 is the length GitHub limits: UTF-16 code units, counted here by
// encoding, not with the function under test.
func countUTF16(s string) int { return len(utf16.Encode([]rune(s))) }

// itemLines are the lines of text that start a checklist item.
func itemLines(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "- [ ] ") {
			out = append(out, line)
		}
	}
	return out
}

func issueInfo() Info {
	return Info{RawTag: "raw-20261105", RawToken: "tok", RulesCommit: "abc1234"}
}

func reviewPending(n int, errText func(i int) string) []store.PendingItem {
	var out []store.PendingItem
	for i := range n {
		out = append(out, store.PendingItem{
			Key:       fmt.Sprintf("StoryData/S%04d.json|/dataList/%d/content", i%50, i),
			Reason:    "new",
			Attempts:  3,
			Flags:     []string{store.FlagNeedsReview},
			LastError: errText(i),
		})
	}
	return out
}

// Nothing for a person to do is no issue, however much there is to see in the
// report.
func TestReviewIssueOnlyWithItems(t *testing.T) {
	info := issueInfo()
	info.Model, info.CostUSD, info.Stopped = "model-x", 1.5, translate.StopDone
	info.Report = Report{Pending: 40, NeedsReview: 0, StructuralFallback: 3, Unaligned: 2, Warnings: map[string]int{"kana": 4}}
	sum := &plan.Summary{
		RawTag: "raw-20261105", PrevRawTag: "raw-20261029",
		Changes:     plan.Changes{Changed: 5, New: 6, Deleted: 7, FilesAdded: []string{"New.json"}},
		Adjustments: plan.AdjustmentSummary{Applied: 2, Moved: []plan.MovedKey{{From: "A.json|/a", To: "A.json|/b"}}},
	}
	pending := []store.PendingItem{
		{Key: "A.json|/dataList/0/content", Reason: "new"},
		{Key: "A.json|/dataList/1/content", Reason: "kr-changed", Attempts: 2, LastError: "x: y"}, // still being tried
	}
	for name, s := range map[string]*plan.Summary{"with a plan": sum, "without a plan": nil} {
		title, parts, ok := ReviewIssue(info, s, pending)
		if ok || title != "" || len(parts) != 0 {
			t.Errorf("%s: ReviewIssue = %q, %d parts, %v, want no issue", name, title, len(parts), ok)
		}
	}
	// A translation loop that stopped for another reason than an error is not
	// an item.
	for _, stopped := range []string{"", translate.StopDone, translate.StopCost, translate.StopDeadline, translate.StopNoProgress} {
		info.Stopped = stopped
		if _, _, ok := ReviewIssue(info, sum, pending); ok {
			t.Errorf("stopped %q: ReviewIssue made an issue", stopped)
		}
	}
}

// One item of each kind: a checklist line each, a title from the day of the
// snapshot, and the place of the full report.
func TestReviewIssueChecklist(t *testing.T) {
	info := issueInfo()
	info.Stopped, info.StopError = translate.StopError, "exit status 1: boom"
	info.Report.FrozenFallback = []string{"BgmLyrics/L.json|/dataList/3/content"}
	sum := &plan.Summary{
		RawTag: "raw-20261105", PrevRawTag: "raw-20261029",
		Adjustments:           plan.AdjustmentSummary{Expired: []string{"StoryData/S1.json|/dataList/4/content"}},
		FrozenChanged:         []string{"BgmLyrics/L.json|/dataList/9/content"},
		NewSpeakers:           []string{"신규 NPC"},
		FieldChanges:          []fields.Change{{Op: "add", Group: "Skills", Field: "tag"}},
		NewFilesDefaultFamily: []string{"Brand/New.json"},
	}
	pending := append(reviewPending(1, func(int) string { return "id-mismatch: [Bleed] is missing" }),
		store.PendingItem{Key: "A.json|/dataList/0/content", Reason: "new"}) // not an item

	title, parts, ok := ReviewIssue(info, sum, pending)
	if !ok {
		t.Fatal("ReviewIssue made no issue")
	}
	if title != "審核：2026-11-05 更新" {
		t.Errorf("title = %q", title)
	}
	if len(parts) != 1 {
		t.Fatalf("%d parts, want 1", len(parts))
	}
	body := parts[0]
	lines := itemLines(body)
	if len(lines) != 8 {
		t.Errorf("%d items, want one of each of the 8 kinds:\n%s", len(lines), body)
	}
	for _, want := range []string{
		"StoryData/S0000.json|/dataList/0/content", "id-mismatch: [Bleed] is missing", // needs_review and why
		"StoryData/S1.json|/dataList/4/content", // adjustment expired
		"BgmLyrics/L.json|/dataList/9/content",  // frozen KR changed
		"BgmLyrics/L.json|/dataList/3/content",  // frozen fallback
		"신규 NPC",                                // new speaker
		"+ group_fields: Skills.tag",            // field rule
		"Brand/New.json",                        // default family
		"exit status 1: boom",                   // the translation loop
		".meta/reports/raw-20261105.md",         // the full report
	} {
		if !strings.Contains(body, want) {
			t.Errorf("issue lacks %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "`A.json|/dataList/0/content`") {
		t.Errorf("a pending item that is not flagged is listed:\n%s", body)
	}
	// One part: no part header.
	if strings.Contains(body, "（第") {
		t.Errorf("a single part has a part header:\n%s", body)
	}

	// The tag of a second snapshot of the day carries the day too; one that
	// does not hold a date is used as it is.
	for tag, want := range map[string]string{
		"raw-20261105.2": "審核：2026-11-05 更新",
		"raw-nonsense":   "審核：raw-nonsense 更新",
	} {
		i := info
		i.RawTag = tag
		if got, _, ok := ReviewIssue(i, sum, pending); !ok || got != want {
			t.Errorf("title of %s = %q, %v, want %q", tag, got, ok, want)
		}
	}

	// A needs_review item without a recorded reason says so.
	noReason := reviewPending(1, func(int) string { return "" })
	if _, parts, _ := ReviewIssue(info, nil, noReason); !strings.Contains(parts[0], "/dataList/0/content`：（沒有記錄原因）") {
		t.Errorf("an item without a reason does not say so:\n%s", parts[0])
	}

	// The reason holds the tags of the game text and perhaps a backtick: it
	// is a code span, so that none of it is read as HTML or Markdown.
	tricky := reviewPending(1, func(int) string { return "tag <color=#ff0000> and `code`\nnext line" })
	_, parts, _ = ReviewIssue(info, nil, tricky)
	if want := "：`` tag <color=#ff0000> and `code` next line ``\n"; !strings.Contains(parts[0], want) {
		t.Errorf("the reason is not a code span on one line (want %q):\n%s", want, parts[0])
	}
}

// 3,000 units that need a person do not fit in one issue: they are cut into
// parts of at most 60,000 characters, between two items, each saying which
// part it is; none is lost or cut in two.
func TestReviewIssueSplits(t *testing.T) {
	info := issueInfo()
	// Han characters and emoji: a character count that does not look at
	// UTF-16 would let parts through that GitHub refuses.
	pending := reviewPending(3000, func(i int) string {
		return fmt.Sprintf("id-mismatch: 缺少 [Bleed] 😀 %d", i)
	})
	title, parts, ok := ReviewIssue(info, nil, pending)
	if !ok || title == "" {
		t.Fatalf("ReviewIssue = %q, %d parts, %v", title, len(parts), ok)
	}
	if len(parts) < 2 {
		t.Fatalf("%d parts, want several", len(parts))
	}

	var all []string
	for i, p := range parts {
		if n := countUTF16(p); n > 60000 {
			t.Errorf("part %d has %d UTF-16 code units, over 60,000", i+1, n)
		}
		if want := fmt.Sprintf("（第 %d／%d 段）", i+1, len(parts)); !strings.HasPrefix(p, want) {
			t.Errorf("part %d starts %q, want %q", i+1, p[:min(40, len(p))], want)
		}
		all = append(all, itemLines(p)...)
	}
	if !strings.Contains(parts[0], ".meta/reports/raw-20261105.md") {
		t.Errorf("the first part does not name the full report:\n%s", parts[0][:300])
	}

	// Every item once and whole.
	if len(all) != 3000 {
		t.Fatalf("%d items in all, want 3000", len(all))
	}
	for i, p := range pending {
		line := fmt.Sprintf("- [ ] `%s`：`%s`", p.Key, p.LastError)
		if all[i] != line {
			t.Fatalf("item %d = %q, want %q", i, all[i], line)
		}
	}
	// Counting the marker over the text, not the lines, gives the same.
	total := 0
	for _, p := range parts {
		total += strings.Count(p, "- [ ] ")
	}
	if total != 3000 {
		t.Errorf("%d checklist markers, want 3000", total)
	}
}

// The unit of the limit is the UTF-16 code unit: a part made only of emoji
// (two units each) must be cut by that count, not by runes or bytes.
func TestReviewIssueCountsUTF16(t *testing.T) {
	info := issueInfo()
	long := strings.Repeat("😀", 150) // 300 UTF-16 code units, 150 runes, 600 bytes
	pending := reviewPending(400, func(int) string { return long })
	_, parts, ok := ReviewIssue(info, nil, pending)
	if !ok {
		t.Fatal("no issue")
	}
	items := 0
	for i, p := range parts {
		if n := countUTF16(p); n > 60000 {
			t.Errorf("part %d has %d UTF-16 code units", i+1, n)
		}
		items += len(itemLines(p))
	}
	if items != 400 {
		t.Errorf("%d items, want 400", items)
	}
	// 400 items of about 350 code units each are 140,000 code units: three
	// parts. Counted in runes they would fit in two (84,000), counted in bytes
	// they would need five (about 260,000).
	if len(parts) != 3 {
		t.Errorf("%d parts, want 3", len(parts))
	}
}

// An item longer than a whole part is cut, and says so; the items around it
// are not touched.
func TestReviewIssueTruncatesAnItemOverTheLimit(t *testing.T) {
	info := issueInfo()
	sum := &plan.Summary{
		RawTag: "raw-20261105", PrevRawTag: "raw-20261029",
		NewSpeakers: []string{"before", strings.Repeat("長", 100000), "after"},
	}
	_, parts, ok := ReviewIssue(info, sum, nil)
	if !ok {
		t.Fatal("no issue")
	}
	var all []string
	for i, p := range parts {
		if n := countUTF16(p); n > 60000 {
			t.Errorf("part %d has %d UTF-16 code units", i+1, n)
		}
		all = append(all, itemLines(p)...)
	}
	if len(all) != 3 {
		t.Fatalf("%d items, want 3:\n%v", len(all), parts)
	}
	if !strings.Contains(all[0], "before") || !strings.Contains(all[2], "after") {
		t.Errorf("the items around the long one changed: %q, %q", all[0], all[2])
	}
	if !strings.Contains(all[1], "已截斷") || countUTF16(all[1]) >= 60000 {
		t.Errorf("the long item is not cut and marked (length %d)", countUTF16(all[1]))
	}
}

// issueWithEveryKind is one of each of the eight kinds of item.
func issueWithEveryKind() (Info, *plan.Summary, []store.PendingItem) {
	info := issueInfo()
	info.Stopped, info.StopError = translate.StopError, "exit status 1: boom"
	info.Report.FrozenFallback = []string{"BgmLyrics/L.json|/dataList/3/content"}
	sum := &plan.Summary{
		RawTag: "raw-20261105", PrevRawTag: "raw-20261029",
		Adjustments:           plan.AdjustmentSummary{Expired: []string{"StoryData/S1.json|/dataList/4/content", "StoryData/S1.json|/dataList/7/content"}},
		FrozenChanged:         []string{"BgmLyrics/L.json|/dataList/9/content"},
		NewSpeakers:           []string{"신규 NPC", "또 다른 NPC"},
		FieldChanges:          []fields.Change{{Op: "add", Group: "Skills", Field: "tag"}},
		NewFilesDefaultFamily: []string{"Brand/New.json"},
	}
	pending := append(reviewPending(2, func(i int) string { return fmt.Sprintf("id-mismatch: [Bleed] #%d", i) }),
		store.PendingItem{Key: "A.json|/dataList/0/content", Reason: "new"}) // not an item
	return info, sum, pending
}

// Everything that ReviewItems lists is on the checklist of the issue, in the
// same order, under the heading of its section: the issue and "lct review"
// cannot drift apart because they are one list.
func TestReviewItemsMatchTheIssue(t *testing.T) {
	info, sum, pending := issueWithEveryKind()
	secs := ReviewItems(info, sum, pending)
	if len(secs) != 8 {
		t.Fatalf("%d sections, want one of each of the 8 kinds: %+v", len(secs), secs)
	}
	kinds := map[string]bool{}
	for _, s := range secs {
		if s.Kind == "" || s.Title == "" || s.Hint == "" || len(s.Items) == 0 {
			t.Errorf("section %+v lacks a kind, a title, a hint or items", s)
		}
		if kinds[s.Kind] {
			t.Errorf("kind %q is in two sections", s.Kind)
		}
		kinds[s.Kind] = true
	}
	wantKinds := []string{"needs_review", "adjustment_expired", "frozen_changed", "frozen_fallback", "new_speaker", "field_change", "new_family", "loop_error"}
	for i, s := range secs {
		if s.Kind != wantKinds[i] {
			t.Errorf("section %d is %q, want %q", i, s.Kind, wantKinds[i])
		}
	}
	// ReviewKinds, which the contract of the review skill reads, lists every
	// kind in this order, name_differs and inconsistent_new (which this
	// build has none of) before loop_error, not_translated last.
	if want := append(slices.Clone(wantKinds[:7]), "name_differs", "inconsistent_new", "loop_error", "not_translated"); !slices.Equal(ReviewKinds, want) {
		t.Errorf("ReviewKinds = %q, want %q", ReviewKinds, want)
	}

	_, parts, ok := ReviewIssue(info, sum, pending)
	if !ok || len(parts) != 1 {
		t.Fatalf("ReviewIssue made %d parts, %v", len(parts), ok)
	}
	body := parts[0]
	// Walk the body once: each heading and each item is found after the
	// one before it.
	at := 0
	find := func(what string) {
		t.Helper()
		i := strings.Index(body[at:], what)
		if i < 0 {
			t.Fatalf("%q is not in the issue after offset %d (or is out of order):\n%s", what, at, body)
		}
		at += i + len(what)
	}
	total := 0
	for _, s := range secs {
		find("### " + s.Title + "\n\n" + s.Hint + "\n")
		for _, item := range s.Items {
			find("- [ ] " + item + "\n")
			total++
		}
	}
	if got := len(itemLines(body)); got != total {
		t.Errorf("the issue has %d items, ReviewItems %d", got, total)
	}
	if !strings.Contains(body, fmt.Sprintf("本次更新有 %d 項", total)) {
		t.Errorf("the issue does not count %d items:\n%s", total, body)
	}

	// Without a plan (none, or one of another snapshot) the kinds that come
	// from the plan are not listed.
	for _, sec := range ReviewItems(info, nil, pending) {
		switch sec.Kind {
		case "adjustment_expired", "frozen_changed", "new_speaker", "field_change", "new_family":
			t.Errorf("without a plan, section %q is listed", sec.Kind)
		}
	}
}

// With several parts the items are the same ones, in the same order, as in the
// list.
func TestReviewItemsMatchTheSplitIssue(t *testing.T) {
	info := issueInfo()
	pending := reviewPending(3000, func(i int) string { return fmt.Sprintf("id-mismatch: 缺少 [Bleed] 😀 %d", i) })
	secs := ReviewItems(info, nil, pending)
	if len(secs) != 1 || secs[0].Kind != "needs_review" || len(secs[0].Items) != 3000 {
		t.Fatalf("sections = %d, want one of 3000 needs_review items", len(secs))
	}
	_, parts, _ := ReviewIssue(info, nil, pending)
	var lines []string
	for _, p := range parts {
		lines = append(lines, itemLines(p)...)
	}
	if len(lines) != 3000 {
		t.Fatalf("%d items in the issue, want 3000", len(lines))
	}
	for i, item := range secs[0].Items {
		if lines[i] != "- [ ] "+item {
			t.Fatalf("item %d of the issue is %q, of the list %q", i, lines[i], item)
		}
	}
}

// Nothing to do is an empty list that encodes as [], not null; the JSON names
// are the ones "lct review" prints.
func TestReviewItemsJSON(t *testing.T) {
	info := issueInfo()
	secs := ReviewItems(info, nil, nil)
	if secs == nil || len(secs) != 0 {
		t.Errorf("ReviewItems with nothing to do = %#v, want an empty list that is not nil", secs)
	}
	if b, err := json.Marshal(secs); err != nil || string(b) != "[]" {
		t.Errorf("json = %s, %v, want []", b, err)
	}

	pending := reviewPending(1, func(int) string { return "boom" })
	b, err := json.Marshal(ReviewItems(info, nil, pending))
	if err != nil {
		t.Fatal(err)
	}
	var got []map[string]any
	if err := json.Unmarshal(b, &got); err != nil || len(got) != 1 {
		t.Fatalf("json = %s, %v", b, err)
	}
	for _, name := range []string{"kind", "title", "hint", "items", "keys"} {
		if _, ok := got[0][name]; !ok {
			t.Errorf("json %s lacks %q", b, name)
		}
	}
	if len(got[0]) != 5 {
		t.Errorf("json %s has fields other than kind, title, hint, items, keys", b)
	}
}

// The title of the issue is the title of "lct review" too.
func TestIssueTitle(t *testing.T) {
	for tag, want := range map[string]string{
		"raw-20261105":   "審核：2026-11-05 更新",
		"raw-20261105.2": "審核：2026-11-05 更新",
		"raw-nonsense":   "審核：raw-nonsense 更新",
	} {
		if got := IssueTitle(tag); got != want {
			t.Errorf("IssueTitle(%q) = %q, want %q", tag, got, want)
		}
	}
}

// An adjustment whose unit moved when a line was inserted before it names
// both keys, the adjustment's and the unit of this snapshot, so that a person
// finds the retranslated text; one whose unit kept its key is the key alone.
func TestReviewIssueNamesTheUnitOfAnExpiredAdjustment(t *testing.T) {
	info := issueInfo()
	moved := "StoryData/S1.json|/dataList/4/content"
	unit := "StoryData/S1.json|/dataList/5/content"
	stayed := "StoryData/S1.json|/dataList/7/content"
	sum := &plan.Summary{
		RawTag: "raw-20261105", PrevRawTag: "raw-20261029",
		Adjustments: plan.AdjustmentSummary{
			Expired:      []string{moved, stayed, "StoryData/S2.json|/dataList/1/content"},
			ExpiredUnits: map[string]string{moved: unit, "StoryData/S2.json|/dataList/1/content": "StoryData/S2.json|/dataList/1/content"},
		},
	}
	secs := ReviewItems(info, sum, nil)
	if len(secs) != 1 || secs[0].Kind != KindAdjustmentExpired {
		t.Fatalf("sections = %+v, want the expired adjustments", secs)
	}
	want := []string{
		"調整 `" + moved + "`（這一版的單位：`" + unit + "`）",
		"`" + stayed + "`",
		"`StoryData/S2.json|/dataList/1/content`", // an entry that names its own key adds nothing
	}
	if !reflect.DeepEqual(secs[0].Items, want) {
		t.Errorf("items = %q, want %q", secs[0].Items, want)
	}

	_, parts, ok := ReviewIssue(info, sum, nil)
	if !ok || len(parts) != 1 {
		t.Fatalf("ReviewIssue made %d parts, %v", len(parts), ok)
	}
	for _, line := range want {
		if !strings.Contains(parts[0], "- [ ] "+line+"\n") {
			t.Errorf("the issue lacks the item %q:\n%s", line, parts[0])
		}
	}
}

// The hint of needs_review does not send a person to a command that only
// changes a local worktree: the fix has to reach main.
func TestReviewNeedsReviewHintPointsToMain(t *testing.T) {
	secs := ReviewItems(issueInfo(), nil, reviewPending(1, func(int) string { return "boom" }))
	if len(secs) != 1 || secs[0].Kind != KindNeedsReview {
		t.Fatalf("sections = %+v", secs)
	}
	hint := secs[0].Hint
	for _, want := range []string{"lct adjust add", "main", "lct requeue --to-file"} {
		if !strings.Contains(hint, want) {
			t.Errorf("the hint lacks %q: %s", want, hint)
		}
	}
	if strings.Contains(hint, "--needs-review") {
		t.Errorf("the hint still names lct requeue --needs-review, which never reaches CI: %s", hint)
	}
}

// A translation that was not run (no API key or no model) leaves the pending
// units untranslated: the issue says so, with why, once and only while units
// are still waiting.
func TestReviewIssueListsATranslationThatDidNotRun(t *testing.T) {
	const why = "缺少 ANTHROPIC_API_KEY 或 LCT_MODEL"
	info := issueInfo()
	info.Stopped, info.StopError = translate.StopNotRun, why
	waiting := []store.PendingItem{
		{Key: "A.json|/dataList/0/content", Reason: "new"},
		{Key: "A.json|/dataList/1/content", Reason: "kr-changed"},
		{Key: "A.json|/dataList/2/content", Reason: "new", Flags: []string{store.FlagNeedsReview}, Attempts: 3, LastError: "x: y"},
	}

	secs := ReviewItems(info, nil, waiting)
	// The unit that waits for a person is its own section; the two that wait
	// for a translation are counted in the one about the translation.
	if len(secs) != 2 || secs[0].Kind != KindNeedsReview || secs[1].Kind != KindNotTranslated {
		t.Fatalf("sections = %+v, want needs_review and then not_translated", secs)
	}
	sec := secs[1]
	if sec.Title != "翻譯沒有執行" || sec.Hint == "" {
		t.Errorf("section = %+v, want the title 翻譯沒有執行 and a hint", sec)
	}
	if len(sec.Items) != 1 || !strings.Contains(sec.Items[0], "`"+why+"`") || !strings.Contains(sec.Items[0], "2 個單位") {
		t.Errorf("items = %q, want the reason as a code span and 2 units waiting", sec.Items)
	}
	for _, name := range []string{"ANTHROPIC_API_KEY", "LCT_MODEL"} {
		if !strings.Contains(sec.Hint, name) {
			t.Errorf("hint %q does not name %s", sec.Hint, name)
		}
	}

	_, parts, ok := ReviewIssue(info, nil, waiting)
	if !ok || len(parts) != 1 || !strings.Contains(parts[0], "### 翻譯沒有執行") || !strings.Contains(parts[0], "- [ ] `"+why+"`（待譯 2 個單位）\n") {
		t.Errorf("the issue lacks the translation that was not run (%v):\n%v", ok, parts)
	}

	// Nothing waiting, nothing to say: a later build of the same inputs reads
	// the same translate.json, and the units have been translated since.
	for name, pending := range map[string][]store.PendingItem{
		"no pending units":                    nil,
		"only a unit that waits for a person": waiting[2:],
	} {
		for _, sec := range ReviewItems(info, nil, pending) {
			if sec.Kind == KindNotTranslated {
				t.Errorf("%s: %+v", name, sec)
			}
		}
	}
	// Any other stop is not it, and an error is the loop's own section.
	for _, stopped := range []string{"", translate.StopDone, translate.StopCost, translate.StopDeadline, translate.StopNoProgress, translate.StopError} {
		info.Stopped = stopped
		for _, sec := range ReviewItems(info, nil, waiting) {
			if sec.Kind == KindNotTranslated {
				t.Errorf("stopped %q: %+v", stopped, sec)
			}
		}
	}
	// A reason that was not recorded is said so, not left empty.
	info.Stopped, info.StopError = translate.StopNotRun, ""
	secs = ReviewItems(info, nil, waiting[:1])
	if len(secs) != 1 || !strings.Contains(secs[0].Items[0], unknownReason) {
		t.Errorf("sections = %+v, want the unknown reason", secs)
	}
}

// The sections about units carry their keys, parallel to the items; the
// others have none.
func TestReviewSectionsCarryKeys(t *testing.T) {
	info := issueInfo()
	info.Stopped = translate.StopError
	info.StopError = "boom"
	info.Report.FrozenFallback = []string{"F.json|/a"}
	sum := &plan.Summary{FrozenChanged: []string{"G.json|/b", "G.json|/c"}}
	pending := reviewPending(2, func(int) string { return "boom" })
	for _, sec := range ReviewItems(info, sum, pending) {
		switch sec.Kind {
		case KindNeedsReview:
			if want := []string{pending[0].Key, pending[1].Key}; !slices.Equal(sec.Keys, want) {
				t.Errorf("needs_review Keys = %v, want %v", sec.Keys, want)
			}
		case KindFrozenChanged:
			if !slices.Equal(sec.Keys, sum.FrozenChanged) {
				t.Errorf("frozen_changed Keys = %v", sec.Keys)
			}
		case KindFrozenFallback:
			if !slices.Equal(sec.Keys, info.Report.FrozenFallback) {
				t.Errorf("frozen_fallback Keys = %v", sec.Keys)
			}
		case KindLoopError:
			if sec.Keys != nil {
				t.Errorf("loop_error has Keys %v", sec.Keys)
			}
			if b, _ := json.Marshal(sec); strings.Contains(string(b), `"keys"`) {
				t.Errorf("loop_error json has keys: %s", b)
			}
		}
		if sec.Keys != nil && len(sec.Keys) != len(sec.Items) {
			t.Errorf("%s: %d keys for %d items", sec.Kind, len(sec.Keys), len(sec.Items))
		}
	}
}

func sectionOfKind(secs []ReviewSection, kind string) *ReviewSection {
	for i := range secs {
		if secs[i].Kind == kind {
			return &secs[i]
		}
	}
	return nil
}

func TestReviewKindsOrder(t *testing.T) {
	want := []string{"needs_review", "adjustment_expired", "frozen_changed", "frozen_fallback", "new_speaker",
		"field_change", "new_family", "name_differs", "inconsistent_new", "loop_error", "not_translated"}
	if !slices.Equal(ReviewKinds, want) {
		t.Errorf("ReviewKinds = %v, want %v", ReviewKinds, want)
	}
}

func TestReviewIssueNameDiffers(t *testing.T) {
	info := issueInfo()
	var units []NameDiffUnit
	for i := range 7 {
		units = append(units, NameDiffUnit{Key: fmt.Sprintf("S.json|/dataList/%d/name", i), ZH: "晚霞之劍", Kind: NameDiffField})
	}
	units[1].Kind, units[1].ZH = NameDiffMention, "長譯文"
	info.Report.NameDiffs = []NameDiff{
		{KR: "노을빛 검", Existing: []existing.Rendering{{ZH: "霞光劍刃", Units: 12}, {ZH: "晚霞", Units: 1}}, Units: units},
		{KR: "작은 검", Existing: []existing.Rendering{{ZH: "小劍", Units: 3}}, Units: units[:2]},
	}
	info.Report.TermDiffs = []TermDiff{
		{KR: "도발치", ZH: "嘲諷值", Existing: "挑釁值", OldUnits: 95, OldWith: 0, Keys: []string{"K1", "K2"}},
		{KR: "무명", ZH: "無名", OldUnits: 4, OldWith: 1},
	}
	secs := ReviewItems(info, nil, nil)
	sec := sectionOfKind(secs, KindNameDiffers)
	if sec == nil {
		t.Fatalf("no name_differs section in %+v", secs)
	}
	if sec.Title != "與既有譯名不同的新譯" || !strings.Contains(sec.Hint, "`glossary/terms.yaml`") ||
		!strings.Contains(sec.Hint, "`rules/rewrite/`") || !strings.Contains(sec.Hint, "`rules/declined.yaml`") || !strings.Contains(sec.Hint, "`lct accept`") {
		t.Errorf("title/hint = %q / %q", sec.Title, sec.Hint)
	}
	if len(sec.Items) != 4 {
		t.Fatalf("items = %q", sec.Items)
	}
	if !strings.HasPrefix(sec.Items[0], "`name/노을빛 검`：既有 `霞光劍刃`（12）、`晚霞`（1）；這次 7 個單位不同：`S.json|/dataList/0/name`（`晚霞之劍`）、`S.json|/dataList/1/name`（提及）") ||
		!strings.HasSuffix(sec.Items[0], "…等") {
		t.Errorf("item 0 = %q", sec.Items[0])
	}
	if strings.Contains(sec.Items[0], "/dataList/5/name") {
		t.Errorf("item 0 lists more than 5 units: %q", sec.Items[0])
	}
	if strings.HasSuffix(sec.Items[1], "等") {
		t.Errorf("item 1 has 2 units but ends with 等: %q", sec.Items[1])
	}
	if want := "`term-new/도발치`：暫定詞 → `嘲諷值`（未生效），舊譯 95 個單位、含暫定譯法 0 個；既有譯名 `挑釁值`；這次用了暫定譯法：`K1`、`K2`"; sec.Items[2] != want {
		t.Errorf("item 2 = %q, want %q", sec.Items[2], want)
	}
	if want := "`term-new/무명`：暫定詞 → `無名`（未生效），舊譯 4 個單位、含暫定譯法 1 個"; sec.Items[3] != want {
		t.Errorf("item 3 = %q, want %q", sec.Items[3], want)
	}
}

func TestReviewIssueInconsistentNew(t *testing.T) {
	info := issueInfo()
	info.Report.NameDiffs = []NameDiff{{KR: "검", Existing: []existing.Rendering{{ZH: "劍", Units: 2}},
		Units: []NameDiffUnit{{Key: "A|/1", ZH: "刀", Kind: NameDiffField}, {Key: "A|/2", ZH: "刃", Kind: NameDiffField}}}}
	listed := Inconsistency{Family: "f", Field: "name", KR: "검", ZH: []string{"刀", "刃"}, Keys: []string{"A|/1", "A|/2"},
		Counts: []int{1, 1}, Units: 2, New: 2, NewKeys: []string{"A|/1", "A|/2"}}
	other := Inconsistency{Family: "f", Field: "content", KR: "방패", ZH: []string{"盾", "盾牌"}, Keys: []string{"B|/1", "B|/2"},
		Counts: []int{3, 1}, Units: 4, New: 1, NewKeys: []string{"B|/2"}}
	oldOnly := Inconsistency{Family: "f", Field: "content", KR: "옛", ZH: []string{"甲", "乙"}, Keys: []string{"C|/1", "C|/2"}, Units: 2}
	info.InconsistentAll = []Inconsistency{listed, other, oldOnly}
	info.Report.Inconsistent = []Inconsistency{oldOnly}

	sec := sectionOfKind(ReviewItems(info, nil, nil), KindInconsistentNew)
	if sec == nil {
		t.Fatal("no inconsistent_new section")
	}
	if sec.Title != "同一 KR 不同譯法（含這次新譯）" || !strings.Contains(sec.Hint, "`rules-review`") || !strings.Contains(sec.Hint, "`inconsistent`") || !strings.Contains(sec.Hint, "`rules/declined.yaml`") {
		t.Errorf("title/hint = %q / %q", sec.Title, sec.Hint)
	}
	if len(sec.Items) != 1 {
		t.Fatalf("items = %q", sec.Items)
	}
	want := "`" + itemid.Inconsistent("f", "content", "방패") + "`：`방패`（族群 `f`、欄位 `content`，4 個單位，這次新譯 1 個）：`盾`（3）、`盾牌`（1）"
	if sec.Items[0] != want {
		t.Errorf("item = %q, want %q", sec.Items[0], want)
	}
}

func TestReviewIssueCapsLongKRAndZH(t *testing.T) {
	long := strings.Repeat("長", 100)
	capped := strings.Repeat("長", 60) + "…"
	info := issueInfo()
	info.Report.NameDiffs = []NameDiff{{KR: "검", Existing: []existing.Rendering{{ZH: "劍", Units: 2}},
		Units: []NameDiffUnit{{Key: "A|/1", ZH: long, Kind: NameDiffField}}}}
	info.InconsistentAll = []Inconsistency{{Family: "f", Field: "content", KR: long, ZH: []string{"甲", "乙"},
		Keys: []string{"B|/1", "B|/2"}, Units: 2, New: 1, NewKeys: []string{"B|/2"}}}
	secs := ReviewItems(info, nil, nil)
	if it := sectionOfKind(secs, KindNameDiffers).Items[0]; !strings.Contains(it, "（`"+capped+"`）") || strings.Contains(it, long) {
		t.Errorf("name_differs item not capped: %q", it)
	}
	if it := sectionOfKind(secs, KindInconsistentNew).Items[0]; !strings.Contains(it, "：`"+capped+"`（族群") || strings.Contains(it, long) {
		t.Errorf("inconsistent_new item not capped: %q", it)
	}
}

// A group with new units but no NewKeys (an older record) cannot be shown to
// be listed under a name, so it is kept.
func TestReviewIssueInconsistentNewKeepsAGroupWithoutNewKeys(t *testing.T) {
	info := issueInfo()
	info.Report.NameDiffs = []NameDiff{{KR: "검", Existing: []existing.Rendering{{ZH: "劍", Units: 1}},
		Units: []NameDiffUnit{{Key: "A|/1", ZH: "刀", Kind: NameDiffField}}}}
	info.InconsistentAll = []Inconsistency{{Family: "f", Field: "name", KR: "검", ZH: []string{"刀", "劍"},
		Keys: []string{"A|/1", "A|/2"}, Units: 2, New: 1}}
	sec := sectionOfKind(ReviewItems(info, nil, nil), KindInconsistentNew)
	if sec == nil || len(sec.Items) != 1 {
		t.Errorf("section = %+v, want one item", sec)
	}
}
