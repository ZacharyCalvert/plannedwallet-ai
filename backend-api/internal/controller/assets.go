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

var (
	ErrAssetNotFound        = NewAppError(http.StatusNotFound, "ASSET_NOT_FOUND", "Asset not found.")
	ErrAssetConflict        = NewAppError(http.StatusConflict, "ASSET_CONFLICT", "An asset with this label already exists.")
	ErrAssetValueNotFound   = NewAppError(http.StatusNotFound, "ASSET_VALUE_NOT_FOUND", "Asset value record not found.")
	ErrAssetValueConflict   = NewAppError(http.StatusConflict, "ASSET_VALUE_CONFLICT", "A value for this asset on this date already exists.")
	ErrSystemAssetImmutable = NewAppError(http.StatusForbidden, "SYSTEM_ASSET_IMMUTABLE", "System-defined assets cannot be modified or deleted.")
	ErrAssetForbidden       = NewAppError(http.StatusForbidden, "FORBIDDEN", "You do not have permission to modify or delete this asset.")
)

func validateAssetRequest(label string) error {
	if strings.TrimSpace(label) == "" {
		return NewAppError(http.StatusBadRequest, "MISSING_LABEL", "Asset label cannot be empty.")
	}
	return nil
}

func validateAssetValueRequest(value *float64, valueDate string) error {
	if value == nil {
		return NewAppError(http.StatusBadRequest, "MISSING_VALUE", "Asset value is required.")
	}
	if valueDate == "" {
		return NewAppError(http.StatusBadRequest, "MISSING_VALUE_DATE", "Asset value date is required.")
	}
	if _, err := time.Parse("2006-01-02", valueDate); err != nil {
		return NewAppError(http.StatusBadRequest, "INVALID_VALUE_DATE", "Invalid date format, expected YYYY-MM-DD.")
	}
	return nil
}

func verifyAssetOwnership(asset *models.Asset, userID string) error {
	if asset.UserID == nil {
		return ErrSystemAssetImmutable
	}
	if *asset.UserID != userID {
		return ErrAssetForbidden
	}
	return nil
}

func isAssetAccessible(asset *models.Asset, userID string) bool {
	if asset.UserID == nil {
		return true
	}
	return *asset.UserID == userID
}

func (c *Controller) checkAssetEditable(ctx context.Context, userID, assetID string) error {
	asset, err := c.repo.GetAssetByID(ctx, assetID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrAssetNotFound
		}
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve asset.")
	}
	return verifyAssetOwnership(asset, userID)
}

func toAssetResponse(a *models.Asset) *models.AssetResponse {
	return &models.AssetResponse{
		ID:          a.ID,
		UserID:      a.UserID,
		Label:       a.Label,
		Description: a.Description,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}

func toAssetValueResponse(av *models.AssetValue) *models.AssetValueResponse {
	return &models.AssetValueResponse{
		ID:        av.ID,
		AssetID:   av.AssetID,
		Value:     av.Value,
		ValueDate: av.ValueDate,
		CreatedAt: av.CreatedAt,
		UpdatedAt: av.UpdatedAt,
	}
}

// CreateAsset registers a new asset definition.
func (c *Controller) CreateAsset(ctx context.Context, userID string, req models.AssetCreateRequest) (*models.AssetResponse, error) {
	label := strings.TrimSpace(req.Label)
	if err := validateAssetRequest(label); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	asset := &models.Asset{
		ID:          uuid.New().String(),
		UserID:      &userID,
		Label:       label,
		Description: strings.TrimSpace(req.Description),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := c.repo.CreateAsset(ctx, asset); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrAssetConflict
		}
		trace.LogError(ctx, err, "Failed to create asset %s", label)
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create asset.")
	}

	trace.LogInfo(ctx, "Asset created: %s (id: %s)", asset.Label, asset.ID)
	return toAssetResponse(asset), nil
}

// ListAssets retrieves visible assets for the user.
func (c *Controller) ListAssets(ctx context.Context, userID string, label *string) ([]models.AssetResponse, error) {
	records, err := c.repo.ListAssets(ctx, &userID, label)
	if err != nil {
		trace.LogError(ctx, err, "Failed to list assets for user %s", userID)
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve assets.")
	}

	res := make([]models.AssetResponse, 0, len(records))
	for _, r := range records {
		res = append(res, *toAssetResponse(r))
	}
	return res, nil
}

// GetAsset retrieves an asset by ID, checking accessibility.
func (c *Controller) GetAsset(ctx context.Context, userID, id string) (*models.AssetResponse, error) {
	asset, err := c.repo.GetAssetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrAssetNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve asset.")
	}

	if !isAssetAccessible(asset, userID) {
		return nil, ErrAssetNotFound
	}
	return toAssetResponse(asset), nil
}

