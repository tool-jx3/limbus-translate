package adjust_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/adjust"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// writeFile writes rel under root, creating directories.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// krPack opens a KR pack holding files (pack file name -> JSON text).
func krPack(t *testing.T, files map[string]string) *pack.Pack {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		d, base := path.Split(name)
		writeFile(t, dir, d+"KR_"+base, content)
	}
	p, err := pack.Open(dir, "KR_")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// dataList is a file whose dataList has one {"id", field} item per text.
func dataList(field string, texts ...string) string {
	var items []string
	for i, text := range texts {
		quoted, _ := json.Marshal(text)
		items = append(items, fmt.Sprintf(`{"id":%d,%q:%s}`, i+1, field, quoted))
	}
	return `{"dataList":[` + strings.Join(items, ",") + `]}`
}

// texts returns n distinct filler texts with text[at] = special for every
// index in ats.
func texts(n int, special string, ats ...int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("filler %d", i)
	}
	for _, at := range ats {
		out[at] = special
	}
	return out
}

func adjustmentFor(key, kr string) adjust.Adjustment {
	return adjust.Adjustment{Key: key, Raw: "raw-20261029", KRSha: units.KRSha(kr), ZH: "新譯文", Reason: "測試", Rule: adjust.OneOff}
}

func TestResolveApplied(t *testing.T) {
	// The same text twice: the key still names a unit with the sha, so it
	// stays where it is whatever else matches.
	kr := krPack(t, map[string]string{"Sample.json": dataList("content", texts(6, "同一句", 1, 4)...)})
	a := adjustmentFor("Sample.json|/dataList/4/content", "同一句")

	got := adjust.Resolve(a, kr)
	if got.Status != adjust.Applied || got.Key != a.Key {
		t.Errorf("Resolve = %+v, want applied at %s", got, a.Key)
	}
}

func TestResolveMovedPicksTheNearest(t *testing.T) {
	// A line was inserted before the adjusted one: the text that was at
	// index 2 is at 3 now, and the same sentence also occurs at 9.
	kr := krPack(t, map[string]string{"Sample.json": dataList("content", texts(12, "鴻璐的台詞", 3, 9)...)})
	a := adjustmentFor("Sample.json|/dataList/2/content", "鴻璐的台詞")

	got := adjust.Resolve(a, kr)
	if got.Status != adjust.Moved || got.Key != "Sample.json|/dataList/3/content" {
		t.Errorf("Resolve = %+v, want moved to index 3", got)
	}

	// Seen from index 8, the sentence at 9 is the nearest.
	a.Key = "Sample.json|/dataList/8/content"
	got = adjust.Resolve(a, kr)
	if got.Status != adjust.Moved || got.Key != "Sample.json|/dataList/9/content" {
		t.Errorf("Resolve from 8 = %+v, want moved to index 9", got)
	}
}

func TestResolveMovedTieTakesTheEarlier(t *testing.T) {
	kr := krPack(t, map[string]string{"Sample.json": dataList("content", texts(12, "鴻璐的台詞", 3, 9)...)})
	a := adjustmentFor("Sample.json|/dataList/6/content", "鴻璐的台詞") // 3 away from both

	got := adjust.Resolve(a, kr)
	if got.Status != adjust.Moved || got.Key != "Sample.json|/dataList/3/content" {
		t.Errorf("Resolve = %+v, want moved to the earlier index 3", got)
	}
}

func TestResolveMovedInANestedList(t *testing.T) {
	const skills = `{"dataList":[
 {"id":1,"levelList":[{"name":"a","desc":"filler"}]},
 {"id":2,"levelList":[{"name":"b","desc":"filler 2"}]},
 {"id":3,"levelList":[{"name":"c","desc":"filler 3"},{"name":"d","desc":"賦予 5 層"}]}
]}`
	kr := krPack(t, map[string]string{"Skills.json": skills})
	a := adjustmentFor("Skills.json|/dataList/1/levelList/1/desc", "賦予 5 層")

	got := adjust.Resolve(a, kr)
	if got.Status != adjust.Moved || got.Key != "Skills.json|/dataList/2/levelList/1/desc" {
		t.Errorf("Resolve = %+v, want moved to dataList/2/levelList/1/desc", got)
	}
}

