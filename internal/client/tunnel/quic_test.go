package tunnel

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	clientcfg "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/amalshaji/portr/internal/tunnel/quicproto"
	"github.com/quic-go/quic-go"
)

// fakeQUICServer runs handle for every accepted connection after reading the
// client's hello frame.
func fakeQUICServer(t *testing.T, handle func(conn *quic.Conn, control *quic.Stream, hello quicproto.Frame)) (addr, pin string) {
	t.Helper()
	key, err := quicproto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	pin, err = quicproto.KeyPin(key)
	if err != nil {
		t.Fatal(err)
	}
	tlsConfig, err := quicproto.ServerTLSConfig(key)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := quic.ListenAddr("127.0.0.1:0", tlsConfig, quicproto.ServerConfig())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept(context.Background())
			if err != nil {
				return
			}
			go func() {
				control, err := conn.AcceptStream(context.Background())
				if err != nil {
					return
				}
				hello, err := quicproto.ReadFrame(control)
				if err != nil {
					return
				}
				handle(conn, control, hello)
			}()
		}
	}()
	return listener.Addr().String(), pin
}

func quicClientConfig(addr, pin string) clientcfg.ClientConfig {
	return clientcfg.ClientConfig{QuicUrl: addr, QuicCertSha256: pin, SecretKey: "secret"}
}

func TestConnectQUICAcceptsServerStreams(t *testing.T) {
	addr, pin := fakeQUICServer(t, func(conn *quic.Conn, control *quic.Stream, hello quicproto.Frame) {
		if hello.ConnectionID != "conn-1" || hello.SecretKey != "secret" || hello.Version != quicproto.ProtocolVersion {
			_ = conn.CloseWithError(1, "bad hello")
			return
		}
		_ = quicproto.WriteFrame(control, quicproto.Frame{Type: quicproto.TypeReady, Version: quicproto.ProtocolVersion, Port: 31000})
		go func() {
			for {
				frame, err := quicproto.ReadFrame(control)
				if err != nil {
					return
				}
				if frame.Type == quicproto.TypePing {
					_ = quicproto.WriteFrame(control, quicproto.Frame{Type: quicproto.TypePong})
				}
			}
		}()
		stream, err := conn.OpenStreamSync(context.Background())
		if err != nil {
			return
		}
		_, _ = stream.Write(append([]byte{quicproto.DataStreamHeader}, "payload"...))
		_ = stream.Close()
	})

	session, err := ConnectQUIC(context.Background(), quicClientConfig(addr, pin), "conn-1")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()
	if session.RemotePort() != 31000 {
		t.Fatalf("remote port = %d, want 31000", session.RemotePort())
	}

	stream, err := session.Accept()
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	defer stream.Close()
	_ = stream.SetReadDeadline(time.Now().Add(2 * time.Second))
	body, err := io.ReadAll(stream)
	if err != nil || string(body) != "payload" {
		t.Fatalf("stream body = %q, %v", body, err)
	}
	if _, ok := stream.(interface{ CloseWrite() error }); !ok {
		t.Fatal("quic stream does not support half-close")
	}

	if err := session.HealthCheck(2 * time.Second); err != nil {
		t.Fatalf("health check: %v", err)
	}
}

func TestConnectQUICReportsServerRejection(t *testing.T) {
	addr, pin := fakeQUICServer(t, func(conn *quic.Conn, _ *quic.Stream, _ quicproto.Frame) {
		_ = conn.CloseWithError(1, "invalid connection credentials")
	})

	_, err := ConnectQUIC(context.Background(), quicClientConfig(addr, pin), "conn-1")
	if err == nil || err.Error() != "invalid connection credentials" {
		t.Fatalf("error = %v, want server rejection message", err)
	}
	if errors.Is(err, ErrQUICUnreachable) {
		t.Fatal("rejection must not trigger websocket fallback")
	}
}

func TestConnectQUICRejectsWrongPinWithoutFallback(t *testing.T) {
	addr, _ := fakeQUICServer(t, func(*quic.Conn, *quic.Stream, quicproto.Frame) {})
	otherKey, err := quicproto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	otherPin, err := quicproto.KeyPin(otherKey)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ConnectQUIC(context.Background(), quicClientConfig(addr, otherPin), "conn-1")
	if err == nil || !strings.Contains(err.Error(), "pin mismatch") {
		t.Fatalf("error = %v, want pin mismatch", err)
	}
	if errors.Is(err, ErrQUICUnreachable) {
		t.Fatal("pin mismatch must not trigger websocket fallback")
	}
}

func TestConnectQUICReportsUnreachableEndpoint(t *testing.T) {
	previous := quicDialTimeout
	quicDialTimeout = 300 * time.Millisecond
	t.Cleanup(func() { quicDialTimeout = previous })

	// A bound UDP socket that never answers stands in for a blocked port.
	silent, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer silent.Close()

	started := time.Now()
	_, err = ConnectQUIC(context.Background(), quicClientConfig(silent.LocalAddr().String(), "00"), "conn-1")
	if !errors.Is(err, ErrQUICUnreachable) {
		t.Fatalf("error = %v, want ErrQUICUnreachable", err)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("unreachable endpoint took %s to fail", elapsed)
	}
}

func TestConnectQUICRequiresPin(t *testing.T) {
	_, err := ConnectQUIC(context.Background(), clientcfg.ClientConfig{QuicUrl: "127.0.0.1:1"}, "conn-1")
	if err == nil || !strings.Contains(err.Error(), "quic_cert_sha256") {
		t.Fatalf("error = %v, want missing pin error", err)
	}
}

func TestQUICAcceptRejectsMissingHeader(t *testing.T) {
	addr, pin := fakeQUICServer(t, func(conn *quic.Conn, control *quic.Stream, _ quicproto.Frame) {
		_ = quicproto.WriteFrame(control, quicproto.Frame{Type: quicproto.TypeReady, Version: quicproto.ProtocolVersion})
		stream, err := conn.OpenStreamSync(context.Background())
		if err != nil {
			return
		}
		_, _ = stream.Write([]byte("no header"))
	})

	session, err := ConnectQUIC(context.Background(), quicClientConfig(addr, pin), "conn-1")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()
	if _, err := session.Accept(); err == nil || !strings.Contains(err.Error(), "header") {
		t.Fatalf("accept error = %v, want header error", err)
	}
}
