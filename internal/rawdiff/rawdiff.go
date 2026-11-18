// Package rawdiff compares two snapshots of the KR pack as data: the files,
// the kinds of file (groups), the keys of their JSON, the entries and the
// tokens of the text that one has and the other has not. What the text of an
// entry says is not compared: that is the update record (internal/history).
package rawdiff

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/jsonx"
	"github.com/tool-jx3/limbus-translate/internal/pack"
)

// Diff is what the snapshot To has that From has not, and the reverse. No
// list is nil, and each is in a fixed order: files and groups by name, keys by
// path, tokens by kind, then the added before the removed, then by name.
type Diff struct {
	From   string `json:"from"` // the tags, which the caller sets
	To     string `json:"to"`
	Counts Counts `json:"counts"`
	// NewGroups are the groups (fields.Group) no file of From belongs to, each
	// with every key of its files; GoneGroups the reverse.
	NewGroups    []Group `json:"new_groups"`
	GoneGroups   []Group `json:"gone_groups"`
	FilesAdded   []File  `json:"files_added"`
	FilesRemoved []File  `json:"files_removed"`
	// Keys is, for each group that both snapshots have, the keys that only one
	// has.
	Keys []GroupKeys `json:"keys"`
	// Entries is the files that both snapshots have whose entries differ in
	// number or in their ids.
	Entries []FileEntries `json:"entries"`
	// Tokens are the tokens of the text that the whole pack of only one
	// snapshot has.
	Tokens []Token `json:"tokens"`
}

// Counts is the size of each list of a Diff. Keys, entries and tokens are
// counted one by one, not by group or file.
type Counts struct {
	FilesAdded     int `json:"files_added"`
	FilesRemoved   int `json:"files_removed"`
	NewGroups      int `json:"new_groups"`
	GoneGroups     int `json:"gone_groups"`
	KeysAdded      int `json:"keys_added"`
	KeysRemoved    int `json:"keys_removed"`
	EntriesAdded   int `json:"entries_added"` // ids; the entries of an added file are not among them
	EntriesRemoved int `json:"entries_removed"`
	TokensAdded    int `json:"tokens_added"`
	TokensRemoved  int `json:"tokens_removed"`
}

// Zero reports whether nothing differs.
func (c Counts) Zero() bool { return c == Counts{} }

// File is a file that only one snapshot has.
type File struct {
	File    string `json:"file"` // the pack file name (pack.Pack.Files)
	Group   string `json:"group"`
	Family  string `json:"family,omitempty"` // of the routing rules; "" when no family takes the file
	Entries int    `json:"entries"`          // the elements of its dataList
}

// Group is a group that only one snapshot has.
type Group struct {
	Group string `json:"group"`
	Files int    `json:"files"`
	Keys  []Key  `json:"keys"`
}

// Key is a key of the JSON of a group, with the keys that lead to it: a value
// that is no object and no array. An array adds "[]" to the path, so
// "dataList[].levelList[].desc" is the desc of every level of every entry.
type Key struct {
	Path  string   `json:"path"`
	Kinds []string `json:"kinds"` // the kinds of its values, sorted: "bool", "null", "number", "string"
	Count int      `json:"count"` // how many values the group has under it
	// File and Pointer are where the first of them is.
	File    string `json:"file"`
	Pointer string `json:"pointer"`
}

// GroupKeys is the keys of a group that only one snapshot has.
type GroupKeys struct {
	Group   string `json:"group"`
	Added   []Key  `json:"added"`
	Removed []Key  `json:"removed"`
}

// FileEntries is how the entries (the elements of dataList) of a file differ.
// Added and Removed are ids, in the order of the file that has them; an entry
// without an id only shows in Before and After.
type FileEntries struct {
	File    string   `json:"file"`
	Group   string   `json:"group"`
	Family  string   `json:"family,omitempty"`
	Before  int      `json:"before"`
	After   int      `json:"after"`
	Added   []string `json:"added"`
	Removed []string `json:"removed"`
}

// The kinds of a Token.
const (
	TokenBracket     = "bracket"     // [Name]
	TokenPlaceholder = "placeholder" // {name}; one that is only a number, such as {0}, is not counted
	TokenStyle       = "style"       // the name of <style="name">
	TokenTag         = "tag"         // the name of <name>, <name=…> or </name>, in lower case
)

// The values of Token.Change.
const (
	Added   = "added"
	Removed = "removed"
)

// Token is a token of the text that one snapshot has somewhere in the pack
// and the other nowhere.
type Token struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Change string `json:"change"`
	Count  int    `json:"count"` // how often the snapshot that has it has it
	// File and Pointer are the first text with it.
	File    string `json:"file"`
	Pointer string `json:"pointer"`
	// DefinedIn is, for a bracket, the groups with an entry whose id is the
	// name, sorted; empty when there is none, and for the other kinds.
	DefinedIn []string `json:"defined_in"`
}

