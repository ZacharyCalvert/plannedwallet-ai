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
	ErrBillNotFound = NewAppError(http.StatusNotFound, "BILL_NOT_FOUND", "Bill not found.")
)

func parseOptionalDateStr(raw *string, code, msg string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil, nil
	}
	if _, err := time.Parse("2006-01-02", trimmed); err != nil {
		return nil, NewAppError(http.StatusBadRequest, code, msg)
	}
	return &trimmed, nil
}

func (c *Controller) verifyWritableRecordAccount(ctx context.Context, userID, accountID string) (*models.Account, error) {
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
		if err != nil || access == nil || !checkAccessLevel(access.AccessLevel, models.AccessLevelEdit) {
			return nil, NewAppError(http.StatusForbidden, "FORBIDDEN", "Insufficient permissions for account.")
		}
	}
	if !isRecordSupportedAccount(acc.Type) {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_ACCOUNT_TYPE", "Bills may only link to checking, savings, or credit card accounts.")
	}
	return acc, nil
}

func (c *Controller) resolveBillAccountAndCategory(ctx context.Context, userID string, rawAccID, rawCatID *string) (*string, *string, error) {
	var accID *string
	if rawAccID != nil && strings.TrimSpace(*rawAccID) != "" {
		trimmed := strings.TrimSpace(*rawAccID)
		if _, err := c.verifyWritableRecordAccount(ctx, userID, trimmed); err != nil {
			return nil, nil, err
		}
		accID = &trimmed
	}

	if rawCatID != nil && strings.TrimSpace(*rawCatID) != "" && accID == nil {
		return nil, nil, NewAppError(http.StatusBadRequest, "INVALID_BILL_CATEGORY", "category_id cannot be set unless account_id is linked.")
	}
	catID, err := c.resolveCategoryIDForUser(ctx, userID, rawCatID)
	if err != nil {
		return nil, nil, err
	}
	return accID, catID, nil
}

func toBillResponse(b *models.Bill, refNow time.Time) models.BillResponse {
	nextDue, status, daysUntil := ComputeBillDueInfo(b, refNow)
	return models.BillResponse{
		ID:           b.ID,
		UserID:       b.UserID,
		AccountID:    b.AccountID,
		CategoryID:   b.CategoryID,
		Amount:       b.Amount,
		LastDatePaid: b.LastDatePaid,
		Frequency:    b.Frequency,
		Interval:     b.Interval,
		Schedule:     b.Schedule,
		Recipient:    b.Recipient,
		Description:  b.Description,
		Reminder:     b.Reminder,
		NextDueDate:  nextDue,
		DueStatus:    status,
		DaysUntilDue: daysUntil,
		CreatedAt:    b.CreatedAt,
		UpdatedAt:    b.UpdatedAt,
	}
}

func (c *Controller) validateAndBuildBill(ctx context.Context, userID, id string, req models.BillCreateRequest, createdAt time.Time) (*models.Bill, error) {
	recipient := strings.TrimSpace(req.Recipient)
	if recipient == "" {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_RECIPIENT", "recipient is required.")
	}
	if req.Amount == nil || *req.Amount == 0 {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_AMOUNT", "amount must be non-zero.")
	}
	lastPaid, err := parseOptionalDateStr(req.LastDatePaid, "INVALID_DATE", "last_date_paid must be formatted as YYYY-MM-DD.")
	if err != nil {
		return nil, err
	}
	freq, err := normalizeBillFrequency(req.Frequency)
	if err != nil {
		return nil, err
	}
	interval, err := normalizeBillInterval(req.Interval)
	if err != nil {
		return nil, err
	}
	sched, err := normalizeAndValidateSchedule(freq, req.Schedule)
	if err != nil {
		return nil, err
	}
	accID, catID, err := c.resolveBillAccountAndCategory(ctx, userID, req.AccountID, req.CategoryID)
	if err != nil {
		return nil, err
	}
	reminder := req.Reminder != nil && *req.Reminder
	return &models.Bill{
		ID:           id,
		UserID:       userID,
		AccountID:    accID,
		CategoryID:   catID,
		Amount:       *req.Amount,
		LastDatePaid: lastPaid,
		Frequency:    freq,
		Interval:     interval,
		Schedule:     sched,
		Recipient:    recipient,
		Description:  strings.TrimSpace(req.Description),
		Reminder:     reminder,
		CreatedAt:    createdAt,
		UpdatedAt:    time.Now().UTC(),
	}, nil
}

// CreateBill creates a new recurring bill owned by userID.
func (c *Controller) CreateBill(ctx context.Context, userID string, req models.BillCreateRequest) (*models.BillResponse, error) {
	now := time.Now().UTC()
	bill, err := c.validateAndBuildBill(ctx, userID, uuid.New().String(), req, now)
	if err != nil {
		return nil, err
	}
	bill.UpdatedAt = now
	if err := c.repo.CreateBill(ctx, bill); err != nil {
		trace.LogError(ctx, err, "Failed to create bill for user %s", userID)
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create bill.")
	}
	trace.LogInfo(ctx, "Created bill %s for user %s", bill.ID, userID)
	res := toBillResponse(bill, now)
	return &res, nil
}

