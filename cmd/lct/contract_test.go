package main

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/plan"
	"github.com/tool-jx3/limbus-translate/internal/proposal"
	"github.com/tool-jx3/limbus-translate/internal/publish"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/store"
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
	// notModelNames are what modelNameRe matches that is not a model: the npm
	// package of the CLI, which the pipeline installs at a fixed version, and
	// the action rules-proposal.yml runs the rules-review skill with.
	notModelNames = []string{"claude-code", "claude-code-action"}
)

// skillFrontmatter is the part of a SKILL.md frontmatter the contract checks.
type skillFrontmatter struct {
	Name                   string   `yaml:"name"`
	Description            string   `yaml:"description"`
	DisableModelInvocation bool     `yaml:"disable-model-invocation"`
	AllowedTools           []string `yaml:"allowed-tools"`
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

// adjustTools are the tools of the adjust skill: it searches, writes an
// adjustment or a request file through lct, lints the rules, and edits the
// guides, the glossary and the rules (a change to a guide or a term only after
// the person agrees). It runs requeue only in the form that writes a request
// file, which touches no state.
var adjustTools = []string{
	"Bash(bin/lct search *)",
	"Bash(bin/lct adjust add *)",
	"Bash(bin/lct requeue --to-file *)",
	"Bash(bin/lct rules lint*)",
	"Bash(go build -o bin/lct ./cmd/lct)",
	"Read(./**)",
	"Edit(./guides/**)",
	"Edit(./glossary/**)",
	"Edit(./rules/**)",
}

// weeklyUpdateTools are the tools of the weekly-update skill: the steps of the
// weekly pipeline up to the build, and the translation loop of translate-pending.
// Nothing here publishes or pushes: no publish, and no git.
var weeklyUpdateTools = []string{
	"Bash(bin/lct fetch *)",
	"Bash(bin/lct snapshot *)",
	"Bash(bin/lct plan *)",
	"Bash(bin/lct batch *)",
	"Bash(bin/lct pending *)",
	"Bash(bin/lct build *)",
	"Bash(go build -o bin/lct ./cmd/lct)",
	"Read(./**)",
	"Edit(./work/**)",
}

// rulesReviewTools are the tools of the rules-review skill. It reads what the
// build found and edits the rules (guides/, glossary/, rules/) and its own
// files in work/; every lct command it runs either reads, writes a rule file
// or request file, or publishes the rules PR, and the only way it reaches
// origin is lct proposal publish, which pushes a rules-proposal/* branch and
// touches nothing but guides/, glossary/ and rules/. lct impact is pinned to
// the one file it writes (and refuses one outside work/ and out/ anyway). lct
// accept is granted only with --dry-run, which reads (the JP/EN of the variants
// of an inconsistent group): it refuses --dry-run turned off again
// (TestAcceptDryRunCannotBeTurnedOff), and reads a --file, whose lines it
// prints back, only inside the repo and outside .git
// (TestAcceptFileStaysInTheRepo), so the grant writes nothing and reads
// nothing that Read(./**) without .git/ does not. rules-proposal.yml hands
// the same list to the action (TestWorkflowContract); a tool added to one and
// not to the other fails there.
var rulesReviewTools = []string{
	"Bash(bin/lct suggest *)",
	"Bash(bin/lct search *)",
	"Bash(bin/lct accept --dry-run *)",
	"Bash(bin/lct glossary derive*)",
	"Bash(bin/lct adjust tidy*)",
	"Bash(bin/lct requeue --to-file *)",
	"Bash(bin/lct rules lint*)",
	"Bash(bin/lct impact --base origin/main --markdown work/proposal/impact.md*)",
	"Bash(bin/lct proposal publish *)",
	"Bash(go build -o bin/lct ./cmd/lct)",
	"Read(./**)",
	"Edit(./guides/**)",
	"Edit(./glossary/**)",
	"Edit(./rules/**)",
	"Edit(./work/**)",
}

// reviewTools are the tools of the review skill: it reads what needs a person
// (lct review), looks units up, and writes what the person decides on: an
// adjustment, a request file, the tidied adjustments, the learned field rules,
// and the guides, the glossary, the rules and the frozen list of lct.yaml. It
// runs requeue only in the form that writes a request file, which touches no
// state, and nothing that pushes or publishes.
var reviewTools = []string{
	"Bash(bin/lct review*)",
	"Bash(bin/lct search *)",
	"Bash(bin/lct adjust add *)",
	"Bash(bin/lct adjust tidy*)",
	"Bash(bin/lct requeue --to-file *)",
	"Bash(bin/lct fields learn*)",
	"Bash(bin/lct rules lint*)",
	"Bash(go build -o bin/lct ./cmd/lct)",
	"Read(./**)",
	"Edit(./guides/**)",
	"Edit(./glossary/**)",
	"Edit(./rules/**)",
	"Edit(./lct.yaml)",
}

// termDecisionTools are the tools of the term-decision skill: it lists the
// candidates (lct suggest, lct glossary derive), looks units up, writes a
// request file, and edits the glossary and the rules (the declined list). It
// reaches neither the guides nor the config.
var termDecisionTools = []string{
	"Bash(bin/lct suggest *)",
	"Bash(bin/lct glossary derive*)",
	"Bash(bin/lct search *)",
	"Bash(bin/lct requeue --to-file *)",
	"Bash(bin/lct rules lint*)",
	"Bash(go build -o bin/lct ./cmd/lct)",
	"Read(./**)",
	"Edit(./glossary/**)",
	"Edit(./rules/**)",
}

// reviewPageTools are the tools of the review-page skill: it handles the item
// ids a person pastes from the review page. It confirms items (lct accept),
// looks them up (lct suggest, search, validate), writes adjustments and
// request files, and edits the guides, the glossary and the rules. It reaches
// neither the config nor git.
var reviewPageTools = []string{
	"Bash(bin/lct accept *)",
	"Bash(bin/lct suggest*)",
	"Bash(bin/lct search *)",
	"Bash(bin/lct validate *)",
	"Bash(bin/lct adjust add *)",
	"Bash(bin/lct requeue --to-file *)",
	"Bash(bin/lct rules lint*)",
	"Bash(go build -o bin/lct ./cmd/lct)",
	"Read(./**)",
	"Edit(./guides/**)",
	"Edit(./glossary/**)",
	"Edit(./rules/**)",
}

// requiredSkills are the skills this repo must have, and the tools of each.
// Each is run on purpose by a person or by CI, so none may be started by the
// model on its own.
var requiredSkills = map[string][]string{
	"translate-pending": translatePendingTools,
	"adjust":            adjustTools,
	"weekly-update":     weeklyUpdateTools,
	"rules-review":      rulesReviewTools,
	"review":            reviewTools,
	"term-decision":     termDecisionTools,
	"review-page":       reviewPageTools,
}

// goBuildTool is the one go command a skill may run: it builds bin/lct.
const goBuildTool = "Bash(go build -o bin/lct ./cmd/lct)"

// lctWordRe is one word of an lct command name.
var lctWordRe = regexp.MustCompile(`^[a-z][a-z-]*$`)

// checkPermissions reports the entries of a permission list (the allow list
// of a settings file, or a skill's allowed-tools) that grant too much:
//   - a bare Read, Edit, Write or Bash, or one whose specifier is only * or
//     **, which allows every file or command;
//   - a Bash(bin/lct ...) that names no registered command: the words before
//     the closing * must be a command (two words, else the first, as
//     checkLctCommands reads them) or, as a single word, the group of
//     two-word commands it starts (batch for batch next and batch submit),
//     optionally followed by flags (requeue --to-file), each of which may
//     take one value (impact --base origin/main). A word that follows the
//     command, or a value, without a flag before it is refused.
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
		// The command comes first. Flags may follow it and narrow the entry
		// ("requeue --to-file" allows the form of requeue that writes a request
		// file and no other), and so may the value of a flag ("impact --base
		// origin/main --markdown work/proposal/impact.md" allows one comparison
		// written to one file).
		n := 0
		for n < len(words) && lctWordRe.MatchString(words[n]) {
			n++
		}
		if n == 0 {
			problems = append(problems, fmt.Sprintf("%s: %q does not start with an lct command", file, e))
			continue
		}
		if !flagsAndValues(words[n:]) {
			problems = append(problems, fmt.Sprintf("%s: %q has more than a command and flags before the *", file, e))
			continue
		}
		words = words[:n]
		group := len(words) == 1 && slices.ContainsFunc(commands, func(c string) bool { return strings.HasPrefix(c, words[0]+" ") })
		if (len(words) > 1 && slices.Contains(commands, words[0]+" "+words[1])) || slices.Contains(commands, words[0]) || group {
			continue
		}
		problems = append(problems, fmt.Sprintf("%s: %q names no registered lct command", file, e))
	}
	return problems
}

// flagsAndValues reports whether words are flags, each followed by one value
// at most: every word that is not a flag follows one.
func flagsAndValues(words []string) bool {
	for i, w := range words {
		if !strings.HasPrefix(w, "-") && (i == 0 || !strings.HasPrefix(words[i-1], "-")) {
			return false
		}
	}
	return true
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

// checkSkillTools reports the allowed-tools of the skill at file that are not
// of the four kinds a skill may use (bin/lct commands, the go build of bin/lct,
// Read and Edit) and those checkPermissions refuses.
func checkSkillTools(file string, tools, commands []string) []string {
	var problems []string
	for _, e := range tools {
		tool, spec, _ := strings.Cut(e, "(")
		switch {
		case tool == "Read" || tool == "Edit":
		case e == goBuildTool:
		case tool == "Bash" && strings.HasPrefix(spec, "bin/lct "):
		default:
			problems = append(problems, fmt.Sprintf("%s: allowed-tools has %q; a skill may use only bin/lct commands, the go build of bin/lct, Read and Edit", file, e))
		}
	}
	return append(problems, checkPermissions(file, tools, commands)...)
}

// settingsAllow is the whole allow list of .claude/settings.json: what
// translate-pending runs, plus lct validate and lct rules lint for the person
// who works in the repo. A skill carries its own tools, so the other skills add
// nothing here; an entry is added to this list on purpose, in the same change as
// the entry of the file.
var settingsAllow = []string{
	"Bash(bin/lct batch *)",
	"Bash(bin/lct pending *)",
	"Bash(bin/lct validate *)",
	"Bash(bin/lct rules lint*)",
	"Bash(bin/lct guide resolve *)",
	"Bash(go build -o bin/lct ./cmd/lct)",
	"Read(./**)",
	"Edit(./work/**)",
}

// checkSettings checks the text of .claude/settings.json: that it parses, that
// its allow list is exactly settingsAllow (as a set of entries, none twice),
// and that no entry grants too much.
func checkSettings(file, content string, commands []string) []string {
	var settings struct {
		Permissions struct {
			Allow []string `json:"allow"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal([]byte(content), &settings); err != nil {
		return []string{fmt.Sprintf("%s: does not parse: %v", file, err)}
	}
	allow := settings.Permissions.Allow
	if len(allow) == 0 {
		return []string{file + ": allows nothing: permissions.allow is missing"}
	}
	var problems []string
	got := slices.Sorted(slices.Values(allow))
	want := slices.Sorted(slices.Values(settingsAllow))
	for i := 1; i < len(got); i++ {
		if got[i] == got[i-1] {
			problems = append(problems, fmt.Sprintf("%s: allows %q twice", file, got[i]))
		}
	}
	got = slices.Compact(got)
	for _, e := range got {
		if !slices.Contains(want, e) {
			problems = append(problems, fmt.Sprintf("%s: allows %q, which is not in the fixed list of the contract (settingsAllow)", file, e))
		}
	}
	for _, e := range want {
		if !slices.Contains(got, e) {
			problems = append(problems, fmt.Sprintf("%s: does not allow %q, which the fixed list of the contract (settingsAllow) has", file, e))
		}
	}
	return append(problems, checkPermissions(file, allow, commands)...)
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
				if slices.Contains(notModelNames, strings.ToLower(name)) {
					continue
				}
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
	for name := range requiredSkills {
		if !slices.Contains(skills, filepath.Join(repoRoot, ".claude", "skills", name, "SKILL.md")) {
			t.Errorf(".claude/skills/%s/SKILL.md does not exist", name)
		}
	}
	commands := cli.Commands()
	for _, p := range skills {
		rel, content := readRepoFile(t, p)
		for _, problem := range checkSkill(repoRoot, rel, content, commands) {
			t.Error(problem)
		}
		front, _, ok := splitFrontmatter(content)
		if !ok {
			continue // checkSkill said so
		}
		var fm skillFrontmatter
		if err := yaml.Unmarshal([]byte(front), &fm); err != nil {
			continue // checkSkill said so
		}
		for _, problem := range checkSkillTools(rel, fm.AllowedTools, commands) {
			t.Error(problem)
		}
		if _, required := requiredSkills[fm.Name]; required && !fm.DisableModelInvocation {
			t.Errorf("%s: no disable-model-invocation: true; only a person or CI starts this skill", rel)
		}
		if lines := strings.Count(content, "\n"); lines > skillMaxLines {
			t.Errorf("%s: %d lines, want at most %d: a skill is the flow, the rules are in guides/ and glossary/", rel, lines, skillMaxLines)
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

// skillMaxLines is the length of a SKILL.md the repo allows (the spec allows
// 500; the flow of a skill here fits in 200).
const skillMaxLines = 200

// The repo's .claude/settings.json grants what the skills run and nothing
// broader: other lct commands, and files outside the repo, ask a person.
func TestSettingsContract(t *testing.T) {
	file, content := readRepoFile(t, filepath.Join(repoRoot, ".claude", "settings.json"))
	for _, problem := range checkSettings(file, content, cli.Commands()) {
		t.Error(problem)
	}
}

// The allow list is exact: an entry that is added, removed or doubled fails,
// even one checkPermissions finds nothing wrong with (bin/lct build is a
// registered command and scoped), because it is a grant nobody decided on.
func TestSettingsContractIsExact(t *testing.T) {
	file, content := readRepoFile(t, filepath.Join(repoRoot, ".claude", "settings.json"))
	commands := cli.Commands()
	if problems := checkSettings(file, content, commands); len(problems) > 0 {
		t.Fatalf("the settings of the repo already fail:\n%s", strings.Join(problems, "\n"))
	}
	// withAllow returns the settings with the allow list changed by change.
	withAllow := func(t *testing.T, change func([]string) []string) string {
		t.Helper()
		var settings map[string]any
		if err := json.Unmarshal([]byte(content), &settings); err != nil {
			t.Fatal(err)
		}
		perms := settings["permissions"].(map[string]any)
		var allow []string
		for _, e := range perms["allow"].([]any) {
			allow = append(allow, e.(string))
		}
		perms["allow"] = change(allow)
		data, err := json.Marshal(settings)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}

	for _, tc := range []struct {
		name   string
		change func([]string) []string
		want   string
	}{
		{"an entry added", func(a []string) []string { return append(a, "Bash(bin/lct build *)") }, `allows "Bash(bin/lct build *)"`},
		{"an entry removed", func(a []string) []string {
			return slices.DeleteFunc(a, func(e string) bool { return e == "Read(./**)" })
		}, `does not allow "Read(./**)"`},
		{"an entry twice", func(a []string) []string { return append(a, a[0]) }, "twice"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			problems := checkSettings(file, withAllow(t, tc.change), commands)
			if !slices.ContainsFunc(problems, func(p string) bool { return strings.Contains(p, tc.want) && strings.Contains(p, file) }) {
				t.Errorf("problems = %q, want one naming %s and %s", problems, file, tc.want)
			}
		})
	}
	t.Run("an added entry is the only problem", func(t *testing.T) {
		problems := checkSettings(file, withAllow(t, func(a []string) []string { return append(a, "Bash(bin/lct build *)") }), commands)
		if len(problems) != 1 {
			t.Errorf("problems = %q, want exactly one", problems)
		}
	})
	t.Run("the order does not matter", func(t *testing.T) {
		problems := checkSettings(file, withAllow(t, func(a []string) []string { slices.Reverse(a); return a }), commands)
		if len(problems) > 0 {
			t.Errorf("problems in a reordered list = %q, want none", problems)
		}
	})
	t.Run("no allow list", func(t *testing.T) {
		problems := checkSettings(file, `{"permissions":{}}`, commands)
		if len(problems) != 1 || !strings.Contains(problems[0], "allows nothing") {
			t.Errorf("problems = %q, want one saying nothing is allowed", problems)
		}
	})
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

// The translator registers a new term in the `terms` of the unit: the schema
// of the out file declares it and the shared prompt explains it.
func TestBatchOutSchemaDeclaresTerms(t *testing.T) {
	_, content := readRepoFile(t, filepath.Join(repoRoot, "schemas", "batch-out.schema.json"))
	var schema struct {
		Items struct {
			Properties map[string]struct {
				Type  string `json:"type"`
				Items struct {
					Type       string `json:"type"`
					Properties map[string]struct {
						Type string `json:"type"`
					} `json:"properties"`
				} `json:"items"`
			} `json:"properties"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(content), &schema); err != nil {
		t.Fatalf("schemas/batch-out.schema.json does not parse: %v", err)
	}
	terms, ok := schema.Items.Properties["terms"]
	if !ok || terms.Type != "array" || terms.Items.Type != "object" {
		t.Fatalf("batch-out.schema.json: items.properties.terms = %+v (declared %v), want an array of objects", terms, ok)
	}
	for _, f := range []string{"kr", "zh"} {
		if terms.Items.Properties[f].Type != "string" {
			t.Errorf("batch-out.schema.json: terms items property %s = %+v, want a string", f, terms.Items.Properties[f])
		}
	}
	if _, prompt := readRepoFile(t, filepath.Join(repoRoot, "prompts", "translate.md")); !strings.Contains(prompt, "`terms`") {
		t.Error("prompts/translate.md does not explain `terms`")
	}
}

// A unit whose adjustment expired carries the person's old text in
// refs.adjustment (store.RefAdjustment): the shared prompt tells the translator
// what that is and what to do with it, in the section of the refs.
func TestTranslatePromptExplainsTheAdjustmentRef(t *testing.T) {
	_, prompt := readRepoFile(t, filepath.Join(repoRoot, "prompts", "translate.md"))
	_, refs, ok := strings.Cut(prompt, "## 參考譯文 `refs`")
	if !ok {
		t.Fatal("prompts/translate.md has no section 參考譯文 `refs`")
	}
	refs, _, _ = strings.Cut(refs, "\n## ")
	for _, want := range []string{"`" + store.RefAdjustment + "`", "adjustment-expired"} {
		if !strings.Contains(refs, want) {
			t.Errorf("the refs section of prompts/translate.md does not mention %s", want)
		}
	}
}

// checkExactTools fails the test unless the skill's allowed-tools are exactly
// want, in that order.
func checkExactTools(t *testing.T, skill string, want []string) {
	t.Helper()
	file, content := readRepoFile(t, filepath.Join(repoRoot, ".claude", "skills", skill, "SKILL.md"))
	front, _, ok := splitFrontmatter(content)
	if !ok {
		t.Fatalf("%s has no frontmatter", file)
	}
	var fm skillFrontmatter
	if err := yaml.Unmarshal([]byte(front), &fm); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(fm.AllowedTools, want) {
		t.Errorf("%s: allowed-tools = %q, want %q", file, fm.AllowedTools, want)
	}
	for _, problem := range checkSkillTools(file, fm.AllowedTools, cli.Commands()) {
		t.Error(problem)
	}
}

// skillAllowedTools returns the allowed-tools of the skill; a skill that does
// not exist, or has no frontmatter, fails the test.
func skillAllowedTools(t *testing.T, skill string) []string {
	t.Helper()
	file, content := readRepoFile(t, filepath.Join(repoRoot, ".claude", "skills", skill, "SKILL.md"))
	front, _, ok := splitFrontmatter(content)
	if !ok {
		t.Fatalf("%s has no frontmatter", file)
	}
	var fm skillFrontmatter
	if err := yaml.Unmarshal([]byte(front), &fm); err != nil {
		t.Fatal(err)
	}
	return fm.AllowedTools
}

// The translate-pending skill may use exactly the R35 tools.
func TestTranslatePendingTools(t *testing.T) {
	checkExactTools(t, "translate-pending", translatePendingTools)
}

// The adjust skill writes rules for a person to commit to main; it may not
// reach the pending list, the records or the branches that only CI writes.
func TestAdjustTools(t *testing.T) {
	checkExactTools(t, "adjust", adjustTools)
}

// The weekly-update skill is a rehearsal: it may build the pack on this
// machine and nothing else.
func TestWeeklyUpdateTools(t *testing.T) {
	checkExactTools(t, "weekly-update", weeklyUpdateTools)
}

// weekly-update says plainly that nothing leaves the machine: it pushes
// neither raw, tw nor main, and never publishes. The tools it has cannot do
// either (see TestWeeklyUpdateTools); its text says so to the person and to
// the model.
func TestWeeklyUpdateSaysItStaysLocal(t *testing.T) {
	file, content := readRepoFile(t, filepath.Join(repoRoot, ".claude", "skills", "weekly-update", "SKILL.md"))
	_, body, ok := splitFrontmatter(content)
	if !ok {
		t.Fatalf("%s has no frontmatter", file)
	}
	for _, part := range []string{"git push", "lct publish", "`raw`", "`tw`", "`main`", "`out/`"} {
		if !strings.Contains(body, part) {
			t.Errorf("%s does not mention %s", file, part)
		}
	}
	if strings.Contains(body, "bin/lct publish") {
		t.Errorf("%s writes bin/lct publish as a command to run", file)
	}
}

// The adjust skill reaches main only through the person: it says to commit and
// not to push raw or tw.
func TestAdjustSaysWhereItEnds(t *testing.T) {
	file, content := readRepoFile(t, filepath.Join(repoRoot, ".claude", "skills", "adjust", "SKILL.md"))
	_, body, ok := splitFrontmatter(content)
	if !ok {
		t.Fatalf("%s has no frontmatter", file)
	}
	for _, part := range []string{"commit", "`main`", "`raw`", "`tw`"} {
		if !strings.Contains(body, part) {
			t.Errorf("%s does not mention %s", file, part)
		}
	}
}

// The rules-review skill may use exactly the tools rules-proposal.yml hands to
// the action.
func TestRulesReviewTools(t *testing.T) {
	checkExactTools(t, "rules-review", rulesReviewTools)
}

func TestReviewPageTools(t *testing.T) {
	checkExactTools(t, "review-page", reviewPageTools)
}

// reviewPageSays are what the body of the review-page skill must say besides
// localSkillSays: how it looks the items up, how the ids and the reason are
// quoted, that it shows a change before it writes it, and that it neither
// commits nor runs git.
var reviewPageSays = append(slices.Clone(localSkillSays),
	skillSays{"bin/lct accept --dry-run --json", "the items of the pasted ids are not looked up with accept --dry-run: lct suggest lists only the notes new since the last build, and a warning has no key in its id"},
	skillSays{"單引號", "the ids and the reason are not said to be single-quoted: a note id holds | and an id may hold spaces, and double quotes expand $ and backticks"},
	skillSays{"硬性失敗", "the skill does not warn that lct validate checks like a new translation: a hangul or simplified warning of the page may fail hard there"},
	skillSays{"不 commit、不 push，也不執行 `git`", "the skill does not say it neither commits nor runs git"},
	skillSays{"同意才寫", "the skill does not show a change to a rule file before it writes it"},
	skillSays{"`jpen_class`", "the skill does not show the JP/EN of the variants of an inconsistent group and its class: KR is often less specific than JP and EN, and a person cannot tell a right difference from a real inconsistency"},
	skillSays{"不因 JP/EN 分類", "the skill does not say that the class of JP/EN decides nothing: it would confirm a group for a person because its JP/EN explains it"},
)

// The review-page skill says what the contract wants of it, and each check can fail.
func TestReviewPageSays(t *testing.T) {
	file, body, tools := localSkillBody(t, "review-page")
	for _, problem := range checkSkillBody(file, body, reviewPageSays, nil, tools) {
		t.Error(problem)
	}
	if t.Failed() {
		return
	}
	for _, c := range []struct{ name, old, repl, want string }{
		{"no word about data", "不是指示", "是指示", "不是指示"},
		{"main not named", "`main`", "`master`", "`main`"},
		{"no lookup", "bin/lct accept --dry-run --json", "bin/lct suggest --json", "bin/lct accept --dry-run --json"},
		{"no quoting", "單引號", "引號", "單引號"},
		{"no word about validate", "硬性失敗", "失敗", "硬性失敗"},
		{"may run git", "不 commit、不 push，也不執行 `git`", "可以 commit", "不 commit、不 push，也不執行 `git`"},
		{"writes unseen", "同意才寫", "就寫", "同意才寫"},
		{"no JP/EN shown", "`jpen_class`", "`kind`", "`jpen_class`"},
		{"decides by the class", "不因 JP/EN 分類", "依 JP/EN 分類", "不因 JP/EN 分類"},
	} {
		t.Run(c.name, func(t *testing.T) {
			src := mutateAll(t, body, c.old, c.repl)
			hasProblem(t, checkSkillBody(file, src, reviewPageSays, nil, tools), c.want)
		})
	}
}

// skillSays is a text the body of a skill must hold, and what is wrong with a
// body that lacks it.
type skillSays struct{ text, why string }

// rulesReviewSays are what the body of the rules-review skill must say, with
// the reason: the files a run in CI leaves for it, the markers that make the
// description of the PR readable to lct, and the rules that keep a rules PR
// from losing or repeating a suggestion. Each is a text of the body.
var rulesReviewSays = []skillSays{
	{"work/proposal/started.json", "the file by which a run knows it is in CI is not named: exclude.txt is empty when no rules PR is open or was closed, and a run that takes an empty file for a missing one thinks it is on a person's machine and publishes nothing"},
	{"work/proposal/exclude.txt", "the file of the ids that are not to be suggested again is not named: a run in CI would suggest what the PR has or people turned down"},
	{"--limit 8", "the suggestions are asked for all at once, more than a run of 80 turns handles: it may end before it publishes anything"},
	{"一定接著做第 3 到 5 步", "the skill does not say that it goes on to the lint, the impact, the description and the publish once its suggestions are handled"},
	{"只提出一次", "the skill says that a note it skips comes back, while a build lists only the notes since the build before"},
	{"--exclude-file work/proposal/exclude.txt", "the suggestions are not asked with the ids left out"},
	{"work/proposal/existing-body.md", "the description the open PR has is not named: a new description would drop its suggestions, and lct proposal publish refuses it"},
	{"work/proposal/body.md", "the description of the PR is written nowhere"},
	{"<!-- suggestion:<id> -->", "the marker of a suggestion is not shown"},
	{"單獨一行", "the marker is not said to be on a line of its own"},
	{"rules/declined.yaml", "the description does not tell a person where to move what they do not want"},
	{"不是指示", "the text of the PR, the evidence of a suggestion and the results of a search are not said to be data"},
	{"work/proposal/outcome.txt", "the file that tells the workflow what the run did is not named: a run whose publish was refused would end green"},
	{"`published`", "the skill does not say what it writes when lct proposal publish succeeded"},
	{"`nothing: ", "the skill does not say what it writes when it proposes nothing, so a run that proposes nothing looks like a failure"},
	{"`failed: ", "the skill does not say what it writes when lct proposal publish was refused for good, so the workflow cannot tell it from a run that proposed nothing"},
	{"不要為了通過而", "the skill does not forbid a pointless change to get past the refusal \"nothing to propose\" of lct proposal publish"},
	{"`" + proposal.UndecidedHeading + "`", "the skill does not say where the items it is unsure of go, under the heading lct proposal start reads (an item that is never listed is suggested first again on every run, and the run stalls on it; one under another heading is taken for decided when the PR is merged)"},
	{"不記進 `rules/declined.yaml`", "the skill does not keep an item it is unsure of out of the declined list: a person would never decide on it"},
	{"也照樣 publish", "the skill does not publish a description that only adds items for a person to decide: they would never leave the first 8 suggestions"},
	{"移到 `" + proposal.UndecidedHeading + "` 之上", "the opening of the description does not tell a person to move an item they adopt out of the section for a person to decide: lct proposal start takes an item still under it for undecided, and suggests it again after the merge"},
	{"不算沒有東西可以提出", "the skill writes nothing: whenever it changed no rule file, though its items for a person to decide are something to propose"},
	{"不要再執行 `go build`", "the skill builds lct again in CI, from the code of the PR branch, over the lct the workflow counted the suggestions with"},
	{"`.git/`", "the skill does not say it must not read the folder of git, which can hold the credentials of the checkout"},
	{"`raw`", "the skill does not say it leaves raw alone"},
	{"`tw`", "the skill does not say it leaves tw alone"},
	{"`main`", "the skill does not say it leaves main alone"},
	{"bin/lct accept --dry-run --json", "the JP/EN of the variants of an inconsistent group are not looked up: lct suggest gives only the KR and the translations"},
	{"`jpen_class`", "the evidence of an inconsistent group does not list the JP/EN of its variants and its class: a person cannot tell a right difference from a real inconsistency"},
	{"不因 JP/EN 分類", "the skill does not say that the class of JP/EN decides nothing: it would decline or unify a group for a person because of its JP/EN"},
	{"JP/EN 的分類或內容", "the skill forbids only the class as a reason: it would decline or unify a group by citing what its JP and EN say, the same reason in other words"},
	{"不要統一成一種寫法", "the skill does not keep apart the translations that follow JP/EN: told to settle on one rendering, it would merge a right difference such as brother and sister"},
}

// rulesReviewFlow are the commands the body of the rules-review skill writes in
// full, in the order the flow runs them: the first before the second, by the
// first place each is written.
var rulesReviewFlow = [][2]string{
	{"work/proposal/started.json", "go build -o bin/lct ./cmd/lct"}, // where it runs decides whether lct is built
	{"go build -o bin/lct ./cmd/lct", "bin/lct suggest --json"},
	{"bin/lct suggest --json", "bin/lct rules lint --json"},
	{"bin/lct rules lint --json", "bin/lct impact --base origin/main --markdown work/proposal/impact.md"},
	{"bin/lct impact --base origin/main --markdown work/proposal/impact.md", "bin/lct proposal publish --body-file work/proposal/body.md"},
}

// checkRulesReviewSkill checks the body of the rules-review skill against the
// contract: it says what rulesReviewSays has, writes the commands of the flow
// in order, and runs every lct command its allowed-tools grant (a grant the
// flow never uses is a way in that nothing needs).
func checkRulesReviewSkill(file, body string, tools []string) []string {
	return checkSkillBody(file, body, rulesReviewSays, rulesReviewFlow, tools)
}

// checkSkillBody checks the body of a skill against its contract: it says
// what says has, writes the commands of flow in order, and runs every lct
// command its allowed-tools grant.
func checkSkillBody(file, body string, says []skillSays, flow [][2]string, tools []string) []string {
	var problems []string
	for _, s := range says {
		if !strings.Contains(body, s.text) {
			problems = append(problems, fmt.Sprintf("%s: does not say %q: %s", file, s.text, s.why))
		}
	}
	for _, o := range flow {
		first, second := strings.Index(body, o[0]), strings.Index(body, o[1])
		switch {
		case first < 0:
			problems = append(problems, fmt.Sprintf("%s: does not run %q", file, o[0]))
		case second < 0:
			problems = append(problems, fmt.Sprintf("%s: does not run %q", file, o[1]))
		case first > second:
			problems = append(problems, fmt.Sprintf("%s: %q must come before %q", file, o[0], o[1]))
		}
	}
	return append(problems, checkToolsRun(file, body, tools)...)
}

// checkToolsRun reports the lct commands among tools that body never writes
// out: a grant the flow never uses is a way in that nothing needs.
func checkToolsRun(file, body string, tools []string) []string {
	var problems []string
	for _, e := range tools {
		cmd, ok := strings.CutPrefix(e, "Bash(")
		if !ok || !strings.Contains(cmd, "bin/lct") {
			continue
		}
		cmd = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(cmd, ")"), "*"))
		if !strings.Contains(body, cmd) {
			problems = append(problems, fmt.Sprintf("%s: allowed-tools has %q, but the flow never runs %q", file, e, cmd))
		}
	}
	return problems
}

