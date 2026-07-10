package tunnel

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/client/db"
	"github.com/amalshaji/portr/internal/constants"
	"github.com/amalshaji/portr/internal/tunnel/wsproto"
	"github.com/amalshaji/portr/internal/utils"
	"github.com/charmbracelet/log"
	"github.com/go-resty/resty/v2"

	"github.com/amalshaji/portr/internal/client/tui"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/net/websocket"
)

var (
	ErrLocalSetupIncomplete = fmt.Errorf("local setup incomplete")
	errClientShuttingDown   = errors.New("client is shutting down")
	newRestyClient          = resty.New
	tunnelStartTimeout      = 20 * time.Second
)

type Client struct {
	config       config.ClientConfig
	listener     net.Listener
	db           *db.Db
	conn         *websocket.Conn
	writer       *wsproto.Writer
	streams      map[string]*tunnelStream
	streamsMu    sync.Mutex
	tui          *tea.Program
	fatal        func(error)
	eventHandler func(Event)
	mu           sync.RWMutex
	reconnecting int32 // atomic flag to prevent concurrent reconnects
	shutdown     int32 // atomic flag for shutdown state
	tuiActive    int32 // atomic flag for whether this worker contributes to the TUI pool count
}

type tunnelStream struct {
	frames    chan wsproto.Frame
	done      chan struct{}
	closeOnce sync.Once
}

func newTunnelStream() *tunnelStream {
	return &tunnelStream{
		frames: make(chan wsproto.Frame, 32),
		done:   make(chan struct{}),
	}
}

func (s *tunnelStream) close() {
	s.closeOnce.Do(func() {
		close(s.done)
	})
}

type EventType string

const (
	EventStarted     EventType = "started"
	EventStopped     EventType = "stopped"
	EventUnhealthy   EventType = "unhealthy"
	EventReconnected EventType = "reconnected"
	EventFailed      EventType = "failed"
)

type Event struct {
	Type       EventType     `json:"type"`
	Tunnel     config.Tunnel `json:"tunnel"`
	TunnelAddr string        `json:"tunnel_addr"`
	Error      string        `json:"error,omitempty"`
	At         time.Time     `json:"at"`
}

func New(config config.ClientConfig, db *db.Db, tui *tea.Program, fatal func(error)) *Client {
	return &Client{
		config:   config,
		listener: nil,
		db:       db,
		streams:  make(map[string]*tunnelStream),
		tui:      tui,
		fatal:    fatal,
	}
}

func (s *Client) SetEventHandler(handler func(Event)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eventHandler = handler
}

func (s *Client) ConfigSnapshot() config.ClientConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *Client) emitEvent(eventType EventType, err error) {
	s.mu.RLock()
	handler := s.eventHandler
	cfg := s.config
	s.mu.RUnlock()

	if handler == nil {
		return
	}

	event := Event{
		Type:       eventType,
		Tunnel:     cfg.Tunnel,
		TunnelAddr: cfg.GetTunnelAddr(),
		At:         time.Now().UTC(),
	}
	if err != nil {
		event.Error = err.Error()
	}
	handler(event)
}

func (s *Client) setTUIActive(active bool) {
	if s.tui == nil {
		return
	}

	var next int32
	delta := -1
	if active {
		next = 1
		delta = 1
	}
	if atomic.SwapInt32(&s.tuiActive, next) == next {
		return
	}

	s.mu.RLock()
	port := tunnelStatusKey(s.config.Tunnel)
	s.mu.RUnlock()
	s.tui.Send(tui.UpdateConnCountMsg{Port: port, Delta: delta})
}

func (s *Client) reportFatal(err error) {
	if err == nil {
		return
	}

	s.emitEvent(EventFailed, err)

	if s.fatal != nil {
		s.fatal(err)
		return
	}

	log.Error("Tunnel worker failed", "error", err, "address", s.config.GetTunnelAddr())
}

func (s *Client) recoverPanic(scope string) {
	if r := recover(); r != nil {
		s.reportFatal(fmt.Errorf("%s panic: %v", scope, r))
	}
}

