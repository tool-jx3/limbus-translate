package batch

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/align"
	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

const testTag = "raw-20261029"

var testNow = time.Date(2026, 11, 1, 3, 4, 5, 0, time.UTC)

// The rules of the tests: two families, two characters and two glossary
// terms (one still a candidate).
const (
	testRouting = `families:
  - id: story-main
    nodes: ["StoryData/S"]
    guide: guides/families.md
  - id: other
    nodes: []
    guide: guides/families.md
priority:
  - story-main
  - other
`
	testGlossary = `- kr: 위력
  zh: 威力
  status: approved
  forbidden: [力量]
- kr: 후보
  zh: 候選
  status: candidate
`
	testFaust = `---
id: faust
name: 浮士德
name_kr: 파우스트
models: [파우스트]
file_names: [Faust]
---

# 浮士德
`
	testSinclair = `---
id: sinclair
name: 辛克萊
name_kr: 싱클레어
models: [싱클레어]
file_names: [Sinclair]
---

# 辛克萊
`
)

func writeFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func testRules(t *testing.T) *rules.Rules { return testRulesWith(t, testGlossary) }

// testRulesWith is testRules with the glossary replaced.
func testRulesWith(t *testing.T, glossary string) *rules.Rules {
	t.Helper()
	return testRulesWithRouting(t, testRouting, glossary)
}

// testRulesWithRouting is testRules with the routing and the glossary
// replaced.
func testRulesWithRouting(t *testing.T, routing, glossary string) *rules.Rules {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "guides/routing.yaml"), routing)
	writeFile(t, filepath.Join(root, "glossary/terms.yaml"), glossary)
	writeFile(t, filepath.Join(root, "guides/characters/faust.md"), testFaust)
	writeFile(t, filepath.Join(root, "guides/characters/sinclair.md"), testSinclair)
	r, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// packFile is where a pack file lives under the raw root: the language
// prefix goes on the file name only.
func packFile(lang, prefix, file string) string {
	dir, name := path.Split(file)
	return path.Join(lang, dir, prefix+name)
}

// newEnv writes the KR, EN and JP files (file name to JSON text), loads the
// rules and opens an empty store.
func newEnv(t *testing.T, kr, en, jp map[string]string) Env {
	t.Helper()
	return newEnvIn(t, filepath.Join(t.TempDir(), "tw"), kr, en, jp)
}

// newEnvIn is newEnv with the tw worktree at twRoot, for tests that open the
// same store more than once.
func newEnvIn(t *testing.T, twRoot string, kr, en, jp map[string]string) Env {
	t.Helper()
	raw := t.TempDir()
	for _, l := range []struct {
		lang, prefix string
		files        map[string]string
	}{{"kr", "KR_", kr}, {"en", "EN_", en}, {"jp", "JP_", jp}} {
		if err := os.MkdirAll(filepath.Join(raw, l.lang), 0o755); err != nil {
			t.Fatal(err)
		}
		for file, text := range l.files {
			writeFile(t, filepath.Join(raw, filepath.FromSlash(packFile(l.lang, l.prefix, file))), text)
		}
	}
	src, err := pack.OpenRaw(raw)
	if err != nil {
		t.Fatal(err)
	}
	return Env{
		Store:  store.Open(twRoot),
		Src:    src,
		Rules:  testRules(t),
		RawTag: testTag,
		Now:    testNow,
	}
}

func q(t *testing.T, s string) string {
	t.Helper()
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// dataList makes a file of n entries {"id": i, "model": model, "content": text(i)};
// the model is left out when it is empty.
func dataList(t *testing.T, n int, model string, text func(i int) string) string {
	t.Helper()
	var b strings.Builder
	b.WriteString(`{"dataList":[`)
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `{"id":%d,`, i)
		if model != "" {
			fmt.Fprintf(&b, `"model":%s,`, q(t, model))
		}
		fmt.Fprintf(&b, `"content":%s}`, q(t, text(i)))
	}
	b.WriteString(`]}`)
	return b.String()
}

func line(i int) string { return fmt.Sprintf("대사 %d", i) }

func key(file string, i int) string { return fmt.Sprintf("%s|/dataList/%d/content", file, i) }

// items makes pending items for entries from..to-1 of file.
func items(file string, from, to int) []store.PendingItem {
	var out []store.PendingItem
	for i := from; i < to; i++ {
		out = append(out, store.PendingItem{Key: key(file, i), Reason: "new"})
	}
	return out
}

func seed(t *testing.T, env Env, pending ...store.PendingItem) {
	t.Helper()
	if err := env.Store.SavePending(pending); err != nil {
		t.Fatal(err)
	}
}

func mustNext(t *testing.T, env Env, opt NextOptions) *Batch {
	t.Helper()
	b, err := Next(env, opt)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	return b
}

func pendingMap(t *testing.T, env Env) map[string]store.PendingItem {
	t.Helper()
	list, err := env.Store.LoadPending()
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]store.PendingItem{}
	for _, it := range list {
		m[it.Key] = it
	}
	return m
}

func records(t *testing.T, env Env) map[string]store.Record {
	t.Helper()
	recs, err := env.Store.LoadRecords()
	if err != nil {
		t.Fatal(err)
	}
	return recs
}

func keysOf(b *Batch) []string {
	var out []string
	for _, u := range b.Units {
		out = append(out, u.Key)
	}
	return out
}

func idSuffix(k string) string {
	sum := sha256.Sum256([]byte(k))
	return hex.EncodeToString(sum[:])[:8]
}

var entryRe = regexp.MustCompile(`/dataList/(\d+)/`)

// translate makes a valid translation for every unit of b: the Chinese
// "譯文" and the number of the entry.
func translate(b *Batch) []OutUnit {
	var out []OutUnit
	for _, u := range b.Units {
		out = append(out, OutUnit{Key: u.Key, ZH: "譯文 " + entryRe.FindStringSubmatch(u.Key)[1]})
	}
	return out
}

func TestNextOrderAndLease(t *testing.T) {
	kr := map[string]string{
		"Other.json":           dataList(t, 1, "", line),
		"StoryData/S001A.json": dataList(t, 1, "파우스트", line),
	}
	env := newEnv(t, kr, nil, nil)
	// Stored in key order, "Other.json" comes first; the priority puts the
	// story first.
	seed(t, env, append(items("Other.json", 0, 1), items("StoryData/S001A.json", 0, 1)...)...)

	b1 := mustNext(t, env, NextOptions{})
	if b1 == nil || b1.File != "StoryData/S001A.json" || b1.Family != "story-main" {
		t.Fatalf("first batch = %+v, want the story file of family story-main", b1)
	}
	if want := key("StoryData/S001A.json", 0); !slices.Equal(keysOf(b1), []string{want}) {
		t.Fatalf("first batch keys = %v, want [%s]", keysOf(b1), want)
	}
	wantID := regexp.MustCompile(`^20261101T030405-` + idSuffix(keysOf(b1)[0]) + `$`)
	if !wantID.MatchString(b1.ID) {
		t.Errorf("batch id = %q, want 20261101T030405-%s", b1.ID, idSuffix(keysOf(b1)[0]))
	}

	b2 := mustNext(t, env, NextOptions{})
	if b2 == nil || b2.File != "Other.json" {
		t.Fatalf("second batch = %+v, want Other.json", b2)
	}
	for _, k := range keysOf(b2) {
		if slices.Contains(keysOf(b1), k) {
			t.Errorf("key %s was handed out twice", k)
		}
	}
	if b3 := mustNext(t, env, NextOptions{}); b3 != nil {
		t.Errorf("third Next = %+v, want nil: everything is leased", b3)
	}

	pending := pendingMap(t, env)
	lease := pending[keysOf(b1)[0]].Lease
	if lease == nil || lease.Batch != b1.ID || !lease.Until.Equal(testNow.Add(120*time.Minute)) {
		t.Errorf("lease = %+v, want batch %s until %s", lease, b1.ID, testNow.Add(120*time.Minute))
	}

	// A lease that ran out is handed out again, to a new batch.
	later := env
	later.Now = testNow.Add(3 * time.Hour)
	b4 := mustNext(t, later, NextOptions{})
	if b4 == nil || b4.File != "StoryData/S001A.json" || b4.ID == b1.ID {
		t.Errorf("batch after expiry = %+v, want the story file again under a new id", b4)
	}
}

func TestNextSkipsNeedsReviewAndStale(t *testing.T) {
	kr := map[string]string{"Other.json": dataList(t, 3, "", line)}
	env := newEnv(t, kr, nil, nil)
	list := items("Other.json", 0, 3)
	list[0].Flags = []string{store.FlagNeedsReview}
	list = append(list,
		store.PendingItem{Key: "Gone.json|/dataList/0/content", Reason: "new"},         // file left KR
		store.PendingItem{Key: "Other.json|/dataList/99/content", Reason: "new"},       // pointer left KR
		store.PendingItem{Key: "Other.json|/dataList/0/nothing", Reason: "kr-changed"}, // no such field
	)
	seed(t, env, list...)

	b := mustNext(t, env, NextOptions{})
	if b == nil {
		t.Fatal("Next = nil, want the two healthy items")
	}
	want := []string{key("Other.json", 1), key("Other.json", 2)}
	if !slices.Equal(keysOf(b), want) {
		t.Errorf("keys = %v, want %v", keysOf(b), want)
	}
	pending := pendingMap(t, env)
	if pending[key("Other.json", 0)].Lease != nil || pending["Gone.json|/dataList/0/content"].Lease != nil {
		t.Error("an item that was not handed out must not be leased")
	}
	if len(pending) != 6 {
		t.Errorf("pending has %d items, want all 6 kept", len(pending))
	}
}

// Available counts what Next hands out: every kind of item Next skips is left
// out of the count, and draining the queue with Next takes exactly as many
// units as the count said.
func TestAvailableCountsWhatNextHandsOut(t *testing.T) {
	kr := map[string]string{
		"Other.json":           dataList(t, 6, "", line),
		"StoryData/S001A.json": dataList(t, 3, "파우스트", line),
	}
	env := newEnv(t, kr, nil, nil)
	list := items("Other.json", 0, 6)
	list[0].Flags = []string{store.FlagNeedsReview}
	list[1].Lease = &store.Lease{Batch: "20261101T000000-aaaaaaaa", Until: testNow.Add(time.Hour)}  // leased
	list[2].Lease = &store.Lease{Batch: "20261101T000000-bbbbbbbb", Until: testNow.Add(-time.Hour)} // expired
	list = append(list, items("StoryData/S001A.json", 0, 3)...)
	list = append(list,
		store.PendingItem{Key: "Gone.json|/dataList/0/content", Reason: "new"},         // file left KR
		store.PendingItem{Key: "Other.json|/dataList/99/content", Reason: "new"},       // pointer left KR
		store.PendingItem{Key: "Other.json|/dataList/0/nothing", Reason: "kr-changed"}, // no such field
	)
	seed(t, env, list...)

	got, err := Available(env, testNow)
	if err != nil {
		t.Fatal(err)
	}
	if want := 7; got != want { // Other 2..5 (4) and the story (3)
		t.Errorf("Available = %d, want %d", got, want)
	}

	// Available reads the queue each time it is called.
	b := mustNext(t, env, NextOptions{})
	if after, err := Available(env, testNow); err != nil || after != got-len(b.Units) {
		t.Errorf("Available after a batch of %d = %d, %v, want %d", len(b.Units), after, err, got-len(b.Units))
	}

	taken := len(b.Units)
	for {
		b := mustNext(t, env, NextOptions{})
		if b == nil {
			break
		}
		taken += len(b.Units)
	}
	if taken != got {
		t.Errorf("Next handed out %d units in all, Available said %d", taken, got)
	}
	if left, err := Available(env, testNow); err != nil || left != 0 {
		t.Errorf("Available after draining = %d, %v, want 0", left, err)
	}
}

func TestCountClaimableAsksKROnlyForFreeItems(t *testing.T) {
	pending := items("A.json", 0, 3)
	pending[0].Flags = []string{store.FlagNeedsReview}
	pending[1].Lease = &store.Lease{Batch: "b", Until: testNow.Add(time.Hour)}
	var asked []string
	n := CountClaimable(pending, testNow, func(k string) bool {
		asked = append(asked, k)
		return true
	})
	if n != 1 || !slices.Equal(asked, []string{key("A.json", 2)}) {
		t.Errorf("CountClaimable = %d having asked about %v, want 1 having asked about the free item only", n, asked)
	}
	if !Claimable(pending[2], testNow) || Claimable(pending[0], testNow) || Claimable(pending[1], testNow) {
		t.Error("Claimable must be true for the free item only")
	}
}

func TestNextFamilyAndLimit(t *testing.T) {
	kr := map[string]string{
		"Other.json":           dataList(t, 5, "", line),
		"StoryData/S001A.json": dataList(t, 3, "파우스트", line),
	}
	env := newEnv(t, kr, nil, nil)
	seed(t, env, append(items("Other.json", 0, 5), items("StoryData/S001A.json", 0, 3)...)...)

	b := mustNext(t, env, NextOptions{Family: "other", MaxUnits: 2})
	if b == nil || b.File != "Other.json" || len(b.Units) != 2 {
		t.Fatalf("batch = %+v, want 2 units of Other.json although the story has priority", b)
	}
	b = mustNext(t, env, NextOptions{Family: "other", MaxUnits: 2})
	if b == nil || len(b.Units) != 2 || b.Units[0].Key != key("Other.json", 2) {
		t.Fatalf("second batch = %+v, want the next 2 units", b)
	}
	if b := mustNext(t, env, NextOptions{Family: "enemy"}); b != nil {
		t.Errorf("batch of an empty family = %+v, want nil", b)
	}
	// Story batches have their own limit.
	b = mustNext(t, env, NextOptions{Family: "story-main", MaxUnits: 1, StoryMaxUnits: 2})
	if b == nil || len(b.Units) != 2 {
		t.Errorf("story batch = %+v, want StoryMaxUnits=2 units", b)
	}
}

