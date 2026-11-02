package adjust

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/tool-jx3/limbus-translate/internal/rulefile"
)

// Move says that the adjustment of key From is in effect at the unit of key To
// now: lines were inserted into or removed from its file before the unit
// (plan.MovedKey).
type Move struct{ From, To string }

// Tidied is what a call of TidyReport did.
type Tidied struct {
	// Changed are the files edited or deleted, as "rules/adjustments/<name>"
	// and in order of name.
	Changed []string
	// Moved counts the adjustments that got a new key, Removed those that
	// were deleted.
	Moved, Removed int
}

// Tidy is TidyReport without the counts: it returns the files it changed.
func Tidy(root string, moves []Move, expired []string, settled func(key string) bool) ([]string, error) {
	done, err := TidyReport(root, moves, expired, settled)
	return done.Changed, err
}

// TidyReport brings the adjustment files of root up to date with what the
// plan found (spec 11.1, 14.2).
//
//   - The adjustment of key Move.From gets the key Move.To, and nothing else
//     in it changes.
//   - The adjustment of a key of expired goes when settled(key) is true: its
//     unit was translated again, so the adjustment is only a reference that
//     nobody needs (settled is asked about each such key once, and only about
//     keys that a file holds). A file left without an entry is deleted.
//
// A move that would give two adjustments one key is not made: lint wants a
// key once across all files. Everything else in a file stays as it is, byte
// for byte: comments, blank lines, the way each value is written. The edit is
// made on the text of the file, at the lines of the entries, and read back
// before anything is written; if the file cannot be edited like that (a flow
// list such as [{key: ...}], a second document, a key written as a block
// scalar) or the edit does not read back as meant, it fails with an error
// that wraps ErrInvalid and names the file, and writes nothing, in any file.
// A file without an entry to change is not looked at that closely.
func TidyReport(root string, moves []Move, expired []string, settled func(key string) bool) (Tidied, error) {
	entries, err := rulefile.ReadDir(root, Dir)
	if err != nil {
		return Tidied{}, err
	}
	files := make([]*tidyFile, len(entries))
	for i, e := range entries {
		if files[i], err = parseTidyFile(e); err != nil {
			return Tidied{}, err
		}
	}

	moveTo := map[string]string{}
	for _, m := range moves {
		if _, dup := moveTo[m.From]; !dup && m.From != m.To {
			moveTo[m.From] = m.To
		}
	}
	isExpired := map[string]bool{}
	for _, key := range expired {
		isExpired[key] = true
	}
	asked := map[string]bool{} // key -> the answer of settled
	for _, f := range files {
		for _, it := range f.items {
			key := it.adj.Key
			if isExpired[key] {
				answer, known := asked[key]
				if !known {
					answer = settled != nil && settled(key)
					asked[key] = answer
				}
				it.remove = answer
			}
			if !it.remove {
				it.to = moveTo[key]
			}
		}
	}
	cancelClashingMoves(files)

	type change struct {
		name   string
		text   string
		delete bool
	}
	var changes []change
	for _, f := range files {
		if !f.dirty() {
			continue
		}
		text, left, err := f.edit()
		if err != nil {
			return Tidied{}, err
		}
		changes = append(changes, change{f.name, text, left == 0})
	}
	var done Tidied
	for _, f := range files {
		for _, it := range f.items {
			if it.remove {
				done.Removed++
			} else if it.to != "" {
				done.Moved++
			}
		}
	}
	for _, c := range changes {
		path := filepath.Join(root, filepath.FromSlash(Dir), c.name)
		if c.delete {
			err = os.Remove(path)
		} else {
			err = rulefile.WriteAtomic(path, []byte(c.text))
		}
		if err != nil {
			return Tidied{}, err
		}
		done.Changed = append(done.Changed, Dir+"/"+c.name)
	}
	slices.Sort(done.Changed)
	return done, nil
}

// tidyFile is an adjustment file as Tidy reads it.
type tidyFile struct {
	name  string
	text  string
	seq   *yaml.Node // the list of entries, nil for a file without any
	items []*tidyItem
}

