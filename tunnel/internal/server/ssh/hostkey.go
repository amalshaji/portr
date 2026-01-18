package sshd

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"

	"github.com/charmbracelet/log"
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

func loadHostKey(pemKey string) (ssh.Option, error) {
	if pemKey == "" {
		return nil, errors.New("PORTR_SSH_HOST_KEY is required")
	}

	if _, err := gossh.ParsePrivateKey([]byte(pemKey)); err != nil {
		return nil, err
	}

	log.Info("Loaded SSH host key from environment")
	return ssh.HostKeyPEM([]byte(pemKey)), nil
}

func GenerateHostKey() (string, error) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", err
	}

	pemBlock, err := gossh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		return "", err
	}

	return string(pem.EncodeToMemory(pemBlock)), nil
}
