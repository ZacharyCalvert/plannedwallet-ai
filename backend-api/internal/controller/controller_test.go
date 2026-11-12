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
	if userRes.TOTPSecret == "" || userRes.TOTPURL == "" {
		t.Errorf("expected TOTP secret and URL, got %+v", userRes)
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

func TestExchangeToken_TOTP(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()

	userRes, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "totp_user",
		Email:    "totp@example.com",
		Password: "ValidPassword123!",
	})

	validCode, _ := auth.GenerateCode(userRes.TOTPSecret, time.Now().Unix()/30)

	tok, err := ctrl.ExchangeToken(ctx, "totp_user", "ValidPassword123!", validCode)
	if err != nil || tok.AccessToken == "" {
		t.Fatalf("expected successful token exchange, got %v", err)
	}

	_, err = ctrl.ExchangeToken(ctx, "totp_user", "ValidPassword123!", "000000")
	if err == nil {
		t.Fatal("expected error with invalid TOTP code")
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

func TestSavingsAccountWorkflow(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()

	userRes, err := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "savings_user",
		Email:    "savings@example.com",
		Password: "StrongPassword99$",
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	accRes, err := ctrl.CreateAccount(ctx, userRes.ID, models.AccountCreationRequest{
		Name:        "Emergency Savings",
		Type:        "savings",
		Description: "High-yield emergency savings fund",
	})
	if err != nil {
		t.Fatalf("savings account creation failed: %v", err)
	}
	if accRes.Type != "savings" {
		t.Errorf("expected type savings, got %s", accRes.Type)
	}

	val := 500.00
	recRes, err := ctrl.CreateCheckingRecord(ctx, userRes.ID, accRes.ID, models.CheckingRecordCreateRequest{
		Value:       &val,
		DateSent:    "2026-11-10",
		Entity:      "Interest Payment",
		Description: "Monthly savings interest",
	})
	if err != nil {
		t.Fatalf("record creation failed on savings account: %v", err)
	}
	if recRes.Value != 500.00 {
		t.Errorf("unexpected record value: %v", recRes.Value)
	}
}

func TestCreditCardAccountWorkflow(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()

	userRes, err := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "cc_user",
		Email:    "cc@example.com",
		Password: "StrongPassword99$",
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	accRes, err := ctrl.CreateAccount(ctx, userRes.ID, models.AccountCreationRequest{
		Name:        "Rewards Credit Card",
		Type:        "credit card",
		Description: "Everyday rewards credit card",
	})
	if err != nil {
		t.Fatalf("credit card account creation failed: %v", err)
	}
	if accRes.Type != "credit card" {
		t.Errorf("expected type 'credit card', got %s", accRes.Type)
	}

	val := -120.50
	recRes, err := ctrl.CreateCheckingRecord(ctx, userRes.ID, accRes.ID, models.CheckingRecordCreateRequest{
		Value:       &val,
		DateSent:    "2026-11-10",
		Entity:      "Electronics Store",
		Description: "Headphones purchase",
	})
	if err != nil {
		t.Fatalf("record creation failed on credit card account: %v", err)
	}
	if recRes.Value != -120.50 {
		t.Errorf("unexpected record value: %v", recRes.Value)
	}
}

func TestAssetCreateAndConflict(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()

	u1, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "asset_u1", Email: "u1@example.com", Password: "ValidPass123!",
	})
	u2, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "asset_u2", Email: "u2@example.com", Password: "ValidPass123!",
	})

	asset1, err := ctrl.CreateAsset(ctx, u1.ID, models.AssetCreateRequest{
		Label: "btc", Description: "Bitcoin crypto",
	})
	if err != nil || asset1.Label != "btc" {
		t.Fatalf("failed to create asset: %v", err)
	}

	// Conflict for same user and label (case-insensitive)
	_, err = ctrl.CreateAsset(ctx, u1.ID, models.AssetCreateRequest{
		Label: "BTC", Description: "Duplicate Bitcoin",
	})
	if err == nil {
		t.Fatal("expected conflict for duplicate asset label on same user, got nil")
	}

	// Different user can use the same label
	asset2, err := ctrl.CreateAsset(ctx, u2.ID, models.AssetCreateRequest{
		Label: "btc", Description: "U2 Bitcoin",
	})
	if err != nil || asset2.ID == asset1.ID {
		t.Fatalf("different user should be able to create asset with same label: %v", err)
	}
}

