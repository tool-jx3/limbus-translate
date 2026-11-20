package plan

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/store"
)

// skillsFile has the shape of the Skills_personality files: an entry is a
// skill with one level, whose coins each hold a list of coindescs.
const skillsFile = "Skills_personality-11.json"

type skill struct {
	id    int
	desc  string
	coins [][]string // the desc of every coindesc of every coin
}

func skillsDoc(skills ...skill) string {
	var entries []obj
	for _, s := range skills {
		var coins []obj
		for _, descs := range s.coins {
			var list []obj
			for _, d := range descs {
				list = append(list, obj{"desc": d})
			}
			coins = append(coins, obj{"coindescs": list})
		}
		entries = append(entries, obj{"id": s.id, "levelList": []obj{{"level": 3, "desc": s.desc, "coinlist": coins}}})
	}
	return dataList(entries...)
}

func skillDescKey(entry int) string {
	return at(skillsFile, fmt.Sprintf("/dataList/%d/levelList/0/desc", entry))
}

func coinKey(entry, coin, i int) string {
	return at(skillsFile, fmt.Sprintf("/dataList/%d/levelList/0/coinlist/%d/coindescs/%d/desc", entry, coin, i))
}

// The texts of the nested fixture, in the style of the real file.
const (
	vibration = "[OnSucceedAttack] [Vibration] 1 부여"
	pierce    = "[OnSucceedAttack] 이 코인 최종 피해량의 15%만큼 관통 피해"
	comboA    = "[ComboSkill] : 새벽 사무소 해결사\n- 피해량 +20%"
	comboB    = "[ComboSkill] : 쐐기 사무소 해결사\n- 크리티컬 피해량 +20%"
	comboC    = "[ComboSkill] : 쐐기 사무소 대표\n- [VibrationExplosion]"
	useOld    = "[WhenUse] 자신의 [Breath] 횟수 3 증가\n아군 수만큼 공격 가중치 증가"
	useNew    = "[WhenUse] 자신의 [Breath] 횟수 3 증가\n아군 수만큼 공격 가중치 증가\n적중 시 관통 피해를 준다"
)

// The update of 2026-11-19 in small: one string inserted in the middle of the
// coindescs of two coins of one skill, whose description changed too. The
// strings after the inserted one only moved: their records and pending items
// go with them, and only the inserted strings and the description are queued.
func TestTransitionNestedInsertMovesRecords(t *testing.T) {
	oldDoc := skillsDoc(
		skill{1111605, "다른 기술 설명", [][]string{{vibration, comboA}}},
		skill{1111606, useOld, [][]string{{vibration, comboA}, {vibration, comboB, comboC}}},
	)
	newDoc := skillsDoc(
		skill{1111605, "다른 기술 설명", [][]string{{vibration, comboA}}},
		skill{1111606, useNew, [][]string{{vibration, pierce, comboA}, {vibration, pierce, comboB, comboC}}},
	)
	recs := recordsOf(
		translated(skillDescKey(0), "다른 기술 설명", "另一個技能"),
		translated(coinKey(0, 0, 0), vibration, "震顫甲"),
		translated(coinKey(0, 0, 1), comboA, "合擊甲"),
		translated(skillDescKey(1), useOld, "使用時"),
		translated(coinKey(1, 0, 0), vibration, "震顫乙"),
		translated(coinKey(1, 0, 1), comboA, "合擊乙"),
		translated(coinKey(1, 1, 0), vibration, "震顫丙"),
		translated(coinKey(1, 1, 1), comboB, "合擊丙"),
		translated(coinKey(1, 1, 2), comboC, "合擊丁"),
	)
	lease := &store.Lease{Batch: "b-7", Until: time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)}
	pending := []store.PendingItem{{Key: coinKey(1, 1, 2), Reason: "requeue", PrevZH: "合擊丁", Hint: "語氣", Lease: lease}}

	m := transition(t, map[string]string{skillsFile: oldDoc}, map[string]string{skillsFile: newDoc}, recs, pending)

	wantRecs := map[string]store.Record{}
	for from, to := range map[string]string{
		skillDescKey(0):  skillDescKey(0),
		coinKey(0, 0, 0): coinKey(0, 0, 0),
		coinKey(0, 0, 1): coinKey(0, 0, 1),
		coinKey(1, 0, 0): coinKey(1, 0, 0),
		coinKey(1, 0, 1): coinKey(1, 0, 2),
		coinKey(1, 1, 0): coinKey(1, 1, 0),
		coinKey(1, 1, 1): coinKey(1, 1, 2),
		coinKey(1, 1, 2): coinKey(1, 1, 3),
	} {
		r := recs[from]
		r.Key = to
		wantRecs[to] = r // the same record, Since and all
	}
	if !reflect.DeepEqual(m.Records, wantRecs) {
		t.Errorf("records = %v\nwant %v", m.Records, wantRecs)
	}
	wantPending := []store.PendingItem{
		{Key: coinKey(1, 0, 1), Reason: ReasonNew},
		{Key: coinKey(1, 1, 1), Reason: ReasonNew},
		{Key: coinKey(1, 1, 3), Reason: "requeue", PrevZH: "合擊丁", Hint: "語氣"}, // moved, its lease gone
		{Key: skillDescKey(1), Reason: ReasonKRChanged, PrevKR: useOld, PrevZH: "使用時"},
	}
	if !reflect.DeepEqual(m.Pending, wantPending) {
		t.Errorf("pending = %+v\nwant %+v", m.Pending, wantPending)
	}
	want := Changes{Unchanged: 8, Moved: 3, Changed: 1, New: 2, FilesChanged: []string{skillsFile}, FilesShifted: []string{skillsFile}}
	if !reflect.DeepEqual(m.Changes, want) {
		t.Errorf("Changes = %+v, want %+v", m.Changes, want)
	}
	if len(m.Deleted) != 0 {
		t.Errorf("Deleted = %+v, want none", m.Deleted)
	}
}

