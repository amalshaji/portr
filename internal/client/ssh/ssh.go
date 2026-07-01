package ssh

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
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
)

var (
	ErrLocalSetupIncomplete = fmt.Errorf("local setup incomplete")
	errClientShuttingDown   = errors.New("client is shutting down")
	newRestyClient          = resty.New
	tunnelStartTimeout      = 20 * time.Second
)

type SshClient struct {
	config          config.ClientConfig
	db              *db.Db
	transport       *tunnelTransport
	tui             *tea.Program
	fatal           func(error)
	eventHandler    func(Event)
	recorder        *captureRecorder
	mu              sync.RWMutex
	connections     sync.WaitGroup
	shutdown        int32
	lifecycleCancel context.CancelFunc
	lifecycleDone   chan struct{}
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

func New(config config.ClientConfig, db *db.Db, tui *tea.Program, fatal func(error)) *SshClient {
	return &SshClient{
		config: config,
		db:     db,
		tui:    tui,
		fatal:  fatal,
	}
}

func (s *SshClient) SetEventHandler(handler func(Event)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eventHandler = handler
}

func (s *SshClient) ConfigSnapshot() config.ClientConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *SshClient) emitEvent(eventType EventType, err error) {
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

func (s *SshClient) reportFatal(err error) {
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

func (s *SshClient) recoverPanic(scope string) {
	if r := recover(); r != nil {
		s.reportFatal(fmt.Errorf("%s panic: %v", scope, r))
	}
}

func (s *SshClient) goSafe(scope string, fn func()) {
	go func() {
		defer s.recoverPanic(scope)
		fn()
	}()
}

func (s *SshClient) runConnection(scope string, fn func()) {
	s.connections.Add(1)
	s.goSafe(scope, func() {
		defer s.connections.Done()
		fn()
	})
}

func (s *SshClient) closeTransport() error {
	s.mu.Lock()
	transport := s.transport
	s.transport = nil
	s.mu.Unlock()
	if transport == nil {
		return nil
	}
	return transport.Close()
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

func (s *SshClient) createNewConnection(ctx context.Context) (string, error) {
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

func (s *SshClient) httpTunnel(src net.Conn, localEndpoint string) {
	s.httpTunnelReverseProxy(src, localEndpoint)
}

func writeLocalServerUnavailable(writer io.Writer, localEndpoint string) error {
	htmlContent := []byte(utils.LocalServerNotOnline(localEndpoint))
	response := &http.Response{
		Status:        "503 Service Unavailable",
		StatusCode:    http.StatusServiceUnavailable,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        make(http.Header),
		ContentLength: int64(len(htmlContent)),
		Body:          io.NopCloser(bytes.NewReader(htmlContent)),
	}
	response.Header.Set("Content-Type", "text/html")
	response.Header.Set("X-Portr-Error", "true")
	response.Header.Set("X-Portr-Error-Reason", "local-server-not-online")
	return response.Write(writer)
}

func (s *SshClient) httpTunnelReverseProxy(src net.Conn, localEndpoint string) {
	defer src.Close()

	target := &url.URL{
		Scheme: "http",
		Host:   localEndpoint,
	}

	transport := &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		ForceAttemptHTTP2: false,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			d := net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
			return d.DialContext(ctx, network, localEndpoint)
		},
		MaxIdleConns:        8,
		MaxIdleConnsPerHost: 8,
		IdleConnTimeout:     90 * time.Second,
	}
	defer transport.CloseIdleConnections()

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = transport

	defaultDirector := proxy.Director
	proxy.Director = func(request *http.Request) {
		host := request.Host
		defaultDirector(request)
		request.Host = host
	}

	proxy.ModifyResponse = func(response *http.Response) error {
		logData, ok := response.Request.Context().Value(requestLogContextKey{}).(*requestLogData)
		if !ok || logData == nil || logData.request == nil {
			return nil
		}

		responseSnapshot := *response
		responseSnapshot.Header = response.Header.Clone()
		responseSnapshot.Body = nil
		responseSnapshot.Request = nil
		responseCapture := &bodyCapture{}
		response.Body = &capturingReadCloser{
			ReadCloser: response.Body,
			capture:    responseCapture,
			onDone: func() {
				durationMs := time.Since(logData.startTime).Milliseconds()
				s.submitCapture(httpCaptureTask{
					id:           logData.id,
					request:      logData.request,
					requestBody:  logData.body.Bytes(),
					response:     &responseSnapshot,
					responseBody: responseCapture.Bytes(),
					durationMs:   durationMs,
					bytesIn:      logData.body.Size(),
					bytesOut:     responseCapture.Size(),
				})
			},
		}
		return nil
	}

	proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, err error) {
		if s.config.Debug {
			s.logDebug("HTTP reverse proxy failed", err)
		}

		htmlContent := utils.LocalServerNotOnline(localEndpoint)
		writer.Header().Set("X-Portr-Error", "true")
		writer.Header().Set("X-Portr-Error-Reason", "local-server-not-online")
		writer.Header().Set("Content-Type", "text/html")
		writer.WriteHeader(http.StatusServiceUnavailable)
		_, _ = writer.Write([]byte(htmlContent))
	}

	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("X-Portr-Ping-Request") == "true" {
			writer.WriteHeader(http.StatusOK)
			return
		}

		if isWebSocketUpgrade(request) {
			hijacker, ok := writer.(http.Hijacker)
			if !ok {
				http.Error(writer, "websocket proxy unsupported", http.StatusInternalServerError)
				return
			}

			conn, rw, err := hijacker.Hijack()
			if err != nil {
				if s.config.Debug {
					s.logDebug("Failed to hijack websocket connection", err)
				}
				return
			}
			defer conn.Close()

			if err := s.handleWebSocketRequest(conn, rw.Reader, rw.Writer, request, localEndpoint); err != nil && s.config.Debug {
				s.logDebug("Failed to proxy websocket request", err)
			}
			return
		}

		requestForLog := request.Clone(context.Background())
		requestForLog.Header = request.Header.Clone()
		requestForLog.Host = request.Host
		if request.URL != nil {
			clonedURL := *request.URL
			requestForLog.URL = &clonedURL
		}

		requestCapture := &bodyCapture{}
		if request.Body != nil {
			request.Body = &capturingReadCloser{
				ReadCloser: request.Body,
				capture:    requestCapture,
				onDone:     func() {},
			}
		}

		logCtx := context.WithValue(request.Context(), requestLogContextKey{}, &requestLogData{
			id:        ulid.Make().String(),
			request:   requestForLog,
			body:      requestCapture,
			startTime: time.Now(),
		})

		proxy.ServeHTTP(writer, request.WithContext(logCtx))
	})

	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 15 * time.Second,
	}

	listener := &singleConnListener{conn: src}
	err := server.Serve(listener)
	if err != nil && err != net.ErrClosed {
		if s.config.Debug {
			s.logDebug("Reverse proxy tunnel closed with error", err)
		}
	}
}

