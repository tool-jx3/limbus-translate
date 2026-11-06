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

	"github.com/tool-jx3/limbus-translate/internal/adjust"
	"github.com/tool-jx3/limbus-translate/internal/batch"
	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/history"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/publish"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/site"
	"github.com/tool-jx3/limbus-translate/internal/store"
	"github.com/tool-jx3/limbus-translate/internal/suggest"
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
			"from the raw and tw worktrees: --out DIR (inside the repo, such as site/public/data; emptied first, so refused if it holds tracked files)",
		Run: runSite,
	})
}

// siteOutput is what "site" prints: the Result of the export.
type siteOutput struct{ site.Result }

func (o siteOutput) String() string {
	return fmt.Sprintf("wrote %d files (%d units, %d in the search corpus), %.1f MB (%.1f MB of it the preview), to %s",
		o.Files, o.Units, o.SearchUnits, float64(o.Bytes)/(1<<20), float64(o.PreviewBytes)/(1<<20), o.Out)
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

	ss, err := openSiteSources(ctx, "site")
	if err != nil {
		return err
	}
	src, set, zh, info, records, r := ss.Src, ss.Set, ss.ZH, ss.Info, ss.Records, ss.Rules

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
	review, err := loadReviewInputs(ctx, "site", r)
	if err != nil {
		return err
	}
	preview, err := loadPreviewInputs(ctx, "site", records, &set)
	if err != nil {
		return err
	}
	res, err := site.Export(site.Inputs{KR: set.KR, EN: set.EN, JP: set.JP, ZH: zh, Records: records, Rules: r, Review: &review, Preview: preview}, meta, outDir)
	if err != nil {
		return err
	}
	hist, err := history.List(twRoot)
	if err != nil {
		return err
	}
	// The Releases and their dates are those of the tags, not of the records.
	allTags, err := gitx.Tags(twRoot, publish.TagPrefix)
	if err != nil {
		return err
	}
	var releases []site.Release
	for _, tag := range publish.SortedReleases(allTags) {
		at, err := gitx.RevTime(twRoot, "refs/tags/"+tag)
		if err != nil {
			return err
		}
		releases = append(releases, site.Release{Tag: tag, At: at.Format("2006-01-02T15:04:05Z")})
	}
	if _, err := site.ExportUpdates(releases, hist, r, outDir); err != nil {
		return err
	}
	// Result.Bytes is the size of everything written.
	err = filepath.WalkDir(filepath.Join(outDir, "updates"), func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err == nil {
			res.Bytes += info.Size()
		}
		return err
	})
	if err != nil {
		return err
	}
	return ctx.Emit(siteOutput{res})
}

// previewPromptFile is the translation prompt the preview shows.
const previewPromptFile = "prompts/translate.md"

// loadPreviewInputs opens what the preview data is made from: the Previewer,
// on the environment lct batch preview loads (openPreviewEnv), the rules commit
// of that environment, the translation prompt and every guide under guides/,
// keyed by their path relative to the repo root with forward slashes, the way
// the batches name them (rules.Rules.GuidesFor). cmd names the command in the
// errors. records are the records and set the raw packs the caller has read
// already (nil: they are read here).
func loadPreviewInputs(ctx *cli.Context, cmd string, records map[string]store.Record, set *pack.Set) (*site.PreviewInputs, error) {
	env, frozen, recs, err := openPreviewEnv(ctx, cmd, records, set)
	if err != nil {
		return nil, err
	}
	previewer, err := batch.NewPreviewerWith(env, frozen, recs)
	if err != nil {
		return nil, err
	}
	prompt, err := os.ReadFile(filepath.Join(ctx.Root, filepath.FromSlash(previewPromptFile)))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", cmd, err)
	}
	guides := map[string]string{}
	guidesDir := filepath.Join(ctx.Root, "guides")
	err = filepath.WalkDir(guidesDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return err
		}
		rel, err := filepath.Rel(ctx.Root, p)
		if err != nil {
			return err
		}
		text, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		guides[filepath.ToSlash(rel)] = string(text)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", cmd, err)
	}
	return &site.PreviewInputs{Previewer: previewer, RulesCommit: env.RulesCommit, Prompt: string(prompt), Guides: guides}, nil
}

// siteSources are the packs and stores the review items and the viewer data are
// computed from: the last raw snapshot and its packs, the built zh pack of the
// tw worktree and the build record it came with, the records and the rules.
type siteSources struct {
	Src     *raw.Source
	Set     pack.Set
	ZH      *pack.Pack
	Info    build.Info
	Records map[string]store.Record
	Rules   *rules.Rules
}

