package main

import (
	"slices"
	"strings"

	"github.com/urfave/cli/v2"
)

// urfave/cli stops parsing flags at the first positional argument, so in
// "portr http 8080 --basic-auth admin:admin" the flag would be silently kept
// as an extra argument and the tunnel would start unprotected. Reorder the
// raw arguments so flags come before positionals, letting flags appear
// anywhere on the command line. Commands with subcommands or SkipFlagParsing
// are left untouched, as is anything after a "--" terminator.
func reorderArgs(app *cli.App, args []string) []string {
	cmdIdx := commandIndex(app, args)
	if cmdIdx == -1 {
		return args
	}

	cmd := app.Command(args[cmdIdx])
	if cmd == nil || cmd.SkipFlagParsing || len(cmd.Subcommands) > 0 {
		return args
	}

	rest := args[cmdIdx+1:]
	var flags, positionals []string
	for i := 0; i < len(rest); i++ {
		arg := rest[i]
		if arg == "--" {
			positionals = append(positionals, rest[i:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positionals = append(positionals, arg)
			continue
		}
		flags = append(flags, arg)
		name, hasInlineValue := splitFlagToken(arg)
		if !hasInlineValue && flagTakesValue(cmd.Flags, name) && i+1 < len(rest) {
			i++
			flags = append(flags, rest[i])
		}
	}

	out := make([]string, 0, len(args))
	out = append(out, args[:cmdIdx+1]...)
	out = append(out, flags...)
	out = append(out, positionals...)
	return out
}

// commandIndex returns the index of the command name in args, skipping the
// program name and any global flags (with their values), or -1 if there is
// no command.
func commandIndex(app *cli.App, args []string) int {
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return -1
		}
		if !strings.HasPrefix(arg, "-") {
			return i
		}
		name, hasInlineValue := splitFlagToken(arg)
		if !hasInlineValue && flagTakesValue(app.Flags, name) {
			i++
		}
	}
	return -1
}

func splitFlagToken(arg string) (name string, hasInlineValue bool) {
	name, _, hasInlineValue = strings.Cut(strings.TrimLeft(arg, "-"), "=")
	return name, hasInlineValue
}

// flagTakesValue reports whether the named flag consumes the next argument as
// its value. Unknown flags are treated as valueless; they fail flag parsing
// either way.
func flagTakesValue(flags []cli.Flag, name string) bool {
	for _, f := range flags {
		if slices.Contains(f.Names(), name) {
			_, isBool := f.(*cli.BoolFlag)
			return !isBool
		}
	}
	return false
}
