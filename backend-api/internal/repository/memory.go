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
	assetStores     map[string]*models.AssetStore     // id -> AssetStore
	assetStoreRecords map[string]*models.AssetStoreRecord // id -> AssetStoreRecord
	accountSnapshots map[string]*models.AccountSnapshot // id -> AccountSnapshot
	captchas        map[string]*memoryCaptcha         // id -> memoryCaptcha
}

type memoryCaptcha struct {
	digits    []byte
	createdAt time.Time
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
		assetStores:     make(map[string]*models.AssetStore),
		assetStoreRecords: make(map[string]*models.AssetStoreRecord),
		accountSnapshots: make(map[string]*models.AccountSnapshot),
		captchas:        make(map[string]*memoryCaptcha),
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
		if item.AccountID == aa.AccountID && item.AssetID == aa.AssetID && item.AssetDate == aa.AssetDate {
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
	if aa.AssetDate == "" {
		aa.AssetDate = time.Now().UTC().Format("2006-01-02")
	}
	if m.hasAccountAssetConflict(aa) {
		return ErrConflict
	}
	copied := *aa
	m.accountAssets[aa.ID] = &copied
	return nil
}

func (m *MemoryRepository) findAccountAsset(accountID, identifier string) *models.AccountAsset {
	var latest *models.AccountAsset
	for _, item := range m.accountAssets {
		if item.AccountID != accountID {
			continue
		}
		if item.ID == identifier {
			return item
		}
		if item.AssetID == identifier {
			if latest == nil || item.AssetDate > latest.AssetDate || (item.AssetDate == latest.AssetDate && item.CreatedAt.After(latest.CreatedAt)) {
				latest = item
			}
		}
	}
	return latest
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

func (m *MemoryRepository) getLatestAssetValue(assetID, asOfDate string) (float64, string) {
	var latestVal float64
	var latestDate string
	var latestCreated time.Time

	var asOfVal float64
	var asOfDateStr string
	var asOfCreated time.Time

	for _, v := range m.assetValues {
		if v.AssetID != assetID {
			continue
		}
		if v.ValueDate > latestDate || (v.ValueDate == latestDate && v.CreatedAt.After(latestCreated)) {
			latestDate = v.ValueDate
			latestVal = v.Value
			latestCreated = v.CreatedAt
		}
		if asOfDate != "" && v.ValueDate <= asOfDate {
			if v.ValueDate > asOfDateStr || (v.ValueDate == asOfDateStr && v.CreatedAt.After(asOfCreated)) {
				asOfDateStr = v.ValueDate
				asOfVal = v.Value
				asOfCreated = v.CreatedAt
			}
		}
	}
	if asOfDateStr != "" {
		return asOfVal, asOfDateStr
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
		val, date := m.getLatestAssetValue(item.AssetID, item.AssetDate)
		details = append(details, &models.AccountAssetDetail{
			ID:              item.ID,
			AccountID:       item.AccountID,
			AssetID:         item.AssetID,
			Label:           asset.Label,
			Description:     asset.Description,
			Quantity:        item.Quantity,
			AssetDate:       item.AssetDate,
			LatestValue:     val,
			LatestValueDate: date,
			CreatedAt:       item.CreatedAt,
			UpdatedAt:       item.UpdatedAt,
		})
	}
	sort.Slice(details, func(i, j int) bool {
		if details[i].AssetDate != details[j].AssetDate {
			return details[i].AssetDate > details[j].AssetDate
		}
		if !details[i].CreatedAt.Equal(details[j].CreatedAt) {
			return details[i].CreatedAt.After(details[j].CreatedAt)
		}
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
	if m.hasAccountAssetConflict(aa) {
		return ErrConflict
	}
	copied := *aa
	m.accountAssets[aa.ID] = &copied
	return nil
}

func (m *MemoryRepository) DeleteAccountAsset(_ context.Context, accountID, identifier string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	deleted := false
	for id, item := range m.accountAssets {
		if item.AccountID == accountID && (item.ID == identifier || item.AssetID == identifier) {
			delete(m.accountAssets, id)
			deleted = true
		}
	}
	if !deleted {
		return ErrNotFound
	}
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

func (m *MemoryRepository) SetCaptcha(_ context.Context, id string, digits []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	copied := make([]byte, len(digits))
	copy(copied, digits)
	m.captchas[id] = &memoryCaptcha{
		digits:    copied,
		createdAt: time.Now().UTC(),
	}
	return nil
}

func (m *MemoryRepository) GetCaptcha(_ context.Context, id string, clear bool) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, exists := m.captchas[id]
	if !exists || time.Since(entry.createdAt) > 5*time.Minute {
		if exists {
			delete(m.captchas, id)
		}
		return nil, nil
	}
	copied := make([]byte, len(entry.digits))
	copy(copied, entry.digits)
	if clear {
		delete(m.captchas, id)
	}
	return copied, nil
}

func (m *MemoryRepository) DeleteExpiredCaptchas(_ context.Context, olderThan time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var count int64
	cutoff := time.Now().UTC().Add(-olderThan)
	for id, entry := range m.captchas {
		if entry.createdAt.Before(cutoff) {
			delete(m.captchas, id)
			count++
		}
	}
	return count, nil
}

func (m *MemoryRepository) findLatestStoreRecordLocked(storeID string) (float64, string) {
	var bestDate string
	var bestCreated time.Time
	var qty float64
	for _, r := range m.assetStoreRecords {
		if r.AssetStoreID == storeID {
			if r.RecordDate > bestDate || (r.RecordDate == bestDate && r.CreatedAt.After(bestCreated)) {
				bestDate = r.RecordDate
				bestCreated = r.CreatedAt
				qty = r.Quantity
			}
		}
	}
	return qty, bestDate
}

func (m *MemoryRepository) findLatestAssetValueLocked(assetID string) (float64, string) {
	var bestDate string
	var bestCreated time.Time
	var val float64
	for _, v := range m.assetValues {
		if v.AssetID == assetID {
			if v.ValueDate > bestDate || (v.ValueDate == bestDate && v.CreatedAt.After(bestCreated)) {
				bestDate = v.ValueDate
				bestCreated = v.CreatedAt
				val = v.Value
			}
		}
	}
	return val, bestDate
}

func (m *MemoryRepository) buildStoreDetailLocked(s *models.AssetStore) *models.AssetStoreDetail {
	d := &models.AssetStoreDetail{
		ID:          s.ID,
		AccountID:   s.AccountID,
		AssetID:     s.AssetID,
		Label:       s.Label,
		Description: s.Description,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
	if a, ok := m.assets[s.AssetID]; ok {
		d.AssetLabel = a.Label
		d.AssetDescription = a.Description
	}
	d.LatestQuantity, d.LatestRecordDate = m.findLatestStoreRecordLocked(s.ID)
	d.LatestValue, d.LatestValueDate = m.findLatestAssetValueLocked(s.AssetID)
	d.TotalValue = d.LatestQuantity * d.LatestValue
	return d
}

func (m *MemoryRepository) CreateAssetStore(_ context.Context, store *models.AssetStore) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	lowerLabel := strings.ToLower(strings.TrimSpace(store.Label))
	for _, existing := range m.assetStores {
		if existing.AccountID == store.AccountID && strings.ToLower(existing.Label) == lowerLabel {
			return ErrConflict
		}
	}
	copied := *store
	m.assetStores[store.ID] = &copied
	return nil
}

func (m *MemoryRepository) GetAssetStoreByID(_ context.Context, accountID, storeID string) (*models.AssetStoreDetail, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, exists := m.assetStores[storeID]
	if !exists || s.AccountID != accountID {
		return nil, ErrNotFound
	}
	return m.buildStoreDetailLocked(s), nil
}

func (m *MemoryRepository) GetAssetStoreByAccountAndLabel(_ context.Context, accountID, label string) (*models.AssetStore, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	lowerLabel := strings.ToLower(strings.TrimSpace(label))
	for _, s := range m.assetStores {
		if s.AccountID == accountID && strings.ToLower(s.Label) == lowerLabel {
			copied := *s
			return &copied, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryRepository) ListAssetStores(_ context.Context, accountID string) ([]*models.AssetStoreDetail, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*models.AssetStoreDetail
	for _, s := range m.assetStores {
		if s.AccountID == accountID {
			list = append(list, m.buildStoreDetailLocked(s))
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Label < list[j].Label
	})
	return list, nil
}

func (m *MemoryRepository) UpdateAssetStore(_ context.Context, store *models.AssetStore) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.assetStores[store.ID]
	if !exists || existing.AccountID != store.AccountID {
		return ErrNotFound
	}
	lowerLabel := strings.ToLower(strings.TrimSpace(store.Label))
	for _, other := range m.assetStores {
		if other.ID != store.ID && other.AccountID == store.AccountID && strings.ToLower(other.Label) == lowerLabel {
			return ErrConflict
		}
	}
	existing.Label = store.Label
	existing.Description = store.Description
	if store.AssetID != "" {
		existing.AssetID = store.AssetID
	}
	existing.UpdatedAt = store.UpdatedAt
	return nil
}

func (m *MemoryRepository) DeleteAssetStore(_ context.Context, accountID, storeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.assetStores[storeID]
	if !exists || existing.AccountID != accountID {
		return ErrNotFound
	}
	delete(m.assetStores, storeID)
	for rID, r := range m.assetStoreRecords {
		if r.AssetStoreID == storeID {
			delete(m.assetStoreRecords, rID)
		}
	}
	return nil
}

func (m *MemoryRepository) CreateAssetStoreRecord(_ context.Context, rec *models.AssetStoreRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, existing := range m.assetStoreRecords {
		if existing.AssetStoreID == rec.AssetStoreID && existing.RecordDate == rec.RecordDate {
			return ErrConflict
		}
	}
	copied := *rec
	m.assetStoreRecords[rec.ID] = &copied
	return nil
}

func (m *MemoryRepository) GetAssetStoreRecordByID(_ context.Context, storeID, recordID string) (*models.AssetStoreRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	r, exists := m.assetStoreRecords[recordID]
	if !exists || r.AssetStoreID != storeID {
		return nil, ErrNotFound
	}
	copied := *r
	return &copied, nil
}

func (m *MemoryRepository) ListAssetStoreRecords(_ context.Context, storeID string) ([]*models.AssetStoreRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*models.AssetStoreRecord
	for _, r := range m.assetStoreRecords {
		if r.AssetStoreID == storeID {
			copied := *r
			list = append(list, &copied)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].RecordDate != list[j].RecordDate {
			return list[i].RecordDate > list[j].RecordDate
		}
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})
	return list, nil
}

func (m *MemoryRepository) UpdateAssetStoreRecord(_ context.Context, rec *models.AssetStoreRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.assetStoreRecords[rec.ID]
	if !exists || existing.AssetStoreID != rec.AssetStoreID {
		return ErrNotFound
	}
	for _, other := range m.assetStoreRecords {
		if other.ID != rec.ID && other.AssetStoreID == rec.AssetStoreID && other.RecordDate == rec.RecordDate {
			return ErrConflict
		}
	}
	existing.Quantity = rec.Quantity
	existing.RecordDate = rec.RecordDate
	existing.UpdatedAt = rec.UpdatedAt
	return nil
}

func (m *MemoryRepository) DeleteAssetStoreRecord(_ context.Context, storeID, recordID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.assetStoreRecords[recordID]
	if !exists || existing.AssetStoreID != storeID {
		return ErrNotFound
	}
	delete(m.assetStoreRecords, recordID)
	return nil
}

func (m *MemoryRepository) collectStoreTrendDatesLocked(storeID, assetID string) []string {
	dateSet := make(map[string]bool)
	for _, r := range m.assetStoreRecords {
		if r.AssetStoreID == storeID {
			dateSet[r.RecordDate] = true
		}
	}
	for _, v := range m.assetValues {
		if v.AssetID == assetID {
			dateSet[v.ValueDate] = true
		}
	}
	var dates []string
	for d := range dateSet {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	return dates
}

func (m *MemoryRepository) getStoreQtyAtDateLocked(storeID, dateStr string) float64 {
	var bestDate string
	var bestCreated time.Time
	var qty float64
	for _, r := range m.assetStoreRecords {
		if r.AssetStoreID == storeID && r.RecordDate <= dateStr {
			if r.RecordDate > bestDate || (r.RecordDate == bestDate && r.CreatedAt.After(bestCreated)) {
				bestDate = r.RecordDate
				bestCreated = r.CreatedAt
				qty = r.Quantity
			}
		}
	}
	return qty
}

func (m *MemoryRepository) getAssetValAtDateLocked(assetID, dateStr string) float64 {
	var bestDate string
	var bestCreated time.Time
	var val float64
	for _, v := range m.assetValues {
		if v.AssetID == assetID && v.ValueDate <= dateStr {
			if v.ValueDate > bestDate || (v.ValueDate == bestDate && v.CreatedAt.After(bestCreated)) {
				bestDate = v.ValueDate
				bestCreated = v.CreatedAt
				val = v.Value
			}
		}
	}
	return val
}

func (m *MemoryRepository) GetAssetStoreTrend(_ context.Context, storeID string) ([]models.AssetStoreTrendPoint, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	store, exists := m.assetStores[storeID]
	if !exists {
		return nil, ErrNotFound
	}
	dates := m.collectStoreTrendDatesLocked(storeID, store.AssetID)
	var points []models.AssetStoreTrendPoint
	for _, d := range dates {
		q := m.getStoreQtyAtDateLocked(storeID, d)
		if q <= 0 {
			continue
		}
		p := m.getAssetValAtDateLocked(store.AssetID, d)
		points = append(points, models.AssetStoreTrendPoint{
			Date:       d,
			Quantity:   q,
			UnitValue:  p,
			TotalValue: q * p,
		})
	}
	return points, nil
}