// tidyItem is one entry of a tidyFile and what is to become of it.
type tidyItem struct {
	adj  Adjustment
	node *yaml.Node // the mapping of the entry
	key  *yaml.Node // the value of its "key", nil if the mapping has none of its own
	// remove: the entry goes. to: it gets this key ("" for no change).
	remove bool
	to     string
}

func (it *tidyItem) finalKey() string {
	if it.to != "" {
		return it.to
	}
	return it.adj.Key
}

func parseTidyFile(e rulefile.Entry) (*tidyFile, error) {
	f := &tidyFile{name: e.Name, text: string(e.Data)}
	var doc yaml.Node
	if err := yaml.Unmarshal(e.Data, &doc); err != nil {
		return nil, fmt.Errorf("%s/%s: %w", Dir, e.Name, err)
	}
	if doc.Kind == 0 || len(doc.Content) == 0 || doc.Content[0].Tag == "!!null" {
		return f, nil
	}
	f.seq = doc.Content[0]
	if f.seq.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("%s/%s: line %d: not a list of entries", Dir, e.Name, f.seq.Line)
	}
	for i, n := range f.seq.Content {
		it := &tidyItem{node: n}
		if err := n.Decode(&it.adj); err != nil {
			return nil, fmt.Errorf("%s/%s: entry %d: %w", Dir, e.Name, i+1, err)
		}
		if n.Kind == yaml.MappingNode {
			for j := 0; j+1 < len(n.Content); j += 2 {
				if n.Content[j].Value == "key" {
					it.key = n.Content[j+1]
				}
			}
		}
		f.items = append(f.items, it)
	}
	return f, nil
}

// dirty reports whether anything is to be done to the file.
func (f *tidyFile) dirty() bool {
	for _, it := range f.items {
		if it.remove || it.to != "" {
			return true
		}
	}
	return false
}

// cancelClashingMoves drops the moves that would leave two entries, of any
// file, with one key: one to a key that an entry keeps, or that another move
// gives too (of two such moves, the later one is dropped).
func cancelClashingMoves(files []*tidyFile) {
	for {
		owners := map[string]int{}
		var movers []*tidyItem
		for _, f := range files {
			for _, it := range f.items {
				if it.remove {
					continue
				}
				owners[it.finalKey()]++
				if it.to != "" {
					movers = append(movers, it)
				}
			}
		}
		cancelled := false
		for i := len(movers) - 1; i >= 0 && !cancelled; i-- {
			if owners[movers[i].to] > 1 {
				movers[i].to, cancelled = "", true
			}
		}
		if !cancelled {
			return
		}
	}
}

// edit makes the changes in the text of the file and returns the new text and
// the number of entries it holds.
func (f *tidyFile) edit() (text string, entries int, err error) {
	cannot := func(format string, args ...any) error {
		return invalid("%s/%s: cannot tidy a file written like this (%s); edit it by hand", Dir, f.name, fmt.Sprintf(format, args...))
	}
	if f.seq.Style&yaml.FlowStyle != 0 {
		return "", 0, cannot("the list is a flow list, [...]")
	}
	lines := strings.SplitAfter(f.text, "\n")
	start := make([]int, len(f.items)) // the line of each entry, from 0
	for i, it := range f.items {
		if it.node.Kind != yaml.MappingNode || it.node.Style&yaml.FlowStyle != 0 {
			return "", 0, cannot("entry %d is a flow mapping, {...}", i+1)
		}
		start[i] = it.node.Line - 1
		if (i > 0 && start[i] <= start[i-1]) || start[i] >= len(lines) || !strings.HasPrefix(strings.TrimLeft(lines[start[i]], " "), "- ") {
			return "", 0, cannot("entry %d does not start on a line of its own, with - and a key", i+1)
		}
	}
	for _, line := range lines[start[0]:] {
		if line = strings.TrimRight(line, "\r\n"); line == "---" || line == "..." || strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "... ") {
			return "", 0, cannot("it has a document marker, %q", strings.Fields(line)[0])
		}
	}

	drop := make([]bool, len(lines))
	replace := map[int]string{}
	var want []Adjustment // the entries as they should read back
	for i, it := range f.items {
		if it.remove {
			dash := len(lines[start[i]]) - len(strings.TrimLeft(lines[start[i]], " "))
			end := len(lines)
			if i+1 < len(f.items) {
				end = start[i+1]
			}
			// The comments and blank lines after the entry belong to what
			// follows, and stay.
			for end-1 > start[i] && isGap(lines[end-1], dash) {
				end--
			}
			for l := start[i]; l < end; l++ {
				drop[l] = true
			}
			continue
		}
		a := it.adj
		if it.to != "" {
			line, err := replaceKey(lines, it, it.to)
			if err != nil {
				return "", 0, cannot("entry %d: %v", i+1, err)
			}
			replace[it.key.Line-1] = line
			a.Key = it.to
		}
		want = append(want, a)
	}

	var b strings.Builder
	for l, line := range lines {
		switch {
		case drop[l]:
		case replace[l] != "":
			b.WriteString(replace[l])
		default:
			b.WriteString(line)
		}
	}
	text = b.String()
	var back []Adjustment
	if err := yaml.Unmarshal([]byte(text), &back); err != nil {
		return "", 0, cannot("the result would not be readable: %v", err)
	}
	for i := range back {
		back[i].File = ""
	}
	if !slices.Equal(back, want) {
		return "", 0, cannot("the result would not read back as the entries meant")
	}
	return text, len(want), nil
}

