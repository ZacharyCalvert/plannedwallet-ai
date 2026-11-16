package controller

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/repository"
	"github.com/plannedwallet/internal/trace"
)

func validateAccountRequest(req models.AccountCreationRequest) (string, string, string, error) {
	name := strings.TrimSpace(req.Name)
	accType := strings.TrimSpace(req.Type)
	description := strings.TrimSpace(req.Description)

	if accType == "credit_card" {
		accType = "credit card"
	}
	if accType == "assets" {
		accType = "asset"
	}
	if name == "" {
		return "", "", "", NewAppError(http.StatusBadRequest, "MISSING_NAME", "Account name cannot be empty.")
	}
	if !models.ValidAccountTypes[accType] {
		return "", "", "", NewAppError(http.StatusBadRequest, "INVALID_ACCOUNT_TYPE", "Invalid account type provided.")
	}
	if description == "" {
		return "", "", "", NewAppError(http.StatusBadRequest, "MISSING_DESCRIPTION", "Account description cannot be empty.")
	}
	return name, accType, description, nil
}

func (c *Controller) grantInitialAdminAccess(ctx context.Context, accountID, userID string, now time.Time) {
	access := &models.AccountAccess{
		ID:          uuid.New().String(),
		AccountID:   accountID,
		UserID:      userID,
		AccessLevel: models.AccessLevelAdmin,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_ = c.repo.GrantAccountAccess(ctx, access)
}

// CreateAccount validates account constraints and creates a new financial account.
func (c *Controller) CreateAccount(ctx context.Context, userID string, req models.AccountCreationRequest) (*models.AccountResponse, error) {
	name, accType, description, err := validateAccountRequest(req)
	if err != nil {
		return nil, err
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
		trace.LogError(ctx, err, "Failed to create account %s for user %s", name, userID)
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create account.")
	}

	c.grantInitialAdminAccess(ctx, account.ID, userID, now)
	trace.LogInfo(ctx, "Account created successfully: %s (id: %s) for user: %s", account.Name, account.ID, userID)

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

// DeleteAccount deletes an account. Enforces that only the creator (admin_id) can delete it.
func (c *Controller) DeleteAccount(ctx context.Context, userID, accountID string) error {
	account, err := c.repo.GetAccountByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrAccountNotFound
		}
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve account.")
	}

	if account.AdminID != userID {
		return NewAppError(http.StatusForbidden, "FORBIDDEN", "Only the account creator can delete the account.")
	}

	if err := c.repo.DeleteAccount(ctx, accountID, userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrAccountNotFound
		}
		trace.LogError(ctx, err, "Failed to delete account %s", accountID)
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete account.")
	}
	trace.LogInfo(ctx, "Account deleted successfully: %s by user: %s", accountID, userID)
	return nil
}

