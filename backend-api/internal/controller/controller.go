package controller

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/repository"
)

// AppError represents a domain or business logic error mapped to an HTTP status code.
type AppError struct {
	Status  int
	Code    string
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

func NewAppError(status int, code, message string) *AppError {
	return &AppError{
		Status:  status,
		Code:    code,
		Message: message,
	}
}

var (
	ErrUserAlreadyExists  = NewAppError(http.StatusConflict, "USER_CONFLICT", "Username or email is already registered.")
	ErrUserNotFound       = NewAppError(http.StatusNotFound, "USER_NOT_FOUND", "No user found matching the provided username.")
	ErrInvalidCredentials = NewAppError(http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid username or password.")
	ErrAccountNotFound    = NewAppError(http.StatusNotFound, "ACCOUNT_NOT_FOUND", "No checking account found matching the provided accountId.")
	ErrRecordNotFound     = NewAppError(http.StatusNotFound, "RECORD_NOT_FOUND", "Checking record not found.")
)

// ValidatePassword enforces password rules:
// minimum length of 10 characters, at least 1 uppercase, 1 lowercase, 1 number, and 1 non-alphanumeric special character.
func ValidatePassword(password string) error {
	if len(password) < 10 {
		return NewAppError(http.StatusBadRequest, "INVALID_PASSWORD", "Password must be at least 10 characters long.")
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return NewAppError(http.StatusBadRequest, "INVALID_PASSWORD", "Password must contain at least one uppercase letter.")
	}
	if !hasLower {
		return NewAppError(http.StatusBadRequest, "INVALID_PASSWORD", "Password must contain at least one lowercase letter.")
	}
	if !hasDigit {
		return NewAppError(http.StatusBadRequest, "INVALID_PASSWORD", "Password must contain at least one number.")
	}
	if !hasSpecial {
		return NewAppError(http.StatusBadRequest, "INVALID_PASSWORD", "Password must contain at least one special character not in the alphanumeric set.")
	}

	return nil
}

type Controller struct {
	repo repository.Repository
	auth *auth.Service
}

func NewController(repo repository.Repository, authService *auth.Service) *Controller {
	return &Controller{
		repo: repo,
		auth: authService,
	}
}

// RegisterUser validates user existence, verifies password rules, hashes password, and persists user.
func (c *Controller) RegisterUser(ctx context.Context, req models.UserRegistrationRequest) (*models.UserResponse, error) {
	username := strings.TrimSpace(req.Username)
	email := strings.TrimSpace(req.Email)

	if err := ValidatePassword(req.Password); err != nil {
		return nil, err
	}

	// Validate user does not already exist by username
	existingUser, err := c.repo.GetUserByUsername(ctx, username)
	if err == nil && existingUser != nil {
		return nil, ErrUserAlreadyExists
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check existing username.")
	}

	// Validate user does not already exist by email
	existingEmail, err := c.repo.GetUserByEmail(ctx, email)
	if err == nil && existingEmail != nil {
		return nil, ErrUserAlreadyExists
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check existing email.")
	}

	hash, err := c.auth.HashPassword(req.Password)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to hash password.")
	}

	user := &models.User{
		ID:           uuid.New().String(),
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    time.Now().UTC(),
	}

	if err := c.repo.CreateUser(ctx, user); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrUserAlreadyExists
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create user.")
	}

	return &models.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}, nil
}

// ExchangeToken validates user credentials and issues a signed JWT token.
func (c *Controller) ExchangeToken(ctx context.Context, username, password string) (*models.TokenResponse, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, NewAppError(http.StatusBadRequest, "BAD_REQUEST", "Username is required.")
	}

	user, err := c.repo.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Error retrieving user.")
	}

	if !c.auth.CheckPassword(user.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}

	token, err := c.auth.GenerateToken(user.ID, user.Username)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate token.")
	}

	return &models.TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   c.auth.ExpirationSeconds(),
	}, nil
}

// CreateAccount validates account constraints and creates a new financial account.
func (c *Controller) CreateAccount(ctx context.Context, userID string, req models.AccountCreationRequest) (*models.AccountResponse, error) {
	name := strings.TrimSpace(req.Name)
	accType := strings.TrimSpace(req.Type)
	description := strings.TrimSpace(req.Description)

	if name == "" {
		return nil, NewAppError(http.StatusBadRequest, "MISSING_NAME", "Account name cannot be empty.")
	}
	if !models.ValidAccountTypes[accType] {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_ACCOUNT_TYPE", "Invalid account type provided.")
	}
	if description == "" {
		return nil, NewAppError(http.StatusBadRequest, "MISSING_DESCRIPTION", "Account description cannot be empty.")
	}

	now := time.Now().UTC()
	account := &models.Account{
		ID:          uuid.New().String(),
		AdminID:     userID,
		Name:        name,
		Type:        accType,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := c.repo.CreateAccount(ctx, account); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create account.")
	}

	// Establish admin level access for the account creator
	access := &models.AccountAccess{
		ID:          uuid.New().String(),
		AccountID:   account.ID,
		UserID:      userID,
		AccessLevel: models.AccessLevelAdmin,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_ = c.repo.GrantAccountAccess(ctx, access)

	return &models.AccountResponse{
		ID:          account.ID,
		AdminID:     account.AdminID,
		Name:        account.Name,
		Type:        account.Type,
		Description: account.Description,
		CreatedAt:   account.CreatedAt,
		UpdatedAt:   account.UpdatedAt,
	}, nil
}

// verifyCheckingAccount verifies that the account exists, belongs to or is accessible by the user, and is of type checking.
func (c *Controller) verifyCheckingAccount(ctx context.Context, userID, accountID string) (*models.Account, error) {
	account, err := c.repo.GetAccountByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to verify account.")
	}

	if account.Type != "checking" {
		return nil, ErrAccountNotFound
	}

	if account.AdminID != userID {
		access, err := c.repo.GetAccountAccess(ctx, accountID, userID)
		if err != nil || access == nil {
			return nil, ErrAccountNotFound
		}
	}

	return account, nil
}