func TestNextDisjointUnderConcurrency(t *testing.T) {
	const files, perFile = 10, 7
	kr := map[string]string{}
	var pending []store.PendingItem
	for i := 0; i < files; i++ {
		file := fmt.Sprintf("Other%02d.json", i)
		kr[file] = dataList(t, perFile, "", line)
		pending = append(pending, items(file, 0, perFile)...)
	}
	twRoot := filepath.Join(t.TempDir(), "tw")
	env := newEnvIn(t, twRoot, kr, nil, nil)
	seed(t, env, pending...)

	var (
		mu    sync.Mutex
		seen  = map[string]string{} // key -> batch id
		dups  []string
		wg    sync.WaitGroup
		fails = make(chan error, 8)
	)
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e := env
			e.Store = store.Open(twRoot) // its own store, as another process would have
			for {
				b, err := Next(e, NextOptions{MaxUnits: 4})
				if err != nil {
					fails <- err
					return
				}
				if b == nil {
					return
				}
				mu.Lock()
				for _, k := range keysOf(b) {
					if prev, dup := seen[k]; dup {
						dups = append(dups, fmt.Sprintf("%s in %s and %s", k, prev, b.ID))
					}
					seen[k] = b.ID
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	close(fails)
	for err := range fails {
		t.Fatalf("Next: %v", err)
	}
	if len(dups) > 0 {
		t.Errorf("%d keys handed out twice, such as %v", len(dups), dups[:min(3, len(dups))])
	}
	if len(seen) != files*perFile {
		t.Errorf("%d keys handed out, want %d", len(seen), files*perFile)
	}
}

func TestStorySplit(t *testing.T) {
	const file = "StoryData/S001A.json"
	env := newEnv(t, map[string]string{file: dataList(t, 250, "파우스트", line)}, nil, nil)
	seed(t, env, items(file, 0, 250)...)
	opt := NextOptions{MaxUnits: 40, StoryMaxUnits: 120}

	b1 := mustNext(t, env, opt)
	if len(b1.Units) != 120 || b1.Units[0].Key != key(file, 0) || b1.Units[119].Key != key(file, 119) {
		t.Fatalf("batch 1 = %d units from %s to %s", len(b1.Units), b1.Units[0].Key, b1.Units[len(b1.Units)-1].Key)
	}
	if len(b1.Context) != 0 {
		t.Errorf("batch 1 context = %v, want none: nothing is translated yet", b1.Context)
	}
	res, err := Submit(env, b1, translate(b1), "test-model", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 120 || len(res.Rejected) != 0 {
		t.Fatalf("submit 1: %d accepted, rejected %v", len(res.Accepted), res.Rejected)
	}

	b2 := mustNext(t, env, opt)
	if len(b2.Units) != 120 || b2.Units[0].Key != key(file, 120) || b2.Units[119].Key != key(file, 239) {
		t.Fatalf("batch 2 = %d units from %s to %s", len(b2.Units), b2.Units[0].Key, b2.Units[len(b2.Units)-1].Key)
	}
	// The last three lines of batch 1 come first (the model strings between
	// them are not translations); the end of batch 2 borders on line 240,
	// which waits for batch 3.
	var gotCtx []string
	for _, c := range b2.Context {
		gotCtx = append(gotCtx, c.Key+"="+c.ZH)
	}
	var wantCtx []string
	for i := 117; i < 120; i++ {
		wantCtx = append(wantCtx, fmt.Sprintf("%s=譯文 %d", key(file, i), i))
	}
	if !slices.Equal(gotCtx, wantCtx) {
		t.Errorf("batch 2 context = %v, want %v", gotCtx, wantCtx)
	}
	if c := b2.Context[0]; c.Speaker != "faust" || c.KR != line(117) {
		t.Errorf("context line = %+v, want speaker faust and KR %q", c, line(117))
	}

	// Batch 2 is still out: its lines are not context for batch 3.
	b3 := mustNext(t, env, opt)
	if len(b3.Units) != 10 || b3.Units[0].Key != key(file, 240) {
		t.Fatalf("batch 3 = %d units from %s", len(b3.Units), b3.Units[0].Key)
	}
	if len(b3.Context) != 0 {
		t.Errorf("batch 3 context = %v, want none while batch 2 is untranslated", b3.Context)
	}
	if b4 := mustNext(t, env, opt); b4 != nil {
		t.Errorf("batch 4 = %+v, want nil", b4)
	}
}

// A story batch takes every unit that can be handed out, with the lines that
// cannot in between: a line that needs review is skipped, it does not end the
// batch.
func TestStoryBatchSkipsLinesThatAreNotEligible(t *testing.T) {
	const file = "StoryData/S001A.json"
	env := newEnv(t, map[string]string{file: dataList(t, 12, "파우스트", line)}, nil, nil)
	// Lines 0-1 and 5-6 are translated; 3 needs review; 9 is out with another
	// batch; 2, 4, 7, 8 and 10 wait; 11 is translated.
	pending := []store.PendingItem{}
	for _, i := range []int{2, 3, 4, 7, 8, 9, 10} {
		it := store.PendingItem{Key: key(file, i), Reason: "new"}
		switch i {
		case 3:
			it.Flags = []string{store.FlagNeedsReview}
		case 9:
			it.Lease = &store.Lease{Batch: "20261101T020000-aaaaaaaa", Until: testNow.Add(time.Hour)}
		}
		pending = append(pending, it)
	}
	seed(t, env, pending...)
	recs := map[string]store.Record{}
	for _, i := range []int{0, 1, 5, 6, 11} {
		recs[key(file, i)] = store.Record{Key: key(file, i), KRSha: units.KRSha(line(i)), ZH: fmt.Sprintf("舊 %d", i), Origin: store.OriginParatranz, Since: testTag}
	}
	// A copy record is no translation, and the fallback text of line 3 is not
	// one either.
	recs[key(file, 3)] = store.Record{Key: key(file, 3), ZH: "line 3", Origin: store.OriginFallback, Since: testTag}
	recs["StoryData/S001A.json|/dataList/6/model"] = store.Record{Key: "StoryData/S001A.json|/dataList/6/model", ZH: "파우스트", Origin: store.OriginCopy, Since: testTag}
	if err := env.Store.SaveRecords(recs); err != nil {
		t.Fatal(err)
	}

	// One batch holds 2, 4, 7, 8 and 10; lines 3 and 9, which cannot be handed
	// out, stay behind without ending it.
	b1 := mustNext(t, env, NextOptions{})
	if want := []string{key(file, 2), key(file, 4), key(file, 7), key(file, 8), key(file, 10)}; !slices.Equal(keysOf(b1), want) {
		t.Fatalf("batch 1 = %v, want %v", keysOf(b1), want)
	}
	// Every line is within three lines of one of the units. Lines 3 and 9 are
	// not translations; the model of line 6 is a copy.
	if got, want := ctxKeys(b1), []string{key(file, 0), key(file, 1), key(file, 5), key(file, 6), key(file, 11)}; !slices.Equal(got, want) {
		t.Errorf("batch 1 context = %v, want %v", got, want)
	}
	if b2 := mustNext(t, env, NextOptions{}); b2 != nil {
		t.Errorf("batch 2 = %+v, want nil: line 3 needs review and line 9 is leased", b2)
	}
}

func ctxKeys(b *Batch) []string {
	out := []string{}
	for _, c := range b.Context {
		out = append(out, c.Key)
	}
	return out
}

func TestStoryContextLimit(t *testing.T) {
	const file = "StoryData/S001A.json"
	// No model strings: one unit per line.
	env := newEnv(t, map[string]string{file: dataList(t, 30, "", line)}, nil, nil)
	// Three units: a batch of fewer looks at fewer lines (see
	// TestStoryContextOfASmallBatch).
	seed(t, env, items(file, 10, 13)...)
	recs := map[string]store.Record{}
	for i := 0; i < 30; i++ {
		if i < 10 || i > 12 {
			recs[key(file, i)] = store.Record{Key: key(file, i), KRSha: units.KRSha(line(i)), ZH: fmt.Sprintf("舊 %d", i), Origin: store.OriginParatranz, Since: testTag}
		}
	}
	if err := env.Store.SaveRecords(recs); err != nil {
		t.Fatal(err)
	}
	b := mustNext(t, env, NextOptions{})
	// Three lines before line 10 and three after line 12.
	var want []string
	for _, i := range []int{7, 8, 9, 13, 14, 15} {
		want = append(want, key(file, i))
	}
	if got := ctxKeys(b); !slices.Equal(got, want) {
		t.Errorf("context = %v, want 3 lines on each side: %v", got, want)
	}
}

func TestStoryContextSkipsNeedsReviewLines(t *testing.T) {
	const file = "StoryData/S001A.json"
	env := newEnv(t, map[string]string{file: dataList(t, 10, "", line)}, nil, nil)
	// Lines 0-3 are translated, line 4 waits for a person, 5 and 6 are to
	// translate, 7 needs review too, 8 is translated, 9 is pending.
	var pending []store.PendingItem
	for _, i := range []int{4, 5, 6, 7, 9} {
		it := store.PendingItem{Key: key(file, i), Reason: "new"}
		if i == 4 || i == 7 {
			it.Flags = []string{store.FlagNeedsReview}
		}
		pending = append(pending, it)
	}
	seed(t, env, pending...)
	recs := map[string]store.Record{}
	for _, i := range []int{0, 1, 2, 3, 8} {
		recs[key(file, i)] = store.Record{Key: key(file, i), KRSha: units.KRSha(line(i)), ZH: fmt.Sprintf("舊 %d", i), Origin: store.OriginParatranz, Since: testTag}
	}
	for _, i := range []int{4, 7} {
		recs[key(file, i)] = store.Record{Key: key(file, i), KRSha: units.KRSha(line(i)), ZH: "line", Origin: store.OriginFallback, Since: testTag, Flags: []string{store.FlagNeedsReview}}
	}
	if err := env.Store.SaveRecords(recs); err != nil {
		t.Fatal(err)
	}

	b := mustNext(t, env, NextOptions{})
	if want := []string{key(file, 5), key(file, 6), key(file, 9)}; !slices.Equal(keysOf(b), want) {
		t.Fatalf("units = %v, want %v", keysOf(b), want)
	}
	// Lines 4 and 7 need review: not translations, and not in the way either.
	// Line 0 is out of reach (three lines before line 5 reach line 2).
	want := []string{key(file, 2), key(file, 3), key(file, 8)}
	if got := ctxKeys(b); !slices.Equal(got, want) {
		t.Errorf("context = %v, want %v", got, want)
	}
}

// translated is a translation of kr made from the current KR text.
func translated(k, kr, zh string) store.Record {
	return store.Record{Key: k, KRSha: units.KRSha(kr), ZH: zh, Origin: store.OriginParatranz, Since: testTag}
}

// A story file is one batch: its pending lines need not be next to each other,
// and the lines around them that are translated are the context.
func TestStoryBatchTakesWholeFile(t *testing.T) {
	const file = "StoryData/S001A.json"
	// Every line has a model string too, which has no translation to show and
	// does not count as a line.
	env := newEnv(t, map[string]string{file: dataList(t, 10, "파우스트", line)}, nil, nil)
	seed(t, env, store.PendingItem{Key: key(file, 1), Reason: "new"},
		store.PendingItem{Key: key(file, 4), Reason: "new"}, store.PendingItem{Key: key(file, 8), Reason: "new"})
	recs := map[string]store.Record{}
	for _, i := range []int{0, 2, 3, 5, 6, 7, 9} {
		recs[key(file, i)] = translated(key(file, i), line(i), fmt.Sprintf("舊 %d", i))
	}
	if err := env.Store.SaveRecords(recs); err != nil {
		t.Fatal(err)
	}

	b := mustNext(t, env, NextOptions{})
	if want := []string{key(file, 1), key(file, 4), key(file, 8)}; !slices.Equal(keysOf(b), want) {
		t.Fatalf("units = %v, want all three pending lines of the file", keysOf(b))
	}
	// Within three lines of line 1: 0, 2, 3; of line 4: 2, 3, 5, 6, 7; of line
	// 8: 5, 6, 7, 9. Each once, in document order, none of the batch's own.
	var want []string
	for _, i := range []int{0, 2, 3, 5, 6, 7, 9} {
		want = append(want, key(file, i))
	}
	if got := ctxKeys(b); !slices.Equal(got, want) {
		t.Errorf("context = %v, want %v", got, want)
	}
	for _, c := range b.Context {
		if c.ZH == "" || !strings.HasPrefix(c.ZH, "舊 ") {
			t.Errorf("context line %+v has no stored translation", c)
		}
	}
	if next := mustNext(t, env, NextOptions{}); next != nil {
		t.Errorf("a second batch = %+v, want nil: the file is leased", next)
	}
}

// A file with more units than StoryMaxUnits is cut in document order, the next
// batch carrying on from the first unit left.
func TestStoryBatchSplitsAtMax(t *testing.T) {
	const file = "StoryData/S001A.json"
	env := newEnv(t, map[string]string{file: dataList(t, 500, "", line)}, nil, nil)
	var pending []store.PendingItem
	for i := 0; i < 500; i += 2 { // every other line: 250 pending, none next to another
		pending = append(pending, store.PendingItem{Key: key(file, i), Reason: "new"})
	}
	seed(t, env, pending...)
	opt := NextOptions{MaxUnits: 40, StoryMaxUnits: 120}

	next := 0 // the n-th pending line the next batch starts at
	for n, size := range []int{120, 120, 10} {
		b := mustNext(t, env, opt)
		if b == nil {
			t.Fatalf("batch %d = nil, want %d units", n+1, size)
		}
		if len(b.Units) != size {
			t.Fatalf("batch %d has %d units, want %d", n+1, len(b.Units), size)
		}
		if first, last := b.Units[0].Key, b.Units[size-1].Key; first != key(file, 2*next) || last != key(file, 2*(next+size-1)) {
			t.Errorf("batch %d runs from %s to %s, want lines %d to %d", n+1, first, last, 2*next, 2*(next+size-1))
		}
		next += size
	}
	if b := mustNext(t, env, opt); b != nil {
		t.Errorf("batch 4 = %+v, want nil", b)
	}
}

// The context reaches three lines on each side of a batch of three units, a
// line being a dataList element: the model and teller strings of a dialogue
// line do not use up places.
func TestStoryContextCountsLines(t *testing.T) {
	const file = "StoryData/S001A.json"
	var kr strings.Builder
	kr.WriteString(`{"dataList":[`)
	for i := 0; i < 20; i++ {
		if i > 0 {
			kr.WriteByte(',')
		}
		fmt.Fprintf(&kr, `{"id":%d,"model":"파우스트","teller":"파우스트","content":%s}`, i, q(t, line(i)))
	}
	kr.WriteString(`]}`)
	env := newEnv(t, map[string]string{file: kr.String()}, nil, nil)
	seed(t, env, items(file, 9, 12)...)
	recs := map[string]store.Record{}
	for i := 0; i < 20; i++ {
		k := func(field string) string { return fmt.Sprintf("%s|/dataList/%d/%s", file, i, field) }
		recs[k("model")] = store.Record{Key: k("model"), KRSha: units.KRSha("파우스트"), ZH: "파우스트", Origin: store.OriginCopy, Since: testTag}
		recs[k("teller")] = translated(k("teller"), "파우스트", "浮士德")
		if i < 9 || i > 11 {
			recs[k("content")] = translated(k("content"), line(i), fmt.Sprintf("舊 %d", i))
		}
	}
	if err := env.Store.SaveRecords(recs); err != nil {
		t.Fatal(err)
	}

	b := mustNext(t, env, NextOptions{})
	if want := []string{key(file, 9), key(file, 10), key(file, 11)}; !slices.Equal(keysOf(b), want) {
		t.Fatalf("units = %v, want %v", keysOf(b), want)
	}
	// Three dialogue lines on each side of lines 9 to 11.
	var dialogue []string
	fields := map[string]int{}
	for _, c := range b.Context {
		fields[units.FieldName(c.Key[strings.Index(c.Key, "|")+1:])]++
		n, _ := strconv.Atoi(entryRe.FindStringSubmatch(c.Key)[1])
		if n < 6 || n > 14 {
			t.Errorf("context line %s is more than three lines from the batch", c.Key)
		}
		if strings.HasSuffix(c.Key, "/content") {
			dialogue = append(dialogue, c.Key)
		}
	}
	var want []string
	for _, i := range []int{6, 7, 8, 12, 13, 14} {
		want = append(want, key(file, i))
	}
	if !slices.Equal(dialogue, want) {
		t.Errorf("dialogue lines in the context = %v, want %v", dialogue, want)
	}
	// The tellers are translated strings of those lines (the batch's too); the
	// models are copies.
	if want := map[string]int{"content": 6, "teller": 9}; !reflect.DeepEqual(fields, want) {
		t.Errorf("context lines by field = %v, want %v", fields, want)
	}
}

// A string outside the dataList is in no line, so it is no neighbour of one.
func TestStoryContextSkipsStringsOutsideTheDataList(t *testing.T) {
	const file = "StoryData/S001A.json"
	kr := `{"title":"제목","dataList":[{"id":0,"content":"대사 0"},{"id":1,"content":"대사 1"}]}`
	env := newEnv(t, map[string]string{file: kr}, nil, nil)
	seed(t, env, store.PendingItem{Key: key(file, 0), Reason: "new"})
	if err := env.Store.SaveRecords(map[string]store.Record{
		file + "|/title": translated(file+"|/title", "제목", "標題"),
		key(file, 1):     translated(key(file, 1), "대사 1", "舊 1"),
	}); err != nil {
		t.Fatal(err)
	}
	b := mustNext(t, env, NextOptions{})
	if got, want := ctxKeys(b), []string{key(file, 1)}; !slices.Equal(got, want) {
		t.Errorf("context = %v, want %v", got, want)
	}
}

func TestIsStory(t *testing.T) {
	tests := []struct {
		file string
		want bool
	}{
		{"StoryData/S001A.json", true},
		{"RPGSystem/rpg-loc-dialogue-floor-3.json", true},
		{"RPGSystem/rpg-loc-dialogue-common-a1c10p1.json", true},
		{"RPGSystem/rpg-loc-dialogue-route-c.json", true},
		{"RPGSystem/rpg-loc-dialogue-choice-floor-3.json", false}, // the options of a dialogue
		{"RPGSystem/rpg-loc-dialogue-choice-common-a1c10p1.json", false},
		{"RPGSystem/rpg-loc-location-floor-3.json", false},
		{"RPGSystem/rpg-loc-item-common.json", false},
		{"RPGSystem/other/rpg-loc-dialogue-floor-3.json", false}, // one directory only
		{"rpg-loc-dialogue-floor-3.json", false},
		{"Other.json", false},
	}
	for _, tt := range tests {
		if got := IsStory(tt.file); got != tt.want {
			t.Errorf("IsStory(%q) = %v, want %v", tt.file, got, tt.want)
		}
	}
}

// The RPG dialogue files hold the speaker in the object of each line
// (dataList[].texts[].speaker), not in a model. They are read as a story:
// whole file, context, and the speaker's guide. Their option files are not.
func TestRPGDialogueIsStory(t *testing.T) {
	const dialogue = "RPGSystem/rpg-loc-dialogue-floor-3.json"
	const choice = "RPGSystem/rpg-loc-dialogue-choice-floor-3.json"
	text := func(file string, entry, n int) string {
		return fmt.Sprintf("%s|/dataList/%d/texts/%d/text", file, entry, n)
	}

	t.Run("dialogue", func(t *testing.T) {
		const kr = `{"dataList":[
 {"key":"D1","texts":[{"index":0,"text":"과감하게… 신중하게…"}]},
 {"key":"D2","texts":[
  {"index":0,"text":"엇… 이번엔 저랑 오티스 씨인가 보네요.","speaker":"싱클레어"},
  {"index":1,"text":"무슨 말이지?","speaker":"오티스"},
  {"index":2,"text":"…아, 그렇군요.","speaker":"싱클레어"}]},
 {"key":"D3","texts":[{"index":0,"text":"이쪽이다."}]},
 {"key":"D4","texts":[{"index":0,"text":"넷."}]},
 {"key":"D5","texts":[{"index":0,"text":"다섯."}]},
 {"key":"D6","texts":[{"index":0,"text":"여섯."}]},
 {"key":"D7","texts":[{"index":0,"text":"일곱."}]},
 {"key":"D8","texts":[{"index":0,"text":"여덟."}]}
]}`
		env := newEnv(t, map[string]string{dialogue: kr}, nil, nil)
		// D1 and D4-D8 are translated; the four lines of D2 and D3 wait.
		seed(t, env,
			store.PendingItem{Key: text(dialogue, 1, 0), Reason: "new"},
			store.PendingItem{Key: text(dialogue, 1, 1), Reason: "new"},
			store.PendingItem{Key: text(dialogue, 1, 2), Reason: "new"},
			store.PendingItem{Key: text(dialogue, 2, 0), Reason: "new"})
		recs := map[string]store.Record{
			text(dialogue, 0, 0): translated(text(dialogue, 0, 0), "과감하게… 신중하게…", "放手一搏……"),
		}
		for i, kr := range map[int]string{3: "넷.", 4: "다섯.", 5: "여섯.", 6: "일곱.", 7: "여덟."} {
			recs[text(dialogue, i, 0)] = translated(text(dialogue, i, 0), kr, "譯")
		}
		if err := env.Store.SaveRecords(recs); err != nil {
			t.Fatal(err)
		}

		// Four units are more than a plain batch of 2 would hold.
		b := mustNext(t, env, NextOptions{MaxUnits: 2})
		if b == nil || len(b.Units) != 4 {
			t.Fatalf("batch = %+v, want one batch of all 4 pending lines", b)
		}
		var speakers []string
		for _, u := range b.Units {
			speakers = append(speakers, u.Speaker)
		}
		// 싱클레어 is a character; 오티스 is not one here and stays as written;
		// the last line has no speaker.
		if want := []string{"sinclair", "오티스", "sinclair", ""}; !slices.Equal(speakers, want) {
			t.Errorf("speakers = %q, want %q", speakers, want)
		}
		if !slices.Contains(b.Guides, "guides/characters/sinclair.md") {
			t.Errorf("guides = %v, want sinclair's", b.Guides)
		}
		// A group of texts is a line: D1 before, and D4-D6 within three of D3.
		// D7 and D8 are too far.
		want := []string{text(dialogue, 0, 0), text(dialogue, 3, 0), text(dialogue, 4, 0), text(dialogue, 5, 0)}
		if !slices.Equal(ctxKeys(b), want) {
			t.Errorf("context = %v, want %v", ctxKeys(b), want)
		}
	})

	t.Run("options", func(t *testing.T) {
		const kr = `{"dataList":[
 {"key":"DC1","text":"그대가 되리니"},
 {"key":"DC2","text":"우리가 되리니"},
 {"key":"DC3","text":"값이 오르리니"}]}`
		env := newEnv(t, map[string]string{choice: kr}, nil, nil)
		var pending []store.PendingItem
		for i := 0; i < 3; i++ {
			pending = append(pending, store.PendingItem{Key: fmt.Sprintf("%s|/dataList/%d/text", choice, i), Reason: "new"})
		}
		seed(t, env, pending...)
		b := mustNext(t, env, NextOptions{MaxUnits: 2})
		if b == nil || len(b.Units) != 2 {
			t.Fatalf("batch = %+v, want a plain batch of 2 units", b)
		}
		if len(b.Context) != 0 {
			t.Errorf("context = %v, want none", b.Context)
		}
	})
}

// The speaker of a line is its dataList element's model; when that is empty,
// the "speaker" string of the object the line is in.
func TestSpeakerFromTheObjectOfTheLine(t *testing.T) {
	const file = "Other.json"
	kr := map[string]string{file: `{"dataList":[
 {"id":0,"model":"파우스트","speaker":"싱클레어","content":"가"},
 {"id":1,"texts":[{"text":"나","speaker":"싱클레어"}]},
 {"id":2,"texts":[{"text":"다"}]},
 {"id":3,"tags":["라"],"speaker":"싱클레어"},
 {"id":4,"texts":[{"text":"마","speaker":7}]},
 {"id":5,"model":"","texts":[{"text":"바","speaker":"싱클레어"}]}
]}`}
	env := newEnv(t, kr, nil, nil)
	seed(t, env,
		store.PendingItem{Key: file + "|/dataList/0/content", Reason: "new"},
		store.PendingItem{Key: file + "|/dataList/1/texts/0/text", Reason: "new"},
		store.PendingItem{Key: file + "|/dataList/2/texts/0/text", Reason: "new"},
		store.PendingItem{Key: file + "|/dataList/3/tags/0", Reason: "new"},
		store.PendingItem{Key: file + "|/dataList/4/texts/0/text", Reason: "new"},
		store.PendingItem{Key: file + "|/dataList/5/texts/0/text", Reason: "new"})
	b := mustNext(t, env, NextOptions{})
	if b == nil || len(b.Units) != 6 {
		t.Fatalf("batch = %+v, want 6 units", b)
	}
	got := map[string]string{}
	for _, u := range b.Units {
		got[strings.TrimPrefix(u.Key, file+"|")] = u.Speaker
	}
	want := map[string]string{
		"/dataList/0/content":      "faust",    // the model wins over a speaker beside it
		"/dataList/1/texts/0/text": "sinclair", // no model: the speaker of the object
		"/dataList/2/texts/0/text": "",         // neither
		"/dataList/3/tags/0":       "",         // the speaker is not in the object of the string
		"/dataList/4/texts/0/text": "",         // not a string
		"/dataList/5/texts/0/text": "sinclair", // an empty model is no model
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("speakers = %v, want %v", got, want)
	}
}

func TestOtherBatchesHaveNoContext(t *testing.T) {
	env := newEnv(t, map[string]string{"Other.json": dataList(t, 4, "", line)}, nil, nil)
	seed(t, env, items("Other.json", 1, 2)...)
	if err := env.Store.SaveRecords(map[string]store.Record{
		key("Other.json", 0): {Key: key("Other.json", 0), KRSha: units.KRSha(line(0)), ZH: "舊", Origin: store.OriginParatranz, Since: testTag},
	}); err != nil {
		t.Fatal(err)
	}
	b := mustNext(t, env, NextOptions{})
	if b.Context == nil || len(b.Context) != 0 {
		t.Errorf("context = %#v, want an empty list", b.Context)
	}
}

func TestBatchUnitContent(t *testing.T) {
	const file = "StoryData/S001A.json"
	kr := map[string]string{
		file: `{"dataList":[
 {"id":0,"model":"파우스트","content":"[Bleed] 위력 5 부여"},
 {"id":1,"model":"낯선이","content":"<그 너머의 빛>"},
 {"id":2,"model":"파우스트","content":"<color=#f8c200>공격</color> {0} [Burn] 위력"}
]}`,
		"Bufs.json": `{"dataList":[{"id":"Bleed","name":"출혈"},{"id":"Burn","name":"화상"}]}`,
	}
	en := map[string]string{
		file: `{"dataList":[
 {"id":0,"model":"파우스트","content":"Inflict 5 [Bleed] Potency"},
 {"id":1,"model":"낯선이","content":"<The light beyond>"},
 {"id":2,"model":"파우스트","content":"<color=#f8c200>Attack</color> {0} [Burn] Potency"}
]}`,
	}
	env := newEnv(t, kr, en, nil)
	pending := items(file, 0, 3)
	pending[0].Reason = "paratranz-repair"
	pending[0].PrevKR = "이전 원문"
	pending[0].PrevZH = "舊譯文"
	pending[0].Draft = "施加 5 級流血"
	pending[0].Refs = map[string]string{"llc": "赋予 5 级流血", "ltm": ""}
	seed(t, env, pending...)
	if err := env.Store.SaveRecords(map[string]store.Record{
		"Bufs.json|/dataList/0/name": {Key: "Bufs.json|/dataList/0/name", KRSha: units.KRSha("출혈"), ZH: "流血", Origin: store.OriginParatranz, Since: testTag},
	}); err != nil {
		t.Fatal(err)
	}

	b := mustNext(t, env, NextOptions{})
	if len(b.Units) != 3 {
		t.Fatalf("batch has %d units, want 3", len(b.Units))
	}
	if len(b.Guides) == 0 || b.Guides[0] != "guides/base.md" {
		t.Errorf("guides = %v, want guides/base.md first", b.Guides)
	}
	for _, want := range []string{"guides/families.md", "guides/characters/faust.md"} {
		if !slices.Contains(b.Guides, want) {
			t.Errorf("guides = %v, missing %s", b.Guides, want)
		}
	}
	if len(b.Glossary) != 1 || b.Glossary[0].KR != "위력" || b.Glossary[0].ZH != "威力" {
		t.Errorf("glossary = %+v, want the one approved term, once", b.Glossary)
	}
	// Bleed has our name; Burn is still named in Korean, which is no name of
	// ours.
	if want := map[string]string{"Bleed": "流血"}; !reflect.DeepEqual(b.Keywords, want) {
		t.Errorf("keywords = %v, want %v", b.Keywords, want)
	}

	u := b.Units[0]
	if u.Key != key(file, 0) || u.Field != "content" || u.Speaker != "faust" || u.KR != "[Bleed] 위력 5 부여" {
		t.Errorf("unit 0 = %+v", u)
	}
	if u.EN == nil || u.EN.Method != align.MethodPointer || u.EN.Value != "Inflict 5 [Bleed] Potency" {
		t.Errorf("unit 0 EN = %+v, want the pointer-aligned EN text", u.EN)
	}
	if u.JP != nil {
		t.Errorf("unit 0 JP = %+v, want nil: no JP file", u.JP)
	}
	if !slices.Equal(u.Keep.IDs, []string{"[Bleed]"}) {
		t.Errorf("unit 0 keep.ids = %v", u.Keep.IDs)
	}
	if u.Reason != "paratranz-repair" || u.PrevKR != "이전 원문" || u.PrevZH != "舊譯文" || u.Draft != "施加 5 級流血" {
		t.Errorf("unit 0 reason/prev/draft = %q %q %q %q", u.Reason, u.PrevKR, u.PrevZH, u.Draft)
	}
	if want := map[string]string{"llc": "赋予 5 级流血"}; !reflect.DeepEqual(u.Refs, want) {
		t.Errorf("unit 0 refs = %v, want %v (the empty ltm ref is absent)", u.Refs, want)
	}
	if u.Keep.DanteWrapper || len(u.Keep.Placeholders) != 0 || len(u.Keep.Tags) != 0 {
		t.Errorf("unit 0 keep = %+v", u.Keep)
	}

	d := b.Units[1]
	if !d.Keep.DanteWrapper || d.Speaker != "낯선이" {
		t.Errorf("unit 1 = %+v, want a Dante line by the raw model 낯선이", d)
	}
	if d.Refs != nil {
		t.Errorf("unit 1 refs = %v, want none", d.Refs)
	}

	m := b.Units[2]
	if !slices.Equal(m.Keep.Placeholders, []string{"{0}"}) ||
		!slices.Equal(m.Keep.Tags, []string{"<color=#f8c200>", "</color>"}) ||
		!slices.Equal(m.Keep.IDs, []string{"[Burn]"}) {
		t.Errorf("unit 2 keep = %+v", m.Keep)
	}
}

// The reason a person sent a unit back (lct requeue) reaches the translator.
func TestNextCarriesHint(t *testing.T) {
	env := newEnv(t, map[string]string{"Other.json": dataList(t, 2, "", line)}, nil, nil)
	pending := items("Other.json", 0, 2)
	pending[0].Reason = "requeue"
	pending[0].PrevZH = "舊譯"
	pending[0].Hint = "稱呼改用「您」"
	seed(t, env, pending...)

	b := mustNext(t, env, NextOptions{})
	if len(b.Units) != 2 {
		t.Fatalf("batch has %d units, want 2", len(b.Units))
	}
	if got := b.Units[0].Hint; got != "稱呼改用「您」" {
		t.Errorf("unit 0 hint = %q, want the hint of its pending item", got)
	}
	if got := b.Units[1].Hint; got != "" {
		t.Errorf("unit 1 hint = %q, want none", got)
	}
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(data), `"hint"`); n != 1 {
		t.Errorf("the batch JSON has %d hint properties, want 1 (only a unit with a hint has one): %s", n, data)
	}
}

// A translation made from another KR text is no translation of the text there
// is now: it names no keyword and is no line of context.
func TestStaleRecordsAreNotUsed(t *testing.T) {
	t.Run("keyword names", func(t *testing.T) {
		kr := map[string]string{
			"Other.json": `{"dataList":[{"id":0,"content":"[Bleed] [Burn] 부여"}]}`,
			"Bufs.json":  `{"dataList":[{"id":"Bleed","name":"출혈"},{"id":"Burn","name":"화상"}]}`,
		}
		env := newEnv(t, kr, nil, nil)
		seed(t, env, items("Other.json", 0, 1)...)
		if err := env.Store.SaveRecords(map[string]store.Record{
			"Bufs.json|/dataList/0/name": {Key: "Bufs.json|/dataList/0/name", KRSha: units.KRSha("출혈"), ZH: "流血", Origin: store.OriginParatranz, Since: testTag},
			// Made when Burn was called something else.
			"Bufs.json|/dataList/1/name": {Key: "Bufs.json|/dataList/1/name", KRSha: units.KRSha("옛 이름"), ZH: "舊名", Origin: store.OriginParatranz, Since: testTag},
		}); err != nil {
			t.Fatal(err)
		}
		b := mustNext(t, env, NextOptions{})
		if want := map[string]string{"Bleed": "流血"}; !reflect.DeepEqual(b.Keywords, want) {
			t.Errorf("keywords = %v, want %v: the name of Burn was made from another KR text", b.Keywords, want)
		}
	})
	t.Run("story context", func(t *testing.T) {
		const file = "StoryData/S001A.json"
		env := newEnv(t, map[string]string{file: dataList(t, 4, "", line)}, nil, nil)
		seed(t, env, items(file, 2, 3)...)
		recs := map[string]store.Record{}
		for _, i := range []int{0, 1, 3} {
			recs[key(file, i)] = store.Record{Key: key(file, i), KRSha: units.KRSha(line(i)), ZH: fmt.Sprintf("舊 %d", i), Origin: store.OriginParatranz, Since: testTag}
		}
		stale := recs[key(file, 1)]
		stale.KRSha = units.KRSha("옛 대사")
		recs[stale.Key] = stale
		if err := env.Store.SaveRecords(recs); err != nil {
			t.Fatal(err)
		}
		b := mustNext(t, env, NextOptions{})
		if got, want := ctxKeys(b), []string{key(file, 3)}; !slices.Equal(got, want) {
			t.Errorf("context = %v, want %v: line 1 was translated from another KR text", got, want)
		}
	})
}

func TestSpeakerOfVoiceFile(t *testing.T) {
	const file = "PersonalityVoiceDlg/Voice_Faust_10201.json"
	kr := map[string]string{file: `{"dataList":[{"id":"a","model":"낯선이","content":"안녕"}]}`}
	env := newEnv(t, kr, nil, nil)
	seed(t, env, store.PendingItem{Key: file + "|/dataList/0/content", Reason: "new"})
	b := mustNext(t, env, NextOptions{})
	if got := b.Units[0].Speaker; got != "faust" {
		t.Errorf("speaker = %q, want faust: a voice file belongs to the character in its name", got)
	}
	if !slices.Contains(b.Guides, "guides/characters/faust.md") {
		t.Errorf("guides = %v, want faust's", b.Guides)
	}
}

// Two units are the same text when family, field, speaker and KR all agree;
// a text under ShortTextRunes runes is shared only within its file.
func TestTextGroupOf(t *testing.T) {
	r := testRules(t)
	doc, err := jsonx.Parse([]byte(`{"dataList":[
 {"id":0,"model":"파우스트","content":"같은 말"},
 {"id":1,"model":"낯선이","content":"같은 말"},
 {"id":2,"texts":[{"speaker":"싱클레어","content":"같은 말"}]}
]}`))
	if err != nil {
		t.Fatal(err)
	}
	list := units.Extract("StoryData/S1.json", doc)
	byPointer := map[string]units.Unit{}
	for _, u := range list {
		byPointer[u.Pointer] = u
	}
	group := func(pointer string) TextGroup {
		t.Helper()
		u, ok := byPointer[pointer]
		if !ok {
			t.Fatalf("no unit at %s", pointer)
		}
		return TextGroupOf(r, doc, u)
	}

	if got, want := group("/dataList/0/content"), (TextGroup{Family: "story-main", Field: "content", Speaker: "faust", KR: "같은 말"}); got != want {
		t.Errorf("group of a known speaker = %+v, want %+v", got, want)
	}
	if got := group("/dataList/1/content").Speaker; got != "낯선이" {
		t.Errorf("speaker of an unknown model = %q, want the model itself", got)
	}
	if got := group("/dataList/2/texts/0/content").Speaker; got != "sinclair" {
		t.Errorf("speaker of an RPG line = %q, want sinclair", got)
	}
	if SpeakerOf(r, doc, byPointer["/dataList/0/content"]) != "faust" {
		t.Errorf("SpeakerOf differs from the group's speaker")
	}
	if group("/dataList/0/content") == group("/dataList/1/content") {
		t.Errorf("lines of two speakers are in one group")
	}

	for _, tt := range []struct {
		kr   string
		want bool
	}{{"123456789", true}, {"1234567890", false}, {"아홉 글자의 문장", true}, {"열 글자가 되는 문장", false}} {
		if got := (TextGroup{KR: tt.kr}).FileBound(); got != tt.want {
			t.Errorf("FileBound(%q) = %v, want %v", tt.kr, got, tt.want)
		}
	}
}

// CheckNew holds a text to what Submit accepts: the glossary terms of the KR
// text are enforced, content issues are hard.
func TestCheckNew(t *testing.T) {
	r := testRules(t)
	if issues := CheckNew(r, "Other.json", "위력 5", "威力5"); validate.HasHard(issues) {
		t.Errorf("CheckNew of a good text = %+v, want no hard issue", issues)
	}
	if issues := CheckNew(r, "Other.json", "위력 5", "力量5"); !validate.HasHard(issues) {
		t.Errorf("CheckNew of a forbidden rendering = %+v, want a hard issue", issues)
	}
	if issues := CheckNew(r, "Other.json", "위력 5", "威力5 위"); !validate.HasHard(issues) {
		t.Errorf("CheckNew of a text with Hangul = %+v, want a hard issue", issues)
	}
}

func TestSubmitAcceptReject(t *testing.T) {
	kr := map[string]string{"Other.json": `{"dataList":[
 {"id":0,"content":"[Bleed] 위력 5 부여"},
 {"id":1,"content":"[Burn] 부여"}
]}`}
	env := newEnv(t, kr, nil, nil)
	env.RulesCommit = "0123abc+dirty"
	seed(t, env, items("Other.json", 0, 2)...)
	b := mustNext(t, env, NextOptions{})

	res, err := Submit(env, b, []OutUnit{
		{Key: key("Other.json", 0), ZH: "賦予5級[Bleed]威力", Note: "ok"},
		{Key: key("Other.json", 1), ZH: "賦予燒傷"},
	}, "test-model", 3)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(res.Accepted, []string{key("Other.json", 0)}) {
		t.Errorf("accepted = %v", res.Accepted)
	}
	if len(res.Rejected) != 1 || res.Rejected[0].Key != key("Other.json", 1) {
		t.Fatalf("rejected = %+v, want line 1", res.Rejected)
	}
	if !hasCode(res.Rejected[0].Issues, "id-mismatch") {
		t.Errorf("rejected issues = %+v, want id-mismatch", res.Rejected[0].Issues)
	}
	if len(res.NeedsReview) != 0 {
		t.Errorf("needs review = %v, want none after one failure", res.NeedsReview)
	}

	rec, ok := records(t, env)[key("Other.json", 0)]
	if !ok {
		t.Fatal("no record for the accepted unit")
	}
	want := store.Record{
		Key:    key("Other.json", 0),
		KRSha:  units.KRSha("[Bleed] 위력 5 부여"),
		ZH:     "賦予5級 [Bleed] 威力", // normalized
		Origin: store.OriginClaude,
		Since:  testTag,
		Rules:  "0123abc+dirty",
		Model:  "test-model",
		Batch:  b.ID,
	}
	if !reflect.DeepEqual(rec, want) {
		t.Errorf("record = %+v, want %+v", rec, want)
	}
	pending := pendingMap(t, env)
	if _, still := pending[key("Other.json", 0)]; still {
		t.Error("the accepted unit is still pending")
	}
	// A unit below its attempt limit keeps its lease, so its owner can send
	// the fix.
	failed, ok := pending[key("Other.json", 1)]
	if !ok || failed.Attempts != 1 || failed.Lease == nil || failed.Lease.Batch != b.ID ||
		!failed.Lease.Until.Equal(testNow.Add(120*time.Minute)) {
		t.Errorf("rejected pending item = %+v, want attempts 1 and the lease of %s unchanged", failed, b.ID)
	}
	if _, has := records(t, env)[key("Other.json", 1)]; has {
		t.Error("a rejected unit must not get a record")
	}
	if other := mustNext(t, env, NextOptions{}); other != nil {
		t.Errorf("Next = %+v, want nil: the rejected unit is still leased", other)
	}

	// The owner sends the fix.
	res, err = Submit(env, b, []OutUnit{{Key: key("Other.json", 1), ZH: "賦予[Burn]"}}, "test-model", 3)
	if err != nil || len(res.Accepted) != 1 || len(res.Rejected) != 0 {
		t.Fatalf("resubmit: %+v, %v", res, err)
	}
	if left := pendingMap(t, env); len(left) != 0 {
		t.Errorf("pending = %v, want empty", left)
	}
}

// Notes reach the store, those of rejected units too, once per key, batch
// and text.
func TestSubmitKeepsNotes(t *testing.T) {
	kr := map[string]string{"Other.json": `{"dataList":[
 {"id":0,"content":"[Bleed] 위력 5 부여"},
 {"id":1,"content":"[Burn] 부여"},
 {"id":2,"content":"그냥"}
]}`}
	env := newEnv(t, kr, nil, nil)
	seed(t, env, items("Other.json", 0, 3)...)
	b := mustNext(t, env, NextOptions{})
	out := []OutUnit{
		{Key: key("Other.json", 0), ZH: "賦予5級[Bleed]威力", Note: "「威力」依詞彙表"},
		{Key: key("Other.json", 1), ZH: "賦予燒傷", Note: "[Burn] 不知道放哪裡"}, // rejected: [Burn] is missing
		{Key: key("Other.json", 2), ZH: "就這樣"},                         // no note
		{Key: "Nowhere.json|/dataList/0/content", ZH: "x", Note: "not a unit of the batch"},
	}
	res, err := Submit(env, b, out, "test-model", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 2 || len(res.Rejected) != 2 {
		t.Fatalf("result = %+v, want 2 accepted and 2 rejected", res)
	}
	if res.Notes != 2 {
		t.Errorf("notes = %d, want 2", res.Notes)
	}
	want := []store.Note{
		{Key: key("Other.json", 0), Batch: b.ID, KRSha: units.KRSha("[Bleed] 위력 5 부여"), Model: "test-model", Note: "「威力」依詞彙表"},
		{Key: key("Other.json", 1), Batch: b.ID, KRSha: units.KRSha("[Burn] 부여"), Model: "test-model", Note: "[Burn] 不知道放哪裡"},
	}
	notes, err := env.Store.LoadNotes()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(notes, want) {
		t.Errorf("notes = %+v, want %+v", notes, want)
	}

	// The whole file again: the notes already stored are not added twice.
	res, err = Submit(env, b, out, "test-model", 3)
	if err != nil {
		t.Fatal(err)
	}
	if res.Notes != 0 {
		t.Errorf("notes of the resubmission = %d, want 0", res.Notes)
	}
	if notes, err := env.Store.LoadNotes(); err != nil || !reflect.DeepEqual(notes, want) {
		t.Errorf("notes after resubmitting = %+v, %v, want %+v", notes, err, want)
	}

	// The fix of the rejected unit comes with a new note: it is added.
	res, err = Submit(env, b, []OutUnit{{Key: key("Other.json", 1), ZH: "賦予[Burn]", Note: "補上 [Burn]"}}, "test-model", 3)
	if err != nil || len(res.Accepted) != 1 || res.Notes != 1 {
		t.Fatalf("fix: %+v, %v, want accepted with 1 note", res, err)
	}
	if notes, _ := env.Store.LoadNotes(); len(notes) != 3 || notes[2].Note != "補上 [Burn]" {
		t.Errorf("notes = %+v, want the new note last", notes)
	}
}

// A key that is not a unit key makes the whole out file unusable: it is an
// error, and nothing is written, not the notes of the other units and not
// their records either.
func TestSubmitNoteWithABadKeyWritesNothing(t *testing.T) {
	kr := map[string]string{"Other.json": `{"dataList":[
 {"id":0,"content":"[Bleed] 위력 5 부여"},
 {"id":1,"content":"그냥"}
]}`}
	tw := filepath.Join(t.TempDir(), "tw")
	env := newEnvIn(t, tw, kr, nil, nil)
	seed(t, env, items("Other.json", 0, 2)...)
	b := mustNext(t, env, NextOptions{})
	notesFile := filepath.Join(tw, ".meta", "notes.jsonl")

	for name, bad := range map[string]string{
		"no separator": "Other.json",
		"empty file":   "|/dataList/0/content",
		"bad pointer":  "Other.json|dataList/0",
		"bad escape":   "Other.json|/dataList/~2/content",
		"empty key":    "",
	} {
		out := []OutUnit{
			{Key: key("Other.json", 0), ZH: "賦予5級[Bleed]威力", Note: "「威力」依詞彙表"},
			{Key: bad, ZH: "x", Note: "a note on a key that is not a key"},
		}
		_, err := Submit(env, b, out, "test-model", 3)
		if err == nil {
			t.Errorf("%s: Submit with the key %q succeeded, want an error", name, bad)
			continue
		}
		if !errors.Is(err, ErrBadKey) || !strings.Contains(err.Error(), strconv.Quote(bad)) {
			t.Errorf("%s: err = %v, want ErrBadKey naming the key %q", name, err, bad)
		}
		if _, err := os.Stat(notesFile); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s: notes.jsonl exists after the failed submit (%v)", name, err)
		}
		if got := records(t, env); len(got) != 0 {
			t.Errorf("%s: records = %v, want none", name, got)
		}
		if got := pendingMap(t, env); len(got) != 2 {
			t.Errorf("%s: pending = %v, want both units still there", name, got)
		}
	}

	// With notes stored already, they stay as they are.
	good := []OutUnit{{Key: key("Other.json", 1), ZH: "就這樣", Note: "first"}}
	if _, err := Submit(env, b, good, "test-model", 3); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(notesFile)
	if err != nil {
		t.Fatal(err)
	}
	out := []OutUnit{
		{Key: key("Other.json", 0), ZH: "賦予5級[Bleed]威力", Note: "second"},
		{Key: "no-separator", ZH: "x"},
	}
	if _, err := Submit(env, b, out, "test-model", 3); !errors.Is(err, ErrBadKey) {
		t.Fatalf("err = %v, want ErrBadKey", err)
	}
	if after, err := os.ReadFile(notesFile); err != nil || string(after) != string(before) {
		t.Errorf("notes.jsonl changed: %q, %v, want %q", after, err, before)
	}
}

func hasCode(issues []validate.Issue, code string) bool {
	return slices.ContainsFunc(issues, func(is validate.Issue) bool { return is.Code == code })
}

func TestSubmitKeepsSinceAndDropsReviewFlag(t *testing.T) {
	kr := map[string]string{"Other.json": dataList(t, 1, "", line)}
	env := newEnv(t, kr, nil, nil)
	k := key("Other.json", 0)
	// A person requeued a unit that had been flagged: the flag is on the
	// record, the new pending item has none.
	seed(t, env, store.PendingItem{Key: k, Reason: "requeue"})
	if err := env.Store.SaveRecords(map[string]store.Record{
		k: {Key: k, KRSha: "old", ZH: "line 0", Origin: store.OriginFallback, Since: "raw-20260101", Flags: []string{"x", store.FlagNeedsReview}},
	}); err != nil {
		t.Fatal(err)
	}
	b := mustNext(t, env, NextOptions{})
	res, err := Submit(env, b, []OutUnit{{Key: k, ZH: "譯文 0"}}, "m", 3)
	if err != nil || len(res.Accepted) != 1 {
		t.Fatalf("submit: %+v, %v", res, err)
	}
	rec := records(t, env)[k]
	if rec.Since != "raw-20260101" || !slices.Equal(rec.Flags, []string{"x"}) || rec.Origin != store.OriginClaude {
		t.Errorf("record = %+v, want since kept, flags [x], origin claude", rec)
	}
	if rec.KRSha != units.KRSha(line(0)) {
		t.Errorf("kr_sha = %s, want the sha of the current KR", rec.KRSha)
	}
}

func TestSubmitRefusesAUnitFlaggedNeedsReview(t *testing.T) {
	kr := map[string]string{"Other.json": dataList(t, 1, "", line)}
	env := newEnv(t, kr, nil, nil)
	k := key("Other.json", 0)
	seed(t, env, store.PendingItem{Key: k, Reason: "new", Attempts: 3, Flags: []string{store.FlagNeedsReview}})
	b := &Batch{ID: "20261101T000000-deadbeef", File: "Other.json", Units: []BatchUnit{{Key: k, KR: line(0)}}}

	res, err := Submit(env, b, []OutUnit{{Key: k, ZH: "譯文 0"}}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 0 || len(res.Rejected) != 1 || !hasCode(res.Rejected[0].Issues, "needs-review") {
		t.Fatalf("result = %+v, want one needs-review rejection", res)
	}
	if is := res.Rejected[0].Issues[0]; is.Severity != validate.SeverityHard || is.Class != validate.ClassStructural {
		t.Errorf("issue = %+v, want hard and structural", is)
	}
	item := pendingMap(t, env)[k]
	if item.Attempts != 3 || !slices.Equal(item.Flags, []string{store.FlagNeedsReview}) {
		t.Errorf("pending item = %+v, want it untouched", item)
	}
	if _, has := records(t, env)[k]; has {
		t.Error("a unit waiting for a person must not get a record from a batch")
	}
}

func TestSubmitForeignName(t *testing.T) {
	kr := map[string]string{"Other.json": dataList(t, 1, "", func(int) string { return "뫼르소가 말했다" })}
	env := newEnv(t, kr, nil, nil)
	env.ForeignNames = []string{"默爾索", ""}
	seed(t, env, items("Other.json", 0, 1)...)
	b := mustNext(t, env, NextOptions{})

	res, err := Submit(env, b, []OutUnit{{Key: key("Other.json", 0), ZH: "默爾索說了話"}}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 1 || len(res.Rejected) != 0 {
		t.Fatalf("accepted %v rejected %v, want the unit accepted", res.Accepted, res.Rejected)
	}
	if len(res.Warnings) != 1 || res.Warnings[0].Key != key("Other.json", 0) || !hasCode(res.Warnings[0].Issues, "foreign-name") {
		t.Fatalf("warnings = %+v, want one foreign-name", res.Warnings)
	}
	is := res.Warnings[0].Issues[0]
	if is.Severity != "warn" || !strings.Contains(is.Message, "默爾索") {
		t.Errorf("issue = %+v, want a warn naming 默爾索", is)
	}
	if rec := records(t, env)[key("Other.json", 0)]; rec.ZH != "默爾索說了話" {
		t.Errorf("record = %+v, want it stored all the same", rec)
	}
}

func TestSubmitNeedsReview(t *testing.T) {
	kr := map[string]string{"Other.json": `{"dataList":[{"id":0,"content":"[Bleed] 위력"}]}`}
	en := map[string]string{"Other.json": `{"dataList":[{"id":0,"content":"[Bleed] Potency"}]}`}
	env := newEnv(t, kr, en, nil)
	env.RulesCommit = "0123abc"
	k := key("Other.json", 0)
	seed(t, env, items("Other.json", 0, 1)...)
	b := mustNext(t, env, NextOptions{})
	leaseUntil := testNow.Add(120 * time.Minute)

	// The owner of the batch keeps sending the unit; it keeps its lease until
	// the last attempt.
	for attempt := 1; attempt <= 3; attempt++ {
		res, err := Submit(env, b, []OutUnit{{Key: k, ZH: "賦予流血"}}, "m", 3) // [Bleed] is missing
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Rejected) != 1 || !hasCode(res.Rejected[0].Issues, "id-mismatch") {
			t.Fatalf("attempt %d: rejected = %+v", attempt, res.Rejected)
		}
		item := pendingMap(t, env)[k]
		last := attempt == 3
		if got := len(res.NeedsReview) == 1; got != last {
			t.Errorf("attempt %d: needs review = %v, want %v", attempt, res.NeedsReview, last)
		}
		if item.Attempts != attempt {
			t.Errorf("attempt %d: attempts = %d", attempt, item.Attempts)
		}
		// What was wrong with the last attempt, for the person who looks at
		// the unit once it waits for review.
		if !strings.HasPrefix(item.LastError, "id-mismatch: ") || strings.Contains(item.LastError, "\n") {
			t.Errorf("attempt %d: last error = %q, want the code and the message of the hard issue on one line", attempt, item.LastError)
		}
		if last {
			if item.Lease != nil || !slices.Equal(item.Flags, []string{store.FlagNeedsReview}) {
				t.Errorf("attempt 3: pending item = %+v, want no lease and needs_review", item)
			}
		} else if item.Lease == nil || item.Lease.Batch != b.ID || !item.Lease.Until.Equal(leaseUntil) || len(item.Flags) != 0 {
			t.Errorf("attempt %d: pending item = %+v, want the lease unchanged and no flag", attempt, item)
		}
	}

	rec, ok := records(t, env)[k]
	if !ok {
		t.Fatal("no fallback record")
	}
	if rec.Origin != store.OriginFallback || rec.ZH != "[Bleed] Potency" || !slices.Equal(rec.Flags, []string{store.FlagNeedsReview}) {
		t.Errorf("record = %+v, want fallback with the EN text and needs_review", rec)
	}
	if rec.KRSha != units.KRSha("[Bleed] 위력") || rec.Since != testTag || rec.Rules != "0123abc" || rec.Model != "" || rec.Batch != "" {
		t.Errorf("record = %+v, want the KR sha, since and rules, no model or batch", rec)
	}

	// A fourth submission of the same batch changes nothing, even with a
	// good translation: the unit waits for a person.
	res, err := Submit(env, b, []OutUnit{{Key: k, ZH: "賦予[Bleed]"}}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 0 || len(res.Rejected) != 1 || !hasCode(res.Rejected[0].Issues, "needs-review") || len(res.NeedsReview) != 0 {
		t.Errorf("fourth submit = %+v, want one needs-review rejection", res)
	}
	item := pendingMap(t, env)[k]
	if item.Attempts != 3 || !slices.Equal(item.Flags, []string{store.FlagNeedsReview}) || item.Lease != nil {
		t.Errorf("pending item = %+v, want attempts 3 and the flag in place", item)
	}
	if got := records(t, env)[k]; !reflect.DeepEqual(got, rec) {
		t.Errorf("record = %+v, want it unchanged: %+v", got, rec)
	}
	if b := mustNext(t, env, NextOptions{}); b != nil {
		t.Errorf("Next after needs_review = %+v, want nil", b)
	}
}

// A unit a person sent back (lct requeue) that fails every attempt must not
// lose its translation to the stand-in text, when that translation was made
// from the KR text there is now and is no stand-in itself. The unit waits for
// a person all the same.
func TestSubmitKeepsGoodRecordWhenRequeueFails(t *testing.T) {
	const kr = "[Bleed] 위력"
	krFiles := map[string]string{"Other.json": `{"dataList":[{"id":0,"content":"[Bleed] 위력"}]}`}
	enFiles := map[string]string{"Other.json": `{"dataList":[{"id":0,"content":"[Bleed] Potency"}]}`}
	k := key("Other.json", 0)

	// run requeues the unit over the record, fails it maxAttempts times and
	// returns the record and the pending item that are left.
	run := func(t *testing.T, existing store.Record) (store.Record, store.PendingItem) {
		t.Helper()
		env := newEnv(t, krFiles, enFiles, nil)
		env.RulesCommit = "0123abc"
		if err := env.Store.SaveRecords(map[string]store.Record{k: existing}); err != nil {
			t.Fatal(err)
		}
		seed(t, env, store.PendingItem{Key: k, Reason: "requeue", PrevZH: existing.ZH, Hint: "稱呼改用「您」"})
		b := mustNext(t, env, NextOptions{})
		for attempt := 1; attempt <= 3; attempt++ {
			res, err := Submit(env, b, []OutUnit{{Key: k, ZH: "賦予流血"}}, "m", 3) // [Bleed] is missing
			if err != nil {
				t.Fatal(err)
			}
			if len(res.Rejected) != 1 || !hasCode(res.Rejected[0].Issues, "id-mismatch") {
				t.Fatalf("attempt %d: rejected = %+v", attempt, res.Rejected)
			}
			if last := attempt == 3; (len(res.NeedsReview) == 1) != last || last && res.NeedsReview[0] != k {
				t.Fatalf("attempt %d: needs review = %v", attempt, res.NeedsReview)
			}
		}
		item := pendingMap(t, env)[k]
		if item.Attempts != 3 || item.Lease != nil || !slices.Equal(item.Flags, []string{store.FlagNeedsReview}) {
			t.Errorf("pending item = %+v, want attempts 3, no lease and needs_review", item)
		}
		return records(t, env)[k], item
	}
	good := func(origin string) store.Record {
		return store.Record{Key: k, KRSha: units.KRSha(kr), ZH: "賦予[Bleed]威力", Origin: origin, Since: "raw-20261001", Rules: "oldrules", Model: "old-model", Batch: "old-batch"}
	}

	for _, origin := range []string{store.OriginParatranz, store.OriginClaude, store.OriginHuman} {
		t.Run("a good "+origin+" record is kept", func(t *testing.T) {
			want := good(origin)
			got, _ := run(t, want)
			// Kept as it is, but for the flag that says a person must look.
			want.Flags = []string{store.FlagNeedsReview}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("record = %+v, want it unchanged but flagged: %+v", got, want)
			}
		})
	}

	// Every other case is as before: the stand-in text, flagged.
	t.Run("a stale record is replaced", func(t *testing.T) {
		stale := good(store.OriginParatranz)
		stale.KRSha = units.KRSha("다른 원문")
		got, _ := run(t, stale)
		if got.Origin != store.OriginFallback || got.ZH != "[Bleed] Potency" || got.KRSha != units.KRSha(kr) || !slices.Equal(got.Flags, []string{store.FlagNeedsReview}) {
			t.Errorf("record = %+v, want the fallback with the EN text and needs_review", got)
		}
		if got.Since != "raw-20261001" || got.Rules != "0123abc" {
			t.Errorf("record = %+v, want the old since and the current rules", got)
		}
	})
	t.Run("a fallback record is replaced", func(t *testing.T) {
		old := good(store.OriginFallback)
		old.ZH = "old stand-in"
		old.Flags = []string{store.FlagNeedsReview}
		got, _ := run(t, old)
		if got.Origin != store.OriginFallback || got.ZH != "[Bleed] Potency" || !slices.Equal(got.Flags, []string{store.FlagNeedsReview}) || got.Rules != "0123abc" {
			t.Errorf("record = %+v, want the fresh fallback with the EN text", got)
		}
	})
}

func TestSubmitFallbackWithoutEN(t *testing.T) {
	kr := map[string]string{
		"Other.json": dataList(t, 2, "", line),
		"Third.json": dataList(t, 1, "", line),
	}
	// Other.json 1 has an empty EN text; Third.json has no EN file.
	en := map[string]string{"Other.json": `{"dataList":[{"id":0,"content":"Line 0"},{"id":1,"content":""}]}`}
	env := newEnv(t, kr, en, nil)
	seed(t, env, append(items("Other.json", 0, 2), items("Third.json", 0, 1)...)...)
	for b := mustNext(t, env, NextOptions{}); b != nil; b = mustNext(t, env, NextOptions{}) {
		var out []OutUnit
		for _, u := range b.Units {
			out = append(out, OutUnit{Key: u.Key, ZH: ""}) // empty: always a hard failure
		}
		res, err := Submit(env, b, out, "m", 1) // maxAttempts 1: the first failure is the last
		if err != nil {
			t.Fatal(err)
		}
		if len(res.NeedsReview) != len(b.Units) {
			t.Fatalf("needs review = %v, want all of %v", res.NeedsReview, keysOf(b))
		}
	}
	recs := records(t, env)
	for k, want := range map[string]string{
		key("Other.json", 0): "Line 0", // the EN text
		key("Other.json", 1): line(1),  // EN is empty: the KR text
		key("Third.json", 0): line(0),  // no EN file: the KR text
	} {
		if got := recs[k]; got.ZH != want || got.Origin != store.OriginFallback {
			t.Errorf("record %s = %+v, want fallback %q", k, got, want)
		}
	}
}

func TestSubmitLeaseAndBatchMembership(t *testing.T) {
	kr := map[string]string{"Other.json": dataList(t, 4, "", line), "Third.json": dataList(t, 1, "", line)}
	env := newEnv(t, kr, nil, nil)
	seed(t, env, append(items("Other.json", 0, 4), items("Third.json", 0, 1)...)...)

	a := mustNext(t, env, NextOptions{MaxUnits: 3}) // Other 0-2
	other := env
	other.Now = testNow.Add(3 * time.Hour) // a's leases ran out
	c := mustNext(t, other, NextOptions{MaxUnits: 1})
	if c == nil || c.Units[0].Key != key("Other.json", 0) {
		t.Fatalf("batch after expiry = %+v, want unit 0 again", c)
	}

	out := []OutUnit{
		{Key: key("Other.json", 0), ZH: "譯文 0"},  // its lease is c's now
		{Key: key("Other.json", 1), ZH: "譯文 1"},  // lease ran out but nobody took it
		{Key: key("Other.json", 1), ZH: "譯文 1b"}, // twice
		{Key: key("Third.json", 0), ZH: "譯文 3"},  // not in a
		{Key: key("Other.json", 2), ZH: "譯文 2"},  // fine
		{Key: "Nowhere.json|/dataList/0/content", ZH: "x"},
	}
	res, err := Submit(env, a, out, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{key("Other.json", 1), key("Other.json", 2)}; !slices.Equal(res.Accepted, want) {
		t.Errorf("accepted = %v, want %v", res.Accepted, want)
	}
	codes := map[string]string{}
	for _, r := range res.Rejected {
		if len(r.Issues) != 1 {
			t.Errorf("rejection %+v: want exactly one issue", r)
			continue
		}
		codes[r.Key] += r.Issues[0].Code + ";"
	}
	want := map[string]string{
		key("Other.json", 0):               "lease-lost;",
		key("Other.json", 1):               "duplicate-key;",
		key("Third.json", 0):               "not-in-batch;",
		"Nowhere.json|/dataList/0/content": "not-in-batch;",
	}
	if !reflect.DeepEqual(codes, want) {
		t.Errorf("rejection codes = %v, want %v", codes, want)
	}
	pending := pendingMap(t, env)
	if it := pending[key("Other.json", 0)]; it.Attempts != 0 || it.Lease == nil || it.Lease.Batch != c.ID {
		t.Errorf("unit 0 = %+v, want c's lease kept and attempts untouched", it)
	}
	if it := pending[key("Third.json", 0)]; it.Attempts != 0 {
		t.Errorf("a key outside the batch was touched: %+v", it)
	}
}

// A batch file can be edited: only the units a batch holds a lease on are its
// own. A free unit, never handed out to it, is no unit of the batch.
func TestSubmitRefusesUnitsTheBatchDoesNotHold(t *testing.T) {
	kr := map[string]string{"Other.json": dataList(t, 2, "", line)}
	env := newEnv(t, kr, nil, nil)
	seed(t, env, items("Other.json", 0, 2)...)
	b := mustNext(t, env, NextOptions{MaxUnits: 1}) // unit 0 only
	// The file is edited to claim unit 1 as well, which is still free.
	b.Units = append(b.Units, BatchUnit{Key: key("Other.json", 1), KR: line(1)})

	res, err := Submit(env, b, translate(b), "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{key("Other.json", 0)}; !slices.Equal(res.Accepted, want) {
		t.Errorf("accepted = %v, want %v", res.Accepted, want)
	}
	if len(res.Rejected) != 1 || res.Rejected[0].Key != key("Other.json", 1) || !hasCode(res.Rejected[0].Issues, CodeLeaseLost) {
		t.Fatalf("rejected = %+v, want unit 1 with lease-lost", res.Rejected)
	}
	if _, has := records(t, env)[key("Other.json", 1)]; has {
		t.Error("a unit the batch does not hold got a record")
	}
	if it := pendingMap(t, env)[key("Other.json", 1)]; it.Attempts != 0 || it.Lease != nil {
		t.Errorf("unit 1 = %+v, want it untouched", it)
	}
}

func TestSubmitReleasesOmittedUnits(t *testing.T) {
	kr := map[string]string{"Other.json": dataList(t, 3, "", line)}
	env := newEnv(t, kr, nil, nil)
	seed(t, env, items("Other.json", 0, 3)...)
	b := mustNext(t, env, NextOptions{})

	res, err := Submit(env, b, []OutUnit{{Key: key("Other.json", 0), ZH: "譯文 0"}}, "m", 3)
	if err != nil || len(res.Accepted) != 1 || len(res.Rejected) != 0 {
		t.Fatalf("submit: %+v, %v", res, err)
	}
	if want := []string{key("Other.json", 1), key("Other.json", 2)}; !slices.Equal(res.Released, want) {
		t.Errorf("released = %v, want %v", res.Released, want)
	}
	pending := pendingMap(t, env)
	for _, i := range []int{1, 2} {
		it, ok := pending[key("Other.json", i)]
		if !ok || it.Lease != nil || it.Attempts != 0 {
			t.Errorf("omitted unit %d = %+v, want pending, released and attempts 0", i, it)
		}
	}
	if b2 := mustNext(t, env, NextOptions{}); b2 == nil || len(b2.Units) != 2 {
		t.Errorf("batch after release = %+v, want the two omitted units", b2)
	}

	// Units that are no longer pending and leased to the batch are not
	// reported: unit 0 was accepted before, unit 1 now belongs to b2.
	res, err = Submit(env, b, nil, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Released) != 0 {
		t.Errorf("released = %v, want none: the units are no longer leased to the batch", res.Released)
	}
}

func TestSubmitIgnoresLeaseOfAnotherBatchOnOmitted(t *testing.T) {
	kr := map[string]string{"Other.json": dataList(t, 2, "", line)}
	env := newEnv(t, kr, nil, nil)
	seed(t, env, items("Other.json", 0, 2)...)
	a := mustNext(t, env, NextOptions{})
	later := env
	later.Now = testNow.Add(3 * time.Hour)
	c := mustNext(t, later, NextOptions{})

	// a submits nothing: c's leases are not its to release.
	res, err := Submit(env, a, nil, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Released) != 0 {
		t.Errorf("released = %v, want none", res.Released)
	}
	for _, it := range pendingMap(t, env) {
		if it.Lease == nil || it.Lease.Batch != c.ID {
			t.Errorf("item %+v lost the lease of batch %s", it, c.ID)
		}
	}
}

func TestSubmitKRGone(t *testing.T) {
	kr := map[string]string{"Other.json": dataList(t, 1, "", line)}
	env := newEnv(t, kr, nil, nil)
	gone := "Other.json|/dataList/5/content"
	seed(t, env, store.PendingItem{Key: gone, Reason: "new", Lease: &store.Lease{Batch: "b", Until: testNow.Add(time.Hour)}})
	res, err := Submit(env, &Batch{ID: "b", File: "Other.json", Units: []BatchUnit{{Key: gone, KR: "x"}}}, []OutUnit{{Key: gone, ZH: "譯文"}}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rejected) != 1 || res.Rejected[0].Issues[0].Code != "kr-missing" || len(res.Accepted) != 0 {
		t.Errorf("result = %+v, want one kr-missing rejection", res)
	}
	if it := pendingMap(t, env)[gone]; it.Attempts != 0 || it.Lease != nil {
		t.Errorf("pending item = %+v, want no attempt and the lease released", it)
	}
}

func TestSubmitKRChanged(t *testing.T) {
	kr := map[string]string{"Other.json": dataList(t, 2, "", line)}
	env := newEnv(t, kr, nil, nil)
	seed(t, env, items("Other.json", 0, 2)...)
	b := mustNext(t, env, NextOptions{})
	// The raw snapshot moved on after the batch was cut: unit 0 was made
	// from another KR text.
	b.Units[0].KR = "옛 원문"

	res, err := Submit(env, b, translate(b), "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{key("Other.json", 1)}; !slices.Equal(res.Accepted, want) {
		t.Errorf("accepted = %v, want %v", res.Accepted, want)
	}
	if len(res.Rejected) != 1 || res.Rejected[0].Key != key("Other.json", 0) || !hasCode(res.Rejected[0].Issues, "kr-changed") {
		t.Fatalf("rejected = %+v, want unit 0 with kr-changed", res.Rejected)
	}
	if is := res.Rejected[0].Issues[0]; is.Severity != validate.SeverityHard || is.Class != validate.ClassStructural {
		t.Errorf("issue = %+v, want hard and structural", is)
	}
	item := pendingMap(t, env)[key("Other.json", 0)]
	if item.Attempts != 0 || item.Lease != nil {
		t.Errorf("pending item = %+v, want no attempt and the lease released for a fresh batch", item)
	}
	if _, has := records(t, env)[key("Other.json", 0)]; has {
		t.Error("a translation of another KR text must not become a record")
	}
	if again := mustNext(t, env, NextOptions{}); again == nil || again.Units[0].KR != line(0) {
		t.Errorf("Next = %+v, want unit 0 again with the current KR", again)
	}
}

func TestSubmitIsIdempotent(t *testing.T) {
	kr := map[string]string{"Other.json": dataList(t, 3, "", line)}
	env := newEnv(t, kr, nil, nil)
	seed(t, env, items("Other.json", 0, 3)...)
	b := mustNext(t, env, NextOptions{})
	out := translate(b)
	good := out[2].ZH
	out[2].ZH = "" // rejected the first time

	res, err := Submit(env, b, out, "first", 3)
	if err != nil || len(res.Accepted) != 2 || len(res.Rejected) != 1 {
		t.Fatalf("first submit: %+v, %v", res, err)
	}
	before := records(t, env)

	// The whole out file is sent again with the fix.
	out[2].ZH = good
	res, err = Submit(env, b, out, "second", 3)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{key("Other.json", 0), key("Other.json", 1), key("Other.json", 2)}; !slices.Equal(res.Accepted, want) {
		t.Errorf("accepted = %v, want all of %v", res.Accepted, want)
	}
	if len(res.Rejected) != 0 || len(res.Warnings) != 0 {
		t.Errorf("rejected %v warnings %v, want none", res.Rejected, res.Warnings)
	}
	after := records(t, env)
	for _, i := range []int{0, 1} {
		if !reflect.DeepEqual(after[key("Other.json", i)], before[key("Other.json", i)]) {
			t.Errorf("record %d was rewritten: %+v, was %+v", i, after[key("Other.json", i)], before[key("Other.json", i)])
		}
	}
	if after[key("Other.json", 0)].Model != "first" || after[key("Other.json", 2)].Model != "second" {
		t.Errorf("models = %q, %q, want first and second", after[key("Other.json", 0)].Model, after[key("Other.json", 2)].Model)
	}

	// A third time: all three are records of this batch now.
	res, err = Submit(env, b, out, "third", 3)
	if err != nil || len(res.Accepted) != 3 || len(res.Rejected) != 0 {
		t.Errorf("third submit: %+v, %v", res, err)
	}
	if !reflect.DeepEqual(records(t, env), after) {
		t.Error("the third submission changed the records")
	}

	// A record of another batch is no proof that this one was accepted.
	foreign := key("Other.json", 1)
	recs := records(t, env)
	rec := recs[foreign]
	rec.Batch = "20250101T000000-00000000"
	recs[foreign] = rec
	if err := env.Store.SaveRecords(recs); err != nil {
		t.Fatal(err)
	}
	res, err = Submit(env, b, []OutUnit{{Key: foreign, ZH: "譯文 1"}}, "m", 3)
	if err != nil || len(res.Accepted) != 0 || len(res.Rejected) != 1 || !hasCode(res.Rejected[0].Issues, "lease-lost") {
		t.Errorf("submit of a unit translated by another batch: %+v, %v, want lease-lost", res, err)
	}
}

func TestSubmitArguments(t *testing.T) {
	env := newEnv(t, map[string]string{"Other.json": dataList(t, 1, "", line)}, nil, nil)
	if _, err := Submit(env, nil, nil, "m", 3); err == nil {
		t.Error("Submit with no batch = nil error")
	}
	if _, err := Submit(env, &Batch{}, nil, "m", 3); err == nil {
		t.Error("Submit with a batch without an id = nil error")
	}
	if _, err := Submit(env, &Batch{ID: "b"}, nil, "m", -1); err == nil {
		t.Error("Submit with negative maxAttempts = nil error")
	}
	if _, err := Next(Env{}, NextOptions{}); err == nil {
		t.Error("Next with an empty Env = nil error")
	}
	if _, err := Next(env, NextOptions{MaxUnits: -1}); err == nil {
		t.Error("Next with a negative limit = nil error")
	}
}

func TestResultJSONShape(t *testing.T) {
	env := newEnv(t, map[string]string{"Other.json": dataList(t, 1, "", line)}, nil, nil)
	res, err := Submit(env, &Batch{ID: "20261101T000000-deadbeef"}, nil, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"accepted":[],"reused":[],"rejected":[],"warnings":[],"needs_review":[],"released":[],"notes":0,"terms":0,"terms_skipped":[]}`; string(data) != want {
		t.Errorf("result JSON = %s, want %s", data, want)
	}

	// Spec 9.1: a rejection is {key, errors}.
	data, err = json.Marshal(Rejection{Key: "k", Issues: []validate.Issue{{Code: "c", Severity: validate.SeverityHard, Class: validate.ClassStructural, Message: "m"}}})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"key":"k","errors":[{"code":"c","severity":"hard","class":"structural","message":"m"}]}`; string(data) != want {
		t.Errorf("rejection JSON = %s, want %s", data, want)
	}
}

func TestNextKeepsBatchIDsUnique(t *testing.T) {
	k0, k1 := key("Other.json", 0), key("Other.json", 1)
	base := batchID(testNow, k0)
	tests := []struct {
		name  string
		items []store.PendingItem
		taken func(string) (bool, error)
		want  string
	}{
		{"no clash", []store.PendingItem{{Key: k0, Reason: "new"}}, nil, base},
		{
			"a live lease holds the id",
			[]store.PendingItem{{Key: k0, Reason: "new"}, {Key: k1, Reason: "new", Lease: &store.Lease{Batch: base, Until: testNow.Add(time.Hour)}}},
			nil, base + "-2",
		},
		{
			"an old lease still names the id",
			[]store.PendingItem{{Key: k0, Reason: "new"}, {Key: k1, Reason: "new", Lease: &store.Lease{Batch: base, Until: testNow.Add(-time.Hour)}}},
			nil, base + "-2",
		},
		{
			"the caller knows a file with the id",
			[]store.PendingItem{{Key: k0, Reason: "new"}},
			func(id string) (bool, error) { return id == base, nil }, base + "-2",
		},
		{
			"both the lease and the caller",
			[]store.PendingItem{{Key: k0, Reason: "new"}, {Key: k1, Reason: "new", Lease: &store.Lease{Batch: base + "-2", Until: testNow.Add(time.Hour)}}},
			func(id string) (bool, error) { return id == base, nil }, base + "-3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newEnv(t, map[string]string{"Other.json": dataList(t, 2, "", line)}, nil, nil)
			seed(t, env, tt.items...)
			b := mustNext(t, env, NextOptions{IDTaken: tt.taken})
			// (A lease that ran out frees its unit, so unit 1 may come along.)
			if b == nil || b.ID != tt.want || b.Units[0].Key != k0 {
				t.Fatalf("batch = %+v, want id %s starting with unit 0", b, tt.want)
			}
			if lease := pendingMap(t, env)[k0].Lease; lease == nil || lease.Batch != tt.want {
				t.Errorf("lease = %+v, want the batch id %s", lease, tt.want)
			}
		})
	}
}

// When IDTaken cannot tell, or says every id is taken, Next fails instead of
// leasing anything (or searching forever under the lock).
func TestNextFailsWhenNoIDIsFree(t *testing.T) {
	k0 := key("Other.json", 0)
	base := batchID(testNow, k0)
	broken := errors.New("cannot look")
	tests := []struct {
		name  string
		taken func(string) (bool, error)
		want  string // in the error
		is    error  // wrapped by the error, if not nil
	}{
		{"the check fails", func(string) (bool, error) { return false, broken }, "cannot look", broken},
		{"every id is taken", func(string) (bool, error) { return true, nil }, base, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newEnv(t, map[string]string{"Other.json": dataList(t, 1, "", line)}, nil, nil)
			seed(t, env, items("Other.json", 0, 1)...)
			b, err := Next(env, NextOptions{IDTaken: tt.taken})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Next = %+v, %v, want an error mentioning %q", b, err, tt.want)
			}
			if tt.is != nil && !errors.Is(err, tt.is) {
				t.Errorf("err = %v, want it to wrap %v", err, tt.is)
			}
			if lease := pendingMap(t, env)[k0].Lease; lease != nil {
				t.Errorf("lease = %+v, want none", lease)
			}
		})
	}

	// The last suffix tried is -1001: a thousand suffixes after the plain id.
	env := newEnv(t, map[string]string{"Other.json": dataList(t, 1, "", line)}, nil, nil)
	seed(t, env, items("Other.json", 0, 1)...)
	last := base + "-1001"
	b, err := Next(env, NextOptions{IDTaken: func(id string) (bool, error) { return id != last, nil }})
	if err != nil || b == nil || b.ID != last {
		t.Errorf("Next = %+v, %v, want batch %s", b, err, last)
	}
}

// schemaProps reads the property names a schema object declares, and its
// required list.
func schemaProps(t *testing.T, schema map[string]any) (props map[string]map[string]any, required []string) {
	t.Helper()
	raw, _ := schema["properties"].(map[string]any)
	props = map[string]map[string]any{}
	for name, v := range raw {
		props[name], _ = v.(map[string]any)
	}
	if req, ok := schema["required"].([]any); ok {
		for _, r := range req {
			required = append(required, r.(string))
		}
	}
	return props, required
}

func loadSchema(t *testing.T, name string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "schemas", name))
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return schema
}

