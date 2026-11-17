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

func cloneBill(b *models.Bill) *models.Bill {
	copied := *b
	copied.AccountID = cloneStringPtr(b.AccountID)
	copied.CategoryID = cloneStringPtr(b.CategoryID)
	copied.LastDatePaid = cloneStringPtr(b.LastDatePaid)
	if b.Schedule != nil {
		s := *b.Schedule
		s.Nth = cloneIntPtr(b.Schedule.Nth)
		s.Day = cloneIntPtr(b.Schedule.Day)
		s.Month = cloneIntPtr(b.Schedule.Month)
		copied.Schedule = &s
	}
	return &copied
}

func (m *MemoryRepository) CreateBill(_ context.Context, bill *models.Bill) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.bills[bill.ID]; exists {
		return ErrConflict
	}
	m.bills[bill.ID] = cloneBill(bill)
	return nil
}

func (m *MemoryRepository) GetBillByID(_ context.Context, userID, billID string) (*models.Bill, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bill, ok := m.bills[billID]
	if !ok || bill.UserID != userID {
		return nil, ErrNotFound
	}
	return cloneBill(bill), nil
}

func (m *MemoryRepository) ListBillsByUserID(_ context.Context, userID string) ([]*models.Bill, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*models.Bill, 0)
	for _, bill := range m.bills {
		if bill.UserID == userID {
			result = append(result, cloneBill(bill))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].Recipient < result[j].Recipient
		}
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (m *MemoryRepository) UpdateBill(_ context.Context, bill *models.Bill) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.bills[bill.ID]
	if !ok || existing.UserID != bill.UserID {
		return ErrNotFound
	}
	m.bills[bill.ID] = cloneBill(bill)
	return nil
}

func (m *MemoryRepository) DeleteBill(_ context.Context, userID, billID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.bills[billID]
	if !ok || existing.UserID != userID {
		return ErrNotFound
	}
	delete(m.bills, billID)
	return nil
}

func (m *MemoryRepository) ApplyBillToAccount(_ context.Context, bill *models.Bill, record *models.CheckingRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.bills[bill.ID]
	if !ok || existing.UserID != bill.UserID {
		return ErrNotFound
	}
	copiedRec := *record
	m.checkingRecords[record.ID] = &copiedRec
	m.bills[bill.ID] = cloneBill(bill)
	return nil
}
