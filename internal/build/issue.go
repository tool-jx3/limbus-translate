package build

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/tool-jx3/limbus-translate/internal/existing"
	"github.com/tool-jx3/limbus-translate/internal/itemid"
	"github.com/tool-jx3/limbus-translate/internal/plan"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/translate"
)

const (
	// IssuePartLimit is the most characters one part of the review issue
	// holds, counted in UTF-16 code units: GitHub refuses an issue body or a
	// comment of more than 65,536 characters, and UTF-16 is the safe count
	// for Han characters and emoji.
	IssuePartLimit = 60000
	// maxErrorRunes is how long the reason of an item may be: an error of the
	// check or of the translation loop.
	maxErrorRunes = 400
	// truncated ends an item that was cut to fit a part.
	truncated = "…（內容過長，已截斷）"
	// unknownReason stands for a reason that was not recorded.
	unknownReason = "（沒有記錄原因）"
)

// The kinds of ReviewSection, in the order ReviewItems lists them.
const (
	// KindNeedsReview is the units that ran out of attempts.
	KindNeedsReview = "needs_review"
	// KindAdjustmentExpired is the adjustments that are no longer in effect.
	KindAdjustmentExpired = "adjustment_expired"
	// KindFrozenChanged is the frozen units whose KR text changed.
	KindFrozenChanged = "frozen_changed"
	// KindFrozenFallback is the frozen units whose translation broke the
	// structure of KR.
	KindFrozenFallback = "frozen_fallback"
	// KindNewSpeaker is the speakers the guides do not know.
	KindNewSpeaker = "new_speaker"
	// KindFieldChange is the changes to the field rules.
	KindFieldChange = "field_change"
	// KindNewFamily is the new files in the catch-all family.
	KindNewFamily = "new_family"
	// KindNameDiffers is the new translations that differ from the existing
	// names.
	KindNameDiffers = "name_differs"
	// KindInconsistentNew is the groups of one KR text with different
	// renderings that hold units translated in this update.
	KindInconsistentNew = "inconsistent_new"
	// KindLoopError is the translation loop that stopped on an error.
	KindLoopError = "loop_error"
	// KindNotTranslated is the translation that was not run, with units
	// still waiting.
	KindNotTranslated = "not_translated"
)

// ReviewKinds lists the kinds of ReviewSection in the order ReviewItems lists
// them. A kind added here must be handled by the review skill too: its
// contract test reads this list.
var ReviewKinds = []string{
	KindNeedsReview, KindAdjustmentExpired, KindFrozenChanged, KindFrozenFallback,
	KindNewSpeaker, KindFieldChange, KindNewFamily, KindNameDiffers, KindInconsistentNew,
	KindLoopError, KindNotTranslated,
}

// ReviewSection is one kind of thing that needs a person: a heading, what to
// do about it, and an item for each case. The review issue and "lct review"
// are both made of these.
type ReviewSection struct {
	// Kind names the kind of case, one of the Kind* constants.
	Kind string `json:"kind"`
	// Title is the heading, Hint what to do about the cases.
	Title string `json:"title"`
	Hint  string `json:"hint"`
	// Items are the cases, each the text of a line of the checklist.
	Items []string `json:"items"`
	// Keys is parallel to Items: the key of the unit an item is about, ""
	// when it is about none. Only the kinds about units fill it.
	Keys []string `json:"keys,omitempty"`
}

