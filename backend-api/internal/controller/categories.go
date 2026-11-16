package controller

import (
	"context"
	"errors"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/plannedwallet/internal/models"
	"github.com/plannedwallet/internal/repository"
	"github.com/plannedwallet/internal/trace"
)

var (
	ErrCategoryNotFound        = NewAppError(http.StatusNotFound, "CATEGORY_NOT_FOUND", "Category not found.")
	ErrCategoryConflict        = NewAppError(http.StatusConflict, "CATEGORY_CONFLICT", "A category with this name already exists.")
	ErrSystemCategoryImmutable = NewAppError(http.StatusForbidden, "SYSTEM_CATEGORY_IMMUTABLE", "Default categories cannot be deleted.")
	ErrCategoryForbidden       = NewAppError(http.StatusForbidden, "FORBIDDEN", "You do not have permission to delete this category.")
)

func toCategoryResponse(c *models.Category) models.CategoryResponse {
	return models.CategoryResponse{
		ID:        c.ID,
		UserID:    c.UserID,
		Name:      c.Name,
		IsDefault: c.UserID == nil,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

// CreateCategory creates a new user-scoped category.
func (c *Controller) CreateCategory(ctx context.Context, userID string, req models.CategoryCreateRequest) (*models.CategoryResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_NAME", "Category name cannot be empty.")
	}

	now := time.Now().UTC()
	cat := &models.Category{
		ID:        uuid.New().String(),
		UserID:    &userID,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := c.repo.CreateCategory(ctx, cat); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrCategoryConflict
		}
		trace.LogError(ctx, err, "Failed to create category %s", name)
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create category.")
	}

	trace.LogInfo(ctx, "User %s created category %s (%s)", userID, cat.Name, cat.ID)
	res := toCategoryResponse(cat)
	return &res, nil
}

// ListCategories returns global, user-created, and shared-account-used categories for the user.
func (c *Controller) ListCategories(ctx context.Context, userID string) ([]models.CategoryResponse, error) {
	cats, err := c.repo.ListCategoriesForUser(ctx, userID)
	if err != nil {
		trace.LogError(ctx, err, "Failed to list categories for user %s", userID)
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list categories.")
	}

	res := make([]models.CategoryResponse, 0, len(cats))
	for _, cat := range cats {
		res = append(res, toCategoryResponse(cat))
	}
	return res, nil
}

// GetCategory returns a single category if accessible to the user.
func (c *Controller) GetCategory(ctx context.Context, userID, categoryID string) (*models.CategoryResponse, error) {
	cats, err := c.repo.ListCategoriesForUser(ctx, userID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve category.")
	}
	for _, cat := range cats {
		if cat.ID == categoryID {
			res := toCategoryResponse(cat)
			return &res, nil
		}
	}
	return nil, ErrCategoryNotFound
}

// DeleteCategory deletes a user-owned category, setting matching transactions to uncategorized.
func (c *Controller) DeleteCategory(ctx context.Context, userID, categoryID string) error {
	cat, err := c.repo.GetCategoryByID(ctx, categoryID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrCategoryNotFound
		}
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve category.")
	}
	if cat.UserID == nil {
		return ErrSystemCategoryImmutable
	}
	if *cat.UserID != userID {
		return ErrCategoryForbidden
	}
	if err := c.repo.DeleteCategory(ctx, categoryID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrCategoryNotFound
		}
		trace.LogError(ctx, err, "Failed to delete category %s", categoryID)
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete category.")
	}
	trace.LogInfo(ctx, "User %s deleted category %s (%s)", userID, cat.Name, categoryID)
	return nil
}

func (c *Controller) resolveCategoryIDForUser(ctx context.Context, userID string, rawID *string) (*string, error) {
	if rawID == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*rawID)
	if trimmed == "" {
		return nil, nil
	}
	cats, err := c.repo.ListCategoriesForUser(ctx, userID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to verify category.")
	}
	for _, cat := range cats {
		if cat.ID == trimmed {
			return &trimmed, nil
		}
	}
	return nil, ErrCategoryNotFound
}

type categoryTotals struct {
	id    *string
	name  string
	total float64
	count int
}

func buildCategoryNameMap(cats []*models.Category) map[string]string {
	m := make(map[string]string, len(cats))
	for _, cat := range cats {
		m[cat.ID] = cat.Name
	}
	return m
}

func resolveRecordCategory(r *models.CheckingRecord, names map[string]string) (string, *string, string) {
	if r.CategoryID == nil || *r.CategoryID == "" {
		return "", nil, "Uncategorized"
	}
	key := *r.CategoryID
	idCopy := key
	if resolved, ok := names[key]; ok {
		return key, &idCopy, resolved
	}
	return key, &idCopy, "Uncategorized"
}

func sortExpenseSlices(slices []models.CategoryExpenseSlice) {
	sort.Slice(slices, func(i, j int) bool {
		if slices[i].TotalAmount != slices[j].TotalAmount {
			return slices[i].TotalAmount > slices[j].TotalAmount
		}
		return strings.ToLower(slices[i].CategoryName) < strings.ToLower(slices[j].CategoryName)
	})
}

func aggregateExpenseSlices(records []*models.CheckingRecord, names map[string]string) ([]models.CategoryExpenseSlice, float64) {
	groups := make(map[string]*categoryTotals)
	totalExpenses := 0.0

	for _, r := range records {
		if r.Value >= 0 {
			continue
		}
		amt := math.Abs(r.Value)
		totalExpenses += amt
		key, catID, name := resolveRecordCategory(r, names)
		g, exists := groups[key]
		if !exists {
			g = &categoryTotals{id: catID, name: name}
			groups[key] = g
		}
		g.total += amt
		g.count++
	}

	slices := make([]models.CategoryExpenseSlice, 0, len(groups))
	for _, g := range groups {
		slices = append(slices, models.CategoryExpenseSlice{
			CategoryID:   g.id,
			CategoryName: g.name,
			TotalAmount:  g.total,
			RecordCount:  g.count,
		})
	}
	sortExpenseSlices(slices)
	return slices, totalExpenses
}

// GetAccountCategorySummary returns expense totals grouped by category for pie chart display.
func (c *Controller) GetAccountCategorySummary(ctx context.Context, userID, accountID string, cleared *bool, startDate, endDate *string) (*models.AccountCategorySummaryResponse, error) {
	if _, err := c.verifyCheckingAccount(ctx, userID, accountID); err != nil {
		return nil, err
	}

	records, err := c.repo.ListCheckingRecords(ctx, accountID, cleared, startDate, endDate)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve checking records.")
	}

	cats, err := c.repo.ListCategoriesForUser(ctx, userID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve categories.")
	}

	slices, totalExpenses := aggregateExpenseSlices(records, buildCategoryNameMap(cats))
	return &models.AccountCategorySummaryResponse{
		AccountID:     accountID,
		TotalExpenses: totalExpenses,
		Categories:    slices,
	}, nil
}
