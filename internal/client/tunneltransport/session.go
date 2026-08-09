package tunneltransport

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/amalshaji/portr/internal/constants"
)

func (s *Client) establishTransport(ctx context.Context) (tunnelSession, error) {
	if atomic.LoadInt32(&s.shutdown) == 1 {
		return nil, errClientShuttingDown
	}
	connectionID, err := s.createNewConnection(ctx)
	if err != nil {
		return nil, err
	}
	return s.connect(ctx, s.ConfigSnapshot(), connectionID)
}

func (s *Client) installTransport(ctx context.Context, transport tunnelSession) bool {
	s.mu.Lock()
	if ctx.Err() != nil || atomic.LoadInt32(&s.shutdown) == 1 {
		s.mu.Unlock()
		_ = transport.Close()
		return false
	}
	previous := s.transport
	s.transport = transport
	s.config.Tunnel.RemotePort = transport.RemotePort()
	s.mu.Unlock()
	if previous != nil {
		_ = previous.Close()
	}
	return true
}

func (s *Client) clearTransport(transport tunnelSession) {
	s.mu.Lock()
	if s.transport == transport {
		s.transport = nil
	}
	s.mu.Unlock()
}

func (s *Client) tunnelType() constants.ConnectionType {
	return s.config.Tunnel.Type.WireType()
}

func (s *Client) serveTransport(ctx context.Context, transport tunnelSession) error {
	localEndpoint := s.config.Tunnel.GetLocalAddr()
	tunnelType := s.tunnelType()
	for {
		remoteConn, err := transport.Accept()
		if err != nil {
			if ctx.Err() != nil || atomic.LoadInt32(&s.shutdown) == 1 {
				return errClientShuttingDown
			}
			return fmt.Errorf("failed to accept connection: %w", err)
		}

		if tunnelType == constants.Http {
			s.runConnection("http tunnel", func() {
				s.httpTunnel(remoteConn, localEndpoint)
			})
			continue
		}

		s.runConnection("tcp tunnel", func() {
			dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			localConn, err := (&net.Dialer{KeepAlive: 30 * time.Second}).DialContext(dialCtx, "tcp", localEndpoint)
			if err != nil {
				_ = remoteConn.Close()
				return
			}
			s.tcpTunnel(remoteConn, localConn)
		})
	}
}
