package existing

import (
	"math/rand"
	"reflect"
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/validate"
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

// A table written before Collect left out the names without a letter still
// has them; the index leaves them out, as Collect now does.
func TestIndexExcludesNamesWithoutALetter(t *testing.T) {
	ix := NewIndex(tableOf("...", "<b>!?</b>", "노을빛 검", "...검"), &rules.Rules{}, nil)
	if ix.Lookup("...") != nil || ix.Lookup("<b>!?</b>") != nil || ix.IsName("name", "...") {
		t.Error("a name without a letter is found")
	}
	if ix.Lookup("...검") == nil {
		t.Error("a name with a letter among the punctuation is not found")
	}
	if got := krs(ix.Mentions("그래서... 노을빛 검으로 '!?' 벤다", map[string]bool{"...": true, "<b>!?</b>": true})); !reflect.DeepEqual(got, []string{"노을빛 검"}) {
		t.Errorf("Mentions = %v, want only 노을빛 검", got)
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

// mentionsLinear is Mentions as it was before the index sorted its names by
// their first rune: every name of the index is looked at. The tests hold
// Mentions to it.
func mentionsLinear(ix *Index, kr string, always map[string]bool) []*Name {
	if ix == nil || len(ix.order) == 0 {
		return nil
	}
	plain := validate.MatchText(kr)
	set := rules.NewRuneSet(plain)
	var hits []entry
	var terms []rules.Term
	for _, e := range ix.order {
		if !set.MayStartTerm(e.plain) {
			continue
		}
		term := rules.Term{KR: e.plain, Match: rules.MatchWord}
		if !term.OccursIn(plain) {
			continue
		}
		if e.runes < 3 && !strings.ContainsFunc(e.plain, unicode.IsSpace) &&
			!quoted(plain, e.plain) && !always[e.n.KR] && !always[e.plain] {
			continue
		}
		hits = append(hits, e)
		terms = append(terms, term)
	}
	if len(hits) == 0 {
		return nil
	}
	kept := map[string]bool{}
	for _, t := range rules.DropCovered(plain, terms) {
		kept[t.KR] = true
	}
	var out []*Name
	for _, e := range hits {
		if kept[e.plain] {
			out = append(out, e.n)
		}
	}
	return out
}

// mixedTable has names of every kind Mentions treats apart: short and long,
// with spaces, tags and a no-break space, ones that hold others, one twice
// (by its tags), and Latin and punctuation.
func mixedTable() *Table {
	return tableOf("노을빛 검", "<b>노을빛 검</b>", "회피", "가르는 검", "새벽을 가르는 검", "검",
		"E.G.O 개화 (정오)", "E.G.O", "가르 검", "<color=#fff>출혈</color>", "출혈", "화상 3", "�깨진", "깨진")
}

var mixedTexts = []string{
	"노을빛 검으로 벤다", "붉은노을빛 검", "회피를 한다", "'회피' 스킬", "「출혈」 부여", `"화상" 3`,
	"새벽을 가르는 검과 가르는 검", "그 E.G.O 개화 (정오) 발동", "E.G.O 발동", "그 가르 검 발동",
	"<b>노을빛</b> 검", "<sprite name=x>회피", "출혈 화상 3 회피 검", "�깨진 것", "깨진 \xff것", "",
	"‘가르는 검’ 노을빛 검 노을빛 검으로",
}

// The names Mentions looks at in a text are those whose KR starts with a rune
// of the text, in the order of the index; one that starts with no valid rune
// is looked at in every text.
func TestMentionsLooksAtNamesStartingWithARuneOfTheText(t *testing.T) {
	ix := NewIndex(tableOf("노을빛 검", "가르는 검", "새벽을 가르는 검", "<b>회피</b>", "E.G.O 개화", "\xff깨진"), &rules.Rules{}, nil)
	var got []string
	for _, i := range ix.candidates(validate.MatchText("새벽 가르기 E")) {
		got = append(got, ix.order[i].n.KR)
	}
	if want := []string{"새벽을 가르는 검", "E.G.O 개화", "가르는 검", "\xff깨진"}; !slices.Equal(got, want) {
		t.Errorf("candidates = %q, want %q", got, want)
	}
}

// Mentions finds what looking at every name finds, in the same order, with
// always and without.
func TestMentionsMatchesEveryNameLookedAt(t *testing.T) {
	ix := NewIndex(mixedTable(), &rules.Rules{}, nil)
	for _, always := range []map[string]bool{nil, {"회피": true, "출혈": true, "<color=#fff>출혈</color>": true, "깨진": true}} {
		for _, text := range mixedTexts {
			if got, want := ix.Mentions(text, always), mentionsLinear(ix, text, always); !slices.Equal(got, want) {
				t.Errorf("Mentions(%q, %v) = %v, want %v", text, always, krs(got), krs(want))
			}
		}
	}
}

// randomTable is n names made of a few syllables, Latin letters, spaces,
// quotes, tags and invalid UTF-8, so that they share first runes, hold one
// another and stand in quotes; and texts of them.
func randomTable(rng *rand.Rand, n, texts int) (*Table, []string) {
	pieces := []string{"가", "나", "다", "라", "검", "빛", " ", "A", "b", ".", "'", "「", "」", "<b>", "</b>", "<br>", " ", "\xff", "�"}
	word := func(max int) string {
		var b strings.Builder
		for range 1 + rng.Intn(max) {
			b.WriteString(pieces[rng.Intn(len(pieces))])
		}
		return b.String()
	}
	var names []string
	for range n {
		names = append(names, word(5))
	}
	var out []string
	for range texts {
		var b strings.Builder
		for range 1 + rng.Intn(6) {
			if rng.Intn(2) == 0 {
				b.WriteString(names[rng.Intn(len(names))])
			} else {
				b.WriteString(word(4))
			}
		}
		out = append(out, b.String())
	}
	return tableOf(names...), out
}

func TestMentionsMatchesEveryNameLookedAtOnRandomTables(t *testing.T) {
	for seed := range int64(20) {
		rng := rand.New(rand.NewSource(seed))
		tbl, texts := randomTable(rng, 300, 200)
		ix := NewIndex(tbl, &rules.Rules{}, nil)
		always := map[string]bool{}
		for _, nm := range tbl.Names {
			if rng.Intn(4) == 0 {
				always[nm.KR] = true
			}
		}
		for _, text := range texts {
			if got, want := ix.Mentions(text, always), mentionsLinear(ix, text, always); !slices.Equal(got, want) {
				t.Fatalf("seed %d: Mentions(%q) = %q, want %q", seed, text, krs(got), krs(want))
			}
		}
	}
}

// BenchmarkMentions looks for 5,000 names of two to six syllables in texts
// of a few dozen.
func BenchmarkMentions(b *testing.B) {
	rng := rand.New(rand.NewSource(1))
	syllable := func() string { return string(rune(0xAC00 + rng.Intn(400))) }
	var names []string
	for range 5000 {
		var w strings.Builder
		for range 2 + rng.Intn(5) {
			w.WriteString(syllable())
		}
		if rng.Intn(3) == 0 {
			w.WriteString(" " + syllable() + syllable())
		}
		names = append(names, w.String())
	}
	var texts []string
	for range 100 {
		var w strings.Builder
		for range 30 + rng.Intn(30) {
			if rng.Intn(10) == 0 {
				w.WriteString(names[rng.Intn(len(names))])
			} else {
				w.WriteString(syllable())
			}
			if rng.Intn(4) == 0 {
				w.WriteString(" ")
			}
		}
		texts = append(texts, w.String())
	}
	ix := NewIndex(tableOf(names...), &rules.Rules{}, nil)
	b.ResetTimer()
	for i := range b.N {
		ix.Mentions(texts[i%len(texts)], nil)
	}
}
