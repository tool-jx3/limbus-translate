package store

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestMain doubles as the entry point of the helper process used by
// TestLeaseExclusiveAcrossProcesses: when the helper variables are set the
// binary leases one pending item and exits instead of running the tests.
func TestMain(m *testing.M) {
	if root := os.Getenv("STORE_TEST_HELPER_ROOT"); root != "" {
		if err := leaseOne(Open(root), os.Getenv("STORE_TEST_HELPER_BATCH")); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// leaseOne leases the first free pending item to batch. The pause between
// reading and writing the queue makes the race certain when the lock fails.
func leaseOne(s *Store, batch string) error {
	return s.WithLock(func() error {
		items, err := s.LoadPending()
		if err != nil {
			return err
		}
		now := time.Now()
		for i := range items {
			if LeaseFree(items[i], now) {
				time.Sleep(30 * time.Millisecond)
				items[i].Lease = &Lease{Batch: batch, Until: now.Add(time.Hour)}
				return s.SavePending(items)
			}
		}
		return errors.New("no free item")
	})
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func sampleRecords() map[string]Record {
	recs := []Record{
		{Key: "StoryData/S1.json|/dataList/10/content", KRSha: "c3", ZH: "第十句", Origin: OriginClaude, Since: "raw-20261029", Rules: "abc123", Model: "m1", Batch: "b-1"},
		{Key: "StoryData/S1.json|/dataList/9/content", KRSha: "c2", ZH: "第九句 <color=#ff0000>紅</color> & 更多", Origin: OriginParatranz, Since: "raw-20261029"},
		{Key: "A.json|/dataList/0/name", KRSha: "c1", ZH: "名稱", Origin: OriginFallback, Since: "raw-20261029", Flags: []string{FlagNeedsReview}},
	}
	m := map[string]Record{}
	for _, r := range recs {
		m[r.Key] = r
	}
	return m
}

func TestRecordsRoundTrip(t *testing.T) {
	root := t.TempDir()
	s := Open(root)
	want := sampleRecords()
	if err := s.SaveRecords(want); err != nil {
		t.Fatal(err)
	}

	story := filepath.Join(root, ".meta", "units", "StoryData", "S1.json.jsonl")
	a := filepath.Join(root, ".meta", "units", "A.json.jsonl")
	if !exists(story) || !exists(a) {
		t.Fatalf("expected %s and %s to exist", story, a)
	}

	wantStory := `{"key":"StoryData/S1.json|/dataList/9/content","kr_sha":"c2","zh":"第九句 <color=#ff0000>紅</color> & 更多","origin":"paratranz","since":"raw-20261029"}` + "\n" +
		`{"key":"StoryData/S1.json|/dataList/10/content","kr_sha":"c3","zh":"第十句","origin":"claude","since":"raw-20261029","rules":"abc123","model":"m1","batch":"b-1"}` + "\n"
	if got := readFile(t, story); got != wantStory {
		t.Errorf("S1 file:\n got %q\nwant %q", got, wantStory)
	}
	wantA := `{"key":"A.json|/dataList/0/name","kr_sha":"c1","zh":"名稱","origin":"fallback","since":"raw-20261029","flags":["needs_review"]}` + "\n"
	if got := readFile(t, a); got != wantA {
		t.Errorf("A file:\n got %q\nwant %q", got, wantA)
	}

	got, err := s.LoadRecords()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round trip mismatch:\n got %+v\nwant %+v", got, want)
	}
}

func TestLoadMissingMetaIsEmpty(t *testing.T) {
	s := Open(filepath.Join(t.TempDir(), "no-such-tw"))
	recs, err := s.LoadRecords()
	if err != nil || recs == nil || len(recs) != 0 {
		t.Errorf("LoadRecords = %v, %v; want empty non-nil map, nil", recs, err)
	}
	items, err := s.LoadPending()
	if err != nil || len(items) != 0 {
		t.Errorf("LoadPending = %v, %v; want empty, nil", items, err)
	}
}

func TestSaveRecordsRemovesFilesWithoutRecords(t *testing.T) {
	root := t.TempDir()
	s := Open(root)
	if err := s.SaveRecords(sampleRecords()); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(root, ".meta", "pending.jsonl")
	if err := os.WriteFile(keep, []byte("untouched\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	only := map[string]Record{}
	for k, r := range sampleRecords() {
		if strings.HasPrefix(k, "A.json|") {
			only[k] = r
		}
	}
	if err := s.SaveRecords(only); err != nil {
		t.Fatal(err)
	}
	if exists(filepath.Join(root, ".meta", "units", "StoryData", "S1.json.jsonl")) {
		t.Error("stale S1 file was not removed")
	}
	if exists(filepath.Join(root, ".meta", "units", "StoryData")) {
		t.Error("empty StoryData directory was not pruned")
	}
	if !exists(filepath.Join(root, ".meta", "units", "A.json.jsonl")) {
		t.Error("A file disappeared")
	}
	if readFile(t, keep) != "untouched\n" {
		t.Error("pending.jsonl was modified by SaveRecords")
	}

	if err := s.SaveRecords(nil); err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadRecords()
	if err != nil || len(got) != 0 {
		t.Errorf("after saving nothing, LoadRecords = %v, %v", got, err)
	}
	if !exists(filepath.Join(root, ".meta", "units")) {
		t.Error("the units directory itself must be kept")
	}
}

func TestSaveRecordsLeavesNoTempFiles(t *testing.T) {
	root := t.TempDir()
	s := Open(root)
	if err := s.SaveRecords(sampleRecords()); err != nil {
		t.Fatal(err)
	}
	if err := s.SavePending([]PendingItem{{Key: "A.json|/x", Reason: "new"}}); err != nil {
		t.Fatal(err)
	}
	err := filepath.WalkDir(filepath.Join(root, ".meta"), func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && !strings.HasSuffix(p, ".jsonl") {
			t.Errorf("unexpected leftover file %s", p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSaveRecordsRejectsBadInput(t *testing.T) {
	cases := map[string]map[string]Record{
		"key differs from record key": {"A.json|/x": {Key: "A.json|/y"}},
		"no separator":                {"A.json": {Key: "A.json"}},
		"empty file name":             {"|/x": {Key: "|/x"}},
		"bad pointer":                 {"A.json|x": {Key: "A.json|x"}},
		"parent directory":            {"../A.json|/x": {Key: "../A.json|/x"}},
		"nested parent directory":     {"a/../../A.json|/x": {Key: "a/../../A.json|/x"}},
		"absolute file":               {"/etc/A.json|/x": {Key: "/etc/A.json|/x"}},
	}
	for name, recs := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if err := Open(root).SaveRecords(recs); err == nil {
				t.Error("expected an error")
			}
			if exists(filepath.Join(root, ".meta", "units")) {
				entries, _ := os.ReadDir(filepath.Join(root, ".meta", "units"))
				if len(entries) != 0 {
					t.Errorf("rejected save left files behind: %v", entries)
				}
			}
		})
	}
}

func TestLoadRecordsRejectsCorruptFiles(t *testing.T) {
	good := `{"key":"A.json|/x","kr_sha":"k","zh":"z","origin":"claude","since":"s"}` + "\n"
	cases := map[string]struct {
		file    string
		content string
	}{
		"invalid json":      {"A.json.jsonl", "{not json}\n"},
		"unknown field":     {"A.json.jsonl", `{"key":"A.json|/x","kr_sha":"k","zh":"z","origin":"claude","since":"s","extra":1}` + "\n"},
		"trailing data":     {"A.json.jsonl", strings.TrimSuffix(good, "\n") + " {}\n"},
		"duplicate key":     {"A.json.jsonl", good + good},
		"key of other file": {"B.json.jsonl", good},
		"invalid key":       {"A.json.jsonl", `{"key":"nope","kr_sha":"k","zh":"z","origin":"claude","since":"s"}` + "\n"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, ".meta", "units")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, c.file), []byte(c.content), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := Open(root).LoadRecords()
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), c.file) {
				t.Errorf("error %q should name the file %s", err, c.file)
			}
		})
	}
}

