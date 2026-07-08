package main

import "testing"

func TestRequiresAppServerToken(t *testing.T) {
	tests := []struct {
		name string
		host string
		want bool
	}{
		{name: "all IPv4 interfaces", host: "0.0.0.0", want: true},
		{name: "all IPv6 interfaces", host: "::", want: true},
		{name: "documentation IP", host: "192.0.2.10", want: true},
		{name: "hostname", host: "example.com", want: true},
		{name: "loopback IPv4", host: "127.0.0.1", want: false},
		{name: "localhost", host: "localhost", want: false},
		{name: "loopback IPv6", host: "::1", want: false},
		{name: "bracketed loopback IPv6", host: "[::1]", want: false},
		{name: "empty host", host: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := requiresAppServerToken(tt.host); got != tt.want {
				t.Fatalf("requiresAppServerToken(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}