// ListBills lists all bills owned by userID.
func (c *Controller) ListBills(ctx context.Context, userID string) ([]models.BillResponse, error) {
	bills, err := c.repo.ListBillsByUserID(ctx, userID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list bills.")
	}
	now := time.Now().UTC()
	res := make([]models.BillResponse, 0, len(bills))
	for _, b := range bills {
		res = append(res, toBillResponse(b, now))
	}
	return res, nil
}

// ListBillReminders returns upcoming or overdue bills with reminder enabled.
func (c *Controller) ListBillReminders(ctx context.Context, userID string) ([]models.BillResponse, error) {
	all, err := c.ListBills(ctx, userID)
	if err != nil {
		return nil, err
	}
	reminders := make([]models.BillResponse, 0)
	for _, b := range all {
		if b.Reminder && (b.DueStatus == models.BillStatusOverdue || b.DueStatus == models.BillStatusUpcoming) {
			reminders = append(reminders, b)
		}
	}
	return reminders, nil
}

// GetBill retrieves a bill owned by userID.
func (c *Controller) GetBill(ctx context.Context, userID, billID string) (*models.BillResponse, error) {
	bill, err := c.repo.GetBillByID(ctx, userID, billID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrBillNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve bill.")
	}
	res := toBillResponse(bill, time.Now().UTC())
	return &res, nil
}

// UpdateBill updates an existing bill owned by userID.
func (c *Controller) UpdateBill(ctx context.Context, userID, billID string, req models.BillUpdateRequest) (*models.BillResponse, error) {
	existing, err := c.repo.GetBillByID(ctx, userID, billID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrBillNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve bill.")
	}
	updated, err := c.validateAndBuildBill(ctx, userID, billID, req, existing.CreatedAt)
	if err != nil {
		return nil, err
	}
	if err := c.repo.UpdateBill(ctx, updated); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrBillNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update bill.")
	}
	trace.LogInfo(ctx, "Updated bill %s for user %s", billID, userID)
	res := toBillResponse(updated, time.Now().UTC())
	return &res, nil
}

// DeleteBill deletes a bill owned by userID.
func (c *Controller) DeleteBill(ctx context.Context, userID, billID string) error {
	if err := c.repo.DeleteBill(ctx, userID, billID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrBillNotFound
		}
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete bill.")
	}
	trace.LogInfo(ctx, "Deleted bill %s for user %s", billID, userID)
	return nil
}

func resolveBillApplyTarget(bill *models.Bill, reqAccountID *string) (string, *string, error) {
	if reqAccountID != nil && strings.TrimSpace(*reqAccountID) != "" {
		target := strings.TrimSpace(*reqAccountID)
		var catID *string
		if bill.AccountID != nil && *bill.AccountID == target {
			catID = bill.CategoryID
		}
		return target, catID, nil
	}
	if bill.AccountID == nil || strings.TrimSpace(*bill.AccountID) == "" {
		return "", nil, NewAppError(http.StatusBadRequest, "BILL_NO_ACCOUNT", "Bill is not linked to an account.")
	}
	return *bill.AccountID, bill.CategoryID, nil
}

func parseBillApplyDates(req models.BillApplyRequest, now time.Time) (string, *string, error) {
	applyDate := now.Format("2006-01-02")
	parsed, err := parseOptionalDateStr(req.Date, "INVALID_DATE", "date must be formatted as YYYY-MM-DD.")
	if err != nil {
		return "", nil, err
	}
	if parsed != nil {
		applyDate = *parsed
	}
	dateCleared, err := parseOptionalDateStr(req.DateCleared, "INVALID_DATE", "date_cleared must be formatted as YYYY-MM-DD.")
	if err != nil {
		return "", nil, err
	}
	return applyDate, dateCleared, nil
}

// ApplyBillToAccount creates a checking record on the target account and updates last_date_paid.
func (c *Controller) ApplyBillToAccount(ctx context.Context, userID, billID string, req models.BillApplyRequest) (*models.BillApplyResponse, error) {
	bill, err := c.repo.GetBillByID(ctx, userID, billID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrBillNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve bill.")
	}
	targetAccID, catID, err := resolveBillApplyTarget(bill, req.AccountID)
	if err != nil {
		return nil, err
	}
	if _, err := c.verifyWritableRecordAccount(ctx, userID, targetAccID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	applyDate, dateCleared, err := parseBillApplyDates(req, now)
	if err != nil {
		return nil, err
	}
	rec := &models.CheckingRecord{
		ID: uuid.New().String(), AccountID: targetAccID, CategoryID: catID,
		Value: bill.Amount, DateSent: applyDate, DateCleared: dateCleared,
		Entity: bill.Recipient, Description: bill.Description, CreatedAt: now, UpdatedAt: now,
	}
	bill.LastDatePaid = &applyDate
	bill.UpdatedAt = now
	if err := c.repo.ApplyBillToAccount(ctx, bill, rec); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to apply bill to account.")
	}
	trace.LogInfo(ctx, "Applied bill %s to account %s as record %s", bill.ID, targetAccID, rec.ID)
	return &models.BillApplyResponse{Bill: toBillResponse(bill, now), Record: toCheckingRecordResponse(rec)}, nil
}
