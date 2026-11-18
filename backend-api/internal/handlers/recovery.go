package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/plannedwallet/internal/models"
)

// RecoverAccount handles POST /api/v1/users/{username}/recover
func (h *Handler) RecoverAccount(c *gin.Context) {
	username := strings.TrimSpace(c.Param("username"))
	if username == "" {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", "Username path parameter is required.")
		return
	}

	var req models.AccountRecoveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	res, err := h.ctrl.RecoverAccount(c.Request.Context(), username, resolveClientIP(c), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// RotateRecoveryKey handles POST /api/v1/users/{username}/recovery-key/rotate
func (h *Handler) RotateRecoveryKey(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	username := strings.TrimSpace(c.Param("username"))
	if username == "" {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", "Username path parameter is required.")
		return
	}

	var req models.RotateRecoveryKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", "Password and totp_code are required.")
		return
	}

	res, err := h.ctrl.RotateRecoveryKey(c.Request.Context(), userID, username, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}
