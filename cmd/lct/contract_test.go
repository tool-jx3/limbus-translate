package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/publish"
)

// The contract between the skills, the shared prompts and the rest of the
// repo: a skill's frontmatter names it, what its body refers to exists, and
// no skill or prompt names a model (the model comes from --model or
// LCT_MODEL). The permissions the repo grants (.claude/settings.json) and
// the tools the translate-pending skill may use are scoped too: Claude Code
// adds the two up, so a broad entry in either undoes the scoping of both.

var (
	// repoPathRe matches a repo-relative path in a skill's body.
	repoPathRe = regexp.MustCompile(`\b(?:prompts|guides|schemas)/[A-Za-z0-9_./-]+`)
	// lctRefRe matches a run of bin/lct: the flags before the command word,
	// then the one or two words that can name a command. "Bash(bin/lct *)"
	// and "go build -o bin/lct ./cmd/lct" run no command.
	lctRefRe = regexp.MustCompile(`bin/lct((?: --?[a-z][a-z-]*)*) ([a-z][a-z-]*)(?: ([a-z][a-z-]*))?`)
	// lctGlobalFlags are the flags that may come before the command word:
	// cli.Main takes --json anywhere, while any other flag there leaves it
	// without a command.
	lctGlobalFlags = []string{"--json"}
	// modelNameRe matches a model name.
	modelNameRe = regexp.MustCompile(`(?i)\b(opus|sonnet|haiku|fable|claude-[a-z0-9.-]+)\b`)
)

// skillFrontmatter is the part of a SKILL.md frontmatter the contract checks.
type skillFrontmatter struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	AllowedTools []string `yaml:"allowed-tools"`
}

// translatePendingTools are the tools of the translate-pending skill (R35):
// batch text is untrusted, so it may write nothing outside work/ and run no
// lct command beyond batch and pending.
var translatePendingTools = []string{
	"Bash(bin/lct batch *)",
	"Bash(bin/lct pending *)",
	"Bash(go build -o bin/lct ./cmd/lct)",
	"Read(./**)",
	"Edit(./work/**)",
}

// lctWordRe is one word of an lct command name.
var lctWordRe = regexp.MustCompile(`^[a-z][a-z-]*$`)

// checkPermissions reports the entries of a permission list (the allow list
// of a settings file, or a skill's allowed-tools) that grant too much:
//   - a bare Read, Edit, Write or Bash, or one whose specifier is only * or
//     **, which allows every file or command;
//   - a Bash(bin/lct ...) that names no registered command: the words before
//     the closing * must be a command (two words, else the first, as
//     checkLctCommands reads them) or, as a single word, the group of
//     two-word commands it starts (batch for batch next and batch submit).
//     Bash(bin/lct *) allows every command and is refused.
func checkPermissions(file string, entries, commands []string) []string {
	var problems []string
	for _, e := range entries {
		tool, spec, scoped := strings.Cut(e, "(")
		spec = strings.TrimSuffix(spec, ")")
		if slices.Contains([]string{"Read", "Edit", "Write", "Bash"}, tool) && (!scoped || spec == "*" || spec == "**") {
			problems = append(problems, fmt.Sprintf("%s: %q grants %s without a scope", file, e, tool))
			continue
		}
		rest, isLct := strings.CutPrefix(spec, "bin/lct")
		if tool != "Bash" || !isLct {
			continue
		}
		words := strings.Fields(strings.TrimSuffix(strings.TrimSpace(rest), "*"))
		if len(words) == 0 {
			problems = append(problems, fmt.Sprintf("%s: %q allows every lct command", file, e))
			continue
		}
		if slices.ContainsFunc(words, func(w string) bool { return !lctWordRe.MatchString(w) }) {
			problems = append(problems, fmt.Sprintf("%s: %q does not start with an lct command", file, e))
			continue
		}
		group := len(words) == 1 && slices.ContainsFunc(commands, func(c string) bool { return strings.HasPrefix(c, words[0]+" ") })
		if (len(words) > 1 && slices.Contains(commands, words[0]+" "+words[1])) || slices.Contains(commands, words[0]) || group {
			continue
		}
		problems = append(problems, fmt.Sprintf("%s: %q names no registered lct command", file, e))
	}
	return problems
}

