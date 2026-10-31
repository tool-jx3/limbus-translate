package align

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

func kept(oldPtr, newPtr string) VersionMatch {
	return VersionMatch{OldPointer: oldPtr, NewPointer: newPtr, Status: StatusUnchanged}
}

func changed(oldPtr, newPtr string) VersionMatch {
	return VersionMatch{OldPointer: oldPtr, NewPointer: newPtr, Status: StatusChanged}
}

func added(newPtr string) VersionMatch {
	return VersionMatch{NewPointer: newPtr, Status: StatusNew}
}

func deleted(oldPtr string) VersionMatch {
	return VersionMatch{OldPointer: oldPtr, Status: StatusDeleted}
}

func wantVersions(t *testing.T, got, want []VersionMatch) {
	t.Helper()
	if reflect.DeepEqual(got, want) {
		return
	}
	format := func(ms []VersionMatch) string {
		var b strings.Builder
		for _, m := range ms {
			fmt.Fprintf(&b, "\n\t%-9s %q -> %q", m.Status, m.OldPointer, m.NewPointer)
		}
		return b.String()
	}
	t.Errorf("AlignVersions got:%s\nwant:%s", format(got), format(want))
}

// story builds a story file of content-only lines numbered from 1.
func story(t *testing.T, contents ...string) *jsonx.Node {
	t.Helper()
	items := make([]string, len(contents))
	for i, c := range contents {
		items[i] = fmt.Sprintf(`{"id": %d, "content": %q}`, i+1, c)
	}
	return mustParse(t, `{"dataList": [`+strings.Join(items, ", ")+`]}`)
}

func TestVersionsStoryInsert(t *testing.T) {
	// A line is inserted after line 1 and the later lines are renumbered:
	// new id 2 is the inserted line and old id 2 is new id 3, so matching by
	// id would be wrong.
	old, cur := load(t, "story_insert_old.json"), load(t, "story_insert_new.json")
	got := AlignVersions("StoryData/S918A.json", old, cur)
	wantVersions(t, got, []VersionMatch{
		kept("/dataList/0/place", "/dataList/0/place"),
		kept("/dataList/0/model", "/dataList/0/model"),
		kept("/dataList/0/teller", "/dataList/0/teller"),
		kept("/dataList/0/content", "/dataList/0/content"),
		added("/dataList/1/model"),
		added("/dataList/1/teller"),
		added("/dataList/1/content"),
		kept("/dataList/1/model", "/dataList/2/model"),
		kept("/dataList/1/teller", "/dataList/2/teller"),
		kept("/dataList/1/content", "/dataList/2/content"),
		kept("/dataList/2/content", "/dataList/3/content"),
	})
}

// Story lines for the edit tests. B2 is an edit of B (Dice 0.76) and Y is
// not (Dice 0.24); X and D share no bigram with B or B2.
const (
	lineA  = "선장, 오늘은 어디로 가는 거야?"
	lineB  = "그야 물론 모험이지!"
	lineB2 = "그야 물론 모험이라네!"
	lineY  = "그야 당연하지."
	lineC  = "또 시작이네."
	lineD  = "인터뷰어는 한숨을 쉬었다."
	lineX  = "조용히 해. 누가 듣겠어."
)

