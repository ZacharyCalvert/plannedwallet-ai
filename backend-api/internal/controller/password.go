package controller

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/trace"
)

func (c *Controller) validatePasswordChangeInput(req models.ChangePasswordRequest) error {
	if err := c.verifyCaptcha(req.CaptchaID, req.CaptchaSolution); err != nil {
		return err
	}
	if req.NewPassword != req.NewPasswordConfirm {
		return NewAppError(http.StatusBadRequest, "PASSWORD_MISMATCH", "New password and confirmation do not match.")
	}
	return ValidatePassword(req.NewPassword)
}

// ChangePassword verifies JWT target user, CAPTCHA, old password, and TOTP code before updating password_hash.
func (c *Controller) ChangePassword(ctx context.Context, userID, targetUsername string, req models.ChangePasswordRequest) (*models.ChangePasswordResponse, error) {
	user, err := c.verifyAuthenticatedTargetUser(ctx, userID, targetUsername, "Users can only change their own password.")
	if err != nil {
		return nil, err
	}
	if err := c.validatePasswordChangeInput(req); err != nil {
		trace.LogError(ctx, err, "Password change validation failed for %s", user.Username)
		return nil, err
	}
	if !c.auth.CheckPassword(user.PasswordHash, req.OldPassword) {
		trace.LogError(ctx, nil, "Invalid old password on password change for user: %s", user.Username)
		return nil, ErrInvalidCredentials
	}
	if !auth.ValidateCode(user.TOTPSecret, strings.TrimSpace(req.TOTPCode), time.Now()) {
		trace.LogError(ctx, nil, "Invalid TOTP code on password change for user: %s", user.Username)
		return nil, ErrInvalidTOTP
	}
	newHash, err := c.auth.HashPassword(req.NewPassword)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to hash new password.")
	}
	if err := c.repo.UpdateUserPassword(ctx, user.ID, newHash); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update password.")
	}
	_ = c.repo.RevokeMobileTokensByUserID(ctx, user.ID, time.Now().UTC())
	trace.LogInfo(ctx, "Password updated successfully for user: %s", user.Username)
	return &models.ChangePasswordResponse{
		Message:   "Password updated successfully.",
		UpdatedAt: time.Now().UTC(),
	}, nil
}
