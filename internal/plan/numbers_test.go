package plan

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

func TestPatchNumbers(t *testing.T) {
	for _, c := range []struct {
		name             string
		oldKR, newKR, zh string
		want             string // "" when the translation cannot be made
	}{
		{
			name:  "the numbers that changed",
			oldKR: "속도가 더 빠르면 피해량 +(속도 차이 × 5)% (최대 40%)",
			newKR: "속도가 더 빠르면 피해량 +(속도 차이 × 7)% (최대 56%)",
			zh:    "速度較快時，傷害量+（速度差×5）%（最多40%）",
			want:  "速度較快時，傷害量+（速度差×7）%（最多56%）",
		},
		{
			name:  "a number that stayed is left alone",
			oldKR: "3턴 동안 피해량 +10%", newKR: "3턴 동안 피해량 +20%",
			zh: "3回合內傷害量+10%", want: "3回合內傷害量+20%",
		},
		{
			name:  "numbers of another length",
			oldKR: "최대 5, 99회", newKR: "최대 10, 100회",
			zh: "最多5，99次", want: "最多10，100次",
		},
		{
			name:  "twins that changed alike",
			oldKR: "화상 3 부여, 출혈 3 부여", newKR: "화상 5 부여, 출혈 5 부여",
			zh: "施加3燒傷、3流血", want: "施加5燒傷、5流血",
		},
		{
			name:  "[ID]s, placeholders and tags are not numbers",
			oldKR: "[Burn2] {0}턴 <size=80%>3</size> 부여", newKR: "[Burn2] {0}턴 <size=80%>4</size> 부여",
			zh: "{0}回合施加<size=80%>3</size> [Burn2]", want: "{0}回合施加<size=80%>4</size> [Burn2]",
		},
		{
			name:  "line ends written another way",
			oldKR: "피해량 +10%\r\n최대 2회", newKR: "피해량 +10%\n최대 3회",
			zh: "傷害量+10%\n最多2次", want: "傷害量+10%\n最多3次",
		},

		{name: "the same text", oldKR: "피해량 +10%", newKR: "피해량 +10%", zh: "傷害量+10%"},
		{name: "no number at all", oldKR: "피해를 준다", newKR: "큰 피해를 준다", zh: "造成傷害"},
		{name: "words changed too", oldKR: "피해량 +10%", newKR: "피해량 +20% 증가", zh: "傷害量+10%"},
		{name: "a number more", oldKR: "피해량 +10%", newKR: "피해량 +10% 2", zh: "傷害量+10%"},
		{name: "a number became a word", oldKR: "2회 공격", newKR: "두 번 공격", zh: "攻擊2次"},
		{name: "a digit in a tag changed", oldKR: "<size=80%>3</size>", newKR: "<size=90%>3</size>", zh: "<size=80%>3</size>"},
		{name: "a placeholder changed", oldKR: "{0}턴 3 부여", newKR: "{1}턴 3 부여", zh: "{0}回合施加3"},
		{name: "an [ID] changed", oldKR: "[Burn1] 3 부여", newKR: "[Burn2] 3 부여", zh: "施加3 [Burn1]"},
		{name: "the translation writes a number in words", oldKR: "2회 공격, 피해량 +10%", newKR: "3회 공격, 피해량 +10%", zh: "攻擊兩次，傷害量+10%"},
		{name: "the translation has a number more", oldKR: "피해량 +10%", newKR: "피해량 +20%", zh: "第1枚硬幣傷害量+10%"},
		{name: "the translation has the numbers in another order", oldKR: "5회, 최대 40%", newKR: "7회, 최대 56%", zh: "最多40%，5次"},
		{name: "the translation writes a number another way", oldKR: "최대 1000", newKR: "최대 2000", zh: "最多1,000"},
		{name: "the translation has full-width digits", oldKR: "최대 5회", newKR: "최대 7회", zh: "最多５次"},
		{name: "twins of which one changed", oldKR: "화상 3 부여, 출혈 3 부여", newKR: "화상 5 부여, 출혈 3 부여", zh: "施加3流血、3燒傷"},
		{name: "twins that changed apart", oldKR: "화상 3 부여, 출혈 3 부여", newKR: "화상 5 부여, 출혈 4 부여", zh: "施加3燒傷、3流血"},
		{name: "an empty translation", oldKR: "최대 5회", newKR: "최대 7회", zh: ""},
	} {
		got, ok := patchNumbers(c.oldKR, c.newKR, c.zh)
		if ok != (c.want != "") || got != c.want {
			t.Errorf("%s: patchNumbers(%q, %q, %q) = %q, %v, want %q", c.name, c.oldKR, c.newKR, c.zh, got, ok, c.want)
		}
	}
}

