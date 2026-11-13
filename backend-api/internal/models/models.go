package models

import "time"

// Valid account types as defined in schema.yml
var ValidAccountTypes = map[string]bool{
	"checking":    true,
	"savings":     true,
	"credit card": true,
	"credit_card": true,
	"asset":       true,
	"assets":      true,
	"snapshot":    true,
}

// User domain model
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	TOTPSecret   string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// UserRegistrationRequest represents the payload to register a new user.
type UserRegistrationRequest struct {
	Username        string `json:"username" binding:"required,min=3,max=50"`
	Email           string `json:"email" binding:"required,email,max=255"`
	Password        string `json:"password" binding:"required,min=10"`
	CaptchaID       string `json:"captcha_id,omitempty"`
	CaptchaSolution string `json:"captcha_solution,omitempty"`
}

// CaptchaResponse represents the challenge returned to the client.
type CaptchaResponse struct {
	ID    string `json:"id"`
	Image string `json:"image"`
}

// UserResponse represents the public user info returned after registration.
type UserResponse struct {
	ID         string    `json:"id"`
	Username   string    `json:"username"`
	Email      string    `json:"email"`
	TOTPSecret string    `json:"totp_secret,omitempty"`
	TOTPURL    string    `json:"totp_url,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// TokenExchangeRequest represents the body when exchanging credentials for a JWT.
type TokenExchangeRequest struct {
	Password string `json:"password" binding:"required"`
	TOTPCode string `json:"totp_code" binding:"required"`
}

// TokenResponse contains the issued JWT access token.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// AccessLevel represents the permission level granted to an account.
type AccessLevel string

const (
	AccessLevelRead  AccessLevel = "read"
	AccessLevelEdit  AccessLevel = "edit"
	AccessLevelAdmin AccessLevel = "admin"
)

var ValidAccessLevels = map[AccessLevel]bool{
	AccessLevelRead:  true,
	AccessLevelEdit:  true,
	AccessLevelAdmin: true,
}

// Account domain model
type Account struct {
	ID          string    `json:"id"`
	AdminID     string    `json:"admin_id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AccountAccess domain model representing access granted to an account.
type AccountAccess struct {
	ID          string      `json:"id"`
	AccountID   string      `json:"account_id"`
	UserID      string      `json:"user_id"`
	AccessLevel AccessLevel `json:"access_level"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// AccountCreationRequest represents the request to create a financial account.
type AccountCreationRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Type        string `json:"type" binding:"required"`
	Description string `json:"description" binding:"required,min=1,max=255"`
}

// AccountResponse represents the account returned by the API.
type AccountResponse struct {
	ID          string    `json:"id"`
	AdminID     string    `json:"admin_id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AccountAccessResponse represents the access grant returned by the API.
type AccountAccessResponse struct {
	ID          string      `json:"id"`
	AccountID   string      `json:"account_id"`
	UserID      string      `json:"user_id"`
	AccessLevel AccessLevel `json:"access_level"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// AccountAccessGrantRequest represents the request to grant or update user access to an account.
type AccountAccessGrantRequest struct {
	UserID      string      `json:"user_id" binding:"required"`
	AccessLevel AccessLevel `json:"access_level" binding:"required"`
}

// CheckingRecord domain model
type CheckingRecord struct {
	ID          string    `json:"id"`
	AccountID   string    `json:"account_id"`
	Value       float64   `json:"value"`
	DateSent    string    `json:"date_sent"`
	DateCleared *string   `json:"date_cleared"`
	Entity      string    `json:"entity"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CheckingRecordCreateRequest represents payload to create a checking record.
type CheckingRecordCreateRequest struct {
	Value       *float64 `json:"value" binding:"required"`
	DateSent    string   `json:"date_sent" binding:"required"`
	DateCleared *string  `json:"date_cleared"`
	Entity      string   `json:"entity" binding:"required,min=1,max=255"`
	Description string   `json:"description" binding:"max=500"`
}

// CheckingRecordUpdateRequest represents payload to fully update a checking record.
type CheckingRecordUpdateRequest struct {
	Value       *float64 `json:"value" binding:"required"`
	DateSent    string   `json:"date_sent" binding:"required"`
	DateCleared *string  `json:"date_cleared"`
	Entity      string   `json:"entity" binding:"required,min=1,max=255"`
	Description string   `json:"description" binding:"max=500"`
}

// CheckingRecordPatchRequest represents payload to partially update a checking record.
type CheckingRecordPatchRequest struct {
	Value       *float64 `json:"value,omitempty"`
	DateSent    *string  `json:"date_sent,omitempty"`
	DateCleared *string  `json:"date_cleared,omitempty"`
	Entity      *string  `json:"entity,omitempty"`
	Description *string  `json:"description,omitempty"`
}

// CheckingRecordResponse represents the checking record returned by the API.
type CheckingRecordResponse struct {
	ID          string    `json:"id"`
	AccountID   string    `json:"account_id"`
	Value       float64   `json:"value"`
	DateSent    string    `json:"date_sent"`
	DateCleared *string   `json:"date_cleared"`
	Entity      string    `json:"entity"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Asset represents a tracked financial asset definition (e.g. btc, eth, silver, gold, tx-gas).
type Asset struct {
	ID          string    `json:"id"`
	UserID      *string   `json:"user_id"`
	Label       string    `json:"label"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AssetCreateRequest represents payload to define a new asset.
type AssetCreateRequest struct {
	Label       string `json:"label" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=200"`
}

// AssetUpdateRequest represents payload to update an existing asset definition.
type AssetUpdateRequest struct {
	Label       string `json:"label" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=200"`
}

// AssetResponse represents the asset definition returned by the API.
type AssetResponse struct {
	ID          string    `json:"id"`
	UserID      *string   `json:"user_id"`
	Label       string    `json:"label"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AssetValue represents a recorded USD valuation of an asset on a particular date.
type AssetValue struct {
	ID        string    `json:"id"`
	AssetID   string    `json:"asset_id"`
	Value     float64   `json:"value"`
	ValueDate string    `json:"value_date"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AssetValueCreateRequest represents payload to record an asset value on a date.
type AssetValueCreateRequest struct {
	Value     *float64 `json:"value" binding:"required"`
	ValueDate string   `json:"value_date" binding:"required"`
}

// AssetValueUpdateRequest represents payload to update an asset valuation.
type AssetValueUpdateRequest struct {
	Value     *float64 `json:"value" binding:"required"`
	ValueDate string   `json:"value_date" binding:"required"`
}

// AssetValueResponse represents the valuation record returned by the API.
type AssetValueResponse struct {
	ID        string    `json:"id"`
	AssetID   string    `json:"asset_id"`
	Value     float64   `json:"value"`
	ValueDate string    `json:"value_date"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AccountAsset represents an asset holding assigned to an asset account.
type AccountAsset struct {
	ID        string    `json:"id"`
	AccountID string    `json:"account_id"`
	AssetID   string    `json:"asset_id"`
	Quantity  float64   `json:"quantity"`
	AssetDate string    `json:"asset_date"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AccountAssetDetail contains holding info enriched with asset metadata and latest valuation.
type AccountAssetDetail struct {
	ID              string
	AccountID       string
	AssetID         string
	Label           string
	Description     string
	Quantity        float64
	AssetDate       string
	LatestValue     float64
	LatestValueDate string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// AccountAssetAddRequest represents payload to add an asset with quantity to an account.
type AccountAssetAddRequest struct {
	AssetID   string   `json:"asset_id" binding:"required"`
	Quantity  *float64 `json:"quantity" binding:"required"`
	AssetDate string   `json:"asset_date"`
}

// AccountAssetUpdateRequest represents payload to update the quantity of an asset in an account.
type AccountAssetUpdateRequest struct {
	Quantity  *float64 `json:"quantity"`
	AssetDate string   `json:"asset_date"`
}

// AccountAssetResponse represents an asset holding returned by the API with calculated values.
type AccountAssetResponse struct {
	ID              string    `json:"id"`
	AccountID       string    `json:"account_id"`
	AssetID         string    `json:"asset_id"`
	Label           string    `json:"label"`
	Description     string    `json:"description"`
	Quantity        float64   `json:"quantity"`
	AssetDate       string    `json:"asset_date"`
	LatestValue     float64   `json:"latest_value"`
	LatestValueDate string    `json:"latest_value_date,omitempty"`
	TotalValue      float64   `json:"total_value"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// AssetAccountSummaryResponse represents the account summary with total portfolio value.
type AssetAccountSummaryResponse struct {
	AccountID  string                  `json:"account_id"`
	TotalValue float64                 `json:"total_value"`
	Assets     []*AccountAssetResponse `json:"assets"`
}

// AssetAccountTrendPoint represents a point-in-time total valuation of an asset account.
type AssetAccountTrendPoint struct {
	Date       string  `json:"date"`
	TotalValue float64 `json:"total_value"`
}

// AccountSnapshot represents a point-in-time valuation for a snapshot account.
type AccountSnapshot struct {
	ID           string    `json:"id"`
	AccountID    string    `json:"account_id"`
	Value        float64   `json:"value"`
	SnapshotDate string    `json:"snapshot_date"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AccountSnapshotCreateRequest represents payload to create a snapshot valuation.
type AccountSnapshotCreateRequest struct {
	Value        *float64 `json:"value" binding:"required"`
	SnapshotDate string   `json:"snapshot_date" binding:"required"`
}

// AccountSnapshotUpdateRequest represents payload to update a snapshot valuation.
type AccountSnapshotUpdateRequest struct {
	Value        *float64 `json:"value" binding:"required"`
	SnapshotDate *string  `json:"snapshot_date,omitempty"`
}

// AccountSnapshotResponse represents a snapshot valuation returned by the API.
type AccountSnapshotResponse struct {
	ID           string    `json:"id"`
	AccountID    string    `json:"account_id"`
	Value        float64   `json:"value"`
	SnapshotDate string    `json:"snapshot_date"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ErrorResponse represents standardized API error responses.
type ErrorResponse struct {
	Error     string    `json:"error"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}
