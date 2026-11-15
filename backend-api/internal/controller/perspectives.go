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

var (
	ErrPerspectiveNotFound  = NewAppError(http.StatusNotFound, "PERSPECTIVE_NOT_FOUND", "Perspective not found.")
	ErrAccountAlreadyLinked = NewAppError(http.StatusConflict, "ACCOUNT_ALREADY_LINKED", "Account is already linked to perspective.")
	ErrAccountNotLinked     = NewAppError(http.StatusNotFound, "ACCOUNT_NOT_LINKED", "Account is not linked to perspective.")
)

func (c *Controller) verifyAccountAccess(ctx context.Context, userID, accountID string, minLevel models.AccessLevel) (*models.Account, error) {
	acc, err := c.repo.GetAccountByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve account.")
	}
	if acc.AdminID == userID {
		return acc, nil
	}
	access, err := c.repo.GetAccountAccess(ctx, accountID, userID)
	if err != nil || access == nil {
		return nil, ErrAccountNotFound
	}
	if !checkAccessLevel(access.AccessLevel, minLevel) {
		return nil, NewAppError(http.StatusForbidden, "FORBIDDEN", "Insufficient permissions for account.")
	}
	return acc, nil
}

func (c *Controller) verifyPerspectiveAccess(ctx context.Context, userID, perspectiveID string, minLevel models.AccessLevel) (*models.Perspective, models.AccessLevel, error) {
	p, err := c.repo.GetPerspectiveByID(ctx, perspectiveID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, "", ErrPerspectiveNotFound
		}
		return nil, "", NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get perspective.")
	}
	if p.UserID == userID {
		return p, models.AccessLevelAdmin, nil
	}
	access, err := c.repo.GetPerspectiveAccess(ctx, perspectiveID, userID)
	if err != nil || access == nil {
		return nil, "", ErrPerspectiveNotFound
	}
	if !checkAccessLevel(access.AccessLevel, minLevel) {
		return nil, "", NewAppError(http.StatusForbidden, "FORBIDDEN", "Insufficient permissions for perspective.")
	}
	return p, access.AccessLevel, nil
}

func defaultPerspectiveSettings(perspectiveID string, now time.Time, in *models.PerspectiveSettingsInput) *models.PerspectiveSettings {
	s := &models.PerspectiveSettings{
		ID:                         uuid.New().String(),
		PerspectiveID:              perspectiveID,
		ShowFinancialTrend:         true,
		CalculateAverageDailySpend: false,
		CalculateAverageReturnYoY:  false,
		TrackRuinedDate:            false,
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}
	if in == nil {
		return s
	}
	if in.ShowFinancialTrend != nil {
		s.ShowFinancialTrend = *in.ShowFinancialTrend
	}
	if in.CalculateAverageDailySpend != nil {
		s.CalculateAverageDailySpend = *in.CalculateAverageDailySpend
	}
	if in.CalculateAverageReturnYoY != nil {
		s.CalculateAverageReturnYoY = *in.CalculateAverageReturnYoY
	}
	if in.TrackRuinedDate != nil {
		s.TrackRuinedDate = *in.TrackRuinedDate
	}
	s.CustomDailySpend = in.CustomDailySpend
	return s
}

func (c *Controller) linkInitialAccounts(ctx context.Context, userID, pID string, accIDs []string, now time.Time) {
	for _, accID := range accIDs {
		accID = strings.TrimSpace(accID)
		if accID == "" {
			continue
		}
		if _, err := c.verifyAccountAccess(ctx, userID, accID, models.AccessLevelRead); err != nil {
			continue
		}
		pa := &models.PerspectiveAccount{
			ID:            uuid.New().String(),
			PerspectiveID: pID,
			AccountID:     accID,
			CreatedAt:     now,
		}
		_ = c.repo.AddPerspectiveAccount(ctx, pa)
	}
}

// CreatePerspective creates a new perspective, default settings, and initial linked accounts.
func (c *Controller) CreatePerspective(ctx context.Context, userID string, req models.PerspectiveCreateRequest) (*models.PerspectiveResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, NewAppError(http.StatusBadRequest, "MISSING_NAME", "Perspective name is required.")
	}
	now := time.Now().UTC()
	p := &models.Perspective{
		ID:          uuid.New().String(),
		UserID:      userID,
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := c.repo.CreatePerspective(ctx, p); err != nil {
		trace.LogError(ctx, err, "Failed to create perspective")
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create perspective.")
	}
	settings := defaultPerspectiveSettings(p.ID, now, req.Settings)
	_ = c.repo.UpsertPerspectiveSettings(ctx, settings)
	c.linkInitialAccounts(ctx, userID, p.ID, req.AccountIDs, now)

	linked, _ := c.repo.ListPerspectiveAccountIDs(ctx, p.ID)
	return &models.PerspectiveResponse{
		ID:          p.ID,
		UserID:      p.UserID,
		Name:        p.Name,
		Description: p.Description,
		AccessLevel: models.AccessLevelAdmin,
		Settings:    settings,
		AccountIDs:  linked,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}, nil
}

