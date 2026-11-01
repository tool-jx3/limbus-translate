// Package rulefile holds what the files that people commit to a directory of
// rules/ on main have in common: the .yaml files of rules/adjustments,
// rules/requeue and rules/rewrite. It reads the directory, decodes a file,
// checks the day-and-topic file name, writes a file without leaving half of
// it behind, and lints a directory.
//
// What the files say is the business of the packages that apply them.
package rulefile

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"gopkg.in/yaml.v3"
)

// Ext is the extension of the files that are read.
const Ext = ".yaml"

// IsFile reports whether a file of a rule directory with this name is read.
func IsFile(name string) bool { return strings.HasSuffix(name, Ext) }

// ID is the name of a file without its extension. A file that is applied once
// is told apart by its ID, so renaming it makes it a new one.
func ID(name string) string { return strings.TrimSuffix(name, Ext) }

// ValidateName checks the two parts of the name of a new file,
// <date>-<topic>.yaml: the date is YYYY-MM-DD, and the topic is made of
// letters (Chinese too), digits, "-" and "_" and does not start with "-".
func ValidateName(date, topic string) error {
	if t, err := time.Parse("2006-01-02", date); err != nil || t.Format("2006-01-02") != date {
		return fmt.Errorf("date %q is not YYYY-MM-DD", date)
	}
	ok := topic != "" && !strings.HasPrefix(topic, "-")
	for _, r := range topic {
		ok = ok && (unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_')
	}
	if !ok {
		return fmt.Errorf("topic %q must be made of letters, digits, - and _, and not start with -", topic)
	}
	return nil
}

// WriteAtomic writes data to path through a temporary file in the same
// directory, creating the directory, so that a crash cannot leave half of a
// file behind. The temporary file's name does not end in Ext, so no reader
// takes it for a rule file.
func WriteAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp*")
	if err != nil {
		return err
	}
	_, err = tmp.Write(data)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(tmp.Name(), 0o644)
	}
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

// Entry is one rule file as it was read.
type Entry struct {
	Name string // file name, such as 2026-11-05-honglu.yaml
	Data []byte
}

// ReadDir reads the rule files of dir (slash-separated, relative to root): the
// files directly in it whose name ends in Ext, in order of name. A root
// without dir has none.
func ReadDir(root, dir string) ([]Entry, error) {
	abs := filepath.Join(root, filepath.FromSlash(dir))
	entries, err := os.ReadDir(abs)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, e := range entries { // sorted by name
		if e.IsDir() || !IsFile(e.Name()) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(abs, e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, Entry{Name: e.Name(), Data: data})
	}
	return out, nil
}

// Decode reads the one YAML mapping in data. The value of every key that is a
// key of fields is decoded into the pointer there. It returns the keys the
// mapping has, in the order they are written, so that the caller can tell a
// key that fields lacks (a misspelled one) from one that is known, and a
// field that was left out from one that was written empty. A known key whose
// value is null (nothing after the colon, ~ or null) counts as left out: it is
// not returned and its pointer is not touched, since "replace:" is no more an
// answer than no line at all. An unknown key is returned whatever its value.
// An empty file has no keys. A file that is not a mapping, has more than one
// document, repeats a key or has a value of the wrong kind is an error that
// says which key.
func Decode(data []byte, fields map[string]any) (keys []string, err error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var doc yaml.Node
	if err := dec.Decode(&doc); errors.Is(err, io.EOF) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var extra yaml.Node
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("more than one YAML document in the file")
		}
		return nil, err
	}

	root := doc.Content[0] // yaml.Unmarshal wraps the mapping in a document
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("line %d: the file must be a mapping of keys to values", root.Line)
	}
	seen := map[string]bool{}
	for i := 0; i+1 < len(root.Content); i += 2 {
		name, value := root.Content[i], root.Content[i+1]
		if name.Kind != yaml.ScalarNode {
			return nil, fmt.Errorf("line %d: a key must be a plain word", name.Line)
		}
		key := name.Value
		if seen[key] {
			return nil, fmt.Errorf("line %d: %s is written twice", name.Line, key)
		}
		seen[key] = true
		into, known := fields[key]
		if known && value.Tag == "!!null" {
			continue
		}
		keys = append(keys, key)
		if known {
			if err := value.Decode(into); err != nil {
				return nil, fmt.Errorf("%s: %s", key, decodeMessage(err))
			}
		}
	}
	return keys, nil
}

// decodeMessage is err of a yaml decoder on one line.
func decodeMessage(err error) string {
	var te *yaml.TypeError
	if errors.As(err, &te) {
		return strings.Join(te.Errors, "; ")
	}
	return err.Error()
}

// Issue is one problem Lint found. File is relative to the repo root.
type Issue struct {
	File    string
	Message string
}

// Lint checks the rule files of dir (slash-separated, relative to root). It
// reports an entry that ReadDir would skip, since a person who names a file
// x.yml would otherwise find it silently ignored, and the messages that check
// returns for the content of each .yaml file. A root without dir has no
// issues.
func Lint(root, dir string, check func(data []byte) []string) []Issue {
	var issues []Issue
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(dir)))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return []Issue{{File: dir, Message: ioMessage(err)}}
	}
	for _, e := range entries {
		rel := dir + "/" + e.Name()
		switch {
		case !e.IsDir() && e.Name() == ".gitkeep":
		case e.IsDir() || !IsFile(e.Name()):
			issues = append(issues, Issue{File: rel, Message: fmt.Sprintf("not a rule file: only %s files directly in %s are read", Ext, dir)})
		default:
			data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
			if err != nil {
				issues = append(issues, Issue{File: rel, Message: ioMessage(err)})
				continue
			}
			for _, msg := range check(data) {
				issues = append(issues, Issue{File: rel, Message: msg})
			}
		}
	}
	return issues
}

// ioMessage is err without the absolute path os puts in it.
func ioMessage(err error) string {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err.Error()
	}
	return err.Error()
}
