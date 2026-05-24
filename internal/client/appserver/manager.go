package appserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver"
	clientcfg "github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/client/db"
	"github.com/amalshaji/portr/internal/client/stubresponder"
	tunnelclient "github.com/amalshaji/portr/internal/client/tunnel"
	"github.com/amalshaji/portr/internal/constants"
	"github.com/charmbracelet/log"
	"github.com/oklog/ulid/v2"
)

const (
	statusStarting  = "starting"
	statusRunning   = "running"
	statusUnhealthy = "unhealthy"
	statusStopped   = "stopped"
	statusFailed    = "failed"

	maxStoredEvents = 200
)

var ErrTunnelNotFound = errors.New("tunnel not found")

type tunnelRuntime struct {
	id           string
	cancel       context.CancelFunc
	clients      []*tunnelclient.Client
	callbackURLs []string
	status       TunnelStatus
	startedCh    chan struct{}
	failedCh     chan error
	startOnce    sync.Once
	failOnce     sync.Once
	stopping     bool
}

type Manager struct {
	baseConfig clientcfg.Config
	db         *db.Db
	httpClient *http.Client
	logger     *log.Logger

	mu            sync.RWMutex
	stubMu        sync.Mutex
	stubResponder *stubresponder.Responder
	tunnels       map[string]*tunnelRuntime
	events        []TunnelEvent
}

func NewManager(baseConfig clientcfg.Config, database *db.Db) *Manager {
	baseConfig.DisableTUI = true
	baseConfig.DisableDashboard = true

	return &Manager{
		baseConfig: baseConfig,
		db:         database,
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
		logger:  log.WithPrefix("app-server"),
		tunnels: make(map[string]*tunnelRuntime),
		events:  make([]TunnelEvent, 0),
	}
}

func (m *Manager) StartTunnel(ctx context.Context, request StartTunnelRequest) (TunnelStatus, error) {
	tunnel := clientcfg.Tunnel{
		Name:                 request.Name,
		Subdomain:            request.Subdomain,
		Port:                 request.Port,
		Host:                 request.Host,
		Type:                 request.Type,
		PoolSize:             request.PoolSize,
		ResponseFormat:       request.ResponseFormat,
		ResponseTemplate:     request.ResponseTemplate,
		ResponseTemplateFile: request.ResponseTemplateFile,
	}
	tunnel.SetDefaults()
	if err := tunnel.ResolveStubTemplate("."); err != nil {
		return TunnelStatus{}, err
	}
	if err := validateTunnelRequest(tunnel, request.CallbackURL, request.CallbackURLs); err != nil {
		return TunnelStatus{}, err
	}

	statusTunnel := tunnel
	if err := m.prepareStubTunnel(&tunnel); err != nil {
		return TunnelStatus{}, err
	}

	cfg := m.clientConfigForTunnel(tunnel)
	workers := m.desiredWorkers(cfg)
	if cfg.Tunnel.Type == constants.Http && workers > 1 && cfg.ConnectionID == "" {
		connID, err := tunnelclient.CreateNewConnectionWithContext(ctx, cfg)
		if err != nil {
			return TunnelStatus{}, fmt.Errorf("failed to create shared connection for pool: %w", err)
		}
		cfg.ConnectionID = connID
	}

	id := ulid.Make().String()
	runCtx, cancel := context.WithCancel(context.Background())
	callbackURLs := normalizeCallbackURLs(request.CallbackURL, request.CallbackURLs)
	now := time.Now().UTC()
	runtime := &tunnelRuntime{
		id:           id,
		cancel:       cancel,
		callbackURLs: callbackURLs,
		status: TunnelStatus{
			ID:             id,
			Name:           tunnel.Name,
			Status:         statusStarting,
			Type:           tunnel.Type,
			Host:           statusTunnel.Host,
			Port:           statusTunnel.Port,
			Subdomain:      tunnel.Subdomain,
			ResponseFormat: tunnel.ResponseFormat,
			CallbackURLs:   callbackURLs,
			StartedAt:      now,
			UpdatedAt:      now,
		},
		startedCh: make(chan struct{}),
		failedCh:  make(chan error, 1),
	}

	for i := 0; i < workers; i++ {
		workerCfg := cfg
		tunnelClient := tunnelclient.New(workerCfg, m.db, nil, nil)
		tunnelClient.SetEventHandler(func(event tunnelclient.Event) {
			m.handleTunnelEvent(id, event)
		})
		runtime.clients = append(runtime.clients, tunnelClient)
	}

	m.mu.Lock()
	m.tunnels[id] = runtime
	m.mu.Unlock()

	for _, tunnelClient := range runtime.clients {
		go func(client *tunnelclient.Client) {
			if err := client.Start(runCtx); err != nil {
				m.handleStartFailure(id, err)
			}
		}(tunnelClient)
	}

	select {
	case <-runtime.startedCh:
		return m.GetTunnel(id)
	case err := <-runtime.failedCh:
		cancel()
		m.unregisterStubTunnel(tunnel)
		return TunnelStatus{}, err
	case <-time.After(7 * time.Second):
		return m.GetTunnel(id)
	case <-ctx.Done():
		cancel()
		m.unregisterStubTunnel(tunnel)
		return TunnelStatus{}, ctx.Err()
	}
}

