package dockerwatch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/amalshaji/portr/internal/client/appserver"
	"github.com/charmbracelet/log"
)

const (
	selfID     = "self000000000000000000000000000000000000000000000000000000000000"
	networkID  = "net-shop"
	otherNetID = "net-other"
)

type fakeDocker struct {
	mu         sync.Mutex
	containers map[string]Container
	running    []string
	listErr    error
	filters    []Filters
	inspects   map[string]int
	eventCtxs  []context.Context
	streams    chan chan Event
	streamErrs chan chan error
}

func newFakeDocker() *fakeDocker {
	self := Container{ID: selfID}
	self.Config.Labels = map[string]string{ComposeProjectLabel: "shop"}
	self.NetworkSettings.Networks = map[string]Network{"shop_default": {NetworkID: networkID, IPAddress: "172.18.0.2"}}

	return &fakeDocker{
		containers: map[string]Container{selfID: self},
		inspects:   map[string]int{},
		streams:    make(chan chan Event, 64),
		streamErrs: make(chan chan error, 64),
	}
}

func (d *fakeDocker) add(id string, labels map[string]string, networks map[string]Network) {
	container := labelledContainer(labels, "3000/tcp")
	container.ID = id
	container.State.Running = true
	container.NetworkSettings.Networks = networks
	d.mu.Lock()
	d.containers[id] = container
	d.mu.Unlock()
}

func (d *fakeDocker) Inspect(_ context.Context, id string) (Container, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.inspects[id]++
	container, ok := d.containers[id]
	if !ok {
		return Container{}, fmt.Errorf("%w: no such container %s", ErrNotFound, id)
	}
	return container, nil
}

func (d *fakeDocker) List(_ context.Context, filters Filters) ([]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.filters = append(d.filters, filters)
	return slices.Clone(d.running), d.listErr
}

func (d *fakeDocker) Events(ctx context.Context, filters Filters) (<-chan Event, <-chan error, error) {
	events := make(chan Event)
	errs := make(chan error, 1)
	d.mu.Lock()
	d.filters = append(d.filters, filters)
	d.eventCtxs = append(d.eventCtxs, ctx)
	d.mu.Unlock()
	d.streams <- events
	d.streamErrs <- errs
	return events, errs, nil
}

type fakeTunnels struct {
	mu      sync.Mutex
	next    int
	started map[string]appserver.StartTunnelRequest
	stopped []string
	failFor string
	changed chan struct{}
}

func newFakeTunnels() *fakeTunnels {
	return &fakeTunnels{started: map[string]appserver.StartTunnelRequest{}, changed: make(chan struct{}, 1024)}
}

func (f *fakeTunnels) StartTunnel(_ context.Context, request appserver.StartTunnelRequest) (appserver.TunnelStatus, error) {
	f.mu.Lock()
	defer func() { f.changed <- struct{}{} }()
	defer f.mu.Unlock()
	if request.Name == f.failFor {
		return appserver.TunnelStatus{}, errors.New("subdomain in use")
	}
	f.next++
	id := fmt.Sprintf("tunnel-%d", f.next)
	f.started[id] = request
	return appserver.TunnelStatus{ID: id}, nil
}

func (f *fakeTunnels) StopTunnel(_ context.Context, id string) (appserver.TunnelStatus, error) {
	f.mu.Lock()
	defer func() { f.changed <- struct{}{} }()
	defer f.mu.Unlock()
	f.stopped = append(f.stopped, id)
	return appserver.TunnelStatus{ID: id}, nil
}

func (f *fakeTunnels) snapshot() (map[string]appserver.StartTunnelRequest, []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	started := make(map[string]appserver.StartTunnelRequest, len(f.started))
	for id, request := range f.started {
		started[id] = request
	}
	return started, slices.Clone(f.stopped)
}

func (f *fakeTunnels) wait(t *testing.T, changes int) {
	t.Helper()
	for i := 0; i < changes; i++ {
		select {
		case <-f.changed:
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for tunnel change %d of %d", i+1, changes)
		}
	}
}

