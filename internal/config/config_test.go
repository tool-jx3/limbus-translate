package config

import (
	"os"
	"path/filepath"
	"slices"
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
	wantFont := Font{Path: "assets/fonts/SarasaGothicTC-Bold.ttf"}
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
		"paratranz": {b.Paratranz, Upstream{Repo: "user-unknown-0711/ParatranzUploader", Tag: "1793285016", Asset: "localize_files.zip"}},
		"ltm":       {b.LTM, Upstream{Repo: "LimbusTraditionalMandarin/storyline", Tag: "1792667238", Asset: "Hant.zip"}},
		"llc":       {b.LLC, Upstream{Repo: "LocalizeLimbusCompany/LocalizeLimbusCompany", Tag: "2026103002", Asset: "LimbusLocalize_2026103002.zip"}},
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
