package ssh

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/amalshaji/portr/internal/constants"
	"github.com/amalshaji/portr/internal/utils"
	"golang.org/x/crypto/ssh"
)

type tunnelTransport struct {
	client     *ssh.Client
	listener   net.Listener
	remotePort int
	closeOnce  sync.Once
	closeErr   error
}

func (t *tunnelTransport) Close() error {
	if t == nil {
		return nil
	}
	t.closeOnce.Do(func() {
		if t.client != nil {
			t.closeErr = t.client.Close()
		}
		if t.listener != nil {
			if err := t.listener.Close(); err != nil && t.closeErr == nil {
				t.closeErr = err
			}
		}
	})
	return t.closeErr
}

func (s *SshClient) establishTransport(ctx context.Context) (*tunnelTransport, error) {
	if atomic.LoadInt32(&s.shutdown) == 1 {
		return nil, errClientShuttingDown
	}

	connectionID, err := s.createNewConnection(ctx)
	if err != nil {
		return nil, err
	}
	sshConfig := &ssh.ClientConfig{
		User: fmt.Sprintf("%s:%s", connectionID, s.config.SecretKey),
		Auth: []ssh.AuthMethod{
			ssh.Password(""),
		},
		HostKeyCallback: getHostKeyCallback(s.config.InsecureSkipHostKeyVerification),
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 15 * time.Second}
	rawConn, err := dialer.DialContext(ctx, "tcp", s.config.SshUrl)
	if err != nil {
		return nil, err
	}
	if tcp, ok := rawConn.(*net.TCPConn); ok {
		_ = tcp.SetKeepAlive(true)
		_ = tcp.SetKeepAlivePeriod(15 * time.Second)
		_ = tcp.SetNoDelay(true)
	}

	handshakeDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = rawConn.Close()
		case <-handshakeDone:
		}
	}()
	_ = rawConn.SetDeadline(time.Now().Add(10 * time.Second))
	cc, channels, requests, err := ssh.NewClientConn(rawConn, s.config.SshUrl, sshConfig)
	close(handshakeDone)
	_ = rawConn.SetDeadline(time.Time{})
	if err != nil {
		_ = rawConn.Close()
		return nil, err
	}

	client := ssh.NewClient(cc, channels, requests)
	setupDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = client.Close()
		case <-setupDone:
		}
	}()
	defer close(setupDone)

	ports := remotePortCandidates(s.tunnelType())

	var listenErr error
	for _, port := range ports {
		if ctx.Err() != nil || atomic.LoadInt32(&s.shutdown) == 1 {
			_ = client.Close()
			return nil, errClientShuttingDown
		}
		listener, err := client.Listen("tcp", net.JoinHostPort("0.0.0.0", fmt.Sprint(port)))
		if err != nil {
			listenErr = err
			continue
		}
		return &tunnelTransport{client: client, listener: listener, remotePort: port}, nil
	}

	_ = client.Close()
	if listenErr == nil {
		listenErr = errors.New("no remote ports available")
	}
	return nil, fmt.Errorf("failed to listen on remote endpoint: %w", listenErr)
}

func remotePortCandidates(tunnelType constants.ConnectionType) []int {
	if tunnelType == constants.Http {
		// Keep non-zero ports for compatibility with legacy servers, whose
		// registration callback observes the requested port before binding.
		return utils.GenerateRandomHttpPorts()
	}
	return utils.GenerateRandomTcpPorts()
}

func (s *SshClient) installTransport(ctx context.Context, transport *tunnelTransport) bool {
	s.mu.Lock()
	if ctx.Err() != nil || atomic.LoadInt32(&s.shutdown) == 1 {
		s.mu.Unlock()
		_ = transport.Close()
		return false
	}
	previous := s.transport
	s.transport = transport
	s.config.Tunnel.RemotePort = transport.remotePort
	s.mu.Unlock()
	if previous != nil {
		_ = previous.Close()
	}
	return true
}

func (s *SshClient) clearTransport(transport *tunnelTransport) {
	s.mu.Lock()
	if s.transport == transport {
		s.transport = nil
	}
	s.mu.Unlock()
}

func (s *SshClient) tunnelType() constants.ConnectionType {
	tunnelType := s.config.Tunnel.Type
	if tunnelType == constants.Stub {
		return constants.Http
	}
	return tunnelType
}

func (s *SshClient) serveTransport(ctx context.Context, transport *tunnelTransport) error {
	localEndpoint := s.config.Tunnel.GetLocalAddr()
	tunnelType := s.tunnelType()
	for {
		remoteConn, err := transport.listener.Accept()
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
