package impact

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// krPack writes the files (pack file name to JSON) as a KR pack and opens it.
func krPack(t *testing.T, files map[string]string) *pack.Pack {
	t.Helper()
	root := t.TempDir()
	for file, content := range files {
		dir, name := filepath.Split(file)
		p := filepath.Join(root, filepath.FromSlash(dir), "KR_"+name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p, err := pack.Open(root, "KR_")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// story is a KR file with one text per line: the unit of line i is
// "/dataList/<i>/content".
func story(lines ...string) string {
	var items []string
	for i, line := range lines {
		items = append(items, fmt.Sprintf(`{"id":%d,"content":%q}`, i, line))
	}
	return `{"dataList":[` + strings.Join(items, ",") + `]}`
}

func key(file string, line int) string {
	return units.Key(file, fmt.Sprintf("/dataList/%d/content", line))
}

// record is a stored translation of the text kr.
func record(key, kr, zh, origin string) store.Record {
	return store.Record{Key: key, KRSha: units.KRSha(kr), ZH: zh, Origin: origin, Since: "raw-20261029"}
}

func records(recs ...store.Record) map[string]store.Record {
	m := map[string]store.Record{}
	for _, r := range recs {
		m[r.Key] = r
	}
	return m
}

// oldRules forbids 聖盃 for 성배. newRules adds the term 사제, for which 神父
// is forbidden.
func oldRules() *rules.Rules {
	return &rules.Rules{Glossary: []rules.Term{
		{KR: "성배", ZH: "聖杯", Status: rules.StatusApproved, Forbidden: []string{"聖盃"}},
	}}
}

func newRules() *rules.Rules {
	r := oldRules()
	r.Glossary = append(r.Glossary, rules.Term{KR: "사제", ZH: "祭司", Status: rules.StatusApproved, Forbidden: []string{"神父"}})
	return r
}

func TestImpactCountsOnlyNewViolations(t *testing.T) {
	const f = "Other.json"
	kr := krPack(t, map[string]string{f: story("사제가 왔다", "사제를 만났다", "성배를 찾았다", "사제가 웃었다")})
	recs := records(
		record(key(f, 0), "사제가 왔다", "神父來了", store.OriginClaude),
		record(key(f, 1), "사제를 만났다", "遇見了神父", store.OriginParatranz),
		// The old rules forbid 聖盃 too: not a new violation.
		record(key(f, 2), "성배를 찾았다", "找到了聖盃", store.OriginClaude),
		// Follows the new rule.
		record(key(f, 3), "사제가 웃었다", "祭司笑了", store.OriginHuman),
	)

	got, err := Run(Inputs{Old: oldRules(), New: newRules(), Records: recs, KR: kr})
	if err != nil {
		t.Fatal(err)
	}
	if got.Checked != 4 {
		t.Errorf("Checked = %d, want 4", got.Checked)
	}
	// The unit of line 2 breaks a rule of the old rules too: not new.
	if got.Units != 2 {
		t.Errorf("Units = %d, want 2 (the units of lines 0 and 1)", got.Units)
	}
	if want := map[string]int{"forbidden-term": 2}; !mapsEqual(got.ByCode, want) {
		t.Errorf("ByCode = %v, want %v", got.ByCode, want)
	}
	if want := map[string]int{store.OriginClaude: 1, store.OriginParatranz: 1}; !mapsEqual(got.ByOrigin, want) {
		t.Errorf("ByOrigin = %v, want %v", got.ByOrigin, want)
	}
	if want := []string{key(f, 0), key(f, 1)}; !slices.Equal(got.Samples["forbidden-term"], want) {
		t.Errorf("Samples = %v, want the keys %v", got.Samples, want)
	}
}

// A unit that already breaks one forbidden term under the old rules and also
// breaks a second one that only the new rules forbid is a unit the new rules
// reject for a new reason: it counts although the code is the same under both.
func TestImpactCountsASecondForbiddenTerm(t *testing.T) {
	const f = "Other.json"
	kr := krPack(t, map[string]string{f: story("성배의 사제", "성배를 든 사제")})
	recs := records(
		// 聖盃 is forbidden by the old rules, 神父 only by the new rules.
		record(key(f, 0), "성배의 사제", "聖盃的神父", store.OriginClaude),
		// Only the old rule applies to it, and the new rules keep it.
		record(key(f, 1), "성배를 든 사제", "拿著聖盃的祭司", store.OriginClaude),
	)

	got, err := Run(Inputs{Old: oldRules(), New: newRules(), Records: recs, KR: kr})
	if err != nil {
		t.Fatal(err)
	}
	if got.Checked != 2 || got.Units != 1 {
		t.Errorf("Checked, Units = %d, %d, want 2, 1", got.Checked, got.Units)
	}
	if want := map[string]int{"forbidden-term": 1}; !mapsEqual(got.ByCode, want) {
		t.Errorf("ByCode = %v, want %v", got.ByCode, want)
	}
	if want := []string{key(f, 0)}; !slices.Equal(got.Samples["forbidden-term"], want) {
		t.Errorf("Samples = %v, want %v", got.Samples, want)
	}
}

func TestImpactSameRulesBreakNothing(t *testing.T) {
	const f = "Other.json"
	kr := krPack(t, map[string]string{f: story("사제가 왔다")})
	recs := records(record(key(f, 0), "사제가 왔다", "神父來了", store.OriginClaude))
	got, err := Run(Inputs{Old: newRules(), New: newRules(), Records: recs, KR: kr})
	if err != nil {
		t.Fatal(err)
	}
	if got.Checked != 1 || got.Units != 0 {
		t.Errorf("Checked, Units = %d, %d, want 1, 0", got.Checked, got.Units)
	}
	// The maps are empty and not nil, so that the JSON has {} for them.
	if got.ByCode == nil || got.ByOrigin == nil || got.Samples == nil {
		t.Errorf("a result without findings has nil maps: %+v", got)
	}
}

func TestImpactIgnoresStaleAndFrozen(t *testing.T) {
	const f = "Other.json"
	const frozen = "BgmLyrics/Credit.json"
	kr := krPack(t, map[string]string{
		f:      story("사제가 왔다", "사제가 갔다", "사제가 울었다", "사제가 웃었다", "사제가 잤다"),
		frozen: story("사제의 노래"),
	})
	bad := func(i int, kr, origin string) store.Record { return record(key(f, i), kr, "神父", origin) }

	stale := bad(1, "사제가 갔다", store.OriginClaude)
	stale.KRSha = units.KRSha("사제가 떠났다") // made from a KR text that has changed since
	recs := records(
		bad(0, "사제가 왔다", store.OriginClaude), // the one that counts
		stale,
		record(key(frozen, 0), "사제의 노래", "神父", store.OriginParatranz),
		bad(2, "사제가 울었다", store.OriginCopy),
		bad(3, "사제가 웃었다", store.OriginFallback),
		// A unit that is no longer in KR: its file, and its pointer.
		record(key("Gone.json", 0), "사제", "神父", store.OriginClaude),
		record(units.Key(f, "/dataList/99/content"), "사제", "神父", store.OriginClaude),
	)

	got, err := Run(Inputs{Old: oldRules(), New: newRules(), Records: recs, KR: kr, Frozen: []string{"BgmLyrics/**"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Checked != 1 || got.Units != 1 {
		t.Errorf("Checked, Units = %d, %d, want 1, 1: only the first record is a translation of the KR text there is now, outside the frozen files, of a kind that counts", got.Checked, got.Units)
	}
	if want := []string{key(f, 0)}; !slices.Equal(got.Samples["forbidden-term"], want) {
		t.Errorf("Samples = %v, want %v", got.Samples, want)
	}
}

func TestImpactSkipsOriginsThatAreNotTranslations(t *testing.T) {
	const f = "Other.json"
	kr := krPack(t, map[string]string{f: story("사제")})
	for _, origin := range store.Origins {
		t.Run(origin, func(t *testing.T) {
			recs := records(record(key(f, 0), "사제", "神父", origin))
			got, err := Run(Inputs{Old: oldRules(), New: newRules(), Records: recs, KR: kr})
			if err != nil {
				t.Fatal(err)
			}
			want := 0
			if slices.Contains([]string{store.OriginParatranz, store.OriginClaude, store.OriginHuman}, origin) {
				want = 1
			}
			if got.Checked != want || got.Units != want {
				t.Errorf("origin %s: Checked, Units = %d, %d, want %d, %d", origin, got.Checked, got.Units, want, want)
			}
		})
	}
}

func TestImpactSamplesAreTheFirstTenSorted(t *testing.T) {
	const f = "Other.json"
	var lines []string
	var recs []store.Record
	for i := 0; i < 13; i++ {
		lines = append(lines, "사제")
		recs = append(recs, record(key(f, i), "사제", "神父", store.OriginClaude))
	}
	kr := krPack(t, map[string]string{f: story(lines...)})

	got, err := Run(Inputs{Old: oldRules(), New: newRules(), Records: records(recs...), KR: kr})
	if err != nil {
		t.Fatal(err)
	}
	if got.Units != 13 || got.ByCode["forbidden-term"] != 13 {
		t.Errorf("Units, ByCode = %d, %v, want 13 of them", got.Units, got.ByCode)
	}
	var all []string
	for i := 0; i < 13; i++ {
		all = append(all, key(f, i))
	}
	slices.Sort(all)
	if want := all[:MaxSamples]; !slices.Equal(got.Samples["forbidden-term"], want) {
		t.Errorf("Samples = %v, want the first %d keys in sorted order %v", got.Samples["forbidden-term"], MaxSamples, want)
	}
}

// A unit counts once, whatever number of new codes it has, and each new code
// counts the unit.
func TestImpactCountsAUnitOncePerCode(t *testing.T) {
	const f = "Other.json"
	kr := krPack(t, map[string]string{f: story("사제가 왔다")})
	// 神父 is forbidden by the new rules, and the text has Hangul left: the
	// second is a content issue that no rule set changes.
	recs := records(record(key(f, 0), "사제가 왔다", "神父來了 왔다", store.OriginClaude))
	got, err := Run(Inputs{Old: oldRules(), New: newRules(), Records: recs, KR: kr})
	if err != nil {
		t.Fatal(err)
	}
	if got.Units != 1 {
		t.Errorf("Units = %d, want 1", got.Units)
	}
	if want := map[string]int{"forbidden-term": 1}; !mapsEqual(got.ByCode, want) {
		t.Errorf("ByCode = %v, want %v: the Hangul is a problem of the old rules as well", got.ByCode, want)
	}
	if want := map[string]int{store.OriginClaude: 1}; !mapsEqual(got.ByOrigin, want) {
		t.Errorf("ByOrigin = %v, want %v", got.ByOrigin, want)
	}
}

// Two new issues of one code (two newly forbidden renderings in one unit)
// count the unit once for the code, and once among the samples.
func TestImpactCountsTwoNewIssuesOfOneCodeOnce(t *testing.T) {
	const f = "Other.json"
	kr := krPack(t, map[string]string{f: story("사제와 기사")})
	recs := records(record(key(f, 0), "사제와 기사", "神父與武士", store.OriginClaude))
	two := newRules()
	two.Glossary = append(two.Glossary, rules.Term{KR: "기사", ZH: "騎士", Status: rules.StatusApproved, Forbidden: []string{"武士"}})
	got, err := Run(Inputs{Old: oldRules(), New: two, Records: recs, KR: kr})
	if err != nil {
		t.Fatal(err)
	}
	if got.Units != 1 {
		t.Errorf("Units = %d, want 1", got.Units)
	}
	if want := map[string]int{"forbidden-term": 1}; !mapsEqual(got.ByCode, want) {
		t.Errorf("ByCode = %v, want %v", got.ByCode, want)
	}
	if want := []string{key(f, 0)}; !slices.Equal(got.Samples["forbidden-term"], want) {
		t.Errorf("Samples = %v, want the key once", got.Samples)
	}
}

func TestImpactNeedsBothRulesAndKR(t *testing.T) {
	kr := krPack(t, map[string]string{"Other.json": story("사제")})
	for name, in := range map[string]Inputs{
		"no old rules": {New: newRules(), KR: kr},
		"no new rules": {Old: oldRules(), KR: kr},
		"no KR":        {Old: oldRules(), New: newRules()},
	} {
		if _, err := Run(in); err == nil {
			t.Errorf("%s: Run returned no error", name)
		}
	}
}

func TestResultMarkdown(t *testing.T) {
	res := &Result{
		Checked:  1200,
		Units:    3,
		ByCode:   map[string]int{"forbidden-term": 3},
		ByOrigin: map[string]int{store.OriginClaude: 2, store.OriginParatranz: 1},
		Samples:  map[string][]string{"forbidden-term": {"A.json|/dataList/0/content", "A.json|/dataList/1/content"}},
	}
	md := res.Markdown("origin/main")
	for _, want := range []string{"origin/main", "1200", "3", "forbidden-term", store.OriginClaude, store.OriginParatranz,
		"`A.json|/dataList/0/content`", "`A.json|/dataList/1/content`"} {
		if !strings.Contains(md, want) {
			t.Errorf("the markdown lacks %q:\n%s", want, md)
		}
	}
	if !strings.HasSuffix(md, "\n") {
		t.Errorf("the markdown does not end in a line break: %q", md[len(md)-10:])
	}

	none := (&Result{Checked: 50, ByCode: map[string]int{}, ByOrigin: map[string]int{}, Samples: map[string][]string{}}).Markdown("origin/main")
	if !strings.Contains(none, "50") || strings.Contains(none, "forbidden-term") {
		t.Errorf("the markdown of a result without findings:\n%s", none)
	}
}

func mapsEqual[V comparable](a, b map[string]V) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || w != v {
			return false
		}
	}
	return true
}