func (m *Manager) ListTunnels() []TunnelStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make([]TunnelStatus, 0, len(m.tunnels))
	for _, tunnel := range m.tunnels {
		statuses = append(statuses, tunnel.status)
	}

	slices.SortFunc(statuses, func(a, b TunnelStatus) int {
		return b.StartedAt.Compare(a.StartedAt)
	})
	return statuses
}

func (m *Manager) GetTunnel(id string) (TunnelStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tunnel, ok := m.tunnels[id]
	if !ok {
		return TunnelStatus{}, ErrTunnelNotFound
	}
	return tunnel.status, nil
}

func (m *Manager) StopTunnel(ctx context.Context, id string) (TunnelStatus, error) {
	m.mu.RLock()
	tunnel, ok := m.tunnels[id]
	m.mu.RUnlock()
	if !ok {
		return TunnelStatus{}, ErrTunnelNotFound
	}

	m.mu.Lock()
	tunnel.stopping = true
	m.mu.Unlock()

	tunnel.cancel()
	for _, client := range tunnel.clients {
		_ = client.Shutdown(ctx)
	}
	m.unregisterStubTunnel(clientcfg.Tunnel{
		Type:      tunnel.status.Type,
		Subdomain: tunnel.status.Subdomain,
	})

	now := time.Now().UTC()
	m.mu.Lock()
	tunnel.status.Status = statusStopped
	tunnel.status.UpdatedAt = now
	tunnel.status.StoppedAt = &now
	status := tunnel.status
	m.mu.Unlock()

	return status, nil
}

func (m *Manager) prepareStubTunnel(tunnel *clientcfg.Tunnel) error {
	if tunnel.Type != constants.Stub {
		return nil
	}

	m.stubMu.Lock()
	defer m.stubMu.Unlock()

	if m.stubResponder == nil {
		responder := stubresponder.New()
		if err := responder.Start(); err != nil {
			return err
		}
		m.stubResponder = responder
	}

	if err := m.stubResponder.Register(stubresponder.Route{
		Subdomain:        tunnel.Subdomain,
		ResponseFormat:   tunnel.ResponseFormat,
		ResponseTemplate: tunnel.ResponseTemplate,
	}); err != nil {
		return err
	}

	tunnel.Host = "127.0.0.1"
	tunnel.Port = m.stubResponder.Port()
	tunnel.PoolSize = 1
	return nil
}

func (m *Manager) unregisterStubTunnel(tunnel clientcfg.Tunnel) {
	if tunnel.Type != constants.Stub {
		return
	}

	m.stubMu.Lock()
	defer m.stubMu.Unlock()

	if m.stubResponder != nil {
		m.stubResponder.Unregister(tunnel.Subdomain)
	}
}

func (m *Manager) Events(tunnelID string) []TunnelEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	events := make([]TunnelEvent, 0, len(m.events))
	for _, event := range m.events {
		if tunnelID == "" || event.TunnelID == tunnelID {
			events = append(events, event)
		}
	}
	return events
}

func (m *Manager) Shutdown(ctx context.Context) {
	m.mu.RLock()
	ids := make([]string, 0, len(m.tunnels))
	for id := range m.tunnels {
		ids = append(ids, id)
	}
	m.mu.RUnlock()

	for _, id := range ids {
		_, _ = m.StopTunnel(ctx, id)
	}

	m.stubMu.Lock()
	if m.stubResponder != nil {
		_ = m.stubResponder.Shutdown(ctx)
		m.stubResponder = nil
	}
	m.stubMu.Unlock()
}

