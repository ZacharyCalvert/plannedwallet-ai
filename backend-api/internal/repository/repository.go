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
	// User operations
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, id string) (*models.User, error)

	// Account operations
	CreateAccount(ctx context.Context, account *models.Account) error
	GetAccountByID(ctx context.Context, id string) (*models.Account, error)
	ListAccountsByUserID(ctx context.Context, userID string) ([]*models.Account, error)

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
}
