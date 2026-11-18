package rawdiff

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/pack"
)

// summarize writes files (pack file name, without the KR_ prefix, to its
// JSON) as a KR pack and summarizes it.
func summarize(t *testing.T, files map[string]string) *Summary {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		dir, base := filepath.Split(filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, dir, "KR_"+base), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p, err := pack.Open(root, "KR_")
	if err != nil {
		t.Fatal(err)
	}
	s, err := Summarize(p)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func compare(t *testing.T, from, to map[string]string) *Diff {
	t.Helper()
	return Compare(summarize(t, from), summarize(t, to), func(file string) string {
		if strings.HasPrefix(file, "Skills") {
			return "ally"
		}
		return ""
	})
}

func equal(t *testing.T, what string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		g, _ := json.Marshal(got)
		w, _ := json.Marshal(want)
		t.Errorf("%s:\n got %s\nwant %s", what, g, w)
	}
}

func TestCompareFilesAndGroups(t *testing.T) {
	d := compare(t,
		map[string]string{
			"Skills.json":       `{"dataList":[{"id":1,"name":"가"}]}`,
			"Old-1.json":        `{"dataList":[{"id":"a","desc":"나"},{"id":"b","desc":"다"}]}`,
			"StoryData/S1.json": `{"dataList":[{"id":0,"content":"라"}]}`,
		},
		map[string]string{
			"Skills.json":       `{"dataList":[{"id":1,"name":"가"}]}`,
			"Skills-a1.json":    `{"dataList":[{"id":2,"name":"마"},{"id":3,"name":"바"}]}`,
			"StoryData/S1.json": `{"dataList":[{"id":0,"content":"라"}]}`,
			"StoryData/S2.json": `{"dataList":[{"id":0,"content":"사"}]}`,
			"SynergyKeyword.json": `{"dataList":[{"id":"MAGICAL_GIRL","name":"마법소녀","on":true},
				{"id":"X","name":"아"}]}`,
		})
	equal(t, "files added", d.FilesAdded, []File{
		{File: "Skills-a1.json", Group: "Skills", Family: "ally", Entries: 2},
		{File: "StoryData/S2.json", Group: "StoryData", Entries: 1},
		{File: "SynergyKeyword.json", Group: "SynergyKeyword", Entries: 2},
	})
	equal(t, "files removed", d.FilesRemoved, []File{{File: "Old-1.json", Group: "Old", Entries: 2}})
	equal(t, "new groups", d.NewGroups, []Group{{Group: "SynergyKeyword", Files: 1, Keys: []Key{
		{Path: "dataList[].id", Kinds: []string{"string"}, Count: 2, File: "SynergyKeyword.json", Pointer: "/dataList/0/id"},
		{Path: "dataList[].name", Kinds: []string{"string"}, Count: 2, File: "SynergyKeyword.json", Pointer: "/dataList/0/name"},
		{Path: "dataList[].on", Kinds: []string{"bool"}, Count: 1, File: "SynergyKeyword.json", Pointer: "/dataList/0/on"},
	}}})
	equal(t, "gone groups", d.GoneGroups, []Group{{Group: "Old", Files: 1, Keys: []Key{
		{Path: "dataList[].desc", Kinds: []string{"string"}, Count: 2, File: "Old-1.json", Pointer: "/dataList/0/desc"},
		{Path: "dataList[].id", Kinds: []string{"string"}, Count: 2, File: "Old-1.json", Pointer: "/dataList/0/id"},
	}}})
	// The keys of a new or a gone group are its shape, not a change of keys.
	equal(t, "keys", d.Keys, []GroupKeys{})
	equal(t, "entries", d.Entries, []FileEntries{})
	equal(t, "counts", d.Counts, Counts{FilesAdded: 3, FilesRemoved: 1, NewGroups: 1, GoneGroups: 1})
}

