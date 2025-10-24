package ssh

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/client/db"
	"github.com/amalshaji/portr/internal/constants"
	"github.com/amalshaji/portr/internal/utils"
	"github.com/charmbracelet/log"
	"github.com/go-resty/resty/v2"
	"gorm.io/datatypes"

	"github.com/amalshaji/portr/internal/client/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/oklog/ulid/v2"
	"golang.org/x/crypto/ssh"
)

var (
	ErrLocalSetupIncomplete = fmt.Errorf("local setup incomplete")
)

type SshClient struct {
	config       config.ClientConfig
	listener     net.Listener
	db           *db.Db
	client       *ssh.Client
	tui          *tea.Program
	mu           sync.RWMutex
	reconnecting int32 // atomic flag to prevent concurrent reconnects
	shutdown     int32 // atomic flag for shutdown state
}

func New(config config.ClientConfig, db *db.Db, tui *tea.Program) *SshClient {
	return &SshClient{
		config:   config,
		listener: nil,
		db:       db,
		client:   nil,
		tui:      tui,
	}
}

func CreateNewConnection(cfg config.ClientConfig) (string, error) {
	client := resty.New()
	var reqErr struct {
		Message string `json:"message"`
	}
	var response struct {
		ConnectionId string `json:"connection_id"`
	}

	payload := map[string]any{
		"connection_type": string(cfg.Tunnel.Type),
		"secret_key":      cfg.SecretKey,
		"subdomain":       nil,
	}
	request := client.R().
		SetError(&reqErr).
		SetResult(&response)

	if cfg.Tunnel.Type == constants.Http {
		payload["subdomain"] = cfg.Tunnel.Subdomain
	}

	resp, err := request.SetBody(payload).Post(cfg.GetServerAddr() + "/api/v1/connections/")

	if err != nil {
		return "", err
	}

	if resp.StatusCode() != 200 {
		log.Error("Failed to create new connection", "error", reqErr)
		return "", fmt.Errorf("server error: %s", reqErr.Message)
	}
	return response.ConnectionId, nil
}

func (s *SshClient) createNewConnection() (string, error) {
	if s.config.ConnectionID != "" {
		return s.config.ConnectionID, nil
	}
	return CreateNewConnection(s.config)
}

