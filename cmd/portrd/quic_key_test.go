package main

import (
	"bytes"
	"testing"

	"github.com/amalshaji/portr/internal/tunnel/quicproto"
)

func TestGenerateQUICKeyCommand(t *testing.T) {
	var output bytes.Buffer
	app := newApp()
	app.Writer = &output
	if err := app.Run([]string{"portrd", "generate-quic-key"}); err != nil {
		t.Fatal(err)
	}
	if _, err := quicproto.KeyPin(output.String()); err != nil {
		t.Fatalf("command did not produce a valid quic key: %v", err)
	}
}
