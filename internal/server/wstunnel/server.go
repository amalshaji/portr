package wstunnel

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/amalshaji/portr/internal/constants"
	"github.com/amalshaji/portr/internal/server/config"
	"github.com/amalshaji/portr/internal/server/db"
	"github.com/amalshaji/portr/internal/tunnel/wsproto"
	"github.com/amalshaji/portr/internal/utils"
	"github.com/charmbracelet/log"
	"github.com/hashicorp/yamux"
	"github.com/oklog/ulid/v2"
	"golang.org/x/net/websocket"
)

type Manager struct {
	config       *config.Config
	service      connectionService
	mu           sync.RWMutex
	bySub        map[string][]*session
	sessions     map[string]*session
	rrIdx        map[string]int
	workers      sync.WaitGroup
	shutdownOnce sync.Once
	shutdownDone chan struct{}
	shuttingDown bool
}

type connectionService interface {
	GetReservedConnectionById(context.Context, string) (*db.Connection, error)
	MarkConnectionAsActive(context.Context, string) error
	MarkTCPConnectionAsActive(context.Context, string, uint32) error
	MarkConnectionAsClosed(context.Context, string) error
}

type session struct {
	id         string
	connection *db.Connection
	conn       io.Closer
	// mux carries v2 sessions; v1 sessions use writer and streams instead.
	mux *yamux.Session
	// proxy forwards HTTP requests over mux. Its transport keeps idle streams
	// open, so both visitor connections and tunnel streams are reused.
	proxy     *httputil.ReverseProxy
	transport *http.Transport
	writer    *wsproto.Writer
	streams   map[string]*streamQueue
	streamMu  sync.Mutex
	// ready is closed once the ready frame is on the wire; streams opened
	// before that would put yamux frames ahead of the handshake.
	ready     chan struct{}
	workers   sync.WaitGroup
	workerMu  sync.Mutex
	closing   bool
	listener  net.Listener
	closed    chan struct{}
	closeOnce sync.Once
	reasonMu  sync.Mutex
	reason    string
}

// setEndReason records the first cause seen for the session ending.
func (s *session) setEndReason(reason string) {
	s.reasonMu.Lock()
	defer s.reasonMu.Unlock()
	if s.reason == "" {
		s.reason = reason
	}
}

func (s *session) endReason() string {
	s.reasonMu.Lock()
	defer s.reasonMu.Unlock()
	if s.reason == "" {
		return "closed by server"
	}
	return s.reason
}

// muxLogger routes yamux's own error logs to the server log and keeps the
// first one as the session's end reason.
type muxLogger struct {
	sess *session
}

func (l muxLogger) Print(v ...any)                 { l.log(fmt.Sprint(v...)) }
func (l muxLogger) Println(v ...any)               { l.log(fmt.Sprint(v...)) }
func (l muxLogger) Printf(format string, v ...any) { l.log(fmt.Sprintf(format, v...)) }

func (l muxLogger) log(message string) {
	message = strings.TrimSpace(message)
	l.sess.setEndReason(message)
	log.Warn("Tunnel session error", "connection_id", l.sess.connection.ID, "error", message)
}

