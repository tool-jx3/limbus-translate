package history

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/gitx"
)

const rewriteRule = "reason: %s\nfiles: ['**']\npattern: 'a'\nreplace: 'b'\n"

func commitRules(t *testing.T, root, msg string) string {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, "rules")); err == nil {
		runGit(t, root, nil, "add", "rules")
	}
	runGit(t, root, nil, "commit", "-q", "--allow-empty", "-m", msg)
	sha, err := gitx.Head(root)
	if err != nil {
		t.Fatal(err)
	}
	return sha
}

func TestRulesAtReadsRulesOfCommit(t *testing.T) {
	f := newRepoFixture(t)
	c0 := commitRules(t, f.root, "no rules")
	put(t, f.root, "rules/rewrite/x.yaml", strings.Replace(rewriteRule, "%s", "第一版", 1))
	put(t, f.root, "rules/requeue/r.yaml", "reason: 重送\nfiles: ['A/**']\n")
	put(t, f.root, "rules/adjustments/2026-11.yaml", "- key: A.json|/dataList/0/content\n  raw: raw-1\n  zh: 甲\n")
	c1 := commitRules(t, f.root, "first")
	put(t, f.root, "rules/rewrite/x.yaml", strings.Replace(rewriteRule, "%s", "第二版", 1))
	c2 := commitRules(t, f.root, "second")

	tmp := t.TempDir()
	r1, err := RulesAt(f.root, c1, tmp)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := RulesAt(f.root, c2, tmp)
	if err != nil {
		t.Fatal(err)
	}
	if r1.Commit != c1 || len(r1.Rewrites) != 1 || r1.Rewrites[0].ID != "x" || r1.Rewrites[0].Reason != "第一版" {
		t.Errorf("rules of c1 = %+v", r1)
	}
	if len(r2.Rewrites) != 1 || r2.Rewrites[0].Reason != "第二版" {
		t.Errorf("rules of c2 = %+v", r2.Rewrites)
	}
	if len(r1.Requeues) != 1 || r1.Requeues[0].ID != "r" || len(r1.Adjustments) != 1 || r1.Adjustments[0].Key != "A.json|/dataList/0/content" {
		t.Errorf("requeues %+v, adjustments %+v", r1.Requeues, r1.Adjustments)
	}

	// A commit that has no rules directory has no rules; that is no error.
	r0, err := RulesAt(f.root, c0, tmp)
	if err != nil {
		t.Fatal(err)
	}
	if r0.Commit != c0 || len(r0.Rewrites)+len(r0.Requeues)+len(r0.Adjustments) != 0 {
		t.Errorf("rules of c0 = %+v, want none", r0)
	}
}

func TestRulesAtMissingCommit(t *testing.T) {
	f := newRepoFixture(t)
	const missing = "0123456789abcdef0123456789abcdef01234567"
	_, err := RulesAt(f.root, missing, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), missing) {
		t.Errorf("RulesAt of a missing commit = %v, want an error naming %s", err, missing)
	}
}

func TestCachedRulesReadsACommitOnce(t *testing.T) {
	f := newRepoFixture(t)
	put(t, f.root, "rules/rewrite/x.yaml", strings.Replace(rewriteRule, "%s", "一", 1))
	c := commitRules(t, f.root, "rules")
	const missing = "0123456789abcdef0123456789abcdef01234567"

	src := CachedRules(f.root, t.TempDir())
	first, err := src(c)
	if err != nil {
		t.Fatal(err)
	}
	_, missErr := src(missing)
	if missErr == nil {
		t.Fatal("a missing commit gave no error")
	}
	// With the repo gone, only a cache can still answer.
	if err := os.RemoveAll(filepath.Join(f.root, ".git")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(f.tw); err != nil {
		t.Fatal(err)
	}
	again, err := src(c)
	if err != nil || !reflect.DeepEqual(again, first) {
		t.Errorf("second read = %+v, %v; want the cached %+v", again, err, first)
	}
	if _, err := src(missing); err == nil || err.Error() != missErr.Error() {
		t.Errorf("second read of the missing commit = %v, want the cached %v", err, missErr)
	}
}
