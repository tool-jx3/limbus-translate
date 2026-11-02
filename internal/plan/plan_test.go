package plan

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// The rules of the plan tests: three families (story, skill and the catch-all
// other), one approved term with a forbidden rendering, and id, model and key
// as identifier fields.
const (
	testRouting = `families:
  - id: story
    match: ["StoryData/*"]
    guide: guides/families.md
  - id: skill
    match: ["Skills*"]
    guide: guides/families.md
  - id: other
    match: []
    guide: guides/families.md
priority: [story, skill, other]
`
	testGlossary = `- kr: 마법
  zh: 魔法
  forbidden: [法術]
  status: approved
`
	testFieldRules = "identifier_fields: [id, model, key]\n"

	testRulesCommit = "rules1"
	tag1            = "raw-20261029"
	tag2            = "raw-20261105"
)

// snapshotDates are the created_at of the snapshots planEnv.snapshot takes, in
// order: the tags are raw-20261029, raw-20261105, raw-20261112.
var snapshotDates = []string{"2026-10-29 02:55:23", "2026-11-05 02:55:23", "2026-11-12 02:55:23"}

// planEnv is a repo the plan works in: the rules at root, a raw worktree that
// takes snapshots, and the tw store.
type planEnv struct {
	t      *testing.T
	root   string
	rawWT  string
	twRoot string
	tags   []string
}

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newPlanEnv(t *testing.T) *planEnv {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, rules.RoutingFile, testRouting)
	writeFile(t, root, rules.GlossaryFile, testGlossary)
	writeFile(t, root, "rules/fields.manual.yaml", testFieldRules)
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(rules.CharactersDir)), 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn(t, root, "init", "-q", "-b", "main")
	gitIn(t, root, "config", "user.name", "Test")
	gitIn(t, root, "config", "user.email", "test@example.com")
	gitIn(t, root, "config", "commit.gpgsign", "false")
	gitIn(t, root, "config", "tag.gpgsign", "false")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "rules")
	e := &planEnv{t: t, root: root, rawWT: filepath.Join(root, ".work", "raw"), twRoot: filepath.Join(root, ".work", "tw")}
	if err := gitx.EnsureWorktree(root, "raw", e.rawWT); err != nil {
		t.Fatal(err)
	}
	return e
}

// snapshot commits kr (file name to JSON text, such as "StoryData/S1.json")
// as the next snapshot, with EN and JP packs that share no file with it, and
// returns its tag.
func (e *planEnv) snapshot(kr map[string]string) string {
	e.t.Helper()
	return e.snapshotLangs(kr, nil, nil)
}

// snapshotLangs is snapshot with EN and JP files as well.
func (e *planEnv) snapshotLangs(kr, en, jp map[string]string) string {
	e.t.Helper()
	dir := e.t.TempDir()
	packs := map[string]string{}
	for _, l := range []struct {
		lang, prefix string
		files        map[string]string
	}{
		{"kr", "KR_", kr},
		{"en", "EN_", map[string]string{"Empty.json": "{}"}},
		{"jp", "JP_", map[string]string{"Empty.json": "{}"}},
	} {
		files := l.files
		switch l.lang {
		case "en":
			for f, text := range en {
				files[f] = text
			}
		case "jp":
			for f, text := range jp {
				files[f] = text
			}
		}
		for file, text := range files {
			d, name := filepath.Split(filepath.FromSlash(file))
			writeFile(e.t, filepath.Join(dir, l.lang), filepath.ToSlash(filepath.Join(d, l.prefix+name)), text)
		}
		packs[l.lang] = filepath.Join(dir, l.lang)
	}
	n := len(e.tags)
	tag, _, err := raw.Snapshot(e.rawWT, packs, raw.Source{Token: fmt.Sprintf("token%d", n+1), CreatedAt: snapshotDates[n]})
	if err != nil {
		e.t.Fatal(err)
	}
	e.tags = append(e.tags, tag)
	return tag
}

func (e *planEnv) store() *store.Store { return store.Open(e.twRoot) }

func (e *planEnv) inputs(buildRawTag string) Inputs {
	e.t.Helper()
	set, err := pack.OpenRaw(e.rawWT)
	if err != nil {
		e.t.Fatal(err)
	}
	r, err := rules.Load(e.root)
	if err != nil {
		e.t.Fatal(err)
	}
	fr, err := fields.Load(e.root)
	if err != nil {
		e.t.Fatal(err)
	}
	return Inputs{
		Root: e.root, RawWorktree: e.rawWT, TWRoot: e.twRoot, RulesCommit: testRulesCommit, BuildRawTag: buildRawTag,
		Src: set, Frozen: []string{"BgmLyrics/**"}, Fields: fr, Rules: r, TmpDir: e.t.TempDir(),
	}
}

// run plans with buildRawTag as the raw tag of the last build.
func (e *planEnv) run(buildRawTag string) *Summary {
	e.t.Helper()
	sum, err := Run(e.inputs(buildRawTag))
	if err != nil {
		e.t.Fatal(err)
	}
	return sum
}

func (e *planEnv) seed(recs []store.Record, pending []store.PendingItem) {
	e.t.Helper()
	if err := e.store().SaveRecords(recordsOf(recs...)); err != nil {
		e.t.Fatal(err)
	}
	if err := e.store().SavePending(pending); err != nil {
		e.t.Fatal(err)
	}
}

func (e *planEnv) records() map[string]store.Record {
	e.t.Helper()
	recs, err := e.store().LoadRecords()
	if err != nil {
		e.t.Fatal(err)
	}
	return recs
}

func (e *planEnv) pending() map[string]store.PendingItem {
	e.t.Helper()
	items, err := e.store().LoadPending()
	if err != nil {
		e.t.Fatal(err)
	}
	out := map[string]store.PendingItem{}
	for _, it := range items {
		out[it.Key] = it
	}
	return out
}

func (e *planEnv) applied() store.Applied {
	e.t.Helper()
	a, err := e.store().LoadApplied()
	if err != nil {
		e.t.Fatal(err)
	}
	return a
}

func (e *planEnv) rule(rel, content string) { writeFile(e.t, e.root, rel, content) }

type obj = map[string]any

// dataList is a file whose dataList holds entries.
func dataList(entries ...obj) string {
	data, err := json.Marshal(obj{"dataList": entries})
	if err != nil {
		panic(err)
	}
	return string(data)
}

func at(file, pointer string) string { return units.Key(file, pointer) }

func descKey(file string, i int) string { return at(file, fmt.Sprintf("/dataList/%d/desc", i)) }

// copied is the copy record of a unit whose KR text is kr.
func copied(key, kr string) store.Record {
	return store.Record{Key: key, KRSha: units.KRSha(kr), ZH: kr, Origin: store.OriginCopy, Since: tag1, Rules: "abc123"}
}

// adjustment is one entry of an adjustment file.
func adjustment(key, kr, zh string) string {
	return fmt.Sprintf("- key: %s\n  raw: %s\n  kr_sha: %s\n  zh: %s\n  reason: 更好\n  rule: one-off\n", key, tag1, units.KRSha(kr), zh)
}

