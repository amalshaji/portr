package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDirExistsCreatesPrivateDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "portr", "config")

	if err := EnsureDirExists(path); err != nil {
		t.Fatalf("ensure dir exists: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat directory: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected directory, got %v", info.Mode())
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("expected private directory permissions, got %v", info.Mode().Perm())
	}
}

func TestEnsureDirExistsCorrectsExistingDirectoryPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "portr")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("create directory: %v", err)
	}
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod directory: %v", err)
	}

	if err := EnsureDirExists(path); err != nil {
		t.Fatalf("ensure dir exists: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat directory: %v", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("expected private directory permissions, got %v", info.Mode().Perm())
	}
}

func TestEnsureDirExistsRejectsFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(path, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if err := EnsureDirExists(path); err == nil {
		t.Fatal("expected error for file path, got nil")
	}
}
