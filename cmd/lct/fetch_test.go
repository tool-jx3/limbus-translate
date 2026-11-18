package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/fetch"
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

// cdnCommon adds to the fake CDN of cdnDir the list of every file and the
// common files (path under fetch.LocalizeRoot to content), and returns the
// file:// template of single files. listed are the paths the list names;
// a path of files that is not listed is on the CDN and not in the list.
func cdnCommon(t *testing.T, cdn string, files map[string]string, listed ...string) (template string) {
	t.Helper()
	entries := []string{`"Assets/Resources_moved/Localize/kr/KR_A.json":{"Hash":"h","Size":1,"Crc":0}`}
	for _, rel := range listed {
		entries = append(entries, fmt.Sprintf(`%q:{"Hash":"h","Size":1,"Crc":0}`, fetch.LocalizeRoot+rel))
	}
	writeTestFile(t, cdn, fetch.PatchInfoPath, `{"Version":"0.1","EncryptKey":"SECRET-KEY","EncryptIV":"SECRET-IV","Files":{`+strings.Join(entries, ",")+`}}`)
	for rel, content := range files {
		writeTestFile(t, cdn, fetch.LocalizeRoot+rel, content)
	}
	return "file://" + filepath.Dir(cdn) + "/{token}/{path}"
}

var commonFiles = map[string]string{
	"RemoteLocalizeFileList.json":    "{\r\n  \"ui\": [\"BattleUIText\"]\r\n}",
	"etc/VoiceTable.json":            `{"prefixes":["battle_kill"],"voiceKeys":[]}`,
	"rpg-loc-file-list-a1c10p3.json": `{"ui":[{"file":"rpg-loc-ui-common"}]}`,
}

