package sshd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
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
	config   *config.SshConfig
	proxy    *proxy.Proxy
	service  *service.Service
	server   *ssh.Server
	leaseMu  sync.Mutex
	forwards map[string]*connectionLeases
}

type forwardLease struct {
	connectionType string
	subdomain      string
}

type connectionLeases struct {
	mu       sync.Mutex
	forwards map[string]forwardLease
}

type reservedConnectionContextKey struct{}

func New(config *config.SshConfig, proxy *proxy.Proxy, service *service.Service) *SshServer {
	return &SshServer{
		config:   config,
		proxy:    proxy,
		service:  service,
		forwards: make(map[string]*connectionLeases),
	}
}

func (s *SshServer) GetServerAddr() string {
	return ":" + fmt.Sprint(s.config.Port)
}

func (s *SshServer) GetReservedConnectionFromSshContext(ctx ssh.Context) (*db.Connection, error) {
	if cached, ok := ctx.Value(reservedConnectionContextKey{}).(*db.Connection); ok && cached != nil {
		return cached, nil
	}
	reservedConnection, err := s.authenticateConnection(ctx)
	if err != nil {
		return nil, err
	}
	ctx.SetValue(reservedConnectionContextKey{}, reservedConnection)
	return reservedConnection, nil
}

func (s *SshServer) authenticateConnection(ctx ssh.Context) (*db.Connection, error) {
	connectionID, secretKey, err := connectionCredentials(ctx)
	if err != nil {
		return nil, err
	}

	reservedConnection, err := s.service.GetReservedConnectionById(ctx, connectionID)
	if err != nil {
		log.Error("Failed to get reserved connection", "error", err)
		return nil, fmt.Errorf("failed to get reserved connection")
	}

	if reservedConnection.CreatedBy.SecretKey != secretKey {
		log.Error("Connection not created by the user", "connection_id", connectionID)
		return nil, fmt.Errorf("connection not created by the user")
	}

	return reservedConnection, nil
}

func connectionCredentials(ctx ssh.Context) (string, string, error) {
	userSplit := strings.SplitN(ctx.User(), ":", 2)
	if len(userSplit) != 2 {
		return "", "", fmt.Errorf("invalid user format")
	}
	return userSplit[0], userSplit[1], nil
}

func (s *SshServer) activateForward(ctx ssh.Context, host string, port uint32) error {
	reservedConnection, err := s.GetReservedConnectionFromSshContext(ctx)
	if err != nil {
		return err
	}

	backend := forwardKey(host, port)
	lease := forwardLease{connectionType: reservedConnection.Type}
	switch reservedConnection.Type {
	case string(constants.Tcp):
	case string(constants.Http):
		if reservedConnection.Subdomain == nil || *reservedConnection.Subdomain == "" {
			return fmt.Errorf("http connection has no subdomain")
		}
		lease.subdomain = *reservedConnection.Subdomain
	default:
		return fmt.Errorf("unsupported connection type %q", reservedConnection.Type)
	}

	connectionLeases := s.leasesForConnection(reservedConnection.ID)
	connectionLeases.mu.Lock()
	defer connectionLeases.mu.Unlock()
	if _, exists := connectionLeases.forwards[backend]; exists {
		return nil
	}

	firstForward := len(connectionLeases.forwards) == 0
	if reservedConnection.Type == string(constants.Tcp) {
		if !firstForward {
			return fmt.Errorf("tcp connection already has an active forward")
		}
		if err := s.service.MarkTCPConnectionAsActive(ctx, reservedConnection.ID, port); err != nil {
			return err
		}
	} else {
		if err := s.proxy.AddBackend(lease.subdomain, backend); err != nil {
			return err
		}
		if firstForward {
			if err := s.service.MarkConnectionAsActive(ctx, reservedConnection.ID); err != nil {
				_ = s.proxy.RemoveBackend(lease.subdomain, backend)
				return err
			}
		}
	}

	connectionLeases.forwards[backend] = lease
	return nil
}

func (s *SshServer) leasesForConnection(connectionID string) *connectionLeases {
	s.leaseMu.Lock()
	defer s.leaseMu.Unlock()
	leases := s.forwards[connectionID]
	if leases == nil {
		leases = &connectionLeases{forwards: make(map[string]forwardLease)}
		s.forwards[connectionID] = leases
	}
	return leases
}

func (s *SshServer) closeForward(ctx ssh.Context, host string, port uint32) {
	connectionID, _, err := connectionCredentials(ctx)
	if err != nil {
		return
	}
	backend := forwardKey(host, port)

	s.leaseMu.Lock()
	connectionLeases := s.forwards[connectionID]
	s.leaseMu.Unlock()
	if connectionLeases == nil {
		return
	}
	connectionLeases.mu.Lock()
	defer connectionLeases.mu.Unlock()
	lease, exists := connectionLeases.forwards[backend]
	if !exists {
		return
	}

	if lease.connectionType != string(constants.Tcp) {
		if err := s.proxy.RemoveBackend(lease.subdomain, backend); err != nil {
			log.Error("Failed to remove tunnel backend", "connection_id", connectionID, "backend", backend, "error", err)
		}
	}
	delete(connectionLeases.forwards, backend)
	if len(connectionLeases.forwards) != 0 {
		return
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.service.MarkConnectionAsClosed(closeCtx, connectionID); err != nil {
		log.Error("Failed to mark connection as closed", "connection_id", connectionID, "error", err)
	}
}

func (s *SshServer) Start() {
	srv := s.Build()

	hostKeyOption, err := loadHostKey(s.config.HostKey)
	if err != nil {
		log.Fatal("Failed to load host key", "error", err)
	}
	if hostKeyOption != nil {
		hostKeyOption(srv)
	}

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
	forwardHandler := &forwardedTCPHandler{
		onBound:  s.activateForward,
		onClosed: s.closeForward,
	}

	requestHandlers := map[string]ssh.RequestHandler{
		"tcpip-forward":        forwardHandler.HandleSSHRequest,
		"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
	}

	// Respond OK to ssh application keepalive requests (global request)
	requestHandlers["keepalive@openssh.com"] = func(ssh.Context, *ssh.Server, *gossh.Request) (bool, []byte) {
		return true, nil
	}

	server := &ssh.Server{
		Addr: s.GetServerAddr(),
		Handler: ssh.Handler(func(sh ssh.Session) {
			<-sh.Context().Done()
		}),
		RequestHandlers: requestHandlers,
		PasswordHandler: func(ctx ssh.Context, _ string) bool {
			reservedConnection, err := s.authenticateConnection(ctx)
			if err == nil {
				ctx.SetValue(reservedConnectionContextKey{}, reservedConnection)
			}
			return err == nil
		},
	}

	return server
}
