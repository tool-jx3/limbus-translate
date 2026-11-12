package build

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/tool-jx3/limbus-translate/internal/existing"
	"github.com/tool-jx3/limbus-translate/internal/plan"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/translate"
)

// versionInfo is what Info/version.json records: the inputs of the build and
// what the translation of them cost.
type versionInfo struct {
	RawTag      string            `json:"raw_tag"`
	RawToken    string            `json:"raw_token"`
	RulesCommit string            `json:"rules_commit"`
	BuiltAt     string            `json:"built_at"`
	Upstream    map[string]string `json:"upstream,omitempty"`
	// Model and CostUSD are those of the translation run (see translation).
	Model   string  `json:"model,omitempty"`
	CostUSD float64 `json:"cost_usd,omitempty"`
}

func newVersionInfo(opt Options, tr translation) versionInfo {
	return versionInfo{
		RawTag:      opt.RawTag,
		RawToken:    opt.RawToken,
		RulesCommit: opt.RulesCommit,
		BuiltAt:     opt.BuiltAt.UTC().Truncate(time.Second).Format(time.RFC3339),
		Upstream:    opt.Upstream,
		Model:       tr.Model,
		CostUSD:     tr.CostUSD,
	}
}

// translation is the model, the cost and the end of the translation run a
// build is of. The zero value says there is none.
type translation struct {
	Model   string
	CostUSD float64
	// Stopped is why the run ended (translate.Stop*), Error what went wrong
	// when it is translate.StopError.
	Stopped, Error string
}

// readTranslation reads what lct translate left in .meta/translate.json. It
// is of this build only if it was made for the same raw tag and the same
// rules commit; a run for other inputs, or none, gives the zero translation.
func readTranslation(st *store.Store, opt Options) (translation, error) {
	var res translate.Result
	found, err := st.ReadMetaJSON(translate.MetaFile, &res)
	if err != nil {
		return translation{}, err
	}
	if !found || res.RawTag != opt.RawTag || res.RulesCommit != opt.RulesCommit {
		return translation{}, nil
	}
	return translation{Model: res.Model, CostUSD: res.CostUSD, Stopped: res.Stopped, Error: res.Error}, nil
}

// ReadPlan reads the summary of the plan (.meta/plan.json) of the tw worktree
// at twRoot, if it is the plan of the snapshot rawTag. A worktree without a
// plan, or with the plan of another snapshot, gives nil.
func ReadPlan(twRoot, rawTag string) (*plan.Summary, error) {
	return readPlan(store.Open(twRoot), rawTag)
}

func readPlan(st *store.Store, rawTag string) (*plan.Summary, error) {
	var sum plan.Summary
	found, err := st.ReadMetaJSON(plan.File, &sum)
	if err != nil {
		return nil, err
	}
	if !found || sum.RawTag != rawTag {
		return nil, nil
	}
	return &sum, nil
}

// infoName is the file of the store that holds Info.
const infoName = "build.json"

// ErrNotBuilt is returned by ReadInfo for a tw worktree that has no
// .meta/build.json: nothing has been built there.
var ErrNotBuilt = errors.New("not built")