func TestResolveExpired(t *testing.T) {
	kr := krPack(t, map[string]string{
		"Sample.json": dataList("content", texts(6, "改過的台詞", 2)...),
		// the same sentence in another file does not count
		"Other.json": dataList("content", "原本的台詞"),
		// nor does it in another field of the same file
		"Fields.json": `{"dataList":[{"id":1,"name":"原本的台詞","content":"別的"}]}`,
	})
	const old = "原本的台詞"
	tests := []struct {
		name string
		key  string
	}{
		{"the text changed", "Sample.json|/dataList/2/content"},
		{"the pointer is gone", "Sample.json|/dataList/40/content"},
		{"the other field has the text", "Fields.json|/dataList/0/content"},
		{"the file is gone from KR", "Gone.json|/dataList/0/content"},
		{"the key is malformed", "no separator"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := adjustmentFor(tc.key, old)
			got := adjust.Resolve(a, kr)
			if got.Status != adjust.Expired || got.Key != tc.key {
				t.Errorf("Resolve = %+v, want expired with the key kept", got)
			}
		})
	}

	// A sentence that exists in another file only is expired for this file.
	a := adjustmentFor("Sample.json|/dataList/0/content", old)
	if got := adjust.Resolve(a, kr); got.Status != adjust.Expired {
		t.Errorf("Resolve = %+v, want expired: Other.json has the text, Sample.json does not", got)
	}
	if got := adjust.Resolve(a, nil); got.Status != adjust.Expired {
		t.Errorf("Resolve without a KR pack = %+v, want expired", got)
	}
}

func TestResolveOutsideDataList(t *testing.T) {
	// Strings that are not under dataList have no index to compare: the one
	// that comes first wins. A dataList item is a candidate too, but the
	// farthest one.
	const doc = `{"note":"filler","tags":["x","共通"],"dataList":[{"tags":["共通"]},{"tags":["y"]}]}`
	kr := krPack(t, map[string]string{"Sample.json": doc})
	a := adjustmentFor("Sample.json|/tags/0", "共通")

	got := adjust.Resolve(a, kr)
	if got.Status != adjust.Moved || got.Key != "Sample.json|/tags/1" {
		t.Errorf("Resolve = %+v, want moved to /tags/1, the other string outside dataList", got)
	}

	a = adjustmentFor("Sample.json|/dataList/1/tags/0", "共通")
	got = adjust.Resolve(a, kr)
	if got.Status != adjust.Moved || got.Key != "Sample.json|/dataList/0/tags/0" {
		t.Errorf("Resolve = %+v, want moved to /dataList/0/tags/0, the one with an index", got)
	}
}

func TestLoad(t *testing.T) {
	root := t.TempDir()
	if got, err := adjust.Load(root); err != nil || len(got) != 0 {
		t.Fatalf("Load without rules/adjustments = %v, %v, want nothing", got, err)
	}

	writeFile(t, root, "rules/adjustments/.gitkeep", "")
	writeFile(t, root, "rules/adjustments/notes.txt", "- key: ignored\n")
	writeFile(t, root, "rules/adjustments/2026-11-03-b.yaml", `- key: B.json|/dataList/0/content
  raw: raw-20261102
  kr_sha: bb
  zh: 乙
  reason: r
  rule: one-off
`)
	writeFile(t, root, "rules/adjustments/2026-11-01-a.yaml", `# a note kept by hand
- key: A.json|/dataList/0/content
  raw: raw-20261029
  kr_sha: aa
  zh: 甲一
  reason: r1
  rule: guides/base.md
- key: A.json|/dataList/1/content
  raw: raw-20261029
  kr_sha: a2
  zh: 甲二
  reason: r2
  rule: one-off
`)
	got, err := adjust.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []adjust.Adjustment{
		{Key: "A.json|/dataList/0/content", Raw: "raw-20261029", KRSha: "aa", ZH: "甲一", Reason: "r1", Rule: "guides/base.md", File: "2026-11-01-a.yaml"},
		{Key: "A.json|/dataList/1/content", Raw: "raw-20261029", KRSha: "a2", ZH: "甲二", Reason: "r2", Rule: "one-off", File: "2026-11-01-a.yaml"},
		{Key: "B.json|/dataList/0/content", Raw: "raw-20261102", KRSha: "bb", ZH: "乙", Reason: "r", Rule: "one-off", File: "2026-11-03-b.yaml"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("Load =\n%+v\nwant\n%+v", got, want)
	}
}

func TestLoadNamesTheBrokenFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rules/adjustments/2026-11-01-bad.yaml", "key: not a list\n")
	_, err := adjust.Load(root)
	if err == nil || !strings.Contains(err.Error(), "2026-11-01-bad.yaml") {
		t.Errorf("Load error = %v, want one naming 2026-11-01-bad.yaml", err)
	}
}

