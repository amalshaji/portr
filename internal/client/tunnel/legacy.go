package tunnel

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/amalshaji/portr/internal/tunnel/wsproto"
	"golang.org/x/net/websocket"
)

// legacySession speaks protocol v1, which multiplexes streams as JSON frames
// with per-stream credit windows. It is used only for servers released before
// yamux; remove it once those servers are no longer supported.
type legacySession struct {
	conn      *websocket.Conn
	writer    *wsproto.Writer
	accept    chan net.Conn
	pong      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
	streams   map[string]*tunnelStream
	streamsMu sync.Mutex
	errMu     sync.Mutex
	readErr   error
	healthMu  sync.Mutex
}

type tunnelStream struct {
	frames    chan wsproto.Frame
	credits   chan struct{}
	done      chan struct{}
	closeOnce sync.Once
}

func newLegacySession(conn *websocket.Conn) *legacySession {
	session := &legacySession{
		conn:    conn,
		writer:  wsproto.NewWriter(conn),
		accept:  make(chan net.Conn, wsproto.StreamWindowSize),
		pong:    make(chan struct{}, 1),
		done:    make(chan struct{}),
		streams: make(map[string]*tunnelStream),
	}
	go session.readFrames()
	return session
}

func (s *legacySession) Accept() (net.Conn, error) {
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

func (s *legacySession) HealthCheck(timeout time.Duration) error {
	s.healthMu.Lock()
	defer s.healthMu.Unlock()
	select {
	case <-s.pong:
	default:
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	sent := make(chan error, 1)
	go func() {
		sent <- s.writer.Send(wsproto.Frame{Type: wsproto.TypePing})
	}()
	for {
		select {
		case err := <-sent:
			if err != nil {
				return err
			}
		case <-s.pong:
			return nil
		case <-s.done:
			return net.ErrClosed
		case <-timer.C:
			_ = s.Close()
			return fmt.Errorf("websocket tunnel ping timed out")
		}
	}
}

func (s *legacySession) Close() error {
	s.closeOnce.Do(func() {
		close(s.done)
		// Abort outstanding I/O before Close attempts to write its close frame.
		_ = s.conn.SetDeadline(time.Now())
		_ = s.conn.Close()
		s.closeStreams()
	})
	return nil
}

func (s *legacySession) readFrames() {
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

func (s *legacySession) openStream(frame wsproto.Frame) {
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

func (s *legacySession) send(frame wsproto.Frame) error {
	select {
	case <-s.done:
		return net.ErrClosed
	default:
		return s.writer.Send(frame)
	}
}

func (s *legacySession) deliver(frame wsproto.Frame) {
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

func (s *legacySession) removeStream(streamID string) {
	s.streamsMu.Lock()
	stream := s.streams[streamID]
	delete(s.streams, streamID)
	s.streamsMu.Unlock()
	if stream != nil {
		stream.closeOnce.Do(func() { close(stream.done) })
	}
}

func (s *legacySession) closeStreams() {
	s.streamsMu.Lock()
	streams := s.streams
	s.streams = make(map[string]*tunnelStream)
	s.streamsMu.Unlock()
	for _, stream := range streams {
		stream.closeOnce.Do(func() { close(stream.done) })
	}
}
