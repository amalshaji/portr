package ssh

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	clientcfg "github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/constants"
)

func TestCreateNewConnection_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasPrefix(r.URL.Path, "/api/v1/connections") {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"connection_id": "abc123"})
	}))
	defer ts.Close()

	cfg := clientcfg.ClientConfig{
		ServerUrl:    strings.TrimPrefix(ts.URL, "http://"),
		SecretKey:    "sk",
		UseLocalHost: true,
		Tunnel: clientcfg.Tunnel{
			Type:      constants.Http,
			Subdomain: "test",
		},
	}

	id, err := CreateNewConnection(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != "abc123" {
		t.Fatalf("expected connection id 'abc123', got '%s'", id)
	}
}

func TestCreateNewConnection_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "bad"})
	}))
	defer ts.Close()

	cfg := clientcfg.ClientConfig{
		ServerUrl:    strings.TrimPrefix(ts.URL, "http://"),
		SecretKey:    "sk",
		UseLocalHost: true,
		Tunnel:       clientcfg.Tunnel{Type: constants.Http, Subdomain: "x"},
	}

	_, err := CreateNewConnection(cfg)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
