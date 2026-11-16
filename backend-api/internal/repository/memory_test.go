package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/repository"
)

func TestMemoryRepo_AssetUniqueness(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx := context.Background()
	now := time.Now().UTC()

	u1, u2 := "user-1", "user-2"
	a1 := &models.Asset{
		ID: "a-1", UserID: &u1, Label: "btc", Description: "Bitcoin", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateAsset(ctx, a1); err != nil {
		t.Fatalf("failed to create a1: %v", err)
	}

	// Duplicate label for same user
	a1Dup := &models.Asset{
		ID: "a-1-dup", UserID: &u1, Label: "BTC", Description: "Dup Bitcoin", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateAsset(ctx, a1Dup); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("expected ErrConflict on duplicate user label, got: %v", err)
	}

	// Same label for different user is allowed
	a2 := &models.Asset{
		ID: "a-2", UserID: &u2, Label: "btc", Description: "U2 Bitcoin", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateAsset(ctx, a2); err != nil {
		t.Fatalf("expected different user with same label to succeed, got: %v", err)
	}
}

func TestMemoryRepo_GlobalAssetUniqueness(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx := context.Background()
	now := time.Now().UTC()

	sys1 := &models.Asset{
		ID: "sys-1", UserID: nil, Label: "gold", Description: "Gold ounce", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateAsset(ctx, sys1); err != nil {
		t.Fatalf("failed to create sys1: %v", err)
	}

	sys1Dup := &models.Asset{
		ID: "sys-2", UserID: nil, Label: "GOLD", Description: "Dup gold", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateAsset(ctx, sys1Dup); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("expected ErrConflict on duplicate global system asset label, got: %v", err)
	}
}

func TestMemoryRepo_AssetValueMultiDayAndUniqueness(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx := context.Background()
	now := time.Now().UTC()

	u := "user-1"
	asset := &models.Asset{
		ID: "a-btc", UserID: &u, Label: "btc", Description: "Bitcoin", CreatedAt: now, UpdatedAt: now,
	}
	_ = repo.CreateAsset(ctx, asset)

	dates := []string{"2026-11-09", "2026-11-10", "2026-11-11"}
	for i, d := range dates {
		av := &models.AssetValue{
			ID: "v-" + d, AssetID: asset.ID, Value: 90000.0 + float64(i*1000), ValueDate: d, CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.CreateAssetValue(ctx, av); err != nil {
			t.Fatalf("failed to create asset value for %s: %v", d, err)
		}
	}

	// Duplicate date for same asset must fail
	dupVal := &models.AssetValue{
		ID: "v-dup", AssetID: asset.ID, Value: 91500.0, ValueDate: "2026-11-10", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateAssetValue(ctx, dupVal); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("expected ErrConflict on duplicate asset date, got: %v", err)
	}

	values, err := repo.ListAssetValues(ctx, asset.ID, nil, nil)
	if err != nil || len(values) != 3 {
		t.Fatalf("expected 3 values, got %d (%v)", len(values), err)
	}
}

func TestMemoryRepo_AssetCascadeDelete(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx := context.Background()
	now := time.Now().UTC()

	u := "user-1"
	asset := &models.Asset{
		ID: "a-eth", UserID: &u, Label: "eth", Description: "Ethereum", CreatedAt: now, UpdatedAt: now,
	}
	_ = repo.CreateAsset(ctx, asset)

	av := &models.AssetValue{
		ID: "v-eth-1", AssetID: asset.ID, Value: 3000.0, ValueDate: "2026-11-09", CreatedAt: now, UpdatedAt: now,
	}
	_ = repo.CreateAssetValue(ctx, av)

	if err := repo.DeleteAsset(ctx, asset.ID); err != nil {
		t.Fatalf("failed to delete asset: %v", err)
	}

	// Asset values should be gone
	vals, _ := repo.ListAssetValues(ctx, asset.ID, nil, nil)
	if len(vals) != 0 {
		t.Fatalf("expected 0 asset values after cascade delete, got %d", len(vals))
	}
}

func TestMemoryRepo_AccountAssetOperations(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx := context.Background()
	now := time.Now().UTC()

	u := "user-1"
	acc := &models.Account{
		ID: "acc-pm", AdminID: u, Name: "Precious Metals", Type: "asset", Description: "Metals", CreatedAt: now, UpdatedAt: now,
	}
	_ = repo.CreateAccount(ctx, acc)

	asset := &models.Asset{
		ID: "asset-slv", UserID: &u, Label: "slv-oz", Description: "Silver ounce", CreatedAt: now, UpdatedAt: now,
	}
	_ = repo.CreateAsset(ctx, asset)
	_ = repo.CreateAssetValue(ctx, &models.AssetValue{
		ID: "v-slv-1", AssetID: asset.ID, Value: 30.0, ValueDate: "2026-11-09", CreatedAt: now, UpdatedAt: now,
	})
	_ = repo.CreateAssetValue(ctx, &models.AssetValue{
		ID: "v-slv-2", AssetID: asset.ID, Value: 32.50, ValueDate: "2026-11-10", CreatedAt: now, UpdatedAt: now,
	})

	aa := &models.AccountAsset{
		ID: "aa-1", AccountID: acc.ID, AssetID: asset.ID, Quantity: 45.209, AssetDate: "2026-11-10", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.AddAccountAsset(ctx, aa); err != nil {
		t.Fatalf("failed to add account asset: %v", err)
	}

	// Conflict on duplicate asset in same account on same date
	aaDup := &models.AccountAsset{
		ID: "aa-dup", AccountID: acc.ID, AssetID: asset.ID, Quantity: 10.0, AssetDate: "2026-11-10", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.AddAccountAsset(ctx, aaDup); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("expected ErrConflict on duplicate asset in account on same date, got %v", err)
	}

	// Different date succeeds
	aaDate2 := &models.AccountAsset{
		ID: "aa-2", AccountID: acc.ID, AssetID: asset.ID, Quantity: 50.0, AssetDate: "2026-11-12", CreatedAt: now.Add(time.Hour), UpdatedAt: now.Add(time.Hour),
	}
	if err := repo.AddAccountAsset(ctx, aaDate2); err != nil {
		t.Fatalf("failed to add account asset on different date: %v", err)
	}

	details, err := repo.ListAccountAssets(ctx, acc.ID)
	if err != nil || len(details) != 2 {
		t.Fatalf("expected 2 details, got %d (%v)", len(details), err)
	}
	// Ordered by AssetDate DESC
	if details[0].AssetDate != "2026-11-12" || details[0].Quantity != 50.0 {
		t.Errorf("unexpected newest detail: %+v", details[0])
	}
	if details[1].AssetDate != "2026-11-10" || details[1].Quantity != 45.209 {
		t.Errorf("unexpected older detail: %+v", details[1])
	}
}

func TestMemoryRepo_AccountSnapshotOperations(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx := context.Background()
	now := time.Now().UTC()

	acc := &models.Account{
		ID: "acc-snap", AdminID: "u1", Name: "Trading Account", Type: "snapshot", Description: "Brokerage", CreatedAt: now, UpdatedAt: now,
	}
	_ = repo.CreateAccount(ctx, acc)

	s1 := &models.AccountSnapshot{
		ID: "snap-1", AccountID: acc.ID, Value: 1200.0, SnapshotDate: "2026-11-11", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateAccountSnapshot(ctx, s1); err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	s2 := &models.AccountSnapshot{
		ID: "snap-2", AccountID: acc.ID, Value: 1210.0, SnapshotDate: "2026-11-12", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateAccountSnapshot(ctx, s2); err != nil {
		t.Fatalf("failed to create snapshot 2: %v", err)
	}

	// Duplicate date conflict
	sDup := &models.AccountSnapshot{
		ID: "snap-dup", AccountID: acc.ID, Value: 1215.0, SnapshotDate: "2026-11-12", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateAccountSnapshot(ctx, sDup); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("expected ErrConflict on duplicate snapshot date, got %v", err)
	}

	list, err := repo.ListAccountSnapshots(ctx, acc.ID, nil, nil)
	if err != nil || len(list) != 2 {
		t.Fatalf("expected 2 snapshots, got %d (%v)", len(list), err)
	}
	if list[0].Value != 1200.0 || list[1].Value != 1210.0 {
		t.Errorf("unexpected list ordering or values: %+v", list)
	}

	// Delete
	if err := repo.DeleteAccountSnapshot(ctx, acc.ID, s1.ID); err != nil {
		t.Fatalf("failed to delete snapshot: %v", err)
	}
}

func TestMemoryRepo_CaptchaLifecycleAndExpiration(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx := context.Background()

	// 1. Set and Get (clear=false)
	digits := []byte{1, 2, 3, 4, 5, 6}
	if err := repo.SetCaptcha(ctx, "c-1", digits); err != nil {
		t.Fatalf("SetCaptcha failed: %v", err)
	}
	got, err := repo.GetCaptcha(ctx, "c-1", false)
	if err != nil || len(got) != 6 {
		t.Fatalf("GetCaptcha (clear=false) failed: %v, got %v", err, got)
	}

	// 2. Get (clear=true)
	got2, err := repo.GetCaptcha(ctx, "c-1", true)
	if err != nil || len(got2) != 6 {
		t.Fatalf("GetCaptcha (clear=true) failed: %v, got %v", err, got2)
	}

	// 3. Subsequent Get should return nil (consumed)
	got3, err := repo.GetCaptcha(ctx, "c-1", false)
	if err != nil || got3 != nil {
		t.Fatalf("expected nil for consumed captcha, got %v (%v)", got3, err)
	}

	// 4. DeleteExpiredCaptchas
	_ = repo.SetCaptcha(ctx, "c-2", digits)
	deleted, err := repo.DeleteExpiredCaptchas(ctx, -1*time.Second)
	if err != nil || deleted != 1 {
		t.Fatalf("expected 1 deleted expired captcha, got %d (%v)", deleted, err)
	}
}

func TestMemoryRepo_AssetStoreAndRecords(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx := context.Background()
	now := time.Now().UTC()

	acc := &models.Account{ID: "acc-asset-1", AdminID: "u-1", Name: "Crypto Portfolio", Type: "asset"}
	_ = repo.CreateAccount(ctx, acc)
	btc := &models.Asset{ID: "asset-alpha", Label: "Coin Alpha"}
	_ = repo.CreateAsset(ctx, btc)
	_ = repo.CreateAssetValue(ctx, &models.AssetValue{ID: "v1", AssetID: btc.ID, Value: 50000.0, ValueDate: "2026-01-01"})
	_ = repo.CreateAssetValue(ctx, &models.AssetValue{ID: "v2", AssetID: btc.ID, Value: 60000.0, ValueDate: "2026-06-01"})

	// 1. Create store 'red'
	storeRed := &models.AssetStore{ID: "store-red", AccountID: acc.ID, AssetID: btc.ID, Label: "red", Description: "Red hardware key", CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateAssetStore(ctx, storeRed); err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// 2. Duplicate label in same account
	storeDup := &models.AssetStore{ID: "store-dup", AccountID: acc.ID, AssetID: btc.ID, Label: "RED", CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateAssetStore(ctx, storeDup); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("expected conflict on duplicate store label, got: %v", err)
	}

	// 3. Add records for 'red': Jan 1 (0.2), Feb 1 (0.3), Oct 1 (0.5)
	r1 := &models.AssetStoreRecord{ID: "r1", AssetStoreID: storeRed.ID, Quantity: 0.2, RecordDate: "2026-01-01", CreatedAt: now, UpdatedAt: now}
	r2 := &models.AssetStoreRecord{ID: "r2", AssetStoreID: storeRed.ID, Quantity: 0.3, RecordDate: "2026-02-01", CreatedAt: now, UpdatedAt: now}
	r3 := &models.AssetStoreRecord{ID: "r3", AssetStoreID: storeRed.ID, Quantity: 0.5, RecordDate: "2026-10-01", CreatedAt: now, UpdatedAt: now}
	_ = repo.CreateAssetStoreRecord(ctx, r1)
	_ = repo.CreateAssetStoreRecord(ctx, r2)
	_ = repo.CreateAssetStoreRecord(ctx, r3)

	// Duplicate date record
	rDup := &models.AssetStoreRecord{ID: "rDup", AssetStoreID: storeRed.ID, Quantity: 0.99, RecordDate: "2026-10-01", CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateAssetStoreRecord(ctx, rDup); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("expected conflict on duplicate record date, got: %v", err)
	}

	// 4. Check store detail (latest quantity 0.5, latest value 60000.0 => total 30000.0)
	detail, err := repo.GetAssetStoreByID(ctx, acc.ID, storeRed.ID)
	if err != nil {
		t.Fatalf("failed to get store detail: %v", err)
	}
	if detail.LatestQuantity != 0.5 || detail.LatestValue != 60000.0 || detail.TotalValue != 30000.0 {
		t.Errorf("unexpected detail: %+v", detail)
	}

	// 5. Store trend
	trend, err := repo.GetAssetStoreTrend(ctx, storeRed.ID)
	if err != nil || len(trend) == 0 {
		t.Fatalf("expected trend points, got %d (%v)", len(trend), err)
	}

	// 6. Delete store (cascades records)
	if err := repo.DeleteAssetStore(ctx, acc.ID, storeRed.ID); err != nil {
		t.Fatalf("failed to delete store: %v", err)
	}
	recs, _ := repo.ListAssetStoreRecords(ctx, storeRed.ID)
	if len(recs) != 0 {
		t.Errorf("expected 0 records after cascade delete, got %d", len(recs))
	}
}

func TestMemoryRepo_SubscriptionsCRUD(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx := context.Background()
	now := time.Now().UTC()
	user := &models.User{ID: "sub-user-1", Username: "subuser", Email: "sub@example.com"}
	_ = repo.CreateUser(ctx, user)

	sub := &models.Subscription{
		ID: "sub-1", UserID: user.ID, PaidDate: "2026-11-01",
		SubscriberEnds: "2026-12-01", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateSubscription(ctx, sub); err != nil {
		t.Fatalf("CreateSubscription failed: %v", err)
	}
	// duplicate fails
	if err := repo.CreateSubscription(ctx, sub); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("expected ErrConflict, got: %v", err)
	}
	// nonexistent user fails
	orphan := &models.Subscription{ID: "sub-orphan", UserID: "nonexistent"}
	if err := repo.CreateSubscription(ctx, orphan); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing user, got: %v", err)
	}

	got, err := repo.GetSubscriptionByID(ctx, "sub-1")
	if err != nil || got.ID != "sub-1" {
		t.Fatalf("GetSubscriptionByID failed: %v", err)
	}
	list, err := repo.ListSubscriptionsByUserID(ctx, user.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListSubscriptionsByUserID failed: %v", err)
	}
	if err := repo.DeleteSubscription(ctx, "sub-1"); err != nil {
		t.Fatalf("DeleteSubscription failed: %v", err)
	}
}

func TestMemoryRepo_IsUserSubscriber(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx := context.Background()
	user := &models.User{ID: "sub-user-2", Username: "subuser2", Email: "sub2@example.com"}
	_ = repo.CreateUser(ctx, user)

	isSub, _ := repo.IsUserSubscriber(ctx, user.ID)
	if isSub {
		t.Fatal("expected false for user without subscription")
	}

	// Expired subscription
	expired := &models.Subscription{
		ID: "sub-exp", UserID: user.ID, PaidDate: "2020-01-01", SubscriberEnds: "2020-01-31",
	}
	_ = repo.CreateSubscription(ctx, expired)
	isSub, _ = repo.IsUserSubscriber(ctx, user.ID)
	if isSub {
		t.Fatal("expected false for expired subscription")
	}

	// Active subscription (future end date)
	active := &models.Subscription{
		ID: "sub-act", UserID: user.ID, PaidDate: "2026-01-01", SubscriberEnds: "2099-12-31",
	}
	_ = repo.CreateSubscription(ctx, active)
	isSub, _ = repo.IsUserSubscriber(ctx, user.ID)
	if !isSub {
		t.Fatal("expected true for active subscription")
	}
}

func TestMemoryRepo_PerspectiveCRUD(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx := context.Background()
	now := time.Now().UTC()

	p := &models.Perspective{
		ID: "p-1", UserID: "u-1", Name: "Retirement", Description: "All retirement accounts",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreatePerspective(ctx, p); err != nil {
		t.Fatalf("failed to create perspective: %v", err)
	}
	got, err := repo.GetPerspectiveByID(ctx, "p-1")
	if err != nil || got.Name != "Retirement" {
		t.Fatalf("unexpected get result: %v, %v", got, err)
	}

	p.Name = "Updated Retirement"
	p.UpdatedAt = now.Add(time.Hour)
	if err := repo.UpdatePerspective(ctx, p); err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	// Settings
	settings := &models.PerspectiveSettings{
		ID: "ps-1", PerspectiveID: "p-1", ShowFinancialTrend: true,
		CalculateAverageDailySpend: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.UpsertPerspectiveSettings(ctx, settings); err != nil {
		t.Fatalf("failed to upsert settings: %v", err)
	}
	gotSettings, err := repo.GetPerspectiveSettings(ctx, "p-1")
	if err != nil || !gotSettings.CalculateAverageDailySpend {
		t.Fatalf("unexpected settings: %v, %v", gotSettings, err)
	}

	if err := repo.DeletePerspective(ctx, "p-1"); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}
	if _, err := repo.GetPerspectiveByID(ctx, "p-1"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestMemoryRepo_PerspectiveSharingAndAccounts(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx := context.Background()
	now := time.Now().UTC()

	p := &models.Perspective{ID: "p-2", UserID: "u-1", Name: "General Wealth", CreatedAt: now, UpdatedAt: now}
	_ = repo.CreatePerspective(ctx, p)

	pa := &models.PerspectiveAccount{ID: "pa-1", PerspectiveID: "p-2", AccountID: "acc-1", CreatedAt: now}
	if err := repo.AddPerspectiveAccount(ctx, pa); err != nil {
		t.Fatalf("failed to add account link: %v", err)
	}
	if err := repo.AddPerspectiveAccount(ctx, pa); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("expected conflict on duplicate account link: %v", err)
	}

	accIDs, err := repo.ListPerspectiveAccountIDs(ctx, "p-2")
	if err != nil || len(accIDs) != 1 || accIDs[0] != "acc-1" {
		t.Fatalf("unexpected account IDs: %v, %v", accIDs, err)
	}

	// Access
	access := &models.PerspectiveAccess{
		ID: "pacc-1", PerspectiveID: "p-2", UserID: "u-2", AccessLevel: models.AccessLevelRead,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.GrantPerspectiveAccess(ctx, access); err != nil {
		t.Fatalf("failed to grant access: %v", err)
	}

	u2Perspectives, _ := repo.ListPerspectivesByUserID(ctx, "u-2")
	if len(u2Perspectives) != 1 || u2Perspectives[0].ID != "p-2" {
		t.Fatalf("expected u-2 to see shared perspective, got %v", u2Perspectives)
	}

	_ = repo.RemovePerspectiveAccount(ctx, "p-2", "acc-1")
	accIDsAfter, _ := repo.ListPerspectiveAccountIDs(ctx, "p-2")
	if len(accIDsAfter) != 0 {
		t.Fatalf("expected 0 accounts after removal, got %v", accIDsAfter)
	}
}

func TestMemoryRepo_DeleteAccountCascade(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx := context.Background()

	acc := &models.Account{ID: "acc-del", AdminID: "u-1", Name: "Everyday Checking", Type: "checking"}
	_ = repo.CreateAccount(ctx, acc)
	_ = repo.CreateCheckingRecord(ctx, &models.CheckingRecord{ID: "cr-1", AccountID: acc.ID, Value: 100})
	_ = repo.CreateAccountSnapshot(ctx, &models.AccountSnapshot{ID: "snap-1", AccountID: acc.ID, Value: 200, SnapshotDate: "2026-01-01"})

	// Defensive check: non-matching adminID must return ErrNotFound
	if err := repo.DeleteAccount(ctx, acc.ID, "wrong-admin"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for non-creator adminID, got %v", err)
	}

	// Matching adminID succeeds
	if err := repo.DeleteAccount(ctx, acc.ID, acc.AdminID); err != nil {
		t.Fatalf("expected successful delete: %v", err)
	}

	if _, err := repo.GetAccountByID(ctx, acc.ID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for deleted account, got %v", err)
	}
	if err := repo.DeleteAccount(ctx, acc.ID, acc.AdminID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected ErrNotFound on second delete, got %v", err)
	}
}

func TestMemoryRepo_CategoryDefaultsAndUniqueness(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx := context.Background()
	now := time.Now().UTC()
	u1, u2 := "user-1", "user-2"

	cats, err := repo.ListCategoriesForUser(ctx, u1)
	if err != nil {
		t.Fatalf("ListCategoriesForUser failed: %v", err)
	}
	if len(cats) != 7 {
		t.Fatalf("expected 7 seeded default categories, got %d", len(cats))
	}

	c1 := &models.Category{ID: "cat-1", UserID: &u1, Name: "Dining", CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateCategory(ctx, c1); err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	c1Dup := &models.Category{ID: "cat-1-dup", UserID: &u1, Name: "dining", CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateCategory(ctx, c1Dup); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("expected ErrConflict on case-insensitive duplicate, got %v", err)
	}
	c2 := &models.Category{ID: "cat-2", UserID: &u2, Name: "Dining", CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateCategory(ctx, c2); err != nil {
		t.Fatalf("expected same name for different user to succeed, got %v", err)
	}
	if _, err := repo.GetCategoryByID(ctx, "missing-id"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing category, got %v", err)
	}
}

func TestMemoryRepo_DeleteCategoryNullifiesRecordsAndSharedVisibility(t *testing.T) {
	repo := repository.NewMemoryRepository()
	ctx := context.Background()
	now := time.Now().UTC()
	owner, shared := "owner-1", "shared-1"

	cat := &models.Category{ID: "cat-shared", UserID: &owner, Name: "Pet Care", CreatedAt: now, UpdatedAt: now}
	_ = repo.CreateCategory(ctx, cat)
	acc := &models.Account{ID: "acc-1", AdminID: owner, Name: "Shared Checking", Type: "checking", CreatedAt: now, UpdatedAt: now}
	_ = repo.CreateAccount(ctx, acc)
	_ = repo.GrantAccountAccess(ctx, &models.AccountAccess{
		ID: "aa-1", AccountID: acc.ID, UserID: shared, AccessLevel: models.AccessLevelRead, CreatedAt: now, UpdatedAt: now,
	})

	catID := cat.ID
	rec := &models.CheckingRecord{
		ID: "rec-1", AccountID: acc.ID, CategoryID: &catID, Value: -55.0, DateSent: "2026-11-10", Entity: "Pet Clinic", CreatedAt: now, UpdatedAt: now,
	}
	_ = repo.CreateCheckingRecord(ctx, rec)

	sharedCats, err := repo.ListCategoriesForUser(ctx, shared)
	if err != nil {
		t.Fatalf("ListCategoriesForUser failed: %v", err)
	}
	if len(sharedCats) != 8 {
		t.Fatalf("expected shared user to see 8 categories, got %d", len(sharedCats))
	}

	if err := repo.DeleteCategory(ctx, cat.ID); err != nil {
		t.Fatalf("DeleteCategory failed: %v", err)
	}
	if err := repo.DeleteCategory(ctx, cat.ID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected ErrNotFound on second delete, got %v", err)
	}
	updated, err := repo.GetCheckingRecordByID(ctx, acc.ID, rec.ID)
	if err != nil || updated.CategoryID != nil {
		t.Fatalf("expected record CategoryID to be nil after category deletion, got %+v err=%v", updated, err)
	}
}


