package controller

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/repository"
	"github.com/plannedwallet/internal/trace"
)

var (
	ErrRecordReconciled        = NewAppError(http.StatusConflict, "RECORD_RECONCILED", "Reconciled transactions cannot modify amount, sent date, cleared date, or recipient.")
	ErrRecordAlreadyReconciled = NewAppError(http.StatusConflict, "RECORD_ALREADY_RECONCILED", "One or more records are already included in a reconcile report.")
	ErrReconcileReportNotFound = NewAppError(http.StatusNotFound, "RECONCILE_REPORT_NOT_FOUND", "Reconcile report not found.")
)

func normalizedDatePtr(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func ensureReconciledUpdateAllowed(existing *models.CheckingRecord, val float64, sent string, cleared *string, entity string) error {
	if existing.ReconcileReportID == nil {
		return nil
	}
	if val != existing.Value || sent != existing.DateSent || normalizedDatePtr(cleared) != normalizedDatePtr(existing.DateCleared) || entity != existing.Entity {
		return ErrRecordReconciled
	}
	return nil
}

func ensureReconciledPatchAllowed(existing *models.CheckingRecord, req models.CheckingRecordPatchRequest) error {
	if existing.ReconcileReportID == nil {
		return nil
	}
	if req.Value != nil || req.DateSent != nil || req.DateCleared != nil || req.Entity != nil {
		return ErrRecordReconciled
	}
	return nil
}

func (c *Controller) verifyReconcileAccount(ctx context.Context, userID, accountID string, required models.AccessLevel) (*models.Account, error) {
	acc, err := c.repo.GetAccountByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to verify account.")
	}
	if !c.userHasAccountAccess(ctx, acc, userID) {
		return nil, ErrAccountNotFound
	}
	if acc.AdminID != userID {
		access, err := c.repo.GetAccountAccess(ctx, accountID, userID)
		if err != nil || access == nil || !checkAccessLevel(access.AccessLevel, required) {
			return nil, NewAppError(http.StatusForbidden, "FORBIDDEN", "Insufficient permissions for account.")
		}
	}
	if !isRecordSupportedAccount(acc.Type) {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_ACCOUNT_TYPE", "Account does not support reconciliation.")
	}
	return acc, nil
}

func validateReconcileRequest(req models.ReconcileReportCreateRequest) error {
	if strings.TrimSpace(req.ReportDate) == "" {
		return NewAppError(http.StatusBadRequest, "INVALID_REPORT_DATE", "report_date is required.")
	}
	if _, err := time.Parse("2006-01-02", req.ReportDate); err != nil {
		return NewAppError(http.StatusBadRequest, "INVALID_REPORT_DATE", "report_date must be formatted as YYYY-MM-DD.")
	}
	if req.StartingBalance == nil || req.EndingBalance == nil {
		return NewAppError(http.StatusBadRequest, "INVALID_BALANCE", "starting_balance and ending_balance are required.")
	}
	seen := make(map[string]bool, len(req.RecordIDs))
	for _, id := range req.RecordIDs {
		if strings.TrimSpace(id) == "" || seen[id] {
			return NewAppError(http.StatusBadRequest, "INVALID_RECORD_IDS", "record_ids must contain unique non-empty record UUIDs.")
		}
		seen[id] = true
	}
	return nil
}

func (c *Controller) loadAndValidateReconcileRecords(ctx context.Context, accountID string, recordIDs []string) ([]*models.CheckingRecord, float64, error) {
	records := make([]*models.CheckingRecord, 0, len(recordIDs))
	sum := 0.0
	for _, recID := range recordIDs {
		rec, err := c.repo.GetCheckingRecordByID(ctx, accountID, recID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, 0, ErrRecordNotFound
			}
			return nil, 0, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve checking record.")
		}
		if rec.ReconcileReportID != nil && *rec.ReconcileReportID != "" {
			return nil, 0, ErrRecordAlreadyReconciled
		}
		sum += rec.Value
		records = append(records, rec)
	}
	return records, sum, nil
}

func toReconcileReportResponse(rep *models.ReconcileReport, records []*models.CheckingRecord) models.ReconcileReportResponse {
	recordIDs := make([]string, 0, len(records))
	respRecords := make([]models.CheckingRecordResponse, 0, len(records))
	for _, r := range records {
		recordIDs = append(recordIDs, r.ID)
		respRecords = append(respRecords, toCheckingRecordResponse(r))
	}
	return models.ReconcileReportResponse{
		ID:              rep.ID,
		AccountID:       rep.AccountID,
		Name:            rep.Name,
		ReportDate:      rep.ReportDate,
		StartingBalance: rep.StartingBalance,
		EndingBalance:   rep.EndingBalance,
		RecordCount:     len(respRecords),
		RecordIDs:       recordIDs,
		Records:         respRecords,
		CreatedAt:       rep.CreatedAt,
		UpdatedAt:       rep.UpdatedAt,
	}
}

