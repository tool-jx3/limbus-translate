// Package cli is the command dispatcher of lct. Commands register themselves
// with Register (usually from an init function in cmd/lct) and Main runs them.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/tool-jx3/limbus-translate/internal/config"
)

// Command is one lct subcommand. Name is one or two space-separated words,
// such as "snapshot" or "batch next".
type Command struct {
	Name  string
	Usage string
	Run   func(ctx *Context, args []string) error
}

// Context is what every command receives.
type Context struct {
	Root   string
	Config *config.Config
	JSON   bool
	Out    io.Writer
}

// ErrFindings is returned by a command that ran fine but found problems
// (exit code 1). The command is expected to have reported the findings itself.
var ErrFindings = errors.New("findings reported")

type userError string

func (e userError) Error() string { return string(e) }

// UserError returns an error for bad input or usage (exit code 1). Any other
// error that is not ErrFindings is a system error (exit code 2), unless it
// (or an error it wraps) has a method UserError() bool that returns true: a
// package that owns a file people write can say so without importing cli.
func UserError(msg string) error { return userError(msg) }

var registry = map[string]Command{}

// Register adds cmd to the command table. It panics on a malformed or
// duplicate command, since that is a programming error.
func Register(cmd Command) {
	words := strings.Split(cmd.Name, " ")
	if len(words) > 2 || slices.Contains(words, "") || strings.HasPrefix(cmd.Name, "-") {
		panic(fmt.Sprintf("cli: invalid command name %q", cmd.Name))
	}
	if cmd.Run == nil {
		panic(fmt.Sprintf("cli: command %q has no Run", cmd.Name))
	}
	if _, dup := registry[cmd.Name]; dup {
		panic(fmt.Sprintf("cli: command %q registered twice", cmd.Name))
	}
	registry[cmd.Name] = cmd
}

// Commands returns the registered command names in sorted order.
func Commands() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// Main runs the lct command named by args and returns the process exit code:
// 0 on success, 1 for findings or user errors, 2 for system errors.
func Main(args []string) int {
	return run(args, os.Stdout, os.Stderr)
}

func run(args []string, stdout, stderr io.Writer) int {
	jsonOut := false
	rest := make([]string, 0, len(args))
	for _, a := range args {
		if a == "--json" {
			jsonOut = true
		} else {
			rest = append(rest, a)
		}
	}

	if len(rest) == 1 && slices.Contains([]string{"help", "-h", "--help"}, rest[0]) {
		usage(stdout)
		return 0
	}

	cmd, cmdArgs, ok := lookup(rest)
	if !ok {
		if len(rest) == 0 {
			fmt.Fprintln(stderr, "lct: missing command")
		} else {
			fmt.Fprintf(stderr, "lct: unknown command %q\n", rest[0])
		}
		usage(stderr)
		return 1
	}

	ctx, err := newContext(jsonOut, stdout)
	if err == nil {
		err = cmd.Run(ctx, cmdArgs)
	}
	return exitCode(err, stderr)
}

// lookup finds the command for args, preferring a two-word name over a
// one-word name, and returns the arguments left over.
func lookup(args []string) (cmd Command, rest []string, ok bool) {
	if len(args) >= 2 {
		if cmd, ok = registry[args[0]+" "+args[1]]; ok {
			return cmd, args[2:], true
		}
	}
	if len(args) >= 1 {
		if cmd, ok = registry[args[0]]; ok {
			return cmd, args[1:], true
		}
	}
	return Command{}, nil, false
}

func newContext(jsonOut bool, out io.Writer) (*Context, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	root, err := config.FindRoot(wd)
	if errors.Is(err, config.ErrRootNotFound) {
		return nil, UserError(fmt.Sprintf("not inside an lct repo: no %s in %s or any parent directory", config.FileName, wd))
	}
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return nil, err
	}
	return &Context{Root: root, Config: cfg, JSON: jsonOut, Out: out}, nil
}

func exitCode(err error, stderr io.Writer) int {
	if err == nil {
		return 0
	}
	// A bare ErrFindings means the command already reported what it found.
	if err != ErrFindings {
		fmt.Fprintf(stderr, "lct: %v\n", err)
	}
	var ue userError
	var uer interface{ UserError() bool }
	if errors.Is(err, ErrFindings) || errors.As(err, &ue) || (errors.As(err, &uer) && uer.UserError()) {
		return 1
	}
	return 2
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "Usage: lct [--json] <command> [args]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, name := range Commands() {
		fmt.Fprintf(tw, "  %s\t%s\n", name, registry[name].Usage)
	}
	tw.Flush()
}

// Emit writes v to ctx.Out. With --json it writes indented JSON, otherwise
// strings and fmt.Stringers are written as text and anything else as indented
// JSON, which is also readable by people.
func (ctx *Context) Emit(v any) error {
	if !ctx.JSON {
		switch t := v.(type) {
		case string:
			return writeLine(ctx.Out, t)
		case fmt.Stringer:
			return writeLine(ctx.Out, t.String())
		}
	}
	enc := json.NewEncoder(ctx.Out)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeLine(w io.Writer, s string) error {
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	_, err := io.WriteString(w, s)
	return err
}
