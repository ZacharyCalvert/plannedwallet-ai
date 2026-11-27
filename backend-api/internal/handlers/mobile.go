package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/plannedwallet/internal/models"
)

// MobileTokenValidationMiddleware ensures the mobile JWT is persisted, active, and not revoked.
func (h *Handler) MobileTokenValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := h.getUserID(c)
		if !ok {
			c.Abort()
			return
		}
		tokenIDVal, exists := c.Get("tokenID")
		tokenID, _ := tokenIDVal.(string)
		if !exists || tokenID == "" {
			h.sendError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Missing mobile token identifier.")
			c.Abort()
			return
		}
		if err := h.ctrl.ValidateActiveMobileToken(c.Request.Context(), tokenID, userID); err != nil {
			h.handleError(c, err)
			c.Abort()
			return
		}
		c.Next()
	}
}

// IssueMobileToken handles POST /api/v1/mobile/token
func (h *Handler) IssueMobileToken(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	mobileID := c.GetHeader("X-Mobile-Identifier")
	res, err := h.ctrl.IssueMobileToken(c.Request.Context(), userID, mobileID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

// ListMobileTargets handles GET /api/v1/mobile/accounts
func (h *Handler) ListMobileTargets(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	res, err := h.ctrl.ListMobileTargets(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// CreateMobileRecord handles POST /api/v1/mobile/accounts/:accountId/records
func (h *Handler) CreateMobileRecord(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := c.Param("accountId")

	var req models.MobileRecordCreateRequest
	if err := bindStrictJSON(c, &req); err != nil {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if err := parseDate(req.DateSent); err != nil {
		h.sendError(c, http.StatusBadRequest, "INVALID_DATE_SENT", "date_sent must be formatted as YYYY-MM-DD.")
		return
	}

	res, err := h.ctrl.CreateMobileRecord(c.Request.Context(), userID, accountID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}