// Info is .meta/build.json: the inputs of the build, the zips it made and what
// it did. Run writes it last; ReadInfo reads it. It has every field of the
// file, because the store refuses a file with fields the type does not know.
type Info struct {
	RawTag      string            `json:"raw_tag"`
	RawToken    string            `json:"raw_token"`
	RulesCommit string            `json:"rules_commit"`
	BuiltAt     string            `json:"built_at"`
	Upstream    map[string]string `json:"upstream,omitempty"`
	// Model and CostUSD are the model and the cost (in US dollars) of the
	// translation run of these inputs, from .meta/translate.json; absent when
	// that file is of other inputs or there is none.
	Model   string  `json:"model,omitempty"`
	CostUSD float64 `json:"cost_usd,omitempty"`
	// Stopped is why that translation run ended (translate.Stop*), and
	// StopError what went wrong when it ended on an error; both are empty
	// when there is no run of these inputs.
	Stopped   string `json:"stopped,omitempty"`
	StopError string `json:"stop_error,omitempty"`
	// NotesSeen is the number of translator notes (.meta/notes.jsonl) the
	// build counted: Report.Notes holds those after the number the previous
	// build saw, so a note is reported once.
	NotesSeen int `json:"notes_seen"`
	// Zips is the sha256 (lowercase hex) of each zip the build wrote to
	// OutDir, by file name. publish checks the zips it releases against it.
	Zips   map[string]string `json:"zips"`
	Report Report            `json:"report"`
	// InconsistentAll are the groups of Report.Inconsistent and the ones
	// after them, in the same order: the biggest 500, and after them every
	// other group with units of this update (New > 0); never nil. The report
	// shows twenty, while lct suggest leaves out the groups that are declined
	// or in a rules PR, and takes the next ones from here; the report and the
	// review issue list every group with new units from here. A build.json of
	// an older lct has no such list.
	InconsistentAll []Inconsistency `json:"inconsistent_all"`
}

// ReadInfo reads .meta/build.json of the tw worktree at twRoot. A worktree
// without one is ErrNotBuilt; one that cannot be read or decoded is another
// error.
func ReadInfo(twRoot string) (Info, error) {
	var info Info
	found, err := store.Open(twRoot).ReadMetaJSON(infoName, &info)
	if err != nil {
		return Info{}, err
	}
	if !found {
		return Info{}, fmt.Errorf("build: %w: %s has no .meta/%s; run lct build first", ErrNotBuilt, twRoot, infoName)
	}
	return info, nil
}

// encodeJSON renders v like the store does: 2-space indent, no HTML escaping,
// one trailing newline, map keys sorted.
func encodeJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("build: %w", err)
	}
	return buf.Bytes(), nil
}

// ReportPath is where Run writes the report of the build.
func ReportPath(opt Options) string {
	return filepath.Join(opt.TWRoot, filepath.FromSlash(ReportRel(opt.RawTag)))
}

// ReportRel is the place of the report of the snapshot rawTag in the tw
// branch, with forward slashes.
func ReportRel(rawTag string) string { return ".meta/reports/" + rawTag + ".md" }

// newInfo is the build record of a build: the inputs v that Info/version.json
// says, the end of the translation, the checksums of the zips, the report,
// all the groups of different translations and the number of notes seen.
func newInfo(v versionInfo, tr translation, zips map[string]string, rep *Report, inconsistent []Inconsistency, notesSeen int) Info {
	return Info{
		RawTag: v.RawTag, RawToken: v.RawToken, RulesCommit: v.RulesCommit, BuiltAt: v.BuiltAt, Upstream: v.Upstream,
		Model: v.Model, CostUSD: v.CostUSD, Stopped: tr.Stopped, StopError: tr.Error, NotesSeen: notesSeen,
		Zips: zips, Report: *rep, InconsistentAll: inconsistent,
	}
}

// writeMeta writes the build record and the report to the store. sum is the
// plan of the snapshot, nil if there is none, and pending the queue after the
// build.
func writeMeta(st *store.Store, opt Options, info Info, sum *plan.Summary, pending []store.PendingItem) error {
	if err := st.WriteMetaJSON(infoName, info); err != nil {
		return err
	}
	return st.WriteMetaFile("reports/"+opt.RawTag+".md", Markdown(opt, info, sum, pending))
}

// SummaryLimit is the most bytes of the report that SummaryMarkdown keeps.
const SummaryLimit = 1_000_000

