package fields

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/align"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// writeFile writes content to root/rel, creating directories.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// manualYAML is the repo's real manual rule file, so the tests also pin its
// content.
func manualYAML(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../rules/fields.manual.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// loadRules loads the manual rules plus the given generated file ("" for
// none) from a temporary repo root.
func loadRules(t *testing.T, generated string) *Rules {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, "rules/fields.manual.yaml", manualYAML(t))
	if generated != "" {
		writeFile(t, root, "rules/fields.yaml", generated)
	}
	r, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestGroup(t *testing.T) {
	cases := []struct{ file, want string }{
		{"BattleSpeechBubbleDlg-a1c10p3.json", "BattleSpeechBubbleDlg"},
		{"StoryData/S001A.json", "StoryData"},
		{"PersonalityVoiceDlg/PersonalityVoiceDlg_01.json", "PersonalityVoiceDlg"},
		{"Items-a1c7p1.json", "Items"},
		{"Skills_personality-08.json", "Skills"},
		{"Bufs.json", "Bufs"},
		{"Sample.json", "Sample"},
		{"2024Event.json", "2024Event"}, // nothing before the first digit: keep the whole name
	}
	for _, c := range cases {
		if got := Group(c.file); got != c.want {
			t.Errorf("Group(%q) = %q, want %q", c.file, got, c.want)
		}
	}
}

func TestClassify(t *testing.T) {
	r := loadRules(t, "")
	same := &align.LangAligned{Value: "......", Method: align.MethodPointer}
	other := &align.LangAligned{Value: "Hmm...", Method: align.MethodPointer}

	cases := []struct {
		name   string
		unit   units.Unit
		en, jp *align.LangAligned
		want   Decision
	}{
		{"model is an identifier",
			units.Unit{File: "Skills-a1.json", Field: "model", KR: "Dante"},
			nil, nil, Decision{Reason: ReasonIdentifierField}},
		{"id is an identifier even when text differs between languages",
			units.Unit{File: "StoryData/S001A.json", Field: "id", KR: "A1"},
			other, other, Decision{Reason: ReasonIdentifierField}},
		{"bubble desc is a trigger label",
			units.Unit{File: "BattleSpeechBubbleDlg-a1c9p1.json", Field: "desc", KR: "전투 시작"},
			other, other, Decision{Reason: ReasonGroupField}},
		{"voice desc is a title and is translated",
			units.Unit{File: "PersonalityVoiceDlg/PersonalityVoiceDlg_01.json", Field: "desc", KR: "인격 획득"},
			other, other, Decision{Translate: true}},
		{"bubble content is translated",
			units.Unit{File: "BattleSpeechBubbleDlg-a1c9p1.json", Field: "dlg", KR: "간다!"},
			other, other, Decision{Translate: true}},
		{"same value in all three languages",
			units.Unit{File: "StoryData/S001A.json", Field: "content", KR: "......"},
			same, same, Decision{Reason: ReasonIdenticalValue}},
		{"same value but EN differs",
			units.Unit{File: "StoryData/S001A.json", Field: "content", KR: "......"},
			other, same, Decision{Translate: true}},
		{"same value but JP differs",
			units.Unit{File: "StoryData/S001A.json", Field: "content", KR: "......"},
			same, other, Decision{Translate: true}},
		{"same value but no JP to compare",
			units.Unit{File: "StoryData/S001A.json", Field: "content", KR: "......"},
			same, nil, Decision{Translate: true}},
		{"same value but no EN to compare",
			units.Unit{File: "StoryData/S001A.json", Field: "content", KR: "......"},
			nil, same, Decision{Translate: true}},
		{"dash is empty",
			units.Unit{File: "StoryData/S001A.json", Field: "content", KR: "-"},
			other, other, Decision{Reason: ReasonEmpty}},
		{"empty string",
			units.Unit{File: "StoryData/S001A.json", Field: "content", KR: ""},
			nil, nil, Decision{Reason: ReasonEmpty}},
		{"whitespace only",
			units.Unit{File: "StoryData/S001A.json", Field: "content", KR: " \t\n"},
			nil, nil, Decision{Reason: ReasonEmpty}},
		{"source comment",
			units.Unit{File: "StoryData/S001A.json", Field: "content", KR: "// 효과음"},
			other, other, Decision{Reason: ReasonSourceComment}},
		{"SE source comment",
			units.Unit{File: "StoryData/S001A.json", Field: "content", KR: "SE // 문 닫는 소리"},
			other, other, Decision{Reason: ReasonSourceComment}},
		{"source comment after leading space",
			units.Unit{File: "StoryData/S001A.json", Field: "content", KR: "  // 효과음"},
			other, other, Decision{Reason: ReasonSourceComment}},
		{"a slash inside the text is not a comment",
			units.Unit{File: "StoryData/S001A.json", Field: "content", KR: "그건 // 아니야"},
			other, other, Decision{Translate: true}},
		{"item desc that asks for name only",
			units.Unit{File: "Items-a1c7p1.json", Field: "desc", KR: "name만 번역"},
			other, other, Decision{Reason: ReasonNameOnly}},
		{"item desc with spaces in the note",
			units.Unit{File: "Items-a1c7p1.json", Field: "desc", KR: "name 만 번역해 주세요"},
			other, other, Decision{Reason: ReasonNameOnly}},
		{"name-only applies to the file name without its directory",
			units.Unit{File: "Sub/Items-a1c7p1.json", Field: "desc", KR: "name만 번역"},
			other, other, Decision{Reason: ReasonNameOnly}},
		{"ordinary item desc",
			units.Unit{File: "Items-a1c7p1.json", Field: "desc", KR: "회복 아이템"},
			other, other, Decision{Translate: true}},
		{"name-only note outside Items files",
			units.Unit{File: "Skills-a1.json", Field: "desc", KR: "name만 번역"},
			other, other, Decision{Translate: true}},
		{"name-only note in another field",
			units.Unit{File: "Items-a1c7p1.json", Field: "name", KR: "name만 번역"},
			other, other, Decision{Translate: true}},
		{"notinclude file",
			units.Unit{File: "Foo_notinclude.json", Field: "content", KR: "번역 대상"},
			other, other, Decision{Reason: ReasonNotInclude}},
		{"notinclude in a directory name",
			units.Unit{File: "notinclude/Foo.json", Field: "content", KR: "번역 대상"},
			other, other, Decision{Reason: ReasonNotInclude}},
		{"notinclude is not case sensitive",
			units.Unit{File: "Foo_NotInclude.json", Field: "content", KR: "번역 대상"},
			other, other, Decision{Reason: ReasonNotInclude}},
		{"plain content",
			units.Unit{File: "StoryData/S001A.json", Field: "content", KR: "안녕하세요"},
			other, other, Decision{Translate: true}},
		{"plain content with no references",
			units.Unit{File: "StoryData/S001A.json", Field: "content", KR: "안녕하세요"},
			nil, nil, Decision{Translate: true}},

		// First match wins.
		{"notinclude beats empty",
			units.Unit{File: "Foo_notinclude.json", Field: "content", KR: "-"},
			nil, nil, Decision{Reason: ReasonNotInclude}},
		{"notinclude beats identifier field",
			units.Unit{File: "Foo_notinclude.json", Field: "id", KR: "A1"},
			nil, nil, Decision{Reason: ReasonNotInclude}},
		{"empty beats identifier field",
			units.Unit{File: "Skills-a1.json", Field: "model", KR: ""},
			nil, nil, Decision{Reason: ReasonEmpty}},
		{"identifier field beats group field and identical value",
			units.Unit{File: "BattleSpeechBubbleDlg-a1.json", Field: "id", KR: "......"},
			same, same, Decision{Reason: ReasonIdentifierField}},
		{"group field beats source comment",
			units.Unit{File: "BattleSpeechBubbleDlg-a1.json", Field: "desc", KR: "// x"},
			nil, nil, Decision{Reason: ReasonGroupField}},
		{"source comment beats identical value",
			units.Unit{File: "StoryData/S001A.json", Field: "content", KR: "// x"},
			&align.LangAligned{Value: "// x"}, &align.LangAligned{Value: "// x"}, Decision{Reason: ReasonSourceComment}},
		{"name-only beats identical value",
			units.Unit{File: "Items-a1.json", Field: "desc", KR: "name만 번역"},
			&align.LangAligned{Value: "name만 번역"}, &align.LangAligned{Value: "name만 번역"}, Decision{Reason: ReasonNameOnly}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := r.Classify(c.unit, c.en, c.jp); got != c.want {
				t.Errorf("Classify = %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestLoadMergesGeneratedAndManual(t *testing.T) {
	generated := `# Generated by lct fields learn.
identifier_fields:
  - key
  - usage
group_fields:
  Skills:
    - tag
`
	r := loadRules(t, generated)
	unit := func(file, field string) units.Unit {
		return units.Unit{File: file, Field: field, KR: "텍스트"}
	}
	for _, c := range []struct {
		name string
		unit units.Unit
		want string
	}{
		{"learned identifier", unit("Foo-1.json", "key"), ReasonIdentifierField},
		{"manual identifier", unit("Foo-1.json", "codeName"), ReasonIdentifierField},
		{"learned group field", unit("Skills-a1.json", "tag"), ReasonGroupField},
		{"learned group field in another group", unit("Foo-1.json", "tag"), ""},
		{"manual group field", unit("BattleSpeechBubbleDlg-a1.json", "desc"), ReasonGroupField},
	} {
		if got := r.Classify(c.unit, nil, nil); got.Reason != c.want {
			t.Errorf("%s: Reason = %q, want %q", c.name, got.Reason, c.want)
		}
	}
}

func TestLoadWithoutGeneratedFile(t *testing.T) {
	r := loadRules(t, "") // rules/fields.yaml does not exist
	got := r.Classify(units.Unit{File: "Foo-1.json", Field: "key", KR: "텍스트"}, nil, nil)
	if !got.Translate {
		t.Errorf("key is only an identifier when learned, got %+v", got)
	}
	if got := r.Classify(units.Unit{File: "Foo-1.json", Field: "id", KR: "A"}, nil, nil); got.Reason != ReasonIdentifierField {
		t.Errorf("manual rules must apply without a generated file, got %+v", got)
	}
}

func TestLoadErrors(t *testing.T) {
	manual := manualYAML(t)
	cases := []struct {
		name, manual, generated string
		noManual                bool
		want                    string
	}{
		{name: "manual file missing", noManual: true, want: "fields.manual.yaml"},
		{name: "unknown key in manual", manual: manual + "extra: 1\n", want: "extra"},
		{name: "unknown key in generated", manual: manual, generated: "identifier_fields: []\nextra: 1\n", want: "extra"},
		{name: "bad name_only pattern", manual: strings.Replace(manual, `name\s*만\s*번역`, `name(`, 1), want: "name_only"},
		{name: "name_only without field", manual: "name_only: { file_glob: \"Items*\", pattern: x }\n", want: "name_only"},
		{name: "bad glob", manual: "name_only: { file_glob: \"[\", field: desc, pattern: x }\n", want: "name_only"},
		{name: "empty comment prefix", manual: "source_comment_prefixes: [\"\"]\n", want: "source_comment_prefixes"},
		{name: "empty identifier", manual: "identifier_fields: [\"\"]\n", want: "identifier_fields"},
		{name: "malformed yaml", manual: "identifier_fields: [a\n", want: "fields.manual.yaml"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			if !c.noManual {
				writeFile(t, root, "rules/fields.manual.yaml", c.manual)
			}
			if c.generated != "" {
				writeFile(t, root, "rules/fields.yaml", c.generated)
			}
			_, err := Load(root)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("Load error = %v, want one mentioning %q", err, c.want)
			}
			// A mistake in a rule file is the author's; a file that is not
			// there is not a mistake in its content.
			var re *RuleError
			if got := errors.As(err, &re); got == c.noManual {
				t.Errorf("errors.As(%v, *RuleError) = %v, want %v", err, got, !c.noManual)
			}
		})
	}
}

func TestEmptyNotIncludeSubstringDisablesTheCheck(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rules/fields.manual.yaml", "identifier_fields: [id]\n")
	r, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	got := r.Classify(units.Unit{File: "Foo.json", Field: "content", KR: "텍스트"}, nil, nil)
	if !got.Translate {
		t.Errorf("an unset notinclude_substring must not match every file, got %+v", got)
	}
}

// ---- Learn ----

// fixture builds a KR/EN/JP pack triple under a temporary directory. Each
// file is a dataList of objects {"id": n, <field>: <value>...}; the values
// come from one string per language.
type fixture struct {
	t    *testing.T
	root string
}

func newFixture(t *testing.T) *fixture { return &fixture{t: t, root: t.TempDir()} }

// row is one dataList element: field name -> values in KR, EN and JP. A value
// of absent leaves the field out of that language's file.
type row map[string][3]string

const absent = "\x00absent"

func (f *fixture) add(file string, rows []row) {
	f.t.Helper()
	for i, lang := range []string{"kr", "en", "jp"} {
		f.addLang(lang, file, rows, func(v [3]string) string { return v[i] })
	}
}

// addOnly writes file for the given languages only (indexes 0, 1, 2).
func (f *fixture) addOnly(file string, rows []row, langs ...int) {
	f.t.Helper()
	names := []string{"kr", "en", "jp"}
	for _, i := range langs {
		f.addLang(names[i], file, rows, func(v [3]string) string { return v[i] })
	}
}

func (f *fixture) addLang(lang, file string, rows []row, pick func([3]string) string) {
	f.t.Helper()
	var items []string
	for i, r := range rows {
		parts := []string{fmt.Sprintf(`"id": %d`, i+1)}
		for _, name := range slices.Sorted(maps.Keys(r)) {
			if v := pick(r[name]); v != absent {
				parts = append(parts, fmt.Sprintf("%q: %q", name, v))
			}
		}
		items = append(items, "{"+strings.Join(parts, ", ")+"}")
	}
	doc := `{"dataList": [` + strings.Join(items, ", ") + `]}`
	dir, name := filepath.Split(file)
	writeFile(f.t, filepath.Join(f.root, lang), dir+strings.ToUpper(lang)+"_"+name, doc)
}

func (f *fixture) learn() *Learned {
	f.t.Helper()
	set, err := pack.OpenRaw(f.root)
	if err != nil {
		f.t.Fatal(err)
	}
	l, err := Learn(set.KR, set.EN, set.JP)
	if err != nil {
		f.t.Fatal(err)
	}
	return l
}

// all is a value that is the same in the three languages.
func all(v string) [3]string { return [3]string{v, v, v} }

// differs is a value that is translated.
func differs(i int) [3]string {
	return [3]string{fmt.Sprintf("한국어%d", i), fmt.Sprintf("English %d", i), fmt.Sprintf("日本語%d", i)}
}

// rowsN builds n rows, using mk(i) for the cells of row i.
func rowsN(n int, mk func(i int) row) []row {
	rows := make([]row, n)
	for i := range rows {
		rows[i] = mk(i)
	}
	return rows
}

func TestLearnThresholds(t *testing.T) {
	f := newFixture(t)

	// Layer 1. "key" is identical in all three languages 3 times, spread over
	// two files; "short" only twice; "mixed" is identical 3 times but differs
	// once.
	f.add("Alpha-1.json", []row{
		{"key": all("k1"), "short": all("s1"), "mixed": all("m1"), "text": differs(1)},
		{"key": all("k2"), "short": all("s2"), "mixed": all("m2"), "text": differs(2)},
	})
	f.add("Beta-1.json", []row{
		{"key": all("k3"), "mixed": all("m3"), "text": differs(3)},
		{"mixed": differs(4), "text": differs(4)},
	})

	// A value of "-" is empty and does not count, whatever the languages say.
	f.add("Dash-1.json", rowsN(5, func(i int) row { return row{"blank": all("-")} }))

	// A unit with no aligned EN (or JP) text is left out of the count: it
	// neither breaks "gap" (3 identical occurrences) nor lifts "gap2" (only 2).
	f.add("Gap-1.json", []row{
		{"gap": all("a")}, {"gap": all("b")}, {"gap": all("c")},
		{"gap": [3]string{"d", absent, "d"}},
	})
	f.add("Gap-2.json", []row{
		{"gap2": all("a")}, {"gap2": all("b")},
		{"gap2": [3]string{"c", absent, "c"}},
		{"gap2": [3]string{"d", "d", absent}},
		{"gap2": [3]string{"e", absent, absent}},
	})

	// Layer 2. Each group has a "tag" field that is not an identifier overall
	// (the Mixed group translates it).
	f.add("GroupA-1.json", rowsN(20, func(i int) row { // 20 of 20
		return row{"tag": all(fmt.Sprintf("t%d", i))}
	}))
	f.add("GroupB-1.json", rowsN(20, func(i int) row { // 19 of 20 = 95%
		if i == 0 {
			return row{"tag": differs(i)}
		}
		return row{"tag": all(fmt.Sprintf("t%d", i))}
	}))
	f.add("GroupC-1.json", rowsN(19, func(i int) row { // 19 of 19, but fewer than 20
		return row{"tag": all(fmt.Sprintf("t%d", i))}
	}))
	f.add("GroupE-1.json", rowsN(20, func(i int) row { // 20 of 20, but "key" is layer 1
		return row{"key": all(fmt.Sprintf("e%d", i))}
	}))
	f.add("Mixed-1.json", rowsN(3, func(i int) row { return row{"tag": differs(i)} }))

	got := f.learn()

	if want := []string{"gap", "key"}; !reflect.DeepEqual(got.IdentifierFields, want) {
		t.Errorf("IdentifierFields = %v, want %v", got.IdentifierFields, want)
	}
	if want := map[string][]string{"GroupA": {"tag"}}; !reflect.DeepEqual(got.GroupFields, want) {
		t.Errorf("GroupFields = %v, want %v", got.GroupFields, want)
	}
}

func TestLearnCounted(t *testing.T) {
	f := newFixture(t)
	f.add("Count-1.json", []row{
		{"key": all("k1"), "text": differs(1)}, // 2 counted
		{"key": all("-"), "text": differs(2)},  // "-" is empty: 1 counted
		{"gap": [3]string{"d", absent, "d"}},   // no aligned EN: 0 counted
		{"gap": [3]string{"e", "e", absent}},   // no aligned JP: 0 counted
	})
	f.addOnly("KrOnly.json", rowsN(5, func(i int) row { return row{"key": all("x")} }), 0) // not in EN and JP
	if got, want := f.learn().Counted, 3; got != want {
		t.Errorf("Counted = %d, want %d", got, want)
	}

	// Packs that share no file count nothing.
	none := newFixture(t)
	none.addOnly("A.json", rowsN(3, func(i int) row { return row{"key": all("x")} }), 0)
	none.addOnly("B.json", rowsN(3, func(i int) row { return row{"key": all("x")} }), 1)
	none.addOnly("C.json", rowsN(3, func(i int) row { return row{"key": all("x")} }), 2)
	if got := none.learn().Counted; got != 0 {
		t.Errorf("Counted = %d, want 0", got)
	}
}

func TestLearnGroupRatioBoundary(t *testing.T) {
	f := newFixture(t)
	// 99 of 100 identical: exactly 99% qualifies.
	f.add("GroupD-1.json", rowsN(100, func(i int) row {
		if i == 0 {
			return row{"tag": differs(i), "label": differs(i)}
		}
		return row{"tag": all(fmt.Sprintf("t%d", i)), "label": differs(i)}
	}))
	// 98 of 100 identical: below 99%.
	f.add("GroupF-1.json", rowsN(100, func(i int) row {
		if i < 2 {
			return row{"tag": differs(i), "label": differs(i)}
		}
		return row{"tag": all(fmt.Sprintf("t%d", i)), "label": differs(i)}
	}))
	got := f.learn()
	if len(got.IdentifierFields) != 0 {
		t.Errorf("IdentifierFields = %v, want none", got.IdentifierFields)
	}
	want := map[string][]string{"GroupD": {"tag"}}
	if !reflect.DeepEqual(got.GroupFields, want) {
		t.Errorf("GroupFields = %v, want %v", got.GroupFields, want)
	}
}

func TestLearnGroupsAcrossFilesAndDirectories(t *testing.T) {
	f := newFixture(t)
	// The group is the first directory, and counts add up over its files.
	for i := 0; i < 4; i++ {
		f.add(fmt.Sprintf("Sub/File%d.json", i), rowsN(5, func(j int) row {
			return row{"tag": all(fmt.Sprintf("t%d-%d", i, j))}
		}))
	}
	f.add("Other-1.json", rowsN(1, func(int) row { return row{"tag": differs(1)} }))
	got := f.learn()
	want := map[string][]string{"Sub": {"tag"}}
	if !reflect.DeepEqual(got.GroupFields, want) {
		t.Errorf("GroupFields = %v, want %v", got.GroupFields, want)
	}
}

func TestLearnIgnoresFilesWithoutAllThreeLanguages(t *testing.T) {
	f := newFixture(t)
	f.add("Same-1.json", rowsN(3, func(i int) row { return row{"key": all(fmt.Sprintf("k%d", i))} }))
	f.addOnly("EnOnly.json", rowsN(5, func(i int) row { return row{"other": all("x")} }), 1)
	f.addOnly("KrOnly.json", rowsN(5, func(i int) row { return row{"other": all("x")} }), 0)
	got := f.learn()
	if want := []string{"key"}; !reflect.DeepEqual(got.IdentifierFields, want) {
		t.Errorf("IdentifierFields = %v, want %v", got.IdentifierFields, want)
	}
}

func TestLearnNilPack(t *testing.T) {
	f := newFixture(t)
	f.add("Same-1.json", rowsN(1, func(int) row { return row{"key": all("k")} }))
	set, err := pack.OpenRaw(f.root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Learn(set.KR, nil, set.JP); err == nil {
		t.Error("Learn with a nil pack must fail")
	}
}

func TestLearnBadJSON(t *testing.T) {
	f := newFixture(t)
	f.add("Same-1.json", rowsN(1, func(int) row { return row{"key": all("k")} }))
	writeFile(t, f.root, "kr/KR_Broken.json", "{")
	writeFile(t, f.root, "en/EN_Broken.json", "{}")
	writeFile(t, f.root, "jp/JP_Broken.json", "{}")
	set, err := pack.OpenRaw(f.root)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Learn(set.KR, set.EN, set.JP)
	if err == nil || !strings.Contains(err.Error(), "Broken.json") {
		t.Errorf("Learn error = %v, want one naming Broken.json", err)
	}
}

func TestLearnedYAML(t *testing.T) {
	l := &Learned{
		IdentifierFields: []string{"key", "usage"},
		GroupFields:      map[string][]string{"Skills": {"tag", "zone"}, "Items": {"code"}},
	}
	want := `# Generated by ` + "`lct fields learn`" + `. Do not edit; put manual rules in fields.manual.yaml.
identifier_fields:
  - key
  - usage
group_fields:
  Items:
    - code
  Skills:
    - tag
    - zone
`
	if got := string(l.YAML()); got != want {
		t.Errorf("YAML =\n%s\nwant\n%s", got, want)
	}

	empty := &Learned{}
	wantEmpty := `# Generated by ` + "`lct fields learn`" + `. Do not edit; put manual rules in fields.manual.yaml.
identifier_fields: []
group_fields: {}
`
	if got := string(empty.YAML()); got != wantEmpty {
		t.Errorf("empty YAML =\n%s\nwant\n%s", got, wantEmpty)
	}
}

func TestLearnedYAMLRoundTrip(t *testing.T) {
	root := t.TempDir()
	l := &Learned{
		IdentifierFields: []string{"key", "usage"},
		GroupFields:      map[string][]string{"Skills": {"tag"}},
	}
	writeFile(t, root, "rules/fields.yaml", string(l.YAML()))
	got, err := LoadLearned(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, l) {
		t.Errorf("LoadLearned = %+v, want %+v", got, l)
	}

	missing, err := LoadLearned(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(missing.IdentifierFields) != 0 || len(missing.GroupFields) != 0 {
		t.Errorf("a missing file must read as empty, got %+v", missing)
	}
}

func TestLearnedWithout(t *testing.T) {
	manual := loadRules(t, "") // identifiers id, model, codeName; BattleSpeechBubbleDlg desc
	l := &Learned{
		IdentifierFields: []string{"codeName", "key", "model"},
		GroupFields: map[string][]string{
			"Skills":                {"codeName", "tag"},
			"Items":                 {"id"},
			"BattleSpeechBubbleDlg": {"desc"},
		},
	}
	got := l.Without(manual)
	want := &Learned{
		IdentifierFields: []string{"key"},
		GroupFields:      map[string][]string{"Skills": {"tag"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Without = %+v, want %+v", got, want)
	}
	if len(l.IdentifierFields) != 3 {
		t.Errorf("Without must not modify its receiver, got %+v", l)
	}
}

func TestLearnedDiff(t *testing.T) {
	old := &Learned{
		IdentifierFields: []string{"key", "usage"},
		GroupFields:      map[string][]string{"Skills": {"tag"}, "Items": {"code"}},
	}
	cur := &Learned{
		IdentifierFields: []string{"imgStr", "key"},
		GroupFields:      map[string][]string{"Skills": {"tag", "zone"}},
	}
	got := cur.Diff(old)
	want := []Change{
		{Op: "add", Field: "imgStr"},
		{Op: "remove", Field: "usage"},
		{Op: "remove", Group: "Items", Field: "code"},
		{Op: "add", Group: "Skills", Field: "zone"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Diff = %+v, want %+v", got, want)
	}
	if d := cur.Diff(cur); len(d) != 0 {
		t.Errorf("Diff against itself = %+v, want none", d)
	}
	if s := got[0].String(); s != "+ identifier_fields: imgStr" {
		t.Errorf("Change.String = %q", s)
	}
	if s := got[2].String(); s != "- group_fields: Items.code" {
		t.Errorf("Change.String = %q", s)
	}
}

func TestLearnedEmpty(t *testing.T) {
	for _, c := range []struct {
		name string
		l    Learned
		want bool
	}{
		{"zero value", Learned{}, true},
		{"empty lists", Learned{IdentifierFields: []string{}, GroupFields: map[string][]string{"Items": {}}}, true},
		{"identifier field", Learned{IdentifierFields: []string{"id"}}, false},
		{"group field", Learned{GroupFields: map[string][]string{"Items": {"code"}}}, false},
	} {
		if got := c.l.Empty(); got != c.want {
			t.Errorf("%s: Empty() = %v, want %v", c.name, got, c.want)
		}
	}
}
