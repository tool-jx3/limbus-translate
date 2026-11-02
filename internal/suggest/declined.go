package suggest

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/tool-jx3/limbus-translate/internal/rulefile"
)

// DeclinedFile is the list of the suggestions that people turned down,
// relative to the repo root. It is optional.
//
// A person who does not want a suggestion of a rules PR moves it to this file
// before merging the PR, and lct suggest does not make it again. The file is a
// list of entries:
//
//	# rules/declined.yaml
//	- id: speaker/아무개          # the id of the suggestion (see the kinds)
//	  reason: 路人，不需要指引      # why it is not adopted; required
//	  date: 2026-11-05           # YYYY-MM-DD
const DeclinedFile = "rules/declined.yaml"

// declinedFields are the keys of an entry, in the order they are written.
var declinedFields = []string{"id", "reason", "date"}

// Declined is one entry of DeclinedFile.
type Declined struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
	Date   string `json:"date"`
}

// LoadDeclined reads the declined list of root. A root without the file has
// none. A file that cannot be read as a list of entries (not YAML, not a list,
// an entry that is not a mapping or has a key of another name, a value of the
// wrong kind) is an error that names the file: a list that is skipped in
// silence would let a declined suggestion come back. An entry with a bad
// value (an id of no kind, an empty reason, a date that is no date) is
// returned as written; LintDeclined reports it.
func LoadDeclined(root string) ([]Declined, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(DeclinedFile)))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	list, problems := parseDeclined(data)
	if len(problems) > 0 {
		return nil, fmt.Errorf("%s: %s", DeclinedFile, strings.Join(problems, "; "))
	}
	return list, nil
}

// LintDeclined checks the declined list of root and returns every problem it
// finds: what LoadDeclined refuses, an id whose kind is not one of Kinds or
// that has nothing after the kind, an empty reason, a date that is not
// YYYY-MM-DD, and an id that is listed twice. A root without the file has no
// issues.
func LintDeclined(root string) []rulefile.Issue {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(DeclinedFile)))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		msg := err.Error()
		var pe *fs.PathError
		if errors.As(err, &pe) {
			msg = pe.Err.Error() // without the absolute path
		}
		return []rulefile.Issue{{File: DeclinedFile, Message: msg}}
	}
	list, problems := parseDeclined(data)
	if len(problems) == 0 {
		problems = declinedProblems(list)
	}
	issues := make([]rulefile.Issue, len(problems))
	for i, msg := range problems {
		issues[i] = rulefile.Issue{File: DeclinedFile, Message: msg}
	}
	return issues
}

// parseDeclined reads the entries of the file. The problems are what makes the
// file unreadable as a list of entries; with any, the entries are not to be
// used.
func parseDeclined(data []byte) (list []Declined, problems []string) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, []string{err.Error()}
	}
	if doc.Kind == 0 { // empty file
		return nil, nil
	}
	seq := doc.Content[0]
	if seq.Kind != yaml.SequenceNode {
		return nil, []string{fmt.Sprintf("line %d: the file must be a list of entries with the keys %s", seq.Line, strings.Join(declinedFields, ", "))}
	}
	for i, node := range seq.Content {
		label := fmt.Sprintf("entry %d", i+1)
		if node.Kind != yaml.MappingNode {
			problems = append(problems, fmt.Sprintf("%s: must be a mapping with the keys %s", label, strings.Join(declinedFields, ", ")))
			continue
		}
		var d Declined
		keys, err := rulefile.DecodeMapping(node, map[string]any{"id": &d.ID, "reason": &d.Reason, "date": &d.Date})
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", label, err))
			continue
		}
		for _, k := range keys {
			if !slices.Contains(declinedFields, k) {
				problems = append(problems, fmt.Sprintf("%s: unknown field %q (the fields are %s)", label, k, strings.Join(declinedFields, ", ")))
			}
		}
		list = append(list, d)
	}
	return list, problems
}

// declinedProblems lists what is wrong with the values of the entries, in the
// order of the entries.
func declinedProblems(list []Declined) []string {
	var ps []string
	first := map[string]int{} // id -> the entry that has it first
	for i, d := range list {
		label := fmt.Sprintf("entry %d", i+1)
		if d.ID != "" {
			label += fmt.Sprintf(" (%s)", d.ID)
		}
		add := func(format string, args ...any) { ps = append(ps, label+": "+fmt.Sprintf(format, args...)) }

		kind, rest, hasRest := strings.Cut(d.ID, "/")
		switch {
		case d.ID == "":
			add("id is missing or empty; it is the id of a suggestion, such as speaker/<model>")
		case !slices.Contains(Kinds, kind):
			add("unknown kind %q (the kinds are %s)", kind, strings.Join(Kinds, ", "))
		case !hasRest || rest == "":
			add("nothing after the kind %q; the id is <kind>/<what the suggestion is about>", kind)
		}
		if strings.TrimSpace(d.Reason) == "" {
			add("reason is missing or empty; say why the suggestion is not adopted")
		}
		switch t, err := time.Parse("2006-01-02", d.Date); {
		case d.Date == "":
			add("date is missing; write the day in YYYY-MM-DD")
		case err != nil || t.Format("2006-01-02") != d.Date:
			add("date %q is not YYYY-MM-DD", d.Date)
		}
		if d.ID != "" {
			if n, dup := first[d.ID]; dup {
				add("id %q is listed twice (entry %d has it first)", d.ID, n)
			} else {
				first[d.ID] = i + 1
			}
		}
	}
	return ps
}
