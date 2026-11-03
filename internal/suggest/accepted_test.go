package suggest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const goodWarningID = "warning/kana/0123abcd/89ef4567"

func TestLoadAcceptedMissingFile(t *testing.T) {
	list, err := LoadAccepted(t.TempDir())
	if err != nil || len(list) != 0 {
		t.Fatalf("LoadAccepted of a root without the file = %v, %v; want none", list, err)
	}
}

func TestAppendEntriesRoundTrip(t *testing.T) {
	root := t.TempDir()
	const header = "# 已確認的檢查警告\n#\n# 說明\n"
	writeFile(t, root, AcceptedFile, header+"- id: \"warning/kana/00000000/11111111\"\n  reason: first\n  date: 2026-11-01")
	entries := []Declined{
		{ID: goodWarningID, Reason: "[原文如此] 日文假名是刻意的", Date: "2026-11-03"},
		{ID: "warning/kana/aaaaaaaa/bbbbbbbb", Reason: "*強調*: 見 #12", Date: "2026-11-04"},
	}
	if err := AppendEntries(root, AcceptedFile, entries); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, AcceptedFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), header+"- id: \"warning/kana/00000000/11111111\"\n  reason: first\n  date: 2026-11-01\n") {
		t.Errorf("existing text changed:\n%s", data)
	}
	if !strings.Contains(string(data), "- id: \""+goodWarningID+"\"\n  reason: ") || !strings.Contains(string(data), "\n  date: 2026-11-03\n") {
		t.Errorf("entries are not written like declined.yaml:\n%s", data)
	}
	got, err := LoadAccepted(root)
	if err != nil {
		t.Fatal(err)
	}
	want := append([]Declined{{ID: "warning/kana/00000000/11111111", Reason: "first", Date: "2026-11-01"}}, entries...)
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	if issues := LintAccepted(root); len(issues) != 0 {
		t.Errorf("LintAccepted = %v, want none", issues)
	}
}

func TestAppendEntriesCreatesFileWithHeader(t *testing.T) {
	root := t.TempDir()
	if err := AppendEntries(root, AcceptedFile, []Declined{{ID: goodWarningID, Reason: "ok", Date: "2026-11-03"}}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(root, AcceptedFile))
	if !strings.HasPrefix(string(data), "#") {
		t.Errorf("a new file starts without the explaining comment:\n%s", data)
	}
	if list, err := LoadAccepted(root); err != nil || len(list) != 1 {
		t.Errorf("LoadAccepted = %v, %v", list, err)
	}
}

func TestAppendEntriesNothingToAdd(t *testing.T) {
	root := t.TempDir()
	if err := AppendEntries(root, AcceptedFile, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, AcceptedFile)); err == nil {
		t.Error("no entries still created the file")
	}
}

func TestLintAccepted(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, AcceptedFile, `
- id: warning/kana/xyz
  reason: r
  date: 2026-11-03
- id: `+goodWarningID+`
  reason: ""
  date: 2026-11-03
- id: warning/kana/aaaaaaaa/bbbbbbbb
  reason: r
  date: 3 Nov
- id: `+goodWarningID+`
  reason: r
  date: 2026-11-03
`)
	text := ""
	for _, is := range LintAccepted(root) {
		if is.File != AcceptedFile {
			t.Errorf("issue file = %q", is.File)
		}
		text += is.Message + "\n"
	}
	for _, want := range []string{"warning/kana/xyz", "reason", "date", "listed twice"} {
		if !strings.Contains(text, want) {
			t.Errorf("issues lack %q:\n%s", want, text)
		}
	}
	if issues := LintAccepted(t.TempDir()); len(issues) != 0 {
		t.Errorf("a root without the file has issues: %v", issues)
	}
}

// A new file starts with the same comment that rules/accepted.yaml has.
func TestAcceptedHeaderMatchesRepoFile(t *testing.T) {
	want, err := os.ReadFile("../../rules/accepted.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if acceptedHeader != string(want) {
		t.Errorf("acceptedHeader differs from rules/accepted.yaml:\n%s", acceptedHeader)
	}
}
