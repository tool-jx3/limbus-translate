package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/bootstrap"
	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/store"
)

// bootstrapFixture is the small data set shared with internal/bootstrap.
const bootstrapFixture = "../../testdata/bootstrap"

// bootstrapYAML pins three fake releases. Their sha256s are placeholders;
// a test that fetches them pins the real ones with pinReleases.
var bootstrapYAML = bootstrapConfig(map[string]string{})

// The fake releases, by upstream name: where pinReleases finds them below the
// fake GitHub directory.
var fakeReleases = map[string]string{
	"paratranz": "owner/pz/releases/download/11/localize_files.zip",
	"ltm":       "owner/ltm/releases/download/22/Hant.zip",
	"llc":       "owner/llc/releases/download/33/LimbusLocalize_33.zip",
}

// bootstrapConfig is an lct.yaml for the fake releases with the given sha256s
// by upstream name; a missing one is a placeholder of zeros.
func bootstrapConfig(sums map[string]string) string {
	sum := func(name string) string {
		if s, ok := sums[name]; ok {
			return s
		}
		return strings.Repeat("0", 64)
	}
	return `worktrees: { raw: .work/raw, tw: .work/tw }
frozen: ["BgmLyrics/**"]
bootstrap:
  paratranz: { repo: owner/pz, tag: "11", asset: localize_files.zip, sha256: "` + sum("paratranz") + `" }
  ltm: { repo: owner/ltm, tag: "22", asset: Hant.zip, sha256: "` + sum("ltm") + `" }
  llc: { repo: owner/llc, tag: "33", asset: LimbusLocalize_33.zip, sha256: "` + sum("llc") + `" }
`
}

// pinReleases writes root's lct.yaml with the sha256 of each fake release
// found below the fake GitHub directory gh, except those in skip.
func pinReleases(t *testing.T, root, gh string, skip ...string) {
	t.Helper()
	sums := map[string]string{}
	for name, rel := range fakeReleases {
		if slices.Contains(skip, name) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(gh, filepath.FromSlash(rel)))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		sums[name] = hex.EncodeToString(sum[:])
	}
	writeTestFile(t, root, "lct.yaml", bootstrapConfig(sums))
}

