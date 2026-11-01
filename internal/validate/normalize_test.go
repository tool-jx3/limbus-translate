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
		{"parentheses around an ID need no space", "獲得([Binding] 層數÷2)", "獲得([Binding] 層數÷2)"},
		{"space after the text, none before the closing parenthesis", "(最多[Burn])", "(最多 [Burn])"},
		{"parentheses hugging an ID", "獲得([Binding])", "獲得([Binding])"},
		{"opening parenthesis does not exempt the ID's other side", "獲得([Binding]層數)", "獲得([Binding] 層數)"},
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
		{"mirrored name keeps its brackets", "射影戰鬥][鬥戰影射", "射影戰鬥][鬥戰影射"},
		{"mirrored name in a sentence", "事件「射影戰鬥][鬥戰影射」開始", "事件「射影戰鬥][鬥戰影射」開始"},
		{"adjacent brackets that are not IDs", "[1][2]", "[1][2]"},
		{"adjacent brackets of an ID and a non-ID", "[1][_x][]", "[1][_x][]"},
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

// The Chinese translation writes djp for every language style, whichever
// language KR uses; other styles are left alone.
func TestNormalizeLanguageStyle(t *testing.T) {
	tests := []struct {
		name, kr, in, want string
	}{
		{"dkr", `<style="dkr">피`, `<style="dkr">血`, `<style="djp">血`},
		{"dkr with suffix", `<style="dkr_CP6_2">피`, `<style="dkr_CP6_2">血`, `<style="djp_CP6_2">血`},
		{"upper case keeps its case", `<style="DKR_CP9_1">피`, `<style="DKR_CP9_1">血`, `<style="DJP_CP9_1">血`},
		{"capitalized keeps its case", `<style="Den_x">피`, `<style="Den_x">血`, `<style="Djp_x">血`},
		{"en", `<style="den">피`, `<style="den">血`, `<style="djp">血`},
		{"already djp", `<style="dkr">피`, `<style="djp">血`, `<style="djp">血`},
		{"already DJP with suffix", `<style="DKR_CP9_1">피`, `<style="DJP_CP9_1">血`, `<style="DJP_CP9_1">血`},
		{"unquoted value", `<style=dkr>피`, `<style=dkr>血`, `<style=djp>血`},
		{"with closing tag", `<style="dkr">피</style>`, `<style="dkr">血</style>`, `<style="djp">血</style>`},
		{"several tags", `<style="dkr_CP6_1">가</style> <style="dkr_CP6_2">나</style>`, `<style="dkr_CP6_1">甲</style> <style="DKR_CP6_2">乙</style>`, `<style="djp_CP6_1">甲</style> <style="DJP_CP6_2">乙</style>`},
		{"KR has no style tag", `피`, `<style="dkr">血`, `<style="djp">血`},
		{"KR is empty", ``, `<style="dkr_CP6_2">血`, `<style="djp_CP6_2">血`},
		{"highlight is left alone", `<style="highlight">피`, `<style="highlight">血</style>`, `<style="highlight">血</style>`},
		{"upgradeHighlight is left alone", `<style="upgradeHighlight">피`, `<style="upgradeHighlight">血`, `<style="upgradeHighlight">血`},
		{"yellow is left alone", `<style="yellow">피`, `<style="yellow">血`, `<style="yellow">血`},
		{"highlightConverted is left alone", `<style="highlightConverted">피`, `<style="highlightConverted">血`, `<style="highlightConverted">血`},
		{"unknown language code is left alone", `<style="dkr">피`, `<style="dzh">血`, `<style="dzh">血`},
		{"longer language code is left alone", `<style="dkr">피`, `<style="dkrx">血`, `<style="dkrx">血`},
		{"unbalanced quote is left alone", `<style="dkr">피`, `<style="dkr>血`, `<style="dkr>血`},
		{"text that is not a tag is left alone", `dkr`, `dkr <style> style="dkr"`, `dkr <style> style="dkr"`},
		{"other repairs still apply", `<style="dkr">[Burn]`, `<style="dkr">增加[Burn]強度`, `<style="djp">增加 [Burn] 強度`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Normalize(tc.kr, tc.in)
			if got != tc.want {
				t.Errorf("Normalize(%q, %q) = %q, want %q", tc.kr, tc.in, got, tc.want)
			}
			if again := Normalize(tc.kr, got); again != got {
				t.Errorf("Normalize is not idempotent: %q -> %q", got, again)
			}
		})
	}
}