func TestPlanWithoutTransitionChangesNothing(t *testing.T) {
	e := newPlanEnv(t)
	const s1 = "StoryData/S1.json"
	e.snapshot(map[string]string{
		s1:              dataList(obj{"id": 0, "model": "이상", "content": "오늘은 날씨가 좋다."}, obj{"id": 1, "model": "파우스트", "content": "그렇군요."}),
		"Skills_A.json": dataList(obj{"id": 1, "name": "베기", "desc": "적에게 피해를 준다"}),
	})
	recs := []store.Record{
		copied(at(s1, "/dataList/0/model"), "이상"),
		copied(at(s1, "/dataList/1/model"), "파우스트"),
		translated(at(s1, "/dataList/0/content"), "오늘은 날씨가 좋다.", "今天天氣真好。"),
		translated(at("Skills_A.json", "/dataList/0/name"), "베기", "斬擊"),
	}
	pending := []store.PendingItem{
		{Key: at(s1, "/dataList/1/content"), Reason: ReasonNew},
		{Key: descKey("Skills_A.json", 0), Reason: ReasonKRChanged, PrevKR: "피해", PrevZH: "傷害"},
	}
	e.seed(recs, pending)
	beforeRecs, beforePending := e.records(), e.pending()

	sum := e.run("")

	if got := e.records(); !reflect.DeepEqual(got, beforeRecs) {
		t.Errorf("records changed:\n got %v\nwant %v", got, beforeRecs)
	}
	if got := e.pending(); !reflect.DeepEqual(got, beforePending) {
		t.Errorf("pending changed:\n got %v\nwant %v", got, beforePending)
	}
	if sum.RawTag != tag1 || sum.PrevRawTag != tag1 || sum.RulesCommit != testRulesCommit {
		t.Errorf("tags = %q from %q under %q, want %s from itself under %s", sum.RawTag, sum.PrevRawTag, sum.RulesCommit, tag1, testRulesCommit)
	}
	if !reflect.DeepEqual(sum.Changes, Changes{}) {
		t.Errorf("Changes = %+v, want all zero", sum.Changes)
	}
	if sum.Copy != 0 || sum.Reused != 0 || len(sum.RequeueApplied) != 0 || len(sum.RewriteApplied) != 0 {
		t.Errorf("summary = %+v, want nothing copied, reused or applied", sum)
	}
	if sum.Pending != 2 {
		t.Errorf("Pending = %d, want 2", sum.Pending)
	}
}

