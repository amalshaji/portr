package main

import (
	"bytes"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestGenerateHostKeyCommand(t *testing.T) {
	var output bytes.Buffer
	app := newApp()
	app.Writer = &output
	if err := app.Run([]string{"portrd", "generate-host-key"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ssh.ParsePrivateKey(output.Bytes()); err != nil {
		t.Fatalf("command did not produce a valid private key: %v", err)
	}
}
