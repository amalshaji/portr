package ssh

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/amalshaji/portr/internal/client/config"
	"github.com/charmbracelet/log"
	"golang.org/x/crypto/ssh"
)

var knownHostsPath = filepath.Join(config.DefaultConfigDir, "known_hosts")

func getHostKeyCallback(insecureSkipVerification bool) ssh.HostKeyCallback {
	if insecureSkipVerification {
		return ssh.InsecureIgnoreHostKey()
	}
	return verifyHostKey
}

func verifyHostKey(hostname string, remote net.Addr, key ssh.PublicKey) error {
	host, _, err := net.SplitHostPort(hostname)
	if err != nil {
		host = hostname
	}

	knownKey, err := loadKnownHostKey(host)
	if err != nil {
		if os.IsNotExist(err) {
			return trustOnFirstUse(host, key)
		}
		return fmt.Errorf("failed to load known hosts: %w", err)
	}

	if knownKey == nil {
		return trustOnFirstUse(host, key)
	}

	if !keysEqual(knownKey, key) {
		fingerprint := fingerprintSHA256(key)
		return fmt.Errorf(
			"host key verification failed for %s\n"+
				"The server's host key has changed!\n"+
				"New key fingerprint: %s\n"+
				"This could indicate a man-in-the-middle attack.\n"+
				"If you trust this key, remove the old entry from %s and try again",
			host, fingerprint, knownHostsPath,
		)
	}

	return nil
}

func trustOnFirstUse(host string, key ssh.PublicKey) error {
	fingerprint := fingerprintSHA256(key)
	log.Info("New SSH host key", "host", host, "fingerprint", fingerprint)

	if err := saveHostKey(host, key); err != nil {
		log.Warn("Failed to save host key to known_hosts", "error", err)
	} else {
		log.Info("Host key saved to known_hosts", "path", knownHostsPath)
	}

	return nil
}

func loadKnownHostKey(host string) (ssh.PublicKey, error) {
	if err := ensureKnownHostsFile(); err != nil {
		return nil, err
	}

	file, err := os.Open(knownHostsPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		if fields[0] == host {
			keyBytes, err := base64.StdEncoding.DecodeString(fields[2])
			if err != nil {
				continue
			}
			key, err := ssh.ParsePublicKey(keyBytes)
			if err != nil {
				continue
			}
			return key, nil
		}
	}

	return nil, scanner.Err()
}

func saveHostKey(host string, key ssh.PublicKey) error {
	if err := ensureKnownHostsFile(); err != nil {
		return err
	}

	file, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	keyType := key.Type()
	keyData := base64.StdEncoding.EncodeToString(key.Marshal())
	line := fmt.Sprintf("%s %s %s\n", host, keyType, keyData)

	_, err = file.WriteString(line)
	return err
}

func ensureKnownHostsFile() error {
	dir := filepath.Dir(knownHostsPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
		file, err := os.OpenFile(knownHostsPath, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		file.Close()
	}

	return nil
}

func keysEqual(a, b ssh.PublicKey) bool {
	return string(a.Marshal()) == string(b.Marshal())
}

func fingerprintSHA256(key ssh.PublicKey) string {
	hash := sha256.Sum256(key.Marshal())
	return "SHA256:" + base64.StdEncoding.EncodeToString(hash[:])
}