func TestAssetValuesMultiDay(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()

	u, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "multiday_u", Email: "multiday@example.com", Password: "ValidPass123!",
	})

	asset, err := ctrl.CreateAsset(ctx, u.ID, models.AssetCreateRequest{
		Label: "btc", Description: "Bitcoin crypto asset",
	})
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	days := []struct {
		date string
		val  float64
	}{
		{"2026-11-09", 90000.0},
		{"2026-11-10", 91000.0},
		{"2026-11-11", 92000.0},
	}
	for _, d := range days {
		v := d.val
		_, err := ctrl.CreateAssetValue(ctx, u.ID, asset.ID, models.AssetValueCreateRequest{
			Value: &v, ValueDate: d.date,
		})
		if err != nil {
			t.Fatalf("failed to set value for %s: %v", d.date, err)
		}
	}

	values, err := ctrl.ListAssetValues(ctx, u.ID, asset.ID, nil, nil)
	if err != nil || len(values) != 3 {
		t.Fatalf("expected 3 values for asset, got %d (%v)", len(values), err)
	}
}

func TestAssetValuesDateConflict(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()

	u, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "conflict_u", Email: "conflict@example.com", Password: "ValidPass123!",
	})
	asset, _ := ctrl.CreateAsset(ctx, u.ID, models.AssetCreateRequest{
		Label: "silver", Description: "Silver ounce",
	})

	val1, val2 := 32.50, 33.00
	_, err := ctrl.CreateAssetValue(ctx, u.ID, asset.ID, models.AssetValueCreateRequest{
		Value: &val1, ValueDate: "2026-11-10",
	})
	if err != nil {
		t.Fatalf("first value creation failed: %v", err)
	}

	// Duplicate date for same asset must fail
	_, err = ctrl.CreateAssetValue(ctx, u.ID, asset.ID, models.AssetValueCreateRequest{
		Value: &val2, ValueDate: "2026-11-10",
	})
	if err == nil {
		t.Fatal("expected conflict on duplicate date for asset, got nil")
	}
}

func TestAssetValueUpdateAndDelete(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()

	u, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "updel_u", Email: "updel@example.com", Password: "ValidPass123!",
	})
	asset, _ := ctrl.CreateAsset(ctx, u.ID, models.AssetCreateRequest{
		Label: "gold", Description: "Gold ounce",
	})

	val := 2700.0
	created, err := ctrl.CreateAssetValue(ctx, u.ID, asset.ID, models.AssetValueCreateRequest{
		Value: &val, ValueDate: "2026-11-09",
	})
	if err != nil {
		t.Fatalf("failed to create asset value: %v", err)
	}

	newVal := 2750.0
	updated, err := ctrl.UpdateAssetValue(ctx, u.ID, asset.ID, created.ID, models.AssetValueUpdateRequest{
		Value: &newVal, ValueDate: "2026-11-09",
	})
	if err != nil || updated.Value != 2750.0 {
		t.Fatalf("update failed: %v", err)
	}

	if err := ctrl.DeleteAssetValue(ctx, u.ID, asset.ID, created.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

func setupPreciousMetalsAccount(ctx context.Context, ctrl *controller.Controller) (string, string, string, string) {
	u, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "metals_owner", Email: "owner@metals.com", Password: "ValidPass123!",
	})
	acc, _ := ctrl.CreateAccount(ctx, u.ID, models.AccountCreationRequest{
		Name: "Precious Metals", Type: "asset", Description: "Metals portfolio",
	})
	slv, _ := ctrl.CreateAsset(ctx, u.ID, models.AssetCreateRequest{Label: "slv-oz", Description: "Silver"})
	gld, _ := ctrl.CreateAsset(ctx, u.ID, models.AssetCreateRequest{Label: "gld-oz", Description: "Gold"})

	valSlv, valGld := 32.0, 2700.0
	_, _ = ctrl.CreateAssetValue(ctx, u.ID, slv.ID, models.AssetValueCreateRequest{Value: &valSlv, ValueDate: "2026-11-10"})
	_, _ = ctrl.CreateAssetValue(ctx, u.ID, gld.ID, models.AssetValueCreateRequest{Value: &valGld, ValueDate: "2026-11-10"})
	return u.ID, acc.ID, slv.ID, gld.ID
}

