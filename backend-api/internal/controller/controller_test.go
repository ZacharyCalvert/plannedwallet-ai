package controller_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/controller"
	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/repository"
)

func setupController() (*controller.Controller, repository.Repository) {
	repo := repository.NewMemoryRepository()
	authService := auth.NewService("test-secret-123456", 3600)
	return controller.NewController(repo, authService), repo
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		pw      string
		wantErr bool
	}{
		{"Valid complex password", "SecurePass123!", false},
		{"Length < 10", "Pass123!", true},
		{"No uppercase", "securepass123!", true},
		{"No lowercase", "SECUREPASS123!", true},
		{"No number", "SecurePass!!!!", true},
		{"No special character", "SecurePass1234", true},
		{"Valid with unicode special", "P@ssw0rd2026", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := controller.ValidatePassword(tt.pw)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword(%q) error = %v, wantErr %v", tt.pw, err, tt.wantErr)
			}
		})
	}
}

func TestRegisterUser_PreValidationAndRules(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()

	// 1. Success
	userRes, err := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "valid_user",
		Email:    "valid@example.com",
		Password: "ValidPassword123!",
	})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if userRes.Username != "valid_user" || userRes.Email != "valid@example.com" {
		t.Errorf("unexpected user response: %+v", userRes)
	}

	// 2. Conflict on existing username
	_, err = ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "valid_user",
		Email:    "another@example.com",
		Password: "ValidPassword123!",
	})
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	var appErr *controller.AppError
	if !errors.As(err, &appErr) || appErr.Status != http.StatusConflict {
		t.Errorf("expected 409 conflict, got %v", err)
	}

	// 3. Conflict on existing email
	_, err = ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "another_user",
		Email:    "valid@example.com",
		Password: "ValidPassword123!",
	})
	if err == nil {
		t.Fatal("expected conflict error on email, got nil")
	}
	if !errors.As(err, &appErr) || appErr.Status != http.StatusConflict {
		t.Errorf("expected 409 conflict, got %v", err)
	}

	// 4. Password rule failure
	_, err = ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "weak_user",
		Email:    "weak@example.com",
		Password: "weakpassword",
	})
	if err == nil {
		t.Fatal("expected password error, got nil")
	}
	if !errors.As(err, &appErr) || appErr.Status != http.StatusBadRequest {
		t.Errorf("expected 400 bad request, got %v", err)
	}
}

