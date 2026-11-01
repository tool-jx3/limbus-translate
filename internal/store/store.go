// Package store persists the translation state kept on the tw branch: one
// record per translated unit (.meta/units/<file>.jsonl) and the queue of units
// still to translate (.meta/pending.jsonl), plus the lease and lock that let
// several processes take work from the queue without overlapping.
//
// Every file is written atomically and byte-for-byte deterministically: the
// same data always produces the same bytes, whatever order it was built in.
package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// Where a translation came from (Record.Origin).
const (
	OriginParatranz = "paratranz" // adopted from the Paratranz translation
	OriginClaude    = "claude"    // translated by a model
	OriginHuman     = "human"     // written or adjusted by a person
	OriginCopy      = "copy"      // identifier, or identical in all languages
	OriginFallback  = "fallback"  // translation failed; EN (or KR) text stands in
)

// Origins lists every value Record.Origin can take.
var Origins = []string{OriginParatranz, OriginClaude, OriginHuman, OriginCopy, OriginFallback}

// FlagNeedsReview marks a unit a person should look at.
const FlagNeedsReview = "needs_review"

const (
	metaDirName   = ".meta"
	unitsDirName  = "units"
	pendingName   = "pending.jsonl"
	lockName      = ".lock"
	gitignoreName = ".gitignore"
	recordExt     = ".jsonl"
)

// Record is the stored state of one translated unit: one line of a
// .meta/units/<file>.jsonl file. Key is "<file>|<JSON pointer>".
type Record struct {
	Key    string   `json:"key"`
	KRSha  string   `json:"kr_sha"` // sha256 of the KR text the translation was made from
	ZH     string   `json:"zh"`
	Origin string   `json:"origin"` // one of the Origin constants
	Since  string   `json:"since"`  // raw tag the record was written for
	Rules  string   `json:"rules,omitempty"`
	Model  string   `json:"model,omitempty"`
	Batch  string   `json:"batch,omitempty"`
	Flags  []string `json:"flags,omitempty"`
}

// PendingItem is a unit waiting to be translated: one line of
// .meta/pending.jsonl. Reason is one of "new", "kr-changed",
// "adjustment-expired", "paratranz-repair", "requeue" or "structural-fail".
// Hint is the reason a person gave when sending the unit back (lct requeue);
// the translator is shown it and fixes the unit accordingly.
type PendingItem struct {
	Key      string            `json:"key"`
	Reason   string            `json:"reason"`
	PrevKR   string            `json:"prev_kr,omitempty"`
	PrevZH   string            `json:"prev_zh,omitempty"`
	Draft    string            `json:"draft,omitempty"`
	Hint     string            `json:"hint,omitempty"`
	Refs     map[string]string `json:"refs,omitempty"`
	Attempts int               `json:"attempts,omitempty"`
	Flags    []string          `json:"flags,omitempty"`
	Lease    *Lease            `json:"lease,omitempty"`
}

// Lease reserves a pending item for a batch until Until.
type Lease struct {
	Batch string
	Until time.Time
}

// leaseJSON is Lease on disk: Until is RFC 3339 in UTC.
type leaseJSON struct {
	Batch string `json:"batch"`
	Until string `json:"until"`
}

// MarshalJSON writes the lease with Until in UTC.
func (l Lease) MarshalJSON() ([]byte, error) {
	return json.Marshal(leaseJSON{Batch: l.Batch, Until: l.Until.UTC().Format(time.RFC3339Nano)})
}

// UnmarshalJSON reads what MarshalJSON writes.
func (l *Lease) UnmarshalJSON(data []byte) error {
	var w leaseJSON
	if err := decodeStrict(data, &w); err != nil {
		return err
	}
	until, err := time.Parse(time.RFC3339Nano, w.Until)
	if err != nil {
		return fmt.Errorf("lease until: %w", err)
	}
	*l = Lease{Batch: w.Batch, Until: until.UTC()}
	return nil
}

// LeaseFree reports whether item may be handed out at now: it has no lease,
// or its lease has run out.
func LeaseFree(item PendingItem, now time.Time) bool {
	return item.Lease == nil || !item.Lease.Until.After(now)
}