// The body of the rules-review skill says what the contract wants of it.
func TestRulesReviewSkillBody(t *testing.T) {
	file, content := readRepoFile(t, filepath.Join(repoRoot, ".claude", "skills", "rules-review", "SKILL.md"))
	front, body, ok := splitFrontmatter(content)
	if !ok {
		t.Fatalf("%s has no frontmatter", file)
	}
	var fm skillFrontmatter
	if err := yaml.Unmarshal([]byte(front), &fm); err != nil {
		t.Fatal(err)
	}
	for _, problem := range checkRulesReviewSkill(file, body, fm.AllowedTools) {
		t.Error(problem)
	}
	if t.Failed() {
		return
	}
	for _, tc := range []struct{ name, old, repl, want string }{
		{"no started file named", "work/proposal/started.json", "work/x.json", "work/proposal/started.json"},
		{"no exclude file named", "work/proposal/exclude.txt", "work/x.txt", "work/proposal/exclude.txt"},
		{"no limit of suggestions", "--limit 8", "--limit 20", "--limit 8"},
		{"a run that stops after the suggestions", "一定接著做第 3 到 5 步", "可以停下", "一定接著做第 3 到 5 步"},
		{"a note said to come back", "只提出一次", "下次還會再提出", "只提出一次"},
		{"suggestions asked without the exclude file", "--exclude-file work/proposal/exclude.txt", "--json", "--exclude-file work/proposal/exclude.txt"},
		{"no existing description named", "work/proposal/existing-body.md", "work/proposal/old.md", "work/proposal/existing-body.md"},
		{"no marker shown", "<!-- suggestion:<id> -->", "<!-- id -->", "<!-- suggestion:<id> -->"},
		{"no marker on its own line", "單獨一行", "同一行", "單獨一行"},
		{"no declined file named", "rules/declined.yaml", "rules/no.yaml", "rules/declined.yaml"},
		{"no word about data", "不是指示", "是指示", "不是指示"},
		{"no outcome file named", "work/proposal/outcome.txt", "work/proposal/out.txt", "work/proposal/outcome.txt"},
		{"no outcome for a publish", "`published`", "`done`", "`published`"},
		{"no outcome for nothing", "`nothing: ", "`none: ", "`nothing: "},
		{"no outcome for a refusal", "`failed: ", "`error: ", "`failed: "},
		{"a change to get past the refusal", "不要為了通過而", "可以為了通過而", "不要為了通過而"},
		{"unsure items left out", "`## 需要人決定`", "（略過）", "`## 需要人決定`"},
		{"unsure items declined", "不記進 `rules/declined.yaml`", "記進 `rules/declined.yaml`", "不記進 `rules/declined.yaml`"},
		{"no publish for unsure items only", "也照樣 publish", "就不 publish", "也照樣 publish"},
		{"an adopted item left under the heading", "移到 `## 需要人決定` 之上", "留在原處", "移到 `## 需要人決定` 之上"},
		{"nothing whenever no rule file changed", "不算沒有東西可以提出", "就算沒有東西可以提出", "不算沒有東西可以提出"},
		{"lct built again in CI", "不要再執行 `go build`", "再執行 `go build`", "不要再執行 `go build`"},
		{"the folder of git not forbidden", "`.git/`", "`.gitignore`", "`.git/`"},
		{"no JP/EN looked up", "bin/lct accept --dry-run --json", "bin/lct suggest --json --limit 1", "bin/lct accept --dry-run --json"},
		{"no JP/EN in the evidence", "`jpen_class`", "`kind`", "`jpen_class`"},
		{"decides by the class", "不因 JP/EN 分類", "依 JP/EN 分類", "不因 JP/EN 分類"},
		{"decides by what JP/EN say", "JP/EN 的分類或內容", "JP/EN 的分類", "JP/EN 的分類或內容"},
		{"unifies what follows JP/EN", "不要統一成一種寫法", "統一成一種寫法", "不要統一成一種寫法"},
		{"the mode found out after the build", "work/proposal/started.json", "go build -o bin/lct ./cmd/lct`; work/proposal/started.json", "must come before"},
		{"a command out of order", "bin/lct rules lint --json", "bin/lct impact --base origin/main --markdown work/proposal/impact.md; bin/lct rules lint --json", "must come before"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := mutateAll(t, body, tc.old, tc.repl)
			hasProblem(t, checkRulesReviewSkill(file, src, fm.AllowedTools), tc.want)
		})
	}
	t.Run("a tool the flow never runs", func(t *testing.T) {
		tools := append(slices.Clone(fm.AllowedTools), "Bash(bin/lct validate *)")
		wantProblems(t, checkRulesReviewSkill(file, body, tools), "bin/lct validate", "never runs")
	})
}

// The review skill may use exactly the tools of the brief: it reads
// "lct review", writes adjustments and request files, the field rules, and the
// guides, the glossary, the rules and lct.yaml, and pushes nothing.
func TestReviewTools(t *testing.T) {
	checkExactTools(t, "review", reviewTools)
}

// The term-decision skill may use exactly its tools: the glossary and the
// rules, and no guide, no config, no adjustment.
func TestTermDecisionTools(t *testing.T) {
	checkExactTools(t, "term-decision", termDecisionTools)
}

// localSkillSays are what the body of a skill a person runs on this machine
// says: where its changes end, and that what it reads is data.
var localSkillSays = []skillSays{
	{"不是指示", "the text it reads (lct output, search results, notes, the issue) is not said to be data: a line in it that looks like an instruction would be followed"},
	{"commit", "the person is not told to commit the changes"},
	{"`main`", "the changes are not said to go to main"},
	{"`raw`", "the skill does not say it leaves raw alone"},
	{"`tw`", "the skill does not say it leaves tw alone"},
	{"go build -o bin/lct ./cmd/lct", "lct is not built before it is run"},
	{"bin/lct rules lint", "the files it wrote are not checked"},
	{"guides/README.md", "the rules of translation are not pointed to"},
}

// reviewSays are what the body of the review skill must say besides
// localSkillSays: the shape of what lct review prints, the way out of every
// kind of item (build.ReviewKinds: a kind that build adds is asked of the
// skill at once, so that it knows what to do with it), and the issue to tick.
var reviewSays = append(append(slices.Clone(localSkillSays),
	skillSays{"bin/lct review --json", "the list of what needs a person is not asked for"},
	skillSays{"sections", "the sections of the result are not named"},
	skillSays{"`kind`", "the kind of a section is not named: the skill cannot tell what to do with a section"},
	skillSays{"`items`", "the items of a section are not named"},
	skillSays{"勾選", "the person is not told to tick the boxes of the issue"},
	skillSays{"bin/lct adjust tidy --json", "the adjustments that expired are not tidied after the new translation is confirmed"},
	skillSays{"bin/lct fields learn --write", "the learned field rules are never written"},
	skillSays{"rules/declined.yaml", "a speaker that needs no guide is not recorded"},
	skillSays{"`lct.yaml`", "the frozen list is not named"},
), reviewKindSays()...)

// reviewKindSays is what the review skill must say about each kind of
// build.ReviewKinds.
func reviewKindSays() []skillSays {
	var says []skillSays
	for _, kind := range build.ReviewKinds {
		says = append(says, kindSays(kind))
	}
	return says
}

// kindSays is what the review skill must say about a kind of item.
func kindSays(kind string) skillSays {
	return skillSays{"`" + kind + "`", "the kind " + kind + " of lct review is not handled"}
}

// reviewFlow is the order of the commands the review skill writes: lct is
// built, then the list is asked for.
var reviewFlow = [][2]string{
	{"go build -o bin/lct ./cmd/lct", "bin/lct review --json"},
}

// termDecisionSays are what the body of the term-decision skill must say
// besides localSkillSays: where the candidates come from, how to page through
// them, the four decisions and where each is written.
var termDecisionSays = append(slices.Clone(localSkillSays),
	skillSays{"bin/lct suggest --json", "the term suggestions are not asked for"},
	skillSays{"term/", "the id of a term suggestion is not named"},
	skillSays{"bin/lct glossary derive --json", "the candidates the glossary lacks are not asked for"},
	skillSays{"--limit", "the candidates are not paged: there are about 1900"},
	skillSays{"--offset", "the candidates are not paged: there are about 1900"},
	skillSays{"`total`", "the number of candidates is not read from the result"},
	skillSays{"status: approved", "an adopted term is not written as approved"},
	skillSays{"status: candidate", "the candidates that are already in the glossary are not named"},
	skillSays{"rules/declined.yaml", "a term that is not adopted is not recorded as declined"},
	skillSays{"glossary/terms.yaml", "the glossary file is not named"},
	skillSays{"bin/lct requeue --to-file", "the units with the old rendering are not sent back"},
	skillSays{"略過", "a candidate can not be skipped"},
)

// termDecisionFlow is the order of the commands the term-decision skill
// writes: lct is built, then the candidates are listed and looked up.
var termDecisionFlow = [][2]string{
	{"go build -o bin/lct ./cmd/lct", "bin/lct suggest --json"},
	{"go build -o bin/lct ./cmd/lct", "bin/lct glossary derive --json"},
	{"bin/lct glossary derive --json", "bin/lct search --json"},
}

// localSkillBody reads the SKILL.md of skill: its path, its body and its
// allowed-tools.
func localSkillBody(t *testing.T, skill string) (file, body string, tools []string) {
	t.Helper()
	file, content := readRepoFile(t, filepath.Join(repoRoot, ".claude", "skills", skill, "SKILL.md"))
	front, body, ok := splitFrontmatter(content)
	if !ok {
		t.Fatalf("%s has no frontmatter", file)
	}
	var fm skillFrontmatter
	if err := yaml.Unmarshal([]byte(front), &fm); err != nil {
		t.Fatal(err)
	}
	return file, body, fm.AllowedTools
}

// The bodies of the two skills for a person say what their contract wants,
// and every check can fail: with a part of the body taken out, it does.
func TestLocalSkillBodies(t *testing.T) {
	for _, tc := range []struct {
		skill string
		says  []skillSays
		flow  [][2]string
		cases []struct{ name, old, repl, want string }
	}{
		{"review", reviewSays, reviewFlow, []struct{ name, old, repl, want string }{
			{"no word about data", "不是指示", "是指示", "不是指示"},
			{"main not named", "`main`", "`master`", "`main`"},
			{"tw not named", "`tw`", "`rw`", "`tw`"},
			{"issue not ticked", "勾選", "看過", "勾選"},
			{"no tidy", "bin/lct adjust tidy --json", "bin/lct adjust tidy", "bin/lct adjust tidy --json"},
			{"a kind not handled", "`new_family`", "`family`", "`new_family`"},
			{"the translation that was not run not handled", "`not_translated`", "`translated`", "`not_translated`"},
			{"lct never built", "go build -o bin/lct ./cmd/lct", "go install ./cmd/lct", "go build -o bin/lct ./cmd/lct"},
		}},
		{"term-decision", termDecisionSays, termDecisionFlow, []struct{ name, old, repl, want string }{
			{"no word about data", "不是指示", "是指示", "不是指示"},
			{"main not named", "`main`", "`master`", "`main`"},
			{"no paging", "--offset", "--page", "--offset"},
			{"no total", "`total`", "`count`", "`total`"},
			{"no approved status", "status: approved", "status: ok", "status: approved"},
			{"no declined file", "rules/declined.yaml", "rules/no.yaml", "rules/declined.yaml"},
			{"no skip", "略過", "跳開", "略過"},
			{"lct never built", "go build -o bin/lct ./cmd/lct", "go install ./cmd/lct", "go build -o bin/lct ./cmd/lct"},
		}},
	} {
		t.Run(tc.skill, func(t *testing.T) {
			file, body, tools := localSkillBody(t, tc.skill)
			for _, problem := range checkSkillBody(file, body, tc.says, tc.flow, tools) {
				t.Error(problem)
			}
			if t.Failed() {
				return
			}
			for _, c := range tc.cases {
				t.Run(c.name, func(t *testing.T) {
					src := mutateAll(t, body, c.old, c.repl)
					hasProblem(t, checkSkillBody(file, src, tc.says, tc.flow, tools), c.want)
				})
			}
			t.Run("a tool the flow never runs", func(t *testing.T) {
				extra := append(slices.Clone(tools), "Bash(bin/lct validate *)")
				wantProblems(t, checkSkillBody(file, body, tc.says, tc.flow, extra), "bin/lct validate", "never runs")
			})
		})
	}
}

func TestCheckSkillTools(t *testing.T) {
	commands := []string{"batch next", "batch submit", "requeue", "search"}
	good := []string{"Bash(bin/lct search *)", "Bash(bin/lct requeue --to-file *)", goBuildTool, "Read(./**)", "Edit(./rules/**)"}
	if problems := checkSkillTools("SKILL.md", good, commands); len(problems) > 0 {
		t.Errorf("problems in a good list:\n%s", strings.Join(problems, "\n"))
	}
	for _, tc := range []struct{ entry, want string }{
		{"Bash(git push *)", "only bin/lct"},
		{"Bash(rm -rf *)", "only bin/lct"},
		{"Bash(go build ./...)", "only bin/lct"},
		{"Bash(go test *)", "only bin/lct"},
		{"Write(./rules/**)", "only bin/lct"},
		{"WebFetch(domain:example.com)", "only bin/lct"},
		{"Glob", "only bin/lct"},
		{"Bash(bin/lct *)", "every lct command"},
		{"Bash(bin/lct publish *)", "no registered"},
		{"Read", "without a scope"},
		{"Edit(**)", "without a scope"},
	} {
		t.Run(tc.entry, func(t *testing.T) {
			problems := checkSkillTools("SKILL.md", []string{tc.entry}, commands)
			if len(problems) == 0 || !strings.Contains(strings.Join(problems, "\n"), tc.want) || !strings.Contains(problems[0], "SKILL.md") {
				t.Errorf("problems = %q, want one naming SKILL.md and %q", problems, tc.want)
			}
		})
	}
}

