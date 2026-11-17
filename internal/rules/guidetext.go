package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// TranslatorGuides lists the guide files the translator reads: the base
// guide and the family, shared and character guides that exist. The paths are
// relative to root, slash-separated and sorted.
func TranslatorGuides(root string) ([]string, error) {
	var out []string
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(BaseGuide))); err == nil {
		out = append(out, BaseGuide)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	for _, dir := range []string{FamilyGuidesDir, SharedGuidesDir, CharactersDir} {
		entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(dir)))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				out = append(out, dir+"/"+e.Name())
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

// guideLines returns the lines of a guide the translator reads.
func guideLines(root, rel string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return nil, err
	}
	return strings.Split(string(data), "\n"), nil
}

// guideEvidenceLinks reports a guide the translator reads that points to the
// evidence files, or that cannot be read.
func (l *linter) guideEvidenceLinks() {
	files, err := TranslatorGuides(l.root)
	if err != nil {
		l.report("guides", "%s", ioMessage(err))
		return
	}
	for _, rel := range files {
		lines, err := guideLines(l.root, rel)
		if err != nil {
			l.report(rel, "%s", ioMessage(err))
			continue
		}
		for i, line := range lines {
			if strings.Contains(line, "guides-evidence") {
				l.report(rel, "line %d mentions guides-evidence: a guide the translator reads must not point to the evidence files", i+1)
			}
		}
	}
}

var evidencePatterns = regexp.MustCompile(`既有 ?[0-9]|[0-9][0-9,]*／[0-9]|[0-9]+(?:\.[0-9]+)?%[（）]|使用者(?:決定|確認)|raw-20[0-9]{6}|docs/research/`)

// LintGuideTextWarnings warns of each line of a guide the translator reads
// that looks like evidence: statistics or a source. A guide that cannot be
// read is a warning too, since it was not checked.
func LintGuideTextWarnings(root string) []LintError {
	files, err := TranslatorGuides(root)
	if err != nil {
		return []LintError{{File: "guides", Message: ioMessage(err)}}
	}
	var out []LintError
	for _, rel := range files {
		lines, err := guideLines(root, rel)
		if err != nil {
			out = append(out, LintError{File: rel, Message: ioMessage(err)})
			continue
		}
		for i, line := range lines {
			if evidencePatterns.MatchString(line) {
				out = append(out, LintError{File: rel, Message: fmt.Sprintf("line %d looks like evidence (statistics or a source); move it to docs/guides-evidence/<same path>", i+1)})
			}
		}
	}
	return out
}
