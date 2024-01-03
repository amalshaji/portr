package sshd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amalshaji/localport/internal/constants"
	"github.com/amalshaji/localport/internal/server/admin/service"
	"github.com/amalshaji/localport/internal/server/config"
	"github.com/amalshaji/localport/internal/server/proxy"
	"github.com/amalshaji/localport/internal/utils"
	"github.com/gliderlabs/ssh"
)

type SshServer struct {
	config  *config.SshConfig
	log     *slog.Logger
	proxy   *proxy.Proxy
	service *service.Service
}

func New(config *config.SshConfig, proxy *proxy.Proxy, service *service.Service) *SshServer {
	return &SshServer{
		config:  config,
		log:     utils.GetLogger(),
		proxy:   proxy,
		service: service,
	}
}

func (s *SshServer) GetServerAddr() string {
	return ":" + fmt.Sprint(s.config.Port)
}

func (s *SshServer) getSshPublicKey() ssh.PublicKey {
	publicKey, err := os.ReadFile(s.config.KeysDir + "/id_rsa.pub")
	if err != nil {
		log.Fatalf("could not read public key, make sure the keys are present in the %s folder", s.config.KeysDir)
	}
	key, _, _, _, _ := ssh.ParseAuthorizedKey(publicKey)
	return key
}

func (s *SshServer) Start() {
	forwardHandler := &ssh.ForwardedTCPHandler{}

	keyFromDisk := s.getSshPublicKey()

	server := ssh.Server{
		Addr: s.GetServerAddr(),
		Handler: ssh.Handler(func(sh ssh.Session) {
			select {}
		}),
		ReversePortForwardingCallback: ssh.ReversePortForwardingCallback(func(ctx ssh.Context, host string, port uint32) bool {
			connectionId := ctx.User()
			proxyTarget := fmt.Sprintf("%s:%d", host, port)

			reservedConnection, err := s.service.GetReservedOrActiveConnectionById(ctx, connectionId)
			if err != nil {
				s.log.Error("failed to get reserved connection", "error", err)
				return false
			}

			if reservedConnection.Type == string(constants.Tcp) {
				err = s.service.AddPortToConnection(ctx, reservedConnection.ID, port)
				if err != nil {
					s.log.Error("failed to add port to connection", "error", err)
					return false
				}
			}

			err = s.service.MarkConnectionAsActive(ctx, reservedConnection.ID)
			if err != nil {
				s.log.Error("failed to mark connection as active", "error", err)
				return false
			}

			if reservedConnection.Type == string(constants.Http) {
				err = s.proxy.AddRoute(reservedConnection.Subdomain.(string), proxyTarget)
				if err != nil {
					s.log.Error("failed to add route", "error", err)
					return false
				}
			}

			go func() {
				<-ctx.Done()
				err = s.service.MarkConnectionAsClosed(context.Background(), reservedConnection.ID)
				if err != nil {
					s.log.Error("failed to mark connection as closed", "error", err)
				}

				if reservedConnection.Subdomain == string(constants.Http) {
					err := s.proxy.RemoveRoute(reservedConnection.Subdomain.(string))
					if err != nil {
						s.log.Error("failed to remove route", "error", err)
					}
				}
			}()

			return true
		}),

		RequestHandlers: map[string]ssh.RequestHandler{
			"tcpip-forward":        forwardHandler.HandleSSHRequest,
			"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
		},
		PublicKeyHandler: func(ctx ssh.Context, key ssh.PublicKey) bool {
			return ssh.KeysEqual(key, keyFromDisk)
		},
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	s.log.Info("starting SSH server", "port", s.GetServerAddr())

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			s.log.Error("failed to start SSH server", "error", err)
			done <- nil
		}
	}()

	<-done
	s.log.Info("stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := server.Shutdown(ctx); err != nil {
		s.log.Error("failed to stop SSH server", "error", err)
	}
}