var (
	bracketRe     = regexp.MustCompile(`\[([A-Za-z][A-Za-z0-9_]*)\]`)
	placeholderRe = regexp.MustCompile(`\{([^{}\n]+)\}`)
	styleRe       = regexp.MustCompile(`(?i)<style="?([^">]+)"?>`)
	tagRe         = regexp.MustCompile(`</?([A-Za-z][A-Za-z0-9-]*)(?:=[^>]*)?(?: [^>]*)?>`)
	// nameRe matches an id that a bracket could name.
	nameRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
)

// Summary is what Compare needs of one snapshot.
type Summary struct {
	files  map[string]*fileSum
	groups map[string]*groupSum
	tokens map[tokenID]*seen
	// defined is, by an id a bracket could name, the groups with an entry of
	// that id.
	defined map[string]map[string]bool
}

type fileSum struct {
	group   string
	entries int
	ids     []string // of the entries that have one, in document order
}

type groupSum struct {
	files int
	keys  map[string]*keySum
}

type keySum struct {
	kinds map[string]bool
	seen
}

// seen counts the places of something and keeps the first.
type seen struct {
	count         int
	file, pointer string
}

func (s *seen) at(file, pointer string) {
	if s.count == 0 {
		s.file, s.pointer = file, pointer
	}
	s.count++
}

type tokenID struct{ kind, name string }

// Summarize reads every file of the pack, in the order of Files.
func Summarize(p *pack.Pack) (*Summary, error) {
	s := &Summary{
		files:   map[string]*fileSum{},
		groups:  map[string]*groupSum{},
		tokens:  map[tokenID]*seen{},
		defined: map[string]map[string]bool{},
	}
	for _, file := range p.Files() {
		doc, err := p.Doc(file)
		if err != nil {
			return nil, fmt.Errorf("rawdiff: %s: %w", file, err)
		}
		group := fields.Group(file)
		g := s.groups[group]
		if g == nil {
			g = &groupSum{keys: map[string]*keySum{}}
			s.groups[group] = g
		}
		g.files++
		f := &fileSum{group: group}
		s.files[file] = f
		for _, item := range dataList(doc) {
			f.entries++
			id, ok := idOf(item)
			if !ok {
				continue
			}
			f.ids = append(f.ids, id)
			if nameRe.MatchString(id) {
				if s.defined[id] == nil {
					s.defined[id] = map[string]bool{}
				}
				s.defined[id][group] = true
			}
		}
		s.walk(file, g, doc, "", "")
	}
	return s, nil
}

// walk records the keys and the tokens under n, which is at pointer in file
// and has path as its key.
func (s *Summary) walk(file string, g *groupSum, n *jsonx.Node, path, pointer string) {
	switch n.Kind {
	case jsonx.Object:
		for _, m := range n.Members {
			sub := m.Key
			if path != "" {
				sub = path + "." + m.Key
			}
			s.walk(file, g, m.Value, sub, pointer+"/"+jsonx.EscapeToken(m.Key))
		}
		return
	case jsonx.Array:
		for i, item := range n.Items {
			s.walk(file, g, item, path+"[]", pointer+"/"+strconv.Itoa(i))
		}
		return
	}
	k := g.keys[path]
	if k == nil {
		k = &keySum{kinds: map[string]bool{}}
		g.keys[path] = k
	}
	k.kinds[kindName(n.Kind)] = true
	k.at(file, pointer)
	if n.Kind != jsonx.String {
		return
	}
	for kind, re := range map[string]*regexp.Regexp{TokenBracket: bracketRe, TokenPlaceholder: placeholderRe, TokenStyle: styleRe, TokenTag: tagRe} {
		for _, m := range re.FindAllStringSubmatch(n.Str, -1) {
			name := m[1]
			switch kind {
			case TokenTag:
				name = strings.ToLower(name)
			case TokenPlaceholder:
				if _, err := strconv.Atoi(name); err == nil {
					continue
				}
			}
			id := tokenID{kind, name}
			if s.tokens[id] == nil {
				s.tokens[id] = &seen{}
			}
			s.tokens[id].at(file, pointer)
		}
	}
}

func kindName(k jsonx.Kind) string {
	switch k {
	case jsonx.String:
		return "string"
	case jsonx.Number:
		return "number"
	case jsonx.Bool:
		return "bool"
	}
	return "null"
}

// dataList is the elements of the dataList of a file; none when the file has
// no such array.
func dataList(doc *jsonx.Node) []*jsonx.Node {
	if doc.Kind != jsonx.Object {
		return nil
	}
	for _, m := range doc.Members {
		if m.Key == "dataList" && m.Value.Kind == jsonx.Array {
			return m.Value.Items
		}
	}
	return nil
}

// idOf is the id of an entry as text: a string as it is, a number as it is
// written.
func idOf(item *jsonx.Node) (string, bool) {
	if item.Kind != jsonx.Object {
		return "", false
	}
	for _, m := range item.Members {
		if m.Key != "id" {
			continue
		}
		switch m.Value.Kind {
		case jsonx.String:
			return m.Value.Str, true
		case jsonx.Number:
			return m.Value.Lit, true
		}
		return "", false
	}
	return "", false
}

