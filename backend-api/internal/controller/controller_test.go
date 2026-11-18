package controller_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/controller"
	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/repository"
)

func setupController() (*controller.Controller, repository.Repository) {
	repo := repository.NewMemoryRepository()
	authService := auth.NewService("test-secret-123456", auth.DefaultExpirationSeconds)
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

func TestRegisterUser_CaptchaEnforcement(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()

	ctrl.SetRequireCaptcha(true)

	// 1. Missing captcha ID & solution when required
	_, err := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "need_captcha_user",
		Email:    "need_captcha@example.com",
		Password: "ValidPassword123!",
	})
	if err == nil {
		t.Fatal("expected error on missing captcha, got nil")
	}
	var appErr *controller.AppError
	if !errors.As(err, &appErr) || appErr.Code != "CAPTCHA_REQUIRED" {
		t.Errorf("expected CAPTCHA_REQUIRED, got %v", err)
	}

	// 2. Invalid captcha solution
	_, err = ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username:        "bad_captcha_user",
		Email:           "bad_captcha@example.com",
		Password:        "ValidPassword123!",
		CaptchaID:       "dummy_id",
		CaptchaSolution: "wrong",
	})
	if err == nil {
		t.Fatal("expected error on invalid captcha, got nil")
	}
	if !errors.As(err, &appErr) || appErr.Code != "INVALID_CAPTCHA" {
		t.Errorf("expected INVALID_CAPTCHA, got %v", err)
	}
}

func TestExchangePasswordForToken_TOTP(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()

	userRes, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "totp_user",
		Email:    "totp@example.com",
		Password: "ValidPassword123!",
	})

	validCode, _ := auth.GenerateCode(userRes.TOTPSecret, time.Now().Unix()/30)

	tok, err := ctrl.ExchangePasswordForToken(ctx, "totp_user", "ValidPassword123!", validCode)
	if err != nil || tok.AccessToken == "" {
		t.Fatalf("expected successful token exchange, got %v", err)
	}

	authSvc := auth.NewService("test-secret-123456", 3600)
	claims, _ := authSvc.ValidateToken(tok.AccessToken)
	if claims.Subscriber {
		t.Error("expected new user to have subscriber claim false")
	}

	_, err = ctrl.ExchangePasswordForToken(ctx, "totp_user", "ValidPassword123!", "000000")
	if err == nil {
		t.Fatal("expected error with invalid TOTP code")
	}
}

func TestExchangePasswordForToken_SubscriberClaim(t *testing.T) {
	ctrl, repo := setupController()
	ctx := context.Background()

	userRes, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "sub_user",
		Email:    "sub@example.com",
		Password: "ValidPassword123!",
	})
	_ = repo.CreateSubscription(ctx, &models.Subscription{
		ID: "sub-ctrl-1", UserID: userRes.ID, PaidDate: "2026-01-01", SubscriberEnds: "2099-12-31",
	})

	code, _ := auth.GenerateCode(userRes.TOTPSecret, time.Now().Unix()/30)
	tok, err := ctrl.ExchangePasswordForToken(ctx, "sub_user", "ValidPassword123!", code)
	if err != nil {
		t.Fatalf("ExchangePasswordForToken failed: %v", err)
	}

	authSvc := auth.NewService("test-secret-123456", auth.DefaultExpirationSeconds)
	claims, err := authSvc.ValidateToken(tok.AccessToken)
	if err != nil || !claims.Subscriber {
		t.Fatalf("expected subscriber true, got %v (err: %v)", claims.Subscriber, err)
	}
}

