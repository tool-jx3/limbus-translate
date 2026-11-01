package bootstrap

import (
	"os"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
	"github.com/tool-jx3/limbus-translate/internal/validate"
)

const testTag = "raw-20261029"

// fixtureRoot holds kr/, en/, jp/ (a raw checkout) and the upstream packs
// paratranz/, ltm/ and llc/.
const fixtureRoot = "../../testdata/bootstrap"

func TestValidParatranz(t *testing.T) {
	tests := []struct {
		name       string
		kr, en, zh string
		want       bool
	}{
		{"EN text stands in for the translation", "가", "A", "A", false},
		{"translated", "가", "A", "甲", true},
		{"still Korean", "가", "A", "가", false},
		{"Hangul left in the translation", "가", "A", "甲가", false},
		{"digits only", "가", "A", "123", false},
		{"symbols only", "가", "A", "！？", false},
		{"empty", "가", "A", "", false},
		{"copy of KR that has Han", "甲", "甲", "甲", false},
		{"EN equal to KR does not forbid a different translation", "가", "가", "甲", true},
		{"no EN known", "가", "", "甲", true},
		{"Han with Latin text", "가", "A", "增加 [Bleed] 強度", true},
		{"Latin only", "가", "A", "Restores HP", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidParatranz(tt.kr, tt.en, tt.zh); got != tt.want {
				t.Errorf("ValidParatranz(%q, %q, %q) = %v, want %v", tt.kr, tt.en, tt.zh, got, tt.want)
			}
		})
	}
}

func TestRestoreIDs(t *testing.T) {
	kw := map[string]string{"Laceration": "流血", "Burn": "燒傷"}
	tests := []struct {
		name, kr, zh, want string
		wantNormalized     string            // after Normalize; "" when the raw result is already final
		table              map[string]string // nil means kw
	}{
		{
			name: "official name and space become the ID",
			kr:   "[Laceration] 위력 5 부여", zh: "增加5級流血 強度",
			want: "增加5級[Laceration]強度", wantNormalized: "增加5級 [Laceration] 強度",
		},
		{
			name: "ID already there",
			kr:   "[Laceration] 위력 5 부여", zh: "增加5級 [Laceration] 強度",
			want: "增加5級 [Laceration] 強度",
		},
		{
			name: "only the first occurrence is restored",
			kr:   "[Laceration] 부여", zh: "流血 與流血 提升",
			want: "[Laceration]與流血 提升",
		},
		{
			name: "a repeated ID is restored as many times as it is missing",
			kr:   "[Burn] 또는 [Burn]", zh: "燒傷 或燒傷 ",
			want: "[Burn]或[Burn]",
		},
		{
			name: "one of two present, one missing",
			kr:   "[Burn] 또는 [Burn]", zh: "[Burn] 或燒傷 ",
			want: "[Burn] 或[Burn]",
		},
		{
			name: "name without the half-width space is left alone",
			kr:   "[Laceration] 부여", zh: "賦予流血",
			want: "賦予流血",
		},
		{
			name: "ID unknown to the table is left alone",
			kr:   "[Mystery] 부여", zh: "賦予謎團 強度",
			want: "賦予謎團 強度",
		},
		{
			name: "KR without IDs",
			kr:   "부여", zh: "賦予流血 強度",
			want: "賦予流血 強度",
		},
		{
			name: "two different IDs",
			kr:   "[Laceration] [Burn]", zh: "流血 燒傷 ",
			want: "[Laceration][Burn]", wantNormalized: "[Laceration] [Burn]",
		},
		{
			name: "empty name in the table",
			kr:   "[Laceration] 부여", zh: "賦予 強度",
			want:  "賦予 強度",
			table: map[string]string{"Laceration": ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := tt.table
			if table == nil {
				table = kw
			}
			got := RestoreIDs(tt.kr, tt.zh, table)
			if got != tt.want {
				t.Errorf("RestoreIDs(%q, %q) = %q, want %q", tt.kr, tt.zh, got, tt.want)
			}
			if tt.wantNormalized != "" {
				if n := validate.Normalize(tt.kr, got); n != tt.wantNormalized {
					t.Errorf("Normalize(RestoreIDs(...)) = %q, want %q", n, tt.wantNormalized)
				}
			}
		})
	}

	// The brief's example, spelled out.
	got := validate.Normalize("[Laceration] 위력 5 부여", RestoreIDs("[Laceration] 위력 5 부여", "增加5級流血 強度", kw))
	if want := "增加5級 [Laceration] 強度"; got != want {
		t.Errorf("restored and normalized = %q, want %q", got, want)
	}
}