func TestVersionsStoryEdit(t *testing.T) {
	const file = "StoryData/S100A.json"
	t.Run("replace", func(t *testing.T) {
		got := AlignVersions(file, story(t, lineA, lineB, lineC), story(t, lineA, lineB2, lineC))
		wantVersions(t, got, []VersionMatch{
			kept("/dataList/0/content", "/dataList/0/content"),
			changed("/dataList/1/content", "/dataList/1/content"),
			kept("/dataList/2/content", "/dataList/2/content"),
		})
	})
	t.Run("replace and insert", func(t *testing.T) {
		got := AlignVersions(file, story(t, lineA, lineB, lineC), story(t, lineA, lineB2, lineX, lineC))
		wantVersions(t, got, []VersionMatch{
			kept("/dataList/0/content", "/dataList/0/content"),
			changed("/dataList/1/content", "/dataList/1/content"),
			added("/dataList/2/content"),
			kept("/dataList/2/content", "/dataList/3/content"),
		})
	})
	t.Run("insert before edit", func(t *testing.T) {
		got := AlignVersions(file, story(t, lineA, lineB, lineC), story(t, lineA, lineX, lineB2, lineC))
		wantVersions(t, got, []VersionMatch{
			kept("/dataList/0/content", "/dataList/0/content"),
			added("/dataList/1/content"),
			changed("/dataList/1/content", "/dataList/2/content"),
			kept("/dataList/2/content", "/dataList/3/content"),
		})
	})
	t.Run("replace and delete", func(t *testing.T) {
		got := AlignVersions(file, story(t, lineA, lineB, lineD, lineC), story(t, lineA, lineB2, lineC))
		wantVersions(t, got, []VersionMatch{
			kept("/dataList/0/content", "/dataList/0/content"),
			changed("/dataList/1/content", "/dataList/1/content"),
			kept("/dataList/3/content", "/dataList/2/content"),
			deleted("/dataList/2/content"),
		})
	})
	t.Run("unrelated replacement", func(t *testing.T) {
		// Too little in common to be an edit of the old line.
		got := AlignVersions(file, story(t, lineA, lineB, lineC), story(t, lineA, lineY, lineC))
		wantVersions(t, got, []VersionMatch{
			kept("/dataList/0/content", "/dataList/0/content"),
			added("/dataList/1/content"),
			kept("/dataList/2/content", "/dataList/2/content"),
			deleted("/dataList/1/content"),
		})
	})
	t.Run("inserted speaker", func(t *testing.T) {
		// The inserted line's model and teller come first in the unmatched
		// run, yet only a content string may pair with the old content.
		old := story(t, lineA, lineB, lineC)
		cur := mustParse(t, fmt.Sprintf(`{"dataList": [
			{"id": 1, "content": %q},
			{"id": 2, "model": "중지제자", "teller": "???", "content": %q},
			{"id": 3, "content": %q},
			{"id": 4, "content": %q}]}`, lineA, lineX, lineB2, lineC))
		got := AlignVersions(file, old, cur)
		wantVersions(t, got, []VersionMatch{
			kept("/dataList/0/content", "/dataList/0/content"),
			added("/dataList/1/model"),
			added("/dataList/1/teller"),
			added("/dataList/1/content"),
			changed("/dataList/1/content", "/dataList/2/content"),
			kept("/dataList/2/content", "/dataList/3/content"),
		})
		for _, m := range got {
			if m.Status == StatusChanged && units.FieldName(m.OldPointer) != units.FieldName(m.NewPointer) {
				t.Errorf("changed pair across fields: %+v", m)
			}
		}
	})
	t.Run("same text in another field", func(t *testing.T) {
		old := mustParse(t, `{"dataList": [{"id": 1, "teller": "???", "content": "가"}]}`)
		cur := mustParse(t, `{"dataList": [{"id": 1, "title": "???", "content": "가"}]}`)
		wantVersions(t, AlignVersions(file, old, cur), []VersionMatch{
			added("/dataList/0/title"),
			kept("/dataList/0/content", "/dataList/0/content"),
			deleted("/dataList/0/teller"),
		})
	})
}

