package utils

import (
	"fmt"
	"strings"
	"testing"
)

func TestLocalServerNotOnline(t *testing.T) {
	result := LocalServerNotOnline("http://localhost:8080")
	if result == "" {
		t.Error("LocalServerNotOnline returned empty string")
	}

	if !strings.Contains(result, "Please check if the local server is running and is accessible.") {
		t.Error("LocalServerNotOnline should return HTML content")
	}
}

func TestUnregisteredSubdomain(t *testing.T) {
	testSubdomain := "test-subdomain"
	result := UnregisteredSubdomain(testSubdomain)

	if result == "" {
		t.Error("UnregisteredSubdomain returned empty string")
	}

	if !strings.Contains(result, fmt.Sprintf("portr http 8000 -s %s", testSubdomain)) {
		t.Errorf("UnregisteredSubdomain output should contain the subdomain %q", testSubdomain)
	}

	if !strings.Contains(result, "<!DOCTYPE html>") {
		t.Error("UnregisteredSubdomain should return HTML content")
	}
}

func TestConnectionLost(t *testing.T) {
	result := ConnectionLost()
	if result == "" {
		t.Error("ConnectionLost returned empty string")
	}

	if !strings.Contains(result, "Don't worry, we'll keep retrying to establish the connection") {
		t.Error("ConnectionLost should return HTML content")
	}
}