// ListPerspectives returns all perspectives accessible by the user.
func (c *Controller) ListPerspectives(ctx context.Context, userID string) ([]*models.PerspectiveResponse, error) {
	list, err := c.repo.ListPerspectivesByUserID(ctx, userID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list perspectives.")
	}
	var res []*models.PerspectiveResponse
	for _, p := range list {
		lvl := models.AccessLevelAdmin
		if p.UserID != userID {
			acc, _ := c.repo.GetPerspectiveAccess(ctx, p.ID, userID)
			if acc != nil {
				lvl = acc.AccessLevel
			}
		}
		settings, _ := c.repo.GetPerspectiveSettings(ctx, p.ID)
		accIDs, _ := c.repo.ListPerspectiveAccountIDs(ctx, p.ID)
		res = append(res, &models.PerspectiveResponse{
			ID:          p.ID,
			UserID:      p.UserID,
			Name:        p.Name,
			Description: p.Description,
			AccessLevel: lvl,
			Settings:    settings,
			AccountIDs:  accIDs,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		})
	}
	return res, nil
}

// GetPerspective retrieves a single perspective if user has access.
func (c *Controller) GetPerspective(ctx context.Context, userID, perspectiveID string) (*models.PerspectiveResponse, error) {
	p, lvl, err := c.verifyPerspectiveAccess(ctx, userID, perspectiveID, models.AccessLevelRead)
	if err != nil {
		return nil, err
	}
	settings, _ := c.repo.GetPerspectiveSettings(ctx, p.ID)
	accIDs, _ := c.repo.ListPerspectiveAccountIDs(ctx, p.ID)
	return &models.PerspectiveResponse{
		ID:          p.ID,
		UserID:      p.UserID,
		Name:        p.Name,
		Description: p.Description,
		AccessLevel: lvl,
		Settings:    settings,
		AccountIDs:  accIDs,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}, nil
}

// UpdatePerspective updates a perspective name and description.
func (c *Controller) UpdatePerspective(ctx context.Context, userID, perspectiveID string, req models.PerspectiveUpdateRequest) (*models.PerspectiveResponse, error) {
	p, lvl, err := c.verifyPerspectiveAccess(ctx, userID, perspectiveID, models.AccessLevelEdit)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, NewAppError(http.StatusBadRequest, "MISSING_NAME", "Perspective name cannot be empty.")
	}
	p.Name = name
	p.Description = strings.TrimSpace(req.Description)
	p.UpdatedAt = time.Now().UTC()
	if err := c.repo.UpdatePerspective(ctx, p); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update perspective.")
	}
	settings, _ := c.repo.GetPerspectiveSettings(ctx, p.ID)
	accIDs, _ := c.repo.ListPerspectiveAccountIDs(ctx, p.ID)
	return &models.PerspectiveResponse{
		ID:          p.ID,
		UserID:      p.UserID,
		Name:        p.Name,
		Description: p.Description,
		AccessLevel: lvl,
		Settings:    settings,
		AccountIDs:  accIDs,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}, nil
}

// DeletePerspective deletes a perspective. Requires admin access.
func (c *Controller) DeletePerspective(ctx context.Context, userID, perspectiveID string) error {
	p, lvl, err := c.verifyPerspectiveAccess(ctx, userID, perspectiveID, models.AccessLevelAdmin)
	if err != nil {
		return err
	}
	if p.UserID != userID && lvl != models.AccessLevelAdmin {
		return NewAppError(http.StatusForbidden, "FORBIDDEN", "Only admin can delete perspective.")
	}
	if err := c.repo.DeletePerspective(ctx, perspectiveID); err != nil {
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete perspective.")
	}
	return nil
}

