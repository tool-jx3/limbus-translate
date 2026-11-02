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

// base.md 6: an [ID] has a half-width space on both sides, and two [ID]s that
// touch are separated. The space goes beside a keyword ID, not after any "]":
// a "]" that ends another pair of brackets, such as the "][" of the mirrored
// event name 射影戰鬥][鬥戰影射, is not asked to take one. A placeholder [{n}]
// is tight on both sides (guides/base.md): no space between it and an [ID]
// that follows, as none goes between an [ID] and a [{n}] that follows.
func TestNormalizeSpacesOnlyAfterKeywordIDs(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"] of another pair before an ID", "名][Burn]", "名][Burn]"},
		{"] of another pair, text after the ID", "名][Burn]強度", "名][Burn] 強度"},
		{"a bracketed word before an ID", "[名][Burn]", "[名][Burn]"},
		{"a bracketed number before an ID", "第[1][Burn]項", "第[1][Burn] 項"},
		{"mirrored name next to an ID elsewhere", "射影戰鬥][鬥戰影射增加[Burn]", "射影戰鬥][鬥戰影射增加 [Burn]"},
		{"an ID before and after another pair", "[Burn]名][Bleed]", "[Burn] 名][Bleed]"},
		{"IDs that touch are still separated", "[Burn][Bleed]", "[Burn] [Bleed]"},
		{"IDs that touch, with text around", "增加[Burn][Bleed]強度", "增加 [Burn] [Bleed] 強度"},
		{"IDs with a tag between them", "[Burn]</b>[Bleed]", "[Burn]</b>[Bleed]"},
		{"a name followed by an ID", "釘[Console]", "釘 [Console]"},
		{"an ID at the start", "[Burn]增加", "[Burn] 增加"},
		{"placeholder brackets stay tight", "罪人[{0}]獲得了", "罪人[{0}]獲得了"},
		{"placeholder brackets stay tight beside an ID", "罪人[{0}]獲得[Burn]", "罪人[{0}]獲得 [Burn]"},
		{"an ID right after a placeholder", "[{0}][Burn]", "[{0}][Burn]"},
		{"an ID right after a placeholder, text after it", "獲得[{0}][Burn]強度", "獲得[{0}][Burn] 強度"},
		{"a placeholder between IDs", "[Burn][{0}][Bleed]", "[Burn][{0}][Bleed]"},
		{"an ID before a placeholder", "[Burn][{0}]", "[Burn][{0}]"},
		{"TabExplain after another pair", "名][TabExplain]說明", "名][TabExplain]說明"},
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
		{"tag name in upper case", `<STYLE="dkr">피</STYLE>`, `<STYLE="dkr">血</STYLE>`, `<STYLE="djp">血</STYLE>`},
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

func TestNormalizeKeywordSpace(t *testing.T) {
	names := []string{"破裂", "破裂易損", "強壯"}
	tests := []struct {
		name, in, want string
	}{
		{"name before text", "2級破裂強度", "2級破裂 強度"},
		{"name before full-width punctuation", "2層強壯，", "2層強壯 ，"},
		{"name at the end", "3回破裂易損", "3回破裂易損 "},
		{"the longer name wins", "使目標增加3回破裂易損強度", "使目標增加3回破裂易損 強度"},
		{"the space is already there", "2級破裂 強度", "2級破裂 強度"},
		{"no count before the name", "破裂的力量", "破裂的力量"},
		{"a count of another kind", "2次破裂強度", "2次破裂強度"},
		{"several digits", "12層強壯", "12層強壯 "},
		{"two keywords", "2層強壯和1級破裂強度", "2層強壯 和1級破裂 強度"},
		{"a count that no name follows", "2級強度", "2級強度"},
		{"a name that is not in the list", "2級燒傷強度", "2級燒傷強度"},
		{"a newline is not a half-width space", "2層強壯\n", "2層強壯 \n"},
		{"a full-width space is not a half-width space", "2層強壯　", "2層強壯 　"},
		{"a tag follows", "<color=#f00>2層強壯</color>", "<color=#f00>2層強壯 </color>"},
		{"[TabExplain] must touch the name", "2層強壯[TabExplain]說明", "2層強壯[TabExplain]說明"},
		{"an [ID] keeps its own spacing", "2層 [Burn] 強度", "2層 [Burn] 強度"},
		{"empty", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeKeywordSpace(tc.in, names)
			if got != tc.want {
				t.Errorf("NormalizeKeywordSpace(%q) = %q, want %q", tc.in, got, tc.want)
			}
			if again := NormalizeKeywordSpace(got, names); again != got {
				t.Errorf("NormalizeKeywordSpace is not idempotent: %q -> %q", got, again)
			}
			// Normalize must leave the space alone, or the two would undo each other.
			if again := Normalize("", got); again != got {
				t.Errorf("Normalize undoes the result: %q -> %q", got, again)
			}
		})
	}
}

// The order of names does not decide which one matches, and a list without
// usable names changes nothing.
func TestNormalizeKeywordSpaceNames(t *testing.T) {
	in := "2級破裂易損強度"
	want := "2級破裂易損 強度"
	for _, names := range [][]string{
		{"破裂", "破裂易損"},
		{"破裂易損", "破裂"},
		{"", "破裂", "破裂易損"},
	} {
		if got := NormalizeKeywordSpace(in, names); got != want {
			t.Errorf("NormalizeKeywordSpace(%q, %q) = %q, want %q", in, names, got, want)
		}
	}
	for _, names := range [][]string{nil, {}, {""}} {
		if got := NormalizeKeywordSpace(in, names); got != in {
			t.Errorf("NormalizeKeywordSpace(%q, %q) = %q, want it unchanged", in, names, got)
		}
	}
	names := []string{"破裂", "破裂易損"}
	NormalizeKeywordSpace(in, names)
	if names[0] != "破裂" || names[1] != "破裂易損" {
		t.Errorf("NormalizeKeywordSpace reordered the caller's list: %q", names)
	}
}
