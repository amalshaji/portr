package ssh

import (
	"errors"
	"net"
	"os"
	"sync"
	"time"
)

const deadlineConnReadBufferSize = 32 * 1024

type deadlineReadResult struct {
	buffer []byte
	read   int
	err    error
}

var deadlineConnReadBufferPool = sync.Pool{
	New: func() any {
		return make([]byte, deadlineConnReadBufferSize)
	},
}

type deadlineConn struct {
	net.Conn

	results chan deadlineReadResult
	done    chan struct{}

	readMu     sync.Mutex
	pending    []byte
	pendingBuf []byte
	pendingErr error

	deadlineMu      sync.Mutex
	readDeadline    time.Time
	deadlineChanged chan struct{}

	closeOnce sync.Once
	closeErr  error
}

func newDeadlineConn(conn net.Conn) *deadlineConn {
	wrapped := &deadlineConn{
		Conn:            conn,
		results:         make(chan deadlineReadResult),
		done:            make(chan struct{}),
		deadlineChanged: make(chan struct{}),
	}
	go wrapped.readPump()
	return wrapped
}

func (c *deadlineConn) readPump() {
	for {
		buffer := deadlineConnReadBufferPool.Get().([]byte)
		read, err := c.Conn.Read(buffer)
		if read == 0 {
			deadlineConnReadBufferPool.Put(buffer)
		}
		if read == 0 && err == nil {
			continue
		}
		result := deadlineReadResult{buffer: buffer, read: read, err: err}
		select {
		case c.results <- result:
		case <-c.done:
			if read > 0 {
				deadlineConnReadBufferPool.Put(buffer)
			}
			return
		}
		if err != nil {
			return
		}
	}
}

func (c *deadlineConn) Read(payload []byte) (int, error) {
	if len(payload) == 0 {
		return 0, nil
	}

	c.readMu.Lock()
	defer c.readMu.Unlock()
	for {
		if len(c.pending) > 0 {
			read := copy(payload, c.pending)
			c.pending = c.pending[read:]
			if len(c.pending) == 0 {
				deadlineConnReadBufferPool.Put(c.pendingBuf)
				c.pendingBuf = nil
			}
			return read, nil
		}
		if c.pendingErr != nil {
			err := c.pendingErr
			c.pendingErr = nil
			return 0, err
		}

		result, err := c.nextReadResult()
		if err != nil {
			return 0, err
		}
		if result.read > 0 {
			c.pendingBuf = result.buffer
			c.pending = result.buffer[:result.read]
		}
		c.pendingErr = result.err
	}
}

func (c *deadlineConn) nextReadResult() (deadlineReadResult, error) {
	for {
		c.deadlineMu.Lock()
		deadline := c.readDeadline
		changed := c.deadlineChanged
		c.deadlineMu.Unlock()

		if deadline.IsZero() {
			select {
			case result := <-c.results:
				return result, nil
			case <-changed:
				continue
			case <-c.done:
				return deadlineReadResult{}, net.ErrClosed
			}
		}

		delay := time.Until(deadline)
		if delay <= 0 {
			return deadlineReadResult{}, os.ErrDeadlineExceeded
		}
		timer := time.NewTimer(delay)
		select {
		case result := <-c.results:
			stopTimer(timer)
			return result, nil
		case <-changed:
			stopTimer(timer)
			continue
		case <-c.done:
			stopTimer(timer)
			return deadlineReadResult{}, net.ErrClosed
		case <-timer.C:
			return deadlineReadResult{}, os.ErrDeadlineExceeded
		}
	}
}

func stopTimer(timer *time.Timer) {
	if timer.Stop() {
		return
	}
	select {
	case <-timer.C:
	default:
	}
}

func (c *deadlineConn) SetDeadline(deadline time.Time) error {
	if err := c.SetReadDeadline(deadline); err != nil {
		return err
	}
	return c.SetWriteDeadline(deadline)
}

func (c *deadlineConn) SetReadDeadline(deadline time.Time) error {
	c.deadlineMu.Lock()
	c.readDeadline = deadline
	close(c.deadlineChanged)
	c.deadlineChanged = make(chan struct{})
	c.deadlineMu.Unlock()
	return nil
}

func (c *deadlineConn) SetWriteDeadline(time.Time) error {
	return nil
}

func (c *deadlineConn) Close() error {
	c.closeOnce.Do(func() {
		close(c.done)
		c.closeErr = c.Conn.Close()
		if errors.Is(c.closeErr, net.ErrClosed) {
			c.closeErr = nil
		}
	})
	return c.closeErr
}