// The transition hands the plan the record of every unit it queued as
// changed, with the KR text it translated: not those of frozen files, which
// keep their record, nor of units without one.
func TestTransitionListsTheChangedUnitsWithTheirRecords(t *testing.T) {
	oldLines := []string{"첫째 줄", "피해량 +10% 증가", "셋째 줄은 번역이 없다", "최대 2회 공격"}
	newLines := []string{"끼운 줄", "첫째 줄", "피해량 +20% 증가", "셋째 줄은 번역이 없었다", "최대 3회 공격"}
	bgm := func(s string) string { return fmt.Sprintf(`{"dataList":[{"id":1,"lyrics":%q}]}`, s) }
	plain := translated(contentKey(storyFile, 1), oldLines[1], "傷害量增加10%")
	plain.Flags = []string{"a"}
	asked := translated(contentKey(storyFile, 3), oldLines[3], "最多攻擊2次")
	recs := recordsOf(
		translated(contentKey(storyFile, 0), oldLines[0], "第一行"), plain, asked,
		translated(units.Key(bgmFile, "/dataList/0/lyrics"), "가사 1절", "歌詞第1段"),
	)
	pending := []store.PendingItem{{Key: asked.Key, Reason: "requeue", PrevZH: asked.ZH, Hint: "語氣"}}

	m := transition(t,
		map[string]string{storyFile: story(oldLines...), bgmFile: bgm("가사 1절")},
		map[string]string{storyFile: story(newLines...), bgmFile: bgm("가사 2절")}, recs, pending)

	want := []ChangedUnit{
		{Key: contentKey(storyFile, 2), KR: oldLines[1], Record: plain},
		{Key: contentKey(storyFile, 4), KR: oldLines[3], Record: asked, Queued: true},
	}
	if !reflect.DeepEqual(m.Changed, want) {
		t.Errorf("Changed = %+v\nwant %+v", m.Changed, want)
	}
	// The result shares no memory with the records.
	m.Changed[0].Record.Flags[0] = "changed"
	if recs[plain.Key].Flags[0] != "a" {
		t.Errorf("Changed shares the flags of the record it was made from")
	}
}

// The texts of the number tests: a skill description of which the update
// changed two numbers, as in the update of 2026-11-19.
const (
	skillsA    = "Skills_A.json"
	speedOld   = "속도가 더 빠르면 피해량 +(속도 차이 × 5)% (최대 40%)"
	speedNew   = "속도가 더 빠르면 피해량 +(속도 차이 × 7)% (최대 56%)"
	speedZH    = "速度較快時，傷害量+（速度差×5）%（最多40%）"
	speedNewZH = "速度較快時，傷害量+（速度差×7）%（最多56%）"
)

