package controller

import (
	"time"

	"github.com/plannedwallet/internal/models"
)

func dateOnlyUTC(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func addMonthsClamped(d time.Time, months int) time.Time {
	totalMonths := d.Year()*12 + int(d.Month()) - 1 + months
	targetYear := totalMonths / 12
	targetMonth := time.Month(totalMonths%12 + 1)
	day := d.Day()
	if maxD := daysInMonth(targetYear, targetMonth); day > maxD {
		day = maxD
	}
	return time.Date(targetYear, targetMonth, day, 0, 0, 0, 0, time.UTC)
}

func addYearsClamped(d time.Time, years int) time.Time {
	targetYear := d.Year() + years
	day := d.Day()
	if maxD := daysInMonth(targetYear, d.Month()); day > maxD {
		day = maxD
	}
	return time.Date(targetYear, d.Month(), day, 0, 0, 0, 0, time.UTC)
}

func nthWeekdayInMonth(year int, month time.Month, nth int, wd time.Weekday) (time.Time, bool) {
	if nth == -1 {
		last := time.Date(year, month, daysInMonth(year, month), 0, 0, 0, 0, time.UTC)
		offset := (int(last.Weekday()) - int(wd) + 7) % 7
		return last.AddDate(0, 0, -offset), true
	}
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	offset := (int(wd) - int(first.Weekday()) + 7) % 7
	day := 1 + offset + 7*(nth-1)
	if day > daysInMonth(year, month) {
		return time.Time{}, false
	}
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), true
}

func nthWeekdayInYear(year int, nth int, wd time.Weekday) (time.Time, bool) {
	if nth == -1 {
		dec31 := time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC)
		offset := (int(dec31.Weekday()) - int(wd) + 7) % 7
		return dec31.AddDate(0, 0, -offset), true
	}
	jan1 := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	offset := (int(wd) - int(jan1.Weekday()) + 7) % 7
	occ := jan1.AddDate(0, 0, offset+7*(nth-1))
	if occ.Year() != year {
		return time.Time{}, false
	}
	return occ, true
}

func normalizedInterval(interval int) int {
	if interval < 1 {
		return 1
	}
	return interval
}

func nextDueWithoutSchedule(freq string, interval int, lastApplied time.Time) time.Time {
	step := normalizedInterval(interval)
	if freq == models.RecurringFrequencyWeekly {
		return lastApplied.AddDate(0, 0, 7*step)
	}
	if freq == models.RecurringFrequencyMonthly {
		return addMonthsClamped(lastApplied, step)
	}
	return addYearsClamped(lastApplied, step)
}

func nextDueWeekly(s *models.RecurringSchedule, minDate time.Time, interval int, hasLastApplied bool) time.Time {
	wd := recurringWeekdays[s.Weekday]
	offset := (int(wd) - int(minDate.Weekday()) + 7) % 7
	first := minDate.AddDate(0, 0, offset)
	if hasLastApplied && interval > 1 {
		return first.AddDate(0, 0, 7*(interval-1))
	}
	return first
}

func monthlyOccurrence(year int, month time.Month, s *models.RecurringSchedule) (time.Time, bool) {
	if s.Type == models.RecurringScheduleTypeDayOfPeriod {
		day := *s.Day
		if maxD := daysInMonth(year, month); day > maxD {
			day = maxD
		}
		return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), true
	}
	return nthWeekdayInMonth(year, month, *s.Nth, recurringWeekdays[s.Weekday])
}

func findMonthlyFromCursor(s *models.RecurringSchedule, cursor time.Time, minDate time.Time) time.Time {
	for i := 0; i < 36; i++ {
		if occ, ok := monthlyOccurrence(cursor.Year(), cursor.Month(), s); ok && !occ.Before(minDate) {
			return occ
		}
		cursor = cursor.AddDate(0, 1, 0)
	}
	return minDate
}

