package proxy

import (
	"context"
	"errors"
	"fmt"

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
	routes map[string]string
	lock   sync.RWMutex
	server *http.Server
}

func (p *Proxy) GetServerAddr() string {
	return ":" + fmt.Sprint(p.config.Proxy.Port)
}

func New(config *config.Config) *Proxy {
	p := &Proxy{
		config: config,
		routes: make(map[string]string),
		server: nil,
	}
	p.server = &http.Server{Addr: p.GetServerAddr()}
	return p
}

func (p *Proxy) GetRoute(src string) (string, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	route, ok := p.routes[src]
	if !ok {
		log.Error("Route not found", "subdomain", src)
		return "", fmt.Errorf("route not found")
	}
	return route, nil
}

func (p *Proxy) AddRoute(src, dst string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	_, ok := p.routes[src]
	if ok {
		log.Error("Route already added", "subdomain", src)
		return fmt.Errorf("route already added")
	}
	p.routes[src] = dst
	return nil
}

func (p *Proxy) RemoveRoute(src string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	_, ok := p.routes[src]
	if !ok {
		log.Error("Route not found", "subdomain", src)
		return fmt.Errorf("route not found")
	}
	delete(p.routes, src)
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

func (p *Proxy) ErrHandle(res http.ResponseWriter, req *http.Request, err error) {
	log.Error("Error from proxy", "error", err)
	p.RemoveRoute(p.config.ExtractSubdomain(req.Host))
	connectionLostError(res)
}

func (p *Proxy) reverseProxy(target *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = p.ErrHandle
	return proxy
}

func (p *Proxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	subdomain := p.config.ExtractSubdomain(r.Host)
	target, err := p.GetRoute(subdomain)
	if err != nil {
		unregisteredSubdomainError(w, subdomain)
		return
	}

	proxy := p.reverseProxy(&url.URL{
		Scheme: "http",
		Host:   target,
	})

	proxy.ServeHTTP(w, r)
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
