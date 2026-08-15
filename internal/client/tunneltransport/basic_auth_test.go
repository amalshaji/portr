package tunneltransport

import (
	"bufio"
	"context"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	clientdb "github.com/amalshaji/portr/internal/client/db"
	clientcfg "github.com/amalshaji/portr/internal/clientconfig"
)

const testBasicAuthChallenge = `Basic realm="Restricted", charset="UTF-8"`

func basicAuthHeader(credential string) string {
	return "Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte(credential)) + "\r\n"
}

// runGatedRequest sends one raw request through the HTTP tunnel and parses the
// response head. Unlike runRawRequest it does not wait for EOF: a rejected
// websocket upgrade carries Connection: Upgrade, so the gate answers it without
// closing, and reading to EOF would block until the deadline.
func runGatedRequest(t *testing.T, client *Client, localEndpoint, rawRequest string) *http.Response {
	t.Helper()

	remoteConn, clientConn := net.Pipe()
	done := make(chan struct{})
	go func() {
		client.httpTunnel(remoteConn, localEndpoint)
		close(done)
	}()
	t.Cleanup(func() {
		_ = clientConn.Close()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("tunnel handler did not finish")
		}
	})

	if _, err := clientConn.Write([]byte(rawRequest)); err != nil {
		t.Fatalf("write request: %v", err)
	}
	if err := clientConn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}

	response, err := http.ReadResponse(bufio.NewReader(clientConn), nil)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

// gatedBackend is a local server that records what reached it, so a test can
// assert that a rejected request never got past the gate.
type gatedBackend struct {
	endpoint      string
	hits          *atomic.Int64
	authorization chan string
}

func newGatedBackend(t *testing.T) gatedBackend {
	t.Helper()

	hits := &atomic.Int64{}
	authorization := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		select {
		case authorization <- r.Header.Get("Authorization"):
		default:
		}
		_, _ = io.WriteString(w, "ok")
	}))
	t.Cleanup(server.Close)

	return gatedBackend{
		endpoint:      strings.TrimPrefix(server.URL, "http://"),
		hits:          hits,
		authorization: authorization,
	}
}

func basicAuthClient(credential string) *Client {
	return &Client{config: clientcfg.ClientConfig{Tunnel: clientcfg.Tunnel{
		Name: "test", Subdomain: "test", Port: 3000, BasicAuth: credential,
	}}}
}

func httpRequest(extraHeaders string) string {
	return "GET / HTTP/1.1\r\nHost: test.example\r\n" + extraHeaders + "Connection: close\r\n\r\n"
}

func websocketUpgradeRequest(extraHeaders string) string {
	return "GET /ws HTTP/1.1\r\n" +
		"Host: test.example\r\n" +
		"Connection: Upgrade\r\n" +
		"Upgrade: websocket\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n" +
		extraHeaders + "\r\n"
}

func TestHTTPTunnelRejectsMissingCredentials(t *testing.T) {
	backend := newGatedBackend(t)

	response := runGatedRequest(t, basicAuthClient("admin:s3cret"), backend.endpoint, httpRequest(""))

	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected a 401, got %d", response.StatusCode)
	}
	// Without this header browsers never show their credential prompt.
	if challenge := response.Header.Get("WWW-Authenticate"); challenge != testBasicAuthChallenge {
		t.Fatalf("expected challenge %q, got %q", testBasicAuthChallenge, challenge)
	}
	// A 401 is the feature working, not a portr failure: replayFailure turns an
	// unrecognised error reason into a 502.
	if response.Header.Get("X-Portr-Error") != "" {
		t.Fatal("a 401 must not be marked as a portr error")
	}
	if hits := backend.hits.Load(); hits != 0 {
		t.Fatalf("the local server must not be reached, got %d hits", hits)
	}
}

func TestHTTPTunnelRejectsWrongCredentials(t *testing.T) {
	backend := newGatedBackend(t)

	response := runGatedRequest(t, basicAuthClient("admin:s3cret"), backend.endpoint,
		httpRequest(basicAuthHeader("admin:wrong")))

	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected a 401, got %d", response.StatusCode)
	}
	if hits := backend.hits.Load(); hits != 0 {
		t.Fatalf("the local server must not be reached, got %d hits", hits)
	}
}

func TestHTTPTunnelForwardsAuthorizationOnCorrectCredentials(t *testing.T) {
	backend := newGatedBackend(t)

	response := runGatedRequest(t, basicAuthClient("admin:s3cret"), backend.endpoint,
		httpRequest(basicAuthHeader("admin:s3cret")))

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected the request to pass the gate, got %d", response.StatusCode)
	}

	// Portr validates Authorization but forwards it unchanged, so a local app
	// checking the same credential keeps working.
	select {
	case got := <-backend.authorization:
		want := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:s3cret"))
		if got != want {
			t.Fatalf("expected Authorization to be forwarded as %q, got %q", want, got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("backend did not receive a request")
	}
}

func TestHTTPTunnelPassesThroughWithoutBasicAuth(t *testing.T) {
	backend := newGatedBackend(t)

	response := runGatedRequest(t, basicAuthClient(""), backend.endpoint, httpRequest(""))

	if response.StatusCode != http.StatusOK {
		t.Fatalf("an unprotected tunnel must pass the request through, got %d", response.StatusCode)
	}
	if hits := backend.hits.Load(); hits != 1 {
		t.Fatalf("expected exactly one backend hit, got %d", hits)
	}
}

func TestBasicAuthPasswordMayContainColon(t *testing.T) {
	backend := newGatedBackend(t)

	response := runGatedRequest(t, basicAuthClient("admin:pa:ss"), backend.endpoint,
		httpRequest(basicAuthHeader("admin:pa:ss")))

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected the colon-bearing password to be accepted, got %d", response.StatusCode)
	}
}

