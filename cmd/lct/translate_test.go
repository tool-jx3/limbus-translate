package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tool-jx3/limbus-translate/internal/translate"
)

// fakeClaudeCLI stands in for the claude CLI in the tests of lct translate.
// A run records its arguments and LCT_MODEL, empties the pending queue when
// FAKE_EMPTY names its file, and prints FAKE_OUT.
const fakeClaudeCLI = `#!/bin/sh
echo "$*" >> "$FAKE_DIR/args"
echo "$LCT_MODEL" >> "$FAKE_DIR/model"
if [ -n "$FAKE_EMPTY" ]; then : > "$FAKE_EMPTY"; fi
printf '%s\n' "$FAKE_OUT"
`

// translateRepo is a batch repo (two pending units, rules committed) with a
// fake claude. The working directory is the repo. It returns the repo root
// and the directory the fake records into; script is the fake itself.
func translateRepo(t *testing.T) (root, fakeDir, script string) {
	t.Helper()
	root, _ = batchRepo(t)
	t.Chdir(root)
	fakeDir = t.TempDir()
	script = filepath.Join(fakeDir, "claude")
	if err := os.WriteFile(script, []byte(fakeClaudeCLI), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_DIR", fakeDir)
	t.Setenv("FAKE_EMPTY", "")
	t.Setenv("FAKE_OUT", `{"type":"result","subtype":"success","is_error":false,"num_turns":3,"total_cost_usd":0.25}`)
	return root, fakeDir, script
}

// translateArgs are the flags every run of lct translate needs.
func translateArgs(script string, more ...string) []string {
	return append([]string{"--json", "translate", "--max-usd", "5", "--deadline", "1h", "--max-turns", "9", "--run-timeout", "1m", "--claude", script}, more...)
}

func readTranslateJSON(t *testing.T, root string) (translate.Result, string) {
	t.Helper()
	text := string(readFile(t, filepath.Join(root, ".work", "tw", ".meta", translate.MetaFile)))
	var res translate.Result
	if err := json.Unmarshal([]byte(text), &res); err != nil {
		t.Fatalf("translate.json is not JSON: %v\n%s", err, text)
	}
	return res, text
}

func headCommit(t *testing.T, root string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func TestTranslateCommandRecordsTheRun(t *testing.T) {
	root, fakeDir, script := translateRepo(t)
	t.Setenv("FAKE_EMPTY", filepath.Join(root, ".work", "tw", ".meta", "pending.jsonl"))
	t.Setenv("LCT_MODEL", "from-the-environment")

	code, stdout, stderr := lct(t, translateArgs(script, "--model", "from-the-flag")...)
	if code != 0 {
		t.Fatalf("lct translate: exit code %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	res, stored := readTranslateJSON(t, root)
	if stdout != stored {
		t.Errorf("lct translate printed\n%s\nbut .meta/translate.json holds\n%s", stdout, stored)
	}
	want := translate.Result{
		Model: "from-the-flag", Runs: 1, CostUSD: 0.25, Stopped: translate.StopDone,
		PendingBefore: 2, PendingAfter: 0, RawTag: "raw-20261029", RulesCommit: headCommit(t, root),
	}
	if res != want {
		t.Errorf("translate.json = %+v, want %+v", res, want)
	}

	if got := strings.TrimSpace(string(readFile(t, filepath.Join(fakeDir, "args")))); got != "-p /translate-pending --output-format json --model from-the-flag --max-turns 9" {
		t.Errorf("claude was run with %q", got)
	}
	if got := strings.TrimSpace(string(readFile(t, filepath.Join(fakeDir, "model")))); got != "from-the-flag" {
		t.Errorf("claude saw LCT_MODEL=%q, want from-the-flag", got)
	}
	// The log of the rounds goes to stderr, where --json output is not.
	if !strings.Contains(stderr, "round 1: cost so far $0.00, available 2") {
		t.Errorf("stderr lacks the line of round 1:\n%s", stderr)
	}
}

// Without --model the model is $LCT_MODEL; a run that gets nowhere is not an
// error of the command.
func TestTranslateCommandModelFromEnvironmentAndNoProgress(t *testing.T) {
	root, fakeDir, script := translateRepo(t)
	t.Setenv("LCT_MODEL", "from-the-environment")

	code, stdout, stderr := lct(t, translateArgs(script)...)
	if code != 0 {
		t.Fatalf("lct translate: exit code %d, want 0 for no-progress\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	res, _ := readTranslateJSON(t, root)
	if res.Model != "from-the-environment" || res.Stopped != translate.StopNoProgress || res.Runs != 2 || res.PendingAfter != 2 {
		t.Errorf("translate.json = %+v, want the environment's model and no-progress after 2 runs", res)
	}
	if got := strings.Fields(string(readFile(t, filepath.Join(fakeDir, "model")))); len(got) != 2 || got[0] != "from-the-environment" {
		t.Errorf("claude saw LCT_MODEL %v", got)
	}

	// As text it is a summary for people.
	code, stdout, stderr = lct(t, translateArgs(script)[1:]...)
	if code != 0 || !strings.Contains(stdout, "stopped (no-progress) after 2 runs") {
		t.Errorf("lct translate as text: exit code %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
}

// A failed run is the one stop that fails the command (exit code 2), but
// what it cost and why it failed are stored and printed all the same.
func TestTranslateCommandErrorExitsWithTwo(t *testing.T) {
	root, _, script := translateRepo(t)
	t.Setenv("FAKE_OUT", `{"type":"result","subtype":"error_during_execution","is_error":true,"num_turns":2,"total_cost_usd":0.5}`)

	code, stdout, stderr := lct(t, translateArgs(script, "--model", "m")...)
	if code != 2 {
		t.Errorf("exit code %d, want 2\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	res, stored := readTranslateJSON(t, root)
	if stdout != stored {
		t.Errorf("lct translate printed\n%s\nbut .meta/translate.json holds\n%s", stdout, stored)
	}
	if res.Stopped != translate.StopError || res.Runs != 1 || res.CostUSD != 0.5 || !strings.Contains(res.Error, "error_during_execution") {
		t.Errorf("translate.json = %+v, want error after 1 run costing 0.5", res)
	}
	if !strings.Contains(stderr, "error_during_execution") {
		t.Errorf("stderr does not say why:\n%s", stderr)
	}
}

func TestTranslateCommandUserErrors(t *testing.T) {
	root, fakeDir, script := translateRepo(t)
	t.Setenv("LCT_MODEL", "") // no model unless a case gives one

	flags := func(drop string, more ...string) []string {
		all := map[string]string{"--max-usd": "5", "--deadline": "1h", "--max-turns": "9", "--run-timeout": "1m", "--claude": script, "--model": "m"}
		args := []string{"translate"}
		for _, k := range []string{"--max-usd", "--deadline", "--max-turns", "--run-timeout", "--claude", "--model"} {
			if k != drop {
				args = append(args, k, all[k])
			}
		}
		return append(args, more...)
	}
	for name, args := range map[string][]string{
		"no max-usd":      flags("--max-usd"),
		"no deadline":     flags("--deadline"),
		"no max-turns":    flags("--max-turns"),
		"no run-timeout":  flags("--run-timeout"),
		"no model":        flags("--model"),
		"zero max-usd":    flags("--max-usd", "--max-usd", "0"),
		"negative usd":    flags("--max-usd", "--max-usd", "-1"),
		"bad deadline":    flags("--deadline", "--deadline", "soon"),
		"zero deadline":   flags("--deadline", "--deadline", "0s"),
		"zero turns":      flags("--max-turns", "--max-turns", "0"),
		"bad run-timeout": flags("--run-timeout", "--run-timeout", "-5m"),
		"extra argument":  flags("", "extra"),
		"unknown flag":    flags("", "--nope"),
	} {
		code, stdout, stderr := lct(t, args...)
		if code != 1 {
			t.Errorf("%s: exit code %d, want 1\nstdout:\n%s\nstderr:\n%s", name, code, stdout, stderr)
		}
	}
	if _, err := os.Stat(filepath.Join(fakeDir, "args")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("claude ran although the command was refused (%v)", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".work", "tw", ".meta", translate.MetaFile)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("translate.json was written although the command was refused (%v)", err)
	}
}

// --not-run writes the record of a translation that was never started, for a
// pipeline that has units to translate and no key or no model: it needs
// neither, runs nothing and is no failure.
func TestTranslateNotRun(t *testing.T) {
	root, fakeDir, script := translateRepo(t)
	t.Setenv("LCT_MODEL", "") // the reason for the record
	const why = "缺少 ANTHROPIC_API_KEY 或 LCT_MODEL"

	// A record of an earlier run of the same inputs is replaced.
	if code, _, stderr := lct(t, translateArgs(script, "--model", "m")[1:]...); code != 0 {
		t.Fatalf("lct translate: exit code %d\n%s", code, stderr)
	}
	if err := os.Remove(filepath.Join(fakeDir, "args")); err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := lct(t, "--json", "translate", "--not-run", why)
	if code != 0 {
		t.Fatalf("lct translate --not-run: exit code %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	res, stored := readTranslateJSON(t, root)
	if stdout != stored {
		t.Errorf("lct translate printed\n%s\nbut .meta/translate.json holds\n%s", stdout, stored)
	}
	want := translate.Result{
		Stopped: translate.StopNotRun, Error: why, PendingBefore: 2, PendingAfter: 2,
		RawTag: "raw-20261029", RulesCommit: headCommit(t, root),
	}
	if res != want {
		t.Errorf("translate.json = %+v, want %+v", res, want)
	}
	if res.Runs != 0 || res.Model != "" || res.CostUSD != 0 {
		t.Errorf("translate.json = %+v, want no runs, no model and no cost", res)
	}
	if _, err := os.Stat(filepath.Join(fakeDir, "args")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("claude ran for a translation that was not to run (%v)", err)
	}

	// As text it is a line for people, and the reason is trimmed.
	code, stdout, stderr = lct(t, "translate", "--not-run", "  "+why+"\n")
	if code != 0 || !strings.Contains(stdout, "not run") || !strings.Contains(stdout, why) {
		t.Errorf("lct translate --not-run as text: exit code %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	if res, _ := readTranslateJSON(t, root); res.Error != why {
		t.Errorf("the reason = %q, want it trimmed to %q", res.Error, why)
	}
}

func TestTranslateNotRunUserErrors(t *testing.T) {
	root, _, _ := translateRepo(t)
	for name, args := range map[string][]string{
		"an empty reason":      {"translate", "--not-run", ""},
		"a blank reason":       {"translate", "--not-run", " \t"},
		"a reason as argument": {"translate", "--not-run"},
		"an extra argument":    {"translate", "--not-run", "x", "extra"},
	} {
		code, stdout, stderr := lct(t, args...)
		if code != 1 {
			t.Errorf("%s: exit code %d, want 1\nstdout:\n%s\nstderr:\n%s", name, code, stdout, stderr)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".work", "tw", ".meta", translate.MetaFile)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("translate.json was written although the command was refused (%v)", err)
	}
}