// Compare lists what to has that from has not, and the reverse. familyOf
// gives the family of a file ("" for none); nil leaves every family empty.
func Compare(from, to *Summary, familyOf func(file string) string) *Diff {
	if familyOf == nil {
		familyOf = func(string) string { return "" }
	}
	d := &Diff{
		NewGroups: onlyGroups(to, from), GoneGroups: onlyGroups(from, to),
		FilesAdded: onlyFiles(to, from, familyOf), FilesRemoved: onlyFiles(from, to, familyOf),
		Keys: []GroupKeys{}, Entries: []FileEntries{}, Tokens: []Token{},
	}
	for _, group := range slices.Sorted(maps.Keys(to.groups)) {
		old, ok := from.groups[group]
		if !ok {
			continue
		}
		gk := GroupKeys{Group: group, Added: onlyKeys(to.groups[group], old), Removed: onlyKeys(old, to.groups[group])}
		if len(gk.Added)+len(gk.Removed) == 0 {
			continue
		}
		d.Keys = append(d.Keys, gk)
		d.Counts.KeysAdded += len(gk.Added)
		d.Counts.KeysRemoved += len(gk.Removed)
	}
	for _, file := range slices.Sorted(maps.Keys(to.files)) {
		old, ok := from.files[file]
		if !ok {
			continue
		}
		cur := to.files[file]
		fe := FileEntries{File: file, Group: cur.group, Family: familyOf(file), Before: old.entries, After: cur.entries,
			Added: onlyIDs(cur.ids, old.ids), Removed: onlyIDs(old.ids, cur.ids)}
		if fe.Before == fe.After && len(fe.Added)+len(fe.Removed) == 0 {
			continue
		}
		d.Entries = append(d.Entries, fe)
		d.Counts.EntriesAdded += len(fe.Added)
		d.Counts.EntriesRemoved += len(fe.Removed)
	}
	d.Tokens = append(append(d.Tokens, onlyTokens(to, from, Added)...), onlyTokens(from, to, Removed)...)
	slices.SortStableFunc(d.Tokens, func(a, b Token) int { return strings.Compare(a.Kind, b.Kind) })
	for _, t := range d.Tokens {
		if t.Change == Added {
			d.Counts.TokensAdded++
		} else {
			d.Counts.TokensRemoved++
		}
	}
	d.Counts.FilesAdded, d.Counts.FilesRemoved = len(d.FilesAdded), len(d.FilesRemoved)
	d.Counts.NewGroups, d.Counts.GoneGroups = len(d.NewGroups), len(d.GoneGroups)
	return d
}

// onlyGroups is the groups of a that b has not.
func onlyGroups(a, b *Summary) []Group {
	out := []Group{}
	for _, group := range slices.Sorted(maps.Keys(a.groups)) {
		if _, ok := b.groups[group]; ok {
			continue
		}
		g := a.groups[group]
		out = append(out, Group{Group: group, Files: g.files, Keys: onlyKeys(g, &groupSum{})})
	}
	return out
}

// onlyFiles is the files of a that b has not.
func onlyFiles(a, b *Summary, familyOf func(string) string) []File {
	out := []File{}
	for _, file := range slices.Sorted(maps.Keys(a.files)) {
		if _, ok := b.files[file]; ok {
			continue
		}
		f := a.files[file]
		out = append(out, File{File: file, Group: f.group, Family: familyOf(file), Entries: f.entries})
	}
	return out
}

// onlyKeys is the keys of a that b has not.
func onlyKeys(a, b *groupSum) []Key {
	out := []Key{}
	for _, path := range slices.Sorted(maps.Keys(a.keys)) {
		if _, ok := b.keys[path]; ok {
			continue
		}
		k := a.keys[path]
		out = append(out, Key{Path: path, Kinds: slices.Sorted(maps.Keys(k.kinds)), Count: k.count, File: k.file, Pointer: k.pointer})
	}
	return out
}

// onlyIDs is the ids of a that are not in b, each once, in the order of a.
func onlyIDs(a, b []string) []string {
	skip := make(map[string]bool, len(b))
	for _, id := range b {
		skip[id] = true
	}
	out := []string{}
	for _, id := range a {
		if !skip[id] {
			skip[id] = true
			out = append(out, id)
		}
	}
	return out
}

// onlyTokens is the tokens of a that b has not, by kind and name.
func onlyTokens(a, b *Summary, change string) []Token {
	var out []Token
	for id, at := range a.tokens {
		if _, ok := b.tokens[id]; ok {
			continue
		}
		t := Token{Kind: id.kind, Name: id.name, Change: change, Count: at.count, File: at.file, Pointer: at.pointer, DefinedIn: []string{}}
		if id.kind == TokenBracket {
			t.DefinedIn = append(t.DefinedIn, slices.Sorted(maps.Keys(a.defined[id.name]))...)
		}
		out = append(out, t)
	}
	slices.SortFunc(out, func(x, y Token) int {
		if c := strings.Compare(x.Kind, y.Kind); c != 0 {
			return c
		}
		return strings.Compare(x.Name, y.Name)
	})
	return out
}
