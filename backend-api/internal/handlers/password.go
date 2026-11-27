package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/plannedwallet/internal/models"
)

// ChangePassword handles POST /api/v1/users/{username}/password
func (h *Handler) ChangePassword(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	username := strings.TrimSpace(c.Param("username"))
	if username == "" {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", "Username path parameter is required.")
		return
	}

	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", "Old password, new password, confirmation, and totp_code are required.")
		return
	}

	res, err := h.ctrl.ChangePassword(c.Request.Context(), userID, username, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
