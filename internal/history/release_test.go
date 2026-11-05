package history

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

const testLang = "zh-TW"

func runGit(t *testing.T, dir string, env []string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), env...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func put(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// repoFixture is a repo with a raw and a tw worktree.
type repoFixture struct{ root, raw, tw string }

func newRepoFixture(t *testing.T) repoFixture {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, nil, "init", "-q", "-b", "main")
	runGit(t, root, nil, "config", "user.name", "Test")
	runGit(t, root, nil, "config", "user.email", "test@example.com")
	runGit(t, root, nil, "config", "commit.gpgsign", "false")
	runGit(t, root, nil, "config", "tag.gpgsign", "false")
	put(t, root, "README", "x\n")
	runGit(t, root, nil, "add", "-A")
	runGit(t, root, nil, "commit", "-q", "-m", "init")
	f := repoFixture{root, filepath.Join(root, ".work", "raw"), filepath.Join(root, ".work", "tw")}
	for _, wt := range []struct{ branch, dir string }{{"raw", f.raw}, {"tw", f.tw}} {
		if err := gitx.EnsureWorktree(root, wt.branch, wt.dir); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

// snapshot commits the KR files (keyed by pack file name) as the whole of kr/
// on the raw branch and tags it.
func (f repoFixture) snapshot(t *testing.T, tag string, kr map[string]string) {
	t.Helper()
	if err := os.RemoveAll(filepath.Join(f.raw, "kr")); err != nil {
		t.Fatal(err)
	}
	for name, content := range kr {
		put(t, f.raw, "kr/KR_"+name, content)
	}
	runGit(t, f.raw, nil, "add", "-A")
	runGit(t, f.raw, nil, "commit", "-q", "--allow-empty", "-m", tag)
	runGit(t, f.raw, nil, "tag", tag)
}

// buildSpec is what one build puts in the tw worktree.
type buildSpec struct {
	rawTag, rules string
	files         map[string]string // under LimbusCompany_Data/, replacing it
	records       []store.Record
}

// stage writes spec into the tw worktree without committing.
func (f repoFixture) stage(t *testing.T, spec buildSpec) {
	t.Helper()
	for _, d := range []string{"LimbusCompany_Data", ".meta"} {
		if err := os.RemoveAll(filepath.Join(f.tw, d)); err != nil {
			t.Fatal(err)
		}
	}
	for name, content := range spec.files {
		put(t, f.tw, "LimbusCompany_Data/"+name, content)
	}
	put(t, f.tw, ".meta/build.json", `{"raw_tag":"`+spec.rawTag+`","raw_token":"t","rules_commit":"`+spec.rules+`","built_at":"2026-11-01T00:00:00Z"}`+"\n")
	recs := map[string]store.Record{}
	for _, r := range spec.records {
		recs[r.Key] = r
	}
	if err := store.Open(f.tw).SaveRecords(recs); err != nil {
		t.Fatal(err)
	}
}

// release stages spec, commits it at date and tags the commit.
func (f repoFixture) release(t *testing.T, tag, date string, spec buildSpec) {
	t.Helper()
	f.stage(t, spec)
	env := []string{"GIT_COMMITTER_DATE=" + date, "GIT_AUTHOR_DATE=" + date}
	runGit(t, f.tw, env, "add", "-A")
	runGit(t, f.tw, env, "commit", "-q", "--allow-empty", "-m", tag)
	runGit(t, f.tw, nil, "tag", tag)
}

func (f repoFixture) rev(t *testing.T, rev string) *Side {
	t.Helper()
	s, err := ReadRev(f.tw, f.raw, rev, testLang, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

const (
	relA   = "zh-TW-20261101.1"
	relB   = "zh-TW-20261102.1"
	rulesA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	rulesB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	atB    = "2026-11-02T12:00:00+08:00"
)

var timeB = time.Date(2026, 11, 2, 4, 0, 0, 0, time.UTC)

func lang(name string) string { return "Lang/" + testLang + "/" + name }

func claude(file, ptr, zh string) store.Record {
	return store.Record{Key: units.Key(file, ptr), ZH: zh, Origin: store.OriginClaude}
}

// twoReleases makes the two builds: id 1 is retranslated, id 2 changes in KR,
// id 3 is new and a copy.
func twoReleases(t *testing.T) repoFixture {
	t.Helper()
	f := newRepoFixture(t)
	f.snapshot(t, "raw-A", map[string]string{"Passives.json": list(`{"id":1,"desc":"a"}`, `{"id":2,"desc":"b"}`)})
	f.snapshot(t, "raw-B", map[string]string{"Passives.json": list(`{"id":1,"desc":"a"}`, `{"id":2,"desc":"B"}`, `{"id":3,"desc":"c"}`)})
	f.release(t, relA, "2026-11-01T00:00:00Z", buildSpec{
		rawTag: "raw-A", rules: rulesA,
		files: map[string]string{
			lang("Passives.json"):        list(`{"id":1,"desc":"甲"}`, `{"id":2,"desc":"乙"}`),
			lang("Info/LICENSE-CONTENT"): "license\n",
			lang("Info/version.json"):    `{"v":1}`,
			"Lang/config.json":           `{"lang":"zh-TW"}`,
		},
		records: []store.Record{claude("Passives.json", "/dataList/0/desc", "甲"), claude("Passives.json", "/dataList/1/desc", "乙")},
	})
	f.release(t, relB, atB, buildSpec{
		rawTag: "raw-B", rules: rulesB,
		files: map[string]string{
			lang("Passives.json"):        list(`{"id":1,"desc":"甲甲"}`, `{"id":2,"desc":"乙乙"}`, `{"id":3,"desc":"c"}`),
			lang("Info/LICENSE-CONTENT"): "license\n",
			lang("Info/version.json"):    `{"v":2}`,
			"Lang/config.json":           `{"lang":"zh-TW"}`,
		},
		records: []store.Record{
			claude("Passives.json", "/dataList/0/desc", "甲甲"), claude("Passives.json", "/dataList/1/desc", "乙乙"),
			{Key: units.Key("Passives.json", "/dataList/2/desc"), ZH: "c", Origin: store.OriginCopy},
		},
	})
	return f
}

func TestMakeComparesTwoReleases(t *testing.T) {
	f := twoReleases(t)
	rec, err := Make(f.rev(t, relA), f.rev(t, relB), relB, timeB, testLang)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Tag != relB || rec.PrevTag != relA {
		t.Errorf("tags = %q, %q", rec.Tag, rec.PrevTag)
	}
	if rec.ReleasedAt != "2026-11-02T04:00:00Z" || rec.PrevReleasedAt != "2026-11-01T00:00:00Z" {
		t.Errorf("times = %q, %q", rec.ReleasedAt, rec.PrevReleasedAt)
	}
	if rec.RawTag != "raw-B" || rec.PrevRawTag != "raw-A" || rec.RulesCommit != rulesB || rec.PrevRulesCommit != rulesA {
		t.Errorf("provenance = %+v", rec)
	}
	if want := (Counts{KRChanged: 1, ZHChanged: 1}); rec.Counts != want {
		t.Errorf("Counts = %+v, want %+v", rec.Counts, want)
	}
	if want := (Counts{New: 1}); rec.CopyCounts != want {
		t.Errorf("CopyCounts = %+v, want %+v", rec.CopyCounts, want)
	}
	if len(rec.Entries) != 3 {
		t.Errorf("entries = %+v", rec.Entries)
	}
	if rec.Empty() {
		t.Error("Empty() is true")
	}
}

func TestMakeListsTheOtherFilesThatChanged(t *testing.T) {
	f := newRepoFixture(t)
	f.snapshot(t, "raw-A", map[string]string{"Passives.json": list(`{"id":1,"desc":"a"}`)})
	spec := func(license, version string) buildSpec {
		return buildSpec{
			rawTag: "raw-A", rules: rulesA,
			files: map[string]string{
				lang("Passives.json"):        list(`{"id":1,"desc":"甲"}`),
				lang("Info/LICENSE-CONTENT"): license,
				lang("Info/version.json"):    version,
				"Lang/config.json":           `{"lang":"zh-TW"}`,
			},
			records: []store.Record{claude("Passives.json", "/dataList/0/desc", "甲")},
		}
	}
	f.release(t, relA, "2026-11-01T00:00:00Z", spec("one\n", `{"v":1}`))
	f.release(t, relB, atB, spec("two\n", `{"v":2}`))
	rec, err := Make(f.rev(t, relA), f.rev(t, relB), relB, timeB, testLang)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Lang/zh-TW/Info/LICENSE-CONTENT"}; !reflect.DeepEqual(rec.OtherFiles, want) {
		t.Errorf("OtherFiles = %v, want %v", rec.OtherFiles, want)
	}
	if len(rec.Entries) != 0 {
		t.Errorf("entries = %+v", rec.Entries)
	}
	if rec.Empty() {
		t.Error("Empty() is true although a file changed")
	}
}

func TestMakeComparesOnlyTheFilesThatDiffer(t *testing.T) {
	f := newRepoFixture(t)
	const broken = "{ this is not json"
	f.snapshot(t, "raw-A", map[string]string{"Broken.json": broken, "Passives.json": list(`{"id":1,"desc":"a"}`)})
	f.snapshot(t, "raw-B", map[string]string{"Broken.json": broken, "Passives.json": list(`{"id":1,"desc":"A"}`)})
	spec := func(rawTag, zh string) buildSpec {
		return buildSpec{
			rawTag: rawTag, rules: rulesA,
			files:   map[string]string{lang("Broken.json"): broken, lang("Passives.json"): list(`{"id":1,"desc":"` + zh + `"}`)},
			records: []store.Record{claude("Passives.json", "/dataList/0/desc", zh)},
		}
	}
	f.release(t, relA, "2026-11-01T00:00:00Z", spec("raw-A", "甲"))
	f.release(t, relB, atB, spec("raw-B", "乙"))
	prev, cur := f.rev(t, relA), f.rev(t, relB)

	var handed []string
	rec, err := makeRecord(prev, cur, relB, timeB, testLang, func(o, n State, files []string) ([]Entry, error) {
		handed = files
		return Compare(o, n, files)
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Passives.json"}; !reflect.DeepEqual(handed, want) {
		t.Errorf("files compared = %v, want %v", handed, want)
	}
	if len(rec.Entries) != 1 || len(rec.OtherFiles) != 0 {
		t.Errorf("record = %+v", rec)
	}
	// And the broken file is not even read through the public Make.
	if _, err := Make(prev, cur, relB, timeB, testLang); err != nil {
		t.Errorf("Make: %v", err)
	}
}

func TestReadWorktreeReadsWhatIsNotCommitted(t *testing.T) {
	f := twoReleases(t)
	// A third build, not committed: id 1 is translated yet again.
	f.stage(t, buildSpec{
		rawTag: "raw-B", rules: rulesB,
		files: map[string]string{
			lang("Passives.json"):        list(`{"id":1,"desc":"甲丙"}`, `{"id":2,"desc":"乙乙"}`, `{"id":3,"desc":"c"}`),
			lang("Info/LICENSE-CONTENT"): "license\n",
			lang("Info/version.json"):    `{"v":2}`,
			"Lang/config.json":           `{"lang":"zh-TW"}`,
		},
		records: []store.Record{
			claude("Passives.json", "/dataList/0/desc", "甲丙"), claude("Passives.json", "/dataList/1/desc", "乙乙"),
			{Key: units.Key("Passives.json", "/dataList/2/desc"), ZH: "c", Origin: store.OriginCopy},
		},
	})
	cur, err := ReadWorktree(f.tw, f.raw, testLang, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cur.RawTag != "raw-B" || cur.RulesCommit != rulesB {
		t.Errorf("worktree side = %+v", cur)
	}
	rec, err := Make(f.rev(t, relB), cur, "zh-TW-20261103.1", timeB.Add(24*time.Hour), testLang)
	if err != nil {
		t.Fatal(err)
	}
	if rec.PrevTag != relB || len(rec.Entries) != 1 || rec.Entries[0].Kind != KindZHChanged {
		t.Errorf("record = %+v", rec)
	}
	if got := *rec.Entries[0].NewZH; got != "甲丙" {
		t.Errorf("new zh = %q", got)
	}
}
