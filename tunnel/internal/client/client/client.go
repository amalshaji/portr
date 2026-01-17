package client

import (
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/client/db"
	sshclient "github.com/amalshaji/portr/internal/client/ssh"
	"github.com/amalshaji/portr/internal/client/tui"
	"github.com/amalshaji/portr/internal/constants"
	tea "github.com/charmbracelet/bubbletea"
)

type Client struct {
	config *config.Config
	sshcs  []*sshclient.SshClient
	db     *db.Db
	tui    *tea.Program
}

func NewClient(config *config.Config, db *db.Db) *Client {
	var p *tea.Program

	if !config.DisableTUI {
		p = tui.New(config.Debug)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					p.Kill()
					fmt.Printf("Recovered from panic: %v\n", r)
					os.Exit(1)
				}
			}()

			if _, err := p.Run(); err != nil {
				p.Kill()
				fmt.Printf("Failed to run TUI: %v\n", err)
				os.Exit(1)
			}

			p.Kill()
			os.Exit(0)
		}()
	}

	return &Client{
		config: config,
		sshcs:  make([]*sshclient.SshClient, 0),
		db:     db,
		tui:    p,
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
			ServerUrl:                       c.config.ServerUrl,
			SshUrl:                          c.config.SshUrl,
			TunnelUrl:                       c.config.TunnelUrl,
			SecretKey:                       c.config.SecretKey,
			Tunnel:                          tunnel,
			UseLocalHost:                    c.config.UseLocalHost,
			Debug:                           c.config.Debug,
			EnableRequestLogging:            c.config.EnableRequestLogging,
			HealthCheckInterval:             c.config.HealthCheckInterval,
			HealthCheckMaxRetries:           c.config.HealthCheckMaxRetries,
			DisableTUI:                      c.config.DisableTUI,
			InsecureSkipHostKeyVerification: c.config.InsecureSkipHostKeyVerification,
		})
	}

	if len(clientConfigs) == 0 {
		return fmt.Errorf("please enter a valid service name")
	}

	for _, clientConfig := range clientConfigs {
		tunnelName := clientConfig.Tunnel.Name
		if tunnelName == "" {
			tunnelName = fmt.Sprintf("%d", clientConfig.Tunnel.Port)
		}

		if c.config.DisableTUI {
			fmt.Printf("🚀 Starting tunnel: %s (%s:%d)\n", tunnelName, clientConfig.Tunnel.Host, clientConfig.Tunnel.Port)
		}

		// If HTTP, start a pool of SSH clients; for TCP keep it single
		workers := 1
		if clientConfig.Tunnel.Type == constants.Http && clientConfig.Tunnel.PoolSize > 1 {
			workers = clientConfig.Tunnel.PoolSize
		}

		// For HTTP pools, pre-create a single reserved connection ID and share across all workers
		if clientConfig.Tunnel.Type == constants.Http && workers > 1 && clientConfig.ConnectionID == "" {
			connID, err := sshclient.CreateNewConnection(clientConfig)
			if err != nil {
				return fmt.Errorf("failed to create shared connection for pool: %w", err)
			}
			clientConfig.ConnectionID = connID
		}

		for i := 0; i < workers; i++ {
			cfg := clientConfig
			sshc := sshclient.New(cfg, c.db, c.tui)
			c.Add(sshc)
			go sshc.Start(ctx)
		}
	}

	return nil
}

func (c *Client) Add(sshc *sshclient.SshClient) {
	c.sshcs = append(c.sshcs, sshc)
}

func (c *Client) Shutdown(ctx context.Context) {
	if c.config.DisableTUI {
		fmt.Printf("🛑 Shutting down tunnels...\n")
	}

	for _, sshc := range c.sshcs {
		sshc.Shutdown(ctx)
	}
}

func (c *Client) ReplaceTunnelsFromCli(tunnel config.Tunnel) {
	c.config.Tunnels = []config.Tunnel{tunnel}
}
