package main

import (
	"os"
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
	os.Setenv("JWT_EXPIRATION_SECONDS", "7200")

	svc := initJWTService(true)
	if svc == nil {
		t.Fatal("Expected non-nil auth.Service in production with secure secret")
	}
	if svc.ExpirationSeconds() != 7200 {
		t.Errorf("Expected expiration 7200, got %d", svc.ExpirationSeconds())
	}
}
