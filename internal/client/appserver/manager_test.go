package appserver

import (
	"testing"

	clientcfg "github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/constants"
)

func TestNormalizeCallbackURLsDeduplicates(t *testing.T) {
	got := normalizeCallbackURLs("http://example.test/a", []string{"http://example.test/a", " http://example.test/b "})
	if len(got) != 2 {
		t.Fatalf("expected 2 callback URLs, got %#v", got)
	}
	if got[0] != "http://example.test/a" || got[1] != "http://example.test/b" {
		t.Fatalf("unexpected callback URLs: %#v", got)
	}
}

func TestValidateTunnelRequestRejectsInvalidPort(t *testing.T) {
	tunnel := clientcfg.Tunnel{
		Type: constants.Http,
		Port: 0,
	}
	tunnel.SetDefaults()

	if err := validateTunnelRequest(tunnel, "", nil); err == nil {
		t.Fatal("expected invalid port error")
	}
}

func TestValidateTunnelRequestRejectsInvalidCallbackURL(t *testing.T) {
	tunnel := clientcfg.Tunnel{
		Type: constants.Tcp,
		Port: 5432,
	}
	tunnel.SetDefaults()

	if err := validateTunnelRequest(tunnel, "ftp://example.test/hook", nil); err == nil {
		t.Fatal("expected invalid callback URL error")
	}
}

func TestSupportsHTTPPoolingVersion(t *testing.T) {
	if !supportsHTTPPoolingVersion("v1.0.0") {
		t.Fatal("expected v1.0.0 to support HTTP pooling")
	}
	if supportsHTTPPoolingVersion("0.9.9") {
		t.Fatal("expected 0.9.9 to disable HTTP pooling")
	}
	if supportsHTTPPoolingVersion("not-a-version") {
		t.Fatal("expected invalid versions to disable HTTP pooling")
	}
}
