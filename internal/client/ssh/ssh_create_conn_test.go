package ssh

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	clientcfg "github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/constants"
	"github.com/go-resty/resty/v2"
)

func TestCreateNewConnection_Success(t *testing.T) {
	newRestyClient = func() *resty.Client {
		return resty.NewWithClient(&http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodPost || !strings.HasPrefix(r.URL.Path, "/api/v1/connections") {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				body, _ := json.Marshal(map[string]string{"connection_id": "abc123"})
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(string(body))),
				}, nil
			}),
		})
	}
	defer func() { newRestyClient = resty.New }()

	cfg := clientcfg.ClientConfig{
		ServerUrl:    "localhost:8001",
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
	newRestyClient = func() *resty.Client {
		return resty.NewWithClient(&http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				body, _ := json.Marshal(map[string]string{"message": "bad"})
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(string(body))),
				}, nil
			}),
		})
	}
	defer func() { newRestyClient = resty.New }()

	cfg := clientcfg.ClientConfig{
		ServerUrl:    "localhost:8001",
		SecretKey:    "sk",
		UseLocalHost: true,
		Tunnel:       clientcfg.Tunnel{Type: constants.Http, Subdomain: "x"},
	}

	_, err := CreateNewConnection(cfg)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
