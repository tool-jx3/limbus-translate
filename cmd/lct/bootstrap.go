package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/bootstrap"
	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/config"
	"github.com/tool-jx3/limbus-translate/internal/fetch"
	"github.com/tool-jx3/limbus-translate/internal/fields"
	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/pack"
	"github.com/tool-jx3/limbus-translate/internal/raw"
	"github.com/tool-jx3/limbus-translate/internal/rules"
	"github.com/tool-jx3/limbus-translate/internal/store"
)

const (
	// twBranch is the orphan branch that keeps the translations.
	twBranch = "tw"
	// defaultUpstream is where the upstream packs are kept, relative to the
	// repo root.
	defaultUpstream = "work/upstream"
	// llcPackDir is where the LLC release zip keeps its files.
	llcPackDir = "LimbusCompany_Data/Lang/LLC_zh-CN"
	// foreignNamesFile is written beside the records, relative to the tw
	// worktree.
	foreignNamesFile = ".meta/foreign-names.json"
)

// upstreamBase is the host --fetch downloads release assets from. The tests
// point it at a directory (file:// URLs are accepted).
var upstreamBase = "https://github.com"

func init() {
	cli.Register(cli.Command{
		Name:  "bootstrap",
		Usage: "import the pinned upstream translations into the tw store [--fetch] [--upstream DIR] [--force]",
		Run:   runBootstrap,
	})
}

// upstream is one of the three pinned releases: its directory under the
// upstream directory and what lct.yaml pins.
type upstream struct {
	name string
	up   config.Upstream
}

func upstreams(cfg *config.Config) []upstream {
	return []upstream{
		{"paratranz", cfg.Bootstrap.Paratranz},
		{"ltm", cfg.Bootstrap.LTM},
		{"llc", cfg.Bootstrap.LLC},
	}
}

// upstreamURL is the download URL of a pinned release asset.
func upstreamURL(base string, up config.Upstream) string {
	return base + "/" + up.Repo + "/releases/download/" + url.PathEscape(up.Tag) + "/" + url.PathEscape(up.Asset)
}

// bootstrapReport is what "bootstrap" prints.
type bootstrapReport struct {
	RawTag       string         `json:"raw_tag"`
	Records      int            `json:"records"`
	Pending      int            `json:"pending"`
	ForeignNames int            `json:"foreign_names"`
	Stats        map[string]int `json:"stats"`
}

func (r bootstrapReport) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "bootstrap from %s: %d records, %d pending\n", r.RawTag, r.Records, r.Pending)
	for _, key := range slices.Sorted(maps.Keys(r.Stats)) {
		fmt.Fprintf(&b, "  %-20s %d\n", key, r.Stats[key])
	}
	fmt.Fprintf(&b, "foreign names: %d\n", r.ForeignNames)
	return b.String()
}

func runBootstrap(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("bootstrap", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	doFetch := flags.Bool("fetch", false, "download the pinned upstream releases first")
	upstreamDir := flags.String("upstream", defaultUpstream, "directory holding paratranz/, ltm/ and llc/ (relative to the repo root)")
	force := flags.Bool("force", false, "replace a tw store that already has records or pending items")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("bootstrap: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("bootstrap: unexpected argument " + flags.Arg(0))
	}
	cfg := ctx.Config
	if cfg.Worktrees.Raw == "" || cfg.Worktrees.TW == "" {
		return cli.UserError("bootstrap: worktrees.raw and worktrees.tw must be set in lct.yaml")
	}
	upDir := *upstreamDir
	if !filepath.IsAbs(upDir) {
		upDir = filepath.Join(ctx.Root, upDir)
	}

	if *doFetch {
		if err := fetchUpstreams(context.Background(), cfg, upDir); err != nil {
			return err
		}
	}
	// Everything is read and checked before the tw worktree is touched.
	in, err := openBootstrapInputs(ctx, upDir)
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

	twRoot := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.TW))
	if err := gitx.EnsureWorktree(ctx.Root, twBranch, twRoot); err != nil {
		return err
	}
	st := store.Open(twRoot)
	var res *bootstrap.Result
	err = st.WithLock(func() error {
		if !*force {
			if err := refuseExistingState(st); err != nil {
				return err
			}
		}
		var err error
		if res, err = bootstrap.Run(in, fr, r); err != nil {
			return err
		}
		if err := st.SaveRecords(res.Records); err != nil {
			return err
		}
		if err := st.SavePending(res.Pending); err != nil {
			return err
		}
		return writeForeignNames(filepath.Join(twRoot, filepath.FromSlash(foreignNamesFile)), res.ForeignNames)
	})
	if err != nil {
		return err
	}
	return ctx.Emit(bootstrapReport{
		RawTag:       in.RawTag,
		Records:      len(res.Records),
		Pending:      len(res.Pending),
		ForeignNames: len(res.ForeignNames),
		Stats:        res.Stats,
	})
}

