package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// command is one CLI subcommand and the MCP tool with the same behavior.
type command struct {
	name    string
	summary string
	// run parses CLI args and writes the result to w.
	run func(ctx context.Context, d *deps, args []string, w io.Writer) error
	// register adds the command to an MCP server as a tool.
	register func(s *mcp.Server, d *deps)
}

// texter is implemented by results that have a human-readable form.
// Results without it print as JSON.
type texter interface {
	text() string
}

// newCommand builds a command from a handler. In's fields become CLI
// flags and MCP tool arguments: the json tag names them, the jsonschema
// tag describes them, and a field tagged arg:"" also takes the CLI's
// positional arguments, joined with spaces so player names need no quotes.
func newCommand[In, Out any](name, summary string, handle func(context.Context, *deps, In) (Out, error)) command {
	return command{
		name:    name,
		summary: summary,
		run: func(ctx context.Context, d *deps, args []string, w io.Writer) error {
			var in In
			asJSON, err := parseFlags(name, summary, &in, args)
			if err != nil {
				return err
			}
			out, err := handle(ctx, d, in)
			if err != nil {
				return err
			}
			if t, ok := any(out).(texter); ok && !asJSON {
				_, err := io.WriteString(w, t.text())
				return err
			}
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
		register: func(s *mcp.Server, d *deps) {
			mcp.AddTool(s, &mcp.Tool{Name: strings.ReplaceAll(name, "-", "_"), Description: summary},
				func(ctx context.Context, _ *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
					out, err := handle(ctx, d, in)
					return nil, out, err
				})
		},
	}
}

// parseFlags fills the struct at in from args and reports whether --json
// was set.
func parseFlags(name, summary string, in any, args []string) (bool, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "print JSON")
	v := reflect.ValueOf(in).Elem()
	var argField *reflect.Value
	var argName string
	var usage []string
	for i := range v.NumField() {
		f := v.Type().Field(i)
		fname, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		fname = strings.ReplaceAll(fname, "_", "-")
		desc := f.Tag.Get("jsonschema")
		field := v.Field(i)
		if _, ok := f.Tag.Lookup("arg"); ok {
			argField, argName = &field, fname
			usage = append(usage, "<"+fname+">")
		}
		switch field.Kind() {
		case reflect.String:
			fs.StringVar(field.Addr().Interface().(*string), fname, "", desc)
		case reflect.Int:
			fs.IntVar(field.Addr().Interface().(*int), fname, 0, desc)
		case reflect.Float64:
			fs.Float64Var(field.Addr().Interface().(*float64), fname, 0, desc)
		case reflect.Bool:
			fs.BoolVar(field.Addr().Interface().(*bool), fname, false, desc)
		default:
			panic(fmt.Sprintf("osrs: unsupported flag type %s for %s", field.Kind(), fname))
		}
	}
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "%s\n\nusage: osrs %s %s [flags]\n\n", summary, name, strings.Join(usage, " "))
		fs.PrintDefaults()
	}
	// Go's flag package stops at the first positional argument, so move
	// flags in front of positionals to allow "osrs hiscores Lynx Titan --json".
	if err := fs.Parse(reorder(fs, args)); err != nil {
		return false, err
	}
	if fs.NArg() > 0 {
		if argField == nil {
			return false, fmt.Errorf("osrs %s: unexpected arguments %q", name, fs.Args())
		}
		if err := setArg(*argField, strings.Join(fs.Args(), " ")); err != nil {
			return false, fmt.Errorf("osrs %s: %s: %w", name, argName, err)
		}
	}
	return *asJSON, nil
}

func setArg(field reflect.Value, s string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(s)
	case reflect.Int:
		n, err := strconv.Atoi(s)
		if err != nil {
			return err
		}
		field.SetInt(int64(n))
	default:
		return errors.New("positional argument must be a string or int")
	}
	return nil
}

// reorder moves flags and their values ahead of positional arguments.
func reorder(fs *flag.FlagSet, args []string) []string {
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(a, "-") || a == "-" {
			positional = append(positional, a)
			continue
		}
		flags = append(flags, a)
		name, _, hasValue := strings.Cut(strings.TrimLeft(a, "-"), "=")
		if f := fs.Lookup(name); f != nil && !hasValue && !isBool(f) && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return append(flags, positional...)
}

func isBool(f *flag.Flag) bool {
	b, ok := f.Value.(interface{ IsBoolFlag() bool })
	return ok && b.IsBoolFlag()
}
