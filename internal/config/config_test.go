package config

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.LangDir != "zh-TW" {
		t.Errorf("LangDir = %q, want zh-TW", cfg.LangDir)
	}
	if want := []string{"BgmLyrics/**"}; !slices.Equal(cfg.Frozen, want) {
		t.Errorf("Frozen = %v, want %v", cfg.Frozen, want)
	}
	if want := "https://downloadcommon.limbuscompanycdn.org/{token}/Assets/LocalizePatch/localize_{lang}.zip"; cfg.CDNTemplate != want {
		t.Errorf("CDNTemplate = %q, want %q", cfg.CDNTemplate, want)
	}
	if cfg.Worktrees.Raw != ".work/raw" || cfg.Worktrees.TW != ".work/tw" {
		t.Errorf("Worktrees = %+v, want raw=.work/raw tw=.work/tw", cfg.Worktrees)
	}
	wantBatch := Batch{MaxUnits: 40, StoryMaxUnits: 120, MaxAttempts: 3, LeaseMinutes: 120}
	if cfg.Batch != wantBatch {
		t.Errorf("Batch = %+v, want %+v", cfg.Batch, wantBatch)
	}
	wantFont := Font{
		Path:    "assets/fonts/SarasaGothicTC-Bold.ttf",
		Version: "1.0.43",
		SHA256:  "cddc7ff155199b59656c750f881f696ebd1b3a0baf053ec8333b712f8429c20a",
	}
	if cfg.Font != wantFont {
		t.Errorf("Font = %+v, want %+v", cfg.Font, wantFont)
	}

	b := cfg.Bootstrap
	if b.RawToken != "l20261029_n2rX1Jku6_9sBiu9Np24" {
		t.Errorf("Bootstrap.RawToken = %q", b.RawToken)
	}
	if b.RawCreatedAt != "2026-10-29 02:55:23" {
		t.Errorf("Bootstrap.RawCreatedAt = %q", b.RawCreatedAt)
	}
	wantUpstreams := map[string]struct{ got, want Upstream }{
		"paratranz": {b.Paratranz, Upstream{
			Repo: "user-unknown-0711/ParatranzUploader", Tag: "1793285016", Asset: "localize_files.zip",
			SHA256: "d1af42152a93ea5eb6b7dad75a0646592d8e6e1df1e1e63c13681ef5106528ea",
		}},
		"ltm": {b.LTM, Upstream{
			Repo: "LimbusTraditionalMandarin/storyline", Tag: "1792667238", Asset: "Hant.zip",
			SHA256: "7a918ba656716f4a8a8f296d517d6bae26ae35326589b60ea8ef6dc7e5677217",
		}},
		"llc": {b.LLC, Upstream{
			Repo: "LocalizeLimbusCompany/LocalizeLimbusCompany", Tag: "2026103002", Asset: "LimbusLocalize_2026103002.zip",
			SHA256: "59d7af16870c2de3f1b6ebb42387e738771ba75e0141ac6f6c46893d347e3ca9",
		}},
	}
	for name, u := range wantUpstreams {
		if u.got != u.want {
			t.Errorf("Bootstrap.%s = %+v, want %+v", name, u.got, u.want)
		}
	}
}

func writeConfig(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(t.TempDir()); err == nil {
		t.Fatal("Load without lct.yaml: want error, got nil")
	}
}

func TestLoadRejectsUnknownKey(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "lang_dir: zh-TW\nlang_dri: typo\n")
	if _, err := Load(dir); err == nil {
		t.Fatal("Load with unknown key: want error, got nil")
	}
}

func TestFindRoot(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, "lang_dir: zh-TW\n")
	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, start := range []string{root, deep} {
		got, err := FindRoot(start)
		if err != nil {
			t.Fatalf("FindRoot(%q): %v", start, err)
		}
		if got != root {
			t.Errorf("FindRoot(%q) = %q, want %q", start, got, root)
		}
	}
}

func TestFindRootNotFound(t *testing.T) {
	if _, err := FindRoot(t.TempDir()); err == nil {
		t.Fatal("FindRoot outside a repo: want error, got nil")
	}
}

func TestMatchAny(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		file     string
		want     bool
	}{
		{"directory pattern, file below it", []string{"BgmLyrics/**"}, "BgmLyrics/BgmLyrics_Credit7.json", true},
		{"directory pattern, nested file", []string{"BgmLyrics/**"}, "BgmLyrics/a/b.json", true},
		{"directory pattern, sibling with the same prefix", []string{"BgmLyrics/**"}, "BgmLyricsExtra/a.json", false},
		{"directory pattern, file at the root", []string{"BgmLyrics/**"}, "BgmLyrics.json", false},
		{"directory pattern, the directory name alone", []string{"BgmLyrics/**"}, "BgmLyrics", false},
		{"directory pattern, other directory", []string{"BgmLyrics/**"}, "StoryData/S1.json", false},
		{"glob pattern", []string{"Bufs*.json"}, "Bufs_Mirror3.json", true},
		{"glob does not cross a slash", []string{"Bufs*.json"}, "StoryData/Bufs1.json", false},
		{"exact name", []string{"MainUIText.json"}, "MainUIText.json", true},
		{"second pattern matches", []string{"A/**", "B*.json"}, "Bx.json", true},
		{"malformed glob matches nothing", []string{"[a"}, "a", false},
		{"no patterns", nil, "BgmLyrics/a.json", false},
		{"bare /** has no directory", []string{"/**"}, "a.json", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MatchAny(tt.patterns, tt.file); got != tt.want {
				t.Errorf("MatchAny(%q, %q) = %v, want %v", tt.patterns, tt.file, got, tt.want)
			}
		})
	}
}

func TestValidPattern(t *testing.T) {
	for _, p := range []string{"BgmLyrics/**", "StoryData/S1*", "Bufs*.json", "MainUIText.json", "[a-c]x", "/**"} {
		if err := ValidPattern(p); err != nil {
			t.Errorf("ValidPattern(%q) = %v, want nil", p, err)
		}
	}
	for _, p := range []string{"[a", "StoryData/[", `a\`, "[]"} {
		if err := ValidPattern(p); err == nil {
			t.Errorf("ValidPattern(%q) = nil, want an error (MatchAny would match nothing)", p)
		}
	}
}

// Only a missing lct.yaml sends FindRoot up to the parent; any other error
// (here a symlink loop, which fails for root too) is returned, not skipped.
func TestFindRootStopsOnOtherErrors(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, "lang_dir: zh-TW\n")
	child := filepath.Join(root, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	loop := filepath.Join(child, FileName)
	if err := os.Symlink(FileName, loop); err != nil {
		t.Skipf("cannot create a symlink: %v", err)
	}
	got, err := FindRoot(child)
	if err == nil {
		t.Fatalf("FindRoot = %q, want the stat error of %s", got, loop)
	}
	if errors.Is(err, ErrRootNotFound) {
		t.Errorf("error = %v, want the underlying error, not %v", err, ErrRootNotFound)
	}
	if !strings.Contains(err.Error(), loop) {
		t.Errorf("error = %v, want one naming %s", err, loop)
	}
}
