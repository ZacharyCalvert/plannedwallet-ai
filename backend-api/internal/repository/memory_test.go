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
