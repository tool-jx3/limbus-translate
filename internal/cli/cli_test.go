package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// inRepo creates a temporary directory holding a minimal lct.yaml and makes it
// the working directory, so Main can resolve a repo root without depending on
// where the tests are run from.
func inRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "lct.yaml"), []byte("lang_dir: zh-TW\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	return root
}

// register adds cmd for the duration of the test.
func register(t *testing.T, cmd Command) {
	t.Helper()
	Register(cmd)
	t.Cleanup(func() { delete(registry, cmd.Name) })
}

// exec runs the CLI in-process and returns its exit code and output.
func exec(args ...string) (code int, stdout, stderr string) {
	var out, errOut bytes.Buffer
	code = run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestExitCodes(t *testing.T) {
	inRepo(t)
	tests := []struct {
		name       string
		err        error
		wantCode   int
		wantStderr string // substring; empty means stderr must be empty
	}{
		{"success", nil, 0, ""},
		{"user error", UserError("bad input"), 1, "bad input"},
		{"wrapped user error", fmt.Errorf("ctx: %w", UserError("bad input")), 1, "ctx: bad input"},
		{"findings", ErrFindings, 1, ""},
		{"wrapped findings", fmt.Errorf("3 files: %w", ErrFindings), 1, "3 files"},
		{"other error", errors.New("disk exploded"), 2, "disk exploded"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			register(t, Command{
				Name: "t-exit",
				Run:  func(*Context, []string) error { return tc.err },
			})
			code, stdout, stderr := exec("t-exit")
			if code != tc.wantCode {
				t.Errorf("exit code = %d, want %d", code, tc.wantCode)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if tc.wantStderr == "" && stderr != "" {
				t.Errorf("stderr = %q, want empty", stderr)
			}
			if !strings.Contains(stderr, tc.wantStderr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr, tc.wantStderr)
			}
		})
	}
}

func TestTwoWordCommand(t *testing.T) {
	inRepo(t)
	var got []string
	register(t, Command{
		Name: "batch next",
		Run: func(_ *Context, args []string) error {
			got = append([]string{"called"}, args...)
			return nil
		},
	})

	if code := Main([]string{"batch", "next", "--limit", "5"}); code != 0 {
		t.Fatalf("Main exit code = %d, want 0", code)
	}
	if want := []string{"called", "--limit", "5"}; !slices.Equal(got, want) {
		t.Errorf("handler saw %v, want %v", got, want)
	}
}

func TestOneWordCommandKeepsRemainingArgs(t *testing.T) {
	inRepo(t)
	var one, two []string
	register(t, Command{Name: "t-multi", Run: func(_ *Context, args []string) error { one = args; return nil }})
	register(t, Command{Name: "t-multi sub", Run: func(_ *Context, args []string) error { two = args; return nil }})

	if code, _, _ := exec("t-multi", "other", "x"); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if want := []string{"other", "x"}; !slices.Equal(one, want) || two != nil {
		t.Errorf("one-word handler saw %v (two-word saw %v), want %v", one, two, want)
	}

	one = nil
	if code, _, _ := exec("t-multi", "sub", "x"); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if want := []string{"x"}; !slices.Equal(two, want) || one != nil {
		t.Errorf("two-word handler saw %v (one-word saw %v), want %v", two, one, want)
	}
}

func TestJSONFlagAnywhere(t *testing.T) {
	inRepo(t)
	var gotJSON bool
	var gotArgs []string
	register(t, Command{
		Name: "t-json",
		Run: func(ctx *Context, args []string) error {
			gotJSON, gotArgs = ctx.JSON, args
			return nil
		},
	})

	tests := []struct {
		args     []string
		wantJSON bool
	}{
		{[]string{"t-json", "a", "b"}, false},
		{[]string{"--json", "t-json", "a", "b"}, true},
		{[]string{"t-json", "--json", "a", "b"}, true},
		{[]string{"t-json", "a", "b", "--json"}, true},
	}
	for _, tc := range tests {
		gotJSON, gotArgs = !tc.wantJSON, nil
		if code, _, stderr := exec(tc.args...); code != 0 {
			t.Fatalf("%v: exit code = %d (stderr %q), want 0", tc.args, code, stderr)
		}
		if gotJSON != tc.wantJSON {
			t.Errorf("%v: ctx.JSON = %v, want %v", tc.args, gotJSON, tc.wantJSON)
		}
		if want := []string{"a", "b"}; !slices.Equal(gotArgs, want) {
			t.Errorf("%v: handler args = %v, want %v", tc.args, gotArgs, want)
		}
	}
}