// A unit of which only numbers changed is not queued: its translation is the
// old one with the numbers that changed, in a record that says the same of
// where it comes from as before. The unit may have moved as well. The summary
// counts it, and keeps the count when the snapshot is planned again.
func TestPlanPatchesTheNumbersOfAChangedUnit(t *testing.T) {
	e := newPlanEnv(t)
	e.snapshot(map[string]string{skillsA: dataList(obj{"id": 1, "desc": speedOld}, obj{"id": 2, "desc": "적에게 피해를 준다"})})
	rec := translated(descKey(skillsA, 0), speedOld, speedZH)
	rec.Written, rec.Flags = tag1, []string{"reviewed"}
	e.seed([]store.Record{rec, translated(descKey(skillsA, 1), "적에게 피해를 준다", "對敵人造成傷害")}, nil)
	e.run("")

	e.snapshot(map[string]string{skillsA: dataList(
		obj{"id": 9, "desc": "새로 생긴 기술의 설명"}, obj{"id": 1, "desc": speedNew}, obj{"id": 2, "desc": "적에게 피해를 준다"},
	)})
	sum := e.run("")

	want := rec // the origin, the model, the batch, the raw tags and the flags it had
	want.Key, want.KRSha, want.ZH = descKey(skillsA, 1), units.KRSha(speedNew), speedNewZH
	if got := e.records()[descKey(skillsA, 1)]; !reflect.DeepEqual(got, want) {
		t.Errorf("record = %+v\nwant %+v", got, want)
	}
	wantPending := map[string]store.PendingItem{descKey(skillsA, 0): {Key: descKey(skillsA, 0), Reason: ReasonNew}}
	if got := e.pending(); !reflect.DeepEqual(got, wantPending) {
		t.Errorf("pending = %v\nwant %v", got, wantPending)
	}
	if sum.NumberPatched != 1 || sum.Changes.Changed != 1 || sum.Reused != 0 || sum.Pending != 1 {
		t.Errorf("summary = %+v, want one changed unit, patched, and one pending", sum)
	}
	if !strings.Contains(sum.String(), "number-patched 1") {
		t.Errorf("the report of the CLI does not count the patched unit:\n%s", sum)
	}
	if data, err := json.Marshal(sum); err != nil || !strings.Contains(string(data), `"number_patched":1`) {
		t.Errorf("plan.json = %s (%v), want number_patched 1", data, err)
	}

	if again := e.again(); again.NumberPatched != 1 {
		t.Errorf("a second plan of the snapshot says NumberPatched = %d, want the 1 the snapshot brought", again.NumberPatched)
	}
}

