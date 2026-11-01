package main

import (
	"github.com/tool-jx3/limbus-translate/internal/cli"
	"github.com/tool-jx3/limbus-translate/internal/gitx"
)

// rulesPaths are the repo paths that decide how a text is translated. A
// change to any of them that is not committed marks the rules commit dirty.
var rulesPaths = []string{"guides", "glossary", "rules", "prompts"}

// rulesCommit names the rules command works under, for the records it
// writes and the pack it builds: the repo's HEAD commit, with "+dirty" when
// rulesPaths have changes that are not committed. A repo without commits is
// a user error.
func rulesCommit(ctx *cli.Context, command string) (string, error) {
	commit, err := gitx.Describe(ctx.Root, rulesPaths...)
	if err != nil {
		return "", err
	}
	if commit == "" {
		return "", cli.UserError(command + ": the repo has no commit to record as the rules commit")
	}
	return commit, nil
}
