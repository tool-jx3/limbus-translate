package site

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/store"
)

// exportInputs is a small pack of three files: two stories and one other. The
// stories come first in the testRules order, S2 before S10. It has 8 units,
// one of them a copy, so that the search corpus has 7. S10 has no JP file.
func exportInputs(t *testing.T) Inputs {
	t.Helper()
	const (
		s2     = "StoryData/S2.json"
		s10    = "StoryData/S10.json"
		skills = "Skills.json"
	)
	return Inputs{
		KR: writePack(t, "KR_", map[string]string{
			s10:    `{"dataList":[{"id":1,"teller":"가","content":"셋 <color=#f8c200>& 넷</color>"}]}`,
			s2:     `{"dataList":[{"id":1,"teller":"가","content":"하나"},{"id":2,"teller":"나","content":"둘"}]}`,
			skills: `{"dataList":[{"id":1,"name":"기술","model":"M1"}]}`,
		}),
		EN: writePack(t, "EN_", map[string]string{
			s10:    `{"dataList":[{"id":1,"teller":"A","content":"three"}]}`,
			s2:     `{"dataList":[{"id":2,"teller":"B","content":"two"},{"id":1,"teller":"A","content":"one"}]}`,
			skills: `{"dataList":[{"id":1,"name":"Skill","model":"M1"}]}`,
		}),
		JP: writePack(t, "JP_", map[string]string{
			s2:     `{"dataList":[{"id":1,"teller":"あ","content":"いち"},{"id":2,"teller":"い","content":"に"}]}`,
			skills: `{"dataList":[{"id":1,"name":"スキル","model":"M1"}]}`,
		}),
		ZH: writePack(t, "", map[string]string{
			s10:    `{"dataList":[{"id":1,"teller":"甲","content":"三 <color=#f8c200>& 四</color>"}]}`,
			s2:     `{"dataList":[{"id":1,"teller":"甲","content":"一"},{"id":2,"teller":"乙","content":"二"}]}`,
			skills: `{"dataList":[{"id":1,"name":"技能","model":"M1"}]}`,
		}),
		Records: recordsOf(
			record(s2, "/dataList/0/content", "一", store.OriginParatranz),
			record(s2, "/dataList/1/content", "二", store.OriginClaude, store.FlagNeedsReview),
			record(skills, "/dataList/0/name", "技能", store.OriginHuman),
			record(skills, "/dataList/0/model", "M1", store.OriginCopy),
		),
		Rules: testRules(),
	}
}

// fixtureFiles are the files of exportInputs in the order of the index.
var fixtureFiles = []string{"StoryData/S2.json", "StoryData/S10.json", "Skills.json"}

// listFiles returns the files under dir as slash-separated paths, sorted.
func listFiles(t *testing.T, dir string) []string {
	t.Helper()
	var names []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			rel, err := filepath.Rel(dir, p)
			if err != nil {
				return err
			}
			names = append(names, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(names)
	return names
}

// readJSON decodes the file rel under dir into v, and fails on a field v lacks.
func readJSON(t *testing.T, dir, rel string, v any) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		t.Fatalf("%s: %v", rel, err)
	}
}