func TestContextCarriesRootAndConfig(t *testing.T) {
	root := inRepo(t)
	sub := filepath.Join(root, "x", "y")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)

	var ctx *Context
	register(t, Command{Name: "t-ctx", Run: func(c *Context, _ []string) error { ctx = c; return nil }})

	var out, errOut bytes.Buffer
	if code := run([]string{"t-ctx"}, &out, &errOut); code != 0 {
		t.Fatalf("exit code = %d (stderr %q), want 0", code, errOut.String())
	}
	if ctx.Root != root {
		t.Errorf("ctx.Root = %q, want %q", ctx.Root, root)
	}
	if ctx.Config == nil || ctx.Config.LangDir != "zh-TW" {
		t.Errorf("ctx.Config = %+v, want LangDir zh-TW", ctx.Config)
	}
	if ctx.Out != &out {
		t.Error("ctx.Out is not the writer given to run")
	}
}

func TestOutsideRepoIsUserError(t *testing.T) {
	t.Chdir(t.TempDir())
	called := false
	register(t, Command{Name: "t-norepo", Run: func(*Context, []string) error { called = true; return nil }})

	code, _, stderr := exec("t-norepo")
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if called {
		t.Error("command ran outside an lct repo")
	}
	if !strings.Contains(stderr, "lct.yaml") {
		t.Errorf("stderr = %q, want it to mention lct.yaml", stderr)
	}
}

func TestBrokenConfigIsSystemError(t *testing.T) {
	root := inRepo(t)
	if err := os.WriteFile(filepath.Join(root, "lct.yaml"), []byte("lang_dir: [unterminated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	register(t, Command{Name: "t-broken", Run: func(*Context, []string) error { return nil }})

	if code, _, stderr := exec("t-broken"); code != 2 || !strings.Contains(stderr, "lct.yaml") {
		t.Errorf("exit code = %d, stderr = %q; want 2 and a message naming lct.yaml", code, stderr)
	}
}

func TestUnknownCommand(t *testing.T) {
	inRepo(t)
	register(t, Command{Name: "t-known", Usage: "does a thing", Run: func(*Context, []string) error { return nil }})

	for _, args := range [][]string{{"nope"}, {"t-known-not"}, {}} {
		code, stdout, stderr := exec(args...)
		if code != 1 {
			t.Errorf("%v: exit code = %d, want 1", args, code)
		}
		if stdout != "" {
			t.Errorf("%v: stdout = %q, want empty", args, stdout)
		}
		if !strings.Contains(stderr, "t-known") || !strings.Contains(stderr, "does a thing") {
			t.Errorf("%v: stderr = %q, want usage listing t-known", args, stderr)
		}
	}
	if _, _, stderr := exec("nope"); !strings.Contains(stderr, `unknown command "nope"`) {
		t.Errorf("stderr = %q, want it to name the unknown command", stderr)
	}
}

func TestHelp(t *testing.T) {
	// Help must work outside a repo.
	t.Chdir(t.TempDir())
	register(t, Command{Name: "t-known", Usage: "does a thing", Run: func(*Context, []string) error { return nil }})

	for _, arg := range []string{"help", "-h", "--help"} {
		code, stdout, stderr := exec(arg)
		if code != 0 {
			t.Errorf("%s: exit code = %d, want 0", arg, code)
		}
		if stderr != "" {
			t.Errorf("%s: stderr = %q, want empty", arg, stderr)
		}
		if !strings.Contains(stdout, "t-known") || !strings.Contains(stdout, "does a thing") {
			t.Errorf("%s: stdout = %q, want usage listing t-known", arg, stdout)
		}
	}
}

func TestCommandsSorted(t *testing.T) {
	register(t, Command{Name: "t-zz", Run: func(*Context, []string) error { return nil }})
	register(t, Command{Name: "t-aa", Run: func(*Context, []string) error { return nil }})
	register(t, Command{Name: "t-aa sub", Run: func(*Context, []string) error { return nil }})

	names := Commands()
	if !slices.IsSorted(names) {
		t.Errorf("Commands() = %v, want sorted", names)
	}
	for _, want := range []string{"t-aa", "t-aa sub", "t-zz"} {
		if !slices.Contains(names, want) {
			t.Errorf("Commands() = %v, missing %q", names, want)
		}
	}
}

func TestRegisterRejectsBadCommands(t *testing.T) {
	ok := func(*Context, []string) error { return nil }
	register(t, Command{Name: "t-dup", Run: ok})

	tests := []struct {
		name string
		cmd  Command
	}{
		{"empty name", Command{Run: ok}},
		{"nil run", Command{Name: "t-nilrun"}},
		{"duplicate", Command{Name: "t-dup", Run: ok}},
		{"three words", Command{Name: "a b c", Run: ok}},
		{"extra spaces", Command{Name: "a  b", Run: ok}},
		{"flag-like", Command{Name: "--json", Run: ok}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("Register(%+v) did not panic", tc.cmd)
				}
			}()
			Register(tc.cmd)
		})
	}
}