func TestAccountAndCheckingRecordWorkflow(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()

	userRes, err := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "finance_user",
		Email:    "finance@example.com",
		Password: "StrongPassword99$",
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// 1. Create account
	accRes, err := ctrl.CreateAccount(ctx, userRes.ID, models.AccountCreationRequest{
		Name:        "Personal Checking",
		Type:        "checking",
		Description: "Day to day expenses",
	})
	if err != nil {
		t.Fatalf("account creation failed: %v", err)
	}
	if accRes.Name != "Personal Checking" || accRes.Type != "checking" || accRes.AdminID != userRes.ID {
		t.Errorf("unexpected account data: %+v", accRes)
	}

	// 2. Create checking record
	val := -45.50
	recRes, err := ctrl.CreateCheckingRecord(ctx, userRes.ID, accRes.ID, models.CheckingRecordCreateRequest{
		Value:       &val,
		DateSent:    "2026-11-10",
		Entity:      "Supermarket",
		Description: "Groceries",
	})
	if err != nil {
		t.Fatalf("record creation failed: %v", err)
	}
	if recRes.Value != -45.50 || recRes.Entity != "Supermarket" {
		t.Errorf("unexpected record data: %+v", recRes)
	}

	// 3. Unauthorized access by different user to this checking record
	_, err = ctrl.GetCheckingRecord(ctx, "different-user-id", accRes.ID, recRes.ID)
	if err == nil {
		t.Fatal("expected unauthorized/account not found for different user")
	}
	var appErr *controller.AppError
	if !errors.As(err, &appErr) || appErr.Status != http.StatusNotFound {
		t.Errorf("expected 404 for different user account access, got %v", err)
	}

	// 4. Update checking record
	updatedVal := -50.00
	clearedDate := "2026-11-12"
	updatedRec, err := ctrl.UpdateCheckingRecord(ctx, userRes.ID, accRes.ID, recRes.ID, models.CheckingRecordUpdateRequest{
		Value:       &updatedVal,
		DateSent:    "2026-11-10",
		DateCleared: &clearedDate,
		Entity:      "Supermarket Plus",
		Description: "Groceries and snacks",
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updatedRec.Value != -50.00 || updatedRec.Entity != "Supermarket Plus" {
		t.Errorf("unexpected updated record: %+v", updatedRec)
	}

	// 5. Delete checking record
	if err := ctrl.DeleteCheckingRecord(ctx, userRes.ID, accRes.ID, recRes.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// 6. Verify deleted
	_, err = ctrl.GetCheckingRecord(ctx, userRes.ID, accRes.ID, recRes.ID)
	if err == nil {
		t.Fatal("expected error getting deleted record, got nil")
	}
}

func TestAccountAccessPermissions(t *testing.T) {
	ctrl, repo := setupController()
	ctx := context.Background()

	adminUser, err := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "admin_owner",
		Email:    "admin_owner@example.com",
		Password: "Password123!",
	})
	if err != nil {
		t.Fatalf("failed to register admin: %v", err)
	}

	secondUser, err := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "collaborator",
		Email:    "collab@example.com",
		Password: "Password123!",
	})
	if err != nil {
		t.Fatalf("failed to register collaborator: %v", err)
	}

	acc, err := ctrl.CreateAccount(ctx, adminUser.ID, models.AccountCreationRequest{
		Name:        "Shared Checking",
		Type:        "checking",
		Description: "Joint checking account",
	})
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}
	if acc.AdminID != adminUser.ID {
		t.Errorf("expected AdminID to match adminUser.ID, got %s", acc.AdminID)
	}

	// 1. Before granting access, collaborator cannot access account records
	val := -10.00
	_, err = ctrl.CreateCheckingRecord(ctx, secondUser.ID, acc.ID, models.CheckingRecordCreateRequest{
		Value:    &val,
		DateSent: "2026-11-10",
		Entity:   "Coffee Shop",
	})
	if err == nil {
		t.Fatal("expected error for unauthorized user creating record, got nil")
	}

	// 2. Admin grants 'edit' access to collaborator
	grantRes, err := ctrl.GrantAccountAccess(ctx, adminUser.ID, acc.ID, secondUser.ID, models.AccessLevelEdit)
	if err != nil {
		t.Fatalf("failed to grant access: %v", err)
	}
	if grantRes.AccessLevel != models.AccessLevelEdit || grantRes.UserID != secondUser.ID {
		t.Errorf("unexpected grant result: %+v", grantRes)
	}

	// 3. Collaborator can now create and view checking records
	rec, err := ctrl.CreateCheckingRecord(ctx, secondUser.ID, acc.ID, models.CheckingRecordCreateRequest{
		Value:    &val,
		DateSent: "2026-11-10",
		Entity:   "Coffee Shop",
	})
	if err != nil {
		t.Fatalf("expected collaborator with edit access to succeed, got %v", err)
	}

	gotRec, err := ctrl.GetCheckingRecord(ctx, secondUser.ID, acc.ID, rec.ID)
	if err != nil || gotRec.ID != rec.ID {
		t.Fatalf("collaborator failed to get record: %v", err)
	}

	// 4. Verify accounts list for collaborator includes this shared account
	collabAccounts, err := repo.ListAccountsByUserID(ctx, secondUser.ID)
	if err != nil {
		t.Fatalf("failed to list collaborator accounts: %v", err)
	}
	if len(collabAccounts) != 1 || collabAccounts[0].ID != acc.ID {
		t.Errorf("expected 1 shared account for collaborator, got %d", len(collabAccounts))
	}

	// 5. Admin revokes collaborator access
	if err := ctrl.RevokeAccountAccess(ctx, adminUser.ID, acc.ID, secondUser.ID); err != nil {
		t.Fatalf("failed to revoke access: %v", err)
	}

	// 6. After revocation, collaborator access is denied
	_, err = ctrl.GetCheckingRecord(ctx, secondUser.ID, acc.ID, rec.ID)
	if err == nil {
		t.Fatal("expected access denied after revocation, got nil")
	}
}