// SummaryMarkdown is the report for the summary page of a workflow run: report
// itself, or when it is longer than SummaryLimit bytes its first lines and a
// notice of where the whole report is (reportRel, in the tw branch).
func SummaryMarkdown(report []byte, reportRel string) []byte {
	if len(report) <= SummaryLimit {
		return report
	}
	notice := fmt.Sprintf("\n（報表超過 1,000,000 位元組，已截斷；完整報表在 `tw` 分支的 `%s`。）\n", reportRel)
	cut := report[:SummaryLimit-len(notice)]
	if i := bytes.LastIndexByte(cut, '\n'); i >= 0 {
		cut = cut[:i+1] // at the end of a line, which is the end of a character too
	} else {
		// One line without an end: at least not inside a character.
		for n := 0; n < utf8.UTFMax && len(cut) > 0; n++ {
			if r, size := utf8.DecodeLastRune(cut); r != utf8.RuneError || size != 1 {
				break
			}
			cut = cut[:len(cut)-1]
		}
	}
	return append(append([]byte{}, cut...), notice...)
}

// maxListedFiles is how many files of a kind the report names.
const maxListedFiles = 30

// Markdown renders the report of a build (spec 14): the part for a person,
// then the information. info is the build record, sum the plan of the
// snapshot (nil when there is none) and pending the queue after the build;
// opt says where the repo is, for the link of the compare page of the KR text.
//
// This is the report Run archives in the tw branch, so it must be the same for
// the same inputs: it does not hold where tw stood before the build
// (Options.PrevTWCommit), which moves with every publish. A report that
// followed it would change the branch on every build, and a Release would
// follow for a pack that did not change. JobSummary has that link.
func Markdown(opt Options, info Info, sum *plan.Summary, pending []store.PendingItem) []byte {
	return markdown(opt, info, sum, pending, false)
}

// JobSummary is the report for the summary page of a workflow run (cut at
// SummaryLimit): the archived report and, in the information about the
// translations, the compare page of tw from where it stood before the build.
// It is not committed anywhere.
func JobSummary(opt Options, info Info, sum *plan.Summary, pending []store.PendingItem) []byte {
	return SummaryMarkdown(markdown(opt, info, sum, pending, true), ReportRel(info.RawTag))
}

// markdown renders the report; twLink says whether it links the compare page
// of tw.
func markdown(opt Options, info Info, sum *plan.Summary, pending []store.PendingItem, twLink bool) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "# Build report %s\n", info.RawTag)

	b.WriteString("\n## 需要你處理\n\n")
	secs := ReviewItems(info, sum, pending)
	if len(secs) == 0 {
		b.WriteString("沒有需要處理的項目。\n")
	}
	for i, sec := range secs {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "### %s：%d 項\n\n%s\n\n", sec.Title, len(sec.Items), sec.Hint)
		for _, item := range sec.Items {
			fmt.Fprintf(&b, "- %s\n", item)
		}
	}

	b.WriteString("\n## 資訊\n")
	infoInputs(&b, info)
	infoSource(&b, opt, info, sum)
	infoTranslations(&b, opt, info, sum, twLink)
	infoAutomatic(&b, sum)
	infoUnits(&b, info.Report)
	infoMisaligned(&b, info.Report)
	infoWarnings(&b, info.Report)
	infoNameDiffs(&b, info.Report)
	infoInconsistent(&b, info)
	infoNotes(&b, info.Report)
	infoGlyphs(&b, info.Report)
	return []byte(b.String())
}

// stopReasons says in words why a translation run ended.
var stopReasons = map[string]string{
	translate.StopDone:       "沒有可領的待譯單位",
	translate.StopCost:       "花費達上限",
	translate.StopDeadline:   "接近時間上限",
	translate.StopNoProgress: "連續兩輪待譯單位沒有減少",
	translate.StopError:      "發生錯誤",
	translate.StopNotRun:     "沒有執行",
}

