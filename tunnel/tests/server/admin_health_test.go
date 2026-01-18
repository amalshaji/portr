package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthcheck(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	srv := NewTestServer(t, db)

	req := httptest.NewRequest("GET", "/api/v1/healthcheck", nil)
	resp := DoRequest(t, srv, req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	status, ok := body["status"]
	if !ok {
		t.Fatalf("response missing 'status' field: %v", body)
	}

	if status != "ok" {
		t.Fatalf("expected status 'ok', got %q", status)
	}
}
