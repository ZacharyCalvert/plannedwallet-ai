package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/plannedwallet/internal/models"
)

// CreateRecurringTransaction handles POST /api/v1/recurring-transactions
func (h *Handler) CreateRecurringTransaction(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var req models.RecurringTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload: "+err.Error())
		return
	}

	res, err := h.ctrl.CreateRecurringTransaction(c.Request.Context(), userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

// ListRecurringTransactions handles GET /api/v1/recurring-transactions
func (h *Handler) ListRecurringTransactions(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	res, err := h.ctrl.ListRecurringTransactions(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// ListRecurringReminders handles GET /api/v1/recurring-transactions/reminders
func (h *Handler) ListRecurringReminders(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	res, err := h.ctrl.ListRecurringReminders(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetRecurringTransaction handles GET /api/v1/recurring-transactions/:recurringId
func (h *Handler) GetRecurringTransaction(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	res, err := h.ctrl.GetRecurringTransaction(c.Request.Context(), userID, c.Param("recurringId"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// UpdateRecurringTransaction handles PUT /api/v1/recurring-transactions/:recurringId
func (h *Handler) UpdateRecurringTransaction(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var req models.RecurringTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload: "+err.Error())
		return
	}

	res, err := h.ctrl.UpdateRecurringTransaction(c.Request.Context(), userID, c.Param("recurringId"), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// DeleteRecurringTransaction handles DELETE /api/v1/recurring-transactions/:recurringId
func (h *Handler) DeleteRecurringTransaction(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	if err := h.ctrl.DeleteRecurringTransaction(c.Request.Context(), userID, c.Param("recurringId")); err != nil {
		h.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ApplyRecurringTransactionToAccount handles POST /api/v1/recurring-transactions/:recurringId/apply
func (h *Handler) ApplyRecurringTransactionToAccount(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var req models.RecurringTransactionApplyRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			h.sendError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload: "+err.Error())
			return
		}
	}

	res, err := h.ctrl.ApplyRecurringTransactionToAccount(c.Request.Context(), userID, c.Param("recurringId"), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}