// GetPerspectiveSettings retrieves settings for a perspective.
func (c *Controller) GetPerspectiveSettings(ctx context.Context, userID, perspectiveID string) (*models.PerspectiveSettings, error) {
	if _, _, err := c.verifyPerspectiveAccess(ctx, userID, perspectiveID, models.AccessLevelRead); err != nil {
		return nil, err
	}
	s, err := c.repo.GetPerspectiveSettings(ctx, perspectiveID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			now := time.Now().UTC()
			s = defaultPerspectiveSettings(perspectiveID, now, nil)
			_ = c.repo.UpsertPerspectiveSettings(ctx, s)
			return s, nil
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get settings.")
	}
	return s, nil
}

func applySettingsInput(s *models.PerspectiveSettings, in models.PerspectiveSettingsInput, now time.Time) {
	if in.ShowFinancialTrend != nil {
		s.ShowFinancialTrend = *in.ShowFinancialTrend
	}
	if in.CalculateAverageDailySpend != nil {
		s.CalculateAverageDailySpend = *in.CalculateAverageDailySpend
	}
	if in.CalculateAverageReturnYoY != nil {
		s.CalculateAverageReturnYoY = *in.CalculateAverageReturnYoY
	}
	if in.TrackRuinedDate != nil {
		s.TrackRuinedDate = *in.TrackRuinedDate
	}
	if in.CustomDailySpend != nil {
		s.CustomDailySpend = in.CustomDailySpend
	}
	s.UpdatedAt = now
}

// UpdatePerspectiveSettings updates settings for a perspective. Requires edit access.
func (c *Controller) UpdatePerspectiveSettings(ctx context.Context, userID, perspectiveID string, in models.PerspectiveSettingsInput) (*models.PerspectiveSettings, error) {
	if _, _, err := c.verifyPerspectiveAccess(ctx, userID, perspectiveID, models.AccessLevelEdit); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	s, err := c.repo.GetPerspectiveSettings(ctx, perspectiveID)
	if err != nil {
		s = defaultPerspectiveSettings(perspectiveID, now, &in)
	} else {
		applySettingsInput(s, in, now)
	}
	if err := c.repo.UpsertPerspectiveSettings(ctx, s); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update settings.")
	}
	return s, nil
}

// ListPerspectiveAccounts returns the full account details linked to a perspective.
func (c *Controller) ListPerspectiveAccounts(ctx context.Context, userID, perspectiveID string) ([]*models.AccountResponse, error) {
	if _, _, err := c.verifyPerspectiveAccess(ctx, userID, perspectiveID, models.AccessLevelRead); err != nil {
		return nil, err
	}
	accIDs, err := c.repo.ListPerspectiveAccountIDs(ctx, perspectiveID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list accounts.")
	}
	var res []*models.AccountResponse
	for _, accID := range accIDs {
		acc, err := c.repo.GetAccountByID(ctx, accID)
		if err == nil {
			res = append(res, &models.AccountResponse{
				ID:          acc.ID,
				AdminID:     acc.AdminID,
				Name:        acc.Name,
				Type:        acc.Type,
				Description: acc.Description,
				CreatedAt:   acc.CreatedAt,
				UpdatedAt:   acc.UpdatedAt,
			})
		}
	}
	return res, nil
}

// LinkPerspectiveAccount links an account to a perspective. Requires edit access on perspective and read on account.
func (c *Controller) LinkPerspectiveAccount(ctx context.Context, userID, perspectiveID, accountID string) error {
	if _, _, err := c.verifyPerspectiveAccess(ctx, userID, perspectiveID, models.AccessLevelEdit); err != nil {
		return err
	}
	if _, err := c.verifyAccountAccess(ctx, userID, accountID, models.AccessLevelRead); err != nil {
		return err
	}
	pa := &models.PerspectiveAccount{
		ID:            uuid.New().String(),
		PerspectiveID: perspectiveID,
		AccountID:     accountID,
		CreatedAt:     time.Now().UTC(),
	}
	if err := c.repo.AddPerspectiveAccount(ctx, pa); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return ErrAccountAlreadyLinked
		}
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to link account.")
	}
	return nil
}

// UnlinkPerspectiveAccount removes an account from a perspective. Requires edit access.
func (c *Controller) UnlinkPerspectiveAccount(ctx context.Context, userID, perspectiveID, accountID string) error {
	if _, _, err := c.verifyPerspectiveAccess(ctx, userID, perspectiveID, models.AccessLevelEdit); err != nil {
		return err
	}
	if err := c.repo.RemovePerspectiveAccount(ctx, perspectiveID, accountID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrAccountNotLinked
		}
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to unlink account.")
	}
	return nil
}

