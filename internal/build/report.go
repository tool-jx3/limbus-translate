package build

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/translate"
)

// versionInfo is what Info/version.json records: the inputs of the build and
// what the translation of them cost.
type versionInfo struct {
	RawTag      string            `json:"raw_tag"`
	RawToken    string            `json:"raw_token"`
	RulesCommit string            `json:"rules_commit"`
	BuiltAt     string            `json:"built_at"`
	Upstream    map[string]string `json:"upstream,omitempty"`
	// Model and CostUSD are those of the translation run (see translation).
	Model   string  `json:"model,omitempty"`
	CostUSD float64 `json:"cost_usd,omitempty"`
}

func newVersionInfo(opt Options, tr translation) versionInfo {
	return versionInfo{
		RawTag:      opt.RawTag,
		RawToken:    opt.RawToken,
		RulesCommit: opt.RulesCommit,
		BuiltAt:     opt.BuiltAt.UTC().Truncate(time.Second).Format(time.RFC3339),
		Upstream:    opt.Upstream,
		Model:       tr.Model,
		CostUSD:     tr.CostUSD,
	}
}

// translation is the model and the cost of the translation run a build is of.
// The zero value says there is none.
type translation struct {
	Model   string
	CostUSD float64
}

// readTranslation reads what lct translate left in .meta/translate.json. It
// is of this build only if it was made for the same raw tag and the same
// rules commit; a run for other inputs, or none, gives the zero translation.
func readTranslation(st *store.Store, opt Options) (translation, error) {
	var res translate.Result
	found, err := st.ReadMetaJSON(translate.MetaFile, &res)
	if err != nil {
		return translation{}, err
	}
	if !found || res.RawTag != opt.RawTag || res.RulesCommit != opt.RulesCommit {
		return translation{}, nil
	}
	return translation{Model: res.Model, CostUSD: res.CostUSD}, nil
}

// infoName is the file of the store that holds Info.
const infoName = "build.json"

// ErrNotBuilt is returned by ReadInfo for a tw worktree that has no
// .meta/build.json: nothing has been built there.
var ErrNotBuilt = errors.New("not built")

// Info is .meta/build.json: the inputs of the build, the zips it made and what
// it did. Run writes it last; ReadInfo reads it. It has every field of the
// file, because the store refuses a file with fields the type does not know.
type Info struct {
	RawTag      string            `json:"raw_tag"`
	RawToken    string            `json:"raw_token"`
	RulesCommit string            `json:"rules_commit"`
	BuiltAt     string            `json:"built_at"`
	Upstream    map[string]string `json:"upstream,omitempty"`
	// Model and CostUSD are the model and the cost (in US dollars) of the
	// translation run of these inputs, from .meta/translate.json; absent when
	// that file is of other inputs or there is none.
	Model   string  `json:"model,omitempty"`
	CostUSD float64 `json:"cost_usd,omitempty"`
	// Zips is the sha256 (lowercase hex) of each zip the build wrote to
	// OutDir, by file name. publish checks the zips it releases against it.
	Zips   map[string]string `json:"zips"`
	Report Report            `json:"report"`
}

// ReadInfo reads .meta/build.json of the tw worktree at twRoot. A worktree
// without one is ErrNotBuilt; one that cannot be read or decoded is another
// error.
func ReadInfo(twRoot string) (Info, error) {
	var info Info
	found, err := store.Open(twRoot).ReadMetaJSON(infoName, &info)
	if err != nil {
		return Info{}, err
	}
	if !found {
		return Info{}, fmt.Errorf("build: %w: %s has no .meta/%s; run lct build first", ErrNotBuilt, twRoot, infoName)
	}
	return info, nil
}

// encodeJSON renders v like the store does: 2-space indent, no HTML escaping,
// one trailing newline, map keys sorted.
func encodeJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("build: %w", err)
	}
	return buf.Bytes(), nil
}

// ReportPath is where Run writes the report of the build.
func ReportPath(opt Options) string {
	return filepath.Join(opt.TWRoot, ".meta", "reports", opt.RawTag+".md")
}

