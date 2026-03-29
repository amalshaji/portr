package handler

import (
	"testing"
	"time"

	"github.com/amalshaji/portr/internal/client/db"
)

func TestSerializeWebSocketEventDecodesTextBinaryFrames(t *testing.T) {
	event := db.WebSocketEvent{
		ID:            "evt-1",
		Direction:     "server",
		Opcode:        2,
		OpcodeName:    "binary",
		IsFinal:       true,
		Payload:       []byte(`{"type":"echo","message":"hello"}`),
		PayloadLength: len(`{"type":"echo","message":"hello"}`),
		LoggedAt:      time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC),
	}

	payload := serializeWebSocketEvent(event)
	if payload.PayloadText != `{"type":"echo","message":"hello"}` {
		t.Fatalf("expected payload text to be decoded, got %q", payload.PayloadText)
	}
}

func TestSerializeWebSocketEventLeavesOpaqueBinaryFramesUndecoded(t *testing.T) {
	event := db.WebSocketEvent{
		ID:            "evt-2",
		Direction:     "server",
		Opcode:        2,
		OpcodeName:    "binary",
		IsFinal:       true,
		Payload:       []byte{0xff, 0xfe, 0xfd, 0x00},
		PayloadLength: 4,
		LoggedAt:      time.Date(2026, 3, 29, 12, 5, 0, 0, time.UTC),
	}

	payload := serializeWebSocketEvent(event)
	if payload.PayloadText != "" {
		t.Fatalf("expected no payload text for opaque binary frame, got %q", payload.PayloadText)
	}
}
