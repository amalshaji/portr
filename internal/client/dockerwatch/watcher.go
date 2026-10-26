package dockerwatch

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"time"

	"github.com/amalshaji/portr/internal/client/appserver"
	"github.com/charmbracelet/log"
)

type Docker interface {
	Inspect(ctx context.Context, id string) (Container, error)
	List(ctx context.Context, filters Filters) ([]string, error)
	Events(ctx context.Context, filters Filters) (<-chan Event, <-chan error, error)
}

type Tunnels interface {
	StartTunnel(ctx context.Context, request appserver.StartTunnelRequest) (appserver.TunnelStatus, error)
	StopTunnel(ctx context.Context, id string) (appserver.TunnelStatus, error)
}

type Options struct {
	// Project overrides the compose project read from the portr container.
	Project string
	Logger  *log.Logger
}

type Watcher struct {
	docker  Docker
	tunnels Tunnels
	logger  *log.Logger

	projectOverride string
	selfIDs         []string

	self    Container
	project string
	// active maps a container ID to the tunnel started for it.
	active map[string]string
	// skipped holds containers whose labels or networking are invalid. They
	// are not retried until the container starts again.
	skipped map[string]bool
	// retries holds containers whose tunnel failed to start, for example
	// while the portr server was unreachable.
	retries map[string]retryState

	reconnectDelay time.Duration
	resyncInterval time.Duration
	retryDelay     time.Duration
	maxRetryDelay  time.Duration
}

type retryState struct {
	attempts int
	next     time.Time
}

func New(docker Docker, tunnels Tunnels, opts Options) *Watcher {
	logger := opts.Logger
	if logger == nil {
		logger = log.WithPrefix("docker")
	}

	return &Watcher{
		docker:          docker,
		tunnels:         tunnels,
		logger:          logger,
		projectOverride: opts.Project,
		selfIDs:         selfContainerCandidates(),
		active:          make(map[string]string),
		skipped:         make(map[string]bool),
		retries:         make(map[string]retryState),
		reconnectDelay:  2 * time.Second,
		resyncInterval:  15 * time.Second,
		retryDelay:      15 * time.Second,
		maxRetryDelay:   5 * time.Minute,
	}
}

// Run watches the compose project until ctx is cancelled. It fails fast when
// the daemon or the portr container cannot be found, and afterwards
// reconnects to the event stream for as long as ctx lives.
func (w *Watcher) Run(ctx context.Context) error {
	if err := w.resolveSelf(ctx); err != nil {
		return err
	}
	w.logger.Info("Watching compose project", "project", w.project)

	for {
		err := w.watch(ctx)
		if ctx.Err() != nil {
			return nil
		}
		w.logger.Warn("Lost docker event stream, reconnecting", "error", err)

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(w.reconnectDelay):
		}

		if err := w.resolveSelf(ctx); err != nil && ctx.Err() == nil {
			w.logger.Warn("Failed to refresh portr container details", "error", err)
		}
	}
}

func (w *Watcher) filters() Filters {
	return Filters{
		"label": {ComposeProjectLabel + "=" + w.project, LabelEnable + "=true"},
	}
}

func (w *Watcher) watch(ctx context.Context) error {
	// Cancelling on return releases the event stream on every exit path.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	eventFilters := w.filters()
	eventFilters["type"] = []string{"container"}
	eventFilters["event"] = []string{"start", "die"}

	// Subscribe before listing so a container that starts in between is seen
	// by at least one of the two; handling a start twice is a no-op.
	events, errs, err := w.docker.Events(ctx, eventFilters)
	if err != nil {
		return err
	}

	if err := w.resync(ctx); err != nil {
		return err
	}

	// The periodic resync retries tunnels that failed to start.
	ticker := time.NewTicker(w.resyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errs:
			return err
		case <-ticker.C:
			if err := w.resync(ctx); err != nil {
				return err
			}
		case event := <-events:
			switch event.Action {
			case "start":
				w.forget(event.Actor.ID)
				w.start(ctx, event.Actor.ID)
			case "die":
				w.stop(ctx, event.Actor.ID)
				w.forget(event.Actor.ID)
			}
		}
	}
}

func (w *Watcher) resync(ctx context.Context) error {
	running, err := w.docker.List(ctx, w.filters())
	if err != nil {
		return err
	}
	w.sync(ctx, running)
	return nil
}

// forget clears skip and retry state so a restarted container is evaluated
// from scratch.
func (w *Watcher) forget(containerID string) {
	delete(w.skipped, containerID)
	delete(w.retries, containerID)
}

