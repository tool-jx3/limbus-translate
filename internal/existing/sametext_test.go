package existing

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"slices"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// unit is a unit of F/<key>.json (pointer /0) in group.
func unit(key, group, kr string) TextUnit {
	return TextUnit{Key: fkey(key), Group: group, KR: kr}
}

func fkey(key string) string { return units.Key("F/"+key+".json", "/0") }

// translations is a zhOf over a fixed map of keys (made with fkey) to
// translations.
func translations(zh map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		for k, v := range zh {
			if fkey(k) == key {
				return v, true
			}
		}
		return "", false
	}
}

func TestCollectSameTextNeedsTwoGroups(t *testing.T) {
	s := CollectSameText("raw1", []TextUnit{
		unit("1", "g1", "one"), unit("2", "g1", "one"), // one group
		unit("3", "g1", "two"), unit("4", "g2", "two"),
	})
	if len(s.Texts) != 1 {
		t.Fatalf("Texts = %v", s.Texts)
	}
	want := []SameGroup{{Group: "g1", Keys: []string{fkey("3")}}, {Group: "g2", Keys: []string{fkey("4")}}}
	if got := s.Texts["two"]; !reflect.DeepEqual(got, want) {
		t.Errorf("two = %v, want %v", got, want)
	}
	if s.RawTag != "raw1" {
		t.Errorf("RawTag = %q", s.RawTag)
	}
}

// The index holds the units, not their translations: a sentence that no unit
// has translated yet is kept all the same.
func TestCollectSameTextKeepsUntranslatedGroups(t *testing.T) {
	in := []TextUnit{
		{Key: fkey("9"), Group: "g2", Speaker: "philip", KR: "x"},
		unit("8", "g2", "alone"), unit("7", "g2", "alone"),
	}
	for i := 10 + MaxSameKeys + 4; i >= 10; i-- { // more keys than kept, in reverse order
		in = append(in, unit(fmt.Sprint(i), "g1", "x"))
	}
	s := CollectSameText("r", in)
	if _, ok := s.Texts["alone"]; ok {
		t.Errorf("a sentence of one group is kept: %v", s.Texts["alone"])
	}
	gs := s.Texts["x"]
	if len(gs) != 2 || gs[0].Group != "g1" || gs[1].Group != "g2" {
		t.Fatalf("x = %+v, want g1 and g2", gs)
	}
	if gs[1].Speaker != "philip" || !slices.Equal(gs[1].Keys, []string{fkey("9")}) {
		t.Errorf("g2 = %+v", gs[1])
	}
	keys := gs[0].Keys
	if len(keys) != MaxSameKeys || !slices.IsSorted(keys) {
		t.Fatalf("g1 keys = %v, want the first %d in key order", keys, MaxSameKeys)
	}
	var all []string
	for i := 10; i <= 10+MaxSameKeys+4; i++ {
		all = append(all, fkey(fmt.Sprint(i)))
	}
	slices.Sort(all)
	if !slices.Equal(keys, all[:MaxSameKeys]) {
		t.Errorf("g1 keys = %v, want %v", keys, all[:MaxSameKeys])
	}
}

