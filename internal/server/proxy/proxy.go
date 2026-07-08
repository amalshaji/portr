package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/amalshaji/portr/internal/server/config"
	"github.com/amalshaji/portr/internal/server/wstunnel"
	"github.com/amalshaji/portr/internal/utils"
	"github.com/charmbracelet/log"
)

type Proxy struct {
	config    *config.Config
	routes    map[string][]string // subdomain -> list of backends (host:port)
	rrIdx     map[string]int      // round-robin index per subdomain
	lock      sync.RWMutex
	server    *http.Server
	transport *http.Transport
	tunnel    *wstunnel.Manager
}

func (p *Proxy) GetServerAddr() string {
	return ":" + fmt.Sprint(p.config.Proxy.Port)
}

func New(config *config.Config) *Proxy {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          256,
		MaxIdleConnsPerHost:   64,
		IdleConnTimeout:       90 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
	p := &Proxy{
		config:    config,
		routes:    make(map[string][]string),
		rrIdx:     make(map[string]int),
		transport: transport,
	}
	p.server = &http.Server{
		Addr:              p.GetServerAddr(),
		Handler:           p,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       90 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	return p
}

func (p *Proxy) SetTunnelManager(manager *wstunnel.Manager) {
	p.tunnel = manager
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p.tunnel != nil && r.URL.Path == "/_portr/tunnel/connect" {
		p.tunnel.Handler().ServeHTTP(w, r)
		return
	}
	p.handleRequest(w, r)
}

// GetRoute is kept for backward compatibility and returns the first backend if available.
func (p *Proxy) GetRoute(src string) (string, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	backends, ok := p.routes[src]
	if !ok || len(backends) == 0 {
		log.Error("Route not found", "subdomain", src)
		return "", fmt.Errorf("route not found")
	}
	return backends[0], nil
}

// AddRoute is kept for backward compatibility and only allows adding a new subdomain once with a single backend.
func (p *Proxy) AddRoute(src, dst string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	_, ok := p.routes[src]
	if ok {
		log.Error("Route already added", "subdomain", src)
		return fmt.Errorf("route already added")
	}
	p.routes[src] = []string{dst}
	p.rrIdx[src] = 0
	return nil
}

// RemoveRoute removes all backends for the subdomain.
func (p *Proxy) RemoveRoute(src string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	_, ok := p.routes[src]
	if !ok {
		log.Error("Route not found", "subdomain", src)
		return fmt.Errorf("route not found")
	}
	delete(p.routes, src)
	delete(p.rrIdx, src)
	return nil
}

// AddBackend adds a backend to a subdomain, creating the subdomain entry if needed.
func (p *Proxy) AddBackend(src, dst string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	list, ok := p.routes[src]
	if !ok {
		p.routes[src] = []string{dst}
		p.rrIdx[src] = 0
		return nil
	}
	// Prevent duplicate backend entries
	if slices.Contains(list, dst) {
		return nil
	}
	p.routes[src] = append(list, dst)
	return nil
}

// RemoveBackend removes a single backend from a subdomain. If it is the last backend, the subdomain is removed.
func (p *Proxy) RemoveBackend(src, dst string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	list, ok := p.routes[src]
	if !ok {
		return fmt.Errorf("route not found")
	}
	idx := -1
	for i, b := range list {
		if b == dst {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("backend not found")
	}
	list = append(list[:idx], list[idx+1:]...)
	if len(list) == 0 {
		delete(p.routes, src)
		delete(p.rrIdx, src)
		return nil
	}
	p.routes[src] = list
	// Keep rrIdx within bounds
	if p.rrIdx[src] >= len(list) {
		p.rrIdx[src] = 0
	}
	return nil
}

func unregisteredSubdomainError(w http.ResponseWriter, subdomain string) {
	w.Header().Set("X-Portr-Error", "true")
	w.Header().Set("X-Portr-Error-Reason", "unregistered-subdomain")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(utils.UnregisteredSubdomain(subdomain)))
}

func connectionLostError(w http.ResponseWriter) {
	w.Header().Set("X-Portr-Error", "true")
	w.Header().Set("X-Portr-Error-Reason", "connection-lost")
	w.WriteHeader(http.StatusServiceUnavailable)
	w.Write([]byte(utils.ConnectionLost()))
}

func (p *Proxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	subdomain := p.config.ExtractSubdomain(r.Host)

	if r.Header.Get("X-Portr-Ping-Request") == "true" {
		if p.hasHTTPBackend(subdomain) {
			w.WriteHeader(http.StatusOK)
			return
		}
		unregisteredSubdomainError(w, subdomain)
		return
	}

	if p.tunnel != nil && p.tunnel.HasHTTPBackend(subdomain) {
		conn, initial, err := wstunnel.HijackRequest(w, r)
		if err != nil {
			log.Error("Failed to hijack proxied request", "error", err, "subdomain", subdomain)
			http.Error(w, "failed to proxy request", http.StatusBadGateway)
			return
		}
		p.tunnel.OpenHTTPStream(subdomain, conn, initial)
		return
	}

	backends, err := p.nextBackends(subdomain, 3)
	if err != nil {
		unregisteredSubdomainError(w, subdomain)
		return
	}
	if isUpgradeRequest(r) || !isReplaySafe(r) {
		backends = backends[:1]
	}

	proxy := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: backends[0]})
	proxy.Transport = &backendTransport{
		base:      p.transport,
		backends:  backends,
		subdomain: subdomain,
	}
	proxy.ErrorHandler = func(res http.ResponseWriter, _ *http.Request, err error) {
		if !errors.Is(err, io.EOF) {
			log.Error("Error from proxy", "error", err, "subdomain", subdomain)
		}
		connectionLostError(res)
	}
	proxy.ServeHTTP(w, r)
}

