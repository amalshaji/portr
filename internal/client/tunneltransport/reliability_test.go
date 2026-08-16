package tunneltransport

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	"testing"
	"time"
)

type captureTaskFunc func()

func (f captureTaskFunc) persist(*Client) { f() }

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
	client := &Client{}
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

func TestDeadlineConnInterruptsReadWithoutLosingData(t *testing.T) {
	server, client := net.Pipe()
	wrapped := newDeadlineConn(server)
	t.Cleanup(func() {
		_ = wrapped.Close()
		_ = client.Close()
	})

	readDone := make(chan error, 1)
	go func() {
		buffer := make([]byte, 1)
		_, err := wrapped.Read(buffer)
		readDone <- err
	}()
	if err := wrapped.SetReadDeadline(time.Now().Add(20 * time.Millisecond)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	select {
	case err := <-readDone:
		if !errors.Is(err, os.ErrDeadlineExceeded) {
			t.Fatalf("expected deadline error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("deadline did not interrupt read")
	}

	if err := wrapped.SetReadDeadline(time.Time{}); err != nil {
		t.Fatalf("clear read deadline: %v", err)
	}
	writeDone := make(chan error, 1)
	go func() {
		_, err := client.Write([]byte("websocket-frame"))
		writeDone <- err
	}()
	buffer := make([]byte, len("websocket-frame"))
	if _, err := io.ReadFull(wrapped, buffer); err != nil {
		t.Fatalf("read data after deadline: %v", err)
	}
	if err := <-writeDone; err != nil {
		t.Fatalf("write data after deadline: %v", err)
	}
	if string(buffer) != "websocket-frame" {
		t.Fatalf("data was lost after interrupted read: %q", buffer)
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

	client := &Client{}
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
