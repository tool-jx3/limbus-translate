package main

import (
	"strings"

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

// rulesDirty reports whether commit, a rules commit as rulesCommit returns it,
// says the rules had changes that were not committed: what the records made
// under it name is a commit that does not hold those rules.
func rulesDirty(commit string) bool { return strings.HasSuffix(commit, gitx.DirtySuffix) }
