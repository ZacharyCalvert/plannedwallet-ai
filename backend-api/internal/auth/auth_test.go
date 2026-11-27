package auth_test

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/plannedwallet/internal/auth"
)

func TestDefaultExpiration_FifteenMinutesAndUniqueJTI(t *testing.T) {
	svc := auth.NewService("test-secret", 0)
	if svc.ExpirationSeconds() != 900 {
		t.Fatalf("expected default expiration 900s (15m), got %d", svc.ExpirationSeconds())
	}

	tok1, err := svc.GenerateToken("user-1", "alice", false, true)
	if err != nil {
		t.Fatalf("GenerateToken 1 failed: %v", err)
	}
	tok2, err := svc.GenerateToken("user-1", "alice", false, true)
	if err != nil {
		t.Fatalf("GenerateToken 2 failed: %v", err)
	}
	if tok1 == tok2 {
		t.Fatal("expected distinct tokens due to unique jti claims")
	}

	claims, err := svc.ValidateToken(tok1)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time) != 15*time.Minute {
		t.Fatalf("expected 15m TTL, got %v", claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time))
	}
}

func TestGenerateAndValidateToken_SubscriberTrue(t *testing.T) {
	svc := auth.NewService("test-secret", 3600)
	token, err := svc.GenerateToken("user-123", "alice", true, true)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if claims.UserID != "user-123" || claims.Username != "alice" || !claims.Subscriber || !claims.TermsApproved {
		t.Errorf("unexpected claims: %+v", claims)
	}
}

func TestGenerateAndValidateToken_SubscriberFalse(t *testing.T) {
	svc := auth.NewService("test-secret", 3600)
	token, err := svc.GenerateToken("user-456", "bob", false, false)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if claims.UserID != "user-456" || claims.Username != "bob" || claims.Subscriber || claims.TermsApproved {
		t.Errorf("unexpected claims: %+v", claims)
	}
}

func TestAuthMiddlewareSetsContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := auth.NewService("test-secret", 3600)
	token, _ := svc.GenerateToken("user-789", "carol", true, false)

	r := gin.New()
	r.Use(svc.Middleware())
	r.GET("/protected", func(c *gin.Context) {
		uid, _ := c.Get("userID")
		uname, _ := c.Get("username")
		sub, _ := c.Get("subscriber")
		terms, _ := c.Get("termsApproved")
		c.JSON(http.StatusOK, gin.H{"uid": uid, "uname": uname, "sub": sub, "terms": terms})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Body.String(); !strings.Contains(got, `"terms":false`) {
		t.Fatalf("expected termsApproved false in context response, got %s", got)
	}
}

func TestValidateToken_RejectsForgeryAndAlgorithmConfusion(t *testing.T) {
	svc := auth.NewService("test-secret-with-32-bytes-minimum!!", 900)
	validTok, err := svc.GenerateToken("user-1", "alice", false, true)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	parts := strings.Split(validTok, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 JWT segments, got %d", len(parts))
	}

	noneHeader := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	if _, err := svc.ValidateToken(noneHeader + "." + parts[1] + "."); err == nil {
		t.Fatal("expected alg:none token to be rejected")
	}
	rsHeader := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	if _, err := svc.ValidateToken(rsHeader + "." + parts[1] + "." + parts[2]); err == nil {
		t.Fatal("expected RS256 header swap token to be rejected")
	}
	tamperedPayload := base64.RawURLEncoding.EncodeToString([]byte(`{"user_id":"user-1","username":"alice","subscriber":true,"terms_approved":true}`))
	if _, err := svc.ValidateToken(parts[0] + "." + tamperedPayload + "." + parts[2]); err == nil {
		t.Fatal("expected tampered payload token to be rejected")
	}
	wrongSigner := auth.NewService("different-secret-32-bytes-minimum!", 900)
	forged, _ := wrongSigner.GenerateToken("user-1", "alice", false, true)
	if _, err := svc.ValidateToken(forged); err == nil {
		t.Fatal("expected token signed with wrong secret to be rejected")
	}
}

func TestMobileToken_LimitationScopeAndDeviceBindingMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := auth.NewService("test-secret-with-32-bytes-minimum!!", 900)
	deviceID := "device-install-uuid-1234"
	deviceHash := auth.HashMobileIdentifier(deviceID)

	stdTok, err := svc.GenerateToken("user-1", "alice", false, true)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	mobTok, exp, err := svc.GenerateMobileToken("tok-mob-1", "user-1", "alice", false, true, deviceHash)
	if err != nil {
		t.Fatalf("GenerateMobileToken failed: %v", err)
	}
	claims, err := svc.ValidateToken(mobTok)
	if err != nil {
		t.Fatalf("ValidateToken mobile failed: %v", err)
	}
	if claims.Limitation != auth.LimitationMobile || claims.DeviceHash != deviceHash {
		t.Fatalf("unexpected mobile claims: %+v", claims)
	}
	if ttl := exp.Sub(claims.IssuedAt.Time); ttl < 179*24*time.Hour || ttl > 181*24*time.Hour {
		t.Fatalf("expected ~180 day TTL, got %v", ttl)
	}

	r := gin.New()
	r.GET("/standard", svc.Middleware(), func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/mobile", svc.MobileMiddleware(), func(c *gin.Context) {
		tid, _ := c.Get("tokenID")
		c.JSON(http.StatusOK, gin.H{"tokenID": tid})
	})

	// Mobile token must be blocked on standard routes
	reqStd := httptest.NewRequest(http.MethodGet, "/standard", nil)
	reqStd.Header.Set("Authorization", "Bearer "+mobTok)
	wStd := httptest.NewRecorder()
	r.ServeHTTP(wStd, reqStd)
	if wStd.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for mobile token on standard route, got %d", wStd.Code)
	}

	// Standard token must be blocked on mobile routes
	reqMobWithStd := httptest.NewRequest(http.MethodGet, "/mobile", nil)
	reqMobWithStd.Header.Set("Authorization", "Bearer "+stdTok)
	reqMobWithStd.Header.Set("X-Mobile-Identifier", deviceID)
	wMobStd := httptest.NewRecorder()
	r.ServeHTTP(wMobStd, reqMobWithStd)
	if wMobStd.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for standard token on mobile route, got %d", wMobStd.Code)
	}

	// Missing or mismatched X-Mobile-Identifier rejected with 401
	for _, hdr := range []string{"", "wrong-device-id"} {
		req := httptest.NewRequest(http.MethodGet, "/mobile", nil)
		req.Header.Set("Authorization", "Bearer "+mobTok)
		if hdr != "" {
			req.Header.Set("X-Mobile-Identifier", hdr)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for mobile identifier %q, got %d", hdr, w.Code)
		}
	}

	// Matching device identifier succeeds
	reqOK := httptest.NewRequest(http.MethodGet, "/mobile", nil)
	reqOK.Header.Set("Authorization", "Bearer "+mobTok)
	reqOK.Header.Set("X-Mobile-Identifier", deviceID)
	wOK := httptest.NewRecorder()
	r.ServeHTTP(wOK, reqOK)
	if wOK.Code != http.StatusOK || !strings.Contains(wOK.Body.String(), "tok-mob-1") {
		t.Fatalf("expected 200 with tokenID, got %d: %s", wOK.Code, wOK.Body.String())
	}
}
