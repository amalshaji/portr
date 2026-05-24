package stubresponder

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestResponderRendersQueryValues(t *testing.T) {
	responder := testResponder(t, Route{
		Subdomain:        "yaml",
		ResponseFormat:   "application/yml",
		ResponseTemplate: "message: {{message}}\nmissing: {{missing}}\n",
	})

	req := httptest.NewRequest(http.MethodGet, "http://yaml.example.test/?message=hello", nil)
	req.Host = "yaml.example.test"
	rec := httptest.NewRecorder()

	responder.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/yml" {
		t.Fatalf("expected content type application/yml, got %q", got)
	}
	if got := rec.Body.String(); got != "message: hello\nmissing: \n" {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestResponderRendersJSONBodyValues(t *testing.T) {
	responder := testResponder(t, Route{
		Subdomain:        "json",
		ResponseFormat:   "application/json",
		ResponseTemplate: `{"name":"{{name}}","count":"{{count}}","enabled":"{{enabled}}"}`,
	})

	req := httptest.NewRequest(http.MethodPost, "http://json.example.test/", strings.NewReader(`{"name":"portr","count":3,"enabled":true}`))
	req.Host = "json.example.test"
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	responder.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Body.String(); got != `{"name":"portr","count":"3","enabled":"true"}` {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestResponderRendersTypedJSONValues(t *testing.T) {
	responder := testResponder(t, Route{
		Subdomain:        "json",
		ResponseFormat:   "application/json",
		ResponseTemplate: `{"status":"{{status|int}}","active":"{{active|bool}}","score":"{{score|float}}","message":"{{message}}"}`,
	})

	req := httptest.NewRequest(http.MethodGet, "http://json.example.test/?status=200&active=true&score=12.5&message=ok", nil)
	req.Host = "json.example.test"
	rec := httptest.NewRecorder()

	responder.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Body.String(); got != `{"status":200,"active":true,"score":12.5,"message":"ok"}` {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestResponderEscapesNormalJSONValues(t *testing.T) {
	responder := testResponder(t, Route{
		Subdomain:        "json",
		ResponseFormat:   "application/json",
		ResponseTemplate: `{"message":"{{message}}","raw":{{raw}}}`,
	})

	message := `hello "portr" \ ok`
	raw := `x"y`
	req := httptest.NewRequest(http.MethodGet, "http://json.example.test/?message="+url.QueryEscape(message)+"&raw="+url.QueryEscape(raw), nil)
	req.Host = "json.example.test"
	rec := httptest.NewRecorder()

	responder.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !json.Valid(rec.Body.Bytes()) {
		t.Fatalf("expected valid JSON, got %q", rec.Body.String())
	}
	var got map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got["message"] != message {
		t.Fatalf("expected escaped message %q, got %q", message, got["message"])
	}
	if got["raw"] != raw {
		t.Fatalf("expected escaped raw value %q, got %q", raw, got["raw"])
	}
}

func TestResponderKeepsFailedTypedJSONCastAsString(t *testing.T) {
	responder := testResponder(t, Route{
		Subdomain:        "json",
		ResponseFormat:   "application/json",
		ResponseTemplate: `{"status":"{{status|int}}"}`,
	})

	req := httptest.NewRequest(http.MethodGet, "http://json.example.test/?status=queued", nil)
	req.Host = "json.example.test"
	rec := httptest.NewRecorder()

	responder.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Body.String(); got != `{"status":"queued"}` {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestResponderKeepsFailedUnquotedTypedJSONCastAsString(t *testing.T) {
	responder := testResponder(t, Route{
		Subdomain:        "json",
		ResponseFormat:   "application/json",
		ResponseTemplate: `{"status":{{status|int}}}`,
	})

	req := httptest.NewRequest(http.MethodGet, "http://json.example.test/?status=queued", nil)
	req.Host = "json.example.test"
	rec := httptest.NewRecorder()

	responder.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Body.String(); got != `{"status":"queued"}` {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestResponderRendersTypedTextValues(t *testing.T) {
	responder := testResponder(t, Route{
		Subdomain:        "text",
		ResponseFormat:   "text/plain",
		ResponseTemplate: `status={{status|int}} active={{active|bool}} score={{score|float}}`,
	})

	req := httptest.NewRequest(http.MethodGet, "http://text.example.test/?status=200&active=true&score=12.5", nil)
	req.Host = "text.example.test"
	rec := httptest.NewRecorder()

	responder.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Body.String(); got != `status=200 active=true score=12.5` {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestResponderRendersFormBodyValues(t *testing.T) {
	responder := testResponder(t, Route{
		Subdomain:        "form",
		ResponseFormat:   "text/plain",
		ResponseTemplate: "name={{name}}",
	})

	form := url.Values{"name": []string{"from-form"}}
	req := httptest.NewRequest(http.MethodPost, "http://form.example.test/", strings.NewReader(form.Encode()))
	req.Host = "form.example.test"
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	responder.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Body.String(); got != "name=from-form" {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestResponderRendersMultipartFormBodyValues(t *testing.T) {
	responder := testResponder(t, Route{
		Subdomain:        "form",
		ResponseFormat:   "text/plain",
		ResponseTemplate: "name={{name}}",
	})

	body := &strings.Builder{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("name", "from-multipart"); err != nil {
		t.Fatalf("write field: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://form.example.test/", strings.NewReader(body.String()))
	req.Host = "form.example.test"
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	responder.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Body.String(); got != "name=from-multipart" {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestResponderQueryTakesPrecedenceOverBody(t *testing.T) {
	responder := testResponder(t, Route{
		Subdomain:        "json",
		ResponseFormat:   "application/json",
		ResponseTemplate: `{"name":"{{name}}"}`,
	})

	req := httptest.NewRequest(http.MethodPost, "http://json.example.test/?name=query", strings.NewReader(`{"name":"body"}`))
	req.Host = "json.example.test"
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	responder.ServeHTTP(rec, req)

	if got := rec.Body.String(); got != `{"name":"query"}` {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestResponderRejectsInvalidJSONBody(t *testing.T) {
	responder := testResponder(t, Route{
		Subdomain:        "json",
		ResponseFormat:   "application/json",
		ResponseTemplate: `{"name":"{{name}}"}`,
	})

	req := httptest.NewRequest(http.MethodPost, "http://json.example.test/", strings.NewReader(`{"name":`))
	req.Host = "json.example.test"
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	responder.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestResponderRejectsInvalidFormBody(t *testing.T) {
	responder := testResponder(t, Route{
		Subdomain:        "form",
		ResponseFormat:   "text/plain",
		ResponseTemplate: "name={{name}}",
	})

	req := httptest.NewRequest(http.MethodPost, "http://form.example.test/", strings.NewReader("%zz"))
	req.Host = "form.example.test"
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	responder.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestResponderRoutesMultipleSubdomainsByHost(t *testing.T) {
	responder := New()
	if err := responder.Register(Route{Subdomain: "json", ResponseFormat: "application/json", ResponseTemplate: `{"type":"json"}`}); err != nil {
		t.Fatalf("register json route: %v", err)
	}
	if err := responder.Register(Route{Subdomain: "yaml", ResponseFormat: "application/yml", ResponseTemplate: "type: yaml\n"}); err != nil {
		t.Fatalf("register yaml route: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://yaml.example.test/", nil)
	req.Host = "yaml.example.test"
	rec := httptest.NewRecorder()

	responder.ServeHTTP(rec, req)

	if got := rec.Body.String(); got != "type: yaml\n" {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestResponderReturnsOKForHealthCheck(t *testing.T) {
	responder := testResponder(t, Route{
		Subdomain:        "json",
		ResponseFormat:   "application/json",
		ResponseTemplate: `{}`,
	})

	req := httptest.NewRequest(http.MethodGet, "http://json.example.test/", nil)
	req.Header.Set("X-Portr-Ping-Request", "true")
	rec := httptest.NewRecorder()

	responder.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty health check body, got %q", rec.Body.String())
	}
}

func testResponder(t *testing.T, route Route) *Responder {
	t.Helper()

	responder := New()
	if err := responder.Register(route); err != nil {
		t.Fatalf("register route: %v", err)
	}
	return responder
}
