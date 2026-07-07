package auth

import "testing"

func TestSafeNextPath(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
		ok   bool
	}{
		{name: "admin path", raw: "/team/overview", want: "/team/overview", ok: true},
		{name: "query string", raw: "/team/users?page=2", want: "/team/users?page=2", ok: true},
		{name: "root", raw: "/", want: "/", ok: true},
		{name: "trim whitespace", raw: " /team/overview ", want: "/team/overview", ok: true},
		{name: "https url", raw: "https://example.invalid/path", ok: false},
		{name: "http url", raw: "http://example.invalid/path", ok: false},
		{name: "protocol relative", raw: "//example.invalid/path", ok: false},
		{name: "javascript", raw: "javascript:alert(1)", ok: false},
		{name: "newline", raw: "/team/overview\nLocation: https://example.invalid", ok: false},
		{name: "carriage return", raw: "/team/overview\rLocation: https://example.invalid", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := safeNextPath(tt.raw)
			if ok != tt.ok {
				t.Fatalf("expected ok=%v, got %v", tt.ok, ok)
			}
			if got != tt.want {
				t.Fatalf("expected path %q, got %q", tt.want, got)
			}
		})
	}
}
