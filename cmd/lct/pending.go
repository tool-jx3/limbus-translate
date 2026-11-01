package main

import (
	"flag"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/store"
)

func init() {
	cli.Register(cli.Command{
		Name:  "pending",
		Usage: "list the units waiting for translation [--count]",
		Run:   runPending,
	})
}

// pendingCount is what "pending --count" prints.
type pendingCount struct {
	Total       int            `json:"total"`
	ByReason    map[string]int `json:"by_reason"`
	Leased      int            `json:"leased"`
	NeedsReview int            `json:"needs_review"`
}

func (c pendingCount) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "pending: %d\n", c.Total)
	for _, reason := range slices.Sorted(maps.Keys(c.ByReason)) {
		fmt.Fprintf(&b, "  %-20s %d\n", reason, c.ByReason[reason])
	}
	fmt.Fprintf(&b, "leased: %d\n", c.Leased)
	fmt.Fprintf(&b, "needs_review: %d\n", c.NeedsReview)
	return b.String()
}

// pendingList is what "pending" prints: the items, a line each as text.
type pendingList []store.PendingItem

func (l pendingList) String() string {
	if len(l) == 0 {
		return "no pending units"
	}
	var b strings.Builder
	for _, it := range l {
		fmt.Fprintf(&b, "%s\t%s", it.Key, it.Reason)
		if it.Attempts > 0 {
			fmt.Fprintf(&b, "\tattempts=%d", it.Attempts)
		}
		if it.Lease != nil {
			fmt.Fprintf(&b, "\tleased=%s", it.Lease.Batch)
		}
		for _, f := range it.Flags {
			fmt.Fprintf(&b, "\t%s", f)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func runPending(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("pending", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	count := flags.Bool("count", false, "print the number of pending units, by reason, instead of the units")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("pending: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("pending: unexpected argument " + flags.Arg(0))
	}
	if ctx.Config.Worktrees.TW == "" {
		return cli.UserError("pending: worktrees.tw must be set in lct.yaml")
	}
	items, err := openStore(ctx).LoadPending()
	if err != nil {
		return err
	}
	if !*count {
		return ctx.Emit(pendingList(items))
	}

	now := time.Now()
	c := pendingCount{Total: len(items), ByReason: map[string]int{}}
	for _, it := range items {
		c.ByReason[it.Reason]++
		if !store.LeaseFree(it, now) {
			c.Leased++
		}
		if slices.Contains(it.Flags, store.FlagNeedsReview) {
			c.NeedsReview++
		}
	}
	return ctx.Emit(c)
}
