package plan

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

const (
	storyFile = "StoryData/S1.json"
	bgmFile   = "BgmLyrics/L.json"
	bufsFile  = "Bufs.json"
)

var frozen = []string{"BgmLyrics/**"}

// openKR writes the KR files (file name to JSON text) under a temporary
// directory and opens them as a KR pack.
func openKR(t *testing.T, files map[string]string) *pack.Pack {
	t.Helper()
	root := t.TempDir()
	for file, text := range files {
		dir, name := filepath.Split(file)
		full := filepath.Join(root, filepath.FromSlash(dir), "KR_"+name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p, err := pack.Open(root, "KR_")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// story is a story file with one line per element of lines, numbered from 0.
func story(lines ...string) string {
	var parts []string
	for i, l := range lines {
		parts = append(parts, fmt.Sprintf(`{"id":%d,"content":%q}`, i, l))
	}
	return `{"dataList":[` + strings.Join(parts, ",") + `]}`
}

func contentKey(file string, i int) string {
	return units.Key(file, fmt.Sprintf("/dataList/%d/content", i))
}

// translated is a record of the unit key whose KR text is kr.
func translated(key, kr, zh string) store.Record {
	return store.Record{Key: key, KRSha: units.KRSha(kr), ZH: zh, Origin: store.OriginClaude, Since: "raw-20261029", Rules: "abc123", Model: "m1", Batch: "b-1"}
}

func recordsOf(rs ...store.Record) map[string]store.Record {
	m := map[string]store.Record{}
	for _, r := range rs {
		m[r.Key] = r
	}
	return m
}

func transition(t *testing.T, oldFiles, newFiles map[string]string, recs map[string]store.Record, pending []store.PendingItem) *Moved {
	t.Helper()
	m, err := Transition(openKR(t, oldFiles), openKR(t, newFiles), recs, pending, frozen)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func pendingKeys(m *Moved) []string {
	var keys []string
	for _, p := range m.Pending {
		keys = append(keys, p.Key)
	}
	return keys
}

func mustEqual(t *testing.T, what string, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("%s = %q, want %q", what, got, want)
	}
}

func TestTransitionStoryInsertMovesRecords(t *testing.T) {
	oldLines := []string{"첫째 줄", "둘째 줄", "셋째 줄", "넷째 줄", "다섯째 줄"}
	newLines := []string{"첫째 줄", "새로 끼운 줄", "둘째 줄", "셋째 줄", "넷째 줄", "다섯째 줄"}
	var all []store.Record
	for i, l := range oldLines {
		all = append(all, translated(contentKey(storyFile, i), l, fmt.Sprintf("舊譯 %d", i)))
	}

	m := transition(t, map[string]string{storyFile: story(oldLines...)}, map[string]string{storyFile: story(newLines...)}, recordsOf(all...), nil)

	if len(m.Records) != 5 {
		t.Fatalf("got %d records, want 5: %v", len(m.Records), m.Records)
	}
	for i, l := range oldLines {
		to := i
		if i >= 1 {
			to = i + 1
		}
		key := contentKey(storyFile, to)
		got, ok := m.Records[key]
		if !ok {
			t.Fatalf("record of line %d is not at %s", i, key)
		}
		want := all[i]
		want.Key = key
		if !reflect.DeepEqual(got, want) {
			t.Errorf("record at %s = %+v, want %+v", key, got, want)
		}
		if got.KRSha != units.KRSha(l) {
			t.Errorf("record at %s lost its kr_sha", key)
		}
	}
	mustEqual(t, "pending keys", pendingKeys(m), []string{contentKey(storyFile, 1)})
	if p := m.Pending[0]; p.Reason != "new" || p.PrevKR != "" || p.PrevZH != "" || p.Lease != nil {
		t.Errorf("pending = %+v, want a bare new item", p)
	}
	want := Changes{Unchanged: 5, Moved: 4, New: 1, FilesChanged: []string{storyFile}, FilesShifted: []string{storyFile}}
	if !reflect.DeepEqual(m.Changes, want) {
		t.Errorf("Changes = %+v, want %+v", m.Changes, want)
	}
	if len(m.Deleted) != 0 || len(m.FrozenChanged) != 0 {
		t.Errorf("Deleted = %v, FrozenChanged = %v, want none", m.Deleted, m.FrozenChanged)
	}
}

func TestTransitionDuplicateLines(t *testing.T) {
	oldLines := []string{"첫째 줄", "……", "둘째 줄", "……", "마지막 줄"}
	newLines := []string{"첫째 줄", "둘째 줄", "……", "마지막 줄"}
	zh := []string{"一", "第一個省略號", "二", "第二個省略號", "末"}
	var all []store.Record
	for i, l := range oldLines {
		all = append(all, translated(contentKey(storyFile, i), l, zh[i]))
	}

	m := transition(t, map[string]string{storyFile: story(oldLines...)}, map[string]string{storyFile: story(newLines...)}, recordsOf(all...), nil)

	if len(m.Pending) != 0 {
		t.Errorf("pending = %v, want none", m.Pending)
	}
	if len(m.Records) != 4 {
		t.Fatalf("got %d records, want 4", len(m.Records))
	}
	if got := m.Records[contentKey(storyFile, 2)].ZH; got != "第二個省略號" {
		t.Errorf("the remaining … has zh %q, want the record of the second one", got)
	}
	if len(m.Deleted) != 1 || m.Deleted[0].KR != "……" || m.Deleted[0].Record == nil || m.Deleted[0].Record.ZH != "第一個省略號" {
		t.Errorf("Deleted = %+v, want the first …", m.Deleted)
	} else if m.Deleted[0].Pointer != "/dataList/1/content" || m.Deleted[0].Record.Key != contentKey(storyFile, 1) {
		t.Errorf("Deleted[0] = %+v, want the old pointer /dataList/1/content", m.Deleted[0])
	}
	if m.Changes.Deleted != 1 || m.Changes.Unchanged != 4 || m.Changes.New != 0 {
		t.Errorf("Changes = %+v", m.Changes)
	}
}

func TestTransitionChangedQueuesWithPrevious(t *testing.T) {
	const oldKR, newKR = "안녕하세요 오늘은 날씨가 좋네요", "안녕하세요 오늘은 날씨가 나쁘네요"
	oldLines := []string{"첫째 줄", oldKR, "셋째 줄", "넷째 줄"}
	newLines := []string{"첫째 줄", newKR, "셋째 줄", "넷째 줄"}
	var all []store.Record
	for i, l := range oldLines {
		all = append(all, translated(contentKey(storyFile, i), l, fmt.Sprintf("舊譯 %d", i)))
	}

	m := transition(t, map[string]string{storyFile: story(oldLines...)}, map[string]string{storyFile: story(newLines...)}, recordsOf(all...), nil)

	key := contentKey(storyFile, 1)
	if _, ok := m.Records[key]; ok {
		t.Errorf("the record of the edited line is still there")
	}
	if len(m.Records) != 3 {
		t.Errorf("got %d records, want 3", len(m.Records))
	}
	if len(m.Pending) != 1 {
		t.Fatalf("pending = %+v, want 1 item", m.Pending)
	}
	want := store.PendingItem{Key: key, Reason: "kr-changed", PrevKR: oldKR, PrevZH: "舊譯 1"}
	if !reflect.DeepEqual(m.Pending[0], want) {
		t.Errorf("pending = %+v, want %+v", m.Pending[0], want)
	}
	if m.Changes.Changed != 1 || m.Changes.Unchanged != 3 || m.Changes.Moved != 0 || m.Changes.New != 0 || m.Changes.Deleted != 0 {
		t.Errorf("Changes = %+v", m.Changes)
	}
	mustEqual(t, "FilesChanged", m.Changes.FilesChanged, []string{storyFile})
	if len(m.Changes.FilesShifted) != 0 {
		t.Errorf("FilesShifted = %v, want none", m.Changes.FilesShifted)
	}
}

func TestTransitionChangedWithoutRecordOrWithHint(t *testing.T) {
	oldLines := []string{"첫째 줄", "안녕하세요 오늘은 날씨가 좋네요", "셋째 줄", "안녕하세요 내일은 비가 올 거예요", "다섯째 줄"}
	newLines := []string{"첫째 줄", "안녕하세요 오늘은 날씨가 나쁘네요", "셋째 줄", "안녕하세요 내일은 눈이 올 거예요", "다섯째 줄"}
	// Line 1 was never translated; line 3 was sent back with a hint.
	recs := recordsOf(translated(contentKey(storyFile, 3), oldLines[3], "舊譯 3"))
	pending := []store.PendingItem{
		{Key: contentKey(storyFile, 1), Reason: "new", Attempts: 2, Draft: "舊草稿", Lease: &store.Lease{Batch: "b-9", Until: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)}},
		{Key: contentKey(storyFile, 3), Reason: "requeue", PrevZH: "舊譯 3", Hint: "改用「您」", Attempts: 1},
	}

	m := transition(t, map[string]string{storyFile: story(oldLines...)}, map[string]string{storyFile: story(newLines...)}, recs, pending)

	if len(m.Records) != 0 {
		t.Errorf("records = %v, want none", m.Records)
	}
	want := []store.PendingItem{
		{Key: contentKey(storyFile, 1), Reason: "kr-changed", PrevKR: oldLines[1]},
		{Key: contentKey(storyFile, 3), Reason: "kr-changed", PrevKR: oldLines[3], PrevZH: "舊譯 3", Hint: "改用「您」"},
	}
	if !reflect.DeepEqual(m.Pending, want) {
		t.Errorf("pending = %+v, want %+v", m.Pending, want)
	}
}

func TestTransitionChangedTwiceKeepsTheLastTranslatedPair(t *testing.T) {
	// The KR of line 1 changed last week (A to B) and the unit was queued
	// with A and its translation, then changed again (B to C) before anyone
	// translated it. Line 3 changed last week too, but its item carries no
	// translation to keep.
	const kA, kB, kC = "안녕하세요 오늘은 날씨가 좋네요", "안녕하세요 오늘은 날씨가 나쁘네요", "안녕하세요 오늘은 날씨가 흐리네요"
	const mB, mC = "안녕하세요 내일은 비가 올 거예요", "안녕하세요 내일은 눈이 올 거예요"
	oldLines := []string{"첫째 줄", kB, "셋째 줄", mB, "다섯째 줄"}
	newLines := []string{"첫째 줄", kC, "셋째 줄", mC, "다섯째 줄"}
	pending := []store.PendingItem{
		{Key: contentKey(storyFile, 1), Reason: "kr-changed", PrevKR: kA, PrevZH: "A 的譯文", Hint: "改用「您」", Attempts: 1},
		{Key: contentKey(storyFile, 3), Reason: "kr-changed", PrevKR: "더 오래된 줄"},
	}

	m := transition(t, map[string]string{storyFile: story(oldLines...)}, map[string]string{storyFile: story(newLines...)}, nil, pending)

	want := []store.PendingItem{
		{Key: contentKey(storyFile, 1), Reason: "kr-changed", PrevKR: kA, PrevZH: "A 的譯文", Hint: "改用「您」"},
		{Key: contentKey(storyFile, 3), Reason: "kr-changed", PrevKR: mB},
	}
	if !reflect.DeepEqual(m.Pending, want) {
		t.Errorf("pending = %+v, want %+v", m.Pending, want)
	}
}

func TestTransitionChangedPrefersTheRecordOverAnOldPair(t *testing.T) {
	// A record exists (a requeued unit) next to an old pair: the record is
	// the translation the unit has now.
	const oldKR, newKR = "안녕하세요 오늘은 날씨가 좋네요", "안녕하세요 오늘은 날씨가 나쁘네요"
	oldLines := []string{"첫째 줄", oldKR, "셋째 줄"}
	newLines := []string{"첫째 줄", newKR, "셋째 줄"}
	recs := recordsOf(translated(contentKey(storyFile, 1), oldKR, "現行譯文"))
	pending := []store.PendingItem{{Key: contentKey(storyFile, 1), Reason: "requeue", PrevKR: "더 오래된 줄", PrevZH: "更舊的譯文"}}

	m := transition(t, map[string]string{storyFile: story(oldLines...)}, map[string]string{storyFile: story(newLines...)}, recs, pending)

	want := []store.PendingItem{{Key: contentKey(storyFile, 1), Reason: "kr-changed", PrevKR: oldKR, PrevZH: "現行譯文"}}
	if !reflect.DeepEqual(m.Pending, want) {
		t.Errorf("pending = %+v, want %+v", m.Pending, want)
	}
}

func TestTransitionFrozenKeepsRecord(t *testing.T) {
	bgm := func(first, second string) string {
		return fmt.Sprintf(`{"dataList":[{"id":1,"lyrics":%q},{"id":2,"lyrics":%q}]}`, first, second)
	}
	const oldKR, newKR = "가사 한 줄 입니다", "가사 한 줄 입니다 수정"
	key := units.Key(bgmFile, "/dataList/0/lyrics") // the pointer does not change
	recs := recordsOf(
		translated(key, oldKR, "歌詞一行"),
		translated(units.Key(bgmFile, "/dataList/1/lyrics"), "다른 가사", "另一句"),
	)

	m := transition(t, map[string]string{bgmFile: bgm(oldKR, "다른 가사")}, map[string]string{bgmFile: bgm(newKR, "다른 가사")}, recs, nil)

	got, ok := m.Records[key]
	if !ok {
		t.Fatalf("the record of the frozen line is gone: %v", m.Records)
	}
	if got.ZH != "歌詞一行" || got.KRSha != units.KRSha(oldKR) {
		t.Errorf("record = %+v, want the old zh and the old kr_sha", got)
	}
	mustEqual(t, "FrozenChanged", m.FrozenChanged, []string{key})
	if len(m.Pending) != 0 {
		t.Errorf("pending = %v, want none", m.Pending)
	}
	if m.Changes.Changed != 1 || m.Changes.Unchanged != 1 {
		t.Errorf("Changes = %+v, want 1 changed and 1 unchanged", m.Changes)
	}
}

func TestTransitionFrozenMovesTheRecordToTheNewKey(t *testing.T) {
	bgm := func(entries ...string) string { return `{"dataList":[` + strings.Join(entries, ",") + `]}` }
	// The file has no ids, so lines match by sequence; a line is added in
	// front and the next one is edited.
	oldDoc := bgm(`{"lyrics":"처음 가사입니다"}`, `{"lyrics":"둘째 가사 한 줄입니다"}`)
	newDoc := bgm(`{"lyrics":"새 가사"}`, `{"lyrics":"처음 가사입니다"}`, `{"lyrics":"둘째 가사 한 줄입니다 고침"}`)
	recs := recordsOf(
		translated(units.Key(bgmFile, "/dataList/0/lyrics"), "처음 가사입니다", "第一"),
		translated(units.Key(bgmFile, "/dataList/1/lyrics"), "둘째 가사 한 줄입니다", "第二"),
	)

	m := transition(t, map[string]string{bgmFile: oldDoc}, map[string]string{bgmFile: newDoc}, recs, nil)

	newKey := units.Key(bgmFile, "/dataList/2/lyrics")
	if got := m.Records[newKey]; got.Key != newKey || got.ZH != "第二" || got.KRSha != units.KRSha("둘째 가사 한 줄입니다") {
		t.Errorf("record at %s = %+v", newKey, got)
	}
	if got := m.Records[units.Key(bgmFile, "/dataList/1/lyrics")]; got.ZH != "第一" {
		t.Errorf("the unchanged line lost its record: %+v", got)
	}
	mustEqual(t, "FrozenChanged", m.FrozenChanged, []string{newKey})
	mustEqual(t, "pending keys", pendingKeys(m), []string{units.Key(bgmFile, "/dataList/0/lyrics")}) // the added line only
}

func TestTransitionFrozenChangedWithoutRecordIsQueued(t *testing.T) {
	bgm := func(s string) string { return fmt.Sprintf(`{"dataList":[{"id":1,"lyrics":%q}]}`, s) }
	m := transition(t, map[string]string{bgmFile: bgm("가사 한 줄 입니다")}, map[string]string{bgmFile: bgm("가사 한 줄 입니다 수정")}, nil, nil)

	if len(m.FrozenChanged) != 0 {
		t.Errorf("FrozenChanged = %v, want none: there was no translation to keep", m.FrozenChanged)
	}
	want := []store.PendingItem{{Key: units.Key(bgmFile, "/dataList/0/lyrics"), Reason: "kr-changed", PrevKR: "가사 한 줄 입니다"}}
	if !reflect.DeepEqual(m.Pending, want) {
		t.Errorf("pending = %+v, want %+v", m.Pending, want)
	}
}

func TestTransitionNewAndDeletedFiles(t *testing.T) {
	const goneFile, freshFile = "Gone.json", "Fresh.json"
	gone := `{"dataList":[{"id":1,"name":"사라진 이름"},{"id":2,"name":"또 사라진 이름"}]}`
	fresh := `{"dataList":[{"id":1,"name":"새 이름"},{"id":2,"name":"또 새 이름"}]}`
	recs := recordsOf(
		translated(units.Key(goneFile, "/dataList/0/name"), "사라진 이름", "消失的名字"),
		translated(units.Key(goneFile, "/dataList/1/name"), "또 사라진 이름", "又消失的名字"),
	)
	pending := []store.PendingItem{{Key: units.Key(goneFile, "/dataList/1/name"), Reason: "new"}}

	m := transition(t,
		map[string]string{goneFile: gone, storyFile: story("그대로")},
		map[string]string{freshFile: fresh, storyFile: story("그대로")},
		recs, pending)

	if len(m.Records) != 0 {
		t.Errorf("records = %v, want none", m.Records)
	}
	wantPending := []string{units.Key(freshFile, "/dataList/0/name"), units.Key(freshFile, "/dataList/1/name")}
	mustEqual(t, "pending keys", pendingKeys(m), wantPending)
	for _, p := range m.Pending {
		if p.Reason != "new" {
			t.Errorf("pending %+v: reason is not new", p)
		}
	}
	mustEqual(t, "FilesAdded", m.Changes.FilesAdded, []string{freshFile})
	mustEqual(t, "FilesDeleted", m.Changes.FilesDeleted, []string{goneFile})
	if len(m.Changes.FilesChanged) != 0 {
		t.Errorf("FilesChanged = %v: added and deleted files are not counted as changed", m.Changes.FilesChanged)
	}
	if m.Changes.New != 2 || m.Changes.Deleted != 2 || m.Changes.Unchanged != 1 {
		t.Errorf("Changes = %+v", m.Changes)
	}
	if len(m.Deleted) != 2 {
		t.Fatalf("Deleted = %+v, want 2", m.Deleted)
	}
	for i, d := range m.Deleted {
		wantKR, wantZH := []string{"사라진 이름", "또 사라진 이름"}[i], []string{"消失的名字", "又消失的名字"}[i]
		wantPointer := fmt.Sprintf("/dataList/%d/name", i)
		if d.File != goneFile || d.Pointer != wantPointer || d.Field != "name" || d.KR != wantKR || d.Record == nil || d.Record.ZH != wantZH {
			t.Errorf("Deleted[%d] = %+v", i, d)
		}
	}
}

func TestTransitionDeletedUnitWithoutRecord(t *testing.T) {
	m := transition(t,
		map[string]string{storyFile: story("그대로", "지워질 줄")},
		map[string]string{storyFile: story("그대로")},
		recordsOf(translated(contentKey(storyFile, 0), "그대로", "照舊")), nil)

	if len(m.Deleted) != 1 || m.Deleted[0].KR != "지워질 줄" || m.Deleted[0].Record != nil || m.Deleted[0].Pointer != "/dataList/1/content" {
		t.Errorf("Deleted = %+v, want one untranslated unit with no record at /dataList/1/content", m.Deleted)
	}
	mustEqual(t, "FilesChanged", m.Changes.FilesChanged, []string{storyFile})
}

// A deleted unit carries the pending item it had, as it was, so that the plan
// can hand a request on it to the unit that takes its place; no item is left
// at its key.
func TestTransitionDeletedUnitCarriesItsPendingItem(t *testing.T) {
	lease := &store.Lease{Batch: "b-7", Until: time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)}
	item := store.PendingItem{Key: contentKey(storyFile, 1), Reason: "requeue", PrevZH: "要刪的句子", Hint: "改用您", Attempts: 1, Lease: lease}
	m := transition(t,
		map[string]string{storyFile: story("그대로", "지워질 줄", "지워질 다른 줄")},
		map[string]string{storyFile: story("그대로")},
		recordsOf(translated(contentKey(storyFile, 1), "지워질 줄", "要刪的句子")),
		[]store.PendingItem{item})

	if len(m.Deleted) != 2 {
		t.Fatalf("Deleted = %+v, want 2", m.Deleted)
	}
	if got := m.Deleted[0].Pending; got == nil || !reflect.DeepEqual(*got, item) {
		t.Errorf("Deleted[0].Pending = %+v, want %+v", got, item)
	} else if got.Lease == lease {
		t.Errorf("Deleted[0].Pending shares the lease of the input item")
	}
	if got := m.Deleted[1].Pending; got != nil {
		t.Errorf("Deleted[1].Pending = %+v, want nil: the unit had no item", got)
	}
	if len(m.Pending) != 0 {
		t.Errorf("pending = %+v, want none", m.Pending)
	}
}

func TestTransitionMovesPendingItemsAndClearsLeases(t *testing.T) {
	oldLines := []string{"첫째 줄", "둘째 줄", "셋째 줄"}
	newLines := []string{"끼운 줄", "첫째 줄", "둘째 줄", "셋째 줄"}
	lease := &store.Lease{Batch: "b-7", Until: time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)}
	pending := []store.PendingItem{{
		Key: contentKey(storyFile, 1), Reason: "new", Attempts: 2, Hint: "다시", Flags: []string{store.FlagNeedsReview},
		Refs: map[string]string{"llc": "舊參考"}, Draft: "草稿", PrevKR: "pk", PrevZH: "pz", Lease: lease,
	}}

	m := transition(t, map[string]string{storyFile: story(oldLines...)}, map[string]string{storyFile: story(newLines...)}, nil, pending)

	var moved *store.PendingItem
	for i := range m.Pending {
		if m.Pending[i].Key == contentKey(storyFile, 2) {
			moved = &m.Pending[i]
		}
	}
	if moved == nil {
		t.Fatalf("no pending item at the new key: %+v", m.Pending)
	}
	want := store.PendingItem{
		Key: contentKey(storyFile, 2), Reason: "new", Attempts: 2, Hint: "다시", Flags: []string{store.FlagNeedsReview},
		Refs: map[string]string{"llc": "舊參考"}, Draft: "草稿", PrevKR: "pk", PrevZH: "pz",
	}
	if !reflect.DeepEqual(*moved, want) {
		t.Errorf("moved item = %+v, want %+v", *moved, want)
	}
	if moved.Lease != nil {
		t.Errorf("the lease was not cleared: %+v", moved.Lease)
	}
	if len(m.Pending) != 2 { // the moved item and the inserted line
		t.Errorf("pending keys = %v", pendingKeys(m))
	}
	if pending[0].Lease != lease || pending[0].Key != contentKey(storyFile, 1) {
		t.Errorf("the input item was changed: %+v", pending[0])
	}
}