// UpdateAsset updates an owned asset's label and description.
func (c *Controller) UpdateAsset(ctx context.Context, userID, id string, req models.AssetUpdateRequest) (*models.AssetResponse, error) {
	label := strings.TrimSpace(req.Label)
	if err := validateAssetRequest(label); err != nil {
		return nil, err
	}

	asset, err := c.repo.GetAssetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrAssetNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve asset.")
	}

	if err := verifyAssetOwnership(asset, userID); err != nil {
		return nil, err
	}

	asset.Label = label
	asset.Description = strings.TrimSpace(req.Description)
	asset.UpdatedAt = time.Now().UTC()

	if err := c.repo.UpdateAsset(ctx, asset); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrAssetConflict
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update asset.")
	}
	return toAssetResponse(asset), nil
}

// DeleteAsset removes a user-owned asset and its values.
func (c *Controller) DeleteAsset(ctx context.Context, userID, id string) error {
	asset, err := c.repo.GetAssetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrAssetNotFound
		}
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve asset.")
	}

	if err := verifyAssetOwnership(asset, userID); err != nil {
		return err
	}

	if err := c.repo.DeleteAsset(ctx, id); err != nil {
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete asset.")
	}
	return nil
}

// CreateAssetValue records a valuation for a specific asset on a date.
func (c *Controller) CreateAssetValue(ctx context.Context, userID, assetID string, req models.AssetValueCreateRequest) (*models.AssetValueResponse, error) {
	valueDate := strings.TrimSpace(req.ValueDate)
	if err := validateAssetValueRequest(req.Value, valueDate); err != nil {
		return nil, err
	}
	if err := c.checkAssetEditable(ctx, userID, assetID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	av := &models.AssetValue{
		ID:        uuid.New().String(),
		AssetID:   assetID,
		Value:     *req.Value,
		ValueDate: valueDate,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := c.repo.CreateAssetValue(ctx, av); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrAssetValueConflict
		}
		trace.LogError(ctx, err, "Failed to create asset value for %s", assetID)
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create asset value.")
	}

	trace.LogInfo(ctx, "Asset value created for asset %s on %s = %v (id: %s)", av.AssetID, av.ValueDate, av.Value, av.ID)
	return toAssetValueResponse(av), nil
}

// ListAssetValues retrieves recorded valuations for an asset.
func (c *Controller) ListAssetValues(ctx context.Context, userID, assetID string, startDate, endDate *string) ([]models.AssetValueResponse, error) {
	asset, err := c.repo.GetAssetByID(ctx, assetID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrAssetNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve asset.")
	}

	if !isAssetAccessible(asset, userID) {
		return nil, ErrAssetNotFound
	}

	records, err := c.repo.ListAssetValues(ctx, assetID, startDate, endDate)
	if err != nil {
		trace.LogError(ctx, err, "Failed to list asset values for asset %s", assetID)
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve asset values.")
	}

	res := make([]models.AssetValueResponse, 0, len(records))
	for _, r := range records {
		res = append(res, *toAssetValueResponse(r))
	}
	return res, nil
}

// GetAssetValue fetches a single valuation for an asset.
func (c *Controller) GetAssetValue(ctx context.Context, userID, assetID, valueID string) (*models.AssetValueResponse, error) {
	asset, err := c.repo.GetAssetByID(ctx, assetID)
	if err != nil || !isAssetAccessible(asset, userID) {
		return nil, ErrAssetNotFound
	}

	av, err := c.repo.GetAssetValueByID(ctx, assetID, valueID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrAssetValueNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve asset value.")
	}

	return toAssetValueResponse(av), nil
}

// UpdateAssetValue modifies an existing valuation for an owned asset.
func (c *Controller) UpdateAssetValue(ctx context.Context, userID, assetID, valueID string, req models.AssetValueUpdateRequest) (*models.AssetValueResponse, error) {
	valueDate := strings.TrimSpace(req.ValueDate)
	if err := validateAssetValueRequest(req.Value, valueDate); err != nil {
		return nil, err
	}
	if err := c.checkAssetEditable(ctx, userID, assetID); err != nil {
		return nil, err
	}

	av, err := c.repo.GetAssetValueByID(ctx, assetID, valueID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrAssetValueNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve asset value.")
	}

	av.Value = *req.Value
	av.ValueDate = valueDate
	av.UpdatedAt = time.Now().UTC()

	if err := c.repo.UpdateAssetValue(ctx, av); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrAssetValueConflict
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update asset value.")
	}
	return toAssetValueResponse(av), nil
}

// DeleteAssetValue removes an existing valuation for an owned asset.
func (c *Controller) DeleteAssetValue(ctx context.Context, userID, assetID, valueID string) error {
	if err := c.checkAssetEditable(ctx, userID, assetID); err != nil {
		return err
	}
	if err := c.repo.DeleteAssetValue(ctx, assetID, valueID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrAssetValueNotFound
		}
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete asset value.")
	}
	return nil
}
