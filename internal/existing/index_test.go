package existing

import (
	"reflect"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/rules"
)

func krs(ns []*Name) []string {
	out := []string{}
	for _, n := range ns {
		out = append(out, n.KR)
	}
	return out
}

func tableOf(names ...string) *Table {
	t := &Table{RawTag: "raw1", Fields: []string{"name", "abName"}}
	for _, kr := range names {
		t.Names = append(t.Names, Name{KR: kr, Variants: []Variant{{Renderings: []Rendering{{"譯", 1}}}}})
	}
	return t
}

func TestIndexExcludesGlossary(t *testing.T) {
	r := &rules.Rules{Glossary: []rules.Term{{KR: "격돌", ZH: "格鬥", Status: "candidate"}}}
	ix := NewIndex(tableOf("격돌", "노을빛 검"), r, nil)
	if ix.Lookup("격돌") != nil {
		t.Error("glossary name found")
	}
	if got := krs(ix.Mentions("격돌 노을빛 검", map[string]bool{"격돌": true})); !reflect.DeepEqual(got, []string{"노을빛 검"}) {
		t.Errorf("Mentions = %v", got)
	}
	if ix.Lookup(" 노을빛 검 ") == nil {
		t.Error("trimmed lookup failed")
	}
}

func TestIndexDeclined(t *testing.T) {
	ix := NewIndex(tableOf("노을빛 검"), &rules.Rules{}, map[string]bool{"name/노을빛 검": true})
	if !ix.Declined("노을빛 검") || ix.Declined("다른") {
		t.Error("Declined wrong")
	}
	if ix.Lookup("노을빛 검") == nil {
		t.Error("declined name must still be found")
	}
}

func TestMentions(t *testing.T) {
	ix := NewIndex(tableOf("노을빛 검", "회피", "가르는 검", "새벽을 가르는 검"), &rules.Rules{}, nil)
	cases := []struct {
		text   string
		always map[string]bool
		want   []string
	}{
		{"노을빛 검으로 벤다", nil, []string{"노을빛 검"}},
		{"붉은노을빛 검", nil, []string{}},
		{"회피를 한다", nil, []string{}},
		{"'회피' 스킬", nil, []string{"회피"}},
		{"회피를 한다", map[string]bool{"회피": true}, []string{"회피"}},
		{"새벽을 가르는 검", nil, []string{"새벽을 가르는 검"}},
	}
	for _, c := range cases {
		if got := krs(ix.Mentions(c.text, c.always)); !reflect.DeepEqual(got, c.want) {
			t.Errorf("Mentions(%q) = %v, want %v", c.text, got, c.want)
		}
	}
}

func TestMentionsLiteral(t *testing.T) {
	kr := "E.G.O 개화 (정오)"
	ix := NewIndex(tableOf(kr), &rules.Rules{}, nil)
	if got := krs(ix.Mentions("그 "+kr+" 발동", nil)); !reflect.DeepEqual(got, []string{kr}) {
		t.Errorf("Mentions = %v", got)
	}
	if got := ix.Mentions("EXGXO 개화 정오", nil); len(got) != 0 {
		t.Errorf("regex-like match: %v", krs(got))
	}
}

func TestNilIndex(t *testing.T) {
	var ix *Index
	if ix.RawTag() != "" || ix.Fields() != nil || ix.Lookup("a") != nil || ix.IsName("name", "a") ||
		ix.Declined("a") || ix.Mentions("a", nil) != nil {
		t.Error("nil index returned non-zero")
	}
	if NewIndex(nil, nil, nil).Lookup("a") != nil {
		t.Error("empty index found a name")
	}
}

func TestIsName(t *testing.T) {
	ix := NewIndex(tableOf("노을빛 검"), &rules.Rules{}, nil)
	if !ix.IsName("name", "노을빛 검") || ix.IsName("desc", "노을빛 검") || ix.IsName("name", "없음") {
		t.Error("IsName wrong")
	}
}

func TestMentionsTaggedKR(t *testing.T) {
	ix := NewIndex(tableOf("<b>노을빛 검</b>"), &rules.Rules{}, nil)
	if got := krs(ix.Mentions("노을빛 검으로 벤다", nil)); !reflect.DeepEqual(got, []string{"<b>노을빛 검</b>"}) {
		t.Errorf("Mentions = %v", got)
	}
	if ix.Lookup("<b>노을빛 검</b>") == nil {
		t.Error("Lookup by raw KR failed")
	}
}

func TestMentionsNoBreakSpace(t *testing.T) {
	kr := "가르\u00a0검"
	ix := NewIndex(tableOf(kr), &rules.Rules{}, nil)
	if got := krs(ix.Mentions("그 "+kr+" 발동", nil)); !reflect.DeepEqual(got, []string{kr}) {
		t.Errorf("Mentions = %v", got)
	}
}

func TestFieldsIsACopy(t *testing.T) {
	tbl := tableOf("노을빛 검")
	ix := NewIndex(tbl, &rules.Rules{}, nil)
	ix.Fields()[0] = "changed"
	if tbl.Fields[0] != "name" || ix.Fields()[0] != "name" {
		t.Error("Fields exposes the table")
	}
}
