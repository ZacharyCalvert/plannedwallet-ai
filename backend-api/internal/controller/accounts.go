package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/repository"
	"github.com/plannedwallet/internal/trace"
)

const maxAccountFieldDigits = 4

func countDigits(s string) int {
	count := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			count++
		}
	}
	return count
}

func validateNoProhibitedDigits(val, fieldLabel string) error {
	if countDigits(val) > maxAccountFieldDigits {
		return NewAppError(
			http.StatusBadRequest,
			"PROHIBITED_SENSITIVE_DIGITS",
			fmt.Sprintf("%s cannot contain 5 or more digits (Planned Wallet prohibits storing account, routing, credit card, or Social Security numbers).", fieldLabel),
		)
	}
	return nil
}

func normalizeAccountType(raw string) string {
	accType := strings.TrimSpace(raw)
	switch accType {
	case "credit_card":
		return "credit card"
	case "assets":
		return "asset"
	default:
		return accType
	}
}

func validateAccountRequest(req models.AccountCreationRequest) (string, string, string, error) {
	name := strings.TrimSpace(req.Name)
	accType := normalizeAccountType(req.Type)
	description := strings.TrimSpace(req.Description)

	if name == "" {
		return "", "", "", NewAppError(http.StatusBadRequest, "MISSING_NAME", "Account name cannot be empty.")
	}
	if err := validateNoProhibitedDigits(name, "Account name"); err != nil {
		return "", "", "", err
	}
	if !models.ValidAccountTypes[accType] {
		return "", "", "", NewAppError(http.StatusBadRequest, "INVALID_ACCOUNT_TYPE", "Invalid account type provided.")
	}
	if description == "" {
		return "", "", "", NewAppError(http.StatusBadRequest, "MISSING_DESCRIPTION", "Account description cannot be empty.")
	}
	if err := validateNoProhibitedDigits(description, "Account description"); err != nil {
		return "", "", "", err
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

	return toAccountResponse(account), nil
}

func toAccountResponse(account *models.Account) *models.AccountResponse {
	return &models.AccountResponse{
		ID:          account.ID,
		AdminID:     account.AdminID,
		Name:        account.Name,
		Type:        account.Type,
		Description: account.Description,
		CreatedAt:   account.CreatedAt,
		UpdatedAt:   account.UpdatedAt,
	}
}

func validateTrimmedField(raw string, maxRunes int, missingCode, invalidCode, fieldLabel string) (string, error) {
	val := strings.TrimSpace(raw)
	if val == "" {
		return "", NewAppError(http.StatusBadRequest, missingCode, fieldLabel+" cannot be empty.")
	}
	if len([]rune(val)) > maxRunes {
		return "", NewAppError(http.StatusBadRequest, invalidCode, fmt.Sprintf("%s must not exceed %d characters.", fieldLabel, maxRunes))
	}
	if err := validateNoProhibitedDigits(val, fieldLabel); err != nil {
		return "", err
	}
	return val, nil
}

func resolveOptionalField(ptr *string, fallback string, maxRunes int, missingCode, invalidCode, label string) (string, error) {
	if ptr == nil {
		return fallback, nil
	}
	return validateTrimmedField(*ptr, maxRunes, missingCode, invalidCode, label)
}

func resolveAccountPatchFields(existing *models.Account, req models.AccountPatchRequest) (string, string, error) {
	if req.Name == nil && req.Description == nil {
		return "", "", NewAppError(http.StatusBadRequest, "MISSING_PATCH_FIELDS", "At least one of name or description must be provided.")
	}
	name, err := resolveOptionalField(req.Name, existing.Name, 100, "MISSING_NAME", "INVALID_NAME", "Account name")
	if err != nil {
		return "", "", err
	}
	desc, err := resolveOptionalField(req.Description, existing.Description, 255, "MISSING_DESCRIPTION", "INVALID_DESCRIPTION", "Account description")
	if err != nil {
		return "", "", err
	}
	return name, desc, nil
}

// PatchAccount updates an existing account's name and/or description. Requires edit or admin access.
func (c *Controller) PatchAccount(ctx context.Context, userID, accountID string, req models.AccountPatchRequest) (*models.AccountResponse, error) {
	existing, err := c.verifyAccountAccess(ctx, userID, accountID, models.AccessLevelEdit)
	if err != nil {
		return nil, err
	}
	name, desc, err := resolveAccountPatchFields(existing, req)
	if err != nil {
		return nil, err
	}

	updated, err := c.repo.UpdateAccount(ctx, accountID, name, desc, time.Now().UTC())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrAccountNotFound
		}
		trace.LogError(ctx, err, "Failed to update account %s", accountID)
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update account.")
	}

	trace.LogInfo(ctx, "Account %s updated (name=%q) by user %s", accountID, name, userID)
	return toAccountResponse(updated), nil
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

