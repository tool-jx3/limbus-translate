package main

import (
	"os"
	"path/filepath"
	"slices"
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
	if out.RateSource != ".meta/translate.json" || out.RateUnits == nil || *out.RateUnits != 60 {
		t.Errorf("rate basis = %q, %v units, want .meta/translate.json and 60", out.RateSource, out.RateUnits)
	}
	if _, text, _ := lct(t, "audit"); !strings.Contains(text, ".meta/translate.json") || !strings.Contains(text, "60 units") {
		t.Errorf("text output lacks the rate basis:\n%s", text)
	}
	if got := *out.Waves[0].EstimatedUSD; got != 0.05*float64(out.Waves[0].Units) {
		t.Errorf("estimate = %v", got)
	}

	write(translate.Result{CostUSD: 3, PendingBefore: 40, PendingAfter: 40})
	mustLct(t, &out, "audit")
	if out.USDPerUnit != nil || out.Waves[0].EstimatedUSD != nil {
		t.Errorf("a run that cleared nothing gave rate %v", out.USDPerUnit)
	}

	var over auditOutput
	mustLct(t, &over, "audit", "--usd-per-unit", "0.5")
	if over.USDPerUnit == nil || *over.USDPerUnit != 0.5 || *over.Waves[0].EstimatedUSD != 0.5*float64(over.Waves[0].Units) {
		t.Errorf("--usd-per-unit not used: %+v", over)
	}
	if over.RateSource != "--usd-per-unit" || over.RateUnits != nil {
		t.Errorf("override: rate_source %q, rate_units %v, want --usd-per-unit and none", over.RateSource, over.RateUnits)
	}
}

// longRecord makes the translation of one unit of the fixture far longer than
// its KR, so that it has a length-ratio signal that the other unit lacks.
func longRecord(t *testing.T, s acceptState) (long string) {
	t.Helper()
	st := store.Open(filepath.Join(s.root, ".work", "tw"))
	records, err := st.LoadRecords()
	if err != nil {
		t.Fatal(err)
	}
	long = "Skills_personality-01.json|/dataList/0/levelList/0/desc"
	rec, ok := records[long]
	if !ok {
		t.Fatalf("the fixture has no record %s", long)
	}
	rec.ZH = strings.Repeat("長", 60) + rec.ZH
	records[long] = rec
	if err := st.SaveRecords(records); err != nil {
		t.Fatal(err)
	}
	mustLct(t, nil, "build")
	return long
}

