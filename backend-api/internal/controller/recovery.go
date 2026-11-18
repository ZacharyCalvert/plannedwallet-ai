package controller

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/repository"
	"github.com/plannedwallet/internal/trace"
)

var ErrInvalidRecoveryCredential = NewAppError(
	http.StatusUnauthorized,
	"INVALID_RECOVERY_CREDENTIAL",
	"Invalid username or recovery phrase.",
)

func generateHashedRecoveryKey(userID string, now time.Time) (string, *models.UserRecoveryKey, error) {
	phrase, err := auth.GenerateRecoveryPhrase()
	if err != nil {
		return "", nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate recovery phrase.")
	}
	hash, err := auth.HashRecoveryPhrase(phrase)
	if err != nil {
		return "", nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to hash recovery phrase.")
	}
	return phrase, &models.UserRecoveryKey{
		ID:        uuid.New().String(),
		UserID:    userID,
		KeyHash:   hash,
		KeyHint:   auth.BuildRecoveryKeyHint(phrase),
		Active:    true,
		CreatedAt: now,
	}, nil
}

func (c *Controller) verifyRecoveryCandidate(ctx context.Context, username, candidate string) (*models.User, *models.UserRecoveryKey, error) {
	user, err := c.repo.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, ErrInvalidRecoveryCredential
		}
		return nil, nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Error retrieving user.")
	}
	activeKey, err := c.repo.GetActiveRecoveryKeyByUserID(ctx, user.ID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, ErrInvalidRecoveryCredential
		}
		return nil, nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Error retrieving recovery key.")
	}
	if !auth.VerifyRecoveryPhrase(activeKey.KeyHash, candidate) {
		return nil, nil, ErrInvalidRecoveryCredential
	}
	return user, activeKey, nil
}

func (c *Controller) buildResetCredentials(newPassword string) (string, string, error) {
	newPasswordHash, err := c.auth.HashPassword(newPassword)
	if err != nil {
		return "", "", NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to hash new password.")
	}
	newTOTPSecret, err := auth.GenerateTOTPSecret()
	if err != nil {
		return "", "", NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate new TOTP secret.")
	}
	return newPasswordHash, newTOTPSecret, nil
}

func (c *Controller) applyRecoveryReset(ctx context.Context, user *models.User, consumedKeyID, clientIP, newPassword string) (*models.AccountRecoveryResponse, error) {
	newPasswordHash, newTOTPSecret, err := c.buildResetCredentials(newPassword)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	nextPhrase, replacementKey, err := generateHashedRecoveryKey(user.ID, now)
	if err != nil {
		return nil, err
	}
	if err := c.repo.CompleteAccountRecovery(ctx, user.ID, newPasswordHash, newTOTPSecret, consumedKeyID, clientIP, now, replacementKey); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrInvalidRecoveryCredential
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to complete account recovery.")
	}
	trace.LogInfo(ctx, "Account recovery completed and recovery phrase rotated for user: %s", user.Username)
	return &models.AccountRecoveryResponse{
		Username:        user.Username,
		TOTPSecret:      newTOTPSecret,
		TOTPURL:         auth.BuildTOTPURL("PlannedWallet", user.Username, newTOTPSecret),
		RecoveryPhrase:  nextPhrase,
		RecoveryKeyHint: replacementKey.KeyHint,
	}, nil
}

// RecoverAccount verifies CAPTCHA and a single-use recovery phrase, resets password & TOTP, and rotates the recovery phrase.
func (c *Controller) RecoverAccount(ctx context.Context, targetUsername, clientIP string, req models.AccountRecoveryRequest) (*models.AccountRecoveryResponse, error) {
	username := strings.TrimSpace(targetUsername)
	if username == "" {
		return nil, NewAppError(http.StatusBadRequest, "BAD_REQUEST", "Username path parameter is required.")
	}
	if err := c.verifyCaptcha(req.CaptchaID, req.CaptchaSolution); err != nil {
		return nil, err
	}
	if err := ValidatePassword(req.NewPassword); err != nil {
		return nil, err
	}
	user, consumedKey, err := c.verifyRecoveryCandidate(ctx, username, req.RecoveryPhrase)
	if err != nil {
		trace.LogError(ctx, err, "Account recovery verification failed for %s", username)
		return nil, err
	}
	return c.applyRecoveryReset(ctx, user, consumedKey.ID, clientIP, req.NewPassword)
}

// RotateRecoveryKey authenticates a signed-in user and replaces their active recovery phrase.
func (c *Controller) RotateRecoveryKey(ctx context.Context, userID, targetUsername string, req models.RotateRecoveryKeyRequest) (*models.RotateRecoveryKeyResponse, error) {
	user, err := c.verifyAuthenticatedTargetUser(ctx, userID, targetUsername, "Users can only rotate their own recovery key.")
	if err != nil {
		return nil, err
	}
	if !c.auth.CheckPassword(user.PasswordHash, req.Password) {
		return nil, ErrInvalidCredentials
	}
	if !auth.ValidateCode(user.TOTPSecret, strings.TrimSpace(req.TOTPCode), time.Now()) {
		return nil, ErrInvalidTOTP
	}
	now := time.Now().UTC()
	nextPhrase, newKey, err := generateHashedRecoveryKey(user.ID, now)
	if err != nil {
		return nil, err
	}
	if err := c.repo.RotateRecoveryKey(ctx, user.ID, newKey); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to rotate recovery phrase.")
	}
	trace.LogInfo(ctx, "Recovery phrase rotated by user: %s", user.Username)
	return &models.RotateRecoveryKeyResponse{
		RecoveryPhrase:  nextPhrase,
		RecoveryKeyHint: newKey.KeyHint,
		CreatedAt:       now,
	}, nil
}
