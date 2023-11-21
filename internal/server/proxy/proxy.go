package proxy

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/amalshaji/localport/internal/server/config"
	"github.com/amalshaji/localport/internal/utils"
)

type Proxy struct {
	log    *slog.Logger
	config *config.Config
	routes map[string]string
	lock   sync.RWMutex
	server *http.Server
}

func New(config *config.Config) *Proxy {
	return &Proxy{
		log:    utils.GetLogger(),
		config: config,
		routes: make(map[string]string),
		server: &http.Server{Addr: ":" + fmt.Sprint(config.Proxy.Port)},
	}
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

func (p *Proxy) reverseProxy(target *url.URL) *httputil.ReverseProxy {
	return httputil.NewSingleHostReverseProxy(target)
}

func (p *Proxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	subdomain := p.config.ExtractSubdomain(r.Host)
	target, err := p.GetRoute(subdomain)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	proxy := p.reverseProxy(&url.URL{
		Scheme: "http",
		Host:   target,
	})

	proxy.ServeHTTP(w, r)
}

func (p *Proxy) Start() {
	p.log.Info("starting proxy server", "host", p.config.Proxy.Host, "port", p.config.Proxy.Port)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	http.HandleFunc("/", p.handleRequest)
	go func() {
		if err := p.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			p.log.Error("failed to start proxy server", "error", err)
			done <- nil
		}
	}()

	<-done
	p.log.Info("stopping proxy server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := p.server.Shutdown(ctx); err != nil {
		p.log.Error("failed to stop proxy server", "error", err)
	}
}
