package main

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/proposal"
)

func init() {
	cli.Register(cli.Command{
		Name: "proposal start",
		Usage: "check out the branch of the rules PR: that of the open rules PR, brought up to date and with origin/main merged into it (a conflict is refused and named), or rules-proposal/DATE (the free rules-proposal/DATE-N when a closed PR used that name) from origin/main; " +
			"writes work/proposal/exclude.txt, the suggestion ids that lct suggest --exclude-file must leave out, " +
			"and work/proposal/existing-body.md, the description the open PR has now: --date YYYY-MM-DD",
		Run: runProposalStart,
	})
	cli.Register(cli.Command{
		Name: "proposal publish",
		Usage: "commit the changes of guides/, glossary/ and rules/ on the rules-proposal/* branch, push it, and open the rules PR or update the open one; " +
			"refuses any other branch and any other changed file, and a description or a change that holds what looks like a credential; the description holds a <!-- suggestion:ID --> marker for each suggestion, " +
			"and keeps those of the open PR's current description (work/proposal/existing-body.md): " +
			"--body-file F [--date YYYY-MM-DD]",
		Run: runProposalPublish,
	})
}

// proposalError maps what the proposal package returns to the exit codes: a
// refusal is a user error, a failure of git or gh a system error.
func proposalError(command string, err error) error {
	var refusal *proposal.Refusal
	if errors.As(err, &refusal) {
		return cli.UserError(command + ": " + err.Error())
	}
	return fmt.Errorf("%s: %w", command, err)
}

func runProposalStart(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("proposal start", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	date := flags.String("date", "", "the day of this run, YYYY-MM-DD; the branch is named after it (required)")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("proposal start: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("proposal start: unexpected argument " + flags.Arg(0))
	}
	if *date == "" {
		return cli.UserError("proposal start: --date is required, such as --date 2026-11-05")
	}
	started, err := proposal.Start(proposal.Options{Repo: ctx.Root, Date: *date})
	if err != nil {
		return proposalError("proposal start", err)
	}
	return ctx.Emit(started)
}

func runProposalPublish(ctx *cli.Context, args []string) error {
	flags := flag.NewFlagSet("proposal publish", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	bodyFile := flags.String("body-file", "", "the description of the PR, with a <!-- suggestion:ID --> marker for each suggestion (required)")
	date := flags.String("date", "", "the day of this run, YYYY-MM-DD, for the commit and the title; the day in the name of the branch by default")
	if err := flags.Parse(args); err != nil {
		return cli.UserError("proposal publish: " + err.Error())
	}
	if flags.NArg() > 0 {
		return cli.UserError("proposal publish: unexpected argument " + flags.Arg(0))
	}
	if *bodyFile == "" {
		return cli.UserError("proposal publish: --body-file is required: the description of the PR")
	}
	published, err := proposal.Publish(proposal.Options{Repo: ctx.Root, Date: *date}, *bodyFile)
	if err != nil {
		return proposalError("proposal publish", err)
	}
	return ctx.Emit(published)
}