func TestCheckPermissions(t *testing.T) {
	commands := []string{"batch next", "batch submit", "bootstrap", "impact", "pending", "requeue", "rules lint", "validate"}
	good := []string{
		"Bash(bin/lct batch *)", "Bash(bin/lct pending *)", "Bash(bin/lct validate *)", "Bash(bin/lct rules lint*)",
		"Bash(bin/lct requeue --to-file *)", "Bash(bin/lct batch --json *)", "Bash(bin/lct requeue --to-file x *)",
		"Bash(bin/lct impact --base origin/main --markdown work/proposal/impact.md*)",
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
		{"Bash(bin/lct requeue ./x *)", "more than a command and flags"},
		{"Bash(bin/lct requeue --to-file x y *)", "more than a command and flags"},
		{"Bash(bin/lct impact --base origin/main work/x.md*)", "more than a command and flags"},
		{"Bash(bin/lct publish --to-file *)", "no registered"},
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
	writeTestFile(t, root, "prompts/cli.md", "npm install -g @anthropic-ai/claude-code@1.2.3 裝的是 CLI，不是模型；claude-code-x 仍是名稱。\n")

	problems := checkModelNames(root, ".claude/skills", "prompts")
	want := [][2]string{
		{".claude/skills/demo/SKILL.md", `"Opus"`},
		{"prompts/cli.md", `"claude-code-x"`},
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

// The workflows are part of the contract too. pipeline.yml is the one that
// writes raw, tw and the Release, one run at a time; update.yml and
// rebuild.yml only call it. What the contract holds them to: what they run
// exists, they name no model, the secrets are the two the pipeline declares and
// each is read only by the step that needs it, every secret and variable is
// described in README.md, and the Release carries no forbidden name. The
// pipeline is built so that a fresh clone numbers and publishes the Release
// right (the raw and tw worktrees and every tag are there before it runs), and
// stops early only when there is nothing to do.

const (
	workflowDir = ".github/workflows"
	// pipelineUses is how a caller names the pipeline.
	pipelineUses = "./.github/workflows/pipeline.yml"
	// readmeActions is the heading of the README section that describes the
	// secrets and variables.
	readmeActions = "GitHub Actions 設定"
	// readmeWorkflows is the heading of the README section with a table row
	// for each workflow that runs by itself.
	readmeWorkflows = "各 workflow 何時執行"
)

// workflowNames are the workflows this repo must have.
var workflowNames = []string{"ci.yml", "pages.yml", "pipeline.yml", "rebuild.yml", "rules-proposal.yml", "update.yml"}

var (
	// workflowActions are the only actions a workflow may use, pinned to the
	// major version. The action of Anthropic runs the rules-review skill in
	// rules-proposal.yml and nowhere else; the actions of Node and Pages are
	// those of ci.yml (setup-node) and pages.yml (see pagesActions).
	workflowActions = []string{
		"actions/checkout@v5", "actions/setup-go@v6", "anthropics/claude-code-action@v1",
		"actions/setup-node@v7", "actions/configure-pages@v6", "actions/upload-pages-artifact@v5", "actions/deploy-pages@v5",
	}
	// workflowRunner is the runner of every job that has steps.
	workflowRunner = "ubuntu-24.04"
	// workflowTitleRe matches the value of a --title flag.
	workflowTitleRe = regexp.MustCompile(`--title(?:\s+|=)("[^"]*"|'[^']*'|\S+)`)
	// forbiddenNameRe matches the names a Release must not carry.
	forbiddenNameRe = regexp.MustCompile(`(?i)claude\s*code|anthropic`)
	// workflowRefRe matches a secret or a variable of the repo.
	workflowRefRe = regexp.MustCompile(`\b(secrets|vars)\.([A-Za-z_][A-Za-z0-9_]*)`)
	// workflowSecrets are the secrets of the repo. Neither is required: the
	// pipeline runs without them (the API key only translates, the status URL
	// only asks for a new game update).
	workflowSecrets = []string{"ANTHROPIC_API_KEY", "LIMBUS_STATUS_URL"}
	// claudeCLIRe matches the install of the translation CLI; the group is
	// its version.
	claudeCLIRe = regexp.MustCompile(`npm install -g @anthropic-ai/claude-code(?:@([^\s;&|]+))?`)
	semverRe    = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	// ghCommandRe matches a gh command that talks to GitHub with a token.
	ghCommandRe = regexp.MustCompile(`\bgh (?:release|issue)\b`)
	// jqCallRe matches a call of jq: its flags (--arg takes two words), the
	// filter, and what it reads, which is either <<<"$variable" (group 3) or
	// a file (group 4).
	jqCallRe = regexp.MustCompile(`\bjq((?:\s+(?:--arg\s+\w+\s+\S+|-{1,2}[A-Za-z][\w-]*))*)\s+('[^']*'|"[^"]*"|[^\s'"<>|;&]+)(?:\s*<<<\s*"\$(\w+)"|\s+([^\s<>|;&)]+))?`)
	// jqKeyRe matches a key a filter reads: .key, but not .[] or a dot inside
	// a word.
	jqKeyRe = regexp.MustCompile(`(?:^|[^\w\]"'$])\.([A-Za-z_]\w*)`)
	// jqExitStatusRe matches the flags of jq that make a false or null result
	// a failure.
	jqExitStatusRe = regexp.MustCompile(`(?:^|\s)(?:-[A-Za-z]*e[A-Za-z]*|--exit-status)(?:\s|$)`)
)

// pipelineInputs are the inputs of pipeline.yml and their types.
var pipelineInputs = map[string]string{
	"fetch": "boolean", "force": "boolean", "requeue": "string", "max_usd": "string", "dry_run": "boolean",
}

// pipelinePermissions are the permissions of the token of the pipeline: it
// pushes and releases, and opens the issue. The callers grant the same.
var pipelinePermissions = map[string]string{"contents": "write", "issues": "write"}

// pipelineGated are the commands of pipeline.yml that must not run when
// there is nothing to do; the step that runs one reads steps.decide.outputs
// in its if. The publish that commits and prepares the Release is gated; the
// publish --check that the decide step asks is not, as it only reads and is
// how the run decides.
var pipelineGated = []string{
	"npm install", "bin/lct translate", "git -C .work/tw push origin tw", "bin/lct build", "bin/lct publish --json", "git push origin tw", "gh release create", "gh issue",
}

// The commit lct translate --push-each-run makes after each run has the
// subject the idle check reads: otherwise a tw whose last commit kept the
// progress of a run, and was never built, would count as done.
func TestProgressSubjectIsTheOneTheIdleCheckReads(t *testing.T) {
	if progressSubject != keptProgressPrefix {
		t.Errorf("lct translate commits %q, but the pipeline keeps and checks %q", progressSubject, keptProgressPrefix)
	}
}

// keptProgressPrefix is how the subject of the commit that keeps the
// translations on tw begins; the idle check looks for it in the last commit of
// tw, so the commit step and the check must agree.
const keptProgressPrefix = "translate: keep the progress of"

// Every push of tw runs in this loop. GitHub can refuse a push that comes a
// second after the one before it (missing necessary objects: the server that
// takes it has not seen that push yet), and a try a few seconds later goes
// through. The step fails when the last try fails, so a Release is never made
// from a commit origin lacks.
// keepCommitIf is the first line of the step that keeps the translations on
// tw: it commits only when the worktree changed.
const keepCommitIf = `if [ -n "$(git -C .work/tw status --porcelain)" ]; then`

const (
	twPushRetry  = "for wait in 2 4 8 16 x; do"
	twPushGiveUp = `if [ "$wait" = x ]; then exit 1; fi`
)

// pipelineIdleTerms are the terms, in this order, of the condition under which
// the decide step ends the run with nothing to do. Every term keeps a run from
// ending early that has something to do; the terms are joined by &&.
var pipelineIdleTerms = []struct{ text, why string }{
	{`[ "$FETCH" = true ]`, "a rebuild, which asks for no game update, could end as idle"},
	{`[ "$NEW" != true ]`, "a new game update could end as idle"},
	{`[ "$FORCE" != true ]`, "a forced run could end as idle"},
	{`[ "$PENDING" -eq 0 ]`, "a run with units to translate could end as idle"},
	{`[ "$APPLIED" -eq 0 ]`, "a run that applies a request or rewrite file could end as idle"},
	{`[ -z "$REQUEUE" ]`, "a run asked by hand to requeue could end as idle"},
	{`[ "$built_tag" = "$raw_tag" ]`, "a tw that was last built from another raw snapshot could end as idle, and would never be built"},
	// A tw whose last build was committed but whose Release did not follow
	// (the Release step failed), and which no earlier term sees: nothing is
	// pending and tw was built from this raw.
	{`[ "$release_needed" = false ]`, "a Release that is still due (publish --check says so) could end as idle, and would never be made"},
	// A tw whose progress was kept and whose build then failed: the pack is
	// still the one of the last Release, so publish --check says no Release is
	// due, and what was translated would never be built.
	{`[[ $tw_subject != '` + keptProgressPrefix + `'* ]]`, "a tw whose last commit only keeps the progress of a translation, whose build never came, could end as idle"},
}

// pipelineMarkers are the commands pipeline.yml must run, with the reason.
var pipelineMarkers = []struct{ text, why string }{
	{"+refs/heads/raw:refs/remotes/origin/raw", "the raw branch is not fetched"},
	{"+refs/heads/tw:refs/remotes/origin/tw", "the tw branch is not fetched"},
	{"+refs/tags/*:refs/tags/*", "the tags are not fetched, so the Release number restarts at 1"},
	{"git worktree add .work/raw raw", "lct reads the raw worktree and does not make it for plan and build"},
	{"git worktree add .work/tw tw", "the tw worktree is not made"},
	{"meta/source.json", "the raw tag is not read from the raw worktree"},
	// The answer is assigned first, so that set -e sees a failing git.
	{"remote=$(git ls-remote --tags origin", "origin is not asked for the raw tag, or its answer is not kept before it is tested"},
	{`git push origin "refs/tags/`, "a raw tag that origin lacks is not pushed"},
	{"go build -o bin/lct ./cmd/lct", "lct is not built"},
	{"bin/lct fetch --check", "the status API is not asked before the packs are downloaded"},
	{"--out .work/fetch", "the packs are not downloaded"},
	{"bin/lct snapshot --from .work/fetch", "the packs are not snapshotted"},
	{`git push origin raw "refs/tags/`, "the snapshot and its tag are not pushed"},
	{"bin/lct plan", "the state is not carried to the snapshot"},
	{"bin/lct requeue --file", "a requested requeue is not applied"},
	{"bin/lct pending --count", "the pending units are not counted again after a requeue"},
	{"installed=true", "the install step does not say that the claude CLI is there, so the translate step never runs"},
	{"bin/lct translate --max-usd", "nothing is translated"},
	// A run of the CLI has to end at its turn limit, which is a normal end of
	// a round; with more turns than fit in 90 minutes it would be killed and
	// counted as an error, which ends the whole loop. After 100 turns a run
	// gets no new batch and ends once its batch is submitted; 130 is the hard
	// limit, for the batch it has.
	{"--deadline 5h --max-turns 130 --soft-turns 100 --run-timeout 90m", "the translation has no time limits, a turn limit that does not end a run before the 90 minutes do, or no soft turn limit, so that the hard one cuts a batch in half"},
	// Each run is kept on tw as it ends, not only once the step is over.
	{"--run-timeout 90m --push-each-run", "the translations are pushed only once the translate step is over: a runner that is lost takes every run of the step with it"},
	// What was translated is kept before anything can fail.
	{"git -C .work/tw status --porcelain", "the step that keeps the translations does not look for what changed, so it commits nothing or commits when there is nothing"},
	{"git -C .work/tw commit", "the translations are not committed to tw right after translating"},
	{"git -C .work/tw push origin tw", "the translations are not pushed to tw right after translating, so a build, publish or push that fails loses them"},
	// The idle check looks at what tw was built from, at whether a Release is
	// still due for it, and at whether its last commit only kept progress.
	{".work/tw/.meta/build.json", "the raw tag tw was built from is not read, so a tw that lags raw counts as done"},
	{"bin/lct publish --check --json", "the idle check does not ask whether a Release is still due, so a Release that failed is never made"},
	{"git -C .work/tw log -1 --format=%s", "the subject of the last commit of tw is not read, so a tw whose progress was kept and never built counts as done"},
	// created_at goes to $GITHUB_OUTPUT: nothing but a date and a time.
	{`[!A-Za-z0-9_:.+\ -]`, "the token and the date of the status API are written to the outputs unchecked"},
	{"bin/lct build", "the pack is not built"},
	{"$GITHUB_STEP_SUMMARY", "the report is not shown on the summary page"},
	{"bin/lct publish --json", "the Release is not prepared"},
	// A Release that is skipped says so on the summary page.
	{"語言包與 $same_as 相同，這次不建立 Release", "a run that makes no Release because the pack is the one of the last does not say so on the summary page"},
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
	{"::error::publish.json has ${#assets[@]} assets, want 2", "a Release step that refuses to run without both zips fails without a word"},
	// Listed, not searched: the search index lags behind and does not hold
	// Chinese titles well.
	{"gh issue list --state open --limit 500 --json number,title,url", "the open issues are not listed, so a rerun opens a second issue"},
	{"gh issue create --title", "the review issue is not opened"},
	{"gh issue comment", "the parts after the first are not posted"},
}

// pipelineOrder pairs the commands of pipeline.yml that must run in that
// order: the first before the second, by the first place each is written.
var pipelineOrder = [][2]string{
	{"+refs/tags/*:refs/tags/*", "bin/lct publish"}, // the tags number the Release
	{"git worktree add .work/raw raw", "meta/source.json"},
	{"meta/source.json", "remote=$(git ls-remote --tags origin"},
	{"git worktree add .work/raw raw", "bin/lct fetch --check"},
	{"git worktree add .work/tw tw", "bin/lct plan"},
	{"go build -o bin/lct ./cmd/lct", "bin/lct fetch --check"},
	{"bin/lct fetch --check", "bin/lct snapshot --from .work/fetch"},
	{"bin/lct snapshot --from .work/fetch", `git push origin raw "refs/tags/`},
	{`git push origin raw "refs/tags/`, "bin/lct plan"},
	{"bin/lct plan", "bin/lct requeue --file"},
	{"bin/lct requeue --file", "bin/lct pending --count"},
	{"bin/lct pending --count", "bin/lct translate --max-usd"},
	{"npm install -g @anthropic-ai/claude-code", "bin/lct translate --max-usd"}, // installed by a step of its own, before it
	{"bin/lct translate --max-usd", "git -C .work/tw commit"},
	{"git -C .work/tw commit", "git -C .work/tw push origin tw"},
	{"git -C .work/tw push origin tw", "bin/lct build"}, // kept before anything that can fail
	{"bin/lct translate --max-usd", "bin/lct build"},
	{"bin/lct translate --not-run", "bin/lct build"}, // the build reads the record
	// The record is made once the plan and a requeue counted what is pending,
	// and after the install step, whose output its if reads.
	{"bin/lct pending --count", "bin/lct translate --not-run"},
	{"npm install -g @anthropic-ai/claude-code", "bin/lct translate --not-run"},
	// The idle check asks about tw as the last run left it, before this run
	// commits to it.
	{"bin/lct publish --check", "git -C .work/tw commit"},
	{"git -C .work/tw log -1 --format=%s", "git -C .work/tw commit"},
	{"bin/lct build", "bin/lct publish --json"},
	{"bin/lct publish --json", "git push origin tw"},
	{"git push origin tw", "gh release create"}, // the Release is made from a pushed commit
	{"assets_json=$(jq -ec", "mapfile -t assets"},
	{"mapfile -t assets", `"${#assets[@]}" -eq 2`},
	{`"${#assets[@]}" -eq 2`, "gh release create"}, // no Release without its zips
	{"gh release create", "gh issue create --title"},
	{"gh issue list --state open --limit 500", "gh issue create --title"},
}

// workflowStep is a step of a job, as far as the contract reads it.
type workflowStep struct {
	ID               string            `yaml:"id"`
	If               string            `yaml:"if"`
	Uses             string            `yaml:"uses"`
	Run              string            `yaml:"run"`
	Shell            string            `yaml:"shell"`
	WorkingDirectory string            `yaml:"working-directory"`
	ContinueOnError  any               `yaml:"continue-on-error"`
	With             map[string]any    `yaml:"with"`
	Env              map[string]string `yaml:"env"`
}

// workflowJob is a job: one with steps, or one that calls a workflow.
type workflowJob struct {
	If              string `yaml:"if"`
	RunsOn          string `yaml:"runs-on"`
	TimeoutMinutes  int    `yaml:"timeout-minutes"`
	ContinueOnError any    `yaml:"continue-on-error"`
	Environment     any    `yaml:"environment"` // a name, or a map with a name and a url
	Concurrency     struct {
		Group            string `yaml:"group"`
		CancelInProgress bool   `yaml:"cancel-in-progress"`
	} `yaml:"concurrency"`
	Defaults struct {
		Run struct {
			Shell string `yaml:"shell"`
		} `yaml:"run"`
	} `yaml:"defaults"`
	Permissions map[string]string `yaml:"permissions"`
	Uses        string            `yaml:"uses"`
	With        map[string]any    `yaml:"with"`
	Secrets     any               `yaml:"secrets"` // a map, or "inherit"
	Steps       []workflowStep    `yaml:"steps"`
}

// workflowInput is an input of workflow_call or workflow_dispatch.
type workflowInput struct {
	Type     string `yaml:"type"`
	Required bool   `yaml:"required"`
}

// workflowEvents is what the workflow says about the events it runs on.
type workflowEvents struct {
	WorkflowCall struct {
		Inputs  map[string]workflowInput `yaml:"inputs"`
		Secrets map[string]struct {
			Required bool `yaml:"required"`
		} `yaml:"secrets"`
	} `yaml:"workflow_call"`
	WorkflowDispatch struct {
		Inputs map[string]workflowInput `yaml:"inputs"`
	} `yaml:"workflow_dispatch"`
	Schedule []struct {
		Cron string `yaml:"cron"`
	} `yaml:"schedule"`
	RepositoryDispatch struct {
		Types []string `yaml:"types"`
	} `yaml:"repository_dispatch"`
	Push struct {
		Branches []string `yaml:"branches"`
		Paths    []string `yaml:"paths"`
	} `yaml:"push"`
	WorkflowRun struct {
		Workflows []string `yaml:"workflows"`
		Types     []string `yaml:"types"`
	} `yaml:"workflow_run"`
}

// workflowFile is the part of a workflow file the contract reads.
type workflowFile struct {
	Name        string `yaml:"name"`
	On          any    `yaml:"on"`
	Concurrency struct {
		Group string `yaml:"group"`
	} `yaml:"concurrency"`
	Permissions map[string]string      `yaml:"permissions"`
	Jobs        map[string]workflowJob `yaml:"jobs"`
}

// jobNames returns the names of the jobs, sorted.
func (w *workflowFile) jobNames() []string {
	return slices.Sorted(maps.Keys(w.Jobs))
}

// steps returns the steps of all jobs, the jobs in name order.
func (w *workflowFile) steps() []workflowStep {
	var steps []workflowStep
	for _, name := range w.jobNames() {
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

// events returns what "on" says about the events, when it is a map.
func (w *workflowFile) events() workflowEvents {
	var ev workflowEvents
	if on, ok := w.On.(map[string]any); ok {
		if data, err := yaml.Marshal(on); err == nil {
			_ = yaml.Unmarshal(data, &ev)
		}
	}
	return ev
}

// parseWorkflow decodes a workflow file. A file that is not YAML, or has no
// job with a step or a workflow to call, gives the problems and no workflow.
func parseWorkflow(file, content string) (*workflowFile, []string) {
	var w workflowFile
	if err := yaml.Unmarshal([]byte(content), &w); err != nil {
		return nil, []string{fmt.Sprintf("%s: not valid YAML: %v", file, err)}
	}
	if !slices.ContainsFunc(slices.Collect(maps.Values(w.Jobs)), func(j workflowJob) bool { return len(j.Steps) > 0 || j.Uses != "" }) {
		return nil, []string{file + ": has no job with a step or a workflow to call"}
	}
	return &w, nil
}

func workflowPath(name string) string { return workflowDir + "/" + name }

// checkWorkflowRunNames checks that the workflows rules-proposal.yml waits for
// (workflow_run.workflows, which GitHub matches against the name: of a
// workflow, not its file) are update.yml and rebuild.yml by their names: a
// workflow renamed would never start it again, and no test of one file sees
// that. files holds the workflows by file name.
func checkWorkflowRunNames(files map[string]string) []string {
	var problems []string
	var want []string
	for _, name := range []string{"update.yml", "rebuild.yml"} {
		w, p := parseWorkflow(workflowPath(name), files[name])
		if w == nil {
			return append(problems, p...)
		}
		want = append(want, w.Name)
	}
	rules, p := parseWorkflow(workflowPath("rules-proposal.yml"), files["rules-proposal.yml"])
	if rules == nil {
		return append(problems, p...)
	}
	if got := rules.events().WorkflowRun.Workflows; !slices.Equal(slices.Sorted(slices.Values(got)), slices.Sorted(slices.Values(want))) {
		problems = append(problems, fmt.Sprintf("%s: workflow_run workflows = %v, want the names of update.yml and rebuild.yml %v", workflowPath("rules-proposal.yml"), got, want))
	}
	return problems
}

// checkWorkflowSet reports the workflows of workflowNames that are not among
// have (file names).
func checkWorkflowSet(have []string) []string {
	var problems []string
	for _, want := range workflowNames {
		if !slices.Contains(have, want) {
			problems = append(problems, workflowPath(want)+" does not exist")
		}
	}
	return problems
}

// checkWorkflow checks a workflow file (a slash path relative to the root)
// whose text is content, whatever it does: that it parses, that the lct
// commands it runs are registered, that it runs on the pinned runner and uses
// only the pinned actions, that its scripts hold no expression and name no
// secret (expressions reach a script through env), that its secrets are the
// two of the repo, that no caller passes every secret on, and that the --title
// of a Release has no forbidden name.
func checkWorkflow(file, content string, commands []string) []string {
	w, problems := parseWorkflow(file, content)
	if w == nil {
		return problems
	}
	problems = append(problems, checkLctCommands(file, content, commands)...)
	for _, name := range w.jobNames() {
		job := w.Jobs[name]
		if len(job.Steps) > 0 && job.RunsOn != workflowRunner {
			problems = append(problems, fmt.Sprintf("%s: job %s runs on %q, want %s", file, name, job.RunsOn, workflowRunner))
		}
		if inherit, ok := job.Secrets.(string); ok {
			problems = append(problems, fmt.Sprintf("%s: job %s has secrets: %s, which hands every secret of the repo to the callee; list the ones it needs", file, name, inherit))
		}
	}
	for _, s := range w.steps() {
		if s.Uses != "" && !slices.Contains(workflowActions, s.Uses) {
			problems = append(problems, fmt.Sprintf("%s: uses %q, which is not one of %s", file, s.Uses, strings.Join(workflowActions, ", ")))
		}
		if strings.Contains(s.Run, "${{") {
			problems = append(problems, fmt.Sprintf("%s: a script holds an expression (${{ }}); pass the value through env: %q", file, firstLine(s.Run)))
		}
		if strings.Contains(s.Run, "secrets.") || strings.Contains(s.Run, "LIMBUS_STATUS_URL") {
			problems = append(problems, fmt.Sprintf("%s: a script names a secret; lct reads LIMBUS_STATUS_URL from its own environment: %q", file, firstLine(s.Run)))
		}
	}
	for _, m := range workflowRefRe.FindAllStringSubmatch(content, -1) {
		if m[1] == "secrets" && !slices.Contains(workflowSecrets, m[2]) {
			problems = append(problems, fmt.Sprintf("%s: refers to secrets.%s, which is not one of %s", file, m[2], strings.Join(workflowSecrets, ", ")))
		}
	}
	for _, m := range workflowTitleRe.FindAllStringSubmatch(content, -1) {
		if forbiddenNameRe.MatchString(m[1]) {
			problems = append(problems, fmt.Sprintf("%s: --title %s names the Release with a name it must not carry", file, m[1]))
		}
	}
	return problems
}

// firstLine returns the first line of a script, to say which step it is.
func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}

// checkPipelineWorkflow checks pipeline.yml: it is only called, takes the
// inputs and the two optional secrets, and has one job that runs one at a
// time with the permissions it writes with. Its steps fetch the history, the
// branches and every tag, make the worktrees, give origin the raw tag, ask the
// status API, snapshot, plan, decide, install the claude CLI, translate, keep
// the translations on tw, build, publish, release and open the issue, in that
// order; only the steps that need a secret have it (the install of the CLI has
// none); the run ends early only under the exact idle condition; nothing but
// the first steps runs when there is nothing to do; the translations are
// pushed to tw before anything that can fail; and the Release is made only
// when the publish says it is needed.
func checkPipelineWorkflow(file, content string) []string {
	w, problems := parseWorkflow(file, content)
	if w == nil {
		return problems
	}
	add := func(format string, args ...any) { problems = append(problems, file+": "+fmt.Sprintf(format, args...)) }

	if events := w.triggers(); !slices.Equal(events, []string{"workflow_call"}) {
		add("runs on %v, want only workflow_call", events)
	}
	call := w.events().WorkflowCall
	for _, name := range slices.Sorted(maps.Keys(pipelineInputs)) {
		in, ok := call.Inputs[name]
		switch {
		case !ok:
			add("has no input %q", name)
		case in.Type != pipelineInputs[name]:
			add("input %q has type %q, want %q", name, in.Type, pipelineInputs[name])
		}
	}
	for _, name := range slices.Sorted(maps.Keys(call.Inputs)) {
		if _, ok := pipelineInputs[name]; !ok {
			add("input %q is not one of the inputs of the contract", name)
		}
	}
	for _, name := range workflowSecrets {
		secret, ok := call.Secrets[name]
		switch {
		case !ok:
			add("does not declare the secret %s", name)
		case secret.Required:
			add("the secret %s is required, but the pipeline must run without it", name)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(call.Secrets)) {
		if !slices.Contains(workflowSecrets, name) {
			add("declares the secret %s, which is not one of %s", name, strings.Join(workflowSecrets, ", "))
		}
	}

	if len(w.Jobs) != 1 {
		add("has %d jobs, want one", len(w.Jobs))
		return problems
	}
	job := w.Jobs[w.jobNames()[0]]
	if job.TimeoutMinutes != 350 {
		add("timeout-minutes = %d, want 350", job.TimeoutMinutes)
	}
	if job.Concurrency.Group != "lct-state" || job.Concurrency.CancelInProgress {
		add("concurrency = {group: %q, cancel-in-progress: %v}, want {group: \"lct-state\", cancel-in-progress: false}",
			job.Concurrency.Group, job.Concurrency.CancelInProgress)
	}
	if !maps.Equal(job.Permissions, pipelinePermissions) {
		add("permissions = %v, want %v", job.Permissions, pipelinePermissions)
	}
	deep := slices.ContainsFunc(job.Steps, func(s workflowStep) bool {
		return strings.HasPrefix(s.Uses, "actions/checkout@") && fmt.Sprint(s.With["fetch-depth"]) == "0"
	})
	if !deep {
		add("no actions/checkout step with fetch-depth: 0")
	}

	runs := w.runs()
	for _, m := range pipelineMarkers {
		if !strings.Contains(runs, m.text) {
			add("no %q: %s", m.text, m.why)
		}
	}
	for _, o := range pipelineOrder {
		first, second := strings.Index(runs, o[0]), strings.Index(runs, o[1])
		if first >= 0 && second >= 0 && first > second {
			add("%q must come before %q", o[0], o[1])
		}
	}
	pins := claudeCLIRe.FindAllStringSubmatch(runs, -1)
	if len(pins) == 0 {
		add("does not install the claude CLI")
	}
	for _, m := range pins {
		if !semverRe.MatchString(m[1]) {
			add("installs the claude CLI as %q, want an exact version such as @1.2.3", m[0])
		}
	}

	for _, s := range job.Steps {
		if _, ok := s.Env["LIMBUS_STATUS_URL"]; ok && !strings.Contains(s.Run, "bin/lct fetch --check") {
			add("the status API URL is in the environment of a step that does not ask the status API: %q", firstLine(s.Run))
		}
		if _, ok := s.Env["ANTHROPIC_API_KEY"]; ok && !strings.Contains(s.Run, "bin/lct translate") {
			add("the API key is in the environment of a step that does not translate: %q", firstLine(s.Run))
		}
		if m := ghCommandRe.FindString(s.Run); m != "" && s.Env["GH_TOKEN"] != "${{ github.token }}" {
			add("the step that runs %q has no GH_TOKEN: ${{ github.token }}", m)
		}
		for _, command := range pipelineGated {
			if strings.Contains(s.Run, command) && !strings.Contains(s.If, "steps.decide.outputs.") {
				add("%q runs even when there is nothing to do: the if of its step does not read steps.decide.outputs", command)
			}
		}
	}
	find := func(sub string) (workflowStep, bool) {
		i := slices.IndexFunc(job.Steps, func(s workflowStep) bool { return strings.Contains(s.Run, sub) })
		if i < 0 {
			return workflowStep{}, false
		}
		return job.Steps[i], true
	}
	if s, ok := find("bin/lct fetch --check"); ok && s.Env["LIMBUS_STATUS_URL"] != "${{ secrets.LIMBUS_STATUS_URL }}" {
		add("the step that asks the status API has no LIMBUS_STATUS_URL: ${{ secrets.LIMBUS_STATUS_URL }} in its env")
	}
	if s, ok := find("--out .work/fetch"); ok && s.Env["LIMBUS_CDN_TEMPLATE"] != "${{ vars.LIMBUS_CDN_TEMPLATE }}" {
		add("the step that downloads the packs has no LIMBUS_CDN_TEMPLATE: ${{ vars.LIMBUS_CDN_TEMPLATE }} in its env")
	}
	// The condition of the idle exit: each term, then all of them joined by &&.
	if s, ok := find("idle=true"); ok {
		lines := strings.Split(s.Run, "\n")
		at := slices.IndexFunc(lines, func(l string) bool { return strings.TrimSpace(l) == "idle=true" })
		cond := ""
		if at > 0 {
			cond = strings.TrimSpace(lines[at-1])
		}
		var terms []string
		lacking := false
		for _, term := range pipelineIdleTerms {
			terms = append(terms, term.text)
			if !strings.Contains(cond, term.text) {
				lacking = true
				add("the idle condition lacks %s: %s", term.text, term.why)
			}
		}
		if want := "if " + strings.Join(terms, " && ") + "; then"; !lacking && cond != want {
			add("the terms of the idle condition are not joined by && as %q: one term that is false must be enough to keep the run going", want)
		}
	} else {
		add("no step sets idle=true: the run never ends when there is nothing to do")
	}
	for _, s := range job.Steps {
		for _, push := range []string{"git -C .work/tw push origin tw", "git push origin tw"} {
			if strings.Contains(s.Run, push) && (!strings.Contains(s.Run, twPushRetry) || !strings.Contains(s.Run, twPushGiveUp)) {
				add("the step that runs %q tries the push once: GitHub can refuse a push of tw that comes right after another one (missing necessary objects), so the step tries again in the loop %q and fails with %q only when the last try fails", push, twPushRetry, twPushGiveUp)
			}
		}
	}
	if s, ok := find("bin/lct translate --max-usd"); ok {
		if claudeCLIRe.MatchString(s.Run) {
			add("the translate step installs the claude CLI itself: the API key is in its environment while npm runs the install scripts; install it in a step of its own")
		}
		if !strings.Contains(s.If, "steps.cli.outputs.installed == 'true'") {
			add("the translate step does not read steps.cli.outputs.installed == 'true' in its if: it would run without the claude CLI")
		}
		if s.Env["DISABLE_AUTOUPDATER"] != "1" {
			add("the translate step has no DISABLE_AUTOUPDATER: \"1\" in its env: the CLI may update itself in the middle of a run")
		}
		if keep, ok := find("git -C .work/tw push origin tw"); ok {
			if !strings.HasPrefix(keep.If, "always() && ") {
				add("the step that keeps the translations on tw does not start its if with always(): a translate step that fails or is cancelled would lose them")
			}
			if keep.If != "always() && "+s.If {
				add("the if of the step that keeps the translations on tw is %q, want always() followed by the if of the translate step: it must run exactly when the translate step does", keep.If)
			}
			if !strings.Contains(keep.Run, `-m "`+keptProgressPrefix+` $RAW_TAG"`) {
				add("the commit of the step that keeps the translations on tw has no message %q followed by the raw tag: the idle check tells a tw whose progress was kept and never built by the subject of its last commit", keptProgressPrefix)
			}
			// The commit is made when the worktree changed; the push is
			// made every time, after the if: tw can be ahead of origin
			// with a clean worktree (the push of a run failed and the
			// step was cancelled before .meta/translate.json was written),
			// and a push with nothing new does nothing.
			before, after, found := strings.Cut(keep.Run, keepCommitIf)
			body, _, closed := strings.Cut(after, "\nfi\n")
			switch {
			case !found || strings.TrimSpace(before) != "":
				add("the step that keeps the translations on tw does not begin with %q: it commits nothing, or commits when nothing changed", keepCommitIf)
			case !closed:
				add("the if of the step that keeps the translations on tw is not closed by a fi of its own line")
			case strings.Contains(keep.Run, "git -C .work/tw commit") && !strings.Contains(body, "git -C .work/tw commit"):
				add("the step that keeps the translations on tw commits outside its if: a commit with nothing changed fails the step")
			case strings.Contains(body, "push origin tw"):
				add("the step that keeps the translations on tw pushes only when the worktree changed: push after the fi, so that commits a run made and could not push are pushed with a clean worktree too")
			}
		}
		if s.Env["ANTHROPIC_API_KEY"] != "${{ secrets.ANTHROPIC_API_KEY }}" {
			add("the translate step has no ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }} in its env")
		}
		if s.Env["LCT_MODEL"] != "${{ vars.LCT_MODEL }}" {
			add("the translate step has no LCT_MODEL: ${{ vars.LCT_MODEL }} in its env")
		}
		for _, part := range []string{"inputs.max_usd", "vars.LCT_MAX_USD", "'30'"} {
			if !strings.Contains(s.Env["MAX_USD"], part) {
				add("MAX_USD of the translate step is %q, which does not use %s", s.Env["MAX_USD"], part)
			}
		}
		// A translation that fails must not stop the build: the command is
		// followed by || so that set -e lets it pass, and by a warning.
		line, after, _ := strings.Cut(s.Run[strings.Index(s.Run, "bin/lct translate"):], "\n")
		if !strings.Contains(line, "||") {
			add("the translate command is not followed by ||, so a failing translation stops the run before the build")
		}
		if !strings.Contains(after, "::warning::") {
			add("the translate step does not say with ::warning:: after the command that it failed")
		}
	}
	problems = append(problems, checkPipelineNotRun(file, job.Steps)...)
	// The line about a Release that is not made goes to the summary page, and
	// only when the publish says no Release is needed and names the Release
	// whose pack this is.
	if s, ok := find("這次不建立 Release"); ok {
		if !strings.Contains(s.Run, `>> "$GITHUB_STEP_SUMMARY"`) && !strings.Contains(s.Run, `-a "$GITHUB_STEP_SUMMARY"`) {
			add("the line that says no Release is made is not written to $GITHUB_STEP_SUMMARY")
		}
		if !strings.Contains(s.Run, `[ "$release_needed" = false ] && [ -n "$same_as" ]`) {
			add(`the line that says no Release is made is not under [ "$release_needed" = false ] && [ -n "$same_as" ]: it would name no Release, or be said when one is made`)
		}
	}
	if s, ok := find("gh release create"); ok {
		if !strings.Contains(s.If, "steps.publish.outputs.release_needed == 'true'") {
			add("the if of the gh release create step is %q, which lacks steps.publish.outputs.release_needed == 'true': the Release must not follow the tag, which is a free number even when nothing is released", s.If)
		}
		if !strings.Contains(s.If, "steps.decide.outputs.idle == 'false'") {
			add("the if of the gh release create step is %q, which lacks steps.decide.outputs.idle == 'false': no Release when there is nothing to do", s.If)
		}
		if !workflowTitleRe.MatchString(s.Run) {
			add("gh release create has no --title")
		}
	}
	return problems
}

// notRunReasonNames are what the reason the pipeline gives for a translation it
// did not run must name: the two things whose absence stops the translation,
// so that the review issue tells a person what to set.
var notRunReasonNames = []string{"ANTHROPIC_API_KEY", "LCT_MODEL"}

// notRunIf is the if of the step that records a translation that did not run:
// the key or the model is missing, or the claude CLI did not install. A dry
// run is neither.
const notRunIf = "steps.decide.outputs.not_run == 'true' || (steps.decide.outputs.translate == 'true' && steps.cli.outputs.installed == 'false')"

// notRunInstallReason is the reason of the record when the claude CLI did not
// install, chosen when decide did not say not_run.
const notRunInstallReason = `if [ "$NOT_RUN" != true ]; then
  reason="安裝 Claude Code CLI 失敗，待譯的單位沒有翻譯"
fi`

// checkPipelineNotRun checks how the pipeline treats units that are pending
// when the API key or the model is missing: the decide step tells that from a
// dry run (which skips translating on purpose) and from a run that translates,
// and a step of its own, gated on that or on a claude CLI that did not
// install, records the translation that did not run with lct translate
// --not-run, so that the build reads it and the review issue says the units
// are untranslated and why. That step needs no key, and it commits nothing:
// the publish commit takes the record to tw.
func checkPipelineNotRun(file string, steps []workflowStep) []string {
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, file+": "+fmt.Sprintf(format, args...)) }

	decide := slices.IndexFunc(steps, func(s workflowStep) bool { return s.ID == "decide" })
	if decide < 0 {
		return append(problems, file+": no step with the id decide")
	}
	d := steps[decide]
	for name, want := range map[string]string{
		"HAS_API_KEY": "${{ secrets.ANTHROPIC_API_KEY != '' }}",
		"HAS_MODEL":   "${{ vars.LCT_MODEL != '' }}",
	} {
		if d.Env[name] != want {
			add("the decide step has %s = %q, want %q: it must tell a missing key or model from a set one", name, d.Env[name], want)
		}
	}
	lines := strings.Split(d.Run, "\n")
	at := func(from int, want string) int {
		for i := from; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == want {
				return i
			}
		}
		return -1
	}
	const missing = `elif [ "$HAS_API_KEY" != true ] || [ "$HAS_MODEL" != true ]; then`
	dry := at(0, `if [ "$DRY_RUN" = true ]; then`)
	elif := at(max(dry, 0), missing)
	switch {
	case dry < 0:
		add(`the decide step does not test if [ "$DRY_RUN" = true ]; then first: a dry run skips translating on purpose and must not count as a translation that did not run`)
	case elif < 0:
		add("the decide step lacks %s: a missing key or a missing model, either, after the test of the dry run, is a translation that did not run", missing)
	default:
		other := at(elif+1, "else")
		end := at(max(other, 0), "fi")
		if other < 0 || end < 0 {
			add("the decide step has no else after the branch of a missing key or model")
			break
		}
		branch, rest := strings.Join(lines[elif+1:other], "\n"), strings.Join(lines[other+1:end], "\n")
		if !strings.Contains(branch, "not_run=true") || strings.Contains(branch, "translate=true") {
			add("the branch of a missing key or model must set not_run=true and not translate=true")
		}
		if !strings.Contains(rest, "translate=true") || strings.Contains(rest, "not_run=true") {
			add("the branch that translates must set translate=true and not not_run=true")
		}
	}
	for _, want := range []string{"not_run=false", `echo "not_run=$not_run"`} {
		if !strings.Contains(d.Run, want) {
			add("the decide step lacks %q: the record of a translation that did not run reads steps.decide.outputs.not_run", want)
		}
	}
	// A pipeline that does not ask at all is a problem of pipelineMarkers.
	asked := slices.ContainsFunc(steps, func(s workflowStep) bool { return strings.Contains(s.Run, "bin/lct publish --check --json") })
	if asked && !strings.Contains(d.Run, "release_check=$(bin/lct publish --check --json)") {
		add("the decide step does not ask release_check=$(bin/lct publish --check --json): whether a Release is still due is part of the idle check, and asked of tw as the last run left it")
	}

	i := slices.IndexFunc(steps, func(s workflowStep) bool { return strings.Contains(s.Run, "bin/lct translate --not-run") })
	if i < 0 {
		return append(problems, file+": no step runs bin/lct translate --not-run: units that are pending while the key or the model is missing are untranslated and nothing says so")
	}
	s := steps[i]
	if s.If != notRunIf {
		add("the step that runs bin/lct translate --not-run has the if %q, want %s: it records a translation that did not run, only when there is one (a dry run is none)", s.If, notRunIf)
	}
	if s.Env["NOT_RUN"] != "${{ steps.decide.outputs.not_run }}" || !strings.Contains(s.Run, notRunInstallReason) {
		add("the step that runs bin/lct translate --not-run does not give the reason %q when decide did not say not_run (NOT_RUN: ${{ steps.decide.outputs.not_run }}): a claude CLI that did not install would be recorded as a missing key", "安裝 Claude Code CLI 失敗")
	}
	if _, ok := s.Env["ANTHROPIC_API_KEY"]; ok {
		add("the step that runs bin/lct translate --not-run has ANTHROPIC_API_KEY in its environment: it runs when the key may be missing, and runs no claude")
	}
	for _, name := range notRunReasonNames {
		if !strings.Contains(s.Run, name) {
			add("the reason of bin/lct translate --not-run does not name %s: the review issue would not say what to set", name)
		}
	}
	if strings.Contains(s.Run, "git ") {
		add("the step that runs bin/lct translate --not-run uses git: it commits nothing, the publish commit takes .meta/translate.json to tw")
	}
	if !notRunGuardRe.MatchString(s.Run) {
		add(`the step that runs bin/lct translate --not-run does not go on when it fails (|| echo "::warning::…", as the translation does): a record that cannot be written would stop the build and the Release`)
	}
	return problems
}

// notRunGuardRe matches the call of lct translate --not-run that warns, and
// does not fail, when the record cannot be written.
var notRunGuardRe = regexp.MustCompile(`bin/lct translate --not-run "[^"]*" \|\| echo "::warning::`)

// ciPermissions are the permissions of ci.yml: it reads the repo and writes
// nothing. A pull request from a fork gets no more than this anyway; the
// workflow does not ask for more.
var ciPermissions = map[string]string{"contents": "read"}

// ciImpactMarkers are the commands the step of ci.yml that assesses a rule
// change must run, with the reason.
var ciImpactMarkers = []struct{ text, why string }{
	// The three dots compare with the merge base, so that what the base branch
	// gained since the branch was made does not count as a change of the pull
	// request. The result is assigned first, so that set -e sees a failing git.
	{`changed=$(git diff --name-only "origin/$BASE_REF...HEAD" -- guides glossary rules)`, "the pull request is not asked whether it changes guides/, glossary/ or rules/ (compared with the merge base, assigned before it is tested)"},
	{`[ -z "$changed" ]`, "a pull request that changes no rule is assessed all the same"},
	{"+refs/heads/raw:refs/remotes/origin/raw", "the raw branch is not fetched"},
	{"+refs/heads/tw:refs/remotes/origin/tw", "the tw branch is not fetched"},
	{"git worktree add .work/raw raw", "lct impact reads the raw worktree and it is not made"},
	{"git worktree add .work/tw tw", "lct impact reads the tw worktree and it is not made"},
	{`bin/lct impact --base "origin/$BASE_REF"`, "the rules of the base branch are not the ones to compare with"},
	{"--markdown out/impact.md", "the assessment is not written for the summary page"},
	{`out/impact.md >> "$GITHUB_STEP_SUMMARY"`, "the assessment is not shown on the summary page"},
}

// ciImpactOrder pairs the commands of the assessment step that must run in
// that order: the first before the second, by the first place each is written.
var ciImpactOrder = [][2]string{
	{"go build -o bin/lct ./cmd/lct", "bin/lct impact"},
	{"changed=$(git diff", `[ -z "$changed" ]`},
	{`[ -z "$changed" ]`, "+refs/heads/raw:refs/remotes/origin/raw"}, // nothing is fetched when no rule changed
	{"+refs/heads/raw:refs/remotes/origin/raw", "git worktree add .work/raw raw"},
	{"+refs/heads/tw:refs/remotes/origin/tw", "git worktree add .work/tw tw"},
	{"git worktree add .work/raw raw", "bin/lct impact"},
	{"git worktree add .work/tw tw", "bin/lct impact"},
	{"--markdown out/impact.md", `out/impact.md >> "$GITHUB_STEP_SUMMARY"`},
}

// checkCIWorkflow checks ci.yml: it runs on a push and on a pull request, with
// the permission to read and no more, and its checkout has the whole history.
// One step assesses a rule change (spec 11.2 and 12): it runs on a pull
// request only, gets the name of the base branch through env (a script that
// held it would run what a branch name makes of it), finds out whether the
// pull request changes the rules, and only then fetches raw and tw, makes their
// worktrees, runs lct impact against the base branch and shows the result on
// the summary page, after lct is built. A second job, site, tests and builds
// the viewer (checkCISiteJob).
func checkCIWorkflow(file, content string) []string {
	w, problems := parseWorkflow(file, content)
	if w == nil {
		return problems
	}
	add := func(format string, args ...any) { problems = append(problems, file+": "+fmt.Sprintf(format, args...)) }

	if events := w.triggers(); !slices.Equal(events, []string{"pull_request", "push"}) {
		add("runs on %v, want [pull_request push]", events)
	}
	if !maps.Equal(w.Permissions, ciPermissions) {
		add("permissions = %v, want %v: the assessment only reads", w.Permissions, ciPermissions)
	}
	for name, job := range w.Jobs {
		if len(job.Permissions) > 0 {
			add("job %s has permissions of its own %v; the workflow grants %v", name, job.Permissions, ciPermissions)
		}
	}
	steps := w.steps()
	deep := slices.ContainsFunc(steps, func(s workflowStep) bool {
		return strings.HasPrefix(s.Uses, "actions/checkout@") && fmt.Sprint(s.With["fetch-depth"]) == "0"
	})
	if !deep {
		add("no actions/checkout step with fetch-depth: 0: the base branch of a pull request and its merge base are not there to compare with")
	}
	problems = append(problems, checkCISiteJob(file, w)...)
	for _, s := range steps {
		if strings.Contains(s.Run, "base_ref") {
			add("a script names base_ref: pass it through env (BASE_REF: ${{ github.base_ref }}): %q", firstLine(s.Run))
		}
	}

	var impact []workflowStep
	for _, s := range steps {
		if strings.Contains(s.Run, "bin/lct impact") {
			impact = append(impact, s)
		}
	}
	if len(impact) != 1 {
		add("%d steps run bin/lct impact, want one: a pull request that changes the rules must say how many translations the change breaks", len(impact))
		return problems
	}
	step := impact[0]
	if !strings.Contains(step.If, "github.event_name == 'pull_request'") {
		add("the if of the step that runs bin/lct impact is %q, which lacks github.event_name == 'pull_request': there is no base branch to compare with on a push", step.If)
	}
	if step.Env["BASE_REF"] != "${{ github.base_ref }}" {
		add("the step that runs bin/lct impact has BASE_REF = %q in its env, want ${{ github.base_ref }}", step.Env["BASE_REF"])
	}
	for _, m := range ciImpactMarkers {
		if !strings.Contains(step.Run, m.text) {
			add("the step that runs bin/lct impact has no %q: %s", m.text, m.why)
		}
	}
	runs := w.runs()
	for _, o := range ciImpactOrder {
		first, second := strings.Index(runs, o[0]), strings.Index(runs, o[1])
		if first >= 0 && second >= 0 && first > second {
			add("%q must come before %q", o[0], o[1])
		}
	}
	return problems
}

// callerSpec says what a workflow that calls the pipeline must be.
type callerSpec struct {
	// inputs are the inputs the person gives by hand, and their types.
	inputs map[string]string
	// secrets are the secrets the caller passes.
	secrets []string
	// fetch is the value of the fetch input.
	fetch bool
}

// checkCaller checks a workflow that only calls pipeline.yml: one job that
// calls it, passes the inputs it has and exactly the secrets it needs by
// name, grants the permissions the pipeline writes with (a called workflow can
// only keep or reduce them), and has no concurrency group of its own (the
// pipeline holds lct-state; a caller in the same group would wait for itself).
// pipeline is the parsed pipeline.yml, nil when it did not parse.
func checkCaller(file string, w *workflowFile, pipeline *workflowFile, spec callerSpec) []string {
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, file+": "+fmt.Sprintf(format, args...)) }

	if len(w.Jobs) != 1 {
		add("has %d jobs, want one that calls pipeline.yml", len(w.Jobs))
		return problems
	}
	job := w.Jobs[w.jobNames()[0]]
	if job.Uses != pipelineUses {
		add("the job calls %q, want %s", job.Uses, pipelineUses)
	}
	if !maps.Equal(job.Permissions, pipelinePermissions) {
		add("the job has permissions %v, want %v: a called workflow gets no more than its caller", job.Permissions, pipelinePermissions)
	}
	if job.Concurrency.Group != "" || w.Concurrency.Group != "" {
		add("has a concurrency group of its own; pipeline.yml holds lct-state, and a caller in the same group waits for itself")
	}
	switch secrets := job.Secrets.(type) {
	case map[string]any:
		for _, name := range slices.Sorted(maps.Keys(secrets)) {
			if !slices.Contains(spec.secrets, name) {
				add("passes the secret %s, which this workflow does not need", name)
			} else if secrets[name] != "${{ secrets."+name+" }}" {
				add("passes %s as %v, want ${{ secrets.%s }}", name, secrets[name], name)
			}
		}
		for _, name := range spec.secrets {
			if _, ok := secrets[name]; !ok {
				add("does not pass the secret %s", name)
			}
		}
	case string:
		// checkWorkflow reports secrets: inherit.
	default:
		add("passes no secrets, want %s", strings.Join(spec.secrets, ", "))
	}

	if fetch, ok := job.With["fetch"].(bool); !ok || fetch != spec.fetch {
		add("passes fetch = %v, want %v", job.With["fetch"], spec.fetch)
	}
	for _, name := range slices.Sorted(maps.Keys(spec.inputs)) {
		if expr, _ := job.With[name].(string); !strings.Contains(expr, "inputs."+name) {
			add("passes %s as %v, want an expression of inputs.%s", name, job.With[name], name)
		}
	}
	if pipeline != nil {
		called := pipeline.events().WorkflowCall.Inputs
		for _, name := range slices.Sorted(maps.Keys(job.With)) {
			if _, ok := called[name]; !ok {
				add("passes %q, which pipeline.yml has no input for", name)
			}
		}
		for _, name := range slices.Sorted(maps.Keys(called)) {
			if _, ok := job.With[name]; !ok && called[name].Required {
				add("does not pass the required input %q of pipeline.yml", name)
			}
		}
	}
	return problems
}