// checkObject compares the keys of a marshaled value to a schema object:
// every key is declared, every required property is there.
func checkObject(t *testing.T, where string, got map[string]any, schema map[string]any) {
	t.Helper()
	props, required := schemaProps(t, schema)
	for k := range got {
		if _, ok := props[k]; !ok {
			t.Errorf("%s: property %q is not in the schema", where, k)
		}
	}
	for _, r := range required {
		if _, ok := got[r]; !ok {
			t.Errorf("%s: required property %q is missing", where, r)
		}
	}
	if schema["additionalProperties"] != false {
		t.Errorf("%s: the schema should say additionalProperties: false", where)
	}
}

func TestSchemasMatchTheTypes(t *testing.T) {
	const file = "StoryData/S001A.json"
	kr := map[string]string{
		file: `{"dataList":[
 {"id":0,"model":"파우스트","content":"[Bleed] 위력 5 부여"},
 {"id":1,"model":"파우스트","content":"<color=#f8c200>공격</color> {0}"},
 {"id":2,"model":"파우스트","content":"끝"}
]}`,
		"Bufs.json": `{"dataList":[{"id":"Bleed","name":"출혈"}]}`,
	}
	en := map[string]string{file: `{"dataList":[
 {"id":0,"model":"파우스트","content":"Inflict 5 [Bleed] Potency"},
 {"id":1,"model":"파우스트","content":"Attack {0}"},
 {"id":2,"model":"파우스트","content":"The end"}
]}`}
	jp := map[string]string{file: `{"dataList":[
 {"id":0,"model":"파우스트","content":"[Bleed]を5付与"},
 {"id":1,"model":"파우스트","content":"攻撃 {0}"},
 {"id":2,"model":"파우스트","content":"終"}
]}`}
	env := newEnv(t, kr, en, jp)
	// A term for some files, so that the term's files are in the batch.
	env.Rules = testRulesWith(t, "- kr: 위력\n  zh: 威力\n  status: approved\n  forbidden: [力量]\n  files: [\"StoryData/S*\"]\n")
	pending := items(file, 0, 2)
	pending[0].Refs = map[string]string{"llc": "a", "ltm": "b"}
	pending[0].PrevKR, pending[0].PrevZH, pending[0].Draft = "p", "q", "r"
	pending[0].Hint = "s"
	seed(t, env, pending...)
	if err := env.Store.SaveRecords(map[string]store.Record{
		"Bufs.json|/dataList/0/name": {Key: "Bufs.json|/dataList/0/name", KRSha: units.KRSha("출혈"), ZH: "流血", Origin: store.OriginParatranz, Since: testTag},
		key(file, 2):                 {Key: key(file, 2), KRSha: units.KRSha("끝"), ZH: "終", Origin: store.OriginParatranz, Since: testTag},
	}); err != nil {
		t.Fatal(err)
	}
	b := mustNext(t, env, NextOptions{})
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]any
	if err := json.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}

	schema := loadSchema(t, "batch.schema.json")
	checkObject(t, "batch", top, schema)
	props, _ := schemaProps(t, schema)
	idPattern := regexp.MustCompile(props["id"]["pattern"].(string))
	for _, id := range []string{b.ID, b.ID + "-2", b.ID + "-13"} {
		if !idPattern.MatchString(id) {
			t.Errorf("the schema's id pattern %s rejects %s", idPattern, id)
		}
	}
	if idPattern.MatchString(b.ID+"-0") || idPattern.MatchString(b.ID+"-x") {
		t.Errorf("the schema's id pattern %s accepts a bad suffix", idPattern)
	}
	unitSchema := props["units"]["items"].(map[string]any)
	ctxSchema := props["context"]["items"].(map[string]any)
	glossSchema := props["glossary"]["items"].(map[string]any)
	unitProps, _ := schemaProps(t, unitSchema)
	enSchema := unitProps["en"]
	keepSchema := unitProps["keep"]

	// Every optional property of a unit is set by the fixture, so the check
	// below sees it.
	first := top["units"].([]any)[0].(map[string]any)
	for name := range unitProps {
		if _, ok := first[name]; !ok {
			t.Errorf("the fixture's first unit has no %q, so the schema's property is not checked", name)
		}
	}
	if hint, ok := unitProps["hint"]; !ok || hint["type"] != "string" {
		t.Errorf("the unit schema's hint = %v, want a string property", hint)
	}
	// A batch names its files, never none, and every unit its own.
	if _, required := schemaProps(t, schema); !slices.Contains(required, "files") {
		t.Errorf("the batch schema does not require files: %v", required)
	}
	files := props["files"]
	fileItems, _ := files["items"].(map[string]any)
	if files["type"] != "array" || files["minItems"] != float64(1) || fileItems["type"] != "string" {
		t.Errorf("the batch schema's files = %v, want an array of at least one string", files)
	}
	if _, required := schemaProps(t, unitSchema); !slices.Contains(required, "file") || unitProps["file"]["type"] != "string" {
		t.Errorf("the unit schema's file = %v (required %v), want a required string", unitProps["file"], required)
	}

	for i, u := range top["units"].([]any) {
		um := u.(map[string]any)
		checkObject(t, fmt.Sprintf("unit %d", i), um, unitSchema)
		if v, ok := um["en"]; ok {
			checkObject(t, "en", v.(map[string]any), enSchema)
		}
		if v, ok := um["jp"]; ok {
			checkObject(t, "jp", v.(map[string]any), unitProps["jp"])
		}
		checkObject(t, "keep", um["keep"].(map[string]any), keepSchema)
	}
	if len(top["context"].([]any)) == 0 {
		t.Fatal("the fixture should produce a context line")
	}
	for _, c := range top["context"].([]any) {
		checkObject(t, "context", c.(map[string]any), ctxSchema)
	}
	if len(top["glossary"].([]any)) == 0 {
		t.Fatal("the fixture should produce a glossary term")
	}
	for _, g := range top["glossary"].([]any) {
		term := g.(map[string]any)
		checkObject(t, "glossary", term, glossSchema)
		if _, ok := term["files"]; !ok {
			t.Errorf("the fixture's term has no files, so the schema's property is not checked: %v", term)
		}
	}

	// The glossary items are rules.Term values: the schema declares exactly
	// the fields Term writes as JSON (match and provisional included).
	var wantTerm []string
	termType := reflect.TypeOf(rules.Term{})
	for i := 0; i < termType.NumField(); i++ {
		if name, _, _ := strings.Cut(termType.Field(i).Tag.Get("json"), ","); name != "" && name != "-" {
			wantTerm = append(wantTerm, name)
		}
	}
	glossProps, _ := schemaProps(t, glossSchema)
	gotTerm := slices.Sorted(maps.Keys(glossProps))
	slices.Sort(wantTerm)
	if !slices.Equal(gotTerm, wantTerm) {
		t.Errorf("batch.schema.json glossary item properties = %v, want the json fields of rules.Term %v", gotTerm, wantTerm)
	}

	// A batch survives the trip through its file.
	var back Batch
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(&back, b) {
		t.Errorf("round trip changed the batch:\n got  %+v\n want %+v", back, *b)
	}

	// The output schema: an array of {key, zh, note?, terms?}.
	outSchema := loadSchema(t, "batch-out.schema.json")
	if outSchema["type"] != "array" {
		t.Errorf("batch-out schema type = %v, want array", outSchema["type"])
	}
	outProps, outRequired := schemaProps(t, outSchema["items"].(map[string]any))
	sort.Strings(outRequired)
	if !slices.Equal(outRequired, []string{"key", "zh"}) {
		t.Errorf("batch-out required = %v, want key and zh", outRequired)
	}
	omsg, _ := json.Marshal(OutUnit{Key: "k", ZH: "z", Note: "n", Terms: []NewTerm{{KR: "kr", ZH: "zh"}}})
	var om map[string]any
	if err := json.Unmarshal(omsg, &om); err != nil {
		t.Fatal(err)
	}
	for k := range om {
		if _, ok := outProps[k]; !ok {
			t.Errorf("OutUnit property %q is not in the batch-out schema", k)
		}
	}
	if len(outProps) != len(om) {
		t.Errorf("batch-out declares %d properties, OutUnit has %d", len(outProps), len(om))
	}
}

