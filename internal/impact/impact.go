// Package impact answers what a change to the rules would do to the
// translations there are: how many of them the new rules reject that the old
// rules did not (spec 11.2 and 12). Nothing in the rules is applied to a
// translation that exists, so this is the number of units a person would have
// to send back to be translated again for the translations to follow the new
// rules. A rule PR shows it, and ci.yml shows it on the pull requests that
// change the rules.
package impact

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/batch"
	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// MaxSamples is how many keys Result.Samples holds for each code.
const MaxSamples = 10

// Inputs is what Run compares.
type Inputs struct {
	// Old and New are the rules before and after the change.
	Old, New *rules.Rules
	// Records are the translations, by unit key (store.Store.LoadRecords).
	Records map[string]store.Record
	// KR is the KR pack of the current raw snapshot.
	KR *pack.Pack
	// Frozen are the patterns of the files that are never translated
	// (lct.yaml's frozen list; config.MatchAny): their units do not count.
	Frozen []string
}

// Result is what Run finds.
type Result struct {
	// Checked is the number of translations that count (see Run): each was
	// held to the new rules, and to the old ones when the new ones reject it.
	Checked int `json:"checked"`
	// Units is the number of those the new rules reject for a reason the old
	// rules did not give. A unit counts once, whatever number of codes it has.
	Units int `json:"units"`
	// ByCode counts those units by the issue codes they newly have; a unit
	// with two new codes is in two of the counts.
	ByCode map[string]int `json:"by_code"`
	// ByOrigin counts those units by the origin of their record.
	ByOrigin map[string]int `json:"by_origin"`
	// Samples are the keys of those units for each code: the first MaxSamples
	// in sorted order.
	Samples map[string][]string `json:"samples"`
}

// counted reports whether the translations of an origin are looked at: the
// ones a person or a model wrote or adopted. A copy is an identifier or a text
// that is the same in every language, and a fallback is a stand-in for a
// translation that failed; neither is a translation that a rule could break.
func counted(origin string) bool {
	switch origin {
	case store.OriginParatranz, store.OriginClaude, store.OriginHuman:
		return true
	}
	return false
}

// Run holds every translation that counts to the new rules and to the old
// rules the way a new translation is held (batch.CheckNew), and counts the
// units that have a hard issue under the new rules that they do not have under
// the old.
//
// A translation counts when its origin does (see counted), its record was made
// from the KR text there is now (kr_sha), and its file is not frozen. The
// others are not checked: a translation made from an older text is pending
// again anyway, and a frozen file is not ours to translate.
//
// An issue is told from another by its code and its message: the message names
// the forbidden rendering, so a unit that already breaks one forbidden term and
// now breaks a second, newly forbidden one has an issue the old rules did not
// give, and is counted. A forbidden rendering written as a pattern (re:) is
// named with its pattern too, so a unit the old pattern rejected already counts
// again when the pattern is edited. Each code of the new issues counts the
// unit once, however many new issues of that code it has.
func Run(in Inputs) (*Result, error) {
	if in.Old == nil || in.New == nil || in.KR == nil {
		return nil, errors.New("impact: the old rules, the new rules and the KR pack are all needed")
	}
	res := &Result{ByCode: map[string]int{}, ByOrigin: map[string]int{}, Samples: map[string][]string{}}
	keys := map[string][]string{} // code to the keys of its units
	for _, file := range filesToCheck(in) {
		doc, err := in.KR.Doc(file)
		if err != nil {
			return nil, fmt.Errorf("impact: %w", err)
		}
		for _, u := range units.Extract(file, doc) {
			key := units.Key(file, u.Pointer)
			rec, ok := in.Records[key]
			if !ok || !counted(rec.Origin) || rec.KRSha != units.KRSha(u.KR) {
				continue
			}
			res.Checked++
			// The old rules are asked only about a unit the new rules reject:
			// that is almost none, and each question costs as much as the first.
			added := hardIssues(batch.CheckNew(in.New, file, u.KR, rec.ZH))
			if len(added) == 0 {
				continue
			}
			before := hardIssues(batch.CheckNew(in.Old, file, u.KR, rec.ZH))
			added = slices.DeleteFunc(added, func(is issue) bool { return slices.Contains(before, is) })
			if len(added) == 0 {
				continue
			}
			res.Units++
			res.ByOrigin[rec.Origin]++
			var codes []string // the codes of the new issues, each once
			for _, is := range added {
				if !slices.Contains(codes, is.code) {
					codes = append(codes, is.code)
				}
			}
			for _, code := range codes {
				res.ByCode[code]++
				keys[code] = append(keys[code], key)
			}
		}
	}
	for code, list := range keys {
		slices.Sort(list)
		res.Samples[code] = list[:min(len(list), MaxSamples)]
	}
	return res, nil
}