func (s *SshClient) startListenerForClient() error {
	if atomic.LoadInt32(&s.shutdown) == 1 {
		return fmt.Errorf("client is shutting down")
	}

	var err error
	var connectionId string

	if connectionId, err = s.createNewConnection(); err != nil {
		return err
	}

	sshConfig := &ssh.ClientConfig{
		User: fmt.Sprintf("%s:%s", connectionId, s.config.SecretKey),
		Auth: []ssh.AuthMethod{
			ssh.Password(""),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 15 * time.Second}
	rawConn, err := dialer.Dial("tcp", s.config.SshUrl)
	if err != nil {
		if s.config.Debug {
			s.logDebug("Failed to dial ssh tcp", err)
		}
		return err
	}
	if tcp, ok := rawConn.(*net.TCPConn); ok {
		_ = tcp.SetKeepAlive(true)
		_ = tcp.SetKeepAlivePeriod(15 * time.Second)
		_ = tcp.SetNoDelay(true)
	}

	cc, chans, reqs, err := ssh.NewClientConn(rawConn, s.config.SshUrl, sshConfig)
	if err != nil {
		_ = rawConn.Close()
		if s.config.Debug {
			s.logDebug("Failed to establish ssh client conn", err)
		}
		return err
	}

	s.mu.Lock()
	s.client = ssh.NewClient(cc, chans, reqs)
	clientRef := s.client
	s.mu.Unlock()

	go s.startSSHKeepAlive(clientRef)

	localEndpoint := s.config.Tunnel.GetLocalAddr()

	tunnelType := s.config.Tunnel.Type

	var randomPorts []int
	if tunnelType == constants.Http {
		randomPorts = utils.GenerateRandomHttpPorts()
	} else {
		randomPorts = utils.GenerateRandomTcpPorts()
	}

	var remotePort int
	var ln net.Listener

	for _, port := range randomPorts {
		l, lerr := s.client.Listen("tcp", "0.0.0.0:"+fmt.Sprint(port))
		if lerr == nil {
			ln = l
			remotePort = port
			break
		} else {
			err = lerr
		}
	}

	if ln == nil {
		return fmt.Errorf("failed to listen on remote endpoint: %v", err)
	}

	s.mu.Lock()
	s.listener = ln
	s.config.Tunnel.RemotePort = remotePort
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		if s.listener != nil {
			s.listener.Close()
			s.listener = nil
		}
		s.mu.Unlock()
	}()

	if s.tui != nil {
		s.tui.Send(tui.AddTunnelMsg{
			Config:       &s.config.Tunnel,
			ClientConfig: &s.config,
			Healthy:      true,
		})
	} else {
		tunnelAddr := s.config.GetTunnelAddr()
		fmt.Printf("✅ Tunnel started: %s → %s\n", s.config.Tunnel.GetLocalAddr(), tunnelAddr)
	}

	if s.tui != nil {
		s.tui.Send(tui.UpdateConnCountMsg{Port: fmt.Sprintf("%d", s.config.Tunnel.Port), Delta: 1})
	}

	for {
		if atomic.LoadInt32(&s.shutdown) == 1 {
			return fmt.Errorf("client is shutting down")
		}

		s.mu.RLock()
		listener := s.listener
		s.mu.RUnlock()

		if listener == nil {
			return fmt.Errorf("listener is nil, cannot accept connections")
		}

		remoteConn, err := listener.Accept()
		if err != nil {
			if s.config.Debug {
				log.Error("Failed to accept connection", "error", err)
			}
			if s.tui != nil {
				s.tui.Send(tui.UpdateConnCountMsg{Port: fmt.Sprintf("%d", s.config.Tunnel.Port), Delta: -1})
			}
			break
		}

		localConn, err := net.Dial("tcp", localEndpoint)
		if err != nil {
			if tunnelType == constants.Http {
				htmlContent := utils.LocalServerNotOnline(localEndpoint)
				fmt.Fprintf(remoteConn, "HTTP/1.1 503 Service Unavailable\r\n")
				fmt.Fprintf(remoteConn, "Content-Length: %d\r\n", len(htmlContent))
				fmt.Fprintf(remoteConn, "Content-Type: text/html\r\n")
				fmt.Fprintf(remoteConn, "X-Portr-Error: true\r\n")
				fmt.Fprintf(remoteConn, "X-Portr-Error-Reason: local-server-not-online\r\n\r\n")
				fmt.Fprint(remoteConn, htmlContent)
			}
			remoteConn.Close()
			continue
		}

		if tunnelType == constants.Http {
			go s.httpTunnel(remoteConn, localConn)
		} else {
			go s.tcpTunnel(remoteConn, localConn)
		}
	}

	return nil
}