func (s *Client) goSafe(scope string, fn func()) {
	go func() {
		defer s.recoverPanic(scope)
		fn()
	}()
}

func (s *Client) shouldIgnoreListenerError(ctx context.Context, err error) bool {
	return err == nil ||
		ctx.Err() != nil ||
		atomic.LoadInt32(&s.shutdown) == 1 ||
		atomic.LoadInt32(&s.reconnecting) == 1 ||
		errors.Is(err, errClientShuttingDown)
}

func (s *Client) forwardListenerErrors(ctx context.Context, errChan <-chan error) {
	go func() {
		select {
		case err := <-errChan:
			if s.shouldIgnoreListenerError(ctx, err) {
				return
			}
			s.reportFatal(s.startError(err))
		case <-ctx.Done():
		}
	}()
}

func (s *Client) closeTransport() error {
	s.mu.Lock()
	var err error
	if s.listener != nil {
		err = s.listener.Close()
		s.listener = nil
	}

	if s.conn != nil {
		if clientErr := s.conn.Close(); clientErr != nil && err == nil {
			err = clientErr
		}
		s.conn = nil
	}
	s.writer = nil
	s.mu.Unlock()

	s.closeTunnelStreams()
	return err
}

func (s *Client) closeListenerIfCurrent(listener net.Listener) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listener == listener {
		_ = s.listener.Close()
		s.listener = nil
	}
}

func (s *Client) handleAcceptError(listener net.Listener, err error) error {
	if err == nil {
		return nil
	}

	s.mu.RLock()
	currentListener := s.listener
	s.mu.RUnlock()

	if atomic.LoadInt32(&s.shutdown) == 1 || currentListener != listener {
		return errClientShuttingDown
	}

	return fmt.Errorf("failed to accept connection: %w", err)
}

func CreateNewConnection(cfg config.ClientConfig) (string, error) {
	return CreateNewConnectionWithContext(context.Background(), cfg)
}

func CreateNewConnectionWithContext(ctx context.Context, cfg config.ClientConfig) (string, error) {
	client := newRestyClient().SetTimeout(10 * time.Second)
	var reqErr struct {
		Message string `json:"message"`
	}
	var response struct {
		ConnectionId string `json:"connection_id"`
	}

	connectionType := cfg.Tunnel.Type
	if connectionType == constants.Stub {
		connectionType = constants.Http
	}

	payload := map[string]any{
		"connection_type": string(connectionType),
		"secret_key":      cfg.SecretKey,
		"subdomain":       nil,
	}
	request := client.R().
		SetError(&reqErr).
		SetResult(&response)

	if cfg.Tunnel.Type == constants.Http || cfg.Tunnel.Type == constants.Stub {
		payload["subdomain"] = cfg.Tunnel.Subdomain
	}

	resp, err := request.SetContext(ctx).SetBody(payload).Post(cfg.GetServerAddr() + "/api/v1/connections/")

	if err != nil {
		return "", err
	}

	if resp.StatusCode() != 200 {
		if reqErr.Message == "" {
			reqErr.Message = resp.Status()
		}
		return "", fmt.Errorf("server error: %s", reqErr.Message)
	}
	return response.ConnectionId, nil
}

func (s *Client) createNewConnection(ctx context.Context) (string, error) {
	if s.config.ConnectionID != "" {
		return s.config.ConnectionID, nil
	}
	return CreateNewConnectionWithContext(ctx, s.config)
}

func tunnelStatusKey(tunnel config.Tunnel) string {
	if tunnel.Type == constants.Stub {
		return "stub:" + tunnel.Subdomain
	}
	return fmt.Sprintf("%d", tunnel.Port)
}

func (s *Client) startListenerForClient(ctx context.Context) error {
	return s.startListenerForClientWithReady(ctx, nil)
}

