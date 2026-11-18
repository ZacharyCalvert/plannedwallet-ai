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
