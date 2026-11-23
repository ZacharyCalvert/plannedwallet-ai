package controller

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/captcha"
	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/repository"
	"github.com/plannedwallet/internal/trace"
)

func classifyPasswordChars(password string) (upper, lower, digit, special bool) {
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsLower(r):
			lower = true
		case unicode.IsDigit(r):
			digit = true
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			special = true
		}
	}
	return upper, lower, digit, special
}

func validatePasswordComplexity(password string) error {
	upper, lower, digit, special := classifyPasswordChars(password)
	if !upper {
		return NewAppError(http.StatusBadRequest, "INVALID_PASSWORD", "Password must contain at least one uppercase letter.")
	}
	if !lower {
		return NewAppError(http.StatusBadRequest, "INVALID_PASSWORD", "Password must contain at least one lowercase letter.")
	}
	if !digit {
		return NewAppError(http.StatusBadRequest, "INVALID_PASSWORD", "Password must contain at least one number.")
	}
	if !special {
		return NewAppError(http.StatusBadRequest, "INVALID_PASSWORD", "Password must contain at least one special character not in the alphanumeric set.")
	}
	return nil
}

// ValidatePassword enforces password rules:
// 10-72 bytes, at least 1 uppercase, 1 lowercase, 1 number, and 1 non-alphanumeric special character.
func ValidatePassword(password string) error {
	if len(password) < 10 {
		return NewAppError(http.StatusBadRequest, "INVALID_PASSWORD", "Password must be at least 10 characters long.")
	}
	if len(password) > 72 {
		return NewAppError(http.StatusBadRequest, "INVALID_PASSWORD", "Password must not exceed 72 bytes.")
	}
	return validatePasswordComplexity(password)
}

func (c *Controller) validateUserUniqueness(ctx context.Context, username, email string) error {
	if existing, err := c.repo.GetUserByUsername(ctx, username); err == nil && existing != nil {
		return ErrUserAlreadyExists
	}
	if existing, err := c.repo.GetUserByEmail(ctx, email); err == nil && existing != nil {
		return ErrUserAlreadyExists
	}
	return nil
}

func (c *Controller) persistNewUserWithRecovery(ctx context.Context, user *models.User, key *models.UserRecoveryKey) error {
	if err := c.repo.CreateUserWithRecoveryKey(ctx, user, key); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return ErrUserAlreadyExists
		}
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create user.")
	}
	return nil
}

func (c *Controller) buildNewUser(username, email, password string) (*models.User, error) {
	hash, err := c.auth.HashPassword(password)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to hash password.")
	}
	totpSecret, err := auth.GenerateTOTPSecret()
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate TOTP secret.")
	}
	return &models.User{
		ID:           uuid.New().String(),
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		TOTPSecret:   totpSecret,
		CreatedAt:    time.Now().UTC(),
	}, nil
}

func (c *Controller) verifyCaptcha(id, solution string) error {
	id = strings.TrimSpace(id)
	solution = strings.TrimSpace(solution)
	if id == "" || solution == "" {
		if c.requireCaptcha {
			return ErrCaptchaRequired
		}
		return nil
	}
	if !captcha.Verify(id, solution) {
		return ErrInvalidCaptcha
	}
	return nil
}

func (c *Controller) validateRegistrationInput(ctx context.Context, req models.UserRegistrationRequest) error {
	if err := c.verifyCaptcha(req.CaptchaID, req.CaptchaSolution); err != nil {
		return err
	}
	if err := ValidatePassword(req.Password); err != nil {
		return err
	}
	return c.validateUserUniqueness(ctx, strings.TrimSpace(req.Username), strings.TrimSpace(req.Email))
}

func buildUserResponse(user *models.User, recoveryPhrase, recoveryHint string) *models.UserResponse {
	return &models.UserResponse{
		ID:              user.ID,
		Username:        user.Username,
		Email:           user.Email,
		TOTPSecret:      user.TOTPSecret,
		TOTPURL:         auth.BuildTOTPURL("PlannedWallet", user.Username, user.TOTPSecret),
		RecoveryPhrase:  recoveryPhrase,
		RecoveryKeyHint: recoveryHint,
		CreatedAt:       user.CreatedAt,
	}
}

