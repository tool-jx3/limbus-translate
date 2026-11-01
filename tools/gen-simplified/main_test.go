package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleST = `# Open Chinese Convert (OpenCC) Dictionary
# License: Apache-2.0

这	這
后	後 后
干	幹 乾 干 榦
个	個
么	麼
群	羣
秘	祕
并	並 併 幷

个	個
`

const sampleTW = `# Taiwan variants
羣	群
麼	麼 么
`

func TestSimplifiedOnly(t *testing.T) {
	got, err := simplifiedOnly(sampleST, sampleTW)
	if err != nil {
		t.Fatal(err)
	}
	// 后 and 干 are Traditional characters too; 么 and 群 are Taiwan forms
	// per TWVariants; 秘 is in taiwanKept. The duplicate 个 is listed once
	// and the result is in code point order.
	if want := "个并这"; string(got) != want {
		t.Errorf("simplifiedOnly = %q, want %q", string(got), want)
	}
}

func TestSimplifiedOnlyRejectsMalformedLines(t *testing.T) {
	for _, dict := range []string{"这 這\n", "这\t\n", "这这\t這\n"} {
		if _, err := simplifiedOnly(dict, sampleTW); err == nil {
			t.Errorf("STCharacters %q was accepted, want an error", dict)
		}
		if _, err := simplifiedOnly(sampleST, dict); err == nil {
			t.Errorf("TWVariants %q was accepted, want an error", dict)
		}
	}
}

func TestGenerate(t *testing.T) {
	dir := t.TempDir()
	st, tw := filepath.Join(dir, "STCharacters.txt"), filepath.Join(dir, "TWVariants.txt")
	for path, content := range map[string]string{st: sampleST, tw: sampleTW} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var out bytes.Buffer
	if err := generate(st, tw, &out); err != nil {
		t.Fatal(err)
	}
	var header, chars []string
	for _, line := range strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n") {
		if strings.HasPrefix(line, "#") {
			header = append(header, line)
		} else {
			chars = append(chars, line)
		}
	}
	if want := "个 并 这"; strings.Join(chars, " ") != want {
		t.Errorf("characters = %q, want %q", strings.Join(chars, " "), want)
	}
	if h := strings.Join(header, "\n"); !strings.Contains(h, "OpenCC") || !strings.Contains(h, "Apache-2.0") {
		t.Errorf("header should credit OpenCC under Apache-2.0:\n%s", h)
	}

	empty := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(empty, []byte("# nothing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := generate(empty, tw, &bytes.Buffer{}); err == nil {
		t.Error("generate on a file without entries succeeded, want an error")
	}
	if err := generate(st, filepath.Join(dir, "missing.txt"), &bytes.Buffer{}); err == nil {
		t.Error("generate with a missing TWVariants file succeeded, want an error")
	}
}