// checkDispatchInputs reports the manual inputs of ev that are not exactly
// want, name and type.
func checkDispatchInputs(file string, ev workflowEvents, want map[string]string) []string {
	var problems []string
	have := ev.WorkflowDispatch.Inputs
	for _, name := range slices.Sorted(maps.Keys(want)) {
		in, ok := have[name]
		switch {
		case !ok:
			problems = append(problems, fmt.Sprintf("%s: workflow_dispatch has no input %q", file, name))
		case in.Type != want[name]:
			problems = append(problems, fmt.Sprintf("%s: workflow_dispatch input %q has type %q, want %q", file, name, in.Type, want[name]))
		}
	}
	for _, name := range slices.Sorted(maps.Keys(have)) {
		if _, ok := want[name]; !ok {
			problems = append(problems, fmt.Sprintf("%s: workflow_dispatch has input %q, which is not one of the inputs of the contract", file, name))
		}
	}
	return problems
}

// checkEvents reports the events of w when they are not exactly want.
func checkEvents(file string, w *workflowFile, want ...string) []string {
	if got := w.triggers(); !slices.Equal(got, want) {
		return []string{fmt.Sprintf("%s: runs on %v, want %v", file, got, want)}
	}
	return nil
}

// updateCron is the schedule of update.yml: Thursday 11:35 in UTC+8.
const updateCron = "35 3 * * 4"

// checkUpdateWorkflow checks update.yml: every Thursday, on a dispatch of the
// update event and by hand, it calls the pipeline with fetch on.
func checkUpdateWorkflow(file, content, pipeline string) []string {
	w, problems := parseWorkflow(file, content)
	if w == nil {
		return problems
	}
	parsed, _ := parseWorkflow(workflowPath("pipeline.yml"), pipeline)
	inputs := map[string]string{"force": "boolean", "requeue": "string", "max_usd": "string", "dry_run": "boolean"}
	problems = append(problems, checkCaller(file, w, parsed, callerSpec{inputs: inputs, secrets: workflowSecrets, fetch: true})...)
	problems = append(problems, checkEvents(file, w, "repository_dispatch", "schedule", "workflow_dispatch")...)
	problems = append(problems, checkDispatchInputs(file, w.events(), inputs)...)
	ev := w.events()
	if len(ev.Schedule) != 1 || ev.Schedule[0].Cron != updateCron {
		problems = append(problems, fmt.Sprintf("%s: schedule = %v, want one cron %q", file, ev.Schedule, updateCron))
	}
	if !slices.Equal(ev.RepositoryDispatch.Types, []string{"limbus_update"}) {
		problems = append(problems, fmt.Sprintf("%s: repository_dispatch types = %v, want [limbus_update]", file, ev.RepositoryDispatch.Types))
	}
	return problems
}

// rebuildPaths are the paths of main whose change rebuilds the pack.
var rebuildPaths = []string{"rules/**", "glossary/**", "guides/**", "lct.yaml"}

// checkRebuildWorkflow checks rebuild.yml: on a push to main that changes the
// rules, and by hand, it calls the pipeline with fetch off.
func checkRebuildWorkflow(file, content, pipeline string) []string {
	w, problems := parseWorkflow(file, content)
	if w == nil {
		return problems
	}
	parsed, _ := parseWorkflow(workflowPath("pipeline.yml"), pipeline)
	inputs := map[string]string{"requeue": "string", "max_usd": "string", "dry_run": "boolean"}
	problems = append(problems, checkCaller(file, w, parsed, callerSpec{inputs: inputs, secrets: []string{"ANTHROPIC_API_KEY"}, fetch: false})...)
	problems = append(problems, checkEvents(file, w, "push", "workflow_dispatch")...)
	problems = append(problems, checkDispatchInputs(file, w.events(), inputs)...)
	ev := w.events()
	if !slices.Equal(ev.Push.Branches, []string{"main"}) {
		problems = append(problems, fmt.Sprintf("%s: push branches = %v, want [main]", file, ev.Push.Branches))
	}
	if !slices.Equal(slices.Sorted(slices.Values(ev.Push.Paths)), slices.Sorted(slices.Values(rebuildPaths))) {
		problems = append(problems, fmt.Sprintf("%s: push paths = %v, want %v", file, ev.Push.Paths, rebuildPaths))
	}
	return problems
}

// rulesProposalPermissions are the permissions of the job of
// rules-proposal.yml: it pushes a branch (contents), opens or edits a pull
// request, labels it (pull-requests, issues) and lets the action exchange its
// OIDC token for the token of the Claude GitHub App (id-token).
var rulesProposalPermissions = map[string]string{
	"contents": "write", "pull-requests": "write", "issues": "write", "id-token": "write",
}

const (
	// rulesProposalIf is the condition of the job: a finished run of update or
	// rebuild triggers the workflow whatever it ended in and on whatever branch
	// it ran, and only a green one of main has a build to learn from; a run by
	// hand is not held back.
	rulesProposalIf = "github.event_name == 'workflow_dispatch' || (github.event.workflow_run.conclusion == 'success' && github.event.workflow_run.head_branch == 'main' && github.event.workflow_run.event != 'pull_request')"
	// rulesProposalPrompt is what the action is told: the skill, and nothing else.
	rulesProposalPrompt = "/rules-review"
	// rulesProposalDenied is the tool the action may not use: the folder of git
	// is where the checkout leaves its token, and the run reads text that anyone
	// can write.
	rulesProposalDenied = "Read(./.git/**)"
	// rulesProposalCount is what the action step and the steps after it read to
	// run only when lct suggest found something.
	rulesProposalCount = "steps.suggest.outputs.count != '0'"
	// rulesProposalHasModel and rulesProposalNoModel are what the action step
	// and the step that stands in for it read: without LCT_MODEL there is no
	// model to run the action with.
	rulesProposalHasModel = "vars.LCT_MODEL != ''"
	rulesProposalNoModel  = "vars.LCT_MODEL == ''"
	// rulesProposalInCI and rulesProposalNotInCI are what the action step and
	// the step that stands in for it read: the review runs in CI only when the
	// variable RULES_REVIEW_IN_CI is true, and otherwise waits for a person to
	// run /rules-review on their machine.
	rulesProposalInCI    = "vars.RULES_REVIEW_IN_CI == 'true'"
	rulesProposalNotInCI = "vars.RULES_REVIEW_IN_CI != 'true'"
)

var (
	// claudeAllowedToolsRe matches the --allowedTools flag of claude_args and its
	// quoted, comma-separated list; the group is the list.
	claudeAllowedToolsRe = regexp.MustCompile(`--allowedTools\s+"([^"]*)"`)
	claudeMaxTurnsRe     = regexp.MustCompile(`(?:^|\s)--max-turns 80(?:\s|$)`)
	// claudeDisallowedToolsRe matches the --disallowedTools flag and its list.
	claudeDisallowedToolsRe = regexp.MustCompile(`--disallowedTools\s+"([^"]*)"`)
)

// rulesProposalMarkers are the commands the scripts of rules-proposal.yml must
// run, with the reason.
var rulesProposalMarkers = []struct{ text, why string }{
	{"+refs/heads/raw:refs/remotes/origin/raw", "the raw branch is not fetched"},
	{"+refs/heads/tw:refs/remotes/origin/tw", "the tw branch is not fetched"},
	{"git worktree add .work/raw raw", "lct suggest and lct impact read the raw worktree and it is not made"},
	{"git worktree add .work/tw tw", "lct suggest and lct impact read the tw worktree and it is not made"},
	{"go build -o bin/lct ./cmd/lct", "lct is not built"},
	{"git config user.name", "git has no identity to commit the rules with"},
	{"git config user.email", "git has no identity to commit the rules with"},
	{`bin/lct proposal start --date "$(date -u +%F)" --json`, "the branch of the rules PR is not chosen, nor the ids left out written"},
	{"bin/lct suggest --json --exclude-file work/proposal/exclude.txt", "the suggestions are not counted with the ids of the open PR, the declined and the handled left out, so the run would go on for what is in a PR already"},
	// The PR is looked for by its branch: the label goes through the search
	// index, which lags behind a PR that was just opened.
	{`gh pr list --state open --head "$BRANCH"`, "the rules PR is not looked for, so a run that opened none is green"},
	// What the skill says it did decides what is checked: when lct proposal
	// publish refuses, the action does not fail.
	{"outcome_file=work/proposal/outcome.txt", "the file the skill says what it did in is not read, so a run whose publish was refused is green when a rules PR is open"},
	{"'nothing: '*)", "a run that proposes nothing is not told from one that failed"},
	{"'nothing: 本機'*)", "a run that took itself for one on a person's machine (it found no work/proposal/started.json) and published nothing is green"},
	{"::notice::the rules-review run proposed nothing", "a run that proposes nothing does not say so in the log"},
	{"--json url,body,isCrossRepository", "the description of the PR is not read, so a PR that is open already is taken for a PR this run added to"},
	{`grep -qxF -- "$id" work/proposal/exclude.txt`, "the suggestions the PR had before the run are not told from those it added"},
	{`>> "$GITHUB_STEP_SUMMARY"`, "the address of the rules PR, or that there was nothing to suggest, is not on the summary page"},
	// The action hides the session; its last message is all a failed run
	// shows of why.
	{`select(.type == "result")`, "a review that failed does not say why"},
}

// rulesProposalOrder pairs the commands of the scripts of rules-proposal.yml
// that must run in that order: the first before the second, by the first place
// each is written.
var rulesProposalOrder = [][2]string{
	{"+refs/heads/raw:refs/remotes/origin/raw", "git worktree add .work/raw raw"},
	{"+refs/heads/tw:refs/remotes/origin/tw", "git worktree add .work/tw tw"},
	{"git worktree add .work/raw raw", "bin/lct suggest"},
	{"git worktree add .work/tw tw", "bin/lct suggest"},
	{"go build -o bin/lct ./cmd/lct", "bin/lct proposal start"},
	{"git config user.name", "bin/lct proposal start"},
	{"git config user.email", "bin/lct proposal start"},
	{"bin/lct proposal start", "bin/lct suggest"}, // the ids to leave out are written by start
	{"bin/lct suggest", "gh pr list"},
}

// rulesProposalJSONInputs are what rules-proposal.yml feeds to jq: the output
// of lct proposal start and of lct suggest, and of gh, as pipeline.yml does
// (see lctJSONInputs).
func rulesProposalJSONInputs() map[string]any {
	return map[string]any{
		"started": proposal.Started{},
		"suggest": suggestReport{},
		"prs":     nil, // gh pr list
		// The messages of the session of the action (its execution file),
		// read when the session failed.
		"execution": nil,
	}
}

// rulesProposalJSONRequired are the keys rules-proposal.yml must read.
var rulesProposalJSONRequired = map[string][]string{
	"started": {"branch"},
	"suggest": {"suggestions"},
}

// checkRulesProposalWorkflow checks rules-proposal.yml, whose allowed tools
// must be those of the rules-review skill (skillTools): it runs when update or
// rebuild finished green, and by hand; one job at a time, with the permissions
// the rules PR is made with, whose steps fetch the history and the raw and tw
// branches, build lct, choose the branch of the rules PR, count the suggestions
// that are not in a PR yet, run the skill with the action only when there are
// some, and look for the rules PR. The action gets the key, the model and
// exactly the tools of the skill; no step but it has the key, and it has no
// token of its own to override that of the app.
func checkRulesProposalWorkflow(file, content string, skillTools []string) []string {
	w, problems := parseWorkflow(file, content)
	if w == nil {
		return problems
	}
	add := func(format string, args ...any) { problems = append(problems, file+": "+fmt.Sprintf(format, args...)) }

	if events := w.triggers(); !slices.Equal(events, []string{"workflow_dispatch", "workflow_run"}) {
		add("runs on %v, want [workflow_dispatch workflow_run]", events)
	}
	run := w.events().WorkflowRun
	if !slices.Equal(slices.Sorted(slices.Values(run.Workflows)), []string{"rebuild", "update"}) {
		add("workflow_run workflows = %v, want [update rebuild]: the workflows that build the pack", run.Workflows)
	}
	if !slices.Equal(run.Types, []string{"completed"}) {
		add("workflow_run types = %v, want [completed]", run.Types)
	}
	if len(w.Jobs) != 1 {
		add("has %d jobs, want one", len(w.Jobs))
		return problems
	}
	job := w.Jobs[w.jobNames()[0]]
	if job.If != rulesProposalIf {
		add("the if of the job is %q, want %q: a run of update or rebuild that did not succeed, or not of main, has no build to learn from", job.If, rulesProposalIf)
	}
	if job.TimeoutMinutes != 60 {
		add("timeout-minutes = %d, want 60", job.TimeoutMinutes)
	}
	if job.Concurrency.Group != "rules-proposal" || job.Concurrency.CancelInProgress {
		add("concurrency = {group: %q, cancel-in-progress: %v}, want {group: \"rules-proposal\", cancel-in-progress: false}",
			job.Concurrency.Group, job.Concurrency.CancelInProgress)
	}
	if !maps.Equal(job.Permissions, rulesProposalPermissions) {
		add("permissions = %v, want %v", job.Permissions, rulesProposalPermissions)
	}
	deep := slices.ContainsFunc(job.Steps, func(s workflowStep) bool {
		return strings.HasPrefix(s.Uses, "actions/checkout@") && fmt.Sprint(s.With["fetch-depth"]) == "0"
	})
	if !deep {
		add("no actions/checkout step with fetch-depth: 0: origin/main, the base of the rules PR, and its history are not there")
	}
	for _, s := range job.Steps {
		if strings.HasPrefix(s.Uses, "actions/checkout@") && fmt.Sprint(s.With["persist-credentials"]) != "false" {
			add("an actions/checkout step has no persist-credentials: false: the checkout leaves its token in .git/config, where the action, which reads text anyone can write, can find it")
		}
	}

	runs := w.runs()
	for _, m := range rulesProposalMarkers {
		if !strings.Contains(runs, m.text) {
			add("no %q: %s", m.text, m.why)
		}
	}
	for _, o := range rulesProposalOrder {
		first, second := strings.Index(runs, o[0]), strings.Index(runs, o[1])
		if first >= 0 && second >= 0 && first > second {
			add("%q must come before %q", o[0], o[1])
		}
	}

	// The steps, by what they do.
	at := func(match func(workflowStep) bool) int { return slices.IndexFunc(job.Steps, match) }
	running := func(sub string) int { return at(func(s workflowStep) bool { return strings.Contains(s.Run, sub) }) }
	start, count, confirm := running("bin/lct proposal start"), running("bin/lct suggest"), running("gh pr list")
	var actions []int
	for i, s := range job.Steps {
		if strings.HasPrefix(s.Uses, "anthropics/claude-code-action@") {
			actions = append(actions, i)
		}
	}
	if len(actions) != 1 {
		add("%d steps use anthropics/claude-code-action, want one", len(actions))
		return problems
	}
	act := actions[0]
	if start >= 0 && count >= 0 && confirm >= 0 && !(start < count && count < act && act < confirm) {
		add("the steps must run in this order: proposal start, suggest, the action, the look for the PR")
	}
	step := job.Steps[act]

	// The action: the skill, the key, the model, the turns and the tools.
	if step.With["prompt"] != rulesProposalPrompt {
		add("the prompt of the action is %q, want %q: it runs the skill and nothing else", step.With["prompt"], rulesProposalPrompt)
	}
	if step.With["anthropic_api_key"] != "${{ secrets.ANTHROPIC_API_KEY }}" {
		add("the action has anthropic_api_key = %v, want ${{ secrets.ANTHROPIC_API_KEY }}", step.With["anthropic_api_key"])
	}
	args, _ := step.With["claude_args"].(string)
	if !strings.Contains(args, "--model ${{ vars.LCT_MODEL }}") {
		add("claude_args has no --model ${{ vars.LCT_MODEL }}: the model comes from the variable and is not written in the file")
	}
	if !claudeMaxTurnsRe.MatchString(args) {
		add("claude_args has no --max-turns 80: the run is not limited in turns")
	}
	if strings.Contains(args, "--dangerously-skip-permissions") {
		add("claude_args has --dangerously-skip-permissions, which makes the list of allowed tools mean nothing")
	}
	if m := claudeDisallowedToolsRe.FindAllStringSubmatch(args, -1); len(m) != 1 || !slices.Contains(strings.Split(m[0][1], ","), rulesProposalDenied) {
		add("claude_args has no --disallowedTools \"%s\": the folder of git, with the token of the checkout, is not closed to the action", rulesProposalDenied)
	}
	if m := claudeAllowedToolsRe.FindAllStringSubmatch(args, -1); len(m) != 1 {
		add("claude_args has %d --allowedTools \"...\" lists, want one: the tools of the rules-review skill, separated by commas", len(m))
	} else {
		var tools []string
		for _, tool := range strings.Split(m[0][1], ",") {
			tools = append(tools, strings.TrimSpace(tool))
		}
		for _, tool := range skillTools {
			if !slices.Contains(tools, tool) {
				add("claude_args --allowedTools lacks %q, which the allowed-tools of the rules-review skill has: the skill would ask for a permission nobody answers", tool)
			}
		}
		for i, tool := range tools {
			switch {
			case !slices.Contains(skillTools, tool):
				add("claude_args --allowedTools has %q, which the allowed-tools of the rules-review skill lacks", tool)
			case slices.Index(tools, tool) != i:
				add("claude_args --allowedTools has %q twice", tool)
			}
		}
	}
	for _, name := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if _, ok := step.Env[name]; ok {
			add("the action step has %s in its env: the action brings the token of the Claude GitHub App, and a token of the workflow would open the PR as github-actions, whose pull requests start no workflow", name)
		}
	}
	if !strings.Contains(step.If, rulesProposalCount) {
		add("the if of the action step is %q, which lacks %s: it would run with nothing to suggest", step.If, rulesProposalCount)
	}
	if !strings.Contains(step.If, rulesProposalHasModel) {
		add("the if of the action step is %q, which lacks %s: it would run without a model", step.If, rulesProposalHasModel)
	}
	if !strings.Contains(step.If, rulesProposalInCI) {
		add("the if of the action step is %q, which lacks %s: it would review the rules in CI when a person reviews them on their machine", step.If, rulesProposalInCI)
	}

	// When the rules are reviewed on a person's machine a step says so and
	// stands in for the skill, as for a missing model.
	notInCI := at(func(s workflowStep) bool { return strings.Contains(s.If, rulesProposalNotInCI) })
	switch {
	case notInCI < 0:
		add("no step runs if %s: a run that leaves the review to a person would end red for the missing outcome", rulesProposalNotInCI)
	default:
		s := job.Steps[notInCI]
		if !strings.Contains(s.If, rulesProposalCount) || !strings.Contains(s.If, rulesProposalHasModel) {
			add("the if of the step for a review on a person's machine is %q, which lacks %s or %s", s.If, rulesProposalCount, rulesProposalHasModel)
		}
		if !strings.Contains(s.Run, "::notice::") || !strings.Contains(s.Run, "/rules-review") {
			add("the step for a review on a person's machine does not say that the suggestions wait for /rules-review: %q", firstLine(s.Run))
		}
		if !strings.Contains(s.Run, `"nothing: `) || !strings.Contains(s.Run, "> work/proposal/outcome.txt") || strings.Contains(s.Run, "nothing: 本機") {
			add("the step for a review on a person's machine does not write an outcome of nothing that the check accepts (not nothing: 本機…) to work/proposal/outcome.txt")
		}
		if confirm >= 0 && notInCI > confirm {
			add("the step for a review on a person's machine comes after the step that checks the outcome")
		}
	}

	// Without the model a step says so and stands in for the skill: it writes
	// an outcome of nothing, so that the check of the outcome stays green.
	noModel := at(func(s workflowStep) bool { return strings.Contains(s.If, rulesProposalNoModel) })
	switch {
	case noModel < 0:
		add("no step runs if %s: a run without LCT_MODEL would end without a word, or red for the missing outcome", rulesProposalNoModel)
	default:
		s := job.Steps[noModel]
		if !strings.Contains(s.If, rulesProposalCount) {
			add("the if of the step for a missing LCT_MODEL is %q, which lacks %s", s.If, rulesProposalCount)
		}
		if !strings.Contains(s.Run, "::warning::") || !strings.Contains(s.Run, "LCT_MODEL") {
			add("the step for a missing LCT_MODEL does not warn that LCT_MODEL is not set: %q", firstLine(s.Run))
		}
		if !strings.Contains(s.Run, "'nothing: ") || !strings.Contains(s.Run, "> work/proposal/outcome.txt") {
			add("the step for a missing LCT_MODEL does not write an outcome of nothing (echo 'nothing: …' > work/proposal/outcome.txt), so the check of the outcome fails")
		}
		if confirm >= 0 && noModel > confirm {
			add("the step for a missing LCT_MODEL comes after the step that checks the outcome")
		}
	}

	// The other steps.
	if start >= 0 && job.Steps[start].Env["GH_TOKEN"] != "${{ github.token }}" {
		add("the step that runs bin/lct proposal start has no GH_TOKEN: ${{ github.token }} in its env: it asks gh for the pull requests")
	}
	if count >= 0 && job.Steps[count].ID != "suggest" {
		add("the step that runs bin/lct suggest has no id: suggest, which the if of the steps after it reads")
	}
	if confirm >= 0 {
		s := job.Steps[confirm]
		if s.Env["GH_TOKEN"] != "${{ github.token }}" {
			add("the step that looks for the rules PR has no GH_TOKEN: ${{ github.token }} in its env")
		}
		if s.Env["BRANCH"] != "${{ steps.start.outputs.branch }}" {
			add("the step that looks for the rules PR has BRANCH = %q in its env, want ${{ steps.start.outputs.branch }}", s.Env["BRANCH"])
		}
		if !strings.Contains(s.If, rulesProposalCount) {
			add("the if of the step that looks for the rules PR is %q, which lacks %s: with nothing to suggest no PR is due", s.If, rulesProposalCount)
		}
	}
	for i, s := range job.Steps {
		if i != act && strings.Contains(fmt.Sprint(s.Env), "secrets.") {
			add("a step other than the action has a secret in its env: %q", firstLine(s.Run))
		}
		if i != act && strings.Contains(fmt.Sprint(s.With), "secrets.") {
			add("a step other than the action has a secret in its with: %q", cmp.Or(s.Uses, firstLine(s.Run)))
		}
	}
	return append(problems, checkJQKeys(file, content, rulesProposalJSONInputs(), rulesProposalJSONRequired)...)
}

// continues reports whether a continue-on-error value lets the run go on after
// the job or the step failed: anything but false, an expression included.
func continues(v any) bool { return v != nil && fmt.Sprint(v) != "false" }

// stepLabel names a step in a message.
func stepLabel(s workflowStep) string { return cmp.Or(s.Uses, firstLine(s.Run)) }

const (
	// siteDir is the folder of the viewer; its npm steps run there.
	siteDir = "site"
	// siteNodeAction is the setup-node of ci.yml and pages.yml.
	siteNodeAction = "actions/setup-node@v7"
)

// siteNodeWith is the with of the setup-node step of ci.yml and pages.yml: the
// Node version of .nvmrc and the cache of npm, keyed by the lock file. No
// other key: a node-version would take the place of the .nvmrc.
var siteNodeWith = map[string]string{
	"node-version-file": "site/.nvmrc", "cache": "npm", "cache-dependency-path": "site/package-lock.json",
}

// checkSetupNode reports what is wrong with the setup-node step s.
func checkSetupNode(s workflowStep) []string {
	var problems []string
	if s.Uses != siteNodeAction {
		problems = append(problems, fmt.Sprintf("the setup-node step uses %q, want %s", s.Uses, siteNodeAction))
	}
	for _, key := range slices.Sorted(maps.Keys(siteNodeWith)) {
		if got, ok := s.With[key]; !ok || fmt.Sprint(got) != siteNodeWith[key] {
			problems = append(problems, fmt.Sprintf("the setup-node step has %s = %v, want %s", key, s.With[key], siteNodeWith[key]))
		}
	}
	for _, key := range slices.Sorted(maps.Keys(s.With)) {
		if _, ok := siteNodeWith[key]; !ok {
			problems = append(problems, fmt.Sprintf("the setup-node step has %s, which is not one of %s: the Node version is that of site/.nvmrc", key, strings.Join(slices.Sorted(maps.Keys(siteNodeWith)), ", ")))
		}
	}
	return problems
}

// npmStep returns the index in steps of the step whose script is exactly the
// npm command run, or -1.
func npmStep(steps []workflowStep, run string) int {
	return slices.IndexFunc(steps, func(s workflowStep) bool { return strings.TrimSpace(s.Run) == run })
}

// checkNpmStep reports the step that runs the npm command run when it is not
// in the folder of the viewer.
func checkNpmStep(steps []workflowStep, i int, run string) []string {
	if i >= 0 && steps[i].WorkingDirectory != siteDir {
		return []string{fmt.Sprintf("the step that runs %s has working-directory = %q, want %s: package.json is there", run, steps[i].WorkingDirectory, siteDir)}
	}
	return nil
}

// checkKeepsGoing reports the job and the steps of job that let the run go on
// after a step failed: a continue-on-error, a step with an if (the if of the
// job alone says when it runs), a shell that does not stop at the first
// failure (only bash, whose -e the runner adds, is allowed).
func checkKeepsGoing(job workflowJob) []string {
	var problems []string
	if continues(job.ContinueOnError) {
		problems = append(problems, "the job has continue-on-error: its failure would not fail the run")
	}
	if job.Defaults.Run.Shell != "" && job.Defaults.Run.Shell != "bash" {
		problems = append(problems, fmt.Sprintf("the job runs scripts with the shell %q, want bash, which stops at the first command that fails", job.Defaults.Run.Shell))
	}
	for _, s := range job.Steps {
		if continues(s.ContinueOnError) {
			problems = append(problems, fmt.Sprintf("the step %q has continue-on-error: a step that fails would not fail the run", stepLabel(s)))
		}
		if s.If != "" {
			problems = append(problems, fmt.Sprintf("the step %q has an if (%s): a step that is skipped, or one that runs after a failure, lets the run go on with what is missing", stepLabel(s), s.If))
		}
		if s.Shell != "" && s.Shell != "bash" {
			problems = append(problems, fmt.Sprintf("the step %q runs with the shell %q, want bash, which stops at the first command that fails", stepLabel(s), s.Shell))
		}
	}
	return problems
}

// pagesPermissions are the permissions of the job of pages.yml: it reads the
// repo, publishes the site to Pages (pages) and proves to the deployment who
// it is with its OIDC token (id-token).
var pagesPermissions = map[string]string{"contents": "read", "pages": "write", "id-token": "write"}