// infoInputs lists what the build and the translation were made from.
func infoInputs(b *strings.Builder, info Info) {
	b.WriteString("\n### 這次的輸入\n\n")
	fmt.Fprintf(b, "- raw tag：`%s`，token：`%s`\n", info.RawTag, info.RawToken)
	fmt.Fprintf(b, "- 規則 commit：`%s`\n- 建置時間：%s\n", info.RulesCommit, info.BuiltAt)
	if info.Model != "" {
		fmt.Fprintf(b, "- 翻譯模型：%s\n", info.Model)
	}
	if info.Model != "" || info.CostUSD > 0 {
		fmt.Fprintf(b, "- 翻譯花費：$%.2f\n", info.CostUSD)
	}
	if info.Stopped != "" {
		fmt.Fprintf(b, "- 翻譯迴圈停止原因：%s（%s）\n", info.Stopped, stopReasons[info.Stopped])
		if info.StopError != "" {
			// A translation that was not run has a reason, not an error.
			label := "錯誤"
			if info.Stopped == translate.StopNotRun {
				label = "原因"
			}
			fmt.Fprintf(b, "  - %s：%s\n", label, reason(info.StopError))
		}
	}
	for _, name := range slices.Sorted(maps.Keys(info.Upstream)) {
		fmt.Fprintf(b, "- 上游 %s：`%s`\n", name, info.Upstream[name])
	}
}

// compareURL is the compare page of two refs of the repo, or "" without a
// repo URL or a start.
func compareURL(repo, from, to string) string {
	if repo == "" || from == "" {
		return ""
	}
	return strings.TrimRight(repo, "/") + "/compare/" + from + "..." + to
}

// infoSource tells how the KR text changed since the previous snapshot.
func infoSource(b *strings.Builder, opt Options, info Info, sum *plan.Summary) {
	b.WriteString("\n### 原文變化\n\n")
	switch {
	case sum == nil:
		b.WriteString("沒有這個原文版本的 plan 紀錄（`.meta/plan.json`），不知道原文有什麼變化。\n")
		return
	case sum.PrevRawTag == sum.RawTag:
		b.WriteString("沒有更舊的狀態可以比較（第一次建置）。\n")
		return
	}
	c := sum.Changes
	fmt.Fprintf(b, "- 單位：新增 %d、修改 %d、刪除 %d；不變 %d（其中 %d 個只是搬到別的 key）\n", c.New, c.Changed, c.Deleted, c.Unchanged, c.Moved)
	for _, l := range []struct {
		label string
		files []string
	}{
		{"新增的檔", c.FilesAdded},
		{"刪除的檔", c.FilesDeleted},
		{"有修改的檔", c.FilesChanged},
		{"中間插入或刪行的檔（key 位移）", c.FilesShifted},
	} {
		if len(l.files) == 0 {
			continue
		}
		fmt.Fprintf(b, "- %s（%d）：", l.label, len(l.files))
		shown := l.files[:min(len(l.files), maxListedFiles)]
		b.WriteString(strings.Join(codes(shown), "、"))
		if rest := len(l.files) - len(shown); rest > 0 {
			fmt.Fprintf(b, "，另有 %d 個", rest)
		}
		b.WriteString("\n")
	}
	if u := compareURL(opt.RepoURL, sum.PrevRawTag, info.RawTag); u != "" {
		fmt.Fprintf(b, "- KR 的差異：%s\n", u)
	}
}

