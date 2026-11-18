package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/trace"
)

// SetReadOnly toggles read-only maintenance enforcement and the read_only feature flag.
func (h *Handler) SetReadOnly(readOnly bool) {
	h.readOnly = readOnly
}

// IsReadOnly reports whether the backend is operating in read-only maintenance mode.
func (h *Handler) IsReadOnly() bool {
	return h.readOnly
}

func isWriteMethod(method string) bool {
	return method == http.MethodPost ||
		method == http.MethodPut ||
		method == http.MethodDelete ||
		method == http.MethodPatch
}

func trimSingleSegmentPrefix(trimmed, suffix string) bool {
	if !strings.HasSuffix(trimmed, suffix) {
		return false
	}
	username := strings.TrimSuffix(trimmed, suffix)
	return username != "" && !strings.Contains(username, "/")
}

func isLoginTokenPath(method, path string) bool {
	if method != http.MethodPost || !strings.HasPrefix(path, "/api/v1/users/") {
		return false
	}
	trimmed := strings.TrimPrefix(path, "/api/v1/users/")
	return trimSingleSegmentPrefix(trimmed, "/token") || trimSingleSegmentPrefix(trimmed, "/token/refresh")
}

// ReadOnlyMiddleware rejects state-mutating HTTP methods with 503 Service Unavailable
// while allowing credential login via POST /api/v1/users/{username}/token.
func (h *Handler) ReadOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.IsReadOnly() || !isWriteMethod(c.Request.Method) {
			c.Next()
			return
		}
		if isLoginTokenPath(c.Request.Method, c.Request.URL.Path) {
			c.Next()
			return
		}
		trace.LogInfo(
			c.Request.Context(),
			"Blocked write operation %s %s due to PW_READONLY maintenance mode",
			c.Request.Method,
			c.Request.URL.Path,
		)
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Error:     "READ_ONLY_MODE",
			Message:   "System is undergoing maintenance and is currently in read-only mode.",
			Timestamp: time.Now().UTC(),
		})
	}
}

// GetFeatures handles GET /api/v1/features and returns enabled backend feature flags.
func (h *Handler) GetFeatures(c *gin.Context) {
	features := []string{}
	if h.IsReadOnly() {
		features = append(features, models.FeatureReadOnly)
	}
	c.JSON(http.StatusOK, models.FeaturesResponse{Features: features})
}
