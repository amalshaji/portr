package wstunnel

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/amalshaji/portr/internal/tunnel/quicproto"
	"github.com/charmbracelet/log"
	"github.com/oklog/ulid/v2"
	"github.com/quic-go/quic-go"
)

// quicHandshakeTimeout bounds how long a connection may stay unauthenticated.
var quicHandshakeTimeout = 5 * time.Second

// sourceValidationThreshold is the number of new connection attempts per second
// above which clients must prove their address with a Retry round trip, so the
// server cannot be used to amplify traffic towards spoofed addresses.
const sourceValidationThreshold = 64

const (
	quicCloseNormal   quic.ApplicationErrorCode = 0
	quicCloseRejected quic.ApplicationErrorCode = 1

	quicStreamCancelled quic.StreamErrorCode = 0
)

type quicServer struct {
	conn      *net.UDPConn
	transport *quic.Transport
	listener  *quic.Listener
}

// close stops accepting connections and releases the socket. Established
// connections are closed by unregistering their sessions first.
func (s *quicServer) close() {
	_ = s.listener.Close()
	_ = s.transport.Close()
	_ = s.conn.Close()
}

// ListenQUIC accepts QUIC tunnel connections on addr. It returns once the
// socket is bound, so bind errors surface at startup.
func (m *Manager) ListenQUIC(addr, tlsKey string) (net.Addr, error) {
	tlsConfig, err := quicproto.ServerTLSConfig(tlsKey)
	if err != nil {
		return nil, err
	}
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}
	limiter := &attemptLimiter{}
	transport := &quic.Transport{Conn: udpConn, VerifySourceAddress: limiter.overLimit}
	listener, err := transport.Listen(tlsConfig, quicproto.ServerConfig())
	if err != nil {
		_ = transport.Close()
		_ = udpConn.Close()
		return nil, err
	}
	server := &quicServer{conn: udpConn, transport: transport, listener: listener}

	m.mu.Lock()
	if m.shuttingDown {
		m.mu.Unlock()
		server.close()
		return nil, fmt.Errorf("tunnel manager is shutting down")
	}
	m.quic = server
	m.workers.Add(1)
	m.mu.Unlock()

	go func() {
		defer m.workers.Done()
		m.acceptQUIC(listener)
	}()
	return listener.Addr(), nil
}

func (m *Manager) acceptQUIC(listener *quic.Listener) {
	for {
		conn, err := listener.Accept(context.Background())
		if err != nil {
			return
		}
		m.mu.Lock()
		if m.shuttingDown {
			m.mu.Unlock()
			_ = conn.CloseWithError(quicCloseRejected, "server shutting down")
			return
		}
		m.workers.Add(1)
		m.mu.Unlock()
		go func() {
			defer m.workers.Done()
			m.handleQUIC(conn)
		}()
	}
}

func (m *Manager) handleQUIC(conn *quic.Conn) {
	// Rejections travel as the connection close reason, which the client
	// reports verbatim.
	reject := func(message string) {
		_ = conn.CloseWithError(quicCloseRejected, message)
	}

	acceptCtx, cancel := context.WithTimeout(conn.Context(), quicHandshakeTimeout)
	control, err := conn.AcceptStream(acceptCtx)
	cancel()
	if err != nil {
		reject("no control stream")
		return
	}
	_ = control.SetReadDeadline(time.Now().Add(quicHandshakeTimeout))
	hello, err := quicproto.ReadFrame(control)
	if err != nil {
		reject("invalid hello frame")
		return
	}
	_ = control.SetReadDeadline(time.Time{})

	if hello.Type != quicproto.TypeHello || hello.Version != quicproto.ProtocolVersion {
		reject(fmt.Sprintf("portr quic protocol mismatch: server speaks v%d, client sent v%d; upgrade the portr client",
			quicproto.ProtocolVersion, hello.Version))
		return
	}
	if hello.ConnectionID == "" || hello.SecretKey == "" {
		reject("missing connection credentials")
		return
	}
	reserved, err := m.service.GetReservedConnectionById(conn.Context(), hello.ConnectionID)
	if err != nil || reserved.CreatedBy.SecretKey != hello.SecretKey {
		reject("invalid connection credentials")
		return
	}

	mux := quicMux{conn: conn}
	sess := &session{
		id:         ulid.Make().String(),
		connection: reserved,
		conn:       mux,
		mux:        mux,
		ready:      make(chan struct{}),
		closed:     make(chan struct{}),
	}
	sess.newHTTPProxy()
	if err := m.registerSession(conn.Context(), sess); err != nil {
		reject(err.Error())
		return
	}
	started := time.Now()
	defer func() {
		m.unregisterSession(sess)
		sess.workers.Wait()
		log.Info("Tunnel session ended", "connection_id", reserved.ID, "protocol", "quic",
			"duration", time.Since(started).Round(time.Second), "reason", sess.endReason())
	}()

	ready := quicproto.Frame{Type: quicproto.TypeReady, Version: quicproto.ProtocolVersion}
	if reserved.Port != nil {
		ready.Port = int(*reserved.Port)
	}
	if err := quicproto.WriteFrame(control, ready); err != nil {
		sess.setEndReason("send ready frame: " + err.Error())
		return
	}
	close(sess.ready)
	sess.goWorker(func() { serveQUICControl(control) })

	select {
	case <-conn.Context().Done():
		sess.setEndReason(context.Cause(conn.Context()).Error())
	case <-sess.closed:
	}
}

// serveQUICControl answers client pings until the connection closes. The
// ready frame is written before this starts, so it is the only writer.
func serveQUICControl(control *quic.Stream) {
	for {
		frame, err := quicproto.ReadFrame(control)
		if err != nil {
			return
		}
		if frame.Type == quicproto.TypePing {
			if err := quicproto.WriteFrame(control, quicproto.Frame{Type: quicproto.TypePong}); err != nil {
				return
			}
		}
	}
}

type quicMux struct {
	conn *quic.Conn
}

func (m quicMux) OpenStream(ctx context.Context) (muxStream, error) {
	stream, err := m.conn.OpenStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := stream.Write([]byte{quicproto.DataStreamHeader}); err != nil {
		stream.CancelWrite(quicStreamCancelled)
		return nil, err
	}
	return quicStream{Stream: stream, conn: m.conn}, nil
}

func (m quicMux) CloseChan() <-chan struct{} {
	return m.conn.Context().Done()
}

func (m quicMux) Close() error {
	return m.conn.CloseWithError(quicCloseNormal, "tunnel closed")
}

type quicStream struct {
	*quic.Stream
	conn *quic.Conn
}

func (s quicStream) LocalAddr() net.Addr  { return s.conn.LocalAddr() }
func (s quicStream) RemoteAddr() net.Addr { return s.conn.RemoteAddr() }

// CloseWrite sends FIN; QUIC Close only ends the sending direction.
func (s quicStream) CloseWrite() error {
	return s.Stream.Close()
}

// Close releases both directions, discarding any response still in flight.
func (s quicStream) Close() error {
	s.Stream.CancelRead(quicStreamCancelled)
	return s.Stream.Close()
}

type attemptLimiter struct {
	mu     sync.Mutex
	window time.Time
	count  int
}

func (l *attemptLimiter) overLimit(net.Addr) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if now.Sub(l.window) >= time.Second {
		l.window = now
		l.count = 0
	}
	l.count++
	return l.count > sourceValidationThreshold
}
