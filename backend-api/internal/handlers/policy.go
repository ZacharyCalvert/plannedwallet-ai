package handlers

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/plannedwallet/internal/models"
)

func isPolicyExemptPath(method, path string) bool {
	if !strings.HasPrefix(path, "/api/v1/users/") {
		return false
	}
	trimmed := strings.TrimPrefix(path, "/api/v1/users/")
	if method == http.MethodPost && trimSingleSegmentPrefix(trimmed, "/token/refresh") {
		return true
	}
	return (method == http.MethodPost || method == http.MethodGet) &&
		trimSingleSegmentPrefix(trimmed, "/policy-agreements")
}

func abortPolicyApprovalRequired(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusForbidden, models.ErrorResponse{
		Error:     "POLICY_APPROVAL_REQUIRED",
		Message:   message,
		Timestamp: time.Now().UTC(),
	})
}

// PolicyEnforcementMiddleware ensures authenticated callers have an active, current policy agreement
// and matching terms_approved claim whenever policy enforcement is enabled.
func (h *Handler) PolicyEnforcementMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.ctrl.IsEnforcePolicy() || isPolicyExemptPath(c.Request.Method, c.Request.URL.Path) {
			c.Next()
			return
		}
		termsApproved, _ := c.Get("termsApproved")
		if approvedClaim, ok := termsApproved.(bool); !ok || !approvedClaim {
			abortPolicyApprovalRequired(c, "Policy approval is required before accessing data.")
			return
		}
		userID, ok := h.getUserID(c)
		if !ok {
			c.Abort()
			return
		}
		approved, err := h.ctrl.IsUserPolicyApproved(c.Request.Context(), userID)
		if err != nil {
			h.sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to verify policy agreement.")
			c.Abort()
			return
		}
		if !approved {
			abortPolicyApprovalRequired(c, "Your policy agreement is inactive or outdated. Please accept the current terms.")
			return
		}
		c.Next()
	}
}

func parseHeaderClientIP(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if ip := net.ParseIP(trimmed); ip != nil {
		return ip.String()
	}
	if host, _, err := net.SplitHostPort(trimmed); err == nil {
		if ip := net.ParseIP(strings.TrimSpace(host)); ip != nil {
			return ip.String()
		}
	}
	return ""
}

func resolveClientIP(c *gin.Context) string {
	if c.GetBool("originVerified") {
		if ip := parseHeaderClientIP(c.GetHeader("CloudFront-Viewer-Address")); ip != "" {
			return ip
		}
	}
	return c.ClientIP()
}

// RecordPolicyAgreement handles POST /api/v1/users/{username}/policy-agreements
func (h *Handler) RecordPolicyAgreement(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	username := strings.TrimSpace(c.Param("username"))
	if username == "" {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", "Username path parameter is required.")
		return
	}

	var req models.PolicyAgreementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	res, err := h.ctrl.RecordPolicyAgreement(c.Request.Context(), userID, username, resolveClientIP(c), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, res)
}

// ListPolicyAgreements handles GET /api/v1/users/{username}/policy-agreements
func (h *Handler) ListPolicyAgreements(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	username := strings.TrimSpace(c.Param("username"))
	if username == "" {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", "Username path parameter is required.")
		return
	}

	res, err := h.ctrl.ListPolicyAgreements(c.Request.Context(), userID, username)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}
