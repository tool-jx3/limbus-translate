package rules

import (
	"reflect"
	"slices"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/validate"
)

// referenceGlossaryHits is GlossaryHits without the rune prefilter: every
// approved term is looked for in the text.
func referenceGlossaryHits(r *Rules, file, kr string) []Term {
	kr = validate.MatchText(kr)
	var hits []Term
	scoped := map[string]bool{}
	for _, t := range r.Glossary {
		if t.Status != StatusApproved || t.KR == "" || !t.OccursIn(kr) {
			continue
		}
		if r.scopedTo(t, file) {
			scoped[t.KR] = true
		}
		if !r.Applies(t, file) {
			continue
		}
		hits = append(hits, t)
	}
	hits = slices.DeleteFunc(hits, func(t Term) bool { return len(t.Files) == 0 && scoped[t.KR] })
	return DropCovered(kr, hits)
}

func TestGlossaryHitsPrefilterEquivalent(t *testing.T) {
	r, err := Load("../..")
	if err != nil {
		t.Fatal(err)
	}
	// Texts made from the glossary itself: every term alone, with a tag
	// inside, with its first two runes and something else, and with only its
	// first rune; plus fixed oddities.
	texts := []string{
		"", "가", "[ID]", "<b>패키지</b>를 받았다", "패<i>키</i>지", "\xff\xfe", "패\xff", "시즌 8",
		"시 즌", "패 키지", "키패지", "이 패키지는 한정 판매입니다.", "<br/>시즌<br/>보상",
	}
	for _, term := range r.Glossary {
		if term.KR == "" {
			continue
		}
		rs := []rune(term.KR)
		texts = append(texts, term.KR, "앞 "+term.KR+" 뒤", string(rs[:1]))
		if len(rs) >= 2 {
			texts = append(texts, string(rs[:2])+"x", string(rs[1])+string(rs[0])) // not the word
		}
	}
	files := []string{"Other.json", "StoryData/Story_1.json", "EGOGift.json"}
	for _, f := range files {
		for _, text := range texts {
			got, want := r.GlossaryHits(f, text), referenceGlossaryHits(r, f, text)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("GlossaryHits(%q, %q) = %+v, reference %+v", f, text, got, want)
			}
		}
	}
}

func TestRuneSetMayStartTerm(t *testing.T) {
	set := NewRuneSet("시즌 보상")
	for kr, want := range map[string]bool{"시즌": true, "보": true, "즌 ": true, "상시": false, "코": false, "\xff": true} {
		if got := set.MayStartTerm(kr); got != want {
			t.Errorf("MayStartTerm(%q) = %v, want %v", kr, got, want)
		}
	}
}

func BenchmarkGlossaryHits(b *testing.B) {
	r, err := Load("../..")
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		r.GlossaryHits("Other.json", "이 패키지는 한정 판매입니다. 시즌 보상: 도약 모듈을 받았다.")
	}
}
