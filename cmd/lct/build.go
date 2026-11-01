package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/raw"
)

const (
	// buildOutDir is where the zips go, relative to the repo root (it is not
	// under version control).
	buildOutDir = "out"
	// licenseFile is the content licence, copied into the pack.
	licenseFile = "LICENSE-CONTENT"
	// sourceDateEpoch is the environment variable of the reproducible-builds
	// convention: the build time in seconds since the Unix epoch.
	sourceDateEpoch = "SOURCE_DATE_EPOCH"
	// zipEpochYear is the first year a zip file can store.
	zipEpochYear = 1980
)

func init() {
	cli.Register(cli.Command{
		Name:  "build",
		Usage: "assemble the zh-TW language pack in the tw worktree and zip it into out/",
		Run:   runBuild,
	})
}

// buildOutput is what "build" prints. Report is the object that
// .meta/build.json holds; ReportPath is the markdown report.
type buildOutput struct {
	RawTag      string        `json:"raw_tag"`
	TextZip     string        `json:"text_zip"`
	CompleteZip string        `json:"complete_zip"`
	Report      *build.Report `json:"report"`
	ReportPath  string        `json:"report_path"`
}

func (o buildOutput) String() string {
	r := o.Report
	return fmt.Sprintf("built %s: %d pending, %d structural fallbacks, %d need review, %d unaligned, %d missing glyphs\n"+
		"text zip:     %s\ncomplete zip: %s\nreport:       %s",
		o.RawTag, r.Pending, r.StructuralFallback, r.NeedsReview, r.Unaligned, len(r.MissingGlyphs),
		o.TextZip, o.CompleteZip, o.ReportPath)
}

func runBuild(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("build", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	if err := flags.Parse(args); err != nil {
		return cli.UserError("build: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("build: unexpected argument " + flags.Arg(0))
	}
	cfg := ctx.Config
	if cfg.Worktrees.Raw == "" || cfg.Worktrees.TW == "" || cfg.LangDir == "" {
		return cli.UserError("build: worktrees.raw, worktrees.tw and lang_dir must be set in lct.yaml")
	}
	if cfg.Font.Path == "" || cfg.Font.SHA256 == "" {
		return cli.UserError("build: font.path and font.sha256 must be set in lct.yaml")
	}

	// Everything that is cheap to check comes first, so that a refusal leaves
	// the tw worktree alone.
	rawWorktree := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.Raw))
	src, err := raw.LastSource(rawWorktree)
	if err != nil {
		return err
	}
	if src == nil {
		return cli.UserError(fmt.Sprintf("build: %s has no snapshot; run lct snapshot first", cfg.Worktrees.Raw))
	}
	set, err := pack.OpenRaw(rawWorktree)
	if err != nil {
		return err
	}
	fr, err := fields.Load(ctx.Root)
	if err != nil {
		return err
	}
	fontPath := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Font.Path))
	licensePath := filepath.Join(ctx.Root, licenseFile)
	for _, f := range []string{fontPath, build.FontLicensePath(fontPath), licensePath} {
		if _, err := os.Stat(f); err != nil {
			return cli.UserError("build: " + err.Error())
		}
	}
	// The commit comes before the build time, which is read from it unless
	// SOURCE_DATE_EPOCH is set: a repo without commits is a user error.
	rulesCommit, err := gitx.Head(ctx.Root)
	if err != nil {
		return err
	}
	if rulesCommit == "" {
		return cli.UserError("build: the repo has no commit to record as the rules commit")
	}
	builtAt, err := buildTime(ctx.Root)
	if err != nil {
		return err
	}

	twRoot := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.TW))
	if err := gitx.EnsureWorktree(ctx.Root, twBranch, twRoot); err != nil {
		return err
	}
	upstream := map[string]string{}
	for _, u := range upstreams(cfg) {
		if u.up.Repo != "" && u.up.Tag != "" {
			upstream[u.name] = u.up.Repo + "@" + u.up.Tag
		}
	}
	opt := build.Options{
		TWRoot:      twRoot,
		OutDir:      filepath.Join(ctx.Root, buildOutDir),
		LangDir:     cfg.LangDir,
		FontPath:    fontPath,
		FontSHA256:  cfg.Font.SHA256,
		LicensePath: licensePath,
		RawTag:      src.Tag,
		RawToken:    src.Token,
		RulesCommit: rulesCommit,
		Upstream:    upstream,
		BuiltAt:     builtAt,
		Frozen:      cfg.Frozen,
	}
	rep, err := build.Run(opt, set, fr)
	if errors.Is(err, build.ErrFontChecksum) {
		return cli.UserError(err.Error() + " (font.sha256 in lct.yaml)")
	}
	if err != nil {
		return err
	}
	textZip, completeZip := build.ZipPaths(opt)
	return ctx.Emit(buildOutput{
		RawTag:      src.Tag,
		TextZip:     textZip,
		CompleteZip: completeZip,
		Report:      rep,
		ReportPath:  build.ReportPath(opt),
	})
}

// buildTime is the build time: SOURCE_DATE_EPOCH when it is set (an empty
// value counts as unset), otherwise the committer time of the repo's HEAD. It
// never depends on the clock, so a build of the same commit gives the same
// bytes whenever it runs.
func buildTime(root string) (time.Time, error) {
	if v := os.Getenv(sourceDateEpoch); v != "" {
		sec, err := strconv.ParseInt(v, 10, 64)
		if err != nil || sec < 0 {
			return time.Time{}, cli.UserError(fmt.Sprintf("build: %s=%q is not a number of seconds since the Unix epoch", sourceDateEpoch, v))
		}
		t := time.Unix(sec, 0).UTC()
		if t.Year() < zipEpochYear {
			return time.Time{}, cli.UserError(fmt.Sprintf("build: %s=%q is before %d, which zip files cannot store", sourceDateEpoch, v, zipEpochYear))
		}
		return t, nil
	}
	return gitx.CommitTime(root)
}