func TestRefreshToken_ValidAndForbiddenCases(t *testing.T) {
	ctrl, repo := setupController()
	ctx := context.Background()

	userRes, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "refresh_user", Email: "refresh@example.com", Password: "ValidPassword123!",
	})
	code, _ := auth.GenerateCode(userRes.TOTPSecret, time.Now().Unix()/30)
	initial, err := ctrl.ExchangePasswordForToken(ctx, "refresh_user", "ValidPassword123!", code)
	if err != nil || initial.ExpiresIn != 900 {
		t.Fatalf("expected initial ExpiresIn=900, got %+v err=%v", initial, err)
	}

	_ = repo.CreateSubscription(ctx, &models.Subscription{
		ID: "sub-ref-1", UserID: userRes.ID, PaidDate: "2026-01-01", SubscriberEnds: "2099-12-31",
	})
	refreshed, err := ctrl.RefreshToken(ctx, userRes.ID, "refresh_user")
	if err != nil || refreshed.ExpiresIn != 900 || refreshed.AccessToken == initial.AccessToken {
		t.Fatalf("unexpected refresh result: %+v err=%v", refreshed, err)
	}
	claims, err := auth.NewService("test-secret-123456", 900).ValidateToken(refreshed.AccessToken)
	if err != nil || !claims.Subscriber || claims.UserID != userRes.ID {
		t.Fatalf("unexpected refreshed claims: %+v err=%v", claims, err)
	}

	_, err = ctrl.RefreshToken(ctx, userRes.ID, "other_user")
	assertAppErrorStatus(t, err, http.StatusForbidden)
	_, err = ctrl.RefreshToken(ctx, "missing-user-id", "refresh_user")
	assertAppErrorStatus(t, err, http.StatusNotFound)
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

func TestSnapshotAccountWorkflow(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()
	user, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "trader_user", Email: "trader@example.com", Password: "ValidPass123!",
	})

	acc, err := ctrl.CreateAccount(ctx, user.ID, models.AccountCreationRequest{
		Name: "Trading Account", Type: "snapshot", Description: "Portfolio trending",
	})
	if err != nil {
		t.Fatalf("failed to create snapshot account: %v", err)
	}

	val1 := 1200.00
	snap1, err := ctrl.CreateAccountSnapshot(ctx, user.ID, acc.ID, models.AccountSnapshotCreateRequest{
		Value: &val1, SnapshotDate: "2026-11-11",
	})
	if err != nil || snap1.Value != 1200.00 {
		t.Fatalf("failed to create snapshot 1: %v", err)
	}

	val2 := 1210.00
	snap2, err := ctrl.CreateAccountSnapshot(ctx, user.ID, acc.ID, models.AccountSnapshotCreateRequest{
		Value: &val2, SnapshotDate: "2026-11-12",
	})
	if err != nil || snap2.Value != 1210.00 {
		t.Fatalf("failed to create snapshot 2: %v", err)
	}

	// Conflict on duplicate date
	valDup := 1215.00
	_, err = ctrl.CreateAccountSnapshot(ctx, user.ID, acc.ID, models.AccountSnapshotCreateRequest{
		Value: &valDup, SnapshotDate: "2026-11-12",
	})
	var appErr *controller.AppError
	if !errors.As(err, &appErr) || appErr.Status != http.StatusConflict {
		t.Fatalf("expected 409 conflict on duplicate snapshot date, got: %v", err)
	}

	// List trending snapshots
	list, err := ctrl.ListAccountSnapshots(ctx, user.ID, acc.ID, nil, nil)
	if err != nil || len(list) != 2 {
		t.Fatalf("expected 2 snapshots, got %d (%v)", len(list), err)
	}
	if list[0].Value != 1200.00 || list[1].Value != 1210.00 {
		t.Errorf("unexpected snapshot ordering: %+v", list)
	}
}

func TestAssetAccountTrendAndDateTracking(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()

	u, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "crypto_holder", Email: "holder@crypto.test", Password: "ValidPass123!",
	})
	acc, _ := ctrl.CreateAccount(ctx, u.ID, models.AccountCreationRequest{
		Name: "Crypto Holdings", Type: "asset", Description: "Crypto portfolio",
	})
	btc, _ := ctrl.CreateAsset(ctx, u.ID, models.AssetCreateRequest{Label: "btc", Description: "Bitcoin"})

	setupBtcValuations(ctx, ctrl, u.ID, btc.ID)
	setupBtcHoldings(ctx, ctrl, u.ID, acc.ID, btc.ID)

	summary, err := ctrl.GetAssetAccountSummary(ctx, u.ID, acc.ID)
	if err != nil || summary.TotalValue != 13000.0 {
		t.Fatalf("expected summary total 13000, got %v (err: %v)", summary.TotalValue, err)
	}

	trend, err := ctrl.GetAssetAccountTrend(ctx, u.ID, acc.ID, nil, nil)
	if err != nil || len(trend) != 3 {
		t.Fatalf("expected 3 trend points, got %d (err: %v)", len(trend), err)
	}
	if trend[0].TotalValue != 12000.0 || trend[1].TotalValue != 30000.0 || trend[2].TotalValue != 13000.0 {
		t.Errorf("unexpected trend points: %+v", trend)
	}
}

