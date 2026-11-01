package requeue

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/store"
)

func TestWriteRequeueFileAndLoadIt(t *testing.T) {
	root := t.TempDir()
	sel := Selector{
		Keys:    []string{key(storyFile, 3)},
		Files:   []string{"StoryData/S1*", "Bufs*.json"},
		Origins: []string{store.OriginClaude, store.OriginParatranz},
	}
	const reason = "稱呼改用「您」"
	path, err := WriteFile(root, "2026-11-05", "honglu", sel, reason)
	if err != nil {
		t.Fatal(err)
	}
	if path != "rules/requeue/2026-11-05-honglu.yaml" {
		t.Errorf("path = %q, want it relative to the root, with forward slashes", path)
	}
	want := "reason: 稱呼改用「您」\n" +
		"keys:\n  - StoryData/S001A.json|/dataList/3/content\n" +
		"files:\n  - StoryData/S1*\n  - Bufs*.json\n" +
		"origins:\n  - claude\n  - paratranz\n"
	if got := readFileT(t, filepath.Join(root, filepath.FromSlash(path))); got != want {
		t.Errorf("file =\n%s\nwant\n%s", got, want)
	}

	files, err := LoadFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("LoadFiles = %+v, want one file", files)
	}
	if f := files[0]; f.ID != "2026-11-05-honglu" || f.Reason != reason || !reflect.DeepEqual(f.Selector, sel) {
		t.Errorf("file = %+v, want id 2026-11-05-honglu, reason %q and selector %+v", f, reason, sel)
	}
}

func readFileT(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestWriteRequeueFileOmitsWhatIsEmpty(t *testing.T) {
	root := t.TempDir()
	path, err := WriteFile(root, "2026-11-05", "x", Selector{Files: []string{"Bufs*.json"}}, "r")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := readFileT(t, filepath.Join(root, filepath.FromSlash(path))), "reason: r\nfiles:\n  - Bufs*.json\n"; got != want {
		t.Errorf("file = %q, want %q", got, want)
	}
}

// Whatever a person types in --reason or --file must come back as it was
// written, or the plan would apply something else than what was asked.
func TestWriteRequeueFileRoundTripsAwkwardText(t *testing.T) {
	texts := []string{
		" leading space", "trailing space ", "two\nlines", "\nstarts with a newline", "tab\there",
		`"quoted"`, "it's", "# not a comment", "key: value", "- dash", "* star", "| pipe", "> fold", "& anchor",
		"null", "true", "123", "1e3", "~", "!tag", "@at", "`tick", "%percent", `back\slash`, "[a]", "{b}", "a, b", "日本語：「您」",
	}
	for _, text := range texts {
		root := t.TempDir()
		sel := Selector{Files: []string{"Story*.json"}, Origins: []string{store.OriginClaude}}
		if _, err := WriteFile(root, "2026-11-05", "x", sel, text); err != nil {
			t.Errorf("WriteFile with reason %q: %v", text, err)
			continue
		}
		files, err := LoadFiles(root)
		if err != nil || len(files) != 1 || files[0].Reason != text || !reflect.DeepEqual(files[0].Selector, sel) {
			t.Errorf("reason %q came back as %+v, %v", text, files, err)
		}
	}

	// The same for the patterns of files.
	root := t.TempDir()
	sel := Selector{Files: []string{"*.json", "[ab]*", "# x", "a: b", "Story'Data/*", `Story"Data/*`}}
	if _, err := WriteFile(root, "2026-11-05", "x", sel, "r"); err != nil {
		t.Fatal(err)
	}
	files, err := LoadFiles(root)
	if err != nil || len(files) != 1 || !reflect.DeepEqual(files[0].Selector, sel) {
		t.Errorf("patterns came back as %+v, %v; want %+v", files, err, sel)
	}
}

func TestWriteRequeueFileRefusals(t *testing.T) {
	good := Selector{Files: []string{"Bufs*.json"}}
	for name, tc := range map[string]struct {
		date, topic, reason string
		sel                 Selector
		want                string
	}{
		"bad date":         {"2026-13-01", "x", "r", good, "date"},
		"bad topic":        {"2026-11-05", "a/b", "r", good, "topic"},
		"empty topic":      {"2026-11-05", "", "r", good, "topic"},
		"no reason":        {"2026-11-05", "x", "", good, "reason"},
		"blank reason":     {"2026-11-05", "x", " \n ", good, "reason"},
		"nothing selected": {"2026-11-05", "x", "r", Selector{}, "key or file"},
		"origins alone":    {"2026-11-05", "x", "r", Selector{Origins: []string{"claude"}}, "key or file"},
		"needs review":     {"2026-11-05", "x", "r", Selector{Files: []string{"a"}, NeedsReview: true}, "pending"},
		"leased":           {"2026-11-05", "x", "r", Selector{Files: []string{"a"}, Leased: true}, "pending"},
		"malformed key":    {"2026-11-05", "x", "r", Selector{Keys: []string{"nope"}}, "nope"},
		"unknown origin":   {"2026-11-05", "x", "r", Selector{Files: []string{"a"}, Origins: []string{"robot"}}, "robot"},
		"malformed glob":   {"2026-11-05", "x", "r", Selector{Files: []string{"[a"}}, "[a"},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			_, err := WriteFile(root, tc.date, tc.topic, tc.sel, tc.reason)
			if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("WriteFile = %v, want ErrInvalid mentioning %q", err, tc.want)
			}
			if entries, _ := os.ReadDir(root); len(entries) != 0 {
				t.Errorf("a refused call wrote %v", entries)
			}
		})
	}
}

