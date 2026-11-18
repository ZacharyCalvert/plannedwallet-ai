package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/controller"
	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/trace"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Handler serves as the view layer in the MVC architecture, translating
// HTTP requests/responses and interacting with the controller layer.
type Handler struct {
	ctrl     *controller.Controller
	auth     *auth.Service
	readOnly bool
}

func NewHandler(ctrl *controller.Controller, authService *auth.Service) *Handler {
	return &Handler{
		ctrl: ctrl,
		auth: authService,
	}
}

func (h *Handler) sendError(c *gin.Context, statusCode int, errCode, message string) {
	trace.LogError(c.Request.Context(), nil, "Handler error [%s]: %s (status %d)", errCode, message, statusCode)
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
	trace.LogError(c.Request.Context(), err, "Internal server error")
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

func bindStrictJSON(c *gin.Context, dst any) error {
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return errors.New("unexpected trailing JSON data")
	}
	return binding.Validator.ValidateStruct(dst)
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

// ExchangePasswordForToken handles POST /api/v1/users/{username}/token
func (h *Handler) ExchangePasswordForToken(c *gin.Context) {
	username := strings.TrimSpace(c.Param("username"))
	if username == "" {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", "Username path parameter is required.")
		return
	}

	var req models.TokenExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", "Password and totp_code are required.")
		return
	}

	res, err := h.ctrl.ExchangePasswordForToken(c.Request.Context(), username, req.Password, req.TOTPCode)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// RefreshToken handles POST /api/v1/users/{username}/token/refresh
func (h *Handler) RefreshToken(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	username := strings.TrimSpace(c.Param("username"))
	if username == "" {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", "Username path parameter is required.")
		return
	}

	res, err := h.ctrl.RefreshToken(c.Request.Context(), userID, username)
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

// ListAccounts handles GET /api/v1/accounts
func (h *Handler) ListAccounts(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	res, err := h.ctrl.ListAccounts(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// GetAccount handles GET /api/v1/accounts/:accountId
func (h *Handler) GetAccount(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	accountID := c.Param("accountId")
	res, err := h.ctrl.GetAccount(c.Request.Context(), userID, accountID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// PatchAccount handles PATCH /api/v1/accounts/:accountId (updates account name and/or description)
func (h *Handler) PatchAccount(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	accountID := strings.TrimSpace(c.Param("accountId"))
	var req models.AccountPatchRequest
	if err := bindStrictJSON(c, &req); err != nil {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	res, err := h.ctrl.PatchAccount(c.Request.Context(), userID, accountID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// DeleteAccount handles DELETE /api/v1/accounts/:accountId
func (h *Handler) DeleteAccount(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	accountID := strings.TrimSpace(c.Param("accountId"))
	if err := h.ctrl.DeleteAccount(c.Request.Context(), userID, accountID); err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateCheckingRecord handles POST /api/v1/accounts/{accountId}/records
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

// ListCheckingRecords handles GET /api/v1/accounts/{accountId}/records
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

// GetCheckingRecord handles GET /api/v1/accounts/{accountId}/records/{recordId}
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

// UpdateCheckingRecord handles PUT /api/v1/accounts/{accountId}/records/{recordId}
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

// PatchCheckingRecord handles PATCH /api/v1/accounts/{accountId}/records/{recordId}
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

// DeleteCheckingRecord handles DELETE /api/v1/accounts/{accountId}/records/{recordId}
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

// Hello handles GET /api/v1/hello
func (h *Handler) Hello(c *gin.Context) {
	c.String(http.StatusOK, "world")
}

// Health handles GET /api/health and responds with {"status":"up"} once DB connection is confirmed.
func (h *Handler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := h.ctrl.HealthCheck(ctx); err != nil {
		trace.LogError(c.Request.Context(), err, "Database health check failed")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "down",
			"error":  "database unavailable",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "up",
	})
}

// RegisterRoutes registers all API routes according to schema.yml
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/health", h.Health)
	api := r.Group("/api/v1")
	api.Use(h.ReadOnlyMiddleware())
	h.registerPublicRoutes(api)

	protected := api.Group("")
	protected.Use(h.auth.Middleware())
	protected.POST("/users/:username/token/refresh", h.RefreshToken)
	h.registerAccountRoutes(protected)
	h.registerStoreAndSnapshotRoutes(protected)
	h.registerAssetRoutes(protected)
	h.registerCategoryRoutes(protected)
	h.registerPerspectiveRoutes(protected)
	h.registerRecurringTransactionRoutes(protected)
}

func (h *Handler) registerPublicRoutes(api *gin.RouterGroup) {
	api.GET("/health", h.Health)
	api.GET("/hello", h.Hello)
	api.GET("/features", h.GetFeatures)
	api.GET("/captcha", h.GetCaptcha)
	api.GET("/captcha/:captchaId", h.GetCaptchaImage)
	api.POST("/users", h.RegisterUser)
	api.POST("/users/:username/token", h.ExchangePasswordForToken)
}

func (h *Handler) registerAccountRoutes(p *gin.RouterGroup) {
	p.POST("/accounts", h.CreateAccount)
	p.GET("/accounts", h.ListAccounts)
	p.GET("/accounts/:accountId", h.GetAccount)
	p.PATCH("/accounts/:accountId", h.PatchAccount)
	p.DELETE("/accounts/:accountId", h.DeleteAccount)
	p.POST("/accounts/:accountId/access", h.GrantAccountAccess)
	p.GET("/accounts/:accountId/access/:userId", h.GetAccountAccess)
	p.DELETE("/accounts/:accountId/access/:userId", h.RevokeAccountAccess)

	p.POST("/accounts/:accountId/records", h.CreateCheckingRecord)
	p.GET("/accounts/:accountId/records", h.ListCheckingRecords)
	p.GET("/accounts/:accountId/records/categories", h.GetAccountCategorySummary)
	p.GET("/accounts/:accountId/records/:recordId", h.GetCheckingRecord)
	p.PUT("/accounts/:accountId/records/:recordId", h.UpdateCheckingRecord)
	p.PATCH("/accounts/:accountId/records/:recordId", h.PatchCheckingRecord)
	p.DELETE("/accounts/:accountId/records/:recordId", h.DeleteCheckingRecord)

	p.POST("/accounts/:accountId/reconcile-reports", h.CreateReconcileReport)
	p.GET("/accounts/:accountId/reconcile-reports", h.ListReconcileReports)
	p.GET("/accounts/:accountId/reconcile-reports/:reportId", h.GetReconcileReport)
	p.DELETE("/accounts/:accountId/reconcile-reports/:reportId", h.DeleteReconcileReport)

	p.POST("/accounts/:accountId/assets", h.AddAccountAsset)
	p.GET("/accounts/:accountId/assets", h.ListAccountAssets)
	p.GET("/accounts/:accountId/assets/summary", h.GetAssetAccountSummary)
	p.GET("/accounts/:accountId/assets/trend", h.GetAssetAccountTrend)
	p.GET("/accounts/:accountId/assets/:assetId", h.GetAccountAsset)
	p.PUT("/accounts/:accountId/assets/:assetId", h.UpdateAccountAsset)
	p.DELETE("/accounts/:accountId/assets/:assetId", h.DeleteAccountAsset)
}

func (h *Handler) registerStoreAndSnapshotRoutes(p *gin.RouterGroup) {
	p.POST("/accounts/:accountId/stores", h.CreateAssetStore)
	p.GET("/accounts/:accountId/stores", h.ListAssetStores)
	p.GET("/accounts/:accountId/stores/summary", h.GetAssetAccountSummary)
	p.GET("/accounts/:accountId/stores/trend", h.GetAssetAccountTrend)
	p.GET("/accounts/:accountId/stores/:storeId", h.GetAssetStore)
	p.PUT("/accounts/:accountId/stores/:storeId", h.UpdateAssetStore)
	p.DELETE("/accounts/:accountId/stores/:storeId", h.DeleteAssetStore)
	p.GET("/accounts/:accountId/stores/:storeId/trend", h.GetAssetStoreTrend)

	p.POST("/accounts/:accountId/stores/:storeId/records", h.CreateAssetStoreRecord)
	p.GET("/accounts/:accountId/stores/:storeId/records", h.ListAssetStoreRecords)
	p.GET("/accounts/:accountId/stores/:storeId/records/:recordId", h.GetAssetStoreRecord)
	p.PUT("/accounts/:accountId/stores/:storeId/records/:recordId", h.UpdateAssetStoreRecord)
	p.DELETE("/accounts/:accountId/stores/:storeId/records/:recordId", h.DeleteAssetStoreRecord)

	p.POST("/accounts/:accountId/snapshots", h.CreateAccountSnapshot)
	p.GET("/accounts/:accountId/snapshots", h.ListAccountSnapshots)
	p.GET("/accounts/:accountId/snapshots/:snapshotId", h.GetAccountSnapshot)
	p.PUT("/accounts/:accountId/snapshots/:snapshotId", h.UpdateAccountSnapshot)
	p.DELETE("/accounts/:accountId/snapshots/:snapshotId", h.DeleteAccountSnapshot)
}

func (h *Handler) registerAssetRoutes(p *gin.RouterGroup) {
	p.POST("/assets", h.CreateAsset)
	p.GET("/assets", h.ListAssets)
	p.GET("/assets/:assetId", h.GetAsset)
	p.PUT("/assets/:assetId", h.UpdateAsset)
	p.DELETE("/assets/:assetId", h.DeleteAsset)

	p.POST("/assets/:assetId/values", h.CreateAssetValue)
	p.GET("/assets/:assetId/values", h.ListAssetValues)
	p.GET("/assets/:assetId/values/:valueId", h.GetAssetValue)
	p.PUT("/assets/:assetId/values/:valueId", h.UpdateAssetValue)
	p.DELETE("/assets/:assetId/values/:valueId", h.DeleteAssetValue)
}

func (h *Handler) registerCategoryRoutes(p *gin.RouterGroup) {
	p.POST("/categories", h.CreateCategory)
	p.GET("/categories", h.ListCategories)
	p.GET("/categories/:categoryId", h.GetCategory)
	p.DELETE("/categories/:categoryId", h.DeleteCategory)
}

func (h *Handler) registerPerspectiveRoutes(p *gin.RouterGroup) {
	p.POST("/perspectives", h.CreatePerspective)
	p.GET("/perspectives", h.ListPerspectives)
	p.GET("/perspectives/:perspectiveId", h.GetPerspective)
	p.PUT("/perspectives/:perspectiveId", h.UpdatePerspective)
	p.DELETE("/perspectives/:perspectiveId", h.DeletePerspective)

	p.GET("/perspectives/:perspectiveId/settings", h.GetPerspectiveSettings)
	p.PUT("/perspectives/:perspectiveId/settings", h.UpdatePerspectiveSettings)

	p.GET("/perspectives/:perspectiveId/accounts", h.ListPerspectiveAccounts)
	p.POST("/perspectives/:perspectiveId/accounts", h.LinkPerspectiveAccount)
	p.DELETE("/perspectives/:perspectiveId/accounts/:accountId", h.UnlinkPerspectiveAccount)

	p.GET("/perspectives/:perspectiveId/access", h.ListPerspectiveAccess)
	p.GET("/perspectives/:perspectiveId/access/:userId", h.GetPerspectiveAccess)
	p.POST("/perspectives/:perspectiveId/access", h.GrantPerspectiveAccess)
	p.DELETE("/perspectives/:perspectiveId/access/:userId", h.RevokePerspectiveAccess)

	p.GET("/perspectives/:perspectiveId/view", h.GetPerspectiveView)
}

func (h *Handler) registerRecurringTransactionRoutes(p *gin.RouterGroup) {
	p.POST("/recurring-transactions", h.CreateRecurringTransaction)
	p.GET("/recurring-transactions", h.ListRecurringTransactions)
	p.GET("/recurring-transactions/reminders", h.ListRecurringReminders)
	p.GET("/recurring-transactions/:recurringId", h.GetRecurringTransaction)
	p.PUT("/recurring-transactions/:recurringId", h.UpdateRecurringTransaction)
	p.DELETE("/recurring-transactions/:recurringId", h.DeleteRecurringTransaction)
	p.POST("/recurring-transactions/:recurringId/apply", h.ApplyRecurringTransactionToAccount)
}
