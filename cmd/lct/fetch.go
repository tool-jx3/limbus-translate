package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/fetch"
	"github.com/tool-jx3/limbus-translate/internal/raw"
)

// packLangs are the official packs, in the order they are fetched. The names
// are the CDN's {lang} values and the directory names under --out.
var packLangs = []string{"kr", "en", "jp"}

func init() {
	cli.Register(cli.Command{
		Name: "fetch",
		Usage: "download the official KR/EN/JP packs [--token T --created-at S] [--status-url U] [--cdn-template T] [--cdn-file-template T] --out DIR, " +
			"and with a file template (the flag, $LIMBUS_CDN_FILE_TEMPLATE or cdn_file_template of lct.yaml; one from a place the pack template outranks is not used) the files of the update that belong to no language " +
			"(the file list of the game, the voice table, the file list of the RPG) into DIR/common; " +
			"or, with --check [--status-url U], only ask the status API whether its token differs from the snapshot on raw",
		Run: runFetch,
	})
}

// fetchResult is what "fetch" prints: the token and its creation time, and
// either where the packs went (Dir) or, for --check, whether the token is a
// new one (New).
type fetchResult struct {
	Token     string `json:"token"`
	CreatedAt string `json:"created_at"`
	Dir       string `json:"dir,omitempty"`
	// Common are the common files that were downloaded into Dir/common, as
	// paths under it; absent without a file template.
	Common []string `json:"common,omitempty"`
	// New is set by --check only: the token of the status API is not the
	// token of the snapshot on raw (or raw has no snapshot yet).
	New *bool `json:"new,omitempty"`
}

