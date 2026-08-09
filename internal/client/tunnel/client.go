package tunnel

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	config "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/amalshaji/portr/internal/tunnel/wsproto"
	"golang.org/x/net/websocket"
)

type Session struct {
	config     config.ClientConfig
	conn       *websocket.Conn
	writer     *wsproto.Writer
	remotePort int
	accept     chan net.Conn
	pong       chan struct{}
	done       chan struct{}
	closeOnce  sync.Once
	streams    map[string]*tunnelStream
	streamsMu  sync.Mutex
	errMu      sync.Mutex
	readErr    error
	healthMu   sync.Mutex
}

type tunnelStream struct {
	frames    chan wsproto.Frame
	credits   chan struct{}
	done      chan struct{}
	closeOnce sync.Once
}

func Connect(ctx context.Context, cfg config.ClientConfig, connectionID string) (*Session, error) {
	wsURL, err := tunnelWebSocketURL(cfg)
	if err != nil {
		return nil, err
	}
	wsConfig, err := websocket.NewConfig(wsURL, cfg.GetServerAddr())
	if err != nil {
		return nil, err
	}
	wsConfig.Header.Set(wsproto.ConnectionIDHeader, connectionID)
	wsConfig.Header.Set(wsproto.SecretKeyHeader, cfg.SecretKey)
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
			return nil, errors.New(readyFrame.Message)
		}
		if readyFrame.Type != wsproto.TypeReady {
			_ = conn.Close()
			return nil, fmt.Errorf("expected websocket tunnel ready frame, got %q", readyFrame.Type)
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

	session := &Session{
		config:     cfg,
		conn:       conn,
		writer:     wsproto.NewWriter(conn),
		remotePort: readyFrame.Port,
		accept:     make(chan net.Conn, wsproto.StreamWindowSize),
		pong:       make(chan struct{}, 1),
		done:       make(chan struct{}),
		streams:    make(map[string]*tunnelStream),
	}
	go session.readFrames()
	return session, nil
}

func (s *Session) Accept() (net.Conn, error) {
	select {
	case conn := <-s.accept:
		return conn, nil
	case <-s.done:
		s.errMu.Lock()
		err := s.readErr
		s.errMu.Unlock()
		if err == nil {
			err = net.ErrClosed
		}
		return nil, err
	}
}

func (s *Session) RemotePort() int {
	return s.remotePort
}

func (s *Session) HealthCheck(timeout time.Duration) error {
	s.healthMu.Lock()
	defer s.healthMu.Unlock()
	select {
	case <-s.pong:
	default:
	}
	if err := s.writer.Send(wsproto.Frame{Type: wsproto.TypePing}); err != nil {
		return err
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-s.pong:
		return nil
	case <-s.done:
		return net.ErrClosed
	case <-timer.C:
		return fmt.Errorf("websocket tunnel ping timed out")
	}
}

func (s *Session) Close() error {
	s.closeOnce.Do(func() {
		close(s.done)
		_ = s.conn.Close()
		s.closeStreams()
	})
	return nil
}

func (s *Session) readFrames() {
	for {
		frame, err := wsproto.Receive(s.conn)
		if err != nil {
			s.errMu.Lock()
			s.readErr = err
			s.errMu.Unlock()
			_ = s.Close()
			return
		}
		switch frame.Type {
		case wsproto.TypeOpen:
			s.openStream(frame)
		case wsproto.TypeData, wsproto.TypeWindow, wsproto.TypeCloseWrite, wsproto.TypeClose, wsproto.TypeError:
			s.deliver(frame)
		case wsproto.TypePong:
			select {
			case s.pong <- struct{}{}:
			default:
			}
		}
	}
}

func (s *Session) openStream(frame wsproto.Frame) {
	if frame.StreamID == "" {
		return
	}
	stream := &tunnelStream{
		frames:  make(chan wsproto.Frame, wsproto.StreamWindowSize+1),
		credits: make(chan struct{}, wsproto.StreamWindowSize),
		done:    make(chan struct{}),
	}
	s.streamsMu.Lock()
	s.streams[frame.StreamID] = stream
	s.streamsMu.Unlock()
	conn := newTunnelStreamConn(frame.StreamID, frame.Data, stream.frames, stream.credits, stream.done, s.send)
	go func() {
		select {
		case <-conn.closed:
		case <-s.done:
		}
		s.removeStream(frame.StreamID)
	}()
	if err := s.send(wsproto.Frame{
		Type:     wsproto.TypeWindow,
		StreamID: frame.StreamID,
		Window:   wsproto.StreamWindowSize,
	}); err != nil {
		_ = conn.Close()
		return
	}
	select {
	case s.accept <- conn:
	case <-s.done:
		_ = conn.Close()
	default:
		_ = conn.Close()
	}
}

func (s *Session) send(frame wsproto.Frame) error {
	select {
	case <-s.done:
		return net.ErrClosed
	default:
		return s.writer.Send(frame)
	}
}

func (s *Session) deliver(frame wsproto.Frame) {
	s.streamsMu.Lock()
	stream := s.streams[frame.StreamID]
	s.streamsMu.Unlock()
	if stream == nil {
		return
	}
	if frame.Type == wsproto.TypeWindow {
		for range min(frame.Window, wsproto.StreamWindowSize) {
			select {
			case stream.credits <- struct{}{}:
			default:
				return
			}
		}
		return
	}
	select {
	case stream.frames <- frame:
	case <-stream.done:
	case <-s.done:
	default:
		s.removeStream(frame.StreamID)
	}
}

func (s *Session) removeStream(streamID string) {
	s.streamsMu.Lock()
	stream := s.streams[streamID]
	delete(s.streams, streamID)
	s.streamsMu.Unlock()
	if stream != nil {
		stream.closeOnce.Do(func() { close(stream.done) })
	}
}

func (s *Session) closeStreams() {
	s.streamsMu.Lock()
	streams := s.streams
	s.streams = make(map[string]*tunnelStream)
	s.streamsMu.Unlock()
	for _, stream := range streams {
		stream.closeOnce.Do(func() { close(stream.done) })
	}
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
