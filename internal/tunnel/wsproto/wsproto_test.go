package wsproto

import (
	"io"
	"log"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/yamux"
	"golang.org/x/net/websocket"
)

func dialPair(t *testing.T, serve func(*websocket.Conn)) *websocket.Conn {
	t.Helper()
	server := httptest.NewServer(websocket.Handler(serve))
	t.Cleanup(server.Close)
	conn, err := websocket.Dial("ws"+strings.TrimPrefix(server.URL, "http"), "", server.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestYamuxSessionOverMuxConn(t *testing.T) {
	conn := dialPair(t, func(ws *websocket.Conn) {
		session, err := yamux.Server(MuxConn(ws), MuxConfig(log.New(io.Discard, "", 0)))
		if err != nil {
			return
		}
		defer session.Close()
		stream, err := session.Accept()
		if err != nil {
			return
		}
		_, _ = io.Copy(stream, stream)
		_ = stream.Close()
	})
	session, err := yamux.Client(MuxConn(conn), MuxConfig(log.New(io.Discard, "", 0)))
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	stream, err := session.OpenStream()
	if err != nil {
		t.Fatal(err)
	}
	payload := strings.Repeat("portr", 200_000)
	go func() {
		_, _ = stream.Write([]byte(payload))
		_ = stream.Close()
	}()
	echoed, err := io.ReadAll(stream)
	if err != nil {
		t.Fatal(err)
	}
	if string(echoed) != payload {
		t.Fatalf("echo mismatch: got %d bytes, want %d", len(echoed), len(payload))
	}
}
