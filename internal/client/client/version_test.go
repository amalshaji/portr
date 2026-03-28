package client

import (
	"testing"

	clientcfg "github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/constants"
)

func TestSupportsHTTPPoolingVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{
			name:    "missing version falls back to single worker",
			version: "",
			want:    false,
		},
		{
			name:    "older server version falls back to single worker",
			version: "0.9.9",
			want:    false,
		},
		{
			name:    "v1 server enables pooling",
			version: "1.0.0",
			want:    true,
		},
		{
			name:    "v-prefixed semver is accepted",
			version: "v1.2.0",
			want:    true,
		},
		{
			name:    "invalid semver falls back to single worker",
			version: "dev",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := supportsHTTPPoolingVersion(tt.version); got != tt.want {
				t.Fatalf("supportsHTTPPoolingVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

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

	if got := desiredWorkers(httpCfg, false); got != 1 {
		t.Fatalf("expected single worker for unsupported pooling, got %d", got)
	}
	if got := desiredWorkers(httpCfg, true); got != 4 {
		t.Fatalf("expected configured pool size when supported, got %d", got)
	}

	tcpCfg := clientcfg.ClientConfig{
		Tunnel: clientcfg.Tunnel{
			Type:     constants.Tcp,
			PoolSize: 4,
		},
	}

	if got := desiredWorkers(tcpCfg, true); got != 1 {
		t.Fatalf("expected tcp tunnels to use a single worker, got %d", got)
	}
}