func newTestWatcher(docker *fakeDocker, tunnels *fakeTunnels) *Watcher {
	watcher := New(docker, tunnels, Options{Logger: log.New(io.Discard)})
	watcher.selfIDs = []string{"missing-hostname", selfID}
	watcher.reconnectDelay = time.Millisecond
	return watcher
}

// runWatcher starts the watcher and returns a stop function that cancels it
// and returns Run's result. Stop is safe to call more than once.
func runWatcher(t *testing.T, watcher *Watcher) func() error {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- watcher.Run(ctx) }()

	var once sync.Once
	var result error
	stop := func() error {
		once.Do(func() {
			cancel()
			select {
			case result = <-done:
			case <-time.After(2 * time.Second):
				result = errors.New("watcher did not stop")
			}
		})
		return result
	}
	t.Cleanup(func() {
		if err := stop(); err != nil {
			t.Error(err)
		}
	})
	return stop
}

func sharedNetwork(ip string) map[string]Network {
	return map[string]Network{"shop_default": {NetworkID: networkID, IPAddress: ip}}
}

func startEvent(id string) Event {
	var event Event
	event.Type = "container"
	event.Action = "start"
	event.Actor.ID = id
	return event
}

func dieEvent(id string) Event {
	event := startEvent(id)
	event.Action = "die"
	return event
}

func TestWatcherStartsRunningContainersAndFollowsEvents(t *testing.T) {
	docker := newFakeDocker()
	tunnels := newFakeTunnels()
	docker.add("web", nil, sharedNetwork("172.18.0.3"))
	docker.running = []string{"web"}

	runWatcher(t, newTestWatcher(docker, tunnels))
	events := <-docker.streams
	tunnels.wait(t, 1)

	started, _ := tunnels.snapshot()
	if got := started["tunnel-1"]; got.Host != "172.18.0.3" || got.Port != 3000 || got.Subdomain != "web-shop" {
		t.Fatalf("unexpected tunnel for running container: %+v", got)
	}

	docker.add("api", map[string]string{ComposeServiceLabel: "api"}, sharedNetwork("172.18.0.4"))
	events <- startEvent("api")
	tunnels.wait(t, 1)

	// A duplicate start for a container that already has a tunnel is ignored.
	events <- startEvent("api")
	events <- dieEvent("web")
	tunnels.wait(t, 1)

	started, stopped := tunnels.snapshot()
	if len(started) != 2 || started["tunnel-2"].Subdomain != "api-shop" {
		t.Fatalf("unexpected started tunnels: %+v", started)
	}
	if !slices.Equal(stopped, []string{"tunnel-1"}) {
		t.Fatalf("expected tunnel-1 to stop, got %v", stopped)
	}
}

func TestWatcherFiltersByProjectAndLabel(t *testing.T) {
	docker := newFakeDocker()
	tunnels := newFakeTunnels()

	runWatcher(t, newTestWatcher(docker, tunnels))
	<-docker.streams

	deadline := time.After(2 * time.Second)
	for {
		docker.mu.Lock()
		count := len(docker.filters)
		docker.mu.Unlock()
		if count >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("watcher never listed containers")
		case <-time.After(5 * time.Millisecond):
		}
	}

	docker.mu.Lock()
	defer docker.mu.Unlock()
	eventFilters, listFilters := docker.filters[0], docker.filters[1]
	wantLabels := []string{"com.docker.compose.project=shop", "portr.enable=true"}
	if !slices.Equal(eventFilters["label"], wantLabels) || !slices.Equal(listFilters["label"], wantLabels) {
		t.Fatalf("unexpected label filters: events=%v list=%v", eventFilters["label"], listFilters["label"])
	}
	if !slices.Equal(eventFilters["type"], []string{"container"}) || !slices.Equal(eventFilters["event"], []string{"start", "die"}) {
		t.Fatalf("unexpected event filters: %v", eventFilters)
	}
}