// writeFakeReleases writes the three fake releases below gh, made from the
// fixture.
func writeFakeReleases(t *testing.T, gh string) {
	t.Helper()
	writeZipFile(t, filepath.Join(gh, fakeReleases["paratranz"]), map[string]string{
		"Sample.json": readFixtureFile(t, "paratranz/Sample.json"),
		"Bufs.json":   readFixtureFile(t, "paratranz/Bufs.json"),

		"BgmLyrics/Lyrics.json": bgmLyricsParatranz,
	})
	writeZipFile(t, filepath.Join(gh, fakeReleases["ltm"]), map[string]string{
		"Sample.json": readFixtureFile(t, "ltm/Sample.json"),
	})
	writeZipFile(t, filepath.Join(gh, fakeReleases["llc"]), map[string]string{
		"LimbusCompany_Data/Lang/LLC_zh-CN/Sample.json": readFixtureFile(t, "llc/Sample.json"),
	})
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		copyFile(t, p, filepath.Join(dst, rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// speakers are the ScenarioModelCodes files added to the fixture's upstream
// packs: LTM names 뫼르소 differently.
const (
	speakersOurs = `{"dataList":[{"id":"뫼르소","name":"莫梭"},{"id":"이상","name":"李箱"}]}`
	speakersLTM  = `{"dataList":[{"id":"뫼르소","name":"默爾索"},{"id":"이상","name":"李箱"}]}`
)

// A frozen file: lct.yaml freezes BgmLyrics/**, so the Paratranz text is
// adopted as it is, although it has no Han.
const (
	bgmLyricsKR        = `{"dataList":[{"id":1,"content":"가사 하나"}]}`
	bgmLyricsParatranz = `{"dataList":[{"id":1,"content":"La la la"}]}`
	bgmLyricsKey       = "BgmLyrics/Lyrics.json|/dataList/0/content"
)

// bootstrapRepo makes an lct repo root ready for "lct bootstrap": rules, a raw
// worktree holding the fixture packs and a snapshot record, and (unless
// noUpstream) an upstream directory laid out as --fetch leaves it.
func bootstrapRepo(t *testing.T, noUpstream bool) (root, upstream string) {
	t.Helper()
	root = snapshotRepo(t)
	writeTestFile(t, root, "lct.yaml", bootstrapYAML)

	for _, f := range []string{"rules/fields.manual.yaml", "glossary/terms.yaml", "guides/routing.yaml"} {
		copyFile(t, filepath.Join(repoRoot, f), filepath.Join(root, f))
	}
	copyDir(t, filepath.Join(repoRoot, "guides/characters"), filepath.Join(root, "guides/characters"))

	rawWT := filepath.Join(root, ".work", "raw")
	for _, lang := range packLangs {
		copyDir(t, filepath.Join(bootstrapFixture, lang), filepath.Join(rawWT, lang))
	}
	writeTestFile(t, rawWT, "kr/BgmLyrics/KR_Lyrics.json", bgmLyricsKR)
	writeTestFile(t, rawWT, "meta/source.json", `{"tag":"raw-20261029","token":"`+testToken+`","created_at":"`+testCreatedAt+`","fetched_at":"2026-10-29T03:00:00Z","files":{}}`+"\n")

	upstream = filepath.Join(t.TempDir(), "upstream")
	if !noUpstream {
		copyDir(t, filepath.Join(bootstrapFixture, "paratranz"), filepath.Join(upstream, "paratranz"))
		copyDir(t, filepath.Join(bootstrapFixture, "ltm"), filepath.Join(upstream, "ltm"))
		copyDir(t, filepath.Join(bootstrapFixture, "llc"), filepath.Join(upstream, "llc", "LimbusCompany_Data", "Lang", "LLC_zh-CN"))
		writeTestFile(t, upstream, "paratranz/BgmLyrics/Lyrics.json", bgmLyricsParatranz)
		writeTestFile(t, upstream, "paratranz/ScenarioModelCodes-AutoCreated.json", speakersOurs)
		writeTestFile(t, upstream, "ltm/ScenarioModelCodes-AutoCreated.json", speakersLTM)
	}
	return root, upstream
}

type bootstrapOut struct {
	RawTag       string         `json:"raw_tag"`
	Records      int            `json:"records"`
	Pending      int            `json:"pending"`
	ForeignNames int            `json:"foreign_names"`
	Stats        map[string]int `json:"stats"`
}

func runBootstrapJSON(t *testing.T, root string, args ...string) (bootstrapOut, error) {
	t.Helper()
	ctx, out := testCtx(t, root, "", true)
	err := runBootstrap(ctx, args)
	var got bootstrapOut
	if err == nil {
		if err := json.Unmarshal(out.Bytes(), &got); err != nil {
			t.Fatalf("output is not JSON: %v\n%s", err, out)
		}
	}
	return got, err
}

func TestBootstrapImportsFixture(t *testing.T) {
	root, upstream := bootstrapRepo(t, false)
	got, err := runBootstrapJSON(t, root, "--upstream", upstream)
	if err != nil {
		t.Fatal(err)
	}

	wantStats := map[string]int{
		"copy": 2, "paratranz": 3, "paratranz-repair": 1, "new": 1, "unaligned-paratranz": 0, "frozen": 1,
		"refs-llc": 2, "refs-ltm": 2,
	}
	if got.RawTag != "raw-20261029" || got.Records != 5 || got.Pending != 2 || got.ForeignNames != 1 || !reflect.DeepEqual(got.Stats, wantStats) {
		t.Errorf("output = %+v, want raw-20261029, 5 records, 2 pending, 1 foreign name, stats %v", got, wantStats)
	}
	// lct.yaml's frozen list reached the import.
	if got.Stats["frozen"] == 0 {
		t.Error("no frozen adoption: cfg.Frozen did not reach bootstrap.Inputs.Frozen")
	}

	tw := filepath.Join(root, ".work", "tw")
	if branch := runGitIn(t, tw, "symbolic-ref", "--short", "HEAD"); branch != "tw" {
		t.Errorf("worktree is on %q, want tw", branch)
	}
	st := store.Open(tw)
	recs, err := st.LoadRecords()
	if err != nil {
		t.Fatal(err)
	}
	if r := recs["Sample.json|/dataList/1/content"]; r.ZH != "增加5級 [Laceration] 強度" || r.Origin != store.OriginParatranz || r.Since != "raw-20261029" {
		t.Errorf("adopted record = %+v", r)
	}
	// The rules were copied in after the repo's only commit: they are dirty.
	wantRules := runGitIn(t, root, "rev-parse", "HEAD") + "+dirty"
	for key, r := range recs {
		if r.Rules != wantRules {
			t.Errorf("record %s: rules = %q, want %q", key, r.Rules, wantRules)
		}
	}
	if r := recs[bgmLyricsKey]; r.ZH != "La la la" || r.Origin != store.OriginParatranz || len(r.Flags) != 0 {
		t.Errorf("frozen record = %+v, want the Paratranz text verbatim", r)
	}
	pending, err := st.LoadPending()
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 2 || pending[0].Reason != "paratranz-repair" || pending[1].Reason != "new" {
		t.Errorf("pending = %+v, want the repair item and the new item in key order", pending)
	}
	names, err := os.ReadFile(filepath.Join(tw, ".meta", "foreign-names.json"))
	if err != nil || string(names) != "[\n  \"默爾索\"\n]\n" {
		t.Errorf("foreign-names.json = %q, %v", names, err)
	}
	var read []string
	if found, err := st.ReadMetaJSON(bootstrap.ForeignNamesFile, &read); err != nil || !found || !reflect.DeepEqual(read, []string{"默爾索"}) {
		t.Errorf("ReadMetaJSON(%s) = %q, %v, %v", bootstrap.ForeignNamesFile, read, found, err)
	}
}

func TestBootstrapTextOutput(t *testing.T) {
	root, upstream := bootstrapRepo(t, false)
	ctx, out := testCtx(t, root, "", false)
	if err := runBootstrap(ctx, []string{"--upstream", upstream}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"raw-20261029", "paratranz-repair", "unaligned-paratranz", "5 records", "2 pending"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output does not mention %q:\n%s", want, out)
		}
	}
}

func TestBootstrapEmptyForeignNamesIsEmptyList(t *testing.T) {
	root, upstream := bootstrapRepo(t, false)
	for _, side := range []string{"paratranz", "ltm"} {
		if err := os.Remove(filepath.Join(upstream, side, "ScenarioModelCodes-AutoCreated.json")); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := runBootstrapJSON(t, root, "--upstream", upstream); err != nil {
		t.Fatal(err)
	}
	names, err := os.ReadFile(filepath.Join(root, ".work", "tw", ".meta", "foreign-names.json"))
	if err != nil || string(names) != "[]\n" {
		t.Errorf("foreign-names.json = %q, %v, want []", names, err)
	}
}

func TestBootstrapRefusesToOverwriteWithoutForce(t *testing.T) {
	root, upstream := bootstrapRepo(t, false)
	if _, err := runBootstrapJSON(t, root, "--upstream", upstream); err != nil {
		t.Fatal(err)
	}
	st := store.Open(filepath.Join(root, ".work", "tw"))
	before, err := st.LoadRecords()
	if err != nil {
		t.Fatal(err)
	}

	// A person's work since the import must survive a second run.
	edited := before["Sample.json|/dataList/1/content"]
	edited.ZH, edited.Origin = "手改的譯文", store.OriginHuman
	before[edited.Key] = edited
	if err := st.SaveRecords(before); err != nil {
		t.Fatal(err)
	}

	_, err = runBootstrapJSON(t, root, "--upstream", upstream)
	if !isUserError(err) || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("err = %v (%T), want a user error that mentions --force", err, err)
	}
	after, err := st.LoadRecords()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Error("a refused bootstrap changed the records")
	}

	// A store with only pending items is just as protected.
	if err := st.SaveRecords(map[string]store.Record{}); err != nil {
		t.Fatal(err)
	}
	if _, err := runBootstrapJSON(t, root, "--upstream", upstream); !isUserError(err) {
		t.Errorf("pending items alone: err = %v, want a user error", err)
	}
}

func TestBootstrapForceReplacesTheStore(t *testing.T) {
	root, upstream := bootstrapRepo(t, false)
	tw := filepath.Join(root, ".work", "tw")
	if _, err := runBootstrapJSON(t, root, "--upstream", upstream); err != nil {
		t.Fatal(err)
	}
	st := store.Open(tw)
	recs, err := st.LoadRecords()
	if err != nil {
		t.Fatal(err)
	}
	recs["Old.json|/dataList/0/name"] = store.Record{Key: "Old.json|/dataList/0/name", ZH: "舊", Origin: store.OriginClaude}
	if err := st.SaveRecords(recs); err != nil {
		t.Fatal(err)
	}
	if err := st.SavePending([]store.PendingItem{{Key: "Old.json|/dataList/1/name", Reason: "requeue"}}); err != nil {
		t.Fatal(err)
	}

	got, err := runBootstrapJSON(t, root, "--upstream", upstream, "--force")
	if err != nil {
		t.Fatal(err)
	}
	if got.Records != 5 || got.Pending != 2 {
		t.Errorf("output = %+v, want 5 records and 2 pending", got)
	}
	recs, err = st.LoadRecords()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := recs["Old.json|/dataList/0/name"]; ok || len(recs) != 5 {
		t.Errorf("after --force: %d records, stale one present: %v", len(recs), ok)
	}
	pending, err := st.LoadPending()
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 2 {
		t.Errorf("after --force: %d pending items, want 2: %+v", len(pending), pending)
	}
	if _, err := os.Stat(filepath.Join(tw, ".meta", "units", "Old.json.jsonl")); !os.IsNotExist(err) {
		t.Errorf("the stale record file is still there: %v", err)
	}
}

func TestBootstrapUsageErrors(t *testing.T) {
	root, upstream := bootstrapRepo(t, false)
	noSnapshot, upstream2 := bootstrapRepo(t, false)
	if err := os.Remove(filepath.Join(noSnapshot, ".work", "raw", "meta", "source.json")); err != nil {
		t.Fatal(err)
	}
	missingUpstream, _ := bootstrapRepo(t, true)
	unborn, upstream4 := bootstrapRepo(t, false)
	runGitIn(t, unborn, "checkout", "-q", "--orphan", "unborn")
	noLLC, upstream3 := bootstrapRepo(t, false)
	if err := os.RemoveAll(filepath.Join(upstream3, "llc")); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		root string
		args []string
		want string
	}{
		{"stray argument", root, []string{"--upstream", upstream, "stray"}, "stray"},
		{"unknown flag", root, []string{"--nope"}, "nope"},
		{"no raw snapshot", noSnapshot, []string{"--upstream", upstream2}, "snapshot"},
		{"no upstream directory", missingUpstream, []string{"--upstream", filepath.Join(missingUpstream, "nowhere")}, "--fetch"},
		{"no LLC directory", noLLC, []string{"--upstream", upstream3}, "llc"},
		{"repo without commits", unborn, []string{"--upstream", upstream4}, "no commit"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runBootstrapJSON(t, tc.root, tc.args...)
			if !isUserError(err) {
				t.Fatalf("err = %v (%T), want a user error", err, err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %q, want it to mention %q", err, tc.want)
			}
		})
	}
	// Refusing must leave no tw worktree behind.
	if _, err := os.Stat(filepath.Join(noSnapshot, ".work", "tw")); !os.IsNotExist(err) {
		t.Errorf("a refused bootstrap created the tw worktree: %v", err)
	}
}

