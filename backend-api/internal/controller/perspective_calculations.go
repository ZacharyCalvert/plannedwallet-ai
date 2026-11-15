package controller

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/plannedwallet/internal/models"
)

func (c *Controller) calculateAccountCurrentValue(ctx context.Context, acc *models.Account) float64 {
	switch acc.Type {
	case "snapshot":
		snaps, err := c.repo.ListAccountSnapshots(ctx, acc.ID, nil, nil)
		if err == nil && len(snaps) > 0 {
			return snaps[len(snaps)-1].Value
		}
	case "checking", "savings", "credit card", "credit_card":
		recs, err := c.repo.ListCheckingRecords(ctx, acc.ID, nil, nil, nil)
		if err == nil {
			var sum float64
			for _, r := range recs {
				sum += r.Value
			}
			return sum
		}
	case "asset", "assets":
		stores, err := c.repo.ListAssetStores(ctx, acc.ID)
		if err == nil {
			var sum float64
			for _, s := range stores {
				sum += s.TotalValue
			}
			return sum
		}
	}
	return 0.0
}

func (c *Controller) collectAccountDailyValues(ctx context.Context, acc *models.Account) map[string]float64 {
	points := make(map[string]float64)
	if acc.Type == "snapshot" {
		snaps, err := c.repo.ListAccountSnapshots(ctx, acc.ID, nil, nil)
		if err == nil {
			for _, s := range snaps {
				points[s.SnapshotDate] = s.Value
			}
		}
		return points
	}
	if acc.Type == "checking" || acc.Type == "savings" || acc.Type == "credit card" {
		recs, err := c.repo.ListCheckingRecords(ctx, acc.ID, nil, nil, nil)
		if err == nil && len(recs) > 0 {
			sort.Slice(recs, func(i, j int) bool { return recs[i].DateSent < recs[j].DateSent })
			var running float64
			for _, r := range recs {
				running += r.Value
				points[r.DateSent] = running
			}
		}
	}
	return points
}

