package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/rawdiff"
)

// rawdiffRepo is a repo with the rules the families come from and one
// snapshot for each KR pack of packs (the files of kr/ to their JSON), a week
// apart from 2026-10-29.
func rawdiffRepo(t *testing.T, packs ...map[string]string) string {
	t.Helper()
	root := snapshotRepo(t)
	for _, f := range []string{"glossary/terms.yaml", "guides/routing.yaml"} {
		copyFile(t, filepath.Join(repoRoot, f), filepath.Join(root, f))
	}
	copyDir(t, filepath.Join(repoRoot, "guides/characters"), filepath.Join(root, "guides/characters"))
	days := []string{"2026-10-29", "2026-11-05", "2026-11-12"}
	for i, files := range packs {
		dir := t.TempDir()
		for name, content := range files {
			writeTestFile(t, dir, "kr/"+name, content)
		}
		writeTestFile(t, dir, "en/EN_A.json", "{}\n")
		writeTestFile(t, dir, "jp/JP_A.json", "{}\n")
		if _, err := runSnapshotJSON(t, root, "--from", dir, "--token", "l"+strings.ReplaceAll(days[i], "-", "")+"_token", "--created-at", days[i]+" 02:00:00"); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

var rawdiffPacks = []map[string]string{
	{"KR_Skills.json": `{"dataList":[{"id":1,"name":"가"}]}`},
	{"KR_Skills.json": `{"dataList":[{"id":1,"name":"가"},{"id":2,"name":"나"}]}`},
	{"KR_Skills.json": `{"dataList":[{"id":1,"name":"가"},{"id":2,"name":"나"}]}`, "KR_Skills-a1.json": `{"dataList":[{"id":3,"name":"[Storm]"}]}`},
}

func runRawdiffJSON(t *testing.T, root string, args ...string) (rawdiff.Diff, error) {
	t.Helper()
	ctx, out := testCtx(t, root, "", true)
	err := runRawdiff(ctx, args)
	var got rawdiff.Diff
	if err == nil {
		if err := json.Unmarshal(out.Bytes(), &got); err != nil {
			t.Fatalf("output is not JSON: %v\n%s", err, out)
		}
	}
	return got, err
}

func TestRawdiffComparesTheLastTwoSnapshots(t *testing.T) {
	root := rawdiffRepo(t, rawdiffPacks...)
	d, err := runRawdiffJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}
	if d.From != "raw-20261105" || d.To != "raw-20261112" {
		t.Errorf("compared %s with %s, want raw-20261105 with raw-20261112", d.From, d.To)
	}
	// The family is that of the routing rules of the repo.
	if len(d.FilesAdded) != 1 || d.FilesAdded[0] != (rawdiff.File{File: "Skills-a1.json", Group: "Skills", Family: "ally", Entries: 1}) {
		t.Errorf("files added = %+v, want Skills-a1.json of the family ally", d.FilesAdded)
	}
	if d.Counts.TokensAdded != 1 || d.Counts.EntriesAdded != 0 {
		t.Errorf("counts = %+v, want one token and no entry of a file in both", d.Counts)
	}
}

func TestRawdiffComparesTheTagsItIsGiven(t *testing.T) {
	root := rawdiffRepo(t, rawdiffPacks...)
	d, err := runRawdiffJSON(t, root, "--from", "raw-20261029", "--to", "raw-20261105")
	if err != nil {
		t.Fatal(err)
	}
	if d.From != "raw-20261029" || d.To != "raw-20261105" || len(d.Entries) != 1 || len(d.Entries[0].Added) != 1 || d.Entries[0].Added[0] != "2" {
		t.Errorf("diff = %+v, want the entry 2 added between raw-20261029 and raw-20261105", d)
	}
	// --to alone compares with the snapshot before it.
	d, err = runRawdiffJSON(t, root, "--to", "raw-20261105")
	if err != nil || d.From != "raw-20261029" {
		t.Errorf("--to raw-20261105 compared with %q, %v; want raw-20261029", d.From, err)
	}
	// --from alone compares with the newest.
	d, err = runRawdiffJSON(t, root, "--from", "raw-20261029")
	if err != nil || d.To != "raw-20261112" || d.Counts.FilesAdded != 1 || d.Counts.EntriesAdded != 1 {
		t.Errorf("--from raw-20261029 = %+v, %v; want everything up to raw-20261112", d, err)
	}
}

func TestRawdiffUsageErrors(t *testing.T) {
	root := rawdiffRepo(t, rawdiffPacks...)
	for _, args := range [][]string{
		{"--from", "raw-20260101"},
		{"--to", "raw-20260101"},
		{"--to", "raw-20261029"}, // no snapshot before the first
		{"extra"},
		{"--nope"},
	} {
		if _, err := runRawdiffJSON(t, root, args...); !isUserError(err) {
			t.Errorf("rawdiff %v: err = %v, want a user error", args, err)
		}
	}
	one := rawdiffRepo(t, rawdiffPacks[0])
	if _, err := runRawdiffJSON(t, one); !isUserError(err) || !strings.Contains(err.Error(), "two snapshots") {
		t.Errorf("one snapshot: err = %v, want a user error that asks for two snapshots", err)
	}
}

func TestRawdiffTextOutput(t *testing.T) {
	root := rawdiffRepo(t, rawdiffPacks...)
	ctx, out := testCtx(t, root, "", false)
	if err := runRawdiff(ctx, []string{"--from", "raw-20261029"}); err != nil {
		t.Fatal(err)
	}
	want := "raw-20261029 -> raw-20261112\n" +
		"files: +1 -0\n" +
		"  + Skills-a1.json (ally, 1 entries)\n" +
		"groups: +0 -0\n" +
		"keys: +0 -0\n" +
		"entries: +1 -0 in 1 files\n" +
		"  Skills.json: 1 -> 2 (+1 -0)\n" +
		"tokens: +1 -0\n" +
		"  + bracket Storm (1; Skills-a1.json /dataList/0/name; defined in no group)\n"
	if out.String() != want {
		t.Errorf("output:\n%s\nwant:\n%s", out, want)
	}
}
