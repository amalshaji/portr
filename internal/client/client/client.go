package client

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/client/db"
	sshclient "github.com/amalshaji/portr/internal/client/ssh"
	"github.com/amalshaji/portr/internal/client/tui"
	"github.com/amalshaji/portr/internal/constants"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
)

type Client struct {
	config          *config.Config
	sshcs           []*sshclient.SshClient
	db              *db.Db
	tui             *tea.Program
	retentionCancel context.CancelFunc
	retentionDone   chan struct{}
	exitCh          chan error
	exitOnce        sync.Once
	tuiStart        chan struct{}
	tuiDone         chan struct{}
	tuiStopOnce     sync.Once
}

func NewClient(config *config.Config, db *db.Db) *Client {
	var p *tea.Program
	c := &Client{
		config: config,
		sshcs:  make([]*sshclient.SshClient, 0),
		db:     db,
		exitCh: make(chan error, 1),
	}

	if !config.DisableTUI {
		p = tui.New(config.Debug)
		c.tui = p
		c.tuiStart = make(chan struct{})
		c.tuiDone = make(chan struct{})
		go c.runTUI()
	}

	return c
}

func (c *Client) GetConfig() *config.Config {
	return c.config
}

func (c *Client) Done() <-chan error {
	return c.exitCh
}

func (c *Client) reportExit(err error) {
	c.exitOnce.Do(func() {
		c.exitCh <- err
	})
}

func (c *Client) stopTUI() {
	if c.tui == nil {
		return
	}

	c.tuiStopOnce.Do(func() {
		<-c.tuiStart
		c.tui.Kill()
		<-c.tuiDone
	})
}

func (c *Client) reportFatal(err error) {
	if err == nil {
		return
	}

	c.reportExit(err)
	c.stopTUI()
}

func (c *Client) runTUI() {
	close(c.tuiStart)
	defer close(c.tuiDone)
	defer func() {
		if r := recover(); r != nil {
			c.reportExit(fmt.Errorf("tui panic: %v", r))
		}
	}()

	_, err := c.tui.Run()
	switch {
	case err == nil:
		c.reportExit(nil)
	case errors.Is(err, tea.ErrProgramKilled):
	default:
		c.reportExit(fmt.Errorf("failed to run TUI: %w", err))
	}
}

func (c *Client) runFatalWorker(name string, fn func() error) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				c.reportFatal(fmt.Errorf("%s panic: %v", name, r))
			}
		}()

		if err := fn(); err != nil {
			c.reportFatal(err)
		}
	}()
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
			EnableHttpReverseProxy:          c.config.EnableHttpReverseProxy,
			InsecureSkipHostKeyVerification: *c.config.InsecureSkipHostKeyVerification,
		})
	}

	if len(clientConfigs) == 0 {
		return fmt.Errorf("please enter a valid service name")
	}

	poolingSupported := true
	for _, clientConfig := range clientConfigs {
		if desiredWorkers(clientConfig, true) > 1 {
			poolingSupported = supportsHTTPPooling(c.config.ServerUrl, c.config.UseLocalHost)
			break
		}
	}

	for _, clientConfig := range clientConfigs {
		tunnelName := clientConfig.Tunnel.Name
		if tunnelName == "" {
			tunnelName = fmt.Sprintf("%d", clientConfig.Tunnel.Port)
		}

		if c.config.DisableTUI {
			fmt.Printf("🚀 Starting tunnel: %s (%s:%d)\n", tunnelName, clientConfig.Tunnel.Host, clientConfig.Tunnel.Port)
		}

		workers := desiredWorkers(clientConfig, poolingSupported)

		if clientConfig.Tunnel.Type == constants.Http && workers > 1 && clientConfig.ConnectionID == "" {
			connID, err := sshclient.CreateNewConnection(clientConfig)
			if err != nil {
				return fmt.Errorf("failed to create shared connection for pool: %w", err)
			}
			clientConfig.ConnectionID = connID
		}

		for i := 0; i < workers; i++ {
			cfg := clientConfig
			sshc := sshclient.New(cfg, c.db, c.tui, c.reportFatal)
			c.Add(sshc)
			c.runFatalWorker("tunnel worker", func() error {
				return sshc.Start(ctx)
			})
		}
	}

	c.startConnectionLogRetention(ctx)

	return nil
}

func (c *Client) Add(sshc *sshclient.SshClient) {
	c.sshcs = append(c.sshcs, sshc)
}

func (c *Client) Shutdown(ctx context.Context) {
	c.stopTUI()

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
		defer func() {
			if r := recover(); r != nil {
				c.reportFatal(fmt.Errorf("connection log retention panic: %v", r))
			}
		}()

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