func TestFetchDownloadsTheCommonFiles(t *testing.T) {
	cdn, template := cdnDir(t)
	fileTemplate := cdnCommon(t, cdn, commonFiles, "RemoteLocalizeFileList.json", "etc/VoiceTable.json", "rpg-loc-file-list-a1c10p3.json")
	ctx, out := testCtx(t, t.TempDir(), "worktrees: { raw: .work/raw }\n", true)
	dest := filepath.Join(t.TempDir(), "dl")
	// A leftover from an earlier fetch must not survive.
	writeTestFile(t, dest, "common/Stale.json", "{}")

	err := runFetch(ctx, []string{"--token", testToken, "--created-at", testCreatedAt, "--cdn-template", template, "--cdn-file-template", fileTemplate, "--out", dest})
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Common []string `json:"common"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if want := []string{"RemoteLocalizeFileList.json", "etc/VoiceTable.json", "rpg-loc-file-list-a1c10p3.json"}; !slices.Equal(got.Common, want) {
		t.Errorf("common = %v, want %v", got.Common, want)
	}
	// Byte for byte, under common/ with the layout of the CDN.
	for rel, want := range commonFiles {
		if b, err := os.ReadFile(filepath.Join(dest, "common", filepath.FromSlash(rel))); err != nil || string(b) != want {
			t.Errorf("common/%s = %q, %v; want %q", rel, b, err, want)
		}
	}
	if _, err := os.Stat(filepath.Join(dest, "common", "Stale.json")); !os.IsNotExist(err) {
		t.Errorf("the old contents of common/ must be replaced, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "kr", "KR_A.json")); err != nil {
		t.Errorf("the packs are fetched as before: %v", err)
	}
	// The list of every file holds a key of the game: it is written nowhere.
	err = filepath.WalkDir(dest, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if b, _ := os.ReadFile(p); bytes.Contains(b, []byte("SECRET")) || strings.Contains(p, "PatchInfo") {
			t.Errorf("%s keeps the list of every file", p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "SECRET") {
		t.Errorf("the output shows the key:\n%s", out)
	}
}

func TestFetchWithoutAFileTemplateFetchesNoCommonFiles(t *testing.T) {
	_, template := cdnDir(t)
	t.Setenv("LIMBUS_CDN_FILE_TEMPLATE", "")
	ctx, out := testCtx(t, t.TempDir(), "worktrees: { raw: .work/raw }\n", true)
	dest := filepath.Join(t.TempDir(), "dl")
	writeTestFile(t, dest, "common/Stale.json", "{}")
	if err := runFetch(ctx, []string{"--token", testToken, "--created-at", testCreatedAt, "--cdn-template", template, "--out", dest}); err != nil {
		t.Fatal(err)
	}
	// What an earlier fetch left is not of this update.
	if _, err := os.Stat(filepath.Join(dest, "common")); !os.IsNotExist(err) {
		t.Errorf("common/ must be gone, stat err = %v", err)
	}
	if strings.Contains(out.String(), `"common"`) {
		t.Errorf("the output names common files:\n%s", out)
	}
}

// The file template is taken from where the pack template comes from, or from
// a place that outranks it (flag, then environment, then lct.yaml): a pack
// template given by hand names another CDN than the file template of lct.yaml.
func TestFetchFileTemplateGoesWithThePackTemplate(t *testing.T) {
	cdn, pack := cdnDir(t)
	good := cdnCommon(t, cdn, commonFiles, "etc/VoiceTable.json")
	bad := "file:///nonexistent/{token}/{path}"
	type src struct{ flag, env, config string }
	for name, c := range map[string]struct {
		pack, file src
		want       bool
	}{
		"both by flag":                    {src{flag: pack}, src{flag: good, env: bad, config: bad}, true},
		"both from the environment":       {src{env: pack}, src{env: good, config: bad}, true},
		"both from lct.yaml":              {src{config: pack}, src{config: good}, true},
		"file by flag, pack from config":  {src{config: pack}, src{flag: good, config: bad}, true},
		"file from env, pack from config": {src{config: pack}, src{env: good, config: bad}, true},
		"pack by flag, file from config":  {src{flag: pack}, src{config: bad}, false},
		"pack by flag, file from env":     {src{flag: pack}, src{env: bad}, false},
		"pack from env, file from config": {src{env: pack}, src{config: bad}, false},
	} {
		t.Setenv("LIMBUS_CDN_TEMPLATE", c.pack.env)
		t.Setenv("LIMBUS_CDN_FILE_TEMPLATE", c.file.env)
		yaml := "worktrees: { raw: .work/raw }\n"
		if c.pack.config != "" {
			yaml += "cdn_template: " + c.pack.config + "\n"
		}
		if c.file.config != "" {
			yaml += "cdn_file_template: " + c.file.config + "\n"
		}
		ctx, _ := testCtx(t, t.TempDir(), yaml, false)
		dest := filepath.Join(t.TempDir(), "dl")
		args := []string{"--token", testToken, "--created-at", testCreatedAt, "--out", dest}
		if c.pack.flag != "" {
			args = append(args, "--cdn-template", c.pack.flag)
		}
		if c.file.flag != "" {
			args = append(args, "--cdn-file-template", c.file.flag)
		}
		if err := runFetch(ctx, args); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		_, err := os.Stat(filepath.Join(dest, "common", "etc", "VoiceTable.json"))
		if got := err == nil; got != c.want {
			t.Errorf("%s: common files fetched = %v, want %v", name, got, c.want)
		}
	}
}

func TestFetchCommonFailureKeepsExistingOutput(t *testing.T) {
	cdn, template := cdnDir(t)
	// The list names a file the CDN has not.
	fileTemplate := cdnCommon(t, cdn, map[string]string{"etc/VoiceTable.json": "{}"}, "etc/VoiceTable.json", "RemoteLocalizeFileList.json")
	ctx, _ := testCtx(t, t.TempDir(), "worktrees: { raw: .work/raw }\n", false)
	dest := filepath.Join(t.TempDir(), "dl")
	writeTestFile(t, dest, "kr/KR_Old.json", "old")
	writeTestFile(t, dest, "common/Old.json", "old")

	err := runFetch(ctx, []string{"--token", testToken, "--created-at", testCreatedAt, "--cdn-template", template, "--cdn-file-template", fileTemplate, "--out", dest})
	if err == nil {
		t.Fatal("a common file that cannot be downloaded must be an error")
	}
	for _, rel := range []string{"kr/KR_Old.json", "common/Old.json"} {
		if b, err := os.ReadFile(filepath.Join(dest, filepath.FromSlash(rel))); err != nil || string(b) != "old" {
			t.Errorf("a failed fetch changed %s: %q, %v", rel, b, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dest, "kr", "KR_A.json")); !os.IsNotExist(err) {
		t.Errorf("a failed fetch left a partial kr/: stat err = %v", err)
	}
}

func TestFetchFileTemplateUsageErrors(t *testing.T) {
	_, template := cdnDir(t)
	ctx, _ := testCtx(t, t.TempDir(), "worktrees: { raw: .work/raw }\n", false)
	for _, bad := range []string{"file:///x/{token}/list.json", "file:///x/{path}"} {
		err := runFetch(ctx, []string{"--token", testToken, "--created-at", testCreatedAt, "--cdn-template", template, "--cdn-file-template", bad, "--out", t.TempDir()})
		if !isUserError(err) {
			t.Errorf("--cdn-file-template %s: err = %v, want a user error", bad, err)
		}
	}
	if err := runFetch(ctx, []string{"--check", "--cdn-file-template", "x", "--status-url", "file:///x"}); !isUserError(err) {
		t.Errorf("--check with --cdn-file-template: err = %v, want a user error", err)
	}
}