// A file that was applied is named in applied.json: writing over it with
// other content would hide what was applied.
func TestWriteRequeueFileKeepsAnExistingFile(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteFile(root, "2026-11-05", "x", Selector{Files: []string{"a"}}, "first"); err != nil {
		t.Fatal(err)
	}
	_, err := WriteFile(root, "2026-11-05", "x", Selector{Files: []string{"b"}}, "second")
	if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "2026-11-05-x.yaml") {
		t.Fatalf("WriteFile of an existing file = %v, want ErrInvalid naming the file", err)
	}
	files, err := LoadFiles(root)
	if err != nil || len(files) != 1 || files[0].Reason != "first" {
		t.Errorf("LoadFiles = %+v, %v; want the first file untouched", files, err)
	}
	// Another topic or another day is another file.
	if _, err := WriteFile(root, "2026-11-05", "y", Selector{Files: []string{"b"}}, "second"); err != nil {
		t.Errorf("another topic: %v", err)
	}
	if _, err := WriteFile(root, "2026-11-06", "x", Selector{Files: []string{"b"}}, "third"); err != nil {
		t.Errorf("another date: %v", err)
	}
}

func TestLoadRequeueFilesRejectsUnknownKeys(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, Dir, "2026-11-05-x.yaml"), "reason: r\nfiles: [Bufs*.json]\nfoo: 1\n")
	_, err := LoadFiles(root)
	if err == nil || !strings.Contains(err.Error(), "2026-11-05-x.yaml") || !strings.Contains(err.Error(), "foo") {
		t.Errorf("LoadFiles = %v, want an error naming the file and the key foo", err)
	}
}

// requeueFileProblems are files that cannot be applied, with what the error
// must say. LoadFiles stops at the first of them, Lint lists them all.
var requeueFileProblems = map[string]struct{ text, want string }{
	"not a mapping":    {"- a\n- b\n", "mapping"},
	"empty":            {"", "reason"},
	"no reason":        {"files: [a]\n", "reason"},
	"blank reason":     {"reason: ' '\nfiles: [a]\n", "reason"},
	"nothing selected": {"reason: r\n", "key or file"},
	"origins alone":    {"reason: r\norigins: [claude]\n", "key or file"},
	"unknown origin":   {"reason: r\nfiles: [a]\norigins: [robot]\n", "robot"},
	"malformed key":    {"reason: r\nkeys: [nope]\n", "nope"},
	"malformed glob":   {"reason: r\nfiles: ['[a']\n", "[a"},
	"files is a text":  {"reason: r\nfiles: a\n", "files"},
	"unknown key":      {"reason: r\nfiles: [a]\nneeds_review: true\n", "needs_review"},
	"two documents":    {"reason: r\nfiles: [a]\n---\nreason: s\n", "document"},
}

