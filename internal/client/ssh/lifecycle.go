package ssh

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync/atomic"
	"time"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/client/tui"
	"github.com/amalshaji/portr/internal/constants"
	"github.com/charmbracelet/log"
)

func (s *SshClient) Shutdown(ctx context.Context) error {
	atomic.StoreInt32(&s.shutdown, 1)
	s.mu.Lock()
	if s.lifecycleCancel != nil {
		s.lifecycleCancel()
	}
	lifecycleDone := s.lifecycleDone
	s.mu.Unlock()

	err := s.closeTransport()
	if lifecycleDone != nil {
		select {
		case <-lifecycleDone:
		case <-ctx.Done():
			if err == nil {
				err = ctx.Err()
			}
		}
	} else {
		s.mu.RLock()
		recorder := s.recorder
		s.mu.RUnlock()
		if recorderErr := recorder.close(ctx); recorderErr != nil && err == nil {
			err = recorderErr
		}
	}
	cfg := s.ConfigSnapshot()
	if s.tui != nil {
		s.tui.Send(tui.UpdateConnCountMsg{Port: tunnelStatusKey(cfg.Tunnel), Delta: -1})
	}
	s.emitEvent(EventStopped, nil)
	log.Info("Stopped tunnel connection", "address", cfg.GetTunnelAddr())
	return err
}

func reconnectBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 5 {
		attempt = 5
	}
	base := time.Second * time.Duration(1<<(attempt-1))
	return base + time.Duration(rand.IntN(500))*time.Millisecond
}

func (s *SshClient) startError(err error) error {
	return fmt.Errorf("failed to start tunnel '%s': %w", tunnelDisplayName(s.config.Tunnel), err)
}

func tunnelDisplayName(tunnel config.Tunnel) string {
	if tunnel.Name != "" {
		return tunnel.Name
	}
	if tunnel.Type == constants.Stub && tunnel.Subdomain != "" {
		return tunnel.Subdomain
	}
	return fmt.Sprintf("%d", tunnel.Port)
}

func (s *SshClient) Start(ctx context.Context) error {
	if atomic.LoadInt32(&s.shutdown) == 1 {
		return errClientShuttingDown
	}
	lifecycleCtx, lifecycleCancel := context.WithCancel(ctx)
	lifecycleDone := make(chan struct{})
	s.mu.Lock()
	s.lifecycleCancel = lifecycleCancel
	s.lifecycleDone = lifecycleDone
	s.mu.Unlock()
	defer func() {
		lifecycleCancel()
		_ = s.closeTransport()
		s.connections.Wait()
		s.mu.RLock()
		recorder := s.recorder
		s.mu.RUnlock()
		recorderCtx, cancelRecorder := context.WithTimeout(context.Background(), 5*time.Second)
		_ = recorder.close(recorderCtx)
		cancelRecorder()
		s.mu.Lock()
		s.lifecycleCancel = nil
		s.lifecycleDone = nil
		s.mu.Unlock()
		close(lifecycleDone)
	}()

	transport, err := s.establishWithTimeout(lifecycleCtx)
	if err != nil {
		if lifecycleCtx.Err() != nil || atomic.LoadInt32(&s.shutdown) == 1 {
			return nil
		}
		startErr := s.startError(err)
		s.emitEvent(EventFailed, startErr)
		return startErr
	}
	if !s.installTransport(lifecycleCtx, transport) {
		return nil
	}
	s.announceTransportReady(true)

	maxRetries := s.config.HealthCheckMaxRetries
	if maxRetries < 1 {
		maxRetries = 1
	}
	for {
		err = s.monitorTransport(lifecycleCtx, transport)
		s.clearTransport(transport)
		_ = transport.Close()
		if lifecycleCtx.Err() != nil || atomic.LoadInt32(&s.shutdown) == 1 || errors.Is(err, errClientShuttingDown) {
			return nil
		}
		s.announceTransportLost(err)

		for attempt := 1; ; attempt++ {
			transport, err = s.establishWithTimeout(lifecycleCtx)
			if err == nil && s.installTransport(lifecycleCtx, transport) {
				s.announceTransportReady(false)
				break
			}
			if lifecycleCtx.Err() != nil || atomic.LoadInt32(&s.shutdown) == 1 {
				return nil
			}
			if s.config.Debug {
				s.logDebug(fmt.Sprintf("Failed to reconnect to ssh tunnel (attempt %d)", attempt), err)
			}
			if attempt >= maxRetries {
				return fmt.Errorf("failed to reconnect tunnel '%s' after %d attempts: %w", tunnelDisplayName(s.config.Tunnel), attempt, err)
			}
			if !waitForReconnect(lifecycleCtx, reconnectBackoff(attempt)) {
				return nil
			}
		}
	}
}

