package existing

import (
	"encoding/json"
	"math/rand"
	"reflect"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/store"
)

func unit(key, group, kr, zh string) TextUnit {
	return TextUnit{Key: key, Group: group, File: "F/" + key + ".json", KR: kr, ZH: zh}
}

func TestCollectSameTextNeedsTwoGroups(t *testing.T) {
	s := CollectSameText("raw1", []TextUnit{
		unit("1", "g1", "one", "甲"), unit("2", "g1", "one", "乙"), // one group
		unit("3", "g1", "two", "丙"), unit("4", "g2", "two", ""), // one translated
		unit("5", "g1", "three", ""), unit("6", "g2", "three", ""), // none translated
	})
	if len(s.Texts) != 1 {
		t.Fatalf("Texts = %v", s.Texts)
	}
	got := s.Texts["two"]
	want := []SameRendering{{Group: "g1", ZH: "丙", File: "F/3.json"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("two = %v, want %v", got, want)
	}
	if s.RawTag != "raw1" {
		t.Errorf("RawTag = %q", s.RawTag)
	}
}

func TestCollectSameTextMainRendering(t *testing.T) {
	s := CollectSameText("r", []TextUnit{
		unit("1", "g1", "x", "B"), unit("2", "g1", "x", "A"), unit("3", "g1", "x", "A"),
		unit("4", "g2", "x", "C"),
	})
	rs := s.Texts["x"]
	if len(rs) != 2 || rs[0].ZH != "A" || rs[0].File != "F/2.json" {
		t.Errorf("rs = %v", rs)
	}
}

func TestCollectSameTextTieFirstKey(t *testing.T) {
	s := CollectSameText("r", []TextUnit{
		unit("2", "g1", "x", "A"), unit("1", "g1", "x", "B"), unit("3", "g2", "x", "C"),
	})
	if rs := s.Texts["x"]; rs[0].ZH != "B" {
		t.Errorf("rs = %v", rs)
	}
}

func threeGroups() *SameText {
	return CollectSameText("r", []TextUnit{
		unit("1", "g1", "x", "A"), unit("2", "g2", "x", "A"), unit("3", "g3", "x", "C"),
	})
}

func TestSameTextForOtherGroups(t *testing.T) {
	s := threeGroups()
	got := s.For("x", "g1")
	if len(got) != 2 || got[0].ZH != "A" || got[0].Group != "g2" || got[1].ZH != "C" {
		t.Errorf("For = %v", got)
	}
	for _, r := range s.For("x", "g2") {
		if r.Group == "g2" {
			t.Errorf("own group returned: %v", r)
		}
	}
	if s.For("missing", "g1") != nil {
		t.Error("missing KR not nil")
	}
}

func TestSameTextForAtMostTwo(t *testing.T) {
	s := CollectSameText("r", []TextUnit{
		unit("1", "g1", "x", "A"), unit("2", "g2", "x", "B"), unit("3", "g3", "x", "C"), unit("4", "g4", "x", "D"),
	})
	if got := s.For("x", "g1"); len(got) != MaxSameText {
		t.Errorf("For = %v", got)
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
	units := []TextUnit{
		unit("1", "g1", "x", "A"), unit("2", "g2", "x", "B"), unit("3", "g1", "x", "B"),
		unit("4", "g3", "y", "C"), unit("5", "g2", "y", "D"), unit("6", "g3", "x", "A"),
	}
	rng := rand.New(rand.NewSource(1))
	var first []byte
	for i := 0; i < 20; i++ {
		u := append([]TextUnit(nil), units...)
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
	s := threeGroups()
	if err := SaveSameText(st, s); err != nil {
		t.Fatal(err)
	}
	got, err = LoadSameText(st)
	if err != nil || !reflect.DeepEqual(got, s) {
		t.Fatalf("round trip = %v, %v", got, err)
	}
	var nilS *SameText
	if nilS.For("x", "g") != nil {
		t.Error("nil receiver not nil")
	}
	b, _ := json.Marshal(CollectSameText("r", nil))
	if string(b) != `{"raw_tag":"r","texts":{}}` {
		t.Errorf("empty = %s", b)
	}
}