func TestLoadRequeueFilesRejectsBadFiles(t *testing.T) {
	for name, tc := range requeueFileProblems {
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

func TestLoadRequeueFiles(t *testing.T) {
	root := t.TempDir()
	if files, err := LoadFiles(root); err != nil || len(files) != 0 {
		t.Errorf("LoadFiles without the directory = %+v, %v; want none", files, err)
	}
	writeFile(t, filepath.Join(root, Dir, ".gitkeep"), "")
	writeFile(t, filepath.Join(root, Dir, "2026-11-06-b.yaml"), "reason: b\nkeys: [\"Bufs.json|/dataList/0/name\"]\n")
	writeFile(t, filepath.Join(root, Dir, "2026-11-02-a.yaml"), "# why\nreason: a\nfiles: ['StoryData/**']\norigins: [paratranz, claude]\n")
	writeFile(t, filepath.Join(root, Dir, "notes.md"), "not a request")
	writeFile(t, filepath.Join(root, Dir, "old.yml"), "not read")

	files, err := LoadFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []File{
		{ID: "2026-11-02-a", Reason: "a", Selector: Selector{Files: []string{"StoryData/**"}, Origins: []string{"paratranz", "claude"}}},
		{ID: "2026-11-06-b", Reason: "b", Selector: Selector{Keys: []string{"Bufs.json|/dataList/0/name"}}},
	}
	if !reflect.DeepEqual(files, want) {
		t.Errorf("LoadFiles =\n %+v\nwant\n %+v", files, want)
	}
}

// A request file means what the flags of lct requeue mean.
func TestRequeueFileSelectsLikeTheFlags(t *testing.T) {
	set := openRaw(t, map[string]string{storyFile: dataList(3), otherFile: dataList(1)})
	st := newStore(t)
	saveRecords(t, st,
		rec(key(storyFile, 0), "譯文 0", store.OriginClaude),
		rec(key(storyFile, 1), "譯文 1", store.OriginHuman),
		rec(key(storyFile, 2), "譯文 2", store.OriginParatranz),
		rec(key(otherFile, 0), "其他", store.OriginClaude),
	)
	root := t.TempDir()
	if _, err := WriteFile(root, "2026-11-05", "story", Selector{Files: []string{"StoryData/*"}, Origins: []string{"claude", "paratranz"}}, hint); err != nil {
		t.Fatal(err)
	}
	files, err := LoadFiles(root)
	if err != nil || len(files) != 1 {
		t.Fatalf("LoadFiles = %+v, %v", files, err)
	}
	res, err := Apply(st, set.KR, frozen, files[0].Selector, files[0].Reason)
	if err != nil {
		t.Fatal(err)
	}
	checkResult(t, res, Result{Requeued: 2})
	var keys []string
	for _, it := range loadPending(t, st) {
		keys = append(keys, it.Key)
	}
	if want := []string{key(storyFile, 0), key(storyFile, 2)}; !slices.Equal(keys, want) {
		t.Errorf("pending = %v, want %v", keys, want)
	}
}

func TestLintRequeueFiles(t *testing.T) {
	root := t.TempDir()
	if got := Lint(root); len(got) != 0 {
		t.Errorf("Lint without the directory = %v, want nothing", got)
	}
	writeFile(t, filepath.Join(root, Dir, ".gitkeep"), "")
	writeFile(t, filepath.Join(root, Dir, "2026-11-01-good.yaml"), "reason: r\nfiles: [a]\n")
	if got := Lint(root); len(got) != 0 {
		t.Errorf("Lint of a good file = %v, want nothing", got)
	}

	for name, tc := range requeueFileProblems {
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
	writeFile(t, filepath.Join(root, Dir, "2026-11-05-x.yaml"), "foo: 1\nbar: 2\norigins: [robot]\n")
	var msgs []string
	for _, is := range Lint(root) {
		msgs = append(msgs, is.Message)
	}
	all := strings.Join(msgs, "\n")
	for _, want := range []string{"foo", "bar", "reason", "key or file", "robot"} {
		if !strings.Contains(all, want) {
			t.Errorf("Lint messages %q lack %q", all, want)
		}
	}

	// A file that Load would not read.
	writeFile(t, filepath.Join(root, Dir, "2026-11-06-y.yml"), "reason: r\nfiles: [a]\n")
	var named []string
	for _, is := range Lint(root) {
		named = append(named, is.File)
	}
	if !slices.Contains(named, Dir+"/2026-11-06-y.yml") {
		t.Errorf("Lint reported %v, want the .yml file among them", named)
	}
}
