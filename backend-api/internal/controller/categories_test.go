package controller_test

import (
	"context"
	"testing"

	"github.com/plannedwallet/internal/controller"
	"github.com/plannedwallet/internal/models"
)

func registerTestUser(t *testing.T, ctrl *controller.Controller, username string) *models.UserResponse {
	t.Helper()
	res, err := ctrl.RegisterUser(context.Background(), models.UserRegistrationRequest{
		Username: username,
		Email:    username + "@example.com",
		Password: "SecurePass123!",
	})
	if err != nil {
		t.Fatalf("RegisterUser(%s) failed: %v", username, err)
	}
	return res
}

func TestDefaultCategoriesAndUserIsolation(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()
	u1 := registerTestUser(t, ctrl, "catuser1")
	u2 := registerTestUser(t, ctrl, "catuser2")

	cats, err := ctrl.ListCategories(ctx, u1.ID)
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}
	if len(cats) != 7 {
		t.Fatalf("expected 7 default categories, got %d", len(cats))
	}

	created, err := ctrl.CreateCategory(ctx, u1.ID, models.CategoryCreateRequest{Name: "Dining Out"})
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	if created.IsDefault {
		t.Fatalf("expected user-created category to have IsDefault=false")
	}

	u2Cats, err := ctrl.ListCategories(ctx, u2.ID)
	if err != nil {
		t.Fatalf("ListCategories for u2 failed: %v", err)
	}
	if len(u2Cats) != 7 {
		t.Fatalf("expected u2 to see only 7 default categories, got %d", len(u2Cats))
	}

	if _, err := ctrl.CreateCategory(ctx, u1.ID, models.CategoryCreateRequest{Name: "dining out"}); err == nil {
		t.Fatalf("expected duplicate category name conflict")
	}
}

func TestCategorySharedAccountVisibilityAndDeletion(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()
	owner := registerTestUser(t, ctrl, "catowner")
	sharedUser := registerTestUser(t, ctrl, "catshared")
	outsider := registerTestUser(t, ctrl, "catoutsider")

	acc, err := ctrl.CreateAccount(ctx, owner.ID, models.AccountCreationRequest{
		Name: "Household Checking", Type: "checking", Description: "Shared household",
	})
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	custom, err := ctrl.CreateCategory(ctx, owner.ID, models.CategoryCreateRequest{Name: "Subscriptions"})
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	amt := -25.50
	rec, err := ctrl.CreateCheckingRecord(ctx, owner.ID, acc.ID, models.CheckingRecordCreateRequest{
		Value: &amt, DateSent: "2026-03-01", Entity: "Streaming Service", CategoryID: &custom.ID,
	})
	if err != nil || rec.CategoryID == nil || *rec.CategoryID != custom.ID {
		t.Fatalf("CreateCheckingRecord with category failed: rec=%+v err=%v", rec, err)
	}

	if _, err := ctrl.GrantAccountAccess(ctx, owner.ID, acc.ID, sharedUser.ID, models.AccessLevelEdit); err != nil {
		t.Fatalf("GrantAccountAccess failed: %v", err)
	}

	sharedCats, _ := ctrl.ListCategories(ctx, sharedUser.ID)
	if len(sharedCats) != 8 {
		t.Fatalf("expected shared user to see 8 categories (7 default + 1 shared), got %d", len(sharedCats))
	}
	outsiderCats, _ := ctrl.ListCategories(ctx, outsider.ID)
	if len(outsiderCats) != 7 {
		t.Fatalf("expected outsider to see 7 categories, got %d", len(outsiderCats))
	}

	if err := ctrl.DeleteCategory(ctx, sharedUser.ID, custom.ID); err == nil {
		t.Fatalf("expected non-owner delete to fail")
	}
	if err := ctrl.DeleteCategory(ctx, owner.ID, custom.ID); err != nil {
		t.Fatalf("owner DeleteCategory failed: %v", err)
	}

	updatedRec, err := ctrl.GetCheckingRecord(ctx, owner.ID, acc.ID, rec.ID)
	if err != nil || updatedRec.CategoryID != nil {
		t.Fatalf("expected deleted category to set record CategoryID=nil, got %+v err=%v", updatedRec, err)
	}
}

func TestAccountCategoryExpenseSummary(t *testing.T) {
	ctrl, _ := setupController()
	ctx := context.Background()
	u := registerTestUser(t, ctrl, "pieuser")
	acc, err := ctrl.CreateAccount(ctx, u.ID, models.AccountCreationRequest{
		Name: "Credit Card", Type: "credit card", Description: "Monthly expenses",
	})
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}

	cats, _ := ctrl.ListCategories(ctx, u.ID)
	housingID := cats[0].ID
	for _, item := range []struct {
		val float64
		cat *string
	}{
		{-1200.00, &housingID},
		{-45.00, nil},
		{500.00, nil},
	} {
		v := item.val
		if _, err := ctrl.CreateCheckingRecord(ctx, u.ID, acc.ID, models.CheckingRecordCreateRequest{
			Value: &v, DateSent: "2026-03-05", Entity: "Test Entry", CategoryID: item.cat,
		}); err != nil {
			t.Fatalf("CreateCheckingRecord failed: %v", err)
		}
	}

	summary, err := ctrl.GetAccountCategorySummary(ctx, u.ID, acc.ID, nil, nil, nil)
	if err != nil {
		t.Fatalf("GetAccountCategorySummary failed: %v", err)
	}
	if summary.TotalExpenses != 1245.00 || len(summary.Categories) != 2 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if summary.Categories[0].TotalAmount != 1200.00 || summary.Categories[1].CategoryName != "Uncategorized" {
		t.Fatalf("unexpected category slices: %+v", summary.Categories)
	}
}
