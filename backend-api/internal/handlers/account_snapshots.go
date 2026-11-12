package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/plannedwallet/internal/models"
)

// CreateAccountSnapshot handles POST /api/v1/accounts/:accountId/snapshots
func (h *Handler) CreateAccountSnapshot(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := c.Param("accountId")

	var req models.AccountSnapshotCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload: "+err.Error())
		return
	}

	res, err := h.ctrl.CreateAccountSnapshot(c.Request.Context(), userID, accountID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

// ListAccountSnapshots handles GET /api/v1/accounts/:accountId/snapshots
func (h *Handler) ListAccountSnapshots(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := c.Param("accountId")

	var startDate, endDate *string
	if s := c.Query("start_date"); s != "" {
		startDate = &s
	}
	if e := c.Query("end_date"); e != "" {
		endDate = &e
	}

	res, err := h.ctrl.ListAccountSnapshots(c.Request.Context(), userID, accountID, startDate, endDate)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetAccountSnapshot handles GET /api/v1/accounts/:accountId/snapshots/:snapshotId
func (h *Handler) GetAccountSnapshot(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := c.Param("accountId")
	snapshotID := c.Param("snapshotId")

	res, err := h.ctrl.GetAccountSnapshot(c.Request.Context(), userID, accountID, snapshotID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// UpdateAccountSnapshot handles PUT /api/v1/accounts/:accountId/snapshots/:snapshotId
func (h *Handler) UpdateAccountSnapshot(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := c.Param("accountId")
	snapshotID := c.Param("snapshotId")

	var req models.AccountSnapshotUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload: "+err.Error())
		return
	}

	res, err := h.ctrl.UpdateAccountSnapshot(c.Request.Context(), userID, accountID, snapshotID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// DeleteAccountSnapshot handles DELETE /api/v1/accounts/:accountId/snapshots/:snapshotId
func (h *Handler) DeleteAccountSnapshot(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := c.Param("accountId")
	snapshotID := c.Param("snapshotId")

	if err := h.ctrl.DeleteAccountSnapshot(c.Request.Context(), userID, accountID, snapshotID); err != nil {
		h.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
