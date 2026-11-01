package rulefile

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestValidateName(t *testing.T) {
	for _, tc := range []struct{ date, topic string }{
		{"2026-11-05", "honglu"}, {"2026-02-28", "鴻璐-稱呼_2"}, {"2028-02-29", "x"},
	} {
		if err := ValidateName(tc.date, tc.topic); err != nil {
			t.Errorf("ValidateName(%q, %q) = %v, want nil", tc.date, tc.topic, err)
		}
	}
	for _, tc := range []struct{ date, topic, want string }{
		{"", "x", "date"},
		{"2026-1-1", "x", "date"},
		{"2026-13-01", "x", "date"},
		{"2026-02-30", "x", "date"},
		{"20261105", "x", "date"},
		{"2026-11-05", "", "topic"},
		{"2026-11-05", "-x", "topic"},
		{"2026-11-05", "a b", "topic"},
		{"2026-11-05", "a/b", "topic"},
		{"2026-11-05", "../x", "topic"},
		{"2026-11-05", "x.yaml", "topic"},
	} {
		err := ValidateName(tc.date, tc.topic)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("ValidateName(%q, %q) = %v, want an error about the %s", tc.date, tc.topic, err, tc.want)
		}
	}
}

func TestIDAndIsFile(t *testing.T) {
	if !IsFile("a.yaml") || IsFile("a.yml") || IsFile("a.yaml.tmp") {
		t.Error("IsFile accepts only names that end in .yaml")
	}
	if got := ID("2026-11-05-honglu.yaml"); got != "2026-11-05-honglu" {
		t.Errorf("ID = %q", got)
	}
}

func TestWriteAtomic(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "rules", "requeue", "2026-11-05-x.yaml")
	if err := WriteAtomic(p, []byte("a: 1\n")); err != nil {
		t.Fatal(err)
	}
	if err := WriteAtomic(p, []byte("a: 2\n")); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(p); err != nil || string(got) != "a: 2\n" {
		t.Errorf("file = %q, %v; want the second write", got, err)
	}
	entries, err := os.ReadDir(filepath.Dir(p))
	if err != nil || len(entries) != 1 {
		t.Errorf("directory = %v, %v; want only the file (no temporary files left)", entries, err)
	}
	if info, err := os.Stat(p); err != nil || info.Mode().Perm() != 0o644 {
		t.Errorf("mode = %v, %v; want 0644", info.Mode(), err)
	}
}

func TestReadDir(t *testing.T) {
	root := t.TempDir()
	if got, err := ReadDir(root, "rules/requeue"); err != nil || got != nil {
		t.Errorf("ReadDir of a missing directory = %v, %v; want nothing", got, err)
	}
	write(t, root, "rules/requeue/b.yaml", "b")
	write(t, root, "rules/requeue/a.yaml", "a")
	write(t, root, "rules/requeue/.gitkeep", "")
	write(t, root, "rules/requeue/notes.md", "x")
	write(t, root, "rules/requeue/old.yml", "x")
	write(t, root, "rules/requeue/sub/c.yaml", "c")
	got, err := ReadDir(root, "rules/requeue")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range got {
		names = append(names, e.Name+"="+string(e.Data))
	}
	if want := []string{"a.yaml=a", "b.yaml=b"}; !slices.Equal(names, want) {
		t.Errorf("ReadDir = %v, want %v (only .yaml files directly in the directory, by name)", names, want)
	}
}

func TestDecode(t *testing.T) {
	var reason string
	var files []string
	fields := func() map[string]any { return map[string]any{"reason": &reason, "files": &files} }

	keys, err := Decode([]byte("# c\nreason: r\nfiles: [a, b]\nextra: 1\n"), fields())
	if err != nil || !slices.Equal(keys, []string{"reason", "files", "extra"}) || reason != "r" || !slices.Equal(files, []string{"a", "b"}) {
		t.Errorf("Decode = %v, %v, %q, %v; want the three keys in file order and the two known fields", keys, err, reason, files)
	}

	for name, text := range map[string]string{"empty": "", "comments only": "# nothing\n"} {
		reason, files = "", nil
		if keys, err := Decode([]byte(text), fields()); err != nil || len(keys) != 0 || reason != "" {
			t.Errorf("%s: Decode = %v, %v, %q; want nothing and no error", name, keys, err, reason)
		}
	}

	// A key that is written empty is a key; one that is left out is not.
	if keys, err := Decode([]byte("reason: ''\n"), fields()); err != nil || !slices.Equal(keys, []string{"reason"}) {
		t.Errorf("Decode of an empty value = %v, %v", keys, err)
	}

	for name, tc := range map[string]struct{ text, want string }{
		"a list":              {"- a\n- b\n", "mapping"},
		"a scalar":            {"just text\n", "mapping"},
		"two documents":       {"reason: a\n---\nreason: b\n", "document"},
		"a repeated key":      {"reason: a\nreason: b\n", "reason is written twice"},
		"a repeated unknown":  {"x: a\nx: b\n", "x is written twice"},
		"a list for a text":   {"reason: [a]\n", "reason: "},
		"a text for a list":   {"files: a\n", "files: "},
		"a mapping for a key": {"files: {a: b}\n", "files: "},
		"a complex key":       {"? [a]\n: b\n", "plain word"},
		"broken yaml":         {"reason: [a\n", "yaml"},
		"a tab for an indent": {"files:\n\t- a\n", "yaml"},
	} {
		if _, err := Decode([]byte(tc.text), fields()); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: Decode = %v, want an error mentioning %q", name, err, tc.want)
		}
	}
}

func TestLint(t *testing.T) {
	root := t.TempDir()
	write(t, root, "rules/x/.gitkeep", "")
	write(t, root, "rules/x/a.yaml", "ok")
	write(t, root, "rules/x/b.yaml", "bad")
	write(t, root, "rules/x/c.yml", "x")
	write(t, root, "rules/x/sub/d.yaml", "x")

	var seen []string
	got := Lint(root, "rules/x", func(data []byte) []string {
		seen = append(seen, string(data))
		if string(data) == "bad" {
			return []string{"first problem", "second problem"}
		}
		return nil
	})
	var msgs []string
	for _, is := range got {
		msgs = append(msgs, is.File+": "+is.Message)
	}
	want := []string{
		"rules/x/b.yaml: first problem",
		"rules/x/b.yaml: second problem",
		"rules/x/c.yml: not a rule file: only .yaml files directly in rules/x are read",
		"rules/x/sub: not a rule file: only .yaml files directly in rules/x are read",
	}
	if !slices.Equal(msgs, want) {
		t.Errorf("Lint =\n%s\nwant\n%s", strings.Join(msgs, "\n"), strings.Join(want, "\n"))
	}
	if !slices.Equal(seen, []string{"ok", "bad"}) {
		t.Errorf("check saw %v, want the two .yaml files", seen)
	}

	if got := Lint(t.TempDir(), "rules/x", nil); len(got) != 0 {
		t.Errorf("Lint of a missing directory = %v, want nothing", got)
	}
}
