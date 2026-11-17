package controller

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/plannedwallet/internal/models"
)

var billWeekdays = map[string]time.Weekday{
	"sunday":    time.Sunday,
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,
}

// maxDaysPerMonth defines maximum valid calendar days per month (1-indexed),
// allowing Feb 29 (clamped to Feb 28 in non-leap years during due calculation).
var maxDaysPerMonth = [13]int{0, 31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

func invalidScheduleErr(msg string) error {
	return NewAppError(http.StatusBadRequest, "INVALID_SCHEDULE", msg)
}

func normalizeBillFrequency(raw string) (string, error) {
	freq := strings.ToLower(strings.TrimSpace(raw))
	switch freq {
	case models.BillFrequencyWeekly, models.BillFrequencyMonthly, models.BillFrequencyYearly:
		return freq, nil
	default:
		return "", NewAppError(http.StatusBadRequest, "INVALID_FREQUENCY", "frequency must be one of: weekly, monthly, yearly.")
	}
}

// normalizeBillInterval validates that an optional recurrence interval is >= 1,
// defaulting to 1 when omitted. An interval multiplies the base frequency
// (e.g., frequency="monthly", interval=6 represents every 6 months).
func normalizeBillInterval(raw *int) (int, error) {
	if raw == nil {
		return 1, nil
	}
	if *raw < 1 {
		return 0, NewAppError(http.StatusBadRequest, "INVALID_INTERVAL", "interval must be at least 1.")
	}
	return *raw, nil
}

func parseBillWeekday(raw string) (time.Weekday, string, error) {
	clean := strings.ToLower(strings.TrimSpace(raw))
	wd, ok := billWeekdays[clean]
	if !ok {
		return 0, "", invalidScheduleErr("weekday must be sunday through saturday.")
	}
	return wd, clean, nil
}

func validateWeeklySchedule(s *models.BillSchedule) error {
	if s.Scope != models.BillScheduleScopeWeek {
		return invalidScheduleErr("weekly frequency requires schedule scope 'week'.")
	}
	if s.Type != models.BillScheduleTypeWeekday {
		return invalidScheduleErr("weekly frequency only supports schedule type 'weekday'.")
	}
	_, norm, err := parseBillWeekday(s.Weekday)
	if err != nil {
		return err
	}
	s.Weekday = norm
	if s.Nth != nil || s.Day != nil || s.Month != nil {
		return invalidScheduleErr("weekly 'weekday' schedule cannot set nth, day, or month.")
	}
	return nil
}

func validateNthValue(nth *int, maxNth int) error {
	if nth == nil {
		return invalidScheduleErr("nth is required for nth_weekday schedule.")
	}
	if (*nth >= 1 && *nth <= maxNth) || *nth == -1 {
		return nil
	}
	return invalidScheduleErr(fmt.Sprintf("nth must be between 1 and %d, or -1 for last.", maxNth))
}

func validateMonthlySchedule(s *models.BillSchedule) error {
	if s.Scope != models.BillScheduleScopeMonth {
		return invalidScheduleErr("monthly frequency requires schedule scope 'month'.")
	}
	switch s.Type {
	case models.BillScheduleTypeDayOfPeriod:
		return validateMonthlyDayOfPeriod(s)
	case models.BillScheduleTypeNthWeekday:
		return validateMonthlyNthWeekday(s)
	default:
		return invalidScheduleErr("monthly frequency only supports 'day_of_period' or 'nth_weekday'.")
	}
}

// validateMonthlyDayOfPeriod accepts days 1..31; shorter months (e.g., Feb, Apr)
// clamp days 29..31 to their last calendar day during due-date evaluation.
func validateMonthlyDayOfPeriod(s *models.BillSchedule) error {
	if s.Day == nil || *s.Day < 1 || *s.Day > 31 {
		return invalidScheduleErr("monthly 'day_of_period' requires day between 1 and 31.")
	}
	if s.Nth != nil || s.Month != nil || s.Weekday != "" {
		return invalidScheduleErr("monthly 'day_of_period' cannot set nth, month, or weekday.")
	}
	return nil
}

func validateMonthlyNthWeekday(s *models.BillSchedule) error {
	if err := validateNthValue(s.Nth, 5); err != nil {
		return err
	}
	_, norm, err := parseBillWeekday(s.Weekday)
	if err != nil {
		return err
	}
	s.Weekday = norm
	if s.Day != nil || s.Month != nil {
		return invalidScheduleErr("monthly 'nth_weekday' cannot set day or month.")
	}
	return nil
}

func validateYearlySchedule(s *models.BillSchedule) error {
	if s.Scope != models.BillScheduleScopeYear {
		return invalidScheduleErr("yearly frequency requires schedule scope 'year'.")
	}
	switch s.Type {
	case models.BillScheduleTypeDayOfPeriod:
		return validateYearlyDayOfPeriod(s)
	case models.BillScheduleTypeMonthDay:
		return validateYearlyMonthDay(s)
	case models.BillScheduleTypeNthWeekday:
		return validateYearlyNthWeekday(s)
	default:
		return invalidScheduleErr("yearly frequency only supports 'day_of_period', 'month_day', or 'nth_weekday'.")
	}
}

// validateYearlyDayOfPeriod accepts days 1..366; day 366 clamps to Dec 31
// during non-leap years in yearlyOccurrence.
func validateYearlyDayOfPeriod(s *models.BillSchedule) error {
	if s.Day == nil || *s.Day < 1 || *s.Day > 366 {
		return invalidScheduleErr("yearly 'day_of_period' requires day between 1 and 366.")
	}
	if s.Nth != nil || s.Month != nil || s.Weekday != "" {
		return invalidScheduleErr("yearly 'day_of_period' cannot set nth, month, or weekday.")
	}
	return nil
}

// validateYearlyMonthDay validates a calendar month/day pair (accepting Feb 29,
// which clamps to Feb 28 in non-leap years when calculating due dates).
func validateYearlyMonthDay(s *models.BillSchedule) error {
	if s.Month == nil || *s.Month < 1 || *s.Month > 12 {
		return invalidScheduleErr("yearly 'month_day' requires month between 1 and 12.")
	}
	maxDay := maxDaysPerMonth[*s.Month]
	if s.Day == nil || *s.Day < 1 || *s.Day > maxDay {
		return invalidScheduleErr(fmt.Sprintf("day must be between 1 and %d for month %d.", maxDay, *s.Month))
	}
	if s.Nth != nil || s.Weekday != "" {
		return invalidScheduleErr("yearly 'month_day' cannot set nth or weekday.")
	}
	return nil
}

func validateYearlyNthWeekday(s *models.BillSchedule) error {
	_, norm, err := parseBillWeekday(s.Weekday)
	if err != nil {
		return err
	}
	s.Weekday = norm
	if s.Day != nil {
		return invalidScheduleErr("yearly 'nth_weekday' cannot set day.")
	}
	if s.Month != nil {
		if *s.Month < 1 || *s.Month > 12 {
			return invalidScheduleErr("month must be between 1 and 12.")
		}
		return validateNthValue(s.Nth, 5)
	}
	return validateNthValue(s.Nth, 53)
}

// normalizeAndValidateSchedule ensures that if a schedule rule is provided on a
// bill, its scope and type match the bill's outer frequency (e.g., a monthly
// scope such as "1st Sunday of the month" is invalid on a yearly frequency,
// whereas "1st Sunday of the year" with scope "year" is valid). A nil schedule
// is always valid and delegates recurrence timing to frequency + last_date_paid.
func normalizeAndValidateSchedule(frequency string, raw *models.BillSchedule) (*models.BillSchedule, error) {
	if raw == nil {
		return nil, nil
	}
	s := *raw
	s.Type = strings.ToLower(strings.TrimSpace(s.Type))
	s.Scope = strings.ToLower(strings.TrimSpace(s.Scope))
	s.Weekday = strings.ToLower(strings.TrimSpace(s.Weekday))

	validators := map[string]func(*models.BillSchedule) error{
		models.BillFrequencyWeekly:  validateWeeklySchedule,
		models.BillFrequencyMonthly: validateMonthlySchedule,
		models.BillFrequencyYearly:  validateYearlySchedule,
	}
	fn, ok := validators[frequency]
	if !ok {
		return nil, invalidScheduleErr("unsupported bill frequency.")
	}
	if err := fn(&s); err != nil {
		return nil, err
	}
	return &s, nil
}
