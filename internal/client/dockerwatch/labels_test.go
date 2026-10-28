package dockerwatch

import (
	"strings"
	"testing"

	"github.com/amalshaji/portr/internal/constants"
)

func labelledContainer(labels map[string]string, exposed ...string) Container {
	var container Container
	container.Name = "/shop-web-1"
	container.Config.Labels = map[string]string{
		ComposeProjectLabel: "shop",
		ComposeServiceLabel: "web",
		LabelEnable:         "true",
	}
	for key, value := range labels {
		container.Config.Labels[key] = value
	}
	container.Config.ExposedPorts = map[string]struct{}{}
	for _, port := range exposed {
		container.Config.ExposedPorts[port] = struct{}{}
	}
	return container
}

func TestTunnelRequestDefaults(t *testing.T) {
	request, err := tunnelRequest(labelledContainer(nil, "3000/tcp"), "172.18.0.4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if request.Type != constants.Http {
		t.Fatalf("expected http, got %q", request.Type)
	}
	if request.Port != 3000 || request.Host != "172.18.0.4" {
		t.Fatalf("unexpected target %s:%d", request.Host, request.Port)
	}
	if request.Subdomain != "web-shop" {
		t.Fatalf("expected default subdomain web-shop, got %q", request.Subdomain)
	}
	if request.Name != "web" {
		t.Fatalf("expected name web, got %q", request.Name)
	}
}

func TestTunnelRequestReadsLabels(t *testing.T) {
	request, err := tunnelRequest(labelledContainer(map[string]string{
		LabelPort:       "8080",
		LabelSubdomain:  "my-app",
		LabelName:       "frontend",
		LabelBasicAuth:  "admin:secret",
		LabelHostHeader: "rewrite",
	}, "3000/tcp", "8080/tcp"), "10.0.0.2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if request.Port != 8080 || request.Subdomain != "my-app" || request.Name != "frontend" {
		t.Fatalf("labels not applied: %+v", request)
	}
	if request.BasicAuth != "admin:secret" || request.HostHeader != "rewrite" {
		t.Fatalf("labels not applied: %+v", request)
	}
}

func TestTunnelRequestTCP(t *testing.T) {
	request, err := tunnelRequest(labelledContainer(map[string]string{LabelType: "TCP"}, "5432/tcp"), "10.0.0.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if request.Type != constants.Tcp || request.Port != 5432 || request.Subdomain != "" {
		t.Fatalf("unexpected tcp request: %+v", request)
	}
}

func TestTunnelRequestErrors(t *testing.T) {
	tests := []struct {
		name    string
		labels  map[string]string
		exposed []string
		want    string
	}{
		{"no exposed port", nil, nil, "exposes no tcp port"},
		{"several exposed ports", nil, []string{"3000/tcp", "9229/tcp"}, "several ports [3000 9229]"},
		{"udp only", nil, []string{"53/udp"}, "exposes no tcp port"},
		{"bad port", map[string]string{LabelPort: "http"}, nil, "portr.port must be a port"},
		{"port out of range", map[string]string{LabelPort: "70000"}, nil, "portr.port must be a port"},
		{"bad type", map[string]string{LabelType: "udp"}, []string{"53/tcp"}, "portr.type must be http or tcp"},
		{"tcp subdomain", map[string]string{LabelType: "tcp", LabelSubdomain: "db"}, []string{"5432/tcp"}, "only supported for http"},
		{"typo", map[string]string{"portr.subdomian": "x"}, []string{"3000/tcp"}, "unknown label portr.subdomian"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tunnelRequest(labelledContainer(tt.labels, tt.exposed...), "10.0.0.2")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestDefaultSubdomain(t *testing.T) {
	tests := []struct {
		service, project, want string
	}{
		{"web", "shop", "web-shop"},
		{"api_v2", "My.Project", "api-v2-my-project"},
		{"-web-", "", "web"},
		{strings.Repeat("a", 60), "project", strings.Repeat("a", 60) + "-pr"},
		{strings.Repeat("a", 62), "project", strings.Repeat("a", 62)},
	}

	for _, tt := range tests {
		if got := defaultSubdomain(tt.service, tt.project); got != tt.want {
			t.Errorf("defaultSubdomain(%q, %q) = %q, want %q", tt.service, tt.project, got, tt.want)
		}
	}
}
