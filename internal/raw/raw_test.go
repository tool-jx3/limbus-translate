package raw

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/gitx"
)

const (
	token1   = "l20261029_n2rX1Jku6_9sBiu9Np24"
	created1 = "2026-10-29 02:55:23"
	token2   = "l20261105_AbCdEf_123"
	created2 = "2026-11-05 03:10:00"
)

// runGit runs a git command in dir and returns its output without the final
// newline.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSuffix(string(out), "\n")
}

func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, dir, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func sha(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// rawWorktree makes a repo with a main branch and an orphan raw worktree, and
// returns the worktree. The repo converts line endings on its own accord
// (core.autocrlf), which the snapshot must keep git from doing.
func rawWorktree(t *testing.T) string {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "init", "-q", "-b", "main")
	runGit(t, repo, "config", "user.name", "Test")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "commit.gpgsign", "false")
	runGit(t, repo, "config", "tag.gpgsign", "false")
	runGit(t, repo, "config", "core.autocrlf", "true")
	write(t, repo, "main.txt", "main\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "init")
	wt := filepath.Join(repo, ".work", "raw")
	if err := gitx.EnsureWorktree(repo, "raw", wt); err != nil {
		t.Fatal(err)
	}
	return wt
}

// extracted builds the three extracted pack directories and returns the packs
// map Snapshot takes. Every file maps its relative path to its content.
func extracted(t *testing.T, kr, en, jp map[string]string) map[string]string {
	t.Helper()
	root := t.TempDir()
	packs := map[string]string{}
	for lang, files := range map[string]map[string]string{"kr": kr, "en": en, "jp": jp} {
		dir := filepath.Join(root, lang)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		for rel, content := range files {
			write(t, dir, rel, content)
		}
		packs[lang] = dir
	}
	return packs
}

const bomCRLF = "\xef\xbb\xbf{\r\n  \"dataList\": []\r\n}\r\n"

func firstPacks(t *testing.T) map[string]string {
	return extracted(t,
		map[string]string{"KR_A.json": bomCRLF, "StoryData/KR_S.json": "{\"a\":1}\n", "KR_Old.json": "{}\n"},
		map[string]string{"EN_A.json": "{}\r\n"},
		map[string]string{"JP_A.json": "{}\n"})
}

func TestNextTag(t *testing.T) {
	tests := []struct {
		name      string
		existing  []string
		createdAt string
		want      string
	}{
		{"first of the day", []string{}, "2026-10-29 02:55:23", "raw-20261029"},
		{"nil list", nil, "2026-10-29 02:55:23", "raw-20261029"},
		{"second of the day", []string{"raw-20261029"}, "2026-10-29 02:55:23", "raw-20261029.2"},
		{"third of the day", []string{"raw-20261029", "raw-20261029.2"}, "2026-10-29 09:00:00", "raw-20261029.3"},
		{"fills the first gap", []string{"raw-20261029", "raw-20261029.3"}, "2026-10-29 09:00:00", "raw-20261029.2"},
		{"other days do not count", []string{"raw-20261028", "raw-20261028.2", "raw-20261030"}, "2026-10-29 02:55:23", "raw-20261029"},
		{"RFC 3339 timestamp", nil, "2026-10-29T02:55:23Z", "raw-20261029"},
		{"date only", nil, "2026-10-29", "raw-20261029"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := NextTag(tc.existing, tc.createdAt); got != tc.want {
				t.Errorf("NextTag(%v, %q) = %q, want %q", tc.existing, tc.createdAt, got, tc.want)
			}
		})
	}
}

