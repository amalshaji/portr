// Package quicproto holds the pieces of the QUIC tunnel transport that the
// client and the server share: ALPN, control frames, TLS and quic-go settings.
package quicproto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
	"time"

	"github.com/quic-go/quic-go"
)

const (
	ALPN = "portr-tunnel/1"

	DefaultPort = 4433

	// ProtocolVersion is sent in the hello frame and echoed on the ready frame.
	ProtocolVersion = 1
)

// Control frames travel on the first stream the client opens. Data streams are
// opened by the server and carry raw tunnel bytes.
const (
	TypeHello = "hello"
	TypeReady = "ready"
	TypeError = "error"
	TypePing  = "ping"
	TypePong  = "pong"
)

// DataStreamHeader is the first byte the server writes on every data stream.
// QUIC only announces a stream once it carries data, so without it a TCP
// tunnel to a server-speaks-first service (SMTP, MySQL) would never reach the
// client.
const DataStreamHeader byte = 0x01

// maxFrameSize bounds control frames so a peer cannot make the other side
// allocate arbitrary memory before authenticating.
const maxFrameSize = 64 << 10

type Frame struct {
	Type         string `json:"type"`
	ConnectionID string `json:"connection_id,omitempty"`
	SecretKey    string `json:"secret_key,omitempty"`
	Version      int    `json:"version,omitempty"`
	Port         int    `json:"port,omitempty"`
	Message      string `json:"message,omitempty"`
}

// WriteFrame writes frame as a 4-byte big-endian length followed by JSON.
func WriteFrame(w io.Writer, frame Frame) error {
	payload, err := json.Marshal(frame)
	if err != nil {
		return err
	}
	if len(payload) > maxFrameSize {
		return fmt.Errorf("control frame too large: %d bytes", len(payload))
	}
	buf := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(buf, uint32(len(payload)))
	copy(buf[4:], payload)
	_, err = w.Write(buf)
	return err
}

func ReadFrame(r io.Reader) (Frame, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return Frame{}, err
	}
	size := binary.BigEndian.Uint32(header[:])
	if size > maxFrameSize {
		return Frame{}, fmt.Errorf("control frame too large: %d bytes", size)
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return Frame{}, err
	}
	var frame Frame
	if err := json.Unmarshal(payload, &frame); err != nil {
		return Frame{}, err
	}
	return frame, nil
}

// GenerateKey returns a PEM-encoded Ed25519 private key for the server's
// QUIC certificate.
func GenerateKey() (string, error) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", err
	}
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})), nil
}

func parseKey(keyPEM string) (ed25519.PrivateKey, error) {
	block, _ := pem.Decode([]byte(keyPEM))
	if block == nil {
		return nil, errors.New("quic tls key is not PEM encoded")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse quic tls key: %w", err)
	}
	privateKey, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("quic tls key must be an Ed25519 key")
	}
	return privateKey, nil
}

// KeyPin returns the pin clients use to verify the server: the hex SHA-256 of
// the public key's SubjectPublicKeyInfo. It depends only on the key, so it
// survives certificate regeneration on restart.
func KeyPin(keyPEM string) (string, error) {
	privateKey, err := parseKey(keyPEM)
	if err != nil {
		return "", err
	}
	spki, err := x509.MarshalPKIXPublicKey(privateKey.Public())
	if err != nil {
		return "", err
	}
	return spkiPin(spki), nil
}

func spkiPin(spki []byte) string {
	sum := sha256.Sum256(spki)
	return hex.EncodeToString(sum[:])
}

// ServerTLSConfig builds a self-signed certificate from keyPEM. Clients trust
// it by pin, not by a CA, so names and expiry are not checked.
func ServerTLSConfig(keyPEM string) (*tls.Config, error) {
	privateKey, err := parseKey(keyPEM)
	if err != nil {
		return nil, err
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "portr"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, privateKey.Public(), privateKey)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: privateKey}},
		NextProtos:   []string{ALPN},
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// ClientTLSConfig verifies the server by pin. insecure skips verification and
// is only meant for local development.
func ClientTLSConfig(pin string, insecure bool) (*tls.Config, error) {
	pin = strings.ToLower(strings.TrimSpace(pin))
	if pin == "" && !insecure {
		return nil, errors.New("quic_cert_sha256 is not set; run `portr auth set` again or set insecure_skip_quic_verification")
	}
	return &tls.Config{
		// The certificate is self-signed; VerifyConnection checks the pin.
		InsecureSkipVerify: true,
		NextProtos:         []string{ALPN},
		MinVersion:         tls.VersionTLS13,
		VerifyConnection: func(state tls.ConnectionState) error {
			if insecure {
				return nil
			}
			if len(state.PeerCertificates) == 0 {
				return errors.New("quic server sent no certificate")
			}
			got := spkiPin(state.PeerCertificates[0].RawSubjectPublicKeyInfo)
			if subtle.ConstantTimeCompare([]byte(got), []byte(pin)) != 1 {
				return fmt.Errorf("quic server certificate pin mismatch: expected %s, got %s", pin, got)
			}
			return nil
		},
	}, nil
}

// A stream whose reader stalls, such as a visitor that stops reading a
// download, keeps its whole window of the connection window. yamux has no
// connection window, but QUIC does, so it is fixed at many times the largest
// stream window: one stalled stream must not freeze the rest of the tunnel.
const (
	maxStreamWindow  = 2 << 20
	connectionWindow = 32 << 20
)

func baseConfig() *quic.Config {
	return &quic.Config{
		HandshakeIdleTimeout:           5 * time.Second,
		MaxIdleTimeout:                 30 * time.Second,
		KeepAlivePeriod:                10 * time.Second,
		InitialStreamReceiveWindow:     512 << 10,
		MaxStreamReceiveWindow:         maxStreamWindow,
		InitialConnectionReceiveWindow: connectionWindow,
		MaxConnectionReceiveWindow:     connectionWindow,
		MaxIncomingUniStreams:          -1,
		// The hello frame carries the secret key; 0-RTT data can be replayed.
		Allow0RTT: false,
	}
}

// ServerConfig accepts only the client's control stream; the server opens all
// data streams.
func ServerConfig() *quic.Config {
	config := baseConfig()
	config.MaxIncomingStreams = 1
	return config
}

func ClientConfig() *quic.Config {
	config := baseConfig()
	config.MaxIncomingStreams = 1024
	return config
}
