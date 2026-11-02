package requeue

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/tool-jx3/limbus-translate/internal/rulefile"
)

// Dir holds the request files, relative to the repo root.
//
// A request file is how a person asks, through main, for units to be
// translated again (spec 11.1): the plan sends the records it selects back to
// the pending queue, once. A selected unit that waits for a person (flagged
// needs_review) is reset, so it is handed out again with the reason as its
// hint, as lct requeue --needs-review does where the state is. The file is
// told apart by its ID, the name without ".yaml", and the ID is recorded when
// the file is applied, so renaming an applied file requeues its units again.
//
//	reason: 稱呼改用「您」          # the hint the translator sees; required
//	keys:    [StoryData/E002X.json|/dataList/14/content]
//	files:   [StoryData/E002*]    # patterns as in lct.yaml (config.MatchAny)
//	origins: [claude]             # default: all but copy
//
// The keys, files and origins mean what --key, --file and --origin of lct
// requeue mean (Selector); at least one key or file is needed.
const Dir = "rules/requeue"

// fieldNames are the keys of a request file, in the order they are written.
var fieldNames = []string{"reason", "keys", "files", "origins"}

// ErrInvalid marks a request (or file name) that WriteFile refuses because of
// what it says, before anything is written. Other errors are system errors.
var ErrInvalid = errors.New("invalid requeue file")

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

// File is one request file.
type File struct {
	// ID is the name of the file without ".yaml".
	ID string
	// Reason is the hint the translator is shown for every unit requeued.
	Reason string
	// Selector has the Keys, Files and Origins of the file; NeedsReview and
	// Leased are never set from a file.
	Selector Selector
}

// problems lists what is wrong with the reason and the selector of f.
func (f File) problems() []string {
	var ps []string
	if strings.TrimSpace(f.Reason) == "" {
		ps = append(ps, "reason is missing or empty; the translator is shown it as the hint")
	}
	s := f.Selector
	if s.stuckMode() {
		ps = append(ps, "a request file requeues records; resetting and releasing pending units are not part of it")
	}
	if len(s.Keys)+len(s.Files) == 0 {
		ps = append(ps, "name at least one key or file (origins alone would requeue every file)")
	}
	if s.narrowed() {
		if err := s.Validate(); err != nil {
			ps = append(ps, err.Error())
		}
	}
	return ps
}

// parse reads the content of a request file and says everything that is wrong
// with it. The File is only usable when there are no problems.
func parse(data []byte) (File, []string) {
	var f File
	fields := map[string]any{"reason": &f.Reason, "keys": &f.Selector.Keys, "files": &f.Selector.Files, "origins": &f.Selector.Origins}
	keys, err := rulefile.Decode(data, fields)
	if err != nil {
		return File{}, []string{err.Error()}
	}
	var ps []string
	for _, k := range keys {
		if _, known := fields[k]; !known {
			ps = append(ps, fmt.Sprintf("unknown field %q (the fields are %s)", k, strings.Join(fieldNames, ", ")))
		}
	}
	return f, append(ps, f.problems()...)
}

// LoadFiles reads the request files of root: Dir/*.yaml in order of file name.
// A root without Dir has none. A file that cannot be applied is an error that
// names the file and says what is wrong: an unknown key, a missing reason,
// nothing selected, an origin that does not exist, a malformed key or pattern.
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

// Lint checks the request files of root and returns every problem it finds.
func Lint(root string) []rulefile.Issue {
	return rulefile.Lint(root, Dir, func(data []byte) []string {
		_, problems := parse(data)
		return problems
	})
}

// WriteFile writes the request of sel and reason to the file
// Dir/<date>-<topic>.yaml under root and returns the path of that file relative
// to root, with forward slashes. date is YYYY-MM-DD.
//
// WriteFile refuses (an error that wraps ErrInvalid) a request that LoadFiles
// would refuse, a Selector for the pending list (NeedsReview, Leased), a date
// or topic that is not usable in a file name, and a file that exists: it may
// have been applied already, and a request is applied only once. It writes
// nothing then.
func WriteFile(root, date, topic string, sel Selector, reason string) (string, error) {
	if err := rulefile.ValidateName(date, topic); err != nil {
		return "", invalid("%v", err)
	}
	f := File{Reason: reason, Selector: sel}
	if ps := f.problems(); len(ps) > 0 {
		return "", invalid("%s", strings.Join(ps, "; "))
	}

	rel := Dir + "/" + date + "-" + topic + rulefile.Ext
	path := filepath.Join(root, filepath.FromSlash(rel))
	if _, err := os.Stat(path); err == nil {
		return "", invalid("%s exists already; a request is applied once, so give the new one another topic or date", rel)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	data, err := encode(f)
	if err != nil {
		return "", err
	}
	if err := rulefile.WriteAtomic(path, data); err != nil {
		return "", err
	}
	return rel, nil
}

// encode writes f as YAML in the layout of the doc of Dir. The emitter of
// yaml.v3 gets some strings wrong (a text that starts with a blank or a
// newline comes out as a block scalar that cannot be read back), and a file
// that does not load would stop the plan, so the result is read back and
// compared. When it differs, every text is written double-quoted, which the
// emitter does right.
func encode(f File) ([]byte, error) {
	for _, style := range []yaml.Style{0, yaml.DoubleQuotedStyle} {
		text := func(s string) *yaml.Node {
			return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s, Style: style}
		}
		list := func(items []string) *yaml.Node {
			n := &yaml.Node{Kind: yaml.SequenceNode}
			for _, s := range items {
				n.Content = append(n.Content, text(s))
			}
			return n
		}
		root := &yaml.Node{Kind: yaml.MappingNode}
		add := func(name string, value *yaml.Node) {
			root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: name}, value)
		}
		add("reason", text(f.Reason))
		for _, l := range []struct {
			name  string
			items []string
		}{{"keys", f.Selector.Keys}, {"files", f.Selector.Files}, {"origins", f.Selector.Origins}} {
			if len(l.items) > 0 {
				add(l.name, list(l.items))
			}
		}

		var buf bytes.Buffer
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2)
		err := enc.Encode(root)
		if cerr := enc.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return nil, err
		}
		if back, problems := parse(buf.Bytes()); len(problems) == 0 && sameRequest(back, f) {
			return buf.Bytes(), nil
		}
	}
	return nil, errors.New("requeue: the request cannot be written as YAML")
}

// sameRequest reports whether a and b ask for the same, ignoring the ID.
func sameRequest(a, b File) bool {
	return a.Reason == b.Reason &&
		slices.Equal(a.Selector.Keys, b.Selector.Keys) &&
		slices.Equal(a.Selector.Files, b.Selector.Files) &&
		slices.Equal(a.Selector.Origins, b.Selector.Origins)
}
