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
	"time"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/client/db"
	"github.com/amalshaji/portr/internal/constants"
	"github.com/amalshaji/portr/internal/utils"
	"github.com/charmbracelet/log"
	"github.com/go-resty/resty/v2"
	"github.com/labstack/gommon/color"
	"gorm.io/datatypes"

	"github.com/oklog/ulid/v2"
	"golang.org/x/crypto/ssh"
)

var (
	ErrLocalSetupIncomplete = fmt.Errorf("local setup incomplete")
)

type SshClient struct {
	config   config.ClientConfig
	listener net.Listener
	db       *db.Db
	client   *ssh.Client
}

func New(config config.ClientConfig, db *db.Db) *SshClient {
	return &SshClient{
		config:   config,
		listener: nil,
		db:       db,
		client:   nil,
	}
}

func (s *SshClient) createNewConnection() (string, error) {
	client := resty.New()
	var reqErr struct {
		Message string `json:"message"`
	}
	var response struct {
		ConnectionId string `json:"connection_id"`
	}

	payload := map[string]any{
		"connection_type": string(s.config.Tunnel.Type),
		"secret_key":      s.config.SecretKey,
		"subdomain":       nil,
	}
	request := client.R().
		SetError(&reqErr).
		SetResult(&response)

	if s.config.Tunnel.Type == constants.Http {
		payload["subdomain"] = s.config.Tunnel.Subdomain
	}

	resp, err := request.SetBody(payload).Post(s.config.GetServerAddr() + "/api/v1/connections/")

	if err != nil {
		return "", err
	}

	if resp.StatusCode() != 200 {
		if s.config.Debug {
			log.Error("Failed to create new connection", "error", reqErr)
		}
		return "", fmt.Errorf("server error: %s", reqErr.Message)
	}
	return response.ConnectionId, nil
}

func (s *SshClient) startListenerForClient() error {
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

	s.client, err = ssh.Dial("tcp", s.config.SshUrl, sshConfig)
	if err != nil {
		if s.config.Debug {
			log.Error("Failed to connect to ssh server", "error", err)
		}
		return err
	}

	localEndpoint := s.config.Tunnel.GetLocalAddr() // Local address to forward to

	tunnelType := s.config.Tunnel.Type

	var randomPorts []int
	if tunnelType == constants.Http {
		randomPorts = utils.GenerateRandomHttpPorts()
	} else {
		randomPorts = utils.GenerateRandomTcpPorts()
	}

	var remotePort int

	// try to connect to 10 random ports
	for _, port := range randomPorts {
		s.listener, err = s.client.Listen("tcp", "0.0.0.0:"+fmt.Sprint(port))
		remotePort = port
		if err == nil {
			break
		}
	}

	if s.listener == nil {
		return fmt.Errorf("failed to listen on remote endpoint")
	}

	s.config.Tunnel.RemotePort = remotePort

	defer s.listener.Close()

	fmt.Printf(
		"🎉 Tunnel connected: %s -> 🌐 -> %s\n",
		s.config.GetTunnelAddr(),
		s.config.Tunnel.GetLocalAddr(),
	)

	for {
		// Accept incoming connections on the remote port
		remoteConn, err := s.listener.Accept()
		if err != nil {
			if s.config.Debug {
				log.Error("Failed to accept connection", "error", err)
			}
			break
		}

		// Connect to the local endpoint
		localConn, err := net.Dial("tcp", localEndpoint)
		if err != nil {
			// serve local html if the local server is not available
			// change this to a beautiful template
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
			log.Error("Failed to read request", "error", err)
		}
		return
	}

	// Early return for health check requests with a direct response
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

	// read and replace request body
	requestBody, err := io.ReadAll(request.Body)
	if err != nil {
		if s.config.Debug {
			log.Error("Failed to read request body", "error", err)
		}
		return
	}
	defer request.Body.Close()
	request.Body = io.NopCloser(bytes.NewBuffer(requestBody))

	err = request.Write(dstWriter)
	if err != nil {
		if s.config.Debug {
			log.Error("Failed to tunnel request to local", "error", err)
		}
		return
	}
	dstWriter.Flush()

	response, err := http.ReadResponse(dstReader, request)
	if err != nil {
		if s.config.Debug {
			log.Error("Failed to read response", "error", err)
		}
		return
	}

	// read and replace response body
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		if s.config.Debug {
			log.Error("Failed to read response body", "error", err)
		}
		return
	}
	defer response.Body.Close()
	response.Body = io.NopCloser(bytes.NewBuffer(responseBody))

	err = response.Write(srcWriter)
	if err != nil {
		if s.config.Debug {
			log.Error("Failed to write response to remote", "error", err)
		}
		return
	}
	srcWriter.Flush()

	if response.StatusCode == http.StatusSwitchingProtocols {
		// handle websocket
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
			log.Error("Failed to marshal request headers", "error", err)
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
			log.Error("Failed to marshal request headers", "error", err)
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
			log.Error("Failed to log request", "error", result.Error)
		}
		return
	}

	if s.config.EnableRequestLogging {
		fmt.Printf(
			"%s [%d] %-6s %d %s\n",
			req.LoggedAt.Local().Format("2006-01-02 15:04:05"),
			req.Localport,
			req.Method,
			req.ResponseStatusCode,
			req.Url,
		)
	}
}

