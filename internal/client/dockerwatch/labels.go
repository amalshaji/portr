package dockerwatch

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/amalshaji/portr/internal/client/appserver"
	"github.com/amalshaji/portr/internal/constants"
)

const (
	LabelPrefix     = "portr."
	LabelEnable     = "portr.enable"
	LabelType       = "portr.type"
	LabelPort       = "portr.port"
	LabelSubdomain  = "portr.subdomain"
	LabelName       = "portr.name"
	LabelBasicAuth  = "portr.basic_auth"
	LabelHostHeader = "portr.host_header"

	ComposeProjectLabel = "com.docker.compose.project"
	ComposeServiceLabel = "com.docker.compose.service"
)

var knownLabels = map[string]bool{
	LabelEnable:     true,
	LabelType:       true,
	LabelPort:       true,
	LabelSubdomain:  true,
	LabelName:       true,
	LabelBasicAuth:  true,
	LabelHostHeader: true,
}

var invalidSubdomainChars = regexp.MustCompile(`[^a-z0-9-]+`)

// tunnelRequest turns a container's portr.* labels into a tunnel request.
// host is the address the portr container reaches the service on.
func tunnelRequest(container Container, host string) (appserver.StartTunnelRequest, error) {
	labels := container.Config.Labels

	var unknown []string
	for key := range labels {
		if strings.HasPrefix(key, LabelPrefix) && !knownLabels[key] {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return appserver.StartTunnelRequest{}, fmt.Errorf("unknown label %s", strings.Join(unknown, ", "))
	}

	tunnelType := constants.ConnectionType(strings.ToLower(strings.TrimSpace(labels[LabelType])))
	switch tunnelType {
	case "":
		tunnelType = constants.Http
	case constants.Http, constants.Tcp:
	default:
		return appserver.StartTunnelRequest{}, fmt.Errorf("%s must be http or tcp, got %q", LabelType, labels[LabelType])
	}

	port, err := tunnelPort(container)
	if err != nil {
		return appserver.StartTunnelRequest{}, err
	}

	request := appserver.StartTunnelRequest{
		Name:       tunnelName(container),
		Type:       tunnelType,
		Host:       host,
		Port:       port,
		BasicAuth:  labels[LabelBasicAuth],
		HostHeader: labels[LabelHostHeader],
	}

	subdomain := strings.TrimSpace(labels[LabelSubdomain])
	if tunnelType == constants.Tcp {
		if subdomain != "" {
			return appserver.StartTunnelRequest{}, fmt.Errorf("%s is only supported for http tunnels", LabelSubdomain)
		}
		return request, nil
	}

	if subdomain == "" {
		subdomain = defaultSubdomain(labels[ComposeServiceLabel], labels[ComposeProjectLabel])
	}
	request.Subdomain = subdomain
	return request, nil
}

// tunnelPort reads portr.port, falling back to the only exposed TCP port.
func tunnelPort(container Container) (int, error) {
	if raw := strings.TrimSpace(container.Config.Labels[LabelPort]); raw != "" {
		port, err := strconv.Atoi(raw)
		if err != nil || port < 1 || port > 65535 {
			return 0, fmt.Errorf("%s must be a port between 1 and 65535, got %q", LabelPort, raw)
		}
		return port, nil
	}

	var ports []int
	for spec := range container.Config.ExposedPorts {
		number, proto, _ := strings.Cut(spec, "/")
		if proto != "" && proto != "tcp" {
			continue
		}
		if port, err := strconv.Atoi(number); err == nil {
			ports = append(ports, port)
		}
	}

	switch len(ports) {
	case 1:
		return ports[0], nil
	case 0:
		return 0, fmt.Errorf("%s is required: the container exposes no tcp port", LabelPort)
	default:
		sort.Ints(ports)
		return 0, fmt.Errorf("%s is required: the container exposes several ports %v", LabelPort, ports)
	}
}

func tunnelName(container Container) string {
	if name := strings.TrimSpace(container.Config.Labels[LabelName]); name != "" {
		return name
	}
	if service := container.Config.Labels[ComposeServiceLabel]; service != "" {
		return service
	}
	return strings.TrimPrefix(container.Name, "/")
}

// defaultSubdomain derives <service>-<project>, reduced to the characters a
// subdomain allows. Validation still happens when the tunnel starts.
func defaultSubdomain(service, project string) string {
	parts := make([]string, 0, 2)
	for _, part := range []string{service, project} {
		part = invalidSubdomainChars.ReplaceAllString(strings.ToLower(part), "-")
		part = strings.Trim(part, "-")
		if part != "" {
			parts = append(parts, part)
		}
	}

	subdomain := strings.Join(parts, "-")
	if len(subdomain) > 63 {
		subdomain = strings.TrimRight(subdomain[:63], "-")
	}
	return subdomain
}
