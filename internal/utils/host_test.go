package utils

import "testing"

func TestIsLoopbackHost(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{host: "localhost", want: true},
		{host: "LOCALHOST:7777", want: true},
		{host: "127.0.0.1", want: true},
		{host: "127.0.0.1:7777", want: true},
		{host: "::1", want: true},
		{host: "[::1]", want: true},
		{host: "[::1]:7777", want: true},
		{host: "", want: false},
		{host: "0.0.0.0", want: false},
		{host: "::", want: false},
		{host: "192.168.1.20:7777", want: false},
		{host: "localhost.attacker.example", want: false},
		{host: "attacker.example:7777", want: false},
	}

	for _, tt := range tests {
		if got := IsLoopbackHost(tt.host); got != tt.want {
			t.Errorf("IsLoopbackHost(%q) = %v, want %v", tt.host, got, tt.want)
		}
	}
}
