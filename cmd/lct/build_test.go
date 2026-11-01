package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/image/font/gofont/goregular"

	"github.com/tool-jx3/limbus-translate/internal/build"
)

// buildYAML is the lct.yaml of the bootstrap tests with the language directory.
const buildYAML = "lang_dir: zh-TW\n" + bootstrapYAML

// buildRepo makes the repo of the bootstrap tests with a font (Go Regular, a
// small stand-in for the pinned one), its licence and the content licence,
// and imports the fixture, so that the tw store has records.
func buildRepo(t *testing.T) string {
	t.Helper()
	root, upstream := bootstrapRepo(t, false)
	sum := sha256.Sum256(goregular.TTF)
	writeTestFile(t, root, "lct.yaml", buildYAML+
		"font: { path: assets/fonts/SarasaGothicTC-Bold.ttf, version: \"1.0.43\", sha256: \""+hex.EncodeToString(sum[:])+"\" }\n")
	if err := os.MkdirAll(filepath.Join(root, "assets", "fonts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "assets", "fonts", "SarasaGothicTC-Bold.ttf"), goregular.TTF, 0o644); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, "assets/fonts/OFL.txt", "OFL text\n")
	writeTestFile(t, root, "LICENSE-CONTENT", "content license\n")
	if _, err := runBootstrapJSON(t, root, "--upstream", upstream); err != nil {
		t.Fatal(err)
	}
	return root
}

type buildOut struct {
	RawTag      string        `json:"raw_tag"`
	TextZip     string        `json:"text_zip"`
	CompleteZip string        `json:"complete_zip"`
	Report      *build.Report `json:"report"`
	ReportPath  string        `json:"report_path"`
}

func runBuildJSON(t *testing.T, root string, args ...string) (buildOut, error) {
	t.Helper()
	ctx, out := testCtx(t, root, "", true)
	err := runBuild(ctx, args)
	var got buildOut
	if err == nil {
		if err := json.Unmarshal(out.Bytes(), &got); err != nil {
			t.Fatalf("output is not JSON: %v\n%s", err, out)
		}
	}
	return got, err
}

func zipNames(t *testing.T, name string) []string {
	t.Helper()
	zr, err := zip.OpenReader(name)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	return names
}

func TestBuildCommandBuildsThePack(t *testing.T) {
	root := buildRepo(t)
	t.Setenv(sourceDateEpoch, "")
	got, err := runBuildJSON(t, root)
	if err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(root, "out")
	if got.RawTag != "raw-20261029" ||
		got.TextZip != filepath.Join(out, "zh-TW_text_raw-20261029.zip") ||
		got.CompleteZip != filepath.Join(out, "zh-TW_complete_raw-20261029.zip") ||
		got.ReportPath != filepath.Join(root, ".work", "tw", ".meta", "reports", "raw-20261029.md") {
		t.Errorf("output = %+v", got)
	}
	// The fixture import leaves two units to translate; they show EN until then.
	if got.Report == nil || got.Report.Pending != 2 || got.Report.StructuralFallback != 0 {
		t.Errorf("report = %+v, want 2 pending", got.Report)
	}

	tw := filepath.Join(root, ".work", "tw")
	lang := filepath.Join(tw, "LimbusCompany_Data", "Lang")
	if b, err := os.ReadFile(filepath.Join(lang, "config.json")); err != nil || string(b) != "{\"lang\": \"zh-TW\"}\n" {
		t.Errorf("config.json = %q, %v", b, err)
	}
	sample, err := os.ReadFile(filepath.Join(lang, "zh-TW", "Sample.json"))
	if err != nil || !strings.Contains(string(sample), "增加5級 [Laceration] 強度") || !strings.Contains(string(sample), "<color=#f8c200>") {
		t.Errorf("Sample.json = %q, %v", sample, err)
	}
	for _, name := range []string{got.ReportPath, filepath.Join(tw, ".meta", "build.json"), got.TextZip, got.CompleteZip} {
		if _, err := os.Stat(name); err != nil {
			t.Error(err)
		}
	}
	if names := zipNames(t, got.CompleteZip); len(names) <= len(zipNames(t, got.TextZip)) {
		t.Errorf("the complete zip (%d files) must hold more than the text zip (%d)", len(names), len(zipNames(t, got.TextZip)))
	}

	// The inputs are recorded; the time is the one of the commit.
	var meta struct {
		RawTag      string            `json:"raw_tag"`
		RawToken    string            `json:"raw_token"`
		RulesCommit string            `json:"rules_commit"`
		BuiltAt     string            `json:"built_at"`
		Upstream    map[string]string `json:"upstream"`
	}
	b, err := os.ReadFile(filepath.Join(lang, "zh-TW", "Info", "version.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &meta); err != nil {
		t.Fatal(err)
	}
	sec, err := strconv.ParseInt(runGitIn(t, root, "log", "-1", "--format=%ct"), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	wantTime := time.Unix(sec, 0).UTC().Format(time.RFC3339)
	if meta.RawTag != "raw-20261029" || meta.RawToken != testToken || meta.RulesCommit != runGitIn(t, root, "rev-parse", "HEAD") || meta.BuiltAt != wantTime {
		t.Errorf("version.json = %+v, want the commit time %s", meta, wantTime)
	}
	if meta.Upstream["paratranz"] != "owner/pz@11" || meta.Upstream["ltm"] != "owner/ltm@22" || meta.Upstream["llc"] != "owner/llc@33" {
		t.Errorf("upstream = %v", meta.Upstream)
	}

	// Building again gives the same bytes.
	before, err := os.ReadFile(got.CompleteZip)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runBuildJSON(t, root); err != nil {
		t.Fatal(err)
	}
	if after, err := os.ReadFile(got.CompleteZip); err != nil || string(after) != string(before) {
		t.Errorf("a second build changed the complete zip (%v)", err)
	}
}

func TestBuildCommandJSONHasTheReportObject(t *testing.T) {
	root := buildRepo(t)
	t.Setenv(sourceDateEpoch, "")
	ctx, out := testCtx(t, root, "", true)
	if err := runBuild(ctx, nil); err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	var keys []string
	for k := range fields {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	if want := []string{"complete_zip", "raw_tag", "report", "report_path", "text_zip"}; !slices.Equal(keys, want) {
		t.Errorf("output keys = %v, want %v", keys, want)
	}

	// "report" is the Report that .meta/build.json holds.
	var fromOutput build.Report
	if err := json.Unmarshal(fields["report"], &fromOutput); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, ".work", "tw", ".meta", "build.json"))
	if err != nil {
		t.Fatal(err)
	}
	var meta struct {
		Report build.Report `json:"report"`
	}
	if err := json.Unmarshal(b, &meta); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fromOutput, meta.Report) || fromOutput.Pending != 2 {
		t.Errorf("report = %+v, build.json has %+v", fromOutput, meta.Report)
	}
	var path string
	if err := json.Unmarshal(fields["report_path"], &path); err != nil || !strings.HasSuffix(path, "raw-20261029.md") {
		t.Errorf("report_path = %q, %v", path, err)
	}
}

func TestBuildCommandUsesSourceDateEpoch(t *testing.T) {
	root := buildRepo(t)
	t.Setenv(sourceDateEpoch, "1793500000")
	if _, err := runBuildJSON(t, root); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, ".work", "tw", "LimbusCompany_Data", "Lang", "zh-TW", "Info", "version.json"))
	if err != nil || !strings.Contains(string(b), `"built_at": "`+time.Unix(1793500000, 0).UTC().Format(time.RFC3339)+`"`) {
		t.Errorf("version.json = %q, %v", b, err)
	}

	for _, bad := range []string{"yesterday", "-5", "1.5", "0"} {
		t.Setenv(sourceDateEpoch, bad)
		if _, err := runBuildJSON(t, root); !isUserError(err) || !strings.Contains(err.Error(), sourceDateEpoch) {
			t.Errorf("%s=%q: err = %v, want a user error naming the variable", sourceDateEpoch, bad, err)
		}
	}
}

