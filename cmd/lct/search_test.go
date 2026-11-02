package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/store"
)

// The KR files of searchRepo, and the units they hold:
//
//	S1001  #0 honglu   claude  "經理，您好"        (the one every filter of TestSearchFilters lets through)
//	       #1 faust    human   "經理，來了啊"
//	       #2 honglu   claude  "茶涼了呢"
//	S1002  #0 honglu   claude  "您還好嗎"
//	S1003  #0 엑스트라 (no record)
//	E002   #0 honglu   claude  "經理"
//	PersonalityVoiceDlg/Voice_HongLu_10601  #0 honglu (by the file name)  claude  "經理，您好"
const (
	searchS1001 = `{"dataList":[
 {"id":1,"model":"홍루","content":"매니저님, 안녕하세요"},
 {"id":2,"model":"파우스트","content":"매니저, 왔나"},
 {"id":3,"model":"홍루","content":"차가 식었네요"}
]}`
	searchS1002 = `{"dataList":[{"id":1,"model":"홍루","content":"괜찮으세요?"}]}`
	searchS1003 = `{"dataList":[{"id":1,"model":"엑스트라","content":"매니저?"}]}`
	searchE002  = `{"dataList":[{"id":1,"model":"홍루","content":"매니저님"}]}`
	searchVoice = `{"dataList":[{"id":"a","model":"낯선이","content":"매니저님 안녕"}]}`
)

const (
	searchKeyS1001a = "StoryData/S1001.json|/dataList/0/content"
	searchKeyS1001b = "StoryData/S1001.json|/dataList/1/content"
	searchKeyS1001c = "StoryData/S1001.json|/dataList/2/content"
	searchKeyS1002  = "StoryData/S1002.json|/dataList/0/content"
	searchKeyS1003  = "StoryData/S1003.json|/dataList/0/content"
	searchKeyE002   = "StoryData/E002.json|/dataList/0/content"
	searchKeyVoice  = "PersonalityVoiceDlg/Voice_HongLu_10601.json|/dataList/0/content"
)

// searchCopies are the units of searchRepo that hold identifiers, which the
// plan records as copies: the model of every line, and the string id of the
// voice line.
var searchCopies = map[string][]string{
	"StoryData/S1001.json":                        {"/dataList/0/model", "/dataList/1/model", "/dataList/2/model"},
	"StoryData/S1002.json":                        {"/dataList/0/model"},
	"StoryData/S1003.json":                        {"/dataList/0/model"},
	"StoryData/E002.json":                         {"/dataList/0/model"},
	"PersonalityVoiceDlg/Voice_HongLu_10601.json": {"/dataList/0/id", "/dataList/0/model"},
}

// searchRepo is batchRepo with more KR files and a record for most of their
// units, and the repo as the working directory.
func searchRepo(t *testing.T) string {
	t.Helper()
	root := adjustRepo(t)
	rawWT := filepath.Join(root, ".work", "raw", "kr")
	for file, text := range map[string]string{
		"StoryData/KR_S1001.json":                        searchS1001,
		"StoryData/KR_S1002.json":                        searchS1002,
		"StoryData/KR_S1003.json":                        searchS1003,
		"StoryData/KR_E002.json":                         searchE002,
		"PersonalityVoiceDlg/KR_Voice_HongLu_10601.json": searchVoice,
	} {
		writeTestFile(t, rawWT, file, text)
	}
	rec := func(key, zh, origin string) store.Record {
		return store.Record{Key: key, KRSha: "00", ZH: zh, Origin: origin, Since: "raw-20261029"}
	}
	recs := map[string]store.Record{}
	for file, pointers := range searchCopies {
		for _, p := range pointers {
			recs[file+"|"+p] = rec(file+"|"+p, "", store.OriginCopy)
		}
	}
	st := store.Open(filepath.Join(root, ".work", "tw"))
	for k, v := range map[string]store.Record{
		searchKeyS1001a: rec(searchKeyS1001a, "經理，您好", store.OriginClaude),
		searchKeyS1001b: rec(searchKeyS1001b, "經理，來了啊", store.OriginHuman),
		searchKeyS1001c: rec(searchKeyS1001c, "茶涼了呢", store.OriginClaude),
		searchKeyS1002:  rec(searchKeyS1002, "您還好嗎", store.OriginClaude),
		searchKeyE002:   rec(searchKeyE002, "經理", store.OriginClaude),
		searchKeyVoice:  rec(searchKeyVoice, "經理，您好", store.OriginClaude),
		// A unit of the old KR: no longer in the snapshot, so never a result.
		"StoryData/S1001.json|/dataList/9/content": rec("StoryData/S1001.json|/dataList/9/content", "經理已離開", store.OriginClaude),
	} {
		recs[k] = v
	}
	if err := st.SaveRecords(recs); err != nil {
		t.Fatal(err)
	}
	return root
}