func setupBtcValuations(ctx context.Context, ctrl *controller.Controller, uID, btcID string) {
	p1, p2, p3 := 40000.0, 60000.0, 65000.0
	_, _ = ctrl.CreateAssetValue(ctx, uID, btcID, models.AssetValueCreateRequest{Value: &p1, ValueDate: "2026-01-01"})
	_, _ = ctrl.CreateAssetValue(ctx, uID, btcID, models.AssetValueCreateRequest{Value: &p2, ValueDate: "2026-07-01"})
	_, _ = ctrl.CreateAssetValue(ctx, uID, btcID, models.AssetValueCreateRequest{Value: &p3, ValueDate: "2026-10-01"})
}

func setupBtcHoldings(ctx context.Context, ctrl *controller.Controller, uID, accID, btcID string) {
	q1, q2, q3 := 0.3, 0.5, 0.2
	_, _ = ctrl.AddAccountAsset(ctx, uID, accID, models.AccountAssetAddRequest{AssetID: btcID, Quantity: &q1, AssetDate: "2026-01-01"})
	_, _ = ctrl.AddAccountAsset(ctx, uID, accID, models.AccountAssetAddRequest{AssetID: btcID, Quantity: &q2, AssetDate: "2026-07-01"})
	_, _ = ctrl.AddAccountAsset(ctx, uID, accID, models.AccountAssetAddRequest{AssetID: btcID, Quantity: &q3, AssetDate: "2026-10-01"})
}

func TestPerspectiveController_LifecycleAndView(t *testing.T) {
	ctrl, repo := setupController()
	ctx := context.Background()

	u, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "perspective_user", Email: "persp@test.com", Password: "ValidPass123!",
	})
	snapAcc, _ := ctrl.CreateAccount(ctx, u.ID, models.AccountCreationRequest{
		Name: "Retirement Snapshot", Type: "snapshot", Description: "IRA Account",
	})
	v1, v2 := 10000.0, 12000.0
	_, _ = ctrl.CreateAccountSnapshot(ctx, u.ID, snapAcc.ID, models.AccountSnapshotCreateRequest{Value: &v1, SnapshotDate: "2025-01-01"})
	_, _ = ctrl.CreateAccountSnapshot(ctx, u.ID, snapAcc.ID, models.AccountSnapshotCreateRequest{Value: &v2, SnapshotDate: "2026-01-01"})

	chkAcc, _ := ctrl.CreateAccount(ctx, u.ID, models.AccountCreationRequest{
		Name: "Checking Ledger", Type: "checking", Description: "Everyday Checking",
	})
	dep, exp := 5000.0, -100.0
	_, _ = ctrl.CreateCheckingRecord(ctx, u.ID, chkAcc.ID, models.CheckingRecordCreateRequest{Value: &dep, DateSent: "2026-01-01", Entity: "Payroll"})
	_, _ = ctrl.CreateCheckingRecord(ctx, u.ID, chkAcc.ID, models.CheckingRecordCreateRequest{Value: &exp, DateSent: "2026-01-11", Entity: "Store"})

	testPerspectiveFlow(t, ctx, ctrl, repo, u.ID, snapAcc.ID, chkAcc.ID)
}