func nextDueMonthly(s *models.RecurringSchedule, minDate time.Time, interval int, hasLastApplied bool) time.Time {
	start := time.Date(minDate.Year(), minDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	first := findMonthlyFromCursor(s, start, minDate)
	if !hasLastApplied || interval <= 1 {
		return first
	}
	target := time.Date(first.Year(), first.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, interval-1, 0)
	return findMonthlyFromCursor(s, target, target)
}

func yearlyOccurrence(year int, s *models.RecurringSchedule) (time.Time, bool) {
	if s.Type == models.RecurringScheduleTypeDayOfPeriod {
		jan1 := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
		occ := jan1.AddDate(0, 0, *s.Day-1)
		if occ.Year() != year {
			return time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC), true
		}
		return occ, true
	}
	if s.Type == models.RecurringScheduleTypeMonthDay {
		m := time.Month(*s.Month)
		day := *s.Day
		if maxD := daysInMonth(year, m); day > maxD {
			day = maxD
		}
		return time.Date(year, m, day, 0, 0, 0, 0, time.UTC), true
	}
	if s.Month != nil {
		return nthWeekdayInMonth(year, time.Month(*s.Month), *s.Nth, recurringWeekdays[s.Weekday])
	}
	return nthWeekdayInYear(year, *s.Nth, recurringWeekdays[s.Weekday])
}

func findYearlyFromYear(s *models.RecurringSchedule, startYear int, minDate time.Time) time.Time {
	for year := startYear; year <= startYear+10; year++ {
		if occ, ok := yearlyOccurrence(year, s); ok && !occ.Before(minDate) {
			return occ
		}
	}
	return minDate
}

func nextDueYearly(s *models.RecurringSchedule, minDate time.Time, interval int, hasLastApplied bool) time.Time {
	first := findYearlyFromYear(s, minDate.Year(), minDate)
	if !hasLastApplied || interval <= 1 {
		return first
	}
	targetYear := first.Year() + interval - 1
	targetStart := time.Date(targetYear, time.January, 1, 0, 0, 0, 0, time.UTC)
	return findYearlyFromYear(s, targetYear, targetStart)
}

func parseLastDateApplied(raw *string) (time.Time, bool) {
	if raw == nil || *raw == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse("2006-01-02", *raw)
	if err != nil {
		return time.Time{}, false
	}
	return dateOnlyUTC(parsed), true
}

func computeNextDueDate(rt *models.RecurringTransaction, today time.Time) *time.Time {
	step := normalizedInterval(rt.Interval)
	lastApplied, hasLastApplied := parseLastDateApplied(rt.LastDateApplied)
	if rt.Schedule == nil {
		if !hasLastApplied {
			return nil
		}
		d := nextDueWithoutSchedule(rt.Frequency, step, lastApplied)
		return &d
	}

	minDate := dateOnlyUTC(today)
	if hasLastApplied {
		minDate = lastApplied.AddDate(0, 0, 1)
	}

	var d time.Time
	switch rt.Frequency {
	case models.RecurringFrequencyWeekly:
		d = nextDueWeekly(rt.Schedule, minDate, step, hasLastApplied)
	case models.RecurringFrequencyMonthly:
		d = nextDueMonthly(rt.Schedule, minDate, step, hasLastApplied)
	default:
		d = nextDueYearly(rt.Schedule, minDate, step, hasLastApplied)
	}
	return &d
}

// ComputeRecurringDueInfo calculates next_due_date, due_status, and days_until_due relative to refToday.
func ComputeRecurringDueInfo(rt *models.RecurringTransaction, refToday time.Time) (*string, string, *int) {
	today := dateOnlyUTC(refToday)
	nextDue := computeNextDueDate(rt, today)
	if nextDue == nil {
		return nil, models.RecurringStatusUnscheduled, nil
	}
	iso := nextDue.Format("2006-01-02")
	days := int(nextDue.Sub(today).Hours() / 24)
	status := models.RecurringStatusScheduled
	if days < 0 {
		status = models.RecurringStatusOverdue
	} else if days <= 7 {
		status = models.RecurringStatusUpcoming
	}
	return &iso, status, &days
}