func TestCompareKeysOfAGroupInBoth(t *testing.T) {
	d := compare(t,
		map[string]string{
			"Bufs.json":    `{"dataList":[{"id":"A","name":"가","desc":"나","undefined":"-"}]}`,
			"Bufs-a1.json": `{"dataList":[{"id":"B","name":"다","desc":"라"}]}`,
			"Skills.json":  `{"dataList":[{"id":1,"levelList":[{"level":1,"desc":"마"}]}]}`,
		},
		map[string]string{
			"Bufs.json":    `{"dataList":[{"id":"A","name":"가","desc":"나"}]}`,
			"Bufs-a1.json": `{"dataList":[{"id":"B","name":"다","desc":"라","excludeAutoLink":true},{"id":"C","name":"바","desc":"사","excludeAutoLink":false}]}`,
			"Skills.json":  `{"dataList":[{"id":1,"levelList":[{"level":1,"desc":"마","coinlist":[{"coindescs":[{"desc":"아","summary":null},{"desc":"자","summary":"차"}]}]}]}]}`,
		})
	equal(t, "keys", d.Keys, []GroupKeys{
		{Group: "Bufs",
			Added:   []Key{{Path: "dataList[].excludeAutoLink", Kinds: []string{"bool"}, Count: 2, File: "Bufs-a1.json", Pointer: "/dataList/0/excludeAutoLink"}},
			Removed: []Key{{Path: "dataList[].undefined", Kinds: []string{"string"}, Count: 1, File: "Bufs.json", Pointer: "/dataList/0/undefined"}}},
		{Group: "Skills",
			Added: []Key{
				{Path: "dataList[].levelList[].coinlist[].coindescs[].desc", Kinds: []string{"string"}, Count: 2, File: "Skills.json", Pointer: "/dataList/0/levelList/0/coinlist/0/coindescs/0/desc"},
				{Path: "dataList[].levelList[].coinlist[].coindescs[].summary", Kinds: []string{"null", "string"}, Count: 2, File: "Skills.json", Pointer: "/dataList/0/levelList/0/coinlist/0/coindescs/0/summary"},
			},
			Removed: []Key{}},
	})
	if d.Counts.KeysAdded != 3 || d.Counts.KeysRemoved != 1 {
		t.Errorf("counts = %+v, want 3 keys added and 1 removed", d.Counts)
	}
	equal(t, "new groups", d.NewGroups, []Group{})
}

func TestCompareEntriesOfAFileInBoth(t *testing.T) {
	d := compare(t,
		map[string]string{
			"Skills.json":       `{"dataList":[{"id":1,"name":"가"},{"id":2,"name":"나"},{"id":3,"name":"다"}]}`,
			"UI.json":           `{"dataList":[{"id":"a","content":"라"}]}`,
			"Same.json":         `{"dataList":[{"id":"a","content":"마"}]}`,
			"NoID.json":         `{"dataList":[{"content":"바"}]}`,
			"StoryData/S1.json": `{"dataList":[{"id":0,"content":"사"}]}`,
		},
		map[string]string{
			"Skills.json":       `{"dataList":[{"id":1,"name":"가"},{"id":3,"name":"다"},{"id":5,"name":"아"},{"id":4,"name":"자"}]}`,
			"UI.json":           `{"dataList":[{"id":"a","content":"라"},{"id":"b","content":"차"}]}`,
			"Same.json":         `{"dataList":[{"id":"a","content":"the text changed, the entries did not"}]}`,
			"NoID.json":         `{"dataList":[{"content":"바"},{"content":"카"}]}`,
			"StoryData/S1.json": `{"dataList":[{"id":0,"content":"사"}]}`,
			"StoryData/S2.json": `{"dataList":[{"id":0,"content":"타"}]}`,
		})
	equal(t, "entries", d.Entries, []FileEntries{
		{File: "NoID.json", Group: "NoID", Before: 1, After: 2, Added: []string{}, Removed: []string{}},
		{File: "Skills.json", Group: "Skills", Family: "ally", Before: 3, After: 4, Added: []string{"5", "4"}, Removed: []string{"2"}},
		{File: "UI.json", Group: "UI", Before: 1, After: 2, Added: []string{"b"}, Removed: []string{}},
	})
	// The entries of an added file are counted with the file, not here.
	if d.Counts.EntriesAdded != 3 || d.Counts.EntriesRemoved != 1 {
		t.Errorf("counts = %+v, want 3 entries added and 1 removed", d.Counts)
	}
}

