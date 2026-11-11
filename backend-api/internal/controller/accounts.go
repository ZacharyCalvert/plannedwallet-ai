package controller

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/trace"
)

func validateAccountRequest(req models.AccountCreationRequest) (string, string, string, error) {
	name := strings.TrimSpace(req.Name)
	accType := strings.TrimSpace(req.Type)
	description := strings.TrimSpace(req.Description)

	if accType == "credit_card" {
		accType = "credit card"
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
