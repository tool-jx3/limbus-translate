package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/batch"
	"github.com/tool-jx3/limbus-translate/internal/requeue"
	"github.com/tool-jx3/limbus-translate/internal/store"
)

const (
	requeueYAML = batchYAML + "frozen: [\"Frozen/**\"]\n"
	frozenKey   = "Frozen/Lyric.json|/dataList/0/content"
)

// requeueRepo is batchRepo with a frozen file, records instead of the pending
// units, and the repo as the working directory (the tests run the real
// entry point).
func requeueRepo(t *testing.T) (root string, st *store.Store) {
	t.Helper()
	root, st = batchRepo(t)
	writeTestFile(t, root, "lct.yaml", requeueYAML)
	writeTestFile(t, filepath.Join(root, ".work", "raw"), "kr/Frozen/KR_Lyric.json", `{"dataList":[{"id":1,"content":"가사"}]}`)
	rec := func(key, zh, origin string) store.Record {
		return store.Record{Key: key, KRSha: "00", ZH: zh, Origin: origin, Since: "raw-20261029"}
	}
	if err := st.SaveRecords(map[string]store.Record{
		batchKey0: rec(batchKey0, "賦予 5 層 [Laceration] 強度", store.OriginClaude),
		batchKey1: rec(batchKey1, "Meursault", store.OriginCopy),
		frozenKey: rec(frozenKey, "歌詞", store.OriginClaude),
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.SavePending(nil); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	return root, st
}

func TestRequeueCLIUsage(t *testing.T) {
	_, st := requeueRepo(t)

	for name, tc := range map[string]struct {
		args   []string
		stderr string
	}{
		"no selection":        {[]string{"--reason", "x"}, "at least one"},
		"no reason":           {[]string{"--needs-review"}, "--reason"},
		"blank reason":        {[]string{"--file", "Sample.json", "--reason", "  "}, "--reason"},
		"unknown origin":      {[]string{"--origin", "robot", "--reason", "x"}, "robot"},
		"malformed key":       {[]string{"--key", "nope", "--reason", "x"}, "nope"},
		"unexpected argument": {[]string{"--file", "Sample.json", "--reason", "x", "extra"}, "extra"},
		"unknown flag":        {[]string{"--bogus", "--reason", "x"}, "bogus"},
	} {
		t.Run(name, func(t *testing.T) {
			code, stdout, stderr := lct(t, append([]string{"requeue"}, tc.args...)...)
			if code != 1 {
				t.Errorf("exit code = %d, want 1\nstderr: %s", code, stderr)
			}
			if stdout != "" || !strings.Contains(stderr, tc.stderr) {
				t.Errorf("stdout = %q, stderr = %q, want nothing on stdout and a message with %q", stdout, stderr, tc.stderr)
			}
		})
	}
	if got, err := st.LoadPending(); err != nil || len(got) != 0 {
		t.Errorf("a refused call changed the pending list: %+v, %v", got, err)
	}

	// Without a snapshot there is no KR to check against.
	t.Run("no snapshot", func(t *testing.T) {
		t.Chdir(snapshotRepo(t))
		code, _, stderr := lct(t, "requeue", "--leased", "--reason", "x")
		if code != 1 || !strings.Contains(stderr, "snapshot") {
			t.Errorf("exit code = %d, stderr = %q, want 1 and a hint to run lct snapshot", code, stderr)
		}
	})
}

func TestRequeueCLIJSON(t *testing.T) {
	_, st := requeueRepo(t)

	var raw map[string]json.RawMessage
	mustLct(t, &raw, "requeue", "--file", "Sample.json", "--file", "Frozen/**", "--reason", "稱呼改用您")
	var names []string
	for name := range raw {
		names = append(names, name)
	}
	slices.Sort(names)
	want := []string{"already_pending", "missing", "missing_keys", "released", "requeued", "reset", "skipped_copy", "skipped_frozen"}
	if !slices.Equal(names, want) {
		t.Errorf("JSON fields = %v, want %v", names, want)
	}
	counts := map[string]string{"requeued": "1", "reset": "0", "released": "0", "already_pending": "0", "skipped_frozen": "1", "skipped_copy": "1", "missing": "0", "missing_keys": "[]"}
	for name, wantValue := range counts {
		if got := strings.Join(strings.Fields(string(raw[name])), ""); got != wantValue {
			t.Errorf("%s = %s, want %s", name, got, wantValue)
		}
	}

	items, err := st.LoadPending()
	if err != nil {
		t.Fatal(err)
	}
	wantItem := store.PendingItem{Key: batchKey0, Reason: "requeue", PrevZH: "賦予 5 層 [Laceration] 強度", Hint: "稱呼改用您"}
	if len(items) != 1 || items[0].Key != wantItem.Key || items[0].Reason != wantItem.Reason || items[0].PrevZH != wantItem.PrevZH || items[0].Hint != wantItem.Hint {
		t.Errorf("pending = %+v, want [%+v]", items, wantItem)
	}

	// As text, and with a key that is gone.
	const gone = "Sample.json|/dataList/9/content"
	code, stdout, stderr := lct(t, "requeue", "--key", gone, "--reason", "x")
	if code != 0 || stderr != "" {
		t.Fatalf("exit code = %d, stderr = %q, want 0", code, stderr)
	}
	for _, part := range []string{"requeued 0", "missing (gone from KR, or no record): 1", gone} {
		if !strings.Contains(stdout, part) {
			t.Errorf("output lacks %q:\n%s", part, stdout)
		}
	}

	// --needs-review and --leased only narrow the pending units: no record
	// is looked at, so a key that is gone is not reported.
	var res struct {
		Requeued, Missing int
		MissingKeys       []string `json:"missing_keys"`
	}
	mustLct(t, &res, "requeue", "--key", gone, "--needs-review", "--leased", "--reason", "x")
	if res.Requeued != 0 || res.Missing != 0 || len(res.MissingKeys) != 0 {
		t.Errorf("result = %+v, want nothing: the key is no pending unit", res)
	}
}

// The hint written by requeue reaches the batch file a translator reads.
func TestRequeueCLIReachesTheBatch(t *testing.T) {
	root, st := requeueRepo(t)
	if err := st.SavePending([]store.PendingItem{{Key: batchKey1, Reason: "new", Attempts: 3, Flags: []string{store.FlagNeedsReview}}}); err != nil {
		t.Fatal(err)
	}
	var raw struct{ Reset, Requeued int }
	mustLct(t, &raw, "requeue", "--needs-review", "--key", batchKey0, "--key", batchKey1, "--reason", "別再用「閣下」")
	if raw.Reset != 1 || raw.Requeued != 0 {
		t.Fatalf("result = %+v, want 1 reset and nothing requeued (batchKey0 has a record, but is no pending unit)", raw)
	}

	got, _, err := runNextJSON(t, root)
	if err != nil || got.Batch == nil {
		t.Fatalf("batch next: %v, %+v", err, got)
	}
	data, err := os.ReadFile(got.Path)
	if err != nil {
		t.Fatal(err)
	}
	var b batch.Batch
	if err := json.Unmarshal(data, &b); err != nil {
		t.Fatal(err)
	}
	if len(b.Units) != 1 || b.Units[0].Key != batchKey1 {
		t.Fatalf("batch units = %+v, want only %s", b.Units, batchKey1)
	}
	if got := b.Units[0].Hint; got != "別再用「閣下」" {
		t.Errorf("unit %s hint = %q", b.Units[0].Key, got)
	}
}

// toFileArgs is a valid "requeue --to-file" call; the tests change single flags.
func toFileArgs(extra ...string) []string {
	return append([]string{"requeue", "--to-file", "honglu", "--date", "2026-11-05", "--file", "StoryData/S1*", "--reason", "稱呼改用您"}, extra...)
}

// twFiles is every file under the tw worktree with its content.
func twFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(filepath.Join(root, ".work", "tw"), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(p)
		files[p] = string(data)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestRequeueToFileTouchesNoState(t *testing.T) {
	root, st := requeueRepo(t)
	if err := st.SavePending([]store.PendingItem{{Key: batchKey1, Reason: "new", Hint: "old"}}); err != nil {
		t.Fatal(err)
	}
	before := twFiles(t, root)
	pendingPath := filepath.Join(root, ".work", "tw", ".meta", "pending.jsonl")
	pendingBefore, err := os.ReadFile(pendingPath)
	if err != nil || len(pendingBefore) == 0 {
		t.Fatalf("pending list = %q, %v", pendingBefore, err)
	}

	var out struct{ Path, ID string }
	mustLct(t, &out, "requeue", "--to-file", "x", "--date", "2026-11-05", "--file", "StoryData/S1*", "--reason", "r")
	if out.Path != "rules/requeue/2026-11-05-x.yaml" || out.ID != "2026-11-05-x" {
		t.Errorf("output = %+v, want the path and id of the file", out)
	}
	if _, err := os.Stat(filepath.Join(root, "rules", "requeue", "2026-11-05-x.yaml")); err != nil {
		t.Errorf("the request file is missing: %v", err)
	}
	if after, err := os.ReadFile(pendingPath); err != nil || string(after) != string(pendingBefore) {
		t.Errorf("pending list changed: %q, %v", after, err)
	}
	if after := twFiles(t, root); !reflect.DeepEqual(after, before) {
		t.Errorf("the tw worktree changed:\n got  %v\n want %v", after, before)
	}

	files, err := requeue.LoadFiles(root)
	if err != nil || len(files) != 1 || files[0].Reason != "r" || !reflect.DeepEqual(files[0].Selector, requeue.Selector{Files: []string{"StoryData/S1*"}}) {
		t.Errorf("LoadFiles = %+v, %v; want the request that was made", files, err)
	}
}

func TestRequeueToFileKeysOriginsAndText(t *testing.T) {
	root, _ := requeueRepo(t)
	code, stdout, stderr := lct(t, "requeue", "--to-file", "bufs", "--date", "2026-11-06",
		"--key", batchKey0, "--key", batchKey1, "--file", "Bufs*.json", "--origin", "claude", "--origin", "paratranz", "--reason", "改名")
	if code != 0 || stderr != "" {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	if want := "wrote rules/requeue/2026-11-06-bufs.yaml\n"; !strings.HasPrefix(stdout, want) {
		t.Errorf("output = %q, want it to start with %q", stdout, want)
	}
	files, err := requeue.LoadFiles(root)
	if err != nil || len(files) != 1 {
		t.Fatalf("LoadFiles = %+v, %v", files, err)
	}
	wantSel := requeue.Selector{Keys: []string{batchKey0, batchKey1}, Files: []string{"Bufs*.json"}, Origins: []string{"claude", "paratranz"}}
	if files[0].ID != "2026-11-06-bufs" || files[0].Reason != "改名" || !reflect.DeepEqual(files[0].Selector, wantSel) {
		t.Errorf("file = %+v, want selector %+v", files[0], wantSel)
	}
}

func TestRequeueToFileDefaultsToToday(t *testing.T) {
	root, _ := requeueRepo(t)
	old := ruleFileNow
	ruleFileNow = func() time.Time { return cliNow }
	t.Cleanup(func() { ruleFileNow = old })

	var out struct{ Path string }
	mustLct(t, &out, "requeue", "--to-file", "x", "--file", "Bufs*.json", "--reason", "r")
	if out.Path != "rules/requeue/2026-11-01-x.yaml" {
		t.Errorf("path = %q, want today's date (2026-11-01) in the name", out.Path)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(out.Path))); err != nil {
		t.Error(err)
	}
}

// The request file is for CI to apply later: it needs neither the raw
// snapshot nor the tw worktree.
func TestRequeueToFileNeedsNoSnapshot(t *testing.T) {
	root := snapshotRepo(t)
	t.Chdir(root)
	code, _, stderr := lct(t, toFileArgs()...)
	if code != 0 || stderr != "" {
		t.Fatalf("exit code = %d, stderr = %q, want 0", code, stderr)
	}
	if _, err := os.Stat(filepath.Join(root, "rules", "requeue", "2026-11-05-honglu.yaml")); err != nil {
		t.Error(err)
	}
}

func TestRequeueToFileUsage(t *testing.T) {
	root, _ := requeueRepo(t)
	if _, _, stderr := lct(t, toFileArgs()...); stderr != "" {
		t.Fatalf("the first call failed: %s", stderr)
	}
	before := twFiles(t, root)

	for name, tc := range map[string]struct {
		args   []string
		stderr string
	}{
		"with --needs-review": {toFileArgs("--needs-review"), "--needs-review"},
		"with --leased":       {toFileArgs("--leased"), "--leased"},
		"--date alone":        {[]string{"requeue", "--date", "2026-11-05", "--file", "a", "--reason", "r"}, "--date"},
		"no key or file":      {[]string{"requeue", "--to-file", "x", "--origin", "claude", "--reason", "r"}, "--key or --file"},
		"nothing selected":    {[]string{"requeue", "--to-file", "x", "--reason", "r"}, "--key or --file"},
		"no reason":           {[]string{"requeue", "--to-file", "x", "--file", "a"}, "--reason"},
		"bad date":            {[]string{"requeue", "--to-file", "x", "--date", "2026-13-40", "--file", "a", "--reason", "r"}, "2026-13-40"},
		"bad topic":           {[]string{"requeue", "--to-file", "a/b", "--file", "a", "--reason", "r"}, "a/b"},
		// An empty topic must not fall back to the direct mode.
		"empty topic":         {[]string{"requeue", "--to-file", "", "--file", "a", "--reason", "r"}, "--to-file needs a topic"},
		"empty topic with =":  {[]string{"requeue", "--to-file=", "--file", "a", "--reason", "r"}, "--to-file needs a topic"},
		"empty topic, a date": {[]string{"requeue", "--to-file", "", "--date", "2026-11-05", "--file", "a", "--reason", "r"}, "--to-file needs a topic"},
		"bad key":             {[]string{"requeue", "--to-file", "x", "--key", "nope", "--reason", "r"}, "nope"},
		"bad origin":          {[]string{"requeue", "--to-file", "x", "--file", "a", "--origin", "robot", "--reason", "r"}, "robot"},
		"bad glob":            {[]string{"requeue", "--to-file", "x", "--file", "[a", "--reason", "r"}, "[a"},
		"the file exists":     {toFileArgs(), "2026-11-05-honglu.yaml"},
	} {
		t.Run(name, func(t *testing.T) {
			code, stdout, stderr := lct(t, tc.args...)
			if code != 1 {
				t.Errorf("exit code = %d, want 1\nstderr: %s", code, stderr)
			}
			if stdout != "" || !strings.Contains(stderr, tc.stderr) {
				t.Errorf("stdout = %q, stderr = %q, want nothing on stdout and a message with %q", stdout, stderr, tc.stderr)
			}
		})
	}

	entries, err := os.ReadDir(filepath.Join(root, "rules", "requeue"))
	if err != nil || len(entries) != 1 {
		t.Errorf("rules/requeue = %v, %v; want only the file of the first call", entries, err)
	}
	if after := twFiles(t, root); !reflect.DeepEqual(after, before) {
		t.Errorf("a refused call changed the tw worktree")
	}
}
