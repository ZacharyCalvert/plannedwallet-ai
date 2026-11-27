package controller_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/models"
)

func TestMobileFlow_TokenTargetsRecordAndRevocation(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()

	owner, err := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "mob_owner", Email: "mob_owner@example.com", Password: "ComplexPassword123!",
	})
	if err != nil {
		t.Fatalf("RegisterUser owner failed: %v", err)
	}
	other, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "mob_other", Email: "mob_other@example.com", Password: "ComplexPassword123!",
	})

	chk, _ := ctrl.CreateAccount(ctx, owner.ID, models.AccountCreationRequest{Name: "Everyday Checking", Type: "checking", Description: "Checking desc"})
	sav, _ := ctrl.CreateAccount(ctx, owner.ID, models.AccountCreationRequest{Name: "Emergency Reserve", Type: "savings", Description: "Savings desc"})
	cc, _ := ctrl.CreateAccount(ctx, owner.ID, models.AccountCreationRequest{Name: "Rewards Card", Type: "credit card", Description: "Card desc"})
	snp, _ := ctrl.CreateAccount(ctx, owner.ID, models.AccountCreationRequest{Name: "Brokerage Snapshot", Type: "snapshot", Description: "Snapshot desc"})
	ast, _ := ctrl.CreateAccount(ctx, owner.ID, models.AccountCreationRequest{Name: "Precious Metals", Type: "asset", Description: "Asset desc"})
	sharedRO, _ := ctrl.CreateAccount(ctx, other.ID, models.AccountCreationRequest{Name: "Shared Read Only", Type: "checking", Description: "Read only"})
	sharedEdit, _ := ctrl.CreateAccount(ctx, other.ID, models.AccountCreationRequest{Name: "Shared Edit", Type: "savings", Description: "Editable"})
	_, _ = ctrl.GrantAccountAccess(ctx, other.ID, sharedRO.ID, owner.ID, models.AccessLevelRead)
	_, _ = ctrl.GrantAccountAccess(ctx, other.ID, sharedEdit.ID, owner.ID, models.AccessLevelEdit)

	// Missing mobile identifier fails
	_, err = ctrl.IssueMobileToken(ctx, owner.ID, "   ")
	assertAppErrorStatus(t, err, http.StatusBadRequest)

	mobTok, err := ctrl.IssueMobileToken(ctx, owner.ID, "ios-install-uuid-0001")
	if err != nil {
		t.Fatalf("IssueMobileToken failed: %v", err)
	}
	if mobTok.Limitation != auth.LimitationMobile || mobTok.ExpiresIn != 180*24*3600 {
		t.Fatalf("unexpected mobile token metadata: %+v", mobTok)
	}

	// ListMobileTargets includes checking/savings/credit card (own or edit) + categories, excludes asset/snapshot/read-only
	targets, err := ctrl.ListMobileTargets(ctx, owner.ID)
	if err != nil {
		t.Fatalf("ListMobileTargets failed: %v", err)
	}
	if len(targets.Accounts) != 4 {
		t.Fatalf("expected 4 editable record accounts (chk, sav, cc, sharedEdit), got %d: %+v", len(targets.Accounts), targets.Accounts)
	}
	for _, a := range targets.Accounts {
		if a.ID == snp.ID || a.ID == ast.ID || a.ID == sharedRO.ID {
			t.Fatalf("unexpected non-editable/non-record account in mobile list: %+v", a)
		}
	}
	if len(targets.Categories) == 0 {
		t.Fatal("expected seeded default categories in mobile targets")
	}

	val := -18.50
	catID := targets.Categories[0].ID
	recRes, err := ctrl.CreateMobileRecord(ctx, owner.ID, cc.ID, models.MobileRecordCreateRequest{
		Value: &val, DateSent: "2026-11-26", Entity: "Corner Grocery", CategoryID: &catID,
	})
	if err != nil {
		t.Fatalf("CreateMobileRecord on credit card failed: %v", err)
	}
	if recRes.ID == "" || recRes.AccountID != cc.ID {
		t.Fatalf("unexpected mobile record response: %+v", recRes)
	}
	_, _ = ctrl.CreateMobileRecord(ctx, owner.ID, chk.ID, models.MobileRecordCreateRequest{Value: &val, DateSent: "2026-11-26", Entity: "Cafe"})
	_, _ = ctrl.CreateMobileRecord(ctx, owner.ID, sav.ID, models.MobileRecordCreateRequest{Value: &val, DateSent: "2026-11-26", Entity: "Transfer"})

	// Non-record account rejected with 400; read-only rejected with 403; unshared rejected with 404
	_, errSnp := ctrl.CreateMobileRecord(ctx, owner.ID, snp.ID, models.MobileRecordCreateRequest{Value: &val, DateSent: "2026-11-26", Entity: "X"})
	assertAppErrorStatus(t, errSnp, http.StatusBadRequest)
	_, errRO := ctrl.CreateMobileRecord(ctx, owner.ID, sharedRO.ID, models.MobileRecordCreateRequest{Value: &val, DateSent: "2026-11-26", Entity: "X"})
	assertAppErrorStatus(t, errRO, http.StatusForbidden)
	_, errNoAccess := ctrl.CreateMobileRecord(ctx, other.ID, chk.ID, models.MobileRecordCreateRequest{Value: &val, DateSent: "2026-11-26", Entity: "X"})
	assertAppErrorStatus(t, errNoAccess, http.StatusNotFound)

	// Changing password revokes the active mobile token
	svc := auth.NewService("test-secret-123456", 900)
	claims, _ := svc.ValidateToken(mobTok.AccessToken)
	if err := ctrl.ValidateActiveMobileToken(ctx, claims.ID, owner.ID); err != nil {
		t.Fatalf("expected active mobile token before password change, got %v", err)
	}
	code, _ := auth.GenerateCode(owner.TOTPSecret, time.Now().Unix()/30)
	if _, err := ctrl.ChangePassword(ctx, owner.ID, owner.Username, models.ChangePasswordRequest{
		OldPassword: "ComplexPassword123!", NewPassword: "NewComplexPassword456!", NewPasswordConfirm: "NewComplexPassword456!", TOTPCode: code,
	}); err != nil {
		t.Fatalf("ChangePassword failed: %v", err)
	}
	assertAppErrorStatus(t, ctrl.ValidateActiveMobileToken(ctx, claims.ID, owner.ID), http.StatusUnauthorized)
}
