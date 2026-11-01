package validate

import (
	"reflect"
	"strings"
	"testing"
)

// codes returns the issue codes in order.
func codes(issues []Issue) []string {
	out := make([]string, len(issues))
	for i, is := range issues {
		out[i] = is.Code
	}
	return out
}

func find(issues []Issue, code string) (Issue, bool) {
	for _, is := range issues {
		if is.Code == code {
			return is, true
		}
	}
	return Issue{}, false
}

func TestTokens(t *testing.T) {
	s := "[Bleed] 위력 {0} <color=#f8c200>[ATTRIBUTES]</color> <1> {Slot} [Bleed]<br><sprite=3></color>"
	tests := []struct {
		name string
		fn   func(string) []string
		want []string
	}{
		{"IDs", IDs, []string{"[Bleed]", "[ATTRIBUTES]", "[Bleed]"}},
		{"Placeholders", Placeholders, []string{"{0}", "<1>", "{Slot}"}},
		{"Tags", Tags, []string{"<color=#f8c200>", "</color>", "<br>", "<sprite=3>", "</color>"}},
	}
	for _, tc := range tests {
		if got := tc.fn(s); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s = %q, want %q", tc.name, got, tc.want)
		}
	}
	for _, fn := range []func(string) []string{IDs, Placeholders, Tags} {
		if got := fn("沒有任何標記"); len(got) != 0 {
			t.Errorf("token function found %q in plain text", got)
		}
	}
	// "<0>" is a placeholder and not a tag, "[1]" is not an ID.
	if got := Tags("<0> [1]"); len(got) != 0 {
		t.Errorf("Tags(\"<0>\") = %q, want none", got)
	}
	if got := IDs("[1] [_x] []"); len(got) != 0 {
		t.Errorf("IDs found %q, want none", got)
	}
	if got := Tags(`<sprite name="Bleed"> <size=80%> <link="a b">`); len(got) != 3 {
		t.Errorf("Tags with attributes = %q, want 3 tags", got)
	}
}

