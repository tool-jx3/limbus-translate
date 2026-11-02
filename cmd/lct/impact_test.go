package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// impactOut is the part of "impact --json" the tests look at.
type impactOut struct {
	Checked  int                 `json:"checked"`
	Units    int                 `json:"units"`
	ByCode   map[string]int      `json:"by_code"`
	ByOrigin map[string]int      `json:"by_origin"`
	Samples  map[string][]string `json:"samples"`
}

const (
	impactBaseGlossary = "- {kr: 성배, zh: 聖杯, status: approved, forbidden: [聖盃]}\n"
	// impactNewGlossary adds a term that forbids 神父.
	impactNewGlossary = impactBaseGlossary + "- {kr: 사제, zh: 祭司, status: approved, forbidden: [神父]}\n"
)

// impactKR are the lines of Other.json in KR, and what the translations of
// them are (index i of one is index i of the other).
var (
	impactKR = []string{"사제가 왔다", "사제를 만났다", "성배를 찾았다", "사제가 웃었다"}
	impactZH = []string{"神父來了", "遇見了神父", "找到了聖盃", "祭司笑了"}
	impactBy = []string{store.OriginClaude, store.OriginParatranz, store.OriginClaude, store.OriginHuman}
)

func impactKey(i int) string { return units.Key("Other.json", fmt.Sprintf("/dataList/%d/content", i)) }

// impactRepo is a git repo whose main branch holds the rules of the repo with
// a glossary of one term, a raw worktree with Other.json, and a tw worktree
// with a translation of each line. The glossary of the working tree has the
// new term besides, which is not committed. The working directory is the repo.
func impactRepo(t *testing.T) string {
	t.Helper()
	root := e2eRepo(t)
	writeTestFile(t, root, "glossary/terms.yaml", impactBaseGlossary)
	runGitIn(t, root, "add", "-A")
	runGitIn(t, root, "commit", "-q", "-m", "the glossary of main")
	writeTestFile(t, root, "glossary/terms.yaml", impactNewGlossary)

	var items []string
	for i, kr := range impactKR {
		items = append(items, fmt.Sprintf(`{"id":%d,"content":%q}`, i, kr))
	}
	writeTestFile(t, root, ".work/raw/meta/source.json", `{"tag": "raw-20261029", "token": "t", "created_at": "2026-10-29 02:55:23", "fetched_at": "2026-10-29T00:00:00Z", "files": {}}`)
	writeTestFile(t, root, ".work/raw/kr/KR_Other.json", `{"dataList":[`+strings.Join(items, ",")+`]}`)
	writeTestFile(t, root, ".work/raw/en/EN_Other.json", "{}\n")
	writeTestFile(t, root, ".work/raw/jp/JP_Other.json", "{}\n")

	recs := map[string]store.Record{}
	for i := range impactKR {
		k := impactKey(i)
		recs[k] = store.Record{Key: k, KRSha: units.KRSha(impactKR[i]), ZH: impactZH[i], Origin: impactBy[i], Since: "raw-20261029"}
	}
	if err := store.Open(filepath.Join(root, ".work", "tw")).SaveRecords(recs); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	return root
}

