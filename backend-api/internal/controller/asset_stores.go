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

func validateStoreLabel(label string) (string, error) {
	trimmed := strings.TrimSpace(label)
	if trimmed == "" || len(trimmed) > 100 {
		return "", NewAppError(http.StatusBadRequest, "INVALID_LABEL", "Store label must be between 1 and 100 characters.")
	}
	return trimmed, nil
}

func validateStoreRecordDate(dateStr string) (string, error) {
	trimmed := strings.TrimSpace(dateStr)
	if trimmed == "" {
		return time.Now().UTC().Format("2006-01-02"), nil
	}
	if _, err := time.Parse("2006-01-02", trimmed); err != nil {
		return "", NewAppError(http.StatusBadRequest, "INVALID_DATE_FORMAT", "record_date must be formatted as YYYY-MM-DD.")
	}
	return trimmed, nil
}

func (c *Controller) verifyStoreAccount(ctx context.Context, userID, accountID, storeID string, required models.AccessLevel) (*models.AssetStoreDetail, error) {
	if _, err := c.verifyAssetAccount(ctx, userID, accountID, required); err != nil {
		return nil, err
	}
	detail, err := c.repo.GetAssetStoreByID(ctx, accountID, storeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, NewAppError(http.StatusNotFound, "STORE_NOT_FOUND", "Asset store not found.")
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve asset store.")
	}
	return detail, nil
}