func (m *Manager) clientConfigForTunnel(tunnel clientcfg.Tunnel) clientcfg.ClientConfig {
	return clientcfg.ClientConfig{
		ServerUrl:              m.baseConfig.ServerUrl,
		WsUrl:                  m.baseConfig.WsUrl,
		TunnelUrl:              m.baseConfig.TunnelUrl,
		SecretKey:              m.baseConfig.SecretKey,
		Tunnel:                 tunnel,
		UseLocalHost:           m.baseConfig.UseLocalHost,
		Debug:                  m.baseConfig.Debug,
		EnableRequestLogging:   *m.baseConfig.EnableRequestLogging,
		HealthCheckInterval:    m.baseConfig.HealthCheckInterval,
		HealthCheckMaxRetries:  m.baseConfig.HealthCheckMaxRetries,
		DisableTUI:             true,
		DisableTerminalLogs:    true,
		EnableHttpReverseProxy: m.baseConfig.EnableHttpReverseProxy,
	}
}

func (m *Manager) handleStartFailure(id string, err error) {
	m.mu.RLock()
	tunnel, ok := m.tunnels[id]
	m.mu.RUnlock()
	if !ok {
		return
	}

	m.unregisterStubTunnel(clientcfg.Tunnel{
		Type:      tunnel.status.Type,
		Subdomain: tunnel.status.Subdomain,
	})

	tunnel.failOnce.Do(func() {
		select {
		case tunnel.failedCh <- err:
		default:
		}
	})

	m.handleTunnelEvent(id, tunnelclient.Event{
		Type:  tunnelclient.EventFailed,
		Error: err.Error(),
		At:    time.Now().UTC(),
	})
}

func (m *Manager) handleTunnelEvent(id string, event tunnelclient.Event) {
	m.mu.RLock()
	tunnel, ok := m.tunnels[id]
	m.mu.RUnlock()
	if !ok {
		return
	}

	now := event.At
	if now.IsZero() {
		now = time.Now().UTC()
		event.At = now
	}

	shouldRecord := true
	m.mu.Lock()
	if (tunnel.stopping || tunnel.status.Status == statusStopped) && event.Type != tunnelclient.EventStopped {
		m.mu.Unlock()
		return
	}
	switch event.Type {
	case tunnelclient.EventStarted:
		if tunnel.status.Status == statusRunning {
			shouldRecord = false
			break
		}
		tunnel.status.Status = statusRunning
		tunnel.status.RemotePort = event.Tunnel.RemotePort
		tunnel.status.TunnelURL = event.TunnelAddr
		tunnel.status.LastError = ""
		tunnel.startOnce.Do(func() {
			close(tunnel.startedCh)
		})
	case tunnelclient.EventUnhealthy:
		if tunnel.status.Status == statusUnhealthy && tunnel.status.LastError == event.Error {
			shouldRecord = false
			break
		}
		tunnel.status.Status = statusUnhealthy
		tunnel.status.LastError = event.Error
	case tunnelclient.EventReconnected:
		tunnel.status.Status = statusRunning
		tunnel.status.RemotePort = event.Tunnel.RemotePort
		tunnel.status.TunnelURL = event.TunnelAddr
		tunnel.status.LastError = ""
	case tunnelclient.EventStopped:
		if tunnel.status.Status == statusStopped {
			shouldRecord = false
			break
		}
		stoppedAt := now
		tunnel.status.Status = statusStopped
		tunnel.status.StoppedAt = &stoppedAt
	case tunnelclient.EventFailed:
		if tunnel.status.Status == statusFailed && tunnel.status.LastError == event.Error {
			shouldRecord = false
			break
		}
		tunnel.status.Status = statusFailed
		tunnel.status.LastError = event.Error
	}
	tunnel.status.UpdatedAt = now
	m.mu.Unlock()

	if shouldRecord {
		m.recordEvent(tunnel, event)
	}
}

func (m *Manager) recordEvent(tunnel *tunnelRuntime, event tunnelclient.Event) {
	m.mu.RLock()
	status := tunnel.status
	m.mu.RUnlock()

	tunnelEvent := TunnelEvent{
		ID:         ulid.Make().String(),
		TunnelID:   tunnel.id,
		Type:       string(event.Type),
		Name:       status.Name,
		Connection: status.Type,
		Host:       status.Host,
		Port:       status.Port,
		Subdomain:  status.Subdomain,
		RemotePort: status.RemotePort,
		TunnelURL:  status.TunnelURL,
		Error:      event.Error,
		At:         event.At,
	}
	if tunnelEvent.At.IsZero() {
		tunnelEvent.At = time.Now().UTC()
	}

	m.mu.Lock()
	m.events = append(m.events, tunnelEvent)
	if len(m.events) > maxStoredEvents {
		m.events = m.events[len(m.events)-maxStoredEvents:]
	}
	m.mu.Unlock()

	m.logEvent(tunnelEvent)
	m.dispatchCallbacks(tunnel.callbackURLs, tunnelEvent)
}