// Store reads and writes the state under one tw worktree.
type Store struct {
	root string
}

// Open returns the store of the tw worktree at twRoot. It touches nothing on
// disk; a worktree without .meta/ simply has no state yet.
func Open(twRoot string) *Store {
	return &Store{root: twRoot}
}

func (s *Store) metaDir() string     { return filepath.Join(s.root, metaDirName) }
func (s *Store) unitsDir() string    { return filepath.Join(s.metaDir(), unitsDirName) }
func (s *Store) pendingPath() string { return filepath.Join(s.metaDir(), pendingName) }

// recordPath is the file holding the records of one pack file. file may
// contain "/" (e.g. "StoryData/S1.json").
func (s *Store) recordPath(file string) string {
	return filepath.Join(s.unitsDir(), filepath.FromSlash(file)+recordExt)
}

// LoadRecords reads every record, keyed by unit key. A worktree without
// state yields an empty map. A malformed line, a duplicate key, or a key
// filed under the wrong file is an error naming the file and line.
func (s *Store) LoadRecords() (map[string]Record, error) {
	recs := map[string]Record{}
	err := filepath.WalkDir(s.unitsDir(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && path == s.unitsDir() {
				return nil
			}
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, recordExt) {
			return nil
		}
		rel, err := filepath.Rel(s.unitsDir(), path)
		if err != nil {
			return err
		}
		file := strings.TrimSuffix(filepath.ToSlash(rel), recordExt)
		return readLines(path, func(line int, text []byte) error {
			var r Record
			if err := decodeStrict(text, &r); err != nil {
				return err
			}
			keyFile, err := fileOfKey(r.Key)
			if err != nil {
				return err
			}
			if keyFile != file {
				return fmt.Errorf("key %q belongs in the file of %q", r.Key, keyFile)
			}
			if _, dup := recs[r.Key]; dup {
				return fmt.Errorf("duplicate key %q", r.Key)
			}
			recs[r.Key] = r
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("store: load records: %w", err)
	}
	return recs, nil
}

// SaveRecords makes recs the whole record state: it writes one file per
// pack file that has records, each sorted by natural pointer order, and
// removes record files whose pack file no longer has any. Each map key must
// equal its Record.Key. A rejected map changes nothing on disk.
func (s *Store) SaveRecords(recs map[string]Record) error {
	type entry struct {
		pointer string
		rec     Record
	}
	byFile := map[string][]entry{}
	for key, r := range recs {
		if r.Key != key {
			return fmt.Errorf("store: save records: map key %q differs from record key %q", key, r.Key)
		}
		file, err := fileOfKey(key)
		if err != nil {
			return fmt.Errorf("store: save records: %w", err)
		}
		_, pointer, _ := units.SplitKey(key)
		byFile[file] = append(byFile[file], entry{pointer, r})
	}

	files := make([]string, 0, len(byFile))
	for file := range byFile {
		files = append(files, file)
	}
	sort.Strings(files)

	keep := map[string]bool{}
	for _, file := range files {
		es := byFile[file]
		sort.Slice(es, func(i, j int) bool { return comparePointers(es[i].pointer, es[j].pointer) < 0 })
		var buf bytes.Buffer
		for _, e := range es {
			if err := appendLine(&buf, e.rec); err != nil {
				return fmt.Errorf("store: save records: %w", err)
			}
		}
		path := s.recordPath(file)
		keep[path] = true
		if err := writeFileAtomic(path, buf.Bytes()); err != nil {
			return fmt.Errorf("store: save records: %w", err)
		}
	}
	if err := s.removeStale(keep); err != nil {
		return fmt.Errorf("store: save records: %w", err)
	}
	return nil
}

// removeStale deletes record files not in keep, then the directories that
// became empty (the units directory itself stays).
func (s *Store) removeStale(keep map[string]bool) error {
	var dirs []string
	err := filepath.WalkDir(s.unitsDir(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && path == s.unitsDir() {
				return nil
			}
			return err
		}
		if d.IsDir() {
			if path != s.unitsDir() {
				dirs = append(dirs, path)
			}
			return nil
		}
		if strings.HasSuffix(path, recordExt) && !keep[path] {
			return os.Remove(path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for i := len(dirs) - 1; i >= 0; i-- { // children before parents
		if err := os.Remove(dirs[i]); err != nil && !isNotEmpty(err) {
			return err
		}
	}
	return nil
}

// LoadPending reads the pending queue in file order (sorted by key). A
// worktree without state yields an empty queue.
func (s *Store) LoadPending() ([]PendingItem, error) {
	items := []PendingItem{}
	seen := map[string]bool{}
	err := readLines(s.pendingPath(), func(line int, text []byte) error {
		var it PendingItem
		if err := decodeStrict(text, &it); err != nil {
			return err
		}
		if _, err := fileOfKey(it.Key); err != nil {
			return err
		}
		if seen[it.Key] {
			return fmt.Errorf("duplicate key %q", it.Key)
		}
		seen[it.Key] = true
		items = append(items, it)
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("store: load pending: %w", err)
	}
	return items, nil
}

// SavePending makes items the whole pending queue, sorted by key. The
// caller's slice is left as it was. Keys must be valid and unique.
func (s *Store) SavePending(items []PendingItem) error {
	sorted := append([]PendingItem(nil), items...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Key < sorted[j].Key })
	var buf bytes.Buffer
	for i, it := range sorted {
		if _, err := fileOfKey(it.Key); err != nil {
			return fmt.Errorf("store: save pending: %w", err)
		}
		if i > 0 && sorted[i-1].Key == it.Key {
			return fmt.Errorf("store: save pending: duplicate key %q", it.Key)
		}
		if err := appendLine(&buf, it); err != nil {
			return fmt.Errorf("store: save pending: %w", err)
		}
	}
	if err := writeFileAtomic(s.pendingPath(), buf.Bytes()); err != nil {
		return fmt.Errorf("store: save pending: %w", err)
	}
	return nil
}

// fileOfKey returns the pack file of a unit key, requiring a valid key and a
// file name that stays inside the units directory.
func fileOfKey(key string) (string, error) {
	file, _, err := units.SplitKey(key)
	if err != nil {
		return "", err
	}
	if !filepath.IsLocal(filepath.FromSlash(file)) {
		return "", fmt.Errorf("invalid key %q: file name leaves the units directory", key)
	}
	return file, nil
}

// appendLine writes v as one compact JSON line: struct field order, no HTML
// escaping, newline-terminated.
func appendLine(buf *bytes.Buffer, v any) error {
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// decodeStrict decodes exactly one JSON value, refusing unknown fields and
// trailing data, so a hand edit or a newer writer is noticed instead of being
// silently dropped on the next save.
func decodeStrict(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		return errors.New("unexpected data after the JSON value")
	}
	return nil
}

// readLines calls fn for every non-blank line of the file. Errors from fn
// are wrapped with the path and 1-based line number. A missing file is
// returned as an error matching fs.ErrNotExist.
func readLines(path string, fn func(line int, text []byte) error) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for i, text := range bytes.Split(data, []byte("\n")) {
		if len(bytes.TrimSpace(text)) == 0 {
			continue
		}
		if err := fn(i+1, text); err != nil {
			return fmt.Errorf("%s:%d: %w", path, i+1, err)
		}
	}
	return nil
}

// writeFileAtomic replaces path with data: it writes a temporary file in the
// same directory and renames it over path, so readers see the old or the new
// content, never a mix. A file that already holds data is left alone.
func writeFileAtomic(path string, data []byte) error {
	if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, data) {
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// The name must not end in .jsonl, or a crash would leave a "record file".
	f, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, err = f.Write(data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(tmp, 0o644)
	}
	if err == nil {
		err = os.Rename(tmp, path)
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// isNotEmpty reports whether err is os.Remove refusing a directory that still
// has files in it (POSIX allows ENOTEMPTY or EEXIST).
func isNotEmpty(err error) bool {
	return errors.Is(err, syscall.ENOTEMPTY) || errors.Is(err, syscall.EEXIST)
}

// comparePointers orders two JSON pointers naturally and totally. Pointers
// are compared segment by segment (unescaped); a pointer that is a prefix of
// another comes first. Two segments compare as:
//   - both all digits: numerically (so 9 < 10), then, for equal numbers
//     written differently (7, 07), as text;
//   - one all digits: the number first. A plain "numerically if both are
//     digits, else as text" rule is not transitive (2 < 10 < 1a < 2), which
//     would make the sorted output depend on the input order;
//   - neither: as text.
func comparePointers(a, b string) int {
	ta, _ := jsonx.SplitPointer(a)
	tb, _ := jsonx.SplitPointer(b)
	for i := 0; i < len(ta) && i < len(tb); i++ {
		if c := compareSegments(ta[i], tb[i]); c != 0 {
			return c
		}
	}
	switch {
	case len(ta) < len(tb):
		return -1
	case len(ta) > len(tb):
		return 1
	}
	return strings.Compare(a, b)
}

func compareSegments(a, b string) int {
	da, db := allDigits(a), allDigits(b)
	switch {
	case da && db:
		na, nb := strings.TrimLeft(a, "0"), strings.TrimLeft(b, "0")
		if len(na) != len(nb) { // no leading zeros left: longer is larger
			if len(na) < len(nb) {
				return -1
			}
			return 1
		}
		if c := strings.Compare(na, nb); c != 0 {
			return c
		}
	case da:
		return -1
	case db:
		return 1
	}
	return strings.Compare(a, b)
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// metaPath returns the path of .meta/<name> for a file the caller keeps next
// to the records (such as foreign-names.json). name is a slash-separated path
// below .meta that is already clean and ends in ext, so it can neither leave
// .meta nor be mistaken for a record file (".jsonl"), the lock or the
// gitignore.
func (s *Store) metaPath(name, ext string) (string, error) {
	if !strings.HasSuffix(name, ext) || path.Clean(name) != name || !filepath.IsLocal(filepath.FromSlash(name)) {
		return "", fmt.Errorf("store: invalid meta file name %q: want a clean relative path ending in %s", name, ext)
	}
	return filepath.Join(s.metaDir(), filepath.FromSlash(name)), nil
}

// WriteMetaJSON writes v as JSON to .meta/<name> (see metaPath for the
// names allowed), creating directories as needed: 2-space indent, no HTML
// escaping, one trailing newline, so the same value always gives the same
// bytes. The file is replaced atomically. Like the other writers it does not
// take the lock; callers that need one run it inside WithLock.
func (s *Store) WriteMetaJSON(name string, v any) error {
	p, err := s.metaPath(name, ".json")
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("store: write %s: %w", name, err)
	}
	if err := writeFileAtomic(p, buf.Bytes()); err != nil {
		return fmt.Errorf("store: write %s: %w", name, err)
	}
	return nil
}

// WriteMetaFile writes data, a Markdown document such as a build report, to
// .meta/<name>, creating directories as needed. name is a clean relative path
// below .meta that ends in ".md". The file is replaced atomically and left
// alone when it already holds data. Like the other writers it does not take
// the lock.
func (s *Store) WriteMetaFile(name string, data []byte) error {
	p, err := s.metaPath(name, ".md")
	if err != nil {
		return err
	}
	if err := writeFileAtomic(p, data); err != nil {
		return fmt.Errorf("store: write %s: %w", name, err)
	}
	return nil
}

// ReadMetaJSON reads .meta/<name> into v. A missing file is not an error: it
// reports found as false and leaves v as it is. Unknown fields, trailing data
// and malformed JSON are errors naming the file.
func (s *Store) ReadMetaJSON(name string, v any) (found bool, err error) {
	p, err := s.metaPath(name, ".json")
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("store: read %s: %w", name, err)
	}
	if err := decodeStrict(data, v); err != nil {
		return false, fmt.Errorf("store: read %s: %w", name, err)
	}
	return true, nil
}
