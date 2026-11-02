package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/plan"
	"github.com/tool-jx3/limbus-translate/internal/proposal"
	"github.com/tool-jx3/limbus-translate/internal/publish"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/suggest"
)

// fakeGHEnv makes the test binary the gh CLI of lct proposal (see TestMain);
// its value is the file that holds the fake GitHub (fakeGitHub). Only the
// script that stands in for gh sets it, for the binary it runs.
const fakeGHEnv = "LCT_E2E_FAKE_GH"

// fakeGitHub is the GitHub of the rules loop replay, as the fake gh keeps it
// in its file: the pull requests and labels of the repo, and every call of gh
// that was made, as its arguments.
type fakeGitHub struct {
	// Origin is the bare repo the pull requests are made on: a PR is only
	// opened from a branch it has.
	Origin string     `json:"origin"`
	PRs    []fakePR   `json:"prs"`
	Labels []string   `json:"labels"`
	Calls  [][]string `json:"calls"`
}

// fakePR is a pull request of the fake GitHub.
type fakePR struct {
	Number      int    `json:"number"`
	URL         string `json:"url"`
	HeadRefName string `json:"headRefName"`
	BaseRefName string `json:"baseRefName"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	// State is OPEN, CLOSED or MERGED.
	State string `json:"state"`
	// MergedAt is nil unless the state is MERGED.
	MergedAt          *string       `json:"mergedAt"`
	Labels            []fakePRLabel `json:"labels"`
	IsCrossRepository bool          `json:"isCrossRepository"`
}

type fakePRLabel struct {
	Name string `json:"name"`
}

// field is the value gh pr list --json prints for name, and whether gh knows
// the name.
func (p fakePR) field(name string) (any, bool) {
	switch name {
	case "number":
		return p.Number, true
	case "url":
		return p.URL, true
	case "headRefName":
		return p.HeadRefName, true
	case "baseRefName":
		return p.BaseRefName, true
	case "title":
		return p.Title, true
	case "body":
		return p.Body, true
	case "state":
		return p.State, true
	case "mergedAt":
		return p.MergedAt, true
	case "labels":
		if p.Labels == nil {
			return []fakePRLabel{}, true
		}
		return p.Labels, true
	case "isCrossRepository":
		return p.IsCrossRepository, true
	}
	return nil, false
}

// fakeGH is one call of gh against the fake GitHub in the file state. It
// answers the calls lct proposal makes the way gh does, and refuses any other
// command and any flag it does not know, so that a change of what lct asks gh
// fails here. Every call is recorded, the refused ones too. It returns the
// exit code.
func fakeGH(state string, args []string) int {
	gh, err := loadFakeGitHub(state)
	if err == nil {
		gh.Calls = append(gh.Calls, args)
		var out string
		out, err = gh.run(args)
		if saveErr := gh.save(state); err == nil {
			err = saveErr
		}
		if err == nil {
			fmt.Print(out)
			return 0
		}
	}
	fmt.Fprintln(os.Stderr, "fake gh:", err)
	return 1
}

func loadFakeGitHub(path string) (*fakeGitHub, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var gh fakeGitHub
	if err := json.Unmarshal(data, &gh); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &gh, nil
}

func (gh *fakeGitHub) save(path string) error {
	data, err := json.MarshalIndent(gh, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// run answers one call. It changes gh only when the call succeeds.
func (gh *fakeGitHub) run(args []string) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("unknown command %q", args)
	}
	switch rest := args[2:]; args[0] + " " + args[1] {
	case "pr list":
		return gh.prList(rest)
	case "pr create":
		return gh.prCreate(rest)
	case "pr edit":
		return gh.prEdit(rest)
	case "label create":
		return gh.labelCreate(rest)
	}
	return "", fmt.Errorf("unknown command %q", args)
}

// parseFakeGHFlags parses the flags of a call of the fake gh; an argument
// that is not a flag is an error.
func parseFakeGHFlags(f *flag.FlagSet, args []string) error {
	f.SetOutput(io.Discard)
	if err := f.Parse(args); err != nil {
		return fmt.Errorf("%s: %w", f.Name(), err)
	}
	if f.NArg() > 0 {
		return fmt.Errorf("%s: unexpected argument %q", f.Name(), f.Arg(0))
	}
	return nil
}

// prList is gh pr list --json: the PRs of the state (open, closed, which
// takes in the merged ones, merged or all) and the head branch, newest first,
// with the fields asked for.
func (gh *fakeGitHub) prList(args []string) (string, error) {
	f := flag.NewFlagSet("pr list", flag.ContinueOnError)
	state := f.String("state", "open", "")
	head := f.String("head", "", "")
	fields := f.String("json", "", "")
	limit := f.Int("limit", 30, "")
	if err := parseFakeGHFlags(f, args); err != nil {
		return "", err
	}
	in := map[string][]string{"open": {"OPEN"}, "closed": {"CLOSED", "MERGED"}, "merged": {"MERGED"}, "all": {"OPEN", "CLOSED", "MERGED"}}[*state]
	if in == nil {
		return "", fmt.Errorf("pr list: invalid state %q", *state)
	}
	if *fields == "" {
		return "", errors.New("pr list: the fake answers --json only")
	}
	prs := slices.Clone(gh.PRs)
	slices.SortFunc(prs, func(a, b fakePR) int { return b.Number - a.Number })
	list := []map[string]any{}
	for _, p := range prs {
		if !slices.Contains(in, p.State) || *head != "" && p.HeadRefName != *head {
			continue
		}
		if len(list) == *limit {
			break
		}
		entry := map[string]any{}
		for _, name := range strings.Split(*fields, ",") {
			v, ok := p.field(name)
			if !ok {
				return "", fmt.Errorf("pr list: unknown JSON field %q", name)
			}
			entry[name] = v
		}
		list = append(list, entry)
	}
	data, err := json.Marshal(list)
	return string(data) + "\n", err
}

// prCreate is gh pr create: the head must be a branch of origin with no open
// PR, and every label must exist. It prints the URL of the new PR.
func (gh *fakeGitHub) prCreate(args []string) (string, error) {
	f := flag.NewFlagSet("pr create", flag.ContinueOnError)
	base := f.String("base", "", "")
	head := f.String("head", "", "")
	title := f.String("title", "", "")
	bodyFile := f.String("body-file", "", "")
	var labels stringList
	f.Var(&labels, "label", "")
	if err := parseFakeGHFlags(f, args); err != nil {
		return "", err
	}
	if *base == "" || *head == "" || *title == "" || *bodyFile == "" {
		return "", errors.New("pr create: the fake wants --base, --head, --title and --body-file")
	}
	body, err := os.ReadFile(*bodyFile)
	if err != nil {
		return "", err
	}
	pr := fakePR{
		Number: len(gh.PRs) + 1, HeadRefName: *head, BaseRefName: *base, Title: *title,
		Body: string(body), State: "OPEN", Labels: []fakePRLabel{},
	}
	pr.URL = fmt.Sprintf("https://github.com/o/r/pull/%d", pr.Number)
	for _, l := range labels {
		if !slices.Contains(gh.Labels, l) {
			return "", fmt.Errorf("pr create: could not add label: %q not found", l)
		}
		pr.Labels = append(pr.Labels, fakePRLabel{Name: l})
	}
	for _, p := range gh.PRs {
		if p.State == "OPEN" && p.HeadRefName == *head {
			return "", fmt.Errorf("pr create: a pull request for branch %q into branch %q already exists: %s", *head, p.BaseRefName, p.URL)
		}
	}
	if err := exec.Command("git", "--git-dir", gh.Origin, "rev-parse", "--verify", "--quiet", "refs/heads/"+*head).Run(); err != nil {
		return "", fmt.Errorf("pr create: origin has no branch %s", *head)
	}
	gh.PRs = append(gh.PRs, pr)
	return pr.URL + "\n", nil
}

// prEdit is gh pr edit NUMBER --body-file F.
func (gh *fakeGitHub) prEdit(args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("pr edit: no PR number")
	}
	n, err := strconv.Atoi(args[0])
	if err != nil {
		return "", fmt.Errorf("pr edit: the fake takes a PR number, not %q", args[0])
	}
	f := flag.NewFlagSet("pr edit", flag.ContinueOnError)
	bodyFile := f.String("body-file", "", "")
	if err := parseFakeGHFlags(f, args[1:]); err != nil {
		return "", err
	}
	if *bodyFile == "" {
		return "", errors.New("pr edit: the fake edits the body only")
	}
	body, err := os.ReadFile(*bodyFile)
	if err != nil {
		return "", err
	}
	i := slices.IndexFunc(gh.PRs, func(p fakePR) bool { return p.Number == n })
	if i < 0 {
		return "", fmt.Errorf("pr edit: no pull request #%d", n)
	}
	gh.PRs[i].Body = string(body)
	return gh.PRs[i].URL + "\n", nil
}

// labelCreate is gh label create NAME; a label that exists needs --force.
func (gh *fakeGitHub) labelCreate(args []string) (string, error) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return "", errors.New("label create: no label name")
	}
	f := flag.NewFlagSet("label create", flag.ContinueOnError)
	force := f.Bool("force", false, "")
	f.String("color", "", "")
	f.String("description", "", "")
	if err := parseFakeGHFlags(f, args[1:]); err != nil {
		return "", err
	}
	if slices.Contains(gh.Labels, args[0]) {
		if !*force {
			return "", fmt.Errorf("label create: label with name %q already exists; use --force to update its color and description", args[0])
		}
		return "", nil
	}
	gh.Labels = append(gh.Labels, args[0])
	return "", nil
}

// shellQuote quotes s for sh.
func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// rulesLoop is the rules loop replay as it goes: the phase 2 replay of the
// repo, with a bare origin and the fake gh.
type rulesLoop struct {
	*phase2
	origin  string
	ghState string // the file of the fake GitHub
}

// The replay's own texts. The second snapshot gives StoryData/S2.json a third
// line, said by 모나, whom v1 does not have and no guide knows. A person
// adjusts the first line of S2, which is also the fifth line of S1, so that
// the two differ.
const (
	newSpeaker      = "모나"
	newSpeakerKR    = "다들 버스에 타세요."
	busKR           = "버스는 정해진 시간에 출발합니다."
	busZH           = "巴士會準時發車。"
	busHint         = "與 StoryData/S2.json 的同一句統一，譯為「巴士會準時發車。」"
	stopKR          = "정거장"
	rulesDay        = "2026-11-05" // the day of the second update; rules-proposal.yml runs after it
	rulesBranch     = proposal.BranchPrefix + rulesDay
	rulesNextDay    = "2026-11-06"
	rulesRequestID  = rulesDay + "-bus"
	umbrellaFile    = "rules/adjustments/2026-11-03-umbrella.yaml"
	newSpeakerGuide = "guides/characters/mona.md"
	declinedFile    = "rules/declined.yaml"
	proposalBodyRel = "work/proposal/body.md"
)

// The suggestions of the second snapshot: the line of S1 and S2 that is
// translated in two ways, the new speaker, and the adjustment of S1 that the
// inserted line moved down one line.
var (
	inconsistentID = "inconsistent/story-main/content/" + sha8Hex(busKR)
	speakerID      = "speaker/" + newSpeaker
	movedID        = "adjustment-moved/" + p2Content(s1, 2)
	stopTermID     = "term/" + stopKR
	loopIDs        = []string{movedID, inconsistentID, speakerID}
)

func sha8Hex(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])[:8]
}

// rulesLoopFixture is testdata/phase2 with the third line of S2 in v2, in
// every language.
func rulesLoopFixture(t *testing.T) string {
	t.Helper()
	src, err := filepath.Abs(phase2Fixture)
	if err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir()
	copyDir(t, src, dst)
	type line struct {
		ID      int    `json:"id"`
		Model   string `json:"model,omitempty"`
		Content string `json:"content"`
	}
	for lang, content := range map[string]string{"kr": newSpeakerKR, "en": "Everyone, please get on the bus.", "jp": "皆さん、バスに乗ってください。"} {
		path := filepath.Join(dst, "v2", lang, "StoryData", strings.ToUpper(lang)+"_S2.json")
		var doc struct {
			DataList []line `json:"dataList"`
		}
		if err := json.Unmarshal(readFile(t, path), &doc); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		doc.DataList = append(doc.DataList, line{ID: len(doc.DataList) + 1, Model: newSpeaker, Content: content})
		data, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dst
}

// newRulesLoop makes the repo of the replay, its bare origin (which has main)
// and the fake gh, first in the PATH. Neither the machine's git settings nor
// its tokens count.
func newRulesLoop(t *testing.T) *rulesLoop {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL", "EMAIL", "GITHUB_TOKEN"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
	t.Setenv("GH_TOKEN", "test-token")

	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	r := &rulesLoop{phase2: &phase2{t: t, root: e2eRepo(t), fixture: rulesLoopFixture(t), claude: bin, fakeDir: t.TempDir()}}
	r.origin = filepath.Join(t.TempDir(), "origin.git")
	runGitIn(t, r.root, "init", "-q", "--bare", "-b", "main", r.origin)
	runGitIn(t, r.root, "remote", "add", "origin", r.origin)
	r.pushMain()

	ghDir := t.TempDir()
	r.ghState = filepath.Join(ghDir, "github.json")
	if err := (&fakeGitHub{Origin: r.origin, PRs: []fakePR{}, Labels: []string{}, Calls: [][]string{}}).save(r.ghState); err != nil {
		t.Fatal(err)
	}
	// The binary is gh for its run only: lct translate runs it as claude, with
	// the variable of the fake claude, which gh does not pass on.
	script := "#!/bin/sh\nunset " + fakeClaudeEnv + "\n" + fakeGHEnv + "=" + shellQuote(r.ghState) + "\nexport " + fakeGHEnv + "\nexec " + shellQuote(bin) + " \"$@\"\n"
	writeTestFile(t, ghDir, "gh", script)
	if err := os.Chmod(filepath.Join(ghDir, "gh"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", ghDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	t.Chdir(r.root)
	t.Setenv("LCT_MODEL", "e2e") // not a real model name
	t.Setenv(sourceDateEpoch, "")
	t.Setenv(fakeClaudeEnv, r.fakeDir)
	return r
}

func (r *rulesLoop) git(args ...string) string { r.t.Helper(); return runGitIn(r.t, r.root, args...) }

// pushMain pushes main to origin, as people do once they have committed to it.
func (r *rulesLoop) pushMain() { r.t.Helper(); r.git("push", "-q", "origin", "main") }

// commitMain commits everything on main and pushes it.
func (r *rulesLoop) commitMain(msg string) {
	r.t.Helper()
	if b := r.git("branch", "--show-current"); b != "main" {
		r.t.Fatalf("committing on %s, want main", b)
	}
	r.git("add", "-A")
	r.git("commit", "-q", "-m", msg)
	r.pushMain()
}

// originHead is the commit origin has for the branch, or "".
func (r *rulesLoop) originHead(branch string) string {
	r.t.Helper()
	out, err := exec.Command("git", "--git-dir", r.origin, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (r *rulesLoop) github() *fakeGitHub {
	r.t.Helper()
	gh, err := loadFakeGitHub(r.ghState)
	if err != nil {
		r.t.Fatal(err)
	}
	return gh
}

// editGitHub is a person on the GitHub site: edit changes the fake GitHub.
func (r *rulesLoop) editGitHub(edit func(*fakeGitHub)) {
	r.t.Helper()
	gh := r.github()
	edit(gh)
	if err := gh.save(r.ghState); err != nil {
		r.t.Fatal(err)
	}
}

// ghCallsSince returns the gh calls after the first n, as "verb noun".
func (r *rulesLoop) ghCallsSince(n int) []string {
	r.t.Helper()
	var out []string
	for _, c := range r.github().Calls[n:] {
		out = append(out, strings.Join(c[:min(2, len(c))], " "))
	}
	return out
}

// rebuild is the pipeline as rebuild.yml runs it after a change of the rules
// on main when it translates nothing (nothing is pending, or a dry run): plan,
// build, publish.
func (r *rulesLoop) rebuild() (plan.Summary, buildOut, publish.Result) {
	r.t.Helper()
	sum := r.plan()
	built := r.build()
	return sum, built, r.publish()
}

// start is the first step of rules-proposal.yml: lct proposal start.
func (r *rulesLoop) start(date string) proposal.Started {
	r.t.Helper()
	var started proposal.Started
	mustLct(r.t, &started, "proposal", "start", "--date", date)
	if b := r.git("branch", "--show-current"); b != started.Branch {
		r.t.Errorf("proposal start said %s, but %s is checked out", started.Branch, b)
	}
	return started
}

// suggestions is the second step: lct suggest with the exclude file of start.
func (r *rulesLoop) suggestions() suggestReport {
	r.t.Helper()
	var rep suggestReport
	mustLct(r.t, &rep, "suggest", "--exclude-file", proposal.ExcludeFile)
	return rep
}

func suggestionIDsOf(rep suggestReport) []string {
	ids := []string{}
	for _, s := range rep.Suggestions {
		ids = append(ids, s.ID)
	}
	return ids
}

// candidateTerms are the ids of the glossary candidates of the repo's own
// glossary/terms.yaml, which the replay copies in and leaves alone: lct
// suggest lists them every time, but those rules/declined.yaml turns down,
// which it counts as excluded. It also makes sure that no guide of the repo
// knows the new speaker of the replay.
func candidateTerms(t *testing.T, root string) (ids []string, declined int) {
	t.Helper()
	r, err := rules.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if c, _, ok := r.CharacterByModel(newSpeaker); ok {
		t.Fatalf("%s knows the model %s, the new speaker of the replay: give the replay another one", c.Path, newSpeaker)
	}
	turnedDown, err := suggest.LoadDeclined(root)
	if err != nil {
		t.Fatal(err)
	}
	ids = []string{}
	for _, term := range r.Glossary {
		if term.Status != rules.StatusCandidate || term.KR == "" {
			continue
		}
		id := suggest.KindTerm + "/" + term.KR
		if slices.ContainsFunc(turnedDown, func(d suggest.Declined) bool { return d.ID == id }) {
			declined++
		} else {
			ids = append(ids, id)
		}
	}
	return ids, declined
}

func sortedIDs(ids ...[]string) []string {
	out := slices.Concat(ids...)
	if out == nil {
		out = []string{}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// readExclude reads the exclude file start wrote.
func (r *rulesLoop) readExclude() []string {
	r.t.Helper()
	ids := []string{}
	for _, line := range strings.Split(string(readFile(r.t, filepath.Join(r.root, proposal.ExcludeFile))), "\n") {
		if line != "" {
			ids = append(ids, line)
		}
	}
	return ids
}

// The description the rules-review skill writes (its step 4): an opening, one
// section per suggestion, each starting with its marker on a line of its own,
// and the impact assessment of lct impact last. The opening says what the
// skill tells it to, the marker in a line of text included: that is not a
// marker, and no suggestion.
const (
	proposalOpening = "這是依 `lct suggest` 整理的規則建議。不要的項目請把它的 id 移到 `rules/declined.yaml`（`id`、`reason`、`date`）並還原該項修改，再合併；" +
		"關閉而未合併，裡面的項目全部視為不採用；不要刪掉 `<!-- suggestion:… -->` 那一行。\n\n"
	impactHeading = "## 規則變更的影響評估"
)

func proposalSection(id, title, text string) string {
	return "<!-- suggestion:" + id + " -->\n### " + title + "\n\n" + text + "\n\n"
}

// writeProposalBody writes work/proposal/body.md as the skill does: from the
// description the open PR has (existing, "" without one), kept as it is up to
// its impact assessment, then the new sections and this run's assessment.
func (r *rulesLoop) writeProposalBody(existing string, sections ...string) string {
	r.t.Helper()
	var impact impactOut
	mustLct(r.t, &impact, "impact", "--base", "origin/main", "--markdown", "work/proposal/impact.md")
	if impact.Units != 0 || impact.Checked == 0 {
		r.t.Errorf("impact = %+v, want the translations checked and none that the rules of the replay break", impact)
	}
	assessment := string(readFile(r.t, filepath.Join(r.root, "work", "proposal", "impact.md")))
	if !strings.HasPrefix(assessment, impactHeading+"\n") {
		r.t.Errorf("work/proposal/impact.md does not start with %q:\n%s", impactHeading, assessment)
	}
	body := proposalOpening
	if existing != "" {
		body = existing
		if i := strings.Index(body, impactHeading); i >= 0 {
			body = body[:i]
		}
	}
	body += strings.Join(sections, "") + assessment
	writeTestFile(r.t, r.root, proposalBodyRel, body)
	return body
}

// publishProposal is the last step of the skill: lct proposal publish.
func (r *rulesLoop) publishProposal() proposal.Published {
	r.t.Helper()
	var published proposal.Published
	mustLct(r.t, &published, "proposal", "publish", "--body-file", proposalBodyRel)
	return published
}

// openPRs are the open PRs of the fake GitHub.
func (r *rulesLoop) openPRs() []fakePR {
	r.t.Helper()
	var open []fakePR
	for _, p := range r.github().PRs {
		if p.State == "OPEN" {
			open = append(open, p)
		}
	}
	return open
}

// markerIDs are the ids of the markers that stand on a line of their own, as
// the last step of rules-proposal.yml reads them.
func markerIDs(body string) []string {
	var ids []string
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r", ""), "\n") {
		if id, ok := strings.CutPrefix(line, "<!-- suggestion:"); ok {
			if id, ok := strings.CutSuffix(id, " -->"); ok && id != "" {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// changedOnOrigin lists the files the branch of origin changes against its
// main.
func (r *rulesLoop) changedOnOrigin(branch string) []string {
	r.t.Helper()
	out := runGitIn(r.t, r.origin, "-c", "core.quotePath=false", "diff", "--name-only", "main..."+branch)
	return strings.Fields(out)
}

// The rules improvement loop (spec 11.2) as the workflows run it, from the
// suggestions of a build to a rules PR that is opened, updated, closed and at
// last merged into main, and the rebuild that applies it.
func TestRulesLoopReplay(t *testing.T) {
	r := newRulesLoop(t)
	candidates, base := candidateTerms(t, r.root) // base: the candidates the repo declined

	// 1. The two updates of the phase 2 replay, and a rebuild after the
	// second: the build of v2 finds the line of S1 and S2 translated in two
	// ways, the plan the new speaker and the adjustment that moved.
	inconsistentEvidence := rulesLoopBuildToV2(t, r)

	// 2. rules-proposal.yml runs: no rules PR is open, so the branch is new;
	// lct suggest gives the three suggestions, with ids in the form of
	// Task 3, and the repo's glossary candidates.
	started := r.start(rulesDay)
	if want := (proposal.Started{Branch: rulesBranch, Existing: []string{}, Declined: []string{}, Handled: []string{}}); !reflect.DeepEqual(started, want) {
		t.Errorf("first proposal start = %+v, want %+v", started, want)
	}
	if head, main := r.git("rev-parse", "HEAD"), r.originHead("main"); head != main {
		t.Errorf("the new branch is at %s, want origin's main %s", head, main)
	}
	if ex := r.readExclude(); len(ex) != 0 {
		t.Errorf("exclude.txt = %q, want it empty", ex)
	}
	if _, err := os.Stat(filepath.Join(r.root, proposal.ExistingBodyFile)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s is there without an open PR (%v)", proposal.ExistingBodyFile, err)
	}
	sug := r.suggestions()
	if got, want := suggestionIDsOf(sug), sortedIDs(loopIDs, candidates); !slices.Equal(got, want) || sug.Excluded != base {
		t.Fatalf("suggest = %q (%d excluded), want %q and none excluded but the %d candidates the repo declined", got, sug.Excluded, want, base)
	}
	checkRulesLoopSuggestions(t, sug, inconsistentEvidence)

	// 3. What the rules-review skill does with them: a guide for the new
	// speaker, a request to translate the line of S1 again as S2 has it, lct
	// adjust tidy for the moved adjustment, lint, the impact assessment, the
	// description; then lct proposal publish opens the rules PR.
	writeTestFile(t, r.root, newSpeakerGuide, newSpeakerGuideText)
	var written requeueWritten
	mustLct(t, &written, "requeue", "--to-file", "bus", "--key", p2Content(s1, 4), "--reason", busHint, "--date", rulesDay)
	if written.ID != rulesRequestID {
		t.Errorf("requeue --to-file = %+v, want the id %s", written, rulesRequestID)
	}
	var tidied tidyOut
	mustLct(t, &tidied, "adjust", "tidy")
	if !slices.Equal(tidied.Changed, []string{umbrellaFile}) || tidied.Moved != 1 || tidied.Removed != 0 {
		t.Errorf("adjust tidy = %+v, want %s changed, 1 moved", tidied, umbrellaFile)
	}
	mustLct(t, nil, "rules", "lint")
	calls := len(r.github().Calls)
	body := r.writeProposalBody("",
		proposalSection(inconsistentID, "不一致的譯法：`"+busKR+"`",
			"S1 第 5 行與 S2 第 1 行同一句，人工調整的 S2 譯為 `"+busZH+"`。附重翻要求檔 `rules/requeue/"+rulesRequestID+".yaml`：`"+p2Content(s1, 4)+"`。"),
		proposalSection(speakerID, "新說話者 `"+newSpeaker+"`", "新增 `"+newSpeakerGuide+"`。"),
		proposalSection(movedID, "調整的 key 已移動", "`lct adjust tidy` 改了 `"+umbrellaFile+"`。"),
	)
	published := r.publishProposal()
	if published.Branch != rulesBranch || published.URL != "https://github.com/o/r/pull/1" || !published.Created ||
		published.Commit != r.git("rev-parse", "HEAD") {
		t.Errorf("first proposal publish = %+v, want PR 1 opened from %s at HEAD", published, rulesBranch)
	}
	if got, want := r.ghCallsSince(calls), []string{"pr list", "label create", "pr create"}; !slices.Equal(got, want) {
		t.Errorf("gh calls of publish = %q, want %q", got, want)
	}
	open := r.openPRs()
	if len(open) != 1 || open[0].Number != 1 || open[0].HeadRefName != rulesBranch || open[0].BaseRefName != "main" ||
		open[0].Title != "規則建議："+rulesDay || !reflect.DeepEqual(open[0].Labels, []fakePRLabel{{proposal.Label}}) || open[0].Body != body {
		t.Errorf("open PRs = %+v, want PR 1 from %s with the label and the description", open, rulesBranch)
	}
	// What the last step of rules-proposal.yml checks: the PR holds
	// suggestions that exclude.txt, written before the run, does not.
	if len(open) == 1 {
		if added := sortedIDs(missingFrom(markerIDs(open[0].Body), r.readExclude())); !slices.Equal(added, loopIDs) {
			t.Errorf("the PR adds %q, want %q", added, loopIDs)
		}
	}
	if got := r.originHead(rulesBranch); got != published.Commit {
		t.Errorf("origin has %s at %q, want %s", rulesBranch, got, published.Commit)
	}
	if got, want := r.changedOnOrigin(rulesBranch), []string{newSpeakerGuide, umbrellaFile, "rules/requeue/" + rulesRequestID + ".yaml"}; !slices.Equal(got, sortedIDs(want)) {
		t.Errorf("the rules PR changes %q, want only the rule files %q", got, sortedIDs(want))
	}
	mainBefore := r.originHead("main")

	// 4. A person adds a glossary candidate on main; rebuild.yml builds again
	// and releases nothing, as the pack has not changed; rules-proposal.yml
	// runs again. The branch of the open PR is checked out with main merged
	// into it, and what the PR holds is not suggested again: only the new
	// candidate is. The skill turns it down in rules/declined.yaml, keeps the
	// description of the PR and adds its section; publish updates the PR.
	r.git("checkout", "-q", "main")
	appendTestFile(t, r.root, "glossary/terms.yaml", "\n- kr: "+stopKR+"\n  zh: 停靠站\n  status: candidate\n  category: [general]\n")
	mustLct(t, nil, "rules", "lint")
	r.commitMain("glossary: a candidate")
	if _, _, pub := r.rebuild(); !pub.Changed || pub.ReleaseNeeded || pub.SameAs != "zh-TW-20261105.2" {
		t.Errorf("rebuild after a glossary candidate: publish = %+v, want tw committed and no Release, as the pack is the one of zh-TW-20261105.2", pub)
	}
	again := r.start(rulesDay)
	if want := (proposal.Started{
		Branch: rulesBranch, PR: 1, URL: "https://github.com/o/r/pull/1", Existing: loopIDs,
		Declined: []string{}, Handled: []string{}, ExistingBody: proposal.ExistingBodyFile,
	}); !reflect.DeepEqual(again, want) {
		t.Errorf("second proposal start = %+v, want %+v", again, want)
	}
	existing := string(readFile(t, filepath.Join(r.root, proposal.ExistingBodyFile)))
	if existing != body {
		t.Errorf("%s = %q, want the description of the PR %q", proposal.ExistingBodyFile, existing, body)
	}
	if got := r.git("rev-parse", "HEAD^1", "HEAD^2"); got != published.Commit+"\n"+r.originHead("main") {
		t.Errorf("the branch of the open PR is at %s with parents %q, want origin's main merged into %s", r.git("rev-parse", "HEAD"), got, published.Commit)
	}
	if got := r.readExclude(); !slices.Equal(got, loopIDs) {
		t.Errorf("exclude.txt = %q, want the ids of the PR %q", got, loopIDs)
	}
	// The rules of the branch answer the speaker and the moved adjustment
	// already; the inconsistent translation is still found, and left out.
	sug = r.suggestions()
	if got, want := suggestionIDsOf(sug), sortedIDs([]string{stopTermID}, candidates); !slices.Equal(got, want) || sug.Excluded != base+1 {
		t.Errorf("suggest = %q (%d excluded), want %q and the inconsistent translation excluded", got, sug.Excluded, want)
	}
	declined := string(readFile(t, filepath.Join(r.root, declinedFile)))
	writeTestFile(t, r.root, declinedFile, strings.TrimSuffix(declined, "[]\n")+
		"- id: \""+stopTermID+"\"\n  reason: 只出現在一句台詞，不需要統一譯名\n  date: "+rulesDay+"\n")
	mustLct(t, nil, "rules", "lint")
	calls = len(r.github().Calls)
	body = r.writeProposalBody(existing, proposalSection(stopTermID, "詞彙候選：`"+stopKR+"`", "不採用，記在 `"+declinedFile+"`。"))
	updated := r.publishProposal()
	if updated.Branch != rulesBranch || updated.URL != published.URL || updated.Created || updated.Commit != r.git("rev-parse", "HEAD") {
		t.Errorf("second proposal publish = %+v, want PR 1 updated from HEAD", updated)
	}
	if got, want := r.ghCallsSince(calls), []string{"pr list", "pr edit"}; !slices.Equal(got, want) {
		t.Errorf("gh calls of the second publish = %q, want %q", got, want)
	}
	open = r.openPRs()
	if len(open) != 1 || open[0].Number != 1 || open[0].Body != body {
		t.Fatalf("open PRs = %+v, want PR 1 with the new description", open)
	}
	if got, want := markerIDs(open[0].Body), []string{inconsistentID, speakerID, movedID, stopTermID}; !slices.Equal(got, want) {
		t.Errorf("the PR has the suggestions %q, want %q", got, want)
	}
	if got := r.originHead(rulesBranch); got != updated.Commit {
		t.Errorf("origin has %s at %q, want %s", rulesBranch, got, updated.Commit)
	}
	if got, want := r.changedOnOrigin(rulesBranch), sortedIDs([]string{newSpeakerGuide, umbrellaFile, "rules/requeue/" + rulesRequestID + ".yaml", declinedFile}); !slices.Equal(got, want) {
		t.Errorf("the rules PR changes %q, want only the rule files %q", got, want)
	}
	if got := r.originHead("main"); got == mainBefore || got != r.git("rev-parse", "main") {
		t.Errorf("origin's main is at %s, want the commit of the glossary candidate %s and nothing of the PR", got, r.git("rev-parse", "main"))
	}

	// 5. A person closes the PR without merging it. The next run takes the
	// free branch rules-proposal/<day>-2, and every suggestion of the PR is
	// declined: none of them is suggested again.
	r.editGitHub(func(gh *fakeGitHub) { gh.PRs[0].State = "CLOSED" })
	closed := r.start(rulesDay)
	allIDs := sortedIDs(loopIDs, []string{stopTermID})
	if want := (proposal.Started{Branch: rulesBranch + "-2", Existing: []string{}, Declined: allIDs, Handled: []string{}}); !reflect.DeepEqual(closed, want) {
		t.Errorf("proposal start after the PR was closed = %+v, want %+v", closed, want)
	}
	if _, err := os.Stat(filepath.Join(r.root, proposal.ExistingBodyFile)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s is left after the PR was closed (%v)", proposal.ExistingBodyFile, err)
	}
	sug = r.suggestions()
	if got, want := suggestionIDsOf(sug), sortedIDs(candidates); !slices.Equal(got, want) || sug.Excluded != base+len(allIDs) {
		t.Errorf("suggest after the PR was closed = %q (%d excluded), want %q and the %d ids of the PR excluded", got, sug.Excluded, want, len(allIDs))
	}

	// 6. The person reopens the PR and merges it into main after all.
	// rebuild.yml runs: the request is applied once, and that build, which
	// changed rules and no translation, releases nothing. The unit asked for
	// is handed to the translator with the reason of the request.
	r.editGitHub(func(gh *fakeGitHub) {
		at := "2026-11-06T01:02:03Z"
		gh.PRs[0].State, gh.PRs[0].MergedAt = "MERGED", &at
	})
	r.git("checkout", "-q", "main")
	r.git("merge", "-q", "--no-ff", "-m", "Merge pull request #1 from "+rulesBranch, r.originHead(rulesBranch))
	r.pushMain()
	merged := r.git("rev-parse", "HEAD")
	sum, _, pub := r.rebuild()
	if !reflect.DeepEqual(sum.RequeueApplied, []plan.Applied{{ID: rulesRequestID, Units: 1}}) || sum.Pending != 1 || sum.RulesCommit != merged ||
		sum.Adjustments.Applied != 2 || len(sum.Adjustments.Moved) != 0 || len(sum.Adjustments.Expired) != 0 {
		t.Errorf("plan after the merge = %+v, want the request applied to 1 unit and both adjustments at their own key", sum)
	}
	wantItem := store.PendingItem{Key: p2Content(s1, 4), Reason: "requeue", PrevZH: "譯", Hint: busHint}
	if got := r.pending(); !reflect.DeepEqual(got, map[string]store.PendingItem{wantItem.Key: wantItem}) {
		t.Errorf("pending after the merge = %v, want only %+v", got, wantItem)
	}
	if !pub.Changed || pub.ReleaseNeeded || pub.SameAs != "zh-TW-20261105.2" {
		t.Errorf("publish after the merge = %+v, want tw committed and no Release, as the pack is the one of zh-TW-20261105.2", pub)
	}
	before := r.state()
	if replan := r.plan(); len(replan.RequeueApplied) != 0 || replan.Pending != 1 {
		t.Errorf("a second plan after the merge = %+v, want the request not applied again", replan)
	}
	if after := r.state(); !reflect.DeepEqual(after, before) {
		t.Errorf("a second plan after the merge changed the state:\nbefore %v\nafter  %v", before, after)
	}
	firstRun := len(r.batches())
	r.translate(1)
	if handed := handedOut(r.batches()[firstRun:]); len(handed) != 1 || handed[wantItem.Key].Reason != "requeue" || handed[wantItem.Key].Hint != busHint {
		t.Errorf("translated after the merge: %+v, want only %s with the reason of the request as its hint", handed, wantItem.Key)
	}

	// The next run of rules-proposal.yml after the merge: the inconsistent
	// translation is handled (its line is still translated in two ways until
	// the build after the translation), and the guide, the tidied adjustment
	// and the declined candidate answer the rest.
	final := r.start(rulesNextDay)
	if want := (proposal.Started{Branch: proposal.BranchPrefix + rulesNextDay, Existing: []string{}, Declined: []string{}, Handled: []string{inconsistentID}}); !reflect.DeepEqual(final, want) {
		t.Errorf("proposal start after the merge = %+v, want %+v", final, want)
	}
	sug = r.suggestions()
	if got, want := suggestionIDsOf(sug), sortedIDs(candidates); !slices.Equal(got, want) || sug.Excluded != base+2 {
		t.Errorf("suggest after the merge = %q (%d excluded), want %q, the handled and the declined id excluded", got, sug.Excluded, want)
	}
}

// rulesLoopBuildToV2 runs the two updates and the rebuild of step 1, and
// returns the evidence the suggestion of the inconsistent group of the last
// build has.
func rulesLoopBuildToV2(t *testing.T, r *rulesLoop) []suggest.Evidence {
	t.Helper()
	r.update("v1", testToken, testCreatedAt, phase2Raw1)
	var learned learnReport
	mustLct(t, &learned, "fields", "learn", "--write") // see TestPhase2Replay
	r.commitMain("rules: learn the field rules of the packs")
	r.plan()
	r.translate(14)
	r.build()
	if pub := r.publish(); !pub.ReleaseNeeded || pub.Tag != "zh-TW-20261029.1" {
		t.Errorf("first publish = %+v, want the Release zh-TW-20261029.1", pub)
	}

	// A person adjusts the line of S1 that v2 moves down one line.
	var added adjustAdded
	mustLct(t, &added, "adjust", "add", "--key", p2Content(s1, 2), "--zh", adjKeptZH,
		"--reason", "以實瑪利的口吻", "--rule", "one-off", "--topic", "umbrella", "--date", "2026-11-03")
	r.commitMain("rules: adjust S1")

	r.update("v2", phase2Token2, phase2CreatedAt2, phase2Raw2)
	sum := r.plan()
	if !slices.Equal(sum.NewSpeakers, []string{newSpeaker}) || !reflect.DeepEqual(sum.Adjustments.Moved, []plan.MovedKey{{From: p2Content(s1, 2), To: p2Content(s1, 3)}}) {
		t.Errorf("plan of v2 = %+v, want the new speaker %s and the adjustment moved to %s", sum, newSpeaker, p2Content(s1, 3))
	}
	r.translate(sum.Pending)
	r.build()
	if pub := r.publish(); !pub.ReleaseNeeded || pub.Tag != "zh-TW-20261105.1" {
		t.Errorf("second publish = %+v, want the Release zh-TW-20261105.1", pub)
	}

	// A person adjusts the first line of S2, which is the fifth line of S1:
	// rebuild.yml builds again, and the pack has changed.
	mustLct(t, &added, "adjust", "add", "--key", p2Content(s2, 0), "--zh", busZH,
		"--reason", "巴士用「發車」", "--rule", "one-off", "--topic", "bus-s2", "--date", rulesDay)
	r.commitMain("rules: adjust S2")
	sum, built, pub := r.rebuild()
	if !slices.Equal(sum.NewSpeakers, []string{newSpeaker}) || len(sum.Adjustments.Moved) != 1 || sum.Adjustments.Applied != 1 || sum.Pending != 0 {
		t.Errorf("rebuild plan of v2 = %+v, want the new speaker and the moved adjustment kept, the adjustment of S2 applied", sum)
	}
	if !pub.ReleaseNeeded || pub.Tag != "zh-TW-20261105.2" {
		t.Errorf("rebuild publish = %+v, want the Release zh-TW-20261105.2", pub)
	}
	want := []suggest.Evidence{{Key: p2Content(s2, 0), KR: busKR, ZH: busZH}, {Key: p2Content(s1, 4), KR: busKR, ZH: "譯"}}
	if built.Report == nil || len(built.Report.Inconsistent) != 1 {
		t.Fatalf("rebuild report = %+v, want one inconsistent group", built.Report)
	}
	g := built.Report.Inconsistent[0]
	if g.Family != "story-main" || g.Field != "content" || g.KR != busKR || g.Units != 2 || len(g.ZH) != 2 {
		t.Errorf("inconsistent group = %+v, want %q in story-main content, translated as 譯 and %s", g, busKR, busZH)
	}
	return want
}

// checkRulesLoopSuggestions checks what the three suggestions show.
func checkRulesLoopSuggestions(t *testing.T, rep suggestReport, inconsistent []suggest.Evidence) {
	t.Helper()
	byID := map[string]suggest.Suggestion{}
	for _, s := range rep.Suggestions {
		byID[s.ID] = s
	}
	if s := byID[inconsistentID]; s.Kind != suggest.KindInconsistent || !reflect.DeepEqual(s.Evidence, inconsistent) {
		t.Errorf("%s = %+v, want the two translations %+v", inconsistentID, s, inconsistent)
	}
	if s := byID[speakerID]; s.Kind != suggest.KindSpeaker || !reflect.DeepEqual(s.Evidence, []suggest.Evidence{{KR: newSpeaker}}) {
		t.Errorf("%s = %+v, want the model as evidence", speakerID, s)
	}
	wantMoved := []suggest.Evidence{{Key: p2Content(s1, 2), ZH: adjKeptZH}, {Key: p2Content(s1, 3)}}
	if s := byID[movedID]; s.Kind != suggest.KindAdjustmentMoved || !reflect.DeepEqual(s.Evidence, wantMoved) || !strings.Contains(s.Title, umbrellaFile) {
		t.Errorf("%s = %+v, want the old and the new key and %s", movedID, s, umbrellaFile)
	}
}

// missingFrom returns the ids of have that skip does not hold, in order.
func missingFrom(have, skip []string) []string {
	var out []string
	for _, id := range have {
		if !slices.Contains(skip, id) {
			out = append(out, id)
		}
	}
	return out
}

func appendTestFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

// newSpeakerGuideText is the guide the skill writes for the new speaker.
const newSpeakerGuideText = `---
id: mona
name: 莫娜
name_kr: 모나
gender: female
number: null
code: null
models: [모나]
models_other_self: []
models_not_this: []
file_names: [Mona]
---

# 莫娜

## 決定

- 對大家說話客氣，請求時用「請」。
`