func (m *Manager) logEvent(event TunnelEvent) {
	logger := m.appLogger()
	fields := tunnelEventLogFields(event)

	switch event.Type {
	case string(tunnelclient.EventFailed):
		logger.Error("App-server tunnel event", fields...)
	case string(tunnelclient.EventUnhealthy):
		logger.Warn("App-server tunnel event", fields...)
	default:
		logger.Info("App-server tunnel event", fields...)
	}
}

func (m *Manager) appLogger() *log.Logger {
	if m.logger != nil {
		return m.logger
	}
	return log.Default()
}

func tunnelEventLogFields(event TunnelEvent) []interface{} {
	fields := []interface{}{
		"event", event.Type,
		"tunnel_id", event.TunnelID,
		"connection_type", event.Connection,
		"host", event.Host,
		"port", event.Port,
	}
	if event.Name != "" {
		fields = append(fields, "name", event.Name)
	}
	if event.Subdomain != "" {
		fields = append(fields, "subdomain", event.Subdomain)
	}
	if event.RemotePort != 0 {
		fields = append(fields, "remote_port", event.RemotePort)
	}
	if event.TunnelURL != "" {
		fields = append(fields, "tunnel_url", event.TunnelURL)
	}
	if event.Error != "" {
		fields = append(fields, "error", event.Error)
	}
	return fields
}

func (m *Manager) dispatchCallbacks(callbackURLs []string, event TunnelEvent) {
	if len(callbackURLs) == 0 {
		return
	}

	body, err := json.Marshal(event)
	if err != nil {
		return
	}

	for _, callbackURL := range callbackURLs {
		callbackURL := callbackURL
		go func() {
			req, err := http.NewRequest(http.MethodPost, callbackURL, bytes.NewReader(body))
			if err != nil {
				return
			}
			req.Header.Set("Content-Type", "application/json")
			resp, err := m.httpClient.Do(req)
			if err != nil {
				m.appLogger().Error("Failed to dispatch app-server callback", "url", callbackURL, "error", err)
				return
			}
			_ = resp.Body.Close()
		}()
	}
}

func validateTunnelRequest(tunnel clientcfg.Tunnel, callbackURL string, callbackURLs []string) error {
	if tunnel.Type != constants.Stub && (tunnel.Port <= 0 || tunnel.Port > 65535) {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if tunnel.Type != constants.Http && tunnel.Type != constants.Tcp && tunnel.Type != constants.Stub {
		return fmt.Errorf("type must be http, tcp, or stub")
	}
	if err := tunnel.Validate(); err != nil {
		return err
	}
	for _, rawURL := range normalizeCallbackURLs(callbackURL, callbackURLs) {
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("callback_url must be a valid absolute URL")
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("callback_url must use http or https")
		}
	}
	return nil
}

func normalizeCallbackURLs(callbackURL string, callbackURLs []string) []string {
	seen := make(map[string]struct{})
	urls := make([]string, 0, len(callbackURLs)+1)
	for _, rawURL := range append([]string{callbackURL}, callbackURLs...) {
		rawURL = strings.TrimSpace(rawURL)
		if rawURL == "" {
			continue
		}
		if _, ok := seen[rawURL]; ok {
			continue
		}
		seen[rawURL] = struct{}{}
		urls = append(urls, rawURL)
	}
	return urls
}

func (m *Manager) desiredWorkers(cfg clientcfg.ClientConfig) int {
	if cfg.Tunnel.Type != constants.Http {
		return 1
	}
	if cfg.Tunnel.PoolSize <= 1 {
		return 1
	}
	if !supportsHTTPPooling(cfg.ServerUrl, cfg.UseLocalHost, m.httpClient) {
		return 1
	}
	return cfg.Tunnel.PoolSize
}

func supportsHTTPPooling(serverURL string, useLocalHost bool, client *http.Client) bool {
	var response struct {
		Version string `json:"version"`
	}

	resp, err := client.Get(serverBaseURL(serverURL, useLocalHost) + "/api/v1/version")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return false
	}

	return supportsHTTPPoolingVersion(response.Version)
}

func supportsHTTPPoolingVersion(rawVersion string) bool {
	version, err := semver.NewVersion(strings.TrimPrefix(rawVersion, "v"))
	if err != nil {
		return false
	}

	minVersion, err := semver.NewVersion("1.0.0")
	if err != nil {
		return false
	}

	return !version.LessThan(minVersion)
}

func serverBaseURL(serverURL string, useLocalHost bool) string {
	if strings.HasPrefix(serverURL, "http://") || strings.HasPrefix(serverURL, "https://") {
		return strings.TrimRight(serverURL, "/")
	}

	protocol := "http"
	if !useLocalHost {
		protocol = "https"
	}

	return protocol + "://" + strings.TrimRight(serverURL, "/")
}