func TestAssetAccountLifecycleAndValuation(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()
	uID, accID, slvID, gldID := setupPreciousMetalsAccount(ctx, ctrl)

	qSlv, qGld := 45.209, 2.5
	item1, err := ctrl.AddAccountAsset(ctx, uID, accID, models.AccountAssetAddRequest{AssetID: slvID, Quantity: &qSlv})
	if err != nil || item1.TotalValue != 45.209*32.0 {
		t.Fatalf("unexpected silver item: %+v, err: %v", item1, err)
	}
	item2, err := ctrl.AddAccountAsset(ctx, uID, accID, models.AccountAssetAddRequest{AssetID: gldID, Quantity: &qGld})
	if err != nil || item2.TotalValue != 2.5*2700.0 {
		t.Fatalf("unexpected gold item: %+v, err: %v", item2, err)
	}

	summary, err := ctrl.GetAssetAccountSummary(ctx, uID, accID)
	expectedTotal := 45.209*32.0 + 2.5*2700.0
	if err != nil || summary.TotalValue != expectedTotal {
		t.Fatalf("expected total %v, got %v (err: %v)", expectedTotal, summary.TotalValue, err)
	}

	newQ := 50.0
	updated, err := ctrl.UpdateAccountAsset(ctx, uID, accID, slvID, models.AccountAssetUpdateRequest{Quantity: &newQ})
	if err != nil || updated.Quantity != 50.0 || updated.TotalValue != 1600.0 {
		t.Fatalf("update failed: %+v (err: %v)", updated, err)
	}

	if err := ctrl.DeleteAccountAsset(ctx, uID, accID, gldID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

func TestAssetAccountAccessControl_Permissions(t *testing.T) {
	ctrl, repo := setupController()
	ctx := context.Background()
	_, accID, slvID, _ := setupPreciousMetalsAccount(ctx, ctrl)

	uEdit, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "editor_user", Email: "editor@test.com", Password: "ValidPass123!",
	})
	uRead, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "reader_user", Email: "reader@test.com", Password: "ValidPass123!",
	})

	now := time.Now().UTC()
	_ = repo.GrantAccountAccess(ctx, &models.AccountAccess{
		ID: "acc-edit", AccountID: accID, UserID: uEdit.ID, AccessLevel: models.AccessLevelEdit, CreatedAt: now, UpdatedAt: now,
	})
	_ = repo.GrantAccountAccess(ctx, &models.AccountAccess{
		ID: "acc-read", AccountID: accID, UserID: uRead.ID, AccessLevel: models.AccessLevelRead, CreatedAt: now, UpdatedAt: now,
	})

	// Editor can add asset
	q := 10.0
	_, err := ctrl.AddAccountAsset(ctx, uEdit.ID, accID, models.AccountAssetAddRequest{AssetID: slvID, Quantity: &q})
	if err != nil {
		t.Fatalf("editor should be allowed to add asset: %v", err)
	}

	// Reader can list assets
	list, err := ctrl.ListAccountAssets(ctx, uRead.ID, accID)
	if err != nil || len(list) != 1 {
		t.Fatalf("reader should be allowed to list assets: %v", err)
	}

	// Reader cannot add asset (403 Forbidden)
	_, err = ctrl.AddAccountAsset(ctx, uRead.ID, accID, models.AccountAssetAddRequest{AssetID: slvID, Quantity: &q})
	var appErr *controller.AppError
	if !errors.As(err, &appErr) || appErr.Status != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for reader add asset, got: %v", err)
	}
}

func TestAssetAccountNegativeCases(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()
	ownerID, accID, slvID, _ := setupPreciousMetalsAccount(ctx, ctrl)

	uNoAccess, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "no_access", Email: "noaccess@test.com", Password: "ValidPass123!",
	})

	// Non-member cannot access account (404)
	_, err := ctrl.ListAccountAssets(ctx, uNoAccess.ID, accID)
	var appErr *controller.AppError
	if !errors.As(err, &appErr) || appErr.Status != http.StatusNotFound {
		t.Fatalf("expected 404 for unshared user, got: %v", err)
	}

	// Negative/zero quantity rejected (400)
	negQ := -5.0
	_, err = ctrl.AddAccountAsset(ctx, ownerID, accID, models.AccountAssetAddRequest{AssetID: slvID, Quantity: &negQ})
	if !errors.As(err, &appErr) || appErr.Status != http.StatusBadRequest {
		t.Fatalf("expected 400 for negative quantity, got: %v", err)
	}

	// Non-asset account rejects asset operations (400)
	chkAcc, _ := ctrl.CreateAccount(ctx, ownerID, models.AccountCreationRequest{
		Name: "Checking", Type: "checking", Description: "Personal checking",
	})
	posQ := 1.0
	_, err = ctrl.AddAccountAsset(ctx, ownerID, chkAcc.ID, models.AccountAssetAddRequest{AssetID: slvID, Quantity: &posQ})
	if !errors.As(err, &appErr) || appErr.Status != http.StatusBadRequest {
		t.Fatalf("expected 400 for adding asset to checking account, got: %v", err)
	}
}
