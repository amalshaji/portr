package replay

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amalshaji/portr/internal/client/db"
	"github.com/go-resty/resty/v2"
	"gorm.io/datatypes"
)

func TestExecuteUsesOriginalRequestAndInjectsReplayHeader(t *testing.T) {
	t.Parallel()

	type observedRequest struct {
		Method        string
		Path          string
		Authorization string
		ReplayID      string
		ContentLength string
		Body          string
	}

	var observed observedRequest
	server := openReplayTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		payload, err := ioReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		observed = observedRequest{
			Method:        r.Method,
			Path:          r.URL.RequestURI(),
			Authorization: r.Header.Get("Authorization"),
			ReplayID:      r.Header.Get("X-Portr-Replayed-Request-Id"),
			ContentLength: r.Header.Get("Content-Length"),
			Body:          string(payload),
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})
	defer server.Close()

	request := &db.Request{
		ID:      "req-1",
		Host:    strings.TrimPrefix(server.URL, "https://"),
		Url:     "/submit?x=1",
		Method:  "POST",
		Headers: mustJSONHeaders(t, map[string][]string{"Authorization": {"Bearer token"}, "Content-Length": {"999"}}),
		Body:    []byte("hello"),
	}

	result, err := Execute(request, EditOptions{})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if observed.Method != "POST" {
		t.Fatalf("expected POST, got %q", observed.Method)
	}
	if observed.Path != "/submit?x=1" {
		t.Fatalf("expected original path, got %q", observed.Path)
	}
	if observed.Authorization != "Bearer token" {
		t.Fatalf("expected authorization header, got %q", observed.Authorization)
	}
	if observed.ReplayID != "req-1" {
		t.Fatalf("expected replay id req-1, got %q", observed.ReplayID)
	}
	if observed.ContentLength == "999" {
		t.Fatalf("expected content-length to be recomputed, got %q", observed.ContentLength)
	}
	if observed.Body != "hello" {
		t.Fatalf("expected original body, got %q", observed.Body)
	}

	if result.EffectiveURL != requestURL(server, "/submit?x=1") {
		t.Fatalf("unexpected effective url %q", result.EffectiveURL)
	}
	if result.ResponseStatus != http.StatusOK {
		t.Fatalf("expected 200, got %d", result.ResponseStatus)
	}
	if string(result.ResponseBody) != "ok" {
		t.Fatalf("expected response body ok, got %q", string(result.ResponseBody))
	}
}

