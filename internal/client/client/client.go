package client

import (
	"context"
	"log"
	"slices"

	"github.com/amalshaji/localport/internal/client/config"
	"github.com/amalshaji/localport/internal/client/ssh"
)

type Client struct {
	config *config.Config
	sshcs  []*ssh.SshClient
}

func NewClient(configFile string) *Client {
	config, err := config.Load(configFile)

	if err != nil {
		log.Fatal("failed to load config file")
	}

	return &Client{
		config: &config,
		sshcs:  make([]*ssh.SshClient, 0),
	}
}

func (c *Client) Start(ctx context.Context, services ...string) {
	c.config.SetDefaults()

	var clientConfigs []config.ClientConfig

	for _, tunnel := range c.config.Tunnels {
		if len(services) > 0 && !slices.Contains(services, tunnel.Name) {
			continue
		}
		clientConfigs = append(clientConfigs, config.ClientConfig{
			ServerUrl: c.config.ServerUrl,
			SshUrl:    c.config.SshUrl,
			TunnelUrl: c.config.TunnelUrl,
			Secretkey: c.config.Secretkey,
			Tunnel:    tunnel,
			Secure:    c.config.Secure,
			Debug:     c.config.Debug,
		})
	}

	for _, clientConfig := range clientConfigs {
		sshc := ssh.New(clientConfig)
		c.Add(sshc)
		go sshc.Start(ctx)
	}
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
