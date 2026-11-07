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
	"github.com/tool-jx3/limbus-translate/internal/validate"
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
	characters := l.characterIDs()
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
		if t.Check != "" && t.Check != CheckHard && t.Check != CheckWarn {
			l.report(GlossaryFile, "%s: check is %q, want %s or %s", label, t.Check, CheckHard, CheckWarn)
		}
		if t.Meaning != "" && len(t.Sources) == 0 {
			l.report(GlossaryFile, "%s: meaning needs sources (game:<file>#<pointer> or a limbuscompany.wiki.gg URL)", label)
		}
		for _, src := range t.Sources {
			if !SourceRe.MatchString(src) {
				l.report(GlossaryFile, "%s: sources holds %q, want game:<file>#<pointer> or https://limbuscompany.wiki.gg/...", label, src)
			}
		}
		for _, f := range t.Forbidden {
			if pattern, isRegexp := strings.CutPrefix(f, "re:"); isRegexp {
				if _, err := regexp.Compile(pattern); err != nil {
					l.report(GlossaryFile, `%s: forbidden pattern "%s" is not a valid regular expression: %v`, label, f, err)
				}
			}
		}
		for _, a := range t.Also {
			switch {
			case a == "":
				l.report(GlossaryFile, "%s: also holds an empty rendering", label)
			case strings.TrimSpace(validate.MatchText(a)) == "":
				l.report(GlossaryFile, "%s: also holds %q, which is only tags and would match every translation", label, a)
			case a == t.ZH:
				l.report(GlossaryFile, "%s: also holds %q, which is the zh already", label, a)
			case validate.MatchText(a) == validate.MatchText(t.ZH):
				l.report(GlossaryFile, "%s: also holds %q, which is the zh already once its tags are stripped", label, a)
			default:
				if f, ok := forbiddenIn(a, t.Forbidden); ok {
					l.report(GlossaryFile, "%s: also holds %q, which forbidden %q rules out", label, a, f)
				}
			}
		}
		if t.Match != "" && t.Match != MatchWord && t.Match != MatchSubstring {
			l.report(GlossaryFile, "%s: match is %q, want %q (hit only where the KR stands as a word; the default) or %q (hit inside a longer word too)", label, t.Match, MatchWord, MatchSubstring)
		}
		for _, list := range []struct {
			name     string
			patterns []string
		}{{"files", t.Files}, {"not_files", t.NotFiles}} {
			for _, pattern := range list.patterns {
				if msg := scopeItemError(pattern, families, characters); msg != "" {
					l.report(GlossaryFile, "%s: %s %s", label, list.name, msg)
				}
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
		if f.Glossary != "" && f.Glossary != GlossaryHard && f.Glossary != GlossaryWarn {
			l.report(RoutingFile, "%s: glossary %q must be empty, %q or %q", label, f.Glossary, GlossaryHard, GlossaryWarn)
		}
		switch {
		case f.Guide == "":
			l.report(RoutingFile, "%s: guide is missing", label)
		case !l.fileExists(f.Guide):
			l.report(RoutingFile, "%s: guide file %s does not exist", label, f.Guide)
		}
		if len(f.Nodes) == 0 && i != len(rt.Families)-1 {
			l.report(RoutingFile, "%s: has no nodes, so it takes every file no other family lists; only the last family may do that", label)
		}
	}
	if n := len(rt.Families); n == 0 || len(rt.Families[n-1].Nodes) != 0 {
		l.report(RoutingFile, "the last family must have an empty nodes list to take the remaining files")
	}
	for _, id := range matchKeys(data) {
		l.report(RoutingFile, "family %q: match is no longer read; list node names or patterns in nodes", id)
	}
	for _, err := range patternErrors(rt) {
		l.report(RoutingFile, "%v", err)
	}

	for i, g := range rt.Guides {
		label := fmt.Sprintf("guides entry %d", i+1)
		if msg := whenError(g.When); msg != "" {
			l.report(RoutingFile, "%s: %s", label, msg)
		}
		switch {
		case g.Guide == "":
			l.report(RoutingFile, "%s: guide is missing", label)
		case !l.fileExists(g.Guide):
			l.report(RoutingFile, "%s: guide file %s does not exist", label, g.Guide)
		}
	}
	l.familyGuideSizes()

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

// FamilyGuidesDir holds the family guides; a file there may be
// MaxFamilyGuideBytes long. MaxGuideTotalBytes is the limit of the guides of one
// batch (speakers' characters left out; see LintRoutingWarnings).
const (
	FamilyGuidesDir     = "guides/families"
	MaxFamilyGuideBytes = 4096
	MaxGuideTotalBytes  = 35000
)

// familyGuideSizes reports the guides/families/*.md files over the limit.
func (l *linter) familyGuideSizes() {
	entries, err := os.ReadDir(filepath.Join(l.root, filepath.FromSlash(FamilyGuidesDir)))
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		info, err := e.Info()
		if err == nil && info.Size() > MaxFamilyGuideBytes {
			l.report(FamilyGuidesDir+"/"+e.Name(), "is %d bytes, over the limit of %d", info.Size(), MaxFamilyGuideBytes)
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

// scopeItemError describes what is wrong with an item of files or not_files,
// or returns "". families and characters are the known ids; nil means they
// could not be read and the tokens naming them are not judged.
func scopeItemError(item string, families, characters []string) string {
	prefix, arg, isToken := cutScopeToken(item)
	switch {
	case item == "":
		return "holds an empty pattern"
	case !isToken:
		if err := config.ValidPattern(item); err != nil {
			return fmt.Sprintf("pattern %q is not valid: %v", item, err)
		}
		return ""
	case arg == "":
		return fmt.Sprintf("holds %q without a name after it", item)
	}
	switch prefix {
	case FamilyPrefix:
		if families != nil && !slices.Contains(families, arg) {
			return fmt.Sprintf("names family %q, which is not a family of %s", arg, RoutingFile)
		}
	case GroupPrefix, EventPrefix:
		if _, err := path.Match(arg, ""); err != nil {
			return fmt.Sprintf("token %q is not a valid pattern: %v", item, err)
		}
	case ChapterPrefix:
		if strings.Trim(arg, "0123456789") != "" {
			return fmt.Sprintf("token %q needs a chapter number (digits only)", item)
		}
	case CharacterPrefix:
		if characters != nil && !slices.Contains(characters, arg) {
			return fmt.Sprintf("token %q names a character with no file in %s", item, CharactersDir)
		}
	}
	return ""
}

// characterIDs returns the ids of the character guides, or nil when they
// cannot be read (Lint reports that).
func (l *linter) characterIDs() []string {
	chars, err := loadCharacters(l.root)
	if err != nil {
		return nil
	}
	return characterIDList(chars)
}

// loadCharacters reads the character guides under root; one that cannot be
// read or parsed is left out (Lint reports it).
func loadCharacters(root string) ([]Character, error) {
	files, err := characterFiles(root)
	if err != nil {
		return nil, err
	}
	chars := []Character{}
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		if c, _, err := parseCharacter(data); err == nil {
			c.Path = rel
			chars = append(chars, c)
		}
	}
	return chars, nil
}

func characterIDList(chars []Character) []string {
	ids := []string{}
	for _, c := range chars {
		if c.ID != "" {
			ids = append(ids, c.ID)
		}
	}
	return ids
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
	// A missing guides/characters, or one that is not a directory, leaves
	// r.Characters nil (a single unreadable guide is skipped, and only its id
	// is missing from the list), but characterIDList of it is an empty,
	// non-nil list. scopeItemError then judges every
	// character: token and reports an error for each, and the loop below skips
	// the patterns it reports (Lint reports them), so no warning appears.
	r.Characters, _ = loadCharacters(root)
	characterIDs := characterIDList(r.Characters)
	families := make([]string, 0, len(rt.Families))
	for _, f := range rt.Families {
		families = append(families, f.ID)
	}
	var out []LintError
	for _, t := range terms {
		for _, list := range []struct {
			name     string
			patterns []string
		}{{"files", t.Files}, {"not_files", t.NotFiles}} {
			for _, pattern := range list.patterns {
				if scopeItemError(pattern, families, characterIDs) != "" {
					continue // no routing to judge by, an unknown name or a bad pattern: Lint reports these as errors
				}
				if !slices.ContainsFunc(krFiles, func(f string) bool { return r.matchScope([]string{pattern}, f) }) {
					out = append(out, LintError{File: GlossaryFile, Message: fmt.Sprintf("term %q: %s pattern %q matches no file of the KR pack", t.KR, list.name, pattern)})
				}
			}
		}
	}
	return out
}

// forbiddenIn returns the first item of forbidden that a translation holding
// rendering would trip (validate's rule: a substring, or a regular expression
// for an item starting with "re:"), if there is one. An invalid pattern is
// reported elsewhere and skipped here.
func forbiddenIn(rendering string, forbidden []string) (string, bool) {
	for _, f := range forbidden {
		if f == "" || f == "re:" {
			continue
		}
		pattern, isRegexp := strings.CutPrefix(f, "re:")
		if !isRegexp {
			if strings.Contains(rendering, f) {
				return f, true
			}
			continue
		}
		if re, err := regexp.Compile(pattern); err == nil && re.MatchString(rendering) {
			return f, true
		}
	}
	return "", false
}

// LintRoutingWarnings returns the warnings of the routing against the KR pack
// krFiles (the file names Family receives): a nodes entry that fits no node of
// the pack (a misspelled name, or a node the game has dropped), top nodes of
// files that fall in the fallback family (a new node no family lists), and a
// node two families of the same level both list (the warning says which one
// takes it). krFiles empty means the pack is not at hand, and there are no
// warnings. A routing that does not parse is Lint's to report.
func LintRoutingWarnings(root string, krFiles []string) []LintError {
	if len(krFiles) == 0 {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(RoutingFile)))
	if err != nil {
		return nil
	}
	rt, err := parseRouting(data)
	if err != nil || len(patternErrors(rt)) != 0 {
		return nil
	}
	tops, subs := map[string]bool{}, map[string]bool{}
	unsorted := map[string]bool{}
	fb := rt.fallback()
	for _, f := range krFiles {
		n := FileNode(f)
		tops[n.Top] = true
		if n.Sub != "" {
			subs[n.Sub] = true
		}
		if fb >= 0 && rt.route(f) == fb {
			unsorted[n.Top] = true
		}
	}
	topList, subList := sortedKeys(tops), sortedKeys(subs)
	var out []LintError
	warn := func(format string, args ...any) {
		out = append(out, LintError{File: RoutingFile, Message: fmt.Sprintf(format, args...)})
	}
	for _, f := range rt.Families {
		for _, pattern := range f.Nodes {
			fitsNode := func(node string) bool {
				ok, _ := path.Match(pattern, node)
				return ok
			}
			if !slices.ContainsFunc(topList, fitsNode) && !slices.ContainsFunc(subList, fitsNode) {
				warn("family %q: nodes pattern %q matches no node of the KR pack", f.ID, pattern)
			}
		}
	}
	if len(unsorted) > 0 {
		warn("files of these top nodes fall in %q, which no family lists: %s", rt.Families[fb].ID, strings.Join(sortedKeys(unsorted), ", "))
	}
	for _, level := range [][]string{topList, subList} {
		for _, node := range level {
			var ids []string
			byName := false
			for _, f := range rt.Families {
				exact, wildcard := f.fits(node)
				if exact || wildcard {
					ids = append(ids, f.ID)
				}
				byName = byName || exact
			}
			if len(ids) < 2 {
				continue
			}
			why := "listed first"
			if byName {
				why = "it lists the full name"
			}
			warn("node %q is fit by families %s; %s takes it (%s)", node, strings.Join(ids, ", "), rt.Families[rt.routeNode(node)].ID, why)
		}
	}
	out = append(out, guideTotalWarnings(root, krFiles)...)
	return out
}

// guideTotalWarnings warns of the KR files whose GuidesFor (no speakers) add
// up to more than MaxGuideTotalBytes.
func guideTotalWarnings(root string, krFiles []string) []LintError {
	r, err := Load(root)
	if err != nil {
		return nil // Lint reports what does not load
	}
	sizes := map[string]int64{}
	size := func(rel string) int64 {
		n, ok := sizes[rel]
		if !ok {
			if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err == nil {
				n = info.Size()
			}
			sizes[rel] = n
		}
		return n
	}
	var out []LintError
	for _, f := range krFiles {
		var total int64
		for _, g := range r.GuidesFor(f, nil) {
			total += size(g)
		}
		if total > MaxGuideTotalBytes {
			out = append(out, LintError{File: RoutingFile, Message: fmt.Sprintf("the guides of %s add up to %d bytes, over %d", f, total, MaxGuideTotalBytes)})
		}
	}
	return out
}

// sortedKeys lists the keys of set in order.
func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
