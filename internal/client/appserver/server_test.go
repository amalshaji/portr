package appserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/amalshaji/portr/internal/constants"
	"github.com/gofiber/fiber/v2"
)

type fakeTunnelService struct {
	startRequest StartTunnelRequest
	startStatus  TunnelStatus
	tunnels      []TunnelStatus
	events       []TunnelEvent
}

func (f *fakeTunnelService) StartTunnel(_ context.Context, request StartTunnelRequest) (TunnelStatus, error) {
	f.startRequest = request
	if f.startStatus.ID == "" {
		f.startStatus = TunnelStatus{
			ID:        "tun_1",
			Status:    statusRunning,
			Type:      request.Type,
			Host:      request.Host,
			Port:      request.Port,
			Subdomain: request.Subdomain,
			StartedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
	}
	return f.startStatus, nil
}

func (f *fakeTunnelService) ListTunnels() []TunnelStatus {
	return f.tunnels
}

func (f *fakeTunnelService) GetTunnel(id string) (TunnelStatus, error) {
	for _, tunnel := range f.tunnels {
		if tunnel.ID == id {
			return tunnel, nil
		}
	}
	return TunnelStatus{}, ErrTunnelNotFound
}

func (f *fakeTunnelService) StopTunnel(_ context.Context, id string) (TunnelStatus, error) {
	for _, tunnel := range f.tunnels {
		if tunnel.ID == id {
			tunnel.Status = statusStopped
			return tunnel, nil
		}
	}
	return TunnelStatus{}, ErrTunnelNotFound
}

func (f *fakeTunnelService) Events(tunnelID string) []TunnelEvent {
	if tunnelID == "" {
		return f.events
	}
	filtered := make([]TunnelEvent, 0)
	for _, event := range f.events {
		if event.TunnelID == tunnelID {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func TestServerCreatesTunnel(t *testing.T) {
	service := &fakeTunnelService{}
	app := NewServer(service, "").App()

	body := bytes.NewBufferString(`{"name":"api","type":"http","host":"127.0.0.1","port":3000,"subdomain":"demo","callback_url":"http://example.test/hook"}`)
	req := newRequest(t, http.MethodPost, "/api/v1/tunnels", body)
	req.Header.Set("Content-Type", "application/json")
	resp := doRequest(t, app, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}
	if service.startRequest.Name != "api" {
		t.Fatalf("expected request to be passed to service, got %#v", service.startRequest)
	}
	if service.startRequest.Type != constants.Http {
		t.Fatalf("expected http tunnel type, got %q", service.startRequest.Type)
	}
	if service.startRequest.CallbackURL != "http://example.test/hook" {
		t.Fatalf("expected callback URL to be preserved")
	}
}

func TestServerListsAndStopsTunnels(t *testing.T) {
	service := &fakeTunnelService{
		tunnels: []TunnelStatus{{
			ID:        "tun_1",
			Status:    statusRunning,
			Type:      constants.Tcp,
			Host:      "localhost",
			Port:      5432,
			StartedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}},
	}
	app := NewServer(service, "").App()

	resp := doRequest(t, app, newRequest(t, http.MethodGet, "/api/v1/tunnels", nil))
	defer resp.Body.Close()

	var listBody struct {
		Tunnels []TunnelStatus `json:"tunnels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listBody); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}
	if len(listBody.Tunnels) != 1 || listBody.Tunnels[0].ID != "tun_1" {
		t.Fatalf("unexpected tunnels response: %#v", listBody)
	}

	stopResp := doRequest(t, app, newRequest(t, http.MethodDelete, "/api/v1/tunnels/tun_1", nil))
	defer stopResp.Body.Close()

	if stopResp.StatusCode != http.StatusOK {
		t.Fatalf("expected stop status %d, got %d", http.StatusOK, stopResp.StatusCode)
	}
}

func TestServerRequiresBearerTokenWhenConfigured(t *testing.T) {
	service := &fakeTunnelService{}
	app := NewServer(service, "secret").App()

	resp := doRequest(t, app, newRequest(t, http.MethodGet, "/api/v1/health", nil))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized status, got %d", resp.StatusCode)
	}

	req := newRequest(t, http.MethodGet, "/api/v1/health", nil)
	req.Header.Set("Authorization", "Bearer secret")

	authedResp := doRequest(t, app, req)
	defer authedResp.Body.Close()

	if authedResp.StatusCode != http.StatusOK {
		t.Fatalf("expected OK status, got %d", authedResp.StatusCode)
	}
}

func newRequest(t *testing.T, method, target string, body io.Reader) *http.Request {
	t.Helper()

	req, err := http.NewRequest(method, target, body)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	return req
}

func doRequest(t *testing.T, app *fiber.App, req *http.Request) *http.Response {
	t.Helper()

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}
