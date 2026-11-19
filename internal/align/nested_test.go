package align

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/jsonx"
)

// skillDescs is a skills file with one entry per element of coins: the entry
// i has the id 100+i and one level whose only coin has coins[i] as the texts
// of its coindescs.
func skillDescs(t *testing.T, coins ...[]string) *jsonx.Node {
	t.Helper()
	entries := make([]string, len(coins))
	for i, descs := range coins {
		items := make([]string, len(descs))
		for k, d := range descs {
			items[k] = fmt.Sprintf(`{"desc": %q}`, d)
		}
		entries[i] = fmt.Sprintf(`{"id": %d, "levelList": [{"level": 3, "name": "기술 %d", "coinlist": [{"coindescs": [%s]}]}]}`,
			100+i, i, strings.Join(items, ", "))
	}
	return mustParse(t, `{"dataList": [`+strings.Join(entries, ", ")+`]}`)
}

func skillName(entry int) VersionMatch {
	p := fmt.Sprintf("/dataList/%d/levelList/0/name", entry)
	return kept(p, p)
}

func coinDesc(entry, i int) string {
	return fmt.Sprintf("/dataList/%d/levelList/0/coinlist/0/coindescs/%d/desc", entry, i)
}

func TestVersionsNestedInsert(t *testing.T) {
	// A string inserted in the middle of a nested array moves the ones after
	// it; they are the same strings.
	old := skillDescs(t, []string{"적중 시 진동 1 부여", "합동 기술 : 가", "합동 기술 : 나"})
	cur := skillDescs(t, []string{"적중 시 진동 1 부여", "적중 시 관통 피해", "합동 기술 : 가", "합동 기술 : 나"})
	wantVersions(t, AlignVersions("Skills_sample.json", old, cur), []VersionMatch{
		skillName(0),
		kept(coinDesc(0, 0), coinDesc(0, 0)),
		added(coinDesc(0, 1)),
		kept(coinDesc(0, 1), coinDesc(0, 2)),
		kept(coinDesc(0, 2), coinDesc(0, 3)),
	})
}

func TestVersionsNestedRemove(t *testing.T) {
	old := skillDescs(t, []string{"가 설명", "나 설명", "다 설명", "라 설명"})
	cur := skillDescs(t, []string{"가 설명", "다 설명", "라 설명"})
	wantVersions(t, AlignVersions("Skills_sample.json", old, cur), []VersionMatch{
		skillName(0),
		kept(coinDesc(0, 0), coinDesc(0, 0)),
		kept(coinDesc(0, 2), coinDesc(0, 1)),
		kept(coinDesc(0, 3), coinDesc(0, 2)),
		deleted(coinDesc(0, 1)),
	})
}

func TestVersionsNestedWithoutShiftPairsByPosition(t *testing.T) {
	// No element is found at another index of its array: the elements pair by
	// position, whatever their texts.
	for _, c := range []struct {
		name     string
		old, cur []string
		want     []VersionMatch
	}{
		{
			"edit", []string{"가 설명", "나 설명", "다 설명"}, []string{"가 설명", "전혀 다른 글", "다 설명"},
			[]VersionMatch{kept(coinDesc(0, 0), coinDesc(0, 0)), changed(coinDesc(0, 1), coinDesc(0, 1)), kept(coinDesc(0, 2), coinDesc(0, 2))},
		},
		{
			"edit and append", []string{"가 설명", "나 설명"}, []string{"가 설명", "전혀 다른 글", "또 다른 글"},
			[]VersionMatch{kept(coinDesc(0, 0), coinDesc(0, 0)), changed(coinDesc(0, 1), coinDesc(0, 1)), added(coinDesc(0, 2))},
		},
		{
			"append a twin", []string{"가 설명", "나 설명"}, []string{"가 설명", "나 설명", "나 설명"},
			[]VersionMatch{kept(coinDesc(0, 0), coinDesc(0, 0)), kept(coinDesc(0, 1), coinDesc(0, 1)), added(coinDesc(0, 2))},
		},
		{
			// The second 가 became 나: the 나 after it did not move.
			"edit into a twin", []string{"가 설명", "가 설명", "나 설명"}, []string{"가 설명", "나 설명", "나 설명"},
			[]VersionMatch{kept(coinDesc(0, 0), coinDesc(0, 0)), changed(coinDesc(0, 1), coinDesc(0, 1)), kept(coinDesc(0, 2), coinDesc(0, 2))},
		},
		{
			"truncate", []string{"가 설명", "나 설명", "다 설명"}, []string{"가 설명", "나 설명"},
			[]VersionMatch{kept(coinDesc(0, 0), coinDesc(0, 0)), kept(coinDesc(0, 1), coinDesc(0, 1)), deleted(coinDesc(0, 2))},
		},
		{
			"all edited", []string{"가 설명", "나 설명"}, []string{"전혀 다른 글", "또 다른 글", "셋째 글"},
			[]VersionMatch{changed(coinDesc(0, 0), coinDesc(0, 0)), changed(coinDesc(0, 1), coinDesc(0, 1)), added(coinDesc(0, 2))},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			want := append([]VersionMatch{skillName(0)}, c.want...)
			wantVersions(t, AlignVersions("Skills_sample.json", skillDescs(t, c.old), skillDescs(t, c.cur)), want)
		})
	}
}

