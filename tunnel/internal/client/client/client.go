package client

import (
	"context"
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/client/db"
	"github.com/amalshaji/portr/internal/client/ssh"
	"github.com/amalshaji/portr/internal/client/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
)

type Client struct {
	config          *config.Config
	sshcs           []*ssh.SshClient
	db              *db.Db
	tui             *tea.Program
	retentionCancel context.CancelFunc
	retentionDone   chan struct{}
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
		sshcs:  make([]*ssh.SshClient, 0),
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
			DisableTUI:            c.config.DisableTUI,
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

		sshc := ssh.New(clientConfig, c.db, c.tui)
		c.Add(sshc)
		go sshc.Start(ctx)
	}

	c.startConnectionLogRetention(ctx)

	return nil
}

func (c *Client) Add(sshc *ssh.SshClient) {
	c.sshcs = append(c.sshcs, sshc)
}

func (c *Client) Shutdown(ctx context.Context) {
	if c.retentionCancel != nil {
		c.retentionCancel()
		if c.retentionDone != nil {
			<-c.retentionDone
		}
		c.retentionCancel = nil
		c.retentionDone = nil
	}

	if c.config.DisableTUI {
		fmt.Printf("🛑 Shutting down tunnels...\n")
	}

	for _, sshc := range c.sshcs {
		sshc.Shutdown(ctx)
	}
}

// Create tunnel from cli args and replaces it in config
func (c *Client) ReplaceTunnelsFromCli(tunnel config.Tunnel) {
	c.config.Tunnels = []config.Tunnel{tunnel}
}

func (c *Client) startConnectionLogRetention(ctx context.Context) {
	if c.config.ConnectionLogRetentionDays <= 0 || c.retentionCancel != nil {
		return
	}

	retentionCtx, cancel := context.WithCancel(context.Background())
	c.retentionCancel = cancel
	c.retentionDone = make(chan struct{})

	go func() {
		defer close(c.retentionDone)

		c.pruneConnectionLogs()

		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-retentionCtx.Done():
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.pruneConnectionLogs()
			}
		}
	}()
}

func (c *Client) pruneConnectionLogs() {
	if c.config.ConnectionLogRetentionDays <= 0 {
		return
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -c.config.ConnectionLogRetentionDays)
	deletedCount, err := c.db.DeleteRequestsOlderThan(cutoff)
	if err != nil {
		if c.config.Debug {
			log.Error("Failed to prune connection logs", "error", err)
		}
		return
	}

	if c.config.Debug && deletedCount > 0 {
		log.Debug(
			"Pruned connection logs",
			"deleted", deletedCount,
			"retention_days", c.config.ConnectionLogRetentionDays,
		)
	}
}