// searchKeys runs "lct search --json args" and returns the keys of the
// result, in order.
func searchKeys(t *testing.T, args ...string) []string {
	t.Helper()
	var hits []searchHit
	mustLct(t, &hits, append([]string{"search"}, args...)...)
	keys := make([]string, len(hits))
	for i, h := range hits {
		keys[i] = h.Key
	}
	return keys
}

func TestSearchFilters(t *testing.T) {
	searchRepo(t)

	// Each of the three conditions leaves out a unit the other two let through:
	// S1001 #1 is human, S1001 #2 has no 經理, E002 is not a StoryData/S1* file.
	got := searchKeys(t, "--file", "StoryData/S1*", "--origin", "claude", "--text", "經理")
	if want := []string{searchKeyS1001a}; !slices.Equal(got, want) {
		t.Errorf("--file --origin --text = %v, want %v", got, want)
	}

	t.Run("file", func(t *testing.T) {
		got := searchKeys(t, "--file", "StoryData/S1*")
		want := []string{searchKeyS1001a, searchKeyS1001b, searchKeyS1001c, searchKeyS1002, searchKeyS1003}
		if !slices.Equal(got, want) {
			t.Errorf("--file StoryData/S1* = %v, want %v", got, want)
		}
		// Patterns are repeatable and a "/**" pattern takes a directory.
		got = searchKeys(t, "--file", "StoryData/E*", "--file", "PersonalityVoiceDlg/**")
		want = []string{searchKeyVoice, searchKeyE002}
		if !slices.Equal(got, want) {
			t.Errorf("--file StoryData/E* --file PersonalityVoiceDlg/** = %v, want %v", got, want)
		}
	})
	t.Run("origin", func(t *testing.T) {
		got := searchKeys(t, "--file", "StoryData/S1001.json", "--origin", "human")
		if want := []string{searchKeyS1001b}; !slices.Equal(got, want) {
			t.Errorf("--origin human = %v, want %v", got, want)
		}
		// A unit without a record has no origin to match.
		got = searchKeys(t, "--file", "StoryData/S1003.json", "--origin", "claude", "--origin", "human", "--origin", "paratranz", "--origin", "fallback")
		if len(got) != 0 {
			t.Errorf("origins on a unit without a record = %v, want none", got)
		}
		// Without --origin the copies (the model and id fields) are left out,
		// the unit without a record is not; --origin copy asks for the copies.
		got = searchKeys(t, "--file", "StoryData/S1003.json")
		if want := []string{searchKeyS1003}; !slices.Equal(got, want) {
			t.Errorf("S1003 without --origin = %v, want %v", got, want)
		}
		got = searchKeys(t, "--file", "StoryData/S1003.json", "--origin", "copy")
		if want := []string{"StoryData/S1003.json|/dataList/0/model"}; !slices.Equal(got, want) {
			t.Errorf("--origin copy = %v, want %v", got, want)
		}
		got = searchKeys(t, "--file", "StoryData/S1001.json", "--origin", "human", "--origin", "claude")
		if len(got) != 3 {
			t.Errorf("two origins = %v, want the three units of S1001", got)
		}
	})
	t.Run("text on KR or zh", func(t *testing.T) {
		// 식었 is in the KR of S1001 #2 only; its zh has no match.
		got := searchKeys(t, "--text", "식었")
		if want := []string{searchKeyS1001c}; !slices.Equal(got, want) {
			t.Errorf("--text on KR = %v, want %v", got, want)
		}
		// 茶涼 is in the zh of S1001 #2 only.
		got = searchKeys(t, "--text", "茶涼")
		if want := []string{searchKeyS1001c}; !slices.Equal(got, want) {
			t.Errorf("--text on zh = %v, want %v", got, want)
		}
		// A Go regexp, either side: 매니저$ is KR S1003 (a ? ends it, so no), ^經理$ is zh E002 only.
		got = searchKeys(t, "--text", "^經理$")
		if want := []string{searchKeyE002}; !slices.Equal(got, want) {
			t.Errorf("--text ^經理$ = %v, want %v", got, want)
		}
		got = searchKeys(t, "--text", "(?i)^매니저[,?]")
		if want := []string{searchKeyS1001b, searchKeyS1003}; !slices.Equal(got, want) {
			t.Errorf("--text (?i)^매니저[,?] = %v, want %v", got, want)
		}
	})
	t.Run("key", func(t *testing.T) {
		got := searchKeys(t, "--key", searchKeyS1002, "--key", searchKeyE002, "--key", "StoryData/S1002.json|/dataList/7/content")
		want := []string{searchKeyE002, searchKeyS1002}
		if !slices.Equal(got, want) {
			t.Errorf("--key ×3 = %v, want %v: a key that is not in KR is no result", got, want)
		}
		// Keys and the other conditions all have to hold.
		got = searchKeys(t, "--key", searchKeyS1002, "--key", searchKeyE002, "--file", "StoryData/E*")
		if want := []string{searchKeyE002}; !slices.Equal(got, want) {
			t.Errorf("--key with --file = %v, want %v", got, want)
		}
	})
	t.Run("no condition", func(t *testing.T) {
		// Every unit of KR, in the order of the files (sorted by name) and of
		// the documents: the 7 units of this repo's files and the 2 of Sample.json.
		want := []string{searchKeyVoice, batchKey0, batchKey1, searchKeyE002,
			searchKeyS1001a, searchKeyS1001b, searchKeyS1001c, searchKeyS1002, searchKeyS1003}
		if got := searchKeys(t); !slices.Equal(got, want) {
			t.Errorf("no condition = %v, want %v", got, want)
		}
	})
}