func parseDoc(t *testing.T, s string) *jsonx.Node {
	t.Helper()
	n, err := jsonx.Parse([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestForeignNames(t *testing.T) {
	ours := parseDoc(t, `{"dataList":[
		{"id":"뫼르소","name":"莫梭"},
		{"id":"이상","name":"李箱"},
		{"id":"파우스트","name":"浮士德"},
		{"id":"홍루","name":"鴻璐"}]}`)
	ltm := parseDoc(t, `{"dataList":[
		{"id":"뫼르소","name":"默爾索"},
		{"id":"이상","name":"李箱"},
		{"id":"파우스트","name":"浮士德"},
		{"id":"홍루","name":"默爾索"},
		{"id":"그레고르","name":"格里高爾"}]}`)

	got := ForeignNames(ours, ltm)
	if want := []string{"默爾索"}; !slices.Equal(got, want) {
		t.Errorf("ForeignNames = %q, want %q (equal names, ids only LTM has, and duplicates are not listed)", got, want)
	}

	single := parseDoc(t, `{"dataList":[{"id":"뫼르소","name":"莫梭"}]}`)
	singleLTM := parseDoc(t, `{"dataList":[{"id":"뫼르소","name":"默爾索"}]}`)
	if got := ForeignNames(single, singleLTM); !slices.Equal(got, []string{"默爾索"}) {
		t.Errorf("ForeignNames of the brief's example = %q", got)
	}
	if got := ForeignNames(single, single); len(got) != 0 {
		t.Errorf("ForeignNames of identical tables = %q, want none", got)
	}
	if got := ForeignNames(nil, singleLTM); len(got) != 0 {
		t.Errorf("ForeignNames without our table = %q, want none", got)
	}
	if got := ForeignNames(single, nil); len(got) != 0 {
		t.Errorf("ForeignNames without LTM's table = %q, want none", got)
	}

	// A name ours uses for any id is not foreign: LTM's 乙 for id a differs
	// from our 甲, but 乙 is what we call id b.
	shared := ForeignNames(
		parseDoc(t, `{"dataList":[{"id":"a","name":"甲"},{"id":"b","name":"乙"},{"id":"c","name":"丙"}]}`),
		parseDoc(t, `{"dataList":[{"id":"a","name":"乙"},{"id":"b","name":"乙"},{"id":"c","name":"丁"}]}`))
	if want := []string{"丁"}; !slices.Equal(shared, want) {
		t.Errorf("ForeignNames with a name ours also uses = %q, want %q", shared, want)
	}

	// sorted, so the file written from it is stable
	many := ForeignNames(
		parseDoc(t, `{"dataList":[{"id":"a","name":"甲"},{"id":"b","name":"乙"},{"id":"c","name":"丙"}]}`),
		parseDoc(t, `{"dataList":[{"id":"a","name":"丁"},{"id":"b","name":"戊"},{"id":"c","name":"己"}]}`))
	if !slices.IsSorted(many) || len(many) != 3 {
		t.Errorf("ForeignNames = %q, want 3 sorted names", many)
	}
}

// testFieldRules loads field rules the way the CLI does, from a root with a
// manual rules file.
func testFieldRules(t *testing.T) *fields.Rules {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "rules", "fields.manual.yaml"),
		"identifier_fields: [id, model]\nsource_comment_prefixes: [\"//\"]\nnotinclude_substring: notinclude\n")
	fr, err := fields.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return fr
}

func writeFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fixtureInputs opens testdata/bootstrap the way the CLI opens the real data.
func fixtureInputs(t *testing.T) Inputs {
	t.Helper()
	raw, err := pack.OpenRaw(fixtureRoot)
	if err != nil {
		t.Fatal(err)
	}
	in := Inputs{Raw: raw, RawTag: testTag}
	for _, p := range []struct {
		dir string
		dst **pack.Pack
	}{{"paratranz", &in.Paratranz}, {"ltm", &in.LTM}, {"llc", &in.LLC}} {
		if *p.dst, err = pack.Open(filepath.Join(fixtureRoot, p.dir), ""); err != nil {
			t.Fatal(err)
		}
	}
	return in
}

// Every record says which rules it was made under.
func TestRunRecordsTheRulesCommit(t *testing.T) {
	in := fixtureInputs(t)
	in.RulesCommit = "0123abc+dirty"
	res, err := Run(in, testFieldRules(t), &rules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Records) == 0 {
		t.Fatal("no records")
	}
	for key, rec := range res.Records {
		if rec.Rules != in.RulesCommit {
			t.Errorf("record %s: Rules = %q, want %q", key, rec.Rules, in.RulesCommit)
		}
	}
}

func TestBootstrapFixture(t *testing.T) {
	res, err := Run(fixtureInputs(t), testFieldRules(t), &rules.Rules{})
	if err != nil {
		t.Fatal(err)
	}

	wantStats := map[string]int{
		"copy":                2, // Sample.json model, Bufs.json id
		"paratranz":           2, // Sample.json item 2, Bufs.json name
		"paratranz-repair":    1, // Sample.json item 3 lost its <color> tags
		"new":                 1, // Sample.json item 4 holds the EN text
		"unaligned-paratranz": 0,
		"frozen":              0,
	}
	if !reflect.DeepEqual(res.Stats, wantStats) {
		t.Errorf("Stats = %v, want %v", res.Stats, wantStats)
	}
	if len(res.Records) != 4 || len(res.Pending) != 2 {
		t.Fatalf("got %d records and %d pending items, want 4 and 2", len(res.Records), len(res.Pending))
	}
	for key, rec := range res.Records {
		if rec.Since != "raw-20261029" {
			t.Errorf("record %s: Since = %q, want raw-20261029", key, rec.Since)
		}
		if rec.Key != key {
			t.Errorf("record under key %s has Key %s", key, rec.Key)
		}
	}

	copyKey := units.Key("Sample.json", "/dataList/0/model")
	if got, want := res.Records[copyKey], (store.Record{
		Key: copyKey, KRSha: units.KRSha("Faust"), ZH: "Faust", Origin: store.OriginCopy, Since: testTag,
	}); !reflect.DeepEqual(got, want) {
		t.Errorf("copy record = %+v, want %+v", got, want)
	}

	// The valid Paratranz line had its keyword name restored from Bufs.json
	// and its spaces normalized.
	adoptedKey := units.Key("Sample.json", "/dataList/1/content")
	krAdopted := "[Laceration] 위력 5 부여"
	if got, want := res.Records[adoptedKey], (store.Record{
		Key: adoptedKey, KRSha: units.KRSha(krAdopted), ZH: "增加5級 [Laceration] 強度", Origin: store.OriginParatranz, Since: testTag,
	}); !reflect.DeepEqual(got, want) {
		t.Errorf("paratranz record = %+v, want %+v", got, want)
	}
	nameKey := units.Key("Bufs.json", "/dataList/0/name")
	if got := res.Records[nameKey]; got.ZH != "流血" || got.Origin != store.OriginParatranz {
		t.Errorf("Bufs name record = %+v, want 流血 from paratranz", got)
	}

	pending := map[string]store.PendingItem{}
	for _, it := range res.Pending {
		pending[it.Key] = it
	}
	repairKey := units.Key("Sample.json", "/dataList/2/content")
	repair, ok := pending[repairKey]
	if !ok {
		t.Fatalf("no pending item for %s", repairKey)
	}
	if repair.Reason != "paratranz-repair" || repair.Draft != "攻擊時傷害增加" {
		t.Errorf("repair item = %+v, want reason paratranz-repair and the Paratranz text as the draft", repair)
	}
	wantRefs := map[string]string{"llc": "<color=#f8c200>攻击</color>时，伤害增加", "ltm": "<color=#f8c200>攻擊</color>時，傷害增加"}
	if !reflect.DeepEqual(repair.Refs, wantRefs) {
		t.Errorf("repair item Refs = %v, want %v", repair.Refs, wantRefs)
	}
	newKey := units.Key("Sample.json", "/dataList/3/content")
	fresh, ok := pending[newKey]
	if !ok {
		t.Fatalf("no pending item for %s", newKey)
	}
	if fresh.Reason != "new" || fresh.Draft != "" {
		t.Errorf("new item = %+v, want reason new and no draft", fresh)
	}
	if want := map[string]string{"llc": "恢复体力", "ltm": "恢復體力"}; !reflect.DeepEqual(fresh.Refs, want) {
		t.Errorf("new item Refs = %v, want %v", fresh.Refs, want)
	}
	// pending units have no record
	for _, key := range []string{repairKey, newKey} {
		if _, ok := res.Records[key]; ok {
			t.Errorf("%s is pending and must not have a record", key)
		}
	}
	// every unit is accounted for exactly once
	total := 0
	for _, f := range []string{"Bufs.json", "Sample.json"} {
		doc, err := fixtureInputs(t).Raw.KR.Doc(f)
		if err != nil {
			t.Fatal(err)
		}
		total += len(units.Extract(f, doc))
	}
	if total != len(res.Records)+len(res.Pending) {
		t.Errorf("%d units, %d records + %d pending", total, len(res.Records), len(res.Pending))
	}
}

// packFiles maps a pack file name ("BgmLyrics/Lyrics.json") to its content.
type packFiles map[string]string

// writePack writes files under root, adding prefix to each file's own name.
func writePack(t *testing.T, root, prefix string, files packFiles) {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil { // a pack may have no files
		t.Fatal(err)
	}
	for name, content := range files {
		dir, base := path.Split(name)
		writeFile(t, filepath.Join(root, filepath.FromSlash(dir), prefix+base), content)
	}
}

// sources are the six packs of an Inputs, as file maps. A nil map is a pack
// with no files.
type sources struct{ kr, en, jp, pz, ltm, llc packFiles }

func (s sources) inputs(t *testing.T) Inputs {
	t.Helper()
	raw := t.TempDir()
	writePack(t, filepath.Join(raw, "kr"), "KR_", s.kr)
	writePack(t, filepath.Join(raw, "en"), "EN_", s.en)
	writePack(t, filepath.Join(raw, "jp"), "JP_", s.jp)
	set, err := pack.OpenRaw(raw)
	if err != nil {
		t.Fatal(err)
	}
	in := Inputs{Raw: set, RawTag: testTag}
	for _, p := range []struct {
		files packFiles
		dst   **pack.Pack
	}{{s.pz, &in.Paratranz}, {s.ltm, &in.LTM}, {s.llc, &in.LLC}} {
		dir := t.TempDir()
		writePack(t, dir, "", p.files)
		if *p.dst, err = pack.Open(dir, ""); err != nil {
			t.Fatal(err)
		}
	}
	return in
}

func pendingByKey(res *Result) map[string]store.PendingItem {
	m := map[string]store.PendingItem{}
	for _, it := range res.Pending {
		m[it.Key] = it
	}
	return m
}

func TestRunFrozenFilesAdoptParatranzVerbatim(t *testing.T) {
	const krLyrics = `{"dataList":[
		{"id":1,"content":"[Laceration]가사 하나"},
		{"id":2,"content":"<i>가사</i> 둘"},
		{"id":3,"content":"가사 셋"},
		{"id":4,"content":"가사 넷"},
		{"id":5,"content":"가사 다섯"}]}`
	// Item 5 is missing from Paratranz.
	const pzLyrics = `{"dataList":[
		{"id":1,"content":"[Laceration]La la la"},
		{"id":2,"content":"no tags here"},
		{"id":3,"content":""},
		{"id":4,"content":"가사 넷"}]}`
	in := sources{
		kr: packFiles{"BgmLyrics/Lyrics.json": krLyrics, "Songs/Lyrics.json": krLyrics},
		pz: packFiles{"BgmLyrics/Lyrics.json": pzLyrics, "Songs/Lyrics.json": pzLyrics},
	}.inputs(t)
	in.Frozen = []string{"BgmLyrics/**"}

	res, err := Run(in, testFieldRules(t), &rules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	key := func(file string, i int) string {
		return units.Key(file, "/dataList/"+string(rune('0'+i))+"/content")
	}

	// Frozen, non-empty, no Hangul: adopted as it is, even without Han, EN
	// differences, normalization or a passing check.
	r1 := res.Records[key("BgmLyrics/Lyrics.json", 0)]
	if r1.ZH != "[Laceration]La la la" || r1.Origin != store.OriginParatranz || len(r1.Flags) != 0 || r1.Since != testTag {
		t.Errorf("frozen item 1 record = %+v, want the text verbatim from paratranz, no flags", r1)
	}
	// A hard issue does not stop the adoption; it only asks for a review.
	r2 := res.Records[key("BgmLyrics/Lyrics.json", 1)]
	if r2.ZH != "no tags here" || r2.Origin != store.OriginParatranz || !slices.Equal(r2.Flags, []string{store.FlagNeedsReview}) {
		t.Errorf("frozen item 2 record = %+v, want adopted with needs_review", r2)
	}
	// Empty, Hangul and unaligned go to the queue as usual.
	pending := pendingByKey(res)
	for _, i := range []int{2, 3, 4} {
		k := key("BgmLyrics/Lyrics.json", i)
		if it, ok := pending[k]; !ok || it.Reason != "new" {
			t.Errorf("frozen item %d: pending = %+v (found %v), want reason new", i+1, it, ok)
		}
		if _, ok := res.Records[k]; ok {
			t.Errorf("frozen item %d must not have a record", i+1)
		}
	}

	// The same content outside the frozen list follows the normal rules: a
	// translation without Han is no translation, so items 1 and 2 (adopted
	// above when frozen) are queued as new, and not as a repair either, since
	// they never reach the check.
	for _, i := range []int{0, 1} {
		k := key("Songs/Lyrics.json", i)
		if _, ok := res.Records[k]; ok {
			t.Errorf("Songs item %d: a translation without Han must not be adopted outside the frozen files", i+1)
		}
		if it, ok := pending[k]; !ok || it.Reason != "new" || it.Draft != "" {
			t.Errorf("Songs item %d = %+v (found %v), want reason new without a draft", i+1, it, ok)
		}
	}

	// Frozen adoptions count as paratranz and as frozen.
	if res.Stats["frozen"] != 2 || res.Stats["paratranz"] != 2 {
		t.Errorf("Stats = %v, want frozen 2 and paratranz 2", res.Stats)
	}
	// Item 5 of both files has no counterpart in Paratranz; the empty and
	// Hangul values of item 3 and 4 do (they are aligned, just not usable).
	if res.Stats["unaligned-paratranz"] != 2 {
		t.Errorf("unaligned-paratranz = %d, want 2", res.Stats["unaligned-paratranz"])
	}
}

func TestRunClassifiesBeforeFrozen(t *testing.T) {
	in := sources{
		kr: packFiles{"BgmLyrics/L.json": `{"dataList":[{"id":"bgm1","content":"가사"}]}`},
		pz: packFiles{"BgmLyrics/L.json": `{"dataList":[{"id":"bgm1","content":"歌詞"}]}`},
	}.inputs(t)
	in.Frozen = []string{"BgmLyrics/**"}
	res, err := Run(in, testFieldRules(t), &rules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	idKey := units.Key("BgmLyrics/L.json", "/dataList/0/id")
	if r := res.Records[idKey]; r.Origin != store.OriginCopy || r.ZH != "bgm1" {
		t.Errorf("identifier in a frozen file = %+v, want a copy", r)
	}
	if res.Stats["copy"] != 1 || res.Stats["frozen"] != 1 {
		t.Errorf("Stats = %v, want copy 1 and frozen 1", res.Stats)
	}
}

func TestRunUnalignedAndEmptyParatranz(t *testing.T) {
	krExtra := `{"dataList":[{"id":1,"model":"Faust"},{"id":2,"content":"추가 문장"}]}`
	krPartial := `{"dataList":[{"id":1,"content":"하나"},{"id":2,"content":"둘"},{"id":3,"content":"셋"},{"id":4,"content":"넷"},{"id":5,"content":"다섯"}]}`
	pzPartial := `{"dataList":[{"id":1,"content":""},{"id":2,"content":"二"},{"id":3,"content":"三"},{"id":4,"content":"四"}]}`
	in := sources{
		kr:  packFiles{"Extra.json": krExtra, "Partial.json": krPartial},
		pz:  packFiles{"Partial.json": pzPartial},
		llc: packFiles{"Extra.json": `{"dataList":[{"id":1,"model":"Faust"},{"id":2,"content":"附加句"}]}`},
	}.inputs(t)

	res, err := Run(in, testFieldRules(t), &rules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	pending := pendingByKey(res)

	// A KR file Paratranz lacks: identifiers are copied, text is queued.
	if r := res.Records[units.Key("Extra.json", "/dataList/0/model")]; r.Origin != store.OriginCopy {
		t.Errorf("identifier of a file without paratranz = %+v, want copy", r)
	}
	extra, ok := pending[units.Key("Extra.json", "/dataList/1/content")]
	if !ok || extra.Reason != "new" {
		t.Fatalf("Extra.json text = %+v (found %v), want pending new", extra, ok)
	}
	// Refs only name the sources that have the unit.
	if want := map[string]string{"llc": "附加句"}; !reflect.DeepEqual(extra.Refs, want) {
		t.Errorf("Extra.json Refs = %v, want %v (no ltm key without an LTM file)", extra.Refs, want)
	}

	// An empty Paratranz value is not a translation, but it is aligned.
	empty := pending[units.Key("Partial.json", "/dataList/0/content")]
	if empty.Reason != "new" || empty.Refs != nil {
		t.Errorf("Partial.json item 1 = %+v, want pending new without refs", empty)
	}
	// "二" is a valid translation of "둘" (the check passes: lengths are short).
	if r := res.Records[units.Key("Partial.json", "/dataList/1/content")]; r.ZH != "二" || r.Origin != store.OriginParatranz {
		t.Errorf("Partial.json item 2 = %+v, want adopted", r)
	}
	// Item 5 has no counterpart in Paratranz; Extra.json's text has no file.
	if it := pending[units.Key("Partial.json", "/dataList/4/content")]; it.Reason != "new" {
		t.Errorf("Partial.json item 5 = %+v, want pending new", it)
	}
	// unaligned: Extra.json text (1) and Partial.json item 5 (1); the empty value is aligned.
	if res.Stats["unaligned-paratranz"] != 2 {
		t.Errorf("unaligned-paratranz = %d, want 2 (Stats %v)", res.Stats["unaligned-paratranz"], res.Stats)
	}
	if res.Stats["new"] != 3 {
		t.Errorf("new = %d, want 3 (Stats %v)", res.Stats["new"], res.Stats)
	}
}

func TestRunRepairDraftIsNormalizedAndCheckedWithGlossary(t *testing.T) {
	in := sources{
		kr: packFiles{"A.json": `{"dataList":[
			{"id":1,"content":"[Bleed] 위력 <b>강조</b>"},
			{"id":2,"content":"패키지 구매"},
			{"id":3,"content":"패키지 사용"}]}`},
		pz: packFiles{"A.json": `{"dataList":[
			{"id":1,"content":"增加[Bleed]強度"},
			{"id":2,"content":"購買禮包"},
			{"id":3,"content":"使用組合包"}]}`},
	}.inputs(t)
	glossary := &rules.Rules{Glossary: []rules.Term{{
		KR: "패키지", ZH: "組合包", Status: rules.StatusApproved, Forbidden: []string{"禮包"},
	}}}

	res, err := Run(in, testFieldRules(t), glossary)
	if err != nil {
		t.Fatal(err)
	}
	pending := pendingByKey(res)

	// Missing <b> tags: a repair whose draft already has the [ID] spacing.
	if it := pending[units.Key("A.json", "/dataList/0/content")]; it.Reason != "paratranz-repair" || it.Draft != "增加 [Bleed] 強度" {
		t.Errorf("tag repair = %+v, want paratranz-repair with the normalized draft", it)
	}
	// A forbidden rendering of an approved term is a hard content issue.
	if it := pending[units.Key("A.json", "/dataList/1/content")]; it.Reason != "paratranz-repair" || it.Draft != "購買禮包" {
		t.Errorf("glossary repair = %+v, want paratranz-repair", it)
	}
	if r := res.Records[units.Key("A.json", "/dataList/2/content")]; r.ZH != "使用組合包" || r.Origin != store.OriginParatranz {
		t.Errorf("approved rendering = %+v, want adopted", r)
	}
	if res.Stats["paratranz-repair"] != 2 || res.Stats["paratranz"] != 1 {
		t.Errorf("Stats = %v", res.Stats)
	}
}

func TestRunKeywordTableComesFromParatranz(t *testing.T) {
	in := sources{
		kr: packFiles{"A.json": `{"dataList":[{"id":1,"content":"[Burn] 부여"}]}`},
		pz: packFiles{
			"A.json":    `{"dataList":[{"id":1,"content":"賦予燒傷 效果"}]}`,
			"Bufs.json": `{"dataList":[{"id":"Burn","name":"燒傷"}]}`,
		},
	}.inputs(t)
	res, err := Run(in, testFieldRules(t), &rules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	r := res.Records[units.Key("A.json", "/dataList/0/content")]
	if r.ZH != "賦予 [Burn] 效果" {
		t.Errorf("ZH = %q, want the keyword restored and spaced", r.ZH)
	}
}

func TestRunIsDeterministic(t *testing.T) {
	run := func() *Result {
		res, err := Run(fixtureInputs(t), testFieldRules(t), &rules.Rules{})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	a, b := run(), run()
	if !reflect.DeepEqual(a, b) {
		t.Errorf("two runs differ:\n%+v\n%+v", a, b)
	}
	// Pending items come in file order, then unit order.
	var keys []string
	for _, it := range a.Pending {
		keys = append(keys, it.Key)
	}
	want := []string{units.Key("Sample.json", "/dataList/2/content"), units.Key("Sample.json", "/dataList/3/content")}
	if !slices.Equal(keys, want) {
		t.Errorf("pending order = %q, want %q", keys, want)
	}
}

func TestRunForeignNamesFromPacks(t *testing.T) {
	in := sources{
		kr:  packFiles{"A.json": `{"dataList":[]}`},
		pz:  packFiles{"ScenarioModelCodes-AutoCreated.json": `{"dataList":[{"id":"뫼르소","name":"莫梭"},{"id":"이상","name":"李箱"}]}`},
		ltm: packFiles{"ScenarioModelCodes-AutoCreated.json": `{"dataList":[{"id":"뫼르소","name":"默爾索"},{"id":"이상","name":"李箱"}]}`},
	}.inputs(t)
	res, err := Run(in, testFieldRules(t), &rules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"默爾索"}; !slices.Equal(res.ForeignNames, want) {
		t.Errorf("ForeignNames = %q, want %q", res.ForeignNames, want)
	}

	// Without the file in either pack there is nothing to compare, and the
	// result is an empty list, not nil (it is written as [] ).
	in = sources{kr: packFiles{"A.json": `{"dataList":[]}`}}.inputs(t)
	res, err = Run(in, testFieldRules(t), &rules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	if res.ForeignNames == nil || len(res.ForeignNames) != 0 {
		t.Errorf("ForeignNames = %#v, want an empty non-nil list", res.ForeignNames)
	}
}

func TestRunRejectsBadInputs(t *testing.T) {
	good := sources{kr: packFiles{"A.json": `{"dataList":[]}`}}.inputs(t)
	fr, r := testFieldRules(t), &rules.Rules{}

	noTag := good
	noTag.RawTag = ""
	noPZ := good
	noPZ.Paratranz = nil
	noKR := good
	noKR.Raw.KR = nil
	tests := []struct {
		name string
		in   Inputs
		fr   *fields.Rules
		r    *rules.Rules
	}{
		{"no raw tag", noTag, fr, r},
		{"no paratranz pack", noPZ, fr, r},
		{"no KR pack", noKR, fr, r},
		{"no field rules", good, nil, r},
		{"no rules", good, fr, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Run(tt.in, tt.fr, tt.r); err == nil {
				t.Error("Run succeeded, want an error")
			}
		})
	}

	// A broken KR file is an error that names the file.
	bad := sources{kr: packFiles{"Broken.json": `{"dataList":`}}.inputs(t)
	if _, err := Run(bad, fr, r); err == nil || !strings.Contains(err.Error(), "Broken.json") {
		t.Errorf("Run with an unparsable KR file: err = %v, want an error naming Broken.json", err)
	}
}
