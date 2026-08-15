package main

import (
	"slices"
	"testing"

	"github.com/urfave/cli/v2"
)

// TestBasicAuthFlagIsRegisteredOnHTTPLikeCommands guards the three commands that
// serve traffic through the client's HTTP reverse proxy. tcp is excluded on
// purpose: Tunnel.Validate rejects basic_auth there.
func TestBasicAuthFlagIsRegisteredOnHTTPLikeCommands(t *testing.T) {
	commands := map[string]*cli.Command{
		"http":  httpCmd(),
		"serve": serveCmd(),
		"stub":  stubCmd(),
	}

	for name, command := range commands {
		t.Run(name, func(t *testing.T) {
			flag := findStringFlag(command, "basic-auth")
			if flag == nil {
				t.Fatalf("%s is missing the --basic-auth flag", name)
			}
			// Without the env var the credential lands in the process table
			// and in shell history.
			if !slices.Contains(flag.EnvVars, "PORTR_BASIC_AUTH") {
				t.Fatalf("%s must accept PORTR_BASIC_AUTH, got %v", name, flag.EnvVars)
			}
		})
	}
}

func TestBasicAuthFlagIsNotOfferedForTcpTunnels(t *testing.T) {
	if findStringFlag(tcpCmd(), "basic-auth") != nil {
		t.Fatal("tcp tunnels cannot enforce basic auth, so the flag must not be offered")
	}
}

func findStringFlag(command *cli.Command, name string) *cli.StringFlag {
	for _, flag := range command.Flags {
		stringFlag, ok := flag.(*cli.StringFlag)
		if ok && stringFlag.Name == name {
			return stringFlag
		}
	}
	return nil
}
