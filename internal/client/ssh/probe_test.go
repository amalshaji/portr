package ssh

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	gssh "github.com/gliderlabs/ssh"
	"golang.org/x/crypto/ssh"

	clientcfg "github.com/amalshaji/portr/internal/clientconfig"
)

type probeServer struct {
	addr        string
	mu          sync.Mutex
	seenUser    string
	forwardSeen bool
}

func startProbeServer(t *testing.T, allowAuth bool) *probeServer {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	probe := &probeServer{addr: listener.Addr().String()}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}

	server := &gssh.Server{
		Handler: func(s gssh.Session) { _ = s.Exit(0) },
		PasswordHandler: func(ctx gssh.Context, password string) bool {
			probe.mu.Lock()
			probe.seenUser = ctx.User()
			probe.mu.Unlock()
			return allowAuth
		},
		RequestHandlers: map[string]gssh.RequestHandler{
			"tcpip-forward": func(ctx gssh.Context, srv *gssh.Server, req *ssh.Request) (bool, []byte) {
				probe.mu.Lock()
				probe.forwardSeen = true
				probe.mu.Unlock()
				return false, nil
			},
		},
	}
	server.AddHostKey(signer)

	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })

	return probe
}

func probeConfig(addr string) clientcfg.ClientConfig {
	return clientcfg.ClientConfig{
		SshUrl:                          addr,
		ConnectionID:                    "conn-1",
		SecretKey:                       "sk-1",
		InsecureSkipHostKeyVerification: true,
	}
}

func TestProbeSendsConnectionIDAndSecretAsUser(t *testing.T) {
	server := startProbeServer(t, true)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fingerprint, err := Probe(ctx, probeConfig(server.addr))
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if !strings.HasPrefix(fingerprint, "SHA256:") {
		t.Fatalf("expected a SHA256 fingerprint, got %q", fingerprint)
	}

	server.mu.Lock()
	defer server.mu.Unlock()
	if server.seenUser != "conn-1:sk-1" {
		t.Fatalf("expected connection id and secret as the ssh user, got %q", server.seenUser)
	}
}

func TestProbeFailsWhenAuthRejected(t *testing.T) {
	server := startProbeServer(t, false)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := Probe(ctx, probeConfig(server.addr)); err == nil {
		t.Fatal("expected an authentication error")
	} else if !strings.Contains(err.Error(), "unable to authenticate") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProbeDoesNotRequestForward(t *testing.T) {
	// Requesting a forward is what reserves a subdomain and marks a connection
	// active. A diagnostic run must never do that.
	server := startProbeServer(t, true)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := Probe(ctx, probeConfig(server.addr)); err != nil {
		t.Fatalf("probe: %v", err)
	}

	server.mu.Lock()
	defer server.mu.Unlock()
	if server.forwardSeen {
		t.Fatal("probe must not request a port forward")
	}
}

func TestProbeFailsWhenEndpointUnreachable(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := Probe(ctx, probeConfig(addr)); err == nil {
		t.Fatal("expected a dial error")
	}
}
