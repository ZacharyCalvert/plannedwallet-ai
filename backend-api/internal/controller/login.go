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
			return nil, ErrUserNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Error retrieving user.")
	}

	if !c.auth.CheckPassword(user.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}

	if !auth.ValidateCode(user.TOTPSecret, strings.TrimSpace(totpCode), time.Now()) {
		return nil, ErrInvalidTOTP
	}

	token, err := c.auth.GenerateToken(user.ID, user.Username)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate token.")
	}

	return &models.TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   c.auth.ExpirationSeconds(),
	}, nil
}
