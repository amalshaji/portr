package wstunnel

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/amalshaji/portr/internal/constants"
	"github.com/amalshaji/portr/internal/server/config"
	"github.com/amalshaji/portr/internal/server/db"
	"github.com/amalshaji/portr/internal/server/service"
	"github.com/amalshaji/portr/internal/tunnel/wsproto"
	"github.com/amalshaji/portr/internal/utils"
	"github.com/charmbracelet/log"
	"github.com/oklog/ulid/v2"
	"golang.org/x/net/websocket"
)

type Manager struct {
	config   *config.Config
	service  *service.Service
	mu       sync.RWMutex
	bySub    map[string][]*session
	sessions map[string]*session
	rrIdx    map[string]int
}

type session struct {
	id         string
	connection *db.Connection
	writer     *wsproto.Writer
	streams    map[string]chan wsproto.Frame
	streamMu   sync.Mutex
	listener   net.Listener
	closed     chan struct{}
	closeOnce  sync.Once
}

func New(config *config.Config, service *service.Service) *Manager {
	return &Manager{
		config:   config,
		service:  service,
		bySub:    make(map[string][]*session),
		sessions: make(map[string]*session),
		rrIdx:    make(map[string]int),
	}
}

func (m *Manager) Handler() websocket.Handler {
	return websocket.Handler(func(conn *websocket.Conn) {
		m.handle(conn)
	})
}

func (m *Manager) HasHTTPBackend(subdomain string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.bySub[subdomain]) > 0
}

func (m *Manager) HTTPBackendCount(subdomain string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.bySub[subdomain])
}

func (m *Manager) OpenHTTPStream(subdomain string, clientConn net.Conn, initial []byte) {
	sess, err := m.nextHTTPSession(subdomain)
	if err != nil {
		unregisteredSubdomainError(clientConn, subdomain)
		_ = clientConn.Close()
		return
	}

	m.pipeStream(sess, clientConn, initial)
}

func (m *Manager) handle(conn *websocket.Conn) {
	req := conn.Request()
	connectionID := req.URL.Query().Get("connection_id")
	secretKey := req.URL.Query().Get("secret_key")
	if connectionID == "" || secretKey == "" {
		_ = wsproto.NewWriter(conn).Send(wsproto.Frame{Type: wsproto.TypeError, Message: "missing connection credentials"})
		_ = conn.Close()
		return
	}

	reserved, err := m.service.GetReservedConnectionById(req.Context(), connectionID)
	if err != nil || reserved.CreatedBy.SecretKey != secretKey {
		_ = wsproto.NewWriter(conn).Send(wsproto.Frame{Type: wsproto.TypeError, Message: "invalid connection credentials"})
		_ = conn.Close()
		return
	}

	sess := &session{
		id:         ulid.Make().String(),
		connection: reserved,
		writer:     wsproto.NewWriter(conn),
		streams:    make(map[string]chan wsproto.Frame),
		closed:     make(chan struct{}),
	}

	if err := m.registerSession(req.Context(), sess); err != nil {
		_ = sess.writer.Send(wsproto.Frame{Type: wsproto.TypeError, Message: err.Error()})
		_ = conn.Close()
		return
	}
	defer m.unregisterSession(sess)

	ready := wsproto.Frame{Type: wsproto.TypeReady}
	if reserved.Port != nil {
		ready.Port = int(*reserved.Port)
	}
	if err := sess.writer.Send(ready); err != nil {
		return
	}

	for {
		frame, err := wsproto.Receive(conn)
		if err != nil {
			return
		}
		if frame.Type == wsproto.TypePing {
			_ = sess.writer.Send(wsproto.Frame{Type: wsproto.TypePong})
			continue
		}
		sess.deliver(frame)
	}
}

func (m *Manager) registerSession(ctx context.Context, sess *session) error {
	conn := sess.connection
	switch conn.Type {
	case string(constants.Http):
		if conn.Subdomain == nil || *conn.Subdomain == "" {
			return fmt.Errorf("http connection is missing a subdomain")
		}
		m.mu.Lock()
		m.bySub[*conn.Subdomain] = append(m.bySub[*conn.Subdomain], sess)
		m.sessions[sess.id] = sess
		m.mu.Unlock()
	case string(constants.Tcp):
		port, listener, err := m.listenTCP()
		if err != nil {
			return err
		}
		port32 := uint32(port)
		if err := m.service.AddPortToConnection(ctx, conn.ID, port32); err != nil {
			_ = listener.Close()
			return err
		}
		conn.Port = &port32
		sess.listener = listener
		m.mu.Lock()
		m.sessions[sess.id] = sess
		m.mu.Unlock()
		go m.acceptTCP(sess, listener)
	default:
		return fmt.Errorf("unsupported connection type: %s", conn.Type)
	}

	if err := m.service.MarkConnectionAsActive(ctx, conn.ID); err != nil {
		m.unregisterSession(sess)
		return err
	}
	return nil
}

