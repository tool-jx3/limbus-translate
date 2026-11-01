package rewrite

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

const (
	storyFile = "StoryData/S001A.json"
	otherFile = "Other.json"
)

func writeFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// openKR writes the KR files (file name to the texts of its lines) of a raw
// snapshot, with empty EN and JP packs, and returns the KR pack.
func openKR(t *testing.T, kr map[string][]string) *pack.Pack {
	t.Helper()
	raw := t.TempDir()
	for _, lang := range []string{"kr", "en", "jp"} {
		if err := os.MkdirAll(filepath.Join(raw, lang), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for file, lines := range kr {
		var items []string
		for i, line := range lines {
			items = append(items, fmt.Sprintf(`{"id":%d,"content":%q}`, i, line))
		}
		dir, name := path.Split(file)
		writeFile(t, filepath.Join(raw, "kr", filepath.FromSlash(dir), "KR_"+name), `{"dataList":[`+strings.Join(items, ",")+`]}`)
	}
	set, err := pack.OpenRaw(raw)
	if err != nil {
		t.Fatal(err)
	}
	return set.KR
}

func key(file string, i int) string { return fmt.Sprintf("%s|/dataList/%d/content", file, i) }

func rec(key, zh, origin string) store.Record {
	return store.Record{Key: key, KRSha: units.KRSha("대사"), ZH: zh, Origin: origin, Since: "raw-20261029", Rules: "abc", Model: "m", Batch: "b1", Flags: []string{"needs_review"}}
}

func accept(_, _, _ string) bool { return true }

func cloneRecs(recs map[string]store.Record) map[string]store.Record {
	out := make(map[string]store.Record, len(recs))
	for k, r := range recs {
		r.Flags = slices.Clone(r.Flags)
		out[k] = r
	}
	return out
}

// The project keeps a half-width space after the name of a keyword, so a
// translation that lacks it is rewritten by adding one.
var addSpace = File{
	ID:      "2026-11-05-keyword-space",
	Reason:  "關鍵字名稱後加半形空格",
	Files:   []string{"StoryData/**"},
	Pattern: `(\d+級破裂)(強度)`,
	Replace: "$1 $2",
}

func TestRewriteApply(t *testing.T) {
	kr := openKR(t, map[string][]string{
		storyFile: {"대사 0", "대사 1", "대사 2", "대사 3", "대사 4", "대사 5"},
		otherFile: {"대사 0"},
	})
	recs := map[string]store.Record{
		key(storyFile, 0): rec(key(storyFile, 0), "賦予 2級破裂強度", store.OriginClaude),
		key(storyFile, 1): rec(key(storyFile, 1), "3級破裂強度，4級破裂強度", store.OriginParatranz),
		key(storyFile, 2): rec(key(storyFile, 2), "2級破裂強度", store.OriginHuman),    // a person wrote it
		key(storyFile, 3): rec(key(storyFile, 3), "沒有關鍵字", store.OriginClaude),    // does not match
		key(storyFile, 4): rec(key(storyFile, 4), "2級破裂強度", store.OriginCopy),     // not an origin of the default
		key(storyFile, 5): rec(key(storyFile, 5), "2級破裂強度", store.OriginFallback), // likewise
		key(otherFile, 0): rec(key(otherFile, 0), "2級破裂強度", store.OriginClaude),   // not in files
	}
	before := cloneRecs(recs)

	got := Apply(addSpace, recs, kr, accept)
	if want := []string{key(storyFile, 0), key(storyFile, 1)}; !slices.Equal(got, want) {
		t.Fatalf("Apply = %v, want %v", got, want)
	}
	wantZH := map[string]string{
		key(storyFile, 0): "賦予 2級破裂 強度",
		key(storyFile, 1): "3級破裂 強度，4級破裂 強度", // every match
	}
	for k, r := range recs {
		want := before[k]
		if zh, changed := wantZH[k]; changed {
			want.ZH = zh // only the text: the origin, the sha and the rest stay
		}
		if !reflect.DeepEqual(r, want) {
			t.Errorf("record %s = %+v, want %+v", k, r, want)
		}
	}

	// The pattern is gone from what it rewrote: the second time changes nothing.
	if again := Apply(addSpace, recs, kr, accept); len(again) != 0 {
		t.Errorf("Apply again = %v, want nothing", again)
	}
}

func TestRewriteApplyHonoursTheCheck(t *testing.T) {
	kr := openKR(t, map[string][]string{storyFile: {"대사 0", "대사 1"}})
	recs := map[string]store.Record{
		key(storyFile, 0): rec(key(storyFile, 0), "2級破裂強度", store.OriginClaude),
		key(storyFile, 1): rec(key(storyFile, 1), "3級破裂強度", store.OriginClaude),
	}
	before := cloneRecs(recs)

	type call struct{ key, kr, zh string }
	var calls []call
	first := key(storyFile, 0)
	got := Apply(addSpace, recs, kr, func(k, krText, zh string) bool {
		calls = append(calls, call{k, krText, zh})
		return k == first // refuse the second
	})
	if want := []string{key(storyFile, 0)}; !slices.Equal(got, want) {
		t.Errorf("Apply = %v, want only the unit the check accepted", got)
	}
	if recs[key(storyFile, 0)].ZH != "2級破裂 強度" || !reflect.DeepEqual(recs[key(storyFile, 1)], before[key(storyFile, 1)]) {
		t.Errorf("records = %+v, want the refused unit as it was", recs)
	}
	slices.SortFunc(calls, func(a, b call) int { return strings.Compare(a.key, b.key) })
	wantCalls := []call{
		{key(storyFile, 0), "대사 0", "2級破裂 強度"}, // the KR text and the rewritten text
		{key(storyFile, 1), "대사 1", "3級破裂 強度"},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Errorf("check calls = %+v, want %+v", calls, wantCalls)
	}

	// Refusing everything changes nothing.
	recs = cloneRecs(before)
	if got := Apply(addSpace, recs, kr, func(_, _, _ string) bool { return false }); len(got) != 0 || !reflect.DeepEqual(recs, before) {
		t.Errorf("Apply with a check that says no = %v, records %+v; want nothing changed", got, recs)
	}
}

func TestRewriteApplyAsksNoCheckForWhatStaysTheSame(t *testing.T) {
	kr := openKR(t, map[string][]string{storyFile: {"대사 0", "대사 1"}})
	recs := map[string]store.Record{
		key(storyFile, 0): rec(key(storyFile, 0), "2級破裂強度", store.OriginClaude),
		key(storyFile, 1): rec(key(storyFile, 1), "不相干", store.OriginClaude),
	}
	f := addSpace
	f.Replace = "$1$2" // rewrites to the same text
	asked := false
	got := Apply(f, recs, kr, func(_, _, _ string) bool { asked = true; return true })
	if len(got) != 0 || asked {
		t.Errorf("Apply = %v (check asked: %v), want nothing: the text does not change", got, asked)
	}
}

func TestRewriteKRPattern(t *testing.T) {
	kr := openKR(t, map[string][]string{storyFile: {"[Laceration] 위력 5 부여", "뫼르소가 말했다", "위력 1"}})
	recs := map[string]store.Record{
		key(storyFile, 0): rec(key(storyFile, 0), "2級破裂強度", store.OriginClaude),
		key(storyFile, 1): rec(key(storyFile, 1), "2級破裂強度", store.OriginClaude),
		key(storyFile, 2): rec(key(storyFile, 2), "2級破裂強度", store.OriginClaude),
	}
	f := addSpace
	f.KRPattern = `위력 \d`

	got := Apply(f, recs, kr, accept)
	if want := []string{key(storyFile, 0), key(storyFile, 2)}; !slices.Equal(got, want) {
		t.Errorf("Apply = %v, want only the units whose KR matches kr_pattern", got)
	}
	if recs[key(storyFile, 1)].ZH != "2級破裂強度" {
		t.Errorf("a unit whose KR does not match was rewritten: %+v", recs[key(storyFile, 1)])
	}
}

func TestRewriteOrigins(t *testing.T) {
	kr := openKR(t, map[string][]string{storyFile: {"대사 0", "대사 1", "대사 2"}})
	fresh := func() map[string]store.Record {
		return map[string]store.Record{
			key(storyFile, 0): rec(key(storyFile, 0), "2級破裂強度", store.OriginClaude),
			key(storyFile, 1): rec(key(storyFile, 1), "2級破裂強度", store.OriginHuman),
			key(storyFile, 2): rec(key(storyFile, 2), "2級破裂強度", store.OriginParatranz),
		}
	}
	for name, tc := range map[string]struct {
		origins []string
		want    []string
	}{
		"default":  {nil, []string{key(storyFile, 0), key(storyFile, 2)}},
		"human":    {[]string{store.OriginHuman}, []string{key(storyFile, 1)}},
		"claude":   {[]string{store.OriginClaude}, []string{key(storyFile, 0)}},
		"two":      {[]string{store.OriginHuman, store.OriginParatranz}, []string{key(storyFile, 1), key(storyFile, 2)}},
		"explicit": {[]string{store.OriginParatranz, store.OriginClaude, store.OriginHuman}, []string{key(storyFile, 0), key(storyFile, 1), key(storyFile, 2)}},
	} {
		t.Run(name, func(t *testing.T) {
			f := addSpace
			f.Origins = tc.origins
			recs := fresh()
			got := Apply(f, recs, kr, accept)
			if !slices.Equal(got, tc.want) {
				t.Errorf("Apply = %v, want %v", got, tc.want)
			}
			for k, r := range recs {
				if want := fresh()[k].Origin; r.Origin != want {
					t.Errorf("%s: origin = %q, want %q: a rewrite never changes the origin", k, r.Origin, want)
				}
			}
		})
	}
}

func TestRewriteFilesUseTheGlobsOfLctYaml(t *testing.T) {
	kr := openKR(t, map[string][]string{
		"StoryData/S1.json":     {"대사"},
		"StoryData/sub/S2.json": {"대사"},
		"Bufs_Mirror3.json":     {"대사"},
		"BufsExtra/X.json":      {"대사"},
	})
	fresh := func() map[string]store.Record {
		recs := map[string]store.Record{}
		for _, f := range []string{"StoryData/S1.json", "StoryData/sub/S2.json", "Bufs_Mirror3.json", "BufsExtra/X.json"} {
			recs[key(f, 0)] = rec(key(f, 0), "2級破裂強度", store.OriginClaude)
		}
		return recs
	}
	for name, tc := range map[string]struct {
		files []string
		want  []string
	}{
		"directory":      {[]string{"StoryData/**"}, []string{key("StoryData/S1.json", 0), key("StoryData/sub/S2.json", 0)}},
		"glob":           {[]string{"Bufs*.json"}, []string{key("Bufs_Mirror3.json", 0)}},
		"exact and glob": {[]string{"BufsExtra/X.json", "StoryData/S?.json"}, []string{key("BufsExtra/X.json", 0), key("StoryData/S1.json", 0)}},
		"none":           {nil, nil},
		"malformed glob": {[]string{"[a"}, nil},
	} {
		t.Run(name, func(t *testing.T) {
			f := addSpace
			f.Files = tc.files
			got := Apply(f, fresh(), kr, accept)
			if !slices.Equal(got, tc.want) {
				t.Errorf("Apply = %v, want %v", got, tc.want)
			}
		})
	}
}

// Without the KR text of a unit the check cannot be made.
func TestRewriteLeavesUnitsWithoutKRAlone(t *testing.T) {
	kr := openKR(t, map[string][]string{storyFile: {"대사 0"}})
	gone := key("StoryData/Gone.json", 0)
	moved := key(storyFile, 9)
	recs := map[string]store.Record{
		key(storyFile, 0): rec(key(storyFile, 0), "2級破裂強度", store.OriginClaude),
		gone:              rec(gone, "2級破裂強度", store.OriginClaude),
		moved:             rec(moved, "2級破裂強度", store.OriginClaude),
	}
	got := Apply(addSpace, recs, kr, accept)
	if want := []string{key(storyFile, 0)}; !slices.Equal(got, want) {
		t.Errorf("Apply = %v, want %v: a file or a line KR no longer has is left alone", got, want)
	}

	recs = map[string]store.Record{key(storyFile, 0): rec(key(storyFile, 0), "2級破裂強度", store.OriginClaude)}
	if got := Apply(addSpace, recs, nil, accept); len(got) != 0 || recs[key(storyFile, 0)].ZH != "2級破裂強度" {
		t.Errorf("Apply without KR = %v, %+v; want nothing changed", got, recs)
	}
}

func TestRewriteApplyReturnsSortedKeys(t *testing.T) {
	lines := make([]string, 30)
	recs := map[string]store.Record{}
	for i := range lines {
		lines[i] = "대사"
		recs[key(storyFile, i)] = rec(key(storyFile, i), "2級破裂強度", store.OriginClaude)
	}
	got := Apply(addSpace, recs, openKR(t, map[string][]string{storyFile: lines}), accept)
	if len(got) != 30 || !slices.IsSorted(got) {
		t.Errorf("Apply = %v, want 30 keys in sorted order", got)
	}
}

func TestRewriteApplyWithoutACheck(t *testing.T) {
	kr := openKR(t, map[string][]string{storyFile: {"대사"}})
	recs := map[string]store.Record{key(storyFile, 0): rec(key(storyFile, 0), "2級破裂強度", store.OriginClaude)}
	if got := Apply(addSpace, recs, kr, nil); len(got) != 1 {
		t.Errorf("Apply with a nil check = %v, want the unit rewritten", got)
	}
}

const goodRewrite = `reason: 關鍵字名稱後加半形空格
files:
  - StoryData/**
  - Bufs*.json
origins: [paratranz, claude]
pattern: '(\d+級破裂)(強度)'
replace: '$1 $2'
kr_pattern: '강도'
`

func TestLoadRewriteFiles(t *testing.T) {
	root := t.TempDir()
	if files, err := LoadFiles(root); err != nil || len(files) != 0 {
		t.Errorf("LoadFiles without the directory = %+v, %v; want none", files, err)
	}
	writeFile(t, filepath.Join(root, Dir, ".gitkeep"), "")
	writeFile(t, filepath.Join(root, Dir, "2026-11-06-b.yaml"), "reason: 刪掉多餘的空格\nfiles: ['**/x.json', 'A/**']\npattern: ' +$'\nreplace: ''\n")
	writeFile(t, filepath.Join(root, Dir, "2026-11-05-a.yaml"), goodRewrite)
	writeFile(t, filepath.Join(root, Dir, "notes.md"), "not a rewrite")
	writeFile(t, filepath.Join(root, Dir, "old.yml"), "not read")

	files, err := LoadFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []File{
		{
			ID: "2026-11-05-a", Reason: "關鍵字名稱後加半形空格",
			Files: []string{"StoryData/**", "Bufs*.json"}, Origins: []string{"paratranz", "claude"},
			Pattern: `(\d+級破裂)(強度)`, Replace: "$1 $2", KRPattern: "강도",
		},
		// No origins in the file: Origins stays empty, Apply uses the default. An
		// empty replace deletes what the pattern matches.
		{ID: "2026-11-06-b", Reason: "刪掉多餘的空格", Files: []string{"**/x.json", "A/**"}, Pattern: " +$", Replace: ""},
	}
	if !reflect.DeepEqual(files, want) {
		t.Errorf("LoadFiles =\n %+v\nwant\n %+v", files, want)
	}
}

// rewriteFileProblems are files that cannot be applied, with what the error
// must say. LoadFiles stops at the first of them, Lint lists them all.
var rewriteFileProblems = map[string]struct{ text, want string }{
	"not a mapping":   {"- a\n", "mapping"},
	"empty":           {"", "reason"},
	"unknown key":     {goodRewrite + "foo: 1\n", "foo"},
	"misspelled key":  {strings.Replace(goodRewrite, "kr_pattern", "kr-pattern", 1), "kr-pattern"},
	"no reason":       {strings.Replace(goodRewrite, "reason: 關鍵字名稱後加半形空格\n", "", 1), "reason"},
	"blank reason":    {strings.Replace(goodRewrite, "reason: 關鍵字名稱後加半形空格", "reason: ' '", 1), "reason"},
	"no files":        {strings.Replace(goodRewrite, "files:\n  - StoryData/**\n  - Bufs*.json\n", "", 1), "files"},
	"empty files":     {strings.Replace(goodRewrite, "files:\n  - StoryData/**\n  - Bufs*.json\n", "files: []\n", 1), "files"},
	"malformed glob":  {strings.Replace(goodRewrite, "Bufs*.json", "Bufs[.json", 1), "Bufs[.json"},
	"unknown origin":  {strings.Replace(goodRewrite, "claude]", "robot]", 1), "robot"},
	"no pattern":      {strings.Replace(goodRewrite, "pattern: '(\\d+級破裂)(強度)'\n", "", 1), "pattern"},
	"empty pattern":   {strings.Replace(goodRewrite, "pattern: '(\\d+級破裂)(強度)'", "pattern: ''", 1), "pattern"},
	"bad pattern":     {strings.Replace(goodRewrite, `'(\d+級破裂)(強度)'`, `'(\d+級破裂'`, 1), "pattern"},
	"no replace":      {strings.Replace(goodRewrite, "replace: '$1 $2'\n", "", 1), "replace"},
	"bad kr_pattern":  {strings.Replace(goodRewrite, "'강도'", "'(강도'", 1), "kr_pattern"},
	"files is a text": {strings.Replace(goodRewrite, "files:\n  - StoryData/**\n  - Bufs*.json\n", "files: StoryData/**\n", 1), "files"},
	"two documents":   {goodRewrite + "---\nreason: x\n", "document"},
}

// regexp.ReplaceAllString reads a reference to a group that does not exist as
// empty text, which would delete text from every translation the pattern
// matches; the files are refused instead.
func TestRewriteReplaceReferences(t *testing.T) {
	file := func(pattern, replace string) string {
		return "reason: r\nfiles: [A/**]\npattern: '" + pattern + "'\nreplace: '" + replace + "'\n"
	}
	for _, tc := range []struct {
		pattern, replace string
		bad              string // the group named in the problem; empty when the file is fine
	}{
		{`(\d+級破裂)(強度)`, "$1 $2", ""},
		{`(\d+級破裂)(強度)`, "${1} ${2}", ""},
		{`(\d+級破裂)(強度)`, "$0", ""},
		{`(\d+級破裂)(強度)`, "$$1", ""}, // a dollar sign and a 1
		{`(\d+級破裂)(強度)`, "價格$", ""}, // a dollar sign
		{`(\d+級破裂)(強度)`, "$ 1", ""}, // no name follows
		{`(\d+級破裂)(強度)`, "${1", ""}, // not closed: text
		{`(\d+級破裂)(強度)`, "", ""},    // delete what matches
		{`(?P<n>\d+)級`, "$n 級", ""},
		{`(?P<n>\d+)級`, "${n}級", ""},
		{`(?P<n>\d+)級`, "$1 級", ""},
		{`(\d+級破裂)(強度)`, "$1 $3", "3"},
		{`(\d+級破裂)(強度)`, "$3", "3"},
		{`\d+級`, "$1", "1"},    // no group at all
		{`(\d+)`, "$1級", "1級"}, // the group "1級"
		{`(\d+)`, "$01", "01"}, // leading zeros make a name
		{`(?P<n>\d+)`, "$m", "m"},
		{`(\d+)`, "${2}級", "2"},
	} {
		_, problems := parse([]byte(file(tc.pattern, tc.replace)))
		switch {
		case tc.bad == "" && len(problems) != 0:
			t.Errorf("pattern %s, replace %q: problems %v, want none", tc.pattern, tc.replace, problems)
		case tc.bad != "" && (len(problems) != 1 || !strings.Contains(problems[0], `group "`+tc.bad+`"`)):
			t.Errorf("pattern %s, replace %q: problems %v, want one about the group %q", tc.pattern, tc.replace, problems, tc.bad)
		}
	}
}

func TestLoadRewriteFilesRejectsBadFiles(t *testing.T) {
	for name, tc := range rewriteFileProblems {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, Dir, "2026-11-05-x.yaml"), tc.text)
			files, err := LoadFiles(root)
			if err == nil || !strings.Contains(err.Error(), "2026-11-05-x.yaml") || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("LoadFiles = %+v, %v; want an error naming the file and mentioning %q", files, err, tc.want)
			}
		})
	}
}

