package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tool-jx3/limbus-translate/internal/batch"
	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/translate"
)

// progressSubject is how the commit that keeps what a translation did begins,
// followed by the raw tag; the idle check of the pipeline reads it in the last
// commit of tw, and the pipeline's own commit after the step uses it too.
const progressSubject = "translate: keep the progress of"

func init() {
	cli.Register(cli.Command{
		Name: "translate",
		Usage: "run the translate-pending skill with the claude CLI until nothing is pending, the money or the time is out, " +
			"or a run fails: --max-usd F --deadline D --max-turns N --run-timeout D [--soft-turns N] [--claude PATH] [--model M] [--push-each-run] " +
			"(--soft-turns: after this many turns a run gets no new batch and ends once its batch is submitted; the batches a run leaves unsubmitted are submitted when their out file is there and released otherwise) " +
			"(--push-each-run: after every run, commit the tw worktree as \"" + progressSubject + " <raw tag>\" and push it to the tw of origin); " +
			"or, with --not-run REASON, run nothing and record in .meta/translate.json that the translation was not run",
		Run: runTranslate,
	})
}

// runTranslate is the translation step of CI (spec 9.2): it runs after plan
// and before build. Whatever the reason it stops, what it did is written to
// .meta/translate.json; only an error of a run makes the command fail. With
// --not-run it starts nothing and only records that, and why.
func runTranslate(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("translate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	maxUSD := flags.Float64("max-usd", 0, "stop once the runs have cost this many US dollars")
	deadline := flags.Duration("deadline", 0, "stop starting runs this long from now, such as 5h")
	maxTurns := flags.Int("max-turns", 0, "the --max-turns of every run")
	runTimeout := flags.Duration("run-timeout", 0, "kill a run that takes longer than this, such as 90m")
	softTurns := flags.Int("soft-turns", 0, "after this many turns a run gets no new batch, and ends once the batch it has is submitted (0: only --max-turns)")
	claude := flags.String("claude", "", "the claude command (default: claude, from PATH)")
	model := flags.String("model", "", "the model of the runs (default: $LCT_MODEL)")
	pushEachRun := flags.Bool("push-each-run", false, "after every run, commit the tw worktree and push it to the tw of origin, so that a runner that is lost loses one run at most")
	notRun := flags.String("not-run", "", "run nothing: record in .meta/translate.json that the translation was not run, for this reason "+
		"(a pipeline with units to translate and no API key or no model); the other flags are not needed")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("translate: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("translate: unexpected argument " + flags.Arg(0))
	}
	// --not-run "" must not fall back to a run: a flag that was given is a
	// request not to run.
	notRunGiven := false
	flags.Visit(func(f *flag.Flag) { notRunGiven = notRunGiven || f.Name == "not-run" })
	if notRunGiven {
		return runTranslateNotRun(ctx, strings.TrimSpace(*notRun))
	}
	switch {
	case !(*maxUSD > 0):
		return cli.UserError("translate: --max-usd must be a positive number")
	case *deadline <= 0:
		return cli.UserError("translate: --deadline must be a positive duration, such as 5h")
	case *maxTurns < 1:
		return cli.UserError("translate: --max-turns must be at least 1")
	case *runTimeout <= 0:
		return cli.UserError("translate: --run-timeout must be a positive duration, such as 90m")
	case *softTurns < 0 || *softTurns >= *maxTurns:
		return cli.UserError("translate: --soft-turns must be 0, or less than --max-turns")
	}
	if *model == "" {
		*model = os.Getenv("LCT_MODEL")
	}
	if *model == "" {
		return cli.UserError("translate: no model to run: set LCT_MODEL (or give --model)")
	}
	// A command with a directory in it is relative to where the user is, not
	// to the repo root the runs work in.
	if strings.ContainsRune(*claude, filepath.Separator) {
		abs, err := filepath.Abs(*claude)
		if err != nil {
			return err
		}
		*claude = abs
	}

	// Everything that is cheap to check comes first, so that a refusal costs
	// nothing.
	env, _, err := openBatchEnv(ctx, "translate")
	if err != nil {
		return err
	}
	commit, err := rulesCommit(ctx, "translate")
	if err != nil {
		return err
	}

	// Batches a run left: the run that took them is over. One lost earlier
	// (a runner that died) holds its units too, so they are settled before
	// the first round as well.
	settle := settleBatches(ctx, env, *model)
	if err := settle(0); err != nil {
		fmt.Fprintf(os.Stderr, "before round 1: settling the batches left: %v\n", err)
	}
	var afterRun func(int) error
	if *pushEachRun {
		afterRun = keepProgress(env, filepath.Join(ctx.Root, filepath.FromSlash(ctx.Config.Worktrees.TW)), progressSubject+" "+env.RawTag)
	}
	res, err := translate.Run(translate.Options{
		Root:       ctx.Root,
		Claude:     *claude,
		Model:      *model,
		MaxUSD:     *maxUSD,
		Deadline:   time.Now().Add(*deadline),
		MaxTurns:   *maxTurns,
		RunTimeout: *runTimeout,
		// The queue is read anew each time; KR does not change meanwhile.
		Available: func() (int, error) { return batch.Available(env, batchNow()) },
		Log:       os.Stderr,
		SoftTurns: *softTurns,
		StopFile:  filepath.Join(ctx.Root, filepath.FromSlash(batchStopFile)),
		Settle:    settle,
		AfterRun:  afterRun,
	})
	if err != nil {
		return err
	}
	res.RawTag, res.RulesCommit = env.RawTag, commit
	if err := storeTranslation(ctx, env, res); err != nil {
		return err
	}
	if res.Stopped == translate.StopError {
		return fmt.Errorf("translate: %s", res.Error)
	}
	return nil
}

// settleBatches returns what lct translate does with the batches still leased
// after a run (round 0: before the first): a run that is over will not submit
// them. One whose out file is there is submitted as lct batch submit does; the
// others, and one whose out file is refused, are released, so that the next
// round can take their units at once rather than when the lease runs out.
func settleBatches(ctx *cli.Context, env batch.Env, model string) func(int) error {
	return func(round int) error {
		when := fmt.Sprintf("round %d", round)
		if round == 0 {
			when = "before round 1"
		}
		ids, err := batch.LeasedBatches(env, batchNow())
		if err != nil {
			return err
		}
		for _, id := range ids {
			if has, err := pathExists(batchPath(ctx, id, ".out.json")); err != nil {
				return err
			} else if has {
				rep, err := submitBatch(ctx, id, model)
				if err == nil {
					fmt.Fprintf(os.Stderr, "%s: submitted the batch %s the run left: %d accepted, %d rejected\n", when, id, len(rep.Accepted), len(rep.Rejected))
					// Units it did not cover are still leased: free them too.
					if _, err := batch.Release(env, id); err != nil {
						return err
					}
					continue
				}
				fmt.Fprintf(os.Stderr, "%s: the batch %s the run left could not be submitted: %v\n", when, id, err)
			}
			n, err := batch.Release(env, id)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "%s: released the batch %s the run left (%d units)\n", when, id, n)
		}
		return nil
	}
}

