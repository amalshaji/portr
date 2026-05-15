package appserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/amalshaji/portr/internal/constants"
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
	server := httptest.NewServer(NewServer(service, "").Handler())
	defer server.Close()

	body := bytes.NewBufferString(`{"name":"api","type":"http","host":"127.0.0.1","port":3000,"subdomain":"demo","callback_url":"http://example.test/hook"}`)
	resp, err := http.Post(server.URL+"/api/v1/tunnels", "application/json", body)
	if err != nil {
		t.Fatalf("expected request to succeed: %v", err)
	}
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
	server := httptest.NewServer(NewServer(service, "").Handler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/tunnels")
	if err != nil {
		t.Fatalf("expected list request to succeed: %v", err)
	}
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

	req, err := http.NewRequest(http.MethodDelete, server.URL+"/api/v1/tunnels/tun_1", nil)
	if err != nil {
		t.Fatalf("failed to build delete request: %v", err)
	}
	stopResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("expected stop request to succeed: %v", err)
	}
	defer stopResp.Body.Close()

	if stopResp.StatusCode != http.StatusOK {
		t.Fatalf("expected stop status %d, got %d", http.StatusOK, stopResp.StatusCode)
	}
}

func TestServerRequiresBearerTokenWhenConfigured(t *testing.T) {
	service := &fakeTunnelService{}
	server := httptest.NewServer(NewServer(service, "secret").Handler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/health")
	if err != nil {
		t.Fatalf("expected request to succeed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized status, got %d", resp.StatusCode)
	}

	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/health", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer secret")

	authedResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("expected authenticated request to succeed: %v", err)
	}
	defer authedResp.Body.Close()

	if authedResp.StatusCode != http.StatusOK {
		t.Fatalf("expected OK status, got %d", authedResp.StatusCode)
	}
}