// isGap reports whether line, after an entry whose "-" is at column dash, is
// blank or a comment that is not part of the entry.
func isGap(line string, dash int) bool {
	text := strings.TrimSpace(line)
	return text == "" || strings.HasPrefix(text, "#") && len(line)-len(strings.TrimLeft(line, " ")) <= dash
}

// replaceKey returns the line of the "key" value of it with that value
// replaced by the key to. Only the value is replaced: what follows it on the
// line, a comment say, stays.
func replaceKey(lines []string, it *tidyItem, to string) (string, error) {
	kn := it.key
	switch {
	case kn == nil:
		return "", fmt.Errorf("its key is not written in the entry")
	case kn.Kind != yaml.ScalarNode || kn.Anchor != "" || kn.Style&yaml.TaggedStyle != 0:
		return "", fmt.Errorf("its key is not a plain or quoted text")
	}
	line := lines[kn.Line-1]
	runes := []rune(line) // yaml counts the columns in characters
	if kn.Column < 1 || kn.Column > len(runes) {
		return "", fmt.Errorf("its key cannot be found on line %d", kn.Line)
	}
	at := len(string(runes[:kn.Column-1]))
	rest := line[at:]
	n := -1
	switch kn.Style {
	case 0:
		if strings.HasPrefix(rest, kn.Value) {
			n = len(kn.Value)
		}
	case yaml.DoubleQuotedStyle:
		n = quotedLen(rest, '"')
	case yaml.SingleQuotedStyle:
		n = quotedLen(rest, '\'')
	}
	if n < 0 {
		return "", fmt.Errorf("its key is not written on one line")
	}
	text, err := scalar(to)
	if err != nil {
		return "", err
	}
	return line[:at] + text + rest[n:], nil
}

// quotedLen is the length of the quoted text at the start of s, or -1 when it
// does not end on the line.
func quotedLen(s string, quote byte) int {
	if s == "" || s[0] != quote {
		return -1
	}
	for i := 1; i < len(s); i++ {
		switch {
		case quote == '"' && s[i] == '\\':
			i++
		case s[i] == quote && quote == '\'' && i+1 < len(s) && s[i+1] == '\'':
			i++
		case s[i] == quote:
			return i + 1
		}
	}
	return -1
}

// scalar writes s as a YAML text on one line: plain when the emitter can,
// double-quoted otherwise. Like encode it reads the result back, since the
// emitter gets some strings wrong.
func scalar(s string) (string, error) {
	for _, style := range []yaml.Style{0, yaml.DoubleQuotedStyle} {
		data, err := yaml.Marshal(&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s, Style: style})
		if err != nil {
			return "", err
		}
		text := strings.TrimSuffix(string(data), "\n")
		var back string
		if !strings.Contains(text, "\n") && yaml.Unmarshal([]byte(text), &back) == nil && back == s {
			return text, nil
		}
	}
	return "", fmt.Errorf("%q cannot be written as YAML on one line", s)
}
