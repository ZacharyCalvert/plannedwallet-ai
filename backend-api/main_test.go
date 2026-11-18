package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/repository"
)

func TestIsProduction(t *testing.T) {
	orig := os.Getenv("PRODUCTION_ENV")
	defer os.Setenv("PRODUCTION_ENV", orig)

	testCases := []struct {
		envVal   string
		expected bool
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"false", false},
		{"FALSE", false},
		{"", false},
		{"development", false},
	}

	for _, tc := range testCases {
		os.Setenv("PRODUCTION_ENV", tc.envVal)
		if actual := isProduction(); actual != tc.expected {
			t.Errorf("isProduction() for %q: expected %v, got %v", tc.envVal, tc.expected, actual)
		}
	}
}

func TestIsReadOnlyMode(t *testing.T) {
	orig := os.Getenv("PW_READONLY")
	defer os.Setenv("PW_READONLY", orig)

	testCases := []struct {
		envVal   string
		expected bool
	}{
		{"true", true},
		{"TRUE", true},
		{" true ", true},
		{"false", false},
		{"", false},
		{"1", false},
	}

	for _, tc := range testCases {
		os.Setenv("PW_READONLY", tc.envVal)
		if actual := isReadOnlyMode(); actual != tc.expected {
			t.Errorf("isReadOnlyMode() for %q: expected %v, got %v", tc.envVal, tc.expected, actual)
		}
	}
}

func TestIsDefaultJWTSecret(t *testing.T) {
	defaults := []string{
		defaultJWTSecret,
		"dev-super-secret-jwt-key",
		"",
		"   ",
		"some-super-secret-key",
		"my super secret token",
	}
	for _, s := range defaults {
		if !isDefaultJWTSecret(s) {
			t.Errorf("isDefaultJWTSecret(%q) should be true", s)
		}
	}

	secure := []string{
		"c29tZS1zZWN1cmUtcHJvZHVjdGlvbi1rZXktMjU2LWJpdHM=",
		"k8s-injected-random-production-token-string",
		"production-hmac-key-plannedwallet-ai-prod",
	}
	for _, s := range secure {
		if isDefaultJWTSecret(s) {
			t.Errorf("isDefaultJWTSecret(%q) should be false", s)
		}
	}
}

func TestConfigureProxies(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rDev := gin.New()
	configureProxies(rDev, false)

	rProd := gin.New()
	configureProxies(rProd, true)
}

func TestApplyDatabaseSchemaProductionSkipped(t *testing.T) {
	// Should return immediately without attempting any DB operations when isProd is true
	applyDatabaseSchema(nil, true)
}

func TestAddHandlersCaptchaEnforcement(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := repository.NewMemoryRepository()
	authService := auth.NewService("test-secret-key", 3600)

	orig := os.Getenv("REQUIRE_CAPTCHA")
	defer os.Setenv("REQUIRE_CAPTCHA", orig)

	// 1. In production, captcha is always enabled even if REQUIRE_CAPTCHA is false
	os.Setenv("REQUIRE_CAPTCHA", "false")
	r1 := gin.New()
	addHandlers(r1, repo, authService, true)

	// 2. In dev mode with REQUIRE_CAPTCHA=false, captcha is not forced
	os.Setenv("REQUIRE_CAPTCHA", "false")
	r2 := gin.New()
	addHandlers(r2, repo, authService, false)

	// 3. In dev mode with REQUIRE_CAPTCHA=true, captcha is enabled
	os.Setenv("REQUIRE_CAPTCHA", "true")
	r3 := gin.New()
	addHandlers(r3, repo, authService, false)
}

func TestInitJWTServiceSecureInProduction(t *testing.T) {
	origSecret := os.Getenv("JWT_SECRET")
	origExp := os.Getenv("JWT_EXPIRATION_SECONDS")
	defer func() {
		os.Setenv("JWT_SECRET", origSecret)
		os.Setenv("JWT_EXPIRATION_SECONDS", origExp)
	}()

	os.Setenv("JWT_SECRET", "custom-production-secret-token-key-256")
	os.Unsetenv("JWT_EXPIRATION_SECONDS")
	defaultSvc := initJWTService(true)
	if defaultSvc.ExpirationSeconds() != 900 {
		t.Errorf("Expected default expiration 900, got %d", defaultSvc.ExpirationSeconds())
	}

	os.Setenv("JWT_EXPIRATION_SECONDS", "7200")

	svc := initJWTService(true)
	if svc == nil {
		t.Fatal("Expected non-nil auth.Service in production with secure secret")
	}
	if svc.ExpirationSeconds() != 7200 {
		t.Errorf("Expected expiration 7200, got %d", svc.ExpirationSeconds())
	}
}

