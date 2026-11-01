package batch_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"golang.org/x/image/font/gofont/goregular"

	"github.com/tool-jx3/limbus-translate/internal/batch"
	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// This file is an external test package: build imports plan, which imports
// batch, so a test of batch that builds cannot be in package batch.

const (
	testTag          = "raw-20261029"
	testRoutingYAML  = "families:\n  - id: other\n    match: []\n    guide: guides/families.md\npriority:\n  - other\n"
	testGlossaryYAML = "- kr: 위력\n  zh: 威力\n  status: approved\n  forbidden: [力量]\n"
)

var testNow = time.Date(2026, 11, 1, 3, 4, 5, 0, time.UTC)

func writeFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newEnv makes a raw checkout of the given KR and EN files (JP is empty), the
// rules of the tests and the batch environment of the tw worktree at twRoot.
func newEnv(t *testing.T, twRoot string, kr, en map[string]string) batch.Env {
	t.Helper()
	raw := t.TempDir()
	for _, l := range []struct {
		lang, prefix string
		files        map[string]string
	}{{"kr", "KR_", kr}, {"en", "EN_", en}, {"jp", "JP_", nil}} {
		if err := os.MkdirAll(filepath.Join(raw, l.lang), 0o755); err != nil {
			t.Fatal(err)
		}
		for file, text := range l.files {
			writeFile(t, filepath.Join(raw, l.lang, l.prefix+file), text)
		}
	}
	src, err := pack.OpenRaw(raw)
	if err != nil {
		t.Fatal(err)
	}

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "guides", "routing.yaml"), testRoutingYAML)
	writeFile(t, filepath.Join(root, "glossary", "terms.yaml"), testGlossaryYAML)
	if err := os.MkdirAll(filepath.Join(root, "guides", "characters"), 0o755); err != nil {
		t.Fatal(err)
	}
	r, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return batch.Env{Store: store.Open(twRoot), Src: src, Rules: r, RawTag: testTag, Now: testNow}
}

// A requeued unit that cannot be translated again keeps its good record; the
// record is flagged needs_review, so that the build reports it: a unit stuck in
// the queue is not left unseen.
func TestSubmitFlagsAKeptRecordNeedsReview(t *testing.T) {
	const kr = "[Bleed] 위력"
	krFiles := map[string]string{"Other.json": `{"dataList":[{"id":0,"content":"[Bleed] 위력"}]}`}
	enFiles := map[string]string{"Other.json": `{"dataList":[{"id":0,"content":"[Bleed] Potency"}]}`}
	k := "Other.json|/dataList/0/content"
	tw := filepath.Join(t.TempDir(), "tw")
	env := newEnv(t, tw, krFiles, enFiles)
	good := store.Record{Key: k, KRSha: units.KRSha(kr), ZH: "賦予[Bleed]威力", Origin: store.OriginClaude, Since: "raw-20261001", Batch: "old-batch"}
	if err := env.Store.SaveRecords(map[string]store.Record{k: good}); err != nil {
		t.Fatal(err)
	}
	if err := env.Store.SavePending([]store.PendingItem{{Key: k, Reason: "requeue", PrevZH: good.ZH, Hint: "稱呼改用「您」"}}); err != nil {
		t.Fatal(err)
	}
	b, err := batch.Next(env, batch.NextOptions{})
	if err != nil || b == nil {
		t.Fatalf("Next: %v, %v", b, err)
	}
	for range 3 {
		if _, err := batch.Submit(env, b, []batch.OutUnit{{Key: k, ZH: "賦予流血"}}, "m", 3); err != nil { // [Bleed] is missing
			t.Fatal(err)
		}
	}

	recs, err := env.Store.LoadRecords()
	if err != nil {
		t.Fatal(err)
	}
	got := recs[k]
	if got.ZH != good.ZH || got.Origin != good.Origin || !slices.Equal(got.Flags, []string{store.FlagNeedsReview}) {
		t.Fatalf("record = %+v, want the good translation kept and flagged needs_review", got)
	}

	rep, err := build.Run(buildOptions(t, tw), env.Src, mustFieldRules(t))
	if err != nil {
		t.Fatal(err)
	}
	if rep.NeedsReview != 1 {
		t.Errorf("NeedsReview = %d, want 1: the kept record is reported", rep.NeedsReview)
	}
}

// buildOptions are the options of a build into the tw worktree at tw. The
// font is Go Regular: small, with Latin letters and no Han.
func buildOptions(t *testing.T, tw string) build.Options {
	t.Helper()
	dir := t.TempDir()
	font := filepath.Join(dir, "font.ttf")
	writeFile(t, font, string(goregular.TTF))
	writeFile(t, filepath.Join(dir, "OFL.txt"), "OFL text\n")
	writeFile(t, filepath.Join(dir, "LICENSE-CONTENT"), "content license\n")
	sum := sha256.Sum256(goregular.TTF)
	return build.Options{
		TWRoot: tw, OutDir: filepath.Join(dir, "out"), LangDir: "zh-TW",
		FontPath: font, FontSHA256: hex.EncodeToString(sum[:]), LicensePath: filepath.Join(dir, "LICENSE-CONTENT"),
		RawTag: testTag, RawToken: "token", RulesCommit: "0123abc", BuiltAt: testNow,
	}
}

func mustFieldRules(t *testing.T) *fields.Rules {
	t.Helper()
	fr, err := fields.Load("../..")
	if err != nil {
		t.Fatal(err)
	}
	return fr
}
