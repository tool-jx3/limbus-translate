package adjust_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/adjust"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// tidyKey is the key of the unit at line n of Story.json.
func tidyKey(n int) string { return "Story.json|/dataList/" + string(rune('0'+n)) + "/content" }

// tidyEntry is one entry in the layout Add writes.
func tidyEntry(key, zh string) string {
	return "- key: " + key + "\n  raw: raw-20261029\n  kr_sha: " + units.KRSha("kr of "+key) + "\n  zh: " + zh + "\n  reason: 測試\n  rule: one-off\n"
}

const tidyDir = "rules/adjustments/"

func never(string) bool  { return false }
func always(string) bool { return true }

func TestTidyUpdatesMovedKeys(t *testing.T) {
	root := t.TempDir()
	// A hand-edited file: a comment on top, a blank line, a block scalar, a
	// comment at the end of a line, quotes, and no newline at the end.
	body := "# 鴻璐的敬語\n" +
		tidyEntry(tidyKey(1), "您好") +
		"\n# 這一筆是手寫的\n" +
		"- key: " + tidyKey(2) + " # 移動的是這一筆\n" +
		"  raw: raw-20261029\n  kr_sha: " + units.KRSha("kr of 2") + "\n" +
		"  zh: |\n    第一行\n    第二行\n" +
		"  reason: \"引號\"\n  rule: one-off\n" +
		tidyEntry(tidyKey(3), "'單引號'") + "- key: \"" + tidyKey(4) + "\"\n  raw: r\n  kr_sha: " + units.KRSha("kr of 4") + "\n  zh: 結尾\n  reason: r\n  rule: one-off"
	other := tidyEntry(tidyKey(5), "另一個檔")
	writeFile(t, root, tidyDir+"2026-11-01-a.yaml", body)
	writeFile(t, root, tidyDir+"2026-11-02-b.yaml", other)

	moves := []adjust.Move{{From: tidyKey(2), To: tidyKey(7)}, {From: tidyKey(4), To: tidyKey(8)}, {From: tidyKey(9), To: tidyKey(6)}}
	changed, err := adjust.Tidy(root, moves, nil, never)
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"rules/adjustments/2026-11-01-a.yaml"}; !slices.Equal(changed, want) {
		t.Errorf("changed = %v, want %v", changed, want)
	}
	want := strings.Replace(body, "key: "+tidyKey(2), "key: "+tidyKey(7), 1)
	want = strings.Replace(want, "\""+tidyKey(4)+"\"", tidyKey(8), 1)
	if got := readFile(t, root, tidyDir+"2026-11-01-a.yaml"); got != want {
		t.Errorf("file =\n%s\nwant only the two keys changed:\n%s", got, want)
	}
	if got := readFile(t, root, tidyDir+"2026-11-02-b.yaml"); got != other {
		t.Errorf("a file with nothing to tidy changed:\n%s", got)
	}

	// The entries read back with the new keys and nothing else different.
	list, err := adjust.Load(root)
	if err != nil || len(list) != 5 {
		t.Fatalf("Load = %v, %v, want 5 entries", list, err)
	}
	gotKeys := []string{list[0].Key, list[1].Key, list[2].Key, list[3].Key, list[4].Key}
	if wantKeys := []string{tidyKey(1), tidyKey(7), tidyKey(3), tidyKey(8), tidyKey(5)}; !slices.Equal(gotKeys, wantKeys) {
		t.Errorf("keys = %v, want %v", gotKeys, wantKeys)
	}
	if list[1].ZH != "第一行\n第二行\n" || list[1].Raw != "raw-20261029" || list[1].KRSha != units.KRSha("kr of 2") {
		t.Errorf("a moved entry lost something besides its key: %+v", list[1])
	}

	// A second run finds nothing to do.
	if changed, err := adjust.Tidy(root, moves, nil, never); err != nil || len(changed) != 0 {
		t.Errorf("second Tidy = %v, %v, want nothing", changed, err)
	}
}

func TestTidyWritesAKeyThatNeedsQuotes(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, tidyDir+"2026-11-01-a.yaml", tidyEntry(tidyKey(1), "您好"))
	for _, to := range []string{
		"Story.json|/dataList/2/a: b # c",
		"Story.json|/dataList/2/'x'",
		`Story.json|/dataList/2/"x"`,
		"Story.json|/dataList/2/ 空白",
		"Story.json|/dataList/2/" + strings.Repeat("很長的欄位名 ", 40),
	} {
		from := tidyKey(1)
		if list, _ := adjust.Load(root); len(list) == 1 {
			from = list[0].Key
		}
		if _, err := adjust.Tidy(root, []adjust.Move{{From: from, To: to}}, nil, never); err != nil {
			t.Errorf("Tidy to %q: %v", to, err)
			continue
		}
		if list, err := adjust.Load(root); err != nil || len(list) != 1 || list[0].Key != to || list[0].ZH != "您好" {
			t.Errorf("after the move to %q the file reads as %+v, %v", to, list, err)
		}
	}
}