func testPerspectiveFlow(t *testing.T, ctx context.Context, ctrl *controller.Controller, _ repository.Repository, uID, snapID, chkID string) {
	trendTrue, spendTrue, yoyTrue, ruinedTrue := true, true, true, true
	pRes, err := ctrl.CreatePerspective(ctx, uID, models.PerspectiveCreateRequest{
		Name: "Total Net Worth", Description: "Combined view",
		AccountIDs: []string{snapID, chkID},
		Settings: &models.PerspectiveSettingsInput{
			ShowFinancialTrend:         &trendTrue,
			CalculateAverageDailySpend: &spendTrue,
			CalculateAverageReturnYoY:  &yoyTrue,
			TrackRuinedDate:            &ruinedTrue,
		},
	})
	if err != nil || pRes.Name != "Total Net Worth" {
		t.Fatalf("failed to create perspective: %v", err)
	}

	view, err := ctrl.GetPerspectiveView(ctx, uID, pRes.ID)
	if err != nil {
		t.Fatalf("failed to get perspective view: %v", err)
	}
	if view.CurrentTotalValue != 16900.0 { // 12000 (snapshot) + 4900 (checking: 5000 - 100)
		t.Fatalf("expected current total 16900.0, got %f", view.CurrentTotalValue)
	}
	if len(view.Trend) == 0 {
		t.Fatalf("expected non-empty trend")
	}
	if view.AverageDailySpend == nil || *view.AverageDailySpend <= 0 {
		t.Fatalf("expected calculated average daily spend, got %v", view.AverageDailySpend)
	}
	if view.RuinedDate == nil || view.DaysUntilRuined == nil {
		t.Fatalf("expected ruined forecast date and days, got %v, %v", view.RuinedDate, view.DaysUntilRuined)
	}
}

func registerTestAccountParticipants(t *testing.T, ctx context.Context, ctrl *controller.Controller) (ownerID, readerID, editorID, outsiderID, accID string) {
	t.Helper()
	owner, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{Username: "acc_owner", Email: "acc_owner@test.com", Password: "ValidPass123!"})
	reader, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{Username: "acc_reader", Email: "acc_reader@test.com", Password: "ValidPass123!"})
	editor, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{Username: "acc_editor", Email: "acc_editor@test.com", Password: "ValidPass123!"})
	outsider, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{Username: "acc_out", Email: "acc_out@test.com", Password: "ValidPass123!"})
	acc, err := ctrl.CreateAccount(ctx, owner.ID, models.AccountCreationRequest{
		Name: "Primary Everyday Checking", Type: "checking", Description: "Monthly household budget",
	})
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	_, _ = ctrl.GrantAccountAccess(ctx, owner.ID, acc.ID, reader.ID, models.AccessLevelRead)
	_, _ = ctrl.GrantAccountAccess(ctx, owner.ID, acc.ID, editor.ID, models.AccessLevelEdit)
	return owner.ID, reader.ID, editor.ID, outsider.ID, acc.ID
}

func assertAppErrorStatus(t *testing.T, err error, wantStatus int) {
	t.Helper()
	var appErr *controller.AppError
	if !errors.As(err, &appErr) || appErr.Status != wantStatus {
		t.Fatalf("expected status %d, got err=%v", wantStatus, err)
	}
}

func TestPatchAccount_PermissionsAndPartialUpdates(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()
	ownerID, readerID, editorID, outsiderID, accID := registerTestAccountParticipants(t, ctx, ctrl)

	newName := "  Renamed Operating Checking  "
	res, err := ctrl.PatchAccount(ctx, ownerID, accID, models.AccountPatchRequest{Name: &newName})
	if err != nil || res.Name != "Renamed Operating Checking" || res.Description != "Monthly household budget" {
		t.Fatalf("owner name-only PatchAccount failed: res=%+v err=%v", res, err)
	}

	newDesc := "  Updated household expense tracking  "
	resDesc, err := ctrl.PatchAccount(ctx, editorID, accID, models.AccountPatchRequest{Description: &newDesc})
	if err != nil || resDesc.Name != "Renamed Operating Checking" || resDesc.Description != "Updated household expense tracking" {
		t.Fatalf("editor description-only PatchAccount failed: res=%+v err=%v", resDesc, err)
	}

	bothName, bothDesc := "Joint Household Checking", "Shared monthly bills and groceries"
	resBoth, err := ctrl.PatchAccount(ctx, ownerID, accID, models.AccountPatchRequest{Name: &bothName, Description: &bothDesc})
	if err != nil || resBoth.Name != bothName || resBoth.Description != bothDesc || resBoth.Type != "checking" {
		t.Fatalf("owner combined PatchAccount failed: res=%+v err=%v", resBoth, err)
	}

	_, err = ctrl.PatchAccount(ctx, readerID, accID, models.AccountPatchRequest{Name: &bothName})
	assertAppErrorStatus(t, err, http.StatusForbidden)
	_, err = ctrl.PatchAccount(ctx, outsiderID, accID, models.AccountPatchRequest{Name: &bothName})
	assertAppErrorStatus(t, err, http.StatusNotFound)
}

