package repository

import (
	"context"
	"sort"

	"github.com/plannedwallet/internal/models"
)

func (m *MemoryRepository) CreatePolicyAgreement(_ context.Context, agreement *models.PolicyAgreement) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.users[agreement.UserID]; !ok {
		return ErrNotFound
	}
	for _, existing := range m.policyAgreements {
		if existing.UserID == agreement.UserID && existing.Active {
			existing.Active = false
		}
	}
	copied := *agreement
	m.policyAgreements[agreement.ID] = &copied
	return nil
}

func (m *MemoryRepository) GetActivePolicyAgreement(_ context.Context, userID string) (*models.PolicyAgreement, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var latest *models.PolicyAgreement
	for _, pa := range m.policyAgreements {
		if pa.UserID == userID && pa.Active {
			if latest == nil || pa.AgreedAt.After(latest.AgreedAt) {
				latest = pa
			}
		}
	}
	if latest == nil {
		return nil, ErrNotFound
	}
	copied := *latest
	return &copied, nil
}

func (m *MemoryRepository) ListPolicyAgreementsByUserID(_ context.Context, userID string) ([]*models.PolicyAgreement, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*models.PolicyAgreement, 0)
	for _, pa := range m.policyAgreements {
		if pa.UserID == userID {
			copied := *pa
			result = append(result, &copied)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].AgreedAt.After(result[j].AgreedAt)
	})
	return result, nil
}

func (m *MemoryRepository) DeactivateAllPolicyAgreements(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, pa := range m.policyAgreements {
		pa.Active = false
	}
	return nil
}
