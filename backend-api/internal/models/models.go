package models

import "time"

// Valid account types as defined in schema.yml
var ValidAccountTypes = map[string]bool{
	"checking":    true,
	"savings":     true,
	"credit card": true,
	"credit_card": true,
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
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=10"`
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

// ErrorResponse represents standardized API error responses.
type ErrorResponse struct {
	Error     string    `json:"error"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}
