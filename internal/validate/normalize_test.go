package validate

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"space before and after", "增加5級[Laceration] 強度", "增加5級 [Laceration] 強度"},
		{"no space on either side", "命中[Sinking]強度", "命中 [Sinking] 強度"},
		{"adjacent IDs", "[Burn][Bleed]", "[Burn] [Bleed]"},
		{"all-caps variable", "擁有[ATTRIBUTES]屬性攻擊技能的", "擁有[ATTRIBUTES]屬性攻擊技能的"},
		{"TabExplain loses the space before it", "效果 [TabExplain]說明", "效果[TabExplain]說明"},
		{"fullwidth punctuation needs no space", "，[Burn]。", "，[Burn]。"},
		{"escaped angle brackets", "&lt;color=#fff&gt;A&lt;/color&gt;", "<color=#fff>A</color>"},

		{"ID at the start and end", "[Burn]", "[Burn]"},
		{"ID at the start", "[Burn]增加", "[Burn] 增加"},
		{"ID at the end", "增加[Burn]", "增加 [Burn]"},
		{"existing spaces are kept", "增加 [Burn] 強度", "增加 [Burn] 強度"},
		{"newline counts as a space", "增加\n[Burn]\n強度", "增加\n[Burn]\n強度"},
		{"tags need no space", "<color=#f00>[Burn]</color>", "<color=#f00>[Burn]</color>"},
		{"closing parenthesis needs no space after", "(造成[Burn])", "(造成 [Burn])"},
		{"fullwidth brackets and quotes", "「[Burn]」『[Bleed]』（[Sinking]）", "「[Burn]」『[Bleed]』（[Sinking]）"},
		{"ellipsis and dashes", "……[Burn]——[Bleed]～", "……[Burn]——[Bleed]～"},
		{"three IDs in a row", "[Burn][Bleed][Sinking]", "[Burn] [Bleed] [Sinking]"},
		{"ID with digits and underscore", "給予[Defense_Up2]吧", "給予 [Defense_Up2] 吧"},
		{"all-caps variable next to an ID", "[ATTRIBUTES][Burn]", "[ATTRIBUTES] [Burn]"},
		{"ID next to an all-caps variable", "[Burn][ATTRIBUTES]", "[Burn] [ATTRIBUTES]"},
		{"all-caps with digits", "擁有[SKILL_2]的", "擁有[SKILL_2]的"},
		{"single capital letter is not a variable", "擁有[A]的", "擁有 [A] 的"},
		{"mixed case is not a variable", "擁有[Atk]的", "擁有 [Atk] 的"},
		{"TabExplain after an ID stays attached", "[Burn][TabExplain]", "[Burn][TabExplain]"},
		{"TabExplain with several spaces", "效果   [TabExplain] 說明", "效果[TabExplain] 說明"},
		{"TabExplain loses a tab or ideographic space before it", "效果\t[TabExplain]說明　[TabExplain]", "效果[TabExplain]說明[TabExplain]"},
		{"TabExplain keeps a newline before it", "效果\n[TabExplain]說明", "效果\n[TabExplain]說明"},
		{"TabExplain gets no space after it", "效果[TabExplain]說明", "效果[TabExplain]說明"},
		{"ID after TabExplain", "[TabExplain][Burn]", "[TabExplain] [Burn]"},
		{"brackets that are not IDs are left alone", "第[1]項[2]", "第[1]項[2]"},
		{"unescape only touches lt and gt", "&amp;&lt;", "&amp;<"},
		{"empty", "", ""},
		{"plain text", "沒有任何標記的句子。", "沒有任何標記的句子。"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Normalize("", tc.in)
			if got != tc.want {
				t.Errorf("Normalize(%q) = %q, want %q", tc.in, got, tc.want)
			}
			if again := Normalize("", got); again != got {
				t.Errorf("Normalize is not idempotent: %q -> %q", got, again)
			}
		})
	}
}

// Normalize repairs only the translation, so the KR text must not change it.
func TestNormalizeIgnoresKR(t *testing.T) {
	zh := "命中[Sinking]強度"
	if a, b := Normalize("", zh), Normalize("명중 [Sinking] 위력", zh); a != b {
		t.Errorf("Normalize depends on kr: %q vs %q", a, b)
	}
}
