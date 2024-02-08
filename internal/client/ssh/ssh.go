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
	"os/signal"
	"syscall"
	"time"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/amalshaji/portr/internal/client/db"
	"github.com/amalshaji/portr/internal/constants"
	"github.com/amalshaji/portr/internal/utils"
	"github.com/go-resty/resty/v2"
	"gorm.io/datatypes"

	"github.com/oklog/ulid/v2"
	"golang.org/x/crypto/ssh"
)

var (
	ErrLocalSetupIncomplete = fmt.Errorf("local setup incomplete")
)

type SshClient struct {
	config    config.ClientConfig
	listener  net.Listener
	log       *slog.Logger
	db        *db.Db
	connected chan bool
}

func New(config config.ClientConfig, db *db.Db) *SshClient {
	return &SshClient{
		config:    config,
		listener:  nil,
		log:       slog.New(slog.NewTextHandler(os.Stdout, nil)),
		db:        db,
		connected: make(chan bool),
	}
}

func (s *SshClient) getSshSigner() ssh.Signer {
	homeDir, _ := os.UserHomeDir()
	pemBytes, err := os.ReadFile(homeDir + "/.portr/keys/id_rsa")
	if err != nil {
		if s.config.Debug {
			s.log.Error("failed to read ssh key", "error", err)
		}
		log.Fatal(ErrLocalSetupIncomplete)
	}

	signer, err := ssh.ParsePrivateKey(pemBytes)
	if err != nil {
		if s.config.Debug {
			s.log.Error("failed to parse ssh key", "error", err)
		}
		log.Fatal(ErrLocalSetupIncomplete)
	}
	return signer
}

func (s *SshClient) createNewConnection() (string, error) {
	client := resty.New()
	var reqErr struct {
		Message string `json:"message"`
	}
	var response struct {
		ConnectionId string `json:"connectionId"`
	}

	request := client.R().
		SetHeader("X-Connection-Type", string(s.config.Tunnel.Type)).
		SetHeader("X-SecretKey", s.config.SecretKey).
		SetError(&reqErr).
		SetResult(&response)

	if s.config.Tunnel.Type == constants.Http {
		request = request.SetHeader("X-Subdomain", s.config.Tunnel.Subdomain)
	}

	resp, err := request.Post(s.config.GetServerAddr() + "/api/connection/create")
	if err != nil {
		return "", err
	}

	if resp.StatusCode() != 200 {
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

	signer := s.getSshSigner()

	sshConfig := &ssh.ClientConfig{
		User: connectionId,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
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

	s.connected <- true

	fmt.Println()

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

	responseHeaders := make(map[string][]string)
	for key, values := range response.Header {
		responseHeaders[key] = values
	}

	responseHeadersBytes, err := json.Marshal(responseHeaders)
	if err != nil {
		if s.config.Debug {
			s.log.Error("failed to marshal request headers", "error", err)
		}
		return
	}

	req := db.Request{
		ID:              id,
		Url:             request.URL.String(),
		Subdomain:       s.config.Tunnel.Subdomain,
		Localport:       s.config.Tunnel.Port,
		Method:          request.Method,
		Headers:         datatypes.JSON(requestHeadersBytes),
		Body:            requestBody,
		ResponseStatus:  response.StatusCode,
		ResponseHeaders: datatypes.JSON(responseHeadersBytes),
		ResponseBody:    responseBody,
	}
	result := s.db.Conn.Create(&req)
	if result.Error != nil {
		if s.config.Debug {
			s.log.Error("failed to log request", "error", result.Error)
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
	err := s.listener.Close()
	if err != nil {
		return err
	}
	s.log.Info("stopping tunnel client server")
	return nil
}

func (s *SshClient) Start(_ context.Context) {
	utils.ShowLoading("Tunnel connecting", s.connected)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := s.startListenerForClient(); err != nil {
			log.Fatalf("failed to establish tunnel connection: error=%v\n", err)
		}
	}()

	<-done
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil {
		if s.config.Debug {
			s.log.Error("failed to stop tunnel client", "error", err)
		}
	}
}