func TestCompareTokensFirstSeenInThePack(t *testing.T) {
	d := compare(t,
		map[string]string{
			"Skills.json": `{"dataList":[{"id":1,"desc":"[Burn] {0} <color=#fff>가</color> <style=\"highlight\">나</style> {amount} [Gone]"}]}`,
			"Bufs.json":   `{"dataList":[{"id":"Burn","name":"화상"},{"id":"Gone","name":"다"}]}`,
		},
		map[string]string{
			"Skills.json": `{"dataList":[{"id":1,"desc":"[Burn] {0} {1} <color=#fff>가</color> <style=\"highlight\">나</style> {amount}"},
				{"id":2,"desc":"[Storm] [Storm] {0:F1} {reward} <MARK=#000>라</mark> <style=\"upgradeHighlight\">마</style> <sprite name=\"x\"> <3> [Typo]"}]}`,
			"Bufs.json":           `{"dataList":[{"id":"Burn","name":"화상"},{"id":"Storm","name":"[Storm]"},{"id":"Gone","name":"다"}]}`,
			"BattleKeywords.json": `{"dataList":[{"id":"Storm","name":"바"}]}`,
		})
	equal(t, "tokens", d.Tokens, []Token{
		{Kind: "bracket", Name: "Storm", Change: "added", Count: 3, File: "Bufs.json", Pointer: "/dataList/1/name", DefinedIn: []string{"BattleKeywords", "Bufs"}},
		{Kind: "bracket", Name: "Typo", Change: "added", Count: 1, File: "Skills.json", Pointer: "/dataList/1/desc", DefinedIn: []string{}},
		{Kind: "bracket", Name: "Gone", Change: "removed", Count: 1, File: "Skills.json", Pointer: "/dataList/0/desc", DefinedIn: []string{"Bufs"}},
		{Kind: "placeholder", Name: "0:F1", Change: "added", Count: 1, File: "Skills.json", Pointer: "/dataList/1/desc", DefinedIn: []string{}},
		{Kind: "placeholder", Name: "reward", Change: "added", Count: 1, File: "Skills.json", Pointer: "/dataList/1/desc", DefinedIn: []string{}},
		{Kind: "style", Name: "upgradeHighlight", Change: "added", Count: 1, File: "Skills.json", Pointer: "/dataList/1/desc", DefinedIn: []string{}},
		{Kind: "tag", Name: "mark", Change: "added", Count: 2, File: "Skills.json", Pointer: "/dataList/1/desc", DefinedIn: []string{}},
		{Kind: "tag", Name: "sprite", Change: "added", Count: 1, File: "Skills.json", Pointer: "/dataList/1/desc", DefinedIn: []string{}},
	})
	if d.Counts.TokensAdded != 7 || d.Counts.TokensRemoved != 1 {
		t.Errorf("counts = %+v, want 7 tokens added and 1 removed", d.Counts)
	}
}

