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
	ErrSnapshotNotFound = NewAppError(http.StatusNotFound, "SNAPSHOT_NOT_FOUND", "Snapshot record not found.")
	ErrSnapshotConflict = NewAppError(http.StatusConflict, "SNAPSHOT_CONFLICT", "A snapshot for this account on this date already exists.")
)

func validateSnapshotDate(dateStr string) error {
	if strings.TrimSpace(dateStr) == "" {
		return NewAppError(http.StatusBadRequest, "MISSING_SNAPSHOT_DATE", "snapshot_date is required.")
	}
	if _, err := time.Parse("2006-01-02", dateStr); err != nil {
		return NewAppError(http.StatusBadRequest, "INVALID_SNAPSHOT_DATE", "snapshot_date must be formatted as YYYY-MM-DD.")
	}
	return nil
}

func (c *Controller) verifySnapshotAccount(ctx context.Context, userID, accountID string, required models.AccessLevel) (*models.Account, error) {
	acc, err := c.repo.GetAccountByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve account.")
	}
	if acc.Type != "snapshot" {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_ACCOUNT_TYPE", "Account is not a snapshot account.")
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

func toAccountSnapshotResponse(s *models.AccountSnapshot) *models.AccountSnapshotResponse {
	return &models.AccountSnapshotResponse{
		ID:           s.ID,
		AccountID:    s.AccountID,
		Value:        s.Value,
		SnapshotDate: s.SnapshotDate,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

// CreateAccountSnapshot records a new point-in-time account valuation.
func (c *Controller) CreateAccountSnapshot(ctx context.Context, userID, accountID string, req models.AccountSnapshotCreateRequest) (*models.AccountSnapshotResponse, error) {
	if _, err := c.verifySnapshotAccount(ctx, userID, accountID, models.AccessLevelEdit); err != nil {
		return nil, err
	}
	if req.Value == nil {
		return nil, NewAppError(http.StatusBadRequest, "MISSING_VALUE", "Snapshot value is required.")
	}
	if err := validateSnapshotDate(req.SnapshotDate); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	snap := &models.AccountSnapshot{
		ID:           uuid.New().String(),
		AccountID:    accountID,
		Value:        *req.Value,
		SnapshotDate: req.SnapshotDate,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := c.repo.CreateAccountSnapshot(ctx, snap); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrSnapshotConflict
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to record account snapshot.")
	}
	trace.LogInfo(ctx, "Snapshot for account %s recorded: %v on %s", accountID, *req.Value, req.SnapshotDate)
	return toAccountSnapshotResponse(snap), nil
}

// ListAccountSnapshots retrieves snapshots for trending the account over time.
func (c *Controller) ListAccountSnapshots(ctx context.Context, userID, accountID string, startDate, endDate *string) ([]*models.AccountSnapshotResponse, error) {
	if _, err := c.verifySnapshotAccount(ctx, userID, accountID, models.AccessLevelRead); err != nil {
		return nil, err
	}
	if startDate != nil && *startDate != "" {
		if err := validateSnapshotDate(*startDate); err != nil {
			return nil, err
		}
	}
	if endDate != nil && *endDate != "" {
		if err := validateSnapshotDate(*endDate); err != nil {
			return nil, err
		}
	}

	snapshots, err := c.repo.ListAccountSnapshots(ctx, accountID, startDate, endDate)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve snapshots.")
	}
	res := make([]*models.AccountSnapshotResponse, 0, len(snapshots))
	for _, s := range snapshots {
		res = append(res, toAccountSnapshotResponse(s))
	}
	return res, nil
}

// GetAccountSnapshot retrieves a specific snapshot record by ID.
func (c *Controller) GetAccountSnapshot(ctx context.Context, userID, accountID, snapshotID string) (*models.AccountSnapshotResponse, error) {
	if _, err := c.verifySnapshotAccount(ctx, userID, accountID, models.AccessLevelRead); err != nil {
		return nil, err
	}
	snap, err := c.repo.GetAccountSnapshotByID(ctx, accountID, snapshotID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrSnapshotNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve snapshot.")
	}
	return toAccountSnapshotResponse(snap), nil
}

// UpdateAccountSnapshot updates an existing account snapshot.
func (c *Controller) UpdateAccountSnapshot(ctx context.Context, userID, accountID, snapshotID string, req models.AccountSnapshotUpdateRequest) (*models.AccountSnapshotResponse, error) {
	if _, err := c.verifySnapshotAccount(ctx, userID, accountID, models.AccessLevelEdit); err != nil {
		return nil, err
	}
	if req.Value == nil {
		return nil, NewAppError(http.StatusBadRequest, "MISSING_VALUE", "Snapshot value is required.")
	}
	snap, err := c.repo.GetAccountSnapshotByID(ctx, accountID, snapshotID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrSnapshotNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve snapshot.")
	}

	if req.SnapshotDate != nil && *req.SnapshotDate != "" {
		if err := validateSnapshotDate(*req.SnapshotDate); err != nil {
			return nil, err
		}
		snap.SnapshotDate = *req.SnapshotDate
	}
	snap.Value = *req.Value
	snap.UpdatedAt = time.Now().UTC()

	if err := c.repo.UpdateAccountSnapshot(ctx, snap); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrSnapshotConflict
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update snapshot.")
	}
	trace.LogInfo(ctx, "Snapshot %s for account %s updated to %v", snapshotID, accountID, snap.Value)
	return toAccountSnapshotResponse(snap), nil
}

// DeleteAccountSnapshot removes a snapshot from a snapshot account.
func (c *Controller) DeleteAccountSnapshot(ctx context.Context, userID, accountID, snapshotID string) error {
	if _, err := c.verifySnapshotAccount(ctx, userID, accountID, models.AccessLevelEdit); err != nil {
		return err
	}
	if err := c.repo.DeleteAccountSnapshot(ctx, accountID, snapshotID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrSnapshotNotFound
		}
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete snapshot.")
	}
	trace.LogInfo(ctx, "Snapshot %s removed from account %s", snapshotID, accountID)
	return nil
}
