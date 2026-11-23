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

// authenticateUser verifies username, password, and TOTP code.
func (c *Controller) authenticateUser(ctx context.Context, username, password, totpCode string) (*models.User, error) {
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
	return user, nil
}

func (c *Controller) resolveSubscriberStatus(ctx context.Context, userID string) (bool, error) {
	if c.subscriberAlwaysOn {
		return true, nil
	}
	return c.repo.IsUserSubscriber(ctx, userID)
}

func (c *Controller) issueTokenForUser(ctx context.Context, user *models.User, action string) (*models.TokenResponse, error) {
	isSubscriber, err := c.resolveSubscriberStatus(ctx, user.ID)
	if err != nil {
		trace.LogError(ctx, err, "Error checking subscriber status for: %s", user.Username)
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Error checking subscriber status.")
	}
	termsApproved, err := c.resolveTokenTermsApproved(ctx, user.ID)
	if err != nil {
		trace.LogError(ctx, err, "Error checking policy agreement status for: %s", user.Username)
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Error checking policy agreement status.")
	}
	token, err := c.auth.GenerateToken(user.ID, user.Username, isSubscriber, termsApproved)
	if err != nil {
		trace.LogError(ctx, err, "Token generation failed for user: %s", user.Username)
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate token.")
	}
	trace.LogInfo(ctx, "%s succeeded for user: %s (subscriber=%t, terms_approved=%t)", action, user.Username, isSubscriber, termsApproved)
	return &models.TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   c.auth.ExpirationSeconds(),
	}, nil
}

// ExchangePasswordForToken validates user credentials including TOTP and issues a signed JWT token.
func (c *Controller) ExchangePasswordForToken(ctx context.Context, username, password, totpCode string) (*models.TokenResponse, error) {
	user, err := c.authenticateUser(ctx, username, password, totpCode)
	if err != nil {
		return nil, err
	}
	return c.issueTokenForUser(ctx, user, "Token exchange")
}

func (c *Controller) verifyAuthenticatedTargetUser(ctx context.Context, userID, targetUsername, forbiddenMsg string) (*models.User, error) {
	targetUsername = strings.TrimSpace(targetUsername)
	if targetUsername == "" {
		return nil, NewAppError(http.StatusBadRequest, "BAD_REQUEST", "Username path parameter is required.")
	}
	user, err := c.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Error retrieving user.")
	}
	if !strings.EqualFold(user.Username, targetUsername) {
		return nil, NewAppError(http.StatusForbidden, "FORBIDDEN", forbiddenMsg)
	}
	return user, nil
}

// RefreshToken exchanges an existing valid JWT for a freshly signed JWT for the same user.
func (c *Controller) RefreshToken(ctx context.Context, userID, targetUsername string) (*models.TokenResponse, error) {
	user, err := c.verifyAuthenticatedTargetUser(ctx, userID, targetUsername, "Cannot refresh token for another user.")
	if err != nil {
		return nil, err
	}
	return c.issueTokenForUser(ctx, user, "Token refresh")
}
