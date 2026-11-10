package repository

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/plannedwallet/internal/models"
)

type MemoryRepository struct {
	mu              sync.RWMutex
	users           map[string]*models.User           // id -> User
	usersByUsername map[string]*models.User           // username -> User
	usersByEmail    map[string]*models.User           // email -> User
	accounts        map[string]*models.Account        // id -> Account
	accountAccess   map[string]*models.AccountAccess  // accountID:userID -> AccountAccess
	checkingRecords map[string]*models.CheckingRecord // id -> CheckingRecord
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		users:           make(map[string]*models.User),
		usersByUsername: make(map[string]*models.User),
		usersByEmail:    make(map[string]*models.User),
		accounts:        make(map[string]*models.Account),
		accountAccess:   make(map[string]*models.AccountAccess),
		checkingRecords: make(map[string]*models.CheckingRecord),
	}
}

func (m *MemoryRepository) CreateUser(_ context.Context, user *models.User) error {
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

	copied := *user
	m.users[user.ID] = &copied
	m.usersByUsername[lowerUsername] = &copied
	m.usersByEmail[lowerEmail] = &copied
	return nil
}

func (m *MemoryRepository) GetUserByUsername(_ context.Context, username string) (*models.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, ok := m.usersByUsername[strings.ToLower(username)]
	if !ok {
		return nil, ErrNotFound
	}
	copied := *user
	return &copied, nil
}

func (m *MemoryRepository) GetUserByEmail(_ context.Context, email string) (*models.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, ok := m.usersByEmail[strings.ToLower(email)]
	if !ok {
		return nil, ErrNotFound
	}
	copied := *user
	return &copied, nil
}

func (m *MemoryRepository) GetUserByID(_ context.Context, id string) (*models.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, ok := m.users[id]
	if !ok {
		return nil, ErrNotFound
	}
	copied := *user
	return &copied, nil
}

func (m *MemoryRepository) CreateAccount(_ context.Context, account *models.Account) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	copied := *account
	m.accounts[account.ID] = &copied
	return nil
}

func (m *MemoryRepository) GetAccountByID(_ context.Context, id string) (*models.Account, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	acc, ok := m.accounts[id]
	if !ok {
		return nil, ErrNotFound
	}
	copied := *acc
	return &copied, nil
}

func (m *MemoryRepository) ListAccountsByUserID(_ context.Context, userID string) ([]*models.Account, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*models.Account
	for _, acc := range m.accounts {
		hasAccess := acc.AdminID == userID
		if !hasAccess {
			key := acc.ID + ":" + userID
			_, hasAccess = m.accountAccess[key]
		}
		if hasAccess {
			copied := *acc
			result = append(result, &copied)
		}
	}
	return result, nil
}

func (m *MemoryRepository) GrantAccountAccess(_ context.Context, access *models.AccountAccess) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	copied := *access
	key := access.AccountID + ":" + access.UserID
	m.accountAccess[key] = &copied
	return nil
}

func (m *MemoryRepository) GetAccountAccess(_ context.Context, accountID, userID string) (*models.AccountAccess, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := accountID + ":" + userID
	acc, ok := m.accountAccess[key]
	if !ok {
		return nil, ErrNotFound
	}
	copied := *acc
	return &copied, nil
}

func (m *MemoryRepository) ListAccountAccess(_ context.Context, accountID string) ([]*models.AccountAccess, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*models.AccountAccess
	for _, acc := range m.accountAccess {
		if acc.AccountID == accountID {
			copied := *acc
			result = append(result, &copied)
		}
	}
	return result, nil
}

func (m *MemoryRepository) RevokeAccountAccess(_ context.Context, accountID, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := accountID + ":" + userID
	if _, ok := m.accountAccess[key]; !ok {
		return ErrNotFound
	}
	delete(m.accountAccess, key)
	return nil
}

func (m *MemoryRepository) CreateCheckingRecord(_ context.Context, record *models.CheckingRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	copied := *record
	m.checkingRecords[record.ID] = &copied
	return nil
}

func (m *MemoryRepository) GetCheckingRecordByID(_ context.Context, accountID, recordID string) (*models.CheckingRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rec, ok := m.checkingRecords[recordID]
	if !ok || rec.AccountID != accountID {
		return nil, ErrNotFound
	}
	copied := *rec
	return &copied, nil
}

func (m *MemoryRepository) ListCheckingRecords(_ context.Context, accountID string, cleared *bool, startDate, endDate *string) ([]*models.CheckingRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*models.CheckingRecord
	for _, rec := range m.checkingRecords {
		if rec.AccountID != accountID {
			continue
		}
		if cleared != nil {
			isCleared := rec.DateCleared != nil && *rec.DateCleared != ""
			if *cleared != isCleared {
				continue
			}
		}
		if startDate != nil && *startDate != "" && rec.DateSent < *startDate {
			continue
		}
		if endDate != nil && *endDate != "" && rec.DateSent > *endDate {
			continue
		}
		copied := *rec
		result = append(result, &copied)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].DateSent == result[j].DateSent {
			return result[i].CreatedAt.After(result[j].CreatedAt)
		}
		return result[i].DateSent > result[j].DateSent
	})

	return result, nil
}

func (m *MemoryRepository) UpdateCheckingRecord(_ context.Context, record *models.CheckingRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rec, ok := m.checkingRecords[record.ID]
	if !ok || rec.AccountID != record.AccountID {
		return ErrNotFound
	}
	copied := *record
	m.checkingRecords[record.ID] = &copied
	return nil
}

func (m *MemoryRepository) DeleteCheckingRecord(_ context.Context, accountID, recordID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rec, ok := m.checkingRecords[recordID]
	if !ok || rec.AccountID != accountID {
		return ErrNotFound
	}
	delete(m.checkingRecords, recordID)
	return nil
}
