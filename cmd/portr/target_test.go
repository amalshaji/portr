package main

import "testing"

func TestParseLocalTarget(t *testing.T) {
	cases := []struct {
		in       string
		wantHost string
		wantPort int
		wantErr  bool
	}{
		{"8000", "", 8000, false},
		{"192.168.1.50:8000", "192.168.1.50", 8000, false},
		{"localhost:8000", "localhost", 8000, false},
		{"[::1]:8000", "::1", 8000, false},
		{"", "", 0, true},
		{"abc", "", 0, true},
		{"::1:8000", "", 0, true},
		{":8000", "", 0, true},
		{"192.168.1.50", "", 0, true},
		{"192.168.1.50:abc", "", 0, true},
		{"8000:", "", 0, true},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			host, port, err := parseLocalTarget(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseLocalTarget(%q) = %q, %d; want error", tc.in, host, port)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseLocalTarget(%q) unexpected error: %v", tc.in, err)
			}
			if host != tc.wantHost || port != tc.wantPort {
				t.Fatalf("parseLocalTarget(%q) = %q, %d; want %q, %d", tc.in, host, port, tc.wantHost, tc.wantPort)
			}
		})
	}
}
