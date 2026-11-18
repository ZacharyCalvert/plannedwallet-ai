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

// Subscription domain model representing a user's subscription.
type Subscription struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	PaidDate       string    `json:"paid_date"`
	SubscriberEnds string    `json:"subscriber_ends"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
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

// AccountPatchRequest represents the payload to patch an account's name and/or description.
type AccountPatchRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
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

// Category represents a transaction category (global if UserID is nil, or user-scoped).
type Category struct {
	ID        string    `json:"id"`
	UserID    *string   `json:"user_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CategoryCreateRequest represents payload to create a user-scoped category.
type CategoryCreateRequest struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
}

// CategoryResponse represents a category returned by the API.
type CategoryResponse struct {
	ID        string    `json:"id"`
	UserID    *string   `json:"user_id"`
	Name      string    `json:"name"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CategoryExpenseSlice represents a single category's expense total for pie charts.
type CategoryExpenseSlice struct {
	CategoryID   *string `json:"category_id"`
	CategoryName string  `json:"category_name"`
	TotalAmount  float64 `json:"total_amount"`
	RecordCount  int     `json:"record_count"`
}

// AccountCategorySummaryResponse represents an account's expense breakdown by category.
type AccountCategorySummaryResponse struct {
	AccountID     string                 `json:"account_id"`
	TotalExpenses float64                `json:"total_expenses"`
	Categories    []CategoryExpenseSlice `json:"categories"`
}

// CheckingRecord domain model
type CheckingRecord struct {
	ID                string    `json:"id"`
	AccountID         string    `json:"account_id"`
	CategoryID        *string   `json:"category_id"`
	ReconcileReportID *string   `json:"reconcile_report_id"`
	Value             float64   `json:"value"`
	DateSent          string    `json:"date_sent"`
	DateCleared       *string   `json:"date_cleared"`
	Entity            string    `json:"entity"`
	Description       string    `json:"description"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// CheckingRecordCreateRequest represents payload to create a checking record.
type CheckingRecordCreateRequest struct {
	Value       *float64 `json:"value" binding:"required"`
	DateSent    string   `json:"date_sent" binding:"required"`
	DateCleared *string  `json:"date_cleared"`
	CategoryID  *string  `json:"category_id"`
	Entity      string   `json:"entity" binding:"required,min=1,max=255"`
	Description string   `json:"description" binding:"max=500"`
}

// CheckingRecordUpdateRequest represents payload to fully update a checking record.
type CheckingRecordUpdateRequest struct {
	Value       *float64 `json:"value" binding:"required"`
	DateSent    string   `json:"date_sent" binding:"required"`
	DateCleared *string  `json:"date_cleared"`
	CategoryID  *string  `json:"category_id"`
	Entity      string   `json:"entity" binding:"required,min=1,max=255"`
	Description string   `json:"description" binding:"max=500"`
}

// CheckingRecordPatchRequest represents payload to partially update a checking record.
type CheckingRecordPatchRequest struct {
	Value       *float64 `json:"value,omitempty"`
	DateSent    *string  `json:"date_sent,omitempty"`
	DateCleared *string  `json:"date_cleared,omitempty"`
	CategoryID  *string  `json:"category_id,omitempty"`
	Entity      *string  `json:"entity,omitempty"`
	Description *string  `json:"description,omitempty"`
}

// CheckingRecordResponse represents the checking record returned by the API.
type CheckingRecordResponse struct {
	ID                string    `json:"id"`
	AccountID         string    `json:"account_id"`
	CategoryID        *string   `json:"category_id"`
	ReconcileReportID *string   `json:"reconcile_report_id"`
	Value             float64   `json:"value"`
	DateSent          string    `json:"date_sent"`
	DateCleared       *string   `json:"date_cleared"`
	Entity            string    `json:"entity"`
	Description       string    `json:"description"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ReconcileReport domain model representing a completed account reconciliation.
type ReconcileReport struct {
	ID              string    `json:"id"`
	AccountID       string    `json:"account_id"`
	Name            string    `json:"name"`
	ReportDate      string    `json:"report_date"`
	StartingBalance float64   `json:"starting_balance"`
	EndingBalance   float64   `json:"ending_balance"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ReconcileReportCreateRequest represents payload to create a reconcile report.
type ReconcileReportCreateRequest struct {
	Name            string   `json:"name" binding:"max=100"`
	ReportDate      string   `json:"report_date" binding:"required"`
	StartingBalance *float64 `json:"starting_balance" binding:"required"`
	EndingBalance   *float64 `json:"ending_balance" binding:"required"`
	RecordIDs       []string `json:"record_ids" binding:"required"`
}

// ReconcileReportResponse represents a reconcile report returned by the API.
type ReconcileReportResponse struct {
	ID              string                   `json:"id"`
	AccountID       string                   `json:"account_id"`
	Name            string                   `json:"name"`
	ReportDate      string                   `json:"report_date"`
	StartingBalance float64                  `json:"starting_balance"`
	EndingBalance   float64                  `json:"ending_balance"`
	RecordCount     int                      `json:"record_count"`
	RecordIDs       []string                 `json:"record_ids"`
	Records         []CheckingRecordResponse `json:"records"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
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
	Stores     []*AssetStoreDetail     `json:"stores,omitempty"`
}

// AssetStore represents a labeled storage location for an asset class in an account.
type AssetStore struct {
	ID          string    `json:"id"`
	AccountID   string    `json:"account_id"`
	AssetID     string    `json:"asset_id"`
	Label       string    `json:"label"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AssetStoreDetail contains store info enriched with asset metadata and latest valuation/quantity.
type AssetStoreDetail struct {
	ID               string    `json:"id"`
	AccountID        string    `json:"account_id"`
	AssetID          string    `json:"asset_id"`
	Label            string    `json:"label"`
	Description      string    `json:"description"`
	AssetLabel       string    `json:"asset_label"`
	AssetDescription string    `json:"asset_description"`
	LatestQuantity   float64   `json:"latest_quantity"`
	LatestRecordDate string    `json:"latest_record_date,omitempty"`
	LatestValue      float64   `json:"latest_value"`
	LatestValueDate  string    `json:"latest_value_date,omitempty"`
	TotalValue       float64   `json:"total_value"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// CreateAssetStoreRequest payload to create a new asset store
type CreateAssetStoreRequest struct {
	AssetID     string   `json:"asset_id" binding:"required"`
	Label       string   `json:"label" binding:"required,min=1,max=100"`
	Description string   `json:"description"`
	Quantity    *float64 `json:"quantity,omitempty"`
	RecordDate  string   `json:"record_date,omitempty"`
}

// UpdateAssetStoreRequest payload to update an asset store
type UpdateAssetStoreRequest struct {
	Label       string  `json:"label" binding:"required,min=1,max=100"`
	Description string  `json:"description"`
	AssetID     *string `json:"asset_id,omitempty"`
}

// AssetStoreRecord represents a point-in-time quantity record for an asset store.
type AssetStoreRecord struct {
	ID           string    `json:"id"`
	AssetStoreID string    `json:"asset_store_id"`
	Quantity     float64   `json:"quantity"`
	RecordDate   string    `json:"record_date"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CreateStoreRecordRequest payload to create a new quantity record for a store
type CreateStoreRecordRequest struct {
	Quantity   *float64 `json:"quantity" binding:"required"`
	RecordDate string   `json:"record_date" binding:"required"`
}

// UpdateStoreRecordRequest payload to update a quantity record
type UpdateStoreRecordRequest struct {
	Quantity   *float64 `json:"quantity"`
	RecordDate *string  `json:"record_date,omitempty"`
}

// AssetStoreTrendPoint represents a point-in-time valuation point for a single store.
type AssetStoreTrendPoint struct {
	Date       string  `json:"date"`
	Quantity   float64 `json:"quantity"`
	UnitValue  float64 `json:"unit_value"`
	TotalValue float64 `json:"total_value"`
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

// Perspective domain model representing a multi-account aggregated viewing window.
type Perspective struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PerspectiveSettings domain model holding viewing window calculation and display preferences.
type PerspectiveSettings struct {
	ID                         string    `json:"id"`
	PerspectiveID              string    `json:"perspective_id"`
	ShowFinancialTrend         bool      `json:"show_financial_trend"`
	CalculateAverageDailySpend bool      `json:"calculate_average_daily_spend"`
	CalculateAverageReturnYoY  bool      `json:"calculate_average_return_yoy"`
	TrackRuinedDate            bool      `json:"track_ruined_date"`
	CustomDailySpend           *float64  `json:"custom_daily_spend,omitempty"`
	CreatedAt                  time.Time `json:"created_at"`
	UpdatedAt                  time.Time `json:"updated_at"`
}

// PerspectiveAccount domain model representing the link between a perspective and an account.
type PerspectiveAccount struct {
	ID            string    `json:"id"`
	PerspectiveID string    `json:"perspective_id"`
	AccountID     string    `json:"account_id"`
	CreatedAt     time.Time `json:"created_at"`
}

// PerspectiveAccess domain model representing user sharing and permission on a perspective.
type PerspectiveAccess struct {
	ID            string      `json:"id"`
	PerspectiveID string      `json:"perspective_id"`
	UserID        string      `json:"user_id"`
	AccessLevel   AccessLevel `json:"access_level"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

// PerspectiveCreateRequest represents the payload to create a financial perspective.
type PerspectiveCreateRequest struct {
	Name        string                    `json:"name" binding:"required,min=1,max=100"`
	Description string                    `json:"description" binding:"max=255"`
	AccountIDs  []string                  `json:"account_ids,omitempty"`
	Settings    *PerspectiveSettingsInput `json:"settings,omitempty"`
}

// PerspectiveUpdateRequest represents the payload to update perspective name/description.
type PerspectiveUpdateRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=255"`
}

// PerspectiveSettingsInput represents the configurable settings for a perspective.
type PerspectiveSettingsInput struct {
	ShowFinancialTrend         *bool    `json:"show_financial_trend,omitempty"`
	CalculateAverageDailySpend *bool    `json:"calculate_average_daily_spend,omitempty"`
	CalculateAverageReturnYoY  *bool    `json:"calculate_average_return_yoy,omitempty"`
	TrackRuinedDate            *bool    `json:"track_ruined_date,omitempty"`
	CustomDailySpend           *float64 `json:"custom_daily_spend,omitempty"`
}

// PerspectiveResponse represents a perspective returned by the API.
type PerspectiveResponse struct {
	ID          string               `json:"id"`
	UserID      string               `json:"user_id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	AccessLevel AccessLevel          `json:"access_level"`
	Settings    *PerspectiveSettings `json:"settings,omitempty"`
	AccountIDs  []string             `json:"account_ids,omitempty"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

// PerspectiveAccessGrantRequest represents the request to grant or update sharing permissions.
type PerspectiveAccessGrantRequest struct {
	UserID      string      `json:"user_id" binding:"required"`
	AccessLevel AccessLevel `json:"access_level" binding:"required"`
}

// PerspectiveAccessResponse represents access grant details returned by the API.
type PerspectiveAccessResponse struct {
	ID            string      `json:"id"`
	PerspectiveID string      `json:"perspective_id"`
	UserID        string      `json:"user_id"`
	AccessLevel   AccessLevel `json:"access_level"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

// PerspectiveAccountLinkRequest represents the request to link an account to a perspective.
type PerspectiveAccountLinkRequest struct {
	AccountID string `json:"account_id" binding:"required"`
}

// PerspectiveTrendPoint represents a point-in-time aggregate valuation across linked accounts.
type PerspectiveTrendPoint struct {
	Date          string             `json:"date"`
	TotalValue    float64            `json:"total_value"`
	AccountValues map[string]float64 `json:"account_values,omitempty"`
}

// PerspectiveViewResponse represents the comprehensive aggregated viewing window response.
type PerspectiveViewResponse struct {
	Perspective       *PerspectiveResponse    `json:"perspective"`
	Settings          *PerspectiveSettings    `json:"settings"`
	Accounts          []*AccountResponse      `json:"accounts"`
	CurrentTotalValue float64                 `json:"current_total_value"`
	Trend             []PerspectiveTrendPoint `json:"trend,omitempty"`
	AverageDailySpend *float64                `json:"average_daily_spend,omitempty"`
	AverageReturnYoY  *float64                `json:"average_return_yoy,omitempty"`
	RuinedDate        *string                 `json:"ruined_date,omitempty"`
	DaysUntilRuined   *int                    `json:"days_until_ruined,omitempty"`
}

// Feature flag constants exposed by GET /api/v1/features.
const (
	FeatureReadOnly = "read_only"
)

// FeaturesResponse represents active feature flags enabled on the backend.
type FeaturesResponse struct {
	Features []string `json:"features"`
}

// VersionResponse represents build provenance metadata stamped into the binary at compile time.
type VersionResponse struct {
	GitHubRun string `json:"github_run"`
	BuildDate string `json:"build_date"`
	GitSHA    string `json:"git_sha"`
}

// PolicyAgreement represents a ledger entry capturing user acceptance of Terms, Privacy Policy, and US residency.
type PolicyAgreement struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	TermsSHA256     string    `json:"terms_sha256"`
	TermsApproved   bool      `json:"terms_approved"`
	PrivacyApproved bool      `json:"privacy_approved"`
	USCustomer      bool      `json:"us_customer"`
	Active          bool      `json:"active"`
	ClientIP        string    `json:"client_ip"`
	AgreedAt        time.Time `json:"agreed_at"`
}

// PolicyAgreementRequest represents the payload submitted by a user approving Terms, Privacy Policy, and US consumer status.
type PolicyAgreementRequest struct {
	TermsApproved   *bool  `json:"terms_approved" binding:"required"`
	PrivacyApproved *bool  `json:"privacy_approved" binding:"required"`
	USCustomer      *bool  `json:"us_customer" binding:"required"`
	TermsSHA256     string `json:"terms_sha256" binding:"required"`
}

// PolicyAgreementResponse represents a recorded policy agreement returned by the API.
type PolicyAgreementResponse struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	TermsSHA256     string    `json:"terms_sha256"`
	TermsApproved   bool      `json:"terms_approved"`
	PrivacyApproved bool      `json:"privacy_approved"`
	USCustomer      bool      `json:"us_customer"`
	Active          bool      `json:"active"`
	ClientIP        string    `json:"client_ip"`
	AgreedAt        time.Time `json:"agreed_at"`
}