func TestVersionsCRLF(t *testing.T) {
	// KR text is compared after "\r\n" becomes "\n", as kr_sha does.
	t.Run("sequence", func(t *testing.T) {
		got := AlignVersions("StoryData/S100A.json", story(t, lineA, "x\r\ny", lineC), story(t, lineA, "x\ny", lineC))
		wantVersions(t, got, []VersionMatch{
			kept("/dataList/0/content", "/dataList/0/content"),
			kept("/dataList/1/content", "/dataList/1/content"),
			kept("/dataList/2/content", "/dataList/2/content"),
		})
	})
	t.Run("id and pointer", func(t *testing.T) {
		old := mustParse(t, `{"title": "제\r\n목", "dataList": [{"id": "a", "desc": "가\r\n나"}]}`)
		cur := mustParse(t, `{"title": "제\n목", "dataList": [{"id": "a", "desc": "가\n나"}]}`)
		wantVersions(t, AlignVersions("Sample.json", old, cur), []VersionMatch{
			kept("/title", "/title"),
			kept("/dataList/0/id", "/dataList/0/id"),
			kept("/dataList/0/desc", "/dataList/0/desc"),
		})
	})
	t.Run("next to an insertion", func(t *testing.T) {
		got := AlignVersions("StoryData/S100A.json",
			story(t, lineA, "x\r\ny", lineC), story(t, lineA, lineX, "x\ny", lineC))
		wantVersions(t, got, []VersionMatch{
			kept("/dataList/0/content", "/dataList/0/content"),
			added("/dataList/1/content"),
			kept("/dataList/1/content", "/dataList/2/content"),
			kept("/dataList/2/content", "/dataList/3/content"),
		})
	})
}