func newAdjustment(key string) adjust.Adjustment {
	return adjust.Adjustment{
		Key:    key,
		Raw:    "raw-20261029",
		KRSha:  units.KRSha("kr of " + key),
		ZH:     "啊……但丁閣下，時鐘的時針好像在不停顫抖呢？您還好嗎？",
		Reason: "鴻璐對但丁說話用敬語，第二人稱用「您」",
		Rule:   "guides/base.md",
	}
}

func TestAddCreatesAndAppends(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "guides/base.md", "# base\n")
	first, second := newAdjustment("Story.json|/dataList/1/content"), newAdjustment("Story.json|/dataList/2/content")
	second.Rule = adjust.OneOff

	path1, err := adjust.Add(root, first, "2026-11-02", "honglu")
	if err != nil {
		t.Fatal(err)
	}
	const wantPath = "rules/adjustments/2026-11-02-honglu.yaml"
	if path1 != wantPath {
		t.Errorf("path = %q, want %q", path1, wantPath)
	}
	afterFirst := readFile(t, root, wantPath)
	if !strings.HasPrefix(afterFirst, "- key: Story.json|/dataList/1/content\n  raw: raw-20261029\n  kr_sha: "+first.KRSha+"\n  zh: ") {
		t.Errorf("the file starts with\n%s\nwant the spec's layout: key, raw, kr_sha, zh, reason, rule", afterFirst)
	}

	path2, err := adjust.Add(root, second, "2026-11-02", "honglu")
	if err != nil || path2 != wantPath {
		t.Fatalf("second Add = %q, %v, want to append to %s", path2, err, wantPath)
	}
	after := readFile(t, root, wantPath)
	if !strings.HasPrefix(after, afterFirst) {
		t.Errorf("appending changed what was in the file:\n%s", after)
	}
	if strings.Count(after, "- key: ") != 2 || strings.Index(after, "/dataList/1/") > strings.Index(after, "/dataList/2/") {
		t.Errorf("the file should hold the two entries in order:\n%s", after)
	}
	got, err := adjust.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	first.File, second.File = "2026-11-02-honglu.yaml", "2026-11-02-honglu.yaml"
	if !slices.Equal(got, []adjust.Adjustment{first, second}) {
		t.Errorf("Load = %+v\nwant %+v, %+v", got, first, second)
	}

	// The same key again is refused, in the file that has it and in any other.
	for _, topic := range []string{"honglu", "another-topic"} {
		_, err = adjust.Add(root, newAdjustment(first.Key), "2026-11-02", topic)
		if !errors.Is(err, adjust.ErrInvalid) || !strings.Contains(err.Error(), "2026-11-02-honglu.yaml") {
			t.Errorf("Add of a known key to topic %q: err = %v, want ErrInvalid naming 2026-11-02-honglu.yaml", topic, err)
		}
	}
	// ... also when the file with the key belongs to another day.
	if _, err = adjust.Add(root, newAdjustment(second.Key), "2026-12-24", "honglu"); !errors.Is(err, adjust.ErrInvalid) {
		t.Errorf("Add of a known key on another date: err = %v, want ErrInvalid", err)
	}
	if got := readFile(t, root, wantPath); got != after {
		t.Errorf("a refused Add changed the file:\n%s", got)
	}
	entries, err := os.ReadDir(filepath.Join(root, "rules/adjustments"))
	if err != nil || len(entries) != 1 {
		t.Errorf("rules/adjustments = %v, %v, want only the one file (no leftovers of refused calls)", entries, err)
	}
}

