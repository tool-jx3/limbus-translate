package rules

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/tool-jx3/limbus-translate/internal/adjust"
	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/rulefile"
)

// LintError is one problem Lint found. File is relative to the repo root.
type LintError struct {
	File    string `json:"file"`
	Message string `json:"message"`
}

func (e LintError) Error() string {
	if e.File == "" {
		return e.Message
	}
	return e.File + ": " + e.Message
}

// Lint checks the rule files under root and returns every problem it finds,
// in a fixed order (every error is a LintError). Unlike Load it does not stop
// at the first problem.
func Lint(root string) []error {
	l := &linter{root: root}
	l.glossary()
	l.baseGuide()
	l.characters()
	l.routing()
	l.foreignIgnore()
	l.adjustments()
	return l.errs
}

type linter struct {
	root string
	errs []error
}

func (l *linter) report(file, format string, args ...any) {
	l.errs = append(l.errs, LintError{File: file, Message: fmt.Sprintf(format, args...)})
}

// read returns the content of a file under root, reporting a failure.
func (l *linter) read(rel string) ([]byte, bool) {
	data, err := os.ReadFile(filepath.Join(l.root, filepath.FromSlash(rel)))
	if err != nil {
		l.report(rel, "%s", ioMessage(err))
		return nil, false
	}
	return data, true
}

// ioMessage is err without the absolute path os puts in it.
func ioMessage(err error) string {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err.Error()
	}
	return err.Error()
}

func (l *linter) fileExists(rel string) bool {
	info, err := os.Stat(filepath.Join(l.root, filepath.FromSlash(rel)))
	return err == nil && !info.IsDir()
}

func (l *linter) baseGuide() {
	if !l.fileExists(BaseGuide) {
		l.report(BaseGuide, "the base guide does not exist")
	}
}

var validStatus = []string{StatusApproved, StatusCandidate}

func (l *linter) glossary() {
	data, ok := l.read(GlossaryFile)
	if !ok {
		return
	}
	terms, unknown, err := parseGlossary(data)
	if err != nil {
		l.report(GlossaryFile, "%v", err)
		return
	}
	general := map[string]int{} // KR text -> approved entries that apply to every file
	families := l.familyIDs()
	for i, t := range terms {
		label := fmt.Sprintf("entry %d", i+1)
		if t.KR != "" {
			label = fmt.Sprintf("term %q", t.KR)
		}
		if t.KR == "" {
			l.report(GlossaryFile, "%s: kr is missing", label)
		}
		if t.ZH == "" {
			l.report(GlossaryFile, "%s: zh is missing", label)
		}
		if !slices.Contains(validStatus, t.Status) {
			l.report(GlossaryFile, "%s: status is %q, want one of %s", label, t.Status, strings.Join(validStatus, ", "))
		}
		for _, f := range t.Forbidden {
			if pattern, isRegexp := strings.CutPrefix(f, "re:"); isRegexp {
				if _, err := regexp.Compile(pattern); err != nil {
					l.report(GlossaryFile, `%s: forbidden pattern "%s" is not a valid regular expression: %v`, label, f, err)
				}
			}
		}
		if t.Match != "" && t.Match != MatchWord && t.Match != MatchSubstring {
			l.report(GlossaryFile, "%s: match is %q, want %q (hit only where the KR stands as a word; the default) or %q (hit inside a longer word too)", label, t.Match, MatchWord, MatchSubstring)
		}
		for _, pattern := range t.Files {
			if id, isFamily := strings.CutPrefix(pattern, FamilyPrefix); isFamily {
				if families != nil && !slices.Contains(families, id) {
					l.report(GlossaryFile, "%s: files names family %q, which is not a family of %s", label, id, RoutingFile)
				}
			} else if pattern == "" {
				l.report(GlossaryFile, "%s: files holds an empty pattern", label)
			} else if err := config.ValidPattern(pattern); err != nil {
				l.report(GlossaryFile, "%s: files pattern %q is not valid: %v", label, pattern, err)
			}
		}
		for _, pattern := range t.NotFiles {
			if id, isFamily := strings.CutPrefix(pattern, FamilyPrefix); isFamily {
				if families != nil && !slices.Contains(families, id) {
					l.report(GlossaryFile, "%s: not_files names family %q, which is not a family of %s", label, id, RoutingFile)
				}
			} else if pattern == "" {
				l.report(GlossaryFile, "%s: not_files holds an empty pattern", label)
			} else if err := config.ValidPattern(pattern); err != nil {
				l.report(GlossaryFile, "%s: not_files pattern %q is not valid: %v", label, pattern, err)
			}
		}
		// A pattern in both lists takes the files of files out again. Whether
		// two different globs overlap is not decided here: only the plain case.
		if len(t.Files) > 0 && len(t.NotFiles) > 0 && !slices.ContainsFunc(t.Files, func(p string) bool { return !slices.Contains(t.NotFiles, p) }) {
			l.report(GlossaryFile, "%s: every pattern of files is also in not_files, so the entry applies to no file", label)
		}
		// A candidate binds nothing (see GlossaryHits), so only the approved
		// entries count.
		if t.KR != "" && len(t.Files) == 0 && t.Status == StatusApproved {
			general[t.KR]++
			if general[t.KR] > 1 {
				l.report(GlossaryFile, "%s: more than one approved entry has no files; at most one approved entry of a KR text may apply to every file, the others need files", label)
			}
		}
		for _, key := range unknown[i] {
			l.report(GlossaryFile, "%s: unknown field %q", label, key)
		}
	}
}