func TestLastSource(t *testing.T) {
	wt := t.TempDir()
	got, err := LastSource(wt)
	if err != nil || got != nil {
		t.Fatalf("without meta/source.json: LastSource = %v, %v; want nil, nil", got, err)
	}

	write(t, wt, "meta/source.json", `{"tag":"raw-20261029","token":"tok","created_at":"2026-10-29 02:55:23","fetched_at":"2026-10-30T01:00:00Z","files":{"kr/KR_A.json":"abc"}}`)
	got, err = LastSource(wt)
	if err != nil {
		t.Fatal(err)
	}
	want := Source{Tag: "raw-20261029", Token: "tok", CreatedAt: "2026-10-29 02:55:23", FetchedAt: "2026-10-30T01:00:00Z",
		Files: map[string]string{"kr/KR_A.json": "abc"}}
	if got.Tag != want.Tag || got.Token != want.Token || got.CreatedAt != want.CreatedAt || got.FetchedAt != want.FetchedAt ||
		len(got.Files) != 1 || got.Files["kr/KR_A.json"] != "abc" {
		t.Errorf("LastSource = %+v, want %+v", got, want)
	}

	write(t, wt, "meta/source.json", "{not json")
	if _, err := LastSource(wt); err == nil {
		t.Error("a broken meta/source.json must be an error")
	}
}

func TestSnapshotIdempotent(t *testing.T) {
	wt := rawWorktree(t)
	packs := firstPacks(t)
	src := Source{Token: token1, CreatedAt: created1}

	tag, changed, err := Snapshot(wt, packs, src)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || tag != "raw-20261029" {
		t.Fatalf("first Snapshot = %q, %v; want raw-20261029, true", tag, changed)
	}
	head := runGit(t, wt, "rev-parse", "HEAD")
	if got := runGit(t, wt, "rev-parse", tag+"^{commit}"); got != head {
		t.Errorf("tag %s points to %s, want HEAD %s", tag, got, head)
	}
	if got := runGit(t, wt, "log", "-1", "--format=%B"); strings.TrimSpace(got) != "raw: "+token1+" ("+created1+")" {
		t.Errorf("commit message = %q", got)
	}

	tag2, changed, err := Snapshot(wt, packs, src)
	if err != nil {
		t.Fatal(err)
	}
	if changed || tag2 != tag {
		t.Errorf("second Snapshot = %q, %v; want %q, false", tag2, changed, tag)
	}
	if got := runGit(t, wt, "rev-list", "--count", "HEAD"); got != "1" {
		t.Errorf("%s commits, want still 1", got)
	}
	if got := runGit(t, wt, "tag", "--list"); got != tag {
		t.Errorf("tags = %q, want just %s", got, tag)
	}
	if got := runGit(t, wt, "status", "--porcelain"); got != "" {
		t.Errorf("worktree is dirty after an unchanged snapshot:\n%s", got)
	}
}

func TestSnapshotLayout(t *testing.T) {
	wt := rawWorktree(t)
	packs := firstPacks(t)
	if _, _, err := Snapshot(wt, packs, Source{Token: token1, CreatedAt: created1, FetchedAt: "2026-10-30T01:02:03Z"}); err != nil {
		t.Fatal(err)
	}

	for rel, want := range map[string]string{
		"kr/KR_A.json":           bomCRLF,
		"kr/StoryData/KR_S.json": "{\"a\":1}\n",
		"en/EN_A.json":           "{}\r\n",
		"jp/JP_A.json":           "{}\n",
		".gitattributes":         "* -text\n",
	} {
		if got := read(t, wt, rel); got != want {
			t.Errorf("%s = %q, want %q", rel, got, want)
		}
	}
	// What git stored is the same bytes, although the repo has core.autocrlf.
	blob := runGit(t, wt, "show", "HEAD:kr/KR_A.json")
	if blob != strings.TrimSuffix(bomCRLF, "\n") {
		t.Errorf("committed KR_A.json = %q, want the CRLF and BOM kept", blob)
	}

	wantMeta := `{
  "tag": "raw-20261029",
  "token": "` + token1 + `",
  "created_at": "` + created1 + `",
  "fetched_at": "2026-10-30T01:02:03Z",
  "files": {
    "en/EN_A.json": "` + sha("{}\r\n") + `",
    "jp/JP_A.json": "` + sha("{}\n") + `",
    "kr/KR_A.json": "` + sha(bomCRLF) + `",
    "kr/KR_Old.json": "` + sha("{}\n") + `",
    "kr/StoryData/KR_S.json": "` + sha("{\"a\":1}\n") + `"
  }
}
`
	if got := read(t, wt, "meta/source.json"); got != wantMeta {
		t.Errorf("meta/source.json:\n%s\nwant:\n%s", got, wantMeta)
	}

	last, err := LastSource(wt)
	if err != nil || last == nil || last.Tag != "raw-20261029" || last.Token != token1 {
		t.Errorf("LastSource = %+v, %v", last, err)
	}
	if got := runGit(t, wt, "ls-files"); strings.Contains(got, "main.txt") {
		t.Errorf("raw must not hold main's files:\n%s", got)
	}
}

