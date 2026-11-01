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

var validStatus = []string{StatusApproved, "candidate"}

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
		for _, key := range unknown[i] {
			l.report(GlossaryFile, "%s: unknown field %q", label, key)
		}
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
