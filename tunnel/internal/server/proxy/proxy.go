package proxy

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/amalshaji/portr/internal/server/config"
	"github.com/amalshaji/portr/internal/utils"
)

type Proxy struct {
	log    *slog.Logger
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
		log:    utils.GetLogger(),
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
		return "", fmt.Errorf("route not found")
	}
	return route, nil
}

func (p *Proxy) AddRoute(src, dst string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	_, ok := p.routes[src]
	if ok {
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

func (p *Proxy) ErrHandle(res http.ResponseWriter, req *http.Request, err error) {
	p.log.Error("proxy error", "error", err)
	p.RemoveRoute(p.config.ExtractSubdomain(req.Host))
	unregisteredSubdomainError(res, p.config.ExtractSubdomain(req.Host))
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
	p.log.Info("starting proxy server", "port", p.GetServerAddr())

	http.HandleFunc("/", p.handleRequest)

	if err := p.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("failed to start proxy server: %v", err)
	}
}

func (p *Proxy) Shutdown(_ context.Context) {
	p.log.Info("stopping proxy server")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer func() { cancel() }()

	if err := p.server.Shutdown(ctx); err != nil {
		p.log.Error("failed to stop proxy server", "error", err)
	}
}
