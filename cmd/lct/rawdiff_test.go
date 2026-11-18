package main

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/rawdiff"
)

// rawdiffRepo is a repo with the rules the families come from and one
// snapshot for each of packs, a week apart from 2026-10-29. A pack is its
// files to their JSON: those of kr/ by their name, those of the other packs
// with the directory ("en/EN_Skills.json").
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
			if !strings.Contains(name, "/") {
				name = "kr/" + name
			}
			writeTestFile(t, dir, name, content)
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
	{"KR_Skills.json": `{"dataList":[{"id":1,"name":"가"},{"id":2,"name":"나"}]}`, "KR_Skills-a1.json": `{"dataList":[{"id":3,"name":"[Storm]"}]}`,
		// JP gets another file than KR, with a tag KR has not.
		"jp/JP_Skills.json": `{"dataList":[{"id":1,"name":"あ <pause>"}]}`},
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
	if d.From != "raw-20261105" || d.To != "raw-20261112" || d.Lang != "kr" || d.VsKR != nil {
		t.Errorf("compared %s with %s of %q (vs KR: %v), want raw-20261105 with raw-20261112 of the KR pack", d.From, d.To, d.Lang, d.VsKR)
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
		{"--lang", "fr"},
		{"--lang", "KR"},
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
	want := "raw-20261029 -> raw-20261112 (KR)\n" +
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

func TestRawdiffComparesThePackOfAnotherLanguage(t *testing.T) {
	root := rawdiffRepo(t, rawdiffPacks...)
	d, err := runRawdiffJSON(t, root, "--lang", "jp")
	if err != nil {
		t.Fatal(err)
	}
	if d.Lang != "jp" || d.From != "raw-20261105" || d.To != "raw-20261112" {
		t.Errorf("compared %s with %s of %q, want raw-20261105 with raw-20261112 of the JP pack", d.From, d.To, d.Lang)
	}
	if len(d.FilesAdded) != 1 || d.FilesAdded[0] != (rawdiff.File{File: "Skills.json", Group: "Skills", Family: "ally", Entries: 1}) {
		t.Errorf("files added = %+v, want Skills.json, the name without the prefix of the language", d.FilesAdded)
	}
	// What JP changed and KR did not, and the reverse.
	want := &rawdiff.Versus{
		OnlyHere: []rawdiff.Item{
			{Kind: rawdiff.ItemGroup, Change: rawdiff.Added, Name: "Skills"},
			{Kind: rawdiff.ItemFile, Change: rawdiff.Added, Name: "Skills.json"},
			{Kind: rawdiff.ItemToken, Change: rawdiff.Added, Name: "<pause>"},
		},
		OnlyOther: []rawdiff.Item{
			{Kind: rawdiff.ItemFile, Change: rawdiff.Added, Name: "Skills-a1.json"},
			{Kind: rawdiff.ItemToken, Change: rawdiff.Added, Name: "[Storm]"},
		},
	}
	if !reflect.DeepEqual(d.VsKR, want) {
		t.Errorf("vs KR = %+v, want %+v", d.VsKR, want)
	}

	ctx, out := testCtx(t, root, "", false)
	if err := runRawdiff(ctx, []string{"--lang", "jp"}); err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{
		"raw-20261105 -> raw-20261112 (JP)\n",
		"differs from KR: 3 only in JP, 2 only in KR\n",
		"  JP + file Skills.json\n",
		"  JP + token <pause>\n",
		"  KR + file Skills-a1.json\n",
	} {
		if !strings.Contains(out.String(), line) {
			t.Errorf("output has no line %q:\n%s", line, out)
		}
	}
}
