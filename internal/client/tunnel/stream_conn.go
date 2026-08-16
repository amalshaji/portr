package tunnel

import (
	"bytes"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/amalshaji/portr/internal/tunnel/wsproto"
)

type tunnelAddr string

func (a tunnelAddr) Network() string { return "websocket" }
func (a tunnelAddr) String() string  { return string(a) }

type tunnelStreamConn struct {
	streamID string
	frames   <-chan wsproto.Frame
	credits  <-chan struct{}
	done     <-chan struct{}
	send     func(wsproto.Frame) error

	mu       sync.Mutex
	buffer   bytes.Buffer
	readEOF  bool
	writeMu  sync.Mutex
	writeEOF bool
	closed   chan struct{}
	closeMu  sync.Once
}

func newTunnelStreamConn(
	streamID string,
	initial []byte,
	frames <-chan wsproto.Frame,
	credits <-chan struct{},
	done <-chan struct{},
	send func(wsproto.Frame) error,
) *tunnelStreamConn {
	conn := &tunnelStreamConn{
		streamID: streamID,
		frames:   frames,
		credits:  credits,
		done:     done,
		send:     send,
		closed:   make(chan struct{}),
	}
	if len(initial) > 0 {
		conn.buffer.Write(initial)
	}
	return conn
}

func (c *tunnelStreamConn) Read(p []byte) (int, error) {
	for {
		c.mu.Lock()
		if c.buffer.Len() > 0 {
			n, err := c.buffer.Read(p)
			c.mu.Unlock()
			return n, err
		}
		if c.readEOF {
			c.mu.Unlock()
			return 0, io.EOF
		}
		c.mu.Unlock()

		select {
		case <-c.closed:
			return 0, net.ErrClosed
		case <-c.done:
			return 0, net.ErrClosed
		case frame, ok := <-c.frames:
			if !ok {
				return 0, io.EOF
			}
			switch frame.Type {
			case wsproto.TypeData:
				if len(frame.Data) == 0 {
					continue
				}
				c.mu.Lock()
				c.buffer.Write(frame.Data)
				c.mu.Unlock()
				if err := c.send(wsproto.Frame{Type: wsproto.TypeWindow, StreamID: c.streamID, Window: 1}); err != nil {
					return 0, err
				}
			case wsproto.TypeClose:
				c.mu.Lock()
				c.readEOF = true
				c.mu.Unlock()
				return 0, io.EOF
			case wsproto.TypeCloseWrite:
				c.mu.Lock()
				c.readEOF = true
				c.mu.Unlock()
				return 0, io.EOF
			case wsproto.TypeError:
				if frame.Message != "" {
					return 0, errors.New(frame.Message)
				}
				return 0, io.EOF
			}
		}
	}
}

func (c *tunnelStreamConn) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.writeEOF {
		return 0, io.ErrClosedPipe
	}
	select {
	case <-c.closed:
		return 0, net.ErrClosed
	default:
	}
	select {
	case <-c.credits:
	case <-c.closed:
		return 0, net.ErrClosed
	case <-c.done:
		return 0, net.ErrClosed
	}

	data := append([]byte(nil), p...)
	if err := c.send(wsproto.Frame{Type: wsproto.TypeData, StreamID: c.streamID, Data: data}); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (c *tunnelStreamConn) CloseWrite() error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	select {
	case <-c.closed:
		return net.ErrClosed
	default:
	}
	if c.writeEOF {
		return nil
	}
	c.writeEOF = true
	return c.send(wsproto.Frame{Type: wsproto.TypeCloseWrite, StreamID: c.streamID})
}

func (c *tunnelStreamConn) Close() error {
	c.closeMu.Do(func() {
		close(c.closed)
		_ = c.send(wsproto.Frame{Type: wsproto.TypeClose, StreamID: c.streamID})
	})
	return nil
}

func (c *tunnelStreamConn) LocalAddr() net.Addr              { return tunnelAddr("portr-local") }
func (c *tunnelStreamConn) RemoteAddr() net.Addr             { return tunnelAddr(c.streamID) }
func (c *tunnelStreamConn) SetDeadline(time.Time) error      { return nil }
func (c *tunnelStreamConn) SetReadDeadline(time.Time) error  { return nil }
func (c *tunnelStreamConn) SetWriteDeadline(time.Time) error { return nil }
