package handlers

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/controller"
	"github.com/plannedwallet/internal/models"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Handler serves as the view layer in the MVC architecture, translating
// HTTP requests/responses and interacting with the controller layer.
type Handler struct {
	ctrl *controller.Controller
	auth *auth.Service
}

func NewHandler(ctrl *controller.Controller, authService *auth.Service) *Handler {
	return &Handler{
		ctrl: ctrl,
		auth: authService,
	}
}

func (h *Handler) sendError(c *gin.Context, statusCode int, errCode, message string) {
	c.JSON(statusCode, models.ErrorResponse{
		Error:     errCode,
		Message:   message,
		Timestamp: time.Now().UTC(),
	})
}

func (h *Handler) handleError(c *gin.Context, err error) {
	var appErr *controller.AppError
	if errors.As(err, &appErr) {
		h.sendError(c, appErr.Status, appErr.Code, appErr.Message)
		return
	}
	h.sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.")
}

func parseDate(dateStr string) error {
	_, err := time.Parse("2006-01-02", dateStr)
	return err
}

func (h *Handler) getUserID(c *gin.Context) (string, bool) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		h.sendError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authenticated user.")
		return "", false
	}
	return userIDVal.(string), true
}

// RegisterUser handles POST /api/v1/users
func (h *Handler) RegisterUser(c *gin.Context) {
	var req models.UserRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)

	if len(req.Username) < 3 || len(req.Username) > 50 || !usernameRegex.MatchString(req.Username) {
		h.sendError(c, http.StatusBadRequest, "INVALID_USERNAME", "Username must be 3-50 alphanumeric characters, hyphens, or underscores.")
		return
	}

	res, err := h.ctrl.RegisterUser(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, res)
}

// ExchangeToken handles POST /api/v1/users/{username}/token
func (h *Handler) ExchangeToken(c *gin.Context) {
	username := strings.TrimSpace(c.Param("username"))
	if username == "" {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", "Username path parameter is required.")
		return
	}

	var req models.TokenExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", "Password is required.")
		return
	}

	res, err := h.ctrl.ExchangeToken(c.Request.Context(), username, req.Password)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// CreateAccount handles POST /api/v1/accounts
func (h *Handler) CreateAccount(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var req models.AccountCreationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	res, err := h.ctrl.CreateAccount(c.Request.Context(), userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, res)
}

// CreateCheckingRecord handles POST /api/v1/accounts/{accountId}/checking-records
func (h *Handler) CreateCheckingRecord(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := c.Param("accountId")

	var req models.CheckingRecordCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	if err := parseDate(req.DateSent); err != nil {
		h.sendError(c, http.StatusBadRequest, "INVALID_DATE_SENT", "date_sent must be formatted as YYYY-MM-DD.")
		return
	}

	if req.DateCleared != nil && *req.DateCleared != "" {
		if err := parseDate(*req.DateCleared); err != nil {
			h.sendError(c, http.StatusBadRequest, "INVALID_DATE_CLEARED", "date_cleared must be formatted as YYYY-MM-DD.")
			return
		}
	}

	res, err := h.ctrl.CreateCheckingRecord(c.Request.Context(), userID, accountID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, res)
}

