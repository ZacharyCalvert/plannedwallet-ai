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

func validateQuantity(q *float64) (float64, error) {
	if q == nil || *q <= 0 {
		return 0, NewAppError(http.StatusBadRequest, "INVALID_QUANTITY", "Asset quantity must be greater than zero.")
	}
	return *q, nil
}

func checkAccessLevel(accessLevel, required models.AccessLevel) bool {
	if accessLevel == models.AccessLevelAdmin {
		return true
	}
	if required == models.AccessLevelEdit {
		return accessLevel == models.AccessLevelEdit
	}
	return required == models.AccessLevelRead
}

func (c *Controller) verifyAssetAccount(ctx context.Context, userID, accountID string, required models.AccessLevel) (*models.Account, error) {
	acc, err := c.repo.GetAccountByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve account.")
	}
	if acc.Type != "asset" && acc.Type != "assets" {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_ACCOUNT_TYPE", "Account is not an asset account.")
	}
	if acc.AdminID == userID {
		return acc, nil
	}
	access, err := c.repo.GetAccountAccess(ctx, accountID, userID)
	if err != nil || access == nil {
		return nil, ErrAccountNotFound
	}
	if !checkAccessLevel(access.AccessLevel, required) {
		return nil, NewAppError(http.StatusForbidden, "FORBIDDEN", "Insufficient permissions for account.")
	}
	return acc, nil
}

func toAccountAssetResponse(d *models.AccountAssetDetail) *models.AccountAssetResponse {
	return &models.AccountAssetResponse{
		ID:              d.ID,
		AccountID:       d.AccountID,
		AssetID:         d.AssetID,
		Label:           d.Label,
		Description:     d.Description,
		Quantity:        d.Quantity,
		LatestValue:     d.LatestValue,
		LatestValueDate: d.LatestValueDate,
		TotalValue:      d.Quantity * d.LatestValue,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
	}
}

// AddAccountAsset associates an asset and its quantity with an asset account.
func (c *Controller) AddAccountAsset(ctx context.Context, userID, accountID string, req models.AccountAssetAddRequest) (*models.AccountAssetResponse, error) {
	if _, err := c.verifyAssetAccount(ctx, userID, accountID, models.AccessLevelEdit); err != nil {
		return nil, err
	}
	qty, err := validateQuantity(req.Quantity)
	if err != nil {
		return nil, err
	}
	assetID := strings.TrimSpace(req.AssetID)
	if assetID == "" {
		return nil, NewAppError(http.StatusBadRequest, "MISSING_ASSET_ID", "Asset ID is required.")
	}
	asset, err := c.repo.GetAssetByID(ctx, assetID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrAssetNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve asset.")
	}

	now := time.Now().UTC()
	aa := &models.AccountAsset{
		ID:        uuid.New().String(),
		AccountID: accountID,
		AssetID:   asset.ID,
		Quantity:  qty,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := c.repo.AddAccountAsset(ctx, aa); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, NewAppError(http.StatusConflict, "ACCOUNT_ASSET_CONFLICT", "Asset already exists in this account.")
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to add asset to account.")
	}
	trace.LogInfo(ctx, "Asset %s added to account %s with quantity %v", asset.Label, accountID, qty)
	return c.GetAccountAsset(ctx, userID, accountID, aa.ID)
}

// ListAccountAssets returns all assets in an asset account with calculated values.
func (c *Controller) ListAccountAssets(ctx context.Context, userID, accountID string) ([]*models.AccountAssetResponse, error) {
	if _, err := c.verifyAssetAccount(ctx, userID, accountID, models.AccessLevelRead); err != nil {
		return nil, err
	}
	details, err := c.repo.ListAccountAssets(ctx, accountID)
	if err != nil {
		trace.LogError(ctx, err, "Failed to list assets for account %s", accountID)
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list account assets.")
	}
	res := make([]*models.AccountAssetResponse, 0, len(details))
	for _, d := range details {
		res = append(res, toAccountAssetResponse(d))
	}
	return res, nil
}

// GetAccountAsset retrieves a specific asset in an account by item ID or Asset ID.
func (c *Controller) GetAccountAsset(ctx context.Context, userID, accountID, identifier string) (*models.AccountAssetResponse, error) {
	items, err := c.ListAccountAssets(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.ID == identifier || item.AssetID == identifier {
			return item, nil
		}
	}
	return nil, NewAppError(http.StatusNotFound, "ACCOUNT_ASSET_NOT_FOUND", "Asset not found in account.")
}

// UpdateAccountAsset updates the held quantity of an asset in an account.
func (c *Controller) UpdateAccountAsset(ctx context.Context, userID, accountID, identifier string, req models.AccountAssetUpdateRequest) (*models.AccountAssetResponse, error) {
	if _, err := c.verifyAssetAccount(ctx, userID, accountID, models.AccessLevelEdit); err != nil {
		return nil, err
	}
	qty, err := validateQuantity(req.Quantity)
	if err != nil {
		return nil, err
	}
	existing, err := c.repo.GetAccountAsset(ctx, accountID, identifier)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, NewAppError(http.StatusNotFound, "ACCOUNT_ASSET_NOT_FOUND", "Asset not found in account.")
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve account asset.")
	}
	existing.Quantity = qty
	existing.UpdatedAt = time.Now().UTC()
	if err := c.repo.UpdateAccountAsset(ctx, existing); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update account asset.")
	}
	trace.LogInfo(ctx, "Account asset %s updated in account %s to quantity %v", identifier, accountID, qty)
	return c.GetAccountAsset(ctx, userID, accountID, existing.ID)
}

// DeleteAccountAsset removes an asset holding from an account.
func (c *Controller) DeleteAccountAsset(ctx context.Context, userID, accountID, identifier string) error {
	if _, err := c.verifyAssetAccount(ctx, userID, accountID, models.AccessLevelEdit); err != nil {
		return err
	}
	if err := c.repo.DeleteAccountAsset(ctx, accountID, identifier); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return NewAppError(http.StatusNotFound, "ACCOUNT_ASSET_NOT_FOUND", "Asset not found in account.")
		}
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete account asset.")
	}
	trace.LogInfo(ctx, "Account asset %s removed from account %s", identifier, accountID)
	return nil
}

// GetAssetAccountSummary computes total current worth and summarizes all assets.
func (c *Controller) GetAssetAccountSummary(ctx context.Context, userID, accountID string) (*models.AssetAccountSummaryResponse, error) {
	assets, err := c.ListAccountAssets(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}
	var total float64
	for _, a := range assets {
		total += a.TotalValue
	}
	return &models.AssetAccountSummaryResponse{
		AccountID:  accountID,
		TotalValue: total,
		Assets:     assets,
	}, nil
}
