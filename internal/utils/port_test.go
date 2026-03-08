package utils

import (
	"testing"
)

func TestGenerateRandomHttpPorts(t *testing.T) {
	ports := GenerateRandomHttpPorts()

	if len(ports) != 10 {
		t.Errorf("Expected 10 ports, got %d", len(ports))
	}

	for i, port := range ports {
		if port < 20000 || port > 30000 {
			t.Errorf("Port[%d] = %d is outside expected range [20000-30000]", i, port)
		}
	}

	portMap := make(map[int]bool)
	for _, port := range ports {
		if portMap[port] {
			t.Errorf("Duplicate port found: %d", port)
		}
		portMap[port] = true
	}
}

func TestGenerateRandomTcpPorts(t *testing.T) {
	ports := GenerateRandomTcpPorts()

	if len(ports) != 10 {
		t.Errorf("Expected 10 ports, got %d", len(ports))
	}

	for i, port := range ports {
		if port < 30001 || port > 40001 {
			t.Errorf("Port[%d] = %d is outside expected range [30001-40001]", i, port)
		}
	}

	portMap := make(map[int]bool)
	for _, port := range ports {
		if portMap[port] {
			t.Errorf("Duplicate port found: %d", port)
		}
		portMap[port] = true
	}
}
