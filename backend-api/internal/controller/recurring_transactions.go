package controller

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/repository"
	"github.com/plannedwallet/internal/trace"
)

// ErrRecurringTransactionNotFound is returned when a recurring transaction does not exist or belongs to another user.
var ErrRecurringTransactionNotFound = NewAppError(http.StatusNotFound, "RECURRING_TRANSACTION_NOT_FOUND", "Recurring transaction not found.")

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
		return nil, NewAppError(http.StatusBadRequest, "INVALID_ACCOUNT_TYPE", "Recurring transactions may only link to checking, savings, or credit card accounts.")
	}
	return acc, nil
}

func (c *Controller) resolveRecurringAccountAndCategory(ctx context.Context, userID string, rawAccID, rawCatID *string) (*string, *string, error) {
	var accID *string
	if rawAccID != nil && strings.TrimSpace(*rawAccID) != "" {
		trimmed := strings.TrimSpace(*rawAccID)
		if _, err := c.verifyWritableRecordAccount(ctx, userID, trimmed); err != nil {
			return nil, nil, err
		}
		accID = &trimmed
	}

	if rawCatID != nil && strings.TrimSpace(*rawCatID) != "" && accID == nil {
		return nil, nil, NewAppError(http.StatusBadRequest, "INVALID_RECURRING_CATEGORY", "category_id cannot be set unless account_id is linked.")
	}
	catID, err := c.resolveCategoryIDForUser(ctx, userID, rawCatID)
	if err != nil {
		return nil, nil, err
	}
	return accID, catID, nil
}

func toRecurringTransactionResponse(rt *models.RecurringTransaction, refNow time.Time) models.RecurringTransactionResponse {
	nextDue, status, daysUntil := ComputeRecurringDueInfo(rt, refNow)
	return models.RecurringTransactionResponse{
		ID:              rt.ID,
		UserID:          rt.UserID,
		AccountID:       rt.AccountID,
		CategoryID:      rt.CategoryID,
		Amount:          rt.Amount,
		LastDateApplied: rt.LastDateApplied,
		Frequency:       rt.Frequency,
		Interval:        rt.Interval,
		Schedule:        rt.Schedule,
		Name:            rt.Name,
		Entity:          rt.Entity,
		Description:     rt.Description,
		Reminder:        rt.Reminder,
		NextDueDate:     nextDue,
		DueStatus:       status,
		DaysUntilDue:    daysUntil,
		CreatedAt:       rt.CreatedAt,
		UpdatedAt:       rt.UpdatedAt,
	}
}

func validateRecurringIdentity(req models.RecurringTransactionRequest) (string, string, error) {
	name := strings.TrimSpace(req.Name)
	if utf8.RuneCountInString(name) >= 20 {
		return "", "", NewAppError(http.StatusBadRequest, "INVALID_NAME", "name must be under 20 characters.")
	}
	entity := strings.TrimSpace(req.Entity)
	if entity == "" {
		return "", "", NewAppError(http.StatusBadRequest, "INVALID_ENTITY", "entity is required.")
	}
	if req.Amount == nil || *req.Amount == 0 {
		return "", "", NewAppError(http.StatusBadRequest, "INVALID_AMOUNT", "amount must be non-zero.")
	}
	return name, entity, nil
}

func validateRecurringTiming(req models.RecurringTransactionRequest) (*string, string, int, *models.RecurringSchedule, error) {
	lastApplied, err := parseOptionalDateStr(req.LastDateApplied, "INVALID_DATE", "last_date_applied must be formatted as YYYY-MM-DD.")
	if err != nil {
		return nil, "", 0, nil, err
	}
	freq, err := normalizeRecurringFrequency(req.Frequency)
	if err != nil {
		return nil, "", 0, nil, err
	}
	interval, err := normalizeRecurringInterval(req.Interval)
	if err != nil {
		return nil, "", 0, nil, err
	}
	sched, err := normalizeAndValidateRecurringSchedule(freq, req.Schedule)
	if err != nil {
		return nil, "", 0, nil, err
	}
	return lastApplied, freq, interval, sched, nil
}

