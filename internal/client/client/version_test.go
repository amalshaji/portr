package client

import (
	"testing"

	clientcfg "github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/constants"
)

func TestServerBaseURL(t *testing.T) {
	if got := serverBaseURL("localhost:8001", true); got != "http://localhost:8001" {
		t.Fatalf("expected localhost server url to use http, got %q", got)
	}
	if got := serverBaseURL("https://portr.dev", false); got != "https://portr.dev" {
		t.Fatalf("expected schemed url to stay unchanged, got %q", got)
	}
}

func TestDesiredWorkers(t *testing.T) {
	httpCfg := clientcfg.ClientConfig{
		Tunnel: clientcfg.Tunnel{
			Type:     constants.Http,
			PoolSize: 4,
		},
	}

	if got := desiredWorkers(httpCfg); got != 4 {
		t.Fatalf("expected configured pool size when supported, got %d", got)
	}

	tcpCfg := clientcfg.ClientConfig{
		Tunnel: clientcfg.Tunnel{
			Type:     constants.Tcp,
			PoolSize: 4,
		},
	}

	if got := desiredWorkers(tcpCfg); got != 1 {
		t.Fatalf("expected tcp tunnels to use a single worker, got %d", got)
	}
}
