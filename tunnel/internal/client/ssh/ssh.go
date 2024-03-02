package ssh

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/client/db"
	"github.com/amalshaji/portr/internal/constants"
	"github.com/amalshaji/portr/internal/utils"
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
	log      *slog.Logger
	db       *db.Db
}

func New(config config.ClientConfig, db *db.Db) *SshClient {
	return &SshClient{
		config:   config,
		listener: nil,
		log:      slog.New(slog.NewTextHandler(os.Stdout, nil)),
		db:       db,
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
			s.log.Error("failed to create new connection", "error", reqErr)
		}
		return "", fmt.Errorf(reqErr.Message)
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

	sshClient, err := ssh.Dial("tcp", s.config.SshUrl, sshConfig)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	localEndpoint := s.config.Tunnel.GetLocalAddr() // Local address to forward to

	tunnelType := s.config.Tunnel.Type

	var randomPorts []int
	if tunnelType == constants.Http {
		randomPorts = utils.GenerateRandomHttpPorts()
	} else {
		randomPorts = utils.GenerateRandomTcpPorts()
	}

	var remotePort int

	// try to connect to 100 random ports (too much??)
	for _, port := range randomPorts {
		s.listener, err = sshClient.Listen("tcp", "0.0.0.0:"+fmt.Sprint(port))
		remotePort = port
		if err == nil {
			break
		}
	}

	if s.listener == nil {
		log.Fatal("failed to listen on remote endpoint")
	}

	defer s.listener.Close()

	if tunnelType == constants.Http {
		fmt.Printf(
			"Tunnel connected: %s -> 🌐 -> %s\n",
			s.config.GetHttpTunnelAddr(),
			s.config.Tunnel.GetLocalAddr(),
		)
	} else {
		fmt.Printf(
			"Tunnel connected: %s -> 🌐 -> %s\n",
			s.config.GetTcpTunnelAddr(remotePort),
			s.config.Tunnel.GetLocalAddr(),
		)
	}

	for {
		// Accept incoming connections on the remote port
		remoteConn, err := s.listener.Accept()
		if err != nil {
			if s.config.Debug {
				s.log.Error("failed to accept connection", "error", err)
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
				fmt.Fprintf(remoteConn, "Content-Type: text/html\r\n\r\n")
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
			s.log.Error("failed to read request", "error", err)
		}
		return
	}

	// read and replace request body
	requestBody, err := io.ReadAll(request.Body)
	if err != nil {
		if s.config.Debug {
			s.log.Error("failed to read request body", "error", err)
		}
		return
	}
	defer request.Body.Close()
	request.Body = io.NopCloser(bytes.NewBuffer(requestBody))

	err = request.Write(dstWriter)
	if err != nil {
		if s.config.Debug {
			s.log.Error("failed to tunnel request to local", "error", err)
		}
		return
	}
	dstWriter.Flush()

	response, err := http.ReadResponse(dstReader, request)
	if err != nil {
		if s.config.Debug {
			s.log.Error("failed to read response", "error", err)
		}
		return
	}

	// read and replace response body
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		if s.config.Debug {
			s.log.Error("failed to read response body", "error", err)
		}
		return
	}
	defer response.Body.Close()
	response.Body = io.NopCloser(bytes.NewBuffer(responseBody))

	err = response.Write(srcWriter)
	if err != nil {
		if s.config.Debug {
			s.log.Error("failed to write response to remote", "error", err)
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
		requestHeaders[key] = values
	}

	requestHeadersBytes, err := json.Marshal(requestHeaders)
	if err != nil {
		if s.config.Debug {
			s.log.Error("failed to marshal request headers", "error", err)
		}
		return
	}

	req := db.Request{
		ID:        id,
		Url:       request.URL.String(),
		Subdomain: s.config.Tunnel.Subdomain,
		Localport: s.config.Tunnel.Port,
		Method:    request.Method,
		Headers:   datatypes.JSON(requestHeadersBytes),
		Body:      requestBody,
	}
	result := s.db.Conn.Create(&req)
	if result.Error != nil {
		if s.config.Debug {
			s.log.Error("failed to log request", "error", result.Error)
		}
		return
	}

	responseHeaders := make(map[string][]string)
	for key, values := range request.Header {
		responseHeaders[key] = values
	}

	responseHeadersBytes, err := json.Marshal(responseHeaders)
	if err != nil {
		if s.config.Debug {
			s.log.Error("failed to marshal request headers", "error", err)
		}
		return
	}

	resp := db.Response{
		ID:      id,
		Headers: responseHeadersBytes,
		Body:    responseBody,
	}
	result = s.db.Conn.Create(&resp)
	if result.Error != nil {
		if s.config.Debug {
			s.log.Error("failed to log response", "error", result.Error)
		}
		return
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
	s.log.Info("stopping tunnel client server")
	return nil
}

func (s *SshClient) Start(_ context.Context) {
	fmt.Println("Tunnel connecting...")
	fmt.Println(s.config.Tunnel)

	if err := s.startListenerForClient(); err != nil {
		fmt.Println()
		fmt.Println(color.Red(err))
		os.Exit(1)
	}
}
