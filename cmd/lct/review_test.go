package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/plan"
	"github.com/tool-jx3/limbus-translate/internal/store"
)

// reviewJSON is "review --json".
type reviewJSON struct {
	Title    string              `json:"title"`
	Sections []reviewJSONSection `json:"sections"`
}

type reviewJSONSection struct {
	Kind  string   `json:"kind"`
	Title string   `json:"title"`
	Hint  string   `json:"hint"`
	Items []string `json:"items"`
	Keys  []string `json:"keys"`
}

// writeReviewPlan writes sum as the plan of the tw worktree of root, as it is.
func writeReviewPlan(t *testing.T, root string, sum plan.Summary) {
	t.Helper()
	if err := store.Open(filepath.Join(root, ".work", "tw")).WriteMetaJSON(plan.File, sum); err != nil {
		t.Fatal(err)
	}
}

// checklist reads the issue that the build wrote: its title, and for each part
// the lines of the checklist and the headings, in order.
func checklist(t *testing.T, root string) (title string, headings, items []string) {
	t.Helper()
	dir := reviewIssueDir(root)
	title = strings.TrimSuffix(string(readFile(t, filepath.Join(dir, "title.txt"))), "\n")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() == "title.txt" {
			continue
		}
		for _, line := range strings.Split(string(readFile(t, filepath.Join(dir, e.Name()))), "\n") {
			if item, ok := strings.CutPrefix(line, "- [ ] "); ok {
				items = append(items, item)
			} else if h, ok := strings.CutPrefix(line, "### "); ok {
				headings = append(headings, h)
			}
		}
	}
	return title, headings, items
}