func TestTransitionIDFieldAdded(t *testing.T) {
	oldDoc := `{"dataList":[{"id":1,"name":"이름 가","summary":"요약 가"},{"id":2,"name":"이름 나","summary":"요약 나"}]}`
	newDoc := `{"dataList":[{"id":1,"name":"이름 가","desc":"새 설명","summary":"요약 가"},{"id":2,"name":"이름 나","summary":"요약 나"}]}`
	var all []store.Record
	for _, k := range []struct{ ptr, kr string }{
		{"/dataList/0/name", "이름 가"}, {"/dataList/0/summary", "요약 가"},
		{"/dataList/1/name", "이름 나"}, {"/dataList/1/summary", "요약 나"},
	} {
		all = append(all, translated(units.Key(bufsFile, k.ptr), k.kr, "譯 "+k.kr))
	}

	m := transition(t, map[string]string{bufsFile: oldDoc}, map[string]string{bufsFile: newDoc}, recordsOf(all...), nil)

	mustEqual(t, "pending keys", pendingKeys(m), []string{units.Key(bufsFile, "/dataList/0/desc")})
	if m.Pending[0].Reason != "new" {
		t.Errorf("pending = %+v", m.Pending[0])
	}
	if len(m.Records) != 4 {
		t.Errorf("got %d records, want 4", len(m.Records))
	}
	if m.Changes.Unchanged != 4 || m.Changes.New != 1 || m.Changes.Changed != 0 || m.Changes.Deleted != 0 || m.Changes.Moved != 0 {
		t.Errorf("Changes = %+v", m.Changes)
	}
	if len(m.Changes.FilesShifted) != 0 {
		t.Errorf("FilesShifted = %v: adding a field moves no pointer", m.Changes.FilesShifted)
	}
}