func TestExportWritesSpecLayout(t *testing.T) {
	in := exportInputs(t)
	out := filepath.Join(t.TempDir(), "data")
	// Format, Langs, Files and Units are Export's to say, whatever it is given.
	res, err := Export(in, Meta{Format: 7, RawTag: "raw-20261029", RulesCommit: "rules1", TWCommit: "tw1", Release: "zh-TW-20261029.1",
		Langs: []string{"x"}, Files: 99, Units: 99}, out)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"files/Skills.json", "files/StoryData/S10.json", "files/StoryData/S2.json",
		"index.json", "meta.json",
		"search/en.json", "search/jp.json", "search/keys.json", "search/kr.json", "search/zh.json",
	}
	if got := listFiles(t, out); !slices.Equal(got, want) {
		t.Fatalf("files =\n%v\nwant\n%v", got, want)
	}

	var meta Meta
	readJSON(t, out, "meta.json", &meta)
	wantMeta := Meta{Format: 1, RawTag: "raw-20261029", RulesCommit: "rules1", TWCommit: "tw1", Release: "zh-TW-20261029.1",
		Langs: []string{"kr", "jp", "en", "zh"}, Files: 3, Units: 8}
	if !reflect.DeepEqual(meta, wantMeta) {
		t.Errorf("meta = %+v, want %+v", meta, wantMeta)
	}
	var keys map[string]json.RawMessage
	readJSON(t, out, "meta.json", &keys)
	var names []string
	for k := range keys {
		names = append(names, k)
	}
	slices.Sort(names)
	if want := []string{"files", "format", "langs", "raw_tag", "release", "rules_commit", "tw_commit", "units"}; !slices.Equal(names, want) {
		t.Errorf("meta.json keys = %v, want %v", names, want)
	}

	// The index lists the files in the order of OrderFiles; a family lists the
	// numbers of its files.
	var idx Index
	readJSON(t, out, "index.json", &idx)
	if got := OrderFiles(in.KR.Files(), in.Rules); !slices.Equal(got, fixtureFiles) {
		t.Fatalf("OrderFiles = %v, the fixture expects %v", got, fixtureFiles)
	}
	var paths []string
	for _, f := range idx.Files {
		paths = append(paths, f.Path)
	}
	if !slices.Equal(paths, fixtureFiles) {
		t.Errorf("index.files = %v, want %v", paths, fixtureFiles)
	}
	wantFamilies := []IndexFamily{{ID: "story", Files: []int{0, 1}}, {ID: "other", Files: []int{2}}}
	if !reflect.DeepEqual(idx.Families, wantFamilies) {
		t.Errorf("index.families = %+v, want %+v", idx.Families, wantFamilies)
	}

	// Each file holds what BuildFile says, and the index holds its summary.
	bytesTotal := int64(0)
	for _, rel := range listFiles(t, out) {
		fi, err := os.Stat(filepath.Join(out, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		bytesTotal += fi.Size()
	}
	for i, path := range fixtureFiles {
		wantFile, wantSum, err := BuildFile(in, path)
		if err != nil {
			t.Fatal(err)
		}
		var got File
		readJSON(t, out, "files/"+path, &got)
		if !reflect.DeepEqual(got, wantFile) {
			t.Errorf("files/%s = %+v, want %+v", path, got, wantFile)
		}
		if !reflect.DeepEqual(idx.Files[i], wantSum) {
			t.Errorf("index.files[%d] = %+v, want %+v", i, idx.Files[i], wantSum)
		}
	}

	if want := (Result{Out: out, Files: 3, Units: 8, SearchUnits: 7, Bytes: bytesTotal}); res != want {
		t.Errorf("result = %+v, want %+v", res, want)
	}
	var resKeys map[string]json.RawMessage
	b, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &resKeys); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"out", "files", "units", "search_units", "bytes"} {
		if _, ok := resKeys[k]; !ok {
			t.Errorf("result JSON has no %q: %s", k, b)
		}
	}
}

func TestExportSearchCorpus(t *testing.T) {
	in := exportInputs(t)
	out := filepath.Join(t.TempDir(), "data")
	if _, err := Export(in, Meta{}, out); err != nil {
		t.Fatal(err)
	}
	var idx Index
	readJSON(t, out, "index.json", &idx)
	var keys [][]any
	readJSON(t, out, "search/keys.json", &keys)
	corpus := map[string][]string{}
	for _, lang := range []string{"kr", "jp", "en", "zh"} {
		var texts []string
		readJSON(t, out, "search/"+lang+".json", &texts)
		if len(texts) != len(keys) {
			t.Fatalf("%s.json has %d strings, keys.json %d entries", lang, len(texts), len(keys))
		}
		corpus[lang] = texts
	}
	if len(keys) != 7 {
		t.Fatalf("%d entries, want 7: the 8 units without the copy", len(keys))
	}

	for n, key := range keys {
		if len(key) != 2 {
			t.Fatalf("keys[%d] = %v, want [file, pointer]", n, key)
		}
		fileNo, ok := key[0].(float64)
		pointer, ok2 := key[1].(string)
		if !ok || !ok2 || fileNo != float64(int(fileNo)) || int(fileNo) < 0 || int(fileNo) >= len(idx.Files) {
			t.Fatalf("keys[%d] = %v, want [index in index.files, pointer]", n, key)
		}
		// The strings of the entry are those of the unit it names.
		var f File
		readJSON(t, out, "files/"+idx.Files[int(fileNo)].Path, &f)
		u := unitAt(t, f, pointer)
		if u.O == store.OriginCopy {
			t.Errorf("keys[%d] = %v is a copy unit", n, key)
		}
		deref := func(p *string) string {
			if p == nil {
				return ""
			}
			return *p
		}
		got := [4]string{corpus["kr"][n], corpus["jp"][n], corpus["en"][n], corpus["zh"][n]}
		if want := [4]string{u.KR, deref(u.JP), deref(u.EN), u.ZH}; got != want {
			t.Errorf("entry %d (%v): kr jp en zh = %q, want %q", n, key, got, want)
		}
	}

	// The copy is not there; the order is that of the index and of the units.
	var got []string
	for _, key := range keys {
		got = append(got, idx.Files[int(key[0].(float64))].Path+"|"+key[1].(string))
	}
	want := []string{
		"StoryData/S2.json|/dataList/0/teller", "StoryData/S2.json|/dataList/0/content",
		"StoryData/S2.json|/dataList/1/teller", "StoryData/S2.json|/dataList/1/content",
		"StoryData/S10.json|/dataList/0/teller", "StoryData/S10.json|/dataList/0/content",
		"Skills.json|/dataList/0/name",
	}
	if !slices.Equal(got, want) {
		t.Errorf("keys = %v, want %v", got, want)
	}

	// S10 has no JP file: its entries are empty in jp.json, not missing.
	for n := 4; n <= 5; n++ {
		if corpus["jp"][n] != "" || corpus["kr"][n] == "" || corpus["en"][n] == "" {
			t.Errorf("entry %d: jp %q kr %q en %q, want an empty jp and the other texts", n, corpus["jp"][n], corpus["kr"][n], corpus["en"][n])
		}
	}
}

