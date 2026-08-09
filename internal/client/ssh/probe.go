package ssh

import (
	"context"
	"fmt"
	"net"
	"time"

	config "github.com/amalshaji/portr/internal/clientconfig"
	"golang.org/x/crypto/ssh"
)

// Probe completes an SSH handshake using cfg.ConnectionID and cfg.SecretKey and
// closes the connection without requesting a port forward. It returns the
// server host key fingerprint.
//
// Requesting a forward is what reserves a subdomain and marks the connection
// active, so skipping it keeps a diagnostic run from colliding with a tunnel
// the same user is about to start.
func Probe(ctx context.Context, cfg config.ClientConfig) (string, error) {
	verify := getHostKeyCallback(cfg.InsecureSkipHostKeyVerification)

	var fingerprint string
	callback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		fingerprint = fingerprintSHA256(key)
		return verify(hostname, remote, key)
	}

	sshConfig := &ssh.ClientConfig{
		User:            fmt.Sprintf("%s:%s", cfg.ConnectionID, cfg.SecretKey),
		Auth:            []ssh.AuthMethod{ssh.Password("")},
		HostKeyCallback: callback,
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	rawConn, err := dialer.DialContext(ctx, "tcp", cfg.SshUrl)
	if err != nil {
		return "", err
	}

	_ = rawConn.SetDeadline(time.Now().Add(10 * time.Second))
	cc, channels, requests, err := ssh.NewClientConn(rawConn, cfg.SshUrl, sshConfig)
	if err != nil {
		_ = rawConn.Close()
		return fingerprint, err
	}
	_ = rawConn.SetDeadline(time.Time{})

	go ssh.DiscardRequests(requests)
	go func() {
		for channel := range channels {
			_ = channel.Reject(ssh.Prohibited, "diagnostic probe")
		}
	}()

	_ = cc.Close()
	return fingerprint, nil
}
