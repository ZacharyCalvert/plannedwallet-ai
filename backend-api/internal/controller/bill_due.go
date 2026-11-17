// Bill Due Calculation & Date Clamping
//
// Scheduling vs. Frequency Calculation Model:
//   - When bill.Schedule is nil (frequency-only mode), the next due date is derived
//     strictly from bill.LastDatePaid + (bill.Interval * bill.Frequency), such as
//     weekly (+7*interval days), monthly (+interval months), or yearly (+interval years).
//     For example, frequency="monthly" with interval=6 paid on 2026-11-16 yields
//     2027-05-16. Month and year increments clamp overflowing days to the target
//     period's end (e.g., May 31 + 6 months -> Nov 30; Feb 29 + 1 year -> Feb 28)
//     instead of rolling into the next month as Go's time.AddDate would.
//   - When bill.Schedule is non-nil (explicit schedule mode), frequency governs the
//     outer cadence/scope (week, month, year), interval skips (interval-1) additional
//     periods after the next scheduled occurrence when last_date_paid is present, and
//     next due dates search forward for the earliest valid calendar occurrence.
//     Calendar-day rules (day_of_period, month_day) clamp short months and non-leap
//     years so every period yields a due date, whereas nth_weekday rules advance to
//     the next period that contains that nth weekday.

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

// addMonthsClamped advances d by the given number of calendar months while
// clamping the day-of-month to the target month's length.
//
// Unlike Go's time.AddDate(0, months, 0), which normalizes overflowing dates into
// the following month (turning Jan 31 + 1 month into Mar 2 or Mar 3), clamping
// keeps billing within the intended target month (Jan 31 -> Feb 28 or Feb 29).
// Note that repeated +1 month increments from a clamped date reflect the clamped
// day stored on last_date_paid once applied unless an explicit Schedule is set.
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

// addYearsClamped advances d by the given number of calendar years, clamping
// leap day (Feb 29) down to Feb 28 when the target year is not a leap year.
func addYearsClamped(d time.Time, years int) time.Time {
	targetYear := d.Year() + years
	day := d.Day()
	if maxD := daysInMonth(targetYear, d.Month()); day > maxD {
		day = maxD
	}
	return time.Date(targetYear, d.Month(), day, 0, 0, 0, 0, time.UTC)
}

// nthWeekdayInMonth resolves the nth occurrence of weekday wd in the given month,
// or the final occurrence in that month when nth == -1. Returns ok=false without
// clamping if nth (e.g., 5th occurrence) does not exist in that month.
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

// nthWeekdayInYear resolves the nth occurrence of weekday wd in year (1..53),
// or the final occurrence in the year when nth == -1. Returns ok=false without
// clamping when a 53rd occurrence falls outside the target year.
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

// nextDueWithoutSchedule computes the next due date from lastPaid using only
// the bill frequency and interval multiplier (e.g., monthly + interval=6 adds
// 6 clamped months), clamping month/year overflows cleanly.
func nextDueWithoutSchedule(freq string, interval int, lastPaid time.Time) time.Time {
	step := normalizedInterval(interval)
	if freq == models.BillFrequencyWeekly {
		return lastPaid.AddDate(0, 0, 7*step)
	}
	if freq == models.BillFrequencyMonthly {
		return addMonthsClamped(lastPaid, step)
	}
	return addYearsClamped(lastPaid, step)
}

// nextDueWeekly finds the earliest date on or after minDate matching s.Weekday,
// advancing (interval-1) additional weeks when anchored past last_date_paid.
func nextDueWeekly(s *models.BillSchedule, minDate time.Time, interval int, hasLastPaid bool) time.Time {
	wd := billWeekdays[s.Weekday]
	offset := (int(wd) - int(minDate.Weekday()) + 7) % 7
	first := minDate.AddDate(0, 0, offset)
	if hasLastPaid && interval > 1 {
		return first.AddDate(0, 0, 7*(interval-1))
	}
	return first
}

// monthlyOccurrence computes the scheduled due date within (year, month).
// For day_of_period schedules (e.g., 31st of the month), shorter months clamp
// down to the month's final calendar day so the bill remains due every month.
// For nth_weekday schedules, months missing that nth weekday return ok=false.
func monthlyOccurrence(year int, month time.Month, s *models.BillSchedule) (time.Time, bool) {
	if s.Type == models.BillScheduleTypeDayOfPeriod {
		day := *s.Day
		if maxD := daysInMonth(year, month); day > maxD {
			day = maxD
		}
		return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), true
	}
	return nthWeekdayInMonth(year, month, *s.Nth, billWeekdays[s.Weekday])
}

func findMonthlyFromCursor(s *models.BillSchedule, cursor time.Time, minDate time.Time) time.Time {
	for i := 0; i < 36; i++ {
		if occ, ok := monthlyOccurrence(cursor.Year(), cursor.Month(), s); ok && !occ.Before(minDate) {
			return occ
		}
		cursor = cursor.AddDate(0, 1, 0)
	}
	return minDate
}

