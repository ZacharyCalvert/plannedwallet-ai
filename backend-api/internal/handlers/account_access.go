package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/plannedwallet/internal/models"
)

// GrantAccountAccess handles POST /api/v1/accounts/:accountId/access
func (h *Handler) GrantAccountAccess(c *gin.Context) {
	adminUserID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := c.Param("accountId")

	var req models.AccountAccessGrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	res, err := h.ctrl.GrantAccountAccess(c.Request.Context(), adminUserID, accountID, req.UserID, req.AccessLevel)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetAccountAccess handles GET /api/v1/accounts/:accountId/access/:userId
func (h *Handler) GetAccountAccess(c *gin.Context) {
	reqUserID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := c.Param("accountId")
	targetUserID := c.Param("userId")

	res, err := h.ctrl.GetAccountAccess(c.Request.Context(), reqUserID, accountID, targetUserID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// RevokeAccountAccess handles DELETE /api/v1/accounts/:accountId/access/:userId
func (h *Handler) RevokeAccountAccess(c *gin.Context) {
	adminUserID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := c.Param("accountId")
	targetUserID := c.Param("userId")

	if err := h.ctrl.RevokeAccountAccess(c.Request.Context(), adminUserID, accountID, targetUserID); err != nil {
		h.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
