package fetch

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleStatus = `{"latest_token":{"created_at":"2026-10-29 02:55:23","f_token":"f20261029_FMWPLYWVe3fbkNrw12En","token":"l20261029_n2rX1Jku6_9sBiu9Np24"}}`

func TestParseStatus(t *testing.T) {
	got, err := ParseStatus([]byte(sampleStatus))
	if err != nil {
		t.Fatal(err)
	}
	want := Status{Token: "l20261029_n2rX1Jku6_9sBiu9Np24", CreatedAt: "2026-10-29 02:55:23"}
	if got != want {
		t.Errorf("ParseStatus = %+v, want %+v", got, want)
	}

	bad := map[string]string{
		"slash in token":       `{"latest_token":{"created_at":"2026-10-29 02:55:23","token":"l2026/../x"}}`,
		"space in token":       `{"latest_token":{"created_at":"2026-10-29 02:55:23","token":"l2026 x"}}`,
		"empty token":          `{"latest_token":{"created_at":"2026-10-29 02:55:23","token":""}}`,
		"missing created_at":   `{"latest_token":{"token":"l20261029"}}`,
		"missing latest_token": `{"other":{}}`,
		"not json":             `<html>maintenance</html>`,
	}
	for name, body := range bad {
		t.Run(name, func(t *testing.T) {
			if got, err := ParseStatus([]byte(body)); err == nil {
				t.Errorf("ParseStatus(%s) = %+v, want an error", body, got)
			}
		})
	}
}

func TestPackURL(t *testing.T) {
	const template = "https://downloadcommon.limbuscompanycdn.org/{token}/Assets/LocalizePatch/localize_{lang}.zip"
	got := PackURL(template, "l20261029_n2rX1Jku6_9sBiu9Np24", "kr")
	want := "https://downloadcommon.limbuscompanycdn.org/l20261029_n2rX1Jku6_9sBiu9Np24/Assets/LocalizePatch/localize_kr.zip"
	if got != want {
		t.Errorf("PackURL = %q, want %q", got, want)
	}
}

func TestCheckToken(t *testing.T) {
	if err := CheckToken("l20261029_n2rX1Jku6_9sBiu9Np24"); err != nil {
		t.Errorf("a real token is rejected: %v", err)
	}
	for _, bad := range []string{"", "a/b", "a b", "a?b=c", "../x", "tok\n"} {
		if err := CheckToken(bad); err == nil {
			t.Errorf("CheckToken(%q) = nil, want an error", bad)
		}
	}
}

// writeZip writes a zip with the given entries (name -> content) to a new file
// and returns its path. Names ending in "/" become directory entries.
func writeZip(t *testing.T, entries [][2]string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		w, err := zw.Create(e[0])
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(e[1])); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "pack.zip")
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestExtractPack(t *testing.T) {
	const bomCRLF = "\xef\xbb\xbf{\r\n  \"dataList\": []\r\n}\r\n"
	zipPath := writeZip(t, [][2]string{
		{"LocalizeTemp_kr/KR_A.json", bomCRLF},
		{"LocalizeTemp_kr/StoryData/KR_S.json", "{}\n"},
	})
	dest := filepath.Join(t.TempDir(), "kr")
	if err := ExtractPack(zipPath, dest); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dest, "KR_A.json")); got != bomCRLF {
		t.Errorf("KR_A.json = %q, want the BOM and CRLF kept: %q", got, bomCRLF)
	}
	if got := readFile(t, filepath.Join(dest, "StoryData", "KR_S.json")); got != "{}\n" {
		t.Errorf("StoryData/KR_S.json = %q", got)
	}
	if _, err := os.Stat(filepath.Join(dest, "LocalizeTemp_kr")); !os.IsNotExist(err) {
		t.Errorf("the top-level directory must be stripped, stat err = %v", err)
	}
}

func TestExtractPackRejectsOddZips(t *testing.T) {
	tests := map[string][][2]string{
		"no files":            {},
		"file without a dir":  {{"KR_A.json", "{}"}},
		"two top-level dirs":  {{"LocalizeTemp_kr/KR_A.json", "{}"}, {"Other/KR_B.json", "{}"}},
		"parent traversal":    {{"LocalizeTemp_kr/../../evil.json", "{}"}},
		"backslash traversal": {{`LocalizeTemp_kr\..\..\evil.json`, "{}"}},
		"absolute path":       {{"/LocalizeTemp_kr/KR_A.json", "{}"}},
		"duplicate entry":     {{"LocalizeTemp_kr/KR_A.json", "{}"}, {"LocalizeTemp_kr/KR_A.json", "{}"}},
	}
	for name, entries := range tests {
		t.Run(name, func(t *testing.T) {
			parent := t.TempDir()
			dest := filepath.Join(parent, "out", "kr")
			if err := ExtractPack(writeZip(t, entries), dest); err == nil {
				t.Error("ExtractPack succeeded, want an error")
			}
			if _, err := os.Stat(filepath.Join(parent, "evil.json")); !os.IsNotExist(err) {
				t.Errorf("a file escaped the destination, stat err = %v", err)
			}
		})
	}
}