func (s *SshClient) logHttpRequest(
	id string,
	request *http.Request,
	requestBody []byte,
	response *http.Response,
	responseBody []byte,
	durationMs int64,
) {
	s.logHttpRequestSized(
		id,
		request,
		requestBody,
		response,
		responseBody,
		durationMs,
		int64(len(requestBody)),
		int64(len(responseBody)),
	)
}

func (s *SshClient) logHttpRequestSized(
	id string,
	request *http.Request,
	requestBody []byte,
	response *http.Response,
	responseBody []byte,
	durationMs int64,
	bytesIn int64,
	bytesOut int64,
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
		DurationMs:         durationMs,
		BytesIn:            bytesIn,
		BytesOut:           bytesOut,
		Protocol:           request.Proto,
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
		if s.config.Tunnel.Type == constants.Stub && s.config.Tunnel.Subdomain != "" {
			tunnelName = s.config.Tunnel.Subdomain
		} else {
			tunnelName = fmt.Sprintf("%d", s.config.Tunnel.Port)
		}
	}

	if !s.config.EnableRequestLogging {
		return
	}

	if s.tui != nil {
		s.tui.Send(tui.AddLogMsg{
			Time:   req.LoggedAt.Local().Format("15:04:05"),
			Name:   tunnelName,
			Method: req.Method,
			Status: req.ResponseStatusCode,
			URL:    req.Url,
		})
	} else if !s.config.DisableTerminalLogs {
		fmt.Printf("[%s] %s %s → %d\n",
			req.LoggedAt.Local().Format("15:04:05"),
			req.Method,
			req.Url,
			req.ResponseStatusCode)
	}
}

func (s *SshClient) tcpTunnel(src, dst net.Conn) {
	defer func() {
		_ = src.Close()
		_ = dst.Close()
	}()

	results := make(chan error, 2)
	copyDirection := func(target, source net.Conn) {
		_, err := io.Copy(target, source)
		if closeWriter, ok := target.(interface{ CloseWrite() error }); ok {
			_ = closeWriter.CloseWrite()
		}
		results <- err
	}

	go copyDirection(dst, src)
	go copyDirection(src, dst)

	firstErr := <-results
	if firstErr != nil && !errors.Is(firstErr, net.ErrClosed) {
		_ = src.Close()
		_ = dst.Close()
	}
	<-results
}
