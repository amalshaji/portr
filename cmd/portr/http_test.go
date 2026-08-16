package main

import (
	"flag"
	"testing"

	"github.com/amalshaji/portr/internal/constants"
	"github.com/urfave/cli/v2"
)

func TestHttpTunnelFromContextUsesSubdomainFlag(t *testing.T) {
	set := flag.NewFlagSet("http", flag.ContinueOnError)
	set.String("subdomain", "", "")
	if err := set.Parse([]string{"--subdomain", "demo", "5999"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	tunnel, err := httpTunnelFromContext(cli.NewContext(nil, set, nil))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if tunnel.Port != 5999 {
		t.Fatalf("expected port 5999, got %d", tunnel.Port)
	}
	if tunnel.Subdomain != "demo" {
		t.Fatalf("expected subdomain demo, got %q", tunnel.Subdomain)
	}
	if tunnel.Type != constants.Http {
		t.Fatalf("expected http tunnel type, got %q", tunnel.Type)
	}
}

func TestHttpTunnelFromContextSupportsSubdomainFlagAfterPort(t *testing.T) {
	set := flag.NewFlagSet("http", flag.ContinueOnError)
	set.String("subdomain", "", "")
	if err := set.Parse([]string{"5999", "-s", "demo"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	tunnel, err := httpTunnelFromContext(cli.NewContext(nil, set, nil))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if tunnel.Port != 5999 {
		t.Fatalf("expected port 5999, got %d", tunnel.Port)
	}
	if tunnel.Subdomain != "demo" {
		t.Fatalf("expected subdomain demo, got %q", tunnel.Subdomain)
	}
}
