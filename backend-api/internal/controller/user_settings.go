package controller

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/repository"
	"github.com/plannedwallet/internal/trace"
)

const (
	maxAccountOrderEntries     = 200
	maxDailyUpdateOrderEntries = 300
	maxPinnedBalanceEntries    = 200
)

func emptyUserSettingsPayload() models.UserSettingsPayload {
	return models.UserSettingsPayload{
		AccountOrder:     []string{},
		DailyUpdateOrder: []string{},
		PinnedBalances:   []models.PinnedBalanceRef{},
	}
}

func normalizeUserSettingsPayload(p *models.UserSettingsPayload) {
	if p.AccountOrder == nil {
		p.AccountOrder = []string{}
	}
	if p.DailyUpdateOrder == nil {
		p.DailyUpdateOrder = []string{}
	}
	if p.PinnedBalances == nil {
		p.PinnedBalances = []models.PinnedBalanceRef{}
	}
}

func validateAccountOrder(raw []string) ([]string, error) {
	if len(raw) > maxAccountOrderEntries {
		return nil, NewAppError(http.StatusBadRequest, "SETTINGS_LIMIT_EXCEEDED", "account_order cannot exceed 200 entries.")
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		parsed, err := uuid.Parse(strings.TrimSpace(item))
		if err != nil {
			return nil, NewAppError(http.StatusBadRequest, "INVALID_ACCOUNT_ORDER", "account_order entries must be valid UUIDs.")
		}
		id := parsed.String()
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

func parseDailyUpdateKey(raw string) (string, error) {
	parts := strings.SplitN(strings.TrimSpace(raw), ":", 2)
	if len(parts) != 2 {
		return "", NewAppError(http.StatusBadRequest, "INVALID_DAILY_UPDATE_ORDER", "daily_update_order entries must use 'snapshot:<uuid>' or 'asset:<uuid>'.")
	}
	kind := strings.ToLower(strings.TrimSpace(parts[0]))
	if kind != "snapshot" && kind != "asset" {
		return "", NewAppError(http.StatusBadRequest, "INVALID_DAILY_UPDATE_ORDER", "daily_update_order entries must use 'snapshot:<uuid>' or 'asset:<uuid>'.")
	}
	parsed, err := uuid.Parse(strings.TrimSpace(parts[1]))
	if err != nil {
		return "", NewAppError(http.StatusBadRequest, "INVALID_DAILY_UPDATE_ORDER", "daily_update_order entries must contain valid UUIDs.")
	}
	return kind + ":" + parsed.String(), nil
}

func validateDailyUpdateOrder(raw []string) ([]string, error) {
	if len(raw) > maxDailyUpdateOrderEntries {
		return nil, NewAppError(http.StatusBadRequest, "SETTINGS_LIMIT_EXCEEDED", "daily_update_order cannot exceed 300 entries.")
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		key, err := parseDailyUpdateKey(item)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out, nil
}

func validatePinnedBalances(raw []models.PinnedBalanceRef) ([]models.PinnedBalanceRef, error) {
	if len(raw) > maxPinnedBalanceEntries {
		return nil, NewAppError(http.StatusBadRequest, "SETTINGS_LIMIT_EXCEEDED", "pinned_balances cannot exceed 200 entries.")
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]models.PinnedBalanceRef, 0, len(raw))
	for _, ref := range raw {
		kind := strings.ToLower(strings.TrimSpace(ref.Kind))
		parsed, err := uuid.Parse(strings.TrimSpace(ref.ID))
		if (kind != "account" && kind != "perspective") || err != nil {
			return nil, NewAppError(http.StatusBadRequest, "INVALID_PINNED_BALANCE", "Pinned balance requires kind ('account' or 'perspective') and a valid UUID.")
		}
		id := parsed.String()
		key := kind + ":" + id
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, models.PinnedBalanceRef{Kind: kind, ID: id})
	}
	return out, nil
}

func (c *Controller) verifyPremiumSettingsAccess(ctx context.Context, userID, targetUsername string) (*models.User, error) {
	user, err := c.verifyAuthenticatedTargetUser(ctx, userID, targetUsername, "Users can only access their own settings.")
	if err != nil {
		return nil, err
	}
	isSub, err := c.resolveSubscriberStatus(ctx, user.ID)
	if err != nil {
		trace.LogError(ctx, err, "Failed to verify subscriber status for settings: %s", user.Username)
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Error verifying subscriber status.")
	}
	if !isSub {
		return nil, NewAppError(http.StatusForbidden, "PREMIUM_REQUIRED", "Custom viewing settings are available to premium subscribers only.")
	}
	return user, nil
}

func (c *Controller) loadOrInitUserSettings(ctx context.Context, userID string) (*models.UserSettings, error) {
	existing, err := c.repo.GetUserSettings(ctx, userID)
	if err == nil {
		normalizeUserSettingsPayload(&existing.Settings)
		return existing, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load user settings.")
	}
	now := time.Now().UTC()
	return &models.UserSettings{
		ID:        uuid.New().String(),
		UserID:    userID,
		Settings:  emptyUserSettingsPayload(),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// GetUserSettings retrieves the authenticated premium user's viewing settings.
func (c *Controller) GetUserSettings(ctx context.Context, userID, targetUsername string) (*models.UserSettings, error) {
	user, err := c.verifyPremiumSettingsAccess(ctx, userID, targetUsername)
	if err != nil {
		return nil, err
	}
	return c.loadOrInitUserSettings(ctx, user.ID)
}

// PatchUserSettings partially updates the authenticated premium user's viewing settings.
func (c *Controller) PatchUserSettings(ctx context.Context, userID, targetUsername string, req models.PatchUserSettingsRequest) (*models.UserSettings, error) {
	user, err := c.verifyPremiumSettingsAccess(ctx, userID, targetUsername)
	if err != nil {
		return nil, err
	}
	current, err := c.loadOrInitUserSettings(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if err := applyUserSettingsPatch(&current.Settings, req); err != nil {
		return nil, err
	}
	current.UpdatedAt = time.Now().UTC()
	if err := c.repo.UpsertUserSettings(ctx, current); err != nil {
		trace.LogError(ctx, err, "Failed to save user settings for %s", user.Username)
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to save user settings.")
	}
	trace.LogInfo(ctx, "Updated premium user settings for %s", user.Username)
	return current, nil
}

func applyUserSettingsPatch(dst *models.UserSettingsPayload, req models.PatchUserSettingsRequest) error {
	if req.AccountOrder != nil {
		validated, err := validateAccountOrder(*req.AccountOrder)
		if err != nil {
			return err
		}
		dst.AccountOrder = validated
	}
	if req.DailyUpdateOrder != nil {
		validated, err := validateDailyUpdateOrder(*req.DailyUpdateOrder)
		if err != nil {
			return err
		}
		dst.DailyUpdateOrder = validated
	}
	if req.PinnedBalances != nil {
		validated, err := validatePinnedBalances(*req.PinnedBalances)
		if err != nil {
			return err
		}
		dst.PinnedBalances = validated
	}
	normalizeUserSettingsPayload(dst)
	return nil
}
