// Package names builds the name tables that keep keyword and speaker names
// consistent: the keyword table ([ID] to its name) and the speaker table
// (model key to character name). They are read from the language pack's own
// current translation, so they reflect the names the pack already uses.
package names

import (
	"path"
	"slices"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/jsonx"
)

// keywordFilePrefixes are the files whose dataList holds keyword names.
var keywordFilePrefixes = []string{"BattleKeywords", "Bufs"}

// IsKeywordFile reports whether file (a relative path without language
// prefix) is one of the files whose dataList holds keyword names, the ones
// KeywordTable reads.
func IsKeywordFile(file string) bool {
	base := path.Base(file)
	for _, prefix := range keywordFilePrefixes {
		if strings.HasPrefix(base, prefix) {
			return true
		}
	}
	return false
}

// KeywordTable maps keyword ids to names, read from the dataList of the
// BattleKeywords* and Bufs* files in docs (file name to document). Files are
// read in name order, ignoring ".json", so the base file comes before its
// variants and BattleKeywords* before Bufs*. The first file to name an id
// wins and a later file only fills ids that have no name yet. Numeric ids are
// keyed by their literal. Files of other families are ignored.
func KeywordTable(docs map[string]*jsonx.Node) map[string]string {
	files := make([]string, 0, len(docs))
	for file := range docs {
		if IsKeywordFile(file) {
			files = append(files, file)
		}
	}
	slices.SortFunc(files, func(a, b string) int {
		return strings.Compare(strings.TrimSuffix(a, ".json"), strings.TrimSuffix(b, ".json"))
	})
	table := map[string]string{}
	for _, file := range files {
		addNames(table, docs[file])
	}
	return table
}

// SpeakerTable maps the ids of ScenarioModelCodes-AutoCreated's dataList (the
// model keys of story lines) to their names. When an id occurs twice the
// first name wins.
func SpeakerTable(doc *jsonx.Node) map[string]string {
	table := map[string]string{}
	addNames(table, doc)
	return table
}

// addNames adds the id/name pairs of doc's dataList to table for the ids it
// does not name yet. Entries without an id, or with an empty or non-string
// name, are skipped.
func addNames(table map[string]string, doc *jsonx.Node) {
	if doc == nil {
		return
	}
	list, ok := doc.Get("/dataList")
	if !ok || list.Kind != jsonx.Array {
		return
	}
	for _, entry := range list.Items {
		if entry.Kind != jsonx.Object {
			continue
		}
		id, ok := idOf(member(entry, "id"))
		if !ok {
			continue
		}
		name := member(entry, "name")
		if name == nil || name.Kind != jsonx.String || name.Str == "" {
			continue
		}
		if _, named := table[id]; !named {
			table[id] = name.Str
		}
	}
}

// member returns the first member of an object with the given key, or nil.
func member(obj *jsonx.Node, key string) *jsonx.Node {
	for _, m := range obj.Members {
		if m.Key == key {
			return m.Value
		}
	}
	return nil
}

// idOf returns the key of an id value: a string, or the literal of a number.
func idOf(n *jsonx.Node) (string, bool) {
	if n == nil {
		return "", false
	}
	switch n.Kind {
	case jsonx.String:
		return n.Str, n.Str != ""
	case jsonx.Number:
		return n.Lit, true
	}
	return "", false
}