func TestMalformedBasicAuthLocksTheTunnel(t *testing.T) {
	backend := newGatedBackend(t)

	// Validate rejects a colon-less credential, so this is only reachable by a
	// caller that skipped validation. The gate hashes the credential whole, and
	// every request reproduces it as user + ":" + password, so a value with no
	// colon can never be matched -- including by the request that would match
	// if the two halves were compared separately.
	for _, attempt := range []string{"nocolon", "nocolon:", ":nocolon"} {
		t.Run(attempt, func(t *testing.T) {
			response := runGatedRequest(t, basicAuthClient("nocolon"), backend.endpoint,
				httpRequest(basicAuthHeader(attempt)))

			if response.StatusCode != http.StatusUnauthorized {
				t.Fatalf("expected %q to be rejected, got %d", attempt, response.StatusCode)
			}
		})
	}

	if hits := backend.hits.Load(); hits != 0 {
		t.Fatalf("the local server must not be reached, got %d hits", hits)
	}
}

func TestPingRequestBypassesBasicAuth(t *testing.T) {
	backend := newGatedBackend(t)

	// The portr server answers pings itself once it has a backend, but gating
	// them here would still break reconciliation against older servers.
	response := runGatedRequest(t, basicAuthClient("admin:s3cret"), backend.endpoint,
		httpRequest("X-Portr-Ping-Request: true\r\n"))

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected the ping to bypass the gate, got %d", response.StatusCode)
	}
}

func TestWebSocketUpgradeRejectedWithoutCredentials(t *testing.T) {
	received := make(chan string, 1)
	localEndpoint := startWebSocketEchoServer(t, received)

	// Gating upgrades matters: otherwise an Upgrade header bypasses the gate.
	response := runGatedRequest(t, basicAuthClient("admin:s3cret"), localEndpoint,
		websocketUpgradeRequest(""))

	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected a 401, got %d", response.StatusCode)
	}
	select {
	case host := <-received:
		t.Fatalf("the local websocket server must not be reached, saw a handshake for %q", host)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestWebSocketUpgradeAllowedWithCredentials(t *testing.T) {
	received := make(chan string, 1)
	localEndpoint := startWebSocketEchoServer(t, received)

	response := runGatedRequest(t, basicAuthClient("admin:s3cret"), localEndpoint,
		websocketUpgradeRequest(basicAuthHeader("admin:s3cret")))

	if response.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("expected the upgrade to be relayed, got %d", response.StatusCode)
	}
	select {
	case <-received:
	case <-time.After(2 * time.Second):
		t.Fatal("local websocket server did not receive a handshake")
	}
}

func TestRejectedRequestIsLoggedWithRedactedCredential(t *testing.T) {
	// The custom list omits Authorization on purpose: redact_headers must not
	// be able to expose the credential the auth gate itself relies on.
	tests := []struct {
		name          string
		redactHeaders []string
	}{
		{"default redact list", clientcfg.DefaultRedactHeaders},
		{"custom redact list without Authorization", []string{"Cookie"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := newGatedBackend(t)

			// Built through ClientConfigForTunnel because that is where
			// Authorization is forced into the redaction list.
			enabled := true
			base := clientcfg.Config{
				EnableRequestLogging:            &enabled,
				InsecureSkipHostKeyVerification: &enabled,
				RedactHeaders:                   tt.redactHeaders,
			}
			client := &Client{
				config: base.ClientConfigForTunnel(clientcfg.Tunnel{
					Name: "test", Subdomain: "test", Port: 3000, BasicAuth: "admin:s3cret",
				}),
				db: newTestRequestStore(t),
			}

			// runRawRequest rather than runGatedRequest: Shutdown must flush the
			// capture recorder after the tunnel handler has finished.
			runRawRequest(t, client, backend.endpoint, httpRequest(basicAuthHeader("admin:guess")))

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := client.Shutdown(shutdownCtx); err != nil {
				t.Fatalf("shutdown tunnel client: %v", err)
			}

			// Without this the inspector stays empty and a working gate looks like a
			// broken tunnel.
			var logged clientdb.Request
			if err := client.db.Conn.First(&logged).Error; err != nil {
				t.Fatalf("load persisted request: %v", err)
			}
			if logged.ResponseStatusCode != http.StatusUnauthorized {
				t.Fatalf("expected the rejection to be logged as 401, got %d", logged.ResponseStatusCode)
			}
			// Checking for the base64 token rather than the plaintext password:
			// a verbatim header contains "Basic YWRtaW46Z3Vlc3M=", never "guess",
			// so a plaintext check passes even when redaction is broken.
			if !strings.Contains(string(logged.Headers), `"Authorization":["[redacted]"]`) {
				t.Fatalf("the Authorization header must be stored as [redacted], got %s", logged.Headers)
			}
			encoded := base64.StdEncoding.EncodeToString([]byte("admin:guess"))
			if strings.Contains(string(logged.Headers), encoded) {
				t.Fatalf("the attempted credential must be redacted, got %s", logged.Headers)
			}
		})
	}
}
