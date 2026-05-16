package ssh

import (
	"context"
	"encoding/json"
	"errors"
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

func TestCreateNewConnectionWithContext_Canceled(t *testing.T) {
	newRestyClient = func() *resty.Client {
		return resty.NewWithClient(&http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				<-r.Context().Done()
				return nil, r.Context().Err()
			}),
		})
	}
	defer func() { newRestyClient = resty.New }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := clientcfg.ClientConfig{
		ServerUrl:    "localhost:8001",
		SecretKey:    "sk",
		UseLocalHost: true,
		Tunnel:       clientcfg.Tunnel{Type: constants.Http, Subdomain: "x"},
	}

	_, err := CreateNewConnectionWithContext(ctx, cfg)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled error, got %v", err)
	}
}

func TestStartReturnsServerConnectionError(t *testing.T) {
	newRestyClient = func() *resty.Client {
		return resty.NewWithClient(&http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				body, _ := json.Marshal(map[string]string{"message": "Invalid secret key"})
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(string(body))),
				}, nil
			}),
		})
	}
	defer func() { newRestyClient = resty.New }()

	client := New(clientcfg.ClientConfig{
		ServerUrl:           "localhost:8001",
		SecretKey:           "bad",
		UseLocalHost:        true,
		DisableTerminalLogs: true,
		Tunnel: clientcfg.Tunnel{
			Name:      "demo",
			Type:      constants.Http,
			Host:      "localhost",
			Port:      3000,
			Subdomain: "demo",
		},
	}, nil, nil, nil)

	err := client.Start(context.Background())
	if err == nil {
		t.Fatal("expected startup error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to start tunnel 'demo': server error: Invalid secret key") {
		t.Fatalf("unexpected startup error: %v", err)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
