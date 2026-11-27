package controller_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/controller"
	"github.com/plannedwallet/internal/models"
)

func TestChangePassword_SuccessAndOldPasswordRejected(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()

	u, err := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "pw_alice",
		Email:    "pw_alice@example.com",
		Password: "InitialPassword123!",
	})
	if err != nil {
		t.Fatalf("RegisterUser failed: %v", err)
	}

	code, err := auth.GenerateCode(u.TOTPSecret, time.Now().Unix()/30)
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}

	res, err := ctrl.ChangePassword(ctx, u.ID, u.Username, models.ChangePasswordRequest{
		OldPassword:        "InitialPassword123!",
		NewPassword:        "UpdatedPassword456!",
		NewPasswordConfirm: "UpdatedPassword456!",
		TOTPCode:           code,
	})
	if err != nil {
		t.Fatalf("ChangePassword failed: %v", err)
	}
	if res.Message == "" || res.UpdatedAt.IsZero() {
		t.Fatalf("expected valid ChangePasswordResponse, got %+v", res)
	}

	if _, err := ctrl.ExchangePasswordForToken(ctx, u.Username, "InitialPassword123!", code); err == nil {
		t.Fatal("expected old password to be rejected after ChangePassword")
	}
	if _, err := ctrl.ExchangePasswordForToken(ctx, u.Username, "UpdatedPassword456!", code); err != nil {
		t.Fatalf("expected new password to succeed after ChangePassword, got %v", err)
	}
}

func TestChangePassword_NegativeCasesKeepPasswordUnchanged(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()

	u1, err := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "pw_owner",
		Email:    "pw_owner@example.com",
		Password: "InitialPassword123!",
	})
	if err != nil {
		t.Fatalf("RegisterUser u1 failed: %v", err)
	}
	u2, err := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "pw_other",
		Email:    "pw_other@example.com",
		Password: "OtherPassword123!",
	})
	if err != nil {
		t.Fatalf("RegisterUser u2 failed: %v", err)
	}

	validCode, _ := auth.GenerateCode(u1.TOTPSecret, time.Now().Unix()/30)
	wrongCode := "000000"
	if validCode == wrongCode {
		wrongCode = "999999"
	}

	// Wrong target user
	_, err = ctrl.ChangePassword(ctx, u2.ID, u1.Username, models.ChangePasswordRequest{
		OldPassword:        "InitialPassword123!",
		NewPassword:        "UpdatedPassword456!",
		NewPasswordConfirm: "UpdatedPassword456!",
		TOTPCode:           validCode,
	})
	assertAppErrorStatus(t, err, http.StatusForbidden)

	// Confirmation mismatch
	_, err = ctrl.ChangePassword(ctx, u1.ID, u1.Username, models.ChangePasswordRequest{
		OldPassword:        "InitialPassword123!",
		NewPassword:        "UpdatedPassword456!",
		NewPasswordConfirm: "DifferentPassword789!",
		TOTPCode:           validCode,
	})
	assertAppErrorStatus(t, err, http.StatusBadRequest)

	// Weak new password
	_, err = ctrl.ChangePassword(ctx, u1.ID, u1.Username, models.ChangePasswordRequest{
		OldPassword:        "InitialPassword123!",
		NewPassword:        "short",
		NewPasswordConfirm: "short",
		TOTPCode:           validCode,
	})
	assertAppErrorStatus(t, err, http.StatusBadRequest)

	// Wrong old password
	_, err = ctrl.ChangePassword(ctx, u1.ID, u1.Username, models.ChangePasswordRequest{
		OldPassword:        "WrongOldPassword123!",
		NewPassword:        "UpdatedPassword456!",
		NewPasswordConfirm: "UpdatedPassword456!",
		TOTPCode:           validCode,
	})
	assertAppErrorStatus(t, err, http.StatusUnauthorized)

	// Wrong TOTP code
	_, err = ctrl.ChangePassword(ctx, u1.ID, u1.Username, models.ChangePasswordRequest{
		OldPassword:        "InitialPassword123!",
		NewPassword:        "UpdatedPassword456!",
		NewPasswordConfirm: "UpdatedPassword456!",
		TOTPCode:           wrongCode,
	})
	assertAppErrorStatus(t, err, http.StatusUnauthorized)

	// Original password still works
	if _, err := ctrl.ExchangePasswordForToken(ctx, u1.Username, "InitialPassword123!", validCode); err != nil {
		t.Fatalf("expected original password to remain active, got %v", err)
	}
}

func TestChangePassword_CaptchaEnforcement(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()

	u, err := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "pw_cap",
		Email:    "pw_cap@example.com",
		Password: "InitialPassword123!",
	})
	if err != nil {
		t.Fatalf("RegisterUser failed: %v", err)
	}
	validCode, _ := auth.GenerateCode(u.TOTPSecret, time.Now().Unix()/30)

	ctrl.SetRequireCaptcha(true)

	_, err = ctrl.ChangePassword(ctx, u.ID, u.Username, models.ChangePasswordRequest{
		OldPassword:        "InitialPassword123!",
		NewPassword:        "UpdatedPassword456!",
		NewPasswordConfirm: "UpdatedPassword456!",
		TOTPCode:           validCode,
	})
	var appErr *controller.AppError
	if !errors.As(err, &appErr) || appErr.Code != "CAPTCHA_REQUIRED" {
		t.Fatalf("expected CAPTCHA_REQUIRED, got %v", err)
	}

	_, err = ctrl.ChangePassword(ctx, u.ID, u.Username, models.ChangePasswordRequest{
		OldPassword:        "InitialPassword123!",
		NewPassword:        "UpdatedPassword456!",
		NewPasswordConfirm: "UpdatedPassword456!",
		TOTPCode:           validCode,
		CaptchaID:          "bad-id",
		CaptchaSolution:    "wrong",
	})
	if !errors.As(err, &appErr) || appErr.Code != "INVALID_CAPTCHA" {
		t.Fatalf("expected INVALID_CAPTCHA, got %v", err)
	}
}