func TestCheckStructural(t *testing.T) {
	tests := []struct {
		name, kr, zh string
		want         []string // expected structural codes, in order
	}{
		{"id missing", "[Bleed] 위력 2 부여", "增加2級流血 強度", []string{"id-mismatch"}},
		{"id extra", "위력 2 부여", "增加2級 [Bleed] 強度", []string{"id-mismatch"}},
		{"id renamed", "[Bleed] 위력 2 부여", "增加2級 [Burn] 強度", []string{"id-mismatch"}},
		{"id count", "[Bleed] 2 [Bleed]", "[Bleed] 2", []string{"id-mismatch"}},
		{"id all-caps must be kept", "[ATTRIBUTES] 속성", "屬性", []string{"id-mismatch"}},
		{"id ok in other order", "[Burn] 그리고 [Bleed]", "[Bleed] 以及 [Burn]", nil},
		{"tag nesting", "<color=#f00>A</color>", "</color>A<color=#f00>", []string{"tag-nesting"}},
		{"tag unclosed", "<color=#f00>A</color>", "<color=#f00>A", []string{"tag-mismatch", "tag-nesting"}},
		{"tag missing", "<color=#f00>A</color>", "A", []string{"tag-mismatch"}},
		{"tag attribute changed", "<color=#f00>A</color>", "<color=#00f>A</color>", []string{"tag-mismatch"}},
		{"tag kind changed", "<b>A</b>", "<i>A</i>", []string{"tag-mismatch"}},
		{"tag extra", "A", "<b>A</b>", []string{"tag-mismatch"}},
		{"tag ok", "<color=#f00>A</color>", "<color=#f00>甲</color>", nil},
		{"tag ok nested", "<b><i>A</i></b>", "<b><i>甲</i></b>", nil},
		{"tag crossed", "<b><i>A</i></b>", "<b><i>甲</b></i>", []string{"tag-nesting"}},
		{"self-closing tags need no closing", "A<br>B<sprite=1>C", "甲<br>乙<sprite=1>丙", nil},
		{"self-closing tag dropped", "A<br>B", "甲乙", []string{"tag-mismatch"}},
		{"self-closing tag between pairs", "<b>A<br>B</b>", "<b>甲<br>乙</b>", nil},
		{"unbalanced KR fragment keeps its tags", "{0}의</color> <color=#e30000>체력 - {1}</color>", "{0}的</color> <color=#e30000>體力 - {1}</color>", nil},
		{"unclosed KR fragment keeps its tags", "<color=#ebcaa2>[{0}] %[{1}] {2}/%<color=#ebcaa2> 감소</color>", "<color=#ebcaa2>[{0}] %[{1}] {2}/%<color=#ebcaa2> 減少</color>", nil},
		{"unbalanced KR fragment missing a tag", "{0}의</color> <color=#e30000>체력 - {1}</color>", "{0}的 <color=#e30000>體力 - {1}</color>", []string{"tag-mismatch"}},
		{"unbalanced KR fragment with a changed attribute", "{0}의</color> <color=#e30000>체력</color>", "{0}的</color> <color=#00e300>體力</color>", []string{"tag-mismatch"}},
		{"balanced KR still requires nesting", "<color=#f00>A</color> <b>B</b>", "<color=#f00>甲 <b></color>乙</b>", []string{"tag-nesting"}},
		{"tag names are case-insensitive when nesting", "<b>A</B>", "<b>甲</B>", nil},
		{"tag name case is part of the tag", "<b>A</b>", "<B>甲</B>", []string{"tag-mismatch"}},
		{"placeholder missing", "{0} 증가", "增加", []string{"placeholder-mismatch"}},
		{"placeholder renamed", "{0} 증가", "增加 {1}", []string{"placeholder-mismatch"}},
		{"placeholder ok", "{0} 증가, {Slot}", "{Slot} 與 {0} 增加", nil},
		{"angle placeholder missing", "<0> 증가", "增加", []string{"placeholder-mismatch"}},
		{"angle placeholder ok", "<0> 증가", "增加 <0>", nil},
		{"dante wrapper dropped", "<별이라… 아픈 건 아니지?>", "星星嗎……這不會痛吧？", []string{"dante-wrapper"}},
		{"dante wrapper kept", "<별이라… 아픈 건 아니지?>", "<星星嗎……這不會痛吧？>", nil},
		{"dante wrapper with spaces around", " <별이라…> ", " <星星嗎……> ", nil},
		{"dante wrapper over lines", "<별이라…\n아픈 건 아니지?>", "星星嗎……\n這不會痛吧？", []string{"dante-wrapper"}},
		{"a tag is not a dante wrapper", "<color=#f00>A</color>", "甲", []string{"tag-mismatch"}},
		{"empty", "가", "", []string{"empty"}},
		{"blank", "가", " \n\t", []string{"empty"}},
		{"nothing to translate", "", "", nil},
		{"blank KR and zh", "  ", "", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Style warnings (such as the length ratio) are not under test.
			var structural []Issue
			for _, is := range Check(tc.kr, tc.zh, Context{}) {
				if is.Class == ClassStructural {
					structural = append(structural, is)
				}
			}
			if len(tc.want) == 0 && len(structural) == 0 {
				return
			}
			if got := codes(structural); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Check(%q, %q) structural codes = %v, want %v (%v)", tc.kr, tc.zh, got, tc.want, structural)
			}
			for _, is := range structural {
				if is.Severity != SeverityHard {
					t.Errorf("%s is %s, want hard", is.Code, is.Severity)
				}
				if is.Message == "" {
					t.Errorf("%s has no message", is.Code)
				}
			}
		})
	}
}

func TestCheckIssueMessagesNameTheDifference(t *testing.T) {
	issues := Check("[Bleed] {0} <b>A</b>", "[Burn] {1} <i>甲</i>", Context{})
	want := map[string][]string{
		"id-mismatch":          {"[Bleed]", "[Burn]"},
		"placeholder-mismatch": {"{0}", "{1}"},
		"tag-mismatch":         {"<b>", "<i>"},
	}
	for code, parts := range want {
		is, ok := find(issues, code)
		if !ok {
			t.Errorf("missing %s in %v", code, codes(issues))
			continue
		}
		for _, p := range parts {
			if !strings.Contains(is.Message, p) {
				t.Errorf("%s message %q should mention %s", code, is.Message, p)
			}
		}
	}
}