func (p *Proxy) hasHTTPBackend(subdomain string) bool {
	if p.tunnel != nil && p.tunnel.HasHTTPBackend(subdomain) {
		return true
	}
	p.lock.RLock()
	defer p.lock.RUnlock()
	return len(p.routes[subdomain]) > 0
}

func (p *Proxy) nextBackends(src string, limit int) ([]string, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	list := p.routes[src]
	if len(list) == 0 {
		return nil, fmt.Errorf("route not found")
	}
	if limit > len(list) {
		limit = len(list)
	}

	start := p.rrIdx[src] % len(list)
	result := make([]string, 0, limit)
	for offset := 0; offset < limit; offset++ {
		result = append(result, list[(start+offset)%len(list)])
	}
	p.rrIdx[src] = (start + 1) % len(list)
	return result, nil
}

func isReplaySafe(r *http.Request) bool {
	if r.Body != nil && r.Body != http.NoBody {
		return false
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

type backendTransport struct {
	base      http.RoundTripper
	backends  []string
	subdomain string
}

func (t *backendTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	var lastErr error
	for attempt, backend := range t.backends {
		outbound := request.Clone(request.Context())
		outboundURL := *request.URL
		outboundURL.Scheme = "http"
		outboundURL.Host = backend
		outbound.URL = &outboundURL

		response, err := t.base.RoundTrip(outbound)
		if err == nil {
			return response, nil
		}
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		lastErr = err
		log.Warn("Tunnel backend transport failed", "error", err, "subdomain", t.subdomain, "backend", backend, "attempt", attempt+1)
		if request.Context().Err() != nil {
			break
		}
	}
	return nil, lastErr
}

func isUpgradeRequest(request *http.Request) bool {
	if request.Header.Get("Upgrade") == "" {
		return false
	}
	for _, token := range strings.Split(request.Header.Get("Connection"), ",") {
		if strings.EqualFold(strings.TrimSpace(token), "upgrade") {
			return true
		}
	}
	return false
}

func (p *Proxy) Start() {
	log.Info("Starting proxy server", "port", p.GetServerAddr())

	if err := p.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("Failed to start proxy server", "error", err)
	}
}

func (p *Proxy) Shutdown(_ context.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := p.server.Shutdown(ctx); err != nil {
		log.Error("Failed to stop proxy server", "error", err)
		return
	}
	p.transport.CloseIdleConnections()

	log.Info("Stopped proxy server")
}
