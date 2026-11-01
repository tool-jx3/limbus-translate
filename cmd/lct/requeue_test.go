package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/batch"
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