func TestVersionsNestedInsertBesideEdit(t *testing.T) {
	// Between two elements that only moved, an edited element pairs with the
	// new one that is like it, not with the one inserted before it.
	old := skillDescs(t, []string{"적중 시 진동 1 부여", "합동 기술 : 피해량 +20%, 진동 2 부여", "끝 설명"})
	cur := skillDescs(t, []string{"적중 시 진동 1 부여", "이 코인 최종 피해만큼 관통", "합동 기술 : 피해량 +30%, 진동 2 부여", "끝 설명"})
	wantVersions(t, AlignVersions("Skills_sample.json", old, cur), []VersionMatch{
		skillName(0),
		kept(coinDesc(0, 0), coinDesc(0, 0)),
		added(coinDesc(0, 1)),
		changed(coinDesc(0, 1), coinDesc(0, 2)),
		kept(coinDesc(0, 2), coinDesc(0, 3)),
	})
}

func TestVersionsNestedReplacedBesideInsert(t *testing.T) {
	// Where a shift was found, an old element that is like none of the new
	// ones beside it is deleted, and they are new: nothing says which of them
	// it became.
	old := skillDescs(t, []string{"가 설명", "나 설명", "다 설명"})
	cur := skillDescs(t, []string{"가 설명", "전혀 다른 글", "또 다른 글", "다 설명"})
	wantVersions(t, AlignVersions("Skills_sample.json", old, cur), []VersionMatch{
		skillName(0),
		kept(coinDesc(0, 0), coinDesc(0, 0)),
		added(coinDesc(0, 1)),
		added(coinDesc(0, 2)),
		kept(coinDesc(0, 2), coinDesc(0, 3)),
		deleted(coinDesc(0, 1)),
	})
}

func TestVersionsNestedEditBetweenMovedElements(t *testing.T) {
	// As many old as new elements between two that moved: they pair in order.
	old := skillDescs(t, []string{"가 설명", "나 설명", "다 설명"})
	cur := skillDescs(t, []string{"새 설명", "가 설명", "전혀 다른 글", "다 설명"})
	wantVersions(t, AlignVersions("Skills_sample.json", old, cur), []VersionMatch{
		skillName(0),
		added(coinDesc(0, 0)),
		kept(coinDesc(0, 0), coinDesc(0, 1)),
		changed(coinDesc(0, 1), coinDesc(0, 2)),
		kept(coinDesc(0, 2), coinDesc(0, 3)),
	})
}

func TestVersionsNestedTwins(t *testing.T) {
	// Twins keep their order: the first one of the old version is the one
	// that went.
	old := skillDescs(t, []string{"첫째", "……", "둘째", "……", "끝"})
	cur := skillDescs(t, []string{"첫째", "둘째", "……", "끝"})
	wantVersions(t, AlignVersions("Skills_sample.json", old, cur), []VersionMatch{
		skillName(0),
		kept(coinDesc(0, 0), coinDesc(0, 0)),
		kept(coinDesc(0, 2), coinDesc(0, 1)),
		kept(coinDesc(0, 3), coinDesc(0, 2)),
		kept(coinDesc(0, 4), coinDesc(0, 3)),
		deleted(coinDesc(0, 1)),
	})
}

func TestVersionsNestedNeverAcrossEntries(t *testing.T) {
	// The text one entry lost and another gained is deleted and new: an
	// entry's strings only move within it.
	old := skillDescs(t, []string{"가 설명", "옮긴 설명", "다 설명"}, []string{"라 설명", "마 설명"})
	cur := skillDescs(t, []string{"가 설명", "다 설명"}, []string{"라 설명", "옮긴 설명", "마 설명"})
	wantVersions(t, AlignVersions("Skills_sample.json", old, cur), []VersionMatch{
		skillName(0),
		kept(coinDesc(0, 0), coinDesc(0, 0)),
		kept(coinDesc(0, 2), coinDesc(0, 1)),
		skillName(1),
		kept(coinDesc(1, 0), coinDesc(1, 0)),
		added(coinDesc(1, 1)),
		kept(coinDesc(1, 1), coinDesc(1, 2)),
		deleted(coinDesc(0, 1)),
	})
}