func TestTransitionSameVersionChangesNothing(t *testing.T) {
	files := map[string]string{storyFile: story("첫째 줄", "둘째 줄"), bufsFile: `{"dataList":[{"id":1,"name":"이름"}]}`}
	recs := recordsOf(translated(contentKey(storyFile, 0), "첫째 줄", "一"), translated(units.Key(bufsFile, "/dataList/0/name"), "이름", "名"))

	m := transition(t, files, files, recs, nil)

	if !reflect.DeepEqual(m.Records, recs) {
		t.Errorf("records = %v, want %v", m.Records, recs)
	}
	mustEqual(t, "pending keys", pendingKeys(m), nil)
	if m.Changes.Moved != 0 || m.Changes.Changed != 0 || m.Changes.New != 0 || m.Changes.Deleted != 0 || m.Changes.Unchanged != 3 {
		t.Errorf("Changes = %+v", m.Changes)
	}
	if len(m.Changes.FilesChanged)+len(m.Changes.FilesShifted)+len(m.Changes.FilesAdded)+len(m.Changes.FilesDeleted) != 0 {
		t.Errorf("Changes = %+v, want no files", m.Changes)
	}
}

func TestTransitionDropsStateOfUnitsTheOldKRDoesNotHave(t *testing.T) {
	// A record and a pending item whose keys are not strings of the old KR
	// (a file that is in neither version, a pointer that does not exist)
	// cannot be placed in the new one.
	recs := recordsOf(
		translated(contentKey(storyFile, 0), "첫째 줄", "一"),
		translated(contentKey(storyFile, 9), "없는 줄", "無"),
		translated(units.Key("Nowhere.json", "/dataList/0/name"), "없는 파일", "無檔"),
	)
	pending := []store.PendingItem{{Key: contentKey(storyFile, 8), Reason: "new"}, {Key: units.Key("Nowhere.json", "/x"), Reason: "new"}}

	m := transition(t, map[string]string{storyFile: story("첫째 줄")}, map[string]string{storyFile: story("첫째 줄")}, recs, pending)

	if len(m.Records) != 1 || m.Records[contentKey(storyFile, 0)].ZH != "一" {
		t.Errorf("records = %v", m.Records)
	}
	if len(m.Pending) != 0 {
		t.Errorf("pending = %v", m.Pending)
	}
}

