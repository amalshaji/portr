package cron

import (
	"context"
	"fmt"
	"net"
	"time"

	models "github.com/amalshaji/localport/internal/server/db/models"
	"github.com/go-resty/resty/v2"
)

var ErrInactiveTunnel = fmt.Errorf("inactive tunnel")

func (c *Cron) pingHttpConnection(connection models.Connection) error {
	client := resty.New().R()
	resp, err := client.Get(c.config.HttpTunnelUrl(connection.Subdomain.(string)))
	// don't care about the error, just care about the response
	if err != nil {
		return nil
	}
	if resp.StatusCode() == 404 && resp.Header().Get("X-LocalPort-Error") == "true" {
		return ErrInactiveTunnel
	}
	return nil
}

func (c *Cron) pingTcpConnection(connection models.Connection) error {
	timeout := time.Second * 5

	conn, err := net.DialTimeout("tcp", c.config.TcpTunnelUrl(connection.Port.(int64)), timeout)
	if err != nil {
		return ErrInactiveTunnel
	}
	defer conn.Close()
	return nil
}

func (c *Cron) pingActiveConnections(ctx context.Context) {
	var err error
	connections, err := c.db.Queries.GetAllActiveConnections(ctx)
	fmt.Printf("connections: %d\n", len(connections))
	if err != nil {
		c.logger.Error("error getting active connections", "error", err)
		return
	}

	for _, connection := range connections {
		go func(connection models.Connection) {
			if connection.Type == "http" {
				err = c.pingHttpConnection(connection)
			} else {
				err = c.pingTcpConnection(connection)
			}

			if err != nil {
				c.db.Queries.MarkConnectionAsClosed(ctx, connection.ID)
			}
		}(connection)
	}
}
