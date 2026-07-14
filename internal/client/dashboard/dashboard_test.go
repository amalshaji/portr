package dashboard

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amalshaji/portr/internal/client/config"
)

func TestEmbeddedDashboardBundle(t *testing.T) {
	if _, err := fs.Stat(dashboardStaticFS, "index.html"); err != nil {
		t.Fatalf("embedded bundle does not contain index.html: %v", err)
	}

	entries, err := fs.ReadDir(dashboardStaticFS, "assets")
	if err != nil {
		t.Fatalf("read embedded assets: %v", err)
	}

	foundJavaScript := false
	foundCSS := false
	for _, entry := range entries {
		foundJavaScript = foundJavaScript || strings.HasSuffix(entry.Name(), ".js")
		foundCSS = foundCSS || strings.HasSuffix(entry.Name(), ".css")
	}
	if !foundJavaScript || !foundCSS {
		t.Fatalf("embedded assets missing representative bundle files: javascript=%t css=%t", foundJavaScript, foundCSS)
	}
}

func TestDashboardServesEmbeddedStaticAssetsAndSPAFallback(t *testing.T) {
	dashboard := New(nil, &config.Config{})

	asset := firstAssetWithSuffix(t, ".js")
	response, err := dashboard.app.Test(httptest.NewRequest(http.MethodGet, "/static/assets/"+asset, nil))
	if err != nil {
		t.Fatalf("request embedded asset: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("embedded asset status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	for _, path := range []string{"/", "/example-tunnel"} {
		response, err := dashboard.app.Test(httptest.NewRequest(http.MethodGet, path, nil))
		if err != nil {
			t.Fatalf("request %s: %v", path, err)
		}
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s status = %d, want %d", path, response.StatusCode, http.StatusOK)
		}
	}
}

func firstAssetWithSuffix(t *testing.T, suffix string) string {
	t.Helper()
	entries, err := fs.ReadDir(dashboardStaticFS, "assets")
	if err != nil {
		t.Fatalf("read embedded assets: %v", err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), suffix) {
			return entry.Name()
		}
	}
	t.Fatalf("embedded assets do not contain a %s file", suffix)
	return ""
}
