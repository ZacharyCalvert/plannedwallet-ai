package repository

import (
	"context"
	"sort"

	"github.com/plannedwallet/internal/models"
)

func cloneStringPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func cloneIntPtr(p *int) *int {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func cloneRecurringSchedule(src *models.RecurringSchedule) *models.RecurringSchedule {
	if src == nil {
		return nil
	}
	s := *src
	s.Nth = cloneIntPtr(src.Nth)
	s.Day = cloneIntPtr(src.Day)
	s.Month = cloneIntPtr(src.Month)
	return &s
}

func cloneRecurringTransaction(rt *models.RecurringTransaction) *models.RecurringTransaction {
	copied := *rt
	copied.AccountID = cloneStringPtr(rt.AccountID)
	copied.CategoryID = cloneStringPtr(rt.CategoryID)
	copied.LastDateApplied = cloneStringPtr(rt.LastDateApplied)
	copied.Schedule = cloneRecurringSchedule(rt.Schedule)
	return &copied
}

// CreateRecurringTransaction persists a recurring transaction in memory.
func (m *MemoryRepository) CreateRecurringTransaction(_ context.Context, rt *models.RecurringTransaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.recurringTransactions[rt.ID]; exists {
		return ErrConflict
	}
	m.recurringTransactions[rt.ID] = cloneRecurringTransaction(rt)
	return nil
}

// GetRecurringTransactionByID returns the user-owned recurring transaction or ErrNotFound.
func (m *MemoryRepository) GetRecurringTransactionByID(_ context.Context, userID, recurringID string) (*models.RecurringTransaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rt, ok := m.recurringTransactions[recurringID]
	if !ok || rt.UserID != userID {
		return nil, ErrNotFound
	}
	return cloneRecurringTransaction(rt), nil
}

// ListRecurringTransactionsByUserID lists all recurring transactions owned by userID ordered newest-first.
func (m *MemoryRepository) ListRecurringTransactionsByUserID(_ context.Context, userID string) ([]*models.RecurringTransaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*models.RecurringTransaction, 0)
	for _, rt := range m.recurringTransactions {
		if rt.UserID == userID {
			result = append(result, cloneRecurringTransaction(rt))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].Entity < result[j].Entity
		}
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

// UpdateRecurringTransaction replaces an existing user-owned recurring transaction.
func (m *MemoryRepository) UpdateRecurringTransaction(_ context.Context, rt *models.RecurringTransaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.recurringTransactions[rt.ID]
	if !ok || existing.UserID != rt.UserID {
		return ErrNotFound
	}
	m.recurringTransactions[rt.ID] = cloneRecurringTransaction(rt)
	return nil
}

// DeleteRecurringTransaction removes a user-owned recurring transaction.
func (m *MemoryRepository) DeleteRecurringTransaction(_ context.Context, userID, recurringID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.recurringTransactions[recurringID]
	if !ok || existing.UserID != userID {
		return ErrNotFound
	}
	delete(m.recurringTransactions, recurringID)
	return nil
}

// ApplyRecurringTransactionToAccount stores the generated checking record and updates last_date_applied together.
func (m *MemoryRepository) ApplyRecurringTransactionToAccount(_ context.Context, rt *models.RecurringTransaction, record *models.CheckingRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.recurringTransactions[rt.ID]
	if !ok || existing.UserID != rt.UserID {
		return ErrNotFound
	}
	copiedRec := *record
	m.checkingRecords[record.ID] = &copiedRec
	m.recurringTransactions[rt.ID] = cloneRecurringTransaction(rt)
	return nil
}
