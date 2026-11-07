// Package gold holds the gold sample: units, drawn at a fixed seed from each
// family, that a person judges by hand, and the accuracy the judgements give.
//
// The sample and the judgements live in rules/gold.yaml, which only lct gold
// changes.
package gold

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"gopkg.in/yaml.v3"

	"github.com/tool-jx3/limbus-translate/internal/rulefile"
)

// File is the path of the gold sample, relative to the repo root.
const File = "rules/gold.yaml"

// Defaults of lct gold sample.
const (
	DefaultSeed      = 20261107
	DefaultPerFamily = 20
)

// Verdicts. A unit that no one judged has VerdictNone.
const (
	VerdictNone    = ""
	VerdictCorrect = "correct"
	VerdictWrong   = "wrong"
	VerdictUnsure  = "unsure"
)

// ValidVerdict reports whether v is one a person can set.
func ValidVerdict(v string) bool {
	return v == VerdictCorrect || v == VerdictWrong || v == VerdictUnsure
}

const header = `# 基準樣本
#
# 每個族群固定種子抽出的單位，由人逐一判定譯文對不對，用來量譯文品質。
# 只由 ` + "`lct gold`" + ` 修改，不要手改：
#   lct gold sample   抽樣（檔案已有判定時拒絕，--force 才覆蓋）
#   lct gold set      記判定
#   lct gold report   依族群與來源列正確率
#
# verdict 為空（未判定）、correct、wrong 或 unsure；judged_zh 是判定當時的譯文，
# 現有譯文與它不同的單位在報告列為 stale。
`

// Unit is one unit of the sample.
type Unit struct {
	Key      string `yaml:"key" json:"key"`
	Family   string `yaml:"family" json:"family"`
	Origin   string `yaml:"origin" json:"origin"`
	Verdict  string `yaml:"verdict" json:"verdict"`
	Note     string `yaml:"note" json:"note"`
	JudgedZH string `yaml:"judged_zh" json:"judged_zh"`
	JudgedAt string `yaml:"judged_at" json:"judged_at"`
}

// Sample is the content of rules/gold.yaml.
type Sample struct {
	Seed      int    `yaml:"seed" json:"seed"`
	PerFamily int    `yaml:"per_family" json:"per_family"`
	Units     []Unit `yaml:"units" json:"units"`
}

// Load reads the sample of root; a root without the file has an empty one.
// A file that is malformed, or holds a verdict or a duplicate key that is not
// valid, is an error.
func Load(root string) (Sample, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(File)))
	if errors.Is(err, fs.ErrNotExist) {
		return Sample{}, nil
	}
	if err != nil {
		return Sample{}, err
	}
	var s Sample
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&s); err != nil && !errors.Is(err, io.EOF) {
		return Sample{}, fmt.Errorf("%s: %w", File, err)
	}
	seen := map[string]bool{}
	for _, u := range s.Units {
		if u.Verdict != VerdictNone && !ValidVerdict(u.Verdict) {
			return Sample{}, fmt.Errorf("%s: %s: verdict %q is not correct, wrong or unsure", File, u.Key, u.Verdict)
		}
		if seen[u.Key] {
			return Sample{}, fmt.Errorf("%s: %s is listed twice", File, u.Key)
		}
		seen[u.Key] = true
	}
	return s, nil
}

