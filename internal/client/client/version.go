package client

import (
	"strings"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/constants"
)

func desiredWorkers(cfg config.ClientConfig) int {
	if cfg.Tunnel.Type != constants.Http {
		return 1
	}
	if cfg.Tunnel.PoolSize <= 1 {
		return 1
	}
	return cfg.Tunnel.PoolSize
}

func serverBaseURL(serverURL string, useLocalHost bool) string {
	if strings.HasPrefix(serverURL, "http://") || strings.HasPrefix(serverURL, "https://") {
		return strings.TrimRight(serverURL, "/")
	}

	protocol := "http"
	if !useLocalHost {
		protocol = "https"
	}

	return protocol + "://" + strings.TrimRight(serverURL, "/")
}
