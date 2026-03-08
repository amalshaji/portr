package client

import (
	"net/http"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/constants"
	"github.com/go-resty/resty/v2"
)

func supportsHTTPPooling(serverURL string, useLocalHost bool) bool {
	var response struct {
		Version string `json:"version"`
	}

	client := resty.New().SetTimeout(3 * time.Second)
	resp, err := client.R().SetResult(&response).Get(serverBaseURL(serverURL, useLocalHost) + "/api/v1/version")
	if err != nil || resp.StatusCode() != http.StatusOK {
		return false
	}

	return supportsHTTPPoolingVersion(response.Version)
}

func supportsHTTPPoolingVersion(rawVersion string) bool {
	version, err := semver.NewVersion(strings.TrimPrefix(rawVersion, "v"))
	if err != nil {
		return false
	}

	minVersion, err := semver.NewVersion("1.0.0")
	if err != nil {
		return false
	}

	return !version.LessThan(minVersion)
}

func desiredWorkers(cfg config.ClientConfig, poolingSupported bool) int {
	if cfg.Tunnel.Type != constants.Http {
		return 1
	}
	if cfg.Tunnel.PoolSize <= 1 {
		return 1
	}
	if !poolingSupported {
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