func TestAuditToFileWritesWhatRequeueWrites(t *testing.T) {
	s := acceptRepo(t)
	old := ruleFileNow
	ruleFileNow = func() time.Time { return time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { ruleFileNow = old })

	var out auditOutput
	mustLct(t, &out, "audit", "--wave-size", "1", "--wave", "1", "--to-file", "first")
	if len(out.Written) != 1 || out.Written[0].Path != "rules/requeue/2026-11-07-audit-first-format.yaml" {
		t.Fatalf("written = %+v", out.Written)
	}
	got, err := os.ReadFile(filepath.Join(s.root, filepath.FromSlash(out.Written[0].Path)))
	if err != nil {
		t.Fatal(err)
	}
	key := out.Waves[0].Keys[0]
	reason := "複查：格式警告；保留原措辭，只修正問題"
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

func TestAuditToFileWritesOneRequestForEachSetOfSignals(t *testing.T) {
	s := acceptRepo(t)
	long := longRecord(t, s)
	old := ruleFileNow
	ruleFileNow = func() time.Time { return time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { ruleFileNow = old })

	var out auditOutput
	mustLct(t, &out, "audit", "--wave-size", "10", "--to-file", "w")
	if len(out.Written) != 2 {
		t.Fatalf("written = %+v, want a request for each of the 2 sets of signals", out.Written)
	}
	want := map[string]struct {
		keys   []string
		reason string
	}{
		"rules/requeue/2026-11-07-audit-w-length-ratio-format.yaml": {[]string{long}, "複查：譯文長度與 KR 比例異常、格式警告；保留原措辭，只修正問題"},
		"rules/requeue/2026-11-07-audit-w-format.yaml":              {[]string{"Bufs.json|/dataList/0/name"}, "複查：格式警告；保留原措辭，只修正問題"},
	}
	for _, w := range out.Written {
		exp, ok := want[w.Path]
		if !ok {
			t.Errorf("unexpected file %s", w.Path)
			continue
		}
		delete(want, w.Path)
		got, err := os.ReadFile(filepath.Join(s.root, filepath.FromSlash(w.Path)))
		if err != nil {
			t.Fatal(err)
		}
		// The file is what lct requeue --to-file writes for its keys and reason.
		topic := "ref-" + strings.TrimSuffix(filepath.Base(w.Path), ".yaml")
		topic = strings.ReplaceAll(topic, "2026-11-07-", "")
		mustLct(t, nil, "requeue", "--to-file", topic, "--date", "2026-11-07", "--key", exp.keys[0], "--reason", exp.reason)
		ref, err := os.ReadFile(filepath.Join(s.root, "rules", "requeue", "2026-11-07-"+topic+".yaml"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(ref) {
			t.Errorf("%s:\n%s\nrequeue file:\n%s", w.Path, got, ref)
		}
		if strings.Count(exp.reason, "、") == 0 && strings.Contains(string(got), "長度") {
			t.Errorf("%s names a signal its units lack:\n%s", w.Path, got)
		}
	}
	for left := range want {
		t.Errorf("%s was not written", left)
	}
}

// A second set whose file exists already must not leave the first set's file
// behind: all or nothing.
func TestAuditToFileWritesNothingWhenAnyTargetExists(t *testing.T) {
	s := acceptRepo(t)
	longRecord(t, s)
	old := ruleFileNow
	ruleFileNow = func() time.Time { return time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { ruleFileNow = old })

	dir := filepath.Join(s.root, "rules", "requeue")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	taken := filepath.Join(dir, "2026-11-07-audit-w-format.yaml")
	if err := os.WriteFile(taken, []byte("reason: x\nkeys: [a]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := lct(t, "audit", "--wave-size", "10", "--to-file", "w")
	if code != 1 || !strings.Contains(stderr, "exists") {
		t.Errorf("exit %d, stderr %q, want 1 and exists", code, stderr)
	}
	left, _ := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if len(left) != 1 || left[0] != taken {
		t.Errorf("files = %v, want only the one that existed", left)
	}
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

// An entry of check: off gives the audit no signal: the review has no
// warning of a missing rendering of it. The same entry with the default
// check gives its units a term-missing signal.
func TestAuditGivesNoSignalForAnOffEntry(t *testing.T) {
	s := acceptRepo(t)
	glossary := filepath.Join(s.root, "glossary", "terms.yaml")
	data, err := os.ReadFile(glossary)
	if err != nil {
		t.Fatal(err)
	}
	termSignals := func(entry string) int {
		t.Helper()
		if err := os.WriteFile(glossary, append(slices.Clip(data), entry...), 0o644); err != nil {
			t.Fatal(err)
		}
		var out auditOutput
		mustLct(t, &out, "audit", "--wave-size", "1000")
		n := 0
		for _, w := range out.Waves {
			n += w.Signals[audit.SignalTermMissing] + w.Signals[audit.SignalTermMissingSame]
		}
		return n
	}
	base := termSignals("")
	const entry = "\n\n- kr: 열상\n  zh: 絕無此譯\n  status: approved\n"
	if warn := termSignals(entry); warn <= base {
		t.Fatalf("term signals with the warn entry = %d, without it %d: the test checks nothing", warn, base)
	}
	if off := termSignals(entry + "  check: off\n"); off != base {
		t.Errorf("term signals with the off entry = %d, want %d as without it", off, base)
	}
}
