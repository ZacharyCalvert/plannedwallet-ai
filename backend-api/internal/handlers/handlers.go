package handlers

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/repository"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type Handler struct {
	repo repository.Repository
	auth *auth.Service
}

func NewHandler(repo repository.Repository, authService *auth.Service) *Handler {
	return &Handler{
		repo: repo,
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

func parseDate(dateStr string) error {
	_, err := time.Parse("2006-01-02", dateStr)
	return err
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

	if len(req.Password) < 8 {
		h.sendError(c, http.StatusBadRequest, "INVALID_PASSWORD", "Password must be at least 8 characters long.")
		return
	}

	hash, err := h.auth.HashPassword(req.Password)
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to hash password.")
		return
	}

	user := &models.User{
		ID:           uuid.New().String(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
		CreatedAt:    time.Now().UTC(),
	}

	if err := h.repo.CreateUser(c.Request.Context(), user); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			h.sendError(c, http.StatusConflict, "USER_CONFLICT", "Username or email is already registered.")
			return
		}
		h.sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create user.")
		return
	}

	c.JSON(http.StatusCreated, models.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	})
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

	user, err := h.repo.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.sendError(c, http.StatusNotFound, "USER_NOT_FOUND", "No user found matching the provided username.")
			return
		}
		h.sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Error retrieving user.")
		return
	}

	if !h.auth.CheckPassword(user.PasswordHash, req.Password) {
		h.sendError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid username or password.")
		return
	}

	token, err := h.auth.GenerateToken(user.ID, user.Username)
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate token.")
		return
	}

	c.JSON(http.StatusOK, models.TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   h.auth.ExpirationSeconds(),
	})
}

// CreateAccount handles POST /api/v1/accounts
func (h *Handler) CreateAccount(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		h.sendError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authenticated user.")
		return
	}
	userID := userIDVal.(string)

	var req models.AccountCreationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Type = strings.TrimSpace(req.Type)
	req.Description = strings.TrimSpace(req.Description)

	if req.Name == "" {
		h.sendError(c, http.StatusBadRequest, "MISSING_NAME", "Account name cannot be empty.")
		return
	}
	if !models.ValidAccountTypes[req.Type] {
		h.sendError(c, http.StatusBadRequest, "INVALID_ACCOUNT_TYPE", "Invalid account type provided.")
		return
	}
	if req.Description == "" {
		h.sendError(c, http.StatusBadRequest, "MISSING_DESCRIPTION", "Account description cannot be empty.")
		return
	}

	now := time.Now().UTC()
	account := &models.Account{
		ID:          uuid.New().String(),
		UserID:      userID,
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.repo.CreateAccount(c.Request.Context(), account); err != nil {
		h.sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create account.")
		return
	}

	c.JSON(http.StatusCreated, models.AccountResponse{
		ID:          account.ID,
		UserID:      account.UserID,
		Name:        account.Name,
		Type:        account.Type,
		Description: account.Description,
		CreatedAt:   account.CreatedAt,
		UpdatedAt:   account.UpdatedAt,
	})
}

// verifyCheckingAccount checks that the account exists, belongs to the authenticated user, and is of type checking.
func (h *Handler) verifyCheckingAccount(c *gin.Context, accountID string) (*models.Account, bool) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		h.sendError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authenticated user.")
		return nil, false
	}
	userID := userIDVal.(string)

	account, err := h.repo.GetAccountByID(c.Request.Context(), accountID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.sendError(c, http.StatusNotFound, "ACCOUNT_NOT_FOUND", "No checking account found matching the provided accountId.")
			return nil, false
		}
		h.sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to verify account.")
		return nil, false
	}

	if account.UserID != userID || account.Type != "checking" {
		h.sendError(c, http.StatusNotFound, "ACCOUNT_NOT_FOUND", "No checking account found matching the provided accountId.")
		return nil, false
	}

	return account, true
}