func TestSnapshotFetchedAtDefaultsToNow(t *testing.T) {
	wt := rawWorktree(t)
	before := time.Now().UTC().Add(-time.Second)
	if _, _, err := Snapshot(wt, firstPacks(t), Source{Token: token1, CreatedAt: created1}); err != nil {
		t.Fatal(err)
	}
	last, err := LastSource(wt)
	if err != nil {
		t.Fatal(err)
	}
	got, err := time.Parse(time.RFC3339, last.FetchedAt)
	if err != nil || !strings.HasSuffix(last.FetchedAt, "Z") {
		t.Fatalf("fetched_at = %q, want an RFC 3339 UTC time (%v)", last.FetchedAt, err)
	}
	if got.Before(before) || got.After(time.Now().Add(time.Second)) {
		t.Errorf("fetched_at = %v, want about now", got)
	}
}

func TestSnapshotNewTokenReplacesTree(t *testing.T) {
	wt := rawWorktree(t)
	if _, _, err := Snapshot(wt, firstPacks(t), Source{Token: token1, CreatedAt: created1}); err != nil {
		t.Fatal(err)
	}

	second := extracted(t,
		map[string]string{"KR_A.json": "{\"changed\":true}\n", "StoryData/KR_New.json": "{}\n"}, // KR_Old.json and StoryData/KR_S.json are gone
		map[string]string{"EN_A.json": "{}\r\n"},
		map[string]string{"JP_A.json": "{}\n"})
	tag, changed, err := Snapshot(wt, second, Source{Token: token2, CreatedAt: created2})
	if err != nil {
		t.Fatal(err)
	}
	if !changed || tag != "raw-20261105" {
		t.Fatalf("Snapshot = %q, %v; want raw-20261105, true", tag, changed)
	}
	want := ".gitattributes\nen/EN_A.json\njp/JP_A.json\nkr/KR_A.json\nkr/StoryData/KR_New.json\nmeta/source.json"
	if got := runGit(t, wt, "ls-files"); got != want {
		t.Errorf("tracked files:\n%s\nwant:\n%s", got, want)
	}
	if _, err := os.Stat(filepath.Join(wt, "kr", "KR_Old.json")); !os.IsNotExist(err) {
		t.Errorf("a file missing from the new pack must be removed, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(wt, "kr", "StoryData", "KR_S.json")); !os.IsNotExist(err) {
		t.Errorf("a file missing from the new pack must be removed, stat err = %v", err)
	}
	if got := runGit(t, wt, "rev-list", "--count", "HEAD"); got != "2" {
		t.Errorf("%s commits, want 2", got)
	}
	if got := runGit(t, wt, "tag", "--list"); got != "raw-20261029\nraw-20261105" {
		t.Errorf("tags = %q", got)
	}
	last, err := LastSource(wt)
	if err != nil || last.Token != token2 || last.Tag != tag {
		t.Errorf("LastSource = %+v, %v", last, err)
	}
	if _, ok := last.Files["kr/KR_Old.json"]; ok {
		t.Error("meta/source.json still lists a removed file")
	}
}

func TestSnapshotSameDayGetsSuffix(t *testing.T) {
	wt := rawWorktree(t)
	if _, _, err := Snapshot(wt, firstPacks(t), Source{Token: token1, CreatedAt: created1}); err != nil {
		t.Fatal(err)
	}
	tag, changed, err := Snapshot(wt, firstPacks(t), Source{Token: "l20261029_second", CreatedAt: "2026-10-29 18:00:00"})
	if err != nil || !changed || tag != "raw-20261029.2" {
		t.Errorf("Snapshot = %q, %v, %v; want raw-20261029.2, true, nil", tag, changed, err)
	}
}

func TestSnapshotRejectsBadInput(t *testing.T) {
	complete := func(t *testing.T) map[string]string { return firstPacks(t) }
	tests := []struct {
		name  string
		packs func(t *testing.T) map[string]string
		src   Source
	}{
		{"no token", complete, Source{CreatedAt: created1}},
		{"created_at is not a date", complete, Source{Token: token1, CreatedAt: "yesterday"}},
		{"no created_at", complete, Source{Token: token1}},
		{"missing jp", func(t *testing.T) map[string]string {
			p := complete(t)
			delete(p, "jp")
			return p
		}, Source{Token: token1, CreatedAt: created1}},
		{"unknown language", func(t *testing.T) map[string]string {
			p := complete(t)
			p["de"] = p["en"]
			return p
		}, Source{Token: token1, CreatedAt: created1}},
		{"jp directory does not exist", func(t *testing.T) map[string]string {
			p := complete(t)
			p["jp"] = filepath.Join(t.TempDir(), "nope")
			return p
		}, Source{Token: token1, CreatedAt: created1}},
		{"empty kr pack", func(t *testing.T) map[string]string {
			p := complete(t)
			p["kr"] = t.TempDir()
			return p
		}, Source{Token: token1, CreatedAt: created1}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wt := rawWorktree(t)
			// An earlier snapshot, which a bad call must leave alone.
			if _, _, err := Snapshot(wt, firstPacks(t), Source{Token: token1, CreatedAt: created1}); err != nil {
				t.Fatal(err)
			}
			src := tc.src
			if src.Token == token1 {
				src.Token = token2 // not the same token, so the call is not skipped as unchanged
			}
			_, _, err := Snapshot(wt, tc.packs(t), src)
			if err == nil {
				t.Fatal("Snapshot succeeded, want an error")
			}
			if !errors.Is(err, ErrInvalidInput) {
				t.Errorf("error = %v, want ErrInvalidInput", err)
			}
			if got := runGit(t, wt, "status", "--porcelain"); got != "" {
				t.Errorf("a rejected snapshot changed the worktree:\n%s", got)
			}
			if got := runGit(t, wt, "rev-list", "--count", "HEAD"); got != "1" {
				t.Errorf("%s commits, want 1", got)
			}
		})
	}
}