func (c *Controller) grantFreeTrialSubscription(ctx context.Context, user *models.User) error {
	if c.subscriberFreeDays <= 0 {
		return nil
	}
	createdUTC := user.CreatedAt.UTC()
	sub := &models.Subscription{
		ID:             uuid.New().String(),
		UserID:         user.ID,
		PaidDate:       createdUTC.Format("2006-01-02"),
		SubscriberEnds: createdUTC.AddDate(0, 0, c.subscriberFreeDays).Format("2006-01-02"),
		CreatedAt:      createdUTC,
		UpdatedAt:      createdUTC,
	}
	if err := c.repo.CreateSubscription(ctx, sub); err != nil {
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create free trial subscription.")
	}
	trace.LogInfo(ctx, "Granted %d-day free subscription to new user: %s (paid_date=%s, ends=%s)", c.subscriberFreeDays, user.Username, sub.PaidDate, sub.SubscriberEnds)
	return nil
}

// RegisterUser validates user existence, verifies password rules, generates TOTP & recovery credentials, and persists user.
func (c *Controller) RegisterUser(ctx context.Context, req models.UserRegistrationRequest) (*models.UserResponse, error) {
	username, email := strings.TrimSpace(req.Username), strings.TrimSpace(req.Email)
	trace.LogInfo(ctx, "Registering user: %s (email: %s)", username, email)
	if err := c.validateRegistrationInput(ctx, req); err != nil {
		trace.LogError(ctx, err, "User registration validation failed for %s", username)
		return nil, err
	}
	user, err := c.buildNewUser(username, email, req.Password)
	if err != nil {
		trace.LogError(ctx, err, "Failed to build user %s", username)
		return nil, err
	}
	phrase, recoveryKey, err := generateHashedRecoveryKey(user.ID, user.CreatedAt)
	if err != nil {
		trace.LogError(ctx, err, "Failed to generate recovery key for %s", username)
		return nil, err
	}
	if err := c.persistNewUserWithRecovery(ctx, user, recoveryKey); err != nil {
		trace.LogError(ctx, err, "Failed to persist user %s", username)
		return nil, err
	}
	if err := c.grantFreeTrialSubscription(ctx, user); err != nil {
		trace.LogError(ctx, err, "Failed to grant free trial subscription for %s", username)
		return nil, err
	}
	trace.LogInfo(ctx, "User registered successfully: %s (id: %s)", user.Username, user.ID)
	return buildUserResponse(user, phrase, recoveryKey.KeyHint), nil
}

// GetUser returns the signed-in user's profile details without sensitive secrets.
func (c *Controller) GetUser(ctx context.Context, userID, targetUsername string) (*models.UserResponse, error) {
	user, err := c.verifyAuthenticatedTargetUser(ctx, userID, targetUsername, "Users can only view their own profile.")
	if err != nil {
		return nil, err
	}
	hint := ""
	if rk, rkErr := c.repo.GetActiveRecoveryKeyByUserID(ctx, user.ID); rkErr == nil && rk != nil {
		hint = rk.KeyHint
	}
	return &models.UserResponse{
		ID:              user.ID,
		Username:        user.Username,
		Email:           user.Email,
		RecoveryKeyHint: hint,
		CreatedAt:       user.CreatedAt,
	}, nil
}

// DeleteUser permanently deletes a signed-in user and cascades all owned accounts and records.
func (c *Controller) DeleteUser(ctx context.Context, userID, targetUsername string) error {
	user, err := c.verifyAuthenticatedTargetUser(ctx, userID, targetUsername, "Users can only delete their own account.")
	if err != nil {
		return err
	}
	if err := c.repo.DeleteUser(ctx, user.ID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrUserNotFound
		}
		trace.LogError(ctx, err, "Failed to delete user %s (id: %s)", user.Username, user.ID)
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete user.")
	}
	trace.LogInfo(ctx, "User deleted permanently: %s (id: %s)", user.Username, user.ID)
	return nil
}