func (s *SshClient) establishWithTimeout(ctx context.Context) (*tunnelTransport, error) {
	setupCtx, cancel := context.WithTimeout(ctx, tunnelStartTimeout)
	defer cancel()
	return s.establishTransport(setupCtx)
}

func waitForReconnect(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (s *SshClient) monitorTransport(ctx context.Context, transport *tunnelTransport) error {
	serveErr := make(chan error, 1)
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				serveErr <- fmt.Errorf("tunnel listener panic: %v", recovered)
			}
		}()
		serveErr <- s.serveTransport(ctx, transport)
	}()

	interval := time.Duration(s.config.HealthCheckInterval) * time.Second
	if interval <= 0 {
		interval = 3 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = transport.Close()
			<-serveErr
			return errClientShuttingDown
		case err := <-serveErr:
			return err
		case <-ticker.C:
			if err := checkSSHKeepAlive(transport.client, 5*time.Second); err != nil {
				_ = transport.Close()
				<-serveErr
				return err
			}
			if s.tui != nil {
				s.tui.Send(tui.UpdateHealthMsg{Port: tunnelStatusKey(s.config.Tunnel), Healthy: true})
			}
		}
	}
}

func (s *SshClient) announceTransportReady(initial bool) {
	cfg := s.ConfigSnapshot()
	if initial {
		if s.tui != nil {
			s.tui.Send(tui.AddTunnelMsg{Config: &cfg.Tunnel, ClientConfig: &cfg, Healthy: true})
		} else if !cfg.DisableTerminalLogs {
			fmt.Printf("✅ Tunnel started: %s → %s\n", cfg.Tunnel.GetLocalAddr(), cfg.GetTunnelAddr())
		}
		s.emitEvent(EventStarted, nil)
	} else {
		if s.tui != nil {
			s.tui.Send(tui.UpdateHealthMsg{Port: tunnelStatusKey(cfg.Tunnel), Healthy: true})
		} else if !cfg.DisableTerminalLogs {
			fmt.Printf("🔄 Tunnel reconnected: %s\n", cfg.GetTunnelAddr())
		}
		s.emitEvent(EventReconnected, nil)
	}
	if s.tui != nil {
		s.tui.Send(tui.UpdateConnCountMsg{Port: tunnelStatusKey(cfg.Tunnel), Delta: 1})
	}
}

func (s *SshClient) announceTransportLost(err error) {
	cfg := s.ConfigSnapshot()
	if s.config.Debug {
		s.logDebug("Tunnel transport failed", err)
	}
	if s.tui != nil {
		s.tui.Send(tui.UpdateConnCountMsg{Port: tunnelStatusKey(cfg.Tunnel), Delta: -1})
		s.tui.Send(tui.UpdateHealthMsg{Port: tunnelStatusKey(cfg.Tunnel), Healthy: false})
	} else if !cfg.DisableTerminalLogs {
		fmt.Printf("❌ Tunnel unhealthy: %s (attempting reconnect)\n", cfg.GetTunnelAddr())
	}
	s.emitEvent(EventUnhealthy, err)
}

func (s *SshClient) HealthCheck() error {
	s.mu.RLock()
	transport := s.transport
	s.mu.RUnlock()
	if transport == nil || transport.client == nil || transport.listener == nil {
		return fmt.Errorf("ssh tunnel transport is not connected")
	}
	return checkSSHKeepAlive(transport.client, 5*time.Second)
}

type sshRequestSender interface {
	SendRequest(string, bool, []byte) (bool, []byte, error)
}

func checkSSHKeepAlive(client sshRequestSender, timeout time.Duration) error {
	type keepAliveResult struct {
		accepted bool
		err      error
	}
	result := make(chan keepAliveResult, 1)
	go func() {
		accepted, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
		result <- keepAliveResult{accepted: accepted, err: err}
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case response := <-result:
		if response.err != nil {
			return response.err
		}
		if !response.accepted {
			return fmt.Errorf("ssh keepalive rejected")
		}
	case <-timer.C:
		return fmt.Errorf("ssh keepalive timed out")
	}
	return nil
}

func (s *SshClient) logDebug(message string, err error) {
	if !s.config.Debug {
		return
	}
	errString := ""
	if err != nil {
		errString = err.Error()
	}
	if s.tui != nil {
		s.tui.Send(tui.AddDebugLogMsg{
			Time: time.Now().Format("15:04:05"), Level: "DEBUG", Message: message, Error: errString,
		})
	}
}