func (c *Controller) validateAndBuildRecurring(ctx context.Context, userID, id string, req models.RecurringTransactionRequest, createdAt time.Time) (*models.RecurringTransaction, error) {
	name, entity, err := validateRecurringIdentity(req)
	if err != nil {
		return nil, err
	}
	lastApplied, freq, interval, sched, err := validateRecurringTiming(req)
	if err != nil {
		return nil, err
	}
	accID, catID, err := c.resolveRecurringAccountAndCategory(ctx, userID, req.AccountID, req.CategoryID)
	if err != nil {
		return nil, err
	}
	return &models.RecurringTransaction{
		ID:              id,
		UserID:          userID,
		AccountID:       accID,
		CategoryID:      catID,
		Amount:          *req.Amount,
		LastDateApplied: lastApplied,
		Frequency:       freq,
		Interval:        interval,
		Schedule:        sched,
		Name:            name,
		Entity:          entity,
		Description:     strings.TrimSpace(req.Description),
		Reminder:        req.Reminder != nil && *req.Reminder,
		CreatedAt:       createdAt,
		UpdatedAt:       time.Now().UTC(),
	}, nil
}

// CreateRecurringTransaction creates a new recurring transaction owned by userID.
func (c *Controller) CreateRecurringTransaction(ctx context.Context, userID string, req models.RecurringTransactionRequest) (*models.RecurringTransactionResponse, error) {
	now := time.Now().UTC()
	rt, err := c.validateAndBuildRecurring(ctx, userID, uuid.New().String(), req, now)
	if err != nil {
		return nil, err
	}
	rt.UpdatedAt = now
	if err := c.repo.CreateRecurringTransaction(ctx, rt); err != nil {
		trace.LogError(ctx, err, "Failed to create recurring transaction for user %s", userID)
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create recurring transaction.")
	}
	trace.LogInfo(ctx, "Created recurring transaction %s for user %s", rt.ID, userID)
	res := toRecurringTransactionResponse(rt, now)
	return &res, nil
}

// ListRecurringTransactions lists all recurring transactions owned by userID.
func (c *Controller) ListRecurringTransactions(ctx context.Context, userID string) ([]models.RecurringTransactionResponse, error) {
	items, err := c.repo.ListRecurringTransactionsByUserID(ctx, userID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list recurring transactions.")
	}
	now := time.Now().UTC()
	res := make([]models.RecurringTransactionResponse, 0, len(items))
	for _, rt := range items {
		res = append(res, toRecurringTransactionResponse(rt, now))
	}
	return res, nil
}

// ListRecurringReminders returns upcoming or overdue recurring transactions with reminder enabled.
func (c *Controller) ListRecurringReminders(ctx context.Context, userID string) ([]models.RecurringTransactionResponse, error) {
	all, err := c.ListRecurringTransactions(ctx, userID)
	if err != nil {
		return nil, err
	}
	reminders := make([]models.RecurringTransactionResponse, 0)
	for _, rt := range all {
		if rt.Reminder && (rt.DueStatus == models.RecurringStatusOverdue || rt.DueStatus == models.RecurringStatusUpcoming) {
			reminders = append(reminders, rt)
		}
	}
	return reminders, nil
}

// GetRecurringTransaction retrieves a recurring transaction owned by userID.
func (c *Controller) GetRecurringTransaction(ctx context.Context, userID, recurringID string) (*models.RecurringTransactionResponse, error) {
	rt, err := c.repo.GetRecurringTransactionByID(ctx, userID, recurringID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrRecurringTransactionNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve recurring transaction.")
	}
	res := toRecurringTransactionResponse(rt, time.Now().UTC())
	return &res, nil
}

