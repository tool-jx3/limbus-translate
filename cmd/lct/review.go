package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tool-jx3/limbus-translate/internal/build"
	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/plan"
	"github.com/tool-jx3/limbus-translate/internal/store"
)

func init() {
	cli.Register(cli.Command{
		Name: "review",
		Usage: "list what needs a person after the last build of the tw worktree, the same items as the review issue " +
			"(a title, and sections of kind, title, hint and items); writes nothing",
		Run: runReview,
	})
}

// buildState is what the last build left in the tw worktree and the review
// issue is made of: the build record, the plan of that snapshot (nil when the
// worktree has none, or one of another snapshot) and the pending list.
type buildState struct {
	Info    build.Info
	Plan    *plan.Summary
	Pending []store.PendingItem
}

// readBuildState reads the build state of the tw worktree at twRoot. "lct
// build" and "lct review" both read it here, so that the issue and the review
// are made of the same inputs.
func readBuildState(twRoot string) (buildState, error) {
	info, err := build.ReadInfo(twRoot)
	if err != nil {
		return buildState{}, err
	}
	sum, err := build.ReadPlan(twRoot, info.RawTag)
	if err != nil {
		return buildState{}, err
	}
	pending, err := store.Open(twRoot).LoadPending()
	if err != nil {
		return buildState{}, err
	}
	return buildState{Info: info, Plan: sum, Pending: pending}, nil
}

// reviewReport is what "review" prints: the title the review issue of this
// build has, and what needs a person. Sections is an empty list, not null,
// when nothing does.
type reviewReport struct {
	Title    string                `json:"title"`
	Sections []build.ReviewSection `json:"sections"`
}

func (r reviewReport) String() string {
	var b strings.Builder
	b.WriteString(r.Title + "\n")
	n := 0
	for _, s := range r.Sections {
		n += len(s.Items)
	}
	if n == 0 {
		b.WriteString("\n沒有需要處理的項目。\n")
		return b.String()
	}
	fmt.Fprintf(&b, "\n%d 項需要處理\n", n)
	for _, s := range r.Sections {
		fmt.Fprintf(&b, "\n## %s [%s]：%d 項\n\n%s\n\n", s.Title, s.Kind, len(s.Items), s.Hint)
		for _, item := range s.Items {
			fmt.Fprintf(&b, "- %s\n", item)
		}
	}
	return b.String()
}

// runReview lists what needs a person (spec 14.2) from the tw worktree, as the
// review issue of the build does. Having items is the result and not a
// finding, so the exit code is 0. A worktree with no build is a user error.
func runReview(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("review", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	if err := flags.Parse(args); err != nil {
		return cli.UserError("review: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("review: unexpected argument " + flags.Arg(0))
	}
	cfg := ctx.Config
	if cfg.Worktrees.TW == "" {
		return cli.UserError("review: worktrees.tw must be set in lct.yaml")
	}
	twRoot := filepath.Join(ctx.Root, filepath.FromSlash(cfg.Worktrees.TW))
	state, err := readBuildState(twRoot)
	if errors.Is(err, build.ErrNotBuilt) {
		if _, statErr := os.Stat(twRoot); statErr != nil {
			return cli.UserError(fmt.Sprintf("review: %s does not exist; make the worktree of the tw branch "+
				"(git fetch origin tw, then git worktree add %s tw)", cfg.Worktrees.TW, cfg.Worktrees.TW))
		}
		return cli.UserError(fmt.Sprintf("review: %s has no build record (.meta/build.json); bring the worktree of the tw branch "+
			"up to date (git fetch origin tw) or run lct build", cfg.Worktrees.TW))
	}
	if err != nil {
		return err
	}
	return ctx.Emit(reviewReport{
		Title:    build.IssueTitle(state.Info.RawTag),
		Sections: build.ReviewItems(state.Info, state.Plan, state.Pending),
	})
}
