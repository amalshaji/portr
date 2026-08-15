package main

import (
	"slices"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestReorderArgs(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{
			"flags after positional move before it",
			[]string{"portr", "http", "8080", "--basic-auth", "admin:admin"},
			[]string{"portr", "http", "--basic-auth", "admin:admin", "8080"},
		},
		{
			"flags before positional stay put",
			[]string{"portr", "http", "--basic-auth", "admin:admin", "8080"},
			[]string{"portr", "http", "--basic-auth", "admin:admin", "8080"},
		},
		{
			"mixed flags collect in order",
			[]string{"portr", "http", "-s", "foo", "8080", "--basic-auth", "admin:admin"},
			[]string{"portr", "http", "-s", "foo", "--basic-auth", "admin:admin", "8080"},
		},
		{
			"inline value form",
			[]string{"portr", "serve", "./dist", "--basic-auth=admin:admin"},
			[]string{"portr", "serve", "--basic-auth=admin:admin", "./dist"},
		},
		{
			"global flags before command are skipped over",
			[]string{"portr", "-c", "portr.yaml", "http", "8080", "-s", "foo"},
			[]string{"portr", "-c", "portr.yaml", "http", "-s", "foo", "8080"},
		},
		{
			"everything after -- stays positional",
			[]string{"portr", "http", "-s", "foo", "8080", "--", "--basic-auth", "x"},
			[]string{"portr", "http", "-s", "foo", "8080", "--", "--basic-auth", "x"},
		},
		{
			"unknown flag moves alone so parsing fails loudly",
			[]string{"portr", "tcp", "5432", "--subdomain", "my-postgres"},
			[]string{"portr", "tcp", "--subdomain", "5432", "my-postgres"},
		},
		{
			"commands with subcommands untouched",
			[]string{"portr", "config", "edit", "--something"},
			[]string{"portr", "config", "edit", "--something"},
		},
		{
			"SkipFlagParsing commands untouched",
			[]string{"portr", "logs", "foo", "--json"},
			[]string{"portr", "logs", "foo", "--json"},
		},
		{
			"no command untouched",
			[]string{"portr", "--version"},
			[]string{"portr", "--version"},
		},
	}

	app := buildApp()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := reorderArgs(app, tc.in)
			if !slices.Equal(got, tc.want) {
				t.Fatalf("reorderArgs(%v)\n got %v\nwant %v", tc.in, got, tc.want)
			}
		})
	}
}

// urfave/cli stops parsing flags at the first positional argument, so without
// reordering, "portr http 8080 --basic-auth admin:admin" would start an
// unprotected tunnel with no warning. Verify the reordered args parse into
// flag values end to end.
func TestReorderedArgsParse(t *testing.T) {
	var basicAuth, port string
	app := &cli.App{
		Commands: []*cli.Command{{
			Name: "http",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "basic-auth"},
				&cli.StringFlag{Name: "subdomain", Aliases: []string{"s"}},
			},
			Action: func(c *cli.Context) error {
				basicAuth = c.String("basic-auth")
				port = c.Args().First()
				return nil
			},
		}},
	}

	args := reorderArgs(app, []string{"portr", "http", "8080", "--basic-auth", "admin:admin"})
	if err := app.Run(args); err != nil {
		t.Fatalf("app.Run failed: %v", err)
	}
	if basicAuth != "admin:admin" {
		t.Fatalf("basic-auth = %q, want %q", basicAuth, "admin:admin")
	}
	if port != "8080" {
		t.Fatalf("port = %q, want %q", port, "8080")
	}
}