// A copy (the model and id fields) is left out of a search by default: it is
// an identifier, nothing to find. A key that is asked for by name is a search
// for exactly that unit, and finds it whatever its record is.
func TestSearchKeyIncludesCopies(t *testing.T) {
	searchRepo(t)
	const copyKey = "StoryData/S1003.json|/dataList/0/model"

	if got := searchKeys(t, "--file", "StoryData/S1003.json"); !slices.Equal(got, []string{searchKeyS1003}) {
		t.Fatalf("without --key = %v, want the copies left out", got)
	}

	var hits []searchHit
	mustLct(t, &hits, "search", "--key", copyKey)
	if len(hits) != 1 || hits[0].Key != copyKey || hits[0].Origin != store.OriginCopy || hits[0].KR != "엑스트라" {
		t.Fatalf("--key of a copy = %+v, want the copy with its KR text", hits)
	}

	got := searchKeys(t, "--key", copyKey, "--key", searchKeyS1003)
	slices.Sort(got)
	if want := []string{searchKeyS1003, copyKey}; !slices.Equal(got, want) {
		t.Errorf("--key of a copy and of a translation = %v, want %v", got, want)
	}
	// The voice line's id is a copy too, in a file with a record of each kind.
	if got := searchKeys(t, "--key", "PersonalityVoiceDlg/Voice_HongLu_10601.json|/dataList/0/id"); len(got) != 1 {
		t.Errorf("--key of the id of the voice line = %v, want that unit", got)
	}

	// What else is asked for still holds: a key is no way around --origin,
	// --file or --text.
	if got := searchKeys(t, "--key", copyKey, "--origin", "claude"); len(got) != 0 {
		t.Errorf("--key of a copy with --origin claude = %v, want none", got)
	}
	if got := searchKeys(t, "--key", copyKey, "--origin", "copy"); !slices.Equal(got, []string{copyKey}) {
		t.Errorf("--key of a copy with --origin copy = %v, want the copy", got)
	}
	if got := searchKeys(t, "--key", copyKey, "--file", "StoryData/E*"); len(got) != 0 {
		t.Errorf("--key of a copy with another --file = %v, want none", got)
	}
	// Without a key, --file does not bring the copies in.
	if got := searchKeys(t, "--file", "PersonalityVoiceDlg/**"); !slices.Equal(got, []string{searchKeyVoice}) {
		t.Errorf("--file of a file with copies = %v, want its translation only", got)
	}
}