func TestPatchAccount_ValidationErrors(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()
	ownerID, _, _, _, accID := registerTestAccountParticipants(t, ctx, ctrl)

	blank := "   "
	overName := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	overDesc := overName + overName + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	cases := []models.AccountPatchRequest{
		{},
		{Name: &blank},
		{Description: &blank},
		{Name: &overName},
		{Description: &overDesc},
	}
	for _, tc := range cases {
		_, err := ctrl.PatchAccount(ctx, ownerID, accID, tc)
		assertAppErrorStatus(t, err, http.StatusBadRequest)
	}
}

func TestDeleteUser_CascadesAndAuthorization(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()

	uA, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{Username: "gone_user", Email: "gone@test.com", Password: "ValidPass123!"})
	uB, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{Username: "stay_user", Email: "stay@test.com", Password: "ValidPass123!"})
	accA, _ := ctrl.CreateAccount(ctx, uA.ID, models.AccountCreationRequest{Name: "Checking", Type: "checking", Description: "Personal"})
	val := -25.0
	_, _ = ctrl.CreateCheckingRecord(ctx, uA.ID, accA.ID, models.CheckingRecordCreateRequest{Value: &val, DateSent: "2026-11-10", Entity: "Market"})

	assertAppErrorStatus(t, ctrl.DeleteUser(ctx, uB.ID, "gone_user"), http.StatusForbidden)
	_, errGetB := ctrl.GetUser(ctx, uB.ID, "gone_user")
	assertAppErrorStatus(t, errGetB, http.StatusForbidden)
	profileA, errGetA := ctrl.GetUser(ctx, uA.ID, "gone_user")
	if errGetA != nil || profileA.Email != "gone@test.com" || profileA.TOTPSecret != "" {
		t.Fatalf("unexpected GetUser result: %+v err=%v", profileA, errGetA)
	}
	if err := ctrl.DeleteUser(ctx, uA.ID, "gone_user"); err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}
	assertAppErrorStatus(t, ctrl.DeleteUser(ctx, uA.ID, "gone_user"), http.StatusNotFound)
	_, err := ctrl.RefreshToken(ctx, uA.ID, "gone_user")
	assertAppErrorStatus(t, err, http.StatusNotFound)
	_, err = ctrl.RegisterUser(ctx, models.UserRegistrationRequest{Username: "gone_user", Email: "gone@test.com", Password: "ValidPass123!"})
	if err != nil {
		t.Fatalf("re-registering deleted username/email failed: %v", err)
	}
}

func TestPolicyAgreementLifecycleAndEnforcement(t *testing.T) {
	ctrl, repo := setupController()
	ctx := context.Background()
	termsHash := strings.Repeat("c", 64)
	ctrl.SetEnforcePolicy(true)
	ctrl.SetTermsHash(termsHash)
	bTrue, bFalse := true, false

	u, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{Username: "policy_alice", Email: "pa@test.com", Password: "ValidPass123!"})
	code, _ := auth.GenerateCode(u.TOTPSecret, time.Now().Unix()/30)
	tok, err := ctrl.ExchangePasswordForToken(ctx, "policy_alice", "ValidPass123!", code)
	if err != nil || tok.ExpiresIn <= 0 {
		t.Fatalf("ExchangePasswordForToken failed: %v", err)
	}
	if ok, _ := ctrl.IsUserPolicyApproved(ctx, u.ID); ok {
		t.Fatal("expected IsUserPolicyApproved=false before agreement")
	}

	_, err = ctrl.RecordPolicyAgreement(ctx, u.ID, "policy_alice", "203.0.113.42", models.PolicyAgreementRequest{
		TermsApproved: &bTrue, PrivacyApproved: &bTrue, USCustomer: &bFalse, TermsSHA256: termsHash,
	})
	assertAppErrorStatus(t, err, http.StatusBadRequest)

	_, err = ctrl.RecordPolicyAgreement(ctx, u.ID, "policy_alice", "203.0.113.42", models.PolicyAgreementRequest{
		TermsApproved: &bTrue, PrivacyApproved: &bTrue, USCustomer: &bTrue, TermsSHA256: strings.Repeat("d", 64),
	})
	assertAppErrorStatus(t, err, http.StatusBadRequest)

	agreed, err := ctrl.RecordPolicyAgreement(ctx, u.ID, "policy_alice", "203.0.113.42", models.PolicyAgreementRequest{
		TermsApproved: &bTrue, PrivacyApproved: &bTrue, USCustomer: &bTrue, TermsSHA256: strings.ToUpper(termsHash),
	})
	ok, _ := ctrl.IsUserPolicyApproved(ctx, u.ID)
	if err != nil || !agreed.Active || agreed.ClientIP != "203.0.113.42" || !ok {
		t.Fatalf("expected active agreement recorded, got %+v err=%v", agreed, err)
	}
	_ = repo.DeactivateAllPolicyAgreements(ctx)
	if okAfter, _ := ctrl.IsUserPolicyApproved(ctx, u.ID); okAfter {
		t.Fatal("expected IsUserPolicyApproved=false after ledger deactivation")
	}
}