func TestTidyRemovesRetranslatedExpired(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, tidyDir+"2026-11-01-a.yaml",
		"# 頭\n"+tidyEntry(tidyKey(1), "甲")+"\n# 乙要留下\n"+tidyEntry(tidyKey(2), "乙")+tidyEntry(tidyKey(3), "丙")+"# 尾\n")
	writeFile(t, root, tidyDir+"2026-11-02-b.yaml", tidyEntry(tidyKey(4), "丁"))
	writeFile(t, root, tidyDir+"2026-11-03-c.yaml", tidyEntry(tidyKey(5), "戊"))

	// Units 1, 3 and 4 were translated again; unit 2 is not, and 5 is not
	// expired at all.
	settled := func(key string) bool { return key != tidyKey(2) }
	done, err := adjust.TidyReport(root, nil, []string{tidyKey(1), tidyKey(2), tidyKey(3), tidyKey(4)}, settled)
	if err != nil {
		t.Fatal(err)
	}
	changed := done.Changed

	if done.Removed != 3 || done.Moved != 0 {
		t.Errorf("TidyReport = %+v, want 3 removed", done)
	}
	if want := []string{"rules/adjustments/2026-11-01-a.yaml", "rules/adjustments/2026-11-02-b.yaml"}; !slices.Equal(changed, want) {
		t.Errorf("changed = %v, want %v", changed, want)
	}
	if got, want := readFile(t, root, tidyDir+"2026-11-01-a.yaml"), "# 頭\n\n# 乙要留下\n"+tidyEntry(tidyKey(2), "乙")+"# 尾\n"; got != want {
		t.Errorf("file a =\n%s\nwant\n%s", got, want)
	}
	if _, err := os.Stat(filepath.Join(root, tidyDir+"2026-11-02-b.yaml")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the file whose only entry went is still there: %v", err)
	}
	if got := readFile(t, root, tidyDir+"2026-11-03-c.yaml"); got != tidyEntry(tidyKey(5), "戊") {
		t.Errorf("file c changed:\n%s", got)
	}
	if list, err := adjust.Load(root); err != nil || len(list) != 2 || list[0].Key != tidyKey(2) || list[1].Key != tidyKey(5) {
		t.Errorf("Load = %+v, %v, want the entries of units 2 and 5", list, err)
	}
}

func TestTidyRemovesTheLastEntryOfAFile(t *testing.T) {
	root := t.TempDir()
	const kept = "- key: " + "Story.json|/dataList/1/content" + "\n  raw: r\n  kr_sha: x\n  zh: |\n    留\n  reason: r\n  rule: one-off\n"
	gone := "- key: " + tidyKey(2) + "\n  raw: r\n  kr_sha: x\n  zh: |\n    走\n\n    兩段\n  reason: r\n  rule: one-off"
	// The last entry has a block scalar with a blank line in it, and the file
	// has no newline at its end.
	writeFile(t, root, tidyDir+"2026-11-01-a.yaml", kept+gone)

	if _, err := adjust.Tidy(root, nil, []string{tidyKey(2)}, always); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, root, tidyDir+"2026-11-01-a.yaml"); got != kept {
		t.Errorf("file =\n%q\nwant\n%q", got, kept)
	}
}

func TestTidyKeepsPendingExpired(t *testing.T) {
	root := t.TempDir()
	body := tidyEntry(tidyKey(1), "甲") + tidyEntry(tidyKey(2), "乙")
	writeFile(t, root, tidyDir+"2026-11-01-a.yaml", body)

	changed, err := adjust.Tidy(root, nil, []string{tidyKey(1), tidyKey(2), "Story.json|/dataList/9/content"}, never)
	if err != nil || len(changed) != 0 {
		t.Errorf("Tidy = %v, %v, want nothing changed", changed, err)
	}
	if got := readFile(t, root, tidyDir+"2026-11-01-a.yaml"); got != body {
		t.Errorf("the file changed:\n%s", got)
	}
}

