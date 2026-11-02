package build

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/tool-jx3/limbus-translate/internal/fields"
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
	for _, name := range []string{"kind", "title", "hint", "items"} {
		if _, ok := got[0][name]; !ok {
			t.Errorf("json %s lacks %q", b, name)
		}
	}
	if len(got[0]) != 4 {
		t.Errorf("json %s has fields other than kind, title, hint, items", b)
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