// "lct review" lists what the review issue of the last build lists, from the
// state of the tw worktree: the same items, in the same order; with nothing to
// do the list of sections is empty.
func TestReviewCommand(t *testing.T) {
	root := buildRepo(t)
	t.Setenv(sourceDateEpoch, "")
	t.Chdir(root)

	// Nothing needs a person: an empty list, and the title of the day all the
	// same.
	if _, err := runBuildJSON(t, root); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := lct(t, "--json", "review")
	if code != 0 || stderr != "" {
		t.Fatalf("review: exit code %d, stderr %q\n%s", code, stderr, stdout)
	}
	var none reviewJSON
	if err := json.Unmarshal([]byte(stdout), &none); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, stdout)
	}
	if none.Title != "審核：2026-10-29 更新" || none.Sections == nil || len(none.Sections) != 0 {
		t.Errorf("review with nothing to do = %+v, want the title and sections []", none)
	}
	if !strings.Contains(stdout, `"sections": []`) {
		t.Errorf("sections is not an empty JSON list:\n%s", stdout)
	}
	if code, stdout, _ := lct(t, "review"); code != 0 || !strings.Contains(stdout, "沒有需要處理的項目") {
		t.Errorf("text output with nothing to do: exit code %d\n%s", code, stdout)
	}

	// Units that need a person, and a plan with the other kinds of item.
	stuck := markNeedsReview(t, root, 3)
	writeReviewPlan(t, root, plan.Summary{
		RawTag: "raw-20261029", PrevRawTag: "raw-20261022",
		Adjustments:           plan.AdjustmentSummary{Expired: []string{"StoryData/S1.json|/dataList/4/content"}},
		FrozenChanged:         []string{"BgmLyrics/L.json|/dataList/9/content"},
		NewSpeakers:           []string{"신규 NPC"},
		FieldChanges:          []fields.Change{{Op: "add", Group: "Skills", Field: "tag"}},
		NewFilesDefaultFamily: []string{"Brand/New.json"},
	})
	if got, err := runBuildJSON(t, root); err != nil || got.ReviewIssueParts != 1 {
		t.Fatalf("build: %+v, %v, want one part of the review issue", got, err)
	}
	wantTitle, issueHeadings, issueItems := checklist(t, root)
	if len(issueItems) != 3+1+1+1+1+1 {
		t.Fatalf("the issue has %d items: %q", len(issueItems), issueItems)
	}

	code, stdout, stderr = lct(t, "--json", "review")
	if code != 0 || stderr != "" {
		t.Fatalf("review: exit code %d, stderr %q\n%s", code, stderr, stdout)
	}
	var got reviewJSON
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, stdout)
	}
	if got.Title != wantTitle {
		t.Errorf("title = %q, want that of the issue, %q", got.Title, wantTitle)
	}
	var headings, items []string
	for _, s := range got.Sections {
		if s.Kind == "" || s.Hint == "" {
			t.Errorf("section %+v lacks a kind or a hint", s)
		}
		headings = append(headings, s.Title)
		items = append(items, s.Items...)
	}
	if strings.Join(headings, "\n") != strings.Join(issueHeadings, "\n") {
		t.Errorf("sections = %q, the issue has %q", headings, issueHeadings)
	}
	if strings.Join(items, "\n") != strings.Join(issueItems, "\n") {
		t.Errorf("items = %q\nthe issue has %q", items, issueItems)
	}
	if got.Sections[0].Kind != "needs_review" || len(got.Sections[0].Items) != 3 || !strings.Contains(got.Sections[0].Items[0], stuck[0].Key) {
		t.Errorf("first section = %+v, want the 3 units that need review", got.Sections[0])
	}
	if k := got.Sections[0].Keys; len(k) != 3 || k[0] != stuck[0].Key {
		t.Errorf("first section keys = %q, want the keys of the 3 units, the first %q", k, stuck[0].Key)
	}

	// The text is for a person: the title, each section with its hint, each item.
	code, stdout, stderr = lct(t, "review")
	if code != 0 || stderr != "" {
		t.Fatalf("review (text): exit code %d, stderr %q", code, stderr)
	}
	for _, want := range append([]string{wantTitle, "8 項"}, append(issueHeadings, issueItems...)...) {
		if !strings.Contains(stdout, want) {
			t.Errorf("the text output lacks %q:\n%s", want, stdout)
		}
	}

	// The plan of another snapshot is no plan, as in the build.
	writeReviewPlan(t, root, plan.Summary{RawTag: "raw-20260101", NewSpeakers: []string{"舊 NPC"}})
	var other reviewJSON
	mustLct(t, &other, "review")
	for _, s := range other.Sections {
		if s.Kind != "needs_review" {
			t.Errorf("a plan of another snapshot gave the section %q", s.Kind)
		}
	}
}

// Without a build there is nothing to list: exit 1, and a message that says
// what to do. Nothing is written.
func TestReviewCommandNeedsABuild(t *testing.T) {
	root := buildRepo(t)
	t.Chdir(root)

	// The tw worktree is there but nothing was built in it.
	code, stdout, stderr := lct(t, "--json", "review")
	if code != 1 || !strings.Contains(stderr, "tw") || !strings.Contains(stderr, "lct build") || stdout != "" {
		t.Errorf("before a build: exit code %d (want 1), stderr %q, stdout %q", code, stderr, stdout)
	}

	// No worktree at all.
	if err := os.RemoveAll(filepath.Join(root, ".work", "tw")); err != nil {
		t.Fatal(err)
	}
	code, _, stderr = lct(t, "review")
	if code != 1 || !strings.Contains(stderr, ".work/tw") || !strings.Contains(stderr, "git worktree add") {
		t.Errorf("without the tw worktree: exit code %d (want 1), stderr %q", code, stderr)
	}
	if _, err := os.Stat(filepath.Join(root, ".work", "tw")); !os.IsNotExist(err) {
		t.Errorf("review created the tw worktree (%v)", err)
	}

	code, _, stderr = lct(t, "review", "extra")
	if code != 1 || !strings.Contains(stderr, "unexpected argument") {
		t.Errorf("review extra: exit code %d (want 1), stderr %q", code, stderr)
	}
}
