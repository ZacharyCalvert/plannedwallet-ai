package repository

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/plannedwallet/internal/models"
)

func copyRecoveryKey(rk *models.UserRecoveryKey) *models.UserRecoveryKey {
	copied := *rk
	if rk.UsedAt != nil {
		t := *rk.UsedAt
		copied.UsedAt = &t
	}
	return &copied
}

func (m *MemoryRepository) CreateUserWithRecoveryKey(_ context.Context, user *models.User, key *models.UserRecoveryKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	lowerUsername := strings.ToLower(user.Username)
	lowerEmail := strings.ToLower(user.Email)
	if _, exists := m.usersByUsername[lowerUsername]; exists {
		return ErrConflict
	}
	if _, exists := m.usersByEmail[lowerEmail]; exists {
		return ErrConflict
	}

	copiedUser := *user
	m.users[user.ID] = &copiedUser
	m.usersByUsername[lowerUsername] = &copiedUser
	m.usersByEmail[lowerEmail] = &copiedUser
	m.recoveryKeys[key.ID] = copyRecoveryKey(key)
	return nil
}

func (m *MemoryRepository) GetActiveRecoveryKeyByUserID(_ context.Context, userID string) (*models.UserRecoveryKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var latest *models.UserRecoveryKey
	for _, rk := range m.recoveryKeys {
		if rk.UserID == userID && rk.Active {
			if latest == nil || rk.CreatedAt.After(latest.CreatedAt) {
				latest = rk
			}
		}
	}
	if latest == nil {
		return nil, ErrNotFound
	}
	return copyRecoveryKey(latest), nil
}

func (m *MemoryRepository) ListRecoveryKeysByUserID(_ context.Context, userID string) ([]*models.UserRecoveryKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*models.UserRecoveryKey
	for _, rk := range m.recoveryKeys {
		if rk.UserID == userID {
			result = append(result, copyRecoveryKey(rk))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (m *MemoryRepository) CompleteAccountRecovery(_ context.Context, userID, newPasswordHash, newTOTPSecret, consumedKeyID, usedIP string, usedAt time.Time, replacementKey *models.UserRecoveryKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.users[userID]
	if !ok {
		return ErrNotFound
	}
	consumed, ok := m.recoveryKeys[consumedKeyID]
	if !ok || consumed.UserID != userID || !consumed.Active {
		return ErrConflict
	}

	t := usedAt
	consumed.Active = false
	consumed.UsedAt = &t
	consumed.UsedIP = usedIP

	user.PasswordHash = newPasswordHash
	user.TOTPSecret = newTOTPSecret
	m.recoveryKeys[replacementKey.ID] = copyRecoveryKey(replacementKey)
	return nil
}

func (m *MemoryRepository) RotateRecoveryKey(_ context.Context, userID string, newKey *models.UserRecoveryKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.users[userID]; !ok {
		return ErrNotFound
	}
	for _, rk := range m.recoveryKeys {
		if rk.UserID == userID && rk.Active {
			rk.Active = false
		}
	}
	m.recoveryKeys[newKey.ID] = copyRecoveryKey(newKey)
	return nil
}
