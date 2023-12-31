package ssh

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amalshaji/localport/internal/client/config"
	"github.com/amalshaji/localport/internal/utils"
	"github.com/go-resty/resty/v2"
	"golang.org/x/crypto/ssh"
)

var (
	ErrLocalSetupIncomplete = fmt.Errorf("local setup incomplete")
)

type SshClient struct {
	config   config.ClientConfig
	listener net.Listener
	log      *slog.Logger
}

func New(config config.ClientConfig) *SshClient {
	return &SshClient{
		config:   config,
		listener: nil,
		log:      slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}
}

func (s *SshClient) generateUsername() string {
	return fmt.Sprintf("%s:%s", s.config.SecretKey, s.config.Tunnel.Subdomain)
}

func (s *SshClient) getSshSigner() ssh.Signer {
	homeDir, _ := os.UserHomeDir()
	pemBytes, err := os.ReadFile(homeDir + "/.localport/keys/id_rsa")
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

func (s *SshClient) createNewConnection() error {
	client := resty.New()
	var reqErr struct {
		Message string `json:"message"`
	}

	resp, err := client.R().
		SetHeader("X-Subdomain", s.config.Tunnel.Subdomain).
		SetHeader("X-SecretKey", s.config.SecretKey).
		SetError(&reqErr).
		Post(s.config.GetServerAddr() + "/api/connection/create")
	if err != nil {
		return err
	}
	if resp.StatusCode() != 201 {
		return fmt.Errorf(reqErr.Message)
	}
	return nil
}

func (s *SshClient) startListenerForClient() error {
	if err := s.createNewConnection(); err != nil {
		return err
	}

	signer := s.getSshSigner()

	sshConfig := &ssh.ClientConfig{
		User: s.generateUsername(),
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

	localEndpoint := s.config.Tunnel.GetAddr() // Local address to forward to

	randomPorts := utils.GenerateRandomHttpPorts()[:100]

	// try to connect to 100 random ports (too much??)
	for _, port := range randomPorts {
		s.listener, err = sshClient.Listen("tcp", "localhost:"+fmt.Sprint(port))
		if err == nil {
			break
		}
	}

	if s.listener == nil {
		log.Fatal("failed to listen on remote endpoint")
	}

	defer s.listener.Close()

	s.log.Info("listening on remote endpoint", "remote", s.config.GetAddr(), "local", s.config.Tunnel.GetAddr())

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
			// change this to beautiful template
			remoteConn.Write([]byte(utils.LocalServerNotOnline(localEndpoint)))
			remoteConn.Close()
			continue
		}

		// Copy data between the remote and local connections
		go tunnel(remoteConn, localConn)
		go tunnel(localConn, remoteConn)
	}

	return nil
}

func tunnel(src, dst net.Conn) {
	defer src.Close()
	defer dst.Close()

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

func (s *SshClient) Start(ctx context.Context) {
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
