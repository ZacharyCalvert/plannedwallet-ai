package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/plannedwallet/internal/models"
)

func (h *Handler) parseRecordFilters(c *gin.Context) (*bool, *string, *string, bool) {
	var clearedFilter *bool
	if clearedStr := c.Query("cleared"); clearedStr != "" {
		val, err := strconv.ParseBool(clearedStr)
		if err != nil {
			h.sendError(c, http.StatusBadRequest, "INVALID_QUERY_PARAM", "cleared must be true or false.")
			return nil, nil, nil, false
		}
		clearedFilter = &val
	}

	var startDateFilter *string
	if sDate := c.Query("start_date"); sDate != "" {
		if err := parseDate(sDate); err != nil {
			h.sendError(c, http.StatusBadRequest, "INVALID_QUERY_PARAM", "start_date must be formatted as YYYY-MM-DD.")
			return nil, nil, nil, false
		}
		startDateFilter = &sDate
	}

	var endDateFilter *string
	if eDate := c.Query("end_date"); eDate != "" {
		if err := parseDate(eDate); err != nil {
			h.sendError(c, http.StatusBadRequest, "INVALID_QUERY_PARAM", "end_date must be formatted as YYYY-MM-DD.")
			return nil, nil, nil, false
		}
		endDateFilter = &eDate
	}
	return clearedFilter, startDateFilter, endDateFilter, true
}

// CreateCategory handles POST /api/v1/categories
func (h *Handler) CreateCategory(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var req models.CategoryCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload: "+err.Error())
		return
	}

	res, err := h.ctrl.CreateCategory(c.Request.Context(), userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

// ListCategories handles GET /api/v1/categories
func (h *Handler) ListCategories(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	res, err := h.ctrl.ListCategories(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetCategory handles GET /api/v1/categories/:categoryId
func (h *Handler) GetCategory(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	res, err := h.ctrl.GetCategory(c.Request.Context(), userID, c.Param("categoryId"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// DeleteCategory handles DELETE /api/v1/categories/:categoryId
func (h *Handler) DeleteCategory(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	if err := h.ctrl.DeleteCategory(c.Request.Context(), userID, c.Param("categoryId")); err != nil {
		h.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// GetAccountCategorySummary handles GET /api/v1/accounts/:accountId/records/categories
func (h *Handler) GetAccountCategorySummary(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	clearedFilter, startDateFilter, endDateFilter, valid := h.parseRecordFilters(c)
	if !valid {
		return
	}

	res, err := h.ctrl.GetAccountCategorySummary(c.Request.Context(), userID, c.Param("accountId"), clearedFilter, startDateFilter, endDateFilter)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
