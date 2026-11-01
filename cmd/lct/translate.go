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
			"or a run fails: --max-usd F --deadline D --max-turns N --run-timeout D [--claude PATH] [--model M]",
		Run: runTranslate,
	})
}

// runTranslate is the translation step of CI (spec 9.2): it runs after plan
// and before build. Whatever the reason it stops, what it did is written to
// .meta/translate.json; only an error of a run makes the command fail.
func runTranslate(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("translate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	maxUSD := flags.Float64("max-usd", 0, "stop once the runs have cost this many US dollars")
	deadline := flags.Duration("deadline", 0, "stop starting runs this long from now, such as 5h")
	maxTurns := flags.Int("max-turns", 0, "the --max-turns of every run")
	runTimeout := flags.Duration("run-timeout", 0, "kill a run that takes longer than this, such as 90m")
	claude := flags.String("claude", "", "the claude command (default: claude, from PATH)")
	model := flags.String("model", "", "the model of the runs (default: $LCT_MODEL)")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("translate: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("translate: unexpected argument " + flags.Arg(0))
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
	// Say what happened even when it cannot be stored.
	saveErr := env.Store.WithLock(func() error {
		return env.Store.WriteMetaJSON(translate.MetaFile, res)
	})
	if err := ctx.Emit(res); err != nil {
		return err
	}
	if saveErr != nil {
		return saveErr
	}
	if res.Stopped == translate.StopError {
		return fmt.Errorf("translate: %s", res.Error)
	}
	return nil
}