// pagesActions are the only actions pages.yml uses, pinned to the major
// version. None of them is the action of Anthropic: the site is made without a
// model.
var pagesActions = []string{
	"actions/checkout@v5", "actions/setup-go@v6", siteNodeAction,
	"actions/configure-pages@v6", "actions/upload-pages-artifact@v5", "actions/deploy-pages@v5",
}

const (
	// pagesEnvironment is the environment the Pages deployment needs.
	pagesEnvironment = "github-pages"
	// pagesSiteCommand is the whole script of the step that writes the data of
	// the site. Anything around it (|| true, a pipe, a second command) can hide
	// that lct site failed, which exits 1 for a refusal and 2 for a system error.
	pagesSiteCommand = "bin/lct site --out site/public/data --json"
	// pagesArtifactPath is what the build makes and the artifact holds.
	pagesArtifactPath = "site/dist"
)

// pagesSteps are the steps pages.yml must have, in the order of pagesOrder: by
// key, what each does and how to find it.
var pagesSteps = []struct {
	key, what string
	match     func(workflowStep) bool
}{
	{"checkout", "checks out main", func(s workflowStep) bool { return strings.HasPrefix(s.Uses, "actions/checkout@") }},
	{"fetch", "fetches raw and tw and makes their worktrees", func(s workflowStep) bool { return strings.Contains(s.Run, "git fetch origin") }},
	{"setup-go", "sets up Go", func(s workflowStep) bool { return strings.HasPrefix(s.Uses, "actions/setup-go@") }},
	{"build", "builds lct", func(s workflowStep) bool { return strings.TrimSpace(s.Run) == "go build -o bin/lct ./cmd/lct" }},
	{"site", "runs bin/lct site", func(s workflowStep) bool { return strings.Contains(s.Run, "bin/lct site") }},
	{"setup-node", "sets up Node", func(s workflowStep) bool { return strings.HasPrefix(s.Uses, "actions/setup-node@") }},
	{"npm-ci", "runs npm ci", func(s workflowStep) bool { return strings.TrimSpace(s.Run) == "npm ci" }},
	{"npm-build", "runs npm run build", func(s workflowStep) bool { return strings.TrimSpace(s.Run) == "npm run build" }},
	{"configure", "configures Pages", func(s workflowStep) bool { return strings.HasPrefix(s.Uses, "actions/configure-pages@") }},
	{"upload", "uploads the artifact", func(s workflowStep) bool { return strings.HasPrefix(s.Uses, "actions/upload-pages-artifact@") }},
	{"deploy", "deploys the artifact", func(s workflowStep) bool { return strings.HasPrefix(s.Uses, "actions/deploy-pages@") }},
}

// pagesOrder pairs the steps of pagesSteps that must run in that order: the
// first before the second.
var pagesOrder = []struct{ first, second, why string }{
	{"checkout", "fetch", "there is no repo to fetch into"},
	{"fetch", "site", "lct site reads the raw and tw worktrees"},
	{"setup-go", "build", "the Go of go.mod is not there to build lct"},
	{"build", "site", "lct is not built to write the data"},
	{"site", "npm-build", "the build copies site/public into site/dist, and the data is not there yet: the site would be deployed without it"},
	{"setup-node", "npm-ci", "the Node of site/.nvmrc is not there to install the packages"},
	{"npm-ci", "npm-build", "the packages are not installed to build the site"},
	{"npm-build", "upload", "the artifact would hold a site/dist the build did not make"},
	{"configure", "upload", "Pages is not configured before the artifact is made"},
	{"upload", "deploy", "there is no artifact to deploy"},
}

// pagesFetchMarkers are what the step of pages.yml that fetches must do, with
// the reason.
var pagesFetchMarkers = []struct{ text, why string }{
	{"+refs/heads/raw:refs/remotes/origin/raw", "the raw branch is not fetched"},
	{"+refs/heads/tw:refs/remotes/origin/tw", "the tw branch is not fetched"},
	{"git worktree add .work/raw raw", "lct site reads the raw worktree and it is not made"},
	{"git worktree add .work/tw tw", "lct site reads the tw worktree and it is not made"},
}

// checkPagesWorkflow checks pages.yml: it runs when update or rebuild finished
// green on main, and by hand; one job at a time (a deployment that runs is
// never cancelled), with the permissions of a Pages deployment and no more, in
// the environment github-pages, and with no secret. Its steps check out main
// with the whole history and the tags (the Release the site names is a tag of
// tw), fetch raw and tw and make their worktrees, build lct, write the data of
// the site, set up Node as ci.yml does, install the packages and build the
// site, configure Pages, upload site/dist and deploy it, in that order. Nothing
// lets the deploy go on when a step failed: lct site, which exits non-zero on
// a refusal or a system error, is the whole script of its step, and no step
// has an if or a continue-on-error.
func checkPagesWorkflow(file, content string) []string {
	w, problems := parseWorkflow(file, content)
	if w == nil {
		return problems
	}
	add := func(format string, args ...any) { problems = append(problems, file+": "+fmt.Sprintf(format, args...)) }

	problems = append(problems, checkEvents(file, w, "workflow_dispatch", "workflow_run")...)
	ev := w.events()
	problems = append(problems, checkDispatchInputs(file, ev, map[string]string{})...)
	if !slices.Equal(slices.Sorted(slices.Values(ev.WorkflowRun.Workflows)), []string{"rebuild", "update"}) {
		add("workflow_run workflows = %v, want [update rebuild]: the workflows that build the pack", ev.WorkflowRun.Workflows)
	}
	if !slices.Equal(ev.WorkflowRun.Types, []string{"completed"}) {
		add("workflow_run types = %v, want [completed]", ev.WorkflowRun.Types)
	}
	if strings.Contains(content, "secrets.") {
		add("refers to a secret: the site is made from the branches of the repo and needs none")
	}
	if len(w.Jobs) != 1 {
		add("has %d jobs, want one", len(w.Jobs))
		return problems
	}
	job := w.Jobs[w.jobNames()[0]]
	if job.If != rulesProposalIf {
		add("the if of the job is %q, want %q: a run of update or rebuild that did not succeed, or not of main, has no build to show", job.If, rulesProposalIf)
	}
	if job.TimeoutMinutes != 30 {
		add("timeout-minutes = %d, want 30", job.TimeoutMinutes)
	}
	if job.Concurrency.Group != "pages" || job.Concurrency.CancelInProgress {
		add("concurrency = {group: %q, cancel-in-progress: %v}, want {group: \"pages\", cancel-in-progress: false}: a deployment that runs is not cancelled",
			job.Concurrency.Group, job.Concurrency.CancelInProgress)
	}
	if !maps.Equal(job.Permissions, pagesPermissions) {
		add("permissions = %v, want %v", job.Permissions, pagesPermissions)
	}
	switch env := job.Environment.(type) {
	case string:
		if env != pagesEnvironment {
			add("environment = %q, want %s", env, pagesEnvironment)
		}
	case map[string]any:
		if env["name"] != pagesEnvironment {
			add("environment name = %v, want %s", env["name"], pagesEnvironment)
		}
	default:
		add("environment = %v, want %s: the deployment to Pages needs it", job.Environment, pagesEnvironment)
	}
	for _, problem := range checkKeepsGoing(job) {
		add("%s", problem)
	}

	// The actions are those of pagesActions, and the checkout has the whole
	// history and every tag, and keeps no token.
	for _, s := range job.Steps {
		if s.Uses != "" && !slices.Contains(pagesActions, s.Uses) {
			add("uses %q, which is not one of %s", s.Uses, strings.Join(pagesActions, ", "))
		}
		if strings.HasPrefix(s.Uses, "actions/checkout@") {
			if fmt.Sprint(s.With["fetch-depth"]) != "0" {
				add("an actions/checkout step has no fetch-depth: 0: the tags, and with them the Release of tw that the site names, are not there")
			}
			if fmt.Sprint(s.With["persist-credentials"]) != "false" {
				add("an actions/checkout step has no persist-credentials: false: the checkout leaves its token in .git/config, where the install scripts of the packages of the site can read it")
			}
		}
	}

	// The steps and their order.
	index := map[string]int{}
	what := map[string]string{}
	for _, kind := range pagesSteps {
		what[kind.key] = kind.what
		index[kind.key] = slices.IndexFunc(job.Steps, kind.match)
		if index[kind.key] < 0 {
			add("no step that %s", kind.what)
		}
	}
	for _, o := range pagesOrder {
		if first, second := index[o.first], index[o.second]; first >= 0 && second >= 0 && first > second {
			add("the step that %s must come before the step that %s: %s", what[o.first], what[o.second], o.why)
		}
	}
	if i := index["fetch"]; i >= 0 {
		for _, m := range pagesFetchMarkers {
			if !strings.Contains(job.Steps[i].Run, m.text) {
				add("the step that fetches has no %q: %s", m.text, m.why)
			}
		}
		if fetch, worktree := strings.Index(job.Steps[i].Run, "git fetch origin"), strings.Index(job.Steps[i].Run, "git worktree add"); fetch > worktree && worktree >= 0 {
			add("the step that fetches adds a worktree before it fetches the branch")
		}
	}

	// What the steps do.
	for _, s := range job.Steps {
		if got := strings.TrimSpace(s.Run); strings.Contains(got, "bin/lct site") && got != pagesSiteCommand {
			add("a step that runs bin/lct site runs %q, want exactly %q: anything around it (|| true, a pipe, another command) can hide that lct site failed, and the deploy would go on with missing or stale data", got, pagesSiteCommand)
		}
	}
	if i := index["setup-node"]; i >= 0 {
		for _, problem := range checkSetupNode(job.Steps[i]) {
			add("%s", problem)
		}
	}
	for _, run := range []string{"npm ci", "npm run build"} {
		for _, problem := range checkNpmStep(job.Steps, npmStep(job.Steps, run), run) {
			add("%s", problem)
		}
	}
	if i := index["upload"]; i >= 0 {
		if got := job.Steps[i].With["path"]; got != pagesArtifactPath {
			add("the upload has path = %v, want %s: the artifact is the site that the build made", got, pagesArtifactPath)
		}
	}
	return problems
}

// ciSiteRuns are the npm commands the site job of ci.yml runs, in this order.
var ciSiteRuns = []string{"npm ci", "npm run typecheck", "npm test", "npm run build"}

// checkCISiteJob checks the job site of ci.yml: it checks out the repo, sets up
// Node as pages.yml does, and runs, in the folder site, npm ci, the type check,
// the tests and the build, in that order (the build here is the test that the
// site builds before pages.yml has to). Nothing lets a failure pass.
func checkCISiteJob(file string, w *workflowFile) []string {
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, file+": "+fmt.Sprintf(format, args...)) }

	job, ok := w.Jobs["site"]
	if !ok {
		add("has no job site: the tests and the build of the viewer do not run on a push or a pull request")
		return problems
	}
	if job.If != "" {
		add("the job site has an if (%s): it would be skipped, which is no failure", job.If)
	}
	for _, problem := range checkKeepsGoing(job) {
		add("%s", problem)
	}
	if !slices.ContainsFunc(job.Steps, func(s workflowStep) bool { return strings.HasPrefix(s.Uses, "actions/checkout@") }) {
		add("the job site has no actions/checkout step")
	}
	node := slices.IndexFunc(job.Steps, func(s workflowStep) bool { return strings.HasPrefix(s.Uses, "actions/setup-node@") })
	if node < 0 {
		add("the job site has no actions/setup-node step: the Node of site/.nvmrc is not set up")
	} else {
		for _, problem := range checkSetupNode(job.Steps[node]) {
			add("%s", problem)
		}
	}
	last := node
	for _, run := range ciSiteRuns {
		i := npmStep(job.Steps, run)
		if i < 0 {
			add("the job site has no step that runs %s", run)
			continue
		}
		for _, problem := range checkNpmStep(job.Steps, i, run) {
			add("%s", problem)
		}
		if i < last {
			add("the steps of the job site must run in this order: the setup of Node, then %s", strings.Join(ciSiteRuns, ", "))
		}
		last = max(last, i)
	}
	return problems
}

// readmeSection returns the text of the section of readme under the heading
// "## <heading>", up to the next heading of that level.
func readmeSection(readme, heading string) (string, bool) {
	lines := strings.Split(strings.ReplaceAll(readme, "\r\n", "\n"), "\n")
	start := slices.Index(lines, "## "+heading)
	if start < 0 {
		return "", false
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	return strings.Join(lines[start+1:end], "\n"), true
}

// checkRefsDocumented reports the secrets and variables the workflows (file
// name to text) use that no table row of the README section "GitHub Actions
// 設定" lists by name.
func checkRefsDocumented(files map[string]string, readme string) []string {
	section, ok := readmeSection(readme, readmeActions)
	if !ok {
		return []string{fmt.Sprintf("README.md: no section %q", "## "+readmeActions)}
	}
	var rows []string
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "|") {
			rows = append(rows, line)
		}
	}
	var problems []string
	seen := map[string]bool{}
	for _, name := range slices.Sorted(maps.Keys(files)) {
		for _, m := range workflowRefRe.FindAllStringSubmatch(files[name], -1) {
			ref := m[1] + "." + m[2]
			if seen[ref] {
				continue
			}
			seen[ref] = true
			if !slices.ContainsFunc(rows, func(row string) bool { return strings.Contains(row, "`"+m[2]+"`") }) {
				problems = append(problems, fmt.Sprintf("%s: uses %s, which no table row of the section %q in README.md lists as `%s`", workflowPath(name), ref, readmeActions, m[2]))
			}
		}
	}
	return problems
}

// jqCall is a call of jq in a script: its flags, its filter and what it reads,
// a variable name (from <<<"$name") or a file.
type jqCall struct{ flags, filter, input string }

// jqCalls returns the calls of jq in the scripts text.
func jqCalls(runs string) []jqCall {
	var calls []jqCall
	for _, m := range jqCallRe.FindAllStringSubmatch(runs, -1) {
		input := m[3]
		if input == "" {
			input = m[4]
		}
		calls = append(calls, jqCall{flags: m[1], filter: strings.Trim(m[2], `'"`), input: input})
	}
	return calls
}

// jsonFields maps the JSON keys of the struct v (or of what it points to) to
// the kind of the field behind each.
func jsonFields(v any) map[string]reflect.Kind {
	t := reflect.TypeOf(v)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	fields := map[string]reflect.Kind{}
	for i := 0; i < t.NumField(); i++ {
		name, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ",")
		if name == "" || name == "-" {
			continue
		}
		ft := t.Field(i).Type
		for ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		fields[name] = ft.Kind()
	}
	return fields
}

// lctJSONForbidden are the keys of an lct output that pipeline.yml must not
// read, with the reason.
var lctJSONForbidden = map[string]map[string]string{
	// Check.Tag is the Release the HEAD of tw has, not the tag of the next
	// Release, which only a build numbers (out/publish.json).
	"release_check": {"tag": "the tag of the check is the Release HEAD has, not the next one: the next tag is the build's (out/publish.json)"},
}

// checkJQForbidden reports the keys of forbidden that a jq of the workflow
// reads from their input.
func checkJQForbidden(file, content string, forbidden map[string]map[string]string) []string {
	w, problems := parseWorkflow(file, content)
	if w == nil {
		return problems
	}
	for _, call := range jqCalls(w.runs()) {
		for _, m := range jqKeyRe.FindAllStringSubmatch(call.filter, -1) {
			if why, ok := forbidden[call.input][m[1]]; ok {
				problems = append(problems, fmt.Sprintf("%s: reads %q from %s: %s", file, m[1], call.input, why))
			}
		}
	}
	return problems
}

// lctJSONInputs are what pipeline.yml feeds to jq: each is the output of an
// lct command, kept in a variable or in a file, and the value that command
// prints (its JSON keys are the ones jq may read). nil is output of another
// tool, which is not checked.
func lctJSONInputs() map[string]any {
	return map[string]any{
		"check":                      fetchResult{},
		"snapshot":                   snapshotResult{},
		"plan":                       plan.Summary{},
		"count":                      pendingCount{},
		"build":                      buildOutput{},
		"out/publish.json":           publish.Result{},
		"release_check":              publish.Check{},
		".work/raw/meta/source.json": raw.Source{},
		".work/tw/.meta/build.json":  build.Info{},
		"issues":                     nil, // gh issue list
	}
}

// lctJSONRequired are the keys pipeline.yml must read from each output, so
// that the check cannot pass by finding nothing.
var lctJSONRequired = map[string][]string{
	"check":                      {"new", "token", "created_at"},
	"snapshot":                   {"tag"},
	"plan":                       {"pending", "requeue_applied", "rewrite_applied"},
	"count":                      {"available"},
	"build":                      {"review_issue_parts"},
	"out/publish.json":           {"tag", "tw_commit", "assets", "release_needed", "same_as"},
	"release_check":              {"release_needed"},
	".work/raw/meta/source.json": {"tag"},
	".work/tw/.meta/build.json":  {"raw_tag"},
}

