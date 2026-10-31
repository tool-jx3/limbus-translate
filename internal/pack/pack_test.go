package pack

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const (
	krRoot = "../../testdata/pack/kr"
	enRoot = "../../testdata/pack/en"
)

// writeFiles creates the given files (slash paths relative to root) with
// content "{}".
func writeFiles(t *testing.T, root string, names ...string) {
	t.Helper()
	for _, name := range names {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestOpenStripsPrefix(t *testing.T) {
	p, err := Open(krRoot, "KR_")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Sample.json", "StoryData/S001A.json"}
	if got := p.Files(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Files() = %v, want %v", got, want)
	}
	if !p.Has("StoryData/S001A.json") {
		t.Error("Has(StoryData/S001A.json) = false, want true")
	}
	if p.Has("KR_Sample.json") || p.Has("StoryData/KR_S001A.json") || p.Has("Missing.json") {
		t.Error("Has accepted a name that is prefixed or absent")
	}

	en, err := Open(enRoot, "EN_")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := en.Files(), []string{"Sample.json"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("EN Files() = %v, want %v", got, want)
	}
}

func TestFilesIsACopy(t *testing.T) {
	p, err := Open(krRoot, "KR_")
	if err != nil {
		t.Fatal(err)
	}
	p.Files()[0] = "Changed.json"
	if got := p.Files()[0]; got != "Sample.json" {
		t.Fatalf("mutating the result of Files() changed the pack: %q", got)
	}
}

func TestOpenFiltersAndSorts(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root,
		"KR_b.json",
		"KR_a.json",
		"KR_notes.txt", // not json
		"other.json",   // no prefix
		"KR_dir/KR_c.json",
		"dir/KR_a-b.json",
		"dir/a/KR_x.json",
		"dir/plain.json", // no prefix
	)
	p, err := Open(root, "KR_")
	if err != nil {
		t.Fatal(err)
	}
	// The prefix is only stripped from the file name; sorting is by the
	// slash-separated relative path ("-" sorts before "/").
	want := []string{"KR_dir/c.json", "a.json", "b.json", "dir/a-b.json", "dir/a/x.json"}
	if got := p.Files(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Files() = %v, want %v", got, want)
	}
}

func TestOpenEmptyPrefix(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, "Sample.json", "StoryData/S001A.json", "README.md")
	p, err := Open(root, "")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Sample.json", "StoryData/S001A.json"}
	if got := p.Files(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Files() = %v, want %v", got, want)
	}
}

func TestOpenErrors(t *testing.T) {
	if _, err := Open(filepath.Join(t.TempDir(), "missing"), "KR_"); err == nil {
		t.Error("Open on a missing directory succeeded")
	}
	file := filepath.Join(t.TempDir(), "file.json")
	if err := os.WriteFile(file, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(file, "KR_"); err == nil {
		t.Error("Open on a regular file succeeded")
	}
}

func TestPath(t *testing.T) {
	p, err := Open(krRoot, "KR_")
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"Sample.json":          filepath.Join(krRoot, "KR_Sample.json"),
		"StoryData/S001A.json": filepath.Join(krRoot, "StoryData", "KR_S001A.json"),
	}
	for file, want := range cases {
		if got := p.Path(file); got != want {
			t.Errorf("Path(%q) = %q, want %q", file, got, want)
		}
		if _, err := os.Stat(p.Path(file)); err != nil {
			t.Errorf("Path(%q) does not exist: %v", file, err)
		}
	}
}

func TestDoc(t *testing.T) {
	p, err := Open(krRoot, "KR_")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := p.Doc("StoryData/S001A.json")
	if err != nil {
		t.Fatal(err)
	}
	model, ok := doc.Get("/dataList/1/model")
	if !ok || model.Str != "이스마엘" {
		t.Fatalf("/dataList/1/model = %+v, %v", model, ok)
	}
	again, err := p.Doc("StoryData/S001A.json")
	if err != nil {
		t.Fatal(err)
	}
	if again != doc {
		t.Error("second Doc call returned a different node, want the cached one")
	}
}

func TestDocErrors(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, "KR_ok.json")
	if err := os.WriteFile(filepath.Join(root, "KR_bad.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := Open(root, "KR_")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Doc("nope.json"); err == nil || !strings.Contains(err.Error(), "nope.json") {
		t.Errorf("Doc(nope.json) error = %v, want one naming the file", err)
	}
	if _, err := p.Doc("bad.json"); err == nil || !strings.Contains(err.Error(), "bad.json") {
		t.Errorf("Doc(bad.json) error = %v, want a parse error naming the file", err)
	}
	// A failed parse is not cached as success: the file stays an error.
	if _, err := p.Doc("bad.json"); err == nil {
		t.Error("second Doc(bad.json) succeeded")
	}
	// Files deleted after Open surface as read errors, not panics.
	if err := os.Remove(filepath.Join(root, "KR_ok.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Doc("ok.json"); err == nil {
		t.Error("Doc of a deleted file succeeded")
	}
}

func TestOpenRaw(t *testing.T) {
	raw := t.TempDir()
	writeFiles(t, raw,
		"kr/KR_Sample.json", "kr/StoryData/KR_S001A.json",
		"en/EN_Sample.json",
		"jp/JP_Sample.json", "jp/JP_Extra.json",
	)
	set, err := OpenRaw(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := set.KR.Files(), []string{"Sample.json", "StoryData/S001A.json"}; !reflect.DeepEqual(got, want) {
		t.Errorf("KR Files() = %v, want %v", got, want)
	}
	if got, want := set.EN.Files(), []string{"Sample.json"}; !reflect.DeepEqual(got, want) {
		t.Errorf("EN Files() = %v, want %v", got, want)
	}
	if got, want := set.JP.Files(), []string{"Extra.json", "Sample.json"}; !reflect.DeepEqual(got, want) {
		t.Errorf("JP Files() = %v, want %v", got, want)
	}
	if got, want := set.JP.Path("Sample.json"), filepath.Join(raw, "jp", "JP_Sample.json"); got != want {
		t.Errorf("JP Path = %q, want %q", got, want)
	}
}

func TestOpenRawMissingLanguage(t *testing.T) {
	raw := t.TempDir()
	writeFiles(t, raw, "kr/KR_a.json", "en/EN_a.json")
	_, err := OpenRaw(raw)
	if err == nil {
		t.Fatal("OpenRaw succeeded without a jp directory")
	}
	if !strings.Contains(err.Error(), "jp") {
		t.Errorf("error %q does not name the missing language", err)
	}
}
