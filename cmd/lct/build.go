package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
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
	// summaryFile is the report for the summary page of a workflow run, and
	// issueDir holds the review issue, both in buildOutDir.
	summaryFile = "summary.md"
	issueDir    = "review-issue"
	issueTitle  = "title.txt"
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
// ReviewIssueParts is the number of parts of the review issue that
// out/review-issue/ holds, 0 when nothing needs a person and there is no issue.
type buildOutput struct {
	RawTag           string        `json:"raw_tag"`
	TextZip          string        `json:"text_zip"`
	CompleteZip      string        `json:"complete_zip"`
	Report           *build.Report `json:"report"`
	ReportPath       string        `json:"report_path"`
	ReviewIssueParts int           `json:"review_issue_parts"`
}

func (o buildOutput) String() string {
	r := o.Report
	text := fmt.Sprintf("built %s: %d pending, %d structural fallbacks, %d need review, %d unaligned, %d missing glyphs\n"+
		"text zip:     %s\ncomplete zip: %s\nreport:       %s",
		o.RawTag, r.Pending, r.StructuralFallback, r.NeedsReview, r.Unaligned, len(r.MissingGlyphs),
		o.TextZip, o.CompleteZip, o.ReportPath)
	if o.ReviewIssueParts > 0 {
		text += fmt.Sprintf("\nreview issue: %d parts in %s/%s/", o.ReviewIssueParts, buildOutDir, issueDir)
	}
	return text
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
	r, err := rules.Load(ctx.Root)
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
	rules, err := rulesCommit(ctx, "build")
	if err != nil {
		return err
	}
	builtAt, err := buildTime(ctx.Root)
	if err != nil {
		return err
	}

	twRoot := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.TW))
	if err := gitx.EnsureWorktree(ctx.Root, twBranch, twRoot); err != nil {
		return err
	}
	// Where tw stands before the build, for the compare page of the changes.
	prevTW, err := gitx.Head(twRoot)
	if err != nil {
		return err
	}
	upstream := map[string]string{}
	for _, u := range upstreams(cfg) {
		if u.up.Repo != "" && u.up.Tag != "" {
			upstream[u.name] = u.up.Repo + "@" + u.up.Tag
		}
	}
	opt := build.Options{
		TWRoot:       twRoot,
		OutDir:       filepath.Join(ctx.Root, buildOutDir),
		LangDir:      cfg.LangDir,
		FontPath:     fontPath,
		FontSHA256:   cfg.Font.SHA256,
		LicensePath:  licensePath,
		RawTag:       src.Tag,
		RawToken:     src.Token,
		RulesCommit:  rules,
		Upstream:     upstream,
		BuiltAt:      builtAt,
		Frozen:       cfg.Frozen,
		Rules:        r,
		RepoURL:      repoURL(),
		PrevTWCommit: prevTW,
	}
	rep, err := build.Run(opt, set, fr)
	if errors.Is(err, build.ErrFontChecksum) {
		return cli.UserError(err.Error() + " (font.sha256 in lct.yaml)")
	}
	if err != nil {
		return err
	}
	parts, err := writeReviewFiles(opt)
	if err != nil {
		return err
	}
	textZip, completeZip := build.ZipPaths(opt)
	return ctx.Emit(buildOutput{
		RawTag:           src.Tag,
		TextZip:          textZip,
		CompleteZip:      completeZip,
		Report:           rep,
		ReportPath:       build.ReportPath(opt),
		ReviewIssueParts: parts,
	})
}

// repoURL is the URL of the repo the workflow runs in, from the variables
// GitHub Actions sets; "" outside of it.
func repoURL() string {
	server, repo := os.Getenv("GITHUB_SERVER_URL"), os.Getenv("GITHUB_REPOSITORY")
	if server == "" || repo == "" {
		return ""
	}
	return strings.TrimRight(server, "/") + "/" + repo
}

// writeReviewFiles writes what the workflow posts, to opt.OutDir: the report
// as summary.md, and the review issue as review-issue/title.txt (the title and
// a newline) and the parts 01.md, 02.md, … in order (the number has two digits
// at least). The directory of the issue is removed first and stays removed
// when nothing needs a person, so that the issue of an earlier build is not
// posted again. It returns the number of parts.
func writeReviewFiles(opt build.Options) (int, error) {
	twRoot := opt.TWRoot
	info, err := build.ReadInfo(twRoot)
	if err != nil {
		return 0, err
	}
	sum, err := build.ReadPlan(twRoot, info.RawTag)
	if err != nil {
		return 0, err
	}
	pending, err := store.Open(twRoot).LoadPending()
	if err != nil {
		return 0, err
	}
	report, err := os.ReadFile(build.ReportPath(opt))
	if err != nil {
		return 0, err
	}
	summary := build.SummaryMarkdown(report, build.ReportRel(info.RawTag))
	if err := os.WriteFile(filepath.Join(opt.OutDir, summaryFile), summary, 0o644); err != nil {
		return 0, err
	}

	dir := filepath.Join(opt.OutDir, issueDir)
	if err := os.RemoveAll(dir); err != nil {
		return 0, err
	}
	title, parts, ok := build.ReviewIssue(info, sum, pending)
	if !ok {
		return 0, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, err
	}
	if err := os.WriteFile(filepath.Join(dir, issueTitle), []byte(title+"\n"), 0o644); err != nil {
		return 0, err
	}
	width := max(2, len(strconv.Itoa(len(parts))))
	for i, part := range parts {
		name := fmt.Sprintf("%0*d.md", width, i+1)
		if err := os.WriteFile(filepath.Join(dir, name), []byte(part), 0o644); err != nil {
			return 0, err
		}
	}
	return len(parts), nil
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
