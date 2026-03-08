package server_test

import (
	"testing"

	serverConfig "github.com/amalshaji/portr/internal/server/config"
	sshd "github.com/amalshaji/portr/internal/server/ssh"
)

func TestSSHServer_KeepaliveHandlerRegistered(t *testing.T) {
	cfg := &serverConfig.SshConfig{Host: "localhost", Port: 0}
	srv := sshd.New(cfg, nil, nil)

	server := srv.Build()
	if server == nil {
		t.Fatalf("expected server to be built")
	}

	handler, ok := server.RequestHandlers["keepalive@openssh.com"]
	if !ok {
		t.Fatalf("keepalive@openssh.com handler not registered")
	}

	// Invoke handler; it should acknowledge
	okResp, _ := handler(nil, server, nil)
	if !okResp {
		t.Fatalf("keepalive handler did not acknowledge request")
	}
}
