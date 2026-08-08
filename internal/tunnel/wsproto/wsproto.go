package wsproto

import (
	"encoding/json"
	"sync"

	"golang.org/x/net/websocket"
)

const (
	ConnectPath        = "/_portr/tunnel/connect"
	ConnectionIDHeader = "X-Portr-Connection-ID"
	SecretKeyHeader    = "X-Portr-Secret-Key"

	TypeReady      = "ready"
	TypeOpen       = "open"
	TypeData       = "data"
	TypeWindow     = "window"
	TypeCloseWrite = "close_write"
	TypeClose      = "close"
	TypeError      = "error"
	TypePing       = "ping"
	TypePong       = "pong"
)

const StreamWindowSize = 32

type Frame struct {
	Type     string `json:"type"`
	StreamID string `json:"stream_id,omitempty"`
	Data     []byte `json:"data,omitempty"`
	Port     int    `json:"port,omitempty"`
	Window   int    `json:"window,omitempty"`
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
