// Package raw keeps the original text of the game on the raw branch: one
// commit per game update holding byte-exact copies of the official KR, EN and
// JP packs, so that git history shows how the original text changed.
//
// Layout of the raw worktree:
//
//	kr/ en/ jp/        the extracted packs, as they came out of the zips
//	meta/source.json   what the snapshot was made from (see Source)
//	.gitattributes     "* -text", so git never changes a line ending
package raw

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/gitx"
)

// TagPrefix starts the name of every snapshot tag.
const TagPrefix = "raw-"

const (
	sourceFile     = "meta/source.json"
	attributesFile = ".gitattributes"
	attributes     = "* -text\n"
)

// langs are the pack directories of the worktree, in the order they are written.
var langs = []string{"kr", "en", "jp"}

// ErrInvalidInput marks a Snapshot call refused because of its arguments (the
// token, the date or the pack directories), before anything was written.
var ErrInvalidInput = errors.New("invalid snapshot input")

func invalid(format string, args ...any) error {
	return fmt.Errorf("raw: %w: %s", ErrInvalidInput, fmt.Sprintf(format, args...))
}

// Source is the content of meta/source.json.
type Source struct {
	// Tag is the git tag of this snapshot, such as "raw-20261029".
	Tag string `json:"tag"`
	// Token and CreatedAt are the CDN token of the game update and the
	// creation time the status API reported for it.
	Token     string `json:"token"`
	CreatedAt string `json:"created_at"`
	// FetchedAt is when the packs were downloaded (RFC 3339, UTC).
	FetchedAt string `json:"fetched_at"`
	// Files maps every file of the snapshot ("kr/KR_X.json") to the lowercase
	// hex sha256 of its bytes.
	Files map[string]string `json:"files"`
}

// LastSource reads meta/source.json of the raw worktree. It returns nil, nil
// when the file does not exist, which is the case before the first snapshot.
func LastSource(worktree string) (*Source, error) {
	data, err := os.ReadFile(filepath.Join(worktree, filepath.FromSlash(sourceFile)))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var s Source
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("raw: parse %s: %w", sourceFile, err)
	}
	return &s, nil
}

// tagDay returns the date of createdAt as YYYYMMDD. createdAt must start with
// a YYYY-MM-DD date, optionally followed by a space or "T" and a time.
func tagDay(createdAt string) (string, error) {
	day := createdAt
	if i := strings.IndexAny(day, " T"); i >= 0 {
		day = day[:i]
	}
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return "", fmt.Errorf("created_at %q does not start with a YYYY-MM-DD date", createdAt)
	}
	return t.Format("20060102"), nil
}

// NextTag returns the name for a new snapshot tag: "raw-YYYYMMDD" for the date
// of createdAt, or "raw-YYYYMMDD.2", ".3", ... when existing already holds the
// earlier names of that day. createdAt must be valid (Snapshot checks it); an
// invalid one gives a tag without a date.
func NextTag(existing []string, createdAt string) string {
	day, _ := tagDay(createdAt)
	taken := make(map[string]bool, len(existing))
	for _, name := range existing {
		taken[name] = true
	}
	base := TagPrefix + day
	tag := base
	for n := 2; taken[tag]; n++ {
		tag = fmt.Sprintf("%s.%d", base, n)
	}
	return tag
}

