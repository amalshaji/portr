package ssh

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/constants"
	"github.com/amalshaji/portr/internal/utils"
	gossh "golang.org/x/crypto/ssh"
)

type Session struct {
	client     *gossh.Client
	listener   net.Listener
	remotePort int
	closeOnce  sync.Once
	closeErr   error
}

type requestSender interface {
	SendRequest(string, bool, []byte) (bool, []byte, error)
}

func Connect(ctx context.Context, cfg config.ClientConfig, connectionID string) (*Session, error) {
	sshConfig := &gossh.ClientConfig{
		User:            fmt.Sprintf("%s:%s", connectionID, cfg.SecretKey),
		Auth:            []gossh.AuthMethod{gossh.Password("")},
		HostKeyCallback: getHostKeyCallback(cfg.InsecureSkipHostKeyVerification),
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 15 * time.Second}
	rawConn, err := dialer.DialContext(ctx, "tcp", cfg.SshUrl)
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
	clientConn, channels, requests, err := gossh.NewClientConn(rawConn, cfg.SshUrl, sshConfig)
	close(handshakeDone)
	_ = rawConn.SetDeadline(time.Time{})
	if err != nil {
		_ = rawConn.Close()
		return nil, err
	}

	client := gossh.NewClient(clientConn, channels, requests)
	setupDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = client.Close()
		case <-setupDone:
		}
	}()
	defer close(setupDone)

	var listenErr error
	for _, port := range remotePortCandidates(tunnelType(cfg)) {
		if ctx.Err() != nil {
			_ = client.Close()
			return nil, ctx.Err()
		}
		listener, err := client.Listen("tcp", net.JoinHostPort("0.0.0.0", fmt.Sprint(port)))
		if err != nil {
			listenErr = err
			continue
		}
		return &Session{client: client, listener: listener, remotePort: port}, nil
	}

	_ = client.Close()
	if listenErr == nil {
		listenErr = errors.New("no remote ports available")
	}
	return nil, fmt.Errorf("failed to listen on remote endpoint: %w", listenErr)
}

func (s *Session) Accept() (net.Conn, error) {
	return s.listener.Accept()
}

func (s *Session) RemotePort() int {
	return s.remotePort
}

func (s *Session) HealthCheck(timeout time.Duration) error {
	return checkKeepAlive(s.client, timeout)
}

func checkKeepAlive(sender requestSender, timeout time.Duration) error {
	type result struct {
		accepted bool
		err      error
	}
	resultCh := make(chan result, 1)
	go func() {
		accepted, _, err := sender.SendRequest("keepalive@openssh.com", true, nil)
		resultCh <- result{accepted: accepted, err: err}
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case response := <-resultCh:
		if response.err != nil {
			return response.err
		}
		if !response.accepted {
			return fmt.Errorf("ssh keepalive rejected")
		}
		return nil
	case <-timer.C:
		return fmt.Errorf("ssh keepalive timed out")
	}
}

func (s *Session) Close() error {
	if s == nil {
		return nil
	}
	s.closeOnce.Do(func() {
		if s.client != nil {
			s.closeErr = s.client.Close()
		}
		if s.listener != nil {
			if err := s.listener.Close(); err != nil && s.closeErr == nil {
				s.closeErr = err
			}
		}
	})
	return s.closeErr
}

func tunnelType(cfg config.ClientConfig) constants.ConnectionType {
	if cfg.Tunnel.Type == constants.Stub {
		return constants.Http
	}
	return cfg.Tunnel.Type
}

func remotePortCandidates(connectionType constants.ConnectionType) []int {
	if connectionType == constants.Http {
		return utils.GenerateRandomHttpPorts()
	}
	return utils.GenerateRandomTcpPorts()
}
