package align

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

func mustParse(t *testing.T, src string) *jsonx.Node {
	t.Helper()
	n, err := jsonx.Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// load parses testdata/align/<name>.
func load(t *testing.T, name string) *jsonx.Node {
	t.Helper()
	data, err := os.ReadFile("../../testdata/align/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return mustParse(t, string(data))
}

// wantAligned checks that the KR string at krPtr was matched by method to the
// string at otherPtr in other.
func wantAligned(t *testing.T, got map[string]LangAligned, krPtr string, other *jsonx.Node, otherPtr string, method Method) {
	t.Helper()
	n, ok := other.Get(otherPtr)
	if !ok || n.Kind != jsonx.String {
		t.Fatalf("fixture has no string at %s", otherPtr)
	}
	want := LangAligned{Value: n.Str, Method: method}
	if g, ok := got[krPtr]; !ok || g != want {
		t.Errorf("%s = %+v (present: %v), want %+v", krPtr, g, ok, want)
	}
}

func wantMissing(t *testing.T, got map[string]LangAligned, krPtr string) {
	t.Helper()
	if g, ok := got[krPtr]; ok {
		t.Errorf("%s = %+v, want it left out", krPtr, g)
	}
}

func TestAlignSameOrder(t *testing.T) {
	kr, jp := load(t, "same_order_kr.json"), load(t, "same_order_jp.json")
	// Identical item sequences keep their pointers, even in story files.
	for _, file := range []string{"Sample.json", "StoryData/S001A.json"} {
		got := AlignLang(file, kr, jp)
		us := units.Extract(file, kr)
		if len(got) != len(us) {
			t.Errorf("%s: %d aligned, want %d: %+v", file, len(got), len(us), got)
		}
		for _, u := range us {
			wantAligned(t, got, u.Pointer, jp, u.Pointer, MethodPointer)
		}
	}
}

func TestAlignReordered(t *testing.T) {
	kr, jp := load(t, "reordered_kr.json"), load(t, "reordered_jp.json")
	got := AlignLang("EventVerificationUIText.json", kr, jp)

	// JP lists ids a, c, b: KR's b (index 1) is JP's index 2.
	if v := got["/dataList/1/name"].Value; v != "検証イベントB" {
		t.Errorf(`"/dataList/1/name" = %q, want "検証イベントB"`, v)
	}
	wantAligned(t, got, "/dataList/0/name", jp, "/dataList/0/name", MethodID)
	wantAligned(t, got, "/dataList/1/name", jp, "/dataList/2/name", MethodID)
	wantAligned(t, got, "/dataList/2/name", jp, "/dataList/1/name", MethodID)
	wantAligned(t, got, "/dataList/1/id", jp, "/dataList/2/id", MethodID)
	if len(got) != 6 {
		t.Errorf("%d aligned, want 6: %+v", len(got), got)
	}
}

func TestAlignStoryIDTypo(t *testing.T) {
	kr, jp := load(t, "story_id_typo_kr.json"), load(t, "story_id_typo_jp.json")
	got := AlignLang("StoryData/E001X.json", kr, jp)

	// Line 2 is id 201 in KR but 211 in JP; position and speaker still agree.
	for i := range 3 {
		for _, field := range []string{"model", "content"} {
			p := fmt.Sprintf("/dataList/%d/%s", i, field)
			wantAligned(t, got, p, jp, p, MethodSequence)
		}
	}
	if len(got) != 6 {
		t.Errorf("%d aligned, want 6: %+v", len(got), got)
	}
}

func TestAlignExtraLine(t *testing.T) {
	kr, jp := load(t, "extra_line_kr.json"), load(t, "extra_line_jp.json")
	got := AlignLang("StoryData/P10511.json", kr, jp)

	// JP has an id-less line after line 1. It looks like KR's line 2 (both
	// are content-only), yet line 2 must skip it and pair with JP line 3.
	wantAligned(t, got, "/dataList/0/content", jp, "/dataList/0/content", MethodSequence)
	wantAligned(t, got, "/dataList/0/teller", jp, "/dataList/0/teller", MethodSequence)
	wantAligned(t, got, "/dataList/1/content", jp, "/dataList/2/content", MethodSequence)
	wantAligned(t, got, "/dataList/2/content", jp, "/dataList/3/content", MethodSequence)
	for p, a := range got {
		if a.Value == "（KRにはない一行）" {
			t.Errorf("%s was given JP's extra line", p)
		}
	}
}

func TestAlignModelMismatch(t *testing.T) {
	t.Run("sequence", func(t *testing.T) {
		kr, jp := load(t, "model_mismatch_kr.json"), load(t, "model_mismatch_jp.json")
		got := AlignLang("StoryData/S100A.json", kr, jp)

		// Line 2 is spoken by 돈키호테 in KR but 파우스트 in JP.
		wantMissing(t, got, "/dataList/1/content")
		wantMissing(t, got, "/dataList/1/model")
		wantAligned(t, got, "/dataList/0/content", jp, "/dataList/0/content", MethodSequence)
		wantAligned(t, got, "/dataList/2/content", jp, "/dataList/2/content", MethodSequence)
	})
	t.Run("id", func(t *testing.T) {
		kr := mustParse(t, `{"dataList": [
			{"id": "a", "model": "x", "name": "가"},
			{"id": "b", "model": "y", "name": "나"}]}`)
		jp := mustParse(t, `{"dataList": [
			{"id": "b", "model": "z", "name": "ナ"},
			{"id": "a", "model": "x", "name": "ア"}]}`)
		got := AlignLang("Sample.json", kr, jp)

		wantMissing(t, got, "/dataList/1/name")
		wantMissing(t, got, "/dataList/1/id")
		wantAligned(t, got, "/dataList/0/name", jp, "/dataList/1/name", MethodID)
	})
}

func TestAlignNoDataList(t *testing.T) {
	got := AlignLang("Sample.json", mustParse(t, `{"a": "x", "b": "z"}`), mustParse(t, `{"a": "y"}`))
	want := map[string]LangAligned{"/a": {Value: "y", Method: MethodPointer}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("AlignLang = %+v, want %+v", got, want)
	}
}

func TestAlignOutsideDataList(t *testing.T) {
	kr := mustParse(t, `{"title": "제목", "dataList": [{"id": "a", "name": "가"}, {"id": "b", "name": "나"}]}`)
	jp := mustParse(t, `{"title": "タイトル", "dataList": [{"id": "b", "name": "ナ"}, {"id": "a", "name": "ア"}]}`)
	got := AlignLang("Sample.json", kr, jp)

	wantAligned(t, got, "/title", jp, "/title", MethodPointer)
	wantAligned(t, got, "/dataList/0/name", jp, "/dataList/1/name", MethodID)
}

// idDoc builds {"dataList": [{"id": id, "name": "<lang> <id>"}, ...]}.
func idDoc(t *testing.T, lang string, ids ...string) *jsonx.Node {
	t.Helper()
	items := make([]string, len(ids))
	for i, id := range ids {
		items[i] = fmt.Sprintf(`{"id": %q, "name": %q}`, id, lang+" "+id)
	}
	return mustParse(t, `{"dataList": [`+strings.Join(items, ", ")+`]}`)
}

func TestAlignIDThreshold(t *testing.T) {
	kr := idDoc(t, "kr", "k0", "k1", "k2", "k3", "k4", "k5", "k6", "k7", "k8", "k9")

	t.Run("90% found", func(t *testing.T) {
		jp := idDoc(t, "jp", "x", "k8", "k7", "k6", "k5", "k4", "k3", "k2", "k1", "k0")
		got := AlignLang("Sample.json", kr, jp)
		wantAligned(t, got, "/dataList/0/name", jp, "/dataList/9/name", MethodID)
		wantAligned(t, got, "/dataList/8/name", jp, "/dataList/1/name", MethodID)
		wantMissing(t, got, "/dataList/9/name")
	})
	t.Run("80% found", func(t *testing.T) {
		jp := idDoc(t, "jp", "k0", "k1", "k2", "k3", "k4", "k5", "k6", "k7", "x8", "x9")
		got := AlignLang("Sample.json", kr, jp)
		wantAligned(t, got, "/dataList/0/name", jp, "/dataList/0/name", MethodSequence)
		wantAligned(t, got, "/dataList/8/name", jp, "/dataList/8/name", MethodSequence)
	})
	t.Run("duplicate ids", func(t *testing.T) {
		kr := idDoc(t, "kr", "k0", "k0", "k1")
		jp := idDoc(t, "jp", "k1", "k0", "k0")
		got := AlignLang("Sample.json", kr, jp)
		wantAligned(t, got, "/dataList/0/name", jp, "/dataList/0/name", MethodSequence)
	})
	t.Run("story files ignore ids", func(t *testing.T) {
		kr := idDoc(t, "kr", "k0", "k1", "k2")
		jp := idDoc(t, "jp", "k0", "k2", "k1")
		got := AlignLang("StoryData/S001A.json", kr, jp)
		wantAligned(t, got, "/dataList/1/name", jp, "/dataList/1/name", MethodSequence)
	})
}

func TestAlignTieBreakKeepsPairCount(t *testing.T) {
	// Content-only lines with KR ids 0..n-1 and JP ids 1..n. Shifting by one
	// would match n-1 ids, but the id tie-break must never cost a pair: all n
	// lines pair by position.
	const n = 5
	lines := func(lang string, firstID int) *jsonx.Node {
		items := make([]string, n)
		for i := range items {
			items[i] = fmt.Sprintf(`{"id": %d, "content": "%s line %d"}`, firstID+i, lang, i)
		}
		return mustParse(t, `{"dataList": [`+strings.Join(items, ", ")+`]}`)
	}
	kr, jp := lines("kr", 0), lines("jp", 1)
	got := AlignLang("StoryData/S001A.json", kr, jp)

	for i := range n {
		p := fmt.Sprintf("/dataList/%d/content", i)
		wantAligned(t, got, p, jp, p, MethodSequence)
	}
	if len(got) != n {
		t.Errorf("%d aligned, want %d: %+v", len(got), n, got)
	}
}

func TestAlignNested(t *testing.T) {
	kr := mustParse(t, `{"dataList": [
		{"id": "burn", "levelList": [{"id": 1, "desc": "화상 1"}, {"id": 2, "desc": "화상 2"}],
		 "coinList": [{"desc": "동전 1"}, {"desc": "동전 2"}], "tags": ["가", "나"]},
		{"id": "bleed", "name": "출혈"}]}`)
	jp := mustParse(t, `{"dataList": [
		{"id": "bleed", "name": "出血"},
		{"id": "burn", "levelList": [{"id": 2, "desc": "火傷2"}, {"id": 1, "desc": "火傷1"}],
		 "coinList": [{"desc": "コイン1"}, {"desc": "コイン2"}], "tags": ["ア", "イ"]}]}`)
	got := AlignLang("Bufs.json", kr, jp)

	// levelList elements have ids, so they pair by id; the others by position.
	wantAligned(t, got, "/dataList/0/levelList/0/desc", jp, "/dataList/1/levelList/1/desc", MethodID)
	wantAligned(t, got, "/dataList/0/levelList/1/desc", jp, "/dataList/1/levelList/0/desc", MethodID)
	wantAligned(t, got, "/dataList/0/coinList/1/desc", jp, "/dataList/1/coinList/1/desc", MethodID)
	wantAligned(t, got, "/dataList/0/tags/1", jp, "/dataList/1/tags/1", MethodID)
	wantAligned(t, got, "/dataList/1/name", jp, "/dataList/0/name", MethodID)
}

func TestLCS(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 0},
		{"abc", "abc", 3},
		{"abc", "xyz", 0},
		{"abcbdab", "bdcaba", 4},
		{"aXbcY", "abZc", 3},
	}
	for _, c := range cases {
		a, b := []rune(c.a), []rune(c.b)
		pairs := LCS(a, b)
		if len(pairs) != c.want {
			t.Errorf("LCS(%q, %q) has %d pairs, want %d: %v", c.a, c.b, len(pairs), c.want, pairs)
		}
		for k, p := range pairs {
			if a[p[0]] != b[p[1]] {
				t.Errorf("LCS(%q, %q): pair %v joins %q and %q", c.a, c.b, p, a[p[0]], b[p[1]])
			}
			if k > 0 && (p[0] <= pairs[k-1][0] || p[1] <= pairs[k-1][1]) {
				t.Errorf("LCS(%q, %q): pairs not increasing: %v", c.a, c.b, pairs)
			}
		}
	}
	// Ties go to the earliest pairs.
	if got, want := LCS([]string{"a", "a"}, []string{"a"}), [][2]int{{0, 0}}; !reflect.DeepEqual(got, want) {
		t.Errorf("LCS([a a], [a]) = %v, want %v", got, want)
	}
}

// Batches write an aligned string to their JSON file, so its field names are
// part of the batch schema.
func TestLangAlignedJSON(t *testing.T) {
	data, err := json.Marshal(LangAligned{Value: "Inflict <b>", Method: MethodID})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"value":"Inflict \u003cb\u003e","method":"id"}`; string(data) != want {
		t.Errorf("JSON = %s, want %s", data, want)
	}
	var back LangAligned
	if err := json.Unmarshal(data, &back); err != nil || back != (LangAligned{Value: "Inflict <b>", Method: MethodID}) {
		t.Errorf("round trip = %+v, %v", back, err)
	}
}