// keepProgress returns what lct translate --push-each-run does after a run:
// commit the tw worktree at dir with msg, under the lock of the store, and push
// it to the tw of origin. A push that failed is tried again after the next
// run even when that run changed nothing, so the error it returns loses
// nothing: the commit stays, and the pipeline pushes once the step is over.
func keepProgress(env batch.Env, dir, msg string) func(int) error {
	unpushed := false
	return func(round int) error {
		var sha string
		var changed bool
		err := env.Store.WithLock(func() error {
			var err error
			sha, changed, err = gitx.CommitAll(dir, msg)
			return err
		})
		if err != nil {
			return fmt.Errorf("committing the progress on tw: %w", err)
		}
		if !changed && !unpushed {
			return nil
		}
		if err := gitx.Push(dir, "origin", "tw"); err != nil {
			unpushed = true
			return fmt.Errorf("pushing the progress to tw (it stays committed): %w", err)
		}
		unpushed = false
		fmt.Fprintf(os.Stderr, "round %d: kept the progress on tw (%.7s)\n", round, sha)
		return nil
	}
}

// runTranslateNotRun records a translation that was not run: the pipeline had
// units to translate and no API key or no model. No claude is started and no
// model is needed. The record names the inputs, so that the build of the same
// inputs reads it and the review issue says why the units are untranslated;
// nothing is translated and the queue is left as it is. It is no failure.
func runTranslateNotRun(ctx *cli.Context, reason string) error {
	if reason == "" {
		return cli.UserError("translate: --not-run needs the reason the translation is not run")
	}
	env, _, err := openBatchEnv(ctx, "translate")
	if err != nil {
		return err
	}
	commit, err := rulesCommit(ctx, "translate")
	if err != nil {
		return err
	}
	available, err := batch.Available(env, batchNow())
	if err != nil {
		return err
	}
	return storeTranslation(ctx, env, &translate.Result{
		Stopped: translate.StopNotRun, Error: reason,
		PendingBefore: available, PendingAfter: available,
		RawTag: env.RawTag, RulesCommit: commit,
	})
}

// storeTranslation writes res to .meta/translate.json and prints it. It says
// what happened even when that cannot be stored.
func storeTranslation(ctx *cli.Context, env batch.Env, res *translate.Result) error {
	saveErr := env.Store.WithLock(func() error {
		return env.Store.WriteMetaJSON(translate.MetaFile, res)
	})
	if err := ctx.Emit(res); err != nil {
		return err
	}
	return saveErr
}