func TestTidyAsksAboutTheKeyOfTheAdjustmentOnly(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, tidyDir+"2026-11-01-a.yaml", tidyEntry(tidyKey(1), "甲")+tidyEntry(tidyKey(2), "乙"))
	var asked []string
	_, err := adjust.Tidy(root, nil, []string{tidyKey(2), "Story.json|/dataList/9/content"}, func(key string) bool {
		asked = append(asked, key)
		return false
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{tidyKey(2)}; !slices.Equal(asked, want) {
		t.Errorf("settled was asked about %v, want %v (an expired key that no file holds is not asked about)", asked, want)
	}
}

// Tidy must leave the files lint-clean: a key is held once across all files.
func TestTidyDoesNotCreateADuplicateKey(t *testing.T) {
	root := t.TempDir()
	// 1 moves to 2, which an adjustment that stays holds: not moved. 3 moves
	// to 4 while 4 moves on to 5: both move. 6 and 7 both want 8: the first
	// gets it.
	writeFile(t, root, tidyDir+"2026-11-01-a.yaml",
		tidyEntry(tidyKey(1), "甲")+tidyEntry(tidyKey(2), "乙")+tidyEntry(tidyKey(3), "丙")+tidyEntry(tidyKey(4), "丁")+
			tidyEntry(tidyKey(6), "己")+tidyEntry(tidyKey(7), "庚"))

	moves := []adjust.Move{
		{From: tidyKey(1), To: tidyKey(2)},
		{From: tidyKey(3), To: tidyKey(4)}, {From: tidyKey(4), To: tidyKey(5)},
		{From: tidyKey(6), To: tidyKey(8)}, {From: tidyKey(7), To: tidyKey(8)},
	}
	done, err := adjust.TidyReport(root, moves, nil, never)
	if err != nil {
		t.Fatal(err)
	}
	// The counts are of what was done: the two moves that were not made are
	// not counted.
	if done.Moved != 3 || done.Removed != 0 || !slices.Equal(done.Changed, []string{"rules/adjustments/2026-11-01-a.yaml"}) {
		t.Errorf("TidyReport = %+v, want 3 moved, none removed and the one file changed", done)
	}
	list, err := adjust.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, a := range list {
		keys = append(keys, a.Key+"="+a.ZH)
	}
	want := []string{tidyKey(1) + "=甲", tidyKey(2) + "=乙", tidyKey(4) + "=丙", tidyKey(5) + "=丁", tidyKey(8) + "=己", tidyKey(7) + "=庚"}
	if !slices.Equal(keys, want) {
		t.Errorf("entries = %v, want %v", keys, want)
	}
}

// A key moves to a place an expired adjustment leaves in the same run.
func TestTidyMovesToTheKeyOfARemovedEntry(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, tidyDir+"2026-11-01-a.yaml", tidyEntry(tidyKey(1), "甲"))
	writeFile(t, root, tidyDir+"2026-11-02-b.yaml", tidyEntry(tidyKey(2), "乙"))
	changed, err := adjust.Tidy(root, []adjust.Move{{From: tidyKey(1), To: tidyKey(2)}}, []string{tidyKey(2)}, always)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"rules/adjustments/2026-11-01-a.yaml", "rules/adjustments/2026-11-02-b.yaml"}; !slices.Equal(changed, want) {
		t.Errorf("changed = %v, want %v", changed, want)
	}
	if list, err := adjust.Load(root); err != nil || len(list) != 1 || list[0].Key != tidyKey(2) || list[0].ZH != "甲" {
		t.Errorf("Load = %+v, %v, want the first entry at the second key", list, err)
	}
}

func TestTidyKeepsLineEndings(t *testing.T) {
	root := t.TempDir()
	crlf := func(s string) string { return strings.ReplaceAll(s, "\n", "\r\n") }
	writeFile(t, root, tidyDir+"2026-11-01-a.yaml", crlf(tidyEntry(tidyKey(1), "甲")+tidyEntry(tidyKey(2), "乙")+tidyEntry(tidyKey(3), "丙")))
	if _, err := adjust.Tidy(root, []adjust.Move{{From: tidyKey(3), To: tidyKey(4)}}, []string{tidyKey(1)}, always); err != nil {
		t.Fatal(err)
	}
	moved := strings.Replace(tidyEntry(tidyKey(3), "丙"), "key: "+tidyKey(3), "key: "+tidyKey(4), 1)
	if got, want := readFile(t, root, tidyDir+"2026-11-01-a.yaml"), crlf(tidyEntry(tidyKey(2), "乙")+moved); got != want {
		t.Errorf("file = %q, want %q", got, want)
	}
}