// splitFrontmatter returns the YAML between the leading "---" lines of a
// SKILL.md and the body after them; ok is false when there is no such block.
func splitFrontmatter(content string) (front, body string, ok bool) {
	lines := strings.SplitAfter(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	if strings.TrimSuffix(lines[0], "\n") != "---" {
		return "", "", false
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSuffix(lines[i], "\n") == "---" {
			return strings.Join(lines[1:i], ""), strings.Join(lines[i+1:], ""), true
		}
	}
	return "", "", false
}

// checkSkill checks the SKILL.md file (a slash path relative to root, in
// .claude/skills/<name>/) whose text is content: its frontmatter, the repo
// paths its body refers to and the lct commands its body runs. commands are
// the registered command names. Each problem names the file and what is
// wrong.
func checkSkill(root, file, content string, commands []string) []string {
	front, body, ok := splitFrontmatter(content)
	if !ok {
		return []string{file + ": no frontmatter between leading --- lines"}
	}
	var problems []string
	var fm skillFrontmatter
	if err := yaml.Unmarshal([]byte(front), &fm); err != nil {
		problems = append(problems, fmt.Sprintf("%s: frontmatter is not valid YAML: %v", file, err))
	} else {
		if dir := path.Base(path.Dir(file)); strings.TrimSpace(fm.Name) == "" {
			problems = append(problems, file+": frontmatter has no name")
		} else if fm.Name != dir {
			problems = append(problems, fmt.Sprintf("%s: frontmatter name %q is not the directory name %q", file, fm.Name, dir))
		}
		if strings.TrimSpace(fm.Description) == "" {
			problems = append(problems, file+": frontmatter has no description")
		}
	}
	problems = append(problems, checkRepoPaths(root, file, body)...)
	return append(problems, checkLctCommands(file, body, commands)...)
}

// checkRepoPaths reports the paths under prompts/, guides/ and schemas/ in
// body that do not exist under root.
func checkRepoPaths(root, file, body string) []string {
	var problems []string
	seen := map[string]bool{}
	for _, m := range repoPathRe.FindAllString(body, -1) {
		p := strings.TrimRight(m, ".") // the full stop of a sentence
		if seen[p] {
			continue
		}
		seen[p] = true
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(p))); err != nil {
			problems = append(problems, fmt.Sprintf("%s: refers to %s, which does not exist", file, p))
		}
	}
	return problems
}

// checkLctCommands reports the runs of bin/lct in body that name no
// registered command: a flag other than a global one comes before the
// command word, or neither the two words there nor the first one is a
// command.
func checkLctCommands(file, body string, commands []string) []string {
	var problems []string
	for _, m := range lctRefRe.FindAllStringSubmatch(body, -1) {
		flags, first, second := strings.Fields(m[1]), m[2], m[3]
		if i := slices.IndexFunc(flags, func(f string) bool { return !slices.Contains(lctGlobalFlags, f) }); i >= 0 {
			problems = append(problems, fmt.Sprintf("%s: %q puts %s before the command, where only %s may come",
				file, m[0], flags[i], strings.Join(lctGlobalFlags, ", ")))
			continue
		}
		if (second != "" && slices.Contains(commands, first+" "+second)) || slices.Contains(commands, first) {
			continue
		}
		problems = append(problems, fmt.Sprintf("%s: %q runs no registered lct command", file, m[0]))
	}
	return problems
}

// checkModelNames reports every model name in the files under dirs (slash
// paths relative to root), and a directory that cannot be walked.
func checkModelNames(root string, dirs ...string) []string {
	var problems []string
	for _, dir := range dirs {
		err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(dir)), func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			for _, name := range modelNameRe.FindAllString(string(data), -1) {
				problems = append(problems, fmt.Sprintf("%s: names a model: %q", filepath.ToSlash(rel), name))
			}
			return nil
		})
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", dir, err))
		}
	}
	return problems
}