// What keeps a unit of which only numbers changed for a translator, or as it
// was: every case is queued or settled as it would be without the patch.
func TestPlanLeavesNumbersItCannotPatch(t *testing.T) {
	const (
		magicOld = "마법 피해량 +10%"
		magicNew = "마법 피해량 +20%"
		adjFile  = "2026-11-01-wording.yaml"
	)
	key := descKey(skillsA, 0)
	changed := func(prevZH string) store.PendingItem {
		return store.PendingItem{Key: key, Reason: ReasonKRChanged, PrevKR: speedOld, PrevZH: prevZH}
	}
	for _, c := range []struct {
		name string
		// rec makes the record of the unit from a translation of speedOld.
		rec     func(r store.Record) store.Record
		pending []store.PendingItem
		// rules writes rule files once the first snapshot was planned.
		rules func(e *planEnv)
		// The unit after the plan of the second snapshot: its pending item
		// (nil: none) and the text of its record ("": none).
		wantItem *store.PendingItem
		wantZH   string
	}{
		{
			name:     "a stand-in is no translation",
			rec:      func(r store.Record) store.Record { r.Origin = store.OriginFallback; return r },
			wantItem: ptr(changed(speedZH)),
		},
		{
			name:     "a record a person has to look at",
			rec:      func(r store.Record) store.Record { r.Flags = []string{store.FlagNeedsReview}; return r },
			wantItem: ptr(changed(speedZH)),
		},
		{
			name: "what a person wrote",
			rec: func(r store.Record) store.Record {
				r.Origin, r.Model, r.Batch = store.OriginHuman, "", BatchAdjustment+"gone.yaml"
				return r
			},
			wantItem: ptr(changed(speedZH)),
		},
		{
			name:     "a unit a person sent back",
			rec:      func(r store.Record) store.Record { return r },
			pending:  []store.PendingItem{{Key: key, Reason: "requeue", PrevZH: speedZH, Hint: "語氣"}},
			wantItem: ptr(store.PendingItem{Key: key, Reason: ReasonKRChanged, PrevKR: speedOld, PrevZH: speedZH, Hint: "語氣"}),
		},
		{
			name:     "a unit that was queued already",
			rec:      func(r store.Record) store.Record { return r },
			pending:  []store.PendingItem{{Key: key, Reason: "structural-fail", PrevZH: speedZH}},
			wantItem: ptr(changed(speedZH)),
		},
		{
			name: "a translation that writes the numbers another way",
			rec: func(r store.Record) store.Record {
				r.ZH = "速度較快時，傷害量+（速度差×五）%（最多40%）"
				return r
			},
			wantItem: ptr(changed("速度較快時，傷害量+（速度差×五）%（最多40%）")),
		},
		{
			name: "an adjustment made for the new text",
			rec:  func(r store.Record) store.Record { return r },
			rules: func(e *planEnv) {
				e.rule("rules/adjustments/"+adjFile, strings.Replace(adjustment(key, speedNew, "人工新譯7、56。"), tag1, tag2, 1))
			},
			wantZH: "人工新譯7、56。",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			e := newPlanEnv(t)
			e.snapshot(map[string]string{skillsA: dataList(obj{"id": 1, "desc": speedOld})})
			e.seed([]store.Record{c.rec(translated(key, speedOld, speedZH))}, c.pending)
			e.run("")
			if c.rules != nil {
				c.rules(e)
			}
			e.snapshot(map[string]string{skillsA: dataList(obj{"id": 1, "desc": speedNew})})

			sum := e.run("")

			if sum.NumberPatched != 0 {
				t.Errorf("NumberPatched = %d, want 0", sum.NumberPatched)
			}
			it, queued := e.pending()[key]
			if c.wantItem == nil && queued || c.wantItem != nil && !reflect.DeepEqual(it, *c.wantItem) {
				t.Errorf("pending item = %+v (queued %v), want %+v", it, queued, c.wantItem)
			}
			if got := e.records()[key].ZH; got != c.wantZH {
				t.Errorf("record text = %q, want %q", got, c.wantZH)
			}
			e.again()
		})
	}

	// An adjustment in effect whose KR text changed expires as it always did:
	// the unit goes to a translator with the adjustment as a reference.
	t.Run("an adjustment that expires", func(t *testing.T) {
		e := newPlanEnv(t)
		e.snapshot(map[string]string{skillsA: dataList(obj{"id": 1, "desc": speedOld})})
		e.seed([]store.Record{translated(key, speedOld, speedZH)}, nil)
		e.rule("rules/adjustments/"+adjFile, adjustment(key, speedOld, "人工舊譯5、40。"))
		e.run("")
		e.snapshot(map[string]string{skillsA: dataList(obj{"id": 1, "desc": speedNew})})

		sum := e.run("")

		want := store.PendingItem{
			Key: key, Reason: ReasonAdjustmentExpired, PrevKR: speedOld, PrevZH: "人工舊譯5、40。",
			Refs: map[string]string{RefAdjustment: "人工舊譯5、40。"},
		}
		if got := e.pending()[key]; !reflect.DeepEqual(got, want) || sum.NumberPatched != 0 {
			t.Errorf("pending item = %+v, NumberPatched = %d\nwant %+v and none patched", got, sum.NumberPatched, want)
		}
		if _, has := e.records()[key]; has {
			t.Errorf("the unit has a record, want none before it is translated again")
		}
		e.again()
	})

	// So does an adjustment that was made for the old text but never planned
	// before the text changed: the record is still the translator's, and what
	// the person wrote must reach a translator all the same.
	t.Run("an adjustment that expires before it was in effect", func(t *testing.T) {
		e := newPlanEnv(t)
		e.snapshot(map[string]string{skillsA: dataList(obj{"id": 1, "desc": speedOld})})
		e.seed([]store.Record{translated(key, speedOld, speedZH)}, nil)
		e.rule("rules/adjustments/"+adjFile, adjustment(key, speedOld, "人工舊譯5、40。"))
		e.snapshot(map[string]string{skillsA: dataList(obj{"id": 1, "desc": speedNew})})

		sum := e.run(tag1)

		want := store.PendingItem{
			Key: key, Reason: ReasonAdjustmentExpired, PrevKR: speedOld, PrevZH: speedZH,
			Refs: map[string]string{RefAdjustment: "人工舊譯5、40。"},
		}
		if got := e.pending()[key]; !reflect.DeepEqual(got, want) || sum.NumberPatched != 0 {
			t.Errorf("pending item = %+v, NumberPatched = %d\nwant %+v and none patched", got, sum.NumberPatched, want)
		}
		if _, has := e.records()[key]; has {
			t.Errorf("the unit has a record, want none before it is translated again")
		}
	})

	// The new text is held to the rules of a new translation: an old
	// translation with a rendering the glossary forbids is not carried on.
	t.Run("a translation the rules refuse", func(t *testing.T) {
		e := newPlanEnv(t)
		e.snapshot(map[string]string{skillsA: dataList(obj{"id": 1, "desc": magicOld})})
		e.seed([]store.Record{translated(key, magicOld, "法術傷害量+10%")}, nil)
		e.run("")
		e.snapshot(map[string]string{skillsA: dataList(obj{"id": 1, "desc": magicNew})})

		sum := e.run("")

		want := store.PendingItem{Key: key, Reason: ReasonKRChanged, PrevKR: magicOld, PrevZH: "法術傷害量+10%"}
		if got := e.pending()[key]; !reflect.DeepEqual(got, want) || sum.NumberPatched != 0 {
			t.Errorf("pending item = %+v, NumberPatched = %d\nwant %+v and none patched", got, sum.NumberPatched, want)
		}
		e.again()
	})

	// A frozen file keeps its translation as it is, numbers and all.
	t.Run("a frozen file", func(t *testing.T) {
		e := newPlanEnv(t)
		const bgm = "BgmLyrics/L.json"
		e.snapshot(map[string]string{bgm: dataList(obj{"id": 1, "desc": speedOld})})
		rec := translated(descKey(bgm, 0), speedOld, speedZH)
		e.seed([]store.Record{rec}, nil)
		e.run("")
		e.snapshot(map[string]string{bgm: dataList(obj{"id": 1, "desc": speedNew})})

		sum := e.run("")

		if got := e.records()[descKey(bgm, 0)]; !reflect.DeepEqual(got, rec) {
			t.Errorf("record = %+v, want it as it was: %+v", got, rec)
		}
		if want := []string{descKey(bgm, 0)}; !reflect.DeepEqual(sum.FrozenChanged, want) || sum.NumberPatched != 0 || len(e.pending()) != 0 {
			t.Errorf("FrozenChanged = %q, NumberPatched = %d, pending = %v, want %q, 0 and none", sum.FrozenChanged, sum.NumberPatched, e.pending(), want)
		}
		e.again()
	})
}