// ReviewItems lists what needs a person after a build (spec 14.2): the units
// that ran out of attempts, the adjustments that expired, the frozen files
// whose KR changed or whose translation broke the structure of KR, the
// speakers and the files the rules do not know, the field rules that changed,
// a translation that stopped on an error, and a translation that was not run
// while units wait for it. A kind with no case is left
// out, so the list is empty (not nil) when nothing needs a person. sum is the
// plan of this snapshot (nil when there is none) and pending the queue after
// the build; info is the build record.
func ReviewItems(info Info, sum *plan.Summary, pending []store.PendingItem) []ReviewSection {
	secs := []ReviewSection{}
	add := func(kind, title, hint string, items []string, keys ...string) {
		if len(items) > 0 {
			secs = append(secs, ReviewSection{Kind: kind, Title: title, Hint: hint, Items: items, Keys: keys})
		}
	}

	var stuck, stuckKeys []string
	for _, it := range pending {
		if slices.Contains(it.Flags, store.FlagNeedsReview) {
			stuck = append(stuck, code(it.Key)+"："+reason(it.LastError))
			stuckKeys = append(stuckKeys, it.Key)
		}
	}
	add(KindNeedsReview, "翻譯失敗、暫用英文的單位（needs_review）",
		"看原因：用 `lct adjust add` 寫入人工調整，或在 `main` 改規則、用 `lct requeue --to-file …` 指名這些 key 加一份重翻要求檔。", stuck, stuckKeys...)

	if sum != nil {
		add(KindAdjustmentExpired, "人工調整已過期",
			"KR 改了，該單位已依規則重翻（或這筆調整沒有被採用）；確認新譯文，沒問題就把舊調整從 `rules/adjustments/` 刪掉。",
			expiredItems(sum.Adjustments))
		add(KindFrozenChanged, "凍結檔的 KR 有變",
			"歌詞等原文被修改，仍沿用舊譯文；決定是否解除凍結（`lct.yaml` 的 `frozen`）並重翻。",
			codes(sum.FrozenChanged), sum.FrozenChanged...)
	}
	add(KindFrozenFallback, "凍結檔的譯文破壞原文結構",
		"凍結檔不重新翻譯，這些單位暫用英文；解除凍結（`lct.yaml` 的 `frozen`）後用 `lct requeue --key` 重翻。",
		codes(info.Report.FrozenFallback), info.Report.FrozenFallback...)
	if sum != nil {
		add(KindNewSpeaker, "未對應的說話者",
			"新角色或新 NPC 的 `model`；補上 `guides/characters/` 的角色檔。",
			codes(sum.NewSpeakers))
		var changes []string
		for _, c := range sum.FieldChanges {
			changes = append(changes, code(c.String()))
		}
		add(KindFieldChange, "欄位規則變化",
			"遊戲新增或移除了欄位；確認後用 `lct fields learn` 更新 `rules/fields.yaml`。", changes)
		add(KindNewFamily, "預設族群的新檔",
			"這些新檔沒有對應的檔案族群；到 `guides/routing.yaml` 加上族群。",
			codes(sum.NewFilesDefaultFamily))
	}
	add(KindNameDiffers, "與既有譯名不同的新譯",
		"既有譯文對這些名稱有固定寫法，這次新譯不同（名稱欄位，或說明、劇情裡提到卻沒用既有譯法）。指的是同一個東西：在 `glossary/terms.yaml` 加條目（以既有譯名為準），並寫 `rules/rewrite/` 的改寫檔，只改這次新譯；依語境本來就該不同：把項目開頭的 ID 寫進 `rules/declined.yaml`（審核頁或 `lct accept`）。",
		nameDiffItems(info.Report))
	add(KindInconsistentNew, "同一 KR 不同譯法（含這次新譯）",
		"同一段 KR 在同族群、同欄位有不同譯法，其中有這次新翻的單位。要統一：照 `rules-review` 的 `inconsistent` 處理（詞彙或指引，加重翻要求檔或改寫檔）；依 JP/EN 或語境本來就該不同：把項目開頭的 ID 寫進 `rules/declined.yaml`。",
		inconsistentNewItems(info))
	if info.Stopped == translate.StopError {
		add(KindLoopError, "翻譯迴圈因錯誤中斷",
			"看 Actions 的執行紀錄；修正後重跑，還沒翻的單位仍在待譯清單裡。",
			[]string{reason(info.StopError)})
	}
	// The record of a translation that was not run is read again by every
	// build of the same inputs, so it only counts while something waits.
	if n := waitingForTranslation(pending); info.Stopped == translate.StopNotRun && n > 0 {
		add(KindNotTranslated, "翻譯沒有執行",
			"沒有設定 `ANTHROPIC_API_KEY`（secret）或 `LCT_MODEL`（variable）時不會翻譯，待譯的單位暫用英文或原文；補上後到 Actions 手動執行 `rebuild`，待譯清單還在，會接著翻。",
			[]string{fmt.Sprintf("%s（待譯 %d 個單位）", reason(info.StopError), n)})
	}
	return secs
}

// maxItemUnits is how many units, or renderings, an item of name_differs and
// inconsistent_new lists.
const maxItemUnits = 5

// maxItemRunes is how long a KR text or a translation in such an item is.
const maxItemRunes = 60

// nameDiffItems is the item of each existing name that was rendered otherwise
// and of each provisional term that is not in force.
func nameDiffItems(rep Report) []string {
	var items []string
	for _, d := range rep.NameDiffs {
		var olds []string
		for _, r := range d.Existing {
			olds = append(olds, fmt.Sprintf("%s（%d）", code(r.ZH), r.Units))
		}
		var units []string
		for _, u := range d.Units[:min(len(d.Units), maxItemUnits)] {
			if u.Kind == NameDiffMention {
				units = append(units, code(u.Key)+"（提及）")
			} else {
				units = append(units, code(u.Key)+"（"+code(capRunes(oneLine(u.ZH), maxItemRunes))+"）")
			}
		}
		item := fmt.Sprintf("%s：既有 %s；這次 %d 個單位不同：%s",
			code(existing.SuggestionID(d.KR)), strings.Join(olds, "、"), len(d.Units), strings.Join(units, "、"))
		if len(d.Units) > maxItemUnits {
			item += "…等"
		}
		items = append(items, item)
	}
	for _, d := range rep.TermDiffs {
		item := fmt.Sprintf("%s：暫定詞 → %s（未生效），舊譯 %d 個單位、含暫定譯法 %d 個",
			code("term-new/"+d.KR), code(d.ZH), d.OldUnits, d.OldWith)
		if d.Existing != "" {
			item += "；既有譯名 " + code(d.Existing)
		}
		if len(d.Keys) > 0 {
			item += "；這次用了暫定譯法：" + strings.Join(codes(d.Keys[:min(len(d.Keys), maxItemUnits)]), "、")
			if len(d.Keys) > maxItemUnits {
				item += "…等"
			}
		}
		items = append(items, item)
	}
	return items
}

