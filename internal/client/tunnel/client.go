package tunnel

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	config "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/amalshaji/portr/internal/tunnel/wsproto"
	"github.com/hashicorp/yamux"
	"golang.org/x/net/websocket"
)

type Session struct {
	config     config.ClientConfig
	conn       *websocket.Conn
	mux        *yamux.Session
	legacy     *legacySession
	traffic    *trafficConn
	errors     *muxErrors
	remotePort int
	closeOnce  sync.Once
}

// trafficConn records when bytes last arrived from the server, so a busy
// tunnel whose ping is queued behind data still counts as healthy.
type trafficConn struct {
	net.Conn
	lastRead atomic.Int64
}

func (c *trafficConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if n > 0 {
		c.lastRead.Store(time.Now().UnixNano())
	}
	return n, err
}

func (c *trafficConn) readSince(d time.Duration) bool {
	return time.Since(time.Unix(0, c.lastRead.Load())) < d
}

// muxErrors keeps the first error yamux logs, which explains why a session
// ended better than the generic error Accept returns.
type muxErrors struct {
	mu    sync.Mutex
	first string
}

func (e *muxErrors) Print(v ...any)                 { e.record(fmt.Sprint(v...)) }
func (e *muxErrors) Println(v ...any)               { e.record(fmt.Sprint(v...)) }
func (e *muxErrors) Printf(format string, v ...any) { e.record(fmt.Sprintf(format, v...)) }

func (e *muxErrors) record(message string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.first == "" {
		e.first = strings.TrimSpace(message)
	}
}

func (e *muxErrors) String() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.first
}

// errServerSpeaksV1 marks a handshake rejected by a server that only knows
// protocol v1.
var errServerSpeaksV1 = errors.New("server only speaks websocket protocol v1")

// legacyServers remembers websocket URLs that rejected v2, so reconnects go
// straight to v1 instead of paying a failed handshake each time.
var legacyServers sync.Map

// Connect opens a tunnel session, preferring protocol v2 and falling back to
// v1 for servers released before yamux.
func Connect(ctx context.Context, cfg config.ClientConfig, connectionID string) (*Session, error) {
	wsURL, err := tunnelWebSocketURL(cfg)
	if err != nil {
		return nil, err
	}
	if _, legacy := legacyServers.Load(wsURL); !legacy {
		session, err := connect(ctx, cfg, wsURL, connectionID, wsproto.ProtocolVersion)
		if !errors.Is(err, errServerSpeaksV1) {
			return session, err
		}
		legacyServers.Store(wsURL, true)
	}
	return connect(ctx, cfg, wsURL, connectionID, wsproto.LegacyProtocolVersion)
}

