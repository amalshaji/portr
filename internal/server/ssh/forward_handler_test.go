package sshd

import (
	"context"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	sshserver "github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

type fakeSSHContext struct {
	context.Context
	sync.Mutex
	user   string
	values map[any]any
}

func (c *fakeSSHContext) User() string                        { return c.user }
func (c *fakeSSHContext) SessionID() string                   { return "session" }
func (c *fakeSSHContext) ClientVersion() string               { return "client" }
func (c *fakeSSHContext) ServerVersion() string               { return "server" }
func (c *fakeSSHContext) RemoteAddr() net.Addr                { return testNetworkAddr("remote") }
func (c *fakeSSHContext) LocalAddr() net.Addr                 { return testNetworkAddr("local") }
func (c *fakeSSHContext) Permissions() *sshserver.Permissions { return &sshserver.Permissions{} }
func (c *fakeSSHContext) SetValue(key, value any)             { c.values[key] = value }
func (c *fakeSSHContext) Value(key any) any {
	if value, ok := c.values[key]; ok {
		return value
	}
	return c.Context.Value(key)
}

type testNetworkAddr string

func (a testNetworkAddr) Network() string { return string(a) }
func (a testNetworkAddr) String() string  { return string(a) }

func newFakeSSHContext(t *testing.T) (*fakeSSHContext, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	return &fakeSSHContext{
		Context: ctx,
		user:    "connection:secret",
		values: map[any]any{
			sshserver.ContextKeyConn: &gossh.ServerConn{},
		},
	}, cancel
}

func TestForwardHandlerDoesNotRegisterFailedBind(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer occupied.Close()
	_, portString, _ := net.SplitHostPort(occupied.Addr().String())
	port, _ := strconv.Atoi(portString)

	registered := false
	handler := &forwardedTCPHandler{
		onBound: func(sshserver.Context, string, uint32) error {
			registered = true
			return nil
		},
	}
	ctx, cancel := newFakeSSHContext(t)
	defer cancel()
	request := &gossh.Request{
		Type:    "tcpip-forward",
		Payload: gossh.Marshal(&remoteForwardRequest{BindAddr: "127.0.0.1", BindPort: uint32(port)}),
	}
	ok, _ := handler.HandleSSHRequest(ctx, nil, request)
	if ok {
		t.Fatal("expected occupied port bind to fail")
	}
	if registered {
		t.Fatal("failed bind was registered as a backend")
	}
}

func TestForwardHandlerRegistersOnlyAfterBind(t *testing.T) {
	closed := make(chan struct{}, 1)
	handler := &forwardedTCPHandler{
		onBound: func(_ sshserver.Context, host string, port uint32) error {
			probe, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(int(port))))
			if err == nil {
				_ = probe.Close()
				t.Fatal("forward callback ran before the port was bound")
			}
			return nil
		},
		onClosed: func(sshserver.Context, string, uint32) {
			closed <- struct{}{}
		},
	}
	ctx, cancel := newFakeSSHContext(t)
	defer cancel()
	request := &gossh.Request{
		Type:    "tcpip-forward",
		Payload: gossh.Marshal(&remoteForwardRequest{BindAddr: "127.0.0.1"}),
	}
	ok, response := handler.HandleSSHRequest(ctx, nil, request)
	if !ok {
		t.Fatal("expected dynamic bind to succeed")
	}
	var success remoteForwardSuccess
	if err := gossh.Unmarshal(response, &success); err != nil || success.BindPort == 0 {
		t.Fatalf("invalid bind response port=%d err=%v", success.BindPort, err)
	}

	cancelRequest := &gossh.Request{
		Type: "cancel-tcpip-forward",
		Payload: gossh.Marshal(&remoteForwardCancelRequest{
			BindAddr: "127.0.0.1",
			BindPort: success.BindPort,
		}),
	}
	if canceled, _ := handler.HandleSSHRequest(ctx, nil, cancelRequest); !canceled {
		t.Fatal("expected forward cancellation to succeed")
	}
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("forward close callback was not invoked")
	}
}

func TestForwardHandlerCancelIsIdempotent(t *testing.T) {
	handler := &forwardedTCPHandler{}
	request := &gossh.Request{
		Type: "cancel-tcpip-forward",
		Payload: gossh.Marshal(&remoteForwardCancelRequest{
			BindAddr: "127.0.0.1",
			BindPort: 29999,
		}),
	}
	ctx, cancel := newFakeSSHContext(t)
	defer cancel()
	if ok, _ := handler.HandleSSHRequest(ctx, nil, request); !ok {
		t.Fatal("canceling an already-closed forward should succeed")
	}
}