// refuseExistingState is the guard against importing over work: it fails with
// a user error when the store already has records or pending items.
func refuseExistingState(st *store.Store) error {
	records, err := st.LoadRecords()
	if err != nil {
		return err
	}
	pending, err := st.LoadPending()
	if err != nil {
		return err
	}
	if len(records) > 0 || len(pending) > 0 {
		return cli.UserError(fmt.Sprintf(
			"bootstrap: the tw store already holds %d records and %d pending items; --force replaces them",
			len(records), len(pending)))
	}
	return nil
}

// openBootstrapInputs opens the raw packs and the three upstream packs.
func openBootstrapInputs(ctx *cli.Context, upDir string) (bootstrap.Inputs, error) {
	rawWorktree := filepath.Join(ctx.Root, filepath.FromSlash(ctx.Config.Worktrees.Raw))
	src, err := raw.LastSource(rawWorktree)
	if err != nil {
		return bootstrap.Inputs{}, err
	}
	if src == nil {
		return bootstrap.Inputs{}, cli.UserError(fmt.Sprintf(
			"bootstrap: %s has no snapshot; run lct snapshot first", ctx.Config.Worktrees.Raw))
	}
	set, err := pack.OpenRaw(rawWorktree)
	if err != nil {
		return bootstrap.Inputs{}, err
	}
	in := bootstrap.Inputs{Raw: set, RawTag: src.Tag, Frozen: ctx.Config.Frozen}
	for _, p := range []struct {
		dir string
		dst **pack.Pack
	}{
		{filepath.Join(upDir, "paratranz"), &in.Paratranz},
		{filepath.Join(upDir, "ltm"), &in.LTM},
		{filepath.Join(upDir, "llc", filepath.FromSlash(llcPackDir)), &in.LLC},
	} {
		if info, err := os.Stat(p.dir); err != nil || !info.IsDir() {
			return bootstrap.Inputs{}, cli.UserError(fmt.Sprintf(
				"bootstrap: %s is missing; run with --fetch to download the pinned releases", p.dir))
		}
		if *p.dst, err = pack.Open(p.dir, ""); err != nil {
			return bootstrap.Inputs{}, err
		}
	}
	return in, nil
}

// fetchUpstreams downloads the three pinned release assets to
// <upDir>/<name>.zip and extracts each into <upDir>/<name>, replacing what an
// earlier fetch left there.
func fetchUpstreams(ctx context.Context, cfg *config.Config, upDir string) error {
	list := upstreams(cfg)
	for _, u := range list {
		if u.up.Repo == "" || u.up.Tag == "" || u.up.Asset == "" {
			return cli.UserError(fmt.Sprintf("bootstrap: bootstrap.%s in lct.yaml needs repo, tag and asset", u.name))
		}
	}
	for _, u := range list {
		zipPath := filepath.Join(upDir, u.name+".zip")
		if err := fetch.Download(ctx, upstreamURL(upstreamBase, u.up), zipPath); err != nil {
			return err
		}
		dest := filepath.Join(upDir, u.name)
		if err := os.RemoveAll(dest); err != nil {
			return err
		}
		if err := fetch.ExtractZip(zipPath, dest); err != nil {
			return err
		}
	}
	return nil
}

// writeForeignNames writes names as a JSON array, one name per line, replacing
// the file atomically.
func writeForeignNames(path string, names []string) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(names); err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, err = f.Write(buf.Bytes())
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(tmp, 0o644)
	}
	if err == nil {
		err = os.Rename(tmp, path)
	}
	if err != nil {
		os.Remove(tmp)
	}
	return err
}
