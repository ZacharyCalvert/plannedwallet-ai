package repository

import (
	"context"
	"errors"
	"time"

	"github.com/plannedwallet/internal/models"
)

var (
	ErrNotFound = errors.New("resource not found")
	ErrConflict = errors.New("resource conflict")
)

type Repository interface {
	// Health check
	Ping(ctx context.Context) error

	// User operations
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, id string) (*models.User, error)

	// Account operations
	CreateAccount(ctx context.Context, account *models.Account) error
	GetAccountByID(ctx context.Context, id string) (*models.Account, error)
	ListAccountsByUserID(ctx context.Context, userID string) ([]*models.Account, error)
	DeleteAccount(ctx context.Context, accountID, adminID string) error

	// Account Access operations
	GrantAccountAccess(ctx context.Context, access *models.AccountAccess) error
	GetAccountAccess(ctx context.Context, accountID, userID string) (*models.AccountAccess, error)
	ListAccountAccess(ctx context.Context, accountID string) ([]*models.AccountAccess, error)
	RevokeAccountAccess(ctx context.Context, accountID, userID string) error

	// Checking Record operations
	CreateCheckingRecord(ctx context.Context, record *models.CheckingRecord) error
	GetCheckingRecordByID(ctx context.Context, accountID, recordID string) (*models.CheckingRecord, error)
	ListCheckingRecords(ctx context.Context, accountID string, cleared *bool, startDate, endDate *string) ([]*models.CheckingRecord, error)
	UpdateCheckingRecord(ctx context.Context, record *models.CheckingRecord) error
	DeleteCheckingRecord(ctx context.Context, accountID, recordID string) error

	// Reconcile Report operations
	CreateReconcileReport(ctx context.Context, report *models.ReconcileReport, recordIDs []string) error
	GetReconcileReportByID(ctx context.Context, accountID, reportID string) (*models.ReconcileReport, []*models.CheckingRecord, error)
	ListReconcileReports(ctx context.Context, accountID string) ([]*models.ReconcileReport, map[string][]*models.CheckingRecord, error)
	DeleteReconcileReport(ctx context.Context, accountID, reportID string) error

	// Category operations
	CreateCategory(ctx context.Context, category *models.Category) error
	GetCategoryByID(ctx context.Context, id string) (*models.Category, error)
	ListCategoriesForUser(ctx context.Context, userID string) ([]*models.Category, error)
	DeleteCategory(ctx context.Context, id string) error

	// Asset operations
	CreateAsset(ctx context.Context, asset *models.Asset) error
	GetAssetByID(ctx context.Context, id string) (*models.Asset, error)
	ListAssets(ctx context.Context, userID, label *string) ([]*models.Asset, error)
	UpdateAsset(ctx context.Context, asset *models.Asset) error
	DeleteAsset(ctx context.Context, id string) error

	// Asset Value operations
	CreateAssetValue(ctx context.Context, av *models.AssetValue) error
	GetAssetValueByID(ctx context.Context, assetID, valueID string) (*models.AssetValue, error)
	ListAssetValues(ctx context.Context, assetID string, startDate, endDate *string) ([]*models.AssetValue, error)
	UpdateAssetValue(ctx context.Context, av *models.AssetValue) error
	DeleteAssetValue(ctx context.Context, assetID, valueID string) error

	// Account Asset operations
	AddAccountAsset(ctx context.Context, aa *models.AccountAsset) error
	GetAccountAsset(ctx context.Context, accountID, identifier string) (*models.AccountAsset, error)
	ListAccountAssets(ctx context.Context, accountID string) ([]*models.AccountAssetDetail, error)
	UpdateAccountAsset(ctx context.Context, aa *models.AccountAsset) error
	DeleteAccountAsset(ctx context.Context, accountID, identifier string) error

	// Asset Store operations
	CreateAssetStore(ctx context.Context, store *models.AssetStore) error
	GetAssetStoreByID(ctx context.Context, accountID, storeID string) (*models.AssetStoreDetail, error)
	GetAssetStoreByAccountAndLabel(ctx context.Context, accountID, label string) (*models.AssetStore, error)
	ListAssetStores(ctx context.Context, accountID string) ([]*models.AssetStoreDetail, error)
	UpdateAssetStore(ctx context.Context, store *models.AssetStore) error
	DeleteAssetStore(ctx context.Context, accountID, storeID string) error

	// Asset Store Record operations
	CreateAssetStoreRecord(ctx context.Context, rec *models.AssetStoreRecord) error
	GetAssetStoreRecordByID(ctx context.Context, storeID, recordID string) (*models.AssetStoreRecord, error)
	ListAssetStoreRecords(ctx context.Context, storeID string) ([]*models.AssetStoreRecord, error)
	UpdateAssetStoreRecord(ctx context.Context, rec *models.AssetStoreRecord) error
	DeleteAssetStoreRecord(ctx context.Context, storeID, recordID string) error
	GetAssetStoreTrend(ctx context.Context, storeID string) ([]models.AssetStoreTrendPoint, error)

	// Account Snapshot operations
	CreateAccountSnapshot(ctx context.Context, snapshot *models.AccountSnapshot) error
	GetAccountSnapshotByID(ctx context.Context, accountID, snapshotID string) (*models.AccountSnapshot, error)
	ListAccountSnapshots(ctx context.Context, accountID string, startDate, endDate *string) ([]*models.AccountSnapshot, error)
	UpdateAccountSnapshot(ctx context.Context, snapshot *models.AccountSnapshot) error
	DeleteAccountSnapshot(ctx context.Context, accountID, snapshotID string) error

	// CAPTCHA operations
	SetCaptcha(ctx context.Context, id string, digits []byte) error
	GetCaptcha(ctx context.Context, id string, clear bool) ([]byte, error)
	DeleteExpiredCaptchas(ctx context.Context, olderThan time.Duration) (int64, error)

	// Subscription operations
	CreateSubscription(ctx context.Context, sub *models.Subscription) error
	GetSubscriptionByID(ctx context.Context, id string) (*models.Subscription, error)
	ListSubscriptionsByUserID(ctx context.Context, userID string) ([]*models.Subscription, error)
	IsUserSubscriber(ctx context.Context, userID string) (bool, error)
	DeleteSubscription(ctx context.Context, id string) error

	// Perspective operations
	CreatePerspective(ctx context.Context, p *models.Perspective) error
	GetPerspectiveByID(ctx context.Context, id string) (*models.Perspective, error)
	ListPerspectivesByUserID(ctx context.Context, userID string) ([]*models.Perspective, error)
	UpdatePerspective(ctx context.Context, p *models.Perspective) error
	DeletePerspective(ctx context.Context, id string) error

	// Perspective Settings operations
	GetPerspectiveSettings(ctx context.Context, perspectiveID string) (*models.PerspectiveSettings, error)
	UpsertPerspectiveSettings(ctx context.Context, settings *models.PerspectiveSettings) error

	// Perspective Accounts operations
	AddPerspectiveAccount(ctx context.Context, pa *models.PerspectiveAccount) error
	RemovePerspectiveAccount(ctx context.Context, perspectiveID, accountID string) error
	ListPerspectiveAccountIDs(ctx context.Context, perspectiveID string) ([]string, error)

	// Perspective Access operations
	GrantPerspectiveAccess(ctx context.Context, access *models.PerspectiveAccess) error
	GetPerspectiveAccess(ctx context.Context, perspectiveID, userID string) (*models.PerspectiveAccess, error)
	ListPerspectiveAccess(ctx context.Context, perspectiveID string) ([]*models.PerspectiveAccess, error)
	RevokePerspectiveAccess(ctx context.Context, perspectiveID, userID string) error
}