func (s *Client) startListenerForClientWithReady(ctx context.Context, ready chan<- struct{}) error {
	if atomic.LoadInt32(&s.shutdown) == 1 {
		return errClientShuttingDown
	}

	var err error
	var connectionId string

	if connectionId, err = s.createNewConnection(ctx); err != nil {
		return err
	}

	wsURL, err := s.tunnelWebSocketURL()
	if err != nil {
		return err
	}

	conn, err := s.dialTunnelWebSocket(wsURL, connectionId)
	if err != nil {
		if s.config.Debug {
			s.logDebug("Failed to establish websocket tunnel", err)
		}
		return err
	}

	readyCh := make(chan wsproto.Frame, 1)
	errCh := make(chan error, 1)

	s.mu.Lock()
	s.conn = conn
	s.writer = wsproto.NewWriter(conn)
	s.streams = make(map[string]*tunnelStream)
	s.mu.Unlock()

	s.goSafe("websocket tunnel reader", func() {
		s.readTunnelFrames(conn, readyCh, errCh)
	})

	ctxDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = s.closeTransport()
		case <-ctxDone:
		}
	}()
	defer close(ctxDone)

	select {
	case readyFrame := <-readyCh:
		if readyFrame.Port > 0 {
			s.mu.Lock()
			s.config.Tunnel.RemotePort = readyFrame.Port
			s.mu.Unlock()
		}
	case err := <-errCh:
		return err
	case <-time.After(10 * time.Second):
		_ = conn.Close()
		return fmt.Errorf("timed out waiting for websocket tunnel registration")
	case <-ctx.Done():
		_ = s.closeTransport()
		return errClientShuttingDown
	}

	if s.tui != nil {
		s.tui.Send(tui.AddTunnelMsg{
			Config:       &s.config.Tunnel,
			ClientConfig: &s.config,
			Healthy:      true,
		})
	} else if !s.config.DisableTerminalLogs {
		tunnelAddr := s.config.GetTunnelAddr()
		fmt.Printf("✅ Tunnel started: %s → %s\n", s.config.Tunnel.GetLocalAddr(), tunnelAddr)
	}

	s.emitEvent(EventStarted, nil)
	if ready != nil {
		close(ready)
	}

	s.setTUIActive(true)

	err = <-errCh
	if s.shouldIgnoreListenerError(context.Background(), err) {
		return nil
	}
	s.setTUIActive(false)
	return err
}

func (s *Client) tunnelWebSocketURL() (string, error) {
	raw := strings.TrimRight(s.config.WsUrl, "/")
	if raw == "" {
		raw = strings.TrimRight(s.config.TunnelUrl, "/")
	}
	if raw == "" {
		raw = strings.TrimRight(s.config.ServerUrl, "/")
	}

	if !strings.Contains(raw, "://") {
		scheme := "wss"
		if s.config.UseLocalHost || strings.HasPrefix(raw, "localhost:") || strings.HasPrefix(raw, "127.0.0.1:") {
			scheme = "ws"
		}
		raw = scheme + "://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("unsupported websocket tunnel scheme: %s", u.Scheme)
	}
	u.Path = "/_portr/tunnel/connect"
	u.RawQuery = ""
	return u.String(), nil
}

const (
	tunnelConnectionIDHeader = "X-Portr-Connection-ID"
	tunnelSecretKeyHeader    = "X-Portr-Secret-Key"
)

func (s *Client) dialTunnelWebSocket(wsURL, connectionID string) (*websocket.Conn, error) {
	wsConfig, err := websocket.NewConfig(wsURL, s.config.GetServerAddr())
	if err != nil {
		return nil, err
	}
	wsConfig.Header.Set(tunnelConnectionIDHeader, connectionID)
	wsConfig.Header.Set(tunnelSecretKeyHeader, s.config.SecretKey)
	return websocket.DialConfig(wsConfig)
}

func (s *Client) readTunnelFrames(conn *websocket.Conn, readyCh chan<- wsproto.Frame, errCh chan<- error) {
	for {
		frame, err := wsproto.Receive(conn)
		if err != nil {
			errCh <- err
			return
		}

		switch frame.Type {
		case wsproto.TypeReady:
			select {
			case readyCh <- frame:
			default:
			}
		case wsproto.TypeOpen:
			if frame.StreamID == "" {
				continue
			}
			stream := s.addTunnelStream(frame.StreamID)
			s.goSafe("websocket stream", func() {
				defer s.removeTunnelStream(frame.StreamID)
				s.handleTunnelStream(frame, stream)
			})
		case wsproto.TypeData, wsproto.TypeClose, wsproto.TypeError:
			s.deliverTunnelFrame(frame)
		}
	}
}

