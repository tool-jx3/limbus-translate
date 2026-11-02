package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/impact"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/rules"
)

func init() {
	cli.Register(cli.Command{
		Name: "impact",
		Usage: "count the translations that the rules of the working tree reject and the rules of a revision do not " +
			"(guides/, glossary/ and rules/ of REV are the old rules): --base REV [--markdown FILE] (FILE inside work/ or out/ of the repo)",
		Run: runImpact,
	})
}

// baseRulesPaths are the directories whose files are the rules rules.Load reads.
var baseRulesPaths = []string{"guides", "glossary", "rules"}

// runImpact is the assessment of a rule change (spec 11.2 and 12): the old
// rules are those of the revision, the new rules those of the working tree,
// and the translations are those of the tw worktree. A change that breaks
// translations is no finding: it is the result, and the exit code is 0.
func runImpact(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("impact", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	base := flags.String("base", "", "the revision whose guides/, glossary/ and rules/ are the old rules, such as origin/main (required)")
	markdown := flags.String("markdown", "", "also write the result as Markdown, for a pull request or a summary page, to this file, which must be inside work/ or out/ of the repo")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("impact: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("impact: unexpected argument " + flags.Arg(0))
	}
	switch {
	case *base == "":
		return cli.UserError("impact: --base is required: the revision whose rules the working tree is compared with")
	case strings.HasPrefix(*base, "-"):
		return cli.UserError(fmt.Sprintf("impact: --base %s looks like an option, not a revision", *base))
	}
	mdPath := ""
	if *markdown != "" {
		var err error
		if mdPath, err = outputPath(ctx.Root, *markdown, markdownDirs); err != nil {
			return cli.UserError(fmt.Sprintf("impact: --markdown %s: %v", *markdown, err))
		}
	}
	cfg := ctx.Config
	if cfg.Worktrees.Raw == "" || cfg.Worktrees.TW == "" {
		return cli.UserError("impact: worktrees.raw and worktrees.tw must be set in lct.yaml")
	}

	rawWorktree := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.Raw))
	src, err := raw.LastSource(rawWorktree)
	if err != nil {
		return err
	}
	if src == nil {
		return cli.UserError(fmt.Sprintf("impact: %s has no snapshot; make the worktree of the raw branch (git worktree add %s raw)", cfg.Worktrees.Raw, cfg.Worktrees.Raw))
	}
	set, err := pack.OpenRaw(rawWorktree)
	if err != nil {
		return err
	}
	st, err := openTWStore(ctx, "impact")
	if err != nil {
		return err
	}
	records, err := st.LoadRecords()
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return cli.UserError(fmt.Sprintf("impact: %s has no translations (.meta/units/ is empty)", cfg.Worktrees.TW))
	}

	newRules, err := rules.Load(ctx.Root)
	if err != nil {
		return err
	}
	oldRules, err := loadRulesAt(ctx.Root, *base)
	if err != nil {
		return err
	}

	res, err := impact.Run(impact.Inputs{Old: oldRules, New: newRules, Records: records, KR: set.KR, Frozen: cfg.Frozen})
	if err != nil {
		return err
	}
	if mdPath != "" {
		if err := os.MkdirAll(filepath.Dir(mdPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(mdPath, []byte(res.Markdown(*base)), 0o644); err != nil {
			return err
		}
	}
	return ctx.Emit(res)
}

// markdownDirs are the folders of the repo lct impact --markdown may write
// in: the rules-review skill writes work/proposal/impact.md, ci.yml
// out/impact.md. The skill runs lct impact with text in its context that
// anyone can write, so the file must not land on a rule, on .git/ or outside
// the repo.
var markdownDirs = []string{"work", "out"}

// outputPath returns where a file given as p (relative to the working
// directory) really is, with ".." and symbolic links resolved, if that is
// inside one of dirs (folders of the repo at root, which are not followed when
// they are links themselves); otherwise the error says why not, for a person.
// A link that points nowhere is refused: a write would create its target,
// wherever that is.
func outputPath(root, p string, dirs []string) (string, error) {
	wanted := strings.Join(dirs, "/ or ") + "/"
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	real, err := resolveExisting(abs)
	if errors.Is(err, errDanglingLink) {
		return "", fmt.Errorf("%w; the file must be inside %s of the repo", err, wanted)
	}
	if err != nil {
		return "", err
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	for _, dir := range dirs {
		rel, err := filepath.Rel(filepath.Join(realRoot, dir), real)
		if err == nil && rel != "." && filepath.IsLocal(rel) {
			return real, nil
		}
	}
	return "", fmt.Errorf("the file must be inside %s of the repo, and this one is %s", wanted, real)
}

// errDanglingLink is a symbolic link that points nowhere.
var errDanglingLink = errors.New("a symbolic link that points nowhere")

// resolveExisting resolves the symbolic links of the part of p that exists,
// and keeps the rest of p as it is: a file that is to be written does not
// exist yet, nor may some of the folders above it. A link that exists and
// points nowhere is an error.
func resolveExisting(p string) (string, error) {
	var rest []string
	for {
		real, err := filepath.EvalSymlinks(p)
		if err == nil {
			return filepath.Join(append([]string{real}, rest...)...), nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		if _, lerr := os.Lstat(p); lerr == nil {
			return "", fmt.Errorf("%s is %w", p, errDanglingLink)
		}
		parent := filepath.Dir(p)
		if parent == p {
			return "", err
		}
		rest = append([]string{filepath.Base(p)}, rest...)
		p = parent
	}
}

// loadRulesAt loads the rules as they are in the revision rev of the repo at
// root: guides/, glossary/ and rules/ are exported to a folder of their own,
// which is removed again. A revision git does not know, or one that lacks one
// of the folders, is a user error, and so are rules that do not load.
func loadRulesAt(root, rev string) (*rules.Rules, error) {
	dir, err := os.MkdirTemp("", "lct-impact-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	if err := gitx.Archive(root, rev, baseRulesPaths, dir); err != nil {
		var exited *exec.ExitError
		if errors.As(err, &exited) { // git ran and said no
			return nil, cli.UserError(fmt.Sprintf("impact: the rules of --base %s: %v", rev, err))
		}
		return nil, err
	}
	r, err := rules.Load(dir)
	if err != nil {
		return nil, cli.UserError(fmt.Sprintf("impact: the rules of --base %s do not load: %v", rev, err))
	}
	return r, nil
}
