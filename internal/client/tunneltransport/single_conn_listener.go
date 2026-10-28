package tunneltransport

import (
	"net"
	"sync"
	"time"
)

type singleConnListener struct {
	conn      net.Conn
	accepted  bool
	closed    bool
	done      chan struct{}
	closeOnce sync.Once
	mu        sync.Mutex
}

func (l *singleConnListener) Accept() (net.Conn, error) {
	l.mu.Lock()
	if l.closed || l.accepted {
		done := l.doneChannelLocked()
		l.mu.Unlock()
		<-done
		return nil, net.ErrClosed
	}
	l.accepted = true
	l.doneChannelLocked()
	conn := &listenerConn{Conn: withDeadlines(l.conn), onClose: l.Close}
	l.mu.Unlock()
	return conn, nil
}

func (l *singleConnListener) Close() error {
	l.mu.Lock()
	l.closed = true
	done := l.doneChannelLocked()
	l.mu.Unlock()
	l.closeOnce.Do(func() { close(done) })
	return nil
}

func (l *singleConnListener) Addr() net.Addr { return l.conn.LocalAddr() }

func (l *singleConnListener) doneChannelLocked() chan struct{} {
	if l.done == nil {
		l.done = make(chan struct{})
	}
	return l.done
}

type listenerConn struct {
	net.Conn
	onClose func() error
	once    sync.Once
}

func (c *listenerConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() { _ = c.onClose() })
	return err
}

// withDeadlines returns conn as is when it supports read deadlines, which
// net/http relies on, and wraps it otherwise. yamux streams support them; SSH
// channels do not.
func withDeadlines(conn net.Conn) net.Conn {
	if conn.SetReadDeadline(time.Time{}) == nil {
		return conn
	}
	return newDeadlineConn(conn)
}
