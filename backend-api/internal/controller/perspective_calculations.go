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

type spendItem struct {
	Amount   float64
	DateSent string
}

type perspectiveSpendMetrics struct {
	overallSpend float64
	spend30d     float64
	spend90d     float64
	spend365d    float64
	refDate      time.Time
}

type perspectiveAnalytics struct {
	avgSpend            *float64
	yoyReturn           *float64
	ruinedDate          *string
	daysUntilRuined     *int
	dailySpend30d       *float64
	daysUntilRuined30d  *int
	ruinedDate30d       *string
	dailySpend90d       *float64
	daysUntilRuined90d  *int
	ruinedDate90d       *string
	dailySpend365d      *float64
	daysUntilRuined365d *int
	ruinedDate365d      *string
}

func collectRecordDates(sent string, cleared *string) []string {
	dates := []string{sent}
	if cleared != nil && *cleared != "" {
		dates = append(dates, *cleared)
	}
	return dates
}

func dateDiffDays(d1, d2 string) float64 {
	t1, err1 := time.Parse("2006-01-02", d1)
	t2, err2 := time.Parse("2006-01-02", d2)
	if err1 != nil || err2 != nil {
		return 999999.0
	}
	diff := t1.Sub(t2).Hours() / 24.0
	return math.Abs(diff)
}

func isWithin35Days(r1Sent string, r1Cleared *string, r2Sent string, r2Cleared *string) bool {
	dates1 := collectRecordDates(r1Sent, r1Cleared)
	dates2 := collectRecordDates(r2Sent, r2Cleared)
	for _, d1 := range dates1 {
		for _, d2 := range dates2 {
			if dateDiffDays(d1, d2) <= 35.0 {
				return true
			}
		}
	}
	return false
}

func findExcludedDebitIDs(chkDebits, ccIncomes []*models.CheckingRecord) map[string]bool {
	excluded := make(map[string]bool)
	matchedCC := make(map[string]bool)
	for _, chk := range chkDebits {
		chkAmt := math.Round(math.Abs(chk.Value)*100.0) / 100.0
		for _, cc := range ccIncomes {
			if matchedCC[cc.ID] {
				continue
			}
			ccAmt := math.Round(cc.Value*100.0) / 100.0
			if math.Abs(chkAmt-ccAmt) < 0.001 && isWithin35Days(chk.DateSent, chk.DateCleared, cc.DateSent, cc.DateCleared) {
				excluded[chk.ID] = true
				matchedCC[cc.ID] = true
				break
			}
		}
	}
	return excluded
}

func collectPerspectiveSpendItems(chkDebits, ccDebits []*models.CheckingRecord, excluded map[string]bool) []spendItem {
	var items []spendItem
	for _, chk := range chkDebits {
		if !excluded[chk.ID] {
			items = append(items, spendItem{Amount: math.Abs(chk.Value), DateSent: chk.DateSent})
		}
	}
	for _, cc := range ccDebits {
		items = append(items, spendItem{Amount: math.Abs(cc.Value), DateSent: cc.DateSent})
	}
	return items
}

func categorizeRecordsForSpend(accType string, recs []*models.CheckingRecord, chkDebits, ccDebits, ccIncomes *[]*models.CheckingRecord) {
	isCC := accType == "credit card" || accType == "credit_card"
	for _, r := range recs {
		if isCC && r.Value < 0 {
			*ccDebits = append(*ccDebits, r)
		} else if isCC && r.Value > 0 {
			*ccIncomes = append(*ccIncomes, r)
		} else if !isCC && r.Value < 0 {
			*chkDebits = append(*chkDebits, r)
		}
	}
}

func (c *Controller) loadAccountRecordsForSpend(ctx context.Context, accounts []*models.AccountResponse) ([]*models.CheckingRecord, []*models.CheckingRecord, []*models.CheckingRecord) {
	var chkDebits, ccDebits, ccIncomes []*models.CheckingRecord
	for _, acc := range accounts {
		recs, err := c.repo.ListCheckingRecords(ctx, acc.ID, nil, nil, nil)
		if err != nil {
			continue
		}
		categorizeRecordsForSpend(acc.Type, recs, &chkDebits, &ccDebits, &ccIncomes)
	}
	return chkDebits, ccDebits, ccIncomes
}

func findSpendItemDateRange(items []spendItem) (string, string) {
	var earliest, latest string
	for _, item := range items {
		if earliest == "" || item.DateSent < earliest {
			earliest = item.DateSent
		}
		if latest == "" || item.DateSent > latest {
			latest = item.DateSent
		}
	}
	return earliest, latest
}

func determineSpendRefDate(latestDate string) time.Time {
	now := time.Now().UTC()
	if latestDate == "" {
		return now
	}
	tLatest, err := time.Parse("2006-01-02", latestDate)
	if err != nil {
		return now
	}
	if tLatest.After(now) || now.Sub(tLatest) > 30*24*time.Hour {
		return tLatest
	}
	return now
}

func resolveWindowDays(windowDays float64, refDate time.Time, earliestDate string) float64 {
	days := windowDays
	if earliestDate != "" {
		if t0, err := time.Parse("2006-01-02", earliestDate); err == nil {
			available := refDate.Sub(t0).Hours()/24.0 + 1.0
			if available < days && available > 0 {
				days = available
			}
		}
	}
	if days < 1.0 {
		return 1.0
	}
	return days
}