func (s *Client) handleTunnelStream(openFrame wsproto.Frame, stream *tunnelStream) {
	if s.config.Tunnel.Type == constants.Http {
		s.handleHTTPTunnelStream(openFrame, stream)
		return
	}

	s.handleTCPTunnelStream(openFrame, stream)
}

func (s *Client) handleHTTPTunnelStream(openFrame wsproto.Frame, stream *tunnelStream) {
	conn := newTunnelStreamConn(openFrame.StreamID, openFrame.Data, stream.frames, stream.done, s.sendTunnelFrame)
	s.httpTunnel(conn, s.config.Tunnel.GetLocalAddr())
}

func (s *Client) handleTCPTunnelStream(openFrame wsproto.Frame, stream *tunnelStream) {
	streamID := openFrame.StreamID
	localEndpoint := s.config.Tunnel.GetLocalAddr()
	localConn, err := net.Dial("tcp", localEndpoint)
	if err != nil {
		s.sendTunnelFrame(wsproto.Frame{Type: wsproto.TypeClose, StreamID: streamID})
		return
	}
	defer localConn.Close()

	if len(openFrame.Data) > 0 {
		if _, err := localConn.Write(openFrame.Data); err != nil {
			return
		}
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 32*1024)
		for {
			n, err := localConn.Read(buf)
			if n > 0 {
				if sendErr := s.sendTunnelFrame(wsproto.Frame{
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
		case frame, ok := <-stream.frames:
			if !ok {
				_ = s.sendTunnelFrame(wsproto.Frame{Type: wsproto.TypeClose, StreamID: streamID})
				return
			}
			switch frame.Type {
			case wsproto.TypeData:
				if _, err := localConn.Write(frame.Data); err != nil {
					return
				}
			case wsproto.TypeClose, wsproto.TypeError:
				return
			}
		case <-done:
			_ = s.sendTunnelFrame(wsproto.Frame{Type: wsproto.TypeClose, StreamID: streamID})
			return
		case <-stream.done:
			return
		}
	}
}

func (s *Client) sendTunnelFrame(frame wsproto.Frame) error {
	s.mu.RLock()
	writer := s.writer
	s.mu.RUnlock()
	if writer == nil {
		return errClientShuttingDown
	}
	return writer.Send(frame)
}

func (s *Client) sendLocalServerUnavailable(streamID, localEndpoint string) {
	htmlContent := []byte(utils.LocalServerNotOnline(localEndpoint))
	var response bytes.Buffer
	fmt.Fprintf(&response, "HTTP/1.1 503 Service Unavailable\r\n")
	fmt.Fprintf(&response, "Content-Length: %d\r\n", len(htmlContent))
	fmt.Fprintf(&response, "Content-Type: text/html\r\n")
	fmt.Fprintf(&response, "X-Portr-Error: true\r\n")
	fmt.Fprintf(&response, "X-Portr-Error-Reason: local-server-not-online\r\n\r\n")
	response.Write(htmlContent)
	_ = s.sendTunnelFrame(wsproto.Frame{Type: wsproto.TypeData, StreamID: streamID, Data: response.Bytes()})
}

func (s *Client) addTunnelStream(streamID string) *tunnelStream {
	stream := newTunnelStream()
	s.streamsMu.Lock()
	if s.streams == nil {
		s.streams = make(map[string]*tunnelStream)
	}
	s.streams[streamID] = stream
	s.streamsMu.Unlock()
	return stream
}

func (s *Client) removeTunnelStream(streamID string) {
	s.streamsMu.Lock()
	stream := s.streams[streamID]
	delete(s.streams, streamID)
	s.streamsMu.Unlock()
	if stream != nil {
		stream.close()
	}
}

func (s *Client) deliverTunnelFrame(frame wsproto.Frame) {
	if frame.StreamID == "" {
		return
	}
	s.streamsMu.Lock()
	stream := s.streams[frame.StreamID]
	s.streamsMu.Unlock()
	if stream == nil {
		return
	}
	select {
	case <-stream.done:
		return
	default:
	}
	select {
	case stream.frames <- frame:
	case <-stream.done:
	}
}

func (s *Client) closeTunnelStreams() {
	s.streamsMu.Lock()
	defer s.streamsMu.Unlock()
	for streamID, stream := range s.streams {
		stream.close()
		delete(s.streams, streamID)
	}
}

func (s *Client) tcpTunnel(src, dst net.Conn) {
	defer src.Close()
	defer dst.Close()

	s.goSafe("tcp tunnel copy", func() {
		_, _ = io.Copy(dst, src)
	})
	_, _ = io.Copy(src, dst)
}

func (s *Client) Shutdown(ctx context.Context) error {
	atomic.StoreInt32(&s.shutdown, 1)

	err := s.closeTransport()
	s.mu.RLock()
	cfg := s.config
	address := cfg.GetTunnelAddr()
	s.mu.RUnlock()

	s.setTUIActive(false)
	s.emitEvent(EventStopped, nil)
	log.Info("Stopped tunnel connection", "address", address)
	return err
}

func (s *Client) StartHealthCheck(ctx context.Context) error {
	ticker := time.NewTicker(time.Duration(s.config.HealthCheckInterval) * time.Second)
	defer ticker.Stop()
	retryAttempts := 0

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}

		if ctx.Err() != nil || atomic.LoadInt32(&s.shutdown) == 1 {
			return nil
		}

		err := s.HealthCheck()
		if err == nil {
			retryAttempts = 0
			continue
		}

		if ctx.Err() != nil || atomic.LoadInt32(&s.shutdown) == 1 {
			return nil
		}

		retryAttempts++

		if s.config.Debug {
			s.logDebug("Health check failed", err)
		}

		if ctx.Err() != nil || atomic.LoadInt32(&s.shutdown) == 1 {
			return nil
		}

		if s.tui != nil {
			s.setTUIActive(false)
			s.tui.Send(tui.UpdateHealthMsg{
				Port:    tunnelStatusKey(s.config.Tunnel),
				Healthy: false,
			})
		} else if !s.config.DisableTerminalLogs {
			// Log unhealthy status when TUI is disabled
			fmt.Printf("❌ Tunnel unhealthy: %s (attempting reconnect)\n", s.config.GetTunnelAddr())
		}

		if ctx.Err() != nil || atomic.LoadInt32(&s.shutdown) == 1 {
			return nil
		}

		s.emitEvent(EventUnhealthy, err)

		reconnectErr := s.Reconnect()
		if reconnectErr == nil {
			retryAttempts = 0
			continue
		}
		if errors.Is(reconnectErr, errClientShuttingDown) || ctx.Err() != nil || atomic.LoadInt32(&s.shutdown) == 1 {
			return nil
		}
		if s.config.Debug {
			s.logDebug(fmt.Sprintf("Failed to reconnect websocket tunnel (attempt %d)", retryAttempts), reconnectErr)
		}
		if retryAttempts > s.config.HealthCheckMaxRetries {
			return fmt.Errorf("failed to reconnect tunnel '%s' after %d attempts: %w", tunnelDisplayName(s.config.Tunnel), retryAttempts, reconnectErr)
		}
	}
}

