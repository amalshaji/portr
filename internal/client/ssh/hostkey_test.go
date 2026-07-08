package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestGetHostKeyCallback_Insecure(t *testing.T) {
	callback := getHostKeyCallback(true)
	if callback == nil {
		t.Fatal("expected non-nil callback")
	}
}

func TestGetHostKeyCallback_Secure(t *testing.T) {
	callback := getHostKeyCallback(false)
	if callback == nil {
		t.Fatal("expected non-nil callback")
	}
}

func TestFingerprintSHA256(t *testing.T) {
	key, err := generateTestKey()
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}

	fingerprint := fingerprintSHA256(key)
	if fingerprint == "" {
		t.Fatal("expected non-empty fingerprint")
	}
	if len(fingerprint) < 10 {
		t.Fatalf("fingerprint too short: %s", fingerprint)
	}
	if fingerprint[:7] != "SHA256:" {
		t.Fatalf("expected fingerprint to start with 'SHA256:', got %s", fingerprint)
	}
}

func TestKeysEqual(t *testing.T) {
	key1, err := generateTestKey()
	if err != nil {
		t.Fatalf("failed to generate test key 1: %v", err)
	}

	key2, err := generateTestKey()
	if err != nil {
		t.Fatalf("failed to generate test key 2: %v", err)
	}

	if !keysEqual(key1, key1) {
		t.Fatal("expected same key to be equal")
	}

	if keysEqual(key1, key2) {
		t.Fatal("expected different keys to not be equal")
	}
}

func TestSaveAndLoadHostKey(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "portr-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalPath := knownHostsPath
	knownHostsPath = filepath.Join(tmpDir, "known_hosts")
	defer func() { knownHostsPath = originalPath }()

	key, err := generateTestKey()
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}

	host := "test.example.com"

	loadedKey, err := loadKnownHostKey(host)
	if err != nil {
		t.Fatalf("failed to load (empty) known hosts: %v", err)
	}
	if loadedKey != nil {
		t.Fatal("expected nil key for unknown host")
	}

	err = saveHostKey(host, key)
	if err != nil {
		t.Fatalf("failed to save host key: %v", err)
	}

	loadedKey, err = loadKnownHostKey(host)
	if err != nil {
		t.Fatalf("failed to load known host key: %v", err)
	}
	if loadedKey == nil {
		t.Fatal("expected non-nil key after save")
	}

	if !keysEqual(key, loadedKey) {
		t.Fatal("loaded key does not match saved key")
	}
}

func generateTestKey() (ssh.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	return publicKey, nil
}