func TestWatcherSkipsUnreachableAndInvalidContainers(t *testing.T) {
	docker := newFakeDocker()
	tunnels := newFakeTunnels()
	tunnels.failFor = "busy"
	docker.add("isolated", nil, map[string]Network{"other": {NetworkID: otherNetID, IPAddress: "10.1.0.2"}})
	docker.add("typo", map[string]string{"portr.prot": "3000"}, sharedNetwork("172.18.0.5"))
	docker.add("busy", map[string]string{LabelName: "busy"}, sharedNetwork("172.18.0.6"))
	docker.add("ok", map[string]string{ComposeServiceLabel: "ok"}, sharedNetwork("172.18.0.7"))
	docker.running = []string{"busy", "isolated", "ok", "typo"}

	runWatcher(t, newTestWatcher(docker, tunnels))
	<-docker.streams
	tunnels.wait(t, 2)

	started, _ := tunnels.snapshot()
	if len(started) != 1 || started["tunnel-1"].Name != "ok" {
		t.Fatalf("expected only the valid container to get a tunnel, got %+v", started)
	}
}

func TestWatcherResyncsAfterReconnect(t *testing.T) {
	docker := newFakeDocker()
	tunnels := newFakeTunnels()
	docker.add("web", nil, sharedNetwork("172.18.0.3"))
	docker.add("api", map[string]string{ComposeServiceLabel: "api"}, sharedNetwork("172.18.0.4"))
	docker.running = []string{"web"}

	runWatcher(t, newTestWatcher(docker, tunnels))
	<-docker.streams
	errs := <-docker.streamErrs
	tunnels.wait(t, 1)

	// While disconnected, web died and api started.
	docker.mu.Lock()
	docker.running = []string{"api"}
	docker.mu.Unlock()
	errs <- errors.New("connection reset")

	<-docker.streams
	tunnels.wait(t, 2)

	started, stopped := tunnels.snapshot()
	if !slices.Equal(stopped, []string{"tunnel-1"}) || started["tunnel-2"].Name != "api" {
		t.Fatalf("reconnect did not reconcile: started=%+v stopped=%v", started, stopped)
	}
}

func TestWatcherFailsWithoutComposeProject(t *testing.T) {
	docker := newFakeDocker()
	self := docker.containers[selfID]
	self.Config.Labels = nil
	docker.containers[selfID] = self

	err := newTestWatcher(docker, newFakeTunnels()).Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "--project") {
		t.Fatalf("expected a --project hint, got %v", err)
	}
}

func TestWatcherProjectOverride(t *testing.T) {
	docker := newFakeDocker()
	watcher := newTestWatcher(docker, newFakeTunnels())
	watcher.projectOverride = "other"

	if err := watcher.resolveSelf(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if watcher.project != "other" {
		t.Fatalf("expected project override, got %q", watcher.project)
	}
}

func TestWatcherFailsOutsideContainer(t *testing.T) {
	watcher := newTestWatcher(newFakeDocker(), newFakeTunnels())
	watcher.selfIDs = []string{"laptop"}

	err := watcher.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "could not find the portr container") {
		t.Fatalf("expected a missing container error, got %v", err)
	}
}

func TestContainerIDFromMountinfo(t *testing.T) {
	id := strings.Repeat("ab12", 16)
	mountinfo := "620 610 0:52 / / rw,relatime - overlay overlay rw\n" +
		"640 620 254:1 /var/lib/docker/containers/" + id + "/hostname /etc/hostname rw,relatime - ext4 /dev/vda1 rw\n"

	if got := containerIDFromMountinfo(strings.NewReader(mountinfo)); got != id {
		t.Fatalf("expected %s, got %q", id, got)
	}
	if got := containerIDFromMountinfo(strings.NewReader("1 0 0:1 / / rw - ext4 /dev/sda1 rw\n")); got != "" {
		t.Fatalf("expected no id outside docker, got %q", got)
	}
}

func waitFor(t *testing.T, what string, done func() bool) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for !done() {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %s", what)
		case <-time.After(2 * time.Millisecond):
		}
	}
}

