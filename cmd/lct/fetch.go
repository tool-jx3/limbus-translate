package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/fetch"
)

// packLangs are the official packs, in the order they are fetched. The names
// are the CDN's {lang} values and the directory names under --out.
var packLangs = []string{"kr", "en", "jp"}

func init() {
	cli.Register(cli.Command{
		Name:  "fetch",
		Usage: "download the official KR/EN/JP packs [--token T --created-at S] [--status-url U] [--cdn-template T] --out DIR",
		Run:   runFetch,
	})
}

// fetchResult is what "fetch" prints.
type fetchResult struct {
	Token     string `json:"token"`
	CreatedAt string `json:"created_at"`
	Dir       string `json:"dir"`
}

func (r fetchResult) String() string {
	return fmt.Sprintf("fetched %s (%s) into %s", r.Token, r.CreatedAt, r.Dir)
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
	out := flags.String("out", "", "directory that receives kr/, en/ and jp/")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("fetch: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("fetch: unexpected argument " + flags.Arg(0))
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

	if err := fetchPacks(bg, template, status.Token, outDir); err != nil {
		return err
	}
	return ctx.Emit(fetchResult{Token: status.Token, CreatedAt: status.CreatedAt, Dir: outDir})
}

// fetchPacks downloads and extracts the three packs into outDir/kr, en and jp,
// replacing what is there. Everything is staged inside outDir first, so a
// failure leaves the earlier contents alone.
func fetchPacks(ctx context.Context, template, token, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(outDir, ".fetch-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)

	for _, lang := range packLangs {
		zipPath := filepath.Join(stage, lang+".zip")
		if err := fetch.Download(ctx, fetch.PackURL(template, token, lang), zipPath); err != nil {
			return err
		}
		if err := fetch.ExtractPack(zipPath, filepath.Join(stage, lang)); err != nil {
			return err
		}
	}
	for _, lang := range packLangs {
		dst := filepath.Join(outDir, lang)
		if err := os.RemoveAll(dst); err != nil {
			return err
		}
		if err := os.Rename(filepath.Join(stage, lang), dst); err != nil {
			return err
		}
	}
	return nil
}
