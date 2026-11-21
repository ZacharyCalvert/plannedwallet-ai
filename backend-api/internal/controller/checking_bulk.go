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

// CreateCheckingRecordsBulk validates and atomically creates up to 1,000 checking records.
func (c *Controller) CreateCheckingRecordsBulk(ctx context.Context, userID, accountID string, req models.CheckingRecordBulkCreateRequest) (*models.CheckingRecordBulkCreateResponse, error) {
	if err := c.verifyCheckingAccountWrite(ctx, userID, accountID); err != nil {
		return nil, err
	}
	if err := validateBulkRecordCount(len(req.Records)); err != nil {
		return nil, err
	}

	catSet, err := c.loadUserCategorySet(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	records, err := buildBulkCheckingRecords(accountID, req.Records, catSet, now)
	if err != nil {
		return nil, err
	}

	if err := c.repo.CreateCheckingRecordsBulk(ctx, records); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to bulk create checking records.")
	}

	trace.LogInfo(ctx, "Bulk created checking records", map[string]any{
		"account_id": accountID,
		"count":      len(records),
	})

	responses := make([]models.CheckingRecordResponse, len(records))
	for i, r := range records {
		responses[i] = toCheckingRecordResponse(r)
	}
	return &models.CheckingRecordBulkCreateResponse{
		CreatedCount: len(responses),
		Records:      responses,
	}, nil
}

func (c *Controller) verifyCheckingAccountWrite(ctx context.Context, userID, accountID string) error {
	account, err := c.verifyAccountAccess(ctx, userID, accountID, models.AccessLevelEdit)
	if err != nil {
		return err
	}
	if !isRecordSupportedAccount(account.Type) {
		return NewAppError(http.StatusBadRequest, "INVALID_ACCOUNT_TYPE", "Account does not support checking records.")
	}
	return nil
}

func validateBulkRecordCount(count int) error {
	if count == 0 {
		return NewAppError(http.StatusBadRequest, "INVALID_BULK_RECORDS", "records must contain at least 1 item.")
	}
	if count > models.MaxBulkCheckingRecords {
		return NewAppError(http.StatusBadRequest, "BULK_LIMIT_EXCEEDED", "Cannot upload more than 1000 records in a single bulk request.")
	}
	return nil
}

func (c *Controller) loadUserCategorySet(ctx context.Context, userID string) (map[string]struct{}, error) {
	cats, err := c.repo.ListCategoriesForUser(ctx, userID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to verify category.")
	}
	catSet := make(map[string]struct{}, len(cats))
	for _, cat := range cats {
		catSet[cat.ID] = struct{}{}
	}
	return catSet, nil
}

func buildBulkCheckingRecords(accountID string, items []models.CheckingRecordCreateRequest, catSet map[string]struct{}, now time.Time) ([]*models.CheckingRecord, error) {
	records := make([]*models.CheckingRecord, 0, len(items))
	for _, item := range items {
		rec, err := buildSingleBulkRecord(accountID, item, catSet, now)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, nil
}

func buildSingleBulkRecord(accountID string, item models.CheckingRecordCreateRequest, catSet map[string]struct{}, now time.Time) (*models.CheckingRecord, error) {
	if item.Value == nil {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_VALUE", "Transaction value is required.")
	}
	entity := strings.TrimSpace(item.Entity)
	if entity == "" {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_ENTITY", "entity cannot be empty.")
	}
	catID, err := resolveCategoryFromSet(item.CategoryID, catSet)
	if err != nil {
		return nil, err
	}
	return &models.CheckingRecord{
		ID:          uuid.New().String(),
		AccountID:   accountID,
		CategoryID:  catID,
		Value:       *item.Value,
		DateSent:    item.DateSent,
		DateCleared: item.DateCleared,
		Entity:      entity,
		Description: strings.TrimSpace(item.Description),
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func resolveCategoryFromSet(rawID *string, catSet map[string]struct{}) (*string, error) {
	if rawID == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*rawID)
	if trimmed == "" {
		return nil, nil
	}
	if _, ok := catSet[trimmed]; !ok {
		return nil, ErrCategoryNotFound
	}
	return &trimmed, nil
}
