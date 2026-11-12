package existing

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"slices"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/store"
)

var nameFields = []string{"name", "abName"}

func src(key, kr, zh, jp, en string) Source {
	return Source{Key: key, Field: "name", KR: kr, ZH: zh, Origin: store.OriginClaude, JP: jp, EN: en}
}

func TestCollectGroupsVariantsByJPEN(t *testing.T) {
	kr := "노을빛 검"
	units := []Source{
		src("k5", kr, "夕陽", "夕焼け", "Sunset"),
		src("k3", kr, "晚霞之劍", "茜さす剣", "Glow Blade"),
		src("k2", kr, "霞光劍刃", "茜さす剣", "Glow Blade"),
		src("k1", kr, "霞光劍刃", "茜さす剣", "Glow Blade"),
	}
	tbl := Collect("raw1", nameFields, units)
	if len(tbl.Names) != 1 || len(tbl.Names[0].Variants) != 2 {
		t.Fatalf("Names = %+v, want one name with two variants", tbl.Names)
	}
	v := tbl.Names[0].Variants[0]
	if want := []Rendering{{"霞光劍刃", 2}, {"晚霞之劍", 1}}; !reflect.DeepEqual(v.Renderings, want) {
		t.Errorf("Renderings = %+v, want %+v", v.Renderings, want)
	}
	if want := []string{"k1", "k2", "k3"}; !reflect.DeepEqual(v.Examples, want) {
		t.Errorf("Examples = %v, want %v", v.Examples, want)
	}
	v2 := tbl.Names[0].Variants[1]
	if v2.JP != "夕焼け" || v2.EN != "Sunset" || !reflect.DeepEqual(v2.Renderings, []Rendering{{"夕陽", 1}}) ||
		!reflect.DeepEqual(v2.Examples, []string{"k5"}) {
		t.Errorf("second variant = %+v", v2)
	}
	if v.JP != "茜さす剣" || v.EN != "Glow Blade" {
		t.Errorf("variant = %+v", v)
	}
	if tbl.RawTag != "raw1" || !reflect.DeepEqual(tbl.Fields, nameFields) {
		t.Errorf("header = %q %v", tbl.RawTag, tbl.Fields)
	}
}

func TestCollectENIgnoresCase(t *testing.T) {
	units := []Source{
		src("b", "노을빛 검", "霞光劍刃", "茜さす剣", "glow blade"),
		src("a", "노을빛 검", "霞光劍刃", "茜さす剣", "Glow Blade"),
	}
	tbl := Collect("r", nameFields, units)
	vs := tbl.Names[0].Variants
	if len(vs) != 1 || vs[0].EN != "Glow Blade" || vs[0].Renderings[0].Units != 2 {
		t.Errorf("variants = %+v", vs)
	}
}

func TestCollectSkips(t *testing.T) {
	bad := []Source{
		{Key: "1", Field: "name", KR: "노을빛 검", ZH: "劍", Origin: store.OriginCopy},
		{Key: "2", Field: "name", KR: "노을빛 검", ZH: "劍", Origin: store.OriginFallback},
		{Key: "3", Field: "name", KR: "노을빛 검", ZH: "노을빛 검", Origin: store.OriginClaude},
		{Key: "4", Field: "name", KR: "노을빛 검", ZH: "霞光 검", Origin: store.OriginClaude},
		{Key: "5", Field: "name", KR: " 검 ", ZH: "劍", Origin: store.OriginClaude},
		{Key: "6", Field: "desc", KR: "노을빛 검", ZH: "劍", Origin: store.OriginClaude},
		{Key: "7", Field: "name", KR: "노을빛 검", ZH: "<b></b>", Origin: store.OriginClaude},
		{Key: "8", Field: "name", KR: "노을빛 검", ZH: "劍", Origin: ""},
	}
	if tbl := Collect("r", nameFields, bad); len(tbl.Names) != 0 {
		t.Errorf("Names = %+v, want none", tbl.Names)
	}
}

// A KR without a letter is no name: "..." would be found in nearly every
// line with an ellipsis. One letter among the punctuation is enough.
func TestCollectSkipsKRWithoutALetter(t *testing.T) {
	tbl := Collect("r", nameFields, []Source{
		src("1", "...", "……", "", ""),
		src("2", "!?", "！？", "", ""),
		src("3", "<b>--</b>", "——", "", ""),
		src("4", "1234", "一二三四", "", ""),
		src("5", "...검", "……劍", "", ""),
	})
	var got []string
	for _, n := range tbl.Names {
		got = append(got, n.KR)
	}
	if want := []string{"...검"}; !slices.Equal(got, want) {
		t.Errorf("names %q, want %q", got, want)
	}
}

func TestCollectStripsTags(t *testing.T) {
	tbl := Collect("r", nameFields, []Source{src("a", " 노을빛 검 ", "<b>霞光劍刃</b>", "", "")})
	if got := tbl.Names[0]; got.KR != "노을빛 검" || got.Variants[0].Renderings[0].ZH != "霞光劍刃" {
		t.Errorf("name = %+v", got)
	}
}

func TestCollectIsDeterministic(t *testing.T) {
	var units []Source
	for i := 0; i < 30; i++ {
		units = append(units, src(fmt.Sprintf("k%02d", i), fmt.Sprintf("이름 %d", i%4),
			fmt.Sprintf("譯%d", i%3), fmt.Sprintf("jp%d", i%2), "en"))
	}
	var first []byte
	for seed := int64(0); seed < 2; seed++ {
		shuffled := append([]Source(nil), units...)
		rand.New(rand.NewSource(seed+1)).Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		b, err := json.Marshal(Collect("r", nameFields, shuffled))
		if err != nil {
			t.Fatal(err)
		}
		if first == nil {
			first = b
		} else if string(first) != string(b) {
			t.Errorf("results differ:\n%s\n%s", first, b)
		}
	}
}

func TestTableRoundTrip(t *testing.T) {
	st := store.Open(t.TempDir())
	got, err := LoadTable(st)
	if got != nil || err != nil {
		t.Fatalf("LoadTable without file = %v, %v; want nil, nil", got, err)
	}
	tbl := Collect("raw1", nameFields, []Source{src("a", "노을빛 검", "霞光劍刃", "茜さす剣", "Glow Blade")})
	if err := SaveTable(st, tbl); err != nil {
		t.Fatal(err)
	}
	got, err = LoadTable(st)
	if err != nil || !reflect.DeepEqual(got, tbl) {
		t.Errorf("LoadTable = %+v, %v; want %+v", got, err, tbl)
	}
}

func TestNameHas(t *testing.T) {
	n := &Name{KR: "노을빛 검", Variants: []Variant{
		{Renderings: []Rendering{{"霞光劍刃", 2}}},
		{Renderings: []Rendering{{"夕陽", 1}}},
	}}
	for zh, want := range map[string]bool{"<b>霞光劍刃</b>揮出": true, "夕<b>陽</b>": true, "月光": false} {
		if got := n.Has(zh); got != want {
			t.Errorf("Has(%q) = %v, want %v", zh, got, want)
		}
	}
}

func TestCollectAndHasAgreeOnTags(t *testing.T) {
	zh := "霞光<br>劍刃"
	tbl := Collect("r", nameFields, []Source{src("a", "노을빛 검", zh, "", "")})
	n := tbl.Names[0]
	if got := n.Variants[0].Renderings[0].ZH; got != Plain(zh) {
		t.Errorf("stored %q, want %q", got, Plain(zh))
	}
	if !n.Has(zh) {
		t.Error("Has does not find the rendering it was built from")
	}
}