func TestExportDoesNotEscapeHTML(t *testing.T) {
	out := filepath.Join(t.TempDir(), "data")
	if _, err := Export(exportInputs(t), Meta{}, out); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"files/StoryData/S10.json", "search/kr.json", "search/zh.json"} {
		b, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "<color=#f8c200>") || !strings.Contains(string(b), "&") || strings.Contains(string(b), "\\u003c") || strings.Contains(string(b), "\\u0026") {
			t.Errorf("%s escapes < or &: %s", rel, b)
		}
		if bytes.HasPrefix(b, []byte("\xef\xbb\xbf")) {
			t.Errorf("%s starts with a BOM", rel)
		}
		if bytes.Contains(b, []byte("\n ")) || bytes.Contains(b, []byte("  ")) {
			t.Errorf("%s is indented: %s", rel, b)
		}
	}
}

func TestExportDeterministic(t *testing.T) {
	in := exportInputs(t)
	meta := Meta{RawTag: "raw-20261029", RulesCommit: "r", TWCommit: "t"}
	a, b := filepath.Join(t.TempDir(), "a"), filepath.Join(t.TempDir(), "b")
	resA, err := Export(in, meta, a)
	if err != nil {
		t.Fatal(err)
	}
	resB, err := Export(in, meta, b)
	if err != nil {
		t.Fatal(err)
	}
	filesA, filesB := listFiles(t, a), listFiles(t, b)
	if !slices.Equal(filesA, filesB) {
		t.Fatalf("files differ: %v / %v", filesA, filesB)
	}
	for _, rel := range filesA {
		x, err := os.ReadFile(filepath.Join(a, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		y, err := os.ReadFile(filepath.Join(b, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(x, y) {
			t.Errorf("%s differs between two exports", rel)
		}
	}
	resA.Out, resB.Out = "", ""
	if resA != resB {
		t.Errorf("results differ: %+v / %+v", resA, resB)
	}
}

func TestExportClearsOutDir(t *testing.T) {
	out := filepath.Join(t.TempDir(), "data")
	for _, rel := range []string{"stale.json", "files/Old/Gone.json", "search/old.json"} {
		p := filepath.Join(out, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("stale"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Export(exportInputs(t), Meta{}, out); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"stale.json", "files/Old/Gone.json", "search/old.json", "files/Old"} {
		if _, err := os.Stat(filepath.Join(out, filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Errorf("%s is still there (%v)", rel, err)
		}
	}
	if got := len(listFiles(t, out)); got != 10 {
		t.Errorf("%d files, want the 10 of the layout", got)
	}
}

func TestExportLeavesNoTemporaryFiles(t *testing.T) {
	out := filepath.Join(t.TempDir(), "data")
	if _, err := Export(exportInputs(t), Meta{}, out); err != nil {
		t.Fatal(err)
	}
	for _, rel := range listFiles(t, out) {
		if !strings.HasSuffix(rel, ".json") || strings.Contains(rel, ".tmp") {
			t.Errorf("unexpected file %s", rel)
		}
	}
}

func TestExportRefusesWhatItCannotExport(t *testing.T) {
	t.Run("a file the built pack lacks", func(t *testing.T) {
		in := exportInputs(t)
		in.ZH = writePack(t, "", map[string]string{
			"StoryData/S2.json":  `{"dataList":[]}`,
			"StoryData/S10.json": `{"dataList":[]}`,
		})
		_, err := Export(in, Meta{}, filepath.Join(t.TempDir(), "data"))
		if err == nil || !strings.Contains(err.Error(), "Skills.json") {
			t.Errorf("err = %v, want one that names Skills.json", err)
		}
	})
	t.Run("no directory", func(t *testing.T) {
		if _, err := Export(exportInputs(t), Meta{}, ""); err == nil {
			t.Error("Export to \"\": want an error, not a clearing of the working directory")
		}
	})
	t.Run("a file in place of the directory", func(t *testing.T) {
		out := filepath.Join(t.TempDir(), "data")
		if err := os.WriteFile(out, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Export(exportInputs(t), Meta{}, out); err == nil {
			t.Error("Export to a file: want an error")
		}
	})
}