// writeMeta writes the build record and the report to the store. v is the
// content of Info/version.json, zips the checksums of the zips, by file name.
func writeMeta(st *store.Store, opt Options, v versionInfo, zips map[string]string, rep *Report) error {
	info := Info{
		RawTag: v.RawTag, RawToken: v.RawToken, RulesCommit: v.RulesCommit, BuiltAt: v.BuiltAt, Upstream: v.Upstream,
		Model: v.Model, CostUSD: v.CostUSD, Zips: zips, Report: *rep,
	}
	if err := st.WriteMetaJSON(infoName, info); err != nil {
		return err
	}
	return st.WriteMetaFile("reports/"+opt.RawTag+".md", Markdown(opt, rep))
}

// Markdown renders the report of a build: its inputs and every field of rep.
func Markdown(opt Options, rep *Report) []byte {
	var b strings.Builder
	info := newVersionInfo(opt, translation{})
	fmt.Fprintf(&b, "# Build report %s\n\n", opt.RawTag)
	fmt.Fprintf(&b, "- Raw token: `%s`\n- Rules commit: `%s`\n- Built at: %s\n", info.RawToken, info.RulesCommit, info.BuiltAt)
	for _, name := range slices.Sorted(maps.Keys(info.Upstream)) {
		fmt.Fprintf(&b, "- Upstream %s: `%s`\n", name, info.Upstream[name])
	}

	b.WriteString("\n## Units\n\n| Item | Units |\n|---|---|\n")
	for _, row := range []struct {
		label string
		n     int
	}{
		{"Pending", rep.Pending},
		{"Needs review", rep.NeedsReview},
		{"Structural fallback", rep.StructuralFallback},
		{"Frozen fallback", len(rep.FrozenFallback)},
		{"Unaligned", rep.Unaligned},
		{"Field rule override", rep.FieldRuleOverride},
	} {
		fmt.Fprintf(&b, "| %s | %d |\n", row.label, row.n)
	}
	b.WriteString("\n" +
		"Pending units have no translation made from the current KR text; their EN text, or KR, is shown.\n" +
		"Needs review counts the used records that are flagged needs_review; a frozen fallback is not counted again here.\n" +
		"Structural fallbacks are translations that break the structure of KR (tags, [ID]s, placeholders) and were replaced the same way.\n" +
		"Frozen fallbacks are such translations in frozen files: they are not translated again, so each needs a person (listed below).\n" +
		"Unaligned units are pending units with no EN text either.\n" +
		"Field rule overrides are records of units the field rules do not translate; KR is shown instead.\n")

	counts := func(title, head string, m map[string]int) {
		fmt.Fprintf(&b, "\n## %s\n\n", title)
		if len(m) == 0 {
			b.WriteString("None.\n")
			return
		}
		fmt.Fprintf(&b, "| %s | Units |\n|---|---|\n", head)
		for _, k := range slices.Sorted(maps.Keys(m)) {
			fmt.Fprintf(&b, "| %s | %d |\n", k, m[k])
		}
	}
	counts("Origins", "Origin", rep.Origins)
	counts("Warnings", "Code", rep.Warnings)

	b.WriteString("\n## Frozen fallbacks\n\n")
	if len(rep.FrozenFallback) == 0 {
		b.WriteString("None.\n")
	}
	for _, key := range rep.FrozenFallback {
		fmt.Fprintf(&b, "- `%s`\n", key)
	}

	b.WriteString("\n## Missing glyphs\n\n")
	if len(rep.MissingGlyphs) == 0 {
		b.WriteString("None.\n")
	}
	for _, r := range rep.MissingGlyphs {
		fmt.Fprintf(&b, "- U+%04X", r)
		// A character that cannot be seen, or that would end the code span, is
		// only given by its number.
		if unicode.IsPrint(r) && r != '`' {
			fmt.Fprintf(&b, " `%c`", r)
		}
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

// runeSet is a set of characters: one bit for each code point.
type runeSet struct {
	bits [unicode.MaxRune/8 + 1]byte
}

func (s *runeSet) addString(text string) {
	for _, r := range text { // an invalid byte comes out as U+FFFD
		s.bits[r>>3] |= 1 << (r & 7)
	}
}

// all yields the characters of the set in ascending order.
func (s *runeSet) all() iter.Seq[rune] {
	return func(yield func(rune) bool) {
		for i, b := range s.bits {
			if b == 0 {
				continue
			}
			for bit := 0; bit < 8; bit++ {
				if b&(1<<bit) != 0 && !yield(rune(i*8+bit)) {
					return
				}
			}
		}
	}
}