func TestTransitionDoesNotChangeItsInputs(t *testing.T) {
	oldLines := []string{"첫째 줄", "안녕하세요 오늘은 날씨가 좋네요", "셋째 줄", "지워질 줄"}
	newLines := []string{"끼운 줄", "첫째 줄", "안녕하세요 오늘은 날씨가 나쁘네요", "셋째 줄"}
	recs := map[string]store.Record{}
	for i, l := range oldLines {
		r := translated(contentKey(storyFile, i), l, fmt.Sprintf("舊譯 %d", i))
		r.Flags = []string{store.FlagNeedsReview}
		recs[r.Key] = r
	}
	pending := []store.PendingItem{
		{Key: contentKey(storyFile, 0), Reason: "new", Flags: []string{"a"}, Refs: map[string]string{"llc": "x"}, Lease: &store.Lease{Batch: "b", Until: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)}},
		{Key: contentKey(storyFile, 1), Reason: "requeue", Hint: "h"},
		// Deleted: carried in Moved.Deleted.
		{Key: contentKey(storyFile, 3), Reason: "requeue", Hint: "h", Flags: []string{"b"}, Refs: map[string]string{"llc": "y"}, Lease: &store.Lease{Batch: "c", Until: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)}},
	}
	snapshot := func() string {
		a, _ := json.Marshal(recs)
		b, _ := json.Marshal(pending)
		return string(a) + string(b)
	}
	before := snapshot()

	m := transition(t, map[string]string{storyFile: story(oldLines...)}, map[string]string{storyFile: story(newLines...)}, recs, pending)

	if snapshot() != before {
		t.Errorf("the inputs changed:\nbefore %s\nafter  %s", before, snapshot())
	}
	// Changing the result must not reach the inputs either.
	for k, r := range m.Records {
		if len(r.Flags) > 0 {
			r.Flags[0] = "changed"
		}
		m.Records[k] = r
	}
	for i := range m.Pending {
		if len(m.Pending[i].Flags) > 0 {
			m.Pending[i].Flags[0] = "changed"
		}
		for k := range m.Pending[i].Refs {
			m.Pending[i].Refs[k] = "changed"
		}
	}
	for _, d := range m.Deleted {
		if d.Record != nil && len(d.Record.Flags) > 0 {
			d.Record.Flags[0] = "changed"
		}
		if p := d.Pending; p != nil {
			if len(p.Flags) > 0 {
				p.Flags[0] = "changed"
			}
			for k := range p.Refs {
				p.Refs[k] = "changed"
			}
			if p.Lease != nil {
				p.Lease.Batch = "changed"
			}
		}
	}
	if len(m.Deleted) != 1 || m.Deleted[0].Pending == nil {
		t.Errorf("Deleted = %+v, want the deleted line with its pending item", m.Deleted)
	}
	if snapshot() != before {
		t.Errorf("the result shares memory with the inputs:\nbefore %s\nafter  %s", before, snapshot())
	}
}