func TestSearchSpeaker(t *testing.T) {
	searchRepo(t)

	// A known character is named by id, whether the model says so or the file
	// name does; the lines of a faust are not honglu's.
	got := searchKeys(t, "--speaker", "honglu")
	want := []string{searchKeyVoice, searchKeyE002, searchKeyS1001a, searchKeyS1001c, searchKeyS1002}
	if !slices.Equal(got, want) {
		t.Errorf("--speaker honglu = %v, want %v", got, want)
	}

	got = searchKeys(t, "--speaker", "faust")
	if want := []string{searchKeyS1001b}; !slices.Equal(got, want) {
		t.Errorf("--speaker faust = %v, want %v", got, want)
	}

	// Nobody the rules know: the raw model string is the speaker.
	got = searchKeys(t, "--speaker", "엑스트라")
	if want := []string{searchKeyS1003}; !slices.Equal(got, want) {
		t.Errorf("--speaker 엑스트라 = %v, want %v", got, want)
	}

	// The model string of a known character is not its speaker name.
	if got := searchKeys(t, "--speaker", "홍루"); len(got) != 0 {
		t.Errorf("--speaker 홍루 = %v, want none: the speaker of the model 홍루 is honglu", got)
	}

	// With the other conditions.
	got = searchKeys(t, "--speaker", "honglu", "--origin", "claude", "--text", "您")
	want = []string{searchKeyVoice, searchKeyS1001a, searchKeyS1002}
	if !slices.Equal(got, want) {
		t.Errorf("--speaker honglu --origin claude --text 您 = %v, want %v", got, want)
	}
}

func TestSearchOutput(t *testing.T) {
	searchRepo(t)

	var hits []searchHit
	mustLct(t, &hits, "search", "--key", searchKeyS1001a, "--key", searchKeyS1003)
	want := []searchHit{
		{Key: searchKeyS1001a, Field: "content", Speaker: "honglu", KR: "매니저님, 안녕하세요", ZH: "經理，您好", Origin: "claude"},
		// No record yet: no translation and no origin.
		{Key: searchKeyS1003, Field: "content", Speaker: "엑스트라", KR: "매니저?"},
	}
	if !slices.Equal(hits, want) {
		t.Errorf("hits = %+v, want %+v", hits, want)
	}

	// The fields of a hit are exactly these, and an empty result is [].
	code, stdout, stderr := lct(t, "--json", "search", "--key", searchKeyS1002)
	if code != 0 || stderr != "" {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	var raw []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stdout), &raw); err != nil || len(raw) != 1 {
		t.Fatalf("output is not a JSON array of one hit: %v\n%s", err, stdout)
	}
	var names []string
	for name := range raw[0] {
		names = append(names, name)
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"field", "key", "kr", "origin", "speaker", "zh"}) {
		t.Errorf("JSON fields = %v", names)
	}
	_, stdout, _ = lct(t, "--json", "search", "--text", "no such text anywhere")
	if strings.TrimSpace(stdout) != "[]" {
		t.Errorf("an empty result = %q, want []", stdout)
	}

	// As text: a block a unit with the key, speaker and origin, then both texts.
	code, stdout, _ = lct(t, "search", "--key", searchKeyS1001a)
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	for _, part := range []string{searchKeyS1001a, "honglu", "claude", "매니저님, 안녕하세요", "經理，您好"} {
		if !strings.Contains(stdout, part) {
			t.Errorf("text output lacks %q:\n%s", part, stdout)
		}
	}
	if _, stdout, _ = lct(t, "search", "--text", "no such text anywhere"); !strings.Contains(stdout, "no units") {
		t.Errorf("text output of an empty result = %q, want it to say there are no units", stdout)
	}
}

