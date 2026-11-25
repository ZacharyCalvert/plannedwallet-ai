package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/plannedwallet/internal/models"
)

// GetUserSettings handles GET /api/v1/users/:username/settings
func (h *Handler) GetUserSettings(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	username := c.Param("username")

	res, err := h.ctrl.GetUserSettings(c.Request.Context(), userID, username)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// PatchUserSettings handles PATCH /api/v1/users/:username/settings
func (h *Handler) PatchUserSettings(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	username := c.Param("username")
	var req models.PatchUserSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON request body.")
		return
	}

	res, err := h.ctrl.PatchUserSettings(c.Request.Context(), userID, username, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
