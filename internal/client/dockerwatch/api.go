package dockerwatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultDockerHost = "unix:///var/run/docker.sock"

var ErrNotFound = errors.New("not found")

const requestTimeout = 10 * time.Second

type Container struct {
	ID     string `json:"Id"`
	Name   string `json:"Name"`
	Config struct {
		Labels       map[string]string   `json:"Labels"`
		ExposedPorts map[string]struct{} `json:"ExposedPorts"`
	} `json:"Config"`
	State struct {
		Running bool `json:"Running"`
	} `json:"State"`
	HostConfig struct {
		NetworkMode string `json:"NetworkMode"`
	} `json:"HostConfig"`
	NetworkSettings struct {
		Networks map[string]Network `json:"Networks"`
	} `json:"NetworkSettings"`
}

type Network struct {
	NetworkID string `json:"NetworkID"`
	IPAddress string `json:"IPAddress"`
}

type Event struct {
	Type   string `json:"Type"`
	Action string `json:"Action"`
	Actor  struct {
		ID         string            `json:"ID"`
		Attributes map[string]string `json:"Attributes"`
	} `json:"Actor"`
}

// Filters mirrors the Docker Engine API filter format: each key maps to the
// values it must match. Multiple label filters are ANDed by Docker.
type Filters map[string][]string

func (f Filters) encode() (string, error) {
	raw, err := json.Marshal(f)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// API is a minimal Docker Engine API client. It uses unversioned endpoints so
// the daemon picks its own API version, which keeps it working across Docker
// releases that raise the minimum supported version.
type API struct {
	http *http.Client
	base string
}

func NewAPI(host string) (*API, error) {
	if strings.TrimSpace(host) == "" {
		host = DefaultDockerHost
	}

	parsed, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("invalid docker host %q: %w", host, err)
	}

	switch parsed.Scheme {
	case "unix":
		socketPath := parsed.Path
		transport := &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var dialer net.Dialer
				return dialer.DialContext(ctx, "unix", socketPath)
			},
		}
		return &API{http: &http.Client{Transport: transport}, base: "http://docker"}, nil
	case "tcp", "http":
		return &API{http: &http.Client{}, base: "http://" + parsed.Host}, nil
	default:
		return nil, fmt.Errorf("unsupported docker host %q: use unix:// or tcp://", host)
	}
}

func (a *API) Inspect(ctx context.Context, id string) (Container, error) {
	var container Container
	err := a.getJSON(ctx, "/containers/"+url.PathEscape(id)+"/json", nil, &container)
	return container, err
}

// List returns the IDs of running containers matching filters.
func (a *API) List(ctx context.Context, filters Filters) ([]string, error) {
	query, err := filterQuery(filters)
	if err != nil {
		return nil, err
	}

	var summaries []struct {
		ID string `json:"Id"`
	}
	if err := a.getJSON(ctx, "/containers/json", query, &summaries); err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		ids = append(ids, summary.ID)
	}
	return ids, nil
}

// Events subscribes to the event stream. It returns once the daemon has
// accepted the subscription, so a List issued afterwards cannot miss an event.
// The error channel receives exactly one value when the stream ends.
func (a *API) Events(ctx context.Context, filters Filters) (<-chan Event, <-chan error, error) {
	query, err := filterQuery(filters)
	if err != nil {
		return nil, nil, err
	}

	resp, err := a.do(ctx, "/events", query)
	if err != nil {
		return nil, nil, err
	}

	events := make(chan Event)
	errs := make(chan error, 1)
	go func() {
		defer resp.Body.Close()
		decoder := json.NewDecoder(resp.Body)
		for {
			var event Event
			if err := decoder.Decode(&event); err != nil {
				if errors.Is(err, io.EOF) {
					err = errors.New("docker event stream closed")
				}
				errs <- err
				return
			}
			select {
			case events <- event:
			case <-ctx.Done():
				errs <- ctx.Err()
				return
			}
		}
	}()

	return events, errs, nil
}

func filterQuery(filters Filters) (url.Values, error) {
	query := url.Values{}
	if len(filters) == 0 {
		return query, nil
	}
	encoded, err := filters.encode()
	if err != nil {
		return nil, err
	}
	query.Set("filters", encoded)
	return query, nil
}

func (a *API) getJSON(ctx context.Context, path string, query url.Values, out any) error {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	resp, err := a.do(ctx, path, query)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("failed to decode docker response from %s: %w", path, err)
	}
	return nil
}

func (a *API) do(ctx context.Context, path string, query url.Values) (*http.Response, error) {
	target := a.base + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to reach docker daemon: %w", err)
	}

	if resp.StatusCode == http.StatusOK {
		return resp, nil
	}
	defer resp.Body.Close()

	var body struct {
		Message string `json:"message"`
	}
	_ = json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&body)
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, body.Message)
	}
	return nil, fmt.Errorf("docker daemon returned %d for %s: %s", resp.StatusCode, path, body.Message)
}