// infoTranslations tells where the translations come from and what changed.
func infoTranslations(b *strings.Builder, opt Options, info Info, sum *plan.Summary, twLink bool) {
	rep := info.Report
	b.WriteString("\n### 譯文變化\n\n")
	if len(rep.Origins) == 0 {
		b.WriteString("沒有使用中的譯文紀錄。\n")
	} else {
		b.WriteString("| Origin | Units |\n|---|---|\n")
		for _, k := range slices.Sorted(maps.Keys(rep.Origins)) {
			fmt.Fprintf(b, "| %s | %d |\n", k, rep.Origins[k])
		}
	}
	fmt.Fprintf(b, "\n- 這次新翻（origin claude，Since 為 `%s`）：%d 個單位\n", info.RawTag, rep.NewTranslated)
	if sum != nil {
		if sum.PrevRawTag != sum.RawTag {
			// A frozen unit whose KR changed keeps its translation.
			fmt.Fprintf(b, "- 因 KR 變更需要重翻：%d 個單位\n", max(0, sum.Changes.Changed-len(sum.FrozenChanged)))
		}
		fmt.Fprintf(b, "- 沿用相同文字的譯文：%d 個單位；不需翻譯而複製原文：%d 個單位\n", sum.Reused, sum.Copy)
	}
	if u := compareURL(opt.RepoURL, opt.PrevTWCommit, "tw"); twLink && u != "" {
		fmt.Fprintf(b, "- 譯文（`tw`）的差異：%s\n", u)
	}
}

// infoAutomatic lists what the plan did by itself.
func infoAutomatic(b *strings.Builder, sum *plan.Summary) {
	if sum == nil {
		return
	}
	b.WriteString("\n### 自動處理的事\n\n")
	a := sum.Adjustments
	fmt.Fprintf(b, "- 人工調整：%d 個照原 key 套用，%d 個因 key 已移動而套用到新 key\n", a.Applied, len(a.Moved))
	for _, m := range a.Moved {
		fmt.Fprintf(b, "  - %s → %s\n", code(m.From), code(m.To))
	}
	if sum.PrevRawTag != sum.RawTag {
		fmt.Fprintf(b, "- id 重新編號或位置改變後，沿用原本的譯文：%d 個單位\n", sum.Changes.Moved)
	}
	for _, l := range []struct {
		label string
		files []plan.Applied
	}{{"重翻要求檔", sum.RequeueApplied}, {"改寫檔", sum.RewriteApplied}} {
		for _, f := range l.files {
			fmt.Fprintf(b, "- 套用了%s `%s`：%d 個單位\n", l.label, f.ID, f.Units)
		}
	}
}

// infoUnits counts the units that are not as they should be.
func infoUnits(b *strings.Builder, rep Report) {
	b.WriteString("\n### 單位\n\n| Item | Units |\n|---|---|\n")
	for _, row := range []struct {
		label string
		n     int
	}{
		{"Pending", rep.Pending},
		{"Needs review", rep.NeedsReview},
		{"Structural fallback", rep.StructuralFallback},
		{"Frozen fallback", len(rep.FrozenFallback)},
		{"Unaligned", rep.Unaligned},
		{"Field rule override", rep.FieldRuleOverride},
		{"Stand-in shown", rep.StandIn},
	} {
		fmt.Fprintf(b, "| %s | %d |\n", row.label, row.n)
	}
	b.WriteString("\n" +
		"- Pending units have no translation made from the current KR text; their EN text, or KR, is shown.\n" +
		"- Needs review counts the used records that are flagged needs_review; a frozen fallback is not counted again here.\n" +
		"- Structural fallbacks are translations that break the structure of KR (tags, [ID]s, placeholders) and were replaced the same way.\n" +
		"- Frozen fallbacks are such translations in frozen files: they are not translated again, so each needs a person (listed above).\n" +
		"- Unaligned units are pending units with no EN text either (the order or the ids of the languages do not match).\n" +
		"- Field rule overrides are records of units the field rules do not translate; KR is shown instead.\n" +
		"- Stand-in shown counts every unit that shows EN (or KR) in place of a translation: pending, fallback and structural or frozen fallback.\n")
}

