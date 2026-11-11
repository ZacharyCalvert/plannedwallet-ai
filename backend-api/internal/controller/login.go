package controller

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/repository"
	"github.com/plannedwallet/internal/trace"
)

// ExchangeToken validates user credentials including TOTP and issues a signed JWT token.
func (c *Controller) ExchangeToken(ctx context.Context, username, password, totpCode string) (*models.TokenResponse, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, NewAppError(http.StatusBadRequest, "BAD_REQUEST", "Username is required.")
	}

	user, err := c.repo.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			trace.LogError(ctx, err, "User not found for login: %s", username)
			return nil, ErrUserNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Error retrieving user.")
	}

	if !c.auth.CheckPassword(user.PasswordHash, password) {
		trace.LogError(ctx, nil, "Invalid password for user: %s", username)
		return nil, ErrInvalidCredentials
	}

	if !auth.ValidateCode(user.TOTPSecret, strings.TrimSpace(totpCode), time.Now()) {
		trace.LogError(ctx, nil, "Invalid TOTP code for user: %s", username)
		return nil, ErrInvalidTOTP
	}

	token, err := c.auth.GenerateToken(user.ID, user.Username)
	if err != nil {
		trace.LogError(ctx, err, "Token generation failed for user: %s", username)
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate token.")
	}

	trace.LogInfo(ctx, "Token exchange succeeded for user: %s", username)
	return &models.TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   c.auth.ExpirationSeconds(),
	}, nil
}