// familyIDs returns the family ids of the routing, or nil when the routing
// cannot be read (its own lint reports that, and a family: scope is then not
// judged).
func (l *linter) familyIDs() []string {
	data, err := os.ReadFile(filepath.Join(l.root, filepath.FromSlash(RoutingFile)))
	if err != nil {
		return nil
	}
	rt, err := parseRouting(data)
	if err != nil {
		return nil
	}
	var ids []string
	for _, f := range rt.Families {
		ids = append(ids, f.ID)
	}
	return ids
}

// foreignIgnore checks the list of words the foreign-name warning leaves
// alone. The file is optional.
func (l *linter) foreignIgnore() {
	if !l.fileExists(ForeignIgnoreFile) {
		return
	}
	data, ok := l.read(ForeignIgnoreFile)
	if !ok {
		return
	}
	words, err := parseForeignIgnore(data)
	if err != nil {
		l.report(ForeignIgnoreFile, "must be a list of words: %v", err)
		return
	}
	seen := map[string]bool{}
	for i, w := range words {
		switch {
		case strings.TrimSpace(w) == "":
			l.report(ForeignIgnoreFile, "entry %d is empty", i+1)
		case seen[w]:
			l.report(ForeignIgnoreFile, "%q is listed twice", w)
		}
		seen[w] = true
	}
}

func (l *linter) characters() {
	files, err := characterFiles(l.root)
	if err != nil {
		l.report(CharactersDir, "%s", ioMessage(err))
		return
	}
	idOwner := map[string]string{}    // id -> first file with it
	modelOwner := map[string]string{} // model key -> first file with it
	for _, rel := range files {
		data, ok := l.read(rel)
		if !ok {
			continue
		}
		c, unknown, err := parseCharacter(data)
		if err != nil {
			l.report(rel, "%v", err)
			continue
		}
		for _, key := range unknown {
			l.report(rel, "unknown front matter field %q", key)
		}
		for _, f := range []struct{ name, value string }{{"id", c.ID}, {"name", c.Name}, {"name_kr", c.NameKR}} {
			if f.value == "" {
				l.report(rel, "front matter field %s is missing", f.name)
			}
		}
		if len(c.Models) == 0 {
			l.report(rel, "front matter field models is missing or empty")
		}
		if c.ID != "" {
			if stem := strings.TrimSuffix(path.Base(rel), ".md"); c.ID != stem {
				l.report(rel, "id %q does not match the file name %q", c.ID, stem)
			}
			if first, dup := idOwner[c.ID]; dup {
				l.report(rel, "id %q is also used by %s", c.ID, first)
			} else {
				idOwner[c.ID] = rel
			}
		}
		for _, model := range slices.Concat(c.Models, c.ModelsOtherSelf) {
			if first, dup := modelOwner[model]; dup && first != rel {
				l.report(rel, "model %q is also listed by %s", model, first)
			} else if !dup {
				modelOwner[model] = rel
			}
		}
	}
}

func (l *linter) routing() {
	data, ok := l.read(RoutingFile)
	if !ok {
		return
	}
	rt, err := parseRouting(data)
	if err != nil {
		l.report(RoutingFile, "%v", err)
		return
	}
	ids := map[string]bool{}
	for i, f := range rt.Families {
		label := fmt.Sprintf("family %q", f.ID)
		if f.ID == "" {
			label = fmt.Sprintf("family %d", i+1)
			l.report(RoutingFile, "%s: id is missing", label)
		} else if ids[f.ID] {
			l.report(RoutingFile, "%s: id is used twice", label)
		}
		ids[f.ID] = true
		switch {
		case f.Guide == "":
			l.report(RoutingFile, "%s: guide is missing", label)
		case !l.fileExists(f.Guide):
			l.report(RoutingFile, "%s: guide file %s does not exist", label, f.Guide)
		}
		if len(f.Match) == 0 && i != len(rt.Families)-1 {
			l.report(RoutingFile, "%s: has no match patterns, so it takes every file; only the last family may do that", label)
		}
	}
	if n := len(rt.Families); n == 0 || len(rt.Families[n-1].Match) != 0 {
		l.report(RoutingFile, "the last family must have an empty match list to take the remaining files")
	}
	for _, err := range globErrors(rt) {
		l.report(RoutingFile, "%v", err)
	}

	listed := map[string]bool{}
	for _, id := range rt.Priority {
		switch {
		case !ids[id]:
			l.report(RoutingFile, "priority lists %q, which is not a family", id)
		case listed[id]:
			l.report(RoutingFile, "priority lists %q twice", id)
		}
		listed[id] = true
	}
	for _, f := range rt.Families {
		if f.ID != "" && !listed[f.ID] {
			l.report(RoutingFile, "family %q is missing from priority", f.ID)
		}
	}
}

