// Package rewrite applies the rewrite files (spec 11.1): a person asks,
// through main, for a mechanical change to the translations that exist, such
// as a half-width space after every keyword name, and the plan applies it to
// the records once.
//
// A rewrite never sends a unit back to the translator and never changes
// anything but the text of a record. The plan, which holds the pending queue,
// carries the new text on to a pending item of the unit that showed the old
// one as its previous translation, so that the translation made from it keeps
// the rewrite.
package rewrite

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/rulefile"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/units"
)

// Dir holds the rewrite files, relative to the repo root.
//
// A rewrite file is told apart by its ID, the name without ".yaml", and the
// ID is recorded when the file is applied, so renaming an applied file
// rewrites again.
//
//	reason: 關鍵字名稱後加半形空格
//	files: [StoryData/**, Bufs*.json]   # patterns as in lct.yaml (config.MatchAny); required
//	origins: [paratranz, claude]        # default: paratranz and claude
//	pattern: '(\d+級破裂)(強度)'         # Go regexp, matched against the translation
//	replace: '$1 $2'                    # as in regexp.ReplaceAllString; may be empty (${1}級, not $1級)
//	kr_pattern: '강도'                   # optional: the KR text must match it too
const Dir = "rules/rewrite"

// fieldNames are the keys of a rewrite file, in the order the doc of Dir shows.
var fieldNames = []string{"reason", "files", "origins", "pattern", "replace", "kr_pattern"}

// defaultOrigins are the origins a rewrite changes when the file names none:
// what a person did not write, and no copy or stand-in.
var defaultOrigins = []string{store.OriginParatranz, store.OriginClaude}

// File is one rewrite file.
type File struct {
	// ID is the name of the file without ".yaml".
	ID     string
	Reason string
	// Files are the patterns of the pack files to rewrite (config.MatchAny).
	Files []string
	// Origins are the origins of the records to rewrite; empty means the
	// default (paratranz and claude).
	Origins []string
	// Pattern is a Go regular expression the translation must match, and
	// Replace is what each match is replaced with (regexp.ReplaceAllString).
	Pattern string
	Replace string
	// KRPattern, when set, is a regular expression the KR text of the unit
	// must match too.
	KRPattern string
}

// problems lists what is wrong with the values of f. present says which keys
// the file has: a replace that is left out (or written without a value, which
// rulefile.Decode counts as left out) is a mistake, while an empty one ("")
// deletes the matches.
func (f File) problems(present map[string]bool) []string {
	var ps []string
	if strings.TrimSpace(f.Reason) == "" {
		ps = append(ps, "reason is missing or empty")
	}
	if len(f.Files) == 0 {
		ps = append(ps, "files is missing or empty; name the pack files to rewrite")
	}
	for _, p := range f.Files {
		if err := config.ValidPattern(p); err != nil {
			ps = append(ps, fmt.Sprintf("files: pattern %q: %v", p, err))
		}
	}
	for _, o := range f.Origins {
		if !slices.Contains(store.Origins, o) {
			ps = append(ps, fmt.Sprintf("unknown origin %q (want one of %s)", o, strings.Join(store.Origins, ", ")))
		}
	}
	if f.Pattern == "" {
		ps = append(ps, "pattern is missing or empty")
	} else if re, err := regexp.Compile(f.Pattern); err != nil {
		ps = append(ps, fmt.Sprintf("pattern is not a valid regular expression: %v", err))
	} else {
		ps = append(ps, replaceProblems(f.Replace, re)...)
	}
	if !present["replace"] {
		ps = append(ps, `replace is missing; write replace: "" to delete what the pattern matches`)
	}
	if f.KRPattern != "" {
		if _, err := regexp.Compile(f.KRPattern); err != nil {
			ps = append(ps, fmt.Sprintf("kr_pattern is not a valid regular expression: %v", err))
		}
	}
	return ps
}

// replaceProblems lists the groups template refers to that re does not have.
// regexp.ReplaceAllString reads such a reference as empty text, so a mistake
// would silently delete text from every translation it matches; the usual one
// is $1級, which refers to the group "1級": a name runs on through every
// letter, digit and "_", Chinese characters too.
func replaceProblems(template string, re *regexp.Regexp) []string {
	var ps []string
	for _, name := range groupRefs(template) {
		// A number without a leading zero is a group number, anything else a name.
		n, err := strconv.Atoi(name)
		var has bool
		if numbered := err == nil && (name[0] != '0' || len(name) == 1); numbered {
			has = n <= re.NumSubexp()
		} else {
			has = re.SubexpIndex(name) >= 0
		}
		if has {
			continue
		}
		ps = append(ps, fmt.Sprintf("replace refers to the group %q, which the pattern does not have (a name runs on through letters and digits, so write ${1}級 and not $1級)", name))
	}
	return ps
}

// groupRefs lists the names (or numbers) of the groups template refers to, as
// regexp.Regexp.Expand reads it: $name or ${name}, where $$ is a dollar sign
// and a $ that no name follows is text.
func groupRefs(template string) []string {
	var refs []string
	for {
		_, after, found := strings.Cut(template, "$")
		if !found {
			return refs
		}
		template = after
		if strings.HasPrefix(template, "$") {
			template = template[1:]
			continue
		}
		s, braced := strings.CutPrefix(template, "{")
		end := 0
		for end < len(s) {
			r, size := utf8.DecodeRuneInString(s[end:])
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
				break
			}
			end += size
		}
		name := s[:end]
		if end == 0 || braced && (end >= len(s) || s[end] != '}') {
			continue // not a reference
		}
		if braced {
			end++
		}
		refs = append(refs, name)
		template = s[end:]
	}
}