// The tests of twins (identical pending lines) and of the glossary by file.

// twinText is a line long enough to be shared between files (ShortTextRunes).
const twinText = "같은 문장이 두 군데에 있다"

// oneLine is a file with one line, id 0, saying text.
func oneLine(t *testing.T, text string) string {
	t.Helper()
	return dataList(t, 1, "", func(int) string { return text })
}

func TestNextDefersTwinsOfLeasedUnits(t *testing.T) {
	kr := map[string]string{
		"A.json": oneLine(t, twinText),
		"B.json": `{"dataList":[{"id":0,"content":` + q(t, twinText) + `},{"id":1,"content":"다른 문장"}]}`,
	}
	env := newEnv(t, kr, nil, nil)
	seed(t, env, append(items("A.json", 0, 1), items("B.json", 0, 2)...)...)

	// A batch of one unit, so that B.json is not in it.
	b1 := mustNext(t, env, NextOptions{Family: "other", MaxUnits: 1})
	if b1 == nil || !slices.Equal(b1.Files, []string{"A.json"}) {
		t.Fatalf("first batch = %+v, want A.json", b1)
	}
	// B's first line says what A's line says, and A's line is out for translation.
	b2 := mustNext(t, env, NextOptions{Family: "other"})
	if b2 == nil || !slices.Equal(keysOf(b2), []string{key("B.json", 1)}) {
		t.Fatalf("second batch = %+v, want only the other line of B.json", b2)
	}
	if it := pendingMap(t, env)[key("B.json", 0)]; it.Lease != nil {
		t.Errorf("the twin is leased: %+v", it.Lease)
	}
	if b3 := mustNext(t, env, NextOptions{Family: "other"}); b3 != nil {
		t.Errorf("third batch = %+v, want nil: only the twin is left", b3)
	}

	// Once the lease of the first batch has run out, its line is handed out
	// again and the twin goes on waiting for it.
	later := env
	later.Now = testNow.Add(3 * time.Hour)
	if b4 := mustNext(t, later, NextOptions{Family: "other"}); b4 == nil || b4.File != "A.json" {
		t.Errorf("batch after expiry = %+v, want A.json again", b4)
	}
}

// A lease that has run out does not hold the twin back.
func TestNextIgnoresExpiredLeasesOfTwins(t *testing.T) {
	kr := map[string]string{"A.json": oneLine(t, twinText), "B.json": oneLine(t, twinText)}
	env := newEnv(t, kr, nil, nil)
	pending := append(items("A.json", 0, 1), items("B.json", 0, 1)...)
	pending[1].Lease = &store.Lease{Batch: "20261101T000000-deadbeef", Until: testNow.Add(-time.Minute)}
	seed(t, env, pending...)
	if b := mustNext(t, env, NextOptions{}); b == nil || b.File != "A.json" {
		t.Fatalf("batch = %+v, want A.json: the lease of its twin has run out", b)
	}

	// The same twin with a lease that is still good holds A.json's line back.
	env = newEnv(t, kr, nil, nil)
	pending[1].Lease = &store.Lease{Batch: "20261101T000000-deadbeef", Until: testNow.Add(time.Minute)}
	seed(t, env, pending...)
	if b := mustNext(t, env, NextOptions{}); b != nil {
		t.Errorf("batch = %+v, want nil: the twin is leased to another batch", b)
	}
}

// Texts under ShortTextRunes are twins only within a file, so a short line is
// handed out although the same line is out for translation in another file.
func TestNextDoesNotDeferShortTextsOfOtherFiles(t *testing.T) {
	kr := map[string]string{"A.json": oneLine(t, "확인"), "B.json": oneLine(t, "확인")}
	env := newEnv(t, kr, nil, nil)
	seed(t, env, append(items("A.json", 0, 1), items("B.json", 0, 1)...)...)
	// Batches of one unit, so that the files are in batches of their own (in
	// one batch, see TestMergedBatchHoldsOneOfTwins).
	if b := mustNext(t, env, NextOptions{MaxUnits: 1}); b == nil || b.File != "A.json" {
		t.Fatalf("first batch = %+v, want A.json", b)
	}
	if b := mustNext(t, env, NextOptions{MaxUnits: 1}); b == nil || b.File != "B.json" {
		t.Errorf("second batch = %+v, want B.json", b)
	}
}

// Of identical lines in one file only one goes into a batch; the others are
// filled in when it is translated.
func TestNextHandsOutOneLineOfIdenticalLinesOfAFile(t *testing.T) {
	kr := map[string]string{"A.json": `{"dataList":[
 {"id":0,"content":"확인"},
 {"id":1,"content":"다른 말"},
 {"id":2,"content":"확인"},
 {"id":3,"content":"확인"}
]}`}
	env := newEnv(t, kr, nil, nil)
	seed(t, env, items("A.json", 0, 4)...)
	b := mustNext(t, env, NextOptions{})
	if want := []string{key("A.json", 0), key("A.json", 1)}; b == nil || !slices.Equal(keysOf(b), want) {
		t.Fatalf("batch = %+v, want the keys %v", b, want)
	}
	pending := pendingMap(t, env)
	for _, i := range []int{2, 3} {
		if pending[key("A.json", i)].Lease != nil {
			t.Errorf("line %d is leased", i)
		}
	}

	res, err := Submit(env, b, translate(b), "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{key("A.json", 2), key("A.json", 3)}; !slices.Equal(res.Reused, want) {
		t.Errorf("reused = %v, want %v", res.Reused, want)
	}
	if recs := records(t, env); recs[key("A.json", 2)].ZH != "譯文 0" || recs[key("A.json", 3)].ZH != "譯文 0" {
		t.Errorf("records of the twins = %+v, want the translation of line 0", recs)
	}
}

