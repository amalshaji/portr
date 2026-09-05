package client

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	clientcfg "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/amalshaji/portr/internal/constants"
)

func TestReportFatalPublishesFirstErrorOnce(t *testing.T) {
	c := &Client{
		exitCh: make(chan error, 1),
	}

	firstErr := errors.New("first failure")
	c.reportFatal(firstErr)
	c.reportFatal(errors.New("second failure"))

	select {
	case err := <-c.Done():
		if !errors.Is(err, firstErr) {
			t.Fatalf("expected first error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for fatal error")
	}

	select {
	case err := <-c.Done():
		t.Fatalf("unexpected extra error: %v", err)
	default:
	}
}

func TestRunFatalWorkerRecoversPanic(t *testing.T) {
	c := &Client{
		exitCh: make(chan error, 1),
	}

	c.runFatalWorker("test worker", func() error {
		panic("boom")
	})

	select {
	case err := <-c.Done():
		if err == nil {
			t.Fatal("expected panic error, got nil")
		}
		if !strings.Contains(err.Error(), "test worker panic: boom") {
			t.Fatalf("unexpected panic error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for panic error")
	}
}

func TestStartHandsWorkersEffectivePoolSize(t *testing.T) {
	cfg := clientcfg.Config{
		ServerUrl:    "127.0.0.1:1",
		UseLocalHost: true,
		DisableTUI:   true,
		Tunnels: []clientcfg.Tunnel{
			{Name: "tcp-test", Type: constants.Tcp, Port: 4321},
		},
	}
	cfg.SetDefaults()

	c := &Client{config: &cfg, exitCh: make(chan error, 1)}
	t.Cleanup(func() {
		c.Shutdown(context.Background())
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := c.Start(ctx); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if len(c.sshcs) != 1 {
		t.Fatalf("expected 1 ssh client, got %d", len(c.sshcs))
	}

	// Pins the call site in Start: the TUI reads PoolSize off the config
	// handed to each worker, so the effective count must be stamped onto
	// clientConfig before sshclient.New, not left as the raw configured value.
	if got := c.sshcs[0].ConfigSnapshot().Tunnel.PoolSize; got != 1 {
		t.Fatalf("expected worker to receive effective pool size 1, got %d", got)
	}
}
