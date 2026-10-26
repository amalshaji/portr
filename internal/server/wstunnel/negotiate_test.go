package wstunnel

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
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

// proxyRequest sends request bytes through the manager as a visitor would and
// returns everything the visitor reads back.
func proxyRequest(manager *Manager, subdomain string) <-chan string {
	visitor, proxied := net.Pipe()
	go manager.OpenHTTPStream(subdomain, proxied, []byte("ping"))
	result := make(chan string, 1)
	go func() {
		response, _ := io.ReadAll(visitor)
		result <- string(response)
	}()
	return result
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

	response := proxyRequest(manager, subdomain)
	stream, err := mux.AcceptStream()
	if err != nil {
		t.Fatal(err)
	}
	request := make([]byte, len("ping"))
	if _, err := io.ReadFull(stream, request); err != nil || string(request) != "ping" {
		t.Fatalf("unexpected request %q: %v", request, err)
	}
	_, _ = stream.Write([]byte("pong"))
	_ = stream.Close()
	select {
	case got := <-response:
		if got != "pong" {
			t.Fatalf("unexpected response %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("visitor did not receive the response")
	}
}

func TestHandlerStillServesProtocolV1Clients(t *testing.T) {
	manager, subdomain, conn, ready := dialTunnel(t, strconv.Itoa(wsproto.LegacyProtocolVersion))
	if ready.Type != wsproto.TypeReady || ready.Version != wsproto.LegacyProtocolVersion {
		t.Fatalf("expected a v1 ready frame, got %#v", ready)
	}

	response := proxyRequest(manager, subdomain)
	open, err := wsproto.Receive(conn)
	if err != nil {
		t.Fatal(err)
	}
	if open.Type != wsproto.TypeOpen || string(open.Data) != "ping" {
		t.Fatalf("expected an open frame carrying the request, got %#v", open)
	}
	writer := wsproto.NewWriter(conn)
	for _, frame := range []wsproto.Frame{
		{Type: wsproto.TypeWindow, StreamID: open.StreamID, Window: wsproto.StreamWindowSize},
		{Type: wsproto.TypeData, StreamID: open.StreamID, Data: []byte("pong")},
		{Type: wsproto.TypeClose, StreamID: open.StreamID},
	} {
		if err := writer.Send(frame); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case got := <-response:
		if got != "pong" {
			t.Fatalf("unexpected response %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("visitor did not receive the response")
	}
}

func TestHandlerRejectsUnknownProtocolVersion(t *testing.T) {
	_, _, _, frame := dialTunnel(t, "3")
	if frame.Type != wsproto.TypeError || !strings.Contains(frame.Message, "server speaks v1 and v2") {
		t.Fatalf("expected a protocol mismatch error, got %#v", frame)
	}
}
