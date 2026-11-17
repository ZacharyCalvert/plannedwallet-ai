package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/plannedwallet/internal/models"
)

// CreateBill handles POST /api/v1/bills
func (h *Handler) CreateBill(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var req models.BillCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload: "+err.Error())
		return
	}

	res, err := h.ctrl.CreateBill(c.Request.Context(), userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

// ListBills handles GET /api/v1/bills
func (h *Handler) ListBills(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	res, err := h.ctrl.ListBills(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// ListBillReminders handles GET /api/v1/bills/reminders
func (h *Handler) ListBillReminders(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	res, err := h.ctrl.ListBillReminders(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetBill handles GET /api/v1/bills/:billId
func (h *Handler) GetBill(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	res, err := h.ctrl.GetBill(c.Request.Context(), userID, c.Param("billId"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// UpdateBill handles PUT /api/v1/bills/:billId
func (h *Handler) UpdateBill(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var req models.BillUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload: "+err.Error())
		return
	}

	res, err := h.ctrl.UpdateBill(c.Request.Context(), userID, c.Param("billId"), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// DeleteBill handles DELETE /api/v1/bills/:billId
func (h *Handler) DeleteBill(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	if err := h.ctrl.DeleteBill(c.Request.Context(), userID, c.Param("billId")); err != nil {
		h.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ApplyBillToAccount handles POST /api/v1/bills/:billId/apply
func (h *Handler) ApplyBillToAccount(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var req models.BillApplyRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			h.sendError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload: "+err.Error())
			return
		}
	}

	res, err := h.ctrl.ApplyBillToAccount(c.Request.Context(), userID, c.Param("billId"), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}