// An empty KR or Paratranz pack has nothing to import; with --force it would
// replace a good store with an empty one. It is refused before the tw
// worktree is touched.
func TestBootstrapRefusesEmptyPacks(t *testing.T) {
	for _, tc := range []struct {
		name  string
		empty func(root, upstream string) string // empties a pack, returns its directory
	}{
		{"KR", func(root, _ string) string { return filepath.Join(root, ".work", "raw", "kr") }},
		{"Paratranz", func(_, upstream string) string { return filepath.Join(upstream, "paratranz") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, upstream := bootstrapRepo(t, false)
			if _, err := runBootstrapJSON(t, root, "--upstream", upstream); err != nil {
				t.Fatal(err)
			}
			st := store.Open(filepath.Join(root, ".work", "tw"))
			before, err := st.LoadRecords()
			if err != nil {
				t.Fatal(err)
			}
			dir := tc.empty(root, upstream)
			if err := os.RemoveAll(dir); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}

			_, err = runBootstrapJSON(t, root, "--upstream", upstream, "--force")
			if !isUserError(err) || !strings.Contains(err.Error(), "no files") {
				t.Fatalf("err = %v (%T), want a user error saying the pack has no files", err, err)
			}
			after, err := st.LoadRecords()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) {
				t.Error("a refused bootstrap changed the records")
			}
		})
	}

	// Nothing is created either.
	fresh, upstream := bootstrapRepo(t, false)
	if err := os.RemoveAll(filepath.Join(upstream, "paratranz")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(upstream, "paratranz"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := runBootstrapJSON(t, fresh, "--upstream", upstream); !isUserError(err) {
		t.Fatalf("err = %v, want a user error", err)
	}
	if _, err := os.Stat(filepath.Join(fresh, ".work", "tw")); !os.IsNotExist(err) {
		t.Errorf("a refused bootstrap created the tw worktree: %v", err)
	}
}

func TestUpstreamURL(t *testing.T) {
	up := config.Upstream{Repo: "user-unknown-0711/ParatranzUploader", Tag: "1793285016", Asset: "localize_files.zip"}
	want := "https://github.com/user-unknown-0711/ParatranzUploader/releases/download/1793285016/localize_files.zip"
	if got := upstreamURL("https://github.com", up); got != want {
		t.Errorf("upstreamURL = %q, want %q", got, want)
	}
	if upstreamBase != "https://github.com" {
		t.Errorf("upstreamBase = %q, want https://github.com", upstreamBase)
	}
}

// useUpstreamBase points --fetch at base for the rest of the test.
func useUpstreamBase(t *testing.T, base string) {
	t.Helper()
	old := upstreamBase
	upstreamBase = base
	t.Cleanup(func() { upstreamBase = old })
}

// writeZipFile writes a zip holding files (name to content) at p.
func writeZipFile(t *testing.T, p string, files map[string]string) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFixtureFile(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(bootstrapFixture, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestBootstrapFetchDownloadsAndExtractsThePinnedAssets(t *testing.T) {
	root, upstream := bootstrapRepo(t, true)

	// A fake GitHub: <base>/<repo>/releases/download/<tag>/<asset>.
	gh := t.TempDir()
	writeFakeReleases(t, gh)
	pinReleases(t, root, gh)
	useUpstreamBase(t, "file://"+gh)

	// What an earlier fetch left behind must not survive.
	writeTestFile(t, upstream, "paratranz/Stale.json", `{"dataList":[]}`)

	got, err := runBootstrapJSON(t, root, "--fetch", "--upstream", upstream)
	if err != nil {
		t.Fatal(err)
	}
	if got.Records != 5 || got.Pending != 2 {
		t.Errorf("output = %+v, want 5 records and 2 pending", got)
	}
	for _, rel := range []string{"paratranz.zip", "ltm.zip", "llc.zip", "paratranz/Sample.json", "llc/LimbusCompany_Data/Lang/LLC_zh-CN/Sample.json"} {
		if _, err := os.Stat(filepath.Join(upstream, filepath.FromSlash(rel))); err != nil {
			t.Errorf("--fetch left no %s: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(upstream, "paratranz", "Stale.json")); !os.IsNotExist(err) {
		t.Errorf("a file of the earlier fetch survived: %v", err)
	}
}

func TestBootstrapFetchFailureIsASystemError(t *testing.T) {
	root, upstream := bootstrapRepo(t, true)
	useUpstreamBase(t, "file://"+t.TempDir()) // no release in there

	_, err := runBootstrapJSON(t, root, "--fetch", "--upstream", upstream)
	if err == nil || isUserError(err) {
		t.Errorf("err = %v, want a system error for a failed download", err)
	}
}

// Every asset is checked against its pinned sha256 before anything is
// extracted: a mismatch is a user error naming the asset, and it leaves the
// upstream directory as it was.
func TestBootstrapFetchVerifiesTheChecksums(t *testing.T) {
	root, upstream := bootstrapRepo(t, false)
	writeTestFile(t, upstream, "ltm.zip", "the zip of an earlier fetch")
	before := listTree(t, upstream)

	gh := t.TempDir()
	writeFakeReleases(t, gh)
	pinReleases(t, root, gh, "ltm") // ltm keeps the placeholder
	useUpstreamBase(t, "file://"+gh)

	_, err := runBootstrapJSON(t, root, "--fetch", "--upstream", upstream)
	if !isUserError(err) || !strings.Contains(err.Error(), "Hant.zip") || !strings.Contains(err.Error(), "sha256") {
		t.Fatalf("err = %v (%T), want a user error about the sha256 of Hant.zip", err, err)
	}
	if after := listTree(t, upstream); !reflect.DeepEqual(after, before) {
		t.Errorf("the upstream directory changed:\n%v\nwas\n%v", after, before)
	}

	// An asset without a pin is not downloaded at all.
	writeTestFile(t, root, "lct.yaml", strings.Replace(bootstrapYAML, `, sha256: "`+strings.Repeat("0", 64)+`" }`, " }", 1))
	_, err = runBootstrapJSON(t, root, "--fetch", "--upstream", upstream)
	if !isUserError(err) || !strings.Contains(err.Error(), "sha256") {
		t.Errorf("no pin: err = %v (%T), want a user error naming sha256", err, err)
	}
}

// listTree maps every file below dir to its content.
func listTree(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(p)
		files[p] = string(data)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestBootstrapIsRegistered(t *testing.T) {
	if !strings.Contains(strings.Join(cli.Commands(), " "), "bootstrap") {
		t.Errorf("bootstrap is not registered (have: %v)", cli.Commands())
	}
}

func TestBootstrapFetchChecksBeforeDownloading(t *testing.T) {
	// The base holds no release: any download would fail with a system error.
	useUpstreamBase(t, "file://"+t.TempDir())

	// No raw snapshot: refused before anything is downloaded.
	noSnapshot, upstream := bootstrapRepo(t, true)
	if err := os.Remove(filepath.Join(noSnapshot, ".work", "raw", "meta", "source.json")); err != nil {
		t.Fatal(err)
	}
	_, err := runBootstrapJSON(t, noSnapshot, "--fetch", "--upstream", upstream)
	if !isUserError(err) || !strings.Contains(err.Error(), "snapshot") {
		t.Errorf("no snapshot: err = %v (%T), want a user error about the snapshot", err, err)
	}
	if _, err := os.Stat(upstream); !os.IsNotExist(err) {
		t.Errorf("no snapshot: the upstream directory was touched: %v", err)
	}

	// A store that already has work: refused before anything is downloaded,
	// unless --force says to replace it.
	root, full := bootstrapRepo(t, false)
	if _, err := runBootstrapJSON(t, root, "--upstream", full); err != nil {
		t.Fatal(err)
	}
	fresh := filepath.Join(t.TempDir(), "upstream")
	_, err = runBootstrapJSON(t, root, "--fetch", "--upstream", fresh)
	if !isUserError(err) || !strings.Contains(err.Error(), "--force") {
		t.Errorf("existing store: err = %v (%T), want a user error that mentions --force", err, err)
	}
	if _, err := os.Stat(fresh); !os.IsNotExist(err) {
		t.Errorf("existing store: the upstream directory was touched: %v", err)
	}
	_, err = runBootstrapJSON(t, root, "--fetch", "--force", "--upstream", fresh)
	if err == nil || isUserError(err) {
		t.Errorf("--force: err = %v, want it to get as far as the (failing) download", err)
	}
}

// zipWithBadChecksum is a stored zip of two files whose second file fails its
// CRC check, so extraction writes the first file and then fails.
func zipWithBadChecksum(t *testing.T, p string) {
	t.Helper()
	const good, bad = `{"dataList":[1]}`, `{"dataList":[2]}`
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range []struct{ name, content string }{{"First.json", good}, {"Second.json", good}} {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: f.name, Method: zip.Store})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(f.content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	data := buf.Bytes()
	// The content of the second entry appears once, after the first's.
	i := bytes.LastIndex(data, []byte(good))
	copy(data[i:], bad)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBootstrapFetchFailureKeepsThePreviousDirectories(t *testing.T) {
	root, upstream := bootstrapRepo(t, false)
	gh := t.TempDir()
	writeFakeReleases(t, gh)
	zipWithBadChecksum(t, filepath.Join(gh, fakeReleases["paratranz"]))
	pinReleases(t, root, gh)
	useUpstreamBase(t, "file://"+gh)

	_, err := runBootstrapJSON(t, root, "--fetch", "--upstream", upstream)
	if err == nil || isUserError(err) {
		t.Fatalf("err = %v, want a system error from the bad archive", err)
	}
	if !strings.Contains(err.Error(), "checksum") {
		t.Errorf("err = %v, want the checksum failure", err)
	}
	// The earlier extraction is intact and no half-extracted directory is left.
	if _, err := os.Stat(filepath.Join(upstream, "paratranz", "Sample.json")); err != nil {
		t.Errorf("the previous paratranz directory was damaged: %v", err)
	}
	if _, err := os.Stat(filepath.Join(upstream, "paratranz.tmp")); !os.IsNotExist(err) {
		t.Errorf("the staging directory is still there: %v", err)
	}
}

func TestBootstrapFetchLeavesNoStagingDirectory(t *testing.T) {
	root, upstream := bootstrapRepo(t, true)
	gh := t.TempDir()
	writeFakeReleases(t, gh)
	pinReleases(t, root, gh)
	useUpstreamBase(t, "file://"+gh)

	// A staging directory left by a crashed fetch must not leak into the result.
	writeTestFile(t, upstream, "paratranz.tmp/Crashed.json", `{"dataList":[]}`)
	if _, err := runBootstrapJSON(t, root, "--fetch", "--upstream", upstream); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"paratranz", "ltm", "llc"} {
		if _, err := os.Stat(filepath.Join(upstream, name+".tmp")); !os.IsNotExist(err) {
			t.Errorf("%s.tmp is still there: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(upstream, "paratranz", "Crashed.json")); !os.IsNotExist(err) {
		t.Errorf("a file of the crashed staging directory ended up in paratranz: %v", err)
	}
}