func TestCheckEmptyStopsOtherChecks(t *testing.T) {
	// An empty translation lacks every ID and tag, which says nothing new.
	issues := Check("[Bleed] <b>A</b> {0}", "", Context{})
	if got := codes(issues); !reflect.DeepEqual(got, []string{"empty"}) {
		t.Errorf("codes = %v, want only empty", got)
	}
}

var testTerms = []Term{
	{KR: "패키지", Forbidden: []string{"禮包"}},
	{KR: "시즌", Forbidden: []string{`re:第\d+季`}},
}

func TestCheckContent(t *testing.T) {
	tests := []struct {
		name, kr, zh string
		want         string
	}{
		{"hangul syllable", "가", "這個다", "hangul"},
		{"hangul jamo", "가", "這個ㅋㅋ", "hangul"},
		{"simplified", "가", "这個", "simplified"},
		{"后 is also traditional", "가", "皇后", ""},
		{"後 is traditional", "가", "之後", ""},
		{"traditional text", "가", "這個說明裡面的臺灣", ""},
		{"forbidden substring", "패키지 구매", "購買禮包", "forbidden-term"},
		{"forbidden substring ignored without KR term", "구매", "購買禮包", ""},
		{"forbidden regex hit", "시즌 시작", "第8季開始", "forbidden-term"},
		{"forbidden regex miss", "시즌 시작", "第8賽季開始", ""},
		{"allowed rendering", "패키지 구매", "購買組合包", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			issues := Check(tc.kr, tc.zh, Context{Terms: testTerms})
			if tc.want == "" {
				if len(issues) != 0 {
					t.Fatalf("Check(%q, %q) = %v, want no issues", tc.kr, tc.zh, issues)
				}
				return
			}
			if got := codes(issues); !reflect.DeepEqual(got, []string{tc.want}) {
				t.Fatalf("issue codes = %v, want only %q (%v)", got, tc.want, issues)
			}
			is := issues[0]
			if is.Severity != SeverityHard || is.Class != ClassContent {
				t.Errorf("%s is %s/%s, want hard/content", is.Code, is.Severity, is.Class)
			}
		})
	}
}

func TestCheckContentMessages(t *testing.T) {
	is, ok := find(Check("패키지", "这是禮包，还有禮包", Context{Terms: testTerms}), "simplified")
	if !ok || !strings.Contains(is.Message, "这") || !strings.Contains(is.Message, "还") {
		t.Errorf("simplified message %q should list 这 and 还", is.Message)
	}
	is, ok = find(Check("패키지", "禮包", Context{Terms: testTerms}), "forbidden-term")
	if !ok || !strings.Contains(is.Message, "禮包") || !strings.Contains(is.Message, "패키지") {
		t.Errorf("forbidden-term message %q should name the term and its KR", is.Message)
	}
}

func TestForbiddenTermEdgeCases(t *testing.T) {
	terms := []Term{
		{KR: "", Forbidden: []string{"錯"}},         // no KR: never applies
		{KR: "가", Forbidden: []string{""}},         // empty forbidden entry: ignored
		{KR: "나", Forbidden: []string{`re:([`}},    // broken pattern: reported
		{KR: "다", Forbidden: []string{"re:", "禁"}}, // empty pattern matches everything: ignored
	}
	if got := Check("가", "錯字", Context{Terms: terms}); len(got) != 0 {
		t.Errorf("empty KR or empty forbidden entry must not fire, got %v", got)
	}
	is, ok := find(Check("나", "好", Context{Terms: terms}), "forbidden-term")
	if !ok || !strings.Contains(is.Message, "invalid") {
		t.Errorf("a broken pattern should be reported as invalid, got %v", is)
	}
	if got := Check("다", "好", Context{Terms: terms}); len(got) != 0 {
		t.Errorf("empty re: pattern must be ignored, got %v", got)
	}
	if got := Check("다", "禁止", Context{Terms: terms}); len(got) != 1 {
		t.Errorf("later forbidden entries must still be checked, got %v", got)
	}
}

