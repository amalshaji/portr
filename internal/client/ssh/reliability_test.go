package ssh

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/amalshaji/portr/internal/constants"
)

type captureTaskFunc func()

func (f captureTaskFunc) persist(*SshClient) { f() }

type requestSenderFunc func(string, bool, []byte) (bool, []byte, error)

func (f requestSenderFunc) SendRequest(name string, wantReply bool, payload []byte) (bool, []byte, error) {
	return f(name, wantReply, payload)
}

func TestCheckSSHKeepAliveRequiresAcknowledgement(t *testing.T) {
	called := false
	err := checkSSHKeepAlive(requestSenderFunc(func(name string, wantReply bool, _ []byte) (bool, []byte, error) {
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

func TestCheckSSHKeepAliveRejectsMissingAcknowledgement(t *testing.T) {
	err := checkSSHKeepAlive(requestSenderFunc(func(string, bool, []byte) (bool, []byte, error) {
		return false, nil, nil
	}), time.Second)
	if err == nil {
		t.Fatal("expected rejected keepalive to fail")
	}
}

func TestCheckSSHKeepAliveTimesOut(t *testing.T) {
	blocked := make(chan struct{})
	err := checkSSHKeepAlive(requestSenderFunc(func(string, bool, []byte) (bool, []byte, error) {
		<-blocked
		return false, nil, errors.New("closed")
	}), 10*time.Millisecond)
	close(blocked)
	if err == nil || err.Error() != "ssh keepalive timed out" {
		t.Fatalf("expected timeout, got %v", err)
	}
}

func TestBodyCaptureIsBounded(t *testing.T) {
	capture := &bodyCapture{}
	capture.Write(bytes.Repeat([]byte("x"), maxCapturedBodyBytes+1024))
	if got := len(capture.Bytes()); got != maxCapturedBodyBytes {
		t.Fatalf("expected %d captured bytes, got %d", maxCapturedBodyBytes, got)
	}
	if !capture.truncated {
		t.Fatal("expected capture to be marked truncated")
	}
	if capture.Size() != maxCapturedBodyBytes+1024 {
		t.Fatalf("expected full byte count, got %d", capture.Size())
	}
}

func TestCaptureRecorderDrainsBeforeClose(t *testing.T) {
	recorder := newCaptureRecorder()
	client := &SshClient{}
	persisted := make(chan struct{}, 1)
	if !recorder.submit(client, captureTaskFunc(func() { persisted <- struct{}{} })) {
		t.Fatal("expected capture task to be accepted")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := recorder.close(ctx); err != nil {
		t.Fatalf("close recorder: %v", err)
	}
	select {
	case <-persisted:
	default:
		t.Fatal("recorder closed before draining task")
	}
	if recorder.submit(client, captureTaskFunc(func() {})) {
		t.Fatal("closed recorder accepted another task")
	}
}

type repeatedByteReader byte

func (r repeatedByteReader) Read(payload []byte) (int, error) {
	for index := range payload {
		payload[index] = byte(r)
	}
	return len(payload), nil
}

func TestForwardWebSocketFrameStreamsLargePayloadWithBoundedCapture(t *testing.T) {
	payloadLength := int64(16<<20 + 1)
	header := []byte{0x82, 0x7f}
	length := make([]byte, 8)
	binary.BigEndian.PutUint64(length, uint64(payloadLength))
	reader := io.MultiReader(
		bytes.NewReader(append(header, length...)),
		io.LimitReader(repeatedByteReader('x'), payloadLength),
	)
	frame, err := forwardWebSocketFrame(reader, io.Discard)
	if err != nil {
		t.Fatalf("forward frame: %v", err)
	}
	if frame.PayloadLength != int(payloadLength) || len(frame.Payload) != maxCapturedBodyBytes {
		t.Fatalf("payload length=%d capture=%d", frame.PayloadLength, len(frame.Payload))
	}
}

type shortWriter struct {
	buffer bytes.Buffer
}

func (w *shortWriter) Write(payload []byte) (int, error) {
	if len(payload) > 2 {
		payload = payload[:2]
	}
	return w.buffer.Write(payload)
}

func TestWriteAllHandlesShortWrites(t *testing.T) {
	writer := &shortWriter{}
	payload := []byte("websocket-frame")
	if err := writeAll(writer, payload); err != nil {
		t.Fatalf("writeAll failed: %v", err)
	}
	if !bytes.Equal(writer.buffer.Bytes(), payload) {
		t.Fatalf("unexpected payload %q", writer.buffer.Bytes())
	}
}

func TestWriteAllRejectsZeroLengthWrite(t *testing.T) {
	err := writeAll(writerFunc(func([]byte) (int, error) { return 0, nil }), []byte("x"))
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("expected short write, got %v", err)
	}
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(payload []byte) (int, error) { return f(payload) }

func TestReconnectBackoffIsBounded(t *testing.T) {
	for attempt := 1; attempt <= 20; attempt++ {
		delay := reconnectBackoff(attempt)
		if delay < time.Second || delay >= 33*time.Second {
			t.Fatalf("attempt %d produced out-of-range delay %s", attempt, delay)
		}
	}
}

func TestHTTPRemotePortsRemainLegacyServerCompatible(t *testing.T) {
	for _, port := range remotePortCandidates(constants.Http) {
		if port == 0 || port < 20000 || port > 30000 {
			t.Fatalf("HTTP candidate port %d is not legacy-server compatible", port)
		}
	}
}

func tcpPair(t *testing.T) (*net.TCPConn, *net.TCPConn) {
	t.Helper()
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	accepted := make(chan *net.TCPConn, 1)
	go func() {
		conn, acceptErr := listener.AcceptTCP()
		if acceptErr == nil {
			accepted <- conn
		}
	}()
	client, err := net.DialTCP("tcp", nil, listener.Addr().(*net.TCPAddr))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	server := <-accepted
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})
	return server, client
}

func TestTCPTunnelPropagatesHalfClose(t *testing.T) {
	remoteTunnel, remoteClient := tcpPair(t)
	localTunnel, localServer := tcpPair(t)
	deadline := time.Now().Add(2 * time.Second)
	_ = remoteClient.SetDeadline(deadline)
	_ = localServer.SetDeadline(deadline)

	client := &SshClient{}
	tunnelDone := make(chan struct{})
	go func() {
		client.tcpTunnel(remoteTunnel, localTunnel)
		close(tunnelDone)
	}()

	backendDone := make(chan error, 1)
	go func() {
		request, err := io.ReadAll(localServer)
		if err != nil {
			backendDone <- err
			return
		}
		if string(request) != "request" {
			backendDone <- errors.New("unexpected request")
			return
		}
		_, err = localServer.Write([]byte("response"))
		if closeErr := localServer.CloseWrite(); err == nil {
			err = closeErr
		}
		backendDone <- err
	}()

	if _, err := remoteClient.Write([]byte("request")); err != nil {
		t.Fatalf("write request: %v", err)
	}
	if err := remoteClient.CloseWrite(); err != nil {
		t.Fatalf("half-close request: %v", err)
	}
	response, err := io.ReadAll(remoteClient)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if string(response) != "response" {
		t.Fatalf("unexpected response %q", response)
	}
	if err := <-backendDone; err != nil {
		t.Fatalf("backend failed: %v", err)
	}
	select {
	case <-tunnelDone:
	case <-time.After(2 * time.Second):
		t.Fatal("tunnel did not close after both half-closes")
	}
}
