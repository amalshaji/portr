package wstunnel

import (
	"errors"
	"io"
	"net"

	"github.com/amalshaji/portr/internal/tunnel/wsproto"
	"github.com/oklog/ulid/v2"
	"golang.org/x/net/websocket"
)

// Protocol v1 multiplexes streams as JSON frames with per-stream credit
// windows. It stays so clients released before yamux keep working; remove it
// once those clients are no longer supported.

type streamQueue struct {
	frames  chan wsproto.Frame
	credits chan struct{}
	closed  chan struct{}
}

// serveLegacyFrames reads v1 frames until the websocket closes.
func (s *session) serveLegacyFrames(conn *websocket.Conn) {
	for {
		frame, err := wsproto.Receive(conn)
		if err != nil {
			s.setEndReason("read tunnel connection: " + err.Error())
			return
		}
		if frame.Type == wsproto.TypePing {
			_ = s.writer.Send(wsproto.Frame{Type: wsproto.TypePong})
			continue
		}
		s.deliver(frame)
	}
}

func (m *Manager) pipeLegacyStream(sess *session, conn net.Conn, initial []byte) {
	defer conn.Close()

	streamID := ulid.Make().String()
	stream := sess.addStream(streamID)

	closed := make(chan struct{})
	go func() {
		defer close(closed)
		select {
		case <-sess.closed:
		case <-stream.closed:
		}
		_ = conn.Close()
	}()
	defer func() {
		sess.removeStream(streamID)
		<-closed
	}()

	if err := sess.writer.Send(wsproto.Frame{Type: wsproto.TypeOpen, StreamID: streamID, Data: initial}); err != nil {
		return
	}
	if err := sess.writer.Send(wsproto.Frame{
		Type:     wsproto.TypeWindow,
		StreamID: streamID,
		Window:   wsproto.StreamWindowSize,
	}); err != nil {
		return
	}

	readDone := make(chan error, 1)
	if !sess.goWorker(func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				if !stream.takeCredit(sess.closed) {
					return
				}
				if sendErr := sess.writer.Send(wsproto.Frame{
					Type:     wsproto.TypeData,
					StreamID: streamID,
					Data:     append([]byte(nil), buf[:n]...),
				}); sendErr != nil {
					readDone <- sendErr
					return
				}
			}
			if err != nil {
				readDone <- err
				return
			}
		}
	}) {
		return
	}

	readClosed := false
	writeClosed := false
	for {
		select {
		case frame := <-stream.frames:
			switch frame.Type {
			case wsproto.TypeData:
				if _, err := conn.Write(frame.Data); err != nil {
					return
				}
				if err := sess.writer.Send(wsproto.Frame{
					Type:     wsproto.TypeWindow,
					StreamID: streamID,
					Window:   1,
				}); err != nil {
					return
				}
			case wsproto.TypeCloseWrite:
				closeWriter, ok := conn.(interface{ CloseWrite() error })
				if !ok || closeWriter.CloseWrite() != nil {
					return
				}
				writeClosed = true
				if readClosed {
					return
				}
			case wsproto.TypeClose, wsproto.TypeError:
				return
			}
		case err := <-readDone:
			if err != nil && !errors.Is(err, io.EOF) {
				_ = sess.writer.Send(wsproto.Frame{Type: wsproto.TypeError, StreamID: streamID, Message: err.Error()})
				return
			}
			if err := sess.writer.Send(wsproto.Frame{Type: wsproto.TypeCloseWrite, StreamID: streamID}); err != nil {
				return
			}
			readClosed = true
			readDone = nil
			if writeClosed {
				return
			}
		case <-sess.closed:
			return
		case <-stream.closed:
			return
		}
	}
}

func (s *session) addStream(streamID string) *streamQueue {
	stream := &streamQueue{
		frames:  make(chan wsproto.Frame, wsproto.StreamWindowSize+1),
		credits: make(chan struct{}, wsproto.StreamWindowSize),
		closed:  make(chan struct{}),
	}
	s.streamMu.Lock()
	s.streams[streamID] = stream
	s.streamMu.Unlock()
	return stream
}

func (s *session) removeStream(streamID string) {
	s.streamMu.Lock()
	stream := s.streams[streamID]
	if stream != nil {
		delete(s.streams, streamID)
		close(stream.closed)
	}
	s.streamMu.Unlock()
}

func (s *session) deliver(frame wsproto.Frame) {
	if frame.StreamID == "" {
		return
	}
	s.streamMu.Lock()
	stream := s.streams[frame.StreamID]
	s.streamMu.Unlock()
	if stream == nil {
		return
	}
	if frame.Type == wsproto.TypeWindow {
		stream.grant(frame.Window)
		return
	}
	select {
	case stream.frames <- frame:
	case <-stream.closed:
	case <-s.closed:
	default:
		s.removeStream(frame.StreamID)
	}
}

func (s *streamQueue) grant(credits int) {
	for range min(credits, wsproto.StreamWindowSize) {
		select {
		case s.credits <- struct{}{}:
		default:
			return
		}
	}
}

func (s *streamQueue) takeCredit(sessionClosed <-chan struct{}) bool {
	select {
	case <-s.credits:
		return true
	case <-s.closed:
		return false
	case <-sessionClosed:
		return false
	}
}
