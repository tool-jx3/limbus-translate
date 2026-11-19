package validate

import (
	"reflect"
	"testing"
)

func TestNumbers(t *testing.T) {
	for _, c := range []struct {
		name, s string
		want    []string
	}{
		{"none", "피해를 준다", nil},
		{"empty", "", nil},
		{"runs", "(속도 차이 × 5)% 증가 (최대 40%)", []string{"5", "40"}},
		{"every run is one number", "1,000의 2.5배, 007", []string{"1", "000", "2", "5", "007"}},
		{"beside other text", "+30%만큼 3턴간 x2", []string{"30", "3", "2"}},
		{"full-width digits are text", "３회 3회", []string{"3"}},
		{"not in an [ID]", "[Burn2] 3 부여, [Sinking_1]", []string{"3"}},
		{"brackets that are no [ID]", "[1] 2", []string{"1", "2"}},
		{"not in a placeholder", "{0}턴 동안 {1} 4 <0> [{2}]", []string{"4"}},
		{"not in a tag", `<color=#f8c200>5</color> <size=120%>6</size> <sprite name="Coin1">7`, []string{"5", "6", "7"}},
		{"style and ruby tags", `<style="DKR_CP9_1">8</style> <ruby=9>十</ruby>`, []string{"8"}},
		{"a run cut by a tag is two", "12<b>3</b>", []string{"12", "3"}},
		{"what is written like a tag but is none", "<DAY 3, 2 사무소>", []string{"3", "2"}},
	} {
		var got []string
		for _, n := range Numbers(c.s) {
			got = append(got, c.s[n[0]:n[1]])
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: Numbers(%q) = %q, want %q", c.name, c.s, got, c.want)
		}
	}
}