func TestLoadRecordsSkipsBlankLinesAndOtherFiles(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".meta", "units")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	line := `{"key":"A.json|/x","kr_sha":"k","zh":"z","origin":"claude","since":"s"}`
	if err := os.WriteFile(filepath.Join(dir, "A.json.jsonl"), []byte("\n"+line+"\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("not a record file"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Open(root).LoadRecords()
	if err != nil || len(got) != 1 {
		t.Fatalf("LoadRecords = %v, %v", got, err)
	}
}

func TestPendingRoundTrip(t *testing.T) {
	root := t.TempDir()
	s := Open(root)
	until := time.Date(2026, 11, 1, 12, 30, 0, 0, time.UTC)
	in := []PendingItem{
		{Key: "StoryData/S1.json|/dataList/2/content", Reason: "kr-changed", PrevKR: "舊 KR", PrevZH: "舊譯文", Draft: "草稿 <b>&</b>",
			Refs: map[string]string{"llc": "LLC 譯", "ltm": "LTM 譯"}, Attempts: 2, Flags: []string{"needs_review"},
			Lease: &Lease{Batch: "b-7", Until: until}},
		{Key: "A.json|/dataList/0/name", Reason: "new"},
		{Key: "A.json|/dataList/0/desc", Reason: "paratranz-repair", Refs: map[string]string{}},
	}
	snapshot := append([]PendingItem(nil), in...)
	if err := s.SavePending(in); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, snapshot) {
		t.Error("SavePending reordered or modified the caller's slice")
	}

	want := `{"key":"A.json|/dataList/0/desc","reason":"paratranz-repair"}` + "\n" +
		`{"key":"A.json|/dataList/0/name","reason":"new"}` + "\n" +
		`{"key":"StoryData/S1.json|/dataList/2/content","reason":"kr-changed","prev_kr":"舊 KR","prev_zh":"舊譯文","draft":"草稿 <b>&</b>","refs":{"llc":"LLC 譯","ltm":"LTM 譯"},"attempts":2,"flags":["needs_review"],"lease":{"batch":"b-7","until":"2026-11-01T12:30:00Z"}}` + "\n"
	if got := readFile(t, filepath.Join(root, ".meta", "pending.jsonl")); got != want {
		t.Errorf("pending.jsonl:\n got %q\nwant %q", got, want)
	}

	got, err := s.LoadPending()
	if err != nil {
		t.Fatal(err)
	}
	wantItems := []PendingItem{
		{Key: "A.json|/dataList/0/desc", Reason: "paratranz-repair"},
		{Key: "A.json|/dataList/0/name", Reason: "new"},
		in[0],
	}
	if !reflect.DeepEqual(got, wantItems) {
		t.Errorf("round trip mismatch:\n got %+v\nwant %+v", got, wantItems)
	}
}

