package wsproto

import (
	"encoding/json"
	"net"
	"sync"

	"github.com/hashicorp/yamux"
	"golang.org/x/net/websocket"
)

const (
	ConnectPath           = "/_portr/tunnel/connect"
	ConnectionIDHeader    = "X-Portr-Connection-ID"
	SecretKeyHeader       = "X-Portr-Secret-Key"
	ProtocolVersionHeader = "X-Portr-Ws-Protocol"

	TypeReady = "ready"
	TypeError = "error"
)

// Data-plane frames of protocol v1, which multiplexed streams in JSON frames.
// The server still speaks v1 so older clients keep working; new clients use
// yamux.
const (
	TypeOpen       = "open"
	TypeData       = "data"
	TypeWindow     = "window"
	TypeCloseWrite = "close_write"
	TypeClose      = "close"
	TypePing       = "ping"
	TypePong       = "pong"
)

// StreamWindowSize is the v1 per-stream credit window, counted in frames.
const StreamWindowSize = 32

// ProtocolVersion is exchanged during the connect handshake: the client sends
// it in ProtocolVersionHeader and the server echoes it on the ready frame. The
// handshake is JSON text frames so a mismatched peer still gets a readable
// error; after the ready frame both sides switch the websocket to binary frames
// carrying a yamux session. The client requires an exact match; the server also
// accepts LegacyProtocolVersion.
const ProtocolVersion = 2

const LegacyProtocolVersion = 1

type Frame struct {
	Type     string `json:"type"`
	StreamID string `json:"stream_id,omitempty"`
	Data     []byte `json:"data,omitempty"`
	Port     int    `json:"port,omitempty"`
	Window   int    `json:"window,omitempty"`
	Version  int    `json:"version,omitempty"`
	Message  string `json:"message,omitempty"`
}

type Writer struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func NewWriter(conn *websocket.Conn) *Writer {
	return &Writer{conn: conn}
}

func (w *Writer) Send(frame Frame) error {
	payload, err := json.Marshal(frame)
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	return websocket.Message.Send(w.conn, string(payload))
}

func Receive(conn *websocket.Conn) (Frame, error) {
	var payload string
	if err := websocket.Message.Receive(conn, &payload); err != nil {
		return Frame{}, err
	}

	var frame Frame
	if err := json.Unmarshal([]byte(payload), &frame); err != nil {
		return Frame{}, err
	}
	return frame, nil
}

// MuxConfig is the yamux configuration both peers use for the tunnel session.
// yamux's own error logs go to logger.
func MuxConfig(logger yamux.Logger) *yamux.Config {
	// A 512 KiB stream window measured best on a ~50 ms, 14 MB/s link: yamux's
	// default 256 KiB stalls 256 KB websocket messages on a round trip per
	// message, while 1 MiB lets bulk downloads queue enough data ahead of small
	// responses to double their tail latency without adding throughput.
	config := yamux.DefaultConfig()
	config.MaxStreamWindowSize = 512 << 10
	// yamux keepalive closes the session when one ping takes longer than
	// ConnectionWriteTimeout, which a saturated uplink can cause on a healthy
	// tunnel. The client runs its own health check, which accepts any incoming
	// traffic as proof of life, and the server falls back to TCP keepalive.
	config.EnableKeepAlive = false
	config.LogOutput = nil
	config.Logger = logger
	return config
}

// MuxConn switches a handshaken websocket to binary frames so it can carry a
// yamux session.
func MuxConn(conn *websocket.Conn) net.Conn {
	conn.PayloadType = websocket.BinaryFrame
	return conn
}