func (m *Manager) unregisterSession(sess *session) {
	sess.closeOnce.Do(func() {
		close(sess.closed)
		if sess.listener != nil {
			_ = sess.listener.Close()
		}

		m.mu.Lock()
		shouldMarkClosed := sess.connection.Type != string(constants.Http)
		delete(m.sessions, sess.id)
		if sess.connection.Type == string(constants.Http) && sess.connection.Subdomain != nil {
			subdomain := *sess.connection.Subdomain
			list := m.bySub[subdomain]
			for idx, candidate := range list {
				if candidate == sess {
					list = append(list[:idx], list[idx+1:]...)
					break
				}
			}
			if len(list) == 0 {
				delete(m.bySub, subdomain)
				delete(m.rrIdx, subdomain)
				shouldMarkClosed = true
			} else {
				m.bySub[subdomain] = list
				if m.rrIdx[subdomain] >= len(list) {
					m.rrIdx[subdomain] = 0
				}
			}
		}
		m.mu.Unlock()

		if shouldMarkClosed {
			_ = m.service.MarkConnectionAsClosed(context.Background(), sess.connection.ID)
		}
	})
}

func (m *Manager) nextHTTPSession(subdomain string) (*session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	list := m.bySub[subdomain]
	if len(list) == 0 {
		return nil, fmt.Errorf("route not found")
	}

	idx := m.rrIdx[subdomain]
	sess := list[idx]
	m.rrIdx[subdomain] = (idx + 1) % len(list)
	return sess, nil
}

func (m *Manager) listenTCP() (int, net.Listener, error) {
	var lastErr error
	for _, port := range utils.GenerateRandomTcpPorts() {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			return port, listener, nil
		}
		lastErr = err
	}
	return 0, nil, fmt.Errorf("failed to allocate tcp tunnel port: %w", lastErr)
}

func (m *Manager) acceptTCP(sess *session, listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-sess.closed:
				return
			default:
				log.Error("Failed to accept tcp tunnel connection", "connection_id", sess.connection.ID, "error", err)
				return
			}
		}
		go m.pipeStream(sess, conn, nil)
	}
}

func (m *Manager) pipeStream(sess *session, conn net.Conn, initial []byte) {
	defer conn.Close()

	streamID := ulid.Make().String()
	frames := sess.addStream(streamID)
	defer sess.removeStream(streamID)

	if err := sess.writer.Send(wsproto.Frame{Type: wsproto.TypeOpen, StreamID: streamID, Data: initial}); err != nil {
		return
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 32*1024)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				if sendErr := sess.writer.Send(wsproto.Frame{
					Type:     wsproto.TypeData,
					StreamID: streamID,
					Data:     append([]byte(nil), buf[:n]...),
				}); sendErr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	for {
		select {
		case frame, ok := <-frames:
			if !ok {
				_ = sess.writer.Send(wsproto.Frame{Type: wsproto.TypeClose, StreamID: streamID})
				return
			}
			switch frame.Type {
			case wsproto.TypeData:
				if _, err := conn.Write(frame.Data); err != nil {
					return
				}
			case wsproto.TypeClose, wsproto.TypeError:
				return
			}
		case <-done:
			_ = sess.writer.Send(wsproto.Frame{Type: wsproto.TypeClose, StreamID: streamID})
			return
		case <-sess.closed:
			return
		}
	}
}

func (s *session) addStream(streamID string) chan wsproto.Frame {
	ch := make(chan wsproto.Frame, 32)
	s.streamMu.Lock()
	s.streams[streamID] = ch
	s.streamMu.Unlock()
	return ch
}

func (s *session) removeStream(streamID string) {
	s.streamMu.Lock()
	ch := s.streams[streamID]
	delete(s.streams, streamID)
	s.streamMu.Unlock()
	if ch != nil {
		close(ch)
	}
}

func (s *session) deliver(frame wsproto.Frame) {
	if frame.StreamID == "" {
		return
	}
	s.streamMu.Lock()
	ch := s.streams[frame.StreamID]
	s.streamMu.Unlock()
	if ch == nil {
		return
	}
	select {
	case ch <- frame:
	case <-s.closed:
	}
}

func unregisteredSubdomainError(conn net.Conn, subdomain string) {
	body := []byte(utils.UnregisteredSubdomain(subdomain))
	fmt.Fprintf(conn, "HTTP/1.1 404 Not Found\r\n")
	fmt.Fprintf(conn, "Content-Length: %d\r\n", len(body))
	fmt.Fprintf(conn, "Content-Type: text/html\r\n")
	fmt.Fprintf(conn, "X-Portr-Error: true\r\n")
	fmt.Fprintf(conn, "X-Portr-Error-Reason: unregistered-subdomain\r\n\r\n")
	_, _ = conn.Write(body)
}

func HijackRequest(w http.ResponseWriter, r *http.Request) (net.Conn, []byte, error) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not support hijacking")
	}

	var body []byte
	if r.Body != nil {
		var err error
		body, err = io.ReadAll(r.Body)
		if err != nil {
			return nil, nil, err
		}
		_ = r.Body.Close()
		r.Body = io.NopCloser(bytes.NewReader(body))
	}

	var initial bytes.Buffer
	if err := r.Write(&initial); err != nil {
		return nil, nil, err
	}

	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, nil, err
	}

	if buffered := drainBuffered(rw.Reader); len(buffered) > 0 {
		initial.Write(buffered)
	}
	return conn, initial.Bytes(), nil
}

func drainBuffered(reader *bufio.Reader) []byte {
	if reader == nil || reader.Buffered() == 0 {
		return nil
	}
	buffered := make([]byte, reader.Buffered())
	if _, err := io.ReadFull(reader, buffered); err != nil {
		return nil
	}
	return buffered
}
