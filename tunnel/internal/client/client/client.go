package client

import (
	"context"
	"fmt"
	"slices"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/client/db"
	"github.com/amalshaji/portr/internal/client/ssh"
)

type Client struct {
	config *config.Config
	sshcs  []*ssh.SshClient
	db     *db.Db
}

func NewClient(config *config.Config, db *db.Db) *Client {
	return &Client{
		config: config,
		sshcs:  make([]*ssh.SshClient, 0),
		db:     db,
	}
}

func (c *Client) GetConfig() *config.Config {
	return c.config
}

func (c *Client) Start(ctx context.Context, services ...string) error {
	var clientConfigs []config.ClientConfig

	for _, tunnel := range c.config.Tunnels {
		if len(services) > 0 && !slices.Contains(services, tunnel.Name) {
			continue
		}
		clientConfigs = append(clientConfigs, config.ClientConfig{
			ServerUrl:             c.config.ServerUrl,
			SshUrl:                c.config.SshUrl,
			TunnelUrl:             c.config.TunnelUrl,
			SecretKey:             c.config.SecretKey,
			Tunnel:                tunnel,
			UseLocalHost:          c.config.UseLocalHost,
			Debug:                 c.config.Debug,
			EnableRequestLogging:  c.config.EnableRequestLogging,
			HealthCheckInterval:   c.config.HealthCheckInterval,
			HealthCheckMaxRetries: c.config.HealthCheckMaxRetries,
		})
	}

	if len(clientConfigs) == 0 {
		return fmt.Errorf("please enter a valid service name")
	}

	for _, clientConfig := range clientConfigs {
		sshc := ssh.New(clientConfig, c.db)
		c.Add(sshc)
		go sshc.Start(ctx)
	}

	return nil
}

func (c *Client) Add(sshc *ssh.SshClient) {
	c.sshcs = append(c.sshcs, sshc)
}

func (c *Client) Shutdown(ctx context.Context) {
	for _, sshc := range c.sshcs {
		sshc.Shutdown(ctx)
	}
}

// Create tunnel from cli args and replaces it in config
func (c *Client) ReplaceTunnelsFromCli(tunnel config.Tunnel) {
	c.config.Tunnels = []config.Tunnel{tunnel}
}