// infoMisaligned lists the files whose order or ids differ between the
// languages, so that units of KR have no EN or JP text to go with.
func infoMisaligned(b *strings.Builder, rep Report) {
	b.WriteString("\n### 跨語言未對齊的檔\n\n")
	if len(rep.Misaligned) == 0 {
		b.WriteString("None.\n")
		return
	}
	if rep.MisalignedFiles > 0 { // a build.json of an older lct does not count them
		fmt.Fprintf(b, "這些檔有 KR 單位找不到對應的 EN 或 JP 文字（順序或 id 與 KR 不一致），共 %d 個，最多列 %d 個，缺得最多的在前：\n\n", rep.MisalignedFiles, maxMisaligned)
	} else {
		fmt.Fprintf(b, "這些檔有 KR 單位找不到對應的 EN 或 JP 文字（順序或 id 與 KR 不一致），最多列 %d 個，缺得最多的在前：\n\n", maxMisaligned)
	}
	for _, f := range rep.Misaligned {
		fmt.Fprintf(b, "- %s：EN 缺 %d 個、JP 缺 %d 個\n", code(f.File), f.EN, f.JP)
	}
}

// infoWarnings lists the warnings of the checks and a few units of each.
func infoWarnings(b *strings.Builder, rep Report) {
	b.WriteString("\n### 驗證警告\n\n")
	if len(rep.Warnings) == 0 {
		b.WriteString("None.\n")
		return
	}
	b.WriteString("| Code | Units |\n|---|---|\n")
	for _, k := range slices.Sorted(maps.Keys(rep.Warnings)) {
		fmt.Fprintf(b, "| %s | %d |\n", k, rep.Warnings[k])
	}
	for _, k := range slices.Sorted(maps.Keys(rep.Warnings)) {
		samples := rep.WarningSamples[k]
		if len(samples) == 0 {
			continue
		}
		fmt.Fprintf(b, "\n%s 的前 %d 筆：\n\n", k, len(samples))
		for _, key := range samples {
			fmt.Fprintf(b, "- %s\n", code(key))
		}
	}
}

// infoNameDiffs lists the new translations that differ from the existing
// names and the provisional terms that are not in force.
func infoNameDiffs(b *strings.Builder, rep Report) {
	b.WriteString("\n### 與既有譯名不同的新譯\n\n")
	if len(rep.NameDiffs) == 0 && len(rep.TermDiffs) == 0 {
		b.WriteString("None.\n")
		return
	}
	for _, d := range rep.NameDiffs {
		var olds []string
		for _, r := range d.Existing {
			olds = append(olds, fmt.Sprintf("%s（%d）", code(r.ZH), r.Units))
		}
		fmt.Fprintf(b, "- %s（%s）：既有 %s\n", code(d.KR), code(existing.SuggestionID(d.KR)), strings.Join(olds, "、"))
		for _, u := range d.Units {
			if u.Kind == NameDiffMention {
				fmt.Fprintf(b, "  - %s：提及；譯文 %s\n", code(u.Key), code(capRunes(oneLine(u.ZH), maxNameDiffRunes)))
				continue
			}
			fmt.Fprintf(b, "  - %s：%s（名稱欄位）", code(u.Key), code(u.ZH))
			if note := oneLine(u.Note); note != "" {
				fmt.Fprintf(b, "；note：%s", capRunes(note, maxNoteRunes))
			}
			b.WriteString("\n")
		}
	}
	for _, d := range rep.TermDiffs {
		fmt.Fprintf(b, "- 暫定詞 %s → %s（%s，未生效）：舊譯 %d 個單位，含暫定譯法 %d 個", code(d.KR), code(d.ZH), code("term-new/"+d.KR), d.OldUnits, d.OldWith)
		if d.Existing != "" {
			fmt.Fprintf(b, "；既有譯名 %s", code(d.Existing))
		}
		b.WriteString("\n")
		for _, ex := range d.Examples {
			fmt.Fprintf(b, "  - 舊譯 %s：KR %s／TW %s\n", code(ex.Key), code(capRunes(oneLine(ex.KR), maxNameDiffRunes)), code(capRunes(oneLine(ex.ZH), maxNameDiffRunes)))
		}
		if len(d.Keys) > 0 {
			fmt.Fprintf(b, "  - 這次用了暫定譯法：%s\n", strings.Join(codes(d.Keys), "、"))
		}
	}
}