func TestLintRewriteFiles(t *testing.T) {
	root := t.TempDir()
	if got := Lint(root); len(got) != 0 {
		t.Errorf("Lint without the directory = %v, want nothing", got)
	}
	writeFile(t, filepath.Join(root, Dir, ".gitkeep"), "")
	writeFile(t, filepath.Join(root, Dir, "2026-11-01-good.yaml"), goodRewrite)
	if got := Lint(root); len(got) != 0 {
		t.Errorf("Lint of a good file = %v, want nothing", got)
	}

	for name, tc := range rewriteFileProblems {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			const rel = Dir + "/2026-11-05-x.yaml"
			writeFile(t, filepath.Join(root, rel), tc.text)
			got := Lint(root)
			if len(got) == 0 || got[0].File != rel || !strings.Contains(got[0].Message, tc.want) {
				t.Errorf("Lint = %+v, want a problem in %s mentioning %q", got, rel, tc.want)
			}
		})
	}

	// Every problem is listed, not only the first.
	root = t.TempDir()
	writeFile(t, filepath.Join(root, Dir, "2026-11-05-x.yaml"), "foo: 1\nfiles: ['[a']\npattern: '('\nkr_pattern: ')'\norigins: [robot]\n")
	var msgs []string
	for _, is := range Lint(root) {
		msgs = append(msgs, is.Message)
	}
	all := strings.Join(msgs, "\n")
	for _, want := range []string{"foo", "reason", "[a", "pattern", "kr_pattern", "robot", "replace"} {
		if !strings.Contains(all, want) {
			t.Errorf("Lint messages %q lack %q", all, want)
		}
	}
}