func TestAddKeepsWhatPeopleWroteInTheFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "guides/base.md", "# base\n")
	// A hand-edited file: a comment, and no newline at the end.
	const mine = "# 鴻璐的敬語\n- key: Story.json|/dataList/0/content\n  raw: raw-20261029\n  kr_sha: " + "0000000000000000000000000000000000000000000000000000000000000000" + "\n  zh: 您好\n  reason: 敬語\n  rule: one-off"
	writeFile(t, root, "rules/adjustments/2026-11-02-honglu.yaml", mine)

	if _, err := adjust.Add(root, newAdjustment("Story.json|/dataList/5/content"), "2026-11-02", "honglu"); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, root, "rules/adjustments/2026-11-02-honglu.yaml")
	if !strings.HasPrefix(got, mine+"\n- key: Story.json|/dataList/5/content\n") {
		t.Errorf("the file is\n%s\nwant the old text, a newline, then the new entry", got)
	}
	if list, err := adjust.Load(root); err != nil || len(list) != 2 {
		t.Errorf("Load = %v, %v, want both entries", list, err)
	}
}

// Texts of a translation are full of characters YAML treats as syntax.
func TestAddRoundTripsAwkwardText(t *testing.T) {
	for _, zh := range []string{
		"賦予 {0} 層 [Laceration]，<color=#f8c200>強度</color>",
		"第一行\n第二行\n",
		"一\n\n二",
		"結尾兩個換行\n\n",
		"\n開頭換行",
		"行尾空格 \n下一行",
		"一\r\n二",
		"- 開頭是減號: 冒號 # 井號",
		`含 "雙引號" 與 'tail'`,
		"2級破裂 ",
		" 前面有空格",
		"true",
		"123",
		"\t縮排",
		strings.Repeat("很長的一句話 ", 30),
	} {
		root := t.TempDir()
		a := newAdjustment("Story.json|/dataList/1/content")
		a.ZH, a.Reason = zh, zh
		a.Rule = adjust.OneOff
		if _, err := adjust.Add(root, a, "2026-11-02", "awkward"); err != nil {
			t.Errorf("Add(%q): %v", zh, err)
			continue
		}
		got, err := adjust.Load(root)
		if err != nil || len(got) != 1 || got[0].ZH != zh || got[0].Reason != zh {
			t.Errorf("zh %q read back as %+v, %v", zh, got, err)
		}
	}
}

func TestAddRefusesWhatLintWouldRefuse(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "guides/base.md", "# base\n")
	writeFile(t, root, "guides/dir/placeholder", "")
	good := newAdjustment("Story.json|/dataList/1/content")

	tests := []struct {
		name         string
		edit         func(*adjust.Adjustment)
		date, topic  string
		wantInReason string
	}{
		{"malformed key", func(a *adjust.Adjustment) { a.Key = "Story.json" }, "2026-11-02", "t", "key"},
		{"no raw tag", func(a *adjust.Adjustment) { a.Raw = "" }, "2026-11-02", "t", "raw"},
		{"short sha", func(a *adjust.Adjustment) { a.KRSha = "abc" }, "2026-11-02", "t", "kr_sha"},
		{"upper case sha", func(a *adjust.Adjustment) { a.KRSha = strings.ToUpper(a.KRSha) }, "2026-11-02", "t", "kr_sha"},
		{"empty zh", func(a *adjust.Adjustment) { a.ZH = " \n" }, "2026-11-02", "t", "zh"},
		{"no reason", func(a *adjust.Adjustment) { a.Reason = "" }, "2026-11-02", "t", "reason"},
		{"no rule", func(a *adjust.Adjustment) { a.Rule = "" }, "2026-11-02", "t", "rule"},
		{"rule file that does not exist", func(a *adjust.Adjustment) { a.Rule = "guides/nothing.md" }, "2026-11-02", "t", "guides/nothing.md"},
		{"rule that is a directory", func(a *adjust.Adjustment) { a.Rule = "guides/dir" }, "2026-11-02", "t", "guides/dir"},
		{"rule above the repo", func(a *adjust.Adjustment) { a.Rule = "../base.md" }, "2026-11-02", "t", "../base.md"},
		{"rule that is absolute", func(a *adjust.Adjustment) { a.Rule = filepath.Join(root, "guides/base.md") }, "2026-11-02", "t", "rule"},
		{"bad date", func(a *adjust.Adjustment) {}, "2026-13-02", "t", "date"},
		{"date without zeros", func(a *adjust.Adjustment) {}, "2026-1-2", "t", "date"},
		{"empty topic", func(a *adjust.Adjustment) {}, "2026-11-02", "", "topic"},
		{"topic with a slash", func(a *adjust.Adjustment) {}, "2026-11-02", "../x", "topic"},
		{"topic with a dot", func(a *adjust.Adjustment) {}, "2026-11-02", "a.yaml", "topic"},
		{"topic with a space", func(a *adjust.Adjustment) {}, "2026-11-02", "a b", "topic"},
		{"topic that starts with a dash", func(a *adjust.Adjustment) {}, "2026-11-02", "-a", "topic"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := good
			tc.edit(&a)
			_, err := adjust.Add(root, a, tc.date, tc.topic)
			if !errors.Is(err, adjust.ErrInvalid) || !strings.Contains(err.Error(), tc.wantInReason) {
				t.Errorf("Add: err = %v, want ErrInvalid mentioning %q", err, tc.wantInReason)
			}
			if _, statErr := os.Stat(filepath.Join(root, "rules")); statErr == nil {
				t.Errorf("a refused Add created rules/")
			}
		})
	}

	// Topics may be written in Chinese, and may use - and _.
	for _, topic := range []string{"鴻璐敬語", "fix_typo-2"} {
		if _, err := adjust.Add(t.TempDir(), func() adjust.Adjustment { a := good; a.Rule = adjust.OneOff; return a }(), "2026-11-02", topic); err != nil {
			t.Errorf("Add with topic %q: %v", topic, err)
		}
	}
}