// ListCheckingRecords handles GET /api/v1/accounts/{accountId}/checking-records
func (h *Handler) ListCheckingRecords(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := c.Param("accountId")

	var clearedFilter *bool
	if clearedStr := c.Query("cleared"); clearedStr != "" {
		if val, err := strconv.ParseBool(clearedStr); err == nil {
			clearedFilter = &val
		} else {
			h.sendError(c, http.StatusBadRequest, "INVALID_QUERY_PARAM", "cleared must be true or false.")
			return
		}
	}

	var startDateFilter *string
	if sDate := c.Query("start_date"); sDate != "" {
		if err := parseDate(sDate); err != nil {
			h.sendError(c, http.StatusBadRequest, "INVALID_QUERY_PARAM", "start_date must be formatted as YYYY-MM-DD.")
			return
		}
		startDateFilter = &sDate
	}

	var endDateFilter *string
	if eDate := c.Query("end_date"); eDate != "" {
		if err := parseDate(eDate); err != nil {
			h.sendError(c, http.StatusBadRequest, "INVALID_QUERY_PARAM", "end_date must be formatted as YYYY-MM-DD.")
			return
		}
		endDateFilter = &eDate
	}

	res, err := h.ctrl.ListCheckingRecords(c.Request.Context(), userID, accountID, clearedFilter, startDateFilter, endDateFilter)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// GetCheckingRecord handles GET /api/v1/accounts/{accountId}/checking-records/{recordId}
func (h *Handler) GetCheckingRecord(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := c.Param("accountId")
	recordID := c.Param("recordId")

	res, err := h.ctrl.GetCheckingRecord(c.Request.Context(), userID, accountID, recordID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// UpdateCheckingRecord handles PUT /api/v1/accounts/{accountId}/checking-records/{recordId}
func (h *Handler) UpdateCheckingRecord(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := c.Param("accountId")
	recordID := c.Param("recordId")

	var req models.CheckingRecordUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	if err := parseDate(req.DateSent); err != nil {
		h.sendError(c, http.StatusBadRequest, "INVALID_DATE_SENT", "date_sent must be formatted as YYYY-MM-DD.")
		return
	}

	if req.DateCleared != nil && *req.DateCleared != "" {
		if err := parseDate(*req.DateCleared); err != nil {
			h.sendError(c, http.StatusBadRequest, "INVALID_DATE_CLEARED", "date_cleared must be formatted as YYYY-MM-DD.")
			return
		}
	}

	res, err := h.ctrl.UpdateCheckingRecord(c.Request.Context(), userID, accountID, recordID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// PatchCheckingRecord handles PATCH /api/v1/accounts/{accountId}/checking-records/{recordId}
func (h *Handler) PatchCheckingRecord(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := c.Param("accountId")
	recordID := c.Param("recordId")

	var req models.CheckingRecordPatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	if req.DateSent != nil {
		if err := parseDate(*req.DateSent); err != nil {
			h.sendError(c, http.StatusBadRequest, "INVALID_DATE_SENT", "date_sent must be formatted as YYYY-MM-DD.")
			return
		}
	}
	if req.DateCleared != nil && *req.DateCleared != "" {
		if err := parseDate(*req.DateCleared); err != nil {
			h.sendError(c, http.StatusBadRequest, "INVALID_DATE_CLEARED", "date_cleared must be formatted as YYYY-MM-DD.")
			return
		}
	}

	res, err := h.ctrl.PatchCheckingRecord(c.Request.Context(), userID, accountID, recordID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// DeleteCheckingRecord handles DELETE /api/v1/accounts/{accountId}/checking-records/{recordId}
func (h *Handler) DeleteCheckingRecord(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	accountID := c.Param("accountId")
	recordID := c.Param("recordId")

	if err := h.ctrl.DeleteCheckingRecord(c.Request.Context(), userID, accountID, recordID); err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// RegisterRoutes registers all API routes according to schema.yml
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	{
		// Public endpoints
		api.POST("/users", h.RegisterUser)
		api.POST("/users/:username/token", h.ExchangeToken)

		// Protected endpoints
		protected := api.Group("")
		protected.Use(h.auth.Middleware())
		{
			protected.POST("/accounts", h.CreateAccount)

			// Checking Records
			protected.POST("/accounts/:accountId/checking-records", h.CreateCheckingRecord)
			protected.GET("/accounts/:accountId/checking-records", h.ListCheckingRecords)
			protected.GET("/accounts/:accountId/checking-records/:recordId", h.GetCheckingRecord)
			protected.PUT("/accounts/:accountId/checking-records/:recordId", h.UpdateCheckingRecord)
			protected.PATCH("/accounts/:accountId/checking-records/:recordId", h.PatchCheckingRecord)
			protected.DELETE("/accounts/:accountId/checking-records/:recordId", h.DeleteCheckingRecord)
		}
	}
}
