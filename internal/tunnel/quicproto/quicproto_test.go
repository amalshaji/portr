package quicproto

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"net"
	"strings"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	want := Frame{Type: TypeHello, ConnectionID: "conn", SecretKey: "secret", Version: ProtocolVersion}
	if err := WriteFrame(&buf, want); err != nil {
		t.Fatal(err)
	}
	if err := WriteFrame(&buf, Frame{Type: TypeReady, Port: 30001}); err != nil {
		t.Fatal(err)
	}

	got, err := ReadFrame(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("first frame = %+v, want %+v", got, want)
	}
	got, err = ReadFrame(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != TypeReady || got.Port != 30001 {
		t.Fatalf("second frame = %+v", got)
	}
}

func TestReadFrameRejectsOversizedFrame(t *testing.T) {
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], maxFrameSize+1)
	_, err := ReadFrame(bytes.NewReader(header[:]))
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("ReadFrame error = %v, want too large", err)
	}
}

func TestKeyPinIsStableAndUnique(t *testing.T) {
	key := mustKey(t)
	first, err := KeyPin(key)
	if err != nil {
		t.Fatal(err)
	}
	second, err := KeyPin(key)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || len(first) != 64 {
		t.Fatalf("pins = %q, %q; want equal 64-char hex", first, second)
	}
	other, err := KeyPin(mustKey(t))
	if err != nil {
		t.Fatal(err)
	}
	if other == first {
		t.Fatal("different keys produced the same pin")
	}
}

func TestKeyPinRejectsInvalidKey(t *testing.T) {
	if _, err := KeyPin("not a key"); err == nil {
		t.Fatal("expected error for non-PEM key")
	}
}

func TestClientTLSConfigRequiresPin(t *testing.T) {
	if _, err := ClientTLSConfig("", false); err == nil {
		t.Fatal("expected error when pin is missing")
	}
	if _, err := ClientTLSConfig("", true); err != nil {
		t.Fatalf("insecure config without pin: %v", err)
	}
}

func TestTLSHandshakeVerifiesPin(t *testing.T) {
	key := mustKey(t)
	pin, err := KeyPin(key)
	if err != nil {
		t.Fatal(err)
	}
	otherPin, err := KeyPin(mustKey(t))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		pin      string
		insecure bool
		wantErr  string
	}{
		{name: "matching pin", pin: strings.ToUpper(pin)},
		{name: "wrong pin", pin: otherPin, wantErr: "pin mismatch"},
		{name: "insecure", insecure: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientConfig, err := ClientTLSConfig(tt.pin, tt.insecure)
			if err != nil {
				t.Fatal(err)
			}
			protocol, err := handshake(t, key, clientConfig)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("handshake error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if protocol != ALPN {
				t.Fatalf("negotiated protocol = %q, want %q", protocol, ALPN)
			}
		})
	}
}

func handshake(t *testing.T, key string, clientConfig *tls.Config) (string, error) {
	t.Helper()
	serverConfig, err := ServerTLSConfig(key)
	if err != nil {
		t.Fatal(err)
	}
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() { _ = serverConn.Close(); _ = clientConn.Close() })

	server := tls.Server(serverConn, serverConfig)
	go func() {
		_ = server.Handshake()
		_ = server.Close()
	}()
	client := tls.Client(clientConn, clientConfig)
	if err := client.Handshake(); err != nil {
		return "", err
	}
	return client.ConnectionState().NegotiatedProtocol, nil
}

func mustKey(t *testing.T) string {
	t.Helper()
	key, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	return key
}