func (c *Controller) persistReconcileReport(ctx context.Context, userID, accountID string, req models.ReconcileReportCreateRequest) (*models.ReconcileReport, error) {
	now := time.Now().UTC()
	report := &models.ReconcileReport{
		ID:              uuid.New().String(),
		AccountID:       accountID,
		Name:            strings.TrimSpace(req.Name),
		ReportDate:      req.ReportDate,
		StartingBalance: *req.StartingBalance,
		EndingBalance:   *req.EndingBalance,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := c.repo.CreateReconcileReport(ctx, report, req.RecordIDs); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrRecordAlreadyReconciled
		}
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrRecordNotFound
		}
		trace.LogError(ctx, err, "Failed to create reconcile report for account %s", accountID)
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create reconcile report.")
	}
	trace.LogInfo(ctx, "User %s created reconcile report %s (%s) with %d records", userID, report.ID, report.ReportDate, len(req.RecordIDs))
	return report, nil
}

// CreateReconcileReport creates a reconcile report and marks attached records as reconciled.
func (c *Controller) CreateReconcileReport(ctx context.Context, userID, accountID string, req models.ReconcileReportCreateRequest) (*models.ReconcileReportResponse, error) {
	if _, err := c.verifyReconcileAccount(ctx, userID, accountID, models.AccessLevelEdit); err != nil {
		return nil, err
	}
	if err := validateReconcileRequest(req); err != nil {
		return nil, err
	}
	records, sum, err := c.loadAndValidateReconcileRecords(ctx, accountID, req.RecordIDs)
	if err != nil {
		return nil, err
	}
	calculatedEnding := *req.StartingBalance + sum
	if math.Abs(calculatedEnding-*req.EndingBalance) > 1e-6 {
		return nil, NewAppError(
			http.StatusBadRequest,
			"RECONCILE_BALANCE_MISMATCH",
			fmt.Sprintf("Starting balance plus selected records (%.2f) does not equal ending balance (%.2f).", calculatedEnding, *req.EndingBalance),
		)
	}
	report, err := c.persistReconcileReport(ctx, userID, accountID, req)
	if err != nil {
		return nil, err
	}
	_, attached, getErr := c.repo.GetReconcileReportByID(ctx, accountID, report.ID)
	if getErr != nil {
		attached = records
	}
	res := toReconcileReportResponse(report, attached)
	return &res, nil
}

// ListReconcileReports lists all completed reconcile reports for an account.
func (c *Controller) ListReconcileReports(ctx context.Context, userID, accountID string) ([]models.ReconcileReportResponse, error) {
	if _, err := c.verifyReconcileAccount(ctx, userID, accountID, models.AccessLevelRead); err != nil {
		return nil, err
	}
	reports, grouped, err := c.repo.ListReconcileReports(ctx, accountID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list reconcile reports.")
	}
	res := make([]models.ReconcileReportResponse, 0, len(reports))
	for _, rep := range reports {
		res = append(res, toReconcileReportResponse(rep, grouped[rep.ID]))
	}
	return res, nil
}

// GetReconcileReport retrieves a single reconcile report and its attached records.
func (c *Controller) GetReconcileReport(ctx context.Context, userID, accountID, reportID string) (*models.ReconcileReportResponse, error) {
	if _, err := c.verifyReconcileAccount(ctx, userID, accountID, models.AccessLevelRead); err != nil {
		return nil, err
	}
	rep, records, err := c.repo.GetReconcileReportByID(ctx, accountID, reportID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrReconcileReportNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve reconcile report.")
	}
	res := toReconcileReportResponse(rep, records)
	return &res, nil
}

// DeleteReconcileReport deletes a reconcile report and unlinks attached records without deleting them.
func (c *Controller) DeleteReconcileReport(ctx context.Context, userID, accountID, reportID string) error {
	if _, err := c.verifyReconcileAccount(ctx, userID, accountID, models.AccessLevelEdit); err != nil {
		return err
	}
	if err := c.repo.DeleteReconcileReport(ctx, accountID, reportID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrReconcileReportNotFound
		}
		trace.LogError(ctx, err, "Failed to delete reconcile report %s on account %s", reportID, accountID)
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete reconcile report.")
	}
	trace.LogInfo(ctx, "User %s deleted reconcile report %s on account %s", userID, reportID, accountID)
	return nil
}