func TestWatcherReleasesEventStreamWhenListFails(t *testing.T) {
	docker := newFakeDocker()
	docker.listErr = errors.New("daemon busy")

	runWatcher(t, newTestWatcher(docker, newFakeTunnels()))
	<-docker.streams

	docker.mu.Lock()
	streamCtx := docker.eventCtxs[0]
	docker.mu.Unlock()
	select {
	case <-streamCtx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("the event subscription was not cancelled after the list failed")
	}

	docker.mu.Lock()
	docker.listErr = nil
	docker.mu.Unlock()
}

func TestWatcherRetriesFailedStarts(t *testing.T) {
	docker := newFakeDocker()
	tunnels := newFakeTunnels()
	tunnels.failFor = "web"
	docker.add("web", nil, sharedNetwork("172.18.0.3"))
	docker.running = []string{"web"}

	watcher := newTestWatcher(docker, tunnels)
	watcher.resyncInterval = 2 * time.Millisecond
	watcher.retryDelay = time.Millisecond
	runWatcher(t, watcher)
	tunnels.wait(t, 1)

	// The portr server comes back.
	tunnels.mu.Lock()
	tunnels.failFor = ""
	tunnels.mu.Unlock()

	waitFor(t, "the tunnel to be retried", func() bool {
		started, _ := tunnels.snapshot()
		return len(started) == 1
	})
}

func TestWatcherRetryBacksOff(t *testing.T) {
	docker := newFakeDocker()
	tunnels := newFakeTunnels()
	tunnels.failFor = "web"
	docker.add("web", nil, sharedNetwork("172.18.0.3"))

	watcher := newTestWatcher(docker, tunnels)
	watcher.retryDelay = time.Hour
	watcher.maxRetryDelay = 2 * time.Hour
	if err := watcher.resolveSelf(context.Background()); err != nil {
		t.Fatalf("resolve self: %v", err)
	}

	for i := 0; i < 3; i++ {
		watcher.sync(context.Background(), []string{"web"})
	}

	retry := watcher.retries["web"]
	if retry.attempts != 1 {
		t.Fatalf("expected one attempt inside the backoff window, got %d", retry.attempts)
	}
	if wait := time.Until(retry.next); wait < 59*time.Minute {
		t.Fatalf("expected the next attempt about an hour away, got %s", wait)
	}

	retry.next = time.Now()
	watcher.retries["web"] = retry
	watcher.sync(context.Background(), []string{"web"})
	if retry := watcher.retries["web"]; retry.attempts != 2 || time.Until(retry.next) < 119*time.Minute {
		t.Fatalf("expected the delay to double up to the cap, got %+v", retry)
	}

	// A container that is no longer running is forgotten.
	watcher.sync(context.Background(), nil)
	if _, ok := watcher.retries["web"]; ok {
		t.Fatal("expected retry state to be cleared for a stopped container")
	}
}

func TestWatcherDoesNotReinspectSkippedContainers(t *testing.T) {
	docker := newFakeDocker()
	tunnels := newFakeTunnels()
	docker.add("typo", map[string]string{"portr.prot": "3000"}, sharedNetwork("172.18.0.5"))
	docker.running = []string{"typo"}

	watcher := newTestWatcher(docker, tunnels)
	watcher.resyncInterval = time.Millisecond
	runWatcher(t, watcher)
	events := <-docker.streams

	waitFor(t, "several resyncs", func() bool {
		docker.mu.Lock()
		defer docker.mu.Unlock()
		return len(docker.filters) > 6
	})
	docker.mu.Lock()
	inspects := docker.inspects["typo"]
	docker.mu.Unlock()
	if inspects != 1 {
		t.Fatalf("expected a skipped container to be inspected once, got %d", inspects)
	}

	// Starting again re-evaluates it, e.g. after the labels were fixed.
	events <- startEvent("typo")
	waitFor(t, "a fresh inspect on start", func() bool {
		docker.mu.Lock()
		defer docker.mu.Unlock()
		return docker.inspects["typo"] == 2
	})
}
