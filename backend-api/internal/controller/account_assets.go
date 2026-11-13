package controller

import (
	"context"
	"errors"
	"net/http"
	"sort"
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

func validateAssetDate(dateStr string) (string, error) {
	trimmed := strings.TrimSpace(dateStr)
	if trimmed == "" {
		return time.Now().UTC().Format("2006-01-02"), nil
	}
	if _, err := time.Parse("2006-01-02", trimmed); err != nil {
		return "", NewAppError(http.StatusBadRequest, "INVALID_DATE_FORMAT", "asset_date must be formatted as YYYY-MM-DD.")
	}
	return trimmed, nil
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
		AssetDate:       d.AssetDate,
		LatestValue:     d.LatestValue,
		LatestValueDate: d.LatestValueDate,
		TotalValue:      d.Quantity * d.LatestValue,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
	}
}

func (c *Controller) getExistingAsset(ctx context.Context, assetID string) (*models.Asset, error) {
	aid := strings.TrimSpace(assetID)
	if aid == "" {
		return nil, NewAppError(http.StatusBadRequest, "MISSING_ASSET_ID", "Asset ID is required.")
	}
	asset, err := c.repo.GetAssetByID(ctx, aid)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrAssetNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve asset.")
	}
	return asset, nil
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
	dateStr, err := validateAssetDate(req.AssetDate)
	if err != nil {
		return nil, err
	}
	asset, err := c.getExistingAsset(ctx, req.AssetID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	aa := &models.AccountAsset{
		ID:        uuid.New().String(),
		AccountID: accountID,
		AssetID:   asset.ID,
		Quantity:  qty,
		AssetDate: dateStr,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := c.repo.AddAccountAsset(ctx, aa); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, NewAppError(http.StatusConflict, "ACCOUNT_ASSET_CONFLICT", "Asset holding on this date already exists.")
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to add asset to account.")
	}
	trace.LogInfo(ctx, "Asset %s added to account %s with quantity %v on %s", asset.Label, accountID, qty, dateStr)
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

func applyAssetHoldingUpdate(existing *models.AccountAsset, req models.AccountAssetUpdateRequest) error {
	if req.Quantity != nil {
		qty, err := validateQuantity(req.Quantity)
		if err != nil {
			return err
		}
		existing.Quantity = qty
	}
	if req.AssetDate != "" {
		dateStr, err := validateAssetDate(req.AssetDate)
		if err != nil {
			return err
		}
		existing.AssetDate = dateStr
	}
	existing.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateAccountAsset updates the held quantity of an asset in an account.
func (c *Controller) UpdateAccountAsset(ctx context.Context, userID, accountID, identifier string, req models.AccountAssetUpdateRequest) (*models.AccountAssetResponse, error) {
	if _, err := c.verifyAssetAccount(ctx, userID, accountID, models.AccessLevelEdit); err != nil {
		return nil, err
	}
	existing, err := c.repo.GetAccountAsset(ctx, accountID, identifier)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, NewAppError(http.StatusNotFound, "ACCOUNT_ASSET_NOT_FOUND", "Asset not found in account.")
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve account asset.")
	}
	if err := applyAssetHoldingUpdate(existing, req); err != nil {
		return nil, err
	}
	if err := c.repo.UpdateAccountAsset(ctx, existing); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, NewAppError(http.StatusConflict, "ACCOUNT_ASSET_CONFLICT", "Asset holding on this date already exists.")
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update account asset.")
	}
	trace.LogInfo(ctx, "Account asset %s updated in account %s", identifier, accountID)
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

// GetAssetAccountSummary computes total current worth and summarizes distinct held assets.
func (c *Controller) GetAssetAccountSummary(ctx context.Context, userID, accountID string) (*models.AssetAccountSummaryResponse, error) {
	assets, err := c.ListAccountAssets(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var latestAssets []*models.AccountAssetResponse
	var total float64
	for _, a := range assets {
		if !seen[a.AssetID] {
			seen[a.AssetID] = true
			latestAssets = append(latestAssets, a)
			total += a.TotalValue
		}
	}
	return &models.AssetAccountSummaryResponse{
		AccountID:  accountID,
		TotalValue: total,
		Assets:     latestAssets,
	}, nil
}

func collectTrendDates(details []*models.AccountAssetDetail, allValues map[string][]*models.AssetValue, start, end string) []string {
	dateSet := make(map[string]bool)
	for _, d := range details {
		if (start == "" || d.AssetDate >= start) && (end == "" || d.AssetDate <= end) {
			dateSet[d.AssetDate] = true
		}
	}
	for _, vals := range allValues {
		for _, v := range vals {
			if (start == "" || v.ValueDate >= start) && (end == "" || v.ValueDate <= end) {
				dateSet[v.ValueDate] = true
			}
		}
	}
	var dates []string
	for d := range dateSet {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	return dates
}

func getHoldingQuantityAtDate(details []*models.AccountAssetDetail, assetID, dateStr string) float64 {
	var qty float64
	var bestDate string
	var bestCreated time.Time
	for _, d := range details {
		if d.AssetID != assetID || d.AssetDate > dateStr {
			continue
		}
		if d.AssetDate > bestDate || (d.AssetDate == bestDate && d.CreatedAt.After(bestCreated)) {
			bestDate = d.AssetDate
			bestCreated = d.CreatedAt
			qty = d.Quantity
		}
	}
	return qty
}

func getAssetPriceAtDate(values []*models.AssetValue, dateStr string) float64 {
	var price float64
	var bestDate string
	var bestCreated time.Time
	for _, v := range values {
		if v.ValueDate > dateStr {
			continue
		}
		if v.ValueDate > bestDate || (v.ValueDate == bestDate && v.CreatedAt.After(bestCreated)) {
			bestDate = v.ValueDate
			bestCreated = v.CreatedAt
			price = v.Value
		}
	}
	if bestDate == "" && len(values) > 0 {
		return values[0].Value
	}
	return price
}

func computeAccountValueAtDate(assetIDs []string, details []*models.AccountAssetDetail, allValues map[string][]*models.AssetValue, dateStr string) float64 {
	var total float64
	for _, aid := range assetIDs {
		qty := getHoldingQuantityAtDate(details, aid, dateStr)
		if qty <= 0 {
			continue
		}
		price := getAssetPriceAtDate(allValues[aid], dateStr)
		total += qty * price
	}
	return total
}

func getDistinctAssetIDs(details []*models.AccountAssetDetail) []string {
	seen := make(map[string]bool)
	var ids []string
	for _, d := range details {
		if !seen[d.AssetID] {
			seen[d.AssetID] = true
			ids = append(ids, d.AssetID)
		}
	}
	return ids
}

func (c *Controller) loadAssetsValuationMap(ctx context.Context, assetIDs []string) (map[string][]*models.AssetValue, error) {
	valMap := make(map[string][]*models.AssetValue, len(assetIDs))
	for _, aid := range assetIDs {
		vals, err := c.repo.ListAssetValues(ctx, aid, nil, nil)
		if err != nil {
			return nil, err
		}
		valMap[aid] = vals
	}
	return valMap, nil
}

// GetAssetAccountTrend calculates total account worth over time.
func (c *Controller) GetAssetAccountTrend(ctx context.Context, userID, accountID string, startDate, endDate *string) ([]*models.AssetAccountTrendPoint, error) {
	if _, err := c.verifyAssetAccount(ctx, userID, accountID, models.AccessLevelRead); err != nil {
		return nil, err
	}
	details, err := c.repo.ListAccountAssets(ctx, accountID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list account assets.")
	}
	assetIDs := getDistinctAssetIDs(details)
	valMap, err := c.loadAssetsValuationMap(ctx, assetIDs)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve asset valuations.")
	}
	var start, end string
	if startDate != nil {
		start = *startDate
	}
	if endDate != nil {
		end = *endDate
	}
	dates := collectTrendDates(details, valMap, start, end)
	points := make([]*models.AssetAccountTrendPoint, 0, len(dates))
	for _, d := range dates {
		val := computeAccountValueAtDate(assetIDs, details, valMap, d)
		points = append(points, &models.AssetAccountTrendPoint{Date: d, TotalValue: val})
	}
	return points, nil
}