// A string removed from the middle of a nested array is deleted, with its
// record for a unit of its text to take, and the ones after it move up.
func TestTransitionNestedRemoveMovesRecords(t *testing.T) {
	oldDoc := skillsDoc(skill{1111606, useOld, [][]string{{vibration, pierce, comboA, comboB}}})
	newDoc := skillsDoc(skill{1111606, useOld, [][]string{{vibration, comboA, comboB}}})
	recs := recordsOf(
		translated(skillDescKey(0), useOld, "使用時"),
		translated(coinKey(0, 0, 0), vibration, "震顫"),
		translated(coinKey(0, 0, 1), pierce, "穿透"),
		translated(coinKey(0, 0, 2), comboA, "合擊甲"),
		translated(coinKey(0, 0, 3), comboB, "合擊乙"),
	)

	m := transition(t, map[string]string{skillsFile: oldDoc}, map[string]string{skillsFile: newDoc}, recs, nil)

	if len(m.Pending) != 0 {
		t.Errorf("pending = %+v, want none", m.Pending)
	}
	for key, zh := range map[string]string{coinKey(0, 0, 0): "震顫", coinKey(0, 0, 1): "合擊甲", coinKey(0, 0, 2): "合擊乙"} {
		if got := m.Records[key]; got.ZH != zh {
			t.Errorf("record at %s = %+v, want %s", key, got, zh)
		}
	}
	if len(m.Records) != 4 {
		t.Errorf("got %d records, want 4: %v", len(m.Records), m.Records)
	}
	if len(m.Deleted) != 1 || m.Deleted[0].KR != pierce || m.Deleted[0].Record == nil || m.Deleted[0].Record.ZH != "穿透" {
		t.Errorf("Deleted = %+v, want the removed string with its record", m.Deleted)
	}
	want := Changes{Unchanged: 4, Moved: 2, Deleted: 1, FilesChanged: []string{skillsFile}, FilesShifted: []string{skillsFile}}
	if !reflect.DeepEqual(m.Changes, want) {
		t.Errorf("Changes = %+v, want %+v", m.Changes, want)
	}
}

