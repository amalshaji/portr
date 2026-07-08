package cron

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/amalshaji/portr/internal/server/db"
	"github.com/charmbracelet/log"
	"github.com/go-resty/resty/v2"
)

var ErrInactiveTunnel = errors.New("inactive tunnel")

const maxConcurrentPings = 16

func (c *Cron) pingHttpConnection(ctx context.Context, connection db.Connection) error {
	if connection.Subdomain == nil {
		return ErrInactiveTunnel
	}
	client := c.httpClient
	if client == nil {
		client = resty.New().SetTimeout(5 * time.Second)
	}
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("X-Portr-Ping-Request", "true").
		Get(c.config.HttpTunnelUrl(*connection.Subdomain))
	if err != nil {
		return err
	}
	if resp.StatusCode() == 404 &&
		resp.Header().Get("X-Portr-Error") == "true" &&
		resp.Header().Get("X-Portr-Error-Reason") == "unregistered-subdomain" {
		return ErrInactiveTunnel
	}
	return nil
}

func (c *Cron) pingTcpConnection(ctx context.Context, connection db.Connection) error {
	if connection.Port == nil {
		return ErrInactiveTunnel
	}
	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(dialCtx, "tcp", c.config.TcpTunnelUrl(*connection.Port))
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	return nil
}

func (c *Cron) pingActiveConnections(ctx context.Context) {
	c.pingActiveConnectionsWithProbe(ctx, c.probeConnection)
}

func (c *Cron) pingActiveConnectionsWithProbe(
	ctx context.Context,
	probe func(context.Context, db.Connection) error,
) {
	connections, err := c.service.GetAllActiveConnections(ctx)
	if err != nil {
		log.Error("Error getting active connections", "error", err)
		return
	}

	log.Info("Pinging active connections", "count", len(connections))

	semaphore := make(chan struct{}, maxConcurrentPings)
	var wg sync.WaitGroup
connectionLoop:
	for _, connection := range connections {
		if ctx.Err() != nil {
			break
		}
		select {
		case semaphore <- struct{}{}:
		case <-ctx.Done():
			break connectionLoop
		}
		wg.Add(1)
		go func(connection db.Connection) {
			defer wg.Done()
			defer func() { <-semaphore }()

			pingErr := probe(ctx, connection)

			switch {
			case pingErr == nil:
			case errors.Is(pingErr, ErrInactiveTunnel):
				if err := c.service.MarkConnectionAsClosed(ctx, connection.ID); err != nil {
					log.Error("Failed to close inactive connection", "connection_id", connection.ID, "error", err)
				}
			default:
				log.Warn("Connection reconciliation probe failed", "connection_id", connection.ID, "error", pingErr)
			}
		}(connection)
	}
	wg.Wait()
}

func (c *Cron) probeConnection(ctx context.Context, connection db.Connection) error {
	switch connection.Type {
	case "http":
		return c.pingHttpConnection(ctx, connection)
	case "tcp":
		return c.pingTcpConnection(ctx, connection)
	default:
		return fmt.Errorf("unsupported connection type %q", connection.Type)
	}
}