func New(config *config.Config, service connectionService) *Manager {
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
	sess, err := m.nextHTTPSession(subdomain)
	if err != nil {
		unregisteredSubdomainError(writer, subdomain)
		return
	}
	defer sess.workers.Done()

	if sess.mux != nil {
		sess.proxy.ServeHTTP(writer, request)
		return
	}
	// v1 sessions carry raw bytes, so the visitor connection is handed over.
	conn, initial, err := hijackRequest(writer, request)
	if err != nil {
		log.Error("Failed to hijack proxied request", "error", err, "subdomain", subdomain)
		http.Error(writer, "failed to proxy request", http.StatusBadGateway)
		return
	}
	m.pipeLegacyStream(sess, conn, initial)
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
			sess.setEndReason("server shutting down")
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

func (m *Manager) handle(conn *websocket.Conn) {
	req := conn.Request()
	version, err := strconv.Atoi(req.Header.Get(wsproto.ProtocolVersionHeader))
	if err != nil || (version != wsproto.ProtocolVersion && version != wsproto.LegacyProtocolVersion) {
		_ = wsproto.NewWriter(conn).Send(wsproto.Frame{Type: wsproto.TypeError, Message: fmt.Sprintf(
			"portr websocket protocol mismatch: server speaks v%d and v%d, client sent %q; upgrade the portr client",
			wsproto.LegacyProtocolVersion, wsproto.ProtocolVersion, req.Header.Get(wsproto.ProtocolVersionHeader))})
		_ = conn.Close()
		return
	}
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

	writer := wsproto.NewWriter(conn)
	sess := &session{
		id:         ulid.Make().String(),
		connection: reserved,
		conn:       conn,
		writer:     writer,
		ready:      make(chan struct{}),
		closed:     make(chan struct{}),
	}
	if version == wsproto.ProtocolVersion {
		mux, err := yamux.Server(wsproto.MuxConn(conn), wsproto.MuxConfig(muxLogger{sess: sess}))
		if err != nil {
			_ = writer.Send(wsproto.Frame{Type: wsproto.TypeError, Message: err.Error()})
			_ = conn.Close()
			return
		}
		sess.mux = mux
		sess.newHTTPProxy()
	} else {
		sess.streams = make(map[string]*streamQueue)
	}

	if err := m.registerSession(req.Context(), sess); err != nil {
		_ = writer.Send(wsproto.Frame{Type: wsproto.TypeError, Message: err.Error()})
		if sess.mux != nil {
			_ = sess.mux.Close()
		}
		_ = conn.Close()
		return
	}
	started := time.Now()
	defer func() {
		m.unregisterSession(sess)
		sess.workers.Wait()
		log.Info("Tunnel session ended", "connection_id", reserved.ID, "protocol", version,
			"duration", time.Since(started).Round(time.Second), "reason", sess.endReason())
	}()

	ready := wsproto.Frame{Type: wsproto.TypeReady, Version: version}
	if reserved.Port != nil {
		ready.Port = int(*reserved.Port)
	}
	if err := writer.Send(ready); err != nil {
		sess.setEndReason("send ready frame: " + err.Error())
		return
	}
	close(sess.ready)

	if sess.mux == nil {
		sess.serveLegacyFrames(conn)
		return
	}
	select {
	case <-sess.mux.CloseChan():
		sess.setEndReason("tunnel connection closed")
	case <-sess.closed:
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
			if conn, ok := sess.conn.(interface{ SetDeadline(time.Time) error }); ok {
				_ = conn.SetDeadline(time.Now())
			}
		}
		if sess.mux != nil {
			_ = sess.mux.Close()
			sess.transport.CloseIdleConnections()
		}
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
		if shouldMarkClosed {
			closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = m.service.MarkConnectionAsClosed(closeCtx, sess.connection.ID)
			cancel()
		}
		m.mu.Unlock()
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

// newHTTPProxy sets up the reverse proxy that carries HTTP requests over yamux
// streams. Request and response bytes pass through unchanged apart from
// hop-by-hop headers.
func (s *session) newHTTPProxy() {
	s.transport = &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			select {
			case <-s.ready:
			case <-s.closed:
				return nil, net.ErrClosed
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			return s.mux.OpenStream()
		},
		DisableCompression:  true,
		MaxIdleConnsPerHost: 64,
		IdleConnTimeout:     90 * time.Second,
	}
	s.proxy = &httputil.ReverseProxy{
		Rewrite: func(request *httputil.ProxyRequest) {
			request.Out.URL.Scheme = "http"
			request.Out.URL.Host = "portr-tunnel"
			request.Out.Host = request.In.Host
			// Rewrite strips forwarding headers; keep the visitor's.
			for _, name := range []string{"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto"} {
				if values, ok := request.In.Header[name]; ok {
					request.Out.Header[name] = values
				}
			}
		},
		Transport:     s.transport,
		FlushInterval: -1,
		ErrorHandler: func(writer http.ResponseWriter, request *http.Request, err error) {
			if request.Context().Err() == nil {
				log.Warn("Tunnel stream failed", "connection_id", s.connection.ID, "error", err)
			}
			connectionLostError(writer)
		},
	}
}

func (m *Manager) pipeStream(sess *session, conn net.Conn, initial []byte) {
	if sess.mux == nil {
		m.pipeLegacyStream(sess, conn, initial)
		return
	}
	defer conn.Close()

	finished := make(chan struct{})
	defer close(finished)
	go func() {
		select {
		case <-sess.closed:
			_ = conn.Close()
		case <-finished:
		}
	}()

	select {
	case <-sess.ready:
	case <-sess.closed:
		return
	}
	stream, err := sess.mux.OpenStream()
	if err != nil {
		return
	}
	defer stream.Close()
	if len(initial) > 0 {
		if _, err := stream.Write(initial); err != nil {
			return
		}
	}

	upstreamDone := make(chan struct{})
	go func() {
		defer close(upstreamDone)
		if _, err := io.Copy(stream, conn); err != nil {
			_ = conn.Close()
		}
		// yamux Close only half-closes: the response keeps flowing back.
		_ = stream.Close()
	}()

	if _, err := io.Copy(conn, stream); err != nil {
		return
	}
	closeWriter, ok := conn.(interface{ CloseWrite() error })
	if !ok || closeWriter.CloseWrite() != nil {
		return
	}
	select {
	case <-upstreamDone:
	case <-sess.closed:
	}
}

func unregisteredSubdomainError(writer http.ResponseWriter, subdomain string) {
	writer.Header().Set("Content-Type", "text/html")
	writer.Header().Set("X-Portr-Error", "true")
	writer.Header().Set("X-Portr-Error-Reason", "unregistered-subdomain")
	writer.WriteHeader(http.StatusNotFound)
	_, _ = writer.Write([]byte(utils.UnregisteredSubdomain(subdomain)))
}

func connectionLostError(writer http.ResponseWriter) {
	writer.Header().Set("Content-Type", "text/html")
	writer.Header().Set("X-Portr-Error", "true")
	writer.Header().Set("X-Portr-Error-Reason", "connection-lost")
	writer.WriteHeader(http.StatusServiceUnavailable)
	_, _ = writer.Write([]byte(utils.ConnectionLost()))
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