// A file Tidy cannot edit without guessing is an error, not a half edit; the
// other files are left as they were too.
func TestTidyRefusesAFileItCannotEdit(t *testing.T) {
	flow := "[{key: " + tidyKey(1) + ", raw: r, kr_sha: x, zh: 甲, reason: r, rule: one-off}]\n"
	for name, body := range map[string]string{
		"flow sequence":     flow,
		"flow entry":        "- {key: " + tidyKey(1) + ", raw: r, kr_sha: x, zh: 甲, reason: r, rule: one-off}\n",
		"document end":      tidyEntry(tidyKey(1), "甲") + "...\n",
		"a second document": tidyEntry(tidyKey(1), "甲") + "---\n" + tidyEntry(tidyKey(9), "別的"),
		"block scalar key":  "- key: |-\n    " + tidyKey(1) + "\n  raw: r\n  kr_sha: x\n  zh: 甲\n  reason: r\n  rule: one-off\n",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			good := tidyEntry(tidyKey(2), "乙")
			writeFile(t, root, tidyDir+"2026-11-01-a.yaml", body)
			writeFile(t, root, tidyDir+"2026-11-02-b.yaml", good)

			changed, err := adjust.Tidy(root, []adjust.Move{{From: tidyKey(1), To: tidyKey(3)}, {From: tidyKey(2), To: tidyKey(4)}}, nil, never)
			if !errors.Is(err, adjust.ErrInvalid) || !strings.Contains(err.Error(), "2026-11-01-a.yaml") {
				t.Errorf("Tidy = %v, %v, want ErrInvalid naming 2026-11-01-a.yaml", changed, err)
			}
			if got := readFile(t, root, tidyDir+"2026-11-01-a.yaml"); got != body {
				t.Errorf("the file changed:\n%s", got)
			}
			if got := readFile(t, root, tidyDir+"2026-11-02-b.yaml"); got != good {
				t.Errorf("another file changed although Tidy failed:\n%s", got)
			}
		})
	}
}

// A file with nothing to tidy is not looked at, however it is written.
func TestTidyLeavesAloneTheFilesItDoesNotNeed(t *testing.T) {
	root := t.TempDir()
	flow := "[{key: " + tidyKey(1) + ", raw: r, kr_sha: x, zh: 甲, reason: r, rule: one-off}]\n"
	writeFile(t, root, tidyDir+"2026-11-01-a.yaml", flow)
	writeFile(t, root, tidyDir+"2026-11-02-b.yaml", tidyEntry(tidyKey(2), "乙"))
	changed, err := adjust.Tidy(root, []adjust.Move{{From: tidyKey(2), To: tidyKey(4)}}, nil, never)
	if err != nil || len(changed) != 1 {
		t.Errorf("Tidy = %v, %v, want the one file with a move", changed, err)
	}
	if got := readFile(t, root, tidyDir+"2026-11-01-a.yaml"); got != flow {
		t.Errorf("the flow file changed:\n%s", got)
	}
}

func TestTidyWithoutFilesOrWork(t *testing.T) {
	root := t.TempDir()
	if changed, err := adjust.Tidy(root, []adjust.Move{{From: "a|/b", To: "a|/c"}}, []string{"a|/d"}, always); err != nil || len(changed) != 0 {
		t.Errorf("Tidy without rules/adjustments = %v, %v, want nothing", changed, err)
	}
	writeFile(t, root, tidyDir+".gitkeep", "")
	writeFile(t, root, tidyDir+"README.md", "# 說明\n")
	writeFile(t, root, tidyDir+"2026-11-01-empty.yaml", "")
	if changed, err := adjust.Tidy(root, nil, []string{"a|/d"}, always); err != nil || len(changed) != 0 {
		t.Errorf("Tidy = %v, %v, want nothing", changed, err)
	}
	if got := readFile(t, root, tidyDir+"README.md"); got != "# 說明\n" {
		t.Errorf("README.md changed: %q", got)
	}
}

func TestTidyFailsOnABrokenAdjustmentFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, tidyDir+"2026-11-01-bad.yaml", "key: not a list\n")
	_, err := adjust.Tidy(root, []adjust.Move{{From: tidyKey(1), To: tidyKey(2)}}, nil, never)
	if err == nil || errors.Is(err, adjust.ErrInvalid) || !strings.Contains(err.Error(), "2026-11-01-bad.yaml") {
		t.Errorf("Tidy: err = %v, want a system error naming 2026-11-01-bad.yaml (as Load)", err)
	}
}