func TestExecuteAppliesOverridesAndDropsHeaders(t *testing.T) {
	t.Parallel()

	type observedRequest struct {
		Method        string
		Path          string
		Authorization string
		DropHeader    string
		CustomHeader  string
		ReplayID      string
		Body          string
	}

	var observed observedRequest
	server := openReplayTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		payload, err := ioReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		observed = observedRequest{
			Method:        r.Method,
			Path:          r.URL.RequestURI(),
			Authorization: r.Header.Get("Authorization"),
			DropHeader:    r.Header.Get("X-Drop"),
			CustomHeader:  r.Header.Get("X-Test"),
			ReplayID:      r.Header.Get("X-Portr-Replayed-Request-Id"),
			Body:          string(payload),
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	defer server.Close()

	request := &db.Request{
		ID:      "req-2",
		Host:    strings.TrimPrefix(server.URL, "https://"),
		Url:     "/original",
		Method:  "GET",
		Headers: mustJSONHeaders(t, map[string][]string{"Authorization": {"Bearer token"}, "X-Drop": {"1"}}),
		Body:    []byte("ignored"),
	}

	bodyValue := base64.StdEncoding.EncodeToString([]byte("custom"))
	result, err := Execute(request, EditOptions{
		Method: "PATCH",
		Path:   "/edited?y=1",
		Headers: map[string]string{
			"authorization": "override",
			"X-Test":        "present",
		},
		DropHeaders: []string{"x-drop"},
		Body: BodyOverride{
			Set:      true,
			Value:    bodyValue,
			Encoding: "base64",
		},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if observed.Method != "PATCH" {
		t.Fatalf("expected PATCH, got %q", observed.Method)
	}
	if observed.Path != "/edited?y=1" {
		t.Fatalf("expected edited path, got %q", observed.Path)
	}
	if observed.Authorization != "override" {
		t.Fatalf("expected overridden authorization, got %q", observed.Authorization)
	}
	if observed.DropHeader != "" {
		t.Fatalf("expected dropped header to be removed, got %q", observed.DropHeader)
	}
	if observed.CustomHeader != "present" {
		t.Fatalf("expected custom header, got %q", observed.CustomHeader)
	}
	if observed.ReplayID != "req-2" {
		t.Fatalf("expected replay id req-2, got %q", observed.ReplayID)
	}
	if observed.Body != "custom" {
		t.Fatalf("expected custom body, got %q", observed.Body)
	}
	if result.EffectiveMethod != "PATCH" {
		t.Fatalf("expected PATCH, got %q", result.EffectiveMethod)
	}
	if result.EffectivePath != "/edited?y=1" {
		t.Fatalf("expected edited path, got %q", result.EffectivePath)
	}
}

func TestExecuteReplaceHeaders(t *testing.T) {
	t.Parallel()

	type observedRequest struct {
		Authorization string
		NewHeader     string
	}

	var observed observedRequest
	server := openReplayTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		observed = observedRequest{
			Authorization: r.Header.Get("Authorization"),
			NewHeader:     r.Header.Get("X-New"),
		}
		w.WriteHeader(http.StatusCreated)
	})
	defer server.Close()

	request := &db.Request{
		ID:      "req-3",
		Host:    strings.TrimPrefix(server.URL, "https://"),
		Url:     "/replace",
		Method:  "POST",
		Headers: mustJSONHeaders(t, map[string][]string{"Authorization": {"Bearer token"}, "X-Old": {"1"}}),
	}

	result, err := Execute(request, EditOptions{
		Headers: map[string]string{
			"X-New": "1",
		},
		ReplaceHeaders: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if observed.Authorization != "" {
		t.Fatalf("expected Authorization header to be removed, got %q", observed.Authorization)
	}
	if observed.NewHeader != "1" {
		t.Fatalf("expected X-New header, got %q", observed.NewHeader)
	}
	if result.ResponseStatus != http.StatusCreated {
		t.Fatalf("expected 201, got %d", result.ResponseStatus)
	}
}

func TestExecuteRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	request := &db.Request{
		ID:      "req-4",
		Host:    "demo.example.com",
		Url:     "/demo",
		Method:  "POST",
		Headers: mustJSONHeaders(t, map[string][]string{}),
	}

	testCases := []struct {
		name string
		opts EditOptions
		err  error
	}{
		{
			name: "method",
			opts: EditOptions{Method: "BREW"},
			err:  ErrUnsupportedMethod,
		},
		{
			name: "path",
			opts: EditOptions{Path: "relative"},
			err:  ErrInvalidPath,
		},
		{
			name: "body",
			opts: EditOptions{
				Body: BodyOverride{
					Set:      true,
					Value:    "not-base64",
					Encoding: "base64",
				},
			},
			err: ErrInvalidBodyEncoding,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := Execute(request, tc.opts)
			if !errors.Is(err, tc.err) {
				t.Fatalf("expected %v, got %v", tc.err, err)
			}
		})
	}
}

func TestExecuteMapsPortrErrorReason(t *testing.T) {
	t.Parallel()

	server := openReplayTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Portr-Error-Reason", "connection-lost")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("failed"))
	})
	defer server.Close()

	request := &db.Request{
		ID:      "req-5",
		Host:    strings.TrimPrefix(server.URL, "https://"),
		Url:     "/demo",
		Method:  "GET",
		Headers: mustJSONHeaders(t, map[string][]string{}),
	}

	result, err := Execute(request, EditOptions{})
	if result == nil {
		t.Fatal("expected replay result")
	}

	var failure *Failure
	if !errors.As(err, &failure) {
		t.Fatalf("expected replay failure, got %v", err)
	}
	if failure.StatusCode != 503 {
		t.Fatalf("expected status 503, got %d", failure.StatusCode)
	}
	if failure.Reason != "connection-lost" {
		t.Fatalf("expected reason connection-lost, got %q", failure.Reason)
	}
	if string(result.ResponseBody) != "failed" {
		t.Fatalf("expected response body failed, got %q", string(result.ResponseBody))
	}
}

func openReplayTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	server := httptest.NewTLSServer(handler)
	previous := newRestyClient
	newRestyClient = func() *resty.Client {
		return resty.NewWithClient(&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		})
	}
	t.Cleanup(func() {
		newRestyClient = previous
	})

	return server
}

func mustJSONHeaders(t *testing.T, headers map[string][]string) datatypes.JSON {
	t.Helper()

	encoded, err := json.Marshal(headers)
	if err != nil {
		t.Fatalf("marshal headers: %v", err)
	}

	return encoded
}

func requestURL(server *httptest.Server, path string) string {
	return "https://" + strings.TrimPrefix(server.URL, "https://") + path
}

func ioReadAll(reader io.Reader) ([]byte, error) {
	return io.ReadAll(reader)
}