func ptr[T any](v T) *T { return &v }

// A rewrite file applied in the plan that patches a record rewrites the
// patched text like that of any other record.
func TestPlanRewritesAPatchedRecord(t *testing.T) {
	e := newPlanEnv(t)
	key := descKey(skillsA, 0)
	e.snapshot(map[string]string{skillsA: dataList(obj{"id": 1, "desc": "최대 5회 공격"})})
	e.seed([]store.Record{translated(key, "최대 5회 공격", "最多攻擊5次")}, nil)
	e.run("")
	e.snapshot(map[string]string{skillsA: dataList(obj{"id": 1, "desc": "최대 7회 공격"})})
	e.rule("rules/rewrite/2026-11-06-space.yaml", "reason: 加空格\nfiles: [\"Skills*\"]\npattern: '(\\d)次'\nreplace: '${1} 次'\n")

	sum := e.run("")

	if got := e.records()[key]; got.ZH != "最多攻擊7 次" || got.KRSha != units.KRSha("최대 7회 공격") {
		t.Errorf("record = %+v, want the patched text rewritten", got)
	}
	if want := []Applied{{ID: "2026-11-06-space", Units: 1}}; !reflect.DeepEqual(sum.RewriteApplied, want) || sum.NumberPatched != 1 {
		t.Errorf("RewriteApplied = %+v, NumberPatched = %d, want %+v and 1", sum.RewriteApplied, sum.NumberPatched, want)
	}
	if len(e.pending()) != 0 {
		t.Errorf("pending = %v, want none", e.pending())
	}
	e.again()
}
