package fileresponder

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func testResponder(t *testing.T, routes ...Route) *Responder {
	t.Helper()

	responder := New()
	for _, route := range routes {
		if err := responder.Register(route); err != nil {
			t.Fatalf("register %q: %v", route.Subdomain, err)
		}
	}
	t.Cleanup(func() { _ = responder.Shutdown(context.Background()) })
	return responder
}

func writeFile(t *testing.T, dir, name, body string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func get(t *testing.T, responder *Responder, host, path string) *http.Response {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Host = host
	recorder := httptest.NewRecorder()
	responder.ServeHTTP(recorder, request)
	return recorder.Result()
}

func bodyOf(t *testing.T, response *http.Response) string {
	t.Helper()
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(body)
}

func TestResponderServesIndexFromDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.html", "<h1>portr</h1>")
	responder := testResponder(t, Route{Subdomain: "site", Dir: dir})

	response := get(t, responder, "site.example.test", "/")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}
	if body := bodyOf(t, response); body != "<h1>portr</h1>" {
		t.Fatalf("unexpected body %q", body)
	}
}

func TestResponderServesNestedFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "assets/app.js", "console.log(1)")
	responder := testResponder(t, Route{Subdomain: "site", Dir: dir})

	response := get(t, responder, "site.example.test", "/assets/app.js")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}
	if body := bodyOf(t, response); body != "console.log(1)" {
		t.Fatalf("unexpected body %q", body)
	}
}

func TestResponderListsDirectoryWithoutIndex(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "a")
	responder := testResponder(t, Route{Subdomain: "site", Dir: dir})

	response := get(t, responder, "site.example.test", "/")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}
	if body := bodyOf(t, response); !strings.Contains(body, "a.txt") {
		t.Fatalf("expected a directory listing, got %q", body)
	}
}

func TestResponderRejectsParentTraversal(t *testing.T) {
	parent := t.TempDir()
	if err := os.WriteFile(filepath.Join(parent, "secret.txt"), []byte("classified"), 0o644); err != nil {
		t.Fatalf("write secret: %v", err)
	}
	dir := filepath.Join(parent, "public")
	writeFile(t, dir, "index.html", "ok")
	responder := testResponder(t, Route{Subdomain: "site", Dir: dir})

	response := get(t, responder, "site.example.test", "/../secret.txt")
	if body := bodyOf(t, response); strings.Contains(body, "classified") {
		t.Fatalf("traversal escaped the served directory: %q", body)
	}
}

func TestResponderRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on windows")
	}

	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("classified"), 0o644); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	dir := t.TempDir()
	writeFile(t, dir, "index.html", "ok")
	if err := os.Symlink(secret, filepath.Join(dir, "escape")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	responder := testResponder(t, Route{Subdomain: "site", Dir: dir})

	// This is the assertion that justifies os.OpenRoot over http.Dir, which
	// follows symlinks out of the served tree.
	response := get(t, responder, "site.example.test", "/escape")
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for an escaping symlink, got %d: %q", response.StatusCode, bodyOf(t, response))
	}
}

func TestResponderRoutesMultipleSubdomainsByHost(t *testing.T) {
	dirA := t.TempDir()
	writeFile(t, dirA, "index.html", "A")
	dirB := t.TempDir()
	writeFile(t, dirB, "index.html", "B")

	responder := testResponder(t,
		Route{Subdomain: "a", Dir: dirA},
		Route{Subdomain: "b", Dir: dirB},
	)

	if body := bodyOf(t, get(t, responder, "a.example.test", "/")); body != "A" {
		t.Fatalf("expected A, got %q", body)
	}
	if body := bodyOf(t, get(t, responder, "b.example.test", "/")); body != "B" {
		t.Fatalf("expected B, got %q", body)
	}
}

func TestResponderReturnsOKForHealthCheck(t *testing.T) {
	responder := testResponder(t)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Portr-Ping-Request", "true")
	recorder := httptest.NewRecorder()
	responder.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for a ping, got %d", recorder.Code)
	}
}

func TestResponderNotFoundForUnknownHost(t *testing.T) {
	dirA := t.TempDir()
	writeFile(t, dirA, "index.html", "A")
	dirB := t.TempDir()
	writeFile(t, dirB, "index.html", "B")

	// Two routes, so the single-route fallback cannot mask the miss.
	responder := testResponder(t,
		Route{Subdomain: "a", Dir: dirA},
		Route{Subdomain: "b", Dir: dirB},
	)

	response := get(t, responder, "nope.example.test", "/")
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for an unknown host, got %d", response.StatusCode)
	}
}

func TestRegisterRejectsDuplicateSubdomain(t *testing.T) {
	dir := t.TempDir()
	responder := testResponder(t, Route{Subdomain: "site", Dir: dir})

	err := responder.Register(Route{Subdomain: "site", Dir: dir})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected a duplicate error, got %v", err)
	}
}

func TestRegisterRejectsNonDirectory(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "not-a-dir.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	responder := New()
	t.Cleanup(func() { _ = responder.Shutdown(context.Background()) })

	if err := responder.Register(Route{Subdomain: "site", Dir: file}); err == nil {
		t.Fatal("expected an error when the dir is a regular file")
	}
}

func TestRegisterRejectsMissingFields(t *testing.T) {
	responder := New()
	t.Cleanup(func() { _ = responder.Shutdown(context.Background()) })

	if err := responder.Register(Route{Dir: t.TempDir()}); err == nil {
		t.Fatal("expected an error for a missing subdomain")
	}
	if err := responder.Register(Route{Subdomain: "site"}); err == nil {
		t.Fatal("expected an error for a missing dir")
	}
}
