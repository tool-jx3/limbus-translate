// Package adjust handles the human adjustment files (spec 11.1): a person
// fixes one translation by committing an entry to rules/adjustments/*.yaml on
// main, and the plan step applies it for as long as the KR text it was made
// for is still there.
//
// An adjustment is tied to the KR text, not to a key: Resolve finds the unit
// it still belongs to, or says it has expired.
package adjust

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/rulefile"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// Dir holds the adjustment files, relative to the repo root.
const Dir = "rules/adjustments"

// OneOff is the Rule of an adjustment that has no rule behind it, such as the
// fix of a typo.
const OneOff = "one-off"

// Fields are the keys of an entry of an adjustment file, in the order they are
// written.
var Fields = []string{"key", "raw", "kr_sha", "zh", "reason", "rule"}

// IsFile reports whether a file of Dir with this name is an adjustment file.
func IsFile(name string) bool { return rulefile.IsFile(name) }

// ErrInvalid marks an adjustment (or file name) that Add refuses because of
// what it says, before anything is written. Other errors are system errors.
var ErrInvalid = errors.New("invalid adjustment")

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

// Adjustment is one entry of an adjustment file: the translation a person
// wants for one unit.
type Adjustment struct {
	// Key is the unit the adjustment was made for, "<file>|<JSON pointer>".
	Key string `yaml:"key"`
	// Raw is the raw tag of the KR version the adjustment was made from.
	Raw string `yaml:"raw"`
	// KRSha is units.KRSha of the KR text it was made for. The adjustment
	// only applies to a unit with this text.
	KRSha string `yaml:"kr_sha"`
	ZH    string `yaml:"zh"`
	// Reason says why the translation is better.
	Reason string `yaml:"reason"`
	// Rule is the file under the repo root (a guide, say) in which the
	// reasoning of the adjustment was written down, or OneOff.
	Rule string `yaml:"rule"`

	// File is the name of the file in Dir that holds the adjustment. It is
	// set by Load and is not part of the YAML.
	File string `yaml:"-"`
}

// Status says what became of an adjustment against the current KR.
type Status string

const (
	// Applied: the unit of the key still has the KR text.
	Applied Status = "applied"
	// Moved: the key is out of date (lines were inserted or removed) but the
	// same text, in the same field of the same file, is at Resolution.Key.
	Moved Status = "moved"
	// Expired: the KR text is gone. The unit is translated again; the old
	// adjustment is only a reference.
	Expired Status = "expired"
)

// Resolution is where an adjustment applies. Key is the adjustment's own key
// unless Status is Moved, in which case it is the key of the unit that now
// has the text.
type Resolution struct {
	Status Status
	Key    string
}

// shaRe is the shape of units.KRSha output: 64 lowercase hex digits.
var shaRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Resolve decides where adjustment a applies in the KR pack kr:
//
//  1. the unit of a.Key has a KR text with a.KRSha: Applied;
//  2. otherwise, a unit of the same file and the same field (the last
//     non-numeric segment of the pointer) has it: Moved to that unit. Of
//     several, the one whose first dataList index is nearest to the key's
//     wins, the earlier one when two are equally near. A string outside
//     dataList has no index: it is nearest to another such string, and
//     farther than any dataList item from one in a dataList item;
//  3. otherwise Expired, with the key unchanged.
//
// A key that is malformed, a file that KR lacks or cannot read, and a nil
// pack all give Expired: the adjustment cannot be shown to still apply.
func Resolve(a Adjustment, kr *pack.Pack) Resolution {
	expired := Resolution{Status: Expired, Key: a.Key}
	if kr == nil {
		return expired
	}
	file, pointer, err := units.SplitKey(a.Key)
	if err != nil || !kr.Has(file) {
		return expired
	}
	doc, err := kr.Doc(file)
	if err != nil {
		return expired
	}

	field, at := units.FieldName(pointer), dataListIndex(pointer)
	var nearest *units.Unit
	nearestDist := 0
	list := units.Extract(file, doc)
	for i := range list {
		u := &list[i]
		if u.Field != field || units.KRSha(u.KR) != a.KRSha {
			continue
		}
		if u.Pointer == pointer {
			return Resolution{Status: Applied, Key: a.Key}
		}
		// Document order is index order, so keeping the first of equals
		// takes the earlier one.
		if d := distance(at, u.ItemIndex); nearest == nil || d < nearestDist {
			nearest, nearestDist = u, d
		}
	}
	if nearest == nil {
		return expired
	}
	return Resolution{Status: Moved, Key: units.Key(file, nearest.Pointer)}
}