// Save writes the sample to root atomically, in the one stable format.
func Save(root string, s Sample) error {
	if s.Units == nil {
		s.Units = []Unit{}
	}
	var buf bytes.Buffer
	buf.WriteString(header)
	buf.WriteString("\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(s); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	return rulefile.WriteAtomic(filepath.Join(root, filepath.FromSlash(File)), buf.Bytes())
}

// Judged returns the number of units that have a verdict.
func (s Sample) Judged() int {
	n := 0
	for _, u := range s.Units {
		if u.Verdict != VerdictNone {
			n++
		}
	}
	return n
}

// Has reports whether key is in the sample.
func (s Sample) Has(key string) bool {
	for _, u := range s.Units {
		if u.Key == key {
			return true
		}
	}
	return false
}

// Candidate is a unit the sample can draw: not a copy, with a record.
type Candidate struct {
	Key, Family, Origin string
}

// Draw picks per units of each family from cands, those with the smallest
// sha256(seed|key); a family with fewer takes all. The units come in the order
// of the families, then of the hashes, and have no verdict. A key with
// whitespace is never drawn: the gold line `gold <key> <verdict> <note>` of the
// site and the skill is split on whitespace, so such a unit could not be judged.
func Draw(cands []Candidate, seed, per int) []Unit {
	byFamily := map[string][]Candidate{}
	for _, c := range cands {
		if strings.IndexFunc(c.Key, unicode.IsSpace) >= 0 {
			continue
		}
		byFamily[c.Family] = append(byFamily[c.Family], c)
	}
	families := make([]string, 0, len(byFamily))
	for f := range byFamily {
		families = append(families, f)
	}
	sort.Strings(families)
	out := []Unit{}
	for _, f := range families {
		list := byFamily[f]
		hash := make(map[string]string, len(list))
		for _, c := range list {
			sum := sha256.Sum256([]byte(strconv.Itoa(seed) + "|" + c.Key))
			hash[c.Key] = hex.EncodeToString(sum[:])
		}
		sort.Slice(list, func(i, j int) bool {
			if hash[list[i].Key] != hash[list[j].Key] {
				return hash[list[i].Key] < hash[list[j].Key]
			}
			return list[i].Key < list[j].Key
		})
		if len(list) > per {
			list = list[:per]
		}
		for _, c := range list {
			out = append(out, Unit{Key: c.Key, Family: f, Origin: c.Origin})
		}
	}
	return out
}

// Set records the verdict of key: the note, the translation judged (zh) and the
// day (UTC). A key that is not in the sample, or a verdict that is none of
// correct, wrong and unsure, is an error.
func (s *Sample) Set(key, verdict, note, zh string, at time.Time) error {
	if !ValidVerdict(verdict) {
		return fmt.Errorf("verdict %q is not correct, wrong or unsure", verdict)
	}
	for i := range s.Units {
		if s.Units[i].Key == key {
			s.Units[i].Verdict = verdict
			s.Units[i].Note = note
			s.Units[i].JudgedZH = zh
			s.Units[i].JudgedAt = at.UTC().Format("2006-01-02")
			return nil
		}
	}
	return fmt.Errorf("%s is not in the gold sample", key)
}

// Row counts the units of one family and origin (or of a family, or all).
type Row struct {
	Family  string `json:"family"`
	Origin  string `json:"origin"`
	Sampled int    `json:"sampled"`
	Judged  int    `json:"judged"`
	Correct int    `json:"correct"`
	Wrong   int    `json:"wrong"`
	Unsure  int    `json:"unsure"`
	// Accuracy is correct/(correct+wrong); nil when neither is there.
	Accuracy *float64 `json:"accuracy"`
	// Stale counts the judged units whose current translation is not the
	// one that was judged.
	Stale int `json:"stale"`
}

// Report is the accuracy of the sample: Rows by family and origin, Families by
// family alone (Origin ""), and Total (both "").
type Report struct {
	Rows     []Row `json:"rows"`
	Families []Row `json:"families"`
	Total    Row   `json:"total"`
	// Missing is the number of units of the sample that are not in the KR
	// pack any more; the caller sets it.
	Missing int `json:"missing"`
}

func (r *Row) add(u Unit, stale bool) {
	r.Sampled++
	switch u.Verdict {
	case VerdictCorrect:
		r.Correct++
	case VerdictWrong:
		r.Wrong++
	case VerdictUnsure:
		r.Unsure++
	}
	if u.Verdict != VerdictNone {
		r.Judged++
		if stale {
			r.Stale++
		}
	}
}

func (r *Row) finish() {
	if n := r.Correct + r.Wrong; n > 0 {
		a := float64(r.Correct) / float64(n)
		r.Accuracy = &a
	}
}

// Stale reports whether u was judged on another translation than current.
func Stale(u Unit, current string) bool {
	return u.Verdict != VerdictNone && u.JudgedZH != current
}

// MakeReport counts the sample. current maps a key to its translation now; a
// key it lacks has none.
func MakeReport(s Sample, current map[string]string) Report {
	type pair struct{ f, o string }
	rows := map[pair]*Row{}
	fams := map[string]*Row{}
	total := Row{}
	for _, u := range s.Units {
		stale := Stale(u, current[u.Key])
		p := pair{u.Family, u.Origin}
		if rows[p] == nil {
			rows[p] = &Row{Family: u.Family, Origin: u.Origin}
		}
		if fams[u.Family] == nil {
			fams[u.Family] = &Row{Family: u.Family}
		}
		rows[p].add(u, stale)
		fams[u.Family].add(u, stale)
		total.add(u, stale)
	}
	rep := Report{Rows: []Row{}, Families: []Row{}}
	for _, r := range rows {
		r.finish()
		rep.Rows = append(rep.Rows, *r)
	}
	sort.Slice(rep.Rows, func(i, j int) bool {
		a, b := rep.Rows[i], rep.Rows[j]
		if a.Family != b.Family {
			return a.Family < b.Family
		}
		return a.Origin < b.Origin
	})
	for _, r := range fams {
		r.finish()
		rep.Families = append(rep.Families, *r)
	}
	sort.Slice(rep.Families, func(i, j int) bool { return rep.Families[i].Family < rep.Families[j].Family })
	total.finish()
	rep.Total = total
	return rep
}

// String renders the report as a table.
func (r Report) String() string {
	var b strings.Builder
	line := func(label string, x Row) {
		acc := "-"
		if x.Accuracy != nil {
			acc = fmt.Sprintf("%.1f%%", *x.Accuracy*100)
		}
		fmt.Fprintf(&b, "%-28s sampled %3d  judged %3d  correct %3d  wrong %3d  unsure %3d  accuracy %6s  stale %d\n",
			label, x.Sampled, x.Judged, x.Correct, x.Wrong, x.Unsure, acc, x.Stale)
	}
	for _, x := range r.Rows {
		line(x.Family+"/"+x.Origin, x)
	}
	for _, x := range r.Families {
		line(x.Family+" (all)", x)
	}
	line("total", r.Total)
	if r.Missing > 0 {
		fmt.Fprintf(&b, "missing from the pack: %d\n", r.Missing)
	}
	return b.String()
}
