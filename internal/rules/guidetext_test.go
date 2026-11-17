package rules_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/rules"
)

func writeGuideText(t *testing.T, root, rel, text string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGuideEvidenceLinks(t *testing.T) {
	root := t.TempDir()
	line := "見 docs/guides-evidence/families/x.md\n"
	writeGuideText(t, root, "guides/families/x.md", line)
	writeGuideText(t, root, "guides/README.md", line)
	writeGuideText(t, root, "guides/families.md", line)

	var got []rules.LintError
	for _, err := range rules.Lint(root) {
		if le, ok := err.(rules.LintError); ok && strings.Contains(le.Message, "guides-evidence") {
			got = append(got, le)
		}
	}
	if len(got) != 1 || got[0].File != "guides/families/x.md" || !strings.Contains(got[0].Message, "line 1") {
		t.Fatalf("errors = %+v", got)
	}
}

func TestLintGuideTextWarnings(t *testing.T) {
	root := t.TempDir()
	hits := []string{
		"（既有 2,073 處）",
		"（2,255／2,303）",
		"97.6%（",
		"（使用者決定 2026-11-07）",
		"raw-20261105",
		"docs/research/x.md",
	}
	writeGuideText(t, root, "guides/families/x.md", strings.Join(hits, "\n")+"\n")
	writeGuideText(t, root, "guides/shared/ok.md", "不低於3/5級\n使自身獲得N級 [X] 強度\n+N%\n<size=50%>\n1回 [Laceration]\n")
	writeGuideText(t, root, "guides/README.md", strings.Join(hits, "\n")+"\n")
	writeGuideText(t, root, "guides/families.md", strings.Join(hits, "\n")+"\n")

	got := rules.LintGuideTextWarnings(root)
	if len(got) != len(hits) {
		t.Fatalf("warnings = %+v", got)
	}
	for i, w := range got {
		if w.File != "guides/families/x.md" || !strings.Contains(w.Message, fmt.Sprintf("line %d ", i+1)) ||
			!strings.Contains(w.Message, "docs/guides-evidence/<same path>") {
			t.Errorf("warning %d = %+v", i, w)
		}
	}
}

// An unreadable guide (a dangling symlink, or a directory for base.md) is a lint error and a warning, not
// a silent skip.
func TestUnlistableGuideDir(t *testing.T) {
	// A file in place of guides/shared: listing the guides fails, so lint
	// reports an error and the text check a warning for "guides".
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "guides"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rules.SharedGuidesDir)), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	found := false
	for _, err := range rules.Lint(root) {
		if le, ok := err.(rules.LintError); ok && le.File == "guides" {
			found = true
		}
	}
	if !found {
		t.Errorf("lint reports no error for guides when %s cannot be listed", rules.SharedGuidesDir)
	}

	got := rules.LintGuideTextWarnings(root)
	if len(got) != 1 || got[0].File != "guides" {
		t.Errorf("warnings = %+v", got)
	}
}

func TestUnreadableGuide(t *testing.T) {
	for _, rel := range []string{
		"guides/base.md",
		"guides/families/x.md",
		"guides/shared/x.md",
		"guides/characters/x.md",
	} {
		t.Run(rel, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if rel == rules.BaseGuide {
				// A directory in place of the file: Stat finds it, reading fails.
				if err := os.Mkdir(path, 0o755); err != nil {
					t.Fatal(err)
				}
			} else if err := os.Symlink(filepath.Join(root, "missing.md"), path); err != nil {
				t.Skip("symlinks unavailable:", err)
			}

			found := false
			for _, err := range rules.Lint(root) {
				if le, ok := err.(rules.LintError); ok && le.File == rel {
					found = true
				}
			}
			if !found {
				t.Errorf("lint reports no error for unreadable %s", rel)
			}

			got := rules.LintGuideTextWarnings(root)
			if len(got) != 1 || got[0].File != rel {
				t.Errorf("warnings = %+v", got)
			}
		})
	}
}
