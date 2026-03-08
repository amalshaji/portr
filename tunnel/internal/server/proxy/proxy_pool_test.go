package proxy

import (
	"testing"

	serverConfig "github.com/amalshaji/portr/internal/server/config"
)

func TestProxy_AddBackendAndRoundRobin(t *testing.T) {
	cfg := &serverConfig.Config{}
	p := New(cfg)

	sub := "sub"
	if err := p.AddBackend(sub, "127.0.0.1:1000"); err != nil {
		t.Fatalf("add backend 1: %v", err)
	}
	if err := p.AddBackend(sub, "127.0.0.1:1001"); err != nil {
		t.Fatalf("add backend 2: %v", err)
	}
	if err := p.AddBackend(sub, "127.0.0.1:1002"); err != nil {
		t.Fatalf("add backend 3: %v", err)
	}

	// Round robin over three backends
	got1, _ := p.GetNextBackend(sub)
	got2, _ := p.GetNextBackend(sub)
	got3, _ := p.GetNextBackend(sub)
	got4, _ := p.GetNextBackend(sub)

	if got1 != "127.0.0.1:1000" || got2 != "127.0.0.1:1001" || got3 != "127.0.0.1:1002" || got4 != "127.0.0.1:1000" {
		t.Fatalf("round robin order unexpected: %v, %v, %v, %v", got1, got2, got3, got4)
	}
}

func TestProxy_RemoveBackend(t *testing.T) {
	cfg := &serverConfig.Config{}
	p := New(cfg)
	sub := "sub"
	_ = p.AddBackend(sub, "127.0.0.1:1000")
	_ = p.AddBackend(sub, "127.0.0.1:1001")

	// Remove one backend and ensure the next is consistent
	if err := p.RemoveBackend(sub, "127.0.0.1:1000"); err != nil {
		t.Fatalf("remove backend: %v", err)
	}

	got, err := p.GetNextBackend(sub)
	if err != nil {
		t.Fatalf("get next backend: %v", err)
	}
	if got != "127.0.0.1:1001" {
		t.Fatalf("expected remaining backend, got %v", got)
	}

	// Remove the last backend; follow-up call should error
	if err := p.RemoveBackend(sub, "127.0.0.1:1001"); err != nil {
		t.Fatalf("remove last backend: %v", err)
	}
	if _, err := p.GetNextBackend(sub); err == nil {
		t.Fatalf("expected error after removing all backends")
	}
}
