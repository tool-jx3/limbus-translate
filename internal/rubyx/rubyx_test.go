package rubyx

import "testing"

func TestConvertMatchesInGameTests(t *testing.T) {
	st := Style{Size: "50%", Scale: 0.5, VOffset: "1em"}
	// Sarasa Gothic TC Bold 的 advance（/1000）：漢字、假名 1em；英文字母照字型。
	latin := map[rune]float64{' ': .237, 'A': .747, 'E': .607, 'F': .587, 'G': .760, 'I': .281, 'K': .719, 'N': .762, 'R': .657, 'S': .655, 'U': .732, 'Z': .664}
	adv := func(r rune) float64 {
		if w, ok := latin[r]; ok {
			return w
		}
		return 1
	}
	cases := map[string]string{
		"<rubycustomize=ヨシヒデ>良秀</rubycustomize>":                                      "<nobr>良秀<space=-2em><voffset=1em><size=50%>ヨシヒデ</size></voffset><space=0em></nobr>",
		"<rubycustomize=UNGEZIEFER KAISER>害蟲皇帝</rubycustomize>":                       "<nobr>害蟲皇帝<space=-4.542em><voffset=1em><size=50%>UNGEZIEFER KAISER</size></voffset><space=-0.542em></nobr>",
		"就算<rubycustomize=\"ヨシヒデ\">良秀</rubycustomize>姊姊":                              "就算<nobr>良秀<space=-2em><voffset=1em><size=50%>ヨシヒデ</size></voffset><space=0em></nobr>姊姊",
		"<rubycustomize=李箱>異常</rubycustomize>與<rubycustomize=ヨシヒデ>良秀</rubycustomize>": "<nobr>異常<space=-1.5em><voffset=1em><size=50%>李箱</size></voffset><space=0.5em></nobr>與<nobr>良秀<space=-2em><voffset=1em><size=50%>ヨシヒデ</size></voffset><space=0em></nobr>",
	}
	for in, want := range cases {
		if got := Convert(in, st, adv); got != want {
			t.Errorf("Convert(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestProblems(t *testing.T) {
	ok := []string{"<rubycustomize=ヨシヒデ>良秀</rubycustomize>", "<rubycustomize=\"A B\">甲</rubycustomize>乙", "沒有標籤"}
	bad := []string{
		"<rubycustomize>良秀</rubycustomize>",                                   // 沒有 =
		"<rubycustomize=>良秀</rubycustomize>",                                  // 空的注音
		"<rubycustomize=ヨシヒデ></rubycustomize>",                                // 空的本文
		"<rubycustomize=ヨシヒデ>良<b>秀</b></rubycustomize>",                       // 本文有標籤
		"<rubycustomize=ヨシ\nヒデ>良秀</rubycustomize>",                            // 換行
		"<rubycustomize=ヨシヒデ>良\n秀</rubycustomize>",                            // 本文換行
		"<rubycustomize=ヨシヒデ>良秀",                                              // 沒有閉合
		"良秀</rubycustomize>",                                                  // 多出的閉合
		"<RubyCustomize=ヨシヒデ>良秀</RubyCustomize>",                              // 大寫
		"<rubycustomize=甲><rubycustomize=乙>丙</rubycustomize></rubycustomize>", // 巢狀
	}
	for _, s := range ok {
		if p := Problems(s); p != nil {
			t.Errorf("Problems(%q) = %v, want none", s, p)
		}
	}
	for _, s := range bad {
		if Problems(s) == nil {
			t.Errorf("Problems(%q) = none, want a problem", s)
		}
	}
}

func TestStripAndRestore(t *testing.T) {
	src := "就算<rubycustomize=ヨシヒデ>良秀</rubycustomize>姊姊"
	if got := Strip(src); got != "就算良秀姊姊" {
		t.Errorf("Strip = %q", got)
	}
	conv := Convert(src, Style{"50%", 0.5, "1em"}, func(rune) float64 { return 1 })
	if got := Restore(conv); got != src {
		t.Errorf("Restore(Convert(x)) = %q, want %q", got, src)
	}
	// 不是完整結構的 nobr、space 不動
	for _, s := range []string{"<nobr>甲乙</nobr>", "甲<space=1em>乙", "<nobr>甲<space=-1em><voffset=1em>乙</voffset></nobr>"} {
		if got := Restore(s); got != s {
			t.Errorf("Restore(%q) = %q, want unchanged", s, got)
		}
	}
}

func TestParseStyle(t *testing.T) {
	st, err := ParseStyle("50%", "1em")
	if err != nil || st != (Style{"50%", 0.5, "1em"}) {
		t.Fatalf("ParseStyle = %v, %v", st, err)
	}
	for _, bad := range [][2]string{{"50", "1em"}, {"0%", "1em"}, {"150%", "1em"}, {"50%", "1px"}, {"50%", "em"}} {
		if _, err := ParseStyle(bad[0], bad[1]); err == nil {
			t.Errorf("ParseStyle(%q, %q) accepted", bad[0], bad[1])
		}
	}
}