func (s *SshClient) tcpTunnel(src, dst net.Conn) {
	defer src.Close()
	defer dst.Close()

	go io.Copy(dst, src)
	io.Copy(src, dst)
}

func (s *SshClient) Shutdown(ctx context.Context) error {
	if s.listener == nil {
		return nil
	}

	err := s.listener.Close()
	if err != nil {
		return err
	}
	log.Info("Stopped tunnel connection", "address", s.config.GetTunnelAddr())
	return nil
}

func (s *SshClient) StartHealthCheck(ctx context.Context) {
	ticker := time.Tick(time.Duration(s.config.HealthCheckInterval) * time.Second)
	retryAttempts := 0

	var err error

	for range ticker {
		retryAttempts++
		if retryAttempts > s.config.HealthCheckMaxRetries {
			fmt.Printf(color.Red("Failed to reconnect to tunnel after %d attempts\n"), retryAttempts)
			os.Exit(1)
		}

		err = s.HealthCheck()
		if err == nil {
			retryAttempts = 0
			continue
		}

		if s.config.Debug {
			log.Error("Health check failed", "error", err)
		}

		fmt.Printf(color.Yellow("Tunnel %s is not healthy 🪫 attempting to reconnect\n"), s.config.GetTunnelAddr())

		err = s.Reconnect()
		if err != nil {
			if s.config.Debug {
				log.Error("Failed to reconnect to ssh tunnel", "error", err, "attempts", retryAttempts)
			}
		} else {
			retryAttempts = 0
		}

	}
}

func (s *SshClient) Start(ctx context.Context) {
	fmt.Printf("🌍 Starting tunnel connection for :%d\n", s.config.Tunnel.Port)

	errChan := make(chan error, 1)

	go func() {
		if err := s.startListenerForClient(); err != nil {
			errChan <- err
		}
	}()

	// Wait for either an error or successful connection
	select {
	case err := <-errChan:
		fmt.Println(color.Red(err))
		os.Exit(1)
	case <-time.After(5 * time.Second):
		// If no error after 2 seconds, assume connection is successful
		// Start the health check routine for http connections
		if s.config.Tunnel.Type == constants.Http {
			s.StartHealthCheck(ctx)
		}
	}
}

func (s *SshClient) Reconnect() error {
	if s.client != nil {
		if err := s.client.Close(); err != nil {
			if s.config.Debug {
				log.Error("Failed to close client", "error", err)
			}
		}
		s.client = nil
	}

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			if s.config.Debug {
				log.Error("Failed to close listener", "error", err)
			}
		}
		s.listener = nil
	}

	// Channel to receive errors from the goroutine
	errChan := make(chan error, 1)

	// Start the listener in a goroutine
	go func() {
		if err := s.startListenerForClient(); err != nil {
			errChan <- err
		}
	}()

	// Wait for either an error or successful connection
	select {
	case err := <-errChan:
		return err
	case <-time.After(5 * time.Second):
		return nil
	}
}

func (s *SshClient) HealthCheck() error {
	// Make HTTP request to tunnel address with special header
	client := resty.New().
		SetTimeout(5 * time.Second)

	resp, err := client.R().
		SetHeader("X-Portr-Ping-Request", "true").
		Get(s.config.GetTunnelAddr())

	if err != nil {
		if s.config.Debug {
			log.Error("Health check failed, attempting to reconnect", "error", err)
		}
		return err
	}

	portrError := resp.Header().Get("X-Portr-Error")
	portrErrorReason := resp.Header().Get("X-Portr-Error-Reason")

	// Fix it later to resolve to connection-lost
	if portrError == "true" && (portrErrorReason == "connection-lost" || portrErrorReason == "unregistered-subdomain") {
		return fmt.Errorf("unhealthy tunnel")
	}
	return nil
}