func connect(ctx context.Context, cfg config.ClientConfig, wsURL, connectionID string, version int) (*Session, error) {
	wsConfig, err := websocket.NewConfig(wsURL, cfg.GetServerAddr())
	if err != nil {
		return nil, err
	}
	wsConfig.Header.Set(wsproto.ConnectionIDHeader, connectionID)
	wsConfig.Header.Set(wsproto.SecretKeyHeader, cfg.SecretKey)
	wsConfig.Header.Set(wsproto.ProtocolVersionHeader, strconv.Itoa(version))
	conn, err := wsConfig.DialContext(ctx)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, err
	}

	ready := make(chan wsproto.Frame, 1)
	readyErr := make(chan error, 1)
	go func() {
		frame, receiveErr := wsproto.Receive(conn)
		if receiveErr != nil {
			readyErr <- receiveErr
			return
		}
		ready <- frame
	}()

	var readyFrame wsproto.Frame
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	select {
	case readyFrame = <-ready:
		if readyFrame.Type == wsproto.TypeError {
			_ = conn.Close()
			// A v1 server answers "server speaks v1, client sent ..." to any
			// other version; servers that also speak v2 never send this.
			if version != wsproto.LegacyProtocolVersion && strings.Contains(readyFrame.Message, "server speaks v1,") {
				return nil, fmt.Errorf("%w: %s", errServerSpeaksV1, readyFrame.Message)
			}
			return nil, errors.New(readyFrame.Message)
		}
		if readyFrame.Type != wsproto.TypeReady {
			_ = conn.Close()
			return nil, fmt.Errorf("expected websocket tunnel ready frame, got %q", readyFrame.Type)
		}
		if readyFrame.Version != version {
			_ = conn.Close()
			return nil, fmt.Errorf(
				"portr websocket protocol mismatch: client speaks v%d, server sent v%d; upgrade the portr server", version, readyFrame.Version)
		}
	case err := <-readyErr:
		_ = conn.Close()
		return nil, err
	case <-timer.C:
		_ = conn.Close()
		return nil, fmt.Errorf("timed out waiting for websocket tunnel registration")
	case <-ctx.Done():
		_ = conn.Close()
		return nil, ctx.Err()
	}

	if version == wsproto.LegacyProtocolVersion {
		return &Session{config: cfg, conn: conn, legacy: newLegacySession(conn), remotePort: readyFrame.Port}, nil
	}

	traffic := &trafficConn{Conn: wsproto.MuxConn(conn)}
	muxErrs := &muxErrors{}
	mux, err := yamux.Client(traffic, wsproto.MuxConfig(muxErrs))
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &Session{
		config:     cfg,
		conn:       conn,
		mux:        mux,
		traffic:    traffic,
		errors:     muxErrs,
		remotePort: readyFrame.Port,
	}, nil
}

func (s *Session) Accept() (net.Conn, error) {
	if s.legacy != nil {
		return s.legacy.Accept()
	}
	stream, err := s.mux.AcceptStream()
	if err != nil {
		if cause := s.errors.String(); cause != "" {
			return nil, fmt.Errorf("%w: %s", err, cause)
		}
		return nil, err
	}
	return streamConn{stream}, nil
}

func (s *Session) RemotePort() int {
	return s.remotePort
}

func (s *Session) HealthCheck(timeout time.Duration) error {
	if s.legacy != nil {
		return s.legacy.HealthCheck(timeout)
	}
	pinged := make(chan error, 1)
	go func() {
		_, err := s.mux.Ping()
		pinged <- err
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case err := <-pinged:
		return err
	case <-timer.C:
		if s.traffic.readSince(timeout) {
			return nil
		}
		_ = s.Close()
		return fmt.Errorf("websocket tunnel ping timed out")
	}
}

func (s *Session) Close() error {
	if s.legacy != nil {
		return s.legacy.Close()
	}
	s.closeOnce.Do(func() {
		// Abort outstanding I/O before Close attempts to write its close frame.
		_ = s.conn.SetDeadline(time.Now())
		_ = s.mux.Close()
	})
	return nil
}

// streamConn exposes yamux's half-close as CloseWrite, which tcp tunnels use to
// forward EOF in one direction while the other keeps flowing.
type streamConn struct {
	*yamux.Stream
}

func (c streamConn) CloseWrite() error {
	return c.Stream.Close()
}

// WebSocketURL resolves the websocket connect endpoint for the given client
// config. Exposed for diagnostics such as `portr doctor`.
func WebSocketURL(cfg config.ClientConfig) (string, error) {
	return tunnelWebSocketURL(cfg)
}

func tunnelWebSocketURL(cfg config.ClientConfig) (string, error) {
	raw := strings.TrimRight(cfg.WsUrl, "/")
	if raw == "" {
		raw = strings.TrimRight(cfg.TunnelUrl, "/")
	}
	if raw == "" {
		raw = strings.TrimRight(cfg.ServerUrl, "/")
	}
	if !strings.Contains(raw, "://") {
		scheme := "wss"
		if cfg.UseLocalHost || strings.HasPrefix(raw, "localhost:") || strings.HasPrefix(raw, "127.0.0.1:") {
			scheme = "ws"
		}
		raw = scheme + "://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("unsupported websocket tunnel scheme: %s", u.Scheme)
	}
	u.Path = wsproto.ConnectPath
	u.RawQuery = ""
	return u.String(), nil
}