// filesToCheck lists, sorted, the files of the KR pack that hold a record that
// counts and are not frozen.
func filesToCheck(in Inputs) []string {
	wanted := map[string]bool{} // by file: asked once for each
	for key, rec := range in.Records {
		if !counted(rec.Origin) {
			continue
		}
		file, _, err := units.SplitKey(key)
		if err != nil {
			continue // a key that is no unit of any KR text
		}
		if _, done := wanted[file]; !done {
			wanted[file] = in.KR.Has(file) && !config.MatchAny(in.Frozen, file)
		}
	}
	var files []string
	for file, ok := range wanted {
		if ok {
			files = append(files, file)
		}
	}
	slices.Sort(files)
	return files
}

// issue is a hard issue as far as Run tells one from another: its code and its
// message.
type issue struct{ code, message string }

// hardIssues returns the distinct hard issues, in the order they come.
func hardIssues(issues []validate.Issue) []issue {
	var out []issue
	for _, is := range issues {
		h := issue{is.Code, is.Message}
		if is.Severity == validate.SeverityHard && !slices.Contains(out, h) {
			out = append(out, h)
		}
	}
	return out
}

// String is the result for a person at a terminal.
func (r *Result) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d of %d translations would be rejected by the new rules and were not by the old\n", r.Units, r.Checked)
	for _, code := range slices.Sorted(maps.Keys(r.ByCode)) {
		fmt.Fprintf(&b, "  %s\t%d\n", code, r.ByCode[code])
	}
	if len(r.ByOrigin) > 0 {
		var parts []string
		for _, origin := range slices.Sorted(maps.Keys(r.ByOrigin)) {
			parts = append(parts, fmt.Sprintf("%s %d", origin, r.ByOrigin[origin]))
		}
		fmt.Fprintf(&b, "by origin: %s\n", strings.Join(parts, ", "))
	}
	for _, code := range slices.Sorted(maps.Keys(r.Samples)) {
		fmt.Fprintf(&b, "%s, for example:\n", code)
		for _, key := range r.Samples[code] {
			fmt.Fprintf(&b, "  %s\n", key)
		}
	}
	return b.String()
}

// Markdown is the result for the page of a pull request or a workflow run,
// base being the revision whose rules the new ones are compared with.
func (r *Result) Markdown(base string) string {
	var b strings.Builder
	b.WriteString("## 規則變更的影響評估\n\n")
	scope := fmt.Sprintf("檢查了 %d 筆既有譯文（來源是 Paratranz、Claude 或人工，而且是照目前的 KR 原文翻的；凍結檔不計）", r.Checked)
	if r.Units == 0 {
		fmt.Fprintf(&b, "與 `%s` 的規則相比，這次的規則變更不會讓既有譯文出現新的硬性問題。%s。\n", base, scope)
		return b.String()
	}
	fmt.Fprintf(&b, "與 `%s` 的規則相比，這次的規則變更會讓 **%d** 筆既有譯文出現新的硬性問題。%s。\n\n", base, r.Units, scope)

	b.WriteString("| 問題代碼 | 單位數 |\n|---|---|\n")
	for _, code := range slices.Sorted(maps.Keys(r.ByCode)) {
		fmt.Fprintf(&b, "| `%s` | %d |\n", code, r.ByCode[code])
	}
	b.WriteString("\n| 譯文來源 | 單位數 |\n|---|---|\n")
	for _, origin := range slices.Sorted(maps.Keys(r.ByOrigin)) {
		fmt.Fprintf(&b, "| `%s` | %d |\n", origin, r.ByOrigin[origin])
	}

	b.WriteString("\n### 範例\n")
	for _, code := range slices.Sorted(maps.Keys(r.Samples)) {
		keys := r.Samples[code]
		fmt.Fprintf(&b, "\n`%s`", code)
		if total := r.ByCode[code]; total > len(keys) {
			fmt.Fprintf(&b, "（共 %d 個，列出前 %d 個）", total, len(keys))
		}
		b.WriteString("：\n\n")
		for _, key := range keys {
			fmt.Fprintf(&b, "- `%s`\n", key)
		}
	}
	b.WriteString("\n規則改了不會自動重翻舊譯文。要套用到舊譯文，必須由人指定範圍：用 `lct requeue`，或在規則 PR 附上 `rules/requeue/*.yaml`。\n")
	return b.String()
}
