package repository

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

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
	assets          map[string]*models.Asset          // id -> Asset
	assetValues     map[string]*models.AssetValue     // id -> AssetValue
	accountAssets   map[string]*models.AccountAsset   // id -> AccountAsset
	accountSnapshots map[string]*models.AccountSnapshot // id -> AccountSnapshot
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		users:           make(map[string]*models.User),
		usersByUsername: make(map[string]*models.User),
		usersByEmail:    make(map[string]*models.User),
		accounts:        make(map[string]*models.Account),
		accountAccess:   make(map[string]*models.AccountAccess),
		checkingRecords: make(map[string]*models.CheckingRecord),
		assets:          make(map[string]*models.Asset),
		assetValues:     make(map[string]*models.AssetValue),
		accountAssets:   make(map[string]*models.AccountAsset),
		accountSnapshots: make(map[string]*models.AccountSnapshot),
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

func sameUser(u1, u2 *string) bool {
	if u1 == nil && u2 == nil {
		return true
	}
	if u1 != nil && u2 != nil {
		return *u1 == *u2
	}
	return false
}

func (m *MemoryRepository) hasAssetConflict(asset *models.Asset) bool {
	for _, a := range m.assets {
		if a.ID == asset.ID {
			continue
		}
		if sameUser(a.UserID, asset.UserID) && strings.EqualFold(a.Label, asset.Label) {
			return true
		}
	}
	return false
}

func (m *MemoryRepository) CreateAsset(_ context.Context, asset *models.Asset) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.hasAssetConflict(asset) {
		return ErrConflict
	}
	copied := *asset
	m.assets[asset.ID] = &copied
	return nil
}

func (m *MemoryRepository) GetAssetByID(_ context.Context, id string) (*models.Asset, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	asset, ok := m.assets[id]
	if !ok {
		return nil, ErrNotFound
	}
	copied := *asset
	return &copied, nil
}

func matchesAsset(a *models.Asset, userID, label *string) bool {
	if userID != nil && *userID != "" {
		if a.UserID != nil && *a.UserID != *userID {
			return false
		}
	}
	if label != nil && *label != "" && !strings.EqualFold(a.Label, *label) {
		return false
	}
	return true
}

func (m *MemoryRepository) ListAssets(_ context.Context, userID, label *string) ([]*models.Asset, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*models.Asset
	for _, a := range m.assets {
		if matchesAsset(a, userID, label) {
			copied := *a
			result = append(result, &copied)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Label) < strings.ToLower(result[j].Label)
	})

	return result, nil
}

func (m *MemoryRepository) UpdateAsset(_ context.Context, asset *models.Asset) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.assets[asset.ID]; !ok {
		return ErrNotFound
	}
	if m.hasAssetConflict(asset) {
		return ErrConflict
	}
	copied := *asset
	m.assets[asset.ID] = &copied
	return nil
}

func (m *MemoryRepository) DeleteAsset(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.assets[id]; !ok {
		return ErrNotFound
	}
	delete(m.assets, id)
	for vid, v := range m.assetValues {
		if v.AssetID == id {
			delete(m.assetValues, vid)
		}
	}
	return nil
}

func (m *MemoryRepository) hasAssetValueConflict(av *models.AssetValue) bool {
	for _, v := range m.assetValues {
		if v.ID == av.ID {
			continue
		}
		if v.AssetID == av.AssetID && v.ValueDate == av.ValueDate {
			return true
		}
	}
	return false
}

func (m *MemoryRepository) CreateAssetValue(_ context.Context, av *models.AssetValue) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.assets[av.AssetID]; !ok {
		return ErrNotFound
	}
	if m.hasAssetValueConflict(av) {
		return ErrConflict
	}
	copied := *av
	m.assetValues[av.ID] = &copied
	return nil
}