func TestSubmitReusesForPendingTwins(t *testing.T) {
	kr := map[string]string{"A.json": oneLine(t, twinText), "B.json": oneLine(t, twinText), "C.json": oneLine(t, twinText)}
	env := newEnv(t, kr, nil, nil)
	env.RulesCommit = "0123abc"
	seed(t, env, append(append(items("A.json", 0, 1), items("B.json", 0, 1)...), items("C.json", 0, 1)...)...)
	// C was translated from another text once, in an earlier raw snapshot.
	oldC := store.Record{Key: key("C.json", 0), KRSha: units.KRSha("옛 문장"), ZH: "舊句", Origin: store.OriginParatranz, Since: "raw-20260101"}
	if err := env.Store.SaveRecords(map[string]store.Record{oldC.Key: oldC}); err != nil {
		t.Fatal(err)
	}

	b := mustNext(t, env, NextOptions{MaxUnits: 1})
	if b == nil || b.File != "A.json" || len(b.Units) != 1 {
		t.Fatalf("batch = %+v, want A.json's line", b)
	}
	res, err := Submit(env, b, []OutUnit{{Key: key("A.json", 0), ZH: "同一句話"}}, "test-model", 3)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(res.Accepted, []string{key("A.json", 0)}) {
		t.Errorf("accepted = %v, want only the submitted line", res.Accepted)
	}
	if want := []string{key("B.json", 0), key("C.json", 0)}; !slices.Equal(res.Reused, want) {
		t.Errorf("reused = %v, want %v", res.Reused, want)
	}

	recs := records(t, env)
	for _, k := range []string{key("B.json", 0), key("C.json", 0)} {
		since := testTag
		if k == oldC.Key {
			since = "raw-20260101" // the unit keeps the tag it was first translated under
		}
		want := store.Record{
			Key: k, KRSha: units.KRSha(twinText), ZH: "同一句話", Origin: store.OriginClaude,
			Since: since, Rules: "0123abc", Model: "test-model", Batch: b.ID,
		}
		if !reflect.DeepEqual(recs[k], want) {
			t.Errorf("record of %s = %+v, want %+v", k, recs[k], want)
		}
	}
	if left := pendingMap(t, env); len(left) != 0 {
		t.Errorf("pending = %v, want empty", left)
	}
	if _, err := json.Marshal(res); err != nil {
		t.Fatal(err)
	}
}

