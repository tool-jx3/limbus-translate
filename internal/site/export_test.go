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

	"github.com/tool-jx3/limbus-translate/internal/gold"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/scope"
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
		"glossary/index.json", "gold.json", "index.json", "meta.json", "names.json",
		"review/notes.json", "review/scope.json", "review/suggestions.json", "review/todo.json", "review/warnings.json",
		"search/en.json", "search/jp.json", "search/keys.json", "search/kr.json", "search/zh.json",
	}
	if got := listFiles(t, out); !slices.Equal(got, want) {
		t.Fatalf("files =\n%v\nwant\n%v", got, want)
	}

	var meta Meta
	readJSON(t, out, "meta.json", &meta)
	wantMeta := Meta{Format: 2, RawTag: "raw-20261029", RulesCommit: "rules1", TWCommit: "tw1", Release: "zh-TW-20261029.1",
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
	if want := []string{"files", "format", "langs", "raw_tag", "release", "review", "rules_commit", "tw_commit", "units"}; !slices.Equal(names, want) {
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
	wantFamilies := []IndexFamily{{ID: "story", Files: []int{0, 1}, Glossary: "hard", Nodes: []string{"StoryData/"}}, {ID: "other", Files: []int{2}, Glossary: "hard", Nodes: []string{}}}
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

	if want := (Result{Out: out, Files: 3, Units: 8, SearchUnits: 7, Bytes: bytesTotal, GlossaryBytes: res.GlossaryBytes, ReviewMismatches: []string{}}); !reflect.DeepEqual(res, want) {
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
	for _, k := range []string{"out", "files", "units", "search_units", "bytes", "review", "review_mismatches", "notes_unplaced"} {
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
	if !reflect.DeepEqual(resA, resB) {
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
	if got := len(listFiles(t, out)); got != 18 {
		t.Errorf("%d files, want the 18 of the layout", got)
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

// exportReviewInputs is the fixture of the review tests with the pending list
// of one unit of A.json.
func exportReviewInputs(t *testing.T) (Inputs, []string) {
	t.Helper()
	in, files := reviewInputs(t)
	in.Review.Pending = []store.PendingItem{{Key: k(rvA, 1), Reason: "new"}}
	withBatch(in, k(rvA, 0), "b1")
	in.Review.Notes = []store.Note{noteOf(k(rvA, 0), "b1", rvKR, "a note")}
	// No unit renders 하나 so: a scope proposal (mixed).
	in.Rules.Glossary = []rules.Term{{KR: rvKR, ZH: "絕無此譯", Status: rules.StatusApproved}}
	return in, files
}

func TestExportWritesReviewData(t *testing.T) {
	in, _ := exportReviewInputs(t)
	want, err := BuildReview(in, OrderFiles(in.KR.Files(), in.Rules))
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "data")
	res, err := Export(in, Meta{RawTag: "t1"}, out)
	if err != nil {
		t.Fatal(err)
	}

	var meta Meta
	readJSON(t, out, "meta.json", &meta)
	wantCounts := countReview(want)
	if meta.Format != 2 || meta.Review != wantCounts || res.Review != wantCounts {
		t.Errorf("format %d, meta.review %+v, result.review %+v, want format 2 and %+v", meta.Format, meta.Review, res.Review, wantCounts)
	}
	if wantCounts.Suggestions == 0 || wantCounts.Warnings == 0 || wantCounts.Notes == 0 || wantCounts.Scope == 0 || len(want.Scope.Entries) == 0 {
		t.Fatalf("the fixture has too little to review: %+v", wantCounts)
	}
	if !reflect.DeepEqual(res.ReviewMismatches, want.Mismatches) || res.NotesUnplaced != want.NotesUnplaced {
		t.Errorf("result = %+v, want mismatches %v and %d unplaced notes", res, want.Mismatches, want.NotesUnplaced)
	}

	var sg struct{ Items []SuggestionItem }
	var wr struct{ Items []WarningItem }
	var nt struct{ Items []NoteItem }
	var td struct{ Sections []TodoSection }
	var sc scope.Report
	readJSON(t, out, "review/scope.json", &sc)
	readJSON(t, out, "review/suggestions.json", &sg)
	readJSON(t, out, "review/warnings.json", &wr)
	readJSON(t, out, "review/notes.json", &nt)
	readJSON(t, out, "review/todo.json", &td)
	for name, ok := range map[string]bool{
		"suggestions": reflect.DeepEqual(sg.Items, want.Suggestions),
		"warnings":    reflect.DeepEqual(wr.Items, want.Warnings),
		"notes":       reflect.DeepEqual(nt.Items, want.Notes),
		"todo":        reflect.DeepEqual(td.Sections, want.Todo),
		"scope":       reflect.DeepEqual(sc, withoutUnits(want.Scope)),
	} {
		if !ok {
			t.Errorf("review/%s.json differs from BuildReview", name)
		}
	}
	raw, err := os.ReadFile(filepath.Join(out, "review", "todo.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(raw), `{"sections":[`) {
		t.Errorf("todo.json = %.40s, want the key sections", raw)
	}
	raw, err = os.ReadFile(filepath.Join(out, "review", "warnings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(raw), `{"items":[`) {
		t.Errorf("warnings.json = %.40s, want the key items", raw)
	}

	// The units carry what the review says of them.
	seenRV, seenNT, seenPQ := 0, 0, 0
	for _, path := range []string{rvA, rvB} {
		var f File
		readJSON(t, out, "files/"+path, &f)
		for _, u := range f.Units {
			key := rvKey(path, u.P)
			if !slices.Equal(u.RV, want.Refs[key]) || u.NT != want.NoteText[key] || (u.PQ == 1) != want.Queued[key] {
				t.Errorf("%s: rv %v nt %q pq %d, want %v %q %v", key, u.RV, u.NT, u.PQ, want.Refs[key], want.NoteText[key], want.Queued[key])
			}
			if u.PQ != 0 && u.PQ != 1 {
				t.Errorf("%s: pq = %d, want 0 or 1", key, u.PQ)
			}
			seenRV += len(u.RV)
			if u.NT != "" {
				seenNT++
			}
			seenPQ += u.PQ
		}
	}
	if seenRV == 0 || seenNT != 1 || seenPQ != 1 {
		t.Errorf("units show %d refs, %d notes, %d queued, want some, 1 and 1", seenRV, seenNT, seenPQ)
	}

	// The same inputs give the same bytes.
	again := filepath.Join(t.TempDir(), "again")
	if _, err := Export(in, Meta{RawTag: "t1"}, again); err != nil {
		t.Fatal(err)
	}
	for _, rel := range listFiles(t, out) {
		x, _ := os.ReadFile(filepath.Join(out, filepath.FromSlash(rel)))
		y, _ := os.ReadFile(filepath.Join(again, filepath.FromSlash(rel)))
		if !bytes.Equal(x, y) {
			t.Errorf("%s differs between two exports", rel)
		}
	}
}

func TestExportWithoutReviewInputsWritesEmptyLists(t *testing.T) {
	in := exportInputs(t)
	out := filepath.Join(t.TempDir(), "data")
	if _, err := Export(in, Meta{}, out); err != nil {
		t.Fatal(err)
	}
	for rel, want := range map[string]string{
		"review/suggestions.json": `{"items":[]}`,
		"review/warnings.json":    `{"items":[]}`,
		"review/notes.json":       `{"items":[]}`,
		"review/todo.json":        `{"sections":[]}`,
		"review/scope.json":       `{"thresholds":{"min_n":3,"used":0.8,"not_used":0.3,"explained":3,"lost_units":20,"lost_share":0.1,"hard_rate":0.95,"hard_min_n":20},"entries":[],"proposals":[]}`,
	} {
		b, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.TrimSpace(string(b)); got != want {
			t.Errorf("%s = %s, want %s", rel, got, want)
		}
	}
	var meta Meta
	readJSON(t, out, "meta.json", &meta)
	if meta.Review != (ReviewCounts{}) {
		t.Errorf("meta.review = %+v, want zeros", meta.Review)
	}
	var f File
	readJSON(t, out, "files/StoryData/S2.json", &f)
	for _, u := range f.Units {
		if u.RV != nil || u.NT != "" || u.PQ != 0 {
			t.Errorf("unit %s has review marks without review inputs: %+v", u.P, u)
		}
	}
}

// withoutUnits is r as review/scope.json holds it: Report.Units is not part of
// it (json:"-").
func withoutUnits(r scope.Report) scope.Report {
	r.Units = nil
	return r
}

func TestIndexGroups(t *testing.T) {
	code := "07"
	r := &rules.Rules{Characters: []rules.Character{{ID: "faust", Name: "浮士德", Code: &code}}}
	build := func(paths ...string) []IndexGroup {
		var sums []FileSummary
		for _, p := range paths {
			n := rules.FileNode(p)
			s := FileSummary{Path: p, Top: n.Top, Sub: n.Sub}
			if tag, ok := r.FileTag(p); ok {
				s.Tag = &tag
			}
			sums = append(sums, s)
		}
		return buildGroups(sums)
	}
	got := build(
		"StoryData/S1037B.json", "StoryData/S937A.json", "StoryData/P10702.json",
		"Skills_Enemy-a.json", "Skills_Enemy-b.json", "Bufs-walpu7.json",
	)
	if len(got) != 3 {
		t.Fatalf("groups = %+v", got)
	}
	// By number of files, most first.
	if got[0].Node != "StoryData/" || got[1].Node != "Skills" || got[2].Node != "Bufs" {
		t.Fatalf("top order = %s %s %s", got[0].Node, got[1].Node, got[2].Node)
	}
	// S has two files and P one: P folds into the top, which leaves one sub,
	// so S is not shown either and the third level hangs under the top.
	story := got[0]
	if story.Kind != "top" || story.Label != "StoryData/" || len(story.Files) != 0 || len(story.Children) != 3 {
		t.Fatalf("story = %+v", story)
	}
	c0, c1, c2 := story.Children[0], story.Children[1], story.Children[2]
	if c0.Kind != "chapter" || c0.Node != "9" || c0.Label != "第 9 章" || !reflect.DeepEqual(c0.Files, []int{1}) {
		t.Errorf("chapter 9 = %+v", c0)
	}
	if c1.Kind != "chapter" || c1.Node != "10" || c1.Label != "第 10 章" || !reflect.DeepEqual(c1.Files, []int{0}) {
		t.Errorf("chapter 10 = %+v, a one-file third level is shown", c1)
	}
	if c2.Kind != "character" || c2.Node != "faust" || c2.Label != "浮士德" || !reflect.DeepEqual(c2.Files, []int{2}) {
		t.Errorf("character = %+v", c2)
	}
	if c0.Children == nil || len(c0.Children) != 0 {
		t.Errorf("leaf children = %#v, want []", c0.Children)
	}
	// Skills_Enemy is the only sub: not shown; the events a and b are.
	sk := got[1]
	if len(sk.Children) != 2 || sk.Children[0].Kind != "event" || sk.Children[1].Kind != "event" || len(sk.Files) != 0 {
		t.Errorf("Skills = %+v", sk)
	}
	if b := got[2]; len(b.Files) != 0 || len(b.Children) != 1 || b.Children[0].Kind != "event" || b.Children[0].Node != "walpu7" || !reflect.DeepEqual(b.Children[0].Files, []int{5}) {
		t.Errorf("Bufs = %+v", b)
	}

	// Two subs of two files each: both show, and a file without a tag stays
	// in its sub.
	two := build("Ui_A1.json", "Ui_A2.json", "Ui_B1.json", "Ui_B2.json", "Ui_B3.json")
	if len(two) != 1 || len(two[0].Files) != 0 || len(two[0].Children) != 2 {
		t.Fatalf("two subs = %+v", two)
	}
	if b, a := two[0].Children[0], two[0].Children[1]; b.Kind != "sub" || b.Node != "Ui_B" || !reflect.DeepEqual(b.Files, []int{2, 3, 4}) ||
		a.Kind != "sub" || a.Node != "Ui_A" || !reflect.DeepEqual(a.Files, []int{0, 1}) {
		t.Errorf("subs = %+v", two[0].Children)
	}
}

func TestIndexFamiliesCarryNodes(t *testing.T) {
	in := exportInputs(t)
	in.Rules.Routing.Families[0].Glossary = rules.GlossaryWarn
	dir := t.TempDir()
	if _, err := Export(in, Meta{}, dir); err != nil {
		t.Fatal(err)
	}
	var idx Index
	data, _ := os.ReadFile(filepath.Join(dir, "index.json"))
	if err := json.Unmarshal(data, &idx); err != nil {
		t.Fatal(err)
	}
	if len(idx.Families) != 2 {
		t.Fatalf("families = %+v", idx.Families)
	}
	if f := idx.Families[0]; f.Glossary != "warn" || !reflect.DeepEqual(f.Nodes, []string{"StoryData/"}) {
		t.Errorf("story = %+v", f)
	}
	if f := idx.Families[1]; f.Glossary != "hard" || f.Nodes == nil || len(f.Nodes) != 0 {
		t.Errorf("other = %+v, want glossary hard and nodes []", f)
	}
	if !strings.Contains(string(data), `"nodes":[]`) || !strings.Contains(string(data), `"groups":[`) {
		t.Errorf("index.json = %s", data)
	}
	if idx.Files[0].Top != "StoryData/" || idx.Files[0].Sub != "StoryData/S" {
		t.Errorf("file = %+v", idx.Files[0])
	}
}

func TestExportWritesGold(t *testing.T) {
	// Without a sample: {units: []}, and nothing else.
	in := exportInputs(t)
	out := filepath.Join(t.TempDir(), "data")
	if _, err := Export(in, Meta{}, out); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(out, "gold.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(b)); got != `{"units":[]}` {
		t.Errorf("gold.json without a sample = %s", got)
	}

	// With one: the units come in the order of the sample, with the texts of
	// the packs; stale is a translation that is not the one that was judged.
	in = exportInputs(t)
	// A structural fallback: the pack ships another text than the record holds.
	// The record is what gold set judged, so it is the translation of the unit.
	fb := "StoryData/S2.json|/dataList/0/content"
	in.Records[fb] = store.Record{Key: fb, KRSha: "00", ZH: "記錄譯文", Origin: store.OriginFallback, Since: "raw-20261029"}
	in.Gold = &gold.Sample{Seed: 7, PerFamily: 2, Units: []gold.Unit{
		{Key: "StoryData/S2.json|/dataList/1/content", Family: "story", Origin: "claude", Verdict: "wrong", Note: "n", JudgedZH: "舊"},
		{Key: "Skills.json|/dataList/0/name", Family: "other", Origin: "human", Verdict: "correct", JudgedZH: "技能"},
		{Key: fb, Family: "story", Origin: "fallback", Verdict: "correct", JudgedZH: "記錄譯文"},
		{Key: "StoryData/Gone.json|/dataList/0/content", Family: "story", Origin: "claude", Verdict: "correct", JudgedZH: "x"},
	}}
	out = filepath.Join(t.TempDir(), "data")
	if _, err := Export(in, Meta{}, out); err != nil {
		t.Fatal(err)
	}
	var got struct {
		PerFamily int `json:"per_family"`
		Units     []struct {
			Key, Path, Ptr, Family, Origin, KR, ZH, Verdict, Note string
			JP, EN                                                *string
			Stale                                                 bool
			Shipped                                               string
		}
		Missing int         `json:"missing"`
		Report  gold.Report `json:"report"`
	}
	b, err = os.ReadFile(filepath.Join(out, "gold.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.PerFamily != 2 || len(got.Units) != 3 {
		t.Fatalf("gold.json = %s", b)
	}
	u := got.Units[0]
	if u.Path != "StoryData/S2.json" || u.Ptr != "/dataList/1/content" || u.KR != "둘" || u.ZH != "二" ||
		u.EN == nil || *u.EN != "one" || u.JP == nil || *u.JP != "に" || u.Verdict != "wrong" || u.Note != "n" || !u.Stale || u.Origin != "claude" || u.Family != "story" {
		t.Errorf("unit 0 = %+v en=%v jp=%v", u, *u.EN, *u.JP)
	}
	if got.Units[1].Stale || got.Units[1].Path != "Skills.json" {
		t.Errorf("unit 1 = %+v", got.Units[1])
	}
	if got.Units[0].Shipped != "" || got.Units[1].Shipped != "" {
		t.Errorf("shipped is set where the pack holds the record: %+v %+v", got.Units[0], got.Units[1])
	}
	// The record is canonical: zh and stale follow it, the pack text is shipped.
	if u := got.Units[2]; u.ZH != "記錄譯文" || u.Stale || u.Shipped != "一" {
		t.Errorf("unit 2 = %+v, want zh from the record, not stale, shipped \"一\"", u)
	}
	// The unit that is not in the pack any more is left out of the list and
	// counted; the report counts the whole sample, as lct gold report does.
	if got.Missing != 1 || got.Report.Missing != 1 {
		t.Errorf("missing = %d, report.missing = %d, want 1", got.Missing, got.Report.Missing)
	}
	if got.Report.Total.Sampled != 4 || got.Report.Total.Stale != 2 || got.Report.Total.Correct != 3 {
		t.Errorf("report total = %+v", got.Report.Total)
	}
}