func TestCompareTheSameSnapshot(t *testing.T) {
	files := map[string]string{"Skills.json": `{"dataList":[{"id":1,"desc":"[Burn] {0}"}]}`}
	d := compare(t, files, files)
	if !d.Counts.Zero() {
		t.Errorf("counts = %+v, want none", d.Counts)
	}
	got, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"lang":"","from":"","to":"","counts":{"files_added":0,"files_removed":0,"new_groups":0,"gone_groups":0,"keys_added":0,"keys_removed":0,"entries_added":0,"entries_removed":0,"tokens_added":0,"tokens_removed":0},` +
		`"new_groups":[],"gone_groups":[],"files_added":[],"files_removed":[],"keys":[],"entries":[],"tokens":[]}`
	if string(got) != want {
		t.Errorf("JSON:\n got %s\nwant %s", got, want)
	}
}

func TestSummarizeNamesTheFileItCannotRead(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "KR_Bad.json"), []byte(`{"dataList":[`), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := pack.Open(root, "KR_")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Summarize(p); err == nil || !strings.Contains(err.Error(), "Bad.json") {
		t.Errorf("err = %v, want one that names Bad.json", err)
	}
}

func TestSummarizeAFileWithoutADataList(t *testing.T) {
	d := compare(t,
		map[string]string{"Odd.json": `{"version":1}`},
		map[string]string{"Odd.json": `{"version":2,"note":"가"}`, "List.json": `[{"id":1}]`})
	equal(t, "files added", d.FilesAdded, []File{{File: "List.json", Group: "List", Entries: 0}})
	equal(t, "keys", d.Keys, []GroupKeys{{Group: "Odd",
		Added:   []Key{{Path: "note", Kinds: []string{"string"}, Count: 1, File: "Odd.json", Pointer: "/note"}},
		Removed: []Key{}}})
	equal(t, "new groups", d.NewGroups, []Group{{Group: "List", Files: 1, Keys: []Key{
		{Path: "[].id", Kinds: []string{"number"}, Count: 1, File: "List.json", Pointer: "/0/id"},
	}}})
}

// Against lists what two packs changed differently between the same two
// snapshots: here JP against KR.
func TestAgainstListsWhatOnlyOneDiffHas(t *testing.T) {
	krFrom := map[string]string{
		"Skills.json": `{"dataList":[{"id":1,"desc":"가"}]}`,
		"Bufs.json":   `{"dataList":[{"id":"Burn","name":"화상"}]}`,
		"UI.json":     `{"dataList":[{"id":"a","content":"나"}]}`,
	}
	krTo := map[string]string{
		"Skills.json":    `{"dataList":[{"id":1,"desc":"가"},{"id":2,"desc":"[Storm] {amount}"}]}`,
		"Skills-a1.json": `{"dataList":[{"id":3,"desc":"다"}]}`,
		"Bufs.json":      `{"dataList":[{"id":"Burn","name":"화상","excludeAutoLink":true},{"id":"Storm","name":"폭풍"}]}`,
		"UI.json":        `{"dataList":[{"id":"a","content":"나"},{"id":"b","content":"라"},{"id":"c","content":"마"}]}`,
		"New.json":       `{"dataList":[{"id":1,"name":"바"}]}`,
	}
	// JP had Skills-a1.json and the key a snapshot earlier, got another file
	// and another tag, and has not got one of the entries of UI.json yet.
	jpFrom := map[string]string{
		"Skills.json":    `{"dataList":[{"id":1,"desc":"あ"}]}`,
		"Skills-a1.json": `{"dataList":[{"id":3,"desc":"い"}]}`,
		"Bufs.json":      `{"dataList":[{"id":"Burn","name":"火傷","excludeAutoLink":true}]}`,
		"UI.json":        `{"dataList":[{"id":"a","content":"う"}]}`,
	}
	jpTo := map[string]string{
		"Skills.json":    `{"dataList":[{"id":1,"desc":"あ"},{"id":2,"desc":"[Storm] {amount} <pause>"}]}`,
		"Skills-a1.json": `{"dataList":[{"id":3,"desc":"い"}]}`,
		"Bufs.json":      `{"dataList":[{"id":"Burn","name":"火傷","excludeAutoLink":true},{"id":"Storm","name":"嵐"}]}`,
		"UI.json":        `{"dataList":[{"id":"a","content":"う"},{"id":"b","content":"え"},{"id":"z","content":"お"}]}`,
		"New.json":       `{"dataList":[{"id":1,"name":"か"}]}`,
		"Extra.json":     `{"dataList":[{"id":1,"name":"き"}]}`,
	}
	kr, jp := compare(t, krFrom, krTo), compare(t, jpFrom, jpTo)
	equal(t, "JP against KR", Against(jp, kr), Versus{
		OnlyHere: []Item{
			{Kind: ItemGroup, Change: Added, Name: "Extra"},
			{Kind: ItemFile, Change: Added, Name: "Extra.json"},
			{Kind: ItemEntries, Change: Added, Name: "UI.json", Count: 1},
			{Kind: ItemToken, Change: Added, Name: "<pause>"},
		},
		OnlyOther: []Item{
			{Kind: ItemFile, Change: Added, Name: "Skills-a1.json"},
			{Kind: ItemKey, Change: Added, Name: "Bufs dataList[].excludeAutoLink"},
			{Kind: ItemEntries, Change: Added, Name: "UI.json", Count: 1},
		},
	})
	// A diff differs in nothing from itself, and the lists are not nil.
	equal(t, "KR against KR", Against(kr, kr), Versus{OnlyHere: []Item{}, OnlyOther: []Item{}})
}

func TestTokenTextIsTheTokenAsTheTextWritesIt(t *testing.T) {
	for _, c := range []struct{ kind, name, want string }{
		{TokenBracket, "Storm", "[Storm]"},
		{TokenPlaceholder, "0:F1", "{0:F1}"},
		{TokenStyle, "highlight", `<style="highlight">`},
		{TokenTag, "mark", "<mark>"},
	} {
		if got := (Token{Kind: c.kind, Name: c.name}).Text(); got != c.want {
			t.Errorf("Text of %s %s = %q, want %q", c.kind, c.name, got, c.want)
		}
	}
}