func TestVersionsNestedNeverAcrossArrays(t *testing.T) {
	// Nor from one array of the entry to another.
	old := mustParse(t, `{"dataList": [{"id": 7, "coinlist": [
		{"coindescs": [{"desc": "가 설명"}, {"desc": "옮긴 설명"}]},
		{"coindescs": [{"desc": "나 설명"}]}]}]}`)
	cur := mustParse(t, `{"dataList": [{"id": 7, "coinlist": [
		{"coindescs": [{"desc": "가 설명"}]},
		{"coindescs": [{"desc": "옮긴 설명"}, {"desc": "나 설명"}]}]}]}`)
	wantVersions(t, AlignVersions("Skills_sample.json", old, cur), []VersionMatch{
		kept("/dataList/0/coinlist/0/coindescs/0/desc", "/dataList/0/coinlist/0/coindescs/0/desc"),
		added("/dataList/0/coinlist/1/coindescs/0/desc"),
		kept("/dataList/0/coinlist/1/coindescs/0/desc", "/dataList/0/coinlist/1/coindescs/1/desc"),
		deleted("/dataList/0/coinlist/0/coindescs/1/desc"),
	})
}

func TestVersionsNestedArraysInArrays(t *testing.T) {
	// A whole element with arrays of its own inserted before another: the
	// strings of the one that moved are the same strings, at every depth, and
	// an array of strings shifts like one of objects.
	old := mustParse(t, `{"dataList": [{"id": 7, "levelList": [
		{"name": "기술", "tags": ["가", "나"], "coinlist": [{"coindescs": [{"desc": "하나"}, {"desc": "둘"}]}]}]}]}`)
	cur := mustParse(t, `{"dataList": [{"id": 7, "levelList": [
		{"name": "새 기술", "tags": ["다"], "coinlist": [{"coindescs": [{"desc": "셋"}]}]},
		{"name": "기술", "tags": ["가", "나"], "coinlist": [{"coindescs": [{"desc": "하나"}, {"desc": "둘"}]}]}]}]}`)
	wantVersions(t, AlignVersions("Skills_sample.json", old, cur), []VersionMatch{
		added("/dataList/0/levelList/0/name"),
		added("/dataList/0/levelList/0/tags/0"),
		added("/dataList/0/levelList/0/coinlist/0/coindescs/0/desc"),
		kept("/dataList/0/levelList/0/name", "/dataList/0/levelList/1/name"),
		kept("/dataList/0/levelList/0/tags/0", "/dataList/0/levelList/1/tags/0"),
		kept("/dataList/0/levelList/0/tags/1", "/dataList/0/levelList/1/tags/1"),
		kept("/dataList/0/levelList/0/coinlist/0/coindescs/0/desc", "/dataList/0/levelList/1/coinlist/0/coindescs/0/desc"),
		kept("/dataList/0/levelList/0/coinlist/0/coindescs/1/desc", "/dataList/0/levelList/1/coinlist/0/coindescs/1/desc"),
	})

	old = mustParse(t, `{"dataList": [{"id": 7, "tags": ["가", "나"]}]}`)
	cur = mustParse(t, `{"dataList": [{"id": 7, "tags": ["다", "가", "나"]}]}`)
	wantVersions(t, AlignVersions("Skills_sample.json", old, cur), []VersionMatch{
		added("/dataList/0/tags/0"),
		kept("/dataList/0/tags/0", "/dataList/0/tags/1"),
		kept("/dataList/0/tags/1", "/dataList/0/tags/2"),
	})
}

func TestVersionsNestedShiftInAnEditedElement(t *testing.T) {
	// The level was edited (its name), so it pairs by position; the shift in
	// its own array is still found.
	old := mustParse(t, `{"dataList": [{"id": 7, "levelList": [
		{"name": "기술", "coinlist": [{"coindescs": [{"desc": "하나"}, {"desc": "둘"}]}]}]}]}`)
	cur := mustParse(t, `{"dataList": [{"id": 7, "levelList": [
		{"name": "고친 기술", "coinlist": [{"coindescs": [{"desc": "하나"}, {"desc": "새것"}, {"desc": "둘"}]}]}]}]}`)
	wantVersions(t, AlignVersions("Skills_sample.json", old, cur), []VersionMatch{
		changed("/dataList/0/levelList/0/name", "/dataList/0/levelList/0/name"),
		kept("/dataList/0/levelList/0/coinlist/0/coindescs/0/desc", "/dataList/0/levelList/0/coinlist/0/coindescs/0/desc"),
		added("/dataList/0/levelList/0/coinlist/0/coindescs/1/desc"),
		kept("/dataList/0/levelList/0/coinlist/0/coindescs/1/desc", "/dataList/0/levelList/0/coinlist/0/coindescs/2/desc"),
	})
}

func TestVersionsNestedOutsideDataListPairsByPointer(t *testing.T) {
	// Only the entries of dataList are looked into: elsewhere a string is the
	// one at its pointer, as before.
	old := mustParse(t, `{"notes": ["가", "나"], "dataList": [{"id": 1, "name": "이름"}]}`)
	cur := mustParse(t, `{"notes": ["다", "가", "나"], "dataList": [{"id": 1, "name": "이름"}]}`)
	wantVersions(t, AlignVersions("Sample.json", old, cur), []VersionMatch{
		changed("/notes/0", "/notes/0"),
		changed("/notes/1", "/notes/1"),
		added("/notes/2"),
		kept("/dataList/0/name", "/dataList/0/name"),
	})
}
