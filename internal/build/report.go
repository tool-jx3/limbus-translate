package build

import (
	"bytes"
	"encoding/json"
	"fmt"
	"iter"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/tool-jx3/limbus-translate/internal/store"
)

// versionInfo is what Info/version.json records: the inputs of the build.
type versionInfo struct {
	RawTag      string            `json:"raw_tag"`
	RawToken    string            `json:"raw_token"`
	RulesCommit string            `json:"rules_commit"`
	BuiltAt     string            `json:"built_at"`
	Upstream    map[string]string `json:"upstream,omitempty"`
}

func newVersionInfo(opt Options) versionInfo {
	return versionInfo{
		RawTag:      opt.RawTag,
		RawToken:    opt.RawToken,
		RulesCommit: opt.RulesCommit,
		BuiltAt:     opt.BuiltAt.UTC().Truncate(time.Second).Format(time.RFC3339),
		Upstream:    opt.Upstream,
	}
}

// buildMeta is .meta/build.json: the inputs and what the build did.
type buildMeta struct {
	versionInfo
	Report *Report `json:"report"`
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

// writeMeta writes the build record and the report to the store.
func writeMeta(st *store.Store, opt Options, rep *Report) error {
	if err := st.WriteMetaJSON("build.json", buildMeta{newVersionInfo(opt), rep}); err != nil {
		return err
	}
	return st.WriteMetaFile("reports/"+opt.RawTag+".md", Markdown(opt, rep))
}

// Markdown renders the report of a build: its inputs and every field of rep.
func Markdown(opt Options, rep *Report) []byte {
	var b strings.Builder
	info := newVersionInfo(opt)
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
