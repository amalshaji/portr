package wstunnel

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	serverdb "github.com/amalshaji/portr/internal/server/db"
	"github.com/amalshaji/portr/internal/tunnel/wsproto"
	"github.com/hashicorp/yamux"
	"golang.org/x/net/websocket"
)

// dialTunnel registers an HTTP connection and completes the handshake at the
// given protocol version, returning the manager and the client websocket.
func dialTunnel(t *testing.T, version string) (*Manager, string, *websocket.Conn, wsproto.Frame) {
	t.Helper()
	tunnelService, database := newTestService(t)
	teamUser := serverdb.TeamUser{SecretKey: "secret"}
	if err := database.Create(&teamUser).Error; err != nil {
		t.Fatalf("create team user: %v", err)
	}
	subdomain := "negotiate"
	connection := serverdb.Connection{ID: "conn-" + version, Type: "http", Subdomain: &subdomain, Status: "reserved", CreatedByID: teamUser.ID}
	if err := database.Create(&connection).Error; err != nil {
		t.Fatalf("create connection: %v", err)
	}

	manager := New(nil, tunnelService)
	server := httptest.NewServer(manager.Handler())
	t.Cleanup(server.Close)

	config, err := websocket.NewConfig("ws"+strings.TrimPrefix(server.URL, "http")+wsproto.ConnectPath, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	config.Header = http.Header{}
	config.Header.Set(wsproto.ProtocolVersionHeader, version)
	config.Header.Set(wsproto.ConnectionIDHeader, connection.ID)
	config.Header.Set(wsproto.SecretKeyHeader, "secret")
	conn, err := websocket.DialConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	frame, err := wsproto.Receive(conn)
	if err != nil {
		t.Fatalf("receive handshake frame: %v", err)
	}
	_ = conn.SetReadDeadline(time.Time{})
	return manager, subdomain, conn, frame
}

// visitorProxy serves visitor requests for subdomain through the manager, as
// the proxy server does.
func visitorProxy(t *testing.T, manager *Manager, subdomain string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manager.ServeHTTPStream(subdomain, w, r)
	}))
	t.Cleanup(server.Close)
	return server
}

// proxyRequest sends a GET through the visitor proxy and returns the response
// body.
func proxyRequest(client *http.Client, url string) <-chan string {
	result := make(chan string, 1)
	go func() {
		response, err := client.Get(url)
		if err != nil {
			result <- err.Error()
			return
		}
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)
		result <- string(body)
	}()
	return result
}

// serveStreamRequests answers every HTTP request on stream with body until the
// stream closes.
func serveStreamRequests(stream net.Conn, body string) {
	reader := bufio.NewReader(stream)
	for {
		request, err := http.ReadRequest(reader)
		if err != nil {
			return
		}
		_, _ = io.Copy(io.Discard, request.Body)
		fmt.Fprintf(stream, "HTTP/1.1 200 OK\r\nContent-Length: %d\r\n\r\n%s", len(body), body)
	}
}

func awaitResponse(t *testing.T, response <-chan string, want string) {
	t.Helper()
	select {
	case got := <-response:
		if got != want {
			t.Fatalf("unexpected response %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("visitor did not receive the response")
	}
}

func TestHandlerServesProtocolV2WithYamux(t *testing.T) {
	manager, subdomain, conn, ready := dialTunnel(t, strconv.Itoa(wsproto.ProtocolVersion))
	if ready.Type != wsproto.TypeReady || ready.Version != wsproto.ProtocolVersion {
		t.Fatalf("expected a v2 ready frame, got %#v", ready)
	}
	mux, err := yamux.Client(wsproto.MuxConn(conn), wsproto.MuxConfig(discardLogger))
	if err != nil {
		t.Fatal(err)
	}
	defer mux.Close()
	visitors := visitorProxy(t, manager, subdomain)

	response := proxyRequest(visitors.Client(), visitors.URL)
	stream, err := mux.AcceptStream()
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.ReadRequest(bufio.NewReader(stream))
	if err != nil {
		t.Fatal(err)
	}
	if request.Host != strings.TrimPrefix(visitors.URL, "http://") {
		t.Fatalf("expected the visitor's host, got %q", request.Host)
	}
	_, _ = stream.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 4\r\n\r\npong"))
	awaitResponse(t, response, "pong")
}

func TestHTTPRequestsReuseVisitorConnectionsAndStreams(t *testing.T) {
	manager, subdomain, conn, _ := dialTunnel(t, strconv.Itoa(wsproto.ProtocolVersion))
	mux, err := yamux.Client(wsproto.MuxConn(conn), wsproto.MuxConfig(discardLogger))
	if err != nil {
		t.Fatal(err)
	}
	defer mux.Close()
	var streams atomic.Int32
	go func() {
		for {
			stream, err := mux.AcceptStream()
			if err != nil {
				return
			}
			streams.Add(1)
			go serveStreamRequests(stream, "pong")
		}
	}()

	visitors := visitorProxy(t, manager, subdomain)
	var dials atomic.Int32
	transport := visitors.Client().Transport.(*http.Transport).Clone()
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		dials.Add(1)
		return (&net.Dialer{}).DialContext(ctx, network, addr)
	}
	client := &http.Client{Transport: transport}
	for range 5 {
		awaitResponse(t, proxyRequest(client, visitors.URL), "pong")
	}
	if got := dials.Load(); got != 1 {
		t.Fatalf("expected one visitor connection, got %d", got)
	}
	if got := streams.Load(); got != 1 {
		t.Fatalf("expected one tunnel stream, got %d", got)
	}
}

func TestHandlerStillServesProtocolV1Clients(t *testing.T) {
	manager, subdomain, conn, ready := dialTunnel(t, strconv.Itoa(wsproto.LegacyProtocolVersion))
	if ready.Type != wsproto.TypeReady || ready.Version != wsproto.LegacyProtocolVersion {
		t.Fatalf("expected a v1 ready frame, got %#v", ready)
	}

	visitors := visitorProxy(t, manager, subdomain)
	response := proxyRequest(visitors.Client(), visitors.URL)
	open, err := wsproto.Receive(conn)
	if err != nil {
		t.Fatal(err)
	}
	if open.Type != wsproto.TypeOpen || !strings.HasPrefix(string(open.Data), "GET / HTTP/1.1\r\n") {
		t.Fatalf("expected an open frame carrying the request, got %#v", open)
	}
	writer := wsproto.NewWriter(conn)
	for _, frame := range []wsproto.Frame{
		{Type: wsproto.TypeWindow, StreamID: open.StreamID, Window: wsproto.StreamWindowSize},
		{Type: wsproto.TypeData, StreamID: open.StreamID, Data: []byte("HTTP/1.1 200 OK\r\nContent-Length: 4\r\n\r\npong")},
		{Type: wsproto.TypeClose, StreamID: open.StreamID},
	} {
		if err := writer.Send(frame); err != nil {
			t.Fatal(err)
		}
	}
	awaitResponse(t, response, "pong")
}

func TestHandlerRejectsUnknownProtocolVersion(t *testing.T) {
	_, _, _, frame := dialTunnel(t, "3")
	if frame.Type != wsproto.TypeError || !strings.Contains(frame.Message, "server speaks v1 and v2") {
		t.Fatalf("expected a protocol mismatch error, got %#v", frame)
	}
}
