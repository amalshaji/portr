package ssh

import (
	"errors"
	"testing"
	"time"

	"github.com/amalshaji/portr/internal/constants"
)

type requestSenderFunc func(string, bool, []byte) (bool, []byte, error)

func (f requestSenderFunc) SendRequest(name string, wantReply bool, payload []byte) (bool, []byte, error) {
	return f(name, wantReply, payload)
}

func TestCheckKeepAliveRequiresAcknowledgement(t *testing.T) {
	called := false
	err := checkKeepAlive(requestSenderFunc(func(name string, wantReply bool, _ []byte) (bool, []byte, error) {
		called = true
		if name != "keepalive@openssh.com" {
			t.Fatalf("unexpected request %q", name)
		}
		if !wantReply {
			t.Fatal("keepalive must require a reply")
		}
		return true, nil, nil
	}), time.Second)
	if err != nil {
		t.Fatalf("keepalive failed: %v", err)
	}
	if !called {
		t.Fatal("keepalive request was not sent")
	}
}

func TestCheckKeepAliveRejectsMissingAcknowledgement(t *testing.T) {
	err := checkKeepAlive(requestSenderFunc(func(string, bool, []byte) (bool, []byte, error) {
		return false, nil, nil
	}), time.Second)
	if err == nil {
		t.Fatal("expected rejected keepalive to fail")
	}
}

func TestCheckKeepAliveTimesOut(t *testing.T) {
	blocked := make(chan struct{})
	err := checkKeepAlive(requestSenderFunc(func(string, bool, []byte) (bool, []byte, error) {
		<-blocked
		return false, nil, errors.New("closed")
	}), 10*time.Millisecond)
	close(blocked)
	if err == nil || err.Error() != "ssh keepalive timed out" {
		t.Fatalf("expected timeout, got %v", err)
	}
}

func TestHTTPRemotePortsRemainLegacyServerCompatible(t *testing.T) {
	for _, port := range remotePortCandidates(constants.Http) {
		if port == 0 || port < 20000 || port > 30000 {
			t.Fatalf("HTTP candidate port %d is not legacy-server compatible", port)
		}
	}
}