func (c *Controller) maybeCreateInitialRecord(ctx context.Context, storeID string, qty *float64, dateStr string, now time.Time) error {
	if qty == nil || *qty <= 0 {
		return nil
	}
	recDate, err := validateStoreRecordDate(dateStr)
	if err != nil {
		return err
	}
	rec := &models.AssetStoreRecord{
		ID:           uuid.New().String(),
		AssetStoreID: storeID,
		Quantity:     *qty,
		RecordDate:   recDate,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	return c.repo.CreateAssetStoreRecord(ctx, rec)
}

// CreateAssetStore handles creation of a new labeled storage location in an asset account.
func (c *Controller) CreateAssetStore(ctx context.Context, userID, accountID string, req models.CreateAssetStoreRequest) (*models.AssetStoreDetail, error) {
	if _, err := c.verifyAssetAccount(ctx, userID, accountID, models.AccessLevelEdit); err != nil {
		return nil, err
	}
	label, err := validateStoreLabel(req.Label)
	if err != nil {
		return nil, err
	}
	if _, err := c.getExistingAsset(ctx, req.AssetID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	store := &models.AssetStore{
		ID:          uuid.New().String(),
		AccountID:   accountID,
		AssetID:     req.AssetID,
		Label:       label,
		Description: strings.TrimSpace(req.Description),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := c.repo.CreateAssetStore(ctx, store); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, NewAppError(http.StatusConflict, "DUPLICATE_STORE_LABEL", "A store with this label already exists in this account.")
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create asset store.")
	}
	if err := c.maybeCreateInitialRecord(ctx, store.ID, req.Quantity, req.RecordDate, now); err != nil {
		return nil, err
	}
	trace.LogInfo(ctx, "Created asset store %s (%s) in account %s", store.Label, store.ID, accountID)
	return c.repo.GetAssetStoreByID(ctx, accountID, store.ID)
}

// GetAssetStore retrieves a specific asset store by ID.
func (c *Controller) GetAssetStore(ctx context.Context, userID, accountID, storeID string) (*models.AssetStoreDetail, error) {
	return c.verifyStoreAccount(ctx, userID, accountID, storeID, models.AccessLevelRead)
}

// ListAssetStores returns all asset stores in an account.
func (c *Controller) ListAssetStores(ctx context.Context, userID, accountID string) ([]*models.AssetStoreDetail, error) {
	if _, err := c.verifyAssetAccount(ctx, userID, accountID, models.AccessLevelRead); err != nil {
		return nil, err
	}
	stores, err := c.repo.ListAssetStores(ctx, accountID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list asset stores.")
	}
	return stores, nil
}

// UpdateAssetStore modifies an asset store's label, description, or asset assignment.
func (c *Controller) UpdateAssetStore(ctx context.Context, userID, accountID, storeID string, req models.UpdateAssetStoreRequest) (*models.AssetStoreDetail, error) {
	detail, err := c.verifyStoreAccount(ctx, userID, accountID, storeID, models.AccessLevelEdit)
	if err != nil {
		return nil, err
	}
	label, err := validateStoreLabel(req.Label)
	if err != nil {
		return nil, err
	}
	assetID := detail.AssetID
	if req.AssetID != nil && strings.TrimSpace(*req.AssetID) != "" {
		if _, err := c.getExistingAsset(ctx, *req.AssetID); err != nil {
			return nil, err
		}
		assetID = *req.AssetID
	}
	store := &models.AssetStore{
		ID:          storeID,
		AccountID:   accountID,
		AssetID:     assetID,
		Label:       label,
		Description: strings.TrimSpace(req.Description),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := c.repo.UpdateAssetStore(ctx, store); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, NewAppError(http.StatusConflict, "DUPLICATE_STORE_LABEL", "A store with this label already exists in this account.")
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update asset store.")
	}
	trace.LogInfo(ctx, "Updated asset store %s in account %s", storeID, accountID)
	return c.repo.GetAssetStoreByID(ctx, accountID, storeID)
}

// DeleteAssetStore removes an asset store and all its records.
func (c *Controller) DeleteAssetStore(ctx context.Context, userID, accountID, storeID string) error {
	if _, err := c.verifyStoreAccount(ctx, userID, accountID, storeID, models.AccessLevelEdit); err != nil {
		return err
	}
	if err := c.repo.DeleteAssetStore(ctx, accountID, storeID); err != nil {
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete asset store.")
	}
	trace.LogInfo(ctx, "Deleted asset store %s in account %s", storeID, accountID)
	return nil
}

// CreateAssetStoreRecord records a holding quantity on a specific date for a store.
func (c *Controller) CreateAssetStoreRecord(ctx context.Context, userID, accountID, storeID string, req models.CreateStoreRecordRequest) (*models.AssetStoreRecord, error) {
	if _, err := c.verifyStoreAccount(ctx, userID, accountID, storeID, models.AccessLevelEdit); err != nil {
		return nil, err
	}
	qty, err := validateQuantity(req.Quantity)
	if err != nil {
		return nil, err
	}
	recDate, err := validateStoreRecordDate(req.RecordDate)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	rec := &models.AssetStoreRecord{
		ID:           uuid.New().String(),
		AssetStoreID: storeID,
		Quantity:     qty,
		RecordDate:   recDate,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := c.repo.CreateAssetStoreRecord(ctx, rec); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, NewAppError(http.StatusConflict, "DUPLICATE_RECORD_DATE", "A quantity record already exists for this date.")
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create store record.")
	}
	trace.LogInfo(ctx, "Created quantity record on %s for store %s", rec.RecordDate, storeID)
	return rec, nil
}

// GetAssetStoreRecord retrieves a specific quantity record by ID.
func (c *Controller) GetAssetStoreRecord(ctx context.Context, userID, accountID, storeID, recordID string) (*models.AssetStoreRecord, error) {
	if _, err := c.verifyStoreAccount(ctx, userID, accountID, storeID, models.AccessLevelRead); err != nil {
		return nil, err
	}
	rec, err := c.repo.GetAssetStoreRecordByID(ctx, storeID, recordID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, NewAppError(http.StatusNotFound, "RECORD_NOT_FOUND", "Store record not found.")
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve store record.")
	}
	return rec, nil
}

// ListAssetStoreRecords returns all date-keyed quantity records for a store.
func (c *Controller) ListAssetStoreRecords(ctx context.Context, userID, accountID, storeID string) ([]*models.AssetStoreRecord, error) {
	if _, err := c.verifyStoreAccount(ctx, userID, accountID, storeID, models.AccessLevelRead); err != nil {
		return nil, err
	}
	recs, err := c.repo.ListAssetStoreRecords(ctx, storeID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list store records.")
	}
	return recs, nil
}

// UpdateAssetStoreRecord updates quantity or date of an existing record.
func (c *Controller) UpdateAssetStoreRecord(ctx context.Context, userID, accountID, storeID, recordID string, req models.UpdateStoreRecordRequest) (*models.AssetStoreRecord, error) {
	if _, err := c.verifyStoreAccount(ctx, userID, accountID, storeID, models.AccessLevelEdit); err != nil {
		return nil, err
	}
	rec, err := c.repo.GetAssetStoreRecordByID(ctx, storeID, recordID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, NewAppError(http.StatusNotFound, "RECORD_NOT_FOUND", "Store record not found.")
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve store record.")
	}
	if req.Quantity != nil {
		qty, err := validateQuantity(req.Quantity)
		if err != nil {
			return nil, err
		}
		rec.Quantity = qty
	}
	if req.RecordDate != nil && strings.TrimSpace(*req.RecordDate) != "" {
		dateStr, err := validateStoreRecordDate(*req.RecordDate)
		if err != nil {
			return nil, err
		}
		rec.RecordDate = dateStr
	}
	rec.UpdatedAt = time.Now().UTC()
	if err := c.repo.UpdateAssetStoreRecord(ctx, rec); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, NewAppError(http.StatusConflict, "DUPLICATE_RECORD_DATE", "A quantity record already exists for this date.")
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update store record.")
	}
	trace.LogInfo(ctx, "Updated quantity record %s for store %s", recordID, storeID)
	return rec, nil
}

// DeleteAssetStoreRecord removes a single date-keyed quantity record.
func (c *Controller) DeleteAssetStoreRecord(ctx context.Context, userID, accountID, storeID, recordID string) error {
	if _, err := c.verifyStoreAccount(ctx, userID, accountID, storeID, models.AccessLevelEdit); err != nil {
		return err
	}
	if err := c.repo.DeleteAssetStoreRecord(ctx, storeID, recordID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return NewAppError(http.StatusNotFound, "RECORD_NOT_FOUND", "Store record not found.")
		}
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete store record.")
	}
	trace.LogInfo(ctx, "Deleted quantity record %s for store %s", recordID, storeID)
	return nil
}

// GetAssetStoreTrend retrieves the time-series trend of valuations for a specific store.
func (c *Controller) GetAssetStoreTrend(ctx context.Context, userID, accountID, storeID string) ([]models.AssetStoreTrendPoint, error) {
	if _, err := c.verifyStoreAccount(ctx, userID, accountID, storeID, models.AccessLevelRead); err != nil {
		return nil, err
	}
	trend, err := c.repo.GetAssetStoreTrend(ctx, storeID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to calculate store trend.")
	}
	return trend, nil
}