func TestSearchLimit(t *testing.T) {
	root := searchRepo(t)

	// In file order, then document order; --limit cuts the list there.
	got := searchKeys(t, "--file", "StoryData/S1*", "--limit", "2")
	if want := []string{searchKeyS1001a, searchKeyS1001b}; !slices.Equal(got, want) {
		t.Errorf("--limit 2 = %v, want %v", got, want)
	}

	// Text output says when the list was cut, and not when it was whole.
	_, stdout, _ := lct(t, "search", "--file", "StoryData/S1*", "--limit", "2")
	if !strings.Contains(stdout, "limit") {
		t.Errorf("a cut list does not say so:\n%s", stdout)
	}
	_, stdout, _ = lct(t, "search", "--file", "StoryData/S1*", "--limit", "5")
	if strings.Contains(stdout, "limit") {
		t.Errorf("a list of exactly --limit units says it was cut:\n%s", stdout)
	}

	// The default is 200: a file of 250 units gives 200 of them.
	var b strings.Builder
	b.WriteString(`{"dataList":[`)
	for i := 0; i < 250; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, `{"id":%d,"content":"텍스트 %d"}`, i, i)
	}
	b.WriteString("]}")
	writeTestFile(t, filepath.Join(root, ".work", "raw", "kr"), "KR_Big.json", b.String())
	if got := searchKeys(t, "--file", "Big.json"); len(got) != 200 {
		t.Errorf("default limit: %d units, want 200", len(got))
	}
	if got := searchKeys(t, "--file", "Big.json", "--limit", "250"); len(got) != 250 {
		t.Errorf("--limit 250: %d units, want 250", len(got))
	}
}

func TestSearchUsage(t *testing.T) {
	searchRepo(t)

	for name, tc := range map[string]struct {
		args   []string
		stderr string
	}{
		"bad regexp":          {[]string{"--text", "(unclosed"}, "--text"},
		"bad glob":            {[]string{"--file", "StoryData/[S"}, "StoryData/[S"},
		"unknown origin":      {[]string{"--origin", "robot"}, "robot"},
		"malformed key":       {[]string{"--key", "nope"}, "nope"},
		"zero limit":          {[]string{"--limit", "0"}, "--limit"},
		"negative limit":      {[]string{"--limit", "-3"}, "--limit"},
		"unexpected argument": {[]string{"extra"}, "extra"},
		"unknown flag":        {[]string{"--bogus"}, "bogus"},
	} {
		t.Run(name, func(t *testing.T) {
			code, stdout, stderr := lct(t, append([]string{"search"}, tc.args...)...)
			if code != 1 {
				t.Errorf("exit code = %d, want 1\nstderr: %s", code, stderr)
			}
			if stdout != "" || !strings.Contains(stderr, tc.stderr) {
				t.Errorf("stdout = %q, stderr = %q, want nothing on stdout and a message with %q", stdout, stderr, tc.stderr)
			}
		})
	}

	// Without a snapshot there is no KR to search.
	t.Run("no snapshot", func(t *testing.T) {
		t.Chdir(snapshotRepo(t))
		code, _, stderr := lct(t, "search", "--text", "x")
		if code != 1 || !strings.Contains(stderr, "snapshot") {
			t.Errorf("exit code = %d, stderr = %q, want 1 and a hint to run lct snapshot", code, stderr)
		}
	})
}
