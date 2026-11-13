package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/plannedwallet/internal/models"
)

// CreateAssetStore handles POST /api/v1/accounts/:accountId/stores
func (h *Handler) CreateAssetStore(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := strings.TrimSpace(c.Param("accountId"))
	var req models.CreateAssetStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	res, err := h.ctrl.CreateAssetStore(c.Request.Context(), userID, accountID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

// GetAssetStore handles GET /api/v1/accounts/:accountId/stores/:storeId
func (h *Handler) GetAssetStore(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := strings.TrimSpace(c.Param("accountId"))
	storeID := strings.TrimSpace(c.Param("storeId"))
	res, err := h.ctrl.GetAssetStore(c.Request.Context(), userID, accountID, storeID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// ListAssetStores handles GET /api/v1/accounts/:accountId/stores
func (h *Handler) ListAssetStores(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := strings.TrimSpace(c.Param("accountId"))
	res, err := h.ctrl.ListAssetStores(c.Request.Context(), userID, accountID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// UpdateAssetStore handles PUT /api/v1/accounts/:accountId/stores/:storeId
func (h *Handler) UpdateAssetStore(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := strings.TrimSpace(c.Param("accountId"))
	storeID := strings.TrimSpace(c.Param("storeId"))
	var req models.UpdateAssetStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	res, err := h.ctrl.UpdateAssetStore(c.Request.Context(), userID, accountID, storeID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// DeleteAssetStore handles DELETE /api/v1/accounts/:accountId/stores/:storeId
func (h *Handler) DeleteAssetStore(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := strings.TrimSpace(c.Param("accountId"))
	storeID := strings.TrimSpace(c.Param("storeId"))
	if err := h.ctrl.DeleteAssetStore(c.Request.Context(), userID, accountID, storeID); err != nil {
		h.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// CreateAssetStoreRecord handles POST /api/v1/accounts/:accountId/stores/:storeId/records
func (h *Handler) CreateAssetStoreRecord(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := strings.TrimSpace(c.Param("accountId"))
	storeID := strings.TrimSpace(c.Param("storeId"))
	var req models.CreateStoreRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	res, err := h.ctrl.CreateAssetStoreRecord(c.Request.Context(), userID, accountID, storeID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

// GetAssetStoreRecord handles GET /api/v1/accounts/:accountId/stores/:storeId/records/:recordId
func (h *Handler) GetAssetStoreRecord(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := strings.TrimSpace(c.Param("accountId"))
	storeID := strings.TrimSpace(c.Param("storeId"))
	recordID := strings.TrimSpace(c.Param("recordId"))
	res, err := h.ctrl.GetAssetStoreRecord(c.Request.Context(), userID, accountID, storeID, recordID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// ListAssetStoreRecords handles GET /api/v1/accounts/:accountId/stores/:storeId/records
func (h *Handler) ListAssetStoreRecords(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := strings.TrimSpace(c.Param("accountId"))
	storeID := strings.TrimSpace(c.Param("storeId"))
	res, err := h.ctrl.ListAssetStoreRecords(c.Request.Context(), userID, accountID, storeID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// UpdateAssetStoreRecord handles PUT /api/v1/accounts/:accountId/stores/:storeId/records/:recordId
func (h *Handler) UpdateAssetStoreRecord(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := strings.TrimSpace(c.Param("accountId"))
	storeID := strings.TrimSpace(c.Param("storeId"))
	recordID := strings.TrimSpace(c.Param("recordId"))
	var req models.UpdateStoreRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	res, err := h.ctrl.UpdateAssetStoreRecord(c.Request.Context(), userID, accountID, storeID, recordID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// DeleteAssetStoreRecord handles DELETE /api/v1/accounts/:accountId/stores/:storeId/records/:recordId
func (h *Handler) DeleteAssetStoreRecord(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := strings.TrimSpace(c.Param("accountId"))
	storeID := strings.TrimSpace(c.Param("storeId"))
	recordID := strings.TrimSpace(c.Param("recordId"))
	if err := h.ctrl.DeleteAssetStoreRecord(c.Request.Context(), userID, accountID, storeID, recordID); err != nil {
		h.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// GetAssetStoreTrend handles GET /api/v1/accounts/:accountId/stores/:storeId/trend
func (h *Handler) GetAssetStoreTrend(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := strings.TrimSpace(c.Param("accountId"))
	storeID := strings.TrimSpace(c.Param("storeId"))
	res, err := h.ctrl.GetAssetStoreTrend(c.Request.Context(), userID, accountID, storeID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