// CreateCheckingRecord creates a checking transaction record under the specified account.
func (c *Controller) CreateCheckingRecord(ctx context.Context, userID, accountID string, req models.CheckingRecordCreateRequest) (*models.CheckingRecordResponse, error) {
	if _, err := c.verifyCheckingAccount(ctx, userID, accountID); err != nil {
		return nil, err
	}

	if req.Value == nil {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_VALUE", "Transaction value is required.")
	}

	entity := strings.TrimSpace(req.Entity)
	if entity == "" {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_ENTITY", "entity cannot be empty.")
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

	if err := c.repo.CreateCheckingRecord(ctx, record); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create checking record.")
	}

	return &models.CheckingRecordResponse{
		ID:          record.ID,
		AccountID:   record.AccountID,
		Value:       record.Value,
		DateSent:    record.DateSent,
		DateCleared: record.DateCleared,
		Entity:      record.Entity,
		Description: record.Description,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}, nil
}

// ListCheckingRecords retrieves checking records with optional filters.
func (c *Controller) ListCheckingRecords(ctx context.Context, userID, accountID string, cleared *bool, startDate, endDate *string) ([]models.CheckingRecordResponse, error) {
	if _, err := c.verifyCheckingAccount(ctx, userID, accountID); err != nil {
		return nil, err
	}

	records, err := c.repo.ListCheckingRecords(ctx, accountID, cleared, startDate, endDate)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve checking records.")
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
	return res, nil
}

// GetCheckingRecord fetches a single checking record by ID.
func (c *Controller) GetCheckingRecord(ctx context.Context, userID, accountID, recordID string) (*models.CheckingRecordResponse, error) {
	if _, err := c.verifyCheckingAccount(ctx, userID, accountID); err != nil {
		return nil, err
	}

	record, err := c.repo.GetCheckingRecordByID(ctx, accountID, recordID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve record.")
	}

	return &models.CheckingRecordResponse{
		ID:          record.ID,
		AccountID:   record.AccountID,
		Value:       record.Value,
		DateSent:    record.DateSent,
		DateCleared: record.DateCleared,
		Entity:      record.Entity,
		Description: record.Description,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}, nil
}

// UpdateCheckingRecord updates all mutable fields of an existing checking record.
func (c *Controller) UpdateCheckingRecord(ctx context.Context, userID, accountID, recordID string, req models.CheckingRecordUpdateRequest) (*models.CheckingRecordResponse, error) {
	if _, err := c.verifyCheckingAccount(ctx, userID, accountID); err != nil {
		return nil, err
	}

	existing, err := c.repo.GetCheckingRecordByID(ctx, accountID, recordID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve record.")
	}

	if req.Value == nil {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_VALUE", "Transaction value is required.")
	}

	entity := strings.TrimSpace(req.Entity)
	if entity == "" {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_ENTITY", "entity cannot be empty.")
	}

	existing.Value = *req.Value
	existing.DateSent = req.DateSent
	existing.DateCleared = req.DateCleared
	existing.Entity = entity
	existing.Description = strings.TrimSpace(req.Description)
	existing.UpdatedAt = time.Now().UTC()

	if err := c.repo.UpdateCheckingRecord(ctx, existing); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update record.")
	}

	return &models.CheckingRecordResponse{
		ID:          existing.ID,
		AccountID:   existing.AccountID,
		Value:       existing.Value,
		DateSent:    existing.DateSent,
		DateCleared: existing.DateCleared,
		Entity:      existing.Entity,
		Description: existing.Description,
		CreatedAt:   existing.CreatedAt,
		UpdatedAt:   existing.UpdatedAt,
	}, nil
}

