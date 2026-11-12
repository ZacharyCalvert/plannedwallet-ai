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

// ValidatePassword enforces password rules:
// minimum length of 10 characters, at least 1 uppercase, 1 lowercase, 1 number, and 1 non-alphanumeric special character.
func ValidatePassword(password string) error {
	if len(password) < 10 {
		return NewAppError(http.StatusBadRequest, "INVALID_PASSWORD", "Password must be at least 10 characters long.")
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return NewAppError(http.StatusBadRequest, "INVALID_PASSWORD", "Password must contain at least one uppercase letter.")
	}
	if !hasLower {
		return NewAppError(http.StatusBadRequest, "INVALID_PASSWORD", "Password must contain at least one lowercase letter.")
	}
	if !hasDigit {
		return NewAppError(http.StatusBadRequest, "INVALID_PASSWORD", "Password must contain at least one number.")
	}
	if !hasSpecial {
		return NewAppError(http.StatusBadRequest, "INVALID_PASSWORD", "Password must contain at least one special character not in the alphanumeric set.")
	}

	return nil
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

func (c *Controller) persistNewUser(ctx context.Context, user *models.User) error {
	if err := c.repo.CreateUser(ctx, user); err != nil {
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

func buildUserResponse(user *models.User) *models.UserResponse {
	return &models.UserResponse{
		ID:         user.ID,
		Username:   user.Username,
		Email:      user.Email,
		TOTPSecret: user.TOTPSecret,
		TOTPURL:    auth.BuildTOTPURL("PlannedWallet", user.Username, user.TOTPSecret),
		CreatedAt:  user.CreatedAt,
	}
}

// RegisterUser validates user existence, verifies password rules, generates TOTP secret, and persists user.
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
	if err := c.persistNewUser(ctx, user); err != nil {
		trace.LogError(ctx, err, "Failed to persist user %s", username)
		return nil, err
	}
	trace.LogInfo(ctx, "User registered successfully: %s (id: %s)", user.Username, user.ID)
	return buildUserResponse(user), nil
}