func TestAccountRecoveryAndSingleUseRotation(t *testing.T) {
	ctrl, repo := setupController()
	ctx := context.Background()
	u, err := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{Username: "rec_alice", Email: "rec@test.com", Password: "InitialPass1!"})
	if err != nil || len(strings.Fields(u.RecoveryPhrase)) != 12 || u.RecoveryKeyHint == "" {
		t.Fatalf("expected 12-word recovery phrase at registration, got %+v err=%v", u, err)
	}
	prof, _ := ctrl.GetUser(ctx, u.ID, "rec_alice")
	if prof.RecoveryPhrase != "" || prof.RecoveryKeyHint != u.RecoveryKeyHint {
		t.Fatalf("expected GetUser to return hint only without phrase, got %+v", prof)
	}
	_, errBad := ctrl.RecoverAccount(ctx, "rec_alice", "203.0.113.9", models.AccountRecoveryRequest{
		RecoveryPhrase: "wrong phrase words one two three four five six seven eight nine", NewPassword: "RotatedPass2!",
	})
	assertAppErrorStatus(t, errBad, http.StatusUnauthorized)

	rec, err := ctrl.RecoverAccount(ctx, "rec_alice", "203.0.113.9", models.AccountRecoveryRequest{
		RecoveryPhrase: u.RecoveryPhrase, NewPassword: "RotatedPass2!",
	})
	if err != nil || rec.TOTPSecret == u.TOTPSecret || rec.RecoveryPhrase == u.RecoveryPhrase {
		t.Fatalf("expected new TOTP and rotated phrase on recovery, got %+v err=%v", rec, err)
	}
	// Old phrase & old TOTP/password must fail; new TOTP/password must succeed
	_, errReplay := ctrl.RecoverAccount(ctx, "rec_alice", "203.0.113.9", models.AccountRecoveryRequest{
		RecoveryPhrase: u.RecoveryPhrase, NewPassword: "ReplayPass3!",
	})
	assertAppErrorStatus(t, errReplay, http.StatusUnauthorized)
	newCode, _ := auth.GenerateCode(rec.TOTPSecret, time.Now().Unix()/30)
	if _, err := ctrl.ExchangePasswordForToken(ctx, "rec_alice", "InitialPass1!", newCode); err == nil {
		t.Fatal("expected old password to be rejected after recovery")
	}
	if _, err := ctrl.ExchangePasswordForToken(ctx, "rec_alice", "RotatedPass2!", newCode); err != nil {
		t.Fatalf("expected new password + TOTP to succeed after recovery, got %v", err)
	}
	rot, err := ctrl.RotateRecoveryKey(ctx, u.ID, "rec_alice", models.RotateRecoveryKeyRequest{Password: "RotatedPass2!", TOTPCode: newCode})
	if err != nil || rot.RecoveryPhrase == rec.RecoveryPhrase {
		t.Fatalf("expected rotated recovery phrase, got %+v err=%v", rot, err)
	}
	if entries, _ := repo.ListRecoveryKeysByUserID(ctx, u.ID); len(entries) != 3 {
		t.Fatalf("expected 3 ledger entries (register, recover, rotate), got %d", len(entries))
	}
}


