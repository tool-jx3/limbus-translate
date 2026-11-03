package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/publish"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/site"
)

// siteLangRoot is where the built pack lies in the tw worktree, before the
// language directory: <tw>/LimbusCompany_Data/Lang/<lang_dir>.
var siteLangRoot = filepath.Join("LimbusCompany_Data", "Lang")

// siteReleaseTagRe matches a Release tag, zh-TW-YYYYMMDD.N. Another tag with
// the prefix (zh-TW-latest) is no Release.
var siteReleaseTagRe = regexp.MustCompile(`^` + regexp.QuoteMeta(publish.TagPrefix) + `\d{8}\.\d+$`)

func init() {
	cli.Register(cli.Command{
		Name: "site",
		Usage: "write the data of the static viewer (an index, the files with KR, JP, EN and TW side by side, and the search corpus) " +
			"from the raw and tw worktrees: --out DIR (inside the repo, such as site/public/data; emptied first)",
		Run: runSite,
	})
}

// siteOutput is what "site" prints: the Result of the export.
type siteOutput struct{ site.Result }

func (o siteOutput) String() string {
	return fmt.Sprintf("wrote %d files (%d units, %d in the search corpus), %.1f MB, to %s",
		o.Files, o.Units, o.SearchUnits, float64(o.Bytes)/(1<<20), o.Out)
}

func runSite(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("site", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	out := flags.String("out", "", "the directory to write the data to, relative to the repo root")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("site: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("site: unexpected argument " + flags.Arg(0))
	}
	if *out == "" {
		return cli.UserError("site: --out DIR is required: a directory inside the repo, such as site/public/data")
	}
	cfg := ctx.Config
	if cfg.Worktrees.Raw == "" || cfg.Worktrees.TW == "" || cfg.LangDir == "" {
		return cli.UserError("site: worktrees.raw, worktrees.tw and lang_dir must be set in lct.yaml")
	}
	rawWorktree := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.Raw))
	twRoot := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.TW))
	// The directory is emptied: it is checked before anything is read.
	outDir, err := siteOutDir(ctx.Root, *out, rawWorktree, twRoot)
	if err != nil {
		return err
	}

	src, err := raw.LastSource(rawWorktree)
	if err != nil {
		return err
	}
	if src == nil {
		return cli.UserError(fmt.Sprintf("site: %s has no snapshot; run lct snapshot first", cfg.Worktrees.Raw))
	}
	set, err := pack.OpenRaw(rawWorktree)
	if err != nil {
		return err
	}
	st, err := openTWStore(ctx, "site")
	if err != nil {
		return err
	}
	info, err := build.ReadInfo(twRoot)
	if errors.Is(err, build.ErrNotBuilt) {
		return cli.UserError("site: " + err.Error())
	}
	if err != nil {
		return err
	}
	// The KR texts come from raw and the TW texts from the pack: of two
	// snapshots they would not line up.
	if info.RawTag != src.Tag {
		return cli.UserError(fmt.Sprintf("site: the pack in %s was built from %s, but %s is at %s; run lct build first",
			cfg.Worktrees.TW, info.RawTag, cfg.Worktrees.Raw, src.Tag))
	}
	records, err := st.LoadRecords()
	if err != nil {
		return err
	}
	r, err := rules.Load(ctx.Root)
	if err != nil {
		return err
	}
	zhDir := filepath.Join(twRoot, siteLangRoot, cfg.LangDir)
	if _, err := os.Stat(zhDir); errors.Is(err, fs.ErrNotExist) {
		return cli.UserError(fmt.Sprintf("site: %s has no pack at %s; run lct build first", cfg.Worktrees.TW, filepath.ToSlash(filepath.Join(siteLangRoot, cfg.LangDir))))
	}
	zh, err := pack.Open(zhDir, "")
	if err != nil {
		return err
	}
	var missing []string
	for _, f := range set.KR.Files() {
		if !zh.Has(f) {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		return cli.UserError(fmt.Sprintf("site: the built pack lacks %d of the %d files of the KR pack, such as %s; run lct build first",
			len(missing), len(set.KR.Files()), missing[0]))
	}

	head, err := gitx.Head(twRoot)
	if err != nil {
		return err
	}
	release := ""
	if head != "" {
		tags, err := gitx.TagsAt(twRoot, publish.TagPrefix)
		if err != nil {
			return err
		}
		release = latestReleaseTag(tags)
	}
	meta := site.Meta{RawTag: src.Tag, RulesCommit: info.RulesCommit, TWCommit: head, Release: release}
	res, err := site.Export(site.Inputs{KR: set.KR, EN: set.EN, JP: set.JP, ZH: zh, Records: records, Rules: r}, meta, outDir)
	if err != nil {
		return err
	}
	return ctx.Emit(siteOutput{res})
}

// latestReleaseTag returns the latest of the Release tags (zh-TW-YYYYMMDD.N)
// among names, the later day first and within a day the greater N as a number
// (.10 is after .9); "" when there is none.
func latestReleaseTag(names []string) string {
	last := ""
	for _, name := range names {
		if siteReleaseTagRe.MatchString(name) && (last == "" || site.NaturalLess(last, name)) {
			last = name
		}
	}
	return last
}

// siteOutDir returns the absolute path of out, the directory "site" empties and
// writes to. out is relative to root, the repo root, or an absolute path; it
// must lie inside root without being root, hold no .git, and neither be nor
// hold nor lie in one of the protected directories (the worktrees the data is
// read from). Symbolic links are followed, so a link cannot lead out. Anything
// else is a user error, and nothing has been touched.
func siteOutDir(root, out string, protected ...string) (string, error) {
	refuse := func(why string) error {
		return cli.UserError(fmt.Sprintf("site: --out %q %s", out, why))
	}
	abs := out
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(root, abs)
	}
	abs = filepath.Clean(abs)
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	realAbs, err := evalLoosely(abs)
	if err != nil {
		return "", err
	}
	// The path as written and the place it leads to must both be fine.
	for _, p := range [][2]string{{root, abs}, {realRoot, realAbs}} {
		base, dir := p[0], p[1]
		if !within(base, dir) {
			return "", refuse("is outside the repo")
		}
		if dir == base {
			return "", refuse("is the repo root")
		}
		rel, _ := filepath.Rel(base, dir)
		if slices.Contains(strings.Split(rel, string(filepath.Separator)), ".git") {
			return "", refuse("is inside .git")
		}
	}
	for _, dir := range protected {
		realDir, err := evalLoosely(dir)
		if err != nil {
			return "", err
		}
		if within(realAbs, realDir) || within(realDir, realAbs) {
			return "", refuse("holds or lies in the raw or tw worktree, which the data is made from")
		}
	}
	return abs, nil
}

// within reports whether target is base or lies below it.
func within(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// evalLoosely resolves the symbolic links of p, which need not exist: the part
// that does is resolved, and the rest is appended as written.
func evalLoosely(p string) (string, error) {
	var rest []string
	for {
		real, err := filepath.EvalSymlinks(p)
		if err == nil {
			for i := len(rest) - 1; i >= 0; i-- {
				real = filepath.Join(real, rest[i])
			}
			return real, nil
		}
		parent := filepath.Dir(p)
		if !errors.Is(err, fs.ErrNotExist) || parent == p {
			return "", err
		}
		rest = append(rest, filepath.Base(p))
		p = parent
	}
}
