package wstunnel

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

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
	config       *config.Config
	service      *service.Service
	mu           sync.RWMutex
	bySub        map[string][]*session
	sessions     map[string]*session
	rrIdx        map[string]int
	workers      sync.WaitGroup
	shutdownOnce sync.Once
	shutdownDone chan struct{}
	shuttingDown bool
}

type session struct {
	id         string
	connection *db.Connection
	conn       io.Closer
	writer     *wsproto.Writer
	streams    map[string]*streamQueue
	streamMu   sync.Mutex
	workers    sync.WaitGroup
	workerMu   sync.Mutex
	closing    bool
	listener   net.Listener
	closed     chan struct{}
	closeOnce  sync.Once
}

type streamQueue struct {
	frames chan wsproto.Frame
	closed chan struct{}
}

func New(config *config.Config, service *service.Service) *Manager {
	return &Manager{
		config:       config,
		service:      service,
		bySub:        make(map[string][]*session),
		sessions:     make(map[string]*session),
		rrIdx:        make(map[string]int),
		shutdownDone: make(chan struct{}),
	}
}

func (m *Manager) Handler() http.Handler {
	return websocket.Handler(func(conn *websocket.Conn) {
		m.mu.Lock()
		if m.shuttingDown {
			m.mu.Unlock()
			_ = conn.Close()
			return
		}
		m.workers.Add(1)
		m.mu.Unlock()
		defer m.workers.Done()
		m.handle(conn)
	})
}

func (*Manager) Endpoint() string {
	return wsproto.ConnectPath
}

func (m *Manager) ServeHTTPStream(subdomain string, writer http.ResponseWriter, request *http.Request) {
	conn, initial, err := hijackRequest(writer, request)
	if err != nil {
		log.Error("Failed to hijack proxied request", "error", err, "subdomain", subdomain)
		http.Error(writer, "failed to proxy request", http.StatusBadGateway)
		return
	}
	m.OpenHTTPStream(subdomain, conn, initial)
}

func (m *Manager) Start() {
	<-m.shutdownDone
}

func (m *Manager) Shutdown(ctx context.Context) error {
	m.shutdownOnce.Do(func() {
		m.mu.Lock()
		m.shuttingDown = true
		sessions := make([]*session, 0, len(m.sessions))
		for _, sess := range m.sessions {
			sessions = append(sessions, sess)
		}
		m.mu.Unlock()

		for _, sess := range sessions {
			m.unregisterSession(sess)
		}
		go func() {
			for _, sess := range sessions {
				sess.workers.Wait()
			}
			m.workers.Wait()
			close(m.shutdownDone)
		}()
	})

	select {
	case <-m.shutdownDone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
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

	defer sess.workers.Done()
	m.pipeStream(sess, clientConn, initial)
}

func (m *Manager) handle(conn *websocket.Conn) {
	req := conn.Request()
	connectionID := req.Header.Get(wsproto.ConnectionIDHeader)
	secretKey := req.Header.Get(wsproto.SecretKeyHeader)
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
		conn:       conn,
		writer:     wsproto.NewWriter(conn),
		streams:    make(map[string]*streamQueue),
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
		if m.shuttingDown {
			m.mu.Unlock()
			return fmt.Errorf("websocket tunnel manager is shutting down")
		}
		m.bySub[*conn.Subdomain] = append(m.bySub[*conn.Subdomain], sess)
		m.sessions[sess.id] = sess
		if err := m.service.MarkConnectionAsActive(ctx, conn.ID); err != nil {
			m.mu.Unlock()
			m.unregisterSession(sess)
			return err
		}
		m.mu.Unlock()
	case string(constants.Tcp):
		port, listener, err := m.listenTCP()
		if err != nil {
			return err
		}
		port32 := uint32(port)
		m.mu.Lock()
		if m.shuttingDown {
			m.mu.Unlock()
			_ = listener.Close()
			return fmt.Errorf("websocket tunnel manager is shutting down")
		}
		if err := m.service.MarkTCPConnectionAsActive(ctx, conn.ID, port32); err != nil {
			m.mu.Unlock()
			_ = listener.Close()
			return err
		}
		conn.Port = &port32
		sess.listener = listener
		m.sessions[sess.id] = sess
		m.mu.Unlock()
		sess.goWorker(func() { m.acceptTCP(sess, listener) })
	default:
		return fmt.Errorf("unsupported connection type: %s", conn.Type)
	}

	return nil
}

func (m *Manager) unregisterSession(sess *session) {
	sess.closeOnce.Do(func() {
		sess.workerMu.Lock()
		sess.closing = true
		sess.workerMu.Unlock()
		close(sess.closed)
		if sess.conn != nil {
			_ = sess.conn.Close()
		}
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
			closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = m.service.MarkConnectionAsClosed(closeCtx, sess.connection.ID)
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
	if !sess.acquireWorker() {
		return nil, fmt.Errorf("route not found")
	}
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
		if !sess.goWorker(func() { m.pipeStream(sess, conn, nil) }) {
			_ = conn.Close()
		}
	}
}

func (s *session) acquireWorker() bool {
	s.workerMu.Lock()
	defer s.workerMu.Unlock()
	if s.closing {
		return false
	}
	s.workers.Add(1)
	return true
}

func (s *session) goWorker(worker func()) bool {
	if !s.acquireWorker() {
		return false
	}
	go func() {
		defer s.workers.Done()
		worker()
	}()
	return true
}

func (m *Manager) pipeStream(sess *session, conn net.Conn, initial []byte) {
	defer conn.Close()

	streamID := ulid.Make().String()
	stream := sess.addStream(streamID)
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
		case frame := <-stream.frames:
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

func (s *session) addStream(streamID string) *streamQueue {
	stream := &streamQueue{
		frames: make(chan wsproto.Frame, 32),
		closed: make(chan struct{}),
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
	select {
	case stream.frames <- frame:
	case <-stream.closed:
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

func hijackRequest(w http.ResponseWriter, r *http.Request) (net.Conn, []byte, error) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not support hijacking")
	}

	var initial bytes.Buffer
	if err := writeRequestHead(&initial, r); err != nil {
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

func writeRequestHead(dst io.Writer, request *http.Request) error {
	requestURI := request.URL.RequestURI()
	if requestURI == "" {
		requestURI = "/"
	}
	proto := request.Proto
	if proto == "" {
		proto = "HTTP/1.1"
	}
	if _, err := fmt.Fprintf(dst, "%s %s %s\r\n", request.Method, requestURI, proto); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(dst, "Host: %s\r\n", request.Host); err != nil {
		return err
	}

	headers := request.Header.Clone()
	headers.Del("Host")
	headers.Del("Content-Length")
	headers.Del("Transfer-Encoding")
	if request.ContentLength > 0 {
		headers.Set("Content-Length", fmt.Sprint(request.ContentLength))
	}
	if len(request.TransferEncoding) > 0 {
		headers.Set("Transfer-Encoding", strings.Join(request.TransferEncoding, ", "))
	}
	if request.Close || headers.Get("Upgrade") == "" {
		headers.Set("Connection", "close")
	}
	if len(request.Trailer) > 0 && headers.Get("Trailer") == "" {
		trailerNames := make([]string, 0, len(request.Trailer))
		for name := range request.Trailer {
			trailerNames = append(trailerNames, name)
		}
		slices.Sort(trailerNames)
		headers.Set("Trailer", strings.Join(trailerNames, ", "))
	}
	if err := headers.Write(dst); err != nil {
		return err
	}
	_, err := io.WriteString(dst, "\r\n")
	return err
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