// A unit that moved within a nested array keeps what a person did to it: the
// record of its adjustment goes with it and the adjustment is in effect at the
// new key. An adjustment whose unit moved and changed expires, and the unit is
// queued with it at its new key. A second plan changes nothing.
func TestPlanNestedShiftKeepsAdjustments(t *testing.T) {
	e := newPlanEnv(t)
	const (
		edited    = "[ComboSkill] : 쐐기 사무소 해결사\n- 크리티컬 피해량 +30%\n- 진동 부여"
		adjFile   = "2026-11-01-wording.yaml"
		adjusted  = "合擊甲（人工）"
		adjusted2 = "合擊乙（人工）"
	)
	e.snapshot(map[string]string{skillsFile: skillsDoc(
		skill{1111606, useOld, [][]string{{vibration, comboA}, {vibration, comboB, comboC}}},
	)})
	e.seed([]store.Record{
		translated(skillDescKey(0), useOld, "使用時"),
		translated(coinKey(0, 0, 0), vibration, "震顫甲"),
		translated(coinKey(0, 0, 1), comboA, "合擊甲"),
		translated(coinKey(0, 1, 0), vibration, "震顫乙"),
		translated(coinKey(0, 1, 1), comboB, "合擊乙"),
		translated(coinKey(0, 1, 2), comboC, "合擊丙"),
	}, nil)
	e.rule("rules/adjustments/"+adjFile, adjustment(coinKey(0, 0, 1), comboA, adjusted)+adjustment(coinKey(0, 1, 1), comboB, adjusted2))
	if sum := e.run(""); sum.Adjustments.Applied != 2 {
		t.Fatalf("first run: Adjustments = %+v, want two applied", sum.Adjustments)
	}

	e.snapshot(map[string]string{skillsFile: skillsDoc(
		skill{1111606, useOld, [][]string{{vibration, pierce, comboA}, {vibration, pierce, edited, comboC}}},
	)})
	sum := e.run("")

	if c := sum.Changes; c.Unchanged != 5 || c.Moved != 2 || c.Changed != 1 || c.New != 2 || c.Deleted != 0 {
		t.Errorf("Changes = %+v, want 5 unchanged (2 moved), 1 changed, 2 new", c)
	}
	recs := e.records()
	human := store.Record{
		Key: coinKey(0, 0, 2), KRSha: recs[coinKey(0, 0, 2)].KRSha, ZH: adjusted, Origin: store.OriginHuman,
		Since: tag1, Rules: testRulesCommit, Batch: BatchAdjustment + adjFile,
	}
	if got := recs[coinKey(0, 0, 2)]; !reflect.DeepEqual(got, human) {
		t.Errorf("record of the adjusted unit = %+v\nwant %+v (carried with the unit)", got, human)
	}
	if got := recs[coinKey(0, 1, 3)]; got.ZH != "合擊丙" || got.Since != tag1 {
		t.Errorf("record of the unit that moved = %+v, want 合擊丙 since %s", got, tag1)
	}
	wantPending := map[string]store.PendingItem{
		coinKey(0, 0, 1): {Key: coinKey(0, 0, 1), Reason: ReasonNew},
		coinKey(0, 1, 1): {Key: coinKey(0, 1, 1), Reason: ReasonNew},
		coinKey(0, 1, 2): {
			Key: coinKey(0, 1, 2), Reason: ReasonAdjustmentExpired, PrevKR: comboB, PrevZH: adjusted2,
			Refs: map[string]string{RefAdjustment: adjusted2},
		},
	}
	if got := e.pending(); !reflect.DeepEqual(got, wantPending) {
		t.Errorf("pending = %v\nwant %v", got, wantPending)
	}
	a := sum.Adjustments
	if want := []MovedKey{{From: coinKey(0, 0, 1), To: coinKey(0, 0, 2)}}; a.Applied != 0 || !reflect.DeepEqual(a.Moved, want) {
		t.Errorf("Adjustments = %+v, want one moved: %v", a, want)
	}
	if want := []string{coinKey(0, 1, 1)}; !reflect.DeepEqual(a.Expired, want) {
		t.Errorf("Expired = %q, want %q", a.Expired, want)
	}
	if want := map[string]string{coinKey(0, 1, 1): coinKey(0, 1, 2)}; !reflect.DeepEqual(a.ExpiredUnits, want) {
		t.Errorf("ExpiredUnits = %v, want %v", a.ExpiredUnits, want)
	}

	e.again()
}
