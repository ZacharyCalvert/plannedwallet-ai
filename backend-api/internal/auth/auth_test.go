package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/plannedwallet/internal/auth"
)

func TestGenerateAndValidateToken_SubscriberTrue(t *testing.T) {
	svc := auth.NewService("test-secret", 3600)
	token, err := svc.GenerateToken("user-123", "alice", true)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if claims.UserID != "user-123" || claims.Username != "alice" || !claims.Subscriber {
		t.Errorf("unexpected claims: %+v", claims)
	}
}

func TestGenerateAndValidateToken_SubscriberFalse(t *testing.T) {
	svc := auth.NewService("test-secret", 3600)
	token, err := svc.GenerateToken("user-456", "bob", false)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if claims.UserID != "user-456" || claims.Username != "bob" || claims.Subscriber {
		t.Errorf("unexpected claims: %+v", claims)
	}
}

func TestAuthMiddlewareSetsContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := auth.NewService("test-secret", 3600)
	token, _ := svc.GenerateToken("user-789", "carol", true)

	r := gin.New()
	r.Use(svc.Middleware())
	r.GET("/protected", func(c *gin.Context) {
		uid, _ := c.Get("userID")
		uname, _ := c.Get("username")
		sub, _ := c.Get("subscriber")
		c.JSON(http.StatusOK, gin.H{"uid": uid, "uname": uname, "sub": sub})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