func TestExistingDowngrade(t *testing.T) {
	tests := []struct {
		name         string
		kr, zh       string
		code         string
		wantSeverity Severity
	}{
		{"hangul", "가", "這다", "hangul", SeverityWarn},
		{"simplified", "가", "这", "simplified", SeverityWarn},
		{"forbidden", "패키지", "禮包", "forbidden-term", SeverityWarn},
		{"id-mismatch", "[Bleed] 가", "流血", "id-mismatch", SeverityHard},
		{"placeholder-mismatch", "{0} 가", "增加", "placeholder-mismatch", SeverityHard},
		{"tag-mismatch", "<b>가</b>", "加", "tag-mismatch", SeverityHard},
		{"tag-nesting", "<b>가</b>", "</b>加<b>", "tag-nesting", SeverityHard},
		{"dante-wrapper", "<가>", "加", "dante-wrapper", SeverityHard},
		{"empty", "가", "", "empty", SeverityHard},
		{"style", "가", "加,減", "halfwidth-punct", SeverityWarn},
	}
	for _, tc := range tests {
		for _, existing := range []bool{false, true} {
			is, ok := find(Check(tc.kr, tc.zh, Context{Terms: testTerms, Existing: existing}), tc.code)
			if !ok {
				t.Errorf("%s existing=%v: no %s issue", tc.name, existing, tc.code)
				continue
			}
			want := tc.wantSeverity
			if !existing && is.Class == ClassContent {
				want = SeverityHard
			}
			if is.Severity != want {
				t.Errorf("%s existing=%v: severity = %s, want %s", tc.name, existing, is.Severity, want)
			}
		}
	}

	// A hangul leftover alone does not make an existing unit fail.
	if HasHard(Check("가", "這다", Context{Existing: true})) {
		t.Error("existing unit with only a hangul leftover has a hard issue")
	}
	if !HasHard(Check("가", "這다", Context{})) {
		t.Error("new unit with a hangul leftover has no hard issue")
	}
	if !HasHard(Check("[Bleed] 가", "這다", Context{Existing: true})) {
		t.Error("existing unit with a lost [ID] has no hard issue")
	}
}

func TestStyleWarnings(t *testing.T) {
	tests := []struct {
		name, zh string
		want     []string // expected codes; nil for none
	}{
		{"halfwidth comma", "效果,增加", []string{"halfwidth-punct"}},
		{"halfwidth period after Han", "效果增加.", []string{"halfwidth-punct"}},
		{"halfwidth colon", "效果:增加", []string{"halfwidth-punct"}},
		{"halfwidth comma before Han", "A,增加", []string{"halfwidth-punct"}},
		{"decimal point", "增加1.5倍", nil},
		{"time", "12:30開始", nil},
		{"English sentence", "Hello, world.", nil},
		{"comma between ASCII in Chinese text", "第1,2項", nil},
		{"fullwidth ellipsis", "……", nil},
		{"ascii ellipsis", "...", []string{"ascii-ellipsis"}},
		{"ascii ellipsis next to Han is one warning", "好...", []string{"ascii-ellipsis"}},
		{"long ascii ellipsis", "好.....", []string{"ascii-ellipsis"}},
		{"curly double quotes", "他說“好”", []string{"curly-quotes"}},
		{"corner quotes", "他說「好」", nil},
		{"kana", "これは測試", []string{"kana"}},
		{"katakana", "テスト", []string{"kana"}},
		{"middle dot is not kana", "第8賽季同步・異想解析", nil},
		{"fullwidth tilde", "嘛～", nil},
		{"em dashes", "好——不好", nil},
		{"two warnings", "好,“好”...", []string{"halfwidth-punct", "ascii-ellipsis", "curly-quotes"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			issues := Check("가", tc.zh, Context{})
			if len(tc.want) == 0 && len(issues) != 0 {
				t.Fatalf("Check(%q) = %v, want no issues", tc.zh, issues)
			}
			for _, code := range tc.want {
				is, ok := find(issues, code)
				if !ok {
					t.Errorf("Check(%q) codes = %v, want %s", tc.zh, codes(issues), code)
					continue
				}
				if is.Severity != SeverityWarn || is.Class != ClassStyle {
					t.Errorf("%s is %s/%s, want warn/style", code, is.Severity, is.Class)
				}
			}
			if len(issues) != len(tc.want) {
				t.Errorf("Check(%q) codes = %v, want %v", tc.zh, codes(issues), tc.want)
			}
		})
	}
}