func calculateWindowSpend(items []spendItem, refDate time.Time, windowDays int, earliestDate string) float64 {
	cutoff := refDate.AddDate(0, 0, -windowDays).Format("2006-01-02")
	refDateStr := refDate.Format("2006-01-02")
	var total float64
	for _, item := range items {
		if item.DateSent >= cutoff && item.DateSent <= refDateStr {
			total += item.Amount
		}
	}
	days := resolveWindowDays(float64(windowDays), refDate, earliestDate)
	return total / days
}

func calculateOverallSpend(items []spendItem, earliest, latest string) float64 {
	var total float64
	for _, item := range items {
		total += item.Amount
	}
	return computeDailySpendFromRange(total, earliest, latest)
}

func (c *Controller) calculatePerspectiveSpendMetrics(ctx context.Context, accounts []*models.AccountResponse) perspectiveSpendMetrics {
	chkDebits, ccDebits, ccIncomes := c.loadAccountRecordsForSpend(ctx, accounts)
	excluded := findExcludedDebitIDs(chkDebits, ccIncomes)
	items := collectPerspectiveSpendItems(chkDebits, ccDebits, excluded)
	earliest, latest := findSpendItemDateRange(items)
	refDate := determineSpendRefDate(latest)
	return perspectiveSpendMetrics{
		overallSpend: calculateOverallSpend(items, earliest, latest),
		spend30d:     calculateWindowSpend(items, refDate, 30, earliest),
		spend90d:     calculateWindowSpend(items, refDate, 90, earliest),
		spend365d:    calculateWindowSpend(items, refDate, 365, earliest),
		refDate:      refDate,
	}
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

func calculateRuinedForecast(currentTotal, dailySpend float64, refDate time.Time) (*string, *int) {
	if dailySpend <= 0 {
		return nil, nil
	}
	if currentTotal <= 0 {
		refStr := refDate.Format("2006-01-02")
		zero := 0
		return &refStr, &zero
	}
	days := int(currentTotal / dailySpend)
	ruined := refDate.AddDate(0, 0, days).Format("2006-01-02")
	return &ruined, &days
}

func (c *Controller) populateWindowAnalytics(res *perspectiveAnalytics, s *models.PerspectiveSettings, m perspectiveSpendMetrics, currentTotal float64) {
	if !s.TrackRuinedDate && !s.CalculateAverageDailySpend {
		return
	}
	s30, s90, s365 := m.spend30d, m.spend90d, m.spend365d
	res.dailySpend30d, res.dailySpend90d, res.dailySpend365d = &s30, &s90, &s365
	if s.TrackRuinedDate {
		res.ruinedDate30d, res.daysUntilRuined30d = calculateRuinedForecast(currentTotal, s30, m.refDate)
		res.ruinedDate90d, res.daysUntilRuined90d = calculateRuinedForecast(currentTotal, s90, m.refDate)
		res.ruinedDate365d, res.daysUntilRuined365d = calculateRuinedForecast(currentTotal, s365, m.refDate)
	}
}

func (c *Controller) computePerspectiveAnalytics(ctx context.Context, s *models.PerspectiveSettings, accounts []*models.AccountResponse, currentTotal float64, trend []models.PerspectiveTrendPoint) perspectiveAnalytics {
	metrics := c.calculatePerspectiveSpendMetrics(ctx, accounts)
	dailySpend := metrics.overallSpend
	if s.CustomDailySpend != nil && *s.CustomDailySpend > 0 {
		dailySpend = *s.CustomDailySpend
	}
	var res perspectiveAnalytics
	if s.CalculateAverageDailySpend {
		res.avgSpend = &dailySpend
	}
	if s.CalculateAverageReturnYoY && len(trend) >= 2 {
		res.yoyReturn = calculatePerspectiveYoY(trend)
	}
	if s.TrackRuinedDate {
		res.ruinedDate, res.daysUntilRuined = calculateRuinedForecast(currentTotal, dailySpend, metrics.refDate)
	}
	c.populateWindowAnalytics(&res, s, metrics, currentTotal)
	return res
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
	a := c.computePerspectiveAnalytics(ctx, settings, accounts, currentTotal, trend)

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
		Settings:            settings,
		Accounts:            accounts,
		CurrentTotalValue:   currentTotal,
		Trend:               trend,
		AverageDailySpend:   a.avgSpend,
		AverageReturnYoY:    a.yoyReturn,
		RuinedDate:          a.ruinedDate,
		DaysUntilRuined:     a.daysUntilRuined,
		DailySpend30d:       a.dailySpend30d,
		DaysUntilRuined30d:  a.daysUntilRuined30d,
		RuinedDate30d:       a.ruinedDate30d,
		DailySpend90d:       a.dailySpend90d,
		DaysUntilRuined90d:  a.daysUntilRuined90d,
		RuinedDate90d:       a.ruinedDate90d,
		DailySpend365d:      a.dailySpend365d,
		DaysUntilRuined365d: a.daysUntilRuined365d,
		RuinedDate365d:      a.ruinedDate365d,
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
