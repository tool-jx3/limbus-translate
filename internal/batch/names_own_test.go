package batch

import (
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/existing"
)

// The name of a batch says when its rendering is only this project's own.
func TestNameRefsCarryOwn(t *testing.T) {
	own := &existing.Name{KR: "밀랍날개", Variants: []existing.Variant{{JP: "蜜蝋の翼", Renderings: []existing.Rendering{{ZH: "蜜蠟之翼", Units: 4, Own: true}}}}}
	old := &existing.Name{KR: "거미집", Variants: []existing.Variant{{JP: "蜘蛛の巣", Renderings: []existing.Rendering{{ZH: "蜘蛛巢", Units: 11}, {ZH: "蛛網", Units: 1, Own: true}}}}}
	refs := nameRefs([]nameUse{{name: own, key: "k1"}, {name: old, key: "k2"}}, 10)
	if len(refs) != 2 {
		t.Fatalf("refs = %+v", refs)
	}
	for _, r := range refs {
		switch r.KR {
		case "밀랍날개":
			if !r.Own || r.ZH != "蜜蠟之翼" {
				t.Errorf("own name = %+v", r)
			}
		case "거미집":
			if r.Own || r.ZH != "蜘蛛巢" || len(r.Others) != 1 || !r.Others[0].Own {
				t.Errorf("old name = %+v", r)
			}
		}
	}
}

// The record of a unit translated again because its KR changed keeps the KR
// text and the translation it had before, as the batch showed them; the
// record of a new unit has neither.
func TestSubmitKeepsThePreviousTextOfAChangedUnit(t *testing.T) {
	kr := map[string]string{"Other.json": dataList(t, 2, "", line)}
	env := newEnv(t, kr, nil, nil)
	pending := items("Other.json", 0, 2)
	pending[0].Reason, pending[0].PrevKR, pending[0].PrevZH = "kr-changed", "옛 원문\n둘째 줄", "舊譯\n第二行"
	seed(t, env, pending...)
	b := mustNext(t, env, NextOptions{})
	if _, err := Submit(env, b, translate(b), "m", 3); err != nil {
		t.Fatal(err)
	}
	recs := records(t, env)
	if r := recs[key("Other.json", 0)]; r.PrevKR != "옛 원문\n둘째 줄" || r.PrevZH != "舊譯\n第二行" {
		t.Errorf("record of the changed unit = %+v, want its previous text", r)
	}
	if r := recs[key("Other.json", 1)]; r.PrevKR != "" || r.PrevZH != "" {
		t.Errorf("record of the new unit = %+v, want no previous text", r)
	}
}