func TestBuildCommandRefusesWhatItCannotBuild(t *testing.T) {
	t.Setenv(sourceDateEpoch, "")
	tests := []struct {
		name string
		edit func(t *testing.T, root string)
		args []string
		want string
	}{
		{"wrong font checksum", func(t *testing.T, root string) {
			writeTestFile(t, root, "lct.yaml", buildYAML+"font: { path: assets/fonts/SarasaGothicTC-Bold.ttf, sha256: \""+strings.Repeat("0", 64)+"\" }\n")
		}, nil, "checksum"},
		{"no font in lct.yaml", func(t *testing.T, root string) {
			writeTestFile(t, root, "lct.yaml", buildYAML)
		}, nil, "font.sha256"},
		{"font file missing", func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, "assets", "fonts", "SarasaGothicTC-Bold.ttf")); err != nil {
				t.Fatal(err)
			}
		}, nil, "SarasaGothicTC-Bold.ttf"},
		{"font licence missing", func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, "assets", "fonts", "OFL.txt")); err != nil {
				t.Fatal(err)
			}
		}, nil, "OFL.txt"},
		{"repo without commits", func(t *testing.T, root string) {
			// With SOURCE_DATE_EPOCH unset, the build time would come from
			// HEAD: the missing commit must be reported before it is asked for.
			runGitIn(t, root, "checkout", "-q", "--orphan", "unborn")
		}, nil, "no commit"},
		{"content licence missing", func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, "LICENSE-CONTENT")); err != nil {
				t.Fatal(err)
			}
		}, nil, "LICENSE-CONTENT"},
		{"no snapshot", func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, ".work", "raw", "meta", "source.json")); err != nil {
				t.Fatal(err)
			}
		}, nil, "lct snapshot"},
		{"stray argument", func(t *testing.T, root string) {}, []string{"extra"}, "unexpected argument"},
		{"unknown flag", func(t *testing.T, root string) {}, []string{"--nope"}, "build:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := buildRepo(t)
			tt.edit(t, root)
			_, err := runBuildJSON(t, root, tt.args...)
			if !isUserError(err) || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want a user error mentioning %q", err, tt.want)
			}
			if _, err := os.Stat(filepath.Join(root, "out")); err == nil {
				t.Error("out/ was created although the build was refused")
			}
			if _, err := os.Stat(filepath.Join(root, ".work", "tw", "LimbusCompany_Data")); err == nil {
				t.Error("the pack was written although the build was refused")
			}
		})
	}
}

func TestBuildCommandTextOutput(t *testing.T) {
	root := buildRepo(t)
	t.Setenv(sourceDateEpoch, "")
	ctx, out := testCtx(t, root, "", false)
	if err := runBuild(ctx, nil); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"raw-20261029", "2 pending", "zh-TW_text_raw-20261029.zip", "zh-TW_complete_raw-20261029.zip", "raw-20261029.md"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output does not mention %q:\n%s", want, out)
		}
	}
}
