package models

import "time"

// Supported recurring frequencies.
const (
	RecurringFrequencyWeekly  = "weekly"
	RecurringFrequencyMonthly = "monthly"
	RecurringFrequencyYearly  = "yearly"
)

// Supported recurring schedule rule types and scopes.
const (
	RecurringScheduleTypeWeekday     = "weekday"
	RecurringScheduleTypeDayOfPeriod = "day_of_period"
	RecurringScheduleTypeNthWeekday  = "nth_weekday"
	RecurringScheduleTypeMonthDay    = "month_day"

	RecurringScheduleScopeWeek  = "week"
	RecurringScheduleScopeMonth = "month"
	RecurringScheduleScopeYear  = "year"
)

// Computed recurring transaction due statuses.
const (
	RecurringStatusOverdue     = "overdue"
	RecurringStatusUpcoming    = "upcoming"
	RecurringStatusScheduled   = "scheduled"
	RecurringStatusUnscheduled = "unscheduled"
)

// RecurringSchedule defines a structured recurring rule constrained by its owning Frequency.
// When nil on a RecurringTransaction, due dates advance purely by Frequency + Interval from
// LastDateApplied (with end-of-month and leap-year clamping).
type RecurringSchedule struct {
	Type    string `json:"type"`
	Scope   string `json:"scope"`
	Nth     *int   `json:"nth,omitempty"`
	Weekday string `json:"weekday,omitempty"`
	Day     *int   `json:"day,omitempty"`
	Month   *int   `json:"month,omitempty"`
}

// RecurringTransaction represents a scheduled inflow (positive amount = income) or
// outflow (negative amount = bill) owned by a single user.
type RecurringTransaction struct {
	ID              string             `json:"id"`
	UserID          string             `json:"user_id"`
	AccountID       *string            `json:"account_id"`
	CategoryID      *string            `json:"category_id"`
	Amount          float64            `json:"amount"`
	LastDateApplied *string            `json:"last_date_applied"`
	Frequency       string             `json:"frequency"`
	Interval        int                `json:"interval"`
	Schedule        *RecurringSchedule `json:"schedule"`
	Name            string             `json:"name"`
	Entity          string             `json:"entity"`
	Description     string             `json:"description"`
	Reminder        bool               `json:"reminder"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

// RecurringTransactionRequest represents the create/update payload for a recurring transaction.
type RecurringTransactionRequest struct {
	AccountID       *string            `json:"account_id,omitempty"`
	CategoryID      *string            `json:"category_id,omitempty"`
	Amount          *float64           `json:"amount" binding:"required"`
	LastDateApplied *string            `json:"last_date_applied,omitempty"`
	Frequency       string             `json:"frequency" binding:"required"`
	Interval        *int               `json:"interval,omitempty"`
	Schedule        *RecurringSchedule `json:"schedule,omitempty"`
	Name            string             `json:"name" binding:"max=19"`
	Entity          string             `json:"entity" binding:"required,max=255"`
	Description     string             `json:"description" binding:"max=500"`
	Reminder        *bool              `json:"reminder,omitempty"`
}

// RecurringTransactionResponse includes computed due-date telemetry.
type RecurringTransactionResponse struct {
	ID              string             `json:"id"`
	UserID          string             `json:"user_id"`
	AccountID       *string            `json:"account_id"`
	CategoryID      *string            `json:"category_id"`
	Amount          float64            `json:"amount"`
	LastDateApplied *string            `json:"last_date_applied"`
	Frequency       string             `json:"frequency"`
	Interval        int                `json:"interval"`
	Schedule        *RecurringSchedule `json:"schedule"`
	Name            string             `json:"name"`
	Entity          string             `json:"entity"`
	Description     string             `json:"description"`
	Reminder        bool               `json:"reminder"`
	NextDueDate     *string            `json:"next_due_date"`
	DueStatus       string             `json:"due_status"`
	DaysUntilDue    *int               `json:"days_until_due"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

// RecurringTransactionApplyRequest represents optional parameters when applying a recurring transaction.
type RecurringTransactionApplyRequest struct {
	AccountID   *string `json:"account_id,omitempty"`
	Date        *string `json:"date,omitempty"`
	DateCleared *string `json:"date_cleared,omitempty"`
}

// RecurringTransactionApplyResponse returns the updated recurring transaction and created checking record.
type RecurringTransactionApplyResponse struct {
	RecurringTransaction RecurringTransactionResponse `json:"recurring_transaction"`
	Record               CheckingRecordResponse       `json:"record"`
}