// Snapshot writes the packs into the raw worktree, commits and tags them. packs
// maps "kr", "en" and "jp" to the directories the packs were extracted to; src
// supplies Token, CreatedAt and optionally FetchedAt (default: now). Tag and
// Files of src are ignored, as they are computed.
//
// Each of kr/, en/ and jp/ becomes a byte-exact copy of its pack, so files
// that are gone from the pack are removed. The commit message is
// "raw: <token> (<created_at>)". If the worktree's meta/source.json already
// has src.Token nothing is written: changed is false and tag is the existing
// snapshot's tag.
//
// Bad arguments are reported before the worktree is touched and wrap
// ErrInvalidInput. The worktree must be a clean checkout of the raw branch
// (gitx.EnsureWorktree); a failure while writing leaves it for git to reset.
func Snapshot(worktree string, packs map[string]string, src Source) (tag string, changed bool, err error) {
	if src.Token == "" {
		return "", false, invalid("no token")
	}
	if _, err := tagDay(src.CreatedAt); err != nil {
		return "", false, invalid("%v", err)
	}
	lists, err := listPacks(worktree, packs)
	if err != nil {
		return "", false, err
	}

	last, err := LastSource(worktree)
	if err != nil {
		return "", false, err
	}
	if last != nil && last.Token == src.Token {
		return last.Tag, false, nil
	}

	existing, err := gitx.Tags(worktree, TagPrefix)
	if err != nil {
		return "", false, err
	}
	tag = NextTag(existing, src.CreatedAt)

	files := make(map[string]string)
	for _, lang := range langs {
		dst := filepath.Join(worktree, lang)
		if err := os.RemoveAll(dst); err != nil {
			return "", false, err
		}
		for _, rel := range lists[lang] {
			sum, err := copyFile(filepath.Join(packs[lang], filepath.FromSlash(rel)), filepath.Join(dst, filepath.FromSlash(rel)))
			if err != nil {
				return "", false, err
			}
			files[lang+"/"+rel] = sum
		}
	}
	if err := os.WriteFile(filepath.Join(worktree, attributesFile), []byte(attributes), 0o644); err != nil {
		return "", false, err
	}

	if src.FetchedAt == "" {
		src.FetchedAt = time.Now().UTC().Format(time.RFC3339)
	}
	// meta/source.json is written last: it is what marks a snapshot as done.
	if err := writeSource(worktree, Source{Tag: tag, Token: src.Token, CreatedAt: src.CreatedAt, FetchedAt: src.FetchedAt, Files: files}); err != nil {
		return "", false, err
	}

	_, committed, err := gitx.CommitAll(worktree, fmt.Sprintf("raw: %s (%s)", src.Token, src.CreatedAt))
	if err != nil {
		return "", false, err
	}
	if !committed {
		return "", false, errors.New("raw: nothing to commit after writing the snapshot")
	}
	if err := gitx.Tag(worktree, tag); err != nil {
		return tag, true, fmt.Errorf("raw: the snapshot is committed but not tagged: %w", err)
	}
	return tag, true, nil
}

// listPacks checks the arguments of Snapshot and lists the files of each pack
// as slash-separated paths relative to the pack directory.
func listPacks(worktree string, packs map[string]string) (map[string][]string, error) {
	for lang := range packs {
		if !isLang(lang) {
			return nil, invalid("unknown pack %q (want kr, en and jp)", lang)
		}
	}
	lists := make(map[string][]string, len(langs))
	for _, lang := range langs {
		dir, ok := packs[lang]
		if !ok {
			return nil, invalid("no %s pack", lang)
		}
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			return nil, invalid("the %s pack %s is not a directory", lang, dir)
		}
		if overlaps(dir, worktree) {
			return nil, invalid("the %s pack %s and the worktree %s contain each other", lang, dir, worktree)
		}
		files, err := listFiles(dir)
		if err != nil {
			return nil, fmt.Errorf("raw: the %s pack: %w", lang, err)
		}
		if len(files) == 0 {
			return nil, invalid("the %s pack %s has no files", lang, dir)
		}
		lists[lang] = files
	}
	return lists, nil
}

func isLang(s string) bool {
	for _, l := range langs {
		if s == l {
			return true
		}
	}
	return false
}

// listFiles lists the files under dir. Anything but a regular file (a link, a
// device) is refused: it cannot be copied byte for byte.
func listFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return invalid("%s is not a regular file", p)
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	return files, err
}

// overlaps reports whether a and b are the same directory or one is inside
// the other.
func overlaps(a, b string) bool {
	a, b = realPath(a), realPath(b)
	inside := func(p, parent string) bool {
		rel, err := filepath.Rel(parent, p)
		return err == nil && filepath.IsLocal(rel)
	}
	return inside(a, b) || inside(b, a)
}

func realPath(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

// copyFile copies src to dst byte for byte and returns the sha256 of the bytes.
func copyFile(src, dst string) (string, error) {
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return "", err
	}
	sum := sha256.New()
	_, err = io.Copy(io.MultiWriter(out, sum), in)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return "", fmt.Errorf("copy %s: %w", src, err)
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

// writeSource writes meta/source.json: indented JSON with the files sorted by
// name and a final newline.
func writeSource(worktree string, s Source) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s); err != nil {
		return err
	}
	path := filepath.Join(worktree, filepath.FromSlash(sourceFile))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
