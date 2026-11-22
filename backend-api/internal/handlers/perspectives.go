package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/plannedwallet/internal/models"
)

// CreatePerspective handles POST /api/v1/perspectives
func (h *Handler) CreatePerspective(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	var req models.PerspectiveCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	res, err := h.ctrl.CreatePerspective(c.Request.Context(), userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

// ListPerspectives handles GET /api/v1/perspectives
func (h *Handler) ListPerspectives(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	res, err := h.ctrl.ListPerspectives(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetPerspective handles GET /api/v1/perspectives/:perspectiveId
func (h *Handler) GetPerspective(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	pID := strings.TrimSpace(c.Param("perspectiveId"))
	res, err := h.ctrl.GetPerspective(c.Request.Context(), userID, pID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// UpdatePerspective handles PUT /api/v1/perspectives/:perspectiveId
func (h *Handler) UpdatePerspective(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	pID := strings.TrimSpace(c.Param("perspectiveId"))
	var req models.PerspectiveUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	res, err := h.ctrl.UpdatePerspective(c.Request.Context(), userID, pID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// DeletePerspective handles DELETE /api/v1/perspectives/:perspectiveId
func (h *Handler) DeletePerspective(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	pID := strings.TrimSpace(c.Param("perspectiveId"))
	if err := h.ctrl.DeletePerspective(c.Request.Context(), userID, pID); err != nil {
		h.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// GetPerspectiveSettings handles GET /api/v1/perspectives/:perspectiveId/settings
func (h *Handler) GetPerspectiveSettings(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	pID := strings.TrimSpace(c.Param("perspectiveId"))
	res, err := h.ctrl.GetPerspectiveSettings(c.Request.Context(), userID, pID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// UpdatePerspectiveSettings handles PUT /api/v1/perspectives/:perspectiveId/settings
func (h *Handler) UpdatePerspectiveSettings(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	pID := strings.TrimSpace(c.Param("perspectiveId"))
	var req models.PerspectiveSettingsInput
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	res, err := h.ctrl.UpdatePerspectiveSettings(c.Request.Context(), userID, pID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// ListPerspectiveAccounts handles GET /api/v1/perspectives/:perspectiveId/accounts
func (h *Handler) ListPerspectiveAccounts(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	pID := strings.TrimSpace(c.Param("perspectiveId"))
	res, err := h.ctrl.ListPerspectiveAccounts(c.Request.Context(), userID, pID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// LinkPerspectiveAccount handles POST /api/v1/perspectives/:perspectiveId/accounts
func (h *Handler) LinkPerspectiveAccount(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	pID := strings.TrimSpace(c.Param("perspectiveId"))
	var req models.PerspectiveAccountLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	if err := h.ctrl.LinkPerspectiveAccount(c.Request.Context(), userID, pID, req.AccountID); err != nil {
		h.handleError(c, err)
		return
	}
	c.Status(http.StatusCreated)
}

// UnlinkPerspectiveAccount handles DELETE /api/v1/perspectives/:perspectiveId/accounts/:accountId
func (h *Handler) UnlinkPerspectiveAccount(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	pID := strings.TrimSpace(c.Param("perspectiveId"))
	accID := strings.TrimSpace(c.Param("accountId"))
	if err := h.ctrl.UnlinkPerspectiveAccount(c.Request.Context(), userID, pID, accID); err != nil {
		h.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListPerspectiveRecurringTransactions handles GET /api/v1/perspectives/:perspectiveId/recurring-transactions
func (h *Handler) ListPerspectiveRecurringTransactions(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	pID := strings.TrimSpace(c.Param("perspectiveId"))
	res, err := h.ctrl.ListPerspectiveRecurringTransactions(c.Request.Context(), userID, pID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// LinkPerspectiveRecurringTransaction handles POST /api/v1/perspectives/:perspectiveId/recurring-transactions
func (h *Handler) LinkPerspectiveRecurringTransaction(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	pID := strings.TrimSpace(c.Param("perspectiveId"))
	var req models.PerspectiveRecurringTransactionLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	if err := h.ctrl.LinkPerspectiveRecurringTransaction(c.Request.Context(), userID, pID, req.RecurringTransactionID); err != nil {
		h.handleError(c, err)
		return
	}
	c.Status(http.StatusCreated)
}

// UnlinkPerspectiveRecurringTransaction handles DELETE /api/v1/perspectives/:perspectiveId/recurring-transactions/:recurringId
func (h *Handler) UnlinkPerspectiveRecurringTransaction(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	pID := strings.TrimSpace(c.Param("perspectiveId"))
	recID := strings.TrimSpace(c.Param("recurringId"))
	if err := h.ctrl.UnlinkPerspectiveRecurringTransaction(c.Request.Context(), userID, pID, recID); err != nil {
		h.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListPerspectiveAccess handles GET /api/v1/perspectives/:perspectiveId/access
func (h *Handler) ListPerspectiveAccess(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	pID := strings.TrimSpace(c.Param("perspectiveId"))
	res, err := h.ctrl.ListPerspectiveAccess(c.Request.Context(), userID, pID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetPerspectiveAccess handles GET /api/v1/perspectives/:perspectiveId/access/:userId
func (h *Handler) GetPerspectiveAccess(c *gin.Context) {
	reqUserID, ok := h.getUserID(c)
	if !ok {
		return
	}
	pID := strings.TrimSpace(c.Param("perspectiveId"))
	targetUserID := strings.TrimSpace(c.Param("userId"))
	res, err := h.ctrl.GetPerspectiveAccess(c.Request.Context(), reqUserID, pID, targetUserID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GrantPerspectiveAccess handles POST /api/v1/perspectives/:perspectiveId/access
func (h *Handler) GrantPerspectiveAccess(c *gin.Context) {
	adminUserID, ok := h.getUserID(c)
	if !ok {
		return
	}
	pID := strings.TrimSpace(c.Param("perspectiveId"))
	var req models.PerspectiveAccessGrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	res, err := h.ctrl.GrantPerspectiveAccess(c.Request.Context(), adminUserID, pID, req.UserID, req.AccessLevel)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// RevokePerspectiveAccess handles DELETE /api/v1/perspectives/:perspectiveId/access/:userId
func (h *Handler) RevokePerspectiveAccess(c *gin.Context) {
	adminUserID, ok := h.getUserID(c)
	if !ok {
		return
	}
	pID := strings.TrimSpace(c.Param("perspectiveId"))
	targetUserID := strings.TrimSpace(c.Param("userId"))
	if err := h.ctrl.RevokePerspectiveAccess(c.Request.Context(), adminUserID, pID, targetUserID); err != nil {
		h.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// GetPerspectiveView handles GET /api/v1/perspectives/:perspectiveId/view
func (h *Handler) GetPerspectiveView(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	pID := strings.TrimSpace(c.Param("perspectiveId"))
	res, err := h.ctrl.GetPerspectiveView(c.Request.Context(), userID, pID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