// openSiteSources opens what lct site and lct accept compute from. The pack
// must have been built from the last snapshot and hold every file of KR; if not
// it is a user error that says to run lct snapshot or lct build. cmd names the
// command in the errors.
func openSiteSources(ctx *cli.Context, cmd string) (siteSources, error) {
	cfg := ctx.Config
	if cfg.Worktrees.Raw == "" || cfg.Worktrees.TW == "" || cfg.LangDir == "" {
		return siteSources{}, cli.UserError(cmd + ": worktrees.raw, worktrees.tw and lang_dir must be set in lct.yaml")
	}
	rawWorktree := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.Raw))
	twRoot := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.TW))
	var ss siteSources
	var err error
	if ss.Src, err = raw.LastSource(rawWorktree); err != nil {
		return ss, err
	}
	if ss.Src == nil {
		return ss, cli.UserError(fmt.Sprintf("%s: %s has no snapshot; run lct snapshot first", cmd, cfg.Worktrees.Raw))
	}
	if ss.Set, err = pack.OpenRaw(rawWorktree); err != nil {
		return ss, err
	}
	st, err := openTWStore(ctx, cmd)
	if err != nil {
		return ss, err
	}
	if ss.Info, err = build.ReadInfo(twRoot); errors.Is(err, build.ErrNotBuilt) {
		return ss, cli.UserError(cmd + ": " + err.Error())
	} else if err != nil {
		return ss, err
	}
	// The KR texts come from raw and the TW texts from the pack: of two
	// snapshots they would not line up.
	if ss.Info.RawTag != ss.Src.Tag {
		return ss, cli.UserError(fmt.Sprintf("%s: the pack in %s was built from %s, but %s is at %s; run lct build first",
			cmd, cfg.Worktrees.TW, ss.Info.RawTag, cfg.Worktrees.Raw, ss.Src.Tag))
	}
	if ss.Records, err = st.LoadRecords(); err != nil {
		return ss, err
	}
	if ss.Rules, err = rules.Load(ctx.Root); err != nil {
		return ss, err
	}
	zhDir := filepath.Join(twRoot, siteLangRoot, cfg.LangDir)
	if _, err := os.Stat(zhDir); errors.Is(err, fs.ErrNotExist) {
		return ss, cli.UserError(fmt.Sprintf("%s: %s has no pack at %s; run lct build first", cmd, cfg.Worktrees.TW, filepath.ToSlash(filepath.Join(siteLangRoot, cfg.LangDir))))
	}
	if ss.ZH, err = pack.Open(zhDir, ""); err != nil {
		return ss, err
	}
	var missing []string
	for _, f := range ss.Set.KR.Files() {
		if !ss.ZH.Has(f) {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		return ss, cli.UserError(fmt.Sprintf("%s: the built pack lacks %d of the %d files of the KR pack, such as %s; run lct build first",
			cmd, len(missing), len(ss.Set.KR.Files()), missing[0]))
	}
	return ss, nil
}

// loadReviewInputs reads what the review items are computed from: the raw tag,
// the build record and plan of the tw worktree (nil when there is none; a
// plan of another snapshot is no plan), the adjustments, the field rules, the
// frozen patterns, the notes, the pending list, the declined and accepted
// entries and the provisional terms. r are the rules the caller loaded (the
// glossary decides which provisional terms count); they are not read again.
// cmd names the command in the errors.
func loadReviewInputs(ctx *cli.Context, cmd string, r *rules.Rules) (site.ReviewInputs, error) {
	cfg := ctx.Config
	if cfg.Worktrees.Raw == "" || cfg.Worktrees.TW == "" {
		return site.ReviewInputs{}, cli.UserError(cmd + ": worktrees.raw and worktrees.tw must be set in lct.yaml")
	}
	src, err := raw.LastSource(filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.Raw)))
	if err != nil {
		return site.ReviewInputs{}, err
	}
	if src == nil {
		return site.ReviewInputs{}, cli.UserError(fmt.Sprintf("%s: %s has no snapshot; run lct snapshot first", cmd, cfg.Worktrees.Raw))
	}
	in := site.ReviewInputs{RawTag: src.Tag, Frozen: cfg.Frozen}
	if in.Adjustments, err = adjust.Load(ctx.Root); err != nil {
		return in, err
	}
	if in.Fields, err = fields.Load(ctx.Root); err != nil {
		return in, err
	}
	if in.Declined, err = suggest.LoadDeclined(ctx.Root); err != nil {
		return in, err
	}
	if in.Accepted, err = suggest.LoadAccepted(ctx.Root); err != nil {
		return in, err
	}
	st, err := openTWStore(ctx, cmd)
	if err != nil {
		return in, err
	}
	twRoot := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.TW))
	switch info, err := build.ReadInfo(twRoot); {
	case err == nil:
		in.Build = &info
	case !errors.Is(err, build.ErrNotBuilt):
		return in, err
	}
	if in.Plan, err = build.ReadPlan(twRoot, src.Tag); err != nil {
		return in, err
	}
	if in.Notes, err = st.LoadNotes(); err != nil {
		return in, err
	}
	if in.Pending, err = st.LoadPending(); err != nil {
		return in, err
	}
	if in.Provisional, _, err = activeProvisionalRecords(ctx.Root, st, r); err != nil {
		return in, err
	}
	return in, nil
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
// must lie inside root without being root, hold no .git, hold no file that git
// tracks (committed or staged; untracked and ignored files are the data of an
// earlier run), and neither be nor hold nor lie in one of the protected
// directories (the worktrees the data is read from). Symbolic links are
// followed, so a link cannot lead out. Anything else is a user error, and
// nothing has been touched.
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
	if info, err := os.Stat(realAbs); err == nil && !info.IsDir() {
		return "", refuse("is a file, not a directory")
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
	// The directory is emptied: not one that holds the source of the repo. Git
	// is asked about the path as written and, when a link makes it another, about
	// the place it leads to.
	places := [][2]string{{root, abs}}
	if realRoot != root || realAbs != abs {
		places = append(places, [2]string{realRoot, realAbs})
	}
	for _, p := range places {
		rel, _ := filepath.Rel(p[0], p[1])
		tracked, err := gitx.Tracked(p[0], filepath.ToSlash(rel))
		if err != nil {
			return "", err
		}
		if len(tracked) > 0 {
			return "", refuse(fmt.Sprintf("holds tracked files, such as %s, and is emptied before the data is written; name a directory the repo does not track", tracked[0]))
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