func (s *Client) startError(err error) error {
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

func (s *Client) Start(ctx context.Context) error {
	errChan := make(chan error, 1)
	readyChan := make(chan struct{})
	startupCtx, cancelStartup := context.WithCancel(ctx)
	timer := time.NewTimer(tunnelStartTimeout)
	defer timer.Stop()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("tunnel listener panic: %v", r)
			}
		}()
		if err := s.startListenerForClientWithReady(startupCtx, readyChan); err != nil {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		if s.shouldIgnoreListenerError(ctx, err) {
			return nil
		}

		startErr := s.startError(err)
		s.emitEvent(EventFailed, startErr)
		return startErr

	case <-readyChan:
		s.forwardListenerErrors(startupCtx, errChan)

		if s.config.Tunnel.Type == constants.Http || s.config.Tunnel.Type == constants.Stub {
			defer cancelStartup()
			return s.StartHealthCheck(startupCtx)
		}
		return nil

	case <-timer.C:
		cancelStartup()
		_ = s.closeTransport()
		startErr := s.startError(fmt.Errorf("timed out waiting for tunnel listener after %s", tunnelStartTimeout))
		s.emitEvent(EventFailed, startErr)
		return startErr

	case <-ctx.Done():
		cancelStartup()
		_ = s.closeTransport()
		return nil
	}
}

