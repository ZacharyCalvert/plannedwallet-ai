package controller

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/repository"
	"github.com/plannedwallet/internal/trace"
)

func validateMobileIdentifier(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || len(trimmed) > 256 {
		return "", NewAppError(
			http.StatusBadRequest,
			"INVALID_MOBILE_IDENTIFIER",
			"X-Mobile-Identifier header is required and must not exceed 256 characters.",
		)
	}
	return trimmed, nil
}

// IssueMobileToken mints a 6-month device-bound JWT with limitation="mobile" and persists it for revocation.
func (c *Controller) IssueMobileToken(ctx context.Context, userID, mobileIdentifier string) (*models.MobileTokenResponse, error) {
	cleanID, err := validateMobileIdentifier(mobileIdentifier)
	if err != nil {
		return nil, err
	}
	user, err := c.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve user.")
	}
	return c.persistAndSignMobileToken(ctx, user, cleanID)
}

func (c *Controller) persistAndSignMobileToken(ctx context.Context, user *models.User, cleanMobileID string) (*models.MobileTokenResponse, error) {
	isSub, err := c.resolveSubscriberStatus(ctx, user.ID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check subscriber status.")
	}
	termsApproved, err := c.resolveTokenTermsApproved(ctx, user.ID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check policy agreement status.")
	}
	deviceHash := auth.HashMobileIdentifier(cleanMobileID)
	tokenID := uuid.New().String()
	signed, exp, err := c.auth.GenerateMobileToken(tokenID, user.ID, user.Username, isSub, termsApproved, deviceHash)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate mobile token.")
	}
	record := &models.MobileToken{
		ID:         tokenID,
		UserID:     user.ID,
		DeviceHash: deviceHash,
		ExpiresAt:  exp,
		CreatedAt:  time.Now().UTC(),
	}
	if err := c.repo.CreateMobileToken(ctx, record); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to persist mobile token.")
	}
	trace.LogInfo(ctx, "Issued mobile widget token %s for user %s", tokenID, user.Username)
	return &models.MobileTokenResponse{
		AccessToken: signed,
		TokenType:   "Bearer",
		ExpiresIn:   c.auth.MobileExpirationSeconds(),
		ExpiresAt:   exp,
		Limitation:  auth.LimitationMobile,
	}, nil
}

// ValidateActiveMobileToken verifies the persisted mobile token exists, belongs to userID, and is unrevoked.
func (c *Controller) ValidateActiveMobileToken(ctx context.Context, tokenID, userID string) error {
	mt, err := c.repo.GetMobileTokenByID(ctx, tokenID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return NewAppError(http.StatusUnauthorized, "TOKEN_REVOKED", "Mobile token is inactive or has been revoked.")
		}
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to verify mobile token.")
	}
	if mt.UserID != userID || mt.RevokedAt != nil || time.Now().UTC().After(mt.ExpiresAt) {
		return NewAppError(http.StatusUnauthorized, "TOKEN_REVOKED", "Mobile token is inactive or has been revoked.")
	}
	return nil
}

func (c *Controller) isEditableMobileAccount(ctx context.Context, userID string, acc *models.Account) bool {
	if !isRecordSupportedAccount(acc.Type) {
		return false
	}
	if acc.AdminID == userID {
		return true
	}
	access, err := c.repo.GetAccountAccess(ctx, acc.ID, userID)
	return err == nil && access != nil && checkAccessLevel(access.AccessLevel, models.AccessLevelEdit)
}

// ListMobileTargets returns editable checking/savings/credit-card accounts and categories without balances or history.
func (c *Controller) ListMobileTargets(ctx context.Context, userID string) (*models.MobileTargetsResponse, error) {
	accounts, err := c.repo.ListAccountsByUserID(ctx, userID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list accounts.")
	}
	cats, err := c.repo.ListCategoriesForUser(ctx, userID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list categories.")
	}

	accSummaries := make([]models.MobileAccountSummary, 0, len(accounts))
	for _, acc := range accounts {
		if c.isEditableMobileAccount(ctx, userID, acc) {
			accSummaries = append(accSummaries, models.MobileAccountSummary{
				ID:   acc.ID,
				Name: acc.Name,
				Type: acc.Type,
			})
		}
	}
	catSummaries := make([]models.MobileCategorySummary, 0, len(cats))
	for _, cat := range cats {
		catSummaries = append(catSummaries, models.MobileCategorySummary{
			ID:        cat.ID,
			Name:      cat.Name,
			IsDefault: cat.UserID == nil,
		})
	}
	return &models.MobileTargetsResponse{Accounts: accSummaries, Categories: catSummaries}, nil
}

func validateMobileRecordFields(req models.MobileRecordCreateRequest) (string, error) {
	if req.Value == nil {
		return "", NewAppError(http.StatusBadRequest, "INVALID_VALUE", "Transaction value is required.")
	}
	entity := strings.TrimSpace(req.Entity)
	if entity == "" {
		return "", NewAppError(http.StatusBadRequest, "INVALID_ENTITY", "entity cannot be empty.")
	}
	return entity, nil
}

// CreateMobileRecord adds a record to an editable checking, savings, or credit card account without returning balances.
func (c *Controller) CreateMobileRecord(ctx context.Context, userID, accountID string, req models.MobileRecordCreateRequest) (*models.MobileRecordCreateResponse, error) {
	acc, err := c.verifyAccountAccess(ctx, userID, accountID, models.AccessLevelEdit)
	if err != nil {
		return nil, err
	}
	if !isRecordSupportedAccount(acc.Type) {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_ACCOUNT_TYPE", "Account does not support mobile record entry.")
	}
	entity, err := validateMobileRecordFields(req)
	if err != nil {
		return nil, err
	}
	catID, err := c.resolveCategoryIDForUser(ctx, userID, req.CategoryID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	rec := &models.CheckingRecord{
		ID:         uuid.New().String(),
		AccountID:  accountID,
		CategoryID: catID,
		Value:      *req.Value,
		DateSent:   req.DateSent,
		Entity:     entity,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := c.repo.CreateCheckingRecord(ctx, rec); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create mobile record.")
	}
	trace.LogInfo(ctx, "Mobile widget created record %s for account %s", rec.ID, accountID)
	return &models.MobileRecordCreateResponse{ID: rec.ID, AccountID: rec.AccountID, CreatedAt: rec.CreatedAt}, nil
}