// A second plan of the same snapshot changes nothing: every requeue and
// rewrite file is applied once, and none of them touches a unit whose source
// steps 1 to 3 decide (a copy, an adjusted unit), which the next plan would
// only set back. Reuse runs after the rule files, so it sees what they fixed.
func TestPlanTwiceIsIdempotent(t *testing.T) {
	e := newPlanEnv(t)
	const (
		s1     = "StoryData/S1.json"
		skills = "Skills_A.json"
		magic  = "마법으로 적을 공격합니다"
	)
	adjusted, copyKey := descKey(skills, 1), at(skills, "/dataList/2/key")
	skillsV1 := dataList(obj{"id": 1, "desc": "피해량이 2배 증가한다"}, obj{"id": 2, "desc": "적에게 2배 피해를 준다"}, obj{"id": 3, "key": "2_SLASH"})
	e.snapshot(map[string]string{
		s1: dataList(
			obj{"id": 0, "model": "이상", "content": "첫 번째 줄입니다."},
			obj{"id": 1, "model": "파우스트", "content": "두 번째 줄입니다."},
			obj{"id": 2, "model": "이상", "content": "세 번째 줄입니다."}),
		skills:          skillsV1,
		"Skills_M.json": dataList(obj{"id": 1, "desc": magic}),
	})
	e.seed([]store.Record{
		copied(at(s1, "/dataList/0/model"), "이상"),
		copied(at(s1, "/dataList/1/model"), "파우스트"),
		copied(at(s1, "/dataList/2/model"), "이상"),
		translated(at(s1, "/dataList/0/content"), "첫 번째 줄입니다.", "這是第一行。"),
		translated(at(s1, "/dataList/1/content"), "두 번째 줄입니다.", "這是第二行。"),
		translated(at(s1, "/dataList/2/content"), "세 번째 줄입니다.", "這是第三行。"),
		translated(descKey(skills, 0), "피해량이 2배 증가한다", "傷害增加2倍"),
		translated(adjusted, "적에게 2배 피해를 준다", "對敵人造成兩倍傷害"),
		copied(copyKey, "2_SLASH"),
		translated(descKey("Skills_M.json", 0), magic, "用法術攻擊敵人"), // 마법 is 魔法, never 法術
	}, nil)
	e.rule("rules/adjustments/2026-11-01-wording.yaml", adjustment(adjusted, "적에게 2배 피해를 준다", "對敵人造成2倍傷害。"))
	e.run("") // the state belongs to the first snapshot

	e.snapshot(map[string]string{
		s1: dataList(
			obj{"id": 0, "model": "이상", "content": "첫 번째 줄입니다."},
			obj{"id": 1, "model": "그레고르", "content": "새로 들어온 줄입니다."},
			obj{"id": 2, "model": "파우스트", "content": "두 번째 줄입니다."},
			obj{"id": 3, "model": "이상", "content": "세 번째 줄을 고쳤습니다."}),
		skills:          skillsV1,
		"Skills_M.json": dataList(obj{"id": 1, "desc": magic}),
		"Skills_N.json": dataList(obj{"id": 1, "desc": magic}), // the same text, in a new file
	})
	e.rule("rules/requeue/2026-11-05-honorific.yaml", "reason: 稱呼改用您\nkeys: [\"StoryData/S1.json|/dataList/2/content\"]\n")
	// Every origin but copy by default, human too: the adjusted unit is in it.
	e.rule("rules/requeue/2026-11-06-skills.yaml", "reason: 全部重翻\nfiles: [\"Skills_A*\"]\n")
	e.rule("rules/rewrite/2026-11-05-space.yaml", "reason: 加空格\nfiles: [\"Skills*\"]\npattern: '(2)(倍)'\nreplace: '$1 $2'\n")
	// It names the origins of the adjusted unit and of the copy.
	e.rule("rules/rewrite/2026-11-06-held.yaml", "reason: 加空格\nfiles: [\"Skills*\"]\norigins: [human, copy]\npattern: '(2)(倍|_)'\nreplace: '$1 $2'\n")
	// It fixes the translation that Skills_N.json can then reuse.
	e.rule("rules/rewrite/2026-11-07-magic.yaml", "reason: 마법 是魔法\nfiles: [\"Skills_M*\"]\npattern: '法術'\nreplace: '魔法'\n")

	first := e.run("")
	recs1, pending1, applied1 := e.records(), e.pending(), e.applied()
	appliedJSON1, err := os.ReadFile(filepath.Join(e.twRoot, ".meta", "applied.json"))
	if err != nil {
		t.Fatal(err)
	}

	if first.PrevRawTag != tag1 || first.RawTag != tag2 {
		t.Errorf("first run: %s from %s, want %s from %s", first.RawTag, first.PrevRawTag, tag2, tag1)
	}
	if want := []Applied{{ID: "2026-11-05-honorific", Units: 1}, {ID: "2026-11-06-skills", Units: 1}}; !reflect.DeepEqual(first.RequeueApplied, want) {
		t.Errorf("first RequeueApplied = %+v, want %+v", first.RequeueApplied, want)
	}
	if want := []Applied{{ID: "2026-11-05-space", Units: 1}, {ID: "2026-11-06-held", Units: 0}, {ID: "2026-11-07-magic", Units: 1}}; !reflect.DeepEqual(first.RewriteApplied, want) {
		t.Errorf("first RewriteApplied = %+v, want %+v", first.RewriteApplied, want)
	}
	if got := recs1[descKey(skills, 0)].ZH; got != "傷害增加2 倍" {
		t.Errorf("rewritten zh = %q, want 傷害增加2 倍", got)
	}
	if it := pending1[at(s1, "/dataList/2/content")]; it.Reason != "requeue" || it.Hint != "稱呼改用您" || it.PrevZH != "這是第二行。" {
		t.Errorf("requeued item = %+v, want reason requeue with the hint and the old translation", it)
	}
	if it := pending1[descKey(skills, 0)]; it.Reason != "requeue" || it.Hint != "全部重翻" || it.PrevZH != "傷害增加2 倍" {
		t.Errorf("requeued skill = %+v, want reason requeue with the rewritten text as the old translation", it)
	}
	if got := recs1[adjusted]; got.Origin != store.OriginHuman || got.ZH != "對敵人造成2倍傷害。" {
		t.Errorf("adjusted unit = %+v, want the adjustment's text, not rewritten", got)
	}
	if got := recs1[copyKey]; got.Origin != store.OriginCopy || got.ZH != "2_SLASH" {
		t.Errorf("copied unit = %+v, want the KR text, not rewritten", got)
	}
	for _, key := range []string{adjusted, copyKey} {
		if it, queued := pending1[key]; queued {
			t.Errorf("%s was requeued (%+v), want it left to its source", key, it)
		}
	}
	reusedKey := descKey("Skills_N.json", 0)
	if got := recs1[reusedKey]; got.ZH != "用魔法攻擊敵人" || got.Batch != "reuse:"+descKey("Skills_M.json", 0) || first.Reused != 1 {
		t.Errorf("Skills_N = %+v (reused %d), want the rewritten translation reused", got, first.Reused)
	}
	wantApplied := store.Applied{
		Requeue: []string{"2026-11-05-honorific", "2026-11-06-skills"},
		Rewrite: []string{"2026-11-05-space", "2026-11-06-held", "2026-11-07-magic"},
	}
	if !reflect.DeepEqual(applied1, wantApplied) {
		t.Errorf("applied = %+v, want %+v", applied1, wantApplied)
	}

	second := e.run("")

	if got := e.records(); !reflect.DeepEqual(got, recs1) {
		t.Errorf("second run changed the records:\n got %v\nwant %v", got, recs1)
	}
	if got := e.pending(); !reflect.DeepEqual(got, pending1) {
		t.Errorf("second run changed the pending items:\n got %v\nwant %v", got, pending1)
	}
	appliedJSON2, err := os.ReadFile(filepath.Join(e.twRoot, ".meta", "applied.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(appliedJSON2) != string(appliedJSON1) {
		t.Errorf("second run changed applied.json:\n%s\nwant\n%s", appliedJSON2, appliedJSON1)
	}
	if len(second.RequeueApplied) != 0 || len(second.RewriteApplied) != 0 || second.Reused != 0 || second.Copy != 0 {
		t.Errorf("second run: applied %+v and %+v, reused %d, copied %d; want nothing again", second.RequeueApplied, second.RewriteApplied, second.Reused, second.Copy)
	}
}

// A plan of a snapshot that was planned already keeps what that snapshot
// brought, for the report and the review issue of its raw tag; what the run
// did itself is counted afresh (R15).
func TestPlanRerunKeepsWhatTheSnapshotBrought(t *testing.T) {
	e := newPlanEnv(t)
	const bgm = "BgmLyrics/L.json"
	e.snapshot(map[string]string{
		"StoryData/S1.json": dataList(obj{"id": 0, "model": "이상", "content": "줄 하나"}),
		bgm:                 dataList(obj{"id": 1, "content": "노래 가사"}),
	})
	e.seed([]store.Record{
		copied(at("StoryData/S1.json", "/dataList/0/model"), "이상"),
		translated(at("StoryData/S1.json", "/dataList/0/content"), "줄 하나", "第一行"),
		translated(at(bgm, "/dataList/0/content"), "노래 가사", "歌詞"),
	}, nil)
	e.snapshot(map[string]string{
		"StoryData/S1.json": dataList(obj{"id": 0, "model": "이상", "content": "줄 하나"}, obj{"id": 1, "model": "새인물", "content": "줄 둘"}),
		bgm:                 dataList(obj{"id": 1, "content": "바뀐 노래 가사"}),
		"Misc.json":         dataList(obj{"id": 1, "desc": "새 파일"}),
	})
	e.rule("rules/requeue/2026-11-05-story.yaml", "reason: 重翻\nfiles: [\"StoryData/**\"]\n")

	first := e.run(tag1)
	if first.PrevRawTag != tag1 || first.Changes.New == 0 || len(first.FrozenChanged) != 1 || len(first.NewSpeakers) != 1 ||
		len(first.NewFilesDefaultFamily) != 1 || len(first.RequeueApplied) != 1 {
		t.Fatalf("first run = %+v, want a transition with a frozen change, a new speaker, a new file and a request applied", first)
	}

	second := e.run(tag1)

	if second.RawTag != tag2 || second.PrevRawTag != first.PrevRawTag || !reflect.DeepEqual(second.Changes, first.Changes) ||
		!reflect.DeepEqual(second.FrozenChanged, first.FrozenChanged) || !reflect.DeepEqual(second.NewSpeakers, first.NewSpeakers) ||
		!reflect.DeepEqual(second.NewFilesDefaultFamily, first.NewFilesDefaultFamily) {
		t.Errorf("second run = %+v\nwant what the snapshot brought kept from %+v", second, first)
	}
	if len(second.RequeueApplied) != 0 || len(second.RewriteApplied) != 0 || second.Copy != 0 {
		t.Errorf("second run applied %+v and %+v and copied %d, want nothing", second.RequeueApplied, second.RewriteApplied, second.Copy)
	}
	if second.Pending != first.Pending {
		t.Errorf("second run Pending = %d, want %d: nothing changed", second.Pending, first.Pending)
	}
}

// A request for a unit that is queued already, and that same-text reuse can
// fill, still reaches the translator: the unit takes the translation and is
// sent back with it as the previous one and the request's hint. A record a
// request sends back serves no reuse. Neither changes on the next plan.
func TestPlanRequestFilesAndReuse(t *testing.T) {
	e := newPlanEnv(t)
	const (
		oldKR, newKR = "적을 공격합니다 아주 길게 옛날", "적을 공격합니다 아주 길게 새것"
		guard        = "방어 자세를 취하고 반격합니다"
	)
	a, b, d, newE := descKey("Skills_A.json", 0), descKey("Skills_B.json", 0), descKey("Skills_D.json", 0), descKey("Skills_E.json", 0)
	e.snapshot(map[string]string{
		"Skills_A.json": dataList(obj{"id": 1, "desc": newKR}),
		"Skills_B.json": dataList(obj{"id": 1, "desc": oldKR}),
		"Skills_D.json": dataList(obj{"id": 1, "desc": guard}),
	})
	e.seed([]store.Record{
		translated(a, newKR, "攻擊敵人，非常長，新"),
		translated(b, oldKR, "攻擊敵人，非常長，舊"),
		translated(d, guard, "擺出防禦姿勢並反擊"),
	}, nil)
	e.snapshot(map[string]string{
		"Skills_A.json": dataList(obj{"id": 1, "desc": newKR}),
		"Skills_B.json": dataList(obj{"id": 1, "desc": newKR}), // now A's text: kr-changed, and A has it
		"Skills_D.json": dataList(obj{"id": 1, "desc": guard}),
		"Skills_E.json": dataList(obj{"id": 1, "desc": guard}), // D's text, in a new file
	})
	e.rule("rules/requeue/2026-11-05-b.yaml", "reason: 用您\nkeys: [\""+b+"\"]\n")
	e.rule("rules/requeue/2026-11-05-d.yaml", "reason: 重翻\nfiles: [\"Skills_D*\"]\n")

	first := e.run(tag1)
	recs1, pending1 := e.records(), e.pending()

	if it, queued := pending1[b]; !queued || it.Reason != "requeue" || it.Hint != "用您" || it.PrevZH != "攻擊敵人，非常長，新" {
		t.Errorf("B pending = %+v (queued %v), want it requeued with the hint and the reused text as the old translation", it, queued)
	}
	if got := recs1[b]; got.ZH != "攻擊敵人，非常長，新" || got.Batch != "reuse:"+a {
		t.Errorf("B record = %+v, want A's translation until B is translated again", got)
	}
	if it := pending1[newE]; it.Reason != ReasonNew {
		t.Errorf("E pending = %+v, want it new: the translation of D is sent back", it)
	}
	if got, has := recs1[newE]; has {
		t.Errorf("E has the record %+v, want none", got)
	}
	if want := []Applied{{ID: "2026-11-05-b", Units: 1}, {ID: "2026-11-05-d", Units: 1}}; !reflect.DeepEqual(first.RequeueApplied, want) || first.Reused != 1 {
		t.Errorf("RequeueApplied = %+v, Reused = %d; want %+v and 1", first.RequeueApplied, first.Reused, want)
	}

	second := e.run("")

	if got := e.records(); !reflect.DeepEqual(got, recs1) {
		t.Errorf("second run changed the records:\n got %v\nwant %v", got, recs1)
	}
	if got := e.pending(); !reflect.DeepEqual(got, pending1) {
		t.Errorf("second run changed the pending items:\n got %v\nwant %v", got, pending1)
	}
	if len(second.RequeueApplied) != 0 || second.Reused != 0 {
		t.Errorf("second run applied %+v and reused %d, want nothing", second.RequeueApplied, second.Reused)
	}
}

func TestPlanCopiesUnitsTheFieldRulesSkip(t *testing.T) {
	e := newPlanEnv(t)
	e.snapshot(map[string]string{"Skills_A.json": dataList(obj{"id": 1, "name": "베기"})})
	e.seed([]store.Record{translated(at("Skills_A.json", "/dataList/0/name"), "베기", "斬擊")}, nil)
	e.snapshot(map[string]string{"Skills_A.json": dataList(obj{"id": 1, "name": "베기", "key": "SLASH_1"})})

	sum := e.run(tag1)

	key := at("Skills_A.json", "/dataList/0/key")
	rec, ok := e.records()[key]
	want := store.Record{Key: key, KRSha: units.KRSha("SLASH_1"), ZH: "SLASH_1", Origin: store.OriginCopy, Since: tag2, Rules: testRulesCommit}
	if !ok || !reflect.DeepEqual(rec, want) {
		t.Errorf("record of the new key field = %+v, want %+v", rec, want)
	}
	if _, queued := e.pending()[key]; queued {
		t.Errorf("%s is pending, want it copied", key)
	}
	if sum.Copy != 1 || sum.Changes.New != 1 {
		t.Errorf("Copy = %d, New = %d, want 1 and 1", sum.Copy, sum.Changes.New)
	}
}

func TestPlanAdjustmentBeatsTheRecord(t *testing.T) {
	e := newPlanEnv(t)
	const kr = "적에게 피해를 준다"
	key := descKey("Skills_A.json", 0)
	e.snapshot(map[string]string{"Skills_A.json": dataList(obj{"id": 1, "desc": kr})})
	e.seed([]store.Record{translated(key, kr, "對敵人造成傷害")},
		[]store.PendingItem{{Key: key, Reason: "requeue", PrevZH: "對敵人造成傷害", Hint: "加句號"}})
	e.rule("rules/adjustments/2026-11-01-wording.yaml", adjustment(key, kr, "對敵人造成傷害。"))

	sum := e.run("")

	want := store.Record{Key: key, KRSha: units.KRSha(kr), ZH: "對敵人造成傷害。", Origin: store.OriginHuman, Since: tag1, Rules: testRulesCommit, Batch: "adjustment:2026-11-01-wording.yaml"}
	if got := e.records()[key]; !reflect.DeepEqual(got, want) {
		t.Errorf("record = %+v, want %+v", got, want)
	}
	if _, queued := e.pending()[key]; queued {
		t.Errorf("%s is still pending, want the adjustment to settle it", key)
	}
	if sum.Adjustments.Applied != 1 || len(sum.Adjustments.Moved) != 0 || len(sum.Adjustments.Expired) != 0 {
		t.Errorf("Adjustments = %+v, want one applied", sum.Adjustments)
	}
}

func TestPlanQueuesAnExpiredAdjustmentOnce(t *testing.T) {
	e := newPlanEnv(t)
	const oldKR, newKR = "적에게 피해를 준다", "적에게 큰 피해를 준다"
	key := descKey("Skills_A.json", 0)
	e.snapshot(map[string]string{"Skills_A.json": dataList(obj{"id": 1, "desc": oldKR})})
	e.seed([]store.Record{translated(key, oldKR, "對敵人造成傷害")}, nil)
	e.rule("rules/adjustments/2026-11-01-wording.yaml", adjustment(key, oldKR, "對敵人造成傷害。"))
	if sum := e.run(""); sum.Adjustments.Applied != 1 {
		t.Fatalf("first run: Adjustments = %+v, want one applied", sum.Adjustments)
	}

	e.snapshot(map[string]string{"Skills_A.json": dataList(obj{"id": 1, "desc": newKR})})
	sum := e.run("")

	it, queued := e.pending()[key]
	if !queued || it.Reason != "adjustment-expired" || it.Refs["adjustment"] != "對敵人造成傷害。" {
		t.Errorf("pending item = %+v (queued %v), want adjustment-expired with the adjustment as a reference", it, queued)
	}
	if want := []string{key}; !reflect.DeepEqual(sum.Adjustments.Expired, want) {
		t.Errorf("Expired = %q, want %q", sum.Adjustments.Expired, want)
	}

	// The unit is translated again (as batch submit does it).
	recs := e.records()
	recs[key] = store.Record{Key: key, KRSha: units.KRSha(newKR), ZH: "對敵人造成重大傷害。", Origin: store.OriginClaude, Since: tag2, Model: "m1", Batch: "b-2"}
	if err := e.store().SaveRecords(recs); err != nil {
		t.Fatal(err)
	}
	if err := e.store().SavePending(nil); err != nil {
		t.Fatal(err)
	}
	sum = e.run("")

	if it, queued := e.pending()[key]; queued {
		t.Errorf("after the translation the unit is queued again: %+v", it)
	}
	if want := []string{key}; !reflect.DeepEqual(sum.Adjustments.Expired, want) {
		t.Errorf("Expired after the translation = %q, want %q", sum.Adjustments.Expired, want)
	}
	if got := e.records()[key]; got.Origin != store.OriginClaude {
		t.Errorf("record = %+v, want the new translation kept", got)
	}
}

// The key of an adjustment names a unit of the snapshot it was made for. A
// line inserted before it moves that unit: when the unit's KR changes as well,
// the adjustment expires, and its unit, at its new key, is the one queued with
// it as a reference, not the unit its key names now (here the inserted line).
// A second plan of the snapshot leaves it so.
func TestPlanQueuesAnExpiredAdjustmentAtTheUnitsNewKey(t *testing.T) {
	e := newPlanEnv(t)
	const (
		story    = "StoryData/S1.json"
		first    = "첫 번째 줄은 그대로입니다"
		oldKR    = "두 번째 줄은 곧 바뀔 예정입니다"
		newKR    = "두 번째 줄은 이미 바뀌었습니다"
		inserted = "새로 끼워 넣은 줄입니다"
	)
	content := func(i int) string { return at(story, fmt.Sprintf("/dataList/%d/content", i)) }
	e.snapshot(map[string]string{story: dataList(obj{"id": 1, "content": first}, obj{"id": 2, "content": oldKR})})
	e.seed([]store.Record{translated(content(0), first, "第一行不變"), translated(content(1), oldKR, "第二行即將改變")}, nil)
	// Made for the first snapshot and committed, but not planned before the
	// next snapshot.
	e.rule("rules/adjustments/2026-11-01-wording.yaml", adjustment(content(1), oldKR, "第二行就要改了。"))
	e.snapshot(map[string]string{story: dataList(obj{"id": 1, "content": first}, obj{"id": 2, "content": inserted}, obj{"id": 3, "content": newKR})})

	sum := e.run(tag1)

	want := map[string]store.PendingItem{
		content(1): {Key: content(1), Reason: ReasonNew},
		content(2): {Key: content(2), Reason: ReasonAdjustmentExpired, PrevKR: oldKR, PrevZH: "第二行即將改變", Refs: map[string]string{RefAdjustment: "第二行就要改了。"}},
	}
	if got := e.pending(); !reflect.DeepEqual(got, want) {
		t.Errorf("pending = %v\nwant %v", got, want)
	}
	if want := []string{content(1)}; !reflect.DeepEqual(sum.Adjustments.Expired, want) {
		t.Errorf("Expired = %q, want %q (the adjustment's own key)", sum.Adjustments.Expired, want)
	}

	e.run(tag1)
	if got := e.pending(); !reflect.DeepEqual(got, want) {
		t.Errorf("after a second plan pending = %v\nwant %v", got, want)
	}
}

// The summary says which unit an expired adjustment is about when that is not
// the unit its key names (here a line inserted before it moved the unit), so
// that lct adjust tidy can look at the right unit. An adjustment whose unit
// kept its key has no entry.
func TestPlanReportsTheUnitOfAnExpiredAdjustment(t *testing.T) {
	e := newPlanEnv(t)
	const (
		story    = "StoryData/S1.json"
		first    = "첫 번째 줄은 그대로입니다"
		oldKR    = "두 번째 줄은 곧 바뀔 예정입니다"
		newKR    = "두 번째 줄은 이미 바뀌었습니다"
		inserted = "새로 끼워 넣은 줄입니다"
	)
	content := func(i int) string { return at(story, fmt.Sprintf("/dataList/%d/content", i)) }
	e.snapshot(map[string]string{story: dataList(obj{"id": 1, "content": first}, obj{"id": 2, "content": oldKR})})
	e.seed([]store.Record{translated(content(0), first, "第一行不變"), translated(content(1), oldKR, "第二行即將改變")}, nil)
	e.rule("rules/adjustments/2026-11-01-wording.yaml", adjustment(content(1), oldKR, "第二行就要改了。"))
	e.snapshot(map[string]string{story: dataList(obj{"id": 1, "content": first}, obj{"id": 2, "content": inserted}, obj{"id": 3, "content": newKR})})

	sum := e.run(tag1)

	if want := map[string]string{content(1): content(2)}; !reflect.DeepEqual(sum.Adjustments.ExpiredUnits, want) {
		t.Errorf("ExpiredUnits = %v, want %v", sum.Adjustments.ExpiredUnits, want)
	}
	data, err := json.Marshal(sum.Adjustments)
	if err != nil || !strings.Contains(string(data), `"expired_units":{`) {
		t.Errorf("JSON = %s, %v, want expired_units", data, err)
	}
	// A second plan of the snapshot cannot follow the unit through the
	// transition again; it keeps what the first plan found.
	if again := e.run(tag1); !reflect.DeepEqual(again.Adjustments, sum.Adjustments) {
		t.Errorf("second plan: Adjustments = %+v, want %+v", again.Adjustments, sum.Adjustments)
	}

	// An adjustment whose unit kept its key has no entry, and the field is
	// left out of the JSON.
	e2 := newPlanEnv(t)
	key := descKey("Skills_A.json", 0)
	e2.snapshot(map[string]string{"Skills_A.json": dataList(obj{"id": 1, "desc": "적에게 피해를 준다"})})
	e2.seed([]store.Record{translated(key, "적에게 피해를 준다", "對敵人造成傷害")}, nil)
	e2.rule("rules/adjustments/2026-11-01-wording.yaml", adjustment(key, "적에게 피해를 준다", "對敵人造成傷害。"))
	e2.snapshot(map[string]string{"Skills_A.json": dataList(obj{"id": 1, "desc": "적에게 큰 피해를 준다"})})
	sum = e2.run("")
	if want := []string{key}; !reflect.DeepEqual(sum.Adjustments.Expired, want) || len(sum.Adjustments.ExpiredUnits) != 0 {
		t.Errorf("Adjustments = %+v, want %q expired and no ExpiredUnits", sum.Adjustments, want)
	}
	if data, _ := json.Marshal(sum.Adjustments); strings.Contains(string(data), "expired_units") {
		t.Errorf("JSON = %s, want no expired_units", data)
	}
}

// An adjustment made for the current snapshot names its unit by its key, a
// transition or not: one whose KR text that unit does not have expired, and
// the unit is queued with it.
func TestPlanQueuesAnExpiredAdjustmentOfTheCurrentSnapshotAtItsKey(t *testing.T) {
	e := newPlanEnv(t)
	const story = "StoryData/S1.json"
	content := func(i int) string { return at(story, fmt.Sprintf("/dataList/%d/content", i)) }
	e.snapshot(map[string]string{story: dataList(obj{"id": 1, "content": "첫 번째 줄입니다"})})
	e.seed([]store.Record{translated(content(0), "첫 번째 줄입니다", "第一行")}, nil)
	e.snapshot(map[string]string{story: dataList(obj{"id": 1, "content": "새로 들어온 줄입니다"}, obj{"id": 2, "content": "첫 번째 줄이 바뀌었습니다"})})
	// Written for the new snapshot, but with the sha of a text it lacks.
	e.rule("rules/adjustments/2026-11-05-wording.yaml", strings.ReplaceAll(adjustment(content(1), "첫 번째 줄입니다", "第一行改了。"), tag1, tag2))

	e.run(tag1)

	if it := e.pending()[content(1)]; it.Reason != ReasonAdjustmentExpired || it.Refs[RefAdjustment] != "第一行改了。" {
		t.Errorf("the unit of the key = %+v, want adjustment-expired with the adjustment as a reference", it)
	}
	if it := e.pending()[content(0)]; it.Reason != ReasonNew || len(it.Refs) != 0 {
		t.Errorf("the other unit = %+v, want new without a reference", it)
	}
}

// An adjustment stays in effect over several snapshots, at whatever key its
// text moves to, while its raw stays the snapshot it was made for. When it
// expires, its unit is the one it was in effect at: followed from there to the
// current snapshot, not looked up by its key.
func TestPlanQueuesAnExpiredAdjustmentWhereItWasInEffect(t *testing.T) {
	e := newPlanEnv(t)
	const (
		story  = "StoryData/S1.json"
		first  = "첫 번째 줄은 그대로입니다"
		oldKR  = "조정된 줄은 나중에 바뀝니다"
		newKR  = "조정된 줄은 결국 바뀌었습니다"
		insert = "두 번째 판에서 들어온 줄입니다"
		again  = "세 번째 판에서 들어온 줄입니다"
	)
	content := func(i int) string { return at(story, fmt.Sprintf("/dataList/%d/content", i)) }
	e.snapshot(map[string]string{story: dataList(obj{"id": 1, "content": first}, obj{"id": 2, "content": oldKR})})
	e.seed([]store.Record{translated(content(0), first, "第一行不變"), translated(content(1), oldKR, "調整過的那行")}, nil)
	e.rule("rules/adjustments/2026-11-01-wording.yaml", adjustment(content(1), oldKR, "調整過的那一行。"))
	if sum := e.run(""); sum.Adjustments.Applied != 1 {
		t.Fatalf("first snapshot: Adjustments = %+v, want one applied", sum.Adjustments)
	}

	e.snapshot(map[string]string{story: dataList(obj{"id": 1, "content": first}, obj{"id": 2, "content": insert}, obj{"id": 3, "content": oldKR})})
	if sum := e.run(""); !reflect.DeepEqual(sum.Adjustments.Moved, []MovedKey{{From: content(1), To: content(2)}}) {
		t.Fatalf("second snapshot: Adjustments = %+v, want the adjustment moved to line 3", sum.Adjustments)
	}

	e.snapshot(map[string]string{story: dataList(obj{"id": 1, "content": first}, obj{"id": 2, "content": insert}, obj{"id": 3, "content": again}, obj{"id": 4, "content": newKR})})
	sum := e.run("")

	pending := e.pending()
	if got, want := pending[content(3)], (store.PendingItem{
		Key: content(3), Reason: ReasonAdjustmentExpired, PrevKR: oldKR, PrevZH: "調整過的那一行。",
		Refs: map[string]string{RefAdjustment: "調整過的那一行。"},
	}); !reflect.DeepEqual(got, want) {
		t.Errorf("the adjusted line = %+v\nwant %+v", got, want)
	}
	for _, key := range []string{content(1), content(2)} {
		if it := pending[key]; it.Reason != ReasonNew || len(it.Refs) != 0 {
			t.Errorf("inserted line %s = %+v, want new without a reference", key, it)
		}
	}
	if want := []string{content(1)}; !reflect.DeepEqual(sum.Adjustments.Expired, want) {
		t.Errorf("Expired = %q, want %q", sum.Adjustments.Expired, want)
	}
}

func TestPlanReusesTheSameText(t *testing.T) {
	e := newPlanEnv(t)
	const (
		long      = "같은 문장은 같은 번역을 씁니다" // 17 runes
		short     = "짧은말"               // 3 runes
		forbidden = "마법으로 적을 공격합니다"
	)
	e.snapshot(map[string]string{"Skills_A.json": dataList(obj{"id": 1, "desc": long}, obj{"id": 2, "desc": short}, obj{"id": 3, "desc": forbidden})})
	e.seed([]store.Record{
		translated(descKey("Skills_A.json", 0), long, "相同的句子用相同的翻譯"),
		translated(descKey("Skills_A.json", 1), short, "短話"),
		translated(descKey("Skills_A.json", 2), forbidden, "用法術攻擊敵人"), // 마법 is 魔法, never 法術
	}, nil)
	e.snapshot(map[string]string{
		"Skills_A.json": dataList(obj{"id": 1, "desc": long}, obj{"id": 2, "desc": short}, obj{"id": 3, "desc": forbidden}, obj{"id": 4, "desc": short}),
		"Skills_B.json": dataList(obj{"id": 1, "desc": long}, obj{"id": 2, "desc": short}, obj{"id": 3, "desc": forbidden}),
		"Other.json":    dataList(obj{"id": 1, "desc": long}), // another family
	})

	sum := e.run(tag1)

	recs, pending := e.records(), e.pending()
	want := store.Record{
		Key: descKey("Skills_B.json", 0), KRSha: units.KRSha(long), ZH: "相同的句子用相同的翻譯", Origin: store.OriginClaude,
		Since: tag2, Rules: testRulesCommit, Model: "m1", Batch: "reuse:" + descKey("Skills_A.json", 0),
	}
	if got := recs[want.Key]; !reflect.DeepEqual(got, want) {
		t.Errorf("record of the same text in another file = %+v, want %+v", got, want)
	}
	if got := recs[descKey("Skills_A.json", 3)]; got.ZH != "短話" || got.Batch != "reuse:"+descKey("Skills_A.json", 1) {
		t.Errorf("record of the short text in the same file = %+v, want 短話 reused", got)
	}
	for _, key := range []string{descKey("Skills_A.json", 3), descKey("Skills_B.json", 0)} {
		if it, queued := pending[key]; queued {
			t.Errorf("%s is pending (%+v), want it reused", key, it)
		}
	}
	for _, key := range []string{descKey("Skills_B.json", 1), descKey("Skills_B.json", 2), descKey("Other.json", 0)} {
		if _, has := recs[key]; has {
			t.Errorf("%s has a record %+v, want it not reused", key, recs[key])
		}
		if it := pending[key]; it.Reason != ReasonNew {
			t.Errorf("%s pending = %+v, want a new item", key, it)
		}
	}
	if sum.Reused != 2 {
		t.Errorf("Reused = %d, want 2", sum.Reused)
	}
}

// A glossary term that is for some files is enforced on a reused translation
// in the file the translation goes to, not in the file it comes from.
func TestPlanChecksReuseUnderTheTermsOfTheTargetFile(t *testing.T) {
	e := newPlanEnv(t)
	e.rule(rules.GlossaryFile, "- kr: 마법\n  zh: 魔法\n  forbidden: [法術]\n  status: approved\n  files: [\"Skills_B*\"]\n")
	const text = "마법으로 적을 공격합니다"
	e.snapshot(map[string]string{"Skills_A.json": dataList(obj{"id": 1, "desc": text})})
	// Skills_A is not a file of the term, so its translation may say 法術.
	e.seed([]store.Record{translated(descKey("Skills_A.json", 0), text, "用法術攻擊敵人")}, nil)
	e.snapshot(map[string]string{
		"Skills_A.json": dataList(obj{"id": 1, "desc": text}),
		"Skills_B.json": dataList(obj{"id": 1, "desc": text}),
		"Skills_C.json": dataList(obj{"id": 1, "desc": text}),
	})

	sum := e.run(tag1)

	recs, pending := e.records(), e.pending()
	if got := recs[descKey("Skills_C.json", 0)]; got.ZH != "用法術攻擊敵人" || got.Batch != "reuse:"+descKey("Skills_A.json", 0) {
		t.Errorf("record in a file the term is not for = %+v, want the translation reused", got)
	}
	if it, ok := pending[descKey("Skills_B.json", 0)]; !ok || it.Reason != ReasonNew {
		t.Errorf("Skills_B pending = %+v, %v, want a new item: the term forbids 法術 there", it, ok)
	}
	if _, has := recs[descKey("Skills_B.json", 0)]; has || sum.Reused != 1 {
		t.Errorf("Skills_B has a record %+v, Reused = %d; want no record and 1 reuse", recs[descKey("Skills_B.json", 0)], sum.Reused)
	}
}

// A rewrite is held to the terms of the file of each unit it changes.
func TestPlanChecksRewritesUnderTheTermsOfTheFileOfTheUnit(t *testing.T) {
	e := newPlanEnv(t)
	e.rule(rules.GlossaryFile, "- kr: 마법\n  zh: 魔法\n  forbidden: [法術]\n  status: approved\n  files: [\"Skills_B*\"]\n")
	const text = "마법 피해"
	e.snapshot(map[string]string{
		"Skills_A.json": dataList(obj{"id": 1, "desc": text}),
		"Skills_B.json": dataList(obj{"id": 1, "desc": text}),
	})
	e.seed([]store.Record{
		translated(descKey("Skills_A.json", 0), text, "魔法傷害"),
		translated(descKey("Skills_B.json", 0), text, "魔法傷害"),
	}, nil)
	e.rule("rules/rewrite/2026-11-01-wording.yaml", "reason: 改用法術\nfiles: [\"Skills*\"]\npattern: '魔法'\nreplace: '法術'\n")

	e.run("")

	recs := e.records()
	if got := recs[descKey("Skills_A.json", 0)].ZH; got != "法術傷害" {
		t.Errorf("Skills_A zh = %q, want 法術傷害: the term is not for this file", got)
	}
	if got := recs[descKey("Skills_B.json", 0)].ZH; got != "魔法傷害" {
		t.Errorf("Skills_B zh = %q, want 魔法傷害 kept: the term forbids 法術 here", got)
	}
}

func TestPlanReusesADeletedUnit(t *testing.T) {
	e := newPlanEnv(t)
	const text = "번호가 바뀐 항목의 설명입니다"
	e.snapshot(map[string]string{"Skills_C.json": dataList(obj{"id": 1, "desc": text})})
	e.seed([]store.Record{translated(descKey("Skills_C.json", 0), text, "編號改了的項目說明")}, nil)
	// The entry is numbered again: id 1 is gone and id 2 is new.
	e.snapshot(map[string]string{"Skills_C.json": dataList(obj{"id": 5, "desc": "완전히 새로운 항목입니다"}, obj{"id": 2, "desc": text})})

	sum := e.run(tag1)

	recs := e.records()
	got := recs[descKey("Skills_C.json", 1)]
	if got.ZH != "編號改了的項目說明" || got.Origin != store.OriginClaude || got.Batch != "reuse:"+descKey("Skills_C.json", 0) || got.KRSha != units.KRSha(text) {
		t.Errorf("record of the renumbered entry = %+v, want the deleted unit's translation", got)
	}
	if it := e.pending()[descKey("Skills_C.json", 0)]; it.Reason != ReasonNew {
		t.Errorf("the new entry = %+v, want it pending", it)
	}
	if sum.Changes.Deleted != 1 || sum.Changes.New != 2 || sum.Reused != 1 {
		t.Errorf("Changes = %+v, Reused = %d; want 1 deleted, 2 new, 1 reused", sum.Changes, sum.Reused)
	}
}

func TestPlanReportsNewSpeakersAndFiles(t *testing.T) {
	e := newPlanEnv(t)
	const dialogue = "RPGSystem/rpg-loc-dialogue-floor-1.json"
	e.snapshot(map[string]string{
		"StoryData/S1.json": dataList(obj{"id": 0, "model": "이상", "content": "줄 하나"}, obj{"id": 1, "model": "파우스트", "content": "줄 둘"}),
		dialogue:            dataList(obj{"id": 1, "texts": []obj{{"speaker": "안내원", "content": "어서 오세요"}}}),
	})
	e.snapshot(map[string]string{
		"StoryData/S1.json": dataList(obj{"id": 0, "model": "이상", "content": "줄 하나"}, obj{"id": 1, "model": "파우스트", "content": "줄 둘"}, obj{"id": 2, "model": "새인물", "content": "줄 셋"}),
		dialogue:            dataList(obj{"id": 1, "texts": []obj{{"speaker": "안내원", "content": "어서 오세요"}, {"speaker": "새NPC", "content": "반갑습니다"}}}),
		"Misc.json":         dataList(obj{"id": 1, "desc": "새 파일"}),
		"Skills_New.json":   dataList(obj{"id": 1, "desc": "새 스킬"}),
	})

	sum := e.run(tag1)

	if want := []string{"새NPC", "새인물"}; !reflect.DeepEqual(sum.NewSpeakers, want) {
		t.Errorf("NewSpeakers = %q, want %q", sum.NewSpeakers, want)
	}
	if want := []string{"Misc.json"}; !reflect.DeepEqual(sum.NewFilesDefaultFamily, want) {
		t.Errorf("NewFilesDefaultFamily = %q, want %q", sum.NewFilesDefaultFamily, want)
	}
}

// Of two adjustments for one unit, the one whose key still holds the text
// wins over one that moved there, and of equals the later one in file order;
// the others are listed as expired and queue nothing (spec 11.1, R11).
func TestPlanAdjustmentCollisions(t *testing.T) {
	e := newPlanEnv(t)
	const t0, t1 = "첫 번째 설명입니다", "두 번째 설명입니다"
	e.snapshot(map[string]string{"Skills_A.json": dataList(obj{"id": 1, "desc": t0}, obj{"id": 2, "desc": t1})})
	e.seed([]store.Record{translated(descKey("Skills_A.json", 0), t0, "第一個說明"), translated(descKey("Skills_A.json", 1), t1, "第二個說明")}, nil)
	e.rule("rules/adjustments/2026-11-01-a.yaml", adjustment(descKey("Skills_A.json", 0), t0, "甲"))
	// Its key holds nothing, but the text is at /dataList/0/desc: moved there.
	e.rule("rules/adjustments/2026-11-02-b.yaml", adjustment(descKey("Skills_A.json", 7), t0, "乙")+adjustment(descKey("Skills_A.json", 1), t1, "丙"))
	e.rule("rules/adjustments/2026-11-03-c.yaml", adjustment(descKey("Skills_A.json", 1), t1, "丁"))

	sum := e.run("")

	recs := e.records()
	if got := recs[descKey("Skills_A.json", 0)]; got.ZH != "甲" || got.Batch != "adjustment:2026-11-01-a.yaml" {
		t.Errorf("/dataList/0 = %+v, want the applied adjustment 甲", got)
	}
	if got := recs[descKey("Skills_A.json", 1)]; got.ZH != "丁" || got.Batch != "adjustment:2026-11-03-c.yaml" {
		t.Errorf("/dataList/1 = %+v, want the later adjustment 丁", got)
	}
	// 乙 lost the unit it moved to (/dataList/0), so that is its unit.
	want := AdjustmentSummary{
		Applied: 2, Moved: []MovedKey{}, Expired: []string{descKey("Skills_A.json", 1), descKey("Skills_A.json", 7)},
		ExpiredUnits: map[string]string{descKey("Skills_A.json", 7): descKey("Skills_A.json", 0)},
	}
	if !reflect.DeepEqual(sum.Adjustments, want) {
		t.Errorf("Adjustments = %+v, want %+v", sum.Adjustments, want)
	}
	if p := e.pending(); len(p) != 0 {
		t.Errorf("pending = %v, want nothing queued", p)
	}
}

// The field rules and the frozen files come before the adjustments (spec
// 8.2, R13): an adjustment of a unit that is copied, or of a frozen unit that
// has a translation, is not applied and is listed as expired. A frozen unit
// without one takes the adjustment, and keeps it.
func TestPlanAdjustmentsYieldToFieldRulesAndFrozenFiles(t *testing.T) {
	e := newPlanEnv(t)
	const bgm = "BgmLyrics/L.json"
	keyField, lyric, bare := at("Skills_A.json", "/dataList/0/key"), at(bgm, "/dataList/0/content"), at(bgm, "/dataList/1/content")
	e.snapshot(map[string]string{
		"Skills_A.json": dataList(obj{"id": 1, "key": "SLASH"}),
		bgm:             dataList(obj{"id": 1, "content": "노래 가사"}, obj{"id": 2, "content": "다른 가사"}),
	})
	lyricRec := store.Record{Key: lyric, KRSha: units.KRSha("노래 가사"), ZH: "歌詞", Origin: store.OriginParatranz, Since: tag1}
	e.seed([]store.Record{copied(keyField, "SLASH"), lyricRec}, []store.PendingItem{{Key: bare, Reason: ReasonNew}})
	e.rule("rules/adjustments/2026-11-01-a.yaml",
		adjustment(keyField, "SLASH", "斬擊")+adjustment(lyric, "노래 가사", "新歌詞")+adjustment(bare, "다른 가사", "另一段歌詞"))

	for run := 1; run <= 2; run++ {
		sum := e.run("")
		recs := e.records()
		if got := recs[keyField]; got.Origin != store.OriginCopy || got.ZH != "SLASH" {
			t.Errorf("run %d: copied unit = %+v, want the copy kept", run, got)
		}
		if got := recs[lyric]; !reflect.DeepEqual(got, lyricRec) {
			t.Errorf("run %d: frozen unit = %+v, want its translation kept", run, got)
		}
		if got := recs[bare]; got.Origin != store.OriginHuman || got.ZH != "另一段歌詞" {
			t.Errorf("run %d: frozen unit without a translation = %+v, want the adjustment", run, got)
		}
		want := AdjustmentSummary{Applied: 1, Moved: []MovedKey{}, Expired: []string{lyric, keyField}}
		if !reflect.DeepEqual(sum.Adjustments, want) {
			t.Errorf("run %d: Adjustments = %+v, want %+v", run, sum.Adjustments, want)
		}
		if p := e.pending(); len(p) != 0 {
			t.Errorf("run %d: pending = %v, want nothing queued", run, p)
		}
	}
}

// A rewrite never changes a frozen file (R14), and a text that would break the
// rules for a new translation stays as it was.
func TestPlanRewriteSkipsFrozenFilesAndRefusedText(t *testing.T) {
	e := newPlanEnv(t)
	const bgm = "BgmLyrics/L.json"
	e.snapshot(map[string]string{
		"Skills_A.json": dataList(obj{"id": 1, "desc": "피해 2배"}),
		bgm:             dataList(obj{"id": 1, "desc": "노래 2배"}),
	})
	e.seed([]store.Record{translated(descKey("Skills_A.json", 0), "피해 2배", "傷害2倍"), translated(descKey(bgm, 0), "노래 2배", "歌2倍")}, nil)
	e.rule("rules/rewrite/2026-11-01-space.yaml", "reason: 加空格\nfiles: [\"Skills*\", \"BgmLyrics/**\"]\npattern: '(\\d)倍'\nreplace: '${1} 倍'\n")
	e.rule("rules/rewrite/2026-11-02-hangul.yaml", "reason: 壞的改寫\nfiles: [\"Skills*\"]\npattern: '傷害'\nreplace: '피해'\n")

	sum := e.run("")

	recs := e.records()
	if got := recs[descKey("Skills_A.json", 0)].ZH; got != "傷害2 倍" {
		t.Errorf("Skills_A zh = %q, want 傷害2 倍 (the first file applied, the second refused)", got)
	}
	if got := recs[descKey(bgm, 0)].ZH; got != "歌2倍" {
		t.Errorf("frozen zh = %q, want 歌2倍 unchanged", got)
	}
	want := []Applied{{ID: "2026-11-01-space", Units: 1}, {ID: "2026-11-02-hangul", Units: 0}}
	if !reflect.DeepEqual(sum.RewriteApplied, want) {
		t.Errorf("RewriteApplied = %+v, want %+v", sum.RewriteApplied, want)
	}
	if got := e.applied().Rewrite; !reflect.DeepEqual(got, []string{"2026-11-01-space", "2026-11-02-hangul"}) {
		t.Errorf("applied rewrite ids = %q, want both", got)
	}
}

// A unit the field rules translate and that has no translation of its KR text
// is queued (spec 8.2, step 6), and a copy is no translation: it goes.
func TestPlanQueuesUnitsWithoutASource(t *testing.T) {
	e := newPlanEnv(t)
	e.snapshot(map[string]string{"Skills_A.json": dataList(obj{"id": 1, "name": "베기", "desc": "피해를 준다"})})
	desc, name := descKey("Skills_A.json", 0), at("Skills_A.json", "/dataList/0/name")
	e.seed([]store.Record{copied(desc, "피해를 준다")}, nil) // the field rules once skipped desc

	sum := e.run("")

	if rec, has := e.records()[desc]; has {
		t.Errorf("copy record %+v kept for a unit the field rules translate", rec)
	}
	pending := e.pending()
	for _, key := range []string{desc, name} {
		if it := pending[key]; it.Reason != ReasonNew {
			t.Errorf("%s pending = %+v, want a new item", key, it)
		}
	}
	if sum.Pending != 2 {
		t.Errorf("Pending = %d, want 2", sum.Pending)
	}
}
