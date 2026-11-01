package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
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

// statusURL writes a status API response for token and returns its file:// URL.
func statusURL(t *testing.T, token, createdAt string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "status.json")
	body := `{"latest_token":{"created_at":"` + createdAt + `","token":"` + token + `"}}`
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return "file://" + p
}

// TestFetchCheck: --check asks the status API and compares its token with the
// one of the snapshot on raw; it downloads nothing.
func TestFetchCheck(t *testing.T) {
	// A CDN that must not be asked for anything.
	var hits atomic.Int32
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Error(w, "nothing may be downloaded", http.StatusInternalServerError)
	}))
	t.Cleanup(cdn.Close)
	t.Setenv("LIMBUS_CDN_TEMPLATE", cdn.URL+"/{token}/localize_{lang}.zip")

	snapshotted := snapshotRepo(t)
	if _, err := runSnapshotJSON(t, snapshotted, "--from", extractedDir(t, "{}\n"), "--token", testToken, "--created-at", testCreatedAt); err != nil {
		t.Fatal(err)
	}

	const newToken, newCreatedAt = "l20261105_n3wT0ken_xYz", "2026-11-05 03:10:00"
	check := func(t *testing.T, root, status string) map[string]any {
		t.Helper()
		t.Setenv("LIMBUS_STATUS_URL", status)
		ctx, out := testCtx(t, root, "", true)
		if err := runFetch(ctx, []string{"--check"}); err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err := json.Unmarshal(out.Bytes(), &got); err != nil {
			t.Fatalf("output is not JSON: %v\n%s", err, out)
		}
		if len(got) != 3 {
			t.Errorf("output = %v, want only token, created_at and new", got)
		}
		return got
	}

	t.Run("the token raw holds", func(t *testing.T) {
		got := check(t, snapshotted, statusURL(t, testToken, testCreatedAt))
		if got["token"] != testToken || got["created_at"] != testCreatedAt || got["new"] != false {
			t.Errorf("output = %v, want token %s, created_at %s, new false", got, testToken, testCreatedAt)
		}
	})
	t.Run("another token", func(t *testing.T) {
		got := check(t, snapshotted, statusURL(t, newToken, newCreatedAt))
		if got["token"] != newToken || got["created_at"] != newCreatedAt || got["new"] != true {
			t.Errorf("output = %v, want token %s, created_at %s, new true", got, newToken, newCreatedAt)
		}
	})
	t.Run("raw has no snapshot yet", func(t *testing.T) {
		got := check(t, snapshotRepo(t), statusURL(t, testToken, testCreatedAt))
		if got["new"] != true {
			t.Errorf("output = %v, want new true", got)
		}
	})
	t.Run("--status-url", func(t *testing.T) {
		t.Setenv("LIMBUS_STATUS_URL", "")
		ctx, out := testCtx(t, snapshotted, "", true)
		if err := runFetch(ctx, []string{"--check", "--status-url", statusURL(t, newToken, newCreatedAt)}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), `"new": true`) {
			t.Errorf("output = %s, want new true", out)
		}
	})
	t.Run("text output", func(t *testing.T) {
		t.Setenv("LIMBUS_STATUS_URL", statusURL(t, newToken, newCreatedAt))
		ctx, out := testCtx(t, snapshotted, "", false)
		if err := runFetch(ctx, []string{"--check"}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), newToken) || !strings.HasPrefix(out.String(), "new:") {
			t.Errorf("output = %q, want the token after \"new:\"", out)
		}
		t.Setenv("LIMBUS_STATUS_URL", statusURL(t, testToken, testCreatedAt))
		ctx, out = testCtx(t, snapshotted, "", false)
		if err := runFetch(ctx, []string{"--check"}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), testToken) || !strings.HasPrefix(out.String(), "unchanged:") {
			t.Errorf("output = %q, want the token after \"unchanged:\"", out)
		}
	})

	t.Run("a status API that fails", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "secret-path", "status.json")
		t.Setenv("LIMBUS_STATUS_URL", "file://"+missing)
		ctx, _ := testCtx(t, snapshotted, "", true)
		err := runFetch(ctx, []string{"--check"})
		if err == nil || isUserError(err) {
			t.Fatalf("err = %v, want a system error", err)
		}
		if strings.Contains(err.Error(), "secret-path") {
			t.Errorf("err = %q holds the status URL, which is a secret in CI", err)
		}
	})

	t.Run("usage errors", func(t *testing.T) {
		status := statusURL(t, newToken, newCreatedAt)
		dest := filepath.Join(t.TempDir(), "dl")
		tests := []struct {
			name, status, yaml string
			args               []string
			want               string
		}{
			{"with --out", status, "", []string{"--check", "--out", dest}, "--out"},
			{"with --token", status, "", []string{"--check", "--token", testToken}, "--token"},
			{"with --created-at", status, "", []string{"--check", "--created-at", testCreatedAt}, "--created-at"},
			{"with --cdn-template", status, "", []string{"--check", "--cdn-template", "file:///x/{token}/{lang}.zip"}, "--cdn-template"},
			{"extra argument", status, "", []string{"--check", "stray"}, "stray"},
			{"no status API", "", "", []string{"--check"}, "status"},
			{"no raw worktree in lct.yaml", status, "cdn_template: file:///x/{token}/{lang}.zip\n", []string{"--check"}, "worktrees.raw"},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				t.Setenv("LIMBUS_STATUS_URL", tc.status)
				root := snapshotted
				if tc.yaml != "" {
					root = t.TempDir()
				}
				ctx, _ := testCtx(t, root, tc.yaml, true)
				err := runFetch(ctx, tc.args)
				if !isUserError(err) {
					t.Fatalf("err = %v, want a user error", err)
				}
				if !strings.Contains(err.Error(), tc.want) {
					t.Errorf("err = %q, want it to mention %q", err, tc.want)
				}
			})
		}
	})

	if n := hits.Load(); n != 0 {
		t.Errorf("--check asked the CDN %d times, want none", n)
	}
	if _, err := os.Stat(filepath.Join(snapshotted, ".work", "fetch")); !os.IsNotExist(err) {
		t.Errorf("--check made a download directory, stat err = %v", err)
	}
}