func TestTransitionListsAreSorted(t *testing.T) {
	oldFiles := map[string]string{
		"Z.json": `{"dataList":[{"id":1,"name":"가"}]}`, "A.json": `{"dataList":[{"id":1,"name":"나"}]}`,
		"M.json": `{"dataList":[{"id":1,"name":"다"}]}`,
	}
	newFiles := map[string]string{
		"Z.json": `{"dataList":[{"id":1,"name":"가 고침"}]}`, "A.json": `{"dataList":[{"id":1,"name":"나 고침"}]}`,
		"Y.json": `{"dataList":[{"id":1,"name":"라"}]}`, "B.json": `{"dataList":[{"id":1,"name":"마"}]}`,
	}

	m := transition(t, oldFiles, newFiles, nil, nil)

	mustEqual(t, "FilesAdded", m.Changes.FilesAdded, []string{"B.json", "Y.json"})
	mustEqual(t, "FilesDeleted", m.Changes.FilesDeleted, []string{"M.json"})
	mustEqual(t, "FilesChanged", m.Changes.FilesChanged, []string{"A.json", "Z.json"})
	if !slices.IsSorted(pendingKeys(m)) {
		t.Errorf("pending is not sorted by key: %v", pendingKeys(m))
	}
}

func TestTransitionReportsUnreadableFiles(t *testing.T) {
	newKR := openKR(t, map[string]string{storyFile: story("첫째 줄")})
	oldKR := openKR(t, map[string]string{storyFile: "not json"})

	_, err := Transition(oldKR, newKR, nil, nil, frozen)

	if err == nil || !strings.Contains(err.Error(), storyFile) {
		t.Errorf("err = %v, want one naming %s", err, storyFile)
	}
}