func (r fetchResult) String() string {
	switch {
	case r.New == nil && len(r.Common) > 0:
		return fmt.Sprintf("fetched %s (%s) into %s, with %d common files: %s", r.Token, r.CreatedAt, r.Dir, len(r.Common), strings.Join(r.Common, ", "))
	case r.New == nil:
		return fmt.Sprintf("fetched %s (%s) into %s", r.Token, r.CreatedAt, r.Dir)
	case *r.New:
		return fmt.Sprintf("new: %s (%s) is not the snapshot on raw", r.Token, r.CreatedAt)
	default:
		return fmt.Sprintf("unchanged: %s (%s) is the snapshot on raw", r.Token, r.CreatedAt)
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func runFetch(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("fetch", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	token := flags.String("token", "", "CDN token of the game update (default: ask the status API)")
	createdAt := flags.String("created-at", "", "creation time of --token")
	statusURL := flags.String("status-url", "", "status API URL (default: $LIMBUS_STATUS_URL)")
	cdnTemplate := flags.String("cdn-template", "", "CDN URL with {token} and {lang} (default: $LIMBUS_CDN_TEMPLATE, then lct.yaml)")
	cdnFileTemplate := flags.String("cdn-file-template", "", "CDN URL of a single file with {token} and {path} (default: $LIMBUS_CDN_FILE_TEMPLATE, then lct.yaml, but never from a place the pack template outranks; none: no common files)")
	out := flags.String("out", "", "directory that receives kr/, en/ and jp/, and common/ with a file template")
	check := flags.Bool("check", false, "only ask the status API and compare its token with the snapshot on raw; download nothing")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("fetch: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("fetch: unexpected argument " + flags.Arg(0))
	}
	if *check {
		for _, f := range []struct{ name, value string }{{"--out", *out}, {"--token", *token}, {"--created-at", *createdAt}, {"--cdn-template", *cdnTemplate}, {"--cdn-file-template", *cdnFileTemplate}} {
			if f.value != "" {
				return cli.UserError("fetch: --check only asks the status API: it takes no " + f.name)
			}
		}
		return runFetchCheck(ctx, *statusURL)
	}
	if *out == "" {
		return cli.UserError("fetch: --out is required")
	}
	if (*token == "") != (*createdAt == "") {
		return cli.UserError("fetch: --token and --created-at must be given together")
	}
	template := firstNonEmpty(*cdnTemplate, os.Getenv("LIMBUS_CDN_TEMPLATE"), ctx.Config.CDNTemplate)
	for _, placeholder := range []string{"{token}", "{lang}"} {
		if !strings.Contains(template, placeholder) {
			return cli.UserError(fmt.Sprintf("fetch: the CDN template %q has no %s", template, placeholder))
		}
	}
	// The file template goes with the pack template: one from a place that the
	// pack template outranks is of another CDN (a mirror given by hand, the
	// fake one of a test) and is not used.
	packSources := []string{*cdnTemplate, os.Getenv("LIMBUS_CDN_TEMPLATE"), ctx.Config.CDNTemplate}
	fileSources := []string{*cdnFileTemplate, os.Getenv("LIMBUS_CDN_FILE_TEMPLATE"), ctx.Config.CDNFileTemplate}
	fileTemplate := firstNonEmpty(fileSources[:slices.IndexFunc(packSources, func(s string) bool { return s != "" })+1]...)
	if fileTemplate != "" {
		for _, placeholder := range []string{"{token}", "{path}"} {
			if !strings.Contains(fileTemplate, placeholder) {
				return cli.UserError(fmt.Sprintf("fetch: the CDN file template %q has no %s", fileTemplate, placeholder))
			}
		}
	}
	outDir, err := filepath.Abs(*out)
	if err != nil {
		return err
	}

	bg := context.Background()
	status := fetch.Status{Token: *token, CreatedAt: *createdAt}
	if *token != "" {
		if err := fetch.CheckToken(*token); err != nil {
			return cli.UserError("fetch: " + err.Error())
		}
	} else {
		url := firstNonEmpty(*statusURL, os.Getenv("LIMBUS_STATUS_URL"))
		if url == "" {
			return cli.UserError("fetch: give --token and --created-at, or a status API with --status-url or $LIMBUS_STATUS_URL")
		}
		if status, err = fetch.FetchStatus(bg, url); err != nil {
			return err
		}
	}

	common, err := fetchPacks(bg, template, fileTemplate, status.Token, outDir)
	if err != nil {
		return err
	}
	return ctx.Emit(fetchResult{Token: status.Token, CreatedAt: status.CreatedAt, Dir: outDir, Common: common})
}

// runFetchCheck asks the status API for the latest token and says whether it
// is new: whether it differs from the token of the snapshot on the raw
// worktree. A raw worktree without a snapshot has no token, so everything is
// new. Nothing is downloaded or written.
func runFetchCheck(ctx *cli.Context, statusURL string) error {
	url := firstNonEmpty(statusURL, os.Getenv("LIMBUS_STATUS_URL"))
	if url == "" {
		return cli.UserError("fetch: --check needs a status API: give --status-url or set $LIMBUS_STATUS_URL")
	}
	if ctx.Config.Worktrees.Raw == "" {
		return cli.UserError("fetch: --check compares with the snapshot on raw: worktrees.raw is not set in lct.yaml")
	}
	status, err := fetch.FetchStatus(context.Background(), url)
	if err != nil {
		return err
	}
	last, err := raw.LastSource(filepath.Join(ctx.Root, filepath.FromSlash(ctx.Config.Worktrees.Raw)))
	if err != nil {
		return err
	}
	isNew := last == nil || last.Token != status.Token
	return ctx.Emit(fetchResult{Token: status.Token, CreatedAt: status.CreatedAt, New: &isNew})
}

// fetchPacks downloads and extracts the three packs into outDir/kr, en and jp,
// replacing what is there. With a fileTemplate it also downloads the common
// files of the update (fetch.CommonFiles of its list of every file) into
// outDir/common and returns their paths under it; without one, outDir/common
// is removed, since what an earlier fetch left there is not of this update.
// Everything is staged inside outDir first, so a failure leaves the earlier
// contents alone.
func fetchPacks(ctx context.Context, template, fileTemplate, token, outDir string) ([]string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	stage, err := os.MkdirTemp(outDir, ".fetch-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(stage)

	for _, lang := range packLangs {
		zipPath := filepath.Join(stage, lang+".zip")
		if err := fetch.Download(ctx, fetch.PackURL(template, token, lang), zipPath); err != nil {
			return nil, err
		}
		if err := fetch.ExtractPack(zipPath, filepath.Join(stage, lang)); err != nil {
			return nil, err
		}
	}
	var common []string
	if fileTemplate != "" {
		// The list of every file stays in memory: it also holds a key of the game.
		info, err := fetch.ReadPatchInfo(ctx, fetch.FileURL(fileTemplate, token, fetch.PatchInfoPath))
		if err != nil {
			return nil, err
		}
		if common, err = fetch.CommonFiles(info); err != nil {
			return nil, err
		}
		for _, rel := range common {
			dest := filepath.Join(stage, raw.CommonDir, filepath.FromSlash(rel))
			if err := fetch.Download(ctx, fetch.FileURL(fileTemplate, token, fetch.LocalizeRoot+rel), dest); err != nil {
				return nil, err
			}
		}
	}
	dirs := packLangs
	if len(common) > 0 {
		dirs = append(slices.Clone(packLangs), raw.CommonDir)
	}
	if err := os.RemoveAll(filepath.Join(outDir, raw.CommonDir)); err != nil {
		return nil, err
	}
	for _, dir := range dirs {
		dst := filepath.Join(outDir, dir)
		if err := os.RemoveAll(dst); err != nil {
			return nil, err
		}
		if err := os.Rename(filepath.Join(stage, dir), dst); err != nil {
			return nil, err
		}
	}
	return common, nil
}