func TestDice(t *testing.T) {
	cases := []struct {
		a, b string
		want float64
	}{
		{"가나다", "가나다", 1},
		{"가나다", "라마바", 0},
		{"가나다라", "가나마바", 2.0 / 6}, // bigram 가나 in common
		{"ㅋㅋㅋ", "ㅋㅋ", 1},          // repeated bigrams count once
		{"가", "가", 1},             // shorter than 2 runes: single runes
		{"가", "나", 0},
		{"가", "가나", 0},             // a rune is not a bigram
		{"\x00가", "가", 0},          // nor is a bigram that starts with NUL
		{lineB, lineB2, 16.0 / 21}, // 8 of 10 and 11 bigrams
		{lineB, lineY, 4.0 / 17},   // 그야 and "야 " of 10 and 7
		{lineB, lineX, 0},
	}
	for _, c := range cases {
		if got := dice(grams(c.a), grams(c.b)); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("dice(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestPairEdits(t *testing.T) {
	// Each text has 10 distinct bigrams. s shares 3 with t3 (Dice 0.3) and 2
	// with t2 (Dice 0.2); t9 shares 9 with t3 (Dice 0.9) and 7 with t2.
	const (
		s  = "가나다라마바사아자차카"
		t3 = "가나다라파타하거너더러"
		t2 = "가나다파타하거너더러너"
		t9 = "가나다라파타하거너더리"
	)
	content := func(texts ...string) []text {
		out := make([]text, len(texts))
		for i, s := range texts {
			out[i] = text{"content", s}
		}
		return out
	}
	cases := []struct {
		name string
		a, b []text
		want [][2]int
	}{
		{"at the threshold", content(s), content(t3), [][2]int{{0, 0}}},
		{"below the threshold", content(s), content(t2), nil},
		{"another field", []text{{"name", t3}}, content(t9), nil},
		{"most alike wins", content(s, t9), content(t3), [][2]int{{1, 0}}},
		// 0.7 + 0.3 beats the single 0.9 pair, which would cross them.
		{"most alike in total wins", content(t9, s), content(t2, t3), [][2]int{{0, 0}, {1, 1}}},
	}
	for _, c := range cases {
		if got := pairEdits(c.a, c.b); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: pairEdits = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestVersionsIDFieldAdded(t *testing.T) {
	old := mustParse(t, `{"dataList": [
		{"id": "Agility", "name": "신속", "desc": "이번 턴 동안 속도 +{0}", "summary": "속도 +{0}"},
		{"id": "Burn", "name": "화상", "desc": "{1}턴 동안 턴 종료 시 {0}만큼 고정 체력 피해를 입음"},
		{"id": "Endurance", "name": "수비 위력 증가", "desc": "이번 턴 동안 수비 스킬의 최종 위력 +{0}"}]}`)
	cur := mustParse(t, `{"dataList": [
		{"id": "Agility", "name": "신속", "desc": "이번 턴 동안 속도 +{0}", "summary": "속도 +{0}"},
		{"id": "Burn", "name": "화상", "desc": "{1}턴 동안 턴 종료 시 {0}만큼 고정 체력 피해를 입음", "summary": "고정 체력 피해 -{0}"},
		{"id": "Endurance", "name": "수비 위력 증가", "desc": "이번 턴 동안 수비 스킬의 최종 위력 +{0}"}]}`)
	got := AlignVersions("Bufs.json", old, cur)

	var want []VersionMatch
	for _, p := range []string{
		"/dataList/0/id", "/dataList/0/name", "/dataList/0/desc", "/dataList/0/summary",
		"/dataList/1/id", "/dataList/1/name", "/dataList/1/desc",
	} {
		want = append(want, kept(p, p))
	}
	want = append(want, added("/dataList/1/summary"))
	for _, p := range []string{"/dataList/2/id", "/dataList/2/name", "/dataList/2/desc"} {
		want = append(want, kept(p, p))
	}
	wantVersions(t, got, want)
}

func TestVersionsIDInsertMiddle(t *testing.T) {
	old := mustParse(t, `{"dataList": [
		{"id": 9001, "name": "지옥나비의 꿈", "desc": "화상 위력 3 부여"},
		{"id": 9002, "name": "녹슨 동전", "desc": "공격 위력 +1"}]}`)
	cur := mustParse(t, `{"dataList": [
		{"id": 9001, "name": "지옥나비의 꿈", "desc": "화상 위력 3 부여"},
		{"id": 9050, "name": "깨진 거울", "desc": "속도 +1"},
		{"id": 9002, "name": "녹슨 동전", "desc": "공격 위력 +1"}]}`)
	got := AlignVersions("EGOgift_MirrorDungeon.json", old, cur)
	wantVersions(t, got, []VersionMatch{
		kept("/dataList/0/name", "/dataList/0/name"),
		kept("/dataList/0/desc", "/dataList/0/desc"),
		added("/dataList/1/name"),
		added("/dataList/1/desc"),
		kept("/dataList/1/name", "/dataList/2/name"),
		kept("/dataList/1/desc", "/dataList/2/desc"),
	})
}

func TestVersionsIDReordered(t *testing.T) {
	// Matching is by id, so moving entries changes nothing, while an edited
	// field is changed wherever its entry went.
	old := mustParse(t, `{"dataList": [
		{"id": "a", "name": "가", "desc": "가 설명"},
		{"id": "b", "name": "나", "desc": "나 설명"},
		{"id": "c", "name": "다", "desc": "다 설명"}]}`)
	cur := mustParse(t, `{"dataList": [
		{"id": "c", "name": "다", "desc": "다 설명"},
		{"id": "a", "name": "가", "desc": "가 설명"},
		{"id": "b", "name": "나", "desc": "나 새 설명"}]}`)
	got := AlignVersions("Sample.json", old, cur)
	wantVersions(t, got, []VersionMatch{
		kept("/dataList/2/id", "/dataList/0/id"),
		kept("/dataList/2/name", "/dataList/0/name"),
		kept("/dataList/2/desc", "/dataList/0/desc"),
		kept("/dataList/0/id", "/dataList/1/id"),
		kept("/dataList/0/name", "/dataList/1/name"),
		kept("/dataList/0/desc", "/dataList/1/desc"),
		kept("/dataList/1/id", "/dataList/2/id"),
		kept("/dataList/1/name", "/dataList/2/name"),
		changed("/dataList/1/desc", "/dataList/2/desc"),
	})
}

func TestVersionsDeleted(t *testing.T) {
	t.Run("id", func(t *testing.T) {
		old := mustParse(t, `{"dataList": [
			{"id": 1, "name": "가", "desc": "가 설명"},
			{"id": 2, "name": "나", "desc": "나 설명"},
			{"id": 3, "name": "다", "desc": "다 설명"}]}`)
		cur := mustParse(t, `{"dataList": [
			{"id": 1, "name": "가", "desc": "가 설명"},
			{"id": 3, "name": "다"}]}`)
		got := AlignVersions("Sample.json", old, cur)
		wantVersions(t, got, []VersionMatch{
			kept("/dataList/0/name", "/dataList/0/name"),
			kept("/dataList/0/desc", "/dataList/0/desc"),
			kept("/dataList/2/name", "/dataList/1/name"),
			deleted("/dataList/1/name"),
			deleted("/dataList/1/desc"),
			deleted("/dataList/2/desc"),
		})
	})
	t.Run("sequence", func(t *testing.T) {
		got := AlignVersions("StoryData/S100A.json", story(t, "가", "나", "다", "라"), story(t, "가", "다"))
		wantVersions(t, got, []VersionMatch{
			kept("/dataList/0/content", "/dataList/0/content"),
			kept("/dataList/2/content", "/dataList/1/content"),
			deleted("/dataList/1/content"),
			deleted("/dataList/3/content"),
		})
	})
}

func TestVersionsWithoutUniqueIDs(t *testing.T) {
	// Outside StoryData/, entries still match by sequence when ids repeat
	// or are missing.
	t.Run("repeated ids", func(t *testing.T) {
		old := mustParse(t, `{"dataList": [{"id": 1, "name": "가"}, {"id": 1, "name": "나"}]}`)
		cur := mustParse(t, `{"dataList": [{"id": 1, "name": "다"}, {"id": 1, "name": "가"}, {"id": 1, "name": "나"}]}`)
		wantVersions(t, AlignVersions("Sample.json", old, cur), []VersionMatch{
			added("/dataList/0/name"),
			kept("/dataList/0/name", "/dataList/1/name"),
			kept("/dataList/1/name", "/dataList/2/name"),
		})
	})
	t.Run("missing id", func(t *testing.T) {
		old := mustParse(t, `{"dataList": [{"id": 1, "name": "가"}, {"name": "나"}]}`)
		cur := mustParse(t, `{"dataList": [{"name": "나"}, {"id": 1, "name": "가"}]}`)
		wantVersions(t, AlignVersions("Sample.json", old, cur), []VersionMatch{
			kept("/dataList/1/name", "/dataList/0/name"),
			added("/dataList/1/name"),
			deleted("/dataList/0/name"),
		})
	})
}

func TestVersionsOutsideDataList(t *testing.T) {
	t.Run("no dataList", func(t *testing.T) {
		old := mustParse(t, `{"a": "가", "b": "나"}`)
		cur := mustParse(t, `{"b": "나", "a": "가!", "c": "다"}`)
		wantVersions(t, AlignVersions("Sample.json", old, cur), []VersionMatch{
			kept("/b", "/b"),
			changed("/a", "/a"),
			added("/c"),
		})
	})
	t.Run("beside dataList", func(t *testing.T) {
		old := mustParse(t, `{"title": "제목", "dataList": [{"id": "a", "name": "가"}]}`)
		cur := mustParse(t, `{"dataList": [{"id": "b", "name": "나"}, {"id": "a", "name": "가"}], "title": "새 제목"}`)
		wantVersions(t, AlignVersions("Sample.json", old, cur), []VersionMatch{
			added("/dataList/0/id"),
			added("/dataList/0/name"),
			kept("/dataList/0/id", "/dataList/1/id"),
			kept("/dataList/0/name", "/dataList/1/name"),
			changed("/title", "/title"),
		})
	})
	t.Run("repeated key", func(t *testing.T) {
		// Duplicate members share a pointer; each old one pairs only once.
		old := mustParse(t, `{"a": "가", "a": "나"}`)
		cur := mustParse(t, `{"a": "가", "a": "나!"}`)
		wantVersions(t, AlignVersions("Sample.json", old, cur), []VersionMatch{
			kept("/a", "/a"),
			changed("/a", "/a"),
		})
	})
}

func TestVersionsNilDocument(t *testing.T) {
	doc := story(t, "가", "나")
	wantVersions(t, AlignVersions("StoryData/S100A.json", nil, doc), []VersionMatch{
		added("/dataList/0/content"),
		added("/dataList/1/content"),
	})
	wantVersions(t, AlignVersions("StoryData/S100A.json", doc, nil), []VersionMatch{
		deleted("/dataList/0/content"),
		deleted("/dataList/1/content"),
	})
}