func TestHalfwidthPunctIgnoresMarkup(t *testing.T) {
	// The tag sits between the Chinese text and the comma, but the reader
	// sees them touch.
	if _, ok := find(Check("<b>가</b>가", "<b>效果</b>,增加", Context{}), "halfwidth-punct"); !ok {
		t.Error("a comma after a closing tag should still count as next to Chinese")
	}
	// Nothing inside a tag or placeholder is text.
	kr := `<link="甲.乙:丙">가</link> {甲,乙}`
	if got := Check(kr, `<link="甲.乙:丙">甲</link> {甲,乙}`, Context{}); len(got) != 0 {
		t.Errorf("markup alone raised %v", got)
	}
}

func TestLengthRatio(t *testing.T) {
	kr := strings.Repeat("가", 10)
	tests := []struct {
		name, kr, zh string
		warn         bool
	}{
		{"normal", kr, strings.Repeat("字", 8), false},
		{"exactly 3 times", kr, strings.Repeat("字", 30), false},
		{"over 3 times", kr, strings.Repeat("字", 31), true},
		{"exactly 0.2", kr, strings.Repeat("字", 2), false},
		{"under 0.2", kr, "字", true},
		{"short KR is exempt", "가가가", strings.Repeat("字", 40), false},
		{"counts characters, not bytes", kr, strings.Repeat("字", 30), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, got := find(Check(tc.kr, tc.zh, Context{}), "length-ratio")
			if got != tc.warn {
				t.Errorf("length-ratio fired = %v, want %v", got, tc.warn)
			}
		})
	}
}

func TestHasHard(t *testing.T) {
	if HasHard(nil) {
		t.Error("HasHard(nil) = true")
	}
	if HasHard([]Issue{{Code: "kana", Severity: SeverityWarn}}) {
		t.Error("HasHard of warnings only = true")
	}
	if !HasHard([]Issue{{Code: "kana", Severity: SeverityWarn}, {Code: "empty", Severity: SeverityHard}}) {
		t.Error("HasHard with a hard issue = false")
	}
}

func TestSimplifiedTable(t *testing.T) {
	set := simplifiedChars()
	if len(set) < 3000 {
		t.Fatalf("simplified table has %d characters, want at least 3000", len(set))
	}
	for _, c := range "这个们说为时来还对从" {
		if !set[c] {
			t.Errorf("%c should be in the simplified table", c)
		}
	}
	// Characters that are valid in Traditional Chinese must never be listed.
	for _, c := range "後后干乾幹里裡裏台臺只隻面麵布佈着著發髮這個們說為時來還對從臺灣體國學會" +
		"群床峰秘霉灶痴粽肴苧幺么" {
		if set[c] {
			t.Errorf("%c is valid in Traditional Chinese and must not be in the simplified table", c)
		}
	}
}