// inconsistentNewItems is the item of each group of one KR text with
// different renderings that holds units of this update, except a group whose
// new units are all listed under an existing name already.
func inconsistentNewItems(info Info) []string {
	listed := map[string]bool{}
	for _, d := range info.Report.NameDiffs {
		for _, u := range d.Units {
			listed[u.Key] = true
		}
	}
	var items []string
	for _, g := range inconsistentGroups(info) {
		if g.New == 0 || (len(g.NewKeys) > 0 && !slices.ContainsFunc(g.NewKeys, func(k string) bool { return !listed[k] })) {
			continue
		}
		var zhs []string
		for i, zh := range g.ZH[:min(len(g.ZH), maxItemUnits)] {
			if i < len(g.Counts) {
				zhs = append(zhs, fmt.Sprintf("%s（%d）", code(zh), g.Counts[i]))
			} else {
				zhs = append(zhs, code(zh))
			}
		}
		items = append(items, fmt.Sprintf("%s：%s（族群 %s、欄位 %s，%d 個單位，這次新譯 %d 個）：%s",
			code(itemid.Inconsistent(g.Family, g.Field, g.KR)), code(capRunes(oneLine(g.KR), maxItemRunes)), code(g.Family), code(g.Field), g.Units, g.New, strings.Join(zhs, "、")))
	}
	return items
}

// inconsistentGroups are the groups of one KR text with different renderings
// the build record holds: all that it kept, or, in a record of an older lct,
// the ones its report lists.
func inconsistentGroups(info Info) []Inconsistency {
	if info.InconsistentAll != nil {
		return info.InconsistentAll
	}
	return info.Report.Inconsistent
}

// waitingForTranslation counts the pending units a translation can still
// take: those that do not wait for a person.
func waitingForTranslation(pending []store.PendingItem) int {
	n := 0
	for _, it := range pending {
		if !slices.Contains(it.Flags, store.FlagNeedsReview) {
			n++
		}
	}
	return n
}

// expiredItems is the item of each adjustment that is not in effect. When the
// unit the adjustment is about has another key in this snapshot (a line was
// inserted before it), the item names that key too: the retranslated text is
// there, not at the key of the adjustment.
func expiredItems(a plan.AdjustmentSummary) []string {
	items := make([]string, len(a.Expired))
	for i, key := range a.Expired {
		items[i] = code(key)
		if unit, ok := a.ExpiredUnits[key]; ok && unit != "" && unit != key {
			items[i] = "調整 " + code(key) + "（這一版的單位：" + code(unit) + "）"
		}
	}
	return items
}

// reason is the text of an item that says why: an error, on one line and cut
// to maxErrorRunes, as a code span so that the tags and brackets of the games
// text in it are not read as Markdown or HTML. An error that was not recorded
// is said so.
func reason(err string) string {
	err = capRunes(oneLine(err), maxErrorRunes)
	if err == "" {
		return unknownReason
	}
	return code(err)
}

// ReviewIssue is the review issue of a build (spec 14.1): a checklist of
// everything that needs a person, and nothing else. ok is false when nothing
// does, and then there is no issue. title names the day of the snapshot; parts
// holds the text of the issue, then of the comments to add to it, in order.
// No part is longer than IssuePartLimit UTF-16 code units, and a part ends
// between two items; an item that alone is longer than a part is cut and says
// so. When there are several parts each starts with the number of it, and the
// first names the place of the full report.
func ReviewIssue(info Info, sum *plan.Summary, pending []store.PendingItem) (title string, parts []string, ok bool) {
	secs := ReviewItems(info, sum, pending)
	n := 0
	for _, s := range secs {
		n += len(s.Items)
	}
	if n == 0 {
		return "", nil, false
	}
	intro := fmt.Sprintf("本次更新有 %d 項需要你處理，處理完請勾掉。完整報表：`%s`（`tw` 分支）。", n, ReportRel(info.RawTag))
	return IssueTitle(info.RawTag), splitParts(intro, secs, n), true
}

