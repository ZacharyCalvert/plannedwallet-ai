package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/plannedwallet/internal/models"
)

// CreateAsset handles POST /api/v1/assets
func (h *Handler) CreateAsset(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var req models.AssetCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload: "+err.Error())
		return
	}

	res, err := h.ctrl.CreateAsset(c.Request.Context(), userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, res)
}

// ListAssets handles GET /api/v1/assets
func (h *Handler) ListAssets(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var labelPtr *string
	if l := c.Query("label"); l != "" {
		labelPtr = &l
	}

	res, err := h.ctrl.ListAssets(c.Request.Context(), userID, labelPtr)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// GetAsset handles GET /api/v1/assets/:assetId
func (h *Handler) GetAsset(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	id := c.Param("assetId")

	res, err := h.ctrl.GetAsset(c.Request.Context(), userID, id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// UpdateAsset handles PUT /api/v1/assets/:assetId
func (h *Handler) UpdateAsset(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	id := c.Param("assetId")

	var req models.AssetUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload: "+err.Error())
		return
	}

	res, err := h.ctrl.UpdateAsset(c.Request.Context(), userID, id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// DeleteAsset handles DELETE /api/v1/assets/:assetId
func (h *Handler) DeleteAsset(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	id := c.Param("assetId")

	if err := h.ctrl.DeleteAsset(c.Request.Context(), userID, id); err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateAssetValue handles POST /api/v1/assets/:assetId/values
func (h *Handler) CreateAssetValue(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	assetID := c.Param("assetId")

	var req models.AssetValueCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload: "+err.Error())
		return
	}

	res, err := h.ctrl.CreateAssetValue(c.Request.Context(), userID, assetID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, res)
}

// ListAssetValues handles GET /api/v1/assets/:assetId/values
func (h *Handler) ListAssetValues(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	assetID := c.Param("assetId")

	var startDatePtr, endDatePtr *string
	if s := c.Query("start_date"); s != "" {
		startDatePtr = &s
	}
	if e := c.Query("end_date"); e != "" {
		endDatePtr = &e
	}

	res, err := h.ctrl.ListAssetValues(c.Request.Context(), userID, assetID, startDatePtr, endDatePtr)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// GetAssetValue handles GET /api/v1/assets/:assetId/values/:valueId
func (h *Handler) GetAssetValue(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	assetID := c.Param("assetId")
	valueID := c.Param("valueId")

	res, err := h.ctrl.GetAssetValue(c.Request.Context(), userID, assetID, valueID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// UpdateAssetValue handles PUT /api/v1/assets/:assetId/values/:valueId
func (h *Handler) UpdateAssetValue(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	assetID := c.Param("assetId")
	valueID := c.Param("valueId")

	var req models.AssetValueUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload: "+err.Error())
		return
	}

	res, err := h.ctrl.UpdateAssetValue(c.Request.Context(), userID, assetID, valueID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// DeleteAssetValue handles DELETE /api/v1/assets/:assetId/values/:valueId
func (h *Handler) DeleteAssetValue(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	assetID := c.Param("assetId")
	valueID := c.Param("valueId")

	if err := h.ctrl.DeleteAssetValue(c.Request.Context(), userID, assetID, valueID); err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