// GrantPerspectiveAccess shares or updates access to a perspective. Requires admin access.
func (c *Controller) GrantPerspectiveAccess(ctx context.Context, adminUserID, perspectiveID, targetUserID string, level models.AccessLevel) (*models.PerspectiveAccessResponse, error) {
	if !models.ValidAccessLevels[level] {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_ACCESS_LEVEL", "Access level must be read, edit, or admin.")
	}
	p, lvl, err := c.verifyPerspectiveAccess(ctx, adminUserID, perspectiveID, models.AccessLevelAdmin)
	if err != nil {
		return nil, err
	}
	if p.UserID != adminUserID && lvl != models.AccessLevelAdmin {
		return nil, NewAppError(http.StatusForbidden, "FORBIDDEN", "Only admin can grant access.")
	}
	if _, err := c.repo.GetUserByID(ctx, targetUserID); err != nil {
		return nil, NewAppError(http.StatusNotFound, "USER_NOT_FOUND", "Target user not found.")
	}
	now := time.Now().UTC()
	access := &models.PerspectiveAccess{
		ID:            uuid.New().String(),
		PerspectiveID: perspectiveID,
		UserID:        targetUserID,
		AccessLevel:   level,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := c.repo.GrantPerspectiveAccess(ctx, access); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to grant access.")
	}
	return &models.PerspectiveAccessResponse{
		ID:            access.ID,
		PerspectiveID: access.PerspectiveID,
		UserID:        access.UserID,
		AccessLevel:   access.AccessLevel,
		CreatedAt:     access.CreatedAt,
		UpdatedAt:     access.UpdatedAt,
	}, nil
}

// GetPerspectiveAccess retrieves access level for a specific user on a perspective.
func (c *Controller) GetPerspectiveAccess(ctx context.Context, reqUserID, perspectiveID, targetUserID string) (*models.PerspectiveAccessResponse, error) {
	if _, _, err := c.verifyPerspectiveAccess(ctx, reqUserID, perspectiveID, models.AccessLevelRead); err != nil {
		return nil, err
	}
	access, err := c.repo.GetPerspectiveAccess(ctx, perspectiveID, targetUserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, NewAppError(http.StatusNotFound, "ACCESS_NOT_FOUND", "Access grant not found.")
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get access.")
	}
	return &models.PerspectiveAccessResponse{
		ID:            access.ID,
		PerspectiveID: access.PerspectiveID,
		UserID:        access.UserID,
		AccessLevel:   access.AccessLevel,
		CreatedAt:     access.CreatedAt,
		UpdatedAt:     access.UpdatedAt,
	}, nil
}

// ListPerspectiveAccess lists all access grants on a perspective. Requires admin access.
func (c *Controller) ListPerspectiveAccess(ctx context.Context, reqUserID, perspectiveID string) ([]*models.PerspectiveAccessResponse, error) {
	if _, _, err := c.verifyPerspectiveAccess(ctx, reqUserID, perspectiveID, models.AccessLevelAdmin); err != nil {
		return nil, err
	}
	list, err := c.repo.ListPerspectiveAccess(ctx, perspectiveID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list access.")
	}
	var res []*models.PerspectiveAccessResponse
	for _, a := range list {
		res = append(res, &models.PerspectiveAccessResponse{
			ID:            a.ID,
			PerspectiveID: a.PerspectiveID,
			UserID:        a.UserID,
			AccessLevel:   a.AccessLevel,
			CreatedAt:     a.CreatedAt,
			UpdatedAt:     a.UpdatedAt,
		})
	}
	return res, nil
}

// RevokePerspectiveAccess revokes access from a user. Requires admin access.
func (c *Controller) RevokePerspectiveAccess(ctx context.Context, adminUserID, perspectiveID, targetUserID string) error {
	p, lvl, err := c.verifyPerspectiveAccess(ctx, adminUserID, perspectiveID, models.AccessLevelAdmin)
	if err != nil {
		return err
	}
	if p.UserID != adminUserID && lvl != models.AccessLevelAdmin {
		return NewAppError(http.StatusForbidden, "FORBIDDEN", "Only admin can revoke access.")
	}
	if p.UserID == targetUserID {
		return NewAppError(http.StatusBadRequest, "CANNOT_REVOKE_OWNER", "Cannot revoke owner access.")
	}
	if err := c.repo.RevokePerspectiveAccess(ctx, perspectiveID, targetUserID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return NewAppError(http.StatusNotFound, "ACCESS_NOT_FOUND", "Access grant not found.")
		}
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to revoke access.")
	}
	return nil
}