func TestPendingLeaseTimeIsUTC(t *testing.T) {
	root := t.TempDir()
	s := Open(root)
	zone := time.FixedZone("UTC+8", 8*3600)
	until := time.Date(2026, 11, 1, 20, 30, 15, 0, zone) // 12:30:15 UTC
	if err := s.SavePending([]PendingItem{{Key: "A.json|/x", Reason: "new", Lease: &Lease{Batch: "b", Until: until}}}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(root, ".meta", "pending.jsonl")); !strings.Contains(got, `"until":"2026-11-01T12:30:15Z"`) {
		t.Errorf("lease time not written as UTC: %s", got)
	}
	items, err := s.LoadPending()
	if err != nil || len(items) != 1 {
		t.Fatal(items, err)
	}
	if !items[0].Lease.Until.Equal(until) {
		t.Errorf("lease time changed: %v vs %v", items[0].Lease.Until, until)
	}
}

func TestSavePendingRejectsBadInput(t *testing.T) {
	cases := map[string][]PendingItem{
		"duplicate key": {{Key: "A.json|/x", Reason: "new"}, {Key: "A.json|/x", Reason: "requeue"}},
		"invalid key":   {{Key: "nope", Reason: "new"}},
	}
	for name, items := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if err := Open(root).SavePending(items); err == nil {
				t.Error("expected an error")
			}
			if exists(filepath.Join(root, ".meta", "pending.jsonl")) {
				t.Error("rejected save wrote the file")
			}
		})
	}
}

func TestLoadPendingRejectsCorruptFile(t *testing.T) {
	cases := map[string]string{
		"invalid json":  "{oops\n",
		"unknown field": `{"key":"A.json|/x","reason":"new","extra":true}` + "\n",
		"duplicate key": `{"key":"A.json|/x","reason":"new"}` + "\n" + `{"key":"A.json|/x","reason":"new"}` + "\n",
		"invalid key":   `{"key":"nope","reason":"new"}` + "\n",
		"bad lease":     `{"key":"A.json|/x","reason":"new","lease":{"batch":"b","until":"yesterday"}}` + "\n",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, ".meta"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, ".meta", "pending.jsonl"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := Open(root).LoadPending()
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), "pending.jsonl") {
				t.Errorf("error %q should name the file", err)
			}
		})
	}
}