// parse reads the content of a rewrite file and says everything that is wrong
// with it. The File is only usable when there are no problems.
func parse(data []byte) (File, []string) {
	var f File
	fields := map[string]any{"reason": &f.Reason, "files": &f.Files, "origins": &f.Origins, "pattern": &f.Pattern, "replace": &f.Replace, "kr_pattern": &f.KRPattern}
	keys, err := rulefile.Decode(data, fields)
	if err != nil {
		return File{}, []string{err.Error()}
	}
	var ps []string
	present := map[string]bool{}
	for _, k := range keys {
		present[k] = true
		if _, known := fields[k]; !known {
			ps = append(ps, fmt.Sprintf("unknown field %q (the fields are %s)", k, strings.Join(fieldNames, ", ")))
		}
	}
	return f, append(ps, f.problems(present)...)
}

// LoadFiles reads the rewrite files of root: Dir/*.yaml in order of file name.
// A root without Dir has none. A file that cannot be applied is an error that
// names the file and says what is wrong: an unknown key, a missing field, a
// regular expression that does not compile, a malformed pattern of files.
// (rules lint lists the problems of every file.)
func LoadFiles(root string) ([]File, error) {
	entries, err := rulefile.ReadDir(root, Dir)
	if err != nil {
		return nil, err
	}
	var files []File
	for _, e := range entries {
		f, problems := parse(e.Data)
		if len(problems) > 0 {
			return nil, fmt.Errorf("%s/%s: %s", Dir, e.Name, strings.Join(problems, "; "))
		}
		f.ID = rulefile.ID(e.Name)
		files = append(files, f)
	}
	return files, nil
}

// Lint checks the rewrite files of root and returns every problem it finds.
func Lint(root string) []rulefile.Issue {
	return rulefile.Lint(root, Dir, func(data []byte) []string {
		_, problems := parse(data)
		return problems
	})
}

// Apply rewrites the translations of recs that f selects and returns the keys
// of the records it changed, sorted. A record is rewritten when its origin is
// one of f.Origins (paratranz and claude without any), the file of its key is
// in f.Files, its text matches f.Pattern, and, if f.KRPattern is set, so does
// the KR text of its unit. Every match of the pattern is replaced.
//
// The new text goes through ok, with the key, the KR text and the new text of
// the unit; a record whose new text ok refuses is left as it was. ok may be
// nil. It is where the caller holds the rewrite to the rules, such as
// validation, and keeps it out of what must not change (Apply does not know
// the frozen files of lct.yaml). A record is changed in recs, and only its ZH:
// never its origin or anything else, and not at all when the text stays the
// same.
//
// A record whose unit KR no longer has (the file or the line is gone), or any
// record when kr is nil, is left alone: without the KR text ok cannot decide.
//
// f must be valid, as a File from LoadFiles is: Apply panics on a pattern that
// does not compile.
func Apply(f File, recs map[string]store.Record, kr *pack.Pack, ok func(key, kr, zh string) bool) []string {
	re := regexp.MustCompile(f.Pattern)
	var krRe *regexp.Regexp
	if f.KRPattern != "" {
		krRe = regexp.MustCompile(f.KRPattern)
	}
	origins := f.Origins
	if len(origins) == 0 {
		origins = defaultOrigins
	}
	krTexts := newKRTexts(kr)

	var changed []string
	for key, rec := range recs {
		// The cheap tests first: there are 250 thousand records, and the KR
		// is read only for a file that has a unit to rewrite.
		if !slices.Contains(origins, rec.Origin) || !config.MatchAny(f.Files, fileOf(key)) || !re.MatchString(rec.ZH) {
			continue
		}
		zh := re.ReplaceAllString(rec.ZH, f.Replace)
		if zh == rec.ZH {
			continue
		}
		krText, found := krTexts.get(key)
		if !found || (krRe != nil && !krRe.MatchString(krText)) {
			continue
		}
		if ok != nil && !ok(key, krText, zh) {
			continue
		}
		rec.ZH = zh
		recs[key] = rec
		changed = append(changed, key)
	}
	slices.Sort(changed)
	return changed
}

// fileOf is the file part of a unit key. Keys of records are valid.
func fileOf(key string) string {
	file, _, _ := strings.Cut(key, "|")
	return file
}

// krTexts looks up the KR text of units, reading each KR file once.
type krTexts struct {
	kr     *pack.Pack
	byFile map[string]map[string]string // file -> pointer -> KR text; nil for a file KR lacks
}

func newKRTexts(kr *pack.Pack) *krTexts {
	return &krTexts{kr: kr, byFile: map[string]map[string]string{}}
}

// get returns the KR text of the unit of key, and whether KR has the unit.
func (k *krTexts) get(key string) (string, bool) {
	file, pointer, err := units.SplitKey(key)
	if err != nil {
		return "", false
	}
	texts, loaded := k.byFile[file]
	if !loaded {
		texts = k.load(file)
		k.byFile[file] = texts
	}
	text, found := texts[pointer]
	return text, found
}

// load reads the KR texts of file by pointer. A file that KR lacks or cannot
// read has none.
func (k *krTexts) load(file string) map[string]string {
	if k.kr == nil || !k.kr.Has(file) {
		return nil
	}
	doc, err := k.kr.Doc(file)
	if err != nil {
		return nil
	}
	list := units.Extract(file, doc)
	texts := make(map[string]string, len(list))
	for _, u := range list {
		texts[u.Pointer] = u.KR
	}
	return texts
}