// UpdateRecurringTransaction replaces an existing recurring transaction owned by userID.
func (c *Controller) UpdateRecurringTransaction(ctx context.Context, userID, recurringID string, req models.RecurringTransactionRequest) (*models.RecurringTransactionResponse, error) {
	existing, err := c.repo.GetRecurringTransactionByID(ctx, userID, recurringID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrRecurringTransactionNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve recurring transaction.")
	}
	updated, err := c.validateAndBuildRecurring(ctx, userID, recurringID, req, existing.CreatedAt)
	if err != nil {
		return nil, err
	}
	if err := c.repo.UpdateRecurringTransaction(ctx, updated); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrRecurringTransactionNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update recurring transaction.")
	}
	trace.LogInfo(ctx, "Updated recurring transaction %s for user %s", recurringID, userID)
	res := toRecurringTransactionResponse(updated, time.Now().UTC())
	return &res, nil
}

// DeleteRecurringTransaction deletes a recurring transaction owned by userID.
func (c *Controller) DeleteRecurringTransaction(ctx context.Context, userID, recurringID string) error {
	if err := c.repo.DeleteRecurringTransaction(ctx, userID, recurringID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrRecurringTransactionNotFound
		}
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete recurring transaction.")
	}
	trace.LogInfo(ctx, "Deleted recurring transaction %s for user %s", recurringID, userID)
	return nil
}

func resolveRecurringApplyTarget(rt *models.RecurringTransaction, reqAccountID *string) (string, *string, error) {
	if reqAccountID != nil && strings.TrimSpace(*reqAccountID) != "" {
		target := strings.TrimSpace(*reqAccountID)
		var catID *string
		if rt.AccountID != nil && *rt.AccountID == target {
			catID = rt.CategoryID
		}
		return target, catID, nil
	}
	if rt.AccountID == nil || strings.TrimSpace(*rt.AccountID) == "" {
		return "", nil, NewAppError(http.StatusBadRequest, "RECURRING_NO_ACCOUNT", "Recurring transaction is not linked to an account.")
	}
	return *rt.AccountID, rt.CategoryID, nil
}

func parseRecurringApplyDates(req models.RecurringTransactionApplyRequest, now time.Time) (string, *string, error) {
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

// ApplyRecurringTransactionToAccount creates a checking record on the target account and updates last_date_applied.
func (c *Controller) ApplyRecurringTransactionToAccount(ctx context.Context, userID, recurringID string, req models.RecurringTransactionApplyRequest) (*models.RecurringTransactionApplyResponse, error) {
	rt, err := c.repo.GetRecurringTransactionByID(ctx, userID, recurringID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrRecurringTransactionNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve recurring transaction.")
	}
	targetAccID, catID, err := resolveRecurringApplyTarget(rt, req.AccountID)
	if err != nil {
		return nil, err
	}
	if _, err := c.verifyWritableRecordAccount(ctx, userID, targetAccID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	applyDate, dateCleared, err := parseRecurringApplyDates(req, now)
	if err != nil {
		return nil, err
	}
	rec := &models.CheckingRecord{
		ID: uuid.New().String(), AccountID: targetAccID, CategoryID: catID,
		Value: rt.Amount, DateSent: applyDate, DateCleared: dateCleared,
		Entity: rt.Entity, Description: rt.Description, CreatedAt: now, UpdatedAt: now,
	}
	rt.LastDateApplied = &applyDate
	rt.UpdatedAt = now
	if err := c.repo.ApplyRecurringTransactionToAccount(ctx, rt, rec); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to apply recurring transaction to account.")
	}
	trace.LogInfo(ctx, "Applied recurring transaction %s to account %s as record %s", rt.ID, targetAccID, rec.ID)
	return &models.RecurringTransactionApplyResponse{RecurringTransaction: toRecurringTransactionResponse(rt, now), Record: toCheckingRecordResponse(rec)}, nil
}
