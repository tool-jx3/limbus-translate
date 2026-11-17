package build

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/image/font/sfnt"

	"github.com/tool-jx3/limbus-translate/internal/rubyx"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
)

const (
	rubyFile    = "ScenarioModelCodes-AutoCreated.json"
	rubyPointer = "/dataList/0/name"
	rubyKR      = `{"dataList":[{"id":"x","name":"UNGEZIEFER KAISER"}]}`
)

// rubyEnv is an environment whose rules hold the repo's ruby whitelist.
func rubyEnv(t *testing.T) *testEnv {
	t.Helper()
	e := newEnv(t, map[string]string{rubyFile: rubyKR}, nil)
	r, err := rules.Load("../..")
	if err != nil {
		t.Fatal(err)
	}
	e.opt.Rules = r
	return e
}

func shippedAdvance(t *testing.T) func(rune) float64 {
	t.Helper()
	data, err := os.ReadFile("../../assets/fonts/SarasaGothicTC-Bold.ttf")
	if err != nil {
		t.Fatal(err)
	}
	face, err := sfnt.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	return (&inputs{face: face}).advance
}

// The expected strings are what was measured in the game with the shipped font.
func TestConvertWithShippedFontMatchesInGameTests(t *testing.T) {
	adv := shippedAdvance(t)
	if got := adv('良'); got != 1 {
		t.Errorf("advance(良) = %v, want 1", got)
	}
	if got := adv('A'); got < 0.7465 || got > 0.7475 {
		t.Errorf("advance(A) = %v, want 0.747", got)
	}
	st, err := rubyx.ParseStyle("50%", "1em")
	if err != nil {
		t.Fatal(err)
	}
	for in, want := range map[string]string{
		"<rubycustomize=ヨシヒデ>良秀</rubycustomize>":                "<nobr>良秀<space=-2em><voffset=1em><size=50%>ヨシヒデ</size></voffset><space=0em></nobr>",
		"<rubycustomize=UNGEZIEFER KAISER>害蟲皇帝</rubycustomize>": "<nobr>害蟲皇帝<space=-4.542em><voffset=1em><size=50%>UNGEZIEFER KAISER</size></voffset><space=-0.542em></nobr>",
	} {
		if got := rubyx.Convert(in, st, adv); got != want {
			t.Errorf("Convert(%q) =\n%s\nwant\n%s", in, got, want)
		}
	}
}

func TestAdvanceOfAMissingGlyphIsOne(t *testing.T) {
	adv := shippedAdvance(t)
	if got := adv(0x10FFFD); got != 1 {
		t.Errorf("advance(missing) = %v, want 1", got)
	}
}

func TestBuildConvertsWhitelistedRubyCustomize(t *testing.T) {
	e := rubyEnv(t)
	e.save(rec(rubyFile, rubyPointer, "UNGEZIEFER KAISER", "<rubycustomize=UNGEZIEFER KAISER>害蟲皇帝</rubycustomize>"))

	rep := e.run()
	if rep.RubyConverted != 1 {
		t.Errorf("RubyConverted = %d, want 1", rep.RubyConverted)
	}
	got := e.out(rubyFile)
	if strings.Contains(got, "<rubycustomize") {
		t.Errorf("pack =\n%s\nstill holds <rubycustomize>", got)
	}
	if !strings.Contains(got, "<nobr>害蟲皇帝<space=") || !strings.Contains(got, "<size=50%>UNGEZIEFER KAISER</size></voffset><space=") {
		t.Errorf("pack =\n%s\nwant the converted string", got)
	}
}

func TestBuildFallsBackOnMalformedRubyCustomize(t *testing.T) {
	e := rubyEnv(t)
	bad := rec(rubyFile, rubyPointer, "UNGEZIEFER KAISER", "<rubycustomize=>良秀</rubycustomize>")
	e.save(bad)

	rep := e.run()
	if rep.RubyConverted != 0 || rep.StructuralFallback != 1 {
		t.Errorf("RubyConverted=%d StructuralFallback=%d, want 0 1", rep.RubyConverted, rep.StructuralFallback)
	}
	got := e.out(rubyFile)
	if strings.Contains(got, "rubycustomize") || !strings.Contains(got, `"name": "UNGEZIEFER KAISER"`) {
		t.Errorf("pack =\n%s\nwant the KR stand-in", got)
	}
	want := []store.PendingItem{{Key: bad.Key, Reason: "structural-fail", Draft: bad.ZH}}
	if got := e.pending(); !reflect.DeepEqual(got, want) {
		t.Errorf("pending = %+v, want %+v", got, want)
	}
}

func TestEnsureNoRubyCustomize(t *testing.T) {
	for _, text := range []string{"<rubycustomize=a>b</rubycustomize>", "x</rubycustomize>", "<rubycustomize=>"} {
		err := ensureNoRubyCustomize("F.json", "/a/0/b", text)
		if err == nil || !strings.Contains(err.Error(), "F.json") || !strings.Contains(err.Error(), "/a/0/b") {
			t.Errorf("ensureNoRubyCustomize(%q) = %v, want an error naming file and pointer", text, err)
		}
	}
	for _, text := range []string{"<nobr>良秀<space=-2em><voffset=1em><size=50%>ヨ</size></voffset><space=0em></nobr>", "plain"} {
		if err := ensureNoRubyCustomize("F.json", "/a", text); err != nil {
			t.Errorf("ensureNoRubyCustomize(%q) = %v, want nil", text, err)
		}
	}
}