func TestCheckJWTSecretBits(t *testing.T) {
	if err := checkJWTSecretBits(""); err != nil {
		t.Errorf("Expected nil error for empty secret, got %v", err)
	}

	shortKeys := []string{
		"a",
		"short-key",
		"1234567890123456789012345678901",
	}
	for _, k := range shortKeys {
		if err := checkJWTSecretBits(k); err == nil {
			t.Errorf("Expected error for secret %q with len %d < 32", k, len(k))
		}
	}

	validKeys := []string{
		"12345678901234567890123456789012",
		"super-secret-jwt-key-replace-in-production",
		"c29tZS1zZWN1cmUtcHJvZHVjdGlvbi1rZXktMjU2LWJpdHM=",
	}
	for _, k := range validKeys {
		if err := checkJWTSecretBits(k); err != nil {
			t.Errorf("Expected nil error for secret with len %d >= 32, got %v", len(k), err)
		}
	}
}

func TestValidateExpectedFDID(t *testing.T) {
	validateExpectedFDID("", false)
	validateExpectedFDID("some-fdid", false)
	validateExpectedFDID("12345678-1234-1234-1234-123456789abc", true)
}

func TestGetIncomingFDID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(w1)
	c1.Request = httptest.NewRequest("GET", "/", nil)
	c1.Request.Header.Set("X-Azure-FDID", "azure-id-123")
	if id := getIncomingFDID(c1); id != "azure-id-123" {
		t.Errorf("Expected 'azure-id-123', got %q", id)
	}

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest("GET", "/", nil)
	c2.Request.Header.Set("X-FDID", "fdid-456")
	if id := getIncomingFDID(c2); id != "fdid-456" {
		t.Errorf("Expected 'fdid-456', got %q", id)
	}

	w3 := httptest.NewRecorder()
	c3, _ := gin.CreateTestContext(w3)
	c3.Request = httptest.NewRequest("GET", "/", nil)
	if id := getIncomingFDID(c3); id != "" {
		t.Errorf("Expected empty string, got %q", id)
	}
}

func TestFDIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	expected := "12345678-abcd-1234-abcd-1234567890ab"

	// Dev mode: no header required
	rDev := gin.New()
	rDev.Use(fdidMiddleware("", false))
	rDev.GET("/test", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	wDev := httptest.NewRecorder()
	rDev.ServeHTTP(wDev, httptest.NewRequest("GET", "/test", nil))
	if wDev.Code != http.StatusOK {
		t.Errorf("Expected 200 in dev without FDID, got %d", wDev.Code)
	}

	// Prod mode: header required
	rProd := gin.New()
	rProd.Use(fdidMiddleware(expected, true))
	rProd.GET("/test", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	// 1. Missing header -> 403
	wProd1 := httptest.NewRecorder()
	rProd.ServeHTTP(wProd1, httptest.NewRequest("GET", "/test", nil))
	if wProd1.Code != http.StatusForbidden {
		t.Errorf("Expected 403 for missing FDID, got %d", wProd1.Code)
	}

	// 2. Matching header -> 200
	reqValid := httptest.NewRequest("GET", "/test", nil)
	reqValid.Header.Set("X-Azure-FDID", expected)
	wProd2 := httptest.NewRecorder()
	rProd.ServeHTTP(wProd2, reqValid)
	if wProd2.Code != http.StatusOK {
		t.Errorf("Expected 200 for valid FDID, got %d", wProd2.Code)
	}

	// 3. Case-insensitive matching header -> 200
	var verified bool
	rProd.GET("/verified", func(c *gin.Context) {
		verified = c.GetBool("azureFDIDVerified")
		c.String(http.StatusOK, "ok")
	})
	reqUpper := httptest.NewRequest("GET", "/verified", nil)
	reqUpper.Header.Set("X-Azure-FDID", strings.ToUpper(expected))
	wProd3 := httptest.NewRecorder()
	rProd.ServeHTTP(wProd3, reqUpper)
	if wProd3.Code != http.StatusOK || !verified {
		t.Errorf("Expected 200 and azureFDIDVerified=true for uppercase FDID, got %d verified=%v", wProd3.Code, verified)
	}
}

func TestIsPolicyEnforcedAndValidateTermsHash(t *testing.T) {
	origEnf := os.Getenv("ENFORCE_POLICY")
	defer func() { _ = os.Setenv("ENFORCE_POLICY", origEnf) }()

	_ = os.Setenv("ENFORCE_POLICY", "false")
	if isPolicyEnforced(false) {
		t.Fatal("expected policy enforcement off when both flags false")
	}
	if !isPolicyEnforced(true) {
		t.Fatal("expected policy enforcement on when PRODUCTION_ENV=true")
	}
	_ = os.Setenv("ENFORCE_POLICY", "true")
	if !isPolicyEnforced(false) {
		t.Fatal("expected policy enforcement on when ENFORCE_POLICY=true")
	}
	valid := "B08F3D8B1CD44544888A78ED13316C0910725944BC46808AB542E36D9B2FD84C"
	if got := validateTermsHash(valid, true); got != strings.ToLower(valid) {
		t.Fatalf("expected lowercased valid PW_TERMS, got %q", got)
	}
	if got := validateTermsHash("", false); got != "" {
		t.Fatalf("expected empty PW_TERMS allowed when unenforced, got %q", got)
	}
}