// adjustmentFields are the keys an entry of an adjustment file may have.
var adjustmentFields = func() map[string]bool {
	m := map[string]bool{}
	for _, f := range adjust.Fields {
		m[f] = true
	}
	return m
}()

// adjustments checks the files of rules/adjustments. Anything in the
// directory that Load would not read is reported, since a person who names a
// file x.yml would otherwise find the adjustment silently ignored; a README.md
// and a .gitkeep are allowed (rulefile.IsAside).
func (l *linter) adjustments() {
	entries, err := os.ReadDir(filepath.Join(l.root, filepath.FromSlash(adjust.Dir)))
	if errors.Is(err, fs.ErrNotExist) {
		return // no adjustments yet
	}
	if err != nil {
		l.report(adjust.Dir, "%s", ioMessage(err))
		return
	}
	keyOwner := map[string]string{} // key -> where it was first seen
	for _, e := range entries {
		rel := adjust.Dir + "/" + e.Name()
		switch {
		case !e.IsDir() && rulefile.IsAside(e.Name()):
		case e.IsDir() || !adjust.IsFile(e.Name()):
			l.report(rel, "not an adjustment file: only .yaml files directly in %s are read", adjust.Dir)
		default:
			l.adjustmentFile(rel, keyOwner)
		}
	}
}

// adjustmentFile checks the entries of one adjustment file. keyOwner holds
// the keys of the files before it: a key may be adjusted once across all
// files.
func (l *linter) adjustmentFile(rel string, keyOwner map[string]string) {
	data, ok := l.read(rel)
	if !ok {
		return
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		l.report(rel, "%v", err)
		return
	}
	if doc.Kind == 0 { // empty file
		return
	}
	list := doc.Content[0]
	if list.Kind != yaml.SequenceNode {
		l.report(rel, "line %d: an adjustment file must be a list of entries", list.Line)
		return
	}
	for i, entry := range list.Content {
		label := fmt.Sprintf("entry %d", i+1)
		if entry.Kind != yaml.MappingNode {
			l.report(rel, "%s: must be a mapping with the keys %s", label, strings.Join(adjust.Fields, ", "))
			continue
		}
		var a adjust.Adjustment
		if err := entry.Decode(&a); err != nil {
			l.report(rel, "%s: %v", label, err)
			continue
		}
		if a.Key != "" {
			label += fmt.Sprintf(" (%s)", a.Key)
		}

		present := map[string]bool{}
		for k := 0; k+1 < len(entry.Content); k += 2 {
			present[entry.Content[k].Value] = true
		}
		for _, name := range unknownKeys(entry, adjustmentFields) {
			l.report(rel, "%s: unknown field %q", label, name)
		}
		for _, name := range adjust.Fields {
			if !present[name] {
				l.report(rel, "%s: %s is missing", label, name)
			}
		}
		for _, p := range a.Problems(l.root) {
			if present[p.Field] {
				l.report(rel, "%s: %s", label, p.Message)
			}
		}

		if a.Key == "" {
			continue
		}
		if first, dup := keyOwner[a.Key]; dup {
			l.report(rel, "%s: the key is already adjusted by %s", label, first)
		} else {
			keyOwner[a.Key] = fmt.Sprintf("entry %d of %s", i+1, rel)
		}
	}
}

// LintScopeWarnings returns a warning for each files or not_files pattern of
// the glossary that matches none of krFiles, the file names of the KR pack
// (the names GlossaryHits receives): such a pattern is a misspelled name or a
// file the game has renamed, and scopes nothing. krFiles empty means the pack
// is not at hand, and there are no warnings. A glossary that does not parse is
// Lint's to report.
func LintScopeWarnings(root string, krFiles []string) []LintError {
	if len(krFiles) == 0 {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(GlossaryFile)))
	if err != nil {
		return nil
	}
	terms, _, err := parseGlossary(data)
	if err != nil {
		return nil
	}
	var rt Routing
	if data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(RoutingFile))); err == nil {
		rt, _ = parseRouting(data) // a routing that does not parse is Lint's to report; family: items are then not judged
	}
	r := &Rules{Routing: rt}
	var out []LintError
	for _, t := range terms {
		for _, list := range []struct {
			name     string
			patterns []string
		}{{"files", t.Files}, {"not_files", t.NotFiles}} {
			for _, pattern := range list.patterns {
				id, isFamily := strings.CutPrefix(pattern, FamilyPrefix)
				if isFamily && len(rt.Families) == 0 {
					continue // no routing to judge by
				}
				if !isFamily && (pattern == "" || config.ValidPattern(pattern) != nil) {
					continue // Lint reports these
				}
				if !slices.ContainsFunc(krFiles, func(f string) bool {
					if isFamily {
						return r.Family(f) == id
					}
					return config.MatchAny([]string{pattern}, f)
				}) {
					out = append(out, LintError{File: GlossaryFile, Message: fmt.Sprintf("term %q: %s pattern %q matches no file of the KR pack", t.KR, list.name, pattern)})
				}
			}
		}
	}
	return out
}