// PatchCheckingRecord partially updates an existing checking record.
func (c *Controller) PatchCheckingRecord(ctx context.Context, userID, accountID, recordID string, req models.CheckingRecordPatchRequest) (*models.CheckingRecordResponse, error) {
	if _, err := c.verifyCheckingAccount(ctx, userID, accountID); err != nil {
		return nil, err
	}

	existing, err := c.repo.GetCheckingRecordByID(ctx, accountID, recordID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve record.")
	}

	if req.Value != nil {
		existing.Value = *req.Value
	}
	if req.DateSent != nil {
		existing.DateSent = *req.DateSent
	}
	if req.DateCleared != nil {
		if *req.DateCleared != "" {
			existing.DateCleared = req.DateCleared
		} else {
			existing.DateCleared = nil
		}
	}
	if req.Entity != nil {
		entity := strings.TrimSpace(*req.Entity)
		if entity == "" {
			return nil, NewAppError(http.StatusBadRequest, "INVALID_ENTITY", "entity cannot be empty.")
		}
		existing.Entity = entity
	}
	if req.Description != nil {
		existing.Description = strings.TrimSpace(*req.Description)
	}
	existing.UpdatedAt = time.Now().UTC()

	if err := c.repo.UpdateCheckingRecord(ctx, existing); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update record.")
	}

	return &models.CheckingRecordResponse{
		ID:          existing.ID,
		AccountID:   existing.AccountID,
		Value:       existing.Value,
		DateSent:    existing.DateSent,
		DateCleared: existing.DateCleared,
		Entity:      existing.Entity,
		Description: existing.Description,
		CreatedAt:   existing.CreatedAt,
		UpdatedAt:   existing.UpdatedAt,
	}, nil
}

// DeleteCheckingRecord deletes a checking record.
func (c *Controller) DeleteCheckingRecord(ctx context.Context, userID, accountID, recordID string) error {
	if _, err := c.verifyCheckingAccount(ctx, userID, accountID); err != nil {
		return err
	}

	if err := c.repo.DeleteCheckingRecord(ctx, accountID, recordID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrRecordNotFound
		}
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete record.")
	}

	return nil
}

// GrantAccountAccess allows an account admin to grant or update access permissions for another user.
func (c *Controller) GrantAccountAccess(ctx context.Context, adminUserID, accountID, targetUserID string, level models.AccessLevel) (*models.AccountAccessResponse, error) {
	if !models.ValidAccessLevels[level] {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_ACCESS_LEVEL", "Access level must be read, edit, or admin.")
	}

	account, err := c.repo.GetAccountByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve account.")
	}

	if account.AdminID != adminUserID {
		return nil, NewAppError(http.StatusForbidden, "FORBIDDEN", "Only the account admin can grant access permissions.")
	}

	if _, err := c.repo.GetUserByID(ctx, targetUserID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, NewAppError(http.StatusNotFound, "USER_NOT_FOUND", "Target user not found.")
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to verify target user.")
	}

	now := time.Now().UTC()
	access := &models.AccountAccess{
		ID:          uuid.New().String(),
		AccountID:   accountID,
		UserID:      targetUserID,
		AccessLevel: level,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := c.repo.GrantAccountAccess(ctx, access); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to grant account access.")
	}

	return &models.AccountAccessResponse{
		ID:          access.ID,
		AccountID:   access.AccountID,
		UserID:      access.UserID,
		AccessLevel: access.AccessLevel,
		CreatedAt:   access.CreatedAt,
		UpdatedAt:   access.UpdatedAt,
	}, nil
}

// GetAccountAccess retrieves an access grant for a specific user and account.
func (c *Controller) GetAccountAccess(ctx context.Context, requestingUserID, accountID, targetUserID string) (*models.AccountAccessResponse, error) {
	account, err := c.repo.GetAccountByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve account.")
	}

	if account.AdminID != requestingUserID && requestingUserID != targetUserID {
		return nil, NewAppError(http.StatusForbidden, "FORBIDDEN", "Insufficient permissions to view access details.")
	}

	access, err := c.repo.GetAccountAccess(ctx, accountID, targetUserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, NewAppError(http.StatusNotFound, "ACCESS_NOT_FOUND", "No access grant found for user on this account.")
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve access details.")
	}

	return &models.AccountAccessResponse{
		ID:          access.ID,
		AccountID:   access.AccountID,
		UserID:      access.UserID,
		AccessLevel: access.AccessLevel,
		CreatedAt:   access.CreatedAt,
		UpdatedAt:   access.UpdatedAt,
	}, nil
}

// RevokeAccountAccess allows an account admin to revoke another user's access to an account.
func (c *Controller) RevokeAccountAccess(ctx context.Context, adminUserID, accountID, targetUserID string) error {
	account, err := c.repo.GetAccountByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrAccountNotFound
		}
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve account.")
	}

	if account.AdminID != adminUserID {
		return NewAppError(http.StatusForbidden, "FORBIDDEN", "Only the account admin can revoke access.")
	}

	if err := c.repo.RevokeAccountAccess(ctx, accountID, targetUserID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return NewAppError(http.StatusNotFound, "ACCESS_NOT_FOUND", "No access grant found to revoke.")
		}
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to revoke access.")
	}

	return nil
}
