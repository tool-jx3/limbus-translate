package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/config"
)

const (
	testToken     = "l20261029_n2rX1Jku6_9sBiu9Np24"
	testCreatedAt = "2026-10-29 02:55:23"
)

// writePackZip writes <dir>/<name> holding LocalizeTemp_<lang>/<files...>.
func writePackZip(t *testing.T, dir, name, lang string, files map[string]string) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for rel, content := range files {
		w, err := zw.Create("LocalizeTemp_" + lang + "/" + rel)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

// cdnDir lays out a fake CDN for testToken: <dir>/localize_{kr,en,jp}.zip.
// It returns <dir> and a file:// template that points into it.
func cdnDir(t *testing.T) (dir, template string) {
	t.Helper()
	root := t.TempDir()
	dir = filepath.Join(root, testToken)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, l := range []struct{ lang, prefix string }{{"kr", "KR_"}, {"en", "EN_"}, {"jp", "JP_"}} {
		writePackZip(t, dir, "localize_"+l.lang+".zip", l.lang, map[string]string{
			l.prefix + "A.json":                "\xef\xbb\xbf{\r\n}\r\n",
			"StoryData/" + l.prefix + "S.json": "{}\n",
		})
	}
	return dir, "file://" + root + "/{token}/localize_{lang}.zip"
}

func testCtx(t *testing.T, root, yaml string, jsonOut bool) (*cli.Context, *bytes.Buffer) {
	t.Helper()
	if yaml != "" {
		writeTestFile(t, root, "lct.yaml", yaml)
	}
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	return &cli.Context{Root: root, Config: cfg, JSON: jsonOut, Out: &out}, &out
}

// isUserError reports whether err is a cli.UserError, which cli maps to exit
// code 1 (a system error is 2).
func isUserError(err error) bool {
	return err != nil && reflect.TypeOf(err) == reflect.TypeOf(cli.UserError(""))
}

func TestFetchWithTokenFlags(t *testing.T) {
	_, template := cdnDir(t)
	root := t.TempDir()
	ctx, out := testCtx(t, root, "worktrees: { raw: .work/raw, tw: .work/tw }\n", true)
	dest := filepath.Join(t.TempDir(), "dl")

	// A leftover from an earlier fetch must not survive.
	writeTestFile(t, dest, "kr/KR_Stale.json", "{}")

	err := runFetch(ctx, []string{"--token", testToken, "--created-at", testCreatedAt, "--cdn-template", template, "--out", dest})
	if err != nil {
		t.Fatal(err)
	}
	var got struct{ Token, CreatedAt, Dir string }
	if err := json.Unmarshal(out.Bytes(), &struct {
		Token     *string `json:"token"`
		CreatedAt *string `json:"created_at"`
		Dir       *string `json:"dir"`
	}{&got.Token, &got.CreatedAt, &got.Dir}); err != nil {
		t.Fatalf("output is not the expected JSON: %v\n%s", err, out)
	}
	if got.Token != testToken || got.CreatedAt != testCreatedAt || got.Dir != dest {
		t.Errorf("output = %+v, want token %s, created_at %s, dir %s", got, testToken, testCreatedAt, dest)
	}

	for _, p := range []struct{ rel, want string }{
		{"kr/KR_A.json", "\xef\xbb\xbf{\r\n}\r\n"},
		{"kr/StoryData/KR_S.json", "{}\n"},
		{"en/EN_A.json", "\xef\xbb\xbf{\r\n}\r\n"},
		{"jp/StoryData/JP_S.json", "{}\n"},
	} {
		b, err := os.ReadFile(filepath.Join(dest, filepath.FromSlash(p.rel)))
		if err != nil || string(b) != p.want {
			t.Errorf("%s = %q, %v; want %q", p.rel, b, err, p.want)
		}
	}
	if _, err := os.Stat(filepath.Join(dest, "kr", "KR_Stale.json")); !os.IsNotExist(err) {
		t.Errorf("the old contents of kr/ must be replaced, stat err = %v", err)
	}
	entries, err := os.ReadDir(dest)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if strings.Join(names, ",") != "en,jp,kr" {
		t.Errorf("%s holds %v, want only en, jp and kr (no zips or staging directories)", dest, names)
	}
}

func TestFetchReadsStatusAPI(t *testing.T) {
	_, template := cdnDir(t)
	status := filepath.Join(t.TempDir(), "status.json")
	if err := os.WriteFile(status, []byte(`{"latest_token":{"created_at":"`+testCreatedAt+`","token":"`+testToken+`"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LIMBUS_CDN_TEMPLATE", template)

	t.Run("status-url flag", func(t *testing.T) {
		ctx, out := testCtx(t, t.TempDir(), "worktrees: { raw: .work/raw }\n", false)
		dest := filepath.Join(t.TempDir(), "dl")
		if err := runFetch(ctx, []string{"--status-url", "file://" + status, "--out", dest}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), testToken) {
			t.Errorf("output does not name the token:\n%s", out)
		}
		if _, err := os.Stat(filepath.Join(dest, "jp", "JP_A.json")); err != nil {
			t.Error(err)
		}
	})
	t.Run("LIMBUS_STATUS_URL", func(t *testing.T) {
		t.Setenv("LIMBUS_STATUS_URL", "file://"+status)
		ctx, _ := testCtx(t, t.TempDir(), "worktrees: { raw: .work/raw }\n", false)
		dest := filepath.Join(t.TempDir(), "dl")
		if err := runFetch(ctx, []string{"--out", dest}); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(dest, "en", "EN_A.json")); err != nil {
			t.Error(err)
		}
	})
}

func TestFetchTemplateFallsBackToConfig(t *testing.T) {
	_, template := cdnDir(t)
	t.Setenv("LIMBUS_CDN_TEMPLATE", "")
	ctx, _ := testCtx(t, t.TempDir(), "cdn_template: "+template+"\n", false)
	dest := filepath.Join(t.TempDir(), "dl")
	if err := runFetch(ctx, []string{"--token", testToken, "--created-at", testCreatedAt, "--out", dest}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "kr", "KR_A.json")); err != nil {
		t.Error(err)
	}
}

func TestFetchTemplatePrecedence(t *testing.T) {
	// --cdn-template beats LIMBUS_CDN_TEMPLATE, which beats lct.yaml.
	_, good := cdnDir(t)
	t.Setenv("LIMBUS_CDN_TEMPLATE", "file:///nonexistent/{lang}.zip")
	ctx, _ := testCtx(t, t.TempDir(), "cdn_template: file:///nonexistent/{lang}.zip\n", false)
	dest := filepath.Join(t.TempDir(), "dl")
	if err := runFetch(ctx, []string{"--token", testToken, "--created-at", testCreatedAt, "--cdn-template", good, "--out", dest}); err != nil {
		t.Fatal(err)
	}
}

func TestFetchFailureKeepsExistingOutput(t *testing.T) {
	cdn, template := cdnDir(t)
	if err := os.Remove(filepath.Join(cdn, "localize_jp.zip")); err != nil {
		t.Fatal(err)
	}
	ctx, _ := testCtx(t, t.TempDir(), "worktrees: { raw: .work/raw }\n", false)
	dest := filepath.Join(t.TempDir(), "dl")
	writeTestFile(t, dest, "kr/KR_Old.json", "old")

	err := runFetch(ctx, []string{"--token", testToken, "--created-at", testCreatedAt, "--cdn-template", template, "--out", dest})
	if err == nil {
		t.Fatal("a missing zip must be an error")
	}
	if b, err := os.ReadFile(filepath.Join(dest, "kr", "KR_Old.json")); err != nil || string(b) != "old" {
		t.Errorf("a failed fetch changed the earlier output: %q, %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(dest, "kr", "KR_A.json")); !os.IsNotExist(err) {
		t.Errorf("a failed fetch left a partial kr/: stat err = %v", err)
	}
	entries, _ := os.ReadDir(dest)
	if len(entries) != 1 {
		t.Errorf("a failed fetch left %d entries in %s, want only kr", len(entries), dest)
	}
}

func TestFetchUsageErrors(t *testing.T) {
	_, template := cdnDir(t)
	t.Setenv("LIMBUS_STATUS_URL", "")
	t.Setenv("LIMBUS_CDN_TEMPLATE", "")
	out := filepath.Join(t.TempDir(), "dl")
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no --out", []string{"--token", testToken, "--created-at", testCreatedAt, "--cdn-template", template}, "--out"},
		{"token without created-at", []string{"--token", testToken, "--cdn-template", template, "--out", out}, "--created-at"},
		{"created-at without token", []string{"--created-at", testCreatedAt, "--cdn-template", template, "--out", out}, "--token"},
		{"bad token", []string{"--token", "a/b", "--created-at", testCreatedAt, "--cdn-template", template, "--out", out}, "token"},
		{"no token and no status URL", []string{"--cdn-template", template, "--out", out}, "status"},
		{"template without {lang}", []string{"--token", testToken, "--created-at", testCreatedAt, "--cdn-template", "file:///x/{token}.zip", "--out", out}, "{lang}"},
		{"template without {token}", []string{"--token", testToken, "--created-at", testCreatedAt, "--cdn-template", "file:///x/{lang}.zip", "--out", out}, "{token}"},
		{"extra argument", []string{"--token", testToken, "--created-at", testCreatedAt, "--cdn-template", template, "--out", out, "stray"}, "stray"},
		{"unknown flag", []string{"--bogus"}, "bogus"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := testCtx(t, t.TempDir(), "worktrees: { raw: .work/raw }\n", false)
			err := runFetch(ctx, tc.args)
			if !isUserError(err) {
				t.Fatalf("err = %v, want a user error", err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %q, want it to mention %q", err, tc.want)
			}
			if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
				t.Errorf("a refused fetch created %s", out)
			}
		})
	}
}