func TestSaveEmptyPendingWritesEmptyFile(t *testing.T) {
	root := t.TempDir()
	s := Open(root)
	if err := s.SavePending([]PendingItem{{Key: "A.json|/x", Reason: "new"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.SavePending(nil); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(root, ".meta", "pending.jsonl")); got != "" {
		t.Errorf("want empty file, got %q", got)
	}
	items, err := s.LoadPending()
	if err != nil || len(items) != 0 {
		t.Errorf("LoadPending = %v, %v", items, err)
	}
}

func TestLeaseExclusive(t *testing.T) {
	root := t.TempDir()
	s := Open(root)
	if err := s.SavePending([]PendingItem{
		{Key: "A.json|/dataList/0/name", Reason: "new"},
		{Key: "A.json|/dataList/1/name", Reason: "new"},
	}); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, batch := range []string{"batch-a", "batch-b"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- leaseOne(s, batch)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	items, err := s.LoadPending()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Lease == nil || items[1].Lease == nil {
		t.Fatalf("both items should be leased: %+v", items)
	}
	if items[0].Lease.Batch == items[1].Lease.Batch {
		t.Errorf("one batch leased both items: %q", items[0].Lease.Batch)
	}
}

func TestLeaseExclusiveAcrossProcesses(t *testing.T) {
	root := t.TempDir()
	s := Open(root)
	const n = 4
	var items []PendingItem
	for i := range n {
		items = append(items, PendingItem{Key: fmt.Sprintf("A.json|/dataList/%d/name", i), Reason: "new"})
	}
	if err := s.SavePending(items); err != nil {
		t.Fatal(err)
	}

	cmds := make([]*exec.Cmd, n)
	outs := make([]*bytes.Buffer, n)
	for i := range cmds {
		cmds[i] = exec.Command(os.Args[0])
		cmds[i].Env = append(os.Environ(), "STORE_TEST_HELPER_ROOT="+root, fmt.Sprintf("STORE_TEST_HELPER_BATCH=proc-%d", i))
		outs[i] = new(bytes.Buffer)
		cmds[i].Stderr = outs[i]
	}
	for i, c := range cmds {
		if err := c.Start(); err != nil {
			t.Fatalf("start %d: %v", i, err)
		}
	}
	for i, c := range cmds {
		if err := c.Wait(); err != nil {
			t.Errorf("process %d: %v: %s", i, err, outs[i])
		}
	}

	got, err := s.LoadPending()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, it := range got {
		if it.Lease == nil {
			t.Fatalf("%s was not leased: another process overwrote its lease", it.Key)
		}
		if seen[it.Lease.Batch] {
			t.Errorf("batch %s leased two items", it.Lease.Batch)
		}
		seen[it.Lease.Batch] = true
	}
	if len(seen) != n {
		t.Errorf("want %d distinct batches, got %d", n, len(seen))
	}
}

func TestWithLockReleasesAfterError(t *testing.T) {
	root := t.TempDir()
	s := Open(root)
	boom := errors.New("boom")
	if err := s.WithLock(func() error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("WithLock returned %v, want the function's error", err)
	}
	if !exists(filepath.Join(root, ".meta", ".lock")) {
		t.Error(".meta/.lock was not created")
	}
	done := make(chan error, 1)
	go func() { done <- s.WithLock(func() error { return nil }) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("lock was not released after the function returned an error")
	}
}

func TestLeaseExpiry(t *testing.T) {
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name  string
		lease *Lease
		free  bool
	}{
		{"no lease", nil, true},
		{"expired", &Lease{Batch: "b", Until: now.Add(-time.Minute)}, true},
		{"expires exactly now", &Lease{Batch: "b", Until: now}, true},
		{"active", &Lease{Batch: "b", Until: now.Add(time.Minute)}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := LeaseFree(PendingItem{Key: "A.json|/x", Lease: c.lease}, now); got != c.free {
				t.Errorf("LeaseFree = %v, want %v", got, c.free)
			}
		})
	}
}

func TestDeterministicFiles(t *testing.T) {
	// Insertion order of a map is random; the files must not depend on it.
	var keys []string
	for i := range 40 {
		keys = append(keys, fmt.Sprintf("F%d.json|/dataList/%d/content", i%3, i))
	}
	build := func(seed int64) map[string]Record {
		r := rand.New(rand.NewSource(seed))
		order := r.Perm(len(keys))
		m := map[string]Record{}
		for _, i := range order {
			m[keys[i]] = Record{Key: keys[i], KRSha: "k", ZH: "譯" + keys[i], Origin: OriginClaude, Since: "raw-1"}
		}
		return m
	}
	dump := func(root string) map[string]string {
		out := map[string]string{}
		err := filepath.WalkDir(filepath.Join(root, ".meta"), func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, _ := filepath.Rel(root, p)
			out[rel] = readFile(t, p)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	pend := func(seed int64) []PendingItem {
		r := rand.New(rand.NewSource(seed))
		var items []PendingItem
		for _, i := range r.Perm(len(keys)) {
			items = append(items, PendingItem{Key: keys[i], Reason: "new", Refs: map[string]string{"b": "2", "a": "1", "c": "3"}})
		}
		return items
	}

	first, second := t.TempDir(), t.TempDir()
	for _, tc := range []struct {
		root string
		seed int64
	}{{first, 1}, {second, 2}} {
		s := Open(tc.root)
		if err := s.SaveRecords(build(tc.seed)); err != nil {
			t.Fatal(err)
		}
		if err := s.SavePending(pend(tc.seed)); err != nil {
			t.Fatal(err)
		}
	}
	a, b := dump(first), dump(second)
	if len(a) != 4 { // three record files and pending.jsonl
		t.Errorf("want 4 files, got %d: %v", len(a), a)
	}
	if !reflect.DeepEqual(a, b) {
		t.Error("different insertion orders produced different bytes")
	}

	// Saving the same data again must not change anything.
	s := Open(first)
	if err := s.SaveRecords(build(3)); err != nil {
		t.Fatal(err)
	}
	if err := s.SavePending(pend(3)); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(dump(first), a) {
		t.Error("saving the same data twice changed the files")
	}
}

func TestNaturalPointerOrder(t *testing.T) {
	// Each row is in the order it must come out.
	sorted := []string{
		"",
		"/a",
		"/a/1",
		"/a/2",
		"/a/2/x",
		"/a/9",
		"/a/10",
		"/a/10/b",
		"/a/10/b/0",
		"/a/10/b/1",
		"/a/99999999999999999999",  // beyond int64
		"/a/100000000000000000000", // longer digit string is the larger number
		"/a/b",                     // digits sort before names
		"/a/c",
		"/b",
	}
	for i := 0; i+1 < len(sorted); i++ {
		if comparePointers(sorted[i], sorted[i+1]) >= 0 {
			t.Errorf("%q should sort before %q", sorted[i], sorted[i+1])
		}
		if comparePointers(sorted[i+1], sorted[i]) <= 0 {
			t.Errorf("%q should sort after %q", sorted[i+1], sorted[i])
		}
	}
	// Equal numbers written differently still have a fixed order.
	if comparePointers("/a/7", "/a/07") == 0 || comparePointers("/a/7", "/a/07") != -comparePointers("/a/07", "/a/7") {
		t.Error("/a/7 and /a/07 need a stable, antisymmetric order")
	}
	if comparePointers("/a/x", "/a/x") != 0 {
		t.Error("identical pointers must compare equal")
	}
}

func TestNaturalPointerOrderIsTotal(t *testing.T) {
	// "2" < "10" numerically, "10" < "1a" and "1a" < "2" lexically: with a
	// plain "numbers numerically, else lexically" rule this is a cycle and the
	// sorted output would depend on the input order.
	ptrs := []string{"/o/2", "/o/10", "/o/1a", "/o/", "/o/-1", "/o/9", "/o/a1", "/o/A"}
	var want []string
	r := rand.New(rand.NewSource(7))
	for range 50 {
		shuffled := append([]string(nil), ptrs...)
		r.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		sort.Slice(shuffled, func(i, j int) bool { return comparePointers(shuffled[i], shuffled[j]) < 0 })
		if want == nil {
			want = shuffled
			continue
		}
		if !reflect.DeepEqual(shuffled, want) {
			t.Fatalf("order depends on input:\n%v\n%v", shuffled, want)
		}
	}
	for _, a := range ptrs {
		for _, b := range ptrs {
			for _, c := range ptrs {
				if comparePointers(a, b) < 0 && comparePointers(b, c) < 0 && comparePointers(a, c) >= 0 {
					t.Errorf("not transitive: %q < %q < %q but not %q < %q", a, b, c, a, c)
				}
			}
		}
	}
}

func TestNaturalPointerOrderUsesUnescapedTokens(t *testing.T) {
	// "/a~0b" is the single key "a~b" and "/a/b" is key "a" then "b". Compared
	// as keys, "a" sorts before "a~b"; compared as escaped text, "/a/" would
	// sort before "/a~" too, so use a case where the two readings disagree:
	// "~1" stands for "/" (0x2f) but its escaped text starts with "~" (0x7e).
	if got := comparePointers("/a~1b", "/a0"); got >= 0 {
		t.Errorf("segment \"a/b\" must sort before \"a0\", got %d", got)
	}
}
