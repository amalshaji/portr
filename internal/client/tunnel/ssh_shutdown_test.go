package tunnel

import (
	"context"
	"testing"
	"time"

	clientcfg "github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/constants"
)

func TestShutdownEmitsStoppedEventWithoutDeadlock(t *testing.T) {
	client := New(clientcfg.ClientConfig{
		UseLocalHost: true,
		TunnelUrl:    "example.test",
		Tunnel: clientcfg.Tunnel{
			Type:      constants.Http,
			Host:      "localhost",
			Port:      3000,
			Subdomain: "demo",
		},
	}, nil, nil, nil)

	eventCh := make(chan Event, 1)
	client.SetEventHandler(func(event Event) {
		eventCh <- event
	})

	done := make(chan error, 1)
	go func() {
		done <- client.Shutdown(context.Background())
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected shutdown without error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown deadlocked")
	}

	select {
	case event := <-eventCh:
		if event.Type != EventStopped {
			t.Fatalf("expected stopped event, got %q", event.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("expected stopped event")
	}
}