// checkJQKeys reports what the jq commands of the workflow read that the lct
// output has no JSON key for, the keys in required that it does not read at
// all, a jq -e on a boolean (which fails on false: ask for `| tostring`), and
// keys read from an input the contract does not know. inputs maps what jq
// reads to the value that prints it.
func checkJQKeys(file, content string, inputs map[string]any, required map[string][]string) []string {
	w, problems := parseWorkflow(file, content)
	if w == nil {
		return problems
	}
	read := map[string]map[string]bool{}
	for _, call := range jqCalls(w.runs()) {
		var keys []string
		for _, m := range jqKeyRe.FindAllStringSubmatch(call.filter, -1) {
			keys = append(keys, m[1])
		}
		printer, known := inputs[call.input]
		switch {
		case !known:
			if len(keys) > 0 {
				problems = append(problems, fmt.Sprintf("%s: jq reads %v from %q, whose JSON the contract does not know", file, keys, call.input))
			}
			continue
		case printer == nil:
			continue
		}
		fields := jsonFields(printer)
		for _, key := range keys {
			kind, printed := fields[key]
			if read[call.input] == nil {
				read[call.input] = map[string]bool{}
			}
			read[call.input][key] = true
			switch {
			case !printed:
				problems = append(problems, fmt.Sprintf("%s: reads %q from %s, which %T does not print", file, key, call.input, printer))
			case kind == reflect.Bool && jqExitStatusRe.MatchString(call.flags+" ") && !strings.Contains(call.filter, "tostring"):
				problems = append(problems, fmt.Sprintf("%s: jq -e on the boolean %q fails when it is false; read '.%s | tostring'", file, key, key))
			}
		}
	}
	for _, input := range slices.Sorted(maps.Keys(required)) {
		for _, key := range required[input] {
			if !read[input][key] {
				problems = append(problems, fmt.Sprintf("%s: does not read %q from %s", file, key, input))
			}
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

// pipelineStepWith is the text of the one step of pipeline.yml whose text
// holds sub, from its "- name:" line to the next step.
func pipelineStepWith(t *testing.T, src, sub string) string {
	t.Helper()
	const head = "\n      - "
	var found []string
	parts := strings.Split(src, head)
	for _, part := range parts[1:] {
		if strings.Contains(part, sub) {
			found = append(found, head+part)
		}
	}
	if len(found) != 1 {
		t.Fatalf("pipeline.yml has %d steps with %q, want 1: update the test", len(found), sub)
	}
	return found[0]
}

// mutateAll replaces every copy of old, of which there must be at least one.
func mutateAll(t *testing.T, src, old, repl string) string {
	t.Helper()
	if !strings.Contains(src, old) {
		t.Fatalf("the workflow has no copy of %q: update the test", old)
	}
	return strings.ReplaceAll(src, old, repl)
}

// readWorkflows reads every workflow of the repo: file name to text.
func readWorkflows(t *testing.T) map[string]string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(repoRoot, filepath.FromSlash(workflowDir), "*.yml"))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, p := range paths {
		_, content := readRepoFile(t, p)
		files[filepath.Base(p)] = content
	}
	return files
}

// wantProblems fails the test unless problems is exactly one problem that
// holds all of parts.
func wantProblems(t *testing.T, problems []string, parts ...string) {
	t.Helper()
	if len(problems) != 1 {
		t.Fatalf("got %d problems, want one holding %q:\n%s", len(problems), parts, strings.Join(problems, "\n"))
	}
	for _, part := range parts {
		if !strings.Contains(problems[0], part) {
			t.Errorf("problem %q does not contain %q", problems[0], part)
		}
	}
}

// hasProblem fails the test unless one of problems holds all of parts.
func hasProblem(t *testing.T, problems []string, parts ...string) {
	t.Helper()
	if !slices.ContainsFunc(problems, func(p string) bool {
		return !slices.ContainsFunc(parts, func(part string) bool { return !strings.Contains(p, part) })
	}) {
		t.Errorf("problems = %q, want one holding %q", problems, parts)
	}
}

func TestWorkflowContract(t *testing.T) {
	commands := cli.Commands()
	files := readWorkflows(t)
	names := slices.Sorted(maps.Keys(files))
	_, readme := readRepoFile(t, filepath.Join(repoRoot, "README.md"))
	pipeline, update, rebuild, ci := files["pipeline.yml"], files["update.yml"], files["rebuild.yml"], files["ci.yml"]
	pipelineFile, updateFile, rebuildFile, ciFile := workflowPath("pipeline.yml"), workflowPath("update.yml"), workflowPath("rebuild.yml"), workflowPath("ci.yml")
	rules, rulesFile := files["rules-proposal.yml"], workflowPath("rules-proposal.yml")
	pages, pagesFile := files["pages.yml"], workflowPath("pages.yml")
	skillTools := skillAllowedTools(t, "rules-review")

	// The workflows of workflowNames are under the contract; every workflow in
	// the directory is checked.
	for _, problem := range checkWorkflowSet(names) {
		t.Error(problem)
	}
	for _, name := range names {
		for _, problem := range checkWorkflow(workflowPath(name), files[name], commands) {
			t.Error(problem)
		}
	}
	for _, problem := range checkModelNames(repoRoot, workflowDir) {
		t.Error(problem)
	}
	for _, problem := range checkPipelineWorkflow(pipelineFile, pipeline) {
		t.Error(problem)
	}
	for _, problem := range checkCIWorkflow(ciFile, ci) {
		t.Error(problem)
	}
	for _, problem := range checkUpdateWorkflow(updateFile, update, pipeline) {
		t.Error(problem)
	}
	for _, problem := range checkRebuildWorkflow(rebuildFile, rebuild, pipeline) {
		t.Error(problem)
	}
	for _, problem := range checkRulesProposalWorkflow(rulesFile, rules, skillTools) {
		t.Error(problem)
	}
	for _, problem := range checkPagesWorkflow(pagesFile, pages) {
		t.Error(problem)
	}
	for _, problem := range checkWorkflowRunNames(files) {
		t.Error(problem)
	}
	for _, problem := range checkRefsDocumented(files, readme) {
		t.Error(problem)
	}
	for _, problem := range checkJQKeys(pipelineFile, pipeline, lctJSONInputs(), lctJSONRequired) {
		t.Error(problem)
	}
	if t.Failed() {
		return
	}

	// 1. The workflows exist.
	t.Run("a workflow that is missing", func(t *testing.T) {
		for _, gone := range workflowNames {
			have := slices.DeleteFunc(slices.Clone(names), func(n string) bool { return n == gone })
			wantProblems(t, checkWorkflowSet(have), workflowPath(gone))
		}
	})

	// 2. Every bin/lct a workflow runs is a registered command.
	t.Run("an unregistered lct command", func(t *testing.T) {
		src := mutate(t, pipeline, "bin/lct publish --json", "bin/lct publsh --json")
		wantProblems(t, checkWorkflow(pipelineFile, src, commands), `"bin/lct publsh"`)
	})
	t.Run("a file that is not YAML", func(t *testing.T) {
		wantProblems(t, checkWorkflow(ciFile, ci+"\n  - [unclosed\n", commands), "not valid YAML")
	})

	// 3. No model name.
	t.Run("a model name", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, root, workflowDir+"/pipeline.yml", pipeline+"# claude-example-9\n")
		wantProblems(t, checkModelNames(root, workflowDir), "pipeline.yml", `"claude-example-9"`)
	})

	// 4. Secrets: the two of the repo, read by the step that needs them, and
	// never written out.
	t.Run("a secret that is not one of the two", func(t *testing.T) {
		src := mutate(t, pipeline, "LCT_MODEL: ${{ vars.LCT_MODEL }}", "LCT_MODEL: ${{ secrets.RELEASE_TOKEN }}")
		wantProblems(t, checkWorkflow(pipelineFile, src, commands), "secrets.RELEASE_TOKEN")
	})
	t.Run("the status API URL named in a script", func(t *testing.T) {
		src := mutate(t, pipeline, "bin/lct fetch --check --json", `bin/lct fetch --check --status-url "$LIMBUS_STATUS_URL" --json`)
		wantProblems(t, checkWorkflow(pipelineFile, src, commands), "LIMBUS_STATUS_URL")
	})
	t.Run("an expression in a script", func(t *testing.T) {
		src := mutate(t, pipeline, "bin/lct plan --json", "bin/lct plan --json # ${{ inputs.requeue }}")
		wantProblems(t, checkWorkflow(pipelineFile, src, commands), "expression")
	})
	t.Run("the status API URL in the environment of every step", func(t *testing.T) {
		src := mutate(t, pipeline, "LCT_MODEL: ${{ vars.LCT_MODEL }}", "LCT_MODEL: ${{ vars.LCT_MODEL }}\n          LIMBUS_STATUS_URL: ${{ secrets.LIMBUS_STATUS_URL }}")
		wantProblems(t, checkPipelineWorkflow(pipelineFile, src), "status API URL is in the environment")
	})
	t.Run("the API key in the environment of a step that does not translate", func(t *testing.T) {
		src := mutate(t, pipeline, "LIMBUS_CDN_TEMPLATE: ${{ vars.LIMBUS_CDN_TEMPLATE }}", "LIMBUS_CDN_TEMPLATE: ${{ vars.LIMBUS_CDN_TEMPLATE }}\n          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}")
		wantProblems(t, checkPipelineWorkflow(pipelineFile, src), "API key is in the environment")
	})
	t.Run("a secret that is required", func(t *testing.T) {
		src := mutate(t, pipeline, "ANTHROPIC_API_KEY: { required: false }", "ANTHROPIC_API_KEY: { required: true }")
		wantProblems(t, checkPipelineWorkflow(pipelineFile, src), "ANTHROPIC_API_KEY is required")
	})
	t.Run("a secret that is passed on by inherit", func(t *testing.T) {
		secretsBlock := regexp.MustCompile(`(?m)^    secrets:\n(?:      .*\n)+`)
		for file, src := range map[string]string{updateFile: update, rebuildFile: rebuild} {
			if !secretsBlock.MatchString(src) {
				t.Fatalf("%s has no secrets: block of the job to replace", file)
			}
			inherit := secretsBlock.ReplaceAllString(src, "    secrets: inherit\n")
			wantProblems(t, checkWorkflow(file, inherit, commands), "secrets: inherit")
		}
	})

	// 5. The actions and the runner are pinned.
	t.Run("an action that is not pinned", func(t *testing.T) {
		wantProblems(t, checkWorkflow(pipelineFile, mutate(t, pipeline, "actions/checkout@v5", "actions/checkout@main"), commands), "actions/checkout@main")
		wantProblems(t, checkWorkflow(pipelineFile, mutate(t, pipeline, "actions/setup-go@v6", "actions/setup-go@v5"), commands), "actions/setup-go@v5")
		wantProblems(t, checkWorkflow(ciFile, mutate(t, ci, "actions/checkout@v5\n        with:\n          fetch-depth: 0", "actions/checkout@v4\n        with:\n          fetch-depth: 0"), commands), "actions/checkout@v4")
		wantProblems(t, checkWorkflow(ciFile, mutate(t, ci, "actions/setup-node@v7", "actions/setup-node@v6"), commands), "actions/setup-node@v6")
	})
	t.Run("a runner that is not pinned", func(t *testing.T) {
		wantProblems(t, checkWorkflow(ciFile, mutate(t, ci, "  test:\n    runs-on: ubuntu-24.04", "  test:\n    runs-on: ubuntu-latest"), commands), "ubuntu-latest")
		wantProblems(t, checkWorkflow(ciFile, mutate(t, ci, "  site:\n    runs-on: ubuntu-24.04", "  site:\n    runs-on: ubuntu-latest"), commands), "ubuntu-latest")
		wantProblems(t, checkWorkflow(pipelineFile, mutate(t, pipeline, "runs-on: ubuntu-24.04", "runs-on: ubuntu-latest"), commands), "ubuntu-latest")
	})

	// 6. The claude CLI has a fixed version.
	t.Run("a claude CLI that is not pinned", func(t *testing.T) {
		for _, repl := range []string{"@anthropic-ai/claude-code@latest", "@anthropic-ai/claude-code", "@anthropic-ai/claude-code@2"} {
			src := mutate(t, pipeline, "@anthropic-ai/claude-code@2.1.316", repl)
			wantProblems(t, checkPipelineWorkflow(pipelineFile, src), "claude CLI")
		}
	})

	// 7. The schedule and the callers.
	t.Run("another cron", func(t *testing.T) {
		src := mutate(t, update, "'"+updateCron+"'", "'0 5 * * 4'")
		wantProblems(t, checkUpdateWorkflow(updateFile, src, pipeline), "0 5 * * 4")
	})
	for _, tc := range []struct{ name, file, src, old, repl, want string }{
		{"no dispatch event", updateFile, update, "limbus_update", "limbus_updat", "repository_dispatch"},
		{"fetch off in the update", updateFile, update, "fetch: true", "fetch: false", "passes fetch"},
		{"fetch on in the rebuild", rebuildFile, rebuild, "fetch: false", "fetch: true", "passes fetch"},
		{"another pipeline", updateFile, update, "uses: ./.github/workflows/pipeline.yml", "uses: ./.github/workflows/other.yml", "pipeline.yml"},
		{"no write permission for the issue", rebuildFile, rebuild, "issues: write", "issues: read", "permissions"},
		{"a branch other than main", rebuildFile, rebuild, "branches: [main]", "branches: [master]", "branches"},
		{"a rules path dropped", rebuildFile, rebuild, "'glossary/**', ", "", "paths"},
		{"an input the pipeline lacks", updateFile, update, "      max_usd: ${{ inputs.max_usd }}", "      maxusd: ${{ inputs.max_usd }}", `"maxusd"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := mutate(t, tc.src, tc.old, tc.repl)
			var problems []string
			if tc.file == updateFile {
				problems = checkUpdateWorkflow(tc.file, src, pipeline)
			} else {
				problems = checkRebuildWorkflow(tc.file, src, pipeline)
			}
			hasProblem(t, problems, tc.file, tc.want)
		})
	}
	t.Run("a secret the caller does not pass", func(t *testing.T) {
		src := mutate(t, update, "      LIMBUS_STATUS_URL: ${{ secrets.LIMBUS_STATUS_URL }}\n", "")
		wantProblems(t, checkUpdateWorkflow(updateFile, src, pipeline), "does not pass the secret LIMBUS_STATUS_URL")
	})
	t.Run("a concurrency group in a caller", func(t *testing.T) {
		src := mutate(t, rebuild, "jobs:\n", "concurrency: { group: lct-state, cancel-in-progress: false }\n\njobs:\n")
		wantProblems(t, checkRebuildWorkflow(rebuildFile, src, pipeline), "concurrency group")
	})

	// 8. The pipeline is made for a fresh clone, runs one at a time and
	// stops only when there is nothing to do.
	for _, tc := range []struct{ name, old, repl, want string }{
		{"no concurrency group", "group: lct-state", "group: other", "concurrency"},
		{"a run cancelled by the next", "cancel-in-progress: false", "cancel-in-progress: true", "concurrency"},
		{"a shallow clone", "fetch-depth: 0", "fetch-depth: 1", "fetch-depth: 0"},
		{"a job that may run past six hours", "timeout-minutes: 350", "timeout-minutes: 360", "timeout-minutes"},
		{"no write permission", "contents: write", "contents: read", "permissions"},
		{"no permission to open the issue", "issues: write", "issues: read", "permissions"},
		{"another trigger", "workflow_call:", "workflow_dispatch:", "workflow_call"},
		{"an input renamed", "dry_run: {", "dryrun: {", `"dry_run"`},
		{"an input of the wrong type", "max_usd: { type: string", "max_usd: { type: number", `"max_usd"`},
		{"a secret not declared", "LIMBUS_STATUS_URL: { required: false }", "LIMBUS_STATUS_URL_X: { required: false }", "LIMBUS_STATUS_URL"},
		{"no tag fetch", "+refs/tags/*:refs/tags/*", "+refs/heads/main:refs/remotes/origin/main", `"+refs/tags/*:refs/tags/*"`},
		{"no raw worktree", "git worktree add .work/raw raw", "git worktree prune", `"git worktree add .work/raw raw"`},
		{"no tw worktree", "git worktree add .work/tw tw", "git worktree prune", `"git worktree add .work/tw tw"`},
		{"no raw tag push", `git push origin "refs/tags/`, `echo "refs/tags/`, `git push origin \"refs/tags/`},
		{"no snapshot push", `git push origin raw "refs/tags/`, `echo raw "refs/tags/`, `git push origin raw \"refs/tags/`},
		{"no tw push", "git push origin tw", "git status", `"git push origin tw"`},
		{"an answer of origin that set -e does not see", "remote=$(git ls-remote --tags origin", ": $(git ls-remote --tags origin", `"remote=$(git ls-remote --tags origin"`},
		{"a Release tag not made at the tw commit", `--target "$tw_commit"`, "--target main", "not made at the tw commit"},
		{"assets read in a process substitution", "assets_json=$(jq -ec", "assets_json=$(cat out/publish.json; jq -ec", `"assets_json=$(jq -ec"`},
		{"no check of the assets", `"${#assets[@]}" -eq 2`, `"${#assets[@]}" -ge 0`, "number of assets is not checked"},
		{"assets read before they are captured", "assets_json=$(jq -ec", "mapfile -t assets </dev/null; assets_json=$(jq -ec", `"assets_json=$(jq -ec" must come before "mapfile -t assets"`},
		{"a push before the build", "bin/lct build --json", "git push origin tw && bin/lct build --json", `"bin/lct publish --json" must come before "git push origin tw"`},
		{"a release before the publish", "bin/lct publish --json >", "gh release create x && bin/lct publish --json >", `"git push origin tw" must come before "gh release create"`},
		{"a Release made when none is needed", " && steps.publish.outputs.release_needed == 'true'", "", "release_needed"},
		{"a Release that follows a test that is not the pinned one", "steps.publish.outputs.release_needed == 'true'", "steps.publish.outputs.release_needed != 'false'", "release_needed == 'true'"},
		{"a Release made when there is nothing to do", "if: steps.decide.outputs.idle == 'false' && steps.publish.outputs.release_needed == 'true'", "if: steps.publish.outputs.release_needed == 'true'", "steps.decide.outputs.idle == 'false'"},
		{"a translation that stops the build", "|| status=$?", "", "not followed by ||"},
		{"a translation that fails without a word", "::warning::lct translate stopped", "::notice::lct translate stopped", "::warning::"},
		{"no fallback of the spending limit", "|| vars.LCT_MAX_USD || '30'", "", "MAX_USD"},
		{"a build when nothing is to do", "        if: steps.decide.outputs.idle == 'false'\n        run: |\n          build=", "        run: |\n          build=", `"bin/lct build" runs even when there is nothing to do`},
		{"an issue when nothing is to do", "        if: steps.decide.outputs.idle == 'false'\n        env:\n          GH_TOKEN: ${{ github.token }}\n        run: |\n          if [ ! -f out/review-issue/01.md ]", "        env:\n          GH_TOKEN: ${{ github.token }}\n        run: |\n          if [ ! -f out/review-issue/01.md ]", `"gh issue" runs even when there is nothing to do`},
		{"no open issue is looked for", "gh issue list --state open --limit 500", "gh issue list --state all --limit 500", `"gh issue list --state open --limit 500 --json number,title,url"`},
		{"the issues searched for in the index", "gh issue list --state open --limit 500 --json", `gh issue list --state open --search "$title in:title" --limit 500 --json`, `"gh issue list --state open --limit 500 --json number,title,url"`},
		{"no title for the Release", `--title "$tag"`, "", "--title"},

		// The turn limit ends a run before the time limit does.
		{"a turn limit above the time limit", "--max-turns 130", "--max-turns 300", `--max-turns 130 --soft-turns 100 --run-timeout 90m`},
		{"no soft turn limit", "--soft-turns 100 ", "", `--soft-turns 100`},
		{"translations pushed only at the end", "--run-timeout 90m --push-each-run", "--run-timeout 90m", "--push-each-run"},

		// The translations are kept right after translating.
		{"translations kept without always()", "if: always() && steps.decide.outputs.translate == 'true' && steps.cli.outputs.installed == 'true'", "if: steps.decide.outputs.translate == 'true' && steps.cli.outputs.installed == 'true'", "always()"},
		{"translations kept when the translate step did not run", "if: always() && steps.decide.outputs.translate == 'true' && steps.cli.outputs.installed == 'true'", "if: always()", "exactly when the translate step does"},
		{"translations kept whatever decide says", "if: always() && steps.decide.outputs.translate == 'true' && steps.cli.outputs.installed == 'true'", "if: always() && steps.cli.outputs.installed == 'true'", "exactly when the translate step does"},
		{"translations not looked for", "git -C .work/tw status --porcelain", "git -C .work/tw status --short", `"git -C .work/tw status --porcelain"`},
		{"translations not committed", "git -C .work/tw commit", "git -C .work/tw stash", `"git -C .work/tw commit"`},
		{"translations committed with a message of no meaning", `-m "translate: keep the progress of $RAW_TAG"`, `-m "wip $RAW_TAG"`, `"translate: keep the progress of"`},
		{"translations committed with a message the idle check does not know", `-m "translate: keep the progress of $RAW_TAG"`, `-m "translate: kept the progress of $RAW_TAG"`, `"translate: keep the progress of"`},
		{"translations not pushed", "git -C .work/tw push origin tw", "git -C .work/tw status", `"git -C .work/tw push origin tw"`},

		// Units that are pending while the key or the model is missing are
		// recorded as untranslated, for the build and the review issue; a dry
		// run, which skips translating on purpose, is not.
		{"no record of a translation that did not run", `bin/lct translate --not-run "$reason"`, "echo skipped", "no step runs bin/lct translate --not-run"},
		{"a record whatever decide says", "        if: " + notRunIf + "\n", "", "not_run == 'true'"},
		{"a record under the if of the translation", "if: steps.decide.outputs.not_run == 'true' ||", "if: steps.decide.outputs.translate == 'true' ||", "not_run == 'true'"},
		{"no record when the claude CLI did not install", " || (steps.decide.outputs.translate == 'true' && steps.cli.outputs.installed == 'false')", "", "installed == 'false'"},
		{"a record of a dry run", "if: steps.decide.outputs.not_run == 'true' ||", "if: inputs.dry_run || steps.decide.outputs.not_run == 'true' ||", "a dry run is none"},
		{"a CLI that did not install recorded as a missing key", `            reason="安裝 Claude Code CLI 失敗，待譯的單位沒有翻譯"` + "\n", "", "安裝 Claude Code CLI 失敗"},
		{"no word from decide for the reason", "          NOT_RUN: ${{ steps.decide.outputs.not_run }}\n", "", "安裝 Claude Code CLI 失敗"},
		{"the API key for the record", "          NOT_RUN: ${{ steps.decide.outputs.not_run }}\n", "          NOT_RUN: ${{ steps.decide.outputs.not_run }}\n          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}\n", "ANTHROPIC_API_KEY in its environment"},
		{"a record that commits", `bin/lct translate --not-run "$reason"`, `bin/lct translate --not-run "$reason" && git -C .work/tw commit -am x`, "uses git"},
		{"a record after the build", `          bin/lct translate --not-run "$reason"`, `          bin/lct build --json && bin/lct translate --not-run "$reason"`, `"bin/lct translate --not-run" must come before "bin/lct build"`},
		{"a record before the requeue is counted", `            count=$(bin/lct pending --count --json)`, `            bin/lct translate --not-run x; count=$(bin/lct pending --count --json)`, `"bin/lct pending --count" must come before "bin/lct translate --not-run"`},
		{"a reason that names neither the key nor the model", `"缺少 ANTHROPIC_API_KEY 或 LCT_MODEL，待譯的單位沒有翻譯"`, `"待譯的單位沒有翻譯"`, "does not name ANTHROPIC_API_KEY"},
		{"a reason that names no model", "缺少 ANTHROPIC_API_KEY 或 LCT_MODEL，", "缺少 ANTHROPIC_API_KEY，", "does not name LCT_MODEL"},
		{"a record whose failure stops the run", ` || echo "::warning::lct translate --not-run failed`, "", "does not go on when it fails"},
		{"a missing model not noticed", ` || [ "$HAS_MODEL" != true ]; then`, "; then", `elif [ "$HAS_API_KEY" != true ] || [ "$HAS_MODEL" != true ]; then`},
		{"a missing key not noticed", `elif [ "$HAS_API_KEY" != true ] || `, "elif ", `elif [ "$HAS_API_KEY" != true ] || [ "$HAS_MODEL" != true ]; then`},
		{"a model that is not passed on", "          HAS_MODEL: ${{ vars.LCT_MODEL != '' }}\n", "", "HAS_MODEL"},
		{"a model asked of the secrets", "HAS_MODEL: ${{ vars.LCT_MODEL != '' }}", "HAS_MODEL: ${{ secrets.LCT_MODEL != '' }}", "HAS_MODEL"},
		{"a dry run not told from a missing key", "            if [ \"$DRY_RUN\" = true ]; then\n              echo \"::notice::dry_run: the $PENDING pending units are not translated\"\n            elif", "            if", `"$DRY_RUN" = true`},
		{"a dry run test that is inverted", `if [ "$DRY_RUN" = true ]; then`, `if [ "$DRY_RUN" = false ]; then`, `"$DRY_RUN" = true`},
		{"a translation that is also a record", "            else\n              translate=true\n", "            else\n              translate=true\n              not_run=true\n", "must set translate=true and not not_run=true"},
		{"a missing key that also translates", "              not_run=true\n            else", "              not_run=true\n              translate=true\n            else", "must set not_run=true and not translate=true"},
		{"a record decide does not report", "            echo \"not_run=$not_run\"\n", "", "not_run=$not_run"},
		{"a record decide has no default for", "          not_run=false\n", "", `"not_run=false"`},

		// The idle check: every term of it is pinned.
		{"idle without asking for a fetch", `[ "$FETCH" = true ] && `, "", `[ "$FETCH" = true ]`},
		{"idle with a new game update", ` && [ "$NEW" != true ]`, "", `[ "$NEW" != true ]`},
		{"idle with force", ` && [ "$FORCE" != true ]`, "", `[ "$FORCE" != true ]`},
		{"idle with units pending", ` && [ "$PENDING" -eq 0 ]`, "", `[ "$PENDING" -eq 0 ]`},
		{"idle with request or rewrite files applied", ` && [ "$APPLIED" -eq 0 ]`, "", `[ "$APPLIED" -eq 0 ]`},
		{"idle with a requeue asked for", ` && [ -z "$REQUEUE" ]`, "", `[ -z "$REQUEUE" ]`},
		{"idle with a tw that lags raw", ` && [ "$built_tag" = "$raw_tag" ]`, "", `[ "$built_tag" = "$raw_tag" ]`},
		{"idle when the tags differ", `[ "$built_tag" = "$raw_tag" ]`, `[ "$built_tag" != "$raw_tag" ]`, `[ "$built_tag" = "$raw_tag" ]`},
		{"idle terms joined by ||", `[ "$PENDING" -eq 0 ] && [ "$APPLIED" -eq 0 ]`, `[ "$PENDING" -eq 0 ] || [ "$APPLIED" -eq 0 ]`, "joined by &&"},
		{"the raw tag tw was built from is not read", "jq -r '.raw_tag // empty' .work/tw/.meta/build.json 2>/dev/null", "echo ''", `".work/tw/.meta/build.json"`},
		{"the Release still due is asked after tw is committed to", "release_check=$(bin/lct publish --check --json)", "release_check=$(git -C .work/tw commit --allow-empty -m x; bin/lct publish --check --json)", `"bin/lct publish --check" must come before "git -C .work/tw commit"`},
		{"the subject of tw asked after tw is committed to", "tw_subject=$(git -C .work/tw log -1 --format=%s)", "tw_subject=$(git -C .work/tw commit --allow-empty -m x; git -C .work/tw log -1 --format=%s)", `"git -C .work/tw log -1 --format=%s" must come before "git -C .work/tw commit"`},

		// The status answer is checked before it is written to the outputs.
		{"a status answer written unchecked", `[!A-Za-z0-9_:.+\ -]`, `[!A-Za-z0-9_:.+-]`, "unchecked"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := mutate(t, pipeline, tc.old, tc.repl)
			hasProblem(t, checkPipelineWorkflow(pipelineFile, src), pipelineFile, tc.want)
		})
	}
	// What the run does when nothing is left to do but a Release, or a build
	// that never came, and what it says of a Release it does not make: each
	// change is one problem, no more.
	for _, tc := range []struct{ name, old, repl, want string }{
		{"idle with a Release still due", ` && [ "$release_needed" = false ]`, "", `[ "$release_needed" = false ]`},
		{"idle when a Release is due", `[ "$built_tag" = "$raw_tag" ] && [ "$release_needed" = false ]`, `[ "$built_tag" = "$raw_tag" ] && [ "$release_needed" = true ]`, `[ "$release_needed" = false ]`},
		{"idle with a tw whose progress was kept and never built", ` && [[ $tw_subject != 'translate: keep the progress of'* ]]`, "", `[[ $tw_subject != 'translate: keep the progress of'* ]]`},
		{"idle when the last commit of tw keeps the progress", `[[ $tw_subject != 'translate: keep the progress of'* ]]`, `[[ $tw_subject == 'translate: keep the progress of'* ]]`, `[[ $tw_subject != 'translate: keep the progress of'* ]]`},
		{"idle for a prefix the commit does not have", `[[ $tw_subject != 'translate: keep the progress of'* ]]`, `[[ $tw_subject != 'translate: keep the progress'* ]]`, `[[ $tw_subject != 'translate: keep the progress of'* ]]`},
		{"the Release still due is not asked", "bin/lct publish --check --json", `echo '{"release_needed":false}'`, `"bin/lct publish --check --json"`},
		{"the subject of the last commit of tw is not read", "git -C .work/tw log -1 --format=%s", "git -C .work/tw log -1 --format=%H", `"git -C .work/tw log -1 --format=%s"`},
		{"a skipped Release not said", "語言包與 $same_as 相同，這次不建立 Release", "語言包沒有變", "這次不建立 Release"},
		{"a skipped Release said but not on the summary page", `} | tee -a "$GITHUB_STEP_SUMMARY"`, "}", "GITHUB_STEP_SUMMARY"},
		{"a skipped Release said when none is named", `[ "$release_needed" = false ] && [ -n "$same_as" ]`, `[ "$release_needed" = false ]`, `[ -n "$same_as" ]`},
		{"a skipped Release said when one is made", `[ "$release_needed" = false ] && [ -n "$same_as" ]`, `[ -n "$same_as" ]`, `[ -n "$same_as" ]`},
		{"assets refused without a word", "::error::publish.json has ${#assets[@]} assets, want 2", "publish.json has ${#assets[@]} assets, want 2", "::error::"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := mutate(t, pipeline, tc.old, tc.repl)
			wantProblems(t, checkPipelineWorkflow(pipelineFile, src), pipelineFile, tc.want)
		})
	}
	// The two pushes of tw each run in the retry loop: the one that keeps the
	// translations comes a second after the last push of lct translate.
	for _, tc := range []struct{ name, push string }{
		{"translations pushed once", "git -C .work/tw push origin tw"},
		{"tw pushed once", "git push origin tw"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			step := pipelineStepWith(t, pipeline, tc.push)
			src := mutate(t, pipeline, step, mutate(t, step, twPushRetry, "for wait in x; do"))
			wantProblems(t, checkPipelineWorkflow(pipelineFile, src), pipelineFile, fmt.Sprintf("the step that runs %q tries the push once", tc.push))
		})
		t.Run(tc.name+", its last failure let through", func(t *testing.T) {
			step := pipelineStepWith(t, pipeline, tc.push)
			src := mutate(t, pipeline, step, mutate(t, step, twPushGiveUp, "true"))
			wantProblems(t, checkPipelineWorkflow(pipelineFile, src), pipelineFile, fmt.Sprintf("the step that runs %q tries the push once", tc.push))
		})
	}
	t.Run("translations never kept", func(t *testing.T) {
		src := mutate(t, pipeline, "          "+keepCommitIf+"\n            git -C .work/tw add -A\n            git -C .work/tw commit --quiet -m \"translate: keep the progress of $RAW_TAG\"\n          fi\n", "")
		src = mutate(t, src, "            git -C .work/tw push origin tw && break\n", "            true && break\n")
		hasProblem(t, checkPipelineWorkflow(pipelineFile, src), `"git -C .work/tw commit"`)
	})
	// A run whose push failed leaves tw ahead of origin with a clean
	// worktree: the push comes after the if, not inside it.
	t.Run("translations pushed only when the worktree changed", func(t *testing.T) {
		step := pipelineStepWith(t, pipeline, "git -C .work/tw push origin tw")
		moved := mutate(t, step, "$RAW_TAG\"\n          fi\n", "$RAW_TAG\"\n")
		moved = mutate(t, moved, "          done\n", "          done\n          fi\n")
		src := mutate(t, pipeline, step, moved)
		wantProblems(t, checkPipelineWorkflow(pipelineFile, src), pipelineFile, "pushes only when the worktree changed")
	})
	t.Run("translations committed when nothing changed", func(t *testing.T) {
		step := pipelineStepWith(t, pipeline, "git -C .work/tw push origin tw")
		moved := mutate(t, step, "            git -C .work/tw commit --quiet -m \"translate: keep the progress of $RAW_TAG\"\n          fi\n", "          fi\n            git -C .work/tw commit --quiet -m \"translate: keep the progress of $RAW_TAG\"\n")
		src := mutate(t, pipeline, step, moved)
		wantProblems(t, checkPipelineWorkflow(pipelineFile, src), pipelineFile, "commits outside its if")
	})
	t.Run("translations kept only after the build", func(t *testing.T) {
		src := mutate(t, pipeline, "bin/lct build --json", "bin/lct build --json; git -C .work/tw push origin tw")
		src = mutate(t, src, "            git -C .work/tw push origin tw && break\n", "            git -C .work/tw status && break\n")
		hasProblem(t, checkPipelineWorkflow(pipelineFile, src), `"git -C .work/tw push origin tw" must come before "bin/lct build"`)
	})
	t.Run("the Release still due asked outside the decide step", func(t *testing.T) {
		src := mutate(t, pipeline, "release_check=$(bin/lct publish --check --json)", "release_check=$(cat release.json)")
		src = mutate(t, src, "          plan=$(bin/lct plan --json)\n", "          bin/lct publish --check --json > release.json\n          plan=$(bin/lct plan --json)\n")
		wantProblems(t, checkPipelineWorkflow(pipelineFile, src), "the decide step does not ask release_check=$(bin/lct publish --check --json)")
	})
	t.Run("the claude CLI installed where the API key is", func(t *testing.T) {
		src := mutate(t, pipeline, "          status=0\n          bin/lct translate", "          npm install -g @anthropic-ai/claude-code@2.1.316\n          status=0\n          bin/lct translate")
		hasProblem(t, checkPipelineWorkflow(pipelineFile, src), "translate step installs the claude CLI")
	})
	t.Run("the API key in the environment of the install step", func(t *testing.T) {
		src := mutate(t, pipeline, "        id: cli\n        if: steps.decide.outputs.translate == 'true'\n", "        id: cli\n        if: steps.decide.outputs.translate == 'true'\n        env:\n          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}\n")
		wantProblems(t, checkPipelineWorkflow(pipelineFile, src), "API key is in the environment")
	})
	t.Run("an install that runs when there is nothing to translate", func(t *testing.T) {
		src := mutate(t, pipeline, "        id: cli\n        if: steps.decide.outputs.translate == 'true'\n", "        id: cli\n")
		hasProblem(t, checkPipelineWorkflow(pipelineFile, src), `"npm install" runs even when there is nothing to do`)
	})
	t.Run("a translation that runs although the CLI did not install", func(t *testing.T) {
		src := mutate(t, pipeline, "        if: steps.decide.outputs.translate == 'true' && steps.cli.outputs.installed == 'true'\n", "        if: steps.decide.outputs.translate == 'true'\n")
		hasProblem(t, checkPipelineWorkflow(pipelineFile, src), "steps.cli.outputs.installed == 'true'")
	})
	t.Run("the CLI allowed to update itself", func(t *testing.T) {
		src := mutate(t, pipeline, "          DISABLE_AUTOUPDATER: '1'\n", "")
		hasProblem(t, checkPipelineWorkflow(pipelineFile, src), "DISABLE_AUTOUPDATER")
	})
	t.Run("no token for gh", func(t *testing.T) {
		src := mutateAll(t, pipeline, "GH_TOKEN: ${{ github.token }}", "GH_USER: x")
		problems := checkPipelineWorkflow(pipelineFile, src)
		hasProblem(t, problems, "gh release", "GH_TOKEN")
		hasProblem(t, problems, "gh issue", "GH_TOKEN")
	})
	t.Run("no Release number unless the tags are there", func(t *testing.T) {
		src := mutate(t, pipeline, "+refs/tags/*:refs/tags/*", "+refs/heads/main:refs/remotes/origin/main")
		hasProblem(t, checkPipelineWorkflow(pipelineFile, src), "tags are not fetched")
	})

	// 9. Every secret and variable a workflow uses is in the README.
	t.Run("a variable the README does not list", func(t *testing.T) {
		src := mutate(t, pipeline, "LCT_MODEL: ${{ vars.LCT_MODEL }}", "LCT_MODEL: ${{ vars.LCT_MODEL }}\n          FOO: ${{ vars.FOO }}")
		wantProblems(t, checkRefsDocumented(map[string]string{"pipeline.yml": src}, readme), "vars.FOO")
	})
	t.Run("a row dropped from the README", func(t *testing.T) {
		for _, name := range []string{"ANTHROPIC_API_KEY", "LIMBUS_STATUS_URL", "LIMBUS_CDN_TEMPLATE", "LCT_MODEL", "LCT_MAX_USD"} {
			var kept []string
			for _, line := range strings.Split(readme, "\n") {
				if !(strings.HasPrefix(line, "|") && strings.Contains(line, "`"+name+"`")) {
					kept = append(kept, line)
				}
			}
			if len(kept) == len(strings.Split(readme, "\n")) {
				t.Fatalf("README.md has no table row for %s", name)
			}
			wantProblems(t, checkRefsDocumented(files, strings.Join(kept, "\n")), name)
		}
	})
	t.Run("a README without the section", func(t *testing.T) {
		src := mutate(t, readme, "## "+readmeActions, "## 設定")
		wantProblems(t, checkRefsDocumented(files, src), readmeActions)
	})
	t.Run("a name in a table of another section", func(t *testing.T) {
		section, _ := readmeSection(readme, readmeActions)
		src := strings.Replace(readme, section, "\n", 1)
		src += "\n## 別處\n\n| 名稱 |\n|---|\n| `ANTHROPIC_API_KEY` `LIMBUS_STATUS_URL` `LIMBUS_CDN_TEMPLATE` `LCT_MODEL` `LCT_MAX_USD` |\n"
		problems := checkRefsDocumented(files, src)
		for _, name := range []string{"ANTHROPIC_API_KEY", "LIMBUS_STATUS_URL", "LIMBUS_CDN_TEMPLATE", "LCT_MODEL", "LCT_MAX_USD"} {
			hasProblem(t, problems, name)
		}
	})

	// 10. The name of the Release.
	for _, title := range []string{`"Claude Code $tag"`, `Anthropic-$tag`, `'zh-TW by claude  code'`} {
		t.Run("title "+title, func(t *testing.T) {
			src := mutate(t, pipeline, `--title "$tag"`, "--title "+title)
			wantProblems(t, checkWorkflow(pipelineFile, src, commands), "--title", title)
		})
	}

	// 11. ci.yml says on a pull request how many existing translations a change
	// of the rules would break: the step exists, runs only on a pull request,
	// takes the name of the base branch from env, and fetches nothing when no
	// rule changed.
	for _, tc := range []struct{ name, old, repl, want string }{
		{"a checkout that is shallow", "fetch-depth: 0", "fetch-depth: 1", "fetch-depth: 0"},
		{"runs on pushes only", "on: [push, pull_request]", "on: [push]", "runs on"},
		{"runs on every event", "on: [push, pull_request]", "on: [push, pull_request, workflow_dispatch]", "runs on"},
		{"a token that writes", "contents: read", "contents: write", "permissions"},
		{"a job with a token of its own", "  test:\n    runs-on: ubuntu-24.04\n", "  test:\n    runs-on: ubuntu-24.04\n    permissions: { contents: write }\n", "permissions of its own"},
		{"the job of the viewer with a token of its own", "  site:\n    runs-on: ubuntu-24.04\n", "  site:\n    runs-on: ubuntu-24.04\n    permissions: { contents: write }\n", "permissions of its own"},

		// Only on a pull request.
		{"assessed on a push too", "        if: github.event_name == 'pull_request'\n", "", "github.event_name == 'pull_request'"},
		{"assessed on a push that is not guarded", "if: github.event_name == 'pull_request'", "if: github.event_name != 'push'", "github.event_name == 'pull_request'"},

		// The name of the base branch goes through env.
		{"base_ref in a script", `--base "origin/$BASE_REF"`, `--base "origin/${{ github.base_ref }}"`, "base_ref"},
		{"base_ref in the diff", `"origin/$BASE_REF...HEAD"`, `"origin/${{ github.base_ref }}...HEAD"`, "base_ref"},
		{"base_ref that env does not carry", "BASE_REF: ${{ github.base_ref }}", "BASE_REF: main", "BASE_REF"},
		{"another ref in env", "BASE_REF: ${{ github.base_ref }}", "BASE_REF: ${{ github.head_ref }}", "BASE_REF"},

		// A pull request that changes no rule is not assessed.
		{"no look at what the pull request changes", `changed=$(git diff --name-only "origin/$BASE_REF...HEAD" -- guides glossary rules)`, "changed=x", "git diff"},
		{"a diff that is not against the merge base", `"origin/$BASE_REF...HEAD"`, `"origin/$BASE_REF..HEAD"`, "merge base"},
		{"a diff of every path", " -- guides glossary rules", "", "guides/, glossary/ or rules/"},
		{"a diff that does not name rules", " -- guides glossary rules", " -- guides glossary", "guides/, glossary/ or rules/"},
		{"a diff that set -e does not see", `changed=$(git diff`, `: $(git diff`, "assigned before it is tested"},
		{"assessed although no rule changed", `[ -z "$changed" ]`, `[ -n "$changed" ]`, "changes no rule"},
		{"fetched although no rule changed", `if [ -z "$changed" ]; then`, "git fetch origin '+refs/heads/raw:refs/remotes/origin/raw'\n          if [ -z \"$changed\" ]; then", "must come before"},

		// The worktrees and the command.
		{"no raw branch", "+refs/heads/raw:refs/remotes/origin/raw", "+refs/heads/main:refs/remotes/origin/main", "raw branch"},
		{"no tw branch", "+refs/heads/tw:refs/remotes/origin/tw", "+refs/heads/main:refs/remotes/origin/main", "tw branch"},
		{"no raw worktree for the assessment", "git worktree add .work/raw raw", "git worktree prune", "raw worktree"},
		{"no tw worktree for the assessment", "git worktree add .work/tw tw", "git worktree prune", "tw worktree"},
		{"worktrees made after the assessment", "          git fetch origin", "          bin/lct impact --base HEAD\n          git fetch origin", `must come before "bin/lct impact"`},
		{"the assessment against the working tree", `--base "origin/$BASE_REF"`, `--base HEAD`, "base branch"},
		{"no file for the summary", "--markdown out/impact.md", "--json", "summary page"},
		{"not shown on the summary page", `out/impact.md >> "$GITHUB_STEP_SUMMARY"`, "out/impact.md", "summary page"},
		{"shown before it is written", "          bin/lct impact --base", "          cat out/impact.md >> \"$GITHUB_STEP_SUMMARY\"\n          bin/lct impact --base", "must come before"},
		{"no step for the assessment", "bin/lct impact", "echo impact", "0 steps run bin/lct impact"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := mutate(t, ci, tc.old, tc.repl)
			hasProblem(t, checkCIWorkflow(ciFile, src), ciFile, tc.want)
		})
	}
	t.Run("assessed before lct is built", func(t *testing.T) {
		// The build step moves to the end of the job test, behind the assessment,
		// which is the last step of the job.
		build := "      - name: Build lct\n        run: go build -o bin/lct ./cmd/lct\n"
		src := strings.Replace(mutate(t, ci, build, ""), "\n  # The viewer", build+"\n  # The viewer", 1)
		hasProblem(t, checkCIWorkflow(ciFile, src), ciFile, `"go build -o bin/lct ./cmd/lct" must come before "bin/lct impact"`)
	})
	t.Run("an unregistered lct command in ci.yml", func(t *testing.T) {
		src := mutate(t, ci, "bin/lct impact", "bin/lct imapct")
		wantProblems(t, checkWorkflow(ciFile, src, commands), `"bin/lct imapct"`)
	})
	t.Run("base_ref in a script is an expression too", func(t *testing.T) {
		src := mutate(t, ci, `--base "origin/$BASE_REF"`, `--base "origin/${{ github.base_ref }}"`)
		hasProblem(t, checkWorkflow(ciFile, src, commands), "expression")
	})
	t.Run("no model name", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, root, workflowDir+"/ci.yml", ci+"# claude-example-9\n")
		wantProblems(t, checkModelNames(root, workflowDir), "ci.yml", `"claude-example-9"`)
	})

	// 12. rules-proposal.yml runs the rules-review skill after an update or a
	// rebuild that ended green, and by hand: one job at a time with the
	// permissions the rules PR is made with; it ends when there is nothing new to
	// suggest; the action gets the key, the model and the tools of the skill and
	// no token of its own; and a run that opened no PR is not green.
	for _, tc := range []struct{ name, old, repl, want string }{
		// The events.
		{"not after a rebuild", "workflows: [update, rebuild]", "workflows: [update]", "workflow_run workflows"},
		{"after another workflow", "workflows: [update, rebuild]", "workflows: [update, rebuild, ci]", "workflow_run workflows"},
		{"not when the run is completed", "types: [completed]", "types: [requested]", "workflow_run types"},
		{"also on a schedule", "on:\n  workflow_run:", "on:\n  schedule:\n    - cron: '0 0 * * *'\n  workflow_run:", "runs on"},
		{"not by hand", "  workflow_dispatch:\n", "", "runs on"},
		{"a run that did not succeed", "github.event.workflow_run.conclusion == 'success'", "github.event.workflow_run.conclusion != 'failure'", "the if of the job"},
		{"a run that is not by hand and not green", " || (github.event.workflow_run", " && (github.event.workflow_run", "the if of the job"},
		{"a run of another branch", " && github.event.workflow_run.head_branch == 'main'", "", "the if of the job"},
		{"a run for a pull request", " && github.event.workflow_run.event != 'pull_request'", "", "the if of the job"},

		// The job.
		{"a job that may run for hours", "timeout-minutes: 60", "timeout-minutes: 120", "timeout-minutes"},
		{"another concurrency group", "group: rules-proposal", "group: lct-state", "concurrency"},
		{"a run cancelled by the next", "cancel-in-progress: false", "cancel-in-progress: true", "concurrency"},
		{"no permission for the pull request", "pull-requests: write, ", "", "permissions"},
		{"no permission for the app token", ", id-token: write", "", "permissions"},
		{"a shallow clone", "fetch-depth: 0", "fetch-depth: 1", "fetch-depth: 0"},

		// The action.
		{"a failed review that does not say why", `select(.type == "result")`, `select(.type == "assistant")`, "a review that failed does not say why"},
		{"a review in CI that is not asked for", " && vars.RULES_REVIEW_IN_CI == 'true'", "", "would review the rules in CI"},
		{"no word of a review left to a person", "if: steps.suggest.outputs.count != '0' && vars.LCT_MODEL != '' && vars.RULES_REVIEW_IN_CI != 'true'", "if: false", "no step runs if vars.RULES_REVIEW_IN_CI != 'true'"},
		{"a review left to a person that ends red", "/rules-review\" > work/proposal/outcome.txt", "/rules-review\" > work/proposal/out.txt", "an outcome of nothing that the check accepts"},
		{"another prompt", `prompt: "/rules-review"`, `prompt: "/rules-review now"`, "the prompt of the action"},
		{"no prompt of the skill", `prompt: "/rules-review"`, `prompt: "review the rules"`, "the prompt of the action"},
		{"a key written in the file", "anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}", "anthropic_api_key: sk-example", "anthropic_api_key"},
		{"a model written in the file", "--model ${{ vars.LCT_MODEL }}", "--model example", "--model ${{ vars.LCT_MODEL }}"},
		{"no limit of turns", "            --max-turns 80\n", "", "--max-turns 80"},
		{"another limit of turns", "--max-turns 80", "--max-turns 800", "--max-turns 80"},
		{"permissions skipped", "            --max-turns 80\n", "            --max-turns 80\n            --dangerously-skip-permissions\n", "--dangerously-skip-permissions"},
		{"a token for the action", "        uses: anthropics/claude-code-action@v1\n", "        uses: anthropics/claude-code-action@v1\n        env:\n          GH_TOKEN: ${{ github.token }}\n", "GH_TOKEN"},
		{"a run with nothing to suggest", "        if: steps.suggest.outputs.count != '0' && vars.LCT_MODEL != '' && vars.RULES_REVIEW_IN_CI == 'true'\n        uses: anthropics/claude-code-action@v1\n", "        if: vars.LCT_MODEL != '' && vars.RULES_REVIEW_IN_CI == 'true'\n        uses: anthropics/claude-code-action@v1\n", "the if of the action step"},
		{"an action that runs when the count is not read", "steps.suggest.outputs.count != '0' && vars.LCT_MODEL != '' && vars.RULES_REVIEW_IN_CI == 'true'\n        uses: anthropics", "steps.start.outputs.pr != '0' && vars.LCT_MODEL != '' && vars.RULES_REVIEW_IN_CI == 'true'\n        uses: anthropics", "the if of the action step"},
		{"an action without a model", "steps.suggest.outputs.count != '0' && vars.LCT_MODEL != '' && vars.RULES_REVIEW_IN_CI == 'true'\n        uses: anthropics", "steps.suggest.outputs.count != '0' && vars.RULES_REVIEW_IN_CI == 'true'\n        uses: anthropics", "vars.LCT_MODEL != ''"},

		// Without the model a step says so and writes an outcome of nothing.
		{"no step for a missing model", "steps.suggest.outputs.count != '0' && vars.LCT_MODEL == ''", "steps.suggest.outputs.count != '0'", "no step runs if vars.LCT_MODEL == ''"},
		{"a step for a missing model that runs with nothing to suggest", "if: steps.suggest.outputs.count != '0' && vars.LCT_MODEL == ''", "if: vars.LCT_MODEL == ''", "the if of the step for a missing LCT_MODEL"},
		{"a missing model without a warning", `echo "::warning::the variable LCT_MODEL is not set`, `echo "the variable LCT_MODEL is not set`, "does not warn"},
		{"a missing model without an outcome", "echo 'nothing: 沒有設定 LCT_MODEL，這次不審查建議' > work/proposal/outcome.txt", "true", "outcome of nothing"},

		// The tools of the action are exactly those of the skill.
		{"a tool less", "Bash(bin/lct impact --base origin/main --markdown work/proposal/impact.md*),", "", "Bash(bin/lct impact --base origin/main --markdown work/proposal/impact.md*)"},
		{"an impact that writes anywhere", "Bash(bin/lct impact --base origin/main --markdown work/proposal/impact.md*)", "Bash(bin/lct impact *)", "Bash(bin/lct impact *)"},
		{"a tool more", "Read(./**),", "Read(./**),Bash(git push *),", "Bash(git push *)"},
		{"a tool twice", "Read(./**),", "Read(./**),Read(./**),", "twice"},
		{"a second list", "            --max-turns 80\n", "            --max-turns 80\n            --allowedTools \"Read(./**)\"\n", "2 --allowedTools"},
		{"no list", `--allowedTools "`, `--tools "`, "0 --allowedTools"},

		// What is run, and when.
		{"no raw branch", "+refs/heads/raw:refs/remotes/origin/raw", "+refs/heads/main:refs/remotes/origin/main", "raw branch"},
		{"no tw branch", "+refs/heads/tw:refs/remotes/origin/tw", "+refs/heads/main:refs/remotes/origin/main", "tw branch"},
		{"no raw worktree", "git worktree add .work/raw raw", "git worktree prune", "raw worktree"},
		{"no tw worktree", "git worktree add .work/tw tw", "git worktree prune", "tw worktree"},
		{"no git identity", `git config user.email "41898282+github-actions[bot]@users.noreply.github.com"`, "git config --list", "git config user.email"},
		{"the branch not chosen", "bin/lct proposal start", "bin/lct proposal begin", "proposal start"},
		{"the branch chosen after the suggestions", "          started=$(bin/lct proposal start", "          bin/lct suggest --json\n          started=$(bin/lct proposal start", `"bin/lct proposal start" must come before "bin/lct suggest"`},
		{"the branch chosen before lct is built", "        run: go build -o bin/lct ./cmd/lct\n", "        run: go version\n", "go build -o bin/lct ./cmd/lct"},
		{"suggestions that leave nothing out", " --exclude-file work/proposal/exclude.txt", "", "--exclude-file"},
		{"no token for the choice of the branch", "        id: start\n        env:\n          GH_TOKEN: ${{ github.token }}\n", "        id: start\n", "bin/lct proposal start has no GH_TOKEN"},
		{"the suggestions not named", "        id: suggest\n", "        id: count\n", "id: suggest"},
		{"a look for the PR through the label", `gh pr list --state open --head "$BRANCH"`, "gh pr list --state open --label rules-proposal", "--head"},
		{"a look for the PR that is not asked", "        if: steps.suggest.outputs.count != '0'\n        env:\n          GH_TOKEN: ${{ github.token }}\n          BRANCH:", "        env:\n          GH_TOKEN: ${{ github.token }}\n          BRANCH:", "the if of the step that looks for the rules PR"},
		{"no token for the look", "          GH_TOKEN: ${{ github.token }}\n          BRANCH:", "          BRANCH:", "looks for the rules PR has no GH_TOKEN"},
		{"another branch looked for", "BRANCH: ${{ steps.start.outputs.branch }}", "BRANCH: main", "BRANCH"},
		{"a secret in another step", "        id: start\n        env:\n", "        id: start\n        env:\n          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}\n", "other than the action"},
		{"a key of lct proposal start that was renamed", `jq -er .branch <<<"$started"`, `jq -er .branc <<<"$started"`, "branc"},
		{"a key of lct suggest that was renamed", "jq -er '.suggestions | length'", "jq -er '.suggestion | length'", "suggestion"},

		// The token of the checkout is not where the action can read it.
		{"a checkout that keeps its token", "          persist-credentials: false\n", "", "persist-credentials: false"},
		{"a checkout that says to keep its token", "persist-credentials: false", "persist-credentials: true", "persist-credentials: false"},
		{"the folder of git open", "            --disallowedTools \"Read(./.git/**)\"\n", "", "--disallowedTools"},
		{"another folder closed", `--disallowedTools "Read(./.git/**)"`, `--disallowedTools "Read(./work/**)"`, "--disallowedTools"},

		// What the skill says it did is read and told apart.
		{"the outcome not read", "outcome_file=work/proposal/outcome.txt", "outcome_file=work/proposal/other.txt", "work/proposal/outcome.txt"},
		{"a run that proposes nothing not told apart", "'nothing: '*)", "'nothing - '*)", "'nothing: '*)"},
		{"a run that proposes nothing not said", "echo \"::notice::the rules-review run", "echo \"::warning::the rules-review run", "::notice::"},
		{"a local run in CI not told apart", "'nothing: 本機'*)", "'nothing: local'*)", "'nothing: 本機'*)"},
		{"the description of the PR not read", "--json url,body,isCrossRepository", "--json url,isCrossRepository", "--json url,body,isCrossRepository"},
		{"the suggestions the PR had not told from the new", `grep -qxF -- "$id" work/proposal/exclude.txt`, `grep -qF -- "$id" work/proposal/other.txt`, "grep -qxF"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := mutate(t, rules, tc.old, tc.repl)
			hasProblem(t, checkRulesProposalWorkflow(rulesFile, src, skillTools), rulesFile, tc.want)
		})
	}
	// Each change below is one problem, no more.
	for _, tc := range []struct{ name, old, repl, want string }{
		{"a tool less", "Bash(bin/lct impact --base origin/main --markdown work/proposal/impact.md*),", "", "Bash(bin/lct impact --base origin/main --markdown work/proposal/impact.md*)"},
		{"a tool more", "Read(./**),", "Read(./**),Bash(git push *),", "Bash(git push *)"},
		{"a tool twice", "Read(./**),", "Read(./**),Read(./**),", "twice"},
		{"a model written in the file", "--model ${{ vars.LCT_MODEL }}", "--model example", "--model"},
		{"another prompt", `prompt: "/rules-review"`, `prompt: "/rules-review now"`, "the prompt"},
		{"a token for the action", "        uses: anthropics/claude-code-action@v1\n", "        uses: anthropics/claude-code-action@v1\n        env:\n          GITHUB_TOKEN: ${{ github.token }}\n", "GITHUB_TOKEN"},
		{"another concurrency group", "group: rules-proposal", "group: lct-state", "concurrency"},
		{"no permission for the pull request", "pull-requests: write, ", "", "permissions"},
		{"a checkout that keeps its token", "          persist-credentials: false\n", "", "persist-credentials: false"},
		{"the folder of git open", "            --disallowedTools \"Read(./.git/**)\"\n", "", "--disallowedTools"},
	} {
		t.Run("exactly one problem for "+tc.name, func(t *testing.T) {
			src := mutate(t, rules, tc.old, tc.repl)
			wantProblems(t, checkRulesProposalWorkflow(rulesFile, src, skillTools), rulesFile, tc.want)
		})
	}
	t.Run("a tool the skill lacks", func(t *testing.T) {
		const impactTool = "Bash(bin/lct impact --base origin/main --markdown work/proposal/impact.md*)"
		tools := slices.DeleteFunc(slices.Clone(skillTools), func(tool string) bool { return tool == impactTool })
		wantProblems(t, checkRulesProposalWorkflow(rulesFile, rules, tools), "--allowedTools has", impactTool, "lacks")
	})
	t.Run("a tool the workflow lacks", func(t *testing.T) {
		tools := append(slices.Clone(skillTools), "Bash(bin/lct validate *)")
		wantProblems(t, checkRulesProposalWorkflow(rulesFile, rules, tools), "--allowedTools lacks", "Bash(bin/lct validate *)")
	})
	t.Run("an action that is not pinned", func(t *testing.T) {
		src := mutate(t, rules, "anthropics/claude-code-action@v1", "anthropics/claude-code-action@main")
		wantProblems(t, checkWorkflow(rulesFile, src, commands), "anthropics/claude-code-action@main")
	})
	t.Run("another action", func(t *testing.T) {
		src := mutate(t, rules, "anthropics/claude-code-action@v1", "anthropics/other-action@v1")
		wantProblems(t, checkWorkflow(rulesFile, src, commands), "anthropics/other-action@v1")
	})
	t.Run("a runner that is not pinned", func(t *testing.T) {
		wantProblems(t, checkWorkflow(rulesFile, mutate(t, rules, "runs-on: ubuntu-24.04", "runs-on: ubuntu-latest"), commands), "ubuntu-latest")
	})
	t.Run("an unregistered lct command in rules-proposal.yml", func(t *testing.T) {
		src := mutate(t, rules, "bin/lct suggest --json --exclude-file", "bin/lct sugest --json --exclude-file")
		wantProblems(t, checkWorkflow(rulesFile, src, commands), `"bin/lct sugest"`)
	})
	t.Run("an unregistered lct command in the tools of the action", func(t *testing.T) {
		src := mutate(t, rules, "Bash(bin/lct impact --base", "Bash(bin/lct impct --base")
		wantProblems(t, checkWorkflow(rulesFile, src, commands), `"bin/lct impct"`)
	})
	t.Run("an expression in a script", func(t *testing.T) {
		src := mutate(t, rules, `"$(date -u +%F)"`, `"${{ github.event.repository.pushed_at }}"`)
		wantProblems(t, checkWorkflow(rulesFile, src, commands), "expression")
	})
	t.Run("every secret handed on", func(t *testing.T) {
		src := mutate(t, rules, "    timeout-minutes: 60\n", "    timeout-minutes: 60\n    secrets: inherit\n")
		wantProblems(t, checkWorkflow(rulesFile, src, commands), "secrets: inherit")
	})
	t.Run("a secret that is not the key", func(t *testing.T) {
		src := mutate(t, rules, "anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}", "anthropic_api_key: ${{ secrets.OTHER_KEY }}")
		hasProblem(t, checkWorkflow(rulesFile, src, commands), "secrets.OTHER_KEY")
	})
	t.Run("the key in the with of another step", func(t *testing.T) {
		src := mutate(t, rules, "          go-version-file: go.mod\n", "          go-version-file: go.mod\n          token: ${{ secrets.ANTHROPIC_API_KEY }}\n")
		wantProblems(t, checkRulesProposalWorkflow(rulesFile, src, skillTools), "a secret in its with", "actions/setup-go@v6")
	})
	t.Run("a workflow it waits for renamed", func(t *testing.T) {
		for _, name := range []string{"update.yml", "rebuild.yml"} {
			renamed := maps.Clone(files)
			renamed[name] = mutate(t, files[name], "name: "+strings.TrimSuffix(name, ".yml")+"\n", "name: "+strings.TrimSuffix(name, ".yml")+" pack\n")
			wantProblems(t, checkWorkflowRunNames(renamed), "workflow_run workflows", strings.TrimSuffix(name, ".yml")+" pack")
		}
		other := maps.Clone(files)
		other["rules-proposal.yml"] = mutate(t, rules, "workflows: [update, rebuild]", "workflows: [update, rebuild, ci]")
		wantProblems(t, checkWorkflowRunNames(other), "workflow_run workflows")
	})
	t.Run("a model name", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, root, workflowDir+"/rules-proposal.yml", rules+"# claude-example-9\n")
		wantProblems(t, checkModelNames(root, workflowDir), "rules-proposal.yml", `"claude-example-9"`)
	})

	// 13. pages.yml builds the site and deploys it to Pages after an update or
	// a rebuild that ended green, and by hand: one deployment at a time, with the
	// permissions of a Pages deployment and no secret; the data of the site is
	// written before the build, and a step that fails fails the run, so that
	// nothing is deployed with missing or stale data.
	const (
		pagesSiteStep   = "      - name: Write the data of the site\n        run: bin/lct site --out site/public/data --json\n"
		pagesBuildStep  = "        run: npm run build\n        working-directory: site\n"
		pagesUploadStep = "      - uses: actions/upload-pages-artifact@v5\n        with:\n          path: site/dist\n"
		pagesDeployStep = "      - uses: actions/deploy-pages@v5\n"
		siteNodeStep    = "      - uses: actions/setup-node@v7\n        with:\n          node-version-file: site/.nvmrc\n          cache: npm\n          cache-dependency-path: site/package-lock.json\n"
	)
	for _, tc := range []struct{ name, old, repl, want string }{
		// The events.
		{"not after a rebuild", "workflows: [update, rebuild]", "workflows: [update]", "workflow_run workflows"},
		{"after another workflow", "workflows: [update, rebuild]", "workflows: [update, rebuild, ci]", "workflow_run workflows"},
		{"not when the run is completed", "types: [completed]", "types: [requested]", "workflow_run types"},
		{"also on a schedule", "on:\n  workflow_run:", "on:\n  schedule:\n    - cron: '0 0 * * *'\n  workflow_run:", "runs on"},
		{"also on every push", "on:\n  workflow_run:", "on:\n  push:\n  workflow_run:", "runs on"},
		{"not by hand", "  workflow_dispatch:\n", "", "runs on"},
		{"an input by hand", "  workflow_dispatch:\n", "  workflow_dispatch:\n    inputs:\n      force: { type: boolean }\n", `input "force"`},

		// The condition: only a green run of main, as for the rules PR.
		{"a run that did not succeed", "github.event.workflow_run.conclusion == 'success'", "github.event.workflow_run.conclusion != 'failure'", "the if of the job"},
		{"a run that is not by hand and not green", " || (github.event.workflow_run", " && (github.event.workflow_run", "the if of the job"},
		{"a run of another branch", " && github.event.workflow_run.head_branch == 'main'", "", "the if of the job"},
		{"a run for a pull request", " && github.event.workflow_run.event != 'pull_request'", "", "the if of the job"},

		// The job.
		{"a job that may run for hours", "timeout-minutes: 30", "timeout-minutes: 360", "timeout-minutes"},
		{"another concurrency group", "group: pages", "group: lct-state", "concurrency"},
		{"a deployment cancelled by the next", "cancel-in-progress: false", "cancel-in-progress: true", "concurrency"},
		{"no permission for the id token", ", id-token: write", "", "permissions"},
		{"no permission for pages", "pages: write, ", "", "permissions"},
		{"a token that writes the repo", "contents: read", "contents: write", "permissions"},
		{"a permission more", "id-token: write }", "id-token: write, issues: write }", "permissions"},
		{"no environment", "    environment: github-pages\n", "", "environment"},
		{"another environment", "environment: github-pages", "environment: production", "environment"},

		// The checkout has every tag and keeps no token.
		{"a shallow clone", "fetch-depth: 0", "fetch-depth: 1", "fetch-depth: 0"},
		{"a checkout that keeps its token", "          persist-credentials: false\n", "", "persist-credentials: false"},
		{"a checkout that says to keep its token", "persist-credentials: false", "persist-credentials: true", "persist-credentials: false"},

		// The branches, the worktrees and lct.
		{"no raw branch", "+refs/heads/raw:refs/remotes/origin/raw", "+refs/heads/main:refs/remotes/origin/main", "raw branch"},
		{"no tw branch", "+refs/heads/tw:refs/remotes/origin/tw", "+refs/heads/main:refs/remotes/origin/main", "tw branch"},
		{"no raw worktree", "git worktree add .work/raw raw", "git worktree prune", "raw worktree"},
		{"no tw worktree", "git worktree add .work/tw tw", "git worktree prune", "tw worktree"},
		{"worktrees made before the fetch", "          git fetch origin '+refs/heads/raw:refs/remotes/origin/raw' '+refs/heads/tw:refs/remotes/origin/tw'\n          git worktree add .work/raw raw\n",
			"          git worktree add .work/raw raw\n          git fetch origin '+refs/heads/raw:refs/remotes/origin/raw' '+refs/heads/tw:refs/remotes/origin/tw'\n", "adds a worktree before it fetches"},
		{"lct not built", "run: go build -o bin/lct ./cmd/lct", "run: go version", "no step that builds lct"},
		{"no Go", "      - uses: actions/setup-go@v6\n        with:\n          go-version-file: go.mod\n\n", "", "no step that sets up Go"},

		// A step that fails fails the run, and lct site is the whole of its step.
		{"a failure of lct site that is ignored", "run: bin/lct site --out site/public/data --json\n", "run: bin/lct site --out site/public/data --json || true\n", "bin/lct site"},
		{"a failure of lct site that a pipe hides", "run: bin/lct site --out site/public/data --json\n", "run: bin/lct site --out site/public/data --json | tee site.log\n", "bin/lct site"},
		{"a second command after lct site", "run: bin/lct site --out site/public/data --json\n", "run: |\n          bin/lct site --out site/public/data --json\n          true\n", "bin/lct site"},
		{"another output folder", "--out site/public/data", "--out site/public/other", "bin/lct site"},
		{"no json from lct site", " --json\n", "\n", "bin/lct site"},
		{"no lct site", "bin/lct site --out site/public/data --json", "echo site", "no step that runs bin/lct site"},
		{"a step of lct site that continues on error", pagesSiteStep, pagesSiteStep + "        continue-on-error: true\n", "continue-on-error"},
		{"a step that is allowed to fail", pagesDeployStep, pagesDeployStep + "        continue-on-error: ${{ true }}\n", "continue-on-error"},
		{"a deploy that runs after a failure", pagesDeployStep, pagesDeployStep + "        if: always()\n", "has an if"},
		{"a step of lct site that can be skipped", pagesSiteStep, pagesSiteStep + "        if: github.event_name != 'workflow_dispatch'\n", "has an if"},
		{"a job that continues on error", "    timeout-minutes: 30\n", "    timeout-minutes: 30\n    continue-on-error: true\n", "the job has continue-on-error"},
		{"a shell that does not stop", "        shell: bash\n", "        shell: sh {0}\n", "stops at the first command that fails"},
		{"a step with a shell that does not stop", pagesSiteStep, pagesSiteStep + "        shell: python {0}\n", "stops at the first command that fails"},

		// Node and the build, as in ci.yml.
		{"no Node", siteNodeStep, "", "no step that sets up Node"},
		{"no node version file", "node-version-file: site/.nvmrc", "node-version-file: .nvmrc", "node-version-file"},
		{"a node version beside the file", "          cache: npm\n", "          cache: npm\n          node-version: 22\n", "node-version"},
		{"no cache of npm", "          cache: npm\n", "", "cache"},
		{"another cache", "cache: npm\n", "cache: yarn\n", "cache"},
		{"another lock file", "cache-dependency-path: site/package-lock.json", "cache-dependency-path: package-lock.json", "cache-dependency-path"},
		{"no install of the packages", "run: npm ci", "run: npm install", "no step that runs npm ci"},
		{"the install outside the site", "        run: npm ci\n        working-directory: site\n", "        run: npm ci\n", "working-directory"},
		{"the build outside the site", pagesBuildStep, "        run: npm run build\n", "working-directory"},
		{"no build of the site", "run: npm run build", "run: npm run dev", "no step that runs npm run build"},

		// Pages: configure, upload what the build made, deploy.
		{"another artifact", "path: site/dist", "path: site", "path"},
		{"no configuration of Pages", "      - uses: actions/configure-pages@v6\n", "", "no step that configures Pages"},
		{"no upload", pagesUploadStep, "", "no step that uploads the artifact"},
		{"no deploy", pagesDeployStep, "", "no step that deploys the artifact"},
		{"an action that is not pinned", "actions/deploy-pages@v5", "actions/deploy-pages@v4", "actions/deploy-pages@v4"},
		{"another action", "actions/setup-go@v6", "anthropics/claude-code-action@v1", "anthropics/claude-code-action@v1"},
	} {
		t.Run("pages.yml: "+tc.name, func(t *testing.T) {
			src := mutate(t, pages, tc.old, tc.repl)
			hasProblem(t, checkPagesWorkflow(pagesFile, src), pagesFile, tc.want)
		})
	}
	// Each change below is one problem, no more. The first four, with the
	// subtest for data that comes after the build, are the ways the deployment
	// goes wrong that matter most: the id token, a run that is cancelled, an
	// artifact that is not the site, and a secret.
	for _, tc := range []struct {
		name, old, repl string
		want            []string
	}{
		{"no id-token write", ", id-token: write", "", []string{"permissions"}},
		{"a deployment that is cancelled", "cancel-in-progress: false", "cancel-in-progress: true", []string{"concurrency", "cancel-in-progress: true"}},
		{"an artifact that is not the site", "path: site/dist", "path: site", []string{"upload has path = site", "site/dist"}},
		{"a secret", "      - name: Build lct\n        run: go build -o bin/lct ./cmd/lct\n", "      - name: Build lct\n        run: go build -o bin/lct ./cmd/lct\n        env:\n          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}\n", []string{"secret"}},
		{"a failure of lct site that is ignored", "run: bin/lct site --out site/public/data --json\n", "run: bin/lct site --out site/public/data --json || true\n", []string{"bin/lct site", "|| true"}},
		{"a step of lct site that continues on error", pagesSiteStep, pagesSiteStep + "        continue-on-error: true\n", []string{"continue-on-error"}},
		{"a checkout that keeps its token", "          persist-credentials: false\n", "", []string{"persist-credentials: false"}},
		{"another environment", "environment: github-pages", "environment: production", []string{"environment"}},
	} {
		t.Run("pages.yml: exactly one problem for "+tc.name, func(t *testing.T) {
			src := mutate(t, pages, tc.old, tc.repl)
			wantProblems(t, checkPagesWorkflow(pagesFile, src), append([]string{pagesFile}, tc.want...)...)
		})
	}
	t.Run("pages.yml: exactly one problem for data written after the build", func(t *testing.T) {
		// The step of the data moves from before the setup of Node to behind the
		// build of the site.
		src := mutate(t, mutate(t, pages, pagesSiteStep, ""), pagesBuildStep, pagesBuildStep+"\n"+pagesSiteStep)
		wantProblems(t, checkPagesWorkflow(pagesFile, src), pagesFile,
			"the step that runs bin/lct site must come before the step that runs npm run build")
	})
	t.Run("pages.yml: exactly one problem for an upload after the deploy", func(t *testing.T) {
		src := mutate(t, mutate(t, pages, pagesUploadStep, ""), pagesDeployStep, pagesDeployStep+pagesUploadStep)
		wantProblems(t, checkPagesWorkflow(pagesFile, src), pagesFile,
			"the step that uploads the artifact must come before the step that deploys the artifact")
	})
	t.Run("pages.yml: exactly one problem for a build after the upload", func(t *testing.T) {
		src := mutate(t, mutate(t, pages, "      - name: Build the site\n"+pagesBuildStep, ""), pagesUploadStep, pagesUploadStep+"\n      - name: Build the site\n"+pagesBuildStep)
		wantProblems(t, checkPagesWorkflow(pagesFile, src), pagesFile,
			"the step that runs npm run build must come before the step that uploads the artifact")
	})
	t.Run("a workflow with the wrong shape", func(t *testing.T) {
		wantProblems(t, checkPagesWorkflow(pagesFile, "name: pages\non: [workflow_dispatch]\n"), "has no job")
	})
	t.Run("a second job", func(t *testing.T) {
		src := pages + "  other:\n    runs-on: ubuntu-24.04\n    steps:\n      - run: echo other\n"
		hasProblem(t, checkPagesWorkflow(pagesFile, src), pagesFile, "2 jobs, want one")
	})
	t.Run("an action that is not pinned in pages.yml", func(t *testing.T) {
		for _, tc := range []struct{ old, repl string }{
			{"actions/checkout@v5", "actions/checkout@main"},
			{"actions/setup-go@v6", "actions/setup-go@v5"},
			{"actions/setup-node@v7", "actions/setup-node@v6"},
			{"actions/configure-pages@v6", "actions/configure-pages@v5"},
			{"actions/upload-pages-artifact@v5", "actions/upload-pages-artifact@v4"},
			{"actions/deploy-pages@v5", "actions/deploy-pages@v4"},
		} {
			src := mutate(t, pages, tc.old, tc.repl)
			wantProblems(t, checkWorkflow(pagesFile, src, commands), tc.repl)
			hasProblem(t, checkPagesWorkflow(pagesFile, src), pagesFile, tc.repl)
		}
	})
	t.Run("a runner that is not pinned in pages.yml", func(t *testing.T) {
		wantProblems(t, checkWorkflow(pagesFile, mutate(t, pages, "runs-on: ubuntu-24.04", "runs-on: ubuntu-latest"), commands), "ubuntu-latest")
	})
	t.Run("an unregistered lct command in pages.yml", func(t *testing.T) {
		src := mutate(t, pages, "bin/lct site --out", "bin/lct sitee --out")
		hasProblem(t, checkWorkflow(pagesFile, src, commands), `"bin/lct sitee"`)
	})
	t.Run("an expression in a script of pages.yml", func(t *testing.T) {
		src := mutate(t, pages, "run: npm ci\n", "run: npm ci --prefix ${{ github.workspace }}\n")
		hasProblem(t, checkWorkflow(pagesFile, src, commands), "expression")
	})
	t.Run("a model name in pages.yml", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, root, workflowDir+"/pages.yml", pages+"# claude-example-9\n")
		wantProblems(t, checkModelNames(root, workflowDir), "pages.yml", `"claude-example-9"`)
	})
	t.Run("an environment with a name and an address", func(t *testing.T) {
		src := mutate(t, pages, "environment: github-pages", "environment:\n      name: github-pages\n      url: ${{ steps.deployment.outputs.page_url }}")
		if problems := checkPagesWorkflow(pagesFile, src); len(problems) != 0 {
			t.Errorf("an environment that names github-pages and has an address is a problem: %v", problems)
		}
	})

	// 14. ci.yml tests and builds the viewer in a job of its own, with Node as
	// pages.yml sets it up, and no failure passes.
	for _, tc := range []struct{ name, old, repl, want string }{
		{"no job for the viewer", "\n  site:\n", "\n  viewer:\n", "no job site"},
		{"a job for the viewer that can be skipped", "  site:\n    runs-on: ubuntu-24.04\n", "  site:\n    if: github.event_name == 'push'\n    runs-on: ubuntu-24.04\n", "the job site has an if"},
		{"no checkout", "  site:\n    runs-on: ubuntu-24.04\n    steps:\n      - uses: actions/checkout@v5\n", "  site:\n    runs-on: ubuntu-24.04\n    steps:\n      - run: echo site\n", "no actions/checkout"},
		{"no Node", siteNodeStep, "", "no actions/setup-node step"},
		{"another Node version file", "node-version-file: site/.nvmrc", "node-version-file: .nvmrc", "node-version-file"},
		{"a Node version beside the file", "          cache: npm\n", "          cache: npm\n          node-version: 22\n", "node-version"},
		{"another cache", "cache: npm\n", "cache: yarn\n", "cache"},
		{"another lock file", "cache-dependency-path: site/package-lock.json", "cache-dependency-path: package-lock.json", "cache-dependency-path"},
		{"no install of the packages", "run: npm ci\n", "run: npm install\n", "no step that runs npm ci"},
		{"no type check", "run: npm run typecheck\n", "run: npm run lint\n", "no step that runs npm run typecheck"},
		{"no tests", "run: npm test\n", "run: npm run test:watch\n", "no step that runs npm test"},
		{"no build", "run: npm run build\n", "run: npm run dev\n", "no step that runs npm run build"},
		{"tests outside the site", "        run: npm test\n        working-directory: site\n", "        run: npm test\n", "working-directory"},
		{"a build outside the site", "        run: npm run build\n        working-directory: site\n", "        run: npm run build\n", "working-directory"},
		{"tests whose failure passes", "        run: npm test\n", "        run: npm test\n        continue-on-error: true\n", "continue-on-error"},
		{"a build that can be skipped", "        run: npm run build\n", "        run: npm run build\n        if: github.event_name == 'push'\n", "has an if"},
		{"a job that continues on error", "  site:\n    runs-on: ubuntu-24.04\n", "  site:\n    runs-on: ubuntu-24.04\n    continue-on-error: true\n", "the job has continue-on-error"},
	} {
		t.Run("ci.yml site job: "+tc.name, func(t *testing.T) {
			src := mutate(t, ci, tc.old, tc.repl)
			hasProblem(t, checkCIWorkflow(ciFile, src), ciFile, tc.want)
		})
	}
	t.Run("the tests behind the build", func(t *testing.T) {
		// The tests move from before the build to behind it, the last step.
		test := "      - name: Test\n        run: npm test\n        working-directory: site\n"
		src := mutate(t, ci, test, "") + test
		hasProblem(t, checkCIWorkflow(ciFile, src), ciFile, "must run in this order")
	})
	t.Run("the install behind the tests", func(t *testing.T) {
		install := "      - name: Install the packages\n        run: npm ci\n        working-directory: site\n"
		src := mutate(t, ci, install, "") + install
		hasProblem(t, checkCIWorkflow(ciFile, src), ciFile, "must run in this order")
	})
	t.Run("a setup of Node behind the install", func(t *testing.T) {
		src := mutate(t, ci, siteNodeStep, "") + siteNodeStep
		hasProblem(t, checkCIWorkflow(ciFile, src), ciFile, "must run in this order")
	})
}