func TestSnapshotRejectsPacksInsideTheWorktree(t *testing.T) {
	wt := rawWorktree(t)
	if _, _, err := Snapshot(wt, firstPacks(t), Source{Token: token1, CreatedAt: created1}); err != nil {
		t.Fatal(err)
	}
	// Snapshotting the worktree's own kr/en/jp would delete the source first.
	packs := map[string]string{"kr": filepath.Join(wt, "kr"), "en": filepath.Join(wt, "en"), "jp": filepath.Join(wt, "jp")}
	_, _, err := Snapshot(wt, packs, Source{Token: token2, CreatedAt: created2})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
	if got := read(t, wt, "kr/KR_A.json"); got != bomCRLF {
		t.Errorf("KR_A.json = %q, the source was damaged", got)
	}
}

func TestSnapshotFilesMapIsComplete(t *testing.T) {
	wt := rawWorktree(t)
	if _, _, err := Snapshot(wt, firstPacks(t), Source{Token: token1, CreatedAt: created1}); err != nil {
		t.Fatal(err)
	}
	last, err := LastSource(wt)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for k := range last.Files {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	want := []string{"en/EN_A.json", "jp/JP_A.json", "kr/KR_A.json", "kr/KR_Old.json", "kr/StoryData/KR_S.json"}
	if !slices.Equal(keys, want) {
		t.Errorf("files = %v, want %v", keys, want)
	}
}

func TestSnapshotTagsACommitThatLostItsTag(t *testing.T) {
	wt := rawWorktree(t)
	packs := firstPacks(t)
	src := Source{Token: token1, CreatedAt: created1}
	if _, _, err := Snapshot(wt, packs, src); err != nil {
		t.Fatal(err)
	}
	head := runGit(t, wt, "rev-parse", "HEAD")

	// A run that committed the snapshot but died before it could tag it.
	runGit(t, wt, "tag", "-d", "raw-20261029")

	tag, changed, err := Snapshot(wt, packs, src)
	if err != nil {
		t.Fatal(err)
	}
	if tag != "raw-20261029" || !changed {
		t.Errorf("Snapshot = %q, %v; want raw-20261029, true (the snapshot is complete only now)", tag, changed)
	}
	if got := runGit(t, wt, "rev-parse", "raw-20261029^{commit}"); got != head {
		t.Errorf("the tag points to %s, want the existing commit %s", got, head)
	}
	if got := runGit(t, wt, "rev-list", "--count", "HEAD"); got != "1" {
		t.Errorf("%s commits, want still 1", got)
	}

	// With the tag back in place the next run has nothing to do.
	tag, changed, err = Snapshot(wt, packs, src)
	if err != nil || changed || tag != "raw-20261029" {
		t.Errorf("third Snapshot = %q, %v, %v; want raw-20261029, false, nil", tag, changed, err)
	}
}

func TestSnapshotRedoesAnUnfinishedSnapshot(t *testing.T) {
	wt := rawWorktree(t)
	if _, _, err := Snapshot(wt, firstPacks(t), Source{Token: token1, CreatedAt: created1}); err != nil {
		t.Fatal(err)
	}
	// A run for token2 that wrote meta/source.json and then died, before it
	// committed anything.
	write(t, wt, "meta/source.json", `{"tag":"raw-20261105","token":"`+token2+`","created_at":"`+created2+`","fetched_at":"2026-11-06T00:00:00Z","files":{}}`)

	second := extracted(t,
		map[string]string{"KR_A.json": "{\"changed\":true}\n"},
		map[string]string{"EN_A.json": "{}\n"},
		map[string]string{"JP_A.json": "{}\n"})
	tag, changed, err := Snapshot(wt, second, Source{Token: token2, CreatedAt: created2})
	if err != nil {
		t.Fatal(err)
	}
	if !changed || tag != "raw-20261105" {
		t.Fatalf("Snapshot = %q, %v; want raw-20261105, true (not skipped as unchanged)", tag, changed)
	}
	if got := runGit(t, wt, "rev-list", "--count", "HEAD"); got != "2" {
		t.Errorf("%s commits, want 2", got)
	}
	if got := runGit(t, wt, "rev-parse", "raw-20261105^{commit}"); got != runGit(t, wt, "rev-parse", "HEAD") {
		t.Errorf("tag raw-20261105 is not on HEAD")
	}
	if got := runGit(t, wt, "status", "--porcelain"); got != "" {
		t.Errorf("worktree is dirty:\n%s", got)
	}
	last, err := LastSource(wt)
	if err != nil || len(last.Files) != 3 {
		t.Errorf("meta/source.json after the redo = %+v, %v; want 3 files", last, err)
	}
}

func TestValidate(t *testing.T) {
	packs := firstPacks(t)
	// Validate reads the packs only: the worktree need not exist yet.
	missingWorktree := filepath.Join(t.TempDir(), ".work", "raw")
	if err := Validate(missingWorktree, packs, Source{Token: token1, CreatedAt: created1}); err != nil {
		t.Errorf("valid input: %v", err)
	}
	for name, src := range map[string]Source{
		"no token": {CreatedAt: created1},
		"bad date": {Token: token1, CreatedAt: "yesterday"},
		"no date":  {Token: token1},
	} {
		if err := Validate(missingWorktree, packs, src); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
		}
	}
	bad := firstPacks(t)
	bad["kr"] = t.TempDir() // no files
	if err := Validate(missingWorktree, bad, Source{Token: token1, CreatedAt: created1}); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("empty pack: err = %v, want ErrInvalidInput", err)
	}
	if _, err := os.Stat(missingWorktree); !os.IsNotExist(err) {
		t.Errorf("Validate created the worktree path, stat err = %v", err)
	}
}