func (m *MemoryRepository) GetAssetValueByID(_ context.Context, assetID, valueID string) (*models.AssetValue, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	av, ok := m.assetValues[valueID]
	if !ok || av.AssetID != assetID {
		return nil, ErrNotFound
	}
	copied := *av
	return &copied, nil
}

func matchesAssetValue(av *models.AssetValue, assetID string, startDate, endDate *string) bool {
	if av.AssetID != assetID {
		return false
	}
	if startDate != nil && *startDate != "" && av.ValueDate < *startDate {
		return false
	}
	if endDate != nil && *endDate != "" && av.ValueDate > *endDate {
		return false
	}
	return true
}

func (m *MemoryRepository) ListAssetValues(_ context.Context, assetID string, startDate, endDate *string) ([]*models.AssetValue, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*models.AssetValue
	for _, av := range m.assetValues {
		if matchesAssetValue(av, assetID, startDate, endDate) {
			copied := *av
			result = append(result, &copied)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].ValueDate == result[j].ValueDate {
			return result[i].CreatedAt.After(result[j].CreatedAt)
		}
		return result[i].ValueDate > result[j].ValueDate
	})

	return result, nil
}

func (m *MemoryRepository) UpdateAssetValue(_ context.Context, av *models.AssetValue) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.assetValues[av.ID]
	if !ok || existing.AssetID != av.AssetID {
		return ErrNotFound
	}
	if m.hasAssetValueConflict(av) {
		return ErrConflict
	}
	copied := *av
	m.assetValues[av.ID] = &copied
	return nil
}

func (m *MemoryRepository) DeleteAssetValue(_ context.Context, assetID, valueID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	av, ok := m.assetValues[valueID]
	if !ok || av.AssetID != assetID {
		return ErrNotFound
	}
	delete(m.assetValues, valueID)
	return nil
}

func (m *MemoryRepository) hasAccountAssetConflict(aa *models.AccountAsset) bool {
	for _, item := range m.accountAssets {
		if item.ID == aa.ID {
			continue
		}
		if item.AccountID == aa.AccountID && item.AssetID == aa.AssetID {
			return true
		}
	}
	return false
}

func (m *MemoryRepository) AddAccountAsset(_ context.Context, aa *models.AccountAsset) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.accounts[aa.AccountID]; !ok {
		return ErrNotFound
	}
	if _, ok := m.assets[aa.AssetID]; !ok {
		return ErrNotFound
	}
	if m.hasAccountAssetConflict(aa) {
		return ErrConflict
	}
	copied := *aa
	m.accountAssets[aa.ID] = &copied
	return nil
}

func (m *MemoryRepository) findAccountAsset(accountID, identifier string) *models.AccountAsset {
	for _, item := range m.accountAssets {
		if item.AccountID == accountID && (item.ID == identifier || item.AssetID == identifier) {
			return item
		}
	}
	return nil
}

func (m *MemoryRepository) GetAccountAsset(_ context.Context, accountID, identifier string) (*models.AccountAsset, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item := m.findAccountAsset(accountID, identifier)
	if item == nil {
		return nil, ErrNotFound
	}
	copied := *item
	return &copied, nil
}

func (m *MemoryRepository) getLatestAssetValue(assetID string) (float64, string) {
	var latestVal float64
	var latestDate string
	var latestCreated time.Time

	for _, v := range m.assetValues {
		if v.AssetID != assetID {
			continue
		}
		if v.ValueDate > latestDate || (v.ValueDate == latestDate && v.CreatedAt.After(latestCreated)) {
			latestDate = v.ValueDate
			latestVal = v.Value
			latestCreated = v.CreatedAt
		}
	}
	return latestVal, latestDate
}