// readmeRulesPR are what the README must tell a person about the rules PR, with
// the reason: how to let the workflow open it, and what to do with one.
var readmeRulesPR = []struct{ text, why string }{
	{"https://github.com/apps/claude", "the Claude GitHub App, which opens the PR as itself and so starts ci.yml on it, is not said to be installed"},
	{"`rules-proposal.yml`", "the workflow that opens the rules PR is not described"},
	{"`rules-proposal`", "the label and the branches of the rules PR are not named"},
	{"`rules/declined.yaml`", "a person is not told where to move the suggestion they do not want"},
	{"`<!-- suggestion:", "a person is not told not to take the markers out of the description"},
	{"影響評估", "the impact assessment ci.yml shows on a rules PR is not described"},
	{"`bin/lct impact`", "the command of the impact assessment is not named"},
	{"`persist-credentials: false`", "the README does not say that the checkout keeps no token, and how to revert it if the action cannot push without it"},
	{"`lct proposal publish`", "the README does not say what to do when lct proposal publish finds no token inside the action"},
	{"`gh needs a token`", "the README does not name the refusal of lct proposal publish that calls for the fallback"},
	{"`GH_TOKEN: ${{ github.token }}`", "the README does not give the step of the fallback that hands the token of the workflow to the action"},
	{"Allow GitHub Actions to create and approve pull requests", "the README does not say that a PR opened with the token of the workflow needs this setting"},
	{"`TestWorkflowContract`", "the README does not say that the contract test holds the workflow to the setting the fallback changes"},
	{"最多處理 8 個建議", "the README does not say that a run handles at most 8 suggestions and leaves the rest for the next"},
	{"需要人決定", "the README does not say what a person does with the items the skill was not sure of"},
	{"移到「需要人決定」標題之上", "the README does not say that an adopted item must leave the section for a person to decide, or lct proposal start suggests it again after the merge"},
}

