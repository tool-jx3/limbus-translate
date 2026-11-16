// Package rubyx 處理自訂標籤 <rubycustomize=讀音>本文</rubycustomize>：
// 語法檢查、去標籤、轉成 TextMeshPro 內建標籤，以及還原。
// 本套件不依賴專案內其他套件。
package rubyx

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
)

// Tag 是自訂標籤的名稱（小寫）。
const Tag = "rubycustomize"

// group 比對一組完整的標籤；名稱區分大小寫。
// 讀音可用雙引號包住（引號不屬於讀音）；讀音與本文都不得含標籤符號或換行。
var group = regexp.MustCompile(`<rubycustomize=(?:"([^"<>\r\n]+)"|([^<>"\r\n]+))>([^<>\r\n]+)</rubycustomize>`)

// anyTag 找出任何寫法（不分大小寫）的開頭或結尾標籤起點。
var anyTag = regexp.MustCompile(`(?i)<(/?)rubycustomize`)

// converted 只認 Convert 輸出的完整結構。
var converted = regexp.MustCompile(`<nobr>([^<>\r\n]+)<space=-?[0-9.]+em><voffset=[^<>]+><size=[^<>]+>([^<>\r\n]+)</size></voffset><space=-?[0-9.]+em></nobr>`)

var (
	sizeRe    = regexp.MustCompile(`^([0-9]+)%$`)
	voffsetRe = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?em$`)
)

// Has 回報 s 是否含有 <rubycustomize 或 </rubycustomize，大小寫不拘。
func Has(s string) bool { return anyTag.MatchString(s) }

// Problems 回傳語法問題（英文短句）；沒有問題時為 nil。
func Problems(s string) []string {
	if !Has(s) {
		return nil
	}
	rest := group.ReplaceAllString(s, "\x00")
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	for _, m := range anyTag.FindAllStringSubmatchIndex(rest, -1) {
		name := rest[m[3]:m[1]]
		if name != Tag {
			add("tag name must be lower-case " + Tag)
			continue
		}
		if m[3] > m[2] {
			add("closing tag without a matching opening tag")
		} else {
			add("opening tag must be <" + Tag + "=reading>body</" + Tag + "> with a non-empty reading and a non-empty body without tags, newlines or nesting")
		}
	}
	return out
}

// Strip 把每一組換成本文。只在 Problems 為 nil 時呼叫。
func Strip(s string) string {
	return group.ReplaceAllString(s, "$3")
}

// Style 是轉換用的樣式，例如 {"50%", 0.5, "1em"}。
type Style struct {
	Size    string
	Scale   float64
	VOffset string
}

// ParseStyle 檢查並建立 Style：size 為 N%（1～100），voffset 為數字加 em。
func ParseStyle(size, voffset string) (Style, error) {
	m := sizeRe.FindStringSubmatch(size)
	if m == nil {
		return Style{}, fmt.Errorf("size %q must look like N%%", size)
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n < 1 || n > 100 {
		return Style{}, fmt.Errorf("size %q must be between 1%% and 100%%", size)
	}
	if !voffsetRe.MatchString(voffset) {
		return Style{}, fmt.Errorf("voffset %q must be a number followed by em", voffset)
	}
	return Style{Size: size, Scale: float64(n) / 100, VOffset: voffset}, nil
}

// Convert 把每一組換成 <nobr>B<space=S1em><voffset=V><size=Z>R</size></voffset><space=S2em></nobr>。
// advance 回傳字元寬度（em），缺字由呼叫端回傳 1。
func Convert(s string, st Style, advance func(rune) float64) string {
	return group.ReplaceAllStringFunc(s, func(g string) string {
		m := group.FindStringSubmatch(g)
		reading := m[1] + m[2]
		body := m[3]
		wb := width(body, advance)
		wr := width(reading, advance) * st.Scale
		return "<nobr>" + body +
			"<space=" + num(-(wb+wr)/2) + "em>" +
			"<voffset=" + st.VOffset + ">" +
			"<size=" + st.Size + ">" + reading + "</size></voffset>" +
			"<space=" + num((wb-wr)/2) + "em></nobr>"
	})
}

// Restore 把 Convert 輸出的完整結構還原成 <rubycustomize=R>B</rubycustomize>，其他文字不動。
func Restore(s string) string {
	return converted.ReplaceAllString(s, "<"+Tag+"=$2>$1</"+Tag+">")
}

func width(s string, advance func(rune) float64) float64 {
	var w float64
	for _, r := range s {
		w += advance(r)
	}
	return w
}

// num 四捨五入到小數 3 位，去掉尾端的 0，並把 -0 寫成 0。
func num(x float64) string {
	x = math.Round(x*1000) / 1000
	if x == 0 {
		return "0"
	}
	return strconv.FormatFloat(x, 'f', -1, 64)
}
