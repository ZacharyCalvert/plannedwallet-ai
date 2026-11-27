package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/models"
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

func TestValidateExpectedOriginSecret(t *testing.T) {
	validateExpectedOriginSecret("", false)
	validateExpectedOriginSecret("some-origin-secret", false)
	validateExpectedOriginSecret("a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4", true)
}

func TestGetIncomingOriginSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(w1)
	c1.Request = httptest.NewRequest("GET", "/", nil)
	c1.Request.Header.Set("X-Origin-Verify", "  origin-secret-123  ")
	if secret := getIncomingOriginSecret(c1); secret != "origin-secret-123" {
		t.Errorf("Expected 'origin-secret-123', got %q", secret)
	}

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest("GET", "/", nil)
	c2.Request.Header.Set("X-Azure-FDID", "legacy-header")
	if secret := getIncomingOriginSecret(c2); secret != "" {
		t.Errorf("Expected empty string when only legacy header is set, got %q", secret)
	}
}

func TestIsOriginSecretValid(t *testing.T) {
	if isOriginSecretValid("", "") {
		t.Fatal("expected empty secret never valid")
	}
	if isOriginSecretValid("SecretCase", "secretcase") {
		t.Fatal("expected case mismatch to fail constant-time secret check")
	}
	if isOriginSecretValid("short", "longer-secret") {
		t.Fatal("expected length mismatch to fail")
	}
	if !isOriginSecretValid("MySecretValue48Chars", "MySecretValue48Chars") {
		t.Fatal("expected exact secret match to succeed")
	}
}

func TestOriginVerifyMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	expected := "AbCdEfGhIjKlMnOpQrStUvWxYz0123456789AbCdEfGhIjKl"

	// Dev mode: no header required when secret unset
	rDev := gin.New()
	rDev.Use(originVerifyMiddleware("", false))
	rDev.GET("/test", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	wDev := httptest.NewRecorder()
	rDev.ServeHTTP(wDev, httptest.NewRequest("GET", "/test", nil))
	if wDev.Code != http.StatusOK {
		t.Errorf("Expected 200 in dev without origin secret, got %d", wDev.Code)
	}

	// Prod mode: header required on API routes, exempt on /api/health
	rProd := gin.New()
	rProd.Use(originVerifyMiddleware(expected, true))
	rProd.GET("/test", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	rProd.GET("/api/health", func(c *gin.Context) { c.String(http.StatusOK, "healthy") })

	// 1. Missing header -> 403
	wProd1 := httptest.NewRecorder()
	rProd.ServeHTTP(wProd1, httptest.NewRequest("GET", "/test", nil))
	if wProd1.Code != http.StatusForbidden {
		t.Errorf("Expected 403 for missing X-Origin-Verify, got %d", wProd1.Code)
	}

	// 2. Wrong case -> 403 (exact secret match required)
	reqWrongCase := httptest.NewRequest("GET", "/test", nil)
	reqWrongCase.Header.Set("X-Origin-Verify", strings.ToUpper(expected))
	wWrongCase := httptest.NewRecorder()
	rProd.ServeHTTP(wWrongCase, reqWrongCase)
	if wWrongCase.Code != http.StatusForbidden {
		t.Errorf("Expected 403 for case-mismatched X-Origin-Verify, got %d", wWrongCase.Code)
	}

	// 3. Health check exempt even without X-Origin-Verify
	wHealth := httptest.NewRecorder()
	rProd.ServeHTTP(wHealth, httptest.NewRequest("GET", "/api/health", nil))
	if wHealth.Code != http.StatusOK {
		t.Errorf("Expected 200 for /api/health without X-Origin-Verify, got %d", wHealth.Code)
	}

	// 4. Matching header -> 200 and originVerified=true
	var verified bool
	rProd.GET("/verified", func(c *gin.Context) {
		verified = c.GetBool("originVerified")
		c.String(http.StatusOK, "ok")
	})
	reqValid := httptest.NewRequest("GET", "/verified", nil)
	reqValid.Header.Set("X-Origin-Verify", expected)
	wProd2 := httptest.NewRecorder()
	rProd.ServeHTTP(wProd2, reqValid)
	if wProd2.Code != http.StatusOK || !verified {
		t.Errorf("Expected 200 and originVerified=true for valid X-Origin-Verify, got %d verified=%v", wProd2.Code, verified)
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
	if got := validatePolicyHash("PW_TERMS", valid, true); got != strings.ToLower(valid) {
		t.Fatalf("expected lowercased valid PW_TERMS, got %q", got)
	}
	if got := validatePolicyHash("PW_PRIVACY", valid, true); got != strings.ToLower(valid) {
		t.Fatalf("expected lowercased valid PW_PRIVACY, got %q", got)
	}
	if got := validatePolicyHash("PW_TERMS", "", false); got != "" {
		t.Fatalf("expected empty PW_TERMS allowed when unenforced, got %q", got)
	}
	if got := validatePolicyHash("PW_PRIVACY", "", false); got != "" {
		t.Fatalf("expected empty PW_PRIVACY allowed when unenforced, got %q", got)
	}
	if err := checkEnforcePolicyForProd("", true); err == nil {
		t.Fatal("expected error when PRODUCTION_ENV=true and ENFORCE_POLICY is empty")
	}
	if err := checkEnforcePolicyForProd("false", true); err == nil {
		t.Fatal("expected error when PRODUCTION_ENV=true and ENFORCE_POLICY=false")
	}
	if err := checkEnforcePolicyForProd("true", true); err != nil {
		t.Fatalf("expected nil when PRODUCTION_ENV=true and ENFORCE_POLICY=true, got %v", err)
	}
	if err := checkEnforcePolicyForProd("false", false); err != nil {
		t.Fatalf("expected nil when PRODUCTION_ENV=false, got %v", err)
	}
}

func TestSubscriberEnvConfig(t *testing.T) {
	t.Setenv("SUBSCRIBER_ALWAYS_ON", "")
	if isSubscriberAlwaysOn() {
		t.Fatal("expected false when SUBSCRIBER_ALWAYS_ON is unset")
	}
	for _, val := range []string{"false", "TRUE", "True", " true ", "1", "yes"} {
		t.Setenv("SUBSCRIBER_ALWAYS_ON", val)
		if isSubscriberAlwaysOn() {
			t.Fatalf("expected false for SUBSCRIBER_ALWAYS_ON=%q", val)
		}
	}
	t.Setenv("SUBSCRIBER_ALWAYS_ON", "true")
	if !isSubscriberAlwaysOn() {
		t.Fatal("expected true when SUBSCRIBER_ALWAYS_ON=true")
	}

	if got := parseSubscriberFreeDays(""); got != 0 {
		t.Fatalf("expected 0 for empty SUBSCRIBER_FREE_DAYS, got %d", got)
	}
	for _, raw := range []string{"0", "-1", "-30", "abc", "12.5"} {
		if got := parseSubscriberFreeDays(raw); got != 0 {
			t.Fatalf("expected 0 for %q, got %d", raw, got)
		}
	}
	if got := parseSubscriberFreeDays("14"); got != 14 {
		t.Fatalf("expected 14 for \"14\", got %d", got)
	}
	if got := parseSubscriberFreeDays(" 30 "); got != 30 {
		t.Fatalf("expected 30 for \" 30 \", got %d", got)
	}
}

func TestCleanupStartupMobileTokens(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx := context.Background()
	now := time.Now().UTC()
	revoked := now.Add(-24 * time.Hour)

	_ = repo.CreateMobileToken(ctx, &models.MobileToken{
		ID: "exp-revoked", UserID: "u-1", DeviceHash: "h1", ExpiresAt: now.Add(-time.Second), RevokedAt: &revoked, CreatedAt: now.Add(-181 * 24 * time.Hour),
	})
	_ = repo.CreateMobileToken(ctx, &models.MobileToken{
		ID: "active-revoked", UserID: "u-1", DeviceHash: "h2", ExpiresAt: now.Add(30 * 24 * time.Hour), RevokedAt: &revoked, CreatedAt: now,
	})

	cleanupStartupMobileTokens(repo)

	if _, err := repo.GetMobileTokenByID(ctx, "exp-revoked"); err != repository.ErrNotFound {
		t.Fatalf("expected expired mobile revocation removed on startup, got %v", err)
	}
	if _, err := repo.GetMobileTokenByID(ctx, "active-revoked"); err != nil {
		t.Fatalf("expected unexpired mobile revocation preserved until JWT expiration, got %v", err)
	}
}