func TestIsDanteLine(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"<별이라… 아픈 건 아니지?>", true},
		{" <별이라…> ", true},
		{"<별이라…\n아픈 건 아니지?>", true},
		{"<color=#f00>A</color>", false},
		{"</b>", false},
		{"<0>", false},
		{"별이라", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := IsDanteLine(tt.s); got != tt.want {
			t.Errorf("IsDanteLine(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

// The language code of a lyric style tag differs between KR, EN and JP, and the
// Chinese community packs write djp, so the three count as the same tag.
func TestLanguageStyleTagsAreEquivalent(t *testing.T) {
	tests := []struct {
		name, kr, zh string
	}{
		{"dkr and djp", `<style="dkr">피가 보고 싶었어`, `<style="djp">我期待了暴力`},
		{"dkr and den", `<style="dkr">피가 보고 싶었어`, `<style="den">我期待了暴力`},
		{"suffix kept", `<style="dkr_CP6_2">피가`, `<style="djp_CP6_2">血`},
		{"kr and en upper case", `<style="DKR_CP9_1">피가`, `<style="DEN_CP9_1">血`},
		{"suffix case ignored", `<style="DKR_CP9_1">피가`, `<style="djp_cp9_1">血`},
		{"language code case ignored", `<style="dkr">피가`, `<style="DJP">血`},
		{"mixed case code", `<style="Den_x">피가`, `<style="dJp_X">血`},
		{"unquoted value", `<style=dkr>피가`, `<style=djp>血`},
		{"with closing tag", `<style="dkr">피가</style>`, `<style="djp">血</style>`},
		{"next to another tag", `<b><style="dkr">피가</b>`, `<b><style="djp">血</b>`},
		{"two lines of one lyric", `<style="dkr_CP6_1">가</style> <style="dkr_CP6_2">나</style>`, `<style="djp_CP6_1">甲</style> <style="djp_CP6_2">乙</style>`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if is, ok := find(Check(tc.kr, tc.zh, Context{}), CodeTagMismatch); ok {
				t.Errorf("Check(%q, %q) reports %s: %s", tc.kr, tc.zh, is.Code, is.Message)
			}
		})
	}
}

func TestLanguageStyleTagsStillCompared(t *testing.T) {
	tests := []struct {
		name, kr, zh string
		missing      string // the real KR tag the message must name
		unexpected   string // the real zh tag the message must name
	}{
		{"different suffix", `<style="dkr">피가`, `<style="djp_CP6_2">血`, `<style="dkr">`, `<style="djp_CP6_2">`},
		{"suffix dropped", `<style="dkr_CP6_2">피가`, `<style="djp">血`, `<style="dkr_CP6_2">`, `<style="djp">`},
		{"other suffix", `<style="dkr_CP6_1">피가`, `<style="djp_CP6_2">血`, `<style="dkr_CP6_1">`, `<style="djp_CP6_2">`},
		{"non-language style", `<style="highlight">피가`, `<style="djp">血`, `<style="highlight">`, `<style="djp">`},
		{"language style where KR has a plain one", `<style="dkr">피가`, `<style="highlight">血`, `<style="dkr">`, `<style="highlight">`},
		{"unknown language code", `<style="dkr">피가`, `<style="dzh">血`, `<style="dkr">`, `<style="dzh">`},
		{"longer language code", `<style="dkr">피가`, `<style="djpn">血`, `<style="dkr">`, `<style="djpn">`},
		{"suffix without underscore", `<style="dkr">피가`, `<style="djp2">血`, `<style="dkr">`, `<style="djp2">`},
		{"quotes differ", `<style="dkr">피가`, `<style=djp>血`, `<style="dkr">`, `<style=djp>`},
		{"unbalanced quote", `<style="dkr">피가`, `<style="djp>血`, `<style="dkr">`, `<style="djp>`},
		{"other style names differ", `<style="highlight">피가`, `<style="yellow">血`, `<style="highlight">`, `<style="yellow">`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is, ok := find(Check(tc.kr, tc.zh, Context{}), CodeTagMismatch)
			if !ok {
				t.Fatalf("Check(%q, %q) has no %s", tc.kr, tc.zh, CodeTagMismatch)
			}
			// The canonical form is only for comparing: the message shows real tags.
			for _, want := range []string{"missing " + tc.missing, "unexpected " + tc.unexpected} {
				if !strings.Contains(is.Message, want) {
					t.Errorf("message %q does not contain %q", is.Message, want)
				}
			}
		})
	}
}

// A language style that matches must not hide a real difference, nor show up
// in its message.
func TestLanguageStyleTagsDoNotHideOtherDifferences(t *testing.T) {
	is, ok := find(Check(`<style="dkr"><b>피가</b>`, `<style="djp">血`, Context{}), CodeTagMismatch)
	if !ok {
		t.Fatal("a missing <b> is not reported")
	}
	if want := ": missing </b> <b>"; !strings.Contains(is.Message, want) || strings.Contains(is.Message, "style") {
		t.Errorf("message %q should name only the <b> tags (%q)", is.Message, want)
	}
	// Two KR tags against one zh tag of the same language still differ by one.
	is, ok = find(Check(`<style="dkr">가 <style="dkr">나`, `<style="djp">甲 乙`, Context{}), CodeTagMismatch)
	if !ok || !strings.Contains(is.Message, `missing <style="dkr">`) {
		t.Errorf("a dropped language style is not reported: %+v", is)
	}
}

// A tag KR opens and never closes (the lyric <style="dkr">, a lone <size=20>)
// may be closed in zh: the official EN text does so.
func TestExtraClosingTagForUnclosedKRTag(t *testing.T) {
	tests := []struct {
		name, kr, zh string
	}{
		{"style closed after a sized word", `<style="dkr"><size=20>별</size>`, `<style="djp">一顆<size=20>星星</size></style>`},
		{"style closed at the end", `<style="dkr">어쩌면 그때가 바로`, `<style="djp">可能就是那個時候</style>`},
		{"two tags both closed", `<style="dkr"><size=20>잔물결`, `<style="djp"><size=20>層層波浪</size></style>`},
		{"two unclosed openings, two closings", `<style="dkr">가 <style="dkr">나`, `<style="djp">甲</style> <style="djp">乙</style>`},
		{"other tag", `<b>가`, `<b>甲</b>`},
		{"the closing tag is optional", `<style="dkr">가`, `<style="djp">甲`},
		{"fewer closings than the allowance", `<style="dkr">가 <style="dkr">나`, `<style="djp">甲 <style="djp">乙</style>`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if is, ok := find(Check(tc.kr, tc.zh, Context{}), CodeTagMismatch); ok {
				t.Errorf("Check(%q, %q) reports %s: %s", tc.kr, tc.zh, is.Code, is.Message)
			}
		})
	}
}

