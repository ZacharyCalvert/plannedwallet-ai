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
		ID: "aa-1", AccountID: acc.ID, AssetID: asset.ID, Quantity: 45.209, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.AddAccountAsset(ctx, aa); err != nil {
		t.Fatalf("failed to add account asset: %v", err)
	}

	// Conflict on duplicate asset in same account
	aaDup := &models.AccountAsset{
		ID: "aa-dup", AccountID: acc.ID, AssetID: asset.ID, Quantity: 10.0, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.AddAccountAsset(ctx, aaDup); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("expected ErrConflict on duplicate asset in account, got %v", err)
	}

	details, err := repo.ListAccountAssets(ctx, acc.ID)
	if err != nil || len(details) != 1 {
		t.Fatalf("expected 1 detail, got %d (%v)", len(details), err)
	}
	if details[0].LatestValue != 32.50 || details[0].Quantity != 45.209 {
		t.Errorf("unexpected detail: %+v", details[0])
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
