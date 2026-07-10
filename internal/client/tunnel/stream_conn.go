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
	done     <-chan struct{}
	send     func(wsproto.Frame) error

	mu      sync.Mutex
	buffer  bytes.Buffer
	closed  chan struct{}
	closeMu sync.Once
}

func newTunnelStreamConn(
	streamID string,
	initial []byte,
	frames <-chan wsproto.Frame,
	done <-chan struct{},
	send func(wsproto.Frame) error,
) *tunnelStreamConn {
	conn := &tunnelStreamConn{
		streamID: streamID,
		frames:   frames,
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
			case wsproto.TypeClose:
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
	select {
	case <-c.closed:
		return 0, net.ErrClosed
	default:
	}

	data := append([]byte(nil), p...)
	if err := c.send(wsproto.Frame{Type: wsproto.TypeData, StreamID: c.streamID, Data: data}); err != nil {
		return 0, err
	}
	return len(p), nil
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