// What makes two units twins is their TextGroup (and the file, for a short
// text); even then a unit is filled only when nothing speaks against it.
func TestSubmitReusesOnlyForTwinsInTheSameGroup(t *testing.T) {
	soon := testNow.Add(time.Minute)
	cases := []struct {
		name    string
		file    string
		content string
		mutate  func(*store.PendingItem)
		reused  bool
	}{
		{name: "same family, field and speaker", file: "Twin.json", content: oneLine(t, twinText), reused: true},
		{name: "another family", file: "StoryData/S001A.json", content: oneLine(t, twinText)},
		{name: "another field", file: "Field.json", content: `{"dataList":[{"id":0,"desc":` + q(t, twinText) + `}]}`},
		{name: "another speaker", file: "Speaker.json", content: dataList(t, 1, "낯선이", func(int) string { return twinText })},
		{name: "another text", file: "Text.json", content: oneLine(t, twinText+" 아니다")},
		{name: "leased to another batch", file: "Leased.json", content: oneLine(t, twinText),
			mutate: func(it *store.PendingItem) { it.Lease = &store.Lease{Batch: "20261101T000000-deadbeef", Until: soon} }},
		{name: "lease that has run out", file: "Expired.json", content: oneLine(t, twinText),
			mutate: func(it *store.PendingItem) {
				it.Lease = &store.Lease{Batch: "20261101T000000-deadbeef", Until: testNow.Add(-time.Minute)}
			}, reused: true},
		{name: "waits for a person", file: "Review.json", content: oneLine(t, twinText),
			mutate: func(it *store.PendingItem) { it.Flags = []string{store.FlagNeedsReview} }},
		{name: "a person asked for another translation", file: "Hint.json", content: oneLine(t, twinText),
			mutate: func(it *store.PendingItem) { it.Reason, it.Hint = "requeue", "稱呼改用您" }},
		{name: "a human adjustment expired", file: "Adjusted.json", content: oneLine(t, twinText),
			mutate: func(it *store.PendingItem) {
				it.Reason, it.Refs = "adjustment-expired", map[string]string{store.RefAdjustment: "人工寫的舊譯文"}
			}},
	}
	kr := map[string]string{"A.json": oneLine(t, twinText)}
	pending := items("A.json", 0, 1)
	for _, tc := range cases {
		kr[tc.file] = tc.content
		it := store.PendingItem{Key: units.Key(tc.file, "/dataList/0/content"), Reason: "new"}
		if tc.name == "another field" {
			it.Key = units.Key(tc.file, "/dataList/0/desc")
		}
		if tc.mutate != nil {
			tc.mutate(&it)
		}
		pending = append(pending, it)
	}
	// A's line is out for translation in a batch of its own.
	lease := &store.Lease{Batch: "20261101T030405-aaaaaaaa", Until: testNow.Add(time.Hour)}
	pending[0].Lease = lease
	b := &Batch{ID: lease.Batch, Units: []BatchUnit{{Key: key("A.json", 0), KR: twinText}}}
	env := newEnv(t, kr, nil, nil)
	seed(t, env, pending...)

	res, err := Submit(env, b, []OutUnit{{Key: key("A.json", 0), ZH: "同一句話"}}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	got := pendingMap(t, env)
	for i, tc := range cases {
		k := pending[i+1].Key
		_, stillPending := got[k]
		if reused := slices.Contains(res.Reused, k); reused != tc.reused || stillPending == tc.reused {
			t.Errorf("%s: reused = %v, pending = %v, want reused = %v", tc.name, reused, stillPending, tc.reused)
		}
		if _, has := records(t, env)[k]; has != tc.reused {
			t.Errorf("%s: has a record = %v, want %v", tc.name, has, tc.reused)
		}
	}
}

// The lines of a story are twins when the same character says them.
func TestSubmitReusesForTwinsOfTheSameSpeaker(t *testing.T) {
	const s1, s2, s3 = "StoryData/S001A.json", "StoryData/S002A.json", "StoryData/S003A.json"
	kr := map[string]string{
		s1: dataList(t, 1, "파우스트", func(int) string { return twinText }),
		s2: dataList(t, 1, "파우스트", func(int) string { return twinText }),
		s3: dataList(t, 1, "싱클레어", func(int) string { return twinText }),
	}
	env := newEnv(t, kr, nil, nil)
	seed(t, env, append(append(items(s1, 0, 1), items(s2, 0, 1)...), items(s3, 0, 1)...)...)

	b := mustNext(t, env, NextOptions{})
	if b == nil || b.File != s1 {
		t.Fatalf("batch = %+v, want %s", b, s1)
	}
	// Faust's other line waits for the first; Sinclair's does not.
	if b2 := mustNext(t, env, NextOptions{}); b2 == nil || b2.File != s3 {
		t.Fatalf("second batch = %+v, want %s: Sinclair says the same words", b2, s3)
	}
	res, err := Submit(env, b, []OutUnit{{Key: key(s1, 0), ZH: "同一句話"}}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{key(s2, 0)}; !slices.Equal(res.Reused, want) {
		t.Errorf("reused = %v, want %v", res.Reused, want)
	}
	if _, queued := pendingMap(t, env)[key(s3, 0)]; !queued {
		t.Error("Sinclair's line was filled in with Faust's")
	}
}

// What a person wrote for a unit (a hint with a request for another
// translation, the human text of an adjustment that expired) is for that unit:
// it is not held back for a twin, and goes out with what the person wrote.
func TestNextDoesNotDeferAUnitWithAHint(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*store.PendingItem)
		check  func(*testing.T, BatchUnit)
	}{
		{"hint", func(it *store.PendingItem) { it.Reason, it.Hint = "requeue", "稱呼改用您" }, func(t *testing.T, u BatchUnit) {
			if u.Hint != "稱呼改用您" {
				t.Errorf("hint = %q, want the person's hint", u.Hint)
			}
		}},
		{"expired adjustment", func(it *store.PendingItem) {
			it.Reason, it.Refs = "adjustment-expired", map[string]string{store.RefAdjustment: "人工寫的舊譯文"}
		}, func(t *testing.T, u BatchUnit) {
			if u.Refs[store.RefAdjustment] != "人工寫的舊譯文" {
				t.Errorf("refs = %v, want the old adjustment", u.Refs)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			kr := map[string]string{"A.json": oneLine(t, twinText), "B.json": oneLine(t, twinText)}
			env := newEnv(t, kr, nil, nil)
			pending := append(items("A.json", 0, 1), items("B.json", 0, 1)...)
			pending[0].Lease = &store.Lease{Batch: "20261101T000000-deadbeef", Until: testNow.Add(time.Hour)}
			tc.mutate(&pending[1])
			seed(t, env, pending...)
			b := mustNext(t, env, NextOptions{})
			if b == nil || b.File != "B.json" {
				t.Fatalf("batch = %+v, want B.json: its twin is leased, but the unit has what a person wrote", b)
			}
			tc.check(t, b.Units[0])
		})
	}
}

// A short text is shared within its file only.
func TestSubmitReusesShortTextsWithinTheFileOnly(t *testing.T) {
	kr := map[string]string{
		"A.json": `{"dataList":[{"id":0,"content":"확인"},{"id":1,"content":"확인"}]}`,
		"B.json": oneLine(t, "확인"),
	}
	env := newEnv(t, kr, nil, nil)
	seed(t, env, append(items("A.json", 0, 2), items("B.json", 0, 1)...)...)
	b := mustNext(t, env, NextOptions{MaxUnits: 1})
	if b == nil || b.File != "A.json" {
		t.Fatalf("batch = %+v, want A.json", b)
	}
	res, err := Submit(env, b, []OutUnit{{Key: key("A.json", 0), ZH: "確認"}}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{key("A.json", 1)}; !slices.Equal(res.Reused, want) {
		t.Errorf("reused = %v, want %v", res.Reused, want)
	}
	if _, queued := pendingMap(t, env)[key("B.json", 0)]; !queued {
		t.Error("the short line of another file was filled in")
	}
}

// The units of the batch itself are answered by its out file, never by a twin:
// a unit it left out is released, not reused, even when the lease of the batch
// has run out by the time it is submitted.
func TestSubmitDoesNotReuseForUnitsOfItsOwnBatch(t *testing.T) {
	kr := map[string]string{"A.json": `{"dataList":[{"id":0,"content":` + q(t, twinText) + `},{"id":1,"content":` + q(t, twinText) + `}]}`}
	env := newEnv(t, kr, nil, nil)
	lease := &store.Lease{Batch: "20261101T030405-aaaaaaaa", Until: testNow.Add(-time.Minute)}
	p := items("A.json", 0, 2)
	p[0].Lease, p[1].Lease = lease, lease
	seed(t, env, p...)
	b := &Batch{ID: lease.Batch, Units: []BatchUnit{{Key: p[0].Key, KR: twinText}, {Key: p[1].Key, KR: twinText}}}

	res, err := Submit(env, b, []OutUnit{{Key: p[0].Key, ZH: "同一句話"}}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Reused) != 0 || !slices.Equal(res.Released, []string{p[1].Key}) {
		t.Errorf("reused = %v, released = %v, want the omitted line released and not reused", res.Reused, res.Released)
	}
	if _, has := records(t, env)[p[1].Key]; has {
		t.Error("the omitted line got a record")
	}
}

func TestSubmitSkipsATwinThatFailsTheCheck(t *testing.T) {
	const text = "같은 위력 문장이 두 군데에 있다"
	kr := map[string]string{"A.json": oneLine(t, text), "B.json": oneLine(t, text), "C.json": oneLine(t, text)}
	env := newEnv(t, kr, nil, nil)
	// B's glossary forbids the rendering that A, which the term does not
	// reach, may use.
	env.Rules = testRulesWith(t, `- kr: 위력
  zh: 威力
  status: approved
  forbidden: [力量]
  files: ["B.json"]
`)
	seed(t, env, append(append(items("A.json", 0, 1), items("B.json", 0, 1)...), items("C.json", 0, 1)...)...)
	b := mustNext(t, env, NextOptions{MaxUnits: 1})
	if b == nil || b.File != "A.json" {
		t.Fatalf("batch = %+v, want A.json", b)
	}
	res, err := Submit(env, b, []OutUnit{{Key: key("A.json", 0), ZH: "同樣的力量句子"}}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 1 || len(res.Rejected) != 0 {
		t.Fatalf("accepted %v rejected %v, want A accepted", res.Accepted, res.Rejected)
	}
	if want := []string{key("C.json", 0)}; !slices.Equal(res.Reused, want) {
		t.Errorf("reused = %v, want only %v", res.Reused, want)
	}
	pending := pendingMap(t, env)
	it, ok := pending[key("B.json", 0)]
	if !ok || it.Lease != nil || it.Attempts != 0 || len(it.Flags) != 0 {
		t.Errorf("B's pending item = %+v, %v, want it pending as it was", it, ok)
	}
	if _, has := records(t, env)[key("B.json", 0)]; has {
		t.Error("B got a record although the translation breaks its glossary")
	}
}

func TestGlossaryFileScope(t *testing.T) {
	const glossary = `- kr: 위력
  zh: 威力
  status: approved
  forbidden: [力量]
  files: ["StoryData/S10*"]
`
	const s1, s2 = "StoryData/S1001A.json", "StoryData/S2001A.json"
	kr := map[string]string{
		s1: dataList(t, 1, "파우스트", func(int) string { return "위력 5" }),
		s2: dataList(t, 1, "파우스트", func(int) string { return "위력 7" }),
	}
	env := newEnv(t, kr, nil, nil)
	env.Rules = testRulesWith(t, glossary)
	seed(t, env, append(items(s1, 0, 1), items(s2, 0, 1)...)...)

	b1 := mustNext(t, env, NextOptions{})
	b2 := mustNext(t, env, NextOptions{})
	if b1 == nil || b1.File != s1 || b2 == nil || b2.File != s2 {
		t.Fatalf("batches = %+v, %+v, want S1001A.json and S2001A.json", b1, b2)
	}
	if len(b1.Glossary) != 1 || b1.Glossary[0].KR != "위력" || !slices.Equal(b1.Glossary[0].Files, []string{"StoryData/S10*"}) {
		t.Errorf("glossary of %s = %+v, want the term for the files S10*", s1, b1.Glossary)
	}
	if len(b2.Glossary) != 0 {
		t.Errorf("glossary of %s = %+v, want none: the term is for other files", s2, b2.Glossary)
	}

	// The forbidden rendering is refused where the term applies, and passes
	// where it does not.
	res, err := Submit(env, b1, []OutUnit{{Key: key(s1, 0), ZH: "力量5"}}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rejected) != 1 || !hasCode(res.Rejected[0].Issues, "forbidden-term") {
		t.Errorf("rejected = %+v, want forbidden-term for %s", res.Rejected, s1)
	}
	res, err = Submit(env, b2, []OutUnit{{Key: key(s2, 0), ZH: "力量7"}}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 1 || len(res.Rejected) != 0 {
		t.Errorf("accepted %v rejected %v, want %s accepted: the term does not apply there", res.Accepted, res.Rejected, s2)
	}
}

// Of two entries for one KR text, the one for the file stands for the term in
// it, and the one without files in all the others.
func TestGlossaryFileScopeReplacesTheGeneralTerm(t *testing.T) {
	const glossary = `- kr: 신부
  zh: 牧師
  status: approved
  forbidden: [神父]
  files: ["StoryData/S10*"]
- kr: 신부
  zh: 神父
  status: approved
`
	const s1, s2 = "StoryData/S1001A.json", "StoryData/S2001A.json"
	kr := map[string]string{
		s1: dataList(t, 1, "파우스트", func(int) string { return "신부가 왔다" }),
		s2: dataList(t, 1, "파우스트", func(int) string { return "신부를 만났다" }),
	}
	env := newEnv(t, kr, nil, nil)
	env.Rules = testRulesWith(t, glossary)
	seed(t, env, append(items(s1, 0, 1), items(s2, 0, 1)...)...)

	zh := func(b *Batch) []string {
		var out []string
		for _, g := range b.Glossary {
			out = append(out, g.ZH)
		}
		return out
	}
	b1 := mustNext(t, env, NextOptions{})
	b2 := mustNext(t, env, NextOptions{})
	if b1 == nil || b2 == nil || !slices.Equal(zh(b1), []string{"牧師"}) || !slices.Equal(zh(b2), []string{"神父"}) {
		t.Fatalf("glossaries = %v and %v, want 牧師 for %s and 神父 for %s", b1.Glossary, b2.Glossary, s1, s2)
	}
	res, err := Submit(env, b1, []OutUnit{{Key: key(s1, 0), ZH: "神父來了"}}, "m", 3)
	if err != nil || len(res.Rejected) != 1 || !hasCode(res.Rejected[0].Issues, "forbidden-term") {
		t.Errorf("神父 in %s: %+v, %v, want forbidden-term", s1, res, err)
	}
	res, err = Submit(env, b2, []OutUnit{{Key: key(s2, 0), ZH: "遇見神父"}}, "m", 3)
	if err != nil || len(res.Accepted) != 1 {
		t.Errorf("神父 in %s: %+v, %v, want it accepted", s2, res, err)
	}
}

func TestForeignNameIgnoreList(t *testing.T) {
	kr := map[string]string{"Other.json": dataList(t, 2, "", func(i int) string { return fmt.Sprintf("엄마와 뫼르소 %d", i) })}
	env := newEnv(t, kr, nil, nil)
	env.ForeignNames = []string{"默爾索", "媽媽"}
	env.Rules.ForeignIgnore = []string{"媽媽"}
	seed(t, env, items("Other.json", 0, 2)...)
	b := mustNext(t, env, NextOptions{})

	res, err := Submit(env, b, []OutUnit{
		{Key: key("Other.json", 0), ZH: "媽媽"},
		{Key: key("Other.json", 1), ZH: "媽媽和默爾索"},
	}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 2 {
		t.Fatalf("accepted %v rejected %v, want both accepted", res.Accepted, res.Rejected)
	}
	if len(res.Warnings) != 1 || res.Warnings[0].Key != key("Other.json", 1) {
		t.Fatalf("warnings = %+v, want only the line with 默爾索", res.Warnings)
	}
	for _, is := range res.Warnings[0].Issues {
		if is.Code == "foreign-name" && strings.Contains(is.Message, "媽媽") {
			t.Errorf("warning %+v is about the word on the ignore list", is)
		}
	}
	if !hasCode(res.Warnings[0].Issues, "foreign-name") {
		t.Errorf("warnings = %+v, want a foreign-name for 默爾索", res.Warnings)
	}
}

// The tests of the space after keyword names.

const keywordKR = "대상에게 파열 2레벨 부여"

// The game shows the effect text of a status for "name + space", so a name
// the translator wrote without an [ID] gets the space on submission.
func TestSubmitAddsKeywordSpace(t *testing.T) {
	kr := map[string]string{"Other.json": oneLine(t, keywordKR)}
	env := newEnv(t, kr, nil, nil)
	env.KeywordNames = []string{"破裂", "破裂易損", "強壯"}
	seed(t, env, items("Other.json", 0, 1)...)
	b := mustNext(t, env, NextOptions{})

	res, err := Submit(env, b, []OutUnit{{Key: key("Other.json", 0), ZH: "使目標增加2級破裂強度"}}, "test-model", 3)
	if err != nil || len(res.Accepted) != 1 || len(res.Rejected) != 0 {
		t.Fatalf("Submit = %+v, %v", res, err)
	}
	if got, want := records(t, env)[key("Other.json", 0)].ZH, "使目標增加2級破裂 強度"; got != want {
		t.Errorf("record ZH = %q, want %q", got, want)
	}
}

// Without names the text is only normalized.
func TestSubmitWithoutKeywordNames(t *testing.T) {
	kr := map[string]string{"Other.json": oneLine(t, keywordKR)}
	env := newEnv(t, kr, nil, nil)
	seed(t, env, items("Other.json", 0, 1)...)
	b := mustNext(t, env, NextOptions{})

	if _, err := Submit(env, b, []OutUnit{{Key: key("Other.json", 0), ZH: "使目標增加2級破裂強度"}}, "test-model", 3); err != nil {
		t.Fatal(err)
	}
	if got, want := records(t, env)[key("Other.json", 0)].ZH, "使目標增加2級破裂強度"; got != want {
		t.Errorf("record ZH = %q, want %q", got, want)
	}
}

// A twin takes the translation as it was accepted, space included.
func TestSubmitKeywordSpaceReachesTwins(t *testing.T) {
	text := keywordKR + " 하고 있다"
	kr := map[string]string{"A.json": oneLine(t, text), "B.json": oneLine(t, text)}
	env := newEnv(t, kr, nil, nil)
	env.KeywordNames = []string{"破裂"}
	seed(t, env, append(items("A.json", 0, 1), items("B.json", 0, 1)...)...)
	b := mustNext(t, env, NextOptions{MaxUnits: 1})

	res, err := Submit(env, b, []OutUnit{{Key: key("A.json", 0), ZH: "使目標增加2級破裂強度"}}, "test-model", 3)
	if err != nil || len(res.Accepted) != 1 || len(res.Reused) != 1 {
		t.Fatalf("Submit = %+v, %v, want one accepted and one reused", res, err)
	}
	recs := records(t, env)
	for _, f := range []string{"A.json", "B.json"} {
		if got, want := recs[key(f, 0)].ZH, "使目標增加2級破裂 強度"; got != want {
			t.Errorf("record ZH of %s = %q, want %q", f, got, want)
		}
	}
}

// CheckNew knows the file, which the warning about half-width brackets in the
// labels of the RPG needs.
func TestCheckNewKnowsTheFile(t *testing.T) {
	r := testRules(t)
	if issues := CheckNew(r, "RPGSystem/rpg-loc-quest-floor-3.json", "퀘스트", "(內部)任務"); !hasCode(issues, "halfwidth-bracket") {
		t.Errorf("CheckNew of a quest label = %+v, want halfwidth-bracket", issues)
	}
	if issues := CheckNew(r, "RPGSystem/rpg-loc-dialogue-floor-3.json", "퀘스트", "(內部)任務"); hasCode(issues, "halfwidth-bracket") {
		t.Errorf("CheckNew of a dialogue = %+v, want no halfwidth-bracket", issues)
	}
}

func TestKeywordNames(t *testing.T) {
	kr := map[string]string{
		"Bufs.json": `{"dataList":[
 {"id":"Burst","name":"파열"},
 {"id":"Burn","name":"화상"},
 {"id":"Bleed","name":"출혈"},
 {"id":"Fragile","name":"파열 취약"},
 {"id":"Same","name":"파열"},
 {"id":"Review","name":"검토"}
]}`,
		"BattleKeywords.json": `{"dataList":[{"id":"Poise","name":"호흡"}]}`,
		"Other.json":          oneLine(t, "다른 파일"),
	}
	env := newEnv(t, kr, nil, nil)
	rec := func(file string, i int, krText, zh string, flags ...string) store.Record {
		k := fmt.Sprintf("%s|/dataList/%d/name", file, i)
		return store.Record{Key: k, KRSha: units.KRSha(krText), ZH: zh, Origin: store.OriginParatranz, Since: testTag, Flags: flags}
	}
	recs := []store.Record{
		rec("Bufs.json", 0, "파열", "破裂"),
		rec("Bufs.json", 1, "화상", "燒傷"),
		// Made when Bleed was called something else.
		rec("Bufs.json", 2, "옛 이름", "舊名"),
		// Still Korean: no name of ours.
		rec("Bufs.json", 3, "파열 취약", "파열 易損"),
		// The same name for another id.
		rec("Bufs.json", 4, "파열", "破裂"),
		rec("Bufs.json", 5, "검토", "待審", store.FlagNeedsReview),
		rec("BattleKeywords.json", 0, "호흡", "呼吸"),
		// Not a keyword file.
		{Key: key("Other.json", 0), KRSha: units.KRSha("다른 파일"), ZH: "別的檔案", Origin: store.OriginParatranz, Since: testTag},
	}
	byKey := map[string]store.Record{}
	for _, r := range recs {
		byKey[r.Key] = r
	}
	if err := env.Store.SaveRecords(byKey); err != nil {
		t.Fatal(err)
	}

	got, err := KeywordNames(env.Src.KR, records(t, env))
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"呼吸", "燒傷", "破裂"}; !slices.Equal(got, want) {
		t.Errorf("KeywordNames = %v, want %v: translated by a record of today's KR text, once each, sorted", got, want)
	}

	none, err := KeywordNames(env.Src.KR, nil)
	if err != nil || none == nil || len(none) != 0 {
		t.Errorf("KeywordNames without records = %#v, %v, want an empty list", none, err)
	}
}

// Overlay writes the translations over the KR text and leaves the pack's own
// document as it was.
func TestOverlay(t *testing.T) {
	kr := map[string]string{
		"Bufs.json": `{"dataList":[
 {"id":"Burst","name":"파열"},
 {"id":"Bleed","name":"출혈"},
 {"id":"Review","name":"검토"},
 {"id":"Plain","name":"그대로"}
]}`,
	}
	env := newEnv(t, kr, nil, nil)
	rec := func(i int, krText, zh string, flags ...string) store.Record {
		k := fmt.Sprintf("Bufs.json|/dataList/%d/name", i)
		return store.Record{Key: k, KRSha: units.KRSha(krText), ZH: zh, Origin: store.OriginParatranz, Since: testTag, Flags: flags}
	}
	byKey := map[string]store.Record{}
	for _, r := range []store.Record{
		rec(0, "파열", "破裂"),
		rec(1, "옛 이름", "舊名"), // made from other KR text than today's
		rec(2, "검토", "待審", store.FlagNeedsReview),
	} {
		byKey[r.Key] = r
	}

	doc, err := Overlay(env.Src.KR, byKey, "Bufs.json")
	if err != nil {
		t.Fatal(err)
	}
	name := func(d *jsonx.Node, i int) string {
		n, ok := d.Get(fmt.Sprintf("/dataList/%d/name", i))
		if !ok {
			t.Fatalf("no /dataList/%d/name", i)
		}
		return n.Str
	}
	for i, want := range []string{"破裂", "출혈", "검토", "그대로"} {
		if got := name(doc, i); got != want {
			t.Errorf("name %d = %q, want %q: only a record of today's KR text that is not flagged needs_review is written", i, got, want)
		}
	}

	orig, err := env.Src.KR.Doc("Bufs.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := name(orig, 0); got != "파열" {
		t.Errorf("the pack's document was changed: name 0 = %q, want 파열", got)
	}

	if none, err := Overlay(env.Src.KR, nil, "Bufs.json"); err != nil || name(none, 0) != "파열" {
		t.Errorf("Overlay without records = %v, %v, want the KR document", none, err)
	}
	if _, err := Overlay(env.Src.KR, byKey, "Nothing.json"); err == nil {
		t.Error("Overlay of a file that is not in the pack did not fail")
	}
}

// A line that needs review does not end a story batch or stay out of it by
// accident: the batch takes the pending lines around it, whichever place the
// line has, and the line itself stays in the queue, flagged and unleased. It
// is no context either, for it has no translation to show.
func TestStoryBatchWithANeedsReviewLine(t *testing.T) {
	const file = "StoryData/S001A.json"
	for _, review := range []int{2, 5, 8} { // first, middle and last pending line
		t.Run(fmt.Sprintf("line %d", review), func(t *testing.T) {
			env := newEnv(t, map[string]string{file: dataList(t, 10, "파우스트", line)}, nil, nil)
			// Lines 0 and 9 are translated; 2, 5 and 8 wait for translation, one
			// of them for a person.
			var pending []store.PendingItem
			var want []string
			for _, i := range []int{2, 5, 8} {
				it := store.PendingItem{Key: key(file, i), Reason: "new"}
				if i == review {
					it.Attempts = 3
					it.Flags = []string{store.FlagNeedsReview}
				} else {
					want = append(want, key(file, i))
				}
				pending = append(pending, it)
			}
			seed(t, env, pending...)
			recs := map[string]store.Record{
				key(file, review): {Key: key(file, review), KRSha: units.KRSha(line(review)), ZH: "line", Origin: store.OriginFallback, Since: testTag, Flags: []string{store.FlagNeedsReview}},
			}
			for _, i := range []int{0, 1, 3, 4, 6, 7, 9} {
				recs[key(file, i)] = translated(key(file, i), line(i), fmt.Sprintf("舊 %d", i))
			}
			if err := env.Store.SaveRecords(recs); err != nil {
				t.Fatal(err)
			}

			b := mustNext(t, env, NextOptions{})
			if b == nil {
				t.Fatalf("no batch: the line that needs review stopped the story batch")
			}
			if !slices.Equal(keysOf(b), want) {
				t.Errorf("units = %v, want %v", keysOf(b), want)
			}
			if slices.Contains(ctxKeys(b), key(file, review)) {
				t.Errorf("context %v shows the line that needs review", ctxKeys(b))
			}
			held := pendingMap(t, env)[key(file, review)]
			if held.Lease != nil || held.Attempts != 3 || !slices.Equal(held.Flags, []string{store.FlagNeedsReview}) {
				t.Errorf("line %d = %+v, want it untouched: unleased, three attempts, needs_review", review, held)
			}
			for _, k := range want {
				if pendingMap(t, env)[k].Lease == nil {
					t.Errorf("%s is in the batch but not leased", k)
				}
			}
			if next := mustNext(t, env, NextOptions{}); next != nil {
				t.Errorf("a second batch = %+v, want nil: the line that needs review is not handed out", next)
			}
		})
	}
}

// A story batch of one unit carries one element of context on each side: more
// than the unit it is made for would cost more than it helps. A batch of n
// units carries n on each side of each unit, up to contextElements.
func TestStoryContextOfASmallBatch(t *testing.T) {
	const file = "StoryData/S001A.json"
	tests := []struct {
		name    string
		pending []int
		want    []int // the lines of the context
	}{
		{"one unit", []int{10}, []int{9, 11}},
		{"one unit at the start", []int{0}, []int{1}},
		{"one unit at the end", []int{29}, []int{28}},
		{"two units far apart", []int{10, 20}, []int{8, 9, 11, 12, 18, 19, 21, 22}},
		{"two units next to each other", []int{10, 11}, []int{8, 9, 12, 13}},
		{"three units", []int{5, 15, 25}, []int{2, 3, 4, 6, 7, 8, 12, 13, 14, 16, 17, 18, 22, 23, 24, 26, 27, 28}},
		{"four units are capped at three", []int{5, 15, 20, 28}, []int{2, 3, 4, 6, 7, 8, 12, 13, 14, 16, 17, 18, 19, 21, 22, 23, 25, 26, 27, 29}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newEnv(t, map[string]string{file: dataList(t, 30, "", line)}, nil, nil)
			var pending []store.PendingItem
			recs := map[string]store.Record{}
			for i := 0; i < 30; i++ {
				if slices.Contains(tc.pending, i) {
					pending = append(pending, store.PendingItem{Key: key(file, i), Reason: "new"})
				} else {
					recs[key(file, i)] = translated(key(file, i), line(i), fmt.Sprintf("舊 %d", i))
				}
			}
			seed(t, env, pending...)
			if err := env.Store.SaveRecords(recs); err != nil {
				t.Fatal(err)
			}
			b := mustNext(t, env, NextOptions{})
			if len(b.Units) != len(tc.pending) {
				t.Fatalf("batch has %d units, want %d", len(b.Units), len(tc.pending))
			}
			var want []string
			for _, i := range tc.want {
				want = append(want, key(file, i))
			}
			if got := ctxKeys(b); !slices.Equal(got, want) {
				t.Errorf("context = %v, want %v", got, want)
			}
		})
	}
}

// The glossary rendering is held to at submission: a text without it is
// rejected unless the unit's note names the term (it is the KR of the term
// the note has to carry).
const seasonGlossary = `- kr: 시즌
  zh: 賽季
  status: approved
  check: hard
`

func seasonEnv(t *testing.T) (Env, *Batch) {
	t.Helper()
	kr := map[string]string{"Other.json": dataList(t, 1, "", func(int) string { return "두 시즌 전부터" })}
	env := newEnv(t, kr, nil, nil)
	env.Rules = testRulesWith(t, seasonGlossary)
	seed(t, env, items("Other.json", 0, 1)...)
	return env, mustNext(t, env, NextOptions{})
}

func TestSubmitRejectsAMissingTermWithoutANote(t *testing.T) {
	env, b := seasonEnv(t)
	res, err := Submit(env, b, []OutUnit{{Key: key("Other.json", 0), ZH: "從兩棵季前起"}}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 0 || len(res.Rejected) != 1 {
		t.Fatalf("result = %+v, want one rejection", res)
	}
	var found bool
	for _, is := range res.Rejected[0].Issues {
		if is.Code == validate.CodeTermMissing {
			found = true
			if is.Severity != validate.SeverityHard || is.Term != "시즌" {
				t.Errorf("issue = %+v, want hard for 시즌", is)
			}
		}
	}
	if !found {
		t.Errorf("issues = %+v, want term-missing", res.Rejected[0].Issues)
	}
	if got := pendingMap(t, env)[key("Other.json", 0)].Attempts; got != 1 {
		t.Errorf("Attempts = %d, want 1", got)
	}
}

func TestSubmitAcceptsAMissingTermTheNoteNames(t *testing.T) {
	env, b := seasonEnv(t)
	res, err := Submit(env, b, []OutUnit{{Key: key("Other.json", 0), ZH: "從兩棵季前起", Note: "시즌：劇中季節，非遊戲賽季"}}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 1 || len(res.Rejected) != 0 {
		t.Fatalf("result = %+v, want accepted", res)
	}
	if len(res.Warnings) != 1 || len(res.Warnings[0].Issues) != 1 ||
		res.Warnings[0].Issues[0].Code != validate.CodeTermMissing || res.Warnings[0].Issues[0].Severity != validate.SeverityWarn {
		t.Errorf("warnings = %+v, want one term-missing warn", res.Warnings)
	}
}

func TestSubmitTermMissingNeedsTheTermInTheNote(t *testing.T) {
	env, b := seasonEnv(t)
	res, err := Submit(env, b, []OutUnit{{Key: key("Other.json", 0), ZH: "從兩棵季前起", Note: "탐식：卡包名"}}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 0 || len(res.Rejected) != 1 || !hasCode(res.Rejected[0].Issues, validate.CodeTermMissing) {
		t.Errorf("result = %+v, want rejected for term-missing", res)
	}
}

// In a family with glossary: warn a missing glossary rendering stays a
// warning, with or without a note; a forbidden rendering is still rejected,
// and a family without the setting still rejects the missing rendering.
func TestSubmitWarnFamily(t *testing.T) {
	const routing = `families:
  - id: voice
    nodes: ["Warn*"]
    glossary: warn
    guide: guides/families.md
  - id: other
    nodes: []
    guide: guides/families.md
priority:
  - voice
  - other
`
	const glossary = seasonGlossary + "  forbidden: [季前]\n"
	run := func(t *testing.T, file, zh, note string) (*SubmitResult, *Batch) {
		t.Helper()
		kr := map[string]string{file: dataList(t, 1, "", func(int) string { return "두 시즌 전부터" })}
		env := newEnv(t, kr, nil, nil)
		env.Rules = testRulesWithRouting(t, routing, glossary)
		seed(t, env, items(file, 0, 1)...)
		b := mustNext(t, env, NextOptions{})
		res, err := Submit(env, b, []OutUnit{{Key: key(file, 0), ZH: zh, Note: note}}, "m", 3)
		if err != nil {
			t.Fatal(err)
		}
		return res, b
	}
	t.Run("missing without a note is accepted with a warning", func(t *testing.T) {
		res, b := run(t, "Warn.json", "從兩個賽局前起", "")
		if b.GlossaryCheck != "warn" {
			t.Errorf("GlossaryCheck = %q, want warn", b.GlossaryCheck)
		}
		if len(res.Accepted) != 1 || len(res.Rejected) != 0 {
			t.Fatalf("result = %+v, want accepted", res)
		}
		if len(res.Warnings) != 1 || !hasCode(res.Warnings[0].Issues, validate.CodeTermMissing) || validate.HasHard(res.Warnings[0].Issues) {
			t.Errorf("warnings = %+v, want term-missing as a warning", res.Warnings)
		}
	})
	t.Run("missing with an unrelated note is accepted", func(t *testing.T) {
		res, _ := run(t, "Warn.json", "從兩個賽局前起", "탐식：卡包名")
		if len(res.Accepted) != 1 || len(res.Rejected) != 0 {
			t.Errorf("result = %+v, want accepted", res)
		}
	})
	t.Run("forbidden is still rejected", func(t *testing.T) {
		res, _ := run(t, "Warn.json", "從兩棵季前起", "")
		if len(res.Accepted) != 0 || len(res.Rejected) != 1 || !hasCode(res.Rejected[0].Issues, "forbidden-term") {
			t.Errorf("result = %+v, want rejected for forbidden-term", res)
		}
	})
	t.Run("hard family still rejects a missing rendering", func(t *testing.T) {
		res, b := run(t, "Other.json", "從兩個賽局前起", "")
		if b.GlossaryCheck != "" {
			t.Errorf("GlossaryCheck = %q, want none", b.GlossaryCheck)
		}
		if len(res.Accepted) != 0 || len(res.Rejected) != 1 || !hasCode(res.Rejected[0].Issues, validate.CodeTermMissing) {
			t.Errorf("result = %+v, want rejected for term-missing", res)
		}
	})
}

const warnRouting = `families:
  - id: voice
    nodes: ["Warn*"]
    glossary: warn
    guide: guides/families.md
  - id: other
    nodes: []
    guide: guides/families.md
priority:
  - voice
  - other
`

// A merged batch (several files of one non-story family) in a warn family
// carries glossary_check: warn, and a missing rendering is accepted in every
// one of its files.
func TestSubmitWarnFamilyMergedBatch(t *testing.T) {
	kr := map[string]string{
		"Warn1.json": oneLine(t, "두 시즌 전부터"),
		"Warn2.json": oneLine(t, "세 시즌 전부터"),
	}
	env := newEnv(t, kr, nil, nil)
	env.Rules = testRulesWithRouting(t, warnRouting, seasonGlossary)
	seed(t, env, append(items("Warn1.json", 0, 1), items("Warn2.json", 0, 1)...)...)
	b := mustNext(t, env, NextOptions{})
	if b == nil || !slices.Equal(b.Files, []string{"Warn1.json", "Warn2.json"}) || b.GlossaryCheck != "warn" {
		t.Fatalf("batch = %+v, want both files with glossary_check warn", b)
	}
	res, err := Submit(env, b, []OutUnit{
		{Key: key("Warn1.json", 0), ZH: "從兩個賽局前起"},
		{Key: key("Warn2.json", 0), ZH: "從三個賽局前起"},
	}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 2 || len(res.Rejected) != 0 || len(res.Warnings) != 2 {
		t.Errorf("result = %+v, want both accepted, each with a term-missing warning", res)
	}
}

// Twins of a warn-family unit stay in the warn family: a twin in another warn
// file takes the translation although it lacks the glossary rendering (the
// term-missing path of a warn file), a twin whose file forbids a word of the
// translation is not given it, and a unit of the same KR text in a family
// that is not warn (Other.json, which also lacks the rendering) is not a twin
// at all, whatever the glossary says. The guarantee is the family in the
// TextGroup that keys a twin (the families[...] filter in reuseForTwins only
// narrows the scan).
func TestTwinsOfAWarnFamilyUnit(t *testing.T) {
	const text = "같은 문장에 시즌이 두 군데에 있다"
	kr := map[string]string{}
	for _, f := range []string{"Warn1.json", "Warn2.json", "Warn3.json", "Other.json"} {
		kr[f] = oneLine(t, text)
	}
	// 賽季 is asked of every file; Warn2.json has its own entry for 시즌 that
	// also forbids 季前, and takes the place of the general one there.
	const glossary = seasonGlossary +
		"- kr: 시즌\n  zh: 賽季\n  status: approved\n  forbidden: [季前]\n  files: [\"Warn2.json\"]\n"
	env := newEnv(t, kr, nil, nil)
	env.Rules = testRulesWithRouting(t, warnRouting, glossary)
	var all []store.PendingItem
	for _, f := range []string{"Warn1.json", "Warn2.json", "Warn3.json", "Other.json"} {
		all = append(all, items(f, 0, 1)...)
	}
	seed(t, env, all...)
	b := mustNext(t, env, NextOptions{})
	if b == nil || !slices.Equal(keysOf(b), []string{key("Warn1.json", 0)}) || b.GlossaryCheck != "warn" {
		t.Fatalf("batch = %+v, want only the first twin, glossary_check warn", b)
	}
	res, err := Submit(env, b, []OutUnit{{Key: key("Warn1.json", 0), ZH: "同一句話在季前"}}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(res.Reused, []string{key("Warn3.json", 0)}) {
		t.Errorf("reused = %v, want only Warn3.json (Warn2.json forbids 季前, Other.json is not of the family)", res.Reused)
	}
	pending := pendingMap(t, env)
	for _, f := range []string{"Warn2.json", "Other.json"} {
		if _, ok := pending[key(f, 0)]; !ok {
			t.Errorf("%s is no longer pending", f)
		}
	}
}

func TestCheckNewLeavesTermMissingAWarning(t *testing.T) {
	r := testRulesWith(t, seasonGlossary)
	issues := CheckNew(r, "Other.json", "두 시즌 전부터", "從兩棵季前起")
	if !hasCode(issues, validate.CodeTermMissing) || validate.HasHard(issues) {
		t.Errorf("CheckNew = %+v, want term-missing as a warning only", issues)
	}
}

func termsEnv(t *testing.T) (Env, *Batch) {
	t.Helper()
	kr := map[string]string{"Other.json": `{"dataList":[
 {"id":0,"content":"우레가 쳤다"},
 {"id":1,"content":"우레가 울렸다"},
 {"id":2,"content":"그냥"},
 {"id":3,"content":"[Burn] 부여"}
]}`}
	env := newEnv(t, kr, nil, nil)
	seed(t, env, items("Other.json", 0, 4)...)
	return env, mustNext(t, env, NextOptions{})
}

func TestSubmitRecordsAProvisionalTerm(t *testing.T) {
	env, b := termsEnv(t)
	out := []OutUnit{{Key: key("Other.json", 0), ZH: "雷鳴響起", Terms: []NewTerm{{KR: "우레", ZH: "雷鳴"}}}}
	res, err := Submit(env, b, out, "test-model", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 1 || res.Terms != 1 || res.TermsSkipped == nil || len(res.TermsSkipped) != 0 {
		t.Fatalf("result = %+v, want 1 accepted, 1 term, no skipped", res)
	}
	terms, err := env.Store.LoadTerms()
	if err != nil || len(terms) != 1 {
		t.Fatalf("terms = %+v, %v", terms, err)
	}
	got := terms[0]
	if got.KR != "우레" || got.ZH != "雷鳴" || got.Key != key("Other.json", 0) || got.Batch != b.ID || got.RawTag != env.RawTag || got.Conflict {
		t.Errorf("term = %+v", got)
	}
	if got.Time != env.Now.UTC().Format(time.RFC3339) {
		t.Errorf("time = %q, want %q", got.Time, env.Now.UTC().Format(time.RFC3339))
	}
}

func TestSubmitSkipsATermNotInTheUnit(t *testing.T) {
	env, b := termsEnv(t)
	out := []OutUnit{{Key: key("Other.json", 0), ZH: "雷鳴響起", Terms: []NewTerm{
		{KR: "번개", ZH: "雷鳴"}, // kr not in the KR text
		{KR: "우레", ZH: "閃電"}, // zh not in the translation
		{KR: "우", ZH: "雷"},   // kr of one character
	}}}
	res, err := Submit(env, b, out, "test-model", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 1 || res.Terms != 0 || len(res.TermsSkipped) != 3 {
		t.Fatalf("result = %+v, want the unit accepted and 3 terms skipped", res)
	}
	for i, s := range res.TermsSkipped {
		if s.Key != key("Other.json", 0) || s.Reason == "" || s.KR != out[0].Terms[i].KR || s.ZH != out[0].Terms[i].ZH {
			t.Errorf("skipped[%d] = %+v", i, s)
		}
	}
	if terms, err := env.Store.LoadTerms(); err != nil || len(terms) != 0 {
		t.Errorf("terms = %+v, %v, want none", terms, err)
	}
}

func TestSubmitRecordsTheFirstRenderingAndAConflict(t *testing.T) {
	env, b := termsEnv(t)
	out := []OutUnit{
		{Key: key("Other.json", 0), ZH: "雷鳴響起", Terms: []NewTerm{{KR: "우레", ZH: "雷鳴"}}},
		{Key: key("Other.json", 1), ZH: "驚雷響起", Terms: []NewTerm{{KR: "우레", ZH: "驚雷"}}},
	}
	res, err := Submit(env, b, out, "test-model", 3)
	if err != nil || res.Terms != 2 {
		t.Fatalf("result = %+v, %v, want 2 terms", res, err)
	}
	terms, _ := env.Store.LoadTerms()
	if len(terms) != 2 || terms[0].ZH != "雷鳴" || terms[0].Conflict || terms[1].ZH != "驚雷" || !terms[1].Conflict {
		t.Fatalf("terms = %+v", terms)
	}
	// The same file again records nothing new.
	res, err = Submit(env, b, out, "test-model", 3)
	if err != nil || res.Terms != 0 {
		t.Fatalf("resubmit: %+v, %v, want 0 terms", res, err)
	}
	if terms, _ := env.Store.LoadTerms(); len(terms) != 2 {
		t.Errorf("terms after resubmitting = %+v", terms)
	}
}

func TestSubmitRecordsNoTermOfARejectedUnit(t *testing.T) {
	env, b := termsEnv(t)
	out := []OutUnit{{Key: key("Other.json", 3), ZH: "賦予燒傷", Terms: []NewTerm{{KR: "부여", ZH: "賦予"}}}} // [Burn] missing
	res, err := Submit(env, b, out, "test-model", 3)
	if err != nil || len(res.Rejected) != 1 || res.Terms != 0 || len(res.TermsSkipped) != 0 {
		t.Fatalf("result = %+v, %v", res, err)
	}
	if terms, _ := env.Store.LoadTerms(); len(terms) != 0 {
		t.Errorf("terms = %+v, want none", terms)
	}
}

func provTerm(kr, zh string, conflict bool) store.ProvisionalTerm {
	return store.ProvisionalTerm{KR: kr, ZH: zh, Key: key("Other.json", 0), Conflict: conflict}
}

func TestActiveProvisional(t *testing.T) {
	r := testRulesWith(t, "- kr: 위력\n  zh: 威力\n  status: approved\n- kr: 후보어\n  zh: 候選\n  status: candidate\n")
	all := []store.ProvisionalTerm{
		provTerm("우레", "雷鳴", false),
		provTerm("우레", "驚雷", true),
		provTerm("하", "X", false),
		provTerm("위력", "力量", false),
		provTerm("후보어", "Y", false),
		provTerm("번개", "閃電", false),
	}
	got := ActiveProvisional(all, r, map[string]bool{"term-new/번개": true})
	want := []rules.Term{{KR: "우레", ZH: "雷鳴", Match: rules.MatchWord, Status: rules.StatusApproved, Provisional: true}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ActiveProvisional = %+v, want %+v", got, want)
	}
}

func TestProvisionalTermsYieldToTheGlossary(t *testing.T) {
	r := testRulesWith(t, "- kr: 와일드헌트 히스클리프\n  zh: 狂獵希斯克利夫\n  status: approved\n")
	all := []store.ProvisionalTerm{provTerm("와일드헌트", "荒獵", false)}
	prov := ActiveProvisional(all, r, nil)
	if len(prov) != 1 {
		t.Fatalf("provisional = %+v, want 1", prov)
	}
	// The glossary term covers the provisional one, which does not hit there:
	// the translation need not hold 荒獵 (see
	// TestCheckNewTermsWeighsProvisionalTermsWithTheGlossary).
	issues := CheckNewTerms(r, prov, "Other.json", "와일드헌트 히스클리프", "狂獵希斯克利夫")
	if validate.HasHard(issues) || hasCode(issues, validate.CodeTermMissing) {
		t.Errorf("issues = %+v, want none", issues)
	}
	// The same KR in the glossary: the provisional term does not apply.
	r2 := testRulesWith(t, "- kr: 와일드헌트\n  zh: 狂獵\n  status: candidate\n")
	if got := ActiveProvisional(all, r2, nil); len(got) != 0 {
		t.Errorf("provisional = %+v, want none when the glossary has the KR", got)
	}
}

func TestCheckNewTermsAppliesOnlyHitProvisionalTerms(t *testing.T) {
	r := testRules(t)
	prov := []rules.Term{{KR: "우레", ZH: "雷鳴", Match: rules.MatchWord, Status: rules.StatusApproved, Provisional: true}}
	if issues := CheckNewTerms(r, prov, "Other.json", "우레가 쳤다", "打雷了"); !hasCode(issues, validate.CodeTermMissing) {
		t.Errorf("issues = %+v, want term-missing", issues)
	}
	if issues := CheckNewTerms(r, prov, "Other.json", "우레비", "雨"); hasCode(issues, validate.CodeTermMissing) {
		t.Errorf("issues = %+v, want none for 우레비", issues)
	}
	if issues := CheckNew(r, "Other.json", "우레가 쳤다", "打雷了"); hasCode(issues, validate.CodeTermMissing) {
		t.Errorf("CheckNew = %+v, want no provisional term", issues)
	}
}

func provEnv(t *testing.T) Env {
	t.Helper()
	kr := map[string]string{"Other.json": `{"dataList":[
 {"id":0,"content":"우레가 쳤다"},
 {"id":1,"content":"우레비가 내렸다"}
]}`}
	env := newEnv(t, kr, nil, nil)
	env.Provisional = []rules.Term{{KR: "우레", ZH: "雷鳴", Match: rules.MatchWord, Status: rules.StatusApproved, Provisional: true}}
	return env
}

func TestNextCarriesProvisionalTerms(t *testing.T) {
	env := provEnv(t)
	seed(t, env, items("Other.json", 0, 1)...)
	b := mustNext(t, env, NextOptions{})
	if len(b.Glossary) != 1 || b.Glossary[0].KR != "우레" || !b.Glossary[0].Provisional || b.Glossary[0].ZH != "雷鳴" {
		t.Errorf("glossary = %+v, want the provisional 우레", b.Glossary)
	}
	env2 := provEnv(t)
	seed(t, env2, items("Other.json", 1, 2)...)
	if b := mustNext(t, env2, NextOptions{}); len(b.Glossary) != 0 {
		t.Errorf("glossary = %+v, want none for 우레비", b.Glossary)
	}
}

func TestSubmitChecksProvisionalTerms(t *testing.T) {
	env := provEnv(t)
	seed(t, env, items("Other.json", 0, 1)...)
	b := mustNext(t, env, NextOptions{})
	res, err := Submit(env, b, []OutUnit{{Key: key("Other.json", 0), ZH: "打雷了"}}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 1 || len(res.Rejected) != 0 || len(res.Warnings) != 1 || !hasCode(res.Warnings[0].Issues, validate.CodeTermMissing) {
		t.Errorf("result = %+v, want accepted with a term-missing warning: a provisional term only warns", res)
	}
}

// The tests of the order files are handed out in and of batches of several
// files.

// allyRouting is testRouting with the family ally (Passives*), which comes
// before other.
const allyRouting = `families:
  - id: story-main
    nodes: ["StoryData/S"]
    guide: guides/families.md
  - id: ally
    nodes: ["Passives*"]
    guide: guides/families.md
  - id: other
    nodes: []
    guide: guides/families.md
priority:
  - story-main
  - ally
  - other
`

// fileOf is the file of a unit key.
func fileOf(t *testing.T, k string) string {
	t.Helper()
	file, _, err := units.SplitKey(k)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

// checkUnitFiles fails the test unless every unit of b names the file of its
// key, and b.Files lists those files once each, in the order of the units,
// with b.File the first.
func checkUnitFiles(t *testing.T, b *Batch) {
	t.Helper()
	var files []string
	for _, u := range b.Units {
		if want := fileOf(t, u.Key); u.File != want {
			t.Errorf("unit %s: file = %q, want %q", u.Key, u.File, want)
		}
		if !slices.Contains(files, u.File) {
			files = append(files, u.File)
		}
	}
	if !slices.Equal(b.Files, files) || len(b.Files) == 0 || b.File != b.Files[0] {
		t.Errorf("batch files = %q, file = %q, want the files of the units %q, the first as the file", b.Files, b.File, files)
	}
}

func TestNextHandsOutKeywordFilesFirst(t *testing.T) {
	const passives, bufs, keywords = "Passives.json", "Bufs_X.json", "BattleKeywords_X.json"
	kr := map[string]string{
		passives: dataList(t, 2, "", line),
		bufs:     dataList(t, 2, "", line),
		keywords: dataList(t, 2, "", line),
	}
	env := newEnv(t, kr, nil, nil)
	env.Rules = testRulesWithRouting(t, allyRouting, testGlossary)
	seed(t, env, append(append(items(passives, 0, 2), items(bufs, 0, 2)...), items(keywords, 0, 2)...)...)

	// ally comes before other, but the files that name the keywords come
	// before every family; the two are of one family and share the batch.
	b1 := mustNext(t, env, NextOptions{})
	if b1 == nil || !slices.Equal(b1.Files, []string{keywords, bufs}) || b1.Family != "other" {
		t.Fatalf("first batch = %+v, want %s and %s", b1, keywords, bufs)
	}
	checkUnitFiles(t, b1)
	if b2 := mustNext(t, env, NextOptions{}); b2 == nil || !slices.Equal(b2.Files, []string{passives}) {
		t.Errorf("second batch = %+v, want %s", b2, passives)
	}

	// They come before the story too, which is first of the families.
	const story = "StoryData/S001A.json"
	kr[story] = dataList(t, 1, "파우스트", line)
	env = newEnv(t, kr, nil, nil)
	env.Rules = testRulesWithRouting(t, allyRouting, testGlossary)
	seed(t, env, append(items(story, 0, 1), items(bufs, 0, 2)...)...)
	if b := mustNext(t, env, NextOptions{}); b == nil || !slices.Equal(b.Files, []string{bufs}) {
		t.Errorf("first batch = %+v, want %s before the story", b, bufs)
	}
}

func TestNextMergesSmallFilesOfAFamily(t *testing.T) {
	files := []string{"A.json", "B.json", "C.json"}
	kr := map[string]string{"Passives.json": dataList(t, 2, "", line)}
	pending := items("Passives.json", 0, 2)
	var want []string
	for _, f := range files {
		kr[f] = dataList(t, 2, "", line)
		pending = append(pending, items(f, 0, 2)...)
		want = append(want, key(f, 0), key(f, 1))
	}
	env := newEnv(t, kr, nil, nil)
	env.Rules = testRulesWithRouting(t, allyRouting, testGlossary)
	seed(t, env, pending...)

	// Passives.json is first and of ally: the files of other do not join it.
	b1 := mustNext(t, env, NextOptions{})
	if b1 == nil || !slices.Equal(b1.Files, []string{"Passives.json"}) || len(b1.Units) != 2 {
		t.Fatalf("first batch = %+v, want the 2 units of Passives.json alone", b1)
	}
	checkUnitFiles(t, b1)

	b2 := mustNext(t, env, NextOptions{})
	if b2 == nil || !slices.Equal(b2.Files, files) || b2.File != "A.json" || b2.Family != "other" {
		t.Fatalf("second batch = %+v, want the three files of other", b2)
	}
	if !slices.Equal(keysOf(b2), want) {
		t.Errorf("second batch keys = %v, want %v", keysOf(b2), want)
	}
	checkUnitFiles(t, b2)
	got := pendingMap(t, env)
	for _, k := range want {
		if l := got[k].Lease; l == nil || l.Batch != b2.ID {
			t.Errorf("%s: lease = %+v, want batch %s", k, l, b2.ID)
		}
	}
	if b3 := mustNext(t, env, NextOptions{}); b3 != nil {
		t.Errorf("third batch = %+v, want nil", b3)
	}
}

func TestNextStopsAtMaxUnits(t *testing.T) {
	kr := map[string]string{"Passives.json": dataList(t, 1, "", line)}
	pending := items("Passives.json", 0, 1)
	for _, f := range []string{"A.json", "B.json", "C.json"} {
		kr[f] = dataList(t, 3, "", line)
		pending = append(pending, items(f, 0, 3)...)
	}
	env := newEnv(t, kr, nil, nil)
	env.Rules = testRulesWithRouting(t, allyRouting, testGlossary)
	seed(t, env, pending...)

	// --family keeps Passives.json out, which would come first.
	opt := NextOptions{Family: "other", MaxUnits: 4}
	for i, want := range []struct {
		files []string
		keys  []string
	}{
		{[]string{"A.json", "B.json"}, []string{key("A.json", 0), key("A.json", 1), key("A.json", 2), key("B.json", 0)}},
		{[]string{"B.json", "C.json"}, []string{key("B.json", 1), key("B.json", 2), key("C.json", 0), key("C.json", 1)}},
		{[]string{"C.json"}, []string{key("C.json", 2)}},
	} {
		b := mustNext(t, env, opt)
		if b == nil || !slices.Equal(b.Files, want.files) || !slices.Equal(keysOf(b), want.keys) {
			t.Fatalf("batch %d = %+v, want the keys %v of %v", i+1, b, want.keys, want.files)
		}
		checkUnitFiles(t, b)
	}
	if b := mustNext(t, env, opt); b != nil {
		t.Errorf("batch 4 = %+v, want nil: only Passives.json is left, of another family", b)
	}
}

func TestNextKeepsStoryBatchesToOneFile(t *testing.T) {
	const s1, s2 = "StoryData/S001A.json", "StoryData/S002A.json"
	kr := map[string]string{s1: dataList(t, 3, "파우스트", line), s2: dataList(t, 3, "파우스트", line)}
	env := newEnv(t, kr, nil, nil)
	seed(t, env, append(items(s1, 1, 3), items(s2, 1, 3)...)...)
	if err := env.Store.SaveRecords(map[string]store.Record{
		key(s1, 0): translated(key(s1, 0), line(0), "譯文 0"),
		key(s2, 0): translated(key(s2, 0), line(0), "譯文 0"),
	}); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{s1, s2} {
		b := mustNext(t, env, NextOptions{})
		if b == nil || !slices.Equal(b.Files, []string{file}) || len(b.Units) != 2 {
			t.Fatalf("batch = %+v, want the 2 units of %s alone", b, file)
		}
		checkUnitFiles(t, b)
		if want := []string{key(file, 0)}; !slices.Equal(ctxKeys(b), want) {
			t.Errorf("context of %s = %v, want %v", file, ctxKeys(b), want)
		}
	}

	// An RPG dialogue is a story: it does not join a batch of a plain file of
	// its family.
	const dialogue = "RPGSystem/rpg-loc-dialogue-1.json"
	kr = map[string]string{
		"Other.json": dataList(t, 1, "", line),
		dialogue:     `{"dataList":[{"key":"D1","texts":[{"index":0,"text":"이쪽이다."}]}]}`,
	}
	env = newEnv(t, kr, nil, nil)
	seed(t, env, append(items("Other.json", 0, 1), store.PendingItem{Key: dialogue + "|/dataList/0/texts/0/text", Reason: "new"})...)
	for _, file := range []string{"Other.json", dialogue} {
		if b := mustNext(t, env, NextOptions{}); b == nil || !slices.Equal(b.Files, []string{file}) {
			t.Errorf("batch = %+v, want %s alone", b, file)
		}
	}
}

func TestMergedBatchScopesTermsByUnitFile(t *testing.T) {
	const glossary = `- kr: 위력
  zh: 威力
  status: approved
  files: ["A.json"]
  check: hard
`
	kr := map[string]string{
		"A.json": `{"dataList":[{"id":0,"content":"위력이 강하다"},{"id":1,"content":"그냥 간다"}]}`,
		"B.json": oneLine(t, "위력이 약하다"),
	}
	env := newEnv(t, kr, nil, nil)
	env.Rules = testRulesWith(t, glossary)
	seed(t, env, append(items("A.json", 0, 1), items("B.json", 0, 1)...)...)

	b := mustNext(t, env, NextOptions{})
	if b == nil || !slices.Equal(b.Files, []string{"A.json", "B.json"}) {
		t.Fatalf("batch = %+v, want A.json and B.json", b)
	}
	if len(b.Glossary) != 1 || b.Glossary[0].KR != "위력" || !slices.Equal(b.Glossary[0].Files, []string{"A.json"}) {
		t.Errorf("glossary = %+v, want the term of A.json", b.Glossary)
	}
	// The term holds A's unit, not B's.
	res, err := Submit(env, b, []OutUnit{{Key: key("A.json", 0), ZH: "很強"}, {Key: key("B.json", 0), ZH: "很弱"}}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rejected) != 1 || res.Rejected[0].Key != key("A.json", 0) || !hasCode(res.Rejected[0].Issues, validate.CodeTermMissing) {
		t.Errorf("rejected = %+v, want A's unit for term-missing", res.Rejected)
	}
	if !slices.Equal(res.Accepted, []string{key("B.json", 0)}) {
		t.Errorf("accepted = %v, want B's unit: the term is not for B.json", res.Accepted)
	}

	// The term is in the glossary for A's unit only: with a unit of A that
	// does not contain it, B's unit brings no term.
	env = newEnv(t, kr, nil, nil)
	env.Rules = testRulesWith(t, glossary)
	seed(t, env, append(items("A.json", 1, 2), items("B.json", 0, 1)...)...)
	b = mustNext(t, env, NextOptions{})
	if b == nil || !slices.Equal(b.Files, []string{"A.json", "B.json"}) || len(b.Glossary) != 0 {
		t.Errorf("batch = %+v, want A.json and B.json with no glossary term", b)
	}
}

func TestMergedBatchGuidesCoverEverySpeaker(t *testing.T) {
	kr := map[string]string{
		"A.json": dataList(t, 1, "파우스트", line),
		"B.json": dataList(t, 1, "싱클레어", line),
		"C.json": dataList(t, 1, "", line),
	}
	env := newEnv(t, kr, nil, nil)
	seed(t, env, append(append(items("A.json", 0, 1), items("B.json", 0, 1)...), items("C.json", 0, 1)...)...)
	b := mustNext(t, env, NextOptions{})
	if b == nil || !slices.Equal(b.Files, []string{"A.json", "B.json", "C.json"}) {
		t.Fatalf("batch = %+v, want the three files", b)
	}
	var speakers []string
	for _, u := range b.Units {
		speakers = append(speakers, u.Speaker)
	}
	if want := []string{"faust", "sinclair", ""}; !slices.Equal(speakers, want) {
		t.Errorf("speakers = %q, want %q", speakers, want)
	}
	if want := []string{rules.BaseGuide, "guides/families.md", "guides/characters/faust.md", "guides/characters/sinclair.md"}; !slices.Equal(b.Guides, want) {
		t.Errorf("guides = %v, want %v", b.Guides, want)
	}
}

// Of twins in several files only one goes into the batch, and the others take
// its translation; a short text is shared within its file only, so the same
// short line of each file goes in.
func TestMergedBatchHoldsOneOfTwins(t *testing.T) {
	two := `{"dataList":[{"id":0,"content":` + q(t, twinText) + `},{"id":1,"content":"확인"}]}`
	env := newEnv(t, map[string]string{"A.json": two, "B.json": two}, nil, nil)
	seed(t, env, append(items("A.json", 0, 2), items("B.json", 0, 2)...)...)
	b := mustNext(t, env, NextOptions{})
	if want := []string{key("A.json", 0), key("A.json", 1), key("B.json", 1)}; b == nil || !slices.Equal(keysOf(b), want) {
		t.Fatalf("batch = %+v, want the keys %v", b, want)
	}
	checkUnitFiles(t, b)
	if l := pendingMap(t, env)[key("B.json", 0)].Lease; l != nil {
		t.Errorf("the twin in B.json is leased: %+v", l)
	}
	res, err := Submit(env, b, translate(b), "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 3 || !slices.Equal(res.Reused, []string{key("B.json", 0)}) {
		t.Errorf("accepted %v, reused %v, want all 3 accepted and the twin in B.json reused", res.Accepted, res.Reused)
	}
}

// Of two entries of one KR term and rendering, one for some files and one for
// the rest, a batch whose units are in both kinds of file shows both: each
// tells which units it holds.
func TestMergedBatchShowsTheTermOfEachFile(t *testing.T) {
	kr := map[string]string{"A.json": oneLine(t, "위력이 강하다"), "B.json": oneLine(t, "위력이 약하다")}
	env := newEnv(t, kr, nil, nil)
	env.Rules = testRulesWith(t, `- kr: 위력
  zh: 威力
  status: approved
  forbidden: [力量]
  files: ["A.json"]
- kr: 위력
  zh: 威力
  status: approved
`)
	seed(t, env, append(items("A.json", 0, 1), items("B.json", 0, 1)...)...)
	b := mustNext(t, env, NextOptions{})
	if b == nil || !slices.Equal(b.Files, []string{"A.json", "B.json"}) {
		t.Fatalf("batch = %+v, want A.json and B.json", b)
	}
	if len(b.Glossary) != 2 || !slices.Equal(b.Glossary[0].Files, []string{"A.json"}) || len(b.Glossary[1].Files) != 0 {
		t.Errorf("glossary = %+v, want the entry of A.json, then the one of every file", b.Glossary)
	}
}

// mirrorRouting is testRouting with the family mirror (*Mirror*), which comes
// before other, as in the real routing.
const mirrorRouting = `families:
  - id: story-main
    nodes: ["StoryData/S"]
    guide: guides/families.md
  - id: mirror
    nodes: ["*Mirror*"]
    guide: guides/families.md
  - id: other
    nodes: []
    guide: guides/families.md
priority:
  - story-main
  - mirror
  - other
`

// A batch of keyword files takes no other file of its family: the texts that
// use the names wait for the names to be translated, and then get them in
// the keywords of their batch.
func TestKeywordBatchTakesOnlyKeywordFiles(t *testing.T) {
	const bufs, gift = "Bufs_Mirror8.json", "EGOgift_Mirror8.json"
	kr := map[string]string{
		bufs: `{"dataList":[{"id":"Thunder","name":"우레"}]}`,
		gift: `{"dataList":[{"id":1,"desc":"[Thunder] 3 부여"}]}`,
	}
	env := newEnv(t, kr, nil, nil)
	env.Rules = testRulesWithRouting(t, mirrorRouting, testGlossary)
	nameKey, descKey := bufs+"|/dataList/0/name", gift+"|/dataList/0/desc"
	seed(t, env, store.PendingItem{Key: nameKey, Reason: "new"}, store.PendingItem{Key: descKey, Reason: "new"})

	b1 := mustNext(t, env, NextOptions{})
	if b1 == nil || !slices.Equal(b1.Files, []string{bufs}) || !slices.Equal(keysOf(b1), []string{nameKey}) {
		t.Fatalf("first batch = %+v, want the name in %s alone", b1, bufs)
	}
	res, err := Submit(env, b1, []OutUnit{{Key: nameKey, ZH: "雷鳴"}}, "m", 3)
	if err != nil || !slices.Equal(res.Accepted, []string{nameKey}) {
		t.Fatalf("submit of the name = %+v, %v, want it accepted", res, err)
	}
	b2 := mustNext(t, env, NextOptions{})
	if b2 == nil || !slices.Equal(b2.Files, []string{gift}) || !slices.Equal(keysOf(b2), []string{descKey}) {
		t.Fatalf("second batch = %+v, want the text of %s", b2, gift)
	}
	if want := map[string]string{"Thunder": "雷鳴"}; !reflect.DeepEqual(b2.Keywords, want) {
		t.Errorf("keywords of the second batch = %v, want %v: the name was translated first", b2.Keywords, want)
	}
}

// With keyword files in two families, every unit of a keyword file is handed
// out before any other unit; the keyword files of a family share a batch.
func TestNextHandsOutEveryKeywordUnitFirst(t *testing.T) {
	keywordFiles := []string{"BattleKeywords_Mirror8.json", "Bufs_Mirror8.json", "Bufs-a1c11p1.json"}
	kr := map[string]string{"EGOgift_Mirror8.json": dataList(t, 2, "", line), "Other.json": dataList(t, 2, "", line)}
	pending := append(items("EGOgift_Mirror8.json", 0, 2), items("Other.json", 0, 2)...)
	for _, f := range keywordFiles {
		kr[f] = dataList(t, 2, "", line)
		pending = append(pending, items(f, 0, 2)...)
	}
	env := newEnv(t, kr, nil, nil)
	env.Rules = testRulesWithRouting(t, mirrorRouting, testGlossary)
	seed(t, env, pending...)

	var got [][]string
	for b := mustNext(t, env, NextOptions{}); b != nil; b = mustNext(t, env, NextOptions{}) {
		checkUnitFiles(t, b)
		got = append(got, b.Files)
	}
	want := [][]string{
		{"BattleKeywords_Mirror8.json", "Bufs_Mirror8.json"}, // mirror comes before other
		{"Bufs-a1c11p1.json"},
		{"EGOgift_Mirror8.json"},
		{"Other.json"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("batches = %q, want %q", got, want)
	}
}

// krEnv is an env whose Other.json holds the given KR texts, one per unit,
// under the glossary, all of its units pending and cut into one batch.
func krEnv(t *testing.T, glossary string, texts ...string) (Env, *Batch) {
	t.Helper()
	env := newEnv(t, map[string]string{"Other.json": dataList(t, len(texts), "", func(i int) string { return texts[i] })}, nil, nil)
	env.Rules = testRulesWith(t, glossary)
	seed(t, env, items("Other.json", 0, len(texts))...)
	return env, mustNext(t, env, NextOptions{})
}

// hardTermMissing lists the Term of each hard term-missing among issues.
func hardTermMissing(issues []validate.Issue) []string {
	var out []string
	for _, is := range issues {
		if is.Code == validate.CodeTermMissing && is.Severity == validate.SeverityHard {
			out = append(out, is.Term)
		}
	}
	slices.Sort(out)
	return out
}

// The note has to write the term the way the entry matches it: an entry that
// hits only as a word is not named by a longer word that holds it.
func TestSubmitNoteNamesTheTermAsTheEntryMatches(t *testing.T) {
	const glossary = "- kr: 속도\n  zh: 速度\n  status: approved\n  match: word\n  check: hard\n"
	for _, tc := range []struct {
		note   string
		accept bool
	}{
		{"속도감：此處指節奏感", false},
		{"속도를：此處指別的東西", true},
		{"속도：此處指別的東西", true},
	} {
		env, b := krEnv(t, glossary, "속도가 빠르다")
		res, err := Submit(env, b, []OutUnit{{Key: key("Other.json", 0), ZH: "很快", Note: tc.note}}, "m", 3)
		if err != nil {
			t.Fatal(err)
		}
		if got := len(res.Accepted) == 1; got != tc.accept {
			t.Errorf("note %q: accepted = %v, want %v (result %+v)", tc.note, got, tc.accept, res)
		}
	}
}

// Two entries are missing and the note names one: the unit goes back for the
// other one, and the hard issue is only about that one.
func TestSubmitNoteNamesOnlyOneOfTwoMissingTerms(t *testing.T) {
	const glossary = "- kr: 시즌\n  zh: 賽季\n  status: approved\n  check: hard\n- kr: 탐식\n  zh: 暴食\n  status: approved\n  check: hard\n"
	env, b := krEnv(t, glossary, "시즌과 탐식")
	res, err := Submit(env, b, []OutUnit{{Key: key("Other.json", 0), ZH: "季和貪吃", Note: "시즌：劇中季節"}}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 0 || len(res.Rejected) != 1 {
		t.Fatalf("result = %+v, want one rejection", res)
	}
	if got := hardTermMissing(res.Rejected[0].Issues); !slices.Equal(got, []string{"탐식"}) {
		t.Errorf("hard term-missing = %v, want only 탐식", got)
	}
	last := pendingMap(t, env)[key("Other.json", 0)].LastError
	if !strings.Contains(last, "탐식") || strings.Contains(last, "시즌") {
		t.Errorf("LastError = %q, want it to name 탐식 and not 시즌", last)
	}
}

// A combined name inside the story, with the note naming the Korean term, is
// accepted, like the 시즌 case.
func TestSubmitAcceptsACombinedNameTheNoteNames(t *testing.T) {
	const glossary = "- kr: 탐식\n  zh: 暴食\n  status: approved\n  check: hard\n"
	env, b := krEnv(t, glossary, "탐식의 재림")
	res, err := Submit(env, b, []OutUnit{{Key: key("Other.json", 0), ZH: "貪吃的再臨", Note: "탐식：劇中組合名稱「탐식의 재림」的一部分"}}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 1 || len(res.Rejected) != 0 {
		t.Fatalf("result = %+v, want accepted", res)
	}
	if len(res.Warnings) != 1 || res.Warnings[0].Issues[0].Code != validate.CodeTermMissing || res.Warnings[0].Issues[0].Severity != validate.SeverityWarn {
		t.Errorf("warnings = %+v, want one term-missing warn", res.Warnings)
	}
	env, b = krEnv(t, glossary, "탐식의 재림")
	res, err = Submit(env, b, []OutUnit{{Key: key("Other.json", 0), ZH: "貪吃的再臨"}}, "m", 3)
	if err != nil || len(res.Rejected) != 1 {
		t.Errorf("without the note: %+v, %v, want rejected", res, err)
	}
}

// The same KR with the same rendering from another unit adds nothing; a third
// rendering is stored as a conflict. Through Submit, across two submissions.
func TestSubmitDedupesProvisionalTermsAndRecordsConflicts(t *testing.T) {
	env, b := krEnv(t, "", "우레가 쳤다", "우레가 울렸다")
	res, err := Submit(env, b, []OutUnit{{Key: key("Other.json", 0), ZH: "雷鳴響起", Terms: []NewTerm{{KR: "우레", ZH: "雷鳴"}}}}, "m", 3)
	if err != nil || res.Terms != 1 {
		t.Fatalf("first: %+v, %v, want 1 term", res, err)
	}
	// Unit 1 was not answered, so its lease ended; cut a new batch for it.
	b = mustNext(t, env, NextOptions{})
	res, err = Submit(env, b, []OutUnit{{Key: key("Other.json", 1), ZH: "雷鳴震動", Terms: []NewTerm{{KR: "우레", ZH: "雷鳴"}}}}, "m", 3)
	if err != nil || res.Terms != 0 || len(res.Accepted) != 1 {
		t.Fatalf("same rendering again: %+v, %v, want 0 terms", res, err)
	}
	if terms, _ := env.Store.LoadTerms(); len(terms) != 1 {
		t.Fatalf("terms = %+v, want one", terms)
	}

	env, b = krEnv(t, "", "우레가 쳤다", "우레가 울렸다", "우레가 멎었다")
	out := []OutUnit{
		{Key: key("Other.json", 0), ZH: "雷鳴響起", Terms: []NewTerm{{KR: "우레", ZH: "雷鳴"}}},
		{Key: key("Other.json", 1), ZH: "雷鳴震動", Terms: []NewTerm{{KR: "우레", ZH: "雷鳴"}}},
		{Key: key("Other.json", 2), ZH: "驚雷停了", Terms: []NewTerm{{KR: "우레", ZH: "驚雷"}}},
	}
	res, err = Submit(env, b, out, "m", 3)
	if err != nil || res.Terms != 2 {
		t.Fatalf("one batch: %+v, %v, want 2 terms", res, err)
	}
	terms, _ := env.Store.LoadTerms()
	if len(terms) != 2 || terms[0].ZH != "雷鳴" || terms[0].Conflict || terms[1].ZH != "驚雷" || !terms[1].Conflict {
		t.Errorf("terms = %+v, want 雷鳴 then 驚雷 as a conflict", terms)
	}
}

// A kr of spaces is empty, not too short.
func TestSubmitSkipsABlankTermKR(t *testing.T) {
	env, b := krEnv(t, "", "우레 가 쳤다")
	out := []OutUnit{{Key: key("Other.json", 0), ZH: "雷 鳴", Terms: []NewTerm{{KR: "  ", ZH: "雷"}, {KR: "우", ZH: "雷"}}}}
	res, err := Submit(env, b, out, "m", 3)
	if err != nil || len(res.TermsSkipped) != 2 || res.Terms != 0 {
		t.Fatalf("result = %+v, %v, want 2 skipped", res, err)
	}
	if got := res.TermsSkipped[0].Reason; got != "kr is empty" {
		t.Errorf("reason of a blank kr = %q, want %q", got, "kr is empty")
	}
	// The length is counted in characters, not bytes: one syllable is three
	// bytes and is still too short.
	if got := res.TermsSkipped[1].Reason; got != "kr is shorter than two characters" {
		t.Errorf("reason of a one-syllable kr = %q", got)
	}
}

// Only the provisional terms the batch carried (provisional: true in its
// glossary) and that are still in force are held to at submission.
func TestSubmitChecksTheBatchProvisionalTermsStillInForce(t *testing.T) {
	bad := []OutUnit{{Key: key("Other.json", 0), ZH: "打雷了"}}
	term := rules.Term{KR: "우레", ZH: "雷鳴", Match: rules.MatchWord, Status: rules.StatusApproved, Provisional: true}

	// Registered after the batch was cut: this batch is not held to it.
	env := provEnv(t)
	env.Provisional = nil
	seed(t, env, items("Other.json", 0, 1)...)
	b := mustNext(t, env, NextOptions{})
	env.Provisional = []rules.Term{term}
	if res, err := Submit(env, b, bad, "m", 3); err != nil || len(res.Accepted) != 1 || len(res.Warnings) != 0 {
		t.Errorf("registered after the batch: %+v, %v, want accepted", res, err)
	}

	// Carried by the batch and still in force: checked.
	env = provEnv(t)
	seed(t, env, items("Other.json", 0, 1)...)
	b = mustNext(t, env, NextOptions{})
	if res, err := Submit(env, b, bad, "m", 3); err != nil || len(res.Accepted) != 1 || len(res.Warnings) != 1 || !hasCode(res.Warnings[0].Issues, validate.CodeTermMissing) {
		t.Errorf("in force: %+v, %v, want accepted with a term-missing warning", res, err)
	}

	// Carried by the batch, declined or replaced by a glossary entry since:
	// no longer in env.Provisional, so no longer checked.
	env = provEnv(t)
	seed(t, env, items("Other.json", 0, 1)...)
	b = mustNext(t, env, NextOptions{})
	env.Provisional = nil
	if res, err := Submit(env, b, bad, "m", 3); err != nil || len(res.Accepted) != 1 || len(res.Warnings) != 0 {
		t.Errorf("no longer in force: %+v, %v, want accepted", res, err)
	}
}

// A tag may cut a provisional term in the KR text: the term still enters the
// batch glossary, and so is enforced at submission.
func TestProvisionalTermCutByATagIsInTheBatchAndEnforced(t *testing.T) {
	env := newEnv(t, map[string]string{"Other.json": `{"dataList":[{"id":0,"content":"우<b>레</b>가 쳤다"}]}`}, nil, nil)
	env.Provisional = []rules.Term{{KR: "우레", ZH: "雷鳴", Match: rules.MatchWord, Status: rules.StatusApproved, Provisional: true}}
	seed(t, env, items("Other.json", 0, 1)...)
	b := mustNext(t, env, NextOptions{})
	if len(b.Glossary) != 1 || b.Glossary[0].KR != "우레" || !b.Glossary[0].Provisional {
		t.Fatalf("glossary = %+v, want the provisional 우레", b.Glossary)
	}
	res, err := Submit(env, b, []OutUnit{{Key: key("Other.json", 0), ZH: "<b>打雷</b>了"}}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 1 || len(res.Warnings) != 1 || !hasCode(res.Warnings[0].Issues, validate.CodeTermMissing) {
		t.Errorf("result = %+v, want accepted with a term-missing warning", res)
	}
}

// The batch glossary only decides which provisional KRs are checked; the
// rendering checked is the one in force at submission.
func TestSubmitChecksTheCurrentProvisionalRendering(t *testing.T) {
	env := provEnv(t)
	seed(t, env, items("Other.json", 0, 1)...)
	b := mustNext(t, env, NextOptions{}) // the batch carries 우레 -> 雷鳴
	env.Provisional = []rules.Term{{KR: "우레", ZH: "驚雷", Match: rules.MatchWord, Status: rules.StatusApproved, Provisional: true}}
	if res, err := Submit(env, b, []OutUnit{{Key: key("Other.json", 0), ZH: "雷鳴響起"}}, "m", 3); err != nil || len(res.Warnings) != 1 ||
		len(res.Warnings[0].Issues) != 1 || res.Warnings[0].Issues[0].TermZH != "驚雷" {
		t.Errorf("old rendering: %+v, %v, want a term-missing warning for 驚雷, which is in force", res, err)
	}
	env = provEnv(t)
	seed(t, env, items("Other.json", 0, 1)...)
	b = mustNext(t, env, NextOptions{})
	env.Provisional = []rules.Term{{KR: "우레", ZH: "驚雷", Match: rules.MatchWord, Status: rules.StatusApproved, Provisional: true}}
	if res, err := Submit(env, b, []OutUnit{{Key: key("Other.json", 0), ZH: "驚雷響起"}}, "m", 3); err != nil || len(res.Accepted) != 1 || len(res.Warnings) != 0 {
		t.Errorf("current rendering: %+v, %v, want accepted without a warning", res, err)
	}
}

// A provisional term is weighed with the glossary terms: where its KR lies
// inside an approved term (와일드헌트 in 와일드헌트 히스클리프) it does not hit,
// so a text with only that one does not bring it into the batch glossary; a
// text that also has it alone does, after the approved term. (Until
// 2026-11-06 both came in for the first text too.)
func TestNextLeavesOutAProvisionalTermTheGlossaryCovers(t *testing.T) {
	for _, tc := range []struct {
		text string
		want []string
	}{
		{"와일드헌트 히스클리프가 왔다", []string{"와일드헌트 히스클리프=狂獵希斯克利夫"}},
		{"와일드헌트 히스클리프와 와일드헌트가 왔다", []string{"와일드헌트 히스클리프=狂獵希斯克利夫", "와일드헌트=狂獵"}},
	} {
		env := newEnv(t, map[string]string{"Other.json": oneLine(t, tc.text)}, nil, nil)
		env.Rules = testRulesWith(t, "- kr: 와일드헌트 히스클리프\n  zh: 狂獵希斯克利夫\n  status: approved\n")
		env.Provisional = ActiveProvisional([]store.ProvisionalTerm{provTerm("와일드헌트", "狂獵", false)}, env.Rules, nil)
		seed(t, env, items("Other.json", 0, 1)...)
		b := mustNext(t, env, NextOptions{})
		var got []string
		for _, g := range b.Glossary {
			got = append(got, g.KR+"="+g.ZH)
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s: glossary = %v, want %v", tc.text, got, tc.want)
		}
	}
}

// The batch glossary is the union over the texts of what each one is held
// to: a provisional term covered in one text comes in for another text
// where it stands alone.
func TestNextCarriesAProvisionalTermOneTextHasAlone(t *testing.T) {
	kr := map[string]string{"Other.json": `{"dataList":[
 {"id":0,"content":"와일드헌트 히스클리프가 왔다"},
 {"id":1,"content":"와일드헌트가 왔다"}
]}`}
	env := newEnv(t, kr, nil, nil)
	env.Rules = testRulesWith(t, "- kr: 와일드헌트 히스클리프\n  zh: 狂獵希斯克利夫\n  status: approved\n")
	env.Provisional = ActiveProvisional([]store.ProvisionalTerm{provTerm("와일드헌트", "荒獵", false)}, env.Rules, nil)
	seed(t, env, items("Other.json", 0, 2)...)
	b := mustNext(t, env, NextOptions{})
	var got []string
	for _, g := range b.Glossary {
		got = append(got, g.KR+"="+g.ZH)
	}
	if want := []string{"와일드헌트 히스클리프=狂獵希斯克利夫", "와일드헌트=荒獵"}; !slices.Equal(got, want) {
		t.Fatalf("glossary = %v, want %v", got, want)
	}
	// The unit where the approved term covers it is not held to it.
	res, err := Submit(env, b, []OutUnit{{Key: key("Other.json", 0), ZH: "狂獵希斯克利夫來了"}, {Key: key("Other.json", 1), ZH: "荒獵來了"}}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 2 || len(res.Warnings) != 0 {
		t.Errorf("result = %+v, want both accepted without warnings", res)
	}
}

// Submission holds a text to the glossary hits and the provisional terms that
// occur in it, weighed together: a term each of whose hits lies inside a
// longer one's is not checked, whichever of the two is provisional.
func TestCheckNewTermsWeighsProvisionalTermsWithTheGlossary(t *testing.T) {
	r := testRulesWith(t, "- kr: 와일드헌트 히스클리프\n  zh: 狂獵希斯克利夫\n  status: approved\n")
	prov := ActiveProvisional([]store.ProvisionalTerm{provTerm("와일드헌트", "荒獵", false)}, r, nil)
	if issues := CheckNewTerms(r, prov, "Other.json", "와일드헌트 히스클리프가 왔다", "狂獵希斯克利夫來了"); hasCode(issues, validate.CodeTermMissing) {
		t.Errorf("covered provisional term: issues = %+v, want no term-missing", issues)
	}
	issues := CheckNewTerms(r, prov, "Other.json", "와일드헌트 히스클리프와 와일드헌트", "狂獵希斯克利夫與狂獵")
	if !slices.ContainsFunc(issues, func(is validate.Issue) bool {
		return is.Code == validate.CodeTermMissing && is.Term == "와일드헌트"
	}) {
		t.Errorf("provisional term also alone: issues = %+v, want term-missing for 와일드헌트", issues)
	}

	// The other way round: a provisional term covers a glossary term.
	r2 := testRulesWith(t, "- kr: 와일드헌트\n  zh: 狂獵\n  status: approved\n")
	prov2 := ActiveProvisional([]store.ProvisionalTerm{provTerm("와일드헌트 히스클리프", "荒獵希斯克利夫", false)}, r2, nil)
	if len(prov2) != 1 {
		t.Fatalf("provisional = %+v, want 와일드헌트 히스클리프 in force", prov2)
	}
	if issues := CheckNewTerms(r2, prov2, "Other.json", "와일드헌트 히스클리프가 왔다", "荒獵希斯克利夫來了"); hasCode(issues, validate.CodeTermMissing) {
		t.Errorf("covered glossary term: issues = %+v, want no term-missing", issues)
	}
	if issues := CheckNew(r2, "Other.json", "와일드헌트 히스클리프가 왔다", "荒獵希斯克利夫來了"); !hasCode(issues, validate.CodeTermMissing) {
		t.Errorf("CheckNew = %+v, want term-missing: without the provisional term nothing covers 와일드헌트", issues)
	}
}

// TermsIn marks the provisional terms it returns, whether or not they came
// marked, and ValidateTermsIn, what an existing translation is checked
// against, leaves them out: they only cover glossary terms there.
func TestTermsInMarksProvisionalTerms(t *testing.T) {
	r := testRulesWith(t, "- kr: 와일드헌트\n  zh: 狂獵\n  status: approved\n")
	prov := []rules.Term{{KR: "와일드헌트 히스클리프", ZH: "荒獵希斯克利夫", Match: rules.MatchWord, Status: rules.StatusApproved}}
	got := TermsIn(r, prov, "Other.json", "와일드헌트 히스클리프가 왔다")
	if len(got) != 1 || got[0].KR != "와일드헌트 히스클리프" || !got[0].Provisional {
		t.Errorf("TermsIn = %+v, want the provisional term alone, marked", got)
	}
	if prov[0].Provisional {
		t.Error("TermsIn marked the caller's term")
	}
	if got := ValidateTermsIn(r, prov, "Other.json", "와일드헌트 히스클리프가 왔다"); len(got) != 0 {
		t.Errorf("ValidateTermsIn = %+v, want none: the glossary term is covered, the provisional one not enforced", got)
	}
	if got := ValidateTermsIn(r, prov, "Other.json", "와일드헌트가 왔다"); len(got) != 1 || got[0].KR != "와일드헌트" {
		t.Errorf("ValidateTermsIn = %+v, want the glossary term", got)
	}
}

// A provisional term whose KR is not in the text is neither a coverer nor
// checked, and TermsIn gives what a scan of every provisional term gives (the
// first-rune prefilter changes no result), for odd texts too.
func TestTermsInIgnoresAProvisionalTermAbsentFromTheKR(t *testing.T) {
	r := testRulesWith(t, "- kr: 보스\n  zh: BOSS\n  status: approved\n")
	prov := []rules.Term{{KR: "조직의 보스", ZH: "組織的首領", Match: rules.MatchWord, Status: rules.StatusApproved}}
	got := TermsIn(r, prov, "Other.json", "보스가 왔다")
	if len(got) != 1 || got[0].KR != "보스" || got[0].Provisional {
		t.Errorf("TermsIn = %+v, want the glossary term 보스 alone: the provisional term is absent from the KR, so it covers nothing", got)
	}
	if got := ValidateTermsIn(r, prov, "Other.json", "보스가 왔다"); len(got) != 1 || got[0].KR != "보스" {
		t.Errorf("ValidateTermsIn = %+v, want 보스 checked", got)
	}
	if got := ValidateTermsIn(r, prov, "Other.json", "조직의 보스가 왔다"); len(got) != 0 {
		t.Errorf("ValidateTermsIn = %+v, want none once the provisional term is present and covers 보스", got)
	}

	// Against a scan of every provisional term, over texts of every kind.
	many := []rules.Term{
		{KR: "조직", ZH: "組織", Match: rules.MatchWord},
		{KR: "조직의 보스", ZH: "組織的首領", Match: rules.MatchWord},
		{KR: "ab", ZH: "AB", Match: rules.MatchSubstring},
		{KR: "x", ZH: "X", Match: rules.MatchSubstring},
		{KR: "보\xff", ZH: "壞二", Match: rules.MatchSubstring},
		{KR: "\xff\xfe", ZH: "壞", Match: rules.MatchSubstring},
		{KR: "\ufffd", ZH: "替", Match: rules.MatchSubstring},
		{KR: "", ZH: "空"},
		{KR: "<br>", ZH: "換行", Match: rules.MatchSubstring},
		{KR: " 우레", ZH: "雷鳴", Match: rules.MatchWord},
		{KR: " 우레", ZH: "雷鳴二", Match: rules.MatchSubstring},
		{KR: "우레", ZH: "雷", Match: rules.MatchWord},
		{KR: "우레", ZH: "雷二", Match: rules.MatchSubstring},
	}
	scan := func(file, kr string) []rules.Term {
		terms := r.GlossaryHits(file, kr)
		plain := validate.MatchText(kr)
		n := len(terms)
		for _, p := range many {
			if p.OccursIn(plain) {
				p.Provisional = true
				terms = append(terms, p)
			}
		}
		if len(terms) == n {
			return terms
		}
		return rules.DropCovered(plain, terms)
	}
	for _, kr := range []string{"", "보스", "조직의 보스가 왔다", "조직이 보스를 불렀다", "xabx", "\xff\xfe\xff", "a\xffb", "x\ufffdy", "줄<br>바꿈", "あいう", "큰 우레가 쳤다", "우레탄을 발랐다", "우레 우레탄", " 우레", "우레"} {
		got, want := TermsIn(r, many, "Other.json", kr), scan("Other.json", kr)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("TermsIn(%q) = %+v, want %+v", kr, got, want)
		}
	}
}

// The forbidden renderings of a term are checked where it hits: not where a
// longer term covers it.
func TestCheckNewSkipsTheForbiddenOfACoveredTerm(t *testing.T) {
	r := testRulesWith(t, `- kr: 보스
  zh: BOSS
  status: approved
  forbidden: [首領]
- kr: 조직의 보스
  zh: 組織的首領
  status: approved
`)
	if issues := CheckNew(r, "Other.json", "조직의 보스가 왔다", "組織的首領來了"); validate.HasHard(issues) || hasCode(issues, validate.CodeForbiddenTerm) || hasCode(issues, validate.CodeTermMissing) {
		t.Errorf("covered 보스: issues = %+v, want none", issues)
	}
	issues := CheckNew(r, "Other.json", "조직의 보스가 보스 전투에 나섰다", "組織的首領參加首領戰")
	if !hasCode(issues, validate.CodeForbiddenTerm) || !hasCode(issues, validate.CodeTermMissing) {
		t.Errorf("a 보스 outside: issues = %+v, want forbidden-term and term-missing", issues)
	}
}

// In a merged batch, a unit of a later file whose twin is leased to another
// batch is left out; the other units of that file go in as usual.
func TestMergedBatchLeavesOutAUnitWhoseTwinIsLeasedElsewhere(t *testing.T) {
	kr := map[string]string{
		"A.json": oneLine(t, "첫 번째 파일의 문장"),
		"B.json": `{"dataList":[{"id":0,"content":` + q(t, twinText) + `},{"id":1,"content":"다른 문장"}]}`,
		"C.json": oneLine(t, twinText),
	}
	env := newEnv(t, kr, nil, nil)
	pending := append(items("A.json", 0, 1), items("B.json", 0, 2)...)
	pending = append(pending, items("C.json", 0, 1)...)
	pending[3].Lease = &store.Lease{Batch: "20261101T000000-deadbeef", Until: testNow.Add(time.Hour)}
	seed(t, env, pending...)
	b := mustNext(t, env, NextOptions{})
	want := []string{key("A.json", 0), key("B.json", 1)}
	if b == nil || !slices.Equal(keysOf(b), want) {
		t.Fatalf("batch = %+v, want the keys %v", b, want)
	}
	if it := pendingMap(t, env)[key("B.json", 0)]; it.Lease != nil {
		t.Errorf("the twin is leased: %+v", it.Lease)
	}
}

// ShareKey is what files must have in common to share a batch: the family
// and whether they name keywords; a story file shares with none.
func TestShareKey(t *testing.T) {
	r := testRules(t)
	key := func(file string) string {
		k, ok := ShareKey(r, file)
		if !ok {
			return ""
		}
		return k
	}
	if _, ok := ShareKey(r, "StoryData/S001A.json"); ok {
		t.Error("a story file has a share key")
	}
	if key("A.json") == "" || key("A.json") != key("B.json") {
		t.Errorf("A.json = %q, B.json = %q, want one key", key("A.json"), key("B.json"))
	}
	if key("Bufs.json") == key("A.json") {
		t.Errorf("a keyword file shares the key %q with a plain file of its family", key("A.json"))
	}
	if key("Bufs.json") != key("BattleKeywords.json") {
		t.Errorf("keyword files Bufs.json = %q, BattleKeywords.json = %q, want one key", key("Bufs.json"), key("BattleKeywords.json"))
	}
}

// A text that stands in a batch twice (here under two fields, which are not
// twins) is collected once for the keyword lookup.
func TestDistinctKRs(t *testing.T) {
	texts := []unitText{{"A.json", "[Bleed] 하나", "a"}, {"A.json", "둘", "b"}, {"B.json", "[Bleed] 하나", "c"}}
	if got, want := distinctKRs(texts), []string{"[Bleed] 하나", "둘"}; !slices.Equal(got, want) {
		t.Errorf("distinctKRs = %q, want %q", got, want)
	}
}

// The keywords of a batch with the same text under two fields are all there.
func TestNextNamesKeywordsOfARepeatedText(t *testing.T) {
	kr := map[string]string{
		"A.json":    `{"dataList":[{"id":0,"name":"[Bleed] 하나","desc":"[Bleed] 하나"}]}`,
		"Bufs.json": `{"dataList":[{"id":"Bleed","name":"출혈"}]}`,
	}
	env := newEnv(t, kr, nil, nil)
	if err := env.Store.SaveRecords(map[string]store.Record{
		"Bufs.json|/dataList/0/name": {Key: "Bufs.json|/dataList/0/name", KRSha: units.KRSha("출혈"), ZH: "流血", Origin: store.OriginParatranz, Since: testTag},
	}); err != nil {
		t.Fatal(err)
	}
	seed(t, env, store.PendingItem{Key: "A.json|/dataList/0/name", Reason: "new"}, store.PendingItem{Key: "A.json|/dataList/0/desc", Reason: "new"})
	b := mustNext(t, env, NextOptions{})
	if b == nil || len(b.Units) != 2 || b.Keywords["Bleed"] != "流血" {
		t.Errorf("batch = %+v, want two units and the keyword Bleed", b)
	}
}

// An entry matches as a word unless it says match: substring, so a batch only
// brings a longer word's entry when the entry asks for it, and says so in its
// glossary: the translator sees match: substring, the one value that is not
// the default.
func TestBatchGlossaryMatch(t *testing.T) {
	const glossary = "- kr: 속도\n  zh: 速度\n  status: approved\n- kr: 사무소\n  zh: 事務所\n  status: approved\n  match: substring\n"
	_, b := krEnv(t, glossary, "약속도 정사무소")
	if len(b.Glossary) != 1 || b.Glossary[0].KR != "사무소" || b.Glossary[0].Match != rules.MatchSubstring {
		t.Fatalf("Glossary = %+v, want only 사무소 with match substring (속도 is no word of 약속도)", b.Glossary)
	}
	data, err := json.Marshal(b.Glossary)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"match":"substring"`) {
		t.Errorf("glossary JSON = %s, want match: substring", data)
	}
}

// A merged batch holds units of several files, so an entry that is excluded
// from one of them carries its not_files, as it carries its files: the
// translator and the schema see where it does not apply.
func TestMergedBatchCarriesNotFiles(t *testing.T) {
	const glossary = "- kr: 위력\n  zh: 威力\n  status: approved\n  not_files: [\"B.json\"]\n"
	kr := map[string]string{
		"A.json": oneLine(t, "위력이 강하다"),
		"B.json": oneLine(t, "위력이 약하다"),
	}
	env := newEnv(t, kr, nil, nil)
	env.Rules = testRulesWith(t, glossary)
	seed(t, env, append(items("A.json", 0, 1), items("B.json", 0, 1)...)...)
	b := mustNext(t, env, NextOptions{})
	if b == nil || !slices.Equal(b.Files, []string{"A.json", "B.json"}) {
		t.Fatalf("batch = %+v, want A.json and B.json", b)
	}
	if len(b.Glossary) != 1 || !slices.Equal(b.Glossary[0].NotFiles, []string{"B.json"}) {
		t.Fatalf("glossary = %+v, want the one term with not_files [B.json]", b.Glossary)
	}
	data, err := json.Marshal(b.Glossary)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"not_files":["B.json"]`) {
		t.Errorf("glossary JSON = %s, want not_files", data)
	}
}

// Entries that differ only in not_files are two entries of the batch.
func TestGlossaryForKeepsEntriesApartByNotFiles(t *testing.T) {
	r := testRulesWith(t, "- kr: 위력\n  zh: 威力\n  status: approved\n  files: [\"*.json\"]\n  not_files: [\"B.json\"]\n- kr: 위력\n  zh: 威力\n  status: approved\n  files: [\"*.json\"]\n  not_files: [\"A.json\"]\n")
	got := glossaryFor(r, nil, []unitText{{"A.json", "위력", "a"}, {"B.json", "위력", "b"}})
	if len(got) != 2 {
		t.Errorf("glossaryFor = %+v, want both entries", got)
	}
}

// A family: scope goes into the batch as written, and the entry is in the batch
// only if a unit of its family has the word; the translator reads
// family:<id> against the batch's family.
func TestBatchGlossaryCarriesFamilyScope(t *testing.T) {
	const glossary = "- kr: 위력\n  zh: 威力\n  status: approved\n  files: [\"family:story-main\"]\n" +
		"- kr: 위력\n  zh: 力量\n  status: approved\n  not_files: [\"family:story-main\"]\n"
	for file, want := range map[string]struct{ zh, files, notFiles string }{
		"StoryData/S1.json": {"威力", "family:story-main", ""},
		"A.json":            {"力量", "", "family:story-main"},
	} {
		kr := map[string]string{file: oneLine(t, "위력이 강하다")}
		env := newEnv(t, kr, nil, nil)
		env.Rules = testRulesWith(t, glossary)
		seed(t, env, items(file, 0, 1)...)
		b := mustNext(t, env, NextOptions{})
		if b == nil || len(b.Glossary) != 1 {
			t.Fatalf("%s: batch = %+v, want the one entry for its family", file, b)
		}
		g := b.Glossary[0]
		if g.ZH != want.zh || strings.Join(g.Files, ",") != want.files || strings.Join(g.NotFiles, ",") != want.notFiles {
			t.Errorf("%s: glossary = %+v, want %+v", file, g, want)
		}
		data, _ := json.Marshal(g)
		if !strings.Contains(string(data), "family:story-main") {
			t.Errorf("%s: glossary JSON = %s, want the family: token as written", file, data)
		}
	}
}

const alsoGlossary = `- kr: 시즌
  zh: 賽季
  also: [季度]
  status: approved
`

// An also rendering is as good as the zh: no rejection, no warning, no note.
func TestSubmitAcceptsAnAlsoRenderingWithoutANote(t *testing.T) {
	kr := map[string]string{"Other.json": dataList(t, 1, "", func(int) string { return "두 시즌 전부터" })}
	env := newEnv(t, kr, nil, nil)
	env.Rules = testRulesWith(t, alsoGlossary)
	seed(t, env, items("Other.json", 0, 1)...)
	b := mustNext(t, env, NextOptions{})
	res, err := Submit(env, b, []OutUnit{{Key: key("Other.json", 0), ZH: "從兩個季度前起"}}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 1 || len(res.Rejected) != 0 || len(res.Warnings) != 0 {
		t.Errorf("result = %+v, want accepted without warnings", res)
	}
	if len(b.Glossary) != 1 || !slices.Equal(b.Glossary[0].Also, []string{"季度"}) {
		t.Fatalf("glossary = %+v, want the term with also", b.Glossary)
	}
	data, err := json.Marshal(b.Glossary)
	if err != nil || !strings.Contains(string(data), `"also":["季度"]`) {
		t.Errorf("glossary JSON = %s, %v, want also", data, err)
	}
}

// A glossary entry of the batch lists the units that use it: the scope, the
// longer entries and the provisional terms are already counted in.
func TestBatchGlossaryUnits(t *testing.T) {
	r := testRulesWith(t, "- kr: 보스\n  zh: BOSS\n  status: approved\n  files: [\"group:Skills_Enemy*\"]\n"+
		"- kr: 조직의 보스\n  zh: 組織首領\n  status: approved\n  files: [\"group:Skills_Enemy*\"]\n")
	prov := []rules.Term{{KR: "임시", ZH: "暫定", Status: rules.StatusCandidate}}
	got := glossaryFor(r, prov, []unitText{
		{"Skills_Enemy-x.json", "보스가 온다 임시", "u1"},
		{"Skills_personality-01.json", "보스가 온다 임시", "u2"},
		{"Skills_Enemy-x.json", "조직의 보스가 온다", "u3"},
	})
	units := map[string][]string{}
	for _, g := range got {
		units[g.KR] = g.Units
	}
	if !slices.Equal(units["보스"], []string{"u1"}) {
		t.Errorf("보스 units = %q, want [u1]", units["보스"])
	}
	if !slices.Equal(units["조직의 보스"], []string{"u3"}) {
		t.Errorf("조직의 보스 units = %q, want [u3]", units["조직의 보스"])
	}
	if !slices.Equal(units["임시"], []string{"u1", "u2"}) {
		t.Errorf("임시 units = %q, want [u1 u2]", units["임시"])
	}
}

// A missing glossary rendering is rejected at submission only for an entry
// with check: hard (in a family that does not say glossary: warn), and only
// when the note does not name the term; every other entry, and a provisional
// term, only warns. A forbidden rendering is rejected whatever check says.
func TestSubmitGlossaryCheck(t *testing.T) {
	const (
		plain = "- kr: 시즌\n  zh: 賽季\n  status: approved\n  forbidden: [季前]\n"
		hard  = plain + "  check: hard\n"
	)
	run := func(t *testing.T, routing, glossary, file, zh, note string) *SubmitResult {
		t.Helper()
		env := newEnv(t, map[string]string{file: oneLine(t, "두 시즌 전부터")}, nil, nil)
		if routing == "" {
			env.Rules = testRulesWith(t, glossary)
		} else {
			env.Rules = testRulesWithRouting(t, routing, glossary)
		}
		seed(t, env, items(file, 0, 1)...)
		res, err := Submit(env, mustNext(t, env, NextOptions{}), []OutUnit{{Key: key(file, 0), ZH: zh, Note: note}}, "m", 3)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	warned := func(t *testing.T, res *SubmitResult) {
		t.Helper()
		if len(res.Accepted) != 1 || len(res.Rejected) != 0 || len(res.Warnings) != 1 ||
			!hasCode(res.Warnings[0].Issues, validate.CodeTermMissing) || validate.HasHard(res.Warnings[0].Issues) {
			t.Errorf("result = %+v, want accepted with a term-missing warning", res)
		}
	}
	rejected := func(t *testing.T, res *SubmitResult, code string) {
		t.Helper()
		if len(res.Accepted) != 0 || len(res.Rejected) != 1 || !hasCode(res.Rejected[0].Issues, code) {
			t.Errorf("result = %+v, want rejected for %s", res, code)
		}
	}
	t.Run("no check: a missing rendering only warns", func(t *testing.T) {
		warned(t, run(t, "", plain, "Other.json", "從兩個賽局前起", ""))
	})
	t.Run("check warn: a missing rendering only warns", func(t *testing.T) {
		warned(t, run(t, "", plain+"  check: warn\n", "Other.json", "從兩個賽局前起", ""))
	})
	t.Run("check hard: rejected without the term in the note", func(t *testing.T) {
		rejected(t, run(t, "", hard, "Other.json", "從兩個賽局前起", ""), validate.CodeTermMissing)
		rejected(t, run(t, "", hard, "Other.json", "從兩個賽局前起", "탐식：卡包名"), validate.CodeTermMissing)
	})
	t.Run("check hard: accepted when the note names the term", func(t *testing.T) {
		warned(t, run(t, "", hard, "Other.json", "從兩個賽局前起", "시즌：劇中季節"))
	})
	t.Run("check hard in a warn family only warns", func(t *testing.T) {
		warned(t, run(t, warnRouting, hard, "Warn.json", "從兩個賽局前起", ""))
	})
	t.Run("forbidden is rejected whatever check says", func(t *testing.T) {
		rejected(t, run(t, "", plain, "Other.json", "從兩棵季前起", ""), "forbidden-term")
	})
}

func TestSubmitProvisionalTermOnlyWarns(t *testing.T) {
	env := provEnv(t)
	seed(t, env, items("Other.json", 0, 1)...)
	b := mustNext(t, env, NextOptions{})
	res, err := Submit(env, b, []OutUnit{{Key: key("Other.json", 0), ZH: "打雷了"}}, "m", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Accepted) != 1 || len(res.Rejected) != 0 || len(res.Warnings) != 1 ||
		!hasCode(res.Warnings[0].Issues, validate.CodeTermMissing) || validate.HasHard(res.Warnings[0].Issues) {
		t.Errorf("result = %+v, want accepted with a term-missing warning", res)
	}
}