// nextDueMonthly scans forward from minDate to find the next monthly schedule
// occurrence, stepping (interval-1) extra months past the first match when
// anchored from last_date_paid (e.g., interval=6 skips to the 6th month).
func nextDueMonthly(s *models.BillSchedule, minDate time.Time, interval int, hasLastPaid bool) time.Time {
	start := time.Date(minDate.Year(), minDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	first := findMonthlyFromCursor(s, start, minDate)
	if !hasLastPaid || interval <= 1 {
		return first
	}
	target := time.Date(first.Year(), first.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, interval-1, 0)
	return findMonthlyFromCursor(s, target, target)
}

// yearlyOccurrence computes the scheduled due date within year.
// Calendar-day schedules clamp leap-dependent dates in non-leap years so a bill
// still triggers every year: day_of_period=366 clamps to Dec 31, and
// month_day=Feb 29 clamps to Feb 28.
func yearlyOccurrence(year int, s *models.BillSchedule) (time.Time, bool) {
	if s.Type == models.BillScheduleTypeDayOfPeriod {
		jan1 := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
		occ := jan1.AddDate(0, 0, *s.Day-1)
		if occ.Year() != year {
			// Clamp day 366 in a 365-day non-leap year back to Dec 31 instead of spilling into Jan 1.
			return time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC), true
		}
		return occ, true
	}
	if s.Type == models.BillScheduleTypeMonthDay {
		m := time.Month(*s.Month)
		day := *s.Day
		if maxD := daysInMonth(year, m); day > maxD {
			// Clamp Feb 29 in non-leap years to Feb 28.
			day = maxD
		}
		return time.Date(year, m, day, 0, 0, 0, 0, time.UTC), true
	}
	if s.Month != nil {
		return nthWeekdayInMonth(year, time.Month(*s.Month), *s.Nth, billWeekdays[s.Weekday])
	}
	return nthWeekdayInYear(year, *s.Nth, billWeekdays[s.Weekday])
}

func findYearlyFromYear(s *models.BillSchedule, startYear int, minDate time.Time) time.Time {
	for year := startYear; year <= startYear+10; year++ {
		if occ, ok := yearlyOccurrence(year, s); ok && !occ.Before(minDate) {
			return occ
		}
	}
	return minDate
}

// nextDueYearly scans forward year-by-year from minDate.Year(), advancing by
// (interval-1) additional years past the first match when anchored from last_date_paid.
func nextDueYearly(s *models.BillSchedule, minDate time.Time, interval int, hasLastPaid bool) time.Time {
	first := findYearlyFromYear(s, minDate.Year(), minDate)
	if !hasLastPaid || interval <= 1 {
		return first
	}
	targetYear := first.Year() + interval - 1
	targetStart := time.Date(targetYear, time.January, 1, 0, 0, 0, 0, time.UTC)
	return findYearlyFromYear(s, targetYear, targetStart)
}

func parseLastDatePaid(raw *string) (time.Time, bool) {
	if raw == nil || *raw == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse("2006-01-02", *raw)
	if err != nil {
		return time.Time{}, false
	}
	return dateOnlyUTC(parsed), true
}

// computeNextDueDate branches between frequency-only calculation (Schedule == nil,
// advancing directly from last_date_paid with interval and clamping) and explicit
// schedule calculation (Schedule != nil, searching from last_date_paid+1d or today).
func computeNextDueDate(bill *models.Bill, today time.Time) *time.Time {
	step := normalizedInterval(bill.Interval)
	lastPaid, hasLastPaid := parseLastDatePaid(bill.LastDatePaid)
	if bill.Schedule == nil {
		if !hasLastPaid {
			return nil
		}
		d := nextDueWithoutSchedule(bill.Frequency, step, lastPaid)
		return &d
	}

	minDate := dateOnlyUTC(today)
	if hasLastPaid {
		minDate = lastPaid.AddDate(0, 0, 1)
	}

	var d time.Time
	switch bill.Frequency {
	case models.BillFrequencyWeekly:
		d = nextDueWeekly(bill.Schedule, minDate, step, hasLastPaid)
	case models.BillFrequencyMonthly:
		d = nextDueMonthly(bill.Schedule, minDate, step, hasLastPaid)
	default:
		d = nextDueYearly(bill.Schedule, minDate, step, hasLastPaid)
	}
	return &d
}

// ComputeBillDueInfo calculates next_due_date, due_status, and days_until_due relative to refToday.
//
// Status mapping based on days_until_due = next_due_date - refToday:
//   - "unscheduled": Schedule is nil and LastDatePaid is absent/empty.
//   - "overdue": days_until_due < 0 (next due date has already passed).
//   - "upcoming": 0 <= days_until_due <= 7 (due today or within the next 7 days).
//   - "scheduled": days_until_due > 7.
func ComputeBillDueInfo(bill *models.Bill, refToday time.Time) (*string, string, *int) {
	today := dateOnlyUTC(refToday)
	nextDue := computeNextDueDate(bill, today)
	if nextDue == nil {
		return nil, models.BillStatusUnscheduled, nil
	}
	iso := nextDue.Format("2006-01-02")
	days := int(nextDue.Sub(today).Hours() / 24)
	status := models.BillStatusScheduled
	if days < 0 {
		status = models.BillStatusOverdue
	} else if days <= 7 {
		status = models.BillStatusUpcoming
	}
	return &iso, status, &days
}