func TestSkillContract(t *testing.T) {
	skills, err := filepath.Glob(filepath.Join(repoRoot, ".claude", "skills", "*", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(skills, filepath.Join(repoRoot, ".claude", "skills", "translate-pending", "SKILL.md")) {
		t.Error(".claude/skills/translate-pending/SKILL.md does not exist")
	}
	commands := cli.Commands()
	for _, p := range skills {
		rel, content := readRepoFile(t, p)
		for _, problem := range checkSkill(repoRoot, rel, content, commands) {
			t.Error(problem)
		}
	}
	// The shared prompts point at guides and schemas too.
	prompts, err := filepath.Glob(filepath.Join(repoRoot, "prompts", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range prompts {
		rel, content := readRepoFile(t, p)
		for _, problem := range checkRepoPaths(repoRoot, rel, content) {
			t.Error(problem)
		}
	}
	for _, problem := range checkModelNames(repoRoot, ".claude/skills", "prompts") {
		t.Error(problem)
	}
}

// The repo's .claude/settings.json grants what the skills run and nothing
// broader: other lct commands, and files outside the repo, ask a person.
func TestSettingsContract(t *testing.T) {
	_, content := readRepoFile(t, filepath.Join(repoRoot, ".claude", "settings.json"))
	var settings struct {
		Permissions struct {
			Allow []string `json:"allow"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal([]byte(content), &settings); err != nil {
		t.Fatalf(".claude/settings.json does not parse: %v", err)
	}
	if len(settings.Permissions.Allow) == 0 {
		t.Fatal(".claude/settings.json allows nothing: permissions.allow is missing")
	}
	for _, problem := range checkPermissions(".claude/settings.json", settings.Permissions.Allow, cli.Commands()) {
		t.Error(problem)
	}
}

// A unit can carry a hint (lct requeue): the schema declares it and the
// shared prompt tells the translator what to do with it.
func TestBatchSchemaDeclaresHint(t *testing.T) {
	_, content := readRepoFile(t, filepath.Join(repoRoot, "schemas", "batch.schema.json"))
	var schema struct {
		Properties struct {
			Units struct {
				Items struct {
					Properties map[string]struct {
						Type string `json:"type"`
					} `json:"properties"`
				} `json:"items"`
			} `json:"units"`
		} `json:"properties"`
	}
	if err := json.Unmarshal([]byte(content), &schema); err != nil {
		t.Fatalf("schemas/batch.schema.json does not parse: %v", err)
	}
	if hint, ok := schema.Properties.Units.Items.Properties["hint"]; !ok || hint.Type != "string" {
		t.Errorf("schemas/batch.schema.json: units.items.properties.hint = %+v (declared %v), want a string property", hint, ok)
	}
	if _, prompt := readRepoFile(t, filepath.Join(repoRoot, "prompts", "translate.md")); !strings.Contains(prompt, "`hint`") {
		t.Error("prompts/translate.md does not tell the translator what to do with `hint`")
	}
}

// The translate-pending skill may use exactly the R35 tools.
func TestTranslatePendingTools(t *testing.T) {
	_, content := readRepoFile(t, filepath.Join(repoRoot, ".claude", "skills", "translate-pending", "SKILL.md"))
	front, _, ok := splitFrontmatter(content)
	if !ok {
		t.Fatal("translate-pending/SKILL.md has no frontmatter")
	}
	var fm skillFrontmatter
	if err := yaml.Unmarshal([]byte(front), &fm); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(fm.AllowedTools, translatePendingTools) {
		t.Errorf("allowed-tools = %q, want %q", fm.AllowedTools, translatePendingTools)
	}
	for _, problem := range checkPermissions("translate-pending/SKILL.md", fm.AllowedTools, cli.Commands()) {
		t.Error(problem)
	}
}

func TestCheckPermissions(t *testing.T) {
	commands := []string{"batch next", "batch submit", "bootstrap", "pending", "rules lint", "validate"}
	good := []string{
		"Bash(bin/lct batch *)", "Bash(bin/lct pending *)", "Bash(bin/lct validate *)", "Bash(bin/lct rules lint*)",
		"Bash(bin/lct batch submit *)", "Bash(go build -o bin/lct ./cmd/lct)", "Read(./**)", "Edit(./work/**)",
		"Write(./work/**)", "WebFetch(domain:example.com)", "Glob",
	}
	if problems := checkPermissions("settings.json", good, commands); len(problems) > 0 {
		t.Errorf("problems in a good list:\n%s", strings.Join(problems, "\n"))
	}
	for _, tc := range []struct{ entry, want string }{
		{"Bash(bin/lct *)", "every lct command"},
		{"Bash(bin/lct)", "every lct command"},
		{"Bash(bin/lct --json *)", "does not start with an lct command"},
		{"Bash(bin/lct frobnicate *)", "no registered"},
		{"Bash(bin/lct batch nxt *)", "no registered"},
		{"Bash(bin/lct fields *)", "no registered"}, // no fields command in this list
		{"Read", "without a scope"},
		{"Edit", "without a scope"},
		{"Write", "without a scope"},
		{"Bash", "without a scope"},
		{"Bash(*)", "without a scope"},
		{"Read(**)", "without a scope"},
	} {
		t.Run(tc.entry, func(t *testing.T) {
			problems := checkPermissions("settings.json", []string{tc.entry}, commands)
			if len(problems) != 1 || !strings.Contains(problems[0], tc.want) || !strings.Contains(problems[0], "settings.json") {
				t.Errorf("problems = %q, want one naming settings.json and %q", problems, tc.want)
			}
		})
	}
}

// readRepoFile returns the slash path relative to repoRoot of the file at p,
// and its text.
func readRepoFile(t *testing.T, p string) (rel, content string) {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	rel, err = filepath.Rel(repoRoot, p)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.ToSlash(rel), string(data)
}

// contractCommands stand in for the registered commands in the tests of the
// checkers.
var contractCommands = []string{"batch next", "batch submit", "pending"}

func TestCheckSkillAcceptsGoodSkill(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "prompts/translate.md", "x\n")
	writeTestFile(t, root, "schemas/batch.schema.json", "{}\n")
	content := "---\nname: demo\ndescription: 示範\nallowed-tools:\n  - Bash(bin/lct *)\n---\n" +
		"讀 `prompts/translate.md`，格式見 schemas/batch.schema.json。\n" +
		"若 `bin/lct` 不存在，執行 `go build -o bin/lct ./cmd/lct`；允許 `Bash(bin/lct *)`。\n" +
		"執行 `bin/lct batch next --json`、`bin/lct batch submit <id> --json` 與 `bin/lct pending --count`。\n" +
		"`bin/lct --json batch next` 也可以。\n" +
		"不改 `guides/`。\n"
	if problems := checkSkill(root, ".claude/skills/demo/SKILL.md", content, contractCommands); len(problems) > 0 {
		t.Errorf("problems in a good skill:\n%s", strings.Join(problems, "\n"))
	}
}

func TestCheckSkillFindsProblems(t *testing.T) {
	const file = ".claude/skills/demo/SKILL.md"
	const front = "---\nname: demo\ndescription: 示範\n---\n"
	root := t.TempDir()
	writeTestFile(t, root, "prompts/translate.md", "x\n")
	for _, tc := range []struct {
		name, content, want string
	}{
		{"no frontmatter", "# demo\n", "no frontmatter"},
		{"unclosed frontmatter", "---\nname: demo\ndescription: 示範\n", "no frontmatter"},
		{"invalid YAML", "---\nname: [demo\n---\n", "not valid YAML"},
		{"no name", "---\ndescription: 示範\n---\n", "no name"},
		{"no description", "---\nname: demo\ndescription: \"\"\n---\n", "no description"},
		{"name is not the directory", "---\nname: other\ndescription: 示範\n---\n", `"other"`},
		{"missing prompt", front + "讀 `prompts/missing.md`。\n", "prompts/missing.md"},
		{"missing guide", front + "見 guides/nope.md.\n", "guides/nope.md"},
		{"missing schema", front + "格式見 `schemas/out.schema.json`）\n", "schemas/out.schema.json"},
		{"unknown command", front + "執行 `bin/lct frobnicate --json`\n", `"bin/lct frobnicate"`},
		{"unknown subcommand", front + "執行 `bin/lct batch nxt <id>`\n", `"bin/lct batch nxt"`},
		{"command group only", front + "執行 bin/lct batch <id>\n", `"bin/lct batch"`},
		{"unknown command after a global flag", front + "執行 `bin/lct --json batch nxt`\n", `"bin/lct --json batch nxt"`},
		{"flag that is not global before the command", front + "執行 `bin/lct --count pending`\n", "--count"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			problems := checkSkill(root, file, tc.content, contractCommands)
			if len(problems) != 1 {
				t.Fatalf("got %d problems, want 1:\n%s", len(problems), strings.Join(problems, "\n"))
			}
			if !strings.Contains(problems[0], file) || !strings.Contains(problems[0], tc.want) {
				t.Errorf("problem %q does not name %s and %s", problems[0], file, tc.want)
			}
		})
	}
}

func TestCheckModelNames(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".claude/skills/demo/SKILL.md", "由 Claude 翻譯；不要寫死 Opus。\n")
	writeTestFile(t, root, "prompts/model.md", "預設：claude-example-9。\n")
	writeTestFile(t, root, "prompts/clean.md", "Claude 與 claude 可以單獨出現；opuses、haikus 不是模型名稱。\n")

	problems := checkModelNames(root, ".claude/skills", "prompts")
	want := [][2]string{
		{".claude/skills/demo/SKILL.md", `"Opus"`},
		{"prompts/model.md", `"claude-example-9"`},
	}
	if len(problems) != len(want) {
		t.Fatalf("got %d problems, want %d:\n%s", len(problems), len(want), strings.Join(problems, "\n"))
	}
	for i, w := range want {
		if !strings.Contains(problems[i], w[0]) || !strings.Contains(problems[i], w[1]) {
			t.Errorf("problem %q does not name %s and %s", problems[i], w[0], w[1])
		}
	}

	if problems := checkModelNames(root, "missing"); len(problems) != 1 || !strings.Contains(problems[0], "missing") {
		t.Errorf("a missing directory: got %q, want one problem naming it", problems)
	}
}

// The workflows are part of the contract too: what they run exists, they name
// no model, hold no secret and give the Release no forbidden name, and the
// manual rebuild is built so that a fresh clone numbers and publishes the
// Release right (the raw worktree and every tag are there before it runs).

var (
	// workflowActions are the only actions a workflow may use, pinned to the
	// major version.
	workflowActions = []string{"actions/checkout@v4", "actions/setup-go@v5"}
	// workflowTitleRe matches the value of a --title flag.
	workflowTitleRe = regexp.MustCompile(`--title(?:\s+|=)("[^"]*"|'[^']*'|\S+)`)
	// forbiddenNameRe matches the names a Release must not carry.
	forbiddenNameRe = regexp.MustCompile(`(?i)claude\s*code|anthropic`)
	// workflowSecretRe matches a secret, or the status API URL of the game
	// (the environment variable of lct fetch), which must not be in a
	// workflow.
	workflowSecretRe = regexp.MustCompile(`secrets\.|LIMBUS_STATUS_URL`)
)

// rebuildMarkers are the commands rebuild.yml must run, with the reason.
var rebuildMarkers = []struct{ text, why string }{
	{"+refs/heads/raw:refs/remotes/origin/raw", "the raw branch is not fetched"},
	{"+refs/heads/tw:refs/remotes/origin/tw", "the tw branch is not fetched"},
	{"+refs/tags/*:refs/tags/*", "the tags are not fetched, so the Release number restarts at 1"},
	{"git worktree add .work/raw raw", "lct build reads the raw worktree and does not make it"},
	{"meta/source.json", "the raw tag is not read from the raw worktree"},
	// The answer is assigned first, so that set -e sees a failing git.
	{"remote=$(git ls-remote --tags origin", "origin is not asked for the raw tag, or its answer is not kept before it is tested"},
	{`git push origin "refs/tags/`, "a raw tag that origin lacks is not pushed"},
	{"bin/lct build", "the pack is not built"},
	{"bin/lct publish", "the Release is not prepared"},
	{"git push origin tw", "the tw commit is not pushed"},
	{"gh release create", "no Release is created"},
	{`--target "$tw_commit"`, "the Release tag is not made at the tw commit"},
	{"--notes-file", "the Release has no notes"},
	// set -e does not see a failing command in a process substitution: the
	// assets are captured first, then read, then counted, or a Release could
	// be made with no zip.
	{"assets_json=$(jq -ec", "the assets of publish.json are not captured before they are read"},
	{"mapfile -t assets", "the assets are not read"},
	{`"${#assets[@]}" -eq 2`, "the number of assets is not checked before the Release"},
}

// rebuildOrder pairs the commands of rebuild.yml that must run in that order:
// the first before the second, by the first place each is written.
var rebuildOrder = [][2]string{
	{"+refs/tags/*:refs/tags/*", "bin/lct publish"}, // the tags number the Release
	{"git worktree add .work/raw raw", "meta/source.json"},
	{"meta/source.json", "remote=$(git ls-remote --tags origin"},
	{"git worktree add .work/raw raw", "bin/lct build"},
	{"bin/lct build", "bin/lct publish"},
	{"bin/lct publish", "git push origin tw"},
	{"git push origin tw", "gh release create"}, // the Release is made from a pushed commit
	{"assets_json=$(jq -ec", "mapfile -t assets"},
	{"mapfile -t assets", `"${#assets[@]}" -eq 2`},
	{`"${#assets[@]}" -eq 2`, "gh release create"}, // no Release without its zips
}

// workflowStep is a step of a job, as far as the contract reads it.
type workflowStep struct {
	Uses string            `yaml:"uses"`
	Run  string            `yaml:"run"`
	With map[string]any    `yaml:"with"`
	Env  map[string]string `yaml:"env"`
}

// workflowFile is the part of a workflow file the contract reads.
type workflowFile struct {
	On          any               `yaml:"on"`
	Permissions map[string]string `yaml:"permissions"`
	Concurrency struct {
		Group            string `yaml:"group"`
		CancelInProgress bool   `yaml:"cancel-in-progress"`
	} `yaml:"concurrency"`
	Jobs map[string]struct {
		Steps []workflowStep `yaml:"steps"`
	} `yaml:"jobs"`
}

// steps returns the steps of all jobs, the jobs in name order.
func (w *workflowFile) steps() []workflowStep {
	names := make([]string, 0, len(w.Jobs))
	for name := range w.Jobs {
		names = append(names, name)
	}
	sort.Strings(names)
	var steps []workflowStep
	for _, name := range names {
		steps = append(steps, w.Jobs[name].Steps...)
	}
	return steps
}

// runs returns the shell text of all steps, in order.
func (w *workflowFile) runs() string {
	var b strings.Builder
	for _, s := range w.steps() {
		if s.Run != "" {
			b.WriteString(s.Run)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// triggers returns the events the workflow runs on, whether "on" is written
// as one event, a list or a map.
func (w *workflowFile) triggers() []string {
	var events []string
	switch on := w.On.(type) {
	case string:
		events = []string{on}
	case []any:
		for _, e := range on {
			events = append(events, fmt.Sprint(e))
		}
	case map[string]any:
		for e := range on {
			events = append(events, e)
		}
	}
	sort.Strings(events)
	return events
}

// parseWorkflow decodes a workflow file. A file that is not YAML, or has no
// job with a step, gives the problems and no workflow.
func parseWorkflow(file, content string) (*workflowFile, []string) {
	var w workflowFile
	if err := yaml.Unmarshal([]byte(content), &w); err != nil {
		return nil, []string{fmt.Sprintf("%s: not valid YAML: %v", file, err)}
	}
	if len(w.steps()) == 0 {
		return nil, []string{file + ": has no job with a step"}
	}
	return &w, nil
}

// checkWorkflow checks a workflow file (a slash path relative to the root)
// whose text is content: that it parses, that the lct commands it runs are
// registered, that it uses only the pinned actions, that it holds no secret
// and that the --title of a Release has no forbidden name.
func checkWorkflow(file, content string, commands []string) []string {
	w, problems := parseWorkflow(file, content)
	if w == nil {
		return problems
	}
	problems = append(problems, checkLctCommands(file, content, commands)...)
	for _, s := range w.steps() {
		if s.Uses != "" && !slices.Contains(workflowActions, s.Uses) {
			problems = append(problems, fmt.Sprintf("%s: uses %q, which is not one of %s", file, s.Uses, strings.Join(workflowActions, ", ")))
		}
	}
	for _, m := range workflowSecretRe.FindAllString(content, -1) {
		problems = append(problems, fmt.Sprintf("%s: refers to %q: a workflow uses only the token of the run", file, m))
	}
	for _, m := range workflowTitleRe.FindAllStringSubmatch(content, -1) {
		if forbiddenNameRe.MatchString(m[1]) {
			problems = append(problems, fmt.Sprintf("%s: --title %s names the Release with a name it must not carry", file, m[1]))
		}
	}
	return problems
}

// checkRebuildWorkflow checks the manual rebuild: it runs on request, one at
// a time, may write to the repo, fetches the history and every tag, makes the
// raw worktree, gives origin the raw tag, and builds, publishes and releases
// in that order.
func checkRebuildWorkflow(file, content string) []string {
	w, problems := parseWorkflow(file, content)
	if w == nil {
		return problems
	}
	if !slices.Contains(w.triggers(), "workflow_dispatch") {
		problems = append(problems, file+": does not run on workflow_dispatch")
	}
	if w.Permissions["contents"] != "write" {
		problems = append(problems, fmt.Sprintf("%s: permissions.contents = %q, want write", file, w.Permissions["contents"]))
	}
	if w.Concurrency.Group != "lct-state" || w.Concurrency.CancelInProgress {
		problems = append(problems, fmt.Sprintf("%s: concurrency = {group: %q, cancel-in-progress: %v}, want {group: \"lct-state\", cancel-in-progress: false}",
			file, w.Concurrency.Group, w.Concurrency.CancelInProgress))
	}
	deep := slices.ContainsFunc(w.steps(), func(s workflowStep) bool {
		return strings.HasPrefix(s.Uses, "actions/checkout@") && fmt.Sprint(s.With["fetch-depth"]) == "0"
	})
	if !deep {
		problems = append(problems, file+": no actions/checkout step with fetch-depth: 0")
	}

	runs := w.runs()
	for _, m := range rebuildMarkers {
		if !strings.Contains(runs, m.text) {
			problems = append(problems, fmt.Sprintf("%s: no %q: %s", file, m.text, m.why))
		}
	}
	for _, o := range rebuildOrder {
		first, second := strings.Index(runs, o[0]), strings.Index(runs, o[1])
		if first >= 0 && second >= 0 && first > second {
			problems = append(problems, fmt.Sprintf("%s: %q must come before %q", file, o[0], o[1]))
		}
	}
	// The step that releases takes the token of the run and names the Release.
	if i := slices.IndexFunc(w.steps(), func(s workflowStep) bool { return strings.Contains(s.Run, "gh release create") }); i >= 0 {
		step := w.steps()[i]
		if step.Env["GH_TOKEN"] != "${{ github.token }}" {
			problems = append(problems, file+": the gh release create step has no GH_TOKEN: ${{ github.token }}")
		}
		if !workflowTitleRe.MatchString(step.Run) {
			problems = append(problems, file+": gh release create has no --title")
		}
	}
	return problems
}

// publishKeyRe matches a key of out/publish.json that a jq command of the
// workflow reads: the key after "jq -er ." or "jq -ec '.".
var publishKeyRe = regexp.MustCompile(`jq -[a-z]+ '?\.([a-z_]+)`)

// publishKeysRead are the keys of out/publish.json the rebuild must read.
var publishKeysRead = []string{"tag", "tw_commit", "assets"}

// checkPublishKeys reports the keys the workflow reads from out/publish.json
// that result, the value lct publish --json prints, has no JSON key for, and
// the keys in publishKeysRead it does not read at all (so the test cannot
// pass by finding nothing).
func checkPublishKeys(file, content string, result any) []string {
	data, err := json.Marshal(result)
	if err != nil {
		return []string{fmt.Sprintf("%s: the publish result does not marshal: %v", file, err)}
	}
	var printed map[string]json.RawMessage
	if err := json.Unmarshal(data, &printed); err != nil {
		return []string{fmt.Sprintf("%s: the publish result is not a JSON object: %v", file, err)}
	}
	var problems []string
	read := map[string]bool{}
	for _, line := range strings.Split(content, "\n") {
		if !strings.Contains(line, "out/publish.json") {
			continue
		}
		for _, m := range publishKeyRe.FindAllStringSubmatch(line, -1) {
			read[m[1]] = true
		}
	}
	for _, key := range publishKeysRead {
		if !read[key] {
			problems = append(problems, fmt.Sprintf("%s: does not read %q from out/publish.json", file, key))
		}
	}
	for key := range read {
		if _, ok := printed[key]; !ok {
			problems = append(problems, fmt.Sprintf("%s: reads %q from out/publish.json, which lct publish --json does not print", file, key))
		}
	}
	sort.Strings(problems)
	return problems
}

// mutate returns src with the one copy of old replaced by repl; a workflow
// that has no copy or many fails the test, so a negative test cannot pass
// without changing anything.
func mutate(t *testing.T, src, old, repl string) string {
	t.Helper()
	if n := strings.Count(src, old); n != 1 {
		t.Fatalf("the workflow has %d copies of %q, want 1: update the test", n, old)
	}
	return strings.Replace(src, old, repl, 1)
}

func TestWorkflowContract(t *testing.T) {
	commands := cli.Commands()
	// Every workflow is under the contract; the two of this repo must exist.
	all, err := filepath.Glob(filepath.Join(repoRoot, ".github", "workflows", "*.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ci.yml", "rebuild.yml"} {
		if !slices.Contains(all, filepath.Join(repoRoot, ".github", "workflows", want)) {
			t.Errorf(".github/workflows/%s does not exist", want)
		}
	}
	var ci string
	for _, p := range all {
		rel, content := readRepoFile(t, p)
		for _, problem := range checkWorkflow(rel, content, commands) {
			t.Error(problem)
		}
		if path.Base(rel) == "ci.yml" {
			ci = content
		}
	}
	for _, problem := range checkModelNames(repoRoot, ".github/workflows") {
		t.Error(problem)
	}
	file, rebuild := readRepoFile(t, filepath.Join(repoRoot, ".github", "workflows", "rebuild.yml"))
	for _, problem := range checkRebuildWorkflow(file, rebuild) {
		t.Error(problem)
	}
	if t.Failed() {
		return
	}

	// 1. Every bin/lct a workflow runs is a registered command.
	t.Run("an unregistered lct command", func(t *testing.T) {
		src := mutate(t, rebuild, "bin/lct publish --json", "bin/lct publsh --json")
		problems := checkWorkflow(file, src, commands)
		if len(problems) != 1 || !strings.Contains(problems[0], `"bin/lct publsh"`) {
			t.Errorf("problems = %q, want one naming bin/lct publsh", problems)
		}
	})
	t.Run("a file that is not YAML", func(t *testing.T) {
		problems := checkWorkflow(".github/workflows/ci.yml", ci+"\n  - [unclosed\n", commands)
		if len(problems) != 1 || !strings.Contains(problems[0], "not valid YAML") {
			t.Errorf("problems = %q, want one saying it is not valid YAML", problems)
		}
	})

	// 2. No model name.
	t.Run("a model name", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, root, ".github/workflows/rebuild.yml", rebuild+"# claude-example-9\n")
		problems := checkModelNames(root, ".github/workflows")
		if len(problems) != 1 || !strings.Contains(problems[0], "rebuild.yml") || !strings.Contains(problems[0], `"claude-example-9"`) {
			t.Errorf("problems = %q, want one naming rebuild.yml and the model", problems)
		}
	})
	t.Run("a secret", func(t *testing.T) {
		src := mutate(t, rebuild, "${{ github.token }}", "${{ secrets.RELEASE_TOKEN }}")
		problems := checkWorkflow(file, src, commands)
		if len(problems) != 1 || !strings.Contains(problems[0], "secrets.") {
			t.Errorf("problems = %q, want one naming secrets.", problems)
		}
	})
	t.Run("an action that is not pinned", func(t *testing.T) {
		src := mutate(t, rebuild, "actions/checkout@v4", "actions/checkout@main")
		problems := checkWorkflow(file, src, commands)
		if len(problems) != 1 || !strings.Contains(problems[0], "actions/checkout@main") {
			t.Errorf("problems = %q, want one naming actions/checkout@main", problems)
		}
	})

	// 3. The rebuild is made for a fresh clone.
	for _, tc := range []struct{ name, old, repl, want string }{
		{"no concurrency group", "group: lct-state", "group: other", "concurrency"},
		{"a run cancelled by the next", "cancel-in-progress: false", "cancel-in-progress: true", "concurrency"},
		{"a shallow clone", "fetch-depth: 0", "fetch-depth: 1", "fetch-depth: 0"},
		{"no tag fetch", "+refs/tags/*:refs/tags/*", "+refs/heads/main:refs/remotes/origin/main", `"+refs/tags/*:refs/tags/*"`},
		{"no raw worktree", "git worktree add .work/raw raw", "git worktree prune", `"git worktree add .work/raw raw"`},
		{"no raw tag push", `git push origin "refs/tags/`, `echo "refs/tags/`, `git push origin \"refs/tags/`},
		{"no tw push", "git push origin tw", "git status", `"git push origin tw"`},
		{"an answer of origin that set -e does not see", "remote=$(git ls-remote --tags origin", ": $(git ls-remote --tags origin", `"remote=$(git ls-remote --tags origin"`},
		{"a Release tag not made at the tw commit", `--target "$tw_commit"`, "--target main", "not made at the tw commit"},
		{"assets read in a process substitution", "assets_json=$(jq -ec", "assets_json=$(cat out/publish.json; jq -ec", `"assets_json=$(jq -ec"`},
		{"no check of the assets", `"${#assets[@]}" -eq 2`, `"${#assets[@]}" -ge 0`, "number of assets is not checked"},
		{"assets read before they are captured", "assets_json=$(jq -ec", "mapfile -t assets </dev/null; assets_json=$(jq -ec", `"assets_json=$(jq -ec" must come before "mapfile -t assets"`},
		{"a push before the build", "bin/lct build --json", "git push origin tw && bin/lct build --json", `"bin/lct publish" must come before "git push origin tw"`},
		{"a release before the publish", "bin/lct publish --json >", "gh release create x && bin/lct publish --json >", `"git push origin tw" must come before "gh release create"`},
		{"no token for gh", "GH_TOKEN: ${{ github.token }}", "GH_USER: x", "GH_TOKEN"},
		{"no manual trigger", "workflow_dispatch", "push", "workflow_dispatch"},
		{"no write permission", "contents: write", "contents: read", "permissions.contents"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := mutate(t, rebuild, tc.old, tc.repl)
			problems := checkRebuildWorkflow(file, src)
			if len(problems) == 0 {
				t.Fatalf("no problems, want one naming %q", tc.want)
			}
			if !slices.ContainsFunc(problems, func(p string) bool { return strings.Contains(p, tc.want) && strings.Contains(p, file) }) {
				t.Errorf("problems = %q, want one naming %s and %q", problems, file, tc.want)
			}
		})
	}

	// 4. The name of the Release.
	for _, title := range []string{`"Claude Code $tag"`, `Anthropic-$tag`, `'zh-TW by claude  code'`} {
		t.Run("title "+title, func(t *testing.T) {
			src := mutate(t, rebuild, `--title "$tag"`, "--title "+title)
			problems := checkWorkflow(file, src, commands)
			if len(problems) != 1 || !strings.Contains(problems[0], "--title") || !strings.Contains(problems[0], title) {
				t.Errorf("problems = %q, want one naming --title %s", problems, title)
			}
		})
	}
	t.Run("no title", func(t *testing.T) {
		src := mutate(t, rebuild, `--title "$tag"`, "")
		problems := checkRebuildWorkflow(file, src)
		if !slices.ContainsFunc(problems, func(p string) bool { return strings.Contains(p, "--title") }) {
			t.Errorf("problems = %q, want one about --title", problems)
		}
	})
}

// The jq commands of rebuild.yml read the keys lct publish --json prints
// (publish.Result): a key renamed in the code must fail here, not in the
// Release step.
func TestWorkflowReadsPublishResult(t *testing.T) {
	file, rebuild := readRepoFile(t, filepath.Join(repoRoot, ".github", "workflows", "rebuild.yml"))
	result := publish.Result{
		TWCommit:  "0123456789abcdef0123456789abcdef01234567",
		Changed:   true,
		Tag:       "zh-TW-20261029.1",
		NotesPath: "out/release-notes.md",
		Assets:    []string{"out/text.zip", "out/complete.zip"},
	}
	for _, problem := range checkPublishKeys(file, rebuild, result) {
		t.Error(problem)
	}
	if t.Failed() {
		return
	}

	t.Run("a key the workflow reads that is not printed", func(t *testing.T) {
		src := mutate(t, rebuild, "jq -er .tw_commit", "jq -er .tw_sha")
		problems := checkPublishKeys(file, src, result)
		want := []string{`does not read "tw_commit"`, `reads "tw_sha"`}
		if len(problems) != len(want) {
			t.Fatalf("problems = %q, want %d naming %q", problems, len(want), want)
		}
		for i, w := range want {
			if !strings.Contains(problems[i], w) {
				t.Errorf("problem %q does not contain %q", problems[i], w)
			}
		}
	})
	t.Run("a result that lost a key", func(t *testing.T) {
		lost := struct {
			Tag    string   `json:"tag"`
			Assets []string `json:"assets"`
		}{Tag: result.Tag, Assets: result.Assets}
		problems := checkPublishKeys(file, rebuild, lost)
		if len(problems) != 1 || !strings.Contains(problems[0], `reads "tw_commit"`) {
			t.Errorf("problems = %q, want one saying it reads tw_commit, which is not printed", problems)
		}
	})
}
