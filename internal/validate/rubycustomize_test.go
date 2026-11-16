package validate

import "testing"

func hasHard(issues []Issue, code string) bool {
	for _, i := range issues {
		if i.Code == code && i.Severity == SeverityHard {
			return true
		}
	}
	return false
}

func TestCheckRubyCustomize(t *testing.T) {
	kr, ctx := "요시히데", Context{File: "StoryData/P10916.json", Pointer: "/dataList/3/teller", RubyCustomize: true}
	if is := Check(kr, "<rubycustomize=ヨシヒデ>良秀</rubycustomize>", ctx); len(is) != 0 {
		t.Errorf("allowed: %v", is)
	}
	// 本文仍受詞彙表約束：注音裡有「良秀」不算
	ctx.Terms = []Term{{KR: "요시히데", ZH: "良秀"}}
	if is := Check(kr, "<rubycustomize=良秀>本名</rubycustomize>", ctx); !hasCode(is, CodeTermMissing) {
		t.Errorf("term in the reading only must fail: %v", is)
	}
	ctx.Terms = nil
	if is := Check(kr, "<rubycustomize=>良秀</rubycustomize>", ctx); !hasHard(is, CodeRubyCustomize) {
		t.Errorf("syntax: %v", is)
	}
	ctx.RubyCustomize = false
	if is := Check(kr, "<rubycustomize=ヨシヒデ>良秀</rubycustomize>", ctx); !hasHard(is, CodeRubyCustomize) {
		t.Errorf("place: %v", is)
	}
	// 注音是假名，不觸發殘留假名的檢查
	ctx.RubyCustomize = true
	if is := Check("그레고르", "<rubycustomize=UNGEZIEFER KAISER>害蟲皇帝</rubycustomize>", ctx); len(is) != 0 {
		t.Errorf("reading checked as text: %v", is)
	}
}

func TestRPGDialogueTakesNoRubyKRLacks(t *testing.T) {
	is := Check("나무", "<ruby=Sephirot>生命之樹</ruby>", Context{File: "RPGSystem/rpg-loc-dialogue-common-a1c10p2.json"})
	if !hasCode(is, CodeTagMismatch) {
		t.Errorf("RPG dialogue accepted a ruby KR lacks: %v", is)
	}
	if is := Check("나무", "<ruby=Sephirot>生命之樹</ruby>", Context{File: "StoryData/S945B.json"}); hasCode(is, CodeTagMismatch) {
		t.Errorf("story content: %v", is)
	}
}