// storyPack is a pack set whose only KR story file holds text.
func storyPack(t *testing.T, text string) map[string]string {
	return extracted(t,
		map[string]string{"StoryData/KR_S1.json": "{\"dataList\":[{\"content\":\"" + text + "\"}]}\n", "KR_A.json": bomCRLF},
		map[string]string{"EN_A.json": "{}\n"},
		map[string]string{"JP_A.json": "{}\n"})
}

func TestOpenKRAtReadsAnOlderSnapshot(t *testing.T) {
	wt := rawWorktree(t)
	tag1, _, err := Snapshot(wt, storyPack(t, "first"), Source{Token: token1, CreatedAt: created1})
	if err != nil {
		t.Fatal(err)
	}
	tag2, _, err := Snapshot(wt, storyPack(t, "second"), Source{Token: token2, CreatedAt: created2})
	if err != nil {
		t.Fatal(err)
	}
	if tag1 != "raw-20261029" || tag2 != "raw-20261105" {
		t.Fatalf("tags = %s, %s", tag1, tag2)
	}

	dir := filepath.Join(t.TempDir(), "old")
	p, err := OpenKRAt(wt, tag1, dir)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := p.Doc("StoryData/S1.json")
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := doc.Get("/dataList/0/content"); !ok || n.Str != "first" {
		t.Errorf("/dataList/0/content = %+v, %v; want the first version", n, ok)
	}
	// The files are the bytes of the snapshot, and only the KR pack is exported.
	if got := read(t, dir, "kr/KR_A.json"); got != bomCRLF {
		t.Errorf("kr/KR_A.json = %q, want the BOM and CRLF kept", got)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != "kr" {
		t.Errorf("dir holds %v (%v), want only kr", entries, err)
	}
	// The worktree is left alone: still at the newest snapshot, still clean.
	if got := read(t, wt, "kr/StoryData/KR_S1.json"); !strings.Contains(got, "second") {
		t.Errorf("the worktree's file = %q, want the second version", got)
	}
	if got := runGit(t, wt, "status", "--porcelain"); got != "" {
		t.Errorf("the worktree is dirty after OpenKRAt:\n%s", got)
	}

	// The newer tag gives the newer text.
	p2, err := OpenKRAt(wt, tag2, filepath.Join(t.TempDir(), "new"))
	if err != nil {
		t.Fatal(err)
	}
	doc2, err := p2.Doc("StoryData/S1.json")
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := doc2.Get("/dataList/0/content"); !ok || n.Str != "second" {
		t.Errorf("newer snapshot: /dataList/0/content = %+v, %v", n, ok)
	}
}

func TestOpenKRAtUnknownTag(t *testing.T) {
	wt := rawWorktree(t)
	if _, _, err := Snapshot(wt, storyPack(t, "first"), Source{Token: token1, CreatedAt: created1}); err != nil {
		t.Fatal(err)
	}
	// "raw" is the branch the worktree is on, not a tag: it must not stand in
	// for a snapshot.
	for _, tag := range []string{"raw-20250101", "raw"} {
		dir := filepath.Join(t.TempDir(), "old")
		p, err := OpenKRAt(wt, tag, dir)
		if err == nil || !strings.Contains(err.Error(), tag) {
			t.Errorf("OpenKRAt(%q) = %v, %v; want an error naming the tag", tag, p, err)
		}
	}
}
