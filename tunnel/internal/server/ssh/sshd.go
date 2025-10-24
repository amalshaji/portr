package sshd

import (
	"context"
	"errors"
	"fmt"

	"strings"
	"time"

	"github.com/amalshaji/portr/internal/constants"
	"github.com/charmbracelet/log"

	"github.com/amalshaji/portr/internal/server/config"
	"github.com/amalshaji/portr/internal/server/db"
	"github.com/amalshaji/portr/internal/server/proxy"
	"github.com/amalshaji/portr/internal/server/service"
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

type SshServer struct {
	config  *config.SshConfig
	proxy   *proxy.Proxy
	service *service.Service
	server  *ssh.Server
}

func New(config *config.SshConfig, proxy *proxy.Proxy, service *service.Service) *SshServer {
	return &SshServer{
		config:  config,
		proxy:   proxy,
		service: service,
	}
}

func (s *SshServer) GetServerAddr() string {
	return ":" + fmt.Sprint(s.config.Port)
}

func (s *SshServer) GetReservedConnectionFromSshContext(ctx ssh.Context) (*db.Connection, error) {
	userSplit := strings.Split(ctx.User(), ":")
	if len(userSplit) != 2 {
		return nil, fmt.Errorf("invalid user format")
	}

	connectionId, secretKey := userSplit[0], userSplit[1]

	reservedConnection, err := s.service.GetReservedConnectionById(ctx, connectionId)
	if err != nil {
		log.Error("Failed to get reserved connection", "error", err)
		return nil, fmt.Errorf("failed to get reserved connection")
	}

	if reservedConnection.CreatedBy.SecretKey != secretKey {
		log.Error("Connection not created by the user", "connection_id", connectionId)
		return nil, fmt.Errorf("connection not created by the user")
	}

	return reservedConnection, nil
}

func (s *SshServer) Start() {
	// Build server and start listening
	srv := s.Build()
	s.server = srv

	log.Info("Starting SSH server", "port", s.GetServerAddr())

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Fatal("Failed to start SSH server", "error", err)
	}
}

func (s *SshServer) Shutdown(_ context.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer func() { cancel() }()

	if err := s.server.Shutdown(ctx); err != nil {
		log.Error("Failed to stop SSH server", "error", err)
		return
	}

	log.Info("Stopped SSH server")
}

// Build constructs the ssh.Server with all handlers, without starting it.
func (s *SshServer) Build() *ssh.Server {
	forwardHandler := &ssh.ForwardedTCPHandler{}

	requestHandlers := map[string]ssh.RequestHandler{
		"tcpip-forward":        forwardHandler.HandleSSHRequest,
		"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
	}

	// Respond OK to ssh application keepalive requests (global request)
	requestHandlers["keepalive@openssh.com"] = func(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
		return true, nil
	}

	server := &ssh.Server{
		Addr: s.GetServerAddr(),
		Handler: ssh.Handler(func(sh ssh.Session) {
			select {}
		}),
		ReversePortForwardingCallback: ssh.ReversePortForwardingCallback(func(ctx ssh.Context, host string, port uint32) bool {
			reservedConnection, err := s.GetReservedConnectionFromSshContext(ctx)
			if err != nil {
				return false
			}

			proxyTarget := fmt.Sprintf("%s:%d", host, port)

			if reservedConnection.Type == string(constants.Tcp) {
				err = s.service.AddPortToConnection(ctx, reservedConnection.ID, port)
				if err != nil {
					log.Error("Failed to add port to connection", "connection_id", reservedConnection.ID, "port", port, "error", err)
					return false
				}
			} else {
				// Add this backend to the subdomain's pool
				err = s.proxy.AddBackend(*reservedConnection.Subdomain, proxyTarget)
				if err != nil {
					log.Error("Failed to add backend", "connection_id", reservedConnection.ID, "subdomain", *reservedConnection.Subdomain, "backend", proxyTarget, "error", err)
					return false
				}
			}

			err = s.service.MarkConnectionAsActive(ctx, reservedConnection.ID)
			if err != nil {
				log.Error("Failed to mark connection as active", "connection_id", reservedConnection.ID, "error", err)
				return false
			}

			go func() {
				<-ctx.Done()

				err = s.service.MarkConnectionAsClosed(context.Background(), reservedConnection.ID)
				if err != nil {
					log.Error("Failed to mark connection as closed", "connection_id", reservedConnection.ID, "error", err)
				}

				if reservedConnection.Type == string(constants.Http) {
					// Remove only this backend from the pool
					backend := fmt.Sprintf("%s:%d", host, port)
					err := s.proxy.RemoveBackend(*reservedConnection.Subdomain, backend)
					if err != nil {
						log.Error("Failed to remove backend", "connection_id", reservedConnection.ID, "subdomain", *reservedConnection.Subdomain, "backend", backend, "error", err)
					}
				}
			}()

			return true
		}),
		RequestHandlers: requestHandlers,
		PasswordHandler: func(ctx ssh.Context, password string) bool {
			_, err := s.GetReservedConnectionFromSshContext(ctx)
			return err == nil
		},
	}

	return server
}