// sync reconciles tunnels with the running containers, which matters after a
// reconnect: events missed while disconnected would otherwise leak tunnels.
func (w *Watcher) sync(ctx context.Context, running []string) {
	isRunning := make(map[string]bool, len(running))
	for _, id := range running {
		isRunning[id] = true
	}

	for containerID := range w.active {
		if !isRunning[containerID] {
			w.stop(ctx, containerID)
		}
	}
	for containerID := range w.skipped {
		if !isRunning[containerID] {
			w.forget(containerID)
		}
	}
	for containerID := range w.retries {
		if !isRunning[containerID] {
			w.forget(containerID)
		}
	}

	sort.Strings(running)
	for _, id := range running {
		w.start(ctx, id)
	}
}

func (w *Watcher) start(ctx context.Context, containerID string) {
	if _, ok := w.active[containerID]; ok || containerID == w.self.ID || w.skipped[containerID] {
		return
	}
	if retry, ok := w.retries[containerID]; ok && time.Now().Before(retry.next) {
		return
	}

	container, err := w.docker.Inspect(ctx, containerID)
	if err != nil {
		w.logger.Error("Failed to inspect container", "container", shortID(containerID), "error", err)
		return
	}
	if !container.State.Running {
		return
	}

	name := tunnelName(container)
	host, err := w.reachableHost(container)
	if err != nil {
		w.skipped[containerID] = true
		w.logger.Error("Skipping container", "service", name, "error", err)
		return
	}

	request, err := tunnelRequest(container, host)
	if err != nil {
		w.skipped[containerID] = true
		w.logger.Error("Skipping container", "service", name, "error", err)
		return
	}

	status, err := w.tunnels.StartTunnel(ctx, request)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		retry := w.retries[containerID]
		retry.attempts++
		delay := w.retryDelay << min(retry.attempts-1, 16)
		if delay <= 0 || delay > w.maxRetryDelay {
			delay = w.maxRetryDelay
		}
		retry.next = time.Now().Add(delay)
		w.retries[containerID] = retry
		w.logger.Error("Failed to start tunnel", "service", name, "error", err, "retry_in", delay)
		return
	}
	delete(w.retries, containerID)
	w.active[containerID] = status.ID
}

func (w *Watcher) stop(ctx context.Context, containerID string) {
	tunnelID, ok := w.active[containerID]
	if !ok {
		return
	}
	delete(w.active, containerID)

	// Stopping must finish even while ctx is being cancelled for shutdown.
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if _, err := w.tunnels.StopTunnel(stopCtx, tunnelID); err != nil {
		w.logger.Error("Failed to stop tunnel", "container", shortID(containerID), "error", err)
	}
}

// reachableHost returns the container's IP on a network it shares with the
// portr container. Networks are compared by ID so two projects that both
// call their network "default" are not confused.
func (w *Watcher) reachableHost(container Container) (string, error) {
	shared := make(map[string]bool)
	for _, network := range w.self.NetworkSettings.Networks {
		if network.NetworkID != "" {
			shared[network.NetworkID] = true
		}
	}

	names := make([]string, 0, len(container.NetworkSettings.Networks))
	for name := range container.NetworkSettings.Networks {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		network := container.NetworkSettings.Networks[name]
		if shared[network.NetworkID] && network.IPAddress != "" {
			return network.IPAddress, nil
		}
	}

	return "", fmt.Errorf("not on a network shared with the portr container (network_mode %q)", container.HostConfig.NetworkMode)
}

func (w *Watcher) resolveSelf(ctx context.Context) error {
	var lastErr error
	for _, id := range w.selfIDs {
		container, err := w.docker.Inspect(ctx, id)
		if err != nil {
			lastErr = err
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return err
		}

		project := w.projectOverride
		if project == "" {
			project = container.Config.Labels[ComposeProjectLabel]
		}
		if project == "" {
			return errors.New("the portr container is not part of a compose project; pass --project")
		}

		w.self = container
		w.project = project
		return nil
	}

	if lastErr == nil {
		lastErr = errors.New("no container id candidates")
	}
	return fmt.Errorf("could not find the portr container; portr docker must run inside a container with the docker socket mounted: %w", lastErr)
}

var containerIDInMountinfo = regexp.MustCompile(`/containers/([0-9a-f]{64})/`)

// selfContainerCandidates lists IDs the current container may have. Docker
// sets the hostname to the short ID unless the compose file overrides it, so
// the ID in mountinfo (from /etc/hostname's bind mount) is the fallback.
func selfContainerCandidates() []string {
	var candidates []string
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		candidates = append(candidates, hostname)
	}

	if file, err := os.Open("/proc/self/mountinfo"); err == nil {
		defer file.Close()
		if id := containerIDFromMountinfo(file); id != "" {
			candidates = append(candidates, id)
		}
	}
	return candidates
}

func containerIDFromMountinfo(r io.Reader) string {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if match := containerIDInMountinfo.FindStringSubmatch(scanner.Text()); match != nil {
			return match[1]
		}
	}
	return ""
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
