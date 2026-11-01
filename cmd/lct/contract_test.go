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
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/tool-jx3/limbus-translate/internal/cli"
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
	writeTestFile(t, root, "prompts/model.md", "預設：claude-sonnet-4-5。\n")
	writeTestFile(t, root, "prompts/clean.md", "Claude 與 claude 可以單獨出現；opuses、haikus 不是模型名稱。\n")

	problems := checkModelNames(root, ".claude/skills", "prompts")
	want := [][2]string{
		{".claude/skills/demo/SKILL.md", `"Opus"`},
		{"prompts/model.md", `"claude-sonnet-4-5"`},
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
