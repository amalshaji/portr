package dockerwatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func newUnixAPI(t *testing.T, handler http.Handler) *API {
	t.Helper()
	socket := filepath.Join(t.TempDir(), "docker.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	server := &httptest.Server{Listener: listener, Config: &http.Server{Handler: handler}}
	server.Start()
	t.Cleanup(server.Close)

	api, err := NewAPI("unix://" + socket)
	if err != nil {
		t.Fatalf("new api: %v", err)
	}
	return api
}

func TestAPIInspectAndList(t *testing.T) {
	var gotFilters Filters
	mux := http.NewServeMux()
	mux.HandleFunc("/containers/abc/json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Id":"abc","Name":"/shop-web-1","Config":{"Labels":{"portr.enable":"true"},"ExposedPorts":{"3000/tcp":{}}},
			"State":{"Running":true},"NetworkSettings":{"Networks":{"shop_default":{"NetworkID":"n1","IPAddress":"172.18.0.3"}}}}`)
	})
	mux.HandleFunc("/containers/missing/json", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"message":"No such container: missing"}`)
	})
	mux.HandleFunc("/containers/json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.Unmarshal([]byte(r.URL.Query().Get("filters")), &gotFilters)
		fmt.Fprint(w, `[{"Id":"abc"},{"Id":"def"}]`)
	})
	api := newUnixAPI(t, mux)

	container, err := api.Inspect(context.Background(), "abc")
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if container.ID != "abc" || !container.State.Running || container.NetworkSettings.Networks["shop_default"].IPAddress != "172.18.0.3" {
		t.Fatalf("unexpected container: %+v", container)
	}
	if _, ok := container.Config.ExposedPorts["3000/tcp"]; !ok {
		t.Fatalf("exposed ports not decoded: %+v", container.Config.ExposedPorts)
	}

	if _, err := api.Inspect(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	ids, err := api.List(context.Background(), Filters{"label": {"portr.enable=true"}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(ids) != 2 || ids[0] != "abc" || ids[1] != "def" {
		t.Fatalf("unexpected ids: %v", ids)
	}
	if len(gotFilters["label"]) != 1 || gotFilters["label"][0] != "portr.enable=true" {
		t.Fatalf("filters not sent: %v", gotFilters)
	}
}

func TestAPIEventsStreamsUntilClosed(t *testing.T) {
	release := make(chan struct{})
	api := newUnixAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/events" || r.URL.Query().Get("filters") == "" {
			http.Error(w, `{"message":"bad request"}`, http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-release
		fmt.Fprint(w, `{"Type":"container","Action":"start","Actor":{"ID":"abc","Attributes":{"name":"web"}}}`+"\n")
		fmt.Fprint(w, `{"Type":"container","Action":"die","Actor":{"ID":"abc"}}`+"\n")
	}))

	events, errs, err := api.Events(context.Background(), Filters{"type": {"container"}})
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	close(release)

	for _, want := range []string{"start", "die"} {
		select {
		case event := <-events:
			if event.Action != want || event.Actor.ID != "abc" {
				t.Fatalf("unexpected event: %+v", event)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for %s", want)
		}
	}

	select {
	case err := <-errs:
		if err == nil {
			t.Fatal("expected an error when the stream closes")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("stream end was not reported")
	}
}

func TestAPIEventsReportsDaemonError(t *testing.T) {
	api := newUnixAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"message":"boom"}`)
	}))

	if _, _, err := api.Events(context.Background(), nil); err == nil {
		t.Fatal("expected an error")
	}
}

func TestNewAPIRejectsUnknownScheme(t *testing.T) {
	if _, err := NewAPI("ssh://user@host"); err == nil {
		t.Fatal("expected an error for ssh://")
	}
	api, err := NewAPI("tcp://127.0.0.1:2375")
	if err != nil || api.base != "http://127.0.0.1:2375" {
		t.Fatalf("unexpected tcp api: %+v %v", api, err)
	}
}