// The README tells a person how to set up and handle the rules PR.
// decideOutputs runs the script of the decide step of pipeline.yml as the
// runner does (bash -eo pipefail) with env, in a scratch folder with the files
// and the commands it reads (a raw worktree and a tw worktree built from the
// same snapshot, and an lct and a git that say nothing is due), and returns
// what it wrote to $GITHUB_OUTPUT and what it printed.
func decideOutputs(t *testing.T, script string, env map[string]string) (outputs map[string]string, printed string) {
	t.Helper()
	dir := t.TempDir()
	writeTestFile(t, dir, ".work/raw/meta/source.json", `{"tag":"raw-20261105"}`)
	writeTestFile(t, dir, ".work/tw/.meta/build.json", `{"raw_tag":"raw-20261105"}`)
	writeTestFile(t, dir, "bin/lct", "#!/bin/sh\necho '{\"release_needed\":false}'\n")
	writeTestFile(t, dir, "stub/git", "#!/bin/sh\necho 'build: raw-20261105'\n")
	for _, name := range []string{"bin/lct", "stub/git"} {
		if err := os.Chmod(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeTestFile(t, dir, "step.sh", script)
	out := filepath.Join(dir, "github-output")
	cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "step.sh")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PATH="+filepath.Join(dir, "stub")+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GITHUB_OUTPUT="+out, "GITHUB_STEP_SUMMARY="+filepath.Join(dir, "summary.md"))
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	printedBytes, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the decide step failed: %v\n%s", err, printedBytes)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	outputs = map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		k, v, _ := strings.Cut(line, "=")
		outputs[k] = v
	}
	return outputs, string(printedBytes)
}

// The step that records a translation that did not run says why: the key or
// the model is missing when decide says not_run, and else the claude CLI did
// not install. This runs its script with a stand-in lct.
func TestPipelineRecordsWhyTheTranslationDidNotRun(t *testing.T) {
	_, pipeline := readRepoFile(t, filepath.Join(repoRoot, filepath.FromSlash(workflowPath("pipeline.yml"))))
	w, problems := parseWorkflow("pipeline.yml", pipeline)
	if w == nil {
		t.Fatal(problems)
	}
	var script string
	for _, s := range w.steps() {
		if s.If == notRunIf {
			script = s.Run
		}
	}
	if script == "" {
		t.Fatalf("no step runs if %s", notRunIf)
	}
	for notRun, want := range map[string]string{"true": "缺少 ANTHROPIC_API_KEY 或 LCT_MODEL", "false": "安裝 Claude Code CLI 失敗"} {
		dir := t.TempDir()
		writeTestFile(t, dir, "bin/lct", "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$PWD/args\"\n")
		if err := os.Chmod(filepath.Join(dir, "bin", "lct"), 0o755); err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, dir, "step.sh", script)
		cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "step.sh")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "NOT_RUN="+notRun)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("NOT_RUN=%s: %v\n%s", notRun, err, out)
		}
		args := strings.Split(strings.TrimSpace(string(readFile(t, filepath.Join(dir, "args")))), "\n")
		if len(args) != 3 || args[0] != "translate" || args[1] != "--not-run" || !strings.Contains(args[2], want) {
			t.Errorf("NOT_RUN=%s: lct %q, want translate --not-run and a reason with %q", notRun, args, want)
		}
	}
}

// What the decide step does with units that are pending: translates them, or
// leaves them on purpose (a dry run), or records that the key or the model is
// missing. It is run for real, because a test of its text cannot tell that a
// test in it works under the -e of the runner.
func TestPipelineDecidesWhatToDoWithPendingUnits(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is not installed")
	}
	_, pipeline := readRepoFile(t, filepath.Join(repoRoot, filepath.FromSlash(workflowPath("pipeline.yml"))))
	w, problems := parseWorkflow("pipeline.yml", pipeline)
	if w == nil {
		t.Fatal(problems)
	}
	var script string
	for _, s := range w.steps() {
		if s.ID == "decide" {
			script = s.Run
		}
	}
	if script == "" {
		t.Fatal("no step with the id decide")
	}
	// A rebuild asks for no game update, so it is never idle.
	base := map[string]string{
		"FETCH": "false", "FORCE": "false", "REQUEUE": "", "DRY_RUN": "false", "NEW": "", "APPLIED": "0",
		"PENDING": "7", "HAS_API_KEY": "true", "HAS_MODEL": "true",
	}
	for _, tc := range []struct {
		name                    string
		change                  map[string]string
		translate, notRun       string
		warns, doesNotWarnAbout []string
	}{
		{"a key and a model", nil, "true", "false", nil, []string{"ANTHROPIC_API_KEY", "LCT_MODEL"}},
		{"no key", map[string]string{"HAS_API_KEY": "false"}, "false", "true", []string{"::warning::ANTHROPIC_API_KEY is not set"}, []string{"LCT_MODEL"}},
		{"no model", map[string]string{"HAS_MODEL": "false"}, "false", "true", []string{"::warning::the variable LCT_MODEL is not set"}, []string{"ANTHROPIC_API_KEY"}},
		{"neither", map[string]string{"HAS_API_KEY": "false", "HAS_MODEL": "false"}, "false", "true", []string{"ANTHROPIC_API_KEY is not set", "LCT_MODEL is not set"}, nil},
		{"a dry run with a key and a model", map[string]string{"DRY_RUN": "true"}, "false", "false", []string{"::notice::dry_run"}, nil},
		{"a dry run with neither", map[string]string{"DRY_RUN": "true", "HAS_API_KEY": "false", "HAS_MODEL": "false"}, "false", "false", []string{"::notice::dry_run"}, []string{"::warning::"}},
		{"nothing pending, no key", map[string]string{"PENDING": "0", "HAS_API_KEY": "false"}, "false", "false", nil, []string{"::warning::"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := maps.Clone(base)
			maps.Copy(env, tc.change)
			got, printed := decideOutputs(t, script, env)
			if got["idle"] != "false" || got["translate"] != tc.translate || got["not_run"] != tc.notRun {
				t.Errorf("outputs = %v, want idle=false translate=%s not_run=%s", got, tc.translate, tc.notRun)
			}
			for _, want := range tc.warns {
				if !strings.Contains(printed, want) {
					t.Errorf("the step printed %q, want %q", printed, want)
				}
			}
			for _, no := range tc.doesNotWarnAbout {
				if strings.Contains(printed, no) {
					t.Errorf("the step printed %q, which names %q", printed, no)
				}
			}
		})
	}
}

func TestReadmeExplainsTheRulesPR(t *testing.T) {
	_, readme := readRepoFile(t, filepath.Join(repoRoot, "README.md"))
	for _, s := range readmeRulesPR {
		if !strings.Contains(readme, s.text) {
			t.Errorf("README.md does not say %q: %s", s.text, s.why)
		}
	}
	// The table is under a heading of its own, up to the next heading.
	lines := strings.Split(readme, "\n")
	start := slices.Index(lines, "### "+readmeWorkflows)
	if start < 0 {
		t.Fatalf("README.md: no section %q", "### "+readmeWorkflows)
	}
	var rows []string
	for _, line := range lines[start+1:] {
		if strings.HasPrefix(line, "#") {
			break
		}
		if strings.HasPrefix(line, "|") {
			rows = append(rows, line)
		}
	}
	for _, name := range []string{"update.yml", "rebuild.yml", "rules-proposal.yml", "pages.yml", "ci.yml"} {
		if !slices.ContainsFunc(rows, func(row string) bool { return strings.Contains(row, "`"+name+"`") }) {
			t.Errorf("README.md: no table row of the section %q for %s", readmeWorkflows, name)
		}
	}
}

// readmeSite is the heading of the README section about the site.
const readmeSite = "網站"

// readmeSiteDoc are what the section must tell a person, with the reason: where
// the site is, and the setting that nothing in the repo can make for them.
var readmeSiteDoc = []struct{ text, why string }{
	{"https://tool-jx3.github.io/limbus-translate/", "the address of the site is not given"},
	{"Settings → Pages", "the one-time setting of Pages is not placed in the settings of the repo"},
	{"「GitHub Actions」", "the Source of Pages is not said to be GitHub Actions: without it pages.yml cannot configure Pages or deploy"},
	{"`pages.yml`", "the workflow that deploys the site is not named"},
}

// checkReadmeSite reports what the section "## 網站" of readme does not say.
func checkReadmeSite(readme string) []string {
	section, ok := readmeSection(readme, readmeSite)
	if !ok {
		return []string{fmt.Sprintf("README.md: no section %q", "## "+readmeSite)}
	}
	var problems []string
	for _, s := range readmeSiteDoc {
		if !strings.Contains(section, s.text) {
			problems = append(problems, fmt.Sprintf("the section %q of README.md does not say %q: %s", readmeSite, s.text, s.why))
		}
	}
	return problems
}

// The README tells a person where the site is and the setting of Pages to make
// once.
func TestWorkflowContractDocumentsTheSite(t *testing.T) {
	_, readme := readRepoFile(t, filepath.Join(repoRoot, "README.md"))
	for _, problem := range checkReadmeSite(readme) {
		t.Error(problem)
	}
	if t.Failed() {
		return
	}
	for _, tc := range []struct{ name, old, repl, want string }{
		{"no address", "https://tool-jx3.github.io/limbus-translate/", "", "the address of the site"},
		{"no place for the setting", "Settings → Pages", "Settings", "Settings → Pages"},
		{"no Source of GitHub Actions", "「GitHub Actions」", "「Actions」", "「GitHub Actions」"},
		{"no workflow", "`pages.yml`", "pages", "`pages.yml`"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			section, _ := readmeSection(readme, readmeSite)
			// Every copy goes, in the section only: the rest of the README may say the same.
			problems := checkReadmeSite(strings.Replace(readme, section, strings.ReplaceAll(section, tc.old, tc.repl), 1))
			if len(problems) != 1 || !strings.Contains(problems[0], tc.want) {
				t.Errorf("problems = %q, want one that holds %q", problems, tc.want)
			}
		})
	}
	t.Run("no section", func(t *testing.T) {
		problems := checkReadmeSite(strings.Replace(readme, "## "+readmeSite+"\n", "## 其他\n", 1))
		if len(problems) != 1 || !strings.Contains(problems[0], "no section") {
			t.Errorf("problems = %q, want one that says there is no section", problems)
		}
	})
}

// outcomeStep returns the script of the step of rules-proposal.yml that checks
// what the run did: it reads outcome.txt and looks for the rules PR.
func outcomeStep(t *testing.T, workflow string) string {
	t.Helper()
	w, problems := parseWorkflow("rules-proposal.yml", workflow)
	if w == nil {
		t.Fatal(problems)
	}
	var found []string
	for _, s := range w.steps() {
		if strings.Contains(s.Run, "outcome.txt") && strings.Contains(s.Run, "gh pr list") {
			found = append(found, s.Run)
		}
	}
	if len(found) != 1 {
		t.Fatalf("%d steps read outcome.txt and look for the PR, want one", len(found))
	}
	return found[0]
}

// stepResult is what a run of the outcome step did.
type stepResult struct {
	exit            int
	output, summary string
	ghCalled        bool
}

// runOutcomeStep runs the script of the step in a scratch folder as the runner
// does (bash -eo pipefail), with work/proposal/exclude.txt as given, the file
// outcome.txt when outcome is not nil, and a gh that answers the list of pull
// requests with prs (JSON) and records that it was called. BRANCH is the branch
// of the run.
func runOutcomeStep(t *testing.T, script string, outcome *string, exclude, prs string) stepResult {
	t.Helper()
	dir := t.TempDir()
	writeTestFile(t, dir, "work/proposal/exclude.txt", exclude)
	if outcome != nil {
		writeTestFile(t, dir, "work/proposal/outcome.txt", *outcome)
	}
	writeTestFile(t, dir, "prs.json", prs)
	writeTestFile(t, dir, "bin/gh", "#!/bin/sh\ntouch \"$PWD/gh-called\"\ncat \"$PWD/prs.json\"\n")
	if err := os.Chmod(filepath.Join(dir, "bin", "gh"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, dir, "step.sh", script)
	summary := filepath.Join(dir, "summary.md")
	cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "step.sh")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PATH="+filepath.Join(dir, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GH_TOKEN=test-token", "BRANCH=rules-proposal/2026-11-05", "GITHUB_STEP_SUMMARY="+summary)
	out, err := cmd.CombinedOutput()
	res := stepResult{output: string(out)}
	var exitErr *exec.ExitError
	switch {
	case errors.As(err, &exitErr):
		res.exit = exitErr.ExitCode()
	case err != nil:
		t.Fatal(err)
	}
	if data, err := os.ReadFile(summary); err == nil {
		res.summary = string(data)
	}
	_, statErr := os.Stat(filepath.Join(dir, "gh-called"))
	res.ghCalled = statErr == nil
	return res
}

// The step that ends the run of rules-proposal.yml turns what the skill said
// it did, and what the rules PR holds, into green, a notice or an error. This
// runs the script, because what matters is what it does: when lct proposal
// publish refuses, the action does not fail, and a rules PR that is open and
// waiting for review is the usual state (spec 11.2).
func TestRulesProposalOutcomeStep(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is not installed")
	}
	_, workflow := readRepoFile(t, filepath.Join(repoRoot, filepath.FromSlash(workflowPath("rules-proposal.yml"))))
	script := outcomeStep(t, workflow)

	const (
		url = "https://github.com/o/r/pull/7"
		// The ids lct proposal start excluded: the open PR has the first, the
		// second was declined. Ids hold spaces, CJK text and |.
		exclude = "term/old one\ninconsistent/story/content/ab12cd34 한글|x\n"
	)
	pr := func(body string, cross bool) string {
		data, err := json.Marshal([]map[string]any{{"url": url, "body": body, "isCrossRepository": cross}})
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	marker := func(id string) string { return "<!-- suggestion:" + id + " -->" }
	str := func(s string) *string { return &s }
	old := marker("term/old one") + "\n### old\n\n"

	for _, tc := range []struct {
		name    string
		outcome *string
		prs     string
		exit    int
		ghCall  bool
		output  []string // all in the output
		summary []string // all in the summary
	}{
		{"published a new suggestion", str("published\n"), pr("intro\n\n"+old+marker("speaker/새 화자")+"\n### new\n", false), 0, true, []string{url}, []string{url}},
		{"published a new suggestion in a description from the web page", str("published"), pr(strings.ReplaceAll("intro\n\n"+old+marker("note/k/12345678")+"\n### new\n", "\n", "\r\n"), false), 0, true, []string{url}, []string{url}},
		{"published a new suggestion whose id is like an old one", str("published"), pr(old+marker("term/old one 2")+"\n", false), 0, true, nil, nil},
		// An item the skill was not sure of has its marker too: a run that
		// only lists such items has added to the PR.
		{"published with an item for a person to decide only", str("published"), pr(old+"## 需要人決定\n\n"+marker("term/新 詞")+"\n### 詞彙候選\n\n證據不足。\n", false), 0, true, []string{url}, []string{url, "1 個建議"}},
		{"published with the old suggestions only", str("published"), pr("intro\n\n"+old+marker("inconsistent/story/content/ab12cd34 한글|x")+"\n", false), 1, true, []string{"::error::", "added nothing", url}, nil},
		{"published with a description that has no marker", str("published"), pr("a description a person rewrote\n", false), 1, true, []string{"::error::", "added nothing"}, nil},
		{"published with no rules PR open", str("published"), "[]", 1, true, []string{"::error::", "no open rules PR", "rules-proposal/2026-11-05"}, nil},
		{"published and only a PR from a fork", str("published"), pr(marker("speaker/새 화자")+"\n", true), 1, true, []string{"::error::", "no open rules PR"}, nil},
		{"nothing proposed", str("nothing: 沒有有把握的項目\n"), "[]", 0, false, []string{"::notice::", "沒有有把握的項目"}, []string{"沒有有把握的項目"}},
		{"nothing proposed while a rules PR is open", str("nothing: 規則檔沒有任何改動"), pr(old, false), 0, false, []string{"::notice::", "規則檔沒有任何改動"}, nil},
		{"nothing proposed, a reason of several lines", str("nothing: first line\n::error::second line\n"), "[]", 0, false, []string{"::notice::", "first line"}, nil},
		// The skill took the run for one on a person's machine: it found no
		// work/proposal/started.json and published nothing.
		{"nothing proposed by a run that took itself for a local one", str("nothing: 本機執行，沒有發布\n"), pr(old, false), 1, false, []string{"::error::", "本機執行，沒有發布"}, nil},
		{"no outcome", nil, pr(old+marker("speaker/새 화자")+"\n", false), 1, false, []string{"::error::", "work/proposal/outcome.txt"}, nil},
		{"an outcome that is a failure", str("failed: lct proposal publish: gh needs a token\n"), pr(old, false), 1, false, []string{"::error::", "failed: lct proposal publish: gh needs a token"}, nil},
		{"an outcome that says nothing", str("\n"), pr(old, false), 1, false, []string{"::error::"}, nil},
		{"an outcome of another kind", str("done"), pr(old, false), 1, false, []string{"::error::", "done"}, nil},
		{"nothing without a reason", str("nothing:"), "[]", 1, false, []string{"::error::"}, nil},
		{"published and something more", str("published, and more"), pr(old+marker("speaker/새 화자")+"\n", false), 1, false, []string{"::error::"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := runOutcomeStep(t, script, tc.outcome, exclude, tc.prs)
			if got.exit != tc.exit {
				t.Errorf("exit = %d, want %d\n%s", got.exit, tc.exit, got.output)
			}
			if got.ghCalled != tc.ghCall {
				t.Errorf("gh called = %v, want %v", got.ghCalled, tc.ghCall)
			}
			for _, part := range tc.output {
				if !strings.Contains(got.output, part) {
					t.Errorf("output does not hold %q:\n%s", part, got.output)
				}
			}
			for _, part := range tc.summary {
				if !strings.Contains(got.summary, part) {
					t.Errorf("summary does not hold %q:\n%s", part, got.summary)
				}
			}
			// A reason of several lines is one line of the log: the next line
			// is not a command of the runner.
			if strings.Contains(got.output, "\n::error::second line") {
				t.Errorf("a line of the reason is a command of the runner:\n%s", got.output)
			}
		})
	}
}

// Without LCT_MODEL the action does not run, and the step that stands in for it
// warns and writes an outcome of nothing: the check of the outcome, run after
// it, stays green with a notice and asks gh for nothing.
func TestRulesProposalWithoutAModel(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is not installed")
	}
	_, workflow := readRepoFile(t, filepath.Join(repoRoot, filepath.FromSlash(workflowPath("rules-proposal.yml"))))
	w, problems := parseWorkflow("rules-proposal.yml", workflow)
	if w == nil {
		t.Fatal(problems)
	}
	var script string
	for _, s := range w.steps() {
		if strings.Contains(s.If, rulesProposalNoModel) {
			script = s.Run
		}
	}
	if script == "" {
		t.Fatalf("no step runs if %s", rulesProposalNoModel)
	}
	dir := t.TempDir()
	writeTestFile(t, dir, "work/proposal/exclude.txt", "")
	writeTestFile(t, dir, "step.sh", script)
	cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "step.sh")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "::warning::") || !strings.Contains(string(out), "LCT_MODEL") {
		t.Fatalf("the step: %v\n%s", err, out)
	}
	outcome := string(readFile(t, filepath.Join(dir, "work", "proposal", "outcome.txt")))

	got := runOutcomeStep(t, outcomeStep(t, workflow), &outcome, "", "[]")
	if got.exit != 0 || got.ghCalled || !strings.Contains(got.output, "::notice::") || !strings.Contains(got.summary, "LCT_MODEL") {
		t.Errorf("the check of the outcome %q: exit %d, gh called %v\n%s\nsummary:\n%s", outcome, got.exit, got.ghCalled, got.output, got.summary)
	}
}

// The jq commands of pipeline.yml read the keys the lct commands print: a key
// renamed in the code must fail here, not in a run of the workflow.
func TestWorkflowReadsLctJSON(t *testing.T) {
	file := workflowPath("pipeline.yml")
	_, pipeline := readRepoFile(t, filepath.Join(repoRoot, filepath.FromSlash(file)))
	inputs := lctJSONInputs()
	for _, problem := range checkJQKeys(file, pipeline, inputs, lctJSONRequired) {
		t.Error(problem)
	}
	for _, problem := range checkJQForbidden(file, pipeline, lctJSONForbidden) {
		t.Error(problem)
	}
	if t.Failed() {
		return
	}
	t.Run("the tag of the Release check read", func(t *testing.T) {
		src := mutate(t, pipeline, "          release_needed=$(jq -er '.release_needed | tostring' <<<\"$release_check\")\n",
			"          release_needed=$(jq -er '.release_needed | tostring' <<<\"$release_check\")\n          tag=$(jq -r .tag <<<\"$release_check\")\n")
		wantProblems(t, checkJQForbidden(file, src, lctJSONForbidden), `reads "tag" from release_check`)
	})

	for _, tc := range []struct{ name, old, repl string }{
		{"a key of publish", "jq -er .tw_commit", "jq -er .tw_sha"},
		{"a key of plan", "jq -er .pending <<<\"$plan\"", "jq -er .pendng <<<\"$plan\""},
		{"a key of the check", "jq -er .created_at <<<\"$check\"", "jq -er .createdat <<<\"$check\""},
		{"a key of the snapshot", "jq -er .tag <<<\"$snapshot\"", "jq -er .name <<<\"$snapshot\""},
		{"a key of the pending count", "jq -er .available", "jq -er .free"},
		{"a key of build", "jq -er .review_issue_parts", "jq -er .issue_parts"},
		{"a key of the last build", "jq -r '.raw_tag // empty'", "jq -r '.rawtag // empty'"},
		{"a key of the Release check", "jq -er '.release_needed | tostring' <<<\"$release_check\"", "jq -er '.release_due | tostring' <<<\"$release_check\""},
		{"the pack a Release is skipped for", "jq -r '.same_as // empty'", "jq -r '.sameas // empty'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := mutate(t, pipeline, tc.old, tc.repl)
			problems := checkJQKeys(file, src, inputs, lctJSONRequired)
			if len(problems) != 2 {
				t.Fatalf("problems = %q, want two: the old key is not read, the new one is not printed", problems)
			}
			if !strings.Contains(strings.Join(problems, "\n"), "does not print") || !strings.Contains(strings.Join(problems, "\n"), "does not read") {
				t.Errorf("problems = %q, want one about a key that is not read and one about a key that is not printed", problems)
			}
		})
	}
	t.Run("a key of a result that was renamed", func(t *testing.T) {
		lost := map[string]any{}
		for k, v := range inputs {
			lost[k] = v
		}
		lost["out/publish.json"] = struct {
			Tag    string   `json:"tag"`
			Assets []string `json:"assets"`
		}{}
		problems := checkJQKeys(file, pipeline, lost, lctJSONRequired)
		hasProblem(t, problems, `reads "tw_commit"`)
		hasProblem(t, problems, `reads "release_needed"`)
		hasProblem(t, problems, `reads "same_as"`)
	})
	t.Run("a key of the Release check that was renamed", func(t *testing.T) {
		lost := maps.Clone(inputs)
		lost["release_check"] = struct {
			TWCommit string `json:"tw_commit"`
		}{}
		hasProblem(t, checkJQKeys(file, pipeline, lost, lctJSONRequired), `reads "release_needed"`, "release_check")
	})
	t.Run("jq -e on a boolean", func(t *testing.T) {
		src := mutate(t, pipeline, "jq -er '.release_needed | tostring' out/publish.json", "jq -er .release_needed out/publish.json")
		wantProblems(t, checkJQKeys(file, src, inputs, lctJSONRequired), "tostring")
		src = mutate(t, pipeline, "jq -er '.release_needed | tostring' <<<\"$release_check\"", "jq -er .release_needed <<<\"$release_check\"")
		wantProblems(t, checkJQKeys(file, src, inputs, lctJSONRequired), "tostring")
		src = mutate(t, pipeline, "jq -er '.new | tostring'", "jq -er .new")
		wantProblems(t, checkJQKeys(file, src, inputs, lctJSONRequired), "tostring")
	})
	t.Run("keys read from an output the contract does not know", func(t *testing.T) {
		src := mutateAll(t, pipeline, "jq -er .tag .work/raw/meta/source.json", "jq -er .tag .work/raw/meta/other.json")
		problems := checkJQKeys(file, src, inputs, lctJSONRequired)
		hasProblem(t, problems, "other.json", "does not know")
	})
}

// A missing glossary rendering is rejected at submission (term-missing); the
// exception is a note that names the Korean term. The translator is told so
// by the shared prompt and by the translate-pending skill, which also fixes
// the rejected units.
func TestTermMissingExceptionIsExplained(t *testing.T) {
	_, prompt := readRepoFile(t, filepath.Join(repoRoot, "prompts", "translate.md"))
	_, skill := readRepoFile(t, filepath.Join(repoRoot, ".claude", "skills", "translate-pending", "SKILL.md"))
	for name, text := range map[string]string{"prompts/translate.md": prompt, "translate-pending/SKILL.md": skill} {
		for _, want := range []string{"term-missing", "`note`", "시즌"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s does not mention %s (the glossary rendering is held to, and a note names the exception)", name, want)
			}
		}
	}
}
