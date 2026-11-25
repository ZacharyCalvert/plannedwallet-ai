package controller_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/plannedwallet/internal/auth"
	"github.com/plannedwallet/internal/controller"
	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/repository"
)

func TestUserSettings_PremiumGatingAndPartialPatch(t *testing.T) {
	repo := repository.NewMemoryRepository()
	authSvc := auth.NewService("test-secret-123456", 900)
	ctrl := controller.NewController(repo, authSvc)
	ctx := context.Background()

	basicUser, err := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "basic_settings_u", Email: "basic_settings@example.com", Password: "ValidPassword123!",
	})
	if err != nil {
		t.Fatalf("RegisterUser failed: %v", err)
	}

	acc1 := uuid.New().String()
	acc2 := uuid.New().String()
	persp1 := uuid.New().String()

	// Basic user cannot get or patch settings
	_, err = ctrl.GetUserSettings(ctx, basicUser.ID, basicUser.Username)
	assertAppErrorStatus(t, err, http.StatusForbidden)

	order := []string{acc2, acc1}
	_, err = ctrl.PatchUserSettings(ctx, basicUser.ID, basicUser.Username, models.PatchUserSettingsRequest{
		AccountOrder: &order,
	})
	assertAppErrorStatus(t, err, http.StatusForbidden)

	// Grant subscription -> premium user
	_ = repo.CreateSubscription(ctx, &models.Subscription{
		ID: "sub-settings-1", UserID: basicUser.ID, PaidDate: "2026-01-01", SubscriberEnds: "2099-12-31",
	})

	initial, err := ctrl.GetUserSettings(ctx, basicUser.ID, basicUser.Username)
	if err != nil {
		t.Fatalf("GetUserSettings failed for subscriber: %v", err)
	}
	if len(initial.Settings.AccountOrder) != 0 || len(initial.Settings.DailyUpdateOrder) != 0 || len(initial.Settings.PinnedBalances) != 0 {
		t.Fatalf("expected empty initial slices, got %+v", initial.Settings)
	}

	// Partial patch 1: account_order (deduped)
	orderWithDups := []string{acc2, acc1, acc2}
	afterOrder, err := ctrl.PatchUserSettings(ctx, basicUser.ID, basicUser.Username, models.PatchUserSettingsRequest{
		AccountOrder: &orderWithDups,
	})
	if err != nil {
		t.Fatalf("PatchUserSettings account_order failed: %v", err)
	}
	if len(afterOrder.Settings.AccountOrder) != 2 || afterOrder.Settings.AccountOrder[0] != acc2 {
		t.Fatalf("expected deduped account_order, got %+v", afterOrder.Settings.AccountOrder)
	}

	// Partial patch 2: pinned_balances preserves account_order
	pins := []models.PinnedBalanceRef{
		{Kind: "account", ID: acc1},
		{Kind: "perspective", ID: persp1},
		{Kind: "account", ID: acc1},
	}
	afterPins, err := ctrl.PatchUserSettings(ctx, basicUser.ID, basicUser.Username, models.PatchUserSettingsRequest{
		PinnedBalances: &pins,
	})
	if err != nil {
		t.Fatalf("PatchUserSettings pinned_balances failed: %v", err)
	}
	if len(afterPins.Settings.AccountOrder) != 2 || len(afterPins.Settings.PinnedBalances) != 2 {
		t.Fatalf("expected account_order preserved and pinned_balances deduped, got %+v", afterPins.Settings)
	}
}

func TestUserSettings_BoundsAndUUIDValidation(t *testing.T) {
	repo := repository.NewMemoryRepository()
	authSvc := auth.NewService("test-secret-123456", 900)
	ctrl := controller.NewController(repo, authSvc)
	ctrl.SetSubscriberAlwaysOn(true)
	ctx := context.Background()

	u, _ := ctrl.RegisterUser(ctx, models.UserRegistrationRequest{
		Username: "bounds_u", Email: "bounds@example.com", Password: "ValidPassword123!",
	})

	// Non-UUID account_order -> 400
	badOrder := []string{"not-a-uuid"}
	_, err := ctrl.PatchUserSettings(ctx, u.ID, u.Username, models.PatchUserSettingsRequest{AccountOrder: &badOrder})
	assertAppErrorStatus(t, err, http.StatusBadRequest)

	// > 200 account_order entries -> 400
	tooManyAccounts := make([]string, 201)
	for i := range tooManyAccounts {
		tooManyAccounts[i] = uuid.New().String()
	}
	_, err = ctrl.PatchUserSettings(ctx, u.ID, u.Username, models.PatchUserSettingsRequest{AccountOrder: &tooManyAccounts})
	assertAppErrorStatus(t, err, http.StatusBadRequest)

	// > 300 daily_update_order entries -> 400
	tooManyDaily := make([]string, 301)
	for i := range tooManyDaily {
		tooManyDaily[i] = "snapshot:" + uuid.New().String()
	}
	_, err = ctrl.PatchUserSettings(ctx, u.ID, u.Username, models.PatchUserSettingsRequest{DailyUpdateOrder: &tooManyDaily})
	assertAppErrorStatus(t, err, http.StatusBadRequest)

	// Invalid daily_update_order prefix/UUID -> 400
	badDaily := []string{"checking:" + uuid.New().String()}
	_, err = ctrl.PatchUserSettings(ctx, u.ID, u.Username, models.PatchUserSettingsRequest{DailyUpdateOrder: &badDaily})
	assertAppErrorStatus(t, err, http.StatusBadRequest)

	// Invalid pinned_balances kind or non-UUID -> 400
	badPinKind := []models.PinnedBalanceRef{{Kind: "invalid", ID: uuid.New().String()}}
	_, err = ctrl.PatchUserSettings(ctx, u.ID, u.Username, models.PatchUserSettingsRequest{PinnedBalances: &badPinKind})
	assertAppErrorStatus(t, err, http.StatusBadRequest)

	badPinUUID := []models.PinnedBalanceRef{{Kind: "account", ID: "not-a-uuid"}}
	_, err = ctrl.PatchUserSettings(ctx, u.ID, u.Username, models.PatchUserSettingsRequest{PinnedBalances: &badPinUUID})
	assertAppErrorStatus(t, err, http.StatusBadRequest)
}