func (s *Client) Reconnect() error {
	// Prevent concurrent reconnects using atomic CAS
	if !atomic.CompareAndSwapInt32(&s.reconnecting, 0, 1) {
		return fmt.Errorf("reconnect already in progress")
	}
	defer atomic.StoreInt32(&s.reconnecting, 0)

	// Check if we're shutting down
	if atomic.LoadInt32(&s.shutdown) == 1 {
		return errClientShuttingDown
	}

	// Close existing connections with mutex protection
	s.mu.Lock()
	if s.conn != nil {
		if err := s.conn.Close(); err != nil {
			if s.config.Debug {
				s.logDebug("Failed to close websocket client", err)
			}
		}
		s.conn = nil
	}
	s.writer = nil

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			if s.config.Debug {
				s.logDebug("Failed to close listener", err)
			}
		}
		s.listener = nil
	}
	s.mu.Unlock()
	s.closeTunnelStreams()

	// Channel to receive errors from the goroutine
	errChan := make(chan error, 1)
	readyChan := make(chan struct{})
	listenerCtx, cancelListener := context.WithCancel(context.Background())
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()

	// Start the listener in a goroutine with context
	go func() {
		defer func() {
			if r := recover(); r != nil {
				select {
				case errChan <- fmt.Errorf("reconnect panic: %v", r):
				default:
					s.reportFatal(fmt.Errorf("reconnect panic: %v", r))
				}
			}
		}()
		if err := s.startListenerForClientWithReady(listenerCtx, readyChan); err != nil {
			select {
			case errChan <- err:
			case <-listenerCtx.Done():
			}
		}
	}()

	// Wait for either an error, successful connection, or timeout
	select {
	case err := <-errChan:
		cancelListener()
		_ = s.closeTransport()
		return err
	case <-readyChan:
		// Connection successful, update health status
		if s.tui != nil {
			s.tui.Send(tui.UpdateHealthMsg{
				Port:    tunnelStatusKey(s.config.Tunnel),
				Healthy: true,
			})
		} else if !s.config.DisableTerminalLogs {
			// Log successful reconnection when TUI is disabled
			fmt.Printf("🔄 Tunnel reconnected: %s\n", s.config.GetTunnelAddr())
		}
		s.emitEvent(EventReconnected, nil)
		return nil
	case <-timer.C:
		cancelListener()
		_ = s.closeTransport()
		return fmt.Errorf("reconnect timeout")
	}
}

func (s *Client) HealthCheck() error {
	client := resty.New().
		SetTimeout(5 * time.Second)

	resp, err := client.R().
		SetHeader("X-Portr-Ping-Request", "true").
		Get(s.config.GetTunnelAddr())

	if err != nil {
		if s.config.Debug {
			s.logDebug("Health check failed, attempting to reconnect", err)
		}
		return err
	}

	portrError := resp.Header().Get("X-Portr-Error")
	portrErrorReason := resp.Header().Get("X-Portr-Error-Reason")

	if portrError == "true" && (portrErrorReason == "connection-lost" || portrErrorReason == "unregistered-subdomain") {
		return fmt.Errorf("unhealthy tunnel")
	}

	if s.tui != nil {
		s.tui.Send(tui.UpdateHealthMsg{
			Port:    tunnelStatusKey(s.config.Tunnel),
			Healthy: true,
		})
	}

	return nil
}

func (s *Client) logDebug(message string, err error) {
	if !s.config.Debug {
		return
	}

	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	if s.tui != nil {
		s.tui.Send(tui.AddDebugLogMsg{
			Time:    time.Now().Format("15:04:05"),
			Level:   "DEBUG",
			Message: message,
			Error:   errStr,
		})
	}
}