// The allowance is per tag name: KR's openings minus KR's closings, at least
// 0. Nothing else about the comparison changes.
func TestExtraClosingTagStillLimited(t *testing.T) {
	tests := []struct {
		name, kr, zh string
		diff         string // what the message must say, as describeDiff words it
	}{
		{"KR closes the tag", `<b>x</b>`, `<b>x</b></b>`, ": unexpected </b>"},
		{"beyond the allowance", `<style="dkr">a`, `<style="djp">a</style></style>`, ": unexpected </style>"},
		{"three closings, one allowed", `<style="dkr">a`, `<style="djp">a</style></style></style>`, ": unexpected </style> </style>"},
		{"a name KR never opened", `<style="dkr">a`, `<style="djp">a</b>`, ": unexpected </b>"},
		{"no tags in KR", `a`, `a</b>`, ": unexpected </b>"},
		{"the allowance belongs to another name", `<style="dkr"><size=20>a</size>`, `<style="djp"><size=20>a</size></size>`, ": unexpected </size>"},
		{"KR has more closings than openings", `x</color>`, `x</color></color>`, ": unexpected </color>"},
		{"self-closing tags have no closing tag", `A<br>B`, `A<br></br>B`, ": unexpected </br>"},
		{"an extra opening tag is still reported", `<style="dkr">a`, `<style="djp"><b>a</style>`, ": unexpected <b>"},
		{"a missing closing tag is still reported", `<b>x</b>`, `<b>x`, ": missing </b>"},
		{"a missing opening tag is still reported", `<style="dkr">a`, `a</style>`, ": missing <style=\"dkr\">"},
		{"a forgiven closing tag does not hide a missing one", `<style="dkr"><b>a</b>`, `<style="djp">a</style>`, ": missing </b> <b>"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is, ok := find(Check(tc.kr, tc.zh, Context{}), CodeTagMismatch)
			if !ok {
				t.Fatalf("Check(%q, %q) has no %s", tc.kr, tc.zh, CodeTagMismatch)
			}
			if want := "tags differ from KR" + tc.diff + "; opening tags"; !strings.Contains(is.Message, want) {
				t.Errorf("message %q does not contain %q", is.Message, want)
			}
		})
	}
}

// The comparison form of a language style must not be writable in a
// translation: a literal d* is an ordinary, different style.
func TestLanguageStyleComparisonKeyCannotBeForged(t *testing.T) {
	tests := []struct{ name, kr, zh string }{
		{"star for the language", `<style="dkr">피가`, `<style="d*">血`},
		{"star with a suffix", `<style="dkr_CP6_2">피가`, `<style="d*_cp6_2">血`},
		{"star in KR", `<style="d*">피가`, `<style="djp">血`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok := find(Check(tc.kr, tc.zh, Context{}), CodeTagMismatch); !ok {
				t.Errorf("Check(%q, %q) has no %s", tc.kr, tc.zh, CodeTagMismatch)
			}
		})
	}
}