// dataListIndex returns the index in "/dataList/<n>/...", or -1 for a pointer
// that is not under an item of dataList (units.Unit.ItemIndex has the same
// meaning).
func dataListIndex(pointer string) int {
	rest, ok := strings.CutPrefix(pointer, "/dataList/")
	if !ok {
		return -1
	}
	tok, _, _ := strings.Cut(rest, "/")
	n, err := strconv.Atoi(tok)
	if err != nil || n < 0 {
		return -1
	}
	return n
}

// distance is how far apart two dataList indexes are (-1: no index).
func distance(a, b int) int {
	switch {
	case a < 0 && b < 0:
		return 0
	case a < 0 || b < 0:
		return math.MaxInt
	case a < b:
		return b - a
	}
	return a - b
}

// Load reads the adjustment files of root: Dir/*.yaml in order of file name,
// the entries of a file in the order they are written. A root without Dir has
// none. Load does not check the entries (see Problems and rules lint), but a
// file that is not a list of entries is an error that names the file.
func Load(root string) ([]Adjustment, error) {
	dir := filepath.Join(root, filepath.FromSlash(Dir))
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var all []Adjustment
	for _, e := range entries { // sorted by name
		if e.IsDir() || !IsFile(e.Name()) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var items []Adjustment
		if err := yaml.Unmarshal(data, &items); err != nil {
			return nil, fmt.Errorf("%s/%s: %w", Dir, e.Name(), err)
		}
		for i := range items {
			items[i].File = e.Name()
		}
		all = append(all, items...)
	}
	return all, nil
}

// Problem is one thing wrong with the value of one field of an adjustment.
type Problem struct {
	Field   string // one of Fields
	Message string
}

// Problems lists what is wrong with the values of a, in the order of Fields:
// a key that is no unit key, an empty raw tag, a kr_sha that is not 64
// lowercase hex digits, an empty zh or reason, a rule that is neither OneOff
// nor the path of an existing file under root. Both Add and rules lint use it.
func (a Adjustment) Problems(root string) []Problem {
	var ps []Problem
	add := func(field, format string, args ...any) {
		ps = append(ps, Problem{Field: field, Message: fmt.Sprintf(format, args...)})
	}
	if _, _, err := units.SplitKey(a.Key); err != nil {
		add("key", "key %q is not a unit key (<file>|<JSON pointer>)", a.Key)
	}
	if strings.TrimSpace(a.Raw) == "" {
		add("raw", "raw is empty; it names the raw tag the adjustment was made from")
	}
	if !shaRe.MatchString(a.KRSha) {
		add("kr_sha", "kr_sha %q is not a sha256 in 64 lowercase hex digits", a.KRSha)
	}
	if strings.TrimSpace(a.ZH) == "" {
		add("zh", "zh is empty")
	}
	if strings.TrimSpace(a.Reason) == "" {
		add("reason", "reason is empty")
	}
	if msg := ruleProblem(root, a.Rule); msg != "" {
		add("rule", "%s", msg)
	}
	return ps
}

// ruleProblem says why rule is no valid Rule, or returns "".
func ruleProblem(root, rule string) string {
	if rule == OneOff {
		return ""
	}
	if !fs.ValidPath(rule) || rule == "." {
		return fmt.Sprintf("rule %q must be %q or the path of a file in the repo, relative to its root (no ./, .. or leading /)", rule, OneOff)
	}
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rule)))
	if err != nil || info.IsDir() {
		return fmt.Sprintf("rule file %s does not exist; use %q for an adjustment that no rule explains", rule, OneOff)
	}
	return ""
}

// ValidateName checks the two parts of the name Add gives its file,
// Dir/<date>-<topic>.yaml: the date is YYYY-MM-DD, and the topic is made of
// letters (Chinese too), digits, "-" and "_" and does not start with "-".
// The error wraps ErrInvalid.
func ValidateName(date, topic string) error {
	if err := rulefile.ValidateName(date, topic); err != nil {
		return invalid("%v", err)
	}
	return nil
}

