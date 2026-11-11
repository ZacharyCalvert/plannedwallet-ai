package cors

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCleanHost(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"www.plannedwallet.com", "www.plannedwallet.com"},
		{"local.plannedwallet.com:8080", "local.plannedwallet.com"},
		{"https://dev.plannedwallet.com:443", "dev.plannedwallet.com"},
		{"http://stage.plannedwallet.com", "stage.plannedwallet.com"},
		{"  LOCALHOST.PLANNEDWALLET.COM:3000 ", "localhost.plannedwallet.com"},
	}

	for _, c := range cases {
		if got := CleanHost(c.input); got != c.expected {
			t.Errorf("CleanHost(%q) = %q; expected %q", c.input, got, c.expected)
		}
	}
}

func TestLoadAllowedClients_Default(t *testing.T) {
	_ = os.Unsetenv("CORS_ALLOWED_CLIENTS")
	_ = os.Unsetenv("ALLOWED_CLIENTS")
	_ = os.Unsetenv("CORS_ALLOWED_ORIGINS")

	clients := LoadAllowedClients()
	if len(clients) != 5 {
		t.Fatalf("expected 5 default allowed clients, got %d", len(clients))
	}
	expected := []string{
		"www.plannedwallet.com",
		"local.plannedwallet.com",
		"localhost.plannedwallet.com",
		"stage.plannedwallet.com",
		"dev.plannedwallet.com",
	}
	for i, exp := range expected {
		if clients[i] != exp {
			t.Errorf("expected client %d to be %q, got %q", i, exp, clients[i])
		}
	}
}

func TestLoadAllowedClients_EnvOverride(t *testing.T) {
	_ = os.Setenv("CORS_ALLOWED_CLIENTS", "custom.domain.com, other.domain.org:8080")
	defer func() { _ = os.Unsetenv("CORS_ALLOWED_CLIENTS") }()

	clients := LoadAllowedClients()
	if len(clients) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(clients))
	}
	if clients[0] != "custom.domain.com" || clients[1] != "other.domain.org" {
		t.Errorf("unexpected clients: %+v", clients)
	}
}

func setupCorsTestEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Middleware())
	r.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	return r
}

func TestPreflight_AllowedDefaultDomains(t *testing.T) {
	_ = os.Unsetenv("CORS_ALLOWED_CLIENTS")
	_ = os.Unsetenv("ALLOWED_CLIENTS")
	r := setupCorsTestEngine()

	allowedDomains := []string{
		"www.plannedwallet.com",
		"local.plannedwallet.com",
		"localhost.plannedwallet.com",
		"stage.plannedwallet.com",
		"dev.plannedwallet.com",
	}

	for _, domain := range allowedDomains {
		req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
		req.Host = domain
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("domain %s: expected 204, got %d", domain, w.Code)
		}
		if w.Header().Get("Access-Control-Allow-Origin") == "" {
			t.Errorf("domain %s: expected Access-Control-Allow-Origin header", domain)
		}
	}
}

func TestPreflight_DisallowedDomain(t *testing.T) {
	_ = os.Unsetenv("CORS_ALLOWED_CLIENTS")
	_ = os.Unsetenv("ALLOWED_CLIENTS")
	r := setupCorsTestEngine()

	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Host = "evil.com"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for evil.com, got %d", w.Code)
	}
}

func TestPreflight_OriginHeader(t *testing.T) {
	_ = os.Unsetenv("CORS_ALLOWED_CLIENTS")
	r := setupCorsTestEngine()

	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Host = "local.plannedwallet.com"
	req.Header.Set("Origin", "http://local.plannedwallet.com:3000")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "http://local.plannedwallet.com:3000" {
		t.Errorf("expected reflected origin, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestActualRequest_WithOrigin(t *testing.T) {
	_ = os.Unsetenv("CORS_ALLOWED_CLIENTS")
	r := setupCorsTestEngine()

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "https://dev.plannedwallet.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "https://dev.plannedwallet.com" {
		t.Errorf("expected Access-Control-Allow-Origin header set on actual request")
	}
}
