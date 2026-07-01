package ssh

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	maxCapturedBodyBytes = 1 << 20
	captureQueueSize     = 32
)

type requestLogContextKey struct{}

type requestLogData struct {
	id        string
	request   *http.Request
	body      *bodyCapture
	startTime time.Time
}

type bodyCapture struct {
	mu        sync.Mutex
	data      bytes.Buffer
	total     int64
	truncated bool
}

func (c *bodyCapture) Write(payload []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.total += int64(len(payload))
	remaining := maxCapturedBodyBytes - c.data.Len()
	if remaining <= 0 {
		c.truncated = true
		return
	}
	if len(payload) > remaining {
		payload = payload[:remaining]
		c.truncated = true
	}
	_, _ = c.data.Write(payload)
}

func (c *bodyCapture) Size() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.total
}

func (c *bodyCapture) Bytes() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return bytes.Clone(c.data.Bytes())
}

type capturingReadCloser struct {
	io.ReadCloser
	capture *bodyCapture
	onDone  func()
	once    sync.Once
}

func (r *capturingReadCloser) Read(payload []byte) (int, error) {
	n, err := r.ReadCloser.Read(payload)
	if n > 0 {
		r.capture.Write(payload[:n])
	}
	if err != nil {
		r.once.Do(r.onDone)
	}
	return n, err
}

func (r *capturingReadCloser) Close() error {
	err := r.ReadCloser.Close()
	r.once.Do(r.onDone)
	return err
}

type captureTask interface {
	persist(*SshClient)
}

type httpCaptureTask struct {
	id           string
	request      *http.Request
	requestBody  []byte
	response     *http.Response
	responseBody []byte
	durationMs   int64
	bytesIn      int64
	bytesOut     int64
}

func (t httpCaptureTask) persist(client *SshClient) {
	client.logHttpRequestSized(
		t.id,
		t.request,
		t.requestBody,
		t.response,
		t.responseBody,
		t.durationMs,
		t.bytesIn,
		t.bytesOut,
	)
}

type websocketEventCaptureTask struct {
	sessionID string
	direction string
	frame     *webSocketFrame
}

type websocketOpenCaptureTask struct {
	handshake httpCaptureTask
	sessionID string
	request   *http.Request
	response  *http.Response
}

func (t websocketOpenCaptureTask) persist(client *SshClient) {
	t.handshake.persist(client)
	client.logWebSocketSessionWithID(t.sessionID, t.handshake.id, t.request, t.response)
}

func (t websocketEventCaptureTask) persist(client *SshClient) {
	client.recordWebSocketEvent(t.sessionID, t.direction, t.frame)
}

type websocketCloseCaptureTask struct {
	sessionID string
	err       error
}

func (t websocketCloseCaptureTask) persist(client *SshClient) {
	client.closeWebSocketSession(t.sessionID, t.err)
}

type captureRecorder struct {
	queue  chan queuedCaptureTask
	done   chan struct{}
	mu     sync.Mutex
	closed bool
}

func newCaptureRecorder() *captureRecorder {
	recorder := &captureRecorder{
		queue: make(chan queuedCaptureTask, captureQueueSize),
		done:  make(chan struct{}),
	}
	go func() {
		defer close(recorder.done)
		for queued := range recorder.queue {
			func() {
				defer func() {
					if recovered := recover(); recovered != nil && queued.client.config.Debug {
						queued.client.logDebug("Traffic log persistence panic", nil)
					}
				}()
				queued.task.persist(queued.client)
			}()
		}
	}()
	return recorder
}

func (r *captureRecorder) submit(client *SshClient, task captureTask) bool {
	if r == nil || task == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return false
	}
	select {
	case r.queue <- queuedCaptureTask{client: client, task: task}:
		return true
	default:
		return false
	}
}

func (r *captureRecorder) submitContext(ctx context.Context, client *SshClient, task captureTask) bool {
	if r == nil || task == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return false
	}
	select {
	case r.queue <- queuedCaptureTask{client: client, task: task}:
		return true
	case <-ctx.Done():
		return false
	}
}

func (r *captureRecorder) close(ctx context.Context) error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	if !r.closed {
		r.closed = true
		close(r.queue)
	}
	r.mu.Unlock()
	select {
	case <-r.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type queuedCaptureTask struct {
	client *SshClient
	task   captureTask
}

func (s *SshClient) submitCapture(task captureTask) bool {
	s.mu.Lock()
	if s.recorder == nil {
		s.recorder = newCaptureRecorder()
	}
	recorder := s.recorder
	s.mu.Unlock()
	accepted := recorder.submit(s, task)
	if !accepted && s.config.Debug {
		s.logDebug("Traffic log queue full or closed; dropping capture", nil)
	}
	return accepted
}

func (s *SshClient) submitCaptureContext(ctx context.Context, task captureTask) bool {
	s.mu.Lock()
	if s.recorder == nil {
		s.recorder = newCaptureRecorder()
	}
	recorder := s.recorder
	s.mu.Unlock()
	accepted := recorder.submitContext(ctx, s, task)
	if !accepted && s.config.Debug {
		s.logDebug("Traffic log queue full or closed; dropping capture", nil)
	}
	return accepted
}
