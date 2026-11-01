// Package fetch downloads the official localization packs: it reads the status
// API for the latest token, builds the CDN URLs, downloads the zips and
// extracts them byte for byte.
package fetch

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Timeouts of the two kinds of request. They cover the whole exchange,
// including reading the body. Both clients use the default transport, which
// honors the proxy environment variables.
var (
	statusClient = &http.Client{Timeout: 30 * time.Second}
	packClient   = &http.Client{Timeout: 10 * time.Minute}
)

// maxStatusBytes bounds how much of a status response is read.
const maxStatusBytes = 1 << 20

var tokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// Status is the latest game update the status API reports.
type Status struct {
	Token     string `json:"token"`
	CreatedAt string `json:"created_at"`
}

// CheckToken reports whether token is usable in a CDN URL: letters, digits,
// "_" and "-" only.
func CheckToken(token string) error {
	if !tokenPattern.MatchString(token) {
		return fmt.Errorf("invalid token %q: want only letters, digits, '_' and '-'", token)
	}
	return nil
}

// ParseStatus reads the status API response, which looks like
// {"latest_token":{"created_at":"2026-10-29 02:55:23","token":"l20261029_..."}}.
func ParseStatus(body []byte) (Status, error) {
	var doc struct {
		LatestToken *Status `json:"latest_token"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return Status{}, fmt.Errorf("parse status: %w", err)
	}
	if doc.LatestToken == nil {
		return Status{}, errors.New("parse status: no latest_token")
	}
	st := *doc.LatestToken
	if err := CheckToken(st.Token); err != nil {
		return Status{}, fmt.Errorf("parse status: %w", err)
	}
	if st.CreatedAt == "" {
		return Status{}, errors.New("parse status: latest_token has no created_at")
	}
	if err := checkCreatedAt(st.CreatedAt); err != nil {
		return Status{}, fmt.Errorf("parse status: %w", err)
	}
	return st, nil
}

// createdAtPattern is the shape of created_at: a date, and after it a space or
// "T" and a time, as "2026-10-29 02:55:23" or "2026-10-29T02:55:23Z". It is the
// form a raw tag takes its date from, and nothing more: the value goes to the
// outputs of the workflow, one to a line, and into commit messages.
var createdAtPattern = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})(?:[ T]\d{2}:\d{2}(?::\d{2}(?:\.\d+)?)?(?:Z|[+-]\d{2}:?\d{2})?)?$`)

// checkCreatedAt reports whether createdAt is a real date, alone or followed
// by a time (see createdAtPattern).
func checkCreatedAt(createdAt string) error {
	m := createdAtPattern.FindStringSubmatch(createdAt)
	if m == nil {
		return fmt.Errorf("invalid created_at %q: want YYYY-MM-DD, optionally followed by a space or T and a time", createdAt)
	}
	if _, err := time.Parse("2006-01-02", m[1]); err != nil {
		return fmt.Errorf("invalid created_at %q: %s is not a day", createdAt, m[1])
	}
	return nil
}

// FetchStatus reads and parses the status API at statusURL (https://, http://
// or file://). statusURL is a secret in CI, so a failed request is reported
// only as "status request failed: <kind>" (see requestFailure), which holds no
// part of it.
func FetchStatus(ctx context.Context, statusURL string) (Status, error) {
	body, err := open(ctx, statusClient, statusURL)
	if err != nil {
		return Status{}, requestFailure(err)
	}
	defer body.Close()
	data, err := io.ReadAll(io.LimitReader(body, maxStatusBytes))
	if err != nil {
		return Status{}, requestFailure(err)
	}
	return ParseStatus(data)
}

// requestFailure reports a failed status request by its kind alone: the
// errors of the network and file layers carry the host, the address or the
// path, so none of their text is kept.
func requestFailure(err error) error {
	var kind string
	var httpErr *httpStatusError
	var netErr net.Error
	switch {
	case errors.As(err, &httpErr):
		kind = fmt.Sprintf("HTTP %d", httpErr.code)
	case errors.Is(err, context.DeadlineExceeded) || errors.As(err, &netErr) && netErr.Timeout():
		kind = "timeout"
	case errors.Is(err, context.Canceled):
		kind = "canceled"
	case errors.Is(err, errMalformedURL):
		kind = "malformed URL"
	case errors.Is(err, errUnsupportedScheme):
		kind = "unsupported URL scheme"
	case errors.Is(err, fs.ErrNotExist):
		kind = "no such file"
	case errors.As(err, new(*fs.PathError)):
		kind = "file error"
	default:
		kind = "connection error"
	}
	return fmt.Errorf("status request failed: %s", kind)
}

var (
	errMalformedURL      = errors.New("malformed URL")
	errUnsupportedScheme = errors.New("unsupported URL scheme")
)

// httpStatusError is a response that is not 2xx.
type httpStatusError struct {
	code   int
	status string // such as "404 Not Found"
}

func (e *httpStatusError) Error() string { return "HTTP " + e.status }

// PackURL fills the CDN template: {token} becomes token and {lang} becomes
// lang ("kr", "en" or "jp").
func PackURL(template, token, lang string) string {
	return strings.NewReplacer("{token}", token, "{lang}", lang).Replace(template)
}

// Download saves rawURL (https://, http:// or file://) to dest, creating the
// parent directory. The data goes to dest+".part" first, so dest only ever
// holds a complete download. A non-2xx response or an empty body is an error.
func Download(ctx context.Context, rawURL, dest string) error {
	body, err := open(ctx, packClient, rawURL)
	if err != nil {
		return fmt.Errorf("download %s: %w", rawURL, err)
	}
	defer body.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	part := dest + ".part"
	n, err := writeFile(part, body)
	if err == nil && n == 0 {
		err = errors.New("empty response")
	}
	if err != nil {
		os.Remove(part)
		return fmt.Errorf("download %s: %w", rawURL, unwrapURLError(err))
	}
	if err := os.Rename(part, dest); err != nil {
		os.Remove(part)
		return err
	}
	return nil
}

// open starts reading rawURL. file:// URLs read the local file. Errors do not
// repeat the URL; callers add the context they are willing to show.
func open(ctx context.Context, client *http.Client, rawURL string) (io.ReadCloser, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, errMalformedURL
	}
	switch u.Scheme {
	case "file":
		return os.Open(u.Path)
	case "http", "https":
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, errMalformedURL
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, unwrapURLError(err)
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			resp.Body.Close()
			return nil, &httpStatusError{code: resp.StatusCode, status: resp.Status}
		}
		return resp.Body, nil
	default:
		return nil, fmt.Errorf("%w %q", errUnsupportedScheme, u.Scheme)
	}
}