func collectDistinctDates(maps []map[string]float64) []string {
	dateSet := make(map[string]struct{})
	for _, m := range maps {
		for d := range m {
			dateSet[d] = struct{}{}
		}
	}
	var dates []string
	for d := range dateSet {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	return dates
}

func buildPerspectiveTrend(accIDs []string, accTrends []map[string]float64) []models.PerspectiveTrendPoint {
	dates := collectDistinctDates(accTrends)
	if len(dates) == 0 {
		return nil
	}
	known := make([]float64, len(accIDs))
	var trend []models.PerspectiveTrendPoint
	for _, d := range dates {
		var total float64
		accVals := make(map[string]float64)
		for i, id := range accIDs {
			if v, ok := accTrends[i][d]; ok {
				known[i] = v
			}
			accVals[id] = known[i]
			total += known[i]
		}
		trend = append(trend, models.PerspectiveTrendPoint{
			Date:          d,
			TotalValue:    total,
			AccountValues: accVals,
		})
	}
	return trend
}

func (c *Controller) calculatePerspectiveSpend(ctx context.Context, accIDs []string) float64 {
	var totalExpense float64
	var earliest, latest string
	for _, id := range accIDs {
		recs, err := c.repo.ListCheckingRecords(ctx, id, nil, nil, nil)
		if err != nil {
			continue
		}
		for _, r := range recs {
			if r.Value < 0 {
				totalExpense += math.Abs(r.Value)
				if earliest == "" || r.DateSent < earliest {
					earliest = r.DateSent
				}
				if latest == "" || r.DateSent > latest {
					latest = r.DateSent
				}
			}
		}
	}
	return computeDailySpendFromRange(totalExpense, earliest, latest)
}

func computeDailySpendFromRange(totalExpense float64, earliest, latest string) float64 {
	if totalExpense <= 0 || earliest == "" || latest == "" {
		return 0.0
	}
	t0, err0 := time.Parse("2006-01-02", earliest)
	tn, errN := time.Parse("2006-01-02", latest)
	if err0 != nil || errN != nil {
		return 0.0
	}
	days := tn.Sub(t0).Hours()/24.0 + 1.0
	if days < 1.0 {
		days = 1.0
	}
	return totalExpense / days
}

func calculatePerspectiveYoY(trend []models.PerspectiveTrendPoint) *float64 {
	if len(trend) < 2 {
		return nil
	}
	first := trend[0]
	last := trend[len(trend)-1]
	if first.TotalValue <= 0 || last.TotalValue <= 0 {
		return nil
	}
	d0, err0 := time.Parse("2006-01-02", first.Date)
	dn, errN := time.Parse("2006-01-02", last.Date)
	if err0 != nil || errN != nil {
		return nil
	}
	days := dn.Sub(d0).Hours() / 24.0
	if days < 1.0 {
		return nil
	}
	years := days / 365.25
	cagr := math.Pow(last.TotalValue/first.TotalValue, 1.0/years) - 1.0
	return &cagr
}

func calculateRuinedForecast(currentTotal, dailySpend float64) (*string, *int) {
	if dailySpend <= 0 {
		return nil, nil
	}
	if currentTotal <= 0 {
		today := time.Now().UTC().Format("2006-01-02")
		zero := 0
		return &today, &zero
	}
	days := int(currentTotal / dailySpend)
	ruined := time.Now().UTC().AddDate(0, 0, days).Format("2006-01-02")
	return &ruined, &days
}

func (c *Controller) resolvePerspectiveDailySpend(ctx context.Context, s *models.PerspectiveSettings, accIDs []string) float64 {
	if s.CustomDailySpend != nil && *s.CustomDailySpend > 0 {
		return *s.CustomDailySpend
	}
	return c.calculatePerspectiveSpend(ctx, accIDs)
}

func (c *Controller) computePerspectiveAnalytics(ctx context.Context, s *models.PerspectiveSettings, accIDs []string, currentTotal float64, trend []models.PerspectiveTrendPoint) (*float64, *float64, *string, *int) {
	var avgSpend *float64
	dailySpend := c.resolvePerspectiveDailySpend(ctx, s, accIDs)
	if s.CalculateAverageDailySpend {
		avgSpend = &dailySpend
	}
	var yoyReturn *float64
	if s.CalculateAverageReturnYoY && len(trend) >= 2 {
		yoyReturn = calculatePerspectiveYoY(trend)
	}
	var ruinedDate *string
	var daysUntilRuined *int
	if s.TrackRuinedDate {
		ruinedDate, daysUntilRuined = calculateRuinedForecast(currentTotal, dailySpend)
	}
	return avgSpend, yoyReturn, ruinedDate, daysUntilRuined
}

// GetPerspectiveView computes the aggregated financial trend and analytical metrics for a perspective.
func (c *Controller) GetPerspectiveView(ctx context.Context, userID, perspectiveID string) (*models.PerspectiveViewResponse, error) {
	p, lvl, err := c.verifyPerspectiveAccess(ctx, userID, perspectiveID, models.AccessLevelRead)
	if err != nil {
		return nil, err
	}
	settings, err := c.GetPerspectiveSettings(ctx, userID, perspectiveID)
	if err != nil {
		return nil, err
	}
	accounts, accIDs, currentTotal := c.loadPerspectiveAccountsSummary(ctx, perspectiveID)
	trend := c.computePerspectiveTrend(ctx, settings.ShowFinancialTrend, accIDs, accounts)
	avgSpend, yoyReturn, ruinedDate, daysRuined := c.computePerspectiveAnalytics(ctx, settings, accIDs, currentTotal, trend)

	return &models.PerspectiveViewResponse{
		Perspective: &models.PerspectiveResponse{
			ID:          p.ID,
			UserID:      p.UserID,
			Name:        p.Name,
			Description: p.Description,
			AccessLevel: lvl,
			Settings:    settings,
			AccountIDs:  accIDs,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		},
		Settings:          settings,
		Accounts:          accounts,
		CurrentTotalValue: currentTotal,
		Trend:             trend,
		AverageDailySpend: avgSpend,
		AverageReturnYoY:  yoyReturn,
		RuinedDate:        ruinedDate,
		DaysUntilRuined:   daysRuined,
	}, nil
}

func (c *Controller) loadPerspectiveAccountsSummary(ctx context.Context, pID string) ([]*models.AccountResponse, []string, float64) {
	accIDs, _ := c.repo.ListPerspectiveAccountIDs(ctx, pID)
	var accounts []*models.AccountResponse
	var currentTotal float64
	for _, id := range accIDs {
		acc, err := c.repo.GetAccountByID(ctx, id)
		if err == nil {
			accounts = append(accounts, &models.AccountResponse{
				ID:          acc.ID,
				AdminID:     acc.AdminID,
				Name:        acc.Name,
				Type:        acc.Type,
				Description: acc.Description,
				CreatedAt:   acc.CreatedAt,
				UpdatedAt:   acc.UpdatedAt,
			})
			currentTotal += c.calculateAccountCurrentValue(ctx, acc)
		}
	}
	return accounts, accIDs, currentTotal
}

func (c *Controller) computePerspectiveTrend(ctx context.Context, showTrend bool, accIDs []string, accounts []*models.AccountResponse) []models.PerspectiveTrendPoint {
	if !showTrend || len(accIDs) == 0 {
		return nil
	}
	var accTrends []map[string]float64
	for _, accResp := range accounts {
		acc, err := c.repo.GetAccountByID(ctx, accResp.ID)
		if err == nil {
			accTrends = append(accTrends, c.collectAccountDailyValues(ctx, acc))
		}
	}
	return buildPerspectiveTrend(accIDs, accTrends)
}
