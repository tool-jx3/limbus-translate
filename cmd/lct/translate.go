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
	"github.com/tool-jx3/limbus-translate/internal/translate"
)

func init() {
	cli.Register(cli.Command{
		Name: "translate",
		Usage: "run the translate-pending skill with the claude CLI until nothing is pending, the money or the time is out, " +
			"or a run fails: --max-usd F --deadline D --max-turns N --run-timeout D [--claude PATH] [--model M]; " +
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
	claude := flags.String("claude", "", "the claude command (default: claude, from PATH)")
	model := flags.String("model", "", "the model of the runs (default: $LCT_MODEL)")
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
