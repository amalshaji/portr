package tunnel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	config "github.com/amalshaji/portr/internal/clientconfig"
	"github.com/amalshaji/portr/internal/tunnel/quicproto"
	"github.com/quic-go/quic-go"
)

// ErrQUICUnreachable marks a QUIC handshake that never got an answer, usually
// because UDP to the server is blocked. Callers fall back to websocket on it.
var ErrQUICUnreachable = errors.New("quic endpoint unreachable")

// quicDialTimeout is short so a blocked UDP port falls back to websocket
// quickly.
var quicDialTimeout = 3 * time.Second

const (
	quicCloseNormal     quic.ApplicationErrorCode = 0
	quicStreamCancelled quic.StreamErrorCode      = 0
)

type QUICSession struct {
	conn       *quic.Conn
	control    *quic.Stream
	remotePort int
	pingMu     sync.Mutex
	pongs      chan struct{}
}

// ConnectQUIC opens a tunnel session over QUIC. Errors wrap ErrQUICUnreachable
// when the server never answered the handshake.
func ConnectQUIC(ctx context.Context, cfg config.ClientConfig, connectionID string) (*QUICSession, error) {
	tlsConfig, err := quicproto.ClientTLSConfig(cfg.QuicCertSha256, cfg.InsecureSkipQuicVerification)
	if err != nil {
		return nil, err
	}

	dialCtx, cancel := context.WithTimeout(ctx, quicDialTimeout)
	conn, err := quic.DialAddr(dialCtx, cfg.QuicUrl, tlsConfig, quicproto.ClientConfig())
	cancel()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		if unreachable(err) {
			return nil, fmt.Errorf("%w: %s: %v", ErrQUICUnreachable, cfg.QuicUrl, err)
		}
		return nil, err
	}

	control, ready, err := quicHandshake(ctx, conn, cfg, connectionID)
	if err != nil {
		_ = conn.CloseWithError(quicCloseNormal, "handshake failed")
		return nil, err
	}

	session := &QUICSession{
		conn:       conn,
		control:    control,
		remotePort: ready.Port,
		pongs:      make(chan struct{}, 1),
	}
	go session.readControl()
	return session, nil
}

func quicHandshake(ctx context.Context, conn *quic.Conn, cfg config.ClientConfig, connectionID string) (*quic.Stream, quicproto.Frame, error) {
	control, err := conn.OpenStream()
	if err != nil {
		return nil, quicproto.Frame{}, err
	}
	hello := quicproto.Frame{
		Type:         quicproto.TypeHello,
		ConnectionID: connectionID,
		SecretKey:    cfg.SecretKey,
		Version:      quicproto.ProtocolVersion,
	}
	if err := quicproto.WriteFrame(control, hello); err != nil {
		return nil, quicproto.Frame{}, closeReason(err)
	}

	deadline := time.Now().Add(10 * time.Second)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	_ = control.SetReadDeadline(deadline)
	stop := context.AfterFunc(ctx, func() { _ = control.SetReadDeadline(time.Now()) })
	ready, err := quicproto.ReadFrame(control)
	stop()
	_ = control.SetReadDeadline(time.Time{})
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, quicproto.Frame{}, ctxErr
		}
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return nil, quicproto.Frame{}, fmt.Errorf("timed out waiting for quic tunnel registration")
		}
		return nil, quicproto.Frame{}, closeReason(err)
	}
	if ready.Type == quicproto.TypeError {
		return nil, quicproto.Frame{}, errors.New(ready.Message)
	}
	if ready.Type != quicproto.TypeReady {
		return nil, quicproto.Frame{}, fmt.Errorf("expected quic tunnel ready frame, got %q", ready.Type)
	}
	if ready.Version != quicproto.ProtocolVersion {
		return nil, quicproto.Frame{}, fmt.Errorf(
			"portr quic protocol mismatch: client speaks v%d, server sent v%d; upgrade the portr server",
			quicproto.ProtocolVersion, ready.Version)
	}
	return control, ready, nil
}

// unreachable reports whether a dial error means no answer came back, as
// opposed to the server answering with a failure such as a pin mismatch.
func unreachable(err error) bool {
	var idle *quic.IdleTimeoutError
	var handshake *quic.HandshakeTimeoutError
	var opErr *net.OpError
	return errors.Is(err, context.DeadlineExceeded) ||
		errors.As(err, &idle) || errors.As(err, &handshake) || errors.As(err, &opErr)
}

// closeReason replaces a stream error with the server's close message, which
// says why the server rejected the tunnel.
func closeReason(err error) error {
	var appErr *quic.ApplicationError
	if errors.As(err, &appErr) && appErr.Remote && appErr.ErrorMessage != "" {
		return errors.New(appErr.ErrorMessage)
	}
	return err
}

// readControl delivers pongs to HealthCheck until the connection closes.
func (s *QUICSession) readControl() {
	for {
		frame, err := quicproto.ReadFrame(s.control)
		if err != nil {
			return
		}
		if frame.Type == quicproto.TypePong {
			select {
			case s.pongs <- struct{}{}:
			default:
			}
		}
	}
}

func (s *QUICSession) Accept() (net.Conn, error) {
	stream, err := s.conn.AcceptStream(context.Background())
	if err != nil {
		return nil, closeReason(err)
	}
	// The server writes the header as soon as it opens the stream.
	_ = stream.SetReadDeadline(time.Now().Add(10 * time.Second))
	header := make([]byte, 1)
	if _, err := io.ReadFull(stream, header); err != nil || header[0] != quicproto.DataStreamHeader {
		stream.CancelRead(quicStreamCancelled)
		stream.CancelWrite(quicStreamCancelled)
		_ = s.Close()
		return nil, fmt.Errorf("invalid quic data stream header")
	}
	_ = stream.SetReadDeadline(time.Time{})
	return quicStreamConn{Stream: stream, conn: s.conn}, nil
}

func (s *QUICSession) RemotePort() int {
	return s.remotePort
}

func (s *QUICSession) HealthCheck(timeout time.Duration) error {
	s.pingMu.Lock()
	defer s.pingMu.Unlock()
	select {
	case <-s.pongs:
	default:
	}
	if err := quicproto.WriteFrame(s.control, quicproto.Frame{Type: quicproto.TypePing}); err != nil {
		return closeReason(err)
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-s.pongs:
		return nil
	case <-s.conn.Context().Done():
		return closeReason(context.Cause(s.conn.Context()))
	case <-timer.C:
		_ = s.Close()
		return fmt.Errorf("quic tunnel ping timed out")
	}
}

func (s *QUICSession) Close() error {
	return s.conn.CloseWithError(quicCloseNormal, "client closed")
}

type quicStreamConn struct {
	*quic.Stream
	conn *quic.Conn
}

func (c quicStreamConn) LocalAddr() net.Addr  { return c.conn.LocalAddr() }
func (c quicStreamConn) RemoteAddr() net.Addr { return c.conn.RemoteAddr() }

// CloseWrite sends FIN; QUIC Close only ends the sending direction.
func (c quicStreamConn) CloseWrite() error {
	return c.Stream.Close()
}

// Close releases both directions, discarding data the peer is still sending.
func (c quicStreamConn) Close() error {
	c.Stream.CancelRead(quicStreamCancelled)
	return c.Stream.Close()
}
