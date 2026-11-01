package dashboard

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestLocalOnly(t *testing.T) {
	app := fiber.New()
	app.Use(localOnly)
	app.All("/", func(c *fiber.Ctx) error { return c.SendString("ok") })

	cases := []struct {
		name    string
		method  string
		host    string
		headers map[string]string
		want    int
	}{
		{name: "localhost", method: http.MethodGet, host: "localhost:7777", want: http.StatusOK},
		{name: "ipv4 loopback", method: http.MethodGet, host: "127.0.0.1:7777", want: http.StatusOK},
		{name: "ipv6 loopback", method: http.MethodGet, host: "[::1]:7777", want: http.StatusOK},
		{name: "same origin post", method: http.MethodPost, host: "localhost:7777", headers: map[string]string{"Origin": "http://localhost:7777"}, want: http.StatusOK},
		{name: "rebound host", method: http.MethodGet, host: "attacker.example:7777", want: http.StatusForbidden},
		{name: "lan address", method: http.MethodGet, host: "192.168.1.20:7777", want: http.StatusForbidden},
		{name: "forwarded host is ignored", method: http.MethodGet, host: "attacker.example:7777", headers: map[string]string{"X-Forwarded-Host": "localhost:7777"}, want: http.StatusForbidden},
		{name: "cross origin post", method: http.MethodPost, host: "localhost:7777", headers: map[string]string{"Origin": "https://attacker.example"}, want: http.StatusForbidden},
		{name: "other local port origin", method: http.MethodPost, host: "localhost:7777", headers: map[string]string{"Origin": "http://localhost:3000"}, want: http.StatusForbidden},
		{name: "null origin", method: http.MethodPost, host: "localhost:7777", headers: map[string]string{"Origin": "null"}, want: http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/", nil)
			req.Host = tc.host
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			resp.Body.Close()
			if resp.StatusCode != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, resp.StatusCode)
			}
		})
	}
}
