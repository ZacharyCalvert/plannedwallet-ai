package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/plannedwallet/internal/models"
)

// AddAccountAsset handles POST /api/v1/accounts/:accountId/assets
func (h *Handler) AddAccountAsset(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := c.Param("accountId")

	var req models.AccountAssetAddRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload: "+err.Error())
		return
	}

	res, err := h.ctrl.AddAccountAsset(c.Request.Context(), userID, accountID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

// ListAccountAssets handles GET /api/v1/accounts/:accountId/assets
func (h *Handler) ListAccountAssets(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := c.Param("accountId")

	res, err := h.ctrl.ListAccountAssets(c.Request.Context(), userID, accountID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetAccountAsset handles GET /api/v1/accounts/:accountId/assets/:assetId
func (h *Handler) GetAccountAsset(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := c.Param("accountId")
	assetID := c.Param("assetId")

	res, err := h.ctrl.GetAccountAsset(c.Request.Context(), userID, accountID, assetID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// UpdateAccountAsset handles PUT /api/v1/accounts/:accountId/assets/:assetId
func (h *Handler) UpdateAccountAsset(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := c.Param("accountId")
	assetID := c.Param("assetId")

	var req models.AccountAssetUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload: "+err.Error())
		return
	}

	res, err := h.ctrl.UpdateAccountAsset(c.Request.Context(), userID, accountID, assetID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// DeleteAccountAsset handles DELETE /api/v1/accounts/:accountId/assets/:assetId
func (h *Handler) DeleteAccountAsset(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := c.Param("accountId")
	assetID := c.Param("assetId")

	if err := h.ctrl.DeleteAccountAsset(c.Request.Context(), userID, accountID, assetID); err != nil {
		h.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// GetAssetAccountSummary handles GET /api/v1/accounts/:accountId/assets/summary
func (h *Handler) GetAssetAccountSummary(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := c.Param("accountId")

	res, err := h.ctrl.GetAssetAccountSummary(c.Request.Context(), userID, accountID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetAssetAccountTrend handles GET /api/v1/accounts/:accountId/assets/trend
func (h *Handler) GetAssetAccountTrend(c *gin.Context) {
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

	res, err := h.ctrl.GetAssetAccountTrend(c.Request.Context(), userID, accountID, startDate, endDate)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
