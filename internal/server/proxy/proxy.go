package proxy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/amalshaji/portr/internal/server/config"
	"github.com/amalshaji/portr/internal/utils"
	"github.com/charmbracelet/log"
)

type Proxy struct {
	config *config.Config
	routes map[string][]string // subdomain -> list of backends (host:port)
	rrIdx  map[string]int      // round-robin index per subdomain
	lock   sync.RWMutex
	server *http.Server
}

func (p *Proxy) GetServerAddr() string {
	return ":" + fmt.Sprint(p.config.Proxy.Port)
}

func New(config *config.Config) *Proxy {
	p := &Proxy{
		config: config,
		routes: make(map[string][]string),
		rrIdx:  make(map[string]int),
		server: nil,
	}
	p.server = &http.Server{Addr: p.GetServerAddr()}
	return p
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

// GetNextBackend returns the next backend target for the subdomain using round-robin.
func (p *Proxy) GetNextBackend(src string) (string, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	list, ok := p.routes[src]
	if !ok || len(list) == 0 {
		log.Error("Route not found", "subdomain", src)
		return "", fmt.Errorf("route not found")
	}
	i := p.rrIdx[src]
	target := list[i]
	p.rrIdx[src] = (i + 1) % len(list)
	return target, nil
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

func (p *Proxy) reverseProxy(target *url.URL, subdomain, backend string) *httputil.ReverseProxy {
	rp := httputil.NewSingleHostReverseProxy(target)
	// Capture backend so we can remove only the failing one
	rp.ErrorHandler = func(res http.ResponseWriter, req *http.Request, err error) {
		log.Error("Error from proxy", "error", err, "subdomain", subdomain, "backend", backend)
		_ = p.RemoveBackend(subdomain, backend)
		connectionLostError(res)
	}
	return rp
}

func (p *Proxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	subdomain := p.config.ExtractSubdomain(r.Host)
	// For upgraded connections (e.g., WebSocket), don't retry — stream directly
	if isUpgradeRequest(r) {
		target, err := p.GetNextBackend(subdomain)
		if err != nil {
			unregisteredSubdomainError(w, subdomain)
			return
		}
		proxy := p.reverseProxy(&url.URL{Scheme: "http", Host: target}, subdomain, target)
		proxy.ServeHTTP(w, r)
		return
	}

	// Buffer the request body so we can retry on transport errors
	var bodyBytes []byte
	if r.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}
		// Reset original body for safety
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	// Determine max attempts (bounded by number of backends and a small constant)
	maxAttempts := p.backendCount(subdomain)
	if maxAttempts == 0 {
		unregisteredSubdomainError(w, subdomain)
		return
	}
	if maxAttempts > 3 {
		maxAttempts = 3
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		target, err := p.GetNextBackend(subdomain)
		if err != nil {
			unregisteredSubdomainError(w, subdomain)
			return
		}

		// Build reverse proxy for this target with an error handler that signals retry
		failed := false
		rp := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: target})
		rp.ErrorHandler = func(res http.ResponseWriter, req *http.Request, err error) {
			log.Error("Error from proxy (will retry)", "error", err, "subdomain", subdomain, "backend", target)
			_ = p.RemoveBackend(subdomain, target)
			failed = true
			// Do not write to client here; allow retry loop to proceed
		}

		// Buffer writes so we only send to client on successful attempt
		bw := newBufferResponseWriter()

		// Clone request and reset body for this attempt
		req2 := r.Clone(r.Context())
		if bodyBytes != nil {
			req2.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		rp.ServeHTTP(bw, req2)

		if !failed {
			bw.flushTo(w)
			return
		}
		// else try next backend
	}

	// All attempts failed
	connectionLostError(w)
}

// bufferResponseWriter buffers response status, headers, and body until flushed.
type bufferResponseWriter struct {
	header      http.Header
	body        bytes.Buffer
	statusCode  int
	wroteHeader bool
}

func newBufferResponseWriter() *bufferResponseWriter {
	return &bufferResponseWriter{header: make(http.Header), statusCode: http.StatusOK}
}

func (b *bufferResponseWriter) Header() http.Header { return b.header }

func (b *bufferResponseWriter) WriteHeader(status int) {
	if b.wroteHeader {
		return
	}
	b.wroteHeader = true
	b.statusCode = status
}

func (b *bufferResponseWriter) Write(p []byte) (int, error) {
	if !b.wroteHeader {
		b.WriteHeader(http.StatusOK)
	}
	return b.body.Write(p)
}

func (b *bufferResponseWriter) flushTo(w http.ResponseWriter) {
	// Copy headers
	dst := w.Header()
	maps.Copy(dst, b.header)
	w.WriteHeader(b.statusCode)
	_, _ = w.Write(b.body.Bytes())
}

func isUpgradeRequest(r *http.Request) bool {
	if r.Header.Get("Connection") == "Upgrade" || r.Header.Get("Upgrade") != "" {
		return true
	}
	return false
}

// backendCount returns the number of backends for a subdomain.
func (p *Proxy) backendCount(src string) int {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return len(p.routes[src])
}

func (p *Proxy) Start() {
	log.Info("Starting proxy server", "port", p.GetServerAddr())

	http.HandleFunc("/", p.handleRequest)

	if err := p.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("Failed to start proxy server", "error", err)
	}
}

func (p *Proxy) Shutdown(_ context.Context) {

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer func() { cancel() }()

	if err := p.server.Shutdown(ctx); err != nil {
		log.Error("Failed to stop proxy server", "error", err)
		return
	}

	log.Info("Stopped proxy server")
}