func TestExtractZipKeepsLayout(t *testing.T) {
	zipPath := writeZip(t, [][2]string{
		{"Top/a.txt", "a"},
		{"b.txt", "b"},
		{"dir/", ""},
	})
	dest := t.TempDir()
	if err := ExtractZip(zipPath, dest); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dest, "Top", "a.txt")); got != "a" {
		t.Errorf("Top/a.txt = %q", got)
	}
	if got := readFile(t, filepath.Join(dest, "b.txt")); got != "b" {
		t.Errorf("b.txt = %q", got)
	}
}

func TestExtractZipRejectsTraversal(t *testing.T) {
	parent := t.TempDir()
	zipPath := writeZip(t, [][2]string{{"../evil.txt", "x"}})
	if err := ExtractZip(zipPath, filepath.Join(parent, "out")); err == nil {
		t.Error("ExtractZip succeeded, want an error")
	}
	if _, err := os.Stat(filepath.Join(parent, "evil.txt")); !os.IsNotExist(err) {
		t.Errorf("a file escaped the destination, stat err = %v", err)
	}
}

func TestDownload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok.zip":
			w.Write([]byte("zip bytes"))
		case "/empty.zip":
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	ctx := context.Background()
	dest := filepath.Join(t.TempDir(), "sub", "ok.zip")
	if err := Download(ctx, srv.URL+"/ok.zip", dest); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, dest); got != "zip bytes" {
		t.Errorf("downloaded %q", got)
	}

	for _, path := range []string{"/missing.zip", "/empty.zip"} {
		dest := filepath.Join(t.TempDir(), "x.zip")
		if err := Download(ctx, srv.URL+path, dest); err == nil {
			t.Errorf("Download(%s) succeeded, want an error", path)
		}
		if _, err := os.Stat(dest); !os.IsNotExist(err) {
			t.Errorf("Download(%s) left %s behind, stat err = %v", path, dest, err)
		}
		if _, err := os.Stat(dest + ".part"); !os.IsNotExist(err) {
			t.Errorf("Download(%s) left a .part file behind, stat err = %v", path, err)
		}
	}
}

func TestDownloadFileURL(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src.zip")
	if err := os.WriteFile(src, []byte("local zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "dest.zip")
	if err := Download(context.Background(), "file://"+src, dest); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, dest); got != "local zip" {
		t.Errorf("downloaded %q", got)
	}

	empty := filepath.Join(t.TempDir(), "empty.zip")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Download(context.Background(), "file://"+empty, filepath.Join(t.TempDir(), "x.zip")); err == nil {
		t.Error("an empty file must be an error")
	}
	if err := Download(context.Background(), "ftp://example.com/x.zip", filepath.Join(t.TempDir(), "x.zip")); err == nil {
		t.Error("an unsupported scheme must be an error")
	}
}

func TestFetchStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/secret-path/status" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(sampleStatus))
	}))
	defer srv.Close()

	got, err := FetchStatus(context.Background(), srv.URL+"/secret-path/status")
	if err != nil {
		t.Fatal(err)
	}
	if got.Token != "l20261029_n2rX1Jku6_9sBiu9Np24" || got.CreatedAt != "2026-10-29 02:55:23" {
		t.Errorf("FetchStatus = %+v", got)
	}

	// The status URL is a secret in CI, so no error may echo it.
	_, err = FetchStatus(context.Background(), srv.URL+"/secret-path/missing")
	if err == nil {
		t.Fatal("a 404 must be an error")
	}
	if strings.Contains(err.Error(), "secret-path") {
		t.Errorf("error leaks the status URL: %v", err)
	}
	srv.Close()
	_, err = FetchStatus(context.Background(), srv.URL+"/secret-path/status")
	if err == nil {
		t.Fatal("a closed server must be an error")
	}
	if strings.Contains(err.Error(), "secret-path") {
		t.Errorf("error leaks the status URL: %v", err)
	}
}

func TestFetchStatusFileURL(t *testing.T) {
	p := filepath.Join(t.TempDir(), "status.json")
	if err := os.WriteFile(p, []byte(sampleStatus), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := FetchStatus(context.Background(), "file://"+p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Token != "l20261029_n2rX1Jku6_9sBiu9Np24" {
		t.Errorf("FetchStatus = %+v", got)
	}
}