// The translations For returns are those zhOf gives now.
func TestSameTextForReadsCurrentTranslations(t *testing.T) {
	s := CollectSameText("r", []TextUnit{
		unit("1", "g1", "x"),
		unit("2", "g2", "x"), unit("3", "g2", "x"), unit("4", "g2", "x"),
		{Key: fkey("5"), Group: "g3", Speaker: "philip", KR: "x"}, {Key: fkey("6"), Group: "g3", Speaker: "philip", KR: "x"},
		unit("7", "g4", "x"),
		unit("8", "g5", "x"),
	})
	if got := s.For("x", "g1", translations(nil)); got != nil {
		t.Errorf("For with no translation = %v, want nil", got)
	}
	zh := map[string]string{
		"1": "自己組",                   // the asking group: never returned
		"2": "乙", "3": "甲", "4": "甲", // most keys say 甲; File is that of the first of them
		"6": "丙", "5": "丁", // a tie: the first key wins
		// g4 has no translation
		"8": "戊",
	}
	got := s.For("x", "g1", translations(zh))
	want := []SameRendering{
		{Group: "g2", ZH: "甲", File: "F/3.json"},
		{Group: "g3", ZH: "丁", File: "F/5.json", Speaker: "philip"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("For = %+v, want %+v", got, want)
	}
	// Another translation now: the result follows.
	zh["2"], zh["5"] = "甲", "丙"
	got = s.For("x", "g1", translations(zh))
	if len(got) != MaxSameText || got[0].ZH != "甲" || got[0].File != "F/2.json" || got[1].ZH != "丙" || got[1].File != "F/5.json" {
		t.Errorf("For after the change = %+v", got)
	}
	// A ZH already given is not repeated; g4 has none; so g5 is the second.
	zh["5"], zh["6"] = "甲", "甲"
	got = s.For("x", "g1", translations(zh))
	if len(got) != 2 || got[0].Group != "g2" || got[1].Group != "g5" || got[1].ZH != "戊" {
		t.Errorf("For without repeats = %+v", got)
	}
	for _, r := range s.For("x", "g2", translations(zh)) {
		if r.Group == "g2" {
			t.Errorf("own group returned: %v", r)
		}
	}
	if s.For("missing", "g1", translations(zh)) != nil {
		t.Error("missing KR not nil")
	}
}

func TestUsable(t *testing.T) {
	const kr = "같은 문장입니다"
	ok := store.Record{Key: fkey("1"), KRSha: units.KRSha(kr), ZH: "同一句", Origin: store.OriginClaude}
	sha := units.KRSha(kr)
	for _, origin := range []string{store.OriginParatranz, store.OriginClaude, store.OriginHuman} {
		r := ok
		r.Origin = origin
		if !Usable(r, kr) || !UsableSHA(r, sha) {
			t.Errorf("origin %s: not usable (Usable %v, UsableSHA %v)", origin, Usable(r, kr), UsableSHA(r, sha))
		}
	}
	for name, edit := range map[string]func(*store.Record){
		"copy":         func(r *store.Record) { r.Origin = store.OriginCopy },
		"fallback":     func(r *store.Record) { r.Origin = store.OriginFallback },
		"stale KR":     func(r *store.Record) { r.KRSha = units.KRSha("예전 문장") },
		"needs review": func(r *store.Record) { r.Flags = []string{store.FlagNeedsReview} },
	} {
		r := ok
		edit(&r)
		if Usable(r, kr) || UsableSHA(r, sha) {
			t.Errorf("%s: usable (Usable %v, UsableSHA %v)", name, Usable(r, kr), UsableSHA(r, sha))
		}
	}
}

func TestSameTextRef(t *testing.T) {
	got := SameTextRef([]SameRendering{
		{File: "StoryData/S1.json", Speaker: "philip", ZH: "為了比誰都悲傷的我"},
		{File: "PersonalityVoiceDlg/V.json", ZH: "為了最悲傷的我"},
	})
	want := "StoryData/S1.json（philip）：為了比誰都悲傷的我\nPersonalityVoiceDlg/V.json：為了最悲傷的我"
	if got != want {
		t.Errorf("got %q", got)
	}
	if SameTextRef(nil) != "" {
		t.Error("empty not \"\"")
	}
}

func TestCollectSameTextIsDeterministic(t *testing.T) {
	in := []TextUnit{
		unit("1", "g1", "x"), unit("2", "g2", "x"), unit("3", "g1", "x"),
		unit("4", "g3", "y"), unit("5", "g2", "y"), unit("6", "g3", "x"),
	}
	rng := rand.New(rand.NewSource(1))
	var first []byte
	for i := 0; i < 20; i++ {
		u := append([]TextUnit(nil), in...)
		rng.Shuffle(len(u), func(a, b int) { u[a], u[b] = u[b], u[a] })
		b, err := json.Marshal(CollectSameText("r", u))
		if err != nil {
			t.Fatal(err)
		}
		if first == nil {
			first = b
		} else if string(first) != string(b) {
			t.Fatalf("results differ:\n%s\n%s", first, b)
		}
	}
}

func TestSameTextRoundTrip(t *testing.T) {
	st := store.Open(t.TempDir())
	got, err := LoadSameText(st)
	if got != nil || err != nil {
		t.Fatalf("Load without file = %v, %v", got, err)
	}
	s := CollectSameText("r", []TextUnit{
		unit("1", "g1", "x"), {Key: fkey("2"), Group: "g2", Speaker: "philip", KR: "x"},
	})
	if err := SaveSameText(st, s); err != nil {
		t.Fatal(err)
	}
	got, err = LoadSameText(st)
	if err != nil || !reflect.DeepEqual(got, s) {
		t.Fatalf("round trip = %v, %v", got, err)
	}
	var nilS *SameText
	if nilS.For("x", "g", translations(nil)) != nil {
		t.Error("nil receiver not nil")
	}
	b, _ := json.Marshal(CollectSameText("r", nil))
	if string(b) != `{"raw_tag":"r","texts":{}}` {
		t.Errorf("empty = %s", b)
	}
	b, _ = json.Marshal(s)
	want := `{"raw_tag":"r","texts":{"x":[{"group":"g1","keys":["F/1.json|/0"]},{"group":"g2","speaker":"philip","keys":["F/2.json|/0"]}]}}`
	if string(b) != want {
		t.Errorf("JSON = %s, want %s", b, want)
	}
}