func (m *MemoryRepository) ListAccountAssets(_ context.Context, accountID string) ([]*models.AccountAssetDetail, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var details []*models.AccountAssetDetail
	for _, item := range m.accountAssets {
		if item.AccountID != accountID {
			continue
		}
		asset, ok := m.assets[item.AssetID]
		if !ok {
			continue
		}
		val, date := m.getLatestAssetValue(item.AssetID)
		details = append(details, &models.AccountAssetDetail{
			ID:              item.ID,
			AccountID:       item.AccountID,
			AssetID:         item.AssetID,
			Label:           asset.Label,
			Description:     asset.Description,
			Quantity:        item.Quantity,
			LatestValue:     val,
			LatestValueDate: date,
			CreatedAt:       item.CreatedAt,
			UpdatedAt:       item.UpdatedAt,
		})
	}
	sort.Slice(details, func(i, j int) bool {
		return strings.ToLower(details[i].Label) < strings.ToLower(details[j].Label)
	})
	return details, nil
}

func (m *MemoryRepository) UpdateAccountAsset(_ context.Context, aa *models.AccountAsset) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.accountAssets[aa.ID]
	if !ok || existing.AccountID != aa.AccountID {
		return ErrNotFound
	}
	copied := *aa
	m.accountAssets[aa.ID] = &copied
	return nil
}

func (m *MemoryRepository) DeleteAccountAsset(_ context.Context, accountID, identifier string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	item := m.findAccountAsset(accountID, identifier)
	if item == nil {
		return ErrNotFound
	}
	delete(m.accountAssets, item.ID)
	return nil
}

func (m *MemoryRepository) hasAccountSnapshotConflict(s *models.AccountSnapshot) bool {
	for _, item := range m.accountSnapshots {
		if item.ID == s.ID {
			continue
		}
		if item.AccountID == s.AccountID && item.SnapshotDate == s.SnapshotDate {
			return true
		}
	}
	return false
}

func (m *MemoryRepository) CreateAccountSnapshot(_ context.Context, s *models.AccountSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.accounts[s.AccountID]; !ok {
		return ErrNotFound
	}
	if m.hasAccountSnapshotConflict(s) {
		return ErrConflict
	}
	copied := *s
	m.accountSnapshots[s.ID] = &copied
	return nil
}

func (m *MemoryRepository) GetAccountSnapshotByID(_ context.Context, accountID, snapshotID string) (*models.AccountSnapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, ok := m.accountSnapshots[snapshotID]
	if !ok || item.AccountID != accountID {
		return nil, ErrNotFound
	}
	copied := *item
	return &copied, nil
}

func matchesSnapshot(s *models.AccountSnapshot, accountID string, startDate, endDate *string) bool {
	if s.AccountID != accountID {
		return false
	}
	if startDate != nil && *startDate != "" && s.SnapshotDate < *startDate {
		return false
	}
	if endDate != nil && *endDate != "" && s.SnapshotDate > *endDate {
		return false
	}
	return true
}

func (m *MemoryRepository) ListAccountSnapshots(_ context.Context, accountID string, startDate, endDate *string) ([]*models.AccountSnapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*models.AccountSnapshot
	for _, s := range m.accountSnapshots {
		if matchesSnapshot(s, accountID, startDate, endDate) {
			copied := *s
			result = append(result, &copied)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].SnapshotDate == result[j].SnapshotDate {
			return result[i].CreatedAt.Before(result[j].CreatedAt)
		}
		return result[i].SnapshotDate < result[j].SnapshotDate
	})
	return result, nil
}

func (m *MemoryRepository) UpdateAccountSnapshot(_ context.Context, s *models.AccountSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.accountSnapshots[s.ID]
	if !ok || existing.AccountID != s.AccountID {
		return ErrNotFound
	}
	if m.hasAccountSnapshotConflict(s) {
		return ErrConflict
	}
	copied := *s
	m.accountSnapshots[s.ID] = &copied
	return nil
}

func (m *MemoryRepository) DeleteAccountSnapshot(_ context.Context, accountID, snapshotID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.accountSnapshots[snapshotID]
	if !ok || existing.AccountID != accountID {
		return ErrNotFound
	}
	delete(m.accountSnapshots, snapshotID)
	return nil
}