type result struct {
	Name  string `json:"name"`
	Note  string `json:"note"`
	Count int    `json:"count"`
}

type stringerResult struct{ S string }

func (r stringerResult) String() string { return r.S }

func TestEmitJSON(t *testing.T) {
	var buf bytes.Buffer
	ctx := &Context{JSON: true, Out: &buf}
	v := result{Name: "a", Note: "<b> & c", Count: 2}

	if err := ctx.Emit(v); err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"name\": \"a\",\n  \"note\": \"<b> & c\",\n  \"count\": 2\n}\n"
	if buf.String() != want {
		t.Errorf("Emit JSON =\n%q\nwant\n%q", buf.String(), want)
	}

	// Same input must give the same bytes.
	var again bytes.Buffer
	if err := (&Context{JSON: true, Out: &again}).Emit(v); err != nil {
		t.Fatal(err)
	}
	if again.String() != buf.String() {
		t.Errorf("Emit is not deterministic: %q vs %q", again.String(), buf.String())
	}
}

func TestEmitText(t *testing.T) {
	tests := []struct {
		name string
		v    any
		want string
	}{
		{"string", "hello", "hello\n"},
		{"string with newline", "hello\n", "hello\n"},
		{"stringer", stringerResult{"line 1\nline 2"}, "line 1\nline 2\n"},
		{"other value as indented json", result{Name: "a", Note: "<b>", Count: 2},
			"{\n  \"name\": \"a\",\n  \"note\": \"<b>\",\n  \"count\": 2\n}\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := (&Context{Out: &buf}).Emit(tc.v); err != nil {
				t.Fatal(err)
			}
			if buf.String() != tc.want {
				t.Errorf("Emit text = %q, want %q", buf.String(), tc.want)
			}
		})
	}
}

func TestEmitJSONOverridesStringer(t *testing.T) {
	var buf bytes.Buffer
	if err := (&Context{JSON: true, Out: &buf}).Emit(stringerResult{"text"}); err != nil {
		t.Fatal(err)
	}
	if want := "{\n  \"S\": \"text\"\n}\n"; buf.String() != want {
		t.Errorf("Emit JSON of a Stringer = %q, want the JSON encoding %q", buf.String(), want)
	}
}
