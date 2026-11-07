package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/audit"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/translate"
)

func TestAuditRanksWavesAndWritesNothing(t *testing.T) {
	s := acceptRepo(t)
	var out auditOutput
	mustLct(t, &out, "audit", "--wave-size", "1")
	if out.Candidates < 2 || len(out.Waves) != out.Candidates {
		t.Fatalf("audit = %+v, want at least 2 candidates and one wave each", out)
	}
	for _, w := range out.Waves {
		if w.Units != 1 || len(w.Keys) != 0 || w.EstimatedUSD != nil {
			t.Errorf("wave %+v: want one unit, no keys, no estimate without a translation record", w)
		}
		if w.Signals[audit.SignalFormat] == 0 {
			t.Errorf("wave %+v has no format signal; the fixture's broken records warn", w)
		}
	}
	if out.USDPerUnit != nil {
		t.Errorf("rate = %v, want null without .meta/translate.json", *out.USDPerUnit)
	}
	if left, _ := filepath.Glob(filepath.Join(s.root, "rules", "requeue", "*.yaml")); len(left) != 0 {
		t.Errorf("audit without --to-file left %v", left)
	}

	// --wave lists the keys of that wave only.
	var one auditOutput
	mustLct(t, &one, "audit", "--wave-size", "1", "--wave", "2")
	for _, w := range one.Waves {
		if (w.Number == 2) != (len(w.Keys) == 1) {
			t.Errorf("wave %d keys = %v", w.Number, w.Keys)
		}
	}
	code, _, stderr := lct(t, "audit", "--wave", "99")
	if code != 1 || !strings.Contains(stderr, "waves") {
		t.Errorf("--wave 99: exit %d, stderr %q", code, stderr)
	}
}

func TestAuditCost(t *testing.T) {
	s := acceptRepo(t)
	st := store.Open(filepath.Join(s.root, ".work", "tw"))
	write := func(res translate.Result) {
		if err := st.WriteMetaJSON(translate.MetaFile, res); err != nil {
			t.Fatal(err)
		}
	}
	write(translate.Result{CostUSD: 3, PendingBefore: 100, PendingAfter: 40})
	var out auditOutput
	mustLct(t, &out, "audit")
	if out.USDPerUnit == nil || *out.USDPerUnit != 0.05 {
		t.Fatalf("rate = %v, want 0.05", out.USDPerUnit)
	}
	if got := *out.Waves[0].EstimatedUSD; got != 0.05*float64(out.Waves[0].Units) {
		t.Errorf("estimate = %v", got)
	}

	write(translate.Result{CostUSD: 3, PendingBefore: 40, PendingAfter: 40})
	mustLct(t, &out, "audit")
	if out.USDPerUnit != nil || out.Waves[0].EstimatedUSD != nil {
		t.Errorf("a run that cleared nothing gave rate %v", out.USDPerUnit)
	}

	mustLct(t, &out, "audit", "--usd-per-unit", "0.5")
	if out.USDPerUnit == nil || *out.USDPerUnit != 0.5 || *out.Waves[0].EstimatedUSD != 0.5*float64(out.Waves[0].Units) {
		t.Errorf("--usd-per-unit not used: %+v", out)
	}
}

func TestAuditToFileWritesWhatRequeueWrites(t *testing.T) {
	s := acceptRepo(t)
	old := ruleFileNow
	ruleFileNow = func() time.Time { return time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { ruleFileNow = old })

	var out auditOutput
	mustLct(t, &out, "audit", "--wave-size", "1", "--wave", "1", "--to-file", "first")
	if out.Written == nil || out.Written.Path != "rules/requeue/2026-11-07-audit-first.yaml" {
		t.Fatalf("written = %+v", out.Written)
	}
	got, err := os.ReadFile(filepath.Join(s.root, filepath.FromSlash(out.Written.Path)))
	if err != nil {
		t.Fatal(err)
	}
	key := out.Waves[0].Keys[0]
	reason := "複查：" + strings.Join(signalsOf(out.Waves[0].Signals), "、") + "；保留原措辭，只修正問題"
	mustLct(t, nil, "requeue", "--to-file", "ref", "--date", "2026-11-07", "--key", key, "--reason", reason)
	want, err := os.ReadFile(filepath.Join(s.root, "rules", "requeue", "2026-11-07-ref.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("audit file:\n%s\nrequeue file:\n%s", got, want)
	}

	// The same topic and day again is refused, not overwritten.
	code, _, stderr := lct(t, "audit", "--wave-size", "1", "--to-file", "first")
	if code != 1 || !strings.Contains(stderr, "exists") {
		t.Errorf("second write: exit %d, stderr %q", code, stderr)
	}
}

func signalsOf(m map[string]int) []string {
	var out []string
	for _, s := range audit.Signals {
		if m[s] > 0 {
			out = append(out, s)
		}
	}
	return out
}

func TestAuditUsage(t *testing.T) {
	acceptRepo(t)
	for name, args := range map[string][]string{
		"wave size":   {"--wave-size", "0"},
		"wave zero":   {"--wave", "0"},
		"empty topic": {"--to-file", " "},
		"extra":       {"nope"},
		"negative":    {"--usd-per-unit", "-1"},
	} {
		if code, _, stderr := lct(t, append([]string{"audit"}, args...)...); code != 1 || stderr == "" {
			t.Errorf("%s: exit %d, stderr %q, want a user error", name, code, stderr)
		}
	}
}