const (
	// maxNameDiffRunes is how long a translation in the list of NameDiffs is.
	maxNameDiffRunes = 80
	// maxNoteRunes is how long a note in that list is.
	maxNoteRunes = 200
)

// infoInconsistent lists the texts that were translated in different ways:
// first the groups with units of this update, then the ones of old
// translations only.
func infoInconsistent(b *strings.Builder, info Info) {
	b.WriteString("\n### 同一 KR 不同譯法\n")
	groups := inconsistentGroups(info)
	var withNew, oldOnly []Inconsistency
	for _, g := range groups {
		if g.New > 0 {
			withNew = append(withNew, g)
		} else {
			oldOnly = append(oldOnly, g)
		}
	}
	if len(withNew) == 0 && len(oldOnly) == 0 {
		b.WriteString("\nNone.\n")
		return
	}
	fmt.Fprintf(b, "\n#### 含這次新譯（%d 組）\n\n", len(withNew))
	inconsistentGroupLines(b, withNew)
	b.WriteString("\n#### 只有舊譯\n\n")
	inconsistentGroupLines(b, oldOnly[:min(len(oldOnly), maxInconsistent)])
}

// inconsistentGroupLines writes the lines of groups, None. for no group.
func inconsistentGroupLines(b *strings.Builder, groups []Inconsistency) {
	if len(groups) == 0 {
		b.WriteString("None.\n")
		return
	}
	for _, g := range groups {
		where := "欄位 " + code(g.Field)
		if g.Family != "" {
			where = "族群 " + code(g.Family) + "、" + where
		}
		extra := ""
		if g.New > 0 {
			extra = fmt.Sprintf("，這次新譯 %d 個", g.New)
		}
		fmt.Fprintf(b, "- %s（%s，共 %d 個單位%s）\n", code(g.KR), where, g.Units, extra)
		for i, zh := range g.ZH {
			fmt.Fprintf(b, "  - %s：%s\n", code(zh), code(g.Keys[i]))
		}
	}
}

// infoNotes lists the notes the translator added since the last build.
func infoNotes(b *strings.Builder, rep Report) {
	b.WriteString("\n### Claude 的 note\n\n")
	if len(rep.Notes) == 0 {
		b.WriteString("None.\n")
		return
	}
	for _, n := range rep.Notes {
		fmt.Fprintf(b, "- %s：%s\n", code(n.Key), code(n.Note))
	}
}

// infoGlyphs lists the characters the font cannot show.
func infoGlyphs(b *strings.Builder, rep Report) {
	b.WriteString("\n### Missing glyphs\n\n")
	if len(rep.MissingGlyphs) == 0 {
		b.WriteString("None.\n")
	}
	for _, r := range rep.MissingGlyphs {
		fmt.Fprintf(b, "- U+%04X", r)
		// A character that cannot be seen, or that would end the code span, is
		// only given by its number.
		if unicode.IsPrint(r) && r != '`' {
			fmt.Fprintf(b, " `%c`", r)
		}
		b.WriteByte('\n')
	}
}

// runeSet is a set of characters: one bit for each code point.
type runeSet struct {
	bits [unicode.MaxRune/8 + 1]byte
}

func (s *runeSet) addString(text string) {
	for _, r := range text { // an invalid byte comes out as U+FFFD
		s.bits[r>>3] |= 1 << (r & 7)
	}
}

// all yields the characters of the set in ascending order.
func (s *runeSet) all() iter.Seq[rune] {
	return func(yield func(rune) bool) {
		for i, b := range s.bits {
			if b == 0 {
				continue
			}
			for bit := 0; bit < 8; bit++ {
				if b&(1<<bit) != 0 && !yield(rune(i*8+bit)) {
					return
				}
			}
		}
	}
}