func (s *SshClient) startSSHKeepAlive(clientRef *ssh.Client) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		if atomic.LoadInt32(&s.shutdown) == 1 {
			return
		}

		s.mu.RLock()
		sameClient := s.client == clientRef
		s.mu.RUnlock()
		if !sameClient {
			return
		}

		select {
		case <-ticker.C:
			errCh := make(chan error, 1)
			go func() {
				_, _, err := clientRef.SendRequest("keepalive@openssh.com", false, nil)
				errCh <- err
			}()

			select {
			case err := <-errCh:
				if err != nil {
					if s.config.Debug {
						s.logDebug("SSH keepalive failed, reconnecting", err)
					}
					_ = s.Reconnect()
					return
				}
			case <-time.After(5 * time.Second):
				if s.config.Debug {
					s.logDebug("SSH keepalive timed out, reconnecting", nil)
				}
				_ = s.Reconnect()
				return
			}
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (s *SshClient) httpTunnel(src, dst net.Conn) {
	defer src.Close()
	defer dst.Close()

	srcReader := bufio.NewReader(src)
	srcWriter := bufio.NewWriter(src)

	dstReader := bufio.NewReader(dst)
	dstWriter := bufio.NewWriter(dst)

	request, err := http.ReadRequest(srcReader)
	if err != nil {
		if s.config.Debug {
			s.logDebug("Failed to read request", err)
		}
		return
	}

	if request.Header.Get("X-Portr-Ping-Request") == "true" {
		response := &http.Response{
			Status:     "200 OK",
			StatusCode: 200,
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     http.Header{},
			Body:       io.NopCloser(bytes.NewBufferString("")),
		}

		err = response.Write(srcWriter)
		if err != nil {
			if s.config.Debug {
				log.Error("Failed to write health check response", "error", err)
			}
			return
		}
		srcWriter.Flush()
		return
	}

	requestBody, err := io.ReadAll(request.Body)
	if err != nil {
		if s.config.Debug {
			s.logDebug("Failed to read request body", err)
		}
		return
	}
	defer request.Body.Close()
	request.Body = io.NopCloser(bytes.NewBuffer(requestBody))

	err = request.Write(dstWriter)
	if err != nil {
		if s.config.Debug {
			s.logDebug("Failed to tunnel request to local", err)
		}
		return
	}
	dstWriter.Flush()

	response, err := http.ReadResponse(dstReader, request)
	if err != nil {
		if s.config.Debug {
			s.logDebug("Failed to read response", err)
		}
		return
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		if s.config.Debug {
			s.logDebug("Failed to read response body", err)
		}
		return
	}
	defer response.Body.Close()
	response.Body = io.NopCloser(bytes.NewBuffer(responseBody))

	err = response.Write(srcWriter)
	if err != nil {
		if s.config.Debug {
			s.logDebug("Failed to write response to remote", err)
		}
		return
	}
	srcWriter.Flush()

	if response.StatusCode == http.StatusSwitchingProtocols {
		s.tcpTunnel(src, dst)
		return
	}

	s.logHttpRequest(ulid.Make().String(), request, requestBody, response, responseBody)
}

func (s *SshClient) logHttpRequest(
	id string,
	request *http.Request,
	requestBody []byte,
	response *http.Response,
	responseBody []byte,
) {
	requestHeaders := make(map[string][]string)
	for key, values := range request.Header {
		if key == "X-Portr-Ping-Request" && len(values) > 0 {
			if values[0] == "true" {
				return
			}
		}
		requestHeaders[key] = values
	}

	var replayedRequestId string
	var isReplayedRequest bool

	_, isReplayedRequest = requestHeaders["X-Portr-Replayed-Request-Id"]
	if isReplayedRequest {
		replayedRequestId = requestHeaders["X-Portr-Replayed-Request-Id"][0]
		delete(requestHeaders, "X-Portr-Replayed-Request-Id")
	}

	requestHeadersBytes, err := json.Marshal(requestHeaders)
	if err != nil {
		if s.config.Debug {
			s.logDebug("Failed to marshal request headers", err)
		}
		return
	}

	responseHeaders := make(map[string][]string)
	for key, values := range response.Header {
		responseHeaders[key] = values
	}

	responseHeadersBytes, err := json.Marshal(responseHeaders)
	if err != nil {
		if s.config.Debug {
			s.logDebug("Failed to marshal request headers", err)
		}
		return
	}

	req := db.Request{
		ID:                 id,
		Host:               request.Host,
		Url:                request.URL.String(),
		Subdomain:          s.config.Tunnel.Subdomain,
		Localport:          s.config.Tunnel.Port,
		Method:             request.Method,
		Headers:            datatypes.JSON(requestHeadersBytes),
		Body:               requestBody,
		ResponseHeaders:    datatypes.JSON(responseHeadersBytes),
		ResponseBody:       responseBody,
		ResponseStatusCode: response.StatusCode,
		LoggedAt:           time.Now().UTC(),
		IsReplayed:         isReplayedRequest,
		ParentID:           replayedRequestId,
	}
	result := s.db.Conn.Create(&req)
	if result.Error != nil {
		if s.config.Debug {
			s.logDebug("Failed to log request", result.Error)
		}
		return
	}

	tunnelName := s.config.Tunnel.Name
	if tunnelName == "" {
		tunnelName = fmt.Sprintf("%d", s.config.Tunnel.Port)
	}

	if s.tui != nil {
		s.tui.Send(tui.AddLogMsg{
			Time:   req.LoggedAt.Local().Format("15:04:05"),
			Name:   tunnelName,
			Method: req.Method,
			Status: req.ResponseStatusCode,
			URL:    req.Url,
		})
	} else {
		fmt.Printf("[%s] %s %s → %d\n",
			req.LoggedAt.Local().Format("15:04:05"),
			req.Method,
			req.Url,
			req.ResponseStatusCode)
	}
}

func (s *SshClient) tcpTunnel(src, dst net.Conn) {
	defer src.Close()
	defer dst.Close()

	go io.Copy(dst, src)
	io.Copy(src, dst)
}

func (s *SshClient) Shutdown(ctx context.Context) error {
	atomic.StoreInt32(&s.shutdown, 1)

	s.mu.Lock()
	defer s.mu.Unlock()

	var err error
	if s.listener != nil {
		err = s.listener.Close()
		s.listener = nil
	}

	if s.client != nil {
		if clientErr := s.client.Close(); clientErr != nil && err == nil {
			err = clientErr
		}
		s.client = nil
	}

	if s.tui != nil {
		s.tui.Send(tui.UpdateConnCountMsg{Port: fmt.Sprintf("%d", s.config.Tunnel.Port), Delta: -1})
	}
	log.Info("Stopped tunnel connection", "address", s.config.GetTunnelAddr())
	return err
}

func (s *SshClient) StartHealthCheck(ctx context.Context) {
	ticker := time.Tick(time.Duration(s.config.HealthCheckInterval) * time.Second)
	retryAttempts := 0

	var err error

	for range ticker {
		retryAttempts++
		if retryAttempts > s.config.HealthCheckMaxRetries {
			if s.tui != nil {
				s.tui.Kill()
			}
			tunnelName := s.config.Tunnel.Name
			if tunnelName == "" {
				tunnelName = fmt.Sprintf("%d", s.config.Tunnel.Port)
			}
			fmt.Printf("💀 Failed to reconnect tunnel '%s' after %d attempts\n", tunnelName, retryAttempts)
			os.Exit(1)
		}

		err = s.HealthCheck()
		if err == nil {
			retryAttempts = 0
			continue
		}

		// Increment retry attempts only on failure
		retryAttempts++
		if retryAttempts > s.config.HealthCheckMaxRetries {
			if s.tui != nil {
				s.tui.Send(tui.ErrorMsg{Error: fmt.Errorf("failed to reconnect to tunnel after %d attempts", retryAttempts)})
				// Give TUI time to show the error
				time.Sleep(2 * time.Second)
			}
			return
		}

		if s.config.Debug {
			s.logDebug("Health check failed", err)
		}

		if s.tui != nil {
			s.tui.Send(tui.UpdateHealthMsg{
				Port:    fmt.Sprintf("%d", s.config.Tunnel.Port),
				Healthy: false,
			})
		} else {
			// Log unhealthy status when TUI is disabled
			fmt.Printf("❌ Tunnel unhealthy: %s (attempting reconnect)\n", s.config.GetTunnelAddr())
		}

		err = s.Reconnect()
		if err != nil {
			if s.config.Debug {
				s.logDebug(fmt.Sprintf("Failed to reconnect to ssh tunnel (attempt %d)", retryAttempts), err)
			}
		} else {
			retryAttempts = 0
		}

	}
}

func (s *SshClient) Start(ctx context.Context) {
	errChan := make(chan error, 1)

	go func() {
		if err := s.startListenerForClient(); err != nil {
			errChan <- err
		}
	}()

	// Wait for either an error or successful connection
	select {
	case err := <-errChan:
		// Update TUI with error and wait for it to quit
		if s.tui != nil {
			s.tui.Send(tui.ErrorMsg{Error: err})

			// Wait for TUI to quit
			done := make(chan struct{})
			go func() {
				s.tui.Wait()
				close(done)
			}()

			// Wait for either context cancellation or TUI to quit
			select {
			case <-ctx.Done():
				os.Exit(1)
			case <-done:
				os.Exit(1)
			}
		} else {
			// No TUI, just log the error and exit
			tunnelName := s.config.Tunnel.Name
			if tunnelName == "" {
				tunnelName = fmt.Sprintf("%d", s.config.Tunnel.Port)
			}
			fmt.Printf("❌ Failed to start tunnel '%s': %v\n", tunnelName, err)
			os.Exit(1)
		}
		return

	case <-time.After(5 * time.Second):
		// Start the health check routine for http connections
		if s.config.Tunnel.Type == constants.Http {
			s.StartHealthCheck(ctx)
		}
	}
}

func (s *SshClient) Reconnect() error {
	// Prevent concurrent reconnects using atomic CAS
	if !atomic.CompareAndSwapInt32(&s.reconnecting, 0, 1) {
		return fmt.Errorf("reconnect already in progress")
	}
	defer atomic.StoreInt32(&s.reconnecting, 0)

	// Check if we're shutting down
	if atomic.LoadInt32(&s.shutdown) == 1 {
		return fmt.Errorf("client is shutting down")
	}

	// Close existing connections with mutex protection
	s.mu.Lock()
	if s.client != nil {
		if err := s.client.Close(); err != nil {
			if s.config.Debug {
				s.logDebug("Failed to close client", err)
			}
		}
		s.client = nil
	}

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			if s.config.Debug {
				s.logDebug("Failed to close listener", err)
			}
		}
		s.listener = nil
	}
	s.mu.Unlock()

	// Create context with timeout for the reconnection attempt
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Channel to receive errors from the goroutine
	errChan := make(chan error, 1)
	done := make(chan struct{})

	// Start the listener in a goroutine with context
	go func() {
		defer close(done)
		if err := s.startListenerForClient(); err != nil {
			select {
			case errChan <- err:
			case <-ctx.Done():
			}
		}
	}()

	// Wait for either an error, successful connection, or timeout
	select {
	case err := <-errChan:
		return err
	case <-done:
		// Connection successful, update health status
		if s.tui != nil {
			s.tui.Send(tui.UpdateHealthMsg{
				Port:    fmt.Sprintf("%d", s.config.Tunnel.Port),
				Healthy: true,
			})
			// Increase active count after successful reconnect
			s.tui.Send(tui.UpdateConnCountMsg{Port: fmt.Sprintf("%d", s.config.Tunnel.Port), Delta: 1})
		} else {
			// Log successful reconnection when TUI is disabled
			fmt.Printf("🔄 Tunnel reconnected: %s\n", s.config.GetTunnelAddr())
		}
		return nil
	case <-time.After(5 * time.Second):
		// Fallback timeout in case done channel doesn't signal
		if s.tui != nil {
			s.tui.Send(tui.UpdateHealthMsg{
				Port:    fmt.Sprintf("%d", s.config.Tunnel.Port),
				Healthy: true,
			})
		} else {
			// Log successful reconnection when TUI is disabled
			fmt.Printf("🔄 Tunnel reconnected: %s\n", s.config.GetTunnelAddr())
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("reconnect timeout")
	}
}

func (s *SshClient) HealthCheck() error {
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
			Port:    fmt.Sprintf("%d", s.config.Tunnel.Port),
			Healthy: true,
		})
	}

	return nil
}

func (s *SshClient) logDebug(message string, err error) {
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