var rawDayRe = regexp.MustCompile(`^raw-(\d{8})(?:\.\d+)?$`)

// IssueTitle is "審核：YYYY-MM-DD 更新", the date being the day of the raw
// tag; a tag that holds no date is named as it is.
func IssueTitle(rawTag string) string {
	if m := rawDayRe.FindStringSubmatch(rawTag); m != nil {
		if day, err := time.Parse("20060102", m[1]); err == nil {
			return "審核：" + day.Format("2006-01-02") + " 更新"
		}
	}
	return "審核：" + rawTag + " 更新"
}

// partHeader says which of m parts the i-th is.
func partHeader(i, m int) string { return fmt.Sprintf("（第 %d／%d 段）", i, m) }

// splitParts cuts the checklist of secs, which holds n items, into parts of at
// most IssuePartLimit UTF-16 code units, between two items. The first part
// starts with intro. The numbering of the parts is put on when there are
// several; room is kept for the longest header there could be.
func splitParts(intro string, secs []ReviewSection, n int) []string {
	p := &packer{intro: intro, budget: IssuePartLimit - utf16Len(partHeader(n, n)+"\n\n"), headed: map[int]bool{}}
	p.begin()
	for si, sec := range secs {
		for _, item := range sec.Items {
			p.add(si, sec, item)
		}
	}
	p.flush()
	if len(p.parts) > 1 {
		for i, part := range p.parts {
			p.parts[i] = partHeader(i+1, len(p.parts)) + "\n\n" + part
		}
	}
	return p.parts
}

// packer fills the parts of the issue one item at a time.
type packer struct {
	intro  string
	budget int // code units a part may hold before its header
	parts  []string

	cur     strings.Builder
	used    int
	section int          // section whose heading the current part wrote last, -1 for none
	items   int          // items in the current part
	headed  map[int]bool // sections whose full heading was written in some part
}

func (p *packer) write(s string) {
	p.cur.WriteString(s)
	p.used += utf16Len(s)
}

// begin starts a part.
func (p *packer) begin() {
	p.cur.Reset()
	p.used, p.section, p.items = 0, -1, 0
	if len(p.parts) == 0 {
		p.write(p.intro + "\n\n")
	}
}

// flush ends the current part.
func (p *packer) flush() {
	p.parts = append(p.parts, p.cur.String())
}

// add puts item of section sec (the si-th) on the checklist: in the current
// part when it fits, else at the start of the next. A section that goes on in
// another part gets its heading again, marked as a continuation.
func (p *packer) add(si int, sec ReviewSection, item string) {
	const bullet = "- [ ] "
	for {
		heading := ""
		if p.section != si {
			if p.headed[si] {
				heading = "### " + sec.Title + "（續）\n\n"
			} else {
				heading = "### " + sec.Title + "\n\n" + sec.Hint + "\n\n"
			}
			if p.items > 0 {
				heading = "\n" + heading // a blank line after the section before
			}
		}
		line := bullet + item + "\n"
		if p.used+utf16Len(heading)+utf16Len(line) <= p.budget {
			p.put(si, heading, line)
			return
		}
		if p.items > 0 {
			p.flush()
			p.begin()
			continue
		}
		// Alone in a part and still too long: cut it.
		room := p.budget - p.used - utf16Len(heading) - utf16Len(bullet) - utf16Len(truncated) - 1
		p.put(si, heading, bullet+cutUTF16(item, max(room, 0))+truncated+"\n")
		return
	}
}

func (p *packer) put(si int, heading, line string) {
	p.write(heading)
	p.write(line)
	p.section = si
	p.headed[si] = true
	p.items++
}

// utf16Len is the length of s in UTF-16 code units, the unit GitHub counts the
// characters of an issue in. A character outside the BMP, such as an emoji,
// is two.
func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		n += runeUTF16Len(r)
	}
	return n
}

func runeUTF16Len(r rune) int {
	if n := utf16.RuneLen(r); n > 0 {
		return n
	}
	return 1 // U+FFFD stands for what cannot be encoded
}

// cutUTF16 returns the longest prefix of s that is at most limit UTF-16 code
// units long, cut between two characters.
func cutUTF16(s string, limit int) string {
	n := 0
	for i, r := range s {
		n += runeUTF16Len(r)
		if n > limit {
			return s[:i]
		}
	}
	return s
}

// oneLine collapses every run of white space of s to one space, so that s is
// the text of a single line.
func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// capRunes cuts s to n characters at most, marking the cut.
func capRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}

// code is s as an inline code span on one line. A backtick in s is handled by
// the longer delimiter Markdown allows.
func code(s string) string {
	s = oneLine(s)
	if !strings.Contains(s, "`") {
		return "`" + s + "`"
	}
	return "`` " + s + " ``"
}

// codes is code for each of keys.
func codes(keys []string) []string {
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = code(k)
	}
	return out
}