// CreateCheckingRecord handles POST /api/v1/accounts/{accountId}/checking-records
func (h *Handler) CreateCheckingRecord(c *gin.Context) {
	accountID := c.Param("accountId")
	if _, ok := h.verifyCheckingAccount(c, accountID); !ok {
		return
	}

	var req models.CheckingRecordCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	if req.Value == nil {
		h.sendError(c, http.StatusBadRequest, "INVALID_VALUE", "Transaction value is required.")
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

	entity := strings.TrimSpace(req.Entity)
	if entity == "" {
		h.sendError(c, http.StatusBadRequest, "INVALID_ENTITY", "entity cannot be empty.")
		return
	}

	now := time.Now().UTC()
	record := &models.CheckingRecord{
		ID:          uuid.New().String(),
		AccountID:   accountID,
		Value:       *req.Value,
		DateSent:    req.DateSent,
		DateCleared: req.DateCleared,
		Entity:      entity,
		Description: strings.TrimSpace(req.Description),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.repo.CreateCheckingRecord(c.Request.Context(), record); err != nil {
		h.sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create checking record.")
		return
	}

	c.JSON(http.StatusCreated, models.CheckingRecordResponse{
		ID:          record.ID,
		AccountID:   record.AccountID,
		Value:       record.Value,
		DateSent:    record.DateSent,
		DateCleared: record.DateCleared,
		Entity:      record.Entity,
		Description: record.Description,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	})
}

// ListCheckingRecords handles GET /api/v1/accounts/{accountId}/checking-records
func (h *Handler) ListCheckingRecords(c *gin.Context) {
	accountID := c.Param("accountId")
	if _, ok := h.verifyCheckingAccount(c, accountID); !ok {
		return
	}

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

	records, err := h.repo.ListCheckingRecords(c.Request.Context(), accountID, clearedFilter, startDateFilter, endDateFilter)
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve checking records.")
		return
	}

	res := make([]models.CheckingRecordResponse, 0, len(records))
	for _, r := range records {
		res = append(res, models.CheckingRecordResponse{
			ID:          r.ID,
			AccountID:   r.AccountID,
			Value:       r.Value,
			DateSent:    r.DateSent,
			DateCleared: r.DateCleared,
			Entity:      r.Entity,
			Description: r.Description,
			CreatedAt:   r.CreatedAt,
			UpdatedAt:   r.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, res)
}

// GetCheckingRecord handles GET /api/v1/accounts/{accountId}/checking-records/{recordId}
func (h *Handler) GetCheckingRecord(c *gin.Context) {
	accountID := c.Param("accountId")
	if _, ok := h.verifyCheckingAccount(c, accountID); !ok {
		return
	}

	recordID := c.Param("recordId")
	record, err := h.repo.GetCheckingRecordByID(c.Request.Context(), accountID, recordID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.sendError(c, http.StatusNotFound, "RECORD_NOT_FOUND", "Checking record not found.")
			return
		}
		h.sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve record.")
		return
	}

	c.JSON(http.StatusOK, models.CheckingRecordResponse{
		ID:          record.ID,
		AccountID:   record.AccountID,
		Value:       record.Value,
		DateSent:    record.DateSent,
		DateCleared: record.DateCleared,
		Entity:      record.Entity,
		Description: record.Description,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	})
}

// UpdateCheckingRecord handles PUT /api/v1/accounts/{accountId}/checking-records/{recordId}
func (h *Handler) UpdateCheckingRecord(c *gin.Context) {
	accountID := c.Param("accountId")
	if _, ok := h.verifyCheckingAccount(c, accountID); !ok {
		return
	}

	recordID := c.Param("recordId")
	existing, err := h.repo.GetCheckingRecordByID(c.Request.Context(), accountID, recordID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.sendError(c, http.StatusNotFound, "RECORD_NOT_FOUND", "Checking record not found.")
			return
		}
		h.sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve record.")
		return
	}

	var req models.CheckingRecordUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	if req.Value == nil {
		h.sendError(c, http.StatusBadRequest, "INVALID_VALUE", "Transaction value is required.")
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

	entity := strings.TrimSpace(req.Entity)
	if entity == "" {
		h.sendError(c, http.StatusBadRequest, "INVALID_ENTITY", "entity cannot be empty.")
		return
	}

	existing.Value = *req.Value
	existing.DateSent = req.DateSent
	existing.DateCleared = req.DateCleared
	existing.Entity = entity
	existing.Description = strings.TrimSpace(req.Description)
	existing.UpdatedAt = time.Now().UTC()

	if err := h.repo.UpdateCheckingRecord(c.Request.Context(), existing); err != nil {
		h.sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update record.")
		return
	}

	c.JSON(http.StatusOK, models.CheckingRecordResponse{
		ID:          existing.ID,
		AccountID:   existing.AccountID,
		Value:       existing.Value,
		DateSent:    existing.DateSent,
		DateCleared: existing.DateCleared,
		Entity:      existing.Entity,
		Description: existing.Description,
		CreatedAt:   existing.CreatedAt,
		UpdatedAt:   existing.UpdatedAt,
	})
}

// PatchCheckingRecord handles PATCH /api/v1/accounts/{accountId}/checking-records/{recordId}
func (h *Handler) PatchCheckingRecord(c *gin.Context) {
	accountID := c.Param("accountId")
	if _, ok := h.verifyCheckingAccount(c, accountID); !ok {
		return
	}

	recordID := c.Param("recordId")
	existing, err := h.repo.GetCheckingRecordByID(c.Request.Context(), accountID, recordID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.sendError(c, http.StatusNotFound, "RECORD_NOT_FOUND", "Checking record not found.")
			return
		}
		h.sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve record.")
		return
	}

	var req models.CheckingRecordPatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	if req.Value != nil {
		existing.Value = *req.Value
	}
	if req.DateSent != nil {
		if err := parseDate(*req.DateSent); err != nil {
			h.sendError(c, http.StatusBadRequest, "INVALID_DATE_SENT", "date_sent must be formatted as YYYY-MM-DD.")
			return
		}
		existing.DateSent = *req.DateSent
	}
	if req.DateCleared != nil {
		if *req.DateCleared != "" {
			if err := parseDate(*req.DateCleared); err != nil {
				h.sendError(c, http.StatusBadRequest, "INVALID_DATE_CLEARED", "date_cleared must be formatted as YYYY-MM-DD.")
				return
			}
			existing.DateCleared = req.DateCleared
		} else {
			existing.DateCleared = nil
		}
	}
	if req.Entity != nil {
		entity := strings.TrimSpace(*req.Entity)
		if entity == "" {
			h.sendError(c, http.StatusBadRequest, "INVALID_ENTITY", "entity cannot be empty.")
			return
		}
		existing.Entity = entity
	}
	if req.Description != nil {
		existing.Description = strings.TrimSpace(*req.Description)
	}
	existing.UpdatedAt = time.Now().UTC()

	if err := h.repo.UpdateCheckingRecord(c.Request.Context(), existing); err != nil {
		h.sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update record.")
		return
	}

	c.JSON(http.StatusOK, models.CheckingRecordResponse{
		ID:          existing.ID,
		AccountID:   existing.AccountID,
		Value:       existing.Value,
		DateSent:    existing.DateSent,
		DateCleared: existing.DateCleared,
		Entity:      existing.Entity,
		Description: existing.Description,
		CreatedAt:   existing.CreatedAt,
		UpdatedAt:   existing.UpdatedAt,
	})
}

// DeleteCheckingRecord handles DELETE /api/v1/accounts/{accountId}/checking-records/{recordId}
func (h *Handler) DeleteCheckingRecord(c *gin.Context) {
	accountID := c.Param("accountId")
	if _, ok := h.verifyCheckingAccount(c, accountID); !ok {
		return
	}

	recordID := c.Param("recordId")
	if err := h.repo.DeleteCheckingRecord(c.Request.Context(), accountID, recordID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.sendError(c, http.StatusNotFound, "RECORD_NOT_FOUND", "Checking record not found.")
			return
		}
		h.sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete record.")
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
