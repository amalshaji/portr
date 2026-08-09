package localserver

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Server is a loopback HTTP server that dispatches to a handler per subdomain.
//
// Tunnels served from inside the client process (stub responses, static
// directories) all share this shape: one ephemeral local port that the tunnel
// worker forwards to, with the Host header as the only discriminator between
// the tunnels registered on it.
//
// A registered handler that implements io.Closer is closed when it is
// unregistered or when the server shuts down.
type Server struct {
	name     string
	server   *http.Server
	listener net.Listener
	mu       sync.RWMutex
	handlers map[string]http.Handler
}

// New returns a server that is not yet listening. name is used in start errors.
func New(name string) *Server {
	s := &Server{
		name:     name,
		handlers: make(map[string]http.Handler),
	}
	s.server = &http.Server{
		Handler:           s,
		ReadHeaderTimeout: 15 * time.Second,
	}
	return s
}

func (s *Server) Start() error {
	if s.listener != nil {
		return nil
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("start %s: %w", s.name, err)
	}
	s.listener = listener

	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			// The tunnel worker health checks surface unreachable local endpoints;
			// there is no caller to return this asynchronous server error to.
		}
	}()

	return nil
}

func (s *Server) Addr() string {
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

func (s *Server) Port() int {
	if s.listener == nil {
		return 0
	}
	if addr, ok := s.listener.Addr().(*net.TCPAddr); ok {
		return addr.Port
	}
	return 0
}

func (s *Server) Register(subdomain string, handler http.Handler) error {
	subdomain = strings.ToLower(strings.TrimSpace(subdomain))
	if subdomain == "" {
		return fmt.Errorf("subdomain is required")
	}
	if handler == nil {
		return fmt.Errorf("handler is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.handlers[subdomain]; exists {
		return fmt.Errorf("%s route for subdomain %q already exists", s.name, subdomain)
	}
	s.handlers[subdomain] = handler
	return nil
}

func (s *Server) Unregister(subdomain string) {
	subdomain = strings.ToLower(strings.TrimSpace(subdomain))
	if subdomain == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if handler, ok := s.handlers[subdomain]; ok {
		closeHandler(handler)
		delete(s.handlers, subdomain)
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	for subdomain, handler := range s.handlers {
		closeHandler(handler)
		delete(s.handlers, subdomain)
	}
	s.mu.Unlock()

	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Header.Get("X-Portr-Ping-Request") == "true" {
		w.WriteHeader(http.StatusOK)
		return
	}

	handler, ok := s.handlerForHost(req.Host)
	if !ok {
		http.NotFound(w, req)
		return
	}
	handler.ServeHTTP(w, req)
}

// handlerForHost resolves the handler for a request. Every tunnel on this
// server shares one local port, so the Host header is the only discriminator.
func (s *Server) handlerForHost(host string) (http.Handler, bool) {
	hostname := strings.ToLower(strings.TrimSpace(stripPort(host)))

	s.mu.RLock()
	defer s.mu.RUnlock()

	if handler, ok := s.handlers[hostname]; ok {
		return handler, true
	}
	for subdomain, handler := range s.handlers {
		if strings.HasPrefix(hostname, subdomain+".") {
			return handler, true
		}
	}
	// A direct request to the loopback address has no subdomain to match.
	if len(s.handlers) == 1 {
		for _, handler := range s.handlers {
			return handler, true
		}
	}
	return nil, false
}

func closeHandler(handler http.Handler) {
	if closer, ok := handler.(io.Closer); ok {
		_ = closer.Close()
	}
}

func stripPort(host string) string {
	if hostname, _, err := net.SplitHostPort(host); err == nil {
		return hostname
	}
	if strings.Count(host, ":") == 1 {
		if idx := strings.LastIndex(host, ":"); idx > -1 {
			return host[:idx]
		}
	}
	return host
}
