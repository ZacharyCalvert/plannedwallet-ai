package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/plannedwallet/internal/models"
)

// CreateReconcileReport handles POST /api/v1/accounts/:accountId/reconcile-reports
func (h *Handler) CreateReconcileReport(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var req models.ReconcileReportCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload: "+err.Error())
		return
	}

	res, err := h.ctrl.CreateReconcileReport(c.Request.Context(), userID, c.Param("accountId"), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

// ListReconcileReports handles GET /api/v1/accounts/:accountId/reconcile-reports
func (h *Handler) ListReconcileReports(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	res, err := h.ctrl.ListReconcileReports(c.Request.Context(), userID, c.Param("accountId"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetReconcileReport handles GET /api/v1/accounts/:accountId/reconcile-reports/:reportId
func (h *Handler) GetReconcileReport(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	res, err := h.ctrl.GetReconcileReport(c.Request.Context(), userID, c.Param("accountId"), c.Param("reportId"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// DeleteReconcileReport handles DELETE /api/v1/accounts/:accountId/reconcile-reports/:reportId
func (h *Handler) DeleteReconcileReport(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	if err := h.ctrl.DeleteReconcileReport(c.Request.Context(), userID, c.Param("accountId"), c.Param("reportId")); err != nil {
		h.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
