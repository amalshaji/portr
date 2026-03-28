package cron

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/amalshaji/portr/internal/server/db"
	"github.com/charmbracelet/log"
	"github.com/go-resty/resty/v2"
)

var ErrInactiveTunnel = fmt.Errorf("inactive tunnel")

func (c *Cron) pingHttpConnection(connection db.Connection) error {
	client := resty.New().R()
	resp, err := client.SetHeader("X-Portr-Ping-Request", "true").Get(c.config.HttpTunnelUrl(*connection.Subdomain))
	// don't care about the error, just care about the response
	if err != nil {
		return nil
	}
	if resp.StatusCode() == 404 && resp.Header().Get("X-Portr-Error") == "true" {
		return ErrInactiveTunnel
	}
	return nil
}

func (c *Cron) pingTcpConnection(connection db.Connection) error {
	timeout := time.Second * 5

	conn, err := net.DialTimeout("tcp", c.config.TcpTunnelUrl(*connection.Port), timeout)
	if err != nil {
		return ErrInactiveTunnel
	}
	defer conn.Close()
	return nil
}

func (c *Cron) pingActiveConnections(ctx context.Context) {
	var err error
	connections := c.service.GetAllActiveConnections(ctx)
	if err != nil {
		log.Error("Error getting active connections", "error", err)
		return
	}

	log.Info("Pinging active connections", "count", len(connections))

	for _, connection := range connections {
		go func(connection db.Connection) {
			if connection.Type == "http" {
				err = c.pingHttpConnection(connection)
			} else {
				err = c.pingTcpConnection(connection)
			}

			if err != nil {
				c.service.MarkConnectionAsClosed(ctx, connection.ID)
			}
		}(connection)
	}
}