func TestAddFailsOnABrokenAdjustmentFile(t *testing.T) {
	// A file Add cannot read cannot be checked for the key either: a system
	// error, not ErrInvalid, and nothing is written.
	root := t.TempDir()
	writeFile(t, root, "rules/adjustments/2026-11-01-bad.yaml", "key: not a list\n")
	a := newAdjustment("Story.json|/dataList/1/content")
	a.Rule = adjust.OneOff
	_, err := adjust.Add(root, a, "2026-11-02", "t")
	if err == nil || errors.Is(err, adjust.ErrInvalid) || !strings.Contains(err.Error(), "2026-11-01-bad.yaml") {
		t.Errorf("Add: err = %v, want a system error naming 2026-11-01-bad.yaml", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "rules/adjustments/2026-11-02-t.yaml")); statErr == nil {
		t.Error("the file was written")
	}
}

// Add reads the file back after appending, and writes nothing unless the file
// holds exactly one more entry: a hand-written list that is not a block
// sequence, or a file that ends its document, would swallow the new entry or
// make the whole file unreadable.
func TestAddRefusesAFileItCannotAppendTo(t *testing.T) {
	const entry = "- key: Story.json|/dataList/0/content\n  raw: raw-20261029\n  kr_sha: 0000000000000000000000000000000000000000000000000000000000000000\n  zh: 您好\n  reason: 敬語\n  rule: one-off\n"
	for name, content := range map[string]string{
		"flow sequence": "[{key: Story.json|/dataList/0/content, raw: raw-20261029, kr_sha: '0000000000000000000000000000000000000000000000000000000000000000', zh: 您好, reason: 敬語, rule: one-off}]\n",
		"document end":  entry + "...\n",
		"next document": entry + "---\n",
		"empty list":    "[]\n",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			const file = "rules/adjustments/2026-11-02-honglu.yaml"
			writeFile(t, root, file, content)

			_, err := adjust.Add(root, func() adjust.Adjustment {
				a := newAdjustment("Story.json|/dataList/5/content")
				a.Rule = adjust.OneOff
				return a
			}(), "2026-11-02", "honglu")

			if !errors.Is(err, adjust.ErrInvalid) || !strings.Contains(err.Error(), "2026-11-02-honglu.yaml") {
				t.Errorf("Add: err = %v, want ErrInvalid naming 2026-11-02-honglu.yaml", err)
			}
			if got := readFile(t, root, file); got != content {
				t.Errorf("the file changed:\n%s", got)
			}
			entries, readErr := os.ReadDir(filepath.Join(root, "rules/adjustments"))
			if readErr != nil || len(entries) != 1 {
				t.Errorf("rules/adjustments = %v, %v, want only the one file", entries, readErr)
			}
		})
	}
}