// unwrapURLError drops the *url.Error wrapper net/http adds, which carries
// the full request URL.
func unwrapURLError(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		return ue.Err
	}
	return err
}

// writeFile creates path and copies r into it, returning the bytes written.
func writeFile(path string, r io.Reader) (int64, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(f, r)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return n, err
}

// ExtractZip extracts the zip at zipPath into destDir, keeping the layout of
// the archive and the bytes of every file. Existing files in destDir are kept
// unless the archive overwrites them.
func ExtractZip(zipPath, destDir string) error {
	return extract(zipPath, destDir, false)
}

// ExtractPack extracts an official pack zip into destDir. Such a zip holds
// one top-level directory (LocalizeTemp_kr/, ...) that is dropped, so
// LocalizeTemp_kr/StoryData/KR_S.json becomes destDir/StoryData/KR_S.json.
// An archive that is empty, has files outside a single top-level directory, or
// holds a path that would leave destDir is an error. Existing files in destDir
// are kept unless the archive overwrites them.
func ExtractPack(zipPath, destDir string) error {
	return extract(zipPath, destDir, true)
}

type zipEntry struct {
	file *zip.File
	name string // slash-separated, relative to the destination
}

func extract(zipPath, destDir string, stripTop bool) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("extract %s: %w", zipPath, err)
	}
	defer zr.Close()

	entries, err := zipEntries(zr.File, stripTop)
	if err != nil {
		return fmt.Errorf("extract %s: %w", zipPath, err)
	}
	for _, e := range entries {
		if err := extractFile(e.file, filepath.Join(destDir, filepath.FromSlash(e.name))); err != nil {
			return fmt.Errorf("extract %s: %s: %w", zipPath, e.file.Name, err)
		}
	}
	return nil
}

// zipEntries lists the files of an archive with their destination names,
// after checking that every name stays inside the destination.
func zipEntries(files []*zip.File, stripTop bool) ([]zipEntry, error) {
	var entries []zipEntry
	for _, f := range files {
		// Directories are created on demand. Backslashes count as
		// separators so a Windows-made zip cannot slip past the checks.
		name := strings.ReplaceAll(f.Name, `\`, "/")
		if strings.HasSuffix(name, "/") {
			continue
		}
		if f.Mode()&fs.ModeSymlink != 0 {
			return nil, fmt.Errorf("%s is a symbolic link", f.Name)
		}
		entries = append(entries, zipEntry{f, name})
	}
	if len(entries) == 0 {
		return nil, errors.New("the archive has no files")
	}

	if stripTop {
		top := ""
		for i := range entries {
			first, rest, ok := strings.Cut(entries[i].name, "/")
			if !ok || first == "" {
				return nil, fmt.Errorf("%s is not inside a top-level directory", entries[i].file.Name)
			}
			if i == 0 {
				top = first
			} else if first != top {
				return nil, fmt.Errorf("more than one top-level directory (%q and %q)", top, first)
			}
			entries[i].name = rest
		}
	}

	seen := make(map[string]bool, len(entries))
	for _, e := range entries {
		local := filepath.FromSlash(e.name)
		if !filepath.IsLocal(local) {
			return nil, fmt.Errorf("%s would be written outside the destination", e.file.Name)
		}
		key := filepath.Clean(local)
		if seen[key] {
			return nil, fmt.Errorf("%s appears twice", e.file.Name)
		}
		seen[key] = true
	}
	return entries, nil
}

func extractFile(f *zip.File, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	// Reading to the end also verifies the CRC recorded in the archive.
	_, err = writeFile(dest, rc)
	return err
}
