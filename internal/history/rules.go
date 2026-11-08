package history

import (
	"fmt"
	"os"
	"sync"

	"github.com/tool-jx3/limbus-translate/internal/adjust"
	"github.com/tool-jx3/limbus-translate/internal/gitx"
	"github.com/tool-jx3/limbus-translate/internal/requeue"
	"github.com/tool-jx3/limbus-translate/internal/rewrite"
)

// RuleFiles are the rule files that one commit of main holds.
type RuleFiles struct {
	Commit      string
	Rewrites    []rewrite.File
	Requeues    []requeue.File
	Adjustments []adjust.Adjustment
}

// RulesSource gives the rule files of a commit.
type RulesSource func(commit string) (*RuleFiles, error)

// RulesAt reads rules/rewrite, rules/requeue and rules/adjustments of commit in
// repo. tmp is a directory the caller owns and removes; the rules are exported
// to a new directory below it. A commit without a rules directory, or without
// one of them, has none of those rules; a commit that does not exist is an
// error.
func RulesAt(repo, commit, tmp string) (*RuleFiles, error) {
	rules, err := rulesAt(repo, commit, tmp)
	if err != nil {
		return nil, fmt.Errorf("history: read the rules of %s: %w", commit, err)
	}
	return rules, nil
}

func rulesAt(repo, commit, tmp string) (*RuleFiles, error) {
	if _, err := gitx.RevTime(repo, commit); err != nil {
		return nil, err
	}
	has, err := gitx.HasPath(repo, commit, "rules")
	if err != nil {
		return nil, err
	}
	out := &RuleFiles{Commit: commit}
	if !has {
		return out, nil
	}
	root, err := os.MkdirTemp(tmp, "rules-")
	if err != nil {
		return nil, err
	}
	if err := gitx.Archive(repo, commit, []string{"rules"}, root); err != nil {
		return nil, err
	}
	if out.Rewrites, err = rewrite.LoadFiles(root); err != nil {
		return nil, err
	}
	if out.Requeues, err = requeue.LoadFiles(root); err != nil {
		return nil, err
	}
	if out.Adjustments, err = adjust.Load(root); err != nil {
		return nil, err
	}
	return out, nil
}

// CachedRules is RulesAt as a RulesSource that reads each commit at most once:
// a commit that failed to read fails the same way again, without a new read.
// It is safe for concurrent use.
func CachedRules(repo, tmp string) RulesSource {
	type result struct {
		rules *RuleFiles
		err   error
	}
	var mu sync.Mutex
	cache := map[string]result{}
	return func(commit string) (*RuleFiles, error) {
		mu.Lock()
		defer mu.Unlock()
		r, ok := cache[commit]
		if !ok {
			r.rules, r.err = RulesAt(repo, commit, tmp)
			cache[commit] = r
		}
		return r.rules, r.err
	}
}