func TestImpactCommand(t *testing.T) {
	root := impactRepo(t)

	var got impactOut
	mustLct(t, &got, "impact", "--base", "HEAD", "--markdown", "out/impact.md")
	// Lines 0 and 1 have the 神父 that the new term forbids; line 2 has a 聖盃
	// that the old glossary forbids as well; line 3 is fine.
	if got.Checked != 4 || got.Units != 2 {
		t.Errorf("checked, units = %d, %d, want 4, 2", got.Checked, got.Units)
	}
	if got.ByCode["forbidden-term"] != 2 || got.ByOrigin[store.OriginClaude] != 1 || got.ByOrigin[store.OriginParatranz] != 1 || len(got.ByOrigin) != 2 {
		t.Errorf("by_code, by_origin = %v, %v, want 2 forbidden-term, one claude and one paratranz", got.ByCode, got.ByOrigin)
	}
	if want := []string{impactKey(0), impactKey(1)}; strings.Join(got.Samples["forbidden-term"], "\n") != strings.Join(want, "\n") {
		t.Errorf("samples = %v, want the keys %v", got.Samples, want)
	}

	md := string(readFile(t, filepath.Join(root, "out", "impact.md")))
	for _, want := range []string{"HEAD", "4", "2", "forbidden-term", "`" + impactKey(0) + "`", "`" + impactKey(1) + "`"} {
		if !strings.Contains(md, want) {
			t.Errorf("out/impact.md lacks %q:\n%s", want, md)
		}
	}
	if strings.Contains(md, impactKey(2)) || strings.Contains(md, impactKey(3)) {
		t.Errorf("out/impact.md names a unit that the new rules do not break:\n%s", md)
	}

	// As text it is a summary for people, and no file is written without
	// --markdown.
	if err := os.Remove(filepath.Join(root, "out", "impact.md")); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := lct(t, "impact", "--base", "HEAD")
	if code != 0 || stderr != "" || !strings.Contains(stdout, "2") || !strings.Contains(stdout, "forbidden-term") {
		t.Errorf("lct impact as text: exit code %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(root, "out", "impact.md")); err == nil {
		t.Errorf("lct impact wrote out/impact.md without --markdown")
	}
}

// The old rules are those of the revision, not of the working tree: when the
// working tree is the revision, nothing is new.
func TestImpactCommandSameRules(t *testing.T) {
	root := impactRepo(t)
	runGitIn(t, root, "add", "-A")
	runGitIn(t, root, "commit", "-q", "-m", "the new term")

	var got impactOut
	mustLct(t, &got, "impact", "--base", "HEAD")
	if got.Checked != 4 || got.Units != 0 {
		t.Errorf("checked, units = %d, %d, want 4, 0", got.Checked, got.Units)
	}
	var older impactOut
	mustLct(t, &older, "impact", "--base", "HEAD~1")
	if older.Units != 2 {
		t.Errorf("units against HEAD~1 = %d, want 2", older.Units)
	}
}

// A rule change that breaks nothing is no failure: finding units is the
// result of the command, not a finding.
func TestImpactCommandExitsZeroWithUnits(t *testing.T) {
	impactRepo(t)
	if code, stdout, stderr := lct(t, "--json", "impact", "--base", "HEAD"); code != 0 {
		t.Errorf("exit code %d, want 0\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
}

func TestImpactCommandUserErrors(t *testing.T) {
	root := impactRepo(t)
	for name, tc := range map[string]struct {
		args   []string
		stderr string
	}{
		"no base":             {[]string{"impact"}, "--base"},
		"empty base":          {[]string{"impact", "--base", ""}, "--base"},
		"a base like a flag":  {[]string{"impact", "--base", "-x"}, "--base"},
		"unknown revision":    {[]string{"impact", "--base", "no-such-branch"}, "no-such-branch"},
		"unexpected argument": {[]string{"impact", "--base", "HEAD", "extra"}, "extra"},
		"unknown flag":        {[]string{"impact", "--base", "HEAD", "--bogus"}, "bogus"},
	} {
		t.Run(name, func(t *testing.T) {
			code, stdout, stderr := lct(t, tc.args...)
			if code != 1 {
				t.Errorf("exit code = %d, want 1\nstdout: %s\nstderr: %s", code, stdout, stderr)
			}
			if stdout != "" || !strings.Contains(stderr, tc.stderr) {
				t.Errorf("stdout = %q, stderr = %q, want nothing on stdout and a message with %q", stdout, stderr, tc.stderr)
			}
		})
	}

	// No translations to compare: the tw worktree is not there.
	t.Run("no tw worktree", func(t *testing.T) {
		if err := os.RemoveAll(filepath.Join(root, ".work", "tw")); err != nil {
			t.Fatal(err)
		}
		code, stdout, stderr := lct(t, "impact", "--base", "HEAD")
		if code != 1 || stdout != "" || !strings.Contains(stderr, ".work/tw") {
			t.Errorf("exit code %d, stdout %q, stderr %q, want 1, nothing, and a message that names .work/tw", code, stdout, stderr)
		}
	})
	t.Run("no raw snapshot", func(t *testing.T) {
		if err := os.RemoveAll(filepath.Join(root, ".work", "raw")); err != nil {
			t.Fatal(err)
		}
		code, stdout, stderr := lct(t, "impact", "--base", "HEAD")
		if code != 1 || stdout != "" || !strings.Contains(stderr, "snapshot") {
			t.Errorf("exit code %d, stdout %q, stderr %q, want 1, nothing, and a message about the snapshot", code, stdout, stderr)
		}
	})
}

// The JSON of a run with nothing new has empty maps and not null.
func TestImpactCommandJSONHasNoNull(t *testing.T) {
	root := impactRepo(t)
	runGitIn(t, root, "add", "-A")
	runGitIn(t, root, "commit", "-q", "-m", "the new term")
	code, stdout, stderr := lct(t, "--json", "impact", "--base", "HEAD")
	if code != 0 {
		t.Fatalf("exit code %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stdout), &raw); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"checked", "units", "by_code", "by_origin", "samples"} {
		if v, ok := raw[name]; !ok || string(v) == "null" {
			t.Errorf("%q = %s, want a value that is not null", name, v)
		}
	}
}