// Add writes a to the file Dir/<date>-<topic>.yaml under root, creating it, or
// appending to it when it exists, and returns the path of that file relative
// to root, with forward slashes. date is YYYY-MM-DD. What people wrote in an
// existing file stays as it is.
//
// Add refuses (an error that wraps ErrInvalid) an adjustment with a Problem, a
// date or topic that is not usable in a file name, and a key that any file of
// Dir already holds: the error names that file, since lint wants a key once
// across all adjustment files. It writes nothing then. The text of a.ZH is
// written as it is; the caller normalizes and checks it.
func Add(root string, a Adjustment, date, topic string) (string, error) {
	if err := ValidateName(date, topic); err != nil {
		return "", err
	}
	if ps := a.Problems(root); len(ps) > 0 {
		msgs := make([]string, len(ps))
		for i, p := range ps {
			msgs[i] = p.Message
		}
		return "", invalid("%s", strings.Join(msgs, "; "))
	}

	existing, err := Load(root)
	if err != nil {
		return "", err
	}
	for _, e := range existing {
		if e.Key == a.Key {
			return "", invalid("%s already holds an adjustment for %s", Dir+"/"+e.File, a.Key)
		}
	}

	rel := Dir + "/" + date + "-" + topic + rulefile.Ext
	path := filepath.Join(root, filepath.FromSlash(rel))
	content, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	entry, err := encode(a)
	if err != nil {
		return "", err
	}
	if len(content) > 0 && !bytes.HasSuffix(content, []byte("\n")) {
		content = append(content, '\n')
	}
	appended := append(slices.Clone(content), entry...)
	if err := checkAppended(content, appended, a); err != nil {
		return "", invalid("cannot add to %s: %v (write it as a block list of \"- key: ...\" entries, or use another topic)", rel, err)
	}
	if err := rulefile.WriteAtomic(path, appended); err != nil {
		return "", err
	}
	return rel, nil
}

// checkAppended reads the whole file again as Load would, before it is
// written: after appending a to the entries of before it must have exactly
// one more, and a last one that is a. A hand-written file can swallow what is
// appended (it ends its document with "...", or starts the next one) or turn
// unreadable (a flow sequence "[...]" ends where the new item begins), and a
// file that does not load stops the plan.
func checkAppended(before, after []byte, a Adjustment) error {
	var old, now []Adjustment
	if err := yaml.Unmarshal(before, &old); err != nil {
		return err
	}
	if err := yaml.Unmarshal(after, &now); err != nil {
		return fmt.Errorf("the file would not be readable: %w", err)
	}
	a.File = ""
	if len(now) != len(old)+1 || now[len(now)-1] != a {
		return fmt.Errorf("the file would hold %d entries instead of %d, with the new one last", len(now), len(old)+1)
	}
	return nil
}

// encode writes a as one YAML list item, in the layout of the spec. The
// emitter of yaml.v3 gets some strings wrong (a text that starts with a blank
// or a newline comes out as a block scalar that cannot be read back), and a
// file that does not load would stop the plan, so the result is read back and
// compared. When it differs, every value is written double-quoted, which the
// emitter does right.
func encode(a Adjustment) ([]byte, error) {
	a.File = ""
	for _, style := range []yaml.Style{0, yaml.DoubleQuotedStyle} {
		values := map[string]string{"key": a.Key, "raw": a.Raw, "kr_sha": a.KRSha, "zh": a.ZH, "reason": a.Reason, "rule": a.Rule}
		entry := &yaml.Node{Kind: yaml.MappingNode}
		for _, f := range Fields {
			entry.Content = append(entry.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: f},
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: values[f], Style: style})
		}
		data, err := yaml.Marshal(&yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{entry}})
		if err != nil {
			return nil, err
		}
		var back []Adjustment
		if yaml.Unmarshal(data, &back) == nil && len(back) == 1 && back[0] == a {
			return data, nil
		}
	}
	return nil, fmt.Errorf("adjust: the adjustment of %s cannot be written as YAML", a.Key)
}
